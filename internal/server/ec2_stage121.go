package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage121Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DescribeReservedInstancesListings":
		listings, err := s.ec2.DescribeReservedInstancesListings(
			parseEC2OptionalString(r.Form.Get("ReservedInstancesId")),
			parseEC2OptionalString(r.Form.Get("ReservedInstancesListingId")),
			parseEC2Filters(r.Form),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeReservedInstancesListingsResponse{
			XMLName:                   xml.Name{Local: "DescribeReservedInstancesListingsResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			ReservedInstancesListings: ec2Stage95ReservedInstancesListingSet{Items: ec2Stage95ReservedInstancesListingsFrom(listings)},
		})
		return true
	case "DescribeReservedInstancesModifications":
		modifications, nextToken, err := s.ec2.DescribeReservedInstancesModifications(
			parseEC2MembersOrItemList(r.Form, "ReservedInstancesModificationId"),
			parseEC2Filters(r.Form),
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeReservedInstancesModificationsResponse{
			XMLName:                        xml.Name{Local: "DescribeReservedInstancesModificationsResponse"},
			Xmlns:                          ec2Namespace,
			RequestID:                      "stackyard-request",
			ReservedInstancesModifications: ec2Stage121ReservedInstancesModificationSet{Items: ec2Stage121ReservedInstancesModificationItemsFrom(modifications)},
			NextToken:                      nextToken,
		})
		return true
	case "DescribeReservedInstancesOfferings":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxDuration, ok := parseEC2OptionalInt64(r.Form.Get("MaxDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxInstanceCount, ok := parseEC2OptionalInt32(r.Form.Get("MaxInstanceCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		minDuration, ok := parseEC2OptionalInt64(r.Form.Get("MinDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		var includeMarketplace *bool
		if parsed, has := parseEC2OptionalBool(r.Form.Get("IncludeMarketplace")); has {
			includeMarketplace = &parsed
		}
		filters := parseEC2Filters(r.Form)
		if availabilityZone := strings.TrimSpace(r.Form.Get("AvailabilityZone")); availabilityZone != "" {
			filters["availability-zone"] = append(filters["availability-zone"], availabilityZone)
		}
		if availabilityZoneID := strings.TrimSpace(r.Form.Get("AvailabilityZoneId")); availabilityZoneID != "" {
			filters["availability-zone-id"] = append(filters["availability-zone-id"], availabilityZoneID)
		}
		offerings, nextToken, err := s.ec2.DescribeReservedInstancesOfferings(
			parseEC2MembersOrItemList(r.Form, "ReservedInstancesOfferingId"),
			filters,
			includeMarketplace,
			strings.TrimSpace(r.Form.Get("InstanceTenancy")),
			strings.TrimSpace(r.Form.Get("InstanceType")),
			maxDuration,
			maxInstanceCount,
			minDuration,
			strings.TrimSpace(r.Form.Get("OfferingClass")),
			strings.TrimSpace(r.Form.Get("OfferingType")),
			strings.TrimSpace(r.Form.Get("ProductDescription")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeReservedInstancesOfferingsResponse{
			XMLName:                    xml.Name{Local: "DescribeReservedInstancesOfferingsResponse"},
			Xmlns:                      ec2Namespace,
			RequestID:                  "stackyard-request",
			ReservedInstancesOfferings: ec2Stage121ReservedInstancesOfferingSet{Items: ec2Stage121ReservedInstancesOfferingItemsFrom(offerings)},
			NextToken:                  nextToken,
		})
		return true
	case "DescribeScheduledInstanceAvailability":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxSlotDurationInHours, ok := parseEC2OptionalInt32(r.Form.Get("MaxSlotDurationInHours"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		minSlotDurationInHours, ok := parseEC2OptionalInt32(r.Form.Get("MinSlotDurationInHours"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		firstSlotEarliestTime, ok := parseEC2Stage121OptionalRFC3339Time(r.Form, "FirstSlotStartTimeRange.EarliestTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		firstSlotLatestTime, ok := parseEC2Stage121OptionalRFC3339Time(r.Form, "FirstSlotStartTimeRange.LatestTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		recurrence, ok := parseEC2Stage121ScheduledInstanceRecurrence(r.Form, "Recurrence.")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		availabilitySet, nextToken, err := s.ec2.DescribeScheduledInstanceAvailability(
			parseEC2Filters(r.Form),
			firstSlotEarliestTime,
			firstSlotLatestTime,
			maxSlotDurationInHours,
			minSlotDurationInHours,
			recurrence,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeScheduledInstanceAvailabilityResponse{
			XMLName:                       xml.Name{Local: "DescribeScheduledInstanceAvailabilityResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			ScheduledInstanceAvailability: ec2Stage121ScheduledInstanceAvailabilitySet{Items: ec2Stage121ScheduledInstanceAvailabilityItemsFrom(availabilitySet)},
			NextToken:                     nextToken,
		})
		return true
	case "DescribeScheduledInstances":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		slotStartEarliestTime, ok := parseEC2Stage121OptionalRFC3339Time(r.Form, "SlotStartTimeRange.EarliestTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		slotStartLatestTime, ok := parseEC2Stage121OptionalRFC3339Time(r.Form, "SlotStartTimeRange.LatestTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		scheduledInstances, nextToken, err := s.ec2.DescribeScheduledInstances(
			parseEC2MembersOrItemList(r.Form, "ScheduledInstanceId"),
			parseEC2Filters(r.Form),
			slotStartEarliestTime,
			slotStartLatestTime,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeScheduledInstancesResponse{
			XMLName:            xml.Name{Local: "DescribeScheduledInstancesResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			ScheduledInstances: ec2Stage121ScheduledInstanceSet{Items: ec2Stage121ScheduledInstanceItemsFrom(scheduledInstances)},
			NextToken:          nextToken,
		})
		return true
	case "DescribeServiceLinkVirtualInterfaces":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		virtualInterfaces, nextToken, err := s.ec2.DescribeServiceLinkVirtualInterfaces(
			parseEC2MembersOrItemList(r.Form, "ServiceLinkVirtualInterfaceId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeServiceLinkVirtualInterfacesResponse{
			XMLName:                      xml.Name{Local: "DescribeServiceLinkVirtualInterfacesResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			ServiceLinkVirtualInterfaces: ec2Stage121ServiceLinkVirtualInterfaceSet{Items: ec2Stage121ServiceLinkVirtualInterfaceItemsFrom(virtualInterfaces)},
			NextToken:                    nextToken,
		})
		return true
	case "DescribeSnapshotAttribute":
		snapshotAttribute, err := s.ec2.DescribeSnapshotAttribute(
			strings.TrimSpace(r.Form.Get("SnapshotId")),
			strings.TrimSpace(r.Form.Get("Attribute")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeSnapshotAttributeResponse{
			XMLName:                 xml.Name{Local: "DescribeSnapshotAttributeResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			CreateVolumePermissions: ec2Stage121CreateVolumePermissionSet{Items: ec2Stage121CreateVolumePermissionItemsFrom(snapshotAttribute.CreateVolumePermissions)},
			ProductCodes:            ec2Stage121ProductCodeSet{Items: ec2Stage121ProductCodeItemsFrom(snapshotAttribute.ProductCodes)},
			SnapshotID:              snapshotAttribute.SnapshotID,
		})
		return true
	case "DescribeSnapshotTierStatus":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		snapshotTierStatuses, nextToken, err := s.ec2.DescribeSnapshotTierStatus(
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeSnapshotTierStatusResponse{
			XMLName:              xml.Name{Local: "DescribeSnapshotTierStatusResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			SnapshotTierStatuses: ec2Stage121SnapshotTierStatusSet{Items: ec2Stage121SnapshotTierStatusItemsFrom(snapshotTierStatuses)},
			NextToken:            nextToken,
		})
		return true
	case "DescribeSpotDatafeedSubscription":
		var spotDatafeedSubscription *ec2Stage110SpotDatafeedSubscriptionItem
		if subscription := s.ec2.DescribeSpotDatafeedSubscription(); subscription != nil {
			item := ec2Stage110SpotDatafeedSubscriptionItemFrom(*subscription)
			spotDatafeedSubscription = &item
		}
		respondEC2XML(w, ec2Stage121DescribeSpotDatafeedSubscriptionResponse{
			XMLName:                  xml.Name{Local: "DescribeSpotDatafeedSubscriptionResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			SpotDatafeedSubscription: spotDatafeedSubscription,
		})
		return true
	case "DescribeSpotFleetInstances":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		spotFleetRequestID, activeInstances, nextToken, err := s.ec2.DescribeSpotFleetInstances(
			strings.TrimSpace(r.Form.Get("SpotFleetRequestId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage121DescribeSpotFleetInstancesResponse{
			XMLName:            xml.Name{Local: "DescribeSpotFleetInstancesResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			ActiveInstances:    ec2Stage116ActiveInstanceSet{Items: ec2Stage116ActiveInstanceItemsFrom(activeInstances)},
			NextToken:          nextToken,
			SpotFleetRequestID: spotFleetRequestID,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage121OptionalRFC3339Time(values url.Values, key string) (*time.Time, bool) {
	value := strings.TrimSpace(values.Get(key))
	if value == "" {
		return nil, true
	}
	parsed, err := parseEC2RFC3339Time(value)
	if err != nil {
		return nil, false
	}
	return &parsed, true
}

func parseEC2Stage121ScheduledInstanceRecurrence(values url.Values, prefix string) (*ec2svc.ScheduledInstanceRecurrence, bool) {
	frequency := strings.TrimSpace(values.Get(prefix + "Frequency"))
	interval, ok := parseEC2OptionalInt32(values.Get(prefix + "Interval"))
	if !ok {
		return nil, false
	}
	occurrenceRelativeToEndValue, occurrenceRelativeToEndSet := parseEC2OptionalBool(values.Get(prefix + "OccurrenceRelativeToEnd"))
	occurrenceUnit := strings.TrimSpace(values.Get(prefix + "OccurrenceUnit"))
	occurrenceDays, ok := parseEC2Stage121Int32Members(values, prefix+"OccurrenceDay.")
	if !ok {
		return nil, false
	}

	if frequency == "" && interval == nil && !occurrenceRelativeToEndSet && occurrenceUnit == "" && len(occurrenceDays) == 0 {
		return nil, true
	}

	recurrence := &ec2svc.ScheduledInstanceRecurrence{
		Frequency:      frequency,
		Interval:       interval,
		OccurrenceDays: occurrenceDays,
		OccurrenceUnit: occurrenceUnit,
	}
	if occurrenceRelativeToEndSet {
		recurrence.OccurrenceRelativeToEnd = &occurrenceRelativeToEndValue
	}
	return recurrence, true
}

func parseEC2Stage121Int32Members(values url.Values, prefix string) ([]int32, bool) {
	members := parseEC2Members(values, prefix)
	if len(members) == 0 {
		return nil, true
	}
	out := make([]int32, 0, len(members))
	for _, member := range members {
		parsed, err := strconv.ParseInt(strings.TrimSpace(member), 10, 32)
		if err != nil {
			return nil, false
		}
		out = append(out, int32(parsed))
	}
	return out, true
}

func ec2Stage121TimeString(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func ec2Stage121ReservedInstancesModificationItemsFrom(in []ec2svc.ReservedInstancesModification) []ec2Stage121ReservedInstancesModificationItem {
	out := make([]ec2Stage121ReservedInstancesModificationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ReservedInstancesModificationItem{
			ClientToken:                     item.ClientToken,
			CreateDate:                      ec2Stage121TimeString(item.CreateDate),
			EffectiveDate:                   ec2Stage121TimeString(item.EffectiveDate),
			ModificationResults:             ec2Stage121ReservedInstancesModificationResultSet{Items: ec2Stage121ReservedInstancesModificationResultItemsFrom(item.ModificationResults)},
			ReservedInstancesIDs:            ec2StringSet{Items: append([]string(nil), item.ReservedInstancesIDs...)},
			ReservedInstancesModificationID: item.ReservedInstancesModificationID,
			Status:                          item.Status,
			StatusMessage:                   item.StatusMessage,
			UpdateDate:                      ec2Stage121TimeString(item.UpdateDate),
		})
	}
	return out
}

func ec2Stage121ReservedInstancesModificationResultItemsFrom(in []ec2svc.ReservedInstancesModificationResult) []ec2Stage121ReservedInstancesModificationResultItem {
	out := make([]ec2Stage121ReservedInstancesModificationResultItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ReservedInstancesModificationResultItem{ReservedInstancesID: item.ReservedInstancesID})
	}
	return out
}

func ec2Stage121ReservedInstancesOfferingItemsFrom(in []ec2svc.ReservedInstancesOffering) []ec2Stage121ReservedInstancesOfferingItem {
	out := make([]ec2Stage121ReservedInstancesOfferingItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ReservedInstancesOfferingItem{
			AvailabilityZone:            item.AvailabilityZone,
			CurrencyCode:                item.CurrencyCode,
			Duration:                    item.Duration,
			FixedPrice:                  item.FixedPrice,
			InstanceTenancy:             item.InstanceTenancy,
			InstanceType:                item.InstanceType,
			Marketplace:                 item.Marketplace,
			OfferingClass:               item.OfferingClass,
			OfferingType:                item.OfferingType,
			PricingDetails:              ec2Stage121PricingDetailSet{Items: ec2Stage121PricingDetailItemsFrom(item.PricingDetails)},
			ProductDescription:          item.ProductDescription,
			RecurringCharges:            ec2Stage121RecurringChargeSet{Items: ec2Stage121RecurringChargeItemsFrom(item.RecurringCharges)},
			ReservedInstancesOfferingID: item.ReservedInstancesOfferingID,
			Scope:                       item.Scope,
			UsagePrice:                  item.UsagePrice,
		})
	}
	return out
}

func ec2Stage121PricingDetailItemsFrom(in []ec2svc.ReservedInstancesOfferingPricingDetail) []ec2Stage121PricingDetailItem {
	out := make([]ec2Stage121PricingDetailItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121PricingDetailItem{Count: item.Count, Price: item.Price})
	}
	return out
}

func ec2Stage121RecurringChargeItemsFrom(in []ec2svc.ReservedInstancesOfferingRecurringCharge) []ec2Stage121RecurringChargeItem {
	out := make([]ec2Stage121RecurringChargeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121RecurringChargeItem{Amount: item.Amount, Frequency: item.Frequency})
	}
	return out
}

func ec2Stage121ScheduledInstanceAvailabilityItemsFrom(in []ec2svc.ScheduledInstanceAvailability) []ec2Stage121ScheduledInstanceAvailabilityItem {
	out := make([]ec2Stage121ScheduledInstanceAvailabilityItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ScheduledInstanceAvailabilityItem{
			AvailabilityZone:            item.AvailabilityZone,
			AvailableInstanceCount:      item.AvailableInstanceCount,
			FirstSlotStartTime:          ec2Stage121TimeString(item.FirstSlotStartTime),
			HourlyPrice:                 item.HourlyPrice,
			InstanceType:                item.InstanceType,
			MaxTermDurationInDays:       item.MaxTermDurationInDays,
			MinTermDurationInDays:       item.MinTermDurationInDays,
			NetworkPlatform:             item.NetworkPlatform,
			Platform:                    item.Platform,
			PurchaseToken:               item.PurchaseToken,
			Recurrence:                  ec2Stage121ScheduledInstanceRecurrenceItemFrom(item.Recurrence),
			SlotDurationInHours:         item.SlotDurationInHours,
			TotalScheduledInstanceHours: item.TotalScheduledInstanceHours,
		})
	}
	return out
}

func ec2Stage121ScheduledInstanceItemsFrom(in []ec2svc.ScheduledInstance) []ec2Stage121ScheduledInstanceItem {
	out := make([]ec2Stage121ScheduledInstanceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ScheduledInstanceItem{
			AvailabilityZone:            item.AvailabilityZone,
			CreateDate:                  ec2Stage121TimeString(item.CreateDate),
			HourlyPrice:                 item.HourlyPrice,
			InstanceCount:               item.InstanceCount,
			InstanceType:                item.InstanceType,
			NetworkPlatform:             item.NetworkPlatform,
			NextSlotStartTime:           ec2Stage121TimeString(item.NextSlotStartTime),
			Platform:                    item.Platform,
			PreviousSlotEndTime:         ec2Stage121TimeString(item.PreviousSlotEndTime),
			Recurrence:                  ec2Stage121ScheduledInstanceRecurrenceItemFrom(item.Recurrence),
			ScheduledInstanceID:         item.ScheduledInstanceID,
			SlotDurationInHours:         item.SlotDurationInHours,
			TermEndDate:                 ec2Stage121TimeString(item.TermEndDate),
			TermStartDate:               ec2Stage121TimeString(item.TermStartDate),
			TotalScheduledInstanceHours: item.TotalScheduledInstanceHours,
		})
	}
	return out
}

func ec2Stage121ScheduledInstanceRecurrenceItemFrom(in *ec2svc.ScheduledInstanceRecurrence) ec2Stage121ScheduledInstanceRecurrenceItem {
	if in == nil {
		return ec2Stage121ScheduledInstanceRecurrenceItem{}
	}
	return ec2Stage121ScheduledInstanceRecurrenceItem{
		Frequency:               in.Frequency,
		Interval:                in.Interval,
		OccurrenceDays:          ec2Stage121OccurrenceDaySet{Items: append([]int32(nil), in.OccurrenceDays...)},
		OccurrenceRelativeToEnd: in.OccurrenceRelativeToEnd,
		OccurrenceUnit:          in.OccurrenceUnit,
	}
}

func ec2Stage121ServiceLinkVirtualInterfaceItemsFrom(in []ec2svc.ServiceLinkVirtualInterface) []ec2Stage121ServiceLinkVirtualInterfaceItem {
	out := make([]ec2Stage121ServiceLinkVirtualInterfaceItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ServiceLinkVirtualInterfaceItem{
			ConfigurationState:             item.ConfigurationState,
			LocalAddress:                   item.LocalAddress,
			OutpostARN:                     item.OutpostARN,
			OutpostID:                      item.OutpostID,
			OutpostLagID:                   item.OutpostLagID,
			OwnerID:                        item.OwnerID,
			PeerAddress:                    item.PeerAddress,
			PeerBgpASN:                     item.PeerBgpASN,
			ServiceLinkVirtualInterfaceARN: item.ServiceLinkVirtualInterfaceARN,
			ServiceLinkVirtualInterfaceID:  item.ServiceLinkVirtualInterfaceID,
			TagSet:                         ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			VLAN:                           item.VLAN,
		})
	}
	return out
}

func ec2Stage121CreateVolumePermissionItemsFrom(in []ec2svc.SnapshotCreateVolumePermission) []ec2Stage121CreateVolumePermissionItem {
	out := make([]ec2Stage121CreateVolumePermissionItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121CreateVolumePermissionItem{Group: item.Group, UserID: item.UserID})
	}
	return out
}

func ec2Stage121ProductCodeItemsFrom(in []ec2svc.SnapshotProductCode) []ec2Stage121ProductCodeItem {
	out := make([]ec2Stage121ProductCodeItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121ProductCodeItem{ProductCode: item.ProductCode, Type: item.Type})
	}
	return out
}

func ec2Stage121SnapshotTierStatusItemsFrom(in []ec2svc.SnapshotTierStatus) []ec2Stage121SnapshotTierStatusItem {
	out := make([]ec2Stage121SnapshotTierStatusItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage121SnapshotTierStatusItem{
			ArchivalCompleteTime:             ec2Stage121TimeString(item.ArchivalCompleteTime),
			LastTieringOperationStatus:       item.LastTieringOperationStatus,
			LastTieringOperationStatusDetail: item.LastTieringOperationStatusDetail,
			LastTieringProgress:              item.LastTieringProgress,
			LastTieringStartTime:             ec2Stage121TimeString(item.LastTieringStartTime),
			OwnerID:                          item.OwnerID,
			RestoreExpiryTime:                ec2Stage121TimeString(item.RestoreExpiryTime),
			SnapshotID:                       item.SnapshotID,
			Status:                           item.Status,
			StorageTier:                      item.StorageTier,
			TagSet:                           ec2TagSet{Items: ec2TagItemsFromMap(item.Tags)},
			VolumeID:                         item.VolumeID,
		})
	}
	return out
}

type ec2Stage121DescribeReservedInstancesListingsResponse struct {
	XMLName                   xml.Name                              `xml:"DescribeReservedInstancesListingsResponse"`
	Xmlns                     string                                `xml:"xmlns,attr"`
	RequestID                 string                                `xml:"requestId"`
	ReservedInstancesListings ec2Stage95ReservedInstancesListingSet `xml:"reservedInstancesListingsSet"`
}

type ec2Stage121DescribeReservedInstancesModificationsResponse struct {
	XMLName                        xml.Name                                    `xml:"DescribeReservedInstancesModificationsResponse"`
	Xmlns                          string                                      `xml:"xmlns,attr"`
	RequestID                      string                                      `xml:"requestId"`
	ReservedInstancesModifications ec2Stage121ReservedInstancesModificationSet `xml:"reservedInstancesModificationsSet"`
	NextToken                      *string                                     `xml:"nextToken,omitempty"`
}

type ec2Stage121ReservedInstancesModificationSet struct {
	Items []ec2Stage121ReservedInstancesModificationItem `xml:"item"`
}

type ec2Stage121ReservedInstancesModificationItem struct {
	ClientToken                     string                                            `xml:"clientToken,omitempty"`
	CreateDate                      string                                            `xml:"createDate,omitempty"`
	EffectiveDate                   string                                            `xml:"effectiveDate,omitempty"`
	ModificationResults             ec2Stage121ReservedInstancesModificationResultSet `xml:"modificationResultSet"`
	ReservedInstancesIDs            ec2StringSet                                      `xml:"reservedInstancesSet"`
	ReservedInstancesModificationID string                                            `xml:"reservedInstancesModificationId,omitempty"`
	Status                          string                                            `xml:"status,omitempty"`
	StatusMessage                   string                                            `xml:"statusMessage,omitempty"`
	UpdateDate                      string                                            `xml:"updateDate,omitempty"`
}

type ec2Stage121ReservedInstancesModificationResultSet struct {
	Items []ec2Stage121ReservedInstancesModificationResultItem `xml:"item"`
}

type ec2Stage121ReservedInstancesModificationResultItem struct {
	ReservedInstancesID string `xml:"reservedInstancesId,omitempty"`
}

type ec2Stage121DescribeReservedInstancesOfferingsResponse struct {
	XMLName                    xml.Name                                `xml:"DescribeReservedInstancesOfferingsResponse"`
	Xmlns                      string                                  `xml:"xmlns,attr"`
	RequestID                  string                                  `xml:"requestId"`
	ReservedInstancesOfferings ec2Stage121ReservedInstancesOfferingSet `xml:"reservedInstancesOfferingsSet"`
	NextToken                  *string                                 `xml:"nextToken,omitempty"`
}

type ec2Stage121ReservedInstancesOfferingSet struct {
	Items []ec2Stage121ReservedInstancesOfferingItem `xml:"item"`
}

type ec2Stage121ReservedInstancesOfferingItem struct {
	AvailabilityZone            string                        `xml:"availabilityZone,omitempty"`
	CurrencyCode                string                        `xml:"currencyCode,omitempty"`
	Duration                    *int64                        `xml:"duration,omitempty"`
	FixedPrice                  *float32                      `xml:"fixedPrice,omitempty"`
	InstanceTenancy             string                        `xml:"instanceTenancy,omitempty"`
	InstanceType                string                        `xml:"instanceType,omitempty"`
	Marketplace                 *bool                         `xml:"marketplace,omitempty"`
	OfferingClass               string                        `xml:"offeringClass,omitempty"`
	OfferingType                string                        `xml:"offeringType,omitempty"`
	PricingDetails              ec2Stage121PricingDetailSet   `xml:"pricingDetailsSet"`
	ProductDescription          string                        `xml:"productDescription,omitempty"`
	RecurringCharges            ec2Stage121RecurringChargeSet `xml:"recurringCharges"`
	ReservedInstancesOfferingID string                        `xml:"reservedInstancesOfferingId,omitempty"`
	Scope                       string                        `xml:"scope,omitempty"`
	UsagePrice                  *float32                      `xml:"usagePrice,omitempty"`
}

type ec2Stage121PricingDetailSet struct {
	Items []ec2Stage121PricingDetailItem `xml:"item"`
}

type ec2Stage121PricingDetailItem struct {
	Count *int32   `xml:"count,omitempty"`
	Price *float64 `xml:"price,omitempty"`
}

type ec2Stage121RecurringChargeSet struct {
	Items []ec2Stage121RecurringChargeItem `xml:"item"`
}

type ec2Stage121RecurringChargeItem struct {
	Amount    *float64 `xml:"amount,omitempty"`
	Frequency string   `xml:"frequency,omitempty"`
}

type ec2Stage121DescribeScheduledInstanceAvailabilityResponse struct {
	XMLName                       xml.Name                                    `xml:"DescribeScheduledInstanceAvailabilityResponse"`
	Xmlns                         string                                      `xml:"xmlns,attr"`
	RequestID                     string                                      `xml:"requestId"`
	ScheduledInstanceAvailability ec2Stage121ScheduledInstanceAvailabilitySet `xml:"scheduledInstanceAvailabilitySet"`
	NextToken                     *string                                     `xml:"nextToken,omitempty"`
}

type ec2Stage121ScheduledInstanceAvailabilitySet struct {
	Items []ec2Stage121ScheduledInstanceAvailabilityItem `xml:"item"`
}

type ec2Stage121ScheduledInstanceAvailabilityItem struct {
	AvailabilityZone            string                                     `xml:"availabilityZone,omitempty"`
	AvailableInstanceCount      *int32                                     `xml:"availableInstanceCount,omitempty"`
	FirstSlotStartTime          string                                     `xml:"firstSlotStartTime,omitempty"`
	HourlyPrice                 string                                     `xml:"hourlyPrice,omitempty"`
	InstanceType                string                                     `xml:"instanceType,omitempty"`
	MaxTermDurationInDays       *int32                                     `xml:"maxTermDurationInDays,omitempty"`
	MinTermDurationInDays       *int32                                     `xml:"minTermDurationInDays,omitempty"`
	NetworkPlatform             string                                     `xml:"networkPlatform,omitempty"`
	Platform                    string                                     `xml:"platform,omitempty"`
	PurchaseToken               string                                     `xml:"purchaseToken,omitempty"`
	Recurrence                  ec2Stage121ScheduledInstanceRecurrenceItem `xml:"recurrence"`
	SlotDurationInHours         *int32                                     `xml:"slotDurationInHours,omitempty"`
	TotalScheduledInstanceHours *int32                                     `xml:"totalScheduledInstanceHours,omitempty"`
}

type ec2Stage121DescribeScheduledInstancesResponse struct {
	XMLName            xml.Name                        `xml:"DescribeScheduledInstancesResponse"`
	Xmlns              string                          `xml:"xmlns,attr"`
	RequestID          string                          `xml:"requestId"`
	ScheduledInstances ec2Stage121ScheduledInstanceSet `xml:"scheduledInstanceSet"`
	NextToken          *string                         `xml:"nextToken,omitempty"`
}

type ec2Stage121ScheduledInstanceSet struct {
	Items []ec2Stage121ScheduledInstanceItem `xml:"item"`
}

type ec2Stage121ScheduledInstanceItem struct {
	AvailabilityZone            string                                     `xml:"availabilityZone,omitempty"`
	CreateDate                  string                                     `xml:"createDate,omitempty"`
	HourlyPrice                 string                                     `xml:"hourlyPrice,omitempty"`
	InstanceCount               *int32                                     `xml:"instanceCount,omitempty"`
	InstanceType                string                                     `xml:"instanceType,omitempty"`
	NetworkPlatform             string                                     `xml:"networkPlatform,omitempty"`
	NextSlotStartTime           string                                     `xml:"nextSlotStartTime,omitempty"`
	Platform                    string                                     `xml:"platform,omitempty"`
	PreviousSlotEndTime         string                                     `xml:"previousSlotEndTime,omitempty"`
	Recurrence                  ec2Stage121ScheduledInstanceRecurrenceItem `xml:"recurrence"`
	ScheduledInstanceID         string                                     `xml:"scheduledInstanceId,omitempty"`
	SlotDurationInHours         *int32                                     `xml:"slotDurationInHours,omitempty"`
	TermEndDate                 string                                     `xml:"termEndDate,omitempty"`
	TermStartDate               string                                     `xml:"termStartDate,omitempty"`
	TotalScheduledInstanceHours *int32                                     `xml:"totalScheduledInstanceHours,omitempty"`
}

type ec2Stage121ScheduledInstanceRecurrenceItem struct {
	Frequency               string                      `xml:"frequency,omitempty"`
	Interval                *int32                      `xml:"interval,omitempty"`
	OccurrenceDays          ec2Stage121OccurrenceDaySet `xml:"occurrenceDaySet"`
	OccurrenceRelativeToEnd *bool                       `xml:"occurrenceRelativeToEnd,omitempty"`
	OccurrenceUnit          string                      `xml:"occurrenceUnit,omitempty"`
}

type ec2Stage121OccurrenceDaySet struct {
	Items []int32 `xml:"item"`
}

type ec2Stage121DescribeServiceLinkVirtualInterfacesResponse struct {
	XMLName                      xml.Name                                  `xml:"DescribeServiceLinkVirtualInterfacesResponse"`
	Xmlns                        string                                    `xml:"xmlns,attr"`
	RequestID                    string                                    `xml:"requestId"`
	ServiceLinkVirtualInterfaces ec2Stage121ServiceLinkVirtualInterfaceSet `xml:"serviceLinkVirtualInterfaceSet"`
	NextToken                    *string                                   `xml:"nextToken,omitempty"`
}

type ec2Stage121ServiceLinkVirtualInterfaceSet struct {
	Items []ec2Stage121ServiceLinkVirtualInterfaceItem `xml:"item"`
}

type ec2Stage121ServiceLinkVirtualInterfaceItem struct {
	ConfigurationState             string    `xml:"configurationState,omitempty"`
	LocalAddress                   string    `xml:"localAddress,omitempty"`
	OutpostARN                     string    `xml:"outpostArn,omitempty"`
	OutpostID                      string    `xml:"outpostId,omitempty"`
	OutpostLagID                   string    `xml:"outpostLagId,omitempty"`
	OwnerID                        string    `xml:"ownerId,omitempty"`
	PeerAddress                    string    `xml:"peerAddress,omitempty"`
	PeerBgpASN                     *int64    `xml:"peerBgpAsn,omitempty"`
	ServiceLinkVirtualInterfaceARN string    `xml:"serviceLinkVirtualInterfaceArn,omitempty"`
	ServiceLinkVirtualInterfaceID  string    `xml:"serviceLinkVirtualInterfaceId,omitempty"`
	TagSet                         ec2TagSet `xml:"tagSet"`
	VLAN                           *int32    `xml:"vlan,omitempty"`
}

type ec2Stage121DescribeSnapshotAttributeResponse struct {
	XMLName                 xml.Name                             `xml:"DescribeSnapshotAttributeResponse"`
	Xmlns                   string                               `xml:"xmlns,attr"`
	RequestID               string                               `xml:"requestId"`
	CreateVolumePermissions ec2Stage121CreateVolumePermissionSet `xml:"createVolumePermission"`
	ProductCodes            ec2Stage121ProductCodeSet            `xml:"productCodes"`
	SnapshotID              string                               `xml:"snapshotId,omitempty"`
}

type ec2Stage121CreateVolumePermissionSet struct {
	Items []ec2Stage121CreateVolumePermissionItem `xml:"item"`
}

type ec2Stage121CreateVolumePermissionItem struct {
	Group  string `xml:"group,omitempty"`
	UserID string `xml:"userId,omitempty"`
}

type ec2Stage121ProductCodeSet struct {
	Items []ec2Stage121ProductCodeItem `xml:"item"`
}

type ec2Stage121ProductCodeItem struct {
	ProductCode string `xml:"productCode,omitempty"`
	Type        string `xml:"type,omitempty"`
}

type ec2Stage121DescribeSnapshotTierStatusResponse struct {
	XMLName              xml.Name                         `xml:"DescribeSnapshotTierStatusResponse"`
	Xmlns                string                           `xml:"xmlns,attr"`
	RequestID            string                           `xml:"requestId"`
	SnapshotTierStatuses ec2Stage121SnapshotTierStatusSet `xml:"snapshotTierStatusSet"`
	NextToken            *string                          `xml:"nextToken,omitempty"`
}

type ec2Stage121SnapshotTierStatusSet struct {
	Items []ec2Stage121SnapshotTierStatusItem `xml:"item"`
}

type ec2Stage121SnapshotTierStatusItem struct {
	ArchivalCompleteTime             string    `xml:"archivalCompleteTime,omitempty"`
	LastTieringOperationStatus       string    `xml:"lastTieringOperationStatus,omitempty"`
	LastTieringOperationStatusDetail string    `xml:"lastTieringOperationStatusDetail,omitempty"`
	LastTieringProgress              *int32    `xml:"lastTieringProgress,omitempty"`
	LastTieringStartTime             string    `xml:"lastTieringStartTime,omitempty"`
	OwnerID                          string    `xml:"ownerId,omitempty"`
	RestoreExpiryTime                string    `xml:"restoreExpiryTime,omitempty"`
	SnapshotID                       string    `xml:"snapshotId,omitempty"`
	Status                           string    `xml:"status,omitempty"`
	StorageTier                      string    `xml:"storageTier,omitempty"`
	TagSet                           ec2TagSet `xml:"tagSet"`
	VolumeID                         string    `xml:"volumeId,omitempty"`
}

type ec2Stage121DescribeSpotDatafeedSubscriptionResponse struct {
	XMLName                  xml.Name                                 `xml:"DescribeSpotDatafeedSubscriptionResponse"`
	Xmlns                    string                                   `xml:"xmlns,attr"`
	RequestID                string                                   `xml:"requestId"`
	SpotDatafeedSubscription *ec2Stage110SpotDatafeedSubscriptionItem `xml:"spotDatafeedSubscription,omitempty"`
}

type ec2Stage121DescribeSpotFleetInstancesResponse struct {
	XMLName            xml.Name                     `xml:"DescribeSpotFleetInstancesResponse"`
	Xmlns              string                       `xml:"xmlns,attr"`
	RequestID          string                       `xml:"requestId"`
	ActiveInstances    ec2Stage116ActiveInstanceSet `xml:"activeInstanceSet"`
	NextToken          *string                      `xml:"nextToken,omitempty"`
	SpotFleetRequestID string                       `xml:"spotFleetRequestId,omitempty"`
}
