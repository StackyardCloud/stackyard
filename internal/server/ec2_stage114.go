package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage114Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeprovisionIpamByoasn":
		byoasn, err := s.ec2.DeprovisionIpamByoasn(
			strings.TrimSpace(r.Form.Get("Asn")),
			strings.TrimSpace(r.Form.Get("IpamId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DeprovisionIpamByoasnResponse{
			XMLName:   xml.Name{Local: "DeprovisionIpamByoasnResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Byoasn:    ec2Stage114ByoasnItemFrom(byoasn),
		})
		return true
	case "DeprovisionIpamPoolCidr":
		ipamPoolCidr, err := s.ec2.DeprovisionIpamPoolCidr(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			parseEC2OptionalString(r.Form.Get("Cidr")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DeprovisionIpamPoolCidrResponse{
			XMLName:      xml.Name{Local: "DeprovisionIpamPoolCidrResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			IpamPoolCidr: ec2Stage114IpamPoolCidrItemFrom(ipamPoolCidr),
		})
		return true
	case "DeprovisionPublicIpv4PoolCidr":
		deprovisionedAddresses, poolID, err := s.ec2.DeprovisionPublicIpv4PoolCidr(
			strings.TrimSpace(r.Form.Get("PoolId")),
			strings.TrimSpace(r.Form.Get("Cidr")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DeprovisionPublicIpv4PoolCidrResponse{
			XMLName:                 xml.Name{Local: "DeprovisionPublicIpv4PoolCidrResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			DeprovisionedAddressSet: ec2StringSet{Items: deprovisionedAddresses},
			PoolID:                  poolID,
		})
		return true
	case "DeregisterInstanceEventNotificationAttributes":
		var includeAllTagsOfInstance *bool
		if hasEC2Field(r.Form, "InstanceTagAttribute.IncludeAllTagsOfInstance") {
			parsed, ok := parseEC2OptionalBool(r.Form.Get("InstanceTagAttribute.IncludeAllTagsOfInstance"))
			if !ok {
				respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
				return true
			}
			includeAllTagsOfInstance = &parsed
		}

		attribute, err := s.ec2.DeregisterInstanceEventNotificationAttributes(
			includeAllTagsOfInstance,
			parseEC2MembersOrItemList(r.Form, "InstanceTagAttribute.InstanceTagKey"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DeregisterInstanceEventNotificationAttributesResponse{
			XMLName:              xml.Name{Local: "DeregisterInstanceEventNotificationAttributesResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			InstanceTagAttribute: ec2Stage114InstanceTagNotificationAttributeItemFrom(attribute),
		})
		return true
	case "DescribeBundleTasks":
		bundleTasks := s.ec2.DescribeBundleTasks(
			parseEC2MembersOrItemList(r.Form, "BundleId"),
			parseEC2Filters(r.Form),
		)
		respondEC2XML(w, ec2Stage114DescribeBundleTasksResponse{
			XMLName:                xml.Name{Local: "DescribeBundleTasksResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			BundleInstanceTasksSet: ec2Stage114BundleTaskSet{Items: ec2Stage114BundleTaskItemsFrom(bundleTasks)},
		})
		return true
	case "DescribeByoipCidrs":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok || maxResults == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		byoipCidrs, nextToken, err := s.ec2.DescribeByoipCidrs(
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DescribeByoipCidrsResponse{
			XMLName:      xml.Name{Local: "DescribeByoipCidrsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			ByoipCidrSet: ec2Stage114ByoipCidrSet{Items: ec2Stage114ByoipCidrItemsFrom(byoipCidrs)},
			NextToken:    nextToken,
		})
		return true
	case "DescribeCapacityBlockExtensionHistory":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		items, nextToken, err := s.ec2.DescribeCapacityBlockExtensionHistory(
			parseEC2MembersOrItemList(r.Form, "CapacityReservationId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DescribeCapacityBlockExtensionHistoryResponse{
			XMLName:                   xml.Name{Local: "DescribeCapacityBlockExtensionHistoryResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			CapacityBlockExtensionSet: ec2Stage114CapacityBlockExtensionSet{Items: ec2Stage114CapacityBlockExtensionItemsFrom(items)},
			NextToken:                 nextToken,
		})
		return true
	case "DescribeCapacityBlockExtensionOfferings":
		capacityBlockExtensionDurationHours, ok := parseEC2OptionalInt32(r.Form.Get("CapacityBlockExtensionDurationHours"))
		if !ok || capacityBlockExtensionDurationHours == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		items, nextToken, err := s.ec2.DescribeCapacityBlockExtensionOfferings(
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
			*capacityBlockExtensionDurationHours,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DescribeCapacityBlockExtensionOfferingsResponse{
			XMLName:                           xml.Name{Local: "DescribeCapacityBlockExtensionOfferingsResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			CapacityBlockExtensionOfferingSet: ec2Stage114CapacityBlockExtensionOfferingSet{Items: ec2Stage114CapacityBlockExtensionOfferingItemsFrom(items)},
			NextToken:                         nextToken,
		})
		return true
	case "DescribeCapacityBlockOfferings":
		capacityDurationHours, ok := parseEC2OptionalInt32(r.Form.Get("CapacityDurationHours"))
		if !ok || capacityDurationHours == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endDateRange, ok := parseEC2Stage114OptionalRFC3339Time(r.Form, "EndDateRange")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		startDateRange, ok := parseEC2Stage114OptionalRFC3339Time(r.Form, "StartDateRange")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ultraserverCount, ok := parseEC2OptionalInt32(r.Form.Get("UltraserverCount"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		items, nextToken, err := s.ec2.DescribeCapacityBlockOfferings(
			*capacityDurationHours,
			endDateRange,
			instanceCount,
			parseEC2OptionalString(r.Form.Get("InstanceType")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
			startDateRange,
			ultraserverCount,
			parseEC2OptionalString(r.Form.Get("UltraserverType")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DescribeCapacityBlockOfferingsResponse{
			XMLName:                  xml.Name{Local: "DescribeCapacityBlockOfferingsResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			CapacityBlockOfferingSet: ec2Stage114CapacityBlockOfferingSet{Items: ec2Stage114CapacityBlockOfferingItemsFrom(items)},
			NextToken:                nextToken,
		})
		return true
	case "DescribeCapacityBlockStatus":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		items, nextToken, err := s.ec2.DescribeCapacityBlockStatus(
			parseEC2MembersOrItemList(r.Form, "CapacityBlockId"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage114DescribeCapacityBlockStatusResponse{
			XMLName:                xml.Name{Local: "DescribeCapacityBlockStatusResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			CapacityBlockStatusSet: ec2Stage114CapacityBlockStatusSet{Items: ec2Stage114CapacityBlockStatusItemsFrom(items)},
			NextToken:              nextToken,
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage114OptionalRFC3339Time(values url.Values, key string) (*time.Time, bool) {
	if !hasEC2Field(values, key) {
		return nil, true
	}
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

func ec2Stage114ByoasnItemFrom(in ec2svc.Byoasn) ec2Stage114ByoasnItem {
	return ec2Stage114ByoasnItem{
		Asn:           in.Asn,
		IpamID:        in.IpamID,
		State:         in.State,
		StatusMessage: in.StatusMessage,
	}
}

func ec2Stage114IpamPoolCidrItemFrom(in ec2svc.IpamPoolCidr) ec2Stage114IpamPoolCidrItem {
	out := ec2Stage114IpamPoolCidrItem{
		Cidr:           in.Cidr,
		IpamPoolCidrID: in.IpamPoolCidrID,
		NetmaskLength:  in.NetmaskLength,
		State:          in.State,
	}
	if in.FailureReason != nil {
		out.FailureReason = &ec2Stage114IpamPoolCidrFailureReasonItem{
			Code:    in.FailureReason.Code,
			Message: in.FailureReason.Message,
		}
	}
	return out
}

func ec2Stage114InstanceTagNotificationAttributeItemFrom(in ec2svc.InstanceTagNotificationAttribute) ec2Stage114InstanceTagNotificationAttributeItem {
	out := ec2Stage114InstanceTagNotificationAttributeItem{
		IncludeAllTagsOfInstance: in.IncludeAllTagsOfInstance,
	}
	if len(in.InstanceTagKeys) > 0 {
		out.InstanceTagKeySet = &ec2StringSet{Items: in.InstanceTagKeys}
	}
	return out
}

func ec2Stage114BundleTaskItemsFrom(in []ec2svc.BundleTask) []ec2Stage87BundleTaskItem {
	out := make([]ec2Stage87BundleTaskItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage87BundleTaskItemFrom(item))
	}
	return out
}

func ec2Stage114ByoipCidrItemsFrom(in []ec2svc.ByoipCidr) []ec2ByoipCidrItem {
	out := make([]ec2ByoipCidrItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2ByoipCidrItemFrom(item))
	}
	return out
}

func ec2Stage114CapacityBlockExtensionItemsFrom(in []ec2svc.CapacityBlockExtension) []ec2Stage114CapacityBlockExtensionItem {
	out := make([]ec2Stage114CapacityBlockExtensionItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage114CapacityBlockExtensionItem{
			AvailabilityZone:                    item.AvailabilityZone,
			AvailabilityZoneID:                  item.AvailabilityZoneID,
			CapacityBlockExtensionDurationHours: item.CapacityBlockExtensionDurationHours,
			CapacityBlockExtensionEndDate:       ec2OptionalRFC3339(item.CapacityBlockExtensionEndDate),
			CapacityBlockExtensionOfferingID:    item.CapacityBlockExtensionOfferingID,
			CapacityBlockExtensionPurchaseDate:  ec2OptionalRFC3339(item.CapacityBlockExtensionPurchaseDate),
			CapacityBlockExtensionStartDate:     ec2OptionalRFC3339(item.CapacityBlockExtensionStartDate),
			CapacityBlockExtensionStatus:        item.CapacityBlockExtensionStatus,
			CapacityReservationID:               item.CapacityReservationID,
			CurrencyCode:                        item.CurrencyCode,
			InstanceCount:                       item.InstanceCount,
			InstanceType:                        item.InstanceType,
			UpfrontFee:                          item.UpfrontFee,
		})
	}
	return out
}

func ec2Stage114CapacityBlockExtensionOfferingItemsFrom(in []ec2svc.CapacityBlockExtensionOffering) []ec2Stage114CapacityBlockExtensionOfferingItem {
	out := make([]ec2Stage114CapacityBlockExtensionOfferingItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage114CapacityBlockExtensionOfferingItem{
			AvailabilityZone:                    item.AvailabilityZone,
			AvailabilityZoneID:                  item.AvailabilityZoneID,
			CapacityBlockExtensionDurationHours: item.CapacityBlockExtensionDurationHours,
			CapacityBlockExtensionEndDate:       ec2OptionalRFC3339(item.CapacityBlockExtensionEndDate),
			CapacityBlockExtensionOfferingID:    item.CapacityBlockExtensionOfferingID,
			CapacityBlockExtensionStartDate:     ec2OptionalRFC3339(item.CapacityBlockExtensionStartDate),
			CurrencyCode:                        item.CurrencyCode,
			InstanceCount:                       item.InstanceCount,
			InstanceType:                        item.InstanceType,
			StartDate:                           ec2OptionalRFC3339(item.StartDate),
			Tenancy:                             item.Tenancy,
			UpfrontFee:                          item.UpfrontFee,
		})
	}
	return out
}

func ec2Stage114CapacityBlockOfferingItemsFrom(in []ec2svc.CapacityBlockOffering) []ec2Stage114CapacityBlockOfferingItem {
	out := make([]ec2Stage114CapacityBlockOfferingItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage114CapacityBlockOfferingItem{
			AvailabilityZone:           item.AvailabilityZone,
			CapacityBlockDurationHours: item.CapacityBlockDurationHours,
			CapacityBlockOfferingID:    item.CapacityBlockOfferingID,
			CurrencyCode:               item.CurrencyCode,
			EndDate:                    ec2OptionalRFC3339(item.EndDate),
			InstanceCount:              item.InstanceCount,
			InstanceType:               item.InstanceType,
			StartDate:                  ec2OptionalRFC3339(item.StartDate),
			Tenancy:                    item.Tenancy,
			UltraserverCount:           item.UltraserverCount,
			UltraserverType:            item.UltraserverType,
			UpfrontFee:                 item.UpfrontFee,
		})
	}
	return out
}

func ec2Stage114CapacityBlockStatusItemsFrom(in []ec2svc.CapacityBlockStatus) []ec2Stage114CapacityBlockStatusItem {
	out := make([]ec2Stage114CapacityBlockStatusItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage114CapacityBlockStatusItem{
			CapacityBlockID: item.CapacityBlockID,
			CapacityReservationStatusSet: ec2Stage114CapacityReservationStatusSet{
				Items: ec2Stage114CapacityReservationStatusItemsFrom(item.CapacityReservationStatuses),
			},
			InterconnectStatus:       item.InterconnectStatus,
			TotalAvailableCapacity:   item.TotalAvailableCapacity,
			TotalCapacity:            item.TotalCapacity,
			TotalUnavailableCapacity: item.TotalUnavailableCapacity,
		})
	}
	return out
}

func ec2Stage114CapacityReservationStatusItemsFrom(in []ec2svc.CapacityReservationStatus) []ec2Stage114CapacityReservationStatusItem {
	out := make([]ec2Stage114CapacityReservationStatusItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage114CapacityReservationStatusItem{
			CapacityReservationID:    item.CapacityReservationID,
			TotalAvailableCapacity:   item.TotalAvailableCapacity,
			TotalCapacity:            item.TotalCapacity,
			TotalUnavailableCapacity: item.TotalUnavailableCapacity,
		})
	}
	return out
}

type ec2Stage114DeprovisionIpamByoasnResponse struct {
	XMLName   xml.Name              `xml:"DeprovisionIpamByoasnResponse"`
	Xmlns     string                `xml:"xmlns,attr"`
	RequestID string                `xml:"requestId"`
	Byoasn    ec2Stage114ByoasnItem `xml:"byoasn"`
}

type ec2Stage114ByoasnItem struct {
	Asn           string `xml:"asn,omitempty"`
	IpamID        string `xml:"ipamId,omitempty"`
	State         string `xml:"state,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}

type ec2Stage114DeprovisionIpamPoolCidrResponse struct {
	XMLName      xml.Name                    `xml:"DeprovisionIpamPoolCidrResponse"`
	Xmlns        string                      `xml:"xmlns,attr"`
	RequestID    string                      `xml:"requestId"`
	IpamPoolCidr ec2Stage114IpamPoolCidrItem `xml:"ipamPoolCidr"`
}

type ec2Stage114IpamPoolCidrItem struct {
	Cidr           string                                    `xml:"cidr,omitempty"`
	FailureReason  *ec2Stage114IpamPoolCidrFailureReasonItem `xml:"failureReason,omitempty"`
	IpamPoolCidrID string                                    `xml:"ipamPoolCidrId,omitempty"`
	NetmaskLength  *int32                                    `xml:"netmaskLength,omitempty"`
	State          string                                    `xml:"state,omitempty"`
}

type ec2Stage114IpamPoolCidrFailureReasonItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage114DeprovisionPublicIpv4PoolCidrResponse struct {
	XMLName                 xml.Name     `xml:"DeprovisionPublicIpv4PoolCidrResponse"`
	Xmlns                   string       `xml:"xmlns,attr"`
	RequestID               string       `xml:"requestId"`
	DeprovisionedAddressSet ec2StringSet `xml:"deprovisionedAddressSet"`
	PoolID                  string       `xml:"poolId,omitempty"`
}

type ec2Stage114DeregisterInstanceEventNotificationAttributesResponse struct {
	XMLName              xml.Name                                        `xml:"DeregisterInstanceEventNotificationAttributesResponse"`
	Xmlns                string                                          `xml:"xmlns,attr"`
	RequestID            string                                          `xml:"requestId"`
	InstanceTagAttribute ec2Stage114InstanceTagNotificationAttributeItem `xml:"instanceTagAttribute"`
}

type ec2Stage114InstanceTagNotificationAttributeItem struct {
	IncludeAllTagsOfInstance *bool         `xml:"includeAllTagsOfInstance,omitempty"`
	InstanceTagKeySet        *ec2StringSet `xml:"instanceTagKeySet,omitempty"`
}

type ec2Stage114DescribeBundleTasksResponse struct {
	XMLName                xml.Name                 `xml:"DescribeBundleTasksResponse"`
	Xmlns                  string                   `xml:"xmlns,attr"`
	RequestID              string                   `xml:"requestId"`
	BundleInstanceTasksSet ec2Stage114BundleTaskSet `xml:"bundleInstanceTasksSet"`
}

type ec2Stage114BundleTaskSet struct {
	Items []ec2Stage87BundleTaskItem `xml:"item"`
}

type ec2Stage114DescribeByoipCidrsResponse struct {
	XMLName      xml.Name                `xml:"DescribeByoipCidrsResponse"`
	Xmlns        string                  `xml:"xmlns,attr"`
	RequestID    string                  `xml:"requestId"`
	ByoipCidrSet ec2Stage114ByoipCidrSet `xml:"byoipCidrSet"`
	NextToken    *string                 `xml:"nextToken,omitempty"`
}

type ec2Stage114ByoipCidrSet struct {
	Items []ec2ByoipCidrItem `xml:"item"`
}

type ec2Stage114DescribeCapacityBlockExtensionHistoryResponse struct {
	XMLName                   xml.Name                             `xml:"DescribeCapacityBlockExtensionHistoryResponse"`
	Xmlns                     string                               `xml:"xmlns,attr"`
	RequestID                 string                               `xml:"requestId"`
	CapacityBlockExtensionSet ec2Stage114CapacityBlockExtensionSet `xml:"capacityBlockExtensionSet"`
	NextToken                 *string                              `xml:"nextToken,omitempty"`
}

type ec2Stage114CapacityBlockExtensionSet struct {
	Items []ec2Stage114CapacityBlockExtensionItem `xml:"item"`
}

type ec2Stage114CapacityBlockExtensionItem struct {
	AvailabilityZone                    *string `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID                  *string `xml:"availabilityZoneId,omitempty"`
	CapacityBlockExtensionDurationHours *int32  `xml:"capacityBlockExtensionDurationHours,omitempty"`
	CapacityBlockExtensionEndDate       string  `xml:"capacityBlockExtensionEndDate,omitempty"`
	CapacityBlockExtensionOfferingID    *string `xml:"capacityBlockExtensionOfferingId,omitempty"`
	CapacityBlockExtensionPurchaseDate  string  `xml:"capacityBlockExtensionPurchaseDate,omitempty"`
	CapacityBlockExtensionStartDate     string  `xml:"capacityBlockExtensionStartDate,omitempty"`
	CapacityBlockExtensionStatus        string  `xml:"capacityBlockExtensionStatus,omitempty"`
	CapacityReservationID               *string `xml:"capacityReservationId,omitempty"`
	CurrencyCode                        *string `xml:"currencyCode,omitempty"`
	InstanceCount                       *int32  `xml:"instanceCount,omitempty"`
	InstanceType                        *string `xml:"instanceType,omitempty"`
	UpfrontFee                          *string `xml:"upfrontFee,omitempty"`
}

type ec2Stage114DescribeCapacityBlockExtensionOfferingsResponse struct {
	XMLName                           xml.Name                                     `xml:"DescribeCapacityBlockExtensionOfferingsResponse"`
	Xmlns                             string                                       `xml:"xmlns,attr"`
	RequestID                         string                                       `xml:"requestId"`
	CapacityBlockExtensionOfferingSet ec2Stage114CapacityBlockExtensionOfferingSet `xml:"capacityBlockExtensionOfferingSet"`
	NextToken                         *string                                      `xml:"nextToken,omitempty"`
}

type ec2Stage114CapacityBlockExtensionOfferingSet struct {
	Items []ec2Stage114CapacityBlockExtensionOfferingItem `xml:"item"`
}

type ec2Stage114CapacityBlockExtensionOfferingItem struct {
	AvailabilityZone                    *string `xml:"availabilityZone,omitempty"`
	AvailabilityZoneID                  *string `xml:"availabilityZoneId,omitempty"`
	CapacityBlockExtensionDurationHours *int32  `xml:"capacityBlockExtensionDurationHours,omitempty"`
	CapacityBlockExtensionEndDate       string  `xml:"capacityBlockExtensionEndDate,omitempty"`
	CapacityBlockExtensionOfferingID    *string `xml:"capacityBlockExtensionOfferingId,omitempty"`
	CapacityBlockExtensionStartDate     string  `xml:"capacityBlockExtensionStartDate,omitempty"`
	CurrencyCode                        *string `xml:"currencyCode,omitempty"`
	InstanceCount                       *int32  `xml:"instanceCount,omitempty"`
	InstanceType                        *string `xml:"instanceType,omitempty"`
	StartDate                           string  `xml:"startDate,omitempty"`
	Tenancy                             *string `xml:"tenancy,omitempty"`
	UpfrontFee                          *string `xml:"upfrontFee,omitempty"`
}

type ec2Stage114DescribeCapacityBlockOfferingsResponse struct {
	XMLName                  xml.Name                            `xml:"DescribeCapacityBlockOfferingsResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	CapacityBlockOfferingSet ec2Stage114CapacityBlockOfferingSet `xml:"capacityBlockOfferingSet"`
	NextToken                *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage114CapacityBlockOfferingSet struct {
	Items []ec2Stage114CapacityBlockOfferingItem `xml:"item"`
}

type ec2Stage114CapacityBlockOfferingItem struct {
	AvailabilityZone             *string `xml:"availabilityZone,omitempty"`
	CapacityBlockDurationHours   *int32  `xml:"capacityBlockDurationHours,omitempty"`
	CapacityBlockOfferingID      *string `xml:"capacityBlockOfferingId,omitempty"`
	CurrencyCode                 *string `xml:"currencyCode,omitempty"`
	EndDate                      string  `xml:"endDate,omitempty"`
	InstanceCount                *int32  `xml:"instanceCount,omitempty"`
	InstanceType                 *string `xml:"instanceType,omitempty"`
	StartDate                    string  `xml:"startDate,omitempty"`
	Tenancy                      *string `xml:"tenancy,omitempty"`
	UltraserverCount             *int32  `xml:"ultraserverCount,omitempty"`
	UltraserverType              *string `xml:"ultraserverType,omitempty"`
	UpfrontFee                   *string `xml:"upfrontFee,omitempty"`
}

type ec2Stage114DescribeCapacityBlockStatusResponse struct {
	XMLName                xml.Name                          `xml:"DescribeCapacityBlockStatusResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	CapacityBlockStatusSet ec2Stage114CapacityBlockStatusSet `xml:"capacityBlockStatusSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage114CapacityBlockStatusSet struct {
	Items []ec2Stage114CapacityBlockStatusItem `xml:"item"`
}

type ec2Stage114CapacityBlockStatusItem struct {
	CapacityBlockID              *string                                 `xml:"capacityBlockId,omitempty"`
	CapacityReservationStatusSet ec2Stage114CapacityReservationStatusSet `xml:"capacityReservationStatusSet"`
	InterconnectStatus           *string                                 `xml:"interconnectStatus,omitempty"`
	TotalAvailableCapacity       *int32                                  `xml:"totalAvailableCapacity,omitempty"`
	TotalCapacity                *int32                                  `xml:"totalCapacity,omitempty"`
	TotalUnavailableCapacity     *int32                                  `xml:"totalUnavailableCapacity,omitempty"`
}

type ec2Stage114CapacityReservationStatusSet struct {
	Items []ec2Stage114CapacityReservationStatusItem `xml:"item"`
}

type ec2Stage114CapacityReservationStatusItem struct {
	CapacityReservationID    *string `xml:"capacityReservationId,omitempty"`
	TotalAvailableCapacity   *int32  `xml:"totalAvailableCapacity,omitempty"`
	TotalCapacity            *int32  `xml:"totalCapacity,omitempty"`
	TotalUnavailableCapacity *int32  `xml:"totalUnavailableCapacity,omitempty"`
}
