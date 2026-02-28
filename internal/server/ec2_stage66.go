package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage66Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateRouteServer":
		association, err := s.ec2.AssociateRouteServer(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateRouteServerResponse{
			XMLName:                xml.Name{Local: "AssociateRouteServerResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			RouteServerAssociation: ec2RouteServerAssociationItemFrom(association),
		})
		return true
	case "DisassociateRouteServer":
		association, err := s.ec2.DisassociateRouteServer(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateRouteServerResponse{
			XMLName:                xml.Name{Local: "DisassociateRouteServerResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			RouteServerAssociation: ec2RouteServerAssociationItemFrom(association),
		})
		return true
	case "GetRouteServerAssociations":
		associations, err := s.ec2.GetRouteServerAssociations(strings.TrimSpace(r.Form.Get("RouteServerId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetRouteServerAssociationsResponse{
			XMLName:   xml.Name{Local: "GetRouteServerAssociationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			RouteServerAssociationSet: ec2RouteServerAssociationSet{
				Items: ec2RouteServerAssociationItems(associations),
			},
		})
		return true
	case "EnableRouteServerPropagation":
		propagation, err := s.ec2.EnableRouteServerPropagation(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			strings.TrimSpace(r.Form.Get("RouteTableId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EnableRouteServerPropagationResponse{
			XMLName:                xml.Name{Local: "EnableRouteServerPropagationResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			RouteServerPropagation: ec2RouteServerPropagationItemFrom(propagation),
		})
		return true
	case "DisableRouteServerPropagation":
		propagation, err := s.ec2.DisableRouteServerPropagation(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			strings.TrimSpace(r.Form.Get("RouteTableId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisableRouteServerPropagationResponse{
			XMLName:                xml.Name{Local: "DisableRouteServerPropagationResponse"},
			Xmlns:                  ec2Namespace,
			RequestID:              "stackyard-request",
			RouteServerPropagation: ec2RouteServerPropagationItemFrom(propagation),
		})
		return true
	case "GetRouteServerPropagations":
		propagations, err := s.ec2.GetRouteServerPropagations(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			parseEC2OptionalString(r.Form.Get("RouteTableId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2GetRouteServerPropagationsResponse{
			XMLName:   xml.Name{Local: "GetRouteServerPropagationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			RouteServerPropagationSet: ec2RouteServerPropagationSet{
				Items: ec2RouteServerPropagationItems(propagations),
			},
		})
		return true
	case "ModifyRouteServer":
		persistRoutesDuration, ok := parseEC2OptionalInt64(r.Form.Get("PersistRoutesDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		snsEnabled, hasSNSValue, ok := ec2OptionalBoolFromForm(r.Form, "SnsNotificationsEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasSNSValue {
			snsEnabled = nil
		}

		routeServer, err := s.ec2.ModifyRouteServer(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			strings.TrimSpace(r.Form.Get("PersistRoutes")),
			persistRoutesDuration,
			snsEnabled,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyRouteServerResponse{
			XMLName:     xml.Name{Local: "ModifyRouteServerResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			RouteServer: ec2RouteServerItemFrom(routeServer),
		})
		return true
	case "GetRouteServerRoutingDatabase":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		areRoutesPersisted, routes, nextToken, err := s.ec2.GetRouteServerRoutingDatabase(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetRouteServerRoutingDatabaseResponse{
			XMLName:            xml.Name{Local: "GetRouteServerRoutingDatabaseResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			AreRoutesPersisted: &areRoutesPersisted,
			RouteSet: ec2RouteServerRouteSet{
				Items: ec2RouteServerRouteItems(routes),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2RouteServerAssociationItems(in []ec2svc.RouteServerAssociation) []ec2RouteServerAssociationItem {
	out := make([]ec2RouteServerAssociationItem, 0, len(in))
	for _, association := range in {
		out = append(out, ec2RouteServerAssociationItemFrom(association))
	}
	return out
}

func ec2RouteServerAssociationItemFrom(in ec2svc.RouteServerAssociation) ec2RouteServerAssociationItem {
	return ec2RouteServerAssociationItem{
		RouteServerID: in.RouteServerID,
		State:         in.State,
		VpcID:         in.VpcID,
	}
}

func ec2RouteServerPropagationItems(in []ec2svc.RouteServerPropagation) []ec2RouteServerPropagationItem {
	out := make([]ec2RouteServerPropagationItem, 0, len(in))
	for _, propagation := range in {
		out = append(out, ec2RouteServerPropagationItemFrom(propagation))
	}
	return out
}

func ec2RouteServerPropagationItemFrom(in ec2svc.RouteServerPropagation) ec2RouteServerPropagationItem {
	return ec2RouteServerPropagationItem{
		RouteServerID: in.RouteServerID,
		RouteTableID:  in.RouteTableID,
		State:         in.State,
	}
}

func ec2RouteServerRouteItems(in []ec2svc.RouteServerRoute) []ec2RouteServerRouteItem {
	out := make([]ec2RouteServerRouteItem, 0, len(in))
	for _, route := range in {
		out = append(out, ec2RouteServerRouteItemFrom(route))
	}
	return out
}

func ec2RouteServerRouteItemFrom(in ec2svc.RouteServerRoute) ec2RouteServerRouteItem {
	out := ec2RouteServerRouteItem{
		AsPathSet: ec2StringSet{Items: append([]string(nil), in.AsPaths...)},
		Med:       in.Med,
		NextHopIP: in.NextHopIP,
		Prefix:    in.Prefix,
		RouteInstallationDetailSet: ec2RouteServerRouteInstallationDetailSet{
			Items: ec2RouteServerRouteInstallationDetailItems(in.RouteInstallationDetails),
		},
		RouteServerEndpointID: in.RouteServerEndpointID,
		RouteServerPeerID:     in.RouteServerPeerID,
		RouteStatus:           in.RouteStatus,
	}
	return out
}

func ec2RouteServerRouteInstallationDetailItems(in []ec2svc.RouteServerRouteInstallationDetail) []ec2RouteServerRouteInstallationDetailItem {
	out := make([]ec2RouteServerRouteInstallationDetailItem, 0, len(in))
	for _, detail := range in {
		out = append(out, ec2RouteServerRouteInstallationDetailItem{
			RouteInstallationStatus:       detail.RouteInstallationStatus,
			RouteInstallationStatusReason: detail.RouteInstallationStatusReason,
			RouteTableID:                  detail.RouteTableID,
		})
	}
	return out
}

type ec2AssociateRouteServerResponse struct {
	XMLName                xml.Name                      `xml:"AssociateRouteServerResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	RouteServerAssociation ec2RouteServerAssociationItem `xml:"routeServerAssociation"`
}

type ec2DisassociateRouteServerResponse struct {
	XMLName                xml.Name                      `xml:"DisassociateRouteServerResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	RouteServerAssociation ec2RouteServerAssociationItem `xml:"routeServerAssociation"`
}

type ec2GetRouteServerAssociationsResponse struct {
	XMLName                   xml.Name                     `xml:"GetRouteServerAssociationsResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	RouteServerAssociationSet ec2RouteServerAssociationSet `xml:"routeServerAssociationSet"`
}

type ec2RouteServerAssociationSet struct {
	Items []ec2RouteServerAssociationItem `xml:"item"`
}

type ec2RouteServerAssociationItem struct {
	RouteServerID string `xml:"routeServerId,omitempty"`
	State         string `xml:"state,omitempty"`
	VpcID         string `xml:"vpcId,omitempty"`
}

type ec2EnableRouteServerPropagationResponse struct {
	XMLName                xml.Name                      `xml:"EnableRouteServerPropagationResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	RouteServerPropagation ec2RouteServerPropagationItem `xml:"routeServerPropagation"`
}

type ec2DisableRouteServerPropagationResponse struct {
	XMLName                xml.Name                      `xml:"DisableRouteServerPropagationResponse"`
	Xmlns                  string                        `xml:"xmlns,attr"`
	RequestID              string                        `xml:"requestId"`
	RouteServerPropagation ec2RouteServerPropagationItem `xml:"routeServerPropagation"`
}

type ec2GetRouteServerPropagationsResponse struct {
	XMLName                   xml.Name                     `xml:"GetRouteServerPropagationsResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	RouteServerPropagationSet ec2RouteServerPropagationSet `xml:"routeServerPropagationSet"`
}

type ec2RouteServerPropagationSet struct {
	Items []ec2RouteServerPropagationItem `xml:"item"`
}

type ec2RouteServerPropagationItem struct {
	RouteServerID string `xml:"routeServerId,omitempty"`
	RouteTableID  string `xml:"routeTableId,omitempty"`
	State         string `xml:"state,omitempty"`
}

type ec2ModifyRouteServerResponse struct {
	XMLName     xml.Name           `xml:"ModifyRouteServerResponse"`
	Xmlns       string             `xml:"xmlns,attr"`
	RequestID   string             `xml:"requestId"`
	RouteServer ec2RouteServerItem `xml:"routeServer"`
}

type ec2GetRouteServerRoutingDatabaseResponse struct {
	XMLName            xml.Name               `xml:"GetRouteServerRoutingDatabaseResponse"`
	Xmlns              string                 `xml:"xmlns,attr"`
	RequestID          string                 `xml:"requestId"`
	AreRoutesPersisted *bool                  `xml:"areRoutesPersisted,omitempty"`
	NextToken          string                 `xml:"nextToken,omitempty"`
	RouteSet           ec2RouteServerRouteSet `xml:"routeSet"`
}

type ec2RouteServerRouteSet struct {
	Items []ec2RouteServerRouteItem `xml:"item"`
}

type ec2RouteServerRouteItem struct {
	AsPathSet                  ec2StringSet                             `xml:"asPathSet"`
	Med                        *int32                                   `xml:"med,omitempty"`
	NextHopIP                  string                                   `xml:"nextHopIp,omitempty"`
	Prefix                     string                                   `xml:"prefix,omitempty"`
	RouteInstallationDetailSet ec2RouteServerRouteInstallationDetailSet `xml:"routeInstallationDetailSet"`
	RouteServerEndpointID      string                                   `xml:"routeServerEndpointId,omitempty"`
	RouteServerPeerID          string                                   `xml:"routeServerPeerId,omitempty"`
	RouteStatus                string                                   `xml:"routeStatus,omitempty"`
}

type ec2RouteServerRouteInstallationDetailSet struct {
	Items []ec2RouteServerRouteInstallationDetailItem `xml:"item"`
}

type ec2RouteServerRouteInstallationDetailItem struct {
	RouteInstallationStatus       string  `xml:"routeInstallationStatus,omitempty"`
	RouteInstallationStatusReason *string `xml:"routeInstallationStatusReason,omitempty"`
	RouteTableID                  string  `xml:"routeTableId,omitempty"`
}
