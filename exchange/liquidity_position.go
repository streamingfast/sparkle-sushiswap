package exchange

import (
	"fmt"

	"github.com/streamingfast/eth-go"
)

func (s *Subgraph) createLiquidityPosition(user, pair eth.Address) (*LiquidityPosition, error) {
	id := fmt.Sprintf("%s-%s", pair.Pretty(), user.Pretty())

	position := NewLiquidityPosition(id)
	if err := s.Load(position); err != nil {
		return nil, err
	}

	if !position.Exists() {
		position.User = user.Pretty()
		position.Pair = pair.Pretty()

		position.Timestamp = s.Block().Timestamp().Unix()
		position.Block = int64(s.Block().Number())

		position.LiquidityTokenBalance = FL(0)

		if err := s.Save(position); err != nil {
			return nil, err
		}
	}

	return position, nil
}

func (s *Subgraph) createLiquidityPositionSnapshot(position *LiquidityPosition) error {
	id := fmt.Sprintf("%s-%d", position.ID, s.Block().Timestamp().Unix())

	bundle, err := s.getBundle()
	if err != nil {
		return err
	}

	pair, err := s.getPair(eth.MustNewAddress(position.Pair), nil, nil)
	if err != nil {
		return err
	}

	token0, err := s.getToken(eth.MustNewAddress(pair.Token0))
	if err != nil {
		return err
	}

	token1, err := s.getToken(eth.MustNewAddress(pair.Token1))
	if err != nil {
		return err
	}

	snapshot := NewLiquidityPositionSnapshot(id)
	snapshot.Timestamp = s.Block().Timestamp().Unix()
	snapshot.User = position.User
	snapshot.Pair = position.Pair
	snapshot.Token0PriceUSD = F(bf().Mul(token0.DerivedETH.Float(), bundle.EthPrice.Float()))
	snapshot.Token1PriceUSD = F(bf().Mul(token1.DerivedETH.Float(), bundle.EthPrice.Float()))
	snapshot.Reserve0 = pair.Reserve0
	snapshot.Reserve1 = pair.Reserve1
	snapshot.ReserveUSD = pair.ReserveUSD
	snapshot.LiquidityTokenTotalSupply = pair.TotalSupply
	snapshot.LiquidityTokenBalance = position.LiquidityTokenBalance
	snapshot.LiquidityPosition = position.ID

	err = s.Save(snapshot)
	if err != nil {
		return err
	}

	return nil
}
