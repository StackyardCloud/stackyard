package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type applicationSignalsStore struct {
	mu sync.Mutex

	nextID int64

	slos             map[string]map[string]any
	services         map[string]map[string]any
	groupingConfig   map[string]any
	exclusionWindows map[string][]map[string]any
	tags             map[string]map[string]string
	discoveryStarted bool
}

func newApplicationSignalsStore() *applicationSignalsStore {
	s := &applicationSignalsStore{
		nextID:           2,
		slos:             map[string]map[string]any{},
		services:         map[string]map[string]any{},
		groupingConfig:   map[string]any{},
		exclusionWindows: map[string][]map[string]any{},
		tags:             map[string]map[string]string{},
	}

	service := s.ensureServiceLocked("stackyard-service")
	slo := s.ensureSLOLocked("stackyard-slo")
	s.exclusionWindows[appSignalsSLOID(slo)] = []map[string]any{
		{
			"Name":      "stackyard-maintenance-window",
			"StartTime": time.Now().UTC().Add(-1 * time.Hour),
			"EndTime":   time.Now().UTC().Add(1 * time.Hour),
		},
	}
	s.tags[appSignalsServiceARN(service)] = map[string]string{"stackyard": "true"}
	s.tags[appSignalsSLOARN(slo)] = map[string]string{"stackyard": "true"}
	return s
}

func (s *applicationSignalsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "BatchGetServiceLevelObjectiveBudgetReport":
		reports := make([]any, 0, len(s.slos))
		sloIDs := s.sortedSLOIDsLocked()
		for _, id := range sloIDs {
			slo := s.ensureSLOLocked(id)
			reports = append(reports, map[string]any{
				"SloId":            id,
				"ServiceName":      appSignalsString(slo, "ServiceName"),
				"BudgetStatus":     "HEALTHY",
				"TimeToExhaustion": "PT168H",
			})
		}
		return map[string]any{
			"ServiceLevelObjectiveBudgetReports": reports,
			"Errors":                             []any{},
		}

	case "BatchUpdateExclusionWindows":
		sloID := appSignalsDefaultString(pathParams, "Id", appSignalsDefaultStringAny(payload, "SloId", "stackyard-slo"))
		s.ensureSLOLocked(sloID)
		s.exclusionWindows[sloID] = []map[string]any{
			{
				"Name":      "stackyard-updated-window",
				"StartTime": time.Now().UTC().Add(-30 * time.Minute),
				"EndTime":   time.Now().UTC().Add(30 * time.Minute),
			},
		}
		return map[string]any{"Errors": []any{}}

	case "CreateServiceLevelObjective":
		id := appSignalsDefaultStringAny(payload, "Id", "")
		if id == "" {
			id = fmt.Sprintf("stackyard-slo-%06d", s.nextLocked())
		}
		slo := s.ensureSLOLocked(id)
		s.applySLOPayloadLocked(slo, payload)
		return map[string]any{"ServiceLevelObjective": appSignalsCloneMap(slo)}

	case "DeleteGroupingConfiguration":
		s.groupingConfig = map[string]any{}
		return map[string]any{}

	case "DeleteServiceLevelObjective":
		id := appSignalsDefaultString(pathParams, "Id", "stackyard-slo")
		delete(s.slos, id)
		delete(s.exclusionWindows, id)
		delete(s.tags, appSignalsSLOARNByID(id))
		return map[string]any{}

	case "GetService":
		service := s.ensureServiceLocked("stackyard-service")
		return map[string]any{"Service": appSignalsCloneMap(service)}

	case "GetServiceLevelObjective":
		id := appSignalsDefaultString(pathParams, "Id", "stackyard-slo")
		return map[string]any{"ServiceLevelObjective": appSignalsCloneMap(s.ensureSLOLocked(id))}

	case "ListAuditFindings":
		return map[string]any{
			"AuditFindings": []any{
				map[string]any{
					"FindingType": "BURN_RATE",
					"Severity":    "LOW",
					"Message":     "Stackyard synthetic audit finding",
					"Timestamp":   time.Now().UTC(),
				},
			},
			"NextToken": "",
		}

	case "ListEntityEvents":
		return map[string]any{
			"EntityEvents": []any{
				map[string]any{
					"EntityType": "SERVICE",
					"EntityName": "stackyard-service",
					"EventType":  "DEPLOYMENT",
					"Timestamp":  time.Now().UTC(),
				},
			},
			"NextToken": "",
		}

	case "ListGroupingAttributeDefinitions":
		return map[string]any{
			"GroupingAttributeDefinitions": []any{
				map[string]any{
					"Name":        "ServiceName",
					"DisplayName": "Service Name",
				},
			},
			"NextToken": "",
		}

	case "ListServiceDependencies":
		return map[string]any{
			"ServiceDependencies": []any{
				map[string]any{
					"SourceService": "stackyard-service",
					"TargetService": "stackyard-dependency",
				},
			},
			"NextToken": "",
		}

	case "ListServiceDependents":
		return map[string]any{
			"ServiceDependents": []any{
				map[string]any{
					"SourceService": "stackyard-dependent",
					"TargetService": "stackyard-service",
				},
			},
			"NextToken": "",
		}

	case "ListServiceLevelObjectiveExclusionWindows":
		id := appSignalsDefaultString(pathParams, "Id", "stackyard-slo")
		windows := s.exclusionWindows[id]
		out := make([]any, 0, len(windows))
		for _, window := range windows {
			out = append(out, appSignalsCloneMap(window))
		}
		return map[string]any{"ExclusionWindows": out, "NextToken": ""}

	case "ListServiceLevelObjectives":
		ids := s.sortedSLOIDsLocked()
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			out = append(out, appSignalsCloneMap(s.slos[id]))
		}
		return map[string]any{"ServiceLevelObjectives": out, "NextToken": ""}

	case "ListServiceOperations":
		return map[string]any{
			"ServiceOperations": []any{
				map[string]any{
					"Name":        "GET /health",
					"ServiceName": "stackyard-service",
				},
			},
			"NextToken": "",
		}

	case "ListServices":
		names := make([]string, 0, len(s.services))
		for name := range s.services {
			names = append(names, name)
		}
		sort.Strings(names)
		out := make([]any, 0, len(names))
		for _, name := range names {
			out = append(out, appSignalsCloneMap(s.services[name]))
		}
		return map[string]any{"ServiceSummaries": out, "NextToken": ""}

	case "ListServiceStates":
		return map[string]any{
			"ServiceStates": []any{
				map[string]any{
					"ServiceName": "stackyard-service",
					"State":       "HEALTHY",
					"Timestamp":   time.Now().UTC(),
				},
			},
			"NextToken": "",
		}

	case "ListTagsForResource":
		resourceARN := appSignalsResourceARNFromQueryOrPayload(query, payload)
		return map[string]any{"Tags": appSignalsCloneStringMap(s.tags[resourceARN])}

	case "PutGroupingConfiguration":
		config := appSignalsMapAny(payload, "GroupingConfiguration")
		if len(config) == 0 {
			config = appSignalsCloneMap(payload)
		}
		s.groupingConfig = config
		return map[string]any{"GroupingConfiguration": appSignalsCloneMap(s.groupingConfig)}

	case "StartDiscovery":
		s.discoveryStarted = true
		return map[string]any{"State": "STARTED"}

	case "TagResource":
		resourceARN := appSignalsResourceARNFromQueryOrPayload(query, payload)
		newTags := appSignalsMapString(payload, "Tags")
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range newTags {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := appSignalsResourceARNFromQueryOrPayload(query, payload)
		tagKeys := appSignalsStringSlice(payload, "TagKeys")
		for _, k := range tagKeys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], k)
			}
		}
		return map[string]any{}

	case "UpdateServiceLevelObjective":
		id := appSignalsDefaultString(pathParams, "Id", "stackyard-slo")
		slo := s.ensureSLOLocked(id)
		s.applySLOPayloadLocked(slo, payload)
		return map[string]any{"ServiceLevelObjective": appSignalsCloneMap(slo)}
	}

	return map[string]any{}
}

func (s *applicationSignalsStore) ensureSLOLocked(id string) map[string]any {
	key := strings.TrimSpace(id)
	if key == "" {
		key = "stackyard-slo"
	}
	if slo := s.slos[key]; slo != nil {
		return slo
	}
	slo := map[string]any{
		"Id":          key,
		"Arn":         appSignalsSLOARNByID(key),
		"Name":        key,
		"ServiceName": "stackyard-service",
		"CreatedTime": time.Now().UTC(),
		"Goal": map[string]any{
			"Interval":  "MONTHLY",
			"Threshold": 99.0,
		},
	}
	s.slos[key] = slo
	return slo
}

func (s *applicationSignalsStore) ensureServiceLocked(name string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-service"
	}
	if svc := s.services[key]; svc != nil {
		return svc
	}
	svc := map[string]any{
		"Name":        key,
		"Arn":         appSignalsServiceARNByName(key),
		"Environment": "stackyard",
	}
	s.services[key] = svc
	return svc
}

func (s *applicationSignalsStore) sortedSLOIDsLocked() []string {
	ids := make([]string, 0, len(s.slos))
	for id := range s.slos {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (s *applicationSignalsStore) applySLOPayloadLocked(slo map[string]any, payload map[string]any) {
	if name := appSignalsString(payload, "Name"); name != "" {
		slo["Name"] = name
	}
	if serviceName := appSignalsString(payload, "ServiceName"); serviceName != "" {
		slo["ServiceName"] = serviceName
		s.ensureServiceLocked(serviceName)
	}
	if goal := appSignalsMapAny(payload, "Goal"); len(goal) != 0 {
		slo["Goal"] = goal
	}
	if sliConfig := appSignalsMapAny(payload, "SliConfig"); len(sliConfig) != 0 {
		slo["SliConfig"] = sliConfig
	}
}

func (s *applicationSignalsStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func appSignalsSLOID(slo map[string]any) string {
	if v := appSignalsString(slo, "Id"); v != "" {
		return v
	}
	return "stackyard-slo"
}

func appSignalsSLOARN(slo map[string]any) string {
	if v := appSignalsString(slo, "Arn"); v != "" {
		return v
	}
	return appSignalsSLOARNByID(appSignalsSLOID(slo))
}

func appSignalsSLOARNByID(id string) string {
	key := strings.TrimSpace(id)
	if key == "" {
		key = "stackyard-slo"
	}
	if strings.HasPrefix(key, "arn:") {
		return key
	}
	return fmt.Sprintf("arn:aws:application-signals:us-east-1:123456789012:slo/%s", key)
}

func appSignalsServiceARN(service map[string]any) string {
	if v := appSignalsString(service, "Arn"); v != "" {
		return v
	}
	return appSignalsServiceARNByName(appSignalsString(service, "Name"))
}

func appSignalsServiceARNByName(name string) string {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-service"
	}
	if strings.HasPrefix(key, "arn:") {
		return key
	}
	return fmt.Sprintf("arn:aws:application-signals:us-east-1:123456789012:service/%s", key)
}

func appSignalsResourceARNFromQueryOrPayload(query url.Values, payload map[string]any) string {
	if query != nil {
		if value := strings.TrimSpace(query.Get("ResourceArn")); value != "" {
			return value
		}
	}
	if value := appSignalsString(payload, "ResourceArn"); value != "" {
		return value
	}
	return appSignalsServiceARNByName("stackyard-service")
}

func appSignalsString(payload map[string]any, key string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return ""
}

func appSignalsDefaultString(payload map[string]string, key, fallback string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func appSignalsDefaultStringAny(payload map[string]any, key, fallback string) string {
	if value := appSignalsString(payload, key); value != "" {
		return value
	}
	return fallback
}

func appSignalsMapAny(payload map[string]any, key string) map[string]any {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if typed, ok := v.(map[string]any); ok {
			return appSignalsCloneMap(typed)
		}
	}
	return map[string]any{}
}

func appSignalsMapString(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if typed, ok := v.(map[string]string); ok {
			return appSignalsCloneStringMap(typed)
		}
		if typed, ok := v.(map[string]any); ok {
			out := map[string]string{}
			for tk, tv := range typed {
				out[tk] = strings.TrimSpace(fmt.Sprintf("%v", tv))
			}
			return out
		}
	}
	return map[string]string{}
}

func appSignalsStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch typed := v.(type) {
		case []string:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if trimmed := strings.TrimSpace(item); trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if trimmed := strings.TrimSpace(fmt.Sprintf("%v", item)); trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		}
	}
	return []string{}
}

func appSignalsCloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func appSignalsCloneStringMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
