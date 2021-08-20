package exchange

import (
	"github.com/streamingfast/eth-go"
	"github.com/streamingfast/sparkle/entity"
)

func (s *Subgraph) createUser(address eth.Address) (*User, error) {
	factory, err := s.getFactory()
	if err != nil {
		return nil, err
	}

	factory.UserCount = entity.IntAdd(factory.UserCount, IL(1))
	if err := s.Save(factory); err != nil {
		return nil, err
	}

	user := NewUser(address.Pretty())
	if err := s.Save(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *Subgraph) getUser(address eth.Address) (*User, error) {
	user := NewUser(address.Pretty())
	err := s.Load(user)
	if err != nil {
		return nil, err
	}

	if !user.Exists() {
		return s.createUser(address)
	}

	return user, nil
}
