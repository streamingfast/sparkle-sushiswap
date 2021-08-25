package exchange

import (
	"fmt"
	"github.com/streamingfast/sparkle/subgraph"
	"math/big"

	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) getPair(pairAddress, token0Address, token1Address eth.Address) (*Pair, error) {
	pair := NewPair(pairAddress.Pretty())
	if err := s.Load(pair); err != nil {
		return nil, err
	}

	if pair.Exists() {
		return pair, nil
	}

	if token0Address == nil && token1Address == nil {
		return pair, nil
	}

	token0, err := s.getToken(token0Address)
	if err != nil {
		return nil, err
	}

	token1, err := s.getToken(token1Address)
	if err != nil {
		return nil, err
	}

	if isWhitelistedAddress(token0.ID) {
		token1.WhitelistPairs = append(token1.WhitelistPairs, pairAddress.Pretty())
	}

	if isWhitelistedAddress(token1.ID) {
		token0.WhitelistPairs = append(token0.WhitelistPairs, pairAddress.Pretty())
	}

	if err := s.Save(token0); err != nil {
		return nil, err
	}

	if err := s.Save(token1); err != nil {
		return nil, err
	}

	pair.Token0 = token0.ID
	pair.Token1 = token1.ID
	pair.Factory = FactoryAddress
	pair.Block = entity.NewIntFromLiteralUnsigned(s.Block().Number())
	pair.Timestamp = entity.NewIntFromLiteral(s.Block().Timestamp().Unix())
	pair.Name = fmt.Sprintf("%s-%s", token0.Symbol, token1.Symbol)

	return pair, nil
}

func (s *Subgraph) getToken(tokenAddress eth.Address) (*Token, error) {
	if tokenAddress == nil {
		return nil, nil
	}

	token := NewToken(tokenAddress.Pretty())
	err := s.Load(token)
	if err != nil {
		return nil, err
	}

	if token.Exists() {
		return token, nil
	}

	factory, err := s.getFactory()
	if err != nil {
		return nil, err
	}

	factory.TokenCount = entity.IntAdd(factory.TokenCount, IL(1))
	err = s.Save(factory)
	if err != nil {
		return nil, err
	}

	calls := []*subgraph.RPCCall{
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "decimals() (uint256)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "name() (string)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "symbol() (string)",
		},
		{
			ToAddr:          tokenAddress.Pretty(),
			MethodSignature: "totalSupply() (uint256)",
		},
	}

	resps, err := s.RPC(calls)
	if err != nil {
		return nil, fmt.Errorf("rpc call error: %w", err)
	}

	decimalsResponse := resps[0]
	if decimalsResponse.CallError == nil && decimalsResponse.DecodingError == nil {
		token.Decimals = IL(decimalsResponse.Decoded[0].(*big.Int).Int64())
	}

	nameResponse := resps[1]
	if nameResponse.CallError == nil && nameResponse.DecodingError == nil {
		token.Name = nameResponse.Decoded[0].(string)
	} else {
		token.Name = "unknown"
	}

	symbolResponse := resps[2]
	if symbolResponse.CallError == nil && symbolResponse.DecodingError == nil {
		token.Symbol = symbolResponse.Decoded[0].(string)
	} else {
		token.Symbol = "unknown"
	}

	totalSupplyResponse := resps[3]
	if totalSupplyResponse.CallError == nil && totalSupplyResponse.DecodingError == nil {
		token.TotalSupply = IL(totalSupplyResponse.Decoded[0].(*big.Int).Int64())
	}

	token.Factory = factory.ID
	token.DerivedETH = FL(0)
	token.WhitelistPairs = []string{}

	if err := s.Save(token); err != nil {
		return nil, fmt.Errorf("saving token: %w", err)
	}

	return token, nil
}
