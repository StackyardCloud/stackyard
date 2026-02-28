package ec2

import (
	"sort"
	"strings"
	"time"
)

type TransitGatewayAttachmentAssociation struct {
	State                      string
	TransitGatewayRouteTableID string
}

type TransitGatewayAttachment struct {
	Association                *TransitGatewayAttachmentAssociation
	CreationTime               time.Time
	ResourceID                 string
	ResourceOwnerID            string
	ResourceType               string
	State                      string
	Tags                       map[string]string
	TransitGatewayAttachmentID string
	TransitGatewayID           string
	TransitGatewayOwnerID      string
}

func (s *Service) AcceptTransitGatewayVpcAttachment(transitGatewayAttachmentID string) (TransitGatewayVpcAttachment, error) {
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
	attachment.State = "available"
	return cloneTransitGatewayVpcAttachment(attachment), nil
}

func (s *Service) RejectTransitGatewayVpcAttachment(transitGatewayAttachmentID string) (TransitGatewayVpcAttachment, error) {
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
	attachment.State = "rejected"
	return cloneTransitGatewayVpcAttachment(attachment), nil
}

func (s *Service) AcceptTransitGatewayPeeringAttachment(transitGatewayAttachmentID string) (TransitGatewayPeeringAttachment, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayPeeringAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	attachment := s.transitGatewayPeeringAttachments[transitGatewayAttachmentID]
	if attachment == nil {
		return TransitGatewayPeeringAttachment{}, ErrNotFound
	}
	attachment.State = "available"
	return cloneTransitGatewayPeeringAttachment(attachment), nil
}

func (s *Service) RejectTransitGatewayPeeringAttachment(transitGatewayAttachmentID string) (TransitGatewayPeeringAttachment, error) {
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayAttachmentID == "" {
		return TransitGatewayPeeringAttachment{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	attachment := s.transitGatewayPeeringAttachments[transitGatewayAttachmentID]
	if attachment == nil {
		return TransitGatewayPeeringAttachment{}, ErrNotFound
	}
	attachment.State = "rejected"
	return cloneTransitGatewayPeeringAttachment(attachment), nil
}

func (s *Service) AcceptTransitGatewayMulticastDomainAssociations(
	transitGatewayMulticastDomainID,
	transitGatewayAttachmentID string,
	subnetIDs []string,
) (TransitGatewayMulticastDomainAssociations, error) {
	return s.AssociateTransitGatewayMulticastDomain(
		transitGatewayMulticastDomainID,
		transitGatewayAttachmentID,
		subnetIDs,
	)
}

func (s *Service) RejectTransitGatewayMulticastDomainAssociations(
	transitGatewayMulticastDomainID,
	transitGatewayAttachmentID string,
	subnetIDs []string,
) (TransitGatewayMulticastDomainAssociations, error) {
	return s.DisassociateTransitGatewayMulticastDomain(
		transitGatewayMulticastDomainID,
		transitGatewayAttachmentID,
		subnetIDs,
	)
}

func (s *Service) DescribeTransitGatewayAttachments(
	transitGatewayAttachmentIDs []string,
	filters map[string][]string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayAttachment, *string, error) {
	start, err := ec2PageStart(nextToken)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, nil, ErrInvalidParameter
	}

	standard, tagKeys, tagFilters := splitEC2Filters(filters)

	attachmentIDSet := toStringSet(append(dedupeTrimmedStrings(transitGatewayAttachmentIDs), standard["transit-gateway-attachment-id"]...))
	associationStateSet := toLowerStringSet(standard["association.state"])
	associationRouteTableIDSet := toStringSet(standard["association.transit-gateway-route-table-id"])
	resourceIDSet := toStringSet(standard["resource-id"])
	resourceOwnerIDSet := toStringSet(standard["resource-owner-id"])
	resourceTypeSet := toLowerStringSet(standard["resource-type"])
	stateSet := toLowerStringSet(standard["state"])
	transitGatewayIDSet := toStringSet(standard["transit-gateway-id"])
	transitGatewayOwnerIDSet := toStringSet(standard["transit-gateway-owner-id"])

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayAttachment, 0, len(s.transitGatewayVpcAttachments)+len(s.transitGatewayPeeringAttachments)+len(s.transitGatewayConnects))

	appendIfMatch := func(attachment TransitGatewayAttachment) {
		if len(attachmentIDSet) > 0 {
			if _, ok := attachmentIDSet[attachment.TransitGatewayAttachmentID]; !ok {
				return
			}
		}
		if len(resourceIDSet) > 0 {
			if _, ok := resourceIDSet[attachment.ResourceID]; !ok {
				return
			}
		}
		if len(resourceOwnerIDSet) > 0 {
			if _, ok := resourceOwnerIDSet[attachment.ResourceOwnerID]; !ok {
				return
			}
		}
		if len(resourceTypeSet) > 0 {
			if _, ok := resourceTypeSet[strings.ToLower(attachment.ResourceType)]; !ok {
				return
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(attachment.State)]; !ok {
				return
			}
		}
		if len(transitGatewayIDSet) > 0 {
			if _, ok := transitGatewayIDSet[attachment.TransitGatewayID]; !ok {
				return
			}
		}
		if len(transitGatewayOwnerIDSet) > 0 {
			if _, ok := transitGatewayOwnerIDSet[attachment.TransitGatewayOwnerID]; !ok {
				return
			}
		}
		if len(associationStateSet) > 0 {
			if attachment.Association == nil {
				return
			}
			if _, ok := associationStateSet[strings.ToLower(attachment.Association.State)]; !ok {
				return
			}
		}
		if len(associationRouteTableIDSet) > 0 {
			if attachment.Association == nil {
				return
			}
			if _, ok := associationRouteTableIDSet[attachment.Association.TransitGatewayRouteTableID]; !ok {
				return
			}
		}
		if !matchesTagFilters(attachment.Tags, tagKeys, tagFilters) {
			return
		}
		out = append(out, attachment)
	}

	for _, attachment := range s.transitGatewayVpcAttachments {
		appendIfMatch(TransitGatewayAttachment{
			Association:                s.findTransitGatewayAttachmentAssociationLocked(attachment.ID),
			CreationTime:               attachment.CreationTime,
			ResourceID:                 attachment.VpcID,
			ResourceOwnerID:            attachment.VpcOwnerID,
			ResourceType:               "vpc",
			State:                      attachment.State,
			Tags:                       cloneStringMap(attachment.Tags),
			TransitGatewayAttachmentID: attachment.ID,
			TransitGatewayID:           attachment.TransitGatewayID,
			TransitGatewayOwnerID:      DefaultAccountID,
		})
	}
	for _, attachment := range s.transitGatewayPeeringAttachments {
		appendIfMatch(TransitGatewayAttachment{
			Association:                s.findTransitGatewayAttachmentAssociationLocked(attachment.ID),
			CreationTime:               attachment.CreationTime,
			ResourceID:                 attachment.AccepterTgwInfo.TransitGatewayID,
			ResourceOwnerID:            attachment.AccepterTgwInfo.OwnerID,
			ResourceType:               "peering",
			State:                      attachment.State,
			Tags:                       cloneStringMap(attachment.Tags),
			TransitGatewayAttachmentID: attachment.ID,
			TransitGatewayID:           attachment.RequesterTgwInfo.TransitGatewayID,
			TransitGatewayOwnerID:      attachment.RequesterTgwInfo.OwnerID,
		})
	}
	for _, attachment := range s.transitGatewayConnects {
		appendIfMatch(TransitGatewayAttachment{
			Association:                s.findTransitGatewayAttachmentAssociationLocked(attachment.ID),
			CreationTime:               attachment.CreationTime,
			ResourceID:                 attachment.TransportTransitGatewayAttachmentID,
			ResourceOwnerID:            DefaultAccountID,
			ResourceType:               "connect",
			State:                      attachment.State,
			Tags:                       cloneStringMap(attachment.Tags),
			TransitGatewayAttachmentID: attachment.ID,
			TransitGatewayID:           attachment.TransitGatewayID,
			TransitGatewayOwnerID:      DefaultAccountID,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].TransitGatewayAttachmentID < out[j].TransitGatewayAttachmentID })

	start, end, outputToken, err := ec2PageWindow(len(out), start, maxResults)
	if err != nil {
		return nil, nil, ErrInvalidParameter
	}
	return append([]TransitGatewayAttachment(nil), out[start:end]...), outputToken, nil
}

func (s *Service) findTransitGatewayAttachmentAssociationLocked(transitGatewayAttachmentID string) *TransitGatewayAttachmentAssociation {
	var selected *TransitGatewayAttachmentAssociation
	for _, association := range s.transitGatewayRouteTableAssocs {
		if association.TransitGatewayAttachmentID != transitGatewayAttachmentID {
			continue
		}
		candidate := &TransitGatewayAttachmentAssociation{
			State:                      association.State,
			TransitGatewayRouteTableID: association.TransitGatewayRouteTableID,
		}
		if selected == nil || candidate.TransitGatewayRouteTableID < selected.TransitGatewayRouteTableID {
			selected = candidate
		}
	}
	return selected
}
