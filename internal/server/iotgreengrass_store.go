package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type iotGreengrassStore struct {
	mu   sync.Mutex
	next int64
	tags map[string]map[string]string
}

func newIoTGreengrassStore() *iotGreengrassStore {
	return &iotGreengrassStore{
		next: 1,
		tags: map[string]map[string]string{
			"arn:aws:greengrass:us-east-1:123456789012:components:stackyard-component": {
				"seed": "true",
			},
		},
	}
}

func (s *iotGreengrassStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "AssociateServiceRoleToAccount", "GetServiceRoleForAccount":
		return map[string]any{
			"associatedAt": now,
			"roleArn":      "arn:aws:iam::123456789012:role/stackyard-greengrass",
		}
	case "DisassociateServiceRoleFromAccount":
		return map[string]any{}

	case "TagResource":
		arn := iotGreengrassPathParam(pathParams, "resourceArn", "")
		if arn == "" {
			arn = iotGreengrassDefaultString(payload, "resourceArn", "arn:aws:greengrass:us-east-1:123456789012:components:stackyard-component")
		}
		incoming := iotGreengrassExtractTagMap(iotGreengrassValue(payload, "tags"))
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
		arn := iotGreengrassPathParam(pathParams, "resourceArn", "")
		if arn == "" {
			arn = iotGreengrassDefaultString(payload, "resourceArn", "arn:aws:greengrass:us-east-1:123456789012:components:stackyard-component")
		}
		current := s.tags[arn]
		if current == nil {
			return map[string]any{}
		}
		for _, key := range iotGreengrassStringSlice(iotGreengrassValue(payload, "tagKeys")) {
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
		arn := iotGreengrassPathParam(pathParams, "resourceArn", "")
		if arn == "" {
			arn = iotGreengrassDefaultString(payload, "resourceArn", "arn:aws:greengrass:us-east-1:123456789012:components:stackyard-component")
		}
		return map[string]any{"tags": iotGreengrassCloneStringMap(s.tags[arn])}

	case "BatchAssociateClientDeviceWithCoreDevice", "BatchDisassociateClientDeviceFromCoreDevice":
		return map[string]any{"errorEntries": []any{}}

	case "CreateComponentVersion":
		componentName := iotGreengrassDefaultString(payload, "componentName", "stackyard.component")
		version := iotGreengrassDefaultString(payload, "componentVersion", "1.0.0")
		arn := fmt.Sprintf("arn:aws:greengrass:us-east-1:123456789012:components:%s:versions:%s", componentName, version)
		return map[string]any{"arn": arn, "componentName": componentName, "componentVersion": version, "creationTimestamp": now, "status": map[string]any{"state": "AVAILABLE"}}

	case "CreateDeployment":
		id := iotGreengrassDefaultString(payload, "deploymentId", s.nextID("deployment"))
		return map[string]any{"deploymentId": id, "iotJobId": "job-" + id, "iotJobArn": "arn:aws:iot:us-east-1:123456789012:job/job-" + id, "creationTimestamp": now}

	case "CancelDeployment", "DeleteComponent", "DeleteCoreDevice", "DeleteDeployment", "UpdateConnectivityInfo":
		return map[string]any{}

	case "ResolveComponentCandidates":
		return map[string]any{"resolvedComponentVersions": []any{map[string]any{"arn": "arn:aws:greengrass:us-east-1:123456789012:components:stackyard.component:versions:1.0.0", "componentName": "stackyard.component", "componentVersion": "1.0.0", "recipe": ""}}}

	case "GetConnectivityInfo":
		return map[string]any{"connectivityInfo": []any{map[string]any{"hostAddress": "127.0.0.1", "id": "stackyard-connectivity", "metadata": "stackyard", "portNumber": 8883}}}
	}

	if strings.HasPrefix(action, "List") {
		key := iotGreengrassListKey(action)
		if key == "tags" {
			return map[string]any{"tags": map[string]string{}, "nextToken": ""}
		}
		entry := map[string]any{"arn": "arn:aws:greengrass:us-east-1:123456789012:resource/stackyard", "name": "stackyard", "status": "ACTIVE"}
		return map[string]any{key: []any{entry}, "nextToken": ""}
	}
	if strings.HasPrefix(action, "Get") || strings.HasPrefix(action, "Describe") {
		id := iotGreengrassResolveID(payload, pathParams)
		return map[string]any{
			"id":                        id,
			"arn":                       iotGreengrassARNFor(action, id),
			"creationTimestamp":         now,
			"lastStatusChangeTimestamp": now,
			"status":                    "ACTIVE",
		}
	}
	if strings.HasPrefix(action, "Create") {
		id := iotGreengrassResolveID(payload, pathParams)
		if id == "" {
			id = s.nextID("resource")
		}
		return map[string]any{"id": id, "arn": iotGreengrassARNFor(action, id), "creationTimestamp": now}
	}
	return map[string]any{"operation": action, "status": "SUCCESS", "timestamp": now}
}

func (s *iotGreengrassStore) nextID(prefix string) string {
	s.next++
	return fmt.Sprintf("%s-%06d", prefix, s.next)
}

func iotGreengrassListKey(action string) string {
	keys := map[string]string{
		"ListClientDevicesAssociatedWithCoreDevice": "associatedClientDevices",
		"ListComponents":           "components",
		"ListComponentVersions":    "componentVersions",
		"ListCoreDevices":          "coreDevices",
		"ListDeployments":          "deployments",
		"ListEffectiveDeployments": "effectiveDeployments",
		"ListInstalledComponents":  "installedComponents",
		"ListTagsForResource":      "tags",
	}
	if key := keys[action]; key != "" {
		return key
	}
	return "items"
}

func iotGreengrassResolveID(payload map[string]any, pathParams map[string]string) string {
	keys := []string{"deploymentId", "coreDeviceThingName", "thingName", "arn", "resourceArn"}
	for _, key := range keys {
		if v := iotGreengrassPathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range keys {
		if v := iotGreengrassDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	return "stackyard"
}

func iotGreengrassARNFor(action, id string) string {
	typeByAction := map[string]string{
		"Component":  "components",
		"CoreDevice": "coreDevices",
		"Deployment": "deployments",
	}
	for marker, resourceType := range typeByAction {
		if strings.Contains(action, marker) {
			return fmt.Sprintf("arn:aws:greengrass:us-east-1:123456789012:%s:%s", resourceType, id)
		}
	}
	return fmt.Sprintf("arn:aws:greengrass:us-east-1:123456789012:resources:%s", id)
}

func iotGreengrassValue(payload map[string]any, key string) any {
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

func iotGreengrassDefaultString(payload map[string]any, key, fallback string) string {
	value := iotGreengrassValue(payload, key)
	text := strings.TrimSpace(iotGreengrassToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func iotGreengrassPathParam(pathParams map[string]string, key, fallback string) string {
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

func iotGreengrassToString(value any) string {
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

func iotGreengrassExtractTagMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(iotGreengrassToString(val))
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

func iotGreengrassCloneStringMap(input map[string]string) map[string]string {
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

func iotGreengrassStringSlice(value any) []string {
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
			text := strings.TrimSpace(iotGreengrassToString(item))
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
