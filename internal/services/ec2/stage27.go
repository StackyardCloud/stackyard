package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type SecurityGroupForVPC struct {
	Description  string
	GroupID      string
	GroupName    string
	OwnerID      string
	PrimaryVpcID string
	Tags         map[string]string
}

type StaleSecurityGroup struct {
	Description              string
	GroupID                  string
	GroupName                string
	StaleIPPermissions       []StaleIPPermission
	StaleIPPermissionsEgress []StaleIPPermission
	VpcID                    string
}

type StaleIPPermission struct {
	FromPort         *int32
	IPProtocol       *string
	IPRanges         []string
	PrefixListIDs    []string
	ToPort           *int32
	UserIDGroupPairs []StaleUserIDGroupPair
}

type StaleUserIDGroupPair struct {
	GroupID                string
	GroupName              string
	PeeringStatus          string
	UserID                 string
	VpcID                  string
	VpcPeeringConnectionID string
}

type SecurityGroupRuleUpdateRequest struct {
	SecurityGroupRuleID string
	CidrIPv4            *string
	CidrIPv6            *string
	Description         *string
	FromPort            *int32
	IPProtocol          *string
	PrefixListID        *string
	ReferencedGroupID   *string
	ToPort              *int32
}

func (s *Service) GetSecurityGroupsForVpc(
	vpcID string,
	groupIDs, descriptions, groupNames, ownerIDs, primaryVpcIDs []string,
	maxResults *int32,
	nextToken *string,
) ([]SecurityGroupForVPC, *string, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return nil, nil, ErrNotFound
	}

	groupIDSet := toStringSet(groupIDs)
	descriptionSet := toStringSet(descriptions)
	groupNameSet := toStringSet(groupNames)
	ownerIDSet := toStringSet(ownerIDs)
	primaryVpcIDSet := toStringSet(primaryVpcIDs)

	candidateGroupIDs := map[string]struct{}{}
	for _, group := range s.securityGroups {
		if group.VpcID == vpcID {
			candidateGroupIDs[group.ID] = struct{}{}
		}
	}
	for _, association := range s.securityGroupVpcAssociations {
		if association.VpcID == vpcID && association.State == "associated" {
			candidateGroupIDs[association.GroupID] = struct{}{}
		}
	}

	out := make([]SecurityGroupForVPC, 0, len(candidateGroupIDs))
	for groupID := range candidateGroupIDs {
		group := s.securityGroups[groupID]
		if group == nil {
			continue
		}
		if len(groupIDSet) > 0 {
			if _, ok := groupIDSet[group.ID]; !ok {
				continue
			}
		}
		if len(descriptionSet) > 0 {
			if _, ok := descriptionSet[group.Description]; !ok {
				continue
			}
		}
		if len(groupNameSet) > 0 {
			if _, ok := groupNameSet[group.Name]; !ok {
				continue
			}
		}
		if len(ownerIDSet) > 0 {
			if _, ok := ownerIDSet[DefaultAccountID]; !ok {
				continue
			}
		}
		if len(primaryVpcIDSet) > 0 {
			if _, ok := primaryVpcIDSet[group.VpcID]; !ok {
				continue
			}
		}
		out = append(out, SecurityGroupForVPC{
			Description:  group.Description,
			GroupID:      group.ID,
			GroupName:    group.Name,
			OwnerID:      DefaultAccountID,
			PrimaryVpcID: group.VpcID,
			Tags:         cloneStringMap(group.Tags),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })

	if start > len(out) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(out)
	if maxResults != nil {
		if *maxResults < 0 {
			return nil, nil, ErrInvalidParameter
		}
		if *maxResults > 0 {
			end = start + int(*maxResults)
			if end > len(out) {
				end = len(out)
			}
		}
	}

	page := append([]SecurityGroupForVPC(nil), out[start:end]...)
	var outputToken *string
	if end < len(out) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) DescribeStaleSecurityGroups(vpcID string, maxResults *int32, nextToken *string) ([]StaleSecurityGroup, *string, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return nil, nil, ErrNotFound
	}

	groups := make([]StaleSecurityGroup, 0)

	if start > len(groups) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(groups)
	if maxResults != nil {
		if *maxResults < 0 {
			return nil, nil, ErrInvalidParameter
		}
		if *maxResults > 0 {
			end = start + int(*maxResults)
			if end > len(groups) {
				end = len(groups)
			}
		}
	}

	page := append([]StaleSecurityGroup(nil), groups[start:end]...)
	var outputToken *string
	if end < len(groups) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return page, outputToken, nil
}

func (s *Service) ModifySecurityGroupRules(groupID string, updates []SecurityGroupRuleUpdateRequest) (bool, error) {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" || len(updates) == 0 {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.securityGroups[groupID]
	if group == nil {
		return false, ErrNotFound
	}

	for _, update := range updates {
		if err := applySecurityGroupRuleUpdate(group, update); err != nil {
			return false, err
		}
	}

	return true, nil
}

func applySecurityGroupRuleUpdate(group *SecurityGroup, update SecurityGroupRuleUpdateRequest) error {
	ruleID := strings.TrimSpace(update.SecurityGroupRuleID)
	if ruleID == "" {
		return ErrInvalidParameter
	}

	if nonEmptyOptionalString(update.CidrIPv6) || nonEmptyOptionalString(update.PrefixListID) || nonEmptyOptionalString(update.ReferencedGroupID) {
		return ErrInvalidParameter
	}

	idx := findSecurityGroupPermissionIndexByRuleID(group.ID, false, group.Ingress, ruleID)
	target := &group.Ingress
	if idx < 0 {
		idx = findSecurityGroupPermissionIndexByRuleID(group.ID, true, group.Egress, ruleID)
		if idx < 0 {
			return ErrNotFound
		}
		target = &group.Egress
	}

	next := (*target)[idx]
	if update.CidrIPv4 != nil {
		next.CidrIP = strings.TrimSpace(*update.CidrIPv4)
	}
	if update.Description != nil {
		next.Description = strings.TrimSpace(*update.Description)
	}
	if update.FromPort != nil {
		next.FromPort = *update.FromPort
	}
	if update.IPProtocol != nil {
		next.Protocol = strings.TrimSpace(*update.IPProtocol)
	}
	if update.ToPort != nil {
		next.ToPort = *update.ToPort
	}

	next = normalizePermission(next)
	if next.Protocol == "" {
		return ErrInvalidParameter
	}

	for i, current := range *target {
		if i == idx {
			continue
		}
		if samePermissionIdentity(current, next) {
			return ErrAlreadyExists
		}
	}

	(*target)[idx] = next
	return nil
}

func nonEmptyOptionalString(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
