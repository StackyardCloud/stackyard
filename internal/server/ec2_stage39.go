package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage39Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AcceptTransitGatewayMulticastDomainAssociations":
		associations, err := s.ec2.AcceptTransitGatewayMulticastDomainAssociations(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			parseEC2Members(r.Form, "SubnetIds."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AcceptTransitGatewayMulticastDomainAssociationsResponse{
			XMLName:      xml.Name{Local: "AcceptTransitGatewayMulticastDomainAssociationsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Associations: ec2TransitGatewayMulticastDomainAssociationsItemFrom(associations),
		})
		return true
	case "RejectTransitGatewayMulticastDomainAssociations":
		associations, err := s.ec2.RejectTransitGatewayMulticastDomainAssociations(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			parseEC2Members(r.Form, "SubnetIds."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RejectTransitGatewayMulticastDomainAssociationsResponse{
			XMLName:      xml.Name{Local: "RejectTransitGatewayMulticastDomainAssociationsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Associations: ec2TransitGatewayMulticastDomainAssociationsItemFrom(associations),
		})
		return true
	case "AcceptTransitGatewayVpcAttachment":
		attachment, err := s.ec2.AcceptTransitGatewayVpcAttachment(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AcceptTransitGatewayVpcAttachmentResponse{
			XMLName:                     xml.Name{Local: "AcceptTransitGatewayVpcAttachmentResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			TransitGatewayVpcAttachment: ec2TransitGatewayVpcAttachmentItemFrom(attachment),
		})
		return true
	case "RejectTransitGatewayVpcAttachment":
		attachment, err := s.ec2.RejectTransitGatewayVpcAttachment(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RejectTransitGatewayVpcAttachmentResponse{
			XMLName:                     xml.Name{Local: "RejectTransitGatewayVpcAttachmentResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			TransitGatewayVpcAttachment: ec2TransitGatewayVpcAttachmentItemFrom(attachment),
		})
		return true
	case "AcceptTransitGatewayPeeringAttachment":
		attachment, err := s.ec2.AcceptTransitGatewayPeeringAttachment(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AcceptTransitGatewayPeeringAttachmentResponse{
			XMLName:                         xml.Name{Local: "AcceptTransitGatewayPeeringAttachmentResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			TransitGatewayPeeringAttachment: ec2TransitGatewayPeeringAttachmentItemFrom(attachment),
		})
		return true
	case "RejectTransitGatewayPeeringAttachment":
		attachment, err := s.ec2.RejectTransitGatewayPeeringAttachment(strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RejectTransitGatewayPeeringAttachmentResponse{
			XMLName:                         xml.Name{Local: "RejectTransitGatewayPeeringAttachmentResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			TransitGatewayPeeringAttachment: ec2TransitGatewayPeeringAttachmentItemFrom(attachment),
		})
		return true
	case "DescribeTransitGatewayAttachments":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		attachments, nextToken, err := s.ec2.DescribeTransitGatewayAttachments(
			parseEC2MembersOrItemList(r.Form, "TransitGatewayAttachmentIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeTransitGatewayAttachmentsResponse{
			XMLName:                   xml.Name{Local: "DescribeTransitGatewayAttachmentsResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			TransitGatewayAttachments: ec2TransitGatewayAttachmentSet{Items: ec2TransitGatewayAttachmentItems(attachments)},
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

func ec2TransitGatewayAttachmentItemFrom(in ec2svc.TransitGatewayAttachment) ec2TransitGatewayAttachmentItem {
	item := ec2TransitGatewayAttachmentItem{
		CreationTime:               in.CreationTime,
		ResourceID:                 in.ResourceID,
		ResourceOwnerID:            in.ResourceOwnerID,
		ResourceType:               in.ResourceType,
		State:                      in.State,
		TagSet:                     ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		TransitGatewayAttachmentID: in.TransitGatewayAttachmentID,
		TransitGatewayID:           in.TransitGatewayID,
		TransitGatewayOwnerID:      in.TransitGatewayOwnerID,
	}
	if in.Association != nil {
		item.Association = &ec2TransitGatewayAttachmentAssociationItem{
			State:                      in.Association.State,
			TransitGatewayRouteTableID: in.Association.TransitGatewayRouteTableID,
		}
	}
	return item
}

func ec2TransitGatewayAttachmentItems(in []ec2svc.TransitGatewayAttachment) []ec2TransitGatewayAttachmentItem {
	out := make([]ec2TransitGatewayAttachmentItem, 0, len(in))
	for _, attachment := range in {
		out = append(out, ec2TransitGatewayAttachmentItemFrom(attachment))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })
	return out
}

type ec2AcceptTransitGatewayMulticastDomainAssociationsResponse struct {
	XMLName      xml.Name                                         `xml:"AcceptTransitGatewayMulticastDomainAssociationsResponse"`
	Xmlns        string                                           `xml:"xmlns,attr"`
	RequestID    string                                           `xml:"requestId"`
	Associations ec2TransitGatewayMulticastDomainAssociationsItem `xml:"associations"`
}

type ec2RejectTransitGatewayMulticastDomainAssociationsResponse struct {
	XMLName      xml.Name                                         `xml:"RejectTransitGatewayMulticastDomainAssociationsResponse"`
	Xmlns        string                                           `xml:"xmlns,attr"`
	RequestID    string                                           `xml:"requestId"`
	Associations ec2TransitGatewayMulticastDomainAssociationsItem `xml:"associations"`
}

type ec2AcceptTransitGatewayVpcAttachmentResponse struct {
	XMLName                     xml.Name                           `xml:"AcceptTransitGatewayVpcAttachmentResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	TransitGatewayVpcAttachment ec2TransitGatewayVpcAttachmentItem `xml:"transitGatewayVpcAttachment"`
}

type ec2RejectTransitGatewayVpcAttachmentResponse struct {
	XMLName                     xml.Name                           `xml:"RejectTransitGatewayVpcAttachmentResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	TransitGatewayVpcAttachment ec2TransitGatewayVpcAttachmentItem `xml:"transitGatewayVpcAttachment"`
}

type ec2AcceptTransitGatewayPeeringAttachmentResponse struct {
	XMLName                         xml.Name                               `xml:"AcceptTransitGatewayPeeringAttachmentResponse"`
	Xmlns                           string                                 `xml:"xmlns,attr"`
	RequestID                       string                                 `xml:"requestId"`
	TransitGatewayPeeringAttachment ec2TransitGatewayPeeringAttachmentItem `xml:"transitGatewayPeeringAttachment"`
}

type ec2RejectTransitGatewayPeeringAttachmentResponse struct {
	XMLName                         xml.Name                               `xml:"RejectTransitGatewayPeeringAttachmentResponse"`
	Xmlns                           string                                 `xml:"xmlns,attr"`
	RequestID                       string                                 `xml:"requestId"`
	TransitGatewayPeeringAttachment ec2TransitGatewayPeeringAttachmentItem `xml:"transitGatewayPeeringAttachment"`
}

type ec2DescribeTransitGatewayAttachmentsResponse struct {
	XMLName                   xml.Name                       `xml:"DescribeTransitGatewayAttachmentsResponse"`
	Xmlns                     string                         `xml:"xmlns,attr"`
	RequestID                 string                         `xml:"requestId"`
	NextToken                 string                         `xml:"nextToken,omitempty"`
	TransitGatewayAttachments ec2TransitGatewayAttachmentSet `xml:"transitGatewayAttachments"`
}

type ec2TransitGatewayAttachmentSet struct {
	Items []ec2TransitGatewayAttachmentItem `xml:"item"`
}

type ec2TransitGatewayAttachmentItem struct {
	Association                *ec2TransitGatewayAttachmentAssociationItem `xml:"association,omitempty"`
	CreationTime               time.Time                                   `xml:"creationTime,omitempty"`
	ResourceID                 string                                      `xml:"resourceId,omitempty"`
	ResourceOwnerID            string                                      `xml:"resourceOwnerId,omitempty"`
	ResourceType               string                                      `xml:"resourceType,omitempty"`
	State                      string                                      `xml:"state,omitempty"`
	TagSet                     ec2TagSet                                   `xml:"tagSet"`
	TransitGatewayAttachmentID string                                      `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayID           string                                      `xml:"transitGatewayId,omitempty"`
	TransitGatewayOwnerID      string                                      `xml:"transitGatewayOwnerId,omitempty"`
}

type ec2TransitGatewayAttachmentAssociationItem struct {
	State                      string `xml:"state,omitempty"`
	TransitGatewayRouteTableID string `xml:"transitGatewayRouteTableId,omitempty"`
}
