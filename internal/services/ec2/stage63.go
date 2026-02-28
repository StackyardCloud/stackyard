package ec2

import (
	"sort"
	"strings"
)

type VpcEndpointServiceDetail struct {
	AcceptanceRequired         bool
	Owner                      string
	PayerResponsibility        string
	PrivateDNSName             string
	ServiceID                  string
	ServiceName                string
	ServiceRegion              string
	ServiceTypes               []string
	SupportedIPAddressTypes    []string
	VpcEndpointPolicySupported bool
}

func (s *Service) DescribeVpcEndpointServices(
	serviceNames []string,
	serviceRegions []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpointServiceDetail, []string, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, nil, ErrInvalidParameter
	}

	serviceNameSet := toStringSet(dedupeTrimmedStrings(serviceNames))
	serviceRegionSet := toLowerStringSet(dedupeTrimmedStrings(serviceRegions))
	standardFilters, tagKeyFilters, tagFilters := splitEC2Filters(filters)
	ownerFilterSet := toStringSet(standardFilters["owner"])
	serviceNameFilterSet := toStringSet(standardFilters["service-name"])
	serviceRegionFilterSet := toLowerStringSet(standardFilters["service-region"])
	serviceTypeFilterSet := toLowerStringSet(standardFilters["service-type"])
	supportedIPTypesFilterSet := toLowerStringSet(standardFilters["supported-ip-address-types"])

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]VpcEndpointServiceDetail, 0, len(s.vpcEndpointServiceConfigurations))
	for _, cfg := range s.vpcEndpointServiceConfigurations {
		if cfg == nil {
			continue
		}
		detail := vpcEndpointServiceDetailFromConfig(cfg)
		if len(serviceNameSet) > 0 {
			if _, ok := serviceNameSet[detail.ServiceName]; !ok {
				continue
			}
		}
		if len(serviceRegionSet) > 0 {
			if _, ok := serviceRegionSet[strings.ToLower(detail.ServiceRegion)]; !ok {
				continue
			}
		}
		if len(ownerFilterSet) > 0 {
			if _, ok := ownerFilterSet[detail.Owner]; !ok {
				continue
			}
		}
		if len(serviceNameFilterSet) > 0 {
			if _, ok := serviceNameFilterSet[detail.ServiceName]; !ok {
				continue
			}
		}
		if len(serviceRegionFilterSet) > 0 {
			if _, ok := serviceRegionFilterSet[strings.ToLower(detail.ServiceRegion)]; !ok {
				continue
			}
		}
		if len(serviceTypeFilterSet) > 0 {
			serviceTypes := make([]string, 0, len(detail.ServiceTypes))
			for _, value := range detail.ServiceTypes {
				value = strings.ToLower(strings.TrimSpace(value))
				if value == "" {
					continue
				}
				serviceTypes = append(serviceTypes, value)
			}
			if !containsAnyString(serviceTypes, serviceTypeFilterSet) {
				continue
			}
		}
		if len(supportedIPTypesFilterSet) > 0 {
			supportedIPTypes := make([]string, 0, len(detail.SupportedIPAddressTypes))
			for _, value := range detail.SupportedIPAddressTypes {
				value = strings.ToLower(strings.TrimSpace(value))
				if value == "" {
					continue
				}
				supportedIPTypes = append(supportedIPTypes, value)
			}
			if !containsAnyString(supportedIPTypes, supportedIPTypesFilterSet) {
				continue
			}
		}
		if !matchesTagFilters(nil, tagKeyFilters, tagFilters) {
			continue
		}
		items = append(items, detail)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].ServiceName == items[j].ServiceName {
			return items[i].ServiceID < items[j].ServiceID
		}
		return items[i].ServiceName < items[j].ServiceName
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, nil, err
	}
	page := append([]VpcEndpointServiceDetail(nil), items[start:end]...)
	outNames := make([]string, 0, len(page))
	for _, detail := range page {
		outNames = append(outNames, detail.ServiceName)
	}
	return page, outNames, outputToken, nil
}

func vpcEndpointServiceDetailFromConfig(cfg *VpcEndpointServiceConfiguration) VpcEndpointServiceDetail {
	serviceRegion := DefaultRegion
	for _, region := range cfg.SupportedRegions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		serviceRegion = region
		break
	}
	return VpcEndpointServiceDetail{
		AcceptanceRequired:         cfg.AcceptanceRequired,
		Owner:                      DefaultAccountID,
		PayerResponsibility:        cfg.PayerResponsibility,
		PrivateDNSName:             cfg.PrivateDNSName,
		ServiceID:                  cfg.ServiceID,
		ServiceName:                cfg.ServiceName,
		ServiceRegion:              serviceRegion,
		ServiceTypes:               vpcEndpointServiceTypesFromConfig(cfg),
		SupportedIPAddressTypes:    append([]string(nil), cfg.SupportedIPAddressTypes...),
		VpcEndpointPolicySupported: true,
	}
}

func vpcEndpointServiceTypesFromConfig(cfg *VpcEndpointServiceConfiguration) []string {
	out := make([]string, 0, 2)
	if len(cfg.NetworkLoadBalancerARNs) > 0 {
		out = append(out, "Interface")
	}
	if len(cfg.GatewayLoadBalancerARNs) > 0 {
		out = append(out, "GatewayLoadBalancer")
	}
	if len(out) == 0 {
		out = append(out, "Interface")
	}
	return out
}
