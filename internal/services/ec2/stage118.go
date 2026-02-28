package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type InstanceTypeInfo struct {
	InstanceType string
}

type Ipv6Pool struct {
	Description    string
	PoolID         string
	PoolCIDRBlocks []string
	Tags           map[string]string
}

func (s *Service) DescribeInstanceTypes(instanceTypes []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]InstanceTypeInfo, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedTypes := dedupeTrimmedStrings(instanceTypes)
	standardFilters, _, _ := splitEC2Filters(filters)
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])

	s.mu.Lock()
	candidateSet := map[string]struct{}{}
	if len(requestedTypes) > 0 {
		for _, instanceType := range requestedTypes {
			candidateSet[instanceType] = struct{}{}
		}
	} else {
		for _, instance := range s.instances {
			if instance == nil {
				continue
			}
			instanceType := strings.TrimSpace(instance.InstanceType)
			if instanceType == "" {
				continue
			}
			candidateSet[instanceType] = struct{}{}
		}
		for _, host := range s.dedicatedHosts {
			if host == nil {
				continue
			}
			instanceType := strings.TrimSpace(host.InstanceType)
			if instanceType == "" {
				instanceType = "m5.large"
			}
			candidateSet[instanceType] = struct{}{}
		}
		for _, fleet := range s.fleets {
			if fleet == nil {
				continue
			}
			for _, fleetInstance := range fleet.Instances {
				instanceType := strings.TrimSpace(fleetInstance.InstanceType)
				if instanceType == "" {
					continue
				}
				candidateSet[instanceType] = struct{}{}
			}
		}
		if len(candidateSet) == 0 {
			candidateSet["t3.micro"] = struct{}{}
			candidateSet["m5.large"] = struct{}{}
		}
	}
	s.mu.Unlock()

	keys := make([]string, 0, len(candidateSet))
	for instanceType := range candidateSet {
		keys = append(keys, instanceType)
	}
	sort.Strings(keys)

	items := make([]InstanceTypeInfo, 0, len(keys))
	for _, instanceType := range keys {
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(instanceType)]; !ok {
				continue
			}
		}
		items = append(items, InstanceTypeInfo{InstanceType: instanceType})
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceTypeInfo(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpamByoasn(maxResults *int32, nextToken *string) ([]Byoasn, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	ipamIDs := make([]string, 0, len(s.ipams))
	for ipamID := range s.ipams {
		ipamIDs = append(ipamIDs, ipamID)
	}
	sort.Strings(ipamIDs)
	defaultIpamID := ""
	if len(ipamIDs) > 0 {
		defaultIpamID = ipamIDs[0]
	}

	byASN := map[string]Byoasn{}
	for _, record := range s.byoipCidrs {
		if record == nil {
			continue
		}
		for _, assoc := range record.AsnAssociations {
			asn := strings.TrimSpace(assoc.Asn)
			if asn == "" {
				continue
			}
			state := strings.TrimSpace(assoc.State)
			if state == "" {
				state = "associated"
			}
			statusMessage := strings.TrimSpace(assoc.StatusMessage)
			if statusMessage == "" {
				statusMessage = state
			}
			byASN[asn] = Byoasn{
				Asn:           asn,
				IpamID:        defaultIpamID,
				State:         state,
				StatusMessage: statusMessage,
			}
		}
	}
	s.mu.Unlock()

	asns := make([]string, 0, len(byASN))
	for asn := range byASN {
		asns = append(asns, asn)
	}
	sort.Strings(asns)

	items := make([]Byoasn, 0, len(asns))
	for _, asn := range asns {
		items = append(items, byASN[asn])
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]Byoasn(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpamExternalResourceVerificationTokens(tokenIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]IpamExternalResourceVerificationToken, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(tokenIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	tokenIDFilterSet := toStringSet(standardFilters["ipam-external-resource-verification-token-id"])
	ipamIDFilterSet := toStringSet(standardFilters["ipam-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	statusFilterSet := toLowerStringSet(standardFilters["status"])
	tokenNameFilterSet := toStringSet(standardFilters["token-name"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.ipamExternalResourceVerificationTokens))
	for tokenID := range s.ipamExternalResourceVerificationTokens {
		ids = append(ids, tokenID)
	}
	sort.Strings(ids)

	items := make([]IpamExternalResourceVerificationToken, 0, len(ids))
	for _, tokenID := range ids {
		token := s.ipamExternalResourceVerificationTokens[tokenID]
		if token == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[tokenID]; !ok {
				continue
			}
		}
		if len(tokenIDFilterSet) > 0 {
			if _, ok := tokenIDFilterSet[tokenID]; !ok {
				continue
			}
		}
		if len(ipamIDFilterSet) > 0 {
			if _, ok := ipamIDFilterSet[token.IpamID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(token.State)]; !ok {
				continue
			}
		}
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(token.Status)]; !ok {
				continue
			}
		}
		if len(tokenNameFilterSet) > 0 {
			if _, ok := tokenNameFilterSet[token.TokenName]; !ok {
				continue
			}
		}
		if !matchesTagFilters(token.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage107IpamExternalResourceVerificationToken(token))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamExternalResourceVerificationToken(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpamPools(poolIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]IpamPool, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(poolIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	poolIDFilterSet := toStringSet(standardFilters["ipam-pool-id"])
	scopeIDFilterSet := toStringSet(standardFilters["ipam-scope-id"])
	scopeTypeFilterSet := toLowerStringSet(standardFilters["ipam-scope-type"])
	addressFamilyFilterSet := toLowerStringSet(standardFilters["address-family"])
	localeFilterSet := toLowerStringSet(standardFilters["locale"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	awsServiceFilterSet := toLowerStringSet(standardFilters["aws-service"])
	publicIPSourceFilterSet := toLowerStringSet(standardFilters["public-ip-source"])
	sourcePoolIDFilterSet := toStringSet(standardFilters["source-ipam-pool-id"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.ipamPools))
	for poolID := range s.ipamPools {
		ids = append(ids, poolID)
	}
	sort.Strings(ids)

	items := make([]IpamPool, 0, len(ids))
	for _, poolID := range ids {
		pool := s.ipamPools[poolID]
		if pool == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[poolID]; !ok {
				continue
			}
		}
		if len(poolIDFilterSet) > 0 {
			if _, ok := poolIDFilterSet[poolID]; !ok {
				continue
			}
		}
		if len(scopeIDFilterSet) > 0 {
			if _, ok := scopeIDFilterSet[pool.IpamScopeID]; !ok {
				continue
			}
		}
		if len(scopeTypeFilterSet) > 0 {
			if _, ok := scopeTypeFilterSet[strings.ToLower(pool.IpamScopeType)]; !ok {
				continue
			}
		}
		if len(addressFamilyFilterSet) > 0 {
			if _, ok := addressFamilyFilterSet[strings.ToLower(pool.AddressFamily)]; !ok {
				continue
			}
		}
		if len(localeFilterSet) > 0 {
			if _, ok := localeFilterSet[strings.ToLower(pool.Locale)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(pool.State)]; !ok {
				continue
			}
		}
		if len(awsServiceFilterSet) > 0 {
			if _, ok := awsServiceFilterSet[strings.ToLower(pool.AwsService)]; !ok {
				continue
			}
		}
		if len(publicIPSourceFilterSet) > 0 {
			if _, ok := publicIPSourceFilterSet[strings.ToLower(pool.PublicIpSource)]; !ok {
				continue
			}
		}
		if len(sourcePoolIDFilterSet) > 0 {
			if _, ok := sourcePoolIDFilterSet[pool.SourceIpamPoolID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(pool.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108IpamPool(pool))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamPool(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpamResourceDiscoveries(discoveryIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]IpamResourceDiscovery, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(discoveryIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	discoveryIDFilterSet := toStringSet(standardFilters["ipam-resource-discovery-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	isDefaultFilterSet := toLowerStringSet(standardFilters["is-default"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.ipamResourceDiscoveries))
	for discoveryID := range s.ipamResourceDiscoveries {
		ids = append(ids, discoveryID)
	}
	sort.Strings(ids)

	items := make([]IpamResourceDiscovery, 0, len(ids))
	for _, discoveryID := range ids {
		discovery := s.ipamResourceDiscoveries[discoveryID]
		if discovery == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[discoveryID]; !ok {
				continue
			}
		}
		if len(discoveryIDFilterSet) > 0 {
			if _, ok := discoveryIDFilterSet[discoveryID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[discovery.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(discovery.State)]; !ok {
				continue
			}
		}
		if len(isDefaultFilterSet) > 0 {
			if !matchesStage118BoolFilterSet(isDefaultFilterSet, discovery.IsDefault) {
				continue
			}
		}
		if !matchesTagFilters(discovery.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108IpamResourceDiscovery(discovery))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamResourceDiscovery(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpamResourceDiscoveryAssociations(associationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]IpamResourceDiscoveryAssociation, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(associationIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	associationIDFilterSet := toStringSet(standardFilters["ipam-resource-discovery-association-id"])
	ipamIDFilterSet := toStringSet(standardFilters["ipam-id"])
	discoveryIDFilterSet := toStringSet(standardFilters["ipam-resource-discovery-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	resourceDiscoveryStatusFilterSet := toLowerStringSet(standardFilters["resource-discovery-status"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	isDefaultFilterSet := toLowerStringSet(standardFilters["is-default"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.ipamResourceDiscoveryAssociations))
	for associationID := range s.ipamResourceDiscoveryAssociations {
		ids = append(ids, associationID)
	}
	sort.Strings(ids)

	items := make([]IpamResourceDiscoveryAssociation, 0, len(ids))
	for _, associationID := range ids {
		association := s.ipamResourceDiscoveryAssociations[associationID]
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
		if len(ipamIDFilterSet) > 0 {
			if _, ok := ipamIDFilterSet[association.IpamID]; !ok {
				continue
			}
		}
		if len(discoveryIDFilterSet) > 0 {
			if _, ok := discoveryIDFilterSet[association.IpamResourceDiscoveryID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[association.OwnerID]; !ok {
				continue
			}
		}
		if len(resourceDiscoveryStatusFilterSet) > 0 {
			if _, ok := resourceDiscoveryStatusFilterSet[strings.ToLower(association.ResourceDiscoveryStatus)]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(association.State)]; !ok {
				continue
			}
		}
		if len(isDefaultFilterSet) > 0 {
			if !matchesStage118BoolFilterSet(isDefaultFilterSet, association.IsDefault) {
				continue
			}
		}
		if !matchesTagFilters(stage118TagMapFromSlice(association.Tags), tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneIpamResourceDiscoveryAssociation(association))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamResourceDiscoveryAssociation(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpamScopes(scopeIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]IpamScope, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(scopeIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	scopeIDFilterSet := toStringSet(standardFilters["ipam-scope-id"])
	ipamIDFilterSet := toStringSet(standardFilters["ipam-id"])
	scopeTypeFilterSet := toLowerStringSet(standardFilters["ipam-scope-type"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	isDefaultFilterSet := toLowerStringSet(standardFilters["is-default"])
	regionFilterSet := toLowerStringSet(standardFilters["ipam-region"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.ipamScopes))
	for scopeID := range s.ipamScopes {
		ids = append(ids, scopeID)
	}
	sort.Strings(ids)

	items := make([]IpamScope, 0, len(ids))
	for _, scopeID := range ids {
		scope := s.ipamScopes[scopeID]
		if scope == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[scopeID]; !ok {
				continue
			}
		}
		if len(scopeIDFilterSet) > 0 {
			if _, ok := scopeIDFilterSet[scopeID]; !ok {
				continue
			}
		}
		if len(ipamIDFilterSet) > 0 {
			if _, ok := ipamIDFilterSet[scope.IpamID]; !ok {
				continue
			}
		}
		if len(scopeTypeFilterSet) > 0 {
			if _, ok := scopeTypeFilterSet[strings.ToLower(scope.IpamScopeType)]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[scope.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(scope.State)]; !ok {
				continue
			}
		}
		if len(isDefaultFilterSet) > 0 {
			if !matchesStage118BoolFilterSet(isDefaultFilterSet, scope.IsDefault) {
				continue
			}
		}
		if len(regionFilterSet) > 0 {
			if _, ok := regionFilterSet[strings.ToLower(scope.IpamRegion)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(scope.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage108IpamScope(scope))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamScope(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpams(ipamIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]Ipam, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(ipamIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	ipamIDFilterSet := toStringSet(standardFilters["ipam-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	tierFilterSet := toLowerStringSet(standardFilters["tier"])
	meteredAccountFilterSet := toStringSet(standardFilters["metered-account"])

	s.mu.Lock()
	ids := make([]string, 0, len(s.ipams))
	for ipamID := range s.ipams {
		ids = append(ids, ipamID)
	}
	sort.Strings(ids)

	items := make([]Ipam, 0, len(ids))
	for _, ipamID := range ids {
		ipam := s.ipams[ipamID]
		if ipam == nil {
			continue
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[ipamID]; !ok {
				continue
			}
		}
		if len(ipamIDFilterSet) > 0 {
			if _, ok := ipamIDFilterSet[ipamID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[ipam.OwnerID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(ipam.State)]; !ok {
				continue
			}
		}
		if len(tierFilterSet) > 0 {
			if _, ok := tierFilterSet[strings.ToLower(ipam.Tier)]; !ok {
				continue
			}
		}
		if len(meteredAccountFilterSet) > 0 {
			if _, ok := meteredAccountFilterSet[ipam.MeteredAccount]; !ok {
				continue
			}
		}
		if !matchesTagFilters(ipam.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage107Ipam(ipam))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]Ipam(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeIpv6Pools(poolIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]Ipv6Pool, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDs := dedupeTrimmedStrings(poolIDs)
	requestedIDSet := toStringSet(requestedIDs)
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	poolIDFilterSet := toStringSet(standardFilters["pool-id"])
	descriptionFilterSet := toStringSet(standardFilters["description"])

	s.mu.Lock()
	poolMap := map[string]*Ipv6Pool{}
	for _, assoc := range s.vpcIPv6CidrAssociations {
		if assoc == nil {
			continue
		}
		poolID := strings.TrimSpace(assoc.IPv6Pool)
		if poolID == "" {
			continue
		}
		pool := poolMap[poolID]
		if pool == nil {
			pool = &Ipv6Pool{PoolID: poolID, Tags: map[string]string{}}
			poolMap[poolID] = pool
		}
		cidr := strings.TrimSpace(assoc.IPv6CidrBlock)
		if cidr != "" {
			pool.PoolCIDRBlocks = append(pool.PoolCIDRBlocks, cidr)
		}
	}
	ids := make([]string, 0, len(poolMap))
	for poolID := range poolMap {
		ids = append(ids, poolID)
	}
	sort.Strings(ids)

	items := make([]Ipv6Pool, 0, len(ids))
	for _, poolID := range ids {
		pool := poolMap[poolID]
		if pool == nil {
			continue
		}
		pool.PoolCIDRBlocks = dedupeTrimmedStrings(pool.PoolCIDRBlocks)
		sort.Strings(pool.PoolCIDRBlocks)

		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[poolID]; !ok {
				continue
			}
		}
		if len(poolIDFilterSet) > 0 {
			if _, ok := poolIDFilterSet[poolID]; !ok {
				continue
			}
		}
		if len(descriptionFilterSet) > 0 {
			if _, ok := descriptionFilterSet[pool.Description]; !ok {
				continue
			}
		}
		if !matchesTagFilters(pool.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage118Ipv6Pool(pool))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]Ipv6Pool(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeLaunchTemplateVersions(
	launchTemplateID string,
	launchTemplateName string,
	versions []string,
	minVersion *string,
	maxVersion *string,
	resolveAlias *bool,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]LaunchTemplateVersion, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	launchTemplateID = strings.TrimSpace(launchTemplateID)
	launchTemplateName = strings.TrimSpace(launchTemplateName)
	if launchTemplateID == "" && launchTemplateName == "" {
		return nil, nil, ErrInvalidParameter
	}
	if launchTemplateID != "" && launchTemplateName != "" {
		return nil, nil, ErrInvalidParameter
	}
	_ = resolveAlias

	versionSelectors := dedupeTrimmedStrings(versions)
	standardFilters, _, _ := splitEC2Filters(filters)
	launchTemplateIDFilterSet := toStringSet(standardFilters["launch-template-id"])
	launchTemplateNameFilterSet := toStringSet(standardFilters["launch-template-name"])
	versionNumberFilterSet := toStringSet(standardFilters["version-number"])
	defaultVersionFilterSet := toLowerStringSet(append(append([]string{}, standardFilters["default-version"]...), standardFilters["is-default-version"]...))

	s.mu.Lock()
	defer s.mu.Unlock()

	if launchTemplateID == "" {
		launchTemplateID = s.launchTemplateNameIndex[launchTemplateName]
	}
	template := s.launchTemplates[launchTemplateID]
	if template == nil {
		return nil, nil, ErrNotFound
	}
	if launchTemplateName != "" && template.LaunchTemplateName != launchTemplateName {
		return nil, nil, ErrInvalidParameter
	}
	if len(launchTemplateIDFilterSet) > 0 {
		if _, ok := launchTemplateIDFilterSet[template.LaunchTemplateID]; !ok {
			return []LaunchTemplateVersion{}, nil, nil
		}
	}
	if len(launchTemplateNameFilterSet) > 0 {
		if _, ok := launchTemplateNameFilterSet[template.LaunchTemplateName]; !ok {
			return []LaunchTemplateVersion{}, nil, nil
		}
	}

	versionSet := s.launchTemplateVersions[template.LaunchTemplateID]
	if versionSet == nil {
		versionSet = map[int64]*LaunchTemplateVersion{}
	}

	versionNumbers := make([]int64, 0, len(versionSet))
	if len(versionSelectors) > 0 {
		seen := map[int64]struct{}{}
		for _, selector := range versionSelectors {
			versionNumber, ok := resolveDeleteLaunchTemplateVersion(selector, template)
			if !ok {
				return nil, nil, ErrInvalidParameter
			}
			if _, duplicate := seen[versionNumber]; duplicate {
				continue
			}
			seen[versionNumber] = struct{}{}
			versionNumbers = append(versionNumbers, versionNumber)
		}
	} else {
		for versionNumber := range versionSet {
			versionNumbers = append(versionNumbers, versionNumber)
		}
	}
	sort.Slice(versionNumbers, func(i, j int) bool { return versionNumbers[i] < versionNumbers[j] })

	minVersionNumber := int64(0)
	if minVersion != nil {
		resolvedMinVersion, ok := resolveDeleteLaunchTemplateVersion(*minVersion, template)
		if !ok {
			return nil, nil, ErrInvalidParameter
		}
		minVersionNumber = resolvedMinVersion
	}
	maxVersionNumber := int64(^uint64(0) >> 1)
	if maxVersion != nil {
		resolvedMaxVersion, ok := resolveDeleteLaunchTemplateVersion(*maxVersion, template)
		if !ok {
			return nil, nil, ErrInvalidParameter
		}
		maxVersionNumber = resolvedMaxVersion
	}
	if maxVersionNumber < minVersionNumber {
		return nil, nil, ErrInvalidParameter
	}

	items := make([]LaunchTemplateVersion, 0, len(versionNumbers))
	for _, versionNumber := range versionNumbers {
		if versionNumber < minVersionNumber || versionNumber > maxVersionNumber {
			continue
		}
		version := versionSet[versionNumber]
		if version == nil {
			continue
		}
		if len(versionNumberFilterSet) > 0 {
			if _, ok := versionNumberFilterSet[strconv.FormatInt(versionNumber, 10)]; !ok {
				continue
			}
		}
		if len(defaultVersionFilterSet) > 0 {
			if !matchesStage118BoolFilterSet(defaultVersionFilterSet, version.DefaultVersion) {
				continue
			}
		}
		items = append(items, cloneStage108LaunchTemplateVersion(version))
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]LaunchTemplateVersion(nil), items[start:end]...), outputToken, nil
}

func matchesStage118BoolFilterSet(filterSet map[string]struct{}, value bool) bool {
	if len(filterSet) == 0 {
		return true
	}
	if _, ok := filterSet[strconv.FormatBool(value)]; ok {
		return true
	}
	if value {
		if _, ok := filterSet["1"]; ok {
			return true
		}
		if _, ok := filterSet["yes"]; ok {
			return true
		}
		if _, ok := filterSet["on"]; ok {
			return true
		}
	} else {
		if _, ok := filterSet["0"]; ok {
			return true
		}
		if _, ok := filterSet["no"]; ok {
			return true
		}
		if _, ok := filterSet["off"]; ok {
			return true
		}
	}
	return false
}

func stage118TagMapFromSlice(tags []Tag) map[string]string {
	out := map[string]string{}
	for _, tag := range tags {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(tag.Value)
	}
	return out
}

func cloneStage118Ipv6Pool(in *Ipv6Pool) Ipv6Pool {
	if in == nil {
		return Ipv6Pool{}
	}
	return Ipv6Pool{
		Description:    in.Description,
		PoolID:         in.PoolID,
		PoolCIDRBlocks: append([]string(nil), in.PoolCIDRBlocks...),
		Tags:           cloneStringMap(in.Tags),
	}
}
