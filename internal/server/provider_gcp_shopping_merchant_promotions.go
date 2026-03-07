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
	gcpShoppingMerchantPromotionsAccountRe   = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantPromotionsLanguageRe  = regexp.MustCompile(`^[A-Za-z]{2}$`)
	gcpShoppingMerchantPromotionsCountryRe   = regexp.MustCompile(`^[A-Za-z]{2}$`)
	gcpShoppingMerchantPromotionsPromotionRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	gcpShoppingMerchantPromotionsDataSrcRe   = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

var (
	gcpShoppingMerchantPromotionsProductApplicabilityByNumber = map[int64]string{
		1: "ALL_PRODUCTS",
		2: "SPECIFIC_PRODUCTS",
	}
	gcpShoppingMerchantPromotionsProductApplicabilityAliases = map[string]string{
		"ALL_PRODUCTS":      "ALL_PRODUCTS",
		"SPECIFIC_PRODUCTS": "SPECIFIC_PRODUCTS",
	}
	gcpShoppingMerchantPromotionsOfferTypeByNumber = map[int64]string{
		1: "NO_CODE",
		2: "GENERIC_CODE",
	}
	gcpShoppingMerchantPromotionsOfferTypeAliases = map[string]string{
		"NO_CODE":      "NO_CODE",
		"GENERIC_CODE": "GENERIC_CODE",
	}
	gcpShoppingMerchantPromotionsCouponValueTypeByNumber = map[int64]string{
		1:  "MONEY_OFF",
		2:  "PERCENT_OFF",
		3:  "BUY_M_GET_N_MONEY_OFF",
		4:  "BUY_M_GET_N_PERCENT_OFF",
		5:  "BUY_M_GET_MONEY_OFF",
		6:  "BUY_M_GET_PERCENT_OFF",
		7:  "FREE_GIFT",
		8:  "FREE_GIFT_WITH_VALUE",
		9:  "FREE_GIFT_WITH_ITEM_ID",
		10: "FREE_SHIPPING_STANDARD",
		11: "FREE_SHIPPING_OVERNIGHT",
		12: "FREE_SHIPPING_TWO_DAY",
	}
	gcpShoppingMerchantPromotionsCouponValueTypeAliases = map[string]string{
		"MONEY_OFF":                "MONEY_OFF",
		"PERCENT_OFF":              "PERCENT_OFF",
		"BUY_M_GET_N_MONEY_OFF":    "BUY_M_GET_N_MONEY_OFF",
		"BUY_M_GET_N_PERCENT_OFF":  "BUY_M_GET_N_PERCENT_OFF",
		"BUY_M_GET_MONEY_OFF":      "BUY_M_GET_MONEY_OFF",
		"BUY_M_GET_PERCENT_OFF":    "BUY_M_GET_PERCENT_OFF",
		"FREE_GIFT":                "FREE_GIFT",
		"FREE_GIFT_WITH_VALUE":     "FREE_GIFT_WITH_VALUE",
		"FREE_GIFT_WITH_ITEM_ID":   "FREE_GIFT_WITH_ITEM_ID",
		"FREE_SHIPPING_STANDARD":   "FREE_SHIPPING_STANDARD",
		"FREE_SHIPPING_OVERNIGHT":  "FREE_SHIPPING_OVERNIGHT",
		"FREE_SHIPPING_TWO_DAY":    "FREE_SHIPPING_TWO_DAY",
		"FREESHIPPING_STANDARD":    "FREE_SHIPPING_STANDARD",
		"FREESHIPPING_OVERNIGHT":   "FREE_SHIPPING_OVERNIGHT",
		"FREESHIPPING_TWO_DAY":     "FREE_SHIPPING_TWO_DAY",
		"BUYM_GET_N_MONEY_OFF":     "BUY_M_GET_N_MONEY_OFF",
		"BUYM_GET_N_PERCENT_OFF":   "BUY_M_GET_N_PERCENT_OFF",
		"BUYM_GET_MONEY_OFF":       "BUY_M_GET_MONEY_OFF",
		"BUYM_GET_PERCENT_OFF":     "BUY_M_GET_PERCENT_OFF",
		"FREEGIFT":                 "FREE_GIFT",
		"FREEGIFT_WITH_VALUE":      "FREE_GIFT_WITH_VALUE",
		"FREEGIFT_WITH_ITEM_ID":    "FREE_GIFT_WITH_ITEM_ID",
		"FREE_SHIPPING":            "FREE_SHIPPING_STANDARD",
		"FREE_SHIPPING_EXPEDITED":  "FREE_SHIPPING_OVERNIGHT",
		"FREE_SHIPPING_EXPRESS":    "FREE_SHIPPING_OVERNIGHT",
		"FREE_SHIPPING_PRIORITY":   "FREE_SHIPPING_TWO_DAY",
		"FREE_SHIPPING_2DAY":       "FREE_SHIPPING_TWO_DAY",
		"FREE_SHIPPING_TWO-DAY":    "FREE_SHIPPING_TWO_DAY",
		"FREE_SHIPPING_OVERNIGHTS": "FREE_SHIPPING_OVERNIGHT",
	}
	gcpShoppingMerchantPromotionsRedemptionChannelByNumber = map[int64]string{
		1: "IN_STORE",
		2: "ONLINE",
	}
	gcpShoppingMerchantPromotionsRedemptionChannelAliases = map[string]string{
		"IN_STORE": "IN_STORE",
		"ONLINE":   "ONLINE",
		"INSTORE":  "IN_STORE",
	}
	gcpShoppingMerchantPromotionsStoreApplicabilityByNumber = map[int64]string{
		1: "ALL_STORES",
		2: "SPECIFIC_STORES",
	}
	gcpShoppingMerchantPromotionsStoreApplicabilityAliases = map[string]string{
		"ALL_STORES":      "ALL_STORES",
		"SPECIFIC_STORES": "SPECIFIC_STORES",
	}
)

func (s *Server) handleGCPShoppingMerchantPromotionsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantPromotions(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantPromotionsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantPromotionsPath(path, hasGCPShoppingMerchantPromotionsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantPromotionsRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.promotions.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantPromotionsGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantPromotionsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantPromotionsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantPromotionsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_promotions",
		"shopping-merchant-promotions",
		"shopping-merchant-promotions-apiv1",
		"shopping_merchant_promotions_apiv1",
		"merchant_promotions",
		"merchant-promotions",
		"merchantpromotions",
		"gcp-shopping-merchant-promotions":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-promotions-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/promotions")
}

func isGCPShoppingMerchantPromotionsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/promotions/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.promotions.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/promotions/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantPromotionsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/promotions/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/promotions/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantPromotionsGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantPromotionsCollectionPath(tail); ok {
		return handleGCPShoppingMerchantPromotionsList(w, r, path, account)
	}
	account, promotionToken, ok := parseGCPShoppingMerchantPromotionsNamePath(tail)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(promotionToken), "missing") {
		respondGCPShoppingMerchantPromotionsNotFound(w, path, "promotion not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantPromotionsPromotionFixture(account, promotionToken, fmt.Sprintf("accounts/%s/dataSources/104628", account)))
	return true
}

func handleGCPShoppingMerchantPromotionsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantPromotionsInsertPath(tail)
	if !ok {
		return false
	}
	body, ok := decodeGCPShoppingMerchantPromotionsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	parent := strings.TrimSpace(gcpShoppingMerchantPromotionsString(body, "parent"))
	parentAccount, ok := parseGCPShoppingMerchantPromotionsParent(parent)
	if !ok {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "parent must be in accounts/{account} format")
		return true
	}
	if parentAccount != account {
		respondGCPShoppingMerchantPromotionsFailedPrecondition(w, path, "request parent account must match URL account")
		return true
	}

	dataSource, errType, errMessage := parseGCPShoppingMerchantPromotionsDataSource(account, gcpShoppingMerchantPromotionsString(body, "dataSource", "data_source"))
	if errType != "" {
		respondGCPShoppingMerchantPromotionsByErrorType(w, path, errType, errMessage)
		return true
	}

	promotion := gcpShoppingMerchantPromotionsMap(body, "promotion")
	fixture, fixtureErrType, fixtureErrMessage := buildGCPShoppingMerchantPromotionsPromotionFixture(account, dataSource, promotion)
	if fixtureErrType != "" {
		respondGCPShoppingMerchantPromotionsByErrorType(w, path, fixtureErrType, fixtureErrMessage)
		return true
	}

	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPShoppingMerchantPromotionsList(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantPromotionsPagination(w, r, path)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantPromotionsPromotionFixture(account, "en~US~promo-1001", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		gcpShoppingMerchantPromotionsPromotionFixture(account, "en~US~promo-1002", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		gcpShoppingMerchantPromotionsPromotionFixture(account, "en~CA~promo-1003", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
	}
	if start > len(items) {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "pageToken is out of range")
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
		"promotions":    items[start:end],
		"nextPageToken": next,
	})
	return true
}

func parseGCPShoppingMerchantPromotionsParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantPromotionsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantPromotionsCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "promotions" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantPromotionsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantPromotionsNamePath(tail string) (account, promotionToken string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "promotions" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	promotionToken = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantPromotionsAccountRe.MatchString(account) {
		return "", "", false
	}
	if _, _, _, valid := parseGCPShoppingMerchantPromotionsToken(promotionToken); !valid {
		return "", "", false
	}
	return account, promotionToken, true
}

func parseGCPShoppingMerchantPromotionsInsertPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "promotions:insert" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantPromotionsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantPromotionsName(name string) (account, promotionToken string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "promotions" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	promotionToken = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantPromotionsAccountRe.MatchString(account) {
		return "", "", false
	}
	if _, _, _, valid := parseGCPShoppingMerchantPromotionsToken(promotionToken); !valid {
		return "", "", false
	}
	return account, promotionToken, true
}

func parseGCPShoppingMerchantPromotionsToken(token string) (contentLanguage, targetCountry, promotionID string, ok bool) {
	parts := strings.Split(strings.TrimSpace(token), "~")
	if len(parts) < 3 {
		return "", "", "", false
	}
	contentLanguage = strings.TrimSpace(parts[0])
	targetCountry = strings.TrimSpace(parts[1])
	promotionID = strings.Join(parts[2:], "~")
	if !gcpShoppingMerchantPromotionsLanguageRe.MatchString(contentLanguage) {
		return "", "", "", false
	}
	if !gcpShoppingMerchantPromotionsCountryRe.MatchString(targetCountry) {
		return "", "", "", false
	}
	if !gcpShoppingMerchantPromotionsPromotionRe.MatchString(promotionID) {
		return "", "", "", false
	}
	return strings.ToLower(contentLanguage), strings.ToUpper(targetCountry), promotionID, true
}

func buildGCPShoppingMerchantPromotionsToken(contentLanguage, targetCountry, promotionID string) string {
	return strings.ToLower(strings.TrimSpace(contentLanguage)) + "~" + strings.ToUpper(strings.TrimSpace(targetCountry)) + "~" + strings.TrimSpace(promotionID)
}

func parseGCPShoppingMerchantPromotionsDataSource(account, raw string) (dataSource string, errType, errMessage string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "InvalidArgument", "dataSource is required"
	}
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "dataSources" {
		return "", "InvalidArgument", "dataSource must be in accounts/{account}/dataSources/{datasource} format"
	}
	dataSourceAccount := strings.TrimSpace(parts[1])
	dataSourceID := strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantPromotionsAccountRe.MatchString(dataSourceAccount) || !gcpShoppingMerchantPromotionsDataSrcRe.MatchString(dataSourceID) {
		return "", "InvalidArgument", "dataSource must be in accounts/{account}/dataSources/{datasource} format"
	}
	if dataSourceAccount != account {
		return "", "FailedPrecondition", "dataSource account must match request account"
	}
	if strings.Contains(strings.ToLower(dataSourceID), "missing") {
		return "", "NotFound", "data source not found"
	}
	return fmt.Sprintf("accounts/%s/dataSources/%s", dataSourceAccount, dataSourceID), "", ""
}

func parseGCPShoppingMerchantPromotionsPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 50
	if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	if pageSize > 250 {
		pageSize = 250
	}
	if pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken")); pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func decodeGCPShoppingMerchantPromotionsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func buildGCPShoppingMerchantPromotionsPromotionFixture(account, dataSource string, promotion map[string]any) (map[string]any, string, string) {
	if len(promotion) == 0 {
		return nil, "InvalidArgument", "promotion is required"
	}

	rawName := strings.TrimSpace(gcpShoppingMerchantPromotionsString(promotion, "name"))
	var tokenByName string
	if rawName != "" {
		nameAccount, promotionToken, ok := parseGCPShoppingMerchantPromotionsName(rawName)
		if !ok {
			return nil, "InvalidArgument", "promotion.name must be in accounts/{account}/promotions/{promotion} format"
		}
		if nameAccount != account {
			return nil, "FailedPrecondition", "promotion.name account must match request account"
		}
		tokenByName = promotionToken
	}

	promotionID := strings.TrimSpace(gcpShoppingMerchantPromotionsString(promotion, "promotionId", "promotion_id"))
	contentLanguage := strings.TrimSpace(gcpShoppingMerchantPromotionsString(promotion, "contentLanguage", "content_language"))
	targetCountry := strings.TrimSpace(gcpShoppingMerchantPromotionsString(promotion, "targetCountry", "target_country"))
	if promotionID == "" || contentLanguage == "" || targetCountry == "" {
		return nil, "InvalidArgument", "promotionId, contentLanguage, and targetCountry are required"
	}
	if !gcpShoppingMerchantPromotionsPromotionRe.MatchString(promotionID) {
		return nil, "InvalidArgument", "promotionId contains unsupported characters"
	}
	if !gcpShoppingMerchantPromotionsLanguageRe.MatchString(contentLanguage) {
		return nil, "InvalidArgument", "contentLanguage must be a valid ISO 639-1 language code"
	}
	if !gcpShoppingMerchantPromotionsCountryRe.MatchString(targetCountry) {
		return nil, "InvalidArgument", "targetCountry must be a valid CLDR territory code"
	}
	if strings.Contains(strings.ToLower(promotionID), "missing") {
		return nil, "NotFound", "promotion not found"
	}

	redemptionChannels, ok := gcpShoppingMerchantPromotionsParseRedemptionChannels(promotion)
	if !ok || len(redemptionChannels) == 0 {
		return nil, "InvalidArgument", "redemptionChannel must include at least one supported channel"
	}

	attributes := gcpShoppingMerchantPromotionsMap(promotion, "attributes")
	if len(attributes) == 0 {
		return nil, "InvalidArgument", "promotion.attributes is required"
	}
	productApplicability := gcpShoppingMerchantPromotionsParseEnum(
		gcpShoppingMerchantPromotionsAnyValue(attributes, "productApplicability", "product_applicability"),
		gcpShoppingMerchantPromotionsProductApplicabilityAliases,
		gcpShoppingMerchantPromotionsProductApplicabilityByNumber,
	)
	if productApplicability == "" {
		return nil, "InvalidArgument", "attributes.productApplicability is required and must be valid"
	}
	offerType := gcpShoppingMerchantPromotionsParseEnum(
		gcpShoppingMerchantPromotionsAnyValue(attributes, "offerType", "offer_type"),
		gcpShoppingMerchantPromotionsOfferTypeAliases,
		gcpShoppingMerchantPromotionsOfferTypeByNumber,
	)
	if offerType == "" {
		return nil, "InvalidArgument", "attributes.offerType is required and must be valid"
	}
	couponValueType := gcpShoppingMerchantPromotionsParseEnum(
		gcpShoppingMerchantPromotionsAnyValue(attributes, "couponValueType", "coupon_value_type"),
		gcpShoppingMerchantPromotionsCouponValueTypeAliases,
		gcpShoppingMerchantPromotionsCouponValueTypeByNumber,
	)
	if couponValueType == "" {
		return nil, "InvalidArgument", "attributes.couponValueType is required and must be valid"
	}
	longTitle := strings.TrimSpace(gcpShoppingMerchantPromotionsString(attributes, "longTitle", "long_title"))
	if longTitle == "" {
		return nil, "InvalidArgument", "attributes.longTitle is required"
	}
	if offerType == "GENERIC_CODE" {
		if strings.TrimSpace(gcpShoppingMerchantPromotionsString(attributes, "genericRedemptionCode", "generic_redemption_code")) == "" {
			return nil, "InvalidArgument", "attributes.genericRedemptionCode is required for GENERIC_CODE offer type"
		}
	}

	effectivePeriodRaw := gcpShoppingMerchantPromotionsMap(attributes, "promotionEffectiveTimePeriod", "promotion_effective_time_period")
	effectivePeriod, ok := gcpShoppingMerchantPromotionsNormalizeInterval(effectivePeriodRaw)
	if !ok {
		return nil, "InvalidArgument", "attributes.promotionEffectiveTimePeriod is required and must include valid startTime and endTime"
	}
	displayPeriodRaw := gcpShoppingMerchantPromotionsMap(attributes, "promotionDisplayTimePeriod", "promotion_display_time_period")
	displayPeriod := effectivePeriod
	if len(displayPeriodRaw) > 0 {
		var displayOK bool
		displayPeriod, displayOK = gcpShoppingMerchantPromotionsNormalizeInterval(displayPeriodRaw)
		if !displayOK {
			return nil, "InvalidArgument", "attributes.promotionDisplayTimePeriod must include valid startTime and endTime"
		}
	}

	storeApplicability := gcpShoppingMerchantPromotionsParseEnum(
		gcpShoppingMerchantPromotionsAnyValue(attributes, "storeApplicability", "store_applicability"),
		gcpShoppingMerchantPromotionsStoreApplicabilityAliases,
		gcpShoppingMerchantPromotionsStoreApplicabilityByNumber,
	)
	storeCodesInclusion := gcpShoppingMerchantPromotionsStringSlice(attributes, "storeCodesInclusion", "store_codes_inclusion")
	storeCodesExclusion := gcpShoppingMerchantPromotionsStringSlice(attributes, "storeCodesExclusion", "store_codes_exclusion")
	if storeApplicability == "ALL_STORES" && (len(storeCodesInclusion) > 0 || len(storeCodesExclusion) > 0) {
		return nil, "InvalidArgument", "storeCodesInclusion/storeCodesExclusion must be empty when storeApplicability is ALL_STORES"
	}

	if versionNumber := gcpShoppingMerchantPromotionsInt64(promotion, "versionNumber", "version_number"); versionNumber > 0 && versionNumber < 40 {
		return nil, "Aborted", "version_number is stale"
	}

	promotionToken := buildGCPShoppingMerchantPromotionsToken(contentLanguage, targetCountry, promotionID)
	if tokenByName != "" && tokenByName != promotionToken {
		return nil, "FailedPrecondition", "promotion.name must match promotionId/contentLanguage/targetCountry"
	}

	attrs := map[string]any{
		"productApplicability":         productApplicability,
		"offerType":                    offerType,
		"longTitle":                    longTitle,
		"couponValueType":              couponValueType,
		"promotionEffectiveTimePeriod": effectivePeriod,
		"promotionDisplayTimePeriod":   displayPeriod,
	}
	if storeApplicability != "" {
		attrs["storeApplicability"] = storeApplicability
	}
	if len(storeCodesInclusion) > 0 {
		attrs["storeCodesInclusion"] = storeCodesInclusion
	}
	if len(storeCodesExclusion) > 0 {
		attrs["storeCodesExclusion"] = storeCodesExclusion
	}
	if genericCode := strings.TrimSpace(gcpShoppingMerchantPromotionsString(attributes, "genericRedemptionCode", "generic_redemption_code")); genericCode != "" {
		attrs["genericRedemptionCode"] = genericCode
	}

	customAttributes := gcpShoppingMerchantPromotionsCustomAttributes(gcpShoppingMerchantPromotionsSlice(promotion, "customAttributes", "custom_attributes"))
	status := gcpShoppingMerchantPromotionsStatusFixture(strings.ToUpper(strings.TrimSpace(targetCountry)))

	return map[string]any{
		"name":              fmt.Sprintf("accounts/%s/promotions/%s", account, promotionToken),
		"promotionId":       strings.TrimSpace(promotionID),
		"contentLanguage":   strings.ToLower(strings.TrimSpace(contentLanguage)),
		"targetCountry":     strings.ToUpper(strings.TrimSpace(targetCountry)),
		"redemptionChannel": redemptionChannels,
		"dataSource":        dataSource,
		"attributes":        attrs,
		"customAttributes":  customAttributes,
		"promotionStatus":   status,
		"versionNumber":     "42",
	}, "", ""
}

func gcpShoppingMerchantPromotionsPromotionFixture(account, promotionToken, dataSource string) map[string]any {
	contentLanguage, targetCountry, promotionID, ok := parseGCPShoppingMerchantPromotionsToken(promotionToken)
	if !ok {
		contentLanguage = "en"
		targetCountry = "US"
		promotionID = "promo"
		promotionToken = buildGCPShoppingMerchantPromotionsToken(contentLanguage, targetCountry, promotionID)
	}
	return map[string]any{
		"name":              fmt.Sprintf("accounts/%s/promotions/%s", account, promotionToken),
		"promotionId":       promotionID,
		"contentLanguage":   contentLanguage,
		"targetCountry":     targetCountry,
		"redemptionChannel": []string{"ONLINE"},
		"dataSource":        dataSource,
		"attributes": map[string]any{
			"productApplicability":         "ALL_PRODUCTS",
			"offerType":                    "NO_CODE",
			"longTitle":                    "Stackyard Promotion " + promotionID,
			"couponValueType":              "MONEY_OFF",
			"promotionEffectiveTimePeriod": map[string]any{"startTime": "2026-01-01T00:00:00Z", "endTime": "2026-01-31T23:59:59Z"},
			"promotionDisplayTimePeriod":   map[string]any{"startTime": "2026-01-01T00:00:00Z", "endTime": "2026-01-31T23:59:59Z"},
			"promotionUrl":                 "https://example.com/promotions/" + promotionID,
		},
		"customAttributes": []map[string]any{
			{"name": "campaign", "value": "spring-sale"},
		},
		"promotionStatus": gcpShoppingMerchantPromotionsStatusFixture(targetCountry),
		"versionNumber":   "42",
	}
}

func gcpShoppingMerchantPromotionsStatusFixture(targetCountry string) map[string]any {
	if targetCountry == "" {
		targetCountry = "US"
	}
	return map[string]any{
		"destinationStatuses": []map[string]any{
			{
				"reportingContext": 0,
				"status":           0,
			},
		},
		"itemLevelIssues": []map[string]any{
			{
				"code":                "promotion_under_review",
				"severity":            0,
				"resolution":          "PENDING_REVIEW",
				"attribute":           "promotionId",
				"reportingContext":    0,
				"description":         "Promotion is under review",
				"detail":              "Automatic policy checks are still running",
				"documentation":       "https://support.google.com/merchants",
				"applicableCountries": []string{targetCountry},
			},
		},
		"creationDate":   "2026-01-01T00:00:00Z",
		"lastUpdateDate": "2026-01-02T00:00:00Z",
	}
}

func gcpShoppingMerchantPromotionsNormalizeInterval(interval map[string]any) (map[string]any, bool) {
	if len(interval) == 0 {
		return nil, false
	}
	start := strings.TrimSpace(gcpShoppingMerchantPromotionsString(interval, "startTime", "start_time"))
	end := strings.TrimSpace(gcpShoppingMerchantPromotionsString(interval, "endTime", "end_time"))
	if start == "" || end == "" {
		return nil, false
	}
	startTime, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return nil, false
	}
	endTime, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return nil, false
	}
	if !endTime.After(startTime) {
		return nil, false
	}
	return map[string]any{
		"startTime": startTime.UTC().Format(time.RFC3339),
		"endTime":   endTime.UTC().Format(time.RFC3339),
	}, true
}

func gcpShoppingMerchantPromotionsParseRedemptionChannels(promotion map[string]any) ([]string, bool) {
	raw, ok := gcpShoppingMerchantPromotionsAny(promotion, "redemptionChannel", "redemption_channel")
	if !ok {
		return nil, false
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, false
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		channel := gcpShoppingMerchantPromotionsParseEnum(item, gcpShoppingMerchantPromotionsRedemptionChannelAliases, gcpShoppingMerchantPromotionsRedemptionChannelByNumber)
		if channel == "" {
			return nil, false
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		out = append(out, channel)
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func gcpShoppingMerchantPromotionsParseEnum(raw any, aliases map[string]string, byNumber map[int64]string) string {
	switch value := raw.(type) {
	case string:
		token := strings.ToUpper(strings.TrimSpace(value))
		token = strings.ReplaceAll(token, "-", "_")
		token = strings.ReplaceAll(token, " ", "_")
		token = strings.Trim(token, "_")
		if canonical, ok := aliases[token]; ok {
			return canonical
		}
	case float64:
		if canonical, ok := byNumber[int64(value)]; ok {
			return canonical
		}
	case int:
		if canonical, ok := byNumber[int64(value)]; ok {
			return canonical
		}
	case int64:
		if canonical, ok := byNumber[value]; ok {
			return canonical
		}
	}
	return ""
}

func gcpShoppingMerchantPromotionsCustomAttributes(raw []any) []map[string]any {
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		attr, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(gcpShoppingMerchantPromotionsString(attr, "name"))
		value := strings.TrimSpace(gcpShoppingMerchantPromotionsString(attr, "value"))
		if name == "" || value == "" {
			continue
		}
		out = append(out, map[string]any{
			"name":  name,
			"value": value,
		})
	}
	if len(out) == 0 {
		out = append(out, map[string]any{
			"name":  "campaign",
			"value": "spring-sale",
		})
	}
	return out
}

func gcpShoppingMerchantPromotionsAnyValue(m map[string]any, keys ...string) any {
	value, _ := gcpShoppingMerchantPromotionsAny(m, keys...)
	return value
}

func gcpShoppingMerchantPromotionsAny(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpShoppingMerchantPromotionsMap(m map[string]any, keys ...string) map[string]any {
	raw, ok := gcpShoppingMerchantPromotionsAny(m, keys...)
	if !ok {
		return nil
	}
	value, _ := raw.(map[string]any)
	return value
}

func gcpShoppingMerchantPromotionsSlice(m map[string]any, keys ...string) []any {
	raw, ok := gcpShoppingMerchantPromotionsAny(m, keys...)
	if !ok {
		return nil
	}
	value, _ := raw.([]any)
	return value
}

func gcpShoppingMerchantPromotionsStringSlice(m map[string]any, keys ...string) []string {
	values := gcpShoppingMerchantPromotionsSlice(m, keys...)
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		switch value := item.(type) {
		case string:
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func gcpShoppingMerchantPromotionsString(m map[string]any, keys ...string) string {
	raw, ok := gcpShoppingMerchantPromotionsAny(m, keys...)
	if !ok {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return ""
	}
}

func gcpShoppingMerchantPromotionsInt64(m map[string]any, keys ...string) int64 {
	raw, ok := gcpShoppingMerchantPromotionsAny(m, keys...)
	if !ok {
		return 0
	}
	switch value := raw.(type) {
	case float64:
		return int64(value)
	case int:
		return int64(value)
	case int64:
		return value
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func respondGCPShoppingMerchantPromotionsByErrorType(w http.ResponseWriter, path, errType, message string) {
	switch errType {
	case "NotFound":
		respondGCPShoppingMerchantPromotionsNotFound(w, path, message)
	case "FailedPrecondition":
		respondGCPShoppingMerchantPromotionsFailedPrecondition(w, path, message)
	case "Aborted":
		respondGCPShoppingMerchantPromotionsAborted(w, path, message)
	default:
		respondGCPShoppingMerchantPromotionsInvalidArgument(w, path, message)
	}
}

func respondGCPShoppingMerchantPromotionsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantPromotionsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantPromotionsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantPromotionsAborted(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "Aborted",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantPromotions(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_promotions") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_promotions/sample",
			"service":   "shopping_merchant_promotions",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
