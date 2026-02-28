package ec2

import "strings"

func (s *Service) AcceptCapacityReservationBillingOwnership(capacityReservationID string) (bool, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityReservationID == "" {
		return false, ErrInvalidParameter
	}
	return true, nil
}
