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

const gcpSpannerAdminDatabaseMaxPageSize = 1000

var gcpSpannerAdminDatabaseReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSpannerAdminDatabaseRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_spanner_admin_database(w, r) {
		return true
	}

	path := normalizeGCPSpannerAdminDatabasePath(rawRequestPath(r))
	if isGCPSpannerAdminDatabaseLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSpannerAdminDatabaseListLocations(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSpannerAdminDatabasePath(path, hasGCPSpannerAdminDatabaseHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSpannerAdminDatabaseListDatabases(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetDatabase(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetDatabaseDDL(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseListBackups(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetBackup(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseListDatabaseOperations(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseListBackupOperations(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseListDatabaseRoles(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetBackupSchedule(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseListBackupSchedules(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetOperation(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseListOperations(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSpannerAdminDatabaseCreateDatabase(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseCreateBackup(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseCopyBackup(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseRestoreDatabase(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseAddSplitPoints(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseCreateBackupSchedule(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseCancelOperation(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseGetIAMPolicy(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSpannerAdminDatabaseUpdateDatabase(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseUpdateDatabaseDDL(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseUpdateBackup(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseUpdateBackupSchedule(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSpannerAdminDatabaseDropDatabase(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseDeleteBackup(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseDeleteBackupSchedule(w, path) {
			return true
		}
		if handleGCPSpannerAdminDatabaseDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSpannerAdminDatabasePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSpannerAdminDatabaseHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "spanner_admin_database", "spanner-admin-database", "spanner-admin-database-apiv1", "spanner_admin_database_apiv1", "cloud-spanner-admin-database", "cloud_spanner_admin_database", "cloudspanneradmindatabase", "gcp-cloud-spanner-admin-database":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-spanner-admin-database-apiv1") || strings.Contains(ua, "cloud.google.com/go/spanner/admin/database")
}

func isGCPSpannerAdminDatabaseLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPSpannerAdminDatabaseHint(r)
}

func isGCPSpannerAdminDatabasePath(path string, includeHint bool) bool {
	_, _, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if ok {
		if isGCPSpannerAdminDatabaseDatabasesCollectionTail(tail) ||
			isGCPSpannerAdminDatabaseDatabaseTail(tail) ||
			isGCPSpannerAdminDatabaseDatabaseDDLTail(tail) ||
			isGCPSpannerAdminDatabaseBackupsCollectionTail(tail) ||
			isGCPSpannerAdminDatabaseBackupsCopyTail(tail) ||
			isGCPSpannerAdminDatabaseBackupTail(tail) ||
			isGCPSpannerAdminDatabaseRestoreTail(tail) ||
			isGCPSpannerAdminDatabaseDatabaseOperationsTail(tail) ||
			isGCPSpannerAdminDatabaseBackupOperationsTail(tail) ||
			isGCPSpannerAdminDatabaseDatabaseRolesTail(tail) ||
			isGCPSpannerAdminDatabaseAddSplitPointsTail(tail) ||
			isGCPSpannerAdminDatabaseBackupSchedulesCollectionTail(tail) ||
			isGCPSpannerAdminDatabaseBackupScheduleTail(tail) ||
			isGCPSpannerAdminDatabaseOperationsCollectionTail(tail) ||
			isGCPSpannerAdminDatabaseOperationTail(tail) ||
			isGCPSpannerAdminDatabaseOperationActionTail(tail, "cancel") {
			return true
		}
	}

	if _, _, ok := parseGCPSpannerAdminDatabaseIAMActionPath(path); ok {
		return true
	}

	if includeHint {
		return strings.HasPrefix(path, "/gcp/v1/projects/") && strings.Contains(path, "/instances/")
	}
	return false
}

func handleGCPSpannerAdminDatabaseListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseLocationFixture(project, "us-central1"),
		gcpSpannerAdminDatabaseLocationFixture(project, "global"),
	}
	return respondGCPSpannerAdminDatabaseList(w, "locations", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabaseLocationFixture(project, location))
	return true
}

func handleGCPSpannerAdminDatabaseListDatabases(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseDatabasesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseDatabaseFixture(project, instance, "stackyard-db"),
		gcpSpannerAdminDatabaseDatabaseFixture(project, instance, "analytics-db"),
	}
	return respondGCPSpannerAdminDatabaseList(w, "databases", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseCreateDatabase(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseDatabasesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyParent := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "parent")); bodyParent != "" {
		bp, bi, ok := parseGCPSpannerAdminDatabaseInstanceName(bodyParent)
		if !ok || bp != project || bi != instance {
			respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "parent must match requested instance")
			return true
		}
	}
	createStatement := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "createStatement"))
	if createStatement == "" {
		createStatement = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "create_statement"))
	}
	if createStatement == "" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "createStatement is required")
		return true
	}
	databaseID, ok := gcpSpannerAdminDatabaseIDFromCreateStatement(createStatement)
	if !ok {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "createStatement must contain a valid database identifier")
		return true
	}
	if isGCPSpannerAdminDatabaseAlreadyExists(databaseID) {
		respondGCPSpannerAdminDatabaseAlreadyExists(w, path, "database already exists")
		return true
	}
	response := gcpSpannerAdminDatabaseDatabaseFixture(project, instance, databaseID)
	response["state"] = "CREATING"
	operation := gcpSpannerAdminDatabaseOperationFixture(project, instance, "create-database-"+databaseID, response)
	operation["metadata"] = map[string]any{
		"@type":           "type.googleapis.com/google.spanner.admin.database.v1.CreateDatabaseMetadata",
		"database":        fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID),
		"createTime":      gcpSpannerAdminDatabaseReferenceTime.Format(time.RFC3339),
		"progress":        map[string]any{"progressPercent": 100},
		"createStatement": createStatement,
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPSpannerAdminDatabaseGetDatabase(w http.ResponseWriter, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseDatabasePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabaseDatabaseFixture(project, instance, databaseID))
	return true
}

func handleGCPSpannerAdminDatabaseUpdateDatabase(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseDatabasePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	database := gcpSpannerAdminDatabaseBodyMap(body, "database")
	if len(database) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "database is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID)
	if name := strings.TrimSpace(gcpSpannerAdminDatabaseString(database, "name")); name == "" || name != expectedName {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "database.name must match requested resource")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "updateMask"))
	}
	if updateMask == "" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "updateMask is required")
		return true
	}
	response := gcpSpannerAdminDatabaseDatabaseFixture(project, instance, databaseID)
	if v, ok := database["enableDropProtection"].(bool); ok {
		response["enableDropProtection"] = v
	}
	operation := gcpSpannerAdminDatabaseOperationFixture(project, instance, "update-database-"+databaseID, response)
	operation["metadata"] = map[string]any{
		"@type":      "type.googleapis.com/google.spanner.admin.database.v1.UpdateDatabaseMetadata",
		"name":       expectedName,
		"updateMask": updateMask,
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPSpannerAdminDatabaseUpdateDatabaseDDL(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseDatabaseDDLPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	databaseName := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "database"))
	expectedDatabaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID)
	if databaseName == "" || databaseName != expectedDatabaseName {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "database must match requested resource")
		return true
	}
	statements := gcpSpannerAdminDatabaseStringSlice(body["statements"])
	if len(statements) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "statements is required")
		return true
	}
	operationID := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "operationId"))
	if operationID == "" {
		operationID = "update-ddl-" + databaseID
	}
	if isGCPSpannerAdminDatabaseAlreadyExists(operationID) {
		respondGCPSpannerAdminDatabaseAlreadyExists(w, path, "operation already exists")
		return true
	}
	operation := gcpSpannerAdminDatabaseOperationFixture(project, instance, operationID, map[string]any{})
	operation["metadata"] = map[string]any{
		"@type":      "type.googleapis.com/google.spanner.admin.database.v1.UpdateDatabaseDdlMetadata",
		"database":   expectedDatabaseName,
		"statements": statements,
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPSpannerAdminDatabaseDropDatabase(w http.ResponseWriter, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseDatabasePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminDatabaseGetDatabaseDDL(w http.ResponseWriter, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseDatabaseDDLPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"statements": []string{
			fmt.Sprintf("CREATE TABLE Users_%s (UserId STRING(36) NOT NULL, Name STRING(256)) PRIMARY KEY (UserId)", databaseID),
			"CREATE INDEX UsersByName ON Users (Name)",
		},
	})
	return true
}

func handleGCPSpannerAdminDatabaseSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPSpannerAdminDatabaseIAMActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(resource) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "resource not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpSpannerAdminDatabaseBodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "policy is required")
		return true
	}
	response := gcpSpannerAdminDatabasePolicyFixture(resource)
	if bindings, ok := policy["bindings"].([]any); ok {
		response["bindings"] = bindings
	}
	if etag := strings.TrimSpace(gcpSpannerAdminDatabaseString(policy, "etag")); etag != "" {
		response["etag"] = etag
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpannerAdminDatabaseGetIAMPolicy(w http.ResponseWriter, path string) bool {
	resource, action, ok := parseGCPSpannerAdminDatabaseIAMActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(resource) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "resource not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabasePolicyFixture(resource))
	return true
}

func handleGCPSpannerAdminDatabaseTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPSpannerAdminDatabaseIAMActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(resource) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "resource not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions := gcpSpannerAdminDatabaseStringSlice(body["permissions"])
	if len(permissions) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "permissions is required")
		return true
	}
	filtered := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		if strings.Contains(permission, "spanner") {
			filtered = append(filtered, permission)
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": filtered})
	return true
}

func handleGCPSpannerAdminDatabaseCreateBackup(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupsCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyParent := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "parent")); bodyParent != "" {
		bp, bi, ok := parseGCPSpannerAdminDatabaseInstanceName(bodyParent)
		if !ok || bp != project || bi != instance {
			respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "parent must match requested instance")
			return true
		}
	}
	backupID := strings.TrimSpace(r.URL.Query().Get("backupId"))
	if backupID == "" {
		backupID = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "backupId"))
	}
	if !isGCPSpannerAdminDatabaseIdentifier(backupID, 60) {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupId is required")
		return true
	}
	backup := gcpSpannerAdminDatabaseBodyMap(body, "backup")
	if len(backup) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup is required")
		return true
	}
	databaseName := strings.TrimSpace(gcpSpannerAdminDatabaseString(backup, "database"))
	bp, bi, databaseID, ok := parseGCPSpannerDatabaseName(databaseName)
	if !ok {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup.database is required")
		return true
	}
	if bp != project || bi != instance {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup.database must belong to the requested instance")
		return true
	}
	if strings.TrimSpace(gcpSpannerAdminDatabaseString(backup, "expireTime")) == "" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup.expireTime is required")
		return true
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	if isGCPSpannerAdminDatabaseAlreadyExists(backupID) {
		respondGCPSpannerAdminDatabaseAlreadyExists(w, path, "backup already exists")
		return true
	}
	backupResponse := gcpSpannerAdminDatabaseBackupFixture(project, instance, backupID, databaseID)
	operation := gcpSpannerAdminDatabaseOperationFixture(project, instance, "create-backup-"+backupID, backupResponse)
	operation["metadata"] = map[string]any{
		"@type":    "type.googleapis.com/google.spanner.admin.database.v1.CreateBackupMetadata",
		"name":     backupResponse["name"],
		"database": backupResponse["database"],
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPSpannerAdminDatabaseCopyBackup(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupsCopyTail(tail) {
		return false
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyParent := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "parent")); bodyParent != "" {
		bp, bi, ok := parseGCPSpannerAdminDatabaseInstanceName(bodyParent)
		if !ok || bp != project || bi != instance {
			respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "parent must match requested instance")
			return true
		}
	}
	backupID := strings.TrimSpace(r.URL.Query().Get("backupId"))
	if backupID == "" {
		backupID = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "backupId"))
	}
	if !isGCPSpannerAdminDatabaseIdentifier(backupID, 60) {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupId is required")
		return true
	}
	sourceBackup := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "sourceBackup"))
	sourceProject, sourceInstance, sourceBackupID, ok := parseGCPSpannerAdminDatabaseBackupName(sourceBackup)
	if !ok {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "sourceBackup is required")
		return true
	}
	if sourceProject != project || sourceInstance != instance {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "sourceBackup must belong to the requested instance")
		return true
	}
	if strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "expireTime")) == "" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "expireTime is required")
		return true
	}
	if isGCPSpannerAdminDatabaseMissingResource(sourceBackupID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "source backup not found")
		return true
	}
	if strings.Contains(strings.ToLower(sourceBackupID), "creating") {
		respondGCPSpannerAdminDatabaseFailedPrecondition(w, path, "source backup must be in READY state")
		return true
	}
	if isGCPSpannerAdminDatabaseAlreadyExists(backupID) {
		respondGCPSpannerAdminDatabaseAlreadyExists(w, path, "backup already exists")
		return true
	}
	backupResponse := gcpSpannerAdminDatabaseBackupFixture(project, instance, backupID, "stackyard-db")
	operation := gcpSpannerAdminDatabaseOperationFixture(project, instance, "copy-backup-"+backupID, backupResponse)
	operation["metadata"] = map[string]any{
		"@type":        "type.googleapis.com/google.spanner.admin.database.v1.CopyBackupMetadata",
		"name":         backupResponse["name"],
		"sourceBackup": sourceBackup,
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPSpannerAdminDatabaseGetBackup(w http.ResponseWriter, path string) bool {
	project, instance, backupID, ok := parseGCPSpannerAdminDatabaseBackupPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, backupID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabaseBackupFixture(project, instance, backupID, "stackyard-db"))
	return true
}

func handleGCPSpannerAdminDatabaseUpdateBackup(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, backupID, ok := parseGCPSpannerAdminDatabaseBackupPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, backupID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	backup := gcpSpannerAdminDatabaseBodyMap(body, "backup")
	if len(backup) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instances/%s/backups/%s", project, instance, backupID)
	if name := strings.TrimSpace(gcpSpannerAdminDatabaseString(backup, "name")); name == "" || name != expectedName {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup.name must match requested resource")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "updateMask"))
	}
	if updateMask == "" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "updateMask is required")
		return true
	}
	response := gcpSpannerAdminDatabaseBackupFixture(project, instance, backupID, "stackyard-db")
	if expireTime := strings.TrimSpace(gcpSpannerAdminDatabaseString(backup, "expireTime")); expireTime != "" {
		response["expireTime"] = expireTime
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpannerAdminDatabaseDeleteBackup(w http.ResponseWriter, path string) bool {
	project, instance, backupID, ok := parseGCPSpannerAdminDatabaseBackupPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, backupID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminDatabaseListBackups(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	_ = strings.TrimSpace(r.URL.Query().Get("filter"))
	items := []map[string]any{
		gcpSpannerAdminDatabaseBackupFixture(project, instance, "backup-1", "stackyard-db"),
		gcpSpannerAdminDatabaseBackupFixture(project, instance, "backup-2", "analytics-db"),
	}
	return respondGCPSpannerAdminDatabaseList(w, "backups", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseRestoreDatabase(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseRestoreTail(tail) {
		return false
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyParent := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "parent")); bodyParent != "" {
		bp, bi, ok := parseGCPSpannerAdminDatabaseInstanceName(bodyParent)
		if !ok || bp != project || bi != instance {
			respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "parent must match requested instance")
			return true
		}
	}
	databaseID := strings.TrimSpace(r.URL.Query().Get("databaseId"))
	if databaseID == "" {
		databaseID = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "databaseId"))
	}
	if !isGCPSpannerAdminDatabaseIdentifier(databaseID, 30) {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "databaseId is required")
		return true
	}
	sourceBackup := strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "backup"))
	_, _, sourceBackupID, ok := parseGCPSpannerAdminDatabaseBackupName(sourceBackup)
	if !ok {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backup is required")
		return true
	}
	if isGCPSpannerAdminDatabaseMissingResource(sourceBackupID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup not found")
		return true
	}
	if strings.Contains(strings.ToLower(sourceBackupID), "creating") {
		respondGCPSpannerAdminDatabaseFailedPrecondition(w, path, "backup must be in READY state")
		return true
	}
	if isGCPSpannerAdminDatabaseAlreadyExists(databaseID) {
		respondGCPSpannerAdminDatabaseAlreadyExists(w, path, "database already exists")
		return true
	}
	databaseResponse := gcpSpannerAdminDatabaseDatabaseFixture(project, instance, databaseID)
	databaseResponse["restoreInfo"] = map[string]any{
		"backupInfo": map[string]any{"backup": sourceBackup},
		"sourceType": "BACKUP",
	}
	operation := gcpSpannerAdminDatabaseOperationFixture(project, instance, "restore-database-"+databaseID, databaseResponse)
	operation["metadata"] = map[string]any{
		"@type":      "type.googleapis.com/google.spanner.admin.database.v1.RestoreDatabaseMetadata",
		"name":       databaseResponse["name"],
		"sourceType": "BACKUP",
		"backupInfo": map[string]any{"backup": sourceBackup},
	}
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPSpannerAdminDatabaseListDatabaseOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseDatabaseOperationsTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseOperationFixture(project, instance, "create-database-stackyard-db", gcpSpannerAdminDatabaseDatabaseFixture(project, instance, "stackyard-db")),
		gcpSpannerAdminDatabaseOperationFixture(project, instance, "update-ddl-stackyard-db", map[string]any{}),
	}
	return respondGCPSpannerAdminDatabaseList(w, "operations", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseListBackupOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupOperationsTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseOperationFixture(project, instance, "create-backup-backup-1", gcpSpannerAdminDatabaseBackupFixture(project, instance, "backup-1", "stackyard-db")),
	}
	return respondGCPSpannerAdminDatabaseList(w, "operations", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseListDatabaseRoles(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseDatabaseRolesPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseRoleFixture(project, instance, databaseID, "reader"),
		gcpSpannerAdminDatabaseRoleFixture(project, instance, databaseID, "writer"),
	}
	return respondGCPSpannerAdminDatabaseList(w, "databaseRoles", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseAddSplitPoints(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseAddSplitPointsPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	splitPoints, ok := body["splitPoints"].([]any)
	if !ok || len(splitPoints) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "splitPoints is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminDatabaseCreateBackupSchedule(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseBackupSchedulesCollectionPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	scheduleID := strings.TrimSpace(r.URL.Query().Get("backupScheduleId"))
	if scheduleID == "" {
		scheduleID = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "backupScheduleId"))
	}
	if !isGCPSpannerAdminDatabaseIdentifier(scheduleID, 60) {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupScheduleId is required")
		return true
	}
	schedule := gcpSpannerAdminDatabaseBodyMap(body, "backupSchedule")
	if len(schedule) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupSchedule is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instances/%s/databases/%s/backupSchedules/%s", project, instance, databaseID, scheduleID)
	if provided := strings.TrimSpace(gcpSpannerAdminDatabaseString(schedule, "name")); provided != "" && provided != expectedName {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupSchedule.name must match requested resource")
		return true
	}
	if isGCPSpannerAdminDatabaseAlreadyExists(scheduleID) {
		respondGCPSpannerAdminDatabaseAlreadyExists(w, path, "backup schedule already exists")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabaseBackupScheduleFixture(project, instance, databaseID, scheduleID))
	return true
}

func handleGCPSpannerAdminDatabaseGetBackupSchedule(w http.ResponseWriter, path string) bool {
	project, instance, databaseID, scheduleID, ok := parseGCPSpannerAdminDatabaseBackupSchedulePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID, scheduleID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup schedule not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabaseBackupScheduleFixture(project, instance, databaseID, scheduleID))
	return true
}

func handleGCPSpannerAdminDatabaseUpdateBackupSchedule(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, scheduleID, ok := parseGCPSpannerAdminDatabaseBackupSchedulePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID, scheduleID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup schedule not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminDatabaseJSONBody(w, r, path)
	if !valid {
		return true
	}
	schedule := gcpSpannerAdminDatabaseBodyMap(body, "backupSchedule")
	if len(schedule) == 0 {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupSchedule is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instances/%s/databases/%s/backupSchedules/%s", project, instance, databaseID, scheduleID)
	if name := strings.TrimSpace(gcpSpannerAdminDatabaseString(schedule, "name")); name == "" || name != expectedName {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "backupSchedule.name must match requested resource")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminDatabaseString(body, "updateMask"))
	}
	if updateMask == "" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "updateMask is required")
		return true
	}
	response := gcpSpannerAdminDatabaseBackupScheduleFixture(project, instance, databaseID, scheduleID)
	if retentionDuration := strings.TrimSpace(gcpSpannerAdminDatabaseString(schedule, "retentionDuration")); retentionDuration != "" {
		response["retentionDuration"] = retentionDuration
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpannerAdminDatabaseDeleteBackupSchedule(w http.ResponseWriter, path string) bool {
	project, instance, databaseID, scheduleID, ok := parseGCPSpannerAdminDatabaseBackupSchedulePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID, scheduleID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "backup schedule not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminDatabaseListBackupSchedules(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, databaseID, ok := parseGCPSpannerAdminDatabaseBackupSchedulesCollectionPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "database not found")
		return true
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseBackupScheduleFixture(project, instance, databaseID, "daily-full"),
		gcpSpannerAdminDatabaseBackupScheduleFixture(project, instance, databaseID, "hourly-incremental"),
	}
	return respondGCPSpannerAdminDatabaseList(w, "backupSchedules", items, pageSize, start, path)
}

func handleGCPSpannerAdminDatabaseCancelOperation(w http.ResponseWriter, path string) bool {
	project, instance, operationID, ok := parseGCPSpannerAdminDatabaseOperationActionPath(path, "cancel")
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, operationID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminDatabaseDeleteOperation(w http.ResponseWriter, path string) bool {
	project, instance, operationID, ok := parseGCPSpannerAdminDatabaseOperationPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, operationID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminDatabaseGetOperation(w http.ResponseWriter, path string) bool {
	project, instance, operationID, ok := parseGCPSpannerAdminDatabaseOperationPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminDatabaseMissingResource(project, instance, operationID) {
		respondGCPSpannerAdminDatabaseNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminDatabaseOperationFixture(project, instance, operationID, map[string]any{}))
	return true
}

func handleGCPSpannerAdminDatabaseListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminDatabasePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminDatabaseOperationFixture(project, instance, "create-database-stackyard-db", gcpSpannerAdminDatabaseDatabaseFixture(project, instance, "stackyard-db")),
		gcpSpannerAdminDatabaseOperationFixture(project, instance, "create-backup-backup-1", gcpSpannerAdminDatabaseBackupFixture(project, instance, "backup-1", "stackyard-db")),
	}
	return respondGCPSpannerAdminDatabaseList(w, "operations", items, pageSize, start, path)
}

func parseGCPSpannerAdminDatabaseInstanceTail(path string) (project, instance string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 {
		return "", "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "instances" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	instance = strings.TrimSpace(parts[5])
	if project == "" || instance == "" {
		return "", "", nil, false
	}
	return project, instance, parts[6:], true
}

func parseGCPSpannerAdminDatabaseInstanceName(name string) (project, instance string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "instances" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	if project == "" || instance == "" {
		return "", "", false
	}
	return project, instance, true
}

func parseGCPSpannerAdminDatabaseBackupName(name string) (project, instance, backupID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "backups" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	backupID = strings.TrimSpace(parts[5])
	if project == "" || instance == "" || backupID == "" || strings.Contains(backupID, ":") {
		return "", "", "", false
	}
	return project, instance, backupID, true
}

func parseGCPSpannerAdminDatabaseBackupScheduleName(name string) (project, instance, databaseID, scheduleID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "databases" || parts[6] != "backupSchedules" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	databaseID = strings.TrimSpace(parts[5])
	scheduleID = strings.TrimSpace(parts[7])
	if project == "" || instance == "" || databaseID == "" || scheduleID == "" || strings.Contains(databaseID, ":") || strings.Contains(scheduleID, ":") {
		return "", "", "", "", false
	}
	return project, instance, databaseID, scheduleID, true
}

func parseGCPSpannerAdminDatabaseOperationName(name string) (project, instance, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if project == "" || instance == "" || operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return project, instance, operationID, true
}

func parseGCPSpannerAdminDatabaseDatabasePath(path string) (project, instance, databaseID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseDatabaseTail(tail) {
		return "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), true
}

func parseGCPSpannerAdminDatabaseDatabaseDDLPath(path string) (project, instance, databaseID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseDatabaseDDLTail(tail) {
		return "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), true
}

func parseGCPSpannerAdminDatabaseBackupPath(path string) (project, instance, backupID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupTail(tail) {
		return "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), true
}

func parseGCPSpannerAdminDatabaseDatabaseRolesPath(path string) (project, instance, databaseID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseDatabaseRolesTail(tail) {
		return "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), true
}

func parseGCPSpannerAdminDatabaseAddSplitPointsPath(path string) (project, instance, databaseID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseAddSplitPointsTail(tail) {
		return "", "", "", false
	}
	databaseID, _, _ = strings.Cut(strings.TrimSpace(tail[1]), ":")
	return project, instance, strings.TrimSpace(databaseID), true
}

func parseGCPSpannerAdminDatabaseBackupSchedulesCollectionPath(path string) (project, instance, databaseID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupSchedulesCollectionTail(tail) {
		return "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), true
}

func parseGCPSpannerAdminDatabaseBackupSchedulePath(path string) (project, instance, databaseID, scheduleID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseBackupScheduleTail(tail) {
		return "", "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPSpannerAdminDatabaseOperationPath(path string) (project, instance, operationID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseOperationTail(tail) {
		return "", "", "", false
	}
	return project, instance, strings.TrimSpace(tail[1]), true
}

func parseGCPSpannerAdminDatabaseOperationActionPath(path, action string) (project, instance, operationID string, ok bool) {
	project, instance, tail, ok := parseGCPSpannerAdminDatabaseInstanceTail(path)
	if !ok || !isGCPSpannerAdminDatabaseOperationActionTail(tail, action) {
		return "", "", "", false
	}
	operationID, _, _ = strings.Cut(strings.TrimSpace(tail[1]), ":")
	return project, instance, strings.TrimSpace(operationID), true
}

func parseGCPSpannerAdminDatabaseIAMActionPath(path string) (resource, action string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if !strings.HasPrefix(trimmed, "gcp/v1/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(trimmed, "gcp/v1/")
	resource, action, ok = strings.Cut(rest, ":")
	if !ok {
		return "", "", false
	}
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" {
		return "", "", false
	}
	switch strings.ToLower(action) {
	case "setiampolicy":
		action = "setIamPolicy"
	case "getiampolicy":
		action = "getIamPolicy"
	case "testiampermissions":
		action = "testIamPermissions"
	default:
		return "", "", false
	}
	if _, _, _, ok := parseGCPSpannerDatabaseName(resource); ok {
		return resource, action, true
	}
	if _, _, _, ok := parseGCPSpannerAdminDatabaseBackupName(resource); ok {
		return resource, action, true
	}
	return "", "", false
}

func isGCPSpannerAdminDatabaseDatabasesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "databases"
}

func isGCPSpannerAdminDatabaseDatabaseTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "databases" {
		return false
	}
	databaseID := strings.TrimSpace(tail[1])
	return databaseID != "" && !strings.Contains(databaseID, ":")
}

func isGCPSpannerAdminDatabaseDatabaseDDLTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "databases" && strings.TrimSpace(tail[1]) != "" && tail[2] == "ddl"
}

func isGCPSpannerAdminDatabaseBackupsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "backups"
}

func isGCPSpannerAdminDatabaseBackupsCopyTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "backups:copy"
}

func isGCPSpannerAdminDatabaseBackupTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "backups" {
		return false
	}
	backupID := strings.TrimSpace(tail[1])
	return backupID != "" && !strings.Contains(backupID, ":")
}

func isGCPSpannerAdminDatabaseRestoreTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "databases:restore"
}

func isGCPSpannerAdminDatabaseDatabaseOperationsTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "databaseOperations"
}

func isGCPSpannerAdminDatabaseBackupOperationsTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "backupOperations"
}

func isGCPSpannerAdminDatabaseDatabaseRolesTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "databases" && strings.TrimSpace(tail[1]) != "" && tail[2] == "databaseRoles"
}

func isGCPSpannerAdminDatabaseAddSplitPointsTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "databases" {
		return false
	}
	databaseID, action, ok := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return ok && strings.TrimSpace(databaseID) != "" && action == "addSplitPoints"
}

func isGCPSpannerAdminDatabaseBackupSchedulesCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "databases" && strings.TrimSpace(tail[1]) != "" && tail[2] == "backupSchedules"
}

func isGCPSpannerAdminDatabaseBackupScheduleTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "databases" && strings.TrimSpace(tail[1]) != "" && tail[2] == "backupSchedules" && strings.TrimSpace(tail[3]) != "" && !strings.Contains(tail[3], ":")
}

func isGCPSpannerAdminDatabaseOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPSpannerAdminDatabaseOperationTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	return operationID != "" && !strings.Contains(operationID, ":")
}

func isGCPSpannerAdminDatabaseOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	operationID, parsedAction, ok := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return ok && strings.TrimSpace(operationID) != "" && parsedAction == action
}

func decodeGCPSpannerAdminDatabaseJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, 1<<20)
	dec := json.NewDecoder(limited)
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPSpannerAdminDatabasePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > gcpSpannerAdminDatabaseMaxPageSize {
			respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "pageSize must be a non-negative integer up to 1000")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func respondGCPSpannerAdminDatabaseList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	payload := map[string]any{field: items[start:end]}
	if end < len(items) {
		payload["nextPageToken"] = strconv.Itoa(end)
	} else {
		payload["nextPageToken"] = ""
	}
	respondJSON(w, http.StatusOK, payload)
	return true
}

func gcpSpannerAdminDatabaseLocationFixture(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Spanner Admin Database " + location,
		"labels": map[string]string{
			"service": "spanner_admin_database",
			"stage":   "emulated",
		},
	}
}

func gcpSpannerAdminDatabaseDatabaseFixture(project, instance, databaseID string) map[string]any {
	name := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID)
	return map[string]any{
		"name":                   name,
		"state":                  "READY",
		"createTime":             gcpSpannerAdminDatabaseReferenceTime.Add(2 * time.Minute).Format(time.RFC3339),
		"versionRetentionPeriod": "604800s",
		"defaultLeader":          "regional-us-central1",
		"databaseDialect":        "GOOGLE_STANDARD_SQL",
		"enableDropProtection":   true,
		"reconciling":            false,
	}
}

func gcpSpannerAdminDatabaseBackupFixture(project, instance, backupID, databaseID string) map[string]any {
	if databaseID == "" {
		databaseID = "stackyard-db"
	}
	name := fmt.Sprintf("projects/%s/instances/%s/backups/%s", project, instance, backupID)
	return map[string]any{
		"name":              name,
		"database":          fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID),
		"state":             "READY",
		"createTime":        gcpSpannerAdminDatabaseReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		"expireTime":        gcpSpannerAdminDatabaseReferenceTime.Add(24 * time.Hour).Format(time.RFC3339),
		"versionTime":       gcpSpannerAdminDatabaseReferenceTime.Add(4 * time.Minute).Format(time.RFC3339),
		"sizeBytes":         "1024",
		"freeableSizeBytes": "1024",
		"databaseDialect":   "GOOGLE_STANDARD_SQL",
	}
}

func gcpSpannerAdminDatabaseBackupScheduleFixture(project, instance, databaseID, scheduleID string) map[string]any {
	name := fmt.Sprintf("projects/%s/instances/%s/databases/%s/backupSchedules/%s", project, instance, databaseID, scheduleID)
	return map[string]any{
		"name":              name,
		"retentionDuration": "172800s",
		"spec": map[string]any{
			"cronSpec": map[string]any{"text": "0 */6 * * *"},
		},
		"fullBackupSpec": map[string]any{},
		"updateTime":     gcpSpannerAdminDatabaseReferenceTime.Add(6 * time.Minute).Format(time.RFC3339),
	}
}

func gcpSpannerAdminDatabaseRoleFixture(project, instance, databaseID, roleID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/instances/%s/databases/%s/databaseRoles/%s", project, instance, databaseID, roleID),
	}
}

func gcpSpannerAdminDatabasePolicyFixture(resource string) map[string]any {
	return map[string]any{
		"version": 1,
		"etag":    "etag-spanner-admin-database",
		"bindings": []any{
			map[string]any{
				"role":    "roles/spanner.databaseAdmin",
				"members": []string{"user:stackyard@example.com"},
			},
		},
		"resource": resource,
	}
}

func gcpSpannerAdminDatabaseOperationFixture(project, instance, operationID string, response map[string]any) map[string]any {
	if response == nil {
		response = map[string]any{}
	}
	typeURL := "type.googleapis.com/google.protobuf.Empty"
	if name, _ := response["name"].(string); strings.Contains(name, "/databases/") && strings.Contains(name, "/backupSchedules/") {
		typeURL = "type.googleapis.com/google.spanner.admin.database.v1.BackupSchedule"
	} else if _, ok := response["database"]; ok {
		typeURL = "type.googleapis.com/google.spanner.admin.database.v1.Backup"
	} else if _, ok := response["databaseDialect"]; ok {
		typeURL = "type.googleapis.com/google.spanner.admin.database.v1.Database"
	}
	responseAny := map[string]any{
		"@type": typeURL,
	}
	for key, value := range response {
		responseAny[key] = value
	}
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/instances/%s/operations/%s", project, instance, operationID),
		"done": true,
		"metadata": map[string]any{
			"progress": map[string]any{"progressPercent": 100},
		},
		"response": responseAny,
	}
}

func gcpSpannerAdminDatabaseString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	value, ok := body[key]
	if !ok || value == nil {
		return ""
	}
	text, _ := value.(string)
	return text
}

func gcpSpannerAdminDatabaseBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	nested, _ := body[key].(map[string]any)
	if nested == nil {
		return map[string]any{}
	}
	return nested
}

func gcpSpannerAdminDatabaseStringSlice(value any) []string {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		text, _ := item.(string)
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		out = append(out, text)
	}
	return out
}

func gcpSpannerAdminDatabaseIDFromCreateStatement(statement string) (string, bool) {
	raw := strings.TrimSpace(statement)
	raw = strings.TrimSuffix(raw, ";")
	if raw == "" {
		return "", false
	}
	lower := strings.ToLower(raw)
	prefix := "create database "
	if !strings.HasPrefix(lower, prefix) {
		return "", false
	}
	id := strings.TrimSpace(raw[len(prefix):])
	if idx := strings.IndexAny(id, " \t\n\r"); idx >= 0 {
		id = id[:idx]
	}
	id = strings.Trim(strings.TrimSpace(id), "`")
	if !isGCPSpannerAdminDatabaseIdentifier(id, 30) {
		return "", false
	}
	return id, true
}

func isGCPSpannerAdminDatabaseIdentifier(id string, maxLen int) bool {
	id = strings.TrimSpace(id)
	if len(id) < 2 || len(id) > maxLen {
		return false
	}
	for idx, ch := range id {
		if idx == 0 {
			if ch < 'a' || ch > 'z' {
				return false
			}
			continue
		}
		if idx == len(id)-1 {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
				return false
			}
			continue
		}
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

func isGCPSpannerAdminDatabaseMissingResource(parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(strings.ToLower(strings.TrimSpace(part)), "missing") {
			return true
		}
	}
	return false
}

func isGCPSpannerAdminDatabaseAlreadyExists(parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(strings.ToLower(strings.TrimSpace(part)), "existing") {
			return true
		}
	}
	return false
}

func respondGCPSpannerAdminDatabaseInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminDatabaseError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpannerAdminDatabaseNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminDatabaseError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpannerAdminDatabaseFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminDatabaseError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpannerAdminDatabaseAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminDatabaseError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPSpannerAdminDatabaseError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_spanner_admin_database(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "spanner_admin_database") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpannerAdminDatabaseInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/instances/stackyard-instance/databases/stackyard-db",
			"service":  "spanner_admin_database",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
