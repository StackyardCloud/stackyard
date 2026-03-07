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

const gcpVMMigrationGRPCPathPrefix = "/gcp/google.cloud.vmmigration.v1.VmMigration/"

var (
	gcpVMMigrationReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	gcpVMMigrationCollectionDefaults = map[string]string{
		"sources":              "source-1",
		"utilizationReports":   "report-1",
		"datacenterConnectors": "connector-1",
		"migratingVms":         "migrating-vm-1",
		"cloneJobs":            "clone-job-1",
		"cutoverJobs":          "cutover-job-1",
		"groups":               "group-1",
		"targetProjects":       "target-project-1",
		"replicationCycles":    "replication-cycle-1",
		"imageImports":         "image-import-1",
		"imageImportJobs":      "image-import-job-1",
		"diskMigrationJobs":    "disk-migration-job-1",
	}

	gcpVMMigrationCreateIDQueryParam = map[string]string{
		"sources":              "sourceId",
		"utilizationReports":   "utilizationReportId",
		"datacenterConnectors": "datacenterConnectorId",
		"migratingVms":         "migratingVmId",
		"cloneJobs":            "cloneJobId",
		"cutoverJobs":          "cutoverJobId",
		"groups":               "groupId",
		"targetProjects":       "targetProjectId",
		"imageImports":         "imageImportId",
		"diskMigrationJobs":    "diskMigrationJobId",
	}

	gcpVMMigrationCreateBodyKey = map[string]string{
		"sources":              "source",
		"utilizationReports":   "utilizationReport",
		"datacenterConnectors": "datacenterConnector",
		"migratingVms":         "migratingVm",
		"cloneJobs":            "cloneJob",
		"cutoverJobs":          "cutoverJob",
		"groups":               "group",
		"targetProjects":       "targetProject",
		"imageImports":         "imageImport",
		"diskMigrationJobs":    "diskMigrationJob",
	}

	gcpVMMigrationUpdateBodyKey = map[string]string{
		"sources":           "source",
		"migratingVms":      "migratingVm",
		"groups":            "group",
		"targetProjects":    "targetProject",
		"diskMigrationJobs": "diskMigrationJob",
	}
)

func (s *Server) handleGCPVMMigrationRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_vmmigration(w, r) {
		return true
	}

	path := normalizeGCPVMMigrationPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpVMMigrationGRPCPathPrefix) {
		return handleGCPVMMigrationGRPCBridge(w, r, path)
	}

	if isGCPVMMigrationLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVMMigrationListLocations(w, r, path) {
			return true
		}
		if handleGCPVMMigrationGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPVMMigrationRESTPath(path, hasGCPVMMigrationHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPVMMigrationListOperations(w, r, path) {
			return true
		}
		if handleGCPVMMigrationGetOperation(w, path) {
			return true
		}
		if handleGCPVMMigrationRESTGet(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPVMMigrationCancelOperation(w, r, path) {
			return true
		}
		if handleGCPVMMigrationRESTPost(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPVMMigrationRESTPatch(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPVMMigrationDeleteOperation(w, path) {
			return true
		}
		if handleGCPVMMigrationRESTDelete(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPVMMigrationPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVMMigrationHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "vmmigration",
		"vmmigration-apiv1",
		"vmmigration_apiv1",
		"vm-migration",
		"vm_migration",
		"cloud-vm-migration",
		"cloud_vm_migration",
		"gcp-vm-migration",
		"gcp-vmmigration":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-vmmigration-apiv1") || strings.Contains(ua, "cloud.google.com/go/vmmigration")
}

func isGCPVMMigrationLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVMMigrationHint(r) {
		return false
	}
	_, _, list, ok := parseGCPVMMigrationProjectLocationsPath(path)
	if !ok {
		return false
	}
	return list || strings.Count(strings.Trim(path, "/"), "/") == 5
}

func isGCPVMMigrationRESTPath(path string, includeHint bool) bool {
	if _, _, list, ok := parseGCPVMMigrationProjectLocationsPath(path); ok {
		return includeHint || list
	}
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok {
		return false
	}
	if project == "" || location == "" {
		return false
	}
	if len(tail) == 0 {
		return includeHint
	}
	if isGCPVMMigrationOperationsTail(tail) {
		return true
	}
	_, parsed := parseGCPVMMigrationResourceTail(tail)
	if parsed {
		return true
	}
	return includeHint
}

func handleGCPVMMigrationListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPVMMigrationProjectLocationsPath(path)
	if !ok || !list {
		return false
	}

	pageSize, start, ok := parseGCPVMMigrationPagination(w, r, path, 1000)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpVMMigrationLocation(project, "us-central1"),
		gcpVMMigrationLocation(project, "global"),
	}
	return respondGCPVMMigrationList(w, "locations", items, pageSize, start, path)
}

func handleGCPVMMigrationGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPVMMigrationProjectLocationsPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVMMigrationLocation(project, location))
	return true
}

func handleGCPVMMigrationListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok || !isGCPVMMigrationOperationsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPVMMigrationPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpVMMigrationOperation(project, location, "vmmigration-op-1", false),
		gcpVMMigrationOperation(project, location, "vmmigration-op-2", true),
	}
	return respondGCPVMMigrationList(w, "operations", items, pageSize, start, path)
}

func handleGCPVMMigrationGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok || !isGCPVMMigrationOperationResourceTail(tail) {
		return false
	}
	operationID := strings.TrimSpace(tail[1])
	if isGCPVMMigrationMissingID(operationID) {
		respondGCPVMMigrationNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, operationID, strings.Contains(strings.ToLower(operationID), "done")))
	return true
}

func handleGCPVMMigrationCancelOperation(w http.ResponseWriter, _ *http.Request, path string) bool {
	_, _, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok || !isGCPVMMigrationOperationActionTail(tail, "cancel") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVMMigrationDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok || !isGCPVMMigrationOperationResourceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVMMigrationRESTGet(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMMigrationResourceTail(tail)
	if !ok {
		return false
	}

	if info.action != "" {
		switch info.action {
		case "fetchInventory":
			respondJSON(w, http.StatusOK, map[string]any{
				"vmwareVms": map[string]any{
					"details": []map[string]any{
						{
							"vmId":        "vm-1001",
							"displayName": "source-vm-1001",
							"memoryMb":    4096,
							"powerState":  "POWER_ON",
						},
					},
				},
			})
			return true
		case "fetchStorageInventory":
			respondJSON(w, http.StatusOK, map[string]any{
				"resources":     []map[string]any{},
				"nextPageToken": "",
			})
			return true
		default:
			return false
		}
	}

	if info.isCollection {
		pageSize, start, valid := parseGCPVMMigrationPagination(w, r, path, 1000)
		if !valid {
			return true
		}
		id := gcpVMMigrationCollectionDefaults[info.collection]
		itemTail := append(append([]string{}, info.segments...), id)
		items := []map[string]any{
			gcpVMMigrationResourceFixture(project, location, info.collection, itemTail),
		}
		return respondGCPVMMigrationList(w, info.collection, items, pageSize, start, path)
	}

	if isGCPVMMigrationMissingID(info.id) {
		respondGCPVMMigrationNotFound(w, path, "resource not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMMigrationResourceFixture(project, location, info.collection, info.segments))
	return true
}

func handleGCPVMMigrationRESTPost(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMMigrationResourceTail(tail)
	if !ok {
		return false
	}

	body, valid := decodeGCPVMMigrationJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}

	if info.action != "" {
		targetID := info.id
		switch info.action {
		case "upgradeAppliance", "startMigration", "resumeMigration", "pauseMigration", "finalizeMigration", "extendMigration", "addGroupMigration", "removeGroupMigration", "run":
			if info.action == "pauseMigration" && strings.Contains(strings.ToLower(targetID), "paused") {
				respondGCPVMMigrationFailedPrecondition(w, path, "migration is already paused")
				return true
			}
			if info.action == "resumeMigration" && strings.Contains(strings.ToLower(targetID), "active") {
				respondGCPVMMigrationFailedPrecondition(w, path, "migration is already active")
				return true
			}
			respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, info.action+"."+targetID, false))
			return true
		case "cancel":
			if info.collection == "cloneJobs" || info.collection == "cutoverJobs" || info.collection == "imageImportJobs" || info.collection == "diskMigrationJobs" {
				if strings.Contains(strings.ToLower(targetID), "succeeded") || strings.Contains(strings.ToLower(targetID), "failed") || strings.Contains(strings.ToLower(targetID), "canceled") {
					respondGCPVMMigrationFailedPrecondition(w, path, "resource is in a terminal state")
					return true
				}
				respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "cancel."+targetID, false))
				return true
			}
			return false
		default:
			return false
		}
	}

	if !info.isCollection {
		return false
	}

	resourceID := strings.TrimSpace(r.URL.Query().Get(gcpVMMigrationCreateIDQueryParam[info.collection]))
	if resourceID == "" {
		resourceID = gcpVMMigrationCollectionDefaults[info.collection]
	}
	if isGCPVMMigrationMissingID(resourceID) || strings.Contains(strings.ToLower(resourceID), "existing") {
		respondGCPVMMigrationAlreadyExists(w, path, "resource already exists")
		return true
	}

	requiredBodyKey := gcpVMMigrationCreateBodyKey[info.collection]
	if requiredBodyKey != "" {
		if wrapped, exists := body[requiredBodyKey]; exists {
			if _, ok := wrapped.(map[string]any); !ok {
				respondGCPVMMigrationInvalidArgument(w, path, requiredBodyKey+" must be an object")
				return true
			}
		} else if info.collection == "sources" && len(body) == 0 {
			respondGCPVMMigrationInvalidArgument(w, path, requiredBodyKey+" is required")
			return true
		}
	}
	if info.collection == "targetProjects" && location != "global" {
		respondGCPVMMigrationInvalidArgument(w, path, "targetProjects must use locations/global")
		return true
	}

	respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "create"+strings.Title(info.collection)+"."+resourceID, false))
	return true
}

func handleGCPVMMigrationRESTPatch(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMMigrationResourceTail(tail)
	if !ok || info.isCollection || info.action != "" {
		return false
	}
	if _, updateable := gcpVMMigrationUpdateBodyKey[info.collection]; !updateable {
		return false
	}
	body, valid := decodeGCPVMMigrationJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = gcpVMMigrationUpdateMaskFromBody(body)
	}
	if mask == "" {
		respondGCPVMMigrationInvalidArgument(w, path, "updateMask is required")
		return true
	}
	bodyKey := gcpVMMigrationUpdateBodyKey[info.collection]
	resourceBody := gcpVMMigrationBodyResource(body, bodyKey)
	if len(resourceBody) == 0 {
		respondGCPVMMigrationInvalidArgument(w, path, bodyKey+" is required")
		return true
	}
	expectedName := gcpVMMigrationRESTResourceName(project, location, info.segments)
	if name := strings.TrimSpace(gcpVMMigrationString(resourceBody, "name")); name != "" && name != expectedName {
		respondGCPVMMigrationInvalidArgument(w, path, bodyKey+".name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "update."+info.collection+"."+info.id, false))
	return true
}

func handleGCPVMMigrationRESTDelete(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPVMMigrationLocationTail(path)
	if !ok {
		return false
	}
	info, ok := parseGCPVMMigrationResourceTail(tail)
	if !ok || info.isCollection || info.action != "" {
		return false
	}
	if isGCPVMMigrationMissingID(info.id) {
		respondGCPVMMigrationNotFound(w, path, "resource not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "delete."+info.collection+"."+info.id, false))
	return true
}

func handleGCPVMMigrationGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	method := strings.TrimPrefix(path, gcpVMMigrationGRPCPathPrefix)
	if method == "" || strings.Contains(method, "/") {
		return false
	}

	body, ok := decodeGCPVMMigrationJSONBodyOptional(w, r, path)
	if !ok {
		return true
	}

	if collection, found := gcpVMMigrationGRPCListMethodCollection(method); found {
		parent := strings.TrimSpace(gcpVMMigrationString(body, "parent"))
		project, location, _, parsed := parseGCPVMMigrationResourceName(parent)
		if !parsed {
			respondGCPVMMigrationInvalidArgument(w, path, "parent is required")
			return true
		}
		pageSize, start, valid := parseGCPVMMigrationBridgePagination(w, body, path)
		if !valid {
			return true
		}
		id := gcpVMMigrationCollectionDefaults[collection]
		var tail []string
		if parentTail, parsedParent := parseGCPVMMigrationParentTail(parent); parsedParent {
			tail = append(parentTail, id)
		}
		item := gcpVMMigrationResourceFixture(project, location, collection, tail)
		return respondGCPVMMigrationList(w, collection, []map[string]any{item}, pageSize, start, path)
	}

	if collection, found := gcpVMMigrationGRPCGetMethodCollection(method); found {
		name := strings.TrimSpace(gcpVMMigrationString(body, "name"))
		project, location, tail, parsed := parseGCPVMMigrationResourceName(name)
		if !parsed {
			respondGCPVMMigrationInvalidArgument(w, path, "name is required")
			return true
		}
		if !gcpVMMigrationTailHasCollection(tail, collection) {
			respondGCPVMMigrationInvalidArgument(w, path, "name is invalid")
			return true
		}
		resourceID := tail[len(tail)-1]
		if isGCPVMMigrationMissingID(resourceID) {
			respondGCPVMMigrationNotFound(w, path, "resource not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMMigrationResourceFixture(project, location, collection, tail))
		return true
	}

	if collection, idField, bodyField, found := gcpVMMigrationGRPCCreateMethodSpec(method); found {
		parent := strings.TrimSpace(gcpVMMigrationString(body, "parent"))
		project, location, _, parsed := parseGCPVMMigrationResourceName(parent)
		if !parsed {
			respondGCPVMMigrationInvalidArgument(w, path, "parent is required")
			return true
		}
		resourceID := strings.TrimSpace(gcpVMMigrationString(body, idField))
		if resourceID == "" {
			respondGCPVMMigrationInvalidArgument(w, path, idField+" is required")
			return true
		}
		if resourceBody, _ := body[bodyField].(map[string]any); len(resourceBody) == 0 {
			respondGCPVMMigrationInvalidArgument(w, path, bodyField+" is required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "create."+collection+"."+resourceID, false))
		return true
	}

	if collection, bodyField, found := gcpVMMigrationGRPCUpdateMethodSpec(method); found {
		resourceBody, _ := body[bodyField].(map[string]any)
		if len(resourceBody) == 0 {
			respondGCPVMMigrationInvalidArgument(w, path, bodyField+" is required")
			return true
		}
		name := strings.TrimSpace(gcpVMMigrationString(resourceBody, "name"))
		project, location, _, parsed := parseGCPVMMigrationResourceName(name)
		if !parsed {
			respondGCPVMMigrationInvalidArgument(w, path, bodyField+".name is required")
			return true
		}
		if gcpVMMigrationUpdateMaskFromBody(body) == "" {
			respondGCPVMMigrationInvalidArgument(w, path, "updateMask is required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "update."+collection, false))
		return true
	}

	if found := gcpVMMigrationGRPCDeleteMethod(method); found {
		name := strings.TrimSpace(gcpVMMigrationString(body, "name"))
		project, location, tail, parsed := parseGCPVMMigrationResourceName(name)
		if !parsed {
			respondGCPVMMigrationInvalidArgument(w, path, "name is required")
			return true
		}
		id := tail[len(tail)-1]
		respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, "delete."+id, false))
		return true
	}

	if action, nameField, found := gcpVMMigrationGRPCActionSpec(method); found {
		name := strings.TrimSpace(gcpVMMigrationString(body, nameField))
		project, location, tail, parsed := parseGCPVMMigrationResourceName(name)
		if !parsed {
			respondGCPVMMigrationInvalidArgument(w, path, nameField+" is required")
			return true
		}
		id := tail[len(tail)-1]
		if action == "pauseMigration" && strings.Contains(strings.ToLower(id), "paused") {
			respondGCPVMMigrationFailedPrecondition(w, path, "migration is already paused")
			return true
		}
		if action == "resumeMigration" && strings.Contains(strings.ToLower(id), "active") {
			respondGCPVMMigrationFailedPrecondition(w, path, "migration is already active")
			return true
		}
		if action == "fetchInventory" {
			respondJSON(w, http.StatusOK, map[string]any{
				"vmwareVms": map[string]any{
					"details": []map[string]any{
						{"vmId": "vm-1001", "displayName": "source-vm-1001", "powerState": "POWER_ON"},
					},
				},
			})
			return true
		}
		if action == "fetchStorageInventory" {
			respondJSON(w, http.StatusOK, map[string]any{
				"resources":     []map[string]any{},
				"nextPageToken": "",
			})
			return true
		}
		respondJSON(w, http.StatusOK, gcpVMMigrationOperation(project, location, action+"."+id, false))
		return true
	}

	respondProviderNotImplemented(w, providerGCP, path)
	return true
}

func gcpVMMigrationGRPCListMethodCollection(method string) (string, bool) {
	methods := map[string]string{
		"ListSources":              "sources",
		"ListUtilizationReports":   "utilizationReports",
		"ListDatacenterConnectors": "datacenterConnectors",
		"ListMigratingVms":         "migratingVms",
		"ListCloneJobs":            "cloneJobs",
		"ListCutoverJobs":          "cutoverJobs",
		"ListGroups":               "groups",
		"ListTargetProjects":       "targetProjects",
		"ListReplicationCycles":    "replicationCycles",
		"ListImageImports":         "imageImports",
		"ListImageImportJobs":      "imageImportJobs",
		"ListDiskMigrationJobs":    "diskMigrationJobs",
	}
	collection, ok := methods[method]
	return collection, ok
}

func gcpVMMigrationGRPCGetMethodCollection(method string) (string, bool) {
	methods := map[string]string{
		"GetSource":              "sources",
		"GetUtilizationReport":   "utilizationReports",
		"GetDatacenterConnector": "datacenterConnectors",
		"GetMigratingVm":         "migratingVms",
		"GetCloneJob":            "cloneJobs",
		"GetCutoverJob":          "cutoverJobs",
		"GetGroup":               "groups",
		"GetTargetProject":       "targetProjects",
		"GetReplicationCycle":    "replicationCycles",
		"GetImageImport":         "imageImports",
		"GetImageImportJob":      "imageImportJobs",
		"GetDiskMigrationJob":    "diskMigrationJobs",
	}
	collection, ok := methods[method]
	return collection, ok
}

func gcpVMMigrationGRPCCreateMethodSpec(method string) (collection, idField, bodyField string, ok bool) {
	type createSpec struct {
		collection string
		idField    string
		bodyField  string
	}
	specs := map[string]createSpec{
		"CreateSource":              {collection: "sources", idField: "sourceId", bodyField: "source"},
		"CreateUtilizationReport":   {collection: "utilizationReports", idField: "utilizationReportId", bodyField: "utilizationReport"},
		"CreateDatacenterConnector": {collection: "datacenterConnectors", idField: "datacenterConnectorId", bodyField: "datacenterConnector"},
		"CreateMigratingVm":         {collection: "migratingVms", idField: "migratingVmId", bodyField: "migratingVm"},
		"CreateCloneJob":            {collection: "cloneJobs", idField: "cloneJobId", bodyField: "cloneJob"},
		"CreateCutoverJob":          {collection: "cutoverJobs", idField: "cutoverJobId", bodyField: "cutoverJob"},
		"CreateGroup":               {collection: "groups", idField: "groupId", bodyField: "group"},
		"CreateTargetProject":       {collection: "targetProjects", idField: "targetProjectId", bodyField: "targetProject"},
		"CreateImageImport":         {collection: "imageImports", idField: "imageImportId", bodyField: "imageImport"},
		"CreateDiskMigrationJob":    {collection: "diskMigrationJobs", idField: "diskMigrationJobId", bodyField: "diskMigrationJob"},
	}
	spec, found := specs[method]
	if !found {
		return "", "", "", false
	}
	return spec.collection, spec.idField, spec.bodyField, true
}

func gcpVMMigrationGRPCUpdateMethodSpec(method string) (collection, bodyField string, ok bool) {
	specs := map[string]struct {
		collection string
		bodyField  string
	}{
		"UpdateSource":           {collection: "sources", bodyField: "source"},
		"UpdateMigratingVm":      {collection: "migratingVms", bodyField: "migratingVm"},
		"UpdateGroup":            {collection: "groups", bodyField: "group"},
		"UpdateTargetProject":    {collection: "targetProjects", bodyField: "targetProject"},
		"UpdateDiskMigrationJob": {collection: "diskMigrationJobs", bodyField: "diskMigrationJob"},
	}
	spec, found := specs[method]
	if !found {
		return "", "", false
	}
	return spec.collection, spec.bodyField, true
}

func gcpVMMigrationGRPCDeleteMethod(method string) bool {
	switch method {
	case "DeleteSource", "DeleteUtilizationReport", "DeleteDatacenterConnector", "DeleteMigratingVm", "DeleteGroup", "DeleteTargetProject", "DeleteImageImport", "DeleteDiskMigrationJob":
		return true
	default:
		return false
	}
}

func gcpVMMigrationGRPCActionSpec(method string) (action, nameField string, ok bool) {
	specs := map[string]struct {
		action    string
		nameField string
	}{
		"FetchInventory":         {action: "fetchInventory", nameField: "source"},
		"FetchStorageInventory":  {action: "fetchStorageInventory", nameField: "source"},
		"UpgradeAppliance":       {action: "upgradeAppliance", nameField: "datacenterConnector"},
		"StartMigration":         {action: "startMigration", nameField: "migratingVm"},
		"ResumeMigration":        {action: "resumeMigration", nameField: "migratingVm"},
		"PauseMigration":         {action: "pauseMigration", nameField: "migratingVm"},
		"FinalizeMigration":      {action: "finalizeMigration", nameField: "migratingVm"},
		"ExtendMigration":        {action: "extendMigration", nameField: "migratingVm"},
		"CancelCloneJob":         {action: "cancel", nameField: "name"},
		"CancelCutoverJob":       {action: "cancel", nameField: "name"},
		"AddGroupMigration":      {action: "addGroupMigration", nameField: "group"},
		"RemoveGroupMigration":   {action: "removeGroupMigration", nameField: "group"},
		"CancelImageImportJob":   {action: "cancel", nameField: "name"},
		"RunDiskMigrationJob":    {action: "run", nameField: "name"},
		"CancelDiskMigrationJob": {action: "cancel", nameField: "name"},
	}
	spec, found := specs[method]
	if !found {
		return "", "", false
	}
	return spec.action, spec.nameField, true
}

func parseGCPVMMigrationBridgePagination(w http.ResponseWriter, body map[string]any, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw, exists := body["pageSize"]; exists {
		value, valid := gcpVMMigrationNumberToInt(raw)
		if !valid || value < 0 || value > 1000 {
			respondGCPVMMigrationInvalidArgument(w, path, "pageSize must be a non-negative integer <= 1000")
			return 0, 0, false
		}
		pageSize = value
	}
	if raw, exists := body["pageToken"]; exists {
		token, ok := raw.(string)
		if !ok {
			respondGCPVMMigrationInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		if strings.TrimSpace(token) != "" {
			value, err := strconv.Atoi(strings.TrimSpace(token))
			if err != nil || value < 0 {
				respondGCPVMMigrationInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = value
		}
	}
	return pageSize, start, true
}

func parseGCPVMMigrationProjectLocationsPath(path string) (project, location string, list bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		if project == "" {
			return "", "", false, false
		}
		return project, "", true, true
	}
	if len(parts) == 6 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		location = strings.TrimSpace(parts[5])
		if project == "" || location == "" {
			return "", "", false, false
		}
		return project, location, false, true
	}
	return "", "", false, false
}

func parseGCPVMMigrationLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func parseGCPVMMigrationResourceName(name string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[4:], true
}

func parseGCPVMMigrationParentTail(name string) (tail []string, ok bool) {
	_, _, tail, ok = parseGCPVMMigrationResourceName(name)
	return tail, ok
}

type gcpVMMigrationTailInfo struct {
	segments     []string
	collection   string
	id           string
	action       string
	isCollection bool
}

func parseGCPVMMigrationResourceTail(tail []string) (gcpVMMigrationTailInfo, bool) {
	if len(tail) == 0 {
		return gcpVMMigrationTailInfo{}, false
	}
	segments := append([]string{}, tail...)
	last := strings.TrimSpace(segments[len(segments)-1])
	if last == "" {
		return gcpVMMigrationTailInfo{}, false
	}
	action := ""
	if resource, act, found := strings.Cut(last, ":"); found {
		if strings.TrimSpace(resource) == "" || strings.TrimSpace(act) == "" {
			return gcpVMMigrationTailInfo{}, false
		}
		segments[len(segments)-1] = strings.TrimSpace(resource)
		action = strings.TrimSpace(act)
	}
	for _, part := range segments {
		if strings.TrimSpace(part) == "" {
			return gcpVMMigrationTailInfo{}, false
		}
	}

	isCollection := len(segments)%2 == 1
	collection := ""
	resourceID := ""
	if isCollection {
		collection = segments[len(segments)-1]
	} else {
		collection = segments[len(segments)-2]
		resourceID = segments[len(segments)-1]
	}
	if _, ok := gcpVMMigrationCollectionDefaults[collection]; !ok {
		return gcpVMMigrationTailInfo{}, false
	}
	return gcpVMMigrationTailInfo{
		segments:     segments,
		collection:   collection,
		id:           resourceID,
		action:       action,
		isCollection: isCollection,
	}, true
}

func gcpVMMigrationTailHasCollection(tail []string, collection string) bool {
	if len(tail) == 0 {
		return false
	}
	if len(tail)%2 == 1 {
		return tail[len(tail)-1] == collection
	}
	return tail[len(tail)-2] == collection
}

func isGCPVMMigrationOperationsTail(tail []string) bool {
	return isGCPVMMigrationOperationsCollectionTail(tail) || isGCPVMMigrationOperationResourceTail(tail) || isGCPVMMigrationOperationActionTail(tail, "cancel")
}

func isGCPVMMigrationOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPVMMigrationOperationResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPVMMigrationOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	id, parsedAction, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return found && strings.TrimSpace(id) != "" && parsedAction == action
}

func parseGCPVMMigrationPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVMMigrationInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPVMMigrationOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVMMigrationInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}

	return pageSize, start, true
}

func respondGCPVMMigrationList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPVMMigrationOutOfRange(w, path, "pageToken is out of range")
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

func decodeGCPVMMigrationJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPVMMigrationInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPVMMigrationInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPVMMigrationJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPVMMigrationJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPVMMigrationInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpVMMigrationBodyResource(body map[string]any, key string) map[string]any {
	if strings.TrimSpace(key) == "" {
		return body
	}
	if nested, ok := body[key].(map[string]any); ok {
		return nested
	}
	// REST clients typically send the resource object as the top-level body for
	// create/update calls, while bridge-style tests often wrap it by field name.
	// Accept both forms.
	return body
}

func gcpVMMigrationString(body map[string]any, key string) string {
	raw, ok := body[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return strings.TrimSpace(value)
}

func gcpVMMigrationUpdateMaskFromBody(body map[string]any) string {
	if value := strings.TrimSpace(gcpVMMigrationString(body, "updateMask")); value != "" {
		return value
	}
	mask, _ := body["updateMask"].(map[string]any)
	if len(mask) == 0 {
		return ""
	}
	if paths, ok := mask["paths"].([]any); ok && len(paths) > 0 {
		return "paths"
	}
	return ""
}

func gcpVMMigrationNumberToInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v < 0 || v != float64(int(v)) {
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

func gcpVMMigrationRESTResourceName(project, location string, tail []string) string {
	return fmt.Sprintf("projects/%s/locations/%s/%s", project, location, strings.Join(tail, "/"))
}

func gcpVMMigrationResourceFixture(project, location, collection string, tail []string) map[string]any {
	name := gcpVMMigrationRESTResourceName(project, location, tail)
	switch collection {
	case "sources":
		return map[string]any{
			"name":        name,
			"description": "Stackyard VM Migration source",
			"labels": map[string]any{
				"env": "staged",
			},
			"createTime": gcpVMMigrationReferenceTime.Format(time.RFC3339),
			"updateTime": gcpVMMigrationReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
			"vmware": map[string]any{
				"vcenterIp": "10.0.0.20",
			},
		}
	case "utilizationReports":
		return map[string]any{
			"name":        name,
			"state":       "UtilizationReport_STATE_READY",
			"displayName": "Stackyard Utilization Report",
			"createTime":  gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "datacenterConnectors":
		return map[string]any{
			"name":                           name,
			"state":                          "DatacenterConnector_STATE_ACTIVE",
			"applianceInfrastructureVersion": "1.0.0",
			"createTime":                     gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "migratingVms":
		return map[string]any{
			"name":        name,
			"displayName": "Stackyard Migrating VM",
			"description": "Deterministic migrating VM fixture",
			"sourceVmId":  "vm-1001",
			"state":       "MigratingVm_STATE_ACTIVE",
			"createTime":  gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "cloneJobs":
		return map[string]any{
			"name":       name,
			"state":      "CloneJob_STATE_PENDING",
			"createTime": gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "cutoverJobs":
		return map[string]any{
			"name":       name,
			"state":      "CutoverJob_STATE_PENDING",
			"createTime": gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "groups":
		return map[string]any{
			"name":                name,
			"description":         "Stackyard migration group",
			"migrationTargetType": "Group_MigrationTargetType_GROUP_MIGRATION_TARGET_TYPE_PROJECT",
		}
	case "targetProjects":
		return map[string]any{
			"name":        name,
			"project":     "target-stackyard-project",
			"description": "Stackyard target project",
			"createTime":  gcpVMMigrationReferenceTime.Format(time.RFC3339),
			"updateTime":  gcpVMMigrationReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		}
	case "replicationCycles":
		return map[string]any{
			"name":      name,
			"state":     "ReplicationCycle_State_RUNNING",
			"startTime": gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "imageImports":
		return map[string]any{
			"name":        name,
			"displayName": "Stackyard Image Import",
			"state":       "ImageImport_State_CREATING",
			"createTime":  gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "imageImportJobs":
		return map[string]any{
			"name":       name,
			"state":      "ImageImportJob_State_PENDING",
			"createTime": gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	case "diskMigrationJobs":
		return map[string]any{
			"name":        name,
			"displayName": "Stackyard Disk Migration Job",
			"state":       "DiskMigrationJob_State_PENDING",
			"createTime":  gcpVMMigrationReferenceTime.Format(time.RFC3339),
		}
	default:
		return map[string]any{
			"name": name,
		}
	}
}

func gcpVMMigrationOperation(project, location, operationID string, done bool) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": done,
		"metadata": map[string]any{
			"@type":         "type.googleapis.com/google.cloud.vmmigration.v1.OperationMetadata",
			"createTime":    gcpVMMigrationReferenceTime.Format(time.RFC3339),
			"endTime":       gcpVMMigrationReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
			"target":        "vmmigration",
			"verb":          "execute",
			"statusMessage": "Stackyard staged VM Migration operation",
		},
	}
}

func gcpVMMigrationLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"metadata": map[string]any{
			"provider": providerGCP,
			"service":  "vmmigration",
		},
	}
}

func isGCPVMMigrationMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "missing") || strings.Contains(id, "notfound")
}

func respondGCPVMMigrationInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPVMMigrationError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPVMMigrationNotFound(w http.ResponseWriter, path, message string) {
	respondGCPVMMigrationError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPVMMigrationAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPVMMigrationError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPVMMigrationFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPVMMigrationError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPVMMigrationOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPVMMigrationError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPVMMigrationError(w http.ResponseWriter, status int, errType, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errType,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_vmmigration(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "vmmigration") {
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
			"name":     "projects/stackyard/locations/us-central1/sources/source-1",
			"service":  "vmmigration",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
