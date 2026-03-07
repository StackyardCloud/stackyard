package server

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpShoppingMerchantQuotaAccountRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func (s *Server) handleGCPShoppingMerchantQuotaRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantQuota(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantQuotaPath(rawRequestPath(r))
	if !isGCPShoppingMerchantQuotaPath(path, hasGCPShoppingMerchantQuotaHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantQuotaRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.quota.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantQuotaGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func normalizeGCPShoppingMerchantQuotaPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantQuotaHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_quota",
		"shopping-merchant-quota",
		"shopping-merchant-quota-apiv1",
		"shopping_merchant_quota_apiv1",
		"merchant_quota",
		"merchant-quota",
		"merchantquota",
		"gcp-shopping-merchant-quota":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-quota-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/quota")
}

func isGCPShoppingMerchantQuotaPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/quota/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.quota.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/quota/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantQuotaRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/quota/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/quota/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantQuotaGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantQuotaCollectionPath(tail)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(account), "missing") {
		respondGCPShoppingMerchantQuotaNotFound(w, path, "account not found")
		return true
	}

	pageSize, start, ok := parseGCPShoppingMerchantQuotaPagination(w, r, path)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantQuotaGroupFixture(account, "product-read", 15, 3000, 120, "products.list"),
		gcpShoppingMerchantQuotaGroupFixture(account, "product-write", 3, 800, 45, "products.insert"),
		gcpShoppingMerchantQuotaGroupFixture(account, "promotion-write", 2, 400, 20, "promotions.insert"),
	}
	if start > len(items) {
		respondGCPShoppingMerchantQuotaInvalidArgument(w, path, "pageToken is out of range")
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
		"quotaGroups":   items[start:end],
		"nextPageToken": next,
	})
	return true
}

func parseGCPShoppingMerchantQuotaParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantQuotaAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantQuotaCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "quotas" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantQuotaAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantQuotaPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 500
	if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantQuotaInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > 1000 {
		pageSize = 1000
	}
	if pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken")); pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantQuotaInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func gcpShoppingMerchantQuotaGroupFixture(account, group string, usage, limit, minuteLimit int64, method string) map[string]any {
	return map[string]any{
		"name":             gcpShoppingMerchantQuotaGroupName(account, group),
		"quotaUsage":       strconv.FormatInt(usage, 10),
		"quotaLimit":       strconv.FormatInt(limit, 10),
		"quotaMinuteLimit": strconv.FormatInt(minuteLimit, 10),
		"methodDetails": []map[string]any{
			gcpShoppingMerchantQuotaMethodDetailsFixture(method),
		},
	}
}

func gcpShoppingMerchantQuotaMethodDetailsFixture(method string) map[string]any {
	return map[string]any{
		"method":  method,
		"version": "v1",
		"subapi":  "merchant-quota",
		"path":    "quota/v1/" + method,
	}
}

func gcpShoppingMerchantQuotaGroupName(account, group string) string {
	return fmt.Sprintf("accounts/%s/quotas/%s", strings.TrimSpace(account), strings.TrimSpace(group))
}

func respondGCPShoppingMerchantQuotaInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantQuotaFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantQuotaNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantQuota(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_quota") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_quota/sample",
			"service":   "shopping_merchant_quota",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
