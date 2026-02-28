package ec2

import (
	"sort"
	"strconv"
	"strings"
)

type VpcClassicLink struct {
	VpcID              string
	ClassicLinkEnabled bool
	Tags               map[string]string
}

type VpcClassicLinkDnsSupport struct {
	VpcID                   string
	ClassicLinkDnsSupported bool
}

type ClassicLinkInstance struct {
	InstanceID string
	VpcID      string
	GroupIDs   []string
	Tags       map[string]string
}

type classicLinkAttachment struct {
	InstanceID string
	VpcID      string
	GroupIDs   []string
}

func (s *Service) EnableVpcClassicLink(vpcID string) (bool, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return false, ErrNotFound
	}
	s.vpcClassicLinkEnabled[vpcID] = true
	return true, nil
}

func (s *Service) DisableVpcClassicLink(vpcID string) (bool, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return false, ErrNotFound
	}
	for _, attachment := range s.classicLinkAttachments {
		if attachment.VpcID == vpcID {
			return false, ErrConflict
		}
	}
	delete(s.vpcClassicLinkEnabled, vpcID)
	delete(s.vpcClassicLinkDnsSupported, vpcID)
	return true, nil
}

func (s *Service) EnableVpcClassicLinkDnsSupport(vpcID string) (bool, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return false, ErrNotFound
	}
	s.vpcClassicLinkDnsSupported[vpcID] = true
	return true, nil
}

func (s *Service) DisableVpcClassicLinkDnsSupport(vpcID string) (bool, error) {
	vpcID = strings.TrimSpace(vpcID)
	if vpcID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return false, ErrNotFound
	}
	delete(s.vpcClassicLinkDnsSupported, vpcID)
	return true, nil
}

func (s *Service) DescribeVpcClassicLink(vpcIDs []string, classicLinkEnabled []bool) []VpcClassicLink {
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := toStringSet(vpcIDs)
	enabledSet := map[bool]struct{}{}
	for _, enabled := range classicLinkEnabled {
		enabledSet[enabled] = struct{}{}
	}

	out := make([]VpcClassicLink, 0, len(s.vpcs))
	for _, vpc := range s.vpcs {
		if len(idSet) > 0 {
			if _, ok := idSet[vpc.ID]; !ok {
				continue
			}
		}

		enabled := s.vpcClassicLinkEnabled[vpc.ID]
		if len(enabledSet) > 0 {
			if _, ok := enabledSet[enabled]; !ok {
				continue
			}
		}

		out = append(out, VpcClassicLink{
			VpcID:              vpc.ID,
			ClassicLinkEnabled: enabled,
			Tags:               cloneStringMap(vpc.Tags),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].VpcID < out[j].VpcID })
	return out
}

func (s *Service) DescribeVpcClassicLinkDnsSupport(vpcIDs []string, maxResults *int32, nextToken *string) ([]VpcClassicLinkDnsSupport, *string, error) {
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

	idSet := toStringSet(vpcIDs)
	items := make([]VpcClassicLinkDnsSupport, 0, len(s.vpcs))
	for _, vpc := range s.vpcs {
		if len(idSet) > 0 {
			if _, ok := idSet[vpc.ID]; !ok {
				continue
			}
		}
		items = append(items, VpcClassicLinkDnsSupport{
			VpcID:                   vpc.ID,
			ClassicLinkDnsSupported: s.vpcClassicLinkDnsSupported[vpc.ID],
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].VpcID < items[j].VpcID })

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

	out := append([]VpcClassicLinkDnsSupport(nil), items[start:end]...)
	var outputToken *string
	if end < len(items) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return out, outputToken, nil
}

func (s *Service) AttachClassicLinkVpc(instanceID, vpcID string, groupIDs []string) (bool, error) {
	instanceID = strings.TrimSpace(instanceID)
	vpcID = strings.TrimSpace(vpcID)
	if instanceID == "" || vpcID == "" {
		return false, ErrInvalidParameter
	}

	normalizedGroups := normalizeClassicLinkGroupIDs(groupIDs)
	if len(normalizedGroups) == 0 {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	instance := s.instances[instanceID]
	if instance == nil {
		return false, ErrNotFound
	}
	if instance.StateName != "running" {
		return false, ErrConflict
	}
	if s.vpcs[vpcID] == nil {
		return false, ErrNotFound
	}
	if !s.vpcClassicLinkEnabled[vpcID] {
		return false, ErrConflict
	}

	for _, groupID := range normalizedGroups {
		group := s.securityGroups[groupID]
		if group == nil {
			return false, ErrNotFound
		}
		if group.VpcID != vpcID {
			return false, ErrConflict
		}
	}

	if existing := s.classicLinkAttachments[instanceID]; existing != nil {
		if existing.VpcID != vpcID {
			return false, ErrConflict
		}
		if !classicLinkGroupsEqual(existing.GroupIDs, normalizedGroups) {
			return false, ErrConflict
		}
		return true, nil
	}

	s.classicLinkAttachments[instanceID] = &classicLinkAttachment{
		InstanceID: instanceID,
		VpcID:      vpcID,
		GroupIDs:   append([]string(nil), normalizedGroups...),
	}
	return true, nil
}

func (s *Service) DetachClassicLinkVpc(instanceID, vpcID string) (bool, error) {
	instanceID = strings.TrimSpace(instanceID)
	vpcID = strings.TrimSpace(vpcID)
	if instanceID == "" || vpcID == "" {
		return false, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.vpcs[vpcID] == nil {
		return false, ErrNotFound
	}

	attachment := s.classicLinkAttachments[instanceID]
	if attachment == nil || attachment.VpcID != vpcID {
		return false, ErrNotFound
	}
	delete(s.classicLinkAttachments, instanceID)
	return true, nil
}

func (s *Service) DescribeClassicLinkInstances(instanceIDs, filterVpcIDs, filterGroupIDs []string, maxResults *int32, nextToken *string) ([]ClassicLinkInstance, *string, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	instanceIDSet := toStringSet(instanceIDs)
	vpcIDSet := toStringSet(filterVpcIDs)
	groupIDSet := toStringSet(filterGroupIDs)

	items := make([]ClassicLinkInstance, 0, len(s.classicLinkAttachments))
	for _, attachment := range s.classicLinkAttachments {
		if len(instanceIDSet) > 0 {
			if _, ok := instanceIDSet[attachment.InstanceID]; !ok {
				continue
			}
		}
		if len(vpcIDSet) > 0 {
			if _, ok := vpcIDSet[attachment.VpcID]; !ok {
				continue
			}
		}
		if len(groupIDSet) > 0 && !classicLinkContainsAnyGroup(attachment.GroupIDs, groupIDSet) {
			continue
		}

		instance := s.instances[attachment.InstanceID]
		tags := map[string]string{}
		if instance != nil {
			tags = cloneStringMap(instance.Tags)
		}

		items = append(items, ClassicLinkInstance{
			InstanceID: attachment.InstanceID,
			VpcID:      attachment.VpcID,
			GroupIDs:   append([]string(nil), attachment.GroupIDs...),
			Tags:       tags,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].InstanceID < items[j].InstanceID })

	if start > len(items) {
		return nil, nil, ErrInvalidParameter
	}
	end := len(items)
	if maxResults != nil && *maxResults > 0 {
		end = start + int(*maxResults)
		if end > len(items) {
			end = len(items)
		}
	}

	out := append([]ClassicLinkInstance(nil), items[start:end]...)
	var outputToken *string
	if end < len(items) {
		token := strconv.Itoa(end)
		outputToken = &token
	}
	return out, outputToken, nil
}

func normalizeClassicLinkGroupIDs(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
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

func classicLinkGroupsEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftSet := toStringSet(left)
	for _, groupID := range right {
		if _, ok := leftSet[groupID]; !ok {
			return false
		}
	}
	return true
}

func classicLinkContainsAnyGroup(groupIDs []string, filter map[string]struct{}) bool {
	for _, groupID := range groupIDs {
		if _, ok := filter[groupID]; ok {
			return true
		}
	}
	return false
}
