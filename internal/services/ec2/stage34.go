package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type TransitGatewayMulticastDomainAssociation struct {
	ResourceID                 string
	ResourceOwnerID            string
	ResourceType               string
	Subnet                     TransitGatewaySubnetAssociation
	TransitGatewayAttachmentID string
}

func (s *Service) GetTransitGatewayMulticastDomainAssociations(
	transitGatewayMulticastDomainID string,
	resourceIDs, resourceTypes, states, subnetIDs, transitGatewayAttachmentIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayMulticastDomainAssociation, *string, error) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	if transitGatewayMulticastDomainID == "" {
		return nil, nil, ErrInvalidParameter
	}

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

	resourceIDSet := toStringSet(resourceIDs)
	resourceTypeSet := toLowerStringSet(resourceTypes)
	stateSet := toLowerStringSet(states)
	subnetIDSet := toStringSet(subnetIDs)
	transitGatewayAttachmentIDSet := toStringSet(transitGatewayAttachmentIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayMulticastDomainAssociation, 0)
	for _, associations := range s.transitGatewayMulticastDomainAssocs {
		if associations.TransitGatewayMulticastDomainID != transitGatewayMulticastDomainID {
			continue
		}
		for _, subnet := range associations.Subnets {
			item := TransitGatewayMulticastDomainAssociation{
				ResourceID:                 associations.ResourceID,
				ResourceOwnerID:            associations.ResourceOwnerID,
				ResourceType:               associations.ResourceType,
				Subnet:                     subnet,
				TransitGatewayAttachmentID: associations.TransitGatewayAttachmentID,
			}
			if len(resourceIDSet) > 0 {
				if _, ok := resourceIDSet[item.ResourceID]; !ok {
					continue
				}
			}
			if len(resourceTypeSet) > 0 {
				if _, ok := resourceTypeSet[strings.ToLower(item.ResourceType)]; !ok {
					continue
				}
			}
			if len(stateSet) > 0 {
				if _, ok := stateSet[strings.ToLower(item.Subnet.State)]; !ok {
					continue
				}
			}
			if len(subnetIDSet) > 0 {
				if _, ok := subnetIDSet[item.Subnet.SubnetID]; !ok {
					continue
				}
			}
			if len(transitGatewayAttachmentIDSet) > 0 {
				if _, ok := transitGatewayAttachmentIDSet[item.TransitGatewayAttachmentID]; !ok {
					continue
				}
			}
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TransitGatewayAttachmentID != out[j].TransitGatewayAttachmentID {
			return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID
		}
		return out[i].Subnet.SubnetID < out[j].Subnet.SubnetID
	})

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

	page := append([]TransitGatewayMulticastDomainAssociation(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) GetTransitGatewayPolicyTableAssociations(
	transitGatewayPolicyTableID string,
	resourceIDs, resourceTypes, states, transitGatewayAttachmentIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayPolicyTableAssociation, *string, error) {
	transitGatewayPolicyTableID = strings.TrimSpace(transitGatewayPolicyTableID)
	if transitGatewayPolicyTableID == "" {
		return nil, nil, ErrInvalidParameter
	}

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

	resourceIDSet := toStringSet(resourceIDs)
	resourceTypeSet := toLowerStringSet(resourceTypes)
	stateSet := toLowerStringSet(states)
	transitGatewayAttachmentIDSet := toStringSet(transitGatewayAttachmentIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayPolicyTableAssociation, 0)
	for _, association := range s.transitGatewayPolicyTableAssocs {
		if association.TransitGatewayPolicyTableID != transitGatewayPolicyTableID {
			continue
		}
		if len(resourceIDSet) > 0 {
			if _, ok := resourceIDSet[association.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceTypeSet) > 0 {
			if _, ok := resourceTypeSet[strings.ToLower(association.ResourceType)]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(association.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayAttachmentIDSet) > 0 {
			if _, ok := transitGatewayAttachmentIDSet[association.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		out = append(out, *association)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })

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

	page := append([]TransitGatewayPolicyTableAssociation(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) GetTransitGatewayRouteTableAssociations(
	transitGatewayRouteTableID string,
	resourceIDs, resourceTypes, states, transitGatewayAttachmentIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayRouteTableAssociation, *string, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	if transitGatewayRouteTableID == "" {
		return nil, nil, ErrInvalidParameter
	}

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

	resourceIDSet := toStringSet(resourceIDs)
	resourceTypeSet := toLowerStringSet(resourceTypes)
	stateSet := toLowerStringSet(states)
	transitGatewayAttachmentIDSet := toStringSet(transitGatewayAttachmentIDs)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayRouteTableAssociation, 0)
	for _, association := range s.transitGatewayRouteTableAssocs {
		if association.TransitGatewayRouteTableID != transitGatewayRouteTableID {
			continue
		}
		if len(resourceIDSet) > 0 {
			if _, ok := resourceIDSet[association.ResourceID]; !ok {
				continue
			}
		}
		if len(resourceTypeSet) > 0 {
			if _, ok := resourceTypeSet[strings.ToLower(association.ResourceType)]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(association.State)]; !ok {
				continue
			}
		}
		if len(transitGatewayAttachmentIDSet) > 0 {
			if _, ok := transitGatewayAttachmentIDSet[association.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		out = append(out, *association)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })

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

	page := append([]TransitGatewayRouteTableAssociation(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}
