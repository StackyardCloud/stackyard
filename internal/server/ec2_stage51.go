package server

import (
	"encoding/xml"
	"net/http"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage51Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcEndpointServiceConfiguration":
		acceptanceRequired, hasAcceptanceRequired, ok := ec2OptionalBoolFromForm(r.Form, "AcceptanceRequired")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAcceptanceRequired {
			acceptanceRequired = nil
		}

		removePrivateDNSName, hasRemovePrivateDNSName, ok := ec2OptionalBoolFromForm(r.Form, "RemovePrivateDnsName")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasRemovePrivateDNSName {
			removePrivateDNSName = nil
		}

		ret, err := s.ec2.ModifyVpcEndpointServiceConfiguration(
			strings.TrimSpace(r.Form.Get("ServiceId")),
			acceptanceRequired,
			parseEC2MembersOrItemList(r.Form, "AddGatewayLoadBalancerArns"),
			parseEC2MembersOrItemList(r.Form, "AddNetworkLoadBalancerArns"),
			parseEC2MembersOrItemList(r.Form, "AddSupportedIpAddressTypes"),
			parseEC2MembersOrItemList(r.Form, "AddSupportedRegions"),
			parseEC2OptionalString(r.Form.Get("PrivateDnsName")),
			parseEC2MembersOrItemList(r.Form, "RemoveGatewayLoadBalancerArns"),
			parseEC2MembersOrItemList(r.Form, "RemoveNetworkLoadBalancerArns"),
			removePrivateDNSName,
			parseEC2MembersOrItemList(r.Form, "RemoveSupportedIpAddressTypes"),
			parseEC2MembersOrItemList(r.Form, "RemoveSupportedRegions"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVpcEndpointServiceConfigurationResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}
