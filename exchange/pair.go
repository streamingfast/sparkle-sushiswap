package exchange

import (
	"fmt"
	"github.com/streamingfast/sparkle/subgraph"
	"go.uber.org/zap"
	"math/big"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) getPair(pairAddress, token0Address, token1Address eth.Address) (*Pair, error) {
	pair := NewPair(pairAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return nil, err
	}

	if pair.Exists() {
		return pair, nil
	}

	if token0Address == nil && token1Address == nil {
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

	if isWhitelistedAddress(token0.ID) {
		token1.WhitelistPairs = append(token1.WhitelistPairs, pairAddress.Pretty())
	}

	if isWhitelistedAddress(token1.ID) {
		token0.WhitelistPairs = append(token0.WhitelistPairs, pairAddress.Pretty())
	}

	if err := s.Save(token0); err != nil {
		return nil, err
	}

	if err := s.Save(token1); err != nil {
		return nil, err
	}

	pair.Token0 = token0.ID
	pair.Token1 = token1.ID
	pair.Factory = FactoryAddress
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

	if token.Exists() {
		return token, nil
	}

	factory, err := s.getFactory()
	if err != nil {
		return nil, err
	}

	factory.TokenCount = entity.IntAdd(factory.TokenCount, IL(1))
	err = s.Save(factory)
	if err != nil {
		return nil, err
	}

	calls := []*subgraph.RPCCall{
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "decimals() (uint256)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "name() (string)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "symbol() (string)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "totalSupply() (uint256)",
		},
	}

	resps, err := s.RPC(calls)
	if err != nil {
		return nil, fmt.Errorf("rpc call error: %w", err)
	}

	decimalsResponse := resps[0]
	if decimalsResponse.CallError == nil && decimalsResponse.DecodingError == nil {
		token.Decimals = IL(decimalsResponse.Decoded[0].(*big.Int).Int64())
	}

	nameResponse := resps[1]
	if nameResponse.CallError == nil && nameResponse.DecodingError == nil {
		token.Name = nameResponse.Decoded[0].(string)
	} else {
		token.Name = "unknown"
	}

	symbolResponse := resps[2]
	if symbolResponse.CallError == nil && symbolResponse.DecodingError == nil {
		token.Symbol = symbolResponse.Decoded[0].(string)
	} else {
		token.Symbol = "unknown"
	}

	totalSupplyResponse := resps[3]
	if totalSupplyResponse.CallError == nil && totalSupplyResponse.DecodingError == nil {
		token.TotalSupply = IL(totalSupplyResponse.Decoded[0].(*big.Int).Int64())
	}

	token.Factory = factory.ID
	token.DerivedETH = FL(0)
	token.WhitelistPairs = []string{}

	if err := s.Save(token); err != nil {
		return nil, fmt.Errorf("saving token: %w", err)
	}

	return token, nil
}

func (s *Subgraph) getTrackedVolumeUSD(tokenAmount0 *big.Float, token0 *Token, tokenAmount1 *big.Float, token1 *Token, pair *Pair) (*big.Float, error) {
	bundle, err := s.getBundle()
	if err != nil {
		return nil, err
	}

	price0 := bf().Mul(token0.DerivedETH.Float(), bundle.EthPrice.Float())
	price1 := bf().Mul(token1.DerivedETH.Float(), bundle.EthPrice.Float())
	zlog.Debug("bundle", zap.String("pair_name", pair.Name), zap.String("EthPrice", bundle.EthPrice.Float().Text('g', -1)))

	token0Whitelisted := isWhitelistedAddress(token0.ID)
	token1Whitelisted := isWhitelistedAddress(token1.ID)

	// if less than 5 LPs, require high minimum reserve amount amount or return 0
	count := pair.LiquidityProviderCount.Int()
	if count.Cmp(big.NewInt(5)) < 0 {
		reserve0USD := bf().Mul(pair.Reserve0.Float(), price0)
		zlog.Debug("reserve 0 usd", zap.String("pair_name", pair.Name), zap.String("pair_reserve_0", pair.Reserve0.Float().Text('g', -1)), zap.String("price 0", price0.Text('g', -1)), zap.String("value", reserve0USD.Text('g', -1)))
		reserve1USD := bf().Mul(pair.Reserve1.Float(), price1)
		zlog.Debug("reserve 1 usd", zap.String("pair_name", pair.Name), zap.String("pair_reserve_1", pair.Reserve1.Float().Text('g', -1)), zap.String("price 1", price1.Text('g', -1)), zap.String("value", reserve1USD.Text('g', -1)))

		if token0Whitelisted && token1Whitelisted {
			totalReserve := bf().Add(reserve0USD, reserve1USD)
			zlog.Debug("total pair reserve", zap.String("pair_name", pair.Name), zap.String("value", totalReserve.Text('g', -1)))

			if totalReserve.Cmp(MinimumUSDThresholdNewPairs) < 0 {
				zlog.Debug("under minimum threshold. returning 0", zap.String("pair_name", pair.Name))
				return big.NewFloat(0), nil
			}
		}
		if token0Whitelisted && !token1Whitelisted {
			if bf().Mul(reserve0USD, big.NewFloat(2)).Cmp(MinimumUSDThresholdNewPairs) < 0 {
				zlog.Debug("under minimum threshold. returning 0", zap.String("pair_name", pair.Name))
				return big.NewFloat(0), nil
			}
		}
		if !token0Whitelisted && token1Whitelisted {
			if bf().Mul(reserve1USD, big.NewFloat(2)).Cmp(MinimumUSDThresholdNewPairs) < 0 {
				zlog.Debug("under minimum threshold. returning 0", zap.String("pair_name", pair.Name))
				return big.NewFloat(0), nil
			}
		}
	}

	// both are whitelist tokens, take average of both amounts
	if token0Whitelisted && token1Whitelisted {
		sum := bf().Add(
			bf().Mul(tokenAmount0, price0),
			bf().Mul(tokenAmount1, price1),
		)
		avg := bf().Quo(sum, big.NewFloat(2.0))
		return avg, nil
	}

	if token0Whitelisted && !token1Whitelisted {
		// take full value of the whitelisted token amount
		return bf().Mul(tokenAmount0, price0), nil
	}

	if !token0Whitelisted && token1Whitelisted {
		// take full value of the whitelisted token amount
		return bf().Mul(tokenAmount1, price1), nil
	}

	// neither token is on white list, tracked volume is 0
	return big.NewFloat(0), nil
}

func (s *Subgraph) getTrackedLiquidityUSD(tokenAmount0 *big.Float, token0 *Token, tokenAmount1 *big.Float, token1 *Token) (*big.Float, error) {
	bundle, err := s.getBundle()
	if err != nil {
		return nil, err
	}

	price0 := bf().Mul(token0.DerivedETH.Float().SetPrec(100), bundle.EthPrice.Float().SetPrec(100)).SetPrec(100)
	price1 := bf().Mul(token1.DerivedETH.Float().SetPrec(100), bundle.EthPrice.Float().SetPrec(100)).SetPrec(100)

	token0Whitelisted := isWhitelistedAddress(token0.ID)
	token1Whitelisted := isWhitelistedAddress(token1.ID)

	// both are whitelist tokens, take average of both amounts
	if token0Whitelisted && token1Whitelisted {
		return bf().Add(
			bf().Mul(tokenAmount0, price0).SetPrec(100),
			bf().Mul(tokenAmount1, price1).SetPrec(100),
		).SetPrec(100), nil
	}

	floatTwo := big.NewFloat(2)
	if token0Whitelisted && !token1Whitelisted {
		// take double value of the whitelisted token amount
		return bf().Mul(
			bf().Mul(tokenAmount0, price0).SetPrec(100),
			floatTwo,
		).SetPrec(100), nil
	}

	if !token0Whitelisted && token1Whitelisted {
		// take double value of the whitelisted token amount
		return bf().Mul(
			bf().Mul(tokenAmount1, price1).SetPrec(100),
			floatTwo,
		).SetPrec(100), nil
	}

	// neither token is on white list, tracked volume is 0
	return big.NewFloat(0), nil
}
