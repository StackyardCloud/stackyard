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

const azureSpringAppsAPIVersion = "2023-12-01"

type azureSpringService struct {
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

type azureSpringApp struct {
	ServiceID        string
	Name             string
	Location         string
	Tags             map[string]string
	Properties       map[string]any
	ActiveDeployment string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ETag             string
}

type azureSpringDeployment struct {
	AppID      string
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureSpringChildResource struct {
	ParentID   string
	Collection string
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureSpringBuildService struct {
	ServiceID  string
	Name       string
	Location   string
	Tags       map[string]string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ETag       string
}

type azureSpringBuild struct {
	BuildServiceID string
	Name           string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

var azureSpringServiceChildCollections = map[string]string{
	"certificates":          "certificates",
	"apms":                  "apms",
	"gateways":              "gateways",
	"apiportals":            "apiPortals",
	"applicationliveviews":  "applicationLiveViews",
	"configurationservices": "configurationServices",
	"serviceregistries":     "serviceRegistries",
	"devtoolportals":        "devToolPortals",
	"containerregistries":   "containerRegistries",
	"storages":              "storages",
}

var azureSpringAppChildCollections = map[string]string{
	"bindings":      "bindings",
	"domains":       "domains",
	"customdomains": "domains",
}

func (s *Server) handleAzureSpringAppsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureSpringAppsIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureSpringAppsAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AppPlatform") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureSpringAppsOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureSpringAppsRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}
	if !strings.EqualFold(tail[0], "Spring") {
		respondAzureImplemented(w, path)
		return true
	}

	s.handleAzureSpringServicesRoute(w, r, path, subscriptionID, resourceGroup, tail[1:])
	return true
}

func azureSpringAppsIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AppPlatform") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureSpringAppsRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "Spring")
}

func azureSpringAppsRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AppPlatform") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureSpringAppsAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureSpringServicesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			if resourceGroup == "" {
				s.handleAzureSpringListServicesBySubscription(w, subscriptionID)
				return
			}
			s.handleAzureSpringListServicesByResourceGroup(w, subscriptionID, resourceGroup)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	if resourceGroup == "" {
		respondAzureImplemented(w, path)
		return
	}

	serviceName := strings.TrimSpace(tail[0])
	if !azureSpringValidName(serviceName) {
		respondAzureInvalidRequest(w, path, "spring service name is invalid")
		return
	}

	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSpringGetService(w, path, subscriptionID, resourceGroup, serviceName, false)
		case http.MethodHead:
			s.handleAzureSpringGetService(w, path, subscriptionID, resourceGroup, serviceName, true)
		case http.MethodPut:
			s.handleAzureSpringCreateOrUpdateService(w, r, path, subscriptionID, resourceGroup, serviceName)
		case http.MethodPatch:
			s.handleAzureSpringPatchService(w, r, path, subscriptionID, resourceGroup, serviceName)
		case http.MethodDelete:
			s.handleAzureSpringDeleteService(w, subscriptionID, resourceGroup, serviceName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "enabletestendpoint":
		if r.Method == http.MethodPost {
			s.handleAzureSpringServiceAction(w, path, subscriptionID, resourceGroup, serviceName, "testEndpointEnabled", true)
			return
		}
	case "disabletestendpoint":
		if r.Method == http.MethodPost {
			s.handleAzureSpringServiceAction(w, path, subscriptionID, resourceGroup, serviceName, "testEndpointEnabled", false)
			return
		}
	case "disableapmglobally":
		if r.Method == http.MethodPost {
			s.handleAzureSpringServiceAction(w, path, subscriptionID, resourceGroup, serviceName, "apmGlobalEnabled", false)
			return
		}
	case "stop":
		if r.Method == http.MethodPost {
			s.handleAzureSpringStopService(w, path, subscriptionID, resourceGroup, serviceName)
			return
		}
	case "apps":
		s.handleAzureSpringAppsCollectionRoute(w, r, path, subscriptionID, resourceGroup, serviceName, tail[2:])
		return
	case "buildservices":
		s.handleAzureSpringBuildServicesRoute(w, r, path, subscriptionID, resourceGroup, serviceName, tail[2:])
		return
	default:
		if collection, ok := azureSpringCanonicalServiceChildCollection(tail[1]); ok {
			s.handleAzureSpringServiceChildRoute(w, r, path, subscriptionID, resourceGroup, serviceName, collection, tail[2:])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureSpringAppsCollectionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName string, tail []string) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureSpringListApps(w, serviceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	appName := strings.TrimSpace(tail[0])
	if !azureSpringValidName(appName) {
		respondAzureInvalidRequest(w, path, "spring app name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSpringGetApp(w, path, serviceID, appName, false)
		case http.MethodHead:
			s.handleAzureSpringGetApp(w, path, serviceID, appName, true)
		case http.MethodPut:
			s.handleAzureSpringCreateOrUpdateApp(w, r, path, subscriptionID, resourceGroup, serviceName, appName)
		case http.MethodPatch:
			s.handleAzureSpringPatchApp(w, r, path, serviceID, appName)
		case http.MethodDelete:
			s.handleAzureSpringDeleteApp(w, serviceID, appName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "setactivedeployments":
		if r.Method == http.MethodPost {
			s.handleAzureSpringSetActiveDeployment(w, r, path, serviceID, appName)
			return
		}
	case "deployments":
		s.handleAzureSpringDeploymentsRoute(w, r, path, subscriptionID, resourceGroup, serviceName, appName, tail[2:])
		return
	default:
		if collection, ok := azureSpringCanonicalAppChildCollection(tail[1]); ok {
			s.handleAzureSpringAppChildRoute(w, r, path, subscriptionID, resourceGroup, serviceName, appName, collection, tail[2:])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureSpringDeploymentsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName, appName string, tail []string) {
	appID := azureSpringAppID(subscriptionID, resourceGroup, serviceName, appName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureSpringListDeployments(w, appID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	deploymentName := strings.TrimSpace(tail[0])
	if !azureSpringValidName(deploymentName) {
		respondAzureInvalidRequest(w, path, "deployment name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSpringGetDeployment(w, path, appID, deploymentName, false)
		case http.MethodHead:
			s.handleAzureSpringGetDeployment(w, path, appID, deploymentName, true)
		case http.MethodPut:
			s.handleAzureSpringCreateOrUpdateDeployment(w, r, path, subscriptionID, resourceGroup, serviceName, appName, deploymentName)
		case http.MethodPatch:
			s.handleAzureSpringPatchDeployment(w, r, path, appID, deploymentName)
		case http.MethodDelete:
			s.handleAzureSpringDeleteDeployment(w, appID, deploymentName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "start":
		if r.Method == http.MethodPost {
			s.handleAzureSpringDeploymentAction(w, path, appID, deploymentName, "Running")
			return
		}
	case "stop":
		if r.Method == http.MethodPost {
			s.handleAzureSpringDeploymentAction(w, path, appID, deploymentName, "Stopped")
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureSpringAppChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName, appName, collection string, tail []string) {
	appID := azureSpringAppID(subscriptionID, resourceGroup, serviceName, appName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureSpringListChildren(w, appID, collection)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(tail[0])
	if !azureSpringValidName(name) {
		respondAzureInvalidRequest(w, path, collection+" name is invalid")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureSpringGetChild(w, path, appID, collection, name, false)
	case http.MethodHead:
		s.handleAzureSpringGetChild(w, path, appID, collection, name, true)
	case http.MethodPut:
		s.handleAzureSpringCreateOrUpdateChild(w, r, path, appID, collection, name, azureSpringServiceLocationForApp(s, appID), true)
	case http.MethodPatch:
		s.handleAzureSpringPatchChild(w, r, path, appID, collection, name)
	case http.MethodDelete:
		s.handleAzureSpringDeleteChild(w, appID, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureSpringServiceChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName, collection string, tail []string) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)
	if len(tail) == 0 {
		if collection == "containerRegistries" && r.Method == http.MethodPost {
			s.handleAzureSpringValidateContainerRegistry(w, r, path, serviceID)
			return
		}
		if r.Method == http.MethodGet {
			s.handleAzureSpringListChildren(w, serviceID, collection)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimSpace(tail[0])
	if !azureSpringValidName(name) {
		respondAzureInvalidRequest(w, path, collection+" name is invalid")
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "validate") {
		if collection == "configurationServices" && r.Method == http.MethodPost {
			s.handleAzureSpringValidateConfigurationService(w, path, serviceID, name)
			return
		}
		if collection == "containerRegistries" && r.Method == http.MethodPost {
			s.handleAzureSpringValidateContainerRegistry(w, r, path, serviceID)
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureSpringGetChild(w, path, serviceID, collection, name, false)
	case http.MethodHead:
		s.handleAzureSpringGetChild(w, path, serviceID, collection, name, true)
	case http.MethodPut:
		s.handleAzureSpringCreateOrUpdateChild(w, r, path, serviceID, collection, name, azureSpringServiceLocationForService(s, serviceID), false)
	case http.MethodPatch:
		s.handleAzureSpringPatchChild(w, r, path, serviceID, collection, name)
	case http.MethodDelete:
		s.handleAzureSpringDeleteChild(w, serviceID, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureSpringBuildServicesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName string, tail []string) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureSpringListBuildServices(w, serviceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	buildServiceName := strings.TrimSpace(tail[0])
	if !azureSpringValidName(buildServiceName) {
		respondAzureInvalidRequest(w, path, "build service name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSpringGetBuildService(w, path, serviceID, buildServiceName, false)
		case http.MethodHead:
			s.handleAzureSpringGetBuildService(w, path, serviceID, buildServiceName, true)
		case http.MethodPut:
			s.handleAzureSpringCreateOrUpdateBuildService(w, r, path, serviceID, buildServiceName, azureSpringServiceLocationForService(s, serviceID))
		case http.MethodPatch:
			s.handleAzureSpringPatchBuildService(w, r, path, serviceID, buildServiceName)
		case http.MethodDelete:
			s.handleAzureSpringDeleteBuildService(w, serviceID, buildServiceName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "builds":
		s.handleAzureSpringBuildsRoute(w, r, path, serviceID, buildServiceName, tail[2:])
		return
	case "buildresults":
		if r.Method == http.MethodGet && len(tail) == 2 {
			s.handleAzureSpringListBuildResults(w, path, serviceID, buildServiceName)
			return
		}
	case "supportedbuildpacks":
		if r.Method == http.MethodGet && len(tail) == 2 {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureSpringSupportedBuildpacksFixture()})
			return
		}
	case "supportedstacks":
		if r.Method == http.MethodGet && len(tail) == 2 {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureSpringSupportedStacksFixture()})
			return
		}
	case "builders":
		if len(tail) == 4 && strings.EqualFold(tail[3], "buildpackBindings") && r.Method == http.MethodGet {
			s.handleAzureSpringListBuildpackBindings(w, path, serviceID, buildServiceName, tail[2])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureSpringBuildsRoute(w http.ResponseWriter, r *http.Request, path, serviceID, buildServiceName string, tail []string) {
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureSpringListBuilds(w, buildServiceID)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	buildName := strings.TrimSpace(tail[0])
	if !azureSpringValidName(buildName) {
		respondAzureInvalidRequest(w, path, "build name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSpringGetBuild(w, path, buildServiceID, buildName, false)
		case http.MethodHead:
			s.handleAzureSpringGetBuild(w, path, buildServiceID, buildName, true)
		case http.MethodPut:
			s.handleAzureSpringCreateOrUpdateBuild(w, r, path, buildServiceID, buildName)
		case http.MethodPatch:
			s.handleAzureSpringPatchBuild(w, r, path, buildServiceID, buildName)
		case http.MethodDelete:
			s.handleAzureSpringDeleteBuild(w, buildServiceID, buildName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "results") && r.Method == http.MethodGet {
		s.handleAzureSpringListBuildResults(w, path, serviceID, buildServiceName)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureSpringCreateOrUpdateService(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureSpringString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	existing, exists := s.azureSpringServices[serviceID]
	service := &azureSpringService{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           serviceName,
		Location:       location,
		Tags:           azureSpringStringMap(payload["tags"]),
		Properties:     azureSpringNormalizeServiceProperties(payload["properties"]),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if exists {
		service.CreatedAt = existing.CreatedAt
	}
	service.ETag = azureSpringETag(serviceID, service.UpdatedAt)
	s.azureSpringServices[serviceID] = service

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}
	respondJSON(w, status, azureSpringServiceResponse(service))
}

func (s *Server) handleAzureSpringPatchService(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName string) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	service := s.azureSpringServices[serviceID]
	if service == nil {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}

	if tags, found := payload["tags"]; found {
		service.Tags = azureSpringStringMap(tags)
	}
	if properties, found := payload["properties"]; found {
		for key, value := range azureSpringNormalizeServiceProperties(properties) {
			service.Properties[key] = value
		}
	}
	service.UpdatedAt = time.Now().UTC()
	service.ETag = azureSpringETag(serviceID, service.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSpringServiceResponse(service))
}

func (s *Server) handleAzureSpringGetService(w http.ResponseWriter, path, subscriptionID, resourceGroup, serviceName string, head bool) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)

	s.providerStorageMu.Lock()
	service := s.azureSpringServices[serviceID]
	s.providerStorageMu.Unlock()

	if service == nil {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSpringServiceResponse(service))
}

func (s *Server) handleAzureSpringDeleteService(w http.ResponseWriter, subscriptionID, resourceGroup, serviceName string) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSpringServices[serviceID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSpringServices, serviceID)
	for key, app := range s.azureSpringApps {
		if app.ServiceID == serviceID {
			delete(s.azureSpringApps, key)
		}
	}
	for key, deployment := range s.azureSpringDeployments {
		if strings.HasPrefix(deployment.AppID, serviceID+"/apps/") {
			delete(s.azureSpringDeployments, key)
		}
	}
	for key, child := range s.azureSpringChildren {
		if child.ParentID == serviceID || strings.HasPrefix(child.ParentID, serviceID+"/apps/") {
			delete(s.azureSpringChildren, key)
		}
	}
	for key, buildService := range s.azureSpringBuildServices {
		if buildService.ServiceID == serviceID {
			delete(s.azureSpringBuildServices, key)
		}
	}
	for key, build := range s.azureSpringBuilds {
		if strings.HasPrefix(build.BuildServiceID, serviceID+"/buildServices/") {
			delete(s.azureSpringBuilds, key)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureSpringListServicesBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, service := range s.azureSpringServices {
		if service.SubscriptionID == subscriptionID {
			items = append(items, azureSpringServiceResponse(service))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringListServicesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, service := range s.azureSpringServices {
		if service.SubscriptionID == subscriptionID && strings.EqualFold(service.ResourceGroup, resourceGroup) {
			items = append(items, azureSpringServiceResponse(service))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringServiceAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, serviceName, property string, value any) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	service := s.azureSpringServices[serviceID]
	if service == nil {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}
	service.Properties[property] = value
	service.UpdatedAt = time.Now().UTC()
	service.ETag = azureSpringETag(serviceID, service.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSpringServiceResponse(service))
}

func (s *Server) handleAzureSpringStopService(w http.ResponseWriter, path, subscriptionID, resourceGroup, serviceName string) {
	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	service := s.azureSpringServices[serviceID]
	if service == nil {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}
	service.Properties["powerState"] = "Stopped"
	service.UpdatedAt = time.Now().UTC()
	service.ETag = azureSpringETag(serviceID, service.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSpringServiceResponse(service))
}

func (s *Server) handleAzureSpringCreateOrUpdateApp(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName, appName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}

	serviceID := azureSpringServiceID(subscriptionID, resourceGroup, serviceName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	service := s.azureSpringServices[serviceID]
	if service == nil {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}

	appID := azureSpringAppID(subscriptionID, resourceGroup, serviceName, appName)
	existing, exists := s.azureSpringApps[appID]
	app := &azureSpringApp{
		ServiceID:  serviceID,
		Name:       appName,
		Location:   azureSpringFirstNonEmptyString(azureSpringString(payload["location"]), service.Location),
		Tags:       azureSpringStringMap(payload["tags"]),
		Properties: azureSpringNormalizeAppProperties(payload["properties"]),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if exists {
		app.CreatedAt = existing.CreatedAt
		app.ActiveDeployment = existing.ActiveDeployment
	}
	app.ETag = azureSpringETag(appID, app.UpdatedAt)
	s.azureSpringApps[appID] = app

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}
	respondJSON(w, status, azureSpringAppResponse(subscriptionID, resourceGroup, serviceName, app))
}

func (s *Server) handleAzureSpringPatchApp(w http.ResponseWriter, r *http.Request, path, serviceID, appName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}

	appID := serviceID + "/apps/" + appName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	app := s.azureSpringApps[appID]
	if app == nil {
		respondAzureSpringAppsNotFound(w, path, "spring app not found")
		return
	}
	if tags, found := payload["tags"]; found {
		app.Tags = azureSpringStringMap(tags)
	}
	if properties, found := payload["properties"]; found {
		for key, value := range azureSpringNormalizeAppProperties(properties) {
			app.Properties[key] = value
		}
	}
	app.UpdatedAt = time.Now().UTC()
	app.ETag = azureSpringETag(appID, app.UpdatedAt)
	subscriptionID, resourceGroup, serviceName := azureSpringParseServiceID(serviceID)
	respondJSON(w, http.StatusOK, azureSpringAppResponse(subscriptionID, resourceGroup, serviceName, app))
}

func (s *Server) handleAzureSpringGetApp(w http.ResponseWriter, path, serviceID, appName string, head bool) {
	appID := serviceID + "/apps/" + appName

	s.providerStorageMu.Lock()
	app := s.azureSpringApps[appID]
	s.providerStorageMu.Unlock()

	if app == nil {
		respondAzureSpringAppsNotFound(w, path, "spring app not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	subscriptionID, resourceGroup, serviceName := azureSpringParseServiceID(serviceID)
	respondJSON(w, http.StatusOK, azureSpringAppResponse(subscriptionID, resourceGroup, serviceName, app))
}

func (s *Server) handleAzureSpringDeleteApp(w http.ResponseWriter, serviceID, appName string) {
	appID := serviceID + "/apps/" + appName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSpringApps[appID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSpringApps, appID)
	for key, deployment := range s.azureSpringDeployments {
		if deployment.AppID == appID {
			delete(s.azureSpringDeployments, key)
		}
	}
	for key, child := range s.azureSpringChildren {
		if child.ParentID == appID {
			delete(s.azureSpringChildren, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureSpringListApps(w http.ResponseWriter, serviceID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	subscriptionID, resourceGroup, serviceName := azureSpringParseServiceID(serviceID)
	items := make([]map[string]any, 0)
	for _, app := range s.azureSpringApps {
		if app.ServiceID == serviceID {
			items = append(items, azureSpringAppResponse(subscriptionID, resourceGroup, serviceName, app))
		}
	}
	sort.Slice(items, func(i, j int) bool { return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringSetActiveDeployment(w http.ResponseWriter, r *http.Request, path, serviceID, appName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	appID := serviceID + "/apps/" + appName

	active := strings.TrimSpace(azureSpringString(payload["deploymentName"]))
	if active == "" {
		if names, ok := payload["activeDeploymentNames"].([]any); ok && len(names) > 0 {
			active = strings.TrimSpace(azureSpringString(names[0]))
		}
	}
	if active == "" {
		respondAzureInvalidRequest(w, path, "active deployment name is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	app := s.azureSpringApps[appID]
	if app == nil {
		respondAzureSpringAppsNotFound(w, path, "spring app not found")
		return
	}
	deploymentID := appID + "/deployments/" + active
	if s.azureSpringDeployments[deploymentID] == nil {
		respondAzureSpringAppsNotFound(w, path, "deployment not found")
		return
	}
	app.ActiveDeployment = active
	app.Properties["activeDeploymentName"] = active
	app.UpdatedAt = time.Now().UTC()
	app.ETag = azureSpringETag(appID, app.UpdatedAt)
	subscriptionID, resourceGroup, serviceName := azureSpringParseServiceID(serviceID)
	respondJSON(w, http.StatusOK, azureSpringAppResponse(subscriptionID, resourceGroup, serviceName, app))
}

func (s *Server) handleAzureSpringCreateOrUpdateDeployment(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, serviceName, appName, deploymentName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	appID := azureSpringAppID(subscriptionID, resourceGroup, serviceName, appName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	app := s.azureSpringApps[appID]
	if app == nil {
		respondAzureSpringAppsNotFound(w, path, "spring app not found")
		return
	}

	deploymentID := appID + "/deployments/" + deploymentName
	existing, exists := s.azureSpringDeployments[deploymentID]
	properties := azureSpringNormalizeDeploymentProperties(payload["properties"])
	deployment := &azureSpringDeployment{
		AppID:      appID,
		Name:       deploymentName,
		Location:   azureSpringFirstNonEmptyString(azureSpringString(payload["location"]), app.Location),
		Tags:       azureSpringStringMap(payload["tags"]),
		Properties: properties,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if exists {
		deployment.CreatedAt = existing.CreatedAt
	}
	deployment.ETag = azureSpringETag(deploymentID, deployment.UpdatedAt)
	s.azureSpringDeployments[deploymentID] = deployment
	if !exists && strings.TrimSpace(app.ActiveDeployment) == "" {
		app.ActiveDeployment = deploymentName
		app.Properties["activeDeploymentName"] = deploymentName
	}

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}
	respondJSON(w, status, azureSpringDeploymentResponse(subscriptionID, resourceGroup, serviceName, appName, deployment))
}

func (s *Server) handleAzureSpringPatchDeployment(w http.ResponseWriter, r *http.Request, path, appID, deploymentName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	deploymentID := appID + "/deployments/" + deploymentName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	deployment := s.azureSpringDeployments[deploymentID]
	if deployment == nil {
		respondAzureSpringAppsNotFound(w, path, "deployment not found")
		return
	}
	if tags, found := payload["tags"]; found {
		deployment.Tags = azureSpringStringMap(tags)
	}
	if properties, found := payload["properties"]; found {
		for key, value := range azureSpringNormalizeDeploymentProperties(properties) {
			deployment.Properties[key] = value
		}
	}
	deployment.UpdatedAt = time.Now().UTC()
	deployment.ETag = azureSpringETag(deploymentID, deployment.UpdatedAt)
	subscriptionID, resourceGroup, serviceName, appName := azureSpringParseAppID(appID)
	respondJSON(w, http.StatusOK, azureSpringDeploymentResponse(subscriptionID, resourceGroup, serviceName, appName, deployment))
}

func (s *Server) handleAzureSpringGetDeployment(w http.ResponseWriter, path, appID, deploymentName string, head bool) {
	deploymentID := appID + "/deployments/" + deploymentName

	s.providerStorageMu.Lock()
	deployment := s.azureSpringDeployments[deploymentID]
	s.providerStorageMu.Unlock()

	if deployment == nil {
		respondAzureSpringAppsNotFound(w, path, "deployment not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	subscriptionID, resourceGroup, serviceName, appName := azureSpringParseAppID(appID)
	respondJSON(w, http.StatusOK, azureSpringDeploymentResponse(subscriptionID, resourceGroup, serviceName, appName, deployment))
}

func (s *Server) handleAzureSpringDeleteDeployment(w http.ResponseWriter, appID, deploymentName string) {
	deploymentID := appID + "/deployments/" + deploymentName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSpringDeployments[deploymentID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSpringDeployments, deploymentID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureSpringListDeployments(w http.ResponseWriter, appID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	subscriptionID, resourceGroup, serviceName, appName := azureSpringParseAppID(appID)
	items := make([]map[string]any, 0)
	for _, deployment := range s.azureSpringDeployments {
		if deployment.AppID == appID {
			items = append(items, azureSpringDeploymentResponse(subscriptionID, resourceGroup, serviceName, appName, deployment))
		}
	}
	sort.Slice(items, func(i, j int) bool { return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringDeploymentAction(w http.ResponseWriter, path, appID, deploymentName, state string) {
	deploymentID := appID + "/deployments/" + deploymentName

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	deployment := s.azureSpringDeployments[deploymentID]
	if deployment == nil {
		respondAzureSpringAppsNotFound(w, path, "deployment not found")
		return
	}
	deployment.Properties["status"] = state
	deployment.UpdatedAt = time.Now().UTC()
	deployment.ETag = azureSpringETag(deploymentID, deployment.UpdatedAt)
	subscriptionID, resourceGroup, serviceName, appName := azureSpringParseAppID(appID)
	respondJSON(w, http.StatusOK, azureSpringDeploymentResponse(subscriptionID, resourceGroup, serviceName, appName, deployment))
}

func (s *Server) handleAzureSpringCreateOrUpdateChild(w http.ResponseWriter, r *http.Request, path, parentID, collection, name, defaultLocation string, requireResourceID bool) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureSpringNormalizeChildProperties(collection, payload["properties"])
	if collection == "bindings" && strings.TrimSpace(azureSpringString(properties["resourceId"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.resourceId is required")
		return
	}
	childID := azureSpringChildID(parentID, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if !azureSpringParentExists(s, parentID) {
		respondAzureSpringAppsNotFound(w, path, "parent resource not found")
		return
	}
	existing, exists := s.azureSpringChildren[childID]
	child := &azureSpringChildResource{
		ParentID:   parentID,
		Collection: collection,
		Name:       name,
		Location:   azureSpringFirstNonEmptyString(azureSpringString(payload["location"]), defaultLocation),
		Tags:       azureSpringStringMap(payload["tags"]),
		Properties: properties,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if exists {
		child.CreatedAt = existing.CreatedAt
	}
	child.ETag = azureSpringETag(childID, child.UpdatedAt)
	s.azureSpringChildren[childID] = child

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}
	respondJSON(w, status, azureSpringChildResponse(child))
}

func (s *Server) handleAzureSpringPatchChild(w http.ResponseWriter, r *http.Request, path, parentID, collection, name string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	childID := azureSpringChildID(parentID, collection, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	child := s.azureSpringChildren[childID]
	if child == nil {
		respondAzureSpringAppsNotFound(w, path, collection+" resource not found")
		return
	}
	if tags, found := payload["tags"]; found {
		child.Tags = azureSpringStringMap(tags)
	}
	if properties, found := payload["properties"]; found {
		for key, value := range azureSpringNormalizeChildProperties(collection, properties) {
			child.Properties[key] = value
		}
	}
	child.UpdatedAt = time.Now().UTC()
	child.ETag = azureSpringETag(childID, child.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSpringChildResponse(child))
}

func (s *Server) handleAzureSpringGetChild(w http.ResponseWriter, path, parentID, collection, name string, head bool) {
	childID := azureSpringChildID(parentID, collection, name)

	s.providerStorageMu.Lock()
	child := s.azureSpringChildren[childID]
	s.providerStorageMu.Unlock()

	if child == nil {
		respondAzureSpringAppsNotFound(w, path, collection+" resource not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSpringChildResponse(child))
}

func (s *Server) handleAzureSpringDeleteChild(w http.ResponseWriter, parentID, collection, name string) {
	childID := azureSpringChildID(parentID, collection, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSpringChildren[childID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSpringChildren, childID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureSpringListChildren(w http.ResponseWriter, parentID, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, child := range s.azureSpringChildren {
		if child.ParentID == parentID && child.Collection == collection {
			items = append(items, azureSpringChildResponse(child))
		}
	}
	sort.Slice(items, func(i, j int) bool { return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringValidateConfigurationService(w http.ResponseWriter, path, serviceID, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !azureSpringParentExists(s, serviceID) {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":          name,
		"isValid":       true,
		"message":       "configuration service validated successfully",
		"provider":      providerAzure,
		"validatedType": "configurationServices",
	})
}

func (s *Server) handleAzureSpringValidateContainerRegistry(w http.ResponseWriter, r *http.Request, path, serviceID string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if !azureSpringParentExists(s, serviceID) {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}
	server := strings.TrimSpace(azureSpringStringMapAny(payload["properties"])["server"].(string))
	if server == "" {
		server = strings.TrimSpace(azureSpringString(payload["server"]))
	}
	if server == "" {
		server = "registry.azurecr.io"
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"isValid":       true,
		"message":       "container registry validated successfully",
		"validatedType": "containerRegistries",
		"server":        server,
	})
}

func (s *Server) handleAzureSpringCreateOrUpdateBuildService(w http.ResponseWriter, r *http.Request, path, serviceID, buildServiceName, defaultLocation string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if !azureSpringParentExists(s, serviceID) {
		respondAzureSpringAppsNotFound(w, path, "spring service not found")
		return
	}
	existing, exists := s.azureSpringBuildServices[buildServiceID]
	buildService := &azureSpringBuildService{
		ServiceID:  serviceID,
		Name:       buildServiceName,
		Location:   azureSpringFirstNonEmptyString(azureSpringString(payload["location"]), defaultLocation),
		Tags:       azureSpringStringMap(payload["tags"]),
		Properties: azureSpringNormalizeBuildServiceProperties(payload["properties"]),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if exists {
		buildService.CreatedAt = existing.CreatedAt
	}
	buildService.ETag = azureSpringETag(buildServiceID, buildService.UpdatedAt)
	s.azureSpringBuildServices[buildServiceID] = buildService

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}
	respondJSON(w, status, azureSpringBuildServiceResponse(buildService))
}

func (s *Server) handleAzureSpringPatchBuildService(w http.ResponseWriter, r *http.Request, path, serviceID, buildServiceName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	buildService := s.azureSpringBuildServices[buildServiceID]
	if buildService == nil {
		respondAzureSpringAppsNotFound(w, path, "build service not found")
		return
	}
	if tags, found := payload["tags"]; found {
		buildService.Tags = azureSpringStringMap(tags)
	}
	if properties, found := payload["properties"]; found {
		for key, value := range azureSpringNormalizeBuildServiceProperties(properties) {
			buildService.Properties[key] = value
		}
	}
	buildService.UpdatedAt = time.Now().UTC()
	buildService.ETag = azureSpringETag(buildServiceID, buildService.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSpringBuildServiceResponse(buildService))
}

func (s *Server) handleAzureSpringGetBuildService(w http.ResponseWriter, path, serviceID, buildServiceName string, head bool) {
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)

	s.providerStorageMu.Lock()
	buildService := s.azureSpringBuildServices[buildServiceID]
	s.providerStorageMu.Unlock()

	if buildService == nil {
		respondAzureSpringAppsNotFound(w, path, "build service not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSpringBuildServiceResponse(buildService))
}

func (s *Server) handleAzureSpringDeleteBuildService(w http.ResponseWriter, serviceID, buildServiceName string) {
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSpringBuildServices[buildServiceID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSpringBuildServices, buildServiceID)
	for key, build := range s.azureSpringBuilds {
		if build.BuildServiceID == buildServiceID {
			delete(s.azureSpringBuilds, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureSpringListBuildServices(w http.ResponseWriter, serviceID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, buildService := range s.azureSpringBuildServices {
		if buildService.ServiceID == serviceID {
			items = append(items, azureSpringBuildServiceResponse(buildService))
		}
	}
	sort.Slice(items, func(i, j int) bool { return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringCreateOrUpdateBuild(w http.ResponseWriter, r *http.Request, path, buildServiceID, buildName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	buildID := azureSpringBuildIDByBuildServiceID(buildServiceID, buildName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if s.azureSpringBuildServices[buildServiceID] == nil {
		respondAzureSpringAppsNotFound(w, path, "build service not found")
		return
	}

	existing, exists := s.azureSpringBuilds[buildID]
	build := &azureSpringBuild{
		BuildServiceID: buildServiceID,
		Name:           buildName,
		Properties:     azureSpringNormalizeBuildProperties(payload["properties"]),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if exists {
		build.CreatedAt = existing.CreatedAt
	}
	build.ETag = azureSpringETag(buildID, build.UpdatedAt)
	s.azureSpringBuilds[buildID] = build

	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	}
	respondJSON(w, status, azureSpringBuildResponse(build))
}

func (s *Server) handleAzureSpringPatchBuild(w http.ResponseWriter, r *http.Request, path, buildServiceID, buildName string) {
	_, payload, ok := azureSpringReadRequestBody(w, r, path)
	if !ok {
		return
	}
	buildID := azureSpringBuildIDByBuildServiceID(buildServiceID, buildName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	build := s.azureSpringBuilds[buildID]
	if build == nil {
		respondAzureSpringAppsNotFound(w, path, "build not found")
		return
	}
	if properties, found := payload["properties"]; found {
		for key, value := range azureSpringNormalizeBuildProperties(properties) {
			build.Properties[key] = value
		}
	}
	build.UpdatedAt = time.Now().UTC()
	build.ETag = azureSpringETag(buildID, build.UpdatedAt)
	respondJSON(w, http.StatusOK, azureSpringBuildResponse(build))
}

func (s *Server) handleAzureSpringGetBuild(w http.ResponseWriter, path, buildServiceID, buildName string, head bool) {
	buildID := azureSpringBuildIDByBuildServiceID(buildServiceID, buildName)

	s.providerStorageMu.Lock()
	build := s.azureSpringBuilds[buildID]
	s.providerStorageMu.Unlock()

	if build == nil {
		respondAzureSpringAppsNotFound(w, path, "build not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSpringBuildResponse(build))
}

func (s *Server) handleAzureSpringDeleteBuild(w http.ResponseWriter, buildServiceID, buildName string) {
	buildID := azureSpringBuildIDByBuildServiceID(buildServiceID, buildName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureSpringBuilds[buildID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureSpringBuilds, buildID)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureSpringListBuilds(w http.ResponseWriter, buildServiceID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, build := range s.azureSpringBuilds {
		if build.BuildServiceID == buildServiceID {
			items = append(items, azureSpringBuildResponse(build))
		}
	}
	sort.Slice(items, func(i, j int) bool { return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringListBuildResults(w http.ResponseWriter, path, serviceID, buildServiceName string) {
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if s.azureSpringBuildServices[buildServiceID] == nil {
		respondAzureSpringAppsNotFound(w, path, "build service not found")
		return
	}

	items := make([]map[string]any, 0)
	for _, build := range s.azureSpringBuilds {
		if build.BuildServiceID == buildServiceID {
			items = append(items, map[string]any{
				"id":           azureSpringBuildIDByBuildServiceID(buildServiceID, build.Name) + "/results/default",
				"name":         "default",
				"type":         "Microsoft.AppPlatform/Spring/buildServices/buildResults",
				"provisioning": "Succeeded",
				"properties": map[string]any{
					"buildName":    build.Name,
					"status":       "Succeeded",
					"artifactName": build.Name + ".jar",
					"logFileUrl":   "https://stackyard.local/build-logs/" + build.Name,
				},
			})
		}
	}
	sort.Slice(items, func(i, j int) bool { return azureSpringString(items[i]["id"]) < azureSpringString(items[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSpringListBuildpackBindings(w http.ResponseWriter, path, serviceID, buildServiceName, builderName string) {
	buildServiceID := azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if s.azureSpringBuildServices[buildServiceID] == nil {
		respondAzureSpringAppsNotFound(w, path, "build service not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"id":   buildServiceID + "/builders/" + builderName + "/buildpackBindings/default",
				"name": "default",
				"type": "Microsoft.AppPlatform/Spring/buildServices/builders/buildpackBindings",
				"properties": map[string]any{
					"builderName":     builderName,
					"bindingType":     "ApplicationInsights",
					"resourceBinding": "default",
				},
			},
		},
	})
}

func azureSpringServiceResponse(service *azureSpringService) map[string]any {
	id := azureSpringServiceID(service.SubscriptionID, service.ResourceGroup, service.Name)
	return map[string]any{
		"id":         id,
		"name":       service.Name,
		"type":       "Microsoft.AppPlatform/Spring",
		"location":   service.Location,
		"tags":       azureSpringAnyStringMap(service.Tags),
		"etag":       service.ETag,
		"systemData": azureSpringSystemData(service.CreatedAt, service.UpdatedAt),
		"properties": azureSpringCloneAnyMap(service.Properties),
	}
}

func azureSpringAppResponse(subscriptionID, resourceGroup, serviceName string, app *azureSpringApp) map[string]any {
	id := azureSpringAppID(subscriptionID, resourceGroup, serviceName, app.Name)
	props := azureSpringCloneAnyMap(app.Properties)
	if strings.TrimSpace(app.ActiveDeployment) != "" {
		props["activeDeploymentName"] = app.ActiveDeployment
	}
	return map[string]any{
		"id":         id,
		"name":       app.Name,
		"type":       "Microsoft.AppPlatform/Spring/apps",
		"location":   app.Location,
		"tags":       azureSpringAnyStringMap(app.Tags),
		"etag":       app.ETag,
		"systemData": azureSpringSystemData(app.CreatedAt, app.UpdatedAt),
		"properties": props,
	}
}

func azureSpringDeploymentResponse(subscriptionID, resourceGroup, serviceName, appName string, deployment *azureSpringDeployment) map[string]any {
	id := azureSpringDeploymentID(subscriptionID, resourceGroup, serviceName, appName, deployment.Name)
	return map[string]any{
		"id":         id,
		"name":       deployment.Name,
		"type":       "Microsoft.AppPlatform/Spring/apps/deployments",
		"location":   deployment.Location,
		"tags":       azureSpringAnyStringMap(deployment.Tags),
		"etag":       deployment.ETag,
		"systemData": azureSpringSystemData(deployment.CreatedAt, deployment.UpdatedAt),
		"properties": azureSpringCloneAnyMap(deployment.Properties),
	}
}

func azureSpringChildResponse(child *azureSpringChildResource) map[string]any {
	return map[string]any{
		"id":         azureSpringChildID(child.ParentID, child.Collection, child.Name),
		"name":       child.Name,
		"type":       "Microsoft.AppPlatform/Spring/" + child.Collection,
		"location":   child.Location,
		"tags":       azureSpringAnyStringMap(child.Tags),
		"etag":       child.ETag,
		"systemData": azureSpringSystemData(child.CreatedAt, child.UpdatedAt),
		"properties": azureSpringCloneAnyMap(child.Properties),
	}
}

func azureSpringBuildServiceResponse(buildService *azureSpringBuildService) map[string]any {
	return map[string]any{
		"id":         azureSpringBuildServiceIDByServiceID(buildService.ServiceID, buildService.Name),
		"name":       buildService.Name,
		"type":       "Microsoft.AppPlatform/Spring/buildServices",
		"location":   buildService.Location,
		"tags":       azureSpringAnyStringMap(buildService.Tags),
		"etag":       buildService.ETag,
		"systemData": azureSpringSystemData(buildService.CreatedAt, buildService.UpdatedAt),
		"properties": azureSpringCloneAnyMap(buildService.Properties),
	}
}

func azureSpringBuildResponse(build *azureSpringBuild) map[string]any {
	return map[string]any{
		"id":         azureSpringBuildIDByBuildServiceID(build.BuildServiceID, build.Name),
		"name":       build.Name,
		"type":       "Microsoft.AppPlatform/Spring/buildServices/builds",
		"etag":       build.ETag,
		"systemData": azureSpringSystemData(build.CreatedAt, build.UpdatedAt),
		"properties": azureSpringCloneAnyMap(build.Properties),
	}
}

func azureSpringAppsOperationsFixture() []map[string]any {
	operations := []struct {
		name        string
		resource    string
		operation   string
		description string
	}{
		{"Microsoft.AppPlatform/Spring/read", "Spring", "Get Spring Service", "Get a Spring Apps service"},
		{"Microsoft.AppPlatform/Spring/write", "Spring", "Create Or Update Spring Service", "Create or update a Spring Apps service"},
		{"Microsoft.AppPlatform/Spring/delete", "Spring", "Delete Spring Service", "Delete a Spring Apps service"},
		{"Microsoft.AppPlatform/Spring/apps/read", "Apps", "Get App", "Get an app resource"},
		{"Microsoft.AppPlatform/Spring/apps/write", "Apps", "Create Or Update App", "Create or update an app resource"},
		{"Microsoft.AppPlatform/Spring/apps/delete", "Apps", "Delete App", "Delete an app resource"},
		{"Microsoft.AppPlatform/Spring/apps/deployments/read", "Deployments", "Get Deployment", "Get a deployment resource"},
		{"Microsoft.AppPlatform/Spring/apps/deployments/write", "Deployments", "Create Or Update Deployment", "Create or update a deployment resource"},
		{"Microsoft.AppPlatform/Spring/apps/deployments/delete", "Deployments", "Delete Deployment", "Delete a deployment resource"},
		{"Microsoft.AppPlatform/Spring/apps/bindings/read", "Bindings", "Get Binding", "Get a binding resource"},
		{"Microsoft.AppPlatform/Spring/apps/bindings/write", "Bindings", "Create Or Update Binding", "Create or update a binding resource"},
		{"Microsoft.AppPlatform/Spring/apps/customDomains/read", "Custom Domains", "Get Custom Domain", "Get a custom domain resource"},
		{"Microsoft.AppPlatform/Spring/apps/customDomains/write", "Custom Domains", "Create Or Update Custom Domain", "Create or update a custom domain resource"},
		{"Microsoft.AppPlatform/Spring/certificates/read", "Certificates", "Get Certificate", "Get a certificate resource"},
		{"Microsoft.AppPlatform/Spring/certificates/write", "Certificates", "Create Or Update Certificate", "Create or update a certificate resource"},
		{"Microsoft.AppPlatform/Spring/containerRegistries/write", "Container Registries", "Create Or Update Container Registry", "Create or update a container registry resource"},
		{"Microsoft.AppPlatform/Spring/buildServices/write", "Build Services", "Create Or Update Build Service", "Create or update a build service resource"},
		{"Microsoft.AppPlatform/Spring/buildServices/builds/write", "Builds", "Create Or Update Build", "Create or update a build resource"},
	}
	value := make([]map[string]any, 0, len(operations))
	for _, operation := range operations {
		value = append(value, map[string]any{
			"name":         operation.name,
			"isDataAction": false,
			"origin":       "user,system",
			"display": map[string]any{
				"provider":    "Microsoft App Platform",
				"resource":    operation.resource,
				"operation":   operation.operation,
				"description": operation.description,
			},
		})
	}
	return value
}

func azureSpringSupportedBuildpacksFixture() []map[string]any {
	return []map[string]any{
		{"id": "java", "name": "paketo-buildpacks/java", "displayName": "Paketo Java"},
		{"id": "nodejs", "name": "paketo-buildpacks/nodejs", "displayName": "Paketo NodeJS"},
	}
}

func azureSpringSupportedStacksFixture() []map[string]any {
	return []map[string]any{
		{"id": "jammy-base", "displayName": "Jammy Base", "version": "latest"},
		{"id": "jammy-tiny", "displayName": "Jammy Tiny", "version": "latest"},
	}
}

func azureSpringNormalizeServiceProperties(raw any) map[string]any {
	props := azureSpringStringMapAny(raw)
	out := azureSpringCloneAnyMap(props)
	if out["provisioningState"] == nil {
		out["provisioningState"] = "Succeeded"
	}
	if out["powerState"] == nil {
		out["powerState"] = "Running"
	}
	if out["fqdn"] == nil {
		out["fqdn"] = "springapps.stackyard.local"
	}
	if out["version"] == nil {
		out["version"] = "Enterprise"
	}
	if out["testEndpointEnabled"] == nil {
		out["testEndpointEnabled"] = false
	}
	if out["apmGlobalEnabled"] == nil {
		out["apmGlobalEnabled"] = true
	}
	return out
}

func azureSpringNormalizeAppProperties(raw any) map[string]any {
	props := azureSpringStringMapAny(raw)
	out := azureSpringCloneAnyMap(props)
	if out["provisioningState"] == nil {
		out["provisioningState"] = "Succeeded"
	}
	if out["public"] == nil {
		out["public"] = true
	}
	if out["httpsOnly"] == nil {
		out["httpsOnly"] = true
	}
	return out
}

func azureSpringNormalizeDeploymentProperties(raw any) map[string]any {
	props := azureSpringStringMapAny(raw)
	out := azureSpringCloneAnyMap(props)
	if out["provisioningState"] == nil {
		out["provisioningState"] = "Succeeded"
	}
	if out["status"] == nil {
		out["status"] = "Running"
	}
	if out["instances"] == nil {
		out["instances"] = 1
	}
	if out["source"] == nil {
		out["source"] = map[string]any{"type": "Jar", "relativePath": "app.jar"}
	}
	return out
}

func azureSpringNormalizeChildProperties(collection string, raw any) map[string]any {
	props := azureSpringStringMapAny(raw)
	out := azureSpringCloneAnyMap(props)
	if out["provisioningState"] == nil {
		out["provisioningState"] = "Succeeded"
	}
	switch collection {
	case "bindings":
		if out["bindingParameters"] == nil {
			out["bindingParameters"] = map[string]any{"type": "default"}
		}
	case "domains":
		if out["thumbprint"] == nil {
			out["thumbprint"] = "STACKYARDTHUMBPRINT"
		}
		if out["certName"] == nil {
			out["certName"] = "cert-a"
		}
	case "certificates":
		if out["issuer"] == nil {
			out["issuer"] = "Stackyard CA"
		}
	case "apms":
		if out["type"] == nil {
			out["type"] = "ApplicationInsights"
		}
	case "gateways":
		if out["public"] == nil {
			out["public"] = true
		}
	case "apiPortals":
		if out["url"] == nil {
			out["url"] = "https://api-portal.stackyard.local"
		}
	case "applicationLiveViews":
		if out["components"] == nil {
			out["components"] = []any{"gateway", "apps"}
		}
	case "configurationServices":
		if out["generation"] == nil {
			out["generation"] = "Gen2"
		}
	case "serviceRegistries":
		if out["registryType"] == nil {
			out["registryType"] = "Eureka"
		}
	case "devToolPortals":
		if out["features"] == nil {
			out["features"] = []any{"appLiveView", "applicationAccelerator"}
		}
	case "containerRegistries":
		if out["server"] == nil {
			out["server"] = "registry.azurecr.io"
		}
	case "storages":
		if out["storageType"] == nil {
			out["storageType"] = "AzureFileVolume"
		}
	}
	return out
}

func azureSpringNormalizeBuildServiceProperties(raw any) map[string]any {
	props := azureSpringStringMapAny(raw)
	out := azureSpringCloneAnyMap(props)
	if out["provisioningState"] == nil {
		out["provisioningState"] = "Succeeded"
	}
	if out["resourceRequests"] == nil {
		out["resourceRequests"] = map[string]any{"cpu": "1", "memory": "2Gi"}
	}
	return out
}

func azureSpringNormalizeBuildProperties(raw any) map[string]any {
	props := azureSpringStringMapAny(raw)
	out := azureSpringCloneAnyMap(props)
	if out["provisioningState"] == nil {
		out["provisioningState"] = "Succeeded"
	}
	if out["status"] == nil {
		out["status"] = "Succeeded"
	}
	if out["builderName"] == nil {
		out["builderName"] = "default"
	}
	if out["agentPool"] == nil {
		out["agentPool"] = "default"
	}
	return out
}

func azureSpringValidName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 64 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func azureSpringCanonicalServiceChildCollection(value string) (string, bool) {
	collection, ok := azureSpringServiceChildCollections[strings.ToLower(strings.TrimSpace(value))]
	return collection, ok
}

func azureSpringCanonicalAppChildCollection(value string) (string, bool) {
	collection, ok := azureSpringAppChildCollections[strings.ToLower(strings.TrimSpace(value))]
	return collection, ok
}

func azureSpringServiceID(subscriptionID, resourceGroup, serviceName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AppPlatform/Spring/%s", subscriptionID, resourceGroup, serviceName)
}

func azureSpringAppID(subscriptionID, resourceGroup, serviceName, appName string) string {
	return azureSpringServiceID(subscriptionID, resourceGroup, serviceName) + "/apps/" + appName
}

func azureSpringDeploymentID(subscriptionID, resourceGroup, serviceName, appName, deploymentName string) string {
	return azureSpringAppID(subscriptionID, resourceGroup, serviceName, appName) + "/deployments/" + deploymentName
}

func azureSpringChildID(parentID, collection, name string) string {
	return parentID + "/" + collection + "/" + name
}

func azureSpringBuildServiceIDByServiceID(serviceID, buildServiceName string) string {
	return serviceID + "/buildServices/" + buildServiceName
}

func azureSpringBuildIDByBuildServiceID(buildServiceID, buildName string) string {
	return buildServiceID + "/builds/" + buildName
}

func azureSpringParseServiceID(serviceID string) (subscriptionID, resourceGroup, serviceName string) {
	segments := splitPathSegments(strings.TrimPrefix(serviceID, "/"))
	if len(segments) >= 8 {
		return segments[1], segments[3], segments[7]
	}
	return "", "", ""
}

func azureSpringParseAppID(appID string) (subscriptionID, resourceGroup, serviceName, appName string) {
	segments := splitPathSegments(strings.TrimPrefix(appID, "/"))
	if len(segments) >= 10 {
		return segments[1], segments[3], segments[7], segments[9]
	}
	return "", "", "", ""
}

func azureSpringParentExists(s *Server, parentID string) bool {
	if s.azureSpringServices[parentID] != nil {
		return true
	}
	if s.azureSpringApps[parentID] != nil {
		return true
	}
	return false
}

func azureSpringServiceLocationForService(s *Server, serviceID string) string {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if service := s.azureSpringServices[serviceID]; service != nil {
		return service.Location
	}
	return ""
}

func azureSpringServiceLocationForApp(s *Server, appID string) string {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if app := s.azureSpringApps[appID]; app != nil {
		return app.Location
	}
	return ""
}

func azureSpringETag(seed string, updatedAt time.Time) string {
	sum := sha1.Sum([]byte(seed + "|" + updatedAt.UTC().Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureSpringSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureSpringReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	defer r.Body.Close()
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
		return body, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureSpringString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func azureSpringStringMap(value any) map[string]string {
	out := map[string]string{}
	raw, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for key, item := range raw {
		text := strings.TrimSpace(azureSpringString(item))
		if text != "" {
			out[key] = text
		}
	}
	return out
}

func azureSpringStringMapAny(value any) map[string]any {
	out := map[string]any{}
	raw, ok := value.(map[string]any)
	if !ok {
		return out
	}
	for key, item := range raw {
		out[key] = item
	}
	return out
}

func azureSpringAnyStringMap(value map[string]string) map[string]any {
	out := map[string]any{}
	for key, item := range value {
		out[key] = item
	}
	return out
}

func azureSpringCloneAnyMap(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		switch typed := item.(type) {
		case map[string]any:
			out[key] = azureSpringCloneAnyMap(typed)
		case []any:
			out[key] = append([]any(nil), typed...)
		default:
			out[key] = item
		}
	}
	return out
}

func azureSpringFirstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func respondAzureSpringAppsNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerAzure,
		"path":     path,
	})
}
