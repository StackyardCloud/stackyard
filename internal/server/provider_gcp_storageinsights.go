package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gcpStorageInsightsReferenceTime    = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpStorageInsightsRequestIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	gcpStorageInsightsDatasetIDPattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,63}$`)
)

func (s *Server) handleGCPStorageInsightsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_storageinsights(w, r) {
		return true
	}

	path := normalizeGCPStorageInsightsPath(rawRequestPath(r))
	if isGCPStorageInsightsLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPStorageInsightsListLocations(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPStorageInsightsPath(path, hasGCPStorageInsightsHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPStorageInsightsListReportConfigs(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsGetReportConfig(w, path) {
			return true
		}
		if handleGCPStorageInsightsListReportDetails(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsGetReportDetail(w, path) {
			return true
		}
		if handleGCPStorageInsightsListDatasetConfigs(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsGetDatasetConfig(w, path) {
			return true
		}
		if handleGCPStorageInsightsListOperations(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPStorageInsightsCreateReportConfig(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsCreateDatasetConfig(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsLinkDataset(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsUnlinkDataset(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPStorageInsightsUpdateReportConfig(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsUpdateDatasetConfig(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPStorageInsightsDeleteReportConfig(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsDeleteDatasetConfig(w, r, path) {
			return true
		}
		if handleGCPStorageInsightsDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPStorageInsightsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPStorageInsightsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "storageinsights",
		"storageinsights-apiv1",
		"storageinsights_apiv1",
		"storage-insights",
		"storage_insights",
		"cloud-storage-insights",
		"cloud_storage_insights",
		"cloudstorageinsights",
		"gcp-storage-insights":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-storageinsights-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/storageinsights/apiv1")
}

func isGCPStorageInsightsLocationRequest(r *http.Request, path string) bool {
	if !hasGCPStorageInsightsHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPStorageInsightsPath(path string, includeAmbiguous bool) bool {
	if !includeAmbiguous {
		return false
	}
	_, _, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok {
		return false
	}
	return isGCPStorageInsightsReportConfigsCollectionTail(tail) ||
		isGCPStorageInsightsReportConfigTail(tail) ||
		isGCPStorageInsightsReportDetailsCollectionTail(tail) ||
		isGCPStorageInsightsReportDetailTail(tail) ||
		isGCPStorageInsightsDatasetConfigsCollectionTail(tail) ||
		isGCPStorageInsightsDatasetConfigTail(tail) ||
		isGCPStorageInsightsDatasetConfigActionTail(tail, "linkDataset") ||
		isGCPStorageInsightsDatasetConfigActionTail(tail, "unlinkDataset") ||
		isGCPStorageInsightsOperationsCollectionTail(tail) ||
		isGCPStorageInsightsOperationTail(tail) ||
		isGCPStorageInsightsOperationActionTail(tail, "cancel")
}

func handleGCPStorageInsightsListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPStorageInsightsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpStorageInsightsLocation(project, "us-central1"),
		gcpStorageInsightsLocation(project, "global"),
	}
	return respondGCPStorageInsightsList(w, "locations", items, pageSize, start, path, false)
}

func handleGCPStorageInsightsGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsLocation(project, location))
	return true
}

func handleGCPStorageInsightsListReportConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsReportConfigsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPStorageInsightsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	orderBy := normalizeGCPStorageInsightsOrderBy(r.URL.Query().Get("orderBy"))
	if !isGCPStorageInsightsOrderBy(orderBy) {
		respondGCPStorageInsightsInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time asc, create_time desc")
		return true
	}
	if !isGCPStorageInsightsListFilter(r.URL.Query().Get("filter")) {
		respondGCPStorageInsightsInvalidArgument(w, path, "filter is unsupported in staged emulation")
		return true
	}

	items := []map[string]any{
		gcpStorageInsightsReportConfig(project, location, "reportconfig1"),
		gcpStorageInsightsReportConfig(project, location, "reportconfig2"),
	}
	sortGCPStorageInsightsItems(items, orderBy)
	return respondGCPStorageInsightsList(w, "reportConfigs", items, pageSize, start, path, true)
}

func handleGCPStorageInsightsGetReportConfig(w http.ResponseWriter, path string) bool {
	project, location, reportConfigID, ok := parseGCPStorageInsightsReportConfigPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(reportConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "report config not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsReportConfig(project, location, reportConfigID))
	return true
}

func handleGCPStorageInsightsCreateReportConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsReportConfigsCollectionTail(tail) {
		return false
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}

	body, valid := decodeGCPStorageInsightsJSONBody(w, r, path)
	if !valid {
		return true
	}
	reportConfig := gcpStorageInsightsBodyMap(body, "reportConfig")
	if len(reportConfig) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig is required")
		return true
	}
	if !validateGCPStorageInsightsReportConfig(w, path, reportConfig, false) {
		return true
	}

	reportConfigID := "reportconfig1"
	if providedName := gcpStorageInsightsString(reportConfig, "name"); providedName != "" {
		p, l, id, parsed := parseGCPStorageInsightsReportConfigName(providedName)
		if !parsed || p != project || l != location {
			respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.name must match parent")
			return true
		}
		reportConfigID = id
	}
	if strings.Contains(strings.ToLower(reportConfigID), "existing") {
		respondGCPStorageInsightsAlreadyExists(w, path, "report config already exists")
		return true
	}

	resp := gcpStorageInsightsReportConfig(project, location, reportConfigID)
	applyGCPStorageInsightsReportConfigOverrides(resp, reportConfig)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPStorageInsightsUpdateReportConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, reportConfigID, ok := parseGCPStorageInsightsReportConfigPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(reportConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "report config not found")
		return true
	}

	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	if updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask")); updateMask != "" {
		if !validateGCPStorageInsightsUpdateMask(updateMask, []string{
			"display_name", "displayName", "labels", "frequency_options", "frequencyOptions", "csv_options", "csvOptions", "parquet_options", "parquetOptions", "object_metadata_report_options", "objectMetadataReportOptions",
		}) {
			respondGCPStorageInsightsInvalidArgument(w, path, "updateMask contains unsupported fields")
			return true
		}
	}

	body, valid := decodeGCPStorageInsightsJSONBody(w, r, path)
	if !valid {
		return true
	}
	reportConfig := gcpStorageInsightsBodyMap(body, "reportConfig")
	if len(reportConfig) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig is required")
		return true
	}
	if !validateGCPStorageInsightsReportConfig(w, path, reportConfig, true) {
		return true
	}

	expectedName := gcpStorageInsightsReportConfigName(project, location, reportConfigID)
	if providedName := gcpStorageInsightsString(reportConfig, "name"); strings.TrimSpace(providedName) != expectedName {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.name must match requested resource")
		return true
	}

	resp := gcpStorageInsightsReportConfig(project, location, reportConfigID)
	applyGCPStorageInsightsReportConfigOverrides(resp, reportConfig)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPStorageInsightsDeleteReportConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, reportConfigID, ok := parseGCPStorageInsightsReportConfigPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(reportConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "report config not found")
		return true
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("force")); raw != "" {
		if _, err := strconv.ParseBool(raw); err != nil {
			respondGCPStorageInsightsInvalidArgument(w, path, "force must be a boolean")
			return true
		}
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageInsightsListReportDetails(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, reportConfigID, tail, ok := parseGCPStorageInsightsReportDetailsCollectionPath(path)
	if !ok || !isGCPStorageInsightsReportDetailsCollectionTail(tail) {
		return false
	}
	if isGCPStorageInsightsMissingID(reportConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "report config not found")
		return true
	}
	pageSize, start, valid := parseGCPStorageInsightsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	orderBy := normalizeGCPStorageInsightsOrderBy(r.URL.Query().Get("orderBy"))
	if !isGCPStorageInsightsOrderBy(orderBy) {
		respondGCPStorageInsightsInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time asc, create_time desc")
		return true
	}
	if !isGCPStorageInsightsListFilter(r.URL.Query().Get("filter")) {
		respondGCPStorageInsightsInvalidArgument(w, path, "filter is unsupported in staged emulation")
		return true
	}
	items := []map[string]any{
		gcpStorageInsightsReportDetail(project, location, reportConfigID, "reportdetail1"),
		gcpStorageInsightsReportDetail(project, location, reportConfigID, "reportdetail2"),
	}
	sortGCPStorageInsightsItems(items, orderBy)
	return respondGCPStorageInsightsList(w, "reportDetails", items, pageSize, start, path, true)
}

func handleGCPStorageInsightsGetReportDetail(w http.ResponseWriter, path string) bool {
	project, location, reportConfigID, reportDetailID, ok := parseGCPStorageInsightsReportDetailPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(reportConfigID) || isGCPStorageInsightsMissingID(reportDetailID) {
		respondGCPStorageInsightsNotFound(w, path, "report detail not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsReportDetail(project, location, reportConfigID, reportDetailID))
	return true
}

func handleGCPStorageInsightsListDatasetConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsDatasetConfigsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPStorageInsightsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	orderBy := normalizeGCPStorageInsightsOrderBy(r.URL.Query().Get("orderBy"))
	if !isGCPStorageInsightsOrderBy(orderBy) {
		respondGCPStorageInsightsInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time asc, create_time desc")
		return true
	}
	if !isGCPStorageInsightsListFilter(r.URL.Query().Get("filter")) {
		respondGCPStorageInsightsInvalidArgument(w, path, "filter is unsupported in staged emulation")
		return true
	}
	items := []map[string]any{
		gcpStorageInsightsDatasetConfig(project, location, "datasetconfig1"),
		gcpStorageInsightsDatasetConfig(project, location, "datasetconfig2"),
	}
	sortGCPStorageInsightsItems(items, orderBy)
	return respondGCPStorageInsightsList(w, "datasetConfigs", items, pageSize, start, path, true)
}

func handleGCPStorageInsightsGetDatasetConfig(w http.ResponseWriter, path string) bool {
	project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(datasetConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "dataset config not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsDatasetConfig(project, location, datasetConfigID))
	return true
}

func handleGCPStorageInsightsCreateDatasetConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsDatasetConfigsCollectionTail(tail) {
		return false
	}
	datasetConfigID := strings.TrimSpace(r.URL.Query().Get("datasetConfigId"))
	if datasetConfigID == "" {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfigId is required")
		return true
	}
	if !isGCPStorageInsightsDatasetConfigID(datasetConfigID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfigId is invalid and must not contain hyphens")
		return true
	}
	if strings.Contains(strings.ToLower(datasetConfigID), "existing") {
		respondGCPStorageInsightsAlreadyExists(w, path, "dataset config already exists")
		return true
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}

	body, valid := decodeGCPStorageInsightsJSONBody(w, r, path)
	if !valid {
		return true
	}
	datasetConfig := gcpStorageInsightsBodyMap(body, "datasetConfig")
	if len(datasetConfig) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig is required")
		return true
	}
	if !validateGCPStorageInsightsDatasetConfig(w, path, datasetConfig, false) {
		return true
	}
	if providedName := gcpStorageInsightsString(datasetConfig, "name"); providedName != "" {
		p, l, id, parsed := parseGCPStorageInsightsDatasetConfigName(providedName)
		if !parsed || p != project || l != location || id != datasetConfigID {
			respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig.name must match parent and datasetConfigId")
			return true
		}
	}

	created := gcpStorageInsightsDatasetConfig(project, location, datasetConfigID)
	applyGCPStorageInsightsDatasetConfigOverrides(created, datasetConfig)
	respondJSON(w, http.StatusOK, gcpStorageInsightsOperationForAction(project, location, "createDatasetConfig."+datasetConfigID, created, false))
	return true
}

func handleGCPStorageInsightsUpdateDatasetConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(datasetConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "dataset config not found")
		return true
	}

	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPStorageInsightsInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if !validateGCPStorageInsightsUpdateMask(updateMask, []string{
		"description", "labels", "retention_period_days", "retentionPeriodDays", "include_newly_created_buckets", "includeNewlyCreatedBuckets", "identity.type", "identity_type",
	}) {
		respondGCPStorageInsightsInvalidArgument(w, path, "updateMask contains unsupported fields")
		return true
	}
	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}

	body, valid := decodeGCPStorageInsightsJSONBody(w, r, path)
	if !valid {
		return true
	}
	datasetConfig := gcpStorageInsightsBodyMap(body, "datasetConfig")
	if len(datasetConfig) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig is required")
		return true
	}
	if !validateGCPStorageInsightsDatasetConfig(w, path, datasetConfig, true) {
		return true
	}
	expectedName := gcpStorageInsightsDatasetConfigName(project, location, datasetConfigID)
	if providedName := gcpStorageInsightsString(datasetConfig, "name"); strings.TrimSpace(providedName) != expectedName {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig.name must match requested resource")
		return true
	}

	updated := gcpStorageInsightsDatasetConfig(project, location, datasetConfigID)
	applyGCPStorageInsightsDatasetConfigOverrides(updated, datasetConfig)
	respondJSON(w, http.StatusOK, gcpStorageInsightsOperationForAction(project, location, "updateDatasetConfig."+datasetConfigID, updated, false))
	return true
}

func handleGCPStorageInsightsDeleteDatasetConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(datasetConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "dataset config not found")
		return true
	}

	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
		respondGCPStorageInsightsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsOperationForAction(project, location, "deleteDatasetConfig."+datasetConfigID, nil, false))
	return true
}

func handleGCPStorageInsightsLinkDataset(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, datasetConfigID, action, ok := parseGCPStorageInsightsDatasetConfigActionPath(path)
	if !ok || action != "linkDataset" {
		return false
	}
	if isGCPStorageInsightsMissingID(datasetConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "dataset config not found")
		return true
	}
	body, valid := decodeGCPStorageInsightsOptionalJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpStorageInsightsDatasetConfigName(project, location, datasetConfigID)
	if name := gcpStorageInsightsString(body, "name"); name != "" && strings.TrimSpace(name) != expectedName {
		respondGCPStorageInsightsInvalidArgument(w, path, "name must match requested dataset config")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsOperationForAction(project, location, "linkDataset."+datasetConfigID, nil, false))
	return true
}

func handleGCPStorageInsightsUnlinkDataset(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, datasetConfigID, action, ok := parseGCPStorageInsightsDatasetConfigActionPath(path)
	if !ok || action != "unlinkDataset" {
		return false
	}
	if isGCPStorageInsightsMissingID(datasetConfigID) {
		respondGCPStorageInsightsNotFound(w, path, "dataset config not found")
		return true
	}
	body, valid := decodeGCPStorageInsightsOptionalJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpStorageInsightsDatasetConfigName(project, location, datasetConfigID)
	if name := gcpStorageInsightsString(body, "name"); name != "" && strings.TrimSpace(name) != expectedName {
		respondGCPStorageInsightsInvalidArgument(w, path, "name must match requested dataset config")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsOperationForAction(project, location, "unlinkDataset."+datasetConfigID, nil, false))
	return true
}

func handleGCPStorageInsightsListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPStorageInsightsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	doneSet, doneValue, filterValid := parseGCPStorageInsightsDoneFilter(r.URL.Query().Get("filter"))
	if !filterValid {
		respondGCPStorageInsightsInvalidArgument(w, path, "filter must be done=true or done=false")
		return true
	}

	items := []map[string]any{
		gcpStorageInsightsOperationForAction(project, location, "createDatasetConfig.datasetconfig1", nil, false),
		gcpStorageInsightsOperationForAction(project, location, "updateDatasetConfig.datasetconfig1", nil, false),
		gcpStorageInsightsOperationForAction(project, location, "deleteDatasetConfig.datasetconfig1", nil, false),
		gcpStorageInsightsOperationForAction(project, location, "linkDataset.datasetconfig1", nil, false),
		gcpStorageInsightsOperationForAction(project, location, "unlinkDataset.datasetconfig1", nil, false),
		gcpStorageInsightsOperationForAction(project, location, "linkDatasetRunning.datasetconfig1", nil, false),
	}
	if doneSet {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			done, _ := item["done"].(bool)
			if done == doneValue {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sort.Slice(items, func(i, j int) bool {
		return gcpStorageInsightsString(items[i], "name") < gcpStorageInsightsString(items[j], "name")
	})
	return respondGCPStorageInsightsList(w, "operations", items, pageSize, start, path, false)
}

func handleGCPStorageInsightsGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPStorageInsightsOperationPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(operationID) {
		respondGCPStorageInsightsNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageInsightsOperationForAction(project, location, operationID, nil, false))
	return true
}

func handleGCPStorageInsightsCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, operationID, action, ok := parseGCPStorageInsightsOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	if isGCPStorageInsightsMissingID(operationID) {
		respondGCPStorageInsightsNotFound(w, path, "operation not found")
		return true
	}
	body, valid := decodeGCPStorageInsightsOptionalJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpStorageInsightsOperationName(project, location, operationID)
	if name := gcpStorageInsightsString(body, "name"); name != "" && strings.TrimSpace(name) != expectedName {
		respondGCPStorageInsightsInvalidArgument(w, path, "name must match requested operation")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageInsightsDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, ok := parseGCPStorageInsightsOperationPath(path)
	if !ok {
		return false
	}
	if isGCPStorageInsightsMissingID(operationID) {
		respondGCPStorageInsightsNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPStorageInsightsLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPStorageInsightsReportConfigsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "reportConfigs"
}

func isGCPStorageInsightsReportConfigTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "reportConfigs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPStorageInsightsReportDetailsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "reportConfigs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":") && tail[2] == "reportDetails"
}

func isGCPStorageInsightsReportDetailTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "reportConfigs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":") && tail[2] == "reportDetails" && strings.TrimSpace(tail[3]) != "" && !strings.Contains(tail[3], ":")
}

func isGCPStorageInsightsDatasetConfigsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "datasetConfigs"
}

func isGCPStorageInsightsDatasetConfigTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "datasetConfigs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPStorageInsightsDatasetConfigActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "datasetConfigs" {
		return false
	}
	resourceID, parsedAction, found := splitGCPStorageInsightsActionSegment(tail[1])
	return found && strings.TrimSpace(resourceID) != "" && parsedAction == action
}

func isGCPStorageInsightsOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPStorageInsightsOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPStorageInsightsOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	resourceID, parsedAction, found := splitGCPStorageInsightsActionSegment(tail[1])
	return found && strings.TrimSpace(resourceID) != "" && parsedAction == action
}

func splitGCPStorageInsightsActionSegment(segment string) (resourceID, action string, ok bool) {
	segment = strings.TrimSpace(segment)
	resourceID, action, ok = strings.Cut(segment, ":")
	if !ok {
		return "", "", false
	}
	resourceID = strings.TrimSpace(resourceID)
	action = strings.TrimSpace(action)
	if resourceID == "" || action == "" {
		return "", "", false
	}
	return resourceID, action, true
}

func parseGCPStorageInsightsReportConfigPath(path string) (project, location, reportConfigID string, ok bool) {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsReportConfigTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPStorageInsightsReportDetailsCollectionPath(path string) (project, location, reportConfigID string, tail []string, ok bool) {
	project, location, tail, ok = parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsReportDetailsCollectionTail(tail) {
		return "", "", "", nil, false
	}
	return project, location, strings.TrimSpace(tail[1]), tail, true
}

func parseGCPStorageInsightsReportDetailPath(path string) (project, location, reportConfigID, reportDetailID string, ok bool) {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsReportDetailTail(tail) {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPStorageInsightsDatasetConfigPath(path string) (project, location, datasetConfigID string, ok bool) {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsDatasetConfigTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPStorageInsightsDatasetConfigActionPath(path string) (project, location, datasetConfigID, action string, ok bool) {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "datasetConfigs" {
		return "", "", "", "", false
	}
	datasetConfigID, action, ok = splitGCPStorageInsightsActionSegment(tail[1])
	if !ok || strings.TrimSpace(datasetConfigID) == "" {
		return "", "", "", "", false
	}
	return project, location, datasetConfigID, action, true
}

func parseGCPStorageInsightsOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || !isGCPStorageInsightsOperationTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPStorageInsightsOperationActionPath(path string) (project, location, operationID, action string, ok bool) {
	project, location, tail, ok := parseGCPStorageInsightsLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", "", false
	}
	operationID, action, ok = splitGCPStorageInsightsActionSegment(tail[1])
	if !ok || strings.TrimSpace(operationID) == "" {
		return "", "", "", "", false
	}
	return project, location, operationID, action, true
}

func parseGCPStorageInsightsParent(parent string) (project, location string, ok bool) {
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

func parseGCPStorageInsightsReportConfigName(name string) (project, location, reportConfigID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "reportConfigs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	reportConfigID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || reportConfigID == "" {
		return "", "", "", false
	}
	return project, location, reportConfigID, true
}

func parseGCPStorageInsightsReportDetailName(name string) (project, location, reportConfigID, reportDetailID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "reportConfigs" || parts[6] != "reportDetails" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	reportConfigID = strings.TrimSpace(parts[5])
	reportDetailID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || reportConfigID == "" || reportDetailID == "" {
		return "", "", "", "", false
	}
	return project, location, reportConfigID, reportDetailID, true
}

func parseGCPStorageInsightsDatasetConfigName(name string) (project, location, datasetConfigID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "datasetConfigs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	datasetConfigID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || datasetConfigID == "" {
		return "", "", "", false
	}
	return project, location, datasetConfigID, true
}

func parseGCPStorageInsightsOperationName(name string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPStorageInsightsPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPStorageInsightsInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPStorageInsightsOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxPageSize))
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPStorageInsightsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPStorageInsightsDoneFilter(raw string) (set, done, ok bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, false, true
	}
	normalized := strings.ToLower(strings.ReplaceAll(value, " ", ""))
	switch {
	case strings.HasPrefix(normalized, "done="):
		normalized = normalized[len("done="):]
	case strings.HasPrefix(normalized, "done:"):
		normalized = normalized[len("done:"):]
	default:
		return false, false, false
	}
	switch normalized {
	case "true":
		return true, true, true
	case "false":
		return true, false, true
	default:
		return false, false, false
	}
}

func normalizeGCPStorageInsightsOrderBy(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func isGCPStorageInsightsOrderBy(orderBy string) bool {
	switch orderBy {
	case "", "name", "name asc", "name desc", "create_time", "create_time asc", "create_time desc":
		return true
	default:
		return false
	}
}

func isGCPStorageInsightsListFilter(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return true
	}
	normalized := strings.ToLower(strings.ReplaceAll(value, " ", ""))
	return strings.HasPrefix(normalized, "name=") ||
		strings.HasPrefix(normalized, "name:") ||
		strings.HasPrefix(normalized, "display_name=") ||
		strings.HasPrefix(normalized, "display_name:")
}

func sortGCPStorageInsightsItems(items []map[string]any, orderBy string) {
	switch orderBy {
	case "create_time", "create_time asc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStorageInsightsTime(items[i], "createTime").Before(gcpStorageInsightsTime(items[j], "createTime"))
		})
	case "create_time desc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStorageInsightsTime(items[i], "createTime").After(gcpStorageInsightsTime(items[j], "createTime"))
		})
	case "name desc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStorageInsightsString(items[i], "name") > gcpStorageInsightsString(items[j], "name")
		})
	default:
		sort.Slice(items, func(i, j int) bool {
			return gcpStorageInsightsString(items[i], "name") < gcpStorageInsightsString(items[j], "name")
		})
	}
}

func gcpStorageInsightsTime(item map[string]any, key string) time.Time {
	if raw := gcpStorageInsightsString(item, key); raw != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func decodeGCPStorageInsightsJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPStorageInsightsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPStorageInsightsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func decodeGCPStorageInsightsOptionalJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPStorageInsightsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPStorageInsightsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpStorageInsightsBodyMap(body map[string]any, key string) map[string]any {
	if nested, ok := body[key].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func validateGCPStorageInsightsReportConfig(w http.ResponseWriter, path string, reportConfig map[string]any, requireName bool) bool {
	if requireName && strings.TrimSpace(gcpStorageInsightsString(reportConfig, "name")) == "" {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.name is required")
		return false
	}
	if displayName := strings.TrimSpace(gcpStorageInsightsString(reportConfig, "displayName")); len(displayName) > 256 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.displayName must be <= 256 characters")
		return false
	}
	if _, ok := reportConfig["frequencyOptions"].(map[string]any); !ok {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.frequencyOptions is required")
		return false
	}
	reportFormatCount := 0
	if _, ok := reportConfig["csvOptions"].(map[string]any); ok {
		reportFormatCount++
	}
	if _, ok := reportConfig["parquetOptions"].(map[string]any); ok {
		reportFormatCount++
	}
	if reportFormatCount == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig report format is required")
		return false
	}
	if reportFormatCount > 1 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig report format oneof must contain exactly one option")
		return false
	}
	objectOptions, ok := reportConfig["objectMetadataReportOptions"].(map[string]any)
	if !ok || len(objectOptions) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.objectMetadataReportOptions is required")
		return false
	}
	if fields, ok := objectOptions["metadataFields"].([]any); !ok || len(fields) == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "reportConfig.objectMetadataReportOptions.metadataFields is required")
		return false
	}
	return true
}

func validateGCPStorageInsightsDatasetConfig(w http.ResponseWriter, path string, datasetConfig map[string]any, requireName bool) bool {
	if requireName && strings.TrimSpace(gcpStorageInsightsString(datasetConfig, "name")) == "" {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig.name is required")
		return false
	}
	if description := strings.TrimSpace(gcpStorageInsightsString(datasetConfig, "description")); len(description) > 256 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig.description must be <= 256 characters")
		return false
	}
	if retentionRaw, ok := datasetConfig["retentionPeriodDays"]; ok {
		if n, ok := asInt64GCPStorageInsights(retentionRaw); !ok || n < 0 {
			respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig.retentionPeriodDays must be non-negative")
			return false
		}
	}

	sourceFields := 0
	for _, key := range []string{"sourceProjects", "sourceFolders", "organizationScope", "cloudStorageObjectPath"} {
		if _, ok := datasetConfig[key]; ok {
			sourceFields++
		}
	}
	if sourceFields == 0 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig source option is required")
		return false
	}
	if sourceFields > 1 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig source option oneof must contain exactly one option")
		return false
	}

	locationFields := 0
	for _, key := range []string{"includeCloudStorageLocations", "excludeCloudStorageLocations"} {
		if _, ok := datasetConfig[key]; ok {
			locationFields++
		}
	}
	if locationFields > 1 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig cloud storage locations oneof must contain at most one option")
		return false
	}

	bucketFields := 0
	for _, key := range []string{"includeCloudStorageBuckets", "excludeCloudStorageBuckets"} {
		if _, ok := datasetConfig[key]; ok {
			bucketFields++
		}
	}
	if bucketFields > 1 {
		respondGCPStorageInsightsInvalidArgument(w, path, "datasetConfig cloud storage buckets oneof must contain at most one option")
		return false
	}
	return true
}

func validateGCPStorageInsightsUpdateMask(mask string, allowed []string) bool {
	if strings.TrimSpace(mask) == "" {
		return false
	}
	allowedSet := map[string]struct{}{}
	for _, field := range allowed {
		allowedSet[strings.ToLower(strings.TrimSpace(field))] = struct{}{}
	}
	for _, token := range strings.Split(mask, ",") {
		field := strings.ToLower(strings.TrimSpace(token))
		if field == "" {
			return false
		}
		if _, ok := allowedSet[field]; !ok {
			return false
		}
	}
	return true
}

func applyGCPStorageInsightsReportConfigOverrides(out, in map[string]any) {
	for _, key := range []string{
		"displayName",
		"labels",
		"frequencyOptions",
		"csvOptions",
		"parquetOptions",
		"objectMetadataReportOptions",
	} {
		if val, ok := in[key]; ok {
			out[key] = val
		}
	}
}

func applyGCPStorageInsightsDatasetConfigOverrides(out, in map[string]any) {
	for _, key := range []string{
		"description",
		"labels",
		"retentionPeriodDays",
		"sourceProjects",
		"sourceFolders",
		"organizationScope",
		"cloudStorageObjectPath",
		"includeCloudStorageLocations",
		"excludeCloudStorageLocations",
		"includeCloudStorageBuckets",
		"excludeCloudStorageBuckets",
		"includeNewlyCreatedBuckets",
		"identity",
		"organizationNumber",
	} {
		if val, ok := in[key]; ok {
			out[key] = val
		}
	}
}

func gcpStorageInsightsLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Storage Insights " + location,
		"labels": map[string]any{
			"provider": providerGCP,
			"service":  "storageinsights",
		},
		"metadata": map[string]any{
			"reportConfigAvailable":  true,
			"datasetConfigAvailable": true,
		},
	}
}

func gcpStorageInsightsReportConfigName(project, location, reportConfigID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/reportConfigs/%s", project, location, reportConfigID)
}

func gcpStorageInsightsReportConfig(project, location, reportConfigID string) map[string]any {
	createTime := gcpStorageInsightsReferenceTime
	updateTime := gcpStorageInsightsReferenceTime.Add(45 * time.Minute)
	return map[string]any{
		"name":        gcpStorageInsightsReportConfigName(project, location, reportConfigID),
		"createTime":  createTime.Format(time.RFC3339Nano),
		"updateTime":  updateTime.Format(time.RFC3339Nano),
		"displayName": fmt.Sprintf("Stackyard Report Config %s", reportConfigID),
		"frequencyOptions": map[string]any{
			"frequency": "DAILY",
			"startDate": map[string]any{
				"year":  2026,
				"month": 1,
				"day":   1,
			},
		},
		"csvOptions": map[string]any{
			"recordSeparator": "\n",
			"delimiter":       ",",
			"headerRequired":  true,
		},
		"objectMetadataReportOptions": map[string]any{
			"metadataFields": []any{"name", "size", "updated"},
			"storageFilters": map[string]any{
				"bucket": "stackyard-source-bucket",
			},
			"storageDestinationOptions": map[string]any{
				"bucket":          "stackyard-insights-bucket",
				"destinationPath": "reports/storageinsights",
			},
		},
		"labels": map[string]any{
			"env":  "staging",
			"team": "platform",
		},
	}
}

func gcpStorageInsightsReportDetailName(project, location, reportConfigID, reportDetailID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/reportConfigs/%s/reportDetails/%s", project, location, reportConfigID, reportDetailID)
}

func gcpStorageInsightsReportDetail(project, location, reportConfigID, reportDetailID string) map[string]any {
	snapshot := gcpStorageInsightsReferenceTime.Add(24 * time.Hour)
	return map[string]any{
		"name":             gcpStorageInsightsReportDetailName(project, location, reportConfigID, reportDetailID),
		"snapshotTime":     snapshot.Format(time.RFC3339Nano),
		"reportPathPrefix": fmt.Sprintf("gs://stackyard-insights-bucket/reports/%s/%s_", reportConfigID, snapshot.Format("2006-01-02T15:04")),
		"shardsCount":      3,
		"status": map[string]any{
			"code":    0,
			"message": "OK",
		},
		"labels": map[string]any{
			"config": reportConfigID,
		},
		"reportMetrics": map[string]any{
			"processedRecordsCount": 4242,
		},
	}
}

func gcpStorageInsightsDatasetConfigName(project, location, datasetConfigID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/datasetConfigs/%s", project, location, datasetConfigID)
}

func gcpStorageInsightsDatasetConfig(project, location, datasetConfigID string) map[string]any {
	createTime := gcpStorageInsightsReferenceTime.Add(2 * time.Hour)
	updateTime := createTime.Add(30 * time.Minute)
	return map[string]any{
		"name":       gcpStorageInsightsDatasetConfigName(project, location, datasetConfigID),
		"createTime": createTime.Format(time.RFC3339Nano),
		"updateTime": updateTime.Format(time.RFC3339Nano),
		"labels": map[string]any{
			"env":  "staging",
			"team": "platform",
		},
		"uid":                "datasetcfg-" + datasetConfigID,
		"organizationNumber": int64(123456789),
		"sourceProjects": map[string]any{
			"projectNumbers": []any{int64(123456789)},
		},
		"includeCloudStorageLocations": map[string]any{
			"locations": []any{"US"},
		},
		"includeCloudStorageBuckets": map[string]any{
			"cloudStorageBuckets": []any{
				map[string]any{
					"bucket": "stackyard-source-bucket",
				},
			},
		},
		"includeNewlyCreatedBuckets": true,
		"retentionPeriodDays":        30,
		"link": map[string]any{
			"dataset": fmt.Sprintf("projects/%s/datasets/storageinsights_%s", project, datasetConfigID),
			"linked":  true,
		},
		"identity": map[string]any{
			"name": fmt.Sprintf("serviceAccount:storageinsights-%s@%s.iam.gserviceaccount.com", datasetConfigID, project),
			"type": "IDENTITY_TYPE_PER_PROJECT",
		},
		"status": map[string]any{
			"code":    0,
			"message": "ACTIVE",
		},
		"datasetConfigState": "CONFIG_STATE_ACTIVE",
		"description":        "Stackyard dataset config " + datasetConfigID,
	}
}

func gcpStorageInsightsOperationName(project, location, operationID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
}

func gcpStorageInsightsOperationForAction(project, location, operationID string, datasetConfig map[string]any, requestedCancellation bool) map[string]any {
	target := fmt.Sprintf("projects/%s/locations/%s", project, location)
	verb := "UNKNOWN"
	done := true
	response := map[string]any(nil)

	switch {
	case strings.HasPrefix(operationID, "createDatasetConfig."):
		id := strings.TrimPrefix(operationID, "createDatasetConfig.")
		target = gcpStorageInsightsDatasetConfigName(project, location, id)
		verb = "create"
		if datasetConfig == nil {
			datasetConfig = gcpStorageInsightsDatasetConfig(project, location, id)
		}
		response = map[string]any{
			"@type": "type.googleapis.com/google.cloud.storageinsights.v1.DatasetConfig",
		}
		for key, value := range datasetConfig {
			response[key] = value
		}
	case strings.HasPrefix(operationID, "updateDatasetConfig."):
		id := strings.TrimPrefix(operationID, "updateDatasetConfig.")
		target = gcpStorageInsightsDatasetConfigName(project, location, id)
		verb = "update"
		if datasetConfig == nil {
			datasetConfig = gcpStorageInsightsDatasetConfig(project, location, id)
		}
		response = map[string]any{
			"@type": "type.googleapis.com/google.cloud.storageinsights.v1.DatasetConfig",
		}
		for key, value := range datasetConfig {
			response[key] = value
		}
	case strings.HasPrefix(operationID, "deleteDatasetConfig."):
		id := strings.TrimPrefix(operationID, "deleteDatasetConfig.")
		target = gcpStorageInsightsDatasetConfigName(project, location, id)
		verb = "delete"
	case strings.HasPrefix(operationID, "linkDataset."):
		id := strings.TrimPrefix(operationID, "linkDataset.")
		target = gcpStorageInsightsDatasetConfigName(project, location, id)
		verb = "link"
		response = map[string]any{
			"@type": "type.googleapis.com/google.cloud.storageinsights.v1.LinkDatasetResponse",
		}
	case strings.HasPrefix(operationID, "unlinkDataset."):
		id := strings.TrimPrefix(operationID, "unlinkDataset.")
		target = gcpStorageInsightsDatasetConfigName(project, location, id)
		verb = "unlink"
	case strings.HasPrefix(operationID, "linkDatasetRunning."):
		id := strings.TrimPrefix(operationID, "linkDatasetRunning.")
		target = gcpStorageInsightsDatasetConfigName(project, location, id)
		verb = "link"
		done = false
	}

	operationName := gcpStorageInsightsOperationName(project, location, operationID)
	metadata := map[string]any{
		"@type":                 "type.googleapis.com/google.cloud.storageinsights.v1.OperationMetadata",
		"createTime":            gcpStorageInsightsReferenceTime.Format(time.RFC3339Nano),
		"target":                target,
		"verb":                  verb,
		"statusMessage":         "done",
		"requestedCancellation": requestedCancellation,
		"apiVersion":            "v1",
	}
	if done {
		metadata["endTime"] = gcpStorageInsightsReferenceTime.Add(2 * time.Minute).Format(time.RFC3339Nano)
	}

	operation := map[string]any{
		"name":     operationName,
		"metadata": metadata,
		"done":     done,
	}
	if response != nil && done {
		operation["response"] = response
	}
	return operation
}

func respondGCPStorageInsightsList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string, includeUnreachable bool) bool {
	if start > len(items) {
		respondGCPStorageInsightsInvalidArgument(w, path, "pageToken out of range")
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
	payload := map[string]any{
		field:           items[start:end],
		"nextPageToken": nextToken,
	}
	if includeUnreachable {
		payload["unreachable"] = []any{}
	}
	respondJSON(w, http.StatusOK, payload)
	return true
}

func isGCPStorageInsightsRequestID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || !gcpStorageInsightsRequestIDPattern.MatchString(value) {
		return false
	}
	return strings.ToLower(value) != "00000000-0000-0000-0000-000000000000"
}

func isGCPStorageInsightsDatasetConfigID(id string) bool {
	return gcpStorageInsightsDatasetIDPattern.MatchString(strings.TrimSpace(id))
}

func isGCPStorageInsightsMissingID(id string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(id)), "missing")
}

func gcpStorageInsightsString(obj map[string]any, key string) string {
	value, ok := obj[key]
	if !ok || value == nil {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

func asInt64GCPStorageInsights(v any) (int64, bool) {
	switch typed := v.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), true
	case json.Number:
		n, err := typed.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}

func respondGCPStorageInsightsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPStorageInsightsError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPStorageInsightsNotFound(w http.ResponseWriter, path, message string) {
	respondGCPStorageInsightsError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPStorageInsightsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPStorageInsightsError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPStorageInsightsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPStorageInsightsError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPStorageInsightsOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPStorageInsightsError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPStorageInsightsError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_storageinsights(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "storageinsights") {
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
			"name":     "projects/stackyard/locations/us-central1/storageinsights/sample",
			"service":  "storageinsights",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
