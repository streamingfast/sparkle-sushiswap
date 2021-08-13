package exchange

import (
	"github.com/streamingfast/eth-go"
	"go.uber.org/zap"
	"math/big"
)

var MINIMUM_LIQUIDITY_THRESHOLD_ETH = big.NewFloat(10)

const (
	NATIVE_ADDRESS = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

	DAI_WETH_PAIR = "0xc3d03e4f041fd4cd388c549ee2a29a9e5075882f"
	USDC_WETH_PAIR = "0x397ff1542f962076d0bfe58ea045ffa2d347aca0"
	USDT_WETH_PAIR = "0x06da0fd433c1a5d7a4faa01111c044910a184553"

	DAI = "0x6b175474e89094c44da98b954eedeac495271d0f"
	USDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	USDT = "0xdac17f958d2ee523a2206206994597c13d831ec7"
)

func (s *Subgraph) GetEthPriceInUSD() (*big.Float, error) {
	daiPair, err := s.getPair(eth.MustNewAddress(DAI_WETH_PAIR), nil, nil)
	if err != nil {
		return nil, err
	}
	usdcPair, err := s.getPair(eth.MustNewAddress(USDC_WETH_PAIR), nil, nil)
	if err != nil {
		return nil, err
	}
	usdtPair, err := s.getPair(eth.MustNewAddress(USDT_WETH_PAIR), nil, nil)
	if err != nil {
		return nil, err
	}

	isDaiPairLiquidEnough := bool(daiPair.ReserveETH.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_ETH) > 0)
	isUsdcPairLiquidEnough := bool(usdcPair.ReserveETH.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_ETH) > 0)
	isUsdtPairLiquidEnough := bool(usdtPair.ReserveETH.Float().Cmp(MINIMUM_LIQUIDITY_THRESHOLD_ETH) > 0)

	if daiPair.Exists() && isDaiPairLiquidEnough && usdcPair.Exists() && isUsdcPairLiquidEnough && usdtPair.Exists() && isUsdtPairLiquidEnough {
		isDaiFirst := daiPair.Token0 == DAI
		isUsdcFirst := usdcPair.Token0 == USDC
		isUsdtFirst := usdtPair.Token0 == USDT

		var daiPairEth *big.Float
		if isDaiFirst {
			daiPairEth = daiPair.Reserve1.Float()
		} else {
			daiPairEth = daiPair.Reserve0.Float()
		}

		var usdcPairEth *big.Float
		if isUsdcFirst {
			usdcPairEth = usdcPair.Reserve1.Float()
		} else {
			usdcPairEth = usdcPair.Reserve0.Float()
		}

		var usdtPairEth *big.Float
		if isUsdtFirst {
			usdtPairEth = usdtPair.Reserve1.Float()
		} else {
			usdtPairEth = usdtPair.Reserve0.Float()
		}

		totalLiquidityEth := bf().Add(daiPairEth, bf().Add(usdcPairEth, usdtPairEth))

		var daiWeight *big.Float
		if !isDaiFirst {
			daiWeight = bf().Quo(daiPair.Reserve0.Float(), totalLiquidityEth)
		} else {
			daiWeight = bf().Quo(daiPair.Reserve1.Float(), totalLiquidityEth)
		}

		var usdcWeight *big.Float
		if !isDaiFirst {
			usdcWeight = bf().Quo(usdcPair.Reserve0.Float(), totalLiquidityEth)
		} else {
			usdcWeight = bf().Quo(usdcPair.Reserve1.Float(), totalLiquidityEth)
		}

		var usdtWeight *big.Float
		if !isDaiFirst {
			usdtWeight = bf().Quo(usdtPair.Reserve0.Float(), totalLiquidityEth)
		} else {
			usdtWeight = bf().Quo(usdtPair.Reserve1.Float(), totalLiquidityEth)
		}

		var daiPrice *big.Float
		if isDaiFirst {
			daiPrice = daiPair.Token0Price.Float()
		} else {
			daiPrice = daiPair.Token1Price.Float()
		}

		var usdcPrice *big.Float
		if isUsdcFirst {
			usdcPrice = usdcPair.Token0Price.Float()
		} else {
			usdcPrice = usdcPair.Token1Price.Float()
		}

		var usdtPrice *big.Float
		if isUsdtFirst {
			usdtPrice = usdtPair.Token0Price.Float()
		} else {
			usdtPrice = usdtPair.Token1Price.Float()
		}

		weightedDaiPrice := bf().Mul(daiPrice, daiWeight).SetPrec(100)
		weightedUsdcPrice := bf().Mul(usdcPrice, usdcWeight).SetPrec(100)
		weightedUsdtPrice := bf().Mul(usdtPrice, usdtWeight).SetPrec(100)

		return bf().Add(weightedDaiPrice, bf().Add(weightedUsdcPrice, weightedUsdtPrice)).SetPrec(100), nil
	} else if daiPair.Exists() && isDaiPairLiquidEnough && usdcPair.Exists() && isUsdcPairLiquidEnough {
		isDaiFirst := daiPair.Token0 == DAI
		isUsdcFirst := usdcPair.Token0 == USDC

		var daiPairEth *big.Float
		if isDaiFirst {
			daiPairEth = daiPair.Reserve1.Float()
		} else {
			daiPairEth = daiPair.Reserve0.Float()
		}

		var usdcPairEth *big.Float
		if isUsdcFirst {
			usdcPairEth = usdcPair.Reserve1.Float()
		} else {
			usdcPairEth = usdcPair.Reserve0.Float()
		}

		totalLiquidityEth := bf().Add(daiPairEth, usdcPairEth)

		var daiWeight *big.Float
		if !isDaiFirst {
			daiWeight = bf().Quo(daiPair.Reserve0.Float(), totalLiquidityEth)
		} else {
			daiWeight = bf().Quo(daiPair.Reserve1.Float(), totalLiquidityEth)
		}

		var usdcWeight *big.Float
		if !isDaiFirst {
			usdcWeight = bf().Quo(usdcPair.Reserve0.Float(), totalLiquidityEth)
		} else {
			usdcWeight = bf().Quo(usdcPair.Reserve1.Float(), totalLiquidityEth)
		}

		var daiPrice *big.Float
		if isDaiFirst {
			daiPrice = daiPair.Token0Price.Float()
		} else {
			daiPrice = daiPair.Token1Price.Float()
		}

		var usdcPrice *big.Float
		if isUsdcFirst {
			usdcPrice = usdcPair.Token0Price.Float()
		} else {
			usdcPrice = usdcPair.Token1Price.Float()
		}

		weightedDaiPrice := bf().Mul(daiPrice, daiWeight).SetPrec(100)
		weightedUsdcPrice := bf().Mul(usdcPrice, usdcWeight).SetPrec(100)

		return bf().Add(weightedDaiPrice, weightedUsdcPrice).SetPrec(100), nil
	} else if usdcPair.Exists() && isUsdcPairLiquidEnough {
		isUsdcFirst := usdcPair.Token0 == USDC

		var usdcPrice *big.Float
		if isUsdcFirst {
			usdcPrice = usdcPair.Token0Price.Float()
		} else {
			usdcPrice = usdcPair.Token1Price.Float()
		}

		return usdcPrice.SetPrec(100), nil
	} else if usdtPair.Exists() && isUsdtPairLiquidEnough {
		isUsdtFirst := usdtPair.Token0 == USDT

		var usdtPrice *big.Float
		if isUsdtFirst {
			usdtPrice = usdtPair.Token0Price.Float()
		} else {
			usdtPrice = usdtPair.Token1Price.Float()
		}

		return usdtPrice.SetPrec(100), nil
	} else if daiPair.Exists() && isDaiPairLiquidEnough {
		isDaiFirst := daiPair.Token0 == DAI

		var daiPrice *big.Float
		if isDaiFirst {
			daiPrice = daiPair.Token0Price.Float()
		} else {
			daiPrice = daiPair.Token1Price.Float()
		}

		return daiPrice.SetPrec(100), nil
	}

	return big.NewFloat(0), nil
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
