package ec2

import (
	"sort"
	"strings"
)

type ModifyTransitGatewayOptionsInput struct {
	AddTransitGatewayCidrBlocks     []string
	AmazonSideASN                   *int64
	AssociationDefaultRouteTableID  *string
	AutoAcceptSharedAttachments     *string
	DefaultRouteTableAssociation    *string
	DefaultRouteTablePropagation    *string
	DnsSupport                      *string
	PropagationDefaultRouteTableID  *string
	RemoveTransitGatewayCidrBlocks  []string
	SecurityGroupReferencingSupport *string
	VpnEcmpSupport                  *string
}

func (s *Service) ModifyTransitGateway(
	transitGatewayID string,
	description *string,
	options ModifyTransitGatewayOptionsInput,
) (TransitGateway, error) {
	transitGatewayID = strings.TrimSpace(transitGatewayID)
	if transitGatewayID == "" {
		return TransitGateway{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	gateway := s.transitGateways[transitGatewayID]
	if gateway == nil {
		return TransitGateway{}, ErrNotFound
	}

	if description != nil {
		gateway.Description = strings.TrimSpace(*description)
	}
	if options.AmazonSideASN != nil {
		gateway.Options.AmazonSideASN = *options.AmazonSideASN
	}
	if options.AssociationDefaultRouteTableID != nil {
		routeTableID := strings.TrimSpace(*options.AssociationDefaultRouteTableID)
		if routeTableID == "" {
			return TransitGateway{}, ErrInvalidParameter
		}
		routeTable := s.transitGatewayRouteTables[routeTableID]
		if routeTable == nil || routeTable.TransitID != transitGatewayID {
			return TransitGateway{}, ErrNotFound
		}
		gateway.Options.AssociationDefaultRouteTableID = routeTableID
	}
	if options.PropagationDefaultRouteTableID != nil {
		routeTableID := strings.TrimSpace(*options.PropagationDefaultRouteTableID)
		if routeTableID == "" {
			return TransitGateway{}, ErrInvalidParameter
		}
		routeTable := s.transitGatewayRouteTables[routeTableID]
		if routeTable == nil || routeTable.TransitID != transitGatewayID {
			return TransitGateway{}, ErrNotFound
		}
		gateway.Options.PropagationDefaultRouteTableID = routeTableID
	}

	gateway.Options.AutoAcceptSharedAttachments = applyTransitGatewayOptionValue(gateway.Options.AutoAcceptSharedAttachments, options.AutoAcceptSharedAttachments)
	gateway.Options.DefaultRouteTableAssociation = applyTransitGatewayOptionValue(gateway.Options.DefaultRouteTableAssociation, options.DefaultRouteTableAssociation)
	gateway.Options.DefaultRouteTablePropagation = applyTransitGatewayOptionValue(gateway.Options.DefaultRouteTablePropagation, options.DefaultRouteTablePropagation)
	gateway.Options.DnsSupport = applyTransitGatewayOptionValue(gateway.Options.DnsSupport, options.DnsSupport)
	gateway.Options.SecurityGroupReferencingSupport = applyTransitGatewayOptionValue(gateway.Options.SecurityGroupReferencingSupport, options.SecurityGroupReferencingSupport)
	gateway.Options.VpnEcmpSupport = applyTransitGatewayOptionValue(gateway.Options.VpnEcmpSupport, options.VpnEcmpSupport)

	if len(options.AddTransitGatewayCidrBlocks) > 0 {
		existing := toStringSet(gateway.Options.TransitGatewayCidrBlocks)
		for _, cidr := range dedupeTrimmedStrings(options.AddTransitGatewayCidrBlocks) {
			if cidr == "" {
				continue
			}
			if _, ok := existing[cidr]; ok {
				continue
			}
			existing[cidr] = struct{}{}
			gateway.Options.TransitGatewayCidrBlocks = append(gateway.Options.TransitGatewayCidrBlocks, cidr)
		}
	}
	if len(options.RemoveTransitGatewayCidrBlocks) > 0 {
		removeSet := toStringSet(options.RemoveTransitGatewayCidrBlocks)
		filtered := gateway.Options.TransitGatewayCidrBlocks[:0]
		for _, cidr := range gateway.Options.TransitGatewayCidrBlocks {
			if _, remove := removeSet[cidr]; remove {
				continue
			}
			filtered = append(filtered, cidr)
		}
		gateway.Options.TransitGatewayCidrBlocks = filtered
	}

	return cloneTransitGateway(gateway), nil
}

func (s *Service) ModifyTransitGatewayVpcAttachment(
	transitGatewayAttachmentID string,
	addSubnetIDs,
	removeSubnetIDs []string,
	options TransitGatewayVpcAttachmentOptionsInput,
) (TransitGatewayVpcAttachment, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayVpcAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	attachment := s.transitGatewayVpcAttachments[transitGatewayAttachmentID]
	if attachment == nil {
		return TransitGatewayVpcAttachment{}, ErrNotFound
	}

	currentSubnetSet := toStringSet(attachment.SubnetIDs)
	for _, subnetID := range dedupeTrimmedStrings(removeSubnetIDs) {
		delete(currentSubnetSet, subnetID)
	}
	for _, subnetID := range dedupeTrimmedStrings(addSubnetIDs) {
		subnet := s.subnets[subnetID]
		if subnet == nil || subnet.VpcID != attachment.VpcID {
			return TransitGatewayVpcAttachment{}, ErrNotFound
		}
		currentSubnetSet[subnetID] = struct{}{}
	}

	if len(currentSubnetSet) == 0 {
		return TransitGatewayVpcAttachment{}, ErrInvalidParameter
	}

	updatedSubnetIDs := make([]string, 0, len(currentSubnetSet))
	for subnetID := range currentSubnetSet {
		updatedSubnetIDs = append(updatedSubnetIDs, subnetID)
	}
	sort.Strings(updatedSubnetIDs)
	attachment.SubnetIDs = updatedSubnetIDs

	if options.ApplianceModeSupport != nil {
		attachment.Options.ApplianceModeSupport = normalizeOptionValue(options.ApplianceModeSupport, attachment.Options.ApplianceModeSupport)
	}
	if options.DnsSupport != nil {
		attachment.Options.DnsSupport = normalizeOptionValue(options.DnsSupport, attachment.Options.DnsSupport)
	}
	if options.Ipv6Support != nil {
		attachment.Options.Ipv6Support = normalizeOptionValue(options.Ipv6Support, attachment.Options.Ipv6Support)
	}
	if options.SecurityGroupReferencingSupport != nil {
		attachment.Options.SecurityGroupReferencingSupport = normalizeOptionValue(options.SecurityGroupReferencingSupport, attachment.Options.SecurityGroupReferencingSupport)
	}

	return cloneTransitGatewayVpcAttachment(attachment), nil
}

func (s *Service) ModifyTransitGatewayPrefixListReference(
	transitGatewayRouteTableID,
	prefixListID string,
	blackhole *bool,
	transitGatewayAttachmentID *string,
	hasTransitGatewayAttachmentID bool,
) (TransitGatewayPrefixListReference, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	prefixListID = strings.TrimSpace(prefixListID)
	if transitGatewayRouteTableID == "" || prefixListID == "" {
		return TransitGatewayPrefixListReference{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := transitGatewayPrefixListReferenceKey(transitGatewayRouteTableID, prefixListID)
	reference := s.transitGatewayPrefixListReferences[key]
	if reference == nil {
		return TransitGatewayPrefixListReference{}, ErrNotFound
	}

	if blackhole != nil {
		reference.Blackhole = *blackhole
	}

	if hasTransitGatewayAttachmentID {
		trimmedAttachmentID := ""
		if transitGatewayAttachmentID != nil {
			trimmedAttachmentID = strings.TrimSpace(*transitGatewayAttachmentID)
		}
		if trimmedAttachmentID == "" {
			reference.TransitGatewayAttachment = nil
		} else {
			reference.TransitGatewayAttachment = &TransitGatewayPrefixListAttachment{
				ResourceID:                 trimmedAttachmentID,
				ResourceType:               "vpc",
				TransitGatewayAttachmentID: trimmedAttachmentID,
			}
		}
	} else if reference.Blackhole {
		reference.TransitGatewayAttachment = nil
	}

	if !reference.Blackhole && reference.TransitGatewayAttachment == nil {
		return TransitGatewayPrefixListReference{}, ErrInvalidParameter
	}

	reference.State = "available"
	return cloneTransitGatewayPrefixListReference(reference), nil
}

func applyTransitGatewayOptionValue(current string, updated *string) string {
	if updated == nil {
		return current
	}
	return normalizeOptionValue(updated, current)
}
