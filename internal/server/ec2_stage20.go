package server

import (
	"encoding/xml"
	"net/http"
	"strings"
)

func (s *Server) handleEC2Stage20Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpnConnection":
		connection, err := s.ec2.ModifyVpnConnection(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			strings.TrimSpace(r.Form.Get("CustomerGatewayId")),
			strings.TrimSpace(r.Form.Get("VpnGatewayId")),
			strings.TrimSpace(r.Form.Get("TransitGatewayId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpnConnectionResponse{
			XMLName:       xml.Name{Local: "ModifyVpnConnectionResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			VpnConnection: ec2VpnConnectionItemFrom(connection),
		})
		return true
	case "ModifyVpnConnectionOptions":
		connection, err := s.ec2.ModifyVpnConnectionOptions(
			strings.TrimSpace(r.Form.Get("VpnConnectionId")),
			parseEC2OptionalString(r.Form.Get("LocalIpv4NetworkCidr")),
			parseEC2OptionalString(r.Form.Get("LocalIpv6NetworkCidr")),
			parseEC2OptionalString(r.Form.Get("RemoteIpv4NetworkCidr")),
			parseEC2OptionalString(r.Form.Get("RemoteIpv6NetworkCidr")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ModifyVpnConnectionOptionsResponse{
			XMLName:       xml.Name{Local: "ModifyVpnConnectionOptionsResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			VpnConnection: ec2VpnConnectionItemFrom(connection),
		})
		return true
	default:
		return false
	}
}

func parseEC2OptionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

type ec2ModifyVpnConnectionResponse struct {
	XMLName       xml.Name
	Xmlns         string               `xml:"xmlns,attr"`
	RequestID     string               `xml:"requestId"`
	VpnConnection ec2VpnConnectionItem `xml:"vpnConnection"`
}

type ec2ModifyVpnConnectionOptionsResponse struct {
	XMLName       xml.Name
	Xmlns         string               `xml:"xmlns,attr"`
	RequestID     string               `xml:"requestId"`
	VpnConnection ec2VpnConnectionItem `xml:"vpnConnection"`
}
