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

const azureWebPubSubAPIVersion = "2024-03-01"

type azureWebPubSubAccount struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Identity       map[string]any
	Tags           map[string]string
	Properties     map[string]any
	PrimaryKey     string
	SecondaryKey   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureWebPubSubChildResource struct {
	SubscriptionID string
	ResourceGroup  string
	AccountName    string
	ReplicaName    string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

var azureWebPubSubMutableAccountCollections = map[string]string{
	"customcertificates":         "customCertificates",
	"customdomains":              "customDomains",
	"hubs":                       "hubs",
	"privateendpointconnections": "privateEndpointConnections",
	"sharedprivatelinkresources": "sharedPrivateLinkResources",
}

func (s *Server) handleAzureWebPubSubRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsWebPubSubPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureWebPubSubAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.SignalRService") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureWebPubSubOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureWebPubSubRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations":
		s.handleAzureWebPubSubLocationRoute(w, r, path, subscriptionID, tail[1:])
		return true
	case "webpubsub":
		s.handleAzureWebPubSubAccountsRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		return false
	}
}

func azureIsWebPubSubPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.SignalRService") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureWebPubSubRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations", "webpubsub":
		return true
	default:
		return false
	}
}

func azureWebPubSubAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureWebPubSubRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.SignalRService") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.SignalRService") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureWebPubSubLocationRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	location := strings.TrimSpace(tail[0])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "usages":
		if r.Method != http.MethodGet || len(tail) != 2 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureWebPubSubUsagesFixture(subscriptionID, location)})
	case "checknameavailability":
		if r.Method != http.MethodPost || len(tail) != 2 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubCheckNameAvailability(w, r, path, subscriptionID, location)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureWebPubSubCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureWebPubSubReadRequestBody(w, r, path)
	if !ok {
		return
	}
	_ = subscriptionID

	name := strings.TrimSpace(azureWebPubSubString(payload["name"]))
	resourceType := strings.TrimSpace(azureWebPubSubString(payload["type"]))
	if name == "" || resourceType == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	// SignalR shares this location-scoped route; its resources live in the shared ARM store.
	if strings.EqualFold(resourceType, azureSignalRType) {
		respondJSON(w, http.StatusOK, s.azureSignalRNameAvailability(name))
		return
	}
	if !azureWebPubSubAllowedNameAvailabilityType(resourceType) {
		respondAzureInvalidRequest(w, path, "type must target Microsoft.SignalRService/WebPubSub")
		return
	}
	if !azureWebPubSubValidName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "resource name must be 3-63 characters, start with a letter, and end with a letter or digit",
		})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, account := range s.azureWebPubSubAccounts {
		if strings.EqualFold(account.Name, name) && strings.EqualFold(account.Location, location) {
			respondJSON(w, http.StatusOK, map[string]any{
				"nameAvailable": false,
				"reason":        "AlreadyExists",
				"message":       "resource name is already in use",
			})
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *Server) handleAzureWebPubSubAccountsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureWebPubSubListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureWebPubSubListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureWebPubSubValidName(name) {
		respondAzureInvalidRequest(w, path, "resource name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureWebPubSubGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureWebPubSubGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureWebPubSubCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureWebPubSubPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureWebPubSubDelete(w, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "listkeys":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureWebPubSubListKeys(w, path, subscriptionID, resourceGroup, name)
			return
		}
	case "regeneratekey":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureWebPubSubRegenerateKey(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "skus":
		if r.Method == http.MethodGet && len(tail) == 3 {
			s.handleAzureWebPubSubListSkus(w, path, subscriptionID, resourceGroup, name)
			return
		}
	case "privatelinkresources":
		s.handleAzureWebPubSubPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	case "replicas":
		s.handleAzureWebPubSubReplicasRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	default:
		if collection, ok := azureWebPubSubMutableAccountCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok {
			s.handleAzureWebPubSubAccountChildRoute(w, r, path, subscriptionID, resourceGroup, name, collection, tail[2:])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureWebPubSubAccountChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, collection string, tail []string) {
	if !s.azureWebPubSubAccountExists(subscriptionID, resourceGroup, accountName) {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, accountName))
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubChildList(w, subscriptionID, resourceGroup, accountName, "", collection)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "child resource name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureWebPubSubChildGet(w, path, subscriptionID, resourceGroup, accountName, "", collection, name, false)
	case http.MethodHead:
		s.handleAzureWebPubSubChildGet(w, path, subscriptionID, resourceGroup, accountName, "", collection, name, true)
	case http.MethodPut:
		s.handleAzureWebPubSubChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, "", collection, name)
	case http.MethodPatch:
		s.handleAzureWebPubSubChildPatch(w, r, path, subscriptionID, resourceGroup, accountName, "", collection, name)
	case http.MethodDelete:
		s.handleAzureWebPubSubChildDelete(w, subscriptionID, resourceGroup, accountName, "", collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureWebPubSubPrivateLinkResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureWebPubSubAccountExists(subscriptionID, resourceGroup, accountName) {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, accountName))
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 1 {
		respondJSON(w, http.StatusOK, map[string]any{
			"value": azureWebPubSubPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName),
		})
		return
	}
	name := strings.TrimSpace(tail[1])
	for _, item := range azureWebPubSubPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName) {
		if strings.EqualFold(azureWebPubSubString(item["name"]), name) {
			respondJSON(w, http.StatusOK, item)
			return
		}
	}
	respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
}

func (s *Server) handleAzureWebPubSubReplicasRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureWebPubSubAccountExists(subscriptionID, resourceGroup, accountName) {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, accountName))
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubChildList(w, subscriptionID, resourceGroup, accountName, "", "replicas")
		return
	}

	replicaName := strings.TrimSpace(tail[1])
	if !azureWebPubSubValidName(replicaName) {
		respondAzureInvalidRequest(w, path, "replica name is invalid")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureWebPubSubChildGet(w, path, subscriptionID, resourceGroup, accountName, "", "replicas", replicaName, false)
		case http.MethodHead:
			s.handleAzureWebPubSubChildGet(w, path, subscriptionID, resourceGroup, accountName, "", "replicas", replicaName, true)
		case http.MethodPut:
			s.handleAzureWebPubSubChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, "", "replicas", replicaName)
		case http.MethodPatch:
			s.handleAzureWebPubSubChildPatch(w, r, path, subscriptionID, resourceGroup, accountName, "", "replicas", replicaName)
		case http.MethodDelete:
			s.handleAzureWebPubSubChildDelete(w, subscriptionID, resourceGroup, accountName, "", "replicas", replicaName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "skus":
		if r.Method != http.MethodGet || len(tail) != 3 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !s.azureWebPubSubChildExists(subscriptionID, resourceGroup, accountName, "", "replicas", replicaName) {
			respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, replicaName))
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureWebPubSubReplicaSkusFixture(subscriptionID, resourceGroup, accountName, replicaName)})
		return
	case "sharedprivatelinkresources":
		s.handleAzureWebPubSubReplicaSharedPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, accountName, replicaName, tail[2:])
		return
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureWebPubSubReplicaSharedPrivateLinkResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, replicaName string, tail []string) {
	if !s.azureWebPubSubChildExists(subscriptionID, resourceGroup, accountName, "", "replicas", replicaName) {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, replicaName))
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureWebPubSubChildList(w, subscriptionID, resourceGroup, accountName, replicaName, "replicaSharedPrivateLinkResources")
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "shared private link resource name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureWebPubSubChildGet(w, path, subscriptionID, resourceGroup, accountName, replicaName, "replicaSharedPrivateLinkResources", name, false)
	case http.MethodHead:
		s.handleAzureWebPubSubChildGet(w, path, subscriptionID, resourceGroup, accountName, replicaName, "replicaSharedPrivateLinkResources", name, true)
	case http.MethodPut:
		s.handleAzureWebPubSubChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, replicaName, "replicaSharedPrivateLinkResources", name)
	case http.MethodPatch:
		s.handleAzureWebPubSubChildPatch(w, r, path, subscriptionID, resourceGroup, accountName, replicaName, "replicaSharedPrivateLinkResources", name)
	case http.MethodDelete:
		s.handleAzureWebPubSubChildDelete(w, subscriptionID, resourceGroup, accountName, replicaName, "replicaSharedPrivateLinkResources", name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureWebPubSubCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureWebPubSubReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureWebPubSubString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account := s.azureWebPubSubAccounts[key]
	status := http.StatusOK
	createdAt := now
	primaryKey := azureWebPubSubStableSecret("primary", subscriptionID, resourceGroup, name, now)
	secondaryKey := azureWebPubSubStableSecret("secondary", subscriptionID, resourceGroup, name, now)
	if account == nil {
		account = &azureWebPubSubAccount{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			PrimaryKey:     primaryKey,
			SecondaryKey:   secondaryKey,
		}
		status = http.StatusCreated
	} else {
		createdAt = account.CreatedAt
		primaryKey = account.PrimaryKey
		secondaryKey = account.SecondaryKey
	}

	account.Location = location
	account.Identity = azureWebPubSubObject(payload["identity"])
	account.Tags = azureWebPubSubStringMap(payload["tags"])
	account.Properties = azureWebPubSubNormalizeAccountProperties(name, subscriptionID, resourceGroup, location, azureWebPubSubObject(payload["properties"]), createdAt, now)
	account.PrimaryKey = primaryKey
	account.SecondaryKey = secondaryKey
	account.CreatedAt = createdAt
	account.UpdatedAt = now
	account.ETag = azureWebPubSubETag(key, body, now)
	s.azureWebPubSubAccounts[key] = account

	respondJSON(w, status, azureWebPubSubAccountFixture(account))
}

func (s *Server) handleAzureWebPubSubPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureWebPubSubReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	key := azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account := s.azureWebPubSubAccounts[key]
	if account == nil {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}

	updated := false
	if _, ok := payload["tags"]; ok {
		account.Tags = azureWebPubSubStringMap(payload["tags"])
		updated = true
	}
	if rawIdentity, ok := payload["identity"]; ok {
		account.Identity = azureWebPubSubObject(rawIdentity)
		updated = true
	}
	if rawProps, ok := payload["properties"]; ok {
		account.Properties = azureWebPubSubMergeObjects(account.Properties, azureWebPubSubObject(rawProps))
		account.Properties = azureWebPubSubNormalizeAccountProperties(name, subscriptionID, resourceGroup, account.Location, account.Properties, account.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	account.UpdatedAt = now
	account.ETag = azureWebPubSubETag(key, body, now)
	respondJSON(w, http.StatusOK, azureWebPubSubAccountFixture(account))
}

func (s *Server) handleAzureWebPubSubGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	account := s.azureWebPubSubAccounts[azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if account == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureWebPubSubAccountFixture(account))
}

func (s *Server) handleAzureWebPubSubDelete(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	key := azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)
	accountID := azureWebPubSubAccountID(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureWebPubSubAccounts[key]
	delete(s.azureWebPubSubAccounts, key)
	for childKey, child := range s.azureWebPubSubChildren {
		if strings.EqualFold(azureWebPubSubChildParentID(child), accountID) {
			delete(s.azureWebPubSubChildren, childKey)
		}
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Location", "/azure"+accountID+"?api-version="+azureWebPubSubAPIVersion)
	w.Header().Set("Retry-After", "10")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureWebPubSubListKeys(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	account := s.azureWebPubSubAccounts[azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if account == nil {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureWebPubSubKeysFixture(account))
}

func (s *Server) handleAzureWebPubSubRegenerateKey(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureWebPubSubReadRequestBody(w, r, path)
	if !ok {
		return
	}
	keyType := strings.TrimSpace(azureWebPubSubString(payload["keyType"]))
	if keyType == "" {
		keyType = strings.TrimSpace(azureWebPubSubString(payload["keyName"]))
	}
	if keyType == "" {
		respondAzureInvalidRequest(w, path, "keyType is required")
		return
	}

	now := time.Now().UTC()
	key := azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	account := s.azureWebPubSubAccounts[key]
	if account == nil {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}
	switch strings.ToLower(strings.TrimSpace(keyType)) {
	case "primary", "primarykey":
		account.PrimaryKey = azureWebPubSubStableSecret("primary", subscriptionID, resourceGroup, name, now)
	case "secondary", "secondarykey":
		account.SecondaryKey = azureWebPubSubStableSecret("secondary", subscriptionID, resourceGroup, name, now)
	default:
		respondAzureInvalidRequest(w, path, "keyType must be Primary or Secondary")
		return
	}
	account.UpdatedAt = now
	account.ETag = azureWebPubSubETag(key, body, now)
	respondJSON(w, http.StatusOK, azureWebPubSubKeysFixture(account))
}

func (s *Server) handleAzureWebPubSubListSkus(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	if !s.azureWebPubSubAccountExists(subscriptionID, resourceGroup, name) {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": azureWebPubSubSkusFixture(subscriptionID, resourceGroup, name)})
}

func (s *Server) handleAzureWebPubSubListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureWebPubSubEnsureSeeded(subscriptionID, "rg-webpubsub")
	values := make([]map[string]any, 0)
	for _, account := range s.azureWebPubSubAccounts {
		if account.SubscriptionID != subscriptionID {
			continue
		}
		values = append(values, azureWebPubSubAccountFixture(account))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureWebPubSubString(values[i]["id"]) < azureWebPubSubString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureWebPubSubListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureWebPubSubEnsureSeeded(subscriptionID, resourceGroup)
	values := make([]map[string]any, 0)
	for _, account := range s.azureWebPubSubAccounts {
		if account.SubscriptionID != subscriptionID || !strings.EqualFold(account.ResourceGroup, resourceGroup) {
			continue
		}
		values = append(values, azureWebPubSubAccountFixture(account))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureWebPubSubString(values[i]["id"]) < azureWebPubSubString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureWebPubSubChildList(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, replicaName, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, child := range s.azureWebPubSubChildren {
		if child.SubscriptionID != subscriptionID ||
			!strings.EqualFold(child.ResourceGroup, resourceGroup) ||
			!strings.EqualFold(child.AccountName, accountName) ||
			!strings.EqualFold(child.Collection, collection) ||
			!strings.EqualFold(child.ReplicaName, replicaName) {
			continue
		}
		values = append(values, azureWebPubSubChildFixture(child))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureWebPubSubString(values[i]["id"]) < azureWebPubSubString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureWebPubSubChildGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, replicaName, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	child := s.azureWebPubSubChildren[azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, replicaName, collection, name)]
	s.providerStorageMu.Unlock()
	if child == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureWebPubSubChildFixture(child))
}

func (s *Server) handleAzureWebPubSubChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, replicaName, collection, name string) {
	body, payload, ok := azureWebPubSubReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if collection == "privateEndpointConnections" {
		properties := azureWebPubSubObject(payload["properties"])
		if azureWebPubSubString(azureWebPubSubObject(properties["privateEndpoint"])["id"]) == "" {
			respondAzureInvalidRequest(w, path, "privateEndpoint.id is required")
			return
		}
	}

	s.providerStorageMu.Lock()
	account := s.azureWebPubSubAccounts[azureWebPubSubAccountKey(subscriptionID, resourceGroup, accountName)]
	if account == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, accountName))
		return
	}
	if replicaName != "" && collection == "replicaSharedPrivateLinkResources" {
		if s.azureWebPubSubChildren[azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, "", "replicas", replicaName)] == nil {
			s.providerStorageMu.Unlock()
			respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, replicaName))
			return
		}
	}

	key := azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, replicaName, collection, name)
	now := time.Now().UTC()
	child := s.azureWebPubSubChildren[key]
	status := http.StatusOK
	createdAt := now
	if child == nil {
		child = &azureWebPubSubChildResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			AccountName:    accountName,
			ReplicaName:    replicaName,
			Collection:     collection,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = child.CreatedAt
	}
	location := strings.TrimSpace(azureWebPubSubString(payload["location"]))
	if location == "" {
		location = account.Location
	}
	child.Location = location
	child.Tags = azureWebPubSubStringMap(payload["tags"])
	child.Properties = azureWebPubSubNormalizeChildProperties(collection, name, replicaName, location, azureWebPubSubObject(payload["properties"]), createdAt, now)
	child.CreatedAt = createdAt
	child.UpdatedAt = now
	child.ETag = azureWebPubSubETag(key, body, now)
	s.azureWebPubSubChildren[key] = child
	s.providerStorageMu.Unlock()

	respondJSON(w, status, azureWebPubSubChildFixture(child))
}

func (s *Server) handleAzureWebPubSubChildPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, replicaName, collection, name string) {
	body, payload, ok := azureWebPubSubReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	key := azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, replicaName, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	child := s.azureWebPubSubChildren[key]
	if child == nil {
		respondJSON(w, http.StatusNotFound, azureWebPubSubNotFound(path, name))
		return
	}
	updated := false
	if _, ok := payload["tags"]; ok {
		child.Tags = azureWebPubSubStringMap(payload["tags"])
		updated = true
	}
	if rawProps, ok := payload["properties"]; ok {
		child.Properties = azureWebPubSubMergeObjects(child.Properties, azureWebPubSubObject(rawProps))
		child.Properties = azureWebPubSubNormalizeChildProperties(collection, name, replicaName, child.Location, child.Properties, child.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	child.UpdatedAt = now
	child.ETag = azureWebPubSubETag(key, body, now)
	respondJSON(w, http.StatusOK, azureWebPubSubChildFixture(child))
}

func (s *Server) handleAzureWebPubSubChildDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, replicaName, collection, name string) {
	key := azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, replicaName, collection, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureWebPubSubChildren[key]
	delete(s.azureWebPubSubChildren, key)
	if collection == "replicas" {
		for childKey, child := range s.azureWebPubSubChildren {
			if child.SubscriptionID == subscriptionID &&
				strings.EqualFold(child.ResourceGroup, resourceGroup) &&
				strings.EqualFold(child.AccountName, accountName) &&
				strings.EqualFold(child.ReplicaName, name) {
				delete(s.azureWebPubSubChildren, childKey)
			}
		}
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) azureWebPubSubEnsureSeeded(subscriptionID, resourceGroup string) {
	key := azureWebPubSubAccountKey(subscriptionID, resourceGroup, "seed-webpubsub")
	if s.azureWebPubSubAccounts[key] != nil {
		return
	}
	now := time.Now().UTC()
	account := &azureWebPubSubAccount{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           "seed-webpubsub",
		Location:       "eastus",
		Tags:           map[string]string{"seed": "true"},
		Properties:     azureWebPubSubNormalizeAccountProperties("seed-webpubsub", subscriptionID, resourceGroup, "eastus", map[string]any{}, now, now),
		PrimaryKey:     azureWebPubSubStableSecret("primary", subscriptionID, resourceGroup, "seed-webpubsub", now),
		SecondaryKey:   azureWebPubSubStableSecret("secondary", subscriptionID, resourceGroup, "seed-webpubsub", now),
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureWebPubSubETag(key, nil, now),
	}
	s.azureWebPubSubAccounts[key] = account
}

func (s *Server) azureWebPubSubAccountExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureWebPubSubAccounts[azureWebPubSubAccountKey(subscriptionID, resourceGroup, name)] != nil
}

func (s *Server) azureWebPubSubChildExists(subscriptionID, resourceGroup, accountName, replicaName, collection, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureWebPubSubChildren[azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, replicaName, collection, name)] != nil
}

func azureWebPubSubOperationsFixture() []map[string]any {
	operations := []string{
		"Microsoft.SignalRService/webPubSub/read",
		"Microsoft.SignalRService/webPubSub/write",
		"Microsoft.SignalRService/webPubSub/delete",
		"Microsoft.SignalRService/webPubSub/listKeys/action",
		"Microsoft.SignalRService/webPubSub/regenerateKey/action",
		"Microsoft.SignalRService/webPubSub/skus/read",
		"Microsoft.SignalRService/webPubSub/customCertificates/read",
		"Microsoft.SignalRService/webPubSub/customCertificates/write",
		"Microsoft.SignalRService/webPubSub/customCertificates/delete",
		"Microsoft.SignalRService/webPubSub/customDomains/read",
		"Microsoft.SignalRService/webPubSub/customDomains/write",
		"Microsoft.SignalRService/webPubSub/customDomains/delete",
		"Microsoft.SignalRService/webPubSub/hubs/read",
		"Microsoft.SignalRService/webPubSub/hubs/write",
		"Microsoft.SignalRService/webPubSub/hubs/delete",
		"Microsoft.SignalRService/webPubSub/privateEndpointConnections/read",
		"Microsoft.SignalRService/webPubSub/privateEndpointConnections/write",
		"Microsoft.SignalRService/webPubSub/privateEndpointConnections/delete",
		"Microsoft.SignalRService/webPubSub/privateLinkResources/read",
		"Microsoft.SignalRService/webPubSub/sharedPrivateLinkResources/read",
		"Microsoft.SignalRService/webPubSub/sharedPrivateLinkResources/write",
		"Microsoft.SignalRService/webPubSub/sharedPrivateLinkResources/delete",
		"Microsoft.SignalRService/webPubSub/replicas/read",
		"Microsoft.SignalRService/webPubSub/replicas/write",
		"Microsoft.SignalRService/webPubSub/replicas/delete",
		"Microsoft.SignalRService/webPubSub/replicas/skus/read",
		"Microsoft.SignalRService/webPubSub/replicas/sharedPrivateLinkResources/read",
		"Microsoft.SignalRService/webPubSub/replicas/sharedPrivateLinkResources/write",
		"Microsoft.SignalRService/webPubSub/replicas/sharedPrivateLinkResources/delete",
		"Microsoft.SignalRService/locations/usages/read",
	}
	out := make([]map[string]any, 0, len(operations))
	for _, name := range operations {
		out = append(out, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider": "Microsoft.SignalRService",
				"resource": strings.TrimPrefix(name, "Microsoft.SignalRService/"),
			},
		})
	}
	return out
}

func azureWebPubSubUsagesFixture(subscriptionID, location string) []map[string]any {
	base := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.SignalRService/locations/%s/usages", subscriptionID, location)
	return []map[string]any{
		{
			"id":           base + "/freeUnits",
			"name":         map[string]any{"value": "FreeUnits", "localizedValue": "Free Units"},
			"currentValue": 1,
			"limit":        5,
			"unit":         "Count",
		},
		{
			"id":           base + "/connections",
			"name":         map[string]any{"value": "ConcurrentConnections", "localizedValue": "Concurrent Connections"},
			"currentValue": 120,
			"limit":        1000,
			"unit":         "Count",
		},
	}
}

func azureWebPubSubSkusFixture(subscriptionID, resourceGroup, name string) []map[string]any {
	base := azureWebPubSubAccountID(subscriptionID, resourceGroup, name) + "/skus"
	return []map[string]any{
		{
			"id":   base + "/Free_F1",
			"name": "Free_F1",
			"sku":  map[string]any{"name": "Free_F1", "tier": "Free", "capacity": 1},
			"capacity": map[string]any{
				"default": 1,
				"minimum": 1,
				"maximum": 1,
			},
		},
		{
			"id":   base + "/Standard_S1",
			"name": "Standard_S1",
			"sku":  map[string]any{"name": "Standard_S1", "tier": "Standard", "capacity": 1},
			"capacity": map[string]any{
				"default": 1,
				"minimum": 1,
				"maximum": 100,
			},
		},
	}
}

func azureWebPubSubReplicaSkusFixture(subscriptionID, resourceGroup, accountName, replicaName string) []map[string]any {
	base := azureWebPubSubAccountID(subscriptionID, resourceGroup, accountName) + "/replicas/" + replicaName + "/skus"
	return []map[string]any{
		{
			"id":   base + "/Standard_S1",
			"name": "Standard_S1",
			"sku":  map[string]any{"name": "Standard_S1", "tier": "Standard", "capacity": 1},
		},
		{
			"id":   base + "/Standard_S2",
			"name": "Standard_S2",
			"sku":  map[string]any{"name": "Standard_S2", "tier": "Standard", "capacity": 2},
		},
	}
}

func azureWebPubSubAccountFixture(account *azureWebPubSubAccount) map[string]any {
	return map[string]any{
		"id":         azureWebPubSubAccountID(account.SubscriptionID, account.ResourceGroup, account.Name),
		"name":       account.Name,
		"type":       "Microsoft.SignalRService/webPubSub",
		"location":   account.Location,
		"identity":   account.Identity,
		"tags":       account.Tags,
		"etag":       account.ETag,
		"systemData": azureWebPubSubSystemData(account.CreatedAt, account.UpdatedAt),
		"sku":        map[string]any{"name": "Standard_S1", "tier": "Standard", "capacity": 1},
		"properties": azureWebPubSubCloneObject(account.Properties),
	}
}

func azureWebPubSubKeysFixture(account *azureWebPubSubAccount) map[string]any {
	return map[string]any{
		"primaryKey":                account.PrimaryKey,
		"secondaryKey":              account.SecondaryKey,
		"primaryConnectionString":   azureWebPubSubConnectionString(account.Name, account.PrimaryKey),
		"secondaryConnectionString": azureWebPubSubConnectionString(account.Name, account.SecondaryKey),
	}
}

func azureWebPubSubChildFixture(child *azureWebPubSubChildResource) map[string]any {
	return map[string]any{
		"id":         azureWebPubSubChildID(child),
		"name":       child.Name,
		"type":       azureWebPubSubChildType(child),
		"location":   child.Location,
		"tags":       child.Tags,
		"etag":       child.ETag,
		"systemData": azureWebPubSubSystemData(child.CreatedAt, child.UpdatedAt),
		"properties": azureWebPubSubCloneObject(child.Properties),
	}
}

func azureWebPubSubPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName string) []map[string]any {
	base := azureWebPubSubAccountID(subscriptionID, resourceGroup, accountName) + "/privateLinkResources"
	return []map[string]any{
		{
			"id":                base + "/webpubsub",
			"name":              "webpubsub",
			"type":              "Microsoft.SignalRService/webPubSub/privateLinkResources",
			"groupId":           "webpubsub",
			"requiredMembers":   []any{"webpubsub"},
			"requiredZoneNames": []any{"privatelink.webpubsub.azure.com"},
			"properties": map[string]any{
				"groupId":           "webpubsub",
				"requiredMembers":   []any{"webpubsub"},
				"requiredZoneNames": []any{"privatelink.webpubsub.azure.com"},
			},
		},
	}
}

func azureWebPubSubNormalizeAccountProperties(name, subscriptionID, resourceGroup, location string, properties map[string]any, createdAt, now time.Time) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureWebPubSubString(properties["hostName"])) == "" {
		properties["hostName"] = fmt.Sprintf("%s.webpubsub.azure.local", name)
	}
	if strings.TrimSpace(azureWebPubSubString(properties["externalIP"])) == "" {
		properties["externalIP"] = fmt.Sprintf("10.%d.%d.%d", len(name)%200+10, len(resourceGroup)%200+10, len(location)%200+10)
	}
	if _, ok := properties["publicPort"]; !ok {
		properties["publicPort"] = 443
	}
	if _, ok := properties["serverPort"]; !ok {
		properties["serverPort"] = 443
	}
	if strings.TrimSpace(azureWebPubSubString(properties["publicNetworkAccess"])) == "" {
		properties["publicNetworkAccess"] = "Enabled"
	}
	if _, ok := properties["resourceStopped"]; !ok {
		properties["resourceStopped"] = false
	}
	properties["provisioningState"] = "Succeeded"
	if _, ok := properties["version"]; !ok {
		properties["version"] = "1.0"
	}
	if _, ok := properties["liveTraceConfiguration"]; !ok {
		properties["liveTraceConfiguration"] = map[string]any{"enabled": false, "categories": []any{}}
	}
	if _, ok := properties["tls"]; !ok {
		properties["tls"] = map[string]any{"clientCertEnabled": false}
	}
	if _, ok := properties["networkACLs"]; !ok {
		properties["networkACLs"] = map[string]any{"defaultAction": "Allow"}
	}
	if _, ok := properties["privateEndpointConnections"]; !ok {
		properties["privateEndpointConnections"] = []any{}
	}
	if _, ok := properties["sharedPrivateLinkResources"]; !ok {
		properties["sharedPrivateLinkResources"] = []any{}
	}
	if _, ok := properties["replicas"]; !ok {
		properties["replicas"] = []any{}
	}
	properties["createdTime"] = createdAt.Format(time.RFC3339)
	properties["lastModifiedTime"] = now.Format(time.RFC3339)
	properties["resourceLogConfiguration"] = map[string]any{"enabled": false}
	properties["resourceUri"] = azureWebPubSubAccountID(subscriptionID, resourceGroup, name)
	return properties
}

func azureWebPubSubNormalizeChildProperties(collection, name, replicaName, location string, properties map[string]any, createdAt, now time.Time) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	switch collection {
	case "customCertificates":
		if strings.TrimSpace(azureWebPubSubString(properties["keyVaultBaseUri"])) == "" {
			properties["keyVaultBaseUri"] = "https://stackyard.vault.azure.net/"
		}
		if strings.TrimSpace(azureWebPubSubString(properties["provisioningState"])) == "" {
			properties["provisioningState"] = "Succeeded"
		}
	case "customDomains":
		if strings.TrimSpace(azureWebPubSubString(properties["domainName"])) == "" {
			properties["domainName"] = name + ".example.com"
		}
		if _, ok := properties["validated"]; !ok {
			properties["validated"] = true
		}
		properties["provisioningState"] = "Succeeded"
	case "hubs":
		if _, ok := properties["anonymousConnectPolicy"]; !ok {
			properties["anonymousConnectPolicy"] = "deny"
		}
		if _, ok := properties["eventHandlers"]; !ok {
			properties["eventHandlers"] = []any{}
		}
		properties["provisioningState"] = "Succeeded"
	case "privateEndpointConnections":
		if _, ok := properties["groupIds"]; !ok {
			properties["groupIds"] = []any{"webpubsub"}
		}
		if _, ok := properties["privateLinkServiceConnectionState"]; !ok {
			properties["privateLinkServiceConnectionState"] = map[string]any{
				"status":          "Approved",
				"description":     "Approved by Stackyard",
				"actionsRequired": "None",
			}
		}
		properties["provisioningState"] = "Succeeded"
	case "sharedPrivateLinkResources", "replicaSharedPrivateLinkResources":
		if strings.TrimSpace(azureWebPubSubString(properties["groupId"])) == "" {
			properties["groupId"] = "blob"
		}
		if strings.TrimSpace(azureWebPubSubString(properties["status"])) == "" {
			properties["status"] = "Approved"
		}
		properties["provisioningState"] = "Succeeded"
	case "replicas":
		if _, ok := properties["regionEndpointEnabled"]; !ok {
			properties["regionEndpointEnabled"] = true
		}
		if strings.TrimSpace(azureWebPubSubString(properties["regionEndpointType"])) == "" {
			properties["regionEndpointType"] = "Primary"
		}
		if strings.TrimSpace(azureWebPubSubString(properties["location"])) == "" {
			properties["location"] = location
		}
		if strings.TrimSpace(azureWebPubSubString(properties["replicaName"])) == "" {
			properties["replicaName"] = name
		}
		if strings.TrimSpace(azureWebPubSubString(properties["hostName"])) == "" {
			properties["hostName"] = fmt.Sprintf("%s-%s.webpubsub.azure.local", replicaName, name)
		}
		properties["provisioningState"] = "Succeeded"
	}
	properties["createdTime"] = createdAt.Format(time.RFC3339)
	properties["lastModifiedTime"] = now.Format(time.RFC3339)
	return properties
}

func azureWebPubSubNotFound(path, resourceName string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("The Resource '%s' under path '%s' was not found.", resourceName, path),
		},
	}
}

func azureWebPubSubReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureWebPubSubAllowedNameAvailabilityType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "microsoft.signalrservice/webpubsub", "microsoft.signalrservice/webpubsub/replicas":
		return true
	default:
		return false
	}
}

func azureWebPubSubValidName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	for idx, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
			if idx == 0 {
				return false
			}
		case r == '-':
			if idx == 0 || idx == len(name)-1 {
				return false
			}
		default:
			return false
		}
	}
	last := name[len(name)-1]
	return (last >= 'a' && last <= 'z') || (last >= 'A' && last <= 'Z') || (last >= '0' && last <= '9')
}

func azureWebPubSubAccountKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + name)
}

func azureWebPubSubChildKey(subscriptionID, resourceGroup, accountName, replicaName, collection, name string) string {
	return strings.ToLower(subscriptionID + "/" + resourceGroup + "/" + accountName + "/" + replicaName + "/" + collection + "/" + name)
}

func azureWebPubSubAccountID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.SignalRService/webPubSub/%s", subscriptionID, resourceGroup, name)
}

func azureWebPubSubChildParentID(child *azureWebPubSubChildResource) string {
	if child.Collection == "replicaSharedPrivateLinkResources" {
		return azureWebPubSubAccountID(child.SubscriptionID, child.ResourceGroup, child.AccountName) + "/replicas/" + child.ReplicaName
	}
	return azureWebPubSubAccountID(child.SubscriptionID, child.ResourceGroup, child.AccountName)
}

func azureWebPubSubChildID(child *azureWebPubSubChildResource) string {
	parent := azureWebPubSubChildParentID(child)
	switch child.Collection {
	case "replicaSharedPrivateLinkResources":
		return parent + "/sharedPrivateLinkResources/" + child.Name
	default:
		return parent + "/" + child.Collection + "/" + child.Name
	}
}

func azureWebPubSubChildType(child *azureWebPubSubChildResource) string {
	base := "Microsoft.SignalRService/webPubSub"
	switch child.Collection {
	case "replicaSharedPrivateLinkResources":
		return base + "/replicas/sharedPrivateLinkResources"
	default:
		return base + "/" + child.Collection
	}
}

func azureWebPubSubConnectionString(name, key string) string {
	return fmt.Sprintf("Endpoint=https://%s.webpubsub.azure.local;AccessKey=%s;Version=1.0;", name, key)
}

func azureWebPubSubStableSecret(kind, subscriptionID, resourceGroup, name string, now time.Time) string {
	sum := sha1.Sum([]byte(kind + "/" + subscriptionID + "/" + resourceGroup + "/" + name + "/" + now.Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:16])
}

func azureWebPubSubETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureWebPubSubString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func azureWebPubSubSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":      createdAt.Format(time.RFC3339),
		"createdBy":      "stackyard",
		"lastModifiedAt": updatedAt.Format(time.RFC3339),
		"lastModifiedBy": "stackyard",
	}
}

func azureWebPubSubStringMap(value any) map[string]string {
	out := map[string]string{}
	if typed, ok := value.(map[string]any); ok {
		for key, raw := range typed {
			out[key] = strings.TrimSpace(azureWebPubSubString(raw))
		}
	}
	if typed, ok := value.(map[string]string); ok {
		for key, raw := range typed {
			out[key] = strings.TrimSpace(raw)
		}
	}
	return out
}

func azureWebPubSubObject(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return azureWebPubSubCloneObject(typed)
	}
	return map[string]any{}
}

func azureWebPubSubCloneObject(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, raw := range value {
		switch typed := raw.(type) {
		case map[string]any:
			out[key] = azureWebPubSubCloneObject(typed)
		case []any:
			copied := make([]any, len(typed))
			copy(copied, typed)
			out[key] = copied
		default:
			out[key] = typed
		}
	}
	return out
}

func azureWebPubSubMergeObjects(base, patch map[string]any) map[string]any {
	out := azureWebPubSubCloneObject(base)
	for key, value := range patch {
		if current, ok := out[key].(map[string]any); ok {
			if incoming, ok := value.(map[string]any); ok {
				out[key] = azureWebPubSubMergeObjects(current, incoming)
				continue
			}
		}
		out[key] = value
	}
	return out
}
