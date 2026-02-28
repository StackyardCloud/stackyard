package server

import (
	"encoding/xml"
	"net/http"

	ec2svc "github.com/stackyard/stackyard/internal/services/ec2"
)

func (s *Server) handleEC2Stage55Action(w http.ResponseWriter, r *http.Request, action string) bool {
	switch action {
	case "CreateVpcEndpointServiceConfiguration":
		acceptanceRequired, hasAcceptanceRequired, ok := ec2OptionalBoolFromForm(r.Form, "AcceptanceRequired")
		if !ok {
			respondEC2ErrorForErr(w, ec2svc.ErrInvalidParameter)
			return true
		}
		if !hasAcceptanceRequired {
			acceptanceRequired = nil
		}

		cfg, clientToken, err := s.ec2.CreateVpcEndpointServiceConfiguration(
			acceptanceRequired,
			parseEC2MembersWithAliases(r.Form, "GatewayLoadBalancerArn", "GatewayLoadBalancerArns"),
			parseEC2MembersWithAliases(r.Form, "NetworkLoadBalancerArn", "NetworkLoadBalancerArns"),
			parseEC2OptionalString(r.Form.Get("PrivateDnsName")),
			parseEC2MembersWithAliases(r.Form, "SupportedIpAddressType", "SupportedIpAddressTypes"),
			parseEC2MembersWithAliases(r.Form, "SupportedRegion", "SupportedRegions"),
			parseEC2OptionalString(r.Form.Get("ClientToken")),
		)
		if err != nil {
			respondEC2ErrorForErr(w, err)
			return true
		}

		respondEC2XML(w, ec2CreateVpcEndpointServiceConfigurationResponse{
			XMLName:              xml.Name{Local: "CreateVpcEndpointServiceConfigurationResponse"},
			Xmlns:                ec2Namespace,
			RequestID:            "stackyard-request",
			ClientToken:          clientToken,
			ServiceConfiguration: ec2ServiceConfigurationItemFrom(cfg),
		})
		return true
	default:
		return false
	}
}

func ec2ServiceConfigurationItemFrom(cfg ec2svc.VpcEndpointServiceConfiguration) ec2ServiceConfigurationItem {
	return ec2ServiceConfigurationItem{
		AcceptanceRequired:      &cfg.AcceptanceRequired,
		GatewayLoadBalancerARNs: ec2Stage55StringSet{Items: append([]string(nil), cfg.GatewayLoadBalancerARNs...)},
		NetworkLoadBalancerARNs: ec2Stage55StringSet{Items: append([]string(nil), cfg.NetworkLoadBalancerARNs...)},
		PayerResponsibility:     cfg.PayerResponsibility,
		PrivateDNSName:          cfg.PrivateDNSName,
		ServiceID:               cfg.ServiceID,
		ServiceName:             cfg.ServiceName,
		ServiceState:            cfg.ServiceState,
		ServiceType:             ec2ServiceTypeDetailSet{Items: ec2ServiceTypeItemsFromConfig(cfg)},
		SupportedIPAddressTypes: ec2Stage55StringSet{Items: append([]string(nil), cfg.SupportedIPAddressTypes...)},
		SupportedRegions:        ec2SupportedRegionDetailSet{Items: ec2SupportedRegionItemsFromConfig(cfg)},
	}
}

func ec2ServiceTypeItemsFromConfig(cfg ec2svc.VpcEndpointServiceConfiguration) []ec2ServiceTypeDetailItem {
	out := make([]ec2ServiceTypeDetailItem, 0, 2)
	if len(cfg.NetworkLoadBalancerARNs) > 0 {
		out = append(out, ec2ServiceTypeDetailItem{ServiceType: "Interface"})
	}
	if len(cfg.GatewayLoadBalancerARNs) > 0 {
		out = append(out, ec2ServiceTypeDetailItem{ServiceType: "GatewayLoadBalancer"})
	}
	if len(out) == 0 {
		out = append(out, ec2ServiceTypeDetailItem{ServiceType: "Interface"})
	}
	return out
}

func ec2SupportedRegionItemsFromConfig(cfg ec2svc.VpcEndpointServiceConfiguration) []ec2SupportedRegionDetailItem {
	out := make([]ec2SupportedRegionDetailItem, 0, len(cfg.SupportedRegions))
	for _, region := range cfg.SupportedRegions {
		out = append(out, ec2SupportedRegionDetailItem{
			Region:       region,
			ServiceState: cfg.ServiceState,
		})
	}
	return out
}

type ec2CreateVpcEndpointServiceConfigurationResponse struct {
	XMLName              xml.Name                    `xml:"CreateVpcEndpointServiceConfigurationResponse"`
	Xmlns                string                      `xml:"xmlns,attr"`
	RequestID            string                      `xml:"requestId"`
	ClientToken          *string                     `xml:"clientToken,omitempty"`
	ServiceConfiguration ec2ServiceConfigurationItem `xml:"serviceConfiguration"`
}

type ec2ServiceConfigurationItem struct {
	AcceptanceRequired      *bool                       `xml:"acceptanceRequired,omitempty"`
	GatewayLoadBalancerARNs ec2Stage55StringSet         `xml:"gatewayLoadBalancerArnSet"`
	NetworkLoadBalancerARNs ec2Stage55StringSet         `xml:"networkLoadBalancerArnSet"`
	PayerResponsibility     string                      `xml:"payerResponsibility,omitempty"`
	PrivateDNSName          string                      `xml:"privateDnsName,omitempty"`
	ServiceID               string                      `xml:"serviceId,omitempty"`
	ServiceName             string                      `xml:"serviceName,omitempty"`
	ServiceState            string                      `xml:"serviceState,omitempty"`
	ServiceType             ec2ServiceTypeDetailSet     `xml:"serviceType"`
	SupportedIPAddressTypes ec2Stage55StringSet         `xml:"supportedIpAddressTypeSet"`
	SupportedRegions        ec2SupportedRegionDetailSet `xml:"supportedRegionSet"`
}

type ec2Stage55StringSet struct {
	Items []string `xml:"item"`
}

type ec2ServiceTypeDetailSet struct {
	Items []ec2ServiceTypeDetailItem `xml:"item"`
}

type ec2ServiceTypeDetailItem struct {
	ServiceType string `xml:"serviceType,omitempty"`
}

type ec2SupportedRegionDetailSet struct {
	Items []ec2SupportedRegionDetailItem `xml:"item"`
}

type ec2SupportedRegionDetailItem struct {
	Region       string `xml:"region,omitempty"`
	ServiceState string `xml:"serviceState,omitempty"`
}
