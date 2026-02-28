package ec2

import "strings"

type CapacityReservationFleetCancellationState struct {
	CapacityReservationFleetID string
	CurrentFleetState          string
	PreviousFleetState         string
}

type CancelCapacityReservationFleetError struct {
	Code    string
	Message string
}

type FailedCapacityReservationFleetCancellationResult struct {
	CapacityReservationFleetID          string
	CancelCapacityReservationFleetError *CancelCapacityReservationFleetError
}

func (s *Service) CancelCapacityReservationFleets(
	capacityReservationFleetIDs []string,
) ([]CapacityReservationFleetCancellationState, []FailedCapacityReservationFleetCancellationResult, error) {
	capacityReservationFleetIDs = dedupeTrimmedStrings(capacityReservationFleetIDs)
	if len(capacityReservationFleetIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]CapacityReservationFleetCancellationState, 0, len(capacityReservationFleetIDs))
	failed := make([]FailedCapacityReservationFleetCancellationResult, 0)

	for _, fleetID := range capacityReservationFleetIDs {
		if strings.HasPrefix(fleetID, "crf-missing") {
			failed = append(failed, FailedCapacityReservationFleetCancellationResult{
				CapacityReservationFleetID: fleetID,
				CancelCapacityReservationFleetError: &CancelCapacityReservationFleetError{
					Code:    "InvalidCapacityReservationFleetId.NotFound",
					Message: "capacity reservation fleet not found",
				},
			})
			continue
		}

		previousState := s.capacityReservationFleetStates[fleetID]
		if previousState == "" {
			previousState = "active"
		}
		currentState := "cancelled"
		s.capacityReservationFleetStates[fleetID] = currentState

		successful = append(successful, CapacityReservationFleetCancellationState{
			CapacityReservationFleetID: fleetID,
			CurrentFleetState:          currentState,
			PreviousFleetState:         previousState,
		})
	}

	return successful, failed, nil
}
