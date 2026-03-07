package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpShoppingMerchantNotificationsAccountRe        = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantNotificationsSubscriptionIDRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	gcpShoppingMerchantNotificationsEventNameRe      = regexp.MustCompile(`^[A-Za-z_]+$`)
)

func (s *Server) handleGCPShoppingMerchantNotificationsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantNotifications(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantNotificationsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantNotificationsPath(path, hasGCPShoppingMerchantNotificationsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantNotificationsRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.notifications.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantNotificationsGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantNotificationsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPShoppingMerchantNotificationsPATCH(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantNotificationsDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantNotificationsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantNotificationsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_notifications",
		"shopping-merchant-notifications",
		"shopping-merchant-notifications-apiv1",
		"shopping_merchant_notifications_apiv1",
		"merchant_notifications",
		"merchant-notifications",
		"merchantnotifications",
		"gcp-shopping-merchant-notifications":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-notifications-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/notifications")
}

func isGCPShoppingMerchantNotificationsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/notifications/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.notifications.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/notifications/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantNotificationsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/notifications/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/notifications/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantNotificationsGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantNotificationsCollectionPath(tail); ok {
		return handleGCPShoppingMerchantNotificationsList(w, r, path, account)
	}
	if account, subscriptionID, ok := parseGCPShoppingMerchantNotificationsHealthPath(tail); ok {
		return handleGCPShoppingMerchantNotificationsGetHealth(w, path, account, subscriptionID)
	}
	if account, subscriptionID, ok := parseGCPShoppingMerchantNotificationsResourcePath(tail); ok {
		return handleGCPShoppingMerchantNotificationsGet(w, path, account, subscriptionID)
	}
	return false
}

func handleGCPShoppingMerchantNotificationsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantNotificationsCollectionPath(tail)
	if !ok {
		return false
	}
	return handleGCPShoppingMerchantNotificationsCreate(w, r, path, account)
}

func handleGCPShoppingMerchantNotificationsPATCH(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, subscriptionID, ok := parseGCPShoppingMerchantNotificationsResourcePath(tail)
	if !ok {
		return false
	}
	return handleGCPShoppingMerchantNotificationsUpdate(w, r, path, account, subscriptionID)
}

func handleGCPShoppingMerchantNotificationsDELETE(w http.ResponseWriter, _ *http.Request, path, tail string) bool {
	_, subscriptionID, ok := parseGCPShoppingMerchantNotificationsResourcePath(tail)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(subscriptionID), "missing") {
		respondGCPShoppingMerchantNotificationsNotFound(w, path, "notification subscription not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantNotificationsGet(w http.ResponseWriter, path, account, subscriptionID string) bool {
	if strings.Contains(strings.ToLower(subscriptionID), "missing") {
		respondGCPShoppingMerchantNotificationsNotFound(w, path, "notification subscription not found")
		return true
	}
	event := int32(1)
	callback := "https://example.com/hooks/merchant-notifications"
	allManaged := true
	targetAccount := ""
	if strings.Contains(subscriptionID, "target-") {
		allManaged = false
		targetAccount = "accounts/567890"
		callback = "https://example.com/hooks/merchant-notifications-target"
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID, event, callback, allManaged, targetAccount))
	return true
}

func handleGCPShoppingMerchantNotificationsCreate(w http.ResponseWriter, r *http.Request, path, account string) bool {
	body, ok := decodeGCPShoppingMerchantNotificationsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	event, callbackURI, allManaged, targetAccount, subscriptionID, valid := validateGCPShoppingMerchantNotificationsSubscriptionBody(w, path, account, body, false)
	if !valid {
		return true
	}
	if strings.Contains(strings.ToLower(callbackURI), "existing") || strings.Contains(strings.ToLower(subscriptionID), "existing") {
		respondGCPShoppingMerchantNotificationsAlreadyExists(w, path, "notification subscription already exists")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID, event, callbackURI, allManaged, targetAccount))
	return true
}

func handleGCPShoppingMerchantNotificationsUpdate(w http.ResponseWriter, r *http.Request, path, account, subscriptionID string) bool {
	if strings.Contains(strings.ToLower(subscriptionID), "missing") {
		respondGCPShoppingMerchantNotificationsNotFound(w, path, "notification subscription not found")
		return true
	}
	updateMaskFields, ok := parseGCPShoppingMerchantNotificationsUpdateMask(w, r, path)
	if !ok {
		return true
	}
	body, ok := decodeGCPShoppingMerchantNotificationsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	event, callbackURI, allManaged, targetAccount, bodySubscriptionID, valid := validateGCPShoppingMerchantNotificationsSubscriptionBody(w, path, account, body, true)
	if !valid {
		return true
	}
	if bodySubscriptionID != subscriptionID {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "notificationSubscription.name must match requested resource")
		return true
	}
	if !validateGCPShoppingMerchantNotificationsUpdateMaskFields(w, path, updateMaskFields) {
		return true
	}
	// Apply staged update semantics based on requested mask fields.
	for _, field := range updateMaskFields {
		switch field {
		case "call_back_uri":
			if callbackURI == "" {
				respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "callBackUri is required by updateMask")
				return true
			}
		case "registered_event":
			if event <= 0 {
				respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "registeredEvent is required by updateMask")
				return true
			}
		case "all_managed_accounts":
			if !allManaged {
				respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "allManagedAccounts=true is required by updateMask")
				return true
			}
		case "target_account":
			if targetAccount == "" {
				respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "targetAccount is required by updateMask")
				return true
			}
		}
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID, event, callbackURI, allManaged, targetAccount))
	return true
}

func handleGCPShoppingMerchantNotificationsList(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize := 100
	if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return true
		}
		pageSize = parsed
	}
	if pageSize > 200 {
		pageSize = 200
	}
	start := 0
	if pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken")); pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return true
		}
		start = parsed
	}

	items := []map[string]any{
		gcpShoppingMerchantNotificationsSubscriptionFixture(account, "all-managed-product-status-change", 1, "https://example.com/hooks/merchant-notifications", true, ""),
		gcpShoppingMerchantNotificationsSubscriptionFixture(account, "target-567890-product-status-change", 1, "https://example.com/hooks/merchant-notifications-target", false, "accounts/567890"),
	}
	if start > len(items) {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "pageToken is out of range")
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
		"notificationSubscriptions": items[start:end],
		"nextPageToken":             next,
	})
	return true
}

func handleGCPShoppingMerchantNotificationsGetHealth(w http.ResponseWriter, path, account, subscriptionID string) bool {
	if strings.Contains(strings.ToLower(subscriptionID), "missing") {
		respondGCPShoppingMerchantNotificationsNotFound(w, path, "notification subscription not found")
		return true
	}
	name := fmt.Sprintf("accounts/%s/notificationsubscriptions/%s", account, subscriptionID)
	respondJSON(w, http.StatusOK, map[string]any{
		"name":                                  name,
		"acknowledgedMessagesCount":             "42",
		"undeliveredMessagesCount":              "3",
		"oldestUnacknowledgedMessageWaitingTime": "3600",
	})
	return true
}

func parseGCPShoppingMerchantNotificationsCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "notificationsubscriptions" {
		return "", false
	}
	return parseGCPShoppingMerchantNotificationsParent(fmt.Sprintf("accounts/%s", strings.TrimSpace(parts[1])))
}

func parseGCPShoppingMerchantNotificationsResourcePath(tail string) (account, subscriptionID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "notificationsubscriptions" {
		return "", "", false
	}
	name := fmt.Sprintf("accounts/%s/notificationsubscriptions/%s", strings.TrimSpace(parts[1]), strings.TrimSpace(parts[3]))
	return parseGCPShoppingMerchantNotificationsName(name)
}

func parseGCPShoppingMerchantNotificationsHealthPath(tail string) (account, subscriptionID string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(tail), "/")
	if !strings.HasSuffix(trimmed, ":getHealth") {
		return "", "", false
	}
	return parseGCPShoppingMerchantNotificationsResourcePath(strings.TrimSuffix(trimmed, ":getHealth"))
}

func parseGCPShoppingMerchantNotificationsParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantNotificationsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantNotificationsName(name string) (account, subscriptionID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "notificationsubscriptions" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	subscriptionID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantNotificationsAccountRe.MatchString(account) || !gcpShoppingMerchantNotificationsSubscriptionIDRe.MatchString(subscriptionID) {
		return "", "", false
	}
	return account, subscriptionID, true
}

func decodeGCPShoppingMerchantNotificationsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func parseGCPShoppingMerchantNotificationsUpdateMask(w http.ResponseWriter, r *http.Request, path string) ([]string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if raw == "" {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "updateMask is required")
		return nil, false
	}
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		field := normalizeGCPShoppingMerchantNotificationsMaskField(part)
		if field == "" {
			continue
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "updateMask is required")
		return nil, false
	}
	return fields, true
}

func normalizeGCPShoppingMerchantNotificationsMaskField(field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	field = strings.Trim(field, "\"'")
	field = strings.ReplaceAll(field, ".", "_")
	switch strings.ToLower(field) {
	case "callbackuri", "call_back_uri", "callback_uri", "callback":
		return "call_back_uri"
	case "registeredevent", "registered_event":
		return "registered_event"
	case "allmanagedaccounts", "all_managed_accounts":
		return "all_managed_accounts"
	case "targetaccount", "target_account":
		return "target_account"
	default:
		return strings.ToLower(field)
	}
}

func validateGCPShoppingMerchantNotificationsUpdateMaskFields(w http.ResponseWriter, path string, fields []string) bool {
	for _, field := range fields {
		switch field {
		case "call_back_uri", "registered_event", "all_managed_accounts", "target_account":
			continue
		default:
			respondGCPShoppingMerchantNotificationsFailedPrecondition(w, path, "updateMask contains unsupported field")
			return false
		}
	}
	return true
}

func validateGCPShoppingMerchantNotificationsSubscriptionBody(w http.ResponseWriter, path, account string, body map[string]any, requireName bool) (event int32, callbackURI string, allManaged bool, targetAccount string, subscriptionID string, ok bool) {
	if requireName {
		name := strings.TrimSpace(gcpShoppingMerchantNotificationsString(body, "name"))
		if name == "" {
			respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "notificationSubscription.name is required")
			return 0, "", false, "", "", false
		}
		nameAccount, nameSubscriptionID, valid := parseGCPShoppingMerchantNotificationsName(name)
		if !valid || nameAccount != account {
			respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "notificationSubscription.name must match parent account")
			return 0, "", false, "", "", false
		}
		subscriptionID = nameSubscriptionID
	}

	parsedEvent, validEvent := parseGCPShoppingMerchantNotificationsEvent(body)
	if !validEvent || parsedEvent <= 0 {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "registeredEvent is required and must be valid")
		return 0, "", false, "", "", false
	}
	event = parsedEvent

	callbackURI = strings.TrimSpace(gcpShoppingMerchantNotificationsString(body, "callBackUri", "call_back_uri"))
	if !isGCPShoppingMerchantNotificationsCallbackURI(callbackURI) {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "callBackUri must be a valid http(s) URL")
		return 0, "", false, "", "", false
	}

	allManagedPresent := gcpShoppingMerchantNotificationsHasAny(body, "allManagedAccounts", "all_managed_accounts")
	allManaged = gcpShoppingMerchantNotificationsBool(body, "allManagedAccounts", "all_managed_accounts")
	targetAccount = strings.TrimSpace(gcpShoppingMerchantNotificationsString(body, "targetAccount", "target_account"))
	targetPresent := targetAccount != ""

	if allManagedPresent && targetPresent {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "exactly one interestedIn variant is allowed")
		return 0, "", false, "", "", false
	}
	if !allManagedPresent && !targetPresent {
		respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "one interestedIn variant is required")
		return 0, "", false, "", "", false
	}
	if allManagedPresent {
		if !allManaged {
			respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "allManagedAccounts must be true when provided")
			return 0, "", false, "", "", false
		}
	} else {
		if _, ok := parseGCPShoppingMerchantNotificationsParent(targetAccount); !ok {
			respondGCPShoppingMerchantNotificationsInvalidArgument(w, path, "targetAccount must be in accounts/{account} format")
			return 0, "", false, "", "", false
		}
	}

	if !requireName {
		subscriptionID = gcpShoppingMerchantNotificationsSubscriptionIDFor(event, allManaged, targetAccount)
	}
	return event, callbackURI, allManaged, targetAccount, subscriptionID, true
}

func parseGCPShoppingMerchantNotificationsEvent(body map[string]any) (int32, bool) {
	raw, ok := gcpShoppingMerchantNotificationsAny(body, "registeredEvent", "registered_event")
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		if v != float64(int32(v)) {
			return 0, false
		}
		return int32(v), true
	case int:
		return int32(v), true
	case int64:
		return int32(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		if gcpShoppingMerchantNotificationsEventNameRe.MatchString(trimmed) {
			if strings.EqualFold(trimmed, "product_status_change") || strings.EqualFold(trimmed, "PRODUCT_STATUS_CHANGE") {
				return 1, true
			}
			if strings.EqualFold(trimmed, "notification_event_type_unspecified") {
				return 0, true
			}
		}
		parsed, err := strconv.ParseInt(trimmed, 10, 32)
		if err != nil {
			return 0, false
		}
		return int32(parsed), true
	default:
		return 0, false
	}
}

func isGCPShoppingMerchantNotificationsCallbackURI(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return false
	}
	return strings.TrimSpace(parsed.Host) != ""
}

func gcpShoppingMerchantNotificationsSubscriptionIDFor(event int32, allManaged bool, targetAccount string) string {
	eventSuffix := "product-status-change"
	if event <= 0 {
		eventSuffix = "unspecified"
	}
	if allManaged {
		return "all-managed-" + eventSuffix
	}
	targetID := "target"
	if account, ok := parseGCPShoppingMerchantNotificationsParent(targetAccount); ok {
		targetID = account
	}
	return "target-" + targetID + "-" + eventSuffix
}

func gcpShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID string, event int32, callbackURI string, allManaged bool, targetAccount string) map[string]any {
	out := map[string]any{
		"name":          fmt.Sprintf("accounts/%s/notificationsubscriptions/%s", account, subscriptionID),
		"registeredEvent": event,
		"callBackUri":   callbackURI,
	}
	if allManaged {
		out["allManagedAccounts"] = true
	} else {
		out["targetAccount"] = targetAccount
	}
	return out
}

func gcpShoppingMerchantNotificationsString(m map[string]any, keys ...string) string {
	raw, ok := gcpShoppingMerchantNotificationsAny(m, keys...)
	if !ok {
		return ""
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
	default:
		return ""
	}
}

func gcpShoppingMerchantNotificationsBool(m map[string]any, keys ...string) bool {
	raw, ok := gcpShoppingMerchantNotificationsAny(m, keys...)
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func gcpShoppingMerchantNotificationsHasAny(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func gcpShoppingMerchantNotificationsAny(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func respondGCPShoppingMerchantNotificationsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantNotificationsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantNotificationsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantNotificationsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantNotifications(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_notifications") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_notifications/sample",
			"service":   "shopping_merchant_notifications",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
