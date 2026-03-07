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
	gcpShoppingMerchantProductsAccountRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantProductsLanguageRe   = regexp.MustCompile(`^[A-Za-z]{2}$`)
	gcpShoppingMerchantProductsFeedLabelRe  = regexp.MustCompile(`^[A-Za-z0-9_-]{1,20}$`)
	gcpShoppingMerchantProductsDataSourceRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

func (s *Server) handleGCPShoppingMerchantProductsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantProducts(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantProductsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantProductsPath(path, hasGCPShoppingMerchantProductsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantProductsRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.products.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantProductsGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantProductsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPShoppingMerchantProductsPATCH(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantProductsDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantProductsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantProductsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_products",
		"shopping-merchant-products",
		"shopping-merchant-products-apiv1",
		"shopping_merchant_products_apiv1",
		"merchant_products",
		"merchant-products",
		"merchantproducts",
		"gcp-shopping-merchant-products":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-products-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/products")
}

func isGCPShoppingMerchantProductsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/products/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.products.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/products/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantProductsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/products/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/products/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantProductsGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, ok := parseGCPShoppingMerchantProductsCollectionPath(tail); ok {
		return handleGCPShoppingMerchantProductsList(w, r, path, account)
	}
	account, productID, ok := parseGCPShoppingMerchantProductsProductPath(tail)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(productID), "missing") {
		respondGCPShoppingMerchantProductsNotFound(w, path, "product not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingMerchantProductsProductFixture(account, productID, fmt.Sprintf("accounts/%s/dataSources/104628", account)))
	return true
}

func handleGCPShoppingMerchantProductsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantProductsInsertPath(tail)
	if !ok {
		return false
	}
	dataSource, errType, errMessage := parseGCPShoppingMerchantProductsDataSourceQuery(account, r.URL.Query().Get("dataSource"))
	if errType != "" {
		respondGCPShoppingMerchantProductsByErrorType(w, path, errType, errMessage)
		return true
	}
	body, ok := decodeGCPShoppingMerchantProductsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	fixture, fixtureErrType, fixtureErrMessage := buildGCPShoppingMerchantProductsProductInputFixture(account, dataSource, body, "insert", "", nil)
	if fixtureErrType != "" {
		respondGCPShoppingMerchantProductsByErrorType(w, path, fixtureErrType, fixtureErrMessage)
		return true
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPShoppingMerchantProductsPATCH(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, productID, ok := parseGCPShoppingMerchantProductsProductInputPath(tail)
	if !ok {
		return false
	}
	dataSource, errType, errMessage := parseGCPShoppingMerchantProductsDataSourceQuery(account, r.URL.Query().Get("dataSource"))
	if errType != "" {
		respondGCPShoppingMerchantProductsByErrorType(w, path, errType, errMessage)
		return true
	}
	maskFields, errType, errMessage := parseGCPShoppingMerchantProductsUpdateMask(r.URL.Query().Get("updateMask"))
	if errType != "" {
		respondGCPShoppingMerchantProductsByErrorType(w, path, errType, errMessage)
		return true
	}
	body, ok := decodeGCPShoppingMerchantProductsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("accounts/%s/productInputs/%s", account, productID)
	fixture, fixtureErrType, fixtureErrMessage := buildGCPShoppingMerchantProductsProductInputFixture(account, dataSource, body, "update", expectedName, maskFields)
	if fixtureErrType != "" {
		respondGCPShoppingMerchantProductsByErrorType(w, path, fixtureErrType, fixtureErrMessage)
		return true
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPShoppingMerchantProductsDELETE(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, productID, ok := parseGCPShoppingMerchantProductsProductInputPath(tail)
	if !ok {
		return false
	}
	_, errType, errMessage := parseGCPShoppingMerchantProductsDataSourceQuery(account, r.URL.Query().Get("dataSource"))
	if errType != "" {
		respondGCPShoppingMerchantProductsByErrorType(w, path, errType, errMessage)
		return true
	}
	if strings.Contains(strings.ToLower(productID), "missing") {
		respondGCPShoppingMerchantProductsNotFound(w, path, "product input not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantProductsList(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize, start, ok := parseGCPShoppingMerchantProductsPagination(w, r, path)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpShoppingMerchantProductsProductFixture(account, "en~US~sku-1001", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		gcpShoppingMerchantProductsProductFixture(account, "en~US~sku-1002", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		gcpShoppingMerchantProductsProductFixture(account, "local~en~US~sku-local-1003", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
	}
	if start > len(items) {
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, "pageToken is out of range")
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
		"products":      items[start:end],
		"nextPageToken": next,
	})
	return true
}

func parseGCPShoppingMerchantProductsCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "products" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantProductsProductPath(tail string) (account, productID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "products" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	productID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(account) {
		return "", "", false
	}
	if _, _, _, _, valid := parseGCPShoppingMerchantProductsProductID(productID); !valid {
		return "", "", false
	}
	return account, productID, true
}

func parseGCPShoppingMerchantProductsInsertPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "productInputs:insert" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantProductsProductInputPath(tail string) (account, productID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "productInputs" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	productID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(account) {
		return "", "", false
	}
	if _, _, _, _, valid := parseGCPShoppingMerchantProductsProductID(productID); !valid {
		return "", "", false
	}
	return account, productID, true
}

func parseGCPShoppingMerchantProductsParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantProductsProductName(name string, resource string) (account, productID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != resource {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	productID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(account) {
		return "", "", false
	}
	if _, _, _, _, valid := parseGCPShoppingMerchantProductsProductID(productID); !valid {
		return "", "", false
	}
	return account, productID, true
}

func parseGCPShoppingMerchantProductsProductID(productID string) (legacyLocal bool, contentLanguage, feedLabel, offerID string, ok bool) {
	parts := strings.Split(strings.TrimSpace(productID), "~")
	switch len(parts) {
	case 3:
		contentLanguage = strings.TrimSpace(parts[0])
		feedLabel = strings.TrimSpace(parts[1])
		offerID = strings.TrimSpace(parts[2])
	case 4:
		if strings.ToLower(strings.TrimSpace(parts[0])) != "local" {
			return false, "", "", "", false
		}
		legacyLocal = true
		contentLanguage = strings.TrimSpace(parts[1])
		feedLabel = strings.TrimSpace(parts[2])
		offerID = strings.TrimSpace(parts[3])
	default:
		return false, "", "", "", false
	}
	if !gcpShoppingMerchantProductsLanguageRe.MatchString(contentLanguage) {
		return false, "", "", "", false
	}
	if !gcpShoppingMerchantProductsFeedLabelRe.MatchString(feedLabel) {
		return false, "", "", "", false
	}
	if offerID == "" {
		return false, "", "", "", false
	}
	return legacyLocal, strings.ToLower(contentLanguage), strings.ToUpper(feedLabel), offerID, true
}

func buildGCPShoppingMerchantProductsProductID(legacyLocal bool, contentLanguage, feedLabel, offerID string) string {
	contentLanguage = strings.ToLower(strings.TrimSpace(contentLanguage))
	feedLabel = strings.ToUpper(strings.TrimSpace(feedLabel))
	offerID = strings.TrimSpace(offerID)
	if legacyLocal {
		return "local~" + contentLanguage + "~" + feedLabel + "~" + offerID
	}
	return contentLanguage + "~" + feedLabel + "~" + offerID
}

func parseGCPShoppingMerchantProductsDataSourceQuery(account, raw string) (dataSource string, errType, errMessage string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "InvalidArgument", "dataSource query parameter is required"
	}
	parts := strings.Split(strings.Trim(raw, "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "dataSources" {
		return "", "InvalidArgument", "dataSource must be in accounts/{account}/dataSources/{datasource} format"
	}
	dataSourceAccount := strings.TrimSpace(parts[1])
	dataSourceID := strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantProductsAccountRe.MatchString(dataSourceAccount) || !gcpShoppingMerchantProductsDataSourceRe.MatchString(dataSourceID) {
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

func parseGCPShoppingMerchantProductsPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 25
	if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantProductsInvalidArgument(w, path, "pageSize must be a non-negative integer")
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
			respondGCPShoppingMerchantProductsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func parseGCPShoppingMerchantProductsUpdateMask(raw string) (fields []string, errType, errMessage string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", ""
	}

	var candidates []string
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "\"paths\"") || strings.HasPrefix(raw, "paths") {
		jsonMask := raw
		if strings.HasPrefix(jsonMask, "\"paths\"") || strings.HasPrefix(jsonMask, "paths") {
			jsonMask = "{" + strings.TrimPrefix(jsonMask, "paths") + "}"
			jsonMask = strings.Replace(jsonMask, "{=", "{", 1)
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(jsonMask), &parsed); err == nil {
			if rawPaths, ok := parsed["paths"].([]any); ok {
				for _, value := range rawPaths {
					if s, ok := value.(string); ok {
						candidates = append(candidates, s)
					}
				}
			}
		}
	}
	if len(candidates) == 0 {
		for _, value := range strings.Split(raw, ",") {
			candidates = append(candidates, strings.TrimSpace(value))
		}
	}

	for _, candidate := range candidates {
		field := normalizeGCPShoppingMerchantProductsUpdateMaskField(candidate)
		if field == "" {
			continue
		}
		if field == "*" {
			return nil, "InvalidArgument", "updateMask does not support full replacement"
		}
		switch field {
		case "productAttributes", "productAttributes.title", "productAttributes.description", "customAttributes":
			fields = append(fields, field)
		default:
			return nil, "InvalidArgument", "updateMask contains unsupported field"
		}
	}
	return fields, "", ""
}

func normalizeGCPShoppingMerchantProductsUpdateMaskField(field string) string {
	field = strings.TrimSpace(field)
	field = strings.Trim(field, "\"")
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	if field == "*" {
		return "*"
	}
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	canonical := strings.ToLower(replacer.Replace(field))
	switch canonical {
	case "productattributes":
		return "productAttributes"
	case "productattributes.title":
		return "productAttributes.title"
	case "productattributes.description":
		return "productAttributes.description"
	case "customattributes", "customattribute":
		return "customAttributes"
	default:
		return strings.TrimSpace(field)
	}
}

func decodeGCPShoppingMerchantProductsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func buildGCPShoppingMerchantProductsProductInputFixture(account, dataSource string, body map[string]any, mode, expectedName string, maskFields []string) (map[string]any, string, string) {
	payload := body
	if nested := gcpShoppingMerchantProductsMap(body, "productInput", "product_input"); len(nested) > 0 {
		payload = nested
	}
	if len(payload) == 0 {
		return nil, "InvalidArgument", "productInput payload is required"
	}

	rawName := strings.TrimSpace(gcpShoppingMerchantProductsString(payload, "name"))
	var (
		productIDByName                    string
		legacyLocalByName                  bool
		contentLanguageByName, offerByName string
		feedLabelByName                    string
	)
	if rawName != "" {
		nameAccount, nameProductID, ok := parseGCPShoppingMerchantProductsProductName(rawName, "productInputs")
		if !ok {
			return nil, "InvalidArgument", "productInput.name must be in accounts/{account}/productInputs/{product} format"
		}
		if nameAccount != account {
			return nil, "FailedPrecondition", "productInput.name account must match requested account"
		}
		productIDByName = nameProductID
		legacyLocalByName, contentLanguageByName, feedLabelByName, offerByName, _ = parseGCPShoppingMerchantProductsProductID(nameProductID)
	}

	if mode == "update" {
		if rawName == "" {
			return nil, "InvalidArgument", "productInput.name is required for update"
		}
		if expectedName != "" && rawName != expectedName {
			return nil, "InvalidArgument", "productInput.name must match requested resource"
		}
		if strings.Contains(strings.ToLower(productIDByName), "missing") {
			return nil, "NotFound", "product input not found"
		}
	}

	contentLanguage := strings.TrimSpace(gcpShoppingMerchantProductsString(payload, "contentLanguage", "content_language"))
	feedLabel := strings.TrimSpace(gcpShoppingMerchantProductsString(payload, "feedLabel", "feed_label"))
	offerID := strings.TrimSpace(gcpShoppingMerchantProductsString(payload, "offerId", "offer_id"))
	legacyLocal := gcpShoppingMerchantProductsBool(payload, "legacyLocal", "legacy_local")

	if rawName != "" {
		if contentLanguage == "" {
			contentLanguage = contentLanguageByName
		}
		if feedLabel == "" {
			feedLabel = feedLabelByName
		}
		if offerID == "" {
			offerID = offerByName
		}
		if legacyLocalByName {
			legacyLocal = true
		}
	}

	if contentLanguage == "" || offerID == "" || feedLabel == "" {
		return nil, "InvalidArgument", "offerId, contentLanguage, and feedLabel are required"
	}
	if !gcpShoppingMerchantProductsLanguageRe.MatchString(contentLanguage) {
		return nil, "InvalidArgument", "contentLanguage must be an ISO 639-1 language code"
	}
	if !gcpShoppingMerchantProductsFeedLabelRe.MatchString(feedLabel) {
		return nil, "InvalidArgument", "feedLabel contains unsupported characters"
	}

	productID := buildGCPShoppingMerchantProductsProductID(legacyLocal, contentLanguage, feedLabel, offerID)
	if rawName != "" && productIDByName != productID {
		return nil, "FailedPrecondition", "offerId/contentLanguage/feedLabel must match productInput.name"
	}

	if versionNumber := gcpShoppingMerchantProductsInt64(payload, "versionNumber", "version_number"); versionNumber > 0 {
		if mode == "update" {
			return nil, "InvalidArgument", "versionNumber is not supported on updates"
		}
		if versionNumber < 40 {
			return nil, "Aborted", "version_number is stale"
		}
	}

	if mode == "insert" && strings.Contains(strings.ToLower(offerID), "missing") {
		return nil, "NotFound", "product input not found"
	}

	title := strings.TrimSpace(gcpShoppingMerchantProductsString(gcpShoppingMerchantProductsMap(payload, "productAttributes", "product_attributes"), "title"))
	description := strings.TrimSpace(gcpShoppingMerchantProductsString(gcpShoppingMerchantProductsMap(payload, "productAttributes", "product_attributes"), "description"))
	if title == "" {
		title = "Stackyard Product " + offerID
	}
	if description == "" {
		description = "Staged merchant product for " + offerID
	}

	customAttributes := gcpShoppingMerchantProductsCustomAttributes(gcpShoppingMerchantProductsSlice(payload, "customAttributes", "custom_attributes"))

	if len(maskFields) > 0 {
		for _, field := range maskFields {
			switch field {
			case "productAttributes.title":
				if strings.TrimSpace(title) == "" {
					return nil, "InvalidArgument", "productAttributes.title is required by updateMask"
				}
			case "customAttributes":
				if len(customAttributes) == 0 {
					return nil, "InvalidArgument", "customAttributes is required by updateMask"
				}
			}
		}
	}

	name := fmt.Sprintf("accounts/%s/productInputs/%s", account, productID)
	if rawName != "" {
		name = rawName
	}
	return map[string]any{
		"name":            name,
		"product":         fmt.Sprintf("accounts/%s/products/%s", account, productID),
		"legacyLocal":     legacyLocal,
		"offerId":         offerID,
		"contentLanguage": strings.ToLower(contentLanguage),
		"feedLabel":       strings.ToUpper(feedLabel),
		"versionNumber":   "42",
		"productAttributes": map[string]any{
			"title":       title,
			"description": description,
			"brand":       "Stackyard",
			"link":        "https://example.com/products/" + offerID,
			"imageLink":   "https://example.com/images/" + offerID + ".jpg",
		},
		"customAttributes": customAttributes,
		"dataSource":       dataSource,
	}, "", ""
}

func gcpShoppingMerchantProductsProductFixture(account, productID, dataSource string) map[string]any {
	legacyLocal, contentLanguage, feedLabel, offerID, _ := parseGCPShoppingMerchantProductsProductID(productID)
	return map[string]any{
		"name":            fmt.Sprintf("accounts/%s/products/%s", account, productID),
		"legacyLocal":     legacyLocal,
		"offerId":         offerID,
		"contentLanguage": contentLanguage,
		"feedLabel":       feedLabel,
		"dataSource":      dataSource,
		"versionNumber":   "42",
		"productAttributes": map[string]any{
			"title":       "Stackyard Product " + offerID,
			"description": "Staged processed product for " + offerID,
			"brand":       "Stackyard",
			"link":        "https://example.com/products/" + offerID,
			"imageLink":   "https://example.com/images/" + offerID + ".jpg",
		},
		"customAttributes": []map[string]any{
			{"name": "material", "value": "cotton"},
			{"name": "color", "value": "blue"},
		},
		"productStatus": map[string]any{
			"destinationStatuses": []map[string]any{
				{
					"reportingContext":  "SHOPPING_ADS",
					"approvedCountries": []string{"US"},
				},
			},
			"itemLevelIssues": []map[string]any{
				{
					"code":                "image_link_pending",
					"severity":            "WARNING",
					"resolution":          "MERCHANT_ACTION",
					"reportingContext":    "SHOPPING_ADS",
					"description":         "Image is being reviewed",
					"detail":              "Image quality review in progress",
					"applicableCountries": []string{"US"},
				},
			},
			"creationDate":         "2026-01-01T00:00:00Z",
			"lastUpdateDate":       "2026-01-02T00:00:00Z",
			"googleExpirationDate": "2026-12-31T00:00:00Z",
		},
	}
}

func gcpShoppingMerchantProductsCustomAttributes(raw []any) []map[string]any {
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		attr, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(gcpShoppingMerchantProductsString(attr, "name"))
		value := strings.TrimSpace(gcpShoppingMerchantProductsString(attr, "value"))
		if name == "" || value == "" {
			continue
		}
		out = append(out, map[string]any{
			"name":  name,
			"value": value,
		})
	}
	if len(out) == 0 {
		out = append(out, map[string]any{"name": "material", "value": "cotton"})
	}
	return out
}

func gcpShoppingMerchantProductsAny(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpShoppingMerchantProductsMap(m map[string]any, keys ...string) map[string]any {
	raw, ok := gcpShoppingMerchantProductsAny(m, keys...)
	if !ok {
		return nil
	}
	value, _ := raw.(map[string]any)
	return value
}

func gcpShoppingMerchantProductsSlice(m map[string]any, keys ...string) []any {
	raw, ok := gcpShoppingMerchantProductsAny(m, keys...)
	if !ok {
		return nil
	}
	value, _ := raw.([]any)
	return value
}

func gcpShoppingMerchantProductsString(m map[string]any, keys ...string) string {
	raw, ok := gcpShoppingMerchantProductsAny(m, keys...)
	if !ok {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatInt(int64(value), 10)
	case int64:
		return strconv.FormatInt(value, 10)
	case int:
		return strconv.Itoa(value)
	default:
		return ""
	}
}

func gcpShoppingMerchantProductsInt64(m map[string]any, keys ...string) int64 {
	raw, ok := gcpShoppingMerchantProductsAny(m, keys...)
	if !ok {
		return 0
	}
	switch value := raw.(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
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

func gcpShoppingMerchantProductsBool(m map[string]any, keys ...string) bool {
	raw, ok := gcpShoppingMerchantProductsAny(m, keys...)
	if !ok {
		return false
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}

func respondGCPShoppingMerchantProductsByErrorType(w http.ResponseWriter, path, errType, message string) {
	switch errType {
	case "NotFound":
		respondGCPShoppingMerchantProductsNotFound(w, path, message)
	case "FailedPrecondition":
		respondGCPShoppingMerchantProductsFailedPrecondition(w, path, message)
	case "Aborted":
		respondGCPShoppingMerchantProductsAborted(w, path, message)
	default:
		respondGCPShoppingMerchantProductsInvalidArgument(w, path, message)
	}
}

func respondGCPShoppingMerchantProductsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantProductsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantProductsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantProductsAborted(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "Aborted",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantProducts(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_products") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_products/sample",
			"service":   "shopping_merchant_products",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
