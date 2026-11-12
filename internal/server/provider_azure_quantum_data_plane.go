package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureQuantumDataPlaneAPIVersion = "2026-01-15-preview"
const azureQuantumDataPlanePrefix = "/azure/quantum-data-plane/2026-01-15-preview"

type azureQuantumDataPlaneJob struct {
	JobID            string
	Name             string
	SessionID        string
	ProviderID       string
	Target           string
	InputDataFormat  string
	OutputDataFormat string
	ContainerURI     string
	InputParams      map[string]any
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	BeginTime        time.Time
	EndTime          time.Time
}

type azureQuantumDataPlaneSession struct {
	SessionID        string
	Name             string
	ProviderID       string
	Target           string
	JobFailurePolicy string
	Status           string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	BeginTime        time.Time
	EndTime          time.Time
}

func (s *Server) handleAzureQuantumDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureQuantumDataPlanePrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureQuantumDataPlaneAPIVersion {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2026-01-15-preview")
		return true
	}

	segments := splitPathSegments(strings.Trim(strings.TrimPrefix(path, azureQuantumDataPlanePrefix), "/"))
	if len(segments) == 0 {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "providers":
		s.handleAzureQuantumDataPlaneProviders(w, r, segments[1:])
	case "quotas":
		s.handleAzureQuantumDataPlaneQuotas(w, r, segments[1:])
	case "toplevelitems":
		s.handleAzureQuantumDataPlaneTopLevelItems(w, r, segments[1:])
	case "storage":
		s.handleAzureQuantumDataPlaneStorage(w, r, segments[1:])
	case "jobs":
		s.handleAzureQuantumDataPlaneJobsRoute(w, r, segments[1:])
	case "sessions":
		s.handleAzureQuantumDataPlaneSessionsRoute(w, r, segments[1:])
	default:
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
	return true
}

func (s *Server) handleAzureQuantumDataPlaneProviders(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) != 0 {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": azureQuantumDataPlaneProviderFixtures()})
}

func (s *Server) handleAzureQuantumDataPlaneQuotas(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) != 0 {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": azureQuantumDataPlaneQuotaFixtures()})
}

func (s *Server) handleAzureQuantumDataPlaneTopLevelItems(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) != 0 {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0, len(s.azureQuantumDataPlaneSessions)+len(s.azureQuantumDataPlaneJobs))
	for _, session := range s.azureQuantumDataPlaneSessions {
		items = append(items, map[string]any{
			"id":           session.SessionID,
			"name":         session.Name,
			"itemType":     "Session",
			"status":       session.Status,
			"providerId":   session.ProviderID,
			"target":       session.Target,
			"creationTime": session.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	for _, job := range s.azureQuantumDataPlaneJobs {
		items = append(items, map[string]any{
			"id":           job.JobID,
			"name":         job.Name,
			"itemType":     "Job",
			"status":       job.Status,
			"providerId":   job.ProviderID,
			"target":       job.Target,
			"sessionId":    job.SessionID,
			"creationTime": job.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if azureQuantumDataPlaneString(items[i]["itemType"]) == azureQuantumDataPlaneString(items[j]["itemType"]) {
			return azureQuantumDataPlaneString(items[i]["id"]) < azureQuantumDataPlaneString(items[j]["id"])
		}
		return azureQuantumDataPlaneString(items[i]["itemType"]) < azureQuantumDataPlaneString(items[j]["itemType"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureQuantumDataPlaneStorage(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) != 1 || !strings.EqualFold(tail[0], "sasUri") {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := readBodyBytes(r)
	if err != nil {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	payload, ok := azureQuantumDataPlaneJSONMap(body, w)
	if !ok {
		return
	}
	containerName := strings.TrimSpace(azureQuantumDataPlaneString(payload["containerName"]))
	if containerName == "" {
		containerName = "quantum-data"
	}
	blobName := strings.TrimSpace(azureQuantumDataPlaneString(payload["blobName"]))
	if blobName == "" {
		blobName = "job-input.json"
	}
	expiry := time.Now().UTC().Add(2 * time.Hour)
	respondJSON(w, http.StatusOK, map[string]any{
		"containerName": containerName,
		"blobName":      blobName,
		"sasUri": fmt.Sprintf(
			"https://stackyard.local/quantum/%s/%s?sv=2026-01-15&se=%s&sig=%s",
			containerName,
			blobName,
			expiry.Format(time.RFC3339),
			azureQuantumDataPlaneStableToken(containerName, blobName),
		),
		"expiryTime": expiry.Format(time.RFC3339),
	})
}

func (s *Server) handleAzureQuantumDataPlaneJobsRoute(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureQuantumDataPlaneListJobs(w, "")
		case http.MethodPost:
			s.handleAzureQuantumDataPlaneCreateJob(w, r, "")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	jobID := strings.TrimSpace(strings.TrimSuffix(tail[0], ":cancel"))
	if jobID == "" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidJobId", "jobId is required")
		return
	}
	if strings.HasSuffix(tail[0], ":cancel") {
		if len(tail) != 1 || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureQuantumDataPlaneCancelJob(w, "", jobID)
		return
	}
	if len(tail) != 1 {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureQuantumDataPlaneGetJob(w, "", jobID)
	case http.MethodDelete:
		s.handleAzureQuantumDataPlaneDeleteJob(w, "", jobID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureQuantumDataPlaneSessionsRoute(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureQuantumDataPlaneListSessions(w)
		case http.MethodPost:
			s.handleAzureQuantumDataPlaneOpenSession(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	sessionID := strings.TrimSpace(strings.TrimSuffix(tail[0], ":close"))
	if sessionID == "" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidSessionId", "sessionId is required")
		return
	}
	if strings.HasSuffix(tail[0], ":close") {
		if len(tail) != 1 || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureQuantumDataPlaneCloseSession(w, sessionID)
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureQuantumDataPlaneGetSession(w, sessionID)
		case http.MethodDelete:
			s.handleAzureQuantumDataPlaneDeleteSession(w, sessionID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if !strings.EqualFold(tail[1], "jobs") {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureQuantumDataPlaneListJobs(w, sessionID)
		case http.MethodPost:
			s.handleAzureQuantumDataPlaneCreateJob(w, r, sessionID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	jobID := strings.TrimSpace(strings.TrimSuffix(tail[2], ":cancel"))
	if jobID == "" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidJobId", "jobId is required")
		return
	}
	if strings.HasSuffix(tail[2], ":cancel") {
		if len(tail) != 3 || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureQuantumDataPlaneCancelJob(w, sessionID, jobID)
		return
	}
	if len(tail) != 3 {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureQuantumDataPlaneGetJob(w, sessionID, jobID)
	case http.MethodDelete:
		s.handleAzureQuantumDataPlaneDeleteJob(w, sessionID, jobID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureQuantumDataPlaneCreateJob(w http.ResponseWriter, r *http.Request, sessionID string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	payload, ok := azureQuantumDataPlaneJSONMap(body, w)
	if !ok {
		return
	}
	providerID := strings.TrimSpace(azureQuantumDataPlaneString(payload["providerId"]))
	target := strings.TrimSpace(azureQuantumDataPlaneString(payload["target"]))
	if providerID == "" || target == "" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "MissingRequiredParameter", "providerId and target are required")
		return
	}

	now := time.Now().UTC()
	jobID := strings.TrimSpace(azureQuantumDataPlaneString(payload["id"]))
	if jobID == "" {
		jobID = "job-" + azureQuantumDataPlaneStableToken(providerID, target, now.Format(time.RFC3339Nano))[:12]
	}
	name := strings.TrimSpace(azureQuantumDataPlaneString(payload["name"]))
	if name == "" {
		name = jobID
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if sessionID != "" {
		session := s.azureQuantumDataPlaneSessions[sessionID]
		if session == nil {
			azureQuantumDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
			return
		}
		if session.Status == "Succeeded" || session.Status == "Cancelled" {
			azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidSessionState", "session is already closed")
			return
		}
	}

	job := &azureQuantumDataPlaneJob{
		JobID:            jobID,
		Name:             name,
		SessionID:        sessionID,
		ProviderID:       providerID,
		Target:           target,
		InputDataFormat:  azureQuantumDataPlaneStringOr(payload["inputDataFormat"], "qir.v1"),
		OutputDataFormat: azureQuantumDataPlaneStringOr(payload["outputDataFormat"], "microsoft.quantum-results.v1"),
		ContainerURI:     azureQuantumDataPlaneStringOr(payload["containerUri"], "https://stackyard.local/quantum/jobs/"+jobID),
		InputParams:      azureQuantumDataPlaneObject(payload["inputParams"]),
		Status:           "Waiting",
		CreatedAt:        now,
		UpdatedAt:        now,
		BeginTime:        now,
	}
	s.azureQuantumDataPlaneJobs[jobID] = job
	respondJSON(w, http.StatusCreated, azureQuantumDataPlaneJobFixture(job))
}

func (s *Server) handleAzureQuantumDataPlaneGetJob(w http.ResponseWriter, expectedSessionID, jobID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	job := s.azureQuantumDataPlaneJobs[jobID]
	if job == nil || (expectedSessionID != "" && !strings.EqualFold(job.SessionID, expectedSessionID)) {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	if expectedSessionID != "" && s.azureQuantumDataPlaneSessions[expectedSessionID] == nil {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
		return
	}
	respondJSON(w, http.StatusOK, azureQuantumDataPlaneJobFixture(job))
}

func (s *Server) handleAzureQuantumDataPlaneListJobs(w http.ResponseWriter, sessionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if sessionID != "" && s.azureQuantumDataPlaneSessions[sessionID] == nil {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
		return
	}
	items := make([]*azureQuantumDataPlaneJob, 0)
	for _, job := range s.azureQuantumDataPlaneJobs {
		if sessionID != "" && !strings.EqualFold(job.SessionID, sessionID) {
			continue
		}
		if sessionID == "" && job.SessionID != "" {
			continue
		}
		items = append(items, job)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].JobID < items[j].JobID })

	value := make([]any, 0, len(items))
	for _, job := range items {
		value = append(value, azureQuantumDataPlaneJobFixture(job))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureQuantumDataPlaneCancelJob(w http.ResponseWriter, expectedSessionID, jobID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	job := s.azureQuantumDataPlaneJobs[jobID]
	if job == nil || (expectedSessionID != "" && !strings.EqualFold(job.SessionID, expectedSessionID)) {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	if expectedSessionID != "" && s.azureQuantumDataPlaneSessions[expectedSessionID] == nil {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
		return
	}
	if job.Status == "Cancelled" || job.Status == "Succeeded" || job.Status == "Failed" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidJobState", "job is already in a terminal state")
		return
	}
	now := time.Now().UTC()
	job.Status = "Cancelled"
	job.UpdatedAt = now
	job.EndTime = now
	respondJSON(w, http.StatusOK, azureQuantumDataPlaneJobFixture(job))
}

func (s *Server) handleAzureQuantumDataPlaneDeleteJob(w http.ResponseWriter, expectedSessionID, jobID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	job := s.azureQuantumDataPlaneJobs[jobID]
	if job != nil && (expectedSessionID == "" || strings.EqualFold(job.SessionID, expectedSessionID)) {
		delete(s.azureQuantumDataPlaneJobs, jobID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureQuantumDataPlaneOpenSession(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	payload, ok := azureQuantumDataPlaneJSONMap(body, w)
	if !ok {
		return
	}
	providerID := strings.TrimSpace(azureQuantumDataPlaneString(payload["providerId"]))
	target := strings.TrimSpace(azureQuantumDataPlaneString(payload["target"]))
	if providerID == "" || target == "" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "MissingRequiredParameter", "providerId and target are required")
		return
	}

	now := time.Now().UTC()
	sessionID := strings.TrimSpace(azureQuantumDataPlaneString(payload["id"]))
	if sessionID == "" {
		sessionID = "session-" + azureQuantumDataPlaneStableToken(providerID, target, now.Format(time.RFC3339Nano))[:12]
	}
	name := strings.TrimSpace(azureQuantumDataPlaneString(payload["name"]))
	if name == "" {
		name = sessionID
	}

	session := &azureQuantumDataPlaneSession{
		SessionID:        sessionID,
		Name:             name,
		ProviderID:       providerID,
		Target:           target,
		JobFailurePolicy: azureQuantumDataPlaneStringOr(payload["jobFailurePolicy"], "Continue"),
		Status:           "Executing",
		CreatedAt:        now,
		UpdatedAt:        now,
		BeginTime:        now,
	}
	s.providerStorageMu.Lock()
	s.azureQuantumDataPlaneSessions[sessionID] = session
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusCreated, azureQuantumDataPlaneSessionFixture(session))
}

func (s *Server) handleAzureQuantumDataPlaneGetSession(w http.ResponseWriter, sessionID string) {
	s.providerStorageMu.Lock()
	session := s.azureQuantumDataPlaneSessions[sessionID]
	s.providerStorageMu.Unlock()
	if session == nil {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
		return
	}
	respondJSON(w, http.StatusOK, azureQuantumDataPlaneSessionFixture(session))
}

func (s *Server) handleAzureQuantumDataPlaneListSessions(w http.ResponseWriter) {
	s.providerStorageMu.Lock()
	items := make([]*azureQuantumDataPlaneSession, 0, len(s.azureQuantumDataPlaneSessions))
	for _, session := range s.azureQuantumDataPlaneSessions {
		items = append(items, session)
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].SessionID < items[j].SessionID })

	value := make([]any, 0, len(items))
	for _, session := range items {
		value = append(value, azureQuantumDataPlaneSessionFixture(session))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureQuantumDataPlaneCloseSession(w http.ResponseWriter, sessionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	session := s.azureQuantumDataPlaneSessions[sessionID]
	if session == nil {
		azureQuantumDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
		return
	}
	if session.Status == "Succeeded" || session.Status == "Cancelled" {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidSessionState", "session is already closed")
		return
	}
	now := time.Now().UTC()
	session.Status = "Succeeded"
	session.UpdatedAt = now
	session.EndTime = now
	respondJSON(w, http.StatusOK, azureQuantumDataPlaneSessionFixture(session))
}

func (s *Server) handleAzureQuantumDataPlaneDeleteSession(w http.ResponseWriter, sessionID string) {
	s.providerStorageMu.Lock()
	delete(s.azureQuantumDataPlaneSessions, sessionID)
	for jobID, job := range s.azureQuantumDataPlaneJobs {
		if strings.EqualFold(job.SessionID, sessionID) {
			delete(s.azureQuantumDataPlaneJobs, jobID)
		}
	}
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func azureQuantumDataPlaneProviderFixtures() []map[string]any {
	return []map[string]any{
		{"id": "ionq", "name": "IonQ", "targets": []any{"ionq.simulator", "ionq.qpu"}},
		{"id": "microsoft-qc", "name": "Microsoft", "targets": []any{"microsoft.estimator"}},
		{"id": "quantinuum", "name": "Quantinuum", "targets": []any{"quantinuum.sim.h1-1e"}},
		{"id": "rigetti", "name": "Rigetti", "targets": []any{"rigetti.sim.qvm"}},
	}
}

func azureQuantumDataPlaneQuotaFixtures() []map[string]any {
	return []map[string]any{
		{"dimension": "Shots", "scope": "ionq.simulator", "providerId": "ionq", "utilization": 120, "limit": 10000},
		{"dimension": "Jobs", "scope": "microsoft.estimator", "providerId": "microsoft-qc", "utilization": 4, "limit": 200},
		{"dimension": "Sessions", "scope": "workspace", "providerId": "workspace", "utilization": 1, "limit": 25},
	}
}

func azureQuantumDataPlaneJobFixture(job *azureQuantumDataPlaneJob) map[string]any {
	out := map[string]any{
		"id":                 job.JobID,
		"name":               job.Name,
		"status":             job.Status,
		"providerId":         job.ProviderID,
		"target":             job.Target,
		"inputDataFormat":    job.InputDataFormat,
		"outputDataFormat":   job.OutputDataFormat,
		"containerUri":       job.ContainerURI,
		"inputParams":        azureQuantumDataPlaneCloneObject(job.InputParams),
		"creationTime":       job.CreatedAt.UTC().Format(time.RFC3339),
		"beginExecutionTime": job.BeginTime.UTC().Format(time.RFC3339),
	}
	if job.SessionID != "" {
		out["sessionId"] = job.SessionID
	}
	if !job.EndTime.IsZero() {
		out["endExecutionTime"] = job.EndTime.UTC().Format(time.RFC3339)
	}
	return out
}

func azureQuantumDataPlaneSessionFixture(session *azureQuantumDataPlaneSession) map[string]any {
	out := map[string]any{
		"id":                 session.SessionID,
		"name":               session.Name,
		"status":             session.Status,
		"providerId":         session.ProviderID,
		"target":             session.Target,
		"jobFailurePolicy":   session.JobFailurePolicy,
		"creationTime":       session.CreatedAt.UTC().Format(time.RFC3339),
		"beginExecutionTime": session.BeginTime.UTC().Format(time.RFC3339),
	}
	if !session.EndTime.IsZero() {
		out["endExecutionTime"] = session.EndTime.UTC().Format(time.RFC3339)
	}
	return out
}

func azureQuantumDataPlaneJSONMap(body []byte, w http.ResponseWriter) (map[string]any, bool) {
	if len(body) == 0 {
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureQuantumDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true
}

func azureQuantumDataPlaneStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func azureQuantumDataPlaneCloneObject(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureQuantumDataPlaneCloneObject(nested)
			continue
		}
		if values, ok := value.([]any); ok {
			cloned := make([]any, len(values))
			copy(cloned, values)
			out[key] = cloned
			continue
		}
		out[key] = value
	}
	return out
}

func azureQuantumDataPlaneObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func azureQuantumDataPlaneString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}

func azureQuantumDataPlaneStringOr(value any, fallback string) string {
	if text := strings.TrimSpace(azureQuantumDataPlaneString(value)); text != "" {
		return text
	}
	return fallback
}

func azureQuantumDataPlaneError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
