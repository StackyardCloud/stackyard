package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureLoadTestingAPIVersion = "2024-12-01-preview"

type azureLoadTestingResource struct {
	SubscriptionID             string
	ResourceGroup              string
	Name                       string
	Location                   string
	Identity                   map[string]any
	Tags                       map[string]string
	Properties                 map[string]any
	MaxMonthlyVirtualUserHours int
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	ETag                       string
}

type azureLoadTestingChildResource struct {
	ParentResourceURI string
	Collection        string
	Name              string
	Properties        map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ETag              string
}

func (s *Server) handleAzureLoadTestingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsLoadTestingPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureLoadTestingAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.LoadTestService") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureLoadTestingOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, parentResourceURI, tail, ok := azureLoadTestingRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	if azureLoadTestingIsLoadTestsScope(parentResourceURI, subscriptionID, resourceGroup) {
		s.handleAzureLoadTestingLoadTestsRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	}

	if parentResourceURI != "" {
		s.handleAzureLoadTestingMappingsRoute(w, r, path, parentResourceURI, tail)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func azureIsLoadTestingPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.LoadTestService") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, _, tail, ok := azureLoadTestingRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "loadtests", "loadtestmappings", "loadtestprofilemappings":
		return true
	default:
		return false
	}
}

func azureLoadTestingAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureLoadTestingRouteContext(segments []string) (subscriptionID, resourceGroup, parentResourceURI string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.LoadTestService") {
			continue
		}
		if idx > 0 {
			parentResourceURI = "/" + strings.Join(segments[:idx], "/")
		}
		return subscriptionID, resourceGroup, parentResourceURI, segments[idx+2:], true
	}
	return "", "", "", nil, false
}

func azureLoadTestingIsLoadTestsScope(parentResourceURI, subscriptionID, resourceGroup string) bool {
	switch parentResourceURI {
	case "/subscriptions/" + subscriptionID:
		return true
	case "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup:
		return resourceGroup != ""
	default:
		return false
	}
}

func (s *Server) handleAzureLoadTestingLoadTestsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && strings.EqualFold(tail[0], "loadTests") && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureLoadTestingListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureLoadTestingListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) < 2 || !strings.EqualFold(tail[0], "loadTests") {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "load test name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureLoadTestingGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureLoadTestingGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureLoadTestingCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureLoadTestingPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureLoadTestingDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch {
	case len(tail) == 3 && strings.EqualFold(tail[2], "outboundNetworkDependenciesEndpoints"):
		if r.Method == http.MethodGet {
			s.handleAzureLoadTestingListOutboundDependencies(w, path, subscriptionID, resourceGroup, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case len(tail) == 4 && strings.EqualFold(tail[2], "limits") && strings.EqualFold(tail[3], "maxMonthlyVirtualUserHours"):
		if r.Method == http.MethodGet {
			s.handleAzureLoadTestingGetMaxMonthlyVUH(w, path, subscriptionID, resourceGroup, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case len(tail) == 5 && strings.EqualFold(tail[2], "limits") && strings.EqualFold(tail[3], "maxMonthlyVirtualUserHours") && strings.EqualFold(tail[4], "set"):
		if r.Method == http.MethodPost {
			s.handleAzureLoadTestingSetMaxMonthlyVUH(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureLoadTestingMappingsRoute(w http.ResponseWriter, r *http.Request, path, parentResourceURI string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}

	collection := azureLoadTestingCanonicalChildCollection(tail[0])
	if collection == "" {
		respondAzureImplemented(w, path)
		return
	}

	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			s.handleAzureLoadTestingChildList(w, parentResourceURI, collection)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "mapping resource name is required")
		return
	}
	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureLoadTestingChildGet(w, path, parentResourceURI, collection, name, false)
	case http.MethodHead:
		s.handleAzureLoadTestingChildGet(w, path, parentResourceURI, collection, name, true)
	case http.MethodPut:
		s.handleAzureLoadTestingChildCreateOrUpdate(w, r, path, parentResourceURI, collection, name)
	case http.MethodPatch:
		if collection != "loadTestMappings" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureLoadTestingChildPatch(w, r, path, parentResourceURI, collection, name)
	case http.MethodDelete:
		s.handleAzureLoadTestingChildDelete(w, path, parentResourceURI, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureLoadTestingCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureLoadTestingReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureLoadTestingString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureLoadTests[key]
	status := http.StatusOK
	createdAt := now
	vuh := 10000
	if resource == nil {
		resource = &azureLoadTestingResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
		vuh = resource.MaxMonthlyVirtualUserHours
	}

	resource.Location = location
	resource.Identity = azureLoadTestingObject(payload["identity"])
	resource.Tags = azureLoadTestingStringMap(payload["tags"])
	resource.Properties = azureLoadTestingNormalizeLoadTestProperties(name, subscriptionID, resourceGroup, azureLoadTestingObject(payload["properties"]), createdAt, now)
	resource.MaxMonthlyVirtualUserHours = vuh
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureLoadTestingETag(key, body, now)
	s.azureLoadTests[key] = resource

	respondJSON(w, status, azureLoadTestingResourceFixture(resource))
}

func (s *Server) handleAzureLoadTestingPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureLoadTestingReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureLoadTests[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["location"]; ok {
		location := azureLoadTestingString(payload["location"])
		if location == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		resource.Location = location
		updated = true
	}
	if _, ok := payload["identity"]; ok {
		resource.Identity = azureLoadTestingObject(payload["identity"])
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureLoadTestingStringMap(payload["tags"])
		updated = true
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureLoadTestingMergeObjects(resource.Properties, azureLoadTestingObject(properties))
		resource.Properties = azureLoadTestingNormalizeLoadTestProperties(name, subscriptionID, resourceGroup, resource.Properties, resource.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	resource.UpdatedAt = now
	resource.ETag = azureLoadTestingETag(key, body, now)
	respondJSON(w, http.StatusOK, azureLoadTestingResourceFixture(resource))
}

func (s *Server) handleAzureLoadTestingGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureLoadTests[azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingResourceFixture(resource))
}

func (s *Server) handleAzureLoadTestingDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureLoadTests[key]
	if exists {
		delete(s.azureLoadTests, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Location", "/azure"+azureLoadTestingResourceID(subscriptionID, resourceGroup, name)+"?api-version="+azureLoadTestingAPIVersion)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureLoadTestingListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, resource := range s.azureLoadTests {
		if resource.SubscriptionID != subscriptionID {
			continue
		}
		values = append(values, azureLoadTestingResourceFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureLoadTestingString(values[i]["id"]) < azureLoadTestingString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureLoadTestingListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, resource := range s.azureLoadTests {
		if resource.SubscriptionID != subscriptionID || !strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			continue
		}
		values = append(values, azureLoadTestingResourceFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureLoadTestingString(values[i]["id"]) < azureLoadTestingString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureLoadTestingListOutboundDependencies(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureLoadTests[azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}

	baseHost := fmt.Sprintf("%s.%s.loadtesting.azure.example", name, strings.ToLower(resource.Location))
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []any{
			map[string]any{
				"category": "Engine",
				"endpoints": []any{
					map[string]any{
						"domainName": baseHost,
						"ports":      []any{443},
					},
				},
			},
			map[string]any{
				"category": "Results",
				"endpoints": []any{
					map[string]any{
						"domainName": "storage." + baseHost,
						"ports":      []any{443},
					},
				},
			},
		},
	})
}

func (s *Server) handleAzureLoadTestingGetMaxMonthlyVUH(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureLoadTests[azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingMaxMonthlyVUHFixture(resource))
}

func (s *Server) handleAzureLoadTestingSetMaxMonthlyVUH(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureLoadTestingReadRequestBody(w, r, path)
	if !ok {
		return
	}

	value, ok := azureLoadTestingExtractLimitValue(payload)
	if !ok {
		respondAzureInvalidRequest(w, path, "limit value is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureLoadTests[azureLoadTestingResourceKey(subscriptionID, resourceGroup, name)]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}
	resource.MaxMonthlyVirtualUserHours = value
	resource.UpdatedAt = time.Now().UTC()
	resource.Properties = azureLoadTestingNormalizeLoadTestProperties(name, subscriptionID, resourceGroup, resource.Properties, resource.CreatedAt, resource.UpdatedAt)

	respondJSON(w, http.StatusOK, azureLoadTestingMaxMonthlyVUHFixture(resource))
}

func (s *Server) handleAzureLoadTestingChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, parentResourceURI, collection, name string) {
	body, payload, ok := azureLoadTestingReadRequestBody(w, r, path)
	if !ok {
		return
	}

	properties := azureLoadTestingObject(payload["properties"])
	if len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}
	if !azureLoadTestingValidChildName(name) {
		respondAzureInvalidRequest(w, path, "mapping resource name is invalid")
		return
	}
	if !azureLoadTestingValidateChildProperties(parentResourceURI, collection, properties) {
		respondAzureInvalidRequest(w, path, "properties contain invalid or missing required fields")
		return
	}

	key := azureLoadTestingChildKey(parentResourceURI, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureLoadTestingChildResources[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureLoadTestingChildResource{
			ParentResourceURI: parentResourceURI,
			Collection:        collection,
			Name:              name,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}

	record.Properties = azureLoadTestingNormalizeChildProperties(collection, properties, createdAt, now)
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureLoadTestingETag(key, body, now)
	s.azureLoadTestingChildResources[key] = record

	respondJSON(w, status, azureLoadTestingChildFixture(record))
}

func (s *Server) handleAzureLoadTestingChildPatch(w http.ResponseWriter, r *http.Request, path, parentResourceURI, collection, name string) {
	body, payload, ok := azureLoadTestingReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureLoadTestingChildKey(parentResourceURI, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureLoadTestingChildResources[key]
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	properties, ok := payload["properties"]
	if !ok {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}

	merged := azureLoadTestingMergeObjects(record.Properties, azureLoadTestingObject(properties))
	if !azureLoadTestingValidateChildProperties(parentResourceURI, collection, merged) {
		respondAzureInvalidRequest(w, path, "properties contain invalid or missing required fields")
		return
	}

	record.Properties = azureLoadTestingNormalizeChildProperties(collection, merged, record.CreatedAt, now)
	record.UpdatedAt = now
	record.ETag = azureLoadTestingETag(key, body, now)
	respondJSON(w, http.StatusOK, azureLoadTestingChildFixture(record))
}

func (s *Server) handleAzureLoadTestingChildGet(w http.ResponseWriter, path, parentResourceURI, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureLoadTestingChildResources[azureLoadTestingChildKey(parentResourceURI, collection, name)]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureLoadTestingNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureLoadTestingChildFixture(record))
}

func (s *Server) handleAzureLoadTestingChildDelete(w http.ResponseWriter, path, parentResourceURI, collection, name string) {
	key := azureLoadTestingChildKey(parentResourceURI, collection, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureLoadTestingChildResources[key]
	if exists {
		delete(s.azureLoadTestingChildResources, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureLoadTestingChildList(w http.ResponseWriter, parentResourceURI, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, record := range s.azureLoadTestingChildResources {
		if record.ParentResourceURI != parentResourceURI || record.Collection != collection {
			continue
		}
		values = append(values, azureLoadTestingChildFixture(record))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureLoadTestingString(values[i]["id"]) < azureLoadTestingString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func azureLoadTestingCanonicalChildCollection(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "loadtestmappings":
		return "loadTestMappings"
	case "loadtestprofilemappings":
		return "loadTestProfileMappings"
	default:
		return ""
	}
}

func azureLoadTestingResourceFixture(resource *azureLoadTestingResource) map[string]any {
	fixture := map[string]any{
		"id":         azureLoadTestingResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.LoadTestService/loadTests",
		"location":   resource.Location,
		"tags":       azureLoadTestingAnyMapFromStrings(resource.Tags),
		"systemData": azureLoadTestingSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureLoadTestingCloneObject(resource.Properties),
	}
	if len(resource.Identity) > 0 {
		fixture["identity"] = azureLoadTestingCloneObject(resource.Identity)
	}
	if resource.ETag != "" {
		fixture["etag"] = resource.ETag
	}
	return fixture
}

func azureLoadTestingChildFixture(record *azureLoadTestingChildResource) map[string]any {
	collectionType := "Microsoft.LoadTestService/" + record.Collection
	return map[string]any{
		"id":         azureLoadTestingChildID(record.ParentResourceURI, record.Collection, record.Name),
		"name":       record.Name,
		"type":       collectionType,
		"systemData": azureLoadTestingSystemData(record.CreatedAt, record.UpdatedAt),
		"properties": azureLoadTestingCloneObject(record.Properties),
		"etag":       record.ETag,
	}
}

func azureLoadTestingMaxMonthlyVUHFixture(resource *azureLoadTestingResource) map[string]any {
	return map[string]any{
		"id":   azureLoadTestingResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Name) + "/limits/maxMonthlyVirtualUserHours",
		"name": "maxMonthlyVirtualUserHours",
		"type": "Microsoft.LoadTestService/loadTests/limits/maxMonthlyVirtualUserHours",
		"properties": map[string]any{
			"value":             resource.MaxMonthlyVirtualUserHours,
			"unit":              "VirtualUserHours",
			"provisioningState": "Succeeded",
		},
	}
}

func azureLoadTestingOperationsFixture() []map[string]any {
	actions := []struct {
		Name    string
		Display string
	}{
		{"Microsoft.LoadTestService/loadTests/read", "Read load test resource"},
		{"Microsoft.LoadTestService/loadTests/write", "Create or update load test resource"},
		{"Microsoft.LoadTestService/loadTests/delete", "Delete load test resource"},
		{"Microsoft.LoadTestService/loadTests/outboundNetworkDependenciesEndpoints/read", "Read outbound dependencies"},
		{"Microsoft.LoadTestService/loadTests/limits/maxMonthlyVirtualUserHours/read", "Read max monthly virtual user hours limit"},
		{"Microsoft.LoadTestService/loadTests/limits/maxMonthlyVirtualUserHours/set/action", "Set max monthly virtual user hours limit"},
		{"Microsoft.LoadTestService/loadTestMappings/read", "Read load test mappings"},
		{"Microsoft.LoadTestService/loadTestMappings/write", "Create or update load test mappings"},
		{"Microsoft.LoadTestService/loadTestMappings/delete", "Delete load test mappings"},
		{"Microsoft.LoadTestService/loadTestProfileMappings/read", "Read load test profile mappings"},
		{"Microsoft.LoadTestService/loadTestProfileMappings/write", "Create or update load test profile mappings"},
		{"Microsoft.LoadTestService/loadTestProfileMappings/delete", "Delete load test profile mappings"},
	}
	values := make([]map[string]any, 0, len(actions))
	for _, action := range actions {
		values = append(values, map[string]any{
			"name": action.Name,
			"display": map[string]any{
				"provider":  "Microsoft LoadTestService",
				"resource":  "Load Testing",
				"operation": action.Display,
			},
		})
	}
	return values
}

func azureLoadTestingReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureLoadTestingNormalizeLoadTestProperties(name, subscriptionID, resourceGroup string, properties map[string]any, createdAt, updatedAt time.Time) map[string]any {
	normalized := azureLoadTestingCloneObject(properties)
	token := subscriptionID
	if len(token) > 8 {
		token = token[:8]
	}
	normalized["dataPlaneURI"] = fmt.Sprintf("https://%s-%s.%s.loadtesting.azure.example", resourceGroup, name, token)
	normalized["provisioningState"] = "Succeeded"
	if _, ok := normalized["description"]; !ok {
		normalized["description"] = "Stackyard load test resource"
	}
	if _, ok := normalized["encryption"]; !ok {
		normalized["encryption"] = map[string]any{"type": "SystemAssigned"}
	}
	if _, ok := normalized["createdDateTime"]; !ok {
		normalized["createdDateTime"] = createdAt.Format(time.RFC3339)
	}
	normalized["lastModifiedDateTime"] = updatedAt.Format(time.RFC3339)
	return normalized
}

func azureLoadTestingNormalizeChildProperties(collection string, properties map[string]any, createdAt, updatedAt time.Time) map[string]any {
	normalized := azureLoadTestingCloneObject(properties)
	normalized["provisioningState"] = "Succeeded"
	if _, ok := normalized["createdDateTime"]; !ok {
		normalized["createdDateTime"] = createdAt.Format(time.RFC3339)
	}
	normalized["lastModifiedDateTime"] = updatedAt.Format(time.RFC3339)
	if collection == "loadTestMappings" {
		if _, ok := normalized["testId"]; !ok {
			normalized["testId"] = "test-default"
		}
	}
	if collection == "loadTestProfileMappings" {
		if _, ok := normalized["testProfileId"]; !ok {
			normalized["testProfileId"] = "profile-default"
		}
	}
	return normalized
}

func azureLoadTestingValidateChildProperties(parentResourceURI, collection string, properties map[string]any) bool {
	sourceResourceID := azureLoadTestingString(properties["sourceResourceId"])
	if sourceResourceID == "" || !strings.EqualFold(sourceResourceID, parentResourceURI) {
		return false
	}
	if azureLoadTestingString(properties["azureLoadTestingResourceId"]) == "" {
		return false
	}
	switch collection {
	case "loadTestMappings":
		return azureLoadTestingString(properties["testId"]) != ""
	case "loadTestProfileMappings":
		return azureLoadTestingString(properties["testProfileId"]) != ""
	default:
		return false
	}
}

func azureLoadTestingExtractLimitValue(payload map[string]any) (int, bool) {
	for _, candidate := range []any{
		payload["value"],
		payload["maxMonthlyVirtualUserHours"],
		azureLoadTestingObject(payload["properties"])["value"],
		azureLoadTestingObject(payload["properties"])["maxMonthlyVirtualUserHours"],
	} {
		if candidate == nil {
			continue
		}
		switch v := candidate.(type) {
		case float64:
			if v >= 0 {
				return int(v), true
			}
		case int:
			if v >= 0 {
				return v, true
			}
		case string:
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err == nil && n >= 0 {
				return n, true
			}
		}
	}
	return 0, false
}

func azureLoadTestingValidChildName(name string) bool {
	if name == "" {
		return false
	}
	for idx, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' && idx > 0 && idx < len(name)-1:
		default:
			return false
		}
	}
	return true
}

func azureLoadTestingResourceID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.LoadTestService/loadTests/%s", subscriptionID, resourceGroup, name)
}

func azureLoadTestingChildID(parentResourceURI, collection, name string) string {
	return parentResourceURI + "/providers/Microsoft.LoadTestService/" + collection + "/" + name
}

func azureLoadTestingResourceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "::" + strings.ToLower(resourceGroup) + "::" + strings.ToLower(name)
}

func azureLoadTestingChildKey(parentResourceURI, collection, name string) string {
	return strings.ToLower(parentResourceURI) + "::" + strings.ToLower(collection) + "::" + strings.ToLower(name)
}

func azureLoadTestingETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureLoadTestingSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureLoadTestingNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("Resource %q was not found for path %s.", name, path),
		},
	}
}

func azureLoadTestingString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func azureLoadTestingObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return azureLoadTestingCloneObject(typed)
}

func azureLoadTestingStringMap(value any) map[string]string {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(typed))
	for key, raw := range typed {
		if str := azureLoadTestingString(raw); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureLoadTestingAnyMapFromStrings(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, raw := range value {
		out[key] = raw
	}
	return out
}

func azureLoadTestingCloneObject(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, raw := range value {
		switch typed := raw.(type) {
		case map[string]any:
			out[key] = azureLoadTestingCloneObject(typed)
		case []any:
			out[key] = azureLoadTestingCloneSlice(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func azureLoadTestingCloneSlice(value []any) []any {
	out := make([]any, 0, len(value))
	for _, raw := range value {
		switch typed := raw.(type) {
		case map[string]any:
			out = append(out, azureLoadTestingCloneObject(typed))
		case []any:
			out = append(out, azureLoadTestingCloneSlice(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func azureLoadTestingMergeObjects(base, overlay map[string]any) map[string]any {
	merged := azureLoadTestingCloneObject(base)
	for key, value := range overlay {
		if nestedOverlay, ok := value.(map[string]any); ok {
			if nestedBase, ok := merged[key].(map[string]any); ok {
				merged[key] = azureLoadTestingMergeObjects(nestedBase, nestedOverlay)
				continue
			}
			merged[key] = azureLoadTestingCloneObject(nestedOverlay)
			continue
		}
		if nestedList, ok := value.([]any); ok {
			merged[key] = azureLoadTestingCloneSlice(nestedList)
			continue
		}
		merged[key] = value
	}
	return merged
}
