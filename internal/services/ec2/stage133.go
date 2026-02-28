package ec2

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

func (s *Service) RestoreSnapshotFromRecycleBin(snapshotID string) (Snapshot, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return Snapshot{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshots[snapshotID]
	if snapshot == nil {
		return Snapshot{}, ErrNotFound
	}
	if snapshot.State == "" {
		snapshot.State = "completed"
	}
	if snapshot.Progress == "" {
		snapshot.Progress = "100%"
	}
	if snapshot.StartTime.IsZero() {
		snapshot.StartTime = time.Now().UTC()
	}
	return cloneSnapshot(snapshot), nil
}

func (s *Service) RestoreSnapshotTier(
	snapshotID string,
	permanentRestore *bool,
	temporaryRestoreDays *int32,
) (string, time.Time, int32, bool, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return "", time.Time{}, 0, false, ErrInvalidParameter
	}
	if temporaryRestoreDays != nil && *temporaryRestoreDays <= 0 {
		return "", time.Time{}, 0, false, ErrInvalidParameter
	}

	isPermanentRestore := permanentRestore != nil && *permanentRestore
	if isPermanentRestore && temporaryRestoreDays != nil {
		return "", time.Time{}, 0, false, ErrInvalidParameter
	}

	restoreDuration := int32(1)
	if temporaryRestoreDays != nil {
		restoreDuration = *temporaryRestoreDays
	}
	if isPermanentRestore {
		restoreDuration = 0
	}

	restoreStartTime := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot := s.snapshots[snapshotID]
	if snapshot == nil {
		return "", time.Time{}, 0, false, ErrNotFound
	}
	if snapshot.Tags == nil {
		snapshot.Tags = map[string]string{}
	}
	snapshot.Tags["storage-tier"] = "standard"
	if snapshot.State == "" {
		snapshot.State = "completed"
	}

	return snapshot.ID, restoreStartTime, restoreDuration, isPermanentRestore, nil
}

func (s *Service) RunScheduledInstances(
	scheduledInstanceID string,
	launchImageID string,
	launchInstanceType string,
	instanceCount *int32,
	clientToken *string,
) ([]string, error) {
	scheduledInstanceID = strings.TrimSpace(scheduledInstanceID)
	launchImageID = strings.TrimSpace(launchImageID)
	launchInstanceType = strings.TrimSpace(launchInstanceType)
	if scheduledInstanceID == "" || launchImageID == "" {
		return nil, ErrInvalidParameter
	}
	if launchInstanceType == "" {
		launchInstanceType = "t3.micro"
	}
	_ = strings.TrimSpace(derefString(clientToken))

	count := int32(1)
	if instanceCount != nil {
		count = *instanceCount
	}
	if count <= 0 {
		return nil, ErrInvalidParameter
	}

	instanceIDs := make([]string, 0, count)
	for i := int32(0); i < count; i++ {
		result, err := s.RunInstances(launchImageID, launchInstanceType, "", "", "", nil, 1, 1, nil)
		if err != nil {
			return nil, err
		}
		if len(result.Instances) == 0 || result.Instances[0].ID == "" {
			return nil, ErrConflict
		}
		instanceIDs = append(instanceIDs, result.Instances[0].ID)
	}

	return instanceIDs, nil
}

func (s *Service) SearchLocalGatewayRoutes(
	localGatewayRouteTableID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]LocalGatewayRoute, *string, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	if localGatewayRouteTableID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	prefixListFilterSet := toStringSet(standardFilters["prefix-list-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	typeFilterSet := toLowerStringSet(standardFilters["type"])

	exactMatches, err := stage133ParseCIDRPrefixFilters(standardFilters["route-search.exact-match"])
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	longestPrefixMatches, err := stage133ParseCIDRPrefixFilters(standardFilters["route-search.longest-prefix-match"])
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	subnetOfMatches, err := stage133ParseCIDRPrefixFilters(standardFilters["route-search.subnet-of-match"])
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	supernetOfMatches, err := stage133ParseCIDRPrefixFilters(standardFilters["route-search.supernet-of-match"])
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	if s.localGatewayRouteTables[localGatewayRouteTableID] == nil {
		s.mu.Unlock()
		return nil, nil, ErrNotFound
	}

	routes := make([]LocalGatewayRoute, 0, len(s.localGatewayRoutes))
	for _, route := range s.localGatewayRoutes {
		if route == nil || route.LocalGatewayRouteTableID != localGatewayRouteTableID {
			continue
		}
		if len(prefixListFilterSet) > 0 {
			if _, ok := prefixListFilterSet[route.DestinationPrefixListID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(route.State)]; !ok {
				continue
			}
		}
		if len(typeFilterSet) > 0 {
			if _, ok := typeFilterSet[strings.ToLower(route.Type)]; !ok {
				continue
			}
		}
		if !stage133MatchLocalGatewayRouteCIDRFilters(route.DestinationCidrBlock, exactMatches, longestPrefixMatches, subnetOfMatches, supernetOfMatches) {
			continue
		}
		routes = append(routes, cloneStage108LocalGatewayRoute(route))
	}
	s.mu.Unlock()

	sort.Slice(routes, func(i, j int) bool {
		return stage133LocalGatewayRouteSortKey(routes[i]) < stage133LocalGatewayRouteSortKey(routes[j])
	})

	windowStart, windowEnd, outputToken, err := ec2PageWindow(len(routes), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LocalGatewayRoute(nil), routes[windowStart:windowEnd]...), outputToken, nil
}

func (s *Service) SendDiagnosticInterrupt(instanceID string) error {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.instances[instanceID] == nil {
		return ErrNotFound
	}
	return nil
}

func (s *Service) StartNetworkInsightsAccessScopeAnalysis(
	networkInsightsAccessScopeID string,
	clientToken string,
	tags []Tag,
) (NetworkInsightsAccessScopeAnalysis, error) {
	networkInsightsAccessScopeID = strings.TrimSpace(networkInsightsAccessScopeID)
	clientToken = strings.TrimSpace(clientToken)
	if networkInsightsAccessScopeID == "" || clientToken == "" {
		return NetworkInsightsAccessScopeAnalysis{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scope := s.networkInsightsAccessScopes[networkInsightsAccessScopeID]
	if scope == nil {
		return NetworkInsightsAccessScopeAnalysis{}, ErrNotFound
	}

	now := time.Now().UTC()
	end := now.Add(1 * time.Minute)
	analysisID := stage120NetworkInsightsAccessScopeAnalysisID(networkInsightsAccessScopeID)
	analyzedENICount := int32(0)
	tagMap := tagsToMap(normalizeEC2Tags(tags))
	if len(tagMap) == 0 {
		tagMap = cloneStringMap(scope.Tags)
	}

	return NetworkInsightsAccessScopeAnalysis{
		AnalyzedENICount:                      cloneInt32Pointer(&analyzedENICount),
		EndDate:                               cloneTimePointer(&end),
		FindingsFound:                         "false",
		NetworkInsightsAccessScopeAnalysisARN: fmt.Sprintf("arn:aws:ec2:%s:%s:network-insights-access-scope-analysis/%s", DefaultRegion, DefaultAccountID, analysisID),
		NetworkInsightsAccessScopeAnalysisID:  analysisID,
		NetworkInsightsAccessScopeID:          networkInsightsAccessScopeID,
		StartDate:                             cloneTimePointer(&now),
		Status:                                "succeeded",
		StatusMessage:                         "analysis complete",
		Tags:                                  tagMap,
		WarningMessage:                        "",
	}, nil
}

func (s *Service) StartNetworkInsightsAnalysis(
	networkInsightsPathID string,
	clientToken string,
	tags []Tag,
) (NetworkInsightsAnalysis, error) {
	networkInsightsPathID = strings.TrimSpace(networkInsightsPathID)
	clientToken = strings.TrimSpace(clientToken)
	if networkInsightsPathID == "" || clientToken == "" {
		return NetworkInsightsAnalysis{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.networkInsightsPaths[networkInsightsPathID]
	if path == nil {
		return NetworkInsightsAnalysis{}, ErrNotFound
	}

	now := time.Now().UTC()
	analysisID := stage120NetworkInsightsAnalysisID(networkInsightsPathID)
	pathFound := true
	tagMap := tagsToMap(normalizeEC2Tags(tags))
	if len(tagMap) == 0 {
		tagMap = cloneStringMap(path.Tags)
	}

	return NetworkInsightsAnalysis{
		NetworkInsightsAnalysisARN: fmt.Sprintf("arn:aws:ec2:%s:%s:network-insights-analysis/%s", DefaultRegion, DefaultAccountID, analysisID),
		NetworkInsightsAnalysisID:  analysisID,
		NetworkInsightsPathID:      networkInsightsPathID,
		NetworkPathFound:           &pathFound,
		StartDate:                  cloneTimePointer(&now),
		Status:                     "succeeded",
		StatusMessage:              "analysis complete",
		Tags:                       tagMap,
		WarningMessage:             "",
	}, nil
}

func (s *Service) UnlockSnapshot(snapshotID string) (string, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.snapshots[snapshotID] == nil {
		return "", ErrNotFound
	}
	if s.lockedSnapshots[snapshotID] == nil {
		return "", ErrNotFound
	}
	delete(s.lockedSnapshots, snapshotID)
	return snapshotID, nil
}

func (s *Service) WithdrawByoipCidr(cidr string) (ByoipCidr, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return ByoipCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	record := s.byoipCidrs[cidr]
	if record == nil {
		return ByoipCidr{}, ErrNotFound
	}

	record.State = "withdrawn"
	record.StatusMessage = "withdrawn"
	for i := range record.AsnAssociations {
		record.AsnAssociations[i].State = "withdrawn"
		record.AsnAssociations[i].StatusMessage = "withdrawn"
	}
	return cloneByoipCidr(record), nil
}

func stage133ParseCIDRPrefixFilters(values []string) ([]netip.Prefix, error) {
	cleaned := dedupeTrimmedStrings(values)
	out := make([]netip.Prefix, 0, len(cleaned))
	for _, value := range cleaned {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, err
		}
		out = append(out, prefix.Masked())
	}
	return out, nil
}

func stage133MatchLocalGatewayRouteCIDRFilters(
	destinationCIDR string,
	exactMatches []netip.Prefix,
	longestPrefixMatches []netip.Prefix,
	subnetOfMatches []netip.Prefix,
	supernetOfMatches []netip.Prefix,
) bool {
	if len(exactMatches) == 0 && len(longestPrefixMatches) == 0 && len(subnetOfMatches) == 0 && len(supernetOfMatches) == 0 {
		return true
	}
	destinationCIDR = strings.TrimSpace(destinationCIDR)
	if destinationCIDR == "" {
		return false
	}

	destinationPrefix, err := netip.ParsePrefix(destinationCIDR)
	if err != nil {
		return false
	}
	destinationPrefix = destinationPrefix.Masked()

	if len(exactMatches) > 0 && !stage133AnyPrefixMatch(destinationPrefix, exactMatches, func(routePrefix, filterPrefix netip.Prefix) bool {
		return routePrefix == filterPrefix
	}) {
		return false
	}
	if len(longestPrefixMatches) > 0 && !stage133AnyPrefixMatch(destinationPrefix, longestPrefixMatches, stage133LongestPrefixMatch) {
		return false
	}
	if len(subnetOfMatches) > 0 && !stage133AnyPrefixMatch(destinationPrefix, subnetOfMatches, stage133SubnetOfMatch) {
		return false
	}
	if len(supernetOfMatches) > 0 && !stage133AnyPrefixMatch(destinationPrefix, supernetOfMatches, stage133SupernetOfMatch) {
		return false
	}

	return true
}

func stage133AnyPrefixMatch(routePrefix netip.Prefix, filters []netip.Prefix, matcher func(netip.Prefix, netip.Prefix) bool) bool {
	for _, filterPrefix := range filters {
		if matcher(routePrefix, filterPrefix) {
			return true
		}
	}
	return false
}

func stage133LongestPrefixMatch(routePrefix, filterPrefix netip.Prefix) bool {
	return routePrefix.Contains(filterPrefix.Addr()) && routePrefix.Bits() <= filterPrefix.Bits()
}

func stage133SubnetOfMatch(routePrefix, filterPrefix netip.Prefix) bool {
	return filterPrefix.Contains(routePrefix.Addr()) && routePrefix.Bits() >= filterPrefix.Bits()
}

func stage133SupernetOfMatch(routePrefix, filterPrefix netip.Prefix) bool {
	return routePrefix.Contains(filterPrefix.Addr()) && routePrefix.Bits() <= filterPrefix.Bits()
}

func stage133LocalGatewayRouteSortKey(route LocalGatewayRoute) string {
	return strings.Join([]string{
		route.LocalGatewayRouteTableID,
		route.DestinationCidrBlock,
		route.DestinationPrefixListID,
		route.LocalGatewayVirtualInterfaceGroupID,
		route.NetworkInterfaceID,
		route.State,
		route.Type,
	}, "|")
}
