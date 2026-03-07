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

const (
	gcpSpannerMaxPageSize      = 1000
	gcpSpannerMaxSessionCount  = 100
	gcpSpannerMaxPartitions    = 200000
	gcpSpannerDefaultSessionID = "s-1"
)

var gcpSpannerReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSpannerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_spanner(w, r) {
		return true
	}

	path := normalizeGCPSpannerPath(rawRequestPath(r))
	if isGCPSpannerLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSpannerListLocations(w, r, path) {
			return true
		}
		if handleGCPSpannerGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSpannerPath(path, hasGCPSpannerHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSpannerGetSession(w, path) {
			return true
		}
		if handleGCPSpannerListSessions(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSpannerCreateSession(w, r, path) {
			return true
		}
		if handleGCPSpannerBatchCreateSessions(w, r, path) {
			return true
		}
		if handleGCPSpannerExecuteSQL(w, r, path) {
			return true
		}
		if handleGCPSpannerExecuteStreamingSQL(w, r, path) {
			return true
		}
		if handleGCPSpannerExecuteBatchDML(w, r, path) {
			return true
		}
		if handleGCPSpannerRead(w, r, path) {
			return true
		}
		if handleGCPSpannerStreamingRead(w, r, path) {
			return true
		}
		if handleGCPSpannerBeginTransaction(w, r, path) {
			return true
		}
		if handleGCPSpannerCommit(w, r, path) {
			return true
		}
		if handleGCPSpannerRollback(w, r, path) {
			return true
		}
		if handleGCPSpannerPartitionQuery(w, r, path) {
			return true
		}
		if handleGCPSpannerPartitionRead(w, r, path) {
			return true
		}
		if handleGCPSpannerBatchWrite(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSpannerDeleteSession(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSpannerPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSpannerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "spanner", "spanner-apiv1", "spanner_apiv1", "cloud-spanner", "cloud_spanner", "cloudspanner", "gcp-cloud-spanner":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-spanner-apiv1") || strings.Contains(ua, "cloud.google.com/go/spanner")
}

func isGCPSpannerLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPSpannerHint(r)
}

func isGCPSpannerPath(path string, includeHint bool) bool {
	_, _, _, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || len(tail) == 0 {
		return includeHint &&
			strings.HasPrefix(path, "/gcp/v1/projects/") &&
			strings.Contains(path, "/instances/") &&
			strings.Contains(path, "/databases/")
	}
	if isGCPSpannerSessionsCollectionTail(tail) ||
		isGCPSpannerBatchCreateSessionsTail(tail) ||
		isGCPSpannerSessionTail(tail) ||
		isGCPSpannerSessionActionTail(tail, "executeSql") ||
		isGCPSpannerSessionActionTail(tail, "executeStreamingSql") ||
		isGCPSpannerSessionActionTail(tail, "executeBatchDml") ||
		isGCPSpannerSessionActionTail(tail, "read") ||
		isGCPSpannerSessionActionTail(tail, "streamingRead") ||
		isGCPSpannerSessionActionTail(tail, "beginTransaction") ||
		isGCPSpannerSessionActionTail(tail, "commit") ||
		isGCPSpannerSessionActionTail(tail, "rollback") ||
		isGCPSpannerSessionActionTail(tail, "partitionQuery") ||
		isGCPSpannerSessionActionTail(tail, "partitionRead") ||
		isGCPSpannerSessionActionTail(tail, "batchWrite") {
		return true
	}
	return false
}

func handleGCPSpannerListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSpannerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerLocationFixture(project, "us-central1"),
		gcpSpannerLocationFixture(project, "global"),
	}
	return respondGCPSpannerList(w, "locations", items, pageSize, start, path)
}

func handleGCPSpannerGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSpannerLocationFixture(project, location))
	return true
}

func handleGCPSpannerCreateSession(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, database, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || !isGCPSpannerSessionsCollectionTail(tail) {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database) {
		respondGCPSpannerNotFound(w, path, "database not found")
		return true
	}

	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	sessionBody := gcpSpannerBodyMap(body, "session")
	if len(sessionBody) == 0 {
		respondGCPSpannerInvalidArgument(w, path, "session is required")
		return true
	}

	sessionID := gcpSpannerSessionIDFromBody(sessionBody)
	if providedName := strings.TrimSpace(gcpSpannerString(sessionBody, "name")); providedName != "" {
		p, i, d, providedSessionID, validName := parseGCPSpannerSessionName(providedName)
		if !validName {
			respondGCPSpannerInvalidArgument(w, path, "session.name is invalid")
			return true
		}
		if p != project || i != instance || d != database {
			respondGCPSpannerInvalidArgument(w, path, "session.name must match database")
			return true
		}
		sessionID = providedSessionID
	}

	fixture := gcpSpannerSessionFixture(project, instance, database, sessionID, gcpSpannerBool(sessionBody["multiplexed"]))
	if labels, ok := sessionBody["labels"].(map[string]any); ok && len(labels) > 0 {
		fixture["labels"] = labels
	}
	if role := strings.TrimSpace(gcpSpannerString(sessionBody, "creatorRole")); role != "" {
		fixture["creatorRole"] = role
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPSpannerBatchCreateSessions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, database, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || !isGCPSpannerBatchCreateSessionsTail(tail) {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database) {
		respondGCPSpannerNotFound(w, path, "database not found")
		return true
	}

	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	sessionCount, ok := gcpSpannerIntFromAny(body["sessionCount"])
	if !ok {
		respondGCPSpannerInvalidArgument(w, path, "sessionCount is required")
		return true
	}
	if sessionCount <= 0 || sessionCount > gcpSpannerMaxSessionCount {
		respondGCPSpannerInvalidArgument(w, path, "sessionCount must be between 1 and 100")
		return true
	}

	template := gcpSpannerBodyMap(body, "sessionTemplate")
	multiplexed := gcpSpannerBool(template["multiplexed"])
	if providedName := strings.TrimSpace(gcpSpannerString(template, "name")); providedName != "" {
		p, i, d, _, validName := parseGCPSpannerSessionName(providedName)
		if !validName {
			respondGCPSpannerInvalidArgument(w, path, "sessionTemplate.name is invalid")
			return true
		}
		if p != project || i != instance || d != database {
			respondGCPSpannerInvalidArgument(w, path, "sessionTemplate.name must match database")
			return true
		}
	}

	items := make([]any, 0, sessionCount)
	for idx := int64(1); idx <= sessionCount; idx++ {
		sessionID := fmt.Sprintf("s-%d", idx)
		fixture := gcpSpannerSessionFixture(project, instance, database, sessionID, multiplexed)
		if labels, ok := template["labels"].(map[string]any); ok && len(labels) > 0 {
			fixture["labels"] = labels
		}
		items = append(items, fixture)
	}
	respondJSON(w, http.StatusOK, map[string]any{"session": items})
	return true
}

func handleGCPSpannerGetSession(w http.ResponseWriter, path string) bool {
	project, instance, database, sessionID, ok := parseGCPSpannerSessionPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database, sessionID) {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerSessionFixture(project, instance, database, sessionID, strings.Contains(sessionID, "mux")))
	return true
}

func handleGCPSpannerListSessions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, database, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || !isGCPSpannerSessionsCollectionTail(tail) {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database) {
		respondGCPSpannerNotFound(w, path, "database not found")
		return true
	}
	pageSize, start, valid := parseGCPSpannerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerSessionFixture(project, instance, database, "s-1", false),
		gcpSpannerSessionFixture(project, instance, database, "s-2", false),
	}
	return respondGCPSpannerList(w, "sessions", items, pageSize, start, path)
}

func handleGCPSpannerDeleteSession(w http.ResponseWriter, path string) bool {
	project, instance, database, sessionID, ok := parseGCPSpannerSessionPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerMissingResource(project, instance, database, sessionID) {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerExecuteSQL(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "executeSql")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpSpannerString(body, "sql")) == "" {
		respondGCPSpannerInvalidArgument(w, path, "sql is required")
		return true
	}
	if reason, valid := validateGCPSpannerTransactionSelector(body["transaction"], false); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if gcpSpannerBool(body["dataBoostEnabled"]) && strings.TrimSpace(gcpSpannerString(body, "partitionToken")) == "" {
		respondGCPSpannerInvalidArgument(w, path, "partitionToken is required when dataBoostEnabled is true")
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerResultSetFixture(sessionID))
	return true
}

func handleGCPSpannerExecuteStreamingSQL(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "executeStreamingSql")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpSpannerString(body, "sql")) == "" {
		respondGCPSpannerInvalidArgument(w, path, "sql is required")
		return true
	}
	if reason, valid := validateGCPSpannerTransactionSelector(body["transaction"], false); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerPartialResultSetFixture(sessionID))
	return true
}

func handleGCPSpannerExecuteBatchDML(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "executeBatchDml")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if reason, valid := validateGCPSpannerTransactionSelector(body["transaction"], true); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	statements, ok := body["statements"].([]any)
	if !ok || len(statements) == 0 {
		respondGCPSpannerInvalidArgument(w, path, "statements is required")
		return true
	}
	for idx, statementRaw := range statements {
		statement, ok := statementRaw.(map[string]any)
		if !ok {
			respondGCPSpannerInvalidArgument(w, path, fmt.Sprintf("statements[%d] must be an object", idx))
			return true
		}
		sql := strings.TrimSpace(gcpSpannerString(statement, "sql"))
		if sql == "" {
			respondGCPSpannerInvalidArgument(w, path, fmt.Sprintf("statements[%d].sql is required", idx))
			return true
		}
		if strings.Contains(strings.ToLower(sql), "abort") {
			respondGCPSpannerAborted(w, path, "transaction aborted")
			return true
		}
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerExecuteBatchDMLResponseFixture())
	return true
}

func handleGCPSpannerRead(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "read")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if reason, valid := validateGCPSpannerReadRequest(body, false); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerResultSetFixture(sessionID))
	return true
}

func handleGCPSpannerStreamingRead(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "streamingRead")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if reason, valid := validateGCPSpannerReadRequest(body, false); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerPartialResultSetFixture(sessionID))
	return true
}

func handleGCPSpannerBeginTransaction(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "beginTransaction")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	options, ok := body["options"].(map[string]any)
	if !ok || len(options) == 0 {
		respondGCPSpannerInvalidArgument(w, path, "options is required")
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerTransactionFixture(gcpSpannerTransactionIDForSession(sessionID)))
	return true
}

func handleGCPSpannerCommit(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "commit")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}

	txID := strings.TrimSpace(gcpSpannerString(body, "transactionId"))
	_, hasSingleUse := body["singleUseTransaction"]
	if txID == "" && !hasSingleUse {
		respondGCPSpannerInvalidArgument(w, path, "transactionId or singleUseTransaction is required")
		return true
	}
	if txID != "" && hasSingleUse {
		respondGCPSpannerInvalidArgument(w, path, "transactionId and singleUseTransaction are mutually exclusive")
		return true
	}
	if mutationsRaw, ok := body["mutations"]; ok {
		mutations, ok := mutationsRaw.([]any)
		if !ok {
			respondGCPSpannerInvalidArgument(w, path, "mutations must be an array")
			return true
		}
		if len(mutations) == 0 {
			respondGCPSpannerInvalidArgument(w, path, "mutations must not be empty")
			return true
		}
	}

	txToken := strings.ToLower(gcpSpannerDecodeMaybeBase64(txID))
	if strings.Contains(txToken, "stale") {
		respondGCPSpannerFailedPrecondition(w, path, "transaction is stale")
		return true
	}
	if strings.Contains(txToken, "abort") {
		respondGCPSpannerAborted(w, path, "transaction aborted")
		return true
	}

	respondJSON(w, http.StatusOK, gcpSpannerCommitResponseFixture())
	return true
}

func handleGCPSpannerRollback(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "rollback")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	txID := strings.TrimSpace(gcpSpannerString(body, "transactionId"))
	if txID == "" {
		respondGCPSpannerInvalidArgument(w, path, "transactionId is required")
		return true
	}
	if strings.Contains(strings.ToLower(gcpSpannerDecodeMaybeBase64(txID)), "stale") {
		respondGCPSpannerFailedPrecondition(w, path, "transaction is stale")
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerPartitionQuery(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "partitionQuery")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpSpannerString(body, "sql")) == "" {
		respondGCPSpannerInvalidArgument(w, path, "sql is required")
		return true
	}
	if reason, valid := validateGCPSpannerTransactionSelector(body["transaction"], true); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if reason, valid := validateGCPSpannerPartitionOptions(body["partitionOptions"]); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerPartitionResponseFixture(sessionID))
	return true
}

func handleGCPSpannerPartitionRead(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "partitionRead")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if reason, valid := validateGCPSpannerReadRequest(body, true); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if reason, valid := validateGCPSpannerPartitionOptions(body["partitionOptions"]); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerPartitionResponseFixture(sessionID))
	return true
}

func handleGCPSpannerBatchWrite(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, sessionID, ok := parseGCPSpannerSessionActionPath(path, "batchWrite")
	if !ok {
		return false
	}
	body, valid := decodeGCPSpannerJSONBody(w, r, path)
	if !valid {
		return true
	}
	if reason, valid := validateGCPSpannerMutationGroups(body["mutationGroups"]); !valid {
		respondGCPSpannerInvalidArgument(w, path, reason)
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "missing") {
		respondGCPSpannerNotFound(w, path, "session not found")
		return true
	}
	if strings.Contains(strings.ToLower(sessionID), "abort") {
		respondGCPSpannerAborted(w, path, "batch write aborted")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerBatchWriteResponseFixture())
	return true
}

func parseGCPSpannerDatabaseTail(path string) (project, instance, database string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 8 {
		return "", "", "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "instances" || parts[6] != "databases" {
		return "", "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	instance = strings.TrimSpace(parts[5])
	database = strings.TrimSpace(parts[7])
	if project == "" || instance == "" || database == "" {
		return "", "", "", nil, false
	}
	return project, instance, database, parts[8:], true
}

func isGCPSpannerSessionsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "sessions"
}

func isGCPSpannerBatchCreateSessionsTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "sessions:batchCreate"
}

func isGCPSpannerSessionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "sessions" {
		return false
	}
	sessionID := strings.TrimSpace(tail[1])
	return sessionID != "" && !strings.Contains(sessionID, ":")
}

func isGCPSpannerSessionActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "sessions" {
		return false
	}
	sessionID, parsedAction, ok := strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !ok {
		return false
	}
	return strings.TrimSpace(sessionID) != "" && parsedAction == action
}

func parseGCPSpannerSessionPath(path string) (project, instance, database, sessionID string, ok bool) {
	project, instance, database, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || !isGCPSpannerSessionTail(tail) {
		return "", "", "", "", false
	}
	sessionID = strings.TrimSpace(tail[1])
	return project, instance, database, sessionID, true
}

func parseGCPSpannerSessionActionPath(path, action string) (project, instance, database, sessionID string, ok bool) {
	project, instance, database, tail, ok := parseGCPSpannerDatabaseTail(path)
	if !ok || !isGCPSpannerSessionActionTail(tail, action) {
		return "", "", "", "", false
	}
	sessionID, _, _ = strings.Cut(strings.TrimSpace(tail[1]), ":")
	return project, instance, database, strings.TrimSpace(sessionID), true
}

func parseGCPSpannerDatabaseName(name string) (project, instance, database string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "databases" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	database = strings.TrimSpace(parts[5])
	if project == "" || instance == "" || database == "" {
		return "", "", "", false
	}
	return project, instance, database, true
}

func parseGCPSpannerSessionName(name string) (project, instance, database, sessionID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "databases" || parts[6] != "sessions" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	database = strings.TrimSpace(parts[5])
	sessionID = strings.TrimSpace(parts[7])
	if project == "" || instance == "" || database == "" || sessionID == "" || strings.Contains(sessionID, ":") {
		return "", "", "", "", false
	}
	return project, instance, database, sessionID, true
}

func parseGCPSpannerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > gcpSpannerMaxPageSize {
			respondGCPSpannerInvalidArgument(w, path, "pageSize must be a non-negative integer up to 1000")
			return 0, 0, false
		}
		pageSize = parsed
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respondGCPSpannerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func respondGCPSpannerList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSpannerInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	result := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		result = append(result, item)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           result,
		"nextPageToken": next,
	})
	return true
}

func decodeGCPSpannerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		return map[string]any{}, true
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		respondGCPSpannerInvalidArgument(w, path, "request body is invalid")
		return nil, false
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, true
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondGCPSpannerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	object, ok := payload.(map[string]any)
	if !ok {
		respondGCPSpannerInvalidArgument(w, path, "request body must be a JSON object")
		return nil, false
	}
	return object, true
}

func validateGCPSpannerReadRequest(body map[string]any, requireTransaction bool) (string, bool) {
	if strings.TrimSpace(gcpSpannerString(body, "table")) == "" {
		return "table is required", false
	}
	columns, ok := body["columns"].([]any)
	if !ok || len(columns) == 0 {
		return "columns is required", false
	}
	for idx, columnRaw := range columns {
		column, ok := columnRaw.(string)
		if !ok || strings.TrimSpace(column) == "" {
			return fmt.Sprintf("columns[%d] is invalid", idx), false
		}
	}
	if reason, valid := validateGCPSpannerKeySet(body["keySet"]); !valid {
		return reason, false
	}
	if reason, valid := validateGCPSpannerTransactionSelector(body["transaction"], requireTransaction); !valid {
		return reason, false
	}
	return "", true
}

func validateGCPSpannerMutationGroups(raw any) (string, bool) {
	groups, ok := raw.([]any)
	if !ok || len(groups) == 0 {
		return "mutationGroups is required", false
	}
	for i, groupRaw := range groups {
		group, ok := groupRaw.(map[string]any)
		if !ok {
			return fmt.Sprintf("mutationGroups[%d] must be an object", i), false
		}
		mutations, ok := group["mutations"].([]any)
		if !ok || len(mutations) == 0 {
			return fmt.Sprintf("mutationGroups[%d].mutations is required", i), false
		}
	}
	return "", true
}

func validateGCPSpannerTransactionSelector(raw any, required bool) (string, bool) {
	if raw == nil {
		if required {
			return "transaction is required", false
		}
		return "", true
	}
	selector, ok := raw.(map[string]any)
	if !ok || len(selector) == 0 {
		return "transaction must be an object", false
	}

	count := 0
	if id := strings.TrimSpace(gcpSpannerString(selector, "id")); id != "" {
		count++
	}
	if singleUse, ok := selector["singleUse"]; ok {
		object, isObject := singleUse.(map[string]any)
		if !isObject || len(object) == 0 {
			return "transaction.singleUse must be an object", false
		}
		count++
	}
	if begin, ok := selector["begin"]; ok {
		object, isObject := begin.(map[string]any)
		if !isObject || len(object) == 0 {
			return "transaction.begin must be an object", false
		}
		count++
	}
	if count == 0 {
		return "transaction selector is required", false
	}
	if count > 1 {
		return "transaction selector must include exactly one of id, singleUse, or begin", false
	}
	return "", true
}

func validateGCPSpannerPartitionOptions(raw any) (string, bool) {
	if raw == nil {
		return "", true
	}
	options, ok := raw.(map[string]any)
	if !ok {
		return "partitionOptions must be an object", false
	}
	if maxPartitions, ok := options["maxPartitions"]; ok {
		value, valid := gcpSpannerIntFromAny(maxPartitions)
		if !valid || value < 0 || value > gcpSpannerMaxPartitions {
			return "partitionOptions.maxPartitions must be between 0 and 200000", false
		}
	}
	if size, ok := options["partitionSizeBytes"]; ok {
		value, valid := gcpSpannerIntFromAny(size)
		if !valid || value < 0 {
			return "partitionOptions.partitionSizeBytes must be non-negative", false
		}
	}
	return "", true
}

func validateGCPSpannerKeySet(raw any) (string, bool) {
	keySet, ok := raw.(map[string]any)
	if !ok || len(keySet) == 0 {
		return "keySet is required", false
	}
	all := false
	if rawAll, ok := keySet["all"]; ok {
		value, valid := rawAll.(bool)
		if !valid {
			return "keySet.all must be boolean", false
		}
		all = value
	}
	keys, hasKeys := keySet["keys"].([]any)
	ranges, hasRanges := keySet["ranges"].([]any)
	if !all && (!hasKeys || len(keys) == 0) && (!hasRanges || len(ranges) == 0) {
		return "keySet must include all=true, keys, or ranges", false
	}
	return "", true
}

func gcpSpannerBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return nil
	}
	value, ok := body[key]
	if !ok {
		return nil
	}
	obj, _ := value.(map[string]any)
	return obj
}

func gcpSpannerString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	value, ok := body[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func gcpSpannerBool(raw any) bool {
	value, _ := raw.(bool)
	return value
}

func gcpSpannerIntFromAny(raw any) (int64, bool) {
	switch typed := raw.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), float64(int64(typed)) == typed
	case json.Number:
		value, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return value, true
	case string:
		value, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	default:
		return 0, false
	}
}

func isGCPSpannerMissingResource(values ...string) bool {
	for _, value := range values {
		lower := strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(lower, "missing") || strings.Contains(lower, "notfound") {
			return true
		}
	}
	return false
}

func gcpSpannerDatabaseResourceName(project, instance, database string) string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
}

func gcpSpannerSessionResourceName(project, instance, database, sessionID string) string {
	return fmt.Sprintf("%s/sessions/%s", gcpSpannerDatabaseResourceName(project, instance, database), sessionID)
}

func gcpSpannerSessionIDFromBody(session map[string]any) string {
	if session == nil {
		return gcpSpannerDefaultSessionID
	}
	if name := strings.TrimSpace(gcpSpannerString(session, "name")); name != "" {
		if _, _, _, sessionID, ok := parseGCPSpannerSessionName(name); ok {
			return sessionID
		}
	}
	return gcpSpannerDefaultSessionID
}

func gcpSpannerTransactionIDForSession(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		sessionID = gcpSpannerDefaultSessionID
	}
	return "tx-" + sessionID
}

func gcpSpannerBytesToken(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

func gcpSpannerDecodeMaybeBase64(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw
	}
	return string(decoded)
}

func gcpSpannerLocationFixture(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"labels": map[string]any{
			"service": "spanner",
		},
		"metadata": map[string]any{
			"provider": providerGCP,
			"service":  "spanner",
		},
	}
}

func gcpSpannerSessionFixture(project, instance, database, sessionID string, multiplexed bool) map[string]any {
	return map[string]any{
		"name":                   gcpSpannerSessionResourceName(project, instance, database, sessionID),
		"labels":                 map[string]any{"env": "staged"},
		"createTime":             gcpSpannerReferenceTime.Format(time.RFC3339Nano),
		"approximateLastUseTime": gcpSpannerReferenceTime.Add(2 * time.Minute).Format(time.RFC3339Nano),
		"creatorRole":            "roles/spanner.databaseUser",
		"multiplexed":            multiplexed,
	}
}

func gcpSpannerTransactionFixture(txID string) map[string]any {
	return map[string]any{
		"id":            gcpSpannerBytesToken(txID),
		"readTimestamp": gcpSpannerReferenceTime.Add(10 * time.Second).Format(time.RFC3339Nano),
	}
}

func gcpSpannerResultSetMetadataFixture(txID string) map[string]any {
	return map[string]any{
		"rowType": map[string]any{
			"fields": []any{
				map[string]any{"name": "id", "type": map[string]any{"code": "INT64"}},
				map[string]any{"name": "value", "type": map[string]any{"code": "STRING"}},
			},
		},
		"transaction": gcpSpannerTransactionFixture(txID),
	}
}

func gcpSpannerResultSetFixture(sessionID string) map[string]any {
	return map[string]any{
		"metadata": gcpSpannerResultSetMetadataFixture(gcpSpannerTransactionIDForSession(sessionID)),
		"rows": []any{
			[]any{"1", "stackyard"},
		},
		"stats": map[string]any{
			"rowCountExact": "1",
		},
	}
}

func gcpSpannerPartialResultSetFixture(sessionID string) map[string]any {
	return map[string]any{
		"metadata":     gcpSpannerResultSetMetadataFixture(gcpSpannerTransactionIDForSession(sessionID)),
		"values":       []any{"1", "stackyard"},
		"chunkedValue": false,
		"resumeToken":  gcpSpannerBytesToken("resume-1"),
		"last":         true,
	}
}

func gcpSpannerExecuteBatchDMLResponseFixture() map[string]any {
	return map[string]any{
		"resultSets": []any{
			map[string]any{
				"metadata": map[string]any{
					"rowType": map[string]any{
						"fields": []any{
							map[string]any{"name": "row_count", "type": map[string]any{"code": "INT64"}},
						},
					},
				},
				"stats": map[string]any{
					"rowCountExact": "1",
				},
			},
		},
		"status": map[string]any{
			"code":    0,
			"message": "OK",
		},
	}
}

func gcpSpannerCommitResponseFixture() map[string]any {
	return map[string]any{
		"commitTimestamp": gcpSpannerReferenceTime.Add(30 * time.Second).Format(time.RFC3339Nano),
		"commitStats": map[string]any{
			"mutationCount": "1",
		},
	}
}

func gcpSpannerPartitionResponseFixture(sessionID string) map[string]any {
	return map[string]any{
		"partitions": []any{
			map[string]any{"partitionToken": gcpSpannerBytesToken("partition-1")},
			map[string]any{"partitionToken": gcpSpannerBytesToken("partition-2")},
		},
		"transaction": gcpSpannerTransactionFixture(gcpSpannerTransactionIDForSession(sessionID)),
	}
}

func gcpSpannerBatchWriteResponseFixture() map[string]any {
	return map[string]any{
		"indexes": []any{0},
		"status": map[string]any{
			"code":    0,
			"message": "OK",
		},
		"commitTimestamp": gcpSpannerReferenceTime.Add(45 * time.Second).Format(time.RFC3339Nano),
	}
}

func respondGCPSpannerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpannerError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpannerNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpannerError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpannerFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpannerError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpannerAborted(w http.ResponseWriter, path, message string) {
	respondGCPSpannerError(w, http.StatusConflict, "Aborted", path, message)
}

func respondGCPSpannerError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_spanner(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "spanner") {
		return false
	}
	project, location, _, ok := parseGCPContractProbeServicePath(path)
	if !ok {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpannerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     gcpSpannerSessionResourceName(project, "stackyard-instance", "stackyard-db", "sample"),
			"service":  "spanner",
			"provider": providerGCP,
			"path":     path,
			"location": location,
		})
		return true
	}
	return false
}
