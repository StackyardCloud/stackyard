package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage6Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateNatGateway":
		gateway, err := s.ec2.CreateNatGateway(
			strings.TrimSpace(r.Form.Get("SubnetId")),
			strings.TrimSpace(r.Form.Get("AllocationId")),
			strings.TrimSpace(r.Form.Get("ConnectivityType")),
			parseEC2TagSpecificationsForResource(r.Form, "natgateway"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateNatGatewayResponse{
			XMLName:    xml.Name{Local: "CreateNatGatewayResponse"},
			Xmlns:      ec2Namespace,
			RequestID:  "stackyard-request",
			NatGateway: ec2NatGatewayItemFrom(gateway),
		})
		return true
	case "DescribeNatGateways":
		gateways := s.ec2.DescribeNatGateways(
			parseEC2Members(r.Form, "NatGatewayId."),
			parseEC2FilterValues(r.Form, "vpc-id"),
			parseEC2FilterValues(r.Form, "subnet-id"),
		)
		respondEC2XML(w, ec2DescribeNatGatewaysResponse{
			XMLName:   xml.Name{Local: "DescribeNatGatewaysResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			NatGatewaySet: ec2NatGatewaySet{
				Items: ec2NatGatewayItems(gateways),
			},
		})
		return true
	case "DeleteNatGateway":
		gatewayID := strings.TrimSpace(r.Form.Get("NatGatewayId"))
		if err := s.ec2.DeleteNatGateway(gatewayID); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteNatGatewayResponse{
			XMLName:      xml.Name{Local: "DeleteNatGatewayResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			NatGatewayID: gatewayID,
		})
		return true
	case "CreateVpcPeeringConnection":
		connection, err := s.ec2.CreateVpcPeeringConnection(
			strings.TrimSpace(r.Form.Get("VpcId")),
			strings.TrimSpace(r.Form.Get("PeerVpcId")),
			parseEC2TagSpecificationsForResource(r.Form, "vpc-peering-connection"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVpcPeeringConnectionResponse{
			XMLName:              xml.Name{Local: "CreateVpcPeeringConnectionResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			VpcPeeringConnection: ec2VpcPeeringConnectionItemFrom(connection),
		})
		return true
	case "DescribeVpcPeeringConnections":
		connections := s.ec2.DescribeVpcPeeringConnections(
			parseEC2Members(r.Form, "VpcPeeringConnectionId."),
			parseEC2FilterValues(r.Form, "requester-vpc-info.vpc-id"),
			parseEC2FilterValues(r.Form, "accepter-vpc-info.vpc-id"),
			parseEC2FilterValues(r.Form, "status-code"),
		)
		respondEC2XML(w, ec2DescribeVpcPeeringConnectionsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcPeeringConnectionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcPeeringConnectionSet: ec2VpcPeeringConnectionSet{
				Items: ec2VpcPeeringConnectionItems(connections),
			},
		})
		return true
	case "AcceptVpcPeeringConnection":
		connection, err := s.ec2.AcceptVpcPeeringConnection(strings.TrimSpace(r.Form.Get("VpcPeeringConnectionId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AcceptVpcPeeringConnectionResponse{
			XMLName:              xml.Name{Local: "AcceptVpcPeeringConnectionResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			VpcPeeringConnection: ec2VpcPeeringConnectionItemFrom(connection),
		})
		return true
	case "RejectVpcPeeringConnection":
		if _, err := s.ec2.RejectVpcPeeringConnection(strings.TrimSpace(r.Form.Get("VpcPeeringConnectionId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "RejectVpcPeeringConnectionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteVpcPeeringConnection":
		if _, err := s.ec2.DeleteVpcPeeringConnection(strings.TrimSpace(r.Form.Get("VpcPeeringConnectionId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteVpcPeeringConnectionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func ec2NatGatewayItems(in []ec2svc.NatGateway) []ec2NatGatewayItem {
	out := make([]ec2NatGatewayItem, 0, len(in))
	for _, gateway := range in {
		out = append(out, ec2NatGatewayItemFrom(gateway))
	}
	return out
}

func ec2NatGatewayItemFrom(in ec2svc.NatGateway) ec2NatGatewayItem {
	addresses := make([]ec2NatGatewayAddressItem, 0, len(in.Addresses))
	for _, address := range in.Addresses {
		addresses = append(addresses, ec2NatGatewayAddressItem{
			AllocationID:       address.AllocationID,
			AssociationID:      address.AssociationID,
			NetworkInterfaceID: address.NetworkInterfaceID,
			PrivateIP:          address.PrivateIP,
			PublicIP:           address.PublicIP,
			Status:             address.Status,
			IsPrimary:          address.IsPrimary,
		})
	}
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	item := ec2NatGatewayItem{
		NatGatewayID:         in.ID,
		VpcID:                in.VpcID,
		SubnetID:             in.SubnetID,
		State:                in.State,
		ConnectivityType:     in.ConnectivityType,
		CreateTime:           in.CreateTime.Format("2006-01-02T15:04:05Z"),
		NatGatewayAddressSet: ec2NatGatewayAddressSet{Items: addresses},
		TagSet:               ec2TagSet{Items: tags},
	}
	if in.DeleteTime != nil {
		item.DeleteTime = in.DeleteTime.Format("2006-01-02T15:04:05Z")
	}
	return item
}

func ec2VpcPeeringConnectionItems(in []ec2svc.VpcPeeringConnection) []ec2VpcPeeringConnectionItem {
	out := make([]ec2VpcPeeringConnectionItem, 0, len(in))
	for _, connection := range in {
		out = append(out, ec2VpcPeeringConnectionItemFrom(connection))
	}
	return out
}

func ec2VpcPeeringConnectionItemFrom(in ec2svc.VpcPeeringConnection) ec2VpcPeeringConnectionItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2VpcPeeringConnectionItem{
		VpcPeeringConnectionID: in.ID,
		Status: ec2VpcPeeringConnectionStateReason{
			Code:    in.StatusCode,
			Message: in.StatusMessage,
		},
		RequesterVpcInfo: ec2VpcPeeringConnectionVpcInfo{
			VpcID:     in.RequesterVpcID,
			CidrBlock: in.RequesterCidrBlock,
			OwnerID:   ec2svc.DefaultAccountID,
			Region:    ec2svc.DefaultRegion,
			PeeringOptions: ec2PeeringConnectionOptionsItem{
				AllowDNSResolutionFromRemoteVPC:            in.RequesterOptions.AllowDNSResolutionFromRemoteVPC,
				AllowEgressFromLocalClassicLinkToRemoteVPC: in.RequesterOptions.AllowEgressFromLocalClassicLinkToRemoteVPC,
				AllowEgressFromLocalVPCToRemoteClassicLink: in.RequesterOptions.AllowEgressFromLocalVPCToRemoteClassicLink,
			},
		},
		AccepterVpcInfo: ec2VpcPeeringConnectionVpcInfo{
			VpcID:     in.AccepterVpcID,
			CidrBlock: in.AccepterCidrBlock,
			OwnerID:   ec2svc.DefaultAccountID,
			Region:    ec2svc.DefaultRegion,
			PeeringOptions: ec2PeeringConnectionOptionsItem{
				AllowDNSResolutionFromRemoteVPC:            in.AccepterOptions.AllowDNSResolutionFromRemoteVPC,
				AllowEgressFromLocalClassicLinkToRemoteVPC: in.AccepterOptions.AllowEgressFromLocalClassicLinkToRemoteVPC,
				AllowEgressFromLocalVPCToRemoteClassicLink: in.AccepterOptions.AllowEgressFromLocalVPCToRemoteClassicLink,
			},
		},
		TagSet: ec2TagSet{Items: tags},
	}
}

type ec2CreateNatGatewayResponse struct {
	XMLName    xml.Name
	Xmlns      string            `xml:"xmlns,attr"`
	RequestID  string            `xml:"requestId"`
	NatGateway ec2NatGatewayItem `xml:"natGateway"`
}

type ec2DescribeNatGatewaysResponse struct {
	XMLName       xml.Name         `xml:"DescribeNatGatewaysResponse"`
	Xmlns         string           `xml:"xmlns,attr"`
	RequestID     string           `xml:"requestId"`
	NatGatewaySet ec2NatGatewaySet `xml:"natGatewaySet"`
}

type ec2DeleteNatGatewayResponse struct {
	XMLName      xml.Name
	Xmlns        string `xml:"xmlns,attr"`
	RequestID    string `xml:"requestId"`
	NatGatewayID string `xml:"natGatewayId"`
}

type ec2NatGatewaySet struct {
	Items []ec2NatGatewayItem `xml:"item"`
}

type ec2NatGatewayItem struct {
	NatGatewayID         string                  `xml:"natGatewayId"`
	VpcID                string                  `xml:"vpcId"`
	SubnetID             string                  `xml:"subnetId"`
	State                string                  `xml:"state"`
	ConnectivityType     string                  `xml:"connectivityType,omitempty"`
	CreateTime           string                  `xml:"createTime,omitempty"`
	DeleteTime           string                  `xml:"deleteTime,omitempty"`
	NatGatewayAddressSet ec2NatGatewayAddressSet `xml:"natGatewayAddressSet"`
	TagSet               ec2TagSet               `xml:"tagSet"`
}

type ec2NatGatewayAddressSet struct {
	Items []ec2NatGatewayAddressItem `xml:"item"`
}

type ec2NatGatewayAddressItem struct {
	AllocationID       string `xml:"allocationId,omitempty"`
	AssociationID      string `xml:"associationId,omitempty"`
	NetworkInterfaceID string `xml:"networkInterfaceId,omitempty"`
	PrivateIP          string `xml:"privateIp,omitempty"`
	PublicIP           string `xml:"publicIp,omitempty"`
	Status             string `xml:"status,omitempty"`
	IsPrimary          bool   `xml:"isPrimary,omitempty"`
}

type ec2CreateVpcPeeringConnectionResponse struct {
	XMLName              xml.Name
	Xmlns                string                      `xml:"xmlns,attr"`
	RequestID            string                      `xml:"requestId"`
	VpcPeeringConnection ec2VpcPeeringConnectionItem `xml:"vpcPeeringConnection"`
}

type ec2DescribeVpcPeeringConnectionsResponse struct {
	XMLName                 xml.Name                   `xml:"DescribeVpcPeeringConnectionsResponse"`
	Xmlns                   string                     `xml:"xmlns,attr"`
	RequestID               string                     `xml:"requestId"`
	VpcPeeringConnectionSet ec2VpcPeeringConnectionSet `xml:"vpcPeeringConnectionSet"`
}

type ec2AcceptVpcPeeringConnectionResponse struct {
	XMLName              xml.Name
	Xmlns                string                      `xml:"xmlns,attr"`
	RequestID            string                      `xml:"requestId"`
	VpcPeeringConnection ec2VpcPeeringConnectionItem `xml:"vpcPeeringConnection"`
}

type ec2VpcPeeringConnectionSet struct {
	Items []ec2VpcPeeringConnectionItem `xml:"item"`
}

type ec2VpcPeeringConnectionItem struct {
	VpcPeeringConnectionID string                             `xml:"vpcPeeringConnectionId"`
	Status                 ec2VpcPeeringConnectionStateReason `xml:"status"`
	RequesterVpcInfo       ec2VpcPeeringConnectionVpcInfo     `xml:"requesterVpcInfo"`
	AccepterVpcInfo        ec2VpcPeeringConnectionVpcInfo     `xml:"accepterVpcInfo"`
	TagSet                 ec2TagSet                          `xml:"tagSet"`
}

type ec2VpcPeeringConnectionStateReason struct {
	Code    string `xml:"code"`
	Message string `xml:"message,omitempty"`
}

type ec2VpcPeeringConnectionVpcInfo struct {
	VpcID          string                          `xml:"vpcId"`
	OwnerID        string                          `xml:"ownerId,omitempty"`
	Region         string                          `xml:"region,omitempty"`
	CidrBlock      string                          `xml:"cidrBlock,omitempty"`
	PeeringOptions ec2PeeringConnectionOptionsItem `xml:"peeringOptions"`
}

type ec2PeeringConnectionOptionsItem struct {
	AllowDNSResolutionFromRemoteVPC            bool `xml:"allowDnsResolutionFromRemoteVpc"`
	AllowEgressFromLocalClassicLinkToRemoteVPC bool `xml:"allowEgressFromLocalClassicLinkToRemoteVpc"`
	AllowEgressFromLocalVPCToRemoteClassicLink bool `xml:"allowEgressFromLocalVpcToRemoteClassicLink"`
}
