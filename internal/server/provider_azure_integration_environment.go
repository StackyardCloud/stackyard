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

const azureIntegrationEnvironmentAPIVersion = "2023-11-14-preview"

type azureIntegrationSpace struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureIntegrationApplication struct {
	SpaceID    string
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureIntegrationApplicationResource struct {
	ApplicationID string
	Name          string
	Location      string
	Tags          map[string]string
	Properties    map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ETag          string
}

type azureIntegrationBusinessProcess struct {
	ApplicationID string
	Name          string
	Location      string
	Tags          map[string]string
	Properties    map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ETag          string
}

type azureIntegrationBusinessProcessVersion struct {
	BusinessProcessID string
	Name              string
	Properties        map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type azureIntegrationInfrastructureResource struct {
	SpaceID    string
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureIntegrationArtifact struct {
	ApplicationID string
	Name          string
	Content       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (s *Server) handleAzureIntegrationEnvironmentRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsIntegrationEnvironmentPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureIntegrationEnvironmentAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.IntegrationSpaces") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureIntegrationEnvironmentOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureIntegrationEnvironmentRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}
	if !strings.EqualFold(tail[0], "spaces") {
		respondAzureImplemented(w, path)
		return true
	}

	s.handleAzureIntegrationSpacesRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsIntegrationEnvironmentPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.IntegrationSpaces") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureIntegrationEnvironmentRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "spaces")
}

func azureIntegrationEnvironmentRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.IntegrationSpaces") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureIntegrationEnvironmentAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureIntegrationSpacesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureIntegrationListSpacesBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureIntegrationListSpacesByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	spaceName := strings.TrimSpace(tail[1])
	if spaceName == "" {
		respondAzureInvalidRequest(w, path, "space name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureIntegrationGetSpace(w, path, subscriptionID, resourceGroup, spaceName, false)
		case http.MethodHead:
			s.handleAzureIntegrationGetSpace(w, path, subscriptionID, resourceGroup, spaceName, true)
		case http.MethodPut:
			s.handleAzureIntegrationCreateOrUpdateSpace(w, r, path, subscriptionID, resourceGroup, spaceName)
		case http.MethodPatch:
			s.handleAzureIntegrationPatchSpace(w, r, path, subscriptionID, resourceGroup, spaceName)
		case http.MethodDelete:
			s.handleAzureIntegrationDeleteSpace(w, subscriptionID, resourceGroup, spaceName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "applications":
		s.handleAzureIntegrationApplicationsRoute(w, r, path, subscriptionID, resourceGroup, spaceName, tail[3:])
	case "infrastructureresources":
		s.handleAzureIntegrationInfrastructureResourcesRoute(w, r, path, subscriptionID, resourceGroup, spaceName, tail[3:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureIntegrationApplicationsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName string, tail []string) {
	spaceID := azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureIntegrationListApplications(w, spaceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	applicationName := strings.TrimSpace(tail[0])
	if applicationName == "" {
		respondAzureInvalidRequest(w, path, "application name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureIntegrationGetApplication(w, path, spaceID, applicationName, false)
		case http.MethodHead:
			s.handleAzureIntegrationGetApplication(w, path, spaceID, applicationName, true)
		case http.MethodPut:
			s.handleAzureIntegrationCreateOrUpdateApplication(w, r, path, subscriptionID, resourceGroup, spaceName, applicationName)
		case http.MethodPatch:
			s.handleAzureIntegrationPatchApplication(w, r, path, spaceID, applicationName)
		case http.MethodDelete:
			s.handleAzureIntegrationDeleteApplication(w, spaceID, applicationName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "resources":
		s.handleAzureIntegrationApplicationResourcesRoute(w, r, path, subscriptionID, resourceGroup, spaceName, applicationName, tail[2:])
	case "businessprocesses":
		s.handleAzureIntegrationBusinessProcessesRoute(w, r, path, subscriptionID, resourceGroup, spaceName, applicationName, tail[2:])
	case "savebusinessprocessdevelopmentartifact":
		if r.Method == http.MethodPost && len(tail) == 2 {
			s.handleAzureIntegrationSaveArtifact(w, r, path, spaceID, applicationName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "getbusinessprocessdevelopmentartifact":
		if r.Method == http.MethodPost && len(tail) == 2 {
			s.handleAzureIntegrationGetArtifact(w, r, path, spaceID, applicationName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "deletebusinessprocessdevelopmentartifact":
		if r.Method == http.MethodPost && len(tail) == 2 {
			s.handleAzureIntegrationDeleteArtifact(w, r, path, spaceID, applicationName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "listbusinessprocessdevelopmentartifacts":
		if r.Method == http.MethodPost && len(tail) == 2 {
			s.handleAzureIntegrationListArtifacts(w, spaceID, applicationName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "validatebusinessprocessdevelopmentartifact":
		if r.Method == http.MethodPost && len(tail) == 2 {
			s.handleAzureIntegrationValidateArtifact(w, r, path, spaceID, applicationName)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureIntegrationApplicationResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName, applicationName string, tail []string) {
	applicationID := azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureIntegrationListApplicationResources(w, applicationID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	resourceName := strings.TrimSpace(tail[0])
	if resourceName == "" {
		respondAzureInvalidRequest(w, path, "resource name is required")
		return
	}
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureIntegrationGetApplicationResource(w, path, applicationID, resourceName, false)
	case http.MethodHead:
		s.handleAzureIntegrationGetApplicationResource(w, path, applicationID, resourceName, true)
	case http.MethodPut:
		s.handleAzureIntegrationCreateOrUpdateApplicationResource(w, r, path, subscriptionID, resourceGroup, spaceName, applicationName, resourceName)
	case http.MethodPatch:
		s.handleAzureIntegrationPatchApplicationResource(w, r, path, applicationID, resourceName)
	case http.MethodDelete:
		s.handleAzureIntegrationDeleteApplicationResource(w, applicationID, resourceName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureIntegrationBusinessProcessesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName, applicationName string, tail []string) {
	applicationID := azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureIntegrationListBusinessProcesses(w, applicationID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	processName := strings.TrimSpace(tail[0])
	if processName == "" {
		respondAzureInvalidRequest(w, path, "business process name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureIntegrationGetBusinessProcess(w, path, applicationID, processName, false)
		case http.MethodHead:
			s.handleAzureIntegrationGetBusinessProcess(w, path, applicationID, processName, true)
		case http.MethodPut:
			s.handleAzureIntegrationCreateOrUpdateBusinessProcess(w, r, path, subscriptionID, resourceGroup, spaceName, applicationName, processName)
		case http.MethodPatch:
			s.handleAzureIntegrationPatchBusinessProcess(w, r, path, applicationID, processName)
		case http.MethodDelete:
			s.handleAzureIntegrationDeleteBusinessProcess(w, applicationID, processName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if !strings.EqualFold(tail[1], "versions") {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureIntegrationBusinessProcessVersionsRoute(w, r, path, applicationID, processName, tail[2:])
}

func (s *Server) handleAzureIntegrationBusinessProcessVersionsRoute(w http.ResponseWriter, r *http.Request, path, applicationID, processName string, tail []string) {
	processID := azureIntegrationBusinessProcessIDFromApplicationID(applicationID, processName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureIntegrationListBusinessProcessVersions(w, processID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) != 1 || r.Method != http.MethodGet {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureIntegrationGetBusinessProcessVersion(w, path, processID, strings.TrimSpace(tail[0]))
}

func (s *Server) handleAzureIntegrationInfrastructureResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName string, tail []string) {
	spaceID := azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureIntegrationListInfrastructureResources(w, spaceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "infrastructure resource name is required")
		return
	}
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureIntegrationGetInfrastructureResource(w, path, spaceID, name, false)
	case http.MethodHead:
		s.handleAzureIntegrationGetInfrastructureResource(w, path, spaceID, name, true)
	case http.MethodPatch:
		s.handleAzureIntegrationPatchInfrastructureResource(w, r, path, subscriptionID, resourceGroup, spaceName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureIntegrationCreateOrUpdateSpace(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureIntegrationString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	key := azureIntegrationSpaceKey(subscriptionID, resourceGroup, spaceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	space := s.azureIntegrationSpaces[key]
	status := http.StatusOK
	createdAt := now
	if space == nil {
		status = http.StatusCreated
		space = &azureIntegrationSpace{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           spaceName,
		}
	} else {
		createdAt = space.CreatedAt
	}
	space.Location = location
	space.Tags = azureIntegrationStringMap(payload["tags"])
	space.Properties = azureIntegrationNormalizeSpaceProperties(spaceName, location, azureIntegrationObject(payload["properties"]))
	space.CreatedAt = createdAt
	space.UpdatedAt = now
	space.ETag = azureIntegrationETag(key, body, now)
	s.azureIntegrationSpaces[key] = space
	s.azureIntegrationEnsureSpaceFixturesLocked(space)

	respondJSON(w, status, azureIntegrationSpaceFixture(space))
}

func (s *Server) handleAzureIntegrationPatchSpace(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := azureIntegrationSpaceKey(subscriptionID, resourceGroup, spaceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	space := s.azureIntegrationSpaces[key]
	if space == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, spaceName))
		return
	}
	if _, ok := payload["tags"]; ok {
		space.Tags = azureIntegrationStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		space.Properties = azureIntegrationMergeObjects(space.Properties, azureIntegrationObject(payload["properties"]))
		space.Properties = azureIntegrationNormalizeSpaceProperties(space.Name, space.Location, space.Properties)
	}
	space.UpdatedAt = now
	space.ETag = azureIntegrationETag(key, body, now)
	respondJSON(w, http.StatusOK, azureIntegrationSpaceFixture(space))
}

func (s *Server) handleAzureIntegrationGetSpace(w http.ResponseWriter, path, subscriptionID, resourceGroup, spaceName string, headOnly bool) {
	s.providerStorageMu.Lock()
	space := s.azureIntegrationSpaces[azureIntegrationSpaceKey(subscriptionID, resourceGroup, spaceName)]
	s.providerStorageMu.Unlock()
	if space == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, spaceName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureIntegrationSpaceFixture(space))
}

func (s *Server) handleAzureIntegrationDeleteSpace(w http.ResponseWriter, subscriptionID, resourceGroup, spaceName string) {
	key := azureIntegrationSpaceKey(subscriptionID, resourceGroup, spaceName)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureIntegrationSpaces[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureIntegrationSpaces, key)
	prefix := key + "/"
	for id := range s.azureIntegrationApplications {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationApplications, id)
		}
	}
	for id := range s.azureIntegrationApplicationResources {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationApplicationResources, id)
		}
	}
	for id := range s.azureIntegrationBusinessProcesses {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationBusinessProcesses, id)
		}
	}
	for id := range s.azureIntegrationBusinessProcessVersions {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationBusinessProcessVersions, id)
		}
	}
	for id := range s.azureIntegrationInfrastructureResources {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationInfrastructureResources, id)
		}
	}
	for id := range s.azureIntegrationArtifacts {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationArtifacts, id)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureIntegrationListSpacesBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	for _, space := range s.azureIntegrationSpaces {
		if strings.EqualFold(space.SubscriptionID, subscriptionID) {
			values = append(values, azureIntegrationSpaceFixture(space))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationListSpacesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	for _, space := range s.azureIntegrationSpaces {
		if strings.EqualFold(space.SubscriptionID, subscriptionID) && strings.EqualFold(space.ResourceGroup, resourceGroup) {
			values = append(values, azureIntegrationSpaceFixture(space))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationCreateOrUpdateApplication(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName, applicationName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	spaceID := azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName)
	now := time.Now().UTC()
	key := azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	space := s.azureIntegrationSpaces[spaceID]
	if space == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, spaceName))
		return
	}

	app := s.azureIntegrationApplications[key]
	status := http.StatusOK
	createdAt := now
	if app == nil {
		status = http.StatusCreated
		app = &azureIntegrationApplication{
			SpaceID: spaceID,
			Name:    applicationName,
		}
	} else {
		createdAt = app.CreatedAt
	}
	app.Location = space.Location
	app.Tags = azureIntegrationStringMap(payload["tags"])
	app.Properties = azureIntegrationNormalizeApplicationProperties(applicationName, azureIntegrationObject(payload["properties"]))
	app.CreatedAt = createdAt
	app.UpdatedAt = now
	app.ETag = azureIntegrationETag(key, body, now)
	s.azureIntegrationApplications[key] = app

	respondJSON(w, status, azureIntegrationApplicationFixture(subscriptionID, resourceGroup, spaceName, app))
}

func (s *Server) handleAzureIntegrationPatchApplication(w http.ResponseWriter, r *http.Request, path, spaceID, applicationName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	app := s.azureIntegrationApplications[key]
	if app == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, applicationName))
		return
	}
	if _, ok := payload["tags"]; ok {
		app.Tags = azureIntegrationStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		app.Properties = azureIntegrationMergeObjects(app.Properties, azureIntegrationObject(payload["properties"]))
		app.Properties = azureIntegrationNormalizeApplicationProperties(app.Name, app.Properties)
	}
	app.UpdatedAt = now
	app.ETag = azureIntegrationETag(key, body, now)
	subscriptionID, resourceGroup, spaceName := azureIntegrationParseSpaceID(spaceID)
	respondJSON(w, http.StatusOK, azureIntegrationApplicationFixture(subscriptionID, resourceGroup, spaceName, app))
}

func (s *Server) handleAzureIntegrationGetApplication(w http.ResponseWriter, path, spaceID, applicationName string, headOnly bool) {
	key := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	s.providerStorageMu.Lock()
	app := s.azureIntegrationApplications[key]
	s.providerStorageMu.Unlock()
	if app == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, applicationName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	subscriptionID, resourceGroup, spaceName := azureIntegrationParseSpaceID(spaceID)
	respondJSON(w, http.StatusOK, azureIntegrationApplicationFixture(subscriptionID, resourceGroup, spaceName, app))
}

func (s *Server) handleAzureIntegrationDeleteApplication(w http.ResponseWriter, spaceID, applicationName string) {
	key := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureIntegrationApplications[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureIntegrationApplications, key)
	prefix := key + "/"
	for id := range s.azureIntegrationApplicationResources {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationApplicationResources, id)
		}
	}
	for id := range s.azureIntegrationBusinessProcesses {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationBusinessProcesses, id)
		}
	}
	for id := range s.azureIntegrationBusinessProcessVersions {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationBusinessProcessVersions, id)
		}
	}
	for id := range s.azureIntegrationArtifacts {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureIntegrationArtifacts, id)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureIntegrationListApplications(w http.ResponseWriter, spaceID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	subscriptionID, resourceGroup, spaceName := azureIntegrationParseSpaceID(spaceID)
	for _, app := range s.azureIntegrationApplications {
		if strings.EqualFold(app.SpaceID, spaceID) {
			values = append(values, azureIntegrationApplicationFixture(subscriptionID, resourceGroup, spaceName, app))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationCreateOrUpdateApplicationResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName, applicationName, resourceName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	properties := azureIntegrationObject(payload["properties"])
	if strings.TrimSpace(azureIntegrationString(properties["resourceId"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.resourceId is required")
		return
	}
	if strings.TrimSpace(azureIntegrationString(properties["resourceType"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.resourceType is required")
		return
	}
	applicationID := azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName)
	now := time.Now().UTC()
	key := applicationID + "/resources/" + resourceName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	app := s.azureIntegrationApplications[applicationID]
	if app == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, applicationName))
		return
	}
	resource := s.azureIntegrationApplicationResources[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		status = http.StatusCreated
		resource = &azureIntegrationApplicationResource{
			ApplicationID: applicationID,
			Name:          resourceName,
		}
	} else {
		createdAt = resource.CreatedAt
	}
	resource.Location = app.Location
	resource.Tags = azureIntegrationStringMap(payload["tags"])
	resource.Properties = azureIntegrationNormalizeApplicationResourceProperties(resourceName, properties)
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureIntegrationETag(key, body, now)
	s.azureIntegrationApplicationResources[key] = resource

	respondJSON(w, status, azureIntegrationApplicationResourceFixture(subscriptionID, resourceGroup, spaceName, applicationName, resource))
}

func (s *Server) handleAzureIntegrationPatchApplicationResource(w http.ResponseWriter, r *http.Request, path, applicationID, resourceName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := applicationID + "/resources/" + resourceName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureIntegrationApplicationResources[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, resourceName))
		return
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureIntegrationStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		resource.Properties = azureIntegrationMergeObjects(resource.Properties, azureIntegrationObject(payload["properties"]))
		resource.Properties = azureIntegrationNormalizeApplicationResourceProperties(resource.Name, resource.Properties)
	}
	resource.UpdatedAt = now
	resource.ETag = azureIntegrationETag(key, body, now)
	subscriptionID, resourceGroup, spaceName, applicationName := azureIntegrationParseApplicationID(applicationID)
	respondJSON(w, http.StatusOK, azureIntegrationApplicationResourceFixture(subscriptionID, resourceGroup, spaceName, applicationName, resource))
}

func (s *Server) handleAzureIntegrationGetApplicationResource(w http.ResponseWriter, path, applicationID, resourceName string, headOnly bool) {
	key := applicationID + "/resources/" + resourceName
	s.providerStorageMu.Lock()
	resource := s.azureIntegrationApplicationResources[key]
	s.providerStorageMu.Unlock()
	if resource == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, resourceName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	subscriptionID, resourceGroup, spaceName, applicationName := azureIntegrationParseApplicationID(applicationID)
	respondJSON(w, http.StatusOK, azureIntegrationApplicationResourceFixture(subscriptionID, resourceGroup, spaceName, applicationName, resource))
}

func (s *Server) handleAzureIntegrationDeleteApplicationResource(w http.ResponseWriter, applicationID, resourceName string) {
	key := applicationID + "/resources/" + resourceName
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureIntegrationApplicationResources[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureIntegrationApplicationResources, key)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureIntegrationListApplicationResources(w http.ResponseWriter, applicationID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	subscriptionID, resourceGroup, spaceName, applicationName := azureIntegrationParseApplicationID(applicationID)
	for _, resource := range s.azureIntegrationApplicationResources {
		if strings.EqualFold(resource.ApplicationID, applicationID) {
			values = append(values, azureIntegrationApplicationResourceFixture(subscriptionID, resourceGroup, spaceName, applicationName, resource))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationCreateOrUpdateBusinessProcess(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName, applicationName, processName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	properties := azureIntegrationObject(payload["properties"])
	if strings.TrimSpace(azureIntegrationString(properties["tableName"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.tableName is required")
		return
	}
	if strings.TrimSpace(azureIntegrationString(properties["identifier"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.identifier is required")
		return
	}
	applicationID := azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName)
	now := time.Now().UTC()
	key := applicationID + "/businessProcesses/" + processName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	app := s.azureIntegrationApplications[applicationID]
	if app == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, applicationName))
		return
	}
	process := s.azureIntegrationBusinessProcesses[key]
	status := http.StatusOK
	createdAt := now
	if process == nil {
		status = http.StatusCreated
		process = &azureIntegrationBusinessProcess{
			ApplicationID: applicationID,
			Name:          processName,
		}
	} else {
		createdAt = process.CreatedAt
	}
	process.Location = app.Location
	process.Tags = azureIntegrationStringMap(payload["tags"])
	process.Properties = azureIntegrationNormalizeBusinessProcessProperties(processName, properties)
	process.CreatedAt = createdAt
	process.UpdatedAt = now
	process.ETag = azureIntegrationETag(key, body, now)
	s.azureIntegrationBusinessProcesses[key] = process
	s.azureIntegrationEnsureBusinessProcessVersionLocked(process)

	respondJSON(w, status, azureIntegrationBusinessProcessFixture(subscriptionID, resourceGroup, spaceName, applicationName, process))
}

func (s *Server) handleAzureIntegrationPatchBusinessProcess(w http.ResponseWriter, r *http.Request, path, applicationID, processName string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	key := applicationID + "/businessProcesses/" + processName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	process := s.azureIntegrationBusinessProcesses[key]
	if process == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, processName))
		return
	}
	if _, ok := payload["tags"]; ok {
		process.Tags = azureIntegrationStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		process.Properties = azureIntegrationMergeObjects(process.Properties, azureIntegrationObject(payload["properties"]))
		process.Properties = azureIntegrationNormalizeBusinessProcessProperties(process.Name, process.Properties)
	}
	process.UpdatedAt = now
	process.ETag = azureIntegrationETag(key, body, now)
	subscriptionID, resourceGroup, spaceName, applicationName := azureIntegrationParseApplicationID(applicationID)
	respondJSON(w, http.StatusOK, azureIntegrationBusinessProcessFixture(subscriptionID, resourceGroup, spaceName, applicationName, process))
}

func (s *Server) handleAzureIntegrationGetBusinessProcess(w http.ResponseWriter, path, applicationID, processName string, headOnly bool) {
	key := applicationID + "/businessProcesses/" + processName
	s.providerStorageMu.Lock()
	process := s.azureIntegrationBusinessProcesses[key]
	s.providerStorageMu.Unlock()
	if process == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, processName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	subscriptionID, resourceGroup, spaceName, applicationName := azureIntegrationParseApplicationID(applicationID)
	respondJSON(w, http.StatusOK, azureIntegrationBusinessProcessFixture(subscriptionID, resourceGroup, spaceName, applicationName, process))
}

func (s *Server) handleAzureIntegrationDeleteBusinessProcess(w http.ResponseWriter, applicationID, processName string) {
	key := applicationID + "/businessProcesses/" + processName
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureIntegrationBusinessProcesses[key]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureIntegrationBusinessProcesses, key)
	for id := range s.azureIntegrationBusinessProcessVersions {
		if strings.HasPrefix(id, key+"/versions/") {
			delete(s.azureIntegrationBusinessProcessVersions, id)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureIntegrationListBusinessProcesses(w http.ResponseWriter, applicationID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	subscriptionID, resourceGroup, spaceName, applicationName := azureIntegrationParseApplicationID(applicationID)
	for _, process := range s.azureIntegrationBusinessProcesses {
		if strings.EqualFold(process.ApplicationID, applicationID) {
			values = append(values, azureIntegrationBusinessProcessFixture(subscriptionID, resourceGroup, spaceName, applicationName, process))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationListBusinessProcessVersions(w http.ResponseWriter, processID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	subscriptionID, resourceGroup, spaceName, applicationName, processName := azureIntegrationParseBusinessProcessID(processID)
	for _, version := range s.azureIntegrationBusinessProcessVersions {
		if strings.EqualFold(version.BusinessProcessID, processID) {
			values = append(values, azureIntegrationBusinessProcessVersionFixture(subscriptionID, resourceGroup, spaceName, applicationName, processName, version))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationGetBusinessProcessVersion(w http.ResponseWriter, path, processID, versionName string) {
	key := processID + "/versions/" + versionName
	s.providerStorageMu.Lock()
	version := s.azureIntegrationBusinessProcessVersions[key]
	s.providerStorageMu.Unlock()
	if version == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, versionName))
		return
	}
	subscriptionID, resourceGroup, spaceName, applicationName, processName := azureIntegrationParseBusinessProcessID(processID)
	respondJSON(w, http.StatusOK, azureIntegrationBusinessProcessVersionFixture(subscriptionID, resourceGroup, spaceName, applicationName, processName, version))
}

func (s *Server) handleAzureIntegrationPatchInfrastructureResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, spaceName, name string) {
	body, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	spaceID := azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName)
	now := time.Now().UTC()
	key := spaceID + "/infrastructureResources/" + name

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	space := s.azureIntegrationSpaces[spaceID]
	if space == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, spaceName))
		return
	}
	resource := s.azureIntegrationInfrastructureResources[key]
	if resource == nil {
		resource = &azureIntegrationInfrastructureResource{
			SpaceID:    spaceID,
			Name:       name,
			Location:   space.Location,
			Tags:       map[string]string{},
			Properties: azureIntegrationNormalizeInfrastructureResourceProperties(name, map[string]any{}),
			CreatedAt:  now,
		}
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureIntegrationStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		resource.Properties = azureIntegrationMergeObjects(resource.Properties, azureIntegrationObject(payload["properties"]))
		resource.Properties = azureIntegrationNormalizeInfrastructureResourceProperties(name, resource.Properties)
	}
	resource.UpdatedAt = now
	resource.ETag = azureIntegrationETag(key, body, now)
	s.azureIntegrationInfrastructureResources[key] = resource

	respondJSON(w, http.StatusOK, azureIntegrationInfrastructureResourceFixture(subscriptionID, resourceGroup, spaceName, resource))
}

func (s *Server) handleAzureIntegrationGetInfrastructureResource(w http.ResponseWriter, path, spaceID, name string, headOnly bool) {
	key := spaceID + "/infrastructureResources/" + name
	s.providerStorageMu.Lock()
	resource := s.azureIntegrationInfrastructureResources[key]
	s.providerStorageMu.Unlock()
	if resource == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	parts := splitPathSegments(spaceID)
	respondJSON(w, http.StatusOK, azureIntegrationInfrastructureResourceFixture(parts[1], parts[3], parts[len(parts)-1], resource))
}

func (s *Server) handleAzureIntegrationListInfrastructureResources(w http.ResponseWriter, spaceID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	parts := splitPathSegments(spaceID)
	for _, resource := range s.azureIntegrationInfrastructureResources {
		if strings.EqualFold(resource.SpaceID, spaceID) {
			values = append(values, azureIntegrationInfrastructureResourceFixture(parts[1], parts[3], parts[len(parts)-1], resource))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["id"]) < azureIntegrationString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationSaveArtifact(w http.ResponseWriter, r *http.Request, path, spaceID, applicationName string) {
	_, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	applicationID := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	name := strings.TrimSpace(azureIntegrationString(payload["artifactName"]))
	if name == "" {
		name = strings.TrimSpace(azureIntegrationString(payload["name"]))
	}
	if name == "" {
		respondAzureInvalidRequest(w, path, "artifactName is required")
		return
	}
	content := strings.TrimSpace(azureIntegrationString(payload["content"]))
	if content == "" {
		content = "{}"
	}
	now := time.Now().UTC()
	key := applicationID + "/artifacts/" + name

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if s.azureIntegrationApplications[applicationID] == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, applicationName))
		return
	}
	artifact := s.azureIntegrationArtifacts[key]
	if artifact == nil {
		artifact = &azureIntegrationArtifact{ApplicationID: applicationID, Name: name, CreatedAt: now}
	}
	artifact.Content = content
	artifact.UpdatedAt = now
	s.azureIntegrationArtifacts[key] = artifact
	respondJSON(w, http.StatusOK, azureIntegrationArtifactFixture(artifact))
}

func (s *Server) handleAzureIntegrationGetArtifact(w http.ResponseWriter, r *http.Request, path, spaceID, applicationName string) {
	_, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	applicationID := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	name := strings.TrimSpace(azureIntegrationString(payload["artifactName"]))
	if name == "" {
		name = strings.TrimSpace(azureIntegrationString(payload["name"]))
	}
	if name == "" {
		respondAzureInvalidRequest(w, path, "artifactName is required")
		return
	}
	key := applicationID + "/artifacts/" + name
	s.providerStorageMu.Lock()
	artifact := s.azureIntegrationArtifacts[key]
	s.providerStorageMu.Unlock()
	if artifact == nil {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureIntegrationArtifactFixture(artifact))
}

func (s *Server) handleAzureIntegrationDeleteArtifact(w http.ResponseWriter, r *http.Request, path, spaceID, applicationName string) {
	_, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	applicationID := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	name := strings.TrimSpace(azureIntegrationString(payload["artifactName"]))
	if name == "" {
		name = strings.TrimSpace(azureIntegrationString(payload["name"]))
	}
	if name == "" {
		respondAzureInvalidRequest(w, path, "artifactName is required")
		return
	}
	key := applicationID + "/artifacts/" + name
	s.providerStorageMu.Lock()
	delete(s.azureIntegrationArtifacts, key)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{"deleted": true, "artifactName": name})
}

func (s *Server) handleAzureIntegrationListArtifacts(w http.ResponseWriter, spaceID, applicationName string) {
	applicationID := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	for _, artifact := range s.azureIntegrationArtifacts {
		if strings.EqualFold(artifact.ApplicationID, applicationID) {
			values = append(values, azureIntegrationArtifactFixture(artifact))
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return azureIntegrationString(values[i]["name"]) < azureIntegrationString(values[j]["name"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureIntegrationValidateArtifact(w http.ResponseWriter, r *http.Request, path, spaceID, applicationName string) {
	_, payload, ok := azureIntegrationReadJSONBody(w, r, path)
	if !ok {
		return
	}
	applicationID := strings.TrimRight(spaceID, "/") + "/applications/" + applicationName
	if s.azureIntegrationApplicationExists(applicationID) == false {
		respondJSON(w, http.StatusNotFound, azureIntegrationNotFound(path, applicationName))
		return
	}
	content := strings.TrimSpace(azureIntegrationString(payload["content"]))
	artifactName := strings.TrimSpace(azureIntegrationString(payload["artifactName"]))
	if artifactName == "" {
		artifactName = strings.TrimSpace(azureIntegrationString(payload["name"]))
	}
	if artifactName == "" {
		respondAzureInvalidRequest(w, path, "artifactName is required")
		return
	}
	valid := !strings.Contains(strings.ToLower(content), "invalid")
	errors := []any{}
	if !valid {
		errors = append(errors, map[string]any{
			"code":    "ArtifactValidationFailed",
			"message": "Artifact content failed mocked validation.",
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"artifactName": artifactName,
		"isValid":      valid,
		"errors":       errors,
	})
}

func (s *Server) azureIntegrationApplicationExists(applicationID string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureIntegrationApplications[applicationID] != nil
}

func (s *Server) azureIntegrationEnsureSpaceFixturesLocked(space *azureIntegrationSpace) {
	key := azureIntegrationInfrastructureResourceID(space.SubscriptionID, space.ResourceGroup, space.Name, "default-runtime")
	if s.azureIntegrationInfrastructureResources[key] != nil {
		return
	}
	now := time.Now().UTC()
	s.azureIntegrationInfrastructureResources[key] = &azureIntegrationInfrastructureResource{
		SpaceID:    azureIntegrationSpaceID(space.SubscriptionID, space.ResourceGroup, space.Name),
		Name:       "default-runtime",
		Location:   space.Location,
		Tags:       map[string]string{"seeded": "true"},
		Properties: azureIntegrationNormalizeInfrastructureResourceProperties("default-runtime", map[string]any{}),
		CreatedAt:  now,
		UpdatedAt:  now,
		ETag:       azureIntegrationETag(key, []byte(key), now),
	}
}

func (s *Server) azureIntegrationEnsureBusinessProcessVersionLocked(process *azureIntegrationBusinessProcess) {
	key := process.ApplicationID + "/businessProcesses/" + process.Name + "/versions/1.0.0"
	if s.azureIntegrationBusinessProcessVersions[key] != nil {
		return
	}
	now := time.Now().UTC()
	s.azureIntegrationBusinessProcessVersions[key] = &azureIntegrationBusinessProcessVersion{
		BusinessProcessID: process.ApplicationID + "/businessProcesses/" + process.Name,
		Name:              "1.0.0",
		Properties:        azureIntegrationNormalizeBusinessProcessVersionProperties(process.Properties),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func azureIntegrationSpaceFixture(space *azureIntegrationSpace) map[string]any {
	return map[string]any{
		"id":         azureIntegrationSpaceID(space.SubscriptionID, space.ResourceGroup, space.Name),
		"name":       space.Name,
		"type":       "Microsoft.IntegrationSpaces/spaces",
		"location":   space.Location,
		"tags":       azureIntegrationAnyMap(space.Tags),
		"etag":       space.ETag,
		"systemData": azureIntegrationSystemData(space.CreatedAt, space.UpdatedAt),
		"properties": azureIntegrationCloneObject(space.Properties),
	}
}

func azureIntegrationApplicationFixture(subscriptionID, resourceGroup, spaceName string, app *azureIntegrationApplication) map[string]any {
	return map[string]any{
		"id":         azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, app.Name),
		"name":       app.Name,
		"type":       "Microsoft.IntegrationSpaces/spaces/applications",
		"location":   app.Location,
		"tags":       azureIntegrationAnyMap(app.Tags),
		"etag":       app.ETag,
		"systemData": azureIntegrationSystemData(app.CreatedAt, app.UpdatedAt),
		"properties": azureIntegrationCloneObject(app.Properties),
	}
}

func azureIntegrationApplicationResourceFixture(subscriptionID, resourceGroup, spaceName, applicationName string, resource *azureIntegrationApplicationResource) map[string]any {
	return map[string]any{
		"id":         azureIntegrationApplicationResourceID(subscriptionID, resourceGroup, spaceName, applicationName, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.IntegrationSpaces/spaces/applications/resources",
		"location":   resource.Location,
		"tags":       azureIntegrationAnyMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureIntegrationSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureIntegrationCloneObject(resource.Properties),
	}
}

func azureIntegrationBusinessProcessFixture(subscriptionID, resourceGroup, spaceName, applicationName string, process *azureIntegrationBusinessProcess) map[string]any {
	return map[string]any{
		"id":         azureIntegrationBusinessProcessID(subscriptionID, resourceGroup, spaceName, applicationName, process.Name),
		"name":       process.Name,
		"type":       "Microsoft.IntegrationSpaces/spaces/applications/businessProcesses",
		"location":   process.Location,
		"tags":       azureIntegrationAnyMap(process.Tags),
		"etag":       process.ETag,
		"systemData": azureIntegrationSystemData(process.CreatedAt, process.UpdatedAt),
		"properties": azureIntegrationCloneObject(process.Properties),
	}
}

func azureIntegrationBusinessProcessVersionFixture(subscriptionID, resourceGroup, spaceName, applicationName, processName string, version *azureIntegrationBusinessProcessVersion) map[string]any {
	return map[string]any{
		"id":         azureIntegrationBusinessProcessVersionID(subscriptionID, resourceGroup, spaceName, applicationName, processName, version.Name),
		"name":       version.Name,
		"type":       "Microsoft.IntegrationSpaces/spaces/applications/businessProcesses/versions",
		"systemData": azureIntegrationSystemData(version.CreatedAt, version.UpdatedAt),
		"properties": azureIntegrationCloneObject(version.Properties),
	}
}

func azureIntegrationInfrastructureResourceFixture(subscriptionID, resourceGroup, spaceName string, resource *azureIntegrationInfrastructureResource) map[string]any {
	return map[string]any{
		"id":         azureIntegrationInfrastructureResourceID(subscriptionID, resourceGroup, spaceName, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.IntegrationSpaces/spaces/infrastructureResources",
		"location":   resource.Location,
		"tags":       azureIntegrationAnyMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureIntegrationSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureIntegrationCloneObject(resource.Properties),
	}
}

func azureIntegrationArtifactFixture(artifact *azureIntegrationArtifact) map[string]any {
	return map[string]any{
		"name":       artifact.Name,
		"content":    artifact.Content,
		"createdAt":  artifact.CreatedAt.UTC().Format(time.RFC3339),
		"updatedAt":  artifact.UpdatedAt.UTC().Format(time.RFC3339),
		"properties": map[string]any{"applicationId": artifact.ApplicationID},
	}
}

func azureIntegrationNormalizeSpaceProperties(spaceName, location string, properties map[string]any) map[string]any {
	out := azureIntegrationCloneObject(properties)
	if _, ok := out["description"]; !ok {
		out["description"] = "Mocked integration space " + spaceName
	}
	out["provisioningState"] = "Succeeded"
	if _, ok := out["endpoint"]; !ok {
		out["endpoint"] = "https://" + spaceName + "." + strings.ToLower(location) + ".integrationspaces.azure.local"
	}
	return out
}

func azureIntegrationNormalizeApplicationProperties(applicationName string, properties map[string]any) map[string]any {
	out := azureIntegrationCloneObject(properties)
	if _, ok := out["displayName"]; !ok {
		out["displayName"] = azureIntegrationDisplayName(applicationName)
	}
	if _, ok := out["description"]; !ok {
		out["description"] = "Mocked integration application " + applicationName
	}
	out["provisioningState"] = "Succeeded"
	return out
}

func azureIntegrationNormalizeApplicationResourceProperties(resourceName string, properties map[string]any) map[string]any {
	out := azureIntegrationCloneObject(properties)
	if _, ok := out["resourceKind"]; !ok {
		out["resourceKind"] = "Connection"
	}
	if _, ok := out["displayName"]; !ok {
		out["displayName"] = resourceName
	}
	out["provisioningState"] = "Succeeded"
	return out
}

func azureIntegrationNormalizeBusinessProcessProperties(processName string, properties map[string]any) map[string]any {
	out := azureIntegrationCloneObject(properties)
	if _, ok := out["description"]; !ok {
		out["description"] = "Mocked business process " + processName
	}
	if _, ok := out["businessProcessStages"]; !ok {
		out["businessProcessStages"] = []any{
			map[string]any{"name": "ingest", "stageType": "Ingress"},
			map[string]any{"name": "process", "stageType": "Processing"},
		}
	}
	if _, ok := out["businessProcessMapping"]; !ok {
		out["businessProcessMapping"] = map[string]any{
			"trigger": "created",
			"target":  processName,
		}
	}
	out["provisioningState"] = "Succeeded"
	return out
}

func azureIntegrationNormalizeBusinessProcessVersionProperties(source map[string]any) map[string]any {
	out := azureIntegrationCloneObject(source)
	out["version"] = "1.0.0"
	out["isLatest"] = true
	return out
}

func azureIntegrationNormalizeInfrastructureResourceProperties(name string, properties map[string]any) map[string]any {
	out := azureIntegrationCloneObject(properties)
	if _, ok := out["resourceType"]; !ok {
		out["resourceType"] = "Runtime"
	}
	if _, ok := out["status"]; !ok {
		out["status"] = "Ready"
	}
	if _, ok := out["displayName"]; !ok {
		out["displayName"] = name
	}
	return out
}

func azureIntegrationEnvironmentOperationsFixture() []map[string]any {
	names := []string{
		"Microsoft.IntegrationSpaces/spaces/read",
		"Microsoft.IntegrationSpaces/spaces/write",
		"Microsoft.IntegrationSpaces/spaces/delete",
		"Microsoft.IntegrationSpaces/spaces/applications/read",
		"Microsoft.IntegrationSpaces/spaces/applications/write",
		"Microsoft.IntegrationSpaces/spaces/applications/delete",
		"Microsoft.IntegrationSpaces/spaces/applications/resources/read",
		"Microsoft.IntegrationSpaces/spaces/applications/resources/write",
		"Microsoft.IntegrationSpaces/spaces/applications/businessProcesses/read",
		"Microsoft.IntegrationSpaces/spaces/applications/businessProcesses/write",
		"Microsoft.IntegrationSpaces/spaces/infrastructureResources/read",
		"Microsoft.IntegrationSpaces/spaces/infrastructureResources/write",
	}
	values := make([]map[string]any, 0, len(names))
	for _, name := range names {
		values = append(values, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":    "Microsoft Integration Spaces",
				"resource":    "Azure Integration Environment",
				"operation":   name,
				"description": "Mocked Azure Integration Environment operation",
			},
			"origin": "user,system",
		})
	}
	return values
}

func azureIntegrationReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
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

func azureIntegrationSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureIntegrationETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.UTC().Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureIntegrationSpaceKey(subscriptionID, resourceGroup, spaceName string) string {
	return azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName)
}

func azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.IntegrationSpaces/spaces/" + spaceName
}

func azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName string) string {
	return azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName) + "/applications/" + applicationName
}

func azureIntegrationApplicationResourceID(subscriptionID, resourceGroup, spaceName, applicationName, resourceName string) string {
	return azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName) + "/resources/" + resourceName
}

func azureIntegrationBusinessProcessID(subscriptionID, resourceGroup, spaceName, applicationName, processName string) string {
	return azureIntegrationApplicationID(subscriptionID, resourceGroup, spaceName, applicationName) + "/businessProcesses/" + processName
}

func azureIntegrationBusinessProcessIDFromApplicationID(applicationID, processName string) string {
	return applicationID + "/businessProcesses/" + processName
}

func azureIntegrationBusinessProcessVersionID(subscriptionID, resourceGroup, spaceName, applicationName, processName, versionName string) string {
	return azureIntegrationBusinessProcessID(subscriptionID, resourceGroup, spaceName, applicationName, processName) + "/versions/" + versionName
}

func azureIntegrationInfrastructureResourceID(subscriptionID, resourceGroup, spaceName, name string) string {
	return azureIntegrationSpaceID(subscriptionID, resourceGroup, spaceName) + "/infrastructureResources/" + name
}

func azureIntegrationParseSpaceID(spaceID string) (subscriptionID, resourceGroup, spaceName string) {
	parts := splitPathSegments(spaceID)
	if len(parts) >= 8 {
		return parts[1], parts[3], parts[7]
	}
	return "", "", ""
}

func azureIntegrationParseApplicationID(applicationID string) (subscriptionID, resourceGroup, spaceName, applicationName string) {
	parts := splitPathSegments(applicationID)
	if len(parts) >= 10 {
		return parts[1], parts[3], parts[7], parts[9]
	}
	return "", "", "", ""
}

func azureIntegrationParseBusinessProcessID(processID string) (subscriptionID, resourceGroup, spaceName, applicationName, processName string) {
	parts := splitPathSegments(processID)
	if len(parts) >= 12 {
		return parts[1], parts[3], parts[7], parts[9], parts[11]
	}
	return "", "", "", "", ""
}

func azureIntegrationDisplayName(name string) string {
	parts := strings.Fields(strings.ReplaceAll(name, "-", " "))
	for idx, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(strings.ToLower(part))
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[idx] = string(runes)
	}
	return strings.Join(parts, " ")
}

func azureIntegrationNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("The Azure Integration Environment resource %q was not found for %s.", name, path),
		},
	}
}

func azureIntegrationCloneObject(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureIntegrationCloneObject(nested)
			continue
		}
		if values, ok := value.([]any); ok {
			cloned := make([]any, len(values))
			copy(cloned, values)
			out[key] = cloned
			continue
		}
		out[key] = value
	}
	return out
}

func azureIntegrationMergeObjects(base, overlay map[string]any) map[string]any {
	if len(base) == 0 {
		return azureIntegrationCloneObject(overlay)
	}
	out := azureIntegrationCloneObject(base)
	for key, value := range overlay {
		current, exists := out[key]
		currentObj, currentOK := current.(map[string]any)
		valueObj, valueOK := value.(map[string]any)
		if exists && currentOK && valueOK {
			out[key] = azureIntegrationMergeObjects(currentObj, valueObj)
			continue
		}
		out[key] = value
	}
	return out
}

func azureIntegrationString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func azureIntegrationObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func azureIntegrationStringMap(value any) map[string]string {
	object := azureIntegrationObject(value)
	out := make(map[string]string, len(object))
	for key, raw := range object {
		out[key] = azureIntegrationString(raw)
	}
	return out
}

func azureIntegrationAnyMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
