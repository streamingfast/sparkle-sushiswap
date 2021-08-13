package exchange

import (
	"fmt"
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
)

func (s *Subgraph) HandleFactoryPairCreatedEvent(ev *FactoryPairCreatedEvent) error {
	factory, err := s.getFactory()
	if err != nil {
		return err
	}

	if !factory.Exists() {
		bundle := NewBundle("1")
		if err := s.Save(bundle); err != nil {
			return err
		}
	}

	pair, err := s.getPair(ev.Pair, ev.Token0, ev.Token1)
	if err != nil {
		return err
	}

	err = s.Save(pair)
	if err != nil {
		return err
	}

	err = s.CreatePairTemplateWithTokens(ev.Pair, ev.Token0, ev.Token1)
	if err != nil {
		return err
	}

	factory.PairCount = entity.IntAdd(factory.PairCount, IL(1))
	err = s.Save(factory)
	if err != nil {
		return err
	}

	return nil
}

func (s *Subgraph) getPair(pairAddress, token0Address, token1Address eth.Address) (*Pair, error) {
	pair := NewPair(pairAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return nil, err
	}

	if pair.Exists() {
		return pair, nil
	}

	token0, err := s.getToken(token0Address)
	if err != nil {
		return nil, err
	}

	token1, err := s.getToken(token1Address)
	if err != nil {
		return nil, err
	}

	pair.Token0 = token0.ID
	pair.Token1 = token1.ID
	pair.Block = entity.NewIntFromLiteralUnsigned(s.Block().Number())
	pair.Timestamp = entity.NewIntFromLiteral(s.Block().Timestamp().Unix())
	pair.Name = fmt.Sprintf("%s-%s", token0.Symbol, token1.Symbol)

	return pair, nil
}

func (s *Subgraph) getToken(tokenAddress eth.Address) (*Token, error) {
	if tokenAddress == nil {
		return nil, nil
	}

	token := NewToken(tokenAddress.Pretty())
	err := s.Load(token)
	if err != nil {
		return nil, err
	}

	if !token.Exists() {
		tm, valid := s.GetTokenInfo(tokenAddress, validateToken)
		if !valid {
			s.Log.Info("token is invalid",
				zap.String("token", tokenAddress.Pretty()),
				zap.Uint64("block_number", s.Block().Number()),
				zap.String("block_id", s.Block().ID()),
			)
			return nil, nil
		}

		token.Name = tm.Name
		token.Symbol = tm.Symbol
		token.Decimals = IL(int64(tm.Decimals))
		token.DerivedETH = FL(0)

		if err := s.Save(token); err != nil {
			return nil, fmt.Errorf("saving token: %w", err)
		}
	}

	return token, nil
}

func validateToken(tok *eth.Token) bool {
	return !tok.IsEmptyDecimal
}