package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage56Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpcEndpoint":
		privateDNSEnabled, hasPrivateDNSEnabled, ok := ec2OptionalBoolFromForm(r.Form, "PrivateDnsEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPrivateDNSEnabled {
			privateDNSEnabled = nil
		}

		endpoint, clientToken, err := s.ec2.CreateVpcEndpoint(
			strings.TrimSpace(r.Form.Get("VpcId")),
			strings.TrimSpace(r.Form.Get("ServiceName")),
			strings.TrimSpace(r.Form.Get("ServiceRegion")),
			strings.TrimSpace(r.Form.Get("VpcEndpointType")),
			strings.TrimSpace(r.Form.Get("IpAddressType")),
			parseEC2MembersWithAliases(r.Form, "RouteTableId", "RouteTableIds"),
			parseEC2MembersWithAliases(r.Form, "SecurityGroupId", "SecurityGroupIds"),
			parseEC2MembersWithAliases(r.Form, "SubnetId", "SubnetIds"),
			parseEC2SubnetConfigurationSubnetIDs(r.Form),
			parseEC2OptionalString(r.Form.Get("PolicyDocument")),
			privateDNSEnabled,
			parseEC2OptionalString(r.Form.Get("ResourceConfigurationArn")),
			parseEC2OptionalString(r.Form.Get("ServiceNetworkArn")),
			parseEC2TagSpecificationsForResource(r.Form, "vpc-endpoint"),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2CreateVpcEndpointResponse{
			XMLName:     xml.Name{Local: "CreateVpcEndpointResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			ClientToken: clientToken,
			VpcEndpoint: ec2VpcEndpointItemFrom(endpoint),
		})
		return true
	default:
		return false
	}
}

func ec2VpcEndpointItemFrom(in ec2svc.VpcEndpoint) ec2VpcEndpointItem {
	groups := make([]ec2GroupSetItem, 0, len(in.SecurityGroupIDs))
	for _, groupID := range in.SecurityGroupIDs {
		groups = append(groups, ec2GroupSetItem{GroupID: groupID})
	}
	return ec2VpcEndpointItem{
		CreationTimestamp: in.CreationTimestamp.Format(timeRFC3339),
		GroupSet:          ec2GroupSet{Items: groups},
		IPAddressType:     in.IPAddressType,
		OwnerID:           in.OwnerID,
		PolicyDocument:    in.PolicyDocument,
		PrivateDNSEnabled: &in.PrivateDNSEnabled,
		ResourceConfigARN: in.ResourceConfigARN,
		RouteTableIDSet:   ec2Stage56StringSet{Items: append([]string(nil), in.RouteTableIDs...)},
		ServiceName:       in.ServiceName,
		ServiceNetworkARN: in.ServiceNetworkARN,
		ServiceRegion:     in.ServiceRegion,
		State:             in.State,
		SubnetIDSet:       ec2Stage56StringSet{Items: append([]string(nil), in.SubnetIDs...)},
		TagSet:            ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		VpcEndpointID:     in.ID,
		VpcEndpointType:   in.VpcEndpointType,
		VpcID:             in.VpcID,
	}
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

type ec2CreateVpcEndpointResponse struct {
	XMLName     xml.Name           `xml:"CreateVpcEndpointResponse"`
	Xmlns       string             `xml:"xmlns,attr"`
	RequestID   string             `xml:"requestId"`
	ClientToken *string            `xml:"clientToken,omitempty"`
	VpcEndpoint ec2VpcEndpointItem `xml:"vpcEndpoint"`
}

type ec2VpcEndpointItem struct {
	CreationTimestamp string              `xml:"creationTimestamp,omitempty"`
	GroupSet          ec2GroupSet         `xml:"groupSet"`
	IPAddressType     string              `xml:"ipAddressType,omitempty"`
	OwnerID           string              `xml:"ownerId,omitempty"`
	PolicyDocument    string              `xml:"policyDocument,omitempty"`
	PrivateDNSEnabled *bool               `xml:"privateDnsEnabled,omitempty"`
	ResourceConfigARN string              `xml:"resourceConfigurationArn,omitempty"`
	RouteTableIDSet   ec2Stage56StringSet `xml:"routeTableIdSet"`
	ServiceName       string              `xml:"serviceName,omitempty"`
	ServiceNetworkARN string              `xml:"serviceNetworkArn,omitempty"`
	ServiceRegion     string              `xml:"serviceRegion,omitempty"`
	State             string              `xml:"state,omitempty"`
	SubnetIDSet       ec2Stage56StringSet `xml:"subnetIdSet"`
	TagSet            ec2TagSet           `xml:"tagSet"`
	VpcEndpointID     string              `xml:"vpcEndpointId,omitempty"`
	VpcEndpointType   string              `xml:"vpcEndpointType,omitempty"`
	VpcID             string              `xml:"vpcId,omitempty"`
}

type ec2Stage56StringSet struct {
	Items []string `xml:"item"`
}
