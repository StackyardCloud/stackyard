package server

import (
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage52Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "ModifyVpcEndpoint":
		privateDNSEnabled, hasPrivateDNSEnabled, ok := ec2OptionalBoolFromForm(r.Form, "PrivateDnsEnabled")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasPrivateDNSEnabled {
			privateDNSEnabled = nil
		}

		resetPolicy, hasResetPolicy, ok := ec2OptionalBoolFromForm(r.Form, "ResetPolicy")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasResetPolicy {
			resetPolicy = nil
		}

		ret, err := s.ec2.ModifyVpcEndpoint(
			strings.TrimSpace(r.Form.Get("VpcEndpointId")),
			parseEC2MembersWithAliases(r.Form, "AddRouteTableId", "AddRouteTableIds"),
			parseEC2MembersWithAliases(r.Form, "AddSecurityGroupId", "AddSecurityGroupIds"),
			parseEC2MembersWithAliases(r.Form, "AddSubnetId", "AddSubnetIds"),
			strings.TrimSpace(r.Form.Get("IpAddressType")),
			parseEC2OptionalString(r.Form.Get("PolicyDocument")),
			privateDNSEnabled,
			parseEC2MembersWithAliases(r.Form, "RemoveRouteTableId", "RemoveRouteTableIds"),
			parseEC2MembersWithAliases(r.Form, "RemoveSecurityGroupId", "RemoveSecurityGroupIds"),
			parseEC2MembersWithAliases(r.Form, "RemoveSubnetId", "RemoveSubnetIds"),
			resetPolicy,
			parseEC2SubnetConfigurationSubnetIDs(r.Form),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyVpcEndpointResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}

func parseEC2MembersWithAliases(values url.Values, keys ...string) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}
	for _, key := range keys {
		for _, value := range parseEC2MembersOrItemList(values, key) {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}

func parseEC2SubnetConfigurationSubnetIDs(values url.Values) []string {
	indices := map[int]struct{}{}
	for key := range values {
		if !strings.HasPrefix(key, "SubnetConfiguration.") {
			continue
		}
		rest := strings.TrimPrefix(key, "SubnetConfiguration.")
		part := rest
		if dot := strings.IndexByte(rest, '.'); dot >= 0 {
			part = rest[:dot]
		}
		idx, err := strconv.Atoi(part)
		if err != nil || idx <= 0 {
			continue
		}
		indices[idx] = struct{}{}
	}
	ordered := make([]int, 0, len(indices))
	for idx := range indices {
		ordered = append(ordered, idx)
	}
	sort.Ints(ordered)
	out := make([]string, 0, len(ordered))
	seen := map[string]struct{}{}
	for _, idx := range ordered {
		value := strings.TrimSpace(values.Get("SubnetConfiguration." + strconv.Itoa(idx) + ".SubnetId"))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
