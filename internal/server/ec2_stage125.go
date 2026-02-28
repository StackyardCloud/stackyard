package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage125Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "GetInstanceUefiData":
		uefiData, err := s.ec2.GetInstanceUefiData(strings.TrimSpace(r.Form.Get("InstanceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetInstanceUefiDataResponse{
			XMLName:    xml.Name{Local: "GetInstanceUefiDataResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			InstanceID: uefiData.InstanceID,
			UefiData:   uefiData.UefiData,
		})
		return true
	case "GetIpamAddressHistory":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		startTime, ok := parseEC2Stage120OptionalRFC3339Time(r.Form, "StartTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endTime, ok := parseEC2Stage120OptionalRFC3339Time(r.Form, "EndTime")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		historyRecords, nextToken, err := s.ec2.GetIpamAddressHistory(
			strings.TrimSpace(r.Form.Get("Cidr")),
			strings.TrimSpace(r.Form.Get("IpamScopeId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			startTime,
			endTime,
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamAddressHistoryResponse{
			XMLName:          xml.Name{Local: "GetIpamAddressHistoryResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			HistoryRecordSet: ec2Stage125IpamAddressHistoryRecordSet{Items: ec2Stage125IpamAddressHistoryRecordItemsFrom(historyRecords)},
			NextToken:        nextToken,
		})
		return true
	case "GetIpamDiscoveredAccounts":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		accounts, nextToken, err := s.ec2.GetIpamDiscoveredAccounts(
			strings.TrimSpace(r.Form.Get("DiscoveryRegion")),
			strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamDiscoveredAccountsResponse{
			XMLName:                  xml.Name{Local: "GetIpamDiscoveredAccountsResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			IpamDiscoveredAccountSet: ec2Stage125IpamDiscoveredAccountSet{Items: ec2Stage125IpamDiscoveredAccountItemsFrom(accounts)},
			NextToken:                nextToken,
		})
		return true
	case "GetIpamDiscoveredPublicAddresses":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		addresses, oldestSampleTime, nextToken, err := s.ec2.GetIpamDiscoveredPublicAddresses(
			strings.TrimSpace(r.Form.Get("AddressRegion")),
			strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamDiscoveredPublicAddressesResponse{
			XMLName:                        xml.Name{Local: "GetIpamDiscoveredPublicAddressesResponse"},
			Xmlns:                          ec2Namespace,
			RequestID:                      "stackyard-request",
			IpamDiscoveredPublicAddressSet: ec2Stage125IpamDiscoveredPublicAddressSet{Items: ec2Stage125IpamDiscoveredPublicAddressItemsFrom(addresses)},
			OldestSampleTime:               ec2TimeString(oldestSampleTime),
			NextToken:                      nextToken,
		})
		return true
	case "GetIpamDiscoveredResourceCidrs":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		resourceCidrs, nextToken, err := s.ec2.GetIpamDiscoveredResourceCidrs(
			strings.TrimSpace(r.Form.Get("IpamResourceDiscoveryId")),
			strings.TrimSpace(r.Form.Get("ResourceRegion")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamDiscoveredResourceCidrsResponse{
			XMLName:                       xml.Name{Local: "GetIpamDiscoveredResourceCidrsResponse"},
			Xmlns:                         ec2Namespace,
			RequestID:                     "stackyard-request",
			IpamDiscoveredResourceCidrSet: ec2Stage125IpamDiscoveredResourceCidrSet{Items: ec2Stage125IpamDiscoveredResourceCidrItemsFrom(resourceCidrs)},
			NextToken:                     nextToken,
		})
		return true
	case "GetIpamPoolAllocations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		allocations, nextToken, err := s.ec2.GetIpamPoolAllocations(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			strings.TrimSpace(r.Form.Get("IpamPoolAllocationId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamPoolAllocationsResponse{
			XMLName:               xml.Name{Local: "GetIpamPoolAllocationsResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			IpamPoolAllocationSet: ec2Stage125IpamPoolAllocationSet{Items: ec2Stage125IpamPoolAllocationItemsFrom(allocations)},
			NextToken:             nextToken,
		})
		return true
	case "GetIpamPoolCidrs":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		poolCidrs, nextToken, err := s.ec2.GetIpamPoolCidrs(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamPoolCidrsResponse{
			XMLName:         xml.Name{Local: "GetIpamPoolCidrsResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			IpamPoolCidrSet: ec2Stage125IpamPoolCidrSet{Items: ec2Stage125IpamPoolCidrItemsFrom(poolCidrs)},
			NextToken:       nextToken,
		})
		return true
	case "GetIpamResourceCidrs":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		resourceCidrs, nextToken, err := s.ec2.GetIpamResourceCidrs(
			strings.TrimSpace(r.Form.Get("IpamPoolId")),
			strings.TrimSpace(r.Form.Get("IpamScopeId")),
			strings.TrimSpace(r.Form.Get("ResourceId")),
			strings.TrimSpace(r.Form.Get("ResourceOwner")),
			ec2Stage125RequestIpamResourceTagFromForm(r.Form),
			strings.TrimSpace(r.Form.Get("ResourceType")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetIpamResourceCidrsResponse{
			XMLName:             xml.Name{Local: "GetIpamResourceCidrsResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			IpamResourceCidrSet: ec2Stage125IpamResourceCidrSet{Items: ec2Stage125IpamResourceCidrItemsFrom(resourceCidrs)},
			NextToken:           nextToken,
		})
		return true
	case "GetLaunchTemplateData":
		launchTemplateData, err := s.ec2.GetLaunchTemplateData(strings.TrimSpace(r.Form.Get("InstanceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetLaunchTemplateDataResponse{
			XMLName:            xml.Name{Local: "GetLaunchTemplateDataResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			LaunchTemplateData: ec2Stage125LaunchTemplateDataItemFrom(launchTemplateData),
		})
		return true
	case "GetManagedPrefixListAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.GetManagedPrefixListAssociations(
			strings.TrimSpace(r.Form.Get("PrefixListId")),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage125GetManagedPrefixListAssociationsResponse{
			XMLName:                  xml.Name{Local: "GetManagedPrefixListAssociationsResponse"},
			Xmlns:                    ec2Namespace,
			RequestID:                "stackyard-request",
			PrefixListAssociationSet: ec2Stage125PrefixListAssociationSet{Items: ec2Stage125PrefixListAssociationItemsFrom(associations)},
			NextToken:                nextToken,
		})
		return true
	default:
		return false
	}
}

func ec2Stage125RequestIpamResourceTagFromForm(form url.Values) *ec2svc.RequestIpamResourceTag {
	if !hasEC2Field(form, "ResourceTag") && !hasEC2PrefixedField(form, "ResourceTag.") {
		return nil
	}
	key := strings.TrimSpace(form.Get("ResourceTag.Key"))
	value := strings.TrimSpace(form.Get("ResourceTag.Value"))
	if key == "" && value == "" {
		return nil
	}
	return &ec2svc.RequestIpamResourceTag{
		Key:   key,
		Value: value,
	}
}

func ec2Stage125IpamAddressHistoryRecordItemsFrom(in []ec2svc.IpamAddressHistoryRecord) []ec2Stage125IpamAddressHistoryRecordItem {
	out := make([]ec2Stage125IpamAddressHistoryRecordItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage125IpamAddressHistoryRecordItem{
			ResourceCidr:             item.ResourceCidr,
			ResourceComplianceStatus: item.ResourceComplianceStatus,
			ResourceID:               item.ResourceID,
			ResourceName:             item.ResourceName,
			ResourceOverlapStatus:    item.ResourceOverlapStatus,
			ResourceOwnerID:          item.ResourceOwnerID,
			ResourceRegion:           item.ResourceRegion,
			ResourceType:             item.ResourceType,
			SampledEndTime:           ec2TimeString(item.SampledEndTime),
			SampledStartTime:         ec2TimeString(item.SampledStartTime),
			VpcID:                    item.VpcID,
		})
	}
	return out
}

func ec2Stage125IpamDiscoveredAccountItemsFrom(in []ec2svc.IpamDiscoveredAccount) []ec2Stage125IpamDiscoveredAccountItem {
	out := make([]ec2Stage125IpamDiscoveredAccountItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage125IpamDiscoveredAccountItem{
			AccountID:                   item.AccountID,
			DiscoveryRegion:             item.DiscoveryRegion,
			LastAttemptedDiscoveryTime:  ec2TimeString(item.LastAttemptedDiscoveryTime),
			LastSuccessfulDiscoveryTime: ec2TimeString(item.LastSuccessfulDiscoveryTime),
			OrganizationalUnitID:        item.OrganizationalUnitID,
		}
		if item.FailureReason != nil {
			entry.FailureReason = &ec2Stage125IpamDiscoveryFailureReasonItem{
				Code:    item.FailureReason.Code,
				Message: item.FailureReason.Message,
			}
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage125IpamDiscoveredPublicAddressItemsFrom(in []ec2svc.IpamDiscoveredPublicAddress) []ec2Stage125IpamDiscoveredPublicAddressItem {
	out := make([]ec2Stage125IpamDiscoveredPublicAddressItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage125IpamDiscoveredPublicAddressItem{
			Address:                     item.Address,
			AddressAllocationID:         item.AddressAllocationID,
			AddressOwnerID:              item.AddressOwnerID,
			AddressRegion:               item.AddressRegion,
			AddressType:                 item.AddressType,
			AssociationStatus:           item.AssociationStatus,
			InstanceID:                  item.InstanceID,
			IpamResourceDiscoveryID:     item.IpamResourceDiscoveryID,
			NetworkBorderGroup:          item.NetworkBorderGroup,
			NetworkInterfaceDescription: item.NetworkInterfaceDescription,
			NetworkInterfaceID:          item.NetworkInterfaceID,
			PublicIpv4PoolID:            item.PublicIpv4PoolID,
			SampleTime:                  ec2TimeString(item.SampleTime),
			Service:                     item.Service,
			ServiceResource:             item.ServiceResource,
			SubnetID:                    item.SubnetID,
			VpcID:                       item.VpcID,
		}
		if len(item.SecurityGroups) > 0 {
			entry.SecurityGroupSet = &ec2Stage125IpamPublicAddressSecurityGroupSet{Items: ec2Stage125IpamPublicAddressSecurityGroupItemsFrom(item.SecurityGroups)}
		}
		if len(item.Tags) > 0 {
			entry.Tags = &ec2Stage125IpamPublicAddressTags{
				EipTagSet: ec2Stage125IpamPublicAddressTagSet{Items: ec2Stage125IpamPublicAddressTagItemsFrom(item.Tags)},
			}
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage125IpamPublicAddressSecurityGroupItemsFrom(in []ec2svc.IpamPublicAddressSecurityGroup) []ec2Stage125IpamPublicAddressSecurityGroupItem {
	out := make([]ec2Stage125IpamPublicAddressSecurityGroupItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage125IpamPublicAddressSecurityGroupItem{
			GroupID:   item.GroupID,
			GroupName: item.GroupName,
		})
	}
	return out
}

func ec2Stage125IpamPublicAddressTagItemsFrom(in []ec2svc.IpamPublicAddressTag) []ec2Stage125IpamPublicAddressTagItem {
	out := make([]ec2Stage125IpamPublicAddressTagItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage125IpamPublicAddressTagItem{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	return out
}

func ec2Stage125IpamDiscoveredResourceCidrItemsFrom(in []ec2svc.IpamDiscoveredResourceCidr) []ec2Stage125IpamDiscoveredResourceCidrItem {
	out := make([]ec2Stage125IpamDiscoveredResourceCidrItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage125IpamDiscoveredResourceCidrItem{
			AvailabilityZoneID:               item.AvailabilityZoneID,
			IpamResourceDiscoveryID:          item.IpamResourceDiscoveryID,
			IpSource:                         item.IpSource,
			IpUsage:                          item.IpUsage,
			NetworkInterfaceAttachmentStatus: item.NetworkInterfaceAttachmentStatus,
			ResourceCidr:                     item.ResourceCidr,
			ResourceID:                       item.ResourceID,
			ResourceOwnerID:                  item.ResourceOwnerID,
			ResourceRegion:                   item.ResourceRegion,
			ResourceType:                     item.ResourceType,
			SampleTime:                       ec2TimeString(item.SampleTime),
			SubnetID:                         item.SubnetID,
			VpcID:                            item.VpcID,
		}
		if len(item.ResourceTags) > 0 {
			entry.ResourceTagSet = &ec2Stage125IpamResourceTagSet{Items: ec2Stage125IpamResourceTagItemsFrom(item.ResourceTags)}
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage125IpamPoolAllocationItemsFrom(in []ec2svc.IpamPoolAllocation) []ec2IpamPoolAllocationItem {
	out := make([]ec2IpamPoolAllocationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2IpamPoolAllocationItem{
			Cidr:                 item.Cidr,
			Description:          item.Description,
			IpamPoolAllocationID: item.IpamPoolAllocationID,
			ResourceID:           item.ResourceID,
			ResourceOwner:        item.ResourceOwner,
			ResourceRegion:       item.ResourceRegion,
			ResourceType:         item.ResourceType,
		})
	}
	return out
}

func ec2Stage125IpamPoolCidrItemsFrom(in []ec2svc.IpamPoolCidr) []ec2Stage114IpamPoolCidrItem {
	out := make([]ec2Stage114IpamPoolCidrItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage114IpamPoolCidrItemFrom(item))
	}
	return out
}

func ec2Stage125IpamResourceCidrItemsFrom(in []ec2svc.IpamResourceCidr) []ec2Stage125IpamResourceCidrItem {
	out := make([]ec2Stage125IpamResourceCidrItem, 0, len(in))
	for _, item := range in {
		entry := ec2Stage125IpamResourceCidrItem{
			AvailabilityZoneID: item.AvailabilityZoneID,
			ComplianceStatus:   item.ComplianceStatus,
			IpamID:             item.IpamID,
			IpamPoolID:         item.IpamPoolID,
			IpamScopeID:        item.IpamScopeID,
			IpUsage:            item.IpUsage,
			ManagementState:    item.ManagementState,
			OverlapStatus:      item.OverlapStatus,
			ResourceCidr:       item.ResourceCidr,
			ResourceID:         item.ResourceID,
			ResourceName:       item.ResourceName,
			ResourceOwnerID:    item.ResourceOwnerID,
			ResourceRegion:     item.ResourceRegion,
			ResourceType:       item.ResourceType,
			VpcID:              item.VpcID,
		}
		if len(item.ResourceTags) > 0 {
			entry.ResourceTagSet = &ec2Stage125IpamResourceTagSet{Items: ec2Stage125IpamResourceTagItemsFrom(item.ResourceTags)}
		}
		out = append(out, entry)
	}
	return out
}

func ec2Stage125IpamResourceTagItemsFrom(in []ec2svc.IpamResourceTag) []ec2Stage125IpamResourceTagItem {
	out := make([]ec2Stage125IpamResourceTagItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage125IpamResourceTagItem{
			Key:   item.Key,
			Value: item.Value,
		})
	}
	return out
}

func ec2Stage125LaunchTemplateDataItemFrom(in ec2svc.LaunchTemplateDataResponse) ec2Stage125LaunchTemplateDataItem {
	out := ec2Stage125LaunchTemplateDataItem{
		ImageID:      in.ImageID,
		InstanceType: in.InstanceType,
		KeyName:      in.KeyName,
		UserData:     in.UserData,
	}
	if len(in.SecurityGroupIDs) > 0 {
		out.SecurityGroupIDSet = &ec2StringSet{Items: append([]string(nil), in.SecurityGroupIDs...)}
	}
	if len(in.SecurityGroups) > 0 {
		out.SecurityGroupSet = &ec2StringSet{Items: append([]string(nil), in.SecurityGroups...)}
	}
	return out
}

func ec2Stage125PrefixListAssociationItemsFrom(in []ec2svc.PrefixListAssociation) []ec2Stage125PrefixListAssociationItem {
	out := make([]ec2Stage125PrefixListAssociationItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2Stage125PrefixListAssociationItem{
			ResourceID:    item.ResourceID,
			ResourceOwner: item.ResourceOwner,
		})
	}
	return out
}

type ec2Stage125GetInstanceUefiDataResponse struct {
	XMLName    xml.Name `xml:"GetInstanceUefiDataResponse"`
	Xmlns      string   `xml:"xmlns,attr"`
	RequestID  string   `xml:"requestId"`
	InstanceID string   `xml:"instanceId,omitempty"`
	UefiData   string   `xml:"uefiData,omitempty"`
}

type ec2Stage125GetIpamAddressHistoryResponse struct {
	XMLName          xml.Name                               `xml:"GetIpamAddressHistoryResponse"`
	Xmlns            string                                 `xml:"xmlns,attr"`
	RequestID        string                                 `xml:"requestId"`
	HistoryRecordSet ec2Stage125IpamAddressHistoryRecordSet `xml:"historyRecordSet"`
	NextToken        *string                                `xml:"nextToken,omitempty"`
}

type ec2Stage125IpamAddressHistoryRecordSet struct {
	Items []ec2Stage125IpamAddressHistoryRecordItem `xml:"item"`
}

type ec2Stage125IpamAddressHistoryRecordItem struct {
	ResourceCidr             string `xml:"resourceCidr,omitempty"`
	ResourceComplianceStatus string `xml:"resourceComplianceStatus,omitempty"`
	ResourceID               string `xml:"resourceId,omitempty"`
	ResourceName             string `xml:"resourceName,omitempty"`
	ResourceOverlapStatus    string `xml:"resourceOverlapStatus,omitempty"`
	ResourceOwnerID          string `xml:"resourceOwnerId,omitempty"`
	ResourceRegion           string `xml:"resourceRegion,omitempty"`
	ResourceType             string `xml:"resourceType,omitempty"`
	SampledEndTime           string `xml:"sampledEndTime,omitempty"`
	SampledStartTime         string `xml:"sampledStartTime,omitempty"`
	VpcID                    string `xml:"vpcId,omitempty"`
}

type ec2Stage125GetIpamDiscoveredAccountsResponse struct {
	XMLName                  xml.Name                            `xml:"GetIpamDiscoveredAccountsResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	IpamDiscoveredAccountSet ec2Stage125IpamDiscoveredAccountSet `xml:"ipamDiscoveredAccountSet"`
	NextToken                *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage125IpamDiscoveredAccountSet struct {
	Items []ec2Stage125IpamDiscoveredAccountItem `xml:"item"`
}

type ec2Stage125IpamDiscoveredAccountItem struct {
	AccountID                   string                                     `xml:"accountId,omitempty"`
	DiscoveryRegion             string                                     `xml:"discoveryRegion,omitempty"`
	FailureReason               *ec2Stage125IpamDiscoveryFailureReasonItem `xml:"failureReason,omitempty"`
	LastAttemptedDiscoveryTime  string                                     `xml:"lastAttemptedDiscoveryTime,omitempty"`
	LastSuccessfulDiscoveryTime string                                     `xml:"lastSuccessfulDiscoveryTime,omitempty"`
	OrganizationalUnitID        string                                     `xml:"organizationalUnitId,omitempty"`
}

type ec2Stage125IpamDiscoveryFailureReasonItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage125GetIpamDiscoveredPublicAddressesResponse struct {
	XMLName                        xml.Name                                  `xml:"GetIpamDiscoveredPublicAddressesResponse"`
	Xmlns                          string                                    `xml:"xmlns,attr"`
	RequestID                      string                                    `xml:"requestId"`
	IpamDiscoveredPublicAddressSet ec2Stage125IpamDiscoveredPublicAddressSet `xml:"ipamDiscoveredPublicAddressSet"`
	NextToken                      *string                                   `xml:"nextToken,omitempty"`
	OldestSampleTime               string                                    `xml:"oldestSampleTime,omitempty"`
}

type ec2Stage125IpamDiscoveredPublicAddressSet struct {
	Items []ec2Stage125IpamDiscoveredPublicAddressItem `xml:"item"`
}

type ec2Stage125IpamDiscoveredPublicAddressItem struct {
	Address                     string                                        `xml:"address,omitempty"`
	AddressAllocationID         string                                        `xml:"addressAllocationId,omitempty"`
	AddressOwnerID              string                                        `xml:"addressOwnerId,omitempty"`
	AddressRegion               string                                        `xml:"addressRegion,omitempty"`
	AddressType                 string                                        `xml:"addressType,omitempty"`
	AssociationStatus           string                                        `xml:"associationStatus,omitempty"`
	InstanceID                  string                                        `xml:"instanceId,omitempty"`
	IpamResourceDiscoveryID     string                                        `xml:"ipamResourceDiscoveryId,omitempty"`
	NetworkBorderGroup          string                                        `xml:"networkBorderGroup,omitempty"`
	NetworkInterfaceDescription string                                        `xml:"networkInterfaceDescription,omitempty"`
	NetworkInterfaceID          string                                        `xml:"networkInterfaceId,omitempty"`
	PublicIpv4PoolID            string                                        `xml:"publicIpv4PoolId,omitempty"`
	SampleTime                  string                                        `xml:"sampleTime,omitempty"`
	SecurityGroupSet            *ec2Stage125IpamPublicAddressSecurityGroupSet `xml:"securityGroupSet,omitempty"`
	Service                     string                                        `xml:"service,omitempty"`
	ServiceResource             string                                        `xml:"serviceResource,omitempty"`
	SubnetID                    string                                        `xml:"subnetId,omitempty"`
	Tags                        *ec2Stage125IpamPublicAddressTags             `xml:"tags,omitempty"`
	VpcID                       string                                        `xml:"vpcId,omitempty"`
}

type ec2Stage125IpamPublicAddressSecurityGroupSet struct {
	Items []ec2Stage125IpamPublicAddressSecurityGroupItem `xml:"item"`
}

type ec2Stage125IpamPublicAddressSecurityGroupItem struct {
	GroupID   string `xml:"groupId,omitempty"`
	GroupName string `xml:"groupName,omitempty"`
}

type ec2Stage125IpamPublicAddressTags struct {
	EipTagSet ec2Stage125IpamPublicAddressTagSet `xml:"eipTagSet"`
}

type ec2Stage125IpamPublicAddressTagSet struct {
	Items []ec2Stage125IpamPublicAddressTagItem `xml:"item"`
}

type ec2Stage125IpamPublicAddressTagItem struct {
	Key   string `xml:"key,omitempty"`
	Value string `xml:"value,omitempty"`
}

type ec2Stage125GetIpamDiscoveredResourceCidrsResponse struct {
	XMLName                       xml.Name                                 `xml:"GetIpamDiscoveredResourceCidrsResponse"`
	Xmlns                         string                                   `xml:"xmlns,attr"`
	RequestID                     string                                   `xml:"requestId"`
	IpamDiscoveredResourceCidrSet ec2Stage125IpamDiscoveredResourceCidrSet `xml:"ipamDiscoveredResourceCidrSet"`
	NextToken                     *string                                  `xml:"nextToken,omitempty"`
}

type ec2Stage125IpamDiscoveredResourceCidrSet struct {
	Items []ec2Stage125IpamDiscoveredResourceCidrItem `xml:"item"`
}

type ec2Stage125IpamDiscoveredResourceCidrItem struct {
	AvailabilityZoneID               string                         `xml:"availabilityZoneId,omitempty"`
	IpamResourceDiscoveryID          string                         `xml:"ipamResourceDiscoveryId,omitempty"`
	IpSource                         string                         `xml:"ipSource,omitempty"`
	IpUsage                          *float64                       `xml:"ipUsage,omitempty"`
	NetworkInterfaceAttachmentStatus string                         `xml:"networkInterfaceAttachmentStatus,omitempty"`
	ResourceCidr                     string                         `xml:"resourceCidr,omitempty"`
	ResourceID                       string                         `xml:"resourceId,omitempty"`
	ResourceOwnerID                  string                         `xml:"resourceOwnerId,omitempty"`
	ResourceRegion                   string                         `xml:"resourceRegion,omitempty"`
	ResourceTagSet                   *ec2Stage125IpamResourceTagSet `xml:"resourceTagSet,omitempty"`
	ResourceType                     string                         `xml:"resourceType,omitempty"`
	SampleTime                       string                         `xml:"sampleTime,omitempty"`
	SubnetID                         string                         `xml:"subnetId,omitempty"`
	VpcID                            string                         `xml:"vpcId,omitempty"`
}

type ec2Stage125GetIpamPoolAllocationsResponse struct {
	XMLName               xml.Name                         `xml:"GetIpamPoolAllocationsResponse"`
	Xmlns                 string                           `xml:"xmlns,attr"`
	RequestID             string                           `xml:"requestId"`
	IpamPoolAllocationSet ec2Stage125IpamPoolAllocationSet `xml:"ipamPoolAllocationSet"`
	NextToken             *string                          `xml:"nextToken,omitempty"`
}

type ec2Stage125IpamPoolAllocationSet struct {
	Items []ec2IpamPoolAllocationItem `xml:"item"`
}

type ec2Stage125GetIpamPoolCidrsResponse struct {
	XMLName         xml.Name                   `xml:"GetIpamPoolCidrsResponse"`
	Xmlns           string                     `xml:"xmlns,attr"`
	RequestID       string                     `xml:"requestId"`
	IpamPoolCidrSet ec2Stage125IpamPoolCidrSet `xml:"ipamPoolCidrSet"`
	NextToken       *string                    `xml:"nextToken,omitempty"`
}

type ec2Stage125IpamPoolCidrSet struct {
	Items []ec2Stage114IpamPoolCidrItem `xml:"item"`
}

type ec2Stage125GetIpamResourceCidrsResponse struct {
	XMLName             xml.Name                       `xml:"GetIpamResourceCidrsResponse"`
	Xmlns               string                         `xml:"xmlns,attr"`
	RequestID           string                         `xml:"requestId"`
	IpamResourceCidrSet ec2Stage125IpamResourceCidrSet `xml:"ipamResourceCidrSet"`
	NextToken           *string                        `xml:"nextToken,omitempty"`
}

type ec2Stage125IpamResourceCidrSet struct {
	Items []ec2Stage125IpamResourceCidrItem `xml:"item"`
}

type ec2Stage125IpamResourceCidrItem struct {
	AvailabilityZoneID string                         `xml:"availabilityZoneId,omitempty"`
	ComplianceStatus   string                         `xml:"complianceStatus,omitempty"`
	IpamID             string                         `xml:"ipamId,omitempty"`
	IpamPoolID         string                         `xml:"ipamPoolId,omitempty"`
	IpamScopeID        string                         `xml:"ipamScopeId,omitempty"`
	IpUsage            *float64                       `xml:"ipUsage,omitempty"`
	ManagementState    string                         `xml:"managementState,omitempty"`
	OverlapStatus      string                         `xml:"overlapStatus,omitempty"`
	ResourceCidr       string                         `xml:"resourceCidr,omitempty"`
	ResourceID         string                         `xml:"resourceId,omitempty"`
	ResourceName       string                         `xml:"resourceName,omitempty"`
	ResourceOwnerID    string                         `xml:"resourceOwnerId,omitempty"`
	ResourceRegion     string                         `xml:"resourceRegion,omitempty"`
	ResourceTagSet     *ec2Stage125IpamResourceTagSet `xml:"resourceTagSet,omitempty"`
	ResourceType       string                         `xml:"resourceType,omitempty"`
	VpcID              string                         `xml:"vpcId,omitempty"`
}

type ec2Stage125IpamResourceTagSet struct {
	Items []ec2Stage125IpamResourceTagItem `xml:"item"`
}

type ec2Stage125IpamResourceTagItem struct {
	Key   string `xml:"key,omitempty"`
	Value string `xml:"value,omitempty"`
}

type ec2Stage125GetLaunchTemplateDataResponse struct {
	XMLName            xml.Name                          `xml:"GetLaunchTemplateDataResponse"`
	Xmlns              string                            `xml:"xmlns,attr"`
	RequestID          string                            `xml:"requestId"`
	LaunchTemplateData ec2Stage125LaunchTemplateDataItem `xml:"launchTemplateData"`
}

type ec2Stage125LaunchTemplateDataItem struct {
	ImageID            string        `xml:"imageId,omitempty"`
	InstanceType       string        `xml:"instanceType,omitempty"`
	KeyName            string        `xml:"keyName,omitempty"`
	SecurityGroupIDSet *ec2StringSet `xml:"securityGroupIdSet,omitempty"`
	SecurityGroupSet   *ec2StringSet `xml:"securityGroupSet,omitempty"`
	UserData           string        `xml:"userData,omitempty"`
}

type ec2Stage125GetManagedPrefixListAssociationsResponse struct {
	XMLName                  xml.Name                            `xml:"GetManagedPrefixListAssociationsResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	PrefixListAssociationSet ec2Stage125PrefixListAssociationSet `xml:"prefixListAssociationSet"`
	NextToken                *string                             `xml:"nextToken,omitempty"`
}

type ec2Stage125PrefixListAssociationSet struct {
	Items []ec2Stage125PrefixListAssociationItem `xml:"item"`
}

type ec2Stage125PrefixListAssociationItem struct {
	ResourceID    string `xml:"resourceId,omitempty"`
	ResourceOwner string `xml:"resourceOwner,omitempty"`
}
