package ec2

import "strings"

type VpcEndpointServiceConfiguration struct {
	ServiceID               string
	ServiceName             string
	ServiceState            string
	PayerResponsibility     string
	AcceptanceRequired      bool
	GatewayLoadBalancerARNs []string
	NetworkLoadBalancerARNs []string
	SupportedIPAddressTypes []string
	SupportedRegions        []string
	PrivateDNSName          string
}

func (s *Service) ModifyVpcEndpointServiceConfiguration(
	serviceID string,
	acceptanceRequired *bool,
	addGatewayLoadBalancerARNs []string,
	addNetworkLoadBalancerARNs []string,
	addSupportedIPAddressTypes []string,
	addSupportedRegions []string,
	privateDNSName *string,
	removeGatewayLoadBalancerARNs []string,
	removeNetworkLoadBalancerARNs []string,
	removePrivateDNSName *bool,
	removeSupportedIPAddressTypes []string,
	removeSupportedRegions []string,
) (bool, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return false, ErrInvalidParameter
	}
	if privateDNSName != nil && strings.TrimSpace(*privateDNSName) == "" {
		return false, ErrInvalidParameter
	}
	if privateDNSName != nil && removePrivateDNSName != nil && *removePrivateDNSName {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcEndpointServicePayerResponsibility[serviceID]; !ok {
		return false, ErrNotFound
	}

	cfg := s.vpcEndpointServiceConfigurations[serviceID]
	if cfg == nil {
		cfg = &VpcEndpointServiceConfiguration{
			ServiceID:               serviceID,
			SupportedIPAddressTypes: []string{"ipv4"},
			SupportedRegions:        []string{DefaultRegion},
		}
		s.vpcEndpointServiceConfigurations[serviceID] = cfg
	}

	if acceptanceRequired != nil {
		cfg.AcceptanceRequired = *acceptanceRequired
	}
	cfg.GatewayLoadBalancerARNs = addToEC2StringSet(cfg.GatewayLoadBalancerARNs, addGatewayLoadBalancerARNs)
	cfg.GatewayLoadBalancerARNs = removeFromEC2StringSet(cfg.GatewayLoadBalancerARNs, removeGatewayLoadBalancerARNs)
	cfg.NetworkLoadBalancerARNs = addToEC2StringSet(cfg.NetworkLoadBalancerARNs, addNetworkLoadBalancerARNs)
	cfg.NetworkLoadBalancerARNs = removeFromEC2StringSet(cfg.NetworkLoadBalancerARNs, removeNetworkLoadBalancerARNs)
	cfg.SupportedIPAddressTypes = addToEC2StringSet(cfg.SupportedIPAddressTypes, addSupportedIPAddressTypes)
	cfg.SupportedIPAddressTypes = removeFromEC2StringSet(cfg.SupportedIPAddressTypes, removeSupportedIPAddressTypes)
	cfg.SupportedRegions = addToEC2StringSet(cfg.SupportedRegions, addSupportedRegions)
	cfg.SupportedRegions = removeFromEC2StringSet(cfg.SupportedRegions, removeSupportedRegions)

	if privateDNSName != nil {
		cfg.PrivateDNSName = strings.TrimSpace(*privateDNSName)
	}
	if removePrivateDNSName != nil && *removePrivateDNSName {
		cfg.PrivateDNSName = ""
	}

	return true, nil
}

func addToEC2StringSet(existing []string, additions []string) []string {
	result := append([]string(nil), existing...)
	existingSet := map[string]struct{}{}
	for _, value := range existing {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		existingSet[value] = struct{}{}
	}
	for _, value := range dedupeTrimmedStrings(additions) {
		if _, ok := existingSet[value]; ok {
			continue
		}
		result = append(result, value)
		existingSet[value] = struct{}{}
	}
	return result
}

func removeFromEC2StringSet(existing []string, removals []string) []string {
	removeSet := map[string]struct{}{}
	for _, value := range dedupeTrimmedStrings(removals) {
		removeSet[value] = struct{}{}
	}
	if len(removeSet) == 0 {
		return append([]string(nil), existing...)
	}
	result := make([]string, 0, len(existing))
	for _, value := range existing {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, remove := removeSet[trimmed]; remove {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
