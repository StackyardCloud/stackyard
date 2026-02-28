package ec2

import (
	"strings"
	"time"
)

type CapacityReservation struct {
	ID                     string
	AvailabilityZone       string
	AvailabilityZoneID     string
	InstanceType           string
	InstancePlatform       string
	InstanceMatchCriteria  string
	Tenancy                string
	State                  string
	TotalInstanceCount     int32
	AvailableInstanceCount int32
	EbsOptimized           *bool
	EphemeralStorage       *bool
	OwnerID                string
	CreateDate             time.Time
	Tags                   map[string]string
}

func (s *Service) CreateCapacityReservation(
	instanceCount int32,
	instancePlatform string,
	instanceType string,
	availabilityZone string,
	availabilityZoneID string,
	instanceMatchCriteria string,
	tenancy string,
	ebsOptimized *bool,
	ephemeralStorage *bool,
	tags []Tag,
) (CapacityReservation, error) {
	instancePlatform = strings.TrimSpace(instancePlatform)
	instanceType = strings.TrimSpace(instanceType)
	availabilityZone = strings.TrimSpace(availabilityZone)
	availabilityZoneID = strings.TrimSpace(availabilityZoneID)
	instanceMatchCriteria = strings.TrimSpace(instanceMatchCriteria)
	tenancy = strings.TrimSpace(tenancy)

	if instanceCount <= 0 || instancePlatform == "" || instanceType == "" {
		return CapacityReservation{}, ErrInvalidParameter
	}
	if availabilityZone == "" && availabilityZoneID == "" {
		availabilityZone = "us-east-1a"
	}
	if instanceMatchCriteria == "" {
		instanceMatchCriteria = "open"
	}
	if tenancy == "" {
		tenancy = "default"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reservation := &CapacityReservation{
		ID:                     s.nextIDLocked("cr"),
		AvailabilityZone:       availabilityZone,
		AvailabilityZoneID:     availabilityZoneID,
		InstanceType:           instanceType,
		InstancePlatform:       instancePlatform,
		InstanceMatchCriteria:  instanceMatchCriteria,
		Tenancy:                tenancy,
		State:                  "active",
		TotalInstanceCount:     instanceCount,
		AvailableInstanceCount: instanceCount,
		EbsOptimized:           cloneBoolPointer(ebsOptimized),
		EphemeralStorage:       cloneBoolPointer(ephemeralStorage),
		OwnerID:                DefaultAccountID,
		CreateDate:             time.Now().UTC(),
		Tags:                   tagsToMap(tags),
	}

	s.capacityReservations[reservation.ID] = reservation
	return cloneCapacityReservation(reservation), nil
}

func cloneCapacityReservation(in *CapacityReservation) CapacityReservation {
	if in == nil {
		return CapacityReservation{}
	}
	return CapacityReservation{
		ID:                     in.ID,
		AvailabilityZone:       in.AvailabilityZone,
		AvailabilityZoneID:     in.AvailabilityZoneID,
		InstanceType:           in.InstanceType,
		InstancePlatform:       in.InstancePlatform,
		InstanceMatchCriteria:  in.InstanceMatchCriteria,
		Tenancy:                in.Tenancy,
		State:                  in.State,
		TotalInstanceCount:     in.TotalInstanceCount,
		AvailableInstanceCount: in.AvailableInstanceCount,
		EbsOptimized:           cloneBoolPointer(in.EbsOptimized),
		EphemeralStorage:       cloneBoolPointer(in.EphemeralStorage),
		OwnerID:                in.OwnerID,
		CreateDate:             in.CreateDate,
		Tags:                   cloneStringMap(in.Tags),
	}
}

func cloneBoolPointer(in *bool) *bool {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

type CapacityReservationSplitResult struct {
	InstanceCount                  int32
	SourceCapacityReservation      CapacityReservation
	DestinationCapacityReservation CapacityReservation
}

func (s *Service) CreateCapacityReservationBySplitting(
	instanceCount int32,
	sourceCapacityReservationID string,
	tags []Tag,
) (CapacityReservationSplitResult, error) {
	sourceCapacityReservationID = strings.TrimSpace(sourceCapacityReservationID)
	if instanceCount <= 0 || sourceCapacityReservationID == "" {
		return CapacityReservationSplitResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	source := s.capacityReservations[sourceCapacityReservationID]
	if source == nil {
		return CapacityReservationSplitResult{}, ErrNotFound
	}
	if source.AvailableInstanceCount < instanceCount || source.TotalInstanceCount < instanceCount {
		return CapacityReservationSplitResult{}, ErrConflict
	}

	destination := &CapacityReservation{
		ID:                     s.nextIDLocked("cr"),
		AvailabilityZone:       source.AvailabilityZone,
		AvailabilityZoneID:     source.AvailabilityZoneID,
		InstanceType:           source.InstanceType,
		InstancePlatform:       source.InstancePlatform,
		InstanceMatchCriteria:  source.InstanceMatchCriteria,
		Tenancy:                source.Tenancy,
		State:                  source.State,
		TotalInstanceCount:     instanceCount,
		AvailableInstanceCount: instanceCount,
		EbsOptimized:           cloneBoolPointer(source.EbsOptimized),
		EphemeralStorage:       cloneBoolPointer(source.EphemeralStorage),
		OwnerID:                source.OwnerID,
		CreateDate:             time.Now().UTC(),
		Tags:                   tagsToMap(tags),
	}

	source.TotalInstanceCount -= instanceCount
	source.AvailableInstanceCount -= instanceCount

	s.capacityReservations[destination.ID] = destination

	return CapacityReservationSplitResult{
		InstanceCount:                  instanceCount,
		SourceCapacityReservation:      cloneCapacityReservation(source),
		DestinationCapacityReservation: cloneCapacityReservation(destination),
	}, nil
}
