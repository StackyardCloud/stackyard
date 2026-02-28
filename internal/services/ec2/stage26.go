package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type SecurityGroupVpcAssociation struct {
	GroupID      string
	GroupOwnerID string
	State        string
	StateReason  string
	VpcID        string
	VpcOwnerID   string
}

func (s *Service) AssociateSecurityGroupVpc(groupID, vpcID string) (string, error) {
	groupID = strings.TrimSpace(groupID)
	vpcID = strings.TrimSpace(vpcID)
	if groupID == "" || vpcID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.securityGroups[groupID]
	if group == nil {
		return "", ErrNotFound
	}
	vpc := s.vpcs[vpcID]
	if vpc == nil {
		return "", ErrNotFound
	}
	if groupID == "sg-00000000" || vpc.IsDefault {
		return "", ErrConflict
	}

	key := securityGroupVpcAssociationKey(groupID, vpcID)
	if existing := s.securityGroupVpcAssociations[key]; existing != nil {
		existing.State = "associated"
		existing.StateReason = ""
		return existing.State, nil
	}
	s.securityGroupVpcAssociations[key] = &SecurityGroupVpcAssociation{
		GroupID:      groupID,
		GroupOwnerID: DefaultAccountID,
		State:        "associated",
		StateReason:  "",
		VpcID:        vpcID,
		VpcOwnerID:   DefaultAccountID,
	}
	return "associated", nil
}

func (s *Service) DisassociateSecurityGroupVpc(groupID, vpcID string) (string, error) {
	groupID = strings.TrimSpace(groupID)
	vpcID = strings.TrimSpace(vpcID)
	if groupID == "" || vpcID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := securityGroupVpcAssociationKey(groupID, vpcID)
	assoc := s.securityGroupVpcAssociations[key]
	if assoc == nil {
		return "", ErrNotFound
	}

	// Mirror AWS behavior: disassociation is blocked while resources in the
	// associated VPC still reference the security group.
	for _, iface := range s.networkInterfaces {
		if iface.VpcID != vpcID {
			continue
		}
		for _, attachedGroupID := range iface.GroupIDs {
			if attachedGroupID == groupID {
				return "", ErrConflict
			}
		}
	}
	for _, inst := range s.instances {
		if inst.VpcID != vpcID || inst.StateName == "terminated" {
			continue
		}
		for _, attachedGroupID := range inst.SecurityGroupIDs {
			if attachedGroupID == groupID {
				return "", ErrConflict
			}
		}
	}

	delete(s.securityGroupVpcAssociations, key)
	return "disassociated", nil
}

func (s *Service) DescribeSecurityGroupVpcAssociations(
	groupIDs, groupOwnerIDs, states, vpcIDs, vpcOwnerIDs []string,
	maxResults *int32, nextToken *string,
) ([]SecurityGroupVpcAssociation, *string, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	groupIDSet := toStringSet(groupIDs)
	groupOwnerIDSet := toStringSet(groupOwnerIDs)
	stateSet := toStringSet(states)
	vpcIDSet := toStringSet(vpcIDs)
	vpcOwnerIDSet := toStringSet(vpcOwnerIDs)

	items := make([]SecurityGroupVpcAssociation, 0, len(s.securityGroupVpcAssociations))
	for _, assoc := range s.securityGroupVpcAssociations {
		if len(groupIDSet) > 0 {
			if _, ok := groupIDSet[assoc.GroupID]; !ok {
				continue
			}
		}
		if len(groupOwnerIDSet) > 0 {
			if _, ok := groupOwnerIDSet[assoc.GroupOwnerID]; !ok {
				continue
			}
		}
		if len(stateSet) > 0 {
			if _, ok := stateSet[assoc.State]; !ok {
				continue
			}
		}
		if len(vpcIDSet) > 0 {
			if _, ok := vpcIDSet[assoc.VpcID]; !ok {
				continue
			}
		}
		if len(vpcOwnerIDSet) > 0 {
			if _, ok := vpcOwnerIDSet[assoc.VpcOwnerID]; !ok {
				continue
			}
		}
		items = append(items, *assoc)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].GroupID != items[j].GroupID {
			return items[i].GroupID < items[j].GroupID
		}
		return items[i].VpcID < items[j].VpcID
	})

	if start > len(items) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(items)
	if maxResults != nil {
		if *maxResults < 0 {
			return nil, nil, ErrInvalidParameter
		}
		if *maxResults > 0 {
			end = start + int(*maxResults)
			if end > len(items) {
				end = len(items)
			}
		}
	}

	out := append([]SecurityGroupVpcAssociation(nil), items[start:end]...)
	var outputToken *string
	if end < len(items) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return out, outputToken, nil
}

func securityGroupVpcAssociationKey(groupID, vpcID string) string {
	return strings.TrimSpace(groupID) + "|" + strings.TrimSpace(vpcID)
}
