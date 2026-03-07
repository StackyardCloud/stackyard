package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const gcpWebRiskGRPCPathPrefix = "/gcp/google.cloud.webrisk.v1.WebRiskService/"

var gcpWebRiskReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPWebRiskRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_webrisk(w, r) {
		return true
	}

	path := normalizeGCPWebRiskPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpWebRiskGRPCPathPrefix) {
		return handleGCPWebRiskGRPCBridge(w, r, path)
	}

	if !isGCPWebRiskRESTPath(path, hasGCPWebRiskHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPWebRiskComputeThreatListDiff(w, r, path) {
			return true
		}
		if handleGCPWebRiskSearchUris(w, r, path) {
			return true
		}
		if handleGCPWebRiskSearchHashes(w, r, path) {
			return true
		}
		if handleGCPWebRiskListOperations(w, r, path) {
			return true
		}
		if handleGCPWebRiskGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPWebRiskComputeThreatListDiff(w, r, path) {
			return true
		}
		if handleGCPWebRiskSearchUris(w, r, path) {
			return true
		}
		if handleGCPWebRiskSearchHashes(w, r, path) {
			return true
		}
		if handleGCPWebRiskCreateSubmission(w, r, path) {
			return true
		}
		if handleGCPWebRiskSubmitURI(w, r, path) {
			return true
		}
		if handleGCPWebRiskCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPWebRiskDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPWebRiskPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPWebRiskHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "webrisk",
		"webrisk-apiv1",
		"webrisk_apiv1",
		"web-risk",
		"web_risk",
		"gcp-webrisk",
		"gcp-web-risk":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-webrisk-apiv1") || strings.Contains(ua, "cloud.google.com/go/webrisk")
}

func isGCPWebRiskRESTPath(path string, includeHint bool) bool {
	if isGCPWebRiskGlobalMethodPath(path) || isGCPWebRiskGRPCPath(path) {
		return true
	}
	if _, tail, ok := parseGCPWebRiskProjectTail(path); ok {
		if isGCPWebRiskSubmissionsCollectionTail(tail) ||
			isGCPWebRiskSubmitURITail(tail) ||
			isGCPWebRiskOperationsCollectionTail(tail) ||
			isGCPWebRiskOperationResourceTail(tail) ||
			isGCPWebRiskOperationActionTail(tail, "cancel") {
			return true
		}
	}
	if includeHint && strings.HasPrefix(path, "/gcp/v1/projects/") {
		return true
	}
	return false
}

func isGCPWebRiskGRPCPath(path string) bool {
	return strings.HasPrefix(path, gcpWebRiskGRPCPathPrefix)
}

func isGCPWebRiskGlobalMethodPath(path string) bool {
	switch path {
	case "/gcp/v1/threatLists:computeDiff",
		"/gcp/v1/uris:search",
		"/gcp/v1/hashes:search":
		return true
	default:
		return false
	}
}

func handleGCPWebRiskComputeThreatListDiff(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/threatLists:computeDiff" {
		return false
	}
	body := map[string]any{}
	if r.Method == http.MethodGet {
		body = gcpWebRiskComputeDiffBodyFromQuery(r)
	} else {
		var valid bool
		body, valid = decodeGCPWebRiskJSONBodyRequired(w, r, path)
		if !valid {
			return true
		}
	}

	threatType, ok := gcpWebRiskRequiredThreatType(w, path, body, "threatType", "threat_type")
	if !ok {
		return true
	}
	constraints, ok := gcpWebRiskObject(body, "constraints")
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "constraints is required")
		return true
	}
	if !validateGCPWebRiskDiffConstraints(w, path, constraints) {
		return true
	}

	respondJSON(w, http.StatusOK, gcpWebRiskComputeThreatListDiffResponse(threatType))
	return true
}

func handleGCPWebRiskSearchUris(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/uris:search" {
		return false
	}
	body := map[string]any{}
	if r.Method == http.MethodGet {
		body = gcpWebRiskSearchURIsBodyFromQuery(r)
	} else {
		var valid bool
		body, valid = decodeGCPWebRiskJSONBodyRequired(w, r, path)
		if !valid {
			return true
		}
	}

	uri := strings.TrimSpace(gcpWebRiskString(body, "uri"))
	if uri == "" {
		respondGCPWebRiskInvalidArgument(w, path, "uri is required")
		return true
	}
	if !isGCPWebRiskURI(uri) {
		respondGCPWebRiskInvalidArgument(w, path, "uri must be an absolute http(s) URI")
		return true
	}
	threatTypes, ok := gcpWebRiskRequiredThreatTypes(w, path, body, "threatTypes", "threat_types")
	if !ok {
		return true
	}

	respondJSON(w, http.StatusOK, gcpWebRiskSearchUrisResponse(uri, threatTypes))
	return true
}

func handleGCPWebRiskSearchHashes(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/hashes:search" {
		return false
	}
	body := map[string]any{}
	if r.Method == http.MethodGet {
		body = gcpWebRiskSearchHashesBodyFromQuery(r)
	} else {
		var valid bool
		body, valid = decodeGCPWebRiskJSONBodyRequired(w, r, path)
		if !valid {
			return true
		}
	}

	hashPrefix, ok := gcpWebRiskRequiredHashPrefix(w, path, body, "hashPrefix", "hash_prefix")
	if !ok {
		return true
	}
	threatTypes, ok := gcpWebRiskRequiredThreatTypes(w, path, body, "threatTypes", "threat_types")
	if !ok {
		return true
	}

	respondJSON(w, http.StatusOK, gcpWebRiskSearchHashesResponse(hashPrefix, threatTypes))
	return true
}

func handleGCPWebRiskCreateSubmission(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebRiskProjectTail(path)
	if !ok || !isGCPWebRiskSubmissionsCollectionTail(tail) {
		return false
	}

	body, valid := decodeGCPWebRiskJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	submission := gcpWebRiskSubmissionFromCreateBody(body)
	if len(submission) == 0 {
		respondGCPWebRiskInvalidArgument(w, path, "submission is required")
		return true
	}
	if parent := strings.TrimSpace(gcpWebRiskString(body, "parent")); parent != "" {
		expectedParent := fmt.Sprintf("projects/%s", project)
		if parent != expectedParent {
			respondGCPWebRiskInvalidArgument(w, path, "parent must match requested project")
			return true
		}
	}
	uri, ok := validateGCPWebRiskSubmissionURI(w, path, submission, "submission")
	if !ok {
		return true
	}

	respondJSON(w, http.StatusOK, gcpWebRiskSubmission(uri))
	return true
}

func handleGCPWebRiskSubmitURI(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebRiskProjectTail(path)
	if !ok || !isGCPWebRiskSubmitURITail(tail) {
		return false
	}

	body, valid := decodeGCPWebRiskJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}

	parent := strings.TrimSpace(gcpWebRiskString(body, "parent"))
	if parent == "" {
		respondGCPWebRiskInvalidArgument(w, path, "parent is required")
		return true
	}
	expectedParent := fmt.Sprintf("projects/%s", project)
	if parent != expectedParent {
		respondGCPWebRiskInvalidArgument(w, path, "parent must match requested project")
		return true
	}
	submission, ok := gcpWebRiskObject(body, "submission")
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "submission is required")
		return true
	}
	uri, ok := validateGCPWebRiskSubmissionURI(w, path, submission, "submission")
	if !ok {
		return true
	}
	if rawThreatInfo, exists := body["threatInfo"]; exists {
		if _, ok := rawThreatInfo.(map[string]any); !ok {
			respondGCPWebRiskInvalidArgument(w, path, "threatInfo must be an object")
			return true
		}
	}
	if rawThreatDiscovery, exists := body["threatDiscovery"]; exists {
		if _, ok := rawThreatDiscovery.(map[string]any); !ok {
			respondGCPWebRiskInvalidArgument(w, path, "threatDiscovery must be an object")
			return true
		}
	}

	respondJSON(w, http.StatusOK, gcpWebRiskOperation(project, "submitUri.op-1", uri, false))
	return true
}

func handleGCPWebRiskListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebRiskProjectTail(path)
	if !ok || !isGCPWebRiskOperationsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPWebRiskPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpWebRiskOperation(project, "submitUri.op-1", "http://phish.stackyard.test/submit", false),
		gcpWebRiskOperation(project, "submitUri.done-2", "http://malware.stackyard.test/hash", true),
	}
	return respondGCPWebRiskList(w, "operations", items, pageSize, start, path)
}

func handleGCPWebRiskGetOperation(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPWebRiskProjectTail(path)
	if !ok || !isGCPWebRiskOperationResourceTail(tail) {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	if isGCPWebRiskMissingID(operationID) {
		respondGCPWebRiskNotFound(w, path, "operation not found")
		return true
	}
	done := strings.Contains(strings.ToLower(operationID), "done")
	respondJSON(w, http.StatusOK, gcpWebRiskOperation(project, operationID, "http://phish.stackyard.test/submit", done))
	return true
}

func handleGCPWebRiskCancelOperation(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPWebRiskProjectTail(path)
	if !ok || !isGCPWebRiskOperationActionTail(tail, "cancel") {
		return false
	}
	operationID, _, _ := parseGCPWebRiskOperationActionTail(tail)
	if isGCPWebRiskMissingID(operationID) {
		respondGCPWebRiskNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("projects/%s/operations/%s", project, operationID),
	})
	return true
}

func handleGCPWebRiskDeleteOperation(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPWebRiskProjectTail(path)
	if !ok || !isGCPWebRiskOperationResourceTail(tail) {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	if isGCPWebRiskMissingID(operationID) {
		respondGCPWebRiskNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPWebRiskGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	method := strings.TrimPrefix(path, gcpWebRiskGRPCPathPrefix)
	if method == "" || strings.Contains(method, "/") {
		return false
	}

	body, valid := decodeGCPWebRiskJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}

	switch method {
	case "ComputeThreatListDiff":
		threatType, ok := gcpWebRiskRequiredThreatType(w, path, body, "threatType", "threat_type")
		if !ok {
			return true
		}
		constraints, ok := gcpWebRiskObject(body, "constraints")
		if !ok {
			respondGCPWebRiskInvalidArgument(w, path, "constraints is required")
			return true
		}
		if !validateGCPWebRiskDiffConstraints(w, path, constraints) {
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebRiskComputeThreatListDiffResponse(threatType))
		return true
	case "SearchUris":
		uri := strings.TrimSpace(gcpWebRiskString(body, "uri"))
		if uri == "" {
			respondGCPWebRiskInvalidArgument(w, path, "uri is required")
			return true
		}
		if !isGCPWebRiskURI(uri) {
			respondGCPWebRiskInvalidArgument(w, path, "uri must be an absolute http(s) URI")
			return true
		}
		threatTypes, ok := gcpWebRiskRequiredThreatTypes(w, path, body, "threatTypes", "threat_types")
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebRiskSearchUrisResponse(uri, threatTypes))
		return true
	case "SearchHashes":
		hashPrefix, ok := gcpWebRiskRequiredHashPrefix(w, path, body, "hashPrefix", "hash_prefix")
		if !ok {
			return true
		}
		threatTypes, ok := gcpWebRiskRequiredThreatTypes(w, path, body, "threatTypes", "threat_types")
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebRiskSearchHashesResponse(hashPrefix, threatTypes))
		return true
	case "CreateSubmission":
		parent := strings.TrimSpace(gcpWebRiskString(body, "parent"))
		_, ok := parseGCPWebRiskProjectParent(parent)
		if !ok {
			respondGCPWebRiskInvalidArgument(w, path, "parent is required")
			return true
		}
		submission, ok := gcpWebRiskObject(body, "submission")
		if !ok {
			respondGCPWebRiskInvalidArgument(w, path, "submission is required")
			return true
		}
		uri, ok := validateGCPWebRiskSubmissionURI(w, path, submission, "submission")
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebRiskSubmission(uri))
		return true
	case "SubmitUri":
		parent := strings.TrimSpace(gcpWebRiskString(body, "parent"))
		project, ok := parseGCPWebRiskProjectParent(parent)
		if !ok {
			respondGCPWebRiskInvalidArgument(w, path, "parent is required")
			return true
		}
		submission, ok := gcpWebRiskObject(body, "submission")
		if !ok {
			respondGCPWebRiskInvalidArgument(w, path, "submission is required")
			return true
		}
		uri, ok := validateGCPWebRiskSubmissionURI(w, path, submission, "submission")
		if !ok {
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebRiskOperation(project, "submitUri.grpc-op-1", uri, false))
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func decodeGCPWebRiskJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPWebRiskInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPWebRiskInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPWebRiskJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPWebRiskJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPWebRiskInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpWebRiskComputeDiffBodyFromQuery(r *http.Request) map[string]any {
	query := r.URL.Query()
	body := map[string]any{}
	if threatType := strings.TrimSpace(query.Get("threatType")); threatType != "" {
		body["threatType"] = threatType
	} else if threatType := strings.TrimSpace(query.Get("threat_type")); threatType != "" {
		body["threatType"] = threatType
	}
	if versionToken := strings.TrimSpace(query.Get("versionToken")); versionToken != "" {
		body["versionToken"] = versionToken
	}

	constraints := map[string]any{}
	if maxDiffEntries := strings.TrimSpace(query.Get("constraints.maxDiffEntries")); maxDiffEntries != "" {
		constraints["maxDiffEntries"] = maxDiffEntries
	}
	if maxDatabaseEntries := strings.TrimSpace(query.Get("constraints.maxDatabaseEntries")); maxDatabaseEntries != "" {
		constraints["maxDatabaseEntries"] = maxDatabaseEntries
	}
	if len(constraints) > 0 {
		body["constraints"] = constraints
	}
	return body
}

func gcpWebRiskSearchURIsBodyFromQuery(r *http.Request) map[string]any {
	query := r.URL.Query()
	body := map[string]any{}
	if uri := strings.TrimSpace(query.Get("uri")); uri != "" {
		body["uri"] = uri
	}
	threatTypes := gcpWebRiskThreatTypesFromQuery(r, "threatTypes")
	if len(threatTypes) == 0 {
		threatTypes = gcpWebRiskThreatTypesFromQuery(r, "threat_types")
	}
	if len(threatTypes) > 0 {
		body["threatTypes"] = threatTypes
	}
	return body
}

func gcpWebRiskSearchHashesBodyFromQuery(r *http.Request) map[string]any {
	query := r.URL.Query()
	body := map[string]any{}
	if hashPrefix := strings.TrimSpace(query.Get("hashPrefix")); hashPrefix != "" {
		body["hashPrefix"] = hashPrefix
	} else if hashPrefix := strings.TrimSpace(query.Get("hash_prefix")); hashPrefix != "" {
		body["hashPrefix"] = hashPrefix
	}
	threatTypes := gcpWebRiskThreatTypesFromQuery(r, "threatTypes")
	if len(threatTypes) == 0 {
		threatTypes = gcpWebRiskThreatTypesFromQuery(r, "threat_types")
	}
	if len(threatTypes) > 0 {
		body["threatTypes"] = threatTypes
	}
	return body
}

func gcpWebRiskThreatTypesFromQuery(r *http.Request, key string) []any {
	values := r.URL.Query()[key]
	if len(values) == 0 {
		return nil
	}
	out := make([]any, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			value := strings.TrimSpace(part)
			if value == "" {
				continue
			}
			out = append(out, value)
		}
	}
	return out
}

func parseGCPWebRiskProjectTail(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	return project, parts[4:], true
}

func parseGCPWebRiskProjectResourceName(name string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 2 || parts[0] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[1])
	if project == "" {
		return "", nil, false
	}
	return project, parts[2:], true
}

func parseGCPWebRiskProjectParent(parent string) (project string, ok bool) {
	project, tail, parsed := parseGCPWebRiskProjectResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 0 {
		return "", false
	}
	return project, true
}

func isGCPWebRiskSubmissionsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "submissions"
}

func isGCPWebRiskSubmitURITail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "uris:submit"
}

func isGCPWebRiskOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPWebRiskOperationResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func parseGCPWebRiskOperationActionTail(tail []string) (operationID, action string, ok bool) {
	if len(tail) != 2 || tail[0] != "operations" {
		return "", "", false
	}
	id, parsedAction, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !found || id == "" || parsedAction == "" {
		return "", "", false
	}
	return id, parsedAction, true
}

func isGCPWebRiskOperationActionTail(tail []string, action string) bool {
	_, parsedAction, ok := parseGCPWebRiskOperationActionTail(tail)
	return ok && parsedAction == action
}

func parseGCPWebRiskPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPWebRiskInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPWebRiskOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPWebRiskInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func respondGCPWebRiskList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPWebRiskOutOfRange(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           items[start:end],
		"nextPageToken": next,
	})
	return true
}

func gcpWebRiskString(m map[string]any, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func gcpWebRiskObject(m map[string]any, key string) (map[string]any, bool) {
	raw, ok := m[key]
	if !ok {
		return nil, false
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	return obj, true
}

func gcpWebRiskField(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		raw, ok := m[key]
		if ok {
			return raw, true
		}
	}
	return nil, false
}

func gcpWebRiskNumberToInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int(v)) {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func gcpWebRiskRequiredThreatType(w http.ResponseWriter, path string, body map[string]any, keys ...string) (int32, bool) {
	raw, ok := gcpWebRiskField(body, keys...)
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "threatType is required")
		return 0, false
	}
	threatType, valid := gcpWebRiskThreatTypeFromAny(raw)
	if !valid {
		respondGCPWebRiskInvalidArgument(w, path, "threatType is invalid")
		return 0, false
	}
	return threatType, true
}

func gcpWebRiskRequiredThreatTypes(w http.ResponseWriter, path string, body map[string]any, keys ...string) ([]int32, bool) {
	raw, ok := gcpWebRiskField(body, keys...)
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "threatTypes is required")
		return nil, false
	}

	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		respondGCPWebRiskInvalidArgument(w, path, "threatTypes must include at least one entry")
		return nil, false
	}

	out := make([]int32, 0, len(values))
	for idx, value := range values {
		threatType, valid := gcpWebRiskThreatTypeFromAny(value)
		if !valid {
			respondGCPWebRiskInvalidArgument(w, path, fmt.Sprintf("threatTypes[%d] is invalid", idx))
			return nil, false
		}
		out = append(out, threatType)
	}
	return out, true
}

func gcpWebRiskThreatTypeFromAny(value any) (int32, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int(v)) {
			return 0, false
		}
		return gcpWebRiskNormalizeThreatType(int32(v))
	case int:
		return gcpWebRiskNormalizeThreatType(int32(v))
	case int32:
		return gcpWebRiskNormalizeThreatType(v)
	case int64:
		return gcpWebRiskNormalizeThreatType(int32(v))
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return 0, false
		}
		if parsed, err := strconv.Atoi(text); err == nil {
			return gcpWebRiskNormalizeThreatType(int32(parsed))
		}
		normalized := strings.ToUpper(text)
		normalized = strings.ReplaceAll(normalized, "-", "_")
		normalized = strings.ReplaceAll(normalized, " ", "_")
		normalized = strings.TrimPrefix(normalized, "THREAT_TYPE_")
		switch normalized {
		case "MALWARE":
			return 1, true
		case "SOCIAL_ENGINEERING":
			return 2, true
		case "UNWANTED_SOFTWARE":
			return 3, true
		case "POTENTIALLY_HARMFUL_APPLICATION":
			return 4, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

func gcpWebRiskNormalizeThreatType(threatType int32) (int32, bool) {
	if threatType <= 0 {
		return 0, false
	}
	return threatType, true
}

func gcpWebRiskRequiredHashPrefix(w http.ResponseWriter, path string, body map[string]any, keys ...string) ([]byte, bool) {
	raw, ok := gcpWebRiskField(body, keys...)
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "hashPrefix is required")
		return nil, false
	}
	hashPrefixText, ok := raw.(string)
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "hashPrefix must be base64")
		return nil, false
	}
	hashPrefixText = strings.TrimSpace(hashPrefixText)
	if hashPrefixText == "" {
		respondGCPWebRiskInvalidArgument(w, path, "hashPrefix is required")
		return nil, false
	}
	hashPrefix, ok := decodeGCPWebRiskHashPrefix(hashPrefixText)
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "hashPrefix must be base64")
		return nil, false
	}
	if len(hashPrefix) < 4 || len(hashPrefix) > 32 {
		respondGCPWebRiskInvalidArgument(w, path, "hashPrefix length must be between 4 and 32 bytes")
		return nil, false
	}
	return hashPrefix, true
}

func decodeGCPWebRiskHashPrefix(value string) ([]byte, bool) {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		inner := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
		if inner == "" {
			return nil, false
		}
		parts := strings.Fields(inner)
		out := make([]byte, 0, len(parts))
		for _, part := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || n < 0 || n > 255 {
				return nil, false
			}
			out = append(out, byte(n))
		}
		return out, true
	}
	return decodeGCPWebRiskBase64(trimmed)
}

func decodeGCPWebRiskBase64(value string) ([]byte, bool) {
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, dec := range decoders {
		out, err := dec.DecodeString(value)
		if err == nil {
			return out, true
		}
	}
	return nil, false
}

func validateGCPWebRiskDiffConstraints(w http.ResponseWriter, path string, constraints map[string]any) bool {
	maxDiffEntries, maxDiffEntriesSet, ok := gcpWebRiskOptionalInt(constraints, "maxDiffEntries", "max_diff_entries")
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "constraints.maxDiffEntries must be an integer")
		return false
	}
	if maxDiffEntriesSet && maxDiffEntries != 0 {
		if !isGCPWebRiskPowerOfTwo(maxDiffEntries) || maxDiffEntries < (1<<10) || maxDiffEntries > (1<<20) {
			respondGCPWebRiskOutOfRange(w, path, "constraints.maxDiffEntries must be a power of two between 1024 and 1048576")
			return false
		}
	}

	maxDatabaseEntries, maxDatabaseEntriesSet, ok := gcpWebRiskOptionalInt(constraints, "maxDatabaseEntries", "max_database_entries")
	if !ok {
		respondGCPWebRiskInvalidArgument(w, path, "constraints.maxDatabaseEntries must be an integer")
		return false
	}
	if maxDatabaseEntriesSet && maxDatabaseEntries != 0 {
		if !isGCPWebRiskPowerOfTwo(maxDatabaseEntries) || maxDatabaseEntries < (1<<10) || maxDatabaseEntries > (1<<20) {
			respondGCPWebRiskOutOfRange(w, path, "constraints.maxDatabaseEntries must be a power of two between 1024 and 1048576")
			return false
		}
	}

	return true
}

func gcpWebRiskOptionalInt(m map[string]any, keys ...string) (value int, present bool, ok bool) {
	raw, found := gcpWebRiskField(m, keys...)
	if !found {
		return 0, false, true
	}
	parsed, valid := gcpWebRiskNumberToInt(raw)
	if !valid {
		return 0, true, false
	}
	return parsed, true, true
}

func isGCPWebRiskPowerOfTwo(value int) bool {
	return value > 0 && (value&(value-1)) == 0
}

func gcpWebRiskSubmissionFromCreateBody(body map[string]any) map[string]any {
	if submission, ok := body["submission"].(map[string]any); ok {
		return submission
	}
	return body
}

func validateGCPWebRiskSubmissionURI(w http.ResponseWriter, path string, submission map[string]any, fieldPrefix string) (string, bool) {
	uri := strings.TrimSpace(gcpWebRiskString(submission, "uri"))
	if uri == "" {
		respondGCPWebRiskInvalidArgument(w, path, fieldPrefix+".uri is required")
		return "", false
	}
	if !isGCPWebRiskURI(uri) {
		respondGCPWebRiskInvalidArgument(w, path, fieldPrefix+".uri must be an absolute http(s) URI")
		return "", false
	}
	return uri, true
}

func isGCPWebRiskURI(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

func gcpWebRiskThreatTypesForURI(uri string) []int32 {
	lowered := strings.ToLower(strings.TrimSpace(uri))
	switch {
	case strings.Contains(lowered, "malware"):
		return []int32{1}
	case strings.Contains(lowered, "phish"):
		return []int32{2}
	default:
		return []int32{2}
	}
}

func gcpWebRiskComputeThreatListDiffResponse(threatType int32) map[string]any {
	checksum := make([]byte, 32)
	for i := range checksum {
		checksum[i] = byte(i + 1)
	}

	return map[string]any{
		"responseType":        "DIFF",
		"newVersionToken":     base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("stackyard-token-%d", threatType))),
		"checksum":            map[string]any{"sha256": base64.StdEncoding.EncodeToString(checksum)},
		"recommendedNextDiff": gcpWebRiskReferenceTime.Add(30 * time.Minute).Format(time.RFC3339),
	}
}

func gcpWebRiskSearchUrisResponse(uri string, threatTypes []int32) map[string]any {
	return map[string]any{
		"threat": map[string]any{
			"threatTypes": threatTypes,
			"expireTime":  gcpWebRiskReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		},
	}
}

func gcpWebRiskSearchHashesResponse(hashPrefix []byte, threatTypes []int32) map[string]any {
	hash := make([]byte, 32)
	copy(hash, hashPrefix)
	for i := len(hashPrefix); i < len(hash); i++ {
		hash[i] = byte(i + 1)
	}

	return map[string]any{
		"threats": []map[string]any{
			{
				"threatTypes": threatTypes,
				"hash":        base64.StdEncoding.EncodeToString(hash),
				"expireTime":  gcpWebRiskReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
			},
		},
		"negativeExpireTime": gcpWebRiskReferenceTime.Add(30 * time.Minute).Format(time.RFC3339),
	}
}

func gcpWebRiskSubmission(uri string) map[string]any {
	return map[string]any{
		"uri":         uri,
		"threatTypes": gcpWebRiskThreatTypesForURI(uri),
	}
}

func gcpWebRiskOperation(project, operationID, uri string, done bool) map[string]any {
	state := "RUNNING"
	if done {
		state = "SUCCEEDED"
	}
	operation := map[string]any{
		"name": fmt.Sprintf("projects/%s/operations/%s", project, operationID),
		"done": done,
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.webrisk.v1.SubmitUriMetadata",
			"state":      state,
			"createTime": gcpWebRiskReferenceTime.Format(time.RFC3339),
			"updateTime": gcpWebRiskReferenceTime.Add(45 * time.Second).Format(time.RFC3339),
		},
	}
	if done {
		operation["response"] = map[string]any{
			"@type":       "type.googleapis.com/google.cloud.webrisk.v1.Submission",
			"uri":         uri,
			"threatTypes": gcpWebRiskThreatTypesForURI(uri),
		}
	}
	return operation
}

func isGCPWebRiskMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "missing") || strings.Contains(id, "notfound")
}

func respondGCPWebRiskInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPWebRiskError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPWebRiskNotFound(w http.ResponseWriter, path, message string) {
	respondGCPWebRiskError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPWebRiskOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPWebRiskError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPWebRiskError(w http.ResponseWriter, status int, errType, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errType,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_webrisk(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "webrisk") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/operations/webrisk-op-1",
			"service":  "webrisk",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
