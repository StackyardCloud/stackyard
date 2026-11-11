package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azurePlaywrightDataPlaneAPIVersion = "2025-09-01"
const azurePlaywrightDataPlanePrefix = "/azure/playwright-data-plane/2025-09-01"

var azurePlaywrightDataPlaneGUIDPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var azurePlaywrightDataPlaneTokenIDPattern = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)+$`)
var azurePlaywrightDataPlaneTokenNamePattern = regexp.MustCompile(`^[a-zA-Z0-9-]{3,24}$`)

type azurePlaywrightDataPlaneWorkspace struct {
	WorkspaceID  string
	DataplaneURI string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type azurePlaywrightDataPlaneAccessToken struct {
	WorkspaceID   string
	AccessTokenID string
	Name          string
	JWTToken      string
	CreatedAt     time.Time
	ExpiryAt      time.Time
}

type azurePlaywrightDataPlaneTestRun struct {
	WorkspaceID string
	RunID       string
	DisplayName string
	Config      map[string]any
	CIConfig    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Server) handleAzurePlaywrightDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azurePlaywrightDataPlanePrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azurePlaywrightDataPlaneAPIVersion {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2025-09-01")
		return true
	}

	relative := strings.TrimPrefix(path, azurePlaywrightDataPlanePrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) < 2 || !strings.EqualFold(segments[0], "playwrightworkspaces") {
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	workspaceID := strings.TrimSpace(segments[1])
	if !azurePlaywrightDataPlaneValidGUID(workspaceID) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidWorkspaceId", "workspaceId is invalid")
		return true
	}

	tail := segments[2:]
	if len(tail) == 0 {
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "access-tokens":
		s.handleAzurePlaywrightDataPlaneAccessTokens(w, r, workspaceID, tail[1:])
		return true
	case "test-runs":
		s.handleAzurePlaywrightDataPlaneTestRuns(w, r, workspaceID, tail[1:])
		return true
	case "browsers":
		s.handleAzurePlaywrightDataPlaneBrowsers(w, r, workspaceID, tail[1:])
		return true
	default:
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzurePlaywrightDataPlaneAccessTokens(w http.ResponseWriter, r *http.Request, workspaceID string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzurePlaywrightDataPlaneListAccessTokens(w, r, workspaceID)
	case len(tail) == 1 && r.Method == http.MethodPut:
		s.handleAzurePlaywrightDataPlanePutAccessToken(w, r, workspaceID, tail[0])
	case len(tail) == 1 && r.Method == http.MethodGet:
		s.handleAzurePlaywrightDataPlaneGetAccessToken(w, workspaceID, tail[0])
	case len(tail) == 1 && r.Method == http.MethodDelete:
		s.handleAzurePlaywrightDataPlaneDeleteAccessToken(w, workspaceID, tail[0])
	case len(tail) == 0 || len(tail) == 1:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzurePlaywrightDataPlaneTestRuns(w http.ResponseWriter, r *http.Request, workspaceID string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzurePlaywrightDataPlaneListTestRuns(w, r, workspaceID)
	case len(tail) == 1 && r.Method == http.MethodPatch:
		s.handleAzurePlaywrightDataPlanePatchTestRun(w, r, workspaceID, tail[0])
	case len(tail) == 0 || len(tail) == 1:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzurePlaywrightDataPlaneBrowsers(w http.ResponseWriter, r *http.Request, workspaceID string, tail []string) {
	if len(tail) != 0 {
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	osName := strings.TrimSpace(r.URL.Query().Get("os"))
	if osName == "" {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "MissingRequiredParameter", "os is required")
		return
	}
	runID := strings.TrimSpace(r.URL.Query().Get("runId"))
	if runID != "" && !azurePlaywrightDataPlaneValidGUID(runID) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRunId", "runId is invalid")
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("stackyard_redirect")) == "1" {
		if clientRequestID := strings.TrimSpace(r.Header.Get("x-ms-client-request-id")); clientRequestID != "" {
			w.Header().Set("x-ms-client-request-id", clientRequestID)
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"workspaceId":           workspaceID,
			"os":                    osName,
			"runId":                 runID,
			"browserConnectionUrl":  "wss://stackyard.local/playwright/" + workspaceID,
			"playwrightEndpointUrl": "wss://stackyard.local/playwright/" + workspaceID + "?os=" + osName,
		})
		return
	}

	s.providerStorageMu.Lock()
	workspace := s.azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID)
	if runID != "" {
		if s.azurePlaywrightDataPlaneTestRuns[azurePlaywrightDataPlaneRunKey(workspaceID, runID)] == nil {
			s.providerStorageMu.Unlock()
			azurePlaywrightDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
			return
		}
	}
	s.providerStorageMu.Unlock()

	location := azurePlaywrightDataPlanePrefix + "/playwrightworkspaces/" + workspaceID + "/browsers?api-version=" + azurePlaywrightDataPlaneAPIVersion + "&stackyard_redirect=1&os=" + osName
	if runID != "" {
		location += "&runId=" + runID
	}
	if strings.TrimSpace(workspace.DataplaneURI) != "" {
		w.Header().Set("x-ms-playwright-dataplane-uri", workspace.DataplaneURI)
	}
	if clientRequestID := strings.TrimSpace(r.Header.Get("x-ms-client-request-id")); clientRequestID != "" {
		w.Header().Set("x-ms-client-request-id", clientRequestID)
	}
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusFound)
}

func (s *Server) handleAzurePlaywrightDataPlanePutAccessToken(w http.ResponseWriter, r *http.Request, workspaceID, accessTokenID string) {
	if !azurePlaywrightDataPlaneValidAccessTokenID(accessTokenID) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidAccessTokenId", "accessTokenId is invalid")
		return
	}
	body, err := readBodyBytes(r)
	if err != nil {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return
	}
	name := strings.TrimSpace(azurePlaywrightDataPlaneString(payload["name"]))
	if !azurePlaywrightDataPlaneValidTokenName(name) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidTokenName", "name must match ^[a-zA-Z0-9-]{3,24}$")
		return
	}
	expiryAt, err := time.Parse(time.RFC3339, strings.TrimSpace(azurePlaywrightDataPlaneString(payload["expiryAt"])))
	if err != nil {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "MissingRequiredParameter", "expiryAt must be a valid RFC3339 timestamp")
		return
	}

	now := time.Now().UTC()
	key := azurePlaywrightDataPlaneAccessTokenKey(workspaceID, accessTokenID)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	_ = s.azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID)
	status := http.StatusOK
	createdAt := now
	if existing := s.azurePlaywrightDataPlaneAccessTokens[key]; existing != nil {
		createdAt = existing.CreatedAt
	} else {
		status = http.StatusCreated
	}

	s.azurePlaywrightDataPlaneAccessTokens[key] = &azurePlaywrightDataPlaneAccessToken{
		WorkspaceID:   workspaceID,
		AccessTokenID: accessTokenID,
		Name:          name,
		JWTToken:      azurePlaywrightDataPlaneJWT(workspaceID, accessTokenID),
		CreatedAt:     createdAt,
		ExpiryAt:      expiryAt.UTC(),
	}
	respondJSON(w, status, azurePlaywrightDataPlaneAccessTokenFixture(s.azurePlaywrightDataPlaneAccessTokens[key]))
}

func (s *Server) handleAzurePlaywrightDataPlaneGetAccessToken(w http.ResponseWriter, workspaceID, accessTokenID string) {
	if !azurePlaywrightDataPlaneValidAccessTokenID(accessTokenID) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidAccessTokenId", "accessTokenId is invalid")
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	_ = s.azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID)
	token := s.azurePlaywrightDataPlaneAccessTokens[azurePlaywrightDataPlaneAccessTokenKey(workspaceID, accessTokenID)]
	if token == nil {
		azurePlaywrightDataPlaneError(w, http.StatusNotFound, "AccessTokenNotFound", "access token not found")
		return
	}
	respondJSON(w, http.StatusOK, azurePlaywrightDataPlaneAccessTokenFixture(token))
}

func (s *Server) handleAzurePlaywrightDataPlaneDeleteAccessToken(w http.ResponseWriter, workspaceID, accessTokenID string) {
	if !azurePlaywrightDataPlaneValidAccessTokenID(accessTokenID) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidAccessTokenId", "accessTokenId is invalid")
		return
	}
	s.providerStorageMu.Lock()
	delete(s.azurePlaywrightDataPlaneAccessTokens, azurePlaywrightDataPlaneAccessTokenKey(workspaceID, accessTokenID))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzurePlaywrightDataPlaneListAccessTokens(w http.ResponseWriter, r *http.Request, workspaceID string) {
	top, skip, ok := azurePlaywrightDataPlanePagination(w, r, 10)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	_ = s.azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID)
	items := make([]*azurePlaywrightDataPlaneAccessToken, 0)
	for _, token := range s.azurePlaywrightDataPlaneAccessTokens {
		if token.WorkspaceID == workspaceID {
			items = append(items, token)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].AccessTokenID < items[j].AccessTokenID })

	value, nextLink := azurePlaywrightDataPlaneAccessTokenPage(items, workspaceID, top, skip)
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) handleAzurePlaywrightDataPlanePatchTestRun(w http.ResponseWriter, r *http.Request, workspaceID, runID string) {
	if !azurePlaywrightDataPlaneValidGUID(runID) {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRunId", "runId is invalid")
		return
	}
	body, err := readBodyBytes(r)
	if err != nil {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return
	}

	now := time.Now().UTC()
	key := azurePlaywrightDataPlaneRunKey(workspaceID, runID)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	_ = s.azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID)
	existing := s.azurePlaywrightDataPlaneTestRuns[key]
	status := http.StatusOK
	createdAt := now
	config := map[string]any{}
	ciConfig := map[string]any{}
	displayName := strings.TrimSpace(azurePlaywrightDataPlaneString(payload["displayName"]))
	if existing != nil {
		status = http.StatusOK
		createdAt = existing.CreatedAt
		config = azurePlaywrightDataPlaneCloneObject(existing.Config)
		ciConfig = azurePlaywrightDataPlaneCloneObject(existing.CIConfig)
		if displayName == "" {
			displayName = existing.DisplayName
		}
	} else {
		status = http.StatusCreated
	}
	if displayName == "" {
		displayName = "Stackyard Playwright Run"
	}
	if raw := azurePlaywrightDataPlaneObject(payload["config"]); len(raw) > 0 {
		config = azurePlaywrightDataPlaneMergeObjects(config, raw)
	}
	if raw := azurePlaywrightDataPlaneObject(payload["ciConfig"]); len(raw) > 0 {
		ciConfig = azurePlaywrightDataPlaneMergeObjects(ciConfig, raw)
	}

	s.azurePlaywrightDataPlaneTestRuns[key] = &azurePlaywrightDataPlaneTestRun{
		WorkspaceID: workspaceID,
		RunID:       runID,
		DisplayName: displayName,
		Config:      azurePlaywrightDataPlaneNormalizeRunConfig(config),
		CIConfig:    azurePlaywrightDataPlaneNormalizeRunCIConfig(ciConfig),
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	respondJSON(w, status, azurePlaywrightDataPlaneTestRunFixture(s.azurePlaywrightDataPlaneTestRuns[key]))
}

func (s *Server) handleAzurePlaywrightDataPlaneListTestRuns(w http.ResponseWriter, r *http.Request, workspaceID string) {
	top, skip, ok := azurePlaywrightDataPlanePagination(w, r, 10)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	_ = s.azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID)
	items := make([]*azurePlaywrightDataPlaneTestRun, 0)
	for _, run := range s.azurePlaywrightDataPlaneTestRuns {
		if run.WorkspaceID == workspaceID {
			items = append(items, run)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].RunID < items[j].RunID })

	value, nextLink := azurePlaywrightDataPlaneRunPage(items, workspaceID, top, skip)
	respondJSON(w, http.StatusOK, map[string]any{
		"value":    value,
		"nextLink": nextLink,
	})
}

func (s *Server) azurePlaywrightDataPlaneEnsureWorkspaceLocked(workspaceID string) *azurePlaywrightDataPlaneWorkspace {
	if workspace := s.azurePlaywrightDataPlaneWorkspaces[workspaceID]; workspace != nil {
		return workspace
	}
	for _, workspace := range s.azurePlaywrightWorkspaces {
		if strings.EqualFold(azurePlaywrightString(workspace.Properties["workspaceId"]), workspaceID) {
			item := &azurePlaywrightDataPlaneWorkspace{
				WorkspaceID:  workspaceID,
				DataplaneURI: strings.TrimSpace(azurePlaywrightString(workspace.Properties["dataplaneUri"])),
				CreatedAt:    workspace.CreatedAt,
				UpdatedAt:    workspace.UpdatedAt,
			}
			if item.DataplaneURI == "" {
				item.DataplaneURI = azurePlaywrightDataPlaneDataplaneURI(workspaceID)
			}
			s.azurePlaywrightDataPlaneWorkspaces[workspaceID] = item
			return item
		}
	}
	now := time.Now().UTC()
	item := &azurePlaywrightDataPlaneWorkspace{
		WorkspaceID:  workspaceID,
		DataplaneURI: azurePlaywrightDataPlaneDataplaneURI(workspaceID),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.azurePlaywrightDataPlaneWorkspaces[workspaceID] = item
	return item
}

func azurePlaywrightDataPlaneAccessTokenFixture(token *azurePlaywrightDataPlaneAccessToken) map[string]any {
	return map[string]any{
		"id":        token.AccessTokenID,
		"name":      token.Name,
		"jwtToken":  token.JWTToken,
		"createdAt": token.CreatedAt.UTC().Format(time.RFC3339),
		"expiryAt":  token.ExpiryAt.UTC().Format(time.RFC3339),
		"state":     azurePlaywrightDataPlaneTokenState(token.ExpiryAt),
	}
}

func azurePlaywrightDataPlaneTestRunFixture(run *azurePlaywrightDataPlaneTestRun) map[string]any {
	return map[string]any{
		"id":          run.RunID,
		"displayName": run.DisplayName,
		"creatorId":   "00000000-0000-0000-0000-000000000001",
		"creatorName": "Stackyard Local",
		"config":      azurePlaywrightDataPlaneCloneObject(run.Config),
		"ciConfig":    azurePlaywrightDataPlaneCloneObject(run.CIConfig),
		"summary": map[string]any{
			"status":                       "SERVER_COMPLETE",
			"billableTime":                 "PT3M",
			"numBrowserSessions":           1,
			"maxConcurrentBrowserSessions": 1,
			"startTime":                    run.CreatedAt.UTC().Format(time.RFC3339),
			"endTime":                      run.UpdatedAt.UTC().Format(time.RFC3339),
			"duration":                     "PT3M",
			"errorMessages":                []any{},
		},
	}
}

func azurePlaywrightDataPlaneNormalizeRunConfig(config map[string]any) map[string]any {
	out := azurePlaywrightDataPlaneCloneObject(config)
	framework := azurePlaywrightDataPlaneMergeObjects(map[string]any{
		"name":       "playwright",
		"version":    "1.48.2",
		"runnerName": "npx playwright test",
	}, azurePlaywrightDataPlaneObject(out["framework"]))
	out["framework"] = framework
	if strings.TrimSpace(azurePlaywrightDataPlaneString(out["sdkLanguage"])) == "" {
		out["sdkLanguage"] = "TYPESCRIPT"
	}
	if _, ok := azurePlaywrightDataPlaneInt(out["maxWorkers"]); !ok {
		out["maxWorkers"] = 1
	}
	return out
}

func azurePlaywrightDataPlaneNormalizeRunCIConfig(config map[string]any) map[string]any {
	out := azurePlaywrightDataPlaneCloneObject(config)
	if strings.TrimSpace(azurePlaywrightDataPlaneString(out["providerName"])) == "" {
		out["providerName"] = "GitHubActions"
	}
	if strings.TrimSpace(azurePlaywrightDataPlaneString(out["branch"])) == "" {
		out["branch"] = "main"
	}
	if strings.TrimSpace(azurePlaywrightDataPlaneString(out["author"])) == "" {
		out["author"] = "stackyard"
	}
	if strings.TrimSpace(azurePlaywrightDataPlaneString(out["commitId"])) == "" {
		out["commitId"] = "0123456789abcdef0123456789abcdef01234567"
	}
	if strings.TrimSpace(azurePlaywrightDataPlaneString(out["revisionUrl"])) == "" {
		out["revisionUrl"] = "https://example.invalid/revision/01234567"
	}
	return out
}

func azurePlaywrightDataPlaneAccessTokenPage(items []*azurePlaywrightDataPlaneAccessToken, workspaceID string, top, skip int) ([]any, string) {
	if skip > len(items) {
		skip = len(items)
	}
	end := skip + top
	if end > len(items) {
		end = len(items)
	}
	value := make([]any, 0, end-skip)
	for _, item := range items[skip:end] {
		value = append(value, azurePlaywrightDataPlaneAccessTokenFixture(item))
	}
	nextLink := ""
	if end < len(items) {
		nextLink = fmt.Sprintf("%s/playwrightworkspaces/%s/access-tokens?api-version=%s&$skip=%d&$top=%d", azurePlaywrightDataPlanePrefix, workspaceID, azurePlaywrightDataPlaneAPIVersion, end, top)
	}
	return value, nextLink
}

func azurePlaywrightDataPlaneRunPage(items []*azurePlaywrightDataPlaneTestRun, workspaceID string, top, skip int) ([]any, string) {
	if skip > len(items) {
		skip = len(items)
	}
	end := skip + top
	if end > len(items) {
		end = len(items)
	}
	value := make([]any, 0, end-skip)
	for _, item := range items[skip:end] {
		value = append(value, azurePlaywrightDataPlaneTestRunFixture(item))
	}
	nextLink := ""
	if end < len(items) {
		nextLink = fmt.Sprintf("%s/playwrightworkspaces/%s/test-runs?api-version=%s&$skip=%d&$top=%d", azurePlaywrightDataPlanePrefix, workspaceID, azurePlaywrightDataPlaneAPIVersion, end, top)
	}
	return value, nextLink
}

func azurePlaywrightDataPlanePagination(w http.ResponseWriter, r *http.Request, defaultTop int) (top, skip int, ok bool) {
	top = defaultTop
	skip = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("$top")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "$top must be a positive integer")
			return 0, 0, false
		}
		top = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("$skip")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			azurePlaywrightDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "$skip must be a non-negative integer")
			return 0, 0, false
		}
		skip = value
	}
	return top, skip, true
}

func azurePlaywrightDataPlaneTokenState(expiryAt time.Time) string {
	if time.Now().UTC().After(expiryAt.UTC()) {
		return "Expired"
	}
	return "Active"
}

func azurePlaywrightDataPlaneDataplaneURI(workspaceID string) string {
	return "https://eastus.api.playwright.microsoft.com/workspaces/" + workspaceID
}

func azurePlaywrightDataPlaneJWT(workspaceID, accessTokenID string) string {
	sum := sha1.Sum([]byte(workspaceID + ":" + accessTokenID))
	return "eyJzdGFja3lhcmQiOiI" + hex.EncodeToString(sum[:12])
}

func azurePlaywrightDataPlaneAccessTokenKey(workspaceID, accessTokenID string) string {
	return strings.ToLower(strings.TrimSpace(workspaceID)) + "::" + strings.ToLower(strings.TrimSpace(accessTokenID))
}

func azurePlaywrightDataPlaneRunKey(workspaceID, runID string) string {
	return strings.ToLower(strings.TrimSpace(workspaceID)) + "::" + strings.ToLower(strings.TrimSpace(runID))
}

func azurePlaywrightDataPlaneValidGUID(value string) bool {
	return azurePlaywrightDataPlaneGUIDPattern.MatchString(strings.TrimSpace(value))
}

func azurePlaywrightDataPlaneValidAccessTokenID(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 3 && len(value) <= 36 && azurePlaywrightDataPlaneTokenIDPattern.MatchString(value)
}

func azurePlaywrightDataPlaneValidTokenName(value string) bool {
	return azurePlaywrightDataPlaneTokenNamePattern.MatchString(strings.TrimSpace(value))
}

func azurePlaywrightDataPlaneError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-ms-error-code", code)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azurePlaywrightDataPlaneString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func azurePlaywrightDataPlaneObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azurePlaywrightDataPlaneCloneObject(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azurePlaywrightDataPlaneCloneObject(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func azurePlaywrightDataPlaneMergeObjects(base, patch map[string]any) map[string]any {
	if len(base) == 0 && len(patch) == 0 {
		return map[string]any{}
	}
	out := azurePlaywrightDataPlaneCloneObject(base)
	for key, value := range patch {
		if nestedPatch, ok := value.(map[string]any); ok {
			if nestedBase, ok := out[key].(map[string]any); ok {
				out[key] = azurePlaywrightDataPlaneMergeObjects(nestedBase, nestedPatch)
			} else {
				out[key] = azurePlaywrightDataPlaneCloneObject(nestedPatch)
			}
			continue
		}
		out[key] = value
	}
	return out
}

func azurePlaywrightDataPlaneInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}
