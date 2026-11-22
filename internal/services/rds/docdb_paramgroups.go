package rds

import (
	"sort"
	"strconv"
	"strings"
)

func (s *Service) ensureDefaultClusterParamGroupsLocked() {
	for _, v := range DocDBEngineVersions {
		name := "default." + v.Family
		if _, ok := s.clusterParamGroups[name]; !ok {
			s.clusterParamGroups[name] = &DBClusterParameterGroup{
				Name: name, ARN: clusterParamGroupARN(name), Family: v.Family,
				Description: "Default cluster parameter group for " + v.Family, Values: map[string]ClusterParameter{},
			}
		}
	}
}

func (s *Service) clusterParamGroupLocked(name string) (*DBClusterParameterGroup, error) {
	s.ensureDefaultClusterParamGroupsLocked()
	group, ok := s.clusterParamGroups[strings.TrimSpace(name)]
	if !ok {
		return nil, fault(ErrNotFound, "DBParameterGroupNotFound", "DBClusterParameterGroup %s not found.", name)
	}
	return group, nil
}

func (s *Service) newClusterParamGroupLocked(name, family, description string) (*DBClusterParameterGroup, error) {
	if err := validateDocDBIdentifier("DBClusterParameterGroupName", name, 255); err != nil {
		return nil, err
	}
	if strings.HasPrefix(strings.ToLower(name), "default.") {
		return nil, fault(ErrInvalidParameter, "InvalidParameterValue", "parameter group names may not start with \"default.\"")
	}
	if !docdbKnownFamily(family) {
		return nil, fault(ErrInvalidParameter, "InvalidParameterValue", "DBParameterGroupFamily %q is not a supported family", family)
	}
	if strings.TrimSpace(description) == "" {
		return nil, fault(ErrInvalidParameter, "InvalidParameterValue", "Description is required")
	}
	s.ensureDefaultClusterParamGroupsLocked()
	if _, exists := s.clusterParamGroups[name]; exists {
		return nil, fault(ErrAlreadyExists, "DBParameterGroupAlreadyExists", "Parameter group %s already exists", name)
	}
	if len(s.clusterParamGroups) >= docdbMaxParamGroups+len(DocDBEngineVersions) {
		return nil, fault(ErrQuotaExceeded, "DBParameterGroupQuotaExceeded", "cluster parameter group quota of %d reached", docdbMaxParamGroups)
	}
	group := &DBClusterParameterGroup{Name: name, ARN: clusterParamGroupARN(name), Family: family, Description: description, Values: map[string]ClusterParameter{}}
	s.clusterParamGroups[name] = group
	return group, nil
}

func (s *Service) CreateClusterParameterGroup(name, family, description string) (DBClusterParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	group, err := s.newClusterParamGroupLocked(strings.TrimSpace(name), strings.TrimSpace(family), description)
	if err != nil {
		return DBClusterParameterGroup{}, err
	}
	return cloneClusterParamGroup(group), nil
}

// CopyClusterParameterGroup copies a group (named or by ARN) with its values.
func (s *Service) CopyClusterParameterGroup(source, target, description string) (DBClusterParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, err := s.clusterParamGroupLocked(arnOrName(source, ":cluster-pg:"))
	if err != nil {
		return DBClusterParameterGroup{}, err
	}
	group, err := s.newClusterParamGroupLocked(strings.TrimSpace(target), src.Family, description)
	if err != nil {
		return DBClusterParameterGroup{}, err
	}
	for k, v := range src.Values {
		group.Values[k] = v
	}
	return cloneClusterParamGroup(group), nil
}

func (s *Service) DeleteClusterParameterGroup(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	group, err := s.clusterParamGroupLocked(name)
	if err != nil {
		return err
	}
	if strings.HasPrefix(group.Name, "default.") {
		return fault(ErrInvalidState, "InvalidDBParameterGroupState", "default parameter groups can't be deleted")
	}
	for _, cluster := range s.clusters {
		if cluster.DBClusterParameterGroup == group.Name {
			return fault(ErrInvalidState, "InvalidDBParameterGroupState", "parameter group %s is in use by cluster %s", group.Name, cluster.Identifier)
		}
	}
	delete(s.clusterParamGroups, group.Name)
	delete(s.tags, group.ARN)
	return nil
}

func (s *Service) DescribeClusterParameterGroups(name, marker string, maxRecords int) ([]DBClusterParameterGroup, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name = strings.TrimSpace(name); name != "" {
		group, err := s.clusterParamGroupLocked(name)
		if err != nil {
			return nil, "", err
		}
		return []DBClusterParameterGroup{cloneClusterParamGroup(group)}, "", nil
	}
	s.ensureDefaultClusterParamGroupsLocked()
	names := make([]string, 0, len(s.clusterParamGroups))
	for n := range s.clusterParamGroups {
		names = append(names, n)
	}
	sort.Strings(names)
	start, end, next, err := docdbPage(len(names), marker, maxRecords)
	if err != nil {
		return nil, "", err
	}
	out := make([]DBClusterParameterGroup, 0, end-start)
	for _, n := range names[start:end] {
		out = append(out, cloneClusterParamGroup(s.clusterParamGroups[n]))
	}
	return out, next, nil
}

// ClusterParameters lists a group's parameters; source filters to "user",
// "system" or "engine-default" (the latter two both mean unmodified).
func (s *Service) ClusterParameters(name, source, marker string, maxRecords int) ([]ClusterParameter, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	group, err := s.clusterParamGroupLocked(name)
	if err != nil {
		return nil, "", err
	}
	switch source {
	case "", "user", "system", "engine-default":
	default:
		return nil, "", fault(ErrInvalidParameter, "InvalidParameterValue", "Source must be user, system or engine-default")
	}
	var params []ClusterParameter
	for _, p := range DocDBDefaultClusterParameters() {
		if v, ok := group.Values[p.Name]; ok {
			p.Value, p.Source, p.ApplyMethod = v.Value, "user", v.ApplyMethod
		}
		if source == "" || (source == "user") == (p.Source == "user") {
			params = append(params, p)
		}
	}
	start, end, next, err := docdbPage(len(params), marker, maxRecords)
	if err != nil {
		return nil, "", err
	}
	return params[start:end], next, nil
}

// DocDBDefaultClusterParameters is the engine-default parameter list.
func DocDBDefaultClusterParameters() []ClusterParameter {
	out := make([]ClusterParameter, len(docdbParameterCatalog))
	for i, p := range docdbParameterCatalog {
		p.Source, p.ApplyMethod, p.MinVersion = "system", "pending-reboot", "3.6.0"
		out[i] = p
	}
	return out
}

func validateClusterParameterValue(def ClusterParameter, value string) bool {
	if lo, hi, ok := strings.Cut(def.AllowedValues, "-"); ok && def.DataType != "string" {
		if def.DataType == "integer" {
			n, err := strconv.ParseInt(value, 10, 64)
			l, _ := strconv.ParseInt(lo, 10, 64)
			h, _ := strconv.ParseInt(hi, 10, 64)
			return err == nil && n >= l && n <= h
		}
		f, err := strconv.ParseFloat(value, 64)
		l, _ := strconv.ParseFloat(lo, 64)
		h, _ := strconv.ParseFloat(hi, 64)
		return err == nil && f >= l && f <= h
	}
	for _, allowed := range strings.Split(def.AllowedValues, ",") {
		if value == allowed {
			return true
		}
	}
	return false
}

// ModifyClusterParameterGroup sets parameter values. Static parameters only
// accept ApplyMethod pending-reboot.
func (s *Service) ModifyClusterParameterGroup(name string, params []ClusterParameter) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	group, err := s.clusterParamGroupLocked(name)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(group.Name, "default.") {
		return "", fault(ErrInvalidState, "InvalidDBParameterGroupState", "default parameter groups can't be modified")
	}
	if len(params) == 0 || len(params) > docdbMaxModifyParams {
		return "", fault(ErrInvalidParameter, "InvalidParameterValue", "between 1 and %d parameters may be modified in one request", docdbMaxModifyParams)
	}
	defs := map[string]ClusterParameter{}
	for _, p := range docdbParameterCatalog {
		defs[p.Name] = p
	}
	for _, p := range params {
		def, ok := defs[p.Name]
		if !ok {
			return "", fault(ErrInvalidParameter, "InvalidParameterValue", "Unknown parameter %q", p.Name)
		}
		if !validateClusterParameterValue(def, p.Value) {
			return "", fault(ErrInvalidParameter, "InvalidParameterValue", "Value %q is not valid for %s (allowed: %s)", p.Value, p.Name, def.AllowedValues)
		}
		switch p.ApplyMethod {
		case "pending-reboot":
		case "immediate":
			if def.ApplyType == "static" {
				return "", fault(ErrInvalidParameter, "InvalidParameterCombination", "cannot use immediate apply method for static parameter %s", p.Name)
			}
		default:
			return "", fault(ErrInvalidParameter, "InvalidParameterValue", "ApplyMethod must be immediate or pending-reboot")
		}
	}
	for _, p := range params {
		group.Values[p.Name] = ClusterParameter{Name: p.Name, Value: p.Value, ApplyMethod: p.ApplyMethod}
	}
	return group.Name, nil
}

func (s *Service) ResetClusterParameterGroup(name string, all bool, names []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	group, err := s.clusterParamGroupLocked(name)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(group.Name, "default.") {
		return "", fault(ErrInvalidState, "InvalidDBParameterGroupState", "default parameter groups can't be modified")
	}
	if all == (len(names) > 0) {
		return "", fault(ErrInvalidParameter, "InvalidParameterCombination", "specify either ResetAllParameters or Parameters, but not both")
	}
	if all {
		group.Values = map[string]ClusterParameter{}
		return group.Name, nil
	}
	for _, n := range names {
		delete(group.Values, n)
	}
	return group.Name, nil
}
