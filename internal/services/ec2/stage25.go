package ec2

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
)

type SecurityGroupReference struct {
	GroupID                string
	ReferencingVpcID       string
	TransitGatewayID       string
	VpcPeeringConnectionID string
}

type SecurityGroupRule struct {
	SecurityGroupRuleID  string
	SecurityGroupRuleARN string
	GroupID              string
	GroupOwnerID         string
	IPProtocol           string
	FromPort             int32
	ToPort               int32
	IsEgress             bool
	CidrIPv4             string
	Description          string
}

type SecurityGroupRuleDescription struct {
	SecurityGroupRuleID string
	Description         string
}

func (s *Service) DescribeSecurityGroupReferences(groupIDs []string) ([]SecurityGroupReference, error) {
	if len(groupIDs) == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]SecurityGroupReference, 0, len(groupIDs))
	seen := map[string]struct{}{}
	for _, groupID := range groupIDs {
		groupID = strings.TrimSpace(groupID)
		if groupID == "" {
			continue
		}
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}

		group := s.securityGroups[groupID]
		if group == nil {
			return nil, ErrNotFound
		}
		out = append(out, SecurityGroupReference{
			GroupID:          group.ID,
			ReferencingVpcID: group.VpcID,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupID < out[j].GroupID })
	return out, nil
}

func (s *Service) DescribeSecurityGroupRules(securityGroupRuleIDs, groupIDs []string, maxResults *int32, nextToken *string) ([]SecurityGroupRule, *string, error) {
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

	groupFilter := toStringSet(groupIDs)
	ruleFilter := toStringSet(securityGroupRuleIDs)

	rules := make([]SecurityGroupRule, 0)
	for _, group := range s.securityGroups {
		if len(groupFilter) > 0 {
			if _, ok := groupFilter[group.ID]; !ok {
				continue
			}
		}

		for _, perm := range group.Ingress {
			rule := securityGroupRuleFromPermission(group, false, perm)
			if len(ruleFilter) > 0 {
				if _, ok := ruleFilter[rule.SecurityGroupRuleID]; !ok {
					continue
				}
			}
			rules = append(rules, rule)
		}
		for _, perm := range group.Egress {
			rule := securityGroupRuleFromPermission(group, true, perm)
			if len(ruleFilter) > 0 {
				if _, ok := ruleFilter[rule.SecurityGroupRuleID]; !ok {
					continue
				}
			}
			rules = append(rules, rule)
		}
	}

	sort.Slice(rules, func(i, j int) bool { return rules[i].SecurityGroupRuleID < rules[j].SecurityGroupRuleID })

	if start > len(rules) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(rules)
	if maxResults != nil {
		if *maxResults < 0 {
			return nil, nil, ErrInvalidParameter
		}
		if *maxResults > 0 {
			end = start + int(*maxResults)
			if end > len(rules) {
				end = len(rules)
			}
		}
	}

	out := append([]SecurityGroupRule(nil), rules[start:end]...)
	var outputToken *string
	if end < len(rules) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return out, outputToken, nil
}

func (s *Service) UpdateSecurityGroupRuleDescriptionsIngress(groupID, groupName, vpcID string, perms []IPPermission, descriptions []SecurityGroupRuleDescription) (bool, error) {
	if len(perms) == 0 && len(descriptions) == 0 {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return false, ErrNotFound
	}

	if err := s.updateSecurityGroupRuleDescriptionsLocked(group, false, perms, descriptions); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) UpdateSecurityGroupRuleDescriptionsEgress(groupID, groupName, vpcID string, perms []IPPermission, descriptions []SecurityGroupRuleDescription) (bool, error) {
	if len(perms) == 0 && len(descriptions) == 0 {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	group := s.resolveSecurityGroupLocked(groupID, groupName, vpcID)
	if group == nil {
		return false, ErrNotFound
	}

	if err := s.updateSecurityGroupRuleDescriptionsLocked(group, true, perms, descriptions); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) updateSecurityGroupRuleDescriptionsLocked(group *SecurityGroup, isEgress bool, perms []IPPermission, descriptions []SecurityGroupRuleDescription) error {
	target := &group.Ingress
	if isEgress {
		target = &group.Egress
	}

	for _, desc := range descriptions {
		desc.SecurityGroupRuleID = strings.TrimSpace(desc.SecurityGroupRuleID)
		desc.Description = strings.TrimSpace(desc.Description)
		if desc.SecurityGroupRuleID == "" {
			return ErrInvalidParameter
		}
		idx := findSecurityGroupPermissionIndexByRuleID(group.ID, isEgress, *target, desc.SecurityGroupRuleID)
		if idx < 0 {
			return ErrNotFound
		}
		(*target)[idx].Description = desc.Description
	}

	for _, perm := range perms {
		perm = normalizePermission(perm)
		if perm.Protocol == "" {
			return ErrInvalidParameter
		}
		idx := findSecurityGroupPermissionIndex(*target, perm)
		if idx < 0 {
			return ErrNotFound
		}
		(*target)[idx].Description = strings.TrimSpace(perm.Description)
	}

	return nil
}

func findSecurityGroupPermissionIndex(perms []IPPermission, expected IPPermission) int {
	expected = normalizePermission(expected)
	for i := range perms {
		if samePermissionIdentity(perms[i], expected) {
			return i
		}
	}
	return -1
}

func findSecurityGroupPermissionIndexByRuleID(groupID string, isEgress bool, perms []IPPermission, ruleID string) int {
	ruleID = strings.TrimSpace(ruleID)
	for i := range perms {
		if securityGroupRuleIDForPermission(groupID, isEgress, perms[i]) == ruleID {
			return i
		}
	}
	return -1
}

func securityGroupRuleFromPermission(group *SecurityGroup, isEgress bool, perm IPPermission) SecurityGroupRule {
	perm = normalizePermission(perm)
	ruleID := securityGroupRuleIDForPermission(group.ID, isEgress, perm)
	return SecurityGroupRule{
		SecurityGroupRuleID:  ruleID,
		SecurityGroupRuleARN: fmt.Sprintf("arn:aws:ec2:%s:%s:security-group-rule/%s", DefaultRegion, DefaultAccountID, ruleID),
		GroupID:              group.ID,
		GroupOwnerID:         DefaultAccountID,
		IPProtocol:           perm.Protocol,
		FromPort:             perm.FromPort,
		ToPort:               perm.ToPort,
		IsEgress:             isEgress,
		CidrIPv4:             perm.CidrIP,
		Description:          perm.Description,
	}
}

func securityGroupRuleIDForPermission(groupID string, isEgress bool, perm IPPermission) string {
	perm = normalizePermission(perm)
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(groupID))
	_, _ = hasher.Write([]byte{0})
	if isEgress {
		_, _ = hasher.Write([]byte{1})
	} else {
		_, _ = hasher.Write([]byte{0})
	}
	_, _ = hasher.Write([]byte(perm.Protocol))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strconv.FormatInt(int64(perm.FromPort), 10)))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strconv.FormatInt(int64(perm.ToPort), 10)))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(perm.CidrIP))
	return fmt.Sprintf("sgr-%016x", hasher.Sum64())
}
