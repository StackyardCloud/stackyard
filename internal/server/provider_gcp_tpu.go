package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gcpTPUReferenceTime    = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpTPUNodeIDRegex      = regexp.MustCompile(`^[a-z](?:[a-z0-9-]{0,38}[a-z0-9])?$`)
	gcpTPUVersionIDRegex   = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	gcpTPUOperationIDRegex = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
)

func (s *Server) handleGCPTPURouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_tpu(w, r) {
		return true
	}

	path := normalizeGCPTPUPath(rawRequestPath(r))
	if isGCPTPULocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPTPUListLocations(w, r, path) {
			return true
		}
		if handleGCPTPUGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPTPUPath(path, hasGCPTPUHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPTPUListNodes(w, r, path) {
			return true
		}
		if handleGCPTPUGetNode(w, path) {
			return true
		}
		if handleGCPTPUListTensorFlowVersions(w, r, path) {
			return true
		}
		if handleGCPTPUGetTensorFlowVersion(w, path) {
			return true
		}
		if handleGCPTPUListAcceleratorTypes(w, r, path) {
			return true
		}
		if handleGCPTPUGetAcceleratorType(w, path) {
			return true
		}
		if handleGCPTPUListOperations(w, r, path) {
			return true
		}
		if handleGCPTPUGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPTPUCreateNode(w, r, path) {
			return true
		}
		if handleGCPTPUReimageNode(w, r, path) {
			return true
		}
		if handleGCPTPUStartNode(w, r, path) {
			return true
		}
		if handleGCPTPUStopNode(w, r, path) {
			return true
		}
		if handleGCPTPUCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPTPUDeleteNode(w, path) {
			return true
		}
		if handleGCPTPUDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPTPUPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPTPUHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "tpu", "tpu-apiv1", "tpu_apiv1", "cloud-tpu", "cloud_tpu", "cloudtpu", "gcp-tpu", "tensor-processing-unit":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-tpu-apiv1") || strings.Contains(ua, "cloud.google.com/go/tpu")
}

func isGCPTPULocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPTPUHint(r)
}

func isGCPTPUPath(path string, includeOperations bool) bool {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || project == "" || location == "" || len(tail) == 0 {
		return false
	}
	if isGCPTPUNodesCollectionTail(tail) ||
		isGCPTPUNodeTail(tail) ||
		isGCPTPUNodeActionTail(tail, "reimage") ||
		isGCPTPUNodeActionTail(tail, "start") ||
		isGCPTPUNodeActionTail(tail, "stop") ||
		isGCPTPUTensorFlowVersionsCollectionTail(tail) ||
		isGCPTPUTensorFlowVersionTail(tail) ||
		isGCPTPUAcceleratorTypesCollectionTail(tail) ||
		isGCPTPUAcceleratorTypeTail(tail) {
		return true
	}
	return includeOperations &&
		(isGCPTPUOperationsCollectionTail(tail) ||
			isGCPTPUOperationTail(tail) ||
			isGCPTPUOperationActionTail(tail, "cancel"))
}

func handleGCPTPUListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPTPUPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTPULocationFixture(project, "us-central1"),
		gcpTPULocationFixture(project, "us-east1"),
	}
	return respondGCPTPUList(w, "locations", items, pageSize, start, path)
}

func handleGCPTPUGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpTPULocationFixture(project, location))
	return true
}

func handleGCPTPUListNodes(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUNodesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPTPUPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpTPUNodeFixture(project, location, "node-1", "READY"),
		gcpTPUNodeFixture(project, location, "node-stopped", "STOPPED"),
	}
	if location == "-" {
		items = []map[string]any{
			gcpTPUNodeFixture(project, "us-central1", "node-1", "READY"),
			gcpTPUNodeFixture(project, "us-east1", "node-stopped", "STOPPED"),
		}
	}
	return respondGCPTPUList(w, "nodes", items, pageSize, start, path)
}

func handleGCPTPUGetNode(w http.ResponseWriter, path string) bool {
	project, location, nodeID, ok := parseGCPTPUNodePath(path)
	if !ok {
		return false
	}
	if isGCPTPUMissingID(nodeID) {
		respondGCPTPUNotFound(w, path, "node not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTPUNodeFixture(project, location, nodeID, gcpTPUStateForNodeID(nodeID)))
	return true
}

func handleGCPTPUCreateNode(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUNodesCollectionTail(tail) {
		return false
	}

	nodeID := strings.TrimSpace(r.URL.Query().Get("nodeId"))
	if nodeID == "" {
		nodeID = strings.TrimSpace(r.URL.Query().Get("node_id"))
	}
	if nodeID == "" {
		respondGCPTPUInvalidArgument(w, path, "nodeId is required")
		return true
	}
	if !gcpTPUNodeIDRegex.MatchString(nodeID) {
		respondGCPTPUInvalidArgument(w, path, "nodeId is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(nodeID), "existing") {
		respondGCPTPUAlreadyExists(w, path, "node already exists")
		return true
	}

	body, valid := decodeGCPTPUJSONBody(w, r, path)
	if !valid {
		return true
	}
	node := gcpTPUBodyMap(body, "node")
	if len(node) == 0 {
		respondGCPTPUInvalidArgument(w, path, "node is required")
		return true
	}

	expectedName := gcpTPUNodeName(project, location, nodeID)
	if providedName := strings.TrimSpace(gcpTPUString(node, "name")); providedName != "" && providedName != expectedName {
		respondGCPTPUInvalidArgument(w, path, "node.name must match parent and nodeId")
		return true
	}

	if acceleratorType := strings.TrimSpace(gcpTPUString(node, "acceleratorType", "accelerator_type")); acceleratorType == "" {
		respondGCPTPUInvalidArgument(w, path, "node.acceleratorType is required")
		return true
	}
	if tensorflowVersion := strings.TrimSpace(gcpTPUString(node, "tensorflowVersion", "tensorflow_version")); tensorflowVersion == "" {
		respondGCPTPUInvalidArgument(w, path, "node.tensorflowVersion is required")
		return true
	}

	responseNode := gcpTPUNodeFixture(project, location, nodeID, "READY")
	gcpTPUApplyNodeOverrides(responseNode, node)
	respondJSON(w, http.StatusOK, gcpTPUOperationFixture(project, location, "createNode."+nodeID, expectedName, "create", responseNode))
	return true
}

func handleGCPTPUDeleteNode(w http.ResponseWriter, path string) bool {
	project, location, nodeID, ok := parseGCPTPUNodePath(path)
	if !ok {
		return false
	}
	if isGCPTPUMissingID(nodeID) {
		respondGCPTPUNotFound(w, path, "node not found")
		return true
	}
	responseNode := gcpTPUNodeFixture(project, location, nodeID, "STOPPED")
	respondJSON(w, http.StatusOK, gcpTPUOperationFixture(project, location, "deleteNode."+nodeID, gcpTPUNodeName(project, location, nodeID), "delete", responseNode))
	return true
}

func handleGCPTPUReimageNode(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, nodeID, ok := parseGCPTPUNodeActionPath(path, "reimage")
	if !ok {
		return false
	}
	if isGCPTPUMissingID(nodeID) {
		respondGCPTPUNotFound(w, path, "node not found")
		return true
	}

	body, valid := decodeGCPTPUJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpTPUNodeName(project, location, nodeID)
	if got := strings.TrimSpace(gcpTPUString(body, "name")); got == "" || got != expectedName {
		respondGCPTPUInvalidArgument(w, path, "name must match the requested resource")
		return true
	}

	state := gcpTPUStateForNodeID(nodeID)
	if state == "CREATING" || state == "DELETING" || state == "STARTING" || state == "STOPPING" {
		respondGCPTPUFailedPrecondition(w, path, "node is not ready for reimage")
		return true
	}

	version := strings.TrimSpace(gcpTPUString(body, "tensorflowVersion", "tensorflow_version"))
	if version != "" && !isGCPTPUValidTensorFlowVersionInput(project, location, version) {
		respondGCPTPUInvalidArgument(w, path, "tensorflowVersion is invalid")
		return true
	}

	responseNode := gcpTPUNodeFixture(project, location, nodeID, "READY")
	if version != "" {
		if strings.Contains(version, "/") {
			_, _, versionID, _ := parseGCPTPUTensorFlowVersionName(version)
			responseNode["tensorflowVersion"] = versionID
		} else {
			responseNode["tensorflowVersion"] = version
		}
	}
	respondJSON(w, http.StatusOK, gcpTPUOperationFixture(project, location, "reimageNode."+nodeID, expectedName, "reimage", responseNode))
	return true
}

func handleGCPTPUStartNode(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, nodeID, ok := parseGCPTPUNodeActionPath(path, "start")
	if !ok {
		return false
	}
	if isGCPTPUMissingID(nodeID) {
		respondGCPTPUNotFound(w, path, "node not found")
		return true
	}

	body, valid := decodeGCPTPUJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpTPUNodeName(project, location, nodeID)
	if got := strings.TrimSpace(gcpTPUString(body, "name")); got == "" || got != expectedName {
		respondGCPTPUInvalidArgument(w, path, "name must match the requested resource")
		return true
	}

	if gcpTPUStateForNodeID(nodeID) != "STOPPED" {
		respondGCPTPUFailedPrecondition(w, path, "node must be STOPPED to start")
		return true
	}

	responseNode := gcpTPUNodeFixture(project, location, nodeID, "READY")
	respondJSON(w, http.StatusOK, gcpTPUOperationFixture(project, location, "startNode."+nodeID, expectedName, "start", responseNode))
	return true
}

func handleGCPTPUStopNode(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, nodeID, ok := parseGCPTPUNodeActionPath(path, "stop")
	if !ok {
		return false
	}
	if isGCPTPUMissingID(nodeID) {
		respondGCPTPUNotFound(w, path, "node not found")
		return true
	}

	body, valid := decodeGCPTPUJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpTPUNodeName(project, location, nodeID)
	if got := strings.TrimSpace(gcpTPUString(body, "name")); got == "" || got != expectedName {
		respondGCPTPUInvalidArgument(w, path, "name must match the requested resource")
		return true
	}

	if gcpTPUStateForNodeID(nodeID) != "READY" {
		respondGCPTPUFailedPrecondition(w, path, "node must be READY to stop")
		return true
	}

	responseNode := gcpTPUNodeFixture(project, location, nodeID, "STOPPED")
	respondJSON(w, http.StatusOK, gcpTPUOperationFixture(project, location, "stopNode."+nodeID, expectedName, "stop", responseNode))
	return true
}

func handleGCPTPUListTensorFlowVersions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUTensorFlowVersionsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPTPUPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTPUTensorFlowVersionFixture(project, location, "v2-alpha"),
		gcpTPUTensorFlowVersionFixture(project, location, "tpu-vm-tf-2.15.0"),
	}
	return respondGCPTPUList(w, "tensorflowVersions", items, pageSize, start, path)
}

func handleGCPTPUGetTensorFlowVersion(w http.ResponseWriter, path string) bool {
	project, location, versionID, ok := parseGCPTPUTensorFlowVersionPath(path)
	if !ok {
		return false
	}
	if isGCPTPUMissingID(versionID) {
		respondGCPTPUNotFound(w, path, "tensorflow version not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTPUTensorFlowVersionFixture(project, location, versionID))
	return true
}

func handleGCPTPUListAcceleratorTypes(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUAcceleratorTypesCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPTPUPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTPUAcceleratorTypeFixture(project, location, "v3-8"),
		gcpTPUAcceleratorTypeFixture(project, location, "v4-8"),
	}
	return respondGCPTPUList(w, "acceleratorTypes", items, pageSize, start, path)
}

func handleGCPTPUGetAcceleratorType(w http.ResponseWriter, path string) bool {
	project, location, acceleratorTypeID, ok := parseGCPTPUAcceleratorTypePath(path)
	if !ok {
		return false
	}
	if isGCPTPUMissingID(acceleratorTypeID) {
		respondGCPTPUNotFound(w, path, "accelerator type not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTPUAcceleratorTypeFixture(project, location, acceleratorTypeID))
	return true
}

func handleGCPTPUListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPTPUPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpTPUOperationFixture(project, location, "createNode.node-1", gcpTPUNodeName(project, location, "node-1"), "create", gcpTPUNodeFixture(project, location, "node-1", "READY")),
		gcpTPUOperationFixture(project, location, "startNode.node-stopped", gcpTPUNodeName(project, location, "node-stopped"), "start", gcpTPUNodeFixture(project, location, "node-stopped", "READY")),
		gcpTPUOperationFixture(project, location, "stopNode.node-1", gcpTPUNodeName(project, location, "node-1"), "stop", gcpTPUNodeFixture(project, location, "node-1", "STOPPED")),
	}
	return respondGCPTPUList(w, "operations", items, pageSize, start, path)
}

func handleGCPTPUGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPTPUOperationPath(path)
	if !ok {
		return false
	}
	if isGCPTPUMissingID(operationID) {
		respondGCPTPUNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpTPUOperationByID(project, location, operationID))
	return true
}

func handleGCPTPUCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, ok := parseGCPTPUOperationActionPath(path, "cancel")
	if !ok {
		return false
	}
	if isGCPTPUMissingID(operationID) {
		respondGCPTPUNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPTPUDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, ok := parseGCPTPUOperationPath(path)
	if !ok {
		return false
	}
	if isGCPTPUMissingID(operationID) {
		respondGCPTPUNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func decodeGCPTPUJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPTPUInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpTPUBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpTPUString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := body[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseGCPTPUPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, valid bool) {
	pageSize = 100
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respondGCPTPUInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if parsed > 1000 {
			respondGCPTPUInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
			return 0, 0, false
		}
		pageSize = parsed
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respondGCPTPUInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func respondGCPTPUList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	sort.Slice(items, func(i, j int) bool {
		return gcpTPUString(items[i], "name") < gcpTPUString(items[j], "name")
	})
	if start > len(items) {
		respondGCPTPUInvalidArgument(w, path, "pageToken out of range")
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

func gcpTPULocationFixture(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Cloud TPU " + location,
		"labels": map[string]string{
			"service": "tpu",
			"stage":   "emulated",
		},
	}
}

func gcpTPUNodeName(project, location, nodeID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/nodes/%s", project, location, nodeID)
}

func gcpTPUNodeFixture(project, location, nodeID, state string) map[string]any {
	health := "HEALTHY"
	if strings.Contains(strings.ToLower(nodeID), "unhealthy") {
		health = "TIMEOUT"
	}
	port := 8470
	if state == "STOPPED" {
		port = 0
	}
	return map[string]any{
		"name":              gcpTPUNodeName(project, location, nodeID),
		"description":       "Stackyard TPU node " + nodeID,
		"acceleratorType":   "v3-8",
		"tensorflowVersion": "v2-alpha",
		"network":           fmt.Sprintf("projects/%s/global/networks/default", project),
		"cidrBlock":         "10.240.0.0/29",
		"serviceAccount":    "stackyard-tpu@" + project + ".iam.gserviceaccount.com",
		"createTime":        gcpTPUReferenceTime.Format(time.RFC3339Nano),
		"state":             state,
		"health":            health,
		"labels": map[string]string{
			"env":   "staged",
			"owner": "stackyard",
		},
		"networkEndpoints": []map[string]any{
			{
				"ipAddress": "10.240.0.2",
				"port":      port,
			},
		},
	}
}

func gcpTPUApplyNodeOverrides(dst, src map[string]any) {
	if description := strings.TrimSpace(gcpTPUString(src, "description")); description != "" {
		dst["description"] = description
	}
	if acceleratorType := strings.TrimSpace(gcpTPUString(src, "acceleratorType", "accelerator_type")); acceleratorType != "" {
		dst["acceleratorType"] = acceleratorType
	}
	if tensorflowVersion := strings.TrimSpace(gcpTPUString(src, "tensorflowVersion", "tensorflow_version")); tensorflowVersion != "" {
		dst["tensorflowVersion"] = tensorflowVersion
	}
	if labels, ok := src["labels"].(map[string]any); ok && len(labels) > 0 {
		converted := map[string]string{}
		for key, value := range labels {
			if str, ok := value.(string); ok {
				converted[strings.TrimSpace(key)] = strings.TrimSpace(str)
			}
		}
		if len(converted) > 0 {
			dst["labels"] = converted
		}
	}
}

func gcpTPUTensorFlowVersionFixture(project, location, versionID string) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("projects/%s/locations/%s/tensorflowVersions/%s", project, location, versionID),
		"version": versionID,
	}
}

func gcpTPUAcceleratorTypeFixture(project, location, acceleratorTypeID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/acceleratorTypes/%s", project, location, acceleratorTypeID),
		"type": acceleratorTypeID,
	}
}

func gcpTPUOperationFixture(project, location, operationID, target, verb string, response map[string]any) map[string]any {
	metadata := map[string]any{
		"@type":           "type.googleapis.com/google.cloud.tpu.v1.OperationMetadata",
		"target":          target,
		"verb":            verb,
		"statusDetail":    "completed",
		"createTime":      gcpTPUReferenceTime.Format(time.RFC3339Nano),
		"endTime":         gcpTPUReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		"cancelRequested": false,
		"apiVersion":      "v1",
	}
	out := map[string]any{
		"name":     fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"metadata": metadata,
		"done":     true,
	}
	if response != nil {
		out["response"] = response
	}
	return out
}

func gcpTPUOperationByID(project, location, operationID string) map[string]any {
	switch {
	case strings.HasPrefix(operationID, "createNode."):
		nodeID := strings.TrimPrefix(operationID, "createNode.")
		return gcpTPUOperationFixture(project, location, operationID, gcpTPUNodeName(project, location, nodeID), "create", gcpTPUNodeFixture(project, location, nodeID, "READY"))
	case strings.HasPrefix(operationID, "deleteNode."):
		nodeID := strings.TrimPrefix(operationID, "deleteNode.")
		return gcpTPUOperationFixture(project, location, operationID, gcpTPUNodeName(project, location, nodeID), "delete", gcpTPUNodeFixture(project, location, nodeID, "STOPPED"))
	case strings.HasPrefix(operationID, "reimageNode."):
		nodeID := strings.TrimPrefix(operationID, "reimageNode.")
		return gcpTPUOperationFixture(project, location, operationID, gcpTPUNodeName(project, location, nodeID), "reimage", gcpTPUNodeFixture(project, location, nodeID, "READY"))
	case strings.HasPrefix(operationID, "startNode."):
		nodeID := strings.TrimPrefix(operationID, "startNode.")
		return gcpTPUOperationFixture(project, location, operationID, gcpTPUNodeName(project, location, nodeID), "start", gcpTPUNodeFixture(project, location, nodeID, "READY"))
	case strings.HasPrefix(operationID, "stopNode."):
		nodeID := strings.TrimPrefix(operationID, "stopNode.")
		return gcpTPUOperationFixture(project, location, operationID, gcpTPUNodeName(project, location, nodeID), "stop", gcpTPUNodeFixture(project, location, nodeID, "STOPPED"))
	default:
		return gcpTPUOperationFixture(project, location, operationID, fmt.Sprintf("projects/%s/locations/%s", project, location), "unknown", nil)
	}
}

func gcpTPUStateForNodeID(nodeID string) string {
	lower := strings.ToLower(strings.TrimSpace(nodeID))
	switch {
	case strings.Contains(lower, "stopped"):
		return "STOPPED"
	case strings.Contains(lower, "stopping"):
		return "STOPPING"
	case strings.Contains(lower, "starting"):
		return "STARTING"
	case strings.Contains(lower, "creating"), strings.Contains(lower, "provision"):
		return "CREATING"
	case strings.Contains(lower, "deleting"):
		return "DELETING"
	default:
		return "READY"
	}
}

func isGCPTPUMissingID(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(lower, "missing") || strings.Contains(lower, "notfound") || strings.Contains(lower, "deleted")
}

func isGCPTPUValidTensorFlowVersionInput(project, location, raw string) bool {
	version := strings.TrimSpace(raw)
	if version == "" {
		return false
	}
	if strings.Contains(version, "/") {
		p, l, _, ok := parseGCPTPUTensorFlowVersionName(version)
		return ok && p == project && l == location
	}
	return gcpTPUVersionIDRegex.MatchString(version)
}

func parseGCPTPULocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 {
		return "", "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	if len(parts) == 6 {
		return project, location, []string{}, true
	}
	return project, location, parts[6:], true
}

func parseGCPTPULocationName(name string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 {
		return "", "", false
	}
	if parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPTPUNodeName(name string) (project, location, nodeID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 {
		return "", "", "", false
	}
	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "nodes" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	nodeID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || nodeID == "" || !gcpTPUNodeIDRegex.MatchString(nodeID) {
		return "", "", "", false
	}
	return project, location, nodeID, true
}

func parseGCPTPUTensorFlowVersionName(name string) (project, location, versionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 {
		return "", "", "", false
	}
	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "tensorflowVersions" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	versionID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || versionID == "" || !gcpTPUVersionIDRegex.MatchString(versionID) {
		return "", "", "", false
	}
	return project, location, versionID, true
}

func parseGCPTPUAcceleratorTypeName(name string) (project, location, acceleratorTypeID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 {
		return "", "", "", false
	}
	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "acceleratorTypes" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	acceleratorTypeID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || acceleratorTypeID == "" || !gcpTPUVersionIDRegex.MatchString(acceleratorTypeID) {
		return "", "", "", false
	}
	return project, location, acceleratorTypeID, true
}

func parseGCPTPUOperationName(name string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 {
		return "", "", "", false
	}
	if parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || operationID == "" || !gcpTPUOperationIDRegex.MatchString(operationID) {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPTPUNodePath(path string) (project, location, nodeID string, ok bool) {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUNodeTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPTPUTensorFlowVersionPath(path string) (project, location, versionID string, ok bool) {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUTensorFlowVersionTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPTPUAcceleratorTypePath(path string) (project, location, acceleratorTypeID string, ok bool) {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUAcceleratorTypeTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPTPUOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || !isGCPTPUOperationTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPTPUNodeActionPath(path, action string) (project, location, nodeID string, ok bool) {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "nodes" {
		return "", "", "", false
	}
	id, act, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !found || act != action {
		return "", "", "", false
	}
	id = strings.TrimSpace(id)
	if id == "" || !gcpTPUNodeIDRegex.MatchString(id) {
		return "", "", "", false
	}
	return project, location, id, true
}

func parseGCPTPUOperationActionPath(path, action string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPTPULocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", false
	}
	id, act, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !found || act != action {
		return "", "", "", false
	}
	id = strings.TrimSpace(id)
	if id == "" || !gcpTPUOperationIDRegex.MatchString(id) {
		return "", "", "", false
	}
	return project, location, id, true
}

func isGCPTPUNodesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "nodes"
}

func isGCPTPUNodeTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "nodes" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPTPUNodeActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "nodes" {
		return false
	}
	id, act, ok := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return ok && strings.TrimSpace(id) != "" && act == action
}

func isGCPTPUTensorFlowVersionsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "tensorflowVersions"
}

func isGCPTPUTensorFlowVersionTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "tensorflowVersions" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPTPUAcceleratorTypesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "acceleratorTypes"
}

func isGCPTPUAcceleratorTypeTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "acceleratorTypes" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPTPUOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPTPUOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPTPUOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	id, act, ok := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return ok && strings.TrimSpace(id) != "" && act == action
}

func respondGCPTPUInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPTPUError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPTPUFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPTPUError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPTPUAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPTPUError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPTPUNotFound(w http.ResponseWriter, path, message string) {
	respondGCPTPUError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPTPUError(w http.ResponseWriter, status int, code, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_tpu(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "tpu") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPTPUInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/nodes/node-1",
			"service":  "tpu",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
