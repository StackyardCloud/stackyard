package ec2

import (
	"sort"
	"strings"
)

func (s *Service) DescribeVpcEndpointServicePermissions(
	serviceID string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]VpcEndpointServiceAddedPrincipal, *string, error) {
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" {
		return nil, nil, ErrInvalidParameter
	}

	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, err
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standardFilters, _, _ := splitEC2Filters(filters)
	principalFilterSet := toStringSet(standardFilters["principal"])
	principalTypeFilterSet := toLowerStringSet(standardFilters["principal-type"])

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.vpcEndpointServicePayerResponsibility[serviceID]; !ok {
		return nil, nil, ErrNotFound
	}

	principalsByService := s.vpcEndpointServicePermissions[serviceID]
	items := make([]VpcEndpointServiceAddedPrincipal, 0, len(principalsByService))
	for principal, servicePermissionID := range principalsByService {
		principalType := ec2PrincipalTypeFromPrincipal(principal)
		if len(principalFilterSet) > 0 {
			if _, ok := principalFilterSet[principal]; !ok {
				continue
			}
		}
		if len(principalTypeFilterSet) > 0 {
			if _, ok := principalTypeFilterSet[strings.ToLower(principalType)]; !ok {
				continue
			}
		}
		items = append(items, VpcEndpointServiceAddedPrincipal{
			Principal:           principal,
			PrincipalType:       principalType,
			ServiceID:           serviceID,
			ServicePermissionID: servicePermissionID,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Principal == items[j].Principal {
			return items[i].ServicePermissionID < items[j].ServicePermissionID
		}
		return items[i].Principal < items[j].Principal
	})

	start, end, outputToken, err := ec2PageWindow(len(items), start, maxResults)
	if err != nil {
		return nil, nil, err
	}
	return append([]VpcEndpointServiceAddedPrincipal(nil), items[start:end]...), outputToken, nil
}
