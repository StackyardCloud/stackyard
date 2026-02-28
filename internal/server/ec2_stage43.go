package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage43Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyTransitGateway":
		amazonSideASN, ok := parseEC2OptionalInt64(r.Form.Get("Options.AmazonSideAsn"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		transitGateway, err := s.ec2.ModifyTransitGateway(
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			ec2OptionalStringPointerFromForm(r.Form, "Description"),
			ec2svc.ModifyTransitGatewayOptionsInput{
				AddTransitGatewayCidrBlocks:     parseEC2MembersOrItemList(r.Form, "Options.AddTransitGatewayCidrBlocks"),
				AmazonSideASN:                   amazonSideASN,
				AssociationDefaultRouteTableID:  parseEC2OptionalString(r.Form.Get("Options.AssociationDefaultRouteTableId")),
				AutoAcceptSharedAttachments:     parseEC2OptionalString(r.Form.Get("Options.AutoAcceptSharedAttachments")),
				DefaultRouteTableAssociation:    parseEC2OptionalString(r.Form.Get("Options.DefaultRouteTableAssociation")),
				DefaultRouteTablePropagation:    parseEC2OptionalString(r.Form.Get("Options.DefaultRouteTablePropagation")),
				DnsSupport:                      parseEC2OptionalString(r.Form.Get("Options.DnsSupport")),
				PropagationDefaultRouteTableID:  parseEC2OptionalString(r.Form.Get("Options.PropagationDefaultRouteTableId")),
				RemoveTransitGatewayCidrBlocks:  parseEC2MembersOrItemList(r.Form, "Options.RemoveTransitGatewayCidrBlocks"),
				SecurityGroupReferencingSupport: parseEC2OptionalString(r.Form.Get("Options.SecurityGroupReferencingSupport")),
				VpnEcmpSupport:                  parseEC2OptionalString(r.Form.Get("Options.VpnEcmpSupport")),
			},
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyTransitGatewayResponse{
			XMLName:        xml.Name{Local: "ModifyTransitGatewayResponse"},
			Xmlns:          ec2Namespace,
			RequestID:      "stackyard-request",
			TransitGateway: ec2TransitGatewayItemFrom(transitGateway),
		})
		return true
	case "ModifyTransitGatewayVpcAttachment":
		transitGatewayVpcAttachment, err := s.ec2.ModifyTransitGatewayVpcAttachment(
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			parseEC2MembersOrItemList(r.Form, "AddSubnetIds"),
			parseEC2MembersOrItemList(r.Form, "RemoveSubnetIds"),
			ec2svc.TransitGatewayVpcAttachmentOptionsInput{
				ApplianceModeSupport:            parseEC2OptionalString(r.Form.Get("Options.ApplianceModeSupport")),
				DnsSupport:                      parseEC2OptionalString(r.Form.Get("Options.DnsSupport")),
				Ipv6Support:                     parseEC2OptionalString(r.Form.Get("Options.Ipv6Support")),
				SecurityGroupReferencingSupport: parseEC2OptionalString(r.Form.Get("Options.SecurityGroupReferencingSupport")),
			},
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyTransitGatewayVpcAttachmentResponse{
			XMLName:                     xml.Name{Local: "ModifyTransitGatewayVpcAttachmentResponse"},
			Xmlns:                       ec2Namespace,
			RequestID:                   "stackyard-request",
			TransitGatewayVpcAttachment: ec2TransitGatewayVpcAttachmentItemFrom(transitGatewayVpcAttachment),
		})
		return true
	case "ModifyTransitGatewayPrefixListReference":
		blackhole, hasBlackhole, ok := ec2OptionalBoolFromForm(r.Form, "Blackhole")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasBlackhole {
			blackhole = nil
		}
		hasTransitGatewayAttachmentID := hasEC2Field(r.Form, "TransitGatewayAttachmentId")
		transitGatewayAttachmentID := ec2OptionalStringPointerFromForm(r.Form, "TransitGatewayAttachmentId")

		reference, err := s.ec2.ModifyTransitGatewayPrefixListReference(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("PrefixListId")),
			blackhole,
			transitGatewayAttachmentID,
			hasTransitGatewayAttachmentID,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyTransitGatewayPrefixListReferenceResponse{
			XMLName:                           xml.Name{Local: "ModifyTransitGatewayPrefixListReferenceResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			TransitGatewayPrefixListReference: ec2TransitGatewayPrefixListReferenceItemFrom(reference),
		})
		return true
	default:
		return false
	}
}

type ec2ModifyTransitGatewayResponse struct {
	XMLName        xml.Name              `xml:"ModifyTransitGatewayResponse"`
	Xmlns          string                `xml:"xmlns,attr"`
	RequestID      string                `xml:"requestId"`
	TransitGateway ec2TransitGatewayItem `xml:"transitGateway"`
}

type ec2ModifyTransitGatewayVpcAttachmentResponse struct {
	XMLName                     xml.Name                           `xml:"ModifyTransitGatewayVpcAttachmentResponse"`
	Xmlns                       string                             `xml:"xmlns,attr"`
	RequestID                   string                             `xml:"requestId"`
	TransitGatewayVpcAttachment ec2TransitGatewayVpcAttachmentItem `xml:"transitGatewayVpcAttachment"`
}

type ec2ModifyTransitGatewayPrefixListReferenceResponse struct {
	XMLName                           xml.Name                                 `xml:"ModifyTransitGatewayPrefixListReferenceResponse"`
	Xmlns                             string                                   `xml:"xmlns,attr"`
	RequestID                         string                                   `xml:"requestId"`
	TransitGatewayPrefixListReference ec2TransitGatewayPrefixListReferenceItem `xml:"transitGatewayPrefixListReference"`
}
