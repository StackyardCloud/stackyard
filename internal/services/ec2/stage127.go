package ec2

import (
	"sort"
	"strings"
	"time"
)

type SnapshotRecycleBinInfo struct {
	Description         string
	RecycleBinEnterTime *time.Time
	RecycleBinExitTime  *time.Time
	SnapshotID          string
}

type InstanceCapacityReservationSpecification struct {
	CapacityReservationPreference       string
	CapacityReservationTargetID         string
	CapacityReservationResourceGroupARN string
}

func (s *Service) ListSnapshotsInRecycleBin(snapshotIDs []string, maxResults *int32, nextToken *string) ([]SnapshotRecycleBinInfo, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(snapshotIDs)
	requestedIDSet := toStringSet(requestedIDs)

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.snapshots))
		for snapshotID := range s.snapshots {
			candidateIDs = append(candidateIDs, snapshotID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]SnapshotRecycleBinInfo, 0, len(candidateIDs))
	for _, snapshotID := range candidateIDs {
		snapshot := s.snapshots[snapshotID]
		if snapshot == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[snapshotID]; !ok {
				continue
			}
		}

		enter := snapshot.StartTime.UTC()
		if enter.IsZero() {
			enter = time.Now().UTC()
		}
		exit := enter.Add(24 * time.Hour)
		items = append(items, SnapshotRecycleBinInfo{
			Description:         snapshot.Description,
			RecycleBinEnterTime: cloneTimePointer(&enter),
			RecycleBinExitTime:  cloneTimePointer(&exit),
			SnapshotID:          snapshotID,
		})
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]SnapshotRecycleBinInfo(nil), items[start:end]...), outputToken, nil
}

func (s *Service) LockSnapshot(snapshotID, lockMode string, lockDuration, coolOffPeriod *int32, expirationDate *time.Time) (LockedSnapshotInfo, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	lockMode = strings.ToLower(strings.TrimSpace(lockMode))
	if snapshotID == "" || (lockMode != "compliance" && lockMode != "governance") {
		return LockedSnapshotInfo{}, ErrInvalidParameter
	}
	if lockDuration != nil && *lockDuration < 0 {
		return LockedSnapshotInfo{}, ErrInvalidParameter
	}
	if coolOffPeriod != nil && *coolOffPeriod < 0 {
		return LockedSnapshotInfo{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshots[snapshotID]
	if snapshot == nil {
		return LockedSnapshotInfo{}, ErrNotFound
	}

	createdOn := time.Now().UTC()
	if !snapshot.StartTime.IsZero() {
		createdOn = snapshot.StartTime.UTC()
	}

	lockDurationValue := int32(24)
	if lockDuration != nil {
		lockDurationValue = *lockDuration
	}
	coolOffValue := int32(0)
	if coolOffPeriod != nil {
		coolOffValue = *coolOffPeriod
	}

	lockExpiresOn := createdOn.Add(time.Duration(lockDurationValue) * time.Hour)
	if expirationDate != nil && !expirationDate.IsZero() {
		if expirationDate.Before(createdOn) {
			return LockedSnapshotInfo{}, ErrInvalidParameter
		}
		lockExpiresOn = expirationDate.UTC()
	}
	coolOffExpiresOn := lockExpiresOn.Add(time.Duration(coolOffValue) * time.Hour)

	locked := &LockedSnapshotInfo{
		CoolOffPeriod:          cloneInt32Pointer(&coolOffValue),
		CoolOffPeriodExpiresOn: cloneTimePointer(&coolOffExpiresOn),
		LockCreatedOn:          cloneTimePointer(&createdOn),
		LockDuration:           cloneInt32Pointer(&lockDurationValue),
		LockDurationStartTime:  cloneTimePointer(&createdOn),
		LockExpiresOn:          cloneTimePointer(&lockExpiresOn),
		LockState:              lockMode,
		OwnerID:                DefaultAccountID,
		SnapshotID:             snapshotID,
	}
	s.lockedSnapshots[snapshotID] = locked
	return cloneStage119LockedSnapshotInfo(*locked), nil
}

func (s *Service) ModifyAvailabilityZoneGroup(groupName, optInStatus string) (bool, error) {
	groupName = strings.TrimSpace(groupName)
	optInStatus = strings.ToLower(strings.TrimSpace(optInStatus))
	if groupName == "" {
		return false, ErrInvalidParameter
	}
	switch optInStatus {
	case "opted-in", "opted-out":
	default:
		return false, ErrInvalidParameter
	}
	return true, nil
}

func (s *Service) ModifyCapacityReservation(
	capacityReservationID string,
	accept *bool,
	additionalInfo *string,
	endDate *time.Time,
	endDateType string,
	instanceCount *int32,
	instanceMatchCriteria string,
) (bool, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	endDateType = strings.ToLower(strings.TrimSpace(endDateType))
	instanceMatchCriteria = strings.ToLower(strings.TrimSpace(instanceMatchCriteria))
	if capacityReservationID == "" {
		return false, ErrInvalidParameter
	}
	if endDateType != "" && endDateType != "limited" && endDateType != "unlimited" {
		return false, ErrInvalidParameter
	}
	if instanceMatchCriteria != "" && instanceMatchCriteria != "open" && instanceMatchCriteria != "targeted" {
		return false, ErrInvalidParameter
	}
	if instanceCount != nil && *instanceCount <= 0 {
		return false, ErrInvalidParameter
	}

	_ = accept
	_ = additionalInfo
	_ = endDate

	s.mu.Lock()
	defer s.mu.Unlock()

	reservation := s.capacityReservations[capacityReservationID]
	if reservation == nil {
		return false, ErrNotFound
	}

	if instanceCount != nil {
		usedInstances := reservation.TotalInstanceCount - reservation.AvailableInstanceCount
		if usedInstances < 0 {
			usedInstances = 0
		}
		reservation.TotalInstanceCount = *instanceCount
		reservation.AvailableInstanceCount = *instanceCount - usedInstances
		if reservation.AvailableInstanceCount < 0 {
			reservation.AvailableInstanceCount = 0
		}
	}
	if instanceMatchCriteria != "" {
		reservation.InstanceMatchCriteria = instanceMatchCriteria
	}

	return true, nil
}

func (s *Service) ModifyCapacityReservationFleet(capacityReservationFleetID string, endDate *time.Time, removeEndDate *bool, totalTargetCapacity *int32) (bool, error) {
	capacityReservationFleetID = strings.TrimSpace(capacityReservationFleetID)
	if capacityReservationFleetID == "" {
		return false, ErrInvalidParameter
	}
	if totalTargetCapacity != nil && *totalTargetCapacity < 0 {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fleet := s.capacityReservationFleets[capacityReservationFleetID]
	if fleet == nil {
		return false, ErrNotFound
	}

	if removeEndDate != nil && *removeEndDate {
		fleet.EndDate = nil
	} else if endDate != nil {
		fleet.EndDate = cloneTimePointer(endDate)
	}

	if totalTargetCapacity != nil {
		fleet.TotalTargetCapacity = *totalTargetCapacity
		fleet.TotalFulfilledCapacity = float64(*totalTargetCapacity)

		if n := len(fleet.FleetCapacityReservations); n > 0 {
			base := int32(0)
			remainder := int32(0)
			if n > 0 {
				base = *totalTargetCapacity / int32(n)
				remainder = *totalTargetCapacity % int32(n)
			}
			for i := range fleet.FleetCapacityReservations {
				count := base
				if int32(i) < remainder {
					count++
				}
				fleet.FleetCapacityReservations[i].TotalInstanceCount = count
				fleet.FleetCapacityReservations[i].FulfilledCapacity = float64(count)
				if reservation := s.capacityReservations[fleet.FleetCapacityReservations[i].CapacityReservationID]; reservation != nil {
					reservation.TotalInstanceCount = count
					if reservation.AvailableInstanceCount > count {
						reservation.AvailableInstanceCount = count
					}
				}
			}
		}
	}

	return true, nil
}

func (s *Service) ModifyDefaultCreditSpecification(instanceFamily, cpuCredits string) (InstanceFamilyCreditSpecification, error) {
	instanceFamily = strings.ToLower(strings.TrimSpace(instanceFamily))
	cpuCredits = strings.ToLower(strings.TrimSpace(cpuCredits))
	if instanceFamily == "" {
		return InstanceFamilyCreditSpecification{}, ErrInvalidParameter
	}
	switch cpuCredits {
	case "standard", "unlimited":
	default:
		return InstanceFamilyCreditSpecification{}, ErrInvalidParameter
	}

	s.mu.Lock()
	s.defaultCreditSpecifications[instanceFamily] = cpuCredits
	s.mu.Unlock()

	return InstanceFamilyCreditSpecification{
		CpuCredits:     cpuCredits,
		InstanceFamily: instanceFamily,
	}, nil
}

func (s *Service) ModifyFleet(
	fleetID string,
	launchTemplateConfigsPresent bool,
	totalTargetCapacity *int32,
	onDemandTargetCapacity *int32,
	spotTargetCapacity *int32,
	context *string,
	excessCapacityTerminationPolicy string,
	instanceType string,
) (bool, error) {
	fleetID = strings.TrimSpace(fleetID)
	instanceType = strings.TrimSpace(instanceType)
	excessCapacityTerminationPolicy = strings.ToLower(strings.TrimSpace(excessCapacityTerminationPolicy))
	if fleetID == "" {
		return false, ErrInvalidParameter
	}
	if totalTargetCapacity != nil && *totalTargetCapacity < 0 {
		return false, ErrInvalidParameter
	}
	if onDemandTargetCapacity != nil && *onDemandTargetCapacity < 0 {
		return false, ErrInvalidParameter
	}
	if spotTargetCapacity != nil && *spotTargetCapacity < 0 {
		return false, ErrInvalidParameter
	}
	if excessCapacityTerminationPolicy != "" && excessCapacityTerminationPolicy != "termination" && excessCapacityTerminationPolicy != "no-termination" {
		return false, ErrInvalidParameter
	}

	_ = launchTemplateConfigsPresent
	_ = context

	targetCapacity := totalTargetCapacity
	if targetCapacity == nil {
		var computed int32
		hasComputed := false
		if onDemandTargetCapacity != nil {
			computed += *onDemandTargetCapacity
			hasComputed = true
		}
		if spotTargetCapacity != nil {
			computed += *spotTargetCapacity
			hasComputed = true
		}
		if hasComputed {
			targetCapacity = &computed
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	fleet := s.fleets[fleetID]
	if fleet == nil {
		return false, ErrNotFound
	}

	existingType := "m5.large"
	lifecycle := "on-demand"
	currentInstanceIDs := make([]string, 0)
	for _, group := range fleet.Instances {
		if strings.TrimSpace(group.InstanceType) != "" && existingType == "m5.large" {
			existingType = strings.TrimSpace(group.InstanceType)
		}
		if strings.TrimSpace(group.Lifecycle) != "" && lifecycle == "on-demand" {
			lifecycle = strings.TrimSpace(group.Lifecycle)
		}
		currentInstanceIDs = append(currentInstanceIDs, group.InstanceIDs...)
	}

	if targetCapacity != nil {
		desired := int(*targetCapacity)
		if desired < 0 {
			return false, ErrInvalidParameter
		}
		switch {
		case desired < len(currentInstanceIDs):
			currentInstanceIDs = currentInstanceIDs[:desired]
		case desired > len(currentInstanceIDs):
			for len(currentInstanceIDs) < desired {
				currentInstanceIDs = append(currentInstanceIDs, s.nextIDLocked("i"))
			}
		}
	}

	if instanceType != "" {
		existingType = instanceType
	}

	fleet.Instances = []FleetInstance{{
		InstanceIDs:  append([]string(nil), currentInstanceIDs...),
		InstanceType: existingType,
		Lifecycle:    lifecycle,
	}}

	return true, nil
}

func (s *Service) ModifyFpgaImageAttribute(
	fpgaImageID, attribute string,
	description *string,
	name *string,
	operationType string,
	productCodes []string,
	userGroups []string,
	userIDs []string,
) (FpgaImageAttributeView, error) {
	fpgaImageID = strings.TrimSpace(fpgaImageID)
	attribute = strings.ToLower(strings.TrimSpace(attribute))
	operationType = strings.ToLower(strings.TrimSpace(operationType))
	if fpgaImageID == "" {
		return FpgaImageAttributeView{}, ErrInvalidParameter
	}
	if operationType != "" && operationType != "add" && operationType != "remove" {
		return FpgaImageAttributeView{}, ErrInvalidParameter
	}
	switch attribute {
	case "", "description", "name", "loadpermission", "productcodes":
	default:
		return FpgaImageAttributeView{}, ErrInvalidParameter
	}

	productCodes = dedupeTrimmedStrings(productCodes)
	userGroups = dedupeTrimmedStrings(userGroups)
	userIDs = dedupeTrimmedStrings(userIDs)
	_ = userGroups

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.fpgaImages[fpgaImageID]
	if image == nil {
		return FpgaImageAttributeView{}, ErrNotFound
	}
	if description != nil {
		image.Description = strings.TrimSpace(*description)
	}
	if name != nil {
		image.Name = strings.TrimSpace(*name)
	}

	view := FpgaImageAttributeView{
		Description: image.Description,
		FpgaImageID: image.FpgaImageID,
		Name:        image.Name,
	}
	if operationType != "remove" {
		view.LoadPermissionUserIDs = append([]string(nil), userIDs...)
		view.ProductCodes = append([]string(nil), productCodes...)
	}

	return view, nil
}

func (s *Service) ModifyHosts(
	hostIDs []string,
	autoPlacement, hostMaintenance, hostRecovery, instanceFamily, instanceType *string,
) ([]string, []UnsuccessfulItem, error) {
	hostIDs = dedupeTrimmedStrings(hostIDs)
	if len(hostIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	autoPlacementValue, ok := stage127OptionalOnOff(autoPlacement)
	if !ok {
		return nil, nil, ErrInvalidParameter
	}
	hostMaintenanceValue, ok := stage127OptionalOnOff(hostMaintenance)
	if !ok {
		return nil, nil, ErrInvalidParameter
	}
	hostRecoveryValue, ok := stage127OptionalOnOff(hostRecovery)
	if !ok {
		return nil, nil, ErrInvalidParameter
	}

	trimmedInstanceType := strings.TrimSpace(derefString(instanceType))
	trimmedInstanceFamily := strings.TrimSpace(derefString(instanceFamily))
	if trimmedInstanceType != "" && trimmedInstanceFamily != "" {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]string, 0, len(hostIDs))
	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, hostID := range hostIDs {
		host := s.dedicatedHosts[hostID]
		if host == nil {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: hostID,
				Code:       "InvalidHostID.NotFound",
				Message:    "host not found",
			})
			continue
		}
		if autoPlacementValue != "" {
			host.AutoPlacement = autoPlacementValue
		}
		if hostMaintenanceValue != "" {
			host.HostMaintenance = hostMaintenanceValue
		}
		if hostRecoveryValue != "" {
			host.HostRecovery = hostRecoveryValue
		}
		if trimmedInstanceType != "" {
			host.InstanceType = trimmedInstanceType
			if trimmedInstanceFamily == "" {
				host.InstanceFamily = stage127HostFamilyFromType(trimmedInstanceType)
			}
		}
		if trimmedInstanceFamily != "" {
			host.InstanceFamily = trimmedInstanceFamily
			if strings.TrimSpace(host.InstanceType) == "" {
				host.InstanceType = trimmedInstanceFamily + ".large"
			}
		}
		successful = append(successful, hostID)
	}

	return successful, unsuccessful, nil
}

func (s *Service) ModifyInstanceCapacityReservationAttributes(
	instanceID, preference, targetID, targetResourceGroupARN string,
) (bool, error) {
	instanceID = strings.TrimSpace(instanceID)
	preference = strings.ToLower(strings.TrimSpace(preference))
	targetID = strings.TrimSpace(targetID)
	targetResourceGroupARN = strings.TrimSpace(targetResourceGroupARN)
	if instanceID == "" {
		return false, ErrInvalidParameter
	}
	if preference != "" && preference != "open" && preference != "none" {
		return false, ErrInvalidParameter
	}
	if preference == "" && targetID == "" && targetResourceGroupARN == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return false, ErrNotFound
	}
	s.instanceCapacityReservationSpecifications[instanceID] = InstanceCapacityReservationSpecification{
		CapacityReservationPreference:       preference,
		CapacityReservationTargetID:         targetID,
		CapacityReservationResourceGroupARN: targetResourceGroupARN,
	}
	return true, nil
}

func stage127OptionalOnOff(value *string) (string, bool) {
	if value == nil {
		return "", true
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if trimmed == "" {
		return "", true
	}
	if trimmed != "on" && trimmed != "off" {
		return "", false
	}
	return trimmed, true
}

func stage127HostFamilyFromType(instanceType string) string {
	instanceType = strings.TrimSpace(instanceType)
	if dot := strings.IndexByte(instanceType, '.'); dot > 0 {
		return instanceType[:dot]
	}
	return ""
}
