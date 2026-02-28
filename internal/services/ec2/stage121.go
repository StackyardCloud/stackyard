package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ReservedInstancesModification struct {
	ClientToken                     string
	CreateDate                      *time.Time
	EffectiveDate                   *time.Time
	ModificationResults             []ReservedInstancesModificationResult
	ReservedInstancesIDs            []string
	ReservedInstancesModificationID string
	Status                          string
	StatusMessage                   string
	UpdateDate                      *time.Time
}

type ReservedInstancesModificationResult struct {
	ReservedInstancesID string
}

type ReservedInstancesOffering struct {
	AvailabilityZone            string
	AvailabilityZoneID          string
	CurrencyCode                string
	Duration                    *int64
	FixedPrice                  *float32
	InstanceTenancy             string
	InstanceType                string
	Marketplace                 *bool
	OfferingClass               string
	OfferingType                string
	PricingDetails              []ReservedInstancesOfferingPricingDetail
	ProductDescription          string
	RecurringCharges            []ReservedInstancesOfferingRecurringCharge
	ReservedInstancesOfferingID string
	Scope                       string
	UsagePrice                  *float32
}

type ReservedInstancesOfferingPricingDetail struct {
	Count *int32
	Price *float64
}

type ReservedInstancesOfferingRecurringCharge struct {
	Amount    *float64
	Frequency string
}

type ScheduledInstanceRecurrence struct {
	Frequency               string
	Interval                *int32
	OccurrenceDays          []int32
	OccurrenceRelativeToEnd *bool
	OccurrenceUnit          string
}

type ScheduledInstanceAvailability struct {
	AvailabilityZone            string
	AvailableInstanceCount      *int32
	FirstSlotStartTime          *time.Time
	HourlyPrice                 string
	InstanceType                string
	MaxTermDurationInDays       *int32
	MinTermDurationInDays       *int32
	NetworkPlatform             string
	Platform                    string
	PurchaseToken               string
	Recurrence                  *ScheduledInstanceRecurrence
	SlotDurationInHours         *int32
	TotalScheduledInstanceHours *int32
}

type ScheduledInstance struct {
	AvailabilityZone            string
	CreateDate                  *time.Time
	HourlyPrice                 string
	InstanceCount               *int32
	InstanceType                string
	NetworkPlatform             string
	NextSlotStartTime           *time.Time
	Platform                    string
	PreviousSlotEndTime         *time.Time
	Recurrence                  *ScheduledInstanceRecurrence
	ScheduledInstanceID         string
	SlotDurationInHours         *int32
	TermEndDate                 *time.Time
	TermStartDate               *time.Time
	TotalScheduledInstanceHours *int32
}

type ServiceLinkVirtualInterface struct {
	ConfigurationState             string
	LocalAddress                   string
	OutpostARN                     string
	OutpostID                      string
	OutpostLagID                   string
	OwnerID                        string
	PeerAddress                    string
	PeerBgpASN                     *int64
	ServiceLinkVirtualInterfaceARN string
	ServiceLinkVirtualInterfaceID  string
	Tags                           map[string]string
	VLAN                           *int32
}

type SnapshotCreateVolumePermission struct {
	Group  string
	UserID string
}

type SnapshotProductCode struct {
	ProductCode string
	Type        string
}

type SnapshotAttribute struct {
	CreateVolumePermissions []SnapshotCreateVolumePermission
	ProductCodes            []SnapshotProductCode
	SnapshotID              string
}

type SnapshotTierStatus struct {
	ArchivalCompleteTime             *time.Time
	LastTieringOperationStatus       string
	LastTieringOperationStatusDetail string
	LastTieringProgress              *int32
	LastTieringStartTime             *time.Time
	OwnerID                          string
	RestoreExpiryTime                *time.Time
	SnapshotID                       string
	Status                           string
	StorageTier                      string
	Tags                             map[string]string
	VolumeID                         string
}

func (s *Service) DescribeReservedInstancesListings(reservedInstancesID *string, reservedInstancesListingID *string, filters map[string][]string) ([]ReservedInstancesListing, error) {
	requestedReservedInstancesID := strings.TrimSpace(derefString(reservedInstancesID))
	requestedListingID := strings.TrimSpace(derefString(reservedInstancesListingID))

	standardFilters, _, _ := splitEC2Filters(filters)
	reservedInstancesIDFilterSet := toStringSet(standardFilters["reserved-instances-id"])
	reservedInstancesListingIDFilterSet := toStringSet(standardFilters["reserved-instances-listing-id"])
	statusFilterSet := toLowerStringSet(standardFilters["status"])

	s.mu.Lock()
	listingIDs := make([]string, 0, len(s.reservedInstancesListingStates)+len(s.reservedInstancesListingCreatedAt))
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
	items := make([]ReservedInstancesListing, 0, len(listingIDs))
	for _, listingID := range listingIDs {
		reservedID := stage120ReservedInstancesIDFromListingID(listingID)
		if requestedReservedInstancesID != "" && reservedID != requestedReservedInstancesID {
			continue
		}
		if requestedListingID != "" && listingID != requestedListingID {
			continue
		}
		if len(reservedInstancesIDFilterSet) > 0 {
			if _, ok := reservedInstancesIDFilterSet[reservedID]; !ok {
				continue
			}
		}
		if len(reservedInstancesListingIDFilterSet) > 0 {
			if _, ok := reservedInstancesListingIDFilterSet[listingID]; !ok {
				continue
			}
		}

		state := strings.TrimSpace(s.reservedInstancesListingStates[listingID])
		if state == "" {
			state = "active"
		}
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(state)]; !ok {
				continue
			}
		}

		createDate := s.reservedInstancesListingCreatedAt[listingID]
		if createDate.IsZero() {
			createDate = now
		}
		item := ReservedInstancesListing{
			ClientToken:                "",
			CreateDate:                 createDate,
			ReservedInstancesID:        reservedID,
			ReservedInstancesListingID: listingID,
			Status:                     state,
			StatusMessage:              stage121ReservedInstancesListingStatusMessage(state),
			UpdateDate:                 now,
		}
		items = append(items, item)
	}
	s.mu.Unlock()

	return items, nil
}

func (s *Service) DescribeReservedInstancesModifications(modificationIDs []string, filters map[string][]string, nextToken *string) ([]ReservedInstancesModification, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(modificationIDs))
	standardFilters, _, _ := splitEC2Filters(filters)
	modificationIDFilterSet := toStringSet(standardFilters["reserved-instances-modification-id"])
	statusFilterSet := toLowerStringSet(standardFilters["status"])

	s.mu.Lock()
	listingIDs := make([]string, 0, len(s.reservedInstancesListingStates)+len(s.reservedInstancesListingCreatedAt))
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
	items := make([]ReservedInstancesModification, 0, len(listingIDs))
	for _, listingID := range listingIDs {
		modificationID := stage121ReservedInstancesModificationIDFromListingID(listingID)
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[modificationID]; !ok {
				continue
			}
		}
		if len(modificationIDFilterSet) > 0 {
			if _, ok := modificationIDFilterSet[modificationID]; !ok {
				continue
			}
		}

		listingState := strings.TrimSpace(s.reservedInstancesListingStates[listingID])
		status := stage121ReservedInstancesModificationStatusFromListingState(listingState)
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(status)]; !ok {
				continue
			}
		}

		createDate := s.reservedInstancesListingCreatedAt[listingID]
		if createDate.IsZero() {
			createDate = now
		}
		effectiveDate := createDate.Add(5 * time.Minute)
		updateDate := now
		reservedID := stage120ReservedInstancesIDFromListingID(listingID)

		item := ReservedInstancesModification{
			ClientToken:                     "",
			CreateDate:                      cloneTimePointer(&createDate),
			EffectiveDate:                   cloneTimePointer(&effectiveDate),
			ModificationResults:             []ReservedInstancesModificationResult{{ReservedInstancesID: reservedID}},
			ReservedInstancesIDs:            []string{reservedID},
			ReservedInstancesModificationID: modificationID,
			Status:                          status,
			StatusMessage:                   stage121ReservedInstancesModificationStatusMessage(status),
			UpdateDate:                      cloneTimePointer(&updateDate),
		}
		items = append(items, cloneStage121ReservedInstancesModification(item))
	}
	s.mu.Unlock()

	if start > len(items) {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ReservedInstancesModification(nil), items[start:]...), nil, nil
}

func (s *Service) DescribeReservedInstancesOfferings(
	reservedInstancesOfferingIDs []string,
	filters map[string][]string,
	includeMarketplace *bool,
	instanceTenancy string,
	instanceType string,
	maxDuration *int64,
	maxInstanceCount *int32,
	minDuration *int64,
	offeringClass string,
	offeringType string,
	productDescription string,
	maxResults *int32,
	nextToken *string,
) ([]ReservedInstancesOffering, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(reservedInstancesOfferingIDs))
	instanceTenancy = strings.ToLower(strings.TrimSpace(instanceTenancy))
	instanceType = strings.ToLower(strings.TrimSpace(instanceType))
	offeringClass = strings.ToLower(strings.TrimSpace(offeringClass))
	offeringType = strings.ToLower(strings.TrimSpace(offeringType))
	productDescription = strings.ToLower(strings.TrimSpace(productDescription))

	standardFilters, _, _ := splitEC2Filters(filters)
	availabilityZoneFilterSet := toStringSet(standardFilters["availability-zone"])
	availabilityZoneIDFilterSet := toStringSet(standardFilters["availability-zone-id"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	offeringClassFilterSet := toLowerStringSet(standardFilters["offering-class"])
	offeringTypeFilterSet := toLowerStringSet(standardFilters["offering-type"])
	productDescriptionFilterSet := toLowerStringSet(standardFilters["product-description"])
	offeringIDFilterSet := toStringSet(standardFilters["reserved-instances-offering-id"])
	scopeFilterSet := toLowerStringSet(standardFilters["scope"])
	marketplaceFilterSet := toLowerStringSet(standardFilters["marketplace"])

	items := stage121DefaultReservedInstancesOfferings()
	filtered := make([]ReservedInstancesOffering, 0, len(items))
	for _, offering := range items {
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[offering.ReservedInstancesOfferingID]; !ok {
				continue
			}
		}
		if len(offeringIDFilterSet) > 0 {
			if _, ok := offeringIDFilterSet[offering.ReservedInstancesOfferingID]; !ok {
				continue
			}
		}
		if includeMarketplace != nil {
			if offering.Marketplace == nil || *offering.Marketplace != *includeMarketplace {
				continue
			}
		}
		if instanceTenancy != "" && strings.ToLower(offering.InstanceTenancy) != instanceTenancy {
			continue
		}
		if instanceType != "" && strings.ToLower(offering.InstanceType) != instanceType {
			continue
		}
		if offeringClass != "" && strings.ToLower(offering.OfferingClass) != offeringClass {
			continue
		}
		if offeringType != "" && strings.ToLower(offering.OfferingType) != offeringType {
			continue
		}
		if productDescription != "" && strings.ToLower(offering.ProductDescription) != productDescription {
			continue
		}
		if maxDuration != nil && offering.Duration != nil && *offering.Duration > *maxDuration {
			continue
		}
		if minDuration != nil && offering.Duration != nil && *offering.Duration < *minDuration {
			continue
		}
		if maxInstanceCount != nil {
			count := int32(0)
			if len(offering.PricingDetails) > 0 && offering.PricingDetails[0].Count != nil {
				count = *offering.PricingDetails[0].Count
			}
			if count > *maxInstanceCount {
				continue
			}
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[offering.AvailabilityZone]; !ok {
				continue
			}
		}
		if len(availabilityZoneIDFilterSet) > 0 {
			if _, ok := availabilityZoneIDFilterSet[offering.AvailabilityZoneID]; !ok {
				continue
			}
		}
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(offering.InstanceType)]; !ok {
				continue
			}
		}
		if len(offeringClassFilterSet) > 0 {
			if _, ok := offeringClassFilterSet[strings.ToLower(offering.OfferingClass)]; !ok {
				continue
			}
		}
		if len(offeringTypeFilterSet) > 0 {
			if _, ok := offeringTypeFilterSet[strings.ToLower(offering.OfferingType)]; !ok {
				continue
			}
		}
		if len(productDescriptionFilterSet) > 0 {
			if _, ok := productDescriptionFilterSet[strings.ToLower(offering.ProductDescription)]; !ok {
				continue
			}
		}
		if len(scopeFilterSet) > 0 {
			if _, ok := scopeFilterSet[strings.ToLower(offering.Scope)]; !ok {
				continue
			}
		}
		if len(marketplaceFilterSet) > 0 {
			marketplaceValue := "false"
			if offering.Marketplace != nil && *offering.Marketplace {
				marketplaceValue = "true"
			}
			if _, ok := marketplaceFilterSet[marketplaceValue]; !ok {
				continue
			}
		}
		filtered = append(filtered, cloneStage121ReservedInstancesOffering(offering))
	}

	start, end, outputToken, err := ec2PageWindow(len(filtered), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ReservedInstancesOffering(nil), filtered[start:end]...), outputToken, nil
}

func (s *Service) DescribeScheduledInstanceAvailability(
	filters map[string][]string,
	firstSlotEarliestTime *time.Time,
	firstSlotLatestTime *time.Time,
	maxSlotDurationInHours *int32,
	minSlotDurationInHours *int32,
	recurrence *ScheduledInstanceRecurrence,
	maxResults *int32,
	nextToken *string,
) ([]ScheduledInstanceAvailability, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	now := time.Now().UTC().Truncate(time.Hour)
	firstSlot := now.Add(24 * time.Hour)
	maxTermDays := int32(30)
	minTermDays := int32(1)
	slotDuration := int32(1)
	totalHours := int32(24)
	availableCount := int32(5)
	defaultRecurrence := &ScheduledInstanceRecurrence{
		Frequency:      "Weekly",
		Interval:       cloneInt32Pointer(&slotDuration),
		OccurrenceDays: []int32{1},
		OccurrenceUnit: "day-of-week",
	}
	if recurrence != nil {
		defaultRecurrence = cloneStage121ScheduledInstanceRecurrence(recurrence)
	}
	if minSlotDurationInHours != nil {
		slotDuration = *minSlotDurationInHours
	}
	if maxSlotDurationInHours != nil && *maxSlotDurationInHours < slotDuration {
		slotDuration = *maxSlotDurationInHours
	}

	items := []ScheduledInstanceAvailability{
		{
			AvailabilityZone:            "us-east-1a",
			AvailableInstanceCount:      cloneInt32Pointer(&availableCount),
			FirstSlotStartTime:          cloneTimePointer(&firstSlot),
			HourlyPrice:                 "0.050",
			InstanceType:                "c5.large",
			MaxTermDurationInDays:       cloneInt32Pointer(&maxTermDays),
			MinTermDurationInDays:       cloneInt32Pointer(&minTermDays),
			NetworkPlatform:             "EC2-VPC",
			Platform:                    "Linux/UNIX",
			PurchaseToken:               "purchase-token-1",
			Recurrence:                  cloneStage121ScheduledInstanceRecurrence(defaultRecurrence),
			SlotDurationInHours:         cloneInt32Pointer(&slotDuration),
			TotalScheduledInstanceHours: cloneInt32Pointer(&totalHours),
		},
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	availabilityZoneFilterSet := toStringSet(standardFilters["availability-zone"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	networkPlatformFilterSet := toLowerStringSet(standardFilters["network-platform"])
	platformFilterSet := toLowerStringSet(standardFilters["platform"])

	filtered := make([]ScheduledInstanceAvailability, 0, len(items))
	for _, item := range items {
		if firstSlotEarliestTime != nil && item.FirstSlotStartTime != nil && item.FirstSlotStartTime.Before(firstSlotEarliestTime.UTC()) {
			continue
		}
		if firstSlotLatestTime != nil && item.FirstSlotStartTime != nil && item.FirstSlotStartTime.After(firstSlotLatestTime.UTC()) {
			continue
		}
		if maxSlotDurationInHours != nil && item.SlotDurationInHours != nil && *item.SlotDurationInHours > *maxSlotDurationInHours {
			continue
		}
		if minSlotDurationInHours != nil && item.SlotDurationInHours != nil && *item.SlotDurationInHours < *minSlotDurationInHours {
			continue
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[item.AvailabilityZone]; !ok {
				continue
			}
		}
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}
		if len(networkPlatformFilterSet) > 0 {
			if _, ok := networkPlatformFilterSet[strings.ToLower(item.NetworkPlatform)]; !ok {
				continue
			}
		}
		if len(platformFilterSet) > 0 {
			if _, ok := platformFilterSet[strings.ToLower(item.Platform)]; !ok {
				continue
			}
		}
		filtered = append(filtered, cloneStage121ScheduledInstanceAvailability(item))
	}

	start, end, outputToken, err := ec2PageWindow(len(filtered), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ScheduledInstanceAvailability(nil), filtered[start:end]...), outputToken, nil
}

func (s *Service) DescribeScheduledInstances(
	scheduledInstanceIDs []string,
	filters map[string][]string,
	slotStartEarliestTime *time.Time,
	slotStartLatestTime *time.Time,
	maxResults *int32,
	nextToken *string,
) ([]ScheduledInstance, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(scheduledInstanceIDs))
	standardFilters, _, _ := splitEC2Filters(filters)
	scheduledIDFilterSet := toStringSet(standardFilters["scheduled-instance-id"])
	availabilityZoneFilterSet := toStringSet(standardFilters["availability-zone"])
	instanceTypeFilterSet := toLowerStringSet(standardFilters["instance-type"])
	platformFilterSet := toLowerStringSet(standardFilters["platform"])

	now := time.Now().UTC().Truncate(time.Hour)
	instanceCount := int32(1)
	slotDuration := int32(1)
	totalHours := int32(24)
	recurrence := &ScheduledInstanceRecurrence{
		Frequency:      "Weekly",
		OccurrenceDays: []int32{1},
		OccurrenceUnit: "day-of-week",
	}

	items := []ScheduledInstance{}
	if len(requestedIDSet) == 0 {
		requestedIDSet["sci-00000000121"] = struct{}{}
	}
	for scheduledID := range requestedIDSet {
		nextSlotStart := now.Add(1 * time.Hour)
		previousSlotEnd := now.Add(-1 * time.Hour)
		termStart := now
		termEnd := now.Add(30 * 24 * time.Hour)
		items = append(items, ScheduledInstance{
			AvailabilityZone:            "us-east-1a",
			CreateDate:                  cloneTimePointer(&now),
			HourlyPrice:                 "0.050",
			InstanceCount:               cloneInt32Pointer(&instanceCount),
			InstanceType:                "c5.large",
			NetworkPlatform:             "EC2-VPC",
			NextSlotStartTime:           cloneTimePointer(&nextSlotStart),
			Platform:                    "Linux/UNIX",
			PreviousSlotEndTime:         cloneTimePointer(&previousSlotEnd),
			Recurrence:                  cloneStage121ScheduledInstanceRecurrence(recurrence),
			ScheduledInstanceID:         scheduledID,
			SlotDurationInHours:         cloneInt32Pointer(&slotDuration),
			TermEndDate:                 cloneTimePointer(&termEnd),
			TermStartDate:               cloneTimePointer(&termStart),
			TotalScheduledInstanceHours: cloneInt32Pointer(&totalHours),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ScheduledInstanceID < items[j].ScheduledInstanceID
	})

	filtered := make([]ScheduledInstance, 0, len(items))
	for _, item := range items {
		if len(scheduledIDFilterSet) > 0 {
			if _, ok := scheduledIDFilterSet[item.ScheduledInstanceID]; !ok {
				continue
			}
		}
		if slotStartEarliestTime != nil && item.NextSlotStartTime != nil && item.NextSlotStartTime.Before(slotStartEarliestTime.UTC()) {
			continue
		}
		if slotStartLatestTime != nil && item.NextSlotStartTime != nil && item.NextSlotStartTime.After(slotStartLatestTime.UTC()) {
			continue
		}
		if len(availabilityZoneFilterSet) > 0 {
			if _, ok := availabilityZoneFilterSet[item.AvailabilityZone]; !ok {
				continue
			}
		}
		if len(instanceTypeFilterSet) > 0 {
			if _, ok := instanceTypeFilterSet[strings.ToLower(item.InstanceType)]; !ok {
				continue
			}
		}
		if len(platformFilterSet) > 0 {
			if _, ok := platformFilterSet[strings.ToLower(item.Platform)]; !ok {
				continue
			}
		}
		filtered = append(filtered, cloneStage121ScheduledInstance(item))
	}

	start, end, outputToken, err := ec2PageWindow(len(filtered), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ScheduledInstance(nil), filtered[start:end]...), outputToken, nil
}

func (s *Service) DescribeServiceLinkVirtualInterfaces(serviceLinkVirtualInterfaceIDs []string, filters map[string][]string, maxResults *int32, nextToken *string) ([]ServiceLinkVirtualInterface, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	requestedIDSet := toStringSet(dedupeTrimmedStrings(serviceLinkVirtualInterfaceIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	serviceLinkIDFilterSet := toStringSet(standardFilters["service-link-virtual-interface-id"])
	outpostARNFilterSet := toStringSet(standardFilters["outpost-arn"])
	outpostIDFilterSet := toStringSet(standardFilters["outpost-id"])
	outpostLagIDFilterSet := toStringSet(standardFilters["outpost-lag-id"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	vlanFilterSet := toStringSet(standardFilters["vlan"])

	s.mu.Lock()
	items := make([]ServiceLinkVirtualInterface, 0, len(s.localGatewayVirtualInterfaces))
	for _, localInterface := range s.localGatewayVirtualInterfaces {
		if localInterface == nil {
			continue
		}
		serviceLinkID := stage121ServiceLinkVirtualInterfaceIDFromLocalGatewayVirtualInterfaceID(localInterface.LocalGatewayVirtualInterfaceID)
		outpostARN := stage120OutpostARNFromLagID(localInterface.OutpostLagID)
		outpostID := stage121OutpostIDFromLagID(localInterface.OutpostLagID)
		item := ServiceLinkVirtualInterface{
			ConfigurationState:             stage121DefaultServiceLinkConfigurationState(localInterface.ConfigurationState),
			LocalAddress:                   localInterface.LocalAddress,
			OutpostARN:                     outpostARN,
			OutpostID:                      outpostID,
			OutpostLagID:                   localInterface.OutpostLagID,
			OwnerID:                        localInterface.OwnerID,
			PeerAddress:                    localInterface.PeerAddress,
			PeerBgpASN:                     stage121Int64PointerFromInt32Pointer(localInterface.PeerBgpASN),
			ServiceLinkVirtualInterfaceARN: stage121ServiceLinkVirtualInterfaceARNFromID(serviceLinkID),
			ServiceLinkVirtualInterfaceID:  serviceLinkID,
			Tags:                           cloneStringMap(localInterface.Tags),
			VLAN:                           cloneInt32Pointer(localInterface.VLAN),
		}
		if len(requestedIDSet) > 0 {
			if _, ok := requestedIDSet[item.ServiceLinkVirtualInterfaceID]; !ok {
				continue
			}
		}
		if len(serviceLinkIDFilterSet) > 0 {
			if _, ok := serviceLinkIDFilterSet[item.ServiceLinkVirtualInterfaceID]; !ok {
				continue
			}
		}
		if len(outpostARNFilterSet) > 0 {
			if _, ok := outpostARNFilterSet[item.OutpostARN]; !ok {
				continue
			}
		}
		if len(outpostIDFilterSet) > 0 {
			if _, ok := outpostIDFilterSet[item.OutpostID]; !ok {
				continue
			}
		}
		if len(outpostLagIDFilterSet) > 0 {
			if _, ok := outpostLagIDFilterSet[item.OutpostLagID]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
				continue
			}
		}
		if len(vlanFilterSet) > 0 {
			if item.VLAN == nil {
				continue
			}
			if _, ok := vlanFilterSet[strconv.FormatInt(int64(*item.VLAN), 10)]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneStage121ServiceLinkVirtualInterface(item))
	}
	s.mu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return items[i].ServiceLinkVirtualInterfaceID < items[j].ServiceLinkVirtualInterfaceID
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]ServiceLinkVirtualInterface(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeSnapshotAttribute(snapshotID string, attribute string) (SnapshotAttribute, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	attribute = strings.TrimSpace(attribute)
	if snapshotID == "" || attribute == "" {
		return SnapshotAttribute{}, ErrInvalidParameter
	}

	s.mu.Lock()
	snapshot := s.snapshots[snapshotID]
	s.mu.Unlock()
	if snapshot == nil {
		return SnapshotAttribute{}, ErrNotFound
	}

	attributeLower := strings.ToLower(attribute)
	if attributeLower != "createvolumepermission" && attributeLower != "productcodes" {
		return SnapshotAttribute{}, ErrInvalidParameter
	}

	result := SnapshotAttribute{SnapshotID: snapshotID}
	if attributeLower == "createvolumepermission" {
		result.CreateVolumePermissions = []SnapshotCreateVolumePermission{{UserID: DefaultAccountID}}
	}
	if attributeLower == "productcodes" {
		result.ProductCodes = []SnapshotProductCode{{ProductCode: "aw0evgkw8e5c1q413zgy5pjce", Type: "marketplace"}}
	}
	return result, nil
}

func (s *Service) DescribeSnapshotTierStatus(filters map[string][]string, maxResults *int32, nextToken *string) ([]SnapshotTierStatus, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	snapshotIDFilterSet := toStringSet(standardFilters["snapshot-id"])
	storageTierFilterSet := toLowerStringSet(standardFilters["storage-tier"])
	statusFilterSet := toLowerStringSet(standardFilters["status"])
	ownerIDFilterSet := toStringSet(standardFilters["owner-id"])
	volumeIDFilterSet := toStringSet(standardFilters["volume-id"])

	s.mu.Lock()
	snapshotIDs := make([]string, 0, len(s.snapshots))
	for snapshotID := range s.snapshots {
		snapshotIDs = append(snapshotIDs, snapshotID)
	}
	sort.Strings(snapshotIDs)

	items := make([]SnapshotTierStatus, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snapshot := s.snapshots[snapshotID]
		if snapshot == nil {
			continue
		}
		storageTier := strings.TrimSpace(snapshot.Tags["storage-tier"])
		if storageTier == "" {
			storageTier = "standard"
		}
		progress := int32(100)
		startTime := snapshot.StartTime.UTC()
		restoreExpiry := startTime.Add(7 * 24 * time.Hour)
		item := SnapshotTierStatus{
			ArchivalCompleteTime:             cloneTimePointer(&startTime),
			LastTieringOperationStatus:       "completed",
			LastTieringOperationStatusDetail: "tiering completed",
			LastTieringProgress:              cloneInt32Pointer(&progress),
			LastTieringStartTime:             cloneTimePointer(&startTime),
			OwnerID:                          DefaultAccountID,
			RestoreExpiryTime:                cloneTimePointer(&restoreExpiry),
			SnapshotID:                       snapshot.ID,
			Status:                           snapshot.State,
			StorageTier:                      storageTier,
			Tags:                             cloneStringMap(snapshot.Tags),
			VolumeID:                         snapshot.VolumeID,
		}

		if len(snapshotIDFilterSet) > 0 {
			if _, ok := snapshotIDFilterSet[item.SnapshotID]; !ok {
				continue
			}
		}
		if len(storageTierFilterSet) > 0 {
			if _, ok := storageTierFilterSet[strings.ToLower(item.StorageTier)]; !ok {
				continue
			}
		}
		if len(statusFilterSet) > 0 {
			if _, ok := statusFilterSet[strings.ToLower(item.Status)]; !ok {
				continue
			}
		}
		if len(ownerIDFilterSet) > 0 {
			if _, ok := ownerIDFilterSet[item.OwnerID]; !ok {
				continue
			}
		}
		if len(volumeIDFilterSet) > 0 {
			if _, ok := volumeIDFilterSet[item.VolumeID]; !ok {
				continue
			}
		}
		if !matchesTagFilters(item.Tags, tagKeyFilters, tagFilters) {
			continue
		}

		items = append(items, cloneStage121SnapshotTierStatus(item))
	}
	s.mu.Unlock()

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]SnapshotTierStatus(nil), items[start:end]...), outputToken, nil
}

func (s *Service) DescribeSpotDatafeedSubscription() *SpotDatafeedSubscription {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.spotDatafeedSubscriptions) == 0 {
		return nil
	}
	buckets := make([]string, 0, len(s.spotDatafeedSubscriptions))
	for bucket := range s.spotDatafeedSubscriptions {
		buckets = append(buckets, bucket)
	}
	sort.Strings(buckets)
	subscription := s.spotDatafeedSubscriptions[buckets[0]]
	if subscription == nil {
		return nil
	}
	cloned := cloneStage110SpotDatafeedSubscription(subscription)
	return &cloned
}

func (s *Service) DescribeSpotFleetInstances(spotFleetRequestID string, maxResults *int32, nextToken *string) (string, []FleetActiveInstance, *string, error) {
	spotFleetRequestID = strings.TrimSpace(spotFleetRequestID)
	if spotFleetRequestID == "" {
		return "", nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return "", nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return "", nil, nil, ErrInvalidParameter
	}

	s.mu.Lock()
	state := strings.TrimSpace(s.spotFleetRequestStates[spotFleetRequestID])
	if state == "" {
		state = "active"
		s.spotFleetRequestStates[spotFleetRequestID] = state
	}
	s.mu.Unlock()

	normalized := stage121NormalizeSpotFleetRequestID(spotFleetRequestID)
	items := []FleetActiveInstance{
		{
			InstanceHealth:        "healthy",
			InstanceID:            fmt.Sprintf("i-%s1", normalized),
			InstanceType:          "t3.micro",
			SpotInstanceRequestID: fmt.Sprintf("sir-%s1", normalized),
		},
		{
			InstanceHealth:        "healthy",
			InstanceID:            fmt.Sprintf("i-%s2", normalized),
			InstanceType:          "t3.micro",
			SpotInstanceRequestID: fmt.Sprintf("sir-%s2", normalized),
		},
	}
	if strings.Contains(strings.ToLower(state), "terminated") {
		items = items[:1]
	}

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return "", nil, nil, ErrInvalidParameter
	}
	return spotFleetRequestID, append([]FleetActiveInstance(nil), items[start:end]...), outputToken, nil
}

func stage121ReservedInstancesListingStatusMessage(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued":
		return "listing queued"
	case "cancelled":
		return "listing cancelled"
	case "active":
		return "listing created"
	default:
		return "listing state recorded"
	}
}

func stage121ReservedInstancesModificationIDFromListingID(listingID string) string {
	suffix := strings.TrimSpace(strings.TrimPrefix(listingID, "ril-"))
	if suffix == "" {
		suffix = "00000000"
	}
	return "rimod-" + suffix
}

func stage121ReservedInstancesModificationStatusFromListingState(listingState string) string {
	switch strings.ToLower(strings.TrimSpace(listingState)) {
	case "queued":
		return "processing"
	case "cancelled":
		return "retired"
	default:
		return "fulfilled"
	}
}

func stage121ReservedInstancesModificationStatusMessage(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "processing":
		return "modification is being processed"
	case "retired":
		return "modification is retired"
	default:
		return "modification fulfilled"
	}
}

func stage121DefaultReservedInstancesOfferings() []ReservedInstancesOffering {
	duration := int64(31536000)
	fixedPrice := float32(0)
	usagePrice := float32(0)
	pricingCount := int32(1)
	pricingPrice := float64(0)
	recurringAmount := float64(0)
	marketplaceFalse := false

	items := []ReservedInstancesOffering{
		{
			AvailabilityZone:            "us-east-1a",
			AvailabilityZoneID:          "use1-az1",
			CurrencyCode:                "USD",
			Duration:                    cloneInt64Pointer(&duration),
			FixedPrice:                  cloneFloat32Pointer(&fixedPrice),
			InstanceTenancy:             "default",
			InstanceType:                "t3.micro",
			Marketplace:                 cloneBoolPointer(&marketplaceFalse),
			OfferingClass:               "standard",
			OfferingType:                "No Upfront",
			PricingDetails:              []ReservedInstancesOfferingPricingDetail{{Count: cloneInt32Pointer(&pricingCount), Price: cloneFloat64Pointer(&pricingPrice)}},
			ProductDescription:          "Linux/UNIX",
			RecurringCharges:            []ReservedInstancesOfferingRecurringCharge{{Amount: cloneFloat64Pointer(&recurringAmount), Frequency: "Hourly"}},
			ReservedInstancesOfferingID: "off-00000000121",
			Scope:                       "Region",
			UsagePrice:                  cloneFloat32Pointer(&usagePrice),
		},
	}
	return items
}

func stage121OutpostIDFromLagID(lagID string) string {
	lagID = strings.TrimSpace(lagID)
	if lagID == "" {
		return "op-00000000"
	}
	suffix := strings.TrimPrefix(lagID, "lag-")
	if suffix == "" {
		suffix = lagID
	}
	return "op-" + suffix
}

func stage121ServiceLinkVirtualInterfaceIDFromLocalGatewayVirtualInterfaceID(localGatewayVirtualInterfaceID string) string {
	suffix := strings.TrimSpace(strings.TrimPrefix(localGatewayVirtualInterfaceID, "lgw-vif-"))
	if suffix == "" {
		suffix = strings.TrimSpace(localGatewayVirtualInterfaceID)
	}
	if suffix == "" {
		suffix = "00000000"
	}
	return "slvi-" + suffix
}

func stage121ServiceLinkVirtualInterfaceARNFromID(serviceLinkVirtualInterfaceID string) string {
	return fmt.Sprintf("arn:aws:ec2:%s:%s:service-link-virtual-interface/%s", DefaultRegion, DefaultAccountID, serviceLinkVirtualInterfaceID)
}

func stage121DefaultServiceLinkConfigurationState(configurationState string) string {
	configurationState = strings.TrimSpace(configurationState)
	if configurationState == "" {
		return "available"
	}
	return configurationState
}

func stage121Int64PointerFromInt32Pointer(in *int32) *int64 {
	if in == nil {
		return nil
	}
	out := int64(*in)
	return &out
}

func stage121NormalizeSpotFleetRequestID(spotFleetRequestID string) string {
	normalized := strings.TrimSpace(strings.TrimPrefix(spotFleetRequestID, "sfr-"))
	normalized = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A')
		}
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, normalized)
	if normalized == "" {
		return "00000000"
	}
	if len(normalized) > 8 {
		return normalized[:8]
	}
	for len(normalized) < 8 {
		normalized += "0"
	}
	return normalized
}

func cloneStage121ReservedInstancesModification(in ReservedInstancesModification) ReservedInstancesModification {
	out := in
	out.CreateDate = cloneTimePointer(in.CreateDate)
	out.EffectiveDate = cloneTimePointer(in.EffectiveDate)
	out.UpdateDate = cloneTimePointer(in.UpdateDate)
	out.ReservedInstancesIDs = append([]string(nil), in.ReservedInstancesIDs...)
	out.ModificationResults = append([]ReservedInstancesModificationResult(nil), in.ModificationResults...)
	return out
}

func cloneStage121ReservedInstancesOffering(in ReservedInstancesOffering) ReservedInstancesOffering {
	out := in
	out.Duration = cloneInt64Pointer(in.Duration)
	out.FixedPrice = cloneFloat32Pointer(in.FixedPrice)
	out.UsagePrice = cloneFloat32Pointer(in.UsagePrice)
	out.Marketplace = cloneBoolPointer(in.Marketplace)
	out.PricingDetails = append([]ReservedInstancesOfferingPricingDetail(nil), in.PricingDetails...)
	for i := range out.PricingDetails {
		out.PricingDetails[i].Count = cloneInt32Pointer(out.PricingDetails[i].Count)
		out.PricingDetails[i].Price = cloneFloat64Pointer(out.PricingDetails[i].Price)
	}
	out.RecurringCharges = append([]ReservedInstancesOfferingRecurringCharge(nil), in.RecurringCharges...)
	for i := range out.RecurringCharges {
		out.RecurringCharges[i].Amount = cloneFloat64Pointer(out.RecurringCharges[i].Amount)
	}
	return out
}

func cloneStage121ScheduledInstanceRecurrence(in *ScheduledInstanceRecurrence) *ScheduledInstanceRecurrence {
	if in == nil {
		return nil
	}
	out := *in
	out.Interval = cloneInt32Pointer(in.Interval)
	out.OccurrenceRelativeToEnd = cloneBoolPointer(in.OccurrenceRelativeToEnd)
	out.OccurrenceDays = append([]int32(nil), in.OccurrenceDays...)
	return &out
}

func cloneStage121ScheduledInstanceAvailability(in ScheduledInstanceAvailability) ScheduledInstanceAvailability {
	out := in
	out.AvailableInstanceCount = cloneInt32Pointer(in.AvailableInstanceCount)
	out.FirstSlotStartTime = cloneTimePointer(in.FirstSlotStartTime)
	out.MaxTermDurationInDays = cloneInt32Pointer(in.MaxTermDurationInDays)
	out.MinTermDurationInDays = cloneInt32Pointer(in.MinTermDurationInDays)
	out.SlotDurationInHours = cloneInt32Pointer(in.SlotDurationInHours)
	out.TotalScheduledInstanceHours = cloneInt32Pointer(in.TotalScheduledInstanceHours)
	out.Recurrence = cloneStage121ScheduledInstanceRecurrence(in.Recurrence)
	return out
}

func cloneStage121ScheduledInstance(in ScheduledInstance) ScheduledInstance {
	out := in
	out.CreateDate = cloneTimePointer(in.CreateDate)
	out.InstanceCount = cloneInt32Pointer(in.InstanceCount)
	out.NextSlotStartTime = cloneTimePointer(in.NextSlotStartTime)
	out.PreviousSlotEndTime = cloneTimePointer(in.PreviousSlotEndTime)
	out.Recurrence = cloneStage121ScheduledInstanceRecurrence(in.Recurrence)
	out.SlotDurationInHours = cloneInt32Pointer(in.SlotDurationInHours)
	out.TermEndDate = cloneTimePointer(in.TermEndDate)
	out.TermStartDate = cloneTimePointer(in.TermStartDate)
	out.TotalScheduledInstanceHours = cloneInt32Pointer(in.TotalScheduledInstanceHours)
	return out
}

func cloneStage121ServiceLinkVirtualInterface(in ServiceLinkVirtualInterface) ServiceLinkVirtualInterface {
	out := in
	out.PeerBgpASN = cloneInt64Pointer(in.PeerBgpASN)
	out.VLAN = cloneInt32Pointer(in.VLAN)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneStage121SnapshotTierStatus(in SnapshotTierStatus) SnapshotTierStatus {
	out := in
	out.ArchivalCompleteTime = cloneTimePointer(in.ArchivalCompleteTime)
	out.LastTieringProgress = cloneInt32Pointer(in.LastTieringProgress)
	out.LastTieringStartTime = cloneTimePointer(in.LastTieringStartTime)
	out.RestoreExpiryTime = cloneTimePointer(in.RestoreExpiryTime)
	out.Tags = cloneStringMap(in.Tags)
	return out
}
