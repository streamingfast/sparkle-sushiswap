package exchange

import (
	"github.com/streamingfast/eth-go"
	"go.uber.org/zap"
	"math/big"
)

var MINIMUM_LIQUIDITY_THRESHOLD_ETH = big.NewFloat(10)

const (
	NATIVE_ADDRESS = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
)

func (s *Subgraph) GetEthPriceInUSD() (*big.Float, error) {
	panic("implement me!")
}

func (s *Subgraph) FindEthPerToken(token *Token) (*big.Float, error) {
	tokenAddress := eth.MustNewAddress(token.GetID()).Pretty()
	if tokenAddress == NATIVE_ADDRESS {
		return big.NewFloat(1), nil
	}

	for _, otherToken := range token.WhitelistPairs {
		pairAddress := s.getPairAddressForTokens(tokenAddress, otherToken)
		if pairAddress == "" {
			s.Log.Debug("pair not found for tokens", zap.String("left", tokenAddress), zap.String("right", otherToken))
			continue
		}

		pair := NewPair(pairAddress)
		if err := s.Load(pair); err != nil {
			return nil, err
		}

		if pair.Token0 == tokenAddress && pair.ReserveETH.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_ETH) > 0 {
			token1 := NewToken(pair.Token1)
			if err := s.Load(token1); err != nil {
				return nil, err
			}
			return bf().Mul(pair.Token1Price.Float(), token1.DerivedETH.Float()), nil
		}
		if pair.Token1 == tokenAddress && pair.ReserveETH.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_ETH) > 0 {
			token0 := NewToken(pair.Token0)
			if err := s.Load(token0); err != nil {
				return nil, err
			}
			return bf().Mul(pair.Token0Price.Float(), token0.DerivedETH.Float()), nil
		}
	}

	return big.NewFloat(0), nil
}
