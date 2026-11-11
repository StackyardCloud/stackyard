package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureDataCatalogDataAssetAPIVersion = "2016-03-30"

var azureDataCatalogDataAssetEffectiveRights = []string{
	"Read",
	"Delete",
	"ChangeOwnership",
	"ChangeVisibility",
	"ViewPermissions",
	"ViewRoles",
	"Update",
	"TakeOwnership",
}

var azureDataCatalogDataAssetAnnotationEffectiveRights = []string{
	"Read",
	"Delete",
	"ViewRoles",
	"Update",
}

type azureDataCatalogDataAsset struct {
	CatalogName string
	ViewName    string
	ViewItemID  string
	IdentityKey string
	ID          string
	ETag        string
	Timestamp   time.Time
	Roles       []map[string]any
	Properties  map[string]any
	Annotations map[string]*azureDataCatalogDataAssetAnnotationCollection
}

type azureDataCatalogDataAssetAnnotationCollection struct {
	Name      string
	Singleton bool
	Items     map[string]*azureDataCatalogDataAssetAnnotation
}

type azureDataCatalogDataAssetAnnotation struct {
	ItemID     string
	ID         string
	ETag       string
	Timestamp  time.Time
	Roles      []map[string]any
	Properties map[string]any
}

func (s *Server) handleAzureDataCatalogDataAssetRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/catalogs/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) < 3 || !strings.EqualFold(segments[0], "catalogs") {
		return false
	}

	catalogName := strings.TrimSpace(segments[1])
	if catalogName == "" {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "catalog_name is required")
		return true
	}

	if !azureDataCatalogDataAssetHasExpectedAPIVersion(r) {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "api-version must be 2016-03-30")
		return true
	}

	if len(segments) == 4 && strings.EqualFold(segments[2], "search") && strings.EqualFold(segments[3], "search") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureDataCatalogDataAssetSearch(w, r, path, catalogName)
		return true
	}

	if !strings.EqualFold(segments[2], "views") || len(segments) < 4 {
		return false
	}

	viewName := strings.TrimSpace(segments[3])
	if viewName == "" {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "view_name is required")
		return true
	}

	switch len(segments) {
	case 4:
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureDataCatalogDataAssetRegisterOrUpdate(w, r, path, catalogName, viewName)
		return true
	case 5:
		viewItemID := strings.TrimSpace(segments[4])
		if viewItemID == "" {
			azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "view_item_id is required")
			return true
		}
		switch r.Method {
		case http.MethodGet:
			s.handleAzureDataCatalogDataAssetGetWithAnnotations(w, r, path, catalogName, viewName, viewItemID)
		case http.MethodDelete:
			s.handleAzureDataCatalogDataAssetDelete(w, r, path, catalogName, viewName, viewItemID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return true
	case 6:
		viewItemID := strings.TrimSpace(segments[4])
		nestedViewName := strings.TrimSpace(segments[5])
		if viewItemID == "" || nestedViewName == "" {
			azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "view_item_id and nested_view_name are required")
			return true
		}
		switch r.Method {
		case http.MethodPost:
			s.handleAzureDataCatalogDataAssetAnnotate(w, r, path, catalogName, viewName, viewItemID, nestedViewName)
		case http.MethodPut:
			s.handleAzureDataCatalogDataAssetUpdateAnnotation(w, r, path, catalogName, viewName, viewItemID, nestedViewName, "")
		case http.MethodDelete:
			s.handleAzureDataCatalogDataAssetDeleteAnnotation(w, r, path, catalogName, viewName, viewItemID, nestedViewName, "")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return true
	case 7:
		viewItemID := strings.TrimSpace(segments[4])
		nestedViewName := strings.TrimSpace(segments[5])
		nestedItemID := strings.TrimSpace(segments[6])
		if viewItemID == "" || nestedViewName == "" || nestedItemID == "" {
			azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "view_item_id, nested_view_name, and nested_non_singleton_view_item_id are required")
			return true
		}
		switch r.Method {
		case http.MethodPut:
			s.handleAzureDataCatalogDataAssetUpdateAnnotation(w, r, path, catalogName, viewName, viewItemID, nestedViewName, nestedItemID)
		case http.MethodDelete:
			s.handleAzureDataCatalogDataAssetDeleteAnnotation(w, r, path, catalogName, viewName, viewItemID, nestedViewName, nestedItemID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureDataCatalogDataAssetRegisterOrUpdate(w http.ResponseWriter, r *http.Request, path, catalogName, viewName string) {
	payload, err := azureDataCatalogDataAssetReadJSONBody(r)
	if err != nil {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", err.Error())
		return
	}

	properties, ok := payload["properties"].(map[string]any)
	if !ok {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "properties object is required")
		return
	}

	identityKey := azureDataCatalogDataAssetIdentityKey(viewName, properties)
	registerETag := strings.TrimSpace(azureDataCatalogDataAssetString(payload["etag"]))

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	var existing *azureDataCatalogDataAsset
	for _, candidate := range s.azureDataCatalogDataAssets {
		if strings.EqualFold(candidate.CatalogName, catalogName) &&
			strings.EqualFold(candidate.ViewName, viewName) &&
			candidate.IdentityKey == identityKey {
			existing = candidate
			break
		}
	}

	if existing != nil {
		if registerETag != "" && !azureDataCatalogDataAssetETagMatches(registerETag, existing.ETag) {
			azureDataCatalogDataAssetError(w, http.StatusPreconditionFailed, path, "PreconditionFailed", "The request was canceled because of the ETag mismatch in at least one item.")
			return
		}
		existing.Timestamp = now
		existing.ETag = azureDataCatalogDataAssetGenerateETag(existing.ID, now, payload)
		existing.Roles = azureDataCatalogDataAssetNormalizeRoles(payload["roles"])
		existing.Properties = azureDataCatalogDataAssetCloneObject(properties)
		existing.Annotations = azureDataCatalogDataAssetParseAssetAnnotations(catalogName, viewName, existing.ViewItemID, payload["annotations"], now)

		w.Header().Set("Location", existing.ID)
		respondJSON(w, http.StatusOK, map[string]any{
			"id": existing.ID,
		})
		return
	}

	viewItemID := azureDataCatalogDataAssetGenerateViewItemID(identityKey)
	asset := &azureDataCatalogDataAsset{
		CatalogName: catalogName,
		ViewName:    viewName,
		ViewItemID:  viewItemID,
		IdentityKey: identityKey,
		ID:          azureDataCatalogDataAssetID(catalogName, viewName, viewItemID),
		Timestamp:   now,
		Roles:       azureDataCatalogDataAssetNormalizeRoles(payload["roles"]),
		Properties:  azureDataCatalogDataAssetCloneObject(properties),
		Annotations: azureDataCatalogDataAssetParseAssetAnnotations(catalogName, viewName, viewItemID, payload["annotations"], now),
	}
	asset.ETag = azureDataCatalogDataAssetGenerateETag(asset.ID, now, payload)

	s.azureDataCatalogDataAssets[azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID)] = asset

	w.Header().Set("Location", asset.ID)
	respondJSON(w, http.StatusCreated, map[string]any{
		"id": asset.ID,
	})
}

func (s *Server) handleAzureDataCatalogDataAssetAnnotate(w http.ResponseWriter, r *http.Request, path, catalogName, viewName, viewItemID, nestedViewName string) {
	payload, err := azureDataCatalogDataAssetReadJSONBody(r)
	if err != nil {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", err.Error())
		return
	}

	now := time.Now().UTC()
	collectionKey := strings.ToLower(strings.TrimSpace(nestedViewName))
	itemID := azureDataCatalogDataAssetAnnotationItemID(payload, nestedViewName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	asset := s.azureDataCatalogDataAssets[azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID)]
	if asset == nil {
		azureDataCatalogDataAssetError(w, http.StatusNotFound, path, "NotFound", "data asset was not found")
		return
	}

	if asset.Annotations == nil {
		asset.Annotations = map[string]*azureDataCatalogDataAssetAnnotationCollection{}
	}
	collection := asset.Annotations[collectionKey]
	if collection == nil {
		collection = &azureDataCatalogDataAssetAnnotationCollection{
			Name:      nestedViewName,
			Singleton: itemID == "",
			Items:     map[string]*azureDataCatalogDataAssetAnnotation{},
		}
		asset.Annotations[collectionKey] = collection
	}

	if collection.Singleton && itemID != "" {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "nested view is singleton; nested_non_singleton_view_item_id is not expected")
		return
	}
	if !collection.Singleton && itemID == "" {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "nested_non_singleton_view_item_id is required for non-singleton nested views")
		return
	}

	annotationKey := itemID
	if collection.Singleton {
		annotationKey = ""
	}
	existing := collection.Items[annotationKey]

	inputETag := strings.TrimSpace(azureDataCatalogDataAssetString(payload["etag"]))
	if existing != nil && inputETag != "" && !azureDataCatalogDataAssetETagMatches(inputETag, existing.ETag) {
		azureDataCatalogDataAssetError(w, http.StatusPreconditionFailed, path, "PreconditionFailed", "The request was canceled because of the ETag mismatch in at least one item.")
		return
	}

	if collection.Name == "" {
		collection.Name = nestedViewName
	}
	if collection.Singleton && annotationKey == "" {
		itemID = nestedViewName
	}
	annotationID := azureDataCatalogDataAssetAnnotationID(asset.ID, collection.Name, annotationKey)
	annotation := &azureDataCatalogDataAssetAnnotation{
		ItemID:     annotationKey,
		ID:         annotationID,
		Timestamp:  now,
		ETag:       azureDataCatalogDataAssetGenerateETag(annotationID, now, payload),
		Roles:      azureDataCatalogDataAssetNormalizeRoles(payload["roles"]),
		Properties: azureDataCatalogDataAssetNormalizeAnnotationProperties(payload),
	}
	collection.Items[annotationKey] = annotation

	asset.Timestamp = now
	asset.ETag = azureDataCatalogDataAssetGenerateETag(asset.ID, now, map[string]any{"annotation": annotationID})

	w.Header().Set("Location", annotation.ID)
	if existing == nil {
		respondJSON(w, http.StatusCreated, map[string]any{"id": annotation.ID})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"id": annotation.ID})
}

func (s *Server) handleAzureDataCatalogDataAssetGetWithAnnotations(w http.ResponseWriter, r *http.Request, path, catalogName, viewName, viewItemID string) {
	metadataMode, ok := azureDataCatalogDataAssetMetadataMode(r.Header.Get("Accept"))
	if !ok {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "adc.metadata must be one of none, minimal, or full")
		return
	}

	s.providerStorageMu.Lock()
	asset := s.azureDataCatalogDataAssets[azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID)]
	s.providerStorageMu.Unlock()
	if asset == nil {
		azureDataCatalogDataAssetError(w, http.StatusNotFound, path, "NotFound", "data asset was not found")
		return
	}

	respondJSON(w, http.StatusOK, azureDataCatalogDataAssetFixture(asset, metadataMode))
}

func (s *Server) handleAzureDataCatalogDataAssetSearch(w http.ResponseWriter, r *http.Request, path, catalogName string) {
	query := r.URL.Query()
	if _, exists := query["searchTerms"]; !exists {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "searchTerms query parameter is required")
		return
	}
	searchTerms := strings.TrimSpace(query.Get("searchTerms"))

	startPage := 1
	if raw := strings.TrimSpace(query.Get("startPage")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "startPage must be an integer greater than 0")
			return
		}
		startPage = parsed
	}

	count := 10
	if raw := strings.TrimSpace(query.Get("count")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "count must be an integer between 1 and 100")
			return
		}
		count = parsed
	}

	if view := strings.TrimSpace(query.Get("view")); view != "" && !strings.EqualFold(view, "DataSource") {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "view must be DataSource when provided")
		return
	}

	terms := strings.Fields(strings.ToLower(searchTerms))
	facets := strings.TrimSpace(query.Get("facets"))

	s.providerStorageMu.Lock()
	candidates := make([]*azureDataCatalogDataAsset, 0)
	for _, asset := range s.azureDataCatalogDataAssets {
		if strings.EqualFold(asset.CatalogName, catalogName) {
			candidates = append(candidates, asset)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(candidates, func(i, j int) bool {
		left := strings.ToLower(candidates[i].ViewName + "/" + candidates[i].ViewItemID)
		right := strings.ToLower(candidates[j].ViewName + "/" + candidates[j].ViewItemID)
		return left < right
	})

	filtered := make([]*azureDataCatalogDataAsset, 0, len(candidates))
	for _, candidate := range candidates {
		if azureDataCatalogDataAssetMatchesSearch(candidate, terms) {
			filtered = append(filtered, candidate)
		}
	}

	startIndex := (startPage - 1) * count
	page := make([]*azureDataCatalogDataAsset, 0)
	if startIndex < len(filtered) {
		endIndex := startIndex + count
		if endIndex > len(filtered) {
			endIndex = len(filtered)
		}
		page = append(page, filtered[startIndex:endIndex]...)
	}

	results := make([]any, 0, len(page))
	for _, asset := range page {
		results = append(results, map[string]any{
			"updated": asset.Timestamp.Format(time.RFC3339Nano),
			"content": azureDataCatalogDataAssetFixture(asset, "full"),
		})
	}

	facetList := make([]any, 0)
	if facets != "" {
		for _, facet := range strings.Split(facets, ",") {
			trimmed := strings.TrimSpace(facet)
			if trimmed == "" {
				continue
			}
			facetList = append(facetList, map[string]any{
				"name":  trimmed,
				"value": map[string]any{},
			})
		}
	}

	responseStartIndex := 0
	if len(page) > 0 {
		responseStartIndex = startIndex + 1
	}

	queryID := azureDataCatalogDataAssetStableID("search|" + strings.ToLower(catalogName) + "|" + searchTerms + "|" + strconv.Itoa(startPage) + "|" + strconv.Itoa(count) + "|" + facets)
	respondJSON(w, http.StatusOK, map[string]any{
		"query": map[string]any{
			"id":           queryID,
			"searchTerms":  searchTerms,
			"startPage":    startPage,
			"count":        count,
			"startIndex":   responseStartIndex,
			"itemsPerPage": len(page),
			"totalResults": len(filtered),
			"results":      results,
			"facets":       facetList,
		},
	})
}

func (s *Server) handleAzureDataCatalogDataAssetDelete(w http.ResponseWriter, r *http.Request, path, catalogName, viewName, viewItemID string) {
	key := azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	asset := s.azureDataCatalogDataAssets[key]
	if asset == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))
	if ifMatch != "" && !azureDataCatalogDataAssetETagMatches(ifMatch, asset.ETag) {
		azureDataCatalogDataAssetError(w, http.StatusPreconditionFailed, path, "PreconditionFailed", "The request was canceled because of the ETag mismatch.")
		return
	}

	delete(s.azureDataCatalogDataAssets, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureDataCatalogDataAssetUpdateAnnotation(w http.ResponseWriter, r *http.Request, path, catalogName, viewName, viewItemID, nestedViewName, nestedItemID string) {
	payload, err := azureDataCatalogDataAssetReadJSONBody(r)
	if err != nil {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", err.Error())
		return
	}

	now := time.Now().UTC()
	collectionKey := strings.ToLower(strings.TrimSpace(nestedViewName))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	asset := s.azureDataCatalogDataAssets[azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID)]
	if asset == nil {
		azureDataCatalogDataAssetError(w, http.StatusNotFound, path, "NotFound", "data asset was not found")
		return
	}

	collection := asset.Annotations[collectionKey]
	if collection == nil {
		azureDataCatalogDataAssetError(w, http.StatusNotFound, path, "NotFound", "annotation was not found")
		return
	}

	itemKey := nestedItemID
	if collection.Singleton {
		if nestedItemID != "" {
			azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "nested_non_singleton_view_item_id is not valid for singleton nested views")
			return
		}
		itemKey = ""
	} else if strings.TrimSpace(itemKey) == "" {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "nested_non_singleton_view_item_id is required for non-singleton nested views")
		return
	}

	annotation := collection.Items[itemKey]
	if annotation == nil {
		azureDataCatalogDataAssetError(w, http.StatusNotFound, path, "NotFound", "annotation was not found")
		return
	}

	inputETag := strings.TrimSpace(azureDataCatalogDataAssetString(payload["etag"]))
	if inputETag != "" && !azureDataCatalogDataAssetETagMatches(inputETag, annotation.ETag) {
		azureDataCatalogDataAssetError(w, http.StatusPreconditionFailed, path, "PreconditionFailed", "The request was canceled because of the ETag mismatch in at least one item.")
		return
	}

	if roles, ok := payload["roles"]; ok {
		annotation.Roles = azureDataCatalogDataAssetNormalizeRoles(roles)
	}
	annotation.Properties = azureDataCatalogDataAssetNormalizeAnnotationProperties(payload)
	annotation.Timestamp = now
	annotation.ETag = azureDataCatalogDataAssetGenerateETag(annotation.ID, now, payload)

	asset.Timestamp = now
	asset.ETag = azureDataCatalogDataAssetGenerateETag(asset.ID, now, map[string]any{"annotation": annotation.ID})
	respondJSON(w, http.StatusOK, map[string]any{"id": annotation.ID})
}

func (s *Server) handleAzureDataCatalogDataAssetDeleteAnnotation(w http.ResponseWriter, r *http.Request, path, catalogName, viewName, viewItemID, nestedViewName, nestedItemID string) {
	collectionKey := strings.ToLower(strings.TrimSpace(nestedViewName))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	asset := s.azureDataCatalogDataAssets[azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID)]
	if asset == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	collection := asset.Annotations[collectionKey]
	if collection == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	ifMatch := strings.TrimSpace(r.Header.Get("If-Match"))

	if nestedItemID == "" {
		if collection.Singleton {
			annotation := collection.Items[""]
			if annotation != nil && ifMatch != "" && !azureDataCatalogDataAssetETagMatches(ifMatch, annotation.ETag) {
				azureDataCatalogDataAssetError(w, http.StatusPreconditionFailed, path, "PreconditionFailed", "The request was canceled because of the ETag mismatch.")
				return
			}
		}
		delete(asset.Annotations, collectionKey)
		asset.Timestamp = time.Now().UTC()
		asset.ETag = azureDataCatalogDataAssetGenerateETag(asset.ID, asset.Timestamp, map[string]any{"delete": collectionKey})
		w.WriteHeader(http.StatusNoContent)
		return
	}

	itemKey := nestedItemID
	if collection.Singleton {
		azureDataCatalogDataAssetError(w, http.StatusBadRequest, path, "BadRequest", "nested_non_singleton_view_item_id is not valid for singleton nested views")
		return
	}

	annotation := collection.Items[itemKey]
	if annotation == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if ifMatch != "" && !azureDataCatalogDataAssetETagMatches(ifMatch, annotation.ETag) {
		azureDataCatalogDataAssetError(w, http.StatusPreconditionFailed, path, "PreconditionFailed", "The request was canceled because of the ETag mismatch.")
		return
	}

	delete(collection.Items, itemKey)
	if len(collection.Items) == 0 {
		delete(asset.Annotations, collectionKey)
	}
	asset.Timestamp = time.Now().UTC()
	asset.ETag = azureDataCatalogDataAssetGenerateETag(asset.ID, asset.Timestamp, map[string]any{"delete": annotation.ID})
	w.WriteHeader(http.StatusNoContent)
}

func azureDataCatalogDataAssetParseAssetAnnotations(catalogName, viewName, viewItemID string, raw any, now time.Time) map[string]*azureDataCatalogDataAssetAnnotationCollection {
	annotationsMap, ok := raw.(map[string]any)
	if !ok || len(annotationsMap) == 0 {
		return map[string]*azureDataCatalogDataAssetAnnotationCollection{}
	}

	output := map[string]*azureDataCatalogDataAssetAnnotationCollection{}
	assetID := azureDataCatalogDataAssetID(catalogName, viewName, viewItemID)

	for rawName, rawValue := range annotationsMap {
		nestedName := strings.TrimSpace(rawName)
		if nestedName == "" {
			continue
		}
		collectionKey := strings.ToLower(nestedName)
		collection := &azureDataCatalogDataAssetAnnotationCollection{
			Name:  nestedName,
			Items: map[string]*azureDataCatalogDataAssetAnnotation{},
		}

		switch typed := rawValue.(type) {
		case map[string]any:
			collection.Singleton = true
			annotationID := azureDataCatalogDataAssetAnnotationID(assetID, nestedName, "")
			collection.Items[""] = &azureDataCatalogDataAssetAnnotation{
				ItemID:     "",
				ID:         annotationID,
				ETag:       azureDataCatalogDataAssetGenerateETag(annotationID, now, typed),
				Timestamp:  now,
				Roles:      azureDataCatalogDataAssetNormalizeRoles(typed["roles"]),
				Properties: azureDataCatalogDataAssetNormalizeAnnotationProperties(typed),
			}
		case []any:
			collection.Singleton = false
			for _, rawItem := range typed {
				itemMap, ok := rawItem.(map[string]any)
				if !ok {
					continue
				}
				itemID := azureDataCatalogDataAssetAnnotationItemID(itemMap, nestedName)
				if itemID == "" {
					itemID = azureDataCatalogDataAssetStableID(nestedName + "|" + azureDataCatalogDataAssetCanonicalJSON(itemMap))
				}
				annotationID := azureDataCatalogDataAssetAnnotationID(assetID, nestedName, itemID)
				collection.Items[itemID] = &azureDataCatalogDataAssetAnnotation{
					ItemID:     itemID,
					ID:         annotationID,
					ETag:       azureDataCatalogDataAssetGenerateETag(annotationID, now, itemMap),
					Timestamp:  now,
					Roles:      azureDataCatalogDataAssetNormalizeRoles(itemMap["roles"]),
					Properties: azureDataCatalogDataAssetNormalizeAnnotationProperties(itemMap),
				}
			}
		default:
			continue
		}
		output[collectionKey] = collection
	}
	return output
}

func azureDataCatalogDataAssetNormalizeAnnotationProperties(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	if rawProperties, ok := input["properties"].(map[string]any); ok {
		return azureDataCatalogDataAssetCloneObject(rawProperties)
	}

	properties := map[string]any{}
	for key, value := range input {
		lowered := strings.ToLower(strings.TrimSpace(key))
		if lowered == "etag" || lowered == "id" || lowered == "timestamp" || lowered == "roles" || lowered == "effectiverights" {
			continue
		}
		properties[key] = azureDataCatalogDataAssetDeepCopy(value)
	}
	return properties
}

func azureDataCatalogDataAssetNormalizeRoles(raw any) []map[string]any {
	roles, ok := raw.([]any)
	if !ok || len(roles) == 0 {
		return []map[string]any{
			{
				"role": "Contributor",
				"members": []any{
					map[string]any{
						"objectId": "00000000-0000-0000-0000-000000000201",
					},
				},
			},
		}
	}

	output := make([]map[string]any, 0, len(roles))
	for _, rawRole := range roles {
		roleMap, ok := rawRole.(map[string]any)
		if !ok {
			continue
		}
		roleName := strings.TrimSpace(azureDataCatalogDataAssetString(roleMap["role"]))
		if roleName == "" {
			roleName = "Contributor"
		}
		members := make([]any, 0)
		if rawMembers, ok := roleMap["members"].([]any); ok {
			for _, rawMember := range rawMembers {
				memberMap, ok := rawMember.(map[string]any)
				if !ok {
					continue
				}
				members = append(members, azureDataCatalogDataAssetCloneObject(memberMap))
			}
		}
		if len(members) == 0 {
			members = append(members, map[string]any{"objectId": "00000000-0000-0000-0000-000000000201"})
		}
		output = append(output, map[string]any{
			"role":    roleName,
			"members": members,
		})
	}
	if len(output) == 0 {
		return []map[string]any{
			{
				"role": "Contributor",
				"members": []any{
					map[string]any{
						"objectId": "00000000-0000-0000-0000-000000000201",
					},
				},
			},
		}
	}
	return output
}

func azureDataCatalogDataAssetFixture(asset *azureDataCatalogDataAsset, metadataMode string) map[string]any {
	if asset == nil {
		return map[string]any{}
	}

	includeETag := strings.ToLower(strings.TrimSpace(metadataMode)) != "none"
	fixture := map[string]any{
		"id":              asset.ID,
		"timestamp":       asset.Timestamp.Format(time.RFC3339Nano),
		"roles":           azureDataCatalogDataAssetCloneRoles(asset.Roles),
		"effectiveRights": append([]string(nil), azureDataCatalogDataAssetEffectiveRights...),
		"properties":      azureDataCatalogDataAssetCloneObject(asset.Properties),
	}
	if includeETag {
		fixture["etag"] = asset.ETag
	}

	annotations := map[string]any{}
	names := make([]string, 0, len(asset.Annotations))
	for key := range asset.Annotations {
		names = append(names, key)
	}
	sort.Strings(names)

	for _, nameKey := range names {
		collection := asset.Annotations[nameKey]
		if collection == nil {
			continue
		}
		if collection.Singleton {
			item := collection.Items[""]
			if item == nil {
				continue
			}
			annotations[collection.Name] = azureDataCatalogDataAssetAnnotationFixture(item, includeETag)
			continue
		}

		itemKeys := make([]string, 0, len(collection.Items))
		for itemKey := range collection.Items {
			itemKeys = append(itemKeys, itemKey)
		}
		sort.Strings(itemKeys)

		values := make([]any, 0, len(itemKeys))
		for _, itemKey := range itemKeys {
			values = append(values, azureDataCatalogDataAssetAnnotationFixture(collection.Items[itemKey], includeETag))
		}
		annotations[collection.Name] = values
	}
	fixture["annotations"] = annotations
	return fixture
}

func azureDataCatalogDataAssetAnnotationFixture(annotation *azureDataCatalogDataAssetAnnotation, includeETag bool) map[string]any {
	if annotation == nil {
		return map[string]any{}
	}
	fixture := map[string]any{
		"id":              annotation.ID,
		"timestamp":       annotation.Timestamp.Format(time.RFC3339Nano),
		"roles":           azureDataCatalogDataAssetCloneRoles(annotation.Roles),
		"effectiveRights": append([]string(nil), azureDataCatalogDataAssetAnnotationEffectiveRights...),
		"properties":      azureDataCatalogDataAssetCloneObject(annotation.Properties),
	}
	if includeETag {
		fixture["etag"] = annotation.ETag
	}
	return fixture
}

func azureDataCatalogDataAssetReadJSONBody(r *http.Request) (map[string]any, error) {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("content type must be application/json")
	}

	body, err := readBodyBytes(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read request body")
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, fmt.Errorf("request body is required")
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("request body must be valid JSON")
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func azureDataCatalogDataAssetMatchesSearch(asset *azureDataCatalogDataAsset, terms []string) bool {
	if asset == nil {
		return false
	}
	if len(terms) == 0 {
		return true
	}
	text := strings.ToLower(asset.ViewName + " " + asset.ViewItemID + " " + azureDataCatalogDataAssetCanonicalJSON(asset.Properties))
	for _, collection := range asset.Annotations {
		if collection == nil {
			continue
		}
		for _, annotation := range collection.Items {
			if annotation == nil {
				continue
			}
			text += " " + strings.ToLower(azureDataCatalogDataAssetCanonicalJSON(annotation.Properties))
		}
	}
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func azureDataCatalogDataAssetMetadataMode(accept string) (string, bool) {
	metadataMode := "full"
	header := strings.ToLower(strings.TrimSpace(accept))
	if header == "" {
		return metadataMode, true
	}

	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "adc.metadata=") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(part, "adc.metadata="))
		switch value {
		case "none", "minimal", "full":
			return value, true
		default:
			return "", false
		}
	}
	return metadataMode, true
}

func azureDataCatalogDataAssetHasExpectedAPIVersion(r *http.Request) bool {
	values, ok := r.URL.Query()["api-version"]
	if !ok || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) != azureDataCatalogDataAssetAPIVersion {
			return false
		}
	}
	return true
}

func azureDataCatalogDataAssetStorageKey(catalogName, viewName, viewItemID string) string {
	return strings.ToLower(strings.TrimSpace(catalogName)) + "|" + strings.ToLower(strings.TrimSpace(viewName)) + "|" + strings.ToLower(strings.TrimSpace(viewItemID))
}

func azureDataCatalogDataAssetIdentityKey(viewName string, properties map[string]any) string {
	identity := map[string]any{
		"viewName": strings.ToLower(strings.TrimSpace(viewName)),
	}
	if properties != nil {
		if dataSource, ok := properties["dataSource"]; ok {
			identity["dataSource"] = dataSource
		}
		if dsl, ok := properties["dsl"]; ok {
			identity["dsl"] = dsl
		}
		if name := strings.TrimSpace(azureDataCatalogDataAssetString(properties["name"])); name != "" {
			identity["name"] = strings.ToLower(name)
		}
	}
	encoded := azureDataCatalogDataAssetCanonicalJSON(identity)
	if encoded == "{}" || encoded == "" {
		encoded = azureDataCatalogDataAssetCanonicalJSON(properties)
	}
	return azureDataCatalogDataAssetStableID(encoded)
}

func azureDataCatalogDataAssetGenerateViewItemID(identityKey string) string {
	stable := azureDataCatalogDataAssetStableID(identityKey)
	if len(stable) > 32 {
		return stable[:32]
	}
	return stable
}

func azureDataCatalogDataAssetGenerateETag(id string, timestamp time.Time, payload map[string]any) string {
	seed := id + "|" + timestamp.Format(time.RFC3339Nano) + "|" + azureDataCatalogDataAssetCanonicalJSON(payload)
	hash := sha1.Sum([]byte(seed))
	return hex.EncodeToString(hash[:16])
}

func azureDataCatalogDataAssetID(catalogName, viewName, viewItemID string) string {
	catalogSegment := strings.TrimSpace(catalogName)
	if catalogSegment == "" {
		catalogSegment = "DefaultCatalog"
	}
	return "https://stackyard.local/catalogs/" + catalogSegment + "/views/" + strings.TrimSpace(viewName) + "/" + strings.TrimSpace(viewItemID)
}

func azureDataCatalogDataAssetAnnotationID(assetID, nestedViewName, nestedItemID string) string {
	base := strings.TrimRight(assetID, "/") + "/" + strings.TrimSpace(nestedViewName)
	if strings.TrimSpace(nestedItemID) == "" {
		return base
	}
	return base + "/" + strings.TrimSpace(nestedItemID)
}

func azureDataCatalogDataAssetAnnotationItemID(payload map[string]any, nestedViewName string) string {
	if payload == nil {
		return ""
	}
	if properties, ok := payload["properties"].(map[string]any); ok {
		if key := strings.TrimSpace(azureDataCatalogDataAssetString(properties["key"])); key != "" {
			return azureDataCatalogDataAssetStableID(strings.ToLower(strings.TrimSpace(nestedViewName)) + "|" + key)
		}
	}
	return ""
}

func azureDataCatalogDataAssetETagMatches(requestETag, currentETag string) bool {
	requestValue := strings.TrimSpace(requestETag)
	currentValue := strings.TrimSpace(currentETag)
	if requestValue == "" || currentValue == "" {
		return false
	}
	if requestValue == "*" {
		return true
	}
	requestNormalized := azureDataCatalogDataAssetTrimETag(requestValue)
	currentNormalized := azureDataCatalogDataAssetTrimETag(currentValue)
	return requestNormalized == currentNormalized
}

func azureDataCatalogDataAssetTrimETag(value string) string {
	trimmed := strings.TrimSpace(value)
	trimmed = strings.TrimPrefix(trimmed, "W/")
	trimmed = strings.TrimPrefix(trimmed, "w/")
	trimmed = strings.Trim(trimmed, "\"")
	return strings.TrimSpace(trimmed)
}

func azureDataCatalogDataAssetCanonicalJSON(value any) string {
	normalized := azureDataCatalogDataAssetCanonicalize(value)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func azureDataCatalogDataAssetCanonicalize(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		ordered := make(map[string]any, len(typed))
		for _, key := range keys {
			ordered[key] = azureDataCatalogDataAssetCanonicalize(typed[key])
		}
		return ordered
	case []any:
		ordered := make([]any, len(typed))
		for idx, item := range typed {
			ordered[idx] = azureDataCatalogDataAssetCanonicalize(item)
		}
		return ordered
	default:
		return typed
	}
}

func azureDataCatalogDataAssetCloneRoles(roles []map[string]any) []any {
	if len(roles) == 0 {
		return []any{}
	}
	cloned := make([]any, 0, len(roles))
	for _, role := range roles {
		cloned = append(cloned, azureDataCatalogDataAssetCloneObject(role))
	}
	return cloned
}

func azureDataCatalogDataAssetCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = azureDataCatalogDataAssetDeepCopy(value)
	}
	return cloned
}

func azureDataCatalogDataAssetDeepCopy(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return azureDataCatalogDataAssetCloneObject(typed)
	case []any:
		clone := make([]any, len(typed))
		for idx, item := range typed {
			clone[idx] = azureDataCatalogDataAssetDeepCopy(item)
		}
		return clone
	default:
		return typed
	}
}

func azureDataCatalogDataAssetStableID(value string) string {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:])
}

func azureDataCatalogDataAssetString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func azureDataCatalogDataAssetError(w http.ResponseWriter, status int, path, code, message string) {
	if strings.TrimSpace(code) == "" {
		code = "BadRequest"
	}
	if strings.TrimSpace(message) == "" {
		message = "request is invalid"
	}
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"target":  path,
		},
	})
}
