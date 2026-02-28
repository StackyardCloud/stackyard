package ec2

import (
	"strings"
	"time"
)

type InstanceCreditSpecificationRequest struct {
	CpuCredits string
	InstanceID string
}

type InstanceCreditSpecificationUnsuccessful struct {
	Code       string
	InstanceID string
	Message    string
}

type ModifiedInstanceEvent struct {
	Code              string
	Description       string
	InstanceEventID   string
	NotAfter          *time.Time
	NotBefore         *time.Time
	NotBeforeDeadline *time.Time
}

type ModifiedInstanceMaintenanceOptions struct {
	AutoRecovery    string
	InstanceID      string
	RebootMigration string
}

type ModifiedInstanceMetadataOptions struct {
	HttpEndpoint            string
	HttpProtocolIpv6        string
	HttpPutResponseHopLimit *int32
	HttpTokens              string
	InstanceID              string
	InstanceMetadataTags    string
	State                   string
}

type instancePlacementOptions struct {
	Affinity             string
	GroupID              string
	GroupName            string
	HostID               string
	HostResourceGroupARN string
	PartitionNumber      *int32
	Tenancy              string
}

func (s *Service) ModifyInstanceConnectEndpoint(
	instanceConnectEndpointID string,
	ipAddressType *string,
	preserveClientIP *bool,
	securityGroupIDs []string,
) (bool, error) {
	instanceConnectEndpointID = strings.TrimSpace(instanceConnectEndpointID)
	if instanceConnectEndpointID == "" {
		return false, ErrInvalidParameter
	}

	ipAddressTypeValue := strings.ToLower(strings.TrimSpace(derefString(ipAddressType)))
	if ipAddressType != nil && ipAddressTypeValue == "" {
		return false, ErrInvalidParameter
	}
	if ipAddressTypeValue != "" && !stage128IsAllowedValue(ipAddressTypeValue, "ipv4", "ipv6", "dualstack") {
		return false, ErrInvalidParameter
	}

	securityGroupIDs = dedupeTrimmedStrings(securityGroupIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	endpoint := s.instanceConnectEndpoints[instanceConnectEndpointID]
	if endpoint == nil {
		return false, ErrNotFound
	}

	if ipAddressTypeValue != "" {
		endpoint.IPAddressType = ipAddressTypeValue
	}
	if preserveClientIP != nil {
		effectiveIPType := strings.ToLower(strings.TrimSpace(endpoint.IPAddressType))
		if *preserveClientIP && effectiveIPType != "ipv4" {
			return false, ErrInvalidParameter
		}
		endpoint.PreserveClientIP = *preserveClientIP
	}
	if len(securityGroupIDs) > 0 {
		endpoint.SecurityGroupIDs = append([]string(nil), securityGroupIDs...)
	}

	return true, nil
}

func (s *Service) ModifyInstanceCpuOptions(instanceID string, coreCount, threadsPerCore int32) (int32, int32, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" || coreCount <= 0 || threadsPerCore <= 0 {
		return 0, 0, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return 0, 0, ErrNotFound
	}

	return coreCount, threadsPerCore, nil
}

func (s *Service) ModifyInstanceCreditSpecification(specifications []InstanceCreditSpecificationRequest) ([]string, []InstanceCreditSpecificationUnsuccessful, error) {
	if len(specifications) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	successful := make([]string, 0, len(specifications))
	unsuccessful := make([]InstanceCreditSpecificationUnsuccessful, 0)
	for _, specification := range specifications {
		instanceID := strings.TrimSpace(specification.InstanceID)
		cpuCredits := strings.ToLower(strings.TrimSpace(specification.CpuCredits))
		if instanceID == "" || cpuCredits == "" {
			return nil, nil, ErrInvalidParameter
		}
		if !stage128IsAllowedValue(cpuCredits, "standard", "unlimited") {
			return nil, nil, ErrInvalidParameter
		}

		if s.instances[instanceID] == nil {
			unsuccessful = append(unsuccessful, InstanceCreditSpecificationUnsuccessful{
				Code:       "InvalidInstanceID.NotFound",
				InstanceID: instanceID,
				Message:    "instance not found",
			})
			continue
		}

		s.instanceCreditSpecifications[instanceID] = cpuCredits
		successful = append(successful, instanceID)
	}

	return successful, unsuccessful, nil
}

func (s *Service) ModifyInstanceEventStartTime(instanceID, instanceEventID string, notBefore time.Time) (ModifiedInstanceEvent, error) {
	instanceID = strings.TrimSpace(instanceID)
	instanceEventID = strings.TrimSpace(instanceEventID)
	if instanceID == "" || instanceEventID == "" || notBefore.IsZero() {
		return ModifiedInstanceEvent{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return ModifiedInstanceEvent{}, ErrNotFound
	}

	notBeforeUTC := notBefore.UTC()
	notAfterUTC := notBeforeUTC.Add(1 * time.Hour)
	notBeforeDeadlineUTC := notBeforeUTC.Add(-30 * time.Minute)
	event := ModifiedInstanceEvent{
		Code:              "system-reboot",
		Description:       "scheduled event start time modified",
		InstanceEventID:   instanceEventID,
		NotAfter:          cloneTimePointer(&notAfterUTC),
		NotBefore:         cloneTimePointer(&notBeforeUTC),
		NotBeforeDeadline: cloneTimePointer(&notBeforeDeadlineUTC),
	}

	if s.instanceStatusEvents[instanceID] == nil {
		s.instanceStatusEvents[instanceID] = map[string]ModifiedInstanceEvent{}
	}
	s.instanceStatusEvents[instanceID][instanceEventID] = cloneStage128ModifiedInstanceEvent(event)
	return event, nil
}

func (s *Service) ModifyInstanceEventWindow(
	instanceEventWindowID string,
	name *string,
	cronExpression *string,
	timeRanges []InstanceEventWindowTimeRange,
	hasTimeRanges bool,
) (InstanceEventWindowDescription, error) {
	instanceEventWindowID = strings.TrimSpace(instanceEventWindowID)
	if instanceEventWindowID == "" {
		return InstanceEventWindowDescription{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	window := s.instanceEventWindows[instanceEventWindowID]
	if window == nil {
		return InstanceEventWindowDescription{}, ErrNotFound
	}

	if name != nil {
		window.Name = strings.TrimSpace(*name)
	}
	if cronExpression != nil {
		trimmed := strings.TrimSpace(*cronExpression)
		if trimmed == "" {
			return InstanceEventWindowDescription{}, ErrInvalidParameter
		}
		window.CronExpression = trimmed
	}
	if hasTimeRanges {
		window.TimeRanges = cloneStage107EventWindowTimeRanges(timeRanges)
	}

	association := s.instanceEventWindowAssociations[instanceEventWindowID]
	out := InstanceEventWindowDescription{
		AssociationDedicatedHostIDs: append([]string(nil), association.DedicatedHostIDs...),
		AssociationInstanceIDs:      append([]string(nil), association.InstanceIDs...),
		AssociationInstanceTags:     cloneEC2Tags(association.InstanceTags),
		CronExpression:              window.CronExpression,
		InstanceEventWindowID:       window.InstanceEventWindowID,
		Name:                        window.Name,
		State:                       firstNonEmptyString(window.State, "active"),
		Tags:                        cloneStringMap(window.Tags),
		TimeRanges:                  cloneStage107EventWindowTimeRanges(window.TimeRanges),
	}
	return cloneStage117InstanceEventWindow(out), nil
}

func (s *Service) ModifyInstanceMaintenanceOptions(instanceID string, autoRecovery, rebootMigration *string) (ModifiedInstanceMaintenanceOptions, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return ModifiedInstanceMaintenanceOptions{}, ErrInvalidParameter
	}

	autoRecoveryValue := strings.ToLower(strings.TrimSpace(derefString(autoRecovery)))
	rebootMigrationValue := strings.ToLower(strings.TrimSpace(derefString(rebootMigration)))
	if autoRecovery != nil && !stage128IsAllowedValue(autoRecoveryValue, "default", "disabled") {
		return ModifiedInstanceMaintenanceOptions{}, ErrInvalidParameter
	}
	if rebootMigration != nil && !stage128IsAllowedValue(rebootMigrationValue, "default", "disabled") {
		return ModifiedInstanceMaintenanceOptions{}, ErrInvalidParameter
	}
	if autoRecovery == nil && rebootMigration == nil {
		return ModifiedInstanceMaintenanceOptions{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return ModifiedInstanceMaintenanceOptions{}, ErrNotFound
	}

	current := s.instanceMaintenanceOptions[instanceID]
	if strings.TrimSpace(current.AutoRecovery) == "" {
		current.AutoRecovery = "default"
	}
	if strings.TrimSpace(current.RebootMigration) == "" {
		current.RebootMigration = "default"
	}
	if autoRecovery != nil {
		current.AutoRecovery = autoRecoveryValue
	}
	if rebootMigration != nil {
		current.RebootMigration = rebootMigrationValue
	}
	current.InstanceID = instanceID
	s.instanceMaintenanceOptions[instanceID] = current
	return current, nil
}

func (s *Service) ModifyInstanceMetadataDefaults(httpEndpoint, httpTokens, instanceMetadataTags *string, httpPutResponseHopLimit *int32) (bool, error) {
	if httpEndpoint == nil && httpTokens == nil && instanceMetadataTags == nil && httpPutResponseHopLimit == nil {
		return false, ErrInvalidParameter
	}

	httpEndpointValue := strings.ToLower(strings.TrimSpace(derefString(httpEndpoint)))
	httpTokensValue := strings.ToLower(strings.TrimSpace(derefString(httpTokens)))
	instanceMetadataTagsValue := strings.ToLower(strings.TrimSpace(derefString(instanceMetadataTags)))

	if httpEndpoint != nil && !stage128IsAllowedValue(httpEndpointValue, "enabled", "disabled", "no-preference") {
		return false, ErrInvalidParameter
	}
	if httpTokens != nil && !stage128IsAllowedValue(httpTokensValue, "optional", "required", "no-preference") {
		return false, ErrInvalidParameter
	}
	if instanceMetadataTags != nil && !stage128IsAllowedValue(instanceMetadataTagsValue, "enabled", "disabled", "no-preference") {
		return false, ErrInvalidParameter
	}
	if httpPutResponseHopLimit != nil && *httpPutResponseHopLimit != -1 && (*httpPutResponseHopLimit < 1 || *httpPutResponseHopLimit > 64) {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.instanceMetadataDefaults
	if strings.TrimSpace(current.HttpEndpoint) == "" {
		current.HttpEndpoint = "enabled"
	}
	if current.HttpPutResponseHopLimit == nil {
		defaultHopLimit := int32(2)
		current.HttpPutResponseHopLimit = &defaultHopLimit
	}
	if strings.TrimSpace(current.HttpTokens) == "" {
		current.HttpTokens = "optional"
	}
	if strings.TrimSpace(current.InstanceMetadataTags) == "" {
		current.InstanceMetadataTags = "disabled"
	}
	if strings.TrimSpace(current.ManagedBy) == "" {
		current.ManagedBy = "account"
	}

	if httpEndpoint != nil {
		current.HttpEndpoint = httpEndpointValue
	}
	if httpTokens != nil {
		current.HttpTokens = httpTokensValue
	}
	if instanceMetadataTags != nil {
		current.InstanceMetadataTags = instanceMetadataTagsValue
	}
	if httpPutResponseHopLimit != nil {
		current.HttpPutResponseHopLimit = cloneInt32Pointer(httpPutResponseHopLimit)
	}

	s.instanceMetadataDefaults = current
	return true, nil
}

func (s *Service) ModifyInstanceMetadataOptions(
	instanceID string,
	httpEndpoint, httpProtocolIpv6, httpTokens, instanceMetadataTags *string,
	httpPutResponseHopLimit *int32,
) (ModifiedInstanceMetadataOptions, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}
	if httpEndpoint == nil && httpProtocolIpv6 == nil && httpTokens == nil && instanceMetadataTags == nil && httpPutResponseHopLimit == nil {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}

	httpEndpointValue := strings.ToLower(strings.TrimSpace(derefString(httpEndpoint)))
	httpProtocolIpv6Value := strings.ToLower(strings.TrimSpace(derefString(httpProtocolIpv6)))
	httpTokensValue := strings.ToLower(strings.TrimSpace(derefString(httpTokens)))
	instanceMetadataTagsValue := strings.ToLower(strings.TrimSpace(derefString(instanceMetadataTags)))

	if httpEndpoint != nil && !stage128IsAllowedValue(httpEndpointValue, "enabled", "disabled") {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}
	if httpProtocolIpv6 != nil && !stage128IsAllowedValue(httpProtocolIpv6Value, "enabled", "disabled") {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}
	if httpTokens != nil && !stage128IsAllowedValue(httpTokensValue, "optional", "required") {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}
	if instanceMetadataTags != nil && !stage128IsAllowedValue(instanceMetadataTagsValue, "enabled", "disabled") {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}
	if httpPutResponseHopLimit != nil && (*httpPutResponseHopLimit < 1 || *httpPutResponseHopLimit > 64) {
		return ModifiedInstanceMetadataOptions{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return ModifiedInstanceMetadataOptions{}, ErrNotFound
	}

	current := s.instanceMetadataOptions[instanceID]
	if strings.TrimSpace(current.InstanceID) == "" {
		current = ModifiedInstanceMetadataOptions{
			HttpEndpoint:            stage128ResolveMetadataDefaultValue(s.instanceMetadataDefaults.HttpEndpoint, "enabled"),
			HttpProtocolIpv6:        "disabled",
			HttpPutResponseHopLimit: cloneInt32Pointer(s.instanceMetadataDefaults.HttpPutResponseHopLimit),
			HttpTokens:              stage128ResolveMetadataDefaultValue(s.instanceMetadataDefaults.HttpTokens, "optional"),
			InstanceID:              instanceID,
			InstanceMetadataTags:    stage128ResolveMetadataDefaultValue(s.instanceMetadataDefaults.InstanceMetadataTags, "disabled"),
			State:                   "applied",
		}
		if current.HttpPutResponseHopLimit == nil {
			defaultHopLimit := int32(2)
			current.HttpPutResponseHopLimit = &defaultHopLimit
		}
	}

	if httpEndpoint != nil {
		current.HttpEndpoint = httpEndpointValue
	}
	if httpProtocolIpv6 != nil {
		current.HttpProtocolIpv6 = httpProtocolIpv6Value
	}
	if httpTokens != nil {
		current.HttpTokens = httpTokensValue
	}
	if instanceMetadataTags != nil {
		current.InstanceMetadataTags = instanceMetadataTagsValue
	}
	if httpPutResponseHopLimit != nil {
		current.HttpPutResponseHopLimit = cloneInt32Pointer(httpPutResponseHopLimit)
	}
	current.InstanceID = instanceID
	current.State = "applied"

	s.instanceMetadataOptions[instanceID] = cloneStage128ModifiedInstanceMetadataOptions(current)
	return current, nil
}

func (s *Service) ModifyInstanceNetworkPerformanceOptions(instanceID, bandwidthWeighting string) (string, error) {
	instanceID = strings.TrimSpace(instanceID)
	bandwidthWeighting = strings.ToLower(strings.TrimSpace(bandwidthWeighting))
	if instanceID == "" || !stage128IsAllowedValue(bandwidthWeighting, "default", "vpc-1", "ebs-1") {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return "", ErrNotFound
	}
	s.instanceNetworkPerformanceOptions[instanceID] = bandwidthWeighting
	return bandwidthWeighting, nil
}

func (s *Service) ModifyInstancePlacement(
	instanceID string,
	affinity, groupID, groupName, hostID, hostResourceGroupARN, tenancy *string,
	partitionNumber *int32,
) (bool, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return false, ErrInvalidParameter
	}

	affinityValue := strings.ToLower(strings.TrimSpace(derefString(affinity)))
	tenancyValue := strings.ToLower(strings.TrimSpace(derefString(tenancy)))
	if affinity != nil && !stage128IsAllowedValue(affinityValue, "default", "host") {
		return false, ErrInvalidParameter
	}
	if tenancy != nil && !stage128IsAllowedValue(tenancyValue, "default", "dedicated", "host") {
		return false, ErrInvalidParameter
	}
	if partitionNumber != nil && *partitionNumber < 0 {
		return false, ErrInvalidParameter
	}
	if affinity == nil && groupID == nil && groupName == nil && hostID == nil && hostResourceGroupARN == nil && tenancy == nil && partitionNumber == nil {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return false, ErrNotFound
	}

	hostIDValue := strings.TrimSpace(derefString(hostID))
	if hostID != nil && hostIDValue != "" && s.dedicatedHosts[hostIDValue] == nil {
		return false, ErrNotFound
	}

	current := s.instancePlacementOptions[instanceID]
	if affinity != nil {
		current.Affinity = affinityValue
	}
	if groupID != nil {
		current.GroupID = strings.TrimSpace(*groupID)
	}
	if groupName != nil {
		current.GroupName = strings.TrimSpace(*groupName)
	}
	if hostID != nil {
		current.HostID = hostIDValue
	}
	if hostResourceGroupARN != nil {
		current.HostResourceGroupARN = strings.TrimSpace(*hostResourceGroupARN)
	}
	if tenancy != nil {
		current.Tenancy = tenancyValue
	}
	if partitionNumber != nil {
		current.PartitionNumber = cloneInt32Pointer(partitionNumber)
	}
	s.instancePlacementOptions[instanceID] = current

	return true, nil
}

func stage128IsAllowedValue(value string, allowed ...string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return false
	}
	for _, candidate := range allowed {
		if value == strings.TrimSpace(strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

func stage128ResolveMetadataDefaultValue(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "no-preference" {
		return fallback
	}
	return value
}

func cloneStage128ModifiedInstanceMetadataOptions(in ModifiedInstanceMetadataOptions) ModifiedInstanceMetadataOptions {
	out := in
	out.HttpPutResponseHopLimit = cloneInt32Pointer(in.HttpPutResponseHopLimit)
	return out
}

func cloneStage128ModifiedInstanceEvent(in ModifiedInstanceEvent) ModifiedInstanceEvent {
	return ModifiedInstanceEvent{
		Code:              in.Code,
		Description:       in.Description,
		InstanceEventID:   in.InstanceEventID,
		NotAfter:          cloneTimePointer(in.NotAfter),
		NotBefore:         cloneTimePointer(in.NotBefore),
		NotBeforeDeadline: cloneTimePointer(in.NotBeforeDeadline),
	}
}
