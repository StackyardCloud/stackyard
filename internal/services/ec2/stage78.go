package ec2

func (s *Service) AcceptReservedInstancesExchangeQuote(reservedInstanceIDs []string) (string, error) {
	ids := dedupeTrimmedStrings(reservedInstanceIDs)
	if len(ids) == 0 {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.nextIDLocked("riex"), nil
}
