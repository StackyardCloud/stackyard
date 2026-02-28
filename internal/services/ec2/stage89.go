package ec2

import "strings"

func (s *Service) CancelCapacityReservation(capacityReservationID string) (bool, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityReservationID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cancelledCapacityReservations[capacityReservationID] = true
	return true, nil
}
