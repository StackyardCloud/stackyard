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
	gcpRedisClusterReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpRedisClusterIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	gcpRedisClusterExportPattern = regexp.MustCompile(`^/gcp/v1/projects/([^/]+)/locations/([^/]+)/backupCollections/([^/]+)/backups/([^/:]+)(?::|%3[Aa])export$`)
)

func (s *Server) handleGCPRedisClusterRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_redis_cluster(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if hasGCPRedisClusterHint(r) &&
		r.Method == http.MethodPost &&
		strings.Contains(path, "/backupCollections/") &&
		strings.Contains(path, "/backups/") &&
		(strings.Contains(path, ":export") || strings.Contains(strings.ToLower(path), "%3aexport")) {
		return handleGCPRedisClusterExportBackup(w, r, path)
	}
	if isGCPRedisClusterLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPRedisClusterListLocations(w, r, path) {
			return true
		}
		if handleGCPRedisClusterGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPRedisClusterPath(path, hasGCPRedisClusterHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRedisClusterListClusters(w, r, path) {
			return true
		}
		if handleGCPRedisClusterGetCluster(w, path) {
			return true
		}
		if handleGCPRedisClusterGetClusterCertificateAuthority(w, path) {
			return true
		}
		if handleGCPRedisClusterListBackupCollections(w, r, path) {
			return true
		}
		if handleGCPRedisClusterGetBackupCollection(w, path) {
			return true
		}
		if handleGCPRedisClusterListBackups(w, r, path) {
			return true
		}
		if handleGCPRedisClusterGetBackup(w, path) {
			return true
		}
		if handleGCPRedisClusterListOperations(w, r, path) {
			return true
		}
		if handleGCPRedisClusterGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRedisClusterCreateCluster(w, r, path) {
			return true
		}
		if handleGCPRedisClusterRescheduleClusterMaintenance(w, r, path) {
			return true
		}
		if handleGCPRedisClusterExportBackup(w, r, path) {
			return true
		}
		if handleGCPRedisClusterBackupCluster(w, r, path) {
			return true
		}
		if handleGCPRedisClusterCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRedisClusterUpdateCluster(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRedisClusterDeleteCluster(w, path) {
			return true
		}
		if handleGCPRedisClusterDeleteBackup(w, path) {
			return true
		}
		if handleGCPRedisClusterDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPRedisClusterHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "redis_cluster", "redis-cluster", "memorystore-redis-cluster":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-redis-cluster-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/redis/cluster")
}

func isGCPRedisClusterLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPRedisClusterHint(r)
}

func isGCPRedisClusterPath(path string, includeAmbiguous bool) bool {
	if !includeAmbiguous {
		return false
	}
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || project == "" || location == "" || len(tail) == 0 {
		return false
	}
	if isGCPRedisClusterClustersCollectionTail(tail) ||
		isGCPRedisClusterClusterTail(tail) ||
		isGCPRedisClusterClusterCertificateAuthorityTail(tail) ||
		isGCPRedisClusterClusterActionTail(tail, "rescheduleClusterMaintenance") ||
		isGCPRedisClusterClusterActionTail(tail, "backup") {
		return true
	}
	return isGCPRedisClusterBackupCollectionsCollectionTail(tail) ||
		isGCPRedisClusterBackupCollectionTail(tail) ||
		isGCPRedisClusterBackupsCollectionTail(tail) ||
		isGCPRedisClusterBackupTail(tail) ||
		isGCPRedisClusterBackupActionTail(tail, "export") ||
		isGCPRedisClusterOperationsCollectionTail(tail) ||
		isGCPRedisClusterOperationTail(tail) ||
		isGCPRedisClusterOperationActionTail(tail, "cancel")
}

func handleGCPRedisClusterListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPRedisClusterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisClusterLocationFixture(project, "us-central1"),
		gcpRedisClusterLocationFixture(project, "global"),
	}
	return respondGCPRedisClusterList(w, "locations", items, pageSize, start, path)
}

func handleGCPRedisClusterGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisClusterLocationFixture(project, location))
	return true
}

func handleGCPRedisClusterListClusters(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterClustersCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRedisClusterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisClusterFixture(project, location, "cluster-1"),
		gcpRedisClusterFixture(project, location, "cluster-2"),
	}
	return respondGCPRedisClusterList(w, "clusters", items, pageSize, start, path)
}

func handleGCPRedisClusterGetCluster(w http.ResponseWriter, path string) bool {
	project, location, clusterID, ok := parseGCPRedisClusterClusterPath(path)
	if !ok {
		return false
	}
	fixture := gcpRedisClusterFixture(project, location, clusterID)
	if strings.Contains(clusterID, "creating") {
		fixture["state"] = "CREATING"
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRedisClusterCreateCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterClustersCollectionTail(tail) {
		return false
	}

	clusterID := strings.TrimSpace(r.URL.Query().Get("clusterId"))
	if clusterID == "" {
		respondGCPRedisClusterInvalidArgument(w, path, "clusterId is required")
		return true
	}
	if !gcpRedisClusterIDPattern.MatchString(clusterID) {
		respondGCPRedisClusterInvalidArgument(w, path, "clusterId is invalid")
		return true
	}

	body, valid := decodeGCPRedisClusterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cluster := gcpRedisClusterBodyMap(body, "cluster")
	if len(cluster) == 0 {
		respondGCPRedisClusterInvalidArgument(w, path, "cluster is required")
		return true
	}
	expectedName := gcpRedisClusterClusterName(project, location, clusterID)
	if got := gcpRedisClusterString(cluster, "name"); got != "" && got != expectedName {
		respondGCPRedisClusterInvalidArgument(w, path, "cluster.name must match parent and clusterId")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "createCluster."+clusterID, expectedName, "createCluster"))
	return true
}

func handleGCPRedisClusterUpdateCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPRedisClusterClusterPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisClusterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cluster := gcpRedisClusterBodyMap(body, "cluster")
	if len(cluster) == 0 {
		respondGCPRedisClusterInvalidArgument(w, path, "cluster is required")
		return true
	}
	expectedName := gcpRedisClusterClusterName(project, location, clusterID)
	if got := gcpRedisClusterString(cluster, "name"); got == "" || got != expectedName {
		respondGCPRedisClusterInvalidArgument(w, path, "cluster.name must match the requested resource")
		return true
	}

	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = strings.TrimSpace(gcpRedisClusterString(body, "updateMask"))
	}
	if mask == "" {
		respondGCPRedisClusterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	maskPaths, ok := parseGCPRedisClusterUpdateMask(mask)
	if !ok {
		respondGCPRedisClusterInvalidArgument(w, path, "updateMask contains unsupported paths")
		return true
	}
	if len(maskPaths) == 0 {
		respondGCPRedisClusterInvalidArgument(w, path, "updateMask is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "updateCluster."+clusterID, expectedName, "updateCluster"))
	return true
}

func handleGCPRedisClusterDeleteCluster(w http.ResponseWriter, path string) bool {
	project, location, clusterID, ok := parseGCPRedisClusterClusterPath(path)
	if !ok {
		return false
	}
	clusterName := gcpRedisClusterClusterName(project, location, clusterID)
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "deleteCluster."+clusterID, clusterName, "deleteCluster"))
	return true
}

func handleGCPRedisClusterGetClusterCertificateAuthority(w http.ResponseWriter, path string) bool {
	project, location, clusterID, ok := parseGCPRedisClusterClusterCertificateAuthorityPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisClusterCertificateAuthorityFixture(project, location, clusterID))
	return true
}

func handleGCPRedisClusterRescheduleClusterMaintenance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPRedisClusterClusterActionPath(path, "rescheduleClusterMaintenance")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisClusterJSONBody(w, r, path)
	if !valid {
		return true
	}
	if name := gcpRedisClusterString(body, "name"); name != "" && name != gcpRedisClusterClusterName(project, location, clusterID) {
		respondGCPRedisClusterInvalidArgument(w, path, "name must match the requested resource")
		return true
	}

	rescheduleType, hasType := parseGCPRedisClusterRescheduleType(body["rescheduleType"])
	if !hasType {
		respondGCPRedisClusterInvalidArgument(w, path, "rescheduleType is required")
		return true
	}
	if rescheduleType == "SPECIFIC_TIME" && gcpRedisClusterString(body, "scheduleTime") == "" {
		respondGCPRedisClusterInvalidArgument(w, path, "scheduleTime is required when rescheduleType is SPECIFIC_TIME")
		return true
	}
	if rescheduleType == "IMMEDIATE" && strings.Contains(clusterID, "locked") {
		respondGCPRedisClusterFailedPrecondition(w, path, "cluster maintenance is locked")
		return true
	}

	clusterName := gcpRedisClusterClusterName(project, location, clusterID)
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "rescheduleClusterMaintenance."+clusterID, clusterName, "rescheduleClusterMaintenance"))
	return true
}

func handleGCPRedisClusterListBackupCollections(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterBackupCollectionsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRedisClusterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisClusterBackupCollectionFixture(project, location, "collection-1", "cluster-1"),
		gcpRedisClusterBackupCollectionFixture(project, location, "collection-2", "cluster-2"),
	}
	return respondGCPRedisClusterList(w, "backupCollections", items, pageSize, start, path)
}

func handleGCPRedisClusterGetBackupCollection(w http.ResponseWriter, path string) bool {
	project, location, collectionID, ok := parseGCPRedisClusterBackupCollectionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisClusterBackupCollectionFixture(project, location, collectionID, "cluster-1"))
	return true
}

func handleGCPRedisClusterListBackups(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collectionID, ok := parseGCPRedisClusterBackupsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRedisClusterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisClusterBackupFixture(project, location, collectionID, "backup-1", "cluster-1"),
		gcpRedisClusterBackupFixture(project, location, collectionID, "backup-2", "cluster-1"),
	}
	return respondGCPRedisClusterList(w, "backups", items, pageSize, start, path)
}

func handleGCPRedisClusterGetBackup(w http.ResponseWriter, path string) bool {
	project, location, collectionID, backupID, ok := parseGCPRedisClusterBackupPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisClusterBackupFixture(project, location, collectionID, backupID, "cluster-1"))
	return true
}

func handleGCPRedisClusterDeleteBackup(w http.ResponseWriter, path string) bool {
	project, location, collectionID, backupID, ok := parseGCPRedisClusterBackupPath(path)
	if !ok {
		return false
	}
	backupName := gcpRedisClusterBackupName(project, location, collectionID, backupID)
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "deleteBackup."+backupID, backupName, "deleteBackup"))
	return true
}

func handleGCPRedisClusterExportBackup(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collectionID, backupID, ok := parseGCPRedisClusterBackupActionPath(path, "export")
	if !ok {
		project, location, collectionID, backupID, ok = parseGCPRedisClusterExportBackupPath(path)
	}
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisClusterJSONBody(w, r, path)
	if !valid {
		return true
	}
	if name := gcpRedisClusterString(body, "name"); name != "" && name != gcpRedisClusterBackupName(project, location, collectionID, backupID) {
		respondGCPRedisClusterInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	gcsBucket := gcpRedisClusterString(body, "gcsBucket")
	if gcsBucket == "" {
		gcsBucket = gcpRedisClusterString(gcpRedisClusterBodyMap(body, "outputConfig"), "gcsBucket")
	}
	if gcsBucket == "" {
		respondGCPRedisClusterInvalidArgument(w, path, "gcsBucket is required")
		return true
	}
	backupName := gcpRedisClusterBackupName(project, location, collectionID, backupID)
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "exportBackup."+backupID, backupName, "exportBackup"))
	return true
}

func handleGCPRedisClusterBackupCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPRedisClusterClusterActionPath(path, "backup")
	if !ok {
		return false
	}
	body, valid := decodeGCPRedisClusterJSONBody(w, r, path)
	if !valid {
		return true
	}
	if name := gcpRedisClusterString(body, "name"); name != "" && name != gcpRedisClusterClusterName(project, location, clusterID) {
		respondGCPRedisClusterInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	if backupID := gcpRedisClusterString(body, "backupId"); backupID != "" && !gcpRedisClusterIDPattern.MatchString(backupID) {
		respondGCPRedisClusterInvalidArgument(w, path, "backupId is invalid")
		return true
	}
	clusterName := gcpRedisClusterClusterName(project, location, clusterID)
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, "backupCluster."+clusterID, clusterName, "backupCluster"))
	return true
}

func handleGCPRedisClusterListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRedisClusterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRedisClusterOperationFixture(project, location, "op-1", gcpRedisClusterClusterName(project, location, "cluster-1"), "updateCluster"),
		gcpRedisClusterOperationFixture(project, location, "op-2", gcpRedisClusterClusterName(project, location, "cluster-1"), "deleteBackup"),
	}
	return respondGCPRedisClusterList(w, "operations", items, pageSize, start, path)
}

func handleGCPRedisClusterGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPRedisClusterOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRedisClusterOperationFixture(project, location, operationID, gcpRedisClusterClusterName(project, location, "cluster-1"), "getOperation"))
	return true
}

func handleGCPRedisClusterCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, valid := decodeGCPRedisClusterJSONBody(w, r, path); !valid {
		return true
	}
	_, _, _, ok := parseGCPRedisClusterOperationActionPath(path, "cancel")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRedisClusterDeleteOperation(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPRedisClusterOperationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPRedisClusterLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPRedisClusterClustersCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "clusters"
}

func isGCPRedisClusterClusterTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "clusters" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRedisClusterClusterCertificateAuthorityTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "clusters" && strings.TrimSpace(tail[1]) != "" && tail[2] == "certificateAuthority"
}

func isGCPRedisClusterClusterActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "clusters" {
		return false
	}
	clusterID, parsedAction, found := splitGCPRedisClusterActionSegment(tail[1])
	return found && strings.TrimSpace(clusterID) != "" && parsedAction == action
}

func isGCPRedisClusterBackupCollectionsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "backupCollections"
}

func isGCPRedisClusterBackupCollectionTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "backupCollections" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRedisClusterBackupsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "backupCollections" && strings.TrimSpace(tail[1]) != "" && tail[2] == "backups"
}

func isGCPRedisClusterBackupTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "backupCollections" && strings.TrimSpace(tail[1]) != "" && tail[2] == "backups" && strings.TrimSpace(tail[3]) != "" && !strings.Contains(tail[3], ":")
}

func isGCPRedisClusterBackupActionTail(tail []string, action string) bool {
	if len(tail) != 4 || tail[0] != "backupCollections" || strings.TrimSpace(tail[1]) == "" || tail[2] != "backups" {
		return false
	}
	backupID, parsedAction, found := splitGCPRedisClusterActionSegment(tail[3])
	return found && strings.TrimSpace(backupID) != "" && parsedAction == action
}

func isGCPRedisClusterOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPRedisClusterOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRedisClusterOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	operationID, parsedAction, found := splitGCPRedisClusterActionSegment(tail[1])
	return found && strings.TrimSpace(operationID) != "" && parsedAction == action
}

func parseGCPRedisClusterClusterPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterClusterTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisClusterClusterCertificateAuthorityPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterClusterCertificateAuthorityTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisClusterClusterActionPath(path, action string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterClusterActionTail(tail, action) {
		return "", "", "", false
	}
	clusterID, parsedAction, _ := splitGCPRedisClusterActionSegment(tail[1])
	if parsedAction != action {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(clusterID), true
}

func parseGCPRedisClusterBackupCollectionPath(path string) (project, location, collectionID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterBackupCollectionTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisClusterBackupsCollectionPath(path string) (project, location, collectionID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterBackupsCollectionTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisClusterBackupPath(path string) (project, location, collectionID, backupID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterBackupTail(tail) {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPRedisClusterBackupActionPath(path, action string) (project, location, collectionID, backupID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterBackupActionTail(tail, action) {
		return "", "", "", "", false
	}
	backupID, parsedAction, _ := splitGCPRedisClusterActionSegment(tail[3])
	if parsedAction != action {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(backupID), true
}

func parseGCPRedisClusterOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterOperationTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRedisClusterOperationActionPath(path, action string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPRedisClusterLocationTail(path)
	if !ok || !isGCPRedisClusterOperationActionTail(tail, action) {
		return "", "", "", false
	}
	operationID, parsedAction, _ := splitGCPRedisClusterActionSegment(tail[1])
	if parsedAction != action {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(operationID), true
}

func splitGCPRedisClusterActionSegment(raw string) (id, action string, ok bool) {
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

func parseGCPRedisClusterExportBackupPath(path string) (project, location, collectionID, backupID string, ok bool) {
	matches := gcpRedisClusterExportPattern.FindStringSubmatch(strings.TrimSpace(path))
	if len(matches) != 5 {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(matches[1])
	location = strings.TrimSpace(matches[2])
	collectionID = strings.TrimSpace(matches[3])
	backupID = strings.TrimSpace(matches[4])
	if project == "" || location == "" || collectionID == "" || backupID == "" {
		return "", "", "", "", false
	}
	return project, location, collectionID, backupID, true
}

func parseGCPRedisClusterPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRedisClusterInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPRedisClusterInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPRedisClusterInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPRedisClusterList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPRedisClusterInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPRedisClusterJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPRedisClusterInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRedisClusterBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpRedisClusterString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func parseGCPRedisClusterUpdateMask(raw string) ([]string, bool) {
	allowed := map[string]struct{}{
		"sizegb":        {},
		"size_gb":       {},
		"replicacount":  {},
		"replica_count": {},
	}
	out := make([]string, 0, 2)
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

func parseGCPRedisClusterRescheduleType(raw any) (string, bool) {
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

func gcpRedisClusterClusterName(project, location, clusterID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterID)
}

func gcpRedisClusterBackupCollectionName(project, location, collectionID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/backupCollections/%s", project, location, collectionID)
}

func gcpRedisClusterBackupName(project, location, collectionID, backupID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/backupCollections/%s/backups/%s", project, location, collectionID, backupID)
}

func gcpRedisClusterLocationFixture(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Redis Cluster " + location,
		"labels": map[string]string{
			"service": "redis_cluster",
			"stage":   "emulated",
		},
	}
}

func gcpRedisClusterFixture(project, location, clusterID string) map[string]any {
	clusterName := gcpRedisClusterClusterName(project, location, clusterID)
	return map[string]any{
		"name":         clusterName,
		"createTime":   gcpRedisClusterReferenceTime.Format(time.RFC3339Nano),
		"state":        "ACTIVE",
		"uid":          "redis-cluster-" + clusterID,
		"shardCount":   3,
		"replicaCount": 1,
		"sizeGb":       12,
		"nodeType":     "REDIS_SHARED_CORE_NANO",
		"redisConfigs": map[string]string{
			"maxmemory-policy": "allkeys-lru",
		},
		"pscConfigs": []map[string]any{
			{"network": fmt.Sprintf("projects/%s/global/networks/default", project)},
		},
		"discoveryEndpoints": []map[string]any{
			{
				"address": "10.0.0.5",
				"port":    6379,
				"pscConfig": map[string]any{
					"network": fmt.Sprintf("projects/%s/global/networks/default", project),
				},
			},
		},
		"backupCollection": gcpRedisClusterBackupCollectionName(project, location, "collection-1"),
	}
}

func gcpRedisClusterCertificateAuthorityFixture(project, location, clusterID string) map[string]any {
	return map[string]any{
		"name": gcpRedisClusterClusterName(project, location, clusterID) + "/certificateAuthority",
		"managedServerCa": map[string]any{
			"caCerts": []map[string]any{
				{
					"certificates": []string{
						"-----BEGIN CERTIFICATE-----",
						"STACKYARD-REDIS-CLUSTER-CA",
						"-----END CERTIFICATE-----",
					},
				},
			},
		},
	}
}

func gcpRedisClusterBackupCollectionFixture(project, location, collectionID, clusterID string) map[string]any {
	return map[string]any{
		"name":       gcpRedisClusterBackupCollectionName(project, location, collectionID),
		"clusterUid": "redis-cluster-" + clusterID,
		"cluster":    gcpRedisClusterClusterName(project, location, clusterID),
		"kmsKey":     fmt.Sprintf("projects/%s/locations/%s/keyRings/stackyard/cryptoKeys/redis-cluster", project, location),
		"uid":        "backup-collection-" + collectionID,
	}
}

func gcpRedisClusterBackupFixture(project, location, collectionID, backupID, clusterID string) map[string]any {
	createTime := gcpRedisClusterReferenceTime
	return map[string]any{
		"name":           gcpRedisClusterBackupName(project, location, collectionID, backupID),
		"createTime":     createTime.Format(time.RFC3339Nano),
		"cluster":        gcpRedisClusterClusterName(project, location, clusterID),
		"clusterUid":     "redis-cluster-" + clusterID,
		"totalSizeBytes": 2147483648,
		"expireTime":     createTime.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"engineVersion":  "redis-7.2",
		"nodeType":       "REDIS_SHARED_CORE_NANO",
		"replicaCount":   1,
		"shardCount":     3,
		"backupType":     "ON_DEMAND",
		"state":          "ACTIVE",
		"uid":            "backup-" + backupID,
	}
}

func gcpRedisClusterOperationFixture(project, location, operationID, target, verb string) map[string]any {
	opName := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	return map[string]any{
		"name": opName,
		"done": false,
		"metadata": map[string]any{
			"@type":       "type.googleapis.com/google.cloud.redis.cluster.v1.OperationMetadata",
			"createTime":  gcpRedisClusterReferenceTime.Format(time.RFC3339Nano),
			"target":      target,
			"verb":        verb,
			"apiVersion":  "v1",
			"requestedBy": providerGCP,
		},
		"response": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	}
}

func respondGCPRedisClusterInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPRedisClusterError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPRedisClusterFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPRedisClusterError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPRedisClusterError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_redis_cluster(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "redis_cluster") {
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
			"name":     "projects/stackyard/locations/us-central1/redis_cluster/sample",
			"service":  "redis_cluster",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
