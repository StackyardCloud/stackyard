package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotEventsStore struct {
	mu   sync.Mutex
	next int64
	tags map[string]map[string]string
}

func newIoTEventsStore() *iotEventsStore {
	return &iotEventsStore{
		next: 1,
		tags: map[string]map[string]string{
			"arn:aws:iotevents:us-east-1:123456789012:alarmModel/stackyard-alarm-model": {
				"seed": "true",
			},
		},
	}
}

func (s *iotEventsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "TagResource":
		arn := iotEventsResolveResourceARN(payload, query)
		if arn == "" {
			arn = "arn:aws:iotevents:us-east-1:123456789012:alarmModel/stackyard-alarm-model"
		}
		incoming := iotEventsExtractTagMap(iotEventsValue(payload, "tags"))
		current := s.tags[arn]
		if current == nil {
			current = map[string]string{}
		}
		for k, v := range incoming {
			current[k] = v
		}
		s.tags[arn] = current
		return map[string]any{}

	case "UntagResource":
		arn := iotEventsResolveResourceARN(payload, query)
		if arn == "" {
			arn = "arn:aws:iotevents:us-east-1:123456789012:alarmModel/stackyard-alarm-model"
		}
		current := s.tags[arn]
		if current == nil {
			return map[string]any{}
		}
		for _, key := range iotEventsStringSlice(iotEventsValue(payload, "tagKeys")) {
			delete(current, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(current, key)
			}
		}
		s.tags[arn] = current
		return map[string]any{}

	case "ListTagsForResource":
		arn := iotEventsResolveResourceARN(payload, query)
		if arn == "" {
			arn = "arn:aws:iotevents:us-east-1:123456789012:alarmModel/stackyard-alarm-model"
		}
		return map[string]any{"tags": iotEventsCloneStringMap(s.tags[arn])}

	case "PutLoggingOptions":
		return map[string]any{}
	case "DescribeLoggingOptions":
		return map[string]any{
			"loggingOptions": map[string]any{
				"level":   "ERROR",
				"enabled": true,
				"roleArn": "arn:aws:iam::123456789012:role/stackyard-iotevents",
			},
		}
	case "StartDetectorModelAnalysis":
		analysisID := iotEventsDefaultString(payload, "analysisId", s.nextID("analysis"))
		return map[string]any{
			"analysisId":     analysisID,
			"analysisStatus": "SUCCESSFUL",
		}
	case "DescribeDetectorModelAnalysis":
		analysisID := iotEventsPathParam(pathParams, "analysisId", "analysis-000001")
		return map[string]any{
			"analysisId":     analysisID,
			"analysisStatus": "SUCCESSFUL",
		}
	case "GetDetectorModelAnalysisResults":
		return map[string]any{
			"analysisResults": []any{},
		}
	}

	if strings.HasPrefix(action, "List") {
		key := iotEventsListKey(action)
		entry := map[string]any{
			"alarmModelName":    "stackyard-alarm-model",
			"detectorModelName": "stackyard-detector-model",
			"inputName":         "stackyard-input",
			"alarmModelArn":     "arn:aws:iotevents:us-east-1:123456789012:alarmModel/stackyard-alarm-model",
			"detectorModelArn":  "arn:aws:iotevents:us-east-1:123456789012:detectorModel/stackyard-detector-model",
			"inputArn":          "arn:aws:iotevents:us-east-1:123456789012:input/stackyard-input",
			"creationTime":      now,
			"lastUpdateTime":    now,
		}
		if key == "tags" {
			return map[string]any{"tags": map[string]string{}, "nextToken": ""}
		}
		return map[string]any{key: []any{entry}, "nextToken": ""}
	}

	if strings.HasPrefix(action, "Describe") {
		id := iotEventsResolveID(payload, pathParams)
		return map[string]any{
			"alarmModelName":    id,
			"detectorModelName": id,
			"inputName":         id,
			"alarmModelArn":     iotEventsARNFor("AlarmModel", id),
			"detectorModelArn":  iotEventsARNFor("DetectorModel", id),
			"inputArn":          iotEventsARNFor("Input", id),
			"creationTime":      now,
			"lastUpdateTime":    now,
			"status":            "ACTIVE",
		}
	}

	if strings.HasPrefix(action, "Create") {
		id := iotEventsResolveID(payload, pathParams)
		if id == "" {
			id = s.nextID("resource")
		}
		return map[string]any{
			"alarmModelArn":        iotEventsARNFor("AlarmModel", id),
			"detectorModelArn":     iotEventsARNFor("DetectorModel", id),
			"inputArn":             iotEventsARNFor("Input", id),
			"alarmModelVersion":    "1",
			"detectorModelVersion": "1",
		}
	}

	if strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Delete") {
		id := iotEventsResolveID(payload, pathParams)
		return map[string]any{
			"alarmModelArn":    iotEventsARNFor("AlarmModel", id),
			"detectorModelArn": iotEventsARNFor("DetectorModel", id),
			"inputArn":         iotEventsARNFor("Input", id),
		}
	}

	return map[string]any{"operation": action, "status": "SUCCESS", "timestamp": now}
}

func (s *iotEventsStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func iotEventsListKey(action string) string {
	keys := map[string]string{
		"ListAlarmModelVersions":    "alarmModelVersionSummaries",
		"ListAlarmModels":           "alarmModelSummaries",
		"ListDetectorModelVersions": "detectorModelVersionSummaries",
		"ListDetectorModels":        "detectorModelSummaries",
		"ListInputRoutings":         "routedResources",
		"ListInputs":                "inputSummaries",
		"ListTagsForResource":       "tags",
	}
	if key := keys[action]; key != "" {
		return key
	}
	return "summaries"
}

func iotEventsResolveID(payload map[string]any, pathParams map[string]string) string {
	keys := []string{"alarmModelName", "detectorModelName", "inputName", "analysisId"}
	for _, key := range keys {
		if v := iotEventsPathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v := iotEventsDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	return "stackyard"
}

func iotEventsARNFor(resourceType, id string) string {
	switch resourceType {
	case "AlarmModel":
		return fmt.Sprintf("arn:aws:iotevents:us-east-1:123456789012:alarmModel/%s", id)
	case "DetectorModel":
		return fmt.Sprintf("arn:aws:iotevents:us-east-1:123456789012:detectorModel/%s", id)
	case "Input":
		return fmt.Sprintf("arn:aws:iotevents:us-east-1:123456789012:input/%s", id)
	default:
		return fmt.Sprintf("arn:aws:iotevents:us-east-1:123456789012:resource/%s", id)
	}
}

func iotEventsResolveResourceARN(payload map[string]any, query url.Values) string {
	if value := strings.TrimSpace(iotEventsDefaultString(payload, "resourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(query.Get("resourceArn")); value != "" {
		return value
	}
	return ""
}

func iotEventsValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return value
		}
	}
	return nil
}

func iotEventsDefaultString(payload map[string]any, key, fallback string) string {
	value := iotEventsValue(payload, key)
	text := strings.TrimSpace(iotEventsToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func iotEventsPathParam(pathParams map[string]string, key, fallback string) string {
	if pathParams == nil {
		return fallback
	}
	if value, ok := pathParams[key]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return fallback
}

func iotEventsToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func iotEventsExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(iotEventsToString(val))
		}
	case map[string]string:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
	}
	return out
}

func iotEventsCloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = input[key]
	}
	return out
}

func iotEventsStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			text := strings.TrimSpace(iotEventsToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{strings.TrimSpace(v)}
	default:
		return nil
	}
}
