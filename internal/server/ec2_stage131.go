package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage131Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ProvisionIpamByoasn":
		asnAuthorizationContext, ok := parseEC2Stage131AsnAuthorizationContext(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		byoasn, err := s.ec2.ProvisionIpamByoasn(
			strings.TrimSpace(r.Form.Get("Asn")),
			asnAuthorizationContext,
			strings.TrimSpace(r.Form.Get("IpamId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131ProvisionIpamByoasnResponse{
			XMLName:   xml.Name{Local: "ProvisionIpamByoasnResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Byoasn:    ec2Stage114ByoasnItemFrom(byoasn),
		})
		return true
	case "ProvisionIpamPoolCidr":
		netmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("NetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		cidrAuthorizationContext, ok := parseEC2Stage131OptionalIpamCidrAuthorizationContext(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		ipamPoolCidr, err := s.ec2.ProvisionIpamPoolCidr(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			parseEC2OptionalString(r.Form.Get("Cidr")),
			netmaskLength,
			cidrAuthorizationContext,
			parseEC2OptionalString(r.Form.Get("IpamExternalResourceVerificationTokenId")),
			strings.TrimSpace(r.Form.Get("VerificationMethod")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131ProvisionIpamPoolCidrResponse{
			XMLName:      xml.Name{Local: "ProvisionIpamPoolCidrResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			IpamPoolCidr: ec2Stage114IpamPoolCidrItemFrom(ipamPoolCidr),
		})
		return true
	case "ProvisionPublicIpv4PoolCidr":
		netmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("NetmaskLength"))
		if !ok || netmaskLength == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		poolAddressRange, poolID, err := s.ec2.ProvisionPublicIpv4PoolCidr(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			*netmaskLength,
			strings.TrimSpace(r.Form.Get("PoolId")),
			ec2OptionalStringPointerFromForm(r.Form, "NetworkBorderGroup"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131ProvisionPublicIpv4PoolCidrResponse{
			XMLName:          xml.Name{Local: "ProvisionPublicIpv4PoolCidrResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			PoolAddressRange: ec2Stage131PublicIpv4PoolRangeItemFrom(poolAddressRange),
			PoolID:           poolID,
		})
		return true
	case "PurchaseCapacityBlock":
		_, capacityReservation, err := s.ec2.PurchaseCapacityBlock(
			strings.TrimSpace(r.Form.Get("CapacityBlockOfferingId")),
			strings.TrimSpace(r.Form.Get("InstancePlatform")),
			parseEC2TagSpecificationsForResource(r.Form, "capacity-block"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131PurchaseCapacityBlockResponse{
			XMLName:             xml.Name{Local: "PurchaseCapacityBlockResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			CapacityReservation: ec2Stage102CapacityReservationItemFrom(capacityReservation),
		})
		return true
	case "PurchaseCapacityBlockExtension":
		extensions, err := s.ec2.PurchaseCapacityBlockExtension(
			strings.TrimSpace(r.Form.Get("CapacityBlockExtensionOfferingId")),
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131PurchaseCapacityBlockExtensionResponse{
			XMLName:                 xml.Name{Local: "PurchaseCapacityBlockExtensionResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			CapacityBlockExtensions: ec2Stage131CapacityBlockExtensionSet{Items: ec2Stage114CapacityBlockExtensionItemsFrom(extensions)},
		})
		return true
	case "PurchaseHostReservation":
		clientToken, currencyCode, purchase, err := s.ec2.PurchaseHostReservation(
			parseEC2MembersOrItemList(r.Form, "HostIdSet"),
			strings.TrimSpace(r.Form.Get("OfferingId")),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			strings.TrimSpace(r.Form.Get("CurrencyCode")),
			parseEC2OptionalString(r.Form.Get("LimitPrice")),
			parseEC2TagSpecificationsForResource(r.Form, "host-reservation"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		totalHourlyPrice, totalUpfrontPrice := ec2Stage131PurchaseTotals(purchase)
		respondEC2XML(w, ec2Stage131PurchaseHostReservationResponse{
			XMLName:           xml.Name{Local: "PurchaseHostReservationResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			ClientToken:       clientToken,
			CurrencyCode:      currencyCode,
			Purchase:          ec2Stage124PurchaseSet{Items: ec2Stage124PurchaseItemsFrom(purchase)},
			TotalHourlyPrice:  totalHourlyPrice,
			TotalUpfrontPrice: totalUpfrontPrice,
		})
		return true
	case "PurchaseReservedInstancesOffering":
		instanceCount, ok := parseEC2OptionalInt32(r.Form.Get("InstanceCount"))
		if !ok || instanceCount == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		purchaseTime, ok := parseEC2Stage121OptionalRFC3339Time(r.Form, "PurchaseTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		reservedInstancesID, err := s.ec2.PurchaseReservedInstancesOffering(
			*instanceCount,
			strings.TrimSpace(r.Form.Get("ReservedInstancesOfferingId")),
			purchaseTime,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131PurchaseReservedInstancesOfferingResponse{
			XMLName:             xml.Name{Local: "PurchaseReservedInstancesOfferingResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			ReservedInstancesID: reservedInstancesID,
		})
		return true
	case "PurchaseScheduledInstances":
		purchaseRequests, ok := parseEC2Stage131PurchaseRequests(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		scheduledInstances, err := s.ec2.PurchaseScheduledInstances(
			purchaseRequests,
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131PurchaseScheduledInstancesResponse{
			XMLName:            xml.Name{Local: "PurchaseScheduledInstancesResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			ScheduledInstances: ec2Stage121ScheduledInstanceSet{Items: ec2Stage121ScheduledInstanceItemsFrom(scheduledInstances)},
		})
		return true
	case "RegisterImage":
		imageID, err := s.ec2.RegisterImage(
			strings.TrimSpace(r.Form.Get("Name")),
			strings.TrimSpace(r.Form.Get("Architecture")),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			ec2OptionalStringPointerFromForm(r.Form, "ImageLocation"),
			ec2OptionalStringPointerFromForm(r.Form, "RootDeviceName"),
			ec2OptionalStringPointerFromForm(r.Form, "VirtualizationType"),
			parseEC2TagSpecificationsForResource(r.Form, "image"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131RegisterImageResponse{
			XMLName:   xml.Name{Local: "RegisterImageResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			ImageID:   imageID,
		})
		return true
	case "RegisterInstanceEventNotificationAttributes":
		includeAllTagsOfInstance, hasIncludeAllTagsOfInstance, ok := ec2OptionalBoolFromForm(r.Form, "InstanceTagAttribute.IncludeAllTagsOfInstance")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasIncludeAllTagsOfInstance {
			includeAllTagsOfInstance = nil
		}
		instanceTagAttribute, err := s.ec2.RegisterInstanceEventNotificationAttributes(
			includeAllTagsOfInstance,
			parseEC2MembersOrItemList(r.Form, "InstanceTagAttribute.InstanceTagKey"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage131RegisterInstanceEventNotificationAttributesResponse{
			XMLName:              xml.Name{Local: "RegisterInstanceEventNotificationAttributesResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			InstanceTagAttribute: ec2Stage131InstanceTagNotificationAttributeItemFrom(instanceTagAttribute),
		})
		return true
	default:
		return false
	}
}

func parseEC2Stage131AsnAuthorizationContext(values url.Values) (*ec2svc.AsnAuthorizationContext, bool) {
	message := strings.TrimSpace(values.Get("AsnAuthorizationContext.Message"))
	signature := strings.TrimSpace(values.Get("AsnAuthorizationContext.Signature"))
	if message == "" || signature == "" {
		return nil, false
	}
	return &ec2svc.AsnAuthorizationContext{Message: message, Signature: signature}, true
}

func parseEC2Stage131OptionalIpamCidrAuthorizationContext(values url.Values) (*ec2svc.IpamCidrAuthorizationContext, bool) {
	const (
		messageKey   = "CidrAuthorizationContext.Message"
		signatureKey = "CidrAuthorizationContext.Signature"
	)
	hasMessage := hasEC2Field(values, messageKey)
	hasSignature := hasEC2Field(values, signatureKey)
	if !hasMessage && !hasSignature {
		return nil, true
	}
	if !hasMessage || !hasSignature {
		return nil, false
	}

	message := strings.TrimSpace(values.Get(messageKey))
	signature := strings.TrimSpace(values.Get(signatureKey))
	if message == "" || signature == "" {
		return nil, false
	}
	return &ec2svc.IpamCidrAuthorizationContext{Message: message, Signature: signature}, true
}

func parseEC2Stage131PurchaseRequests(values url.Values) ([]ec2svc.PurchaseScheduledInstanceRequest, bool) {
	const prefix = "PurchaseRequest."
	if !hasEC2PrefixedField(values, prefix) {
		return nil, false
	}

	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rest := strings.TrimPrefix(key, prefix)
		if strings.HasPrefix(rest, "Item.") {
			rest = strings.TrimPrefix(rest, "Item.")
		}
		if strings.HasPrefix(rest, "Member.") {
			rest = strings.TrimPrefix(rest, "Member.")
		}
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}

	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)
	if len(ordered) == 0 {
		return nil, false
	}

	out := make([]ec2svc.PurchaseScheduledInstanceRequest, 0, len(ordered))
	for _, idx := range ordered {
		instanceCountRaw := ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "InstanceCount")
		instanceCount, ok := parseEC2OptionalInt32(instanceCountRaw)
		if !ok || instanceCount == nil || *instanceCount <= 0 {
			return nil, false
		}
		purchaseToken := ec2ReservationFleetIndexedField(values, []string{prefix}, idx, "PurchaseToken")
		if strings.TrimSpace(purchaseToken) == "" {
			return nil, false
		}
		out = append(out, ec2svc.PurchaseScheduledInstanceRequest{
			InstanceCount: *instanceCount,
			PurchaseToken: strings.TrimSpace(purchaseToken),
		})
	}
	return out, true
}

func ec2Stage131PublicIpv4PoolRangeItemFrom(in ec2svc.PublicIpv4PoolRange) ec2Stage120PublicIpv4PoolRangeItem {
	return ec2Stage120PublicIpv4PoolRangeItem{
		AddressCount:          in.AddressCount,
		AvailableAddressCount: in.AvailableAddressCount,
		FirstAddress:          in.FirstAddress,
		LastAddress:           in.LastAddress,
	}
}

func ec2Stage131PurchaseTotals(purchase []ec2svc.HostReservationPurchase) (*string, *string) {
	if len(purchase) == 0 {
		return nil, nil
	}
	var totalHourly float64
	var totalUpfront float64
	for _, item := range purchase {
		if v, err := strconv.ParseFloat(strings.TrimSpace(item.HourlyPrice), 64); err == nil {
			totalHourly += v
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(item.UpfrontPrice), 64); err == nil {
			totalUpfront += v
		}
	}
	hourly := strconv.FormatFloat(totalHourly, 'f', 2, 64)
	upfront := strconv.FormatFloat(totalUpfront, 'f', 2, 64)
	return &hourly, &upfront
}

func ec2Stage131InstanceTagNotificationAttributeItemFrom(in ec2svc.InstanceTagNotificationAttribute) ec2Stage117InstanceTagNotificationAttributeItem {
	return ec2Stage117InstanceTagNotificationAttributeItem{
		IncludeAllTagsOfInstance: in.IncludeAllTagsOfInstance,
		InstanceTagKeySet:        ec2StringSet{Items: append([]string(nil), in.InstanceTagKeys...)},
	}
}

type ec2Stage131ProvisionIpamByoasnResponse struct {
	XMLName   xml.Name              `xml:"ProvisionIpamByoasnResponse"`
	Xmlns     string                `xml:"xmlns,attr"`
	RequestID string                `xml:"requestId"`
	Byoasn    ec2Stage114ByoasnItem `xml:"byoasn"`
}

type ec2Stage131ProvisionIpamPoolCidrResponse struct {
	XMLName      xml.Name                    `xml:"ProvisionIpamPoolCidrResponse"`
	Xmlns        string                      `xml:"xmlns,attr"`
	RequestID    string                      `xml:"requestId"`
	IpamPoolCidr ec2Stage114IpamPoolCidrItem `xml:"ipamPoolCidr"`
}

type ec2Stage131ProvisionPublicIpv4PoolCidrResponse struct {
	XMLName          xml.Name                           `xml:"ProvisionPublicIpv4PoolCidrResponse"`
	Xmlns            string                             `xml:"xmlns,attr"`
	RequestID        string                             `xml:"requestId"`
	PoolAddressRange ec2Stage120PublicIpv4PoolRangeItem `xml:"poolAddressRange"`
	PoolID           string                             `xml:"poolId,omitempty"`
}

type ec2Stage131PurchaseCapacityBlockResponse struct {
	XMLName             xml.Name                           `xml:"PurchaseCapacityBlockResponse"`
	Xmlns               string                             `xml:"xmlns,attr"`
	RequestID           string                             `xml:"requestId"`
	CapacityReservation ec2Stage102CapacityReservationItem `xml:"capacityReservation"`
}

type ec2Stage131CapacityBlockSet struct {
	Items []ec2Stage115CapacityBlockItem `xml:"item"`
}

type ec2Stage131PurchaseCapacityBlockExtensionResponse struct {
	XMLName                 xml.Name                             `xml:"PurchaseCapacityBlockExtensionResponse"`
	Xmlns                   string                               `xml:"xmlns,attr"`
	RequestID               string                               `xml:"requestId"`
	CapacityBlockExtensions ec2Stage131CapacityBlockExtensionSet `xml:"capacityBlockExtensionSet"`
}

type ec2Stage131CapacityBlockExtensionSet struct {
	Items []ec2Stage114CapacityBlockExtensionItem `xml:"item"`
}

type ec2Stage131PurchaseHostReservationResponse struct {
	XMLName           xml.Name               `xml:"PurchaseHostReservationResponse"`
	Xmlns             string                 `xml:"xmlns,attr"`
	RequestID         string                 `xml:"requestId"`
	ClientToken       *string                `xml:"clientToken,omitempty"`
	CurrencyCode      string                 `xml:"currencyCode,omitempty"`
	Purchase          ec2Stage124PurchaseSet `xml:"purchase"`
	TotalHourlyPrice  *string                `xml:"totalHourlyPrice,omitempty"`
	TotalUpfrontPrice *string                `xml:"totalUpfrontPrice,omitempty"`
}

type ec2Stage131PurchaseReservedInstancesOfferingResponse struct {
	XMLName             xml.Name `xml:"PurchaseReservedInstancesOfferingResponse"`
	Xmlns               string   `xml:"xmlns,attr"`
	RequestID           string   `xml:"requestId"`
	ReservedInstancesID string   `xml:"reservedInstancesId,omitempty"`
}

type ec2Stage131PurchaseScheduledInstancesResponse struct {
	XMLName            xml.Name                        `xml:"PurchaseScheduledInstancesResponse"`
	Xmlns              string                          `xml:"xmlns,attr"`
	RequestID          string                          `xml:"requestId"`
	ScheduledInstances ec2Stage121ScheduledInstanceSet `xml:"scheduledInstanceSet"`
}

type ec2Stage131RegisterImageResponse struct {
	XMLName   xml.Name `xml:"RegisterImageResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	ImageID   string   `xml:"imageId,omitempty"`
}

type ec2Stage131RegisterInstanceEventNotificationAttributesResponse struct {
	XMLName              xml.Name                                        `xml:"RegisterInstanceEventNotificationAttributesResponse"`
	Xmlns                string                                          `xml:"xmlns,attr"`
	RequestID            string                                          `xml:"requestId"`
	InstanceTagAttribute ec2Stage117InstanceTagNotificationAttributeItem `xml:"instanceTagAttribute"`
}
