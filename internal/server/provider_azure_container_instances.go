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

const azureContainerInstancesAPIVersion = "2025-09-01"

type azureContainerInstancesResource struct {
	SubscriptionID string
	ResourceGroup  string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Identity       map[string]any
	Zones          []string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureContainerInstancesProfileRevision struct {
	Revision  int
	Location  string
	Tags      map[string]string
	Identity  map[string]any
	Zones     []string
	Properties map[string]any
	CreatedAt time.Time
}

type azureContainerInstancesProfileResource struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Identity       map[string]any
	Zones          []string
	Properties     map[string]any
	Revision       int
	Revisions      []*azureContainerInstancesProfileRevision
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureContainerInstancesServiceAssociationLink struct {
	SubnetID  string
	DeletedAt time.Time
}

func (s *Server) handleAzureContainerInstancesRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureContainerInstancesIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureContainerInstancesAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ContainerInstance") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerInstancesOperationsFixture()})
		return true
	}

	subscriptionID, resourceGroup, prefixSegments, tail, ok := azureContainerInstancesRouteContext(segments)
	if !ok || subscriptionID == "" || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations":
		s.handleAzureContainerInstancesLocationRoute(w, r, path, subscriptionID, tail)
	case "containergroups":
		s.handleAzureContainerInstancesResources(w, r, path, subscriptionID, resourceGroup, "containerGroups", tail)
	case "containergroupprofiles":
		s.handleAzureContainerInstancesProfiles(w, r, path, subscriptionID, resourceGroup, tail)
	case "ngroups":
		s.handleAzureContainerInstancesResources(w, r, path, subscriptionID, resourceGroup, "ngroups", tail)
	case "serviceassociationlinks":
		s.handleAzureContainerInstancesServiceAssociationLinks(w, r, path, prefixSegments, tail)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func azureContainerInstancesIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ContainerInstance") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, _, tail, ok := azureContainerInstancesRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations", "containergroups", "containergroupprofiles", "ngroups", "serviceassociationlinks":
		return true
	default:
		return false
	}
}

func azureContainerInstancesAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureContainerInstancesRouteContext(segments []string) (subscriptionID, resourceGroup string, prefixSegments, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.ContainerInstance") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[:idx], segments[idx+2:], true
	}
	return "", "", nil, nil, false
}

func (s *Server) handleAzureContainerInstancesLocationRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) != 3 {
		respondAzureImplemented(w, path)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	location := strings.TrimSpace(tail[1])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "cachedimages":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerInstancesCachedImagesFixture(subscriptionID, location)})
	case "capabilities":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerInstancesCapabilitiesFixture(subscriptionID, location)})
	case "usages":
		respondJSON(w, http.StatusOK, map[string]any{"value": azureContainerInstancesUsagesFixture(subscriptionID, location)})
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureContainerInstancesResources(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureContainerInstancesListResources(w, subscriptionID, resourceGroup, collection)
		return
	}

	if resourceGroup == "" {
		respondAzureInvalidRequest(w, path, "resourceGroupName is required")
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "resource name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureContainerInstancesGetResource(w, path, subscriptionID, resourceGroup, collection, name, false)
		case http.MethodHead:
			s.handleAzureContainerInstancesGetResource(w, path, subscriptionID, resourceGroup, collection, name, true)
		case http.MethodPut:
			s.handleAzureContainerInstancesPutResource(w, r, path, subscriptionID, resourceGroup, collection, name)
		case http.MethodPatch:
			s.handleAzureContainerInstancesPatchResource(w, r, path, subscriptionID, resourceGroup, collection, name)
		case http.MethodDelete:
			s.handleAzureContainerInstancesDeleteResource(w, subscriptionID, resourceGroup, collection, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "start", "stop", "restart":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureContainerInstancesAction(w, path, subscriptionID, resourceGroup, collection, name, strings.ToLower(strings.TrimSpace(tail[2])))
	case "outboundnetworkdependenciesendpoints":
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureContainerInstancesOutboundDependencies(w, path, subscriptionID, resourceGroup, collection, name)
	case "containers":
		s.handleAzureContainerInstancesContainerChildRoute(w, r, path, subscriptionID, resourceGroup, collection, name, tail[3:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureContainerInstancesProfiles(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureContainerInstancesListProfiles(w, subscriptionID, resourceGroup)
		return
	}

	if resourceGroup == "" {
		respondAzureInvalidRequest(w, path, "resourceGroupName is required")
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "containerGroupProfileName is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureContainerInstancesGetProfile(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureContainerInstancesGetProfile(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureContainerInstancesPutProfile(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureContainerInstancesPatchProfile(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureContainerInstancesDeleteProfile(w, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 3 && strings.EqualFold(tail[2], "revisions") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureContainerInstancesListProfileRevisions(w, path, subscriptionID, resourceGroup, name)
		return
	}
	if len(tail) == 4 && strings.EqualFold(tail[2], "revisions") {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureContainerInstancesGetProfileRevision(w, path, subscriptionID, resourceGroup, name, tail[3], r.Method == http.MethodHead)
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureContainerInstancesServiceAssociationLinks(w http.ResponseWriter, r *http.Request, path string, prefixSegments, tail []string) {
	if len(prefixSegments) == 0 || len(tail) != 2 || !strings.EqualFold(tail[1], "default") {
		respondAzureImplemented(w, path)
		return
	}
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	subnetID := "/" + strings.Join(prefixSegments, "/")
	s.providerStorageMu.Lock()
	s.azureContainerInstancesServiceAssociationLinks[azureContainerInstancesServiceAssociationKey(subnetID)] = &azureContainerInstancesServiceAssociationLink{
		SubnetID:  subnetID,
		DeletedAt: time.Now().UTC(),
	}
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureContainerInstancesListResources(w http.ResponseWriter, subscriptionID, resourceGroup, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, resource := range s.azureContainerInstancesResources {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.Collection, collection) {
			continue
		}
		if resourceGroup != "" && !strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			continue
		}
		items = append(items, azureContainerInstancesResourceResponse(resource))
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureContainerInstancesString(items[i]["name"])), strings.ToLower(azureContainerInstancesString(items[j]["name"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureContainerInstancesGetResource(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureContainerInstancesResources[azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", collection+" resource was not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureContainerInstancesResourceResponse(resource))
}

func (s *Server) handleAzureContainerInstancesPutResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	_, payload, ok := azureContainerInstancesReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureContainerInstancesString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties, _ := payload["properties"].(map[string]any)
	if collection == "containerGroups" {
		if !azureContainerInstancesValidateContainerProperties(properties) {
			respondAzureInvalidRequest(w, path, "properties.containers and properties.osType are required")
			return
		}
	}

	now := time.Now().UTC()
	key := azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name)
	s.providerStorageMu.Lock()
	resource := s.azureContainerInstancesResources[key]
	created := resource == nil
	if resource == nil {
		resource = &azureContainerInstancesResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Collection:     collection,
			Name:           name,
			CreatedAt:      now,
		}
	}
	resource.Location = location
	resource.Tags = azureContainerInstancesStringMap(payload["tags"])
	resource.Identity = azureContainerInstancesCloneMap(payload["identity"])
	resource.Zones = azureContainerInstancesStringSlice(payload["zones"])
	resource.Properties = azureContainerInstancesNormalizeResourceProperties(collection, resource.Name, azureContainerInstancesCloneMap(properties), resource.Properties)
	resource.UpdatedAt = now
	resource.ETag = azureContainerInstancesETag(collection, subscriptionID, resourceGroup, name, now.Format(time.RFC3339Nano))
	s.azureContainerInstancesResources[key] = resource
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureContainerInstancesResourceResponse(resource))
}

func (s *Server) handleAzureContainerInstancesPatchResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	_, payload, ok := azureContainerInstancesReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	resource := s.azureContainerInstancesResources[azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name)]
	if resource == nil {
		s.providerStorageMu.Unlock()
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", collection+" resource was not found")
		return
	}
	if value, ok := payload["location"]; ok {
		location := strings.TrimSpace(azureContainerInstancesString(value))
		if location == "" {
			s.providerStorageMu.Unlock()
			respondAzureInvalidRequest(w, path, "location must not be empty")
			return
		}
		resource.Location = location
	}
	if value, ok := payload["tags"]; ok {
		resource.Tags = azureContainerInstancesStringMap(value)
	}
	if value, ok := payload["zones"]; ok {
		resource.Zones = azureContainerInstancesStringSlice(value)
	}
	if value, ok := payload["properties"]; ok {
		if props, ok := value.(map[string]any); ok && len(props) > 0 {
			merged := azureContainerInstancesCloneMap(resource.Properties)
			for key, item := range props {
				merged[key] = item
			}
			resource.Properties = azureContainerInstancesNormalizeResourceProperties(collection, resource.Name, merged, resource.Properties)
		}
	}
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureContainerInstancesETag(collection, subscriptionID, resourceGroup, name, resource.UpdatedAt.Format(time.RFC3339Nano))
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureContainerInstancesResourceResponse(resource))
}

func (s *Server) handleAzureContainerInstancesDeleteResource(w http.ResponseWriter, subscriptionID, resourceGroup, collection, name string) {
	s.providerStorageMu.Lock()
	key := azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name)
	_, existed := s.azureContainerInstancesResources[key]
	delete(s.azureContainerInstancesResources, key)
	s.providerStorageMu.Unlock()
	if !existed {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureContainerInstancesAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name, action string) {
	s.providerStorageMu.Lock()
	resource := s.azureContainerInstancesResources[azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name)]
	if resource == nil {
		s.providerStorageMu.Unlock()
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", collection+" resource was not found")
		return
	}
	resource.Properties = azureContainerInstancesNormalizeActionState(resource.Properties, action)
	resource.UpdatedAt = time.Now().UTC()
	resource.ETag = azureContainerInstancesETag(collection, subscriptionID, resourceGroup, name, resource.UpdatedAt.Format(time.RFC3339Nano), action)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureContainerInstancesOutboundDependencies(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureContainerInstancesResources[azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", collection+" resource was not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": []any{}})
}

func (s *Server) handleAzureContainerInstancesContainerChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, groupName string, tail []string) {
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	containerName := strings.TrimSpace(tail[0])
	if containerName == "" {
		respondAzureInvalidRequest(w, path, "containerName is required")
		return
	}
	action := strings.ToLower(strings.TrimSpace(tail[1]))

	s.providerStorageMu.Lock()
	resource := s.azureContainerInstancesResources[azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, groupName)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", collection+" resource was not found")
		return
	}
	container, ok := azureContainerInstancesFindContainer(resource.Properties, containerName)
	if !ok {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", "container was not found")
		return
	}

	switch action {
	case "logs":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, azureContainerInstancesLogsFixture(groupName, container, r.URL.Query().Get("tail"), r.URL.Query().Get("timestamps")))
	case "attach":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, _, ok := azureContainerInstancesReadRequestBody(w, r, path)
		if !ok {
			return
		}
		respondJSON(w, http.StatusOK, azureContainerInstancesAttachFixture(subscriptionID, resourceGroup, groupName, containerName))
	case "exec", "executecommand":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, payload, ok := azureContainerInstancesReadRequestBody(w, r, path)
		if !ok {
			return
		}
		command := strings.TrimSpace(azureContainerInstancesString(payload["command"]))
		if command == "" {
			respondAzureInvalidRequest(w, path, "command is required")
			return
		}
		respondJSON(w, http.StatusOK, azureContainerInstancesExecFixture(subscriptionID, resourceGroup, groupName, containerName, command))
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureContainerInstancesListProfiles(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, profile := range s.azureContainerInstancesProfileResources {
		if !strings.EqualFold(profile.SubscriptionID, subscriptionID) {
			continue
		}
		if resourceGroup != "" && !strings.EqualFold(profile.ResourceGroup, resourceGroup) {
			continue
		}
		items = append(items, azureContainerInstancesProfileResponse(profile))
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureContainerInstancesString(items[i]["name"])), strings.ToLower(azureContainerInstancesString(items[j]["name"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureContainerInstancesGetProfile(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	profile := s.azureContainerInstancesProfileResources[azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if profile == nil {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", "containerGroupProfile was not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureContainerInstancesProfileResponse(profile))
}

func (s *Server) handleAzureContainerInstancesPutProfile(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureContainerInstancesReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureContainerInstancesString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties, _ := payload["properties"].(map[string]any)
	if !azureContainerInstancesValidateContainerProperties(properties) {
		respondAzureInvalidRequest(w, path, "properties.containers and properties.osType are required")
		return
	}

	now := time.Now().UTC()
	key := azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	profile := s.azureContainerInstancesProfileResources[key]
	created := profile == nil
	if profile == nil {
		profile = &azureContainerInstancesProfileResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
			Revision:       0,
		}
	}
	profile.Location = location
	profile.Tags = azureContainerInstancesStringMap(payload["tags"])
	profile.Identity = azureContainerInstancesCloneMap(payload["identity"])
	profile.Zones = azureContainerInstancesStringSlice(payload["zones"])
	profile.Properties = azureContainerInstancesNormalizeProfileProperties(azureContainerInstancesCloneMap(properties))
	profile.Revision++
	profile.Revisions = append(profile.Revisions, &azureContainerInstancesProfileRevision{
		Revision:  profile.Revision,
		Location:  profile.Location,
		Tags:      azureContainerInstancesCopyStringMap(profile.Tags),
		Identity:  azureContainerInstancesCloneMap(profile.Identity),
		Zones:     append([]string(nil), profile.Zones...),
		Properties: azureContainerInstancesCloneMap(profile.Properties),
		CreatedAt: now,
	})
	profile.UpdatedAt = now
	profile.ETag = azureContainerInstancesETag("containerGroupProfiles", subscriptionID, resourceGroup, name, strconv.Itoa(profile.Revision), now.Format(time.RFC3339Nano))
	s.azureContainerInstancesProfileResources[key] = profile
	s.providerStorageMu.Unlock()

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureContainerInstancesProfileResponse(profile))
}

func (s *Server) handleAzureContainerInstancesPatchProfile(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureContainerInstancesReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	profile := s.azureContainerInstancesProfileResources[azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name)]
	if profile == nil {
		s.providerStorageMu.Unlock()
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", "containerGroupProfile was not found")
		return
	}
	if value, ok := payload["location"]; ok {
		location := strings.TrimSpace(azureContainerInstancesString(value))
		if location == "" {
			s.providerStorageMu.Unlock()
			respondAzureInvalidRequest(w, path, "location must not be empty")
			return
		}
		profile.Location = location
	}
	if value, ok := payload["tags"]; ok {
		profile.Tags = azureContainerInstancesStringMap(value)
	}
	if value, ok := payload["zones"]; ok {
		profile.Zones = azureContainerInstancesStringSlice(value)
	}
	profile.UpdatedAt = time.Now().UTC()
	profile.ETag = azureContainerInstancesETag("containerGroupProfiles", subscriptionID, resourceGroup, name, strconv.Itoa(profile.Revision), profile.UpdatedAt.Format(time.RFC3339Nano))
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureContainerInstancesProfileResponse(profile))
}

func (s *Server) handleAzureContainerInstancesDeleteProfile(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	key := azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name)
	_, existed := s.azureContainerInstancesProfileResources[key]
	delete(s.azureContainerInstancesProfileResources, key)
	s.providerStorageMu.Unlock()
	if !existed {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureContainerInstancesListProfileRevisions(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	profile := s.azureContainerInstancesProfileResources[azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if profile == nil {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", "containerGroupProfile was not found")
		return
	}
	items := make([]map[string]any, 0, len(profile.Revisions))
	for _, revision := range profile.Revisions {
		items = append(items, azureContainerInstancesProfileRevisionResponse(profile, revision))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureContainerInstancesGetProfileRevision(w http.ResponseWriter, path, subscriptionID, resourceGroup, name, revisionText string, headOnly bool) {
	revision, err := strconv.Atoi(strings.TrimSpace(revisionText))
	if err != nil || revision <= 0 {
		respondAzureInvalidRequest(w, path, "revisionNumber is invalid")
		return
	}
	s.providerStorageMu.Lock()
	profile := s.azureContainerInstancesProfileResources[azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if profile == nil {
		azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", "containerGroupProfile was not found")
		return
	}
	for _, item := range profile.Revisions {
		if item.Revision == revision {
			if headOnly {
				w.WriteHeader(http.StatusOK)
				return
			}
			respondJSON(w, http.StatusOK, azureContainerInstancesProfileRevisionResponse(profile, item))
			return
		}
	}
	azureContainerInstancesRespondError(w, http.StatusNotFound, "ResourceNotFound", "profile revision was not found")
}

func azureContainerInstancesOperationsFixture() []map[string]any {
	names := []string{
		"Microsoft.ContainerInstance/containerGroups/read",
		"Microsoft.ContainerInstance/containerGroups/write",
		"Microsoft.ContainerInstance/containerGroups/delete",
		"Microsoft.ContainerInstance/containerGroups/start/action",
		"Microsoft.ContainerInstance/containerGroups/stop/action",
		"Microsoft.ContainerInstance/containerGroups/restart/action",
		"Microsoft.ContainerInstance/containerGroups/outboundNetworkDependenciesEndpoints/action",
		"Microsoft.ContainerInstance/containers/logs/read",
		"Microsoft.ContainerInstance/containers/attach/action",
		"Microsoft.ContainerInstance/containers/exec/action",
		"Microsoft.ContainerInstance/containerGroupProfiles/read",
		"Microsoft.ContainerInstance/containerGroupProfiles/write",
		"Microsoft.ContainerInstance/containerGroupProfiles/delete",
		"Microsoft.ContainerInstance/locations/cachedImages/read",
		"Microsoft.ContainerInstance/locations/capabilities/read",
		"Microsoft.ContainerInstance/locations/usages/read",
		"Microsoft.ContainerInstance/serviceAssociationLinks/delete",
		"Microsoft.ContainerInstance/ngroups/read",
		"Microsoft.ContainerInstance/ngroups/write",
		"Microsoft.ContainerInstance/ngroups/delete",
		"Microsoft.ContainerInstance/ngroups/start/action",
		"Microsoft.ContainerInstance/ngroups/stop/action",
		"Microsoft.ContainerInstance/ngroups/restart/action",
	}
	items := make([]map[string]any, 0, len(names))
	for _, name := range names {
		items = append(items, map[string]any{
			"name": name,
			"id":   "/providers/Microsoft.ContainerInstance/operations/" + strings.ReplaceAll(strings.ToLower(name), "/", "_"),
			"display": map[string]any{
				"provider":  "Microsoft Container Instance",
				"resource":  "Container Instances",
				"operation": azureContainerInstancesOperationVerb(name),
			},
			"origin": "user,system",
		})
	}
	return items
}

func azureContainerInstancesCachedImagesFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"osType":            "Linux",
			"image":             "mcr.microsoft.com/azuredocs/aci-helloworld",
			"lastUpdatedTime":   time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339),
			"resourceGroupName": "stackyard-system",
			"subscriptionId":    subscriptionID,
			"location":          location,
		},
		{
			"osType":            "Windows",
			"image":             "mcr.microsoft.com/windows/servercore:ltsc2022",
			"lastUpdatedTime":   time.Now().UTC().Add(-12 * time.Hour).Format(time.RFC3339),
			"resourceGroupName": "stackyard-system",
			"subscriptionId":    subscriptionID,
			"location":          location,
		},
	}
}

func azureContainerInstancesCapabilitiesFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"location":       location,
			"subscriptionId": subscriptionID,
			"osType":         "Linux",
			"capabilities": []map[string]any{
				{"cpu": 1, "memoryInGB": 1.5, "gpu": "None"},
				{"cpu": 2, "memoryInGB": 4.0, "gpu": "None"},
			},
		},
		{
			"location":       location,
			"subscriptionId": subscriptionID,
			"osType":         "Windows",
			"capabilities": []map[string]any{
				{"cpu": 2, "memoryInGB": 8.0, "gpu": "None"},
			},
		},
	}
}

func azureContainerInstancesUsagesFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"name": map[string]any{
				"value":          "ContainerGroups",
				"localizedValue": "Container Groups",
			},
			"unit":          "Count",
			"currentValue":  4,
			"limit":         100,
			"subscriptionId": subscriptionID,
			"location":      location,
		},
		{
			"name": map[string]any{
				"value":          "Cores",
				"localizedValue": "Cores",
			},
			"unit":          "Count",
			"currentValue":  6,
			"limit":         100,
			"subscriptionId": subscriptionID,
			"location":      location,
		},
	}
}

func azureContainerInstancesResourceResponse(resource *azureContainerInstancesResource) map[string]any {
	resp := map[string]any{
		"id":         azureContainerInstancesResourceID(resource.SubscriptionID, resource.ResourceGroup, resource.Collection, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ContainerInstance/" + resource.Collection,
		"location":   resource.Location,
		"tags":       azureContainerInstancesAnyStringMap(resource.Tags),
		"properties": azureContainerInstancesCloneMap(resource.Properties),
		"eTag":       resource.ETag,
	}
	if len(resource.Identity) > 0 {
		resp["identity"] = azureContainerInstancesCloneMap(resource.Identity)
	}
	if len(resource.Zones) > 0 {
		resp["zones"] = append([]string(nil), resource.Zones...)
	}
	return resp
}

func azureContainerInstancesProfileResponse(profile *azureContainerInstancesProfileResource) map[string]any {
	resp := map[string]any{
		"id":       azureContainerInstancesProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name),
		"name":     profile.Name,
		"type":     "Microsoft.ContainerInstance/containerGroupProfiles",
		"location": profile.Location,
		"tags":     azureContainerInstancesAnyStringMap(profile.Tags),
		"properties": azureContainerInstancesMergeMaps(
			azureContainerInstancesCloneMap(profile.Properties),
			map[string]any{"revision": profile.Revision, "provisioningState": "Succeeded"},
		),
		"eTag": profile.ETag,
	}
	if len(profile.Identity) > 0 {
		resp["identity"] = azureContainerInstancesCloneMap(profile.Identity)
	}
	if len(profile.Zones) > 0 {
		resp["zones"] = append([]string(nil), profile.Zones...)
	}
	return resp
}

func azureContainerInstancesProfileRevisionResponse(profile *azureContainerInstancesProfileResource, revision *azureContainerInstancesProfileRevision) map[string]any {
	resp := map[string]any{
		"id":       azureContainerInstancesProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name) + "/revisions/" + strconv.Itoa(revision.Revision),
		"name":     strconv.Itoa(revision.Revision),
		"type":     "Microsoft.ContainerInstance/containerGroupProfiles/revisions",
		"location": revision.Location,
		"tags":     azureContainerInstancesAnyStringMap(revision.Tags),
		"properties": azureContainerInstancesMergeMaps(
			azureContainerInstancesCloneMap(revision.Properties),
			map[string]any{"revision": revision.Revision, "provisioningState": "Succeeded"},
		),
	}
	if len(revision.Identity) > 0 {
		resp["identity"] = azureContainerInstancesCloneMap(revision.Identity)
	}
	if len(revision.Zones) > 0 {
		resp["zones"] = append([]string(nil), revision.Zones...)
	}
	return resp
}

func azureContainerInstancesLogsFixture(groupName string, container map[string]any, tail, timestamps string) map[string]any {
	containerName := azureContainerInstancesString(container["name"])
	lines := []string{
		"stackyard container started",
		"group=" + groupName,
		"container=" + containerName,
	}
	if n, err := strconv.Atoi(strings.TrimSpace(tail)); err == nil && n > 0 && n < len(lines) {
		lines = lines[len(lines)-n:]
	}
	if strings.EqualFold(strings.TrimSpace(timestamps), "true") {
		now := time.Now().UTC()
		for idx := range lines {
			lines[idx] = now.Add(time.Duration(idx) * time.Second).Format(time.RFC3339) + " " + lines[idx]
		}
	}
	return map[string]any{
		"content": strings.Join(lines, "\n"),
	}
}

func azureContainerInstancesAttachFixture(subscriptionID, resourceGroup, groupName, containerName string) map[string]any {
	return map[string]any{
		"webSocketUri": "wss://stackyard.local/aci/attach/" + azureContainerInstancesStableToken(subscriptionID, resourceGroup, groupName, containerName),
		"password":     "pwd-" + azureContainerInstancesStableToken(groupName, containerName),
	}
}

func azureContainerInstancesExecFixture(subscriptionID, resourceGroup, groupName, containerName, command string) map[string]any {
	return map[string]any{
		"webSocketUri": "wss://stackyard.local/aci/exec/" + azureContainerInstancesStableToken(subscriptionID, resourceGroup, groupName, containerName, command),
		"password":     "pwd-" + azureContainerInstancesStableToken(command, containerName),
	}
}

func azureContainerInstancesValidateContainerProperties(properties map[string]any) bool {
	if strings.TrimSpace(azureContainerInstancesString(properties["osType"])) == "" {
		return false
	}
	containers, ok := properties["containers"].([]any)
	if !ok || len(containers) == 0 {
		return false
	}
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			return false
		}
		if strings.TrimSpace(azureContainerInstancesString(container["name"])) == "" {
			return false
		}
		props, _ := container["properties"].(map[string]any)
		if strings.TrimSpace(azureContainerInstancesString(props["image"])) == "" {
			return false
		}
	}
	return true
}

func azureContainerInstancesNormalizeResourceProperties(collection, name string, properties, previous map[string]any) map[string]any {
	normalized := azureContainerInstancesCloneMap(properties)
	state := "Running"
	if previousState := azureContainerInstancesState(previous); previousState != "" {
		state = previousState
	}
	if collection == "ngroups" {
		if previousState := azureContainerInstancesState(previous); previousState != "" {
			state = previousState
		}
	}
	normalized["provisioningState"] = "Succeeded"
	normalized["instanceView"] = map[string]any{
		"state":      state,
		"events":     []any{},
		"startTime":  time.Now().UTC().Format(time.RFC3339),
		"restartCount": 0,
	}
	if _, ok := normalized["isCreatedFromStandbyPool"]; !ok && collection == "containerGroups" {
		normalized["isCreatedFromStandbyPool"] = false
	}
	if containers, ok := normalized["containers"].([]any); ok {
		normalized["containers"] = azureContainerInstancesNormalizeContainers(containers, state, name)
	}
	return normalized
}

func azureContainerInstancesNormalizeProfileProperties(properties map[string]any) map[string]any {
	normalized := azureContainerInstancesCloneMap(properties)
	normalized["provisioningState"] = "Succeeded"
	if containers, ok := normalized["containers"].([]any); ok {
		normalized["containers"] = azureContainerInstancesNormalizeContainers(containers, "Template", "")
	}
	return normalized
}

func azureContainerInstancesNormalizeContainers(containers []any, state, groupName string) []any {
	normalized := make([]any, 0, len(containers))
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		cloned := azureContainerInstancesCloneMap(container)
		props, _ := cloned["properties"].(map[string]any)
		props = azureContainerInstancesCloneMap(props)
		props["instanceView"] = map[string]any{
			"currentState": map[string]any{
				"state":  state,
				"detailStatus": state,
			},
			"events": []any{},
		}
		if strings.TrimSpace(groupName) != "" {
			props["command"] = []string{"echo", "stackyard", groupName}
		}
		cloned["properties"] = props
		normalized = append(normalized, cloned)
	}
	return normalized
}

func azureContainerInstancesNormalizeActionState(properties map[string]any, action string) map[string]any {
	normalized := azureContainerInstancesCloneMap(properties)
	state := "Running"
	switch action {
	case "stop":
		state = "Stopped"
	case "start", "restart":
		state = "Running"
	}
	instanceView, _ := normalized["instanceView"].(map[string]any)
	instanceView = azureContainerInstancesCloneMap(instanceView)
	instanceView["state"] = state
	instanceView["lastAction"] = action
	instanceView["lastActionTime"] = time.Now().UTC().Format(time.RFC3339)
	normalized["instanceView"] = instanceView
	if containers, ok := normalized["containers"].([]any); ok {
		normalized["containers"] = azureContainerInstancesNormalizeContainers(containers, state, "")
	}
	return normalized
}

func azureContainerInstancesFindContainer(properties map[string]any, containerName string) (map[string]any, bool) {
	containers, ok := properties["containers"].([]any)
	if !ok {
		return nil, false
	}
	for _, item := range containers {
		container, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(azureContainerInstancesString(container["name"])), containerName) {
			return container, true
		}
	}
	return nil, false
}

func azureContainerInstancesOperationVerb(name string) string {
	if strings.Contains(name, "/delete") {
		return "Delete"
	}
	if strings.Contains(name, "/write") {
		return "Write"
	}
	if strings.Contains(name, "/action") {
		return "Action"
	}
	return "Read"
}

func azureContainerInstancesReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if strings.TrimSpace(string(body)) == "" {
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

func azureContainerInstancesRespondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func azureContainerInstancesResourceKey(subscriptionID, resourceGroup, collection, name string) string {
	return strings.ToLower(subscriptionID + "::" + resourceGroup + "::" + collection + "::" + name)
}

func azureContainerInstancesProfileKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "::" + resourceGroup + "::containergroupprofiles::" + name)
}

func azureContainerInstancesServiceAssociationKey(subnetID string) string {
	return strings.ToLower(strings.TrimSpace(subnetID))
}

func azureContainerInstancesResourceID(subscriptionID, resourceGroup, collection, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerInstance/%s/%s", subscriptionID, resourceGroup, collection, name)
}

func azureContainerInstancesProfileID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ContainerInstance/containerGroupProfiles/%s", subscriptionID, resourceGroup, name)
}

func azureContainerInstancesCloneMap(value any) map[string]any {
	source, ok := value.(map[string]any)
	if !ok || len(source) == 0 {
		return map[string]any{}
	}
	raw, err := json.Marshal(source)
	if err != nil {
		out := make(map[string]any, len(source))
		for key, item := range source {
			out[key] = item
		}
		return out
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		out = map[string]any{}
	}
	return out
}

func azureContainerInstancesString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func azureContainerInstancesStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if stringsValue, ok := value.([]string); ok {
			return append([]string(nil), stringsValue...)
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(azureContainerInstancesString(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func azureContainerInstancesCopyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func azureContainerInstancesAnyStringMap(src map[string]string) map[string]any {
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func azureContainerInstancesStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	if existing, ok := value.(map[string]string); ok {
		return azureContainerInstancesCopyStringMap(existing)
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, item := range raw {
		out[key] = azureContainerInstancesString(item)
	}
	return out
}

func azureContainerInstancesState(properties map[string]any) string {
	instanceView, _ := properties["instanceView"].(map[string]any)
	return strings.TrimSpace(azureContainerInstancesString(instanceView["state"]))
}

func azureContainerInstancesMergeMaps(base, overlay map[string]any) map[string]any {
	out := azureContainerInstancesCloneMap(base)
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func azureContainerInstancesETag(parts ...string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, strings.Join(parts, "|"))
	return `"` + hex.EncodeToString(h.Sum(nil))[:24] + `"`
}

func azureContainerInstancesStableToken(parts ...string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, strings.ToLower(strings.Join(parts, "|")))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
