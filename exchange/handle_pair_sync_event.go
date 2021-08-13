package exchange

import (
	"fmt"
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
	"go.uber.org/zap"
	"math/big"
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
	if err := s.Load(token0); err != nil {
		return fmt.Errorf("loading token 0: %s of pair: %s:%w", pair.Token0, ev.LogAddress.Pretty(), err)
	}

	token1, err := s.getToken(eth.MustNewAddress(pair.Token1))
	if err := s.Load(token1); err != nil {
		return fmt.Errorf("loading token 1: %s of pair: %s :%w", pair.Token1, ev.LogAddress.Pretty(), err)
	}

	factory := NewFactory(FactoryAddress)
	if err := s.Load(factory); err != nil {
		return err
	}

	s.Log.Debug("handler sync pre dump",
		zap.Reflect("token0", token0),
		zap.Reflect("token1", token1),
		zap.Reflect("pancake", factory),
		zap.Reflect("pair", pair),
	)

	// reset factory liquidity by subtracting only tracked liquidity
	factory.LiquidityETH = F(bf().Sub(
		factory.LiquidityETH.Float(),
		pair.ReserveETH.Float(),
	))

	s.Log.Debug("removed tracked reserved BNB", zap.Stringer("value", factory.LiquidityETH.Float()))

	token0.Liquidity = F(bf().Sub(token0.Liquidity.Float(), pair.Reserve0.Float()))
	token1.Liquidity = F(bf().Sub(token1.Liquidity.Float(), pair.Reserve1.Float()))

	pair.Reserve0 = F(entity.ConvertTokenToDecimal(ev.Reserve0, token0.Decimals.Int().Int64()))
	pair.Reserve1 = F(entity.ConvertTokenToDecimal(ev.Reserve1, token1.Decimals.Int().Int64()))

	if pair.Reserve1.Float().Cmp(bf()) != 0 {
		pair.Token0Price = F(bf().Quo(pair.Reserve0.Float(), pair.Reserve1.Float()))
	} else {
		pair.Token0Price = FL(0)
	}

	if pair.Reserve0.Float().Cmp(bf()) != 0 {
		pair.Token1Price = F(bf().Quo(pair.Reserve1.Float(), pair.Reserve0.Float()))
	} else {
		pair.Token1Price = FL(0)
	}

	zlog.Debug("set token prices",
		zap.Stringer("pair.token_0_price", pair.Token0Price),
		zap.Stringer("pair.token_1_price", pair.Token1Price),
	)

	// We need to compute the ETH price *before* we save the pair (code just below)
	// the reason for this, is that we don't want the reserver that are set above to affect
	// the calcualtion of the ETH price (this was taken from the typsecript code)
	ethPrice, err := s.GetEthPriceInUSD()
	if err != nil {
		return err
	}

	if s.StepBelow(3) {
		// In parralel reproc, we are ending here if step is below 3, as such, we need to save the pair right away
		s.Log.Debug("updated pair", zap.Reflect("pair", pair))
		if err := s.Save(pair); err != nil {
			return err
		}

		return nil
	}

	bundle := NewBundle("1")
	if err := s.Load(bundle); err != nil {
		return err
	}

	prevEthPrice := bundle.EthPrice
	bundle.EthPrice = F(ethPrice)
	if err := s.Save(bundle); err != nil {
		return err
	}
	s.Log.Debug("updated bundle price", zap.Reflect("bundle", bundle), zap.Any("prev_bnb_price", prevEthPrice), zap.Uint64("block_number", ev.Block.Number), zap.Stringer("transaction_id", ev.Transaction.Hash))

	t0DerivedETH, err := s.FindEthPerToken(token0)
	if err != nil {
		return err
	}

	zlog.Debug("calculated derived ETH price for token0", zap.String("value", t0DerivedETH.Text('g', -1)))

	token0.DerivedETH = F(t0DerivedETH)
	if err := s.Save(token0); err != nil {
		return err
	}

	t1DerivedETH, err := s.FindEthPerToken(token1)
	if err != nil {
		return err
	}

	zlog.Debug("calculated derived ETH price for token1", zap.String("value", t1DerivedETH.Text('g', -1)))

	token1.DerivedETH = F(t1DerivedETH)
	if err := s.Save(token1); err != nil {
		return err
	}

	s.Log.Debug("new token prices",
		zap.Stringer("token0", token0.DerivedETH.Float()),
		zap.Stringer("token1", token1.DerivedETH.Float()),
	)

	// get tracked liquidity - will be 0 if neither is in whitelist
	trackedLiquidityETH := big.NewFloat(0)
	if ethPrice.Cmp(bf()) != 0 {
		tr := getTrackedLiquidityUSD(bundle, pair.Reserve0.Float(), token0, pair.Reserve1.Float(), token1)
		trackedLiquidityETH = bf().Quo(
			tr,
			ethPrice,
		)
	}

	s.Log.Debug("new tracked liquidity bnb in the pair",
		zap.String("value", trackedLiquidityETH.Text('g', -1)),
	)

	// use derived amounts within pair
	pair.ReserveETH = F(trackedLiquidityETH)

	pair.ReserveETH = F(bf().Add(
		bf().Mul(
			pair.Reserve0.Float(),
			t0DerivedETH,
		),
		bf().Mul(
			pair.Reserve1.Float(),
			t1DerivedETH,
		),
	))

	pair.ReserveUSD = F(bf().Mul(
		pair.ReserveETH.Float(),
		ethPrice,
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
