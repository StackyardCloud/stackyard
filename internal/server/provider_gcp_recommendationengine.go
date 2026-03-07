package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var gcpRecommendationEngineReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPRecommendationEngineRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_recommendationengine(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPRecommendationEnginePath(path, hasGCPRecommendationEngineHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRecommendationEngineListCatalogItems(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineGetCatalogItem(w, path) {
			return true
		}
		if handleGCPRecommendationEngineCollectUserEvent(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineListUserEvents(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineListPredictionAPIKeyRegistrations(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRecommendationEngineCreateCatalogItem(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineImportCatalogItems(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineWriteUserEvent(w, r, path) {
			return true
		}
		if handleGCPRecommendationEnginePurgeUserEvents(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineImportUserEvents(w, r, path) {
			return true
		}
		if handleGCPRecommendationEnginePredict(w, r, path) {
			return true
		}
		if handleGCPRecommendationEngineCreatePredictionAPIKeyRegistration(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRecommendationEngineUpdateCatalogItem(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRecommendationEngineDeleteCatalogItem(w, path) {
			return true
		}
		if handleGCPRecommendationEngineDeletePredictionAPIKeyRegistration(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPRecommendationEngineHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "recommendationengine" || service == "recommendation-engine" || service == "recommendations-ai" {
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-recommendationengine-apiv1beta1") || strings.Contains(ua, "recommendationengine")
}

func isGCPRecommendationEnginePath(path string, includeOperations bool) bool {
	_, _, _, tail, ok := parseGCPRecommendationCatalogPath(path)
	if ok {
		if isGCPRecommendationCatalogItemsCollectionTail(tail) ||
			isGCPRecommendationCatalogItemTail(tail) ||
			isGCPRecommendationCatalogItemsImportTail(tail) {
			return true
		}
		if includeOperations && isGCPRecommendationOperationTail(tail) {
			return true
		}
	}

	_, _, _, _, tail, ok = parseGCPRecommendationEventStorePath(path)
	if ok {
		if isGCPRecommendationWriteUserEventTail(tail) ||
			isGCPRecommendationCollectUserEventTail(tail) ||
			isGCPRecommendationUserEventsCollectionTail(tail) ||
			isGCPRecommendationPurgeUserEventsTail(tail) ||
			isGCPRecommendationImportUserEventsTail(tail) ||
			isGCPRecommendationPredictionAPIKeyRegistrationsCollectionTail(tail) ||
			isGCPRecommendationPredictionAPIKeyRegistrationTail(tail) ||
			isGCPRecommendationPredictTail(tail) {
			return true
		}
	}

	_, _, _, _, ok = parseGCPRecommendationPredictPath(path)
	return ok
}

func handleGCPRecommendationEngineCreateCatalogItem(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, parent, tail, ok := parseGCPRecommendationCatalogPath(path)
	if !ok || !isGCPRecommendationCatalogItemsCollectionTail(tail) {
		return false
	}
	item, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	id := gcpRecommendationEngineString(item, "id")
	title := gcpRecommendationEngineString(item, "title")
	if id == "" || title == "" {
		respondGCPRecommendationEngineInvalidArgument(w, path, "catalogItem.id and catalogItem.title are required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineCatalogItemFixture(project, catalog, parent, id, title))
	return true
}

func handleGCPRecommendationEngineListCatalogItems(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, parent, tail, ok := parseGCPRecommendationCatalogPath(path)
	if !ok || !isGCPRecommendationCatalogItemsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRecommendationPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRecommendationEngineCatalogItemFixture(project, catalog, parent, "item-1", "Stackyard Item 1"),
		gcpRecommendationEngineCatalogItemFixture(project, catalog, parent, "item-2", "Stackyard Item 2"),
	}
	response, valid := gcpRecommendationEnginePaginateList("catalogItems", items, pageSize, start, path)
	if !valid {
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommendationEngineGetCatalogItem(w http.ResponseWriter, path string) bool {
	project, catalog, parent, itemID, ok := parseGCPRecommendationCatalogItemPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineCatalogItemFixture(project, catalog, parent, itemID, "Stackyard Item "+itemID))
	return true
}

func handleGCPRecommendationEngineUpdateCatalogItem(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, parent, itemID, ok := parseGCPRecommendationCatalogItemPath(path)
	if !ok {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPRecommendationEngineInvalidArgument(w, path, "updateMask is required")
		return true
	}
	item, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if gotID := gcpRecommendationEngineString(item, "id"); gotID == "" || gotID != itemID {
		respondGCPRecommendationEngineInvalidArgument(w, path, "catalogItem.id must match the requested resource")
		return true
	}
	title := gcpRecommendationEngineString(item, "title")
	if title == "" {
		title = "Stackyard Item " + itemID
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineCatalogItemFixture(project, catalog, parent, itemID, title))
	return true
}

func handleGCPRecommendationEngineDeleteCatalogItem(w http.ResponseWriter, path string) bool {
	if _, _, _, _, ok := parseGCPRecommendationCatalogItemPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRecommendationEngineImportCatalogItems(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, _, tail, ok := parseGCPRecommendationCatalogPath(path)
	if !ok || !isGCPRecommendationCatalogItemsImportTail(tail) {
		return false
	}
	body, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	inputConfig := gcpRecommendationEngineBodyMap(body, "inputConfig")
	if len(inputConfig) == 0 {
		respondGCPRecommendationEngineInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	if !gcpRecommendationEngineValidInputConfigSource(inputConfig) {
		respondGCPRecommendationEngineInvalidArgument(w, path, "inputConfig source is required")
		return true
	}
	opID := "importCatalogItems-1"
	if reqID := gcpRecommendationEngineString(body, "requestId"); reqID != "" {
		opID = "importCatalogItems-" + reqID
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineOperationFixture(project, catalog, opID, false))
	return true
}

func handleGCPRecommendationEngineWriteUserEvent(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, eventStore, parent, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationWriteUserEventTail(tail) {
		return false
	}
	event, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if !gcpRecommendationEngineValidUserEvent(event) {
		respondGCPRecommendationEngineInvalidArgument(w, path, "userEvent.eventType and userEvent.userInfo.visitorId are required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineUserEventFixture(project, catalog, eventStore, parent, event))
	return true
}

func handleGCPRecommendationEngineCollectUserEvent(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, _, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationCollectUserEventTail(tail) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("userEvent")) == "" {
		respondGCPRecommendationEngineInvalidArgument(w, path, "userEvent query parameter is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "provider": providerGCP})
	return true
}

func handleGCPRecommendationEngineListUserEvents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, eventStore, parent, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationUserEventsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRecommendationPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRecommendationEngineUserEventFixture(project, catalog, eventStore, parent, map[string]any{}),
		gcpRecommendationEngineUserEventFixture(project, catalog, eventStore, parent, map[string]any{"eventType": "search", "userInfo": map[string]any{"visitorId": "visitor-2"}}),
	}
	response, valid := gcpRecommendationEnginePaginateList("userEvents", items, pageSize, start, path)
	if !valid {
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommendationEnginePurgeUserEvents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, _, _, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationPurgeUserEventsTail(tail) {
		return false
	}
	body, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpRecommendationEngineString(body, "filter")) == "" {
		respondGCPRecommendationEngineInvalidArgument(w, path, "filter is required")
		return true
	}
	opID := "purgeUserEvents-1"
	if force, exists := body["force"]; exists {
		if b, ok := force.(bool); ok && b {
			opID = "purgeUserEvents-force"
		}
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineOperationFixture(project, catalog, opID, false))
	return true
}

func handleGCPRecommendationEngineImportUserEvents(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, _, _, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationImportUserEventsTail(tail) {
		return false
	}
	body, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	inputConfig := gcpRecommendationEngineBodyMap(body, "inputConfig")
	if len(inputConfig) == 0 {
		respondGCPRecommendationEngineInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	if !gcpRecommendationEngineValidInputConfigSource(inputConfig) {
		respondGCPRecommendationEngineInvalidArgument(w, path, "inputConfig source is required")
		return true
	}
	opID := "importUserEvents-1"
	if reqID := gcpRecommendationEngineString(body, "requestId"); reqID != "" {
		opID = "importUserEvents-" + reqID
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineOperationFixture(project, catalog, opID, false))
	return true
}

func handleGCPRecommendationEnginePredict(w http.ResponseWriter, r *http.Request, path string) bool {
	project, catalog, eventStore, placement, ok := parseGCPRecommendationPredictPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	userEvent := gcpRecommendationEngineBodyMap(body, "userEvent")
	if !gcpRecommendationEngineValidUserEvent(userEvent) {
		respondGCPRecommendationEngineInvalidArgument(w, path, "userEvent.eventType and userEvent.userInfo.visitorId are required")
		return true
	}
	pageSize, start, valid := parseGCPRecommendationBodyPagination(w, path, body)
	if !valid {
		return true
	}
	results := []map[string]any{
		gcpRecommendationEnginePredictionResultFixture(project, catalog, eventStore, placement, "item-1", 0.97),
		gcpRecommendationEnginePredictionResultFixture(project, catalog, eventStore, placement, "item-2", 0.91),
	}
	response, valid := gcpRecommendationEnginePaginateList("results", results, pageSize, start, path)
	if !valid {
		return true
	}
	response["recommendationToken"] = "rec-token-1"
	response["itemsMissingInCatalog"] = []any{}
	response["dryRun"] = gcpRecommendationEngineBool(body, "dryRun")
	response["metadata"] = map[string]any{"placement": placement}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommendationEngineCreatePredictionAPIKeyRegistration(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, parent, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationPredictionAPIKeyRegistrationsCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPRecommendationEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if requestParent := gcpRecommendationEngineString(body, "parent"); requestParent != "" && requestParent != parent {
		respondGCPRecommendationEngineInvalidArgument(w, path, "parent must match the requested resource")
		return true
	}
	registration := gcpRecommendationEngineBodyMap(body, "predictionApiKeyRegistration")
	if len(registration) == 0 {
		respondGCPRecommendationEngineInvalidArgument(w, path, "predictionApiKeyRegistration is required")
		return true
	}
	apiKey := gcpRecommendationEngineString(registration, "apiKey")
	if apiKey == "" {
		respondGCPRecommendationEngineInvalidArgument(w, path, "predictionApiKeyRegistration.apiKey is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{"apiKey": apiKey})
	return true
}

func handleGCPRecommendationEngineListPredictionAPIKeyRegistrations(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, _, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationPredictionAPIKeyRegistrationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRecommendationPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{"apiKey": "stackyard-api-key"},
		{"apiKey": "stackyard-api-key-2"},
	}
	response, valid := gcpRecommendationEnginePaginateList("predictionApiKeyRegistrations", items, pageSize, start, path)
	if !valid {
		return true
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPRecommendationEngineDeletePredictionAPIKeyRegistration(w http.ResponseWriter, path string) bool {
	if _, _, _, _, _, ok := parseGCPRecommendationPredictionAPIKeyRegistrationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRecommendationEngineGetOperation(w http.ResponseWriter, path string) bool {
	project, catalog, operationID, ok := parseGCPRecommendationOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRecommendationEngineOperationFixture(project, catalog, operationID, true))
	return true
}

func parseGCPRecommendationPathSegments(path string) ([]string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[0] != "gcp" || parts[1] != "v1beta1" {
		return nil, false
	}
	return parts[2:], true
}

func parseGCPRecommendationCatalogPath(path string) (project, catalog, parent string, tail []string, ok bool) {
	segments, ok := parseGCPRecommendationPathSegments(path)
	if !ok || len(segments) < 6 {
		return "", "", "", nil, false
	}
	project, catalog, parent, ok = parseGCPRecommendationCatalogParentName(strings.Join(segments[:6], "/"))
	if !ok {
		return "", "", "", nil, false
	}
	return project, catalog, parent, segments[6:], true
}

func parseGCPRecommendationCatalogItemPath(path string) (project, catalog, parent, itemID string, ok bool) {
	project, catalog, parent, tail, ok := parseGCPRecommendationCatalogPath(path)
	if !ok || !isGCPRecommendationCatalogItemTail(tail) {
		return "", "", "", "", false
	}
	return project, catalog, parent, strings.TrimSpace(tail[1]), true
}

func parseGCPRecommendationEventStorePath(path string) (project, catalog, eventStore, parent string, tail []string, ok bool) {
	segments, ok := parseGCPRecommendationPathSegments(path)
	if !ok || len(segments) < 8 {
		return "", "", "", "", nil, false
	}
	project, catalog, eventStore, parent, ok = parseGCPRecommendationEventStoreParentName(strings.Join(segments[:8], "/"))
	if !ok {
		return "", "", "", "", nil, false
	}
	return project, catalog, eventStore, parent, segments[8:], true
}

func parseGCPRecommendationPredictPath(path string) (project, catalog, eventStore, placement string, ok bool) {
	project, catalog, eventStore, _, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || len(tail) != 2 || tail[0] != "placements" {
		return "", "", "", "", false
	}
	suffix := strings.TrimSpace(tail[1])
	parts := strings.Split(suffix, ":")
	if len(parts) != 2 || parts[1] != "predict" || strings.TrimSpace(parts[0]) == "" {
		return "", "", "", "", false
	}
	return project, catalog, eventStore, strings.TrimSpace(parts[0]), true
}

func parseGCPRecommendationPredictionAPIKeyRegistrationPath(path string) (project, catalog, eventStore, parent, registrationID string, ok bool) {
	project, catalog, eventStore, parent, tail, ok := parseGCPRecommendationEventStorePath(path)
	if !ok || !isGCPRecommendationPredictionAPIKeyRegistrationTail(tail) {
		return "", "", "", "", "", false
	}
	return project, catalog, eventStore, parent, strings.TrimSpace(tail[1]), true
}

func parseGCPRecommendationOperationPath(path string) (project, catalog, operationID string, ok bool) {
	project, catalog, _, tail, ok := parseGCPRecommendationCatalogPath(path)
	if !ok || !isGCPRecommendationOperationTail(tail) {
		return "", "", "", false
	}
	return project, catalog, strings.TrimSpace(tail[1]), true
}

func parseGCPRecommendationCatalogParentName(name string) (project, catalog, parent string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[3] != "global" || parts[4] != "catalogs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	catalog = strings.TrimSpace(parts[5])
	if project == "" || catalog == "" {
		return "", "", "", false
	}
	return project, catalog, strings.Join(parts, "/"), true
}

func parseGCPRecommendationCatalogItemName(name string) (project, catalog, parent, itemID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[3] != "global" || parts[4] != "catalogs" || parts[6] != "catalogItems" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	catalog = strings.TrimSpace(parts[5])
	itemID = strings.TrimSpace(parts[7])
	if project == "" || catalog == "" || itemID == "" {
		return "", "", "", "", false
	}
	return project, catalog, strings.Join(parts[:6], "/"), itemID, true
}

func parseGCPRecommendationEventStoreParentName(name string) (project, catalog, eventStore, parent string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[3] != "global" || parts[4] != "catalogs" || parts[6] != "eventStores" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	catalog = strings.TrimSpace(parts[5])
	eventStore = strings.TrimSpace(parts[7])
	if project == "" || catalog == "" || eventStore == "" {
		return "", "", "", "", false
	}
	return project, catalog, eventStore, strings.Join(parts, "/"), true
}

func parseGCPRecommendationPlacementName(name string) (project, catalog, eventStore, placement string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[3] != "global" || parts[4] != "catalogs" || parts[6] != "eventStores" || parts[8] != "placements" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	catalog = strings.TrimSpace(parts[5])
	eventStore = strings.TrimSpace(parts[7])
	placement = strings.TrimSpace(parts[9])
	if project == "" || catalog == "" || eventStore == "" || placement == "" {
		return "", "", "", "", false
	}
	return project, catalog, eventStore, placement, true
}

func parseGCPRecommendationPredictionAPIKeyRegistrationName(name string) (project, catalog, eventStore, registrationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[3] != "global" || parts[4] != "catalogs" || parts[6] != "eventStores" || parts[8] != "predictionApiKeyRegistrations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	catalog = strings.TrimSpace(parts[5])
	eventStore = strings.TrimSpace(parts[7])
	registrationID = strings.TrimSpace(parts[9])
	if project == "" || catalog == "" || eventStore == "" || registrationID == "" {
		return "", "", "", "", false
	}
	return project, catalog, eventStore, registrationID, true
}

func parseGCPRecommendationPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRecommendationEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPRecommendationEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPRecommendationBodyPagination(w http.ResponseWriter, path string, body map[string]any) (pageSize, start int, ok bool) {
	pageSize = 0
	if raw, exists := body["pageSize"]; exists {
		parsed, valid := gcpRecommendationEngineIntFromRaw(raw)
		if !valid || parsed < 0 {
			respondGCPRecommendationEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = parsed
	}
	start = 0
	if raw, exists := body["pageToken"]; exists {
		token := strings.TrimSpace(fmt.Sprint(raw))
		if token != "" {
			parsed, err := parseOptionalNonNegativeInt(token)
			if err != nil {
				respondGCPRecommendationEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = parsed
		}
	}
	return pageSize, start, true
}

func decodeGCPRecommendationEngineJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPRecommendationEngineInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRecommendationEngineBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	raw, ok := body[key]
	if !ok {
		return map[string]any{}
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return m
}

func gcpRecommendationEngineString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	raw, ok := body[key]
	if !ok {
		return ""
	}
	v, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func gcpRecommendationEngineBool(body map[string]any, key string) bool {
	if body == nil {
		return false
	}
	raw, ok := body[key]
	if !ok {
		return false
	}
	value, ok := raw.(bool)
	if !ok {
		return false
	}
	return value
}

func gcpRecommendationEngineIntFromRaw(raw any) (int, bool) {
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" {
		return 0, false
	}
	value, err := parseOptionalNonNegativeInt(s)
	if err != nil {
		return 0, false
	}
	return value, true
}

func gcpRecommendationEnginePaginateList(field string, items []map[string]any, pageSize, start int, path string) (map[string]any, bool) {
	if start > len(items) {
		return nil, false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(items) {
		nextToken = fmt.Sprintf("%d", end)
	}
	out := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, item)
	}
	return map[string]any{field: out, "nextPageToken": nextToken}, true
}

func gcpRecommendationEngineCatalogItemFixture(project, catalog, parent, itemID, title string) map[string]any {
	if title == "" {
		title = "Stackyard Item " + itemID
	}
	return map[string]any{
		"id":    itemID,
		"title": title,
		"description": fmt.Sprintf(
			"Stackyard recommendation item %s in %s", itemID, parent,
		),
		"categoryHierarchies": []any{
			map[string]any{"categories": []any{"Books", "Fiction"}},
		},
		"itemAttributes": map[string]any{
			"brand": map[string]any{"text": []any{"Stackyard"}},
		},
		"languageCode": "en",
		"tags":         []any{"staged", "recommendationengine"},
		"itemGroupId":  "group-1",
		"productMetadata": map[string]any{
			"stockState":          "IN_STOCK",
			"availableQuantity":   100,
			"currencyCode":        "USD",
			"canonicalProductUri": fmt.Sprintf("https://example.com/%s/%s", catalog, itemID),
		},
		"provider": providerGCP,
		"project":  project,
	}
}

func gcpRecommendationEngineUserEventFixture(project, catalog, eventStore, parent string, input map[string]any) map[string]any {
	eventType := gcpRecommendationEngineString(input, "eventType")
	if eventType == "" {
		eventType = "detail-page-view"
	}
	userInfo := gcpRecommendationEngineBodyMap(input, "userInfo")
	visitorID := gcpRecommendationEngineString(userInfo, "visitorId")
	if visitorID == "" {
		visitorID = "visitor-1"
	}
	userID := gcpRecommendationEngineString(userInfo, "userId")
	if userID == "" {
		userID = "user-1"
	}
	return map[string]any{
		"eventType": eventType,
		"userInfo": map[string]any{
			"visitorId": visitorID,
			"userId":    userID,
		},
		"productEventDetail": map[string]any{
			"productDetails": []any{
				map[string]any{"id": "item-1", "quantity": 1, "stockState": "IN_STOCK"},
			},
		},
		"eventTime": gcpRecommendationEngineReferenceTime.Format(time.RFC3339Nano),
		"metadata": map[string]any{
			"parent":     parent,
			"catalog":    catalog,
			"eventStore": eventStore,
			"project":    project,
		},
	}
}

func gcpRecommendationEnginePredictionResultFixture(project, catalog, eventStore, placement, itemID string, score float64) map[string]any {
	item := gcpRecommendationEngineCatalogItemFixture(project, catalog, fmt.Sprintf("projects/%s/locations/global/catalogs/%s", project, catalog), itemID, "Stackyard Item "+itemID)
	return map[string]any{
		"id": itemID,
		"itemMetadata": map[string]any{
			"score":       score,
			"catalogItem": item,
			"placement":   placement,
			"eventStore":  eventStore,
		},
	}
}

func gcpRecommendationEngineOperationFixture(project, catalog, operationID string, done bool) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/global/catalogs/%s/operations/%s", project, catalog, operationID),
		"done": done,
		"metadata": map[string]any{
			"provider":      providerGCP,
			"operationType": operationID,
			"createTime":    gcpRecommendationEngineReferenceTime.Format(time.RFC3339Nano),
		},
	}
}

func gcpRecommendationEngineValidInputConfigSource(inputConfig map[string]any) bool {
	if len(inputConfig) == 0 {
		return false
	}
	for _, key := range []string{"catalogInlineSource", "gcsSource", "userEventInlineSource"} {
		if _, ok := inputConfig[key]; ok {
			return true
		}
	}
	return false
}

func gcpRecommendationEngineValidUserEvent(event map[string]any) bool {
	if len(event) == 0 {
		return false
	}
	if gcpRecommendationEngineString(event, "eventType") == "" {
		return false
	}
	userInfo := gcpRecommendationEngineBodyMap(event, "userInfo")
	return gcpRecommendationEngineString(userInfo, "visitorId") != ""
}

func isGCPRecommendationCatalogItemsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "catalogItems"
}

func isGCPRecommendationCatalogItemTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "catalogItems" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRecommendationCatalogItemsImportTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "catalogItems:import"
}

func isGCPRecommendationWriteUserEventTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "userEvents:write"
}

func isGCPRecommendationCollectUserEventTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "userEvents:collect"
}

func isGCPRecommendationUserEventsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "userEvents"
}

func isGCPRecommendationPurgeUserEventsTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "userEvents:purge"
}

func isGCPRecommendationImportUserEventsTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "userEvents:import"
}

func isGCPRecommendationPredictionAPIKeyRegistrationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "predictionApiKeyRegistrations"
}

func isGCPRecommendationPredictionAPIKeyRegistrationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "predictionApiKeyRegistrations" && strings.TrimSpace(tail[1]) != ""
}

func isGCPRecommendationPredictTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "placements" {
		return false
	}
	parts := strings.Split(strings.TrimSpace(tail[1]), ":")
	return len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && parts[1] == "predict"
}

func isGCPRecommendationOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != ""
}

func respondGCPRecommendationEngineInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPRecommendationEngineError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPRecommendationEngineFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPRecommendationEngineError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPRecommendationEngineError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_recommendationengine(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "recommendationengine") {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/recommendationengine/sample",
			"service":  "recommendationengine",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
