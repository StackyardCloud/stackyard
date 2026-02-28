package ec2

import (
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
	"time"
)

type InstanceUefiData struct {
	InstanceID string
	UefiData   string
}

type IpamDiscoveryFailureReason struct {
	Code    string
	Message string
}

type IpamAddressHistoryRecord struct {
	ResourceCidr             string
	ResourceComplianceStatus string
	ResourceID               string
	ResourceName             string
	ResourceOverlapStatus    string
	ResourceOwnerID          string
	ResourceRegion           string
	ResourceType             string
	SampledEndTime           *time.Time
	SampledStartTime         *time.Time
	VpcID                    string
}

type IpamDiscoveredAccount struct {
	AccountID                   string
	DiscoveryRegion             string
	FailureReason               *IpamDiscoveryFailureReason
	LastAttemptedDiscoveryTime  *time.Time
	LastSuccessfulDiscoveryTime *time.Time
	OrganizationalUnitID        string
}

type IpamPublicAddressSecurityGroup struct {
	GroupID   string
	GroupName string
}

type IpamPublicAddressTag struct {
	Key   string
	Value string
}

type IpamDiscoveredPublicAddress struct {
	Address                     string
	AddressAllocationID         string
	AddressOwnerID              string
	AddressRegion               string
	AddressType                 string
	AssociationStatus           string
	InstanceID                  string
	IpamResourceDiscoveryID     string
	NetworkBorderGroup          string
	NetworkInterfaceDescription string
	NetworkInterfaceID          string
	PublicIpv4PoolID            string
	SampleTime                  *time.Time
	SecurityGroups              []IpamPublicAddressSecurityGroup
	Service                     string
	ServiceResource             string
	SubnetID                    string
	Tags                        []IpamPublicAddressTag
	VpcID                       string
}

type IpamResourceTag struct {
	Key   string
	Value string
}

type IpamDiscoveredResourceCidr struct {
	AvailabilityZoneID               string
	IpamResourceDiscoveryID          string
	IpSource                         string
	IpUsage                          *float64
	NetworkInterfaceAttachmentStatus string
	ResourceCidr                     string
	ResourceID                       string
	ResourceOwnerID                  string
	ResourceRegion                   string
	ResourceTags                     []IpamResourceTag
	ResourceType                     string
	SampleTime                       *time.Time
	SubnetID                         string
	VpcID                            string
}

type IpamResourceCidr struct {
	AvailabilityZoneID string
	ComplianceStatus   string
	IpamID             string
	IpamPoolID         string
	IpamScopeID        string
	IpUsage            *float64
	ManagementState    string
	OverlapStatus      string
	ResourceCidr       string
	ResourceID         string
	ResourceName       string
	ResourceOwnerID    string
	ResourceRegion     string
	ResourceTags       []IpamResourceTag
	ResourceType       string
	VpcID              string
}

type RequestIpamResourceTag struct {
	Key   string
	Value string
}

type LaunchTemplateDataResponse struct {
	ImageID          string
	InstanceType     string
	KeyName          string
	UserData         string
	SecurityGroupIDs []string
	SecurityGroups   []string
}

type PrefixListAssociation struct {
	ResourceID    string
	ResourceOwner string
}

func (s *Service) GetInstanceUefiData(instanceID string) (InstanceUefiData, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return InstanceUefiData{}, ErrInvalidParameter
	}

	s.mu.Lock()
	instance := s.instances[instanceID]
	s.mu.Unlock()
	if instance == nil {
		return InstanceUefiData{}, ErrNotFound
	}

	encoded := base64.StdEncoding.EncodeToString([]byte("stackyard:uefi:" + instanceID))
	return InstanceUefiData{
		InstanceID: instanceID,
		UefiData:   encoded,
	}, nil
}

func (s *Service) GetIpamAddressHistory(
	cidr string,
	ipamScopeID string,
	vpcID string,
	startTime *time.Time,
	endTime *time.Time,
	maxResults *int32,
	nextToken *string,
) ([]IpamAddressHistoryRecord, *string, error) {
	cidr = strings.TrimSpace(cidr)
	ipamScopeID = strings.TrimSpace(ipamScopeID)
	vpcID = strings.TrimSpace(vpcID)
	if cidr == "" || ipamScopeID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	var sampledStart time.Time
	var sampledEnd time.Time
	now := time.Now().UTC()
	if startTime != nil {
		sampledStart = startTime.UTC()
	} else {
		sampledStart = now.Add(-1 * time.Hour)
	}
	if endTime != nil {
		sampledEnd = endTime.UTC()
	} else {
		sampledEnd = now
	}
	if sampledStart.After(sampledEnd) {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	if s.ipamScopes[ipamScopeID] == nil {
		s.mu.Unlock()
		return nil, nil, ErrNotFound
	}

	instanceIDs := make([]string, 0, len(s.instances))
	for instanceID := range s.instances {
		instanceIDs = append(instanceIDs, instanceID)
	}
	sort.Strings(instanceIDs)

	items := make([]IpamAddressHistoryRecord, 0, len(instanceIDs))
	for _, instanceID := range instanceIDs {
		instance := s.instances[instanceID]
		if instance == nil {
			continue
		}
		if vpcID != "" && instance.VpcID != vpcID {
			continue
		}
		startCopy := sampledStart
		endCopy := sampledEnd
		items = append(items, IpamAddressHistoryRecord{
			ResourceCidr:             cidr,
			ResourceComplianceStatus: "compliant",
			ResourceID:               instance.ID,
			ResourceName:             instance.ID,
			ResourceOverlapStatus:    "nonoverlapping",
			ResourceOwnerID:          DefaultAccountID,
			ResourceRegion:           DefaultRegion,
			ResourceType:             "instance",
			SampledStartTime:         &startCopy,
			SampledEndTime:           &endCopy,
			VpcID:                    instance.VpcID,
		})
	}

	if len(items) == 0 {
		startCopy := sampledStart
		endCopy := sampledEnd
		items = append(items, IpamAddressHistoryRecord{
			ResourceCidr:             cidr,
			ResourceComplianceStatus: "compliant",
			ResourceID:               "vpc-cidr-association",
			ResourceName:             "vpc-cidr-association",
			ResourceOverlapStatus:    "nonoverlapping",
			ResourceOwnerID:          DefaultAccountID,
			ResourceRegion:           DefaultRegion,
			ResourceType:             "vpc",
			SampledStartTime:         &startCopy,
			SampledEndTime:           &endCopy,
			VpcID:                    vpcID,
		})
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamAddressHistoryRecord(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetIpamDiscoveredAccounts(
	discoveryRegion string,
	ipamResourceDiscoveryID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]IpamDiscoveredAccount, *string, error) {
	ipamResourceDiscoveryID = strings.TrimSpace(ipamResourceDiscoveryID)
	discoveryRegion = strings.TrimSpace(discoveryRegion)
	if ipamResourceDiscoveryID == "" {
		return nil, nil, ErrInvalidParameter
	}
	if discoveryRegion == "" {
		discoveryRegion = DefaultRegion
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	accountIDFilterSet := toStringSet(standardFilters["account-id"])
	discoveryRegionFilterSet := toLowerStringSet(standardFilters["discovery-region"])
	organizationalUnitIDFilterSet := toStringSet(standardFilters["organizational-unit-id"])

	now := time.Now().UTC()
	lastAttempted := now.Add(-2 * time.Minute)
	lastSuccessful := now.Add(-1 * time.Minute)

	s.mu.Lock()
	discovery := s.ipamResourceDiscoveries[ipamResourceDiscoveryID]
	s.mu.Unlock()
	if discovery == nil {
		return nil, nil, ErrNotFound
	}

	account := IpamDiscoveredAccount{
		AccountID:                   DefaultAccountID,
		DiscoveryRegion:             discoveryRegion,
		FailureReason:               nil,
		LastAttemptedDiscoveryTime:  &lastAttempted,
		LastSuccessfulDiscoveryTime: &lastSuccessful,
		OrganizationalUnitID:        "ou-stackyard",
	}

	if len(accountIDFilterSet) > 0 {
		if _, ok := accountIDFilterSet[account.AccountID]; !ok {
			return []IpamDiscoveredAccount{}, nil, nil
		}
	}
	if len(discoveryRegionFilterSet) > 0 {
		if _, ok := discoveryRegionFilterSet[strings.ToLower(account.DiscoveryRegion)]; !ok {
			return []IpamDiscoveredAccount{}, nil, nil
		}
	}
	if len(organizationalUnitIDFilterSet) > 0 {
		if _, ok := organizationalUnitIDFilterSet[account.OrganizationalUnitID]; !ok {
			return []IpamDiscoveredAccount{}, nil, nil
		}
	}

	items := []IpamDiscoveredAccount{account}
	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamDiscoveredAccount(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetIpamDiscoveredPublicAddresses(
	addressRegion string,
	ipamResourceDiscoveryID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]IpamDiscoveredPublicAddress, *time.Time, *string, error) {
	ipamResourceDiscoveryID = strings.TrimSpace(ipamResourceDiscoveryID)
	addressRegion = strings.TrimSpace(addressRegion)
	if ipamResourceDiscoveryID == "" {
		return nil, nil, nil, ErrInvalidParameter
	}
	if addressRegion == "" {
		addressRegion = DefaultRegion
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, nil, ErrInvalidParameter
	}

	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	addressFilterSet := toStringSet(standardFilters["address"])
	allocationIDFilterSet := toStringSet(standardFilters["address-allocation-id"])
	associationStatusFilterSet := toLowerStringSet(standardFilters["association-status"])
	instanceIDFilterSet := toStringSet(standardFilters["instance-id"])
	networkInterfaceIDFilterSet := toStringSet(standardFilters["network-interface-id"])
	serviceFilterSet := toLowerStringSet(standardFilters["service"])
	vpcIDFilterSet := toStringSet(standardFilters["vpc-id"])
	subnetIDFilterSet := toStringSet(standardFilters["subnet-id"])
	addressRegionFilterSet := toLowerStringSet(standardFilters["address-region"])

	now := time.Now().UTC()

	s.mu.Lock()
	if s.ipamResourceDiscoveries[ipamResourceDiscoveryID] == nil {
		s.mu.Unlock()
		return nil, nil, nil, ErrNotFound
	}

	allocationIDs := make([]string, 0, len(s.addresses))
	for allocationID := range s.addresses {
		allocationIDs = append(allocationIDs, allocationID)
	}
	sort.Strings(allocationIDs)

	items := make([]IpamDiscoveredPublicAddress, 0, len(allocationIDs))
	for _, allocationID := range allocationIDs {
		address := s.addresses[allocationID]
		if address == nil {
			continue
		}

		associationStatus := "unassociated"
		if strings.TrimSpace(address.AssociationID) != "" {
			associationStatus = "associated"
		}

		instanceID := strings.TrimSpace(address.InstanceID)
		networkInterfaceID := strings.TrimSpace(address.NetworkInterfaceID)
		subnetID := ""
		vpcID := ""
		securityGroups := []IpamPublicAddressSecurityGroup{}
		if instanceID != "" {
			if instance := s.instances[instanceID]; instance != nil {
				subnetID = instance.SubnetID
				vpcID = instance.VpcID
				for _, securityGroupID := range instance.SecurityGroupIDs {
					securityGroupID = strings.TrimSpace(securityGroupID)
					if securityGroupID == "" {
						continue
					}
					groupName := securityGroupID
					if group := s.securityGroups[securityGroupID]; group != nil && strings.TrimSpace(group.Name) != "" {
						groupName = group.Name
					}
					securityGroups = append(securityGroups, IpamPublicAddressSecurityGroup{
						GroupID:   securityGroupID,
						GroupName: groupName,
					})
				}
			}
		}
		if networkInterfaceID != "" {
			if iface := s.networkInterfaces[networkInterfaceID]; iface != nil {
				if subnetID == "" {
					subnetID = iface.SubnetID
				}
				if vpcID == "" {
					vpcID = iface.VpcID
				}
				for _, securityGroupID := range iface.GroupIDs {
					securityGroupID = strings.TrimSpace(securityGroupID)
					if securityGroupID == "" {
						continue
					}
					groupName := securityGroupID
					if group := s.securityGroups[securityGroupID]; group != nil && strings.TrimSpace(group.Name) != "" {
						groupName = group.Name
					}
					securityGroups = append(securityGroups, IpamPublicAddressSecurityGroup{
						GroupID:   securityGroupID,
						GroupName: groupName,
					})
				}
			}
		}

		tags := make([]IpamPublicAddressTag, 0, len(address.Tags))
		for key, value := range address.Tags {
			tags = append(tags, IpamPublicAddressTag{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

		item := IpamDiscoveredPublicAddress{
			Address:                     address.PublicIP,
			AddressAllocationID:         address.AllocationID,
			AddressOwnerID:              DefaultAccountID,
			AddressRegion:               addressRegion,
			AddressType:                 "eip",
			AssociationStatus:           associationStatus,
			InstanceID:                  instanceID,
			IpamResourceDiscoveryID:     ipamResourceDiscoveryID,
			NetworkBorderGroup:          strings.TrimSpace(address.NetworkBorderGroup),
			NetworkInterfaceDescription: "",
			NetworkInterfaceID:          networkInterfaceID,
			PublicIpv4PoolID:            "",
			SampleTime:                  cloneTimePointer(&now),
			SecurityGroups:              dedupeStage125PublicAddressSecurityGroups(securityGroups),
			Service:                     "ec2",
			ServiceResource:             address.AllocationID,
			SubnetID:                    subnetID,
			Tags:                        tags,
			VpcID:                       vpcID,
		}
		if networkInterfaceID != "" {
			if iface := s.networkInterfaces[networkInterfaceID]; iface != nil {
				item.NetworkInterfaceDescription = iface.Description
			}
		}

		if len(addressFilterSet) > 0 {
			if _, ok := addressFilterSet[item.Address]; !ok {
				continue
			}
		}
		if len(allocationIDFilterSet) > 0 {
			if _, ok := allocationIDFilterSet[item.AddressAllocationID]; !ok {
				continue
			}
		}
		if len(associationStatusFilterSet) > 0 {
			if _, ok := associationStatusFilterSet[strings.ToLower(item.AssociationStatus)]; !ok {
				continue
			}
		}
		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[item.InstanceID]; !ok {
				continue
			}
		}
		if len(networkInterfaceIDFilterSet) > 0 {
			if _, ok := networkInterfaceIDFilterSet[item.NetworkInterfaceID]; !ok {
				continue
			}
		}
		if len(serviceFilterSet) > 0 {
			if _, ok := serviceFilterSet[strings.ToLower(item.Service)]; !ok {
				continue
			}
		}
		if len(vpcIDFilterSet) > 0 {
			if _, ok := vpcIDFilterSet[item.VpcID]; !ok {
				continue
			}
		}
		if len(subnetIDFilterSet) > 0 {
			if _, ok := subnetIDFilterSet[item.SubnetID]; !ok {
				continue
			}
		}
		if len(addressRegionFilterSet) > 0 {
			if _, ok := addressRegionFilterSet[strings.ToLower(item.AddressRegion)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(stage125PublicAddressTagMap(item.Tags), tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, item)
	}
	s.mu.Unlock()

	if len(items) == 0 {
		nowCopy := now
		items = append(items, IpamDiscoveredPublicAddress{
			Address:                 "198.51.100.10",
			AddressAllocationID:     "eipalloc-stage125",
			AddressOwnerID:          DefaultAccountID,
			AddressRegion:           addressRegion,
			AddressType:             "eip",
			AssociationStatus:       "unassociated",
			IpamResourceDiscoveryID: ipamResourceDiscoveryID,
			SampleTime:              &nowCopy,
			Service:                 "ec2",
			ServiceResource:         "eipalloc-stage125",
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, nil, ErrInvalidParameter
	}
	return append([]IpamDiscoveredPublicAddress(nil), items[start:end]...), &now, outputToken, nil
}

func (s *Service) GetIpamDiscoveredResourceCidrs(
	ipamResourceDiscoveryID string,
	resourceRegion string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]IpamDiscoveredResourceCidr, *string, error) {
	ipamResourceDiscoveryID = strings.TrimSpace(ipamResourceDiscoveryID)
	resourceRegion = strings.TrimSpace(resourceRegion)
	if ipamResourceDiscoveryID == "" {
		return nil, nil, ErrInvalidParameter
	}
	if resourceRegion == "" {
		resourceRegion = DefaultRegion
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	resourceIDFilterSet := toStringSet(standardFilters["resource-id"])
	resourceTypeFilterSet := toLowerStringSet(standardFilters["resource-type"])
	resourceRegionFilterSet := toLowerStringSet(standardFilters["resource-region"])
	resourceCidrFilterSet := toStringSet(standardFilters["resource-cidr"])
	vpcIDFilterSet := toStringSet(standardFilters["vpc-id"])
	subnetIDFilterSet := toStringSet(standardFilters["subnet-id"])

	now := time.Now().UTC()
	ipUsage := 0.5

	s.mu.Lock()
	if s.ipamResourceDiscoveries[ipamResourceDiscoveryID] == nil {
		s.mu.Unlock()
		return nil, nil, ErrNotFound
	}

	subnetIDs := make([]string, 0, len(s.subnets))
	for subnetID := range s.subnets {
		subnetIDs = append(subnetIDs, subnetID)
	}
	sort.Strings(subnetIDs)

	items := make([]IpamDiscoveredResourceCidr, 0, len(subnetIDs))
	for _, subnetID := range subnetIDs {
		subnet := s.subnets[subnetID]
		if subnet == nil {
			continue
		}
		item := IpamDiscoveredResourceCidr{
			AvailabilityZoneID:               "",
			IpamResourceDiscoveryID:          ipamResourceDiscoveryID,
			IpSource:                         "amazon",
			IpUsage:                          cloneFloat64Pointer(&ipUsage),
			NetworkInterfaceAttachmentStatus: "",
			ResourceCidr:                     subnet.CidrBlock,
			ResourceID:                       subnet.ID,
			ResourceOwnerID:                  DefaultAccountID,
			ResourceRegion:                   resourceRegion,
			ResourceTags:                     stage125ResourceTagsFromMap(subnet.Tags),
			ResourceType:                     "subnet",
			SampleTime:                       cloneTimePointer(&now),
			SubnetID:                         subnet.ID,
			VpcID:                            subnet.VpcID,
		}

		if len(resourceIDFilterSet) > 0 {
			if _, ok := resourceIDFilterSet[item.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceTypeFilterSet) > 0 {
			if _, ok := resourceTypeFilterSet[strings.ToLower(item.ResourceType)]; !ok {
				continue
			}
		}
		if len(resourceRegionFilterSet) > 0 {
			if _, ok := resourceRegionFilterSet[strings.ToLower(item.ResourceRegion)]; !ok {
				continue
			}
		}
		if len(resourceCidrFilterSet) > 0 {
			if _, ok := resourceCidrFilterSet[item.ResourceCidr]; !ok {
				continue
			}
		}
		if len(vpcIDFilterSet) > 0 {
			if _, ok := vpcIDFilterSet[item.VpcID]; !ok {
				continue
			}
		}
		if len(subnetIDFilterSet) > 0 {
			if _, ok := subnetIDFilterSet[item.SubnetID]; !ok {
				continue
			}
		}
		items = append(items, item)
	}
	s.mu.Unlock()

	if len(items) == 0 {
		items = append(items, IpamDiscoveredResourceCidr{
			IpamResourceDiscoveryID: ipamResourceDiscoveryID,
			IpSource:                "amazon",
			ResourceCidr:            "10.0.0.0/16",
			ResourceID:              defaultVPCID,
			ResourceOwnerID:         DefaultAccountID,
			ResourceRegion:          resourceRegion,
			ResourceType:            "vpc",
			SampleTime:              cloneTimePointer(&now),
			VpcID:                   defaultVPCID,
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamDiscoveredResourceCidr(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetIpamPoolAllocations(
	ipamPoolID string,
	ipamPoolAllocationID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]IpamPoolAllocation, *string, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	ipamPoolAllocationID = strings.TrimSpace(ipamPoolAllocationID)
	if ipamPoolID == "" && ipamPoolAllocationID == "" {
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
	allocationIDFilterSet := toStringSet(standardFilters["ipam-pool-allocation-id"])
	cidrFilterSet := toStringSet(standardFilters["cidr"])
	resourceIDFilterSet := toStringSet(standardFilters["resource-id"])
	resourceOwnerFilterSet := toStringSet(standardFilters["resource-owner"])
	resourceRegionFilterSet := toStringSet(standardFilters["resource-region"])
	resourceTypeFilterSet := toLowerStringSet(standardFilters["resource-type"])

	s.mu.Lock()
	defer s.mu.Unlock()

	if ipamPoolID != "" && s.ipamPools[ipamPoolID] == nil {
		return nil, nil, ErrNotFound
	}

	allocationIDs := make([]string, 0, len(s.ipamPoolAllocations))
	for allocationID := range s.ipamPoolAllocations {
		allocationIDs = append(allocationIDs, allocationID)
	}
	sort.Strings(allocationIDs)

	items := make([]IpamPoolAllocation, 0, len(allocationIDs))
	for _, allocationID := range allocationIDs {
		allocation := s.ipamPoolAllocations[allocationID]
		if allocation == nil {
			continue
		}
		if ipamPoolID != "" && allocation.ResourceID != ipamPoolID {
			continue
		}
		if ipamPoolAllocationID != "" && allocation.IpamPoolAllocationID != ipamPoolAllocationID {
			continue
		}
		if len(allocationIDFilterSet) > 0 {
			if _, ok := allocationIDFilterSet[allocation.IpamPoolAllocationID]; !ok {
				continue
			}
		}
		if len(cidrFilterSet) > 0 {
			if _, ok := cidrFilterSet[allocation.Cidr]; !ok {
				continue
			}
		}
		if len(resourceIDFilterSet) > 0 {
			if _, ok := resourceIDFilterSet[allocation.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceOwnerFilterSet) > 0 {
			if _, ok := resourceOwnerFilterSet[allocation.ResourceOwner]; !ok {
				continue
			}
		}
		if len(resourceRegionFilterSet) > 0 {
			if _, ok := resourceRegionFilterSet[allocation.ResourceRegion]; !ok {
				continue
			}
		}
		if len(resourceTypeFilterSet) > 0 {
			if _, ok := resourceTypeFilterSet[strings.ToLower(allocation.ResourceType)]; !ok {
				continue
			}
		}
		items = append(items, *allocation)
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamPoolAllocation(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetIpamPoolCidrs(
	ipamPoolID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]IpamPoolCidr, *string, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	if ipamPoolID == "" {
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
	cidrFilterSet := toStringSet(standardFilters["cidr"])
	ipamPoolCidrIDFilterSet := toStringSet(standardFilters["ipam-pool-cidr-id"])
	stateFilterSet := toLowerStringSet(standardFilters["state"])
	netmaskLengthFilterSet := toStringSet(standardFilters["netmask-length"])

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ipamPools[ipamPoolID] == nil {
		return nil, nil, ErrNotFound
	}

	allocationIDs := make([]string, 0, len(s.ipamPoolAllocations))
	for allocationID := range s.ipamPoolAllocations {
		allocationIDs = append(allocationIDs, allocationID)
	}
	sort.Strings(allocationIDs)

	items := make([]IpamPoolCidr, 0, len(allocationIDs))
	for _, allocationID := range allocationIDs {
		allocation := s.ipamPoolAllocations[allocationID]
		if allocation == nil || allocation.ResourceID != ipamPoolID {
			continue
		}
		netmaskLength := stage125NetmaskFromCIDR(allocation.Cidr)
		item := IpamPoolCidr{
			Cidr:           allocation.Cidr,
			FailureReason:  nil,
			IpamPoolCidrID: allocation.IpamPoolAllocationID,
			NetmaskLength:  netmaskLength,
			State:          "provisioned",
		}
		if len(cidrFilterSet) > 0 {
			if _, ok := cidrFilterSet[item.Cidr]; !ok {
				continue
			}
		}
		if len(ipamPoolCidrIDFilterSet) > 0 {
			if _, ok := ipamPoolCidrIDFilterSet[item.IpamPoolCidrID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(item.State)]; !ok {
				continue
			}
		}
		if len(netmaskLengthFilterSet) > 0 && item.NetmaskLength != nil {
			if _, ok := netmaskLengthFilterSet[strconv.FormatInt(int64(*item.NetmaskLength), 10)]; !ok {
				continue
			}
		}
		items = append(items, item)
	}

	if len(items) == 0 {
		defaultCIDR := "10.0.0.0/24"
		if pool := s.ipamPools[ipamPoolID]; pool != nil && strings.EqualFold(pool.AddressFamily, "ipv6") {
			defaultCIDR = "2001:db8::/64"
		}
		items = append(items, IpamPoolCidr{
			Cidr:           defaultCIDR,
			IpamPoolCidrID: "ipam-pool-cidr-stage125",
			NetmaskLength:  stage125NetmaskFromCIDR(defaultCIDR),
			State:          "provisioned",
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamPoolCidr(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetIpamResourceCidrs(
	ipamPoolID string,
	ipamScopeID string,
	resourceID string,
	resourceOwner string,
	resourceTag *RequestIpamResourceTag,
	resourceType string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]IpamResourceCidr, *string, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	ipamScopeID = strings.TrimSpace(ipamScopeID)
	resourceID = strings.TrimSpace(resourceID)
	resourceOwner = strings.TrimSpace(resourceOwner)
	resourceType = strings.TrimSpace(resourceType)
	if resourceOwner == "" {
		resourceOwner = DefaultAccountID
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	resourceIDFilterSet := toStringSet(standardFilters["resource-id"])
	resourceOwnerFilterSet := toStringSet(standardFilters["resource-owner"])
	resourceTypeFilterSet := toLowerStringSet(standardFilters["resource-type"])
	resourceRegionFilterSet := toLowerStringSet(standardFilters["resource-region"])
	vpcIDFilterSet := toStringSet(standardFilters["vpc-id"])
	resourceCidrFilterSet := toStringSet(standardFilters["resource-cidr"])
	ipamPoolIDFilterSet := toStringSet(standardFilters["ipam-pool-id"])
	ipamScopeIDFilterSet := toStringSet(standardFilters["ipam-scope-id"])

	tagKey := ""
	tagValue := ""
	if resourceTag != nil {
		tagKey = strings.TrimSpace(resourceTag.Key)
		tagValue = strings.TrimSpace(resourceTag.Value)
	}

	ipUsage := 0.5
	s.mu.Lock()
	defer s.mu.Unlock()

	pool := (*IpamPool)(nil)
	if ipamPoolID != "" {
		pool = s.ipamPools[ipamPoolID]
		if pool == nil {
			return nil, nil, ErrNotFound
		}
	}
	scope := (*IpamScope)(nil)
	if ipamScopeID != "" {
		scope = s.ipamScopes[ipamScopeID]
		if scope == nil {
			return nil, nil, ErrNotFound
		}
	}
	if pool != nil && scope == nil && pool.IpamScopeID != "" {
		scope = s.ipamScopes[pool.IpamScopeID]
	}
	effectiveIpamPoolID := ipamPoolID
	if effectiveIpamPoolID == "" && pool != nil {
		effectiveIpamPoolID = pool.IpamPoolID
	}
	effectiveIpamScopeID := ipamScopeID
	if effectiveIpamScopeID == "" && scope != nil {
		effectiveIpamScopeID = scope.IpamScopeID
	}
	effectiveIpamID := ""
	if scope != nil {
		effectiveIpamID = scope.IpamID
	}
	if effectiveIpamID == "" && pool != nil {
		if scopeFromPool := s.ipamScopes[pool.IpamScopeID]; scopeFromPool != nil {
			effectiveIpamID = scopeFromPool.IpamID
			if effectiveIpamScopeID == "" {
				effectiveIpamScopeID = scopeFromPool.IpamScopeID
			}
		}
	}

	subnetIDs := make([]string, 0, len(s.subnets))
	for subnetID := range s.subnets {
		subnetIDs = append(subnetIDs, subnetID)
	}
	sort.Strings(subnetIDs)

	items := make([]IpamResourceCidr, 0, len(subnetIDs))
	for _, subnetID := range subnetIDs {
		subnet := s.subnets[subnetID]
		if subnet == nil {
			continue
		}

		itemResourceType := "subnet"
		if resourceType != "" && !strings.EqualFold(resourceType, itemResourceType) {
			continue
		}

		item := IpamResourceCidr{
			AvailabilityZoneID: "",
			ComplianceStatus:   "compliant",
			IpamID:             effectiveIpamID,
			IpamPoolID:         effectiveIpamPoolID,
			IpamScopeID:        effectiveIpamScopeID,
			IpUsage:            cloneFloat64Pointer(&ipUsage),
			ManagementState:    "managed",
			OverlapStatus:      "nonoverlapping",
			ResourceCidr:       subnet.CidrBlock,
			ResourceID:         subnet.ID,
			ResourceName:       subnet.ID,
			ResourceOwnerID:    resourceOwner,
			ResourceRegion:     DefaultRegion,
			ResourceTags:       stage125ResourceTagsFromMap(subnet.Tags),
			ResourceType:       itemResourceType,
			VpcID:              subnet.VpcID,
		}

		if resourceID != "" && item.ResourceID != resourceID {
			continue
		}
		if len(resourceIDFilterSet) > 0 {
			if _, ok := resourceIDFilterSet[item.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceOwnerFilterSet) > 0 {
			if _, ok := resourceOwnerFilterSet[item.ResourceOwnerID]; !ok {
				continue
			}
		}
		if len(resourceTypeFilterSet) > 0 {
			if _, ok := resourceTypeFilterSet[strings.ToLower(item.ResourceType)]; !ok {
				continue
			}
		}
		if len(resourceRegionFilterSet) > 0 {
			if _, ok := resourceRegionFilterSet[strings.ToLower(item.ResourceRegion)]; !ok {
				continue
			}
		}
		if len(vpcIDFilterSet) > 0 {
			if _, ok := vpcIDFilterSet[item.VpcID]; !ok {
				continue
			}
		}
		if len(resourceCidrFilterSet) > 0 {
			if _, ok := resourceCidrFilterSet[item.ResourceCidr]; !ok {
				continue
			}
		}
		if len(ipamPoolIDFilterSet) > 0 {
			if _, ok := ipamPoolIDFilterSet[item.IpamPoolID]; !ok {
				continue
			}
		}
		if len(ipamScopeIDFilterSet) > 0 {
			if _, ok := ipamScopeIDFilterSet[item.IpamScopeID]; !ok {
				continue
			}
		}
		if tagKey != "" && !stage125ResourceTagsContain(item.ResourceTags, tagKey, tagValue) {
			continue
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		defaultType := "vpc"
		if resourceType != "" {
			defaultType = strings.ToLower(resourceType)
		}
		items = append(items, IpamResourceCidr{
			ComplianceStatus: "compliant",
			IpamID:           effectiveIpamID,
			IpamPoolID:       effectiveIpamPoolID,
			IpamScopeID:      effectiveIpamScopeID,
			ManagementState:  "managed",
			OverlapStatus:    "nonoverlapping",
			ResourceCidr:     "10.0.0.0/16",
			ResourceID:       defaultVPCID,
			ResourceName:     defaultVPCID,
			ResourceOwnerID:  resourceOwner,
			ResourceRegion:   DefaultRegion,
			ResourceType:     defaultType,
			VpcID:            defaultVPCID,
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]IpamResourceCidr(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetLaunchTemplateData(instanceID string) (LaunchTemplateDataResponse, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return LaunchTemplateDataResponse{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return LaunchTemplateDataResponse{}, ErrNotFound
	}

	securityGroupIDs := dedupeTrimmedStrings(instance.SecurityGroupIDs)
	securityGroups := make([]string, 0, len(securityGroupIDs))
	for _, securityGroupID := range securityGroupIDs {
		groupName := securityGroupID
		if group := s.securityGroups[securityGroupID]; group != nil && strings.TrimSpace(group.Name) != "" {
			groupName = group.Name
		}
		securityGroups = append(securityGroups, groupName)
	}

	return LaunchTemplateDataResponse{
		ImageID:          instance.ImageID,
		InstanceType:     instance.InstanceType,
		KeyName:          instance.KeyName,
		UserData:         instance.UserData,
		SecurityGroupIDs: securityGroupIDs,
		SecurityGroups:   securityGroups,
	}, nil
}

func (s *Service) GetManagedPrefixListAssociations(
	prefixListID string,
	maxResults *int32,
	nextToken *string,
) ([]PrefixListAssociation, *string, error) {
	prefixListID = strings.TrimSpace(prefixListID)
	if prefixListID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.managedPrefixLists[prefixListID] == nil {
		return nil, nil, ErrNotFound
	}

	items := make([]PrefixListAssociation, 0)
	keys := make([]string, 0, len(s.transitGatewayPrefixListReferences))
	for key := range s.transitGatewayPrefixListReferences {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		reference := s.transitGatewayPrefixListReferences[key]
		if reference == nil || reference.PrefixListID != prefixListID {
			continue
		}
		resourceID := ""
		if reference.TransitGatewayAttachment != nil {
			resourceID = strings.TrimSpace(reference.TransitGatewayAttachment.ResourceID)
		}
		if resourceID == "" {
			resourceID = strings.TrimSpace(reference.TransitGatewayRouteTableID)
		}
		if resourceID == "" {
			resourceID = "tgw-prefix-list-reference"
		}
		items = append(items, PrefixListAssociation{
			ResourceID:    resourceID,
			ResourceOwner: DefaultAccountID,
		})
	}

	if len(items) == 0 {
		items = append(items, PrefixListAssociation{
			ResourceID:    defaultVPCID,
			ResourceOwner: DefaultAccountID,
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]PrefixListAssociation(nil), items[start:end]...), outputToken, nil
}

func dedupeStage125PublicAddressSecurityGroups(in []IpamPublicAddressSecurityGroup) []IpamPublicAddressSecurityGroup {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]IpamPublicAddressSecurityGroup, 0, len(in))
	for _, item := range in {
		groupID := strings.TrimSpace(item.GroupID)
		groupName := strings.TrimSpace(item.GroupName)
		key := groupID + "|" + groupName
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, IpamPublicAddressSecurityGroup{
			GroupID:   groupID,
			GroupName: groupName,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupID != out[j].GroupID {
			return out[i].GroupID < out[j].GroupID
		}
		return out[i].GroupName < out[j].GroupName
	})
	return out
}

func stage125PublicAddressTagMap(tags []IpamPublicAddressTag) map[string]string {
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

func stage125ResourceTagsFromMap(in map[string]string) []IpamResourceTag {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]IpamResourceTag, 0, len(keys))
	for _, key := range keys {
		out = append(out, IpamResourceTag{Key: key, Value: in[key]})
	}
	return out
}

func stage125ResourceTagsContain(tags []IpamResourceTag, key string, value string) bool {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" {
		return true
	}
	for _, tag := range tags {
		if strings.TrimSpace(tag.Key) != key {
			continue
		}
		if value == "" {
			return true
		}
		return strings.TrimSpace(tag.Value) == value
	}
	return false
}

func stage125NetmaskFromCIDR(cidr string) *int32 {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return nil
	}
	slash := strings.LastIndex(cidr, "/")
	if slash < 0 || slash == len(cidr)-1 {
		return nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(cidr[slash+1:]), 10, 32)
	if err != nil {
		return nil
	}
	value := int32(parsed)
	return &value
}
