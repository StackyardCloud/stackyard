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

const azureChaosStudioAPIVersion = "2025-01-01"

type azureChaosStudioTarget struct {
	ResourceURI string
	Name        string
	Location    string
	Tags        map[string]string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ETag        string
}

type azureChaosStudioCapability struct {
	ResourceURI string
	TargetName  string
	Name        string
	Location    string
	Tags        map[string]string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ETag        string
}

type azureChaosStudioExperiment struct {
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

type azureChaosStudioExecution struct {
	ExperimentID   string
	SubscriptionID string
	ResourceGroup  string
	ExperimentName string
	ID             string
	Location       string
	Status         string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureChaosStudioOperationStatus struct {
	SubscriptionID  string
	Location        string
	ID              string
	ResourceID      string
	Status          string
	PercentComplete int
	StartTime       time.Time
	EndTime         time.Time
	Error           map[string]any
}

func (s *Server) handleAzureChaosStudioRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureChaosStudioIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureChaosStudioAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Chaos") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureChaosStudioOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	if resourceURI, tail, ok := azureChaosStudioExtensionRouteContext(segments); ok && len(tail) > 0 && strings.EqualFold(tail[0], "targets") {
		s.handleAzureChaosStudioTargetsRoute(w, r, path, resourceURI, tail)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureChaosStudioRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations":
		s.handleAzureChaosStudioLocationsRoute(w, r, path, subscriptionID, tail[1:])
		return true
	case "experiments":
		s.handleAzureChaosStudioExperimentsRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		return false
	}
}

func azureChaosStudioIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Chaos") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	if resourceURI, tail, ok := azureChaosStudioExtensionRouteContext(segments); ok && resourceURI != "" {
		if len(tail) > 0 && strings.EqualFold(tail[0], "targets") {
			return true
		}
	}
	_, _, tail, ok := azureChaosStudioRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations", "experiments":
		return true
	default:
		return false
	}
}

func azureChaosStudioAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureChaosStudioRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Chaos") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Chaos") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureChaosStudioExtensionRouteContext(segments []string) (resourceURI string, tail []string, ok bool) {
	last := -1
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Chaos") {
			last = idx
		}
	}
	if last <= 0 || last+2 > len(segments) {
		return "", nil, false
	}
	resourceURI = "/" + strings.Join(segments[:last], "/")
	resourceURI = strings.TrimSpace(resourceURI)
	if resourceURI == "" || resourceURI == "/" {
		return "", nil, false
	}
	return resourceURI, segments[last+2:], true
}

func (s *Server) handleAzureChaosStudioLocationsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
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
	case "targettypes":
		if len(tail) == 2 && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureChaosStudioTargetTypesFixture(subscriptionID, location)})
			return
		}
		if len(tail) == 3 && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			s.handleAzureChaosStudioTargetTypeGet(w, path, subscriptionID, location, tail[2], r.Method == http.MethodHead)
			return
		}
		if len(tail) >= 4 && strings.EqualFold(tail[3], "capabilityTypes") {
			s.handleAzureChaosStudioCapabilityTypesRoute(w, r, path, subscriptionID, location, tail[2], tail[3:])
			return
		}
	case "operationstatuses":
		if len(tail) == 3 && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
			s.handleAzureChaosStudioOperationStatusGet(w, path, subscriptionID, location, tail[2], r.Method == http.MethodHead)
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureChaosStudioTargetTypeGet(w http.ResponseWriter, path, subscriptionID, location, name string, head bool) {
	for _, item := range azureChaosStudioTargetTypesFixture(subscriptionID, location) {
		if strings.EqualFold(azureChaosStudioString(item["name"]), name) {
			if head {
				w.WriteHeader(http.StatusOK)
				return
			}
			respondJSON(w, http.StatusOK, item)
			return
		}
	}
	respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "target type", name))
}

func (s *Server) handleAzureChaosStudioCapabilityTypesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location, targetType string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureChaosStudioCapabilityTypesFixture(subscriptionID, location, targetType)})
		return
	}
	if len(tail) == 2 && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		name := tail[1]
		for _, item := range azureChaosStudioCapabilityTypesFixture(subscriptionID, location, targetType) {
			if strings.EqualFold(azureChaosStudioString(item["name"]), name) {
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusOK)
					return
				}
				respondJSON(w, http.StatusOK, item)
				return
			}
		}
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "capability type", name))
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureChaosStudioTargetsRoute(w http.ResponseWriter, r *http.Request, path, resourceURI string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureChaosStudioTargetsList(w, resourceURI)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureChaosStudioValidTargetName(name) {
		respondAzureInvalidRequest(w, path, "target name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureChaosStudioTargetGet(w, path, resourceURI, name, false)
		case http.MethodHead:
			s.handleAzureChaosStudioTargetGet(w, path, resourceURI, name, true)
		case http.MethodPut:
			s.handleAzureChaosStudioTargetCreateOrUpdate(w, r, path, resourceURI, name)
		case http.MethodDelete:
			s.handleAzureChaosStudioTargetDelete(w, path, resourceURI, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if strings.EqualFold(tail[2], "capabilities") {
		s.handleAzureChaosStudioCapabilitiesRoute(w, r, path, resourceURI, name, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureChaosStudioTargetsList(w http.ResponseWriter, resourceURI string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, target := range s.azureChaosStudioTargets {
		if strings.EqualFold(target.ResourceURI, resourceURI) {
			items = append(items, azureChaosStudioTargetFixture(target))
		}
	}
	azureChaosStudioSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureChaosStudioTargetGet(w http.ResponseWriter, path, resourceURI, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	target, ok := s.azureChaosStudioTargets[azureChaosStudioTargetKey(resourceURI, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "target", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureChaosStudioTargetFixture(target))
}

func (s *Server) handleAzureChaosStudioTargetCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureChaosStudioReadRequestBody(w, r, path)
	if !ok {
		return
	}

	now := time.Now().UTC()
	location := strings.TrimSpace(azureChaosStudioString(payload["location"]))
	if location == "" {
		location = "eastus"
	}
	properties := azureChaosStudioNormalizeTargetProperties(resourceURI, name, payload["properties"], now)
	tags := azureChaosStudioStringMap(payload["tags"])

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureChaosStudioTargetKey(resourceURI, name)
	target, exists := s.azureChaosStudioTargets[key]
	if exists {
		target.Location = location
		target.Tags = tags
		target.Properties = properties
		target.UpdatedAt = now
		target.ETag = azureChaosStudioETag(target.IDSeed(), now)
		respondJSON(w, http.StatusOK, azureChaosStudioTargetFixture(target))
		return
	}

	target = &azureChaosStudioTarget{
		ResourceURI: resourceURI,
		Name:        name,
		Location:    location,
		Tags:        tags,
		Properties:  properties,
		CreatedAt:   now,
		UpdatedAt:   now,
		ETag:        azureChaosStudioETag(resourceURI+"/"+name, now),
	}
	s.azureChaosStudioTargets[key] = target
	respondJSON(w, http.StatusCreated, azureChaosStudioTargetFixture(target))
}

func (t *azureChaosStudioTarget) IDSeed() string {
	return t.ResourceURI + "/providers/Microsoft.Chaos/targets/" + t.Name
}

func (s *Server) handleAzureChaosStudioTargetDelete(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureChaosStudioTargetKey(resourceURI, name)
	if _, ok := s.azureChaosStudioTargets[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "target", name))
		return
	}
	delete(s.azureChaosStudioTargets, key)
	prefix := strings.ToLower(azureChaosStudioTargetID(resourceURI, name) + "/capabilities/")
	for key := range s.azureChaosStudioCapabilities {
		if strings.HasPrefix(strings.ToLower(key), prefix) {
			delete(s.azureChaosStudioCapabilities, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureChaosStudioCapabilitiesRoute(w http.ResponseWriter, r *http.Request, path, resourceURI, targetName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureChaosStudioCapabilitiesList(w, path, resourceURI, targetName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureChaosStudioValidCapabilityName(name) {
		respondAzureInvalidRequest(w, path, "capability name is invalid")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureChaosStudioCapabilityGet(w, path, resourceURI, targetName, name, false)
	case http.MethodHead:
		s.handleAzureChaosStudioCapabilityGet(w, path, resourceURI, targetName, name, true)
	case http.MethodPut:
		s.handleAzureChaosStudioCapabilityCreateOrUpdate(w, r, path, resourceURI, targetName, name)
	case http.MethodDelete:
		s.handleAzureChaosStudioCapabilityDelete(w, path, resourceURI, targetName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureChaosStudioCapabilitiesList(w http.ResponseWriter, path, resourceURI, targetName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureChaosStudioTargets[azureChaosStudioTargetKey(resourceURI, targetName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "target", targetName))
		return
	}

	items := make([]map[string]any, 0)
	for _, capability := range s.azureChaosStudioCapabilities {
		if strings.EqualFold(capability.ResourceURI, resourceURI) && strings.EqualFold(capability.TargetName, targetName) {
			items = append(items, azureChaosStudioCapabilityFixture(capability))
		}
	}
	azureChaosStudioSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureChaosStudioCapabilityGet(w http.ResponseWriter, path, resourceURI, targetName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	capability, ok := s.azureChaosStudioCapabilities[azureChaosStudioCapabilityKey(resourceURI, targetName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "capability", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureChaosStudioCapabilityFixture(capability))
}

func (s *Server) handleAzureChaosStudioCapabilityCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, targetName, name string) {
	_, payload, ok := azureChaosStudioReadRequestBody(w, r, path)
	if !ok {
		return
	}

	now := time.Now().UTC()
	location := strings.TrimSpace(azureChaosStudioString(payload["location"]))
	if location == "" {
		location = "eastus"
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if _, ok := s.azureChaosStudioTargets[azureChaosStudioTargetKey(resourceURI, targetName)]; !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "target", targetName))
		return
	}

	properties := azureChaosStudioNormalizeCapabilityProperties(resourceURI, targetName, name, payload["properties"], now)
	tags := azureChaosStudioStringMap(payload["tags"])
	key := azureChaosStudioCapabilityKey(resourceURI, targetName, name)
	capability, exists := s.azureChaosStudioCapabilities[key]
	if exists {
		capability.Location = location
		capability.Tags = tags
		capability.Properties = properties
		capability.UpdatedAt = now
		capability.ETag = azureChaosStudioETag(capability.IDSeed(), now)
		respondJSON(w, http.StatusOK, azureChaosStudioCapabilityFixture(capability))
		return
	}

	capability = &azureChaosStudioCapability{
		ResourceURI: resourceURI,
		TargetName:  targetName,
		Name:        name,
		Location:    location,
		Tags:        tags,
		Properties:  properties,
		CreatedAt:   now,
		UpdatedAt:   now,
		ETag:        azureChaosStudioETag(resourceURI+"/"+targetName+"/"+name, now),
	}
	s.azureChaosStudioCapabilities[key] = capability
	respondJSON(w, http.StatusCreated, azureChaosStudioCapabilityFixture(capability))
}

func (c *azureChaosStudioCapability) IDSeed() string {
	return azureChaosStudioCapabilityID(c.ResourceURI, c.TargetName, c.Name)
}

func (s *Server) handleAzureChaosStudioCapabilityDelete(w http.ResponseWriter, path, resourceURI, targetName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureChaosStudioCapabilityKey(resourceURI, targetName, name)
	if _, ok := s.azureChaosStudioCapabilities[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "capability", name))
		return
	}
	delete(s.azureChaosStudioCapabilities, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureChaosStudioExperimentsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureChaosStudioExperimentsListBySubscription(w, r, subscriptionID)
			return
		}
		s.handleAzureChaosStudioExperimentsListByResourceGroup(w, r, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureChaosStudioValidExperimentName(name) {
		respondAzureInvalidRequest(w, path, "experiment name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureChaosStudioExperimentGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureChaosStudioExperimentGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureChaosStudioExperimentCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureChaosStudioExperimentPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureChaosStudioExperimentDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "start":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureChaosStudioExperimentStart(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "cancel":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureChaosStudioExperimentCancel(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "executions":
		s.handleAzureChaosStudioExperimentExecutionsRoute(w, r, path, subscriptionID, resourceGroup, name, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureChaosStudioExperimentsListBySubscription(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	runningOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("running")), "true")
	items := make([]map[string]any, 0)
	for _, experiment := range s.azureChaosStudioExperiments {
		if !strings.EqualFold(experiment.SubscriptionID, subscriptionID) {
			continue
		}
		if runningOnly && !s.azureChaosStudioExperimentHasRunningExecutionLocked(experiment.SubscriptionID, experiment.ResourceGroup, experiment.Name) {
			continue
		}
		items = append(items, azureChaosStudioExperimentFixture(experiment))
	}
	azureChaosStudioSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureChaosStudioExperimentsListByResourceGroup(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	runningOnly := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("running")), "true")
	items := make([]map[string]any, 0)
	for _, experiment := range s.azureChaosStudioExperiments {
		if strings.EqualFold(experiment.SubscriptionID, subscriptionID) && strings.EqualFold(experiment.ResourceGroup, resourceGroup) {
			if runningOnly && !s.azureChaosStudioExperimentHasRunningExecutionLocked(experiment.SubscriptionID, experiment.ResourceGroup, experiment.Name) {
				continue
			}
			items = append(items, azureChaosStudioExperimentFixture(experiment))
		}
	}
	azureChaosStudioSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureChaosStudioExperimentGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureChaosStudioExperimentFixture(experiment))
}

func (s *Server) handleAzureChaosStudioExperimentCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureChaosStudioReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureChaosStudioString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureChaosStudioObject(payload["properties"])
	selectors := azureChaosStudioSlice(properties["selectors"])
	steps := azureChaosStudioSlice(properties["steps"])
	if len(selectors) == 0 || len(steps) == 0 {
		respondAzureInvalidRequest(w, path, "properties.selectors and properties.steps are required")
		return
	}

	now := time.Now().UTC()
	normalized := azureChaosStudioNormalizeExperimentProperties(properties, now)
	tags := azureChaosStudioStringMap(payload["tags"])

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)
	experiment, exists := s.azureChaosStudioExperiments[key]
	if exists {
		experiment.Location = location
		experiment.Tags = tags
		experiment.Properties = normalized
		experiment.UpdatedAt = now
		experiment.ETag = azureChaosStudioETag(experiment.ID(), now)
		respondJSON(w, http.StatusOK, azureChaosStudioExperimentFixture(experiment))
		return
	}

	experiment = &azureChaosStudioExperiment{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           name,
		Location:       location,
		Tags:           tags,
		Properties:     normalized,
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureChaosStudioETag(azureChaosStudioExperimentID(subscriptionID, resourceGroup, name), now),
	}
	s.azureChaosStudioExperiments[key] = experiment
	respondJSON(w, http.StatusCreated, azureChaosStudioExperimentFixture(experiment))
}

func (e *azureChaosStudioExperiment) ID() string {
	return azureChaosStudioExperimentID(e.SubscriptionID, e.ResourceGroup, e.Name)
}

func (s *Server) handleAzureChaosStudioExperimentPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureChaosStudioReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", name))
		return
	}

	if tags := azureChaosStudioStringMap(payload["tags"]); tags != nil {
		experiment.Tags = tags
	}
	if rawProps := azureChaosStudioObject(payload["properties"]); rawProps != nil {
		experiment.Properties = azureChaosStudioMergeObjects(experiment.Properties, rawProps)
		experiment.Properties = azureChaosStudioNormalizeExperimentProperties(experiment.Properties, time.Now().UTC())
	}
	experiment.UpdatedAt = time.Now().UTC()
	experiment.ETag = azureChaosStudioETag(experiment.ID(), experiment.UpdatedAt)
	respondJSON(w, http.StatusOK, azureChaosStudioExperimentFixture(experiment))
}

func (s *Server) handleAzureChaosStudioExperimentDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)
	if _, ok := s.azureChaosStudioExperiments[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", name))
		return
	}
	delete(s.azureChaosStudioExperiments, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureChaosStudioExperimentStart(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, _, ok := azureChaosStudioReadOptionalRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", name))
		return
	}

	now := time.Now().UTC()
	operationID := azureChaosStudioGUIDLike(experiment.ID() + "/start/" + now.Format(time.RFC3339Nano))
	executionID := azureChaosStudioGUIDLike(experiment.ID() + "/execution/" + now.Add(time.Millisecond).Format(time.RFC3339Nano))
	execution := &azureChaosStudioExecution{
		ExperimentID:   experiment.ID(),
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ExperimentName: name,
		ID:             executionID,
		Location:       experiment.Location,
		Status:         "Running",
		CreatedAt:      now,
		UpdatedAt:      now,
		Properties: map[string]any{
			"status":          "Running",
			"startedOn":       now.Format(time.RFC3339),
			"experimentName":  name,
			"targetSelectors": azureChaosStudioSlice(experiment.Properties["selectors"]),
		},
	}
	s.azureChaosStudioExecutions[azureChaosStudioExecutionKey(experiment.ID(), executionID)] = execution
	s.azureChaosStudioOperationStatuses[azureChaosStudioOperationStatusKey(subscriptionID, experiment.Location, operationID)] = &azureChaosStudioOperationStatus{
		SubscriptionID:  subscriptionID,
		Location:        experiment.Location,
		ID:              operationID,
		ResourceID:      experiment.ID(),
		Status:          "Succeeded",
		PercentComplete: 100,
		StartTime:       now,
		EndTime:         now,
	}

	w.Header().Set("Location", "/azure/subscriptions/"+subscriptionID+"/providers/Microsoft.Chaos/locations/"+experiment.Location+"/operationStatuses/"+operationID+"?api-version="+azureChaosStudioAPIVersion)
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"executionId": executionID,
		"status":      "Accepted",
	})
}

func (s *Server) handleAzureChaosStudioExperimentCancel(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, _, ok := azureChaosStudioReadOptionalRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", name))
		return
	}

	execution := s.azureChaosStudioLatestExecutionLocked(experiment.ID())
	if execution == nil {
		respondAzureInvalidRequest(w, path, "experiment has no execution to cancel")
		return
	}

	now := time.Now().UTC()
	execution.Status = "Canceled"
	execution.UpdatedAt = now
	execution.Properties["status"] = "Canceled"
	execution.Properties["stoppedOn"] = now.Format(time.RFC3339)

	operationID := azureChaosStudioGUIDLike(experiment.ID() + "/cancel/" + now.Format(time.RFC3339Nano))
	s.azureChaosStudioOperationStatuses[azureChaosStudioOperationStatusKey(subscriptionID, experiment.Location, operationID)] = &azureChaosStudioOperationStatus{
		SubscriptionID:  subscriptionID,
		Location:        experiment.Location,
		ID:              operationID,
		ResourceID:      experiment.ID(),
		Status:          "Canceled",
		PercentComplete: 100,
		StartTime:       now,
		EndTime:         now,
	}

	w.Header().Set("Location", "/azure/subscriptions/"+subscriptionID+"/providers/Microsoft.Chaos/locations/"+experiment.Location+"/operationStatuses/"+operationID+"?api-version="+azureChaosStudioAPIVersion)
	respondJSON(w, http.StatusAccepted, map[string]any{
		"operationId": operationID,
		"executionId": execution.ID,
		"status":      "Accepted",
	})
}

func (s *Server) handleAzureChaosStudioExperimentExecutionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, experimentName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureChaosStudioExecutionsList(w, path, subscriptionID, resourceGroup, experimentName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	executionID := strings.TrimSpace(tail[1])
	if executionID == "" {
		respondAzureInvalidRequest(w, path, "execution id is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureChaosStudioExecutionGet(w, path, subscriptionID, resourceGroup, experimentName, executionID, false)
		case http.MethodHead:
			s.handleAzureChaosStudioExecutionGet(w, path, subscriptionID, resourceGroup, experimentName, executionID, true)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 3 && strings.EqualFold(tail[2], "getExecutionDetails") && r.Method == http.MethodPost {
		s.handleAzureChaosStudioExecutionDetails(w, path, subscriptionID, resourceGroup, experimentName, executionID)
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureChaosStudioExecutionsList(w http.ResponseWriter, path, subscriptionID, resourceGroup, experimentName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, experimentName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", experimentName))
		return
	}
	items := make([]map[string]any, 0)
	for _, execution := range s.azureChaosStudioExecutions {
		if strings.EqualFold(execution.ExperimentID, experiment.ID()) {
			items = append(items, azureChaosStudioExecutionFixture(execution))
		}
	}
	azureChaosStudioSortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureChaosStudioExecutionGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, experimentName, executionID string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, experimentName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", experimentName))
		return
	}
	execution, ok := s.azureChaosStudioExecutions[azureChaosStudioExecutionKey(experiment.ID(), executionID)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "execution", executionID))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureChaosStudioExecutionFixture(execution))
}

func (s *Server) handleAzureChaosStudioExecutionDetails(w http.ResponseWriter, path, subscriptionID, resourceGroup, experimentName, executionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, experimentName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "experiment", experimentName))
		return
	}
	execution, ok := s.azureChaosStudioExecutions[azureChaosStudioExecutionKey(experiment.ID(), executionID)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "execution", executionID))
		return
	}
	respondJSON(w, http.StatusOK, azureChaosStudioExecutionDetailsFixture(experiment, execution))
}

func (s *Server) handleAzureChaosStudioOperationStatusGet(w http.ResponseWriter, path, subscriptionID, location, operationID string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	status, ok := s.azureChaosStudioOperationStatuses[azureChaosStudioOperationStatusKey(subscriptionID, location, operationID)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureChaosStudioNotFound(path, "operation status", operationID))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureChaosStudioOperationStatusFixture(status))
}

func (s *Server) azureChaosStudioExperimentHasRunningExecutionLocked(subscriptionID, resourceGroup, name string) bool {
	experiment, ok := s.azureChaosStudioExperiments[azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name)]
	if !ok {
		return false
	}
	for _, execution := range s.azureChaosStudioExecutions {
		if strings.EqualFold(execution.ExperimentID, experiment.ID()) && strings.EqualFold(execution.Status, "Running") {
			return true
		}
	}
	return false
}

func (s *Server) azureChaosStudioLatestExecutionLocked(experimentID string) *azureChaosStudioExecution {
	var latest *azureChaosStudioExecution
	for _, execution := range s.azureChaosStudioExecutions {
		if !strings.EqualFold(execution.ExperimentID, experimentID) {
			continue
		}
		if latest == nil || execution.CreatedAt.After(latest.CreatedAt) {
			latest = execution
		}
	}
	return latest
}

func azureChaosStudioOperationsFixture() []map[string]any {
	return []map[string]any{
		{"name": "Microsoft.Chaos/operations/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Operations", "operation": "List Operations", "description": "List Chaos Studio operations"}},
		{"name": "Microsoft.Chaos/locations/targetTypes/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Target Types", "operation": "List Target Types", "description": "List target types"}},
		{"name": "Microsoft.Chaos/locations/targetTypes/capabilityTypes/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Capability Types", "operation": "List Capability Types", "description": "List capability types"}},
		{"name": "Microsoft.Chaos/targets/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Targets", "operation": "Read Target", "description": "Get or list targets"}},
		{"name": "Microsoft.Chaos/targets/write", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Targets", "operation": "Write Target", "description": "Create or update target"}},
		{"name": "Microsoft.Chaos/targets/delete", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Targets", "operation": "Delete Target", "description": "Delete target"}},
		{"name": "Microsoft.Chaos/targets/capabilities/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Capabilities", "operation": "Read Capability", "description": "Get or list capabilities"}},
		{"name": "Microsoft.Chaos/targets/capabilities/write", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Capabilities", "operation": "Write Capability", "description": "Create or update capability"}},
		{"name": "Microsoft.Chaos/targets/capabilities/delete", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Capabilities", "operation": "Delete Capability", "description": "Delete capability"}},
		{"name": "Microsoft.Chaos/experiments/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Experiments", "operation": "Read Experiment", "description": "Get or list experiments"}},
		{"name": "Microsoft.Chaos/experiments/write", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Experiments", "operation": "Write Experiment", "description": "Create or update experiment"}},
		{"name": "Microsoft.Chaos/experiments/delete", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Experiments", "operation": "Delete Experiment", "description": "Delete experiment"}},
		{"name": "Microsoft.Chaos/experiments/start/action", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Experiments", "operation": "Start Experiment", "description": "Start experiment"}},
		{"name": "Microsoft.Chaos/experiments/cancel/action", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Experiments", "operation": "Cancel Experiment", "description": "Cancel experiment"}},
		{"name": "Microsoft.Chaos/locations/operationStatuses/read", "display": map[string]any{"provider": "Microsoft Chaos", "resource": "Operation Statuses", "operation": "Get Operation Status", "description": "Get operation status"}},
	}
}

func azureChaosStudioTargetTypesFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"id":   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Chaos/locations/%s/targetTypes/Microsoft-VirtualMachine", subscriptionID, location),
			"name": "Microsoft-VirtualMachine",
			"type": "Microsoft.Chaos/locations/targetTypes",
			"properties": map[string]any{
				"displayName":   "Virtual Machine",
				"resourceTypes": []any{"Microsoft.Compute/virtualMachines"},
				"location":      location,
			},
		},
		{
			"id":   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Chaos/locations/%s/targetTypes/Microsoft-VirtualMachineScaleSet", subscriptionID, location),
			"name": "Microsoft-VirtualMachineScaleSet",
			"type": "Microsoft.Chaos/locations/targetTypes",
			"properties": map[string]any{
				"displayName":   "Virtual Machine Scale Set",
				"resourceTypes": []any{"Microsoft.Compute/virtualMachineScaleSets"},
				"location":      location,
			},
		},
	}
}

func azureChaosStudioCapabilityTypesFixture(subscriptionID, location, targetType string) []map[string]any {
	items := []map[string]any{}
	appendType := func(name, displayName, urn string) {
		items = append(items, map[string]any{
			"id":   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Chaos/locations/%s/targetTypes/%s/capabilityTypes/%s", subscriptionID, location, targetType, name),
			"name": name,
			"type": "Microsoft.Chaos/locations/targetTypes/capabilityTypes",
			"properties": map[string]any{
				"displayName": displayName,
				"urn":         urn,
				"location":    location,
				"targetType":  targetType,
			},
		})
	}
	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "microsoft-virtualmachinescaleset":
		appendType("CPUPressure-1.0", "CPU Pressure", "urn:csci:microsoft:vmss:cpu-pressure")
		appendType("Shutdown-1.0", "Shutdown", "urn:csci:microsoft:vmss:shutdown")
	default:
		appendType("CPUPressure-1.0", "CPU Pressure", "urn:csci:microsoft:vm:cpu-pressure")
		appendType("Shutdown-1.0", "Shutdown", "urn:csci:microsoft:vm:shutdown")
	}
	return items
}

func azureChaosStudioTargetFixture(target *azureChaosStudioTarget) map[string]any {
	return map[string]any{
		"id":         azureChaosStudioTargetID(target.ResourceURI, target.Name),
		"name":       target.Name,
		"type":       "Microsoft.Chaos/targets",
		"location":   target.Location,
		"tags":       azureChaosStudioAnyMapFromStringMap(target.Tags),
		"properties": azureChaosStudioCloneAnyMap(target.Properties),
		"systemData": azureChaosStudioSystemData(target.CreatedAt, target.UpdatedAt),
		"etag":       target.ETag,
	}
}

func azureChaosStudioCapabilityFixture(capability *azureChaosStudioCapability) map[string]any {
	return map[string]any{
		"id":         azureChaosStudioCapabilityID(capability.ResourceURI, capability.TargetName, capability.Name),
		"name":       capability.Name,
		"type":       "Microsoft.Chaos/targets/capabilities",
		"location":   capability.Location,
		"tags":       azureChaosStudioAnyMapFromStringMap(capability.Tags),
		"properties": azureChaosStudioCloneAnyMap(capability.Properties),
		"systemData": azureChaosStudioSystemData(capability.CreatedAt, capability.UpdatedAt),
		"etag":       capability.ETag,
	}
}

func azureChaosStudioExperimentFixture(experiment *azureChaosStudioExperiment) map[string]any {
	return map[string]any{
		"id":         experiment.ID(),
		"name":       experiment.Name,
		"type":       "Microsoft.Chaos/experiments",
		"location":   experiment.Location,
		"tags":       azureChaosStudioAnyMapFromStringMap(experiment.Tags),
		"properties": azureChaosStudioCloneAnyMap(experiment.Properties),
		"systemData": azureChaosStudioSystemData(experiment.CreatedAt, experiment.UpdatedAt),
		"etag":       experiment.ETag,
	}
}

func azureChaosStudioExecutionFixture(execution *azureChaosStudioExecution) map[string]any {
	return map[string]any{
		"id":         azureChaosStudioExecutionID(execution.ExperimentID, execution.ID),
		"name":       execution.ID,
		"type":       "Microsoft.Chaos/experiments/executions",
		"location":   execution.Location,
		"properties": azureChaosStudioCloneAnyMap(execution.Properties),
		"systemData": azureChaosStudioSystemData(execution.CreatedAt, execution.UpdatedAt),
	}
}

func azureChaosStudioExecutionDetailsFixture(experiment *azureChaosStudioExperiment, execution *azureChaosStudioExecution) map[string]any {
	steps := make([]map[string]any, 0)
	for _, stepValue := range azureChaosStudioSlice(experiment.Properties["steps"]) {
		step := azureChaosStudioObject(stepValue)
		branches := make([]map[string]any, 0)
		for _, branchValue := range azureChaosStudioSlice(step["branches"]) {
			branch := azureChaosStudioObject(branchValue)
			actions := make([]map[string]any, 0)
			for _, actionValue := range azureChaosStudioSlice(branch["actions"]) {
				action := azureChaosStudioObject(actionValue)
				actions = append(actions, map[string]any{
					"name":   azureChaosStudioString(action["name"]),
					"status": execution.Status,
				})
			}
			branches = append(branches, map[string]any{
				"name":    azureChaosStudioString(branch["name"]),
				"status":  execution.Status,
				"actions": actions,
			})
		}
		steps = append(steps, map[string]any{
			"name":     azureChaosStudioString(step["name"]),
			"status":   execution.Status,
			"branches": branches,
		})
	}
	return map[string]any{
		"id":        azureChaosStudioExecutionID(execution.ExperimentID, execution.ID) + "/getExecutionDetails",
		"name":      "getExecutionDetails",
		"type":      "Microsoft.Chaos/experiments/executions/getExecutionDetails",
		"status":    execution.Status,
		"steps":     steps,
		"startedOn": execution.CreatedAt.Format(time.RFC3339),
	}
}

func azureChaosStudioOperationStatusFixture(status *azureChaosStudioOperationStatus) map[string]any {
	out := map[string]any{
		"id":              fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Chaos/locations/%s/operationStatuses/%s", status.SubscriptionID, status.Location, status.ID),
		"name":            status.ID,
		"type":            "Microsoft.Chaos/locations/operationStatuses",
		"status":          status.Status,
		"startTime":       status.StartTime.Format(time.RFC3339),
		"endTime":         status.EndTime.Format(time.RFC3339),
		"percentComplete": status.PercentComplete,
		"resourceId":      status.ResourceID,
	}
	if len(status.Error) > 0 {
		out["error"] = azureChaosStudioCloneAnyMap(status.Error)
	}
	return out
}

func azureChaosStudioNormalizeTargetProperties(resourceURI, name string, raw any, now time.Time) map[string]any {
	properties := azureChaosStudioObject(raw)
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureChaosStudioString(properties["provisioningState"])) == "" {
		properties["provisioningState"] = "Succeeded"
	}
	if strings.TrimSpace(azureChaosStudioString(properties["status"])) == "" {
		properties["status"] = "Enabled"
	}
	if strings.TrimSpace(azureChaosStudioString(properties["targetType"])) == "" {
		properties["targetType"] = "Microsoft-VirtualMachine"
	}
	properties["resourceId"] = resourceURI
	properties["lastModifiedAt"] = now.Format(time.RFC3339)
	properties["displayName"] = name
	return properties
}

func azureChaosStudioNormalizeCapabilityProperties(resourceURI, targetName, name string, raw any, now time.Time) map[string]any {
	properties := azureChaosStudioObject(raw)
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureChaosStudioString(properties["provisioningState"])) == "" {
		properties["provisioningState"] = "Succeeded"
	}
	if strings.TrimSpace(azureChaosStudioString(properties["status"])) == "" {
		properties["status"] = "Enabled"
	}
	if strings.TrimSpace(azureChaosStudioString(properties["urn"])) == "" {
		properties["urn"] = "urn:csci:stackyard:" + strings.ToLower(targetName) + ":" + strings.ToLower(name)
	}
	properties["resourceId"] = azureChaosStudioCapabilityID(resourceURI, targetName, name)
	properties["lastModifiedAt"] = now.Format(time.RFC3339)
	return properties
}

func azureChaosStudioNormalizeExperimentProperties(raw map[string]any, now time.Time) map[string]any {
	properties := azureChaosStudioCloneAnyMap(raw)
	if properties == nil {
		properties = map[string]any{}
	}
	if strings.TrimSpace(azureChaosStudioString(properties["provisioningState"])) == "" {
		properties["provisioningState"] = "Succeeded"
	}
	if strings.TrimSpace(azureChaosStudioString(properties["status"])) == "" {
		properties["status"] = "Ready"
	}
	properties["lastModifiedAt"] = now.Format(time.RFC3339)
	return properties
}

func azureChaosStudioNotFound(path, kind, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "NotFound",
			"message": fmt.Sprintf("%s %q was not found for %s", kind, name, path),
		},
	}
}

func azureChaosStudioReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		respondAzureInvalidRequest(w, path, "request body is required")
		return body, nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return body, nil, false
	}
	return body, payload, true
}

func azureChaosStudioReadOptionalRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
	return body, payload, true
}

func azureChaosStudioString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func azureChaosStudioStringMap(value any) map[string]string {
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil
	}
	out := make(map[string]string, len(object))
	for key, raw := range object {
		if str, ok := raw.(string); ok {
			out[key] = str
		}
	}
	return out
}

func azureChaosStudioAnyMapFromStringMap(in map[string]string) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureChaosStudioObject(value any) map[string]any {
	if object, ok := value.(map[string]any); ok && object != nil {
		return object
	}
	return nil
}

func azureChaosStudioSlice(value any) []any {
	if values, ok := value.([]any); ok && values != nil {
		return values
	}
	return nil
}

func azureChaosStudioCloneAnyMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureChaosStudioCloneAnyMap(typed)
		case []any:
			cloned := make([]any, len(typed))
			for i, item := range typed {
				if object, ok := item.(map[string]any); ok {
					cloned[i] = azureChaosStudioCloneAnyMap(object)
				} else {
					cloned[i] = item
				}
			}
			out[key] = cloned
		default:
			out[key] = value
		}
	}
	return out
}

func azureChaosStudioMergeObjects(base, patch map[string]any) map[string]any {
	out := azureChaosStudioCloneAnyMap(base)
	if out == nil {
		out = map[string]any{}
	}
	for key, value := range patch {
		if existing, ok := out[key].(map[string]any); ok {
			if replacement, ok := value.(map[string]any); ok {
				out[key] = azureChaosStudioMergeObjects(existing, replacement)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func azureChaosStudioSortByID(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureChaosStudioString(items[i]["id"])) < strings.ToLower(azureChaosStudioString(items[j]["id"]))
	})
}

func azureChaosStudioValidTargetName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func azureChaosStudioValidCapabilityName(name string) bool {
	return azureChaosStudioValidTargetName(name) && strings.Contains(name, "-")
}

func azureChaosStudioValidExperimentName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	return !strings.ContainsAny(name, "<>%&:?#/\\")
}

func azureChaosStudioTargetKey(resourceURI, name string) string {
	return strings.ToLower(azureChaosStudioTargetID(resourceURI, name))
}

func azureChaosStudioCapabilityKey(resourceURI, targetName, name string) string {
	return strings.ToLower(azureChaosStudioCapabilityID(resourceURI, targetName, name))
}

func azureChaosStudioExperimentKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureChaosStudioExperimentID(subscriptionID, resourceGroup, name))
}

func azureChaosStudioExecutionKey(experimentID, executionID string) string {
	return strings.ToLower(azureChaosStudioExecutionID(experimentID, executionID))
}

func azureChaosStudioOperationStatusKey(subscriptionID, location, operationID string) string {
	return strings.ToLower(fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Chaos/locations/%s/operationStatuses/%s", subscriptionID, location, operationID))
}

func azureChaosStudioTargetID(resourceURI, name string) string {
	return strings.TrimRight(resourceURI, "/") + "/providers/Microsoft.Chaos/targets/" + name
}

func azureChaosStudioCapabilityID(resourceURI, targetName, name string) string {
	return azureChaosStudioTargetID(resourceURI, targetName) + "/capabilities/" + name
}

func azureChaosStudioExperimentID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Chaos/experiments/%s", subscriptionID, resourceGroup, name)
}

func azureChaosStudioExecutionID(experimentID, executionID string) string {
	return strings.TrimRight(experimentID, "/") + "/executions/" + executionID
}

func azureChaosStudioETag(seed string, now time.Time) string {
	sum := sha1.Sum([]byte(seed + now.UTC().Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureChaosStudioGUIDLike(seed string) string {
	sum := sha1.Sum([]byte(seed))
	hexValue := hex.EncodeToString(sum[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}

func azureChaosStudioSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}
