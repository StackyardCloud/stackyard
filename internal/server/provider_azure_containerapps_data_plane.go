package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureContainerAppsDataPlanePrefix = "/azure/containerapps-data-plane/2025-10-02-preview"

type azureContainerAppsDataPlaneSession struct {
	Identifier     string
	CreatedAt      time.Time
	LastAccessedAt time.Time
	ExpireAt       time.Time
	ETag           string
}

type azureContainerAppsDataPlaneExecution struct {
	Identifier                  string
	ExecutionID                 string
	CreatedAt                   time.Time
	LastUpdatedAt               time.Time
	Status                      string
	TimeoutInSeconds            int
	CodeInputType               string
	ExecutionType               string
	OperationID                 string
	Stdout                      string
	Stderr                      string
	ExecutionResult             any
	ExecutionTimeInMilliseconds int
}

type azureContainerAppsDataPlaneFile struct {
	Identifier     string
	Directory      string
	Name           string
	ContentType    string
	Content        []byte
	LastModifiedAt time.Time
}

func (s *Server) handleAzureContainerAppsDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContainerAppsDataPlanePrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureContainerAppsDataPlanePrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "metadata":
		s.handleAzureContainerAppsDataPlaneMetadata(w, r, path, segments[1:])
		return true
	case "session":
		s.handleAzureContainerAppsDataPlaneSession(w, r, path, segments[1:])
		return true
	case "listsessions":
		s.handleAzureContainerAppsDataPlaneListSessions(w, r, path, segments[1:])
		return true
	case "executions":
		s.handleAzureContainerAppsDataPlaneExecutions(w, r, path, segments[1:])
		return true
	case "files":
		s.handleAzureContainerAppsDataPlaneFiles(w, r, path, segments[1:])
		return true
	default:
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureContainerAppsDataPlaneMetadata(w http.ResponseWriter, r *http.Request, _ string, tail []string) {
	if len(tail) != 0 {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, azureContainerAppsDataPlaneMetadataFixture())
}

func (s *Server) handleAzureContainerAppsDataPlaneSession(w http.ResponseWriter, r *http.Request, _ string, tail []string) {
	if len(tail) != 0 {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	identifier, ok := azureContainerAppsDataPlaneRequireIdentifier(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.providerStorageMu.Lock()
		session := s.azureContainerAppsDataPlaneSessions[identifier]
		if session != nil {
			session.LastAccessedAt = time.Now().UTC()
		}
		s.providerStorageMu.Unlock()
		if session == nil {
			azureContainerAppsDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
			return
		}
		respondJSON(w, http.StatusOK, azureContainerAppsDataPlaneSessionFixture(session))
	case http.MethodDelete:
		s.providerStorageMu.Lock()
		_, exists := s.azureContainerAppsDataPlaneSessions[identifier]
		if exists {
			delete(s.azureContainerAppsDataPlaneSessions, identifier)
			for key, execution := range s.azureContainerAppsDataPlaneExecutions {
				if execution.Identifier == identifier {
					delete(s.azureContainerAppsDataPlaneExecutions, key)
				}
			}
			for key, file := range s.azureContainerAppsDataPlaneFiles {
				if file.Identifier == identifier {
					delete(s.azureContainerAppsDataPlaneFiles, key)
				}
			}
		}
		s.providerStorageMu.Unlock()
		if !exists {
			azureContainerAppsDataPlaneError(w, http.StatusNotFound, "SessionNotFound", "session not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureContainerAppsDataPlaneListSessions(w http.ResponseWriter, r *http.Request, _ string, tail []string) {
	if len(tail) != 0 {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.providerStorageMu.Lock()
	items := make([]*azureContainerAppsDataPlaneSession, 0, len(s.azureContainerAppsDataPlaneSessions))
	for _, session := range s.azureContainerAppsDataPlaneSessions {
		items = append(items, session)
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Identifier < items[j].Identifier })

	skip := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("skip")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "skip must be a non-negative integer")
			return
		}
		skip = value
	}
	if skip > len(items) {
		skip = len(items)
	}

	sessions := make([]any, 0, len(items)-skip)
	for _, session := range items[skip:] {
		sessions = append(sessions, azureContainerAppsDataPlaneSessionFixture(session))
	}
	respondJSON(w, http.StatusOK, map[string]any{"sessions": sessions})
}

func (s *Server) handleAzureContainerAppsDataPlaneExecutions(w http.ResponseWriter, r *http.Request, _ string, tail []string) {
	identifier, ok := azureContainerAppsDataPlaneRequireIdentifier(w, r)
	if !ok {
		return
	}
	switch {
	case len(tail) == 0 && r.Method == http.MethodPost:
		s.handleAzureContainerAppsDataPlaneExecute(w, r, identifier)
	case len(tail) == 1 && r.Method == http.MethodGet:
		s.handleAzureContainerAppsDataPlaneExecutionGet(w, identifier, tail[0])
	case len(tail) == 0 || len(tail) == 1:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureContainerAppsDataPlaneExecute(w http.ResponseWriter, r *http.Request, identifier string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return
	}
	timeout, ok := azureContainerAppsDataPlaneInt(payload["timeoutInSeconds"])
	if !ok || timeout <= 0 {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "timeoutInSeconds is required")
		return
	}
	if strings.TrimSpace(azureContainerAppsDataPlaneString(payload["shellCommand"])) != "" && len(azureContainerAppsDataPlaneStringSlice(payload["execCommandAndArgs"])) > 0 {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "shellCommand and execCommandAndArgs cannot both be supplied")
		return
	}

	now := time.Now().UTC()
	executionID := "execution-" + azureContainerAppsDataPlaneStableToken(identifier, now.Format(time.RFC3339Nano))
	codeInputType := azureContainerAppsDataPlaneStringOr(payload["codeInputType"], "inline")
	executionType := azureContainerAppsDataPlaneStringOr(payload["executionType"], "synchronous")
	stdout := azureContainerAppsDataPlaneExecutionStdout(payload)
	execution := &azureContainerAppsDataPlaneExecution{
		Identifier:                  identifier,
		ExecutionID:                 executionID,
		CreatedAt:                   now,
		LastUpdatedAt:               now,
		Status:                      "Succeeded",
		TimeoutInSeconds:            timeout,
		CodeInputType:               codeInputType,
		ExecutionType:               executionType,
		OperationID:                 strings.TrimSpace(r.Header.Get("operation-id")),
		Stdout:                      stdout,
		Stderr:                      "",
		ExecutionResult:             azureContainerAppsDataPlaneExecutionResultFixture(payload),
		ExecutionTimeInMilliseconds: 25,
	}

	s.providerStorageMu.Lock()
	s.azureContainerAppsDataPlaneSessions[identifier] = s.azureContainerAppsDataPlaneEnsureSessionLocked(identifier, now)
	s.azureContainerAppsDataPlaneExecutions[azureContainerAppsDataPlaneExecutionKey(identifier, executionID)] = execution
	s.providerStorageMu.Unlock()

	w.Header().Set("Operation-Location", azureContainerAppsDataPlanePrefix+"/executions/"+executionID+"?api-version=2025-10-02-preview&identifier="+identifier)
	if execution.OperationID != "" {
		w.Header().Set("operation-id", execution.OperationID)
	}
	respondJSON(w, http.StatusOK, azureContainerAppsDataPlaneExecutionFixture(execution))
}

func (s *Server) handleAzureContainerAppsDataPlaneExecutionGet(w http.ResponseWriter, identifier, executionID string) {
	if strings.TrimSpace(executionID) == "" {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "executionId is required")
		return
	}
	key := azureContainerAppsDataPlaneExecutionKey(identifier, executionID)
	s.providerStorageMu.Lock()
	execution := s.azureContainerAppsDataPlaneExecutions[key]
	if session := s.azureContainerAppsDataPlaneSessions[identifier]; session != nil {
		session.LastAccessedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	if execution == nil {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "ExecutionNotFound", "execution not found")
		return
	}
	respondJSON(w, http.StatusOK, azureContainerAppsDataPlaneExecutionFixture(execution))
}

func (s *Server) handleAzureContainerAppsDataPlaneFiles(w http.ResponseWriter, r *http.Request, _ string, tail []string) {
	identifier, ok := azureContainerAppsDataPlaneRequireIdentifier(w, r)
	if !ok {
		return
	}
	switch {
	case len(tail) == 0 && r.Method == http.MethodPost:
		s.handleAzureContainerAppsDataPlaneFileUpload(w, r, identifier)
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureContainerAppsDataPlaneFileList(w, r, identifier)
	case len(tail) == 1 && r.Method == http.MethodGet:
		s.handleAzureContainerAppsDataPlaneFileGet(w, r, identifier, tail[0])
	case len(tail) == 1 && r.Method == http.MethodDelete:
		s.handleAzureContainerAppsDataPlaneFileDelete(w, r, identifier, tail[0])
	case len(tail) == 2 && strings.EqualFold(tail[1], "content") && r.Method == http.MethodGet:
		s.handleAzureContainerAppsDataPlaneFileContentGet(w, r, identifier, tail[0])
	case len(tail) == 0 || len(tail) == 1 || len(tail) == 2:
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureContainerAppsDataPlaneFileUpload(w http.ResponseWriter, r *http.Request, identifier string) {
	if err := r.ParseMultipartForm(25 << 20); err != nil {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "multipart form with file is required")
		return
	}
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "file is required")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read uploaded file")
		return
	}

	now := time.Now().UTC()
	entry := &azureContainerAppsDataPlaneFile{
		Identifier:     identifier,
		Directory:      azureContainerAppsDataPlaneNormalizeDirectory(r.URL.Query().Get("path")),
		Name:           fileHeader.Filename,
		ContentType:    azureContainerAppsDataPlaneMultipartContentType(fileHeader),
		Content:        append([]byte(nil), content...),
		LastModifiedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureContainerAppsDataPlaneSessions[identifier] = s.azureContainerAppsDataPlaneEnsureSessionLocked(identifier, now)
	s.azureContainerAppsDataPlaneFiles[azureContainerAppsDataPlaneFileKey(identifier, entry.Directory, entry.Name)] = entry
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureContainerAppsDataPlaneFileFixture(entry))
}

func (s *Server) handleAzureContainerAppsDataPlaneFileList(w http.ResponseWriter, r *http.Request, identifier string) {
	directory := azureContainerAppsDataPlaneNormalizeDirectory(r.URL.Query().Get("path"))
	recursive := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("recursive")), "true")

	s.providerStorageMu.Lock()
	if session := s.azureContainerAppsDataPlaneSessions[identifier]; session != nil {
		session.LastAccessedAt = time.Now().UTC()
	}
	items := make([]*azureContainerAppsDataPlaneFile, 0)
	for _, file := range s.azureContainerAppsDataPlaneFiles {
		if file.Identifier != identifier {
			continue
		}
		if recursive {
			if strings.HasPrefix(file.Directory, directory) {
				items = append(items, file)
			}
			continue
		}
		if file.Directory == directory {
			items = append(items, file)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if items[i].Directory == items[j].Directory {
			return items[i].Name < items[j].Name
		}
		return items[i].Directory < items[j].Directory
	})

	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureContainerAppsDataPlaneFileFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value, "nextLink": nil})
}

func (s *Server) handleAzureContainerAppsDataPlaneFileGet(w http.ResponseWriter, r *http.Request, identifier, name string) {
	file := s.azureContainerAppsDataPlaneLookupFile(identifier, r.URL.Query().Get("path"), name)
	if file == nil {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "FileNotFound", "file not found")
		return
	}
	respondJSON(w, http.StatusOK, azureContainerAppsDataPlaneFileFixture(file))
}

func (s *Server) handleAzureContainerAppsDataPlaneFileContentGet(w http.ResponseWriter, r *http.Request, identifier, name string) {
	file := s.azureContainerAppsDataPlaneLookupFile(identifier, r.URL.Query().Get("path"), name)
	if file == nil {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "FileNotFound", "file not found")
		return
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}

func (s *Server) handleAzureContainerAppsDataPlaneFileDelete(w http.ResponseWriter, r *http.Request, identifier, name string) {
	directory := azureContainerAppsDataPlaneNormalizeDirectory(r.URL.Query().Get("path"))
	key := azureContainerAppsDataPlaneFileKey(identifier, directory, name)

	s.providerStorageMu.Lock()
	if session := s.azureContainerAppsDataPlaneSessions[identifier]; session != nil {
		session.LastAccessedAt = time.Now().UTC()
	}
	_, exists := s.azureContainerAppsDataPlaneFiles[key]
	if exists {
		delete(s.azureContainerAppsDataPlaneFiles, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		azureContainerAppsDataPlaneError(w, http.StatusNotFound, "FileNotFound", "file not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) azureContainerAppsDataPlaneLookupFile(identifier, rawDirectory, name string) *azureContainerAppsDataPlaneFile {
	directory := azureContainerAppsDataPlaneNormalizeDirectory(rawDirectory)
	key := azureContainerAppsDataPlaneFileKey(identifier, directory, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	file := s.azureContainerAppsDataPlaneFiles[key]
	if file == nil {
		return nil
	}
	if session := s.azureContainerAppsDataPlaneSessions[identifier]; session != nil {
		session.LastAccessedAt = time.Now().UTC()
	}
	return file
}

func (s *Server) azureContainerAppsDataPlaneEnsureSessionLocked(identifier string, now time.Time) *azureContainerAppsDataPlaneSession {
	session := s.azureContainerAppsDataPlaneSessions[identifier]
	if session == nil {
		session = &azureContainerAppsDataPlaneSession{
			Identifier: identifier,
			CreatedAt:  now,
		}
		s.azureContainerAppsDataPlaneSessions[identifier] = session
	}
	session.LastAccessedAt = now
	session.ExpireAt = now.Add(30 * time.Minute)
	session.ETag = azureContainerAppsDataPlaneETag(identifier, now.Format(time.RFC3339Nano))
	return session
}

func azureContainerAppsDataPlaneMetadataFixture() map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"runtimes": []any{
			map[string]any{
				"name":           "python",
				"version":        "3.12",
				"displayName":    "Python 3.12",
				"default":        true,
				"language":       "python",
				"supportedModes": []any{"inline", "shell"},
				"jupyterKernel":  "python3",
			},
		},
		"preInstalledPackages": []any{
			map[string]any{"name": "numpy", "version": "2.1.0"},
			map[string]any{"name": "pandas", "version": "2.2.3"},
		},
		"createdAt":     now.Add(-24 * time.Hour).Format(time.RFC3339),
		"lastUpdatedAt": now.Format(time.RFC3339),
		"jupyterKernel": "python3",
	}
}

func azureContainerAppsDataPlaneSessionFixture(session *azureContainerAppsDataPlaneSession) map[string]any {
	return map[string]any{
		"identifier":     session.Identifier,
		"createdAt":      session.CreatedAt.Format(time.RFC3339),
		"lastAccessedAt": session.LastAccessedAt.Format(time.RFC3339),
		"expireAt":       session.ExpireAt.Format(time.RFC3339),
		"etag":           session.ETag,
	}
}

func azureContainerAppsDataPlaneExecutionFixture(execution *azureContainerAppsDataPlaneExecution) map[string]any {
	return map[string]any{
		"id":                          execution.ExecutionID,
		"identifier":                  execution.Identifier,
		"status":                      execution.Status,
		"createdAt":                   execution.CreatedAt.Format(time.RFC3339),
		"lastUpdatedAt":               execution.LastUpdatedAt.Format(time.RFC3339),
		"timeoutInSeconds":            execution.TimeoutInSeconds,
		"codeInputType":               execution.CodeInputType,
		"executionType":               execution.ExecutionType,
		"operationId":                 execution.OperationID,
		"stdout":                      execution.Stdout,
		"stderr":                      execution.Stderr,
		"executionResult":             execution.ExecutionResult,
		"executionTimeInMilliseconds": execution.ExecutionTimeInMilliseconds,
	}
}

func azureContainerAppsDataPlaneFileFixture(file *azureContainerAppsDataPlaneFile) map[string]any {
	return map[string]any{
		"name":           file.Name,
		"directory":      file.Directory,
		"type":           "file",
		"contentType":    file.ContentType,
		"sizeInBytes":    len(file.Content),
		"lastModifiedAt": file.LastModifiedAt.Format(time.RFC3339),
	}
}

func azureContainerAppsDataPlaneError(w http.ResponseWriter, status int, code, message string) {
	if strings.TrimSpace(code) == "" {
		code = "InvalidRequest"
	}
	if strings.TrimSpace(message) == "" {
		message = "invalid request"
	}
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureContainerAppsDataPlaneRequireIdentifier(w http.ResponseWriter, r *http.Request) (string, bool) {
	identifier := strings.TrimSpace(r.URL.Query().Get("identifier"))
	if identifier == "" {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "identifier is required")
		return "", false
	}
	if len(identifier) < 3 || len(identifier) > 64 {
		azureContainerAppsDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "identifier length must be between 3 and 64 characters")
		return "", false
	}
	return identifier, true
}

func azureContainerAppsDataPlaneExecutionStdout(payload map[string]any) string {
	if shell := strings.TrimSpace(azureContainerAppsDataPlaneString(payload["shellCommand"])); shell != "" {
		return "executed shell command: " + shell
	}
	if args := azureContainerAppsDataPlaneStringSlice(payload["execCommandAndArgs"]); len(args) > 0 {
		return "executed command: " + strings.Join(args, " ")
	}
	if code := strings.TrimSpace(azureContainerAppsDataPlaneString(payload["code"])); code != "" {
		return "executed code: " + code
	}
	return "executed session operation"
}

func azureContainerAppsDataPlaneExecutionResultFixture(payload map[string]any) any {
	if code := strings.TrimSpace(azureContainerAppsDataPlaneString(payload["code"])); code != "" {
		return map[string]any{
			"type":  "text",
			"value": code,
		}
	}
	if shell := strings.TrimSpace(azureContainerAppsDataPlaneString(payload["shellCommand"])); shell != "" {
		return map[string]any{
			"type":  "shell",
			"value": shell,
		}
	}
	if args := azureContainerAppsDataPlaneStringSlice(payload["execCommandAndArgs"]); len(args) > 0 {
		return map[string]any{
			"type":  "command",
			"value": strings.Join(args, " "),
		}
	}
	return map[string]any{
		"type":  "none",
		"value": nil,
	}
}

func azureContainerAppsDataPlaneExecutionKey(identifier, executionID string) string {
	return strings.ToLower(identifier + "|" + executionID)
}

func azureContainerAppsDataPlaneFileKey(identifier, directory, name string) string {
	return strings.ToLower(identifier + "|" + directory + "|" + name)
}

func azureContainerAppsDataPlaneNormalizeDirectory(raw string) string {
	directory := strings.TrimSpace(raw)
	directory = strings.Trim(directory, "/")
	if directory == "" {
		return "/"
	}
	return "/" + directory
}

func azureContainerAppsDataPlaneStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func azureContainerAppsDataPlaneETag(parts ...string) string {
	return `"` + azureContainerAppsDataPlaneStableToken(parts...) + `"`
}

func azureContainerAppsDataPlaneString(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func azureContainerAppsDataPlaneStringOr(value any, fallback string) string {
	if out := strings.TrimSpace(azureContainerAppsDataPlaneString(value)); out != "" {
		return out
	}
	return fallback
}

func azureContainerAppsDataPlaneStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str := strings.TrimSpace(azureContainerAppsDataPlaneString(item)); str != "" {
			out = append(out, str)
		}
	}
	return out
}

func azureContainerAppsDataPlaneInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func azureContainerAppsDataPlaneMultipartContentType(header *multipart.FileHeader) string {
	if header == nil {
		return "application/octet-stream"
	}
	if value := strings.TrimSpace(header.Header.Get("Content-Type")); value != "" {
		return value
	}
	return "application/octet-stream"
}
