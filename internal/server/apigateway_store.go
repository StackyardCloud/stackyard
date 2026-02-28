package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type apiGatewayStore struct {
	mu sync.Mutex

	nextID int

	restAPIs map[string]map[string]any
	tags     map[string]map[string]string
}

func newAPIGatewayStore() *apiGatewayStore {
	s := &apiGatewayStore{
		nextID: 2,
		restAPIs: map[string]map[string]any{
			"api-000001": {
				"id":          "api-000001",
				"name":        "stackyard-default-api",
				"createdDate": "2026-01-01T00:00:00Z",
				"description": "default local rest api",
			},
		},
		tags: map[string]map[string]string{},
	}
	s.tags[s.restAPIARN("api-000001")] = map[string]string{"stackyard": "true"}
	return s
}

func (s *apiGatewayStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateRestApi":
		id := s.nextRestAPIIDLocked()
		name := apiGatewayPayloadString(payload, "name", "stackyard-api")
		api := map[string]any{
			"id":          id,
			"name":        name,
			"createdDate": "2026-01-01T00:00:00Z",
			"description": apiGatewayPayloadString(payload, "description", ""),
		}
		s.restAPIs[id] = api
		return apiGatewayCloneMap(api)

	case "GetRestApi":
		id := apiGatewayPayloadString(payload, "restapi_id", s.firstRestAPIIDLocked())
		return apiGatewayCloneMap(s.ensureRestAPILocked(id))

	case "GetRestApis":
		items := make([]any, 0, len(s.restAPIs))
		for _, api := range apiGatewaySortedMaps(s.restAPIs) {
			items = append(items, apiGatewayCloneMap(api))
		}
		return map[string]any{"items": items, "position": ""}

	case "DeleteRestApi":
		id := apiGatewayPayloadString(payload, "restapi_id", s.firstRestAPIIDLocked())
		delete(s.restAPIs, id)
		delete(s.tags, s.restAPIARN(id))
		return map[string]any{}

	case "TagResource":
		arn := apiGatewayPayloadString(payload, "resourceArn", s.restAPIARN(s.firstRestAPIIDLocked()))
		incoming := apiGatewayPayloadTags(payload, "tags")
		if len(incoming) == 0 {
			incoming = map[string]string{"stackyard": "true"}
		}
		tags := s.tags[arn]
		if tags == nil {
			tags = map[string]string{}
			s.tags[arn] = tags
		}
		for k, v := range incoming {
			tags[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		arn := apiGatewayPayloadString(payload, "resourceArn", s.restAPIARN(s.firstRestAPIIDLocked()))
		keys := apiGatewayPayloadStringSlice(payload, "tagKeys")
		tags := s.tags[arn]
		if tags == nil {
			return map[string]any{}
		}
		for _, key := range keys {
			delete(tags, key)
		}
		if len(keys) == 0 {
			delete(tags, "stackyard")
		}
		if len(tags) == 0 {
			delete(s.tags, arn)
		}
		return map[string]any{}

	case "GetTags":
		arn := apiGatewayPayloadString(payload, "resourceArn", s.restAPIARN(s.firstRestAPIIDLocked()))
		out := map[string]string{}
		for k, v := range s.tags[arn] {
			out[k] = v
		}
		return map[string]any{"tags": out}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"items": []any{}, "position": ""}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Import") || strings.HasPrefix(action, "Flush") || strings.HasPrefix(action, "Generate") || strings.HasPrefix(action, "Test") || strings.HasPrefix(action, "Tag") || strings.HasPrefix(action, "Untag") || strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Reject") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *apiGatewayStore) restAPIARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "api-000001"
	}
	return "arn:aws:apigateway:us-east-1::/restapis/" + id
}

func (s *apiGatewayStore) firstRestAPIIDLocked() string {
	if len(s.restAPIs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.restAPIs))
	for k := range s.restAPIs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *apiGatewayStore) nextRestAPIIDLocked() string {
	id := fmt.Sprintf("api-%06d", s.nextID)
	s.nextID++
	return id
}

func (s *apiGatewayStore) ensureRestAPILocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "api-000001"
	}
	if api, ok := s.restAPIs[id]; ok {
		return api
	}
	api := map[string]any{
		"id":          id,
		"name":        "stackyard-" + strings.ToLower(id),
		"createdDate": "2026-01-01T00:00:00Z",
	}
	s.restAPIs[id] = api
	return api
}

func apiGatewayPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return strings.TrimSpace(def)
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return strings.TrimSpace(def)
	}
	s, ok := v.(string)
	if !ok {
		return strings.TrimSpace(def)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return strings.TrimSpace(def)
	}
	return s
}

func apiGatewayPayloadTags(payload map[string]any, key string) map[string]string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	src, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for k, v := range src {
		kk := strings.TrimSpace(k)
		if kk == "" {
			continue
		}
		if sv, ok := v.(string); ok {
			out[kk] = strings.TrimSpace(sv)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func apiGatewayPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func apiGatewaySortedMaps(in map[string]map[string]any) []map[string]any {
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
		out = append(out, apiGatewayCloneMap(in[key]))
	}
	return out
}

func apiGatewayCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
