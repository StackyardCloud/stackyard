package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage29Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssociateVpcCidrBlock":
		ipv4NetmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("Ipv4NetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ipv6NetmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("Ipv6NetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		amazonProvidedIPv6CidrBlock, hasAmazonProvided, validAmazonProvided := ec2OptionalBoolFromForm(r.Form, "AmazonProvidedIpv6CidrBlock")
		if hasAmazonProvided && !validAmazonProvided {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		amazonProvided := false
		if amazonProvidedIPv6CidrBlock != nil {
			amazonProvided = *amazonProvidedIPv6CidrBlock
		}

		result, err := s.ec2.AssociateVpcCidrBlock(
			strings.TrimSpace(r.Form.Get("VpcId")),
			strings.TrimSpace(r.Form.Get("CidrBlock")),
			strings.TrimSpace(r.Form.Get("Ipv4IpamPoolId")),
			ipv4NetmaskLength,
			amazonProvided,
			strings.TrimSpace(r.Form.Get("Ipv6CidrBlock")),
			strings.TrimSpace(r.Form.Get("Ipv6Pool")),
			strings.TrimSpace(r.Form.Get("Ipv6IpamPoolId")),
			strings.TrimSpace(r.Form.Get("Ipv6CidrBlockNetworkBorderGroup")),
			ipv6NetmaskLength,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2AssociateVpcCidrBlockResponse{
			XMLName:   xml.Name{Local: "AssociateVpcCidrBlockResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcID:     result.VpcID,
		}
		if result.CidrBlockAssociation != nil {
			item := ec2VpcCidrBlockAssociationItemFrom(*result.CidrBlockAssociation)
			response.CidrBlockAssociation = &item
		}
		if result.IPv6CidrAssociation != nil {
			item := ec2VpcIPv6CidrBlockAssociationItemFrom(*result.IPv6CidrAssociation)
			response.IPv6CidrBlockAssociation = &item
		}
		respondEC2XML(w, response)
		return true
	case "DisassociateVpcCidrBlock":
		result, err := s.ec2.DisassociateVpcCidrBlock(strings.TrimSpace(r.Form.Get("AssociationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2DisassociateVpcCidrBlockResponse{
			XMLName:   xml.Name{Local: "DisassociateVpcCidrBlockResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcID:     result.VpcID,
		}
		if result.CidrBlockAssociation != nil {
			item := ec2VpcCidrBlockAssociationItemFrom(*result.CidrBlockAssociation)
			response.CidrBlockAssociation = &item
		}
		if result.IPv6CidrAssociation != nil {
			item := ec2VpcIPv6CidrBlockAssociationItemFrom(*result.IPv6CidrAssociation)
			response.IPv6CidrBlockAssociation = &item
		}
		respondEC2XML(w, response)
		return true
	case "AssociateSubnetCidrBlock":
		ipv6NetmaskLength, ok := parseEC2OptionalInt32(r.Form.Get("Ipv6NetmaskLength"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		result, err := s.ec2.AssociateSubnetCidrBlock(
			strings.TrimSpace(r.Form.Get("SubnetId")),
			strings.TrimSpace(r.Form.Get("Ipv6CidrBlock")),
			strings.TrimSpace(r.Form.Get("Ipv6IpamPoolId")),
			ipv6NetmaskLength,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2AssociateSubnetCidrBlockResponse{
			XMLName:   xml.Name{Local: "AssociateSubnetCidrBlockResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SubnetID:  result.SubnetID,
		}
		if result.IPv6CidrAssociation != nil {
			item := ec2SubnetIPv6CidrBlockAssociationItemFrom(*result.IPv6CidrAssociation)
			response.IPv6CidrBlockAssociation = &item
		}
		respondEC2XML(w, response)
		return true
	case "DisassociateSubnetCidrBlock":
		result, err := s.ec2.DisassociateSubnetCidrBlock(strings.TrimSpace(r.Form.Get("AssociationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2DisassociateSubnetCidrBlockResponse{
			XMLName:   xml.Name{Local: "DisassociateSubnetCidrBlockResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SubnetID:  result.SubnetID,
		}
		if result.IPv6CidrAssociation != nil {
			item := ec2SubnetIPv6CidrBlockAssociationItemFrom(*result.IPv6CidrAssociation)
			response.IPv6CidrBlockAssociation = &item
		}
		respondEC2XML(w, response)
		return true
	case "CreateSubnetCidrReservation":
		reservation, err := s.ec2.CreateSubnetCidrReservation(
			strings.TrimSpace(r.Form.Get("SubnetId")),
			strings.TrimSpace(r.Form.Get("Cidr")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("ReservationType")),
			parseEC2TagSpecificationsForResource(r.Form, "subnet-cidr-reservation"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateSubnetCidrReservationResponse{
			XMLName:               xml.Name{Local: "CreateSubnetCidrReservationResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			SubnetCidrReservation: ec2SubnetCidrReservationItemFrom(reservation),
		})
		return true
	case "DeleteSubnetCidrReservation":
		reservation, err := s.ec2.DeleteSubnetCidrReservation(strings.TrimSpace(r.Form.Get("SubnetCidrReservationId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteSubnetCidrReservationResponse{
			XMLName:                      xml.Name{Local: "DeleteSubnetCidrReservationResponse"},
			Xmlns:                        ec2Namespace,
			RequestID:                    "stackyard-request",
			DeletedSubnetCidrReservation: ec2SubnetCidrReservationItemFrom(reservation),
		})
		return true
	case "GetSubnetCidrReservations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		result, err := s.ec2.GetSubnetCidrReservations(
			strings.TrimSpace(r.Form.Get("SubnetId")),
			parseEC2FilterValues(r.Form, "cidr"),
			parseEC2FilterValues(r.Form, "owner-id"),
			parseEC2FilterValues(r.Form, "reservation-type"),
			parseEC2FilterValues(r.Form, "subnet-cidr-reservation-id"),
			parseEC2FilterValues(r.Form, "subnet-id"),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		response := ec2GetSubnetCidrReservationsResponse{
			XMLName:   xml.Name{Local: "GetSubnetCidrReservationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			SubnetIPv4CidrReservationSet: ec2SubnetCidrReservationSet{
				Items: ec2SubnetCidrReservationItems(result.SubnetIPv4CidrReservations),
			},
			SubnetIPv6CidrReservationSet: ec2SubnetCidrReservationSet{
				Items: ec2SubnetCidrReservationItems(result.SubnetIPv6CidrReservations),
			},
		}
		if result.NextToken != nil {
			response.NextToken = *result.NextToken
		}
		respondEC2XML(w, response)
		return true
	default:
		return false
	}
}

func ec2VpcCidrBlockAssociationItemFrom(association ec2svc.VpcCidrAssociation) ec2VpcCidrBlockAssociationItem {
	return ec2VpcCidrBlockAssociationItem{
		AssociationID: association.AssociationID,
		CidrBlock:     association.CidrBlock,
		CidrBlockState: ec2VpcCidrBlockStateItem{
			State:         association.State,
			StatusMessage: association.StatusMessage,
		},
	}
}

func ec2VpcIPv6CidrBlockAssociationItemFrom(association ec2svc.VpcIPv6CidrAssociation) ec2VpcIPv6CidrBlockAssociationItem {
	return ec2VpcIPv6CidrBlockAssociationItem{
		AssociationID:        association.AssociationID,
		IPSource:             association.IPSource,
		IPv6AddressAttribute: association.IPv6AddressAttribute,
		IPv6CidrBlock:        association.IPv6CidrBlock,
		IPv6CidrBlockState: ec2VpcCidrBlockStateItem{
			State:         association.State,
			StatusMessage: association.StatusMessage,
		},
		IPv6Pool:           association.IPv6Pool,
		NetworkBorderGroup: association.NetworkBorderGroup,
	}
}

func ec2SubnetIPv6CidrBlockAssociationItemFrom(association ec2svc.SubnetIPv6CidrAssociation) ec2SubnetIPv6CidrBlockAssociationItem {
	return ec2SubnetIPv6CidrBlockAssociationItem{
		AssociationID:        association.AssociationID,
		IPSource:             association.IPSource,
		IPv6AddressAttribute: association.IPv6AddressAttribute,
		IPv6CidrBlock:        association.IPv6CidrBlock,
		IPv6CidrBlockState: ec2SubnetCidrBlockStateItem{
			State:         association.State,
			StatusMessage: association.StatusMessage,
		},
	}
}

func ec2SubnetCidrReservationItems(in []ec2svc.SubnetCidrReservation) []ec2SubnetCidrReservationItem {
	out := make([]ec2SubnetCidrReservationItem, 0, len(in))
	for _, reservation := range in {
		out = append(out, ec2SubnetCidrReservationItemFrom(reservation))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SubnetCidrReservationID < out[j].SubnetCidrReservationID })
	return out
}

func ec2SubnetCidrReservationItemFrom(reservation ec2svc.SubnetCidrReservation) ec2SubnetCidrReservationItem {
	tags := make([]ec2TagItem, 0, len(reservation.Tags))
	for key, value := range reservation.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2SubnetCidrReservationItem{
		Cidr:                    reservation.Cidr,
		Description:             reservation.Description,
		OwnerID:                 reservation.OwnerID,
		ReservationType:         reservation.ReservationType,
		SubnetCidrReservationID: reservation.ID,
		SubnetID:                reservation.SubnetID,
		TagSet:                  ec2TagSet{Items: tags},
	}
}

type ec2AssociateVpcCidrBlockResponse struct {
	XMLName                  xml.Name                            `xml:"AssociateVpcCidrBlockResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	CidrBlockAssociation     *ec2VpcCidrBlockAssociationItem     `xml:"cidrBlockAssociation,omitempty"`
	IPv6CidrBlockAssociation *ec2VpcIPv6CidrBlockAssociationItem `xml:"ipv6CidrBlockAssociation,omitempty"`
	VpcID                    string                              `xml:"vpcId,omitempty"`
}

type ec2DisassociateVpcCidrBlockResponse struct {
	XMLName                  xml.Name                            `xml:"DisassociateVpcCidrBlockResponse"`
	Xmlns                    string                              `xml:"xmlns,attr"`
	RequestID                string                              `xml:"requestId"`
	CidrBlockAssociation     *ec2VpcCidrBlockAssociationItem     `xml:"cidrBlockAssociation,omitempty"`
	IPv6CidrBlockAssociation *ec2VpcIPv6CidrBlockAssociationItem `xml:"ipv6CidrBlockAssociation,omitempty"`
	VpcID                    string                              `xml:"vpcId,omitempty"`
}

type ec2VpcCidrBlockAssociationItem struct {
	AssociationID  string                   `xml:"associationId,omitempty"`
	CidrBlock      string                   `xml:"cidrBlock,omitempty"`
	CidrBlockState ec2VpcCidrBlockStateItem `xml:"cidrBlockState"`
}

type ec2VpcIPv6CidrBlockAssociationItem struct {
	AssociationID        string                   `xml:"associationId,omitempty"`
	IPSource             string                   `xml:"ipSource,omitempty"`
	IPv6AddressAttribute string                   `xml:"ipv6AddressAttribute,omitempty"`
	IPv6CidrBlock        string                   `xml:"ipv6CidrBlock,omitempty"`
	IPv6CidrBlockState   ec2VpcCidrBlockStateItem `xml:"ipv6CidrBlockState"`
	IPv6Pool             string                   `xml:"ipv6Pool,omitempty"`
	NetworkBorderGroup   string                   `xml:"networkBorderGroup,omitempty"`
}

type ec2VpcCidrBlockStateItem struct {
	State         string `xml:"state,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}

type ec2AssociateSubnetCidrBlockResponse struct {
	XMLName                  xml.Name                               `xml:"AssociateSubnetCidrBlockResponse"`
	Xmlns                    string                                 `xml:"xmlns,attr"`
	RequestID                string                                 `xml:"requestId"`
	IPv6CidrBlockAssociation *ec2SubnetIPv6CidrBlockAssociationItem `xml:"ipv6CidrBlockAssociation,omitempty"`
	SubnetID                 string                                 `xml:"subnetId,omitempty"`
}

type ec2DisassociateSubnetCidrBlockResponse struct {
	XMLName                  xml.Name                               `xml:"DisassociateSubnetCidrBlockResponse"`
	Xmlns                    string                                 `xml:"xmlns,attr"`
	RequestID                string                                 `xml:"requestId"`
	IPv6CidrBlockAssociation *ec2SubnetIPv6CidrBlockAssociationItem `xml:"ipv6CidrBlockAssociation,omitempty"`
	SubnetID                 string                                 `xml:"subnetId,omitempty"`
}

type ec2SubnetIPv6CidrBlockAssociationItem struct {
	AssociationID        string                      `xml:"associationId,omitempty"`
	IPSource             string                      `xml:"ipSource,omitempty"`
	IPv6AddressAttribute string                      `xml:"ipv6AddressAttribute,omitempty"`
	IPv6CidrBlock        string                      `xml:"ipv6CidrBlock,omitempty"`
	IPv6CidrBlockState   ec2SubnetCidrBlockStateItem `xml:"ipv6CidrBlockState"`
}

type ec2SubnetCidrBlockStateItem struct {
	State         string `xml:"state,omitempty"`
	StatusMessage string `xml:"statusMessage,omitempty"`
}

type ec2CreateSubnetCidrReservationResponse struct {
	XMLName               xml.Name                     `xml:"CreateSubnetCidrReservationResponse"`
	Xmlns                 string                       `xml:"xmlns,attr"`
	RequestID             string                       `xml:"requestId"`
	SubnetCidrReservation ec2SubnetCidrReservationItem `xml:"subnetCidrReservation"`
}

type ec2DeleteSubnetCidrReservationResponse struct {
	XMLName                      xml.Name                     `xml:"DeleteSubnetCidrReservationResponse"`
	Xmlns                        string                       `xml:"xmlns,attr"`
	RequestID                    string                       `xml:"requestId"`
	DeletedSubnetCidrReservation ec2SubnetCidrReservationItem `xml:"deletedSubnetCidrReservation"`
}

type ec2GetSubnetCidrReservationsResponse struct {
	XMLName                      xml.Name                    `xml:"GetSubnetCidrReservationsResponse"`
	Xmlns                        string                      `xml:"xmlns,attr"`
	RequestID                    string                      `xml:"requestId"`
	NextToken                    string                      `xml:"nextToken,omitempty"`
	SubnetIPv4CidrReservationSet ec2SubnetCidrReservationSet `xml:"subnetIpv4CidrReservationSet"`
	SubnetIPv6CidrReservationSet ec2SubnetCidrReservationSet `xml:"subnetIpv6CidrReservationSet"`
}

type ec2SubnetCidrReservationSet struct {
	Items []ec2SubnetCidrReservationItem `xml:"item"`
}

type ec2SubnetCidrReservationItem struct {
	Cidr                    string    `xml:"cidr,omitempty"`
	Description             string    `xml:"description,omitempty"`
	OwnerID                 string    `xml:"ownerId,omitempty"`
	ReservationType         string    `xml:"reservationType,omitempty"`
	SubnetCidrReservationID string    `xml:"subnetCidrReservationId,omitempty"`
	SubnetID                string    `xml:"subnetId,omitempty"`
	TagSet                  ec2TagSet `xml:"tagSet"`
}
