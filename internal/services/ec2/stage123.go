package ec2

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type VolumeAttribute struct {
	AutoEnableIO *bool
	ProductCodes []VolumeProductCode
	VolumeID     string
}

type VolumeProductCode struct {
	ProductCodeID   string
	ProductCodeType string
}

type VolumeStatus struct {
	AvailabilityZone   string
	AvailabilityZoneID string
	Status             string
	VolumeID           string
}

func (s *Service) DescribeVolumeAttribute(attribute, volumeID string) (VolumeAttribute, error) {
	attribute = strings.TrimSpace(attribute)
	volumeID = strings.TrimSpace(volumeID)
	if volumeID == "" {
		return VolumeAttribute{}, ErrInvalidParameter
	}
	if attribute != "" && !strings.EqualFold(attribute, "autoEnableIO") && !strings.EqualFold(attribute, "productCodes") {
		return VolumeAttribute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	volume := s.volumes[volumeID]
	if volume == nil {
		return VolumeAttribute{}, ErrNotFound
	}

	out := VolumeAttribute{VolumeID: volumeID}

	if !strings.EqualFold(attribute, "productCodes") {
		autoEnableIO := volume.AutoEnableIO
		out.AutoEnableIO = &autoEnableIO
	}

	if !strings.EqualFold(attribute, "autoEnableIO") {
		seen := map[string]struct{}{}
		for _, key := range []string{"product-code", "productCode"} {
			value := strings.TrimSpace(volume.Tags[key])
			if value == "" {
				continue
			}
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out.ProductCodes = append(out.ProductCodes, VolumeProductCode{
				ProductCodeID:   value,
				ProductCodeType: "marketplace",
			})
		}
	}

	return out, nil
}

func (s *Service) DescribeVolumeStatus(
	volumeIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VolumeStatus, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(volumeIDs)
	standardFilters, _, _ := splitEC2Filters(filters)
	volumeIDFilterSet := toStringSet(standardFilters["volume-id"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	availabilityZoneIDFilterSet := toLowerStringSet(standardFilters["availability-zone-id"])
	statusFilterSet := toLowerStringSet(standardFilters["volume-status.status"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.volumes))
		for volumeID := range s.volumes {
			candidateIDs = append(candidateIDs, volumeID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]VolumeStatus, 0, len(candidateIDs))
	for _, volumeID := range candidateIDs {
		volume := s.volumes[volumeID]
		if volume == nil {
			continue
		}

		item := VolumeStatus{
			AvailabilityZone:   volume.AvailabilityZone,
			AvailabilityZoneID: stage123AvailabilityZoneIDFromName(s.availabilityZones, volume.AvailabilityZone),
			Status:             stage123VolumeHealthFromState(volume.State),
			VolumeID:           volume.ID,
		}

		if len(volumeIDFilterSet) > 0 {
			if _, ok := volumeIDFilterSet[item.VolumeID]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if len(availabilityZoneIDFilterSet) > 0 {
			if _, ok := availabilityZoneIDFilterSet[strings.ToLower(item.AvailabilityZoneID)]; !ok {
				continue
			}
		}
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(item.Status)]; !ok {
				continue
			}
		}

		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]VolumeStatus(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeVolumesModifications(
	volumeIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VolumeModification, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(volumeIDs)
	standardFilters, _, _ := splitEC2Filters(filters)
	volumeIDFilterSet := toStringSet(standardFilters["volume-id"])
	stateFilterSet := toLowerStringSet(standardFilters["modification-state"])
	targetSizeFilterSet := toStringSet(standardFilters["target-size"])
	targetVolumeTypeFilterSet := toLowerStringSet(standardFilters["target-volume-type"])
	targetIopsFilterSet := toStringSet(standardFilters["target-iops"])
	targetThroughputFilterSet := toStringSet(standardFilters["target-throughput"])
	targetMultiAttachFilterSet := toLowerStringSet(standardFilters["target-multi-attach-enabled"])
	originalSizeFilterSet := toStringSet(standardFilters["original-size"])
	originalVolumeTypeFilterSet := toLowerStringSet(standardFilters["original-volume-type"])
	originalIopsFilterSet := toStringSet(standardFilters["original-iops"])
	originalThroughputFilterSet := toStringSet(standardFilters["original-throughput"])
	originalMultiAttachFilterSet := toLowerStringSet(standardFilters["original-multi-attach-enabled"])

	now := time.Now().UTC()

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.volumes))
		for volumeID := range s.volumes {
			candidateIDs = append(candidateIDs, volumeID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]VolumeModification, 0, len(candidateIDs))
	for _, volumeID := range candidateIDs {
		volume := s.volumes[volumeID]
		if volume == nil {
			continue
		}

		modification := VolumeModification{
			VolumeID:                   volume.ID,
			ModificationState:          "completed",
			StartTime:                  now,
			EndTime:                    now,
			Progress:                   100,
			StatusMessage:              "completed",
			OriginalSize:               volume.SizeGiB,
			TargetSize:                 volume.SizeGiB,
			OriginalVolumeType:         volume.VolumeType,
			TargetVolumeType:           volume.VolumeType,
			OriginalIops:               volume.Iops,
			TargetIops:                 volume.Iops,
			OriginalThroughput:         volume.Throughput,
			TargetThroughput:           volume.Throughput,
			OriginalMultiAttachEnabled: volume.MultiAttach,
			TargetMultiAttachEnabled:   volume.MultiAttach,
		}

		if len(volumeIDFilterSet) > 0 {
			if _, ok := volumeIDFilterSet[modification.VolumeID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(modification.ModificationState)]; !ok {
				continue
			}
		}
		if len(targetSizeFilterSet) > 0 {
			if _, ok := targetSizeFilterSet[strconv.FormatInt(int64(modification.TargetSize), 10)]; !ok {
				continue
			}
		}
		if len(targetVolumeTypeFilterSet) > 0 {
			if _, ok := targetVolumeTypeFilterSet[strings.ToLower(modification.TargetVolumeType)]; !ok {
				continue
			}
		}
		if len(targetIopsFilterSet) > 0 {
			if _, ok := targetIopsFilterSet[strconv.FormatInt(int64(modification.TargetIops), 10)]; !ok {
				continue
			}
		}
		if len(targetThroughputFilterSet) > 0 {
			if _, ok := targetThroughputFilterSet[strconv.FormatInt(int64(modification.TargetThroughput), 10)]; !ok {
				continue
			}
		}
		if len(targetMultiAttachFilterSet) > 0 {
			if !stage123MatchesBoolFilterSet(targetMultiAttachFilterSet, modification.TargetMultiAttachEnabled) {
				continue
			}
		}
		if len(originalSizeFilterSet) > 0 {
			if _, ok := originalSizeFilterSet[strconv.FormatInt(int64(modification.OriginalSize), 10)]; !ok {
				continue
			}
		}
		if len(originalVolumeTypeFilterSet) > 0 {
			if _, ok := originalVolumeTypeFilterSet[strings.ToLower(modification.OriginalVolumeType)]; !ok {
				continue
			}
		}
		if len(originalIopsFilterSet) > 0 {
			if _, ok := originalIopsFilterSet[strconv.FormatInt(int64(modification.OriginalIops), 10)]; !ok {
				continue
			}
		}
		if len(originalThroughputFilterSet) > 0 {
			if _, ok := originalThroughputFilterSet[strconv.FormatInt(int64(modification.OriginalThroughput), 10)]; !ok {
				continue
			}
		}
		if len(originalMultiAttachFilterSet) > 0 {
			if !stage123MatchesBoolFilterSet(originalMultiAttachFilterSet, modification.OriginalMultiAttachEnabled) {
				continue
			}
		}

		items = append(items, modification)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]VolumeModification(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DisassociateCapacityReservationBillingOwner(capacityReservationID, billingOwnerID string) (bool, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	billingOwnerID = strings.TrimSpace(billingOwnerID)
	if capacityReservationID == "" || billingOwnerID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing := strings.TrimSpace(s.capacityReservationBillingOwners[capacityReservationID])
	if existing == "" {
		return false, ErrNotFound
	}
	if existing != billingOwnerID {
		return false, ErrNotFound
	}

	delete(s.capacityReservationBillingOwners, capacityReservationID)
	return true, nil
}

func (s *Service) DisassociateEnclaveCertificateIamRole(certificateARN, roleARN string) (bool, error) {
	certificateARN = strings.TrimSpace(certificateARN)
	roleARN = strings.TrimSpace(roleARN)
	if certificateARN == "" || roleARN == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	associationsByRole := s.enclaveCertificateRoleAssociations[certificateARN]
	if associationsByRole == nil {
		return false, ErrNotFound
	}
	if _, ok := associationsByRole[roleARN]; !ok {
		return false, ErrNotFound
	}

	delete(associationsByRole, roleARN)
	if len(associationsByRole) == 0 {
		delete(s.enclaveCertificateRoleAssociations, certificateARN)
	}

	return true, nil
}

func (s *Service) DisassociateInstanceEventWindow(
	instanceEventWindowID string,
	dedicatedHostIDs []string,
	instanceIDs []string,
	instanceTags []Tag,
) (InstanceEventWindowAssociation, error) {
	instanceEventWindowID = strings.TrimSpace(instanceEventWindowID)
	if instanceEventWindowID == "" {
		return InstanceEventWindowAssociation{}, ErrInvalidParameter
	}

	dedicatedHostIDs = dedupeTrimmedStrings(dedicatedHostIDs)
	instanceIDs = dedupeTrimmedStrings(instanceIDs)
	instanceTags = normalizeEC2Tags(instanceTags)

	targetKinds := 0
	if len(dedicatedHostIDs) > 0 {
		targetKinds++
	}
	if len(instanceIDs) > 0 {
		targetKinds++
	}
	if len(instanceTags) > 0 {
		targetKinds++
	}
	if targetKinds != 1 {
		return InstanceEventWindowAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	association, ok := s.instanceEventWindowAssociations[instanceEventWindowID]
	if !ok {
		return InstanceEventWindowAssociation{}, ErrNotFound
	}

	updated := cloneInstanceEventWindowAssociation(association)
	matched := false

	switch {
	case len(dedicatedHostIDs) > 0:
		updated.DedicatedHostIDs, matched = stage123DisassociateStringTargets(updated.DedicatedHostIDs, dedicatedHostIDs)
	case len(instanceIDs) > 0:
		updated.InstanceIDs, matched = stage123DisassociateStringTargets(updated.InstanceIDs, instanceIDs)
	case len(instanceTags) > 0:
		updated.InstanceTags, matched = stage123DisassociateTagTargets(updated.InstanceTags, instanceTags)
	}

	if !matched {
		return InstanceEventWindowAssociation{}, ErrNotFound
	}

	if len(updated.DedicatedHostIDs) == 0 && len(updated.InstanceIDs) == 0 && len(updated.InstanceTags) == 0 {
		delete(s.instanceEventWindowAssociations, instanceEventWindowID)
		updated.State = "disassociated"
		return updated, nil
	}

	updated.State = "active"
	s.instanceEventWindowAssociations[instanceEventWindowID] = updated
	return cloneInstanceEventWindowAssociation(updated), nil
}

func (s *Service) DisassociateIpamByoasn(asn, cidr string) (ByoipAsnAssociation, error) {
	asn = strings.TrimSpace(asn)
	cidr = strings.TrimSpace(cidr)
	if asn == "" || cidr == "" {
		return ByoipAsnAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.byoipCidrs[cidr]
	if record == nil {
		return ByoipAsnAssociation{}, ErrNotFound
	}

	for i := range record.AsnAssociations {
		if !strings.EqualFold(strings.TrimSpace(record.AsnAssociations[i].Asn), asn) {
			continue
		}
		record.AsnAssociations[i].State = "disassociated"
		record.AsnAssociations[i].StatusMessage = "disassociated"
		return record.AsnAssociations[i], nil
	}

	return ByoipAsnAssociation{}, ErrNotFound
}

func (s *Service) DisassociateIpamResourceDiscovery(associationID string) (IpamResourceDiscoveryAssociation, error) {
	associationID = strings.TrimSpace(associationID)
	if associationID == "" {
		return IpamResourceDiscoveryAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	association := s.ipamResourceDiscoveryAssociations[associationID]
	if association == nil {
		return IpamResourceDiscoveryAssociation{}, ErrNotFound
	}

	out := cloneIpamResourceDiscoveryAssociation(association)
	out.ResourceDiscoveryStatus = "disassociated"
	out.State = "disassociate-complete"

	delete(s.ipamResourceDiscoveryAssociations, associationID)
	delete(s.ipamResourceDiscoveryAssociationByPair, association.IpamID+"|"+association.IpamResourceDiscoveryID)

	return out, nil
}

func (s *Service) ExportImage(
	clientToken,
	description,
	diskImageFormat,
	imageID,
	roleName string,
	s3Bucket,
	s3Prefix *string,
	tags []Tag,
) (ExportImageTask, error) {
	_ = strings.TrimSpace(clientToken)
	description = strings.TrimSpace(description)
	diskImageFormat = strings.TrimSpace(diskImageFormat)
	imageID = strings.TrimSpace(imageID)
	roleName = strings.TrimSpace(roleName)
	bucket := strings.TrimSpace(derefString(s3Bucket))
	prefix := strings.TrimSpace(derefString(s3Prefix))
	if imageID == "" || diskImageFormat == "" || bucket == "" {
		return ExportImageTask{}, ErrInvalidParameter
	}
	if roleName == "" {
		roleName = "vmimport"
	}
	if description == "" {
		description = "stackyard export image task"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.images[imageID] == nil {
		return ExportImageTask{}, ErrNotFound
	}

	taskID := s.nextIDLocked("export-ami")
	return ExportImageTask{
		Description:       description,
		ExportImageTaskID: taskID,
		ImageID:           imageID,
		Progress:          "100",
		S3ExportLocation: ExportTaskS3Location{
			S3Bucket: bucket,
			S3Prefix: prefix,
		},
		Status:          "completed",
		StatusMessage:   "export image task completed",
		Tags:            tagsToMap(tags),
		RoleName:        roleName,
		DiskImageFormat: diskImageFormat,
	}, nil
}

func (s *Service) GetAssociatedEnclaveCertificateIamRoles(certificateARN string) ([]EnclaveCertificateRoleAssociation, error) {
	certificateARN = strings.TrimSpace(certificateARN)
	if certificateARN == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	associationsByRole := s.enclaveCertificateRoleAssociations[certificateARN]
	if len(associationsByRole) == 0 {
		return []EnclaveCertificateRoleAssociation{}, nil
	}

	roleARNs := make([]string, 0, len(associationsByRole))
	for roleARN := range associationsByRole {
		roleARNs = append(roleARNs, roleARN)
	}
	sort.Strings(roleARNs)

	out := make([]EnclaveCertificateRoleAssociation, 0, len(roleARNs))
	for _, roleARN := range roleARNs {
		out = append(out, associationsByRole[roleARN])
	}
	return out, nil
}

func stage123AvailabilityZoneIDFromName(zones []AvailabilityZone, zoneName string) string {
	zoneName = strings.TrimSpace(zoneName)
	for _, zone := range zones {
		if strings.EqualFold(strings.TrimSpace(zone.Name), zoneName) {
			return strings.TrimSpace(zone.ZoneID)
		}
	}
	if strings.EqualFold(zoneName, "us-east-1a") {
		return "use1-az1"
	}
	if strings.EqualFold(zoneName, "us-east-1b") {
		return "use1-az2"
	}
	if strings.EqualFold(zoneName, "us-west-2a") {
		return "usw2-az1"
	}
	return ""
}

func stage123VolumeHealthFromState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "available", "in-use":
		return "ok"
	default:
		return "impaired"
	}
}

func stage123MatchesBoolFilterSet(filterSet map[string]struct{}, value bool) bool {
	if len(filterSet) == 0 {
		return true
	}
	if value {
		_, ok := filterSet["true"]
		return ok
	}
	_, ok := filterSet["false"]
	return ok
}

func stage123DisassociateStringTargets(existing, remove []string) ([]string, bool) {
	if len(existing) == 0 {
		return existing, false
	}
	removeSet := toStringSet(remove)
	if len(removeSet) == 0 {
		return existing, false
	}

	matched := false
	out := make([]string, 0, len(existing))
	for _, candidate := range existing {
		if _, ok := removeSet[candidate]; ok {
			matched = true
			continue
		}
		out = append(out, candidate)
	}
	return out, matched
}

func stage123DisassociateTagTargets(existing, remove []Tag) ([]Tag, bool) {
	if len(existing) == 0 || len(remove) == 0 {
		return existing, false
	}

	matched := false
	out := make([]Tag, 0, len(existing))
	for _, candidate := range existing {
		candidateKey := strings.TrimSpace(candidate.Key)
		candidateValue := strings.TrimSpace(candidate.Value)
		removeMatch := false
		for _, target := range remove {
			targetKey := strings.TrimSpace(target.Key)
			if targetKey == "" || !strings.EqualFold(targetKey, candidateKey) {
				continue
			}
			targetValue := strings.TrimSpace(target.Value)
			if targetValue != "" && targetValue != candidateValue {
				continue
			}
			removeMatch = true
			break
		}
		if removeMatch {
			matched = true
			continue
		}
		out = append(out, candidate)
	}
	return out, matched
}
