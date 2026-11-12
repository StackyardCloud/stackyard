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

type azureContainerAppsResource struct {
	SubscriptionID string
	ResourceGroup  string
	Collection     string
	Name           string
	Location       string
	Kind           string
	Identity       map[string]any
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureContainerAppsChildResource struct {
	SubscriptionID   string
	ResourceGroup    string
	ParentCollection string
	ParentName       string
	Collection       string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ETag             string
}

var azureContainerAppsTopLevelCollections = map[string]string{
	"managedenvironments":       "managedEnvironments",
	"containerapps":             "containerApps",
	"jobs":                      "jobs",
	"daprcomponents":            "daprComponents",
	"certificates":              "certificates",
	"managedcertificates":       "managedCertificates",
	"connectedenvironments":     "connectedEnvironments",
	"javacomponents":            "javaComponents",
	"logicapps":                 "logicApps",
	"maintenanceconfigurations": "maintenanceConfigurations",
	"namespaces":                "namespaces",
}

var azureContainerAppsChildCollections = map[string]string{
	"authconfigs":                "authConfigs",
	"diagnostics":                "diagnostics",
	"sessionpools":               "sessionPools",
	"sourcecontrols":             "sourceControls",
	"storages":                   "storages",
	"privateendpointconnections": "privateEndpointConnections",
	"privatelinkresources":       "privateLinkResources",
}

func (s *Server) handleAzureContainerAppsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsContainerAppsPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureContainerAppsAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.App") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	if subscriptionID, location, tail, ok := azureContainerAppsLocationRouteContext(segments); ok {
		s.handleAzureContainerAppsLocationRoute(w, r, path, subscriptionID, location, tail)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureContainerAppsRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		s.handleAzureContainerAppsSubscriptionRoute(w, r, path, subscriptionID, tail)
		return true
	}

	s.handleAzureContainerAppsResourceGroupRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsContainerAppsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.App") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	if _, _, tail, ok := azureContainerAppsRouteContext(segments); ok && len(tail) > 0 {
		head := strings.ToLower(strings.TrimSpace(tail[0]))
		if _, found := azureContainerAppsTopLevelCollections[head]; found {
			return true
		}
		switch head {
		case "usages", "managedenvironmentusages", "getcustomdomainverificationid":
			return true
		}
	}
	if _, _, tail, ok := azureContainerAppsLocationRouteContext(segments); ok && len(tail) > 0 {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "availablemanagedenvironmentsworkloadprofiletypes", "billingmeters", "usages", "managedenvironmentusages":
			return true
		}
	}
	return false
}

func azureContainerAppsRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.App") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureContainerAppsLocationRouteContext(segments []string) (subscriptionID, location string, tail []string, ok bool) {
	if len(segments) < 6 || !strings.EqualFold(segments[0], "subscriptions") || !strings.EqualFold(segments[2], "locations") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	location = strings.TrimSpace(segments[3])
	if subscriptionID == "" || location == "" {
		return "", "", nil, false
	}
	for idx := 4; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.App") {
			continue
		}
		return subscriptionID, location, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureContainerAppsAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureContainerAppsSubscriptionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}
	head := strings.ToLower(strings.TrimSpace(tail[0]))
	switch head {
	case "getcustomdomainverificationid":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			respondJSON(w, http.StatusOK, map[string]any{
				"customDomainVerificationId": "stackyard-" + azureContainerAppsStableToken(subscriptionID),
			})
			return
		}
	case "usages":
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsUsageFixture(subscriptionID, "")})
			return
		}
	case "managedenvironmentusages":
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsManagedEnvironmentUsageFixture(subscriptionID, "")})
			return
		}
	default:
		if collection, ok := azureContainerAppsTopLevelCollections[head]; ok && len(tail) == 1 && r.Method == http.MethodGet {
			s.handleAzureContainerAppsListBySubscription(w, subscriptionID, collection)
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureContainerAppsLocationRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) == 0 || r.Method != http.MethodGet {
		respondAzureImplemented(w, path)
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "availablemanagedenvironmentsworkloadprofiletypes":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsAvailableWorkloadProfilesFixture(subscriptionID, location)})
	case "billingmeters":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsBillingMetersFixture(subscriptionID, location)})
	case "usages":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsUsageFixture(subscriptionID, location)})
	case "managedenvironmentusages":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerAppsManagedEnvironmentUsageFixture(subscriptionID, location)})
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureContainerAppsResourceGroupRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}

	collection, found := azureContainerAppsTopLevelCollections[strings.ToLower(strings.TrimSpace(tail[0]))]
	if !found {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureContainerAppsListByResourceGroup(w, subscriptionID, resourceGroup, collection)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, strings.TrimSuffix(collection, "s")+" name is required")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureContainerAppsGet(w, path, subscriptionID, resourceGroup, collection, name, false)
		case http.MethodHead:
			s.handleAzureContainerAppsGet(w, path, subscriptionID, resourceGroup, collection, name, true)
		case http.MethodPut:
			s.handleAzureContainerAppsCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, collection, name)
		case http.MethodPatch:
			s.handleAzureContainerAppsPatch(w, r, path, subscriptionID, resourceGroup, collection, name)
		case http.MethodDelete:
			s.handleAzureContainerAppsDelete(w, path, subscriptionID, resourceGroup, collection, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "revisions":
		if strings.EqualFold(collection, "containerApps") {
			s.handleAzureContainerAppsRevisionsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[3:])
			return
		}
	case "executions":
		if strings.EqualFold(collection, "jobs") {
			s.handleAzureContainerAppsJobExecutionsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[3:])
			return
		}
	default:
		if childCollection, ok := azureContainerAppsChildCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok {
			s.handleAzureContainerAppsChildCollectionRoute(w, r, path, subscriptionID, resourceGroup, collection, name, childCollection, tail[3:])
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureContainerAppsChildCollectionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, parentCollection, parentName, collection string, tail []string) {
	if !s.azureContainerAppsParentExists(subscriptionID, resourceGroup, parentCollection, parentName) {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, parentName))
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureContainerAppsChildList(w, subscriptionID, resourceGroup, parentCollection, parentName, collection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "child resource name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureContainerAppsChildGet(w, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	case http.MethodPut:
		s.handleAzureContainerAppsChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	case http.MethodPatch:
		s.handleAzureContainerAppsChildPatch(w, r, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	case http.MethodDelete:
		s.handleAzureContainerAppsChildDelete(w, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureContainerAppsRevisionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, appName string, tail []string) {
	if !s.azureContainerAppsParentExists(subscriptionID, resourceGroup, "containerApps", appName) {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, appName))
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": s.azureContainerAppsRevisionFixtures(subscriptionID, resourceGroup, appName)})
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	revisionName := strings.TrimSpace(tail[0])
	if revisionName == "" {
		respondAzureInvalidRequest(w, path, "revision name is required")
		return
	}
	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, s.azureContainerAppsRevisionFixture(subscriptionID, resourceGroup, appName, revisionName))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) >= 2 && strings.EqualFold(tail[1], "replicas") {
		if len(tail) == 2 && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": s.azureContainerAppsReplicaFixtures(subscriptionID, resourceGroup, appName, revisionName)})
			return
		}
		if len(tail) == 3 && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, s.azureContainerAppsReplicaFixture(subscriptionID, resourceGroup, appName, revisionName, strings.TrimSpace(tail[2])))
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureContainerAppsJobExecutionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, jobName string, tail []string) {
	if !s.azureContainerAppsParentExists(subscriptionID, resourceGroup, "jobs", jobName) {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, jobName))
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": s.azureContainerAppsJobExecutionFixtures(subscriptionID, resourceGroup, jobName)})
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "execution name is required")
		return
	}
	if r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, s.azureContainerAppsJobExecutionFixture(subscriptionID, resourceGroup, jobName, name))
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleAzureContainerAppsCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	body, payload, ok := azureContainerAppsReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := azureContainerAppsString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureContainerAppsObject(payload["properties"])
	if strings.EqualFold(collection, "containerApps") && azureContainerAppsString(properties["environmentId"]) == "" {
		respondAzureInvalidRequest(w, path, "properties.environmentId is required")
		return
	}
	if strings.EqualFold(collection, "jobs") && azureContainerAppsString(properties["environmentId"]) == "" {
		respondAzureInvalidRequest(w, path, "properties.environmentId is required")
		return
	}

	now := time.Now().UTC()
	key := azureContainerAppsResourceKey(subscriptionID, resourceGroup, collection, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureContainerAppsResources[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureContainerAppsResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Collection:     collection,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}
	resource.Location = location
	resource.Kind = azureContainerAppsString(payload["kind"])
	resource.Identity = azureContainerAppsObject(payload["identity"])
	resource.Tags = azureContainerAppsStringMap(payload["tags"])
	resource.Properties = azureContainerAppsNormalizeProperties(collection, name, properties, createdAt, now)
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureContainerAppsETag(key, body, now)
	s.azureContainerAppsResources[key] = resource

	respondJSON(w, status, azureContainerAppsResourceFixture(resource))
}

func (s *Server) handleAzureContainerAppsPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	body, payload, ok := azureContainerAppsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureContainerAppsResourceKey(subscriptionID, resourceGroup, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	resource := s.azureContainerAppsResources[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["location"]; ok {
		location := azureContainerAppsString(payload["location"])
		if location == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		resource.Location = location
		updated = true
	}
	if _, ok := payload["kind"]; ok {
		resource.Kind = azureContainerAppsString(payload["kind"])
		updated = true
	}
	if _, ok := payload["identity"]; ok {
		resource.Identity = azureContainerAppsObject(payload["identity"])
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureContainerAppsStringMap(payload["tags"])
		updated = true
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureContainerAppsMergeObjects(resource.Properties, azureContainerAppsObject(properties))
		resource.Properties = azureContainerAppsNormalizeProperties(collection, name, resource.Properties, resource.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	resource.UpdatedAt = now
	resource.ETag = azureContainerAppsETag(key, body, now)
	respondJSON(w, http.StatusOK, azureContainerAppsResourceFixture(resource))
}

func (s *Server) handleAzureContainerAppsGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureContainerAppsResources[azureContainerAppsResourceKey(subscriptionID, resourceGroup, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureContainerAppsResourceFixture(resource))
}

func (s *Server) handleAzureContainerAppsDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string) {
	key := azureContainerAppsResourceKey(subscriptionID, resourceGroup, collection, name)
	s.providerStorageMu.Lock()
	_, exists := s.azureContainerAppsResources[key]
	if exists {
		delete(s.azureContainerAppsResources, key)
		for childKey, child := range s.azureContainerAppsChildResources {
			if strings.EqualFold(child.SubscriptionID, subscriptionID) &&
				strings.EqualFold(child.ResourceGroup, resourceGroup) &&
				strings.EqualFold(child.ParentCollection, collection) &&
				strings.EqualFold(child.ParentName, name) {
				delete(s.azureContainerAppsChildResources, childKey)
			}
		}
	}
	s.providerStorageMu.Unlock()
	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureContainerAppsListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup, collection string) {
	s.providerStorageMu.Lock()
	items := make([]*azureContainerAppsResource, 0)
	for _, resource := range s.azureContainerAppsResources {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) &&
			strings.EqualFold(resource.ResourceGroup, resourceGroup) &&
			strings.EqualFold(resource.Collection, collection) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureContainerAppsResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureContainerAppsListBySubscription(w http.ResponseWriter, subscriptionID, collection string) {
	s.providerStorageMu.Lock()
	items := make([]*azureContainerAppsResource, 0)
	for _, resource := range s.azureContainerAppsResources {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) &&
			strings.EqualFold(resource.Collection, collection) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		if strings.EqualFold(items[i].ResourceGroup, items[j].ResourceGroup) {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return strings.ToLower(items[i].ResourceGroup) < strings.ToLower(items[j].ResourceGroup)
	})
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureContainerAppsResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureContainerAppsChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name string) {
	body, payload, ok := azureContainerAppsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureContainerAppsChildKey(subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureContainerAppsChildResources[key]
	status := http.StatusOK
	createdAt := now
	location := azureContainerAppsString(payload["location"])
	if location == "" {
		if parent := s.azureContainerAppsResources[azureContainerAppsResourceKey(subscriptionID, resourceGroup, parentCollection, parentName)]; parent != nil {
			location = parent.Location
		}
	}
	if current == nil {
		current = &azureContainerAppsChildResource{
			SubscriptionID:   subscriptionID,
			ResourceGroup:    resourceGroup,
			ParentCollection: parentCollection,
			ParentName:       parentName,
			Collection:       collection,
			Name:             name,
		}
		status = http.StatusCreated
	} else {
		createdAt = current.CreatedAt
	}
	current.Location = location
	current.Tags = azureContainerAppsStringMap(payload["tags"])
	current.Properties = azureContainerAppsNormalizeChildProperties(collection, name, azureContainerAppsObject(payload["properties"]))
	current.CreatedAt = createdAt
	current.UpdatedAt = now
	current.ETag = azureContainerAppsETag(key, body, now)
	s.azureContainerAppsChildResources[key] = current
	respondJSON(w, status, azureContainerAppsChildFixture(current))
}

func (s *Server) handleAzureContainerAppsChildPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name string) {
	body, payload, ok := azureContainerAppsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureContainerAppsChildKey(subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureContainerAppsChildResources[key]
	if current == nil {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	if _, ok := payload["tags"]; ok {
		current.Tags = azureContainerAppsStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		current.Properties = azureContainerAppsMergeObjects(current.Properties, azureContainerAppsObject(payload["properties"]))
		current.Properties = azureContainerAppsNormalizeChildProperties(collection, name, current.Properties)
	}
	if _, ok := payload["location"]; ok {
		current.Location = azureContainerAppsString(payload["location"])
	}
	current.UpdatedAt = now
	current.ETag = azureContainerAppsETag(key, body, now)
	respondJSON(w, http.StatusOK, azureContainerAppsChildFixture(current))
}

func (s *Server) handleAzureContainerAppsChildGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name string) {
	s.providerStorageMu.Lock()
	current := s.azureContainerAppsChildResources[azureContainerAppsChildKey(subscriptionID, resourceGroup, parentCollection, parentName, collection, name)]
	s.providerStorageMu.Unlock()
	if current == nil {
		respondJSON(w, http.StatusNotFound, azureContainerAppsNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureContainerAppsChildFixture(current))
}

func (s *Server) handleAzureContainerAppsChildDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, parentCollection, parentName, collection, name string) {
	key := azureContainerAppsChildKey(subscriptionID, resourceGroup, parentCollection, parentName, collection, name)
	s.providerStorageMu.Lock()
	_, exists := s.azureContainerAppsChildResources[key]
	if exists {
		delete(s.azureContainerAppsChildResources, key)
	}
	s.providerStorageMu.Unlock()
	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureContainerAppsChildList(w http.ResponseWriter, subscriptionID, resourceGroup, parentCollection, parentName, collection string) {
	s.providerStorageMu.Lock()
	items := make([]*azureContainerAppsChildResource, 0)
	for _, item := range s.azureContainerAppsChildResources {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.ParentCollection, parentCollection) &&
			strings.EqualFold(item.ParentName, parentName) &&
			strings.EqualFold(item.Collection, collection) {
			items = append(items, item)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name) })
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureContainerAppsChildFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) azureContainerAppsParentExists(subscriptionID, resourceGroup, collection, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	_, ok := s.azureContainerAppsResources[azureContainerAppsResourceKey(subscriptionID, resourceGroup, collection, name)]
	return ok
}

func (s *Server) azureContainerAppsRevisionFixtures(subscriptionID, resourceGroup, appName string) []any {
	s.providerStorageMu.Lock()
	app := s.azureContainerAppsResources[azureContainerAppsResourceKey(subscriptionID, resourceGroup, "containerApps", appName)]
	s.providerStorageMu.Unlock()
	if app == nil {
		return []any{}
	}
	name := azureContainerAppsString(app.Properties["latestRevisionName"])
	if name == "" {
		name = appName + "--rev1"
	}
	return []any{s.azureContainerAppsRevisionFixture(subscriptionID, resourceGroup, appName, name)}
}

func (s *Server) azureContainerAppsRevisionFixture(subscriptionID, resourceGroup, appName, revisionName string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/revisions/%s", azureContainerAppsID(subscriptionID, resourceGroup, "containerApps", appName), revisionName),
		"name": revisionName,
		"type": "Microsoft.App/containerApps/revisions",
		"properties": map[string]any{
			"active":            true,
			"createdTime":       time.Now().UTC().Format(time.RFC3339),
			"trafficWeight":     100,
			"healthState":       "Healthy",
			"provisioningState": "Succeeded",
			"replicas":          1,
		},
	}
}

func (s *Server) azureContainerAppsReplicaFixtures(subscriptionID, resourceGroup, appName, revisionName string) []any {
	replicaName := revisionName + "-replica-1"
	return []any{s.azureContainerAppsReplicaFixture(subscriptionID, resourceGroup, appName, revisionName, replicaName)}
}

func (s *Server) azureContainerAppsReplicaFixture(subscriptionID, resourceGroup, appName, revisionName, replicaName string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/revisions/%s/replicas/%s", azureContainerAppsID(subscriptionID, resourceGroup, "containerApps", appName), revisionName, replicaName),
		"name": replicaName,
		"type": "Microsoft.App/containerApps/revisions/replicas",
		"properties": map[string]any{
			"runningState": "Running",
			"createdTime":  time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func (s *Server) azureContainerAppsJobExecutionFixtures(subscriptionID, resourceGroup, jobName string) []any {
	name := jobName + "-execution-1"
	return []any{s.azureContainerAppsJobExecutionFixture(subscriptionID, resourceGroup, jobName, name)}
}

func (s *Server) azureContainerAppsJobExecutionFixture(subscriptionID, resourceGroup, jobName, executionName string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("%s/executions/%s", azureContainerAppsID(subscriptionID, resourceGroup, "jobs", jobName), executionName),
		"name": executionName,
		"type": "Microsoft.App/jobs/executions",
		"properties": map[string]any{
			"status":                 "Succeeded",
			"startTime":              time.Now().UTC().Format(time.RFC3339),
			"replicaCompletionCount": 1,
		},
	}
}

func azureContainerAppsOperationsFixture() []map[string]any {
	return []map[string]any{
		azureContainerAppsOperation("Microsoft.App/managedEnvironments/read", "Read Managed Environments"),
		azureContainerAppsOperation("Microsoft.App/managedEnvironments/write", "Write Managed Environments"),
		azureContainerAppsOperation("Microsoft.App/containerApps/read", "Read Container Apps"),
		azureContainerAppsOperation("Microsoft.App/containerApps/write", "Write Container Apps"),
		azureContainerAppsOperation("Microsoft.App/jobs/read", "Read Jobs"),
		azureContainerAppsOperation("Microsoft.App/jobs/write", "Write Jobs"),
	}
}

func azureContainerAppsOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":    "Microsoft App",
			"resource":    "containerapps",
			"operation":   strings.TrimPrefix(strings.TrimPrefix(name, "Microsoft.App/"), "containerApps/"),
			"description": description,
		},
		"origin":       "user",
		"isDataAction": false,
	}
}

func azureContainerAppsAvailableWorkloadProfilesFixture(subscriptionID, location string) []any {
	return []any{
		map[string]any{"name": "Consumption", "location": location, "displayName": "Consumption"},
		map[string]any{"name": "Dedicated-D4", "location": location, "displayName": "Dedicated D4"},
	}
}

func azureContainerAppsBillingMetersFixture(subscriptionID, location string) []any {
	return []any{
		map[string]any{"meterName": "ContainerAppConsumptionCPU", "location": location, "unit": "core-seconds"},
		map[string]any{"meterName": "ContainerAppConsumptionMemory", "location": location, "unit": "GiB-seconds"},
	}
}

func azureContainerAppsUsageFixture(subscriptionID, location string) []any {
	return []any{
		map[string]any{
			"name":         map[string]any{"value": "ManagedEnvironments", "localizedValue": "Managed Environments"},
			"unit":         "Count",
			"currentValue": 1,
			"limit":        100,
			"location":     location,
		},
	}
}

func azureContainerAppsManagedEnvironmentUsageFixture(subscriptionID, location string) []any {
	return []any{
		map[string]any{
			"name":         map[string]any{"value": "WorkloadProfiles", "localizedValue": "Workload Profiles"},
			"unit":         "Count",
			"currentValue": 1,
			"limit":        10,
			"location":     location,
		},
	}
}

func azureContainerAppsResourceFixture(resource *azureContainerAppsResource) map[string]any {
	properties := azureContainerAppsCloneObject(resource.Properties)
	properties["provisioningState"] = azureContainerAppsStringOr(properties["provisioningState"], "Succeeded")
	switch resource.Collection {
	case "managedEnvironments":
		properties["defaultDomain"] = azureContainerAppsStringOr(properties["defaultDomain"], strings.ToLower(resource.Name)+".eastus.azurecontainerapps.io")
		properties["staticIp"] = azureContainerAppsStringOr(properties["staticIp"], "10.0.0.4")
	case "containerApps":
		latest := azureContainerAppsStringOr(properties["latestRevisionName"], resource.Name+"--rev1")
		properties["latestRevisionName"] = latest
		properties["latestReadyRevisionName"] = azureContainerAppsStringOr(properties["latestReadyRevisionName"], latest)
		if _, ok := properties["outboundIPAddresses"]; !ok {
			properties["outboundIPAddresses"] = []any{"10.0.0.5"}
		}
		properties["runningStatus"] = azureContainerAppsStringOr(properties["runningStatus"], "Running")
	case "jobs":
		properties["latestExecutionName"] = azureContainerAppsStringOr(properties["latestExecutionName"], resource.Name+"-execution-1")
	}

	fixture := map[string]any{
		"id":         azureContainerAppsID(resource.SubscriptionID, resource.ResourceGroup, resource.Collection, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.App/" + resource.Collection,
		"kind":       resource.Kind,
		"location":   resource.Location,
		"identity":   azureContainerAppsCloneObject(resource.Identity),
		"tags":       azureContainerAppsStringMapToAny(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureContainerAppsSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": properties,
	}
	return azureContainerAppsPruneNil(fixture)
}

func azureContainerAppsChildFixture(child *azureContainerAppsChildResource) map[string]any {
	fixture := map[string]any{
		"id":         azureContainerAppsChildID(child.SubscriptionID, child.ResourceGroup, child.ParentCollection, child.ParentName, child.Collection, child.Name),
		"name":       child.Name,
		"type":       "Microsoft.App/" + child.ParentCollection + "/" + child.Collection,
		"location":   child.Location,
		"tags":       azureContainerAppsStringMapToAny(child.Tags),
		"etag":       child.ETag,
		"systemData": azureContainerAppsSystemData(child.CreatedAt, child.UpdatedAt),
		"properties": azureContainerAppsCloneObject(child.Properties),
	}
	return azureContainerAppsPruneNil(fixture)
}

func azureContainerAppsNotFound(path, name string) map[string]any {
	return map[string]any{
		"error":    "NotFound",
		"message":  fmt.Sprintf("Azure Container Apps resource %q was not found", name),
		"provider": providerAzure,
		"path":     path,
	}
}

func azureContainerAppsNormalizeProperties(collection, name string, properties map[string]any, createdAt, updatedAt time.Time) map[string]any {
	normalized := azureContainerAppsCloneObject(properties)
	normalized["provisioningState"] = "Succeeded"
	switch collection {
	case "managedEnvironments":
		normalized["defaultDomain"] = azureContainerAppsStringOr(normalized["defaultDomain"], strings.ToLower(name)+".eastus.azurecontainerapps.io")
		normalized["staticIp"] = azureContainerAppsStringOr(normalized["staticIp"], "10.0.0.4")
	case "containerApps":
		latest := azureContainerAppsStringOr(normalized["latestRevisionName"], name+"--rev1")
		normalized["latestRevisionName"] = latest
		normalized["latestReadyRevisionName"] = azureContainerAppsStringOr(normalized["latestReadyRevisionName"], latest)
		if _, ok := normalized["outboundIPAddresses"]; !ok {
			normalized["outboundIPAddresses"] = []any{"10.0.0.5"}
		}
		normalized["runningStatus"] = azureContainerAppsStringOr(normalized["runningStatus"], "Running")
	case "jobs":
		normalized["latestExecutionName"] = azureContainerAppsStringOr(normalized["latestExecutionName"], name+"-execution-1")
	}
	return normalized
}

func azureContainerAppsNormalizeChildProperties(collection, name string, properties map[string]any) map[string]any {
	normalized := azureContainerAppsCloneObject(properties)
	switch collection {
	case "authConfigs":
		normalized["platform"] = azureContainerAppsCloneAny(azureContainerAppsObject(normalized["platform"]))
	case "diagnostics":
		normalized["status"] = azureContainerAppsStringOr(normalized["status"], "Healthy")
	case "sessionPools":
		normalized["poolStatus"] = azureContainerAppsStringOr(normalized["poolStatus"], "Ready")
	case "sourceControls":
		normalized["repoUrl"] = azureContainerAppsStringOr(normalized["repoUrl"], "https://github.com/stackyard/example")
	case "storages":
		normalized["accessMode"] = azureContainerAppsStringOr(normalized["accessMode"], "ReadWrite")
	case "privateEndpointConnections":
		normalized["status"] = azureContainerAppsStringOr(normalized["status"], "Approved")
	case "privateLinkResources":
		normalized["groupId"] = azureContainerAppsStringOr(normalized["groupId"], name)
	}
	return normalized
}

func azureContainerAppsReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureContainerAppsResourceKey(subscriptionID, resourceGroup, collection, name string) string {
	return strings.ToLower(strings.Join([]string{subscriptionID, resourceGroup, collection, name}, "|"))
}

func azureContainerAppsChildKey(subscriptionID, resourceGroup, parentCollection, parentName, collection, name string) string {
	return strings.ToLower(strings.Join([]string{subscriptionID, resourceGroup, parentCollection, parentName, collection, name}, "|"))
}

func azureContainerAppsID(subscriptionID, resourceGroup, collection, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.App/%s/%s", subscriptionID, resourceGroup, collection, name)
}

func azureContainerAppsChildID(subscriptionID, resourceGroup, parentCollection, parentName, collection, name string) string {
	return fmt.Sprintf("%s/%s/%s", azureContainerAppsID(subscriptionID, resourceGroup, parentCollection, parentName), collection, name)
}

func azureContainerAppsETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureContainerAppsSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureContainerAppsStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func azureContainerAppsMergeObjects(base, patch map[string]any) map[string]any {
	if len(base) == 0 && len(patch) == 0 {
		return map[string]any{}
	}
	out := azureContainerAppsCloneObject(base)
	for key, value := range patch {
		if existing, ok := out[key].(map[string]any); ok {
			if patchMap, ok := value.(map[string]any); ok {
				out[key] = azureContainerAppsMergeObjects(existing, patchMap)
				continue
			}
		}
		out[key] = azureContainerAppsCloneAny(value)
	}
	return out
}

func azureContainerAppsCloneObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = azureContainerAppsCloneAny(item)
	}
	return out
}

func azureContainerAppsCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return azureContainerAppsCloneObject(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = azureContainerAppsCloneAny(item)
		}
		return out
	default:
		return typed
	}
}

func azureContainerAppsObject(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return azureContainerAppsCloneObject(typed)
	}
	return map[string]any{}
}

func azureContainerAppsString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureContainerAppsStringOr(value any, fallback string) string {
	if str := azureContainerAppsString(value); str != "" {
		return str
	}
	return fallback
}

func azureContainerAppsStringMap(value any) map[string]string {
	items, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(items))
	for key, item := range items {
		if str := azureContainerAppsString(item); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureContainerAppsStringMapToAny(value map[string]string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func azureContainerAppsPruneNil(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		switch typed := item.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
		case map[string]any:
			if len(typed) == 0 {
				continue
			}
		case []any:
			if len(typed) == 0 {
				continue
			}
		}
		out[key] = item
	}
	return out
}
