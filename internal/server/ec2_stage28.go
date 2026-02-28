package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage28Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AssignPrivateIpAddresses":
		secondaryCount, hasSecondaryCount, okSecondaryCount := ec2OptionalInt32FromForm(r.Form, "SecondaryPrivateIpAddressCount")
		if hasSecondaryCount && !okSecondaryCount {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ipv4PrefixCount, hasIPv4PrefixCount, okIPv4PrefixCount := ec2OptionalInt32FromForm(r.Form, "Ipv4PrefixCount")
		if hasIPv4PrefixCount && !okIPv4PrefixCount {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		allowReassignment, hasAllowReassignment, okAllowReassignment := ec2OptionalBoolFromForm(r.Form, "AllowReassignment")
		if hasAllowReassignment && !okAllowReassignment {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		allow := false
		if allowReassignment != nil {
			allow = *allowReassignment
		}

		result, err := s.ec2.AssignPrivateIPAddresses(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			parseEC2Members(r.Form, "PrivateIpAddress."),
			secondaryCount,
			parseEC2Members(r.Form, "Ipv4Prefix."),
			ipv4PrefixCount,
			allow,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssignPrivateIPAddressesResponse{
			XMLName:   xml.Name{Local: "AssignPrivateIpAddressesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			AssignedPrivateIPAddressesSet: ec2AssignedPrivateIPAddressSet{
				Items: ec2AssignedPrivateIPAddressItems(result.AssignedPrivateIPs),
			},
			AssignedIPv4PrefixSet: ec2IPv4PrefixSpecificationSet{
				Items: ec2IPv4PrefixSpecificationItems(result.AssignedIPv4Prefixes),
			},
			NetworkInterfaceID: result.NetworkInterfaceID,
		})
		return true
	case "UnassignPrivateIpAddresses":
		if err := s.ec2.UnassignPrivateIPAddresses(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			parseEC2Members(r.Form, "PrivateIpAddress."),
			parseEC2Members(r.Form, "Ipv4Prefix."),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2UnassignPrivateIPAddressesResponse{
			XMLName:   xml.Name{Local: "UnassignPrivateIpAddressesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
		})
		return true
	case "AssignIpv6Addresses":
		ipv6Count, hasIPv6Count, okIPv6Count := ec2OptionalInt32FromForm(r.Form, "Ipv6AddressCount")
		if hasIPv6Count && !okIPv6Count {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		ipv6PrefixCount, hasIPv6PrefixCount, okIPv6PrefixCount := ec2OptionalInt32FromForm(r.Form, "Ipv6PrefixCount")
		if hasIPv6PrefixCount && !okIPv6PrefixCount {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		result, err := s.ec2.AssignIPv6Addresses(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			parseEC2Members(r.Form, "Ipv6Addresses."),
			ipv6Count,
			parseEC2Members(r.Form, "Ipv6Prefix."),
			ipv6PrefixCount,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssignIPv6AddressesResponse{
			XMLName:               xml.Name{Local: "AssignIpv6AddressesResponse"},
			Xmlns:                 ec2Namespace,
			RequestID:             "stackyard-request",
			AssignedIPv6Addresses: ec2ValueStringSet{Items: append([]string(nil), result.AssignedIPv6Addrs...)},
			AssignedIPv6PrefixSet: ec2ValueStringSet{Items: append([]string(nil), result.AssignedIPv6Prefixes...)},
			NetworkInterfaceID:    result.NetworkInterfaceID,
		})
		return true
	case "UnassignIpv6Addresses":
		result, err := s.ec2.UnassignIPv6Addresses(
			strings.TrimSpace(r.Form.Get("NetworkInterfaceId")),
			parseEC2Members(r.Form, "Ipv6Addresses."),
			parseEC2Members(r.Form, "Ipv6Prefix."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2UnassignIPv6AddressesResponse{
			XMLName:                 xml.Name{Local: "UnassignIpv6AddressesResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			NetworkInterfaceID:      result.NetworkInterfaceID,
			UnassignedIPv6Addresses: ec2ValueStringSet{Items: append([]string(nil), result.UnassignedIPv6Addrs...)},
			UnassignedIPv6PrefixSet: ec2ValueStringSet{Items: append([]string(nil), result.UnassignedIPv6Prefixes...)},
		})
		return true
	default:
		return false
	}
}

func ec2AssignedPrivateIPAddressItems(in []ec2svc.AssignedPrivateIPAddress) []ec2AssignedPrivateIPAddressItem {
	out := make([]ec2AssignedPrivateIPAddressItem, 0, len(in))
	for _, item := range in {
		out = append(out, ec2AssignedPrivateIPAddressItem{PrivateIPAddress: item.PrivateIPAddress})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PrivateIPAddress < out[j].PrivateIPAddress })
	return out
}

func ec2IPv4PrefixSpecificationItems(in []string) []ec2IPv4PrefixSpecificationItem {
	values := dedupeStrings(in)
	sort.Strings(values)
	out := make([]ec2IPv4PrefixSpecificationItem, 0, len(values))
	for _, prefix := range values {
		out = append(out, ec2IPv4PrefixSpecificationItem{IPv4Prefix: prefix})
	}
	return out
}

type ec2AssignPrivateIPAddressesResponse struct {
	XMLName                       xml.Name                       `xml:"AssignPrivateIpAddressesResponse"`
	Xmlns                         string                         `xml:"xmlns,attr"`
	RequestID                     string                         `xml:"requestId"`
	AssignedPrivateIPAddressesSet ec2AssignedPrivateIPAddressSet `xml:"assignedPrivateIpAddressesSet"`
	AssignedIPv4PrefixSet         ec2IPv4PrefixSpecificationSet  `xml:"assignedIpv4PrefixSet"`
	NetworkInterfaceID            string                         `xml:"networkInterfaceId,omitempty"`
}

type ec2AssignedPrivateIPAddressSet struct {
	Items []ec2AssignedPrivateIPAddressItem `xml:"item"`
}

type ec2AssignedPrivateIPAddressItem struct {
	PrivateIPAddress string `xml:"privateIpAddress,omitempty"`
}

type ec2IPv4PrefixSpecificationSet struct {
	Items []ec2IPv4PrefixSpecificationItem `xml:"item"`
}

type ec2IPv4PrefixSpecificationItem struct {
	IPv4Prefix string `xml:"ipv4Prefix,omitempty"`
}

type ec2UnassignPrivateIPAddressesResponse struct {
	XMLName   xml.Name `xml:"UnassignPrivateIpAddressesResponse"`
	Xmlns     string   `xml:"xmlns,attr"`
	RequestID string   `xml:"requestId"`
}

type ec2AssignIPv6AddressesResponse struct {
	XMLName               xml.Name          `xml:"AssignIpv6AddressesResponse"`
	Xmlns                 string            `xml:"xmlns,attr"`
	RequestID             string            `xml:"requestId"`
	AssignedIPv6Addresses ec2ValueStringSet `xml:"assignedIpv6Addresses"`
	AssignedIPv6PrefixSet ec2ValueStringSet `xml:"assignedIpv6PrefixSet"`
	NetworkInterfaceID    string            `xml:"networkInterfaceId,omitempty"`
}

type ec2UnassignIPv6AddressesResponse struct {
	XMLName                 xml.Name          `xml:"UnassignIpv6AddressesResponse"`
	Xmlns                   string            `xml:"xmlns,attr"`
	RequestID               string            `xml:"requestId"`
	NetworkInterfaceID      string            `xml:"networkInterfaceId,omitempty"`
	UnassignedIPv6Addresses ec2ValueStringSet `xml:"unassignedIpv6Addresses"`
	UnassignedIPv6PrefixSet ec2ValueStringSet `xml:"unassignedIpv6PrefixSet"`
}
