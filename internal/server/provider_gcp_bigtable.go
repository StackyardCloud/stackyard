package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPBigtableRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPBigtablePath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPBigtableListInstances(w, r, path) {
			return true
		}
		if handleGCPBigtableGetInstance(w, path) {
			return true
		}
		if handleGCPBigtableListTables(w, r, path) {
			return true
		}
		if handleGCPBigtableGetTable(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPBigtableCreateTable(w, r, path) {
			return true
		}
		if handleGCPBigtableDropRowRange(w, r, path) {
			return true
		}
		if handleGCPBigtableReadRows(w, path) {
			return true
		}
		if handleGCPBigtableSampleRowKeys(w, path) {
			return true
		}
		if handleGCPBigtableMutateRow(w, r, path) {
			return true
		}
		if handleGCPBigtablePingAndWarm(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPBigtableDeleteTable(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPBigtablePath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v2/projects/") {
		return false
	}
	if _, ok := parseGCPBigtableInstancesCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPBigtableInstancePath(path); ok {
		return true
	}
	if _, _, ok := parseGCPBigtableTablesCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPBigtableTablePath(path); ok {
		return true
	}
	_, _, _, _, ok := parseGCPBigtableTableActionPath(path)
	return ok
}

func handleGCPBigtableListInstances(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPBigtableInstancesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPBigtablePagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpBigtableInstance(project, "dev-instance"),
	}
	return respondGCPBigtableList(w, "instances", items, pageSize, start, path)
}

func handleGCPBigtableGetInstance(w http.ResponseWriter, path string) bool {
	project, instanceID, ok := parseGCPBigtableInstancePath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpBigtableInstance(project, instanceID))
	return true
}

func handleGCPBigtableListTables(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, ok := parseGCPBigtableTablesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPBigtablePagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpBigtableTable(project, instanceID, "orders"),
	}
	return respondGCPBigtableList(w, "tables", items, pageSize, start, path)
}

func handleGCPBigtableGetTable(w http.ResponseWriter, path string) bool {
	project, instanceID, tableID, ok := parseGCPBigtableTablePath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpBigtableTable(project, instanceID, tableID))
	return true
}

func handleGCPBigtableCreateTable(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, ok := parseGCPBigtableTablesCollectionPath(path)
	if !ok {
		return false
	}

	tableID := strings.TrimSpace(r.URL.Query().Get("tableId"))
	if tableID == "" {
		respondGCPBigtableInvalidArgument(w, path, "tableId is required")
		return true
	}

	body, valid := decodeGCPBigtableJSONBody(w, r, path)
	if !valid {
		return true
	}
	table, _ := body["table"].(map[string]any)
	if len(table) == 0 {
		table = body
	}
	if len(table) == 0 {
		respondGCPBigtableInvalidArgument(w, path, "table is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpBigtableOperation(
		fmt.Sprintf("operations/bigtable.createTable.%s.%s.%s", project, instanceID, tableID),
	))
	return true
}

func handleGCPBigtableDeleteTable(w http.ResponseWriter, path string) bool {
	project, instanceID, tableID, ok := parseGCPBigtableTablePath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpBigtableOperation(
		fmt.Sprintf("operations/bigtable.deleteTable.%s.%s.%s", project, instanceID, tableID),
	))
	return true
}

func handleGCPBigtableDropRowRange(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, tableID, action, ok := parseGCPBigtableTableActionPath(path)
	if !ok || action != "dropRowRange" {
		return false
	}

	body, valid := decodeGCPBigtableJSONBody(w, r, path)
	if !valid {
		return true
	}
	deleteAll, _ := body["deleteAllDataFromTable"].(bool)
	rowKeyPrefix, _ := body["rowKeyPrefix"].(string)
	if !deleteAll && strings.TrimSpace(rowKeyPrefix) == "" {
		respondGCPBigtableInvalidArgument(w, path, "dropRowRange requires deleteAllDataFromTable=true or rowKeyPrefix")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"table":   fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instanceID, tableID),
		"dropped": true,
	})
	return true
}

func handleGCPBigtableReadRows(w http.ResponseWriter, path string) bool {
	project, instanceID, tableID, action, ok := parseGCPBigtableTableActionPath(path)
	if !ok || action != "readRows" {
		return false
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"chunks": []any{
			map[string]any{
				"rowKey": "b3JkZXItMQ==",
				"familyName": map[string]any{
					"value": "cf1",
				},
				"qualifier": "c3RhdHVz",
				"value":     "Y3JlYXRlZA==",
				"commitRow": true,
			},
		},
		"table": fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instanceID, tableID),
	})
	return true
}

func handleGCPBigtableSampleRowKeys(w http.ResponseWriter, path string) bool {
	project, instanceID, tableID, action, ok := parseGCPBigtableTableActionPath(path)
	if !ok || action != "sampleRowKeys" {
		return false
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"sampleRowKeys": []any{
			map[string]any{
				"rowKey":      "b3JkZXItMQ==",
				"offsetBytes": "1024",
			},
		},
		"table": fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instanceID, tableID),
	})
	return true
}

func handleGCPBigtableMutateRow(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, tableID, action, ok := parseGCPBigtableTableActionPath(path)
	if !ok || action != "mutateRow" {
		return false
	}

	body, valid := decodeGCPBigtableJSONBody(w, r, path)
	if !valid {
		return true
	}
	mutations, _ := body["mutations"].([]any)
	if len(mutations) == 0 {
		respondGCPBigtableInvalidArgument(w, path, "mutations must contain at least one entry")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"table": fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instanceID, tableID),
		"status": map[string]any{
			"code":    0,
			"message": "OK",
		},
	})
	return true
}

func handleGCPBigtablePingAndWarm(w http.ResponseWriter, path string) bool {
	project, instanceID, tableID, action, ok := parseGCPBigtableTableActionPath(path)
	if !ok || action != "pingAndWarm" {
		return false
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"name":   fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instanceID, tableID),
		"warmed": true,
	})
	return true
}

func parseGCPBigtableInstancesCollectionPath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "instances" {
		return "", false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", false
	}
	return project, true
}

func parseGCPBigtableInstancePath(path string) (project, instanceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "instances" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	if project == "" || instanceID == "" {
		return "", "", false
	}
	return project, instanceID, true
}

func parseGCPBigtableTablesCollectionPath(path string) (project, instanceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "instances" || parts[6] != "tables" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	if project == "" || instanceID == "" {
		return "", "", false
	}
	return project, instanceID, true
}

func parseGCPBigtableTablePath(path string) (project, instanceID, tableID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "instances" || parts[6] != "tables" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	tableID = strings.TrimSpace(parts[7])
	if project == "" || instanceID == "" || tableID == "" {
		return "", "", "", false
	}
	return project, instanceID, tableID, true
}

func parseGCPBigtableTableActionPath(path string) (project, instanceID, tableID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "instances" || parts[6] != "tables" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	tableAndAction := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(parts[7], "%3A", ":"), "%3a", ":"))
	tableID, action, found := strings.Cut(tableAndAction, ":")
	if !found {
		return "", "", "", "", false
	}
	tableID = strings.TrimSpace(tableID)
	action = strings.TrimSpace(action)
	if project == "" || instanceID == "" || tableID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, instanceID, tableID, action, true
}

func parseGCPBigtablePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPBigtableInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPBigtableInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPBigtableList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPBigtableInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPBigtableJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPBigtableInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpBigtableInstance(project, instanceID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/instances/%s", project, instanceID),
		"displayName": "Dev Instance",
		"type":        "PRODUCTION",
		"state":       "READY",
	}
}

func gcpBigtableTable(project, instanceID, tableID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/instances/%s/tables/%s", project, instanceID, tableID),
		"columnFamilies": map[string]any{
			"cf1": map[string]any{
				"gcRule": map[string]any{
					"maxNumVersions": 1,
				},
			},
		},
		"granularity": "MILLIS",
	}
}

func gcpBigtableOperation(name string) map[string]any {
	return map[string]any{
		"name": name,
		"done": true,
	}
}

func respondGCPBigtableInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
