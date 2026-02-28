package ec2

import "strings"

func (s *Service) AssociateInstanceEventWindow(
	instanceEventWindowID string,
	dedicatedHostIDs []string,
	instanceIDs []string,
	instanceTags []Tag,
) (InstanceEventWindowAssociation, error) {
	instanceEventWindowID = strings.TrimSpace(instanceEventWindowID)
	if instanceEventWindowID == "" {
		return InstanceEventWindowAssociation{}, ErrInvalidParameter
	}

	dedicatedHostIDs = dedupeTrimmedStrings(dedicatedHostIDs)
	instanceIDs = dedupeTrimmedStrings(instanceIDs)
	instanceTags = normalizeEC2Tags(instanceTags)

	targetKinds := 0
	if len(dedicatedHostIDs) > 0 {
		targetKinds++
	}
	if len(instanceIDs) > 0 {
		targetKinds++
	}
	if len(instanceTags) > 0 {
		targetKinds++
	}
	if targetKinds != 1 {
		return InstanceEventWindowAssociation{}, ErrInvalidParameter
	}

	association := InstanceEventWindowAssociation{
		InstanceEventWindowID: instanceEventWindowID,
		DedicatedHostIDs:      append([]string(nil), dedicatedHostIDs...),
		InstanceIDs:           append([]string(nil), instanceIDs...),
		InstanceTags:          cloneEC2Tags(instanceTags),
		State:                 "active",
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.instanceEventWindowAssociations[instanceEventWindowID] = association
	return cloneInstanceEventWindowAssociation(association), nil
}

func cloneInstanceEventWindowAssociation(in InstanceEventWindowAssociation) InstanceEventWindowAssociation {
	out := in
	out.DedicatedHostIDs = append([]string(nil), in.DedicatedHostIDs...)
	out.InstanceIDs = append([]string(nil), in.InstanceIDs...)
	out.InstanceTags = cloneEC2Tags(in.InstanceTags)
	return out
}

func cloneEC2Tags(in []Tag) []Tag {
	out := make([]Tag, 0, len(in))
	for _, tag := range in {
		out = append(out, tag)
	}
	return out
}

func normalizeEC2Tags(in []Tag) []Tag {
	seen := map[string]struct{}{}
	out := make([]Tag, 0, len(in))
	for _, tag := range in {
		key := strings.TrimSpace(tag.Key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Tag{
			Key:   key,
			Value: strings.TrimSpace(tag.Value),
		})
	}
	return out
}
