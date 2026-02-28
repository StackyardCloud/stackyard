package ec2

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type SpotFleetRequestConfig struct {
	ActivityStatus        string
	CreateTime            time.Time
	SpotFleetRequestID    string
	SpotFleetRequestState string
	Tags                  map[string]string
}

type SpotInstanceRequestStatus struct {
	Code       string
	Message    string
	UpdateTime time.Time
}

type SpotInstanceRequest struct {
	AvailabilityZoneGroup string
	CreateTime            time.Time
	InstanceID            string
	LaunchGroup           string
	ProductDescription    string
	SpotInstanceRequestID string
	SpotPrice             string
	State                 string
	Status                SpotInstanceRequestStatus
	Tags                  map[string]string
	Type                  string
}

type SpotPriceHistoryItem struct {
	AvailabilityZone   string
	AvailabilityZoneID string
	InstanceType       string
	ProductDescription string
	SpotPrice          string
	Timestamp          time.Time
}

type StoreImageTaskResult struct {
	AmiID                  string
	Bucket                 string
	ProgressPercentage     int32
	S3ObjectKey            string
	StoreTaskFailureReason string
	StoreTaskState         string
	TaskStartTime          time.Time
}

func (s *Service) DescribeSpotFleetRequestHistory(
	spotFleetRequestID string,
	startTime time.Time,
	eventType string,
	maxResults *int32,
	nextToken *string,
) (string, []FleetHistoryRecord, *time.Time, *string, error) {
	spotFleetRequestID = strings.TrimSpace(spotFleetRequestID)
	if spotFleetRequestID == "" || startTime.IsZero() {
		return "", nil, nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return "", nil, nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return "", nil, nil, nil, ErrInvalidParameter
	}

	normalizedStart := startTime.UTC()
	requestedEvent := strings.TrimSpace(eventType)

	s.mu.Lock()
	state := strings.TrimSpace(s.spotFleetRequestStates[spotFleetRequestID])
	if state == "" {
		state = "active"
		s.spotFleetRequestStates[spotFleetRequestID] = state
	}
	s.mu.Unlock()

	records := []FleetHistoryRecord{
		{
			EventDescription: "Spot Fleet request submitted",
			EventSubType:     "submitted",
			EventType:        "fleetRequestChange",
			Timestamp:        normalizedStart,
		},
		{
			EventDescription: "Spot instance request fulfilled",
			EventSubType:     "fulfilled",
			EventType:        "instanceChange",
			InstanceID:       "i-" + stage121NormalizeSpotFleetRequestID(spotFleetRequestID) + "1",
			Timestamp:        normalizedStart.Add(1 * time.Second),
		},
	}
	if strings.Contains(strings.ToLower(state), "cancel") {
		records = append(records, FleetHistoryRecord{
			EventDescription: "Spot Fleet request cancelled",
			EventSubType:     "cancelled",
			EventType:        "fleetRequestChange",
			Timestamp:        normalizedStart.Add(2 * time.Second),
		})
	}

	filtered := make([]FleetHistoryRecord, 0, len(records))
	for _, record := range records {
		if record.Timestamp.Before(normalizedStart) {
			continue
		}
		if !stage122SpotFleetHistoryEventMatches(record.EventType, requestedEvent) {
			continue
		}
		filtered = append(filtered, record)
	}

	start, end, outputToken, err := ec2PageWindow(len(filtered), start, maxResults)
	if err != nil {
		return "", nil, nil, nil, ErrInvalidParameter
	}
	page := append([]FleetHistoryRecord(nil), filtered[start:end]...)

	var lastEvaluatedTime *time.Time
	if outputToken == nil && len(page) > 0 {
		last := page[len(page)-1].Timestamp.UTC()
		lastEvaluatedTime = &last
	}

	return spotFleetRequestID, page, lastEvaluatedTime, outputToken, nil
}

func (s *Service) DescribeSpotFleetRequests(
	spotFleetRequestIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]SpotFleetRequestConfig, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(spotFleetRequestIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	spotFleetRequestIDFilterSet := toStringSet(standardFilters["spot-fleet-request-id"])
	spotFleetRequestStateFilterSet := toLowerStringSet(standardFilters["spot-fleet-request-state"])
	activityStatusFilterSet := toLowerStringSet(standardFilters["activity-status"])

	now := time.Now().UTC()

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.spotFleetRequestStates))
		for spotFleetRequestID := range s.spotFleetRequestStates {
			candidateIDs = append(candidateIDs, spotFleetRequestID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]SpotFleetRequestConfig, 0, len(candidateIDs))
	for _, spotFleetRequestID := range candidateIDs {
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[spotFleetRequestID]; !ok {
				continue
			}
		}
		if len(spotFleetRequestIDFilterSet) > 0 {
			if _, ok := spotFleetRequestIDFilterSet[spotFleetRequestID]; !ok {
				continue
			}
		}

		state := strings.TrimSpace(s.spotFleetRequestStates[spotFleetRequestID])
		if state == "" {
			state = "active"
		}
		if len(spotFleetRequestStateFilterSet) > 0 {
			if _, ok := spotFleetRequestStateFilterSet[strings.ToLower(state)]; !ok {
				continue
			}
		}

		activityStatus := stage122SpotFleetActivityStatusFromState(state)
		if len(activityStatusFilterSet) > 0 {
			if _, ok := activityStatusFilterSet[strings.ToLower(activityStatus)]; !ok {
				continue
			}
		}

		tags := map[string]string{}
		if !matchesTagFilters(tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, SpotFleetRequestConfig{
			ActivityStatus:        activityStatus,
			CreateTime:            now,
			SpotFleetRequestID:    spotFleetRequestID,
			SpotFleetRequestState: state,
			Tags:                  tags,
		})
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]SpotFleetRequestConfig(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeSpotInstanceRequests(
	spotInstanceRequestIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]SpotInstanceRequest, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(spotInstanceRequestIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	spotInstanceRequestIDFilterSet := toStringSet(standardFilters["spot-instance-request-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	productDescriptionFilterSet := toLowerStringSet(standardFilters["product-description"])
	launchGroupFilterSet := toStringSet(standardFilters["launch-group"])
	availabilityZoneGroupFilterSet := toStringSet(standardFilters["availability-zone-group"])
	statusCodeFilterSet := toStringSet(standardFilters["status-code"])
	typeFilterSet := toLowerStringSet(standardFilters["type"])

	now := time.Now().UTC()

	s.mu.Lock()
	candidateIDs := append([]string(nil), requestedIDs...)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.spotInstanceRequestStates))
		for spotInstanceRequestID := range s.spotInstanceRequestStates {
			candidateIDs = append(candidateIDs, spotInstanceRequestID)
		}
		sort.Strings(candidateIDs)
	}

	items := make([]SpotInstanceRequest, 0, len(candidateIDs))
	for _, spotInstanceRequestID := range candidateIDs {
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[spotInstanceRequestID]; !ok {
				continue
			}
		}
		if len(spotInstanceRequestIDFilterSet) > 0 {
			if _, ok := spotInstanceRequestIDFilterSet[spotInstanceRequestID]; !ok {
				continue
			}
		}

		state := strings.TrimSpace(s.spotInstanceRequestStates[spotInstanceRequestID])
		if state == "" {
			state = "open"
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(state)]; !ok {
				continue
			}
		}

		item := SpotInstanceRequest{
			AvailabilityZoneGroup: "",
			CreateTime:            now,
			InstanceID:            "i-" + strings.TrimPrefix(spotInstanceRequestID, "sir-"),
			LaunchGroup:           "",
			ProductDescription:    "Linux/UNIX",
			SpotInstanceRequestID: spotInstanceRequestID,
			SpotPrice:             "0.0123",
			State:                 state,
			Status: SpotInstanceRequestStatus{
				Code:       stage122SpotInstanceStatusCode(state),
				Message:    stage122SpotInstanceStatusMessage(state),
				UpdateTime: now,
			},
			Tags: map[string]string{},
			Type: "one-time",
		}

		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if len(productDescriptionFilterSet) > 0 {
			if _, ok := productDescriptionFilterSet[strings.ToLower(item.ProductDescription)]; !ok {
				continue
			}
		}
		if len(launchGroupFilterSet) > 0 {
			if _, ok := launchGroupFilterSet[item.LaunchGroup]; !ok {
				continue
			}
		}
		if len(availabilityZoneGroupFilterSet) > 0 {
			if _, ok := availabilityZoneGroupFilterSet[item.AvailabilityZoneGroup]; !ok {
				continue
			}
		}
		if len(statusCodeFilterSet) > 0 {
			if _, ok := statusCodeFilterSet[item.Status.Code]; !ok {
				continue
			}
		}
		if len(typeFilterSet) > 0 {
			if _, ok := typeFilterSet[strings.ToLower(item.Type)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]SpotInstanceRequest(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeSpotPriceHistory(
	availabilityZone string,
	availabilityZoneID string,
	endTime *time.Time,
	startTime *time.Time,
	instanceTypes []string,
	productDescriptions []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]SpotPriceHistoryItem, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	availabilityZone = strings.TrimSpace(availabilityZone)
	availabilityZoneID = strings.TrimSpace(availabilityZoneID)
	instanceTypes = dedupeTrimmedStrings(instanceTypes)
	productDescriptions = dedupeTrimmedStrings(productDescriptions)

	if len(instanceTypes) == 0 {
		instanceTypes = []string{"t3.micro"}
	}
	if len(productDescriptions) == 0 {
		productDescriptions = []string{"Linux/UNIX"}
	}
	if availabilityZone == "" {
		availabilityZone = "us-east-1a"
	}
	if availabilityZoneID == "" {
		availabilityZoneID = "use1-az1"
	}

	baseTimestamp := time.Now().UTC()
	if startTime != nil && !startTime.IsZero() {
		baseTimestamp = startTime.UTC()
	}
	if endTime != nil && !endTime.IsZero() && baseTimestamp.After(endTime.UTC()) {
		return []SpotPriceHistoryItem{}, nil, nil
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	availabilityZoneFilterSet := toStringSet(standardFilters["availability-zone"])
	availabilityZoneIDFilterSet := toStringSet(standardFilters["availability-zone-id"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	productDescriptionFilterSet := toLowerStringSet(standardFilters["product-description"])
	spotPriceFilterSet := toStringSet(standardFilters["spot-price"])
	timestampFilterSet := toStringSet(standardFilters["timestamp"])

	items := make([]SpotPriceHistoryItem, 0, len(instanceTypes)*len(productDescriptions))
	for index, instanceType := range instanceTypes {
		for _, productDescription := range productDescriptions {
			timestamp := baseTimestamp.Add(time.Duration(index) * time.Minute)
			if endTime != nil && !endTime.IsZero() && timestamp.After(endTime.UTC()) {
				continue
			}

			item := SpotPriceHistoryItem{
				AvailabilityZone:   availabilityZone,
				AvailabilityZoneID: availabilityZoneID,
				InstanceType:       instanceType,
				ProductDescription: productDescription,
				SpotPrice:          "0.0123",
				Timestamp:          timestamp.UTC(),
			}

			if len(availabilityZoneFilterSet) > 0 {
				if _, ok := availabilityZoneFilterSet[item.AvailabilityZone]; !ok {
					continue
				}
			}
			if len(availabilityZoneIDFilterSet) > 0 {
				if _, ok := availabilityZoneIDFilterSet[item.AvailabilityZoneID]; !ok {
					continue
				}
			}
			if len(instanceTypeFilterSet) > 0 {
				if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
					continue
				}
			}
			if len(productDescriptionFilterSet) > 0 {
				if _, ok := productDescriptionFilterSet[strings.ToLower(item.ProductDescription)]; !ok {
					continue
				}
			}
			if len(spotPriceFilterSet) > 0 {
				if _, ok := spotPriceFilterSet[item.SpotPrice]; !ok {
					continue
				}
			}
			if len(timestampFilterSet) > 0 {
				timestampValue := item.Timestamp.Format(time.RFC3339)
				if _, ok := timestampFilterSet[timestampValue]; !ok {
					continue
				}
			}

			items = append(items, item)
		}
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]SpotPriceHistoryItem(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeStoreImageTasks(
	imageIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]StoreImageTaskResult, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedImageIDSet := toStringSet(dedupeTrimmedStrings(imageIDs))
	standardFilters, _, _ := splitEC2Filters(filters)
	amiIDFilterSet := toStringSet(standardFilters["ami-id"])
	bucketFilterSet := toStringSet(standardFilters["bucket"])
	s3ObjectKeyFilterSet := toStringSet(standardFilters["s3-object-key"])
	storeTaskStateFilterSet := toLowerStringSet(standardFilters["store-task-state"])
	progressPercentageFilterSet := toStringSet(standardFilters["progress-percentage"])
	taskStartTimeFilterSet := toStringSet(standardFilters["task-start-time"])

	now := time.Now().UTC()

	s.mu.Lock()
	objectKeys := make([]string, 0, len(s.storeImageTasks))
	for objectKey := range s.storeImageTasks {
		objectKeys = append(objectKeys, objectKey)
	}
	sort.Strings(objectKeys)

	items := make([]StoreImageTaskResult, 0, len(objectKeys))
	for _, objectKey := range objectKeys {
		task := s.storeImageTasks[objectKey]
		if task == nil {
			continue
		}

		if len(requestedImageIDSet) > 0 {
			if _, ok := requestedImageIDSet[task.ImageID]; !ok {
				continue
			}
		}
		if len(amiIDFilterSet) > 0 {
			if _, ok := amiIDFilterSet[task.ImageID]; !ok {
				continue
			}
		}
		if len(bucketFilterSet) > 0 {
			if _, ok := bucketFilterSet[task.Bucket]; !ok {
				continue
			}
		}
		if len(s3ObjectKeyFilterSet) > 0 {
			if _, ok := s3ObjectKeyFilterSet[task.ObjectKey]; !ok {
				continue
			}
		}

		item := StoreImageTaskResult{
			AmiID:                  task.ImageID,
			Bucket:                 task.Bucket,
			ProgressPercentage:     100,
			S3ObjectKey:            task.ObjectKey,
			StoreTaskFailureReason: "",
			StoreTaskState:         "Completed",
			TaskStartTime:          now,
		}

		if len(storeTaskStateFilterSet) > 0 {
			if _, ok := storeTaskStateFilterSet[strings.ToLower(item.StoreTaskState)]; !ok {
				continue
			}
		}
		if len(progressPercentageFilterSet) > 0 {
			if _, ok := progressPercentageFilterSet[strconv.FormatInt(int64(item.ProgressPercentage), 10)]; !ok {
				continue
			}
		}
		if len(taskStartTimeFilterSet) > 0 {
			timestamp := item.TaskStartTime.UTC().Format(time.RFC3339)
			if _, ok := taskStartTimeFilterSet[timestamp]; !ok {
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
	return append([]StoreImageTaskResult(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeTrafficMirrorFilterRules(
	trafficMirrorFilterID string,
	trafficMirrorFilterRuleIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TrafficMirrorFilterRule, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	trafficMirrorFilterID = strings.TrimSpace(trafficMirrorFilterID)
	requestedIDSet := toStringSet(dedupeTrimmedStrings(trafficMirrorFilterRuleIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDFilterSet := toStringSet(standardFilters["traffic-mirror-filter-id"])
	ruleIDFilterSet := toStringSet(standardFilters["traffic-mirror-filter-rule-id"])
	trafficDirectionFilterSet := toLowerStringSet(standardFilters["traffic-direction"])
	ruleActionFilterSet := toLowerStringSet(standardFilters["rule-action"])
	destinationCidrBlockFilterSet := toStringSet(standardFilters["destination-cidr-block"])
	sourceCidrBlockFilterSet := toStringSet(standardFilters["source-cidr-block"])

	s.mu.Lock()
	ruleIDs := make([]string, 0, len(s.trafficMirrorFilterRules))
	for ruleID := range s.trafficMirrorFilterRules {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)

	items := make([]TrafficMirrorFilterRule, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		rule := s.trafficMirrorFilterRules[ruleID]
		if rule == nil {
			continue
		}
		if trafficMirrorFilterID != "" && rule.TrafficMirrorFilterID != trafficMirrorFilterID {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[ruleID]; !ok {
				continue
			}
		}
		if len(filterIDFilterSet) > 0 {
			if _, ok := filterIDFilterSet[rule.TrafficMirrorFilterID]; !ok {
				continue
			}
		}
		if len(ruleIDFilterSet) > 0 {
			if _, ok := ruleIDFilterSet[ruleID]; !ok {
				continue
			}
		}
		if len(trafficDirectionFilterSet) > 0 {
			if _, ok := trafficDirectionFilterSet[strings.ToLower(rule.TrafficDirection)]; !ok {
				continue
			}
		}
		if len(ruleActionFilterSet) > 0 {
			if _, ok := ruleActionFilterSet[strings.ToLower(rule.RuleAction)]; !ok {
				continue
			}
		}
		if len(destinationCidrBlockFilterSet) > 0 {
			if _, ok := destinationCidrBlockFilterSet[rule.DestinationCidrBlock]; !ok {
				continue
			}
		}
		if len(sourceCidrBlockFilterSet) > 0 {
			if _, ok := sourceCidrBlockFilterSet[rule.SourceCidrBlock]; !ok {
				continue
			}
		}
		if !matchesTagFilters(rule.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneStage110TrafficMirrorFilterRule(rule))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TrafficMirrorFilterRule(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeTrafficMirrorFilters(
	trafficMirrorFilterIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TrafficMirrorFilter, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(trafficMirrorFilterIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterIDFilterSet := toStringSet(standardFilters["traffic-mirror-filter-id"])

	s.mu.Lock()
	filterIDs := make([]string, 0, len(s.trafficMirrorFilters))
	for filterID := range s.trafficMirrorFilters {
		filterIDs = append(filterIDs, filterID)
	}
	sort.Strings(filterIDs)

	items := make([]TrafficMirrorFilter, 0, len(filterIDs))
	for _, filterID := range filterIDs {
		filter := s.trafficMirrorFilters[filterID]
		if filter == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[filterID]; !ok {
				continue
			}
		}
		if len(filterIDFilterSet) > 0 {
			if _, ok := filterIDFilterSet[filterID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(filter.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneStage110TrafficMirrorFilter(filter))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TrafficMirrorFilter(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeTrafficMirrorSessions(
	trafficMirrorSessionIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TrafficMirrorSession, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(trafficMirrorSessionIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	sessionIDFilterSet := toStringSet(standardFilters["traffic-mirror-session-id"])
	filterIDFilterSet := toStringSet(standardFilters["traffic-mirror-filter-id"])
	targetIDFilterSet := toStringSet(standardFilters["traffic-mirror-target-id"])
	networkInterfaceIDFilterSet := toStringSet(standardFilters["network-interface-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])

	s.mu.Lock()
	sessionIDs := make([]string, 0, len(s.trafficMirrorSessions))
	for sessionID := range s.trafficMirrorSessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	sort.Strings(sessionIDs)

	items := make([]TrafficMirrorSession, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session := s.trafficMirrorSessions[sessionID]
		if session == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[sessionID]; !ok {
				continue
			}
		}
		if len(sessionIDFilterSet) > 0 {
			if _, ok := sessionIDFilterSet[sessionID]; !ok {
				continue
			}
		}
		if len(filterIDFilterSet) > 0 {
			if _, ok := filterIDFilterSet[session.TrafficMirrorFilterID]; !ok {
				continue
			}
		}
		if len(targetIDFilterSet) > 0 {
			if _, ok := targetIDFilterSet[session.TrafficMirrorTargetID]; !ok {
				continue
			}
		}
		if len(networkInterfaceIDFilterSet) > 0 {
			if _, ok := networkInterfaceIDFilterSet[session.NetworkInterfaceID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[session.OwnerID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(session.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneStage110TrafficMirrorSession(session))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TrafficMirrorSession(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeTrafficMirrorTargets(
	trafficMirrorTargetIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TrafficMirrorTarget, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(trafficMirrorTargetIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	targetIDFilterSet := toStringSet(standardFilters["traffic-mirror-target-id"])
	typeFilterSet := toLowerStringSet(standardFilters["type"])
	networkInterfaceIDFilterSet := toStringSet(standardFilters["network-interface-id"])
	networkLoadBalancerARNFilterSet := toStringSet(standardFilters["network-load-balancer-arn"])
	gatewayLoadBalancerEndpointIDFilterSet := toStringSet(standardFilters["gateway-load-balancer-endpoint-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])

	s.mu.Lock()
	targetIDs := make([]string, 0, len(s.trafficMirrorTargets))
	for targetID := range s.trafficMirrorTargets {
		targetIDs = append(targetIDs, targetID)
	}
	sort.Strings(targetIDs)

	items := make([]TrafficMirrorTarget, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		target := s.trafficMirrorTargets[targetID]
		if target == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[targetID]; !ok {
				continue
			}
		}
		if len(targetIDFilterSet) > 0 {
			if _, ok := targetIDFilterSet[targetID]; !ok {
				continue
			}
		}
		if len(typeFilterSet) > 0 {
			if _, ok := typeFilterSet[strings.ToLower(target.Type)]; !ok {
				continue
			}
		}
		if len(networkInterfaceIDFilterSet) > 0 {
			if _, ok := networkInterfaceIDFilterSet[target.NetworkInterfaceID]; !ok {
				continue
			}
		}
		if len(networkLoadBalancerARNFilterSet) > 0 {
			if _, ok := networkLoadBalancerARNFilterSet[target.NetworkLoadBalancerARN]; !ok {
				continue
			}
		}
		if len(gatewayLoadBalancerEndpointIDFilterSet) > 0 {
			if _, ok := gatewayLoadBalancerEndpointIDFilterSet[target.GatewayLoadBalancerEndpointID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[target.OwnerID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(target.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneStage110TrafficMirrorTarget(target))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TrafficMirrorTarget(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeTrunkInterfaceAssociations(
	associationIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TrunkInterfaceAssociation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(associationIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	associationIDFilterSet := toStringSet(standardFilters["association-id"])
	branchInterfaceIDFilterSet := toStringSet(standardFilters["branch-interface-id"])
	trunkInterfaceIDFilterSet := toStringSet(standardFilters["trunk-interface-id"])
	interfaceProtocolFilterSet := toLowerStringSet(standardFilters["interface-protocol"])
	vlanIDFilterSet := toStringSet(standardFilters["vlan-id"])
	greKeyFilterSet := toStringSet(standardFilters["gre-key"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.trunkInterfaceAssociations))
	for associationID := range s.trunkInterfaceAssociations {
		ids = append(ids, associationID)
	}
	sort.Strings(ids)

	items := make([]TrunkInterfaceAssociation, 0, len(ids))
	for _, associationID := range ids {
		association := s.trunkInterfaceAssociations[associationID]
		if association == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[associationID]; !ok {
				continue
			}
		}
		if len(associationIDFilterSet) > 0 {
			if _, ok := associationIDFilterSet[associationID]; !ok {
				continue
			}
		}
		if len(branchInterfaceIDFilterSet) > 0 {
			if _, ok := branchInterfaceIDFilterSet[association.BranchInterfaceID]; !ok {
				continue
			}
		}
		if len(trunkInterfaceIDFilterSet) > 0 {
			if _, ok := trunkInterfaceIDFilterSet[association.TrunkInterfaceID]; !ok {
				continue
			}
		}
		if len(interfaceProtocolFilterSet) > 0 {
			if _, ok := interfaceProtocolFilterSet[strings.ToLower(association.InterfaceProtocol)]; !ok {
				continue
			}
		}
		if len(vlanIDFilterSet) > 0 {
			if association.VlanID == nil {
				continue
			}
			if _, ok := vlanIDFilterSet[strconv.FormatInt(int64(*association.VlanID), 10)]; !ok {
				continue
			}
		}
		if len(greKeyFilterSet) > 0 {
			if association.GreKey == nil {
				continue
			}
			if _, ok := greKeyFilterSet[strconv.FormatInt(int64(*association.GreKey), 10)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(association.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneTrunkInterfaceAssociation(association))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TrunkInterfaceAssociation(nil), items[start:end]...), outputToken, nil
}

func stage122SpotFleetHistoryEventMatches(recordEventType, requestedEventType string) bool {
	if strings.TrimSpace(requestedEventType) == "" {
		return true
	}
	return stage122SpotFleetEventTypeKey(recordEventType) == stage122SpotFleetEventTypeKey(requestedEventType)
}

func stage122SpotFleetEventTypeKey(eventType string) string {
	normalized := strings.ToLower(strings.TrimSpace(eventType))
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}

func stage122SpotFleetActivityStatusFromState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "cancelled", "cancelled_running", "cancelled_terminating":
		return "cancelled"
	case "failed":
		return "error"
	default:
		return "fulfilled"
	}
}

func stage122SpotInstanceStatusCode(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "cancelled":
		return "instance-terminated-by-user"
	case "failed":
		return "instance-terminated-by-price"
	default:
		return "fulfilled"
	}
}

func stage122SpotInstanceStatusMessage(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "cancelled":
		return "Spot Instance request cancelled"
	case "failed":
		return "Spot Instance request failed"
	default:
		return "Spot Instance request fulfilled"
	}
}
