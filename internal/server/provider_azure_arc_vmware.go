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

type azureArcVMwareResource struct {
	SubscriptionID   string
	ResourceGroup    string
	Collection       string
	Name             string
	Location         string
	Kind             string
	Tags             map[string]string
	ExtendedLocation map[string]any
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureArcVMwareInventoryItem struct {
	SubscriptionID   string
	ResourceGroup    string
	VCenterName      string
	Name             string
	Location         string
	Kind             string
	Tags             map[string]string
	ExtendedLocation map[string]any
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureArcVMwareVirtualMachineInstance struct {
	ResourceURI      string
	Name             string
	Location         string
	Tags             map[string]string
	ExtendedLocation map[string]any
	Properties       map[string]any
	PowerState       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureArcVMwareGuestAgent struct {
	ResourceURI string
	Name        string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type azureArcVMwareHybridIdentityMetadata struct {
	ResourceURI string
	Name        string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var azureArcVMwareTopLevelCollections = map[string]string{
	"vcenters":                "vCenters",
	"clusters":                "clusters",
	"hosts":                   "hosts",
	"resourcepools":           "resourcePools",
	"datastores":              "datastores",
	"virtualnetworks":         "virtualNetworks",
	"virtualmachinetemplates": "virtualMachineTemplates",
}

func (s *Server) handleAzureArcVMwareRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsArcVMwarePath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureArcVMwareAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ConnectedVMwarevSphere") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureArcVMwareOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	if subscriptionID, resourceGroup, tail, ok := azureArcVMwareRouteContext(segments); ok {
		if resourceGroup == "" {
			if len(tail) == 1 {
				if collection, found := azureArcVMwareTopLevelCollections[strings.ToLower(strings.TrimSpace(tail[0]))]; found && r.Method == http.MethodGet {
					s.handleAzureArcVMwareTopLevelListBySubscription(w, subscriptionID, collection)
					return true
				}
			}
			respondAzureImplemented(w, path)
			return true
		}
		if handled := s.handleAzureArcVMwareTopLevelResourceRoute(w, r, path, subscriptionID, resourceGroup, tail); handled {
			return true
		}
	}

	if resourceURI, tail, ok := azureArcVMwareResourceURIRouteContext(segments); ok {
		s.handleAzureArcVMwareVirtualMachineInstanceRoute(w, r, path, resourceURI, tail)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureArcVMwareTopLevelResourceRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) bool {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	collection, found := azureArcVMwareTopLevelCollections[strings.ToLower(strings.TrimSpace(tail[0]))]
	if !found {
		return false
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureArcVMwareTopLevelListByResourceGroup(w, subscriptionID, resourceGroup, collection)
		return true
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return true
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, strings.TrimSuffix(collection, "s")+" name is required")
		return true
	}

	if strings.EqualFold(collection, "vCenters") && len(tail) >= 3 && strings.EqualFold(tail[2], "inventoryItems") {
		s.handleAzureArcVMwareInventoryItemsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[3:])
		return true
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureArcVMwareTopLevelGet(w, path, subscriptionID, resourceGroup, collection, name, false)
	case http.MethodHead:
		s.handleAzureArcVMwareTopLevelGet(w, path, subscriptionID, resourceGroup, collection, name, true)
	case http.MethodPut:
		s.handleAzureArcVMwareTopLevelCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, collection, name)
	case http.MethodPatch:
		s.handleAzureArcVMwareTopLevelPatch(w, r, path, subscriptionID, resourceGroup, collection, name)
	case http.MethodDelete:
		s.handleAzureArcVMwareTopLevelDelete(w, path, subscriptionID, resourceGroup, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	return true
}

func (s *Server) handleAzureArcVMwareInventoryItemsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, vCenterName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureArcVMwareInventoryItemList(w, path, subscriptionID, resourceGroup, vCenterName)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "inventory item name is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureArcVMwareInventoryItemGet(w, path, subscriptionID, resourceGroup, vCenterName, name)
	case http.MethodPut, http.MethodPost:
		s.handleAzureArcVMwareInventoryItemCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, vCenterName, name)
	case http.MethodDelete:
		s.handleAzureArcVMwareInventoryItemDelete(w, path, subscriptionID, resourceGroup, vCenterName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstanceRoute(w http.ResponseWriter, r *http.Request, path, resourceURI string, tail []string) {
	if len(tail) == 0 || !strings.EqualFold(tail[0], "virtualMachineInstances") {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			s.handleAzureArcVMwareVirtualMachineInstanceList(w, resourceURI)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "virtual machine instance name is required")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureArcVMwareVirtualMachineInstanceGet(w, path, resourceURI, name)
		case http.MethodPut:
			s.handleAzureArcVMwareVirtualMachineInstanceCreateOrUpdate(w, r, path, resourceURI, name)
		case http.MethodPatch:
			s.handleAzureArcVMwareVirtualMachineInstancePatch(w, r, path, resourceURI, name)
		case http.MethodDelete:
			s.handleAzureArcVMwareVirtualMachineInstanceDelete(w, path, resourceURI, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "start":
		if r.Method == http.MethodPost {
			s.handleAzureArcVMwareVirtualMachineInstanceAction(w, path, resourceURI, "start")
			return
		}
	case "stop":
		if r.Method == http.MethodPost {
			s.handleAzureArcVMwareVirtualMachineInstanceAction(w, path, resourceURI, "stop")
			return
		}
	case "restart":
		if r.Method == http.MethodPost {
			s.handleAzureArcVMwareVirtualMachineInstanceAction(w, path, resourceURI, "restart")
			return
		}
	case "guestagents":
		s.handleAzureArcVMwareGuestAgentRoute(w, r, path, resourceURI, tail[3:])
		return
	case "hybrididentitymetadata":
		s.handleAzureArcVMwareHybridIdentityMetadataRoute(w, r, path, resourceURI, tail[3:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureArcVMwareGuestAgentRoute(w http.ResponseWriter, r *http.Request, path, resourceURI string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "guest agent name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureArcVMwareGuestAgentGet(w, path, resourceURI, name)
	case http.MethodPut:
		s.handleAzureArcVMwareGuestAgentCreateOrUpdate(w, r, path, resourceURI, name)
	case http.MethodDelete:
		s.handleAzureArcVMwareGuestAgentDelete(w, path, resourceURI, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureArcVMwareHybridIdentityMetadataRoute(w http.ResponseWriter, r *http.Request, path, resourceURI string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureArcVMwareHybridIdentityMetadataList(w, path, resourceURI)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, "hybrid identity metadata name is required")
		return
	}
	if r.Method == http.MethodGet {
		s.handleAzureArcVMwareHybridIdentityMetadataGet(w, path, resourceURI, name)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleAzureArcVMwareTopLevelCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	_, payload, ok := azureArcVMwareReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureArcVMwareString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureArcVMwareObject(payload["properties"])
	if !azureArcVMwareValidateTopLevelProperties(w, path, collection, properties) {
		return
	}

	now := time.Now().UTC()
	record := &azureArcVMwareResource{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		Collection:       collection,
		Name:             name,
		Location:         location,
		Kind:             strings.TrimSpace(azureArcVMwareString(payload["kind"])),
		Tags:             azureArcVMwareStringMap(payload["tags"]),
		ExtendedLocation: azureArcVMwareObject(payload["extendedLocation"]),
		Properties: azureArcVMwareMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"uuid":              azureArcVMwareStableToken(collection, subscriptionID, resourceGroup, name),
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if _, ok := record.Properties["inventoryItemId"]; !ok && collection != "vCenters" {
		record.Properties["inventoryItemId"] = fmt.Sprintf("%s/inventoryItems/%s", azureArcVMwareResourceID("vCenters", subscriptionID, resourceGroup, firstNonEmptyString(azureArcVMwareString(record.Properties["vCenterName"]), "vcenter-default")), name)
	}
	if _, ok := record.Properties["vCenterId"]; !ok && collection != "vCenters" {
		record.Properties["vCenterId"] = azureArcVMwareResourceID("vCenters", subscriptionID, resourceGroup, firstNonEmptyString(azureArcVMwareString(record.Properties["vCenterName"]), "vcenter-default"))
	}

	s.providerStorageMu.Lock()
	s.azureArcVMwareResources[azureArcVMwareStorageKey(azureArcVMwareResourceID(collection, subscriptionID, resourceGroup, name))] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureArcVMwareResourceFixture(record))
}

func (s *Server) handleAzureArcVMwareTopLevelPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	_, payload, ok := azureArcVMwareReadRequestBody(w, r, path)
	if !ok {
		return
	}
	id := azureArcVMwareResourceID(collection, subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareResources[azureArcVMwareStorageKey(id)]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcVMwareNotFound(w, path, collection+" not found")
		return
	}
	if location := strings.TrimSpace(azureArcVMwareString(payload["location"])); location != "" {
		record.Location = location
	}
	if kind := strings.TrimSpace(azureArcVMwareString(payload["kind"])); kind != "" {
		record.Kind = kind
	}
	if tags := azureArcVMwareStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if extendedLocation := azureArcVMwareObject(payload["extendedLocation"]); len(extendedLocation) != 0 {
		record.ExtendedLocation = extendedLocation
	}
	if properties := azureArcVMwareObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureArcVMwareMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcVMwareResourceFixture(record))
}

func (s *Server) handleAzureArcVMwareTopLevelGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareResources[azureArcVMwareStorageKey(azureArcVMwareResourceID(collection, subscriptionID, resourceGroup, name))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcVMwareNotFound(w, path, collection+" not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareResourceFixture(record))
}

func (s *Server) handleAzureArcVMwareTopLevelDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string) {
	id := azureArcVMwareResourceID(collection, subscriptionID, resourceGroup, name)
	key := azureArcVMwareStorageKey(id)
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareResources[key]
	delete(s.azureArcVMwareResources, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareResourceFixture(record))
}

func (s *Server) handleAzureArcVMwareTopLevelListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup, collection string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureArcVMwareResources {
		if record.Collection == collection && record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) {
			items = append(items, azureArcVMwareResourceFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureArcVMwareString(items[i]["id"])) < strings.ToLower(azureArcVMwareString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcVMwareTopLevelListBySubscription(w http.ResponseWriter, subscriptionID, collection string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureArcVMwareResources {
		if record.Collection == collection && record.SubscriptionID == subscriptionID {
			items = append(items, azureArcVMwareResourceFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureArcVMwareString(items[i]["id"])) < strings.ToLower(azureArcVMwareString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcVMwareInventoryItemCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, vCenterName, name string) {
	_, payload, ok := azureArcVMwareReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureArcVMwareVCenterExists(subscriptionID, resourceGroup, vCenterName) {
		azureArcVMwareNotFound(w, path, "vcenter not found")
		return
	}
	properties := azureArcVMwareObject(payload["properties"])
	inventoryType := strings.TrimSpace(azureArcVMwareString(properties["inventoryType"]))
	if inventoryType == "" {
		respondAzureInvalidRequest(w, path, "properties.inventoryType is required")
		return
	}
	location := firstNonEmptyString(strings.TrimSpace(azureArcVMwareString(payload["location"])), "eastus")
	now := time.Now().UTC()
	record := &azureArcVMwareInventoryItem{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		VCenterName:      vCenterName,
		Name:             name,
		Location:         location,
		Kind:             strings.TrimSpace(azureArcVMwareString(payload["kind"])),
		Tags:             azureArcVMwareStringMap(payload["tags"]),
		ExtendedLocation: azureArcVMwareObject(payload["extendedLocation"]),
		Properties: azureArcVMwareMergeObject(map[string]any{
			"inventoryType":     inventoryType,
			"inventoryItemName": name,
			"managedResourceId": azureArcVMwareInventoryItemID(subscriptionID, resourceGroup, vCenterName, name),
			"uuid":              azureArcVMwareStableToken(name, inventoryType),
			"provisioningState": "Succeeded",
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.providerStorageMu.Lock()
	s.azureArcVMwareInventoryItems[azureArcVMwareStorageKey(azureArcVMwareInventoryItemID(subscriptionID, resourceGroup, vCenterName, name))] = record
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcVMwareInventoryItemFixture(record))
}

func (s *Server) handleAzureArcVMwareInventoryItemGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, vCenterName, name string) {
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareInventoryItems[azureArcVMwareStorageKey(azureArcVMwareInventoryItemID(subscriptionID, resourceGroup, vCenterName, name))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcVMwareNotFound(w, path, "inventory item not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareInventoryItemFixture(record))
}

func (s *Server) handleAzureArcVMwareInventoryItemDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, vCenterName, name string) {
	id := azureArcVMwareInventoryItemID(subscriptionID, resourceGroup, vCenterName, name)
	key := azureArcVMwareStorageKey(id)
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareInventoryItems[key]
	delete(s.azureArcVMwareInventoryItems, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareInventoryItemFixture(record))
}

func (s *Server) handleAzureArcVMwareInventoryItemList(w http.ResponseWriter, path, subscriptionID, resourceGroup, vCenterName string) {
	if !s.azureArcVMwareVCenterExists(subscriptionID, resourceGroup, vCenterName) {
		azureArcVMwareNotFound(w, path, "vcenter not found")
		return
	}
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureArcVMwareInventoryItems {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) && strings.EqualFold(record.VCenterName, vCenterName) {
			items = append(items, azureArcVMwareInventoryItemFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureArcVMwareString(items[i]["id"])) < strings.ToLower(azureArcVMwareString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstanceCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcVMwareReadRequestBody(w, r, path)
	if !ok {
		return
	}
	extendedLocation := azureArcVMwareObject(payload["extendedLocation"])
	if len(extendedLocation) == 0 || strings.TrimSpace(azureArcVMwareString(extendedLocation["type"])) == "" || strings.TrimSpace(azureArcVMwareString(extendedLocation["name"])) == "" {
		respondAzureInvalidRequest(w, path, "extendedLocation.type and extendedLocation.name are required")
		return
	}
	properties := azureArcVMwareObject(payload["properties"])
	if len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties are required")
		return
	}
	now := time.Now().UTC()
	record := &azureArcVMwareVirtualMachineInstance{
		ResourceURI:      resourceURI,
		Name:             name,
		Location:         strings.TrimSpace(azureArcVMwareString(payload["location"])),
		Tags:             azureArcVMwareStringMap(payload["tags"]),
		ExtendedLocation: extendedLocation,
		Properties: azureArcVMwareMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"powerState":        "Stopped",
			"resourceUid":       azureArcVMwareStableToken(resourceURI, name, "resourceUid"),
		}, properties),
		PowerState: "Stopped",
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	vmKey := azureArcVMwareStorageKey(azureArcVMwareVirtualMachineInstanceID(resourceURI))
	hybridID := azureArcVMwareHybridIdentityMetadataID(resourceURI)
	hybridKey := azureArcVMwareStorageKey(hybridID)

	s.providerStorageMu.Lock()
	s.azureArcVMwareVirtualMachineInstances[vmKey] = record
	if _, exists := s.azureArcVMwareHybridIdentityMetadata[hybridKey]; !exists {
		s.azureArcVMwareHybridIdentityMetadata[hybridKey] = &azureArcVMwareHybridIdentityMetadata{
			ResourceURI: resourceURI,
			Name:        "default",
			Properties: map[string]any{
				"resourceUid":       azureArcVMwareStableToken(resourceURI, "resourceUid"),
				"publicKey":         "stackyard-vmware-public-key",
				"provisioningState": "Succeeded",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureArcVMwareVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstancePatch(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcVMwareReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareVirtualMachineInstances[azureArcVMwareStorageKey(azureArcVMwareVirtualMachineInstanceID(resourceURI))]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcVMwareNotFound(w, path, "virtual machine instance not found")
		return
	}
	if tags := azureArcVMwareStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if properties := azureArcVMwareObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureArcVMwareMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcVMwareVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstanceGet(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareVirtualMachineInstances[azureArcVMwareStorageKey(azureArcVMwareVirtualMachineInstanceID(resourceURI))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcVMwareNotFound(w, path, "virtual machine instance not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstanceDelete(w http.ResponseWriter, path, resourceURI, name string) {
	id := azureArcVMwareVirtualMachineInstanceID(resourceURI)
	key := azureArcVMwareStorageKey(id)
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareVirtualMachineInstances[key]
	delete(s.azureArcVMwareVirtualMachineInstances, key)
	delete(s.azureArcVMwareGuestAgents, azureArcVMwareStorageKey(azureArcVMwareGuestAgentID(resourceURI)))
	delete(s.azureArcVMwareHybridIdentityMetadata, azureArcVMwareStorageKey(azureArcVMwareHybridIdentityMetadataID(resourceURI)))
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstanceList(w http.ResponseWriter, resourceURI string) {
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareVirtualMachineInstances[azureArcVMwareStorageKey(azureArcVMwareVirtualMachineInstanceID(resourceURI))]
	s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	if record != nil {
		items = append(items, azureArcVMwareVirtualMachineInstanceFixture(record))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcVMwareVirtualMachineInstanceAction(w http.ResponseWriter, path, resourceURI, action string) {
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareVirtualMachineInstances[azureArcVMwareStorageKey(azureArcVMwareVirtualMachineInstanceID(resourceURI))]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcVMwareNotFound(w, path, "virtual machine instance not found")
		return
	}
	switch action {
	case "start", "restart":
		record.PowerState = "Running"
	case "stop":
		record.PowerState = "Stopped"
	}
	record.Properties["powerState"] = record.PowerState
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"status":      "Succeeded",
		"action":      action,
		"resourceUri": resourceURI,
		"powerState":  record.PowerState,
		"target":      azureArcVMwareVirtualMachineInstanceID(resourceURI),
	})
}

func (s *Server) handleAzureArcVMwareGuestAgentCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcVMwareReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureArcVMwareVirtualMachineInstanceExists(resourceURI) {
		azureArcVMwareNotFound(w, path, "virtual machine instance not found")
		return
	}
	properties := azureArcVMwareObject(payload["properties"])
	credentials := azureArcVMwareObject(properties["credentials"])
	if strings.TrimSpace(azureArcVMwareString(credentials["username"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.credentials.username is required")
		return
	}

	now := time.Now().UTC()
	record := &azureArcVMwareGuestAgent{
		ResourceURI: resourceURI,
		Name:        name,
		Properties: azureArcVMwareMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"status":            "Installed",
			"uuid":              azureArcVMwareStableToken(resourceURI, name, "guest-agent"),
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureArcVMwareGuestAgents[azureArcVMwareStorageKey(azureArcVMwareGuestAgentID(resourceURI))] = record
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusCreated, azureArcVMwareGuestAgentFixture(record))
}

func (s *Server) handleAzureArcVMwareGuestAgentGet(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareGuestAgents[azureArcVMwareStorageKey(azureArcVMwareGuestAgentID(resourceURI))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcVMwareNotFound(w, path, "guest agent not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareGuestAgentFixture(record))
}

func (s *Server) handleAzureArcVMwareGuestAgentDelete(w http.ResponseWriter, path, resourceURI, name string) {
	id := azureArcVMwareGuestAgentID(resourceURI)
	key := azureArcVMwareStorageKey(id)
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareGuestAgents[key]
	delete(s.azureArcVMwareGuestAgents, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareGuestAgentFixture(record))
}

func (s *Server) handleAzureArcVMwareHybridIdentityMetadataGet(w http.ResponseWriter, path, resourceURI, name string) {
	if !s.azureArcVMwareVirtualMachineInstanceExists(resourceURI) {
		azureArcVMwareNotFound(w, path, "virtual machine instance not found")
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareHybridIdentityMetadata[azureArcVMwareStorageKey(azureArcVMwareHybridIdentityMetadataID(resourceURI))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcVMwareNotFound(w, path, "hybrid identity metadata not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcVMwareHybridIdentityMetadataFixture(record))
}

func (s *Server) handleAzureArcVMwareHybridIdentityMetadataList(w http.ResponseWriter, path, resourceURI string) {
	if !s.azureArcVMwareVirtualMachineInstanceExists(resourceURI) {
		azureArcVMwareNotFound(w, path, "virtual machine instance not found")
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcVMwareHybridIdentityMetadata[azureArcVMwareStorageKey(azureArcVMwareHybridIdentityMetadataID(resourceURI))]
	s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	if record != nil {
		items = append(items, azureArcVMwareHybridIdentityMetadataFixture(record))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) azureArcVMwareVCenterExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureArcVMwareResources[azureArcVMwareStorageKey(azureArcVMwareResourceID("vCenters", subscriptionID, resourceGroup, name))] != nil
}

func (s *Server) azureArcVMwareVirtualMachineInstanceExists(resourceURI string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureArcVMwareVirtualMachineInstances[azureArcVMwareStorageKey(azureArcVMwareVirtualMachineInstanceID(resourceURI))] != nil
}

func azureIsArcVMwarePath(segments []string) bool {
	if len(segments) >= 2 && strings.EqualFold(segments[0], "providers") && strings.EqualFold(segments[1], "Microsoft.ConnectedVMwarevSphere") {
		return true
	}
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ConnectedVMwarevSphere") {
			return true
		}
	}
	return false
}

func azureArcVMwareAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureArcVMwareRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if len(segments) >= 4 && strings.EqualFold(segments[2], "providers") && strings.EqualFold(segments[3], "Microsoft.ConnectedVMwarevSphere") {
		return subscriptionID, "", segments[4:], true
	}
	if len(segments) >= 6 &&
		strings.EqualFold(segments[2], "resourceGroups") &&
		strings.TrimSpace(segments[3]) != "" &&
		strings.EqualFold(segments[4], "providers") &&
		strings.EqualFold(segments[5], "Microsoft.ConnectedVMwarevSphere") {
		return subscriptionID, strings.TrimSpace(segments[3]), segments[6:], true
	}
	return "", "", nil, false
}

func azureArcVMwareResourceURIRouteContext(segments []string) (resourceURI string, tail []string, ok bool) {
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ConnectedVMwarevSphere") {
			if idx == 0 {
				return "", nil, false
			}
			return "/" + strings.Join(segments[:idx], "/"), segments[idx+2:], true
		}
	}
	return "", nil, false
}

func azureArcVMwareOperationsFixture() []map[string]any {
	return []map[string]any{
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/vCenters/read", "Read vCenter"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/vCenters/write", "Create or update vCenter"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/clusters/read", "Read cluster"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/hosts/read", "Read host"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/resourcePools/read", "Read resource pool"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/datastores/read", "Read datastore"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualNetworks/read", "Read virtual network"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualMachineTemplates/read", "Read virtual machine template"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/inventoryItems/read", "Read inventory item"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/read", "Read virtual machine instance"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/write", "Create or update virtual machine instance"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/start/action", "Start virtual machine instance"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/stop/action", "Stop virtual machine instance"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/restart/action", "Restart virtual machine instance"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/guestAgents/write", "Create or update guest agent"),
		azureArcVMwareOperation("Microsoft.ConnectedVMwarevSphere/hybridIdentityMetadata/read", "Read hybrid identity metadata"),
	}
}

func azureArcVMwareOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":  "Microsoft Connected VMware vSphere",
			"resource":  "arcVmware",
			"operation": description,
		},
		"isDataAction": false,
	}
}

func azureArcVMwareResourceFixture(record *azureArcVMwareResource) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcVMwareResourceID(record.Collection, record.SubscriptionID, record.ResourceGroup, record.Name)
	return map[string]any{
		"id":               id,
		"name":             record.Name,
		"type":             "Microsoft.ConnectedVMwarevSphere/" + record.Collection,
		"location":         record.Location,
		"kind":             record.Kind,
		"tags":             azureArcVMwareStringMapAny(record.Tags),
		"extendedLocation": azureArcVMwareCloneMap(record.ExtendedLocation),
		"properties":       azureArcVMwareCloneMap(record.Properties),
		"etag":             azureArcVMwareETag(id, record.UpdatedAt),
		"systemData":       azureArcVMwareSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcVMwareInventoryItemFixture(record *azureArcVMwareInventoryItem) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcVMwareInventoryItemID(record.SubscriptionID, record.ResourceGroup, record.VCenterName, record.Name)
	return map[string]any{
		"id":               id,
		"name":             record.Name,
		"type":             "Microsoft.ConnectedVMwarevSphere/vCenters/inventoryItems",
		"location":         record.Location,
		"kind":             record.Kind,
		"tags":             azureArcVMwareStringMapAny(record.Tags),
		"extendedLocation": azureArcVMwareCloneMap(record.ExtendedLocation),
		"properties":       azureArcVMwareCloneMap(record.Properties),
		"etag":             azureArcVMwareETag(id, record.UpdatedAt),
		"systemData":       azureArcVMwareSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcVMwareVirtualMachineInstanceFixture(record *azureArcVMwareVirtualMachineInstance) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcVMwareVirtualMachineInstanceID(record.ResourceURI)
	properties := azureArcVMwareMergeObject(record.Properties, map[string]any{
		"powerState":        record.PowerState,
		"provisioningState": "Succeeded",
	})
	return map[string]any{
		"id":               id,
		"name":             record.Name,
		"type":             "Microsoft.ConnectedVMwarevSphere/virtualMachineInstances",
		"location":         record.Location,
		"tags":             azureArcVMwareStringMapAny(record.Tags),
		"extendedLocation": azureArcVMwareCloneMap(record.ExtendedLocation),
		"properties":       properties,
		"etag":             azureArcVMwareETag(id, record.UpdatedAt),
		"systemData":       azureArcVMwareSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcVMwareGuestAgentFixture(record *azureArcVMwareGuestAgent) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcVMwareGuestAgentID(record.ResourceURI)
	return map[string]any{
		"id":         id,
		"name":       record.Name,
		"type":       "Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/guestAgents",
		"properties": azureArcVMwareCloneMap(record.Properties),
		"etag":       azureArcVMwareETag(id, record.UpdatedAt),
		"systemData": azureArcVMwareSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcVMwareHybridIdentityMetadataFixture(record *azureArcVMwareHybridIdentityMetadata) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcVMwareHybridIdentityMetadataID(record.ResourceURI)
	return map[string]any{
		"id":         id,
		"name":       record.Name,
		"type":       "Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/hybridIdentityMetadata",
		"properties": azureArcVMwareCloneMap(record.Properties),
		"etag":       azureArcVMwareETag(id, record.UpdatedAt),
		"systemData": azureArcVMwareSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcVMwareResourceID(collection, subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ConnectedVMwarevSphere/%s/%s", subscriptionID, resourceGroup, collection, name)
}

func azureArcVMwareInventoryItemID(subscriptionID, resourceGroup, vCenterName, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ConnectedVMwarevSphere/vCenters/%s/inventoryItems/%s", subscriptionID, resourceGroup, vCenterName, name)
}

func azureArcVMwareVirtualMachineInstanceID(resourceURI string) string {
	return strings.TrimRight(resourceURI, "/") + "/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default"
}

func azureArcVMwareGuestAgentID(resourceURI string) string {
	return azureArcVMwareVirtualMachineInstanceID(resourceURI) + "/guestAgents/default"
}

func azureArcVMwareHybridIdentityMetadataID(resourceURI string) string {
	return azureArcVMwareVirtualMachineInstanceID(resourceURI) + "/hybridIdentityMetadata/default"
}

func azureArcVMwareReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureArcVMwareValidateTopLevelProperties(w http.ResponseWriter, path, collection string, properties map[string]any) bool {
	switch collection {
	case "vCenters":
		if strings.TrimSpace(azureArcVMwareString(properties["fqdn"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.fqdn is required")
			return false
		}
	case "clusters", "hosts", "resourcePools", "datastores", "virtualNetworks", "virtualMachineTemplates":
		if strings.TrimSpace(azureArcVMwareString(properties["vCenterId"])) == "" && strings.TrimSpace(azureArcVMwareString(properties["inventoryItemId"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.vCenterId or properties.inventoryItemId is required")
			return false
		}
	}
	return true
}

func azureArcVMwareNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": message,
			"target":  path,
		},
	})
}

func azureArcVMwareString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func azureArcVMwareObject(value any) map[string]any {
	if value == nil {
		return nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return azureArcVMwareCloneMap(obj)
}

func azureArcVMwareStringMap(value any) map[string]string {
	if value == nil {
		return nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(obj))
	for key, raw := range obj {
		if str, ok := raw.(string); ok {
			out[key] = str
		}
	}
	return out
}

func azureArcVMwareStringMapAny(tags map[string]string) map[string]any {
	if len(tags) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func azureArcVMwareMergeObject(base, overlay map[string]any) map[string]any {
	out := azureArcVMwareCloneMap(base)
	for key, value := range overlay {
		if nested, ok := value.(map[string]any); ok {
			existing, _ := out[key].(map[string]any)
			out[key] = azureArcVMwareMergeObject(existing, nested)
			continue
		}
		out[key] = value
	}
	return out
}

func azureArcVMwareCloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureArcVMwareCloneMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func azureArcVMwareETag(id string, updatedAt time.Time) string {
	sum := sha1.Sum([]byte(id + "|" + updatedAt.Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureArcVMwareSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"createdAt":          createdAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
	}
}

func azureArcVMwareStorageKey(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func azureArcVMwareStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
