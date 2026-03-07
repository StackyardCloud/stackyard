package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudChannelRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if strings.HasPrefix(path, "/gcp/google.cloud.channel.v1.CloudChannelService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.channel.v1.CloudChannelReportsService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	if !isGCPCloudChannelPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudChannelListCustomers(w, r, path) {
			return true
		}
		if handleGCPCloudChannelGetCustomer(w, path) {
			return true
		}
		if handleGCPCloudChannelListProducts(w, r, path) {
			return true
		}
		if handleGCPCloudChannelListOffers(w, r, path) {
			return true
		}
		if handleGCPCloudChannelListEntitlements(w, r, path) {
			return true
		}
		if handleGCPCloudChannelListReports(w, r, path) {
			return true
		}
		if handleGCPCloudChannelFetchReportResults(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudChannelCheckCloudIdentityAccountsExist(w, r, path) {
			return true
		}
		if handleGCPCloudChannelRunReportJob(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudChannelPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/v1/accounts/") {
		accountMarkers := []string{
			"/customers",
			"/entitlements",
			"/offers",
			"/reports",
			"/reportJobs/",
			":checkCloudIdentityAccountsExist",
			":run",
			":fetchReportResults",
		}
		for _, marker := range accountMarkers {
			if strings.Contains(path, marker) {
				return true
			}
		}
		return false
	}
	return path == "/gcp/v1/products" || strings.HasPrefix(path, "/gcp/v1/products/")
}

func handleGCPCloudChannelListCustomers(w http.ResponseWriter, r *http.Request, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 1 || tail[0] != "customers" {
		return false
	}
	pageSize, start, valid := parseGCPCloudChannelPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudChannelCustomer(account, "team-customer")}
	return respondGCPCloudChannelList(w, "customers", items, pageSize, start, path)
}

func handleGCPCloudChannelGetCustomer(w http.ResponseWriter, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 2 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudChannelCustomer(account, tail[1]))
	return true
}

func handleGCPCloudChannelCheckCloudIdentityAccountsExist(w http.ResponseWriter, r *http.Request, path string) bool {
	account, action, ok := parseGCPCloudChannelAccountActionPath(path)
	if !ok || action != "checkCloudIdentityAccountsExist" {
		return false
	}
	body, valid := decodeGCPCloudChannelJSONBody(w, r, path)
	if !valid {
		return true
	}
	domain := strings.TrimSpace(gcpCloudChannelString(body, "domain"))
	if domain == "" {
		respondGCPCloudChannelInvalidArgument(w, path, "domain is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"cloudIdentityAccounts": []any{
			map[string]any{
				"domain": domain,
				"exists": true,
			},
		},
		"nextPageToken": "",
		"account":       "accounts/" + account,
	})
	return true
}

func handleGCPCloudChannelListProducts(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/products" {
		return false
	}
	account := strings.TrimSpace(r.URL.Query().Get("account"))
	if account == "" {
		respondGCPCloudChannelInvalidArgument(w, path, "account is required")
		return true
	}
	pageSize, start, valid := parseGCPCloudChannelPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudChannelProduct("stackyard-product", account)}
	return respondGCPCloudChannelList(w, "products", items, pageSize, start, path)
}

func handleGCPCloudChannelListOffers(w http.ResponseWriter, r *http.Request, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 1 || tail[0] != "offers" {
		return false
	}
	pageSize, start, valid := parseGCPCloudChannelPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudChannelOffer(account, "offer-1")}
	return respondGCPCloudChannelList(w, "offers", items, pageSize, start, path)
}

func handleGCPCloudChannelListEntitlements(w http.ResponseWriter, r *http.Request, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 3 || tail[0] != "customers" || strings.TrimSpace(tail[1]) == "" || tail[2] != "entitlements" {
		return false
	}
	pageSize, start, valid := parseGCPCloudChannelPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudChannelEntitlement(account, tail[1], "entitlement-1")}
	return respondGCPCloudChannelList(w, "entitlements", items, pageSize, start, path)
}

func handleGCPCloudChannelListReports(w http.ResponseWriter, r *http.Request, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 1 || tail[0] != "reports" {
		return false
	}
	pageSize, start, valid := parseGCPCloudChannelPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudChannelReport(account, "613bf59q")}
	return respondGCPCloudChannelList(w, "reports", items, pageSize, start, path)
}

func handleGCPCloudChannelRunReportJob(w http.ResponseWriter, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 2 || tail[0] != "reports" {
		return false
	}
	reportID, action, found := strings.Cut(normalizeGCPCloudChannelActionSegment(tail[1]), ":")
	if !found || action != "run" || strings.TrimSpace(reportID) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudChannelOperation(account, "runReportJob."+strings.TrimSpace(reportID)))
	return true
}

func handleGCPCloudChannelFetchReportResults(w http.ResponseWriter, r *http.Request, path string) bool {
	account, tail, ok := parseGCPCloudChannelAccountTail(path)
	if !ok || len(tail) != 2 || tail[0] != "reportJobs" {
		return false
	}
	reportJobID, action, found := strings.Cut(normalizeGCPCloudChannelActionSegment(tail[1]), ":")
	if !found || action != "fetchReportResults" || strings.TrimSpace(reportJobID) == "" {
		return false
	}
	pageSize, start, valid := parseGCPCloudChannelPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"name":      fmt.Sprintf("accounts/%s/reportJobs/%s/reportResults/result-1", account, strings.TrimSpace(reportJobID)),
			"rowValues": map[string]any{"sku": "stackyard-product", "amount": "100"},
		},
	}
	return respondGCPCloudChannelList(w, "reportResults", items, pageSize, start, path)
}

func parseGCPCloudChannelAccountTail(path string) (account string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" {
		return "", nil, false
	}
	if strings.Contains(parts[3], ":") {
		return "", nil, false
	}
	account = strings.TrimSpace(parts[3])
	if account == "" {
		return "", nil, false
	}
	return account, parts[4:], true
}

func parseGCPCloudChannelAccountActionPath(path string) (account, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" {
		return "", "", false
	}
	accountAndAction := normalizeGCPCloudChannelActionSegment(parts[3])
	account, action, ok = strings.Cut(accountAndAction, ":")
	if !ok {
		return "", "", false
	}
	account = strings.TrimSpace(account)
	action = strings.TrimSpace(action)
	if account == "" || action == "" {
		return "", "", false
	}
	return account, action, true
}

func parseGCPCloudChannelPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudChannelInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCloudChannelInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCloudChannelList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudChannelInvalidArgument(w, path, "pageToken is out of range")
		return false
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

func decodeGCPCloudChannelJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudChannelInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudChannelString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return value
}

func normalizeGCPCloudChannelActionSegment(segment string) string {
	normalized := strings.TrimSpace(segment)
	normalized = strings.ReplaceAll(normalized, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func gcpCloudChannelCustomer(account, customerID string) map[string]any {
	return map[string]any{
		"name":           fmt.Sprintf("accounts/%s/customers/%s", account, customerID),
		"orgDisplayName": "Stackyard Customer",
		"domain":         "example.com",
	}
}

func gcpCloudChannelProduct(productID, account string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("products/%s", productID),
		"marketingInfo": map[string]any{"displayName": "Stackyard Product"},
		"account":       account,
	}
}

func gcpCloudChannelOffer(account, offerID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("accounts/%s/offers/%s", account, offerID),
		"sku":  "products/stackyard-product/skus/stackyard-sku",
		"plan": "ANNUAL",
	}
}

func gcpCloudChannelEntitlement(account, customerID, entitlementID string) map[string]any {
	return map[string]any{
		"name":               fmt.Sprintf("accounts/%s/customers/%s/entitlements/%s", account, customerID, entitlementID),
		"state":              "ACTIVE",
		"offer":              fmt.Sprintf("accounts/%s/offers/offer-1", account),
		"provisionedService": map[string]any{"skuId": "stackyard-sku"},
	}
}

func gcpCloudChannelReport(account, reportID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("accounts/%s/reports/%s", account, reportID),
		"displayName": "Monthly usage",
	}
}

func gcpCloudChannelOperation(account, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("accounts/%s/operations/%s", account, operationID),
		"done": true,
	}
}

func respondGCPCloudChannelInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
