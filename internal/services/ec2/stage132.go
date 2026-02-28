package ec2

import (
	"strconv"
	"strings"
	"time"
)

func (s *Service) RejectCapacityReservationBillingOwnership(capacityReservationID string) (bool, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityReservationID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	delete(s.capacityReservationBillingOwners, capacityReservationID)
	s.mu.Unlock()

	return true, nil
}

func (s *Service) ReleaseHosts(hostIDs []string) ([]string, []UnsuccessfulItem, error) {
	hostIDs = dedupeTrimmedStrings(hostIDs)
	if len(hostIDs) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]string, 0, len(hostIDs))
	unsuccessful := make([]UnsuccessfulItem, 0)
	for _, hostID := range hostIDs {
		if s.dedicatedHosts[hostID] == nil {
			unsuccessful = append(unsuccessful, UnsuccessfulItem{
				ResourceID: hostID,
				Code:       "InvalidHostID.NotFound",
				Message:    "host not found",
			})
			continue
		}
		delete(s.dedicatedHosts, hostID)
		successful = append(successful, hostID)
	}

	return successful, unsuccessful, nil
}

func (s *Service) ReleaseIpamPoolAllocation(ipamPoolID, cidr, ipamPoolAllocationID string) (bool, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	cidr = strings.TrimSpace(cidr)
	ipamPoolAllocationID = strings.TrimSpace(ipamPoolAllocationID)
	if ipamPoolID == "" || cidr == "" || ipamPoolAllocationID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	allocation := s.ipamPoolAllocations[ipamPoolAllocationID]
	if allocation == nil {
		return false, ErrNotFound
	}
	if strings.TrimSpace(allocation.ResourceID) != ipamPoolID || strings.TrimSpace(allocation.Cidr) != cidr {
		return false, ErrNotFound
	}

	delete(s.ipamPoolAllocations, ipamPoolAllocationID)
	return true, nil
}

func (s *Service) ReportInstanceStatus(
	instanceIDs []string,
	status string,
	reasonCodes []string,
	startTime *time.Time,
	endTime *time.Time,
	description *string,
) error {
	instanceIDs = dedupeTrimmedStrings(instanceIDs)
	status = strings.ToLower(strings.TrimSpace(status))
	reasonCodes = dedupeTrimmedStrings(reasonCodes)
	if len(instanceIDs) == 0 || len(reasonCodes) == 0 || !stage132ValidReportStatus(status) {
		return ErrInvalidParameter
	}
	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, instanceID := range instanceIDs {
		if s.instances[instanceID] == nil {
			return ErrNotFound
		}
	}

	eventDescription := strings.TrimSpace(derefString(description))
	if eventDescription == "" {
		eventDescription = "instance status reported"
	}

	for _, instanceID := range instanceIDs {
		events := s.instanceStatusEvents[instanceID]
		if events == nil {
			events = map[string]ModifiedInstanceEvent{}
			s.instanceStatusEvents[instanceID] = events
		}
		for _, reasonCode := range reasonCodes {
			eventID := s.nextIDLocked("ev")
			events[eventID] = ModifiedInstanceEvent{
				Code:              reasonCode,
				Description:       eventDescription,
				InstanceEventID:   eventID,
				NotBefore:         cloneTimePointer(startTime),
				NotAfter:          cloneTimePointer(endTime),
				NotBeforeDeadline: cloneTimePointer(endTime),
			}
		}
	}
	return nil
}

func (s *Service) RequestSpotFleet(iamFleetRole string, targetCapacity int32, clientToken *string) (string, error) {
	iamFleetRole = strings.TrimSpace(iamFleetRole)
	if iamFleetRole == "" || targetCapacity <= 0 {
		return "", ErrInvalidParameter
	}
	_ = strings.TrimSpace(derefString(clientToken))

	s.mu.Lock()
	defer s.mu.Unlock()

	spotFleetRequestID := s.nextIDLocked("sfr")
	s.spotFleetRequestStates[spotFleetRequestID] = "active"
	return spotFleetRequestID, nil
}

func (s *Service) RequestSpotInstances(
	instanceCount *int32,
	spotPrice *string,
	requestType string,
	launchGroup *string,
	availabilityZoneGroup *string,
	clientToken *string,
) ([]SpotInstanceRequest, error) {
	requestType = strings.ToLower(strings.TrimSpace(requestType))
	if requestType == "" {
		requestType = "one-time"
	}
	if requestType != "one-time" && requestType != "persistent" {
		return nil, ErrInvalidParameter
	}
	_ = strings.TrimSpace(derefString(clientToken))

	count := int32(1)
	if instanceCount != nil {
		count = *instanceCount
	}
	if count <= 0 {
		return nil, ErrInvalidParameter
	}

	spotPriceValue := strings.TrimSpace(derefString(spotPrice))
	if spotPriceValue == "" {
		spotPriceValue = "0.0123"
	}

	launchGroupValue := strings.TrimSpace(derefString(launchGroup))
	availabilityZoneGroupValue := strings.TrimSpace(derefString(availabilityZoneGroup))
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]SpotInstanceRequest, 0, count)
	for i := int32(0); i < count; i++ {
		spotInstanceRequestID := s.nextIDLocked("sir")
		s.spotInstanceRequestStates[spotInstanceRequestID] = "open"
		state := "open"
		items = append(items, SpotInstanceRequest{
			AvailabilityZoneGroup: availabilityZoneGroupValue,
			CreateTime:            now,
			InstanceID:            "i-" + strings.TrimPrefix(spotInstanceRequestID, "sir-"),
			LaunchGroup:           launchGroupValue,
			ProductDescription:    "Linux/UNIX",
			SpotInstanceRequestID: spotInstanceRequestID,
			SpotPrice:             spotPriceValue,
			State:                 state,
			Status: SpotInstanceRequestStatus{
				Code:       stage122SpotInstanceStatusCode(state),
				Message:    stage122SpotInstanceStatusMessage(state),
				UpdateTime: now,
			},
			Tags: map[string]string{},
			Type: requestType,
		})
	}
	return items, nil
}

func (s *Service) ResetFpgaImageAttribute(fpgaImageID, attribute string) (bool, error) {
	fpgaImageID = strings.TrimSpace(fpgaImageID)
	attribute = strings.ToLower(strings.TrimSpace(attribute))
	if fpgaImageID == "" {
		return false, ErrInvalidParameter
	}
	if attribute != "" && attribute != "loadpermission" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fpgaImages[fpgaImageID] == nil {
		return false, ErrNotFound
	}
	return true, nil
}

func (s *Service) ResetSnapshotAttribute(snapshotID, attribute string) error {
	snapshotID = strings.TrimSpace(snapshotID)
	attribute = strings.ToLower(strings.TrimSpace(attribute))
	if snapshotID == "" || attribute == "" {
		return ErrInvalidParameter
	}
	if attribute != "createvolumepermission" && attribute != "productcodes" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshots[snapshotID]
	if snapshot == nil {
		return ErrNotFound
	}

	if attribute == "createvolumepermission" && snapshot.Tags != nil {
		delete(snapshot.Tags, stage130SnapshotPermissionGroupsTag)
		delete(snapshot.Tags, stage130SnapshotPermissionUsersTag)
	}
	return nil
}

func (s *Service) RestoreImageFromRecycleBin(imageID string) (bool, error) {
	imageID = strings.TrimSpace(imageID)
	if imageID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	image := s.images[imageID]
	if image == nil {
		return false, ErrNotFound
	}
	image.State = "available"
	if image.CreationDate.IsZero() {
		image.CreationDate = time.Now().UTC()
	}
	return true, nil
}

func (s *Service) RestoreManagedPrefixListVersion(prefixListID string, previousVersion, currentVersion int64) (ManagedPrefixList, error) {
	prefixListID = strings.TrimSpace(prefixListID)
	if prefixListID == "" || previousVersion <= 0 || currentVersion <= 0 || previousVersion >= currentVersion {
		return ManagedPrefixList{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prefixList := s.managedPrefixLists[prefixListID]
	if prefixList == nil {
		return ManagedPrefixList{}, ErrNotFound
	}
	if prefixList.Version != currentVersion {
		return ManagedPrefixList{}, ErrInvalidParameter
	}

	prefixList.Version = currentVersion + 1
	prefixList.State = "restore-complete"
	prefixList.StateMessage = "restored from version " + strconv.FormatInt(previousVersion, 10)
	return cloneStage109ManagedPrefixList(prefixList), nil
}

func stage132ValidReportStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ok", "impaired":
		return true
	default:
		return false
	}
}
