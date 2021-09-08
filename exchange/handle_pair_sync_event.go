package exchange

import (
	"fmt"
	"math/big"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"

	"go.uber.org/zap"
)

func (s *Subgraph) HandlePairSyncEvent(ev *PairSyncEvent) error {
	if s.StepBelow(2) {
		return nil
	}

	pair, err := s.getPair(ev.LogAddress, nil, nil)
	if err != nil {
		return err
	}

	if !pair.Exists() {
		return fmt.Errorf("could not find pair %s", ev.LogAddress.Pretty())
	}

	token0, err := s.getToken(eth.MustNewAddress(pair.Token0))
	if err != nil {
		return fmt.Errorf("loading token 0: %s of pair: %s:%w", pair.Token0, ev.LogAddress.Pretty(), err)
	}
	s.Log.Debug("current derived eth token 0", zap.String("token", token0.Symbol), zap.String("pair_name", pair.Name), zap.Stringer("value", token0.DerivedETH))

	token1, err := s.getToken(eth.MustNewAddress(pair.Token1))
	if err != nil {
		return fmt.Errorf("loading token 1: %s of pair: %s :%w", pair.Token1, ev.LogAddress.Pretty(), err)
	}
	s.Log.Debug("current derived eth token 1", zap.String("token", token1.Symbol), zap.String("pair_name", pair.Name), zap.Stringer("value", token1.DerivedETH))

	factory := NewFactory(FactoryAddress)
	if err := s.Load(factory); err != nil {
		return err
	}

	s.Log.Debug("reserved ETH before removal", zap.String("pair_name", pair.Name), zap.String("value", factory.LiquidityETH.Float().Text('g', -1)))
	// reset factory liquidity by subtracting only tracked liquidity
	factory.LiquidityETH = F(bf().Sub(
		factory.LiquidityETH.Float(),
		pair.TrackedReserveETH.Float(),
	))
	s.Log.Debug("reserved ETH after removal", zap.String("pair_name", pair.Name), zap.String("value", factory.LiquidityETH.Float().Text('g', -1)))

	token0.Liquidity = F(bf().Sub(token0.Liquidity.Float(), pair.Reserve0.Float()))
	token1.Liquidity = F(bf().Sub(token1.Liquidity.Float(), pair.Reserve1.Float()))

	pairReserve0Before := pair.Reserve0
	pairReserve1Before := pair.Reserve1
	pair.Reserve0 = F(entity.ConvertTokenToDecimal(ev.Reserve0, token0.Decimals.Int().Int64()))
	pair.Reserve1 = F(entity.ConvertTokenToDecimal(ev.Reserve1, token1.Decimals.Int().Int64()))

	zlog.Debug("updated pair 0 reserve",
		zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
		zap.String("pair_name", pair.Name),
		zap.String("from", pairReserve0Before.Float().Text('g', -1)),
		zap.String("to", pair.Reserve0.Float().Text('g', -1)),
	)
	zlog.Debug("updated pair 1 reserve",
		zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
		zap.String("pair_name", pair.Name),
		zap.String("from", pairReserve1Before.Float().Text('g', -1)),
		zap.String("to", pair.Reserve1.Float().Text('g', -1)),
	)

	zlog.Debug("pair token0 price before", zap.String("pair_name", pair.Name), zap.String("value", pair.Token0Price.Float().Text('g', -1)))
	if pair.Reserve1.Float().Cmp(bf()) != 0 {
		pair.Token0Price = F(bf().Quo(pair.Reserve0.Float(), pair.Reserve1.Float()))
	} else {
		pair.Token0Price = FL(0)
	}
	zlog.Debug("pair token0 price after", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()), zap.String("pair_name", pair.Name), zap.String("value", pair.Token0Price.Float().Text('g', -1)))

	zlog.Debug("pair token1 price before", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()), zap.String("pair_name", pair.Name), zap.String("value", pair.Token1Price.Float().Text('g', -1)))
	if pair.Reserve0.Float().Cmp(bf()) != 0 {
		pair.Token1Price = F(bf().Quo(pair.Reserve1.Float(), pair.Reserve0.Float()))
	} else {
		pair.Token1Price = FL(0)
	}
	zlog.Debug("pair token1 price after", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()), zap.String("pair_name", pair.Name), zap.String("value", pair.Token1Price.Float().Text('g', -1)))

	err = s.Save(pair)
	if err != nil {
		return err
	}

	if s.StepBelow(3) {
		return nil
	}

	ethPrice, err := s.GetEthPriceInUSD()
	if err != nil {
		return err
	}

	bundle, err := s.getBundle() // creates bundle if it does not exist
	if err != nil {
		return err
	}

	prevEthPrice := bundle.EthPrice
	bundle.EthPrice = F(ethPrice)
	if err := s.Save(bundle); err != nil {
		return err
	}
	s.Log.Debug("updated bundle price", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()), zap.String("pair_name", pair.Name), zap.Reflect("bundle", bundle), zap.Any("prev_eth_price", prevEthPrice), zap.Uint64("block_number", ev.Block.Number), zap.Stringer("transaction_id", ev.Transaction.Hash))

	t0DerivedETH, err := s.FindEthPerToken(token0)
	if err != nil {
		return err
	}
	zlog.Debug("calculated derived ETH price for token0", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()), zap.String("pair_name", pair.Name), zap.String("token", token0.Symbol), zap.String("value", t0DerivedETH.Text('g', -1)))

	t1DerivedETH, err := s.FindEthPerToken(token1)
	if err != nil {
		return err
	}
	zlog.Debug("calculated derived ETH price for token1", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()), zap.String("pair_name", pair.Name), zap.String("token", token1.Symbol), zap.String("value", t1DerivedETH.Text('g', -1)))

	token0.DerivedETH = F(t0DerivedETH)
	token1.DerivedETH = F(t1DerivedETH)

	if err := s.Save(token0); err != nil {
		return err
	}

	if err := s.Save(token1); err != nil {
		return err
	}

	s.Log.Debug("new token prices",
		zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
		zap.String("token0", token0.Symbol),
		zap.String("token0_value", token0.DerivedETH.Float().Text('g', -1)),
		zap.String("token1", token1.Symbol),
		zap.String("token1_value", token1.DerivedETH.Float().Text('g', -1)),
	)

	// get tracked liquidity - will be 0 if neither is in whitelist
	trackedLiquidityETH := big.NewFloat(0)
	if bundle.EthPrice.Float().Cmp(bf()) != 0 {
		trackedLiquidityUSD, err := s.getTrackedLiquidityUSD(pair.Reserve0.Float(), token0, pair.Reserve1.Float(), token1)
		if err != nil {
			return err
		}
		s.Log.Debug("tracked liquidity usd", zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
			zap.String("pair_name", pair.Name), zap.String("value", trackedLiquidityUSD.Text('b', -1)))

		trackedLiquidityETH = bf().Quo(
			trackedLiquidityUSD,
			bundle.EthPrice.Float(),
		)
	}

	s.Log.Debug("new tracked liquidity eth in the pair",
		zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
		zap.String("pair_name", pair.Name),
		zap.String("value", trackedLiquidityETH.Text('g', -1)),
	)

	// use derived amounts within pair
	pair.TrackedReserveETH = F(trackedLiquidityETH)

	s.Log.Debug("calculating pair reserve eth",
		zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
		zap.String("pair_name", pair.Name),
		zap.String("pair.reserve0", pair.Reserve0.Float().Text('b', -1)),
		zap.String("token0.derviedEth", t0DerivedETH.Text('b', -1)),
		zap.String("pair.reserve1", pair.Reserve1.Float().Text('b', -1)),
		zap.String("token0.derviedEth", t1DerivedETH.Text('b', -1)),
	)

	reserveEth := F(bf().Add(
		bf().Mul(
			pair.Reserve0.Float(),
			t0DerivedETH,
		),
		bf().Mul(
			pair.Reserve1.Float(),
			t1DerivedETH,
		),
	))

	pair.ReserveETH = reserveEth

	s.Log.Debug("calculating pair reserve usd",
		zap.Int("step", s.Step()), zap.Uint64("block", s.Block().Number()),
		zap.String("pair_name", pair.Name),
		zap.String("pair.reserve1", pair.ReserveETH.Float().Text('b', -1)),
		zap.String("bundle.EthPrice", bundle.EthPrice.Float().Text('b', -1)),
	)

	pair.ReserveUSD = F(bf().Mul(
		pair.ReserveETH.Float(),
		bundle.EthPrice.Float(),
	))

	// use tracked amounts globally
	factory.LiquidityETH = entity.FloatAdd(factory.LiquidityETH, F(trackedLiquidityETH))
	factory.LiquidityUSD = F(bf().Mul(
		factory.LiquidityETH.Float(),
		ethPrice,
	))

	token0.Liquidity = entity.FloatAdd(token0.Liquidity, pair.Reserve0)
	token1.Liquidity = entity.FloatAdd(token1.Liquidity, pair.Reserve1)

	// save entities
	if err := s.Save(pair); err != nil {
		return err
	}

	if err := s.Save(factory); err != nil {
		return err
	}

	if err := s.Save(token0); err != nil {
		return err
	}

	if err := s.Save(token1); err != nil {
		return err
	}

	return nil
}
