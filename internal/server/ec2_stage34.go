package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage34Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "GetTransitGatewayMulticastDomainAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.GetTransitGatewayMulticastDomainAssociations(
			r.Form.Get("TransitGatewayMulticastDomainId"),
			parseEC2FilterValues(r.Form, "resource-id"),
			parseEC2FilterValues(r.Form, "resource-type"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "subnet-id"),
			parseEC2FilterValues(r.Form, "transit-gateway-attachment-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetTransitGatewayMulticastDomainAssociationsResponse{
			XMLName:                     xml.Name{Local: "GetTransitGatewayMulticastDomainAssociationsResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			MulticastDomainAssociations: ec2TransitGatewayMulticastDomainAssociationSet{Items: ec2TransitGatewayMulticastDomainAssociationItems(associations)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "GetTransitGatewayPolicyTableAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.GetTransitGatewayPolicyTableAssociations(
			r.Form.Get("TransitGatewayPolicyTableId"),
			parseEC2FilterValues(r.Form, "resource-id"),
			parseEC2FilterValues(r.Form, "resource-type"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "transit-gateway-attachment-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetTransitGatewayPolicyTableAssociationsResponse{
			XMLName:      xml.Name{Local: "GetTransitGatewayPolicyTableAssociationsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Associations: ec2TransitGatewayPolicyTableAssociationSet{Items: ec2TransitGatewayPolicyTableAssociationItems(associations)},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "GetTransitGatewayRouteTableAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.GetTransitGatewayRouteTableAssociations(
			r.Form.Get("TransitGatewayRouteTableId"),
			parseEC2FilterValues(r.Form, "resource-id"),
			parseEC2FilterValues(r.Form, "resource-type"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "transit-gateway-attachment-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2GetTransitGatewayRouteTableAssociationsResponse{
			XMLName:      xml.Name{Local: "GetTransitGatewayRouteTableAssociationsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Associations: ec2TransitGatewayRouteTableAssociationGetSet{Items: ec2TransitGatewayRouteTableAssociationGetItems(associations)},
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

func ec2TransitGatewayMulticastDomainAssociationItems(in []ec2svc.TransitGatewayMulticastDomainAssociation) []ec2TransitGatewayMulticastDomainAssociationItem {
	out := make([]ec2TransitGatewayMulticastDomainAssociationItem, 0, len(in))
	for _, association := range in {
		out = append(out, ec2TransitGatewayMulticastDomainAssociationItem{
			ResourceID:      association.ResourceID,
			ResourceOwnerID: association.ResourceOwnerID,
			ResourceType:    association.ResourceType,
			Subnet: ec2SubnetAssociationItem{
				State:    association.Subnet.State,
				SubnetID: association.Subnet.SubnetID,
			},
			TransitGatewayAttachmentID: association.TransitGatewayAttachmentID,
		})
	}
	return out
}

func ec2TransitGatewayPolicyTableAssociationItems(in []ec2svc.TransitGatewayPolicyTableAssociation) []ec2TransitGatewayPolicyTableAssociationItem {
	out := make([]ec2TransitGatewayPolicyTableAssociationItem, 0, len(in))
	for _, association := range in {
		out = append(out, ec2TransitGatewayPolicyTableAssociationItemFrom(association))
	}
	return out
}

func ec2TransitGatewayRouteTableAssociationGetItems(in []ec2svc.TransitGatewayRouteTableAssociation) []ec2TransitGatewayRouteTableAssociationGetItem {
	out := make([]ec2TransitGatewayRouteTableAssociationGetItem, 0, len(in))
	for _, association := range in {
		out = append(out, ec2TransitGatewayRouteTableAssociationGetItem{
			ResourceID:                 association.ResourceID,
			ResourceType:               association.ResourceType,
			State:                      association.State,
			TransitGatewayAttachmentID: association.TransitGatewayAttachmentID,
		})
	}
	return out
}

type ec2GetTransitGatewayMulticastDomainAssociationsResponse struct {
	XMLName                     xml.Name                                       `xml:"GetTransitGatewayMulticastDomainAssociationsResponse"`
	Xmlns                       string                                         `xml:"xmlns,attr"`
	RequestID                   string                                         `xml:"requestId"`
	MulticastDomainAssociations ec2TransitGatewayMulticastDomainAssociationSet `xml:"multicastDomainAssociations"`
	NextToken                   string                                         `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayMulticastDomainAssociationSet struct {
	Items []ec2TransitGatewayMulticastDomainAssociationItem `xml:"item"`
}

type ec2TransitGatewayMulticastDomainAssociationItem struct {
	ResourceID                 string                   `xml:"resourceId,omitempty"`
	ResourceOwnerID            string                   `xml:"resourceOwnerId,omitempty"`
	ResourceType               string                   `xml:"resourceType,omitempty"`
	Subnet                     ec2SubnetAssociationItem `xml:"subnet"`
	TransitGatewayAttachmentID string                   `xml:"transitGatewayAttachmentId,omitempty"`
}

type ec2GetTransitGatewayPolicyTableAssociationsResponse struct {
	XMLName      xml.Name                                   `xml:"GetTransitGatewayPolicyTableAssociationsResponse"`
	Xmlns        string                                     `xml:"xmlns,attr"`
	RequestID    string                                     `xml:"requestId"`
	Associations ec2TransitGatewayPolicyTableAssociationSet `xml:"associations"`
	NextToken    string                                     `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayPolicyTableAssociationSet struct {
	Items []ec2TransitGatewayPolicyTableAssociationItem `xml:"item"`
}

type ec2GetTransitGatewayRouteTableAssociationsResponse struct {
	XMLName      xml.Name                                     `xml:"GetTransitGatewayRouteTableAssociationsResponse"`
	Xmlns        string                                       `xml:"xmlns,attr"`
	RequestID    string                                       `xml:"requestId"`
	Associations ec2TransitGatewayRouteTableAssociationGetSet `xml:"associations"`
	NextToken    string                                       `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayRouteTableAssociationGetSet struct {
	Items []ec2TransitGatewayRouteTableAssociationGetItem `xml:"item"`
}

type ec2TransitGatewayRouteTableAssociationGetItem struct {
	ResourceID                 string `xml:"resourceId,omitempty"`
	ResourceType               string `xml:"resourceType,omitempty"`
	State                      string `xml:"state,omitempty"`
	TransitGatewayAttachmentID string `xml:"transitGatewayAttachmentId,omitempty"`
}
