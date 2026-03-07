package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpShoppingMerchantDatasourcesAccountRe      = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantDatasourcesDatasourceIDRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

func (s *Server) handleGCPShoppingMerchantDatasourcesRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantDatasources(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantDatasourcesPath(rawRequestPath(r))
	if !isGCPShoppingMerchantDatasourcesPath(path, hasGCPShoppingMerchantDatasourcesHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantDatasourcesRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.datasources.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantDatasourcesGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantDatasourcesPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPShoppingMerchantDatasourcesPATCH(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantDatasourcesDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantDatasourcesPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantDatasourcesHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_datasources",
		"shopping-merchant-datasources",
		"shopping-merchant-datasources-apiv1",
		"shopping_merchant_datasources_apiv1",
		"merchant_datasources",
		"merchant-datasources",
		"merchantdatasources",
		"gcp-shopping-merchant-datasources":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-datasources-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/datasources")
}

func isGCPShoppingMerchantDatasourcesPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/datasources/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.datasources.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/datasources/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantDatasourcesRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/datasources/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/datasources/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantDatasourcesGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantDatasourcesCollectionPath(tail); ok {
		return handleGCPShoppingMerchantDatasourcesList(w, r, path, account)
	}
	if account, dataSourceID, alias, ok := parseGCPShoppingMerchantDatasourcesFileUploadPath(tail); ok {
		if alias != "latest" {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "file upload alias must be latest")
			return true
		}
		if strings.Contains(strings.ToLower(dataSourceID), "missing") {
			respondGCPShoppingMerchantDatasourcesNotFound(w, path, "data source not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpShoppingMerchantDatasourcesFileUploadFixture(account, dataSourceID))
		return true
	}
	account, dataSourceID, action, ok := parseGCPShoppingMerchantDatasourcesResourcePath(tail)
	if !ok || action != "" {
		return false
	}
	if strings.Contains(strings.ToLower(dataSourceID), "missing") {
		respondGCPShoppingMerchantDatasourcesNotFound(w, path, "data source not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantDatasourcesDataSourceFixture(account, dataSourceID, false))
	return true
}

func handleGCPShoppingMerchantDatasourcesPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantDatasourcesCollectionPath(tail); ok {
		return handleGCPShoppingMerchantDatasourcesCreate(w, r, path, account)
	}
	account, dataSourceID, action, ok := parseGCPShoppingMerchantDatasourcesResourcePath(tail)
	if !ok || action != "fetch" {
		return false
	}
	if strings.Contains(strings.ToLower(dataSourceID), "missing") {
		respondGCPShoppingMerchantDatasourcesNotFound(w, path, "data source not found")
		return true
	}
	body, ok := decodeGCPShoppingMerchantDatasourcesJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	if name := strings.TrimSpace(gcpShoppingMerchantDatasourcesString(body, "name")); name != "" {
		expected := fmt.Sprintf("accounts/%s/dataSources/%s", account, dataSourceID)
		if name != expected {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "name must match requested resource")
			return true
		}
	}
	if strings.Contains(strings.ToLower(dataSourceID), "nofile") {
		respondGCPShoppingMerchantDatasourcesFailedPrecondition(w, path, "fetch requires a data source with file input")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantDatasourcesPATCH(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, dataSourceID, action, ok := parseGCPShoppingMerchantDatasourcesResourcePath(tail)
	if !ok || action != "" {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, ok := decodeGCPShoppingMerchantDatasourcesJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expected := fmt.Sprintf("accounts/%s/dataSources/%s", account, dataSourceID)
	if name := strings.TrimSpace(gcpShoppingMerchantDatasourcesString(body, "name")); name == "" || name != expected {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "data source name must match requested resource")
		return true
	}
	if strings.TrimSpace(gcpShoppingMerchantDatasourcesString(body, "displayName", "display_name")) == "" {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "displayName is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantDatasourcesDataSourceFixture(account, dataSourceID, false))
	return true
}

func handleGCPShoppingMerchantDatasourcesDELETE(w http.ResponseWriter, _ *http.Request, path, tail string) bool {
	_, dataSourceID, action, ok := parseGCPShoppingMerchantDatasourcesResourcePath(tail)
	if !ok || action != "" {
		return false
	}
	if strings.Contains(strings.ToLower(dataSourceID), "missing") {
		respondGCPShoppingMerchantDatasourcesNotFound(w, path, "data source not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantDatasourcesList(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantDatasourcesPagination(w, r, path)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantDatasourcesDataSourceFixture(account, "1001", false),
		gcpShoppingMerchantDatasourcesDataSourceFixture(account, "1002", false),
	}
	if start > len(items) {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "pageToken is out of range")
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
		"dataSources":   items[start:end],
		"nextPageToken": next,
	})
	return true
}

func handleGCPShoppingMerchantDatasourcesCreate(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantDatasourcesJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	displayName := strings.TrimSpace(gcpShoppingMerchantDatasourcesString(body, "displayName", "display_name"))
	if displayName == "" {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "displayName is required")
		return true
	}
	if strings.Contains(strings.ToLower(displayName), "existing") {
		respondGCPShoppingMerchantDatasourcesAlreadyExists(w, path, "data source already exists")
		return true
	}
	if name := strings.TrimSpace(gcpShoppingMerchantDatasourcesString(body, "name")); name != "" {
		expPrefix := fmt.Sprintf("accounts/%s/dataSources/", account)
		if !strings.HasPrefix(name, expPrefix) {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "name must match parent account")
			return true
		}
	}
	typeCount := 0
	for _, k := range []string{
		"primaryProductDataSource",
		"supplementalProductDataSource",
		"localInventoryDataSource",
		"regionalInventoryDataSource",
		"promotionDataSource",
		"productReviewDataSource",
		"merchantReviewDataSource",
	} {
		if _, ok := body[k]; ok {
			typeCount++
		}
	}
	if typeCount != 1 {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "exactly one data source type is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantDatasourcesDataSourceFixture(account, "1001", true))
	return true
}

func parseGCPShoppingMerchantDatasourcesCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "dataSources" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantDatasourcesAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantDatasourcesResourcePath(tail string) (account, dataSourceID, action string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(tail), "/")
	action = ""
	if strings.HasSuffix(trimmed, ":fetch") {
		action = "fetch"
		trimmed = strings.TrimSuffix(trimmed, ":fetch")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "dataSources" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	dataSourceID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantDatasourcesAccountRe.MatchString(account) || !gcpShoppingMerchantDatasourcesDatasourceIDRe.MatchString(dataSourceID) {
		return "", "", "", false
	}
	return account, dataSourceID, action, true
}

func parseGCPShoppingMerchantDatasourcesFileUploadPath(tail string) (account, dataSourceID, alias string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 6 || parts[0] != "accounts" || parts[2] != "dataSources" || parts[4] != "fileUploads" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	dataSourceID = strings.TrimSpace(parts[3])
	alias = strings.TrimSpace(parts[5])
	if !gcpShoppingMerchantDatasourcesAccountRe.MatchString(account) || !gcpShoppingMerchantDatasourcesDatasourceIDRe.MatchString(dataSourceID) || alias == "" {
		return "", "", "", false
	}
	return account, dataSourceID, alias, true
}

func parseGCPShoppingMerchantDatasourcesPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 1000
	pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func decodeGCPShoppingMerchantDatasourcesJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPShoppingMerchantDatasourcesJSONBody(w, r, path, true)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func decodeGCPShoppingMerchantDatasourcesJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		if required {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		if required {
			respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantDatasourcesInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpShoppingMerchantDatasourcesString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatInt(int64(v), 10)
		case int:
			return strconv.Itoa(v)
		case int64:
			return strconv.FormatInt(v, 10)
		}
	}
	return ""
}

func gcpShoppingMerchantDatasourcesDataSourceFixture(account, datasourceID string, created bool) map[string]any {
	displayName := "Stackyard Data Source " + datasourceID
	if created {
		displayName = "Stackyard Created Data Source"
	}
	return map[string]any{
		"name":         fmt.Sprintf("accounts/%s/dataSources/%s", account, datasourceID),
		"dataSourceId": datasourceID,
		"displayName":  displayName,
		"input":        "FILE",
		"fileInput": map[string]any{
			"fileName":      "products.csv",
			"fileInputType": "UPLOAD",
		},
		"primaryProductDataSource": map[string]any{
			"feedLabel":       "US",
			"contentLanguage": "en",
			"countries":       []any{"US"},
			"destinations": []any{
				map[string]any{
					"destination": "Shopping_ads",
					"state":       "ENABLED",
				},
			},
		},
	}
}

func gcpShoppingMerchantDatasourcesFileUploadFixture(account, datasourceID string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("accounts/%s/dataSources/%s/fileUploads/latest", account, datasourceID),
		"dataSourceId":    datasourceID,
		"processingState": "SUCCEEDED",
		"issues": []any{
			map[string]any{
				"title":            "Missing gtin",
				"description":      "Some products are missing gtin",
				"code":             "validation/missing_gtin",
				"count":            "1",
				"severity":         "WARNING",
				"documentationUri": "https://support.google.com/merchants/answer/7052112",
			},
		},
		"itemsTotal":   "10",
		"itemsCreated": "6",
		"itemsUpdated": "4",
		"uploadTime":   time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
}

func respondGCPShoppingMerchantDatasourcesInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantDatasourcesAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantDatasourcesNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantDatasourcesFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantDatasources(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_datasources") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/shopping_merchant_datasources/sample",
			"service":  "shopping_merchant_datasources",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
