package exchange

import (
	"fmt"
	"math/big"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) HandlePairSwapEvent(ev *PairSwapEvent) error {
	if s.StepBelow(3) {
		return nil
	}

	pair, err := s.getPair(ev.LogAddress, nil, nil)
	if err != nil {
		return fmt.Errorf("loading pair: %w", err)
	}

	token0, err := s.getToken(eth.MustNewAddress(pair.Token0))
	if err != nil {
		return fmt.Errorf("loading initialToken 0: %w", err)
	}

	token1, err := s.getToken(eth.MustNewAddress(pair.Token1))
	if err != nil {
		return fmt.Errorf("loading initialToken 1: %w", err)
	}

	amount0In := entity.ConvertTokenToDecimal(ev.Amount0In, token0.Decimals.Int().Int64())
	amount1In := entity.ConvertTokenToDecimal(ev.Amount1In, token1.Decimals.Int().Int64())
	amount0Out := entity.ConvertTokenToDecimal(ev.Amount0Out, token0.Decimals.Int().Int64())
	amount1Out := entity.ConvertTokenToDecimal(ev.Amount1Out, token1.Decimals.Int().Int64())

	// totals for volume updateTradeVolumes
	amount0Total := bf().Add(amount0Out, amount0In)
	amount1Total := bf().Add(amount1Out, amount1In)

	// ETH/USD prices
	bundle, err := s.getBundle()
	if err != nil {
		return err
	}

	// get total amounts of derived USD and ETH for tracking
	derivedAmountETH := bf().Quo(
		bf().Add(
			bf().Mul(token1.DerivedETH.Float(), amount1Total),
			bf().Mul(token0.DerivedETH.Float(), amount0Total),
		),
		big.NewFloat(2),
	)

	derivedAmountUSD := bf().Mul(derivedAmountETH, bundle.EthPrice.Float())

	// only accounts for volume through white listed tokens
	trackedAmountUSD := getTrackedVolumeUSD(bundle, amount0Total, token0, amount1Total, token1, pair)

	var trackedAmountETH *big.Float
	if bundle.EthPrice.Float().Cmp(big.NewFloat(0)) == 0 {
		trackedAmountETH = big.NewFloat(0)
	} else {
		trackedAmountETH = bf().Quo(trackedAmountUSD, bundle.EthPrice.Float())
	}

	// @ steps 3 trade  volume is realtive per shard
	// @ steps 4 is where you should sqaush and it becomes absolute and that where you can save eneities

	// update token0 global volume and initialToken liquidity stats

	token0.Volume = entity.FloatAdd(token0.Volume, F(bf().Add(amount0In, amount0Out)))
	token0.VolumeUSD = entity.FloatAdd(token0.VolumeUSD, F(trackedAmountUSD))
	token0.UntrackedVolumeUSD = entity.FloatAdd(token0.UntrackedVolumeUSD, F(derivedAmountUSD))

	// update token1 global volume and initialToken liquidity stats
	token1.Volume = entity.FloatAdd(token1.Volume, F(bf().Add(amount1In, amount1Out)))
	token1.VolumeUSD = entity.FloatAdd(token1.VolumeUSD, F(trackedAmountUSD))
	token1.UntrackedVolumeUSD = entity.FloatAdd(token0.UntrackedVolumeUSD, F(derivedAmountUSD))

	// update txn counts
	token0.TxCount = entity.IntAdd(token0.TxCount, IL(1))
	token1.TxCount = entity.IntAdd(token1.TxCount, IL(1))

	// update pair volume data, use tracked amount if we have it as its probably more accurate
	pair.VolumeUSD = entity.FloatAdd(pair.VolumeUSD, F(trackedAmountUSD))
	pair.VolumeToken0 = entity.FloatAdd(pair.VolumeToken0, F(amount0Total))
	pair.VolumeToken1 = entity.FloatAdd(pair.VolumeToken1, F(amount1Total))
	pair.UntrackedVolumeUSD = entity.FloatAdd(pair.UntrackedVolumeUSD, F(derivedAmountUSD))

	pair.TxCount = entity.IntAdd(pair.TxCount, IL(1))
	if err := s.Save(pair); err != nil {
		return fmt.Errorf("saving pair: %w", err)
	}

	// update global values, only used tracked amounts for volume
	factory := NewFactory(FactoryAddress)
	err = s.Load(factory)
	if err != nil {
		return fmt.Errorf("loading pancake factory: %w", err)
	}

	factory.VolumeUSD = entity.FloatAdd(factory.VolumeUSD, F(trackedAmountUSD))
	factory.VolumeETH = entity.FloatAdd(factory.VolumeETH, F(trackedAmountETH))
	factory.UntrackedVolumeUSD = entity.FloatAdd(factory.UntrackedVolumeUSD, F(derivedAmountUSD))

	factory.TxCount = entity.IntAdd(factory.TxCount, IL(1))
	// save entities

	if err := s.Save(token0); err != nil {
		return fmt.Errorf("saving initialToken 0: %w", err)
	}

	if err := s.Save(token1); err != nil {
		return fmt.Errorf("saving initialToken 1: %w", err)
	}

	if err := s.Save(factory); err != nil {
		return fmt.Errorf("saving pancake: %w", err)
	}

	transaction := NewTransaction(ev.Transaction.Hash.Pretty())
	err = s.Load(transaction)
	if err != nil {
		return fmt.Errorf("loading transaction: %w", err)
	}

	if !transaction.Exists() {
		block := s.Block()

		transaction.BlockNumber = IL(int64(block.Number()))
		transaction.Timestamp = IL(block.Timestamp().Unix())
	}

	swap := NewSwap(fmt.Sprintf("%s-%d", transaction.ID, len(transaction.Swaps)))

	// update swap event
	swap.Transaction = transaction.ID
	swap.Pair = pair.ID
	swap.Timestamp = transaction.Timestamp
	swap.Sender = ev.Sender.Pretty()
	swap.Amount0In = F(amount0In)
	swap.Amount1In = F(amount1In)
	swap.Amount0Out = F(amount0Out)
	swap.Amount1Out = F(amount1Out)
	swap.To = ev.To.Pretty()
	swap.LogIndex = IL(int64(ev.LogIndex)).Ptr()

	// use the tracked amount if we have it
	if trackedAmountUSD.Cmp(big.NewFloat(0)) == 0 {
		swap.AmountUSD = F(derivedAmountUSD)
	} else {
		swap.AmountUSD = F(trackedAmountUSD)
	}

	if err := s.Save(swap); err != nil {
		return fmt.Errorf("saving swap: %w", err)
	}

	transaction.Swaps = append(transaction.Swaps, swap.ID)

	if err := s.Save(transaction); err != nil {
		return fmt.Errorf("saving transaction: %w", err)
	}

	pairDayData, err := s.UpdatePairDayData(ev.LogAddress)
	if err != nil {
		return fmt.Errorf("updating pair day data: %w", err)
	}

	pairHourData, err := s.UpdatePairHourData(ev.LogAddress)
	if err != nil {
		return fmt.Errorf("updating pair hour data: %w", err)
	}

	dayData, err := s.UpdateFactoryDayData()
	if err != nil {
		return fmt.Errorf("update pancake day data: %w", err)
	}

	token0DayData, err := s.UpdateTokenDayData(ev.LogAddress, token0, bundle)
	if err != nil {
		return fmt.Errorf("update token0 day data: %w", err)
	}

	token1DayData, err := s.UpdateTokenDayData(ev.LogAddress, token1, bundle)
	if err != nil {
		return fmt.Errorf("udpate token1 day data: %w", err)
	}

	dayData.VolumeUSD = entity.FloatAdd(dayData.VolumeUSD, F(trackedAmountUSD))
	dayData.VolumeETH = entity.FloatAdd(dayData.VolumeETH, F(trackedAmountETH))
	dayData.UntrackedVolume = entity.FloatAdd(dayData.UntrackedVolume, F(derivedAmountUSD))

	err = s.Save(dayData)
	if err != nil {
		return err
	}

	pairDayData.VolumeToken0 = entity.FloatAdd(pairDayData.VolumeToken0, F(amount0Total))
	pairDayData.VolumeToken1 = entity.FloatAdd(pairDayData.VolumeToken1, F(amount1Total))
	pairDayData.VolumeUSD = entity.FloatAdd(pairDayData.VolumeUSD, F(trackedAmountUSD))
	err = s.Save(pairDayData)
	if err != nil {
		return err
	}

	pairHourData.VolumeToken0 = entity.FloatAdd(pairHourData.VolumeToken0, F(amount0Total))
	pairHourData.VolumeToken1 = entity.FloatAdd(pairHourData.VolumeToken1, F(amount1Total))
	pairHourData.VolumeUSD = entity.FloatAdd(pairHourData.VolumeUSD, F(trackedAmountUSD))
	err = s.Save(pairHourData)
	if err != nil {
		return err
	}

	token0DayData.Volume = entity.FloatAdd(token0DayData.Volume, F(amount0Total))
	token0DayData.VolumeETH = entity.FloatAdd(token0DayData.VolumeETH, F(bf().Mul(amount0Total, token0.DerivedETH.Float())))
	token0DayData.VolumeUSD = entity.FloatAdd(token0DayData.VolumeUSD, F(bf().Mul(bf().Mul(amount0Total, token0.DerivedETH.Float()), bundle.EthPrice.Float())))
	err = s.Save(token0DayData)
	if err != nil {
		return err
	}

	token1DayData.Volume = entity.FloatAdd(token1DayData.Volume, F(amount1Total))
	token1DayData.VolumeETH = entity.FloatAdd(token1DayData.VolumeETH, F(bf().Mul(amount1Total, token1.DerivedETH.Float())))
	token1DayData.VolumeUSD = entity.FloatAdd(token1DayData.VolumeUSD, F(bf().Mul(bf().Mul(amount1Total, token1.DerivedETH.Float()), bundle.EthPrice.Float())))
	err = s.Save(token1DayData)
	if err != nil {
		return err
	}

	return nil
}
