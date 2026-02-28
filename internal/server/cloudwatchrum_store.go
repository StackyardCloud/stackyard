package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type cloudWatchRUMStore struct {
	mu sync.Mutex

	nextID int64

	appMonitors        map[string]map[string]any
	metricDefinitions  map[string][]map[string]any
	metricDestinations map[string][]map[string]any
	resourcePolicies   map[string]string
	tags               map[string]map[string]string
	monitorNameByID    map[string]string
}

func newCloudWatchRUMStore() *cloudWatchRUMStore {
	s := &cloudWatchRUMStore{
		nextID:             2,
		appMonitors:        map[string]map[string]any{},
		metricDefinitions:  map[string][]map[string]any{},
		metricDestinations: map[string][]map[string]any{},
		resourcePolicies:   map[string]string{},
		tags:               map[string]map[string]string{},
		monitorNameByID:    map[string]string{},
	}

	monitor := s.ensureAppMonitorLocked("stackyard-app-monitor")
	name := cloudWatchRUMDefaultStringAny(monitor, "Name", "stackyard-app-monitor")
	id := cloudWatchRUMDefaultStringAny(monitor, "Id", "stackyard-app-monitor")
	s.monitorNameByID[id] = name
	s.metricDefinitions[name] = []map[string]any{
		{
			"Name":          "Http4xxErrorCount",
			"Namespace":     "AWS/RUM",
			"DimensionKeys": []any{},
		},
	}
	s.metricDestinations[name] = []map[string]any{
		{
			"Destination": "CloudWatch",
			"Arn":         fmt.Sprintf("arn:aws:logs:us-east-1:123456789012:log-group:/aws/vendedlogs/RUMService_%s", name),
		},
	}
	s.resourcePolicies[name] = `{"Version":"2012-10-17","Statement":[]}`
	arn := cloudWatchRUMAppMonitorARN(name)
	s.tags[arn] = map[string]string{"seed": "true", "service": "cloudwatchrum"}
	return s
}

func (s *cloudWatchRUMStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "CreateAppMonitor":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		monitor := s.ensureAppMonitorLocked(name)
		s.applyAppMonitorPayloadLocked(monitor, payload)
		monitor["LastModified"] = now
		id := cloudWatchRUMDefaultStringAny(monitor, "Id", name)
		s.monitorNameByID[id] = name
		s.ensureTagsLocked(cloudWatchRUMAppMonitorARN(name))
		return map[string]any{"Id": id, "Name": name, "AppMonitor": cloudWatchRUMCloneMap(monitor)}

	case "GetAppMonitor":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		monitor := s.ensureAppMonitorLocked(name)
		return map[string]any{"AppMonitor": cloudWatchRUMCloneMap(monitor)}

	case "ListAppMonitors":
		names := make([]string, 0, len(s.appMonitors))
		for name := range s.appMonitors {
			names = append(names, name)
		}
		sort.Strings(names)
		summaries := make([]any, 0, len(names))
		for _, name := range names {
			monitor := s.appMonitors[name]
			summaries = append(summaries, map[string]any{
				"Name":         cloudWatchRUMDefaultStringAny(monitor, "Name", name),
				"Id":           cloudWatchRUMDefaultStringAny(monitor, "Id", name),
				"Arn":          cloudWatchRUMDefaultStringAny(monitor, "Arn", cloudWatchRUMAppMonitorARN(name)),
				"State":        cloudWatchRUMDefaultStringAny(monitor, "State", "ACTIVE"),
				"Created":      monitor["Created"],
				"LastModified": monitor["LastModified"],
			})
		}
		return map[string]any{"AppMonitorSummaries": summaries, "NextToken": ""}

	case "UpdateAppMonitor":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		monitor := s.ensureAppMonitorLocked(name)
		s.applyAppMonitorPayloadLocked(monitor, payload)
		monitor["LastModified"] = now
		return map[string]any{"AppMonitor": cloudWatchRUMCloneMap(monitor)}

	case "DeleteAppMonitor":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		if monitor := s.appMonitors[name]; monitor != nil {
			id := cloudWatchRUMDefaultStringAny(monitor, "Id", name)
			delete(s.monitorNameByID, id)
		}
		delete(s.appMonitors, name)
		delete(s.metricDefinitions, name)
		delete(s.metricDestinations, name)
		delete(s.resourcePolicies, name)
		delete(s.tags, cloudWatchRUMAppMonitorARN(name))
		return map[string]any{}

	case "PutRumEvents":
		name := s.resolveMonitorNameByIDLocked(cloudWatchRUMPathParam(pathParams, "Id", "stackyard-app-monitor"))
		_ = s.ensureAppMonitorLocked(name)
		events := cloudWatchRUMAnySlice(cloudWatchRUMPayloadValue(payload, "RumEvents"))
		if len(events) == 0 {
			events = cloudWatchRUMAnySlice(cloudWatchRUMPayloadValue(payload, "Batch"))
		}
		return map[string]any{
			"RumEventData": map[string]any{
				"AppMonitorName": name,
				"AcceptedEvents": len(events),
				"RejectedEvents": 0,
			},
		}

	case "GetAppMonitorData":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		return map[string]any{
			"AppMonitorName": name,
			"Events":         []any{},
			"RumEventData":   []any{},
			"NextToken":      "",
		}

	case "BatchCreateRumMetricDefinitions":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		defs := cloudWatchRUMDefinitionSlice(cloudWatchRUMPayloadValue(payload, "MetricDefinitions"))
		if len(defs) == 0 {
			defs = []map[string]any{{"Name": "Http5xxErrorCount", "Namespace": "AWS/RUM", "DimensionKeys": []any{}}}
		}
		s.metricDefinitions[name] = append(s.metricDefinitions[name], defs...)
		return map[string]any{"Errors": []any{}, "MetricDefinitions": cloudWatchRUMCloneDefinitionList(s.metricDefinitions[name])}

	case "BatchGetRumMetricDefinitions":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		defs := cloudWatchRUMCloneDefinitionList(s.metricDefinitions[name])
		return map[string]any{"MetricDefinitions": defs, "NextToken": ""}

	case "UpdateRumMetricDefinition":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		defs := s.metricDefinitions[name]
		if len(defs) == 0 {
			defs = []map[string]any{{"Name": "Http4xxErrorCount", "Namespace": "AWS/RUM", "DimensionKeys": []any{}}}
		}
		def := defs[0]
		for k, v := range payload {
			def[k] = v
		}
		if strings.TrimSpace(cloudWatchRUMDefaultStringAny(def, "Name", "")) == "" {
			def["Name"] = "Http4xxErrorCount"
		}
		defs[0] = def
		s.metricDefinitions[name] = defs
		return map[string]any{"MetricDefinition": cloudWatchRUMCloneMap(def)}

	case "BatchDeleteRumMetricDefinitions":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		deleteNames := cloudWatchRUMStringSlice(cloudWatchRUMPayloadValue(payload, "MetricDefinitionIds"))
		if len(deleteNames) == 0 {
			deleteNames = cloudWatchRUMStringSlice(cloudWatchRUMPayloadValue(payload, "MetricDefinitionNames"))
		}
		if len(deleteNames) == 0 {
			s.metricDefinitions[name] = []map[string]any{}
			return map[string]any{"Errors": []any{}, "MetricDefinitions": []any{}}
		}
		deleteSet := map[string]struct{}{}
		for _, item := range deleteNames {
			deleteSet[strings.TrimSpace(item)] = struct{}{}
		}
		filtered := make([]map[string]any, 0, len(s.metricDefinitions[name]))
		for _, def := range s.metricDefinitions[name] {
			key := strings.TrimSpace(cloudWatchRUMDefaultStringAny(def, "Name", ""))
			if _, remove := deleteSet[key]; remove {
				continue
			}
			filtered = append(filtered, def)
		}
		s.metricDefinitions[name] = filtered
		return map[string]any{"Errors": []any{}, "MetricDefinitions": cloudWatchRUMCloneDefinitionList(filtered)}

	case "PutRumMetricsDestination":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		destination := map[string]any{
			"Destination": cloudWatchRUMDefaultString(payload, "Destination", "CloudWatch"),
			"Arn":         cloudWatchRUMDefaultString(payload, "DestinationArn", fmt.Sprintf("arn:aws:logs:us-east-1:123456789012:log-group:/aws/vendedlogs/RUMService_%s", name)),
		}
		s.metricDestinations[name] = append(s.metricDestinations[name], destination)
		return map[string]any{"Destination": cloudWatchRUMCloneMap(destination)}

	case "ListRumMetricsDestinations":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		_ = s.ensureAppMonitorLocked(name)
		dests := make([]any, 0, len(s.metricDestinations[name]))
		for _, item := range s.metricDestinations[name] {
			dests = append(dests, cloudWatchRUMCloneMap(item))
		}
		return map[string]any{"Destinations": dests, "NextToken": ""}

	case "DeleteRumMetricsDestination":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		delete(s.metricDestinations, name)
		return map[string]any{}

	case "PutResourcePolicy":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		policy := cloudWatchRUMDefaultString(payload, "PolicyDocument", `{"Version":"2012-10-17","Statement":[]}`)
		s.resourcePolicies[name] = policy
		return map[string]any{"PolicyDocument": policy}

	case "GetResourcePolicy":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		policy := strings.TrimSpace(s.resourcePolicies[name])
		if policy == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
			s.resourcePolicies[name] = policy
		}
		return map[string]any{"PolicyDocument": policy}

	case "DeleteResourcePolicy":
		name := cloudWatchRUMResolveMonitorName(payload, pathParams, "stackyard-app-monitor")
		delete(s.resourcePolicies, name)
		return map[string]any{}

	case "TagResource":
		arn := cloudWatchRUMResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = cloudWatchRUMAppMonitorARN("stackyard-app-monitor")
		}
		tags := s.ensureTagsLocked(arn)
		for key, value := range cloudWatchRUMStringMap(cloudWatchRUMPayloadValue(payload, "Tags")) {
			if strings.TrimSpace(key) == "" {
				continue
			}
			tags[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
		return map[string]any{}

	case "UntagResource":
		arn := cloudWatchRUMResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = cloudWatchRUMAppMonitorARN("stackyard-app-monitor")
		}
		tags := s.ensureTagsLocked(arn)
		for _, key := range cloudWatchRUMStringSlice(cloudWatchRUMPayloadValue(payload, "TagKeys")) {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := cloudWatchRUMResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = cloudWatchRUMAppMonitorARN("stackyard-app-monitor")
		}
		return map[string]any{"Tags": cloudWatchRUMCloneStringMap(s.tags[arn])}
	}

	return map[string]any{}
}

func (s *cloudWatchRUMStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *cloudWatchRUMStore) ensureAppMonitorLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-app-monitor"
	}
	if monitor := s.appMonitors[name]; monitor != nil {
		return monitor
	}

	id := name
	if _, exists := s.monitorNameByID[id]; exists {
		id = fmt.Sprintf("app-monitor-%06d", s.nextLocked())
	}
	monitor := map[string]any{
		"Name":         name,
		"Id":           id,
		"Arn":          cloudWatchRUMAppMonitorARN(name),
		"State":        "ACTIVE",
		"Domain":       "stackyard.local",
		"Created":      time.Now().UTC(),
		"LastModified": time.Now().UTC(),
		"AppMonitorConfiguration": map[string]any{
			"Telemetries":  []any{"errors", "performance", "http"},
			"AllowCookies": true,
		},
	}
	s.appMonitors[name] = monitor
	s.monitorNameByID[id] = name
	if s.metricDefinitions[name] == nil {
		s.metricDefinitions[name] = []map[string]any{}
	}
	if s.metricDestinations[name] == nil {
		s.metricDestinations[name] = []map[string]any{}
	}
	return monitor
}

func (s *cloudWatchRUMStore) applyAppMonitorPayloadLocked(monitor map[string]any, payload map[string]any) {
	for key, value := range payload {
		monitor[key] = value
	}
	name := strings.TrimSpace(cloudWatchRUMDefaultStringAny(monitor, "Name", "stackyard-app-monitor"))
	if name == "" {
		name = "stackyard-app-monitor"
	}
	monitor["Name"] = name
	monitor["Arn"] = cloudWatchRUMAppMonitorARN(name)
	if _, ok := monitor["Id"]; !ok || strings.TrimSpace(cloudWatchRUMDefaultStringAny(monitor, "Id", "")) == "" {
		monitor["Id"] = name
	}
	if _, ok := monitor["Created"]; !ok {
		monitor["Created"] = time.Now().UTC()
	}
	if _, ok := monitor["State"]; !ok {
		monitor["State"] = "ACTIVE"
	}
}

func (s *cloudWatchRUMStore) ensureTagsLocked(resourceARN string) map[string]string {
	tags := s.tags[resourceARN]
	if tags == nil {
		tags = map[string]string{}
		s.tags[resourceARN] = tags
	}
	return tags
}

func (s *cloudWatchRUMStore) resolveMonitorNameByIDLocked(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "stackyard-app-monitor"
	}
	if name := strings.TrimSpace(s.monitorNameByID[id]); name != "" {
		return name
	}
	return id
}

func cloudWatchRUMResolveMonitorName(payload map[string]any, pathParams map[string]string, fallback string) string {
	for _, key := range []string{"AppMonitorName", "Name", "Id", "appMonitorName", "name", "id"} {
		if v := cloudWatchRUMPathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range []string{"AppMonitorName", "Name", "Id", "appMonitorName", "name", "id"} {
		if v := cloudWatchRUMDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	return fallback
}

func cloudWatchRUMResolveResourceARN(payload map[string]any, pathParams map[string]string, query url.Values) string {
	if v := cloudWatchRUMDefaultString(payload, "ResourceArn", ""); v != "" {
		return v
	}
	if v := cloudWatchRUMPathParam(pathParams, "ResourceArn", ""); v != "" {
		return v
	}
	if v := strings.TrimSpace(query.Get("ResourceArn")); v != "" {
		return v
	}
	if v := strings.TrimSpace(query.Get("resourceArn")); v != "" {
		return v
	}
	return ""
}

func cloudWatchRUMAppMonitorARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-app-monitor"
	}
	if strings.HasPrefix(name, "arn:") {
		return name
	}
	return fmt.Sprintf("arn:aws:rum:us-east-1:123456789012:appmonitor/%s", name)
}

func cloudWatchRUMPathParam(values map[string]string, key, fallback string) string {
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

func cloudWatchRUMPayloadValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return value
		}
	}
	return nil
}

func cloudWatchRUMDefaultString(values map[string]any, key, fallback string) string {
	if v := cloudWatchRUMPayloadValue(values, key); v != nil {
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return fallback
}

func cloudWatchRUMDefaultStringAny(values map[string]any, key, fallback string) string {
	if values == nil {
		return fallback
	}
	if value, ok := values[key]; ok {
		if s, ok := value.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	for k, value := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if s, ok := value.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					return s
				}
			}
			break
		}
	}
	return fallback
}

func cloudWatchRUMAnySlice(v any) []any {
	switch t := v.(type) {
	case []any:
		out := make([]any, len(t))
		copy(out, t)
		return out
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, cloudWatchRUMCloneMap(item))
		}
		return out
	default:
		return nil
	}
}

func cloudWatchRUMStringMap(v any) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	switch t := v.(type) {
	case map[string]string:
		for key, value := range t {
			if strings.TrimSpace(key) == "" {
				continue
			}
			out[key] = strings.TrimSpace(value)
		}
	case map[string]any:
		for key, value := range t {
			if strings.TrimSpace(key) == "" {
				continue
			}
			if s, ok := value.(string); ok {
				out[key] = strings.TrimSpace(s)
			}
		}
	}
	return out
}

func cloudWatchRUMStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		out := make([]string, 0, len(t))
		for _, item := range t {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func cloudWatchRUMDefinitionSlice(v any) []map[string]any {
	items := cloudWatchRUMAnySlice(v)
	if len(items) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]any); ok {
			out = append(out, cloudWatchRUMCloneMap(m))
		}
	}
	return out
}

func cloudWatchRUMCloneDefinitionList(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, cloudWatchRUMCloneMap(item))
	}
	return out
}

func cloudWatchRUMCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	buf, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		return map[string]any{}
	}
	if out == nil {
		out = map[string]any{}
	}
	return out
}

func cloudWatchRUMCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
