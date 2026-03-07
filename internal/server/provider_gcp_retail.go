package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var gcpRetailReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type gcpRetailParsedPath struct {
	resource string
	action   string
	segments []string
}

func (s *Server) handleGCPRetailRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_retail(w, r) {
		return true
	}

	path := rawRequestPath(r)
	normalizedPath := normalizeGCPRetailPath(path)
	if !isGCPRetailPath(normalizedPath, hasGCPRetailHint(r)) {
		return false
	}

	parsed, ok := parseGCPRetailPath(normalizedPath)
	if !ok {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if parsed.action != "" {
		if handleGCPRetailAction(w, r, path, parsed) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRetailListOperations(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetOperation(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailListCatalogs(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetCompletionConfig(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetAttributesConfig(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetServingConfig(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailListServingConfigs(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetControl(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailListControls(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetProduct(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailListProducts(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetModel(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailListModels(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailGetGenerativeQuestionsFeatureConfig(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailListGenerativeQuestionConfigs(w, r, path, parsed.resource) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRetailCreateProduct(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailCreateServingConfig(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailCreateControl(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailCreateModel(w, r, path, parsed.resource) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRetailUpdateCatalog(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateCompletionConfig(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateAttributesConfig(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateProduct(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateServingConfig(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateControl(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateModel(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateGenerativeQuestionsFeatureConfig(w, r, path, parsed.resource) {
			return true
		}
		if handleGCPRetailUpdateGenerativeQuestionConfig(w, r, path, parsed.resource) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRetailDeleteOperation(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailDeleteProduct(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailDeleteServingConfig(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailDeleteControl(w, path, parsed.resource) {
			return true
		}
		if handleGCPRetailDeleteModel(w, path, parsed.resource) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func handleGCPRetailAction(w http.ResponseWriter, r *http.Request, path string, parsed gcpRetailParsedPath) bool {
	switch r.Method {
	case http.MethodGet:
		if parsed.action == "getDefaultBranch" {
			return handleGCPRetailGetDefaultBranch(w, path, parsed.resource)
		}
	case http.MethodPost:
		switch parsed.action {
		case "setDefaultBranch":
			return handleGCPRetailSetDefaultBranch(w, r, path, parsed.resource)
		case "search":
			return handleGCPRetailSearch(w, r, path, parsed.resource)
		case "predict":
			return handleGCPRetailPredict(w, r, path, parsed.resource)
		case "completeQuery":
			return handleGCPRetailCompleteQuery(w, r, path, parsed.resource)
		case "import":
			if strings.HasSuffix(parsed.resource, "/products") {
				return handleGCPRetailImportProducts(w, r, path, parsed.resource)
			}
			if strings.HasSuffix(parsed.resource, "/completionData") {
				return handleGCPRetailImportCompletionData(w, r, path, parsed.resource)
			}
			if strings.HasSuffix(parsed.resource, "/userEvents") {
				return handleGCPRetailImportUserEvents(w, r, path, parsed.resource)
			}
		case "purge":
			if strings.HasSuffix(parsed.resource, "/products") {
				return handleGCPRetailPurgeProducts(w, r, path, parsed.resource)
			}
			if strings.HasSuffix(parsed.resource, "/userEvents") {
				return handleGCPRetailPurgeUserEvents(w, r, path, parsed.resource)
			}
		case "write":
			if strings.HasSuffix(parsed.resource, "/userEvents") {
				return handleGCPRetailWriteUserEvent(w, r, path, parsed.resource)
			}
		case "collect":
			if strings.HasSuffix(parsed.resource, "/userEvents") {
				return handleGCPRetailCollectUserEvent(w, r, path, parsed.resource)
			}
		case "rejoin":
			if strings.HasSuffix(parsed.resource, "/userEvents") {
				return handleGCPRetailRejoinUserEvents(w, r, path, parsed.resource)
			}
		case "setInventory":
			return handleGCPRetailSetInventory(w, r, path, parsed.resource)
		case "addFulfillmentPlaces":
			return handleGCPRetailAddFulfillmentPlaces(w, r, path, parsed.resource)
		case "removeFulfillmentPlaces":
			return handleGCPRetailRemoveFulfillmentPlaces(w, r, path, parsed.resource)
		case "addLocalInventories":
			return handleGCPRetailAddLocalInventories(w, r, path, parsed.resource)
		case "removeLocalInventories":
			return handleGCPRetailRemoveLocalInventories(w, r, path, parsed.resource)
		case "addControl":
			return handleGCPRetailAddControl(w, r, path, parsed.resource)
		case "removeControl":
			return handleGCPRetailRemoveControl(w, r, path, parsed.resource)
		case "pause":
			return handleGCPRetailPauseModel(w, path, parsed.resource)
		case "resume":
			return handleGCPRetailResumeModel(w, path, parsed.resource)
		case "tune":
			return handleGCPRetailTuneModel(w, r, path, parsed.resource)
		case "exportAnalyticsMetrics":
			return handleGCPRetailExportAnalyticsMetrics(w, r, path, parsed.resource)
		case "batchUpdate":
			if strings.HasSuffix(parsed.resource, "/generativeQuestion") {
				return handleGCPRetailBatchUpdateGenerativeQuestionConfigs(w, r, path, parsed.resource)
			}
		case "addCatalogAttribute":
			return handleGCPRetailAddCatalogAttribute(w, r, path, parsed.resource)
		case "removeCatalogAttribute":
			return handleGCPRetailRemoveCatalogAttribute(w, r, path, parsed.resource)
		case "replaceCatalogAttribute":
			return handleGCPRetailReplaceCatalogAttribute(w, r, path, parsed.resource)
		case "cancel":
			return handleGCPRetailCancelOperation(w, path, parsed.resource)
		}
	}
	return false
}

func hasGCPRetailHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "retail", "retail-apiv2", "vertex-ai-search-commerce", "vertex_ai_search_commerce", "vertexaisearchforcommerce":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-retail-apiv2") || strings.Contains(ua, "cloud.google.com/go/retail")
}

func isGCPRetailPath(path string, includeHint bool) bool {
	path = normalizeGCPRetailPath(path)
	if !strings.HasPrefix(path, "/gcp/v2/") {
		return false
	}
	resource := strings.Trim(strings.TrimPrefix(path, "/gcp/v2/"), "/")
	if resource == "" {
		return false
	}
	if strings.Contains(resource, "/catalogs") {
		return true
	}
	if strings.Contains(resource, "/placements/") && (strings.Contains(path, ":search") || strings.Contains(path, ":predict")) {
		return true
	}
	if includeHint && strings.HasPrefix(resource, "projects/") && strings.Contains(resource, "/locations/") {
		return true
	}
	return false
}

func parseGCPRetailPath(path string) (gcpRetailParsedPath, bool) {
	path = normalizeGCPRetailPath(path)
	if !strings.HasPrefix(path, "/gcp/v2/") {
		return gcpRetailParsedPath{}, false
	}
	resource := strings.Trim(strings.TrimPrefix(path, "/gcp/v2/"), "/")
	if resource == "" {
		return gcpRetailParsedPath{}, false
	}
	base, action, hasAction := strings.Cut(resource, ":")
	if hasAction {
		resource = strings.Trim(base, "/")
		action = strings.TrimSpace(action)
		if resource == "" || action == "" {
			return gcpRetailParsedPath{}, false
		}
	}
	segments := splitRetailPathSegments(resource)
	if len(segments) == 0 {
		return gcpRetailParsedPath{}, false
	}
	return gcpRetailParsedPath{resource: resource, action: action, segments: segments}, true
}

func handleGCPRetailListCatalogs(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, parent, ok := parseGCPRetailProjectLocationCatalogsParent(resource)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailCatalogFixture(project, location, "default_catalog", "Default Catalog"),
		gcpRetailCatalogFixture(project, location, "experiments_catalog", "Experiments Catalog"),
	}
	response, valid := gcpRetailPaginateList("catalogs", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func normalizeGCPRetailPath(path string) string {
	normalized := strings.ReplaceAll(path, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func handleGCPRetailUpdateCatalog(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, name, ok := parseGCPRetailCatalogName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "catalog.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "catalog.name must match the requested resource")
		return true
	}
	displayName := strings.TrimSpace(gcpRetailString(body, "displayName"))
	if displayName == "" {
		displayName = "Catalog " + catalogID
	}
	respondJSON(w, http.StatusOK, gcpRetailCatalogFixture(project, location, catalogID, displayName))
	return true
}

func handleGCPRetailSetDefaultBranch(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	_, _, _, catalogName, ok := parseGCPRetailCatalogName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	branch := strings.TrimSpace(gcpRetailString(body, "branch"))
	branchID := strings.TrimSpace(gcpRetailString(body, "branchId"))
	if branchID == "" {
		branchID = strings.TrimSpace(gcpRetailString(body, "branch_id"))
	}
	if branch == "" && branchID != "" {
		branch = catalogName + "/branches/" + branchID
	}
	if branch == "" {
		respondGCPRetailInvalidArgument(w, path, "branch or branchId is required")
		return true
	}
	expectedPrefix := catalogName + "/branches/"
	if !strings.HasPrefix(branch, expectedPrefix) {
		respondGCPRetailFailedPrecondition(w, path, "branch must belong to the requested catalog")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRetailGetDefaultBranch(w http.ResponseWriter, path, resource string) bool {
	_, _, _, catalogName, ok := parseGCPRetailCatalogName(resource)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"branch":  catalogName + "/branches/default_branch",
		"setTime": gcpRetailReferenceTime.Format(time.RFC3339),
		"note":    "stackyard default branch",
	})
	return true
}

func handleGCPRetailGetCompletionConfig(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "completionConfig")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailCompletionConfigFixture(project, location, catalogID))
	return true
}

func handleGCPRetailUpdateCompletionConfig(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, name, ok := parseGCPRetailCatalogSubresourceName(resource, "completionConfig")
	if !ok {
		return false
	}
	if _, valid := parseGCPRetailUpdateMask(w, r, path, map[string]struct{}{"matching_order": {}, "matchingOrder": {}, "max_suggestions": {}, "maxSuggestions": {}, "min_prefix_length": {}, "minPrefixLength": {}, "auto_learning": {}, "autoLearning": {}}); !valid {
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "completionConfig.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "completionConfig.name must match the requested resource")
		return true
	}
	fixture := gcpRetailCompletionConfigFixture(project, location, catalogID)
	if v := strings.TrimSpace(gcpRetailString(body, "matchingOrder")); v != "" {
		fixture["matchingOrder"] = v
	}
	if n := gcpRetailIntFromAny(body["maxSuggestions"]); n > 0 {
		fixture["maxSuggestions"] = n
	}
	if n := gcpRetailIntFromAny(body["minPrefixLength"]); n > 0 {
		fixture["minPrefixLength"] = n
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailGetAttributesConfig(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "attributesConfig")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailAttributesConfigFixture(project, location, catalogID))
	return true
}

func handleGCPRetailUpdateAttributesConfig(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, name, ok := parseGCPRetailCatalogSubresourceName(resource, "attributesConfig")
	if !ok {
		return false
	}
	if _, valid := parseGCPRetailUpdateMask(w, r, path, nil); !valid {
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "attributesConfig.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "attributesConfig.name must match the requested resource")
		return true
	}
	fixture := gcpRetailAttributesConfigFixture(project, location, catalogID)
	if attrs := gcpRetailBodyMap(body, "catalogAttributes"); len(attrs) > 0 {
		fixture["catalogAttributes"] = attrs
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailAddCatalogAttribute(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "attributesConfig")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	attribute := gcpRetailBodyMap(body, "catalogAttribute")
	if len(attribute) == 0 {
		respondGCPRetailInvalidArgument(w, path, "catalogAttribute is required")
		return true
	}
	key := strings.TrimSpace(gcpRetailString(attribute, "key"))
	if key == "" {
		respondGCPRetailInvalidArgument(w, path, "catalogAttribute.key is required")
		return true
	}
	fixture := gcpRetailAttributesConfigFixture(project, location, catalogID)
	catalogAttributes := gcpRetailBodyMap(fixture, "catalogAttributes")
	catalogAttributes[key] = attribute
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailRemoveCatalogAttribute(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "attributesConfig")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	key := strings.TrimSpace(gcpRetailString(body, "key"))
	if key == "" {
		key = strings.TrimSpace(gcpRetailString(body, "catalogAttributeKey"))
	}
	if key == "" {
		respondGCPRetailInvalidArgument(w, path, "catalog attribute key is required")
		return true
	}
	fixture := gcpRetailAttributesConfigFixture(project, location, catalogID)
	catalogAttributes := gcpRetailBodyMap(fixture, "catalogAttributes")
	delete(catalogAttributes, key)
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailReplaceCatalogAttribute(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "attributesConfig")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	attribute := gcpRetailBodyMap(body, "catalogAttribute")
	if len(attribute) == 0 {
		respondGCPRetailInvalidArgument(w, path, "catalogAttribute is required")
		return true
	}
	key := strings.TrimSpace(gcpRetailString(attribute, "key"))
	if key == "" {
		respondGCPRetailInvalidArgument(w, path, "catalogAttribute.key is required")
		return true
	}
	fixture := gcpRetailAttributesConfigFixture(project, location, catalogID)
	catalogAttributes := gcpRetailBodyMap(fixture, "catalogAttributes")
	catalogAttributes[key] = attribute
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailCreateProduct(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, parent, ok := parseGCPRetailBranchCollectionParent(resource, "products")
	if !ok {
		return false
	}
	productID := strings.TrimSpace(r.URL.Query().Get("productId"))
	if productID == "" {
		respondGCPRetailInvalidArgument(w, path, "productId is required")
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	title := strings.TrimSpace(gcpRetailString(body, "title"))
	if title == "" {
		respondGCPRetailInvalidArgument(w, path, "product.title is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailProductFixture(project, location, catalogID, branchID, productID, title))
	_ = parent
	return true
}

func handleGCPRetailGetProduct(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, branchID, productID, _, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailProductFixture(project, location, catalogID, branchID, productID, "Product "+productID))
	return true
}

func handleGCPRetailListProducts(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, parent, ok := parseGCPRetailBranchCollectionParent(resource, "products")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailProductFixture(project, location, catalogID, branchID, "product-1", "Stackyard Product 1"),
		gcpRetailProductFixture(project, location, catalogID, branchID, "product-2", "Stackyard Product 2"),
	}
	response, valid := gcpRetailPaginateList("products", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRetailUpdateProduct(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, productID, name, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	if _, valid := parseGCPRetailUpdateMask(w, r, path, map[string]struct{}{"title": {}, "description": {}, "categories": {}, "priceInfo": {}, "availability": {}, "attributes": {}}); !valid {
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "product.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "product.name must match the requested resource")
		return true
	}
	title := strings.TrimSpace(gcpRetailString(body, "title"))
	if title == "" {
		title = "Product " + productID
	}
	fixture := gcpRetailProductFixture(project, location, catalogID, branchID, productID, title)
	if description := strings.TrimSpace(gcpRetailString(body, "description")); description != "" {
		fixture["description"] = description
	}
	if availability := strings.TrimSpace(gcpRetailString(body, "availability")); availability != "" {
		fixture["availability"] = availability
	}
	if priceInfo := gcpRetailBodyMap(body, "priceInfo"); len(priceInfo) > 0 {
		fixture["priceInfo"] = priceInfo
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailDeleteProduct(w http.ResponseWriter, path, resource string) bool {
	if _, _, _, _, _, _, ok := parseGCPRetailProductName(resource); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRetailPurgeProducts(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	_, _, _, _, parent, ok := parseGCPRetailBranchCollectionParent(resource, "products")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpRetailString(body, "filter")) == "" {
		respondGCPRetailInvalidArgument(w, path, "filter is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "purge-products", false))
	return true
}

func handleGCPRetailImportProducts(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	_, _, _, _, parent, ok := parseGCPRetailBranchCollectionParent(resource, "products")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if inputConfig := gcpRetailBodyMap(body, "inputConfig"); len(inputConfig) == 0 {
		respondGCPRetailInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "import-products", false))
	return true
}

func handleGCPRetailSetInventory(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, productID, name, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	inventory := gcpRetailBodyMap(body, "inventory")
	if len(inventory) == 0 {
		respondGCPRetailInvalidArgument(w, path, "inventory is required")
		return true
	}
	if inventoryName := strings.TrimSpace(gcpRetailString(inventory, "name")); inventoryName != "" && inventoryName != name {
		respondGCPRetailInvalidArgument(w, path, "inventory.name must match the requested resource")
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "set-inventory-"+productID, false))
	return true
}

func handleGCPRetailAddFulfillmentPlaces(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, productID, name, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if productName := strings.TrimSpace(gcpRetailString(body, "product")); productName != "" && productName != name {
		respondGCPRetailInvalidArgument(w, path, "product must match the requested resource")
		return true
	}
	if len(gcpRetailStringSlice(body["placeIds"])) == 0 {
		respondGCPRetailInvalidArgument(w, path, "placeIds must contain at least one entry")
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "add-fulfillment-"+productID, false))
	return true
}

func handleGCPRetailRemoveFulfillmentPlaces(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, productID, name, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if productName := strings.TrimSpace(gcpRetailString(body, "product")); productName != "" && productName != name {
		respondGCPRetailInvalidArgument(w, path, "product must match the requested resource")
		return true
	}
	if len(gcpRetailStringSlice(body["placeIds"])) == 0 {
		respondGCPRetailInvalidArgument(w, path, "placeIds must contain at least one entry")
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "remove-fulfillment-"+productID, false))
	return true
}

func handleGCPRetailAddLocalInventories(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, productID, name, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if productName := strings.TrimSpace(gcpRetailString(body, "product")); productName != "" && productName != name {
		respondGCPRetailInvalidArgument(w, path, "product must match the requested resource")
		return true
	}
	if len(gcpRetailAnySlice(body["localInventories"])) == 0 {
		respondGCPRetailInvalidArgument(w, path, "localInventories must contain at least one entry")
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "add-local-inventories-"+productID, false))
	return true
}

func handleGCPRetailRemoveLocalInventories(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, branchID, productID, name, ok := parseGCPRetailProductName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if productName := strings.TrimSpace(gcpRetailString(body, "product")); productName != "" && productName != name {
		respondGCPRetailInvalidArgument(w, path, "product must match the requested resource")
		return true
	}
	if len(gcpRetailStringSlice(body["placeIds"])) == 0 {
		respondGCPRetailInvalidArgument(w, path, "placeIds must contain at least one entry")
		return true
	}
	parent := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "remove-local-inventories-"+productID, false))
	return true
}

func handleGCPRetailSearch(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, placementID, placement, ok := parseGCPRetailPlacementName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	query := strings.TrimSpace(gcpRetailString(body, "query"))
	if query == "" {
		respondGCPRetailInvalidArgument(w, path, "query is required")
		return true
	}
	product := gcpRetailProductFixture(project, location, catalogID, "default_branch", "product-1", "Search Product")
	respondJSON(w, http.StatusOK, map[string]any{
		"results": []any{
			map[string]any{
				"id":                   "product-1",
				"product":              product,
				"matchingVariantCount": 1,
			},
		},
		"totalSize":                  1,
		"correctedQuery":             query,
		"attributionToken":           "stackyard-search-token",
		"nextPageToken":              "",
		"appliedControls":            []string{},
		"redirectUri":                "",
		"invalidConditionBoostSpecs": []any{},
		"placement":                  placement,
		"placementId":                placementID,
	})
	return true
}

func handleGCPRetailPredict(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, _, ok := parseGCPRetailPlacementName(resource)
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	userEvent := gcpRetailBodyMap(body, "userEvent")
	if len(userEvent) == 0 {
		respondGCPRetailInvalidArgument(w, path, "userEvent is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"results": []any{
			map[string]any{
				"id":       "product-1",
				"metadata": map[string]any{"score": 0.95},
				"product": map[string]any{
					"name": fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/default_branch/products/product-1", project, location, catalogID),
				},
			},
		},
		"attributionToken": "stackyard-predict-token",
		"missingIds":       []any{},
		"validateOnly":     false,
	})
	return true
}

func handleGCPRetailCreateServingConfig(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, parent, ok := parseGCPRetailCatalogCollectionParent(resource, "servingConfigs")
	if !ok {
		return false
	}
	servingConfigID := strings.TrimSpace(r.URL.Query().Get("servingConfigId"))
	if servingConfigID == "" {
		respondGCPRetailInvalidArgument(w, path, "servingConfigId is required")
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	displayName := strings.TrimSpace(gcpRetailString(body, "displayName"))
	if displayName == "" {
		respondGCPRetailInvalidArgument(w, path, "servingConfig.displayName is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailServingConfigFixture(project, location, catalogID, servingConfigID, displayName))
	_ = parent
	return true
}

func handleGCPRetailGetServingConfig(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, servingConfigID, _, ok := parseGCPRetailCatalogResourceName(resource, "servingConfigs")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailServingConfigFixture(project, location, catalogID, servingConfigID, "Serving Config "+servingConfigID))
	return true
}

func handleGCPRetailListServingConfigs(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, parent, ok := parseGCPRetailCatalogCollectionParent(resource, "servingConfigs")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailServingConfigFixture(project, location, catalogID, "default_config", "Default Config"),
		gcpRetailServingConfigFixture(project, location, catalogID, "boosted_config", "Boosted Config"),
	}
	response, valid := gcpRetailPaginateList("servingConfigs", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRetailUpdateServingConfig(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, servingConfigID, name, ok := parseGCPRetailCatalogResourceName(resource, "servingConfigs")
	if !ok {
		return false
	}
	if _, valid := parseGCPRetailUpdateMask(w, r, path, nil); !valid {
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "servingConfig.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "servingConfig.name must match the requested resource")
		return true
	}
	displayName := strings.TrimSpace(gcpRetailString(body, "displayName"))
	if displayName == "" {
		displayName = "Serving Config " + servingConfigID
	}
	respondJSON(w, http.StatusOK, gcpRetailServingConfigFixture(project, location, catalogID, servingConfigID, displayName))
	return true
}

func handleGCPRetailDeleteServingConfig(w http.ResponseWriter, path, resource string) bool {
	if _, _, _, _, _, ok := parseGCPRetailCatalogResourceName(resource, "servingConfigs"); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRetailAddControl(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, servingConfigID, name, ok := parseGCPRetailCatalogResourceName(resource, "servingConfigs")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	controlID := strings.TrimSpace(gcpRetailString(body, "controlId"))
	if controlID == "" {
		respondGCPRetailInvalidArgument(w, path, "controlId is required")
		return true
	}
	if servingConfig := strings.TrimSpace(gcpRetailString(body, "servingConfig")); servingConfig != "" && servingConfig != name {
		respondGCPRetailInvalidArgument(w, path, "servingConfig must match the requested resource")
		return true
	}
	fixture := gcpRetailServingConfigFixture(project, location, catalogID, servingConfigID, "Serving Config "+servingConfigID)
	fixture["boostControlIds"] = []any{controlID}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailRemoveControl(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, servingConfigID, name, ok := parseGCPRetailCatalogResourceName(resource, "servingConfigs")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	controlID := strings.TrimSpace(gcpRetailString(body, "controlId"))
	if controlID == "" {
		respondGCPRetailInvalidArgument(w, path, "controlId is required")
		return true
	}
	if servingConfig := strings.TrimSpace(gcpRetailString(body, "servingConfig")); servingConfig != "" && servingConfig != name {
		respondGCPRetailInvalidArgument(w, path, "servingConfig must match the requested resource")
		return true
	}
	fixture := gcpRetailServingConfigFixture(project, location, catalogID, servingConfigID, "Serving Config "+servingConfigID)
	fixture["boostControlIds"] = []any{}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailCompleteQuery(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogName(resource); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	query := strings.TrimSpace(gcpRetailString(body, "query"))
	if query == "" {
		respondGCPRetailInvalidArgument(w, path, "query is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"completionResults": []any{
			map[string]any{"suggestion": query + " product"},
		},
		"attributionToken": "stackyard-complete-token",
		"recentSearchResults": []any{
			map[string]any{"recentSearch": query},
		},
		"attributeResults": map[string]any{},
	})
	return true
}

func handleGCPRetailImportCompletionData(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogSubresourceName(resource, "completionData"); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if inputConfig := gcpRetailBodyMap(body, "inputConfig"); len(inputConfig) == 0 {
		respondGCPRetailInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "import-completion-data", false))
	return true
}

func handleGCPRetailCreateControl(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogCollectionParent(resource, "controls")
	if !ok {
		return false
	}
	controlID := strings.TrimSpace(r.URL.Query().Get("controlId"))
	if controlID == "" {
		respondGCPRetailInvalidArgument(w, path, "controlId is required")
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	displayName := strings.TrimSpace(gcpRetailString(body, "displayName"))
	if displayName == "" {
		respondGCPRetailInvalidArgument(w, path, "control.displayName is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailControlFixture(project, location, catalogID, controlID, displayName))
	return true
}

func handleGCPRetailGetControl(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, controlID, _, ok := parseGCPRetailCatalogResourceName(resource, "controls")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailControlFixture(project, location, catalogID, controlID, "Control "+controlID))
	return true
}

func handleGCPRetailListControls(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, parent, ok := parseGCPRetailCatalogCollectionParent(resource, "controls")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailControlFixture(project, location, catalogID, "control-1", "Control One"),
		gcpRetailControlFixture(project, location, catalogID, "control-2", "Control Two"),
	}
	response, valid := gcpRetailPaginateList("controls", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRetailUpdateControl(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, controlID, name, ok := parseGCPRetailCatalogResourceName(resource, "controls")
	if !ok {
		return false
	}
	if _, valid := parseGCPRetailUpdateMask(w, r, path, nil); !valid {
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "control.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "control.name must match the requested resource")
		return true
	}
	displayName := strings.TrimSpace(gcpRetailString(body, "displayName"))
	if displayName == "" {
		displayName = "Control " + controlID
	}
	respondJSON(w, http.StatusOK, gcpRetailControlFixture(project, location, catalogID, controlID, displayName))
	return true
}

func handleGCPRetailDeleteControl(w http.ResponseWriter, path, resource string) bool {
	if _, _, _, _, _, ok := parseGCPRetailCatalogResourceName(resource, "controls"); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRetailWriteUserEvent(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "userEvents")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	userEvent := gcpRetailBodyMap(body, "userEvent")
	if len(userEvent) == 0 {
		respondGCPRetailInvalidArgument(w, path, "userEvent is required")
		return true
	}
	if strings.TrimSpace(gcpRetailString(userEvent, "eventType")) == "" {
		respondGCPRetailInvalidArgument(w, path, "userEvent.eventType is required")
		return true
	}
	if strings.TrimSpace(gcpRetailString(userEvent, "visitorId")) == "" {
		userInfo := gcpRetailBodyMap(userEvent, "userInfo")
		if strings.TrimSpace(gcpRetailString(userInfo, "visitorId")) == "" {
			respondGCPRetailInvalidArgument(w, path, "userEvent.visitorId or userEvent.userInfo.visitorId is required")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpRetailUserEventFixture(project, location, catalogID, userEvent))
	return true
}

func handleGCPRetailCollectUserEvent(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogSubresourceName(resource, "userEvents"); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpRetailString(body, "userEvent")) == "" {
		respondGCPRetailInvalidArgument(w, path, "userEvent is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"contentType": "application/json",
		"data":        base64.StdEncoding.EncodeToString([]byte(`{"status":"ok"}`)),
	})
	return true
}

func handleGCPRetailPurgeUserEvents(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogSubresourceName(resource, "userEvents"); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpRetailString(body, "filter")) == "" {
		respondGCPRetailInvalidArgument(w, path, "filter is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "purge-user-events", false))
	return true
}

func handleGCPRetailImportUserEvents(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogSubresourceName(resource, "userEvents"); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if inputConfig := gcpRetailBodyMap(body, "inputConfig"); len(inputConfig) == 0 {
		respondGCPRetailInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "import-user-events", false))
	return true
}

func handleGCPRetailRejoinUserEvents(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogSubresourceName(resource, "userEvents"); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpRetailString(body, "rejoinScope")) == "" && strings.TrimSpace(gcpRetailString(body, "userEventRejoinScope")) == "" {
		respondGCPRetailInvalidArgument(w, path, "userEventRejoinScope is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "rejoin-user-events", false))
	return true
}

func handleGCPRetailCreateModel(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, parent, ok := parseGCPRetailCatalogCollectionParent(resource, "models")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	model := gcpRetailBodyMap(body, "model")
	if len(model) == 0 {
		model = body
	}
	displayName := strings.TrimSpace(gcpRetailString(model, "displayName"))
	if displayName == "" {
		respondGCPRetailInvalidArgument(w, path, "model.displayName is required")
		return true
	}
	modelID := strings.TrimSpace(r.URL.Query().Get("modelId"))
	if modelID == "" {
		if modelName := strings.TrimSpace(gcpRetailString(model, "name")); modelName != "" {
			if _, _, _, parsedModelID, _, parsed := parseGCPRetailCatalogResourceName(modelName, "models"); parsed {
				modelID = parsedModelID
			}
		}
	}
	if modelID == "" {
		modelID = "model-1"
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, "create-model-"+modelID, false))
	_ = project
	_ = location
	_ = catalogID
	return true
}

func handleGCPRetailGetModel(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, modelID, _, ok := parseGCPRetailCatalogResourceName(resource, "models")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailModelFixture(project, location, catalogID, modelID, "Model "+modelID))
	return true
}

func handleGCPRetailListModels(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, parent, ok := parseGCPRetailCatalogCollectionParent(resource, "models")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailModelFixture(project, location, catalogID, "model-1", "Model One"),
		gcpRetailModelFixture(project, location, catalogID, "model-2", "Model Two"),
	}
	response, valid := gcpRetailPaginateList("models", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRetailUpdateModel(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, modelID, name, ok := parseGCPRetailCatalogResourceName(resource, "models")
	if !ok {
		return false
	}
	if _, valid := parseGCPRetailUpdateMask(w, r, path, nil); !valid {
		return true
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	bodyName := strings.TrimSpace(gcpRetailString(body, "name"))
	if bodyName == "" {
		respondGCPRetailInvalidArgument(w, path, "model.name is required")
		return true
	}
	if bodyName != name {
		respondGCPRetailInvalidArgument(w, path, "model.name must match the requested resource")
		return true
	}
	displayName := strings.TrimSpace(gcpRetailString(body, "displayName"))
	if displayName == "" {
		displayName = "Model " + modelID
	}
	respondJSON(w, http.StatusOK, gcpRetailModelFixture(project, location, catalogID, modelID, displayName))
	return true
}

func handleGCPRetailDeleteModel(w http.ResponseWriter, path, resource string) bool {
	if _, _, _, _, _, ok := parseGCPRetailCatalogResourceName(resource, "models"); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "delete-model", false))
	return true
}

func handleGCPRetailPauseModel(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, modelID, _, ok := parseGCPRetailCatalogResourceName(resource, "models")
	if !ok {
		return false
	}
	fixture := gcpRetailModelFixture(project, location, catalogID, modelID, "Model "+modelID)
	fixture["trainingState"] = "PAUSED"
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailResumeModel(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, modelID, _, ok := parseGCPRetailCatalogResourceName(resource, "models")
	if !ok {
		return false
	}
	fixture := gcpRetailModelFixture(project, location, catalogID, modelID, "Model "+modelID)
	fixture["trainingState"] = "TRAINING"
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailTuneModel(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, _, ok := parseGCPRetailCatalogResourceName(resource, "models"); !ok {
		return false
	}
	if _, valid := decodeGCPRetailJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "tune-model", false))
	return true
}

func handleGCPRetailExportAnalyticsMetrics(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	if _, _, _, _, ok := parseGCPRetailCatalogName(resource); !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	if outputConfig := gcpRetailBodyMap(body, "outputConfig"); len(outputConfig) == 0 {
		respondGCPRetailInvalidArgument(w, path, "outputConfig is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(resource, "export-analytics", false))
	return true
}

func handleGCPRetailGetGenerativeQuestionsFeatureConfig(w http.ResponseWriter, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "generativeQuestionFeature")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRetailGenerativeQuestionsFeatureConfigFixture(project, location, catalogID))
	return true
}

func handleGCPRetailUpdateGenerativeQuestionsFeatureConfig(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, name, ok := parseGCPRetailCatalogSubresourceName(resource, "generativeQuestionFeature")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	catalog := strings.TrimSpace(gcpRetailString(body, "catalog"))
	if catalog == "" {
		respondGCPRetailInvalidArgument(w, path, "generativeQuestionsFeatureConfig.catalog is required")
		return true
	}
	if catalog != strings.TrimSuffix(name, "/generativeQuestionFeature") {
		respondGCPRetailInvalidArgument(w, path, "generativeQuestionsFeatureConfig.catalog must match the requested resource")
		return true
	}
	fixture := gcpRetailGenerativeQuestionsFeatureConfigFixture(project, location, catalogID)
	if enabled, ok := body["featureEnabled"].(bool); ok {
		fixture["featureEnabled"] = enabled
	}
	if min := gcpRetailIntFromAny(body["minimumProducts"]); min >= 0 {
		fixture["minimumProducts"] = min
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailListGenerativeQuestionConfigs(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, parent, ok := parseGCPRetailCatalogCollectionParent(resource, "generativeQuestions")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailGenerativeQuestionConfigFixture(project, location, catalogID, "brand"),
		gcpRetailGenerativeQuestionConfigFixture(project, location, catalogID, "color"),
	}
	response, valid := gcpRetailPaginateList("generativeQuestionConfigs", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["parent"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRetailUpdateGenerativeQuestionConfig(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, name, ok := parseGCPRetailCatalogSubresourceName(resource, "generativeQuestion")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	catalog := strings.TrimSpace(gcpRetailString(body, "catalog"))
	if catalog == "" {
		respondGCPRetailInvalidArgument(w, path, "generativeQuestionConfig.catalog is required")
		return true
	}
	if catalog != strings.TrimSuffix(name, "/generativeQuestion") {
		respondGCPRetailInvalidArgument(w, path, "generativeQuestionConfig.catalog must match the requested resource")
		return true
	}
	facet := strings.TrimSpace(gcpRetailString(body, "facet"))
	if facet == "" {
		respondGCPRetailInvalidArgument(w, path, "generativeQuestionConfig.facet is required")
		return true
	}
	fixture := gcpRetailGenerativeQuestionConfigFixture(project, location, catalogID, facet)
	if finalQuestion := strings.TrimSpace(gcpRetailString(body, "finalQuestion")); finalQuestion != "" {
		fixture["finalQuestion"] = finalQuestion
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func handleGCPRetailBatchUpdateGenerativeQuestionConfigs(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	project, location, catalogID, _, ok := parseGCPRetailCatalogSubresourceName(resource, "generativeQuestion")
	if !ok {
		return false
	}
	body, valid := decodeGCPRetailJSONBody(w, r, path)
	if !valid {
		return true
	}
	requests := gcpRetailAnySlice(body["requests"])
	if len(requests) == 0 {
		respondGCPRetailInvalidArgument(w, path, "requests must contain at least one entry")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"generativeQuestionConfigs": []any{
			gcpRetailGenerativeQuestionConfigFixture(project, location, catalogID, "brand"),
		},
	})
	return true
}

func handleGCPRetailListOperations(w http.ResponseWriter, r *http.Request, path, resource string) bool {
	parent, ok := parseGCPRetailOperationsCollection(resource)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRetailPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRetailOperationFixture(parent, "op-1", false),
		gcpRetailOperationFixture(parent, "op-2", true),
	}
	response, valid := gcpRetailPaginateList("operations", items, pageSize, start, path)
	if !valid {
		return true
	}
	response["name"] = parent
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRetailGetOperation(w http.ResponseWriter, path, resource string) bool {
	parent, opID, _, ok := parseGCPRetailOperationName(resource)
	if !ok {
		return false
	}
	done := strings.Contains(strings.ToLower(opID), "done") || strings.HasSuffix(opID, "-2")
	respondJSON(w, http.StatusOK, gcpRetailOperationFixture(parent, opID, done))
	return true
}

func handleGCPRetailCancelOperation(w http.ResponseWriter, path, resource string) bool {
	if _, _, _, ok := parseGCPRetailOperationName(resource); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRetailDeleteOperation(w http.ResponseWriter, path, resource string) bool {
	if _, _, _, ok := parseGCPRetailOperationName(resource); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPRetailProjectLocationParent(resource string) (project, location, parent string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", "", false
	}
	return project, location, fmt.Sprintf("projects/%s/locations/%s", project, location), true
}

func parseGCPRetailProjectLocationCatalogsParent(resource string) (project, location, parent string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 5 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", "", false
	}
	return project, location, fmt.Sprintf("projects/%s/locations/%s", project, location), true
}

func parseGCPRetailCatalogName(resource string) (project, location, catalogID, name string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || catalogID == "" {
		return "", "", "", "", false
	}
	name = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID)
	return project, location, catalogID, name, true
}

func parseGCPRetailCatalogCollectionParent(resource, collection string) (project, location, catalogID, parent string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 7 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != collection {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || catalogID == "" {
		return "", "", "", "", false
	}
	parent = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID)
	return project, location, catalogID, parent, true
}

func parseGCPRetailCatalogResourceName(resource, collection string) (project, location, catalogID, resourceID, name string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != collection {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	resourceID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || catalogID == "" || resourceID == "" {
		return "", "", "", "", "", false
	}
	name = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/%s/%s", project, location, catalogID, collection, resourceID)
	return project, location, catalogID, resourceID, name, true
}

func parseGCPRetailCatalogSubresourceName(resource, subresource string) (project, location, catalogID, name string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 7 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != subresource {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || catalogID == "" {
		return "", "", "", "", false
	}
	name = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/%s", project, location, catalogID, subresource)
	return project, location, catalogID, name, true
}

func parseGCPRetailBranchCollectionParent(resource, collection string) (project, location, catalogID, branchID, parent string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 9 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != "branches" || parts[8] != collection {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	branchID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || catalogID == "" || branchID == "" {
		return "", "", "", "", "", false
	}
	parent = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	return project, location, catalogID, branchID, parent, true
}

func parseGCPRetailProductName(resource string) (project, location, catalogID, branchID, productID, name string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != "branches" || parts[8] != "products" {
		return "", "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	branchID = strings.TrimSpace(parts[7])
	productID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || catalogID == "" || branchID == "" || productID == "" {
		return "", "", "", "", "", "", false
	}
	name = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s/products/%s", project, location, catalogID, branchID, productID)
	return project, location, catalogID, branchID, productID, name, true
}

func parseGCPRetailPlacementName(resource string) (project, location, catalogID, placementID, name string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != "placements" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	placementID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || catalogID == "" || placementID == "" {
		return "", "", "", "", "", false
	}
	name = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/placements/%s", project, location, catalogID, placementID)
	return project, location, catalogID, placementID, name, true
}

func parseGCPRetailOperationsCollection(resource string) (parent string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) < 2 || parts[len(parts)-1] != "operations" {
		return "", false
	}
	parentParts := parts[:len(parts)-1]
	if len(parentParts) == 0 {
		return "", false
	}
	parent = strings.Join(parentParts, "/")
	return parent, true
}

func parseGCPRetailOperationName(resource string) (parent, operationID, name string, ok bool) {
	parts := splitRetailPathSegments(resource)
	if len(parts) < 3 || parts[len(parts)-2] != "operations" {
		return "", "", "", false
	}
	operationID = strings.TrimSpace(parts[len(parts)-1])
	if operationID == "" {
		return "", "", "", false
	}
	parentParts := parts[:len(parts)-2]
	if len(parentParts) == 0 {
		return "", "", "", false
	}
	parent = strings.Join(parentParts, "/")
	name = parent + "/operations/" + operationID
	return parent, operationID, name, true
}

func decodeGCPRetailJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		respondGCPRetailInvalidArgument(w, path, "request body must be readable")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return map[string]any{}, true
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil {
		respondGCPRetailInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, true
}

func parseGCPRetailPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 50
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPRetailInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > 1000 {
			value = 1000
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPRetailInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func parseGCPRetailUpdateMask(w http.ResponseWriter, r *http.Request, path string, allowed map[string]struct{}) ([]string, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if raw == "" {
		respondGCPRetailInvalidArgument(w, path, "updateMask is required")
		return nil, false
	}
	parts := strings.Split(raw, ",")
	mask := make([]string, 0, len(parts))
	for _, item := range parts {
		field := strings.TrimSpace(item)
		if field == "" {
			continue
		}
		if allowed != nil {
			if _, exists := allowed[field]; !exists {
				respondGCPRetailInvalidArgument(w, path, "updateMask contains unsupported field: "+field)
				return nil, false
			}
		}
		mask = append(mask, field)
	}
	if len(mask) == 0 {
		respondGCPRetailInvalidArgument(w, path, "updateMask must contain at least one field")
		return nil, false
	}
	return mask, true
}

func gcpRetailPaginateList(key string, items []map[string]any, pageSize, start int, path string) (map[string]any, bool) {
	if start > len(items) {
		return map[string]any{}, false
	}
	if pageSize < 0 {
		pageSize = 0
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	list := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		list = append(list, item)
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return map[string]any{
		key:             list,
		"nextPageToken": next,
		"path":          path,
	}, true
}

func gcpRetailCatalogFixture(project, location, catalogID, displayName string) map[string]any {
	if displayName == "" {
		displayName = "Catalog " + catalogID
	}
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID),
		"displayName": displayName,
		"productLevelConfig": map[string]any{
			"ingestionProductType":         "primary",
			"merchantCenterProductIdField": "offerId",
		},
	}
}

func gcpRetailCompletionConfigFixture(project, location, catalogID string) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/completionConfig", project, location, catalogID)
	return map[string]any{
		"name":                           name,
		"matchingOrder":                  "out-of-order",
		"maxSuggestions":                 10,
		"minPrefixLength":                1,
		"autoLearning":                   true,
		"lastSuggestionsImportOperation": name + "/operations/import-suggestions",
	}
}

func gcpRetailAttributesConfigFixture(project, location, catalogID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/attributesConfig", project, location, catalogID),
		"catalogAttributes": map[string]any{
			"color": map[string]any{
				"key":              "color",
				"inUse":            true,
				"searchableOption": "SEARCHABLE_ENABLED",
				"indexableOption":  "INDEXABLE_ENABLED",
			},
		},
		"attributeConfigLevel": "CATALOG_LEVEL_ATTRIBUTE_CONFIG",
	}
}

func gcpRetailProductFixture(project, location, catalogID, branchID, productID, title string) map[string]any {
	if title == "" {
		title = "Product " + productID
	}
	return map[string]any{
		"name":         fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s/products/%s", project, location, catalogID, branchID, productID),
		"id":           productID,
		"type":         "PRIMARY",
		"title":        title,
		"description":  "Stackyard retail product",
		"categories":   []any{"Apparel > Tops"},
		"languageCode": "en-US",
		"availability": "IN_STOCK",
		"priceInfo": map[string]any{
			"currencyCode": "USD",
			"price":        19.99,
		},
		"attributes": map[string]any{
			"colorFamilies": map[string]any{"text": []any{"blue"}},
		},
		"publishTime": gcpRetailReferenceTime.Format(time.RFC3339),
	}
}

func gcpRetailServingConfigFixture(project, location, catalogID, servingConfigID, displayName string) map[string]any {
	if displayName == "" {
		displayName = "Serving Config " + servingConfigID
	}
	return map[string]any{
		"name":                fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/servingConfigs/%s", project, location, catalogID, servingConfigID),
		"displayName":         displayName,
		"modelId":             "recommended-for-you",
		"priceRerankingLevel": "no-price-reranking",
		"facetControlIds":     []any{},
		"boostControlIds":     []any{},
		"filterControlIds":    []any{},
		"redirectControlIds":  []any{},
	}
}

func gcpRetailControlFixture(project, location, catalogID, controlID, displayName string) map[string]any {
	if displayName == "" {
		displayName = "Control " + controlID
	}
	return map[string]any{
		"name":                  fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/controls/%s", project, location, catalogID, controlID),
		"displayName":           displayName,
		"solutionTypes":         []any{"SOLUTION_TYPE_SEARCH"},
		"searchSolutionUseCase": []any{"SEARCH_SOLUTION_USE_CASE_SEARCH"},
		"rule": map[string]any{
			"replacementAction": map[string]any{
				"queryTerms":       []any{map[string]any{"term": "sneeker"}},
				"replacementTerms": []any{map[string]any{"term": "sneaker"}},
			},
		},
	}
}

func gcpRetailModelFixture(project, location, catalogID, modelID, displayName string) map[string]any {
	if displayName == "" {
		displayName = "Model " + modelID
	}
	name := fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/models/%s", project, location, catalogID, modelID)
	return map[string]any{
		"name":                  name,
		"displayName":           displayName,
		"trainingState":         "TRAINING",
		"servingState":          "ACTIVE",
		"createTime":            gcpRetailReferenceTime.Format(time.RFC3339),
		"updateTime":            gcpRetailReferenceTime.Format(time.RFC3339),
		"type":                  "recommended-for-you",
		"optimizationObjective": "ctr",
		"periodicTuningState":   "PERIODIC_TUNING_ENABLED",
	}
}

func gcpRetailOperationFixture(parent, operationID string, done bool) map[string]any {
	if parent == "" {
		parent = "projects/stackyard/locations/global/catalogs/default_catalog"
	}
	if operationID == "" {
		operationID = "op-1"
	}
	name := parent + "/operations/" + operationID
	response := map[string]any{
		"name": name,
		"done": done,
		"metadata": map[string]any{
			"target": parent,
		},
	}
	if done {
		response["response"] = map[string]any{}
	}
	return response
}

func gcpRetailUserEventFixture(project, location, catalogID string, userEvent map[string]any) map[string]any {
	eventType := strings.TrimSpace(gcpRetailString(userEvent, "eventType"))
	if eventType == "" {
		eventType = "detail-page-view"
	}
	visitorID := strings.TrimSpace(gcpRetailString(userEvent, "visitorId"))
	if visitorID == "" {
		userInfo := gcpRetailBodyMap(userEvent, "userInfo")
		visitorID = strings.TrimSpace(gcpRetailString(userInfo, "visitorId"))
	}
	if visitorID == "" {
		visitorID = "visitor-1"
	}
	return map[string]any{
		"eventType":        eventType,
		"visitorId":        visitorID,
		"attributionToken": "stackyard-search-token",
		"eventTime":        gcpRetailReferenceTime.Format(time.RFC3339),
		"uri":              "https://example.com/catalogs/" + catalogID,
		"userInfo": map[string]any{
			"visitorId": visitorID,
		},
		"productDetails": []any{
			map[string]any{"product": map[string]any{"id": "product-1"}},
		},
		"catalog": fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID),
	}
}

func gcpRetailGenerativeQuestionsFeatureConfigFixture(project, location, catalogID string) map[string]any {
	return map[string]any{
		"catalog":         fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID),
		"featureEnabled":  true,
		"minimumProducts": 1,
	}
}

func gcpRetailGenerativeQuestionConfigFixture(project, location, catalogID, facet string) map[string]any {
	if facet == "" {
		facet = "brand"
	}
	return map[string]any{
		"catalog":               fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID),
		"facet":                 facet,
		"generatedQuestion":     "What " + facet + " do you prefer?",
		"finalQuestion":         "Preferred " + facet + "?",
		"exampleValues":         []any{"stackyard"},
		"frequency":             0.5,
		"allowedInConversation": true,
	}
}

func gcpRetailBodyMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	value, ok := m[key]
	if !ok {
		return map[string]any{}
	}
	out, ok := value.(map[string]any)
	if !ok || out == nil {
		return map[string]any{}
	}
	return out
}

func gcpRetailString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, _ := raw.(string)
	return value
}

func gcpRetailStringSlice(v any) []string {
	values, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, entry := range values {
		if str, ok := entry.(string); ok && strings.TrimSpace(str) != "" {
			out = append(out, strings.TrimSpace(str))
		}
	}
	return out
}

func gcpRetailAnySlice(v any) []any {
	values, ok := v.([]any)
	if !ok {
		return nil
	}
	return values
}

func gcpRetailIntFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		return 0
	}
}

func splitRetailPathSegments(path string) []string {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return nil
	}
	parts := strings.Split(clean, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func respondGCPRetailInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPRetailFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_retail(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "retail") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPRetailInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/catalogs/default_catalog",
			"service":  "retail",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
