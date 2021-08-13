package exchange

import (
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) HandleFactoryPairCreatedEvent(ev *FactoryPairCreatedEvent) error {
	factory, err := s.getFactory()
	if err != nil {
		return err
	}

	if !factory.Exists() {
		_, err := s.getBundle() // creates bundle if it does not exist
		if err != nil {
			return err
		}
	}

	pair, err := s.getPair(ev.Pair, ev.Token0, ev.Token1)
	if err != nil {
		return err
	}

	err = s.Save(pair)
	if err != nil {
		return err
	}

	err = s.CreatePairTemplateWithTokens(ev.Pair, ev.Token0, ev.Token1)
	if err != nil {
		return err
	}

	factory.PairCount = entity.IntAdd(factory.PairCount, IL(1))
	err = s.Save(factory)
	if err != nil {
		return err
	}

	return nil
}
