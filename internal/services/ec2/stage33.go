package ec2

import (
	"sort"
	"strconv"
	"strings"
)

func (s *Service) DescribeTransitGatewayMulticastDomains(
	transitGatewayMulticastDomainIDs, stateFilters, transitGatewayIDs, filterDomainIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayMulticastDomain, *string, error) {
	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayMulticastDomainIDs), dedupeTrimmedStrings(filterDomainIDs)...))
	stateSet := toLowerStringSet(stateFilters)
	transitGatewayIDSet := toStringSet(transitGatewayIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayMulticastDomain, 0, len(s.transitGatewayMulticastDomains))
	for _, domain := range s.transitGatewayMulticastDomains {
		if len(idSet) > 0 {
			if _, ok := idSet[domain.ID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(domain.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[domain.TransitID]; !ok {
				continue
			}
		}
		out = append(out, cloneTransitGatewayMulticastDomain(domain))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	if start > len(out) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(out) {
			end = len(out)
		}
	}

	page := append([]TransitGatewayMulticastDomain(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) DescribeTransitGatewayPolicyTables(
	transitGatewayPolicyTableIDs, stateFilters, transitGatewayIDs, filterPolicyTableIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayPolicyTable, *string, error) {
	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayPolicyTableIDs), dedupeTrimmedStrings(filterPolicyTableIDs)...))
	stateSet := toLowerStringSet(stateFilters)
	transitGatewayIDSet := toStringSet(transitGatewayIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayPolicyTable, 0, len(s.transitGatewayPolicyTables))
	for _, table := range s.transitGatewayPolicyTables {
		if len(idSet) > 0 {
			if _, ok := idSet[table.ID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(table.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[table.TransitID]; !ok {
				continue
			}
		}
		out = append(out, cloneTransitGatewayPolicyTable(table))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	if start > len(out) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(out) {
			end = len(out)
		}
	}

	page := append([]TransitGatewayPolicyTable(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) DescribeTransitGatewayRouteTables(
	transitGatewayRouteTableIDs []string,
	defaultAssociationRouteTableFilters, defaultPropagationRouteTableFilters []bool,
	stateFilters, transitGatewayIDs, filterRouteTableIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayRouteTable, *string, error) {
	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, nil, ErrInvalidParameter
			}
			start = parsed
		}
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	idSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayRouteTableIDs), dedupeTrimmedStrings(filterRouteTableIDs)...))
	stateSet := toLowerStringSet(stateFilters)
	transitGatewayIDSet := toStringSet(transitGatewayIDs)
	defaultAssociationSet := toBoolSet(defaultAssociationRouteTableFilters)
	defaultPropagationSet := toBoolSet(defaultPropagationRouteTableFilters)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayRouteTable, 0, len(s.transitGatewayRouteTables))
	for _, table := range s.transitGatewayRouteTables {
		if len(idSet) > 0 {
			if _, ok := idSet[table.ID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(table.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[table.TransitID]; !ok {
				continue
			}
		}
		if len(defaultAssociationSet) > 0 {
			if _, ok := defaultAssociationSet[table.DefaultAssociationRouteTable]; !ok {
				continue
			}
		}
		if len(defaultPropagationSet) > 0 {
			if _, ok := defaultPropagationSet[table.DefaultPropagationRouteTable]; !ok {
				continue
			}
		}
		out = append(out, cloneTransitGatewayRouteTable(table))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	if start > len(out) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(out) {
			end = len(out)
		}
	}

	page := append([]TransitGatewayRouteTable(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func toBoolSet(values []bool) map[bool]struct{} {
	out := map[bool]struct{}{}
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
