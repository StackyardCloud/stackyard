package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var gcpShoppingMerchantProductstudioAccountRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func (s *Server) handleGCPShoppingMerchantProductstudioRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantProductstudio(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantProductstudioPath(rawRequestPath(r))
	if !isGCPShoppingMerchantProductstudioPath(path, hasGCPShoppingMerchantProductstudioHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantProductstudioRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.productstudio.v1alpha.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodPost:
		if handleGCPShoppingMerchantProductstudioPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantProductstudioPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantProductstudioHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_productstudio",
		"shopping-merchant-productstudio",
		"shopping-merchant-productstudio-apiv1alpha",
		"shopping_merchant_productstudio_apiv1alpha",
		"merchant_productstudio",
		"merchant-productstudio",
		"merchantproductstudio",
		"gcp-shopping-merchant-productstudio":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-productstudio-apiv1alpha") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/productstudio")
}

func isGCPShoppingMerchantProductstudioPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/productstudio/v1alpha/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.productstudio.v1alpha.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/productstudio/v1alpha") {
		return true
	}
	return false
}

func gcpShoppingMerchantProductstudioRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/productstudio/v1alpha/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/productstudio/v1alpha/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantProductstudioPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, action, ok := parseGCPShoppingMerchantProductstudioActionPath(tail)
	if !ok {
		return false
	}

	body, ok := decodeGCPShoppingMerchantProductstudioJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	requestName := strings.TrimSpace(gcpShoppingMerchantProductstudioString(body, "name"))
	if requestName == "" {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "name is required")
		return true
	}
	nameAccount, validName := parseGCPShoppingMerchantProductstudioName(requestName)
	if !validName {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "name must be in accounts/{account} format")
		return true
	}
	if nameAccount != account {
		respondGCPShoppingMerchantProductstudioFailedPrecondition(w, path, "request body name must match request path account")
		return true
	}

	var (
		fixture           map[string]any
		fixtureErrType    string
		fixtureErrMessage string
	)

	switch action {
	case "generateProductImageBackground":
		fixture, fixtureErrType, fixtureErrMessage = buildGCPShoppingMerchantProductstudioImageFixture(account, "generateProductImageBackground", body)
	case "removeProductImageBackground":
		fixture, fixtureErrType, fixtureErrMessage = buildGCPShoppingMerchantProductstudioImageFixture(account, "removeProductImageBackground", body)
	case "upscaleProductImage":
		fixture, fixtureErrType, fixtureErrMessage = buildGCPShoppingMerchantProductstudioImageFixture(account, "upscaleProductImage", body)
	case "generateProductTextSuggestions":
		fixture, fixtureErrType, fixtureErrMessage = buildGCPShoppingMerchantProductstudioTextFixture(account, body)
	default:
		return false
	}

	if fixtureErrType != "" {
		respondGCPShoppingMerchantProductstudioByErrorType(w, path, fixtureErrType, fixtureErrMessage)
		return true
	}

	respondJSON(w, http.StatusOK, fixture)
	return true
}

func parseGCPShoppingMerchantProductstudioActionPath(tail string) (account, action string, ok bool) {
	tail = strings.Trim(strings.TrimSpace(tail), "/")
	if !strings.HasPrefix(tail, "accounts/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(tail, "accounts/")

	if accountPart, actionPart, found := strings.Cut(rest, "/"); found {
		account = strings.TrimSpace(accountPart)
		action = strings.TrimSpace(actionPart)
		if !gcpShoppingMerchantProductstudioAccountRe.MatchString(account) {
			return "", "", false
		}
		switch action {
		case "generatedImages:generateProductImageBackground":
			return account, "generateProductImageBackground", true
		case "generatedImages:removeProductImageBackground":
			return account, "removeProductImageBackground", true
		case "generatedImages:upscaleProductImage":
			return account, "upscaleProductImage", true
		default:
			return "", "", false
		}
	}

	accountPart, actionPart, found := strings.Cut(rest, ":")
	if !found {
		return "", "", false
	}
	account = strings.TrimSpace(accountPart)
	action = strings.TrimSpace(actionPart)
	if !gcpShoppingMerchantProductstudioAccountRe.MatchString(account) {
		return "", "", false
	}
	if action != "generateProductTextSuggestions" {
		return "", "", false
	}
	return account, action, true
}

func parseGCPShoppingMerchantProductstudioName(name string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantProductstudioAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func decodeGCPShoppingMerchantProductstudioJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func buildGCPShoppingMerchantProductstudioImageFixture(account, operation string, body map[string]any) (map[string]any, string, string) {
	if strings.Contains(strings.ToLower(account), "missing") {
		return nil, "NotFound", "account not found"
	}

	inputImage := gcpShoppingMerchantProductstudioMap(body, "inputImage", "input_image")
	if len(inputImage) == 0 {
		return nil, "InvalidArgument", "inputImage is required"
	}
	imageURI := strings.TrimSpace(gcpShoppingMerchantProductstudioString(inputImage, "imageUri", "image_uri"))
	imageBytes := strings.TrimSpace(gcpShoppingMerchantProductstudioString(inputImage, "imageBytes", "image_bytes"))
	if imageURI == "" && imageBytes == "" {
		return nil, "InvalidArgument", "inputImage must include imageUri or imageBytes"
	}
	if imageURI != "" && imageBytes != "" {
		return nil, "FailedPrecondition", "inputImage must include exactly one of imageUri or imageBytes"
	}
	if imageURI != "" {
		lower := strings.ToLower(imageURI)
		if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://") {
			return nil, "InvalidArgument", "inputImage.imageUri must be an http(s) URL"
		}
	}
	if imageBytes != "" {
		if _, err := base64.StdEncoding.DecodeString(imageBytes); err != nil {
			return nil, "InvalidArgument", "inputImage.imageBytes must be valid base64"
		}
	}

	if operation == "generateProductImageBackground" {
		config := gcpShoppingMerchantProductstudioMap(body, "config")
		if len(config) == 0 {
			return nil, "InvalidArgument", "config is required"
		}
		productDescription := strings.TrimSpace(gcpShoppingMerchantProductstudioString(config, "productDescription", "product_description"))
		backgroundDescription := strings.TrimSpace(gcpShoppingMerchantProductstudioString(config, "backgroundDescription", "background_description"))
		if productDescription == "" || backgroundDescription == "" {
			return nil, "InvalidArgument", "config.productDescription and config.backgroundDescription are required"
		}
	}

	if operation == "removeProductImageBackground" {
		config := gcpShoppingMerchantProductstudioMap(body, "config")
		if len(config) > 0 {
			backgroundColor := gcpShoppingMerchantProductstudioMap(config, "backgroundColor", "background_color")
			if len(backgroundColor) > 0 {
				for _, channel := range []string{"red", "green", "blue"} {
					value := gcpShoppingMerchantProductstudioInt64(backgroundColor, channel)
					if value < 0 || value > 255 {
						return nil, "InvalidArgument", "config.backgroundColor values must be in the range [0, 255]"
					}
				}
			}
		}
	}

	outputConfig := gcpShoppingMerchantProductstudioMap(body, "outputConfig", "output_config")
	returnImageURI := gcpShoppingMerchantProductstudioBool(outputConfig, "returnImageUri", "return_image_uri")

	seed := imageURI
	if seed == "" {
		seed = imageBytes
	}
	id := gcpShoppingMerchantProductstudioStableID(account, operation, seed)
	fixture := map[string]any{
		"generatedImage": map[string]any{
			"name":           fmt.Sprintf("accounts/%s/generatedImages/%s-%s", account, operation, id),
			"generationTime": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		},
	}
	generatedImage, _ := fixture["generatedImage"].(map[string]any)
	if returnImageURI {
		generatedImage["uri"] = fmt.Sprintf("https://stackyard.example/generated/%s/%s/%s.png", account, operation, id)
	} else {
		generatedImage["imageBytes"] = "c3RhY2t5YXJkLWdlbmVyYXRlZC1pbWFnZQ=="
	}
	return fixture, "", ""
}

func buildGCPShoppingMerchantProductstudioTextFixture(account string, body map[string]any) (map[string]any, string, string) {
	if strings.Contains(strings.ToLower(account), "missing") {
		return nil, "NotFound", "account not found"
	}

	productInfo := gcpShoppingMerchantProductstudioMap(body, "productInfo", "product_info")
	if len(productInfo) == 0 {
		return nil, "InvalidArgument", "productInfo is required"
	}

	productAttributes := gcpShoppingMerchantProductstudioStringMap(productInfo, "productAttributes", "product_attributes")
	if len(productAttributes) == 0 {
		return nil, "InvalidArgument", "productInfo.productAttributes must include at least one attribute"
	}

	outputSpec := gcpShoppingMerchantProductstudioMap(body, "outputSpec", "output_spec")
	workflowID := strings.ToLower(strings.TrimSpace(gcpShoppingMerchantProductstudioString(outputSpec, "workflowId", "workflow_id")))
	if workflowID == "" {
		workflowID = "title"
	}
	switch workflowID {
	case "title", "description", "tide":
	default:
		return nil, "InvalidArgument", "outputSpec.workflowId is unsupported"
	}

	tone := strings.ToLower(strings.TrimSpace(gcpShoppingMerchantProductstudioString(outputSpec, "tone")))
	if tone != "" {
		switch tone {
		case "playful", "formal", "persuasive", "conversational":
		default:
			return nil, "InvalidArgument", "outputSpec.tone is unsupported"
		}
	}

	targetLanguage := strings.ToLower(strings.TrimSpace(gcpShoppingMerchantProductstudioString(outputSpec, "targetLanguage", "target_language")))
	if targetLanguage == "" {
		targetLanguage = "en"
	}

	title := strings.TrimSpace(productAttributes["title"])
	if title == "" {
		title = strings.TrimSpace(productAttributes["product"])
	}
	if title == "" {
		title = "Merchant product"
	}
	brand := strings.TrimSpace(productAttributes["brand"])
	if brand == "" {
		brand = "Stackyard"
	}
	color := strings.TrimSpace(productAttributes["color"])
	if color == "" {
		color = "neutral"
	}

	tonePrefix := "Optimized"
	switch tone {
	case "playful":
		tonePrefix = "Fresh"
	case "formal":
		tonePrefix = "Refined"
	case "persuasive":
		tonePrefix = "Compelling"
	case "conversational":
		tonePrefix = "Friendly"
	}

	generatedTitle := strings.TrimSpace(fmt.Sprintf("%s %s %s", tonePrefix, brand, title))
	generatedDescription := fmt.Sprintf("High-quality %s %s item generated for workflow %s in %s.", color, title, workflowID, targetLanguage)

	fixture := map[string]any{
		"title": map[string]any{
			"text":          generatedTitle,
			"score":         0.93,
			"changeSummary": "Title normalized for merchant feeds",
		},
		"description": map[string]any{
			"text":          generatedDescription,
			"score":         0.88,
			"changeSummary": "Description enriched with key attributes",
		},
		"attributes": map[string]any{
			"workflow":       workflowID,
			"targetLanguage": targetLanguage,
			"brand":          brand,
		},
		"metadata": map[string]any{
			"metadata": map[string]any{
				"workflowId":    workflowID,
				"tone":          tone,
				"targetLang":    targetLanguage,
				"source":        "stackyard-staged-emulation",
				"attributeKeys": gcpShoppingMerchantProductstudioSortedKeys(productAttributes),
			},
		},
	}
	return fixture, "", ""
}

func gcpShoppingMerchantProductstudioStableID(parts ...string) string {
	hash := fnv.New32a()
	for _, part := range parts {
		_, _ = hash.Write([]byte(strings.ToLower(strings.TrimSpace(part))))
		_, _ = hash.Write([]byte{0})
	}
	return fmt.Sprintf("%08x", hash.Sum32())
}

func gcpShoppingMerchantProductstudioSortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func gcpShoppingMerchantProductstudioAny(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpShoppingMerchantProductstudioMap(m map[string]any, keys ...string) map[string]any {
	raw, ok := gcpShoppingMerchantProductstudioAny(m, keys...)
	if !ok {
		return nil
	}
	value, _ := raw.(map[string]any)
	return value
}

func gcpShoppingMerchantProductstudioStringMap(m map[string]any, keys ...string) map[string]string {
	raw, ok := gcpShoppingMerchantProductstudioAny(m, keys...)
	if !ok {
		return nil
	}
	rawMap, _ := raw.(map[string]any)
	out := make(map[string]string, len(rawMap))
	for key, value := range rawMap {
		switch typed := value.(type) {
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed != "" {
				out[key] = trimmed
			}
		case float64:
			out[key] = strconv.FormatFloat(typed, 'f', -1, 64)
		case int:
			out[key] = strconv.Itoa(typed)
		case int64:
			out[key] = strconv.FormatInt(typed, 10)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func gcpShoppingMerchantProductstudioString(m map[string]any, keys ...string) string {
	raw, ok := gcpShoppingMerchantProductstudioAny(m, keys...)
	if !ok {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	default:
		return ""
	}
}

func gcpShoppingMerchantProductstudioInt64(m map[string]any, keys ...string) int64 {
	raw, ok := gcpShoppingMerchantProductstudioAny(m, keys...)
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

func gcpShoppingMerchantProductstudioBool(m map[string]any, keys ...string) bool {
	raw, ok := gcpShoppingMerchantProductstudioAny(m, keys...)
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

func respondGCPShoppingMerchantProductstudioByErrorType(w http.ResponseWriter, path, errType, message string) {
	switch errType {
	case "NotFound":
		respondGCPShoppingMerchantProductstudioNotFound(w, path, message)
	case "FailedPrecondition":
		respondGCPShoppingMerchantProductstudioFailedPrecondition(w, path, message)
	case "Aborted":
		respondGCPShoppingMerchantProductstudioAborted(w, path, message)
	default:
		respondGCPShoppingMerchantProductstudioInvalidArgument(w, path, message)
	}
}

func respondGCPShoppingMerchantProductstudioInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantProductstudioFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantProductstudioNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantProductstudioAborted(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "Aborted",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantProductstudio(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_productstudio") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_productstudio/sample",
			"service":   "shopping_merchant_productstudio",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
