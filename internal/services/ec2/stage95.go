package ec2

import (
	"fmt"
	"strings"
	"time"
)

type ReservedInstancesListing struct {
	ClientToken                string
	CreateDate                 time.Time
	ReservedInstancesID        string
	ReservedInstancesListingID string
	Status                     string
	StatusMessage              string
	UpdateDate                 time.Time
}

func (s *Service) CancelReservedInstancesListing(reservedInstancesListingID string) ([]ReservedInstancesListing, error) {
	reservedInstancesListingID = strings.TrimSpace(reservedInstancesListingID)
	if reservedInstancesListingID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	createDate := s.reservedInstancesListingCreatedAt[reservedInstancesListingID]
	if createDate.IsZero() {
		createDate = now
		s.reservedInstancesListingCreatedAt[reservedInstancesListingID] = createDate
	}
	s.reservedInstancesListingStates[reservedInstancesListingID] = "cancelled"

	listing := ReservedInstancesListing{
		CreateDate:                 createDate,
		ReservedInstancesID:        fmt.Sprintf("ri-%s", strings.TrimPrefix(reservedInstancesListingID, "ril-")),
		ReservedInstancesListingID: reservedInstancesListingID,
		Status:                     "cancelled",
		StatusMessage:              "listing cancelled",
		UpdateDate:                 now,
	}
	return []ReservedInstancesListing{listing}, nil
}
