package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage42Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "RegisterTransitGatewayMulticastGroupMembers":
		registered, err := s.ec2.RegisterTransitGatewayMulticastGroupMembers(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("GroupIpAddress")),
			parseEC2MembersOrItemList(r.Form, "NetworkInterfaceIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RegisterTransitGatewayMulticastGroupMembersResponse{
			XMLName:                         xml.Name{Local: "RegisterTransitGatewayMulticastGroupMembersResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			RegisteredMulticastGroupMembers: ec2TransitGatewayMulticastRegisteredGroupMembersFrom(registered),
		})
		return true
	case "DeregisterTransitGatewayMulticastGroupMembers":
		deregistered, err := s.ec2.DeregisterTransitGatewayMulticastGroupMembers(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("GroupIpAddress")),
			parseEC2MembersOrItemList(r.Form, "NetworkInterfaceIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeregisterTransitGatewayMulticastGroupMembersResponse{
			XMLName:                           xml.Name{Local: "DeregisterTransitGatewayMulticastGroupMembersResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			DeregisteredMulticastGroupMembers: ec2TransitGatewayMulticastDeregisteredGroupMembersFrom(deregistered),
		})
		return true
	case "RegisterTransitGatewayMulticastGroupSources":
		registered, err := s.ec2.RegisterTransitGatewayMulticastGroupSources(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("GroupIpAddress")),
			parseEC2MembersOrItemList(r.Form, "NetworkInterfaceIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RegisterTransitGatewayMulticastGroupSourcesResponse{
			XMLName:                         xml.Name{Local: "RegisterTransitGatewayMulticastGroupSourcesResponse"},
			Xmlns:                           ec2Namespace,
			RequestID:                       "stackyard-request",
			RegisteredMulticastGroupSources: ec2TransitGatewayMulticastRegisteredGroupSourcesFrom(registered),
		})
		return true
	case "DeregisterTransitGatewayMulticastGroupSources":
		deregistered, err := s.ec2.DeregisterTransitGatewayMulticastGroupSources(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			strings.TrimSpace(r.Form.Get("GroupIpAddress")),
			parseEC2MembersOrItemList(r.Form, "NetworkInterfaceIds"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeregisterTransitGatewayMulticastGroupSourcesResponse{
			XMLName:                           xml.Name{Local: "DeregisterTransitGatewayMulticastGroupSourcesResponse"},
			Xmlns:                             ec2Namespace,
			RequestID:                         "stackyard-request",
			DeregisteredMulticastGroupSources: ec2TransitGatewayMulticastDeregisteredGroupSourcesFrom(deregistered),
		})
		return true
	case "SearchTransitGatewayMulticastGroups":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		groups, nextToken, err := s.ec2.SearchTransitGatewayMulticastGroups(
			strings.TrimSpace(r.Form.Get("TransitGatewayMulticastDomainId")),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2SearchTransitGatewayMulticastGroupsResponse{
			XMLName:         xml.Name{Local: "SearchTransitGatewayMulticastGroupsResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			MulticastGroups: ec2TransitGatewayMulticastGroupSet{Items: ec2TransitGatewayMulticastGroupItems(groups)},
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

func ec2TransitGatewayMulticastRegisteredGroupMembersFrom(in ec2svc.TransitGatewayMulticastRegisteredGroupMembers) ec2TransitGatewayMulticastRegisteredGroupMembersItem {
	return ec2TransitGatewayMulticastRegisteredGroupMembersItem{
		GroupIPAddress:                  in.GroupIpAddress,
		RegisteredNetworkInterfaceIDSet: ec2StringSet{Items: ec2StringItems(in.RegisteredNetworkInterfaceIDs)},
		TransitGatewayMulticastDomainID: in.TransitGatewayMulticastDomainID,
	}
}

func ec2TransitGatewayMulticastDeregisteredGroupMembersFrom(in ec2svc.TransitGatewayMulticastDeregisteredGroupMembers) ec2TransitGatewayMulticastDeregisteredGroupMembersItem {
	return ec2TransitGatewayMulticastDeregisteredGroupMembersItem{
		DeregisteredNetworkInterfaceIDSet: ec2StringSet{Items: ec2StringItems(in.DeregisteredNetworkInterfaceIDs)},
		GroupIPAddress:                    in.GroupIpAddress,
		TransitGatewayMulticastDomainID:   in.TransitGatewayMulticastDomainID,
	}
}

func ec2TransitGatewayMulticastRegisteredGroupSourcesFrom(in ec2svc.TransitGatewayMulticastRegisteredGroupSources) ec2TransitGatewayMulticastRegisteredGroupSourcesItem {
	return ec2TransitGatewayMulticastRegisteredGroupSourcesItem{
		GroupIPAddress:                  in.GroupIpAddress,
		RegisteredNetworkInterfaceIDSet: ec2StringSet{Items: ec2StringItems(in.RegisteredNetworkInterfaceIDs)},
		TransitGatewayMulticastDomainID: in.TransitGatewayMulticastDomainID,
	}
}

func ec2TransitGatewayMulticastDeregisteredGroupSourcesFrom(in ec2svc.TransitGatewayMulticastDeregisteredGroupSources) ec2TransitGatewayMulticastDeregisteredGroupSourcesItem {
	return ec2TransitGatewayMulticastDeregisteredGroupSourcesItem{
		DeregisteredNetworkInterfaceIDSet: ec2StringSet{Items: ec2StringItems(in.DeregisteredNetworkInterfaceIDs)},
		GroupIPAddress:                    in.GroupIpAddress,
		TransitGatewayMulticastDomainID:   in.TransitGatewayMulticastDomainID,
	}
}

func ec2TransitGatewayMulticastGroupItems(in []ec2svc.TransitGatewayMulticastGroup) []ec2TransitGatewayMulticastGroupItem {
	out := make([]ec2TransitGatewayMulticastGroupItem, 0, len(in))
	for _, group := range in {
		groupMember := group.GroupMember
		groupSource := group.GroupSource
		out = append(out, ec2TransitGatewayMulticastGroupItem{
			GroupIPAddress:             group.GroupIpAddress,
			GroupMember:                &groupMember,
			GroupSource:                &groupSource,
			MemberType:                 group.MemberType,
			NetworkInterfaceID:         group.NetworkInterfaceID,
			ResourceID:                 group.ResourceID,
			ResourceOwnerID:            group.ResourceOwnerID,
			ResourceType:               group.ResourceType,
			SourceType:                 group.SourceType,
			SubnetID:                   group.SubnetID,
			TransitGatewayAttachmentID: group.TransitGatewayAttachmentID,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].GroupIPAddress != out[j].GroupIPAddress {
			return out[i].GroupIPAddress < out[j].GroupIPAddress
		}
		return out[i].NetworkInterfaceID < out[j].NetworkInterfaceID
	})
	return out
}

func ec2StringItems(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

type ec2RegisterTransitGatewayMulticastGroupMembersResponse struct {
	XMLName                         xml.Name                                             `xml:"RegisterTransitGatewayMulticastGroupMembersResponse"`
	Xmlns                           string                                               `xml:"xmlns,attr"`
	RequestID                       string                                               `xml:"requestId"`
	RegisteredMulticastGroupMembers ec2TransitGatewayMulticastRegisteredGroupMembersItem `xml:"registeredMulticastGroupMembers"`
}

type ec2DeregisterTransitGatewayMulticastGroupMembersResponse struct {
	XMLName                           xml.Name                                               `xml:"DeregisterTransitGatewayMulticastGroupMembersResponse"`
	Xmlns                             string                                                 `xml:"xmlns,attr"`
	RequestID                         string                                                 `xml:"requestId"`
	DeregisteredMulticastGroupMembers ec2TransitGatewayMulticastDeregisteredGroupMembersItem `xml:"deregisteredMulticastGroupMembers"`
}

type ec2RegisterTransitGatewayMulticastGroupSourcesResponse struct {
	XMLName                         xml.Name                                             `xml:"RegisterTransitGatewayMulticastGroupSourcesResponse"`
	Xmlns                           string                                               `xml:"xmlns,attr"`
	RequestID                       string                                               `xml:"requestId"`
	RegisteredMulticastGroupSources ec2TransitGatewayMulticastRegisteredGroupSourcesItem `xml:"registeredMulticastGroupSources"`
}

type ec2DeregisterTransitGatewayMulticastGroupSourcesResponse struct {
	XMLName                           xml.Name                                               `xml:"DeregisterTransitGatewayMulticastGroupSourcesResponse"`
	Xmlns                             string                                                 `xml:"xmlns,attr"`
	RequestID                         string                                                 `xml:"requestId"`
	DeregisteredMulticastGroupSources ec2TransitGatewayMulticastDeregisteredGroupSourcesItem `xml:"deregisteredMulticastGroupSources"`
}

type ec2SearchTransitGatewayMulticastGroupsResponse struct {
	XMLName         xml.Name                           `xml:"SearchTransitGatewayMulticastGroupsResponse"`
	Xmlns           string                             `xml:"xmlns,attr"`
	RequestID       string                             `xml:"requestId"`
	MulticastGroups ec2TransitGatewayMulticastGroupSet `xml:"multicastGroups"`
	NextToken       string                             `xml:"nextToken,omitempty"`
}

type ec2TransitGatewayMulticastRegisteredGroupMembersItem struct {
	GroupIPAddress                  string       `xml:"groupIpAddress,omitempty"`
	RegisteredNetworkInterfaceIDSet ec2StringSet `xml:"registeredNetworkInterfaceIds"`
	TransitGatewayMulticastDomainID string       `xml:"transitGatewayMulticastDomainId,omitempty"`
}

type ec2TransitGatewayMulticastDeregisteredGroupMembersItem struct {
	DeregisteredNetworkInterfaceIDSet ec2StringSet `xml:"deregisteredNetworkInterfaceIds"`
	GroupIPAddress                    string       `xml:"groupIpAddress,omitempty"`
	TransitGatewayMulticastDomainID   string       `xml:"transitGatewayMulticastDomainId,omitempty"`
}

type ec2TransitGatewayMulticastRegisteredGroupSourcesItem struct {
	GroupIPAddress                  string       `xml:"groupIpAddress,omitempty"`
	RegisteredNetworkInterfaceIDSet ec2StringSet `xml:"registeredNetworkInterfaceIds"`
	TransitGatewayMulticastDomainID string       `xml:"transitGatewayMulticastDomainId,omitempty"`
}

type ec2TransitGatewayMulticastDeregisteredGroupSourcesItem struct {
	DeregisteredNetworkInterfaceIDSet ec2StringSet `xml:"deregisteredNetworkInterfaceIds"`
	GroupIPAddress                    string       `xml:"groupIpAddress,omitempty"`
	TransitGatewayMulticastDomainID   string       `xml:"transitGatewayMulticastDomainId,omitempty"`
}

type ec2TransitGatewayMulticastGroupSet struct {
	Items []ec2TransitGatewayMulticastGroupItem `xml:"item"`
}

type ec2TransitGatewayMulticastGroupItem struct {
	GroupIPAddress             string `xml:"groupIpAddress,omitempty"`
	GroupMember                *bool  `xml:"groupMember"`
	GroupSource                *bool  `xml:"groupSource"`
	MemberType                 string `xml:"memberType,omitempty"`
	NetworkInterfaceID         string `xml:"networkInterfaceId,omitempty"`
	ResourceID                 string `xml:"resourceId,omitempty"`
	ResourceOwnerID            string `xml:"resourceOwnerId,omitempty"`
	ResourceType               string `xml:"resourceType,omitempty"`
	SourceType                 string `xml:"sourceType,omitempty"`
	SubnetID                   string `xml:"subnetId,omitempty"`
	TransitGatewayAttachmentID string `xml:"transitGatewayAttachmentId,omitempty"`
}
