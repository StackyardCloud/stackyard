package ec2

import (
	"sort"
	"strings"
)

func (s *Service) DescribeVpcEndpointServiceConfigurations(
	serviceIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpointServiceConfiguration, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	serviceIDSet := toStringSet(dedupeTrimmedStrings(serviceIDs))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	filterServiceIDSet := toStringSet(standardFilters["service-id"])
	filterServiceNameSet := toStringSet(standardFilters["service-name"])
	filterServiceStateSet := toLowerStringSet(standardFilters["service-state"])
	filterSupportedIPTypesSet := toLowerStringSet(standardFilters["supported-ip-address-types"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VpcEndpointServiceConfiguration, 0, len(s.vpcEndpointServiceConfigurations))
	for _, cfg := range s.vpcEndpointServiceConfigurations {
		if cfg == nil {
			continue
		}
		if len(serviceIDSet) > 0 {
			if _, ok := serviceIDSet[cfg.ServiceID]; !ok {
				continue
			}
		}
		if len(filterServiceIDSet) > 0 {
			if _, ok := filterServiceIDSet[cfg.ServiceID]; !ok {
				continue
			}
		}
		if len(filterServiceNameSet) > 0 {
			if _, ok := filterServiceNameSet[cfg.ServiceName]; !ok {
				continue
			}
		}
		if len(filterServiceStateSet) > 0 {
			if _, ok := filterServiceStateSet[strings.ToLower(cfg.ServiceState)]; !ok {
				continue
			}
		}
		if len(filterSupportedIPTypesSet) > 0 {
			supportedIPTypes := make([]string, 0, len(cfg.SupportedIPAddressTypes))
			for _, value := range cfg.SupportedIPAddressTypes {
				value = strings.ToLower(strings.TrimSpace(value))
				if value == "" {
					continue
				}
				supportedIPTypes = append(supportedIPTypes, value)
			}
			if !containsAnyString(supportedIPTypes, filterSupportedIPTypesSet) {
				continue
			}
		}
		if !matchesTagFilters(nil, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, cloneVpcEndpointServiceConfiguration(cfg))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ServiceID < items[j].ServiceID
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VpcEndpointServiceConfiguration(nil), items[start:end]...), outputToken, nil
}

func cloneVpcEndpointServiceConfiguration(in *VpcEndpointServiceConfiguration) VpcEndpointServiceConfiguration {
	out := *in
	out.GatewayLoadBalancerARNs = append([]string(nil), in.GatewayLoadBalancerARNs...)
	out.NetworkLoadBalancerARNs = append([]string(nil), in.NetworkLoadBalancerARNs...)
	out.SupportedIPAddressTypes = append([]string(nil), in.SupportedIPAddressTypes...)
	out.SupportedRegions = append([]string(nil), in.SupportedRegions...)
	return out
}
