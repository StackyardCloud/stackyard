package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type amplifyUIBuilderStore struct {
	mu sync.Mutex

	nextComponentID  int64
	nextFormID       int64
	nextThemeID      int64
	nextCodegenJobID int64

	components  map[string]map[string]map[string]map[string]any
	forms       map[string]map[string]map[string]map[string]any
	themes      map[string]map[string]map[string]map[string]any
	codegenJobs map[string]map[string]map[string]map[string]any

	metadataFlags map[string]map[string]map[string]bool
	tokens        map[string]map[string]any
	tags          map[string]map[string]string
}

func newAmplifyUIBuilderStore() *amplifyUIBuilderStore {
	now := time.Now().UTC()
	s := &amplifyUIBuilderStore{
		nextComponentID:  2,
		nextFormID:       2,
		nextThemeID:      2,
		nextCodegenJobID: 2,
		components:       map[string]map[string]map[string]map[string]any{},
		forms:            map[string]map[string]map[string]map[string]any{},
		themes:           map[string]map[string]map[string]map[string]any{},
		codegenJobs:      map[string]map[string]map[string]map[string]any{},
		metadataFlags:    map[string]map[string]map[string]bool{},
		tokens:           map[string]map[string]any{},
		tags:             map[string]map[string]string{},
	}

	s.ensureComponentLocked("d1234567890", "dev", "component-000001", now)
	s.ensureFormLocked("d1234567890", "dev", "form-000001", now)
	s.ensureThemeLocked("d1234567890", "dev", "theme-000001", now)
	s.ensureCodegenJobLocked("d1234567890", "dev", "codegen-job-000001", now)
	s.ensureMetadataFlagLocked("d1234567890", "dev", "isRelationshipSupported", true)
	s.ensureTokenLocked("figma", now)
	s.ensureTagsLocked("arn:aws:amplify:us-east-1:123456789012:apps/d1234567890")

	return s
}

func (s *amplifyUIBuilderStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := amplifyUIBuilderMergeMaps(payload, pathParams, query)

	appID := amplifyUIBuilderString(ctx, "appId", "d1234567890")
	environmentName := amplifyUIBuilderString(ctx, "environmentName", "dev")
	resourceID := amplifyUIBuilderString(ctx, "id", "component-000001")
	provider := amplifyUIBuilderString(ctx, "provider", "figma")
	resourceARN := amplifyUIBuilderString(ctx, "resourceArn", "arn:aws:amplify:us-east-1:123456789012:apps/d1234567890")
	featureName := amplifyUIBuilderString(ctx, "featureName", "isRelationshipSupported")

	s.ensureComponentLocked(appID, environmentName, "component-000001", now)
	s.ensureFormLocked(appID, environmentName, "form-000001", now)
	s.ensureThemeLocked(appID, environmentName, "theme-000001", now)
	s.ensureCodegenJobLocked(appID, environmentName, "codegen-job-000001", now)
	s.ensureMetadataFlagLocked(appID, environmentName, "isRelationshipSupported", true)
	s.ensureTokenLocked(provider, now)
	s.ensureTagsLocked(resourceARN)

	s.applySharedNamesLocked(payload, appID, environmentName, now)

	switch action {
	case "CreateComponent":
		id := amplifyUIBuilderString(payload, "name", "")
		if id == "" {
			id = amplifyUIBuilderString(payload, "id", "")
		}
		if id == "" {
			id = s.nextComponentIdentifierLocked()
		}
		component := s.ensureComponentLocked(appID, environmentName, id, now)
		s.updateResourceLocked(component, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(component)}

	case "DeleteComponent":
		if env := s.components[appID]; env != nil {
			if items := env[environmentName]; items != nil {
				delete(items, resourceID)
			}
		}
		return map[string]any{}

	case "GetComponent":
		component := s.ensureComponentLocked(appID, environmentName, resourceID, now)
		return map[string]any{"component": amplifyUIBuilderCloneMap(component)}

	case "ListComponents", "ExportComponents":
		return map[string]any{"entities": s.listResourcesLocked(s.components, appID, environmentName), "nextToken": ""}

	case "UpdateComponent":
		component := s.ensureComponentLocked(appID, environmentName, resourceID, now)
		s.updateResourceLocked(component, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(component)}

	case "CreateForm":
		id := amplifyUIBuilderString(payload, "name", "")
		if id == "" {
			id = amplifyUIBuilderString(payload, "id", "")
		}
		if id == "" {
			id = s.nextFormIdentifierLocked()
		}
		form := s.ensureFormLocked(appID, environmentName, id, now)
		s.updateResourceLocked(form, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(form)}

	case "DeleteForm":
		if env := s.forms[appID]; env != nil {
			if items := env[environmentName]; items != nil {
				delete(items, resourceID)
			}
		}
		return map[string]any{}

	case "GetForm":
		form := s.ensureFormLocked(appID, environmentName, resourceID, now)
		return map[string]any{"form": amplifyUIBuilderCloneMap(form)}

	case "ListForms", "ExportForms":
		return map[string]any{"entities": s.listResourcesLocked(s.forms, appID, environmentName), "nextToken": ""}

	case "UpdateForm":
		form := s.ensureFormLocked(appID, environmentName, resourceID, now)
		s.updateResourceLocked(form, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(form)}

	case "CreateTheme":
		id := amplifyUIBuilderString(payload, "name", "")
		if id == "" {
			id = amplifyUIBuilderString(payload, "id", "")
		}
		if id == "" {
			id = s.nextThemeIdentifierLocked()
		}
		theme := s.ensureThemeLocked(appID, environmentName, id, now)
		s.updateResourceLocked(theme, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(theme)}

	case "DeleteTheme":
		if env := s.themes[appID]; env != nil {
			if items := env[environmentName]; items != nil {
				delete(items, resourceID)
			}
		}
		return map[string]any{}

	case "GetTheme":
		theme := s.ensureThemeLocked(appID, environmentName, resourceID, now)
		return map[string]any{"theme": amplifyUIBuilderCloneMap(theme)}

	case "ListThemes", "ExportThemes":
		return map[string]any{"entities": s.listResourcesLocked(s.themes, appID, environmentName), "nextToken": ""}

	case "UpdateTheme":
		theme := s.ensureThemeLocked(appID, environmentName, resourceID, now)
		s.updateResourceLocked(theme, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(theme)}

	case "GetMetadata":
		return map[string]any{
			"appId":           appID,
			"environmentName": environmentName,
			"features":        amplifyUIBuilderCloneBoolMap(s.ensureMetadataBucketLocked(appID, environmentName)),
		}

	case "PutMetadataFlag":
		value := amplifyUIBuilderBool(payload, []string{"newValue", "value", "enabled"}, true)
		s.ensureMetadataFlagLocked(appID, environmentName, featureName, value)
		return map[string]any{"featureName": featureName, "newValue": value}

	case "StartCodegenJob":
		jobID := s.nextCodegenJobIdentifierLocked()
		job := s.ensureCodegenJobLocked(appID, environmentName, jobID, now)
		s.updateResourceLocked(job, payload, now)
		return map[string]any{"entity": amplifyUIBuilderCloneMap(job)}

	case "GetCodegenJob":
		job := s.ensureCodegenJobLocked(appID, environmentName, resourceID, now)
		return map[string]any{"job": amplifyUIBuilderCloneMap(job)}

	case "ListCodegenJobs":
		return map[string]any{"entities": s.listResourcesLocked(s.codegenJobs, appID, environmentName), "nextToken": ""}

	case "ExchangeCodeForToken", "RefreshToken":
		token := s.ensureTokenLocked(provider, now)
		token["issuedAt"] = now.Format(time.RFC3339)
		token["accessToken"] = fmt.Sprintf("%s-access-token", provider)
		token["refreshToken"] = fmt.Sprintf("%s-refresh-token", provider)
		return amplifyUIBuilderCloneMap(token)

	case "TagResource":
		s.mergeTagsLocked(amplifyUIBuilderTagsFromPayload(payload), resourceARN)
		return map[string]any{}

	case "UntagResource":
		for _, key := range amplifyUIBuilderTagKeys(payload, query) {
			delete(s.ensureTagsLocked(resourceARN), key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": amplifyUIBuilderCloneStringMap(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{"operation": action, "status": "SUCCEED"}
}

func (s *amplifyUIBuilderStore) applySharedNamesLocked(payload map[string]any, appID, environmentName string, now time.Time) {
	if name := amplifyUIBuilderString(payload, "name", ""); name != "" {
		s.ensureComponentLocked(appID, environmentName, name, now)
		s.ensureFormLocked(appID, environmentName, name, now)
		s.ensureThemeLocked(appID, environmentName, name, now)
	}
}

func (s *amplifyUIBuilderStore) updateResourceLocked(item map[string]any, payload map[string]any, now time.Time) {
	if item == nil {
		return
	}
	if name := amplifyUIBuilderString(payload, "name", ""); name != "" {
		item["name"] = name
	}
	if data, ok := payload["componentToUpdate"].(map[string]any); ok {
		item["componentToUpdate"] = amplifyUIBuilderCloneMap(data)
	}
	if data, ok := payload["formToUpdate"].(map[string]any); ok {
		item["formToUpdate"] = amplifyUIBuilderCloneMap(data)
	}
	if data, ok := payload["themeToUpdate"].(map[string]any); ok {
		item["themeToUpdate"] = amplifyUIBuilderCloneMap(data)
	}
	if data, ok := payload["codegenJobToCreate"].(map[string]any); ok {
		item["codegenJobToCreate"] = amplifyUIBuilderCloneMap(data)
	}
	item["modifiedAt"] = now.Format(time.RFC3339)
}

func (s *amplifyUIBuilderStore) ensureComponentLocked(appID, environmentName, id string, now time.Time) map[string]any {
	items := s.ensureResourceBucketLocked(s.components, appID, environmentName)
	if item := items[id]; item != nil {
		return item
	}
	ts := now.Format(time.RFC3339)
	item := map[string]any{
		"id":              id,
		"name":            id,
		"appId":           appID,
		"environmentName": environmentName,
		"componentType":   "Text",
		"createdAt":       ts,
		"modifiedAt":      ts,
	}
	items[id] = item
	return item
}

func (s *amplifyUIBuilderStore) ensureFormLocked(appID, environmentName, id string, now time.Time) map[string]any {
	items := s.ensureResourceBucketLocked(s.forms, appID, environmentName)
	if item := items[id]; item != nil {
		return item
	}
	ts := now.Format(time.RFC3339)
	item := map[string]any{
		"id":              id,
		"name":            id,
		"appId":           appID,
		"environmentName": environmentName,
		"createdAt":       ts,
		"modifiedAt":      ts,
	}
	items[id] = item
	return item
}

func (s *amplifyUIBuilderStore) ensureThemeLocked(appID, environmentName, id string, now time.Time) map[string]any {
	items := s.ensureResourceBucketLocked(s.themes, appID, environmentName)
	if item := items[id]; item != nil {
		return item
	}
	ts := now.Format(time.RFC3339)
	item := map[string]any{
		"id":              id,
		"name":            id,
		"appId":           appID,
		"environmentName": environmentName,
		"createdAt":       ts,
		"modifiedAt":      ts,
	}
	items[id] = item
	return item
}

func (s *amplifyUIBuilderStore) ensureCodegenJobLocked(appID, environmentName, id string, now time.Time) map[string]any {
	items := s.ensureResourceBucketLocked(s.codegenJobs, appID, environmentName)
	if item := items[id]; item != nil {
		return item
	}
	ts := now.Format(time.RFC3339)
	item := map[string]any{
		"id":              id,
		"appId":           appID,
		"environmentName": environmentName,
		"status":          "SUCCEED",
		"jobType":         "CODEGEN",
		"createdAt":       ts,
		"modifiedAt":      ts,
	}
	items[id] = item
	return item
}

func (s *amplifyUIBuilderStore) ensureResourceBucketLocked(
	resources map[string]map[string]map[string]map[string]any,
	appID, environmentName string,
) map[string]map[string]any {
	if resources[appID] == nil {
		resources[appID] = map[string]map[string]map[string]any{}
	}
	if resources[appID][environmentName] == nil {
		resources[appID][environmentName] = map[string]map[string]any{}
	}
	return resources[appID][environmentName]
}

func (s *amplifyUIBuilderStore) ensureMetadataBucketLocked(appID, environmentName string) map[string]bool {
	if s.metadataFlags[appID] == nil {
		s.metadataFlags[appID] = map[string]map[string]bool{}
	}
	if s.metadataFlags[appID][environmentName] == nil {
		s.metadataFlags[appID][environmentName] = map[string]bool{}
	}
	return s.metadataFlags[appID][environmentName]
}

func (s *amplifyUIBuilderStore) ensureMetadataFlagLocked(appID, environmentName, featureName string, value bool) {
	bucket := s.ensureMetadataBucketLocked(appID, environmentName)
	bucket[featureName] = value
}

func (s *amplifyUIBuilderStore) ensureTokenLocked(provider string, now time.Time) map[string]any {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = "figma"
	}
	if token := s.tokens[provider]; token != nil {
		return token
	}
	token := map[string]any{
		"provider":     provider,
		"accessToken":  fmt.Sprintf("%s-access-token", provider),
		"refreshToken": fmt.Sprintf("%s-refresh-token", provider),
		"tokenType":    "Bearer",
		"expiresIn":    3600,
		"issuedAt":     now.Format(time.RFC3339),
	}
	s.tokens[provider] = token
	return token
}

func (s *amplifyUIBuilderStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = "arn:aws:amplify:us-east-1:123456789012:apps/d1234567890"
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{"env": "dev"}
	s.tags[resourceARN] = tags
	return tags
}

func (s *amplifyUIBuilderStore) mergeTagsLocked(in map[string]string, resourceARN string) {
	if len(in) == 0 {
		return
	}
	tags := s.ensureTagsLocked(resourceARN)
	for key, value := range in {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		tags[key] = value
	}
}

func (s *amplifyUIBuilderStore) listResourcesLocked(resources map[string]map[string]map[string]map[string]any, appID, environmentName string) []any {
	items := make([]any, 0)
	if resources[appID] == nil || resources[appID][environmentName] == nil {
		return items
	}
	for _, item := range amplifyUIBuilderSortedNestedValues(resources[appID][environmentName]) {
		items = append(items, amplifyUIBuilderCloneMap(item))
	}
	return items
}

func (s *amplifyUIBuilderStore) nextComponentIdentifierLocked() string {
	id := fmt.Sprintf("component-%06d", s.nextComponentID)
	s.nextComponentID++
	return id
}

func (s *amplifyUIBuilderStore) nextFormIdentifierLocked() string {
	id := fmt.Sprintf("form-%06d", s.nextFormID)
	s.nextFormID++
	return id
}

func (s *amplifyUIBuilderStore) nextThemeIdentifierLocked() string {
	id := fmt.Sprintf("theme-%06d", s.nextThemeID)
	s.nextThemeID++
	return id
}

func (s *amplifyUIBuilderStore) nextCodegenJobIdentifierLocked() string {
	id := fmt.Sprintf("codegen-job-%06d", s.nextCodegenJobID)
	s.nextCodegenJobID++
	return id
}

func amplifyUIBuilderMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := amplifyUIBuilderCloneMap(payload)
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		out[key] = values[len(values)-1]
	}
	return out
}

func amplifyUIBuilderString(src map[string]any, key, def string) string {
	if src != nil {
		if value, ok := src[key]; ok && value != nil {
			if v, ok := value.(string); ok {
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return def
}

func amplifyUIBuilderBool(src map[string]any, keys []string, def bool) bool {
	for _, key := range keys {
		if src == nil {
			continue
		}
		value, ok := src[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true", "1", "yes", "enabled":
				return true
			case "false", "0", "no", "disabled":
				return false
			}
		}
	}
	return def
}

func amplifyUIBuilderTagKeys(payload map[string]any, query url.Values) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	for _, value := range query["tagKeys"] {
		for _, part := range strings.Split(value, ",") {
			add(part)
		}
	}

	for _, key := range []string{"tagKeys", "TagKeys"} {
		if payload == nil {
			continue
		}
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			for _, part := range strings.Split(typed, ",") {
				add(part)
			}
		case []any:
			for _, item := range typed {
				if s, ok := item.(string); ok {
					add(s)
				}
			}
		}
	}

	return out
}

func amplifyUIBuilderTagsFromPayload(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		if payload == nil {
			continue
		}
		value, ok := payload[key]
		if !ok || value == nil {
			continue
		}
		tags, ok := value.(map[string]any)
		if !ok {
			continue
		}
		for k, v := range tags {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return out
}

func amplifyUIBuilderCloneMap(src map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range src {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = amplifyUIBuilderCloneMap(typed)
		case []any:
			copied := make([]any, len(typed))
			for i, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					copied[i] = amplifyUIBuilderCloneMap(nested)
				} else {
					copied[i] = item
				}
			}
			out[key] = copied
		default:
			out[key] = value
		}
	}
	return out
}

func amplifyUIBuilderCloneStringMap(src map[string]string) map[string]string {
	out := make(map[string]string, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func amplifyUIBuilderCloneBoolMap(src map[string]bool) map[string]bool {
	out := make(map[string]bool, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func amplifyUIBuilderSortedNestedValues[T any](items map[string]T) []T {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]T, 0, len(keys))
	for _, key := range keys {
		out = append(out, items[key])
	}
	return out
}
