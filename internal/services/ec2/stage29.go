package ec2

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type VpcCidrAssociation struct {
	AssociationID string
	VpcID         string
	CidrBlock     string
	State         string
	StatusMessage string
}

type VpcIPv6CidrAssociation struct {
	AssociationID        string
	VpcID                string
	IPv6CidrBlock        string
	IPv6Pool             string
	NetworkBorderGroup   string
	IPSource             string
	IPv6AddressAttribute string
	State                string
	StatusMessage        string
}

type SubnetIPv6CidrAssociation struct {
	AssociationID        string
	SubnetID             string
	IPv6CidrBlock        string
	IPSource             string
	IPv6AddressAttribute string
	State                string
	StatusMessage        string
}

type SubnetCidrReservation struct {
	ID              string
	SubnetID        string
	Cidr            string
	Description     string
	OwnerID         string
	ReservationType string
	Tags            map[string]string
}

type AssociateVpcCidrBlockResult struct {
	VpcID                string
	CidrBlockAssociation *VpcCidrAssociation
	IPv6CidrAssociation  *VpcIPv6CidrAssociation
}

type AssociateSubnetCidrBlockResult struct {
	SubnetID            string
	IPv6CidrAssociation *SubnetIPv6CidrAssociation
}

type GetSubnetCidrReservationsResult struct {
	SubnetIPv4CidrReservations []SubnetCidrReservation
	SubnetIPv6CidrReservations []SubnetCidrReservation
	NextToken                  *string
}

func (s *Service) AssociateVpcCidrBlock(
	vpcID,
	cidrBlock,
	ipv4IPAMPoolID string,
	ipv4NetmaskLength *int32,
	amazonProvidedIPv6CidrBlock bool,
	ipv6CidrBlock,
	ipv6Pool,
	ipv6IPAMPoolID,
	ipv6CidrBlockNetworkBorderGroup string,
	ipv6NetmaskLength *int32,
) (AssociateVpcCidrBlockResult, error) {
	vpcID = strings.TrimSpace(vpcID)
	cidrBlock = strings.TrimSpace(cidrBlock)
	ipv4IPAMPoolID = strings.TrimSpace(ipv4IPAMPoolID)
	ipv6CidrBlock = strings.TrimSpace(ipv6CidrBlock)
	ipv6Pool = strings.TrimSpace(ipv6Pool)
	ipv6IPAMPoolID = strings.TrimSpace(ipv6IPAMPoolID)
	ipv6CidrBlockNetworkBorderGroup = strings.TrimSpace(ipv6CidrBlockNetworkBorderGroup)

	if vpcID == "" {
		return AssociateVpcCidrBlockResult{}, ErrInvalidParameter
	}

	hasIPv4 := cidrBlock != "" || ipv4IPAMPoolID != "" || ipv4NetmaskLength != nil
	hasIPv6 := amazonProvidedIPv6CidrBlock || ipv6CidrBlock != "" || ipv6Pool != "" || ipv6IPAMPoolID != "" || ipv6NetmaskLength != nil
	if hasIPv4 == hasIPv6 {
		return AssociateVpcCidrBlockResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return AssociateVpcCidrBlockResult{}, ErrNotFound
	}

	if hasIPv4 {
		resolvedCIDR := cidrBlock
		if resolvedCIDR == "" {
			netmask := int32(24)
			if ipv4NetmaskLength != nil {
				netmask = *ipv4NetmaskLength
			}
			resolvedCIDR = s.nextGeneratedVpcIPv4CIDRLocked(netmask)
		}

		for _, assoc := range s.vpcCidrAssociations {
			if assoc.VpcID == vpcID && assoc.CidrBlock == resolvedCIDR {
				return AssociateVpcCidrBlockResult{}, ErrAlreadyExists
			}
		}

		association := &VpcCidrAssociation{
			AssociationID: s.nextIDLocked("vpc-cidr-assoc"),
			VpcID:         vpcID,
			CidrBlock:     resolvedCIDR,
			State:         "associated",
			StatusMessage: "",
		}
		s.vpcCidrAssociations[association.AssociationID] = association

		out := cloneVpcCidrAssociation(association)
		return AssociateVpcCidrBlockResult{VpcID: vpcID, CidrBlockAssociation: &out}, nil
	}

	resolvedIPv6CIDR := ipv6CidrBlock
	if resolvedIPv6CIDR == "" {
		netmask := int32(64)
		if ipv6NetmaskLength != nil {
			netmask = *ipv6NetmaskLength
		}
		if amazonProvidedIPv6CidrBlock {
			if netmask == 0 {
				netmask = 56
			}
		}
		resolvedIPv6CIDR = s.nextGeneratedVpcIPv6CIDRLocked(netmask)
	}
	for _, assoc := range s.vpcIPv6CidrAssociations {
		if assoc.VpcID == vpcID && assoc.IPv6CidrBlock == resolvedIPv6CIDR {
			return AssociateVpcCidrBlockResult{}, ErrAlreadyExists
		}
	}

	ipSource := ""
	if amazonProvidedIPv6CidrBlock {
		ipSource = "amazon"
	} else if ipv6Pool != "" || ipv6IPAMPoolID != "" {
		ipSource = "byoip"
	}

	association := &VpcIPv6CidrAssociation{
		AssociationID:      s.nextIDLocked("vpc-cidr-assoc"),
		VpcID:              vpcID,
		IPv6CidrBlock:      resolvedIPv6CIDR,
		IPv6Pool:           firstNonEmptyString(ipv6Pool, ipv6IPAMPoolID),
		NetworkBorderGroup: ipv6CidrBlockNetworkBorderGroup,
		IPSource:           ipSource,
		State:              "associated",
		StatusMessage:      "",
	}
	s.vpcIPv6CidrAssociations[association.AssociationID] = association

	out := cloneVpcIPv6CidrAssociation(association)
	return AssociateVpcCidrBlockResult{VpcID: vpcID, IPv6CidrAssociation: &out}, nil
}

func (s *Service) DisassociateVpcCidrBlock(associationID string) (AssociateVpcCidrBlockResult, error) {
	associationID = strings.TrimSpace(associationID)
	if associationID == "" {
		return AssociateVpcCidrBlockResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if assoc := s.vpcCidrAssociations[associationID]; assoc != nil {
		out := cloneVpcCidrAssociation(assoc)
		out.State = "disassociated"
		delete(s.vpcCidrAssociations, associationID)
		return AssociateVpcCidrBlockResult{VpcID: out.VpcID, CidrBlockAssociation: &out}, nil
	}
	if assoc := s.vpcIPv6CidrAssociations[associationID]; assoc != nil {
		out := cloneVpcIPv6CidrAssociation(assoc)
		out.State = "disassociated"
		delete(s.vpcIPv6CidrAssociations, associationID)
		return AssociateVpcCidrBlockResult{VpcID: out.VpcID, IPv6CidrAssociation: &out}, nil
	}

	return AssociateVpcCidrBlockResult{}, ErrNotFound
}

func (s *Service) AssociateSubnetCidrBlock(
	subnetID,
	ipv6CidrBlock,
	ipv6IPAMPoolID string,
	ipv6NetmaskLength *int32,
) (AssociateSubnetCidrBlockResult, error) {
	subnetID = strings.TrimSpace(subnetID)
	ipv6CidrBlock = strings.TrimSpace(ipv6CidrBlock)
	ipv6IPAMPoolID = strings.TrimSpace(ipv6IPAMPoolID)

	if subnetID == "" {
		return AssociateSubnetCidrBlockResult{}, ErrInvalidParameter
	}
	if ipv6CidrBlock == "" && ipv6IPAMPoolID == "" && ipv6NetmaskLength == nil {
		return AssociateSubnetCidrBlockResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subnets[subnetID] == nil {
		return AssociateSubnetCidrBlockResult{}, ErrNotFound
	}

	resolvedIPv6CIDR := ipv6CidrBlock
	if resolvedIPv6CIDR == "" {
		netmask := int32(64)
		if ipv6NetmaskLength != nil {
			netmask = *ipv6NetmaskLength
		}
		resolvedIPv6CIDR = s.nextGeneratedSubnetIPv6CIDRLocked(netmask)
	}
	for _, assoc := range s.subnetIPv6CidrAssociations {
		if assoc.SubnetID == subnetID && assoc.IPv6CidrBlock == resolvedIPv6CIDR {
			return AssociateSubnetCidrBlockResult{}, ErrAlreadyExists
		}
	}

	association := &SubnetIPv6CidrAssociation{
		AssociationID: s.nextIDLocked("subnet-cidr-assoc"),
		SubnetID:      subnetID,
		IPv6CidrBlock: resolvedIPv6CIDR,
		IPSource:      "",
		State:         "associated",
		StatusMessage: "",
	}
	if ipv6IPAMPoolID != "" {
		association.IPSource = "byoip"
	}
	s.subnetIPv6CidrAssociations[association.AssociationID] = association

	out := cloneSubnetIPv6CidrAssociation(association)
	return AssociateSubnetCidrBlockResult{SubnetID: subnetID, IPv6CidrAssociation: &out}, nil
}

func (s *Service) DisassociateSubnetCidrBlock(associationID string) (AssociateSubnetCidrBlockResult, error) {
	associationID = strings.TrimSpace(associationID)
	if associationID == "" {
		return AssociateSubnetCidrBlockResult{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	assoc := s.subnetIPv6CidrAssociations[associationID]
	if assoc == nil {
		return AssociateSubnetCidrBlockResult{}, ErrNotFound
	}

	out := cloneSubnetIPv6CidrAssociation(assoc)
	out.State = "disassociated"
	delete(s.subnetIPv6CidrAssociations, associationID)
	return AssociateSubnetCidrBlockResult{SubnetID: out.SubnetID, IPv6CidrAssociation: &out}, nil
}

func (s *Service) CreateSubnetCidrReservation(subnetID, cidr, description, reservationType string, tags []Tag) (SubnetCidrReservation, error) {
	subnetID = strings.TrimSpace(subnetID)
	cidr = strings.TrimSpace(cidr)
	description = strings.TrimSpace(description)
	reservationType = strings.ToLower(strings.TrimSpace(reservationType))

	if subnetID == "" || cidr == "" || reservationType == "" {
		return SubnetCidrReservation{}, ErrInvalidParameter
	}
	if reservationType != "explicit" && reservationType != "prefix" {
		return SubnetCidrReservation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.subnets[subnetID] == nil {
		return SubnetCidrReservation{}, ErrNotFound
	}

	for _, reservation := range s.subnetCidrReservations {
		if reservation.SubnetID == subnetID && reservation.Cidr == cidr {
			return SubnetCidrReservation{}, ErrAlreadyExists
		}
	}

	reservation := &SubnetCidrReservation{
		ID:              s.nextIDLocked("scr"),
		SubnetID:        subnetID,
		Cidr:            cidr,
		Description:     description,
		OwnerID:         DefaultAccountID,
		ReservationType: reservationType,
		Tags:            tagsToMap(tags),
	}
	s.subnetCidrReservations[reservation.ID] = reservation
	return cloneSubnetCidrReservation(reservation), nil
}

func (s *Service) DeleteSubnetCidrReservation(subnetCidrReservationID string) (SubnetCidrReservation, error) {
	subnetCidrReservationID = strings.TrimSpace(subnetCidrReservationID)
	if subnetCidrReservationID == "" {
		return SubnetCidrReservation{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reservation := s.subnetCidrReservations[subnetCidrReservationID]
	if reservation == nil {
		return SubnetCidrReservation{}, ErrNotFound
	}

	out := cloneSubnetCidrReservation(reservation)
	delete(s.subnetCidrReservations, subnetCidrReservationID)
	return out, nil
}

func (s *Service) GetSubnetCidrReservations(
	subnetID string,
	cidrs,
	ownerIDs,
	reservationTypes,
	subnetCidrReservationIDs,
	subnetIDs []string,
	maxResults *int32,
	nextToken *string,
) (GetSubnetCidrReservationsResult, error) {
	subnetID = strings.TrimSpace(subnetID)

	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return GetSubnetCidrReservationsResult{}, ErrInvalidParameter
			}
			start = parsed
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if subnetID != "" && s.subnets[subnetID] == nil {
		return GetSubnetCidrReservationsResult{}, ErrNotFound
	}
	if maxResults != nil && *maxResults < 0 {
		return GetSubnetCidrReservationsResult{}, ErrInvalidParameter
	}

	cidrSet := toStringSet(cidrs)
	ownerIDSet := toStringSet(ownerIDs)
	reservationTypeSet := toLowerStringSet(reservationTypes)
	subnetCidrReservationIDSet := toStringSet(subnetCidrReservationIDs)
	subnetIDSet := toStringSet(subnetIDs)

	items := make([]SubnetCidrReservation, 0, len(s.subnetCidrReservations))
	for _, reservation := range s.subnetCidrReservations {
		if subnetID != "" && reservation.SubnetID != subnetID {
			continue
		}
		if len(cidrSet) > 0 {
			if _, ok := cidrSet[reservation.Cidr]; !ok {
				continue
			}
		}
		if len(ownerIDSet) > 0 {
			if _, ok := ownerIDSet[reservation.OwnerID]; !ok {
				continue
			}
		}
		if len(reservationTypeSet) > 0 {
			if _, ok := reservationTypeSet[strings.ToLower(reservation.ReservationType)]; !ok {
				continue
			}
		}
		if len(subnetCidrReservationIDSet) > 0 {
			if _, ok := subnetCidrReservationIDSet[reservation.ID]; !ok {
				continue
			}
		}
		if len(subnetIDSet) > 0 {
			if _, ok := subnetIDSet[reservation.SubnetID]; !ok {
				continue
			}
		}
		items = append(items, cloneSubnetCidrReservation(reservation))
	}

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	if start > len(items) {
		return GetSubnetCidrReservationsResult{}, ErrInvalidParameter
	}
	end := len(items)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(items) {
			end = len(items)
		}
	}

	page := append([]SubnetCidrReservation(nil), items[start:end]...)
	ipv4Reservations := make([]SubnetCidrReservation, 0, len(page))
	ipv6Reservations := make([]SubnetCidrReservation, 0, len(page))
	for _, reservation := range page {
		if strings.Contains(reservation.Cidr, ":") {
			ipv6Reservations = append(ipv6Reservations, reservation)
			continue
		}
		ipv4Reservations = append(ipv4Reservations, reservation)
	}

	var outputToken *string
	if end < len(items) {
		token := strconv.Itoa(end)
		outputToken = &token
	}

	return GetSubnetCidrReservationsResult{
		SubnetIPv4CidrReservations: ipv4Reservations,
		SubnetIPv6CidrReservations: ipv6Reservations,
		NextToken:                  outputToken,
	}, nil
}

func cloneVpcCidrAssociation(in *VpcCidrAssociation) VpcCidrAssociation {
	if in == nil {
		return VpcCidrAssociation{}
	}
	return *in
}

func cloneVpcIPv6CidrAssociation(in *VpcIPv6CidrAssociation) VpcIPv6CidrAssociation {
	if in == nil {
		return VpcIPv6CidrAssociation{}
	}
	return *in
}

func cloneSubnetIPv6CidrAssociation(in *SubnetIPv6CidrAssociation) SubnetIPv6CidrAssociation {
	if in == nil {
		return SubnetIPv6CidrAssociation{}
	}
	return *in
}

func cloneSubnetCidrReservation(in *SubnetCidrReservation) SubnetCidrReservation {
	if in == nil {
		return SubnetCidrReservation{}
	}
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}

func (s *Service) nextGeneratedVpcIPv4CIDRLocked(netmask int32) string {
	if netmask <= 0 {
		netmask = 24
	}
	index := len(s.vpcCidrAssociations) + 1
	secondOctet := (index % 200) + 16
	return fmt.Sprintf("10.%d.0.0/%d", secondOctet, netmask)
}

func (s *Service) nextGeneratedVpcIPv6CIDRLocked(netmask int32) string {
	if netmask <= 0 {
		netmask = 64
	}
	index := len(s.vpcIPv6CidrAssociations) + 1
	return fmt.Sprintf("2001:db8:%x::/%d", index, netmask)
}

func (s *Service) nextGeneratedSubnetIPv6CIDRLocked(netmask int32) string {
	if netmask <= 0 {
		netmask = 64
	}
	index := len(s.subnetIPv6CidrAssociations) + 1
	return fmt.Sprintf("2001:db8:%x::/%d", index+0x100, netmask)
}

func toLowerStringSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
