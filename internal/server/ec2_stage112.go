package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage112Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DeleteLaunchTemplateVersions":
		successes, errors, err := s.ec2.DeleteLaunchTemplateVersions(
			strings.TrimSpace(r.Form.Get("LaunchTemplateId")),
			strings.TrimSpace(r.Form.Get("LaunchTemplateName")),
			parseEC2MembersOrItemList(r.Form, "LaunchTemplateVersion"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLaunchTemplateVersionsResponse{
			XMLName:   xml.Name{Local: "DeleteLaunchTemplateVersionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SuccessfullyDeletedLaunchTemplateVersionSet:   ec2Stage112DeleteLaunchTemplateVersionsSuccessSet{Items: ec2Stage112DeleteLaunchTemplateVersionsSuccessItemsFrom(successes)},
			UnsuccessfullyDeletedLaunchTemplateVersionSet: ec2Stage112DeleteLaunchTemplateVersionsErrorSet{Items: ec2Stage112DeleteLaunchTemplateVersionsErrorItemsFrom(errors)},
		})
		return true
	case "DeleteLocalGatewayRoute":
		route, err := s.ec2.DeleteLocalGatewayRoute(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")),
			parseEC2OptionalString(r.Form.Get("DestinationCidrBlock")),
			parseEC2OptionalString(r.Form.Get("DestinationPrefixListId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLocalGatewayRouteResponse{
			XMLName:   xml.Name{Local: "DeleteLocalGatewayRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Route:     ec2Stage108LocalGatewayRouteItemFrom(route),
		})
		return true
	case "DeleteLocalGatewayRouteTable":
		routeTable, err := s.ec2.DeleteLocalGatewayRouteTable(strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLocalGatewayRouteTableResponse{
			XMLName:                xml.Name{Local: "DeleteLocalGatewayRouteTableResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			LocalGatewayRouteTable: ec2Stage108LocalGatewayRouteTableItemFrom(routeTable),
		})
		return true
	case "DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociation":
		association, err := s.ec2.DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociation(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableVirtualInterfaceGroupAssociationId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse{
			XMLName:   xml.Name{Local: "DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			LocalGatewayRouteTableVirtualInterfaceGroupAssociation: ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItemFrom(association),
		})
		return true
	case "DeleteLocalGatewayRouteTableVpcAssociation":
		association, err := s.ec2.DeleteLocalGatewayRouteTableVpcAssociation(
			strings.TrimSpace(r.Form.Get("LocalGatewayRouteTableVpcAssociationId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLocalGatewayRouteTableVpcAssociationResponse{
			XMLName:                              xml.Name{Local: "DeleteLocalGatewayRouteTableVpcAssociationResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			LocalGatewayRouteTableVpcAssociation: ec2Stage108LocalGatewayRouteTableVpcAssociationItemFrom(association),
		})
		return true
	case "DeleteLocalGatewayVirtualInterface":
		virtualInterface, err := s.ec2.DeleteLocalGatewayVirtualInterface(strings.TrimSpace(r.Form.Get("LocalGatewayVirtualInterfaceId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLocalGatewayVirtualInterfaceResponse{
			XMLName:                      xml.Name{Local: "DeleteLocalGatewayVirtualInterfaceResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			LocalGatewayVirtualInterface: ec2Stage108LocalGatewayVirtualInterfaceItemFrom(virtualInterface),
		})
		return true
	case "DeleteLocalGatewayVirtualInterfaceGroup":
		group, err := s.ec2.DeleteLocalGatewayVirtualInterfaceGroup(strings.TrimSpace(r.Form.Get("LocalGatewayVirtualInterfaceGroupId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteLocalGatewayVirtualInterfaceGroupResponse{
			XMLName:                           xml.Name{Local: "DeleteLocalGatewayVirtualInterfaceGroupResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			LocalGatewayVirtualInterfaceGroup: ec2Stage109LocalGatewayVirtualInterfaceGroupItemFrom(group),
		})
		return true
	case "DeleteManagedPrefixList":
		prefixList, err := s.ec2.DeleteManagedPrefixList(strings.TrimSpace(r.Form.Get("PrefixListId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteManagedPrefixListResponse{
			XMLName:    xml.Name{Local: "DeleteManagedPrefixListResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			PrefixList: ec2Stage109ManagedPrefixListItemFrom(prefixList),
		})
		return true
	case "DeleteNetworkInsightsAccessScope":
		networkInsightsAccessScopeID, err := s.ec2.DeleteNetworkInsightsAccessScope(strings.TrimSpace(r.Form.Get("NetworkInsightsAccessScopeId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteNetworkInsightsAccessScopeResponse{
			XMLName:                      xml.Name{Local: "DeleteNetworkInsightsAccessScopeResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			NetworkInsightsAccessScopeID: networkInsightsAccessScopeID,
		})
		return true
	case "DeleteNetworkInsightsAccessScopeAnalysis":
		networkInsightsAccessScopeAnalysisID, err := s.ec2.DeleteNetworkInsightsAccessScopeAnalysis(strings.TrimSpace(r.Form.Get("NetworkInsightsAccessScopeAnalysisId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2Stage112DeleteNetworkInsightsAccessScopeAnalysisResponse{
			XMLName:                              xml.Name{Local: "DeleteNetworkInsightsAccessScopeAnalysisResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			NetworkInsightsAccessScopeAnalysisID: networkInsightsAccessScopeAnalysisID,
		})
		return true
	default:
		return false
	}
}

func ec2Stage112DeleteLaunchTemplateVersionsSuccessItemsFrom(in []ec2svc.DeleteLaunchTemplateVersionsResponseSuccessItem) []ec2Stage112DeleteLaunchTemplateVersionsSuccessItem {
	out := make([]ec2Stage112DeleteLaunchTemplateVersionsSuccessItem, 0, len(in))
	for _, item := range in {
		versionNumber := item.VersionNumber
		out = append(out, ec2Stage112DeleteLaunchTemplateVersionsSuccessItem{
			LaunchTemplateID:   item.LaunchTemplateID,
			LaunchTemplateName: item.LaunchTemplateName,
			VersionNumber:      &versionNumber,
		})
	}
	return out
}

func ec2Stage112DeleteLaunchTemplateVersionsErrorItemsFrom(in []ec2svc.DeleteLaunchTemplateVersionsResponseErrorItem) []ec2Stage112DeleteLaunchTemplateVersionsErrorItem {
	out := make([]ec2Stage112DeleteLaunchTemplateVersionsErrorItem, 0, len(in))
	for _, item := range in {
		var versionNumber *int64
		if item.VersionNumber != nil {
			value := *item.VersionNumber
			versionNumber = &value
		}
		out = append(out, ec2Stage112DeleteLaunchTemplateVersionsErrorItem{
			LaunchTemplateID:   item.LaunchTemplateID,
			LaunchTemplateName: item.LaunchTemplateName,
			ResponseError: ec2Stage112DeleteLaunchTemplateVersionsResponseError{
				Code:    item.ResponseError.Code,
				Message: item.ResponseError.Message,
			},
			VersionNumber: versionNumber,
		})
	}
	return out
}

type ec2Stage112DeleteLaunchTemplateVersionsResponse struct {
	XMLName                                       xml.Name                                          `xml:"DeleteLaunchTemplateVersionsResponse"`
	Xmlns                                         string                                            `xml:"xmlns,attr"`
	RequestID                                     string                                            `xml:"requestId"`
	SuccessfullyDeletedLaunchTemplateVersionSet   ec2Stage112DeleteLaunchTemplateVersionsSuccessSet `xml:"successfullyDeletedLaunchTemplateVersionSet"`
	UnsuccessfullyDeletedLaunchTemplateVersionSet ec2Stage112DeleteLaunchTemplateVersionsErrorSet   `xml:"unsuccessfullyDeletedLaunchTemplateVersionSet"`
}

type ec2Stage112DeleteLaunchTemplateVersionsSuccessSet struct {
	Items []ec2Stage112DeleteLaunchTemplateVersionsSuccessItem `xml:"item"`
}

type ec2Stage112DeleteLaunchTemplateVersionsSuccessItem struct {
	LaunchTemplateID   string `xml:"launchTemplateId,omitempty"`
	LaunchTemplateName string `xml:"launchTemplateName,omitempty"`
	VersionNumber      *int64 `xml:"versionNumber,omitempty"`
}

type ec2Stage112DeleteLaunchTemplateVersionsErrorSet struct {
	Items []ec2Stage112DeleteLaunchTemplateVersionsErrorItem `xml:"item"`
}

type ec2Stage112DeleteLaunchTemplateVersionsErrorItem struct {
	LaunchTemplateID   string                                               `xml:"launchTemplateId,omitempty"`
	LaunchTemplateName string                                               `xml:"launchTemplateName,omitempty"`
	ResponseError      ec2Stage112DeleteLaunchTemplateVersionsResponseError `xml:"responseError"`
	VersionNumber      *int64                                               `xml:"versionNumber,omitempty"`
}

type ec2Stage112DeleteLaunchTemplateVersionsResponseError struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2Stage112DeleteLocalGatewayRouteResponse struct {
	XMLName   xml.Name                         `xml:"DeleteLocalGatewayRouteResponse"`
	Xmlns     string                           `xml:"xmlns,attr"`
	RequestID string                           `xml:"requestId"`
	Route     ec2Stage108LocalGatewayRouteItem `xml:"route"`
}

type ec2Stage112DeleteLocalGatewayRouteTableResponse struct {
	XMLName                xml.Name                              `xml:"DeleteLocalGatewayRouteTableResponse"`
	Xmlns                  string                                `xml:"xmlns,attr"`
	RequestID              string                                `xml:"requestId"`
	LocalGatewayRouteTable ec2Stage108LocalGatewayRouteTableItem `xml:"localGatewayRouteTable"`
}

type ec2Stage112DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse struct {
	XMLName                                                xml.Name                                                              `xml:"DeleteLocalGatewayRouteTableVirtualInterfaceGroupAssociationResponse"`
	Xmlns                                                  string                                                                `xml:"xmlns,attr"`
	RequestID                                              string                                                                `xml:"requestId"`
	LocalGatewayRouteTableVirtualInterfaceGroupAssociation ec2Stage108LocalGatewayRouteTableVirtualInterfaceGroupAssociationItem `xml:"localGatewayRouteTableVirtualInterfaceGroupAssociation"`
}

type ec2Stage112DeleteLocalGatewayRouteTableVpcAssociationResponse struct {
	XMLName                              xml.Name                                            `xml:"DeleteLocalGatewayRouteTableVpcAssociationResponse"`
	Xmlns                                string                                              `xml:"xmlns,attr"`
	RequestID                            string                                              `xml:"requestId"`
	LocalGatewayRouteTableVpcAssociation ec2Stage108LocalGatewayRouteTableVpcAssociationItem `xml:"localGatewayRouteTableVpcAssociation"`
}

type ec2Stage112DeleteLocalGatewayVirtualInterfaceResponse struct {
	XMLName                      xml.Name                                    `xml:"DeleteLocalGatewayVirtualInterfaceResponse"`
	Xmlns                        string                                      `xml:"xmlns,attr"`
	RequestID                    string                                      `xml:"requestId"`
	LocalGatewayVirtualInterface ec2Stage108LocalGatewayVirtualInterfaceItem `xml:"localGatewayVirtualInterface"`
}

type ec2Stage112DeleteLocalGatewayVirtualInterfaceGroupResponse struct {
	XMLName                           xml.Name                                         `xml:"DeleteLocalGatewayVirtualInterfaceGroupResponse"`
	Xmlns                             string                                           `xml:"xmlns,attr"`
	RequestID                         string                                           `xml:"requestId"`
	LocalGatewayVirtualInterfaceGroup ec2Stage109LocalGatewayVirtualInterfaceGroupItem `xml:"localGatewayVirtualInterfaceGroup"`
}

type ec2Stage112DeleteManagedPrefixListResponse struct {
	XMLName    xml.Name                         `xml:"DeleteManagedPrefixListResponse"`
	Xmlns      string                           `xml:"xmlns,attr"`
	RequestID  string                           `xml:"requestId"`
	PrefixList ec2Stage109ManagedPrefixListItem `xml:"prefixList"`
}

type ec2Stage112DeleteNetworkInsightsAccessScopeResponse struct {
	XMLName                      xml.Name `xml:"DeleteNetworkInsightsAccessScopeResponse"`
	Xmlns                        string   `xml:"xmlns,attr"`
	RequestID                    string   `xml:"requestId"`
	NetworkInsightsAccessScopeID string   `xml:"networkInsightsAccessScopeId,omitempty"`
}

type ec2Stage112DeleteNetworkInsightsAccessScopeAnalysisResponse struct {
	XMLName                              xml.Name `xml:"DeleteNetworkInsightsAccessScopeAnalysisResponse"`
	Xmlns                                string   `xml:"xmlns,attr"`
	RequestID                            string   `xml:"requestId"`
	NetworkInsightsAccessScopeAnalysisID string   `xml:"networkInsightsAccessScopeAnalysisId,omitempty"`
}
