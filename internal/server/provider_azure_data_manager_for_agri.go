package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureDataManagerForAgriAPIVersion = "2023-06-01-preview"

type azureDataManagerForAgriFarmBeat struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Identity       map[string]any
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureDataManagerForAgriChildResource struct {
	SubscriptionID string
	ResourceGroup  string
	ParentName     string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureDataManagerForAgriOperationResult struct {
	SubscriptionID string
	Location       string
	ID             string
	Action         string
	Status         string
	StartTime      time.Time
	EndTime        time.Time
}

var azureDataManagerForAgriMutableCollections = map[string]string{
	"dataconnectors":             "dataConnectors",
	"extensions":                 "extensions",
	"privateendpointconnections": "privateEndpointConnections",
	"solutions":                  "solutions",
}

func (s *Server) handleAzureDataManagerForAgriRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsDataManagerForAgriPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureDataManagerForAgriAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AgFoodPlatform") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureDataManagerForAgriOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureDataManagerForAgriRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		s.handleAzureDataManagerForAgriSubscriptionRoute(w, r, path, subscriptionID, tail)
		return true
	}

	s.handleAzureDataManagerForAgriResourceGroupRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsDataManagerForAgriPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AgFoodPlatform") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureDataManagerForAgriRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability", "locations", "farmbeats":
		return true
	default:
		return false
	}
}

func azureDataManagerForAgriAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureDataManagerForAgriRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.AgFoodPlatform") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func (s *Server) handleAzureDataManagerForAgriSubscriptionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) == 1 && strings.EqualFold(tail[0], "checkNameAvailability") {
		if r.Method == http.MethodPost {
			s.handleAzureDataManagerForAgriCheckNameAvailability(w, r, path, subscriptionID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 4 &&
		strings.EqualFold(tail[0], "locations") &&
		strings.EqualFold(tail[2], "operationResults") &&
		r.Method == http.MethodGet {
		location := strings.TrimSpace(tail[1])
		operationResultID := strings.TrimSpace(tail[3])
		if location == "" || operationResultID == "" {
			respondAzureInvalidRequest(w, path, "location and operation result ID are required")
			return
		}
		s.handleAzureDataManagerForAgriGetOperationResult(w, path, subscriptionID, location, operationResultID)
		return
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "farmBeats") && r.Method == http.MethodGet {
		s.handleAzureDataManagerForAgriListFarmBeatsBySubscription(w, subscriptionID)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureDataManagerForAgriResourceGroupRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && strings.EqualFold(tail[0], "farmBeats") && r.Method == http.MethodGet {
		s.handleAzureDataManagerForAgriListFarmBeatsByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if len(tail) < 2 || !strings.EqualFold(tail[0], "farmBeats") {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "farmBeats resource name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureDataManagerForAgriGetFarmBeat(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureDataManagerForAgriGetFarmBeat(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureDataManagerForAgriCreateOrUpdateFarmBeat(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureDataManagerForAgriPatchFarmBeat(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureDataManagerForAgriDeleteFarmBeat(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	s.handleAzureDataManagerForAgriChildRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
}

func (s *Server) handleAzureDataManagerForAgriChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, farmBeatName string, tail []string) {
	if !s.azureDataManagerForAgriFarmBeatExists(subscriptionID, resourceGroup, farmBeatName) {
		respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, farmBeatName))
		return
	}
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}

	head := strings.ToLower(strings.TrimSpace(tail[0]))
	switch head {
	case "privatelinkresources":
		s.handleAzureDataManagerForAgriPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, farmBeatName, tail[1:])
		return
	case "farmbeatsextensiondefinitions":
		s.handleAzureDataManagerForAgriExtensionDefinitionsRoute(w, r, path, subscriptionID, resourceGroup, farmBeatName, tail[1:])
		return
	case "farmbeatssolutiondefinitions":
		s.handleAzureDataManagerForAgriSolutionDefinitionsRoute(w, r, path, subscriptionID, resourceGroup, farmBeatName, tail[1:])
		return
	}

	collection, ok := azureDataManagerForAgriMutableCollections[head]
	if !ok {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			s.handleAzureDataManagerForAgriChildList(w, subscriptionID, resourceGroup, farmBeatName, collection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "child resource name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureDataManagerForAgriChildGet(w, path, subscriptionID, resourceGroup, farmBeatName, collection, name)
	case http.MethodPut:
		s.handleAzureDataManagerForAgriChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, farmBeatName, collection, name)
	case http.MethodPatch:
		s.handleAzureDataManagerForAgriChildPatch(w, r, path, subscriptionID, resourceGroup, farmBeatName, collection, name)
	case http.MethodDelete:
		s.handleAzureDataManagerForAgriChildDelete(w, path, subscriptionID, resourceGroup, farmBeatName, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureDataManagerForAgriCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_, payload, ok := azureDataManagerForAgriReadRequestBody(w, r, path)
	if !ok {
		return
	}

	name := strings.TrimSpace(azureDataManagerForAgriString(payload["name"]))
	resourceType := strings.TrimSpace(azureDataManagerForAgriString(payload["type"]))
	if name == "" {
		respondAzureInvalidRequest(w, path, "name is required")
		return
	}
	if resourceType == "" {
		respondAzureInvalidRequest(w, path, "type is required")
		return
	}
	if !strings.EqualFold(resourceType, "Microsoft.AgFoodPlatform/farmBeats") {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "type must be Microsoft.AgFoodPlatform/farmBeats",
		})
		return
	}

	nameAvailable := true
	s.providerStorageMu.Lock()
	for _, record := range s.azureDataManagerForAgriFarmBeats {
		if strings.EqualFold(record.Name, name) {
			nameAvailable = false
			break
		}
	}
	s.providerStorageMu.Unlock()

	response := map[string]any{"nameAvailable": nameAvailable}
	if !nameAvailable {
		response["reason"] = "AlreadyExists"
		response["message"] = fmt.Sprintf("farmBeats resource %q already exists", name)
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleAzureDataManagerForAgriGetOperationResult(w http.ResponseWriter, path, subscriptionID, location, operationResultID string) {
	s.providerStorageMu.Lock()
	record := s.azureDataManagerForAgriOperationResults[azureDataManagerForAgriOperationResultKey(subscriptionID, location, operationResultID)]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, operationResultID))
		return
	}
	respondJSON(w, http.StatusOK, azureDataManagerForAgriOperationResultFixture(record))
}

func (s *Server) handleAzureDataManagerForAgriCreateOrUpdateFarmBeat(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureDataManagerForAgriReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureDataManagerForAgriString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	key := azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	record := s.azureDataManagerForAgriFarmBeats[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureDataManagerForAgriFarmBeat{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}

	record.Location = location
	record.Tags = azureDataManagerForAgriStringMap(payload["tags"])
	record.Identity = azureDataManagerForAgriObject(payload["identity"])
	record.Properties = azureDataManagerForAgriNormalizeFarmBeatProperties(name, record.Location, createdAt, now, azureDataManagerForAgriObject(payload["properties"]))
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureDataManagerForAgriETag(key, body, now)
	s.azureDataManagerForAgriFarmBeats[key] = record
	operationPath := s.azureDataManagerForAgriRecordOperationLocked(subscriptionID, record.Location, "farmBeats.write", key, now)
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("Location", operationPath)
	respondJSON(w, status, azureDataManagerForAgriFarmBeatFixture(record))
}

func (s *Server) handleAzureDataManagerForAgriPatchFarmBeat(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureDataManagerForAgriReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	record := s.azureDataManagerForAgriFarmBeats[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["location"]; ok {
		location := strings.TrimSpace(azureDataManagerForAgriString(payload["location"]))
		if location == "" {
			s.providerStorageMu.Unlock()
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		record.Location = location
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		record.Tags = azureDataManagerForAgriStringMap(payload["tags"])
		updated = true
	}
	if _, ok := payload["identity"]; ok {
		record.Identity = azureDataManagerForAgriObject(payload["identity"])
		updated = true
	}
	if _, ok := payload["properties"]; ok {
		record.Properties = azureDataManagerForAgriMergeObjects(record.Properties, azureDataManagerForAgriObject(payload["properties"]))
		record.Properties = azureDataManagerForAgriNormalizeFarmBeatProperties(name, record.Location, record.CreatedAt, now, record.Properties)
		updated = true
	}
	if !updated {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	record.UpdatedAt = now
	record.ETag = azureDataManagerForAgriETag(key, body, now)
	operationPath := s.azureDataManagerForAgriRecordOperationLocked(subscriptionID, record.Location, "farmBeats.patch", key, now)
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	respondJSON(w, http.StatusOK, azureDataManagerForAgriFarmBeatFixture(record))
}

func (s *Server) handleAzureDataManagerForAgriGetFarmBeat(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureDataManagerForAgriFarmBeats[azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureDataManagerForAgriFarmBeatFixture(record))
}

func (s *Server) handleAzureDataManagerForAgriDeleteFarmBeat(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	record := s.azureDataManagerForAgriFarmBeats[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureDataManagerForAgriFarmBeats, key)
	s.azureDataManagerForAgriDeleteChildrenLocked(subscriptionID, resourceGroup, name)
	operationPath := s.azureDataManagerForAgriRecordOperationLocked(subscriptionID, record.Location, "farmBeats.delete", key, time.Now().UTC())
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	respondJSON(w, http.StatusOK, azureDataManagerForAgriFarmBeatFixture(record))
}

func (s *Server) handleAzureDataManagerForAgriListFarmBeatsByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureDataManagerForAgriFarmBeats {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) {
			items = append(items, azureDataManagerForAgriFarmBeatFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureDataManagerForAgriString(items[i]["id"])) < strings.ToLower(azureDataManagerForAgriString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureDataManagerForAgriListFarmBeatsBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureDataManagerForAgriFarmBeats {
		if record.SubscriptionID == subscriptionID {
			items = append(items, azureDataManagerForAgriFarmBeatFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureDataManagerForAgriString(items[i]["id"])) < strings.ToLower(azureDataManagerForAgriString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureDataManagerForAgriChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, farmBeatName, collection, name string) {
	body, payload, ok := azureDataManagerForAgriReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureDataManagerForAgriString(payload["location"]))
	if location == "" {
		s.providerStorageMu.Lock()
		parent := s.azureDataManagerForAgriFarmBeats[azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, farmBeatName)]
		s.providerStorageMu.Unlock()
		if parent != nil {
			location = parent.Location
		}
	}

	now := time.Now().UTC()
	key := azureDataManagerForAgriChildKey(subscriptionID, resourceGroup, farmBeatName, collection, name)
	s.providerStorageMu.Lock()
	resource := s.azureDataManagerForAgriChildMap(collection)[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureDataManagerForAgriChildResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ParentName:     farmBeatName,
			Collection:     collection,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}
	resource.Location = location
	resource.Tags = azureDataManagerForAgriStringMap(payload["tags"])
	resource.Properties = azureDataManagerForAgriNormalizeChildProperties(collection, name, azureDataManagerForAgriObject(payload["properties"]))
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureDataManagerForAgriETag(key, body, now)
	s.azureDataManagerForAgriChildMap(collection)[key] = resource
	operationPath := s.azureDataManagerForAgriRecordOperationLocked(subscriptionID, s.azureDataManagerForAgriLocationForChildLocked(subscriptionID, resourceGroup, farmBeatName, location), collection+".write", key, now)
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	respondJSON(w, status, azureDataManagerForAgriChildFixture(resource))
}

func (s *Server) handleAzureDataManagerForAgriChildPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, farmBeatName, collection, name string) {
	body, payload, ok := azureDataManagerForAgriReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureDataManagerForAgriChildKey(subscriptionID, resourceGroup, farmBeatName, collection, name)
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	resource := s.azureDataManagerForAgriChildMap(collection)[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	updated := false
	if _, ok := payload["location"]; ok {
		location := strings.TrimSpace(azureDataManagerForAgriString(payload["location"]))
		if location == "" {
			s.providerStorageMu.Unlock()
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		resource.Location = location
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureDataManagerForAgriStringMap(payload["tags"])
		updated = true
	}
	if _, ok := payload["properties"]; ok {
		resource.Properties = azureDataManagerForAgriMergeObjects(resource.Properties, azureDataManagerForAgriObject(payload["properties"]))
		resource.Properties = azureDataManagerForAgriNormalizeChildProperties(collection, name, resource.Properties)
		updated = true
	}
	if !updated {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	resource.UpdatedAt = now
	resource.ETag = azureDataManagerForAgriETag(key, body, now)
	operationPath := s.azureDataManagerForAgriRecordOperationLocked(subscriptionID, s.azureDataManagerForAgriLocationForChildLocked(subscriptionID, resourceGroup, farmBeatName, resource.Location), collection+".patch", key, now)
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	respondJSON(w, http.StatusOK, azureDataManagerForAgriChildFixture(resource))
}

func (s *Server) handleAzureDataManagerForAgriChildGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, farmBeatName, collection, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureDataManagerForAgriChildMap(collection)[azureDataManagerForAgriChildKey(subscriptionID, resourceGroup, farmBeatName, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureDataManagerForAgriChildFixture(resource))
}

func (s *Server) handleAzureDataManagerForAgriChildDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, farmBeatName, collection, name string) {
	key := azureDataManagerForAgriChildKey(subscriptionID, resourceGroup, farmBeatName, collection, name)
	s.providerStorageMu.Lock()
	resource := s.azureDataManagerForAgriChildMap(collection)[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureDataManagerForAgriChildMap(collection), key)
	operationPath := s.azureDataManagerForAgriRecordOperationLocked(subscriptionID, s.azureDataManagerForAgriLocationForChildLocked(subscriptionID, resourceGroup, farmBeatName, resource.Location), collection+".delete", key, time.Now().UTC())
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	respondJSON(w, http.StatusOK, azureDataManagerForAgriChildFixture(resource))
}

func (s *Server) handleAzureDataManagerForAgriChildList(w http.ResponseWriter, subscriptionID, resourceGroup, farmBeatName, collection string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, resource := range s.azureDataManagerForAgriChildMap(collection) {
		if resource.SubscriptionID == subscriptionID &&
			strings.EqualFold(resource.ResourceGroup, resourceGroup) &&
			strings.EqualFold(resource.ParentName, farmBeatName) {
			items = append(items, azureDataManagerForAgriChildFixture(resource))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureDataManagerForAgriString(items[i]["id"])) < strings.ToLower(azureDataManagerForAgriString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureDataManagerForAgriPrivateLinkResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, farmBeatName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	items := azureDataManagerForAgriPrivateLinkResourcesFixtures(subscriptionID, resourceGroup, farmBeatName)
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
		return
	}
	name := strings.TrimSpace(tail[0])
	for _, item := range items {
		if strings.EqualFold(azureDataManagerForAgriString(item["name"]), name) {
			respondJSON(w, http.StatusOK, item)
			return
		}
	}
	respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
}

func (s *Server) handleAzureDataManagerForAgriExtensionDefinitionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, farmBeatName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	items := azureDataManagerForAgriExtensionDefinitionsFixtures(subscriptionID, resourceGroup, farmBeatName)
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
		return
	}
	name := strings.TrimSpace(tail[0])
	for _, item := range items {
		if strings.EqualFold(azureDataManagerForAgriString(item["name"]), name) {
			respondJSON(w, http.StatusOK, item)
			return
		}
	}
	respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
}

func (s *Server) handleAzureDataManagerForAgriSolutionDefinitionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, farmBeatName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	items := azureDataManagerForAgriSolutionDefinitionsFixtures(subscriptionID, resourceGroup, farmBeatName)
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
		return
	}
	name := strings.TrimSpace(tail[0])
	for _, item := range items {
		if strings.EqualFold(azureDataManagerForAgriString(item["name"]), name) {
			respondJSON(w, http.StatusOK, item)
			return
		}
	}
	respondJSON(w, http.StatusNotFound, azureDataManagerForAgriNotFound(path, name))
}

func (s *Server) azureDataManagerForAgriFarmBeatExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureDataManagerForAgriFarmBeats[azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, name)] != nil
}

func (s *Server) azureDataManagerForAgriChildMap(collection string) map[string]*azureDataManagerForAgriChildResource {
	switch collection {
	case "dataConnectors":
		return s.azureDataManagerForAgriDataConnectors
	case "extensions":
		return s.azureDataManagerForAgriExtensions
	case "privateEndpointConnections":
		return s.azureDataManagerForAgriPrivateEndpointConnections
	case "solutions":
		return s.azureDataManagerForAgriSolutions
	default:
		return map[string]*azureDataManagerForAgriChildResource{}
	}
}

func (s *Server) azureDataManagerForAgriDeleteChildrenLocked(subscriptionID, resourceGroup, farmBeatName string) {
	prefix := strings.ToLower(azureDataManagerForAgriChildParentPrefix(subscriptionID, resourceGroup, farmBeatName))
	for key := range s.azureDataManagerForAgriDataConnectors {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureDataManagerForAgriDataConnectors, key)
		}
	}
	for key := range s.azureDataManagerForAgriExtensions {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureDataManagerForAgriExtensions, key)
		}
	}
	for key := range s.azureDataManagerForAgriPrivateEndpointConnections {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureDataManagerForAgriPrivateEndpointConnections, key)
		}
	}
	for key := range s.azureDataManagerForAgriSolutions {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureDataManagerForAgriSolutions, key)
		}
	}
}

func (s *Server) azureDataManagerForAgriRecordOperationLocked(subscriptionID, location, action, scope string, now time.Time) string {
	location = strings.TrimSpace(location)
	if location == "" {
		location = "eastus"
	}
	operationResultID := azureDataManagerForAgriStableToken(action + "|" + scope + "|" + now.Format(time.RFC3339Nano))
	record := &azureDataManagerForAgriOperationResult{
		SubscriptionID: subscriptionID,
		Location:       location,
		ID:             operationResultID,
		Action:         action,
		Status:         "Succeeded",
		StartTime:      now,
		EndTime:        now,
	}
	s.azureDataManagerForAgriOperationResults[azureDataManagerForAgriOperationResultKey(subscriptionID, location, operationResultID)] = record
	return "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.AgFoodPlatform/locations/" + location + "/operationResults/" + operationResultID + "?api-version=" + azureDataManagerForAgriAPIVersion
}

func (s *Server) azureDataManagerForAgriLocationForChildLocked(subscriptionID, resourceGroup, farmBeatName, childLocation string) string {
	if strings.TrimSpace(childLocation) != "" {
		return childLocation
	}
	parent := s.azureDataManagerForAgriFarmBeats[azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, farmBeatName)]
	if parent != nil && strings.TrimSpace(parent.Location) != "" {
		return parent.Location
	}
	return "eastus"
}

func azureDataManagerForAgriFarmBeatFixture(record *azureDataManagerForAgriFarmBeat) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":       azureDataManagerForAgriFarmBeatID(record.SubscriptionID, record.ResourceGroup, record.Name),
		"name":     record.Name,
		"type":     "Microsoft.AgFoodPlatform/farmBeats",
		"location": record.Location,
		"identity": azureDataManagerForAgriNilIfEmpty(record.Identity),
		"tags":     azureDataManagerForAgriStringMapAny(record.Tags),
		"etag":     record.ETag,
		"properties": azureDataManagerForAgriMergeObjects(map[string]any{
			"provisioningState":   "Succeeded",
			"instanceUri":         "https://" + record.Name + ".farmbeats.azure.net",
			"publicNetworkAccess": "Enabled",
			"sensorIntegration": map[string]any{
				"enabled":           false,
				"provisioningState": "Succeeded",
			},
			"privateEndpointConnections": []any{},
		}, record.Properties),
		"systemData": azureDataManagerForAgriSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureDataManagerForAgriChildFixture(resource *azureDataManagerForAgriChildResource) map[string]any {
	if resource == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":         azureDataManagerForAgriChildID(resource.SubscriptionID, resource.ResourceGroup, resource.ParentName, resource.Collection, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.AgFoodPlatform/farmBeats/" + resource.Collection,
		"location":   resource.Location,
		"tags":       azureDataManagerForAgriStringMapAny(resource.Tags),
		"eTag":       resource.ETag,
		"properties": resource.Properties,
		"systemData": azureDataManagerForAgriSystemData(resource.CreatedAt, resource.UpdatedAt),
	}
}

func azureDataManagerForAgriPrivateLinkResourcesFixtures(subscriptionID, resourceGroup, farmBeatName string) []map[string]any {
	baseID := azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, farmBeatName)
	return []map[string]any{
		{
			"id":   baseID + "/privateLinkResources/farmbeats",
			"name": "farmbeats",
			"type": "Microsoft.AgFoodPlatform/farmBeats/privateLinkResources",
			"properties": map[string]any{
				"groupId":                           "farmbeats",
				"requiredMembers":                   []any{"farmbeats"},
				"requiredZoneNames":                 []any{"privatelink.farmbeats.azure.net"},
				"displayName":                       "FarmBeats",
				"shareablePrivateLinkResourceTypes": []any{},
			},
		},
	}
}

func azureDataManagerForAgriExtensionDefinitionsFixtures(subscriptionID, resourceGroup, farmBeatName string) []map[string]any {
	baseID := azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, farmBeatName)
	return []map[string]any{
		{
			"id":   baseID + "/farmBeatsExtensionDefinitions/weather-insights",
			"name": "weather-insights",
			"type": "Microsoft.AgFoodPlatform/farmBeats/farmBeatsExtensionDefinitions",
			"properties": map[string]any{
				"displayName":       "Weather Insights",
				"description":       "Provides deterministic weather insights fixtures",
				"publisherId":       "stackyard",
				"extensionType":     "Insights",
				"supportedVersions": []any{"1.0.0"},
			},
		},
		{
			"id":   baseID + "/farmBeatsExtensionDefinitions/imagery-sync",
			"name": "imagery-sync",
			"type": "Microsoft.AgFoodPlatform/farmBeats/farmBeatsExtensionDefinitions",
			"properties": map[string]any{
				"displayName":       "Imagery Sync",
				"description":       "Provides imagery synchronization fixtures",
				"publisherId":       "stackyard",
				"extensionType":     "Data",
				"supportedVersions": []any{"1.0.0"},
			},
		},
	}
}

func azureDataManagerForAgriSolutionDefinitionsFixtures(subscriptionID, resourceGroup, farmBeatName string) []map[string]any {
	baseID := azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, farmBeatName)
	return []map[string]any{
		{
			"id":   baseID + "/farmBeatsSolutionDefinitions/crop-analytics",
			"name": "crop-analytics",
			"type": "Microsoft.AgFoodPlatform/farmBeats/farmBeatsSolutionDefinitions",
			"properties": map[string]any{
				"displayName":       "Crop Analytics",
				"description":       "Deterministic crop analytics solution",
				"publisherId":       "stackyard",
				"supportedVersions": []any{"1.0.0"},
			},
		},
		{
			"id":   baseID + "/farmBeatsSolutionDefinitions/irrigation-optimizer",
			"name": "irrigation-optimizer",
			"type": "Microsoft.AgFoodPlatform/farmBeats/farmBeatsSolutionDefinitions",
			"properties": map[string]any{
				"displayName":       "Irrigation Optimizer",
				"description":       "Deterministic irrigation optimization solution",
				"publisherId":       "stackyard",
				"supportedVersions": []any{"1.0.0"},
			},
		},
	}
}

func azureDataManagerForAgriOperationResultFixture(record *azureDataManagerForAgriOperationResult) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":         "/subscriptions/" + record.SubscriptionID + "/providers/Microsoft.AgFoodPlatform/locations/" + record.Location + "/operationResults/" + record.ID,
		"name":       record.ID,
		"status":     record.Status,
		"startTime":  record.StartTime.Format(time.RFC3339),
		"endTime":    record.EndTime.Format(time.RFC3339),
		"properties": map[string]any{"action": record.Action},
	}
}

func azureDataManagerForAgriOperationsFixture() []map[string]any {
	return []map[string]any{
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/read", "Read Data Manager for Agriculture resource"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/write", "Create or update Data Manager for Agriculture resource"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/delete", "Delete Data Manager for Agriculture resource"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/dataConnectors/read", "Read data connectors"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/dataConnectors/write", "Create or update data connectors"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/dataConnectors/delete", "Delete data connectors"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/extensions/read", "Read installed extensions"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/extensions/write", "Create or update installed extensions"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/extensions/delete", "Delete installed extensions"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/privateEndpointConnections/read", "Read private endpoint connections"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/privateEndpointConnections/write", "Create or update private endpoint connections"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/privateEndpointConnections/delete", "Delete private endpoint connections"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/privateLinkResources/read", "Read private link resources"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/solutions/read", "Read installed solutions"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/solutions/write", "Create or update installed solutions"),
		azureDataManagerForAgriOperation("Microsoft.AgFoodPlatform/farmBeats/solutions/delete", "Delete installed solutions"),
	}
}

func azureDataManagerForAgriOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":  "Microsoft AgFoodPlatform",
			"resource":  "farmBeats",
			"operation": description,
		},
		"isDataAction": false,
	}
}

func azureDataManagerForAgriNormalizeFarmBeatProperties(name, location string, createdAt, now time.Time, raw map[string]any) map[string]any {
	return azureDataManagerForAgriMergeObjects(map[string]any{
		"provisioningState":   "Succeeded",
		"instanceUri":         "https://" + name + ".farmbeats.azure.net",
		"publicNetworkAccess": "Enabled",
		"createdAt":           createdAt.Format(time.RFC3339),
		"lastModifiedAt":      now.Format(time.RFC3339),
		"sensorIntegration": map[string]any{
			"enabled":           false,
			"provisioningState": "Succeeded",
		},
		"privateEndpointConnections": []any{},
		"regionalEndpoint":           "https://" + name + "." + strings.ToLower(location) + ".farmbeats.azure.net",
	}, raw)
}

func azureDataManagerForAgriNormalizeChildProperties(collection, name string, raw map[string]any) map[string]any {
	base := map[string]any{
		"provisioningState": "Succeeded",
	}
	switch collection {
	case "dataConnectors":
		base["connectorType"] = "OAuth"
		base["credentials"] = map[string]any{
			"kind":         "OAuth",
			"clientId":     "stackyard-client",
			"clientSecret": "stackyard-secret",
			"apiKey":       "stackyard-api-key",
		}
		base["connectionState"] = "Connected"
	case "extensions":
		base["displayName"] = strings.ReplaceAll(strings.Title(strings.ReplaceAll(name, "-", " ")), " ", " ")
		base["version"] = "1.0.0"
		base["status"] = "Installed"
	case "privateEndpointConnections":
		base["provisioningState"] = "Succeeded"
		base["privateEndpoint"] = map[string]any{
			"id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-network/providers/Microsoft.Network/privateEndpoints/" + name,
		}
		base["privateLinkServiceConnectionState"] = map[string]any{
			"status":          "Approved",
			"description":     "Approved by Stackyard",
			"actionsRequired": "None",
		}
	case "solutions":
		base["displayName"] = strings.ReplaceAll(strings.Title(strings.ReplaceAll(name, "-", " ")), " ", " ")
		base["version"] = "1.0.0"
		base["status"] = "Installed"
	}
	return azureDataManagerForAgriMergeObjects(base, raw)
}

func azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + name
}

func azureDataManagerForAgriFarmBeatKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, name))
}

func azureDataManagerForAgriChildID(subscriptionID, resourceGroup, farmBeatName, collection, name string) string {
	return azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, farmBeatName) + "/" + collection + "/" + name
}

func azureDataManagerForAgriChildKey(subscriptionID, resourceGroup, farmBeatName, collection, name string) string {
	return strings.ToLower(azureDataManagerForAgriChildID(subscriptionID, resourceGroup, farmBeatName, collection, name))
}

func azureDataManagerForAgriChildParentPrefix(subscriptionID, resourceGroup, farmBeatName string) string {
	return azureDataManagerForAgriFarmBeatID(subscriptionID, resourceGroup, farmBeatName) + "/"
}

func azureDataManagerForAgriOperationResultKey(subscriptionID, location, id string) string {
	return strings.ToLower(subscriptionID + "|" + location + "|" + id)
}

func azureDataManagerForAgriNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "NotFound",
			"message": fmt.Sprintf("%s was not found for path %s", name, path),
		},
	}
}

func azureDataManagerForAgriReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	defer r.Body.Close()
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "invalid JSON request body")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureDataManagerForAgriString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func azureDataManagerForAgriObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if out, ok := value.(map[string]any); ok {
		return out
	}
	return map[string]any{}
}

func azureDataManagerForAgriStringMap(value any) map[string]string {
	if value == nil {
		return nil
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	result := make(map[string]string, len(raw))
	for key, val := range raw {
		result[key] = azureDataManagerForAgriString(val)
	}
	return result
}

func azureDataManagerForAgriStringMapAny(value map[string]string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for key, val := range value {
		result[key] = val
	}
	return result
}

func azureDataManagerForAgriMergeObjects(base map[string]any, override map[string]any) map[string]any {
	if len(base) == 0 && len(override) == 0 {
		return map[string]any{}
	}
	merged := make(map[string]any, len(base)+len(override))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range override {
		if existing, ok := merged[key].(map[string]any); ok {
			if incoming, ok := value.(map[string]any); ok {
				merged[key] = azureDataManagerForAgriMergeObjects(existing, incoming)
				continue
			}
		}
		merged[key] = value
	}
	return merged
}

func azureDataManagerForAgriSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"createdAt":          createdAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
	}
}

func azureDataManagerForAgriETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureDataManagerForAgriStableToken(seed string) string {
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:8])
}

func azureDataManagerForAgriNilIfEmpty(value map[string]any) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
