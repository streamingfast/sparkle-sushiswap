package exchange

func (s *Subgraph) getBundle() (*Bundle, error) {
	bundle := NewBundle("1")
	if err := s.Load(bundle); err != nil {
		return nil, err
	}

	if !bundle.Exists() {
		err := s.Save(bundle)
		if err != nil {
			return nil, err
		}
	}

	return bundle, nil
}
