package server

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpShoppingMerchantReportsAccountRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func (s *Server) handleGCPShoppingMerchantReportsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantReports(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantReportsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantReportsPath(path, hasGCPShoppingMerchantReportsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantReportsRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.reports.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodPost:
		if handleGCPShoppingMerchantReportsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func normalizeGCPShoppingMerchantReportsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantReportsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_reports",
		"shopping-merchant-reports",
		"shopping-merchant-reports-apiv1",
		"shopping_merchant_reports_apiv1",
		"merchant_reports",
		"merchant-reports",
		"merchantreports",
		"gcp-shopping-merchant-reports":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-reports-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/reports")
}

func isGCPShoppingMerchantReportsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/reports/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.reports.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/reports/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantReportsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/reports/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/reports/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantReportsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantReportsSearchPath(tail)
	if !ok {
		return false
	}

	body, ok := decodeGCPShoppingMerchantReportsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	parent := strings.TrimSpace(gcpShoppingMerchantReportsString(body, "parent"))
	if parent != "" {
		parentAccount, ok := parseGCPShoppingMerchantReportsParent(parent)
		if !ok {
			respondGCPShoppingMerchantReportsInvalidArgument(w, path, "parent must be in accounts/{account} format")
			return true
		}
		if parentAccount != account {
			respondGCPShoppingMerchantReportsFailedPrecondition(w, path, "request parent account must match URL account")
			return true
		}
	}
	if strings.Contains(strings.ToLower(account), "missing") {
		respondGCPShoppingMerchantReportsNotFound(w, path, "account not found")
		return true
	}

	query := strings.TrimSpace(gcpShoppingMerchantReportsString(body, "query"))
	if reason := validateGCPShoppingMerchantReportsQuery(query); reason != "" {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, reason)
		return true
	}

	pageSize, start, ok := parseGCPShoppingMerchantReportsPagination(w, r, path, body)
	if !ok {
		return true
	}

	items := []map[string]any{
		gcpShoppingMerchantReportsRowFixture(account, "online~en~US~sku-1001", "Stackyard Tee", 42, 420),
		gcpShoppingMerchantReportsRowFixture(account, "online~en~US~sku-1002", "Stackyard Hoodie", 21, 280),
		gcpShoppingMerchantReportsRowFixture(account, "online~en~US~sku-1003", "Stackyard Cap", 9, 120),
	}
	if start > len(items) {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, "pageToken is out of range")
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
		"results":       items[start:end],
		"nextPageToken": next,
	})
	return true
}

func parseGCPShoppingMerchantReportsSearchPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "reports:search" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantReportsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantReportsParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantReportsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func validateGCPShoppingMerchantReportsQuery(query string) string {
	if strings.TrimSpace(query) == "" {
		return "query is required"
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if !strings.Contains(q, "select") || !strings.Contains(q, "from") {
		return "query must include SELECT and FROM clauses"
	}
	return ""
}

func parseGCPShoppingMerchantReportsPagination(w http.ResponseWriter, r *http.Request, path string, body map[string]any) (pageSize, start int, ok bool) {
	pageSize = 1000
	if pageSizeRaw := gcpShoppingMerchantReportsString(body, "pageSize", "page_size"); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantReportsInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	} else if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantReportsInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > 5000 {
		pageSize = 5000
	}

	if pageTokenRaw := gcpShoppingMerchantReportsString(body, "pageToken", "page_token"); pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantReportsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	} else if pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken")); pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantReportsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func decodeGCPShoppingMerchantReportsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantReportsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpShoppingMerchantReportsRowFixture(account, id, title string, clicks, impressions int64) map[string]any {
	offerID := id
	if parts := strings.Split(id, "~"); len(parts) > 0 {
		offerID = parts[len(parts)-1]
	}
	return map[string]any{
		"productView": map[string]any{
			"id":           id,
			"languageCode": "en",
			"feedLabel":    "US",
			"offerId":      offerID,
			"title":        title,
			"brand":        "Stackyard",
			"channel":      "ONLINE",
			"availability": "IN_STOCK",
		},
		"productPerformanceView": map[string]any{
			"offerId":          offerID,
			"title":            title,
			"clicks":           strconv.FormatInt(clicks, 10),
			"impressions":      strconv.FormatInt(impressions, 10),
			"clickThroughRate": 0.1,
		},
		"nonProductPerformanceView": map[string]any{
			"clicks":           "4",
			"impressions":      "40",
			"clickThroughRate": 0.1,
		},
	}
}

func gcpShoppingMerchantReportsString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			return strings.TrimSpace(value)
		case float64:
			return strconv.FormatInt(int64(value), 10)
		case int:
			return strconv.Itoa(value)
		case int64:
			return strconv.FormatInt(value, 10)
		}
	}
	return ""
}

func respondGCPShoppingMerchantReportsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantReportsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantReportsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantReports(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_reports") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_reports/sample",
			"service":   "shopping_merchant_reports",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
