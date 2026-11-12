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

const azureCDNAPIVersion = "2025-04-15"

type azureCDNProfile struct {
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

type azureCDNEndpoint struct {
	SubscriptionID string
	ResourceGroup  string
	ProfileName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureCDNChildResource struct {
	SubscriptionID string
	ResourceGroup  string
	ProfileName    string
	EndpointName   string
	Collection     string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

var azureCDNChildCollections = map[string]string{
	"origins":       "origins",
	"origingroups":  "originGroups",
	"customdomains": "customDomains",
}

func (s *Server) handleAzureCDNRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureCDNIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureCDNAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Cdn") {
		switch strings.ToLower(strings.TrimSpace(segments[2])) {
		case "operations":
			if r.Method == http.MethodGet {
				respondJSON(w, http.StatusOK, map[string]any{"value": azureCDNOperationsFixture()})
				return true
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		case "checknameavailability":
			if r.Method == http.MethodPost {
				s.handleAzureCDNCheckNameAvailability(w, r, path, "")
				return true
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		case "edgenodes":
			if r.Method == http.MethodGet {
				respondJSON(w, http.StatusOK, map[string]any{"value": azureCDNEdgeNodesFixture()})
				return true
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
	}

	subscriptionID, resourceGroup, tail, ok := azureCDNRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability":
		if resourceGroup != "" || r.Method != http.MethodPost || len(tail) != 1 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCDNCheckNameAvailability(w, r, path, subscriptionID)
		return true
	case "checkresourceusage":
		if resourceGroup != "" || r.Method != http.MethodPost || len(tail) != 1 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCDNSubscriptionResourceUsage(w, subscriptionID)
		return true
	case "validateprobe":
		if resourceGroup != "" || r.Method != http.MethodPost || len(tail) != 1 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureCDNValidateProbe(w, r, path, subscriptionID)
		return true
	case "profiles":
		s.handleAzureCDNProfilesRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		return false
	}
}

func azureCDNIsPath(segments []string) bool {
	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Cdn") {
		switch strings.ToLower(strings.TrimSpace(segments[2])) {
		case "operations", "checknameavailability", "edgenodes":
			return true
		}
	}
	_, _, tail, ok := azureCDNRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability", "checkresourceusage", "validateprobe", "profiles":
		return true
	default:
		return false
	}
}

func azureCDNAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureCDNRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Cdn") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Cdn") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureCDNCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	_ = subscriptionID

	name := strings.TrimSpace(azureCDNString(payload["name"]))
	resourceType := strings.TrimSpace(azureCDNString(payload["type"]))
	if name == "" || resourceType == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	if !azureCDNAllowedNameAvailabilityType(resourceType) {
		respondAzureInvalidRequest(w, path, "type must be Microsoft.Cdn/Profiles/Endpoints or Microsoft.Cdn/Profiles/AfdEndpoints")
		return
	}
	if !azureCDNValidEndpointName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "endpoint name must be 3-50 characters of letters, digits, or hyphens",
		})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, endpoint := range s.azureCDNEndpoints {
		if strings.EqualFold(endpoint.Name, name) {
			respondJSON(w, http.StatusOK, map[string]any{
				"nameAvailable": false,
				"reason":        "Name is already in use",
				"message":       "Name not available",
			})
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *Server) handleAzureCDNSubscriptionResourceUsage(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	count := 0
	for _, profile := range s.azureCDNProfiles {
		if strings.EqualFold(profile.SubscriptionID, subscriptionID) {
			count++
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []map[string]any{{
			"resourceType": "profile",
			"unit":         "count",
			"currentValue": count,
			"limit":        200,
		}},
	})
}

func (s *Server) handleAzureCDNValidateProbe(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	_ = subscriptionID

	probeURL := strings.TrimSpace(azureCDNString(payload["probeURL"]))
	if probeURL == "" {
		respondAzureInvalidRequest(w, path, "probeURL is required")
		return
	}
	isValid := strings.HasPrefix(strings.ToLower(probeURL), "http://") || strings.HasPrefix(strings.ToLower(probeURL), "https://")
	errorCode := "None"
	message := any(nil)
	if !isValid {
		errorCode = "InvalidProbeUrl"
		message = "probeURL must start with http:// or https://"
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"isValid":   isValid,
		"errorCode": errorCode,
		"message":   message,
	})
}

func (s *Server) handleAzureCDNProfilesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 {
		if r.Method == http.MethodGet {
			if resourceGroup == "" {
				s.handleAzureCDNListProfilesBySubscription(w, subscriptionID)
				return
			}
			s.handleAzureCDNListProfilesByResourceGroup(w, subscriptionID, resourceGroup)
			return
		}
		respondAzureImplemented(w, path)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	profileName := strings.TrimSpace(tail[1])
	if !azureCDNValidProfileName(profileName) {
		respondAzureInvalidRequest(w, path, "profile name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCDNGetProfile(w, path, subscriptionID, resourceGroup, profileName, false)
		case http.MethodHead:
			s.handleAzureCDNGetProfile(w, path, subscriptionID, resourceGroup, profileName, true)
		case http.MethodPut:
			s.handleAzureCDNCreateOrUpdateProfile(w, r, path, subscriptionID, resourceGroup, profileName)
		case http.MethodPatch:
			s.handleAzureCDNPatchProfile(w, r, path, subscriptionID, resourceGroup, profileName)
		case http.MethodDelete:
			s.handleAzureCDNDeleteProfile(w, subscriptionID, resourceGroup, profileName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "checkresourceusage":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNProfileUsage(w, path, subscriptionID, resourceGroup, profileName)
			return
		}
	case "getsupportedoptimizationtypes":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNSupportedOptimizationTypes(w, path, subscriptionID, resourceGroup, profileName)
			return
		}
	case "generatessouri":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNGenerateSSO(w, path, subscriptionID, resourceGroup, profileName)
			return
		}
	case "canmigrate":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNProfileMigrationAction(w, path, subscriptionID, resourceGroup, profileName, "CanMigrate")
			return
		}
	case "migrate":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNProfileMigrationAction(w, path, subscriptionID, resourceGroup, profileName, "Migrated")
			return
		}
	case "migrationabort":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNProfileMigrationAction(w, path, subscriptionID, resourceGroup, profileName, "Aborted")
			return
		}
	case "migrationcommit":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNProfileMigrationAction(w, path, subscriptionID, resourceGroup, profileName, "Committed")
			return
		}
	case "endpoints":
		s.handleAzureCDNEndpointsRoute(w, r, path, subscriptionID, resourceGroup, profileName, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCDNEndpointsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName string, tail []string) {
	if !s.azureCDNProfileExists(subscriptionID, resourceGroup, profileName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCDNListEndpoints(w, subscriptionID, resourceGroup, profileName)
		return
	}

	endpointName := strings.TrimSpace(tail[1])
	if !azureCDNValidEndpointName(endpointName) {
		respondAzureInvalidRequest(w, path, "endpoint name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCDNGetEndpoint(w, path, subscriptionID, resourceGroup, profileName, endpointName, false)
		case http.MethodHead:
			s.handleAzureCDNGetEndpoint(w, path, subscriptionID, resourceGroup, profileName, endpointName, true)
		case http.MethodPut:
			s.handleAzureCDNCreateOrUpdateEndpoint(w, r, path, subscriptionID, resourceGroup, profileName, endpointName)
		case http.MethodPatch:
			s.handleAzureCDNPatchEndpoint(w, r, path, subscriptionID, resourceGroup, profileName, endpointName)
		case http.MethodDelete:
			s.handleAzureCDNDeleteEndpoint(w, subscriptionID, resourceGroup, profileName, endpointName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "start":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNEndpointAction(w, path, subscriptionID, resourceGroup, profileName, endpointName, "Running", map[string]any{"message": "endpoint started"})
			return
		}
	case "stop":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNEndpointAction(w, path, subscriptionID, resourceGroup, profileName, endpointName, "Stopped", map[string]any{"message": "endpoint stopped"})
			return
		}
	case "load":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNEndpointPassiveAction(w, r, path, subscriptionID, resourceGroup, profileName, endpointName, "load")
			return
		}
	case "purge":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNEndpointPassiveAction(w, r, path, subscriptionID, resourceGroup, profileName, endpointName, "purge")
			return
		}
	case "checkresourceusage":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNEndpointUsage(w, path, subscriptionID, resourceGroup, profileName, endpointName)
			return
		}
	case "validatecustomdomain":
		if r.Method == http.MethodPost && len(tail) == 3 {
			s.handleAzureCDNValidateCustomDomain(w, r, path, subscriptionID, resourceGroup, profileName, endpointName)
			return
		}
	default:
		if collection, ok := azureCDNChildCollections[strings.ToLower(strings.TrimSpace(tail[2]))]; ok {
			s.handleAzureCDNChildRoute(w, r, path, subscriptionID, resourceGroup, profileName, endpointName, collection, tail[2:])
			return
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCDNChildRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName, collection string, tail []string) {
	if !s.azureCDNEndpointExists(subscriptionID, resourceGroup, profileName, endpointName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCDNListChildren(w, subscriptionID, resourceGroup, profileName, endpointName, collection)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" || !azureCDNValidChildName(name) {
		respondAzureInvalidRequest(w, path, "child resource name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCDNGetChild(w, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name, false)
		case http.MethodHead:
			s.handleAzureCDNGetChild(w, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name, true)
		case http.MethodPut:
			s.handleAzureCDNCreateOrUpdateChild(w, r, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name)
		case http.MethodPatch:
			s.handleAzureCDNPatchChild(w, r, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name)
		case http.MethodDelete:
			s.handleAzureCDNDeleteChild(w, subscriptionID, resourceGroup, profileName, endpointName, collection, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if collection == "customDomains" && len(tail) == 3 {
		switch strings.ToLower(strings.TrimSpace(tail[2])) {
		case "enablecustomhttps":
			if r.Method == http.MethodPost {
				s.handleAzureCDNCustomDomainHTTPS(w, path, subscriptionID, resourceGroup, profileName, endpointName, name, true)
				return
			}
		case "disablecustomhttps":
			if r.Method == http.MethodPost {
				s.handleAzureCDNCustomDomainHTTPS(w, path, subscriptionID, resourceGroup, profileName, endpointName, name, false)
				return
			}
		}
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureCDNCreateOrUpdateProfile(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName string) {
	body, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	_ = body
	location := strings.TrimSpace(azureCDNString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	profileID := azureCDNProfileID(subscriptionID, resourceGroup, profileName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	profile, exists := s.azureCDNProfiles[profileID]
	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	} else {
		profile = &azureCDNProfile{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           profileName,
			CreatedAt:      now,
		}
	}
	profile.Location = location
	profile.Tags = azureCDNStringMap(payload["tags"])
	profile.Properties = azureCDNNormalizeProfileProperties(profileName, payload["properties"], profile.CreatedAt, now)
	profile.UpdatedAt = now
	profile.ETag = azureCDNETag(profileID, now)
	s.azureCDNProfiles[profileID] = profile

	respondJSON(w, status, azureCDNProfileFixture(profile))
}

func (s *Server) handleAzureCDNPatchProfile(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	profileID := azureCDNProfileID(subscriptionID, resourceGroup, profileName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	profile := s.azureCDNProfiles[profileID]
	if profile == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	now := time.Now().UTC()
	if rawTags, ok := payload["tags"]; ok {
		profile.Tags = azureCDNStringMap(rawTags)
	}
	if rawProps, ok := payload["properties"].(map[string]any); ok {
		profile.Properties = azureCDNMergeObjects(profile.Properties, rawProps)
	}
	profile.UpdatedAt = now
	profile.ETag = azureCDNETag(profileID, now)
	respondJSON(w, http.StatusOK, azureCDNProfileFixture(profile))
}

func (s *Server) handleAzureCDNGetProfile(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	profile := s.azureCDNProfiles[azureCDNProfileID(subscriptionID, resourceGroup, profileName)]
	if profile == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCDNProfileFixture(profile))
}

func (s *Server) handleAzureCDNDeleteProfile(w http.ResponseWriter, subscriptionID, resourceGroup, profileName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	profileID := azureCDNProfileID(subscriptionID, resourceGroup, profileName)
	if _, ok := s.azureCDNProfiles[profileID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureCDNProfiles, profileID)
	for key, endpoint := range s.azureCDNEndpoints {
		if endpoint.SubscriptionID == subscriptionID && strings.EqualFold(endpoint.ResourceGroup, resourceGroup) && strings.EqualFold(endpoint.ProfileName, profileName) {
			delete(s.azureCDNEndpoints, key)
		}
	}
	for key, child := range s.azureCDNChildren {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) && strings.EqualFold(child.ProfileName, profileName) {
			delete(s.azureCDNChildren, key)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureCDNListProfilesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	value := make([]map[string]any, 0)
	for _, profile := range s.azureCDNProfiles {
		if profile.SubscriptionID == subscriptionID && strings.EqualFold(profile.ResourceGroup, resourceGroup) {
			value = append(value, azureCDNProfileFixture(profile))
		}
	}
	sort.Slice(value, func(i, j int) bool { return azureCDNString(value[i]["name"]) < azureCDNString(value[j]["name"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCDNListProfilesBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	value := make([]map[string]any, 0)
	for _, profile := range s.azureCDNProfiles {
		if profile.SubscriptionID == subscriptionID {
			value = append(value, azureCDNProfileFixture(profile))
		}
	}
	sort.Slice(value, func(i, j int) bool { return azureCDNString(value[i]["id"]) < azureCDNString(value[j]["id"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCDNProfileUsage(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName string) {
	if !s.azureCDNProfileExists(subscriptionID, resourceGroup, profileName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	count := 0
	for _, endpoint := range s.azureCDNEndpoints {
		if endpoint.SubscriptionID == subscriptionID && strings.EqualFold(endpoint.ResourceGroup, resourceGroup) && strings.EqualFold(endpoint.ProfileName, profileName) {
			count++
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []map[string]any{{
			"resourceType": "endpoint",
			"unit":         "count",
			"currentValue": count,
			"limit":        25,
		}},
	})
}

func (s *Server) handleAzureCDNSupportedOptimizationTypes(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName string) {
	if !s.azureCDNProfileExists(subscriptionID, resourceGroup, profileName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"supportedOptimizationTypes": []string{
			"GeneralWebDelivery",
			"DynamicSiteAcceleration",
			"LargeFileDownload",
		},
	})
}

func (s *Server) handleAzureCDNGenerateSSO(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName string) {
	if !s.azureCDNProfileExists(subscriptionID, resourceGroup, profileName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"ssoUriValue": fmt.Sprintf("https://stackyard.local/cdn-sso/%s/%s", subscriptionID, profileName),
	})
}

func (s *Server) handleAzureCDNProfileMigrationAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName, state string) {
	profileID := azureCDNProfileID(subscriptionID, resourceGroup, profileName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	profile := s.azureCDNProfiles[profileID]
	if profile == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, profileName))
		return
	}
	if profile.Properties == nil {
		profile.Properties = map[string]any{}
	}
	profile.Properties["migrationState"] = state
	profile.UpdatedAt = time.Now().UTC()
	profile.ETag = azureCDNETag(profileID, profile.UpdatedAt)
	switch state {
	case "CanMigrate":
		respondJSON(w, http.StatusOK, map[string]any{
			"canMigrate":       true,
			"migrationMessage": "profile can be migrated",
		})
	default:
		respondJSON(w, http.StatusAccepted, map[string]any{
			"status": state,
		})
	}
}

func (s *Server) handleAzureCDNCreateOrUpdateEndpoint(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureCDNString(payload["location"]))
	if location == "" {
		s.providerStorageMu.Lock()
		if profile := s.azureCDNProfiles[azureCDNProfileID(subscriptionID, resourceGroup, profileName)]; profile != nil {
			location = profile.Location
		}
		s.providerStorageMu.Unlock()
	}
	if location == "" {
		location = "global"
	}

	now := time.Now().UTC()
	id := azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	endpoint, exists := s.azureCDNEndpoints[id]
	status := http.StatusCreated
	if exists {
		status = http.StatusOK
	} else {
		endpoint = &azureCDNEndpoint{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ProfileName:    profileName,
			Name:           endpointName,
			CreatedAt:      now,
		}
	}
	endpoint.Location = location
	endpoint.Tags = azureCDNStringMap(payload["tags"])
	endpoint.Properties = azureCDNNormalizeEndpointProperties(endpointName, payload["properties"], endpoint.CreatedAt, now)
	endpoint.UpdatedAt = now
	endpoint.ETag = azureCDNETag(id, now)
	s.azureCDNEndpoints[id] = endpoint

	if props, ok := payload["properties"].(map[string]any); ok {
		for _, origin := range azureCDNInlineNamedObjects(props["origins"]) {
			originName := azureCDNString(origin["name"])
			s.azureCDNUpsertChildLocked(subscriptionID, resourceGroup, profileName, endpointName, "origins", originName, location, nil, azureCDNNormalizeChildProperties("origins", originName, origin["properties"], now, now), now, true)
		}
		for _, originGroup := range azureCDNInlineNamedObjects(props["originGroups"]) {
			groupName := azureCDNString(originGroup["name"])
			s.azureCDNUpsertChildLocked(subscriptionID, resourceGroup, profileName, endpointName, "originGroups", groupName, location, nil, azureCDNNormalizeChildProperties("originGroups", groupName, originGroup["properties"], now, now), now, true)
		}
	}

	respondJSON(w, status, azureCDNEndpointFixture(endpoint))
}

func (s *Server) handleAzureCDNPatchEndpoint(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	id := azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	endpoint := s.azureCDNEndpoints[id]
	if endpoint == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	if rawTags, ok := payload["tags"]; ok {
		endpoint.Tags = azureCDNStringMap(rawTags)
	}
	if rawProps, ok := payload["properties"].(map[string]any); ok {
		endpoint.Properties = azureCDNMergeObjects(endpoint.Properties, rawProps)
	}
	endpoint.UpdatedAt = time.Now().UTC()
	endpoint.ETag = azureCDNETag(id, endpoint.UpdatedAt)
	respondJSON(w, http.StatusOK, azureCDNEndpointFixture(endpoint))
}

func (s *Server) handleAzureCDNGetEndpoint(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName, endpointName string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	endpoint := s.azureCDNEndpoints[azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)]
	if endpoint == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCDNEndpointFixture(endpoint))
}

func (s *Server) handleAzureCDNDeleteEndpoint(w http.ResponseWriter, subscriptionID, resourceGroup, profileName, endpointName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	id := azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)
	if _, ok := s.azureCDNEndpoints[id]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureCDNEndpoints, id)
	for key, child := range s.azureCDNChildren {
		if child.SubscriptionID == subscriptionID &&
			strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.ProfileName, profileName) &&
			strings.EqualFold(child.EndpointName, endpointName) {
			delete(s.azureCDNChildren, key)
		}
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureCDNListEndpoints(w http.ResponseWriter, subscriptionID, resourceGroup, profileName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	value := make([]map[string]any, 0)
	for _, endpoint := range s.azureCDNEndpoints {
		if endpoint.SubscriptionID == subscriptionID &&
			strings.EqualFold(endpoint.ResourceGroup, resourceGroup) &&
			strings.EqualFold(endpoint.ProfileName, profileName) {
			value = append(value, azureCDNEndpointFixture(endpoint))
		}
	}
	sort.Slice(value, func(i, j int) bool { return azureCDNString(value[i]["name"]) < azureCDNString(value[j]["name"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCDNEndpointAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName, endpointName, state string, body map[string]any) {
	id := azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	endpoint := s.azureCDNEndpoints[id]
	if endpoint == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	if endpoint.Properties == nil {
		endpoint.Properties = map[string]any{}
	}
	endpoint.Properties["resourceState"] = state
	endpoint.Properties["provisioningState"] = "Succeeded"
	endpoint.UpdatedAt = time.Now().UTC()
	endpoint.ETag = azureCDNETag(id, endpoint.UpdatedAt)
	respondJSON(w, http.StatusAccepted, body)
}

func (s *Server) handleAzureCDNEndpointPassiveAction(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName, action string) {
	if _, _, ok := azureCDNReadRequestBody(w, r, path); !ok {
		return
	}
	if !s.azureCDNEndpointExists(subscriptionID, resourceGroup, profileName, endpointName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	respondJSON(w, http.StatusAccepted, map[string]any{
		"status":  "Accepted",
		"action":  action,
		"message": fmt.Sprintf("endpoint %s request accepted", action),
	})
}

func (s *Server) handleAzureCDNEndpointUsage(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName, endpointName string) {
	if !s.azureCDNEndpointExists(subscriptionID, resourceGroup, profileName, endpointName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	customDomains := 0
	for _, child := range s.azureCDNChildren {
		if child.SubscriptionID == subscriptionID &&
			strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.ProfileName, profileName) &&
			strings.EqualFold(child.EndpointName, endpointName) &&
			child.Collection == "customDomains" {
			customDomains++
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"resourceType": "geoFilter",
				"unit":         "count",
				"currentValue": 0,
				"limit":        25,
			},
			{
				"resourceType": "customDomain",
				"unit":         "count",
				"currentValue": customDomains,
				"limit":        25,
			},
		},
	})
}

func (s *Server) handleAzureCDNValidateCustomDomain(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if !s.azureCDNEndpointExists(subscriptionID, resourceGroup, profileName, endpointName) {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, endpointName))
		return
	}
	hostName := strings.TrimSpace(azureCDNString(payload["hostName"]))
	if hostName == "" {
		respondAzureInvalidRequest(w, path, "hostName is required")
		return
	}
	isValid := strings.Contains(hostName, ".") && !strings.HasPrefix(hostName, ".")
	reason := "Valid"
	message := any(nil)
	if !isValid {
		reason = "Invalid"
		message = "hostName must be a valid DNS name"
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"customDomainValidated": isValid,
		"reason":                reason,
		"message":               message,
	})
}

func (s *Server) handleAzureCDNCreateOrUpdateChild(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := "global"
	s.providerStorageMu.Lock()
	if endpoint := s.azureCDNEndpoints[azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)]; endpoint != nil && strings.TrimSpace(endpoint.Location) != "" {
		location = endpoint.Location
	}
	s.providerStorageMu.Unlock()

	properties := payload["properties"]
	switch collection {
	case "origins":
		if err := azureCDNValidateOriginProperties(properties); err != nil {
			respondAzureInvalidRequest(w, path, err.Error())
			return
		}
	case "originGroups":
		if err := azureCDNValidateOriginGroupProperties(properties); err != nil {
			respondAzureInvalidRequest(w, path, err.Error())
			return
		}
	case "customDomains":
		if err := azureCDNValidateCustomDomainProperties(properties); err != nil {
			respondAzureInvalidRequest(w, path, err.Error())
			return
		}
	}

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	id := azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, collection, name)
	_, exists := s.azureCDNChildren[id]
	s.azureCDNUpsertChildLocked(subscriptionID, resourceGroup, profileName, endpointName, collection, name, location, azureCDNStringMap(payload["tags"]), azureCDNNormalizeChildProperties(collection, name, properties, now, now), now, !exists)
	if exists {
		respondJSON(w, http.StatusOK, azureCDNChildFixture(s.azureCDNChildren[id]))
		return
	}
	respondJSON(w, http.StatusCreated, azureCDNChildFixture(s.azureCDNChildren[id]))
}

func (s *Server) handleAzureCDNPatchChild(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name string) {
	_, payload, ok := azureCDNReadRequestBody(w, r, path)
	if !ok {
		return
	}
	id := azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, collection, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child := s.azureCDNChildren[id]
	if child == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, name))
		return
	}
	if rawTags, ok := payload["tags"]; ok {
		child.Tags = azureCDNStringMap(rawTags)
	}
	if rawProps, ok := payload["properties"].(map[string]any); ok {
		child.Properties = azureCDNMergeObjects(child.Properties, rawProps)
	}
	child.UpdatedAt = time.Now().UTC()
	child.ETag = azureCDNETag(id, child.UpdatedAt)
	respondJSON(w, http.StatusOK, azureCDNChildFixture(child))
}

func (s *Server) handleAzureCDNGetChild(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName, endpointName, collection, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	child := s.azureCDNChildren[azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, collection, name)]
	if child == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureCDNChildFixture(child))
}

func (s *Server) handleAzureCDNDeleteChild(w http.ResponseWriter, subscriptionID, resourceGroup, profileName, endpointName, collection, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	id := azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, collection, name)
	if _, ok := s.azureCDNChildren[id]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureCDNChildren, id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCDNListChildren(w http.ResponseWriter, subscriptionID, resourceGroup, profileName, endpointName, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	value := make([]map[string]any, 0)
	for _, child := range s.azureCDNChildren {
		if child.SubscriptionID == subscriptionID &&
			strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.ProfileName, profileName) &&
			strings.EqualFold(child.EndpointName, endpointName) &&
			child.Collection == collection {
			value = append(value, azureCDNChildFixture(child))
		}
	}
	sort.Slice(value, func(i, j int) bool { return azureCDNString(value[i]["name"]) < azureCDNString(value[j]["name"]) })
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCDNCustomDomainHTTPS(w http.ResponseWriter, path, subscriptionID, resourceGroup, profileName, endpointName, name string, enabled bool) {
	id := azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, "customDomains", name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child := s.azureCDNChildren[id]
	if child == nil {
		respondJSON(w, http.StatusNotFound, azureCDNNotFound(path, name))
		return
	}
	if child.Properties == nil {
		child.Properties = map[string]any{}
	}
	if enabled {
		child.Properties["customHttpsProvisioningState"] = "Enabled"
		child.Properties["customHttpsProvisioningSubstate"] = "CertificateDeployed"
		child.Properties["validationData"] = "stackyard-validation-token"
	} else {
		child.Properties["customHttpsProvisioningState"] = "Disabled"
		child.Properties["customHttpsProvisioningSubstate"] = "None"
	}
	child.UpdatedAt = time.Now().UTC()
	child.ETag = azureCDNETag(id, child.UpdatedAt)
	respondJSON(w, http.StatusAccepted, azureCDNChildFixture(child))
}

func (s *Server) azureCDNProfileExists(subscriptionID, resourceGroup, profileName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureCDNProfiles[azureCDNProfileID(subscriptionID, resourceGroup, profileName)] != nil
}

func (s *Server) azureCDNEndpointExists(subscriptionID, resourceGroup, profileName, endpointName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureCDNEndpoints[azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName)] != nil
}

func (s *Server) azureCDNUpsertChildLocked(subscriptionID, resourceGroup, profileName, endpointName, collection, name, location string, tags map[string]string, properties map[string]any, now time.Time, creating bool) {
	id := azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, collection, name)
	child := s.azureCDNChildren[id]
	if child == nil {
		child = &azureCDNChildResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ProfileName:    profileName,
			EndpointName:   endpointName,
			Collection:     collection,
			Name:           name,
			CreatedAt:      now,
		}
	}
	if tags != nil || creating {
		child.Tags = tags
	}
	child.Location = location
	child.Properties = properties
	child.UpdatedAt = now
	child.ETag = azureCDNETag(id, now)
	s.azureCDNChildren[id] = child
}

func azureCDNOperationsFixture() []map[string]any {
	return []map[string]any{
		{
			"name": "Microsoft.Cdn/operations/read",
			"display": map[string]any{
				"provider":  "Microsoft.Cdn",
				"resource":  "Operations",
				"operation": "List operations",
			},
		},
		{
			"name": "Microsoft.Cdn/checkNameAvailability/action",
			"display": map[string]any{
				"provider":  "Microsoft.Cdn",
				"resource":  "Name Availability",
				"operation": "Check resource name availability",
			},
		},
		{
			"name": "Microsoft.Cdn/profiles/read",
			"display": map[string]any{
				"provider":  "Microsoft.Cdn",
				"resource":  "Profiles",
				"operation": "Read profiles",
			},
		},
		{
			"name": "Microsoft.Cdn/profiles/write",
			"display": map[string]any{
				"provider":  "Microsoft.Cdn",
				"resource":  "Profiles",
				"operation": "Create or update profiles",
			},
		},
		{
			"name": "Microsoft.Cdn/profiles/endpoints/read",
			"display": map[string]any{
				"provider":  "Microsoft.Cdn",
				"resource":  "Endpoints",
				"operation": "Read endpoints",
			},
		},
		{
			"name": "Microsoft.Cdn/profiles/endpoints/write",
			"display": map[string]any{
				"provider":  "Microsoft.Cdn",
				"resource":  "Endpoints",
				"operation": "Create or update endpoints",
			},
		},
	}
}

func azureCDNEdgeNodesFixture() []map[string]any {
	return []map[string]any{
		{
			"id":   "/providers/Microsoft.Cdn/edgenodes/edge-eastus",
			"name": "edge-eastus",
			"type": "Microsoft.Cdn/edgenodes",
			"properties": map[string]any{
				"ipAddressGroups": []map[string]any{
					{
						"deliveryRegion": []string{"North America"},
						"ipv4Addresses": []map[string]any{
							{"baseIpAddress": "192.0.2.10", "prefixLength": 24},
						},
					},
				},
			},
		},
		{
			"id":   "/providers/Microsoft.Cdn/edgenodes/edge-west-europe",
			"name": "edge-west-europe",
			"type": "Microsoft.Cdn/edgenodes",
			"properties": map[string]any{
				"ipAddressGroups": []map[string]any{
					{
						"deliveryRegion": []string{"Europe"},
						"ipv4Addresses": []map[string]any{
							{"baseIpAddress": "192.0.2.20", "prefixLength": 24},
						},
					},
				},
			},
		},
	}
}

func azureCDNProfileFixture(profile *azureCDNProfile) map[string]any {
	return map[string]any{
		"id":         azureCDNProfileID(profile.SubscriptionID, profile.ResourceGroup, profile.Name),
		"name":       profile.Name,
		"type":       "Microsoft.Cdn/profiles",
		"location":   profile.Location,
		"tags":       profile.Tags,
		"etag":       profile.ETag,
		"systemData": azureCDNSystemData(profile.CreatedAt, profile.UpdatedAt),
		"properties": profile.Properties,
	}
}

func azureCDNEndpointFixture(endpoint *azureCDNEndpoint) map[string]any {
	return map[string]any{
		"id":         azureCDNEndpointID(endpoint.SubscriptionID, endpoint.ResourceGroup, endpoint.ProfileName, endpoint.Name),
		"name":       endpoint.Name,
		"type":       "Microsoft.Cdn/profiles/endpoints",
		"location":   endpoint.Location,
		"tags":       endpoint.Tags,
		"etag":       endpoint.ETag,
		"systemData": azureCDNSystemData(endpoint.CreatedAt, endpoint.UpdatedAt),
		"properties": endpoint.Properties,
	}
}

func azureCDNChildFixture(child *azureCDNChildResource) map[string]any {
	return map[string]any{
		"id":         azureCDNChildID(child.SubscriptionID, child.ResourceGroup, child.ProfileName, child.EndpointName, child.Collection, child.Name),
		"name":       child.Name,
		"type":       azureCDNChildType(child.Collection),
		"location":   child.Location,
		"tags":       child.Tags,
		"etag":       child.ETag,
		"systemData": azureCDNSystemData(child.CreatedAt, child.UpdatedAt),
		"properties": child.Properties,
	}
}

func azureCDNNormalizeProfileProperties(name string, raw any, createdAt, now time.Time) map[string]any {
	properties := azureCDNObject(raw)
	out := azureCDNMergeObjects(map[string]any{
		"provisioningState":            "Succeeded",
		"resourceState":                "Active",
		"hostName":                     strings.ToLower(name) + ".azureedge.net",
		"frontDoorId":                  "stackyard-cdn-profile",
		"migrationState":               "NotStarted",
		"originResponseTimeoutSeconds": 60,
	}, properties)
	out["provisioningState"] = "Succeeded"
	out["resourceState"] = "Active"
	out["hostName"] = strings.ToLower(name) + ".azureedge.net"
	_ = createdAt
	_ = now
	return out
}

func azureCDNNormalizeEndpointProperties(name string, raw any, createdAt, now time.Time) map[string]any {
	properties := azureCDNObject(raw)
	out := azureCDNMergeObjects(map[string]any{
		"hostName":          strings.ToLower(name) + ".azureedge.net",
		"originHostHeader":  strings.ToLower(name) + ".origin.stackyard.local",
		"isHttpAllowed":     true,
		"isHttpsAllowed":    true,
		"optimizationType":  "GeneralWebDelivery",
		"provisioningState": "Succeeded",
		"resourceState":     "Running",
	}, properties)
	out["hostName"] = strings.ToLower(name) + ".azureedge.net"
	out["provisioningState"] = "Succeeded"
	if _, ok := out["resourceState"]; !ok {
		out["resourceState"] = "Running"
	}
	_ = createdAt
	_ = now
	return out
}

func azureCDNNormalizeChildProperties(collection, name string, raw any, createdAt, now time.Time) map[string]any {
	properties := azureCDNObject(raw)
	out := azureCDNMergeObjects(map[string]any{
		"provisioningState": "Succeeded",
	}, properties)
	switch collection {
	case "origins":
		if _, ok := out["hostName"]; !ok {
			out["hostName"] = strings.ToLower(name) + ".origin.stackyard.local"
		}
		if _, ok := out["httpPort"]; !ok {
			out["httpPort"] = 80
		}
		if _, ok := out["httpsPort"]; !ok {
			out["httpsPort"] = 443
		}
	case "originGroups":
		if _, ok := out["healthProbeSettings"]; !ok {
			out["healthProbeSettings"] = map[string]any{
				"probePath":              "/healthz",
				"probeRequestType":       "GET",
				"probeProtocol":          "Https",
				"probeIntervalInSeconds": 120,
			}
		}
	case "customDomains":
		if _, ok := out["hostName"]; !ok {
			out["hostName"] = name + ".example.com"
		}
		if _, ok := out["customHttpsProvisioningState"]; !ok {
			out["customHttpsProvisioningState"] = "Disabled"
			out["customHttpsProvisioningSubstate"] = "None"
		}
	}
	_ = createdAt
	_ = now
	return out
}

func azureCDNValidateOriginProperties(raw any) error {
	properties := azureCDNObject(raw)
	hostName := strings.TrimSpace(azureCDNString(properties["hostName"]))
	if hostName == "" {
		return fmt.Errorf("origin hostName is required")
	}
	if httpPort, ok := azureCDNInt(properties["httpPort"]); ok && (httpPort < 1 || httpPort > 65535) {
		return fmt.Errorf("origin httpPort must be between 1 and 65535")
	}
	if httpsPort, ok := azureCDNInt(properties["httpsPort"]); ok && (httpsPort < 1 || httpsPort > 65535) {
		return fmt.Errorf("origin httpsPort must be between 1 and 65535")
	}
	return nil
}

func azureCDNValidateOriginGroupProperties(raw any) error {
	properties := azureCDNObject(raw)
	if lb, ok := properties["loadBalancingSettings"].(map[string]any); ok {
		if sampleSize, ok := azureCDNInt(lb["sampleSize"]); ok && sampleSize < 1 {
			return fmt.Errorf("origin group loadBalancingSettings.sampleSize must be positive")
		}
	}
	return nil
}

func azureCDNValidateCustomDomainProperties(raw any) error {
	properties := azureCDNObject(raw)
	hostName := strings.TrimSpace(azureCDNString(properties["hostName"]))
	if hostName == "" {
		return fmt.Errorf("custom domain hostName is required")
	}
	if !strings.Contains(hostName, ".") {
		return fmt.Errorf("custom domain hostName must be a valid DNS name")
	}
	return nil
}

func azureCDNReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureCDNAllowedNameAvailabilityType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case strings.ToLower("Microsoft.Cdn/Profiles/Endpoints"), strings.ToLower("Microsoft.Cdn/Profiles/AfdEndpoints"):
		return true
	default:
		return false
	}
}

func azureCDNProfileID(subscriptionID, resourceGroup, profileName string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Cdn/profiles/%s", subscriptionID, resourceGroup, profileName)
}

func azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName string) string {
	return fmt.Sprintf("%s/endpoints/%s", azureCDNProfileID(subscriptionID, resourceGroup, profileName), endpointName)
}

func azureCDNChildID(subscriptionID, resourceGroup, profileName, endpointName, collection, name string) string {
	return fmt.Sprintf("%s/%s/%s", azureCDNEndpointID(subscriptionID, resourceGroup, profileName, endpointName), collection, name)
}

func azureCDNChildType(collection string) string {
	switch collection {
	case "origins":
		return "Microsoft.Cdn/profiles/endpoints/origins"
	case "originGroups":
		return "Microsoft.Cdn/profiles/endpoints/originGroups"
	case "customDomains":
		return "Microsoft.Cdn/profiles/endpoints/customDomains"
	default:
		return "Microsoft.Cdn/profiles/endpoints/resources"
	}
}

func azureCDNETag(seed string, now time.Time) string {
	sum := sha1.Sum([]byte(seed + "|" + now.UTC().Format(time.RFC3339Nano)))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureCDNSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureCDNNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("resource %q was not found", name),
		},
		"path": path,
	}
}

func azureCDNValidProfileName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 90 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == '(' || r == ')':
		default:
			return false
		}
	}
	return true
}

func azureCDNValidEndpointName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 3 || len(name) > 50 {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	return true
}

func azureCDNValidChildName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 1 || len(name) > 90 {
		return false
	}
	return !strings.ContainsAny(name, "/?&")
}

func azureCDNString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func azureCDNStringMap(value any) map[string]string {
	out := map[string]string{}
	if value == nil {
		return out
	}
	switch typed := value.(type) {
	case map[string]string:
		for k, v := range typed {
			out[k] = v
		}
	case map[string]any:
		for k, v := range typed {
			out[k] = azureCDNString(v)
		}
	}
	return out
}

func azureCDNObject(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func azureCDNMergeObjects(base, patch map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		out[k] = v
	}
	return out
}

func azureCDNInlineNamedObjects(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := strings.TrimSpace(azureCDNString(object["name"]))
		if name == "" {
			continue
		}
		out = append(out, map[string]any{
			"name":       name,
			"properties": azureCDNObject(object["properties"]),
		})
	}
	return out
}

func azureCDNInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
}
