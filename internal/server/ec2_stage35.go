package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage35Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "EnableTransitGatewayRouteTablePropagation":
		propagation, err := s.ec2.EnableTransitGatewayRouteTablePropagation(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableAnnouncementId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2EnableTransitGatewayRouteTablePropagationResponse{
			XMLName:     xml.Name{Local: "EnableTransitGatewayRouteTablePropagationResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Propagation: ec2TransitGatewayPropagationItemFrom(propagation),
		})
		return true
	case "DisableTransitGatewayRouteTablePropagation":
		propagation, err := s.ec2.DisableTransitGatewayRouteTablePropagation(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableAnnouncementId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisableTransitGatewayRouteTablePropagationResponse{
			XMLName:     xml.Name{Local: "DisableTransitGatewayRouteTablePropagationResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Propagation: ec2TransitGatewayPropagationItemFrom(propagation),
		})
		return true
	case "GetTransitGatewayAttachmentPropagations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		propagations, nextToken, err := s.ec2.GetTransitGatewayAttachmentPropagations(
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			parseEC2FilterValues(r.Form, "transit-gateway-route-table-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetTransitGatewayAttachmentPropagationsResponse{
			XMLName:                              xml.Name{Local: "GetTransitGatewayAttachmentPropagationsResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			TransitGatewayAttachmentPropagations: ec2TransitGatewayAttachmentPropagationSet{Items: ec2TransitGatewayAttachmentPropagationItems(propagations)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "GetTransitGatewayRouteTablePropagations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		propagations, nextToken, err := s.ec2.GetTransitGatewayRouteTablePropagations(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			parseEC2FilterValues(r.Form, "resource-id"),
			parseEC2FilterValues(r.Form, "resource-type"),
			parseEC2FilterValues(r.Form, "transit-gateway-attachment-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetTransitGatewayRouteTablePropagationsResponse{
			XMLName:                              xml.Name{Local: "GetTransitGatewayRouteTablePropagationsResponse"},
			Xmlns:                                ec2Namespace,
			RequestID:                            "stackyard-request",
			TransitGatewayRouteTablePropagations: ec2TransitGatewayRouteTablePropagationSet{Items: ec2TransitGatewayRouteTablePropagationItems(propagations)},
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

func ec2TransitGatewayPropagationItemFrom(in ec2svc.TransitGatewayPropagation) ec2TransitGatewayPropagationItem {
	return ec2TransitGatewayPropagationItem{
		ResourceID:                             in.ResourceID,
		ResourceType:                           in.ResourceType,
		State:                                  in.State,
		TransitGatewayAttachmentID:             in.TransitGatewayAttachmentID,
		TransitGatewayRouteTableAnnouncementID: in.TransitGatewayRouteTableAnnouncementID,
		TransitGatewayRouteTableID:             in.TransitGatewayRouteTableID,
	}
}

func ec2TransitGatewayAttachmentPropagationItems(in []ec2svc.TransitGatewayAttachmentPropagation) []ec2TransitGatewayAttachmentPropagationItem {
	out := make([]ec2TransitGatewayAttachmentPropagationItem, 0, len(in))
	for _, propagation := range in {
		out = append(out, ec2TransitGatewayAttachmentPropagationItem{
			State:                      propagation.State,
			TransitGatewayRouteTableID: propagation.TransitGatewayRouteTableID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayRouteTableID < out[j].TransitGatewayRouteTableID })
	return out
}

func ec2TransitGatewayRouteTablePropagationItems(in []ec2svc.TransitGatewayRouteTablePropagation) []ec2TransitGatewayRouteTablePropagationItem {
	out := make([]ec2TransitGatewayRouteTablePropagationItem, 0, len(in))
	for _, propagation := range in {
		out = append(out, ec2TransitGatewayRouteTablePropagationItem{
			ResourceID:                             propagation.ResourceID,
			ResourceType:                           propagation.ResourceType,
			State:                                  propagation.State,
			TransitGatewayAttachmentID:             propagation.TransitGatewayAttachmentID,
			TransitGatewayRouteTableAnnouncementID: propagation.TransitGatewayRouteTableAnnouncementID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ResourceID < out[j].ResourceID })
	return out
}

type ec2EnableTransitGatewayRouteTablePropagationResponse struct {
	XMLName     xml.Name                         `xml:"EnableTransitGatewayRouteTablePropagationResponse"`
	Xmlns       string                           `xml:"xmlns,attr"`
	RequestID   string                           `xml:"requestId"`
	Propagation ec2TransitGatewayPropagationItem `xml:"propagation"`
}

type ec2DisableTransitGatewayRouteTablePropagationResponse struct {
	XMLName     xml.Name                         `xml:"DisableTransitGatewayRouteTablePropagationResponse"`
	Xmlns       string                           `xml:"xmlns,attr"`
	RequestID   string                           `xml:"requestId"`
	Propagation ec2TransitGatewayPropagationItem `xml:"propagation"`
}

type ec2TransitGatewayPropagationItem struct {
	ResourceID                             string `xml:"resourceId,omitempty"`
	ResourceType                           string `xml:"resourceType,omitempty"`
	State                                  string `xml:"state,omitempty"`
	TransitGatewayAttachmentID             string `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayRouteTableAnnouncementID string `xml:"transitGatewayRouteTableAnnouncementId,omitempty"`
	TransitGatewayRouteTableID             string `xml:"transitGatewayRouteTableId,omitempty"`
}

type ec2GetTransitGatewayAttachmentPropagationsResponse struct {
	XMLName                              xml.Name                                  `xml:"GetTransitGatewayAttachmentPropagationsResponse"`
	Xmlns                                string                                    `xml:"xmlns,attr"`
	RequestID                            string                                    `xml:"requestId"`
	TransitGatewayAttachmentPropagations ec2TransitGatewayAttachmentPropagationSet `xml:"transitGatewayAttachmentPropagations"`
	NextToken                            string                                    `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayAttachmentPropagationSet struct {
	Items []ec2TransitGatewayAttachmentPropagationItem `xml:"item"`
}

type ec2TransitGatewayAttachmentPropagationItem struct {
	State                      string `xml:"state,omitempty"`
	TransitGatewayRouteTableID string `xml:"transitGatewayRouteTableId,omitempty"`
}

type ec2GetTransitGatewayRouteTablePropagationsResponse struct {
	XMLName                              xml.Name                                  `xml:"GetTransitGatewayRouteTablePropagationsResponse"`
	Xmlns                                string                                    `xml:"xmlns,attr"`
	RequestID                            string                                    `xml:"requestId"`
	TransitGatewayRouteTablePropagations ec2TransitGatewayRouteTablePropagationSet `xml:"transitGatewayRouteTablePropagations"`
	NextToken                            string                                    `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayRouteTablePropagationSet struct {
	Items []ec2TransitGatewayRouteTablePropagationItem `xml:"item"`
}

type ec2TransitGatewayRouteTablePropagationItem struct {
	ResourceID                             string `xml:"resourceId,omitempty"`
	ResourceType                           string `xml:"resourceType,omitempty"`
	State                                  string `xml:"state,omitempty"`
	TransitGatewayAttachmentID             string `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayRouteTableAnnouncementID string `xml:"transitGatewayRouteTableAnnouncementId,omitempty"`
}
