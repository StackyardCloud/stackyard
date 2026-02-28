package ec2

import (
	"strings"
	"time"
)

type ReservationFleetInstanceSpecification struct {
	AvailabilityZone   string
	AvailabilityZoneID string
	EbsOptimized       *bool
	InstancePlatform   string
	InstanceType       string
	Priority           *int32
	Weight             *float64
}

type FleetCapacityReservation struct {
	AvailabilityZone      string
	AvailabilityZoneID    string
	CapacityReservationID string
	CreateDate            time.Time
	EbsOptimized          *bool
	FulfilledCapacity     float64
	InstancePlatform      string
	InstanceType          string
	Priority              *int32
	TotalInstanceCount    int32
	Weight                *float64
}

type CapacityReservationFleet struct {
	AllocationStrategy        string
	ID                        string
	CreateTime                time.Time
	EndDate                   *time.Time
	FleetCapacityReservations []FleetCapacityReservation
	InstanceMatchCriteria     string
	State                     string
	Tags                      map[string]string
	Tenancy                   string
	TotalFulfilledCapacity    float64
	TotalTargetCapacity       int32
}

func (s *Service) CreateCapacityReservationFleet(
	specs []ReservationFleetInstanceSpecification,
	totalTargetCapacity int32,
	allocationStrategy *string,
	endDate *time.Time,
	instanceMatchCriteria *string,
	tenancy *string,
	tags []Tag,
) (CapacityReservationFleet, error) {
	if len(specs) == 0 || totalTargetCapacity <= 0 {
		return CapacityReservationFleet{}, ErrInvalidParameter
	}

	normalizedSpecs := make([]ReservationFleetInstanceSpecification, 0, len(specs))
	for _, spec := range specs {
		spec.InstancePlatform = strings.TrimSpace(spec.InstancePlatform)
		spec.InstanceType = strings.TrimSpace(spec.InstanceType)
		spec.AvailabilityZone = strings.TrimSpace(spec.AvailabilityZone)
		spec.AvailabilityZoneID = strings.TrimSpace(spec.AvailabilityZoneID)
		if spec.InstancePlatform == "" || spec.InstanceType == "" {
			return CapacityReservationFleet{}, ErrInvalidParameter
		}
		if spec.AvailabilityZone == "" && spec.AvailabilityZoneID == "" {
			spec.AvailabilityZone = "us-east-1a"
		}
		normalizedSpecs = append(normalizedSpecs, spec)
	}

	fleetAllocationStrategy := "prioritized"
	if allocationStrategy != nil && strings.TrimSpace(*allocationStrategy) != "" {
		fleetAllocationStrategy = strings.TrimSpace(*allocationStrategy)
	}
	fleetInstanceMatchCriteria := "open"
	if instanceMatchCriteria != nil && strings.TrimSpace(*instanceMatchCriteria) != "" {
		fleetInstanceMatchCriteria = strings.TrimSpace(*instanceMatchCriteria)
	}
	fleetTenancy := "default"
	if tenancy != nil && strings.TrimSpace(*tenancy) != "" {
		fleetTenancy = strings.TrimSpace(*tenancy)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	fleetID := s.nextIDLocked("crf")
	base := totalTargetCapacity / int32(len(normalizedSpecs))
	remainder := totalTargetCapacity % int32(len(normalizedSpecs))

	capacityReservations := make([]FleetCapacityReservation, 0, len(normalizedSpecs))
	for i, spec := range normalizedSpecs {
		count := base
		if int32(i) < remainder {
			count++
		}
		if count <= 0 {
			count = 1
		}

		capacityReservationID := s.nextIDLocked("cr")
		reservation := &CapacityReservation{
			ID:                     capacityReservationID,
			AvailabilityZone:       spec.AvailabilityZone,
			AvailabilityZoneID:     spec.AvailabilityZoneID,
			InstanceType:           spec.InstanceType,
			InstancePlatform:       spec.InstancePlatform,
			InstanceMatchCriteria:  fleetInstanceMatchCriteria,
			Tenancy:                fleetTenancy,
			State:                  "active",
			TotalInstanceCount:     count,
			AvailableInstanceCount: count,
			EbsOptimized:           cloneBoolPointer(spec.EbsOptimized),
			OwnerID:                DefaultAccountID,
			CreateDate:             now,
			Tags:                   tagsToMap(tags),
		}
		s.capacityReservations[capacityReservationID] = reservation

		capacityReservations = append(capacityReservations, FleetCapacityReservation{
			AvailabilityZone:      spec.AvailabilityZone,
			AvailabilityZoneID:    spec.AvailabilityZoneID,
			CapacityReservationID: capacityReservationID,
			CreateDate:            now,
			EbsOptimized:          cloneBoolPointer(spec.EbsOptimized),
			FulfilledCapacity:     float64(count),
			InstancePlatform:      spec.InstancePlatform,
			InstanceType:          spec.InstanceType,
			Priority:              cloneInt32Pointer(spec.Priority),
			TotalInstanceCount:    count,
			Weight:                cloneFloat64Pointer(spec.Weight),
		})
	}

	fleet := &CapacityReservationFleet{
		AllocationStrategy:        fleetAllocationStrategy,
		ID:                        fleetID,
		CreateTime:                now,
		EndDate:                   cloneTimePointer(endDate),
		FleetCapacityReservations: cloneFleetCapacityReservations(capacityReservations),
		InstanceMatchCriteria:     fleetInstanceMatchCriteria,
		State:                     "active",
		Tags:                      tagsToMap(tags),
		Tenancy:                   fleetTenancy,
		TotalFulfilledCapacity:    float64(totalTargetCapacity),
		TotalTargetCapacity:       totalTargetCapacity,
	}
	s.capacityReservationFleets[fleetID] = fleet
	s.capacityReservationFleetStates[fleetID] = fleet.State

	return cloneCapacityReservationFleet(fleet), nil
}

func cloneFloat64Pointer(in *float64) *float64 {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}

func cloneFleetCapacityReservations(in []FleetCapacityReservation) []FleetCapacityReservation {
	out := make([]FleetCapacityReservation, 0, len(in))
	for _, item := range in {
		out = append(out, FleetCapacityReservation{
			AvailabilityZone:      item.AvailabilityZone,
			AvailabilityZoneID:    item.AvailabilityZoneID,
			CapacityReservationID: item.CapacityReservationID,
			CreateDate:            item.CreateDate,
			EbsOptimized:          cloneBoolPointer(item.EbsOptimized),
			FulfilledCapacity:     item.FulfilledCapacity,
			InstancePlatform:      item.InstancePlatform,
			InstanceType:          item.InstanceType,
			Priority:              cloneInt32Pointer(item.Priority),
			TotalInstanceCount:    item.TotalInstanceCount,
			Weight:                cloneFloat64Pointer(item.Weight),
		})
	}
	return out
}

func cloneCapacityReservationFleet(in *CapacityReservationFleet) CapacityReservationFleet {
	if in == nil {
		return CapacityReservationFleet{}
	}
	return CapacityReservationFleet{
		AllocationStrategy:        in.AllocationStrategy,
		ID:                        in.ID,
		CreateTime:                in.CreateTime,
		EndDate:                   cloneTimePointer(in.EndDate),
		FleetCapacityReservations: cloneFleetCapacityReservations(in.FleetCapacityReservations),
		InstanceMatchCriteria:     in.InstanceMatchCriteria,
		State:                     in.State,
		Tags:                      cloneStringMap(in.Tags),
		Tenancy:                   in.Tenancy,
		TotalFulfilledCapacity:    in.TotalFulfilledCapacity,
		TotalTargetCapacity:       in.TotalTargetCapacity,
	}
}
