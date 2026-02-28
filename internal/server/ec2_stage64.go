package server

import (
	"encoding/xml"
	"net/http"
	"strings"
	"time"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage64Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "AcceptVpcEndpointConnections":
		unsuccessful, err := s.ec2.AcceptVpcEndpointConnections(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			parseEC2MembersWithAliases(r.Form, "VpcEndpointId", "VpcEndpointIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AcceptVpcEndpointConnectionsResponse{
			XMLName:      xml.Name{Local: "AcceptVpcEndpointConnectionsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	case "RejectVpcEndpointConnections":
		unsuccessful, err := s.ec2.RejectVpcEndpointConnections(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			parseEC2MembersWithAliases(r.Form, "VpcEndpointId", "VpcEndpointIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RejectVpcEndpointConnectionsResponse{
			XMLName:      xml.Name{Local: "RejectVpcEndpointConnectionsResponse"},
			Xmlns:        ec2Namespace,
			RequestID:    "stackyard-request",
			Unsuccessful: ec2UnsuccessfulItemSet{Items: ec2UnsuccessfulItems(unsuccessful)},
		})
		return true
	case "DescribeVpcEndpoints":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endpoints, nextToken, err := s.ec2.DescribeVpcEndpoints(
			parseEC2MembersWithAliases(r.Form, "VpcEndpointId", "VpcEndpointIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcEndpointSet: ec2VpcEndpointSet{
				Items: ec2VpcEndpointItems(endpoints),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeVpcEndpointConnections":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		connections, nextToken, err := s.ec2.DescribeVpcEndpointConnections(
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointConnectionsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointConnectionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcEndpointConnectionSet: ec2VpcEndpointConnectionSet{
				Items: ec2VpcEndpointConnectionItems(connections),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "DescribeVpcEndpointAssociations":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		associations, nextToken, err := s.ec2.DescribeVpcEndpointAssociations(
			parseEC2MembersWithAliases(r.Form, "VpcEndpointId", "VpcEndpointIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeVpcEndpointAssociationsResponse{
			XMLName:   xml.Name{Local: "DescribeVpcEndpointAssociationsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpcEndpointAssociationSet: ec2VpcEndpointAssociationSet{
				Items: ec2VpcEndpointAssociationItems(associations),
			},
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

func ec2VpcEndpointItems(in []ec2svc.VpcEndpoint) []ec2VpcEndpointItem {
	out := make([]ec2VpcEndpointItem, 0, len(in))
	for _, endpoint := range in {
		out = append(out, ec2VpcEndpointItemFrom(endpoint))
	}
	return out
}

func ec2VpcEndpointConnectionItems(in []ec2svc.VpcEndpointConnection) []ec2VpcEndpointConnectionItem {
	out := make([]ec2VpcEndpointConnectionItem, 0, len(in))
	for _, connection := range in {
		out = append(out, ec2VpcEndpointConnectionItem{
			CreationTimestamp:       connection.CreationTimestamp,
			DnsEntrySet:             ec2Stage64DnsEntrySet{Items: ec2Stage64DnsEntryItems(connection.DnsEntries)},
			GatewayLoadBalancerARNs: ec2Stage55StringSet{Items: append([]string(nil), connection.GatewayLoadBalancerARNs...)},
			IPAddressType:           connection.IPAddressType,
			NetworkLoadBalancerARNs: ec2Stage55StringSet{Items: append([]string(nil), connection.NetworkLoadBalancerARNs...)},
			ServiceID:               connection.ServiceID,
			TagSet:                  ec2TagSet{Items: ec2TagItemsFromMap(connection.Tags)},
			VpcEndpointConnectionID: connection.VpcEndpointConnectionID,
			VpcEndpointID:           connection.VpcEndpointID,
			VpcEndpointOwner:        connection.VpcEndpointOwner,
			VpcEndpointRegion:       connection.VpcEndpointRegion,
			VpcEndpointState:        connection.VpcEndpointState,
		})
	}
	return out
}

func ec2VpcEndpointAssociationItems(in []ec2svc.VpcEndpointAssociation) []ec2VpcEndpointAssociationItem {
	out := make([]ec2VpcEndpointAssociationItem, 0, len(in))
	for _, association := range in {
		item := ec2VpcEndpointAssociationItem{
			AssociatedResourceAccessibility: association.AssociatedResourceAccessibility,
			AssociatedResourceARN:           association.AssociatedResourceARN,
			FailureCode:                     association.FailureCode,
			FailureReason:                   association.FailureReason,
			ID:                              association.ID,
			ResourceConfigurationGroupARN:   association.ResourceConfigurationGroupARN,
			ServiceNetworkARN:               association.ServiceNetworkARN,
			ServiceNetworkName:              association.ServiceNetworkName,
			TagSet:                          ec2TagSet{Items: ec2TagItemsFromMap(association.Tags)},
			VpcEndpointID:                   association.VpcEndpointID,
		}
		if association.DnsEntry != nil {
			item.DnsEntry = &ec2Stage64DnsEntryItem{
				DnsName:      association.DnsEntry.DnsName,
				HostedZoneID: association.DnsEntry.HostedZoneID,
			}
		}
		if association.PrivateDnsEntry != nil {
			item.PrivateDnsEntry = &ec2Stage64DnsEntryItem{
				DnsName:      association.PrivateDnsEntry.DnsName,
				HostedZoneID: association.PrivateDnsEntry.HostedZoneID,
			}
		}
		out = append(out, item)
	}
	return out
}

func ec2Stage64DnsEntryItems(in []ec2svc.DnsEntry) []ec2Stage64DnsEntryItem {
	out := make([]ec2Stage64DnsEntryItem, 0, len(in))
	for _, entry := range in {
		out = append(out, ec2Stage64DnsEntryItem{
			DnsName:      entry.DnsName,
			HostedZoneID: entry.HostedZoneID,
		})
	}
	return out
}

type ec2AcceptVpcEndpointConnectionsResponse struct {
	XMLName      xml.Name               `xml:"AcceptVpcEndpointConnectionsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}

type ec2RejectVpcEndpointConnectionsResponse struct {
	XMLName      xml.Name               `xml:"RejectVpcEndpointConnectionsResponse"`
	Xmlns        string                 `xml:"xmlns,attr"`
	RequestID    string                 `xml:"requestId"`
	Unsuccessful ec2UnsuccessfulItemSet `xml:"unsuccessful"`
}

type ec2DescribeVpcEndpointsResponse struct {
	XMLName        xml.Name          `xml:"DescribeVpcEndpointsResponse"`
	Xmlns          string            `xml:"xmlns,attr"`
	RequestID      string            `xml:"requestId"`
	NextToken      string            `xml:"nextToken,omitempty"`
	VpcEndpointSet ec2VpcEndpointSet `xml:"vpcEndpointSet"`
}

type ec2VpcEndpointSet struct {
	Items []ec2VpcEndpointItem `xml:"item"`
}

type ec2DescribeVpcEndpointConnectionsResponse struct {
	XMLName                  xml.Name                    `xml:"DescribeVpcEndpointConnectionsResponse"`
	Xmlns                    string                      `xml:"xmlns,attr"`
	RequestID                string                      `xml:"requestId"`
	NextToken                string                      `xml:"nextToken,omitempty"`
	VpcEndpointConnectionSet ec2VpcEndpointConnectionSet `xml:"vpcEndpointConnectionSet"`
}

type ec2VpcEndpointConnectionSet struct {
	Items []ec2VpcEndpointConnectionItem `xml:"item"`
}

type ec2VpcEndpointConnectionItem struct {
	CreationTimestamp       time.Time             `xml:"creationTimestamp,omitempty"`
	DnsEntrySet             ec2Stage64DnsEntrySet `xml:"dnsEntrySet"`
	GatewayLoadBalancerARNs ec2Stage55StringSet   `xml:"gatewayLoadBalancerArnSet"`
	IPAddressType           string                `xml:"ipAddressType,omitempty"`
	NetworkLoadBalancerARNs ec2Stage55StringSet   `xml:"networkLoadBalancerArnSet"`
	ServiceID               string                `xml:"serviceId,omitempty"`
	TagSet                  ec2TagSet             `xml:"tagSet"`
	VpcEndpointConnectionID string                `xml:"vpcEndpointConnectionId,omitempty"`
	VpcEndpointID           string                `xml:"vpcEndpointId,omitempty"`
	VpcEndpointOwner        string                `xml:"vpcEndpointOwner,omitempty"`
	VpcEndpointRegion       string                `xml:"vpcEndpointRegion,omitempty"`
	VpcEndpointState        string                `xml:"vpcEndpointState,omitempty"`
}

type ec2DescribeVpcEndpointAssociationsResponse struct {
	XMLName                   xml.Name                     `xml:"DescribeVpcEndpointAssociationsResponse"`
	Xmlns                     string                       `xml:"xmlns,attr"`
	RequestID                 string                       `xml:"requestId"`
	NextToken                 string                       `xml:"nextToken,omitempty"`
	VpcEndpointAssociationSet ec2VpcEndpointAssociationSet `xml:"vpcEndpointAssociationSet"`
}

type ec2VpcEndpointAssociationSet struct {
	Items []ec2VpcEndpointAssociationItem `xml:"item"`
}

type ec2VpcEndpointAssociationItem struct {
	AssociatedResourceAccessibility string                  `xml:"associatedResourceAccessibility,omitempty"`
	AssociatedResourceARN           string                  `xml:"associatedResourceArn,omitempty"`
	DnsEntry                        *ec2Stage64DnsEntryItem `xml:"dnsEntry,omitempty"`
	FailureCode                     string                  `xml:"failureCode,omitempty"`
	FailureReason                   string                  `xml:"failureReason,omitempty"`
	ID                              string                  `xml:"id,omitempty"`
	PrivateDnsEntry                 *ec2Stage64DnsEntryItem `xml:"privateDnsEntry,omitempty"`
	ResourceConfigurationGroupARN   string                  `xml:"resourceConfigurationGroupArn,omitempty"`
	ServiceNetworkARN               string                  `xml:"serviceNetworkArn,omitempty"`
	ServiceNetworkName              string                  `xml:"serviceNetworkName,omitempty"`
	TagSet                          ec2TagSet               `xml:"tagSet"`
	VpcEndpointID                   string                  `xml:"vpcEndpointId,omitempty"`
}

type ec2Stage64DnsEntrySet struct {
	Items []ec2Stage64DnsEntryItem `xml:"item"`
}

type ec2Stage64DnsEntryItem struct {
	DnsName      string `xml:"dnsName,omitempty"`
	HostedZoneID string `xml:"hostedZoneId,omitempty"`
}
