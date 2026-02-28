package ec2

import "strings"

func (s *Service) AssociateCapacityReservationBillingOwner(capacityReservationID, billingOwnerID string) (bool, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	billingOwnerID = strings.TrimSpace(billingOwnerID)
	if capacityReservationID == "" || billingOwnerID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.capacityReservationBillingOwners[capacityReservationID] = billingOwnerID
	return true, nil
}
