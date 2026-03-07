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
	gcpShoppingMerchantConversionsAccountRe  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantConversionsSourceIDRe = regexp.MustCompile(`^[a-z]{4}:[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantConversionsCurrencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
)

func (s *Server) handleGCPShoppingMerchantConversionsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantConversions(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantConversionsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantConversionsPath(path, hasGCPShoppingMerchantConversionsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantConversionsRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.conversions.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantConversionsGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantConversionsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPShoppingMerchantConversionsPATCH(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantConversionsDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantConversionsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantConversionsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_conversions",
		"shopping-merchant-conversions",
		"shopping-merchant-conversions-apiv1",
		"shopping_merchant_conversions_apiv1",
		"merchant_conversions",
		"merchant-conversions",
		"merchantconversions",
		"gcp-shopping-merchant-conversions":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-conversions-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/conversions")
}

func isGCPShoppingMerchantConversionsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/conversions/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.conversions.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/conversions/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantConversionsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/conversions/v1/") {
		return "", false
	}
	tail := strings.TrimSpace(strings.TrimPrefix(path, "/gcp/conversions/v1/"))
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantConversionsGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantConversionsCollectionPath(tail); ok {
		return handleGCPShoppingMerchantConversionsList(w, r, path, account)
	}
	account, sourceID, action, ok := parseGCPShoppingMerchantConversionsResourcePath(tail)
	if !ok || action != "" {
		return false
	}
	_ = account
	if strings.Contains(strings.ToLower(sourceID), "missing") {
		respondGCPShoppingMerchantConversionsNotFound(w, path, "conversion source not found")
		return true
	}
	state := "ACTIVE"
	if sourceID == "mcdn:1002" {
		state = "ARCHIVED"
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantConversionsConversionSourceFixture(account, sourceID, state, "", ""))
	return true
}

func handleGCPShoppingMerchantConversionsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantConversionsCollectionPath(tail); ok {
		return handleGCPShoppingMerchantConversionsCreate(w, r, path, account)
	}
	account, sourceID, action, ok := parseGCPShoppingMerchantConversionsResourcePath(tail)
	if !ok || action != "undelete" {
		return false
	}
	if strings.Contains(strings.ToLower(sourceID), "missing") {
		respondGCPShoppingMerchantConversionsNotFound(w, path, "conversion source not found")
		return true
	}
	if strings.HasPrefix(sourceID, "galk:") {
		respondGCPShoppingMerchantConversionsFailedPrecondition(w, path, "google analytics links cannot be undeleted")
		return true
	}
	body, ok := decodeGCPShoppingMerchantConversionsJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	if name := strings.TrimSpace(gcpShoppingMerchantConversionsString(body, "name")); name != "" {
		expected := fmt.Sprintf("accounts/%s/conversionSources/%s", account, sourceID)
		if name != expected {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "name must match requested resource")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantConversionsConversionSourceFixture(account, sourceID, "ACTIVE", "", ""))
	return true
}

func handleGCPShoppingMerchantConversionsPATCH(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, sourceID, action, ok := parseGCPShoppingMerchantConversionsResourcePath(tail)
	if !ok || action != "" {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if strings.HasPrefix(sourceID, "galk:") {
		respondGCPShoppingMerchantConversionsFailedPrecondition(w, path, "google analytics links do not support update")
		return true
	}
	body, ok := decodeGCPShoppingMerchantConversionsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("accounts/%s/conversionSources/%s", account, sourceID)
	if name := strings.TrimSpace(gcpShoppingMerchantConversionsString(body, "name")); name == "" || name != expectedName {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	mcd := gcpShoppingMerchantConversionsMap(body, "merchantCenterDestination", "merchant_center_destination")
	displayName := strings.TrimSpace(gcpShoppingMerchantConversionsString(mcd, "displayName", "display_name"))
	currency := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantConversionsString(mcd, "currencyCode", "currency_code")))
	if displayName == "" && currency == "" {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "merchantCenterDestination update payload is required")
		return true
	}
	if displayName != "" && len(displayName) > 64 {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "displayName must be <= 64 characters")
		return true
	}
	if currency != "" && !gcpShoppingMerchantConversionsCurrencyRe.MatchString(currency) {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "currencyCode must be an ISO 4217 code")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantConversionsConversionSourceFixture(account, sourceID, "ACTIVE", displayName, currency))
	return true
}

func handleGCPShoppingMerchantConversionsDELETE(w http.ResponseWriter, _ *http.Request, path, tail string) bool {
	_, sourceID, action, ok := parseGCPShoppingMerchantConversionsResourcePath(tail)
	if !ok || action != "" {
		return false
	}
	if strings.Contains(strings.ToLower(sourceID), "missing") {
		respondGCPShoppingMerchantConversionsNotFound(w, path, "conversion source not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantConversionsList(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, showDeleted, ok := parseGCPShoppingMerchantConversionsListParams(w, r, path)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantConversionsConversionSourceFixture(account, "mcdn:1001", "ACTIVE", "Primary Destination", "USD"),
		gcpShoppingMerchantConversionsConversionSourceFixture(account, "galk:2001", "ACTIVE", "", ""),
		gcpShoppingMerchantConversionsConversionSourceFixture(account, "mcdn:1002", "ARCHIVED", "Archived Destination", "USD"),
	}
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if !showDeleted && strings.EqualFold(gcpShoppingMerchantConversionsString(item, "state"), "ARCHIVED") {
			continue
		}
		filtered = append(filtered, item)
	}
	if start > len(filtered) {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(filtered)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(filtered) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"conversionSources": filtered[start:end],
		"nextPageToken":     next,
	})
	return true
}

func handleGCPShoppingMerchantConversionsCreate(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantConversionsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	hasGALink := len(gcpShoppingMerchantConversionsMap(body, "googleAnalyticsLink", "google_analytics_link")) > 0
	hasMCDestination := len(gcpShoppingMerchantConversionsMap(body, "merchantCenterDestination", "merchant_center_destination")) > 0
	if hasGALink == hasMCDestination {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "exactly one source type is required")
		return true
	}

	if hasGALink {
		ga := gcpShoppingMerchantConversionsMap(body, "googleAnalyticsLink", "google_analytics_link")
		propertyID := strings.TrimSpace(gcpShoppingMerchantConversionsString(ga, "propertyId", "property_id"))
		if propertyID == "" {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "googleAnalyticsLink.propertyId is required")
			return true
		}
		if _, err := strconv.ParseInt(propertyID, 10, 64); err != nil {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "googleAnalyticsLink.propertyId must be an integer")
			return true
		}
		if strings.HasPrefix(propertyID, "9") {
			respondGCPShoppingMerchantConversionsAlreadyExists(w, path, "conversion source already exists")
			return true
		}
		sourceID := "galk:" + propertyID
		respondJSON(w, http.StatusOK, gcpShoppingMerchantConversionsConversionSourceFixture(account, sourceID, "ACTIVE", "", ""))
		return true
	}

	mcd := gcpShoppingMerchantConversionsMap(body, "merchantCenterDestination", "merchant_center_destination")
	displayName := strings.TrimSpace(gcpShoppingMerchantConversionsString(mcd, "displayName", "display_name"))
	if displayName == "" {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "merchantCenterDestination.displayName is required")
		return true
	}
	if len(displayName) > 64 {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "displayName must be <= 64 characters")
		return true
	}
	if strings.Contains(strings.ToLower(displayName), "existing") {
		respondGCPShoppingMerchantConversionsAlreadyExists(w, path, "conversion source already exists")
		return true
	}
	currency := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantConversionsString(mcd, "currencyCode", "currency_code")))
	if !gcpShoppingMerchantConversionsCurrencyRe.MatchString(currency) {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "merchantCenterDestination.currencyCode must be an ISO 4217 code")
		return true
	}
	attr := gcpShoppingMerchantConversionsMap(mcd, "attributionSettings", "attribution_settings")
	if len(attr) == 0 {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "merchantCenterDestination.attributionSettings is required")
		return true
	}
	lookbackRaw := strings.TrimSpace(gcpShoppingMerchantConversionsString(attr, "attributionLookbackWindowDays", "attribution_lookback_window_days"))
	if lookbackRaw == "" {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "attributionLookbackWindowDays is required")
		return true
	}
	lookback, err := strconv.Atoi(lookbackRaw)
	if err != nil {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "attributionLookbackWindowDays must be an integer")
		return true
	}
	if lookback != 7 && lookback != 30 && lookback != 40 {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "attributionLookbackWindowDays must be one of 7, 30, 40")
		return true
	}
	if strings.TrimSpace(gcpShoppingMerchantConversionsString(attr, "attributionModel", "attribution_model")) == "" {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "attributionModel is required")
		return true
	}
	sourceID := "mcdn:1001"
	respondJSON(w, http.StatusOK, gcpShoppingMerchantConversionsConversionSourceFixture(account, sourceID, "ACTIVE", displayName, currency))
	return true
}

func parseGCPShoppingMerchantConversionsCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "conversionSources" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantConversionsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantConversionsResourcePath(tail string) (account, sourceID, action string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(tail), "/")
	action = ""
	if strings.HasSuffix(trimmed, ":undelete") {
		action = "undelete"
		trimmed = strings.TrimSuffix(trimmed, ":undelete")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "conversionSources" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	sourceID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantConversionsAccountRe.MatchString(account) || !gcpShoppingMerchantConversionsSourceIDRe.MatchString(sourceID) {
		return "", "", "", false
	}
	return account, sourceID, action, true
}

func parseGCPShoppingMerchantConversionsListParams(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, showDeleted, ok bool) {
	pageSize = 100
	pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false, false
		}
		pageSize = parsed
	}
	if pageSize > 200 {
		pageSize = 200
	}
	pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false, false
		}
		start = parsed
	}
	showDeletedRaw := strings.TrimSpace(r.URL.Query().Get("showDeleted"))
	showDeleted = false
	if showDeletedRaw != "" {
		parsed, err := strconv.ParseBool(showDeletedRaw)
		if err != nil {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "showDeleted must be a boolean")
			return 0, 0, false, false
		}
		showDeleted = parsed
	}
	return pageSize, start, showDeleted, true
}

func decodeGCPShoppingMerchantConversionsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPShoppingMerchantConversionsJSONBody(w, r, path, true)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func decodeGCPShoppingMerchantConversionsJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		if required {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		if required {
			respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantConversionsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpShoppingMerchantConversionsMap(m map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			if child, ok := raw.(map[string]any); ok {
				return child
			}
		}
	}
	return map[string]any{}
}

func gcpShoppingMerchantConversionsString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			switch value := raw.(type) {
			case string:
				return value
			case fmt.Stringer:
				return value.String()
			case float64:
				return strconv.FormatInt(int64(value), 10)
			case int64:
				return strconv.FormatInt(value, 10)
			case int:
				return strconv.Itoa(value)
			}
		}
	}
	return ""
}

func gcpShoppingMerchantConversionsConversionSourceFixture(account, sourceID, state, displayName, currency string) map[string]any {
	if state == "" {
		state = "ACTIVE"
	}
	out := map[string]any{
		"name":       fmt.Sprintf("accounts/%s/conversionSources/%s", account, sourceID),
		"state":      state,
		"controller": "MERCHANT",
	}
	switch {
	case strings.HasPrefix(sourceID, "galk:"):
		propertyID := strings.TrimPrefix(sourceID, "galk:")
		if propertyID == "" {
			propertyID = "2001"
		}
		out["googleAnalyticsLink"] = map[string]any{
			"propertyId": propertyID,
			"property":   "properties/" + propertyID,
			"attributionSettings": map[string]any{
				"attributionLookbackWindowDays": 30,
				"attributionModel":              "CROSS_CHANNEL_LAST_CLICK",
				"conversionType": []any{
					map[string]any{"name": "purchase", "report": true},
				},
			},
		}
	default:
		if displayName == "" {
			displayName = "Stackyard Destination"
		}
		if currency == "" {
			currency = "USD"
		}
		out["merchantCenterDestination"] = map[string]any{
			"destination":  fmt.Sprintf("accounts/%s/conversionSources/%s/destination", account, sourceID),
			"displayName":  displayName,
			"currencyCode": strings.ToUpper(currency),
			"attributionSettings": map[string]any{
				"attributionLookbackWindowDays": 30,
				"attributionModel":              "CROSS_CHANNEL_LAST_CLICK",
				"conversionType": []any{
					map[string]any{"name": "purchase", "report": true},
				},
			},
		}
	}
	if state == "ARCHIVED" {
		out["expireTime"] = time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
	}
	return out
}

func respondGCPShoppingMerchantConversionsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantConversionsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantConversionsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantConversionsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantConversions(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_conversions") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/shopping_merchant_conversions/sample",
			"service":  "shopping_merchant_conversions",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
