package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	gcpShoppingCSSAccountIDRe  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingCSSLabelIDRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingCSSInputIDRe    = regexp.MustCompile(`^[A-Za-z0-9._~\-]+$`)
	gcpShoppingCSSLanguageRe   = regexp.MustCompile(`^[a-z]{2}(-[A-Za-z]{2})?$`)
	gcpShoppingCSSFeedLabelRe  = regexp.MustCompile(`^[A-Za-z]{2}$`)
	gcpShoppingCSSRawIDRe      = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	gcpShoppingCSSActionTailRe = regexp.MustCompile(`^([A-Za-z0-9._-]+):([A-Za-z][A-Za-z0-9]+)$`)
)

func (s *Server) handleGCPShoppingCSSRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingCSS(w, r) {
		return true
	}

	path := normalizeGCPShoppingCSSPath(rawRequestPath(r))
	if !isGCPShoppingCSSPath(path, hasGCPShoppingCSSHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingCSSListChildAccounts(w, r, path) {
			return true
		}
		if handleGCPShoppingCSSGetAccount(w, path) {
			return true
		}
		if handleGCPShoppingCSSListAccountLabels(w, r, path) {
			return true
		}
		if handleGCPShoppingCSSGetCssProduct(w, path) {
			return true
		}
		if handleGCPShoppingCSSListCssProducts(w, r, path) {
			return true
		}
		if handleGCPShoppingCSSListQuotaGroups(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingCSSUpdateLabels(w, r, path) {
			return true
		}
		if handleGCPShoppingCSSCreateAccountLabel(w, r, path) {
			return true
		}
		if handleGCPShoppingCSSInsertCssProductInput(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPShoppingCSSUpdateAccountLabel(w, r, path) {
			return true
		}
		if handleGCPShoppingCSSUpdateCssProductInput(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingCSSDeleteAccountLabel(w, path) {
			return true
		}
		if handleGCPShoppingCSSDeleteCssProductInput(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingCSSPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingCSSHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_css",
		"shopping-css",
		"shopping-css-apiv1",
		"shopping_css_apiv1",
		"css",
		"shoppingcss",
		"gcp-shopping-css":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-css-apiv1") || strings.Contains(ua, "cloud.google.com/go/shopping/css")
}

func isGCPShoppingCSSPath(path string, includeHint bool) bool {
	if _, ok := parseGCPShoppingCSSAccountPath(path); ok {
		return true
	}
	if _, action, ok := parseGCPShoppingCSSAccountActionPath(path); ok {
		return action == "listChildAccounts" || action == "updateLabels"
	}
	if _, ok := parseGCPShoppingCSSLabelsCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPShoppingCSSLabelResourcePath(path); ok {
		return true
	}
	if _, ok := parseGCPShoppingCSSCssProductsCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPShoppingCSSCssProductResourcePath(path); ok {
		return true
	}
	if _, action, ok := parseGCPShoppingCSSCssProductInputsActionPath(path); ok {
		return action == "insert"
	}
	if _, _, ok := parseGCPShoppingCSSCssProductInputResourcePath(path); ok {
		return true
	}
	if _, ok := parseGCPShoppingCSSQuotaCollectionPath(path); ok {
		return true
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/accounts/")
}

func handleGCPShoppingCSSListChildAccounts(w http.ResponseWriter, r *http.Request, path string) bool {
	account, action, ok := parseGCPShoppingCSSAccountActionPath(path)
	if !ok || action != "listChildAccounts" {
		return false
	}

	pageSize, start, valid := parseGCPShoppingCSSPagination(w, r, path)
	if !valid {
		return true
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("labelId")); raw != "" {
		id, err := parseOptionalNonNegativeInt(raw)
		if err != nil {
			respondGCPShoppingCSSInvalidArgument(w, path, "labelId must be a non-negative integer")
			return true
		}
		if id == 0 {
			respondJSON(w, http.StatusOK, map[string]any{"accounts": []any{}, "nextPageToken": ""})
			return true
		}
	}

	items := []map[string]any{
		gcpShoppingCSSAccountFixture(account+"-child-1", []int64{1001}),
		gcpShoppingCSSAccountFixture(account+"-child-2", []int64{1002}),
	}
	if fullName := strings.TrimSpace(r.URL.Query().Get("fullName")); fullName != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(gcpShoppingCSSString(item, "fullName")), fullName) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	response, paged := gcpShoppingCSSPaginateList("accounts", items, pageSize, start)
	if !paged {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPShoppingCSSGetAccount(w http.ResponseWriter, path string) bool {
	account, ok := parseGCPShoppingCSSAccountPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpShoppingCSSAccountFixture(account, []int64{1001, 1002}))
	return true
}

func handleGCPShoppingCSSUpdateLabels(w http.ResponseWriter, r *http.Request, path string) bool {
	account, action, ok := parseGCPShoppingCSSAccountActionPath(path)
	if !ok || action != "updateLabels" {
		return false
	}

	body, valid := decodeGCPShoppingCSSJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}

	expectedName := gcpShoppingCSSAccountName(account)
	name := strings.TrimSpace(gcpShoppingCSSString(body, "name"))
	if name == "" {
		respondGCPShoppingCSSInvalidArgument(w, path, "name is required")
		return true
	}
	if name != expectedName {
		respondGCPShoppingCSSInvalidArgument(w, path, "name must match requested resource")
		return true
	}

	if parent := strings.TrimSpace(gcpShoppingCSSString(body, "parent")); parent != "" {
		if _, ok := parseGCPShoppingCSSAccountName(parent); !ok {
			respondGCPShoppingCSSInvalidArgument(w, path, "parent must be accounts/{account}")
			return true
		}
	}

	labelIDs, ok := gcpShoppingCSSInt64Slice(body["labelIds"])
	if !ok {
		respondGCPShoppingCSSInvalidArgument(w, path, "labelIds must be a list of non-negative integers")
		return true
	}
	if len(labelIDs) == 0 {
		respondGCPShoppingCSSInvalidArgument(w, path, "labelIds is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpShoppingCSSAccountFixture(account, labelIDs))
	return true
}

func handleGCPShoppingCSSListAccountLabels(w http.ResponseWriter, r *http.Request, path string) bool {
	account, ok := parseGCPShoppingCSSLabelsCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPShoppingCSSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpShoppingCSSAccountLabelFixture(account, "label-1", "Stackyard Label 1", "Default staged label"),
		gcpShoppingCSSAccountLabelFixture(account, "label-2", "Stackyard Label 2", "Secondary staged label"),
	}
	response, paged := gcpShoppingCSSPaginateList("accountLabels", items, pageSize, start)
	if !paged {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPShoppingCSSCreateAccountLabel(w http.ResponseWriter, r *http.Request, path string) bool {
	account, ok := parseGCPShoppingCSSLabelsCollectionPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPShoppingCSSJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}

	displayName := strings.TrimSpace(gcpShoppingCSSString(body, "displayName", "display_name"))
	if displayName == "" {
		respondGCPShoppingCSSInvalidArgument(w, path, "displayName is required")
		return true
	}

	labelID := gcpShoppingCSSLabelIDFromBody(body, displayName)
	if !isGCPShoppingCSSLabelID(labelID) {
		respondGCPShoppingCSSInvalidArgument(w, path, "accountLabel.name is invalid")
		return true
	}
	if gcpShoppingCSSLabelDuplicate(displayName, labelID) {
		respondGCPShoppingCSSAlreadyExists(w, path, "account label already exists")
		return true
	}

	description := strings.TrimSpace(gcpShoppingCSSString(body, "description"))
	respondJSON(w, http.StatusOK, gcpShoppingCSSAccountLabelFixture(account, labelID, displayName, description))
	return true
}

func handleGCPShoppingCSSUpdateAccountLabel(w http.ResponseWriter, r *http.Request, path string) bool {
	account, labelID, ok := parseGCPShoppingCSSLabelResourcePath(path)
	if !ok {
		return false
	}
	if gcpShoppingCSSLabelMissing(labelID) {
		respondGCPShoppingCSSNotFound(w, path, "account label was not found")
		return true
	}

	body, valid := decodeGCPShoppingCSSJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	name := strings.TrimSpace(gcpShoppingCSSString(body, "name"))
	expectedName := gcpShoppingCSSLabelName(account, labelID)
	if name == "" {
		respondGCPShoppingCSSInvalidArgument(w, path, "accountLabel.name is required")
		return true
	}
	if name != expectedName {
		respondGCPShoppingCSSInvalidArgument(w, path, "accountLabel.name must match requested resource")
		return true
	}

	displayName := strings.TrimSpace(gcpShoppingCSSString(body, "displayName", "display_name"))
	if displayName == "" {
		respondGCPShoppingCSSInvalidArgument(w, path, "displayName is required")
		return true
	}
	description := strings.TrimSpace(gcpShoppingCSSString(body, "description"))
	respondJSON(w, http.StatusOK, gcpShoppingCSSAccountLabelFixture(account, labelID, displayName, description))
	return true
}

func handleGCPShoppingCSSDeleteAccountLabel(w http.ResponseWriter, path string) bool {
	_, labelID, ok := parseGCPShoppingCSSLabelResourcePath(path)
	if !ok {
		return false
	}
	if gcpShoppingCSSLabelMissing(labelID) {
		respondGCPShoppingCSSNotFound(w, path, "account label was not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingCSSGetCssProduct(w http.ResponseWriter, path string) bool {
	account, productID, ok := parseGCPShoppingCSSCssProductResourcePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpShoppingCSSCssProductFixture(account, productID, "Stackyard Product "+productID))
	return true
}

func handleGCPShoppingCSSListCssProducts(w http.ResponseWriter, r *http.Request, path string) bool {
	account, ok := parseGCPShoppingCSSCssProductsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPShoppingCSSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpShoppingCSSCssProductFixture(account, "en~US~sku-1", "Stackyard Tee"),
		gcpShoppingCSSCssProductFixture(account, "en~US~sku-2", "Stackyard Hoodie"),
	}
	response, paged := gcpShoppingCSSPaginateList("cssProducts", items, pageSize, start)
	if !paged {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPShoppingCSSInsertCssProductInput(w http.ResponseWriter, r *http.Request, path string) bool {
	account, action, ok := parseGCPShoppingCSSCssProductInputsActionPath(path)
	if !ok || action != "insert" {
		return false
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("feedId")); raw != "" {
		if _, err := parseOptionalNonNegativeInt(raw); err != nil {
			respondGCPShoppingCSSInvalidArgument(w, path, "feedId must be a non-negative integer")
			return true
		}
	}

	body, valid := decodeGCPShoppingCSSJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	inputID, title, valid := validateGCPShoppingCSSInputBody(w, path, body, account, false)
	if !valid {
		return true
	}
	_, _, rawID := gcpShoppingCSSParseInputID(inputID)
	if strings.Contains(strings.ToLower(rawID), "existing") {
		respondGCPShoppingCSSAlreadyExists(w, path, "css product input already exists")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingCSSCssProductInputFixture(account, inputID, title))
	return true
}

func handleGCPShoppingCSSUpdateCssProductInput(w http.ResponseWriter, r *http.Request, path string) bool {
	account, inputIDFromPath, ok := parseGCPShoppingCSSCssProductInputResourcePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPShoppingCSSInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if gcpShoppingCSSInputMissing(inputIDFromPath) {
		respondGCPShoppingCSSNotFound(w, path, "css product input was not found")
		return true
	}

	body, valid := decodeGCPShoppingCSSJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	inputID, title, valid := validateGCPShoppingCSSInputBody(w, path, body, account, true)
	if !valid {
		return true
	}
	if inputID != inputIDFromPath {
		respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpShoppingCSSCssProductInputFixture(account, inputID, title))
	return true
}

func handleGCPShoppingCSSDeleteCssProductInput(w http.ResponseWriter, r *http.Request, path string) bool {
	_, inputID, ok := parseGCPShoppingCSSCssProductInputResourcePath(path)
	if !ok {
		return false
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("supplementalFeedId")); raw != "" {
		if _, err := parseOptionalNonNegativeInt(raw); err != nil {
			respondGCPShoppingCSSInvalidArgument(w, path, "supplementalFeedId must be a non-negative integer")
			return true
		}
	}
	if gcpShoppingCSSInputMissing(inputID) {
		respondGCPShoppingCSSNotFound(w, path, "css product input was not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingCSSListQuotaGroups(w http.ResponseWriter, r *http.Request, path string) bool {
	account, ok := parseGCPShoppingCSSQuotaCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPShoppingCSSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpShoppingCSSQuotaGroupFixture(account, "css-products-read", 12, 1000, 60, "cssproductsservice.listcssproducts"),
		gcpShoppingCSSQuotaGroupFixture(account, "css-products-write", 4, 300, 30, "cssproductinputsservice.insertcssproductinput"),
	}
	response, paged := gcpShoppingCSSPaginateList("quotaGroups", items, pageSize, start)
	if !paged {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func parseGCPShoppingCSSPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start, err = parseOptionalNonNegativeInt(r.URL.Query().Get("pageToken"))
	if err != nil {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func decodeGCPShoppingCSSJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPShoppingCSSJSONBody(w, r, path, true)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingCSSInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func decodeGCPShoppingCSSJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	if r.Body == nil {
		if required {
			respondGCPShoppingCSSInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	defer r.Body.Close()

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.UseNumber()

	body := map[string]any{}
	if err := decoder.Decode(&body); err != nil {
		if err == io.EOF && !required {
			return map[string]any{}, true
		}
		respondGCPShoppingCSSInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func validateGCPShoppingCSSInputBody(w http.ResponseWriter, path string, body map[string]any, account string, requireName bool) (inputID, title string, ok bool) {
	rawProvidedID := strings.TrimSpace(gcpShoppingCSSString(body, "rawProvidedId", "raw_provided_id"))
	if !gcpShoppingCSSRawIDRe.MatchString(rawProvidedID) {
		respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.rawProvidedId is required")
		return "", "", false
	}
	contentLanguage := strings.TrimSpace(gcpShoppingCSSString(body, "contentLanguage", "content_language"))
	if !gcpShoppingCSSLanguageRe.MatchString(contentLanguage) {
		respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.contentLanguage is invalid")
		return "", "", false
	}
	feedLabel := strings.ToUpper(strings.TrimSpace(gcpShoppingCSSString(body, "feedLabel", "feed_label")))
	if !gcpShoppingCSSFeedLabelRe.MatchString(feedLabel) {
		respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.feedLabel is invalid")
		return "", "", false
	}
	inputID = contentLanguage + "~" + feedLabel + "~" + rawProvidedID

	name := strings.TrimSpace(gcpShoppingCSSString(body, "name"))
	if name != "" {
		parsedAccount, parsedInputID, valid := parseGCPShoppingCSSCssProductInputName(name)
		if !valid {
			respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.name is invalid")
			return "", "", false
		}
		if parsedAccount != account {
			respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.name must match requested account")
			return "", "", false
		}
		inputID = parsedInputID
	}
	if requireName && name == "" {
		respondGCPShoppingCSSInvalidArgument(w, path, "cssProductInput.name is required")
		return "", "", false
	}

	attributes := gcpShoppingCSSMap(body, "attributes")
	title = strings.TrimSpace(gcpShoppingCSSString(attributes, "title"))
	if title == "" {
		title = "Stackyard Product " + rawProvidedID
	}
	return inputID, title, true
}

func gcpShoppingCSSAccountFixture(account string, labelIDs []int64) map[string]any {
	if len(labelIDs) == 0 {
		labelIDs = []int64{1001}
	}
	sort.Slice(labelIDs, func(i, j int) bool { return labelIDs[i] < labelIDs[j] })
	return map[string]any{
		"name":              gcpShoppingCSSAccountName(account),
		"fullName":          "Stackyard CSS Account " + account,
		"displayName":       "Stackyard CSS " + account,
		"homepageUri":       fmt.Sprintf("https://merchant.stackyard.example/%s", account),
		"parent":            "accounts/999999",
		"labelIds":          gcpShoppingCSSInt64JSONSlice(labelIDs),
		"automaticLabelIds": []any{"9001"},
		"accountType":       "CSS_DOMAIN",
	}
}

func gcpShoppingCSSAccountLabelFixture(account, labelID, displayName, description string) map[string]any {
	if displayName == "" {
		displayName = "Stackyard Label " + labelID
	}
	if description == "" {
		description = "Staged account label " + labelID
	}
	return map[string]any{
		"name":        gcpShoppingCSSLabelName(account, labelID),
		"labelId":     strconv.FormatInt(gcpShoppingCSSNumericID(labelID, 1001), 10),
		"accountId":   strconv.FormatInt(gcpShoppingCSSNumericID(account, 123456), 10),
		"displayName": displayName,
		"description": description,
		"labelType":   "MANUAL",
	}
}

func gcpShoppingCSSCssProductInputFixture(account, inputID, title string) map[string]any {
	contentLanguage, feedLabel, rawProvidedID := gcpShoppingCSSParseInputID(inputID)
	if title == "" {
		title = "Stackyard Product " + rawProvidedID
	}
	return map[string]any{
		"name":            gcpShoppingCSSCssProductInputName(account, inputID),
		"finalName":       gcpShoppingCSSCssProductName(account, inputID),
		"rawProvidedId":   rawProvidedID,
		"contentLanguage": contentLanguage,
		"feedLabel":       feedLabel,
		"attributes": map[string]any{
			"title": title,
		},
	}
}

func gcpShoppingCSSCssProductFixture(account, productID, title string) map[string]any {
	contentLanguage, feedLabel, rawProvidedID := gcpShoppingCSSParseInputID(productID)
	if title == "" {
		title = "Stackyard Product " + rawProvidedID
	}
	return map[string]any{
		"name":            gcpShoppingCSSCssProductName(account, productID),
		"rawProvidedId":   rawProvidedID,
		"contentLanguage": contentLanguage,
		"feedLabel":       feedLabel,
		"attributes": map[string]any{
			"title": title,
		},
	}
}

func gcpShoppingCSSQuotaGroupFixture(account, group string, usage, limit, minuteLimit int64, method string) map[string]any {
	return map[string]any{
		"name":             gcpShoppingCSSQuotaGroupName(account, group),
		"quotaUsage":       strconv.FormatInt(usage, 10),
		"quotaLimit":       strconv.FormatInt(limit, 10),
		"quotaMinuteLimit": strconv.FormatInt(minuteLimit, 10),
		"methodDetails": []any{
			map[string]any{
				"method":  method,
				"version": "v1",
				"subapi":  "css",
				"path":    "v1/" + method,
			},
		},
	}
}

func gcpShoppingCSSPaginateList(field string, items []map[string]any, pageSize, start int) (map[string]any, bool) {
	if start < 0 || start > len(items) {
		return nil, false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	values := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		values = append(values, item)
	}
	return map[string]any{
		field:           values,
		"nextPageToken": nextToken,
	}, true
}

func parseGCPShoppingCSSAccountPath(path string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingCSSAccountActionPath(path string) (account, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" {
		return "", "", false
	}
	match := gcpShoppingCSSActionTailRe.FindStringSubmatch(strings.TrimSpace(parts[3]))
	if len(match) != 3 {
		return "", "", false
	}
	account = strings.TrimSpace(match[1])
	action = strings.TrimSpace(match[2])
	if !isGCPShoppingCSSAccountID(account) || action == "" {
		return "", "", false
	}
	return account, action, true
}

func parseGCPShoppingCSSLabelsCollectionPath(path string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" || parts[4] != "labels" {
		return "", false
	}
	account = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingCSSLabelResourcePath(path string) (account, labelID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" || parts[4] != "labels" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[3])
	labelID = strings.TrimSpace(parts[5])
	if !isGCPShoppingCSSAccountID(account) || !isGCPShoppingCSSLabelID(labelID) {
		return "", "", false
	}
	return account, labelID, true
}

func parseGCPShoppingCSSCssProductsCollectionPath(path string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" || parts[4] != "cssProducts" {
		return "", false
	}
	account = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingCSSCssProductResourcePath(path string) (account, productID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" || parts[4] != "cssProducts" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[3])
	productID = strings.TrimSpace(parts[5])
	if !isGCPShoppingCSSAccountID(account) || !isGCPShoppingCSSInputID(productID) {
		return "", "", false
	}
	return account, productID, true
}

func parseGCPShoppingCSSCssProductInputsActionPath(path string) (account, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) {
		return "", "", false
	}
	resourceTail := strings.TrimSpace(parts[4])
	prefix, action, hasAction := strings.Cut(resourceTail, ":")
	if !hasAction || prefix != "cssProductInputs" || strings.TrimSpace(action) == "" {
		return "", "", false
	}
	return account, strings.TrimSpace(action), true
}

func parseGCPShoppingCSSCssProductInputResourcePath(path string) (account, inputID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" || parts[4] != "cssProductInputs" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[3])
	inputID = strings.TrimSpace(parts[5])
	if !isGCPShoppingCSSAccountID(account) || !isGCPShoppingCSSInputID(inputID) {
		return "", "", false
	}
	return account, inputID, true
}

func parseGCPShoppingCSSQuotaCollectionPath(path string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "accounts" || parts[4] != "quotas" {
		return "", false
	}
	account = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingCSSAccountName(name string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !isGCPShoppingCSSAccountID(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingCSSLabelName(name string) (account, labelID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "labels" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	labelID = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) || !isGCPShoppingCSSLabelID(labelID) {
		return "", "", false
	}
	return account, labelID, true
}

func parseGCPShoppingCSSCssProductName(name string) (account, productID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "cssProducts" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	productID = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) || !isGCPShoppingCSSInputID(productID) {
		return "", "", false
	}
	return account, productID, true
}

func parseGCPShoppingCSSCssProductInputName(name string) (account, inputID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "cssProductInputs" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	inputID = strings.TrimSpace(parts[3])
	if !isGCPShoppingCSSAccountID(account) || !isGCPShoppingCSSInputID(inputID) {
		return "", "", false
	}
	return account, inputID, true
}

func gcpShoppingCSSAccountName(account string) string {
	return "accounts/" + strings.TrimSpace(account)
}

func gcpShoppingCSSLabelName(account, labelID string) string {
	return gcpShoppingCSSAccountName(account) + "/labels/" + strings.TrimSpace(labelID)
}

func gcpShoppingCSSCssProductName(account, productID string) string {
	return gcpShoppingCSSAccountName(account) + "/cssProducts/" + strings.TrimSpace(productID)
}

func gcpShoppingCSSCssProductInputName(account, inputID string) string {
	return gcpShoppingCSSAccountName(account) + "/cssProductInputs/" + strings.TrimSpace(inputID)
}

func gcpShoppingCSSQuotaGroupName(account, group string) string {
	return gcpShoppingCSSAccountName(account) + "/quotas/" + strings.TrimSpace(group)
}

func gcpShoppingCSSLabelIDFromBody(body map[string]any, displayName string) string {
	if name := strings.TrimSpace(gcpShoppingCSSString(body, "name")); name != "" {
		if _, labelID, ok := parseGCPShoppingCSSLabelName(name); ok {
			return labelID
		}
	}
	slug := gcpShoppingCSSSlug(displayName)
	if slug == "" {
		return "label-1"
	}
	return "label-" + slug
}

func gcpShoppingCSSLabelDuplicate(displayName, labelID string) bool {
	lowerDisplay := strings.ToLower(strings.TrimSpace(displayName))
	lowerLabelID := strings.ToLower(strings.TrimSpace(labelID))
	return strings.Contains(lowerDisplay, "existing") || strings.Contains(lowerLabelID, "existing")
}

func gcpShoppingCSSLabelMissing(labelID string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(labelID)), "missing")
}

func gcpShoppingCSSInputMissing(inputID string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(inputID)), "missing")
}

func gcpShoppingCSSParseInputID(inputID string) (contentLanguage, feedLabel, rawProvidedID string) {
	contentLanguage = "en"
	feedLabel = "US"
	rawProvidedID = "sku-1"

	parts := strings.Split(strings.TrimSpace(inputID), "~")
	if len(parts) == 3 {
		if strings.TrimSpace(parts[0]) != "" {
			contentLanguage = strings.TrimSpace(parts[0])
		}
		if strings.TrimSpace(parts[1]) != "" {
			feedLabel = strings.ToUpper(strings.TrimSpace(parts[1]))
		}
		if strings.TrimSpace(parts[2]) != "" {
			rawProvidedID = strings.TrimSpace(parts[2])
		}
		return contentLanguage, feedLabel, rawProvidedID
	}

	trimmed := strings.TrimSpace(inputID)
	if trimmed != "" {
		rawProvidedID = trimmed
	}
	return contentLanguage, feedLabel, rawProvidedID
}

func gcpShoppingCSSNumericID(value string, fallback int64) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		if parsed >= 0 {
			return parsed
		}
		return fallback
	}
	digits := strings.Builder{}
	for _, r := range value {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	if digits.Len() == 0 {
		return fallback
	}
	parsed, err := strconv.ParseInt(digits.String(), 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func gcpShoppingCSSInt64JSONSlice(values []int64) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, strconv.FormatInt(value, 10))
	}
	return out
}

func gcpShoppingCSSInt64Slice(raw any) ([]int64, bool) {
	if raw == nil {
		return nil, true
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	out := make([]int64, 0, len(items))
	for _, item := range items {
		value, valid := gcpShoppingCSSInt64FromAny(item)
		if !valid || value < 0 {
			return nil, false
		}
		out = append(out, value)
	}
	return out, true
}

func gcpShoppingCSSInt64FromAny(raw any) (int64, bool) {
	switch v := raw.(type) {
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	case float64:
		if v < 0 || v != float64(int64(v)) {
			return 0, false
		}
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func gcpShoppingCSSString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, exists := body[key]
		if !exists {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func gcpShoppingCSSMap(body map[string]any, key string) map[string]any {
	raw, ok := body[key]
	if !ok {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func gcpShoppingCSSSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	return out
}

func isGCPShoppingCSSAccountID(account string) bool {
	return gcpShoppingCSSAccountIDRe.MatchString(strings.TrimSpace(account))
}

func isGCPShoppingCSSLabelID(labelID string) bool {
	return gcpShoppingCSSLabelIDRe.MatchString(strings.TrimSpace(labelID))
}

func isGCPShoppingCSSInputID(inputID string) bool {
	return gcpShoppingCSSInputIDRe.MatchString(strings.TrimSpace(inputID))
}

func respondGCPShoppingCSSInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPShoppingCSSError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPShoppingCSSFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPShoppingCSSError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPShoppingCSSNotFound(w http.ResponseWriter, path, message string) {
	respondGCPShoppingCSSError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPShoppingCSSAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPShoppingCSSError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPShoppingCSSError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingCSS(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPShoppingCSSPath(rawRequestPath(r))
	if !isGCPShoppingCSSPath(path, true) {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPShoppingCSSInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpShoppingCSSAccountFixture("123456", []int64{1001})
	payload["service"] = "shopping_css"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}
