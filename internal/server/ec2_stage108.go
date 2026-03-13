package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage108Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateIpamPool":
		allocationDefaultNetmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("AllocationDefaultNetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		allocationMaxNetmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("AllocationMaxNetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		allocationMinNetmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("AllocationMinNetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		autoImport, hasAutoImport, ok := ec2OptionalBoolFromForm(r.Form, "AutoImport")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAutoImport {
			autoImport = nil
		}
		publiclyAdvertisable, hasPubliclyAdvertisable, ok := ec2OptionalBoolFromForm(r.Form, "PubliclyAdvertisable")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPubliclyAdvertisable {
			publiclyAdvertisable = nil
		}

		ipamPool, err := s.ec2.CreateIpamPool(
			strings.TrimSpace(r.Form.Get("AddressFamily")),
			strings.TrimSpace(r.Form.Get("IpamScopeId")),
			allocationDefaultNetmaskLength,
			allocationMaxNetmaskLength,
			allocationMinNetmaskLength,
			autoImport,
			strings.TrimSpace(r.Form.Get("AwsService")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2OptionalString(r.Form.Get("Locale")),
			strings.TrimSpace(r.Form.Get("PublicIpSource")),
			publiclyAdvertisable,
			parseEC2OptionalString(r.Form.Get("SourceIpamPoolId")),
			parseEC2TagSpecificationsForResource(r.Form, "ipam-pool"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateIpamPoolResponse{
			XMLName:   xml.Name{Local: "CreateIpamPoolResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamPool:  ec2Stage108IpamPoolItemFrom(ipamPool),
		})
		return true
	case "CreateIpamResourceDiscovery":
		discovery, err := s.ec2.CreateIpamResourceDiscovery(
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2Stage107OperatingRegions(r.Form),
			parseEC2TagSpecificationsForResource(r.Form, "ipam-resource-discovery"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateIpamResourceDiscoveryResponse{
			XMLName:               xml.Name{Local: "CreateIpamResourceDiscoveryResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			IpamResourceDiscovery: ec2Stage108IpamResourceDiscoveryItemFrom(discovery),
		})
		return true
	case "CreateIpamScope":
		scope, err := s.ec2.CreateIpamScope(
			strings.TrimSpace(r.Form.Get("IpamId")),
			parseEC2OptionalString(r.Form.Get("Description")),
			parseEC2TagSpecificationsForResource(r.Form, "ipam-scope"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateIpamScopeResponse{
			XMLName:   xml.Name{Local: "CreateIpamScopeResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			IpamScope: ec2Stage108IpamScopeItemFrom(scope),
		})
		return true
	case "CreateLaunchTemplate":
		if !hasEC2PrefixedField(r.Form, "LaunchTemplateData.") && !hasEC2Field(r.Form, "LaunchTemplateData") {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		launchTemplate, _, err := s.ec2.CreateLaunchTemplate(
			strings.TrimSpace(r.Form.Get("LaunchTemplateName")),
			true,
			parseEC2OptionalString(r.Form.Get("ClientToken")),
			parseEC2OptionalString(r.Form.Get("VersionDescription")),
			parseEC2TagSpecificationsForResource(r.Form, "launch-template"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLaunchTemplateResponse{
			XMLName:        xml.Name{Local: "CreateLaunchTemplateResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			LaunchTemplate: ec2Stage108LaunchTemplateItemFrom(launchTemplate),
		})
		return true
	case "CreateLaunchTemplateVersion":
		if !hasEC2PrefixedField(r.Form, "LaunchTemplateData.") && !hasEC2Field(r.Form, "LaunchTemplateData") {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		resolveAlias, hasResolveAlias, ok := ec2OptionalBoolFromForm(r.Form, "ResolveAlias")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasResolveAlias {
			resolveAlias = nil
		}
		version, err := s.ec2.CreateLaunchTemplateVersion(
			strings.TrimSpace(r.Form.Get("LaunchTemplateId")),
			strings.TrimSpace(r.Form.Get("LaunchTemplateName")),
			true,
			parseEC2OptionalString(r.Form.Get("SourceVersion")),
			parseEC2OptionalString(r.Form.Get("VersionDescription")),
			resolveAlias,
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLaunchTemplateVersionResponse{
			XMLName:               xml.Name{Local: "CreateLaunchTemplateVersionResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			LaunchTemplateVersion: ec2Stage108LaunchTemplateVersionItemFrom(version),
		})
		return true
	case "CreateLocalGatewayRoute":
		route, err := s.ec2.CreateLocalGatewayRoute(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			parseEC2OptionalString(r.Form.Get("DestinationCidrBlock")),
			parseEC2OptionalString(r.Form.Get("DestinationPrefixListId")),
			parseEC2OptionalString(r.Form.Get("LocalGatewayVirtualInterfaceGroupId")),
			parseEC2OptionalString(r.Form.Get("NetworkInterfaceId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLocalGatewayRouteResponse{
			XMLName:   xml.Name{Local: "CreateLocalGatewayRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Route:     ec2Stage108LocalGatewayRouteItemFrom(route),
		})
		return true
	case "CreateLocalGatewayRouteTable":
		routeTable, err := s.ec2.CreateLocalGatewayRouteTable(
			strings.TrimSpace(r.Form.Get("LocalGatewayId")),
			strings.TrimSpace(r.Form.Get("Mode")),
			parseEC2TagSpecificationsForResource(r.Form, "local-gateway-route-table"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLocalGatewayRouteTableResponse{
			XMLName:                xml.Name{Local: "CreateLocalGatewayRouteTableResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			LocalGatewayRouteTable: ec2Stage108LocalGatewayRouteTableItemFrom(routeTable),
		})
		return true
	case "CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation":
		association, err := s.ec2.CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociation(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("LocalGatewayVirtualInterfaceGroupId")),
			parseEC2TagSpecificationsForResource(r.Form, "local-gateway-route-table-virtual-interface-group-association"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse{
			XMLName:   xml.Name{Local: "CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayRouteTableVirtualInterfaceGroupAssociation: ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItemFrom(association),
		})
		return true
	case "CreateLocalGatewayRouteTableVpcAssociation":
		association, err := s.ec2.CreateLocalGatewayRouteTableVpcAssociation(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2TagSpecificationsForResource(r.Form, "local-gateway-route-table-vpc-association"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLocalGatewayRouteTableVpcAssociationResponse{
			XMLName:                              xml.Name{Local: "CreateLocalGatewayRouteTableVpcAssociationResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			LocalGatewayRouteTableVpcAssociation: ec2Stage108LocalGatewayRouteTableVpcAssociationItemFrom(association),
		})
		return true
	case "CreateLocalGatewayVirtualInterface":
		vlan, ok := parseEC2OptionalInt32(r.Form.Get("Vlan"))
		if !ok || vlan == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		peerBgpASN, ok := parseEC2OptionalInt32(r.Form.Get("PeerBgpAsn"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		peerBgpASNExtended, ok := parseEC2OptionalInt64(r.Form.Get("PeerBgpAsnExtended"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		virtualInterface, err := s.ec2.CreateLocalGatewayVirtualInterface(
			strings.TrimSpace(r.Form.Get("LocalAddress")),
			strings.TrimSpace(r.Form.Get("LocalGatewayVirtualInterfaceGroupId")),
			strings.TrimSpace(r.Form.Get("OutpostLagId")),
			strings.TrimSpace(r.Form.Get("PeerAddress")),
			*vlan,
			peerBgpASN,
			peerBgpASNExtended,
			parseEC2TagSpecificationsForResource(r.Form, "local-gateway-virtual-interface"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage108CreateLocalGatewayVirtualInterfaceResponse{
			XMLName:                      xml.Name{Local: "CreateLocalGatewayVirtualInterfaceResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			LocalGatewayVirtualInterface: ec2Stage108LocalGatewayVirtualInterfaceItemFrom(virtualInterface),
		})
		return true
	default:
		return false
	}
}

func ec2Stage108IpamPoolItemFrom(in ec2svc.IpamPool) ec2Stage108IpamPoolItem {
	out := ec2Stage108IpamPoolItem{
		AddressFamily:                  in.AddressFamily,
		AllocationDefaultNetmaskLength: in.AllocationDefaultNetmaskLength,
		AllocationMaxNetmaskLength:     in.AllocationMaxNetmaskLength,
		AllocationMinNetmaskLength:     in.AllocationMinNetmaskLength,
		AwsService:                     in.AwsService,
		Description:                    in.Description,
		IpamARN:                        in.IpamARN,
		IpamPoolARN:                    in.IpamPoolARN,
		IpamPoolID:                     in.IpamPoolID,
		IpamRegion:                     in.IpamRegion,
		IpamScopeARN:                   in.IpamScopeARN,
		IpamScopeID:                    in.IpamScopeID,
		IpamScopeType:                  in.IpamScopeType,
		Locale:                         in.Locale,
		OwnerID:                        in.OwnerID,
		PublicIpSource:                 in.PublicIpSource,
		SourceIpamPoolID:               in.SourceIpamPoolID,
		State:                          in.State,
		TagSet:                         ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
	out.AutoImport = &in.AutoImport
	out.PubliclyAdvertisable = &in.PubliclyAdvertisable
	return out
}

func ec2Stage108IpamResourceDiscoveryItemFrom(in ec2svc.IpamResourceDiscovery) ec2Stage108IpamResourceDiscoveryItem {
	out := ec2Stage108IpamResourceDiscoveryItem{
		Description:              in.Description,
		IpamResourceDiscoveryARN: in.IpamResourceDiscoveryARN,
		IpamResourceDiscoveryID:  in.IpamResourceDiscoveryID,
		IpamResourceRegion:       in.IpamResourceRegion,
		OperatingRegionSet:       ec2Stage108OperatingRegionSetFrom(in.OperatingRegions),
		OwnerID:                  in.OwnerID,
		State:                    in.State,
		TagSet:                   ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
	out.IsDefault = &in.IsDefault
	return out
}

func ec2Stage108OperatingRegionSetFrom(in []string) ec2Stage108OperatingRegionSet {
	items := make([]ec2Stage108OperatingRegionItem, 0, len(in))
	for _, region := range in {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		items = append(items, ec2Stage108OperatingRegionItem{RegionName: region})
	}
	return ec2Stage108OperatingRegionSet{Items: items}
}

func ec2Stage108IpamScopeItemFrom(in ec2svc.IpamScope) ec2Stage108IpamScopeItem {
	out := ec2Stage108IpamScopeItem{
		Description:   in.Description,
		IpamARN:       in.IpamARN,
		IpamRegion:    in.IpamRegion,
		IpamScopeARN:  in.IpamScopeARN,
		IpamScopeID:   in.IpamScopeID,
		IpamScopeType: in.IpamScopeType,
		OwnerID:       in.OwnerID,
		PoolCount:     &in.PoolCount,
		State:         in.State,
		TagSet:        ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
	out.IsDefault = &in.IsDefault
	return out
}

func ec2Stage108LaunchTemplateItemFrom(in ec2svc.LaunchTemplate) ec2Stage108LaunchTemplateItem {
	defaultVersionNumber := in.DefaultVersionNumber
	latestVersionNumber := in.LatestVersionNumber
	return ec2Stage108LaunchTemplateItem{
		CreatedBy:            in.CreatedBy,
		CreateTime:           in.CreateTime.UTC().Format(time.RFC3339),
		DefaultVersionNumber: &defaultVersionNumber,
		LatestVersionNumber:  &latestVersionNumber,
		LaunchTemplateID:     in.LaunchTemplateID,
		LaunchTemplateName:   in.LaunchTemplateName,
		TagSet:               ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
}

func ec2Stage108LaunchTemplateVersionItemFrom(in ec2svc.LaunchTemplateVersion) ec2Stage108LaunchTemplateVersionItem {
	versionNumber := in.VersionNumber
	out := ec2Stage108LaunchTemplateVersionItem{
		CreatedBy:          in.CreatedBy,
		CreateTime:         in.CreateTime.UTC().Format(time.RFC3339),
		LaunchTemplateID:   in.LaunchTemplateID,
		LaunchTemplateName: in.LaunchTemplateName,
		VersionDescription: in.VersionDescription,
		VersionNumber:      &versionNumber,
	}
	out.DefaultVersion = &in.DefaultVersion
	return out
}

func ec2Stage108LocalGatewayRouteItemFrom(in ec2svc.LocalGatewayRoute) ec2Stage108LocalGatewayRouteItem {
	return ec2Stage108LocalGatewayRouteItem{
		CoipPoolID:                          in.CoipPoolID,
		DestinationCidrBlock:                in.DestinationCidrBlock,
		DestinationPrefixListID:             in.DestinationPrefixListID,
		LocalGatewayRouteTableARN:           in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:            in.LocalGatewayRouteTableID,
		LocalGatewayVirtualInterfaceGroupID: in.LocalGatewayVirtualInterfaceGroupID,
		NetworkInterfaceID:                  in.NetworkInterfaceID,
		OwnerID:                             in.OwnerID,
		State:                               in.State,
		SubnetID:                            in.SubnetID,
		Type:                                in.Type,
	}
}

func ec2Stage108LocalGatewayRouteTableItemFrom(in ec2svc.LocalGatewayRouteTable) ec2Stage108LocalGatewayRouteTableItem {
	return ec2Stage108LocalGatewayRouteTableItem{
		LocalGatewayID:            in.LocalGatewayID,
		LocalGatewayRouteTableARN: in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:  in.LocalGatewayRouteTableID,
		Mode:                      in.Mode,
		OutpostARN:                in.OutpostARN,
		OwnerID:                   in.OwnerID,
		State:                     in.State,
		TagSet:                    ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
}

func ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItemFrom(in ec2svc.LocalGatewayRouteTableVirtualInterfaceGroupAssociation) ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem {
	return ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem{
		LocalGatewayID:            in.LocalGatewayID,
		LocalGatewayRouteTableARN: in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:  in.LocalGatewayRouteTableID,
		LocalGatewayRouteTableVirtualInterfaceGroupAssociationID: in.LocalGatewayRouteTableVirtualInterfaceGroupAssociationID,
		LocalGatewayVirtualInterfaceGroupID:                      in.LocalGatewayVirtualInterfaceGroupID,
		OwnerID:                                                  in.OwnerID,
		State:                                                    in.State,
		TagSet:                                                   ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
}

func ec2Stage108LocalGatewayRouteTableVpcAssociationItemFrom(in ec2svc.LocalGatewayRouteTableVpcAssociation) ec2Stage108LocalGatewayRouteTableVpcAssociationItem {
	return ec2Stage108LocalGatewayRouteTableVpcAssociationItem{
		LocalGatewayID:                         in.LocalGatewayID,
		LocalGatewayRouteTableARN:              in.LocalGatewayRouteTableARN,
		LocalGatewayRouteTableID:               in.LocalGatewayRouteTableID,
		LocalGatewayRouteTableVpcAssociationID: in.LocalGatewayRouteTableVpcAssociationID,
		OwnerID:                                in.OwnerID,
		State:                                  in.State,
		TagSet:                                 ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		VpcID:                                  in.VpcID,
	}
}

func ec2Stage108LocalGatewayVirtualInterfaceItemFrom(in ec2svc.LocalGatewayVirtualInterface) ec2Stage108LocalGatewayVirtualInterfaceItem {
	return ec2Stage108LocalGatewayVirtualInterfaceItem{
		LocalAddress:                   in.LocalAddress,
		LocalGatewayID:                 in.LocalGatewayID,
		LocalGatewayVirtualInterfaceID: in.LocalGatewayVirtualInterfaceID,
		OwnerID:                        in.OwnerID,
		PeerAddress:                    in.PeerAddress,
		PeerBgpASN:                     in.PeerBgpASN,
		TagSet:                         ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		VLAN:                           in.VLAN,
	}
}

type ec2Stage108CreateIpamPoolResponse struct {
	XMLName   xml.Name                `xml:"CreateIpamPoolResponse"`
	Xmlns     string                  `xml:"xmlns,attr"`
	RequestID string                  `xml:"requestId"`
	IpamPool  ec2Stage108IpamPoolItem `xml:"ipamPool"`
}

type ec2Stage108IpamPoolItem struct {
	AddressFamily                  string    `xml:"addressFamily,omitempty"`
	AllocationDefaultNetmaskLength *int32    `xml:"allocationDefaultNetmaskLength,omitempty"`
	AllocationMaxNetmaskLength     *int32    `xml:"allocationMaxNetmaskLength,omitempty"`
	AllocationMinNetmaskLength     *int32    `xml:"allocationMinNetmaskLength,omitempty"`
	AutoImport                     *bool     `xml:"autoImport,omitempty"`
	AwsService                     string    `xml:"awsService,omitempty"`
	Description                    string    `xml:"description,omitempty"`
	IpamARN                        string    `xml:"ipamArn,omitempty"`
	IpamPoolARN                    string    `xml:"ipamPoolArn,omitempty"`
	IpamPoolID                     string    `xml:"ipamPoolId,omitempty"`
	IpamRegion                     string    `xml:"ipamRegion,omitempty"`
	IpamScopeARN                   string    `xml:"ipamScopeArn,omitempty"`
	IpamScopeID                    string    `xml:"ipamScopeId,omitempty"`
	IpamScopeType                  string    `xml:"ipamScopeType,omitempty"`
	Locale                         string    `xml:"locale,omitempty"`
	OwnerID                        string    `xml:"ownerId,omitempty"`
	PublicIpSource                 string    `xml:"publicIpSource,omitempty"`
	PubliclyAdvertisable           *bool     `xml:"publiclyAdvertisable,omitempty"`
	SourceIpamPoolID               string    `xml:"sourceIpamPoolId,omitempty"`
	State                          string    `xml:"state,omitempty"`
	TagSet                         ec2TagSet `xml:"tagSet"`
}

type ec2Stage108CreateIpamResourceDiscoveryResponse struct {
	XMLName               xml.Name                             `xml:"CreateIpamResourceDiscoveryResponse"`
	Xmlns                 string                               `xml:"xmlns,attr"`
	RequestID             string                               `xml:"requestId"`
	IpamResourceDiscovery ec2Stage108IpamResourceDiscoveryItem `xml:"ipamResourceDiscovery"`
}

type ec2Stage108IpamResourceDiscoveryItem struct {
	Description              string                        `xml:"description,omitempty"`
	IpamResourceDiscoveryARN string                        `xml:"ipamResourceDiscoveryArn,omitempty"`
	IpamResourceDiscoveryID  string                        `xml:"ipamResourceDiscoveryId,omitempty"`
	IpamResourceRegion       string                        `xml:"ipamResourceDiscoveryRegion,omitempty"`
	IsDefault                *bool                         `xml:"isDefault,omitempty"`
	OperatingRegionSet       ec2Stage108OperatingRegionSet `xml:"operatingRegionSet"`
	OwnerID                  string                        `xml:"ownerId,omitempty"`
	State                    string                        `xml:"state,omitempty"`
	TagSet                   ec2TagSet                     `xml:"tagSet"`
}

type ec2Stage108OperatingRegionSet struct {
	Items []ec2Stage108OperatingRegionItem `xml:"item"`
}

type ec2Stage108OperatingRegionItem struct {
	RegionName string `xml:"regionName,omitempty"`
}

type ec2Stage108CreateIpamScopeResponse struct {
	XMLName   xml.Name                 `xml:"CreateIpamScopeResponse"`
	Xmlns     string                   `xml:"xmlns,attr"`
	RequestID string                   `xml:"requestId"`
	IpamScope ec2Stage108IpamScopeItem `xml:"ipamScope"`
}

type ec2Stage108IpamScopeItem struct {
	Description   string    `xml:"description,omitempty"`
	IpamARN       string    `xml:"ipamArn,omitempty"`
	IpamRegion    string    `xml:"ipamRegion,omitempty"`
	IpamScopeARN  string    `xml:"ipamScopeArn,omitempty"`
	IpamScopeID   string    `xml:"ipamScopeId,omitempty"`
	IpamScopeType string    `xml:"ipamScopeType,omitempty"`
	IsDefault     *bool     `xml:"isDefault,omitempty"`
	OwnerID       string    `xml:"ownerId,omitempty"`
	PoolCount     *int32    `xml:"poolCount,omitempty"`
	State         string    `xml:"state,omitempty"`
	TagSet        ec2TagSet `xml:"tagSet"`
}

type ec2Stage108CreateLaunchTemplateResponse struct {
	XMLName        xml.Name                      `xml:"CreateLaunchTemplateResponse"`
	Xmlns          string                        `xml:"xmlns,attr"`
	RequestID      string                        `xml:"requestId"`
	LaunchTemplate ec2Stage108LaunchTemplateItem `xml:"launchTemplate"`
}

type ec2Stage108LaunchTemplateItem struct {
	CreatedBy            string    `xml:"createdBy,omitempty"`
	CreateTime           string    `xml:"createTime,omitempty"`
	DefaultVersionNumber *int64    `xml:"defaultVersionNumber,omitempty"`
	LatestVersionNumber  *int64    `xml:"latestVersionNumber,omitempty"`
	LaunchTemplateID     string    `xml:"launchTemplateId,omitempty"`
	LaunchTemplateName   string    `xml:"launchTemplateName,omitempty"`
	TagSet               ec2TagSet `xml:"tagSet"`
}

type ec2Stage108CreateLaunchTemplateVersionResponse struct {
	XMLName               xml.Name                             `xml:"CreateLaunchTemplateVersionResponse"`
	Xmlns                 string                               `xml:"xmlns,attr"`
	RequestID             string                               `xml:"requestId"`
	LaunchTemplateVersion ec2Stage108LaunchTemplateVersionItem `xml:"launchTemplateVersion"`
}

type ec2Stage108LaunchTemplateVersionItem struct {
	CreatedBy          string `xml:"createdBy,omitempty"`
	CreateTime         string `xml:"createTime,omitempty"`
	DefaultVersion     *bool  `xml:"defaultVersion,omitempty"`
	LaunchTemplateID   string `xml:"launchTemplateId,omitempty"`
	LaunchTemplateName string `xml:"launchTemplateName,omitempty"`
	VersionDescription string `xml:"versionDescription,omitempty"`
	VersionNumber      *int64 `xml:"versionNumber,omitempty"`
}

type ec2Stage108CreateLocalGatewayRouteResponse struct {
	XMLName   xml.Name                         `xml:"CreateLocalGatewayRouteResponse"`
	Xmlns     string                           `xml:"xmlns,attr"`
	RequestID string                           `xml:"requestId"`
	Route     ec2Stage108LocalGatewayRouteItem `xml:"route"`
}

type ec2Stage108LocalGatewayRouteItem struct {
	CoipPoolID                          string `xml:"coipPoolId,omitempty"`
	DestinationCidrBlock                string `xml:"destinationCidrBlock,omitempty"`
	DestinationPrefixListID             string `xml:"destinationPrefixListId,omitempty"`
	LocalGatewayRouteTableARN           string `xml:"localGatewayRouteTableArn,omitempty"`
	LocalGatewayRouteTableID            string `xml:"localGatewayRouteTableId,omitempty"`
	LocalGatewayVirtualInterfaceGroupID string `xml:"localGatewayVirtualInterfaceGroupId,omitempty"`
	NetworkInterfaceID                  string `xml:"networkInterfaceId,omitempty"`
	OwnerID                             string `xml:"ownerId,omitempty"`
	State                               string `xml:"state,omitempty"`
	SubnetID                            string `xml:"subnetId,omitempty"`
	Type                                string `xml:"type,omitempty"`
}

type ec2Stage108CreateLocalGatewayRouteTableResponse struct {
	XMLName                xml.Name                              `xml:"CreateLocalGatewayRouteTableResponse"`
	Xmlns                  string                                `xml:"xmlns,attr"`
	RequestID              string                                `xml:"requestId"`
	LocalGatewayRouteTable ec2Stage108LocalGatewayRouteTableItem `xml:"localGatewayRouteTable"`
}

type ec2Stage108LocalGatewayRouteTableItem struct {
	LocalGatewayID            string    `xml:"localGatewayId,omitempty"`
	LocalGatewayRouteTableARN string    `xml:"localGatewayRouteTableArn,omitempty"`
	LocalGatewayRouteTableID  string    `xml:"localGatewayRouteTableId,omitempty"`
	Mode                      string    `xml:"mode,omitempty"`
	OutpostARN                string    `xml:"outpostArn,omitempty"`
	OwnerID                   string    `xml:"ownerId,omitempty"`
	State                     string    `xml:"state,omitempty"`
	TagSet                    ec2TagSet `xml:"tagSet"`
}

type ec2Stage108CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse struct {
	XMLName                                                xml.Name                                                              `xml:"CreateLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse"`
	Xmlns                                                  string                                                                `xml:"xmlns,attr"`
	RequestID                                              string                                                                `xml:"requestId"`
	LocalGatewayRouteTableVirtualInterfaceGroupAssociation ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem `xml:"localGatewayRouteTableVirtualInterfaceGroupAssociation"`
}

type ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem struct {
	LocalGatewayID                                           string    `xml:"localGatewayId,omitempty"`
	LocalGatewayRouteTableARN                                string    `xml:"localGatewayRouteTableArn,omitempty"`
	LocalGatewayRouteTableID                                 string    `xml:"localGatewayRouteTableId,omitempty"`
	LocalGatewayRouteTableVirtualInterfaceGroupAssociationID string    `xml:"localGatewayRouteTableVirtualInterfaceGroupAssociationId,omitempty"`
	LocalGatewayVirtualInterfaceGroupID                      string    `xml:"localGatewayVirtualInterfaceGroupId,omitempty"`
	OwnerID                                                  string    `xml:"ownerId,omitempty"`
	State                                                    string    `xml:"state,omitempty"`
	TagSet                                                   ec2TagSet `xml:"tagSet"`
}

type ec2Stage108CreateLocalGatewayRouteTableVpcAssociationResponse struct {
	XMLName                              xml.Name                                            `xml:"CreateLocalGatewayRouteTableVpcAssociationResponse"`
	Xmlns                                string                                              `xml:"xmlns,attr"`
	RequestID                            string                                              `xml:"requestId"`
	LocalGatewayRouteTableVpcAssociation ec2Stage108LocalGatewayRouteTableVpcAssociationItem `xml:"localGatewayRouteTableVpcAssociation"`
}

type ec2Stage108LocalGatewayRouteTableVpcAssociationItem struct {
	LocalGatewayID                         string    `xml:"localGatewayId,omitempty"`
	LocalGatewayRouteTableARN              string    `xml:"localGatewayRouteTableArn,omitempty"`
	LocalGatewayRouteTableID               string    `xml:"localGatewayRouteTableId,omitempty"`
	LocalGatewayRouteTableVpcAssociationID string    `xml:"localGatewayRouteTableVpcAssociationId,omitempty"`
	OwnerID                                string    `xml:"ownerId,omitempty"`
	State                                  string    `xml:"state,omitempty"`
	TagSet                                 ec2TagSet `xml:"tagSet"`
	VpcID                                  string    `xml:"vpcId,omitempty"`
}

type ec2Stage108CreateLocalGatewayVirtualInterfaceResponse struct {
	XMLName                      xml.Name                                    `xml:"CreateLocalGatewayVirtualInterfaceResponse"`
	Xmlns                        string                                      `xml:"xmlns,attr"`
	RequestID                    string                                      `xml:"requestId"`
	LocalGatewayVirtualInterface ec2Stage108LocalGatewayVirtualInterfaceItem `xml:"localGatewayVirtualInterface"`
}

type ec2Stage108LocalGatewayVirtualInterfaceItem struct {
	LocalAddress                   string    `xml:"localAddress,omitempty"`
	LocalGatewayID                 string    `xml:"localGatewayId,omitempty"`
	LocalGatewayVirtualInterfaceID string    `xml:"localGatewayVirtualInterfaceId,omitempty"`
	OwnerID                        string    `xml:"ownerId,omitempty"`
	PeerAddress                    string    `xml:"peerAddress,omitempty"`
	PeerBgpASN                     *int32    `xml:"peerBgpAsn,omitempty"`
	TagSet                         ec2TagSet `xml:"tagSet"`
	VLAN                           *int32    `xml:"vlan,omitempty"`
}
