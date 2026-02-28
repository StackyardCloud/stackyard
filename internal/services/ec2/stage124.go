package ec2

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
)

type AssociatedIpv6PoolCidr struct {
	AssociatedResource string
	Ipv6Cidr           string
}

type CapacityReservationInstanceUsage struct {
	AccountID         string
	UsedInstanceCount int32
}

type CapacityReservationUsage struct {
	AvailableInstanceCount int32
	CapacityReservationID  string
	InstanceType           string
	InstanceUsages         []CapacityReservationInstanceUsage
	State                  string
	TotalInstanceCount     int32
}

type CoipAddressUsage struct {
	AllocationID string
	AwsAccountID string
	AwsService   string
	CoIP         string
}

type CoipPoolUsage struct {
	CoipAddressUsages        []CoipAddressUsage
	CoipPoolID               string
	LocalGatewayRouteTableID string
}

type InstanceFamilyCreditSpecification struct {
	CpuCredits     string
	InstanceFamily string
}

type CapacityReservationGroupAssociation struct {
	GroupARN string
	OwnerID  string
}

type HostReservationPurchase struct {
	CurrencyCode      string
	Duration          int32
	HostIDs           []string
	HostReservationID string
	HourlyPrice       string
	InstanceFamily    string
	PaymentOption     string
	UpfrontPrice      string
}

type HostReservationPurchasePreview struct {
	CurrencyCode      string
	Purchase          []HostReservationPurchase
	TotalHourlyPrice  string
	TotalUpfrontPrice string
}

type InstanceMetadataDefaults struct {
	HttpEndpoint            string
	HttpPutResponseHopLimit *int32
	HttpTokens              string
	InstanceMetadataTags    string
	ManagedBy               string
	ManagedExceptionMessage *string
}

type InstanceTpmEkPub struct {
	InstanceID string
	KeyFormat  string
	KeyType    string
	KeyValue   string
}

type InstanceTypeFromRequirements struct {
	InstanceType string
}

func (s *Service) GetAssociatedIpv6PoolCidrs(poolID string, maxResults *int32, nextToken *string) ([]AssociatedIpv6PoolCidr, *string, error) {
	poolID = strings.TrimSpace(poolID)
	if poolID == "" {
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
	items := make([]AssociatedIpv6PoolCidr, 0)
	for _, association := range s.vpcIPv6CidrAssociations {
		if association == nil {
			continue
		}
		if strings.TrimSpace(association.IPv6Pool) != poolID {
			continue
		}
		cidr := strings.TrimSpace(association.IPv6CidrBlock)
		if cidr == "" {
			continue
		}
		items = append(items, AssociatedIpv6PoolCidr{
			AssociatedResource: strings.TrimSpace(association.VpcID),
			Ipv6Cidr:           cidr,
		})
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].AssociatedResource != items[j].AssociatedResource {
			return items[i].AssociatedResource < items[j].AssociatedResource
		}
		return items[i].Ipv6Cidr < items[j].Ipv6Cidr
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]AssociatedIpv6PoolCidr(nil), items[start:end]...), outputToken, nil
}

func (s *Service) GetCapacityReservationUsage(capacityReservationID string, maxResults *int32, nextToken *string) (CapacityReservationUsage, *string, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityReservationID == "" {
		return CapacityReservationUsage{}, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return CapacityReservationUsage{}, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return CapacityReservationUsage{}, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	reservation := s.capacityReservations[capacityReservationID]
	if reservation == nil {
		s.mu.Unlock()
		return CapacityReservationUsage{}, nil, ErrNotFound
	}

	usedCount := reservation.TotalInstanceCount - reservation.AvailableInstanceCount
	if usedCount < 0 {
		usedCount = 0
	}
	ownerID := strings.TrimSpace(reservation.OwnerID)
	if ownerID == "" {
		ownerID = DefaultAccountID
	}
	usages := []CapacityReservationInstanceUsage{{
		AccountID:         ownerID,
		UsedInstanceCount: usedCount,
	}}
	usage := CapacityReservationUsage{
		AvailableInstanceCount: reservation.AvailableInstanceCount,
		CapacityReservationID:  reservation.ID,
		InstanceType:           reservation.InstanceType,
		State:                  reservation.State,
		TotalInstanceCount:     reservation.TotalInstanceCount,
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(usages), start, maxResults)
	if err != nil {
		return CapacityReservationUsage{}, nil, ErrInvalidParameter
	}
	usage.InstanceUsages = append([]CapacityReservationInstanceUsage(nil), usages[start:end]...)
	return usage, outputToken, nil
}

func (s *Service) GetCoipPoolUsage(poolID string, filters map[string][]string, maxResults *int32, nextToken *string) (CoipPoolUsage, *string, error) {
	poolID = strings.TrimSpace(poolID)
	if poolID == "" {
		return CoipPoolUsage{}, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return CoipPoolUsage{}, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return CoipPoolUsage{}, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	allocationIDFilterSet := toStringSet(standardFilters["coip-address-usage.allocation-id"])
	awsAccountIDFilterSet := toStringSet(standardFilters["coip-address-usage.aws-account-id"])
	awsServiceFilterSet := toLowerStringSet(standardFilters["coip-address-usage.aws-service"])
	coIPFilterSet := toStringSet(standardFilters["coip-address-usage.co-ip"])

	s.mu.Lock()
	pool := s.coipPools[poolID]
	if pool == nil {
		s.mu.Unlock()
		return CoipPoolUsage{}, nil, ErrNotFound
	}
	localGatewayRouteTableID := strings.TrimSpace(pool.LocalGatewayRouteTableID)

	coipCIDRs := make([]string, 0)
	for _, coipCidr := range s.coipCidrs {
		if coipCidr == nil {
			continue
		}
		if strings.TrimSpace(coipCidr.CoipPoolID) != poolID {
			continue
		}
		if localGatewayRouteTableID == "" {
			localGatewayRouteTableID = strings.TrimSpace(coipCidr.LocalGatewayRouteTableID)
		}
		cidr := strings.TrimSpace(coipCidr.Cidr)
		if cidr == "" {
			continue
		}
		coipCIDRs = append(coipCIDRs, cidr)
	}
	s.mu.Unlock()

	coipCIDRs = dedupeTrimmedStrings(coipCIDRs)
	sort.Strings(coipCIDRs)

	items := make([]CoipAddressUsage, 0, len(coipCIDRs))
	for idx, cidr := range coipCIDRs {
		coip := cidr
		if slash := strings.Index(coip, "/"); slash >= 0 {
			coip = coip[:slash]
		}
		item := CoipAddressUsage{
			AllocationID: fmt.Sprintf("eipalloc-%012d", idx+1),
			AwsAccountID: DefaultAccountID,
			AwsService:   "ec2",
			CoIP:         coip,
		}
		if len(allocationIDFilterSet) > 0 {
			if _, ok := allocationIDFilterSet[item.AllocationID]; !ok {
				continue
			}
		}
		if len(awsAccountIDFilterSet) > 0 {
			if _, ok := awsAccountIDFilterSet[item.AwsAccountID]; !ok {
				continue
			}
		}
		if len(awsServiceFilterSet) > 0 {
			if _, ok := awsServiceFilterSet[strings.ToLower(item.AwsService)]; !ok {
				continue
			}
		}
		if len(coIPFilterSet) > 0 {
			if _, ok := coIPFilterSet[item.CoIP]; !ok {
				continue
			}
		}
		items = append(items, item)
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return CoipPoolUsage{}, nil, ErrInvalidParameter
	}

	return CoipPoolUsage{
		CoipAddressUsages:        append([]CoipAddressUsage(nil), items[start:end]...),
		CoipPoolID:               poolID,
		LocalGatewayRouteTableID: localGatewayRouteTableID,
	}, outputToken, nil
}

func (s *Service) GetDefaultCreditSpecification(instanceFamily string) (InstanceFamilyCreditSpecification, error) {
	instanceFamily = strings.TrimSpace(instanceFamily)
	if instanceFamily == "" {
		return InstanceFamilyCreditSpecification{}, ErrInvalidParameter
	}
	familyLower := strings.ToLower(instanceFamily)
	s.mu.Lock()
	cpuCredits := strings.ToLower(strings.TrimSpace(s.defaultCreditSpecifications[familyLower]))
	s.mu.Unlock()
	if cpuCredits == "" {
		cpuCredits = "standard"
		if strings.HasPrefix(familyLower, "t3") || strings.HasPrefix(familyLower, "t4g") {
			cpuCredits = "unlimited"
		}
	}
	return InstanceFamilyCreditSpecification{
		CpuCredits:     cpuCredits,
		InstanceFamily: familyLower,
	}, nil
}

func (s *Service) GetFlowLogsIntegrationTemplate(configDeliveryS3DestinationARN, flowLogID string, hasIntegrateService bool) (string, error) {
	configDeliveryS3DestinationARN = strings.TrimSpace(configDeliveryS3DestinationARN)
	flowLogID = strings.TrimSpace(flowLogID)
	if configDeliveryS3DestinationARN == "" || flowLogID == "" || !hasIntegrateService {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	flowLog := s.flowLogs[flowLogID]
	s.mu.Unlock()
	if flowLog == nil {
		return "", ErrNotFound
	}

	template := fmt.Sprintf(`{"AWSTemplateFormatVersion":"2010-09-09","Description":"Stackyard flow log integration","Resources":{"FlowLogIntegration":{"Type":"Custom::FlowLogIntegration","Properties":{"FlowLogId":"%s","ResourceId":"%s","ConfigDeliveryS3DestinationArn":"%s"}}}}`, flowLogID, flowLog.ResourceID, configDeliveryS3DestinationARN)
	return template, nil
}

func (s *Service) GetGroupsForCapacityReservation(capacityReservationID string, maxResults *int32, nextToken *string) ([]CapacityReservationGroupAssociation, *string, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityReservationID == "" {
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
	reservation := s.capacityReservations[capacityReservationID]
	s.mu.Unlock()
	if reservation == nil {
		return nil, nil, ErrNotFound
	}

	groups := []CapacityReservationGroupAssociation{{
		GroupARN: fmt.Sprintf("arn:aws:resource-groups:%s:%s:group/stackyard-capacity-reservation-%s", DefaultRegion, DefaultAccountID, capacityReservationID),
		OwnerID:  DefaultAccountID,
	}}

	start, end, outputToken, err := ec2PageWindow(len(groups), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityReservationGroupAssociation(nil), groups[start:end]...), outputToken, nil
}

func (s *Service) GetHostReservationPurchasePreview(hostIDs []string, offeringID string) (HostReservationPurchasePreview, error) {
	hostIDs = dedupeTrimmedStrings(hostIDs)
	offeringID = strings.TrimSpace(offeringID)
	if len(hostIDs) == 0 || offeringID == "" {
		return HostReservationPurchasePreview{}, ErrInvalidParameter
	}

	s.mu.Lock()
	instanceFamily := "m5"
	for i, hostID := range hostIDs {
		host := s.dedicatedHosts[hostID]
		if host == nil {
			s.mu.Unlock()
			return HostReservationPurchasePreview{}, ErrNotFound
		}
		if i == 0 {
			instanceFamily = strings.TrimSpace(host.InstanceFamily)
			if instanceFamily == "" {
				instanceFamily = stage116InstanceFamilyFromType(host.InstanceType)
			}
			if instanceFamily == "" {
				instanceFamily = "m5"
			}
		}
	}
	s.mu.Unlock()

	hourly := float64(len(hostIDs)) * 0.75
	hourlyPrice := fmt.Sprintf("%.2f", hourly)
	purchase := HostReservationPurchase{
		CurrencyCode:      "USD",
		Duration:          31536000,
		HostIDs:           append([]string(nil), hostIDs...),
		HostReservationID: fmt.Sprintf("hr-preview-%03d", len(hostIDs)),
		HourlyPrice:       hourlyPrice,
		InstanceFamily:    instanceFamily,
		PaymentOption:     "NoUpfront",
		UpfrontPrice:      "0.00",
	}

	return HostReservationPurchasePreview{
		CurrencyCode:      "USD",
		Purchase:          []HostReservationPurchase{purchase},
		TotalHourlyPrice:  hourlyPrice,
		TotalUpfrontPrice: "0.00",
	}, nil
}

func (s *Service) GetInstanceMetadataDefaults() InstanceMetadataDefaults {
	s.mu.Lock()
	defaults := s.instanceMetadataDefaults
	s.mu.Unlock()

	if strings.TrimSpace(defaults.HttpEndpoint) == "" {
		defaults.HttpEndpoint = "enabled"
	}
	if defaults.HttpPutResponseHopLimit == nil {
		hopLimit := int32(2)
		defaults.HttpPutResponseHopLimit = &hopLimit
	}
	if strings.TrimSpace(defaults.HttpTokens) == "" {
		defaults.HttpTokens = "optional"
	}
	if strings.TrimSpace(defaults.InstanceMetadataTags) == "" {
		defaults.InstanceMetadataTags = "disabled"
	}
	if strings.TrimSpace(defaults.ManagedBy) == "" {
		defaults.ManagedBy = "account"
	}
	return InstanceMetadataDefaults{
		HttpEndpoint:            defaults.HttpEndpoint,
		HttpPutResponseHopLimit: cloneInt32Pointer(defaults.HttpPutResponseHopLimit),
		HttpTokens:              defaults.HttpTokens,
		InstanceMetadataTags:    defaults.InstanceMetadataTags,
		ManagedBy:               defaults.ManagedBy,
		ManagedExceptionMessage: func(in *string) *string {
			if in == nil {
				return nil
			}
			out := *in
			return &out
		}(defaults.ManagedExceptionMessage),
	}
}

func (s *Service) GetInstanceTpmEkPub(instanceID, keyFormat, keyType string) (InstanceTpmEkPub, error) {
	instanceID = strings.TrimSpace(instanceID)
	keyFormat = strings.ToLower(strings.TrimSpace(keyFormat))
	keyType = strings.ToLower(strings.TrimSpace(keyType))
	if instanceID == "" || keyFormat == "" || keyType == "" {
		return InstanceTpmEkPub{}, ErrInvalidParameter
	}
	if keyFormat != "der" && keyFormat != "tpmt" {
		return InstanceTpmEkPub{}, ErrInvalidParameter
	}
	if keyType != "rsa-2048" && keyType != "ecc-sec-p384" {
		return InstanceTpmEkPub{}, ErrInvalidParameter
	}

	s.mu.Lock()
	_, ok := s.instances[instanceID]
	s.mu.Unlock()
	if !ok {
		return InstanceTpmEkPub{}, ErrNotFound
	}

	keyValue := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("stackyard:%s:%s:%s", instanceID, keyType, keyFormat)))
	return InstanceTpmEkPub{
		InstanceID: instanceID,
		KeyFormat:  keyFormat,
		KeyType:    keyType,
		KeyValue:   keyValue,
	}, nil
}

func (s *Service) GetInstanceTypesFromInstanceRequirements(
	architectureTypes []string,
	virtualizationTypes []string,
	hasInstanceRequirements bool,
	maxResults *int32,
	nextToken *string,
) ([]InstanceTypeFromRequirements, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	if !hasInstanceRequirements {
		return nil, nil, ErrInvalidParameter
	}

	architectures := dedupeTrimmedStrings(architectureTypes)
	virtualizations := dedupeTrimmedStrings(virtualizationTypes)
	if len(architectures) == 0 || len(virtualizations) == 0 {
		return nil, nil, ErrInvalidParameter
	}

	architectureSet := toLowerStringSet(architectures)
	virtualizationSet := toLowerStringSet(virtualizations)

	type candidate struct {
		instanceType   string
		architecture   string
		virtualization string
	}
	candidates := []candidate{
		{instanceType: "t3.micro", architecture: "x86_64", virtualization: "hvm"},
		{instanceType: "m5.large", architecture: "x86_64", virtualization: "hvm"},
		{instanceType: "c6i.large", architecture: "x86_64", virtualization: "hvm"},
		{instanceType: "t4g.micro", architecture: "arm64", virtualization: "hvm"},
		{instanceType: "m7g.large", architecture: "arm64", virtualization: "hvm"},
		{instanceType: "c7g.large", architecture: "arm64", virtualization: "hvm"},
		{instanceType: "m1.small", architecture: "x86_64", virtualization: "paravirtual"},
	}

	s.mu.Lock()
	for _, instance := range s.instances {
		if instance == nil {
			continue
		}
		instanceType := strings.TrimSpace(instance.InstanceType)
		if instanceType == "" {
			continue
		}
		arch := "x86_64"
		if stage124LooksArmInstanceType(instanceType) {
			arch = "arm64"
		}
		virt := "hvm"
		if strings.HasPrefix(instanceType, "m1.") || strings.HasPrefix(instanceType, "c1.") || strings.HasPrefix(instanceType, "t1.") {
			virt = "paravirtual"
		}
		candidates = append(candidates, candidate{instanceType: instanceType, architecture: arch, virtualization: virt})
	}
	s.mu.Unlock()

	resultSet := map[string]struct{}{}
	for _, candidate := range candidates {
		if _, ok := architectureSet[strings.ToLower(candidate.architecture)]; !ok {
			continue
		}
		if _, ok := virtualizationSet[strings.ToLower(candidate.virtualization)]; !ok {
			continue
		}
		resultSet[candidate.instanceType] = struct{}{}
	}

	items := make([]InstanceTypeFromRequirements, 0, len(resultSet))
	for instanceType := range resultSet {
		items = append(items, InstanceTypeFromRequirements{InstanceType: instanceType})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].InstanceType < items[j].InstanceType
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]InstanceTypeFromRequirements(nil), items[start:end]...), outputToken, nil
}

func stage124LooksArmInstanceType(instanceType string) bool {
	instanceType = strings.ToLower(strings.TrimSpace(instanceType))
	if instanceType == "" {
		return false
	}
	armPrefixes := []string{"a1.", "t4g.", "m6g.", "m7g.", "c6g.", "c7g.", "r6g.", "r7g."}
	for _, prefix := range armPrefixes {
		if strings.HasPrefix(instanceType, prefix) {
			return true
		}
	}
	return false
}
