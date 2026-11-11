package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const azureComputeLimitAPIVersion = "2026-04-30"

var (
	azureComputeLimitSubscriptionPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)
	azureComputeLimitSharedLimitPattern  = regexp.MustCompile(`^[a-zA-Z0-9-]{3,24}$`)
)

type azureComputeLimitFeatureResource struct {
	SubscriptionID string
	Location       string
	Name           string
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureComputeLimitGuestSubscriptionResource struct {
	SubscriptionID    string
	Location          string
	GuestSubscription string
	DisplayName       string
	AdditionalProps   map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type azureComputeLimitSharedLimitResource struct {
	SubscriptionID        string
	Location              string
	Name                  string
	ResourceNameValue     string
	ResourceNameLocalized string
	Limit                 float64
	Unit                  string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (s *Server) handleAzureComputeLimitRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureComputeLimitIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureComputeLimitAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ComputeLimit") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeLimitOperationsFixture()})
		return true
	}

	subscriptionID, location, tail, ok := azureComputeLimitRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}
	if !azureComputeLimitValidUUID(subscriptionID) {
		respondAzureInvalidRequest(w, path, "subscriptionId is invalid")
		return true
	}
	if strings.TrimSpace(location) == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return true
	}

	switch {
	case strings.EqualFold(tail[0], "features"):
		s.handleAzureComputeLimitFeatures(w, r, path, subscriptionID, location, tail)
	case strings.EqualFold(tail[0], "guestSubscriptions"):
		s.handleAzureComputeLimitGuestSubscriptions(w, r, path, subscriptionID, location, tail)
	case strings.EqualFold(tail[0], "sharedLimits"):
		s.handleAzureComputeLimitSharedLimits(w, r, path, subscriptionID, location, tail)
	case strings.EqualFold(tail[0], "vmFamilies"):
		s.handleAzureComputeLimitVMFamilies(w, r, path, subscriptionID, location, tail)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func azureComputeLimitIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ComputeLimit") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureComputeLimitRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(tail[0]) {
	case "features", "guestsubscriptions", "sharedlimits", "vmfamilies":
		return true
	default:
		return false
	}
}

func azureComputeLimitAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureComputeLimitRouteContext(segments []string) (subscriptionID, location string, tail []string, ok bool) {
	if len(segments) < 7 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+3 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") &&
			strings.EqualFold(segments[idx+1], "Microsoft.ComputeLimit") &&
			strings.EqualFold(segments[idx+2], "locations") {
			return subscriptionID, strings.TrimSpace(segments[idx+3]), segments[idx+4:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureComputeLimitFeatures(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeLimitListFeatures(w, subscriptionID, location)
		return
	}

	featureName := strings.TrimSpace(tail[1])
	if featureName == "" {
		respondAzureInvalidRequest(w, path, "featureName is required")
		return
	}

	if len(tail) == 2 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeLimitGetFeature(w, subscriptionID, location, featureName)
		return
	}

	if len(tail) == 3 && (strings.EqualFold(tail[2], "enable") || strings.EqualFold(tail[2], "disable")) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, _, ok := azureComputeLimitReadRequestBody(w, r, path)
		if !ok {
			return
		}
		s.handleAzureComputeLimitToggleFeature(w, subscriptionID, location, featureName, strings.EqualFold(tail[2], "enable"))
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureComputeLimitGuestSubscriptions(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeLimitListGuestSubscriptions(w, subscriptionID, location)
		return
	}

	guestSubscriptionID := strings.TrimSpace(tail[1])
	if !azureComputeLimitValidUUID(guestSubscriptionID) {
		respondAzureInvalidRequest(w, path, "guestSubscriptionId is invalid")
		return
	}
	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureComputeLimitGetGuestSubscription(w, path, subscriptionID, location, guestSubscriptionID)
	case http.MethodPut:
		s.handleAzureComputeLimitPutGuestSubscription(w, r, path, subscriptionID, location, guestSubscriptionID)
	case http.MethodDelete:
		s.handleAzureComputeLimitDeleteGuestSubscription(w, subscriptionID, location, guestSubscriptionID)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureComputeLimitSharedLimits(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureComputeLimitListSharedLimits(w, subscriptionID, location)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureComputeLimitSharedLimitPattern.MatchString(name) {
		respondAzureInvalidRequest(w, path, "shared limit name is invalid")
		return
	}
	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureComputeLimitGetSharedLimit(w, path, subscriptionID, location, name)
	case http.MethodPut:
		s.handleAzureComputeLimitPutSharedLimit(w, r, path, subscriptionID, location, name)
	case http.MethodDelete:
		s.handleAzureComputeLimitDeleteSharedLimit(w, subscriptionID, location, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureComputeLimitVMFamilies(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 1 {
		s.handleAzureComputeLimitListVMFamilies(w, subscriptionID, location, strings.TrimSpace(r.URL.Query().Get("$filter")))
		return
	}
	vmFamilyName := strings.TrimSpace(tail[1])
	if vmFamilyName == "" {
		respondAzureInvalidRequest(w, path, "vmFamilyName is required")
		return
	}
	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}
	respondJSON(w, http.StatusOK, azureComputeLimitVMFamilyFixture(subscriptionID, location, vmFamilyName))
}

func (s *Server) handleAzureComputeLimitListFeatures(w http.ResponseWriter, subscriptionID, location string) {
	features := s.azureComputeLimitFeaturesForLocation(subscriptionID, location)
	values := make([]map[string]any, 0, len(features))
	for _, feature := range features {
		values = append(values, azureComputeLimitFeatureFixture(feature))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureComputeLimitGetFeature(w http.ResponseWriter, subscriptionID, location, featureName string) {
	respondJSON(w, http.StatusOK, azureComputeLimitFeatureFixture(s.azureComputeLimitEnsureFeature(subscriptionID, location, featureName)))
}

func (s *Server) handleAzureComputeLimitToggleFeature(w http.ResponseWriter, subscriptionID, location, featureName string, enabled bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	now := time.Now().UTC()
	key := azureComputeLimitKey(subscriptionID, location, featureName)
	feature := s.azureComputeLimitFeatures[key]
	if feature == nil {
		feature = &azureComputeLimitFeatureResource{
			SubscriptionID: subscriptionID,
			Location:       location,
			Name:           featureName,
			CreatedAt:      now,
		}
	}
	feature.Enabled = enabled
	if feature.CreatedAt.IsZero() {
		feature.CreatedAt = now
	}
	feature.UpdatedAt = now
	s.azureComputeLimitFeatures[key] = feature

	respondJSON(w, http.StatusOK, azureComputeLimitFeatureActionFixture(subscriptionID, location, featureName, enabled, now))
}

func (s *Server) handleAzureComputeLimitListGuestSubscriptions(w http.ResponseWriter, subscriptionID, location string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, resource := range s.azureComputeLimitGuestSubscriptions {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.Location, location) {
			continue
		}
		values = append(values, azureComputeLimitGuestSubscriptionFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureComputeLimitString(values[i]["name"])), strings.ToLower(azureComputeLimitString(values[j]["name"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureComputeLimitGetGuestSubscription(w http.ResponseWriter, path, subscriptionID, location, guestSubscriptionID string) {
	s.providerStorageMu.Lock()
	resource := s.azureComputeLimitGuestSubscriptions[azureComputeLimitKey(subscriptionID, location, guestSubscriptionID)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureComputeLimitRespondNotFound(w, path, "GuestSubscriptionNotFound", "guest subscription not found")
		return
	}
	respondJSON(w, http.StatusOK, azureComputeLimitGuestSubscriptionFixture(resource))
}

func (s *Server) handleAzureComputeLimitPutGuestSubscription(w http.ResponseWriter, r *http.Request, path, subscriptionID, location, guestSubscriptionID string) {
	_, payload, ok := azureComputeLimitReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties, ok := payload["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}

	now := time.Now().UTC()
	key := azureComputeLimitKey(subscriptionID, location, guestSubscriptionID)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureComputeLimitGuestSubscriptions[key]
	status := http.StatusOK
	if resource == nil {
		resource = &azureComputeLimitGuestSubscriptionResource{
			SubscriptionID:    subscriptionID,
			Location:          location,
			GuestSubscription: guestSubscriptionID,
			CreatedAt:         now,
		}
		status = http.StatusCreated
	}
	resource.DisplayName = azureComputeLimitString(properties["displayName"])
	resource.AdditionalProps = azureComputeLimitCloneMap(properties)
	resource.UpdatedAt = now
	s.azureComputeLimitGuestSubscriptions[key] = resource

	respondJSON(w, status, azureComputeLimitGuestSubscriptionFixture(resource))
}

func (s *Server) handleAzureComputeLimitDeleteGuestSubscription(w http.ResponseWriter, subscriptionID, location, guestSubscriptionID string) {
	s.providerStorageMu.Lock()
	delete(s.azureComputeLimitGuestSubscriptions, azureComputeLimitKey(subscriptionID, location, guestSubscriptionID))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureComputeLimitListSharedLimits(w http.ResponseWriter, subscriptionID, location string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, resource := range s.azureComputeLimitSharedLimits {
		if !strings.EqualFold(resource.SubscriptionID, subscriptionID) || !strings.EqualFold(resource.Location, location) {
			continue
		}
		values = append(values, azureComputeLimitSharedLimitFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureComputeLimitString(values[i]["name"])), strings.ToLower(azureComputeLimitString(values[j]["name"]))) < 0
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureComputeLimitGetSharedLimit(w http.ResponseWriter, path, subscriptionID, location, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureComputeLimitSharedLimits[azureComputeLimitKey(subscriptionID, location, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureComputeLimitRespondNotFound(w, path, "SharedLimitNotFound", "shared limit not found")
		return
	}
	respondJSON(w, http.StatusOK, azureComputeLimitSharedLimitFixture(resource))
}

func (s *Server) handleAzureComputeLimitPutSharedLimit(w http.ResponseWriter, r *http.Request, path, subscriptionID, location, name string) {
	_, payload, ok := azureComputeLimitReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties, ok := payload["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}

	resourceName := azureComputeLimitNestedString(properties, "resourceName", "value")
	if strings.TrimSpace(resourceName) == "" {
		respondAzureInvalidRequest(w, path, "properties.resourceName.value is required")
		return
	}
	limit, ok := azureComputeLimitNumber(properties["limit"])
	if !ok || limit <= 0 {
		respondAzureInvalidRequest(w, path, "properties.limit is required and must be greater than 0")
		return
	}
	unit := azureComputeLimitString(properties["unit"])
	if unit == "" {
		unit = "Count"
	}

	now := time.Now().UTC()
	key := azureComputeLimitKey(subscriptionID, location, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureComputeLimitSharedLimits[key]
	status := http.StatusOK
	if resource == nil {
		resource = &azureComputeLimitSharedLimitResource{
			SubscriptionID: subscriptionID,
			Location:       location,
			Name:           name,
			CreatedAt:      now,
		}
		status = http.StatusCreated
	}
	resource.ResourceNameValue = resourceName
	resource.ResourceNameLocalized = azureComputeLimitNestedString(properties, "resourceName", "localizedValue")
	if resource.ResourceNameLocalized == "" {
		resource.ResourceNameLocalized = resourceName
	}
	resource.Limit = limit
	resource.Unit = unit
	resource.UpdatedAt = now
	s.azureComputeLimitSharedLimits[key] = resource

	respondJSON(w, status, azureComputeLimitSharedLimitFixture(resource))
}

func (s *Server) handleAzureComputeLimitDeleteSharedLimit(w http.ResponseWriter, subscriptionID, location, name string) {
	s.providerStorageMu.Lock()
	delete(s.azureComputeLimitSharedLimits, azureComputeLimitKey(subscriptionID, location, name))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureComputeLimitListVMFamilies(w http.ResponseWriter, subscriptionID, location, filter string) {
	values := make([]map[string]any, 0)
	for _, name := range azureComputeLimitDefaultVMFamilies() {
		resource := azureComputeLimitVMFamilyFixture(subscriptionID, location, name)
		if azureComputeLimitMatchesFilter(resource, filter) {
			values = append(values, resource)
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) azureComputeLimitFeaturesForLocation(subscriptionID, location string) []*azureComputeLimitFeatureResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	features := make([]*azureComputeLimitFeatureResource, 0, len(azureComputeLimitDefaultFeatureNames())+len(s.azureComputeLimitFeatures))
	seen := map[string]struct{}{}
	for _, name := range azureComputeLimitDefaultFeatureNames() {
		key := azureComputeLimitKey(subscriptionID, location, name)
		feature := s.azureComputeLimitFeatures[key]
		if feature == nil {
			feature = azureComputeLimitDefaultFeature(subscriptionID, location, name)
		}
		features = append(features, feature)
		seen[strings.ToLower(name)] = struct{}{}
	}
	for _, feature := range s.azureComputeLimitFeatures {
		if !strings.EqualFold(feature.SubscriptionID, subscriptionID) || !strings.EqualFold(feature.Location, location) {
			continue
		}
		if _, ok := seen[strings.ToLower(feature.Name)]; ok {
			continue
		}
		features = append(features, feature)
	}
	sort.Slice(features, func(i, j int) bool {
		return strings.Compare(strings.ToLower(features[i].Name), strings.ToLower(features[j].Name)) < 0
	})
	return features
}

func (s *Server) azureComputeLimitEnsureFeature(subscriptionID, location, name string) *azureComputeLimitFeatureResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureComputeLimitKey(subscriptionID, location, name)
	feature := s.azureComputeLimitFeatures[key]
	if feature != nil {
		return feature
	}
	return azureComputeLimitDefaultFeature(subscriptionID, location, name)
}

func azureComputeLimitOperationsFixture() []map[string]any {
	return []map[string]any{
		azureComputeLimitOperation("Microsoft.ComputeLimit/operations/read", "Operations", "List Compute Limit operations"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/features/read", "Features", "Get Compute Limit features"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/features/enable/action", "Features", "Enable Compute Limit feature"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/features/disable/action", "Features", "Disable Compute Limit feature"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/guestSubscriptions/read", "Guest Subscriptions", "Get guest subscriptions"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/guestSubscriptions/write", "Guest Subscriptions", "Create or update guest subscription"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/guestSubscriptions/delete", "Guest Subscriptions", "Delete guest subscription"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/sharedLimits/read", "Shared Limits", "Get shared limits"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/sharedLimits/write", "Shared Limits", "Create or update shared limit"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/sharedLimits/delete", "Shared Limits", "Delete shared limit"),
		azureComputeLimitOperation("Microsoft.ComputeLimit/locations/vmFamilies/read", "VM Families", "Get VM families"),
	}
}

func azureComputeLimitOperation(name, resource, operation string) map[string]any {
	return map[string]any{
		"name":       name,
		"origin":     "system",
		"actionType": "Internal",
		"display": map[string]any{
			"provider":  "Microsoft Compute Limit",
			"resource":  resource,
			"operation": operation,
		},
		"isDataAction": false,
	}
}

func azureComputeLimitFeatureFixture(resource *azureComputeLimitFeatureResource) map[string]any {
	return map[string]any{
		"id":         fmt.Sprintf("/subscriptions/%s/providers/Microsoft.ComputeLimit/locations/%s/features/%s", resource.SubscriptionID, resource.Location, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ComputeLimit/locations/features",
		"systemData": azureComputeLimitSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": map[string]any{
			"state":             azureComputeLimitFeatureState(resource.Enabled),
			"provisioningState": "Succeeded",
			"description":       fmt.Sprintf("%s feature for %s", resource.Name, resource.Location),
		},
	}
}

func azureComputeLimitFeatureActionFixture(subscriptionID, location, featureName string, enabled bool, now time.Time) map[string]any {
	action := "disable"
	if enabled {
		action = "enable"
	}
	return map[string]any{
		"id":        fmt.Sprintf("/subscriptions/%s/providers/Microsoft.ComputeLimit/locations/%s/features/%s/%sOperations/stackyard", subscriptionID, location, featureName, action),
		"name":      "stackyard",
		"status":    "Succeeded",
		"startTime": now.Format(time.RFC3339),
		"endTime":   now.Format(time.RFC3339),
		"properties": map[string]any{
			"featureName": featureName,
			"state":       azureComputeLimitFeatureState(enabled),
		},
	}
}

func azureComputeLimitGuestSubscriptionFixture(resource *azureComputeLimitGuestSubscriptionResource) map[string]any {
	properties := azureComputeLimitCloneMap(resource.AdditionalProps)
	properties["guestSubscriptionId"] = resource.GuestSubscription
	properties["hostSubscriptionId"] = resource.SubscriptionID
	if strings.TrimSpace(resource.DisplayName) != "" {
		properties["displayName"] = resource.DisplayName
	}
	properties["status"] = "Active"
	properties["provisioningState"] = "Succeeded"

	return map[string]any{
		"id":         fmt.Sprintf("/subscriptions/%s/providers/Microsoft.ComputeLimit/locations/%s/guestSubscriptions/%s", resource.SubscriptionID, resource.Location, resource.GuestSubscription),
		"name":       resource.GuestSubscription,
		"type":       "Microsoft.ComputeLimit/locations/guestSubscriptions",
		"systemData": azureComputeLimitSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": properties,
	}
}

func azureComputeLimitSharedLimitFixture(resource *azureComputeLimitSharedLimitResource) map[string]any {
	return map[string]any{
		"id":         fmt.Sprintf("/subscriptions/%s/providers/Microsoft.ComputeLimit/locations/%s/sharedLimits/%s", resource.SubscriptionID, resource.Location, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ComputeLimit/locations/sharedLimits",
		"systemData": azureComputeLimitSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": map[string]any{
			"resourceName": map[string]any{
				"value":          resource.ResourceNameValue,
				"localizedValue": resource.ResourceNameLocalized,
			},
			"limit":             resource.Limit,
			"unit":              resource.Unit,
			"provisioningState": "Succeeded",
		},
	}
}

func azureComputeLimitVMFamilyFixture(subscriptionID, location, name string) map[string]any {
	category := "GeneralPurpose"
	lowerName := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lowerName, "standardnd"), strings.HasPrefix(lowerName, "standardnc"), strings.HasPrefix(lowerName, "standardnv"):
		category = "AcceleratedCompute"
	case strings.HasPrefix(lowerName, "standardm"), strings.HasPrefix(lowerName, "standarde"):
		category = "MemoryOptimized"
	}
	displayName := name
	if len(displayName) > 0 {
		displayName = strings.ToUpper(displayName[:1]) + displayName[1:]
	}

	return map[string]any{
		"id":   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.ComputeLimit/locations/%s/vmFamilies/%s", subscriptionID, location, name),
		"name": name,
		"type": "Microsoft.ComputeLimit/locations/vmFamilies",
		"properties": map[string]any{
			"category":          category,
			"displayName":       displayName,
			"provisioningState": "Succeeded",
		},
	}
}

func azureComputeLimitDefaultFeatureNames() []string {
	return []string{
		"CrossSubscriptionSharing",
		"GuestSubscriptionEnrollment",
		"SharedLimitManagement",
	}
}

func azureComputeLimitDefaultVMFamilies() []string {
	return []string{
		"standardDSv5Family",
		"standardDdv5Family",
		"standardFsv2Family",
	}
}

func azureComputeLimitDefaultFeature(subscriptionID, location, name string) *azureComputeLimitFeatureResource {
	now := time.Date(2026, time.April, 12, 0, 0, 0, 0, time.UTC)
	return &azureComputeLimitFeatureResource{
		SubscriptionID: subscriptionID,
		Location:       location,
		Name:           name,
		Enabled:        false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func azureComputeLimitFeatureState(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

func azureComputeLimitSystemData(createdAt, updatedAt time.Time) map[string]any {
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azureComputeLimitMatchesFilter(resource map[string]any, filter string) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	if filter == "" {
		return true
	}
	name := strings.ToLower(azureComputeLimitString(resource["name"]))
	properties, _ := resource["properties"].(map[string]any)
	category := strings.ToLower(azureComputeLimitString(properties["category"]))
	displayName := strings.ToLower(azureComputeLimitString(properties["displayName"]))

	return strings.Contains(filter, name) ||
		strings.Contains(name, filter) ||
		strings.Contains(filter, category) ||
		strings.Contains(category, filter) ||
		strings.Contains(filter, displayName) ||
		strings.Contains(displayName, filter)
}

func azureComputeLimitRespondNotFound(w http.ResponseWriter, path, code, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": fmt.Sprintf("%s: %s", message, path),
		},
	})
}

func azureComputeLimitReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
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
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	return body, payload, true
}

func azureComputeLimitValidUUID(value string) bool {
	return azureComputeLimitSubscriptionPattern.MatchString(strings.TrimSpace(value))
}

func azureComputeLimitKey(parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(part)))
	}
	return strings.Join(normalized, "::")
}

func azureComputeLimitString(value any) string {
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func azureComputeLimitNestedString(value any, keys ...string) string {
	current := value
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = object[key]
	}
	return azureComputeLimitString(current)
}

func azureComputeLimitNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func azureComputeLimitCloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}
