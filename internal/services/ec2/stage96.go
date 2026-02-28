package ec2

import "strings"

type CancelSpotFleetRequestSuccess struct {
	SpotFleetRequestID            string
	CurrentSpotFleetRequestState  string
	PreviousSpotFleetRequestState string
}

type CancelSpotFleetRequestError struct {
	Code    string
	Message string
}

type CancelSpotFleetRequestErrorItem struct {
	SpotFleetRequestID string
	Error              *CancelSpotFleetRequestError
}

func (s *Service) CancelSpotFleetRequests(
	spotFleetRequestIDs []string,
	terminateInstances bool,
) ([]CancelSpotFleetRequestSuccess, []CancelSpotFleetRequestErrorItem, error) {
	spotFleetRequestIDs = dedupeTrimmedStrings(spotFleetRequestIDs)
	if len(spotFleetRequestIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]CancelSpotFleetRequestSuccess, 0, len(spotFleetRequestIDs))
	unsuccessful := make([]CancelSpotFleetRequestErrorItem, 0)
	for _, requestID := range spotFleetRequestIDs {
		if strings.HasPrefix(requestID, "sfr-missing") {
			unsuccessful = append(unsuccessful, CancelSpotFleetRequestErrorItem{
				SpotFleetRequestID: requestID,
				Error: &CancelSpotFleetRequestError{
					Code:    "fleetRequestIdDoesNotExist",
					Message: "spot fleet request does not exist",
				},
			})
			continue
		}

		previousState := s.spotFleetRequestStates[requestID]
		if previousState == "" {
			previousState = "active"
		}
		currentState := "cancelled_running"
		if terminateInstances {
			currentState = "cancelled_terminating"
		}
		s.spotFleetRequestStates[requestID] = currentState

		successful = append(successful, CancelSpotFleetRequestSuccess{
			SpotFleetRequestID:            requestID,
			CurrentSpotFleetRequestState:  currentState,
			PreviousSpotFleetRequestState: previousState,
		})
	}

	return successful, unsuccessful, nil
}
