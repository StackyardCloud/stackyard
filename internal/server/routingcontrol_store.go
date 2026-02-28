package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	routingControlDefaultARN             = "routingcontrol-000001"
	routingControlDefaultName            = "stackyard-routing-control"
	routingControlDefaultControlPanelARN = "controlpanel-000001"
	routingControlDefaultControlPanel    = "stackyard-control-panel"
	routingControlDefaultOwner           = "123456789012"
)

type routingControlStore struct {
	mu sync.Mutex

	routingControls map[string]map[string]any
}

func newRoutingControlStore() *routingControlStore {
	s := &routingControlStore{
		routingControls: map[string]map[string]any{},
	}
	s.seedDefaultsLocked(time.Now().UTC().Format(time.RFC3339))
	return s
}

func (s *routingControlStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	s.seedDefaultsLocked(now)

	switch action {
	case "GetRoutingControlState":
		arn := rtcFirstNonEmpty(
			rtcPathString(pathParams, "RoutingControlArn", ""),
			rtcPayloadString(payload, "RoutingControlArn", ""),
			strings.TrimSpace(query.Get("RoutingControlArn")),
			routingControlDefaultARN,
		)
		rc := s.ensureRoutingControlLocked(arn, now)
		return map[string]any{
			"RoutingControlArn":   rtcAnyString(rc["RoutingControlArn"]),
			"RoutingControlState": rtcAnyString(rc["RoutingControlState"]),
		}

	case "ListRoutingControls":
		controlPanelARN := rtcFirstNonEmpty(
			rtcPayloadString(payload, "ControlPanelArn", ""),
			strings.TrimSpace(query.Get("ControlPanelArn")),
		)
		items := make([]any, 0, len(s.routingControls))
		for _, rc := range s.listRoutingControlsLocked() {
			if controlPanelARN != "" && !strings.EqualFold(rtcAnyString(rc["ControlPanelArn"]), controlPanelARN) {
				continue
			}
			items = append(items, rtcCloneMap(rc))
		}
		return map[string]any{
			"RoutingControls": items,
			"NextToken":       "",
		}

	case "UpdateRoutingControlState":
		arn := rtcFirstNonEmpty(
			rtcPathString(pathParams, "RoutingControlArn", ""),
			rtcPayloadString(payload, "RoutingControlArn", ""),
			strings.TrimSpace(query.Get("RoutingControlArn")),
			routingControlDefaultARN,
		)
		rc := s.ensureRoutingControlLocked(arn, now)
		if state := rtcNormalizeState(rtcPayloadString(payload, "RoutingControlState", "")); state != "" {
			rc["RoutingControlState"] = state
		}
		rc["UpdatedAt"] = now
		return map[string]any{}

	case "UpdateRoutingControlStates":
		entries := rtcAnySlice(rtcLookupAny(payload, "UpdateRoutingControlStateEntries"))
		if len(entries) == 0 {
			entries = rtcAnySlice(rtcLookupAny(payload, "updateRoutingControlStateEntries"))
		}

		if len(entries) == 0 {
			arn := rtcFirstNonEmpty(
				rtcPayloadString(payload, "RoutingControlArn", ""),
				routingControlDefaultARN,
			)
			rc := s.ensureRoutingControlLocked(arn, now)
			if state := rtcNormalizeState(rtcPayloadString(payload, "RoutingControlState", "")); state != "" {
				rc["RoutingControlState"] = state
			}
			rc["UpdatedAt"] = now
			return map[string]any{}
		}

		for _, entryAny := range entries {
			entry, ok := entryAny.(map[string]any)
			if !ok {
				continue
			}
			arn := rtcFirstNonEmpty(
				rtcPayloadString(entry, "RoutingControlArn", ""),
				routingControlDefaultARN,
			)
			rc := s.ensureRoutingControlLocked(arn, now)
			if state := rtcNormalizeState(rtcPayloadString(entry, "RoutingControlState", "")); state != "" {
				rc["RoutingControlState"] = state
			}
			rc["UpdatedAt"] = now
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{"Status": "On"}
	}
	if strings.HasPrefix(action, "Update") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *routingControlStore) seedDefaultsLocked(now string) {
	s.ensureRoutingControlLocked(routingControlDefaultARN, now)
}

func (s *routingControlStore) ensureRoutingControlLocked(arn, now string) map[string]any {
	arn = rtcFirstNonEmpty(strings.TrimSpace(arn), routingControlDefaultARN)
	item, ok := s.routingControls[arn]
	if !ok {
		item = map[string]any{
			"RoutingControlArn":   arn,
			"RoutingControlName":  routingControlDefaultName,
			"RoutingControlState": "On",
			"ControlPanelArn":     routingControlDefaultControlPanelARN,
			"ControlPanelName":    routingControlDefaultControlPanel,
			"Owner":               routingControlDefaultOwner,
			"CreatedAt":           now,
			"UpdatedAt":           now,
		}
		s.routingControls[arn] = item
	}
	if strings.TrimSpace(rtcAnyString(item["RoutingControlArn"])) == "" {
		item["RoutingControlArn"] = arn
	}
	if strings.TrimSpace(rtcAnyString(item["RoutingControlName"])) == "" {
		item["RoutingControlName"] = routingControlDefaultName
	}
	if strings.TrimSpace(rtcAnyString(item["RoutingControlState"])) == "" {
		item["RoutingControlState"] = "On"
	}
	if strings.TrimSpace(rtcAnyString(item["ControlPanelArn"])) == "" {
		item["ControlPanelArn"] = routingControlDefaultControlPanelARN
	}
	if strings.TrimSpace(rtcAnyString(item["ControlPanelName"])) == "" {
		item["ControlPanelName"] = routingControlDefaultControlPanel
	}
	if strings.TrimSpace(rtcAnyString(item["Owner"])) == "" {
		item["Owner"] = routingControlDefaultOwner
	}
	return item
}

func (s *routingControlStore) listRoutingControlsLocked() []map[string]any {
	arns := make([]string, 0, len(s.routingControls))
	for arn := range s.routingControls {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	out := make([]map[string]any, 0, len(arns))
	for _, arn := range arns {
		out = append(out, s.routingControls[arn])
	}
	return out
}

func rtcLookupAny(payload map[string]any, key string) any {
	if payload == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return v
		}
	}
	return nil
}

func rtcPathString(values map[string]string, key, def string) string {
	if values == nil || strings.TrimSpace(key) == "" {
		return def
	}
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if strings.TrimSpace(v) == "" {
				return def
			}
			return v
		}
	}
	return def
}

func rtcPayloadString(payload map[string]any, key, def string) string {
	value := rtcLookupAny(payload, key)
	if value == nil {
		return def
	}
	if str := strings.TrimSpace(rtcAnyString(value)); str != "" {
		return str
	}
	return def
}

func rtcAnyString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func rtcAnySlice(value any) []any {
	switch v := value.(type) {
	case []any:
		return v
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func rtcFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func rtcNormalizeState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "on":
		return "On"
	case "off":
		return "Off"
	default:
		return ""
	}
}

func rtcCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return rtcCloneMap(v)
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, rtcCloneAny(item))
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			out = append(out, rtcCloneMap(item))
		}
		return out
	default:
		return v
	}
}

func rtcCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = rtcCloneAny(v)
	}
	return out
}
