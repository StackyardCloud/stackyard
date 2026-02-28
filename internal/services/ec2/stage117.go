package ec2

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type HostDescription struct {
	AllocationTime     time.Time
	AutoPlacement      string
	AvailabilityZone   string
	AvailabilityZoneID string
	Cores              int32
	HostID             string
	HostMaintenance    string
	HostRecovery       string
	InstanceFamily     string
	InstanceType       string
	Instances          []HostInstanceDescription
	OwnerID            string
	Sockets            int32
	State              string
	Tags               map[string]string
	TotalVCpus         int32
}

type HostInstanceDescription struct {
	InstanceID   string
	InstanceType string
	OwnerID      string
}

type ImportImageTaskDescription struct {
	Description     string
	ImageID         string
	ImportTaskID    string
	Progress        string
	SnapshotDetails []ImportImageSnapshotDetail
	Status          string
	StatusMessage   string
	Tags            map[string]string
}

type ImportImageSnapshotDetail struct {
	Progress   string
	SnapshotID string
	Status     string
}

type ImportSnapshotTaskDescription struct {
	Description        string
	ImportTaskID       string
	SnapshotTaskDetail ImportSnapshotTaskDetail
	Tags               map[string]string
}

type ImportSnapshotTaskDetail struct {
	Description   string
	Progress      string
	SnapshotID    string
	Status        string
	StatusMessage string
}

type InstanceCreditSpecificationDescription struct {
	CpuCredits string
	InstanceID string
}

type InstanceEventWindowDescription struct {
	AssociationDedicatedHostIDs []string
	AssociationInstanceIDs      []string
	AssociationInstanceTags     []Tag
	CronExpression              string
	InstanceEventWindowID       string
	Name                        string
	State                       string
	Tags                        map[string]string
	TimeRanges                  []InstanceEventWindowTimeRange
}

type ImageMetadataDescription struct {
	CreationDate    string
	DeprecationTime string
	ImageAllowed    bool
	ImageID         string
	ImageOwnerAlias string
	IsPublic        bool
	Name            string
	OwnerID         string
	State           string
}

type InstanceImageMetadataDescription struct {
	AvailabilityZone string
	ImageMetadata    ImageMetadataDescription
	InstanceID       string
	InstanceType     string
	LaunchTime       time.Time
	OwnerID          string
	StateCode        int32
	StateName        string
	Tags             map[string]string
	ZoneID           string
}

type InstanceTopologyDescription struct {
	AvailabilityZone string
	CapacityBlockID  string
	GroupName        string
	InstanceID       string
	InstanceType     string
	NetworkNodes     []string
	ZoneID           string
}

type InstanceTypeOfferingDescription struct {
	InstanceType string
	Location     string
	LocationType string
}

func (s *Service) DescribeHosts(hostIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]HostDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedHostIDs := dedupeTrimmedStrings(hostIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	hostIDFilterSet := toStringSet(standardFilters["host-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	availabilityZoneIDFilterSet := toLowerStringSet(standardFilters["availability-zone-id"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	instanceFamilyFilterSet := toLowerStringSet(standardFilters["instance-family"])
	autoPlacementFilterSet := toLowerStringSet(standardFilters["auto-placement"])
	hostRecoveryFilterSet := toLowerStringSet(standardFilters["host-recovery"])
	hostMaintenanceFilterSet := toLowerStringSet(standardFilters["host-maintenance"])

	now := time.Now().UTC()

	s.mu.Lock()
	candidateHostIDs := append([]string(nil), requestedHostIDs...)
	if len(candidateHostIDs) == 0 {
		candidateHostIDs = make([]string, 0, len(s.dedicatedHosts))
		for hostID := range s.dedicatedHosts {
			candidateHostIDs = append(candidateHostIDs, hostID)
		}
		sort.Strings(candidateHostIDs)
	}

	out := make([]HostDescription, 0, len(candidateHostIDs))
	for _, hostID := range candidateHostIDs {
		host := s.dedicatedHosts[hostID]
		if host == nil {
			continue
		}
		instanceType := strings.TrimSpace(host.InstanceType)
		instanceFamily := strings.TrimSpace(host.InstanceFamily)
		if instanceFamily == "" {
			instanceFamily = stage116InstanceFamilyFromType(instanceType)
		}
		if instanceFamily == "" {
			instanceFamily = "m5"
		}
		if instanceType == "" {
			instanceType = instanceFamily + ".large"
		}

		item := HostDescription{
			AllocationTime:     now,
			AutoPlacement:      firstNonEmptyString(host.AutoPlacement, "off"),
			AvailabilityZone:   firstNonEmptyString(host.AvailabilityZone, "us-east-1a"),
			AvailabilityZoneID: stage117ZoneIDForAvailabilityZoneLocked(s.availabilityZones, host.AvailabilityZone),
			Cores:              2,
			HostID:             hostID,
			HostMaintenance:    firstNonEmptyString(host.HostMaintenance, "off"),
			HostRecovery:       firstNonEmptyString(host.HostRecovery, "off"),
			InstanceFamily:     instanceFamily,
			InstanceType:       instanceType,
			Instances:          []HostInstanceDescription{},
			OwnerID:            DefaultAccountID,
			Sockets:            1,
			State:              "available",
			Tags:               map[string]string{},
			TotalVCpus:         2,
		}

		if len(hostIDFilterSet) > 0 {
			if _, ok := hostIDFilterSet[item.HostID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
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
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}
		if len(instanceFamilyFilterSet) > 0 {
			if _, ok := instanceFamilyFilterSet[strings.ToLower(item.InstanceFamily)]; !ok {
				continue
			}
		}
		if len(autoPlacementFilterSet) > 0 {
			if _, ok := autoPlacementFilterSet[strings.ToLower(item.AutoPlacement)]; !ok {
				continue
			}
		}
		if len(hostRecoveryFilterSet) > 0 {
			if _, ok := hostRecoveryFilterSet[strings.ToLower(item.HostRecovery)]; !ok {
				continue
			}
		}
		if len(hostMaintenanceFilterSet) > 0 {
			if _, ok := hostMaintenanceFilterSet[strings.ToLower(item.HostMaintenance)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage117HostDescription(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]HostDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeImportImageTasks(importTaskIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ImportImageTaskDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedImportTaskIDs := dedupeTrimmedStrings(importTaskIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	importTaskIDFilterSet := toStringSet(standardFilters["import-task-id"])
	imageIDFilterSet := toStringSet(standardFilters["image-id"])
	statusFilterSet := toLowerStringSet(standardFilters["status"])

	s.mu.Lock()
	candidateImportTaskIDs := append([]string(nil), requestedImportTaskIDs...)
	if len(candidateImportTaskIDs) == 0 {
		candidateImportTaskIDs = make([]string, 0, len(s.importTaskStates))
		for importTaskID := range s.importTaskStates {
			candidateImportTaskIDs = append(candidateImportTaskIDs, importTaskID)
		}
		sort.Strings(candidateImportTaskIDs)
	}

	out := make([]ImportImageTaskDescription, 0, len(candidateImportTaskIDs))
	for _, importTaskID := range candidateImportTaskIDs {
		importTaskID = strings.TrimSpace(importTaskID)
		if importTaskID == "" {
			continue
		}
		status := strings.TrimSpace(s.importTaskStates[importTaskID])
		if status == "" {
			status = "active"
		}
		statusMessage := strings.TrimSpace(s.importTaskCancelReasons[importTaskID])
		if statusMessage == "" {
			statusMessage = "import task " + status
		}

		progress := "0"
		switch strings.ToLower(status) {
		case "cancelled", "completed":
			progress = "100"
		}

		item := ImportImageTaskDescription{
			Description:  "import image task",
			ImageID:      stage117ImageIDFromImportTaskID(importTaskID),
			ImportTaskID: importTaskID,
			Progress:     progress,
			SnapshotDetails: []ImportImageSnapshotDetail{
				{
					Progress:   progress,
					SnapshotID: stage117SnapshotIDFromImportTaskID(importTaskID),
					Status:     status,
				},
			},
			Status:        status,
			StatusMessage: statusMessage,
			Tags:          map[string]string{},
		}

		if len(importTaskIDFilterSet) > 0 {
			if _, ok := importTaskIDFilterSet[item.ImportTaskID]; !ok {
				continue
			}
		}
		if len(imageIDFilterSet) > 0 {
			if _, ok := imageIDFilterSet[item.ImageID]; !ok {
				continue
			}
		}
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(item.Status)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage117ImportImageTask(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ImportImageTaskDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeImportSnapshotTasks(importTaskIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ImportSnapshotTaskDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedImportTaskIDs := dedupeTrimmedStrings(importTaskIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	importTaskIDFilterSet := toStringSet(standardFilters["import-task-id"])
	statusFilterSet := toLowerStringSet(standardFilters["status"])
	snapshotIDFilterSet := toStringSet(standardFilters["snapshot-id"])

	s.mu.Lock()
	candidateImportTaskIDs := append([]string(nil), requestedImportTaskIDs...)
	if len(candidateImportTaskIDs) == 0 {
		candidateImportTaskIDs = make([]string, 0, len(s.importTaskStates))
		for importTaskID := range s.importTaskStates {
			candidateImportTaskIDs = append(candidateImportTaskIDs, importTaskID)
		}
		sort.Strings(candidateImportTaskIDs)
	}

	out := make([]ImportSnapshotTaskDescription, 0, len(candidateImportTaskIDs))
	for _, importTaskID := range candidateImportTaskIDs {
		importTaskID = strings.TrimSpace(importTaskID)
		if importTaskID == "" {
			continue
		}
		status := strings.TrimSpace(s.importTaskStates[importTaskID])
		if status == "" {
			status = "active"
		}
		statusMessage := strings.TrimSpace(s.importTaskCancelReasons[importTaskID])
		if statusMessage == "" {
			statusMessage = "import task " + status
		}

		progress := "0"
		switch strings.ToLower(status) {
		case "cancelled", "completed":
			progress = "100"
		}

		snapshotID := stage117SnapshotIDFromImportTaskID(importTaskID)
		item := ImportSnapshotTaskDescription{
			Description:  "import snapshot task",
			ImportTaskID: importTaskID,
			SnapshotTaskDetail: ImportSnapshotTaskDetail{
				Description:   "import snapshot detail",
				Progress:      progress,
				SnapshotID:    snapshotID,
				Status:        status,
				StatusMessage: statusMessage,
			},
			Tags: map[string]string{},
		}

		if len(importTaskIDFilterSet) > 0 {
			if _, ok := importTaskIDFilterSet[item.ImportTaskID]; !ok {
				continue
			}
		}
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(item.SnapshotTaskDetail.Status)]; !ok {
				continue
			}
		}
		if len(snapshotIDFilterSet) > 0 {
			if _, ok := snapshotIDFilterSet[item.SnapshotTaskDetail.SnapshotID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage117ImportSnapshotTask(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ImportSnapshotTaskDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeInstanceConnectEndpoints(instanceConnectEndpointIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceConnectEndpoint, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(instanceConnectEndpointIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	endpointIDFilterSet := toStringSet(standardFilters["instance-connect-endpoint-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	subnetIDFilterSet := toStringSet(standardFilters["subnet-id"])
	vpcIDFilterSet := toStringSet(standardFilters["vpc-id"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.instanceConnectEndpoints))
		for endpointID := range s.instanceConnectEndpoints {
			candidateIDs = append(candidateIDs, endpointID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]InstanceConnectEndpoint, 0, len(candidateIDs))
	for _, endpointID := range candidateIDs {
		endpoint := s.instanceConnectEndpoints[endpointID]
		if endpoint == nil {
			continue
		}
		item := cloneStage107InstanceConnectEndpoint(endpoint)

		if len(endpointIDFilterSet) > 0 {
			if _, ok := endpointIDFilterSet[item.InstanceConnectEndpointID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if len(subnetIDFilterSet) > 0 {
			if _, ok := subnetIDFilterSet[item.SubnetID]; !ok {
				continue
			}
		}
		if len(vpcIDFilterSet) > 0 {
			if _, ok := vpcIDFilterSet[item.VpcID]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceConnectEndpoint(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeInstanceCreditSpecifications(instanceIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceCreditSpecificationDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedInstanceIDs := dedupeTrimmedStrings(instanceIDs)
	standardFilters, _, _ := splitEC2Filters(filters)
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	cpuCreditsFilterSet := toLowerStringSet(standardFilters["cpu-credits"])

	s.mu.Lock()
	candidateInstanceIDs := append([]string(nil), requestedInstanceIDs...)
	if len(candidateInstanceIDs) == 0 {
		candidateInstanceIDs = make([]string, 0, len(s.instances))
		for instanceID := range s.instances {
			candidateInstanceIDs = append(candidateInstanceIDs, instanceID)
		}
		sort.Strings(candidateInstanceIDs)
	}

	out := make([]InstanceCreditSpecificationDescription, 0, len(candidateInstanceIDs))
	for _, instanceID := range candidateInstanceIDs {
		instance := s.instances[instanceID]
		if instance == nil {
			continue
		}

		item := InstanceCreditSpecificationDescription{
			CpuCredits: "standard",
			InstanceID: instanceID,
		}
		if cpuCredits := strings.ToLower(strings.TrimSpace(s.instanceCreditSpecifications[instanceID])); cpuCredits != "" {
			item.CpuCredits = cpuCredits
		}

		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if len(cpuCreditsFilterSet) > 0 {
			if _, ok := cpuCreditsFilterSet[strings.ToLower(item.CpuCredits)]; !ok {
				continue
			}
		}

		out = append(out, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceCreditSpecificationDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeInstanceEventNotificationAttributes() InstanceTagNotificationAttribute {
	includeAll := false

	s.mu.Lock()
	tagKeySet := map[string]struct{}{}
	for _, instance := range s.instances {
		if instance == nil {
			continue
		}
		for tagKey := range instance.Tags {
			tagKey = strings.TrimSpace(tagKey)
			if tagKey == "" {
				continue
			}
			tagKeySet[tagKey] = struct{}{}
		}
	}
	tagKeys := make([]string, 0, len(tagKeySet))
	for tagKey := range tagKeySet {
		tagKeys = append(tagKeys, tagKey)
	}
	sort.Strings(tagKeys)
	s.mu.Unlock()

	return InstanceTagNotificationAttribute{
		IncludeAllTagsOfInstance: &includeAll,
		InstanceTagKeys:          tagKeys,
	}
}

func (s *Service) DescribeInstanceEventWindows(instanceEventWindowIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceEventWindowDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(instanceEventWindowIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	windowIDFilterSet := toStringSet(standardFilters["instance-event-window-id"])
	nameFilterSet := toLowerStringSet(standardFilters["name"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.instanceEventWindows))
		for windowID := range s.instanceEventWindows {
			candidateIDs = append(candidateIDs, windowID)
		}
		sort.Strings(candidateIDs)
	}

	out := make([]InstanceEventWindowDescription, 0, len(candidateIDs))
	for _, windowID := range candidateIDs {
		window := s.instanceEventWindows[windowID]
		if window == nil {
			continue
		}
		association := s.instanceEventWindowAssociations[windowID]

		item := InstanceEventWindowDescription{
			AssociationDedicatedHostIDs: append([]string(nil), association.DedicatedHostIDs...),
			AssociationInstanceIDs:      append([]string(nil), association.InstanceIDs...),
			AssociationInstanceTags:     cloneEC2Tags(association.InstanceTags),
			CronExpression:              window.CronExpression,
			InstanceEventWindowID:       window.InstanceEventWindowID,
			Name:                        window.Name,
			State:                       window.State,
			Tags:                        cloneStringMap(window.Tags),
			TimeRanges:                  cloneStage107EventWindowTimeRanges(window.TimeRanges),
		}

		if len(windowIDFilterSet) > 0 {
			if _, ok := windowIDFilterSet[item.InstanceEventWindowID]; !ok {
				continue
			}
		}
		if len(nameFilterSet) > 0 {
			if _, ok := nameFilterSet[strings.ToLower(item.Name)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage117InstanceEventWindow(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceEventWindowDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeInstanceImageMetadata(instanceIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceImageMetadataDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedInstanceIDs := dedupeTrimmedStrings(instanceIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	imageIDFilterSet := toStringSet(standardFilters["image-id"])
	stateNameFilterSet := toLowerStringSet(standardFilters["instance-state-name"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])

	s.mu.Lock()
	candidateInstanceIDs := append([]string(nil), requestedInstanceIDs...)
	if len(candidateInstanceIDs) == 0 {
		candidateInstanceIDs = make([]string, 0, len(s.instances))
		for instanceID := range s.instances {
			candidateInstanceIDs = append(candidateInstanceIDs, instanceID)
		}
		sort.Strings(candidateInstanceIDs)
	}

	out := make([]InstanceImageMetadataDescription, 0, len(candidateInstanceIDs))
	for _, instanceID := range candidateInstanceIDs {
		instance := s.instances[instanceID]
		if instance == nil {
			continue
		}
		image := s.images[instance.ImageID]

		imageMetadata := ImageMetadataDescription{
			CreationDate:    stage117TimestampOrNow(time.Time{}, time.Now().UTC()),
			DeprecationTime: "",
			ImageAllowed:    true,
			ImageID:         instance.ImageID,
			ImageOwnerAlias: "amazon",
			IsPublic:        false,
			Name:            instance.ImageID,
			OwnerID:         DefaultAccountID,
			State:           "available",
		}
		if image != nil {
			imageMetadata.CreationDate = stage117TimestampOrNow(image.CreationDate, time.Now().UTC())
			if image.DeprecationTime != nil && !image.DeprecationTime.IsZero() {
				imageMetadata.DeprecationTime = image.DeprecationTime.UTC().Format(time.RFC3339)
			}
			imageMetadata.Name = firstNonEmptyString(image.Name, imageMetadata.Name)
			imageMetadata.OwnerID = firstNonEmptyString(image.OwnerID, imageMetadata.OwnerID)
			imageMetadata.State = firstNonEmptyString(image.State, imageMetadata.State)
		}

		item := InstanceImageMetadataDescription{
			AvailabilityZone: firstNonEmptyString(instance.AvailabilityZone, "us-east-1a"),
			ImageMetadata:    imageMetadata,
			InstanceID:       instance.ID,
			InstanceType:     instance.InstanceType,
			LaunchTime:       instance.LaunchTime,
			OwnerID:          DefaultAccountID,
			StateCode:        instance.StateCode,
			StateName:        firstNonEmptyString(instance.StateName, "running"),
			Tags:             cloneStringMap(instance.Tags),
			ZoneID:           stage117ZoneIDForAvailabilityZoneLocked(s.availabilityZones, instance.AvailabilityZone),
		}

		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if len(imageIDFilterSet) > 0 {
			if _, ok := imageIDFilterSet[item.ImageMetadata.ImageID]; !ok {
				continue
			}
		}
		if len(stateNameFilterSet) > 0 {
			if _, ok := stateNameFilterSet[strings.ToLower(item.StateName)]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		out = append(out, cloneStage117InstanceImageMetadata(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceImageMetadataDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeInstanceTopology(instanceIDs, groupNames []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceTopologyDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedInstanceIDs := dedupeTrimmedStrings(instanceIDs)
	requestedGroupNames := dedupeTrimmedStrings(groupNames)
	requestedGroupNameSet := toLowerStringSet(requestedGroupNames)

	standardFilters, _, _ := splitEC2Filters(filters)
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	groupNameFilterSet := toLowerStringSet(standardFilters["group-name"])
	availabilityZoneFilterSet := toLowerStringSet(standardFilters["availability-zone"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])

	s.mu.Lock()
	candidateInstanceIDs := append([]string(nil), requestedInstanceIDs...)
	if len(candidateInstanceIDs) == 0 {
		candidateInstanceIDs = make([]string, 0, len(s.instances))
		for instanceID := range s.instances {
			candidateInstanceIDs = append(candidateInstanceIDs, instanceID)
		}
		sort.Strings(candidateInstanceIDs)
	}

	out := make([]InstanceTopologyDescription, 0, len(candidateInstanceIDs))
	for _, instanceID := range candidateInstanceIDs {
		instance := s.instances[instanceID]
		if instance == nil {
			continue
		}

		groupName := strings.TrimSpace(instance.Tags["placement-group"])
		if groupName == "" {
			groupName = "default"
		}

		item := InstanceTopologyDescription{
			AvailabilityZone: firstNonEmptyString(instance.AvailabilityZone, "us-east-1a"),
			CapacityBlockID:  strings.TrimSpace(instance.Tags["capacity-block-id"]),
			GroupName:        groupName,
			InstanceID:       instance.ID,
			InstanceType:     instance.InstanceType,
			NetworkNodes:     []string{stage117NetworkNodeID(instance.ID)},
			ZoneID:           stage117ZoneIDForAvailabilityZoneLocked(s.availabilityZones, instance.AvailabilityZone),
		}

		if len(requestedGroupNameSet) > 0 {
			if _, ok := requestedGroupNameSet[strings.ToLower(item.GroupName)]; !ok {
				continue
			}
		}
		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if len(groupNameFilterSet) > 0 {
			if _, ok := groupNameFilterSet[strings.ToLower(item.GroupName)]; !ok {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[strings.ToLower(item.AvailabilityZone)]; !ok {
				continue
			}
		}
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}

		out = append(out, cloneStage117InstanceTopology(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceTopologyDescription(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeInstanceTypeOfferings(locationType string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceTypeOfferingDescription, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	locationType = strings.ToLower(strings.TrimSpace(locationType))
	if locationType == "" {
		locationType = "region"
	}
	switch locationType {
	case "region", "availability-zone", "availability-zone-id", "outpost":
	default:
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	locationFilterSet := toLowerStringSet(standardFilters["location"])

	s.mu.Lock()
	instanceTypeSet := map[string]struct{}{}
	for _, instance := range s.instances {
		if instance == nil {
			continue
		}
		instanceType := strings.TrimSpace(instance.InstanceType)
		if instanceType == "" {
			continue
		}
		instanceTypeSet[instanceType] = struct{}{}
	}
	for _, host := range s.dedicatedHosts {
		if host == nil {
			continue
		}
		instanceType := strings.TrimSpace(host.InstanceType)
		if instanceType == "" {
			instanceFamily := strings.TrimSpace(host.InstanceFamily)
			if instanceFamily != "" {
				instanceType = instanceFamily + ".large"
			}
		}
		if instanceType == "" {
			continue
		}
		instanceTypeSet[instanceType] = struct{}{}
	}
	if len(instanceTypeSet) == 0 {
		instanceTypeSet["c6i.large"] = struct{}{}
		instanceTypeSet["m5.large"] = struct{}{}
		instanceTypeSet["t3.micro"] = struct{}{}
	}

	instanceTypes := make([]string, 0, len(instanceTypeSet))
	for instanceType := range instanceTypeSet {
		instanceTypes = append(instanceTypes, instanceType)
	}
	sort.Strings(instanceTypes)

	locations := stage117LocationsForTypeLocked(locationType, s.availabilityZones)
	out := make([]InstanceTypeOfferingDescription, 0, len(instanceTypes)*len(locations))
	for _, instanceType := range instanceTypes {
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(instanceType)]; !ok {
				continue
			}
		}
		for _, location := range locations {
			if len(locationFilterSet) > 0 {
				if _, ok := locationFilterSet[strings.ToLower(location)]; !ok {
					continue
				}
			}
			out = append(out, InstanceTypeOfferingDescription{
				InstanceType: instanceType,
				Location:     location,
				LocationType: locationType,
			})
		}
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceTypeOfferingDescription(nil), out[start:end]...), outputToken, nil
}

func stage117ZoneIDForAvailabilityZoneLocked(availabilityZones []AvailabilityZone, zoneName string) string {
	zoneName = strings.TrimSpace(zoneName)
	for _, zone := range availabilityZones {
		if strings.EqualFold(zone.Name, zoneName) {
			return zone.ZoneID
		}
	}
	return ""
}

func stage117ImageIDFromImportTaskID(importTaskID string) string {
	importTaskID = strings.TrimSpace(importTaskID)
	suffix := strings.TrimPrefix(importTaskID, "import-ami-")
	if suffix == importTaskID {
		suffix = strings.TrimPrefix(importTaskID, "import-")
	}
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return "ami-00000000117"
	}
	return "ami-" + suffix
}

func stage117SnapshotIDFromImportTaskID(importTaskID string) string {
	importTaskID = strings.TrimSpace(importTaskID)
	suffix := strings.TrimPrefix(importTaskID, "import-snap-")
	if suffix == importTaskID {
		suffix = strings.TrimPrefix(importTaskID, "import-")
	}
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return "snap-00000000117"
	}
	return "snap-" + suffix
}

func stage117TimestampOrNow(value time.Time, fallback time.Time) string {
	if value.IsZero() {
		value = fallback
	}
	return value.UTC().Format(time.RFC3339)
}

func stage117NetworkNodeID(instanceID string) string {
	instanceID = strings.TrimSpace(strings.TrimPrefix(instanceID, "i-"))
	if instanceID == "" {
		return "node-00000000"
	}
	if len(instanceID) > 8 {
		instanceID = instanceID[len(instanceID)-8:]
	}
	return fmt.Sprintf("node-%s", instanceID)
}

func stage117LocationsForTypeLocked(locationType string, availabilityZones []AvailabilityZone) []string {
	switch locationType {
	case "region":
		return []string{DefaultRegion}
	case "outpost":
		return []string{"op-00000000000000117"}
	case "availability-zone":
		out := make([]string, 0, len(availabilityZones))
		seen := map[string]struct{}{}
		for _, zone := range availabilityZones {
			name := strings.TrimSpace(zone.Name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
		sort.Strings(out)
		return out
	case "availability-zone-id":
		out := make([]string, 0, len(availabilityZones))
		seen := map[string]struct{}{}
		for _, zone := range availabilityZones {
			zoneID := strings.TrimSpace(zone.ZoneID)
			if zoneID == "" {
				continue
			}
			if _, ok := seen[zoneID]; ok {
				continue
			}
			seen[zoneID] = struct{}{}
			out = append(out, zoneID)
		}
		sort.Strings(out)
		return out
	default:
		return []string{DefaultRegion}
	}
}

func cloneStage117HostDescription(in HostDescription) HostDescription {
	out := in
	out.Instances = append([]HostInstanceDescription(nil), in.Instances...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStage117ImportImageTask(in ImportImageTaskDescription) ImportImageTaskDescription {
	out := in
	out.SnapshotDetails = append([]ImportImageSnapshotDetail(nil), in.SnapshotDetails...)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStage117ImportSnapshotTask(in ImportSnapshotTaskDescription) ImportSnapshotTaskDescription {
	out := in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStage117InstanceEventWindow(in InstanceEventWindowDescription) InstanceEventWindowDescription {
	out := in
	out.AssociationDedicatedHostIDs = append([]string(nil), in.AssociationDedicatedHostIDs...)
	out.AssociationInstanceIDs = append([]string(nil), in.AssociationInstanceIDs...)
	out.AssociationInstanceTags = cloneEC2Tags(in.AssociationInstanceTags)
	out.Tags = cloneStringMap(in.Tags)
	out.TimeRanges = cloneStage107EventWindowTimeRanges(in.TimeRanges)
	return out
}

func cloneStage117InstanceImageMetadata(in InstanceImageMetadataDescription) InstanceImageMetadataDescription {
	out := in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStage117InstanceTopology(in InstanceTopologyDescription) InstanceTopologyDescription {
	out := in
	out.NetworkNodes = append([]string(nil), in.NetworkNodes...)
	return out
}
