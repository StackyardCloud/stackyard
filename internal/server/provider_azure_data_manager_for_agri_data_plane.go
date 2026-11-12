package server

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureDataManagerForAgriDataPlaneAPIVersion = "2023-11-01-preview"
const azureDataManagerForAgriDataPlanePrefix = "/azure/data-manager-for-agri/dataplane/2023-11-01-preview"

type azureDataManagerForAgriDataPlaneEntity struct {
	Collection string
	Scope      string
	ID         string
	Payload    map[string]any
	ETag       string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type azureDataManagerForAgriDataPlaneFile struct {
	Collection  string
	Scope       string
	ID          string
	FileName    string
	ContentType string
	Content     []byte
	UpdatedAt   time.Time
}

type azureDataManagerForAgriDataPlaneGrant struct {
	DatasetID string
	GrantID   string
	Payload   map[string]any
	CreatedAt time.Time
}

type azureDataManagerForAgriDataPlaneJob struct {
	Kind      string
	ID        string
	Payload   map[string]any
	CreatedAt time.Time
}

var azureDataManagerForAgriDataPlaneSupportedCollections = map[string]string{
	"parties":                     "parties",
	"farms":                       "farms",
	"fields":                      "fields",
	"seasonalfields":              "seasonalFields",
	"seasons":                     "seasons",
	"crops":                       "crops",
	"cropproducts":                "cropProducts",
	"attachments":                 "attachments",
	"zones":                       "zones",
	"managementzones":             "managementZones",
	"chemicalproducts":            "chemicalProducts",
	"farmequipments":              "farmEquipments",
	"scenes":                      "scenes",
	"image-processing":            "image-processing",
	"device-data-models":          "device-data-models",
	"devices":                     "devices",
	"sensor-data-models":          "sensor-data-models",
	"sensors":                     "sensors",
	"sensor-mappings":             "sensor-mappings",
	"sensor-partner-integrations": "sensor-partner-integrations",
	"sensor-events":               "sensor-events",
	"weather-data":                "weather-data",
	"application-data":            "application-data",
	"tillage-data":                "tillage-data",
	"planting-data":               "planting-data",
	"harvest-data":                "harvest-data",
	"farm-operations":             "farm-operations",
	"prescriptions":               "prescriptions",
	"prescription":                "prescriptions",
	"prescription-maps":           "prescription-maps",
	"plant-tissue-analyses":       "plant-tissue-analyses",
	"nutrient-analyses":           "nutrient-analyses",
	"insights":                    "insights",
	"insight-attachments":         "insight-attachments",
	"datasets":                    "datasets",
	"dataset-records":             "dataset-records",
	"oauth-providers":             "oauth-providers",
	"oauth-tokens":                "oauth-tokens",
	"solutions":                   "solutions",
}

var azureDataManagerForAgriDataPlaneGlobalCollections = map[string]struct{}{
	"farms":                 {},
	"fields":                {},
	"seasonalFields":        {},
	"seasons":               {},
	"crops":                 {},
	"cropProducts":          {},
	"zones":                 {},
	"managementZones":       {},
	"chemicalProducts":      {},
	"farmEquipments":        {},
	"application-data":      {},
	"tillage-data":          {},
	"planting-data":         {},
	"harvest-data":          {},
	"farm-operations":       {},
	"prescriptions":         {},
	"prescription-maps":     {},
	"plant-tissue-analyses": {},
	"nutrient-analyses":     {},
	"scenes":                {},
	"datasets":              {},
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureDataManagerForAgriDataPlanePrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureDataManagerForAgriDataPlaneAPIVersion {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "api-version must be 2023-11-01-preview")
		return true
	}

	relative := strings.TrimPrefix(path, azureDataManagerForAgriDataPlanePrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[0])) {
	case "parties":
		s.handleAzureDataManagerForAgriDataPlaneParties(w, r, segments[1:])
		return true
	case "sensor-partners":
		s.handleAzureDataManagerForAgriDataPlaneSensorPartners(w, r, segments[1:])
		return true
	case "datasets":
		s.handleAzureDataManagerForAgriDataPlaneDatasets(w, r, segments[1:])
		return true
	case "oauth":
		s.handleAzureDataManagerForAgriDataPlaneOAuth(w, r, segments[1:])
		return true
	case "weather-data", "weather-data:fetch":
		s.handleAzureDataManagerForAgriDataPlaneWeather(w, r, segments)
		return true
	case "solutions":
		s.handleAzureDataManagerForAgriDataPlaneSolutions(w, r, segments[1:])
		return true
	default:
		s.handleAzureDataManagerForAgriDataPlaneTopLevel(w, r, segments)
		return true
	}
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneParties(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, "parties", "global", false)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	partyID := strings.TrimSpace(tail[0])
	if partyID == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "partyId is required")
		return
	}
	if len(tail) == 1 {
		s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, "parties", "global", partyID, map[string]any{"partyId": partyID})
		return
	}

	if len(tail) >= 7 &&
		strings.EqualFold(tail[1], "models") &&
		strings.EqualFold(tail[3], "resource-types") &&
		strings.EqualFold(tail[5], "resources") {
		s.handleAzureDataManagerForAgriDataPlaneInsightRoutes(w, r, partyID, tail[1:])
		return
	}

	collection, ok := azureDataManagerForAgriDataPlaneCanonicalCollection(tail[1])
	if !ok {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "collection not found")
		return
	}
	scope := "party:" + partyID
	if len(tail) == 2 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, collection, scope, true)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	resourceID := strings.TrimSpace(tail[2])
	if resourceID == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "resource id is required")
		return
	}
	if len(tail) == 4 && strings.EqualFold(tail[3], "file") && azureDataManagerForAgriDataPlaneSupportsFiles(collection) {
		s.azureDataManagerForAgriDataPlaneDownloadFile(w, scope, collection, resourceID)
		return
	}
	s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, collection, scope, resourceID, map[string]any{"partyId": partyID})
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneInsightRoutes(w http.ResponseWriter, r *http.Request, partyID string, tail []string) {
	if len(tail) < 7 {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "insight route not found")
		return
	}
	modelID := strings.TrimSpace(tail[1])
	resourceType := strings.TrimSpace(tail[3])
	resourceID := strings.TrimSpace(tail[5])
	scope := fmt.Sprintf("party:%s/model:%s/resourceType:%s/resource:%s", partyID, modelID, resourceType, resourceID)

	collection, ok := azureDataManagerForAgriDataPlaneCanonicalCollection(tail[6])
	if !ok || (collection != "insights" && collection != "insight-attachments") {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "insight collection not found")
		return
	}
	if len(tail) == 7 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, collection, scope, true)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(tail[7])
	if id == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "resource id is required")
		return
	}
	defaults := map[string]any{
		"partyId":      partyID,
		"modelId":      modelID,
		"resourceType": resourceType,
		"resourceId":   resourceID,
	}
	if len(tail) == 9 && strings.EqualFold(tail[8], "file") && azureDataManagerForAgriDataPlaneSupportsFiles(collection) {
		s.azureDataManagerForAgriDataPlaneDownloadFile(w, scope, collection, id)
		return
	}
	s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, collection, scope, id, defaults)
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneSensorPartners(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) < 2 {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "sensor partner route not found")
		return
	}
	partnerID := strings.TrimSpace(tail[0])
	if partnerID == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "sensorPartnerId is required")
		return
	}
	collection, ok := azureDataManagerForAgriDataPlaneCanonicalCollection(tail[1])
	if !ok {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "collection not found")
		return
	}
	scope := "sensorPartner:" + partnerID
	if len(tail) == 2 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, collection, scope, true)
			return
		}
		if r.Method == http.MethodPost && collection == "sensor-events" {
			s.azureDataManagerForAgriDataPlaneAppendEvent(w, r, scope, collection, partnerID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	resourceID := strings.TrimSpace(tail[2])
	if resourceID == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "resource id is required")
		return
	}
	s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, collection, scope, resourceID, map[string]any{"sensorPartnerId": partnerID})
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneDatasets(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, "datasets", "global", false)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	datasetID := strings.TrimSpace(tail[0])
	if datasetID == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "datasetId is required")
		return
	}
	if len(tail) == 1 {
		s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, "datasets", "global", datasetID, nil)
		return
	}
	switch {
	case len(tail) == 2 && strings.Contains(strings.ToLower(strings.TrimSpace(tail[1])), "grant"):
		if r.Method == http.MethodPost {
			s.azureDataManagerForAgriDataPlaneGrantAccess(w, r, datasetID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	case len(tail) == 2 && strings.Contains(strings.ToLower(strings.TrimSpace(tail[1])), "authorize"):
		if r.Method == http.MethodPost {
			s.azureDataManagerForAgriDataPlaneGrantAccess(w, r, datasetID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	case len(tail) == 2 && strings.Contains(strings.ToLower(strings.TrimSpace(tail[1])), "access"):
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListGrants(w, datasetID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	case len(tail) == 3 && strings.EqualFold(tail[1], "records"):
		s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, "dataset-records", "dataset:"+datasetID, strings.TrimSpace(tail[2]), map[string]any{"datasetId": datasetID})
		return
	case len(tail) == 2 && strings.EqualFold(tail[1], "records"):
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, "dataset-records", "dataset:"+datasetID, true)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	default:
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "dataset route not found")
	}
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneOAuth(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "oauth route not found")
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "providers":
		if len(tail) == 1 {
			if r.Method == http.MethodGet {
				s.azureDataManagerForAgriDataPlaneListEntities(w, "oauth-providers", "global", false)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, "oauth-providers", "global", strings.TrimSpace(tail[1]), nil)
	case "tokens":
		if len(tail) == 1 {
			if r.Method == http.MethodGet {
				s.azureDataManagerForAgriDataPlaneListEntities(w, "oauth-tokens", "global", false)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if len(tail) == 2 && (strings.EqualFold(tail[1], ":connect") || strings.EqualFold(tail[1], "connect")) {
			if r.Method == http.MethodPost {
				s.azureDataManagerForAgriDataPlaneConnectOAuth(w, r)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, "oauth-tokens", "global", strings.TrimSpace(tail[1]), nil)
	default:
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "oauth route not found")
	}
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneWeather(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 && strings.EqualFold(segments[0], "weather-data:fetch") {
		if r.Method == http.MethodPost {
			s.azureDataManagerForAgriDataPlaneFetchWeather(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(segments) == 1 && strings.EqualFold(segments[0], "weather-data") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{
				"value": []any{azureDataManagerForAgriDataPlaneWeatherRecordFixture("weather-record-1", "party-a")},
			})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(segments) == 2 && strings.EqualFold(segments[0], "weather-data") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, azureDataManagerForAgriDataPlaneWeatherRecordFixture(strings.TrimSpace(segments[1]), "party-a"))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "weather route not found")
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneSolutions(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, "solutions", "global", false)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 1 && strings.Contains(tail[0], ":") {
		id, action, ok := strings.Cut(strings.TrimSpace(tail[0]), ":")
		if !ok || strings.TrimSpace(id) == "" || strings.TrimSpace(action) == "" {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "solution action is invalid")
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch strings.ToLower(strings.TrimSpace(action)) {
		case "create", "fetch":
			s.azureDataManagerForAgriDataPlaneRunSolutionAction(w, r, strings.TrimSpace(id), strings.ToLower(strings.TrimSpace(action)))
			return
		default:
			azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "solution action not found")
			return
		}
	}
	s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, "solutions", "global", strings.TrimSpace(tail[0]), nil)
}

func (s *Server) handleAzureDataManagerForAgriDataPlaneTopLevel(w http.ResponseWriter, r *http.Request, segments []string) {
	head := strings.TrimSpace(segments[0])
	if strings.Contains(head, ":") {
		collectionToken, action, ok := strings.Cut(head, ":")
		if !ok {
			azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		collection, valid := azureDataManagerForAgriDataPlaneCanonicalCollection(collectionToken)
		if !valid {
			azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "collection not found")
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		switch strings.ToLower(strings.TrimSpace(action)) {
		case "search":
			s.azureDataManagerForAgriDataPlaneSearchEntities(w, r, collection)
			return
		case "fetch":
			if collection == "weather-data" {
				s.azureDataManagerForAgriDataPlaneFetchWeather(w, r)
				return
			}
		}
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "action not found")
		return
	}

	collection, ok := azureDataManagerForAgriDataPlaneCanonicalCollection(head)
	if !ok {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	if len(segments) == 1 {
		if r.Method == http.MethodGet {
			s.azureDataManagerForAgriDataPlaneListEntities(w, collection, "global", false)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(segments) == 2 && strings.EqualFold(segments[1], "file") && azureDataManagerForAgriDataPlaneSupportsFiles(collection) {
		s.azureDataManagerForAgriDataPlaneDownloadFile(w, "global", collection, "")
		return
	}
	s.azureDataManagerForAgriDataPlaneEntityRoute(w, r, collection, "global", strings.TrimSpace(segments[1]), nil)
}

func (s *Server) azureDataManagerForAgriDataPlaneEntityRoute(w http.ResponseWriter, r *http.Request, collection, scope, id string, defaults map[string]any) {
	if strings.TrimSpace(id) == "" {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "resource id is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.azureDataManagerForAgriDataPlaneGetEntity(w, collection, scope, id, false)
	case http.MethodHead:
		s.azureDataManagerForAgriDataPlaneGetEntity(w, collection, scope, id, true)
	case http.MethodPatch, http.MethodPut:
		s.azureDataManagerForAgriDataPlaneUpsertEntity(w, r, collection, scope, id, defaults)
	case http.MethodDelete:
		s.azureDataManagerForAgriDataPlaneDeleteEntity(w, collection, scope, id)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) azureDataManagerForAgriDataPlaneGetEntity(w http.ResponseWriter, collection, scope, id string, headOnly bool) {
	s.providerStorageMu.Lock()
	entity := s.azureDataManagerForAgriDataPlaneEntities[azureDataManagerForAgriDataPlaneEntityKey(scope, collection, id)]
	s.providerStorageMu.Unlock()
	if entity == nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
		return
	}
	if headOnly {
		w.Header().Set("ETag", entity.ETag)
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureDataManagerForAgriDataPlaneEntityFixture(entity))
}

func (s *Server) azureDataManagerForAgriDataPlaneUpsertEntity(w http.ResponseWriter, r *http.Request, collection, scope, id string, defaults map[string]any) {
	payload, filePayload, err := azureDataManagerForAgriDataPlaneDecodeUpsertBody(r, collection)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", err.Error())
		return
	}

	now := time.Now().UTC()
	key := azureDataManagerForAgriDataPlaneEntityKey(scope, collection, id)

	s.providerStorageMu.Lock()
	existing := s.azureDataManagerForAgriDataPlaneEntities[key]
	created := existing == nil
	if existing == nil {
		existing = &azureDataManagerForAgriDataPlaneEntity{
			Collection: collection,
			Scope:      scope,
			ID:         id,
			Payload:    map[string]any{},
			CreatedAt:  now,
		}
	}
	for key, value := range defaults {
		existing.Payload[key] = value
	}
	for key, value := range payload {
		existing.Payload[key] = value
	}
	if _, ok := existing.Payload["name"]; !ok {
		existing.Payload["name"] = id
	}
	if _, ok := existing.Payload["status"]; !ok {
		existing.Payload["status"] = "Active"
	}
	existing.UpdatedAt = now
	existing.ETag = azureDataManagerForAgriDataPlaneETag(scope, collection, id, now.Format(time.RFC3339Nano))
	s.azureDataManagerForAgriDataPlaneEntities[key] = existing

	if filePayload != nil {
		s.azureDataManagerForAgriDataPlaneFiles[azureDataManagerForAgriDataPlaneFileKey(scope, collection, id)] = filePayload
	}
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureDataManagerForAgriDataPlaneEntityFixture(existing))
}

func (s *Server) azureDataManagerForAgriDataPlaneDeleteEntity(w http.ResponseWriter, collection, scope, id string) {
	key := azureDataManagerForAgriDataPlaneEntityKey(scope, collection, id)
	fileKey := azureDataManagerForAgriDataPlaneFileKey(scope, collection, id)

	s.providerStorageMu.Lock()
	_, exists := s.azureDataManagerForAgriDataPlaneEntities[key]
	if exists {
		delete(s.azureDataManagerForAgriDataPlaneEntities, key)
		delete(s.azureDataManagerForAgriDataPlaneFiles, fileKey)
	}
	s.providerStorageMu.Unlock()
	if !exists {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) azureDataManagerForAgriDataPlaneListEntities(w http.ResponseWriter, collection, scope string, exactScope bool) {
	s.providerStorageMu.Lock()
	items := make([]*azureDataManagerForAgriDataPlaneEntity, 0)
	for _, entity := range s.azureDataManagerForAgriDataPlaneEntities {
		if entity.Collection != collection {
			continue
		}
		if exactScope {
			if entity.Scope != scope {
				continue
			}
		} else {
			if scope != "global" {
				if !strings.HasPrefix(entity.Scope, scope) {
					continue
				}
			} else {
				if _, isGlobal := azureDataManagerForAgriDataPlaneGlobalCollections[collection]; !isGlobal && entity.Scope != "global" {
					continue
				}
			}
		}
		items = append(items, entity)
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureDataManagerForAgriDataPlaneEntityFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) azureDataManagerForAgriDataPlaneSearchEntities(w http.ResponseWriter, r *http.Request, collection string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
			return
		}
	}
	partyID := strings.TrimSpace(azureDataManagerForAgriDataPlaneString(payload["partyId"]))

	s.providerStorageMu.Lock()
	items := make([]*azureDataManagerForAgriDataPlaneEntity, 0)
	for _, entity := range s.azureDataManagerForAgriDataPlaneEntities {
		if entity.Collection != collection {
			continue
		}
		if partyID != "" {
			if storedPartyID := strings.TrimSpace(azureDataManagerForAgriDataPlaneString(entity.Payload["partyId"])); !strings.EqualFold(storedPartyID, partyID) {
				continue
			}
		}
		items = append(items, entity)
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	values := make([]any, 0, len(items))
	for _, item := range items {
		values = append(values, azureDataManagerForAgriDataPlaneEntityFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) azureDataManagerForAgriDataPlaneAppendEvent(w http.ResponseWriter, r *http.Request, scope, collection, partnerID string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
			return
		}
	}
	id := "event-" + azureDataManagerForAgriDataPlaneStableToken(partnerID, time.Now().UTC().Format(time.RFC3339Nano))
	s.azureDataManagerForAgriDataPlaneUpsertEntity(w, cloneRequestWithBody(r, body), collection, scope, id, map[string]any{"sensorPartnerId": partnerID, "eventType": azureDataManagerForAgriDataPlaneStringOr(payload["eventType"], "Telemetry")})
}

func (s *Server) azureDataManagerForAgriDataPlaneGrantAccess(w http.ResponseWriter, r *http.Request, datasetID string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
			return
		}
	}
	now := time.Now().UTC()
	grantID := "grant-" + azureDataManagerForAgriDataPlaneStableToken(datasetID, now.Format(time.RFC3339Nano))
	grant := &azureDataManagerForAgriDataPlaneGrant{
		DatasetID: datasetID,
		GrantID:   grantID,
		Payload: map[string]any{
			"id":          grantID,
			"datasetId":   datasetID,
			"grantee":     azureDataManagerForAgriDataPlaneStringOr(payload["grantee"], "stackyard@example.com"),
			"permissions": payload["permissions"],
			"expiresOn":   now.Add(24 * time.Hour).Format(time.RFC3339),
		},
		CreatedAt: now,
	}
	if grant.Payload["permissions"] == nil {
		grant.Payload["permissions"] = []any{"read"}
	}
	s.providerStorageMu.Lock()
	s.azureDataManagerForAgriDataPlaneDatasetGrants[datasetID+"|"+grantID] = grant
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, grant.Payload)
}

func (s *Server) azureDataManagerForAgriDataPlaneListGrants(w http.ResponseWriter, datasetID string) {
	s.providerStorageMu.Lock()
	values := make([]any, 0)
	for _, grant := range s.azureDataManagerForAgriDataPlaneDatasetGrants {
		if grant.DatasetID == datasetID {
			values = append(values, grant.Payload)
		}
	}
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) azureDataManagerForAgriDataPlaneFetchWeather(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
			return
		}
	}
	partyID := azureDataManagerForAgriDataPlaneStringOr(payload["partyId"], "party-a")
	response := map[string]any{
		"id":      "weather-fetch-" + azureDataManagerForAgriDataPlaneStableToken(partyID, time.Now().UTC().Format(time.RFC3339Nano)),
		"status":  "Succeeded",
		"partyId": partyID,
		"value": []any{
			azureDataManagerForAgriDataPlaneWeatherRecordFixture("weather-record-1", partyID),
			azureDataManagerForAgriDataPlaneWeatherRecordFixture("weather-record-2", partyID),
		},
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) azureDataManagerForAgriDataPlaneConnectOAuth(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
			return
		}
	}
	now := time.Now().UTC()
	tokenID := "oauth-token-" + azureDataManagerForAgriDataPlaneStableToken(now.Format(time.RFC3339Nano))
	entity := &azureDataManagerForAgriDataPlaneEntity{
		Collection: "oauth-tokens",
		Scope:      "global",
		ID:         tokenID,
		Payload: map[string]any{
			"id":           tokenID,
			"name":         tokenID,
			"status":       "Connected",
			"providerName": azureDataManagerForAgriDataPlaneStringOr(payload["providerName"], "demo-provider"),
			"partyId":      azureDataManagerForAgriDataPlaneStringOr(payload["partyId"], "party-a"),
			"redirectUri":  "https://example.test/oauth/callback",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureDataManagerForAgriDataPlaneETag("global", "oauth-tokens", tokenID, now.Format(time.RFC3339Nano)),
	}
	s.providerStorageMu.Lock()
	s.azureDataManagerForAgriDataPlaneEntities[azureDataManagerForAgriDataPlaneEntityKey("global", "oauth-tokens", tokenID)] = entity
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{
		"id":             tokenID,
		"status":         "Succeeded",
		"connectionLink": "https://login.microsoftonline.com/common/oauth2/v2.0/authorize?client_id=stackyard",
	})
}

func (s *Server) azureDataManagerForAgriDataPlaneRunSolutionAction(w http.ResponseWriter, r *http.Request, solutionID, action string) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return
	}
	var requestPayload map[string]any
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &requestPayload); err != nil {
			azureDataManagerForAgriDataPlaneError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
			return
		}
	}
	now := time.Now().UTC()
	jobID := "solution-" + action + "-" + azureDataManagerForAgriDataPlaneStableToken(solutionID, now.Format(time.RFC3339Nano))
	job := &azureDataManagerForAgriDataPlaneJob{
		Kind: action,
		ID:   jobID,
		Payload: map[string]any{
			"id":         jobID,
			"solutionId": solutionID,
			"action":     action,
			"status":     "Succeeded",
			"request":    requestPayload,
		},
		CreatedAt: now,
	}
	s.providerStorageMu.Lock()
	s.azureDataManagerForAgriDataPlaneJobs[jobID] = job
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, job.Payload)
}

func (s *Server) azureDataManagerForAgriDataPlaneDownloadFile(w http.ResponseWriter, scope, collection, id string) {
	s.providerStorageMu.Lock()
	file := s.azureDataManagerForAgriDataPlaneFiles[azureDataManagerForAgriDataPlaneFileKey(scope, collection, id)]
	s.providerStorageMu.Unlock()
	if file == nil {
		azureDataManagerForAgriDataPlaneError(w, http.StatusNotFound, "ResourceNotFound", "file not found")
		return
	}
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("ETag", azureDataManagerForAgriDataPlaneETag(scope, collection, file.FileName, file.UpdatedAt.Format(time.RFC3339Nano)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Content)
}

func azureDataManagerForAgriDataPlaneDecodeUpsertBody(r *http.Request, collection string) (map[string]any, *azureDataManagerForAgriDataPlaneFile, error) {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/") {
		if !azureDataManagerForAgriDataPlaneSupportsFiles(collection) {
			return nil, nil, fmt.Errorf("multipart uploads are not supported for %s", collection)
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, nil, fmt.Errorf("request must be valid multipart form data")
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return nil, nil, fmt.Errorf("multipart request must include a file field")
		}
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read uploaded file")
		}
		payload := map[string]any{}
		for key, values := range r.MultipartForm.Value {
			if len(values) == 1 {
				payload[key] = values[0]
			} else {
				items := make([]any, 0, len(values))
				for _, value := range values {
					items = append(items, value)
				}
				payload[key] = items
			}
		}
		contentDisposition := header.Header.Get("Content-Disposition")
		_, params, _ := mime.ParseMediaType(contentDisposition)
		fileName := strings.TrimSpace(header.Filename)
		if fileName == "" {
			fileName = strings.TrimSpace(params["filename"])
		}
		if fileName == "" {
			fileName = "stackyard.bin"
		}
		fileContentType := strings.TrimSpace(header.Header.Get("Content-Type"))
		if fileContentType == "" {
			fileContentType = "application/octet-stream"
		}
		payload["fileName"] = fileName
		payload["contentType"] = fileContentType
		payload["sizeInBytes"] = len(content)
		payload["contentEncoding"] = "binary"
		return payload, &azureDataManagerForAgriDataPlaneFile{
			Collection:  collection,
			FileName:    fileName,
			ContentType: fileContentType,
			Content:     content,
			UpdatedAt:   time.Now().UTC(),
		}, nil
	}

	body, err := readBodyBytes(r)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read request body")
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil, nil, fmt.Errorf("request body is required")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, nil, fmt.Errorf("request body must be valid JSON")
	}
	if azureDataManagerForAgriDataPlaneSupportsFiles(collection) {
		if encoded := strings.TrimSpace(azureDataManagerForAgriDataPlaneString(payload["content"])); encoded != "" {
			content, err := base64.StdEncoding.DecodeString(encoded)
			if err == nil {
				return payload, &azureDataManagerForAgriDataPlaneFile{
					Collection:  collection,
					FileName:    azureDataManagerForAgriDataPlaneStringOr(payload["fileName"], "stackyard.bin"),
					ContentType: azureDataManagerForAgriDataPlaneStringOr(payload["contentType"], "application/octet-stream"),
					Content:     content,
					UpdatedAt:   time.Now().UTC(),
				}, nil
			}
		}
	}
	return payload, nil, nil
}

func azureDataManagerForAgriDataPlaneEntityFixture(entity *azureDataManagerForAgriDataPlaneEntity) map[string]any {
	out := map[string]any{
		"id":               entity.ID,
		"name":             entity.Payload["name"],
		"type":             entity.Collection,
		"eTag":             entity.ETag,
		"status":           entity.Payload["status"],
		"createdDateTime":  entity.CreatedAt.Format(time.RFC3339),
		"modifiedDateTime": entity.UpdatedAt.Format(time.RFC3339),
		"createdBy":        "stackyard",
		"modifiedBy":       "stackyard",
	}
	for key, value := range entity.Payload {
		out[key] = value
	}
	return out
}

func azureDataManagerForAgriDataPlaneWeatherRecordFixture(id, partyID string) map[string]any {
	return map[string]any{
		"id":              id,
		"partyId":         partyID,
		"dateTime":        time.Now().UTC().Format(time.RFC3339),
		"temperature":     22.5,
		"humidity":        0.48,
		"precipitation":   0.0,
		"windSpeed":       4.7,
		"weatherDataType": "Forecast",
	}
}

func azureDataManagerForAgriDataPlaneCanonicalCollection(raw string) (string, bool) {
	value := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "_", "-"))
	value = strings.ReplaceAll(value, " ", "")
	display, ok := azureDataManagerForAgriDataPlaneSupportedCollections[value]
	return display, ok
}

func azureDataManagerForAgriDataPlaneSupportsFiles(collection string) bool {
	switch strings.ToLower(strings.TrimSpace(collection)) {
	case "attachments", "insight-attachments", "scenes", "image-processing":
		return true
	default:
		return false
	}
}

func azureDataManagerForAgriDataPlaneEntityKey(scope, collection, id string) string {
	return scope + "|" + collection + "|" + id
}

func azureDataManagerForAgriDataPlaneFileKey(scope, collection, id string) string {
	return scope + "|" + collection + "|" + id
}

func azureDataManagerForAgriDataPlaneStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:12]
}

func azureDataManagerForAgriDataPlaneETag(parts ...string) string {
	return `W/"` + azureDataManagerForAgriDataPlaneStableToken(parts...) + `"`
}

func azureDataManagerForAgriDataPlaneString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureDataManagerForAgriDataPlaneStringOr(value any, fallback string) string {
	if out := strings.TrimSpace(azureDataManagerForAgriDataPlaneString(value)); out != "" {
		return out
	}
	return fallback
}

func azureDataManagerForAgriDataPlaneError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-ms-error-code", code)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func cloneRequestWithBody(r *http.Request, body []byte) *http.Request {
	clone := r.Clone(r.Context())
	clone.Body = io.NopCloser(strings.NewReader(string(body)))
	clone.ContentLength = int64(len(body))
	return clone
}
