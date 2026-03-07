package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var (
	gcpShoppingMerchantInventoriesAccountRe   = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantInventoriesProductRe   = regexp.MustCompile(`^[A-Za-z0-9._~:-]+$`)
	gcpShoppingMerchantInventoriesStoreCodeRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	gcpShoppingMerchantInventoriesRegionRe    = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

func (s *Server) handleGCPShoppingMerchantInventoriesRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantInventories(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantInventoriesPath(rawRequestPath(r))
	if !isGCPShoppingMerchantInventoriesPath(path, hasGCPShoppingMerchantInventoriesHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantInventoriesRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.inventories.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantInventoriesGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantInventoriesPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantInventoriesDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantInventoriesPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantInventoriesHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_inventories",
		"shopping-merchant-inventories",
		"shopping-merchant-inventories-apiv1",
		"shopping_merchant_inventories_apiv1",
		"merchant_inventories",
		"merchant-inventories",
		"merchantinventories",
		"gcp-shopping-merchant-inventories":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-inventories-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/inventories")
}

func isGCPShoppingMerchantInventoriesPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/inventories/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.inventories.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/inventories/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantInventoriesRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/inventories/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/inventories/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantInventoriesGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, product, ok := parseGCPShoppingMerchantInventoriesLocalCollectionPath(tail); ok {
		return handleGCPShoppingMerchantInventoriesListLocal(w, r, path, account, product)
	}
	if account, product, ok := parseGCPShoppingMerchantInventoriesRegionalCollectionPath(tail); ok {
		return handleGCPShoppingMerchantInventoriesListRegional(w, r, path, account, product)
	}
	return false
}

func handleGCPShoppingMerchantInventoriesPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, product, ok := parseGCPShoppingMerchantInventoriesLocalInsertPath(tail); ok {
		return handleGCPShoppingMerchantInventoriesInsertLocal(w, r, path, account, product)
	}
	if account, product, ok := parseGCPShoppingMerchantInventoriesRegionalInsertPath(tail); ok {
		return handleGCPShoppingMerchantInventoriesInsertRegional(w, r, path, account, product)
	}
	return false
}

func handleGCPShoppingMerchantInventoriesDELETE(w http.ResponseWriter, _ *http.Request, path, tail string) bool {
	if _, _, storeCode, ok := parseGCPShoppingMerchantInventoriesLocalNamePath(tail); ok {
		if strings.Contains(strings.ToLower(storeCode), "missing") {
			respondGCPShoppingMerchantInventoriesNotFound(w, path, "local inventory not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	}
	if _, _, region, ok := parseGCPShoppingMerchantInventoriesRegionalNamePath(tail); ok {
		if strings.Contains(strings.ToLower(region), "missing") {
			respondGCPShoppingMerchantInventoriesNotFound(w, path, "regional inventory not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	}
	return false
}

func handleGCPShoppingMerchantInventoriesListLocal(w http.ResponseWriter, r *http.Request, path, account, product string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantInventoriesPagination(w, r, path, 25000, 25000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantInventoriesLocalFixture(account, product, "store-nyc"),
		gcpShoppingMerchantInventoriesLocalFixture(account, product, "store-sfo"),
	}
	if start > len(items) {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "pageToken is out of range")
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
		"localInventories": items[start:end],
		"nextPageToken":    next,
	})
	return true
}

func handleGCPShoppingMerchantInventoriesListRegional(w http.ResponseWriter, r *http.Request, path, account, product string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantInventoriesPagination(w, r, path, 25000, 100000)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantInventoriesRegionalFixture(account, product, "us-east1"),
		gcpShoppingMerchantInventoriesRegionalFixture(account, product, "us-west1"),
	}
	if start > len(items) {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "pageToken is out of range")
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
		"regionalInventories": items[start:end],
		"nextPageToken":       next,
	})
	return true
}

func handleGCPShoppingMerchantInventoriesInsertLocal(w http.ResponseWriter, r *http.Request, path, account, product string) bool {
	body, ok := decodeGCPShoppingMerchantInventoriesJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	storeCode := strings.TrimSpace(gcpShoppingMerchantInventoriesString(body, "storeCode", "store_code"))
	if storeCode == "" {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "storeCode is required")
		return true
	}
	if !gcpShoppingMerchantInventoriesStoreCodeRe.MatchString(storeCode) {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "storeCode is invalid")
		return true
	}
	if name := strings.TrimSpace(gcpShoppingMerchantInventoriesString(body, "name")); name != "" {
		expected := fmt.Sprintf("accounts/%s/products/%s/localInventories/%s", account, product, storeCode)
		if name != expected {
			respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "name must match parent and storeCode")
			return true
		}
	}

	attrs := gcpShoppingMerchantInventoriesMap(body, "localInventoryAttributes", "local_inventory_attributes")
	if len(attrs) > 0 {
		if qty, has, valid := gcpShoppingMerchantInventoriesInt64(attrs, "quantity"); has {
			if !valid || qty < 0 {
				respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "quantity must be a non-negative integer")
				return true
			}
		}
		hasPickupMethod := gcpShoppingMerchantInventoriesHasAny(attrs, "pickupMethod", "pickup_method")
		hasPickupSLA := gcpShoppingMerchantInventoriesHasAny(attrs, "pickupSla", "pickup_sla")
		if hasPickupMethod != hasPickupSLA {
			respondGCPShoppingMerchantInventoriesFailedPrecondition(w, path, "pickupMethod and pickupSla must be provided together")
			return true
		}
		hasSalePriceEffectiveDate := gcpShoppingMerchantInventoriesHasAny(attrs, "salePriceEffectiveDate", "sale_price_effective_date")
		hasSalePrice := gcpShoppingMerchantInventoriesHasAny(attrs, "salePrice", "sale_price")
		if hasSalePriceEffectiveDate && !hasSalePrice {
			respondGCPShoppingMerchantInventoriesFailedPrecondition(w, path, "salePrice is required when salePriceEffectiveDate is set")
			return true
		}
	}

	respondJSON(w, http.StatusOK, gcpShoppingMerchantInventoriesLocalFixture(account, product, storeCode))
	return true
}

func handleGCPShoppingMerchantInventoriesInsertRegional(w http.ResponseWriter, r *http.Request, path, account, product string) bool {
	body, ok := decodeGCPShoppingMerchantInventoriesJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	region := strings.TrimSpace(gcpShoppingMerchantInventoriesString(body, "region"))
	if region == "" {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "region is required")
		return true
	}
	if !gcpShoppingMerchantInventoriesRegionRe.MatchString(region) {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "region is invalid")
		return true
	}
	if name := strings.TrimSpace(gcpShoppingMerchantInventoriesString(body, "name")); name != "" {
		expected := fmt.Sprintf("accounts/%s/products/%s/regionalInventories/%s", account, product, region)
		if name != expected {
			respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "name must match parent and region")
			return true
		}
	}

	attrs := gcpShoppingMerchantInventoriesMap(body, "regionalInventoryAttributes", "regional_inventory_attributes")
	if len(attrs) > 0 {
		hasSalePriceEffectiveDate := gcpShoppingMerchantInventoriesHasAny(attrs, "salePriceEffectiveDate", "sale_price_effective_date")
		hasSalePrice := gcpShoppingMerchantInventoriesHasAny(attrs, "salePrice", "sale_price")
		if hasSalePriceEffectiveDate && !hasSalePrice {
			respondGCPShoppingMerchantInventoriesFailedPrecondition(w, path, "salePrice is required when salePriceEffectiveDate is set")
			return true
		}
	}

	respondJSON(w, http.StatusOK, gcpShoppingMerchantInventoriesRegionalFixture(account, product, region))
	return true
}

func parseGCPShoppingMerchantInventoriesLocalCollectionPath(tail string) (account, product string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 5 || parts[0] != "accounts" || parts[2] != "products" || parts[4] != "localInventories" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) || !gcpShoppingMerchantInventoriesProductRe.MatchString(product) {
		return "", "", false
	}
	return account, product, true
}

func parseGCPShoppingMerchantInventoriesRegionalCollectionPath(tail string) (account, product string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 5 || parts[0] != "accounts" || parts[2] != "products" || parts[4] != "regionalInventories" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) || !gcpShoppingMerchantInventoriesProductRe.MatchString(product) {
		return "", "", false
	}
	return account, product, true
}

func parseGCPShoppingMerchantInventoriesLocalInsertPath(tail string) (account, product string, ok bool) {
	if !strings.HasSuffix(tail, "/localInventories:insert") {
		return "", "", false
	}
	return parseGCPShoppingMerchantInventoriesLocalCollectionPath(strings.TrimSuffix(tail, ":insert"))
}

func parseGCPShoppingMerchantInventoriesRegionalInsertPath(tail string) (account, product string, ok bool) {
	if !strings.HasSuffix(tail, "/regionalInventories:insert") {
		return "", "", false
	}
	return parseGCPShoppingMerchantInventoriesRegionalCollectionPath(strings.TrimSuffix(tail, ":insert"))
}

func parseGCPShoppingMerchantInventoriesLocalNamePath(tail string) (account, product, storeCode string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 6 || parts[0] != "accounts" || parts[2] != "products" || parts[4] != "localInventories" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	storeCode = strings.TrimSpace(parts[5])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) ||
		!gcpShoppingMerchantInventoriesProductRe.MatchString(product) ||
		!gcpShoppingMerchantInventoriesStoreCodeRe.MatchString(storeCode) {
		return "", "", "", false
	}
	return account, product, storeCode, true
}

func parseGCPShoppingMerchantInventoriesRegionalNamePath(tail string) (account, product, region string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 6 || parts[0] != "accounts" || parts[2] != "products" || parts[4] != "regionalInventories" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	region = strings.TrimSpace(parts[5])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) ||
		!gcpShoppingMerchantInventoriesProductRe.MatchString(product) ||
		!gcpShoppingMerchantInventoriesRegionRe.MatchString(region) {
		return "", "", "", false
	}
	return account, product, region, true
}

func parseGCPShoppingMerchantInventoriesPagination(w http.ResponseWriter, r *http.Request, path string, defaultPageSize, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = defaultPageSize
	pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func decodeGCPShoppingMerchantInventoriesJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantInventoriesInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpShoppingMerchantInventoriesString(m map[string]any, keys ...string) string {
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

func gcpShoppingMerchantInventoriesInt64(m map[string]any, key string) (value int64, has bool, valid bool) {
	raw, ok := m[key]
	if !ok {
		return 0, false, true
	}
	switch v := raw.(type) {
	case float64:
		return int64(v), true, v == float64(int64(v))
	case int:
		return int64(v), true, true
	case int64:
		return v, true, true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, true, false
		}
		return parsed, true, true
	default:
		return 0, true, false
	}
}

func gcpShoppingMerchantInventoriesMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok {
			continue
		}
		obj, ok := raw.(map[string]any)
		if ok {
			return obj
		}
	}
	return map[string]any{}
}

func gcpShoppingMerchantInventoriesHasAny(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func gcpShoppingMerchantInventoriesAccountNumberString(account string) string {
	if parsed, err := strconv.ParseInt(account, 10, 64); err == nil && parsed >= 0 {
		return strconv.FormatInt(parsed, 10)
	}
	return "123456"
}

func gcpShoppingMerchantInventoriesLocalFixture(account, product, storeCode string) map[string]any {
	return map[string]any{
		"name":      fmt.Sprintf("accounts/%s/products/%s/localInventories/%s", account, product, storeCode),
		"account":   gcpShoppingMerchantInventoriesAccountNumberString(account),
		"storeCode": storeCode,
		"localInventoryAttributes": map[string]any{
			"quantity": "8",
		},
	}
}

func gcpShoppingMerchantInventoriesRegionalFixture(account, product, region string) map[string]any {
	return map[string]any{
		"name":    fmt.Sprintf("accounts/%s/products/%s/regionalInventories/%s", account, product, region),
		"account": gcpShoppingMerchantInventoriesAccountNumberString(account),
		"region":  region,
		"regionalInventoryAttributes": map[string]any{
			"availability": "REGIONAL_INVENTORY_AVAILABILITY_UNSPECIFIED",
		},
	}
}

func respondGCPShoppingMerchantInventoriesInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantInventoriesNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantInventoriesFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantInventories(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_inventories") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/shopping_merchant_inventories/sample",
			"service":  "shopping_merchant_inventories",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
