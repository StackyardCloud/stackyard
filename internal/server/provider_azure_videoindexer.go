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

const azureVideoIndexerAPIVersion = "2025-04-01"

type azureVideoIndexerAccount struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Identity       map[string]any
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureVideoIndexerPrivateEndpointConnection struct {
	SubscriptionID string
	ResourceGroup  string
	AccountName    string
	Name           string
	Location       string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

func (s *Server) handleAzureVideoIndexerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsVideoIndexerPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureVideoIndexerAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.VideoIndexer") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureVideoIndexerOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureVideoIndexerRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "checkNameAvailability"):
		if r.Method == http.MethodPost {
			s.handleAzureVideoIndexerCheckNameAvailability(w, r, path, subscriptionID)
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	case len(tail) >= 1 && strings.EqualFold(tail[0], "accounts"):
		s.handleAzureVideoIndexerAccountsRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		return false
	}
}

func azureIsVideoIndexerPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.VideoIndexer") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureVideoIndexerRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability", "accounts":
		return true
	default:
		return false
	}
}

func azureVideoIndexerAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureVideoIndexerRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
		for idx := 4; idx+1 < len(segments); idx++ {
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.VideoIndexer") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.VideoIndexer") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureVideoIndexerCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_ = subscriptionID
	_, payload, ok := azureVideoIndexerReadRequestBody(w, r, path)
	if !ok {
		return
	}
	name := strings.TrimSpace(azureVideoIndexerString(payload["name"]))
	typ := strings.TrimSpace(azureVideoIndexerString(payload["type"]))
	if name == "" || typ == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	if !strings.EqualFold(typ, "Microsoft.VideoIndexer/accounts") {
		respondAzureInvalidRequest(w, path, "type must be Microsoft.VideoIndexer/accounts")
		return
	}
	if !azureVideoIndexerValidAccountName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "account name must be 3-50 characters of letters, digits, or hyphens",
		})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, account := range s.azureVideoIndexerAccounts {
		if strings.EqualFold(account.Name, name) {
			respondJSON(w, http.StatusOK, map[string]any{
				"nameAvailable": false,
				"reason":        "AlreadyExists",
				"message":       "account name is already in use",
			})
			return
		}
	}
	if strings.EqualFold(name, "reserved") {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "account name is reserved",
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *Server) handleAzureVideoIndexerAccountsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureVideoIndexerListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureVideoIndexerListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureVideoIndexerValidAccountName(name) {
		respondAzureInvalidRequest(w, path, "account name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureVideoIndexerGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureVideoIndexerGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureVideoIndexerCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureVideoIndexerPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureVideoIndexerDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 3 {
		switch strings.ToLower(strings.TrimSpace(tail[2])) {
		case "generateaccesstoken":
			if r.Method == http.MethodPost {
				s.handleAzureVideoIndexerGenerateToken(w, r, path, subscriptionID, resourceGroup, name, "access")
				return
			}
		case "generaterestrictedvieweraccesstoken":
			if r.Method == http.MethodPost {
				s.handleAzureVideoIndexerGenerateToken(w, r, path, subscriptionID, resourceGroup, name, "restricted_viewer")
				return
			}
		case "generateextensionaccesstoken":
			if r.Method == http.MethodPost {
				s.handleAzureVideoIndexerGenerateToken(w, r, path, subscriptionID, resourceGroup, name, "extension_access")
				return
			}
		case "generateextensionrestrictedvieweraccesstoken":
			if r.Method == http.MethodPost {
				s.handleAzureVideoIndexerGenerateToken(w, r, path, subscriptionID, resourceGroup, name, "extension_restricted_viewer")
				return
			}
		case "privateendpointconnections":
			s.handleAzureVideoIndexerPrivateEndpointConnectionsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
			return
		case "privatelinkresources":
			s.handleAzureVideoIndexerPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
			return
		}
	}
	if strings.EqualFold(tail[2], "privateEndpointConnections") {
		s.handleAzureVideoIndexerPrivateEndpointConnectionsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	}
	if strings.EqualFold(tail[2], "privateLinkResources") {
		s.handleAzureVideoIndexerPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureVideoIndexerListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureVideoIndexerEnsureSeeded(subscriptionID, "rg-videoindexer")
	items := make([]map[string]any, 0)
	for _, account := range s.azureVideoIndexerAccounts {
		if strings.EqualFold(account.SubscriptionID, subscriptionID) {
			items = append(items, azureVideoIndexerAccountFixture(account))
		}
	}
	azureVideoIndexerSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureVideoIndexerListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureVideoIndexerEnsureSeeded(subscriptionID, resourceGroup)
	items := make([]map[string]any, 0)
	for _, account := range s.azureVideoIndexerAccounts {
		if strings.EqualFold(account.SubscriptionID, subscriptionID) && strings.EqualFold(account.ResourceGroup, resourceGroup) {
			items = append(items, azureVideoIndexerAccountFixture(account))
		}
	}
	azureVideoIndexerSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureVideoIndexerGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureVideoIndexerAccountFixture(account))
}

func (s *Server) handleAzureVideoIndexerCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureVideoIndexerReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureVideoIndexerString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	now := time.Now().UTC()
	key := azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name)
	account, exists := s.azureVideoIndexerAccounts[key]
	if !exists {
		account = &azureVideoIndexerAccount{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
		}
	}
	account.Location = location
	account.Identity = azureVideoIndexerCloneAnyMap(azureVideoIndexerObject(payload["identity"]))
	account.Tags = azureVideoIndexerStringMap(payload["tags"])
	account.Properties = azureVideoIndexerCloneAnyMap(azureVideoIndexerObject(payload["properties"]))
	if account.Properties == nil {
		account.Properties = map[string]any{}
	}
	account.Properties["provisioningState"] = "Succeeded"
	if _, ok := account.Properties["accountId"]; !ok {
		account.Properties["accountId"] = "avi-" + name
	}
	if _, ok := account.Properties["endpoint"]; !ok {
		account.Properties["endpoint"] = "https://" + name + ".videoindexer.ai"
	}
	if _, ok := account.Properties["storageServices"]; !ok {
		account.Properties["storageServices"] = map[string]any{"current": "ManagedByMicrosoft"}
	}
	account.UpdatedAt = now
	account.ETag = azureVideoIndexerETag(key, now)
	s.azureVideoIndexerAccounts[key] = account

	s.azureVideoIndexerEnsureDefaultPrivateEndpointConnection(account)

	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureVideoIndexerAccountFixture(account))
}

func (s *Server) handleAzureVideoIndexerPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureVideoIndexerReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", name))
		return
	}
	if tags, ok := payload["tags"]; ok {
		account.Tags = azureVideoIndexerStringMap(tags)
	}
	if identity, ok := payload["identity"]; ok {
		account.Identity = azureVideoIndexerCloneAnyMap(azureVideoIndexerObject(identity))
	}
	if props, ok := payload["properties"]; ok {
		account.Properties = azureVideoIndexerMergeAnyMap(account.Properties, azureVideoIndexerObject(props))
	}
	account.Properties["provisioningState"] = "Succeeded"
	account.UpdatedAt = time.Now().UTC()
	account.ETag = azureVideoIndexerETag(azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name), account.UpdatedAt)
	respondJSON(w, http.StatusOK, azureVideoIndexerAccountFixture(account))
}

func (s *Server) handleAzureVideoIndexerDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name)
	if _, ok := s.azureVideoIndexerAccounts[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", name))
		return
	}
	delete(s.azureVideoIndexerAccounts, key)
	prefix := strings.ToLower(azureVideoIndexerAccountID(subscriptionID, resourceGroup, name) + "/privateEndpointConnections/")
	for connKey := range s.azureVideoIndexerPrivateEndpointConnections {
		if strings.HasPrefix(strings.ToLower(connKey), prefix) {
			delete(s.azureVideoIndexerPrivateEndpointConnections, connKey)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureVideoIndexerGenerateToken(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name, mode string) {
	_, payload, ok := azureVideoIndexerReadOptionalRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name)]; !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", name))
		return
	}

	scope := strings.TrimSpace(azureVideoIndexerString(payload["scope"]))
	if scope == "" {
		scope = "Account"
	}
	if !azureVideoIndexerAllowedScope(scope) {
		respondAzureInvalidRequest(w, path, "scope must be Account, Project, or Video")
		return
	}
	projectID := strings.TrimSpace(azureVideoIndexerString(payload["projectId"]))
	videoID := strings.TrimSpace(azureVideoIndexerString(payload["videoId"]))
	switch strings.ToLower(scope) {
	case "project":
		if projectID == "" {
			respondAzureInvalidRequest(w, path, "projectId is required when scope is Project")
			return
		}
	case "video":
		if videoID == "" {
			respondAzureInvalidRequest(w, path, "videoId is required when scope is Video")
			return
		}
	}
	lifetime := 3600
	if raw, ok := payload["tokenLifetimeInSeconds"]; ok {
		parsed, ok := azureVideoIndexerInt(raw)
		if !ok || parsed <= 0 || parsed > 43200 {
			respondAzureInvalidRequest(w, path, "tokenLifetimeInSeconds must be between 1 and 43200")
			return
		}
		lifetime = parsed
	}
	permissionType := strings.TrimSpace(azureVideoIndexerString(payload["permissionType"]))
	extensionID := strings.TrimSpace(azureVideoIndexerString(payload["extensionId"]))
	switch mode {
	case "access", "extension_access":
		if !azureVideoIndexerAllowedPermission(permissionType) {
			respondAzureInvalidRequest(w, path, "permissionType must be Reader or Contributor")
			return
		}
	default:
		if permissionType != "" {
			respondAzureInvalidRequest(w, path, "permissionType is not allowed for restricted viewer tokens")
			return
		}
	}
	if mode == "extension_access" || mode == "extension_restricted_viewer" {
		if extensionID == "" {
			respondAzureInvalidRequest(w, path, "extensionId is required for extension token generation")
			return
		}
	}

	token := azureVideoIndexerDeterministicToken(mode, name, scope, permissionType, projectID, videoID, extensionID, lifetime)
	respondJSON(w, http.StatusOK, map[string]any{"accessToken": token})
}

func (s *Server) handleAzureVideoIndexerPrivateEndpointConnectionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureVideoIndexerPrivateEndpointConnectionList(w, path, subscriptionID, resourceGroup, accountName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "private endpoint connection name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureVideoIndexerPrivateEndpointConnectionGet(w, path, subscriptionID, resourceGroup, accountName, name)
	case http.MethodPut:
		s.handleAzureVideoIndexerPrivateEndpointConnectionCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, name)
	case http.MethodDelete:
		s.handleAzureVideoIndexerPrivateEndpointConnectionDelete(w, path, subscriptionID, resourceGroup, accountName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureVideoIndexerPrivateEndpointConnectionList(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, accountName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", accountName))
		return
	}
	items := make([]map[string]any, 0)
	for _, conn := range s.azureVideoIndexerPrivateEndpointConnections {
		if strings.EqualFold(conn.SubscriptionID, subscriptionID) && strings.EqualFold(conn.ResourceGroup, resourceGroup) && strings.EqualFold(conn.AccountName, accountName) {
			items = append(items, azureVideoIndexerPrivateEndpointConnectionFixture(conn))
		}
	}
	azureVideoIndexerSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureVideoIndexerPrivateEndpointConnectionGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	conn, ok := s.azureVideoIndexerPrivateEndpointConnections[azureVideoIndexerPrivateEndpointConnectionKey(subscriptionID, resourceGroup, accountName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "PrivateEndpointConnection", name))
		return
	}
	respondJSON(w, http.StatusOK, azureVideoIndexerPrivateEndpointConnectionFixture(conn))
}

func (s *Server) handleAzureVideoIndexerPrivateEndpointConnectionCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, name string) {
	_, payload, ok := azureVideoIndexerReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, accountName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", accountName))
		return
	}
	now := time.Now().UTC()
	key := azureVideoIndexerPrivateEndpointConnectionKey(subscriptionID, resourceGroup, accountName, name)
	conn, exists := s.azureVideoIndexerPrivateEndpointConnections[key]
	if !exists {
		conn = &azureVideoIndexerPrivateEndpointConnection{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			AccountName:    accountName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	conn.Location = account.Location
	conn.Properties = azureVideoIndexerCloneAnyMap(azureVideoIndexerObject(payload["properties"]))
	if conn.Properties == nil {
		conn.Properties = map[string]any{}
	}
	conn.Properties["provisioningState"] = "Succeeded"
	state := azureVideoIndexerObject(conn.Properties["privateLinkServiceConnectionState"])
	if len(state) == 0 {
		state = map[string]any{}
	}
	if _, ok := state["status"]; !ok {
		state["status"] = "Approved"
	}
	if _, ok := state["description"]; !ok {
		state["description"] = "Approved by Stackyard"
	}
	if _, ok := state["actionsRequired"]; !ok {
		state["actionsRequired"] = "None"
	}
	conn.Properties["privateLinkServiceConnectionState"] = state
	if _, ok := conn.Properties["groupIds"]; !ok {
		conn.Properties["groupIds"] = []any{"avi"}
	}
	conn.UpdatedAt = now
	conn.ETag = azureVideoIndexerETag(key, now)
	s.azureVideoIndexerPrivateEndpointConnections[key] = conn
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureVideoIndexerPrivateEndpointConnectionFixture(conn))
}

func (s *Server) handleAzureVideoIndexerPrivateEndpointConnectionDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureVideoIndexerPrivateEndpointConnectionKey(subscriptionID, resourceGroup, accountName, name)
	if _, ok := s.azureVideoIndexerPrivateEndpointConnections[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "PrivateEndpointConnection", name))
		return
	}
	delete(s.azureVideoIndexerPrivateEndpointConnections, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureVideoIndexerPrivateLinkResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureVideoIndexerPrivateLinkResourceList(w, path, subscriptionID, resourceGroup, accountName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "private link resource groupId is required")
		return
	}
	if r.Method == http.MethodGet {
		s.handleAzureVideoIndexerPrivateLinkResourceGet(w, path, subscriptionID, resourceGroup, accountName, name)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleAzureVideoIndexerPrivateLinkResourceList(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, accountName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", accountName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": []any{
		azureVideoIndexerPrivateLinkResourceFixture(account, "avi"),
	}})
}

func (s *Server) handleAzureVideoIndexerPrivateLinkResourceGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, groupID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account, ok := s.azureVideoIndexerAccounts[azureVideoIndexerAccountKey(subscriptionID, resourceGroup, accountName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "Account", accountName))
		return
	}
	if !strings.EqualFold(groupID, "avi") {
		respondJSON(w, http.StatusNotFound, azureVideoIndexerNotFound(path, "PrivateLinkResource", groupID))
		return
	}
	respondJSON(w, http.StatusOK, azureVideoIndexerPrivateLinkResourceFixture(account, "avi"))
}

func (s *Server) azureVideoIndexerEnsureSeeded(subscriptionID, resourceGroup string) {
	resourceGroup = strings.TrimSpace(resourceGroup)
	if resourceGroup == "" {
		resourceGroup = "rg-videoindexer"
	}
	key := azureVideoIndexerAccountKey(subscriptionID, resourceGroup, "seed-account")
	if _, ok := s.azureVideoIndexerAccounts[key]; ok {
		return
	}
	now := time.Now().UTC()
	account := &azureVideoIndexerAccount{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           "seed-account",
		Location:       "eastus",
		Identity:       map[string]any{"type": "SystemAssigned", "principalId": "principal-seed-account"},
		Tags:           map[string]string{"seed": "true"},
		Properties: map[string]any{
			"provisioningState": "Succeeded",
			"accountId":         "avi-seed-account",
			"endpoint":          "https://seed-account.videoindexer.ai",
			"storageServices":   map[string]any{"current": "ManagedByMicrosoft"},
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureVideoIndexerETag(key, now),
	}
	s.azureVideoIndexerAccounts[key] = account
	s.azureVideoIndexerEnsureDefaultPrivateEndpointConnection(account)
}

func (s *Server) azureVideoIndexerEnsureDefaultPrivateEndpointConnection(account *azureVideoIndexerAccount) {
	key := azureVideoIndexerPrivateEndpointConnectionKey(account.SubscriptionID, account.ResourceGroup, account.Name, "default")
	if _, ok := s.azureVideoIndexerPrivateEndpointConnections[key]; ok {
		return
	}
	now := time.Now().UTC()
	s.azureVideoIndexerPrivateEndpointConnections[key] = &azureVideoIndexerPrivateEndpointConnection{
		SubscriptionID: account.SubscriptionID,
		ResourceGroup:  account.ResourceGroup,
		AccountName:    account.Name,
		Name:           "default",
		Location:       account.Location,
		Properties: map[string]any{
			"provisioningState": "Succeeded",
			"privateEndpoint":   map[string]any{"id": "/subscriptions/" + account.SubscriptionID + "/resourceGroups/" + account.ResourceGroup + "/providers/Microsoft.Network/privateEndpoints/pe-default"},
			"privateLinkServiceConnectionState": map[string]any{
				"status":          "Approved",
				"description":     "Approved by Stackyard",
				"actionsRequired": "None",
			},
			"groupIds": []any{"avi"},
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureVideoIndexerETag(key, now),
	}
}

func azureVideoIndexerOperationsFixture() []map[string]any {
	ops := []string{
		"Microsoft.VideoIndexer/accounts/read",
		"Microsoft.VideoIndexer/accounts/write",
		"Microsoft.VideoIndexer/accounts/delete",
		"Microsoft.VideoIndexer/accounts/generateAccessToken/action",
		"Microsoft.VideoIndexer/accounts/generateRestrictedViewerAccessToken/action",
		"Microsoft.VideoIndexer/accounts/generateExtensionAccessToken/action",
		"Microsoft.VideoIndexer/accounts/generateExtensionRestrictedViewerAccessToken/action",
		"Microsoft.VideoIndexer/accounts/privateEndpointConnections/read",
		"Microsoft.VideoIndexer/accounts/privateEndpointConnections/write",
		"Microsoft.VideoIndexer/accounts/privateEndpointConnections/delete",
		"Microsoft.VideoIndexer/accounts/privateLinkResources/read",
	}
	out := make([]map[string]any, 0, len(ops))
	for _, name := range ops {
		out = append(out, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":  "Microsoft Video Indexer",
				"resource":  "accounts",
				"operation": name,
			},
			"isDataAction": false,
			"origin":       "user,system",
		})
	}
	return out
}

func azureVideoIndexerAccountFixture(account *azureVideoIndexerAccount) map[string]any {
	return map[string]any{
		"id":         azureVideoIndexerAccountID(account.SubscriptionID, account.ResourceGroup, account.Name),
		"name":       account.Name,
		"type":       "Microsoft.VideoIndexer/accounts",
		"location":   account.Location,
		"identity":   azureVideoIndexerCloneAnyMap(account.Identity),
		"tags":       azureVideoIndexerAnyMapFromStringMap(account.Tags),
		"properties": azureVideoIndexerCloneAnyMap(account.Properties),
		"etag":       account.ETag,
		"systemData": azureVideoIndexerSystemData(account.CreatedAt, account.UpdatedAt),
	}
}

func azureVideoIndexerPrivateEndpointConnectionFixture(conn *azureVideoIndexerPrivateEndpointConnection) map[string]any {
	return map[string]any{
		"id":         azureVideoIndexerPrivateEndpointConnectionID(conn.SubscriptionID, conn.ResourceGroup, conn.AccountName, conn.Name),
		"name":       conn.Name,
		"type":       "Microsoft.VideoIndexer/accounts/privateEndpointConnections",
		"location":   conn.Location,
		"properties": azureVideoIndexerCloneAnyMap(conn.Properties),
		"etag":       conn.ETag,
		"systemData": azureVideoIndexerSystemData(conn.CreatedAt, conn.UpdatedAt),
	}
}

func azureVideoIndexerPrivateLinkResourceFixture(account *azureVideoIndexerAccount, groupID string) map[string]any {
	return map[string]any{
		"id":       azureVideoIndexerAccountID(account.SubscriptionID, account.ResourceGroup, account.Name) + "/privateLinkResources/" + groupID,
		"name":     groupID,
		"type":     "Microsoft.VideoIndexer/accounts/privateLinkResources",
		"location": account.Location,
		"properties": map[string]any{
			"groupId":           groupID,
			"requiredMembers":   []any{groupID},
			"requiredZoneNames": []any{"privatelink.avi.microsoft.com"},
		},
		"systemData": azureVideoIndexerSystemData(account.CreatedAt, account.UpdatedAt),
	}
}

func azureVideoIndexerSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":      createdAt.Format(time.RFC3339),
		"createdBy":      "stackyard",
		"lastModifiedAt": updatedAt.Format(time.RFC3339),
		"lastModifiedBy": "stackyard",
	}
}

func azureVideoIndexerNotFound(path, kind, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "NotFound",
			"message": fmt.Sprintf("%s %q was not found", kind, name),
		},
		"provider": providerAzure,
		"path":     path,
	}
}

func azureVideoIndexerReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, payload, ok := azureVideoIndexerReadOptionalRequestBody(w, r, path)
	if !ok {
		return nil, nil, false
	}
	if len(body) == 0 {
		respondAzureInvalidRequest(w, path, "request body is required")
		return nil, nil, false
	}
	return body, payload, true
}

func azureVideoIndexerReadOptionalRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	defer r.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "eof") {
			return nil, map[string]any{}, true
		}
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	body, _ := json.Marshal(payload)
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureVideoIndexerString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func azureVideoIndexerObject(value any) map[string]any {
	out, _ := value.(map[string]any)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func azureVideoIndexerStringMap(value any) map[string]string {
	src, _ := value.(map[string]any)
	if src == nil {
		if direct, ok := value.(map[string]string); ok {
			return azureVideoIndexerCloneStringMap(direct)
		}
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, raw := range src {
		out[key] = strings.TrimSpace(azureVideoIndexerString(raw))
	}
	return out
}

func azureVideoIndexerCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureVideoIndexerCloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureVideoIndexerCloneAnyMap(typed)
		case []any:
			out[key] = append([]any(nil), typed...)
		default:
			out[key] = value
		}
	}
	return out
}

func azureVideoIndexerMergeAnyMap(dst, src map[string]any) map[string]any {
	out := azureVideoIndexerCloneAnyMap(dst)
	for key, value := range src {
		out[key] = value
	}
	return out
}

func azureVideoIndexerAnyMapFromStringMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureVideoIndexerSortByID(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureVideoIndexerString(items[i]["id"])) < strings.ToLower(azureVideoIndexerString(items[j]["id"]))
	})
}

func azureVideoIndexerValidAccountName(name string) bool {
	if len(name) < 3 || len(name) > 50 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func azureVideoIndexerAllowedScope(scope string) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "account", "project", "video":
		return true
	default:
		return false
	}
}

func azureVideoIndexerAllowedPermission(permission string) bool {
	switch strings.ToLower(strings.TrimSpace(permission)) {
	case "reader", "contributor":
		return true
	default:
		return false
	}
}

func azureVideoIndexerInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case json.Number:
		n, err := typed.Int64()
		return int(n), err == nil
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(typed))
		return n, err == nil
	default:
		return 0, false
	}
}

func azureVideoIndexerDeterministicToken(parts ...any) string {
	rawParts := make([]string, 0, len(parts))
	for _, part := range parts {
		rawParts = append(rawParts, strings.TrimSpace(azureVideoIndexerString(part)))
	}
	sum := sha1.Sum([]byte(strings.Join(rawParts, "|")))
	return "vi." + hex.EncodeToString(sum[:])
}

func azureVideoIndexerAccountKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureVideoIndexerAccountID(subscriptionID, resourceGroup, name))
}

func azureVideoIndexerPrivateEndpointConnectionKey(subscriptionID, resourceGroup, accountName, name string) string {
	return strings.ToLower(azureVideoIndexerPrivateEndpointConnectionID(subscriptionID, resourceGroup, accountName, name))
}

func azureVideoIndexerAccountID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.VideoIndexer/accounts/%s", subscriptionID, resourceGroup, name)
}

func azureVideoIndexerPrivateEndpointConnectionID(subscriptionID, resourceGroup, accountName, name string) string {
	return azureVideoIndexerAccountID(subscriptionID, resourceGroup, accountName) + "/privateEndpointConnections/" + name
}

func azureVideoIndexerETag(seed string, t time.Time) string {
	sum := sha1.Sum([]byte(seed + "|" + t.UTC().Format(time.RFC3339Nano)))
	return `W/"` + hex.EncodeToString(sum[:8]) + `"`
}
