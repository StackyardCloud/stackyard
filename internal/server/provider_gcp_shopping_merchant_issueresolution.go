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
	gcpShoppingMerchantIssueresolutionAccountRe   = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantIssueresolutionProductRe   = regexp.MustCompile(`^[A-Za-z0-9._~:-]+$`)
	gcpShoppingMerchantIssueresolutionLanguageRe  = regexp.MustCompile(`^[A-Za-z]{2,3}([_-][A-Za-z0-9]{2,8})*$`)
	gcpShoppingMerchantIssueresolutionTimeZoneRe  = regexp.MustCompile(`^UTC$|^[A-Za-z_]+(/[A-Za-z_]+)+$`)
	gcpShoppingMerchantIssueresolutionFilterKeyRe = regexp.MustCompile(`([a-z_]+)\s*=`)
)

func (s *Server) handleGCPShoppingMerchantIssueresolutionRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantIssueResolution(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantIssueresolutionPath(rawRequestPath(r))
	if !isGCPShoppingMerchantIssueresolutionPath(path, hasGCPShoppingMerchantIssueresolutionHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantIssueresolutionRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.issueresolution.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantIssueresolutionGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantIssueresolutionPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantIssueresolutionPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantIssueresolutionHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_issueresolution",
		"shopping-merchant-issueresolution",
		"shopping-merchant-issueresolution-apiv1",
		"shopping_merchant_issueresolution_apiv1",
		"merchant_issueresolution",
		"merchant-issueresolution",
		"merchantissueresolution",
		"gcp-shopping-merchant-issueresolution":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-issueresolution-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/issueresolution")
}

func isGCPShoppingMerchantIssueresolutionPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/issueresolution/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.issueresolution.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/issueresolution/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantIssueresolutionRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/issueresolution/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/issueresolution/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantIssueresolutionGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantIssueresolutionAggregateCollectionPath(tail)
	if !ok {
		return false
	}
	return handleGCPShoppingMerchantIssueresolutionListAggregateProductStatuses(w, r, path, account)
}

func handleGCPShoppingMerchantIssueresolutionPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, action, ok := parseGCPShoppingMerchantIssueresolutionAccountActionPath(tail); ok {
		switch action {
		case "renderaccountissues":
			return handleGCPShoppingMerchantIssueresolutionRenderAccountIssues(w, r, path, account)
		case "triggeraction":
			return handleGCPShoppingMerchantIssueresolutionTriggerAction(w, r, path, account)
		default:
			return false
		}
	}
	if account, product, action, ok := parseGCPShoppingMerchantIssueresolutionProductActionPath(tail); ok {
		if action == "renderproductissues" {
			return handleGCPShoppingMerchantIssueresolutionRenderProductIssues(w, r, path, account, product)
		}
	}
	return false
}

func handleGCPShoppingMerchantIssueresolutionRenderAccountIssues(w http.ResponseWriter, r *http.Request, path, account string) bool {
	if !validateGCPShoppingMerchantIssueresolutionLocaleQuery(w, r, path, true) {
		return true
	}
	body, ok := decodeGCPShoppingMerchantIssueresolutionJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	if !validateGCPShoppingMerchantIssueresolutionRenderPayload(w, path, body) {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"renderedIssues": []map[string]any{
			gcpShoppingMerchantIssueresolutionRenderedIssueFixture("account", account, ""),
		},
	})
	return true
}

func handleGCPShoppingMerchantIssueresolutionRenderProductIssues(w http.ResponseWriter, r *http.Request, path, account, product string) bool {
	if !validateGCPShoppingMerchantIssueresolutionLocaleQuery(w, r, path, true) {
		return true
	}
	body, ok := decodeGCPShoppingMerchantIssueresolutionJSONBody(w, r, path, false)
	if !ok {
		return true
	}
	if !validateGCPShoppingMerchantIssueresolutionRenderPayload(w, path, body) {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"renderedIssues": []map[string]any{
			gcpShoppingMerchantIssueresolutionRenderedIssueFixture("product", account, product),
		},
	})
	return true
}

func handleGCPShoppingMerchantIssueresolutionTriggerAction(w http.ResponseWriter, r *http.Request, path, account string) bool {
	_ = account
	if !validateGCPShoppingMerchantIssueresolutionLocaleQuery(w, r, path, false) {
		return true
	}
	body, ok := decodeGCPShoppingMerchantIssueresolutionJSONBody(w, r, path, true)
	if !ok {
		return true
	}
	actionContext, actionFlowID, ok := validateGCPShoppingMerchantIssueresolutionTriggerPayload(w, path, body)
	if !ok {
		return true
	}
	switch {
	case strings.Contains(strings.ToLower(actionContext), "missing"):
		respondGCPShoppingMerchantIssueresolutionNotFound(w, path, "action context not found")
		return true
	case strings.Contains(strings.ToLower(actionContext), "locked"):
		respondGCPShoppingMerchantIssueresolutionFailedPrecondition(w, path, "action is not available in current state")
		return true
	}
	if actionContext != "ctx-account-review" && actionContext != "ctx-product-review" {
		respondGCPShoppingMerchantIssueresolutionNotFound(w, path, "action context not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"message": fmt.Sprintf("action started for context %s with flow %s", actionContext, actionFlowID),
	})
	return true
}

func handleGCPShoppingMerchantIssueresolutionListAggregateProductStatuses(w http.ResponseWriter, r *http.Request, path, account string) bool {
	pageSize := 25
	pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return true
		}
		pageSize = parsed
	}
	if pageSize > 250 {
		pageSize = 250
	}

	start := 0
	pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageTokenRaw != "" {
		parsed, err := strconv.Atoi(pageTokenRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return true
		}
		start = parsed
	}

	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if !gcpShoppingMerchantIssueresolutionHasSupportedFilter(filter) {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "filter supports only reporting_context and country")
		return true
	}
	filterReportingContext, filterCountry := gcpShoppingMerchantIssueresolutionExtractFilterTerms(filter)

	items := []map[string]any{
		gcpShoppingMerchantIssueresolutionAggregateStatusFixture(account, "shopping_ads-us", "SHOPPING_ADS", "US"),
		gcpShoppingMerchantIssueresolutionAggregateStatusFixture(account, "free_listings-us", "FREE_LISTINGS", "US"),
		gcpShoppingMerchantIssueresolutionAggregateStatusFixture(account, "shopping_ads-ca", "SHOPPING_ADS", "CA"),
	}
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if filterReportingContext != "" && strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(item, "reportingContext", "reporting_context"))) != filterReportingContext {
			continue
		}
		if filterCountry != "" && strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(item, "country"))) != filterCountry {
			continue
		}
		filtered = append(filtered, item)
	}
	if start > len(filtered) {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "pageToken is out of range")
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
		"aggregateProductStatuses": filtered[start:end],
		"nextPageToken":            next,
	})
	return true
}

func parseGCPShoppingMerchantIssueresolutionAccountActionPath(tail string) (account, action string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(tail), "/")
	colon := strings.LastIndex(trimmed, ":")
	if colon <= 0 || colon >= len(trimmed)-1 {
		return "", "", false
	}
	action = strings.ToLower(strings.TrimSpace(trimmed[colon+1:]))
	resource := strings.TrimSpace(trimmed[:colon])
	parts := strings.Split(resource, "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantIssueresolutionAccountRe.MatchString(account) {
		return "", "", false
	}
	return account, action, true
}

func parseGCPShoppingMerchantIssueresolutionProductActionPath(tail string) (account, product, action string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(tail), "/")
	colon := strings.LastIndex(trimmed, ":")
	if colon <= 0 || colon >= len(trimmed)-1 {
		return "", "", "", false
	}
	action = strings.ToLower(strings.TrimSpace(trimmed[colon+1:]))
	resource := strings.TrimSpace(trimmed[:colon])
	parts := strings.Split(resource, "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "products" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantIssueresolutionAccountRe.MatchString(account) || !gcpShoppingMerchantIssueresolutionProductRe.MatchString(product) {
		return "", "", "", false
	}
	return account, product, action, true
}

func parseGCPShoppingMerchantIssueresolutionAggregateCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || parts[2] != "aggregateProductStatuses" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantIssueresolutionAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func decodeGCPShoppingMerchantIssueresolutionJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	defer r.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(bodyBytes))) == 0 {
		if required {
			respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return payload, true
}

func validateGCPShoppingMerchantIssueresolutionRenderPayload(w http.ResponseWriter, path string, body map[string]any) bool {
	if len(body) == 0 {
		return true
	}
	if !gcpShoppingMerchantIssueresolutionValidateEnumLikeValue(body, "contentOption", "content_option") {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "contentOption must be a non-empty string or integer")
		return false
	}
	if !gcpShoppingMerchantIssueresolutionValidateEnumLikeValue(body, "userInputActionOption", "user_input_action_option") {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "userInputActionOption must be a non-empty string or integer")
		return false
	}
	return true
}

func validateGCPShoppingMerchantIssueresolutionLocaleQuery(w http.ResponseWriter, r *http.Request, path string, allowTimeZone bool) bool {
	languageCode := strings.TrimSpace(r.URL.Query().Get("languageCode"))
	if languageCode != "" && !gcpShoppingMerchantIssueresolutionLanguageRe.MatchString(languageCode) {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "languageCode must be a valid BCP-47 tag")
		return false
	}
	if !allowTimeZone {
		return true
	}
	timeZone := strings.TrimSpace(r.URL.Query().Get("timeZone"))
	if timeZone != "" && !gcpShoppingMerchantIssueresolutionTimeZoneRe.MatchString(timeZone) {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "timeZone must be a valid IANA timezone")
		return false
	}
	return true
}

func validateGCPShoppingMerchantIssueresolutionTriggerPayload(w http.ResponseWriter, path string, body map[string]any) (actionContext, actionFlowID string, ok bool) {
	actionContext = strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(body, "actionContext", "action_context"))
	if actionContext == "" {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "actionContext is required")
		return "", "", false
	}

	actionInput := gcpShoppingMerchantIssueresolutionMap(body, "actionInput", "action_input")
	if len(actionInput) == 0 {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "actionInput is required")
		return "", "", false
	}
	actionFlowID = strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(actionInput, "actionFlowId", "action_flow_id"))
	if actionFlowID == "" {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "actionInput.actionFlowId is required")
		return "", "", false
	}
	if actionFlowID != "flow-review" {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "actionInput.actionFlowId is unsupported")
		return "", "", false
	}

	inputValuesRaw := gcpShoppingMerchantIssueresolutionArray(actionInput, "inputValues", "input_values")
	if len(inputValuesRaw) == 0 {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "actionInput.inputValues is required")
		return "", "", false
	}

	hasExplanation := false
	for _, raw := range inputValuesRaw {
		valueMap, ok := raw.(map[string]any)
		if !ok {
			respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "actionInput.inputValues entries must be objects")
			return "", "", false
		}
		inputFieldID := strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(valueMap, "inputFieldId", "input_field_id"))
		if inputFieldID == "" {
			respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "inputFieldId is required for each input value")
			return "", "", false
		}

		hasTypedValue := false
		if textValueMap := gcpShoppingMerchantIssueresolutionMap(valueMap, "textInputValue", "text_input_value"); len(textValueMap) > 0 {
			hasTypedValue = true
			textValue := strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(textValueMap, "value"))
			if inputFieldID == "explanation" && textValue == "" {
				respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "explanation text value is required")
				return "", "", false
			}
			if inputFieldID == "explanation" && textValue != "" {
				hasExplanation = true
			}
		}
		if choiceValueMap := gcpShoppingMerchantIssueresolutionMap(valueMap, "choiceInputValue", "choice_input_value"); len(choiceValueMap) > 0 {
			hasTypedValue = true
			choiceID := strings.TrimSpace(gcpShoppingMerchantIssueresolutionString(choiceValueMap, "choiceInputOptionId", "choice_input_option_id"))
			if choiceID == "" {
				respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "choiceInputValue.choiceInputOptionId is required")
				return "", "", false
			}
		}
		if checkboxValueMap := gcpShoppingMerchantIssueresolutionMap(valueMap, "checkboxInputValue", "checkbox_input_value"); len(checkboxValueMap) > 0 {
			hasTypedValue = true
			if _, ok := checkboxValueMap["value"]; !ok {
				respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "checkboxInputValue.value is required")
				return "", "", false
			}
		}
		if !hasTypedValue {
			respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "each input value requires exactly one typed value")
			return "", "", false
		}
	}

	if !hasExplanation {
		respondGCPShoppingMerchantIssueresolutionInvalidArgument(w, path, "inputValues must include explanation text")
		return "", "", false
	}
	return actionContext, actionFlowID, true
}

func gcpShoppingMerchantIssueresolutionValidateEnumLikeValue(m map[string]any, keys ...string) bool {
	value, ok := gcpShoppingMerchantIssueresolutionAny(m, keys...)
	if !ok {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case float64:
		return typed >= 0
	case int:
		return typed >= 0
	case int64:
		return typed >= 0
	default:
		return false
	}
}

func gcpShoppingMerchantIssueresolutionHasSupportedFilter(filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	matches := gcpShoppingMerchantIssueresolutionFilterKeyRe.FindAllStringSubmatch(strings.ToLower(filter), -1)
	if len(matches) == 0 {
		return false
	}
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		key := strings.TrimSpace(match[1])
		if key != "reporting_context" && key != "country" {
			return false
		}
	}
	return true
}

func gcpShoppingMerchantIssueresolutionExtractFilterTerms(filter string) (reportingContext, country string) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", ""
	}
	reportingContext = strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantIssueresolutionCaptureGroup(filter, `(?i)reporting_context\s*=\s*\"?([A-Z_]+)\"?`)))
	country = strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantIssueresolutionCaptureGroup(filter, `(?i)country\s*=\s*\"?([A-Za-z]{2})\"?`)))
	return reportingContext, country
}

func gcpShoppingMerchantIssueresolutionCaptureGroup(value, expression string) string {
	re := regexp.MustCompile(expression)
	match := re.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func gcpShoppingMerchantIssueresolutionRenderedIssueFixture(scope, account, product string) map[string]any {
	title := "Account issue: business information missing"
	context := "ctx-account-review"
	content := "<p>Update your business profile information to restore product serving.</p>"
	impactMessage := "Disapproves 25 offers in 2 countries"
	if scope == "product" {
		title = fmt.Sprintf("Product issue: policy warning for %s", product)
		context = "ctx-product-review"
		content = fmt.Sprintf("<p>Fix product %s data and request another review.</p>", product)
		impactMessage = "Disapproves 5 offers in 1 country"
	}

	return map[string]any{
		"title":              title,
		"prerenderedContent": content,
		"impact": map[string]any{
			"message":  impactMessage,
			"severity": "ERROR",
			"breakdowns": []map[string]any{
				{
					"regions": []map[string]any{
						{"code": "US", "name": "United States"},
					},
					"details": []string{
						"Products not showing in Shopping ads",
						"Products not showing organically",
					},
				},
			},
		},
		"actions": []map[string]any{
			{
				"buttonLabel": "Open Merchant Center",
				"isAvailable": true,
				"externalAction": map[string]any{
					"type": 0,
					"uri":  fmt.Sprintf("https://merchants.google.com/mc/account/%s/issues", account),
				},
			},
			{
				"buttonLabel": "Request review",
				"isAvailable": true,
				"builtinUserInputAction": map[string]any{
					"actionContext": context,
					"flows": []map[string]any{
						{
							"id":                "flow-review",
							"label":             "I fixed the issue",
							"dialogTitle":       "Request another review",
							"dialogButtonLabel": "Request review",
							"inputs": []map[string]any{
								{
									"id":       "explanation",
									"required": true,
									"label": map[string]any{
										"simpleValue": "Explain what changed",
									},
									"textInput": map[string]any{
										"formatInfo": "Provide details of the fix.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func gcpShoppingMerchantIssueresolutionAggregateStatusFixture(account, id, reportingContext, country string) map[string]any {
	return map[string]any{
		"name":             fmt.Sprintf("accounts/%s/aggregateProductStatuses/%s", account, id),
		"reportingContext": strings.ToUpper(strings.TrimSpace(reportingContext)),
		"country":          strings.ToUpper(strings.TrimSpace(country)),
		"stats": map[string]any{
			"activeCount":      "120",
			"pendingCount":     "5",
			"disapprovedCount": "7",
			"expiringCount":    "3",
		},
		"itemLevelIssues": []map[string]any{
			{
				"code":             "MISSING_IDENTIFIER",
				"severity":         1,
				"resolution":       1,
				"attribute":        "gtin",
				"description":      "Missing product identifier",
				"detail":           "Provide GTIN to improve product quality and eligibility.",
				"documentationUri": "https://support.google.com/merchants/answer/7052112",
				"productCount":     "5",
			},
		},
	}
}

func gcpShoppingMerchantIssueresolutionMap(m map[string]any, keys ...string) map[string]any {
	value, ok := gcpShoppingMerchantIssueresolutionAny(m, keys...)
	if !ok {
		return nil
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return typed
}

func gcpShoppingMerchantIssueresolutionArray(m map[string]any, keys ...string) []any {
	value, ok := gcpShoppingMerchantIssueresolutionAny(m, keys...)
	if !ok {
		return nil
	}
	typed, ok := value.([]any)
	if !ok {
		return nil
	}
	return typed
}

func gcpShoppingMerchantIssueresolutionString(m map[string]any, keys ...string) string {
	value, ok := gcpShoppingMerchantIssueresolutionAny(m, keys...)
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func gcpShoppingMerchantIssueresolutionAny(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func respondGCPShoppingMerchantIssueresolutionInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantIssueresolutionAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantIssueresolutionNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantIssueresolutionFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantIssueResolution(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_issueresolution") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_issueresolution/sample",
			"service":   "shopping_merchant_issueresolution",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
