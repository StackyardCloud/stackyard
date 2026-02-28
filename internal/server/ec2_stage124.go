package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage124Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "GetAssociatedIpv6PoolCidrs":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.GetAssociatedIpv6PoolCidrs(
			strings.TrimSpace(r.Form.Get("PoolId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetAssociatedIpv6PoolCidrsResponse{
			XMLName:                xml.Name{Local: "GetAssociatedIpv6PoolCidrsResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			Ipv6CidrAssociationSet: ec2Stage124Ipv6CidrAssociationSet{Items: ec2Stage124Ipv6CidrAssociationItemsFrom(associations)},
			NextToken:              nextToken,
		})
		return true
	case "GetCapacityReservationUsage":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		usage, nextToken, err := s.ec2.GetCapacityReservationUsage(
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetCapacityReservationUsageResponse{
			XMLName:                xml.Name{Local: "GetCapacityReservationUsageResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			AvailableInstanceCount: usage.AvailableInstanceCount,
			CapacityReservationID:  usage.CapacityReservationID,
			InstanceType:           usage.InstanceType,
			InstanceUsageSet:       ec2Stage124InstanceUsageSet{Items: ec2Stage124InstanceUsageItemsFrom(usage.InstanceUsages)},
			State:                  usage.State,
			TotalInstanceCount:     usage.TotalInstanceCount,
			NextToken:              nextToken,
		})
		return true
	case "GetCoipPoolUsage":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		usage, nextToken, err := s.ec2.GetCoipPoolUsage(
			strings.TrimSpace(r.Form.Get("PoolId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetCoipPoolUsageResponse{
			XMLName:                  xml.Name{Local: "GetCoipPoolUsageResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			CoipAddressUsageSet:      ec2Stage124CoipAddressUsageSet{Items: ec2Stage124CoipAddressUsageItemsFrom(usage.CoipAddressUsages)},
			CoipPoolID:               usage.CoipPoolID,
			LocalGatewayRouteTableID: usage.LocalGatewayRouteTableID,
			NextToken:                nextToken,
		})
		return true
	case "GetDefaultCreditSpecification":
		specification, err := s.ec2.GetDefaultCreditSpecification(strings.TrimSpace(r.Form.Get("InstanceFamily")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetDefaultCreditSpecificationResponse{
			XMLName:   xml.Name{Local: "GetDefaultCreditSpecificationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			InstanceFamilyCreditSpecification: ec2Stage124InstanceFamilyCreditSpecificationItem{
				CpuCredits:     specification.CpuCredits,
				InstanceFamily: specification.InstanceFamily,
			},
		})
		return true
	case "GetFlowLogsIntegrationTemplate":
		hasIntegrateService := hasEC2Field(r.Form, "IntegrateService") || hasEC2PrefixedField(r.Form, "IntegrateService.")
		result, err := s.ec2.GetFlowLogsIntegrationTemplate(
			strings.TrimSpace(r.Form.Get("ConfigDeliveryS3DestinationArn")),
			strings.TrimSpace(r.Form.Get("FlowLogId")),
			hasIntegrateService,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetFlowLogsIntegrationTemplateResponse{
			XMLName:   xml.Name{Local: "GetFlowLogsIntegrationTemplateResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Result:    result,
		})
		return true
	case "GetGroupsForCapacityReservation":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		groups, nextToken, err := s.ec2.GetGroupsForCapacityReservation(
			strings.TrimSpace(r.Form.Get("CapacityReservationId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetGroupsForCapacityReservationResponse{
			XMLName:                     xml.Name{Local: "GetGroupsForCapacityReservationResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			CapacityReservationGroupSet: ec2Stage124CapacityReservationGroupSet{Items: ec2Stage124CapacityReservationGroupItemsFrom(groups)},
			NextToken:                   nextToken,
		})
		return true
	case "GetHostReservationPurchasePreview":
		preview, err := s.ec2.GetHostReservationPurchasePreview(
			parseEC2MembersOrItemList(r.Form, "HostIdSet"),
			strings.TrimSpace(r.Form.Get("OfferingId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetHostReservationPurchasePreviewResponse{
			XMLName:           xml.Name{Local: "GetHostReservationPurchasePreviewResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			CurrencyCode:      preview.CurrencyCode,
			Purchase:          ec2Stage124PurchaseSet{Items: ec2Stage124PurchaseItemsFrom(preview.Purchase)},
			TotalHourlyPrice:  preview.TotalHourlyPrice,
			TotalUpfrontPrice: preview.TotalUpfrontPrice,
		})
		return true
	case "GetInstanceMetadataDefaults":
		defaults := s.ec2.GetInstanceMetadataDefaults()
		respondEC2XML(w, ec2Stage124GetInstanceMetadataDefaultsResponse{
			XMLName:   xml.Name{Local: "GetInstanceMetadataDefaultsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			AccountLevel: ec2Stage124InstanceMetadataDefaultsResponseItem{
				HttpEndpoint:            defaults.HttpEndpoint,
				HttpPutResponseHopLimit: defaults.HttpPutResponseHopLimit,
				HttpTokens:              defaults.HttpTokens,
				InstanceMetadataTags:    defaults.InstanceMetadataTags,
				ManagedBy:               defaults.ManagedBy,
				ManagedExceptionMessage: defaults.ManagedExceptionMessage,
			},
		})
		return true
	case "GetInstanceTpmEkPub":
		pub, err := s.ec2.GetInstanceTpmEkPub(
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("KeyFormat")),
			strings.TrimSpace(r.Form.Get("KeyType")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetInstanceTpmEkPubResponse{
			XMLName:    xml.Name{Local: "GetInstanceTpmEkPubResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			InstanceID: pub.InstanceID,
			KeyFormat:  pub.KeyFormat,
			KeyType:    pub.KeyType,
			KeyValue:   pub.KeyValue,
		})
		return true
	case "GetInstanceTypesFromInstanceRequirements":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		instanceTypes, nextToken, err := s.ec2.GetInstanceTypesFromInstanceRequirements(
			parseEC2MembersOrItemList(r.Form, "ArchitectureType"),
			parseEC2MembersOrItemList(r.Form, "VirtualizationType"),
			hasEC2PrefixedField(r.Form, "InstanceRequirements."),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage124GetInstanceTypesFromInstanceRequirementsResponse{
			XMLName:         xml.Name{Local: "GetInstanceTypesFromInstanceRequirementsResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			InstanceTypeSet: ec2Stage124InstanceTypeFromRequirementsSet{Items: ec2Stage124InstanceTypeFromRequirementsItemsFrom(instanceTypes)},
			NextToken:       nextToken,
		})
		return true
	default:
		return false
	}
}

func ec2Stage124Ipv6CidrAssociationItemsFrom(in []ec2svc.AssociatedIpv6PoolCidr) []ec2Stage124Ipv6CidrAssociationItem {
	out := make([]ec2Stage124Ipv6CidrAssociationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage124Ipv6CidrAssociationItem{
			AssociatedResource: item.AssociatedResource,
			Ipv6Cidr:           item.Ipv6Cidr,
		})
	}
	return out
}

func ec2Stage124InstanceUsageItemsFrom(in []ec2svc.CapacityReservationInstanceUsage) []ec2Stage124InstanceUsageItem {
	out := make([]ec2Stage124InstanceUsageItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage124InstanceUsageItem{
			AccountID:         item.AccountID,
			UsedInstanceCount: item.UsedInstanceCount,
		})
	}
	return out
}

func ec2Stage124CoipAddressUsageItemsFrom(in []ec2svc.CoipAddressUsage) []ec2Stage124CoipAddressUsageItem {
	out := make([]ec2Stage124CoipAddressUsageItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage124CoipAddressUsageItem{
			AllocationID: item.AllocationID,
			AwsAccountID: item.AwsAccountID,
			AwsService:   item.AwsService,
			CoIP:         item.CoIP,
		})
	}
	return out
}

func ec2Stage124CapacityReservationGroupItemsFrom(in []ec2svc.CapacityReservationGroupAssociation) []ec2Stage124CapacityReservationGroupItem {
	out := make([]ec2Stage124CapacityReservationGroupItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage124CapacityReservationGroupItem{
			GroupARN: item.GroupARN,
			OwnerID:  item.OwnerID,
		})
	}
	return out
}

func ec2Stage124PurchaseItemsFrom(in []ec2svc.HostReservationPurchase) []ec2Stage124PurchaseItem {
	out := make([]ec2Stage124PurchaseItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage124PurchaseItem{
			CurrencyCode:      item.CurrencyCode,
			Duration:          item.Duration,
			HostIDSet:         ec2StringSet{Items: append([]string(nil), item.HostIDs...)},
			HostReservationID: item.HostReservationID,
			HourlyPrice:       item.HourlyPrice,
			InstanceFamily:    item.InstanceFamily,
			PaymentOption:     item.PaymentOption,
			UpfrontPrice:      item.UpfrontPrice,
		})
	}
	return out
}

func ec2Stage124InstanceTypeFromRequirementsItemsFrom(in []ec2svc.InstanceTypeFromRequirements) []ec2Stage124InstanceTypeFromRequirementsItem {
	out := make([]ec2Stage124InstanceTypeFromRequirementsItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage124InstanceTypeFromRequirementsItem{InstanceType: item.InstanceType})
	}
	return out
}

type ec2Stage124GetAssociatedIpv6PoolCidrsResponse struct {
	XMLName                xml.Name                          `xml:"GetAssociatedIpv6PoolCidrsResponse"`
	Xmlns                  string                            `xml:"xmlns,attr"`
	RequestID              string                            `xml:"requestId"`
	Ipv6CidrAssociationSet ec2Stage124Ipv6CidrAssociationSet `xml:"ipv6CidrAssociationSet"`
	NextToken              *string                           `xml:"nextToken,omitempty"`
}

type ec2Stage124Ipv6CidrAssociationSet struct {
	Items []ec2Stage124Ipv6CidrAssociationItem `xml:"item"`
}

type ec2Stage124Ipv6CidrAssociationItem struct {
	AssociatedResource string `xml:"associatedResource,omitempty"`
	Ipv6Cidr           string `xml:"ipv6Cidr,omitempty"`
}

type ec2Stage124GetCapacityReservationUsageResponse struct {
	XMLName                xml.Name                    `xml:"GetCapacityReservationUsageResponse"`
	Xmlns                  string                      `xml:"xmlns,attr"`
	RequestID              string                      `xml:"requestId"`
	AvailableInstanceCount int32                       `xml:"availableInstanceCount,omitempty"`
	CapacityReservationID  string                      `xml:"capacityReservationId,omitempty"`
	InstanceType           string                      `xml:"instanceType,omitempty"`
	InstanceUsageSet       ec2Stage124InstanceUsageSet `xml:"instanceUsageSet"`
	State                  string                      `xml:"state,omitempty"`
	TotalInstanceCount     int32                       `xml:"totalInstanceCount,omitempty"`
	NextToken              *string                     `xml:"nextToken,omitempty"`
}

type ec2Stage124InstanceUsageSet struct {
	Items []ec2Stage124InstanceUsageItem `xml:"item"`
}

type ec2Stage124InstanceUsageItem struct {
	AccountID         string `xml:"accountId,omitempty"`
	UsedInstanceCount int32  `xml:"usedInstanceCount,omitempty"`
}

type ec2Stage124GetCoipPoolUsageResponse struct {
	XMLName                  xml.Name                       `xml:"GetCoipPoolUsageResponse"`
	Xmlns                    string                         `xml:"xmlns,attr"`
	RequestID                string                         `xml:"requestId"`
	CoipAddressUsageSet      ec2Stage124CoipAddressUsageSet `xml:"coipAddressUsageSet"`
	CoipPoolID               string                         `xml:"coipPoolId,omitempty"`
	LocalGatewayRouteTableID string                         `xml:"localGatewayRouteTableId,omitempty"`
	NextToken                *string                        `xml:"nextToken,omitempty"`
}

type ec2Stage124CoipAddressUsageSet struct {
	Items []ec2Stage124CoipAddressUsageItem `xml:"item"`
}

type ec2Stage124CoipAddressUsageItem struct {
	AllocationID string `xml:"allocationId,omitempty"`
	AwsAccountID string `xml:"awsAccountId,omitempty"`
	AwsService   string `xml:"awsService,omitempty"`
	CoIP         string `xml:"coIp,omitempty"`
}

type ec2Stage124GetDefaultCreditSpecificationResponse struct {
	XMLName                           xml.Name                                         `xml:"GetDefaultCreditSpecificationResponse"`
	Xmlns                             string                                           `xml:"xmlns,attr"`
	RequestID                         string                                           `xml:"requestId"`
	InstanceFamilyCreditSpecification ec2Stage124InstanceFamilyCreditSpecificationItem `xml:"instanceFamilyCreditSpecification"`
}

type ec2Stage124InstanceFamilyCreditSpecificationItem struct {
	CpuCredits     string `xml:"cpuCredits,omitempty"`
	InstanceFamily string `xml:"instanceFamily,omitempty"`
}

type ec2Stage124GetFlowLogsIntegrationTemplateResponse struct {
	XMLName   xml.Name `xml:"GetFlowLogsIntegrationTemplateResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
	Result    string   `xml:"result,omitempty"`
}

type ec2Stage124GetGroupsForCapacityReservationResponse struct {
	XMLName                     xml.Name                               `xml:"GetGroupsForCapacityReservationResponse"`
	Xmlns                       string                                 `xml:"xmlns,attr"`
	RequestID                   string                                 `xml:"requestId"`
	CapacityReservationGroupSet ec2Stage124CapacityReservationGroupSet `xml:"capacityReservationGroupSet"`
	NextToken                   *string                                `xml:"nextToken,omitempty"`
}

type ec2Stage124CapacityReservationGroupSet struct {
	Items []ec2Stage124CapacityReservationGroupItem `xml:"item"`
}

type ec2Stage124CapacityReservationGroupItem struct {
	GroupARN string `xml:"groupArn,omitempty"`
	OwnerID  string `xml:"ownerId,omitempty"`
}

type ec2Stage124GetHostReservationPurchasePreviewResponse struct {
	XMLName           xml.Name               `xml:"GetHostReservationPurchasePreviewResponse"`
	Xmlns             string                 `xml:"xmlns,attr"`
	RequestID         string                 `xml:"requestId"`
	CurrencyCode      string                 `xml:"currencyCode,omitempty"`
	Purchase          ec2Stage124PurchaseSet `xml:"purchase"`
	TotalHourlyPrice  string                 `xml:"totalHourlyPrice,omitempty"`
	TotalUpfrontPrice string                 `xml:"totalUpfrontPrice,omitempty"`
}

type ec2Stage124PurchaseSet struct {
	Items []ec2Stage124PurchaseItem `xml:"item"`
}

type ec2Stage124PurchaseItem struct {
	CurrencyCode      string       `xml:"currencyCode,omitempty"`
	Duration          int32        `xml:"duration,omitempty"`
	HostIDSet         ec2StringSet `xml:"hostIdSet"`
	HostReservationID string       `xml:"hostReservationId,omitempty"`
	HourlyPrice       string       `xml:"hourlyPrice,omitempty"`
	InstanceFamily    string       `xml:"instanceFamily,omitempty"`
	PaymentOption     string       `xml:"paymentOption,omitempty"`
	UpfrontPrice      string       `xml:"upfrontPrice,omitempty"`
}

type ec2Stage124GetInstanceMetadataDefaultsResponse struct {
	XMLName      xml.Name                                        `xml:"GetInstanceMetadataDefaultsResponse"`
	Xmlns        string                                          `xml:"xmlns,attr"`
	RequestID    string                                          `xml:"requestId"`
	AccountLevel ec2Stage124InstanceMetadataDefaultsResponseItem `xml:"accountLevel"`
}

type ec2Stage124InstanceMetadataDefaultsResponseItem struct {
	HttpEndpoint            string  `xml:"httpEndpoint,omitempty"`
	HttpPutResponseHopLimit *int32  `xml:"httpPutResponseHopLimit,omitempty"`
	HttpTokens              string  `xml:"httpTokens,omitempty"`
	InstanceMetadataTags    string  `xml:"instanceMetadataTags,omitempty"`
	ManagedBy               string  `xml:"managedBy,omitempty"`
	ManagedExceptionMessage *string `xml:"managedExceptionMessage,omitempty"`
}

type ec2Stage124GetInstanceTpmEkPubResponse struct {
	XMLName    xml.Name `xml:"GetInstanceTpmEkPubResponse"`
	Xmlns      string   `xml:"xmlns,attr"`
	RequestID  string   `xml:"requestId"`
	InstanceID string   `xml:"instanceId,omitempty"`
	KeyFormat  string   `xml:"keyFormat,omitempty"`
	KeyType    string   `xml:"keyType,omitempty"`
	KeyValue   string   `xml:"keyValue,omitempty"`
}

type ec2Stage124GetInstanceTypesFromInstanceRequirementsResponse struct {
	XMLName         xml.Name                                   `xml:"GetInstanceTypesFromInstanceRequirementsResponse"`
	Xmlns           string                                     `xml:"xmlns,attr"`
	RequestID       string                                     `xml:"requestId"`
	InstanceTypeSet ec2Stage124InstanceTypeFromRequirementsSet `xml:"instanceTypeSet"`
	NextToken       *string                                    `xml:"nextToken,omitempty"`
}

type ec2Stage124InstanceTypeFromRequirementsSet struct {
	Items []ec2Stage124InstanceTypeFromRequirementsItem `xml:"item"`
}

type ec2Stage124InstanceTypeFromRequirementsItem struct {
	InstanceType string `xml:"instanceType,omitempty"`
}
