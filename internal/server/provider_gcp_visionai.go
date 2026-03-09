package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	gcpVisionAIHealthCheckGRPCPrefix        = "/gcp/google.cloud.visionai.v1.HealthCheckService/"
	gcpVisionAIStreamsGRPCPrefix            = "/gcp/google.cloud.visionai.v1.StreamsService/"
	gcpVisionAIAppPlatformGRPCPrefix        = "/gcp/google.cloud.visionai.v1.AppPlatform/"
	gcpVisionAILiveVideoAnalyticsGRPCPrefix = "/gcp/google.cloud.visionai.v1.LiveVideoAnalytics/"
	gcpVisionAIWarehouseGRPCPrefix          = "/gcp/google.cloud.visionai.v1.Warehouse/"
	gcpVisionAIStreamingGRPCPrefix          = "/gcp/google.cloud.visionai.v1.StreamingService/"
)

var gcpVisionAIReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPVisionAIRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_visionai(w, r) {
		return true
	}

	path := normalizeGCPVisionAIPath(rawRequestPath(r))
	if isGCPVisionAILocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVisionAIListLocations(w, r, path) {
			return true
		}
		if handleGCPVisionAIGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpVisionAIHealthCheckGRPCPrefix) {
		return handleGCPVisionAIUnaryRPC(w, r, path, gcpVisionAIHealthCheckGRPCPrefix, handleGCPVisionAIHealthCheckMethod)
	}
	if strings.HasPrefix(path, gcpVisionAIStreamsGRPCPrefix) {
		return handleGCPVisionAIUnaryRPC(w, r, path, gcpVisionAIStreamsGRPCPrefix, handleGCPVisionAIStreamsMethod)
	}
	if strings.HasPrefix(path, gcpVisionAIAppPlatformGRPCPrefix) {
		return handleGCPVisionAIUnaryRPC(w, r, path, gcpVisionAIAppPlatformGRPCPrefix, handleGCPVisionAIAppPlatformMethod)
	}
	if strings.HasPrefix(path, gcpVisionAILiveVideoAnalyticsGRPCPrefix) {
		return handleGCPVisionAIUnaryRPC(w, r, path, gcpVisionAILiveVideoAnalyticsGRPCPrefix, handleGCPVisionAILiveVideoAnalyticsMethod)
	}
	if strings.HasPrefix(path, gcpVisionAIWarehouseGRPCPrefix) {
		return handleGCPVisionAIUnaryRPC(w, r, path, gcpVisionAIWarehouseGRPCPrefix, handleGCPVisionAIWarehouseMethod)
	}
	if strings.HasPrefix(path, gcpVisionAIStreamingGRPCPrefix) {
		return handleGCPVisionAIUnaryRPC(w, r, path, gcpVisionAIStreamingGRPCPrefix, handleGCPVisionAIStreamingMethod)
	}

	if !isGCPVisionAIPath(path, hasGCPVisionAIHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPVisionAIListOperations(w, r, path) {
			return true
		}
		if handleGCPVisionAIGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPVisionAICancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPVisionAIDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPVisionAIPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVisionAIHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "visionai",
		"visionai-apiv1",
		"visionai_apiv1",
		"vision-ai",
		"vision_ai",
		"gcp-vision-ai":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-visionai-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/visionai/apiv1")
}

func isGCPVisionAILocationRequest(r *http.Request, path string) bool {
	if !hasGCPVisionAIHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPVisionAIPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpVisionAIHealthCheckGRPCPrefix) ||
		strings.HasPrefix(path, gcpVisionAIStreamsGRPCPrefix) ||
		strings.HasPrefix(path, gcpVisionAIAppPlatformGRPCPrefix) ||
		strings.HasPrefix(path, gcpVisionAILiveVideoAnalyticsGRPCPrefix) ||
		strings.HasPrefix(path, gcpVisionAIWarehouseGRPCPrefix) ||
		strings.HasPrefix(path, gcpVisionAIStreamingGRPCPrefix) {
		return true
	}
	if !includeHint {
		return false
	}
	if _, _, ok := parseGCPVisionAIOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPVisionAIOperationPath(path); ok {
		return true
	}
	if _, _, _, action, ok := parseGCPVisionAIOperationActionPath(path); ok {
		return action == "cancel"
	}
	return false
}

func handleGCPVisionAIUnaryRPC(w http.ResponseWriter, r *http.Request, path, prefix string, handler func(http.ResponseWriter, string, string, map[string]any) bool) bool {
	if r.Method != http.MethodPost {
		return false
	}
	body, ok := decodeGCPVisionAIJSONBody(w, r, path)
	if !ok {
		return true
	}
	method := strings.TrimSpace(strings.TrimPrefix(path, prefix))
	if method == "" {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	if handler(w, path, method, body) {
		return true
	}
	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func decodeGCPVisionAIJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPVisionAIInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func handleGCPVisionAIHealthCheckMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	if method != "HealthCheck" {
		return false
	}
	cluster := strings.TrimSpace(gcpVisionAIString(body, "cluster"))
	if cluster == "" {
		respondGCPVisionAIInvalidArgument(w, path, "cluster is required")
		return true
	}
	if !strings.Contains(cluster, "/clusters/") {
		respondGCPVisionAIInvalidArgument(w, path, "cluster must be in projects/{project}/locations/{location}/clusters/{cluster} format")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"healthy": true,
		"reason":  "",
		"clusterInfo": map[string]any{
			"streamsCount":   2,
			"processesCount": 1,
		},
	})
	return true
}

func handleGCPVisionAIStreamsMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	switch method {
	case "ListStreams":
		parent := strings.TrimSpace(gcpVisionAIString(body, "parent"))
		project, location, ok := parseGCPVisionAIParentName(parent)
		if !ok {
			respondGCPVisionAIInvalidArgument(w, path, "parent is required")
			return true
		}
		pageSize, start, valid := parseGCPVisionAIBodyPagination(w, path, body, 100, 1000)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpVisionAIStreamFixture(project, location, "stream-1"),
			gcpVisionAIStreamFixture(project, location, "stream-2"),
		}
		return respondGCPVisionAIList(w, "streams", items, pageSize, start, path)
	case "GetStream":
		name := strings.TrimSpace(gcpVisionAIString(body, "name"))
		project, location, streamID, ok := parseGCPVisionAIStreamName(name)
		if !ok {
			respondGCPVisionAIInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPVisionAIMissingID(streamID) {
			respondGCPVisionAINotFound(w, path, "stream not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVisionAIStreamFixture(project, location, streamID))
		return true
	case "CreateStream":
		parent := strings.TrimSpace(gcpVisionAIString(body, "parent"))
		project, location, ok := parseGCPVisionAIParentName(parent)
		if !ok {
			respondGCPVisionAIInvalidArgument(w, path, "parent is required")
			return true
		}
		streamID := strings.TrimSpace(gcpVisionAIString(body, "streamId", "stream_id"))
		if !isGCPVisionAIResourceID(streamID) {
			respondGCPVisionAIInvalidArgument(w, path, "streamId is required")
			return true
		}
		stream := gcpVisionAIBodyMap(body, "stream")
		if len(stream) == 0 {
			respondGCPVisionAIInvalidArgument(w, path, "stream is required")
			return true
		}
		displayName := strings.TrimSpace(gcpVisionAIString(stream, "displayName", "display_name"))
		if displayName == "" {
			respondGCPVisionAIInvalidArgument(w, path, "stream.displayName is required")
			return true
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/streams/%s", project, location, streamID)
		if providedName := strings.TrimSpace(gcpVisionAIString(stream, "name")); providedName != "" && providedName != expectedName {
			respondGCPVisionAIInvalidArgument(w, path, "stream.name must match parent and streamId")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVisionAIOperationFixture(project, location, "createStream."+streamID, gcpVisionAIStreamFixture(project, location, streamID)))
		return true
	default:
		return false
	}
}

func handleGCPVisionAIAppPlatformMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	if method != "ListApplications" {
		return false
	}
	parent := strings.TrimSpace(gcpVisionAIString(body, "parent"))
	project, location, ok := parseGCPVisionAIParentName(parent)
	if !ok {
		respondGCPVisionAIInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPVisionAIBodyPagination(w, path, body, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionAIApplicationFixture(project, location, "application-1"),
		gcpVisionAIApplicationFixture(project, location, "application-2"),
	}
	return respondGCPVisionAIList(w, "applications", items, pageSize, start, path)
}

func handleGCPVisionAILiveVideoAnalyticsMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	if method != "ListPublicOperators" {
		return false
	}
	parent := strings.TrimSpace(gcpVisionAIString(body, "parent"))
	project, location, ok := parseGCPVisionAIParentName(parent)
	if !ok {
		respondGCPVisionAIInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPVisionAIBodyPagination(w, path, body, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionAIOperatorFixture(project, location, "public-operator-1"),
		gcpVisionAIOperatorFixture(project, location, "public-operator-2"),
	}
	return respondGCPVisionAIList(w, "operators", items, pageSize, start, path)
}

func handleGCPVisionAIWarehouseMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	switch method {
	case "CreateCorpus":
		parent := strings.TrimSpace(gcpVisionAIString(body, "parent"))
		project, location, ok := parseGCPVisionAIParentName(parent)
		if !ok {
			respondGCPVisionAIInvalidArgument(w, path, "parent is required")
			return true
		}
		corpus := gcpVisionAIBodyMap(body, "corpus")
		if len(corpus) == 0 {
			respondGCPVisionAIInvalidArgument(w, path, "corpus is required")
			return true
		}
		displayName := strings.TrimSpace(gcpVisionAIString(corpus, "displayName", "display_name"))
		if displayName == "" {
			respondGCPVisionAIInvalidArgument(w, path, "corpus.displayName is required")
			return true
		}
		response := gcpVisionAICorpusFixture(project, location, "corpus-1")
		response["displayName"] = displayName
		respondJSON(w, http.StatusOK, gcpVisionAIOperationFixture(project, location, "createCorpus.corpus-1", response))
		return true
	case "ListCorpora":
		parent := strings.TrimSpace(gcpVisionAIString(body, "parent"))
		project, location, ok := parseGCPVisionAIParentName(parent)
		if !ok {
			respondGCPVisionAIInvalidArgument(w, path, "parent is required")
			return true
		}
		pageSize, start, valid := parseGCPVisionAIBodyPagination(w, path, body, 10, 20)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpVisionAICorpusFixture(project, location, "corpus-1"),
			gcpVisionAICorpusFixture(project, location, "corpus-2"),
		}
		return respondGCPVisionAIList(w, "corpora", items, pageSize, start, path)
	case "GetCorpus":
		name := strings.TrimSpace(gcpVisionAIString(body, "name"))
		project, location, corpusID, ok := parseGCPVisionAICorpusName(name)
		if !ok {
			respondGCPVisionAIInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPVisionAIMissingID(corpusID) {
			respondGCPVisionAINotFound(w, path, "corpus not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVisionAICorpusFixture(project, location, corpusID))
		return true
	default:
		return false
	}
}

func handleGCPVisionAIStreamingMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	switch method {
	case "AcquireLease":
		series := strings.TrimSpace(gcpVisionAIString(body, "series"))
		owner := strings.TrimSpace(gcpVisionAIString(body, "owner"))
		if series == "" || owner == "" {
			respondGCPVisionAIInvalidArgument(w, path, "series and owner are required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVisionAILeaseFixture("lease-1", series, owner))
		return true
	case "RenewLease":
		leaseID := strings.TrimSpace(gcpVisionAIString(body, "id"))
		series := strings.TrimSpace(gcpVisionAIString(body, "series"))
		owner := strings.TrimSpace(gcpVisionAIString(body, "owner"))
		if leaseID == "" || series == "" || owner == "" {
			respondGCPVisionAIInvalidArgument(w, path, "id, series, and owner are required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVisionAILeaseFixture(leaseID, series, owner))
		return true
	case "ReleaseLease":
		leaseID := strings.TrimSpace(gcpVisionAIString(body, "id"))
		series := strings.TrimSpace(gcpVisionAIString(body, "series"))
		owner := strings.TrimSpace(gcpVisionAIString(body, "owner"))
		if leaseID == "" || series == "" || owner == "" {
			respondGCPVisionAIInvalidArgument(w, path, "id, series, and owner are required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		return false
	}
}

func handleGCPVisionAIListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPVisionAIQueryPagination(w, r, path, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionAILocation(project, "us-central1"),
		gcpVisionAILocation(project, "global"),
	}
	return respondGCPVisionAIList(w, "locations", items, pageSize, start, path)
}

func handleGCPVisionAIGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVisionAILocation(project, location))
	return true
}

func handleGCPVisionAIListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPVisionAIOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPVisionAIQueryPagination(w, r, path, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionAIOperationFixture(project, location, "createStream.stream-1", gcpVisionAIStreamFixture(project, location, "stream-1")),
	}
	return respondGCPVisionAIList(w, "operations", items, pageSize, start, path)
}

func handleGCPVisionAIGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPVisionAIOperationPath(path)
	if !ok {
		return false
	}
	if isGCPVisionAIMissingID(operationID) {
		respondGCPVisionAINotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionAIOperationFixture(project, location, operationID, map[string]any{
		"@type": "type.googleapis.com/google.protobuf.Empty",
	}))
	return true
}

func handleGCPVisionAICancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, operationID, action, ok := parseGCPVisionAIOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	if _, valid := decodeGCPVisionAIJSONBody(w, r, path); !valid {
		return true
	}
	if isGCPVisionAIMissingID(operationID) {
		respondGCPVisionAINotFound(w, path, "operation not found")
		return true
	}
	_ = project
	_ = location
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionAIDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, ok := parseGCPVisionAIOperationPath(path)
	if !ok {
		return false
	}
	if isGCPVisionAIMissingID(operationID) {
		respondGCPVisionAINotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPVisionAIOperationsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPVisionAIOperationPath(path string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPVisionAIOperationActionPath(path string) (project, location, operationID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	last := strings.TrimSpace(parts[7])
	operationID, action, ok = strings.Cut(last, ":")
	if !ok || operationID == "" || action == "" {
		return "", "", "", "", false
	}
	if project == "" || location == "" {
		return "", "", "", "", false
	}
	return project, location, operationID, action, true
}

func parseGCPVisionAIParentName(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPVisionAIStreamName(name string) (project, location, streamID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "streams" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	streamID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || streamID == "" {
		return "", "", "", false
	}
	return project, location, streamID, true
}

func parseGCPVisionAICorpusName(name string) (project, location, corpusID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "corpora" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	corpusID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || corpusID == "" {
		return "", "", "", false
	}
	return project, location, corpusID, true
}

func parseGCPVisionAIBodyPagination(w http.ResponseWriter, path string, body map[string]any, defaultSize, maxSize int) (pageSize int, start int, ok bool) {
	pageSize = defaultSize
	if raw, exists := body["pageSize"]; exists {
		parsed, valid := gcpVisionAIAnyInt(raw)
		if !valid {
			respondGCPVisionAIInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	} else if raw, exists := body["page_size"]; exists {
		parsed, valid := gcpVisionAIAnyInt(raw)
		if !valid {
			respondGCPVisionAIInvalidArgument(w, path, "page_size must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize < 0 {
		respondGCPVisionAIInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize == 0 {
		pageSize = defaultSize
	}
	if pageSize > maxSize {
		respondGCPVisionAIOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxSize))
		return 0, 0, false
	}

	token := strings.TrimSpace(gcpVisionAIString(body, "pageToken", "page_token"))
	if token == "" {
		return pageSize, 0, true
	}
	parsed, err := strconv.Atoi(token)
	if err != nil || parsed < 0 {
		respondGCPVisionAIInvalidArgument(w, path, "pageToken must be a non-negative integer")
		return 0, 0, false
	}
	return pageSize, parsed, true
}

func parseGCPVisionAIQueryPagination(w http.ResponseWriter, r *http.Request, path string, defaultSize, maxSize int) (pageSize int, start int, ok bool) {
	pageSize = defaultSize
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respondGCPVisionAIInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize == 0 {
		pageSize = defaultSize
	}
	if pageSize > maxSize {
		respondGCPVisionAIOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxSize))
		return 0, 0, false
	}

	token := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if token == "" {
		return pageSize, 0, true
	}
	parsed, err := strconv.Atoi(token)
	if err != nil || parsed < 0 {
		respondGCPVisionAIInvalidArgument(w, path, "pageToken must be a non-negative integer")
		return 0, 0, false
	}
	return pageSize, parsed, true
}

func respondGCPVisionAIList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPVisionAIInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	if pageSize < 0 {
		respondGCPVisionAIInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextToken,
	})
	return true
}

func gcpVisionAILocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"metadata": map[string]any{
			"service": "visionai",
		},
	}
}

func gcpVisionAIStreamFixture(project, location, streamID string) map[string]any {
	return map[string]any{
		"name":                fmt.Sprintf("projects/%s/locations/%s/streams/%s", project, location, streamID),
		"displayName":         "Stackyard Stream " + streamID,
		"createTime":          gcpVisionAIReferenceTime.Format(time.RFC3339Nano),
		"updateTime":          gcpVisionAIReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		"labels":              map[string]any{"env": "staged"},
		"annotations":         map[string]any{"source": "stackyard"},
		"enableHlsPlayback":   true,
		"mediaWarehouseAsset": fmt.Sprintf("projects/%s/locations/%s/corpora/corpus-1/assets/asset-1", project, location),
	}
}

func gcpVisionAIApplicationFixture(project, location, appID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/applications/%s", project, location, appID),
		"displayName": "Stackyard Application " + appID,
		"description": "Stackyard staged vision ai application",
		"state":       "DEPLOYED",
		"createTime":  gcpVisionAIReferenceTime.Format(time.RFC3339Nano),
		"updateTime":  gcpVisionAIReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		"labels":      map[string]any{"env": "staged"},
	}
}

func gcpVisionAIOperatorFixture(project, location, operatorID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/operators/%s", project, location, operatorID),
		"dockerImage": "gcr.io/stackyard/visionai/operator:latest",
		"createTime":  gcpVisionAIReferenceTime.Format(time.RFC3339Nano),
		"updateTime":  gcpVisionAIReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		"labels":      map[string]any{"source": "public"},
		"operatorDefinition": map[string]any{
			"operator": operatorID,
		},
	}
}

func gcpVisionAICorpusFixture(project, location, corpusID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/corpora/%s", project, location, corpusID),
		"displayName": "Stackyard Corpus " + corpusID,
		"description": "Stackyard staged corpus",
		"type":        "VIDEO",
	}
}

func gcpVisionAILeaseFixture(leaseID, series, owner string) map[string]any {
	return map[string]any{
		"id":         leaseID,
		"series":     series,
		"owner":      owner,
		"leaseType":  "LEASE_TYPE_READ",
		"expireTime": gcpVisionAIReferenceTime.Add(5 * time.Minute).Format(time.RFC3339Nano),
	}
}

func gcpVisionAIOperationFixture(project, location, operationID string, response map[string]any) map[string]any {
	resp := response
	if resp == nil {
		resp = map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		}
	}
	if _, ok := resp["@type"]; !ok {
		resp["@type"] = "type.googleapis.com/google.protobuf.Struct"
	}
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.visionai.v1.OperationMetadata",
			"createTime": gcpVisionAIReferenceTime.Format(time.RFC3339Nano),
			"updateTime": gcpVisionAIReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		},
		"response": resp,
	}
}

func gcpVisionAIAnyInt(raw any) (int, bool) {
	switch v := raw.(type) {
	case float64:
		if v < 0 || float64(int(v)) != v {
			return 0, false
		}
		return int(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil || n < 0 {
			return 0, false
		}
		return int(n), true
	case int:
		if v < 0 {
			return 0, false
		}
		return v, true
	case int32:
		if v < 0 {
			return 0, false
		}
		return int(v), true
	case int64:
		if v < 0 {
			return 0, false
		}
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < 0 {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func gcpVisionAIString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		val, ok := raw.(string)
		if !ok {
			continue
		}
		if strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}

func gcpVisionAIBodyMap(body map[string]any, key string) map[string]any {
	raw, ok := body[key]
	if !ok {
		return map[string]any{}
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return obj
}

func isGCPVisionAIResourceID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func isGCPVisionAIMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "missing")
}

func respondGCPVisionAIInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVisionAINotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVisionAIFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVisionAIOutOfRange(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "OutOfRange",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_visionai(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "visionai") &&
		!isGCPContractProbeRequestForService(r, path, "visionai-apiv1") &&
		!isGCPContractProbeRequestForService(r, path, "vision-ai") {
		return false
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		if _, err := parseOptionalNonNegativeInt(raw); err != nil {
			respondGCPVisionAIInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return true
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":         "projects/stackyard/locations/us-central1/visionai/sample",
		"service":      "visionai",
		"provider":     providerGCP,
		"typedSuccess": true,
		"resource":     "projects/stackyard/locations/us-central1/streams/stream-1",
		"operation":    "projects/stackyard/locations/us-central1/operations/createStream.stream-1",
	})
	return true
}
