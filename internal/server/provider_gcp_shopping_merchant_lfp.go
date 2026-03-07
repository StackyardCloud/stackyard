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
	gcpShoppingMerchantLFPAccountRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantLFPTargetRe     = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantLFPStoreCodeRe  = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	gcpShoppingMerchantLFPOfferIDRe    = regexp.MustCompile(`^[A-Za-z0-9._~:-]+$`)
	gcpShoppingMerchantLFPRegionCodeRe = regexp.MustCompile(`^[A-Za-z]{2}$`)
	gcpShoppingMerchantLFPLanguageRe   = regexp.MustCompile(`^[A-Za-z]{2,3}([_-][A-Za-z0-9]{2,8})*$`)
	gcpShoppingMerchantLFPCurrencyCode = regexp.MustCompile(`^[A-Z]{3}$`)
	gcpShoppingMerchantLFPGTINRe       = regexp.MustCompile(`^[A-Za-z0-9._-]{4,64}$`)
)

func (s *Server) handleGCPShoppingMerchantLFPRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantLfp(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantLFPPath(rawRequestPath(r))
	if !isGCPShoppingMerchantLFPPath(path, hasGCPShoppingMerchantLFPHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantLFPRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.lfp.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantLFPGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantLFPPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantLFPDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantLFPPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantLFPHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_lfp",
		"shopping-merchant-lfp",
		"shopping-merchant-lfp-apiv1",
		"shopping_merchant_lfp_apiv1",
		"merchant_lfp",
		"merchant-lfp",
		"merchantlfp",
		"gcp-shopping-merchant-lfp":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-lfp-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/lfp")
}

func isGCPShoppingMerchantLFPPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/lfp/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.lfp.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/lfp/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantLFPRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/lfp/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/lfp/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantLFPGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, target, storeCode, ok := parseGCPShoppingMerchantLFPStoreNamePath(tail); ok {
		return handleGCPShoppingMerchantLFPGetStore(w, path, account, target, storeCode)
	}
	if account, ok := parseGCPShoppingMerchantLFPStoreCollectionPath(tail); ok {
		return handleGCPShoppingMerchantLFPListStores(w, r, path, account)
	}
	if account, target, ok := parseGCPShoppingMerchantLFPMerchantStateNamePath(tail); ok {
		return handleGCPShoppingMerchantLFPGetMerchantState(w, path, account, target)
	}
	return false
}

func handleGCPShoppingMerchantLFPPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantLFPStoreInsertPath(tail); ok {
		return handleGCPShoppingMerchantLFPInsertStore(w, r, path, account)
	}
	if account, ok := parseGCPShoppingMerchantLFPInventoryInsertPath(tail); ok {
		return handleGCPShoppingMerchantLFPInsertInventory(w, r, path, account)
	}
	if account, ok := parseGCPShoppingMerchantLFPSaleInsertPath(tail); ok {
		return handleGCPShoppingMerchantLFPInsertSale(w, r, path, account)
	}
	return false
}

func handleGCPShoppingMerchantLFPDELETE(w http.ResponseWriter, _ *http.Request, path, tail string) bool {
	_, target, storeCode, ok := parseGCPShoppingMerchantLFPStoreNamePath(tail)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(target), "missing") || strings.Contains(strings.ToLower(storeCode), "missing") {
		respondGCPShoppingMerchantLFPNotFound(w, path, "lfp store not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantLFPGetStore(w http.ResponseWriter, path, account, target, storeCode string) bool {
	if strings.Contains(strings.ToLower(target), "missing") || strings.Contains(strings.ToLower(storeCode), "missing") {
		respondGCPShoppingMerchantLFPNotFound(w, path, "lfp store not found")
		return true
	}
	targetAccount := gcpShoppingMerchantLFPTargetID(target, 567890)
	respondJSON(w, http.StatusOK, gcpShoppingMerchantLFPStoreFixture(account, targetAccount, storeCode))
	return true
}

func handleGCPShoppingMerchantLFPListStores(w http.ResponseWriter, r *http.Request, path, account string) bool {
	targetAccountRaw := strings.TrimSpace(r.URL.Query().Get("targetAccount"))
	if targetAccountRaw == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "targetAccount is required")
		return true
	}
	targetAccount, err := strconv.ParseInt(targetAccountRaw, 10, 64)
	if err != nil || targetAccount <= 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "targetAccount must be a positive integer")
		return true
	}

	pageSize := 250
	if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return true
		}
		pageSize = parsed
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	start := 0
	if pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken")); pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return true
		}
		start = parsed
	}

	items := []map[string]any{
		gcpShoppingMerchantLFPStoreFixture(account, targetAccount, "store-nyc"),
		gcpShoppingMerchantLFPStoreFixture(account, targetAccount, "store-sfo"),
		gcpShoppingMerchantLFPStoreFixture(account, targetAccount, "store-bos"),
	}
	if start > len(items) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "pageToken is out of range")
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
		"lfpStores":     items[start:end],
		"nextPageToken": next,
	})
	return true
}

func handleGCPShoppingMerchantLFPInsertStore(w http.ResponseWriter, r *http.Request, path, parentAccount string) bool {
	body, ok := decodeGCPShoppingMerchantLFPJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	targetAccount, hasTarget, validTarget := gcpShoppingMerchantLFPInt64Any(body, "targetAccount", "target_account")
	if !hasTarget || !validTarget || targetAccount <= 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "targetAccount is required and must be a positive integer")
		return true
	}
	storeCode := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "storeCode", "store_code"))
	if storeCode == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeCode is required")
		return true
	}
	if !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeCode is invalid")
		return true
	}
	storeAddress := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "storeAddress", "store_address"))
	if storeAddress == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeAddress is required")
		return true
	}
	if name := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "name")); name != "" {
		expected := fmt.Sprintf("accounts/%s/lfpStores/%d~%s", parentAccount, targetAccount, storeCode)
		if name != expected {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "name must match parent, targetAccount, and storeCode")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantLFPStoreFixture(parentAccount, targetAccount, storeCode))
	return true
}

func handleGCPShoppingMerchantLFPInsertInventory(w http.ResponseWriter, r *http.Request, path, parentAccount string) bool {
	body, ok := decodeGCPShoppingMerchantLFPJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	targetAccount, hasTarget, validTarget := gcpShoppingMerchantLFPInt64Any(body, "targetAccount", "target_account")
	if !hasTarget || !validTarget || targetAccount <= 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "targetAccount is required and must be a positive integer")
		return true
	}
	storeCode := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "storeCode", "store_code"))
	if storeCode == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeCode is required")
		return true
	}
	if !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeCode is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(storeCode), "missing") || strings.Contains(strings.ToLower(storeCode), "unknown") {
		respondGCPShoppingMerchantLFPFailedPrecondition(w, path, "inventory requires an existing store code")
		return true
	}

	offerID := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "offerId", "offer_id"))
	if offerID == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "offerId is required")
		return true
	}
	if !gcpShoppingMerchantLFPOfferIDRe.MatchString(offerID) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "offerId is invalid")
		return true
	}

	regionCode := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantLFPString(body, "regionCode", "region_code")))
	if !gcpShoppingMerchantLFPRegionCodeRe.MatchString(regionCode) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "regionCode must be a 2-letter territory code")
		return true
	}
	contentLanguage := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "contentLanguage", "content_language"))
	if !gcpShoppingMerchantLFPLanguageRe.MatchString(contentLanguage) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "contentLanguage is required and must be a valid language code")
		return true
	}
	availability := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "availability"))
	if availability == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "availability is required")
		return true
	}

	if qty, hasQty, validQty := gcpShoppingMerchantLFPInt64Any(body, "quantity"); hasQty {
		if !validQty || qty < 0 {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "quantity must be a non-negative integer")
			return true
		}
	}
	if hasPickupMethod := gcpShoppingMerchantLFPHasAny(body, "pickupMethod", "pickup_method"); hasPickupMethod != gcpShoppingMerchantLFPHasAny(body, "pickupSla", "pickup_sla") {
		respondGCPShoppingMerchantLFPFailedPrecondition(w, path, "pickupMethod and pickupSla must be provided together")
		return true
	}
	if price, hasPrice := gcpShoppingMerchantLFPMap(body, "price"); hasPrice {
		currency := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantLFPString(price, "currencyCode", "currency_code")))
		if !gcpShoppingMerchantLFPCurrencyCode.MatchString(currency) {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "price.currencyCode must be an ISO 4217 currency code")
			return true
		}
		if amount, hasAmount, validAmount := gcpShoppingMerchantLFPInt64Any(price, "amountMicros", "amount_micros"); !hasAmount || !validAmount || amount < 0 {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "price.amountMicros must be a non-negative integer")
			return true
		}
	}
	if name := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "name")); name != "" {
		expected := fmt.Sprintf("accounts/%s/lfpInventories/%d~%s~%s", parentAccount, targetAccount, storeCode, offerID)
		if name != expected {
			respondGCPShoppingMerchantLFPInvalidArgument(w, path, "name must match parent, targetAccount, storeCode, and offerId")
			return true
		}
	}

	respondJSON(w, http.StatusOK, gcpShoppingMerchantLFPInventoryFixture(parentAccount, targetAccount, storeCode, offerID, regionCode, contentLanguage, availability))
	return true
}

func handleGCPShoppingMerchantLFPInsertSale(w http.ResponseWriter, r *http.Request, path, parentAccount string) bool {
	body, ok := decodeGCPShoppingMerchantLFPJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	targetAccount, hasTarget, validTarget := gcpShoppingMerchantLFPInt64Any(body, "targetAccount", "target_account")
	if !hasTarget || !validTarget || targetAccount <= 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "targetAccount is required and must be a positive integer")
		return true
	}
	storeCode := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "storeCode", "store_code"))
	if storeCode == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeCode is required")
		return true
	}
	if !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "storeCode is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(storeCode), "missing") || strings.Contains(strings.ToLower(storeCode), "unknown") {
		respondGCPShoppingMerchantLFPFailedPrecondition(w, path, "sale requires an existing store code")
		return true
	}

	offerID := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "offerId", "offer_id"))
	if offerID == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "offerId is required")
		return true
	}
	if !gcpShoppingMerchantLFPOfferIDRe.MatchString(offerID) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "offerId is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(offerID), "duplicate") {
		respondGCPShoppingMerchantLFPAlreadyExists(w, path, "sale already exists")
		return true
	}

	regionCode := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantLFPString(body, "regionCode", "region_code")))
	if !gcpShoppingMerchantLFPRegionCodeRe.MatchString(regionCode) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "regionCode must be a 2-letter territory code")
		return true
	}
	contentLanguage := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "contentLanguage", "content_language"))
	if !gcpShoppingMerchantLFPLanguageRe.MatchString(contentLanguage) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "contentLanguage is required and must be a valid language code")
		return true
	}
	gtin := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "gtin"))
	if !gcpShoppingMerchantLFPGTINRe.MatchString(gtin) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "gtin is required and must be alphanumeric")
		return true
	}
	if _, hasQty, validQty := gcpShoppingMerchantLFPInt64Any(body, "quantity"); !hasQty || !validQty {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "quantity is required and must be an integer")
		return true
	}
	price, hasPrice := gcpShoppingMerchantLFPMap(body, "price")
	if !hasPrice {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "price is required")
		return true
	}
	currency := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantLFPString(price, "currencyCode", "currency_code")))
	if !gcpShoppingMerchantLFPCurrencyCode.MatchString(currency) {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "price.currencyCode must be an ISO 4217 currency code")
		return true
	}
	if amount, hasAmount, validAmount := gcpShoppingMerchantLFPInt64Any(price, "amountMicros", "amount_micros"); !hasAmount || !validAmount || amount <= 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "price.amountMicros must be a positive integer")
		return true
	}

	saleTime := strings.TrimSpace(gcpShoppingMerchantLFPString(body, "saleTime", "sale_time"))
	if saleTime == "" {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "saleTime is required")
		return true
	}
	if _, err := time.Parse(time.RFC3339, saleTime); err != nil {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "saleTime must be a valid RFC3339 timestamp")
		return true
	}

	respondJSON(w, http.StatusOK, gcpShoppingMerchantLFPSaleFixture(parentAccount, targetAccount, storeCode, offerID, regionCode, contentLanguage, gtin, saleTime))
	return true
}

func handleGCPShoppingMerchantLFPGetMerchantState(w http.ResponseWriter, path, account, target string) bool {
	if strings.Contains(strings.ToLower(target), "missing") {
		respondGCPShoppingMerchantLFPNotFound(w, path, "lfp merchant state not found")
		return true
	}
	targetAccount := gcpShoppingMerchantLFPTargetID(target, 567890)
	respondJSON(w, http.StatusOK, gcpShoppingMerchantLFPMerchantStateFixture(account, targetAccount))
	return true
}

func parseGCPShoppingMerchantLFPParentPath(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantLFPAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantLFPStoreCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "lfpStores" {
		return "", false
	}
	return parseGCPShoppingMerchantLFPParentPath(strings.Join(parts[:2], "/"))
}

func parseGCPShoppingMerchantLFPStoreInsertPath(tail string) (account string, ok bool) {
	if !strings.HasSuffix(tail, "/lfpStores:insert") {
		return "", false
	}
	return parseGCPShoppingMerchantLFPStoreCollectionPath(strings.TrimSuffix(tail, ":insert"))
}

func parseGCPShoppingMerchantLFPInventoryInsertPath(tail string) (account string, ok bool) {
	if !strings.HasSuffix(tail, "/lfpInventories:insert") {
		return "", false
	}
	parent := strings.TrimSuffix(tail, "/lfpInventories:insert")
	return parseGCPShoppingMerchantLFPParentPath(parent)
}

func parseGCPShoppingMerchantLFPSaleInsertPath(tail string) (account string, ok bool) {
	if !strings.HasSuffix(tail, "/lfpSales:insert") {
		return "", false
	}
	parent := strings.TrimSuffix(tail, "/lfpSales:insert")
	return parseGCPShoppingMerchantLFPParentPath(parent)
}

func parseGCPShoppingMerchantLFPStoreNamePath(tail string) (account, target, storeCode string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "lfpStores" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	key := strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantLFPAccountRe.MatchString(account) {
		return "", "", "", false
	}
	target, storeCode, ok = parseGCPShoppingMerchantLFPStoreKey(key)
	if !ok {
		return "", "", "", false
	}
	return account, target, storeCode, true
}

func parseGCPShoppingMerchantLFPMerchantStateNamePath(tail string) (account, target string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "lfpMerchantStates" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	target = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantLFPAccountRe.MatchString(account) || !gcpShoppingMerchantLFPTargetRe.MatchString(target) {
		return "", "", false
	}
	return account, target, true
}

func parseGCPShoppingMerchantLFPStoreKey(value string) (target, storeCode string, ok bool) {
	parts := strings.Split(strings.TrimSpace(value), "~")
	if len(parts) != 2 {
		return "", "", false
	}
	target = strings.TrimSpace(parts[0])
	storeCode = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantLFPTargetRe.MatchString(target) || !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
		return "", "", false
	}
	return target, storeCode, true
}

func decodeGCPShoppingMerchantLFPJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantLFPInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpShoppingMerchantLFPStoreFixture(account string, targetAccount int64, storeCode string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("accounts/%s/lfpStores/%d~%s", account, targetAccount, storeCode),
		"targetAccount": strconv.FormatInt(targetAccount, 10),
		"storeCode":     storeCode,
		"storeAddress":  "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
		"storeName":     "Stackyard Downtown",
		"phoneNumber":   "+15555550123",
		"websiteUri":    "https://example.com/store/" + storeCode,
		"gcidCategory":  []string{"gcid:department_store"},
		"placeId":       "ChIJ2eUgeAK6j4ARbn5u_wAGqWA",
		"matchingState": 2,
	}
}

func gcpShoppingMerchantLFPInventoryFixture(account string, targetAccount int64, storeCode, offerID, regionCode, contentLanguage, availability string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("accounts/%s/lfpInventories/%d~%s~%s", account, targetAccount, storeCode, offerID),
		"targetAccount":   strconv.FormatInt(targetAccount, 10),
		"storeCode":       storeCode,
		"offerId":         offerID,
		"regionCode":      strings.ToUpper(regionCode),
		"contentLanguage": contentLanguage,
		"availability":    availability,
		"price": map[string]any{
			"currencyCode": "USD",
			"amountMicros": "12990000",
		},
		"quantity":       "7",
		"collectionTime": time.Date(2026, time.January, 1, 15, 4, 5, 0, time.UTC).Format(time.RFC3339),
		"pickupMethod":   "buy",
		"pickupSla":      "same day",
		"feedLabel":      strings.ToUpper(regionCode),
	}
}

func gcpShoppingMerchantLFPSaleFixture(account string, targetAccount int64, storeCode, offerID, regionCode, contentLanguage, gtin, saleTime string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("accounts/%s/lfpSales/%d~%s~%s", account, targetAccount, storeCode, offerID),
		"targetAccount":   strconv.FormatInt(targetAccount, 10),
		"storeCode":       storeCode,
		"offerId":         offerID,
		"regionCode":      strings.ToUpper(regionCode),
		"contentLanguage": contentLanguage,
		"gtin":            gtin,
		"price": map[string]any{
			"currencyCode": "USD",
			"amountMicros": "14990000",
		},
		"quantity":  "1",
		"saleTime":  saleTime,
		"uid":       fmt.Sprintf("uid-%d-%s-%s", targetAccount, storeCode, offerID),
		"feedLabel": strings.ToUpper(regionCode),
	}
}

func gcpShoppingMerchantLFPMerchantStateFixture(account string, targetAccount int64) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("accounts/%s/lfpMerchantStates/%d", account, targetAccount),
		"linkedGbps": "2",
		"storeStates": []map[string]any{
			{
				"storeCode":     "store-nyc",
				"matchingState": 2,
			},
			{
				"storeCode":     "store-sfo",
				"matchingState": 2,
			},
		},
		"inventoryStats": map[string]any{
			"submittedEntries":        "42",
			"submittedInStockEntries": "37",
			"unsubmittedEntries":      "5",
			"submittedProducts":       "15",
		},
		"countrySettings": []map[string]any{
			{
				"regionCode":                      "US",
				"freeLocalListingsEnabled":        true,
				"localInventoryAdsEnabled":        true,
				"inventoryVerificationState":      2,
				"productPageType":                 1,
				"instockServingVerificationState": 2,
				"pickupServingVerificationState":  2,
			},
		},
	}
}

func gcpShoppingMerchantLFPString(m map[string]any, keys ...string) string {
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

func gcpShoppingMerchantLFPInt64Any(m map[string]any, keys ...string) (value int64, has bool, valid bool) {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok {
			continue
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
	return 0, false, true
}

func gcpShoppingMerchantLFPMap(m map[string]any, key string) (map[string]any, bool) {
	raw, ok := m[key]
	if !ok {
		return nil, false
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	return obj, true
}

func gcpShoppingMerchantLFPHasAny(m map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			return true
		}
	}
	return false
}

func gcpShoppingMerchantLFPTargetID(target string, fallback int64) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(target), 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func respondGCPShoppingMerchantLFPInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantLFPAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantLFPNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantLFPFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantLfp(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_lfp") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_lfp/sample",
			"service":   "shopping_merchant_lfp",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
