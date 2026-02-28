package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type NetworkInsightsAccessScopeAnalysis struct {
	AnalyzedENICount                      *int32
	EndDate                               *time.Time
	FindingsFound                         string
	NetworkInsightsAccessScopeAnalysisARN string
	NetworkInsightsAccessScopeAnalysisID  string
	NetworkInsightsAccessScopeID          string
	StartDate                             *time.Time
	Status                                string
	StatusMessage                         string
	Tags                                  map[string]string
	WarningMessage                        string
}

type NetworkInsightsAnalysis struct {
	NetworkInsightsAnalysisARN string
	NetworkInsightsAnalysisID  string
	NetworkInsightsPathID      string
	NetworkPathFound           *bool
	StartDate                  *time.Time
	Status                     string
	StatusMessage              string
	Tags                       map[string]string
	WarningMessage             string
}

type OutpostLag struct {
	LocalGatewayVirtualInterfaceIDs []string
	OutpostARN                      string
	OutpostLagID                    string
	OwnerID                         string
	ServiceLinkVirtualInterfaceIDs  []string
	State                           string
	Tags                            map[string]string
}

type PrefixList struct {
	CIDRs          []string
	PrefixListID   string
	PrefixListName string
}

type ReservedInstanceRecurringCharge struct {
	Amount    float32
	Frequency string
}

type ReservedInstance struct {
	AvailabilityZone    string
	AvailabilityZoneID  string
	CurrencyCode        string
	Duration            *int64
	End                 *time.Time
	FixedPrice          *float32
	InstanceCount       *int32
	InstanceTenancy     string
	InstanceType        string
	OfferingClass       string
	OfferingType        string
	ProductDescription  string
	RecurringCharges    []ReservedInstanceRecurringCharge
	ReservedInstancesID string
	Scope               string
	Start               *time.Time
	State               string
	Tags                map[string]string
	UsagePrice          *float32
}

func (s *Service) DescribeManagedPrefixLists(prefixListIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ManagedPrefixList, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(prefixListIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	prefixListIDFilterSet := toStringSet(standardFilters["prefix-list-id"])
	prefixListNameFilterSet := toStringSet(standardFilters["prefix-list-name"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.managedPrefixLists))
	for prefixListID := range s.managedPrefixLists {
		ids = append(ids, prefixListID)
	}
	sort.Strings(ids)

	items := make([]ManagedPrefixList, 0, len(ids))
	for _, prefixListID := range ids {
		prefixList := s.managedPrefixLists[prefixListID]
		if prefixList == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[prefixListID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[prefixList.OwnerID]; !ok {
				continue
			}
		}
		if len(prefixListIDFilterSet) > 0 {
			if _, ok := prefixListIDFilterSet[prefixListID]; !ok {
				continue
			}
		}
		if len(prefixListNameFilterSet) > 0 {
			if _, ok := prefixListNameFilterSet[prefixList.PrefixListName]; !ok {
				continue
			}
		}
		if !matchesTagFilters(prefixList.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage109ManagedPrefixList(prefixList))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ManagedPrefixList(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeNetworkInsightsAccessScopeAnalyses(
	analysisIDs []string,
	networkInsightsAccessScopeID *string,
	analysisStartTimeBegin *time.Time,
	analysisStartTimeEnd *time.Time,
	_ map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]NetworkInsightsAccessScopeAnalysis, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(analysisIDs))
	scopeID := strings.TrimSpace(derefString(networkInsightsAccessScopeID))

	s.mu.Lock()
	scopeIDs := make([]string, 0, len(s.networkInsightsAccessScopes))
	for id := range s.networkInsightsAccessScopes {
		scopeIDs = append(scopeIDs, id)
	}
	sort.Strings(scopeIDs)

	items := make([]NetworkInsightsAccessScopeAnalysis, 0, len(scopeIDs))
	for _, id := range scopeIDs {
		scope := s.networkInsightsAccessScopes[id]
		if scope == nil {
			continue
		}
		if scopeID != "" && id != scopeID {
			continue
		}

		analysisID := stage120NetworkInsightsAccessScopeAnalysisID(id)
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[analysisID]; !ok {
				continue
			}
		}

		startDate := scope.CreatedDate.UTC()
		if startDate.IsZero() {
			startDate = time.Now().UTC()
		}
		endDate := scope.UpdatedDate.UTC()
		if endDate.IsZero() || endDate.Before(startDate) {
			endDate = startDate.Add(1 * time.Minute)
		}

		if analysisStartTimeBegin != nil && startDate.Before(analysisStartTimeBegin.UTC()) {
			continue
		}
		if analysisStartTimeEnd != nil && startDate.After(analysisStartTimeEnd.UTC()) {
			continue
		}

		analyzedENICount := int32(0)
		item := NetworkInsightsAccessScopeAnalysis{
			AnalyzedENICount:                      &analyzedENICount,
			EndDate:                               cloneTimePointer(&endDate),
			FindingsFound:                         "false",
			NetworkInsightsAccessScopeAnalysisARN: fmt.Sprintf("arn:aws:ec2:%s:%s:network-insights-access-scope-analysis/%s", DefaultRegion, DefaultAccountID, analysisID),
			NetworkInsightsAccessScopeAnalysisID:  analysisID,
			NetworkInsightsAccessScopeID:          id,
			StartDate:                             cloneTimePointer(&startDate),
			Status:                                "succeeded",
			StatusMessage:                         "analysis complete",
			Tags:                                  cloneStringMap(scope.Tags),
			WarningMessage:                        "",
		}
		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]NetworkInsightsAccessScopeAnalysis(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeNetworkInsightsAccessScopes(scopeIDs []string, _ map[string][]string, maxResults *int32, nextToken *string) ([]NetworkInsightsAccessScope, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(scopeIDs))

	s.mu.Lock()
	ids := make([]string, 0, len(s.networkInsightsAccessScopes))
	for id := range s.networkInsightsAccessScopes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]NetworkInsightsAccessScope, 0, len(ids))
	for _, id := range ids {
		scope := s.networkInsightsAccessScopes[id]
		if scope == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[id]; !ok {
				continue
			}
		}
		items = append(items, cloneStage109NetworkInsightsAccessScope(scope))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]NetworkInsightsAccessScope(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeNetworkInsightsAnalyses(
	analysisIDs []string,
	networkInsightsPathID *string,
	analysisStartTime *time.Time,
	analysisEndTime *time.Time,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]NetworkInsightsAnalysis, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(analysisIDs))
	pathID := strings.TrimSpace(derefString(networkInsightsPathID))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	statusFilterSet := toLowerStringSet(standardFilters["status"])
	pathFoundFilterSet := toLowerStringSet(standardFilters["path-found"])

	s.mu.Lock()
	pathIDs := make([]string, 0, len(s.networkInsightsPaths))
	for id := range s.networkInsightsPaths {
		pathIDs = append(pathIDs, id)
	}
	sort.Strings(pathIDs)

	items := make([]NetworkInsightsAnalysis, 0, len(pathIDs))
	for _, id := range pathIDs {
		path := s.networkInsightsPaths[id]
		if path == nil {
			continue
		}
		if pathID != "" && id != pathID {
			continue
		}

		analysisID := stage120NetworkInsightsAnalysisID(id)
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[analysisID]; !ok {
				continue
			}
		}

		startDate := path.CreatedDate.UTC()
		if startDate.IsZero() {
			startDate = time.Now().UTC()
		}
		if analysisStartTime != nil && startDate.Before(analysisStartTime.UTC()) {
			continue
		}
		if analysisEndTime != nil && startDate.After(analysisEndTime.UTC()) {
			continue
		}

		pathFound := true
		if len(pathFoundFilterSet) > 0 {
			if _, ok := pathFoundFilterSet[strconv.FormatBool(pathFound)]; !ok {
				continue
			}
		}

		status := "succeeded"
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(status)]; !ok {
				continue
			}
		}

		if !matchesTagFilters(path.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		item := NetworkInsightsAnalysis{
			NetworkInsightsAnalysisARN: fmt.Sprintf("arn:aws:ec2:%s:%s:network-insights-analysis/%s", DefaultRegion, DefaultAccountID, analysisID),
			NetworkInsightsAnalysisID:  analysisID,
			NetworkInsightsPathID:      id,
			NetworkPathFound:           &pathFound,
			StartDate:                  cloneTimePointer(&startDate),
			Status:                     status,
			StatusMessage:              "analysis complete",
			Tags:                       cloneStringMap(path.Tags),
			WarningMessage:             "",
		}
		items = append(items, item)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]NetworkInsightsAnalysis(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeNetworkInsightsPaths(pathIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]NetworkInsightsPath, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(pathIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	destinationFilterSet := toStringSet(standardFilters["destination"])
	destinationPortFilterSet := toStringSet(standardFilters["destination-port"])
	protocolFilterSet := toLowerStringSet(standardFilters["protocol"])
	sourceFilterSet := toStringSet(standardFilters["source"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.networkInsightsPaths))
	for id := range s.networkInsightsPaths {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]NetworkInsightsPath, 0, len(ids))
	for _, id := range ids {
		path := s.networkInsightsPaths[id]
		if path == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[id]; !ok {
				continue
			}
		}
		if len(destinationFilterSet) > 0 {
			if _, ok := destinationFilterSet[path.Destination]; !ok {
				continue
			}
		}
		if len(destinationPortFilterSet) > 0 {
			if path.DestinationPort == nil {
				continue
			}
			if _, ok := destinationPortFilterSet[strconv.FormatInt(int64(*path.DestinationPort), 10)]; !ok {
				continue
			}
		}
		if len(protocolFilterSet) > 0 {
			if _, ok := protocolFilterSet[strings.ToLower(path.Protocol)]; !ok {
				continue
			}
		}
		if len(sourceFilterSet) > 0 {
			if _, ok := sourceFilterSet[path.Source]; !ok {
				continue
			}
		}
		if !matchesTagFilters(path.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage109NetworkInsightsPath(path))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]NetworkInsightsPath(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeOutpostLags(outpostLagIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]OutpostLag, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(outpostLagIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	outpostLagIDFilterSet := toStringSet(standardFilters["outpost-lag-id"])
	outpostARNFilterSet := toStringSet(standardFilters["outpost-arn"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["configuration-state"])

	s.mu.Lock()
	byID := map[string]*OutpostLag{}
	for _, virtualInterface := range s.localGatewayVirtualInterfaces {
		if virtualInterface == nil {
			continue
		}
		lagID := strings.TrimSpace(virtualInterface.OutpostLagID)
		if lagID == "" {
			continue
		}
		lag := byID[lagID]
		if lag == nil {
			lag = &OutpostLag{
				LocalGatewayVirtualInterfaceIDs: []string{},
				OutpostARN:                      stage120OutpostARNFromLagID(lagID),
				OutpostLagID:                    lagID,
				OwnerID:                         DefaultAccountID,
				ServiceLinkVirtualInterfaceIDs:  []string{},
				State:                           "available",
				Tags:                            map[string]string{},
			}
			byID[lagID] = lag
		}
		lag.LocalGatewayVirtualInterfaceIDs = append(lag.LocalGatewayVirtualInterfaceIDs, virtualInterface.LocalGatewayVirtualInterfaceID)
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]OutpostLag, 0, len(ids))
	for _, id := range ids {
		lag := byID[id]
		if lag == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[id]; !ok {
				continue
			}
		}
		if len(outpostLagIDFilterSet) > 0 {
			if _, ok := outpostLagIDFilterSet[id]; !ok {
				continue
			}
		}
		if len(outpostARNFilterSet) > 0 {
			if _, ok := outpostARNFilterSet[lag.OutpostARN]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[lag.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(lag.State)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(lag.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		clone := cloneStage120OutpostLag(lag)
		sort.Strings(clone.LocalGatewayVirtualInterfaceIDs)
		sort.Strings(clone.ServiceLinkVirtualInterfaceIDs)
		items = append(items, clone)
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]OutpostLag(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribePrefixLists(prefixListIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]PrefixList, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(prefixListIDs))
	standardFilters, _, _ := splitEC2Filters(filters)
	prefixListIDFilterSet := toStringSet(standardFilters["prefix-list-id"])
	prefixListNameFilterSet := toStringSet(standardFilters["prefix-list-name"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.managedPrefixLists))
	for id := range s.managedPrefixLists {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]PrefixList, 0, len(ids))
	for _, id := range ids {
		list := s.managedPrefixLists[id]
		if list == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[id]; !ok {
				continue
			}
		}
		if len(prefixListIDFilterSet) > 0 {
			if _, ok := prefixListIDFilterSet[id]; !ok {
				continue
			}
		}
		if len(prefixListNameFilterSet) > 0 {
			if _, ok := prefixListNameFilterSet[list.PrefixListName]; !ok {
				continue
			}
		}
		cidrs := []string{"0.0.0.0/0"}
		if strings.EqualFold(list.AddressFamily, "ipv6") {
			cidrs = []string{"::/0"}
		}
		items = append(items, PrefixList{
			CIDRs:          cidrs,
			PrefixListID:   list.PrefixListID,
			PrefixListName: list.PrefixListName,
		})
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]PrefixList(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribePublicIpv4Pools(poolIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]PublicIpv4Pool, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(poolIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	poolIDFilterSet := toStringSet(standardFilters["pool-id"])
	networkBorderGroupFilterSet := toStringSet(standardFilters["network-border-group"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.publicIpv4Pools))
	for id := range s.publicIpv4Pools {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]PublicIpv4Pool, 0, len(ids))
	for _, id := range ids {
		pool := s.publicIpv4Pools[id]
		if pool == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[id]; !ok {
				continue
			}
		}
		if len(poolIDFilterSet) > 0 {
			if _, ok := poolIDFilterSet[id]; !ok {
				continue
			}
		}
		if len(networkBorderGroupFilterSet) > 0 {
			if _, ok := networkBorderGroupFilterSet[pool.NetworkBorderGroup]; !ok {
				continue
			}
		}
		if !matchesTagFilters(pool.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage109PublicIpv4Pool(pool))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]PublicIpv4Pool(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeReplaceRootVolumeTasks(taskIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ReplaceRootVolumeTask, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(taskIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	taskStateFilterSet := toLowerStringSet(standardFilters["task-state"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.replaceRootVolumeTasks))
	for id := range s.replaceRootVolumeTasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]ReplaceRootVolumeTask, 0, len(ids))
	for _, id := range ids {
		task := s.replaceRootVolumeTasks[id]
		if task == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[id]; !ok {
				continue
			}
		}
		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[task.InstanceID]; !ok {
				continue
			}
		}
		if len(taskStateFilterSet) > 0 {
			if _, ok := taskStateFilterSet[strings.ToLower(task.TaskState)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(task.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage109ReplaceRootVolumeTask(task))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ReplaceRootVolumeTask(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeReservedInstances(reservedInstanceIDs []string, filters map[string][]string, offeringClass string, offeringType string) ([]ReservedInstance, error) {
	requestedIDSet := toStringSet(dedupeTrimmedStrings(reservedInstanceIDs))
	offeringClass = strings.ToLower(strings.TrimSpace(offeringClass))
	offeringType = strings.ToLower(strings.TrimSpace(offeringType))

	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	availabilityZoneFilterSet := toStringSet(standardFilters["availability-zone"])
	availabilityZoneIDFilterSet := toStringSet(standardFilters["availability-zone-id"])
	durationFilterSet := toStringSet(standardFilters["duration"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	offeringClassFilterSet := toLowerStringSet(standardFilters["offering-class"])
	offeringTypeFilterSet := toLowerStringSet(standardFilters["offering-type"])
	productDescriptionFilterSet := toLowerStringSet(standardFilters["product-description"])
	reservedInstancesIDFilterSet := toStringSet(standardFilters["reserved-instances-id"])
	scopeFilterSet := toLowerStringSet(standardFilters["scope"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])

	s.mu.Lock()
	listingIDs := make([]string, 0, len(s.reservedInstancesListingStates))
	for listingID := range s.reservedInstancesListingStates {
		listingIDs = append(listingIDs, listingID)
	}
	for listingID := range s.reservedInstancesListingCreatedAt {
		if _, ok := s.reservedInstancesListingStates[listingID]; !ok {
			listingIDs = append(listingIDs, listingID)
		}
	}
	sort.Strings(listingIDs)

	now := time.Now().UTC()
	items := make([]ReservedInstance, 0, len(listingIDs))
	for _, listingID := range listingIDs {
		reservedID := stage120ReservedInstancesIDFromListingID(listingID)
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[reservedID]; !ok {
				continue
			}
		}

		state := stage120ReservedInstanceStateFromListingState(s.reservedInstancesListingStates[listingID])
		start := s.reservedInstancesListingCreatedAt[listingID]
		if start.IsZero() {
			start = now
		}
		end := start.Add(365 * 24 * time.Hour)
		duration := int64(31536000)
		instanceCount := int32(1)
		fixedPrice := float32(0.0)
		usagePrice := float32(0.0)
		item := ReservedInstance{
			AvailabilityZone:   "us-east-1a",
			AvailabilityZoneID: "use1-az1",
			CurrencyCode:       "USD",
			Duration:           &duration,
			End:                cloneTimePointer(&end),
			FixedPrice:         &fixedPrice,
			InstanceCount:      &instanceCount,
			InstanceTenancy:    "default",
			InstanceType:       "t3.micro",
			OfferingClass:      "standard",
			OfferingType:       "No Upfront",
			ProductDescription: "Linux/UNIX",
			RecurringCharges: []ReservedInstanceRecurringCharge{
				{Amount: 0.0, Frequency: "Hourly"},
			},
			ReservedInstancesID: reservedID,
			Scope:               "Region",
			Start:               cloneTimePointer(&start),
			State:               state,
			Tags:                map[string]string{},
			UsagePrice:          &usagePrice,
		}

		if offeringClass != "" && strings.ToLower(item.OfferingClass) != offeringClass {
			continue
		}
		if offeringType != "" && strings.ToLower(item.OfferingType) != offeringType {
			continue
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
		if len(durationFilterSet) > 0 {
			if _, ok := durationFilterSet[strconv.FormatInt(*item.Duration, 10)]; !ok {
				continue
			}
		}
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}
		if len(offeringClassFilterSet) > 0 {
			if _, ok := offeringClassFilterSet[strings.ToLower(item.OfferingClass)]; !ok {
				continue
			}
		}
		if len(offeringTypeFilterSet) > 0 {
			if _, ok := offeringTypeFilterSet[strings.ToLower(item.OfferingType)]; !ok {
				continue
			}
		}
		if len(productDescriptionFilterSet) > 0 {
			if _, ok := productDescriptionFilterSet[strings.ToLower(item.ProductDescription)]; !ok {
				continue
			}
		}
		if len(reservedInstancesIDFilterSet) > 0 {
			if _, ok := reservedInstancesIDFilterSet[item.ReservedInstancesID]; !ok {
				continue
			}
		}
		if len(scopeFilterSet) > 0 {
			if _, ok := scopeFilterSet[strings.ToLower(item.Scope)]; !ok {
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

		items = append(items, cloneStage120ReservedInstance(item))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].ReservedInstancesID < items[j].ReservedInstancesID
	})
	return items, nil
}

func stage120NetworkInsightsAccessScopeAnalysisID(scopeID string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(scopeID), "nias-")
	if suffix == "" || suffix == scopeID {
		suffix = strings.ReplaceAll(strings.TrimSpace(scopeID), "-", "")
	}
	if suffix == "" {
		suffix = "000000000000"
	}
	return "niasa-" + suffix
}

func stage120NetworkInsightsAnalysisID(pathID string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(pathID), "nip-")
	if suffix == "" || suffix == pathID {
		suffix = strings.ReplaceAll(strings.TrimSpace(pathID), "-", "")
	}
	if suffix == "" {
		suffix = "000000000000"
	}
	return "nia-" + suffix
}

func stage120OutpostARNFromLagID(lagID string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(lagID), "lag-")
	if suffix == "" {
		suffix = "000000000000"
	}
	return fmt.Sprintf("arn:aws:outposts:%s:%s:outpost/op-%s", DefaultRegion, DefaultAccountID, suffix)
}

func stage120ReservedInstancesIDFromListingID(listingID string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(listingID), "ril-")
	if suffix == "" {
		suffix = strings.ReplaceAll(strings.TrimSpace(listingID), "-", "")
	}
	if suffix == "" {
		suffix = "000000000000"
	}
	return "ri-" + suffix
}

func stage120ReservedInstanceStateFromListingState(listingState string) string {
	switch strings.ToLower(strings.TrimSpace(listingState)) {
	case "queued":
		return "payment-pending"
	case "cancelled", "canceled":
		return "retired"
	case "payment-failed":
		return "payment-failed"
	default:
		return "active"
	}
}

func cloneStage120OutpostLag(in *OutpostLag) OutpostLag {
	if in == nil {
		return OutpostLag{}
	}
	return OutpostLag{
		LocalGatewayVirtualInterfaceIDs: append([]string(nil), in.LocalGatewayVirtualInterfaceIDs...),
		OutpostARN:                      in.OutpostARN,
		OutpostLagID:                    in.OutpostLagID,
		OwnerID:                         in.OwnerID,
		ServiceLinkVirtualInterfaceIDs:  append([]string(nil), in.ServiceLinkVirtualInterfaceIDs...),
		State:                           in.State,
		Tags:                            cloneStringMap(in.Tags),
	}
}

func cloneStage120ReservedInstance(in ReservedInstance) ReservedInstance {
	out := in
	out.Duration = cloneInt64Pointer(in.Duration)
	out.End = cloneTimePointer(in.End)
	out.FixedPrice = cloneFloat32Pointer(in.FixedPrice)
	out.InstanceCount = cloneInt32Pointer(in.InstanceCount)
	out.Start = cloneTimePointer(in.Start)
	out.UsagePrice = cloneFloat32Pointer(in.UsagePrice)
	out.Tags = cloneStringMap(in.Tags)
	out.RecurringCharges = append([]ReservedInstanceRecurringCharge(nil), in.RecurringCharges...)
	return out
}

func cloneFloat32Pointer(in *float32) *float32 {
	if in == nil {
		return nil
	}
	v := *in
	return &v
}
