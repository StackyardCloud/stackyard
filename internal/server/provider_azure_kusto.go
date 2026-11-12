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

const azureKustoAPIVersion = "2025-02-14"

type azureKustoCluster struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Identity       map[string]any
	SKU            map[string]any
	Zones          []string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureKustoResource struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	DatabaseName   string
	Collection     string
	Name           string
	Location       string
	Kind           string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureKustoOperationResult struct {
	ID             string
	SubscriptionID string
	Location       string
	ResponseType   string
	Status         string
	Payload        map[string]any
	CreatedAt      time.Time
}

var azureKustoClusterCollections = map[string]string{
	"databases":                      "databases",
	"attacheddatabaseconfigurations": "attachedDatabaseConfigurations",
	"managedprivateendpoints":        "managedPrivateEndpoints",
	"privateendpointconnections":     "privateEndpointConnections",
	"privatelinkresources":           "privateLinkResources",
	"sandboxcustomimages":            "sandboxCustomImages",
	"principalassignments":           "principalAssignments",
}

var azureKustoDatabaseCollections = map[string]string{
	"dataconnections":      "dataConnections",
	"scripts":              "scripts",
	"principalassignments": "principalAssignments",
}

func (s *Server) handleAzureKustoRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsKustoPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureKustoAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Kusto") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureKustoOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureKustoRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		s.handleAzureKustoSubscriptionRoute(w, r, path, subscriptionID, tail)
		return true
	}

	s.handleAzureKustoResourceGroupRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsKustoPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Kusto") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	subscriptionID, resourceGroup, tail, ok := azureKustoRouteContext(segments)
	if !ok || subscriptionID == "" || len(tail) == 0 {
		return false
	}
	if resourceGroup == "" {
		switch strings.ToLower(strings.TrimSpace(tail[0])) {
		case "skus", "locations", "clusters":
			return true
		}
		return false
	}
	return strings.EqualFold(tail[0], "clusters")
}

func azureKustoRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.Kusto") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureKustoAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureKustoSubscriptionRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "clusters") && r.Method == http.MethodGet:
		s.handleAzureKustoListClustersBySubscription(w, subscriptionID)
		return
	case len(tail) == 1 && strings.EqualFold(tail[0], "skus") && r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"value": azureKustoSKUsFixture(subscriptionID, "")})
		return
	case len(tail) == 3 &&
		strings.EqualFold(tail[0], "locations") &&
		strings.TrimSpace(tail[1]) != "" &&
		strings.EqualFold(tail[2], "skus") &&
		r.Method == http.MethodGet:
		respondJSON(w, http.StatusOK, map[string]any{"value": azureKustoSKUsFixture(subscriptionID, tail[1])})
		return
	case len(tail) == 4 &&
		strings.EqualFold(tail[0], "locations") &&
		strings.TrimSpace(tail[1]) != "" &&
		strings.EqualFold(tail[2], "operationResults") &&
		r.Method == http.MethodGet:
		s.handleAzureKustoGetOperationResult(w, path, subscriptionID, tail[1], tail[3])
		return
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureKustoResourceGroupRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "clusters") && r.Method == http.MethodGet {
		s.handleAzureKustoListClustersByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if len(tail) < 2 || !strings.EqualFold(tail[0], "clusters") {
		respondAzureImplemented(w, path)
		return
	}

	clusterName := strings.TrimSpace(tail[1])
	if clusterName == "" {
		respondAzureInvalidRequest(w, path, "cluster name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureKustoGetCluster(w, path, subscriptionID, resourceGroup, clusterName, false)
		case http.MethodHead:
			s.handleAzureKustoGetCluster(w, path, subscriptionID, resourceGroup, clusterName, true)
		case http.MethodPut:
			s.handleAzureKustoCreateOrUpdateCluster(w, r, path, subscriptionID, resourceGroup, clusterName)
		case http.MethodPatch:
			s.handleAzureKustoPatchCluster(w, r, path, subscriptionID, resourceGroup, clusterName)
		case http.MethodDelete:
			s.handleAzureKustoDeleteCluster(w, path, subscriptionID, resourceGroup, clusterName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 3 {
		if strings.EqualFold(tail[2], "listSkus") && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureKustoSKUsFixture(subscriptionID, "")})
			return
		}
		if r.Method == http.MethodPost {
			if s.handleAzureKustoClusterAction(w, r, path, subscriptionID, resourceGroup, clusterName, tail[2]) {
				return
			}
		}
		if collection, ok := azureKustoClusterCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok && r.Method == http.MethodGet {
			s.handleAzureKustoListClusterChildren(w, path, subscriptionID, resourceGroup, clusterName, collection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	if len(tail) == 4 {
		collection, ok := azureKustoClusterCollections[strings.ToLower(strings.TrimSpace(tail[2]))]
		if !ok {
			respondAzureImplemented(w, path)
			return
		}
		name := strings.TrimSpace(tail[3])
		if name == "" {
			respondAzureInvalidRequest(w, path, "resource name is required")
			return
		}
		s.handleAzureKustoClusterChild(w, r, path, subscriptionID, resourceGroup, clusterName, collection, name)
		return
	}

	if len(tail) >= 4 && strings.EqualFold(tail[2], "databases") {
		databaseName := strings.TrimSpace(tail[3])
		if databaseName == "" {
			respondAzureInvalidRequest(w, path, "database name is required")
			return
		}
		switch len(tail) {
		case 4:
			s.handleAzureKustoDatabase(w, r, path, subscriptionID, resourceGroup, clusterName, databaseName)
			return
		case 5:
			switch {
			case strings.EqualFold(tail[4], "inviteFollower") && r.Method == http.MethodPost:
				s.handleAzureKustoInviteFollower(w, path, subscriptionID, resourceGroup, clusterName, databaseName)
				return
			case strings.EqualFold(tail[4], "addPrincipals") && r.Method == http.MethodPost:
				s.handleAzureKustoDatabasePrincipalsMutation(w, r, path, subscriptionID, resourceGroup, clusterName, databaseName, true)
				return
			case strings.EqualFold(tail[4], "removePrincipals") && r.Method == http.MethodPost:
				s.handleAzureKustoDatabasePrincipalsMutation(w, r, path, subscriptionID, resourceGroup, clusterName, databaseName, false)
				return
			}
			if collection, ok := azureKustoDatabaseCollections[strings.ToLower(strings.TrimSpace(tail[4]))]; ok && r.Method == http.MethodGet {
				s.handleAzureKustoListDatabaseChildren(w, path, subscriptionID, resourceGroup, clusterName, databaseName, collection)
				return
			}
		case 6:
			if collection, ok := azureKustoDatabaseCollections[strings.ToLower(strings.TrimSpace(tail[4]))]; ok {
				name := strings.TrimSpace(tail[5])
				if name == "" {
					respondAzureInvalidRequest(w, path, "resource name is required")
					return
				}
				s.handleAzureKustoDatabaseChild(w, r, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
				return
			}
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureKustoCreateOrUpdateCluster(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName string) {
	body, payload, ok := azureKustoReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureKustoString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster := s.azureKustoClusters[key]
	status := http.StatusOK
	createdAt := now
	if cluster == nil {
		cluster = &azureKustoCluster{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           clusterName,
		}
		status = http.StatusCreated
	} else {
		createdAt = cluster.CreatedAt
	}

	cluster.Location = location
	cluster.Tags = azureKustoStringMap(payload["tags"])
	cluster.Identity = azureKustoObject(payload["identity"])
	cluster.SKU = azureKustoObject(payload["sku"])
	cluster.Zones = azureKustoStringSlice(payload["zones"])
	cluster.Properties = azureKustoNormalizeClusterProperties(clusterName, location, azureKustoObject(payload["properties"]), createdAt, now)
	cluster.CreatedAt = createdAt
	cluster.UpdatedAt = now
	cluster.ETag = azureKustoETag(key, body, now)
	s.azureKustoClusters[key] = cluster

	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   "CreateOrUpdateCluster",
	})
	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("ETag", cluster.ETag)
	respondJSON(w, status, azureKustoClusterFixture(cluster))
}

func (s *Server) handleAzureKustoPatchCluster(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName string) {
	body, payload, ok := azureKustoReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster := s.azureKustoClusters[key]
	if cluster == nil {
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, clusterName))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["location"]; ok {
		location := azureKustoString(payload["location"])
		if location == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		cluster.Location = location
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		cluster.Tags = azureKustoStringMap(payload["tags"])
		updated = true
	}
	if _, ok := payload["identity"]; ok {
		cluster.Identity = azureKustoObject(payload["identity"])
		updated = true
	}
	if _, ok := payload["sku"]; ok {
		cluster.SKU = azureKustoObject(payload["sku"])
		updated = true
	}
	if _, ok := payload["zones"]; ok {
		cluster.Zones = azureKustoStringSlice(payload["zones"])
		updated = true
	}
	if properties, ok := payload["properties"]; ok {
		cluster.Properties = azureKustoMergeObjects(cluster.Properties, azureKustoObject(properties))
		cluster.Properties = azureKustoNormalizeClusterProperties(cluster.Name, cluster.Location, cluster.Properties, cluster.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	cluster.UpdatedAt = now
	cluster.ETag = azureKustoETag(key, body, now)
	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, cluster.Location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   "UpdateCluster",
	})
	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("ETag", cluster.ETag)
	respondJSON(w, http.StatusOK, azureKustoClusterFixture(cluster))
}

func (s *Server) handleAzureKustoDeleteCluster(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName string) {
	key := azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)

	s.providerStorageMu.Lock()
	cluster := s.azureKustoClusters[key]
	if cluster == nil {
		s.providerStorageMu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureKustoClusters, key)
	for childKey, resource := range s.azureKustoResources {
		if resource.SubscriptionID == subscriptionID && resource.ResourceGroup == resourceGroup && resource.ClusterName == clusterName {
			delete(s.azureKustoResources, childKey)
		}
	}
	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, cluster.Location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   "DeleteCluster",
	})
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("Retry-After", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureKustoGetCluster(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName string, headOnly bool) {
	s.providerStorageMu.Lock()
	cluster := s.azureKustoClusters[azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)]
	s.providerStorageMu.Unlock()
	if cluster == nil {
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, clusterName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("ETag", cluster.ETag)
	respondJSON(w, http.StatusOK, azureKustoClusterFixture(cluster))
}

func (s *Server) handleAzureKustoListClustersByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]*azureKustoCluster, 0, len(s.azureKustoClusters))
	for _, cluster := range s.azureKustoClusters {
		if cluster.SubscriptionID == subscriptionID && strings.EqualFold(cluster.ResourceGroup, resourceGroup) {
			items = append(items, cluster)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	value := make([]any, 0, len(items))
	for _, cluster := range items {
		value = append(value, azureKustoClusterFixture(cluster))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureKustoListClustersBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureKustoCluster, 0, len(s.azureKustoClusters))
	for _, cluster := range s.azureKustoClusters {
		if cluster.SubscriptionID == subscriptionID {
			items = append(items, cluster)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	value := make([]any, 0, len(items))
	for _, cluster := range items {
		value = append(value, azureKustoClusterFixture(cluster))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureKustoClusterAction(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, action string) bool {
	key := azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)

	s.providerStorageMu.Lock()
	cluster := s.azureKustoClusters[key]
	if cluster == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, clusterName))
		return true
	}

	actionKey := strings.ToLower(strings.TrimSpace(action))
	now := time.Now().UTC()
	response := map[string]any{}

	switch actionKey {
	case "start":
		cluster.Properties["state"] = "Running"
		response = azureKustoClusterFixture(cluster)
	case "stop":
		cluster.Properties["state"] = "Stopped"
		response = azureKustoClusterFixture(cluster)
	case "migrate":
		response = map[string]any{
			"id":               azureKustoClusterID(cluster),
			"name":             cluster.Name,
			"status":           "Succeeded",
			"migrationState":   "Completed",
			"targetApiVersion": azureKustoAPIVersion,
		}
	case "diagnosevirtualnetwork":
		response = map[string]any{
			"findings": []any{
				map[string]any{
					"severity": "Info",
					"message":  "Virtual network configuration is reachable in Stackyard.",
				},
			},
			"status": "Succeeded",
		}
	case "addlanguageextensions":
		body, payload, ok := azureKustoReadRequestBody(w, r, path)
		if !ok {
			s.providerStorageMu.Unlock()
			return true
		}
		extensions := azureKustoStringSlice(payload["languageExtensionsToAdd"])
		cluster.Properties["languageExtensions"] = azureKustoAppendUniqueStrings(azureKustoAnyStringSlice(cluster.Properties["languageExtensions"]), extensions)
		cluster.ETag = azureKustoETag(key, body, now)
		response = azureKustoClusterFixture(cluster)
	case "removelanguageextensions":
		body, payload, ok := azureKustoReadRequestBody(w, r, path)
		if !ok {
			s.providerStorageMu.Unlock()
			return true
		}
		remove := azureKustoStringSlice(payload["languageExtensionsToRemove"])
		cluster.Properties["languageExtensions"] = azureKustoRemoveStrings(azureKustoAnyStringSlice(cluster.Properties["languageExtensions"]), remove)
		cluster.ETag = azureKustoETag(key, body, now)
		response = azureKustoClusterFixture(cluster)
	case "addcalloutpolicies":
		body, payload, ok := azureKustoReadRequestBody(w, r, path)
		if !ok {
			s.providerStorageMu.Unlock()
			return true
		}
		policies := azureKustoAnySlice(payload["calloutPolicies"])
		cluster.Properties["calloutPolicies"] = append(azureKustoAnySlice(cluster.Properties["calloutPolicies"]), policies...)
		cluster.ETag = azureKustoETag(key, body, now)
		response = azureKustoClusterFixture(cluster)
	case "removecalloutpolicy":
		body, payload, ok := azureKustoReadRequestBody(w, r, path)
		if !ok {
			s.providerStorageMu.Unlock()
			return true
		}
		name := azureKustoString(payload["calloutPolicyName"])
		policies := make([]any, 0)
		for _, item := range azureKustoAnySlice(cluster.Properties["calloutPolicies"]) {
			policy, ok := item.(map[string]any)
			if !ok || !strings.EqualFold(azureKustoString(policy["calloutPolicyName"]), name) {
				policies = append(policies, item)
			}
		}
		cluster.Properties["calloutPolicies"] = policies
		cluster.ETag = azureKustoETag(key, body, now)
		response = azureKustoClusterFixture(cluster)
	case "detachfollowerdatabases":
		response = map[string]any{
			"status":             "Succeeded",
			"detachedDatabases":  azureKustoAnySliceMap("attachedDatabaseConfigurations", s.azureKustoResources, subscriptionID, resourceGroup, clusterName),
			"clusterResourceId":  azureKustoClusterID(cluster),
			"operationCompleted": true,
		}
	default:
		s.providerStorageMu.Unlock()
		return false
	}

	cluster.Properties = azureKustoNormalizeClusterProperties(cluster.Name, cluster.Location, cluster.Properties, cluster.CreatedAt, now)
	cluster.UpdatedAt = now
	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, cluster.Location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   action,
	})
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	respondJSON(w, http.StatusOK, response)
	return true
}

func (s *Server) handleAzureKustoDatabase(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, databaseName string) {
	key := azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)
	switch r.Method {
	case http.MethodGet:
		s.providerStorageMu.Lock()
		resource := s.azureKustoResources[key]
		s.providerStorageMu.Unlock()
		if resource == nil {
			respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, databaseName))
			return
		}
		respondJSON(w, http.StatusOK, azureKustoResourceFixture(resource))
	case http.MethodHead:
		s.providerStorageMu.Lock()
		resource := s.azureKustoResources[key]
		s.providerStorageMu.Unlock()
		if resource == nil {
			respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, databaseName))
			return
		}
		w.WriteHeader(http.StatusOK)
	case http.MethodPut:
		s.handleAzureKustoCreateOrUpdateResource(w, r, path, subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)
	case http.MethodPatch:
		s.handleAzureKustoPatchResource(w, r, path, subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)
	case http.MethodDelete:
		s.handleAzureKustoDeleteResource(w, path, subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureKustoClusterChild(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, collection, name string) {
	if strings.EqualFold(collection, "privateLinkResources") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		s.handleAzureKustoPrivateLinkResourceGet(w, path, subscriptionID, resourceGroup, clusterName, name, r.Method == http.MethodHead)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureKustoGetResource(w, path, subscriptionID, resourceGroup, clusterName, "", collection, name, false)
	case http.MethodHead:
		s.handleAzureKustoGetResource(w, path, subscriptionID, resourceGroup, clusterName, "", collection, name, true)
	case http.MethodPut:
		s.handleAzureKustoCreateOrUpdateResource(w, r, path, subscriptionID, resourceGroup, clusterName, "", collection, name)
	case http.MethodPatch:
		s.handleAzureKustoPatchResource(w, r, path, subscriptionID, resourceGroup, clusterName, "", collection, name)
	case http.MethodDelete:
		s.handleAzureKustoDeleteResource(w, path, subscriptionID, resourceGroup, clusterName, "", collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureKustoDatabaseChild(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name string) {
	switch r.Method {
	case http.MethodGet:
		s.handleAzureKustoGetResource(w, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name, false)
	case http.MethodHead:
		s.handleAzureKustoGetResource(w, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name, true)
	case http.MethodPut:
		s.handleAzureKustoCreateOrUpdateResource(w, r, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
	case http.MethodPatch:
		s.handleAzureKustoPatchResource(w, r, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
	case http.MethodDelete:
		s.handleAzureKustoDeleteResource(w, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureKustoCreateOrUpdateResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name string) {
	body, payload, ok := azureKustoReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	cluster := s.azureKustoClusters[azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)]
	if cluster == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, clusterName))
		return
	}
	if databaseName != "" && collection != "databases" {
		database := s.azureKustoResources[azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)]
		if database == nil {
			s.providerStorageMu.Unlock()
			respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, databaseName))
			return
		}
	}

	properties := azureKustoObject(payload["properties"])
	if collection != "privateLinkResources" && collection != "databases" && len(properties) == 0 {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}
	if collection == "databases" && strings.TrimSpace(azureKustoString(payload["kind"])) == "" {
		payload["kind"] = "ReadWrite"
	}

	now := time.Now().UTC()
	key := azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
	resource := s.azureKustoResources[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureKustoResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			DatabaseName:   databaseName,
			Collection:     collection,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}

	resource.Location = azureKustoCoalesceLocation(azureKustoString(payload["location"]), cluster.Location)
	resource.Kind = azureKustoString(payload["kind"])
	resource.Tags = azureKustoStringMap(payload["tags"])
	resource.Properties = azureKustoNormalizeResourceProperties(collection, name, resource.Kind, properties, createdAt, now)
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureKustoETag(key, body, now)
	s.azureKustoResources[key] = resource

	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, resource.Location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   "CreateOrUpdate" + collection,
	})
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("ETag", resource.ETag)
	respondJSON(w, status, azureKustoResourceFixture(resource))
}

func (s *Server) handleAzureKustoPatchResource(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name string) {
	body, payload, ok := azureKustoReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	resource := s.azureKustoResources[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		s.providerStorageMu.Unlock()
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	if _, ok := payload["location"]; ok {
		resource.Location = azureKustoCoalesceLocation(azureKustoString(payload["location"]), resource.Location)
	}
	if _, ok := payload["kind"]; ok {
		resource.Kind = azureKustoString(payload["kind"])
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureKustoStringMap(payload["tags"])
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureKustoMergeObjects(resource.Properties, azureKustoObject(properties))
		resource.Properties = azureKustoNormalizeResourceProperties(collection, name, resource.Kind, resource.Properties, resource.CreatedAt, now)
	}
	resource.UpdatedAt = now
	resource.ETag = azureKustoETag(key, body, now)
	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, resource.Location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   "Update" + collection,
	})
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("ETag", resource.ETag)
	respondJSON(w, http.StatusOK, azureKustoResourceFixture(resource))
}

func (s *Server) handleAzureKustoDeleteResource(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name string) {
	key := azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, databaseName, collection, name)
	s.providerStorageMu.Lock()
	resource := s.azureKustoResources[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureKustoResources, key)
	for childKey, child := range s.azureKustoResources {
		if child.SubscriptionID == subscriptionID && child.ResourceGroup == resourceGroup && child.ClusterName == clusterName && child.DatabaseName == name && collection == "databases" {
			delete(s.azureKustoResources, childKey)
		}
	}
	operationPath := s.azureKustoRegisterOperationResult(subscriptionID, resource.Location, "OperationResult", map[string]any{
		"status": "Succeeded",
		"name":   "Delete" + collection,
	})
	s.providerStorageMu.Unlock()

	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("Retry-After", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureKustoGetResource(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, databaseName, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureKustoResources[azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, databaseName, collection, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("ETag", resource.ETag)
	respondJSON(w, http.StatusOK, azureKustoResourceFixture(resource))
}

func (s *Server) handleAzureKustoListClusterChildren(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, collection string) {
	if strings.EqualFold(collection, "privateLinkResources") {
		respondJSON(w, http.StatusOK, map[string]any{"value": []any{azureKustoPrivateLinkResourceFixture(subscriptionID, resourceGroup, clusterName, "cluster")}})
		return
	}
	s.providerStorageMu.Lock()
	cluster := s.azureKustoClusters[azureKustoClusterKey(subscriptionID, resourceGroup, clusterName)]
	if cluster == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, clusterName))
		return
	}
	items := make([]*azureKustoResource, 0)
	for _, resource := range s.azureKustoResources {
		if resource.SubscriptionID == subscriptionID && resource.ResourceGroup == resourceGroup && resource.ClusterName == clusterName && resource.DatabaseName == "" && strings.EqualFold(resource.Collection, collection) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	value := make([]any, 0, len(items))
	for _, resource := range items {
		value = append(value, azureKustoResourceFixture(resource))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureKustoListDatabaseChildren(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, databaseName, collection string) {
	s.providerStorageMu.Lock()
	database := s.azureKustoResources[azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)]
	if database == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, databaseName))
		return
	}
	items := make([]*azureKustoResource, 0)
	for _, resource := range s.azureKustoResources {
		if resource.SubscriptionID == subscriptionID && resource.ResourceGroup == resourceGroup && resource.ClusterName == clusterName && resource.DatabaseName == databaseName && strings.EqualFold(resource.Collection, collection) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	value := make([]any, 0, len(items))
	for _, resource := range items {
		value = append(value, azureKustoResourceFixture(resource))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureKustoInviteFollower(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, databaseName string) {
	s.providerStorageMu.Lock()
	resource := s.azureKustoResources[azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, databaseName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"databaseName":    databaseName,
		"invitationToken": "stackyard-" + azureKustoStableToken(subscriptionID+"-"+resourceGroup+"-"+clusterName+"-"+databaseName),
		"status":          "Succeeded",
	})
}

func (s *Server) handleAzureKustoDatabasePrincipalsMutation(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, databaseName string, add bool) {
	body, payload, ok := azureKustoReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, "", "databases", databaseName)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	resource := s.azureKustoResources[key]
	if resource == nil {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, databaseName))
		return
	}

	requestPrincipals := azureKustoAnySlice(payload["value"])
	current := azureKustoAnySlice(resource.Properties["principals"])
	if add {
		current = append(current, requestPrincipals...)
	} else {
		removeIDs := map[string]struct{}{}
		for _, item := range requestPrincipals {
			if principal, ok := item.(map[string]any); ok {
				removeIDs[strings.ToLower(azureKustoString(principal["principalId"]))] = struct{}{}
			}
		}
		filtered := make([]any, 0, len(current))
		for _, item := range current {
			principal, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if _, found := removeIDs[strings.ToLower(azureKustoString(principal["principalId"]))]; !found {
				filtered = append(filtered, item)
			}
		}
		current = filtered
	}
	resource.Properties["principals"] = current
	resource.UpdatedAt = now
	resource.ETag = azureKustoETag(key, body, now)
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{"value": current})
}

func (s *Server) handleAzureKustoPrivateLinkResourceGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string, headOnly bool) {
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureKustoPrivateLinkResourceFixture(subscriptionID, resourceGroup, clusterName, name))
}

func (s *Server) handleAzureKustoGetOperationResult(w http.ResponseWriter, path, subscriptionID, location, operationID string) {
	s.providerStorageMu.Lock()
	record := s.azureKustoOperationResults[azureKustoOperationKey(subscriptionID, location, operationID)]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureKustoNotFound(path, operationID))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"id":         "/subscriptions/" + subscriptionID + "/providers/Microsoft.Kusto/locations/" + location + "/operationResults/" + operationID,
		"name":       operationID,
		"status":     record.Status,
		"startTime":  record.CreatedAt.Format(time.RFC3339),
		"endTime":    record.CreatedAt.Format(time.RFC3339),
		"properties": azureKustoCopyObject(record.Payload),
	})
}

func (s *Server) azureKustoRegisterOperationResult(subscriptionID, location, responseType string, payload map[string]any) string {
	now := time.Now().UTC()
	location = azureKustoCoalesceLocation(location, "eastus")
	keySource := fmt.Sprintf("%s|%s|%s|%d", subscriptionID, location, responseType, now.UnixNano())
	sum := sha1.Sum([]byte(keySource))
	id := hex.EncodeToString(sum[:8])
	record := &azureKustoOperationResult{
		ID:             id,
		SubscriptionID: subscriptionID,
		Location:       location,
		ResponseType:   responseType,
		Status:         "Succeeded",
		Payload:        azureKustoCopyObject(payload),
		CreatedAt:      now,
	}
	s.azureKustoOperationResults[azureKustoOperationKey(subscriptionID, location, id)] = record
	return "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.Kusto/locations/" + location + "/operationResults/" + id + "?api-version=" + azureKustoAPIVersion
}

func azureKustoClusterKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID)) + "|" + strings.ToLower(strings.TrimSpace(resourceGroup)) + "|" + strings.ToLower(strings.TrimSpace(name))
}

func azureKustoResourceKey(subscriptionID, resourceGroup, clusterName, databaseName, collection, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID)) + "|" + strings.ToLower(strings.TrimSpace(resourceGroup)) + "|" + strings.ToLower(strings.TrimSpace(clusterName)) + "|" + strings.ToLower(strings.TrimSpace(databaseName)) + "|" + strings.ToLower(strings.TrimSpace(collection)) + "|" + strings.ToLower(strings.TrimSpace(name))
}

func azureKustoOperationKey(subscriptionID, location, operationID string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID)) + "|" + strings.ToLower(strings.TrimSpace(location)) + "|" + strings.ToLower(strings.TrimSpace(operationID))
}

func azureKustoClusterID(cluster *azureKustoCluster) string {
	return "/subscriptions/" + cluster.SubscriptionID + "/resourceGroups/" + cluster.ResourceGroup + "/providers/Microsoft.Kusto/clusters/" + cluster.Name
}

func azureKustoResourceFixture(resource *azureKustoResource) map[string]any {
	resourceType := "Microsoft.Kusto/clusters/" + resource.Collection
	id := "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.Kusto/clusters/" + resource.ClusterName + "/" + resource.Collection + "/" + resource.Name
	if resource.Collection == "databases" {
		resourceType = "Microsoft.Kusto/clusters/databases"
	}
	if resource.DatabaseName != "" {
		resourceType = "Microsoft.Kusto/clusters/databases/" + resource.Collection
		id = "/subscriptions/" + resource.SubscriptionID + "/resourceGroups/" + resource.ResourceGroup + "/providers/Microsoft.Kusto/clusters/" + resource.ClusterName + "/databases/" + resource.DatabaseName + "/" + resource.Collection + "/" + resource.Name
	}
	fixture := map[string]any{
		"id":         id,
		"name":       resource.Name,
		"type":       resourceType,
		"location":   resource.Location,
		"tags":       azureKustoStringMapToAny(resource.Tags),
		"systemData": azureKustoSystemDataFixture(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureKustoCopyObject(resource.Properties),
	}
	if resource.Kind != "" {
		fixture["kind"] = resource.Kind
	}
	if resource.ETag != "" {
		fixture["etag"] = resource.ETag
	}
	return fixture
}

func azureKustoClusterFixture(cluster *azureKustoCluster) map[string]any {
	return map[string]any{
		"id":         azureKustoClusterID(cluster),
		"name":       cluster.Name,
		"type":       "Microsoft.Kusto/clusters",
		"location":   cluster.Location,
		"tags":       azureKustoStringMapToAny(cluster.Tags),
		"identity":   azureKustoCopyObject(cluster.Identity),
		"sku":        azureKustoCopyObject(cluster.SKU),
		"zones":      append([]string(nil), cluster.Zones...),
		"etag":       cluster.ETag,
		"systemData": azureKustoSystemDataFixture(cluster.CreatedAt, cluster.UpdatedAt),
		"properties": azureKustoCopyObject(cluster.Properties),
	}
}

func azureKustoPrivateLinkResourceFixture(subscriptionID, resourceGroup, clusterName, name string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/privateLinkResources/" + name,
		"name": name,
		"type": "Microsoft.Kusto/clusters/privateLinkResources",
		"properties": map[string]any{
			"groupId":                           name,
			"requiredMembers":                   []any{"cluster"},
			"requiredZoneNames":                 []any{"privatelink.kusto.windows.net"},
			"shareablePrivateLinkResourceTypes": []any{},
		},
	}
}

func azureKustoOperationsFixture() []any {
	return []any{
		azureKustoOperation("Microsoft.Kusto/clusters/read", "Read clusters"),
		azureKustoOperation("Microsoft.Kusto/clusters/write", "Create or update clusters"),
		azureKustoOperation("Microsoft.Kusto/clusters/delete", "Delete clusters"),
		azureKustoOperation("Microsoft.Kusto/clusters/start/action", "Start clusters"),
		azureKustoOperation("Microsoft.Kusto/clusters/stop/action", "Stop clusters"),
		azureKustoOperation("Microsoft.Kusto/clusters/databases/read", "Read databases"),
		azureKustoOperation("Microsoft.Kusto/clusters/databases/write", "Create or update databases"),
		azureKustoOperation("Microsoft.Kusto/clusters/databases/delete", "Delete databases"),
		azureKustoOperation("Microsoft.Kusto/clusters/databases/dataConnections/write", "Create or update data connections"),
		azureKustoOperation("Microsoft.Kusto/clusters/databases/scripts/write", "Create or update scripts"),
	}
}

func azureKustoOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":    "Microsoft Kusto",
			"resource":    "Clusters",
			"operation":   description,
			"description": description,
		},
		"origin": "user,system",
	}
}

func azureKustoSKUsFixture(subscriptionID, location string) []any {
	scope := strings.TrimSpace(location)
	if scope == "" {
		scope = "global"
	}
	return []any{
		map[string]any{
			"name": "Standard_D13_v2",
			"tier": "Standard",
			"locations": []any{
				azureKustoCoalesceLocation(location, "eastus"),
				"westus2",
			},
			"resourceType": "clusters",
			"capacity": map[string]any{
				"default":   2,
				"minimum":   2,
				"maximum":   100,
				"scaleType": "Automatic",
			},
			"locationInfo": []any{
				map[string]any{
					"location": azureKustoCoalesceLocation(location, "eastus"),
					"zones":    []any{"1", "2", "3"},
				},
			},
			"restrictions": []any{
				map[string]any{
					"type":       "None",
					"reasonCode": "None",
					"values":     []any{scope},
				},
			},
		},
	}
}

func azureKustoNormalizeClusterProperties(name, location string, properties map[string]any, createdAt, now time.Time) map[string]any {
	out := azureKustoCopyObject(properties)
	out["provisioningState"] = "Succeeded"
	out["state"] = azureKustoCoalesceString(azureKustoString(out["state"]), "Running")
	out["uri"] = azureKustoCoalesceString(azureKustoString(out["uri"]), "https://"+name+"."+azureKustoCoalesceLocation(location, "eastus")+".kusto.windows.net")
	out["dataIngestionURI"] = azureKustoCoalesceString(azureKustoString(out["dataIngestionURI"]), "https://ingest-"+name+"."+azureKustoCoalesceLocation(location, "eastus")+".kusto.windows.net")
	if _, ok := out["languageExtensions"]; !ok {
		out["languageExtensions"] = []any{}
	}
	if _, ok := out["calloutPolicies"]; !ok {
		out["calloutPolicies"] = []any{}
	}
	if _, ok := out["stateChangeReason"]; !ok {
		out["stateChangeReason"] = map[string]any{
			"reason":    "None",
			"message":   "Cluster is healthy",
			"timestamp": now.Format(time.RFC3339),
		}
	}
	return out
}

func azureKustoNormalizeResourceProperties(collection, name, kind string, properties map[string]any, createdAt, now time.Time) map[string]any {
	out := azureKustoCopyObject(properties)
	out["provisioningState"] = "Succeeded"
	switch collection {
	case "databases":
		out["softDeletePeriod"] = azureKustoCoalesceString(azureKustoString(out["softDeletePeriod"]), "P365D")
		out["hotCachePeriod"] = azureKustoCoalesceString(azureKustoString(out["hotCachePeriod"]), "P7D")
		out["kind"] = azureKustoCoalesceString(kind, "ReadWrite")
		if _, ok := out["principals"]; !ok {
			out["principals"] = []any{}
		}
	case "dataConnections":
		out["kind"] = azureKustoCoalesceString(kind, "EventHub")
		out["consumerGroup"] = azureKustoCoalesceString(azureKustoString(out["consumerGroup"]), "$Default")
		out["eventHubResourceId"] = azureKustoCoalesceString(azureKustoString(out["eventHubResourceId"]), "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.EventHub/namespaces/ns/eventhubs/eh")
	case "scripts":
		out["scriptLevel"] = azureKustoCoalesceString(azureKustoString(out["scriptLevel"]), "Database")
		out["continueOnErrors"] = azureKustoCoalesceBool(out["continueOnErrors"], false)
		out["forceUpdateTag"] = azureKustoCoalesceString(azureKustoString(out["forceUpdateTag"]), "stackyard")
	case "attachedDatabaseConfigurations":
		if _, ok := out["attachedDatabaseNames"]; !ok {
			out["attachedDatabaseNames"] = []any{name + "-db"}
		}
	case "managedPrivateEndpoints":
		out["groupId"] = azureKustoCoalesceString(azureKustoString(out["groupId"]), "cluster")
		out["privateLinkResourceId"] = azureKustoCoalesceString(azureKustoString(out["privateLinkResourceId"]), "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa")
	case "privateEndpointConnections":
		if _, ok := out["privateEndpoint"]; !ok {
			out["privateEndpoint"] = map[string]any{
				"id": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Network/privateEndpoints/pe-1",
			}
		}
		if _, ok := out["privateLinkServiceConnectionState"]; !ok {
			out["privateLinkServiceConnectionState"] = map[string]any{
				"status":          "Approved",
				"description":     "Approved by Stackyard",
				"actionsRequired": "None",
			}
		}
	case "principalAssignments":
		out["principalId"] = azureKustoCoalesceString(azureKustoString(out["principalId"]), "11111111-1111-1111-1111-111111111111")
		out["principalType"] = azureKustoCoalesceString(azureKustoString(out["principalType"]), "App")
		out["role"] = azureKustoCoalesceString(azureKustoString(out["role"]), "AllDatabasesAdmin")
		out["tenantId"] = azureKustoCoalesceString(azureKustoString(out["tenantId"]), "22222222-2222-2222-2222-222222222222")
	case "sandboxCustomImages":
		out["language"] = azureKustoCoalesceString(azureKustoString(out["language"]), "Python")
		out["imageName"] = azureKustoCoalesceString(azureKustoString(out["imageName"]), name)
	}
	return out
}

func azureKustoNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "NotFound",
			"message": fmt.Sprintf("%s was not found", name),
		},
		"path": path,
	}
}

func azureKustoReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "invalid JSON body")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureKustoETag(key string, body []byte, updatedAt time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+updatedAt.Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureKustoSystemDataFixture(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureKustoString(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func azureKustoObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if object, ok := value.(map[string]any); ok {
		return azureKustoCopyObject(object)
	}
	return map[string]any{}
}

func azureKustoCopyObject(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = azureKustoCopyAny(value)
	}
	return out
}

func azureKustoCopyAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return azureKustoCopyObject(typed)
	case []any:
		out := make([]any, len(typed))
		for idx, item := range typed {
			out[idx] = azureKustoCopyAny(item)
		}
		return out
	default:
		return typed
	}
}

func azureKustoAnySlice(value any) []any {
	if typed, ok := value.([]any); ok {
		out := make([]any, len(typed))
		for idx, item := range typed {
			out[idx] = azureKustoCopyAny(item)
		}
		return out
	}
	return []any{}
}

func azureKustoStringSlice(value any) []string {
	values, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			return append([]string(nil), typed...)
		}
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, item := range values {
		if text := azureKustoString(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func azureKustoAnyStringSlice(value any) []string {
	return azureKustoStringSlice(value)
}

func azureKustoStringMap(value any) map[string]string {
	typed, ok := value.(map[string]any)
	if !ok {
		if direct, ok := value.(map[string]string); ok {
			out := make(map[string]string, len(direct))
			for key, item := range direct {
				if key = strings.TrimSpace(key); key != "" {
					out[key] = item
				}
			}
			return out
		}
		return map[string]string{}
	}
	out := make(map[string]string, len(typed))
	for key, item := range typed {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = azureKustoString(item)
	}
	return out
}

func azureKustoStringMapToAny(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func azureKustoMergeObjects(base, updates map[string]any) map[string]any {
	out := azureKustoCopyObject(base)
	for key, value := range updates {
		existing, ok := out[key].(map[string]any)
		next, nextOK := value.(map[string]any)
		if ok && nextOK {
			out[key] = azureKustoMergeObjects(existing, next)
			continue
		}
		out[key] = azureKustoCopyAny(value)
	}
	return out
}

func azureKustoStableToken(seed string) string {
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:6])
}

func azureKustoAppendUniqueStrings(current, add []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(current)+len(add))
	for _, item := range current {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	for _, item := range add {
		key := strings.ToLower(strings.TrimSpace(item))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func azureKustoRemoveStrings(current, remove []string) []string {
	blocked := map[string]struct{}{}
	for _, item := range remove {
		if key := strings.ToLower(strings.TrimSpace(item)); key != "" {
			blocked[key] = struct{}{}
		}
	}
	out := make([]string, 0, len(current))
	for _, item := range current {
		if _, ok := blocked[strings.ToLower(strings.TrimSpace(item))]; !ok {
			out = append(out, item)
		}
	}
	return out
}

func azureKustoCoalesceLocation(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "eastus"
}

func azureKustoCoalesceString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func azureKustoCoalesceBool(value any, fallback bool) bool {
	if typed, ok := value.(bool); ok {
		return typed
	}
	return fallback
}

func azureKustoAnySliceMap(collection string, resources map[string]*azureKustoResource, subscriptionID, resourceGroup, clusterName string) []any {
	value := make([]any, 0)
	for _, resource := range resources {
		if resource.SubscriptionID == subscriptionID && resource.ResourceGroup == resourceGroup && resource.ClusterName == clusterName && strings.EqualFold(resource.Collection, collection) {
			value = append(value, azureKustoResourceFixture(resource))
		}
	}
	return value
}
