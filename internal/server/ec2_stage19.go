package server

import (
	"encoding/xml"
	"net/http"
	"sort"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage19Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpnConnection":
		var staticRoutesOnly *bool
		if value := strings.TrimSpace(r.Form.Get("Options.StaticRoutesOnly")); value != "" {
			parsed, ok := parseEC2OptionalBool(value)
			if !ok {
				respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
				return true
			}
			staticRoutesOnly = &parsed
		}

		connection, err := s.ec2.CreateVpnConnection(
			strings.TrimSpace(r.Form.Get("CustomerGatewayId")),
			strings.TrimSpace(r.Form.Get("Type")),
			strings.TrimSpace(r.Form.Get("VpnGatewayId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
			staticRoutesOnly,
			parseEC2TagSpecificationsForResource(r.Form, "vpn-connection"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateVpnConnectionResponse{
			XMLName:       xml.Name{Local: "CreateVpnConnectionResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			VpnConnection: ec2VpnConnectionItemFrom(connection),
		})
		return true
	case "DescribeVpnConnections":
		connections := s.ec2.DescribeVpnConnections(
			parseEC2Members(r.Form, "VpnConnectionId."),
			parseEC2FilterValues(r.Form, "vpn-connection-id"),
			parseEC2FilterValues(r.Form, "customer-gateway-id"),
			parseEC2FilterValues(r.Form, "state"),
			parseEC2FilterValues(r.Form, "type"),
			parseEC2FilterValues(r.Form, "vpn-gateway-id"),
			parseEC2FilterValues(r.Form, "transit-gateway-id"),
			parseEC2FilterValues(r.Form, "customer-gateway-configuration"),
			parseEC2FilterValues(r.Form, "route.destination-cidr-block"),
			parseEC2FilterValues(r.Form, "bgp-asn"),
			parseEC2FilterValues(r.Form, "tag-key"),
			parseEC2BoolFilterValues(parseEC2FilterValues(r.Form, "option.static-routes-only")),
			parseEC2TagValueFilters(r.Form),
		)
		respondEC2XML(w, ec2DescribeVpnConnectionsResponse{
			XMLName:   xml.Name{Local: "DescribeVpnConnectionsResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			VpnConnectionSet: ec2VpnConnectionSet{
				Items: ec2VpnConnectionItems(connections),
			},
		})
		return true
	case "CreateVpnConnectionRoute":
		if err := s.ec2.CreateVpnConnectionRoute(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "CreateVpnConnectionRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteVpnConnectionRoute":
		if err := s.ec2.DeleteVpnConnectionRoute(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
		); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteVpnConnectionRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	case "DeleteVpnConnection":
		if err := s.ec2.DeleteVpnConnection(strings.TrimSpace(r.Form.Get("VpnConnectionId"))); err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "DeleteVpnConnectionResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    true,
		})
		return true
	default:
		return false
	}
}

func parseEC2BoolFilterValues(values []string) []bool {
	out := make([]bool, 0, len(values))
	for _, value := range values {
		parsed, ok := parseEC2OptionalBool(value)
		if !ok {
			continue
		}
		out = append(out, parsed)
	}
	return out
}

func ec2VpnConnectionItems(in []ec2svc.VpnConnection) []ec2VpnConnectionItem {
	out := make([]ec2VpnConnectionItem, 0, len(in))
	for _, connection := range in {
		out = append(out, ec2VpnConnectionItemFrom(connection))
	}
	return out
}

func ec2VpnConnectionItemFrom(in ec2svc.VpnConnection) ec2VpnConnectionItem {
	routes := make([]ec2VpnStaticRouteItem, 0, len(in.Routes))
	for _, route := range in.Routes {
		routes = append(routes, ec2VpnStaticRouteItem{
			DestinationCIDRBlock: route.DestinationCIDRBlock,
			Source:               route.Source,
			State:                route.State,
		})
	}

	telemetryItems := make([]ec2VgwTelemetryItem, 0, len(in.VgwTelemetry))
	for _, telemetry := range in.VgwTelemetry {
		item := ec2VgwTelemetryItem{
			AcceptedRouteCount: telemetry.AcceptedRouteCount,
			CertificateARN:     telemetry.CertificateARN,
			OutsideIPAddress:   telemetry.OutsideIPAddress,
			Status:             telemetry.Status,
			StatusMessage:      telemetry.StatusMessage,
		}
		if telemetry.LastStatusChange != nil {
			item.LastStatusChange = telemetry.LastStatusChange.Format(timeRFC3339UTC)
		}
		telemetryItems = append(telemetryItems, item)
	}

	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	return ec2VpnConnectionItem{
		CustomerGatewayConfiguration: in.CustomerGatewayConfiguration,
		CustomerGatewayID:            in.CustomerGatewayID,
		GatewayAssociationState:      in.GatewayAssociationState,
		Options: ec2VpnConnectionOptionsItem{
			StaticRoutesOnly:      in.Options.StaticRoutesOnly,
			LocalIpv4NetworkCidr:  in.Options.LocalIpv4NetworkCidr,
			LocalIpv6NetworkCidr:  in.Options.LocalIpv6NetworkCidr,
			RemoteIpv4NetworkCidr: in.Options.RemoteIpv4NetworkCidr,
			RemoteIpv6NetworkCidr: in.Options.RemoteIpv6NetworkCidr,
		},
		RouteSet:         ec2VpnStaticRouteSet{Items: routes},
		State:            in.State,
		TagSet:           ec2TagSet{Items: tags},
		TransitGatewayID: in.TransitGatewayID,
		Type:             in.Type,
		VgwTelemetrySet:  ec2VgwTelemetrySet{Items: telemetryItems},
		VpnConnectionID:  in.ID,
		VpnGatewayID:     in.VpnGatewayID,
	}
}

type ec2CreateVpnConnectionResponse struct {
	XMLName       xml.Name
	Xmlns         string               `xml:"xmlns,attr"`
	RequestID     string               `xml:"requestId"`
	VpnConnection ec2VpnConnectionItem `xml:"vpnConnection"`
}

type ec2DescribeVpnConnectionsResponse struct {
	XMLName          xml.Name            `xml:"DescribeVpnConnectionsResponse"`
	Xmlns            string              `xml:"xmlns,attr"`
	RequestID        string              `xml:"requestId"`
	VpnConnectionSet ec2VpnConnectionSet `xml:"vpnConnectionSet"`
}

type ec2VpnConnectionSet struct {
	Items []ec2VpnConnectionItem `xml:"item"`
}

type ec2VpnConnectionItem struct {
	CustomerGatewayConfiguration string                      `xml:"customerGatewayConfiguration,omitempty"`
	CustomerGatewayID            string                      `xml:"customerGatewayId,omitempty"`
	GatewayAssociationState      string                      `xml:"gatewayAssociationState,omitempty"`
	Options                      ec2VpnConnectionOptionsItem `xml:"options,omitempty"`
	RouteSet                     ec2VpnStaticRouteSet        `xml:"routes,omitempty"`
	State                        string                      `xml:"state,omitempty"`
	TagSet                       ec2TagSet                   `xml:"tagSet,omitempty"`
	TransitGatewayID             string                      `xml:"transitGatewayId,omitempty"`
	Type                         string                      `xml:"type,omitempty"`
	VgwTelemetrySet              ec2VgwTelemetrySet          `xml:"vgwTelemetry,omitempty"`
	VpnConnectionID              string                      `xml:"vpnConnectionId,omitempty"`
	VpnGatewayID                 string                      `xml:"vpnGatewayId,omitempty"`
}

type ec2VpnConnectionOptionsItem struct {
	StaticRoutesOnly      bool   `xml:"staticRoutesOnly,omitempty"`
	LocalIpv4NetworkCidr  string `xml:"localIpv4NetworkCidr,omitempty"`
	LocalIpv6NetworkCidr  string `xml:"localIpv6NetworkCidr,omitempty"`
	RemoteIpv4NetworkCidr string `xml:"remoteIpv4NetworkCidr,omitempty"`
	RemoteIpv6NetworkCidr string `xml:"remoteIpv6NetworkCidr,omitempty"`
}

type ec2VpnStaticRouteSet struct {
	Items []ec2VpnStaticRouteItem `xml:"item"`
}

type ec2VpnStaticRouteItem struct {
	DestinationCIDRBlock string `xml:"destinationCidrBlock,omitempty"`
	Source               string `xml:"source,omitempty"`
	State                string `xml:"state,omitempty"`
}

type ec2VgwTelemetrySet struct {
	Items []ec2VgwTelemetryItem `xml:"item"`
}

type ec2VgwTelemetryItem struct {
	AcceptedRouteCount int32  `xml:"acceptedRouteCount,omitempty"`
	CertificateARN     string `xml:"certificateArn,omitempty"`
	LastStatusChange   string `xml:"lastStatusChange,omitempty"`
	OutsideIPAddress   string `xml:"outsideIpAddress,omitempty"`
	Status             string `xml:"status,omitempty"`
	StatusMessage      string `xml:"statusMessage,omitempty"`
}
