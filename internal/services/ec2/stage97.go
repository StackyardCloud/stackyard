package ec2

type CancelledSpotInstanceRequest struct {
	SpotInstanceRequestID string
	State                 string
}

func (s *Service) CancelSpotInstanceRequests(spotInstanceRequestIDs []string) ([]CancelledSpotInstanceRequest, error) {
	spotInstanceRequestIDs = dedupeTrimmedStrings(spotInstanceRequestIDs)
	if len(spotInstanceRequestIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]CancelledSpotInstanceRequest, 0, len(spotInstanceRequestIDs))
	for _, requestID := range spotInstanceRequestIDs {
		state := "cancelled"
		s.spotInstanceRequestStates[requestID] = state
		out = append(out, CancelledSpotInstanceRequest{
			SpotInstanceRequestID: requestID,
			State:                 state,
		})
	}
	return out, nil
}
