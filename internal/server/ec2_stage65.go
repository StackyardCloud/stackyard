package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage65Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateRouteServer":
		amazonSideASN, ok := parseEC2OptionalInt64(r.Form.Get("AmazonSideAsn"))
		if !ok || amazonSideASN == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		persistRoutesDuration, ok := parseEC2OptionalInt64(r.Form.Get("PersistRoutesDuration"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		snsEnabled, hasSNSValue, ok := ec2OptionalBoolFromForm(r.Form, "SnsNotificationsEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasSNSValue {
			snsEnabled = nil
		}

		routeServer, err := s.ec2.CreateRouteServer(
			*amazonSideASN,
			strings.TrimSpace(r.Form.Get("PersistRoutes")),
			persistRoutesDuration,
			snsEnabled,
			parseEC2TagSpecificationsForResource(r.Form, "route-server"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateRouteServerResponse{
			XMLName:     xml.Name{Local: "CreateRouteServerResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			RouteServer: ec2RouteServerItemFrom(routeServer),
		})
		return true
	case "DeleteRouteServer":
		routeServer, err := s.ec2.DeleteRouteServer(strings.TrimSpace(r.Form.Get("RouteServerId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteRouteServerResponse{
			XMLName:     xml.Name{Local: "DeleteRouteServerResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			RouteServer: ec2RouteServerItemFrom(routeServer),
		})
		return true
	case "DescribeRouteServers":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		routeServers, nextToken, err := s.ec2.DescribeRouteServers(
			parseEC2MembersWithAliases(r.Form, "RouteServerId", "RouteServerIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeRouteServersResponse{
			XMLName:   xml.Name{Local: "DescribeRouteServersResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			RouteServerSet: ec2RouteServerSet{
				Items: ec2RouteServerItems(routeServers),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateRouteServerEndpoint":
		endpoint, err := s.ec2.CreateRouteServerEndpoint(
			strings.TrimSpace(r.Form.Get("RouteServerId")),
			strings.TrimSpace(r.Form.Get("SubnetId")),
			parseEC2TagSpecificationsForResource(r.Form, "route-server-endpoint"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateRouteServerEndpointResponse{
			XMLName:             xml.Name{Local: "CreateRouteServerEndpointResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			RouteServerEndpoint: ec2RouteServerEndpointItemFrom(endpoint),
		})
		return true
	case "DeleteRouteServerEndpoint":
		endpoint, err := s.ec2.DeleteRouteServerEndpoint(strings.TrimSpace(r.Form.Get("RouteServerEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteRouteServerEndpointResponse{
			XMLName:             xml.Name{Local: "DeleteRouteServerEndpointResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			RouteServerEndpoint: ec2RouteServerEndpointItemFrom(endpoint),
		})
		return true
	case "DescribeRouteServerEndpoints":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		endpoints, nextToken, err := s.ec2.DescribeRouteServerEndpoints(
			parseEC2MembersWithAliases(r.Form, "RouteServerEndpointId", "RouteServerEndpointIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeRouteServerEndpointsResponse{
			XMLName:   xml.Name{Local: "DescribeRouteServerEndpointsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			RouteServerEndpointSet: ec2RouteServerEndpointSet{
				Items: ec2RouteServerEndpointItems(endpoints),
			},
		}
		if nextToken != nil {
			response.NextToken = *nextToken
		}
		respondEC2XML(w, response)
		return true
	case "CreateRouteServerPeer":
		peerASN, ok := parseEC2OptionalInt64(r.Form.Get("BgpOptions.PeerAsn"))
		if !ok || peerASN == nil {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		peer, err := s.ec2.CreateRouteServerPeer(
			strings.TrimSpace(r.Form.Get("RouteServerEndpointId")),
			strings.TrimSpace(r.Form.Get("PeerAddress")),
			ec2svc.RouteServerBgpOptionsRequest{
				PeerASN:               *peerASN,
				PeerLivenessDetection: strings.TrimSpace(r.Form.Get("BgpOptions.PeerLivenessDetection")),
			},
			parseEC2TagSpecificationsForResource(r.Form, "route-server-peer"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateRouteServerPeerResponse{
			XMLName:         xml.Name{Local: "CreateRouteServerPeerResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			RouteServerPeer: ec2RouteServerPeerItemFrom(peer),
		})
		return true
	case "DeleteRouteServerPeer":
		peer, err := s.ec2.DeleteRouteServerPeer(strings.TrimSpace(r.Form.Get("RouteServerPeerId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteRouteServerPeerResponse{
			XMLName:         xml.Name{Local: "DeleteRouteServerPeerResponse"},
			Xmlns:           ec2Namespace,
			RequestID:       "stackyard-request",
			RouteServerPeer: ec2RouteServerPeerItemFrom(peer),
		})
		return true
	case "DescribeRouteServerPeers":
		maxResults, ok := parseEC2OptionalInt32(r.Form.Get("MaxResults"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		peers, nextToken, err := s.ec2.DescribeRouteServerPeers(
			parseEC2MembersWithAliases(r.Form, "RouteServerPeerId", "RouteServerPeerIds"),
			parseEC2Filters(r.Form),
			maxResults,
			parseEC2OptionalString(r.Form.Get("NextToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		response := ec2DescribeRouteServerPeersResponse{
			XMLName:   xml.Name{Local: "DescribeRouteServerPeersResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			RouteServerPeerSet: ec2RouteServerPeerSet{
				Items: ec2RouteServerPeerItems(peers),
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

func ec2RouteServerItems(in []ec2svc.RouteServer) []ec2RouteServerItem {
	out := make([]ec2RouteServerItem, 0, len(in))
	for _, routeServer := range in {
		out = append(out, ec2RouteServerItemFrom(routeServer))
	}
	return out
}

func ec2RouteServerItemFrom(in ec2svc.RouteServer) ec2RouteServerItem {
	snsEnabled := in.SnsNotificationsEnabled
	return ec2RouteServerItem{
		AmazonSideASN:           in.AmazonSideASN,
		PersistRoutesDuration:   in.PersistRoutesDuration,
		PersistRoutesState:      in.PersistRoutesState,
		RouteServerID:           in.ID,
		SnsNotificationsEnabled: &snsEnabled,
		SnsTopicARN:             in.SnsTopicARN,
		State:                   in.State,
		TagSet:                  ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
	}
}

func ec2RouteServerEndpointItems(in []ec2svc.RouteServerEndpoint) []ec2RouteServerEndpointItem {
	out := make([]ec2RouteServerEndpointItem, 0, len(in))
	for _, endpoint := range in {
		out = append(out, ec2RouteServerEndpointItemFrom(endpoint))
	}
	return out
}

func ec2RouteServerEndpointItemFrom(in ec2svc.RouteServerEndpoint) ec2RouteServerEndpointItem {
	return ec2RouteServerEndpointItem{
		EniAddress:            in.EniAddress,
		EniID:                 in.EniID,
		FailureReason:         in.FailureReason,
		RouteServerEndpointID: in.ID,
		RouteServerID:         in.RouteServerID,
		State:                 in.State,
		SubnetID:              in.SubnetID,
		TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		VpcID:                 in.VpcID,
	}
}

func ec2RouteServerPeerItems(in []ec2svc.RouteServerPeer) []ec2RouteServerPeerItem {
	out := make([]ec2RouteServerPeerItem, 0, len(in))
	for _, peer := range in {
		out = append(out, ec2RouteServerPeerItemFrom(peer))
	}
	return out
}

func ec2RouteServerPeerItemFrom(in ec2svc.RouteServerPeer) ec2RouteServerPeerItem {
	out := ec2RouteServerPeerItem{
		EndpointEniAddress:    in.EndpointEniAddress,
		EndpointEniID:         in.EndpointEniID,
		FailureReason:         in.FailureReason,
		PeerAddress:           in.PeerAddress,
		RouteServerEndpointID: in.RouteServerEndpointID,
		RouteServerID:         in.RouteServerID,
		RouteServerPeerID:     in.ID,
		State:                 in.State,
		SubnetID:              in.SubnetID,
		TagSet:                ec2TagSet{Items: ec2TagItemsFromMap(in.Tags)},
		VpcID:                 in.VpcID,
	}
	if in.BgpOptions.PeerASN > 0 || in.BgpOptions.PeerLivenessDetection != "" {
		out.BgpOptions = &ec2RouteServerBgpOptionsItem{
			PeerASN:               in.BgpOptions.PeerASN,
			PeerLivenessDetection: in.BgpOptions.PeerLivenessDetection,
		}
	}
	if in.BgpStatus != "" {
		out.BgpStatus = &ec2RouteServerStatusItem{Status: in.BgpStatus}
	}
	if in.BfdStatus != "" {
		out.BfdStatus = &ec2RouteServerStatusItem{Status: in.BfdStatus}
	}
	return out
}

type ec2CreateRouteServerResponse struct {
	XMLName     xml.Name           `xml:"CreateRouteServerResponse"`
	Xmlns       string             `xml:"xmlns,attr"`
	RequestID   string             `xml:"requestId"`
	RouteServer ec2RouteServerItem `xml:"routeServer"`
}

type ec2DeleteRouteServerResponse struct {
	XMLName     xml.Name           `xml:"DeleteRouteServerResponse"`
	Xmlns       string             `xml:"xmlns,attr"`
	RequestID   string             `xml:"requestId"`
	RouteServer ec2RouteServerItem `xml:"routeServer"`
}

type ec2DescribeRouteServersResponse struct {
	XMLName        xml.Name          `xml:"DescribeRouteServersResponse"`
	Xmlns          string            `xml:"xmlns,attr"`
	RequestID      string            `xml:"requestId"`
	NextToken      string            `xml:"nextToken,omitempty"`
	RouteServerSet ec2RouteServerSet `xml:"routeServerSet"`
}

type ec2RouteServerSet struct {
	Items []ec2RouteServerItem `xml:"item"`
}

type ec2RouteServerItem struct {
	AmazonSideASN           int64     `xml:"amazonSideAsn,omitempty"`
	PersistRoutesDuration   *int64    `xml:"persistRoutesDuration,omitempty"`
	PersistRoutesState      string    `xml:"persistRoutesState,omitempty"`
	RouteServerID           string    `xml:"routeServerId,omitempty"`
	SnsNotificationsEnabled *bool     `xml:"snsNotificationsEnabled,omitempty"`
	SnsTopicARN             string    `xml:"snsTopicArn,omitempty"`
	State                   string    `xml:"state,omitempty"`
	TagSet                  ec2TagSet `xml:"tagSet"`
}

type ec2CreateRouteServerEndpointResponse struct {
	XMLName             xml.Name                   `xml:"CreateRouteServerEndpointResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	RouteServerEndpoint ec2RouteServerEndpointItem `xml:"routeServerEndpoint"`
}

type ec2DeleteRouteServerEndpointResponse struct {
	XMLName             xml.Name                   `xml:"DeleteRouteServerEndpointResponse"`
	Xmlns               string                     `xml:"xmlns,attr"`
	RequestID           string                     `xml:"requestId"`
	RouteServerEndpoint ec2RouteServerEndpointItem `xml:"routeServerEndpoint"`
}

type ec2DescribeRouteServerEndpointsResponse struct {
	XMLName                xml.Name                  `xml:"DescribeRouteServerEndpointsResponse"`
	Xmlns                  string                    `xml:"xmlns,attr"`
	RequestID              string                    `xml:"requestId"`
	NextToken              string                    `xml:"nextToken,omitempty"`
	RouteServerEndpointSet ec2RouteServerEndpointSet `xml:"routeServerEndpointSet"`
}

type ec2RouteServerEndpointSet struct {
	Items []ec2RouteServerEndpointItem `xml:"item"`
}

type ec2RouteServerEndpointItem struct {
	EniAddress            string    `xml:"eniAddress,omitempty"`
	EniID                 string    `xml:"eniId,omitempty"`
	FailureReason         string    `xml:"failureReason,omitempty"`
	RouteServerEndpointID string    `xml:"routeServerEndpointId,omitempty"`
	RouteServerID         string    `xml:"routeServerId,omitempty"`
	State                 string    `xml:"state,omitempty"`
	SubnetID              string    `xml:"subnetId,omitempty"`
	TagSet                ec2TagSet `xml:"tagSet"`
	VpcID                 string    `xml:"vpcId,omitempty"`
}

type ec2CreateRouteServerPeerResponse struct {
	XMLName         xml.Name               `xml:"CreateRouteServerPeerResponse"`
	Xmlns           string                 `xml:"xmlns,attr"`
	RequestID       string                 `xml:"requestId"`
	RouteServerPeer ec2RouteServerPeerItem `xml:"routeServerPeer"`
}

type ec2DeleteRouteServerPeerResponse struct {
	XMLName         xml.Name               `xml:"DeleteRouteServerPeerResponse"`
	Xmlns           string                 `xml:"xmlns,attr"`
	RequestID       string                 `xml:"requestId"`
	RouteServerPeer ec2RouteServerPeerItem `xml:"routeServerPeer"`
}

type ec2DescribeRouteServerPeersResponse struct {
	XMLName            xml.Name              `xml:"DescribeRouteServerPeersResponse"`
	Xmlns              string                `xml:"xmlns,attr"`
	RequestID          string                `xml:"requestId"`
	NextToken          string                `xml:"nextToken,omitempty"`
	RouteServerPeerSet ec2RouteServerPeerSet `xml:"routeServerPeerSet"`
}

type ec2RouteServerPeerSet struct {
	Items []ec2RouteServerPeerItem `xml:"item"`
}

type ec2RouteServerPeerItem struct {
	BfdStatus             *ec2RouteServerStatusItem     `xml:"bfdStatus,omitempty"`
	BgpOptions            *ec2RouteServerBgpOptionsItem `xml:"bgpOptions,omitempty"`
	BgpStatus             *ec2RouteServerStatusItem     `xml:"bgpStatus,omitempty"`
	EndpointEniAddress    string                        `xml:"endpointEniAddress,omitempty"`
	EndpointEniID         string                        `xml:"endpointEniId,omitempty"`
	FailureReason         string                        `xml:"failureReason,omitempty"`
	PeerAddress           string                        `xml:"peerAddress,omitempty"`
	RouteServerEndpointID string                        `xml:"routeServerEndpointId,omitempty"`
	RouteServerID         string                        `xml:"routeServerId,omitempty"`
	RouteServerPeerID     string                        `xml:"routeServerPeerId,omitempty"`
	State                 string                        `xml:"state,omitempty"`
	SubnetID              string                        `xml:"subnetId,omitempty"`
	TagSet                ec2TagSet                     `xml:"tagSet"`
	VpcID                 string                        `xml:"vpcId,omitempty"`
}

type ec2RouteServerStatusItem struct {
	Status string `xml:"status,omitempty"`
}

type ec2RouteServerBgpOptionsItem struct {
	PeerASN               int64  `xml:"peerAsn,omitempty"`
	PeerLivenessDetection string `xml:"peerLivenessDetection,omitempty"`
}
