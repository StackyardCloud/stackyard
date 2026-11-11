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

const azureMigrateDiscoveryAPIVersion = "2020-01-01"

var azureMigrateDiscoverySiteCollections = map[string]string{
	"vmwaresites": "VMwareSites",
	"hypervsites": "HyperVSites",
}

type azureMigrateDiscoverySite struct {
	SubscriptionID string
	ResourceGroup  string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureMigrateDiscoveryMachine struct {
	SiteID     string
	Collection string
	Name       string
	Properties map[string]any
	PowerState string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type azureMigrateDiscoveryRunAsAccount struct {
	SiteID     string
	Collection string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type azureMigrateDiscoveryInventoryItem struct {
	SiteID     string
	Collection string
	Name       string
	Properties map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (s *Server) handleAzureMigrateDiscoveryRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsMigrateDiscoveryPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureMigrateDiscoveryAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.OffAzure") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureMigrateDiscoveryOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureMigrateDiscoveryRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	collection, ok := azureMigrateDiscoverySiteCollections[strings.ToLower(strings.TrimSpace(tail[0]))]
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	s.handleAzureMigrateDiscoverySitesRoute(w, r, path, subscriptionID, resourceGroup, collection, tail[1:])
	return true
}

func azureIsMigrateDiscoveryPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.OffAzure") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureMigrateDiscoveryRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	_, found := azureMigrateDiscoverySiteCollections[strings.ToLower(strings.TrimSpace(tail[0]))]
	return found
}

func azureMigrateDiscoveryRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.OffAzure") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureMigrateDiscoveryAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureMigrateDiscoverySitesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			if resourceGroup == "" {
				s.handleAzureMigrateDiscoveryListSitesBySubscription(w, subscriptionID, collection)
				return
			}
			s.handleAzureMigrateDiscoveryListSitesByResourceGroup(w, subscriptionID, resourceGroup, collection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	if resourceGroup == "" {
		respondAzureImplemented(w, path)
		return
	}

	siteName := strings.TrimSpace(tail[0])
	if siteName == "" {
		respondAzureInvalidRequest(w, path, "site name is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureMigrateDiscoveryGetSite(w, path, subscriptionID, resourceGroup, collection, siteName, false)
		case http.MethodHead:
			s.handleAzureMigrateDiscoveryGetSite(w, path, subscriptionID, resourceGroup, collection, siteName, true)
		case http.MethodPut:
			s.handleAzureMigrateDiscoveryCreateOrUpdateSite(w, r, path, subscriptionID, resourceGroup, collection, siteName)
		case http.MethodPatch:
			s.handleAzureMigrateDiscoveryPatchSite(w, r, path, subscriptionID, resourceGroup, collection, siteName)
		case http.MethodDelete:
			s.handleAzureMigrateDiscoveryDeleteSite(w, subscriptionID, resourceGroup, collection, siteName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "summary", "siteusage", "sitehealthsummary":
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			s.handleAzureMigrateDiscoverySiteSummary(w, path, subscriptionID, resourceGroup, collection, siteName, strings.ToLower(strings.TrimSpace(tail[1])))
			return
		}
	case "refresh":
		if r.Method == http.MethodPost {
			s.handleAzureMigrateDiscoverySiteRefresh(w, path, subscriptionID, resourceGroup, collection, siteName)
			return
		}
	case "machines":
		s.handleAzureMigrateDiscoveryMachinesRoute(w, r, path, subscriptionID, resourceGroup, collection, siteName, tail[2:])
		return
	case "runasaccounts":
		s.handleAzureMigrateDiscoveryRunAsAccountsRoute(w, r, path, subscriptionID, resourceGroup, collection, siteName, tail[2:])
		return
	case "vcenters":
		s.handleAzureMigrateDiscoveryInventoryRoute(w, r, path, subscriptionID, resourceGroup, collection, siteName, "vCenters", tail[2:])
		return
	case "hosts":
		s.handleAzureMigrateDiscoveryInventoryRoute(w, r, path, subscriptionID, resourceGroup, collection, siteName, "hosts", tail[2:])
		return
	case "clusters":
		s.handleAzureMigrateDiscoveryInventoryRoute(w, r, path, subscriptionID, resourceGroup, collection, siteName, "clusters", tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateDiscoveryCreateOrUpdateSite(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, siteName string) {
	body, payload, ok := azureMigrateDiscoveryReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureMigrateDiscoveryString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	id := azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName)
	key := azureMigrateDiscoveryStorageKey(id)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureMigrateDiscoverySites[key]
	status := http.StatusOK
	createdAt := now
	if record == nil {
		record = &azureMigrateDiscoverySite{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Collection:     collection,
			Name:           siteName,
		}
		status = http.StatusCreated
	} else {
		createdAt = record.CreatedAt
	}
	record.Location = location
	record.Tags = azureMigrateDiscoveryStringMap(payload["tags"])
	record.Properties = azureMigrateDiscoverySiteProperties(collection, siteName, azureMigrateDiscoveryObject(payload["properties"]), createdAt, now)
	record.CreatedAt = createdAt
	record.UpdatedAt = now
	record.ETag = azureMigrateDiscoveryETag(id, body, now)
	s.azureMigrateDiscoverySites[key] = record

	s.azureMigrateDiscoverySeedSiteChildrenLocked(record)

	respondJSON(w, status, azureMigrateDiscoverySiteFixture(record))
}

func (s *Server) handleAzureMigrateDiscoveryPatchSite(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, siteName string) {
	body, payload, ok := azureMigrateDiscoveryReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureMigrateDiscoveryStorageKey(azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName))

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	record := s.azureMigrateDiscoverySites[key]
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "site", siteName))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	updated := false
	if location, ok := payload["location"]; ok {
		loc := azureMigrateDiscoveryString(location)
		if loc == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		record.Location = loc
		updated = true
	}
	if tags, ok := payload["tags"]; ok {
		record.Tags = azureMigrateDiscoveryStringMap(tags)
		updated = true
	}
	if props, ok := payload["properties"]; ok {
		record.Properties = azureMigrateDiscoveryMergeObject(record.Properties, azureMigrateDiscoveryObject(props))
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	record.UpdatedAt = time.Now().UTC()
	record.ETag = azureMigrateDiscoveryETag(key, body, record.UpdatedAt)
	respondJSON(w, http.StatusOK, azureMigrateDiscoverySiteFixture(record))
}

func (s *Server) handleAzureMigrateDiscoveryGetSite(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, siteName string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoverySites[azureMigrateDiscoveryStorageKey(azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName))]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "site", siteName))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateDiscoverySiteFixture(record))
}

func (s *Server) handleAzureMigrateDiscoveryDeleteSite(w http.ResponseWriter, subscriptionID, resourceGroup, collection, siteName string) {
	id := azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName)
	key := azureMigrateDiscoveryStorageKey(id)

	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoverySites[key]
	delete(s.azureMigrateDiscoverySites, key)
	for childKey, child := range s.azureMigrateDiscoveryMachines {
		if child.SiteID == id {
			delete(s.azureMigrateDiscoveryMachines, childKey)
		}
	}
	for childKey, child := range s.azureMigrateDiscoveryRunAsAccounts {
		if child.SiteID == id {
			delete(s.azureMigrateDiscoveryRunAsAccounts, childKey)
		}
	}
	for childKey, child := range s.azureMigrateDiscoveryInventoryItems {
		if child.SiteID == id {
			delete(s.azureMigrateDiscoveryInventoryItems, childKey)
		}
	}
	s.providerStorageMu.Unlock()

	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateDiscoverySiteFixture(record))
}

func (s *Server) handleAzureMigrateDiscoveryListSitesBySubscription(w http.ResponseWriter, subscriptionID, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureMigrateDiscoverySites {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.Collection, collection) {
			items = append(items, azureMigrateDiscoverySiteFixture(record))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateDiscoveryString(items[i]["id"]) < azureMigrateDiscoveryString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureMigrateDiscoveryListSitesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureMigrateDiscoverySites {
		if record.SubscriptionID == subscriptionID && record.ResourceGroup == resourceGroup && strings.EqualFold(record.Collection, collection) {
			items = append(items, azureMigrateDiscoverySiteFixture(record))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateDiscoveryString(items[i]["id"]) < azureMigrateDiscoveryString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureMigrateDiscoverySiteSummary(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, siteName, summaryKind string) {
	siteID := azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName)
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoverySites[azureMigrateDiscoveryStorageKey(siteID)]
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "site", siteName))
		return
	}

	machineCount := s.azureMigrateDiscoveryChildCount(siteID, "machines")
	runAsCount := s.azureMigrateDiscoveryChildCount(siteID, "runAsAccounts")
	hostCount := s.azureMigrateDiscoveryChildCount(siteID, "hosts")
	clusterCount := s.azureMigrateDiscoveryChildCount(siteID, "clusters")
	vCenterCount := s.azureMigrateDiscoveryChildCount(siteID, "vCenters")

	respondJSON(w, http.StatusOK, map[string]any{
		"id":                 siteID + "/" + summaryKind,
		"name":               summaryKind,
		"type":               "Microsoft.OffAzure/" + record.Collection + "/" + summaryKind,
		"machineCount":       machineCount,
		"runAsAccountCount":  runAsCount,
		"hostCount":          hostCount,
		"clusterCount":       clusterCount,
		"vCenterCount":       vCenterCount,
		"provisioningState":  "Succeeded",
		"lastRefreshedTime":  record.UpdatedAt.Format(time.RFC3339),
		"applianceName":      azureMigrateDiscoveryString(record.Properties["applianceName"]),
		"serviceEndpoint":    azureMigrateDiscoveryString(record.Properties["serviceEndpoint"]),
		"healthSummary":      map[string]any{"status": "Healthy", "warningCount": 0, "errorCount": 0},
		"discoveredMachines": machineCount,
	})
}

func (s *Server) handleAzureMigrateDiscoverySiteRefresh(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, siteName string) {
	siteID := azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName)
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoverySites[azureMigrateDiscoveryStorageKey(siteID)]
	if record != nil {
		record.UpdatedAt = time.Now().UTC()
	}
	s.providerStorageMu.Unlock()
	if record == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "site", siteName))
		return
	}

	operationPath := fmt.Sprintf("/azure/providers/Microsoft.OffAzure/operations/%s-refresh?api-version=%s", siteName, azureMigrateDiscoveryAPIVersion)
	w.Header().Set("Azure-AsyncOperation", operationPath)
	w.Header().Set("Location", operationPath)
	respondJSON(w, http.StatusAccepted, map[string]any{
		"status":    "Accepted",
		"operation": "refresh",
		"target":    siteID,
	})
}

func (s *Server) handleAzureMigrateDiscoveryMachinesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, siteName string, tail []string) {
	siteID, ok := s.azureMigrateDiscoveryRequireSite(w, path, subscriptionID, resourceGroup, collection, siteName)
	if !ok {
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureMigrateDiscoveryListMachines(w, siteID, collection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "machine name is required")
		return
	}
	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			s.handleAzureMigrateDiscoveryGetMachine(w, path, siteID, collection, name)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "start", "stop":
		if r.Method == http.MethodPost {
			s.handleAzureMigrateDiscoveryMachineAction(w, path, siteID, collection, name, strings.ToLower(strings.TrimSpace(tail[1])))
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateDiscoveryRunAsAccountsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, siteName string, tail []string) {
	siteID, ok := s.azureMigrateDiscoveryRequireSite(w, path, subscriptionID, resourceGroup, collection, siteName)
	if !ok {
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureMigrateDiscoveryListRunAsAccounts(w, siteID, collection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "run as account name is required")
		return
	}
	if r.Method == http.MethodGet {
		s.handleAzureMigrateDiscoveryGetRunAsAccount(w, path, siteID, collection, name)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateDiscoveryInventoryRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, siteName, inventoryCollection string, tail []string) {
	siteID, ok := s.azureMigrateDiscoveryRequireSite(w, path, subscriptionID, resourceGroup, collection, siteName)
	if !ok {
		return
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureMigrateDiscoveryListInventoryItems(w, siteID, collection, inventoryCollection)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, strings.TrimSuffix(strings.ToLower(inventoryCollection), "s")+" name is required")
		return
	}
	if r.Method == http.MethodGet {
		s.handleAzureMigrateDiscoveryGetInventoryItem(w, path, siteID, collection, inventoryCollection, name)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateDiscoveryListMachines(w http.ResponseWriter, siteID, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureMigrateDiscoveryMachines {
		if record.SiteID == siteID && strings.EqualFold(record.Collection, collection) {
			items = append(items, azureMigrateDiscoveryMachineFixture(record))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateDiscoveryString(items[i]["id"]) < azureMigrateDiscoveryString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureMigrateDiscoveryGetMachine(w http.ResponseWriter, path, siteID, collection, name string) {
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoveryMachines[azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryMachineID(siteID, name))]
	s.providerStorageMu.Unlock()
	if record == nil || !strings.EqualFold(record.Collection, collection) {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "machine", name))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateDiscoveryMachineFixture(record))
}

func (s *Server) handleAzureMigrateDiscoveryMachineAction(w http.ResponseWriter, path, siteID, collection, name, action string) {
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoveryMachines[azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryMachineID(siteID, name))]
	if record == nil || !strings.EqualFold(record.Collection, collection) {
		s.providerStorageMu.Unlock()
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "machine", name))
		return
	}
	if action == "start" {
		record.PowerState = "Running"
	} else {
		record.PowerState = "Stopped"
	}
	record.Properties["powerStatus"] = record.PowerState
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusAccepted, map[string]any{
		"status":     "Accepted",
		"action":     action,
		"powerState": record.PowerState,
		"target":     azureMigrateDiscoveryMachineID(siteID, name),
	})
}

func (s *Server) handleAzureMigrateDiscoveryListRunAsAccounts(w http.ResponseWriter, siteID, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureMigrateDiscoveryRunAsAccounts {
		if record.SiteID == siteID && strings.EqualFold(record.Collection, collection) {
			items = append(items, azureMigrateDiscoveryRunAsAccountFixture(record))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateDiscoveryString(items[i]["id"]) < azureMigrateDiscoveryString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureMigrateDiscoveryGetRunAsAccount(w http.ResponseWriter, path, siteID, collection, name string) {
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoveryRunAsAccounts[azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryRunAsAccountID(siteID, name))]
	s.providerStorageMu.Unlock()
	if record == nil || !strings.EqualFold(record.Collection, collection) {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "run as account", name))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateDiscoveryRunAsAccountFixture(record))
}

func (s *Server) handleAzureMigrateDiscoveryListInventoryItems(w http.ResponseWriter, siteID, collection, inventoryCollection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureMigrateDiscoveryInventoryItems {
		if record.SiteID == siteID && strings.EqualFold(record.Collection, inventoryCollection) && azureMigrateDiscoveryInventoryAllowedForSiteType(collection, inventoryCollection) {
			items = append(items, azureMigrateDiscoveryInventoryFixture(siteID, record))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateDiscoveryString(items[i]["id"]) < azureMigrateDiscoveryString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureMigrateDiscoveryGetInventoryItem(w http.ResponseWriter, path, siteID, collection, inventoryCollection, name string) {
	s.providerStorageMu.Lock()
	record := s.azureMigrateDiscoveryInventoryItems[azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryInventoryID(siteID, inventoryCollection, name))]
	s.providerStorageMu.Unlock()
	if record == nil || !strings.EqualFold(record.Collection, inventoryCollection) || !azureMigrateDiscoveryInventoryAllowedForSiteType(collection, inventoryCollection) {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, strings.TrimSuffix(strings.ToLower(inventoryCollection), "s"), name))
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateDiscoveryInventoryFixture(siteID, record))
}

func (s *Server) azureMigrateDiscoveryRequireSite(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, siteName string) (string, bool) {
	siteID := azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName)
	s.providerStorageMu.Lock()
	_, ok := s.azureMigrateDiscoverySites[azureMigrateDiscoveryStorageKey(siteID)]
	s.providerStorageMu.Unlock()
	if !ok {
		respondJSON(w, http.StatusNotFound, azureMigrateDiscoveryNotFound(path, "site", siteName))
		return "", false
	}
	return siteID, true
}

func (s *Server) azureMigrateDiscoveryChildCount(siteID, collection string) int {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	count := 0
	switch collection {
	case "machines":
		for _, record := range s.azureMigrateDiscoveryMachines {
			if record.SiteID == siteID {
				count++
			}
		}
	case "runAsAccounts":
		for _, record := range s.azureMigrateDiscoveryRunAsAccounts {
			if record.SiteID == siteID {
				count++
			}
		}
	default:
		for _, record := range s.azureMigrateDiscoveryInventoryItems {
			if record.SiteID == siteID && strings.EqualFold(record.Collection, collection) {
				count++
			}
		}
	}
	return count
}

func (s *Server) azureMigrateDiscoverySeedSiteChildrenLocked(site *azureMigrateDiscoverySite) {
	machineNames := []string{"machine-1", "machine-2"}
	if strings.EqualFold(site.Collection, "HyperVSites") {
		machineNames = []string{"hyperv-machine-1", "hyperv-machine-2"}
	}
	for idx, name := range machineNames {
		key := azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryMachineID(azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name), name))
		if _, exists := s.azureMigrateDiscoveryMachines[key]; !exists {
			powerState := "Stopped"
			if idx == 0 {
				powerState = "Running"
			}
			s.azureMigrateDiscoveryMachines[key] = &azureMigrateDiscoveryMachine{
				SiteID:     azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name),
				Collection: site.Collection,
				Name:       name,
				PowerState: powerState,
				Properties: map[string]any{
					"displayName": name,
					"powerStatus": powerState,
					"operatingSystemDetails": map[string]any{
						"osName":         "Windows Server 2022",
						"osVersion":      "10.0",
						"osArchitecture": "x64",
						"osType":         "Windows",
					},
					"disks": []any{
						map[string]any{"name": "osdisk", "maxSizeInBytes": int64(128 * 1024 * 1024 * 1024)},
					},
					"networkAdapters": []any{
						map[string]any{"displayName": "nic0", "ipAddresses": []any{"10.0.0.4"}},
					},
					"errors": []any{},
				},
				CreatedAt: site.CreatedAt,
				UpdatedAt: site.UpdatedAt,
			}
		}
	}

	accountName := "runas-default"
	accountKey := azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryRunAsAccountID(azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name), accountName))
	if _, exists := s.azureMigrateDiscoveryRunAsAccounts[accountKey]; !exists {
		s.azureMigrateDiscoveryRunAsAccounts[accountKey] = &azureMigrateDiscoveryRunAsAccount{
			SiteID:     azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name),
			Collection: site.Collection,
			Name:       accountName,
			Properties: map[string]any{
				"displayName":    "Default Run As Account",
				"credentialType": "Hypervisor",
				"username":       "stackyard\\svc-migrate",
			},
			CreatedAt: site.CreatedAt,
			UpdatedAt: site.UpdatedAt,
		}
	}

	if strings.EqualFold(site.Collection, "VMwareSites") {
		name := "vcenter-default"
		key := azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryInventoryID(azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name), "vCenters", name))
		if _, exists := s.azureMigrateDiscoveryInventoryItems[key]; !exists {
			s.azureMigrateDiscoveryInventoryItems[key] = &azureMigrateDiscoveryInventoryItem{
				SiteID:     azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name),
				Collection: "vCenters",
				Name:       name,
				Properties: map[string]any{
					"fqdn":         "vcenter.local",
					"instanceUuid": azureMigrateDiscoveryStableToken(site.Name, name, "vcenter"),
				},
				CreatedAt: site.CreatedAt,
				UpdatedAt: site.UpdatedAt,
			}
		}
		return
	}

	for _, item := range []struct {
		collection string
		name       string
		properties map[string]any
	}{
		{
			collection: "hosts",
			name:       "host-1",
			properties: map[string]any{"fqdn": "hyperv-host.local", "overallState": "Ok"},
		},
		{
			collection: "clusters",
			name:       "cluster-1",
			properties: map[string]any{"fqdn": "hyperv-cluster.local", "nodeCount": 2},
		},
	} {
		key := azureMigrateDiscoveryStorageKey(azureMigrateDiscoveryInventoryID(azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name), item.collection, item.name))
		if _, exists := s.azureMigrateDiscoveryInventoryItems[key]; !exists {
			s.azureMigrateDiscoveryInventoryItems[key] = &azureMigrateDiscoveryInventoryItem{
				SiteID:     azureMigrateDiscoverySiteID(site.SubscriptionID, site.ResourceGroup, site.Collection, site.Name),
				Collection: item.collection,
				Name:       item.name,
				Properties: item.properties,
				CreatedAt:  site.CreatedAt,
				UpdatedAt:  site.UpdatedAt,
			}
		}
	}
}

func azureMigrateDiscoverySiteProperties(collection, siteName string, incoming map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureMigrateDiscoveryMergeObject(map[string]any{
		"applianceName":       siteName + "-appliance",
		"serviceEndpoint":     "https://" + strings.ToLower(siteName) + ".stackyard.local",
		"discoverySolutionId": "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.OffAzure/discoverySolutions/default",
		"provisioningState":   "Succeeded",
		"servicePrincipalIdentityDetails": map[string]any{
			"tenantId":      "11111111-1111-1111-1111-111111111111",
			"applicationId": azureMigrateDiscoveryStableToken(siteName, "application"),
			"objectId":      azureMigrateDiscoveryStableToken(siteName, "object"),
		},
		"agentDetails": map[string]any{
			"id":               azureMigrateDiscoveryStableToken(siteName, "agent"),
			"version":          "1.0.0",
			"lastHeartbeatUtc": updatedAt.Format(time.RFC3339),
			"agentStatus":      "Healthy",
		},
		"createdTimestamp": createdAt.Format(time.RFC3339),
		"updatedTimestamp": updatedAt.Format(time.RFC3339),
	}, incoming)
	if strings.EqualFold(collection, "HyperVSites") {
		props["siteType"] = "HyperV"
	} else {
		props["siteType"] = "VMware"
	}
	return props
}

func azureMigrateDiscoveryInventoryAllowedForSiteType(siteCollection, inventoryCollection string) bool {
	switch strings.ToLower(strings.TrimSpace(siteCollection)) {
	case "vmwaresites":
		return strings.EqualFold(inventoryCollection, "vCenters")
	case "hypervsites":
		return strings.EqualFold(inventoryCollection, "hosts") || strings.EqualFold(inventoryCollection, "clusters")
	default:
		return false
	}
}

func azureMigrateDiscoverySiteFixture(record *azureMigrateDiscoverySite) map[string]any {
	return map[string]any{
		"id":         azureMigrateDiscoverySiteID(record.SubscriptionID, record.ResourceGroup, record.Collection, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.OffAzure/" + record.Collection,
		"location":   record.Location,
		"tags":       azureMigrateDiscoveryStringMapAny(record.Tags),
		"etag":       record.ETag,
		"properties": azureMigrateDiscoveryMergeObject(map[string]any{}, record.Properties),
		"systemData": map[string]any{
			"createdAt":          record.CreatedAt.Format(time.RFC3339),
			"lastModifiedAt":     record.UpdatedAt.Format(time.RFC3339),
			"createdBy":          "stackyard",
			"lastModifiedBy":     "stackyard",
			"createdByType":      "Application",
			"lastModifiedByType": "Application",
		},
	}
}

func azureMigrateDiscoveryMachineFixture(record *azureMigrateDiscoveryMachine) map[string]any {
	return map[string]any{
		"id":         azureMigrateDiscoveryMachineID(record.SiteID, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.OffAzure/" + record.Collection + "/machines",
		"properties": azureMigrateDiscoveryMergeObject(map[string]any{}, record.Properties),
		"systemData": map[string]any{
			"createdAt":      record.CreatedAt.Format(time.RFC3339),
			"lastModifiedAt": record.UpdatedAt.Format(time.RFC3339),
		},
	}
}

func azureMigrateDiscoveryRunAsAccountFixture(record *azureMigrateDiscoveryRunAsAccount) map[string]any {
	return map[string]any{
		"id":         azureMigrateDiscoveryRunAsAccountID(record.SiteID, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.OffAzure/" + record.Collection + "/runAsAccounts",
		"properties": azureMigrateDiscoveryMergeObject(map[string]any{}, record.Properties),
	}
}

func azureMigrateDiscoveryInventoryFixture(siteID string, record *azureMigrateDiscoveryInventoryItem) map[string]any {
	return map[string]any{
		"id":         azureMigrateDiscoveryInventoryID(siteID, record.Collection, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.OffAzure/" + strings.Trim(strings.Join([]string{strings.TrimPrefix(strings.Split(siteID, "/providers/Microsoft.OffAzure/")[1], "/"), record.Collection}, "/"), "/"),
		"properties": azureMigrateDiscoveryMergeObject(map[string]any{}, record.Properties),
	}
}

func azureMigrateDiscoveryOperationsFixture() []map[string]any {
	return []map[string]any{
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/read", "VMware Sites", "Read VMware discovery site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/write", "VMware Sites", "Create or update VMware discovery site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/delete", "VMware Sites", "Delete VMware discovery site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/refresh/action", "VMware Sites", "Refresh VMware site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/machines/read", "VMware Machines", "Read VMware machines"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/runAsAccounts/read", "VMware Run As Accounts", "Read VMware run as accounts"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/VMwareSites/vCenters/read", "VMware vCenters", "Read VMware vCenters"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/read", "Hyper-V Sites", "Read Hyper-V discovery site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/write", "Hyper-V Sites", "Create or update Hyper-V discovery site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/delete", "Hyper-V Sites", "Delete Hyper-V discovery site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/refresh/action", "Hyper-V Sites", "Refresh Hyper-V site"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/machines/read", "Hyper-V Machines", "Read Hyper-V machines"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/runAsAccounts/read", "Hyper-V Run As Accounts", "Read Hyper-V run as accounts"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/hosts/read", "Hyper-V Hosts", "Read Hyper-V hosts"),
		azureMigrateDiscoveryOperation("Microsoft.OffAzure/HyperVSites/clusters/read", "Hyper-V Clusters", "Read Hyper-V clusters"),
	}
}

func azureMigrateDiscoveryOperation(name, resource, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":    "Microsoft.OffAzure",
			"resource":    resource,
			"operation":   name,
			"description": description,
		},
		"origin": "user,system",
	}
}

func azureMigrateDiscoveryNotFound(path, resourceType, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("%s %q was not found for %s", resourceType, name, path),
		},
	}
}

func azureMigrateDiscoverySiteID(subscriptionID, resourceGroup, collection, siteName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.OffAzure/%s/%s", subscriptionID, resourceGroup, collection, siteName)
}

func azureMigrateDiscoveryMachineID(siteID, name string) string {
	return siteID + "/machines/" + name
}

func azureMigrateDiscoveryRunAsAccountID(siteID, name string) string {
	return siteID + "/runAsAccounts/" + name
}

func azureMigrateDiscoveryInventoryID(siteID, collection, name string) string {
	return siteID + "/" + collection + "/" + name
}

func azureMigrateDiscoveryETag(id string, body []byte, updatedAt time.Time) string {
	sum := sha1.Sum(append([]byte(id+"|"+updatedAt.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureMigrateDiscoveryStorageKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned = append(cleaned, strings.ToLower(strings.TrimSpace(part)))
	}
	return strings.Join(cleaned, "::")
}

func azureMigrateDiscoveryStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func azureMigrateDiscoveryReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
	return body, payload, true
}

func azureMigrateDiscoveryString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func azureMigrateDiscoveryObject(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureMigrateDiscoveryStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	switch typed := value.(type) {
	case map[string]string:
		out := make(map[string]string, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	case map[string]any:
		out := make(map[string]string, len(typed))
		for k, v := range typed {
			out[k] = azureMigrateDiscoveryString(v)
		}
		return out
	default:
		return map[string]string{}
	}
}

func azureMigrateDiscoveryStringMapAny(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for k, v := range value {
		out[k] = v
	}
	return out
}

func azureMigrateDiscoveryMergeObject(base map[string]any, overlays ...map[string]any) map[string]any {
	out := make(map[string]any, len(base))
	for k, v := range base {
		out[k] = v
	}
	for _, overlay := range overlays {
		for k, v := range overlay {
			out[k] = v
		}
	}
	return out
}
