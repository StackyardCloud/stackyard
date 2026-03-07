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
	gcpShoppingMerchantReviewsAccountRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantReviewsReviewIDRe   = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	gcpShoppingMerchantReviewsDataSourceRe = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

func (s *Server) handleGCPShoppingMerchantReviewsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantReviews(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantReviewsPath(rawRequestPath(r))
	if !isGCPShoppingMerchantReviewsPath(path, hasGCPShoppingMerchantReviewsHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantReviewsRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.reviews.v1beta.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPShoppingMerchantReviewsGET(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPShoppingMerchantReviewsPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPShoppingMerchantReviewsDELETE(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func normalizeGCPShoppingMerchantReviewsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantReviewsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_reviews",
		"shopping-merchant-reviews",
		"shopping-merchant-reviews-apiv1beta",
		"shopping_merchant_reviews_apiv1beta",
		"merchant_reviews",
		"merchant-reviews",
		"merchantreviews",
		"gcp-shopping-merchant-reviews":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-reviews-apiv1beta") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/reviews")
}

func isGCPShoppingMerchantReviewsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/reviews/v1beta/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.reviews.v1beta.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/reviews/v1beta") {
		return true
	}
	return false
}

func gcpShoppingMerchantReviewsRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/reviews/v1beta/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/reviews/v1beta/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantReviewsGET(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	if account, collection, reviewID, ok := parseGCPShoppingMerchantReviewsResourcePath(tail); ok {
		return handleGCPShoppingMerchantReviewsGetResource(w, path, account, collection, reviewID)
	}
	if account, collection, ok := parseGCPShoppingMerchantReviewsCollectionPath(tail); ok {
		return handleGCPShoppingMerchantReviewsList(w, r, path, account, collection)
	}
	return false
}

func handleGCPShoppingMerchantReviewsPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, collection, ok := parseGCPShoppingMerchantReviewsInsertPath(tail)
	if !ok {
		return false
	}
	return handleGCPShoppingMerchantReviewsInsert(w, r, path, account, collection)
}

func handleGCPShoppingMerchantReviewsDELETE(w http.ResponseWriter, _ *http.Request, path, tail string) bool {
	account, collection, reviewID, ok := parseGCPShoppingMerchantReviewsResourcePath(tail)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(account), "missing") || strings.Contains(strings.ToLower(reviewID), "missing") {
		respondGCPShoppingMerchantReviewsNotFound(w, path, "review not found")
		return true
	}
	if !isGCPShoppingMerchantReviewsCollection(collection) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPShoppingMerchantReviewsGetResource(w http.ResponseWriter, path, account, collection, reviewID string) bool {
	if strings.Contains(strings.ToLower(account), "missing") || strings.Contains(strings.ToLower(reviewID), "missing") {
		respondGCPShoppingMerchantReviewsNotFound(w, path, "review not found")
		return true
	}
	dataSource := fmt.Sprintf("accounts/%s/dataSources/104628", account)
	switch collection {
	case "merchantReviews":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantReviewsMerchantFixture(account, reviewID, dataSource))
		return true
	case "productReviews":
		respondJSON(w, http.StatusOK, gcpShoppingMerchantReviewsProductFixture(account, reviewID, dataSource))
		return true
	default:
		return false
	}
}

func handleGCPShoppingMerchantReviewsList(w http.ResponseWriter, r *http.Request, path, account, collection string) bool {
	if strings.Contains(strings.ToLower(account), "missing") {
		respondGCPShoppingMerchantReviewsNotFound(w, path, "account not found")
		return true
	}
	pageSize, start, ok := parseGCPShoppingMerchantReviewsPagination(w, r, path)
	if !ok {
		return true
	}

	dataSource := fmt.Sprintf("accounts/%s/dataSources/104628", account)
	switch collection {
	case "merchantReviews":
		items := []map[string]any{
			gcpShoppingMerchantReviewsMerchantFixture(account, "merchant-review-1001", dataSource),
			gcpShoppingMerchantReviewsMerchantFixture(account, "merchant-review-1002", dataSource),
			gcpShoppingMerchantReviewsMerchantFixture(account, "merchant-review-1003", dataSource),
		}
		if start > len(items) {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "pageToken is out of range")
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
			"merchantReviews": items[start:end],
			"nextPageToken":   next,
		})
		return true
	case "productReviews":
		items := []map[string]any{
			gcpShoppingMerchantReviewsProductFixture(account, "product-review-1001", dataSource),
			gcpShoppingMerchantReviewsProductFixture(account, "product-review-1002", dataSource),
			gcpShoppingMerchantReviewsProductFixture(account, "product-review-1003", dataSource),
		}
		if start > len(items) {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "pageToken is out of range")
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
			"productReviews": items[start:end],
			"nextPageToken":  next,
		})
		return true
	default:
		return false
	}
}

func handleGCPShoppingMerchantReviewsInsert(w http.ResponseWriter, r *http.Request, path, account, collection string) bool {
	if strings.Contains(strings.ToLower(account), "missing") {
		respondGCPShoppingMerchantReviewsNotFound(w, path, "account not found")
		return true
	}

	body, ok := decodeGCPShoppingMerchantReviewsJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	dataSource := strings.TrimSpace(r.URL.Query().Get("dataSource"))
	if dataSource == "" {
		dataSource = gcpShoppingMerchantReviewsString(body, "dataSource", "data_source")
	}
	dsAccount, _, dsOK := parseGCPShoppingMerchantReviewsDataSource(dataSource)
	if !dsOK {
		respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "dataSource must be in accounts/{account}/dataSources/{datasource} format")
		return true
	}
	if dsAccount != account {
		respondGCPShoppingMerchantReviewsFailedPrecondition(w, path, "dataSource account must match URL account")
		return true
	}

	switch collection {
	case "merchantReviews":
		review, ok := gcpShoppingMerchantReviewsMap(body, "merchantReview", "merchant_review")
		if !ok {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "merchantReview is required")
			return true
		}
		reviewID := gcpShoppingMerchantReviewsString(review, "merchantReviewId", "merchant_review_id")
		if reviewID == "" {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "merchantReview.merchantReviewId is required")
			return true
		}
		if strings.Contains(strings.ToLower(reviewID), "missing") {
			respondGCPShoppingMerchantReviewsNotFound(w, path, "review not found")
			return true
		}
		if name := gcpShoppingMerchantReviewsString(review, "name"); name != "" {
			nameAccount, nameID, ok := parseGCPShoppingMerchantReviewsMerchantName(name)
			if !ok {
				respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "merchantReview.name must be in accounts/{account}/merchantReviews/{merchantReview} format")
				return true
			}
			if nameAccount != account || nameID != reviewID {
				respondGCPShoppingMerchantReviewsFailedPrecondition(w, path, "merchantReview name must match requested account and merchantReviewId")
				return true
			}
		}
		fixture := gcpShoppingMerchantReviewsMerchantFixture(account, reviewID, dataSource)
		if attrs, ok := gcpShoppingMerchantReviewsMap(review, "merchantReviewAttributes", "merchant_review_attributes"); ok {
			fixtureAttrs, _ := fixture["merchantReviewAttributes"].(map[string]any)
			for _, key := range []string{
				"title", "content", "merchantId", "merchantDisplayName", "merchantLink", "merchantRatingLink",
				"reviewerId", "reviewerUsername", "reviewLanguage", "reviewCountry",
			} {
				if value := gcpShoppingMerchantReviewsString(attrs, key); value != "" {
					fixtureAttrs[key] = value
				}
			}
		}
		respondJSON(w, http.StatusOK, fixture)
		return true
	case "productReviews":
		review, ok := gcpShoppingMerchantReviewsMap(body, "productReview", "product_review")
		if !ok {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "productReview is required")
			return true
		}
		reviewID := gcpShoppingMerchantReviewsString(review, "productReviewId", "product_review_id")
		if reviewID == "" {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "productReview.productReviewId is required")
			return true
		}
		if strings.Contains(strings.ToLower(reviewID), "missing") {
			respondGCPShoppingMerchantReviewsNotFound(w, path, "review not found")
			return true
		}
		if name := gcpShoppingMerchantReviewsString(review, "name"); name != "" {
			nameAccount, nameID, ok := parseGCPShoppingMerchantReviewsProductName(name)
			if !ok {
				respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "productReview.name must be in accounts/{account}/productReviews/{productReview} format")
				return true
			}
			if nameAccount != account || nameID != reviewID {
				respondGCPShoppingMerchantReviewsFailedPrecondition(w, path, "productReview name must match requested account and productReviewId")
				return true
			}
		}
		fixture := gcpShoppingMerchantReviewsProductFixture(account, reviewID, dataSource)
		if attrs, ok := gcpShoppingMerchantReviewsMap(review, "productReviewAttributes", "product_review_attributes"); ok {
			fixtureAttrs, _ := fixture["productReviewAttributes"].(map[string]any)
			for _, key := range []string{
				"title", "content", "reviewLanguage", "reviewCountry", "publisherName", "reviewerId", "reviewerUsername",
			} {
				if value := gcpShoppingMerchantReviewsString(attrs, key); value != "" {
					fixtureAttrs[key] = value
				}
			}
		}
		respondJSON(w, http.StatusOK, fixture)
		return true
	default:
		return false
	}
}

func parseGCPShoppingMerchantReviewsCollectionPath(tail string) (account, collection string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	collection = strings.TrimSpace(parts[2])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) || !isGCPShoppingMerchantReviewsCollection(collection) {
		return "", "", false
	}
	return account, collection, true
}

func parseGCPShoppingMerchantReviewsInsertPath(tail string) (account, collection string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	action := strings.TrimSpace(parts[2])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) {
		return "", "", false
	}
	switch action {
	case "merchantReviews:insert":
		return account, "merchantReviews", true
	case "productReviews:insert":
		return account, "productReviews", true
	default:
		return "", "", false
	}
}

func parseGCPShoppingMerchantReviewsResourcePath(tail string) (account, collection, reviewID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	collection = strings.TrimSpace(parts[2])
	reviewID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) ||
		!isGCPShoppingMerchantReviewsCollection(collection) ||
		!gcpShoppingMerchantReviewsReviewIDRe.MatchString(reviewID) {
		return "", "", "", false
	}
	return account, collection, reviewID, true
}

func parseGCPShoppingMerchantReviewsParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantReviewsMerchantName(name string) (account, reviewID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "merchantReviews" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	reviewID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) || !gcpShoppingMerchantReviewsReviewIDRe.MatchString(reviewID) {
		return "", "", false
	}
	return account, reviewID, true
}

func parseGCPShoppingMerchantReviewsProductName(name string) (account, reviewID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "productReviews" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	reviewID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) || !gcpShoppingMerchantReviewsReviewIDRe.MatchString(reviewID) {
		return "", "", false
	}
	return account, reviewID, true
}

func parseGCPShoppingMerchantReviewsDataSource(dataSource string) (account, sourceID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(dataSource), "/"), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "dataSources" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	sourceID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantReviewsAccountRe.MatchString(account) || !gcpShoppingMerchantReviewsDataSourceRe.MatchString(sourceID) {
		return "", "", false
	}
	return account, sourceID, true
}

func parseGCPShoppingMerchantReviewsPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 100
	if pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize")); pageSizeRaw != "" {
		parsed, err := strconv.Atoi(pageSizeRaw)
		if err != nil || parsed < 0 {
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "pageSize must be a non-negative integer")
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
			respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = parsed
	}
	return pageSize, start, true
}

func decodeGCPShoppingMerchantReviewsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantReviewsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpShoppingMerchantReviewsString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			return strings.TrimSpace(value)
		case float64:
			return strconv.FormatInt(int64(value), 10)
		case int:
			return strconv.Itoa(value)
		case int64:
			return strconv.FormatInt(value, 10)
		}
	}
	return ""
}

func gcpShoppingMerchantReviewsMap(body map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		value, ok := raw.(map[string]any)
		if ok {
			return value, true
		}
	}
	return nil, false
}

func gcpShoppingMerchantReviewsMerchantFixture(account, reviewID, dataSource string) map[string]any {
	return map[string]any{
		"name":             fmt.Sprintf("accounts/%s/merchantReviews/%s", account, reviewID),
		"merchantReviewId": reviewID,
		"dataSource":       dataSource,
		"merchantReviewAttributes": map[string]any{
			"merchantId":          "merchant-" + account,
			"merchantDisplayName": "Stackyard Merchant " + account,
			"merchantLink":        fmt.Sprintf("https://merchant.stackyard.example/%s", account),
			"merchantRatingLink":  fmt.Sprintf("https://merchant.stackyard.example/%s/reviews", account),
			"minRating":           "1",
			"maxRating":           "5",
			"rating":              4.8,
			"title":               "Great merchant experience",
			"content":             "Order arrived quickly and as described.",
			"reviewerId":          "reviewer-" + reviewID,
			"reviewerUsername":    "stackyard-user",
			"collectionMethod":    "AFTER_FULFILLMENT",
			"reviewTime":          time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC).Format(time.RFC3339),
			"reviewLanguage":      "en",
			"reviewCountry":       "US",
		},
		"customAttributes": []map[string]any{
			{
				"name":  "channel",
				"value": "online",
			},
		},
	}
}

func gcpShoppingMerchantReviewsProductFixture(account, reviewID, dataSource string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("accounts/%s/productReviews/%s", account, reviewID),
		"productReviewId": reviewID,
		"dataSource":      dataSource,
		"productReviewAttributes": map[string]any{
			"aggregatorName":   "Stackyard Reviews Aggregator",
			"publisherName":    "Stackyard",
			"reviewLanguage":   "en",
			"reviewCountry":    "US",
			"reviewTime":       time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC).Format(time.RFC3339),
			"title":            "Excellent product quality",
			"content":          "Fabric and fit exceeded expectations.",
			"pros":             []string{"Great quality", "Fast shipping"},
			"cons":             []string{"Limited colors"},
			"reviewLink":       map[string]any{"type": "SINGLETON", "link": "https://merchant.stackyard.example/reviews/" + reviewID},
			"minRating":        "1",
			"maxRating":        "5",
			"rating":           4.6,
			"productNames":     []string{"Stackyard Tee"},
			"productLinks":     []string{"https://merchant.stackyard.example/products/stackyard-tee"},
			"skus":             []string{"sku-1001"},
			"brands":           []string{"Stackyard"},
			"collectionMethod": "POST_FULFILLMENT",
			"transactionId":    "txn-" + reviewID,
		},
		"customAttributes": []map[string]any{
			{
				"name":  "segment",
				"value": "apparel",
			},
		},
	}
}

func isGCPShoppingMerchantReviewsCollection(collection string) bool {
	switch strings.TrimSpace(collection) {
	case "merchantReviews", "productReviews":
		return true
	default:
		return false
	}
}

func respondGCPShoppingMerchantReviewsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantReviewsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantReviewsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantReviews(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_reviews") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_reviews/sample",
			"service":   "shopping_merchant_reviews",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
