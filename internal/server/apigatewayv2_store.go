package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type apiGatewayV2Store struct {
	mu sync.Mutex

	nextID int

	apis map[string]map[string]any
	tags map[string]map[string]string
}

func newAPIGatewayV2Store() *apiGatewayV2Store {
	s := &apiGatewayV2Store{
		nextID: 2,
		apis: map[string]map[string]any{
			"api-000001": {
				"ApiId":                     "api-000001",
				"Name":                      "stackyard-default-http-api",
				"ProtocolType":              "HTTP",
				"ApiEndpoint":               "https://api-000001.execute-api.localhost.localstack.cloud",
				"DisableExecuteApiEndpoint": false,
			},
		},
		tags: map[string]map[string]string{},
	}
	s.tags[s.apiARN("api-000001")] = map[string]string{"stackyard": "true"}
	return s
}

func (s *apiGatewayV2Store) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateApi":
		id := s.nextAPIIDLocked()
		name := apiGatewayV2PayloadString(payload, "Name", "stackyard-http-api")
		protocol := strings.ToUpper(apiGatewayV2PayloadString(payload, "ProtocolType", "HTTP"))
		if protocol == "" {
			protocol = "HTTP"
		}
		api := map[string]any{
			"ApiId":                     id,
			"Name":                      name,
			"ProtocolType":              protocol,
			"ApiEndpoint":               fmt.Sprintf("https://%s.execute-api.localhost.localstack.cloud", id),
			"DisableExecuteApiEndpoint": false,
		}
		s.apis[id] = api
		return apiGatewayV2CloneMap(api)

	case "GetApi":
		id := apiGatewayV2PayloadString(payload, "ApiId", s.firstAPIIDLocked())
		return apiGatewayV2CloneMap(s.ensureAPILocked(id))

	case "GetApis":
		items := make([]any, 0, len(s.apis))
		for _, api := range apiGatewayV2SortedMaps(s.apis) {
			items = append(items, apiGatewayV2CloneMap(api))
		}
		return map[string]any{"Items": items, "NextToken": ""}

	case "DeleteApi":
		id := apiGatewayV2PayloadString(payload, "ApiId", s.firstAPIIDLocked())
		delete(s.apis, id)
		delete(s.tags, s.apiARN(id))
		return map[string]any{}

	case "TagResource":
		arn := apiGatewayV2PayloadString(payload, "ResourceArn", s.apiARN(s.firstAPIIDLocked()))
		incoming := apiGatewayV2PayloadTags(payload, "Tags")
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
		arn := apiGatewayV2PayloadString(payload, "ResourceArn", s.apiARN(s.firstAPIIDLocked()))
		keys := apiGatewayV2PayloadStringSlice(payload, "TagKeys")
		tags := s.tags[arn]
		if tags == nil {
			return map[string]any{}
		}
		for _, key := range keys {
			delete(tags, key)
		}
		if len(tags) == 0 {
			delete(s.tags, arn)
		}
		return map[string]any{}

	case "GetTags":
		arn := apiGatewayV2PayloadString(payload, "ResourceArn", s.apiARN(s.firstAPIIDLocked()))
		out := map[string]string{}
		for k, v := range s.tags[arn] {
			out[k] = v
		}
		return map[string]any{"Tags": out}
	}

	if strings.HasPrefix(action, "Get") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Create") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Update") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Import") || strings.HasPrefix(action, "Export") || strings.HasPrefix(action, "Reset") || strings.HasPrefix(action, "Enable") || strings.HasPrefix(action, "Disable") || strings.HasPrefix(action, "Reimport") || strings.HasPrefix(action, "Tag") || strings.HasPrefix(action, "Untag") {
		return map[string]any{}
	}
	return map[string]any{}
}

func (s *apiGatewayV2Store) apiARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "api-000001"
	}
	return "arn:aws:apigateway:us-east-1::/apis/" + id
}

func (s *apiGatewayV2Store) firstAPIIDLocked() string {
	if len(s.apis) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.apis))
	for key := range s.apis {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *apiGatewayV2Store) nextAPIIDLocked() string {
	id := fmt.Sprintf("api-%06d", s.nextID)
	s.nextID++
	return id
}

func (s *apiGatewayV2Store) ensureAPILocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "api-000001"
	}
	if api, ok := s.apis[id]; ok {
		return api
	}
	api := map[string]any{
		"ApiId":                     id,
		"Name":                      "stackyard-" + strings.ToLower(id),
		"ProtocolType":              "HTTP",
		"ApiEndpoint":               fmt.Sprintf("https://%s.execute-api.localhost.localstack.cloud", id),
		"DisableExecuteApiEndpoint": false,
	}
	s.apis[id] = api
	return api
}

func apiGatewayV2PayloadString(payload map[string]any, key, def string) string {
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

func apiGatewayV2PayloadTags(payload map[string]any, key string) map[string]string {
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

func apiGatewayV2PayloadStringSlice(payload map[string]any, key string) []string {
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

func apiGatewayV2SortedMaps(in map[string]map[string]any) []map[string]any {
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
		out = append(out, apiGatewayV2CloneMap(in[key]))
	}
	return out
}

func apiGatewayV2CloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
