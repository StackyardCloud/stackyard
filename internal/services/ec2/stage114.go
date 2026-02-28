package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Byoasn struct {
	Asn           string
	IpamID        string
	State         string
	StatusMessage string
}

type IpamPoolCidr struct {
	Cidr           string
	FailureReason  *IpamPoolCidrFailureReason
	IpamPoolCidrID string
	NetmaskLength  *int32
	State          string
}

type IpamPoolCidrFailureReason struct {
	Code    string
	Message string
}

type InstanceTagNotificationAttribute struct {
	IncludeAllTagsOfInstance *bool
	InstanceTagKeys          []string
}

type CapacityBlockExtension struct {
	AvailabilityZone                    *string
	AvailabilityZoneID                  *string
	CapacityBlockExtensionDurationHours *int32
	CapacityBlockExtensionEndDate       *time.Time
	CapacityBlockExtensionOfferingID    *string
	CapacityBlockExtensionPurchaseDate  *time.Time
	CapacityBlockExtensionStartDate     *time.Time
	CapacityBlockExtensionStatus        string
	CapacityReservationID               *string
	CurrencyCode                        *string
	InstanceCount                       *int32
	InstanceType                        *string
	UpfrontFee                          *string
}

type CapacityBlockExtensionOffering struct {
	AvailabilityZone                    *string
	AvailabilityZoneID                  *string
	CapacityBlockExtensionDurationHours *int32
	CapacityBlockExtensionEndDate       *time.Time
	CapacityBlockExtensionOfferingID    *string
	CapacityBlockExtensionStartDate     *time.Time
	CurrencyCode                        *string
	InstanceCount                       *int32
	InstanceType                        *string
	StartDate                           *time.Time
	Tenancy                             *string
	UpfrontFee                          *string
}

type CapacityBlockOffering struct {
	AvailabilityZone             *string
	CapacityBlockDurationHours   *int32
	CapacityBlockDurationMinutes *int32
	CapacityBlockOfferingID      *string
	CurrencyCode                 *string
	EndDate                      *time.Time
	InstanceCount                *int32
	InstanceType                 *string
	StartDate                    *time.Time
	Tenancy                      *string
	UltraserverCount             *int32
	UltraserverType              *string
	UpfrontFee                   *string
}

type CapacityReservationStatus struct {
	CapacityReservationID    *string
	TotalAvailableCapacity   *int32
	TotalCapacity            *int32
	TotalUnavailableCapacity *int32
}

type CapacityBlockStatus struct {
	CapacityBlockID             *string
	CapacityReservationStatuses []CapacityReservationStatus
	InterconnectStatus          *string
	TotalAvailableCapacity      *int32
	TotalCapacity               *int32
	TotalUnavailableCapacity    *int32
}

func (s *Service) DeprovisionIpamByoasn(asn, ipamID string) (Byoasn, error) {
	asn = strings.TrimSpace(asn)
	ipamID = strings.TrimSpace(ipamID)
	if asn == "" || ipamID == "" {
		return Byoasn{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ipams[ipamID] == nil {
		return Byoasn{}, ErrNotFound
	}

	for _, record := range s.byoipCidrs {
		if record == nil {
			continue
		}
		for i := range record.AsnAssociations {
			if strings.EqualFold(strings.TrimSpace(record.AsnAssociations[i].Asn), asn) {
				record.AsnAssociations[i].State = "deprovisioned"
				record.AsnAssociations[i].StatusMessage = "deprovisioned"
			}
		}
	}

	return Byoasn{
		Asn:           asn,
		IpamID:        ipamID,
		State:         "deprovisioned",
		StatusMessage: "deprovisioned",
	}, nil
}

func (s *Service) DeprovisionIpamPoolCidr(ipamPoolID string, cidr *string) (IpamPoolCidr, error) {
	ipamPoolID = strings.TrimSpace(ipamPoolID)
	cidrValue := strings.TrimSpace(derefString(cidr))
	if ipamPoolID == "" {
		return IpamPoolCidr{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ipamPools[ipamPoolID] == nil {
		return IpamPoolCidr{}, ErrNotFound
	}

	var allocation *IpamPoolAllocation
	if cidrValue != "" {
		for _, item := range s.ipamPoolAllocations {
			if item == nil {
				continue
			}
			if item.ResourceID != ipamPoolID {
				continue
			}
			if item.Cidr != cidrValue {
				continue
			}
			allocation = item
			break
		}
	} else {
		allocationIDs := make([]string, 0)
		for allocationID, item := range s.ipamPoolAllocations {
			if item == nil || item.ResourceID != ipamPoolID {
				continue
			}
			allocationIDs = append(allocationIDs, allocationID)
		}
		sort.Strings(allocationIDs)
		if len(allocationIDs) > 0 {
			allocation = s.ipamPoolAllocations[allocationIDs[0]]
		}
	}

	if allocation == nil {
		return IpamPoolCidr{}, ErrNotFound
	}
	delete(s.ipamPoolAllocations, allocation.IpamPoolAllocationID)

	ipamPoolCidrID := allocation.IpamPoolAllocationID
	if ipamPoolCidrID == "" {
		ipamPoolCidrID = s.nextIDLocked("ipam-pool-cidr")
	}

	return IpamPoolCidr{
		Cidr:           allocation.Cidr,
		IpamPoolCidrID: ipamPoolCidrID,
		NetmaskLength:  parseCIDRNetmaskLength(allocation.Cidr),
		State:          "deprovisioned",
	}, nil
}

func (s *Service) DeprovisionPublicIpv4PoolCidr(poolID, cidr string) ([]string, string, error) {
	poolID = strings.TrimSpace(poolID)
	cidr = strings.TrimSpace(cidr)
	if poolID == "" || cidr == "" {
		return nil, "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.publicIpv4Pools[poolID] == nil {
		return nil, "", ErrNotFound
	}

	return []string{cidr}, poolID, nil
}

func (s *Service) DeregisterInstanceEventNotificationAttributes(includeAllTagsOfInstance *bool, instanceTagKeys []string) (InstanceTagNotificationAttribute, error) {
	instanceTagKeys = dedupeTrimmedStrings(instanceTagKeys)
	if includeAllTagsOfInstance == nil && len(instanceTagKeys) == 0 {
		return InstanceTagNotificationAttribute{}, ErrInvalidParameter
	}
	if includeAllTagsOfInstance != nil && *includeAllTagsOfInstance {
		return InstanceTagNotificationAttribute{}, ErrInvalidParameter
	}

	includeAll := false
	return InstanceTagNotificationAttribute{
		IncludeAllTagsOfInstance: &includeAll,
		InstanceTagKeys:          []string{},
	}, nil
}

func (s *Service) DescribeBundleTasks(bundleIDs []string, filters map[string][]string) []BundleTask {
	bundleIDs = dedupeTrimmedStrings(bundleIDs)
	bundleIDSet := toStringSet(bundleIDs)
	bundleIDFilterSet := toStringSet(dedupeTrimmedStrings(filters["bundle-id"]))
	instanceIDFilterSet := toStringSet(dedupeTrimmedStrings(filters["instance-id"]))
	stateFilterSet := toLowerStringSet(dedupeTrimmedStrings(filters["state"]))

	s.mu.Lock()
	defer s.mu.Unlock()

	bundleIDList := make([]string, 0, len(s.bundleTasks))
	for bundleID := range s.bundleTasks {
		bundleIDList = append(bundleIDList, bundleID)
	}
	sort.Strings(bundleIDList)

	out := make([]BundleTask, 0, len(bundleIDList))
	for _, bundleID := range bundleIDList {
		task := s.bundleTasks[bundleID]
		if task == nil {
			continue
		}
		if len(bundleIDSet) > 0 {
			if _, ok := bundleIDSet[bundleID]; !ok {
				continue
			}
		}
		if len(bundleIDFilterSet) > 0 {
			if _, ok := bundleIDFilterSet[bundleID]; !ok {
				continue
			}
		}
		if len(instanceIDFilterSet) > 0 {
			if _, ok := instanceIDFilterSet[task.InstanceID]; !ok {
				continue
			}
		}
		if len(stateFilterSet) > 0 {
			if _, ok := stateFilterSet[strings.ToLower(task.State)]; !ok {
				continue
			}
		}
		out = append(out, cloneBundleTask(task))
	}

	return out
}

func (s *Service) DescribeByoipCidrs(maxResults *int32, nextToken *string) ([]ByoipCidr, *string, error) {
	if maxResults == nil || *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cidrs := make([]string, 0, len(s.byoipCidrs))
	for cidr := range s.byoipCidrs {
		cidrs = append(cidrs, cidr)
	}
	sort.Strings(cidrs)

	out := make([]ByoipCidr, 0, len(cidrs))
	for _, cidr := range cidrs {
		out = append(out, cloneByoipCidr(s.byoipCidrs[cidr]))
	}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ByoipCidr(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityBlockExtensionHistory(capacityReservationIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CapacityBlockExtension, *string, error) {
	_ = filters

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	ids := dedupeTrimmedStrings(capacityReservationIDs)
	if len(ids) == 0 {
		ids = []string{"cr-0000000000000114"}
	}
	sort.Strings(ids)

	now := time.Now().UTC()
	out := make([]CapacityBlockExtension, 0, len(ids))
	for i, capacityReservationID := range ids {
		offeringID := fmt.Sprintf("cbext-offer-%03d", i+1)
		availabilityZone := "us-east-1a"
		availabilityZoneID := "use1-az1"
		currencyCode := "USD"
		instanceCount := int32(1)
		instanceType := "p5.48xlarge"
		status := "payment-succeeded"
		upfrontFee := "0.00"
		duration := int32(24)
		startDate := now.Add(time.Duration(i) * time.Hour)
		endDate := startDate.Add(24 * time.Hour)
		purchaseDate := startDate.Add(-15 * time.Minute)
		out = append(out, CapacityBlockExtension{
			AvailabilityZone:                    &availabilityZone,
			AvailabilityZoneID:                  &availabilityZoneID,
			CapacityBlockExtensionDurationHours: &duration,
			CapacityBlockExtensionEndDate:       &endDate,
			CapacityBlockExtensionOfferingID:    &offeringID,
			CapacityBlockExtensionPurchaseDate:  &purchaseDate,
			CapacityBlockExtensionStartDate:     &startDate,
			CapacityBlockExtensionStatus:        status,
			CapacityReservationID:               &capacityReservationID,
			CurrencyCode:                        &currencyCode,
			InstanceCount:                       &instanceCount,
			InstanceType:                        &instanceType,
			UpfrontFee:                          &upfrontFee,
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityBlockExtension(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityBlockExtensionOfferings(capacityReservationID string, capacityBlockExtensionDurationHours int32, maxResults *int32, nextToken *string) ([]CapacityBlockExtensionOffering, *string, error) {
	capacityReservationID = strings.TrimSpace(capacityReservationID)
	if capacityReservationID == "" || capacityBlockExtensionDurationHours <= 0 {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	now := time.Now().UTC()
	availabilityZone := "us-east-1a"
	availabilityZoneID := "use1-az1"
	offeringID := "cbext-offer-0000000114"
	currencyCode := "USD"
	instanceCount := int32(1)
	instanceType := "p5.48xlarge"
	startDate := now
	extensionStartDate := now.Add(2 * time.Hour)
	extensionEndDate := extensionStartDate.Add(time.Duration(capacityBlockExtensionDurationHours) * time.Hour)
	tenancy := "default"
	upfrontFee := "0.00"

	out := []CapacityBlockExtensionOffering{{
		AvailabilityZone:                    &availabilityZone,
		AvailabilityZoneID:                  &availabilityZoneID,
		CapacityBlockExtensionDurationHours: &capacityBlockExtensionDurationHours,
		CapacityBlockExtensionEndDate:       &extensionEndDate,
		CapacityBlockExtensionOfferingID:    &offeringID,
		CapacityBlockExtensionStartDate:     &extensionStartDate,
		CurrencyCode:                        &currencyCode,
		InstanceCount:                       &instanceCount,
		InstanceType:                        &instanceType,
		StartDate:                           &startDate,
		Tenancy:                             &tenancy,
		UpfrontFee:                          &upfrontFee,
	}}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityBlockExtensionOffering(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityBlockOfferings(
	capacityDurationHours int32,
	endDateRange *time.Time,
	instanceCount *int32,
	instanceType *string,
	maxResults *int32,
	nextToken *string,
	startDateRange *time.Time,
	ultraserverCount *int32,
	ultraserverType *string,
) ([]CapacityBlockOffering, *string, error) {
	if capacityDurationHours <= 0 {
		return nil, nil, ErrInvalidParameter
	}
	if instanceCount != nil && *instanceCount <= 0 {
		return nil, nil, ErrInvalidParameter
	}
	if ultraserverCount != nil && *ultraserverCount <= 0 {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	now := time.Now().UTC()
	startDate := now.Add(time.Hour)
	if startDateRange != nil {
		startDate = startDateRange.UTC()
	}
	endDate := startDate.Add(time.Duration(capacityDurationHours) * time.Hour)
	if endDateRange != nil {
		endDate = endDateRange.UTC()
	}

	availabilityZone := "us-east-1a"
	offeringID := "cbo-0000000114"
	currencyCode := "USD"
	tenancy := "default"
	instanceTypeValue := strings.TrimSpace(derefString(instanceType))
	if instanceTypeValue == "" {
		instanceTypeValue = "p5.48xlarge"
	}
	instanceCountValue := int32(1)
	if instanceCount != nil {
		instanceCountValue = *instanceCount
	}
	upfrontFee := fmt.Sprintf("%d.00", instanceCountValue)
	minutes := int32(0)

	out := []CapacityBlockOffering{{
		AvailabilityZone:             &availabilityZone,
		CapacityBlockDurationHours:   &capacityDurationHours,
		CapacityBlockDurationMinutes: &minutes,
		CapacityBlockOfferingID:      &offeringID,
		CurrencyCode:                 &currencyCode,
		EndDate:                      &endDate,
		InstanceCount:                &instanceCountValue,
		InstanceType:                 &instanceTypeValue,
		StartDate:                    &startDate,
		Tenancy:                      &tenancy,
		UltraserverCount:             cloneInt32Pointer(ultraserverCount),
		UltraserverType:              cloneStringPointer(ultraserverType),
		UpfrontFee:                   &upfrontFee,
	}}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityBlockOffering(nil), out[start:end]...), outputToken, nil
}

func (s *Service) DescribeCapacityBlockStatus(capacityBlockIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]CapacityBlockStatus, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	capacityBlockIDs = dedupeTrimmedStrings(capacityBlockIDs)
	if len(capacityBlockIDs) == 0 {
		capacityBlockIDs = []string{"cb-0000000000000114"}
	}
	sort.Strings(capacityBlockIDs)

	interconnectFilterSet := toLowerStringSet(dedupeTrimmedStrings(filters["interconnect-status"]))

	out := make([]CapacityBlockStatus, 0, len(capacityBlockIDs))
	for _, capacityBlockID := range capacityBlockIDs {
		interconnectStatus := "ok"
		if len(interconnectFilterSet) > 0 {
			if _, ok := interconnectFilterSet[interconnectStatus]; !ok {
				continue
			}
		}

		capacityReservationID := "cr-" + strings.TrimPrefix(capacityBlockID, "cb-")
		totalCapacity := int32(8)
		totalUnavailable := int32(0)
		totalAvailable := int32(totalCapacity - totalUnavailable)

		out = append(out, CapacityBlockStatus{
			CapacityBlockID: &capacityBlockID,
			CapacityReservationStatuses: []CapacityReservationStatus{
				{
					CapacityReservationID:    &capacityReservationID,
					TotalAvailableCapacity:   &totalAvailable,
					TotalCapacity:            &totalCapacity,
					TotalUnavailableCapacity: &totalUnavailable,
				},
			},
			InterconnectStatus:       &interconnectStatus,
			TotalAvailableCapacity:   &totalAvailable,
			TotalCapacity:            &totalCapacity,
			TotalUnavailableCapacity: &totalUnavailable,
		})
	}

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]CapacityBlockStatus(nil), out[start:end]...), outputToken, nil
}

func parseCIDRNetmaskLength(cidr string) *int32 {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return nil
	}
	_, suffix, found := strings.Cut(cidr, "/")
	if !found {
		return nil
	}
	netmaskValue, err := strconv.Atoi(strings.TrimSpace(suffix))
	if err != nil {
		return nil
	}
	netmask := int32(netmaskValue)
	return &netmask
}

func cloneStringPointer(in *string) *string {
	if in == nil {
		return nil
	}
	value := strings.TrimSpace(*in)
	return &value
}
