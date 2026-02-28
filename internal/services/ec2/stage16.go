package ec2

import (
	"sort"
	"strings"
)

type PlacementGroup struct {
	GroupARN       string
	GroupID        string
	GroupName      string
	PartitionCount int32
	SpreadLevel    string
	State          string
	Strategy       string
	Tags           map[string]string
}

func (s *Service) CreatePlacementGroup(name, strategy string, partitionCount int32, spreadLevel string, tags []Tag) (PlacementGroup, error) {
	name = strings.TrimSpace(name)
	strategy = strings.ToLower(strings.TrimSpace(strategy))
	spreadLevel = strings.ToLower(strings.TrimSpace(spreadLevel))
	if name == "" {
		return PlacementGroup{}, ErrInvalidParameter
	}
	if strategy == "" {
		strategy = "cluster"
	}
	if strategy != "cluster" && strategy != "spread" && strategy != "partition" {
		return PlacementGroup{}, ErrInvalidParameter
	}
	if spreadLevel != "" && spreadLevel != "host" && spreadLevel != "rack" {
		return PlacementGroup{}, ErrInvalidParameter
	}
	if strategy == "partition" && partitionCount <= 0 {
		partitionCount = 1
	}
	if strategy != "partition" {
		partitionCount = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.placementGroupByName[name]; exists {
		return PlacementGroup{}, ErrAlreadyExists
	}

	group := &PlacementGroup{
		GroupARN:       "arn:aws:ec2:" + DefaultRegion + ":" + DefaultAccountID + ":placement-group/" + name,
		GroupID:        s.nextIDLocked("pg"),
		GroupName:      name,
		PartitionCount: partitionCount,
		SpreadLevel:    spreadLevel,
		State:          "available",
		Strategy:       strategy,
		Tags:           tagsToMap(tags),
	}
	s.placementGroups[group.GroupID] = group
	s.placementGroupByName[group.GroupName] = group.GroupID
	return clonePlacementGroup(group), nil
}

func (s *Service) DescribePlacementGroups(groupNames, groupIDs []string) []PlacementGroup {
	s.mu.Lock()
	defer s.mu.Unlock()

	nameSet := toStringSet(groupNames)
	idSet := toStringSet(groupIDs)

	out := make([]PlacementGroup, 0, len(s.placementGroups))
	for _, group := range s.placementGroups {
		if len(nameSet) > 0 {
			if _, ok := nameSet[group.GroupName]; !ok {
				continue
			}
		}
		if len(idSet) > 0 {
			if _, ok := idSet[group.GroupID]; !ok {
				continue
			}
		}
		out = append(out, clonePlacementGroup(group))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GroupName < out[j].GroupName })
	return out
}

func (s *Service) DeletePlacementGroup(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	groupID := s.placementGroupByName[name]
	if groupID == "" {
		return ErrNotFound
	}
	delete(s.placementGroups, groupID)
	delete(s.placementGroupByName, name)
	return nil
}

func clonePlacementGroup(in *PlacementGroup) PlacementGroup {
	out := *in
	out.Tags = cloneStringMap(in.Tags)
	return out
}
