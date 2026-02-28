package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage4Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AllocateAddress":
		address, err := s.ec2.AllocateAddress(
			strings.TrimSpace(r.Form.Get("Address")),
			parseEC2TagSpecificationsForResource(r.Form, "elastic-ip"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AllocateAddressResponse{
			XMLName:            xml.Name{Local: "AllocateAddressResponse"},
			Xmlns:              ec2Namespace,
			RequestID:          "stackyard-request",
			PublicIP:           address.PublicIP,
			Domain:             address.Domain,
			AllocationID:       address.AllocationID,
			NetworkBorderGroup: address.NetworkBorderGroup,
		})
		return true
	case "DescribeAddresses":
		allocationIDs := parseEC2Members(r.Form, "AllocationId.")
		if len(allocationIDs) == 0 {
			allocationIDs = parseEC2FilterValues(r.Form, "allocation-id")
		}
		publicIPs := parseEC2Members(r.Form, "PublicIp.")
		if len(publicIPs) == 0 {
			publicIPs = parseEC2FilterValues(r.Form, "public-ip")
		}
		associationIDs := parseEC2FilterValues(r.Form, "association-id")
		instanceIDs := parseEC2FilterValues(r.Form, "instance-id")

		addresses := s.ec2.DescribeAddresses(allocationIDs, publicIPs, associationIDs, instanceIDs)
		respondEC2XML(w, ec2DescribeAddressesResponse{
			XMLName:      xml.Name{Local: "DescribeAddressesResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			AddressesSet: ec2AddressSet{Items: ec2AddressItems(addresses)},
		})
		return true
	case "AssociateAddress":
		associationID, err := s.ec2.AssociateAddress(
			strings.TrimSpace(r.Form.Get("AllocationId")),
			strings.TrimSpace(r.Form.Get("PublicIp")),
			strings.TrimSpace(r.Form.Get("InstanceId")),
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			strings.TrimSpace(r.Form.Get("PrivateIpAddress")),
			parseEC2Bool(r.Form.Get("AllowReassociation"), true),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateAddressResponse{
			XMLName:       xml.Name{Local: "AssociateAddressResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			AssociationID: associationID,
		})
		return true
	case "DisassociateAddress":
		if err := s.ec2.DisassociateAddress(strings.TrimSpace(r.Form.Get("AssociationId")), strings.TrimSpace(r.Form.Get("PublicIp"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DisassociateAddressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "ReleaseAddress":
		if err := s.ec2.ReleaseAddress(strings.TrimSpace(r.Form.Get("AllocationId")), strings.TrimSpace(r.Form.Get("PublicIp"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ReleaseAddressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "CreateDefaultVpc":
		vpc, err := s.ec2.CreateDefaultVpc()
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateDefaultVpcResponse{
			XMLName:   xml.Name{Local: "CreateDefaultVpcResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Vpc:       ec2VPCItemFrom(vpc),
		})
		return true
	case "CreateDefaultSubnet":
		subnet, err := s.ec2.CreateDefaultSubnet(
			strings.TrimSpace(r.Form.Get("AvailabilityZone")),
			strings.TrimSpace(r.Form.Get("AvailabilityZoneId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateDefaultSubnetResponse{
			XMLName:   xml.Name{Local: "CreateDefaultSubnetResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Subnet:    ec2SubnetItemFrom(subnet),
		})
		return true
	default:
		return false
	}
}

func ec2AddressItems(in []ec2svc.ElasticAddress) []ec2AddressItem {
	out := make([]ec2AddressItem, 0, len(in))
	for _, address := range in {
		tags := make([]ec2TagItem, 0, len(address.Tags))
		for key, value := range address.Tags {
			tags = append(tags, ec2TagItem{Key: key, Value: value})
		}
		sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })
		out = append(out, ec2AddressItem{
			PublicIP:           address.PublicIP,
			AllocationID:       address.AllocationID,
			AssociationID:      address.AssociationID,
			Domain:             address.Domain,
			InstanceID:         address.InstanceID,
			NetworkInterfaceID: address.NetworkInterfaceID,
			PrivateIPAddress:   address.PrivateIPAddress,
			NetworkBorderGroup: address.NetworkBorderGroup,
			TagSet:             ec2TagSet{Items: tags},
		})
	}
	return out
}

type ec2AllocateAddressResponse struct {
	XMLName            xml.Name
	Xmlns              string `xml:"xmlns,attr"`
	RequestID          string `xml:"requestId"`
	PublicIP           string `xml:"publicIp,omitempty"`
	Domain             string `xml:"domain,omitempty"`
	AllocationID       string `xml:"allocationId,omitempty"`
	NetworkBorderGroup string `xml:"networkBorderGroup,omitempty"`
}

type ec2DescribeAddressesResponse struct {
	XMLName      xml.Name
	Xmlns        string        `xml:"xmlns,attr"`
	RequestID    string        `xml:"requestId"`
	AddressesSet ec2AddressSet `xml:"addressesSet"`
}

type ec2AssociateAddressResponse struct {
	XMLName       xml.Name
	Xmlns         string `xml:"xmlns,attr"`
	RequestID     string `xml:"requestId"`
	AssociationID string `xml:"associationId,omitempty"`
}

type ec2AddressSet struct {
	Items []ec2AddressItem `xml:"item"`
}

type ec2AddressItem struct {
	PublicIP           string    `xml:"publicIp,omitempty"`
	AllocationID       string    `xml:"allocationId,omitempty"`
	AssociationID      string    `xml:"associationId,omitempty"`
	Domain             string    `xml:"domain,omitempty"`
	InstanceID         string    `xml:"instanceId,omitempty"`
	NetworkInterfaceID string    `xml:"networkInterfaceId,omitempty"`
	PrivateIPAddress   string    `xml:"privateIpAddress,omitempty"`
	NetworkBorderGroup string    `xml:"networkBorderGroup,omitempty"`
	TagSet             ec2TagSet `xml:"tagSet"`
}

type ec2CreateDefaultVpcResponse struct {
	XMLName   xml.Name
	Xmlns     string     `xml:"xmlns,attr"`
	RequestID string     `xml:"requestId"`
	Vpc       ec2VPCItem `xml:"vpc"`
}

type ec2CreateDefaultSubnetResponse struct {
	XMLName   xml.Name
	Xmlns     string        `xml:"xmlns,attr"`
	RequestID string        `xml:"requestId"`
	Subnet    ec2SubnetItem `xml:"subnet"`
}
