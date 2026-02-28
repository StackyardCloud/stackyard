package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type controlCatalogStore struct {
	mu sync.Mutex

	nextControl     int64
	nextObjective   int64
	nextDomain      int64
	nextCommon      int64
	nextMapping     int64
	controls        map[string]map[string]any
	domains         map[string]map[string]any
	objectives      map[string]map[string]any
	commonControls  map[string]map[string]any
	controlMappings map[string]map[string]any
}

func newControlCatalogStore() *controlCatalogStore {
	s := &controlCatalogStore{
		nextControl:     2,
		nextObjective:   2,
		nextDomain:      2,
		nextCommon:      2,
		nextMapping:     2,
		controls:        map[string]map[string]any{},
		domains:         map[string]map[string]any{},
		objectives:      map[string]map[string]any{},
		commonControls:  map[string]map[string]any{},
		controlMappings: map[string]map[string]any{},
	}

	domain := s.ensureDomainLocked("operations")
	objective := s.ensureObjectiveLocked("access-management", ccString(domain["Arn"], ""))
	control := s.ensureControlLocked("cis-1-1", ccString(objective["Arn"], ""), ccString(domain["Arn"], ""))
	common := s.ensureCommonControlLocked("ccm-iam-1", ccString(control["Arn"], ""))
	s.ensureControlMappingLocked(ccString(control["Arn"], ""), ccString(common["Arn"], ""))

	return s
}

func (s *controlCatalogStore) Handle(action string, payload map[string]any, _ map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedDataLocked()

	s.maybeApplyControlFilterLocked(payload, query)
	s.maybeApplyObjectiveFilterLocked(payload, query)
	s.maybeApplyDomainFilterLocked(payload, query)

	syncFilters := map[string]any{}
	for key, value := range payload {
		syncFilters[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		syncFilters[key] = values[len(values)-1]
	}

	switch action {
	case "GetControl":
		controlArn := ccStringAny(syncFilters, "ControlArn", "")
		if controlArn == "" {
			controlArn = ccStringAny(syncFilters, "controlArn", "")
		}
		if controlArn == "" {
			controlArn = ccFirstMapKey(s.controls)
		}
		control := s.ensureControlByArnLocked(controlArn)
		return map[string]any{"Control": ccCloneMap(control)}

	case "ListControls":
		items := make([]any, 0, len(s.controls))
		for _, control := range ccSortedMapValues(s.controls) {
			items = append(items, map[string]any{
				"Arn":         ccString(control["Arn"], ""),
				"Name":        ccString(control["Name"], ""),
				"Description": ccString(control["Description"], ""),
			})
		}
		return map[string]any{
			"Controls":  items,
			"NextToken": "",
		}

	case "ListDomains":
		items := make([]any, 0, len(s.domains))
		for _, domain := range ccSortedMapValues(s.domains) {
			items = append(items, map[string]any{
				"Arn":         ccString(domain["Arn"], ""),
				"Name":        ccString(domain["Name"], ""),
				"Description": ccString(domain["Description"], ""),
			})
		}
		return map[string]any{
			"Domains":   items,
			"NextToken": "",
		}

	case "ListObjectives":
		items := make([]any, 0, len(s.objectives))
		for _, objective := range ccSortedMapValues(s.objectives) {
			items = append(items, map[string]any{
				"Arn":         ccString(objective["Arn"], ""),
				"Name":        ccString(objective["Name"], ""),
				"Description": ccString(objective["Description"], ""),
				"Domains":     ccCloneAny(objective["Domains"]),
			})
		}
		return map[string]any{
			"Objectives": items,
			"NextToken":  "",
		}

	case "ListCommonControls":
		items := make([]any, 0, len(s.commonControls))
		for _, common := range ccSortedMapValues(s.commonControls) {
			items = append(items, map[string]any{
				"Arn":         ccString(common["Arn"], ""),
				"Name":        ccString(common["Name"], ""),
				"Description": ccString(common["Description"], ""),
			})
		}
		return map[string]any{
			"CommonControls": items,
			"NextToken":      "",
		}

	case "ListControlMappings":
		items := make([]any, 0, len(s.controlMappings))
		for _, mapping := range ccSortedMapValues(s.controlMappings) {
			items = append(items, ccCloneMap(mapping))
		}
		return map[string]any{
			"ControlMappings": items,
			"NextToken":       "",
		}
	}

	return map[string]any{}
}

func (s *controlCatalogStore) ensureSeedDataLocked() {
	if len(s.controls) == 0 {
		domain := s.ensureDomainLocked("operations")
		objective := s.ensureObjectiveLocked("access-management", ccString(domain["Arn"], ""))
		control := s.ensureControlLocked("cis-1-1", ccString(objective["Arn"], ""), ccString(domain["Arn"], ""))
		common := s.ensureCommonControlLocked("ccm-iam-1", ccString(control["Arn"], ""))
		s.ensureControlMappingLocked(ccString(control["Arn"], ""), ccString(common["Arn"], ""))
	}
}

func (s *controlCatalogStore) ensureControlByArnLocked(controlArn string) map[string]any {
	if controlArn = strings.TrimSpace(controlArn); controlArn == "" {
		controlArn = ccFirstMapKey(s.controls)
	}
	if control := s.controls[controlArn]; control != nil {
		return control
	}

	domain := s.ensureDomainLocked(fmt.Sprintf("domain-%06d", s.nextDomainIDLocked()))
	objective := s.ensureObjectiveLocked(
		fmt.Sprintf("objective-%06d", s.nextObjectiveIDLocked()),
		ccString(domain["Arn"], ""),
	)
	name := ccLastToken(controlArn)
	if name == "" {
		name = fmt.Sprintf("control-%06d", s.nextControlIDLocked())
	}
	control := s.ensureControlLocked(name, ccString(objective["Arn"], ""), ccString(domain["Arn"], ""))
	control["Arn"] = controlArn
	s.controls[controlArn] = control
	return control
}

func (s *controlCatalogStore) ensureControlLocked(name, objectiveArn, domainArn string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = fmt.Sprintf("control-%06d", s.nextControlIDLocked())
	}
	arn := ccControlARN(key)
	if control := s.controls[arn]; control != nil {
		return control
	}
	if strings.TrimSpace(objectiveArn) == "" {
		objectiveArn = ccObjectiveARN("access-management")
	}
	if strings.TrimSpace(domainArn) == "" {
		domainArn = ccDomainARN("operations")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	control := map[string]any{
		"Arn":              arn,
		"Name":             key,
		"Description":      "Stackyard control " + key,
		"Behavior":         "PREVENTIVE",
		"Severity":         "HIGH",
		"CreateTime":       now,
		"UpdateTime":       now,
		"Domains":          []any{map[string]any{"Arn": domainArn, "Name": ccLastToken(domainArn)}},
		"Objectives":       []any{map[string]any{"Arn": objectiveArn, "Name": ccLastToken(objectiveArn)}},
		"Implementation":   map[string]any{"Type": "AWS_NATIVE", "Guidance": "Enabled by Stackyard seed"},
		"Parameters":       []any{},
		"RegionalSettings": []any{map[string]any{"Region": "us-east-1", "ControlAvailability": "AVAILABLE"}},
	}
	s.controls[arn] = control
	return control
}

func (s *controlCatalogStore) ensureDomainLocked(name string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = fmt.Sprintf("domain-%06d", s.nextDomainIDLocked())
	}
	arn := ccDomainARN(key)
	if domain := s.domains[arn]; domain != nil {
		return domain
	}
	domain := map[string]any{
		"Arn":         arn,
		"Name":        key,
		"Description": "Stackyard domain " + key,
	}
	s.domains[arn] = domain
	return domain
}

func (s *controlCatalogStore) ensureObjectiveLocked(name, domainArn string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = fmt.Sprintf("objective-%06d", s.nextObjectiveIDLocked())
	}
	arn := ccObjectiveARN(key)
	if objective := s.objectives[arn]; objective != nil {
		return objective
	}
	if strings.TrimSpace(domainArn) == "" {
		domainArn = ccDomainARN("operations")
	}
	objective := map[string]any{
		"Arn":         arn,
		"Name":        key,
		"Description": "Stackyard objective " + key,
		"Domains":     []any{map[string]any{"Arn": domainArn, "Name": ccLastToken(domainArn)}},
	}
	s.objectives[arn] = objective
	return objective
}

func (s *controlCatalogStore) ensureCommonControlLocked(name, mappedControlArn string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = fmt.Sprintf("common-%06d", s.nextCommonIDLocked())
	}
	arn := ccCommonControlARN(key)
	if common := s.commonControls[arn]; common != nil {
		return common
	}
	if strings.TrimSpace(mappedControlArn) == "" {
		mappedControlArn = ccFirstMapKey(s.controls)
	}
	common := map[string]any{
		"Arn":         arn,
		"Name":        key,
		"Description": "Stackyard common control " + key,
		"Controls": []any{
			map[string]any{
				"Arn":  mappedControlArn,
				"Name": ccLastToken(mappedControlArn),
			},
		},
	}
	s.commonControls[arn] = common
	return common
}

func (s *controlCatalogStore) ensureControlMappingLocked(controlArn, commonControlArn string) map[string]any {
	if strings.TrimSpace(controlArn) == "" {
		controlArn = ccFirstMapKey(s.controls)
	}
	if strings.TrimSpace(commonControlArn) == "" {
		commonControlArn = ccFirstMapKey(s.commonControls)
	}
	key := controlArn + "|" + commonControlArn
	if mapping := s.controlMappings[key]; mapping != nil {
		return mapping
	}
	mapping := map[string]any{
		"ControlArn":       controlArn,
		"CommonControlArn": commonControlArn,
		"MappingType":      "RELATED",
		"Framework":        "NIST-800-53",
		"Details": map[string]any{
			"Coverage": "PARTIAL",
			"Notes":    "Stackyard seed mapping",
		},
	}
	s.controlMappings[key] = mapping
	return mapping
}

func (s *controlCatalogStore) maybeApplyControlFilterLocked(payload map[string]any, query url.Values) {
	controlArn := ccStringAny(payload, "ControlArn", "")
	if controlArn == "" {
		controlArn = ccStringAny(payload, "controlArn", "")
	}
	if controlArn == "" {
		controlArn = strings.TrimSpace(query.Get("controlArn"))
	}
	if controlArn != "" {
		s.ensureControlByArnLocked(controlArn)
	}
}

func (s *controlCatalogStore) maybeApplyObjectiveFilterLocked(payload map[string]any, query url.Values) {
	objectiveArn := ccStringAny(payload, "ObjectiveArn", "")
	if objectiveArn == "" {
		objectiveArn = ccStringAny(payload, "objectiveArn", "")
	}
	if objectiveArn == "" {
		objectiveArn = strings.TrimSpace(query.Get("objectiveArn"))
	}
	if objectiveArn == "" {
		return
	}
	if objective := s.objectives[objectiveArn]; objective != nil {
		_ = objective
		return
	}
	name := ccLastToken(objectiveArn)
	s.ensureObjectiveLocked(name, ccDomainARN("operations"))["Arn"] = objectiveArn
	if obj := s.objectives[ccObjectiveARN(name)]; obj != nil {
		delete(s.objectives, ccObjectiveARN(name))
		s.objectives[objectiveArn] = obj
	}
}

func (s *controlCatalogStore) maybeApplyDomainFilterLocked(payload map[string]any, query url.Values) {
	domainArn := ccStringAny(payload, "DomainArn", "")
	if domainArn == "" {
		domainArn = ccStringAny(payload, "domainArn", "")
	}
	if domainArn == "" {
		domainArn = strings.TrimSpace(query.Get("domainArn"))
	}
	if domainArn == "" {
		return
	}
	if domain := s.domains[domainArn]; domain != nil {
		_ = domain
		return
	}
	name := ccLastToken(domainArn)
	s.ensureDomainLocked(name)["Arn"] = domainArn
	if dom := s.domains[ccDomainARN(name)]; dom != nil {
		delete(s.domains, ccDomainARN(name))
		s.domains[domainArn] = dom
	}
}

func (s *controlCatalogStore) nextControlIDLocked() int64 {
	id := s.nextControl
	s.nextControl++
	return id
}

func (s *controlCatalogStore) nextObjectiveIDLocked() int64 {
	id := s.nextObjective
	s.nextObjective++
	return id
}

func (s *controlCatalogStore) nextDomainIDLocked() int64 {
	id := s.nextDomain
	s.nextDomain++
	return id
}

func (s *controlCatalogStore) nextCommonIDLocked() int64 {
	id := s.nextCommon
	s.nextCommon++
	return id
}

func ccControlARN(id string) string {
	return "arn:aws:controlcatalog:us-east-1::control/" + strings.TrimSpace(id)
}

func ccDomainARN(id string) string {
	return "arn:aws:controlcatalog:us-east-1::domain/" + strings.TrimSpace(id)
}

func ccObjectiveARN(id string) string {
	return "arn:aws:controlcatalog:us-east-1::objective/" + strings.TrimSpace(id)
}

func ccCommonControlARN(id string) string {
	return "arn:aws:controlcatalog:us-east-1::common-control/" + strings.TrimSpace(id)
}

func ccLastToken(arn string) string {
	trimmed := strings.TrimSpace(arn)
	if trimmed == "" {
		return ""
	}
	idx := strings.LastIndex(trimmed, "/")
	if idx >= 0 && idx+1 < len(trimmed) {
		return trimmed[idx+1:]
	}
	idx = strings.LastIndex(trimmed, ":")
	if idx >= 0 && idx+1 < len(trimmed) {
		return trimmed[idx+1:]
	}
	return trimmed
}

func ccFirstMapKey(in map[string]map[string]any) string {
	if len(in) == 0 {
		return ""
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func ccSortedMapValues(in map[string]map[string]any) []map[string]any {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, in[key])
	}
	return out
}

func ccCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = ccCloneAny(value)
	}
	return out
}

func ccCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return ccCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, ccCloneAny(item))
		}
		return out
	default:
		return typed
	}
}

func ccStringAny(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for existingKey, value := range payload {
		if strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			return ccString(value, def)
		}
	}
	return def
}

func ccString(value any, def string) string {
	switch typed := value.(type) {
	case string:
		if trimmed := strings.TrimSpace(typed); trimmed != "" {
			return trimmed
		}
	case fmt.Stringer:
		if trimmed := strings.TrimSpace(typed.String()); trimmed != "" {
			return trimmed
		}
	}
	return def
}
