package ec2

import (
	"fmt"
	"strings"
)

func (s *Service) CreateVpcEndpointServiceConfiguration(
	acceptanceRequired *bool,
	gatewayLoadBalancerARNs []string,
	networkLoadBalancerARNs []string,
	privateDNSName *string,
	supportedIPAddressTypes []string,
	supportedRegions []string,
	clientToken *string,
) (VpcEndpointServiceConfiguration, *string, error) {
	gatewayLoadBalancerARNs = dedupeTrimmedStrings(gatewayLoadBalancerARNs)
	networkLoadBalancerARNs = dedupeTrimmedStrings(networkLoadBalancerARNs)
	if len(gatewayLoadBalancerARNs) == 0 && len(networkLoadBalancerARNs) == 0 {
		return VpcEndpointServiceConfiguration{}, nil, ErrInvalidParameter
	}

	if privateDNSName != nil {
		name := strings.TrimSpace(*privateDNSName)
		if name == "" {
			return VpcEndpointServiceConfiguration{}, nil, ErrInvalidParameter
		}
		privateDNSName = &name
	}

	if len(supportedIPAddressTypes) == 0 {
		supportedIPAddressTypes = []string{"ipv4"}
	} else {
		supportedIPAddressTypes = dedupeTrimmedStrings(supportedIPAddressTypes)
	}
	if len(supportedRegions) == 0 {
		supportedRegions = []string{DefaultRegion}
	} else {
		supportedRegions = dedupeTrimmedStrings(supportedRegions)
	}

	var effectiveAcceptanceRequired bool
	if acceptanceRequired != nil {
		effectiveAcceptanceRequired = *acceptanceRequired
	}

	var effectiveClientToken *string
	if clientToken != nil {
		token := strings.TrimSpace(*clientToken)
		if token != "" {
			effectiveClientToken = &token
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	serviceID := s.nextIDLocked("vpce-svc")
	cfg := VpcEndpointServiceConfiguration{
		ServiceID:               serviceID,
		ServiceName:             fmt.Sprintf("com.amazonaws.vpce.%s.%s", DefaultRegion, serviceID),
		ServiceState:            "Available",
		PayerResponsibility:     "EndpointOwner",
		AcceptanceRequired:      effectiveAcceptanceRequired,
		GatewayLoadBalancerARNs: gatewayLoadBalancerARNs,
		NetworkLoadBalancerARNs: networkLoadBalancerARNs,
		SupportedIPAddressTypes: supportedIPAddressTypes,
		SupportedRegions:        supportedRegions,
	}
	if privateDNSName != nil {
		cfg.PrivateDNSName = *privateDNSName
	}

	s.vpcEndpointServiceConfigurations[serviceID] = &cfg
	s.vpcEndpointServicePayerResponsibility[serviceID] = cfg.PayerResponsibility
	s.vpcEndpointServicePermissions[serviceID] = map[string]string{}

	return cfg, effectiveClientToken, nil
}
