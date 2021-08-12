package exchange

import (
	"fmt"
	"strconv"

	eth "github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) UpdateFactoryDayData() (*DayData, error) {
	factory := NewFactory(FactoryAddress)
	err := s.Load(factory)
	if err != nil {
		return nil, fmt.Errorf("loading pancake factory: %w", err)
	}

	timestamp := s.Block().Timestamp().Unix()
	dayId := timestamp / 86400
	dayStartTimestamp := dayId * 86400

	dayData := NewDayData(strconv.FormatInt(dayId, 10))
	err = s.Load(dayData)
	if err != nil {
		return nil, err
	}
	if !dayData.Exists() {
		// Already created above, not needed here again.
		dayData = NewDayData(strconv.FormatInt(dayId, 10))
		dayData.Date = dayStartTimestamp
	}

	dayData.LiquidityUSD = factory.LiquidityUSD
	dayData.LiquidityETH = factory.LiquidityETH
	dayData.TxCount = factory.TxCount

	err = s.Save(dayData)
	if err != nil {
		return nil, err
	}

	return dayData, nil
}

func (s *Subgraph) UpdatePairDayData(pairAddress eth.Address) (*PairDayData, error) {
	timestamp := s.Block().Timestamp().Unix()
	dayId := timestamp / 86400
	dayStartTimestamp := dayId * 86400
	dayPairId := fmt.Sprintf("%s-%d", pairAddress.Pretty(), dayId)

	pair := NewPair(pairAddress.Pretty())
	err := s.Load(pair)
	if err != nil {
		return nil, fmt.Errorf("loading pair %s: %w", pairAddress.Pretty(), err)
	}

	pairDayData := NewPairDayData(dayPairId)
	err = s.Load(pairDayData)
	if err != nil {
		return nil, fmt.Errorf("loading pair_day_data %s: %w", dayPairId, err)
	}

	if !pairDayData.Exists() {
		pairDayData = NewPairDayData(dayPairId)
		pairDayData.Date = dayStartTimestamp
		pairDayData.Token0 = pair.Token0
		pairDayData.Token1 = pair.Token1
		pairDayData.Pair = pairAddress.Pretty()
	}

	pairDayData.TotalSupply = pair.TotalSupply
	pairDayData.Reserve0 = pair.Reserve0
	pairDayData.Reserve1 = pair.Reserve1
	pairDayData.ReserveUSD = pair.ReserveUSD
	pairDayData.TxCount = entity.IntAdd(pairDayData.TxCount, IL(1))

	err = s.Save(pairDayData)
	if err != nil {
		return nil, fmt.Errorf("saving pair_day_data: %w", err)
	}

	return pairDayData, nil
}

func (s *Subgraph) UpdatePairHourData(pairAddress eth.Address) (*PairHourData, error) {
	timestamp := s.Block().Timestamp().Unix()
	hourId := timestamp / 3600
	hourStartUnix := hourId * 3600
	hourPairId := fmt.Sprintf("%s-%d", pairAddress.Pretty(), hourId)

	pair := NewPair(pairAddress.Pretty())
	err := s.Load(pair)
	if err != nil {
		return nil, fmt.Errorf("loading pair %s: %w", pairAddress.Pretty(), err)
	}

	pairHourData := NewPairHourData(hourPairId)
	err = s.Load(pairHourData)
	if err != nil {
		return nil, fmt.Errorf("loading pair_day_data %s: %w", hourPairId, err)
	}

	if !pairHourData.Exists() {
		pairHourData = NewPairHourData(hourPairId)
		pairHourData.Date = hourStartUnix
		pairHourData.Pair = pairAddress.Pretty()
	}

	pairHourData.Reserve0 = pair.Reserve0
	pairHourData.Reserve1 = pair.Reserve1
	pairHourData.ReserveUSD = pair.ReserveUSD
	pairHourData.TxCount = entity.IntAdd(pairHourData.TxCount, IL(1))

	err = s.Save(pairHourData)
	if err != nil {
		return nil, fmt.Errorf("saving pair_day_data: %w", err)
	}

	return pairHourData, nil
}

func (s *Subgraph) UpdateTokenDayData(pairAddress eth.Address, token *Token, bundle *Bundle) (*TokenDayData, error) {
	timestamp := s.Block().Timestamp().Unix()
	dayId := timestamp / 86400
	dayStartTimestamp := dayId * 86400
	tokenDayId := fmt.Sprintf("%s-%d", token.ID, dayId)

	tokenDayData := NewTokenDayData(tokenDayId)
	err := s.Load(tokenDayData)
	if err != nil {
		return nil, fmt.Errorf("loading token_day_data")
	}

	if !tokenDayData.Exists() {
		tokenDayData = NewTokenDayData(tokenDayId)
		tokenDayData.Date = dayStartTimestamp
		tokenDayData.Token = token.ID
	}

	tokenDayData.PriceUSD = F(bf().Mul(token.DerivedETH.Float(), bundle.EthPrice.Float()))
	tokenDayData.Liquidity = token.Liquidity
	tokenDayData.LiquidityETH = F(bf().Mul(token.Liquidity.Float(), token.DerivedETH.Float()))
	tokenDayData.LiquidityUSD = F(bf().Mul(tokenDayData.LiquidityETH.Float(), bundle.EthPrice.Float()))
	tokenDayData.TxCount = entity.IntAdd(tokenDayData.TxCount, IL(1))

	err = s.Save(tokenDayData)
	if err != nil {
		return nil, fmt.Errorf("saving token_day_data %s: %w", tokenDayData.ID, err)
	}

	return tokenDayData, nil
}
