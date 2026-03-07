package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gcpRedisReferenceTime   = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpRedisInstanceIDRegex = regexp.MustCompile(`^[a-z](?:[a-z0-9-]{0,38}[a-z0-9])?$`)
)

func (s *Server) handleGCPRedisRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_redis(w, r) {
		return true
	}

	path := normalizeGCPRedisPath(rawRequestPath(r))
	if isGCPRedisLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPRedisListLocations(w, r, path) {
			return true
		}
		if handleGCPRedisGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPRedisPath(path, hasGCPRedisHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRedisListInstances(w, r, path) {
			return true
		}
		if handleGCPRedisGetInstance(w, path) {
			return true
		}
		if handleGCPRedisGetInstanceAuthString(w, path) {
			return true
		}
		if handleGCPRedisListOperations(w, r, path) {
			return true
		}
		if handleGCPRedisGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRedisCreateInstance(w, r, path) {
			return true
		}
		if handleGCPRedisUpgradeInstance(w, r, path) {
			return true
		}
		if handleGCPRedisImportInstance(w, r, path) {
			return true
		}
		if handleGCPRedisExportInstance(w, r, path) {
			return true
		}
		if handleGCPRedisFailoverInstance(w, r, path) {
			return true
		}
		if handleGCPRedisRescheduleMaintenance(w, r, path) {
			return true
		}
		if handleGCPRedisCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRedisUpdateInstance(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRedisDeleteInstance(w, path) {
			return true
		}
		if handleGCPRedisDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPRedisPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPRedisHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "redis", "redis-apiv1", "memorystore-redis", "cloud-memorystore-redis", "gcp-redis":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-redis-apiv1") || strings.Contains(ua, "cloud.google.com/go/redis/")
}

func isGCPRedisLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPRedisHint(r)
}

func isGCPRedisPath(path string, includeOperations bool) bool {
	_, _, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	if isGCPRedisInstancesCollectionTail(tail) ||
		isGCPRedisInstanceTail(tail) ||
		isGCPRedisInstanceAuthStringTail(tail) ||
		isGCPRedisInstanceActionTail(tail, "upgrade") ||
		isGCPRedisInstanceActionTail(tail, "import") ||
		isGCPRedisInstanceActionTail(tail, "export") ||
		isGCPRedisInstanceActionTail(tail, "failover") ||
		isGCPRedisInstanceActionTail(tail, "rescheduleMaintenance") {
		return true
	}
	return includeOperations && (isGCPRedisOperationsCollectionTail(tail) || isGCPRedisOperationTail(tail) || isGCPRedisOperationActionTail(tail, "cancel"))
}

func handleGCPRedisListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPRedisPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisLocationFixture(project, "us-central1"),
		gcpRedisLocationFixture(project, "global"),
	}
	return respondGCPRedisList(w, "locations", items, pageSize, start, path)
}

func handleGCPRedisGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisLocationFixture(project, location))
	return true
}

func handleGCPRedisListInstances(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisInstancesCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPRedisPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpRedisInstanceFixture(project, location, "redis-1"),
		gcpRedisInstanceFixture(project, location, "redis-2"),
	}
	if location == "-" {
		items = []map[string]any{
			gcpRedisInstanceFixture(project, "us-central1", "redis-1"),
			gcpRedisInstanceFixture(project, "us-east1", "redis-2"),
		}
	}
	return respondGCPRedisList(w, "instances", items, pageSize, start, path)
}

func handleGCPRedisGetInstance(w http.ResponseWriter, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstancePath(path)
	if !ok {
		return false
	}
	fixture := gcpRedisInstanceFixture(project, location, instanceID)
	if strings.Contains(instanceID, "importing") {
		fixture["state"] = "IMPORTING"
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRedisGetInstanceAuthString(w http.ResponseWriter, path string) bool {
	_, _, instanceID, ok := parseGCPRedisInstanceAuthStringPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"authString": "stackyard-auth-" + instanceID,
	})
	return true
}

func handleGCPRedisCreateInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisInstancesCollectionTail(tail) {
		return false
	}

	instanceID := strings.TrimSpace(r.URL.Query().Get("instanceId"))
	if instanceID == "" {
		respondGCPRedisInvalidArgument(w, path, "instanceId is required")
		return true
	}
	if !gcpRedisInstanceIDRegex.MatchString(instanceID) {
		respondGCPRedisInvalidArgument(w, path, "instanceId is invalid")
		return true
	}

	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	instance := gcpRedisBodyMap(body, "instance")
	if len(instance) == 0 {
		respondGCPRedisInvalidArgument(w, path, "instance is required")
		return true
	}

	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(instance, "name"); got != "" && got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "instance.name must match parent and instanceId")
		return true
	}

	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "createInstance."+instanceID, expectedName, "create"))
	return true
}

func handleGCPRedisUpdateInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstancePath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	instance := gcpRedisBodyMap(body, "instance")
	if len(instance) == 0 {
		respondGCPRedisInvalidArgument(w, path, "instance is required")
		return true
	}

	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(instance, "name"); got == "" || got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "instance.name must match the requested resource")
		return true
	}

	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = strings.TrimSpace(gcpRedisString(body, "updateMask"))
	}
	if mask == "" {
		respondGCPRedisInvalidArgument(w, path, "updateMask is required")
		return true
	}
	maskPaths, ok := parseGCPRedisUpdateMask(mask)
	if !ok {
		respondGCPRedisInvalidArgument(w, path, "updateMask contains unsupported paths")
		return true
	}
	if len(maskPaths) == 0 {
		respondGCPRedisInvalidArgument(w, path, "updateMask is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "updateInstance."+instanceID, expectedName, "update"))
	return true
}

func handleGCPRedisDeleteInstance(w http.ResponseWriter, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstancePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "deleteInstance."+instanceID, gcpRedisInstanceName(project, location, instanceID), "delete"))
	return true
}

func handleGCPRedisUpgradeInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstanceActionPath(path, "upgrade")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(body, "name"); got == "" || got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	if strings.TrimSpace(gcpRedisString(body, "redisVersion")) == "" {
		respondGCPRedisInvalidArgument(w, path, "redisVersion is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "upgradeInstance."+instanceID, expectedName, "upgrade"))
	return true
}

func handleGCPRedisImportInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstanceActionPath(path, "import")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(body, "name"); got == "" || got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	inputConfig, _ := body["inputConfig"].(map[string]any)
	if len(inputConfig) == 0 {
		respondGCPRedisInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	gcsSource, _ := inputConfig["gcsSource"].(map[string]any)
	if len(gcsSource) == 0 || strings.TrimSpace(gcpRedisString(gcsSource, "uri")) == "" {
		respondGCPRedisInvalidArgument(w, path, "inputConfig.gcsSource.uri is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "importInstance."+instanceID, expectedName, "import"))
	return true
}

func handleGCPRedisExportInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstanceActionPath(path, "export")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(body, "name"); got == "" || got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	outputConfig, _ := body["outputConfig"].(map[string]any)
	if len(outputConfig) == 0 {
		respondGCPRedisInvalidArgument(w, path, "outputConfig is required")
		return true
	}
	gcsDestination, _ := outputConfig["gcsDestination"].(map[string]any)
	if len(gcsDestination) == 0 || strings.TrimSpace(gcpRedisString(gcsDestination, "uri")) == "" {
		respondGCPRedisInvalidArgument(w, path, "outputConfig.gcsDestination.uri is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "exportInstance."+instanceID, expectedName, "export"))
	return true
}

func handleGCPRedisFailoverInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstanceActionPath(path, "failover")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(body, "name"); got == "" || got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	if strings.Contains(instanceID, "basic") {
		respondGCPRedisFailedPrecondition(w, path, "failover requires STANDARD_HA tier")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "failoverInstance."+instanceID, expectedName, "failover"))
	return true
}

func handleGCPRedisRescheduleMaintenance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instanceID, ok := parseGCPRedisInstanceActionPath(path, "rescheduleMaintenance")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRedisInstanceName(project, location, instanceID)
	if got := gcpRedisString(body, "name"); got == "" || got != expectedName {
		respondGCPRedisInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	rescheduleType, ok := parseGCPRedisRescheduleType(body["rescheduleType"])
	if !ok {
		respondGCPRedisInvalidArgument(w, path, "rescheduleType is required")
		return true
	}
	if rescheduleType == "SPECIFIC_TIME" && strings.TrimSpace(gcpRedisString(body, "scheduleTime")) == "" {
		respondGCPRedisInvalidArgument(w, path, "scheduleTime is required when rescheduleType is SPECIFIC_TIME")
		return true
	}
	if rescheduleType == "IMMEDIATE" && strings.Contains(instanceID, "locked") {
		respondGCPRedisFailedPrecondition(w, path, "instance maintenance is locked")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRedisOperationFixture(project, location, "rescheduleMaintenance."+instanceID, expectedName, "rescheduleMaintenance"))
	return true
}

func handleGCPRedisListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRedisPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisOperationFixture(project, location, "createInstance.redis-1", gcpRedisInstanceName(project, location, "redis-1"), "create"),
		gcpRedisOperationFixture(project, location, "updateInstance.redis-1", gcpRedisInstanceName(project, location, "redis-1"), "update"),
	}
	return respondGCPRedisList(w, "operations", items, pageSize, start, path)
}

func handleGCPRedisGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPRedisOperationPath(path)
	if !ok {
		return false
	}
	fixture := gcpRedisOperationFixture(project, location, operationID, gcpRedisInstanceName(project, location, "redis-1"), "get")
	if strings.Contains(operationID, "done") {
		fixture["done"] = true
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRedisCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, valid := decodeGCPRedisJSONBody(w, r, path); !valid {
		return true
	}
	if _, _, _, ok := parseGCPRedisOperationActionPath(path, "cancel"); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRedisDeleteOperation(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPRedisOperationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPRedisLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func isGCPRedisInstancesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "instances"
}

func isGCPRedisInstanceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "instances" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRedisInstanceAuthStringTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "instances" && strings.TrimSpace(tail[1]) != "" && tail[2] == "authString"
}

func isGCPRedisInstanceActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "instances" {
		return false
	}
	instanceID, parsedAction, found := splitGCPRedisActionSegment(tail[1])
	return found && strings.TrimSpace(instanceID) != "" && parsedAction == action
}

func isGCPRedisOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPRedisOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRedisOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	opID, parsedAction, found := splitGCPRedisActionSegment(tail[1])
	return found && strings.TrimSpace(opID) != "" && parsedAction == action
}

func parseGCPRedisInstancePath(path string) (project, location, instanceID string, ok bool) {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisInstanceTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisInstanceAuthStringPath(path string) (project, location, instanceID string, ok bool) {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisInstanceAuthStringTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisInstanceActionPath(path, action string) (project, location, instanceID string, ok bool) {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisInstanceActionTail(tail, action) {
		return "", "", "", false
	}
	instanceID, parsedAction, _ := splitGCPRedisActionSegment(tail[1])
	if parsedAction != action {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(instanceID), true
}

func parseGCPRedisOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisOperationTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisOperationActionPath(path, action string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPRedisLocationTail(path)
	if !ok || !isGCPRedisOperationActionTail(tail, action) {
		return "", "", "", false
	}
	operationID, parsedAction, _ := splitGCPRedisActionSegment(tail[1])
	if parsedAction != action {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(operationID), true
}

func splitGCPRedisActionSegment(raw string) (id, action string, ok bool) {
	segment := strings.TrimSpace(raw)
	if segment == "" {
		return "", "", false
	}
	if decoded, err := url.PathUnescape(segment); err == nil {
		segment = decoded
	}
	id, action, ok = strings.Cut(segment, ":")
	if !ok {
		return "", "", false
	}
	id = strings.TrimSpace(id)
	action = strings.TrimSpace(action)
	if id == "" || action == "" {
		return "", "", false
	}
	return id, action, true
}

func parseGCPRedisPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRedisInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPRedisInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPRedisInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPRedisList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPRedisInvalidArgument(w, path, "pageToken is out of range")
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
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": next,
	})
	return true
}

func decodeGCPRedisJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPRedisInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRedisBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpRedisString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func parseGCPRedisUpdateMask(raw string) ([]string, bool) {
	allowed := map[string]struct{}{
		"displayname":  {},
		"labels":       {},
		"redisconfig":  {},
		"redisversion": {},
		"memorysizegb": {},
		"replicacount": {},
	}
	out := make([]string, 0, 4)
	for _, item := range strings.Split(raw, ",") {
		path := strings.TrimSpace(item)
		if path == "" {
			continue
		}
		path = strings.ReplaceAll(path, ".", "_")
		normalized := strings.ReplaceAll(strings.ToLower(path), "_", "")
		if _, ok := allowed[normalized]; !ok {
			return nil, false
		}
		out = append(out, path)
	}
	if len(out) == 0 {
		return nil, false
	}
	sort.Strings(out)
	return out, true
}

func parseGCPRedisRescheduleType(raw any) (string, bool) {
	switch typed := raw.(type) {
	case string:
		value := strings.ToUpper(strings.TrimSpace(typed))
		switch value {
		case "IMMEDIATE", "SPECIFIC_TIME":
			return value, true
		default:
			return "", false
		}
	case float64:
		switch int(typed) {
		case 1:
			return "IMMEDIATE", true
		case 3:
			return "SPECIFIC_TIME", true
		default:
			return "", false
		}
	default:
		return "", false
	}
}

func gcpRedisLocationFixture(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Memorystore for Redis " + location,
		"labels": map[string]string{
			"service": "redis",
			"stage":   "emulated",
		},
	}
}

func gcpRedisInstanceName(project, location, instanceID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, location, instanceID)
}

func gcpRedisInstanceFixture(project, location, instanceID string) map[string]any {
	instanceName := gcpRedisInstanceName(project, location, instanceID)
	return map[string]any{
		"name":              instanceName,
		"displayName":       "Stackyard Redis " + instanceID,
		"labels":            map[string]string{"env": "test", "service": "redis"},
		"host":              "10.0.0.11",
		"port":              6379,
		"state":             "READY",
		"tier":              "STANDARD_HA",
		"memorySizeGb":      4,
		"redisVersion":      "REDIS_7_0",
		"authorizedNetwork": fmt.Sprintf("projects/%s/global/networks/default", project),
		"maintenancePolicy": map[string]any{
			"description": "Weekly maintenance window",
			"weeklyMaintenanceWindow": []map[string]any{
				{
					"day": "MONDAY",
					"startTime": map[string]any{
						"hours":   3,
						"minutes": 0,
					},
				},
			},
		},
		"persistenceConfig": map[string]any{
			"persistenceMode":   "RDB",
			"rdbSnapshotPeriod": "SIX_HOURS",
		},
		"createTime": gcpRedisReferenceTime.Format(time.RFC3339Nano),
	}
}

func gcpRedisOperationFixture(project, location, operationID, target, verb string) map[string]any {
	opName := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	return map[string]any{
		"name": opName,
		"done": false,
		"metadata": map[string]any{
			"@type":       "type.googleapis.com/google.cloud.redis.v1.OperationMetadata",
			"createTime":  gcpRedisReferenceTime.Format(time.RFC3339Nano),
			"target":      target,
			"verb":        verb,
			"apiVersion":  "v1",
			"requestedBy": providerGCP,
		},
		"response": map[string]any{
			"@type": "type.googleapis.com/google.cloud.redis.v1.Instance",
			"name":  target,
		},
	}
}

func respondGCPRedisInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPRedisError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPRedisFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPRedisError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPRedisError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_redis(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "redis") {
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
			"name":     "projects/stackyard/locations/us-central1/redis/sample",
			"service":  "redis",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
