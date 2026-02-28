package ec2

import "strings"

func (s *Service) AllocateHosts(
	assetIDs []string,
	quantity *int32,
	availabilityZone *string,
	availabilityZoneID *string,
	instanceType *string,
	instanceFamily *string,
	autoPlacement *string,
	hostRecovery *string,
	hostMaintenance *string,
) ([]string, error) {
	if instanceType != nil && strings.TrimSpace(*instanceType) != "" && instanceFamily != nil && strings.TrimSpace(*instanceFamily) != "" {
		return nil, ErrInvalidParameter
	}

	normalizedAssets := dedupeTrimmedStrings(assetIDs)
	hostCount := int32(1)
	if quantity != nil {
		hostCount = *quantity
	} else if len(normalizedAssets) > 0 {
		hostCount = int32(len(normalizedAssets))
	}
	if hostCount <= 0 {
		return nil, ErrInvalidParameter
	}
	if len(normalizedAssets) > 0 && int32(len(normalizedAssets)) != hostCount {
		return nil, ErrInvalidParameter
	}

	az := "us-east-1a"
	if availabilityZone != nil && strings.TrimSpace(*availabilityZone) != "" {
		az = strings.TrimSpace(*availabilityZone)
	} else if availabilityZoneID != nil && strings.TrimSpace(*availabilityZoneID) != "" {
		az = strings.TrimSpace(*availabilityZoneID)
	}

	instType := ""
	if instanceType != nil {
		instType = strings.TrimSpace(*instanceType)
	}
	instFamily := ""
	if instanceFamily != nil {
		instFamily = strings.TrimSpace(*instanceFamily)
	}

	autoPlacementValue := "off"
	if autoPlacement != nil && strings.TrimSpace(*autoPlacement) != "" {
		autoPlacementValue = strings.ToLower(strings.TrimSpace(*autoPlacement))
	}
	if autoPlacementValue != "on" && autoPlacementValue != "off" {
		return nil, ErrInvalidParameter
	}

	hostRecoveryValue := "off"
	if hostRecovery != nil && strings.TrimSpace(*hostRecovery) != "" {
		hostRecoveryValue = strings.ToLower(strings.TrimSpace(*hostRecovery))
	}
	if hostRecoveryValue != "on" && hostRecoveryValue != "off" {
		return nil, ErrInvalidParameter
	}

	hostMaintenanceValue := "off"
	if hostMaintenance != nil && strings.TrimSpace(*hostMaintenance) != "" {
		hostMaintenanceValue = strings.ToLower(strings.TrimSpace(*hostMaintenance))
	}
	if hostMaintenanceValue != "on" && hostMaintenanceValue != "off" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	hostIDs := make([]string, 0, hostCount)
	for i := int32(0); i < hostCount; i++ {
		hostID := s.nextIDLocked("h")
		s.dedicatedHosts[hostID] = &DedicatedHost{
			ID:               hostID,
			AvailabilityZone: az,
			InstanceType:     instType,
			InstanceFamily:   instFamily,
			AutoPlacement:    autoPlacementValue,
			HostRecovery:     hostRecoveryValue,
			HostMaintenance:  hostMaintenanceValue,
		}
		hostIDs = append(hostIDs, hostID)
	}
	return hostIDs, nil
}
