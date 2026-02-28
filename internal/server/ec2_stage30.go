package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage30Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateTransitGatewayMulticastDomain":
		associations, err := s.ec2.AssociateTransitGatewayMulticastDomain(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
			parseEC2Members(r.Form, "SubnetIds."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateTransitGatewayMulticastDomainResponse{
			XMLName:      xml.Name{Local: "AssociateTransitGatewayMulticastDomainResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Associations: ec2TransitGatewayMulticastDomainAssociationsItemFrom(associations),
		})
		return true
	case "AssociateTransitGatewayPolicyTable":
		association, err := s.ec2.AssociateTransitGatewayPolicyTable(
			strings.TrimSpace(r.Form.Get("TransitGatewayPolicyTableId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateTransitGatewayPolicyTableResponse{
			XMLName:     xml.Name{Local: "AssociateTransitGatewayPolicyTableResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2TransitGatewayPolicyTableAssociationItemFrom(association),
		})
		return true
	case "AssociateTransitGatewayRouteTable":
		association, err := s.ec2.AssociateTransitGatewayRouteTable(
			strings.TrimSpace(r.Form.Get("TransitGatewayRouteTableId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayAttachmentId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateTransitGatewayRouteTableResponse{
			XMLName:     xml.Name{Local: "AssociateTransitGatewayRouteTableResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Association: ec2TransitGatewayAssociationItemFrom(association),
		})
		return true
	case "AssociateTrunkInterface":
		vlanID, ok := parseEC2OptionalInt32(r.Form.Get("VlanId"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		greKey, ok := parseEC2OptionalInt32(r.Form.Get("GreKey"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		result, err := s.ec2.AssociateTrunkInterface(
			strings.TrimSpace(r.Form.Get("BranchInterfaceId")),
			strings.TrimSpace(r.Form.Get("TrunkInterfaceId")),
			vlanID,
			greKey,
			strings.TrimSpace(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateTrunkInterfaceResponse{
			XMLName:              xml.Name{Local: "AssociateTrunkInterfaceResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			ClientToken:          result.ClientToken,
			InterfaceAssociation: ec2TrunkInterfaceAssociationItemFrom(result.InterfaceAssociation),
		})
		return true
	case "DisassociateTrunkInterface":
		result, err := s.ec2.DisassociateTrunkInterface(
			strings.TrimSpace(r.Form.Get("AssociationId")),
			strings.TrimSpace(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateTrunkInterfaceResponse{
			XMLName:     xml.Name{Local: "DisassociateTrunkInterfaceResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			ClientToken: result.ClientToken,
			Return:      result.Return,
		})
		return true
	default:
		return false
	}
}

func ec2TransitGatewayMulticastDomainAssociationsItemFrom(in ec2svc.TransitGatewayMulticastDomainAssociations) ec2TransitGatewayMulticastDomainAssociationsItem {
	subnets := make([]ec2SubnetAssociationItem, 0, len(in.Subnets))
	for _, subnet := range in.Subnets {
		subnets = append(subnets, ec2SubnetAssociationItem{State: subnet.State, SubnetID: subnet.SubnetID})
	}
	sort.Slice(subnets, func(i, j int) bool { return subnets[i].SubnetID < subnets[j].SubnetID })

	return ec2TransitGatewayMulticastDomainAssociationsItem{
		ResourceID:                      in.ResourceID,
		ResourceOwnerID:                 in.ResourceOwnerID,
		ResourceType:                    in.ResourceType,
		Subnets:                         ec2SubnetAssociationSet{Items: subnets},
		TransitGatewayAttachmentID:      in.TransitGatewayAttachmentID,
		TransitGatewayMulticastDomainID: in.TransitGatewayMulticastDomainID,
	}
}

func ec2TransitGatewayPolicyTableAssociationItemFrom(in ec2svc.TransitGatewayPolicyTableAssociation) ec2TransitGatewayPolicyTableAssociationItem {
	return ec2TransitGatewayPolicyTableAssociationItem{
		ResourceID:                  in.ResourceID,
		ResourceType:                in.ResourceType,
		State:                       in.State,
		TransitGatewayAttachmentID:  in.TransitGatewayAttachmentID,
		TransitGatewayPolicyTableID: in.TransitGatewayPolicyTableID,
	}
}

func ec2TransitGatewayAssociationItemFrom(in ec2svc.TransitGatewayRouteTableAssociation) ec2TransitGatewayAssociationItem {
	return ec2TransitGatewayAssociationItem{
		ResourceID:                 in.ResourceID,
		ResourceType:               in.ResourceType,
		State:                      in.State,
		TransitGatewayAttachmentID: in.TransitGatewayAttachmentID,
		TransitGatewayRouteTableID: in.TransitGatewayRouteTableID,
	}
}

func ec2TrunkInterfaceAssociationItemFrom(in ec2svc.TrunkInterfaceAssociation) ec2TrunkInterfaceAssociationItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
	return ec2TrunkInterfaceAssociationItem{
		AssociationID:     in.AssociationID,
		BranchInterfaceID: in.BranchInterfaceID,
		GreKey:            in.GreKey,
		InterfaceProtocol: in.InterfaceProtocol,
		TagSet:            ec2TagSet{Items: tags},
		TrunkInterfaceID:  in.TrunkInterfaceID,
		VlanID:            in.VlanID,
	}
}

type ec2AssociateTransitGatewayMulticastDomainResponse struct {
	XMLName      xml.Name                                         `xml:"AssociateTransitGatewayMulticastDomainResponse"`
	Xmlns        string                                           `xml:"xmlns,attr"`
	RequestID    string                                           `xml:"requestId"`
	Associations ec2TransitGatewayMulticastDomainAssociationsItem `xml:"associations"`
}

type ec2TransitGatewayMulticastDomainAssociationsItem struct {
	ResourceID                      string                  `xml:"resourceId,omitempty"`
	ResourceOwnerID                 string                  `xml:"resourceOwnerId,omitempty"`
	ResourceType                    string                  `xml:"resourceType,omitempty"`
	Subnets                         ec2SubnetAssociationSet `xml:"subnets"`
	TransitGatewayAttachmentID      string                  `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayMulticastDomainID string                  `xml:"transitGatewayMulticastDomainId,omitempty"`
}

type ec2SubnetAssociationSet struct {
	Items []ec2SubnetAssociationItem `xml:"item"`
}

type ec2SubnetAssociationItem struct {
	State    string `xml:"state,omitempty"`
	SubnetID string `xml:"subnetId,omitempty"`
}

type ec2AssociateTransitGatewayPolicyTableResponse struct {
	XMLName     xml.Name                                    `xml:"AssociateTransitGatewayPolicyTableResponse"`
	Xmlns       string                                      `xml:"xmlns,attr"`
	RequestID   string                                      `xml:"requestId"`
	Association ec2TransitGatewayPolicyTableAssociationItem `xml:"association"`
}

type ec2TransitGatewayPolicyTableAssociationItem struct {
	ResourceID                  string `xml:"resourceId,omitempty"`
	ResourceType                string `xml:"resourceType,omitempty"`
	State                       string `xml:"state,omitempty"`
	TransitGatewayAttachmentID  string `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayPolicyTableID string `xml:"transitGatewayPolicyTableId,omitempty"`
}

type ec2AssociateTransitGatewayRouteTableResponse struct {
	XMLName     xml.Name                         `xml:"AssociateTransitGatewayRouteTableResponse"`
	Xmlns       string                           `xml:"xmlns,attr"`
	RequestID   string                           `xml:"requestId"`
	Association ec2TransitGatewayAssociationItem `xml:"association"`
}

type ec2TransitGatewayAssociationItem struct {
	ResourceID                 string `xml:"resourceId,omitempty"`
	ResourceType               string `xml:"resourceType,omitempty"`
	State                      string `xml:"state,omitempty"`
	TransitGatewayAttachmentID string `xml:"transitGatewayAttachmentId,omitempty"`
	TransitGatewayRouteTableID string `xml:"transitGatewayRouteTableId,omitempty"`
}

type ec2AssociateTrunkInterfaceResponse struct {
	XMLName              xml.Name                         `xml:"AssociateTrunkInterfaceResponse"`
	Xmlns                string                           `xml:"xmlns,attr"`
	RequestID            string                           `xml:"requestId"`
	ClientToken          string                           `xml:"clientToken,omitempty"`
	InterfaceAssociation ec2TrunkInterfaceAssociationItem `xml:"interfaceAssociation"`
}

type ec2TrunkInterfaceAssociationItem struct {
	AssociationID     string    `xml:"associationId,omitempty"`
	BranchInterfaceID string    `xml:"branchInterfaceId,omitempty"`
	GreKey            *int32    `xml:"greKey,omitempty"`
	InterfaceProtocol string    `xml:"interfaceProtocol,omitempty"`
	TagSet            ec2TagSet `xml:"tagSet"`
	TrunkInterfaceID  string    `xml:"trunkInterfaceId,omitempty"`
	VlanID            *int32    `xml:"vlanId,omitempty"`
}

type ec2DisassociateTrunkInterfaceResponse struct {
	XMLName     xml.Name `xml:"DisassociateTrunkInterfaceResponse"`
	Xmlns       string   `xml:"xmlns,attr"`
	RequestID   string   `xml:"requestId"`
	ClientToken string   `xml:"clientToken,omitempty"`
	Return      bool     `xml:"return"`
}
