package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"mime"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureLoadTestingDataPlaneAPIVersion = "2022-11-01"
const azureLoadTestingDataPlanePrefix = "/azure/loadtesting-data-plane/2022-11-01"

type azureLoadTestingDataPlaneTest struct {
	TestID              string
	Payload             map[string]any
	AppComponents       map[string]any
	ServerMetricsConfig map[string]any
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type azureLoadTestingDataPlaneRun struct {
	TestRunID           string
	Payload             map[string]any
	AppComponents       map[string]any
	ServerMetricsConfig map[string]any
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type azureLoadTestingDataPlaneFile struct {
	ScopeKind   string
	ScopeID     string
	FileName    string
	FileType    string
	ContentType string
	Content     []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (s *Server) handleAzureLoadTestingDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureLoadTestingDataPlanePrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureLoadTestingDataPlaneAPIVersion {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "api-version must be 2022-11-01")
		return true
	}

	relative := strings.TrimPrefix(path, azureLoadTestingDataPlanePrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "tests":
		s.handleAzureLoadTestingDataPlaneTests(w, r, segments[1:])
		return true
	case "test-runs":
		s.handleAzureLoadTestingDataPlaneRuns(w, r, segments[1:])
		return true
	default:
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneTests(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureLoadTestingDataPlaneListTests(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	testID := strings.TrimSpace(tail[0])
	if !azureLoadTestingDataPlaneValidID(testID) {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "testId is invalid")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureLoadTestingDataPlaneGetTest(w, testID)
		case http.MethodPatch:
			s.handleAzureLoadTestingDataPlanePatchTest(w, r, testID)
		case http.MethodDelete:
			s.handleAzureLoadTestingDataPlaneDeleteTest(w, testID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "files":
		s.handleAzureLoadTestingDataPlaneTestFiles(w, r, testID, tail[2:])
	case "app-components":
		s.handleAzureLoadTestingDataPlaneTestAppComponents(w, r, testID, tail[2:])
	case "server-metrics-config":
		s.handleAzureLoadTestingDataPlaneTestServerMetrics(w, r, testID, tail[2:])
	default:
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneRuns(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureLoadTestingDataPlaneListRuns(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runIDSegment := strings.TrimSpace(tail[0])
	stopAction := strings.HasSuffix(runIDSegment, ":stop")
	runID := strings.TrimSuffix(runIDSegment, ":stop")
	if !azureLoadTestingDataPlaneValidID(runID) {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "testRunId is invalid")
		return
	}
	if stopAction {
		if len(tail) != 1 {
			azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureLoadTestingDataPlaneStopRun(w, runID)
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureLoadTestingDataPlaneGetRun(w, runID)
		case http.MethodPatch:
			s.handleAzureLoadTestingDataPlanePatchRun(w, r, runID)
		case http.MethodDelete:
			s.handleAzureLoadTestingDataPlaneDeleteRun(w, runID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "files":
		s.handleAzureLoadTestingDataPlaneRunFiles(w, r, runID, tail[2:])
	case "app-components":
		s.handleAzureLoadTestingDataPlaneRunAppComponents(w, r, runID, tail[2:])
	case "server-metrics-config":
		s.handleAzureLoadTestingDataPlaneRunServerMetrics(w, r, runID, tail[2:])
	case "metric-namespaces":
		s.handleAzureLoadTestingDataPlaneMetricNamespaces(w, r, runID, tail[2:])
	case "metric-definitions":
		s.handleAzureLoadTestingDataPlaneMetricDefinitions(w, r, runID, tail[2:])
	case "metric-dimensions":
		s.handleAzureLoadTestingDataPlaneMetricDimensions(w, r, runID, tail[2:])
	case "metrics":
		s.handleAzureLoadTestingDataPlaneMetrics(w, r, runID, tail[2:])
	default:
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneListTests(w http.ResponseWriter, r *http.Request) {
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))

	s.providerStorageMu.Lock()
	items := make([]*azureLoadTestingDataPlaneTest, 0, len(s.azureLoadTestingDataPlaneTests))
	for _, item := range s.azureLoadTestingDataPlaneTests {
		if search != "" {
			displayName := strings.ToLower(azureLoadTestingDataPlaneString(item.Payload["displayName"]))
			description := strings.ToLower(azureLoadTestingDataPlaneString(item.Payload["description"]))
			if !strings.Contains(strings.ToLower(item.TestID), search) && !strings.Contains(displayName, search) && !strings.Contains(description, search) {
				continue
			}
		}
		items = append(items, item)
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].TestID < items[j].TestID })

	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureLoadTestingDataPlaneTestFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureLoadTestingDataPlaneGetTest(w http.ResponseWriter, testID string) {
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneTests[testID]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestNotFound", "test not found")
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneTestFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlanePatchTest(w http.ResponseWriter, r *http.Request, testID string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	payload, ok := azureLoadTestingDataPlaneJSONMap(body, w)
	if !ok {
		return
	}

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item := s.azureLoadTestingDataPlaneTests[testID]
	createdAt := now
	base := map[string]any{}
	appComponents := map[string]any{}
	serverMetrics := map[string]any{}
	if item != nil {
		createdAt = item.CreatedAt
		base = azureLoadTestingDataPlaneCloneObject(item.Payload)
		appComponents = azureLoadTestingDataPlaneCloneObject(item.AppComponents)
		serverMetrics = azureLoadTestingDataPlaneCloneObject(item.ServerMetricsConfig)
	}
	merged := azureLoadTestingDataPlaneMergeObjects(base, payload)
	item = &azureLoadTestingDataPlaneTest{
		TestID:              testID,
		Payload:             azureLoadTestingDataPlaneNormalizeTestPayload(testID, merged, createdAt, now),
		AppComponents:       appComponents,
		ServerMetricsConfig: serverMetrics,
		CreatedAt:           createdAt,
		UpdatedAt:           now,
	}
	s.azureLoadTestingDataPlaneTests[testID] = item
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneTestFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlaneDeleteTest(w http.ResponseWriter, testID string) {
	s.providerStorageMu.Lock()
	_, exists := s.azureLoadTestingDataPlaneTests[testID]
	if exists {
		delete(s.azureLoadTestingDataPlaneTests, testID)
		for key, file := range s.azureLoadTestingDataPlaneFiles {
			if file.ScopeKind == "test" && file.ScopeID == testID {
				delete(s.azureLoadTestingDataPlaneFiles, key)
			}
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureLoadTestingDataPlaneTestFiles(w http.ResponseWriter, r *http.Request, testID string, tail []string) {
	if !s.azureLoadTestingDataPlaneTestExists(testID) {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestNotFound", "test not found")
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureLoadTestingDataPlaneListFiles(w, "test", testID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	fileName := strings.TrimSpace(tail[0])
	if fileName == "" {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "fileName is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureLoadTestingDataPlaneGetFile(w, "test", testID, fileName)
	case http.MethodPut:
		s.handleAzureLoadTestingDataPlanePutFile(w, r, "test", testID, fileName)
	case http.MethodDelete:
		s.handleAzureLoadTestingDataPlaneDeleteFile(w, "test", testID, fileName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneTestAppComponents(w http.ResponseWriter, r *http.Request, testID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneTests[testID]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestNotFound", "test not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"components": azureLoadTestingDataPlaneCloneObject(item.AppComponents)})
	case http.MethodPatch:
		body, err := readBodyBytes(r)
		if err != nil {
			azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
			return
		}
		payload, ok := azureLoadTestingDataPlaneJSONMap(body, w)
		if !ok {
			return
		}
		components := azureLoadTestingDataPlaneObject(payload["components"])
		s.providerStorageMu.Lock()
		item = s.azureLoadTestingDataPlaneTests[testID]
		item.AppComponents = azureLoadTestingDataPlaneMergeObjects(item.AppComponents, components)
		item.UpdatedAt = time.Now().UTC()
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"components": azureLoadTestingDataPlaneCloneObject(item.AppComponents)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneTestServerMetrics(w http.ResponseWriter, r *http.Request, testID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneTests[testID]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestNotFound", "test not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"metrics": azureLoadTestingDataPlaneCloneObject(item.ServerMetricsConfig)})
	case http.MethodPatch:
		body, err := readBodyBytes(r)
		if err != nil {
			azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
			return
		}
		payload, ok := azureLoadTestingDataPlaneJSONMap(body, w)
		if !ok {
			return
		}
		metrics := azureLoadTestingDataPlaneObject(payload["metrics"])
		s.providerStorageMu.Lock()
		item = s.azureLoadTestingDataPlaneTests[testID]
		item.ServerMetricsConfig = azureLoadTestingDataPlaneMergeObjects(item.ServerMetricsConfig, metrics)
		item.UpdatedAt = time.Now().UTC()
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"metrics": azureLoadTestingDataPlaneCloneObject(item.ServerMetricsConfig)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneListRuns(w http.ResponseWriter, r *http.Request) {
	search := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))

	s.providerStorageMu.Lock()
	items := make([]*azureLoadTestingDataPlaneRun, 0, len(s.azureLoadTestingDataPlaneRuns))
	for _, item := range s.azureLoadTestingDataPlaneRuns {
		if search != "" {
			displayName := strings.ToLower(azureLoadTestingDataPlaneString(item.Payload["displayName"]))
			if !strings.Contains(strings.ToLower(item.TestRunID), search) && !strings.Contains(displayName, search) {
				continue
			}
		}
		items = append(items, item)
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].TestRunID < items[j].TestRunID })

	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureLoadTestingDataPlaneRunFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureLoadTestingDataPlaneGetRun(w http.ResponseWriter, runID string) {
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneRuns[runID]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneRunFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlanePatchRun(w http.ResponseWriter, r *http.Request, runID string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	payload, ok := azureLoadTestingDataPlaneJSONMap(body, w)
	if !ok {
		return
	}

	now := time.Now().UTC()
	oldRunID := strings.TrimSpace(r.URL.Query().Get("oldTestRunId"))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item := s.azureLoadTestingDataPlaneRuns[runID]
	createdAt := now
	appComponents := map[string]any{}
	serverMetrics := map[string]any{}
	base := map[string]any{}
	if item != nil {
		createdAt = item.CreatedAt
		base = azureLoadTestingDataPlaneCloneObject(item.Payload)
		appComponents = azureLoadTestingDataPlaneCloneObject(item.AppComponents)
		serverMetrics = azureLoadTestingDataPlaneCloneObject(item.ServerMetricsConfig)
	}
	if item == nil && oldRunID != "" {
		if previous := s.azureLoadTestingDataPlaneRuns[oldRunID]; previous != nil {
			base = azureLoadTestingDataPlaneCloneObject(previous.Payload)
			appComponents = azureLoadTestingDataPlaneCloneObject(previous.AppComponents)
			serverMetrics = azureLoadTestingDataPlaneCloneObject(previous.ServerMetricsConfig)
		}
	}
	merged := azureLoadTestingDataPlaneMergeObjects(base, payload)
	testID := azureLoadTestingDataPlaneString(merged["testId"])
	if testID == "" && oldRunID == "" {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "testId is required")
		return
	}
	if testID == "" && oldRunID != "" {
		testID = azureLoadTestingDataPlaneString(base["testId"])
	}
	if !azureLoadTestingDataPlaneValidID(testID) {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "testId is invalid")
		return
	}
	if s.azureLoadTestingDataPlaneTests[testID] == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestNotFound", "test not found")
		return
	}

	if len(appComponents) == 0 {
		appComponents = azureLoadTestingDataPlaneCloneObject(s.azureLoadTestingDataPlaneTests[testID].AppComponents)
	}
	if len(serverMetrics) == 0 {
		serverMetrics = azureLoadTestingDataPlaneCloneObject(s.azureLoadTestingDataPlaneTests[testID].ServerMetricsConfig)
	}

	item = &azureLoadTestingDataPlaneRun{
		TestRunID:           runID,
		Payload:             azureLoadTestingDataPlaneNormalizeRunPayload(runID, merged, testID, createdAt, now),
		AppComponents:       appComponents,
		ServerMetricsConfig: serverMetrics,
		CreatedAt:           createdAt,
		UpdatedAt:           now,
	}
	s.azureLoadTestingDataPlaneRuns[runID] = item
	s.azureLoadTestingDataPlaneSeedRunFilesLocked(runID, createdAt, now)
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneRunFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlaneDeleteRun(w http.ResponseWriter, runID string) {
	s.providerStorageMu.Lock()
	_, exists := s.azureLoadTestingDataPlaneRuns[runID]
	if exists {
		delete(s.azureLoadTestingDataPlaneRuns, runID)
		for key, file := range s.azureLoadTestingDataPlaneFiles {
			if file.ScopeKind == "run" && file.ScopeID == runID {
				delete(s.azureLoadTestingDataPlaneFiles, key)
			}
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureLoadTestingDataPlaneStopRun(w http.ResponseWriter, runID string) {
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneRuns[runID]
	if item != nil {
		item.Payload["status"] = "CANCELLED"
		item.Payload["lastModifiedDateTime"] = time.Now().UTC().Format(time.RFC3339)
		item.Payload["testResult"] = map[string]any{"status": "CANCELLED"}
		item.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneRunFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlaneRunFiles(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 1 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.providerStorageMu.Lock()
	run := s.azureLoadTestingDataPlaneRuns[runID]
	s.providerStorageMu.Unlock()
	if run == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	fileName := strings.TrimSpace(tail[0])
	if fileName == "" {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "fileName is required")
		return
	}
	s.handleAzureLoadTestingDataPlaneGetFile(w, "run", runID, fileName)
}

func (s *Server) handleAzureLoadTestingDataPlaneRunAppComponents(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneRuns[runID]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"components": azureLoadTestingDataPlaneCloneObject(item.AppComponents)})
	case http.MethodPatch:
		body, err := readBodyBytes(r)
		if err != nil {
			azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
			return
		}
		payload, ok := azureLoadTestingDataPlaneJSONMap(body, w)
		if !ok {
			return
		}
		components := azureLoadTestingDataPlaneObject(payload["components"])
		s.providerStorageMu.Lock()
		item = s.azureLoadTestingDataPlaneRuns[runID]
		item.AppComponents = azureLoadTestingDataPlaneMergeObjects(item.AppComponents, components)
		item.UpdatedAt = time.Now().UTC()
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"components": azureLoadTestingDataPlaneCloneObject(item.AppComponents)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneRunServerMetrics(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneRuns[runID]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"metrics": azureLoadTestingDataPlaneCloneObject(item.ServerMetricsConfig)})
	case http.MethodPatch:
		body, err := readBodyBytes(r)
		if err != nil {
			azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
			return
		}
		payload, ok := azureLoadTestingDataPlaneJSONMap(body, w)
		if !ok {
			return
		}
		metrics := azureLoadTestingDataPlaneObject(payload["metrics"])
		s.providerStorageMu.Lock()
		item = s.azureLoadTestingDataPlaneRuns[runID]
		item.ServerMetricsConfig = azureLoadTestingDataPlaneMergeObjects(item.ServerMetricsConfig, metrics)
		item.UpdatedAt = time.Now().UTC()
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusOK, map[string]any{"metrics": azureLoadTestingDataPlaneCloneObject(item.ServerMetricsConfig)})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureLoadTestingDataPlaneMetricNamespaces(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.azureLoadTestingDataPlaneRunExists(runID) {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []any{
			map[string]any{"namespaceName": "LoadTestRunMetrics"},
			map[string]any{"namespaceName": "EngineHealth"},
		},
	})
}

func (s *Server) handleAzureLoadTestingDataPlaneMetricDefinitions(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.azureLoadTestingDataPlaneRunExists(runID) {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	namespace := strings.TrimSpace(r.URL.Query().Get("metricNamespace"))
	if namespace == "" {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "metricNamespace is required")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []any{
			map[string]any{
				"name":                   "response_time_ms",
				"metricNamespace":        namespace,
				"primaryAggregationType": "Average",
				"unit":                   "Milliseconds",
				"dimensions": []any{
					map[string]any{"name": "sampler"},
					map[string]any{"name": "result"},
				},
			},
			map[string]any{
				"name":                   "requests_per_sec",
				"metricNamespace":        namespace,
				"primaryAggregationType": "Average",
				"unit":                   "CountPerSecond",
				"dimensions": []any{
					map[string]any{"name": "instance"},
				},
			},
		},
	})
}

func (s *Server) handleAzureLoadTestingDataPlaneMetricDimensions(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 1 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.azureLoadTestingDataPlaneRunExists(runID) {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	dimensionName := strings.TrimSpace(tail[0])
	metricName := strings.TrimSpace(r.URL.Query().Get("metricName"))
	namespace := strings.TrimSpace(r.URL.Query().Get("metricNamespace"))
	if dimensionName == "" || metricName == "" || namespace == "" {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "dimensionName, metricName, and metricNamespace are required")
		return
	}
	values := []string{"SamplerA", "SamplerB"}
	if strings.EqualFold(dimensionName, "result") {
		values = []string{"passed", "failed"}
	}
	if strings.EqualFold(dimensionName, "instance") {
		values = []string{"engine-1", "engine-2"}
	}
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, map[string]any{
			"name":            dimensionName,
			"value":           value,
			"metricName":      metricName,
			"metricNamespace": namespace,
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": out})
}

func (s *Server) handleAzureLoadTestingDataPlaneMetrics(w http.ResponseWriter, r *http.Request, runID string, tail []string) {
	if len(tail) != 0 {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.azureLoadTestingDataPlaneRunExists(runID) {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "TestRunNotFound", "test run not found")
		return
	}
	metricName := strings.TrimSpace(r.URL.Query().Get("metricname"))
	namespace := strings.TrimSpace(r.URL.Query().Get("metricNamespace"))
	timespan := strings.TrimSpace(r.URL.Query().Get("timespan"))
	if metricName == "" || namespace == "" || timespan == "" {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "metricname, metricNamespace, and timespan are required")
		return
	}
	aggregation := firstNonEmptyString(strings.TrimSpace(r.URL.Query().Get("aggregation")), "Average")
	interval := firstNonEmptyString(strings.TrimSpace(r.URL.Query().Get("interval")), "PT1M")
	now := time.Now().UTC()
	base := float64(azureLoadTestingDataPlaneStableNumber(runID + metricName))
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []any{
			map[string]any{
				"name":            metricName,
				"metricNamespace": namespace,
				"timeseries": []any{
					map[string]any{
						"metadataValues": []any{
							map[string]any{"name": "sampler", "value": "SamplerA"},
						},
						"data": []any{
							map[string]any{"timeStamp": now.Add(-2 * time.Minute).Format(time.RFC3339), strings.ToLower(aggregation): base + 10},
							map[string]any{"timeStamp": now.Add(-1 * time.Minute).Format(time.RFC3339), strings.ToLower(aggregation): base + 20},
							map[string]any{"timeStamp": now.Format(time.RFC3339), strings.ToLower(aggregation): base + 15},
						},
					},
				},
				"interval": interval,
				"timespan": timespan,
			},
		},
	})
}

func (s *Server) handleAzureLoadTestingDataPlaneListFiles(w http.ResponseWriter, scopeKind, scopeID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureLoadTestingDataPlaneFile, 0)
	for _, item := range s.azureLoadTestingDataPlaneFiles {
		if item.ScopeKind == scopeKind && item.ScopeID == scopeID {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].FileName < items[j].FileName })

	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureLoadTestingDataPlaneFileFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureLoadTestingDataPlaneGetFile(w http.ResponseWriter, scopeKind, scopeID, fileName string) {
	s.providerStorageMu.Lock()
	item := s.azureLoadTestingDataPlaneFiles[azureLoadTestingDataPlaneFileKey(scopeKind, scopeID, fileName)]
	s.providerStorageMu.Unlock()
	if item == nil {
		azureLoadTestingDataPlaneError(w, http.StatusNotFound, "FileNotFound", "file not found")
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneFileFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlanePutFile(w http.ResponseWriter, r *http.Request, scopeKind, scopeID, fileName string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	if len(body) == 0 {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "file content is required")
		return
	}
	now := time.Now().UTC()
	item := &azureLoadTestingDataPlaneFile{
		ScopeKind:   scopeKind,
		ScopeID:     scopeID,
		FileName:    fileName,
		FileType:    firstNonEmptyString(strings.TrimSpace(r.URL.Query().Get("fileType")), azureLoadTestingDataPlaneInferFileType(fileName)),
		ContentType: firstNonEmptyString(strings.TrimSpace(r.Header.Get("Content-Type")), mime.TypeByExtension(strings.ToLower(pathExt(fileName)))),
		Content:     append([]byte(nil), body...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if item.ContentType == "" {
		item.ContentType = "application/octet-stream"
	}
	s.providerStorageMu.Lock()
	if existing := s.azureLoadTestingDataPlaneFiles[azureLoadTestingDataPlaneFileKey(scopeKind, scopeID, fileName)]; existing != nil {
		item.CreatedAt = existing.CreatedAt
	}
	s.azureLoadTestingDataPlaneFiles[azureLoadTestingDataPlaneFileKey(scopeKind, scopeID, fileName)] = item
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureLoadTestingDataPlaneFileFixture(item))
}

func (s *Server) handleAzureLoadTestingDataPlaneDeleteFile(w http.ResponseWriter, scopeKind, scopeID, fileName string) {
	s.providerStorageMu.Lock()
	delete(s.azureLoadTestingDataPlaneFiles, azureLoadTestingDataPlaneFileKey(scopeKind, scopeID, fileName))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) azureLoadTestingDataPlaneTestExists(testID string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureLoadTestingDataPlaneTests[testID] != nil
}

func (s *Server) azureLoadTestingDataPlaneRunExists(runID string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureLoadTestingDataPlaneRuns[runID] != nil
}

func (s *Server) azureLoadTestingDataPlaneSeedRunFilesLocked(runID string, createdAt, updatedAt time.Time) {
	for _, spec := range []struct {
		Name        string
		FileType    string
		ContentType string
		Content     []byte
	}{
		{"results.csv", "RESULTS_FILE", "text/csv", []byte("timestamp,metric,value\n2026-01-01T00:00:00Z,response_time_ms,120\n")},
		{"junit.xml", "JUNIT_XML", "application/xml", []byte("<testsuite tests=\"1\" failures=\"0\"></testsuite>")},
	} {
		key := azureLoadTestingDataPlaneFileKey("run", runID, spec.Name)
		if s.azureLoadTestingDataPlaneFiles[key] != nil {
			continue
		}
		s.azureLoadTestingDataPlaneFiles[key] = &azureLoadTestingDataPlaneFile{
			ScopeKind:   "run",
			ScopeID:     runID,
			FileName:    spec.Name,
			FileType:    spec.FileType,
			ContentType: spec.ContentType,
			Content:     append([]byte(nil), spec.Content...),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}
	}
}

func azureLoadTestingDataPlaneTestFixture(item *azureLoadTestingDataPlaneTest) map[string]any {
	return azureLoadTestingDataPlaneCloneObject(item.Payload)
}

func azureLoadTestingDataPlaneRunFixture(item *azureLoadTestingDataPlaneRun) map[string]any {
	return azureLoadTestingDataPlaneCloneObject(item.Payload)
}

func azureLoadTestingDataPlaneFileFixture(item *azureLoadTestingDataPlaneFile) map[string]any {
	return map[string]any{
		"fileName":         item.FileName,
		"fileType":         item.FileType,
		"contentType":      item.ContentType,
		"url":              azureLoadTestingDataPlaneFileURL(item.ScopeKind, item.ScopeID, item.FileName),
		"expireDateTime":   item.UpdatedAt.Add(24 * time.Hour).Format(time.RFC3339),
		"lastModifiedTime": item.UpdatedAt.Format(time.RFC3339),
		"validationStatus": "VALIDATION_SUCCESS",
	}
}

func azureLoadTestingDataPlaneNormalizeTestPayload(testID string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	normalized := azureLoadTestingDataPlaneCloneObject(payload)
	normalized["testId"] = testID
	if _, ok := normalized["displayName"]; !ok {
		normalized["displayName"] = "Test " + testID
	}
	if _, ok := normalized["description"]; !ok {
		normalized["description"] = "Stackyard load test"
	}
	if _, ok := normalized["loadTestConfiguration"]; !ok {
		normalized["loadTestConfiguration"] = map[string]any{
			"engineInstances": 1,
			"splitAllCSVs":    false,
		}
	}
	if _, ok := normalized["passFailCriteria"]; !ok {
		normalized["passFailCriteria"] = map[string]any{
			"passPercentage": 100,
		}
	}
	normalized["createdDateTime"] = createdAt.Format(time.RFC3339)
	normalized["lastModifiedDateTime"] = updatedAt.Format(time.RFC3339)
	return normalized
}

func azureLoadTestingDataPlaneNormalizeRunPayload(runID string, payload map[string]any, testID string, createdAt, updatedAt time.Time) map[string]any {
	normalized := azureLoadTestingDataPlaneCloneObject(payload)
	normalized["testRunId"] = runID
	normalized["testId"] = testID
	if _, ok := normalized["displayName"]; !ok {
		normalized["displayName"] = "Run " + runID
	}
	if _, ok := normalized["description"]; !ok {
		normalized["description"] = "Stackyard test run"
	}
	if _, ok := normalized["status"]; !ok {
		normalized["status"] = "EXECUTING"
	}
	if _, ok := normalized["virtualUsers"]; !ok {
		normalized["virtualUsers"] = 25
	}
	if _, ok := normalized["testResult"]; !ok {
		normalized["testResult"] = map[string]any{"status": normalized["status"]}
	}
	normalized["createdDateTime"] = createdAt.Format(time.RFC3339)
	normalized["lastModifiedDateTime"] = updatedAt.Format(time.RFC3339)
	normalized["artifacts"] = map[string]any{
		"resultFileInfo":        azureLoadTestingDataPlaneFileFixture(&azureLoadTestingDataPlaneFile{ScopeKind: "run", ScopeID: runID, FileName: "results.csv", FileType: "RESULTS_FILE", ContentType: "text/csv", UpdatedAt: updatedAt}),
		"junitXmlFileInfo":      azureLoadTestingDataPlaneFileFixture(&azureLoadTestingDataPlaneFile{ScopeKind: "run", ScopeID: runID, FileName: "junit.xml", FileType: "JUNIT_XML", ContentType: "application/xml", UpdatedAt: updatedAt}),
		"inputArtifactsZipFile": azureLoadTestingDataPlaneFileURL("run", runID, "input-artifacts.zip"),
	}
	return normalized
}

func azureLoadTestingDataPlaneFileURL(scopeKind, scopeID, fileName string) string {
	base := azureLoadTestingDataPlanePrefix + "/"
	if scopeKind == "test" {
		return base + "tests/" + scopeID + "/files/" + fileName + "?api-version=" + azureLoadTestingDataPlaneAPIVersion
	}
	return base + "test-runs/" + scopeID + "/files/" + fileName + "?api-version=" + azureLoadTestingDataPlaneAPIVersion
}

func azureLoadTestingDataPlaneInferFileType(fileName string) string {
	switch strings.ToLower(strings.TrimSpace(fileName)) {
	case "junit.xml":
		return "JUNIT_XML"
	}
	if strings.HasSuffix(strings.ToLower(fileName), ".jmx") {
		return "JMX_FILE"
	}
	if strings.HasSuffix(strings.ToLower(fileName), ".zip") {
		return "ZIP"
	}
	if strings.HasSuffix(strings.ToLower(fileName), ".csv") {
		return "CSV"
	}
	return "ADDITIONAL_ARTIFACT"
}

func azureLoadTestingDataPlaneFileKey(scopeKind, scopeID, fileName string) string {
	return strings.ToLower(scopeKind + "::" + scopeID + "::" + fileName)
}

func pathExt(name string) string {
	index := strings.LastIndex(name, ".")
	if index < 0 {
		return ""
	}
	return name[index:]
}

func azureLoadTestingDataPlaneValidID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func azureLoadTestingDataPlaneError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureLoadTestingDataPlaneJSONMap(body []byte, w http.ResponseWriter) (map[string]any, bool) {
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) == 0 {
		payload = map[string]any{}
	} else if err := json.Unmarshal(body, &payload); err != nil {
		azureLoadTestingDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func azureLoadTestingDataPlaneString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureLoadTestingDataPlaneObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return azureLoadTestingDataPlaneCloneObject(typed)
}

func azureLoadTestingDataPlaneCloneObject(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, raw := range value {
		switch typed := raw.(type) {
		case map[string]any:
			out[key] = azureLoadTestingDataPlaneCloneObject(typed)
		case []any:
			out[key] = azureLoadTestingDataPlaneCloneSlice(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func azureLoadTestingDataPlaneCloneSlice(value []any) []any {
	out := make([]any, 0, len(value))
	for _, raw := range value {
		switch typed := raw.(type) {
		case map[string]any:
			out = append(out, azureLoadTestingDataPlaneCloneObject(typed))
		case []any:
			out = append(out, azureLoadTestingDataPlaneCloneSlice(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func azureLoadTestingDataPlaneMergeObjects(base, overlay map[string]any) map[string]any {
	merged := azureLoadTestingDataPlaneCloneObject(base)
	for key, value := range overlay {
		switch typed := value.(type) {
		case map[string]any:
			if current, ok := merged[key].(map[string]any); ok {
				merged[key] = azureLoadTestingDataPlaneMergeObjects(current, typed)
			} else {
				merged[key] = azureLoadTestingDataPlaneCloneObject(typed)
			}
		case []any:
			merged[key] = azureLoadTestingDataPlaneCloneSlice(typed)
		default:
			merged[key] = typed
		}
	}
	return merged
}

func azureLoadTestingDataPlaneStableNumber(seed string) int {
	sum := sha1.Sum([]byte(seed))
	value, _ := strconv.ParseInt(hex.EncodeToString(sum[:4]), 16, 64)
	return int(value % 100)
}
