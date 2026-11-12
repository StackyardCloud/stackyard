package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

const azureStackHCIAPIVersion = "2024-04-01"

type azureStackHCICluster struct {
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

type azureStackHCIArcSetting struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureStackHCIExtension struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	ArcSettingName string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureStackHCIDeploymentSetting struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureStackHCIUpdate struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureStackHCIUpdateSummary struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	Name           string
	Location       string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureStackHCIUpdateRun struct {
	SubscriptionID string
	ResourceGroup  string
	ClusterName    string
	UpdateName     string
	Name           string
	Location       string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

type azureStackHCIEdgeDevice struct {
	ResourceURI string
	Name        string
	Location    string
	Kind        string
	Tags        map[string]string
	Properties  map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ETag        string
}

func (s *Server) handleAzureStackHCIRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsStackHCIPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureStackHCIAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureStackHCI") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureStackHCIOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	if resourceURI, tail, ok := azureStackHCIResourceURIRouteContext(segments); ok && len(tail) > 0 && strings.EqualFold(tail[0], "edgeDevices") {
		s.handleAzureStackHCIEdgeDevicesRoute(w, r, path, resourceURI, tail)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureStackHCIRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	if strings.EqualFold(tail[0], "clusters") {
		s.handleAzureStackHCIClustersRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	}
	return false
}

func azureIsStackHCIPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureStackHCI") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureStackHCIRouteContext(segments)
	if ok && len(tail) > 0 && strings.EqualFold(tail[0], "clusters") {
		return true
	}
	if resourceURI, tail, ok := azureStackHCIResourceURIRouteContext(segments); ok && resourceURI != "" {
		return len(tail) > 0 && strings.EqualFold(tail[0], "edgeDevices")
	}
	return false
}

func azureStackHCIAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureStackHCIRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AzureStackHCI") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AzureStackHCI") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureStackHCIResourceURIRouteContext(segments []string) (resourceURI string, tail []string, ok bool) {
	last := -1
	for idx := 0; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AzureStackHCI") {
			last = idx
		}
	}
	if last <= 0 || last+2 > len(segments) {
		return "", nil, false
	}
	resourceURI = "/" + strings.Join(segments[:last], "/")
	resourceURI = strings.TrimSpace(resourceURI)
	if resourceURI == "/" || resourceURI == "" {
		return "", nil, false
	}
	return resourceURI, segments[last+2:], true
}

func (s *Server) handleAzureStackHCIClustersRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureStackHCIClusterListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureStackHCIClusterListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
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
			s.handleAzureStackHCIClusterGet(w, path, subscriptionID, resourceGroup, clusterName, false)
		case http.MethodHead:
			s.handleAzureStackHCIClusterGet(w, path, subscriptionID, resourceGroup, clusterName, true)
		case http.MethodPut:
			s.handleAzureStackHCIClusterCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName)
		case http.MethodPatch:
			s.handleAzureStackHCIClusterPatch(w, r, path, subscriptionID, resourceGroup, clusterName)
		case http.MethodDelete:
			s.handleAzureStackHCIClusterDelete(w, path, subscriptionID, resourceGroup, clusterName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "configureremotesupport":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIClusterAction(w, r, path, subscriptionID, resourceGroup, clusterName, "configureRemoteSupport")
			return
		}
	case "createidentity":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIClusterAction(w, r, path, subscriptionID, resourceGroup, clusterName, "createIdentity")
			return
		}
	case "extendsoftwareassurancebenefit":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIClusterAction(w, r, path, subscriptionID, resourceGroup, clusterName, "extendSoftwareAssuranceBenefit")
			return
		}
	case "triggerlogcollection":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIClusterAction(w, r, path, subscriptionID, resourceGroup, clusterName, "triggerLogCollection")
			return
		}
	case "uploadcertificate":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIClusterAction(w, r, path, subscriptionID, resourceGroup, clusterName, "uploadCertificate")
			return
		}
	case "arcsettings":
		s.handleAzureStackHCIArcSettingsRoute(w, r, path, subscriptionID, resourceGroup, clusterName, tail[2:])
		return
	case "deploymentsettings":
		s.handleAzureStackHCIDeploymentSettingsRoute(w, r, path, subscriptionID, resourceGroup, clusterName, tail[2:])
		return
	case "updates":
		s.handleAzureStackHCIUpdatesRoute(w, r, path, subscriptionID, resourceGroup, clusterName, tail[2:])
		return
	case "updatesummaries":
		s.handleAzureStackHCIUpdateSummariesRoute(w, r, path, subscriptionID, resourceGroup, clusterName, tail[2:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureStackHCIClusterListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackHCIEnsureSeeded(subscriptionID, "rg-stackhci")
	items := make([]map[string]any, 0)
	for _, cluster := range s.azureStackHCIClusters {
		if strings.EqualFold(cluster.SubscriptionID, subscriptionID) {
			items = append(items, azureStackHCIClusterFixture(cluster))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIClusterListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackHCIEnsureSeeded(subscriptionID, resourceGroup)
	items := make([]map[string]any, 0)
	for _, cluster := range s.azureStackHCIClusters {
		if strings.EqualFold(cluster.SubscriptionID, subscriptionID) && strings.EqualFold(cluster.ResourceGroup, resourceGroup) {
			items = append(items, azureStackHCIClusterFixture(cluster))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIClusterGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackHCIEnsureSeeded(subscriptionID, resourceGroup)
	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIClusterFixture(cluster))
}

func (s *Server) handleAzureStackHCIClusterCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureStackHCIString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	now := time.Now().UTC()
	key := azureStackHCIClusterKey(subscriptionID, resourceGroup, name)
	cluster, exists := s.azureStackHCIClusters[key]
	if !exists {
		cluster = &azureStackHCICluster{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			CreatedAt:      now,
		}
	}
	cluster.Location = location
	cluster.Identity = azureStackHCICloneAnyMap(azureStackHCIObject(payload["identity"]))
	if len(cluster.Identity) == 0 {
		cluster.Identity = map[string]any{"type": "SystemAssigned"}
	}
	cluster.Tags = azureStackHCIStringMap(payload["tags"])
	cluster.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if cluster.Properties == nil {
		cluster.Properties = map[string]any{}
	}
	cluster.Properties["provisioningState"] = "Succeeded"
	if _, ok := cluster.Properties["status"]; !ok {
		cluster.Properties["status"] = "Connected"
	}
	if _, ok := cluster.Properties["aadApplicationObjectId"]; !ok {
		cluster.Properties["aadApplicationObjectId"] = "11111111-1111-1111-1111-111111111111"
	}
	if _, ok := cluster.Properties["desiredProperties"]; !ok {
		cluster.Properties["desiredProperties"] = map[string]any{"diagnosticLevel": "Basic"}
	}
	cluster.UpdatedAt = now
	cluster.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIClusters[key] = cluster

	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIClusterFixture(cluster))
}

func (s *Server) handleAzureStackHCIClusterPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", name))
		return
	}
	if tags, ok := payload["tags"]; ok {
		cluster.Tags = azureStackHCIStringMap(tags)
	}
	if identity, ok := payload["identity"]; ok {
		cluster.Identity = azureStackHCICloneAnyMap(azureStackHCIObject(identity))
	}
	if props, ok := payload["properties"]; ok {
		cluster.Properties = azureStackHCIMergeAnyMap(cluster.Properties, azureStackHCIObject(props))
	}
	cluster.Properties["provisioningState"] = "Succeeded"
	cluster.UpdatedAt = time.Now().UTC()
	cluster.ETag = azureStackHCIETag(azureStackHCIClusterKey(subscriptionID, resourceGroup, name), cluster.UpdatedAt)
	respondJSON(w, http.StatusOK, azureStackHCIClusterFixture(cluster))
}

func (s *Server) handleAzureStackHCIClusterDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	clusterID := azureStackHCIClusterID(subscriptionID, resourceGroup, name)
	if _, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, name)]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", name))
		return
	}
	delete(s.azureStackHCIClusters, azureStackHCIClusterKey(subscriptionID, resourceGroup, name))
	for key := range s.azureStackHCIArcSettings {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/arcSettings/")) {
			delete(s.azureStackHCIArcSettings, key)
		}
	}
	for key := range s.azureStackHCIExtensions {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/arcSettings/")) {
			delete(s.azureStackHCIExtensions, key)
		}
	}
	for key := range s.azureStackHCIDeploymentSettings {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/deploymentSettings/")) {
			delete(s.azureStackHCIDeploymentSettings, key)
		}
	}
	for key := range s.azureStackHCIUpdates {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/updates/")) {
			delete(s.azureStackHCIUpdates, key)
		}
	}
	for key := range s.azureStackHCIUpdateSummaries {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/updateSummaries/")) {
			delete(s.azureStackHCIUpdateSummaries, key)
		}
	}
	for key := range s.azureStackHCIUpdateRuns {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/updates/")) {
			delete(s.azureStackHCIUpdateRuns, key)
		}
	}
	for key := range s.azureStackHCIEdgeDevices {
		if strings.HasPrefix(strings.ToLower(key), strings.ToLower(clusterID+"/providers/Microsoft.AzureStackHCI/edgeDevices/")) {
			delete(s.azureStackHCIEdgeDevices, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIClusterAction(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name, action string) {
	_, payload, ok := azureStackHCIReadOptionalRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", name))
		return
	}

	now := time.Now().UTC()
	switch action {
	case "configureRemoteSupport":
		cluster.Properties["remoteSupportEnabled"] = true
		cluster.Properties["remoteSupportRegistrationId"] = "support-registration-" + name
	case "createIdentity":
		cluster.Identity = azureStackHCIMergeAnyMap(cluster.Identity, map[string]any{
			"type":        "SystemAssigned",
			"principalId": "principal-" + name,
			"tenantId":    "tenant-" + name,
		})
	case "extendSoftwareAssuranceBenefit":
		cluster.Properties["softwareAssurance"] = map[string]any{
			"eligibility": "Eligible",
			"status":      "Enabled",
		}
	case "triggerLogCollection":
		cluster.Properties["lastLogCollectionAt"] = now.Format(time.RFC3339)
	case "uploadCertificate":
		cluster.Properties["certificateStatus"] = "Uploaded"
		cluster.Properties["lastUploadedCertificate"] = azureStackHCIString(payload["certificateName"])
	}
	cluster.UpdatedAt = now
	cluster.ETag = azureStackHCIETag(azureStackHCIClusterKey(subscriptionID, resourceGroup, name), now)
	respondJSON(w, http.StatusOK, map[string]any{
		"status":     "Succeeded",
		"action":     action,
		"resourceId": azureStackHCIClusterID(subscriptionID, resourceGroup, name),
	})
}

func (s *Server) handleAzureStackHCIArcSettingsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIArcSettingList(w, subscriptionID, resourceGroup, clusterName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "arc setting name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureStackHCIArcSettingGet(w, path, subscriptionID, resourceGroup, clusterName, name, false)
		case http.MethodHead:
			s.handleAzureStackHCIArcSettingGet(w, path, subscriptionID, resourceGroup, clusterName, name, true)
		case http.MethodPut:
			s.handleAzureStackHCIArcSettingCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName, name)
		case http.MethodPatch:
			s.handleAzureStackHCIArcSettingPatch(w, r, path, subscriptionID, resourceGroup, clusterName, name)
		case http.MethodDelete:
			s.handleAzureStackHCIArcSettingDelete(w, path, subscriptionID, resourceGroup, clusterName, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "createidentity":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIArcSettingAction(w, path, subscriptionID, resourceGroup, clusterName, name, "createIdentity")
			return
		}
	case "generatepassword":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIArcSettingAction(w, path, subscriptionID, resourceGroup, clusterName, name, "generatePassword")
			return
		}
	case "initializedisableprocess":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIArcSettingAction(w, path, subscriptionID, resourceGroup, clusterName, name, "initializeDisableProcess")
			return
		}
	case "consentandinstalldefaultextensions":
		if r.Method == http.MethodPost {
			s.handleAzureStackHCIArcSettingAction(w, path, subscriptionID, resourceGroup, clusterName, name, "consentAndInstallDefaultExtensions")
			return
		}
	case "extensions":
		s.handleAzureStackHCIExtensionsRoute(w, r, path, subscriptionID, resourceGroup, clusterName, name, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureStackHCIArcSettingList(w http.ResponseWriter, subscriptionID, resourceGroup, clusterName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackHCIEnsureSeeded(subscriptionID, resourceGroup)
	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIArcSettings {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.ClusterName, clusterName) {
			items = append(items, azureStackHCIArcSettingFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIArcSettingGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIArcSettings[azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ArcSetting", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIArcSettingFixture(item))
}

func (s *Server) handleAzureStackHCIArcSettingCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, clusterName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", clusterName))
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name)
	item, exists := s.azureStackHCIArcSettings[key]
	if !exists {
		item = &azureStackHCIArcSetting{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	item.Location = strings.TrimSpace(azureStackHCIString(payload["location"]))
	if item.Location == "" {
		item.Location = cluster.Location
	}
	item.Tags = azureStackHCIStringMap(payload["tags"])
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["provisioningState"] = "Succeeded"
	if _, ok := item.Properties["connectivityState"]; !ok {
		item.Properties["connectivityState"] = "Connected"
	}
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIArcSettings[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIArcSettingFixture(item))
}

func (s *Server) handleAzureStackHCIArcSettingPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIArcSettings[azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ArcSetting", name))
		return
	}
	if tags, ok := payload["tags"]; ok {
		item.Tags = azureStackHCIStringMap(tags)
	}
	if props, ok := payload["properties"]; ok {
		item.Properties = azureStackHCIMergeAnyMap(item.Properties, azureStackHCIObject(props))
	}
	item.Properties["provisioningState"] = "Succeeded"
	item.UpdatedAt = time.Now().UTC()
	item.ETag = azureStackHCIETag(azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name), item.UpdatedAt)
	respondJSON(w, http.StatusOK, azureStackHCIArcSettingFixture(item))
}

func (s *Server) handleAzureStackHCIArcSettingDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name)
	if _, ok := s.azureStackHCIArcSettings[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ArcSetting", name))
		return
	}
	delete(s.azureStackHCIArcSettings, key)
	prefix := strings.ToLower(azureStackHCIArcSettingID(subscriptionID, resourceGroup, clusterName, name) + "/extensions/")
	for extensionKey := range s.azureStackHCIExtensions {
		if strings.HasPrefix(strings.ToLower(extensionKey), prefix) {
			delete(s.azureStackHCIExtensions, extensionKey)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIArcSettingAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name, action string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIArcSettings[azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ArcSetting", name))
		return
	}
	now := time.Now().UTC()
	switch action {
	case "createIdentity":
		item.Properties["identity"] = map[string]any{"principalId": "arc-principal-" + name, "tenantId": "arc-tenant-" + name}
	case "generatePassword":
		item.Properties["lastGeneratedPasswordAt"] = now.Format(time.RFC3339)
		respondJSON(w, http.StatusOK, map[string]any{"password": "Stackyard-" + name, "expiresOn": now.Add(24 * time.Hour).Format(time.RFC3339)})
		item.UpdatedAt = now
		item.ETag = azureStackHCIETag(azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name), now)
		return
	case "initializeDisableProcess":
		item.Properties["disableProcessState"] = "Initialized"
	case "consentAndInstallDefaultExtensions":
		item.Properties["defaultExtensionsInstalled"] = true
	}
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name), now)
	respondJSON(w, http.StatusOK, map[string]any{"status": "Succeeded", "action": action, "resourceId": azureStackHCIArcSettingID(subscriptionID, resourceGroup, clusterName, name)})
}

func (s *Server) handleAzureStackHCIExtensionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, arcSettingName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIExtensionList(w, subscriptionID, resourceGroup, clusterName, arcSettingName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "extension name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureStackHCIExtensionGet(w, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name, false)
		case http.MethodHead:
			s.handleAzureStackHCIExtensionGet(w, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name, true)
		case http.MethodPut:
			s.handleAzureStackHCIExtensionCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name)
		case http.MethodPatch:
			s.handleAzureStackHCIExtensionPatch(w, r, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name)
		case http.MethodDelete:
			s.handleAzureStackHCIExtensionDelete(w, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 3 && strings.EqualFold(tail[2], "upgrade") && r.Method == http.MethodPost {
		s.handleAzureStackHCIExtensionUpgrade(w, r, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureStackHCIExtensionList(w http.ResponseWriter, subscriptionID, resourceGroup, clusterName, arcSettingName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIExtensions {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.ClusterName, clusterName) &&
			strings.EqualFold(item.ArcSettingName, arcSettingName) {
			items = append(items, azureStackHCIExtensionFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIExtensionGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIExtensions[azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Extension", name))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIExtensionFixture(item))
}

func (s *Server) handleAzureStackHCIExtensionCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	parent, ok := s.azureStackHCIArcSettings[azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, arcSettingName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ArcSetting", arcSettingName))
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name)
	item, exists := s.azureStackHCIExtensions[key]
	if !exists {
		item = &azureStackHCIExtension{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			ArcSettingName: arcSettingName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	item.Location = strings.TrimSpace(azureStackHCIString(payload["location"]))
	if item.Location == "" {
		item.Location = parent.Location
	}
	item.Tags = azureStackHCIStringMap(payload["tags"])
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["provisioningState"] = "Succeeded"
	if _, ok := item.Properties["version"]; !ok {
		item.Properties["version"] = "1.0.0"
	}
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIExtensions[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIExtensionFixture(item))
}

func (s *Server) handleAzureStackHCIExtensionPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIExtensions[azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Extension", name))
		return
	}
	if tags, ok := payload["tags"]; ok {
		item.Tags = azureStackHCIStringMap(tags)
	}
	if props, ok := payload["properties"]; ok {
		item.Properties = azureStackHCIMergeAnyMap(item.Properties, azureStackHCIObject(props))
	}
	item.Properties["provisioningState"] = "Succeeded"
	item.UpdatedAt = time.Now().UTC()
	item.ETag = azureStackHCIETag(azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name), item.UpdatedAt)
	respondJSON(w, http.StatusOK, azureStackHCIExtensionFixture(item))
}

func (s *Server) handleAzureStackHCIExtensionDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name)
	if _, ok := s.azureStackHCIExtensions[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Extension", name))
		return
	}
	delete(s.azureStackHCIExtensions, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIExtensionUpgrade(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, arcSettingName, name string) {
	_, payload, ok := azureStackHCIReadOptionalRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIExtensions[azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Extension", name))
		return
	}
	targetVersion := strings.TrimSpace(azureStackHCIString(payload["targetVersion"]))
	if targetVersion == "" {
		targetVersion = "1.1.0"
	}
	item.Properties["version"] = targetVersion
	item.Properties["lastUpgradeAt"] = time.Now().UTC().Format(time.RFC3339)
	item.UpdatedAt = time.Now().UTC()
	item.ETag = azureStackHCIETag(azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name), item.UpdatedAt)
	respondJSON(w, http.StatusOK, map[string]any{"status": "Succeeded", "targetVersion": targetVersion, "resourceId": azureStackHCIExtensionID(subscriptionID, resourceGroup, clusterName, arcSettingName, name)})
}

func (s *Server) handleAzureStackHCIDeploymentSettingsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIDeploymentSettingList(w, subscriptionID, resourceGroup, clusterName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "deployment setting name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureStackHCIDeploymentSettingGet(w, path, subscriptionID, resourceGroup, clusterName, name)
	case http.MethodPut:
		s.handleAzureStackHCIDeploymentSettingCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName, name)
	case http.MethodDelete:
		s.handleAzureStackHCIDeploymentSettingDelete(w, path, subscriptionID, resourceGroup, clusterName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureStackHCIDeploymentSettingList(w http.ResponseWriter, subscriptionID, resourceGroup, clusterName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIDeploymentSettings {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.ClusterName, clusterName) {
			items = append(items, azureStackHCIDeploymentSettingFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIDeploymentSettingGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIDeploymentSettings[azureStackHCIDeploymentSettingKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "DeploymentSetting", name))
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIDeploymentSettingFixture(item))
}

func (s *Server) handleAzureStackHCIDeploymentSettingCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, clusterName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", clusterName))
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIDeploymentSettingKey(subscriptionID, resourceGroup, clusterName, name)
	item, exists := s.azureStackHCIDeploymentSettings[key]
	if !exists {
		item = &azureStackHCIDeploymentSetting{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	item.Location = strings.TrimSpace(azureStackHCIString(payload["location"]))
	if item.Location == "" {
		item.Location = cluster.Location
	}
	item.Tags = azureStackHCIStringMap(payload["tags"])
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["provisioningState"] = "Succeeded"
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIDeploymentSettings[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIDeploymentSettingFixture(item))
}

func (s *Server) handleAzureStackHCIDeploymentSettingDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIDeploymentSettingKey(subscriptionID, resourceGroup, clusterName, name)
	if _, ok := s.azureStackHCIDeploymentSettings[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "DeploymentSetting", name))
		return
	}
	delete(s.azureStackHCIDeploymentSettings, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIUpdatesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIUpdateList(w, subscriptionID, resourceGroup, clusterName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "update name is required")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureStackHCIUpdateGet(w, path, subscriptionID, resourceGroup, clusterName, name)
		case http.MethodPut:
			s.handleAzureStackHCIUpdateCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName, name)
		case http.MethodDelete:
			s.handleAzureStackHCIUpdateDelete(w, path, subscriptionID, resourceGroup, clusterName, name)
		case http.MethodPost:
			s.handleAzureStackHCIUpdateApply(w, path, subscriptionID, resourceGroup, clusterName, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if strings.EqualFold(tail[2], "updateRuns") {
		s.handleAzureStackHCIUpdateRunsRoute(w, r, path, subscriptionID, resourceGroup, clusterName, name, tail[2:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureStackHCIUpdateList(w http.ResponseWriter, subscriptionID, resourceGroup, clusterName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackHCIEnsureSeeded(subscriptionID, resourceGroup)
	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIUpdates {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.ClusterName, clusterName) {
			items = append(items, azureStackHCIUpdateFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIUpdateGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIUpdates[azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Update", name))
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIUpdateFixture(item))
}

func (s *Server) handleAzureStackHCIUpdateCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, clusterName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", clusterName))
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, name)
	item, exists := s.azureStackHCIUpdates[key]
	if !exists {
		item = &azureStackHCIUpdate{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	item.Location = strings.TrimSpace(azureStackHCIString(payload["location"]))
	if item.Location == "" {
		item.Location = cluster.Location
	}
	item.Tags = azureStackHCIStringMap(payload["tags"])
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["provisioningState"] = "Succeeded"
	if _, ok := item.Properties["state"]; !ok {
		item.Properties["state"] = "Available"
	}
	if _, ok := item.Properties["packageVersion"]; !ok {
		item.Properties["packageVersion"] = "2404.0"
	}
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIUpdates[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIUpdateFixture(item))
}

func (s *Server) handleAzureStackHCIUpdateDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, name)
	if _, ok := s.azureStackHCIUpdates[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Update", name))
		return
	}
	delete(s.azureStackHCIUpdates, key)
	for runKey, run := range s.azureStackHCIUpdateRuns {
		if strings.EqualFold(run.SubscriptionID, subscriptionID) && strings.EqualFold(run.ResourceGroup, resourceGroup) && strings.EqualFold(run.ClusterName, clusterName) && strings.EqualFold(run.UpdateName, name) {
			delete(s.azureStackHCIUpdateRuns, runKey)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIUpdateApply(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	update, ok := s.azureStackHCIUpdates[azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Update", name))
		return
	}
	now := time.Now().UTC()
	update.Properties["state"] = "Installed"
	update.Properties["lastAppliedAt"] = now.Format(time.RFC3339)
	update.UpdatedAt = now
	update.ETag = azureStackHCIETag(azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, name), now)

	summaryName := "default"
	summaryKey := azureStackHCIUpdateSummaryKey(subscriptionID, resourceGroup, clusterName, summaryName)
	summary := &azureStackHCIUpdateSummary{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    clusterName,
		Name:           summaryName,
		Location:       update.Location,
		Properties: map[string]any{
			"status":            "Succeeded",
			"lastUpdatedAt":     now.Format(time.RFC3339),
			"lastAppliedUpdate": name,
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(summaryKey, now),
	}
	s.azureStackHCIUpdateSummaries[summaryKey] = summary

	runName := "run-latest"
	runKey := azureStackHCIUpdateRunKey(subscriptionID, resourceGroup, clusterName, name, runName)
	s.azureStackHCIUpdateRuns[runKey] = &azureStackHCIUpdateRun{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    clusterName,
		UpdateName:     name,
		Name:           runName,
		Location:       update.Location,
		Properties: map[string]any{
			"status":      "Succeeded",
			"startedAt":   now.Format(time.RFC3339),
			"completedAt": now.Add(2 * time.Minute).Format(time.RFC3339),
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(runKey, now),
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "Succeeded", "resourceId": azureStackHCIUpdateID(subscriptionID, resourceGroup, clusterName, name), "runName": runName})
}

func (s *Server) handleAzureStackHCIUpdateSummariesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIUpdateSummaryList(w, subscriptionID, resourceGroup, clusterName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "update summary name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureStackHCIUpdateSummaryGet(w, path, subscriptionID, resourceGroup, clusterName, name)
	case http.MethodPut:
		s.handleAzureStackHCIUpdateSummaryCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName, name)
	case http.MethodDelete:
		s.handleAzureStackHCIUpdateSummaryDelete(w, path, subscriptionID, resourceGroup, clusterName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureStackHCIUpdateSummaryList(w http.ResponseWriter, subscriptionID, resourceGroup, clusterName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackHCIEnsureSeeded(subscriptionID, resourceGroup)
	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIUpdateSummaries {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) && strings.EqualFold(item.ResourceGroup, resourceGroup) && strings.EqualFold(item.ClusterName, clusterName) {
			items = append(items, azureStackHCIUpdateSummaryFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIUpdateSummaryGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIUpdateSummaries[azureStackHCIUpdateSummaryKey(subscriptionID, resourceGroup, clusterName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "UpdateSummary", name))
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIUpdateSummaryFixture(item))
}

func (s *Server) handleAzureStackHCIUpdateSummaryCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	cluster, ok := s.azureStackHCIClusters[azureStackHCIClusterKey(subscriptionID, resourceGroup, clusterName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Cluster", clusterName))
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIUpdateSummaryKey(subscriptionID, resourceGroup, clusterName, name)
	item, exists := s.azureStackHCIUpdateSummaries[key]
	if !exists {
		item = &azureStackHCIUpdateSummary{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	item.Location = cluster.Location
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["status"] = "Succeeded"
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIUpdateSummaries[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIUpdateSummaryFixture(item))
}

func (s *Server) handleAzureStackHCIUpdateSummaryDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIUpdateSummaryKey(subscriptionID, resourceGroup, clusterName, name)
	if _, ok := s.azureStackHCIUpdateSummaries[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "UpdateSummary", name))
		return
	}
	delete(s.azureStackHCIUpdateSummaries, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIUpdateRunsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, updateName string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIUpdateRunList(w, subscriptionID, resourceGroup, clusterName, updateName)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "update run name is required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureStackHCIUpdateRunGet(w, path, subscriptionID, resourceGroup, clusterName, updateName, name)
	case http.MethodPut:
		s.handleAzureStackHCIUpdateRunCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, clusterName, updateName, name)
	case http.MethodDelete:
		s.handleAzureStackHCIUpdateRunDelete(w, path, subscriptionID, resourceGroup, clusterName, updateName, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureStackHCIUpdateRunList(w http.ResponseWriter, subscriptionID, resourceGroup, clusterName, updateName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIUpdateRuns {
		if strings.EqualFold(item.SubscriptionID, subscriptionID) &&
			strings.EqualFold(item.ResourceGroup, resourceGroup) &&
			strings.EqualFold(item.ClusterName, clusterName) &&
			strings.EqualFold(item.UpdateName, updateName) {
			items = append(items, azureStackHCIUpdateRunFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIUpdateRunGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, updateName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIUpdateRuns[azureStackHCIUpdateRunKey(subscriptionID, resourceGroup, clusterName, updateName, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "UpdateRun", name))
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIUpdateRunFixture(item))
}

func (s *Server) handleAzureStackHCIUpdateRunCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, clusterName, updateName, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	update, ok := s.azureStackHCIUpdates[azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, updateName)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "Update", updateName))
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIUpdateRunKey(subscriptionID, resourceGroup, clusterName, updateName, name)
	item, exists := s.azureStackHCIUpdateRuns[key]
	if !exists {
		item = &azureStackHCIUpdateRun{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			ClusterName:    clusterName,
			UpdateName:     updateName,
			Name:           name,
			CreatedAt:      now,
		}
	}
	item.Location = update.Location
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["status"] = "Succeeded"
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIUpdateRuns[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIUpdateRunFixture(item))
}

func (s *Server) handleAzureStackHCIUpdateRunDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, clusterName, updateName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIUpdateRunKey(subscriptionID, resourceGroup, clusterName, updateName, name)
	if _, ok := s.azureStackHCIUpdateRuns[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "UpdateRun", name))
		return
	}
	delete(s.azureStackHCIUpdateRuns, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIEdgeDevicesRoute(w http.ResponseWriter, r *http.Request, path, resourceURI string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		s.handleAzureStackHCIEdgeDeviceList(w, path, resourceURI)
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}
	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "edge device name is required")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureStackHCIEdgeDeviceGet(w, path, resourceURI, name)
		case http.MethodPut:
			s.handleAzureStackHCIEdgeDeviceCreateOrUpdate(w, r, path, resourceURI, name)
		case http.MethodDelete:
			s.handleAzureStackHCIEdgeDeviceDelete(w, path, resourceURI, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 3 && strings.EqualFold(tail[2], "validate") && r.Method == http.MethodPost {
		s.handleAzureStackHCIEdgeDeviceValidate(w, path, resourceURI, name)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureStackHCIEdgeDeviceList(w http.ResponseWriter, path, resourceURI string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if !s.azureStackHCIParentExists(resourceURI) {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ParentResource", resourceURI))
		return
	}
	items := make([]map[string]any, 0)
	for _, item := range s.azureStackHCIEdgeDevices {
		if strings.EqualFold(item.ResourceURI, resourceURI) {
			items = append(items, azureStackHCIEdgeDeviceFixture(item))
		}
	}
	azureStackHCISortByID(items)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackHCIEdgeDeviceGet(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIEdgeDevices[azureStackHCIEdgeDeviceKey(resourceURI, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "EdgeDevice", name))
		return
	}
	respondJSON(w, http.StatusOK, azureStackHCIEdgeDeviceFixture(item))
}

func (s *Server) handleAzureStackHCIEdgeDeviceCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, resourceURI, name string) {
	_, payload, ok := azureStackHCIReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	if !s.azureStackHCIParentExists(resourceURI) {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "ParentResource", resourceURI))
		return
	}
	kind := strings.TrimSpace(azureStackHCIString(payload["kind"]))
	if kind == "" {
		respondAzureInvalidRequest(w, path, "kind is required")
		return
	}
	now := time.Now().UTC()
	key := azureStackHCIEdgeDeviceKey(resourceURI, name)
	item, exists := s.azureStackHCIEdgeDevices[key]
	if !exists {
		item = &azureStackHCIEdgeDevice{
			ResourceURI: resourceURI,
			Name:        name,
			CreatedAt:   now,
		}
	}
	item.Kind = kind
	item.Location = strings.TrimSpace(azureStackHCIString(payload["location"]))
	if item.Location == "" {
		item.Location = "eastus"
	}
	item.Tags = azureStackHCIStringMap(payload["tags"])
	item.Properties = azureStackHCICloneAnyMap(azureStackHCIObject(payload["properties"]))
	if item.Properties == nil {
		item.Properties = map[string]any{}
	}
	item.Properties["provisioningState"] = "Succeeded"
	item.UpdatedAt = now
	item.ETag = azureStackHCIETag(key, now)
	s.azureStackHCIEdgeDevices[key] = item
	status := http.StatusOK
	if !exists {
		status = http.StatusCreated
	}
	respondJSON(w, status, azureStackHCIEdgeDeviceFixture(item))
}

func (s *Server) handleAzureStackHCIEdgeDeviceDelete(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	key := azureStackHCIEdgeDeviceKey(resourceURI, name)
	if _, ok := s.azureStackHCIEdgeDevices[key]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "EdgeDevice", name))
		return
	}
	delete(s.azureStackHCIEdgeDevices, key)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureStackHCIEdgeDeviceValidate(w http.ResponseWriter, path, resourceURI, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	item, ok := s.azureStackHCIEdgeDevices[azureStackHCIEdgeDeviceKey(resourceURI, name)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackHCINotFound(path, "EdgeDevice", name))
		return
	}
	item.Properties["validationState"] = "Succeeded"
	item.Properties["lastValidatedAt"] = time.Now().UTC().Format(time.RFC3339)
	item.UpdatedAt = time.Now().UTC()
	item.ETag = azureStackHCIETag(azureStackHCIEdgeDeviceKey(resourceURI, name), item.UpdatedAt)
	respondJSON(w, http.StatusOK, map[string]any{"status": "Succeeded", "resourceId": azureStackHCIEdgeDeviceID(resourceURI, name)})
}

func (s *Server) azureStackHCIEnsureSeeded(subscriptionID, resourceGroup string) {
	resourceGroup = strings.TrimSpace(resourceGroup)
	if resourceGroup == "" {
		resourceGroup = "rg-stackhci"
	}
	key := azureStackHCIClusterKey(subscriptionID, resourceGroup, "seed-cluster")
	if _, ok := s.azureStackHCIClusters[key]; ok {
		return
	}
	now := time.Now().UTC()
	cluster := &azureStackHCICluster{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           "seed-cluster",
		Location:       "eastus",
		Identity:       map[string]any{"type": "SystemAssigned", "principalId": "principal-seed-cluster"},
		Tags:           map[string]string{"seed": "true"},
		Properties: map[string]any{
			"provisioningState":      "Succeeded",
			"status":                 "Connected",
			"remoteSupportEnabled":   false,
			"desiredProperties":      map[string]any{"diagnosticLevel": "Basic"},
			"softwareAssurance":      map[string]any{"status": "Disabled"},
			"aadApplicationObjectId": "11111111-1111-1111-1111-111111111111",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(key, now),
	}
	s.azureStackHCIClusters[key] = cluster

	arcKey := azureStackHCIArcSettingKey(subscriptionID, resourceGroup, cluster.Name, "default")
	s.azureStackHCIArcSettings[arcKey] = &azureStackHCIArcSetting{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    cluster.Name,
		Name:           "default",
		Location:       cluster.Location,
		Properties: map[string]any{
			"provisioningState": "Succeeded",
			"connectivityState": "Connected",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(arcKey, now),
	}

	extensionKey := azureStackHCIExtensionKey(subscriptionID, resourceGroup, cluster.Name, "default", "telemetry")
	s.azureStackHCIExtensions[extensionKey] = &azureStackHCIExtension{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    cluster.Name,
		ArcSettingName: "default",
		Name:           "telemetry",
		Location:       cluster.Location,
		Properties: map[string]any{
			"provisioningState": "Succeeded",
			"version":           "1.0.0",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(extensionKey, now),
	}

	deploymentKey := azureStackHCIDeploymentSettingKey(subscriptionID, resourceGroup, cluster.Name, "default")
	s.azureStackHCIDeploymentSettings[deploymentKey] = &azureStackHCIDeploymentSetting{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    cluster.Name,
		Name:           "default",
		Location:       cluster.Location,
		Properties: map[string]any{
			"provisioningState": "Succeeded",
			"deploymentMode":    "ValidateOnly",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(deploymentKey, now),
	}

	updateKey := azureStackHCIUpdateKey(subscriptionID, resourceGroup, cluster.Name, "update-1")
	s.azureStackHCIUpdates[updateKey] = &azureStackHCIUpdate{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    cluster.Name,
		Name:           "update-1",
		Location:       cluster.Location,
		Properties: map[string]any{
			"provisioningState": "Succeeded",
			"state":             "Available",
			"packageVersion":    "2404.0",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(updateKey, now),
	}

	summaryKey := azureStackHCIUpdateSummaryKey(subscriptionID, resourceGroup, cluster.Name, "default")
	s.azureStackHCIUpdateSummaries[summaryKey] = &azureStackHCIUpdateSummary{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		ClusterName:    cluster.Name,
		Name:           "default",
		Location:       cluster.Location,
		Properties: map[string]any{
			"status":            "Available",
			"lastAppliedUpdate": "",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureStackHCIETag(summaryKey, now),
	}
}

func (s *Server) azureStackHCIParentExists(resourceURI string) bool {
	_, ok := s.azureStackHCIClusters[strings.ToLower(strings.TrimSpace(resourceURI))]
	return ok
}

func azureStackHCIOperationsFixture() []map[string]any {
	ops := []string{
		"Microsoft.AzureStackHCI/clusters/read",
		"Microsoft.AzureStackHCI/clusters/write",
		"Microsoft.AzureStackHCI/clusters/delete",
		"Microsoft.AzureStackHCI/clusters/configureRemoteSupport/action",
		"Microsoft.AzureStackHCI/clusters/createIdentity/action",
		"Microsoft.AzureStackHCI/clusters/uploadCertificate/action",
		"Microsoft.AzureStackHCI/clusters/arcSettings/read",
		"Microsoft.AzureStackHCI/clusters/arcSettings/write",
		"Microsoft.AzureStackHCI/clusters/arcSettings/extensions/read",
		"Microsoft.AzureStackHCI/clusters/arcSettings/extensions/write",
		"Microsoft.AzureStackHCI/clusters/deploymentSettings/read",
		"Microsoft.AzureStackHCI/clusters/deploymentSettings/write",
		"Microsoft.AzureStackHCI/clusters/updates/read",
		"Microsoft.AzureStackHCI/clusters/updates/write",
		"Microsoft.AzureStackHCI/clusters/updates/updateRuns/read",
		"Microsoft.AzureStackHCI/clusters/updateSummaries/read",
		"Microsoft.AzureStackHCI/edgeDevices/read",
		"Microsoft.AzureStackHCI/edgeDevices/write",
	}
	out := make([]map[string]any, 0, len(ops))
	for _, name := range ops {
		out = append(out, map[string]any{
			"name":         name,
			"isDataAction": false,
			"display": map[string]any{
				"provider":  "Microsoft Azure Stack HCI",
				"resource":  strings.TrimPrefix(strings.Split(name, "/")[1], "clusters"),
				"operation": name,
			},
			"origin": "user,system",
		})
	}
	return out
}

func azureStackHCIClusterFixture(cluster *azureStackHCICluster) map[string]any {
	return map[string]any{
		"id":         azureStackHCIClusterID(cluster.SubscriptionID, cluster.ResourceGroup, cluster.Name),
		"name":       cluster.Name,
		"type":       "Microsoft.AzureStackHCI/clusters",
		"location":   cluster.Location,
		"identity":   azureStackHCICloneAnyMap(cluster.Identity),
		"tags":       azureStackHCIAnyMapFromStringMap(cluster.Tags),
		"properties": azureStackHCICloneAnyMap(cluster.Properties),
		"etag":       cluster.ETag,
		"systemData": azureStackHCISystemData(cluster.CreatedAt, cluster.UpdatedAt),
	}
}

func azureStackHCIArcSettingFixture(item *azureStackHCIArcSetting) map[string]any {
	return map[string]any{
		"id":         azureStackHCIArcSettingID(item.SubscriptionID, item.ResourceGroup, item.ClusterName, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/clusters/arcSettings",
		"location":   item.Location,
		"tags":       azureStackHCIAnyMapFromStringMap(item.Tags),
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCIExtensionFixture(item *azureStackHCIExtension) map[string]any {
	return map[string]any{
		"id":         azureStackHCIExtensionID(item.SubscriptionID, item.ResourceGroup, item.ClusterName, item.ArcSettingName, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/clusters/arcSettings/extensions",
		"location":   item.Location,
		"tags":       azureStackHCIAnyMapFromStringMap(item.Tags),
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCIDeploymentSettingFixture(item *azureStackHCIDeploymentSetting) map[string]any {
	return map[string]any{
		"id":         azureStackHCIDeploymentSettingID(item.SubscriptionID, item.ResourceGroup, item.ClusterName, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/clusters/deploymentSettings",
		"location":   item.Location,
		"tags":       azureStackHCIAnyMapFromStringMap(item.Tags),
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCIUpdateFixture(item *azureStackHCIUpdate) map[string]any {
	return map[string]any{
		"id":         azureStackHCIUpdateID(item.SubscriptionID, item.ResourceGroup, item.ClusterName, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/clusters/updates",
		"location":   item.Location,
		"tags":       azureStackHCIAnyMapFromStringMap(item.Tags),
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCIUpdateSummaryFixture(item *azureStackHCIUpdateSummary) map[string]any {
	return map[string]any{
		"id":         azureStackHCIUpdateSummaryID(item.SubscriptionID, item.ResourceGroup, item.ClusterName, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/clusters/updateSummaries",
		"location":   item.Location,
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCIUpdateRunFixture(item *azureStackHCIUpdateRun) map[string]any {
	return map[string]any{
		"id":         azureStackHCIUpdateRunID(item.SubscriptionID, item.ResourceGroup, item.ClusterName, item.UpdateName, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/clusters/updates/updateRuns",
		"location":   item.Location,
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCIEdgeDeviceFixture(item *azureStackHCIEdgeDevice) map[string]any {
	return map[string]any{
		"id":         azureStackHCIEdgeDeviceID(item.ResourceURI, item.Name),
		"name":       item.Name,
		"type":       "Microsoft.AzureStackHCI/edgeDevices",
		"kind":       item.Kind,
		"location":   item.Location,
		"tags":       azureStackHCIAnyMapFromStringMap(item.Tags),
		"properties": azureStackHCICloneAnyMap(item.Properties),
		"etag":       item.ETag,
		"systemData": azureStackHCISystemData(item.CreatedAt, item.UpdatedAt),
	}
}

func azureStackHCISystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":      createdAt.Format(time.RFC3339),
		"createdBy":      "stackyard",
		"lastModifiedAt": updatedAt.Format(time.RFC3339),
		"lastModifiedBy": "stackyard",
	}
}

func azureStackHCINotFound(path, kind, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "NotFound",
			"message": fmt.Sprintf("%s %q was not found", kind, name),
		},
		"provider": providerAzure,
		"path":     path,
	}
}

func azureStackHCIReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, payload, ok := azureStackHCIReadOptionalRequestBody(w, r, path)
	if !ok {
		return nil, nil, false
	}
	if len(body) == 0 {
		respondAzureInvalidRequest(w, path, "request body is required")
		return nil, nil, false
	}
	return body, payload, true
}

func azureStackHCIReadOptionalRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureStackHCIString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func azureStackHCIObject(value any) map[string]any {
	out, _ := value.(map[string]any)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func azureStackHCIStringMap(value any) map[string]string {
	src, _ := value.(map[string]any)
	if src == nil {
		if direct, ok := value.(map[string]string); ok {
			return azureStackHCICloneStringMap(direct)
		}
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for key, raw := range src {
		out[key] = strings.TrimSpace(azureStackHCIString(raw))
	}
	return out
}

func azureStackHCICloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureStackHCICloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureStackHCICloneAnyMap(typed)
		case []any:
			out[key] = append([]any(nil), typed...)
		default:
			out[key] = value
		}
	}
	return out
}

func azureStackHCIMergeAnyMap(dst, src map[string]any) map[string]any {
	out := azureStackHCICloneAnyMap(dst)
	for key, value := range src {
		out[key] = value
	}
	return out
}

func azureStackHCIAnyMapFromStringMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureStackHCISortByID(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureStackHCIString(items[i]["id"])) < strings.ToLower(azureStackHCIString(items[j]["id"]))
	})
}

func azureStackHCIClusterKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(azureStackHCIClusterID(subscriptionID, resourceGroup, name))
}

func azureStackHCIArcSettingKey(subscriptionID, resourceGroup, clusterName, name string) string {
	return strings.ToLower(azureStackHCIArcSettingID(subscriptionID, resourceGroup, clusterName, name))
}

func azureStackHCIExtensionKey(subscriptionID, resourceGroup, clusterName, arcSettingName, name string) string {
	return strings.ToLower(azureStackHCIExtensionID(subscriptionID, resourceGroup, clusterName, arcSettingName, name))
}

func azureStackHCIDeploymentSettingKey(subscriptionID, resourceGroup, clusterName, name string) string {
	return strings.ToLower(azureStackHCIDeploymentSettingID(subscriptionID, resourceGroup, clusterName, name))
}

func azureStackHCIUpdateKey(subscriptionID, resourceGroup, clusterName, name string) string {
	return strings.ToLower(azureStackHCIUpdateID(subscriptionID, resourceGroup, clusterName, name))
}

func azureStackHCIUpdateSummaryKey(subscriptionID, resourceGroup, clusterName, name string) string {
	return strings.ToLower(azureStackHCIUpdateSummaryID(subscriptionID, resourceGroup, clusterName, name))
}

func azureStackHCIUpdateRunKey(subscriptionID, resourceGroup, clusterName, updateName, name string) string {
	return strings.ToLower(azureStackHCIUpdateRunID(subscriptionID, resourceGroup, clusterName, updateName, name))
}

func azureStackHCIEdgeDeviceKey(resourceURI, name string) string {
	return strings.ToLower(azureStackHCIEdgeDeviceID(resourceURI, name))
}

func azureStackHCIClusterID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AzureStackHCI/clusters/%s", subscriptionID, resourceGroup, name)
}

func azureStackHCIArcSettingID(subscriptionID, resourceGroup, clusterName, name string) string {
	return azureStackHCIClusterID(subscriptionID, resourceGroup, clusterName) + "/arcSettings/" + name
}

func azureStackHCIExtensionID(subscriptionID, resourceGroup, clusterName, arcSettingName, name string) string {
	return azureStackHCIArcSettingID(subscriptionID, resourceGroup, clusterName, arcSettingName) + "/extensions/" + name
}

func azureStackHCIDeploymentSettingID(subscriptionID, resourceGroup, clusterName, name string) string {
	return azureStackHCIClusterID(subscriptionID, resourceGroup, clusterName) + "/deploymentSettings/" + name
}

func azureStackHCIUpdateID(subscriptionID, resourceGroup, clusterName, name string) string {
	return azureStackHCIClusterID(subscriptionID, resourceGroup, clusterName) + "/updates/" + name
}

func azureStackHCIUpdateSummaryID(subscriptionID, resourceGroup, clusterName, name string) string {
	return azureStackHCIClusterID(subscriptionID, resourceGroup, clusterName) + "/updateSummaries/" + name
}

func azureStackHCIUpdateRunID(subscriptionID, resourceGroup, clusterName, updateName, name string) string {
	return azureStackHCIUpdateID(subscriptionID, resourceGroup, clusterName, updateName) + "/updateRuns/" + name
}

func azureStackHCIEdgeDeviceID(resourceURI, name string) string {
	return strings.TrimRight(resourceURI, "/") + "/providers/Microsoft.AzureStackHCI/edgeDevices/" + name
}

func azureStackHCIETag(seed string, t time.Time) string {
	sum := sha1.Sum([]byte(seed + "|" + t.UTC().Format(time.RFC3339Nano)))
	return `W/"` + hex.EncodeToString(sum[:8]) + `"`
}
