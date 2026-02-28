package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type internetMonitorStore struct {
	mu sync.Mutex

	nextMonitor int64
	nextQuery   int64

	monitors       map[string]map[string]any
	healthEvents   map[string]map[string]any
	internetEvents map[string]map[string]any
	queries        map[string]map[string]any
	tags           map[string]map[string]string
}

func newInternetMonitorStore() *internetMonitorStore {
	s := &internetMonitorStore{
		nextMonitor:    2,
		nextQuery:      2,
		monitors:       map[string]map[string]any{},
		healthEvents:   map[string]map[string]any{},
		internetEvents: map[string]map[string]any{},
		queries:        map[string]map[string]any{},
		tags:           map[string]map[string]string{},
	}

	monitor := s.ensureMonitorLocked("stackyard-monitor")
	healthEvent := s.ensureHealthEventLocked("stackyard-monitor", "event-000001")
	internetEvent := s.ensureInternetEventLocked("internet-event-000001")
	s.tags[imMonitorARN(monitor)] = map[string]string{"stackyard": "true"}
	s.tags[imHealthEventARN(healthEvent)] = map[string]string{"stackyard": "true"}
	s.tags[imInternetEventARN(internetEvent)] = map[string]string{"stackyard": "true"}
	return s
}

func (s *internetMonitorStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateMonitor":
		name := imDefaultStringAny(payload, "MonitorName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-monitor-%06d", s.nextMonitorIDLocked())
		}
		monitor := s.ensureMonitorLocked(name)
		s.applyMonitorPayloadLocked(monitor, payload)
		return map[string]any{"Monitor": imCloneMap(monitor)}

	case "DeleteMonitor":
		name := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		if monitor := s.monitors[name]; monitor != nil {
			delete(s.tags, imMonitorARN(monitor))
		}
		delete(s.monitors, name)
		return map[string]any{}

	case "GetHealthEvent":
		monitorName := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		eventID := imDefaultString(pathParams, "EventId", "event-000001")
		return map[string]any{"HealthEvent": imCloneMap(s.ensureHealthEventLocked(monitorName, eventID))}

	case "GetInternetEvent":
		eventID := imDefaultString(pathParams, "EventId", "internet-event-000001")
		return map[string]any{"InternetEvent": imCloneMap(s.ensureInternetEventLocked(eventID))}

	case "GetMonitor":
		name := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		return map[string]any{"Monitor": imCloneMap(s.ensureMonitorLocked(name))}

	case "GetQueryResults":
		monitorName := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		queryID := imDefaultString(pathParams, "QueryId", "query-000001")
		s.ensureQueryLocked(monitorName, queryID)
		return map[string]any{
			"Fields": []any{
				map[string]any{"Name": "ClientLocation", "Type": "STRING"},
				map[string]any{"Name": "RoundTripTime", "Type": "DOUBLE"},
			},
			"Data": []any{
				map[string]any{"ClientLocation": "US", "RoundTripTime": 21.4},
				map[string]any{"ClientLocation": "DE", "RoundTripTime": 31.7},
			},
		}

	case "GetQueryStatus":
		monitorName := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		queryID := imDefaultString(pathParams, "QueryId", "query-000001")
		query := s.ensureQueryLocked(monitorName, queryID)
		return map[string]any{
			"Status":  imDefaultStringAny(query, "Status", "SUCCEEDED"),
			"QueryId": queryID,
		}

	case "ListHealthEvents":
		monitorName := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		keys := make([]string, 0, len(s.healthEvents))
		for k := range s.healthEvents {
			if strings.HasPrefix(k, monitorName+":") {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, key := range keys {
			out = append(out, imCloneMap(s.healthEvents[key]))
		}
		if len(out) == 0 {
			out = append(out, imCloneMap(s.ensureHealthEventLocked(monitorName, "event-000001")))
		}
		return map[string]any{"HealthEvents": out, "NextToken": ""}

	case "ListInternetEvents":
		keys := make([]string, 0, len(s.internetEvents))
		for key := range s.internetEvents {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, key := range keys {
			out = append(out, imCloneMap(s.internetEvents[key]))
		}
		return map[string]any{"InternetEvents": out, "NextToken": ""}

	case "ListMonitors":
		names := make([]string, 0, len(s.monitors))
		for name := range s.monitors {
			names = append(names, name)
		}
		sort.Strings(names)
		out := make([]any, 0, len(names))
		for _, name := range names {
			out = append(out, imCloneMap(s.monitors[name]))
		}
		return map[string]any{"Monitors": out, "NextToken": ""}

	case "ListTagsForResource":
		resourceARN := imDefaultString(pathParams, "ResourceArn", imMonitorARNByName("stackyard-monitor"))
		return map[string]any{"Tags": imCloneStringMap(s.tags[resourceARN])}

	case "StartQuery":
		monitorName := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		queryID := fmt.Sprintf("query-%06d", s.nextQueryIDLocked())
		query := s.ensureQueryLocked(monitorName, queryID)
		query["Status"] = "SUCCEEDED"
		return map[string]any{"QueryId": queryID, "Status": "SUCCEEDED"}

	case "StopQuery":
		monitorName := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		queryID := imDefaultString(pathParams, "QueryId", "query-000001")
		query := s.ensureQueryLocked(monitorName, queryID)
		query["Status"] = "CANCELLED"
		return map[string]any{"QueryId": queryID, "Status": "CANCELLED"}

	case "TagResource":
		resourceARN := imDefaultString(pathParams, "ResourceArn", imMonitorARNByName("stackyard-monitor"))
		tags := imMapString(payload, "Tags")
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range tags {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := imDefaultString(pathParams, "ResourceArn", imMonitorARNByName("stackyard-monitor"))
		tagKeys := imStringSlice(payload, "TagKeys")
		for _, key := range tagKeys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}

	case "UpdateMonitor":
		name := imDefaultString(pathParams, "MonitorName", "stackyard-monitor")
		monitor := s.ensureMonitorLocked(name)
		s.applyMonitorPayloadLocked(monitor, payload)
		return map[string]any{"Monitor": imCloneMap(monitor)}
	}

	return map[string]any{}
}

func (s *internetMonitorStore) ensureMonitorLocked(name string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-monitor"
	}
	if monitor := s.monitors[key]; monitor != nil {
		return monitor
	}
	monitor := map[string]any{
		"MonitorName": key,
		"MonitorArn":  imMonitorARNByName(key),
		"Status":      "ACTIVE",
		"CreatedAt":   time.Now().UTC(),
	}
	s.monitors[key] = monitor
	return monitor
}

func (s *internetMonitorStore) ensureHealthEventLocked(monitorName, eventID string) map[string]any {
	monitorKey := strings.TrimSpace(monitorName)
	if monitorKey == "" {
		monitorKey = "stackyard-monitor"
	}
	eventKey := strings.TrimSpace(eventID)
	if eventKey == "" {
		eventKey = "event-000001"
	}
	key := monitorKey + ":" + eventKey
	if event := s.healthEvents[key]; event != nil {
		return event
	}
	event := map[string]any{
		"EventId":     eventKey,
		"MonitorName": monitorKey,
		"EventArn":    imHealthEventARNByParts(monitorKey, eventKey),
		"Status":      "RESOLVED",
		"StartedAt":   time.Now().UTC().Add(-10 * time.Minute),
		"EndedAt":     time.Now().UTC().Add(-5 * time.Minute),
	}
	s.healthEvents[key] = event
	return event
}

func (s *internetMonitorStore) ensureInternetEventLocked(eventID string) map[string]any {
	key := strings.TrimSpace(eventID)
	if key == "" {
		key = "internet-event-000001"
	}
	if event := s.internetEvents[key]; event != nil {
		return event
	}
	event := map[string]any{
		"EventId":   key,
		"EventArn":  imInternetEventARNByID(key),
		"Status":    "RESOLVED",
		"StartedAt": time.Now().UTC().Add(-30 * time.Minute),
		"EndedAt":   time.Now().UTC().Add(-20 * time.Minute),
	}
	s.internetEvents[key] = event
	return event
}

func (s *internetMonitorStore) ensureQueryLocked(monitorName, queryID string) map[string]any {
	monitorKey := strings.TrimSpace(monitorName)
	if monitorKey == "" {
		monitorKey = "stackyard-monitor"
	}
	queryKey := strings.TrimSpace(queryID)
	if queryKey == "" {
		queryKey = "query-000001"
	}
	key := monitorKey + ":" + queryKey
	if query := s.queries[key]; query != nil {
		return query
	}
	query := map[string]any{
		"MonitorName": monitorKey,
		"QueryId":     queryKey,
		"Status":      "SUCCEEDED",
		"CreatedAt":   time.Now().UTC(),
	}
	s.queries[key] = query
	return query
}

func (s *internetMonitorStore) applyMonitorPayloadLocked(monitor map[string]any, payload map[string]any) {
	if name := imDefaultStringAny(payload, "MonitorName", ""); name != "" {
		monitor["MonitorName"] = name
		monitor["MonitorArn"] = imMonitorARNByName(name)
	}
	for key, value := range payload {
		monitor[key] = value
	}
}

func (s *internetMonitorStore) nextMonitorIDLocked() int64 {
	id := s.nextMonitor
	s.nextMonitor++
	return id
}

func (s *internetMonitorStore) nextQueryIDLocked() int64 {
	id := s.nextQuery
	s.nextQuery++
	return id
}

func imMonitorARN(monitor map[string]any) string {
	return imMonitorARNByName(imDefaultStringAny(monitor, "MonitorName", "stackyard-monitor"))
}

func imMonitorARNByName(name string) string {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-monitor"
	}
	if strings.HasPrefix(key, "arn:") {
		return key
	}
	return fmt.Sprintf("arn:aws:internetmonitor:us-east-1:123456789012:monitor/%s", key)
}

func imHealthEventARN(event map[string]any) string {
	monitor := imDefaultStringAny(event, "MonitorName", "stackyard-monitor")
	eventID := imDefaultStringAny(event, "EventId", "event-000001")
	return imHealthEventARNByParts(monitor, eventID)
}

func imHealthEventARNByParts(monitorName, eventID string) string {
	return fmt.Sprintf("arn:aws:internetmonitor:us-east-1:123456789012:monitor/%s/health-event/%s", strings.TrimSpace(monitorName), strings.TrimSpace(eventID))
}

func imInternetEventARN(event map[string]any) string {
	return imInternetEventARNByID(imDefaultStringAny(event, "EventId", "internet-event-000001"))
}

func imInternetEventARNByID(eventID string) string {
	key := strings.TrimSpace(eventID)
	if key == "" {
		key = "internet-event-000001"
	}
	return fmt.Sprintf("arn:aws:internetmonitor:us-east-1:123456789012:internet-event/%s", key)
}

func imDefaultString(values map[string]string, key, fallback string) string {
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

func imDefaultStringAny(values map[string]any, key, fallback string) string {
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

func imMapString(values map[string]any, key string) map[string]string {
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

func imStringSlice(values map[string]any, key string) []string {
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

func imCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func imCloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
