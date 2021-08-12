package exchange

import "strconv"

func (s *Subgraph) getFactory() (*Factory, error) {
	factory := NewFactory(FactoryAddress)
	err := s.Load(factory)
	if err != nil {
		return nil, err
	}

	return factory, nil
}

func (s *Subgraph) getDayData() (*DayData, error) {
	timestamp := s.Block().Timestamp().Unix()
	dayId := timestamp / 86400
	dayStartTimestamp := dayId * 86400

	dayData := NewDayData(strconv.FormatInt(dayId, 10))
	err := s.Load(dayData)
	if err != nil {
		return nil, err
	}

	if !dayData.Exists() {
		factory, err := s.getFactory()
		if err != nil {
			return nil, err
		}

		dayData.Date = dayStartTimestamp
		dayData.LiquidityETH = factory.LiquidityETH
		dayData.LiquidityUSD = factory.LiquidityUSD
		dayData.Factory = factory.ID
		dayData.TxCount = factory.TxCount
	}

	return dayData, nil
}

func (s *Subgraph) updateDayData() error {
	factory, err := s.getFactory()
	if err != nil {
		return err
	}

	dayData, err := s.getDayData()
	if err != nil {
		return err
	}

	dayData.LiquidityETH = factory.LiquidityETH
	dayData.LiquidityUSD = factory.LiquidityUSD
	dayData.TxCount = factory.TxCount

	err = s.Save(dayData)
	if err != nil {
		return err
	}

	return nil
}