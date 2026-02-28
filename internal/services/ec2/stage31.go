package ec2

import "strings"

func (s *Service) DisassociateTransitGatewayMulticastDomain(
	transitGatewayMulticastDomainID,
	transitGatewayAttachmentID string,
	subnetIDs []string,
) (TransitGatewayMulticastDomainAssociations, error) {
	transitGatewayMulticastDomainID = strings.TrimSpace(transitGatewayMulticastDomainID)
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayMulticastDomainID == "" || transitGatewayAttachmentID == "" || len(subnetIDs) == 0 {
		return TransitGatewayMulticastDomainAssociations{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resolvedSubnetIDs := dedupeTrimmedStrings(subnetIDs)
	if len(resolvedSubnetIDs) == 0 {
		return TransitGatewayMulticastDomainAssociations{}, ErrInvalidParameter
	}

	key := transitGatewayMulticastDomainAssocKey(transitGatewayMulticastDomainID, transitGatewayAttachmentID)
	existing := s.transitGatewayMulticastDomainAssocs[key]
	if existing == nil {
		return TransitGatewayMulticastDomainAssociations{}, ErrNotFound
	}

	existingBySubnet := map[string]TransitGatewaySubnetAssociation{}
	for _, subnet := range existing.Subnets {
		existingBySubnet[subnet.SubnetID] = subnet
	}
	for _, subnetID := range resolvedSubnetIDs {
		if _, ok := existingBySubnet[subnetID]; !ok {
			return TransitGatewayMulticastDomainAssociations{}, ErrNotFound
		}
	}

	disassociatedSubnets := make([]TransitGatewaySubnetAssociation, 0, len(resolvedSubnetIDs))
	removeSet := toStringSet(resolvedSubnetIDs)
	remainingSubnets := make([]TransitGatewaySubnetAssociation, 0, len(existing.Subnets))
	for _, subnet := range existing.Subnets {
		if _, remove := removeSet[subnet.SubnetID]; remove {
			disassociatedSubnets = append(disassociatedSubnets, TransitGatewaySubnetAssociation{
				State:    "disassociated",
				SubnetID: subnet.SubnetID,
			})
			continue
		}
		remainingSubnets = append(remainingSubnets, subnet)
	}

	if len(remainingSubnets) == 0 {
		delete(s.transitGatewayMulticastDomainAssocs, key)
	} else {
		existing.Subnets = remainingSubnets
	}

	return TransitGatewayMulticastDomainAssociations{
		ResourceID:                      existing.ResourceID,
		ResourceOwnerID:                 existing.ResourceOwnerID,
		ResourceType:                    existing.ResourceType,
		Subnets:                         disassociatedSubnets,
		TransitGatewayAttachmentID:      existing.TransitGatewayAttachmentID,
		TransitGatewayMulticastDomainID: existing.TransitGatewayMulticastDomainID,
	}, nil
}

func (s *Service) DisassociateTransitGatewayPolicyTable(
	transitGatewayPolicyTableID,
	transitGatewayAttachmentID string,
) (TransitGatewayPolicyTableAssociation, error) {
	transitGatewayPolicyTableID = strings.TrimSpace(transitGatewayPolicyTableID)
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayPolicyTableID == "" || transitGatewayAttachmentID == "" {
		return TransitGatewayPolicyTableAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := transitGatewayPolicyTableAssocKey(transitGatewayPolicyTableID, transitGatewayAttachmentID)
	existing := s.transitGatewayPolicyTableAssocs[key]
	if existing == nil {
		return TransitGatewayPolicyTableAssociation{}, ErrNotFound
	}

	out := *existing
	out.State = "disassociated"
	delete(s.transitGatewayPolicyTableAssocs, key)
	return out, nil
}

func (s *Service) DisassociateTransitGatewayRouteTable(
	transitGatewayRouteTableID,
	transitGatewayAttachmentID string,
) (TransitGatewayRouteTableAssociation, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayRouteTableID == "" || transitGatewayAttachmentID == "" {
		return TransitGatewayRouteTableAssociation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := transitGatewayRouteTableAssocKey(transitGatewayRouteTableID, transitGatewayAttachmentID)
	existing := s.transitGatewayRouteTableAssocs[key]
	if existing == nil {
		return TransitGatewayRouteTableAssociation{}, ErrNotFound
	}

	out := *existing
	out.State = "disassociated"
	delete(s.transitGatewayRouteTableAssocs, key)
	return out, nil
}
