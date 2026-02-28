package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage40Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateTransitGatewayRoute":
		route, err := s.ec2.CreateTransitGatewayRoute(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			ec2OptionalBoolPointerFromForm(r.Form.Get("Blackhole")),
			parseEC2OptionalString(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateTransitGatewayRouteResponse{
			XMLName:   xml.Name{Local: "CreateTransitGatewayRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Route:     ec2TransitGatewayRouteItemFrom(route),
		})
		return true
	case "DeleteTransitGatewayRoute":
		route, err := s.ec2.DeleteTransitGatewayRoute(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteTransitGatewayRouteResponse{
			XMLName:   xml.Name{Local: "DeleteTransitGatewayRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Route:     ec2TransitGatewayRouteItemFrom(route),
		})
		return true
	case "ReplaceTransitGatewayRoute":
		route, err := s.ec2.ReplaceTransitGatewayRoute(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			ec2OptionalBoolPointerFromForm(r.Form.Get("Blackhole")),
			parseEC2OptionalString(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ReplaceTransitGatewayRouteResponse{
			XMLName:   xml.Name{Local: "ReplaceTransitGatewayRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Route:     ec2TransitGatewayRouteItemFrom(route),
		})
		return true
	case "SearchTransitGatewayRoutes":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		routes, additionalRoutesAvailable, err := s.ec2.SearchTransitGatewayRoutes(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			parseEC2Filters(r.Form),
			maxResults,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2SearchTransitGatewayRoutesResponse{
			XMLName:   xml.Name{Local: "SearchTransitGatewayRoutesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			RouteSet:  ec2TransitGatewayRouteSet{Items: ec2TransitGatewayRouteItems(routes)},
		}
		if additionalRoutesAvailable != nil {
			response.AdditionalRoutesAvailable = *additionalRoutesAvailable
		}
		respondEC2XML(w, response)
		return true
	case "ExportTransitGatewayRoutes":
		s3Location, err := s.ec2.ExportTransitGatewayRoutes(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("S3Bucket")),
			parseEC2Filters(r.Form),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ExportTransitGatewayRoutesResponse{
			XMLName:    xml.Name{Local: "ExportTransitGatewayRoutesResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			S3Location: s3Location,
		})
		return true
	default:
		return false
	}
}

func ec2OptionalBoolPointerFromForm(value string) *bool {
	resolved, has := parseEC2OptionalBool(value)
	if !has {
		return nil
	}
	return &resolved
}

func ec2TransitGatewayRouteItems(in []ec2svc.TransitGatewayRoute) []ec2TransitGatewayRouteItem {
	out := make([]ec2TransitGatewayRouteItem, 0, len(in))
	for _, route := range in {
		out = append(out, ec2TransitGatewayRouteItemFrom(route))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DestinationCidrBlock < out[j].DestinationCidrBlock })
	return out
}

func ec2TransitGatewayRouteItemFrom(in ec2svc.TransitGatewayRoute) ec2TransitGatewayRouteItem {
	attachments := make([]ec2TransitGatewayRouteAttachmentItem, 0, len(in.TransitGatewayAttachments))
	for _, attachment := range in.TransitGatewayAttachments {
		attachments = append(attachments, ec2TransitGatewayRouteAttachmentItem{
			ResourceID:                 attachment.ResourceID,
			ResourceType:               attachment.ResourceType,
			TransitGatewayAttachmentID: attachment.TransitGatewayAttachmentID,
		})
	}
	sort.Slice(attachments, func(i, j int) bool {
		return attachments[i].TransitGatewayAttachmentID < attachments[j].TransitGatewayAttachmentID
	})

	return ec2TransitGatewayRouteItem{
		DestinationCidrBlock:                   in.DestinationCidrBlock,
		PrefixListID:                           in.PrefixListID,
		State:                                  in.State,
		TransitGatewayAttachments:              ec2TransitGatewayRouteAttachmentSet{Items: attachments},
		TransitGatewayRouteTableAnnouncementID: in.TransitGatewayRouteTableAnnouncementID,
		Type:                                   in.Type,
	}
}

type ec2CreateTransitGatewayRouteResponse struct {
	XMLName   xml.Name                   `xml:"CreateTransitGatewayRouteResponse"`
	Xmlns     string                     `xml:"xmlns,attr"`
	RequestID string                     `xml:"requestId"`
	Route     ec2TransitGatewayRouteItem `xml:"route"`
}

type ec2DeleteTransitGatewayRouteResponse struct {
	XMLName   xml.Name                   `xml:"DeleteTransitGatewayRouteResponse"`
	Xmlns     string                     `xml:"xmlns,attr"`
	RequestID string                     `xml:"requestId"`
	Route     ec2TransitGatewayRouteItem `xml:"route"`
}

type ec2ReplaceTransitGatewayRouteResponse struct {
	XMLName   xml.Name                   `xml:"ReplaceTransitGatewayRouteResponse"`
	Xmlns     string                     `xml:"xmlns,attr"`
	RequestID string                     `xml:"requestId"`
	Route     ec2TransitGatewayRouteItem `xml:"route"`
}

type ec2SearchTransitGatewayRoutesResponse struct {
	XMLName                   xml.Name                  `xml:"SearchTransitGatewayRoutesResponse"`
	Xmlns                     string                    `xml:"xmlns,attr"`
	RequestID                 string                    `xml:"requestId"`
	AdditionalRoutesAvailable bool                      `xml:"additionalRoutesAvailable"`
	RouteSet                  ec2TransitGatewayRouteSet `xml:"routeSet"`
}

type ec2ExportTransitGatewayRoutesResponse struct {
	XMLName    xml.Name `xml:"ExportTransitGatewayRoutesResponse"`
	Xmlns      string   `xml:"xmlns,attr"`
	RequestID  string   `xml:"requestId"`
	S3Location string   `xml:"s3Location,omitempty"`
}

type ec2TransitGatewayRouteSet struct {
	Items []ec2TransitGatewayRouteItem `xml:"item"`
}

type ec2TransitGatewayRouteItem struct {
	DestinationCidrBlock                   string                              `xml:"destinationCidrBlock,omitempty"`
	PrefixListID                           string                              `xml:"prefixListId,omitempty"`
	State                                  string                              `xml:"state,omitempty"`
	TransitGatewayAttachments              ec2TransitGatewayRouteAttachmentSet `xml:"transitGatewayAttachments"`
	TransitGatewayRouteTableAnnouncementID string                              `xml:"transitGatewayRouteTableAnnouncementId,omitempty"`
	Type                                   string                              `xml:"type,omitempty"`
}

type ec2TransitGatewayRouteAttachmentSet struct {
	Items []ec2TransitGatewayRouteAttachmentItem `xml:"item"`
}

type ec2TransitGatewayRouteAttachmentItem struct {
	ResourceID                 string `xml:"resourceId,omitempty"`
	ResourceType               string `xml:"resourceType,omitempty"`
	TransitGatewayAttachmentID string `xml:"transitGatewayAttachmentId,omitempty"`
}
