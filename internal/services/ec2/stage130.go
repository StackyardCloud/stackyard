package ec2

import (
	"sort"
	"strings"
	"time"
)

const (
	stage130SnapshotPermissionGroupsTag = "_stackyard_snapshot_permission_groups"
	stage130SnapshotPermissionUsersTag  = "_stackyard_snapshot_permission_users"
)

type ModifyReservedInstancesTargetConfiguration struct {
	AvailabilityZone   string
	AvailabilityZoneID string
	InstanceCount      int32
	InstanceType       string
	Platform           string
	Scope              string
}

type CidrAuthorizationContext struct {
	Message   string
	Signature string
}

func (s *Service) ModifyReservedInstances(
	reservedInstancesIDs []string,
	targetConfigurations []ModifyReservedInstancesTargetConfiguration,
	clientToken *string,
) (string, error) {
	reservedInstancesIDs = dedupeTrimmedStrings(reservedInstancesIDs)
	if len(reservedInstancesIDs) == 0 || len(targetConfigurations) == 0 {
		return "", ErrInvalidParameter
	}
	for _, cfg := range targetConfigurations {
		if cfg.InstanceCount <= 0 {
			return "", ErrInvalidParameter
		}
	}
	_ = strings.TrimSpace(derefString(clientToken))

	s.mu.Lock()
	defer s.mu.Unlock()

	suffix := stage130ReservedInstancesSuffix(reservedInstancesIDs[0])
	modificationID := ""
	if suffix == "" {
		modificationID = s.nextIDLocked("rimod")
		suffix = strings.TrimPrefix(modificationID, "rimod-")
	} else {
		modificationID = "rimod-" + suffix
	}

	listingID := "ril-" + suffix
	if _, ok := s.reservedInstancesListingCreatedAt[listingID]; !ok {
		s.reservedInstancesListingCreatedAt[listingID] = time.Now().UTC()
	}
	s.reservedInstancesListingStates[listingID] = "active"

	return modificationID, nil
}

func (s *Service) ModifySnapshotAttribute(
	snapshotID string,
	attribute string,
	operationType string,
	groupNames []string,
	userIDs []string,
	addPermissions []SnapshotCreateVolumePermission,
	removePermissions []SnapshotCreateVolumePermission,
) error {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return ErrInvalidParameter
	}

	attribute = strings.ToLower(strings.TrimSpace(attribute))
	if attribute == "" {
		attribute = "createvolumepermission"
	}
	if attribute != "createvolumepermission" && attribute != "productcodes" {
		return ErrInvalidParameter
	}

	operationType = strings.ToLower(strings.TrimSpace(operationType))
	if operationType != "" && operationType != "add" && operationType != "remove" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshots[snapshotID]
	if snapshot == nil {
		return ErrNotFound
	}

	if attribute != "createvolumepermission" {
		return nil
	}

	directPermissions := stage130SnapshotPermissionsFromSubjects(groupNames, userIDs)
	if len(directPermissions) > 0 {
		if operationType == "remove" {
			removePermissions = append(removePermissions, directPermissions...)
		} else {
			addPermissions = append(addPermissions, directPermissions...)
		}
	}

	normalizedAdds := stage130NormalizeSnapshotPermissions(addPermissions)
	normalizedRemoves := stage130NormalizeSnapshotPermissions(removePermissions)
	if len(normalizedAdds) == 0 && len(normalizedRemoves) == 0 {
		return nil
	}

	currentPermissions := stage130SnapshotPermissionsFromTags(snapshot.Tags)
	if len(currentPermissions) == 0 {
		currentPermissions = []SnapshotCreateVolumePermission{{UserID: DefaultAccountID}}
	}

	permissionSet := map[string]SnapshotCreateVolumePermission{}
	for _, permission := range currentPermissions {
		key := stage130SnapshotPermissionKey(permission)
		if key != "" {
			permissionSet[key] = permission
		}
	}
	for _, permission := range normalizedRemoves {
		delete(permissionSet, stage130SnapshotPermissionKey(permission))
	}
	for _, permission := range normalizedAdds {
		permissionSet[stage130SnapshotPermissionKey(permission)] = permission
	}

	updatedPermissions := make([]SnapshotCreateVolumePermission, 0, len(permissionSet))
	for _, permission := range permissionSet {
		updatedPermissions = append(updatedPermissions, permission)
	}
	sort.Slice(updatedPermissions, func(i, j int) bool {
		return stage130SnapshotPermissionKey(updatedPermissions[i]) < stage130SnapshotPermissionKey(updatedPermissions[j])
	})

	if snapshot.Tags == nil {
		snapshot.Tags = map[string]string{}
	}
	stage130SnapshotPermissionsToTags(snapshot.Tags, updatedPermissions)
	return nil
}

func (s *Service) ModifySnapshotTier(snapshotID string, storageTier string) (string, time.Time, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	storageTier = strings.ToLower(strings.TrimSpace(storageTier))
	if snapshotID == "" {
		return "", time.Time{}, ErrInvalidParameter
	}
	if storageTier == "" {
		storageTier = "archive"
	}
	if storageTier != "archive" {
		return "", time.Time{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshots[snapshotID]
	if snapshot == nil {
		return "", time.Time{}, ErrNotFound
	}
	if snapshot.Tags == nil {
		snapshot.Tags = map[string]string{}
	}
	snapshot.Tags["storage-tier"] = storageTier

	tieringStartTime := time.Now().UTC()
	return snapshot.ID, tieringStartTime, nil
}

func (s *Service) ModifySpotFleetRequest(
	spotFleetRequestID string,
	context *string,
	excessCapacityTerminationPolicy string,
	launchTemplateConfigsPresent bool,
	onDemandTargetCapacity *int32,
	targetCapacity *int32,
) (bool, error) {
	spotFleetRequestID = strings.TrimSpace(spotFleetRequestID)
	if spotFleetRequestID == "" {
		return false, ErrInvalidParameter
	}
	if onDemandTargetCapacity != nil && *onDemandTargetCapacity < 0 {
		return false, ErrInvalidParameter
	}
	if targetCapacity != nil && *targetCapacity < 0 {
		return false, ErrInvalidParameter
	}
	if _, ok := stage130NormalizeExcessCapacityTerminationPolicy(excessCapacityTerminationPolicy); !ok {
		return false, ErrInvalidParameter
	}
	_ = strings.TrimSpace(derefString(context))
	_ = launchTemplateConfigsPresent

	s.mu.Lock()
	defer s.mu.Unlock()

	state := strings.TrimSpace(s.spotFleetRequestStates[spotFleetRequestID])
	if state == "" {
		state = "active"
	}
	s.spotFleetRequestStates[spotFleetRequestID] = state
	return true, nil
}

func (s *Service) ModifyTrafficMirrorFilterNetworkServices(
	trafficMirrorFilterID string,
	addNetworkServices []string,
	removeNetworkServices []string,
) (TrafficMirrorFilter, error) {
	trafficMirrorFilterID = strings.TrimSpace(trafficMirrorFilterID)
	if trafficMirrorFilterID == "" {
		return TrafficMirrorFilter{}, ErrInvalidParameter
	}

	add, ok := stage130NormalizeTrafficMirrorNetworkServices(addNetworkServices)
	if !ok {
		return TrafficMirrorFilter{}, ErrInvalidParameter
	}
	remove, ok := stage130NormalizeTrafficMirrorNetworkServices(removeNetworkServices)
	if !ok {
		return TrafficMirrorFilter{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filter := s.trafficMirrorFilters[trafficMirrorFilterID]
	if filter == nil {
		return TrafficMirrorFilter{}, ErrNotFound
	}

	serviceSet := toStringSet(filter.NetworkServices)
	for _, service := range remove {
		delete(serviceSet, service)
	}
	for _, service := range add {
		serviceSet[service] = struct{}{}
	}

	filter.NetworkServices = make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		filter.NetworkServices = append(filter.NetworkServices, service)
	}
	sort.Strings(filter.NetworkServices)

	return cloneStage110TrafficMirrorFilter(filter), nil
}

func (s *Service) ModifyTrafficMirrorFilterRule(
	trafficMirrorFilterRuleID string,
	description *string,
	destinationCidrBlock *string,
	destinationPortRange *TrafficMirrorPortRange,
	protocol *int32,
	removeFields []string,
	ruleAction string,
	ruleNumber *int32,
	sourceCidrBlock *string,
	sourcePortRange *TrafficMirrorPortRange,
	trafficDirection string,
) (TrafficMirrorFilterRule, error) {
	trafficMirrorFilterRuleID = strings.TrimSpace(trafficMirrorFilterRuleID)
	if trafficMirrorFilterRuleID == "" {
		return TrafficMirrorFilterRule{}, ErrInvalidParameter
	}
	if !stage130ValidTrafficMirrorPortRange(destinationPortRange) || !stage130ValidTrafficMirrorPortRange(sourcePortRange) {
		return TrafficMirrorFilterRule{}, ErrInvalidParameter
	}
	if protocol != nil && (*protocol < -1 || *protocol > 255) {
		return TrafficMirrorFilterRule{}, ErrInvalidParameter
	}
	if ruleNumber != nil && *ruleNumber <= 0 {
		return TrafficMirrorFilterRule{}, ErrInvalidParameter
	}
	if ruleAction != "" {
		ruleAction = strings.ToLower(strings.TrimSpace(ruleAction))
		if ruleAction != "accept" && ruleAction != "reject" {
			return TrafficMirrorFilterRule{}, ErrInvalidParameter
		}
	}
	if trafficDirection != "" {
		trafficDirection = strings.ToLower(strings.TrimSpace(trafficDirection))
		if trafficDirection != "ingress" && trafficDirection != "egress" {
			return TrafficMirrorFilterRule{}, ErrInvalidParameter
		}
	}

	normalizedRemoveFields := stage130NormalizeStringList(removeFields)
	for _, field := range normalizedRemoveFields {
		switch field {
		case "description", "destination-port-range", "source-port-range", "protocol":
		default:
			return TrafficMirrorFilterRule{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rule := s.trafficMirrorFilterRules[trafficMirrorFilterRuleID]
	if rule == nil {
		return TrafficMirrorFilterRule{}, ErrNotFound
	}

	for _, field := range normalizedRemoveFields {
		switch field {
		case "description":
			rule.Description = ""
		case "destination-port-range":
			rule.DestinationPortRange = TrafficMirrorPortRange{}
		case "source-port-range":
			rule.SourcePortRange = TrafficMirrorPortRange{}
		case "protocol":
			rule.Protocol = nil
		}
	}

	if description != nil {
		rule.Description = strings.TrimSpace(*description)
	}
	if destinationCidrBlock != nil {
		rule.DestinationCidrBlock = strings.TrimSpace(*destinationCidrBlock)
	}
	if destinationPortRange != nil {
		rule.DestinationPortRange = cloneStage110TrafficMirrorPortRange(*destinationPortRange)
	}
	if protocol != nil {
		rule.Protocol = cloneInt32Pointer(protocol)
	}
	if ruleAction != "" {
		rule.RuleAction = ruleAction
	}
	if ruleNumber != nil {
		rule.RuleNumber = *ruleNumber
	}
	if sourceCidrBlock != nil {
		rule.SourceCidrBlock = strings.TrimSpace(*sourceCidrBlock)
	}
	if sourcePortRange != nil {
		rule.SourcePortRange = cloneStage110TrafficMirrorPortRange(*sourcePortRange)
	}
	if trafficDirection != "" {
		rule.TrafficDirection = trafficDirection
	}

	return cloneStage110TrafficMirrorFilterRule(rule), nil
}

func (s *Service) ModifyTrafficMirrorSession(
	trafficMirrorSessionID string,
	description *string,
	packetLength *int32,
	removeFields []string,
	sessionNumber *int32,
	trafficMirrorFilterID *string,
	trafficMirrorTargetID *string,
	virtualNetworkID *int32,
) (TrafficMirrorSession, error) {
	trafficMirrorSessionID = strings.TrimSpace(trafficMirrorSessionID)
	if trafficMirrorSessionID == "" {
		return TrafficMirrorSession{}, ErrInvalidParameter
	}
	if packetLength != nil && (*packetLength <= 0 || *packetLength > 8500) {
		return TrafficMirrorSession{}, ErrInvalidParameter
	}
	if sessionNumber != nil && (*sessionNumber <= 0 || *sessionNumber > 32766) {
		return TrafficMirrorSession{}, ErrInvalidParameter
	}
	if virtualNetworkID != nil && *virtualNetworkID <= 0 {
		return TrafficMirrorSession{}, ErrInvalidParameter
	}

	normalizedRemoveFields := stage130NormalizeStringList(removeFields)
	for _, field := range normalizedRemoveFields {
		switch field {
		case "description", "packet-length", "virtual-network-id":
		default:
			return TrafficMirrorSession{}, ErrInvalidParameter
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.trafficMirrorSessions[trafficMirrorSessionID]
	if session == nil {
		return TrafficMirrorSession{}, ErrNotFound
	}

	if trafficMirrorFilterID != nil {
		filterID := strings.TrimSpace(*trafficMirrorFilterID)
		if filterID == "" || s.trafficMirrorFilters[filterID] == nil {
			return TrafficMirrorSession{}, ErrNotFound
		}
	}
	if trafficMirrorTargetID != nil {
		targetID := strings.TrimSpace(*trafficMirrorTargetID)
		if targetID == "" || s.trafficMirrorTargets[targetID] == nil {
			return TrafficMirrorSession{}, ErrNotFound
		}
	}

	for _, field := range normalizedRemoveFields {
		switch field {
		case "description":
			session.Description = ""
		case "packet-length":
			session.PacketLength = nil
		case "virtual-network-id":
			session.VirtualNetworkID = nil
		}
	}

	if description != nil {
		session.Description = strings.TrimSpace(*description)
	}
	if packetLength != nil {
		session.PacketLength = cloneInt32Pointer(packetLength)
	}
	if sessionNumber != nil {
		session.SessionNumber = *sessionNumber
	}
	if trafficMirrorFilterID != nil {
		session.TrafficMirrorFilterID = strings.TrimSpace(*trafficMirrorFilterID)
	}
	if trafficMirrorTargetID != nil {
		session.TrafficMirrorTargetID = strings.TrimSpace(*trafficMirrorTargetID)
	}
	if virtualNetworkID != nil {
		session.VirtualNetworkID = cloneInt32Pointer(virtualNetworkID)
	}

	return cloneStage110TrafficMirrorSession(session), nil
}

func (s *Service) MoveByoipCidrToIpam(cidr, ipamPoolID, ipamPoolOwner string) (ByoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	ipamPoolOwner = strings.TrimSpace(ipamPoolOwner)
	if cidr == "" || ipamPoolID == "" || ipamPoolOwner == "" {
		return ByoipCidr{}, ErrInvalidParameter
	}
	if ipamPoolOwner != DefaultAccountID {
		return ByoipCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ipamPools[ipamPoolID] == nil {
		return ByoipCidr{}, ErrNotFound
	}

	record := s.byoipCidrs[cidr]
	if record == nil {
		return ByoipCidr{}, ErrNotFound
	}
	record.State = "provisioned"
	record.StatusMessage = "moved-to-ipam"
	return cloneByoipCidr(record), nil
}

func (s *Service) MoveCapacityReservationInstances(
	destinationCapacityReservationID string,
	instanceCount int32,
	sourceCapacityReservationID string,
	clientToken *string,
) (CapacityReservation, CapacityReservation, int32, error) {
	destinationCapacityReservationID = strings.TrimSpace(destinationCapacityReservationID)
	sourceCapacityReservationID = strings.TrimSpace(sourceCapacityReservationID)
	if destinationCapacityReservationID == "" || sourceCapacityReservationID == "" || instanceCount <= 0 {
		return CapacityReservation{}, CapacityReservation{}, 0, ErrInvalidParameter
	}
	if destinationCapacityReservationID == sourceCapacityReservationID {
		return CapacityReservation{}, CapacityReservation{}, 0, ErrInvalidParameter
	}
	_ = strings.TrimSpace(derefString(clientToken))

	s.mu.Lock()
	defer s.mu.Unlock()

	source := s.capacityReservations[sourceCapacityReservationID]
	if source == nil {
		return CapacityReservation{}, CapacityReservation{}, 0, ErrNotFound
	}
	destination := s.capacityReservations[destinationCapacityReservationID]
	if destination == nil {
		return CapacityReservation{}, CapacityReservation{}, 0, ErrNotFound
	}
	if source.AvailableInstanceCount < instanceCount || source.TotalInstanceCount < instanceCount {
		return CapacityReservation{}, CapacityReservation{}, 0, ErrConflict
	}

	source.TotalInstanceCount -= instanceCount
	source.AvailableInstanceCount -= instanceCount
	if source.AvailableInstanceCount < 0 {
		source.AvailableInstanceCount = 0
	}

	destination.TotalInstanceCount += instanceCount
	destination.AvailableInstanceCount += instanceCount

	return cloneCapacityReservation(source), cloneCapacityReservation(destination), instanceCount, nil
}

func (s *Service) ProvisionByoipCidr(
	cidr string,
	cidrAuthorizationContext *CidrAuthorizationContext,
	description *string,
	multiRegion *bool,
	networkBorderGroup *string,
	tags []Tag,
	publiclyAdvertisable *bool,
) (ByoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return ByoipCidr{}, ErrInvalidParameter
	}
	if cidrAuthorizationContext != nil {
		if strings.TrimSpace(cidrAuthorizationContext.Message) == "" || strings.TrimSpace(cidrAuthorizationContext.Signature) == "" {
			return ByoipCidr{}, ErrInvalidParameter
		}
	}
	if networkBorderGroup != nil && strings.TrimSpace(*networkBorderGroup) == "" {
		return ByoipCidr{}, ErrInvalidParameter
	}
	_ = multiRegion
	_ = tags
	_ = publiclyAdvertisable

	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.byoipCidrs[cidr]
	if record == nil {
		record = &ByoipCidr{
			Cidr: cidr,
		}
		s.byoipCidrs[cidr] = record
	}

	record.Cidr = cidr
	if description != nil {
		record.Description = strings.TrimSpace(*description)
	}
	if networkBorderGroup != nil {
		record.NetworkBorderGroup = strings.TrimSpace(*networkBorderGroup)
	}
	record.State = "provisioned"
	record.StatusMessage = "provisioned"

	return cloneByoipCidr(record), nil
}

func stage130ReservedInstancesSuffix(reservedInstancesID string) string {
	suffix := strings.TrimSpace(strings.TrimPrefix(reservedInstancesID, "ri-"))
	suffix = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, suffix)
	return suffix
}

func stage130SnapshotPermissionsFromSubjects(groupNames, userIDs []string) []SnapshotCreateVolumePermission {
	out := make([]SnapshotCreateVolumePermission, 0, len(groupNames)+len(userIDs))
	for _, groupName := range dedupeTrimmedStrings(groupNames) {
		out = append(out, SnapshotCreateVolumePermission{Group: strings.ToLower(groupName)})
	}
	for _, userID := range dedupeTrimmedStrings(userIDs) {
		out = append(out, SnapshotCreateVolumePermission{UserID: userID})
	}
	return out
}

func stage130NormalizeSnapshotPermissions(in []SnapshotCreateVolumePermission) []SnapshotCreateVolumePermission {
	out := make([]SnapshotCreateVolumePermission, 0, len(in))
	seen := map[string]struct{}{}
	for _, permission := range in {
		normalized, ok := stage130NormalizeSnapshotPermission(permission)
		if !ok {
			continue
		}
		key := stage130SnapshotPermissionKey(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func stage130NormalizeSnapshotPermission(permission SnapshotCreateVolumePermission) (SnapshotCreateVolumePermission, bool) {
	group := strings.ToLower(strings.TrimSpace(permission.Group))
	userID := strings.TrimSpace(permission.UserID)
	if group == "" && userID == "" {
		return SnapshotCreateVolumePermission{}, false
	}
	if group != "" {
		return SnapshotCreateVolumePermission{Group: group}, true
	}
	return SnapshotCreateVolumePermission{UserID: userID}, true
}

func stage130SnapshotPermissionKey(permission SnapshotCreateVolumePermission) string {
	if permission.Group != "" {
		return "group:" + strings.ToLower(strings.TrimSpace(permission.Group))
	}
	if permission.UserID != "" {
		return "user:" + strings.TrimSpace(permission.UserID)
	}
	return ""
}

func stage130SnapshotPermissionsFromTags(tags map[string]string) []SnapshotCreateVolumePermission {
	if len(tags) == 0 {
		return nil
	}

	out := make([]SnapshotCreateVolumePermission, 0)
	for _, group := range strings.Split(strings.TrimSpace(tags[stage130SnapshotPermissionGroupsTag]), ",") {
		group = strings.ToLower(strings.TrimSpace(group))
		if group == "" {
			continue
		}
		out = append(out, SnapshotCreateVolumePermission{Group: group})
	}
	for _, userID := range strings.Split(strings.TrimSpace(tags[stage130SnapshotPermissionUsersTag]), ",") {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		out = append(out, SnapshotCreateVolumePermission{UserID: userID})
	}
	return stage130NormalizeSnapshotPermissions(out)
}

func stage130SnapshotPermissionsToTags(tags map[string]string, permissions []SnapshotCreateVolumePermission) {
	if tags == nil {
		return
	}

	groups := make([]string, 0)
	userIDs := make([]string, 0)
	for _, permission := range permissions {
		if permission.Group != "" {
			groups = append(groups, strings.ToLower(strings.TrimSpace(permission.Group)))
		}
		if permission.UserID != "" {
			userIDs = append(userIDs, strings.TrimSpace(permission.UserID))
		}
	}
	groups = dedupeTrimmedStrings(groups)
	userIDs = dedupeTrimmedStrings(userIDs)
	sort.Strings(groups)
	sort.Strings(userIDs)

	if len(groups) == 0 {
		delete(tags, stage130SnapshotPermissionGroupsTag)
	} else {
		tags[stage130SnapshotPermissionGroupsTag] = strings.Join(groups, ",")
	}
	if len(userIDs) == 0 {
		delete(tags, stage130SnapshotPermissionUsersTag)
	} else {
		tags[stage130SnapshotPermissionUsersTag] = strings.Join(userIDs, ",")
	}
}

func stage130NormalizeExcessCapacityTerminationPolicy(policy string) (string, bool) {
	policy = strings.TrimSpace(policy)
	if policy == "" {
		return "", true
	}
	key := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(policy, "-", ""), "_", ""))
	switch key {
	case "termination":
		return "termination", true
	case "notermination":
		return "noTermination", true
	default:
		return "", false
	}
}

func stage130NormalizeTrafficMirrorNetworkServices(services []string) ([]string, bool) {
	services = stage130NormalizeStringList(services)
	out := make([]string, 0, len(services))
	for _, service := range services {
		switch service {
		case "amazon-dns":
			out = append(out, service)
		default:
			return nil, false
		}
	}
	return out, true
}

func stage130ValidTrafficMirrorPortRange(portRange *TrafficMirrorPortRange) bool {
	if portRange == nil {
		return true
	}
	if (portRange.FromPort == nil) != (portRange.ToPort == nil) {
		return false
	}
	if portRange.FromPort == nil && portRange.ToPort == nil {
		return true
	}
	from := *portRange.FromPort
	to := *portRange.ToPort
	if from < 0 || from > 65535 || to < 0 || to > 65535 || from > to {
		return false
	}
	return true
}

func stage130NormalizeStringList(items []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.ToLower(strings.TrimSpace(item))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
