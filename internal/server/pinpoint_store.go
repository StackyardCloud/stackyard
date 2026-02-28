package server

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	pinpointDefaultRegion    = "us-east-1"
	pinpointDefaultAccountID = "123456789012"
)

type pinpointStore struct {
	mu sync.Mutex

	nextAppSerial int64

	apps map[string]map[string]any
	tags map[string]map[string]string
}

func newPinpointStore() *pinpointStore {
	now := time.Now().UTC()
	s := &pinpointStore{
		nextAppSerial: 2,
		apps:          map[string]map[string]any{},
		tags:          map[string]map[string]string{},
	}
	seed := s.ensureAppLocked("app-000001", "stackyard-pinpoint-app", now)
	s.ensureTagMapLocked(pinpointAppARN(pinpointAnyString(seed, "Id", "app-000001")))
	return s
}

func (s *pinpointStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := pinpointMergeContext(payload, pathParams, query)
	appID := pinpointString(ctx, []string{"application-id", "applicationId", "ApplicationId", "id", "Id"}, "app-000001")
	appName := pinpointString(ctx, []string{"Name", "name", "ApplicationName", "applicationName"}, "stackyard-pinpoint-app")
	resourceARN := pinpointString(ctx, []string{"resource-arn", "resourceArn", "ResourceArn"}, pinpointAppARN(appID))
	templateName := pinpointString(ctx, []string{"template-name", "TemplateName", "templateName"}, "stackyard-template")
	templateType := pinpointString(ctx, []string{"template-type", "TemplateType", "templateType"}, "email")

	app := s.ensureAppLocked(appID, appName, now)

	s.applyTagMutationsLocked(action, payload, query, resourceARN)

	summary := map[string]any{
		"Action":        action,
		"ApplicationId": pinpointAnyString(app, "Id", appID),
		"RequestId":     "stackyard-" + strings.ToLower(action),
		"Status":        "SUCCESS",
		"Timestamp":     now.Format(time.RFC3339),
	}

	switch action {
	case "CreateApp":
		name := pinpointString(ctx, []string{"Name", "name"}, "stackyard-pinpoint-app")
		id := fmt.Sprintf("app-%06d", s.nextAppSerial)
		s.nextAppSerial++
		created := s.ensureAppLocked(id, name, now)
		return map[string]any{
			"ApplicationResponse": pinpointCloneMap(created),
		}
	case "DeleteApp":
		delete(s.apps, appID)
		delete(s.tags, pinpointAppARN(appID))
		return map[string]any{}
	case "GetApps":
		return map[string]any{"Item": s.listAppsLocked(), "NextToken": ""}
	case "GetApp":
		return map[string]any{"ApplicationResponse": pinpointCloneMap(app)}
	case "TagResource", "UntagResource":
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": pinpointCloneStringMap(s.ensureTagMapLocked(resourceARN))}
	}

	if strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{
			"Action":    action,
			"Item":      []any{pinpointCloneMap(app)},
			"NextToken": "",
		}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{
			"Action":                action,
			"ApplicationResponse":   pinpointCloneMap(app),
			"TemplateName":          templateName,
			"TemplateType":          templateType,
			"ResourceArn":           resourceARN,
			"OperationRequestState": "OK",
		}
	}

	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Send") || strings.HasPrefix(action, "Remove") || strings.HasPrefix(action, "Verify") || action == "PhoneNumberValidate" {
		return summary
	}

	return map[string]any{}
}

func (s *pinpointStore) ensureAppLocked(id, name string, now time.Time) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "app-000001"
	}
	if existing := s.apps[id]; existing != nil {
		if n := strings.TrimSpace(name); n != "" {
			existing["Name"] = n
		}
		existing["LastModifiedDate"] = now.Format(time.RFC3339)
		return existing
	}
	if strings.TrimSpace(name) == "" {
		name = "stackyard-pinpoint-app"
	}
	app := map[string]any{
		"Id":               id,
		"Arn":              pinpointAppARN(id),
		"Name":             name,
		"CreationDate":     now.Format(time.RFC3339),
		"LastModifiedDate": now.Format(time.RFC3339),
	}
	s.apps[id] = app
	return app
}

func (s *pinpointStore) listAppsLocked() []any {
	ids := make([]string, 0, len(s.apps))
	for id := range s.apps {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, pinpointCloneMap(s.apps[id]))
	}
	return out
}

func (s *pinpointStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = pinpointAppARN("app-000001")
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	tags := map[string]string{"stackyard": "true", "env": "coverage"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *pinpointStore) applyTagMutationsLocked(action string, payload map[string]any, query url.Values, resourceARN string) {
	tags := s.ensureTagMapLocked(resourceARN)
	switch action {
	case "TagResource":
		for k, v := range pinpointExtractTags(payload) {
			tags[k] = v
		}
	case "UntagResource":
		for _, key := range pinpointExtractTagKeys(payload, query) {
			delete(tags, key)
		}
	}
}

func pinpointAppARN(appID string) string {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = "app-000001"
	}
	return "arn:aws:mobiletargeting:" + pinpointDefaultRegion + ":" + pinpointDefaultAccountID + ":apps/" + appID
}

func pinpointMergeContext(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	ctx := map[string]any{}
	for k, v := range payload {
		ctx[k] = v
	}
	for k, v := range pathParams {
		ctx[k] = v
	}
	for k, vals := range query {
		if len(vals) == 0 {
			continue
		}
		ctx[k] = vals[len(vals)-1]
	}
	return ctx
}

func pinpointString(source map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if val, ok := source[key]; ok {
			if s := strings.TrimSpace(pinpointAnyToString(val)); s != "" {
				return s
			}
		}
	}
	return def
}

func pinpointAnyToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case int:
		return strconv.Itoa(t)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func pinpointAnyString(source map[string]any, key, def string) string {
	if source == nil {
		return def
	}
	if v, ok := source[key]; ok {
		if s := strings.TrimSpace(pinpointAnyToString(v)); s != "" {
			return s
		}
	}
	return def
}

func pinpointCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func pinpointCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func pinpointExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case map[string]any:
			for k, v := range typed {
				if name := strings.TrimSpace(k); name != "" {
					out[name] = strings.TrimSpace(pinpointAnyToString(v))
				}
			}
		case map[string]string:
			for k, v := range typed {
				if name := strings.TrimSpace(k); name != "" {
					out[name] = strings.TrimSpace(v)
				}
			}
		}
	}
	return out
}

func pinpointExtractTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	for _, key := range []string{"tagKeys", "TagKeys"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case []any:
			for _, v := range typed {
				if s := strings.TrimSpace(pinpointAnyToString(v)); s != "" {
					keys = append(keys, s)
				}
			}
		case []string:
			for _, v := range typed {
				if s := strings.TrimSpace(v); s != "" {
					keys = append(keys, s)
				}
			}
		case string:
			if s := strings.TrimSpace(typed); s != "" {
				keys = append(keys, s)
			}
		}
	}

	for _, queryKey := range []string{"tagKeys", "TagKeys"} {
		for _, v := range query[queryKey] {
			if s := strings.TrimSpace(v); s != "" {
				keys = append(keys, s)
			}
		}
	}

	if len(keys) == 0 {
		return keys
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}
