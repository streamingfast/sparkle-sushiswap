package exchange

import (
	"math/big"
	"strings"

	"github.com/streamingfast/eth-go"

	"go.uber.org/zap"
)

var (
	MinimumLiquidityThresholdEth = big.NewFloat(10)
	MinimumUSDThresholdNewPairs  = big.NewFloat(3000.0)
)

const (
	NativeAddress = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

	DaiWethPair  = "0xc3d03e4f041fd4cd388c549ee2a29a9e5075882f"
	UsdcWethPair = "0x397ff1542f962076d0bfe58ea045ffa2d347aca0"
	UsdtWethPair = "0x06da0fd433c1a5d7a4faa01111c044910a184553"

	DAI  = "0x6b175474e89094c44da98b954eedeac495271d0f"
	USDC = "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	USDT = "0xdac17f958d2ee523a2206206994597c13d831ec7"
)

func (s *Subgraph) GetEthPriceInUSD() (*big.Float, error) {
	daiPair, err := s.getPair(eth.MustNewAddress(DaiWethPair), nil, nil)
	if err != nil {
		return nil, err
	}
	usdcPair, err := s.getPair(eth.MustNewAddress(UsdcWethPair), nil, nil)
	if err != nil {
		return nil, err
	}
	usdtPair, err := s.getPair(eth.MustNewAddress(UsdtWethPair), nil, nil)
	if err != nil {
		return nil, err
	}

	isDaiPairLiquidEnough := bool(daiPair.ReserveETH.Float().Cmp(MinimumLiquidityThresholdEth) > 0)
	isUsdcPairLiquidEnough := bool(usdcPair.ReserveETH.Float().Cmp(MinimumLiquidityThresholdEth) > 0)
	isUsdtPairLiquidEnough := bool(usdtPair.ReserveETH.Float().Cmp(MinimumLiquidityThresholdEth) > 0)

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

		totalLiquidityEth := bf().Add(daiPairEth, bf().Add(usdcPairEth, usdtPairEth).SetPrec(100)).SetPrec(100)

		var daiWeight *big.Float
		if !isDaiFirst {
			daiWeight = bf().Quo(daiPair.Reserve0.Float(), totalLiquidityEth).SetPrec(100)
		} else {
			daiWeight = bf().Quo(daiPair.Reserve1.Float(), totalLiquidityEth).SetPrec(100)
		}

		var usdcWeight *big.Float
		if !isDaiFirst {
			usdcWeight = bf().Quo(usdcPair.Reserve0.Float(), totalLiquidityEth).SetPrec(100)
		} else {
			usdcWeight = bf().Quo(usdcPair.Reserve1.Float(), totalLiquidityEth).SetPrec(100)
		}

		var usdtWeight *big.Float
		if !isDaiFirst {
			usdtWeight = bf().Quo(usdtPair.Reserve0.Float(), totalLiquidityEth).SetPrec(100)
		} else {
			usdtWeight = bf().Quo(usdtPair.Reserve1.Float(), totalLiquidityEth).SetPrec(100)
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
		weightedPrice := bf().Add(weightedDaiPrice, bf().Add(weightedUsdcPrice, weightedUsdtPrice)).SetPrec(100)

		s.Log.Debug("eth price calculated from dai/usdc", zap.Stringer("price", weightedPrice))
		return weightedPrice, nil
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

		totalLiquidityEth := bf().Add(daiPairEth, usdcPairEth).SetPrec(100)

		var daiWeight *big.Float
		if !isDaiFirst {
			daiWeight = bf().Quo(daiPair.Reserve0.Float(), totalLiquidityEth).SetPrec(100)
		} else {
			daiWeight = bf().Quo(daiPair.Reserve1.Float(), totalLiquidityEth).SetPrec(100)
		}

		var usdcWeight *big.Float
		if !isDaiFirst {
			usdcWeight = bf().Quo(usdcPair.Reserve0.Float(), totalLiquidityEth).SetPrec(100)
		} else {
			usdcWeight = bf().Quo(usdcPair.Reserve1.Float(), totalLiquidityEth).SetPrec(100)
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
		weightedPrice := bf().Add(weightedDaiPrice, weightedUsdcPrice).SetPrec(100)

		s.Log.Debug("eth price calculated from dai/usdc", zap.Stringer("price", weightedPrice))
		return weightedPrice, nil
	} else if usdcPair.Exists() && isUsdcPairLiquidEnough {
		isUsdcFirst := usdcPair.Token0 == USDC

		var usdcPrice *big.Float
		if isUsdcFirst {
			usdcPrice = usdcPair.Token0Price.Float()
		} else {
			usdcPrice = usdcPair.Token1Price.Float()
		}

		s.Log.Debug("eth price calculated from usdc", zap.Stringer("price", usdcPrice))
		return usdcPrice.SetPrec(100), nil
	} else if usdtPair.Exists() && isUsdtPairLiquidEnough {
		isUsdtFirst := usdtPair.Token0 == USDT

		var usdtPrice *big.Float
		if isUsdtFirst {
			usdtPrice = usdtPair.Token0Price.Float()
		} else {
			usdtPrice = usdtPair.Token1Price.Float()
		}

		s.Log.Debug("eth price calculated from usdt", zap.Stringer("price", usdtPrice))
		return usdtPrice.SetPrec(100), nil
	} else if daiPair.Exists() && isDaiPairLiquidEnough {
		isDaiFirst := daiPair.Token0 == DAI

		var daiPrice *big.Float
		if isDaiFirst {
			daiPrice = daiPair.Token0Price.Float()
		} else {
			daiPrice = daiPair.Token1Price.Float()
		}

		s.Log.Debug("eth price calculated from dai", zap.Stringer("price", daiPrice))
		return daiPrice.SetPrec(100), nil
	}

	s.Log.Debug("eth price could not be calculated")
	return big.NewFloat(0), nil
}

func (s *Subgraph) FindEthPerToken(token *Token) (*big.Float, error) {
	tokenAddress := eth.MustNewAddress(token.GetID()).Pretty()
	if tokenAddress == NativeAddress {
		return big.NewFloat(1), nil
	}

	for _, pairAddress := range token.WhitelistPairs {
		pair := NewPair(pairAddress)
		if err := s.Load(pair); err != nil {
			return nil, err
		}

		s.Log.Debug("",
			zap.Bool("pair exists", pair.Exists()),
			zap.Stringer("pair_token0_price", pair.Token0Price),
			zap.Stringer("pair_token1_price", pair.Token1Price),
		)

		if pair.Token0 == tokenAddress && pair.ReserveETH.Float().Cmp(MinimumLiquidityThresholdEth) > 0 {
			token1 := NewToken(pair.Token1)
			if err := s.Load(token1); err != nil {
				return nil, err
			}
			s.Log.Debug("matched on token 0", zap.Bool("token1_exists", token1.Exists()), zap.Stringer("token1_derived_eth", token1.DerivedETH))
			return bf().Mul(pair.Token1Price.Float(), token1.DerivedETH.Float()), nil
		}
		if pair.Token1 == tokenAddress && pair.ReserveETH.Float().Cmp(MinimumLiquidityThresholdEth) > 0 {
			token0 := NewToken(pair.Token0)
			if err := s.Load(token0); err != nil {
				return nil, err
			}
			s.Log.Debug("matched on token 0", zap.Bool("token1_exists", token0.Exists()), zap.Stringer("token1_derived_eth", token0.DerivedETH))
			return bf().Mul(pair.Token0Price.Float(), token0.DerivedETH.Float()), nil
		}
	}

	s.Log.Debug("no whitelisted pairs")
	return big.NewFloat(0), nil
}

func getTrackedVolumeUSD(bundle *Bundle, tokenAmount0 *big.Float, token0 *Token, tokenAmount1 *big.Float, token1 *Token, pair *Pair) *big.Float {
	price0 := bf().Mul(token0.DerivedETH.Float(), bundle.EthPrice.Float())
	price1 := bf().Mul(token1.DerivedETH.Float(), bundle.EthPrice.Float())

	token0Whitelisted := isWhitelistedAddress(token0.ID)
	token1Whitelisted := isWhitelistedAddress(token1.ID)

	// if less than 5 LPs, require high minimum reserve amount amount or return 0
	count := pair.LiquidityProviderCount.Int()
	if count.Cmp(big.NewInt(5)) < 0 {
		reserve0USD := bf().Mul(pair.Reserve0.Float(), price0)
		reserve1USD := bf().Mul(pair.Reserve1.Float(), price0)

		if token0Whitelisted && token1Whitelisted {
			if bf().Add(reserve0USD, reserve1USD).Cmp(MinimumUSDThresholdNewPairs) < 0 {
				return big.NewFloat(0)
			}
		}
		if token0Whitelisted && !token1Whitelisted {
			if bf().Mul(reserve0USD, big.NewFloat(2)).Cmp(MinimumUSDThresholdNewPairs) < 0 {
				return big.NewFloat(0)
			}
		}
		if !token0Whitelisted && token1Whitelisted {
			if bf().Mul(reserve1USD, big.NewFloat(2)).Cmp(MinimumUSDThresholdNewPairs) < 0 {
				return big.NewFloat(0)
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
		return avg
	}

	if token0Whitelisted && !token1Whitelisted {
		// take full value of the whitelisted token amount
		return bf().Mul(tokenAmount0, price0)
	}

	if !token0Whitelisted && token1Whitelisted {
		// take full value of the whitelisted token amount
		return bf().Mul(tokenAmount1, price1)
	}

	// neither token is on white list, tracked volume is 0
	return big.NewFloat(0)
}

func getTrackedLiquidityUSD(bundle *Bundle, tokenAmount0 *big.Float, token0 *Token, tokenAmount1 *big.Float, token1 *Token) *big.Float {
	price0 := bf().Mul(token0.DerivedETH.Float().SetPrec(100), bundle.EthPrice.Float().SetPrec(100)).SetPrec(100)
	price1 := bf().Mul(token1.DerivedETH.Float().SetPrec(100), bundle.EthPrice.Float().SetPrec(100)).SetPrec(100)

	token0Whitelisted := isWhitelistedAddress(token0.ID)
	token1Whitelisted := isWhitelistedAddress(token1.ID)

	// both are whitelist tokens, take average of both amounts
	if token0Whitelisted && token1Whitelisted {
		return bf().Add(
			bf().Mul(tokenAmount0, price0).SetPrec(100),
			bf().Mul(tokenAmount1, price1).SetPrec(100),
		).SetPrec(100)
	}

	floatTwo := big.NewFloat(2)
	if token0Whitelisted && !token1Whitelisted {
		// take double value of the whitelisted token amount
		return bf().Mul(
			bf().Mul(tokenAmount0, price0).SetPrec(100),
			floatTwo,
		).SetPrec(100)
	}

	if !token0Whitelisted && token1Whitelisted {
		// take double value of the whitelisted token amount
		return bf().Mul(
			bf().Mul(tokenAmount1, price1).SetPrec(100),
			floatTwo,
		).SetPrec(100)
	}

	// neither token is on white list, tracked volume is 0
	return big.NewFloat(0)
}

// whitelist is a slice because we need to respect the order when using it in certain location, so
// we must not converted to a map[string]bool directly unless there is a strict ordering way to list them.
var whitelist = []string{
	"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
	"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
	"0x6b175474e89094c44da98b954eedeac495271d0f",
	"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	"0xdac17f958d2ee523a2206206994597c13d831ec7",
	"0x0000000000085d4780b73119b644ae5ecd22b376",
	"0x5d3a536e4d6dbd6114cc1ead35777bab948e3643",
	"0x57ab1ec28d129707052df4df418d58a2d46d5f51",
	"0x514910771af9ca656af840dff83e8264ecf986ca",
	"0x0bc529c00c6401aef6d220be8c6ea1667f6ad93e",
	"0x8798249c2e607446efb7ad49ec89dd1865ff4272",
	"0x1456688345527be1f37e9e627da0837d6f08c925",
	"0x3449fc1cd036255ba1eb19d65ff4ba2b8903a69a",
	"0x2ba592f78db6436527729929aaf6c908497cb200",
	"0x3432b6a60d23ca0dfca7761b7ab56459d9c964d0",
	"0xa1faa113cbe53436df28ff0aee54275c13b40975",
	"0xdb0f18081b505a7de20b18ac41856bcb4ba86a1a",
	"0x04fa0d235c4abf4bcf4787af4cf447de572ef828",
	"0x3155ba85d5f96b2d030a4966af206230e46849cb",
	"0x87d73e916d7057945c9bcd8cdd94e42a6f47f776",
	"0xdfe66b14d37c77f4e9b180ceb433d1b164f0281d",
	"0xad32a8e6220741182940c5abf610bde99e737b2d",
	"0xafcE9B78D409bF74980CACF610AFB851BF02F257",
	"0x6b3595068778dd592e39a122f4f5a5cf09c90fe2",
}

var blacklist = []string{
	"0x9ea3b5b4ec044b70375236a281986106457b20ef",
}

var whitelistCacheMap = map[string]bool{}
var blacklistCacheMap = map[string]bool{}

func isWhitelistedAddress(address string) bool {
	address = strings.ToLower(address)

	if _, ok := whitelistCacheMap[address]; ok {
		return true
	}

	for _, addr := range whitelist {
		if strings.ToLower(addr) != address {
			continue
		}

		whitelistCacheMap[address] = true
		return true
	}

	return false
}

func isBlacklistedAddress(address string) bool {
	address = strings.ToLower(address)

	if _, ok := blacklistCacheMap[address]; ok {
		return true
	}

	for _, addr := range blacklist {
		if strings.ToLower(addr) != address {
			continue
		}

		blacklistCacheMap[address] = true
		return true
	}

	return false
}
