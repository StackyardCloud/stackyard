package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPContainerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContainerPathWithHint(path, hasGCPContainerHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPContainerListClusters(w, r, path) {
			return true
		}
		if handleGCPContainerGetCluster(w, path) {
			return true
		}
		if handleGCPContainerListNodePools(w, r, path) {
			return true
		}
		if handleGCPContainerGetNodePool(w, path) {
			return true
		}
		if handleGCPContainerListOperations(w, r, path) {
			return true
		}
		if handleGCPContainerGetOperation(w, path) {
			return true
		}
		if handleGCPContainerGetServerConfig(w, path) {
			return true
		}
		if handleGCPContainerListUsableSubnetworks(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPContainerCreateCluster(w, r, path) {
			return true
		}
		if handleGCPContainerClusterAction(w, r, path) {
			return true
		}
		if handleGCPContainerCreateNodePool(w, r, path) {
			return true
		}
		if handleGCPContainerNodePoolAction(w, r, path) {
			return true
		}
		if handleGCPContainerOperationAction(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPContainerDeleteCluster(w, path) {
			return true
		}
		if handleGCPContainerDeleteNodePool(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPut:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPContainerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "container", "container-apiv1", "container_apiv1", "gke", "kubernetes", "kubernetesengine":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-container-apiv1") || strings.Contains(ua, "cloud.google.com/go/container")
}

func isGCPContainerPath(path string) bool {
	return isGCPContainerPathWithHint(path, false)
}

func isGCPContainerPathWithHint(path string, includeHint bool) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !includeHint {
		return false
	}
	if _, _, ok := parseGCPContainerClustersCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPContainerClusterPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPContainerClusterActionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPContainerNodePoolsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPContainerNodePoolPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPContainerNodePoolActionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPContainerOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPContainerOperationPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPContainerOperationActionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPContainerServerConfigPath(path); ok {
		return true
	}
	_, ok := parseGCPContainerUsableSubnetworksPath(path)
	return ok
}

func handleGCPContainerListClusters(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPContainerClustersCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPContainerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpContainerCluster(project, location, "team-cluster")}
	return respondGCPContainerList(w, "clusters", items, pageSize, start, path)
}

func handleGCPContainerGetCluster(w http.ResponseWriter, path string) bool {
	project, location, cluster, ok := parseGCPContainerClusterPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpContainerCluster(project, location, cluster))
	return true
}

func handleGCPContainerCreateCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPContainerClustersCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPContainerJSONBody(w, r, path)
	if !valid {
		return true
	}
	cluster, _ := body["cluster"].(map[string]any)
	if len(cluster) == 0 {
		cluster = body
	}
	name, _ := cluster["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPContainerInvalidArgument(w, path, "cluster.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-create-cluster", "CREATE_CLUSTER", fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, name)))
	return true
}

func handleGCPContainerDeleteCluster(w http.ResponseWriter, path string) bool {
	project, location, cluster, ok := parseGCPContainerClusterPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-delete-cluster", "DELETE_CLUSTER", fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)))
	return true
}

func handleGCPContainerListNodePools(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, cluster, ok := parseGCPContainerNodePoolsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPContainerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpContainerNodePool(project, location, cluster, "default-pool")}
	return respondGCPContainerList(w, "nodePools", items, pageSize, start, path)
}

func handleGCPContainerGetNodePool(w http.ResponseWriter, path string) bool {
	project, location, cluster, nodePool, ok := parseGCPContainerNodePoolPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpContainerNodePool(project, location, cluster, nodePool))
	return true
}

func handleGCPContainerCreateNodePool(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, cluster, ok := parseGCPContainerNodePoolsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPContainerJSONBody(w, r, path)
	if !valid {
		return true
	}
	nodePool, _ := body["nodePool"].(map[string]any)
	if len(nodePool) == 0 {
		nodePool = body
	}
	name, _ := nodePool["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPContainerInvalidArgument(w, path, "nodePool.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-create-nodepool", "CREATE_NODE_POOL", fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, cluster, name)))
	return true
}

func handleGCPContainerDeleteNodePool(w http.ResponseWriter, path string) bool {
	project, location, cluster, nodePool, ok := parseGCPContainerNodePoolPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-delete-nodepool", "DELETE_NODE_POOL", fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, cluster, nodePool)))
	return true
}

func handleGCPContainerClusterAction(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, cluster, action, ok := parseGCPContainerClusterActionPath(path)
	if !ok {
		return false
	}

	switch action {
	case "setLogging":
		body, valid := decodeGCPContainerJSONBody(w, r, path)
		if !valid {
			return true
		}
		loggingService, _ := body["loggingService"].(string)
		if strings.TrimSpace(loggingService) == "" {
			respondGCPContainerInvalidArgument(w, path, "loggingService is required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-set-logging", "SET_LOGGING", fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)))
		return true
	case "setMonitoring":
		body, valid := decodeGCPContainerJSONBody(w, r, path)
		if !valid {
			return true
		}
		monitoringService, _ := body["monitoringService"].(string)
		if strings.TrimSpace(monitoringService) == "" {
			respondGCPContainerInvalidArgument(w, path, "monitoringService is required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-set-monitoring", "SET_MONITORING", fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)))
		return true
	case "setAddons":
		respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-set-addons", "SET_ADDONS", fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster)))
		return true
	case "checkAutopilotCompatibility":
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case "fetchClusterUpgradeInfo":
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		return false
	}
}

func handleGCPContainerNodePoolAction(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, cluster, nodePool, action, ok := parseGCPContainerNodePoolActionPath(path)
	if !ok {
		return false
	}

	switch action {
	case "setManagement":
		respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-set-nodepool-management", "SET_NODE_POOL_MANAGEMENT", fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, cluster, nodePool)))
		return true
	case "setSize":
		body, valid := decodeGCPContainerJSONBody(w, r, path)
		if !valid {
			return true
		}
		nodeCount, ok := body["nodeCount"].(float64)
		if !ok || nodeCount < 0 {
			respondGCPContainerInvalidArgument(w, path, "nodeCount must be a non-negative number")
			return true
		}
		respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-set-nodepool-size", "SET_NODE_POOL_SIZE", fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, cluster, nodePool)))
		return true
	case "completeUpgrade":
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case "rollback":
		respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, "op-rollback-nodepool", "ROLLBACK_NODE_POOL", fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, cluster, nodePool)))
		return true
	case "fetchNodePoolUpgradeInfo":
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		return false
	}
}

func handleGCPContainerListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPContainerOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPContainerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpContainerOperation(project, location, "op-1", "CREATE_CLUSTER", fmt.Sprintf("projects/%s/locations/%s/clusters/team-cluster", project, location))}
	return respondGCPContainerList(w, "operations", items, pageSize, start, path)
}

func handleGCPContainerGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operation, ok := parseGCPContainerOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpContainerOperation(project, location, operation, "CREATE_CLUSTER", fmt.Sprintf("projects/%s/locations/%s/clusters/team-cluster", project, location)))
	return true
}

func handleGCPContainerOperationAction(w http.ResponseWriter, path string) bool {
	_, _, _, action, ok := parseGCPContainerOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPContainerGetServerConfig(w http.ResponseWriter, path string) bool {
	_, _, ok := parseGCPContainerServerConfigPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"defaultClusterVersion": "1.31.2-gke.123",
		"validNodeVersions":     []string{"1.31.2-gke.123", "1.30.8-gke.122"},
	})
	return true
}

func handleGCPContainerListUsableSubnetworks(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPContainerUsableSubnetworksPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPContainerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"subnetwork": fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/default", project),
			"network":    fmt.Sprintf("projects/%s/global/networks/default", project),
		},
	}
	return respondGCPContainerList(w, "subnetworks", items, pageSize, start, path)
}

func parseGCPContainerClustersCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "clusters" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPContainerClusterPath(path string) (project, location, cluster string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "clusters" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	cluster = strings.TrimSpace(parts[7])
	if project == "" || location == "" || cluster == "" || strings.Contains(cluster, ":") {
		return "", "", "", false
	}
	return project, location, cluster, true
}

func parseGCPContainerClusterActionPath(path string) (project, location, cluster, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "clusters" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	clusterAction := normalizeGCPContainerActionSegment(parts[7])
	cluster, action, found := strings.Cut(clusterAction, ":")
	if !found {
		return "", "", "", "", false
	}
	cluster = strings.TrimSpace(cluster)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || cluster == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, cluster, action, true
}

func parseGCPContainerNodePoolsCollectionPath(path string) (project, location, cluster string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "clusters" || parts[8] != "nodePools" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	cluster = strings.TrimSpace(parts[7])
	if project == "" || location == "" || cluster == "" {
		return "", "", "", false
	}
	return project, location, cluster, true
}

func parseGCPContainerNodePoolPath(path string) (project, location, cluster, nodePool string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "clusters" || parts[8] != "nodePools" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	cluster = strings.TrimSpace(parts[7])
	nodePool = strings.TrimSpace(parts[9])
	if project == "" || location == "" || cluster == "" || nodePool == "" || strings.Contains(nodePool, ":") {
		return "", "", "", "", false
	}
	return project, location, cluster, nodePool, true
}

func parseGCPContainerNodePoolActionPath(path string) (project, location, cluster, nodePool, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "clusters" || parts[8] != "nodePools" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	cluster = strings.TrimSpace(parts[7])
	nodePoolAction := normalizeGCPContainerActionSegment(parts[9])
	nodePool, action, found := strings.Cut(nodePoolAction, ":")
	if !found {
		return "", "", "", "", "", false
	}
	nodePool = strings.TrimSpace(nodePool)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || cluster == "" || nodePool == "" || action == "" {
		return "", "", "", "", "", false
	}
	return project, location, cluster, nodePool, action, true
}

func parseGCPContainerOperationsCollectionPath(path string) (project, location string, ok bool) {
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

func parseGCPContainerOperationPath(path string) (project, location, operation string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operation = strings.TrimSpace(parts[7])
	if project == "" || location == "" || operation == "" || strings.Contains(operation, ":") {
		return "", "", "", false
	}
	return project, location, operation, true
}

func parseGCPContainerOperationActionPath(path string) (project, location, operation, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	opAction := normalizeGCPContainerActionSegment(parts[7])
	operation, action, found := strings.Cut(opAction, ":")
	if !found {
		return "", "", "", "", false
	}
	operation = strings.TrimSpace(operation)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || operation == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, operation, action, true
}

func parseGCPContainerServerConfigPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "serverConfig" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPContainerUsableSubnetworksPath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "aggregated" || parts[5] != "usableSubnetworks" {
		return "", false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPContainerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	size, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPContainerInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	token := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if token == "" {
		return size, 0, true
	}
	start, err = parseOptionalNonNegativeInt(token)
	if err != nil {
		respondGCPContainerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return size, start, true
}

func respondGCPContainerList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPContainerInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPContainerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPContainerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func normalizeGCPContainerActionSegment(segment string) string {
	trimmed := strings.TrimSpace(segment)
	trimmed = strings.ReplaceAll(trimmed, "%3A", ":")
	trimmed = strings.ReplaceAll(trimmed, "%3a", ":")
	return trimmed
}

func gcpContainerCluster(project, location, cluster string) map[string]any {
	return map[string]any{
		"name":                 fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, cluster),
		"location":             location,
		"status":               "RUNNING",
		"endpoint":             "10.0.0.2",
		"currentMasterVersion": "1.31.2-gke.123",
	}
}

func gcpContainerNodePool(project, location, cluster, nodePool string) map[string]any {
	return map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", project, location, cluster, nodePool),
		"status":           "RUNNING",
		"version":          "1.31.2-gke.123",
		"initialNodeCount": 1,
	}
}

func gcpContainerOperation(project, location, operationID, opType, targetLink string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"operationType": opType,
		"status":        "DONE",
		"targetLink":    targetLink,
	}
}

func respondGCPContainerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
