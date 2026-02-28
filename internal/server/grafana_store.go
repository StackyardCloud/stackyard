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

type grafanaStore struct {
	mu sync.Mutex

	nextWorkspaceID int64
	nextServiceAcct int64
	nextTokenID     int64

	workspaces             map[string]map[string]any
	workspaceAPIKeys       map[string]map[string]map[string]any
	workspaceServiceAccts  map[string]map[string]map[string]any
	serviceAccountTokens   map[string]map[string]map[string]map[string]any
	workspaceAuth          map[string]map[string]any
	workspaceConfig        map[string]string
	workspacePermissions   map[string][]map[string]any
	workspaceLicenseByType map[string]map[string]bool
	tags                   map[string]map[string]string
}

func newGrafanaStore() *grafanaStore {
	s := &grafanaStore{
		nextWorkspaceID:        1,
		nextServiceAcct:        1,
		nextTokenID:            1,
		workspaces:             map[string]map[string]any{},
		workspaceAPIKeys:       map[string]map[string]map[string]any{},
		workspaceServiceAccts:  map[string]map[string]map[string]any{},
		serviceAccountTokens:   map[string]map[string]map[string]map[string]any{},
		workspaceAuth:          map[string]map[string]any{},
		workspaceConfig:        map[string]string{},
		workspacePermissions:   map[string][]map[string]any{},
		workspaceLicenseByType: map[string]map[string]bool{},
		tags:                   map[string]map[string]string{},
	}

	workspace := s.ensureWorkspaceLocked("g-0000000000")
	workspaceID := grafanaDefaultStringAny(workspace, "id", "g-0000000000")
	_ = s.ensureServiceAccountLocked(workspaceID, "sa-000001")
	_ = s.ensureServiceAccountTokenLocked(workspaceID, "sa-000001", "sat-000001")
	s.tags[grafanaWorkspaceARN(workspaceID)] = map[string]string{"stackyard": "true"}
	return s
}

func (s *grafanaStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateWorkspace":
		workspaceID := fmt.Sprintf("g-%010d", s.nextWorkspaceID)
		s.nextWorkspaceID++
		workspace := s.ensureWorkspaceLocked(workspaceID)
		for k, v := range payload {
			workspace[k] = v
		}
		workspace["status"] = "ACTIVE"
		return map[string]any{"workspace": grafanaCloneAnyMap(workspace)}

	case "DescribeWorkspace":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		return map[string]any{"workspace": grafanaCloneAnyMap(s.ensureWorkspaceLocked(workspaceID))}

	case "ListWorkspaces":
		workspaceIDs := grafanaSortedKeys(s.workspaces)
		items := make([]any, 0, len(workspaceIDs))
		for _, workspaceID := range workspaceIDs {
			workspace := s.ensureWorkspaceLocked(workspaceID)
			items = append(items, map[string]any{
				"id":          workspace["id"],
				"name":        workspace["name"],
				"description": workspace["description"],
				"endpoint":    workspace["endpoint"],
				"status":      workspace["status"],
			})
		}
		return map[string]any{"workspaces": items, "nextToken": ""}

	case "UpdateWorkspace":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		workspace := s.ensureWorkspaceLocked(workspaceID)
		for k, v := range payload {
			workspace[k] = v
		}
		workspace["status"] = "UPDATING"
		workspace["modified"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"workspace": grafanaCloneAnyMap(workspace)}

	case "DeleteWorkspace":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		workspace := s.ensureWorkspaceLocked(workspaceID)
		workspace["status"] = "DELETING"
		delete(s.workspaceAPIKeys, workspaceID)
		delete(s.workspaceServiceAccts, workspaceID)
		delete(s.serviceAccountTokens, workspaceID)
		delete(s.workspaceAuth, workspaceID)
		delete(s.workspaceConfig, workspaceID)
		delete(s.workspacePermissions, workspaceID)
		delete(s.workspaceLicenseByType, workspaceID)
		return map[string]any{"workspace": grafanaCloneAnyMap(workspace)}

	case "DescribeWorkspaceAuthentication":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		return map[string]any{"authentication": grafanaCloneAnyMap(s.ensureWorkspaceAuthLocked(workspaceID))}

	case "UpdateWorkspaceAuthentication":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		auth := s.ensureWorkspaceAuthLocked(workspaceID)
		for k, v := range payload {
			auth[k] = v
		}
		return map[string]any{"authentication": grafanaCloneAnyMap(auth)}

	case "DescribeWorkspaceConfiguration":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		s.ensureWorkspaceLocked(workspaceID)
		return map[string]any{"configuration": s.workspaceConfig[workspaceID]}

	case "UpdateWorkspaceConfiguration":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		s.ensureWorkspaceLocked(workspaceID)
		config := grafanaDefaultStringAny(payload, "configuration", `{"unifiedAlerting":{"enabled":true}}`)
		s.workspaceConfig[workspaceID] = config
		return map[string]any{"configuration": config}

	case "CreateWorkspaceApiKey":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		s.ensureWorkspaceLocked(workspaceID)
		keyName := grafanaDefaultStringAny(payload, "keyName", "stackyard-key")
		ttl := int64(3600)
		if v := grafanaDefaultInt64Any(payload, "secondsToLive", 3600); v > 0 {
			ttl = v
		}
		if s.workspaceAPIKeys[workspaceID] == nil {
			s.workspaceAPIKeys[workspaceID] = map[string]map[string]any{}
		}
		key := map[string]any{
			"keyName":       keyName,
			"workspaceId":   workspaceID,
			"secondsToLive": ttl,
			"key":           fmt.Sprintf("stackyard-%s", keyName),
		}
		s.workspaceAPIKeys[workspaceID][keyName] = key
		return grafanaCloneAnyMap(key)

	case "DeleteWorkspaceApiKey":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		keyName := grafanaDefaultString(pathParams, "keyName", "stackyard-key")
		if s.workspaceAPIKeys[workspaceID] != nil {
			delete(s.workspaceAPIKeys[workspaceID], keyName)
		}
		return map[string]any{"keyName": keyName, "workspaceId": workspaceID}

	case "CreateWorkspaceServiceAccount":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		serviceAccountID := fmt.Sprintf("sa-%06d", s.nextServiceAcct)
		s.nextServiceAcct++
		serviceAccount := s.ensureServiceAccountLocked(workspaceID, serviceAccountID)
		for k, v := range payload {
			serviceAccount[k] = v
		}
		return map[string]any{"serviceAccount": grafanaCloneAnyMap(serviceAccount)}

	case "DeleteWorkspaceServiceAccount":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		serviceAccountID := grafanaDefaultString(pathParams, "serviceAccountId", "sa-000001")
		if s.workspaceServiceAccts[workspaceID] != nil {
			delete(s.workspaceServiceAccts[workspaceID], serviceAccountID)
		}
		if s.serviceAccountTokens[workspaceID] != nil {
			delete(s.serviceAccountTokens[workspaceID], serviceAccountID)
		}
		return map[string]any{}

	case "ListWorkspaceServiceAccounts":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		_ = s.ensureServiceAccountLocked(workspaceID, "sa-000001")
		serviceAccountIDs := grafanaSortedKeys(s.workspaceServiceAccts[workspaceID])
		items := make([]any, 0, len(serviceAccountIDs))
		for _, serviceAccountID := range serviceAccountIDs {
			items = append(items, grafanaCloneAnyMap(s.workspaceServiceAccts[workspaceID][serviceAccountID]))
		}
		return map[string]any{"serviceAccounts": items, "nextToken": ""}

	case "CreateWorkspaceServiceAccountToken":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		serviceAccountID := grafanaDefaultString(pathParams, "serviceAccountId", "sa-000001")
		tokenID := fmt.Sprintf("sat-%06d", s.nextTokenID)
		s.nextTokenID++
		token := s.ensureServiceAccountTokenLocked(workspaceID, serviceAccountID, tokenID)
		for k, v := range payload {
			token[k] = v
		}
		return map[string]any{"serviceAccountToken": grafanaCloneAnyMap(token)}

	case "DeleteWorkspaceServiceAccountToken":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		serviceAccountID := grafanaDefaultString(pathParams, "serviceAccountId", "sa-000001")
		tokenID := grafanaDefaultString(pathParams, "tokenId", "sat-000001")
		if s.serviceAccountTokens[workspaceID] != nil && s.serviceAccountTokens[workspaceID][serviceAccountID] != nil {
			delete(s.serviceAccountTokens[workspaceID][serviceAccountID], tokenID)
		}
		return map[string]any{}

	case "ListWorkspaceServiceAccountTokens":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		serviceAccountID := grafanaDefaultString(pathParams, "serviceAccountId", "sa-000001")
		_ = s.ensureServiceAccountTokenLocked(workspaceID, serviceAccountID, "sat-000001")
		tokenIDs := grafanaSortedKeys(s.serviceAccountTokens[workspaceID][serviceAccountID])
		items := make([]any, 0, len(tokenIDs))
		for _, tokenID := range tokenIDs {
			items = append(items, grafanaCloneAnyMap(s.serviceAccountTokens[workspaceID][serviceAccountID][tokenID]))
		}
		return map[string]any{"serviceAccountTokens": items, "nextToken": ""}

	case "ListPermissions":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		_ = s.ensureWorkspacePermissionsLocked(workspaceID)
		items := make([]any, 0, len(s.workspacePermissions[workspaceID]))
		for _, entry := range s.workspacePermissions[workspaceID] {
			items = append(items, grafanaCloneAnyMap(entry))
		}
		return map[string]any{"permissions": items, "nextToken": ""}

	case "UpdatePermissions":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		if usersRaw, ok := payload["users"].([]any); ok && len(usersRaw) > 0 {
			users := make([]string, 0, len(usersRaw))
			for _, raw := range usersRaw {
				users = append(users, strings.TrimSpace(fmt.Sprintf("%v", raw)))
			}
			s.workspacePermissions[workspaceID] = []map[string]any{{
				"action": "MANAGE",
				"role":   "ADMIN",
				"users":  users,
			}}
		} else {
			_ = s.ensureWorkspacePermissionsLocked(workspaceID)
		}
		return map[string]any{"errors": []any{}}

	case "AssociateLicense":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		licenseType := grafanaDefaultString(pathParams, "licenseType", "ENTERPRISE")
		if s.workspaceLicenseByType[workspaceID] == nil {
			s.workspaceLicenseByType[workspaceID] = map[string]bool{}
		}
		s.workspaceLicenseByType[workspaceID][licenseType] = true
		return map[string]any{}

	case "DisassociateLicense":
		workspaceID := grafanaDefaultString(pathParams, "workspaceId", "g-0000000000")
		licenseType := grafanaDefaultString(pathParams, "licenseType", "ENTERPRISE")
		if s.workspaceLicenseByType[workspaceID] != nil {
			delete(s.workspaceLicenseByType[workspaceID], licenseType)
		}
		return map[string]any{}

	case "ListVersions":
		return map[string]any{"grafanaVersions": []any{
			map[string]any{"version": "9.4", "status": "CURRENT"},
			map[string]any{"version": "10.4", "status": "AVAILABLE"},
		}}

	case "ListTagsForResource":
		resourceARN := grafanaDefaultString(pathParams, "resourceArn", grafanaWorkspaceARN("g-0000000000"))
		return map[string]any{"tags": grafanaCloneStringMap(s.tags[resourceARN])}

	case "TagResource":
		resourceARN := grafanaDefaultString(pathParams, "resourceArn", grafanaWorkspaceARN("g-0000000000"))
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		switch tags := payload["tags"].(type) {
		case map[string]any:
			for k, v := range tags {
				s.tags[resourceARN][k] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		case map[string]string:
			for k, v := range tags {
				s.tags[resourceARN][k] = v
			}
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := grafanaDefaultString(pathParams, "resourceArn", grafanaWorkspaceARN("g-0000000000"))
		if keys, ok := payload["tagKeys"].([]any); ok {
			for _, raw := range keys {
				delete(s.tags[resourceARN], strings.TrimSpace(fmt.Sprintf("%v", raw)))
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *grafanaStore) ensureWorkspaceLocked(workspaceID string) map[string]any {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "g-0000000000"
	}
	if existing := s.workspaces[id]; existing != nil {
		return existing
	}

	item := map[string]any{
		"id":             id,
		"name":           "stackyard-workspace",
		"description":    "Stackyard Managed Grafana workspace",
		"endpoint":       fmt.Sprintf("https://%s.grafana.amazonaws.com", id),
		"status":         "ACTIVE",
		"created":        time.Now().UTC().Format(time.RFC3339),
		"modified":       time.Now().UTC().Format(time.RFC3339),
		"arn":            grafanaWorkspaceARN(id),
		"grafanaVersion": "10.4",
	}
	s.workspaces[id] = item

	s.ensureWorkspaceAuthLocked(id)
	if s.workspaceConfig[id] == "" {
		s.workspaceConfig[id] = `{"unifiedAlerting":{"enabled":true}}`
	}
	s.ensureWorkspacePermissionsLocked(id)
	if s.tags[grafanaWorkspaceARN(id)] == nil {
		s.tags[grafanaWorkspaceARN(id)] = map[string]string{"stackyard": "true"}
	}
	return item
}

func (s *grafanaStore) ensureWorkspaceAuthLocked(workspaceID string) map[string]any {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "g-0000000000"
	}
	if s.workspaceAuth[id] != nil {
		return s.workspaceAuth[id]
	}
	auth := map[string]any{
		"providers":               []any{"AWS_SSO"},
		"samlConfigurationStatus": "NOT_CONFIGURED",
	}
	s.workspaceAuth[id] = auth
	return auth
}

func (s *grafanaStore) ensureWorkspacePermissionsLocked(workspaceID string) []map[string]any {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "g-0000000000"
	}
	if s.workspacePermissions[id] != nil {
		return s.workspacePermissions[id]
	}
	s.workspacePermissions[id] = []map[string]any{
		{
			"action": "MANAGE",
			"role":   "ADMIN",
			"users":  []any{"stackyard-user"},
		},
	}
	return s.workspacePermissions[id]
}

func (s *grafanaStore) ensureServiceAccountLocked(workspaceID, serviceAccountID string) map[string]any {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = "g-0000000000"
	}
	serviceAccountID = strings.TrimSpace(serviceAccountID)
	if serviceAccountID == "" {
		serviceAccountID = "sa-000001"
	}
	s.ensureWorkspaceLocked(workspaceID)
	if s.workspaceServiceAccts[workspaceID] == nil {
		s.workspaceServiceAccts[workspaceID] = map[string]map[string]any{}
	}
	if existing := s.workspaceServiceAccts[workspaceID][serviceAccountID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":          serviceAccountID,
		"name":        "stackyard-service-account",
		"workspaceId": workspaceID,
		"isDisabled":  false,
	}
	s.workspaceServiceAccts[workspaceID][serviceAccountID] = item
	return item
}

func (s *grafanaStore) ensureServiceAccountTokenLocked(workspaceID, serviceAccountID, tokenID string) map[string]any {
	_ = s.ensureServiceAccountLocked(workspaceID, serviceAccountID)
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		tokenID = "sat-000001"
	}
	if s.serviceAccountTokens[workspaceID] == nil {
		s.serviceAccountTokens[workspaceID] = map[string]map[string]map[string]any{}
	}
	if s.serviceAccountTokens[workspaceID][serviceAccountID] == nil {
		s.serviceAccountTokens[workspaceID][serviceAccountID] = map[string]map[string]any{}
	}
	if existing := s.serviceAccountTokens[workspaceID][serviceAccountID][tokenID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":               tokenID,
		"name":             "stackyard-token",
		"serviceAccountId": serviceAccountID,
		"workspaceId":      workspaceID,
		"key":              fmt.Sprintf("stackyard-token-%s", tokenID),
	}
	s.serviceAccountTokens[workspaceID][serviceAccountID][tokenID] = item
	return item
}

func grafanaWorkspaceARN(workspaceID string) string {
	id := strings.TrimSpace(workspaceID)
	if id == "" {
		id = "g-0000000000"
	}
	return fmt.Sprintf("arn:aws:grafana:us-east-1:123456789012:/workspaces/%s", id)
}

func grafanaDefaultString(values map[string]string, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return fallback
}

func grafanaDefaultStringAny(values map[string]any, key, fallback string) string {
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if trimmed := strings.TrimSpace(fmt.Sprintf("%v", v)); trimmed != "" && trimmed != "<nil>" {
				return trimmed
			}
		}
	}
	return fallback
}

func grafanaDefaultInt64Any(values map[string]any, key string, fallback int64) int64 {
	for k, v := range values {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		switch n := v.(type) {
		case int:
			return int64(n)
		case int32:
			return int64(n)
		case int64:
			return n
		case float64:
			return int64(n)
		case float32:
			return int64(n)
		default:
			asString := strings.TrimSpace(fmt.Sprintf("%v", v))
			if asString == "" {
				break
			}
			parsed, err := strconv.ParseInt(asString, 10, 64)
			if err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func grafanaSortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func grafanaCloneAnyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func grafanaCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
