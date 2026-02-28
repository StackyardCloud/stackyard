package ec2

import (
	"sort"
	"strings"
)

const defaultEC2PrincipalArn = "arn:aws:iam::" + DefaultAccountID + ":root"

type IDFormatStatus struct {
	Resource   string
	UseLongIDs bool
}

type PrincipalIDFormat struct {
	ARN      string
	Statuses []IDFormatStatus
}

func (s *Service) DescribeIDFormat(resource string) ([]IDFormatStatus, error) {
	resource = strings.TrimSpace(resource)

	s.mu.Lock()
	defer s.mu.Unlock()

	return cloneIDFormatStatusesLocked(s.idFormatRoot, resource)
}

func (s *Service) DescribeIdentityIDFormat(principalARN, resource string) ([]IDFormatStatus, error) {
	principalARN = strings.TrimSpace(principalARN)
	resource = strings.TrimSpace(resource)
	if principalARN == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	overrides := s.idFormatByPrincipal[principalARN]
	return mergeAndCloneIDFormatStatusesLocked(s.idFormatRoot, overrides, resource)
}

func (s *Service) DescribePrincipalIDFormat(resources []string) []PrincipalIDFormat {
	s.mu.Lock()
	defer s.mu.Unlock()

	filter := toStringSet(resources)
	out := make([]PrincipalIDFormat, 0, len(s.idFormatByPrincipal)+1)

	rootStatuses, err := filterIDFormatStatusesLocked(s.idFormatRoot, filter)
	if err == nil {
		out = append(out, PrincipalIDFormat{
			ARN:      defaultEC2PrincipalArn,
			Statuses: rootStatuses,
		})
	}

	arns := make([]string, 0, len(s.idFormatByPrincipal))
	for arn := range s.idFormatByPrincipal {
		arns = append(arns, arn)
	}
	sort.Strings(arns)

	for _, arn := range arns {
		statuses, err := mergeAndFilterIDFormatStatusesLocked(s.idFormatRoot, s.idFormatByPrincipal[arn], filter)
		if err != nil {
			continue
		}
		out = append(out, PrincipalIDFormat{
			ARN:      arn,
			Statuses: statuses,
		})
	}
	return out
}

func (s *Service) ModifyIDFormat(resource string, useLongIDs bool) error {
	resource = normalizeIDFormatResource(resource)
	if resource == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.idFormatRoot[resource] = useLongIDs
	return nil
}

func (s *Service) ModifyIdentityIDFormat(principalARN, resource string, useLongIDs bool) error {
	principalARN = strings.TrimSpace(principalARN)
	resource = normalizeIDFormatResource(resource)
	if principalARN == "" || resource == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.idFormatRoot[resource]; !ok {
		return ErrInvalidParameter
	}
	principalState := s.idFormatByPrincipal[principalARN]
	if principalState == nil {
		principalState = map[string]bool{}
		s.idFormatByPrincipal[principalARN] = principalState
	}
	principalState[resource] = useLongIDs
	return nil
}

func (s *Service) DescribeAggregateIDFormat() ([]IDFormatStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resources := make([]string, 0, len(s.idFormatRoot))
	for resource := range s.idFormatRoot {
		resources = append(resources, resource)
	}
	sort.Strings(resources)

	statuses := make([]IDFormatStatus, 0, len(resources))
	aggregated := true
	for _, resource := range resources {
		useLongIDs := s.idFormatRoot[resource]
		if useLongIDs {
			for _, principalState := range s.idFormatByPrincipal {
				override, ok := principalState[resource]
				if ok && !override {
					useLongIDs = false
					break
				}
			}
		}
		if !useLongIDs {
			aggregated = false
		}
		statuses = append(statuses, IDFormatStatus{
			Resource:   resource,
			UseLongIDs: useLongIDs,
		})
	}

	return statuses, aggregated
}

func defaultIDFormatState() map[string]bool {
	return map[string]bool{
		"dhcp-options":      true,
		"image":             true,
		"instance":          true,
		"internet-gateway":  true,
		"network-acl":       true,
		"network-interface": true,
		"route-table":       true,
		"security-group":    true,
		"snapshot":          true,
		"subnet":            true,
		"volume":            true,
		"vpc":               true,
	}
}

func cloneIDFormatStatusesLocked(state map[string]bool, resource string) ([]IDFormatStatus, error) {
	filter := map[string]struct{}{}
	if resource != "" {
		resource = normalizeIDFormatResource(resource)
		if resource == "" {
			return nil, ErrInvalidParameter
		}
		filter[resource] = struct{}{}
	}
	return filterIDFormatStatusesLocked(state, filter)
}

func mergeAndCloneIDFormatStatusesLocked(base, overrides map[string]bool, resource string) ([]IDFormatStatus, error) {
	filter := map[string]struct{}{}
	if resource != "" {
		resource = normalizeIDFormatResource(resource)
		if resource == "" {
			return nil, ErrInvalidParameter
		}
		filter[resource] = struct{}{}
	}
	return mergeAndFilterIDFormatStatusesLocked(base, overrides, filter)
}

func mergeAndFilterIDFormatStatusesLocked(base, overrides map[string]bool, filter map[string]struct{}) ([]IDFormatStatus, error) {
	state := make(map[string]bool, len(base)+len(overrides))
	for resource, useLongIDs := range base {
		state[resource] = useLongIDs
	}
	for resource, useLongIDs := range overrides {
		state[resource] = useLongIDs
	}
	return filterIDFormatStatusesLocked(state, filter)
}

func filterIDFormatStatusesLocked(state map[string]bool, filter map[string]struct{}) ([]IDFormatStatus, error) {
	resources := make([]string, 0, len(state))
	for resource := range state {
		if len(filter) > 0 {
			if _, ok := filter[resource]; !ok {
				continue
			}
		}
		resources = append(resources, resource)
	}
	if len(filter) > 0 && len(resources) == 0 {
		return nil, ErrInvalidParameter
	}
	sort.Strings(resources)

	out := make([]IDFormatStatus, 0, len(resources))
	for _, resource := range resources {
		out = append(out, IDFormatStatus{
			Resource:   resource,
			UseLongIDs: state[resource],
		})
	}
	return out, nil
}

func normalizeIDFormatResource(resource string) string {
	resource = strings.TrimSpace(strings.ToLower(resource))
	if resource == "" {
		return ""
	}
	return resource
}
