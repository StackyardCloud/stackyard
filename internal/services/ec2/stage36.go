package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type TransitGatewayPrefixListAttachment struct {
	ResourceID                 string
	ResourceType               string
	TransitGatewayAttachmentID string
}

type TransitGatewayPrefixListReference struct {
	Blackhole                  bool
	PrefixListID               string
	PrefixListOwnerID          string
	State                      string
	TransitGatewayAttachment   *TransitGatewayPrefixListAttachment
	TransitGatewayRouteTableID string
}

type TransitGatewayPolicyRuleMetaData struct {
	MetaDataKey   string
	MetaDataValue string
}

type TransitGatewayPolicyRule struct {
	DestinationCidrBlock string
	DestinationPortRange string
	MetaData             *TransitGatewayPolicyRuleMetaData
	Protocol             string
	SourceCidrBlock      string
	SourcePortRange      string
}

type TransitGatewayPolicyTableEntry struct {
	PolicyRule         TransitGatewayPolicyRule
	PolicyRuleNumber   string
	TargetRouteTableID string
}

func (s *Service) CreateTransitGatewayPrefixListReference(
	transitGatewayRouteTableID,
	prefixListID string,
	blackhole bool,
	transitGatewayAttachmentID string,
) (TransitGatewayPrefixListReference, error) {
	transitGatewayRouteTableID = strings.TrimSpace(transitGatewayRouteTableID)
	prefixListID = strings.TrimSpace(prefixListID)
	transitGatewayAttachmentID = strings.TrimSpace(transitGatewayAttachmentID)
	if transitGatewayRouteTableID == "" || prefixListID == "" {
		return TransitGatewayPrefixListReference{}, ErrInvalidParameter
	}
	if transitGatewayAttachmentID == "" && !blackhole {
		return TransitGatewayPrefixListReference{}, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.transitGatewayRouteTables[transitGatewayRouteTableID] == nil {
		return TransitGatewayPrefixListReference{}, ErrNotFound
	}

	key := transitGatewayPrefixListReferenceKey(transitGatewayRouteTableID, prefixListID)
	reference := s.transitGatewayPrefixListReferences[key]
	if reference == nil {
		reference = &TransitGatewayPrefixListReference{
			PrefixListOwnerID: DefaultAccountID,
		}
		s.transitGatewayPrefixListReferences[key] = reference
	}

	reference.Blackhole = blackhole
	reference.PrefixListID = prefixListID
	reference.PrefixListOwnerID = DefaultAccountID
	reference.State = "available"
	reference.TransitGatewayRouteTableID = transitGatewayRouteTableID
	if transitGatewayAttachmentID != "" {
		reference.TransitGatewayAttachment = &TransitGatewayPrefixListAttachment{
			ResourceID:                 transitGatewayAttachmentID,
			ResourceType:               "vpc",
			TransitGatewayAttachmentID: transitGatewayAttachmentID,
		}
	} else {
		reference.TransitGatewayAttachment = nil
	}

	return cloneTransitGatewayPrefixListReference(reference), nil
}

func (s *Service) DeleteTransitGatewayPrefixListReference(
	transitGatewayRouteTableID,
	prefixListID string,
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

	out := cloneTransitGatewayPrefixListReference(reference)
	out.State = "deleting"
	delete(s.transitGatewayPrefixListReferences, key)
	return out, nil
}

func (s *Service) GetTransitGatewayPrefixListReferences(
	transitGatewayRouteTableID string,
	attachmentResourceIDs,
	attachmentResourceTypes,
	transitGatewayAttachmentIDs []string,
	isBlackholeFilters []bool,
	prefixListIDs,
	prefixListOwnerIDs,
	states []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayPrefixListReference, *string, error) {
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

	attachmentResourceIDSet := toStringSet(attachmentResourceIDs)
	attachmentResourceTypeSet := toLowerStringSet(attachmentResourceTypes)
	transitGatewayAttachmentIDSet := toStringSet(transitGatewayAttachmentIDs)
	isBlackholeSet := toBoolSet(isBlackholeFilters)
	prefixListIDSet := toStringSet(prefixListIDs)
	prefixListOwnerIDSet := toStringSet(prefixListOwnerIDs)
	stateSet := toLowerStringSet(states)

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]TransitGatewayPrefixListReference, 0)
	for _, reference := range s.transitGatewayPrefixListReferences {
		if reference.TransitGatewayRouteTableID != transitGatewayRouteTableID {
			continue
		}
		if len(attachmentResourceIDSet) > 0 {
			if reference.TransitGatewayAttachment == nil {
				continue
			}
			if _, ok := attachmentResourceIDSet[reference.TransitGatewayAttachment.ResourceID]; !ok {
				continue
			}
		}
		if len(attachmentResourceTypeSet) > 0 {
			if reference.TransitGatewayAttachment == nil {
				continue
			}
			if _, ok := attachmentResourceTypeSet[strings.ToLower(reference.TransitGatewayAttachment.ResourceType)]; !ok {
				continue
			}
		}
		if len(transitGatewayAttachmentIDSet) > 0 {
			if reference.TransitGatewayAttachment == nil {
				continue
			}
			if _, ok := transitGatewayAttachmentIDSet[reference.TransitGatewayAttachment.TransitGatewayAttachmentID]; !ok {
				continue
			}
		}
		if len(isBlackholeSet) > 0 {
			if _, ok := isBlackholeSet[reference.Blackhole]; !ok {
				continue
			}
		}
		if len(prefixListIDSet) > 0 {
			if _, ok := prefixListIDSet[reference.PrefixListID]; !ok {
				continue
			}
		}
		if len(prefixListOwnerIDSet) > 0 {
			if _, ok := prefixListOwnerIDSet[reference.PrefixListOwnerID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[strings.ToLower(reference.State)]; !ok {
				continue
			}
		}
		out = append(out, cloneTransitGatewayPrefixListReference(reference))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PrefixListID != out[j].PrefixListID {
			return out[i].PrefixListID < out[j].PrefixListID
		}
		left := ""
		if out[i].TransitGatewayAttachment != nil {
			left = out[i].TransitGatewayAttachment.TransitGatewayAttachmentID
		}
		right := ""
		if out[j].TransitGatewayAttachment != nil {
			right = out[j].TransitGatewayAttachment.TransitGatewayAttachmentID
		}
		return left < right
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

	page := append([]TransitGatewayPrefixListReference(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) GetTransitGatewayPolicyTableEntries(
	transitGatewayPolicyTableID string,
	policyRuleNumbers,
	targetRouteTableIDs,
	sourceCidrBlocks,
	destinationCidrBlocks,
	protocols []string,
	maxResults *int32,
	nextToken *string,
) ([]TransitGatewayPolicyTableEntry, error) {
	transitGatewayPolicyTableID = strings.TrimSpace(transitGatewayPolicyTableID)
	if transitGatewayPolicyTableID == "" {
		return nil, ErrInvalidParameter
	}

	start := 0
	if nextToken != nil {
		token := strings.TrimSpace(*nextToken)
		if token != "" {
			parsed, err := strconv.Atoi(token)
			if err != nil || parsed < 0 {
				return nil, ErrInvalidParameter
			}
			start = parsed
		}
	}
	if maxResults != nil && *maxResults < 0 {
		return nil, ErrInvalidParameter
	}

	policyRuleNumberSet := toStringSet(policyRuleNumbers)
	targetRouteTableIDSet := toStringSet(targetRouteTableIDs)
	sourceCidrBlockSet := toStringSet(sourceCidrBlocks)
	destinationCidrBlockSet := toStringSet(destinationCidrBlocks)
	protocolSet := toLowerStringSet(protocols)

	s.mu.Lock()
	defer s.mu.Unlock()

	associations := make([]TransitGatewayPolicyTableAssociation, 0)
	for _, association := range s.transitGatewayPolicyTableAssocs {
		if association.TransitGatewayPolicyTableID != transitGatewayPolicyTableID {
			continue
		}
		associations = append(associations, *association)
	}
	sort.Slice(associations, func(i, j int) bool {
		return associations[i].TransitGatewayAttachmentID < associations[j].TransitGatewayAttachmentID
	})

	targetRouteTableByAttachment := map[string]string{}
	for _, association := range s.transitGatewayRouteTableAssocs {
		if strings.ToLower(association.State) != "associated" {
			continue
		}
		targetRouteTableByAttachment[association.TransitGatewayAttachmentID] = association.TransitGatewayRouteTableID
	}

	out := make([]TransitGatewayPolicyTableEntry, 0, len(associations))
	for index, association := range associations {
		targetRouteTableID := targetRouteTableByAttachment[association.TransitGatewayAttachmentID]
		entry := TransitGatewayPolicyTableEntry{
			PolicyRule: TransitGatewayPolicyRule{
				DestinationCidrBlock: "0.0.0.0/0",
				MetaData: &TransitGatewayPolicyRuleMetaData{
					MetaDataKey:   "resourceId",
					MetaDataValue: association.ResourceID,
				},
				Protocol:        "ALL",
				SourceCidrBlock: "0.0.0.0/0",
			},
			PolicyRuleNumber:   strconv.Itoa(index + 1),
			TargetRouteTableID: targetRouteTableID,
		}
		if len(policyRuleNumberSet) > 0 {
			if _, ok := policyRuleNumberSet[entry.PolicyRuleNumber]; !ok {
				continue
			}
		}
		if len(targetRouteTableIDSet) > 0 {
			if _, ok := targetRouteTableIDSet[entry.TargetRouteTableID]; !ok {
				continue
			}
		}
		if len(sourceCidrBlockSet) > 0 {
			if _, ok := sourceCidrBlockSet[entry.PolicyRule.SourceCidrBlock]; !ok {
				continue
			}
		}
		if len(destinationCidrBlockSet) > 0 {
			if _, ok := destinationCidrBlockSet[entry.PolicyRule.DestinationCidrBlock]; !ok {
				continue
			}
		}
		if len(protocolSet) > 0 {
			if _, ok := protocolSet[strings.ToLower(entry.PolicyRule.Protocol)]; !ok {
				continue
			}
		}
		out = append(out, entry)
	}

	if start > len(out) {
		return nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(out) {
			end = len(out)
		}
	}

	return append([]TransitGatewayPolicyTableEntry(nil), out[start:end]...), nil
}

func transitGatewayPrefixListReferenceKey(transitGatewayRouteTableID, prefixListID string) string {
	return strings.TrimSpace(transitGatewayRouteTableID) + "|" + strings.TrimSpace(prefixListID)
}

func cloneTransitGatewayPrefixListReference(in *TransitGatewayPrefixListReference) TransitGatewayPrefixListReference {
	if in == nil {
		return TransitGatewayPrefixListReference{}
	}
	out := *in
	if in.TransitGatewayAttachment != nil {
		attachment := *in.TransitGatewayAttachment
		out.TransitGatewayAttachment = &attachment
	}
	return out
}
