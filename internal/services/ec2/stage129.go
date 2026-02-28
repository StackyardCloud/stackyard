package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type ModifyManagedPrefixListAddEntry struct {
	CIDR        string
	Description string
}

type ModifyManagedPrefixListRemoveEntry struct {
	CIDR string
}

func (s *Service) ModifyIpam(
	ipamID string,
	description *string,
	enablePrivateGua *bool,
	meteredAccount string,
	addOperatingRegions []string,
	removeOperatingRegions []string,
	tier string,
) (Ipam, error) {
	ipamID = strings.TrimSpace(ipamID)
	meteredAccount = strings.ToLower(strings.TrimSpace(meteredAccount))
	tier = strings.ToLower(strings.TrimSpace(tier))
	if ipamID == "" {
		return Ipam{}, ErrInvalidParameter
	}
	if meteredAccount != "" && meteredAccount != "ipam-owner" && meteredAccount != "resource-owner" && meteredAccount != "disabled" {
		return Ipam{}, ErrInvalidParameter
	}
	if tier != "" && tier != "free" && tier != "advanced" {
		return Ipam{}, ErrInvalidParameter
	}

	addOperatingRegions = dedupeTrimmedStrings(addOperatingRegions)
	removeOperatingRegions = dedupeTrimmedStrings(removeOperatingRegions)
	removeSet := toStringSet(removeOperatingRegions)

	s.mu.Lock()
	defer s.mu.Unlock()

	ipam := s.ipams[ipamID]
	if ipam == nil {
		return Ipam{}, ErrNotFound
	}

	if description != nil {
		ipam.Description = strings.TrimSpace(*description)
	}
	if enablePrivateGua != nil {
		ipam.EnablePrivateGua = *enablePrivateGua
	}
	if meteredAccount != "" {
		ipam.MeteredAccount = meteredAccount
	}
	if tier != "" {
		ipam.Tier = tier
	}

	regions := make([]string, 0, len(ipam.OperatingRegions)+len(addOperatingRegions))
	seen := map[string]struct{}{}
	for _, region := range ipam.OperatingRegions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		if _, removed := removeSet[region]; removed {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}
	for _, region := range addOperatingRegions {
		if _, removed := removeSet[region]; removed {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}
	if len(regions) == 0 {
		regions = []string{DefaultRegion}
	}
	sort.Strings(regions)
	ipam.OperatingRegions = regions

	return cloneStage107Ipam(ipam), nil
}

func (s *Service) ModifyIpamPool(
	ipamPoolID string,
	addAllocationResourceTags []RequestIpamResourceTag,
	allocationDefaultNetmaskLength *int32,
	allocationMaxNetmaskLength *int32,
	allocationMinNetmaskLength *int32,
	autoImport *bool,
	clearAllocationDefaultNetmaskLength *bool,
	description *string,
	removeAllocationResourceTags []RequestIpamResourceTag,
) (IpamPool, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	if ipamPoolID == "" {
		return IpamPool{}, ErrInvalidParameter
	}
	if !stage129ValidRequestIpamResourceTags(addAllocationResourceTags) || !stage129ValidRequestIpamResourceTags(removeAllocationResourceTags) {
		return IpamPool{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pool := s.ipamPools[ipamPoolID]
	if pool == nil {
		return IpamPool{}, ErrNotFound
	}
	if !validStage108Netmask(allocationDefaultNetmaskLength, pool.AddressFamily) ||
		!validStage108Netmask(allocationMaxNetmaskLength, pool.AddressFamily) ||
		!validStage108Netmask(allocationMinNetmaskLength, pool.AddressFamily) {
		return IpamPool{}, ErrInvalidParameter
	}

	newMin := cloneInt32Pointer(pool.AllocationMinNetmaskLength)
	newMax := cloneInt32Pointer(pool.AllocationMaxNetmaskLength)
	if allocationMinNetmaskLength != nil {
		newMin = cloneInt32Pointer(allocationMinNetmaskLength)
	}
	if allocationMaxNetmaskLength != nil {
		newMax = cloneInt32Pointer(allocationMaxNetmaskLength)
	}
	if newMin != nil && newMax != nil && *newMin > *newMax {
		return IpamPool{}, ErrInvalidParameter
	}

	if clearAllocationDefaultNetmaskLength != nil && *clearAllocationDefaultNetmaskLength {
		pool.AllocationDefaultNetmaskLength = nil
	}
	if allocationDefaultNetmaskLength != nil {
		pool.AllocationDefaultNetmaskLength = cloneInt32Pointer(allocationDefaultNetmaskLength)
	}
	if allocationMaxNetmaskLength != nil {
		pool.AllocationMaxNetmaskLength = cloneInt32Pointer(allocationMaxNetmaskLength)
	}
	if allocationMinNetmaskLength != nil {
		pool.AllocationMinNetmaskLength = cloneInt32Pointer(allocationMinNetmaskLength)
	}
	if autoImport != nil {
		pool.AutoImport = *autoImport
	}
	if description != nil {
		pool.Description = strings.TrimSpace(*description)
	}

	return cloneStage108IpamPool(pool), nil
}

func (s *Service) ModifyIpamResourceCidr(
	currentIpamScopeID string,
	monitored bool,
	resourceCidr string,
	resourceID string,
	resourceRegion string,
	destinationIpamScopeID *string,
) (IpamResourceCidr, error) {
	currentIpamScopeID = strings.TrimSpace(currentIpamScopeID)
	resourceCidr = strings.TrimSpace(resourceCidr)
	resourceID = strings.TrimSpace(resourceID)
	resourceRegion = strings.TrimSpace(resourceRegion)
	destinationScopeID := strings.TrimSpace(derefString(destinationIpamScopeID))
	if currentIpamScopeID == "" || resourceCidr == "" || resourceID == "" || resourceRegion == "" {
		return IpamResourceCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentScope := s.ipamScopes[currentIpamScopeID]
	if currentScope == nil {
		return IpamResourceCidr{}, ErrNotFound
	}
	targetScope := currentScope
	if destinationScopeID != "" {
		targetScope = s.ipamScopes[destinationScopeID]
		if targetScope == nil {
			return IpamResourceCidr{}, ErrNotFound
		}
	}

	resourceType := "resource"
	resourceName := resourceID
	resourceOwnerID := DefaultAccountID
	vpcID := ""
	resourceTags := map[string]string{}
	resourceCIDR := resourceCidr

	switch {
	case strings.HasPrefix(resourceID, "subnet-"):
		subnet := s.subnets[resourceID]
		if subnet == nil {
			return IpamResourceCidr{}, ErrNotFound
		}
		if subnet.CidrBlock != resourceCidr {
			return IpamResourceCidr{}, ErrInvalidParameter
		}
		resourceType = "subnet"
		vpcID = subnet.VpcID
		resourceTags = cloneStringMap(subnet.Tags)
	case strings.HasPrefix(resourceID, "vpc-"):
		vpc := s.vpcs[resourceID]
		if vpc == nil {
			return IpamResourceCidr{}, ErrNotFound
		}
		if vpc.CidrBlock != resourceCidr {
			return IpamResourceCidr{}, ErrInvalidParameter
		}
		resourceType = "vpc"
		vpcID = vpc.ID
		resourceTags = cloneStringMap(vpc.Tags)
	case strings.HasPrefix(resourceID, "eni-"):
		networkInterface := s.networkInterfaces[resourceID]
		if networkInterface == nil {
			return IpamResourceCidr{}, ErrNotFound
		}
		resourceType = "network-interface"
		vpcID = networkInterface.VpcID
		resourceTags = cloneStringMap(networkInterface.Tags)
		if resourceCIDR == "" {
			resourceCIDR = firstNonEmptyString(networkInterface.PrivateIP+"/32", networkInterface.PrivateIP)
		}
	case strings.HasPrefix(resourceID, "i-"):
		instance := s.instances[resourceID]
		if instance == nil {
			return IpamResourceCidr{}, ErrNotFound
		}
		resourceType = "instance"
		vpcID = instance.VpcID
		resourceTags = cloneStringMap(instance.Tags)
		if resourceCIDR == "" {
			resourceCIDR = firstNonEmptyString(instance.PrivateIP+"/32", instance.PrivateIP)
		}
	default:
		return IpamResourceCidr{}, ErrNotFound
	}

	managementState := "managed"
	if !monitored {
		managementState = "unmanaged"
	}
	item := IpamResourceCidr{
		AvailabilityZoneID: "",
		ComplianceStatus:   "compliant",
		IpamID:             targetScope.IpamID,
		IpamPoolID:         "",
		IpamScopeID:        targetScope.IpamScopeID,
		ManagementState:    managementState,
		OverlapStatus:      "nonoverlapping",
		ResourceCidr:       resourceCIDR,
		ResourceID:         resourceID,
		ResourceName:       resourceName,
		ResourceOwnerID:    resourceOwnerID,
		ResourceRegion:     resourceRegion,
		ResourceTags:       stage125ResourceTagsFromMap(resourceTags),
		ResourceType:       resourceType,
		VpcID:              vpcID,
	}
	return item, nil
}

func (s *Service) ModifyIpamResourceDiscovery(
	ipamResourceDiscoveryID string,
	addOperatingRegions []string,
	description *string,
	removeOperatingRegions []string,
) (IpamResourceDiscovery, error) {
	ipamResourceDiscoveryID = strings.TrimSpace(ipamResourceDiscoveryID)
	if ipamResourceDiscoveryID == "" {
		return IpamResourceDiscovery{}, ErrInvalidParameter
	}

	addOperatingRegions = dedupeTrimmedStrings(addOperatingRegions)
	removeOperatingRegions = dedupeTrimmedStrings(removeOperatingRegions)
	removeSet := toStringSet(removeOperatingRegions)

	s.mu.Lock()
	defer s.mu.Unlock()

	discovery := s.ipamResourceDiscoveries[ipamResourceDiscoveryID]
	if discovery == nil {
		return IpamResourceDiscovery{}, ErrNotFound
	}

	if description != nil {
		discovery.Description = strings.TrimSpace(*description)
	}

	regions := make([]string, 0, len(discovery.OperatingRegions)+len(addOperatingRegions))
	seen := map[string]struct{}{}
	for _, region := range discovery.OperatingRegions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		if _, removed := removeSet[region]; removed {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}
	for _, region := range addOperatingRegions {
		if _, removed := removeSet[region]; removed {
			continue
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		regions = append(regions, region)
	}
	if len(regions) == 0 {
		regions = []string{DefaultRegion}
	}
	sort.Strings(regions)
	discovery.OperatingRegions = regions

	return cloneStage108IpamResourceDiscovery(discovery), nil
}

func (s *Service) ModifyIpamScope(ipamScopeID string, description *string) (IpamScope, error) {
	ipamScopeID = strings.TrimSpace(ipamScopeID)
	if ipamScopeID == "" {
		return IpamScope{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	scope := s.ipamScopes[ipamScopeID]
	if scope == nil {
		return IpamScope{}, ErrNotFound
	}
	if description != nil {
		scope.Description = strings.TrimSpace(*description)
	}
	return cloneStage108IpamScope(scope), nil
}

func (s *Service) ModifyLaunchTemplate(
	launchTemplateID string,
	launchTemplateName string,
	defaultVersion *string,
) (LaunchTemplate, error) {
	launchTemplateID = strings.TrimSpace(launchTemplateID)
	launchTemplateName = strings.TrimSpace(launchTemplateName)
	if launchTemplateID == "" && launchTemplateName == "" {
		return LaunchTemplate{}, ErrInvalidParameter
	}
	if launchTemplateID != "" && launchTemplateName != "" {
		return LaunchTemplate{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if launchTemplateID == "" {
		launchTemplateID = s.launchTemplateNameIndex[launchTemplateName]
	}
	template := s.launchTemplates[launchTemplateID]
	if template == nil {
		return LaunchTemplate{}, ErrNotFound
	}
	if launchTemplateName != "" && template.LaunchTemplateName != launchTemplateName {
		return LaunchTemplate{}, ErrInvalidParameter
	}

	if defaultVersion != nil {
		rawDefaultVersion := strings.TrimSpace(*defaultVersion)
		if rawDefaultVersion == "" {
			return LaunchTemplate{}, ErrInvalidParameter
		}
		versionNumber := int64(0)
		switch rawDefaultVersion {
		case "$Latest":
			versionNumber = template.LatestVersionNumber
		case "$Default":
			versionNumber = template.DefaultVersionNumber
		default:
			parsed, err := strconv.ParseInt(rawDefaultVersion, 10, 64)
			if err != nil || parsed <= 0 {
				return LaunchTemplate{}, ErrInvalidParameter
			}
			versionNumber = parsed
		}

		versions := s.launchTemplateVersions[template.LaunchTemplateID]
		if versions == nil || versions[versionNumber] == nil {
			return LaunchTemplate{}, ErrNotFound
		}
		template.DefaultVersionNumber = versionNumber
		for version, versionEntry := range versions {
			if versionEntry == nil {
				continue
			}
			versionEntry.DefaultVersion = version == versionNumber
		}
	}

	return cloneStage108LaunchTemplate(template), nil
}

func (s *Service) ModifyLocalGatewayRoute(
	localGatewayRouteTableID string,
	destinationCidrBlock *string,
	destinationPrefixListID *string,
	localGatewayVirtualInterfaceGroupID *string,
	networkInterfaceID *string,
) (LocalGatewayRoute, error) {
	localGatewayRouteTableID = strings.TrimSpace(localGatewayRouteTableID)
	destinationCIDR := strings.TrimSpace(derefString(destinationCidrBlock))
	destinationPrefixListIDValue := strings.TrimSpace(derefString(destinationPrefixListID))
	newVirtualInterfaceGroupID := strings.TrimSpace(derefString(localGatewayVirtualInterfaceGroupID))
	newNetworkInterfaceID := strings.TrimSpace(derefString(networkInterfaceID))
	if localGatewayRouteTableID == "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}
	if destinationCIDR == "" && destinationPrefixListIDValue == "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}
	if destinationCIDR != "" && destinationPrefixListIDValue != "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}
	if newVirtualInterfaceGroupID != "" && newNetworkInterfaceID != "" {
		return LocalGatewayRoute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.localGatewayRouteTables[localGatewayRouteTableID] == nil {
		return LocalGatewayRoute{}, ErrNotFound
	}

	candidateKeys := make([]string, 0)
	for key, route := range s.localGatewayRoutes {
		if route == nil {
			continue
		}
		if route.LocalGatewayRouteTableID != localGatewayRouteTableID {
			continue
		}
		if route.DestinationCidrBlock != destinationCIDR {
			continue
		}
		if route.DestinationPrefixListID != destinationPrefixListIDValue {
			continue
		}
		candidateKeys = append(candidateKeys, key)
	}
	if len(candidateKeys) == 0 {
		return LocalGatewayRoute{}, ErrNotFound
	}
	sort.Strings(candidateKeys)
	oldKey := candidateKeys[0]
	route := s.localGatewayRoutes[oldKey]
	if route == nil {
		return LocalGatewayRoute{}, ErrNotFound
	}

	if localGatewayVirtualInterfaceGroupID != nil {
		route.LocalGatewayVirtualInterfaceGroupID = newVirtualInterfaceGroupID
	}
	if networkInterfaceID != nil {
		route.NetworkInterfaceID = newNetworkInterfaceID
	}

	newKey := strings.Join([]string{
		route.LocalGatewayRouteTableID,
		route.DestinationCidrBlock,
		route.DestinationPrefixListID,
		route.LocalGatewayVirtualInterfaceGroupID,
		route.NetworkInterfaceID,
	}, "|")
	if newKey != oldKey {
		if _, exists := s.localGatewayRoutes[newKey]; exists {
			return LocalGatewayRoute{}, ErrAlreadyExists
		}
		delete(s.localGatewayRoutes, oldKey)
		s.localGatewayRoutes[newKey] = route
	}

	return cloneStage108LocalGatewayRoute(route), nil
}

func (s *Service) ModifyManagedPrefixList(
	prefixListID string,
	addEntries []ModifyManagedPrefixListAddEntry,
	currentVersion *int64,
	maxEntries *int32,
	prefixListName *string,
	removeEntries []ModifyManagedPrefixListRemoveEntry,
) (ManagedPrefixList, error) {
	prefixListID = strings.TrimSpace(prefixListID)
	if prefixListID == "" {
		return ManagedPrefixList{}, ErrInvalidParameter
	}
	if maxEntries != nil && *maxEntries <= 0 {
		return ManagedPrefixList{}, ErrInvalidParameter
	}

	addEntries = stage129NormalizeModifyManagedPrefixListAddEntries(addEntries)
	removeEntries = stage129NormalizeModifyManagedPrefixListRemoveEntries(removeEntries)
	if maxEntries != nil && (len(addEntries) > 0 || len(removeEntries) > 0) {
		return ManagedPrefixList{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prefixList := s.managedPrefixLists[prefixListID]
	if prefixList == nil {
		return ManagedPrefixList{}, ErrNotFound
	}

	if currentVersion != nil && *currentVersion != prefixList.Version {
		return ManagedPrefixList{}, ErrInvalidParameter
	}
	if prefixListName != nil {
		name := strings.TrimSpace(*prefixListName)
		if name == "" {
			return ManagedPrefixList{}, ErrInvalidParameter
		}
		prefixList.PrefixListName = name
	}
	if maxEntries != nil {
		prefixList.MaxEntries = *maxEntries
	}
	if len(addEntries) > 0 || len(removeEntries) > 0 {
		prefixList.Version++
	}
	return cloneStage109ManagedPrefixList(prefixList), nil
}

func (s *Service) ModifyPrivateDnsNameOptions(
	instanceID string,
	enableResourceNameDnsARecord *bool,
	enableResourceNameDnsAAAARecord *bool,
	privateDnsHostnameType *string,
) (bool, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return false, ErrInvalidParameter
	}
	if privateDnsHostnameType != nil {
		hostnameType := strings.ToLower(strings.TrimSpace(*privateDnsHostnameType))
		if hostnameType != "ip-name" && hostnameType != "resource-name" {
			return false, ErrInvalidParameter
		}
	}

	_ = enableResourceNameDnsARecord
	_ = enableResourceNameDnsAAAARecord

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.instances[instanceID] == nil {
		return false, ErrNotFound
	}
	return true, nil
}

func (s *Service) ModifyPublicIpDnsNameOptions(networkInterfaceID, hostnameType string) (bool, error) {
	networkInterfaceID = strings.TrimSpace(networkInterfaceID)
	hostnameType = strings.ToLower(strings.TrimSpace(hostnameType))
	if networkInterfaceID == "" || hostnameType == "" {
		return false, ErrInvalidParameter
	}
	switch hostnameType {
	case "public-ipv4-dns-name", "public-ipv6-dns-name", "public-dual-stack-dns-name":
	default:
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.networkInterfaces[networkInterfaceID] == nil {
		return false, ErrNotFound
	}
	return true, nil
}

func stage129ValidRequestIpamResourceTags(tags []RequestIpamResourceTag) bool {
	for _, tag := range tags {
		if strings.TrimSpace(tag.Key) == "" {
			return false
		}
	}
	return true
}

func stage129NormalizeModifyManagedPrefixListAddEntries(entries []ModifyManagedPrefixListAddEntry) []ModifyManagedPrefixListAddEntry {
	if len(entries) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]ModifyManagedPrefixListAddEntry, 0, len(entries))
	for _, entry := range entries {
		cidr := strings.TrimSpace(entry.CIDR)
		if cidr == "" {
			continue
		}
		if _, ok := seen[cidr]; ok {
			continue
		}
		seen[cidr] = struct{}{}
		out = append(out, ModifyManagedPrefixListAddEntry{CIDR: cidr, Description: strings.TrimSpace(entry.Description)})
	}
	return out
}

func stage129NormalizeModifyManagedPrefixListRemoveEntries(entries []ModifyManagedPrefixListRemoveEntry) []ModifyManagedPrefixListRemoveEntry {
	if len(entries) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]ModifyManagedPrefixListRemoveEntry, 0, len(entries))
	for _, entry := range entries {
		cidr := strings.TrimSpace(entry.CIDR)
		if cidr == "" {
			continue
		}
		if _, ok := seen[cidr]; ok {
			continue
		}
		seen[cidr] = struct{}{}
		out = append(out, ModifyManagedPrefixListRemoveEntry{CIDR: cidr})
	}
	return out
}
