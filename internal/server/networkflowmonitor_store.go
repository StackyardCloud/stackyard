package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type networkFlowMonitorStore struct {
	mu sync.Mutex

	nextMonitor int64
	nextScope   int64
	nextQuery   int64

	monitors map[string]map[string]any
	scopes   map[string]map[string]any
	queries  map[string]map[string]any
	tags     map[string]map[string]string
}

func newNetworkFlowMonitorStore() *networkFlowMonitorStore {
	s := &networkFlowMonitorStore{
		nextMonitor: 2,
		nextScope:   2,
		nextQuery:   2,
		monitors:    map[string]map[string]any{},
		scopes:      map[string]map[string]any{},
		queries:     map[string]map[string]any{},
		tags:        map[string]map[string]string{},
	}

	monitor := s.ensureMonitorLocked("stackyard-monitor")
	scope := s.ensureScopeLocked("scope-00000000000000001")
	s.tags[nfmMonitorARN(monitor)] = map[string]string{"stackyard": "true"}
	s.tags[nfmScopeARN(scope)] = map[string]string{"stackyard": "true"}
	return s
}

func (s *networkFlowMonitorStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateMonitor":
		name := nfmDefaultStringAny(payload, "MonitorName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-monitor-%06d", s.nextMonitorIDLocked())
		}
		monitor := s.ensureMonitorLocked(name)
		s.applyMonitorPayloadLocked(monitor, payload)
		return map[string]any{
			"Monitor": nfmCloneMap(monitor),
		}

	case "CreateScope":
		scopeID := nfmDefaultStringAny(payload, "ScopeId", "")
		if scopeID == "" {
			scopeID = fmt.Sprintf("scope-%017d", s.nextScopeIDLocked())
		}
		scope := s.ensureScopeLocked(scopeID)
		s.applyScopePayloadLocked(scope, payload)
		return map[string]any{
			"Scope": nfmCloneMap(scope),
		}

	case "DeleteMonitor":
		name := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		if monitor := s.monitors[name]; monitor != nil {
			delete(s.tags, nfmMonitorARN(monitor))
		}
		delete(s.monitors, name)
		return map[string]any{}

	case "DeleteScope":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		if scope := s.scopes[scopeID]; scope != nil {
			delete(s.tags, nfmScopeARN(scope))
		}
		delete(s.scopes, scopeID)
		return map[string]any{}

	case "GetMonitor":
		name := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		return map[string]any{
			"Monitor": nfmCloneMap(s.ensureMonitorLocked(name)),
		}

	case "GetScope":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		return map[string]any{
			"Scope": nfmCloneMap(s.ensureScopeLocked(scopeID)),
		}

	case "ListMonitors":
		names := make([]string, 0, len(s.monitors))
		for name := range s.monitors {
			names = append(names, name)
		}
		sort.Strings(names)
		out := make([]any, 0, len(names))
		for _, name := range names {
			out = append(out, nfmCloneMap(s.monitors[name]))
		}
		return map[string]any{
			"Monitors":  out,
			"NextToken": "",
		}

	case "ListScopes":
		scopeIDs := make([]string, 0, len(s.scopes))
		for scopeID := range s.scopes {
			scopeIDs = append(scopeIDs, scopeID)
		}
		sort.Strings(scopeIDs)
		out := make([]any, 0, len(scopeIDs))
		for _, scopeID := range scopeIDs {
			out = append(out, nfmCloneMap(s.scopes[scopeID]))
		}
		return map[string]any{
			"Scopes":    out,
			"NextToken": "",
		}

	case "StartQueryMonitorTopContributors":
		monitorName := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		queryID := s.ensureQueryLocked(action, monitorName, "")
		return map[string]any{
			"QueryId": queryID,
			"Status":  "SUCCEEDED",
		}

	case "StartQueryWorkloadInsightsTopContributors", "StartQueryWorkloadInsightsTopContributorsData":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		queryID := s.ensureQueryLocked(action, "", scopeID)
		return map[string]any{
			"QueryId": queryID,
			"Status":  "SUCCEEDED",
		}

	case "GetQueryStatusMonitorTopContributors":
		monitorName := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		query := s.ensureExistingQueryLocked(queryID, monitorName, "", action)
		return map[string]any{
			"QueryId": queryID,
			"Status":  nfmDefaultStringAny(query, "Status", "SUCCEEDED"),
		}

	case "GetQueryStatusWorkloadInsightsTopContributors", "GetQueryStatusWorkloadInsightsTopContributorsData":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		query := s.ensureExistingQueryLocked(queryID, "", scopeID, action)
		return map[string]any{
			"QueryId": queryID,
			"Status":  nfmDefaultStringAny(query, "Status", "SUCCEEDED"),
		}

	case "GetQueryResultsMonitorTopContributors":
		monitorName := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		s.ensureExistingQueryLocked(queryID, monitorName, "", action)
		return map[string]any{
			"QueryId": queryID,
			"Rows": []any{
				map[string]any{
					"MonitorName":   monitorName,
					"LocalIP":       "10.0.0.10",
					"RemoteIP":      "10.0.1.20",
					"BytesIn":       1024,
					"BytesOut":      2048,
					"PacketLossPct": 0.01,
				},
			},
		}

	case "GetQueryResultsWorkloadInsightsTopContributors":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		s.ensureExistingQueryLocked(queryID, "", scopeID, action)
		return map[string]any{
			"QueryId": queryID,
			"Rows": []any{
				map[string]any{
					"ScopeId":        scopeID,
					"TopContributor": "stackyard-service",
					"Packets":        128,
					"LatencyP50Ms":   2.1,
				},
			},
		}

	case "GetQueryResultsWorkloadInsightsTopContributorsData":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		s.ensureExistingQueryLocked(queryID, "", scopeID, action)
		return map[string]any{
			"QueryId": queryID,
			"Rows": []any{
				map[string]any{
					"ScopeId":      scopeID,
					"Timestamp":    time.Now().UTC(),
					"PacketsIn":    64,
					"PacketsOut":   80,
					"LatencyP95Ms": 3.8,
				},
			},
		}

	case "StopQueryMonitorTopContributors":
		monitorName := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		query := s.ensureExistingQueryLocked(queryID, monitorName, "", action)
		query["Status"] = "CANCELLED"
		return map[string]any{
			"QueryId": queryID,
			"Status":  "CANCELLED",
		}

	case "StopQueryWorkloadInsightsTopContributors", "StopQueryWorkloadInsightsTopContributorsData":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		queryID := nfmDefaultString(pathParams, "queryId", "query-000001")
		query := s.ensureExistingQueryLocked(queryID, "", scopeID, action)
		query["Status"] = "CANCELLED"
		return map[string]any{
			"QueryId": queryID,
			"Status":  "CANCELLED",
		}

	case "TagResource":
		resourceARN := nfmDefaultString(pathParams, "resourceArn", nfmMonitorARNByName("stackyard-monitor"))
		tags := nfmMapString(payload, "Tags")
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range tags {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := nfmDefaultString(pathParams, "resourceArn", nfmMonitorARNByName("stackyard-monitor"))
		tagKeys := nfmStringSlice(payload, "TagKeys")
		for _, key := range tagKeys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := nfmDefaultString(pathParams, "resourceArn", nfmMonitorARNByName("stackyard-monitor"))
		return map[string]any{
			"Tags": nfmCloneStringMap(s.tags[resourceARN]),
		}

	case "UpdateMonitor":
		name := nfmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		monitor := s.ensureMonitorLocked(name)
		s.applyMonitorPayloadLocked(monitor, payload)
		return map[string]any{
			"Monitor": nfmCloneMap(monitor),
		}

	case "UpdateScope":
		scopeID := nfmDefaultString(pathParams, "scopeId", "scope-00000000000000001")
		scope := s.ensureScopeLocked(scopeID)
		s.applyScopePayloadLocked(scope, payload)
		return map[string]any{
			"Scope": nfmCloneMap(scope),
		}
	}

	return map[string]any{}
}

func (s *networkFlowMonitorStore) ensureMonitorLocked(name string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-monitor"
	}
	if monitor := s.monitors[key]; monitor != nil {
		return monitor
	}
	monitor := map[string]any{
		"MonitorName": key,
		"MonitorArn":  nfmMonitorARNByName(key),
		"Status":      "ACTIVE",
		"CreatedAt":   time.Now().UTC(),
		"LocalResources": []any{
			map[string]any{
				"Type":       "SUBNET",
				"Identifier": "subnet-0123456789abcdef0",
			},
		},
		"RemoteResources": []any{
			map[string]any{
				"Type":       "AWS_SERVICE",
				"Identifier": "S3",
			},
		},
	}
	s.monitors[key] = monitor
	return monitor
}

func (s *networkFlowMonitorStore) ensureScopeLocked(scopeID string) map[string]any {
	key := strings.TrimSpace(scopeID)
	if key == "" {
		key = "scope-00000000000000001"
	}
	if scope := s.scopes[key]; scope != nil {
		return scope
	}
	scope := map[string]any{
		"ScopeId":    key,
		"ScopeArn":   nfmScopeARNByID(key),
		"Status":     "ACTIVE",
		"CreateTime": time.Now().UTC(),
	}
	s.scopes[key] = scope
	return scope
}

func (s *networkFlowMonitorStore) ensureQueryLocked(action, monitorName, scopeID string) string {
	id := fmt.Sprintf("query-%06d", s.nextQueryIDLocked())
	query := map[string]any{
		"QueryId":     id,
		"Action":      action,
		"MonitorName": strings.TrimSpace(monitorName),
		"ScopeId":     strings.TrimSpace(scopeID),
		"Status":      "SUCCEEDED",
		"CreatedAt":   time.Now().UTC(),
	}
	s.queries[id] = query
	return id
}

func (s *networkFlowMonitorStore) ensureExistingQueryLocked(queryID, monitorName, scopeID, action string) map[string]any {
	key := strings.TrimSpace(queryID)
	if key == "" {
		key = "query-000001"
	}
	if query := s.queries[key]; query != nil {
		return query
	}
	query := map[string]any{
		"QueryId":     key,
		"Action":      action,
		"MonitorName": strings.TrimSpace(monitorName),
		"ScopeId":     strings.TrimSpace(scopeID),
		"Status":      "SUCCEEDED",
		"CreatedAt":   time.Now().UTC(),
	}
	s.queries[key] = query
	return query
}

func (s *networkFlowMonitorStore) applyMonitorPayloadLocked(monitor map[string]any, payload map[string]any) {
	if value := nfmDefaultStringAny(payload, "MonitorName", ""); value != "" {
		monitor["MonitorName"] = value
		monitor["MonitorArn"] = nfmMonitorARNByName(value)
	}
	for key, value := range payload {
		monitor[key] = value
	}
}

func (s *networkFlowMonitorStore) applyScopePayloadLocked(scope map[string]any, payload map[string]any) {
	if value := nfmDefaultStringAny(payload, "ScopeId", ""); value != "" {
		scope["ScopeId"] = value
		scope["ScopeArn"] = nfmScopeARNByID(value)
	}
	for key, value := range payload {
		scope[key] = value
	}
}

func (s *networkFlowMonitorStore) nextMonitorIDLocked() int64 {
	id := s.nextMonitor
	s.nextMonitor++
	return id
}

func (s *networkFlowMonitorStore) nextScopeIDLocked() int64 {
	id := s.nextScope
	s.nextScope++
	return id
}

func (s *networkFlowMonitorStore) nextQueryIDLocked() int64 {
	id := s.nextQuery
	s.nextQuery++
	return id
}

func nfmMonitorARN(monitor map[string]any) string {
	return nfmMonitorARNByName(nfmDefaultStringAny(monitor, "MonitorName", "stackyard-monitor"))
}

func nfmMonitorARNByName(name string) string {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-monitor"
	}
	if strings.HasPrefix(key, "arn:") {
		return key
	}
	return fmt.Sprintf("arn:aws:networkflowmonitor:us-east-1:123456789012:monitor/%s", key)
}

func nfmScopeARN(scope map[string]any) string {
	return nfmScopeARNByID(nfmDefaultStringAny(scope, "ScopeId", "scope-00000000000000001"))
}

func nfmScopeARNByID(scopeID string) string {
	key := strings.TrimSpace(scopeID)
	if key == "" {
		key = "scope-00000000000000001"
	}
	if strings.HasPrefix(key, "arn:") {
		return key
	}
	return fmt.Sprintf("arn:aws:networkflowmonitor:us-east-1:123456789012:scope/%s", key)
}

func nfmDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			trimmed := strings.TrimSpace(v)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func nfmDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(k, key) {
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", v))
			if trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return fallback
}

func nfmMapString(values map[string]any, key string) map[string]string {
	for k, v := range values {
		if !strings.EqualFold(k, key) {
			continue
		}
		typed, ok := v.(map[string]any)
		if !ok {
			break
		}
		out := map[string]string{}
		for tk, tv := range typed {
			out[tk] = strings.TrimSpace(fmt.Sprintf("%v", tv))
		}
		return out
	}
	return map[string]string{}
}

func nfmStringSlice(values map[string]any, key string) []string {
	for k, v := range values {
		if !strings.EqualFold(k, key) {
			continue
		}
		typed, ok := v.([]any)
		if !ok {
			break
		}
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	return nil
}

func nfmCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func nfmCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
