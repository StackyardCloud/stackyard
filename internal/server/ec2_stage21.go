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

func (s *Server) handleEC2Stage21Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateClientVpnEndpoint":
		vpnPort, ok := parseEC2OptionalInt32(r.Form.Get("VpnPort"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		sessionTimeoutHours, ok := parseEC2OptionalInt32(r.Form.Get("SessionTimeoutHours"))
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		splitTunnelValue, splitTunnelSet := parseEC2OptionalBool(r.Form.Get("SplitTunnel"))
		disconnectOnSessionTimeoutValue, disconnectOnSessionTimeoutSet := parseEC2OptionalBool(r.Form.Get("DisconnectOnSessionTimeout"))

		var splitTunnel *bool
		if splitTunnelSet {
			splitTunnel = &splitTunnelValue
		}
		var disconnectOnSessionTimeout *bool
		if disconnectOnSessionTimeoutSet {
			disconnectOnSessionTimeout = &disconnectOnSessionTimeoutValue
		}

		endpoint, err := s.ec2.CreateClientVpnEndpoint(
			strings.TrimSpace(r.Form.Get("Authentication.1.Type")),
			strings.TrimSpace(r.Form.Get("ClientCidrBlock")),
			strings.TrimSpace(r.Form.Get("ServerCertificateArn")),
			strings.TrimSpace(r.Form.Get("Description")),
			strings.TrimSpace(r.Form.Get("TransportProtocol")),
			vpnPort,
			splitTunnel,
			parseEC2Members(r.Form, "SecurityGroupId."),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2Members(r.Form, "DnsServers."),
			sessionTimeoutHours,
			disconnectOnSessionTimeout,
			strings.TrimSpace(r.Form.Get("SelfServicePortal")),
			parseEC2TagSpecificationsForResource(r.Form, "client-vpn-endpoint"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2CreateClientVpnEndpointResponse{
			XMLName:             xml.Name{Local: "CreateClientVpnEndpointResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			ClientVpnEndpointID: endpoint.ID,
			DnsName:             endpoint.DnsName,
			Status:              ec2ClientVpnEndpointStatusItemFrom(endpoint.Status),
		})
		return true
	case "DescribeClientVpnEndpoints":
		endpoints := s.ec2.DescribeClientVpnEndpoints(
			parseEC2Members(r.Form, "ClientVpnEndpointId."),
			parseEC2FilterValues(r.Form, "endpoint-id"),
			parseEC2FilterValues(r.Form, "transport-protocol"),
		)
		respondEC2XML(w, ec2DescribeClientVpnEndpointsResponse{
			XMLName:              xml.Name{Local: "DescribeClientVpnEndpointsResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			ClientVpnEndpointSet: ec2ClientVpnEndpointSet{Items: ec2ClientVpnEndpointItems(endpoints)},
		})
		return true
	case "ModifyClientVpnEndpoint":
		endpointID := strings.TrimSpace(r.Form.Get("ClientVpnEndpointId"))
		description := ec2OptionalStringPointerFromForm(r.Form, "Description")

		splitTunnel, hasSplitTunnel, ok := ec2OptionalBoolFromForm(r.Form, "SplitTunnel")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasSplitTunnel {
			splitTunnel = nil
		}

		hasSecurityGroupIDs := hasEC2PrefixedField(r.Form, "SecurityGroupId.")
		securityGroupIDs := parseEC2Members(r.Form, "SecurityGroupId.")

		dnsServers, hasDnsServers, ok := ec2ModifyClientVpnDnsServersFromForm(r.Form)
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}

		sessionTimeoutHours, hasSessionTimeoutHours, ok := ec2OptionalInt32FromForm(r.Form, "SessionTimeoutHours")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasSessionTimeoutHours {
			sessionTimeoutHours = nil
		}

		disconnectOnSessionTimeout, hasDisconnectOnSessionTimeout, ok := ec2OptionalBoolFromForm(r.Form, "DisconnectOnSessionTimeout")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasDisconnectOnSessionTimeout {
			disconnectOnSessionTimeout = nil
		}

		selfServicePortal := ec2OptionalStringPointerFromForm(r.Form, "SelfServicePortal")
		vpcID := ec2OptionalStringPointerFromForm(r.Form, "VpcId")

		vpnPort, hasVpnPort, ok := ec2OptionalInt32FromForm(r.Form, "VpnPort")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasVpnPort {
			vpnPort = nil
		}

		ret, err := s.ec2.ModifyClientVpnEndpoint(
			endpointID,
			description,
			splitTunnel,
			securityGroupIDs,
			hasSecurityGroupIDs,
			dnsServers,
			hasDnsServers,
			sessionTimeoutHours,
			disconnectOnSessionTimeout,
			selfServicePortal,
			vpcID,
			vpnPort,
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ModifyClientVpnEndpointResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	case "DeleteClientVpnEndpoint":
		status, err := s.ec2.DeleteClientVpnEndpoint(strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteClientVpnEndpointResponse{
			XMLName:   xml.Name{Local: "DeleteClientVpnEndpointResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Status:    ec2ClientVpnEndpointStatusItemFrom(status),
		})
		return true
	case "CreateClientVpnRoute":
		status, err := s.ec2.CreateClientVpnRoute(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			strings.TrimSpace(r.Form.Get("TargetVpcSubnetId")),
			strings.TrimSpace(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2CreateClientVpnRouteResponse{
			XMLName:   xml.Name{Local: "CreateClientVpnRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Status:    ec2ClientVpnRouteStatusItemFrom(status),
		})
		return true
	case "DescribeClientVpnRoutes":
		routes, err := s.ec2.DescribeClientVpnRoutes(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			parseEC2FilterValues(r.Form, "destination-cidr"),
			parseEC2FilterValues(r.Form, "origin"),
			parseEC2FilterValues(r.Form, "target-subnet"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeClientVpnRoutesResponse{
			XMLName:   xml.Name{Local: "DescribeClientVpnRoutesResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Routes:    ec2ClientVpnRouteSet{Items: ec2ClientVpnRouteItems(routes)},
		})
		return true
	case "DeleteClientVpnRoute":
		status, err := s.ec2.DeleteClientVpnRoute(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("DestinationCidrBlock")),
			strings.TrimSpace(r.Form.Get("TargetVpcSubnetId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DeleteClientVpnRouteResponse{
			XMLName:   xml.Name{Local: "DeleteClientVpnRouteResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Status:    ec2ClientVpnRouteStatusItemFrom(status),
		})
		return true
	case "AssociateClientVpnTargetNetwork":
		targetNetwork, err := s.ec2.AssociateClientVpnTargetNetwork(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("SubnetId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AssociateClientVpnTargetNetworkResponse{
			XMLName:       xml.Name{Local: "AssociateClientVpnTargetNetworkResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			AssociationID: targetNetwork.AssociationID,
			Status:        ec2ClientVpnAssociationStatusItemFrom(targetNetwork.Status),
		})
		return true
	case "DisassociateClientVpnTargetNetwork":
		targetNetwork, err := s.ec2.DisassociateClientVpnTargetNetwork(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("AssociationId")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DisassociateClientVpnTargetNetworkResponse{
			XMLName:       xml.Name{Local: "DisassociateClientVpnTargetNetworkResponse"},
			Xmlns:         ec2Namespace,
			RequestID:     "stackyard-request",
			AssociationID: targetNetwork.AssociationID,
			Status:        ec2ClientVpnAssociationStatusItemFrom(targetNetwork.Status),
		})
		return true
	case "ApplySecurityGroupsToClientVpnTargetNetwork":
		securityGroupIDs, err := s.ec2.ApplySecurityGroupsToClientVpnTargetNetwork(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("VpcId")),
			parseEC2Members(r.Form, "SecurityGroupId."),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ApplySecurityGroupsToClientVpnTargetNetworkResponse{
			XMLName:          xml.Name{Local: "ApplySecurityGroupsToClientVpnTargetNetworkResponse"},
			Xmlns:            ec2Namespace,
			RequestID:        "stackyard-request",
			SecurityGroupIDs: ec2ValueStringSet{Items: append([]string(nil), securityGroupIDs...)},
		})
		return true
	case "AuthorizeClientVpnIngress":
		status, err := s.ec2.AuthorizeClientVpnIngress(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("TargetNetworkCidr")),
			strings.TrimSpace(r.Form.Get("AccessGroupId")),
			parseEC2Bool(r.Form.Get("AuthorizeAllGroups"), false),
			strings.TrimSpace(r.Form.Get("Description")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2AuthorizeClientVpnIngressResponse{
			XMLName:   xml.Name{Local: "AuthorizeClientVpnIngressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Status:    ec2ClientVpnAuthorizationRuleStatusItemFrom(status),
		})
		return true
	case "RevokeClientVpnIngress":
		status, err := s.ec2.RevokeClientVpnIngress(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("TargetNetworkCidr")),
			strings.TrimSpace(r.Form.Get("AccessGroupId")),
			parseEC2Bool(r.Form.Get("RevokeAllGroups"), false),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2RevokeClientVpnIngressResponse{
			XMLName:   xml.Name{Local: "RevokeClientVpnIngressResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Status:    ec2ClientVpnAuthorizationRuleStatusItemFrom(status),
		})
		return true
	case "TerminateClientVpnConnections":
		result, err := s.ec2.TerminateClientVpnConnections(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("ConnectionId")),
			strings.TrimSpace(r.Form.Get("Username")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2TerminateClientVpnConnectionsResponse{
			XMLName:             xml.Name{Local: "TerminateClientVpnConnectionsResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			ClientVpnEndpointID: result.ClientVpnEndpointID,
			ConnectionStatuses:  ec2TerminateClientVpnConnectionStatusSet{Items: ec2TerminateClientVpnConnectionStatusItems(result.ConnectionStatuses)},
			Username:            result.Username,
		})
		return true
	case "DescribeClientVpnAuthorizationRules":
		rules, err := s.ec2.DescribeClientVpnAuthorizationRules(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			parseEC2FilterValues(r.Form, "description"),
			parseEC2FilterValues(r.Form, "destination-cidr"),
			parseEC2FilterValues(r.Form, "group-id"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeClientVpnAuthorizationRulesResponse{
			XMLName:           xml.Name{Local: "DescribeClientVpnAuthorizationRulesResponse"},
			Xmlns:             ec2Namespace,
			RequestID:         "stackyard-request",
			AuthorizationRule: ec2ClientVpnAuthorizationRuleSet{Items: ec2ClientVpnAuthorizationRuleItems(rules)},
		})
		return true
	case "DescribeClientVpnConnections":
		connections, err := s.ec2.DescribeClientVpnConnections(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			parseEC2FilterValues(r.Form, "connection-id"),
			parseEC2FilterValues(r.Form, "username"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeClientVpnConnectionsResponse{
			XMLName:     xml.Name{Local: "DescribeClientVpnConnectionsResponse"},
			Xmlns:       ec2Namespace,
			RequestID:   "stackyard-request",
			Connections: ec2ClientVpnConnectionSet{Items: ec2ClientVpnConnectionItems(connections)},
		})
		return true
	case "DescribeClientVpnTargetNetworks":
		targetNetworks, err := s.ec2.DescribeClientVpnTargetNetworks(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			parseEC2Members(r.Form, "AssociationIds."),
			parseEC2FilterValues(r.Form, "association-id"),
			parseEC2FilterValues(r.Form, "target-network-id"),
			parseEC2FilterValues(r.Form, "vpc-id"),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2DescribeClientVpnTargetNetworksResponse{
			XMLName:                 xml.Name{Local: "DescribeClientVpnTargetNetworksResponse"},
			Xmlns:                   ec2Namespace,
			RequestID:               "stackyard-request",
			ClientVpnTargetNetworks: ec2ClientVpnTargetNetworkSet{Items: ec2ClientVpnTargetNetworkItems(targetNetworks)},
		})
		return true
	case "ExportClientVpnClientConfiguration":
		clientConfiguration, err := s.ec2.ExportClientVpnClientConfiguration(strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ExportClientVpnClientConfigurationResponse{
			XMLName:             xml.Name{Local: "ExportClientVpnClientConfigurationResponse"},
			Xmlns:               ec2Namespace,
			RequestID:           "stackyard-request",
			ClientConfiguration: clientConfiguration,
		})
		return true
	case "ExportClientVpnClientCertificateRevocationList":
		certificateRevocationList, status, err := s.ec2.ExportClientVpnClientCertificateRevocationList(strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")))
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2ExportClientVpnClientCertificateRevocationListResponse{
			XMLName:                   xml.Name{Local: "ExportClientVpnClientCertificateRevocationListResponse"},
			Xmlns:                     ec2Namespace,
			RequestID:                 "stackyard-request",
			CertificateRevocationList: certificateRevocationList,
			Status:                    ec2ClientCertificateRevocationListStatusItemFrom(status),
		})
		return true
	case "ImportClientVpnClientCertificateRevocationList":
		ret, err := s.ec2.ImportClientVpnClientCertificateRevocationList(
			strings.TrimSpace(r.Form.Get("ClientVpnEndpointId")),
			strings.TrimSpace(r.Form.Get("CertificateRevocationList")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}
		respondEC2XML(w, ec2SimpleReturnResponse{
			XMLName:   xml.Name{Local: "ImportClientVpnClientCertificateRevocationListResponse"},
			Xmlns:     ec2Namespace,
			RequestID: "stackyard-request",
			Return:    ret,
		})
		return true
	default:
		return false
	}
}

func parseEC2OptionalInt32(value string) (*int32, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return nil, false
	}
	n32 := int32(n)
	return &n32, true
}

func ec2OptionalBoolFromForm(values url.Values, key string) (*bool, bool, bool) {
	if !hasEC2Field(values, key) {
		return nil, false, true
	}
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return nil, true, false
	}
	parsed := parseEC2Bool(raw, false)
	if strings.EqualFold(raw, "true") || raw == "1" || strings.EqualFold(raw, "yes") || strings.EqualFold(raw, "on") ||
		strings.EqualFold(raw, "false") || raw == "0" || strings.EqualFold(raw, "no") || strings.EqualFold(raw, "off") {
		return &parsed, true, true
	}
	return nil, true, false
}

func ec2OptionalInt32FromForm(values url.Values, key string) (*int32, bool, bool) {
	if !hasEC2Field(values, key) {
		return nil, false, true
	}
	parsed, ok := parseEC2OptionalInt32(values.Get(key))
	if !ok {
		return nil, true, false
	}
	return parsed, true, true
}

func ec2OptionalStringPointerFromForm(values url.Values, key string) *string {
	if !hasEC2Field(values, key) {
		return nil
	}
	value := strings.TrimSpace(values.Get(key))
	return &value
}

func hasEC2Field(values url.Values, key string) bool {
	_, ok := values[key]
	return ok
}

func hasEC2PrefixedField(values url.Values, prefix string) bool {
	for key := range values {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func ec2ModifyClientVpnDnsServersFromForm(values url.Values) ([]string, bool, bool) {
	hasCustomDnsServers := hasEC2PrefixedField(values, "DnsServers.CustomDnsServers.")
	customDnsServers := parseEC2Members(values, "DnsServers.CustomDnsServers.")
	hasEnabled := hasEC2Field(values, "DnsServers.Enabled")
	if !hasEnabled && !hasCustomDnsServers {
		return nil, false, true
	}
	if hasEnabled {
		enabledRaw := strings.TrimSpace(values.Get("DnsServers.Enabled"))
		if enabledRaw != "" {
			if strings.EqualFold(enabledRaw, "false") || enabledRaw == "0" || strings.EqualFold(enabledRaw, "no") || strings.EqualFold(enabledRaw, "off") {
				return []string{}, true, true
			}
			if !(strings.EqualFold(enabledRaw, "true") || enabledRaw == "1" || strings.EqualFold(enabledRaw, "yes") || strings.EqualFold(enabledRaw, "on")) {
				return nil, true, false
			}
		}
	}
	return customDnsServers, true, true
}

func ec2ClientVpnEndpointItems(in []ec2svc.ClientVpnEndpoint) []ec2ClientVpnEndpointItem {
	out := make([]ec2ClientVpnEndpointItem, 0, len(in))
	for _, endpoint := range in {
		out = append(out, ec2ClientVpnEndpointItemFrom(endpoint))
	}
	return out
}

func ec2ClientVpnEndpointItemFrom(in ec2svc.ClientVpnEndpoint) ec2ClientVpnEndpointItem {
	tags := make([]ec2TagItem, 0, len(in.Tags))
	for key, value := range in.Tags {
		tags = append(tags, ec2TagItem{Key: key, Value: value})
	}
	sort.Slice(tags, func(i, j int) bool { return tags[i].Key < tags[j].Key })

	item := ec2ClientVpnEndpointItem{
		ClientCidrBlock:            in.ClientCidrBlock,
		ClientVpnEndpointID:        in.ID,
		Description:                in.Description,
		DisconnectOnSessionTimeout: in.DisconnectOnSessionTimeout,
		DnsName:                    in.DnsName,
		DnsServers:                 ec2ValueStringSet{Items: append([]string(nil), in.DnsServers...)},
		SecurityGroupIDs:           ec2ValueStringSet{Items: append([]string(nil), in.SecurityGroupIDs...)},
		SelfServicePortalURL:       in.SelfServicePortalURL,
		ServerCertificateARN:       in.ServerCertificateARN,
		SessionTimeoutHours:        in.SessionTimeoutHours,
		SplitTunnel:                in.SplitTunnel,
		Status:                     ec2ClientVpnEndpointStatusItemFrom(in.Status),
		TagSet:                     ec2TagSet{Items: tags},
		TransportProtocol:          in.TransportProtocol,
		VpcID:                      in.VpcID,
		VpnPort:                    in.VpnPort,
		CreationTime:               in.CreationTime.UTC().Format(timeRFC3339UTC),
	}
	if in.DeletionTime != nil {
		item.DeletionTime = in.DeletionTime.UTC().Format(timeRFC3339UTC)
	}
	return item
}

func ec2ClientVpnRouteItems(in []ec2svc.ClientVpnRoute) []ec2ClientVpnRouteItem {
	out := make([]ec2ClientVpnRouteItem, 0, len(in))
	for _, route := range in {
		out = append(out, ec2ClientVpnRouteItem{
			ClientVpnEndpointID: route.ClientVpnEndpointID,
			Description:         route.Description,
			DestinationCIDR:     route.DestinationCIDR,
			Origin:              route.Origin,
			Status:              ec2ClientVpnRouteStatusItemFrom(route.Status),
			TargetSubnet:        route.TargetSubnet,
			Type:                route.Type,
		})
	}
	return out
}

func ec2ClientVpnTargetNetworkItems(in []ec2svc.ClientVpnTargetNetwork) []ec2ClientVpnTargetNetworkItem {
	out := make([]ec2ClientVpnTargetNetworkItem, 0, len(in))
	for _, targetNetwork := range in {
		out = append(out, ec2ClientVpnTargetNetworkItem{
			AssociationID:       targetNetwork.AssociationID,
			ClientVpnEndpointID: targetNetwork.ClientVpnEndpointID,
			SecurityGroups:      ec2ValueStringSet{Items: append([]string(nil), targetNetwork.SecurityGroupIDs...)},
			Status:              ec2ClientVpnAssociationStatusItemFrom(targetNetwork.Status),
			TargetNetworkID:     targetNetwork.TargetNetworkID,
			VpcID:               targetNetwork.VpcID,
		})
	}
	return out
}

func ec2ClientVpnAuthorizationRuleItems(in []ec2svc.ClientVpnAuthorizationRule) []ec2ClientVpnAuthorizationRuleItem {
	out := make([]ec2ClientVpnAuthorizationRuleItem, 0, len(in))
	for _, rule := range in {
		out = append(out, ec2ClientVpnAuthorizationRuleItem{
			AccessAll:           rule.AuthorizeAllGroups,
			ClientVpnEndpointID: rule.ClientVpnEndpointID,
			Description:         rule.Description,
			DestinationCIDR:     rule.DestinationCIDR,
			GroupID:             rule.GroupID,
			Status:              ec2ClientVpnAuthorizationRuleStatusItemFrom(rule.Status),
		})
	}
	return out
}

func ec2ClientVpnConnectionItems(in []ec2svc.ClientVpnConnection) []ec2ClientVpnConnectionItem {
	out := make([]ec2ClientVpnConnectionItem, 0, len(in))
	for _, connection := range in {
		item := ec2ClientVpnConnectionItem{
			ClientIP:                  connection.ClientIP,
			ClientVpnEndpointID:       connection.ClientVpnEndpointID,
			CommonName:                connection.CommonName,
			ConnectionEstablishedTime: connection.ConnectionEstablishedTime.UTC().Format(timeRFC3339UTC),
			ConnectionID:              connection.ConnectionID,
			EgressBytes:               connection.EgressBytes,
			EgressPackets:             connection.EgressPackets,
			IngressBytes:              connection.IngressBytes,
			IngressPackets:            connection.IngressPackets,
			PostureComplianceStatuses: ec2ValueStringSet{Items: append([]string(nil), connection.PostureComplianceStatuses...)},
			Status:                    ec2ClientVpnConnectionStatusItemFrom(connection.Status),
			Timestamp:                 connection.Timestamp.UTC().Format(timeRFC3339UTC),
			Username:                  connection.Username,
		}
		if connection.ConnectionEndTime != nil {
			item.ConnectionEndTime = connection.ConnectionEndTime.UTC().Format(timeRFC3339UTC)
		}
		out = append(out, item)
	}
	return out
}

func ec2TerminateClientVpnConnectionStatusItems(in []ec2svc.TerminateClientVpnConnectionStatus) []ec2TerminateClientVpnConnectionStatusItem {
	out := make([]ec2TerminateClientVpnConnectionStatusItem, 0, len(in))
	for _, status := range in {
		out = append(out, ec2TerminateClientVpnConnectionStatusItem{
			ConnectionID:   status.ConnectionID,
			CurrentStatus:  ec2ClientVpnConnectionStatusItemFrom(status.CurrentStatus),
			PreviousStatus: ec2ClientVpnConnectionStatusItemFrom(status.PreviousStatus),
		})
	}
	return out
}

func ec2ClientVpnEndpointStatusItemFrom(in ec2svc.ClientVpnEndpointStatus) ec2ClientVpnEndpointStatusItem {
	return ec2ClientVpnEndpointStatusItem{Code: in.Code, Message: in.Message}
}

func ec2ClientVpnRouteStatusItemFrom(in ec2svc.ClientVpnRouteStatus) ec2ClientVpnRouteStatusItem {
	return ec2ClientVpnRouteStatusItem{Code: in.Code, Message: in.Message}
}

func ec2ClientVpnAssociationStatusItemFrom(in ec2svc.ClientVpnAssociationStatus) ec2ClientVpnAssociationStatusItem {
	return ec2ClientVpnAssociationStatusItem{Code: in.Code, Message: in.Message}
}

func ec2ClientVpnAuthorizationRuleStatusItemFrom(in ec2svc.ClientVpnAuthorizationRuleStatus) ec2ClientVpnAuthorizationRuleStatusItem {
	return ec2ClientVpnAuthorizationRuleStatusItem{Code: in.Code, Message: in.Message}
}

func ec2ClientVpnConnectionStatusItemFrom(in ec2svc.ClientVpnConnectionStatus) ec2ClientVpnConnectionStatusItem {
	return ec2ClientVpnConnectionStatusItem{Code: in.Code, Message: in.Message}
}

func ec2ClientCertificateRevocationListStatusItemFrom(in ec2svc.ClientCertificateRevocationListStatus) ec2ClientCertificateRevocationListStatusItem {
	return ec2ClientCertificateRevocationListStatusItem{Code: in.Code, Message: in.Message}
}

type ec2CreateClientVpnEndpointResponse struct {
	XMLName             xml.Name
	Xmlns               string                         `xml:"xmlns,attr"`
	RequestID           string                         `xml:"requestId"`
	ClientVpnEndpointID string                         `xml:"clientVpnEndpointId,omitempty"`
	DnsName             string                         `xml:"dnsName,omitempty"`
	Status              ec2ClientVpnEndpointStatusItem `xml:"status,omitempty"`
}

type ec2DescribeClientVpnEndpointsResponse struct {
	XMLName              xml.Name
	Xmlns                string                  `xml:"xmlns,attr"`
	RequestID            string                  `xml:"requestId"`
	ClientVpnEndpointSet ec2ClientVpnEndpointSet `xml:"clientVpnEndpoint"`
	NextToken            string                  `xml:"nextToken,omitempty"`
}

type ec2DeleteClientVpnEndpointResponse struct {
	XMLName   xml.Name
	Xmlns     string                         `xml:"xmlns,attr"`
	RequestID string                         `xml:"requestId"`
	Status    ec2ClientVpnEndpointStatusItem `xml:"status"`
}

type ec2CreateClientVpnRouteResponse struct {
	XMLName   xml.Name
	Xmlns     string                      `xml:"xmlns,attr"`
	RequestID string                      `xml:"requestId"`
	Status    ec2ClientVpnRouteStatusItem `xml:"status"`
}

type ec2DescribeClientVpnRoutesResponse struct {
	XMLName   xml.Name
	Xmlns     string               `xml:"xmlns,attr"`
	RequestID string               `xml:"requestId"`
	Routes    ec2ClientVpnRouteSet `xml:"routes"`
	NextToken string               `xml:"nextToken,omitempty"`
}

type ec2DeleteClientVpnRouteResponse struct {
	XMLName   xml.Name
	Xmlns     string                      `xml:"xmlns,attr"`
	RequestID string                      `xml:"requestId"`
	Status    ec2ClientVpnRouteStatusItem `xml:"status"`
}

type ec2AssociateClientVpnTargetNetworkResponse struct {
	XMLName       xml.Name
	Xmlns         string                            `xml:"xmlns,attr"`
	RequestID     string                            `xml:"requestId"`
	AssociationID string                            `xml:"associationId,omitempty"`
	Status        ec2ClientVpnAssociationStatusItem `xml:"status,omitempty"`
}

type ec2DisassociateClientVpnTargetNetworkResponse struct {
	XMLName       xml.Name
	Xmlns         string                            `xml:"xmlns,attr"`
	RequestID     string                            `xml:"requestId"`
	AssociationID string                            `xml:"associationId,omitempty"`
	Status        ec2ClientVpnAssociationStatusItem `xml:"status,omitempty"`
}

type ec2ApplySecurityGroupsToClientVpnTargetNetworkResponse struct {
	XMLName          xml.Name
	Xmlns            string            `xml:"xmlns,attr"`
	RequestID        string            `xml:"requestId"`
	SecurityGroupIDs ec2ValueStringSet `xml:"securityGroupIds"`
}

type ec2AuthorizeClientVpnIngressResponse struct {
	XMLName   xml.Name
	Xmlns     string                                  `xml:"xmlns,attr"`
	RequestID string                                  `xml:"requestId"`
	Status    ec2ClientVpnAuthorizationRuleStatusItem `xml:"status"`
}

type ec2RevokeClientVpnIngressResponse struct {
	XMLName   xml.Name
	Xmlns     string                                  `xml:"xmlns,attr"`
	RequestID string                                  `xml:"requestId"`
	Status    ec2ClientVpnAuthorizationRuleStatusItem `xml:"status"`
}

type ec2TerminateClientVpnConnectionsResponse struct {
	XMLName             xml.Name
	Xmlns               string                                   `xml:"xmlns,attr"`
	RequestID           string                                   `xml:"requestId"`
	ClientVpnEndpointID string                                   `xml:"clientVpnEndpointId,omitempty"`
	ConnectionStatuses  ec2TerminateClientVpnConnectionStatusSet `xml:"connectionStatuses"`
	Username            string                                   `xml:"username,omitempty"`
}

type ec2DescribeClientVpnAuthorizationRulesResponse struct {
	XMLName           xml.Name
	Xmlns             string                           `xml:"xmlns,attr"`
	RequestID         string                           `xml:"requestId"`
	AuthorizationRule ec2ClientVpnAuthorizationRuleSet `xml:"authorizationRule"`
	NextToken         string                           `xml:"nextToken,omitempty"`
}

type ec2DescribeClientVpnConnectionsResponse struct {
	XMLName     xml.Name
	Xmlns       string                    `xml:"xmlns,attr"`
	RequestID   string                    `xml:"requestId"`
	Connections ec2ClientVpnConnectionSet `xml:"connections"`
	NextToken   string                    `xml:"nextToken,omitempty"`
}

type ec2DescribeClientVpnTargetNetworksResponse struct {
	XMLName                 xml.Name
	Xmlns                   string                       `xml:"xmlns,attr"`
	RequestID               string                       `xml:"requestId"`
	ClientVpnTargetNetworks ec2ClientVpnTargetNetworkSet `xml:"clientVpnTargetNetworks"`
	NextToken               string                       `xml:"nextToken,omitempty"`
}

type ec2ExportClientVpnClientConfigurationResponse struct {
	XMLName             xml.Name
	Xmlns               string `xml:"xmlns,attr"`
	RequestID           string `xml:"requestId"`
	ClientConfiguration string `xml:"clientConfiguration,omitempty"`
}

type ec2ExportClientVpnClientCertificateRevocationListResponse struct {
	XMLName                   xml.Name
	Xmlns                     string                                       `xml:"xmlns,attr"`
	RequestID                 string                                       `xml:"requestId"`
	CertificateRevocationList string                                       `xml:"certificateRevocationList,omitempty"`
	Status                    ec2ClientCertificateRevocationListStatusItem `xml:"status,omitempty"`
}

type ec2ClientVpnEndpointSet struct {
	Items []ec2ClientVpnEndpointItem `xml:"item"`
}

type ec2ClientVpnEndpointItem struct {
	ClientCidrBlock            string                         `xml:"clientCidrBlock,omitempty"`
	ClientVpnEndpointID        string                         `xml:"clientVpnEndpointId,omitempty"`
	Description                string                         `xml:"description,omitempty"`
	DisconnectOnSessionTimeout bool                           `xml:"disconnectOnSessionTimeout"`
	DnsName                    string                         `xml:"dnsName,omitempty"`
	DnsServers                 ec2ValueStringSet              `xml:"dnsServer"`
	SecurityGroupIDs           ec2ValueStringSet              `xml:"securityGroupIdSet"`
	SelfServicePortalURL       string                         `xml:"selfServicePortalUrl,omitempty"`
	ServerCertificateARN       string                         `xml:"serverCertificateArn,omitempty"`
	SessionTimeoutHours        int32                          `xml:"sessionTimeoutHours"`
	SplitTunnel                bool                           `xml:"splitTunnel"`
	Status                     ec2ClientVpnEndpointStatusItem `xml:"status"`
	TagSet                     ec2TagSet                      `xml:"tagSet"`
	TransportProtocol          string                         `xml:"transportProtocol,omitempty"`
	VpcID                      string                         `xml:"vpcId,omitempty"`
	VpnPort                    int32                          `xml:"vpnPort"`
	CreationTime               string                         `xml:"creationTime,omitempty"`
	DeletionTime               string                         `xml:"deletionTime,omitempty"`
}

type ec2ClientVpnRouteSet struct {
	Items []ec2ClientVpnRouteItem `xml:"item"`
}

type ec2ClientVpnRouteItem struct {
	ClientVpnEndpointID string                      `xml:"clientVpnEndpointId,omitempty"`
	Description         string                      `xml:"description,omitempty"`
	DestinationCIDR     string                      `xml:"destinationCidr,omitempty"`
	Origin              string                      `xml:"origin,omitempty"`
	Status              ec2ClientVpnRouteStatusItem `xml:"status,omitempty"`
	TargetSubnet        string                      `xml:"targetSubnet,omitempty"`
	Type                string                      `xml:"type,omitempty"`
}

type ec2ClientVpnTargetNetworkSet struct {
	Items []ec2ClientVpnTargetNetworkItem `xml:"item"`
}

type ec2ClientVpnTargetNetworkItem struct {
	AssociationID       string                            `xml:"associationId,omitempty"`
	ClientVpnEndpointID string                            `xml:"clientVpnEndpointId,omitempty"`
	SecurityGroups      ec2ValueStringSet                 `xml:"securityGroups"`
	Status              ec2ClientVpnAssociationStatusItem `xml:"status,omitempty"`
	TargetNetworkID     string                            `xml:"targetNetworkId,omitempty"`
	VpcID               string                            `xml:"vpcId,omitempty"`
}

type ec2ClientVpnAuthorizationRuleSet struct {
	Items []ec2ClientVpnAuthorizationRuleItem `xml:"item"`
}

type ec2ClientVpnAuthorizationRuleItem struct {
	AccessAll           bool                                    `xml:"accessAll"`
	ClientVpnEndpointID string                                  `xml:"clientVpnEndpointId,omitempty"`
	Description         string                                  `xml:"description,omitempty"`
	DestinationCIDR     string                                  `xml:"destinationCidr,omitempty"`
	GroupID             string                                  `xml:"groupId,omitempty"`
	Status              ec2ClientVpnAuthorizationRuleStatusItem `xml:"status,omitempty"`
}

type ec2ClientVpnConnectionSet struct {
	Items []ec2ClientVpnConnectionItem `xml:"item"`
}

type ec2ClientVpnConnectionItem struct {
	ClientIP                  string                           `xml:"clientIp,omitempty"`
	ClientVpnEndpointID       string                           `xml:"clientVpnEndpointId,omitempty"`
	CommonName                string                           `xml:"commonName,omitempty"`
	ConnectionEndTime         string                           `xml:"connectionEndTime,omitempty"`
	ConnectionEstablishedTime string                           `xml:"connectionEstablishedTime,omitempty"`
	ConnectionID              string                           `xml:"connectionId,omitempty"`
	EgressBytes               string                           `xml:"egressBytes,omitempty"`
	EgressPackets             string                           `xml:"egressPackets,omitempty"`
	IngressBytes              string                           `xml:"ingressBytes,omitempty"`
	IngressPackets            string                           `xml:"ingressPackets,omitempty"`
	PostureComplianceStatuses ec2ValueStringSet                `xml:"postureComplianceStatuses"`
	Status                    ec2ClientVpnConnectionStatusItem `xml:"status,omitempty"`
	Timestamp                 string                           `xml:"timestamp,omitempty"`
	Username                  string                           `xml:"username,omitempty"`
}

type ec2TerminateClientVpnConnectionStatusSet struct {
	Items []ec2TerminateClientVpnConnectionStatusItem `xml:"item"`
}

type ec2TerminateClientVpnConnectionStatusItem struct {
	ConnectionID   string                           `xml:"connectionId,omitempty"`
	CurrentStatus  ec2ClientVpnConnectionStatusItem `xml:"currentStatus,omitempty"`
	PreviousStatus ec2ClientVpnConnectionStatusItem `xml:"previousStatus,omitempty"`
}

type ec2ClientVpnEndpointStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2ClientVpnRouteStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2ClientVpnAssociationStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2ClientVpnAuthorizationRuleStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2ClientVpnConnectionStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2ClientCertificateRevocationListStatusItem struct {
	Code    string `xml:"code,omitempty"`
	Message string `xml:"message,omitempty"`
}

type ec2ValueStringSet struct {
	Items []string `xml:"item"`
}
