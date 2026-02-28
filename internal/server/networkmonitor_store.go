package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type networkMonitorStore struct {
	mu sync.Mutex

	nextMonitor int64
	nextProbe   int64

	monitors map[string]map[string]any
	probes   map[string]map[string]any
	tags     map[string]map[string]string
}

func newNetworkMonitorStore() *networkMonitorStore {
	s := &networkMonitorStore{
		nextMonitor: 2,
		nextProbe:   2,
		monitors:    map[string]map[string]any{},
		probes:      map[string]map[string]any{},
		tags:        map[string]map[string]string{},
	}

	monitor := s.ensureMonitorLocked("stackyard-monitor")
	probe := s.ensureProbeLocked("stackyard-monitor", "probe-000001")
	s.tags[nmMonitorARN(monitor)] = map[string]string{"stackyard": "true"}
	s.tags[nmProbeARN(probe)] = map[string]string{"stackyard": "true"}
	return s
}

func (s *networkMonitorStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateMonitor":
		name := nmDefaultStringAny(payload, "monitorName", "")
		if name == "" {
			name = fmt.Sprintf("stackyard-monitor-%06d", s.nextMonitorIDLocked())
		}
		monitor := s.ensureMonitorLocked(name)
		s.applyMonitorPayloadLocked(monitor, payload)
		s.ensureTagsForARNLocked(nmMonitorARN(monitor))
		return map[string]any{"Monitor": nmCloneMap(monitor)}

	case "DeleteMonitor":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		if monitor := s.monitors[monitorName]; monitor != nil {
			delete(s.tags, nmMonitorARN(monitor))
		}
		for key, probe := range s.probes {
			if nmDefaultStringAny(probe, "MonitorName", "") == monitorName {
				delete(s.tags, nmProbeARN(probe))
				delete(s.probes, key)
			}
		}
		delete(s.monitors, monitorName)
		return map[string]any{}

	case "GetMonitor":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		return map[string]any{"Monitor": nmCloneMap(s.ensureMonitorLocked(monitorName))}

	case "ListMonitors":
		names := make([]string, 0, len(s.monitors))
		for name := range s.monitors {
			names = append(names, name)
		}
		sort.Strings(names)
		monitors := make([]any, 0, len(names))
		for _, name := range names {
			monitors = append(monitors, nmCloneMap(s.monitors[name]))
		}
		return map[string]any{"Monitors": monitors, "NextToken": ""}

	case "UpdateMonitor":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		monitor := s.ensureMonitorLocked(monitorName)
		s.applyMonitorPayloadLocked(monitor, payload)
		s.ensureTagsForARNLocked(nmMonitorARN(monitor))
		return map[string]any{"Monitor": nmCloneMap(monitor)}

	case "CreateProbe":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		probeID := nmDefaultStringAny(payload, "probeId", "")
		if probeID == "" {
			probeID = fmt.Sprintf("probe-%06d", s.nextProbeIDLocked())
		}
		probe := s.ensureProbeLocked(monitorName, probeID)
		s.applyProbePayloadLocked(probe, payload)
		s.ensureTagsForARNLocked(nmProbeARN(probe))
		return map[string]any{"Probe": nmCloneMap(probe)}

	case "DeleteProbe":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		probeID := nmDefaultString(pathParams, "probeId", "probe-000001")
		if probe := s.probes[nmProbeKey(monitorName, probeID)]; probe != nil {
			delete(s.tags, nmProbeARN(probe))
		}
		delete(s.probes, nmProbeKey(monitorName, probeID))
		return map[string]any{}

	case "GetProbe":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		probeID := nmDefaultString(pathParams, "probeId", "probe-000001")
		return map[string]any{"Probe": nmCloneMap(s.ensureProbeLocked(monitorName, probeID))}

	case "UpdateProbe":
		monitorName := nmDefaultString(pathParams, "monitorName", "stackyard-monitor")
		probeID := nmDefaultString(pathParams, "probeId", "probe-000001")
		probe := s.ensureProbeLocked(monitorName, probeID)
		s.applyProbePayloadLocked(probe, payload)
		s.ensureTagsForARNLocked(nmProbeARN(probe))
		return map[string]any{"Probe": nmCloneMap(probe)}

	case "ListTagsForResource":
		resourceARN := nmDefaultString(pathParams, "resourceArn", nmMonitorARNByName("stackyard-monitor"))
		return map[string]any{"Tags": nmCloneStringMap(s.tags[resourceARN])}

	case "TagResource":
		resourceARN := nmDefaultString(pathParams, "resourceArn", nmMonitorARNByName("stackyard-monitor"))
		s.ensureTagsForARNLocked(resourceARN)
		for key, value := range nmMapString(payload, "Tags") {
			s.tags[resourceARN][key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := nmDefaultString(pathParams, "resourceArn", nmMonitorARNByName("stackyard-monitor"))
		for _, key := range nmStringSlice(payload, "TagKeys") {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *networkMonitorStore) ensureMonitorLocked(name string) map[string]any {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-monitor"
	}
	if monitor := s.monitors[key]; monitor != nil {
		return monitor
	}

	monitor := map[string]any{
		"MonitorName": key,
		"MonitorArn":  nmMonitorARNByName(key),
		"State":       "ACTIVE",
		"CreatedAt":   time.Now().UTC(),
	}
	s.monitors[key] = monitor
	return monitor
}

func (s *networkMonitorStore) ensureProbeLocked(monitorName, probeID string) map[string]any {
	monitor := s.ensureMonitorLocked(monitorName)
	monitorKey := nmDefaultStringAny(monitor, "MonitorName", "stackyard-monitor")
	key := nmProbeKey(monitorKey, probeID)
	if probe := s.probes[key]; probe != nil {
		return probe
	}

	id := strings.TrimSpace(probeID)
	if id == "" {
		id = "probe-000001"
	}
	probe := map[string]any{
		"ProbeId":     id,
		"ProbeArn":    nmProbeARNByParts(monitorKey, id),
		"MonitorName": monitorKey,
		"State":       "ACTIVE",
		"CreatedAt":   time.Now().UTC(),
	}
	s.probes[key] = probe
	return probe
}

func (s *networkMonitorStore) applyMonitorPayloadLocked(monitor map[string]any, payload map[string]any) {
	if name := nmDefaultStringAny(payload, "monitorName", ""); name != "" {
		monitor["MonitorName"] = name
		monitor["MonitorArn"] = nmMonitorARNByName(name)
	}
	for key, value := range payload {
		monitor[key] = value
	}
}

func (s *networkMonitorStore) applyProbePayloadLocked(probe map[string]any, payload map[string]any) {
	for key, value := range payload {
		probe[key] = value
	}
	monitorName := nmDefaultStringAny(probe, "MonitorName", "stackyard-monitor")
	probeID := nmDefaultStringAny(probe, "ProbeId", nmDefaultStringAny(probe, "probeId", "probe-000001"))
	probe["MonitorName"] = monitorName
	probe["ProbeId"] = probeID
	probe["ProbeArn"] = nmProbeARNByParts(monitorName, probeID)
}

func (s *networkMonitorStore) ensureTagsForARNLocked(arn string) {
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{}
	}
}

func (s *networkMonitorStore) nextMonitorIDLocked() int64 {
	id := s.nextMonitor
	s.nextMonitor++
	return id
}

func (s *networkMonitorStore) nextProbeIDLocked() int64 {
	id := s.nextProbe
	s.nextProbe++
	return id
}

func nmProbeKey(monitorName, probeID string) string {
	return strings.TrimSpace(monitorName) + ":" + strings.TrimSpace(probeID)
}

func nmMonitorARN(monitor map[string]any) string {
	return nmMonitorARNByName(nmDefaultStringAny(monitor, "MonitorName", "stackyard-monitor"))
}

func nmMonitorARNByName(name string) string {
	key := strings.TrimSpace(name)
	if key == "" {
		key = "stackyard-monitor"
	}
	if strings.HasPrefix(key, "arn:") {
		return key
	}
	return fmt.Sprintf("arn:aws:networkmonitor:us-east-1:123456789012:monitor/%s", key)
}

func nmProbeARN(probe map[string]any) string {
	monitorName := nmDefaultStringAny(probe, "MonitorName", "stackyard-monitor")
	probeID := nmDefaultStringAny(probe, "ProbeId", "probe-000001")
	return nmProbeARNByParts(monitorName, probeID)
}

func nmProbeARNByParts(monitorName, probeID string) string {
	return fmt.Sprintf(
		"arn:aws:networkmonitor:us-east-1:123456789012:monitor/%s/probe/%s",
		strings.TrimSpace(monitorName),
		strings.TrimSpace(probeID),
	)
}

func nmDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			value := strings.TrimSpace(v)
			if value != "" {
				return value
			}
			break
		}
	}
	return fallback
}

func nmDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if str, ok := v.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					return str
				}
			}
			break
		}
	}
	return fallback
}

func nmMapString(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if raw, ok := v.(map[string]any); ok {
			out := map[string]string{}
			for rk, rv := range raw {
				if rs, ok := rv.(string); ok {
					out[rk] = rs
				}
			}
			return out
		}
	}
	return map[string]string{}
}

func nmStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		raw, ok := v.([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if str, ok := item.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					out = append(out, str)
				}
			}
		}
		return out
	}
	return nil
}

func nmCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = nmCloneAny(v)
	}
	return out
}

func nmCloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return nmCloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = nmCloneAny(item)
		}
		return out
	case map[string]string:
		return nmCloneStringMap(t)
	default:
		return t
	}
}

func nmCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
