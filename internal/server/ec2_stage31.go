package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage31Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "DisassociateTransitGatewayMulticastDomain":
		associations, err := s.ec2.DisassociateTransitGatewayMulticastDomain(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			parseEC2Members(r.Form, "SubnetIds."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateTransitGatewayMulticastDomainResponse{
			XMLName:      xml.Name{Local: "DisassociateTransitGatewayMulticastDomainResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Associations: ec2TransitGatewayMulticastDomainAssociationsItemFrom(associations),
		})
		return true
	case "DisassociateTransitGatewayPolicyTable":
		association, err := s.ec2.DisassociateTransitGatewayPolicyTable(
			strings.TrimSpace(r.Form.Get("TransitGatewayPolicyTableId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateTransitGatewayPolicyTableResponse{
			XMLName:     xml.Name{Local: "DisassociateTransitGatewayPolicyTableResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2TransitGatewayPolicyTableAssociationItemFrom(association),
		})
		return true
	case "DisassociateTransitGatewayRouteTable":
		association, err := s.ec2.DisassociateTransitGatewayRouteTable(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateTransitGatewayRouteTableResponse{
			XMLName:     xml.Name{Local: "DisassociateTransitGatewayRouteTableResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2TransitGatewayAssociationItemFrom(association),
		})
		return true
	default:
		return false
	}
}

type ec2DisassociateTransitGatewayMulticastDomainResponse struct {
	XMLName      xml.Name                                         `xml:"DisassociateTransitGatewayMulticastDomainResponse"`
	Xmlns        string                                           `xml:"xmlns,attr"`
	RequestID    string                                           `xml:"requestId"`
	Associations ec2TransitGatewayMulticastDomainAssociationsItem `xml:"associations"`
}

type ec2DisassociateTransitGatewayPolicyTableResponse struct {
	XMLName     xml.Name                                    `xml:"DisassociateTransitGatewayPolicyTableResponse"`
	Xmlns       string                                      `xml:"xmlns,attr"`
	RequestID   string                                      `xml:"requestId"`
	Association ec2TransitGatewayPolicyTableAssociationItem `xml:"association"`
}

type ec2DisassociateTransitGatewayRouteTableResponse struct {
	XMLName     xml.Name                         `xml:"DisassociateTransitGatewayRouteTableResponse"`
	Xmlns       string                           `xml:"xmlns,attr"`
	RequestID   string                           `xml:"requestId"`
	Association ec2TransitGatewayAssociationItem `xml:"association"`
}
