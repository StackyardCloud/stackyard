package ec2

import (
	"sort"
	"strings"
)

type TransitGatewaySubnetAssociation struct {
	State    string
	SubnetID string
}

type TransitGatewayMulticastDomainAssociations struct {
	ResourceID                      string
	ResourceOwnerID                 string
	ResourceType                    string
	Subnets                         []TransitGatewaySubnetAssociation
	TransitGatewayAttachmentID      string
	TransitGatewayMulticastDomainID string
}

type TransitGatewayPolicyTableAssociation struct {
	ResourceID                  string
	ResourceType                string
	State                       string
	TransitGatewayAttachmentID  string
	TransitGatewayPolicyTableID string
}

type TransitGatewayRouteTableAssociation struct {
	ResourceID                 string
	ResourceType               string
	State                      string
	TransitGatewayAttachmentID string
	TransitGatewayRouteTableID string
}

type TrunkInterfaceAssociation struct {
	AssociationID     string
	BranchInterfaceID string
	ClientToken       string
	GreKey            *int32
	InterfaceProtocol string
	Tags              map[string]string
	TrunkInterfaceID  string
	VlanID            *int32
}

type AssociateTrunkInterfaceResult struct {
	ClientToken          string
	InterfaceAssociation TrunkInterfaceAssociation
}

type DisassociateTrunkInterfaceResult struct {
	ClientToken string
	Return      bool
}

func (s *Service) AssociateTransitGatewayMulticastDomain(
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
	for _, subnetID := range resolvedSubnetIDs {
		if s.subnets[subnetID] == nil {
			return TransitGatewayMulticastDomainAssociations{}, ErrNotFound
		}
	}
	sort.Strings(resolvedSubnetIDs)

	key := transitGatewayMulticastDomainAssocKey(transitGatewayMulticastDomainID, transitGatewayAttachmentID)
	association := s.transitGatewayMulticastDomainAssocs[key]
	if association == nil {
		association = &TransitGatewayMulticastDomainAssociations{
			ResourceID:                      transitGatewayAttachmentID,
			ResourceOwnerID:                 DefaultAccountID,
			ResourceType:                    "vpc",
			TransitGatewayAttachmentID:      transitGatewayAttachmentID,
			TransitGatewayMulticastDomainID: transitGatewayMulticastDomainID,
			Subnets:                         nil,
		}
		s.transitGatewayMulticastDomainAssocs[key] = association
	}

	association.Subnets = make([]TransitGatewaySubnetAssociation, 0, len(resolvedSubnetIDs))
	for _, subnetID := range resolvedSubnetIDs {
		association.Subnets = append(association.Subnets, TransitGatewaySubnetAssociation{
			State:    "associated",
			SubnetID: subnetID,
		})
	}

	return cloneTransitGatewayMulticastDomainAssociation(association), nil
}

func (s *Service) AssociateTransitGatewayPolicyTable(
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
	association := s.transitGatewayPolicyTableAssocs[key]
	if association == nil {
		association = &TransitGatewayPolicyTableAssociation{}
		s.transitGatewayPolicyTableAssocs[key] = association
	}
	association.ResourceID = transitGatewayAttachmentID
	association.ResourceType = "vpc"
	association.State = "associated"
	association.TransitGatewayAttachmentID = transitGatewayAttachmentID
	association.TransitGatewayPolicyTableID = transitGatewayPolicyTableID

	return *association, nil
}

func (s *Service) AssociateTransitGatewayRouteTable(
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
	association := s.transitGatewayRouteTableAssocs[key]
	if association == nil {
		association = &TransitGatewayRouteTableAssociation{}
		s.transitGatewayRouteTableAssocs[key] = association
	}
	association.ResourceID = transitGatewayAttachmentID
	association.ResourceType = "vpc"
	association.State = "associated"
	association.TransitGatewayAttachmentID = transitGatewayAttachmentID
	association.TransitGatewayRouteTableID = transitGatewayRouteTableID

	return *association, nil
}

func (s *Service) AssociateTrunkInterface(
	branchInterfaceID,
	trunkInterfaceID string,
	vlanID,
	greKey *int32,
	clientToken string,
) (AssociateTrunkInterfaceResult, error) {
	branchInterfaceID = strings.TrimSpace(branchInterfaceID)
	trunkInterfaceID = strings.TrimSpace(trunkInterfaceID)
	clientToken = strings.TrimSpace(clientToken)

	if branchInterfaceID == "" || trunkInterfaceID == "" || branchInterfaceID == trunkInterfaceID {
		return AssociateTrunkInterfaceResult{}, ErrInvalidParameter
	}
	if (vlanID == nil && greKey == nil) || (vlanID != nil && greKey != nil) {
		return AssociateTrunkInterfaceResult{}, ErrInvalidParameter
	}
	if vlanID != nil && *vlanID < 0 {
		return AssociateTrunkInterfaceResult{}, ErrInvalidParameter
	}
	if greKey != nil && *greKey < 0 {
		return AssociateTrunkInterfaceResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.networkInterfaces[branchInterfaceID] == nil || s.networkInterfaces[trunkInterfaceID] == nil {
		return AssociateTrunkInterfaceResult{}, ErrNotFound
	}

	for _, existing := range s.trunkInterfaceAssociations {
		if existing.BranchInterfaceID == branchInterfaceID {
			return AssociateTrunkInterfaceResult{}, ErrConflict
		}
		if existing.TrunkInterfaceID == trunkInterfaceID && existing.VlanID != nil && vlanID != nil && *existing.VlanID == *vlanID {
			return AssociateTrunkInterfaceResult{}, ErrConflict
		}
	}

	associationID := s.nextIDLocked("trunk-assoc")
	if clientToken == "" {
		clientToken = associationID
	}
	interfaceProtocol := "VLAN"
	if greKey != nil {
		interfaceProtocol = "GRE"
	}
	association := &TrunkInterfaceAssociation{
		AssociationID:     associationID,
		BranchInterfaceID: branchInterfaceID,
		ClientToken:       clientToken,
		GreKey:            cloneInt32Pointer(greKey),
		InterfaceProtocol: interfaceProtocol,
		Tags:              map[string]string{},
		TrunkInterfaceID:  trunkInterfaceID,
		VlanID:            cloneInt32Pointer(vlanID),
	}
	s.trunkInterfaceAssociations[associationID] = association

	return AssociateTrunkInterfaceResult{
		ClientToken:          clientToken,
		InterfaceAssociation: cloneTrunkInterfaceAssociation(association),
	}, nil
}

func (s *Service) DisassociateTrunkInterface(associationID, clientToken string) (DisassociateTrunkInterfaceResult, error) {
	associationID = strings.TrimSpace(associationID)
	clientToken = strings.TrimSpace(clientToken)
	if associationID == "" {
		return DisassociateTrunkInterfaceResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	association := s.trunkInterfaceAssociations[associationID]
	if association == nil {
		return DisassociateTrunkInterfaceResult{}, ErrNotFound
	}
	delete(s.trunkInterfaceAssociations, associationID)

	if clientToken == "" {
		clientToken = association.ClientToken
	}
	return DisassociateTrunkInterfaceResult{
		ClientToken: clientToken,
		Return:      true,
	}, nil
}

func transitGatewayMulticastDomainAssocKey(domainID, attachmentID string) string {
	return strings.TrimSpace(domainID) + "|" + strings.TrimSpace(attachmentID)
}

func transitGatewayPolicyTableAssocKey(tableID, attachmentID string) string {
	return strings.TrimSpace(tableID) + "|" + strings.TrimSpace(attachmentID)
}

func transitGatewayRouteTableAssocKey(tableID, attachmentID string) string {
	return strings.TrimSpace(tableID) + "|" + strings.TrimSpace(attachmentID)
}

func cloneTransitGatewayMulticastDomainAssociation(in *TransitGatewayMulticastDomainAssociations) TransitGatewayMulticastDomainAssociations {
	if in == nil {
		return TransitGatewayMulticastDomainAssociations{}
	}
	out := *in
	out.Subnets = append([]TransitGatewaySubnetAssociation(nil), in.Subnets...)
	return out
}

func cloneTrunkInterfaceAssociation(in *TrunkInterfaceAssociation) TrunkInterfaceAssociation {
	if in == nil {
		return TrunkInterfaceAssociation{}
	}
	out := *in
	out.GreKey = cloneInt32Pointer(in.GreKey)
	out.VlanID = cloneInt32Pointer(in.VlanID)
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func cloneInt32Pointer(in *int32) *int32 {
	if in == nil {
		return nil
	}
	value := *in
	return &value
}

func dedupeTrimmedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
