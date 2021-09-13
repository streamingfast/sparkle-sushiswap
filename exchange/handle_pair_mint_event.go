package exchange

import (
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"

	"go.uber.org/zap"
)

func (s *Subgraph) HandlePairMintEvent(ev *PairMintEvent) error {
	if s.StepBelow(4) {
		return nil
	}

	trx := NewTransaction(ev.Transaction.Hash.Pretty())
	if err := s.Load(trx); err != nil {
		return err
	}

	mint := NewMint(trx.Mints[len(trx.Mints)-1])
	if err := s.Load(mint); err != nil {
		return err
	}
	s.Log.Debug("mint things - mint", zap.String("to", eth.Address(mint.To).Pretty()))

	pairAddress := ev.LogAddress
	pair, err := s.getPair(pairAddress, nil, nil)
	if err != nil {
		return err
	}

	factory := NewFactory(FactoryAddress)
	if err := s.Load(factory); err != nil {
		return err
	}

	token0 := NewToken(pair.Token0)
	if err := s.Load(token0); err != nil {
		return err
	}
	token1 := NewToken(pair.Token1)
	if err := s.Load(token1); err != nil {
		return err
	}

	token0Amount := entity.ConvertTokenToDecimal(ev.Amount0, token0.Decimals.Int().Int64())
	token1Amount := entity.ConvertTokenToDecimal(ev.Amount1, token1.Decimals.Int().Int64())

	token0.TxCount = entity.IntAdd(token0.TxCount, IL(1))
	token1.TxCount = entity.IntAdd(token1.TxCount, IL(1))

	bundle, err := s.getBundle() // creates bundle if it does not exist
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

	// update txn counts
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

	sender := ev.Sender.Pretty()
	mint.Sender = &sender
	mint.Amount0 = F(token0Amount).Ptr()
	mint.Amount1 = F(token1Amount).Ptr()
	mint.LogIndex = IL(int64(ev.LogIndex)).Ptr()
	mint.AmountUSD = F(amountTotalUSD).Ptr()
	if err := s.Save(mint); err != nil {
		return err
	}

	position, err := s.createLiquidityPosition(eth.Address(mint.To), pairAddress)
	if err != nil {
		return err
	}

	err = s.createLiquidityPositionSnapshot(position)
	if err != nil {
		return err
	}

	// // update day entities
	if _, err := s.UpdatePairDayData(ev.LogAddress); err != nil {
		return err
	}

	if _, err := s.UpdatePairHourData(ev.LogAddress); err != nil {
		return err
	}

	if _, err := s.UpdateFactoryDayData(); err != nil {
		return err
	}

	if _, err := s.UpdateTokenDayData(token0); err != nil {
		return err
	}
	if _, err := s.UpdateTokenDayData(token1); err != nil {
		return err
	}

	return nil
}

func (s *Subgraph) isCompleteMint(mintId string) (bool, error) {
	mint := NewMint(mintId)
	err := s.Load(mint)
	if err != nil {
		return false, err
	}
	senderStr := ""
	var completed bool
	if mint.Sender != nil {
		senderStr = *mint.Sender
		completed = true
	}
	s.Log.Debug("checking if mint is completed", zap.String("mint_id", mintId), zap.Bool("completed", completed), zap.String("sender", senderStr))

	//   return MintEvent.load(mintId).sender !== null
	return completed, nil
}
