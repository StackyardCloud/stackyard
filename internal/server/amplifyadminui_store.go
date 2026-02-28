package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type amplifyAdminUIStore struct {
	mu sync.Mutex

	nextSessionID int64
	nextJobID     int64

	backends map[string]map[string]map[string]any
	apis     map[string]map[string]map[string]any
	auths    map[string]map[string]map[string]any
	storages map[string]map[string]map[string]any
	configs  map[string]map[string]any
	jobs     map[string]map[string]map[string]map[string]any
	tokens   map[string]map[string]any
}

func newAmplifyAdminUIStore() *amplifyAdminUIStore {
	s := &amplifyAdminUIStore{
		nextSessionID: 2,
		nextJobID:     2,
		backends:      map[string]map[string]map[string]any{},
		apis:          map[string]map[string]map[string]any{},
		auths:         map[string]map[string]map[string]any{},
		storages:      map[string]map[string]map[string]any{},
		configs:       map[string]map[string]any{},
		jobs:          map[string]map[string]map[string]map[string]any{},
		tokens:        map[string]map[string]any{},
	}
	now := time.Now().UTC()
	s.ensureBackendLocked("d1234567890", "dev", now)
	s.ensureBackendConfigLocked("d1234567890", now)
	s.ensureBackendAPIResourceLocked("d1234567890", "dev", now)
	s.ensureBackendAuthResourceLocked("d1234567890", "dev", now)
	s.ensureBackendStorageResourceLocked("d1234567890", "dev", now)
	s.ensureBackendJobLocked("d1234567890", "dev", "job-000001", now)
	s.ensureTokenLocked("d1234567890", "session-000001", now)
	return s
}

func (s *amplifyAdminUIStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := amplifyAdminUIMergeMaps(payload, pathParams, query)
	appID := amplifyAdminUIString(ctx, "appId", "d1234567890")
	backendEnvironmentName := amplifyAdminUIString(ctx, "backendEnvironmentName", "dev")
	jobID := amplifyAdminUIString(ctx, "jobId", "job-000001")
	sessionID := amplifyAdminUIString(ctx, "sessionId", "session-000001")
	appName := amplifyAdminUIString(payload, "appName", "stackyard-amplify-adminui-app")

	s.ensureBackendLocked(appID, backendEnvironmentName, now)
	s.ensureBackendConfigLocked(appID, now)
	s.ensureBackendAPIResourceLocked(appID, backendEnvironmentName, now)
	s.ensureBackendAuthResourceLocked(appID, backendEnvironmentName, now)
	s.ensureBackendStorageResourceLocked(appID, backendEnvironmentName, now)
	s.ensureBackendJobLocked(appID, backendEnvironmentName, jobID, now)
	s.ensureTokenLocked(appID, sessionID, now)

	switch action {
	case "CloneBackend":
		s.ensureBackendLocked(appID, backendEnvironmentName+"-clone", now)
		return s.backendOpResponse(appID, backendEnvironmentName+"-clone", "CLONE_BACKEND", now)

	case "CreateBackend":
		if v := amplifyAdminUIString(payload, "appId", ""); v != "" {
			appID = v
		}
		if v := amplifyAdminUIString(payload, "backendEnvironmentName", ""); v != "" {
			backendEnvironmentName = v
		}
		s.ensureBackendLocked(appID, backendEnvironmentName, now)
		return s.backendOpResponse(appID, backendEnvironmentName, "CREATE_BACKEND", now)

	case "DeleteBackend":
		if envs := s.backends[appID]; envs != nil {
			delete(envs, backendEnvironmentName)
		}
		return s.backendOpResponse(appID, backendEnvironmentName, "DELETE_BACKEND", now)

	case "GetBackend":
		backend := s.ensureBackendLocked(appID, backendEnvironmentName, now)
		return map[string]any{
			"appId":                  appID,
			"appName":                appName,
			"backendEnvironmentList": []any{amplifyAdminUICloneMap(backend)},
			"error":                  "",
		}

	case "CreateBackendConfig":
		cfg := s.ensureBackendConfigLocked(appID, now)
		cfg["updateTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"appId":          appID,
			"backendManager": "AMPLIFY_CONSOLE",
			"error":          "",
		}

	case "RemoveBackendConfig":
		delete(s.configs, appID)
		return map[string]any{
			"appId": appID,
			"error": "",
		}

	case "UpdateBackendConfig":
		cfg := s.ensureBackendConfigLocked(appID, now)
		cfg["updateTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"appId": appID,
			"error": "",
		}

	case "CreateBackendAPI":
		api := s.ensureBackendAPIResourceLocked(appID, backendEnvironmentName, now)
		api["updateTime"] = now.Format(time.RFC3339)
		return s.backendOpResponse(appID, backendEnvironmentName, "CREATE_BACKEND_API", now)

	case "DeleteBackendAPI":
		if appAPIs := s.apis[appID]; appAPIs != nil {
			delete(appAPIs, backendEnvironmentName)
		}
		return s.backendOpResponse(appID, backendEnvironmentName, "DELETE_BACKEND_API", now)

	case "GenerateBackendAPIModels":
		return s.backendOpResponse(appID, backendEnvironmentName, "GENERATE_BACKEND_API_MODELS", now)

	case "GetBackendAPI":
		api := s.ensureBackendAPIResourceLocked(appID, backendEnvironmentName, now)
		return map[string]any{
			"appId":                  appID,
			"backendEnvironmentName": backendEnvironmentName,
			"resourceConfig":         amplifyAdminUICloneMap(api),
			"resourceName":           "api",
			"status":                 "SUCCEED",
		}

	case "GetBackendAPIModels":
		return map[string]any{
			"models":                 "{}",
			"status":                 "SUCCEED",
			"backendEnvironmentName": backendEnvironmentName,
		}

	case "UpdateBackendAPI":
		api := s.ensureBackendAPIResourceLocked(appID, backendEnvironmentName, now)
		api["updateTime"] = now.Format(time.RFC3339)
		return s.backendOpResponse(appID, backendEnvironmentName, "UPDATE_BACKEND_API", now)

	case "CreateBackendAuth":
		auth := s.ensureBackendAuthResourceLocked(appID, backendEnvironmentName, now)
		auth["updateTime"] = now.Format(time.RFC3339)
		return s.backendOpResponse(appID, backendEnvironmentName, "CREATE_BACKEND_AUTH", now)

	case "DeleteBackendAuth":
		if appAuth := s.auths[appID]; appAuth != nil {
			delete(appAuth, backendEnvironmentName)
		}
		return s.backendOpResponse(appID, backendEnvironmentName, "DELETE_BACKEND_AUTH", now)

	case "GetBackendAuth":
		auth := s.ensureBackendAuthResourceLocked(appID, backendEnvironmentName, now)
		return map[string]any{
			"appId":                  appID,
			"backendEnvironmentName": backendEnvironmentName,
			"resourceName":           "auth",
			"resourceConfig":         amplifyAdminUICloneMap(auth),
		}

	case "ImportBackendAuth":
		_ = s.ensureBackendAuthResourceLocked(appID, backendEnvironmentName, now)
		return s.backendOpResponse(appID, backendEnvironmentName, "IMPORT_BACKEND_AUTH", now)

	case "UpdateBackendAuth":
		auth := s.ensureBackendAuthResourceLocked(appID, backendEnvironmentName, now)
		auth["updateTime"] = now.Format(time.RFC3339)
		return s.backendOpResponse(appID, backendEnvironmentName, "UPDATE_BACKEND_AUTH", now)

	case "CreateBackendStorage":
		storage := s.ensureBackendStorageResourceLocked(appID, backendEnvironmentName, now)
		storage["updateTime"] = now.Format(time.RFC3339)
		return s.backendOpResponse(appID, backendEnvironmentName, "CREATE_BACKEND_STORAGE", now)

	case "DeleteBackendStorage":
		if appStorage := s.storages[appID]; appStorage != nil {
			delete(appStorage, backendEnvironmentName)
		}
		return s.backendOpResponse(appID, backendEnvironmentName, "DELETE_BACKEND_STORAGE", now)

	case "GetBackendStorage":
		storage := s.ensureBackendStorageResourceLocked(appID, backendEnvironmentName, now)
		return map[string]any{
			"appId":                  appID,
			"backendEnvironmentName": backendEnvironmentName,
			"resourceName":           "storage",
			"resourceConfig":         amplifyAdminUICloneMap(storage),
		}

	case "ImportBackendStorage":
		_ = s.ensureBackendStorageResourceLocked(appID, backendEnvironmentName, now)
		return s.backendOpResponse(appID, backendEnvironmentName, "IMPORT_BACKEND_STORAGE", now)

	case "UpdateBackendStorage":
		storage := s.ensureBackendStorageResourceLocked(appID, backendEnvironmentName, now)
		storage["updateTime"] = now.Format(time.RFC3339)
		return s.backendOpResponse(appID, backendEnvironmentName, "UPDATE_BACKEND_STORAGE", now)

	case "CreateToken":
		if sessionID == "session-000001" {
			sessionID = fmt.Sprintf("session-%06d", s.nextSessionID)
			s.nextSessionID++
		}
		token := s.ensureTokenLocked(appID, sessionID, now)
		return amplifyAdminUICloneMap(token)

	case "DeleteToken":
		delete(s.tokens, s.tokenKey(appID, sessionID))
		return map[string]any{
			"isSuccess": true,
		}

	case "GetToken":
		token := s.ensureTokenLocked(appID, sessionID, now)
		return amplifyAdminUICloneMap(token)

	case "GetBackendJob":
		job := s.ensureBackendJobLocked(appID, backendEnvironmentName, jobID, now)
		return map[string]any{
			"appId":                  appID,
			"backendEnvironmentName": backendEnvironmentName,
			"job":                    amplifyAdminUICloneMap(job),
		}

	case "ListBackendJobs":
		items := make([]any, 0)
		if appJobs := s.jobs[appID]; appJobs != nil {
			for _, job := range amplifyAdminUISortedNestedValues(appJobs[backendEnvironmentName]) {
				items = append(items, amplifyAdminUICloneMap(job))
			}
		}
		return map[string]any{
			"jobs":      items,
			"nextToken": "",
		}

	case "RemoveAllBackends":
		delete(s.backends, appID)
		delete(s.apis, appID)
		delete(s.auths, appID)
		delete(s.storages, appID)
		delete(s.jobs, appID)
		return s.backendOpResponse(appID, backendEnvironmentName, "REMOVE_ALL_BACKENDS", now)

	case "UpdateBackendJob":
		job := s.ensureBackendJobLocked(appID, backendEnvironmentName, jobID, now)
		job["status"] = "SUCCEED"
		job["updateTime"] = now.Format(time.RFC3339)
		return map[string]any{
			"appId":                  appID,
			"backendEnvironmentName": backendEnvironmentName,
			"jobId":                  jobID,
			"operation":              "UPDATE_BACKEND_JOB",
			"status":                 "SUCCEED",
		}

	case "ListS3Buckets":
		return map[string]any{
			"buckets": []any{
				map[string]any{"name": "stackyard-amplify-adminui-assets"},
				map[string]any{"name": "stackyard-amplify-adminui-storage"},
			},
			"nextToken": "",
		}
	}

	return map[string]any{
		"operation": action,
		"status":    "SUCCEED",
	}
}

func (s *amplifyAdminUIStore) backendOpResponse(appID, backendEnvironmentName, operation string, now time.Time) map[string]any {
	jobID := fmt.Sprintf("job-%06d", s.nextJobID)
	s.nextJobID++
	s.ensureBackendJobLocked(appID, backendEnvironmentName, jobID, now)
	return map[string]any{
		"appId":                  appID,
		"backendEnvironmentName": backendEnvironmentName,
		"jobId":                  jobID,
		"operation":              operation,
		"status":                 "SUCCEED",
	}
}

func (s *amplifyAdminUIStore) ensureBackendLocked(appID, backendEnvironmentName string, now time.Time) map[string]any {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = "d1234567890"
	}
	backendEnvironmentName = strings.TrimSpace(backendEnvironmentName)
	if backendEnvironmentName == "" {
		backendEnvironmentName = "dev"
	}
	if s.backends[appID] == nil {
		s.backends[appID] = map[string]map[string]any{}
	}
	if backend := s.backends[appID][backendEnvironmentName]; backend != nil {
		return backend
	}
	timestamp := now.Format(time.RFC3339)
	backend := map[string]any{
		"appId":                  appID,
		"backendEnvironmentName": backendEnvironmentName,
		"createTime":             timestamp,
		"updateTime":             timestamp,
		"stackName":              fmt.Sprintf("amplify-%s-%s", appID, backendEnvironmentName),
	}
	s.backends[appID][backendEnvironmentName] = backend
	return backend
}

func (s *amplifyAdminUIStore) ensureBackendConfigLocked(appID string, now time.Time) map[string]any {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = "d1234567890"
	}
	if cfg := s.configs[appID]; cfg != nil {
		return cfg
	}
	timestamp := now.Format(time.RFC3339)
	cfg := map[string]any{
		"appId":          appID,
		"backendManager": "AMPLIFY_CONSOLE",
		"updateTime":     timestamp,
	}
	s.configs[appID] = cfg
	return cfg
}

func (s *amplifyAdminUIStore) ensureBackendAPIResourceLocked(appID, backendEnvironmentName string, now time.Time) map[string]any {
	if s.apis[appID] == nil {
		s.apis[appID] = map[string]map[string]any{}
	}
	if api := s.apis[appID][backendEnvironmentName]; api != nil {
		return api
	}
	timestamp := now.Format(time.RFC3339)
	api := map[string]any{
		"service":    "AppSync",
		"apiName":    "stackyardapi",
		"updateTime": timestamp,
	}
	s.apis[appID][backendEnvironmentName] = api
	return api
}

func (s *amplifyAdminUIStore) ensureBackendAuthResourceLocked(appID, backendEnvironmentName string, now time.Time) map[string]any {
	if s.auths[appID] == nil {
		s.auths[appID] = map[string]map[string]any{}
	}
	if auth := s.auths[appID][backendEnvironmentName]; auth != nil {
		return auth
	}
	timestamp := now.Format(time.RFC3339)
	auth := map[string]any{
		"service":    "Cognito",
		"updateTime": timestamp,
		"userPoolId": "us-east-1_stackyard",
	}
	s.auths[appID][backendEnvironmentName] = auth
	return auth
}

func (s *amplifyAdminUIStore) ensureBackendStorageResourceLocked(appID, backendEnvironmentName string, now time.Time) map[string]any {
	if s.storages[appID] == nil {
		s.storages[appID] = map[string]map[string]any{}
	}
	if storage := s.storages[appID][backendEnvironmentName]; storage != nil {
		return storage
	}
	timestamp := now.Format(time.RFC3339)
	storage := map[string]any{
		"service":    "S3",
		"bucketName": "stackyard-amplify-adminui-storage",
		"updateTime": timestamp,
	}
	s.storages[appID][backendEnvironmentName] = storage
	return storage
}

func (s *amplifyAdminUIStore) ensureBackendJobLocked(appID, backendEnvironmentName, jobID string, now time.Time) map[string]any {
	if s.jobs[appID] == nil {
		s.jobs[appID] = map[string]map[string]map[string]any{}
	}
	if s.jobs[appID][backendEnvironmentName] == nil {
		s.jobs[appID][backendEnvironmentName] = map[string]map[string]any{}
	}
	if job := s.jobs[appID][backendEnvironmentName][jobID]; job != nil {
		return job
	}
	timestamp := now.Format(time.RFC3339)
	job := map[string]any{
		"jobId":      jobID,
		"status":     "SUCCEED",
		"operation":  "DEPLOY",
		"createTime": timestamp,
		"updateTime": timestamp,
	}
	s.jobs[appID][backendEnvironmentName][jobID] = job
	return job
}

func (s *amplifyAdminUIStore) tokenKey(appID, sessionID string) string {
	return strings.TrimSpace(appID) + "|" + strings.TrimSpace(sessionID)
}

func (s *amplifyAdminUIStore) ensureTokenLocked(appID, sessionID string, now time.Time) map[string]any {
	key := s.tokenKey(appID, sessionID)
	if token := s.tokens[key]; token != nil {
		return token
	}
	token := map[string]any{
		"appId":         appID,
		"sessionId":     sessionID,
		"challengeCode": "123456",
		"ttl":           300,
		"isValid":       true,
		"createTime":    now.Format(time.RFC3339),
	}
	s.tokens[key] = token
	return token
}

func amplifyAdminUIMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := amplifyAdminUICloneMap(payload)
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

func amplifyAdminUIString(src map[string]any, key, def string) string {
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

func amplifyAdminUISortedNestedValues[T any](items map[string]T) []T {
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

func amplifyAdminUICloneMap(src map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range src {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = amplifyAdminUICloneMap(typed)
		case []any:
			copied := make([]any, len(typed))
			for i, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					copied[i] = amplifyAdminUICloneMap(nested)
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
