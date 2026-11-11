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

type azureArcSCVMMResource struct {
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

type azureArcSCVMMInventoryItem struct {
	SubscriptionID   string
	ResourceGroup    string
	VMMServerName    string
	Name             string
	Location         string
	Kind             string
	Tags             map[string]string
	ExtendedLocation map[string]any
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureArcSCVMMVirtualMachineInstance struct {
	ResourceURI      string
	Name             string
	Location         string
	Tags             map[string]string
	ExtendedLocation map[string]any
	Properties       map[string]any
	PowerState       string
	Checkpoints      []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type azureArcSCVMMGuestAgent struct {
	ResourceURI string
	Name        string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type azureArcSCVMMHybridIdentityMetadata struct {
	ResourceURI string
	Name        string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var azureArcSCVMMTopLevelCollections = map[string]string{
	"vmmservers":              "vmmServers",
	"clouds":                  "clouds",
	"availabilitysets":        "availabilitySets",
	"virtualnetworks":         "virtualNetworks",
	"virtualmachinetemplates": "virtualMachineTemplates",
}

func (s *Server) handleAzureArcSCVMMRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsArcSCVMMPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureArcSCVMMAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ScVmm") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureArcSCVMMOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	if subscriptionID, resourceGroup, tail, ok := azureArcSCVMMRouteContext(segments); ok {
		if resourceGroup == "" {
			if len(tail) == 1 {
				if collection, found := azureArcSCVMMTopLevelCollections[strings.ToLower(strings.TrimSpace(tail[0]))]; found && r.Method == http.MethodGet {
					s.handleAzureArcSCVMMTopLevelListBySubscription(w, subscriptionID, collection)
					return true
				}
			}
			respondAzureImplemented(w, path)
			return true
		}

		if handled := s.handleAzureArcSCVMMTopLevelResourceRoute(w, r, path, subscriptionID, resourceGroup, tail); handled {
			return true
		}
	}

	if resourceURI, tail, ok := azureArcSCVMMResourceURIRouteContext(segments); ok {
		s.handleAzureArcSCVMMVirtualMachineInstanceRoute(w, r, path, resourceURI, tail)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureArcSCVMMTopLevelResourceRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) bool {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	collection, found := azureArcSCVMMTopLevelCollections[strings.ToLower(strings.TrimSpace(tail[0]))]
	if !found {
		return false
	}
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureArcSCVMMTopLevelListByResourceGroup(w, subscriptionID, resourceGroup, collection)
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

	if strings.EqualFold(collection, "vmmServers") && len(tail) >= 3 && strings.EqualFold(tail[2], "inventoryItems") {
		s.handleAzureArcSCVMMInventoryItemsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[3:])
		return true
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureArcSCVMMTopLevelGet(w, path, subscriptionID, resourceGroup, collection, name, false)
	case http.MethodHead:
		s.handleAzureArcSCVMMTopLevelGet(w, path, subscriptionID, resourceGroup, collection, name, true)
	case http.MethodPut:
		s.handleAzureArcSCVMMTopLevelCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, collection, name)
	case http.MethodPatch:
		s.handleAzureArcSCVMMTopLevelPatch(w, r, path, subscriptionID, resourceGroup, collection, name)
	case http.MethodDelete:
		s.handleAzureArcSCVMMTopLevelDelete(w, path, subscriptionID, resourceGroup, collection, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	return true
}

func (s *Server) handleAzureArcSCVMMInventoryItemsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, vmmServerName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureArcSCVMMInventoryItemList(w, path, subscriptionID, resourceGroup, vmmServerName)
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
		s.handleAzureArcSCVMMInventoryItemGet(w, path, subscriptionID, resourceGroup, vmmServerName, name)
	case http.MethodPut, http.MethodPost:
		s.handleAzureArcSCVMMInventoryItemCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, vmmServerName, name)
	case http.MethodDelete:
		s.handleAzureArcSCVMMInventoryItemDelete(w, path, subscriptionID, resourceGroup, vmmServerName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceRoute(w http.ResponseWriter, r *http.Request, path, resourceURI string, tail []string) {
	if len(tail) == 0 || !strings.EqualFold(tail[0], "virtualMachineInstances") {
		respondAzureImplemented(w, path)
		return
	}
	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			s.handleAzureArcSCVMMVirtualMachineInstanceList(w, resourceURI)
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
			s.handleAzureArcSCVMMVirtualMachineInstanceGet(w, path, resourceURI, name)
		case http.MethodPut:
			s.handleAzureArcSCVMMVirtualMachineInstanceCreateOrUpdate(w, r, path, resourceURI, name)
		case http.MethodPatch:
			s.handleAzureArcSCVMMVirtualMachineInstancePatch(w, r, path, resourceURI, name)
		case http.MethodDelete:
			s.handleAzureArcSCVMMVirtualMachineInstanceDelete(w, path, resourceURI, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "start":
		if r.Method == http.MethodPost {
			s.handleAzureArcSCVMMVirtualMachineInstanceAction(w, path, resourceURI, name, "start", nil)
			return
		}
	case "stop":
		if r.Method == http.MethodPost {
			s.handleAzureArcSCVMMVirtualMachineInstanceStop(w, r, path, resourceURI, name)
			return
		}
	case "restart":
		if r.Method == http.MethodPost {
			s.handleAzureArcSCVMMVirtualMachineInstanceAction(w, path, resourceURI, name, "restart", nil)
			return
		}
	case "createcheckpoint":
		if r.Method == http.MethodPost {
			s.handleAzureArcSCVMMCheckpointAction(w, r, path, resourceURI, name, "create")
			return
		}
	case "deletecheckpoint":
		if r.Method == http.MethodPost {
			s.handleAzureArcSCVMMCheckpointAction(w, r, path, resourceURI, name, "delete")
			return
		}
	case "restorecheckpoint":
		if r.Method == http.MethodPost {
			s.handleAzureArcSCVMMCheckpointAction(w, r, path, resourceURI, name, "restore")
			return
		}
	case "guestagents":
		s.handleAzureArcSCVMMGuestAgentRoute(w, r, path, resourceURI, name, tail[3:])
		return
	case "hybrididentitymetadata":
		s.handleAzureArcSCVMMHybridIdentityMetadataRoute(w, r, path, resourceURI, name, tail[3:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureArcSCVMMGuestAgentRoute(w http.ResponseWriter, r *http.Request, path, resourceURI, vmName string, tail []string) {
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
		s.handleAzureArcSCVMMGuestAgentGet(w, path, resourceURI, name)
	case http.MethodPut:
		s.handleAzureArcSCVMMGuestAgentCreateOrUpdate(w, r, path, resourceURI, name)
	case http.MethodDelete:
		s.handleAzureArcSCVMMGuestAgentDelete(w, path, resourceURI, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureArcSCVMMHybridIdentityMetadataRoute(w http.ResponseWriter, r *http.Request, path, resourceURI, vmName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureArcSCVMMHybridIdentityMetadataList(w, path, resourceURI)
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
		s.handleAzureArcSCVMMHybridIdentityMetadataGet(w, path, resourceURI, name)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (s *Server) handleAzureArcSCVMMTopLevelCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureArcSCVMMString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	extendedLocation := azureArcSCVMMObject(payload["extendedLocation"])
	if len(extendedLocation) == 0 || strings.TrimSpace(azureArcSCVMMString(extendedLocation["type"])) == "" || strings.TrimSpace(azureArcSCVMMString(extendedLocation["name"])) == "" {
		respondAzureInvalidRequest(w, path, "extendedLocation.type and extendedLocation.name are required")
		return
	}
	properties := azureArcSCVMMObject(payload["properties"])
	if !azureArcSCVMMValidateTopLevelProperties(w, path, collection, properties) {
		return
	}

	now := time.Now().UTC()
	record := &azureArcSCVMMResource{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		Collection:       collection,
		Name:             name,
		Location:         location,
		Kind:             strings.TrimSpace(azureArcSCVMMString(payload["kind"])),
		Tags:             azureArcSCVMMStringMap(payload["tags"]),
		ExtendedLocation: extendedLocation,
		Properties: azureArcSCVMMMergeObject(map[string]any{
			"provisioningState": "Succeeded",
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureArcSCVMMResources[azureArcSCVMMStorageKey(azureArcSCVMMResourceID(collection, subscriptionID, resourceGroup, name))] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureArcSCVMMResourceFixture(record))
}

func (s *Server) handleAzureArcSCVMMTopLevelPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, collection, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	id := azureArcSCVMMResourceID(collection, subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMResources[azureArcSCVMMStorageKey(id)]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcSCVMMNotFound(w, path, collection+" not found")
		return
	}
	if location := strings.TrimSpace(azureArcSCVMMString(payload["location"])); location != "" {
		record.Location = location
	}
	if kind := strings.TrimSpace(azureArcSCVMMString(payload["kind"])); kind != "" {
		record.Kind = kind
	}
	if tags := azureArcSCVMMStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if extendedLocation := azureArcSCVMMObject(payload["extendedLocation"]); len(extendedLocation) != 0 {
		record.ExtendedLocation = extendedLocation
	}
	if properties := azureArcSCVMMObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureArcSCVMMMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcSCVMMResourceFixture(record))
}

func (s *Server) handleAzureArcSCVMMTopLevelGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMResources[azureArcSCVMMStorageKey(azureArcSCVMMResourceID(collection, subscriptionID, resourceGroup, name))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcSCVMMNotFound(w, path, collection+" not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMResourceFixture(record))
}

func (s *Server) handleAzureArcSCVMMTopLevelDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, collection, name string) {
	id := azureArcSCVMMResourceID(collection, subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	key := azureArcSCVMMStorageKey(id)
	record := s.azureArcSCVMMResources[key]
	delete(s.azureArcSCVMMResources, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMResourceFixture(record))
}

func (s *Server) handleAzureArcSCVMMTopLevelListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup, collection string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureArcSCVMMResources {
		if record.Collection == collection && record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) {
			items = append(items, azureArcSCVMMResourceFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureArcSCVMMString(items[i]["id"])) < strings.ToLower(azureArcSCVMMString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcSCVMMTopLevelListBySubscription(w http.ResponseWriter, subscriptionID, collection string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureArcSCVMMResources {
		if record.Collection == collection && record.SubscriptionID == subscriptionID {
			items = append(items, azureArcSCVMMResourceFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureArcSCVMMString(items[i]["id"])) < strings.ToLower(azureArcSCVMMString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcSCVMMInventoryItemCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, vmmServerName, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureArcSCVMMVMMServerExists(subscriptionID, resourceGroup, vmmServerName) {
		azureArcSCVMMNotFound(w, path, "vmm server not found")
		return
	}
	properties := azureArcSCVMMObject(payload["properties"])
	inventoryType := strings.TrimSpace(azureArcSCVMMString(properties["inventoryType"]))
	if inventoryType == "" {
		respondAzureInvalidRequest(w, path, "properties.inventoryType is required")
		return
	}
	location := strings.TrimSpace(azureArcSCVMMString(payload["location"]))
	if location == "" {
		location = "eastus"
	}
	now := time.Now().UTC()
	record := &azureArcSCVMMInventoryItem{
		SubscriptionID:   subscriptionID,
		ResourceGroup:    resourceGroup,
		VMMServerName:    vmmServerName,
		Name:             name,
		Location:         location,
		Kind:             strings.TrimSpace(azureArcSCVMMString(payload["kind"])),
		Tags:             azureArcSCVMMStringMap(payload["tags"]),
		ExtendedLocation: azureArcSCVMMObject(payload["extendedLocation"]),
		Properties: azureArcSCVMMMergeObject(map[string]any{
			"inventoryType":     inventoryType,
			"inventoryItemName": name,
			"managedResourceId": azureArcSCVMMInventoryItemID(subscriptionID, resourceGroup, vmmServerName, name),
			"uuid":              azureArcSCVMMStableToken(name, inventoryType),
			"provisioningState": "Succeeded",
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.providerStorageMu.Lock()
	s.azureArcSCVMMInventoryItems[azureArcSCVMMStorageKey(azureArcSCVMMInventoryItemID(subscriptionID, resourceGroup, vmmServerName, name))] = record
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcSCVMMInventoryItemFixture(record))
}

func (s *Server) handleAzureArcSCVMMInventoryItemGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, vmmServerName, name string) {
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMInventoryItems[azureArcSCVMMStorageKey(azureArcSCVMMInventoryItemID(subscriptionID, resourceGroup, vmmServerName, name))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcSCVMMNotFound(w, path, "inventory item not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMInventoryItemFixture(record))
}

func (s *Server) handleAzureArcSCVMMInventoryItemDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, vmmServerName, name string) {
	id := azureArcSCVMMInventoryItemID(subscriptionID, resourceGroup, vmmServerName, name)
	s.providerStorageMu.Lock()
	key := azureArcSCVMMStorageKey(id)
	record := s.azureArcSCVMMInventoryItems[key]
	delete(s.azureArcSCVMMInventoryItems, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMInventoryItemFixture(record))
}

func (s *Server) handleAzureArcSCVMMInventoryItemList(w http.ResponseWriter, path, subscriptionID, resourceGroup, vmmServerName string) {
	if !s.azureArcSCVMMVMMServerExists(subscriptionID, resourceGroup, vmmServerName) {
		azureArcSCVMMNotFound(w, path, "vmm server not found")
		return
	}
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureArcSCVMMInventoryItems {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) && strings.EqualFold(record.VMMServerName, vmmServerName) {
			items = append(items, azureArcSCVMMInventoryItemFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureArcSCVMMString(items[i]["id"])) < strings.ToLower(azureArcSCVMMString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	extendedLocation := azureArcSCVMMObject(payload["extendedLocation"])
	if len(extendedLocation) == 0 || strings.TrimSpace(azureArcSCVMMString(extendedLocation["type"])) == "" || strings.TrimSpace(azureArcSCVMMString(extendedLocation["name"])) == "" {
		respondAzureInvalidRequest(w, path, "extendedLocation.type and extendedLocation.name are required")
		return
	}
	properties := azureArcSCVMMObject(payload["properties"])
	if len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties are required")
		return
	}
	now := time.Now().UTC()
	record := &azureArcSCVMMVirtualMachineInstance{
		ResourceURI:      resourceURI,
		Name:             name,
		Location:         strings.TrimSpace(azureArcSCVMMString(payload["location"])),
		Tags:             azureArcSCVMMStringMap(payload["tags"]),
		ExtendedLocation: extendedLocation,
		Properties: azureArcSCVMMMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"powerState":        "Stopped",
		}, properties),
		PowerState:  "Stopped",
		Checkpoints: []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.providerStorageMu.Lock()
	vmKey := azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))
	s.azureArcSCVMMVirtualMachineInstances[vmKey] = record
	hybridID := azureArcSCVMMHybridIdentityMetadataID(resourceURI)
	hybridKey := azureArcSCVMMStorageKey(hybridID)
	if _, exists := s.azureArcSCVMMHybridIdentityMetadata[hybridKey]; !exists {
		s.azureArcSCVMMHybridIdentityMetadata[hybridKey] = &azureArcSCVMMHybridIdentityMetadata{
			ResourceURI: resourceURI,
			Name:        "default",
			Properties: map[string]any{
				"resourceUid":       azureArcSCVMMStableToken(resourceURI, "resourceUid"),
				"publicKey":         "stackyard-public-key",
				"provisioningState": "Succeeded",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcSCVMMVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstancePatch(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMVirtualMachineInstances[azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	if tags := azureArcSCVMMStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if properties := azureArcSCVMMObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureArcSCVMMMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, azureArcSCVMMVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceGet(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMVirtualMachineInstances[azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceDelete(w http.ResponseWriter, path, resourceURI, name string) {
	id := azureArcSCVMMVirtualMachineInstanceID(resourceURI)
	s.providerStorageMu.Lock()
	key := azureArcSCVMMStorageKey(id)
	record := s.azureArcSCVMMVirtualMachineInstances[key]
	delete(s.azureArcSCVMMVirtualMachineInstances, key)
	delete(s.azureArcSCVMMGuestAgents, azureArcSCVMMStorageKey(azureArcSCVMMGuestAgentID(resourceURI)))
	delete(s.azureArcSCVMMHybridIdentityMetadata, azureArcSCVMMStorageKey(azureArcSCVMMHybridIdentityMetadataID(resourceURI)))
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMVirtualMachineInstanceFixture(record))
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceList(w http.ResponseWriter, resourceURI string) {
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMVirtualMachineInstances[azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))]
	s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	if record != nil {
		items = append(items, azureArcSCVMMVirtualMachineInstanceFixture(record))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceAction(w http.ResponseWriter, path, resourceURI, name, action string, extra map[string]any) {
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMVirtualMachineInstances[azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	switch action {
	case "start":
		record.PowerState = "Running"
	case "stop":
		record.PowerState = "Stopped"
	case "restart":
		record.PowerState = "Running"
	}
	record.Properties["powerState"] = record.PowerState
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	payload := map[string]any{
		"status":      "Succeeded",
		"action":      action,
		"resourceUri": resourceURI,
		"powerState":  record.PowerState,
		"target":      azureArcSCVMMVirtualMachineInstanceID(resourceURI),
		"requestPath": path,
		"completedAt": time.Now().UTC().Format(time.RFC3339),
	}
	for key, value := range extra {
		payload[key] = value
	}
	respondJSON(w, http.StatusOK, payload)
}

func (s *Server) handleAzureArcSCVMMVirtualMachineInstanceStop(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	extra := map[string]any{}
	if skipShutdown, found := payload["skipShutdown"]; found {
		extra["skipShutdown"] = skipShutdown
	}
	s.handleAzureArcSCVMMVirtualMachineInstanceAction(w, path, resourceURI, name, "stop", extra)
}

func (s *Server) handleAzureArcSCVMMCheckpointAction(w http.ResponseWriter, r *http.Request, path, resourceURI, name, action string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMVirtualMachineInstances[azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	checkpointName := "checkpoint-a"
	if propertiesName := strings.TrimSpace(azureArcSCVMMString(payload["checkpointName"])); propertiesName != "" {
		checkpointName = propertiesName
	}
	switch action {
	case "create":
		if !azureArcSCVMMContainsString(record.Checkpoints, checkpointName) {
			record.Checkpoints = append(record.Checkpoints, checkpointName)
		}
	case "delete":
		record.Checkpoints = azureArcSCVMMWithoutString(record.Checkpoints, checkpointName)
	case "restore":
		if !azureArcSCVMMContainsString(record.Checkpoints, checkpointName) {
			s.providerStorageMu.Unlock()
			respondAzureInvalidRequest(w, path, "checkpoint not found")
			return
		}
		record.PowerState = "Stopped"
		record.Properties["powerState"] = record.PowerState
	}
	record.Properties["checkpoints"] = append([]string(nil), record.Checkpoints...)
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "Succeeded",
		"action":         action + "Checkpoint",
		"checkpointName": checkpointName,
		"checkpoints":    append([]string(nil), record.Checkpoints...),
	})
}

func (s *Server) handleAzureArcSCVMMGuestAgentCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureArcSCVMMReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureArcSCVMMVirtualMachineInstanceExists(resourceURI) {
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	properties := azureArcSCVMMObject(payload["properties"])
	credentials := azureArcSCVMMObject(properties["credentials"])
	if strings.TrimSpace(azureArcSCVMMString(credentials["username"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.credentials.username is required")
		return
	}
	now := time.Now().UTC()
	record := &azureArcSCVMMGuestAgent{
		ResourceURI: resourceURI,
		Name:        name,
		Properties: azureArcSCVMMMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"status":            "Installed",
			"uuid":              azureArcSCVMMStableToken(resourceURI, name, "guest-agent"),
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.providerStorageMu.Lock()
	s.azureArcSCVMMGuestAgents[azureArcSCVMMStorageKey(azureArcSCVMMGuestAgentID(resourceURI))] = record
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusCreated, azureArcSCVMMGuestAgentFixture(record))
}

func (s *Server) handleAzureArcSCVMMGuestAgentGet(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMGuestAgents[azureArcSCVMMStorageKey(azureArcSCVMMGuestAgentID(resourceURI))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcSCVMMNotFound(w, path, "guest agent not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMGuestAgentFixture(record))
}

func (s *Server) handleAzureArcSCVMMGuestAgentDelete(w http.ResponseWriter, path, resourceURI, name string) {
	id := azureArcSCVMMGuestAgentID(resourceURI)
	s.providerStorageMu.Lock()
	key := azureArcSCVMMStorageKey(id)
	record := s.azureArcSCVMMGuestAgents[key]
	delete(s.azureArcSCVMMGuestAgents, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMGuestAgentFixture(record))
}

func (s *Server) handleAzureArcSCVMMHybridIdentityMetadataGet(w http.ResponseWriter, path, resourceURI, name string) {
	if !s.azureArcSCVMMVirtualMachineInstanceExists(resourceURI) {
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMHybridIdentityMetadata[azureArcSCVMMStorageKey(azureArcSCVMMHybridIdentityMetadataID(resourceURI))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureArcSCVMMNotFound(w, path, "hybrid identity metadata not found")
		return
	}
	respondJSON(w, http.StatusOK, azureArcSCVMMHybridIdentityMetadataFixture(record))
}

func (s *Server) handleAzureArcSCVMMHybridIdentityMetadataList(w http.ResponseWriter, path, resourceURI string) {
	if !s.azureArcSCVMMVirtualMachineInstanceExists(resourceURI) {
		azureArcSCVMMNotFound(w, path, "virtual machine instance not found")
		return
	}
	s.providerStorageMu.Lock()
	record := s.azureArcSCVMMHybridIdentityMetadata[azureArcSCVMMStorageKey(azureArcSCVMMHybridIdentityMetadataID(resourceURI))]
	s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	if record != nil {
		items = append(items, azureArcSCVMMHybridIdentityMetadataFixture(record))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) azureArcSCVMMVMMServerExists(subscriptionID, resourceGroup, name string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureArcSCVMMResources[azureArcSCVMMStorageKey(azureArcSCVMMResourceID("vmmServers", subscriptionID, resourceGroup, name))] != nil
}

func (s *Server) azureArcSCVMMVirtualMachineInstanceExists(resourceURI string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureArcSCVMMVirtualMachineInstances[azureArcSCVMMStorageKey(azureArcSCVMMVirtualMachineInstanceID(resourceURI))] != nil
}

func azureIsArcSCVMMPath(segments []string) bool {
	if len(segments) >= 2 && strings.EqualFold(segments[0], "providers") && strings.EqualFold(segments[1], "Microsoft.ScVmm") {
		return true
	}
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ScVmm") {
			return true
		}
	}
	return false
}

func azureArcSCVMMAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureArcSCVMMRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if len(segments) >= 4 && strings.EqualFold(segments[2], "providers") && strings.EqualFold(segments[3], "Microsoft.ScVmm") {
		return subscriptionID, "", segments[4:], true
	}
	if len(segments) >= 6 &&
		strings.EqualFold(segments[2], "resourceGroups") &&
		strings.TrimSpace(segments[3]) != "" &&
		strings.EqualFold(segments[4], "providers") &&
		strings.EqualFold(segments[5], "Microsoft.ScVmm") {
		return subscriptionID, strings.TrimSpace(segments[3]), segments[6:], true
	}
	return "", "", nil, false
}

func azureArcSCVMMResourceURIRouteContext(segments []string) (resourceURI string, tail []string, ok bool) {
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ScVmm") {
			if idx == 0 {
				return "", nil, false
			}
			return "/" + strings.Join(segments[:idx], "/"), segments[idx+2:], true
		}
	}
	return "", nil, false
}

func azureArcSCVMMOperationsFixture() []map[string]any {
	return []map[string]any{
		azureArcSCVMMOperation("Microsoft.ScVmm/vmmServers/read", "Read VMM server"),
		azureArcSCVMMOperation("Microsoft.ScVmm/vmmServers/write", "Create or update VMM server"),
		azureArcSCVMMOperation("Microsoft.ScVmm/vmmServers/delete", "Delete VMM server"),
		azureArcSCVMMOperation("Microsoft.ScVmm/clouds/read", "Read cloud"),
		azureArcSCVMMOperation("Microsoft.ScVmm/clouds/write", "Create or update cloud"),
		azureArcSCVMMOperation("Microsoft.ScVmm/clouds/delete", "Delete cloud"),
		azureArcSCVMMOperation("Microsoft.ScVmm/availabilitySets/read", "Read availability set"),
		azureArcSCVMMOperation("Microsoft.ScVmm/virtualNetworks/read", "Read virtual network"),
		azureArcSCVMMOperation("Microsoft.ScVmm/virtualMachineTemplates/read", "Read virtual machine template"),
		azureArcSCVMMOperation("Microsoft.ScVmm/inventoryItems/read", "Read inventory item"),
		azureArcSCVMMOperation("Microsoft.ScVmm/virtualMachineInstances/read", "Read virtual machine instance"),
		azureArcSCVMMOperation("Microsoft.ScVmm/virtualMachineInstances/write", "Create or update virtual machine instance"),
		azureArcSCVMMOperation("Microsoft.ScVmm/virtualMachineInstances/start/action", "Start virtual machine instance"),
		azureArcSCVMMOperation("Microsoft.ScVmm/virtualMachineInstances/stop/action", "Stop virtual machine instance"),
		azureArcSCVMMOperation("Microsoft.ScVmm/guestAgents/write", "Create or update guest agent"),
		azureArcSCVMMOperation("Microsoft.ScVmm/hybridIdentityMetadata/read", "Read hybrid identity metadata"),
	}
}

func azureArcSCVMMOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":  "Microsoft ScVmm",
			"resource":  "arcScvmm",
			"operation": description,
		},
		"isDataAction": false,
	}
}

func azureArcSCVMMResourceFixture(record *azureArcSCVMMResource) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcSCVMMResourceID(record.Collection, record.SubscriptionID, record.ResourceGroup, record.Name)
	return map[string]any{
		"id":               id,
		"name":             record.Name,
		"type":             "Microsoft.ScVmm/" + record.Collection,
		"location":         record.Location,
		"kind":             record.Kind,
		"tags":             azureArcSCVMMStringMapAny(record.Tags),
		"extendedLocation": azureArcSCVMMCloneMap(record.ExtendedLocation),
		"properties":       azureArcSCVMMCloneMap(record.Properties),
		"etag":             azureArcSCVMMETag(id, record.UpdatedAt),
		"systemData":       azureArcSCVMMSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcSCVMMInventoryItemFixture(record *azureArcSCVMMInventoryItem) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcSCVMMInventoryItemID(record.SubscriptionID, record.ResourceGroup, record.VMMServerName, record.Name)
	return map[string]any{
		"id":               id,
		"name":             record.Name,
		"type":             "Microsoft.ScVmm/vmmServers/inventoryItems",
		"location":         record.Location,
		"kind":             record.Kind,
		"tags":             azureArcSCVMMStringMapAny(record.Tags),
		"extendedLocation": azureArcSCVMMCloneMap(record.ExtendedLocation),
		"properties":       azureArcSCVMMCloneMap(record.Properties),
		"etag":             azureArcSCVMMETag(id, record.UpdatedAt),
		"systemData":       azureArcSCVMMSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcSCVMMVirtualMachineInstanceFixture(record *azureArcSCVMMVirtualMachineInstance) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcSCVMMVirtualMachineInstanceID(record.ResourceURI)
	properties := azureArcSCVMMMergeObject(record.Properties, map[string]any{
		"powerState":        record.PowerState,
		"provisioningState": "Succeeded",
		"checkpoints":       append([]string(nil), record.Checkpoints...),
	})
	return map[string]any{
		"id":               id,
		"name":             record.Name,
		"type":             "Microsoft.ScVmm/virtualMachineInstances",
		"location":         record.Location,
		"tags":             azureArcSCVMMStringMapAny(record.Tags),
		"extendedLocation": azureArcSCVMMCloneMap(record.ExtendedLocation),
		"properties":       properties,
		"etag":             azureArcSCVMMETag(id, record.UpdatedAt),
		"systemData":       azureArcSCVMMSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcSCVMMGuestAgentFixture(record *azureArcSCVMMGuestAgent) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcSCVMMGuestAgentID(record.ResourceURI)
	return map[string]any{
		"id":         id,
		"name":       record.Name,
		"type":       "Microsoft.ScVmm/virtualMachineInstances/guestAgents",
		"properties": azureArcSCVMMCloneMap(record.Properties),
		"etag":       azureArcSCVMMETag(id, record.UpdatedAt),
		"systemData": azureArcSCVMMSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcSCVMMHybridIdentityMetadataFixture(record *azureArcSCVMMHybridIdentityMetadata) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	id := azureArcSCVMMHybridIdentityMetadataID(record.ResourceURI)
	return map[string]any{
		"id":         id,
		"name":       record.Name,
		"type":       "Microsoft.ScVmm/virtualMachineInstances/hybridIdentityMetadata",
		"properties": azureArcSCVMMCloneMap(record.Properties),
		"etag":       azureArcSCVMMETag(id, record.UpdatedAt),
		"systemData": azureArcSCVMMSystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureArcSCVMMResourceID(collection, subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ScVmm/%s/%s", subscriptionID, resourceGroup, collection, name)
}

func azureArcSCVMMInventoryItemID(subscriptionID, resourceGroup, vmmServerName, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ScVmm/vmmServers/%s/inventoryItems/%s", subscriptionID, resourceGroup, vmmServerName, name)
}

func azureArcSCVMMVirtualMachineInstanceID(resourceURI string) string {
	return strings.TrimRight(resourceURI, "/") + "/providers/Microsoft.ScVmm/virtualMachineInstances/default"
}

func azureArcSCVMMGuestAgentID(resourceURI string) string {
	return azureArcSCVMMVirtualMachineInstanceID(resourceURI) + "/guestAgents/default"
}

func azureArcSCVMMHybridIdentityMetadataID(resourceURI string) string {
	return azureArcSCVMMVirtualMachineInstanceID(resourceURI) + "/hybridIdentityMetadata/default"
}

func azureArcSCVMMReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureArcSCVMMValidateTopLevelProperties(w http.ResponseWriter, path, collection string, properties map[string]any) bool {
	switch collection {
	case "vmmServers":
		if strings.TrimSpace(azureArcSCVMMString(properties["fqdn"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.fqdn is required")
			return false
		}
	case "clouds", "virtualNetworks", "virtualMachineTemplates":
		if strings.TrimSpace(azureArcSCVMMString(properties["vmmServerId"])) == "" {
			respondAzureInvalidRequest(w, path, "properties.vmmServerId is required")
			return false
		}
	}
	return true
}

func azureArcSCVMMNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": message,
			"target":  path,
		},
	})
}

func azureArcSCVMMString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func azureArcSCVMMObject(value any) map[string]any {
	if value == nil {
		return nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return azureArcSCVMMCloneMap(obj)
}

func azureArcSCVMMStringMap(value any) map[string]string {
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

func azureArcSCVMMStringMapAny(tags map[string]string) map[string]any {
	if len(tags) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func azureArcSCVMMMergeObject(base, overlay map[string]any) map[string]any {
	out := azureArcSCVMMCloneMap(base)
	for key, value := range overlay {
		if nested, ok := value.(map[string]any); ok {
			existing, _ := out[key].(map[string]any)
			out[key] = azureArcSCVMMMergeObject(existing, nested)
			continue
		}
		out[key] = value
	}
	return out
}

func azureArcSCVMMCloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureArcSCVMMCloneMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func azureArcSCVMMETag(id string, updatedAt time.Time) string {
	sum := sha1.Sum([]byte(id + "|" + updatedAt.Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureArcSCVMMSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"createdAt":          createdAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
	}
}

func azureArcSCVMMStorageKey(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func azureArcSCVMMStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func azureArcSCVMMContainsString(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func azureArcSCVMMWithoutString(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !strings.EqualFold(value, target) {
			out = append(out, value)
		}
	}
	return out
}
