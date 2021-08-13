package exchange

import (
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) HandlePairBurnEvent(ev *PairBurnEvent) error {
	if s.StepBelow(3) {
		return nil
	}

	transaction := NewTransaction(ev.Transaction.Hash.Pretty())
	if err := s.Load(transaction); err != nil {
		return err
	}

	pair := NewPair(ev.LogAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return err
	}

	// safety check
	if !transaction.Exists() {
		block := s.Block()
		transaction.BlockNumber = IL(int64(block.Number()))
		transaction.Timestamp = IL(block.Timestamp().Unix())
	}

	burn := NewBurn(transaction.Burns[len(transaction.Burns)-1])
	if err := s.Load(burn); err != nil {
		return err
	}

	if !burn.Exists() || bool(burn.Complete) {
		burn.Complete = true
		burn.Pair = pair.ID
		burn.Liquidity = FL(0)
		burn.Transaction = transaction.ID
		burn.Timestamp = transaction.Timestamp
	}

	factory, err := s.getFactory()
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

	token0Amount := entity.ConvertTokenToDecimal(ev.Amount0, token0.Decimals.Int().Int64())
	token1Amount := entity.ConvertTokenToDecimal(ev.Amount1, token1.Decimals.Int().Int64())

	token0.TxCount = entity.IntAdd(token0.TxCount, IL(1))
	token1.TxCount = entity.IntAdd(token1.TxCount, IL(1))

	bundle, err := s.getBundle()
	if err != nil {
		return err
	}

	amountTotalUSD := bf().Mul(
		bf().Add(
			bf().Mul(token1.DerivedETH.Float(), token1Amount),
			bf().Mul(token0.DerivedETH.Float(), token0Amount),
		),
		bundle.EthPrice.Float(),
	)

	pair.TxCount = entity.IntAdd(pair.TxCount, IL(1))
	factory.TxCount = entity.IntAdd(factory.TxCount, IL(1))

	// save entities
	if err := s.Save(token0); err != nil {
		return err
	}
	if err := s.Save(token1); err != nil {
		return err
	}
	if err := s.Save(pair); err != nil {
		return err
	}
	if err := s.Save(factory); err != nil {
		return err
	}

	// burn.Sender = ev.Sender.Bytes()
	burn.Amount0 = F(token0Amount).Ptr()
	burn.Amount1 = F(token1Amount).Ptr()
	burn.LogIndex = IL(int64(ev.LogIndex)).Ptr()
	burn.AmountUSD = F(amountTotalUSD).Ptr()

	if err := s.Save(burn); err != nil {
		return err
	}

	position, err := s.createLiquidityPosition(eth.MustNewAddress(*burn.Sender), ev.LogAddress)
	if err != nil {
		return err
	}
	err = s.createLiquidityPositionSnapshot(position)
	if err != nil {
		return err
	}

	// // update day entities
	if _, err := s.UpdateFactoryDayData(); err != nil {
		return err
	}

	if _, err := s.UpdatePairDayData(ev.LogAddress); err != nil {
		return err
	}

	if _, err := s.UpdatePairHourData(ev.LogAddress); err != nil {
		return err
	}

	if _, err := s.UpdateTokenDayData(ev.LogAddress, token0, bundle); err != nil {
		return err
	}
	if _, err := s.UpdateTokenDayData(ev.LogAddress, token1, bundle); err != nil {
		return err
	}

	return nil
}
