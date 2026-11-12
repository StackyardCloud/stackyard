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

const azureNetAppAPIVersion = "2025-12-01"

type azureNetAppResource struct {
	ID             string
	Name           string
	Type           string
	SubscriptionID string
	ResourceGroup  string
	Location       string
	AccountName    string
	PoolName       string
	VolumeName     string
	ParentID       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

func (s *Server) handleAzureNetAppRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsNetAppPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureNetAppAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.NetApp") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureNetAppRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations":
		s.handleAzureNetAppLocationRoute(w, r, path, subscriptionID, tail[1:])
		return true
	case "netappaccounts":
		s.handleAzureNetAppAccountsRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func azureIsNetAppPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.NetApp") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureNetAppRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations", "netappaccounts":
		return true
	default:
		return false
	}
}

func azureNetAppRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.NetApp") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureNetAppAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureNetAppLocationRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID string, tail []string) {
	if len(tail) < 1 {
		respondAzureImplemented(w, path)
		return
	}
	location := strings.TrimSpace(tail[0])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	if len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "checknameavailability":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureNetAppCheckNameAvailability(w, r, path, subscriptionID, location)
	case "checkquotavailability", "checkquotaavailability":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureNetAppCheckQuotaAvailability(w, r, path, subscriptionID, location)
	case "checkfilepathavailability":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureNetAppCheckFilePathAvailability(w, r, path, subscriptionID, location)
	case "regioninfos":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppRegionInfosFixture(subscriptionID, location)})
	case "quotalimits":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppQuotaLimitsFixture(subscriptionID, location)})
	case "usages":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppUsagesFixture(subscriptionID, location)})
	case "querynetworksiblingset":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, azureNetAppNetworkSiblingFixture(subscriptionID, location))
	case "updatenetworksiblingset":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondJSON(w, http.StatusOK, azureNetAppNetworkSiblingFixture(subscriptionID, location))
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureNetAppCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	name := strings.TrimSpace(azureNetAppString(payload["name"]))
	if name == "" {
		respondAzureInvalidRequest(w, path, "name is required")
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	available := true
	for _, resource := range s.azureNetAppResources {
		if resource.SubscriptionID == subscriptionID &&
			strings.EqualFold(resource.Location, location) &&
			strings.EqualFold(resource.Name, name) &&
			strings.EqualFold(resource.Type, "Microsoft.NetApp/netAppAccounts") {
			available = false
			break
		}
	}
	response := map[string]any{
		"isAvailable":   available,
		"nameAvailable": available,
		"reason":        "",
		"message":       "name is available",
	}
	if !available {
		response["reason"] = "AlreadyExists"
		response["message"] = "The specified name is already in use."
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Server) handleAzureNetAppCheckQuotaAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	requested := azureNetAppObject(payload["properties"])
	quota := azureNetAppInt64(requested["currentUsage"], 0) + azureNetAppInt64(requested["requestedQuota"], 0)
	limit := int64(1024 * 1024 * 1024 * 10)
	available := quota <= limit
	respondJSON(w, http.StatusOK, map[string]any{
		"subscriptionId": subscriptionID,
		"location":       location,
		"isAvailable":    available,
		"currentUsage":   azureNetAppInt64(requested["currentUsage"], 0),
		"requestedQuota": azureNetAppInt64(requested["requestedQuota"], 0),
		"message":        map[bool]string{true: "quota is available", false: "quota request exceeds the mocked limit"}[available],
	})
}

func (s *Server) handleAzureNetAppCheckFilePathAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	filePath := strings.TrimSpace(azureNetAppString(payload["filePath"]))
	if filePath == "" {
		filePath = strings.TrimSpace(azureNetAppString(payload["name"]))
	}
	if filePath == "" {
		respondAzureInvalidRequest(w, path, "filePath is required")
		return
	}
	available := !strings.Contains(strings.ToLower(filePath), "taken")
	respondJSON(w, http.StatusOK, map[string]any{
		"subscriptionId": subscriptionID,
		"location":       location,
		"isAvailable":    available,
		"message":        map[bool]string{true: "file path is available", false: "file path is unavailable"}[available],
	})
}

func (s *Server) handleAzureNetAppAccountsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureNetAppListAccountsBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureNetAppListAccountsByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	accountName := strings.TrimSpace(tail[1])
	if accountName == "" {
		respondAzureInvalidRequest(w, path, "account name is required")
		return
	}
	accountID := azureNetAppAccountID(subscriptionID, resourceGroup, accountName)

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureNetAppGetResource(w, path, accountID, false)
		case http.MethodHead:
			s.handleAzureNetAppGetResource(w, path, accountID, true)
		case http.MethodPut:
			s.handleAzureNetAppCreateOrUpdateAccount(w, r, path, subscriptionID, resourceGroup, accountName)
		case http.MethodPatch:
			s.handleAzureNetAppPatchResource(w, r, path, accountID)
		case http.MethodDelete:
			s.handleAzureNetAppDeleteResource(w, accountID, true)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	childType := strings.ToLower(strings.TrimSpace(tail[2]))
	switch childType {
	case "capacitypools":
		s.handleAzureNetAppPoolsRoute(w, r, path, subscriptionID, resourceGroup, accountName, accountID, tail[3:])
	case "volumegroups":
		s.handleAzureNetAppNamedChildCollectionRoute(w, r, path, accountID, "Microsoft.NetApp/netAppAccounts/volumeGroups", "volume group", accountName, "", "", tail[3:])
	case "backuppolicies":
		s.handleAzureNetAppNamedChildCollectionRoute(w, r, path, accountID, "Microsoft.NetApp/netAppAccounts/backupPolicies", "backup policy", accountName, "", "", tail[3:])
	case "snapshotpolicies":
		s.handleAzureNetAppNamedChildCollectionRoute(w, r, path, accountID, "Microsoft.NetApp/netAppAccounts/snapshotPolicies", "snapshot policy", accountName, "", "", tail[3:])
	case "backupvaults":
		s.handleAzureNetAppBackupVaultsRoute(w, r, path, accountID, subscriptionID, resourceGroup, accountName, tail[3:])
	case "backups":
		if r.Method != http.MethodGet || len(tail) != 3 {
			respondAzureImplemented(w, path)
			return
		}
		s.handleAzureNetAppListBackupsByAccount(w, accountID)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureNetAppPoolsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, accountID string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureNetAppListChildrenByParent(w, accountID, "Microsoft.NetApp/netAppAccounts/capacityPools")
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	poolName := strings.TrimSpace(tail[0])
	if poolName == "" {
		respondAzureInvalidRequest(w, path, "pool name is required")
		return
	}
	poolID := azureNetAppPoolID(subscriptionID, resourceGroup, accountName, poolName)
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureNetAppGetResource(w, path, poolID, false)
		case http.MethodHead:
			s.handleAzureNetAppGetResource(w, path, poolID, true)
		case http.MethodPut:
			s.handleAzureNetAppCreateOrUpdatePool(w, r, path, subscriptionID, resourceGroup, accountName, poolName, accountID)
		case http.MethodPatch:
			s.handleAzureNetAppPatchResource(w, r, path, poolID)
		case http.MethodDelete:
			s.handleAzureNetAppDeleteResource(w, poolID, true)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if !strings.EqualFold(tail[1], "volumes") {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureNetAppVolumesRoute(w, r, path, subscriptionID, resourceGroup, accountName, poolName, poolID, tail[2:])
}

func (s *Server) handleAzureNetAppVolumesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, poolName, poolID string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureNetAppListChildrenByParent(w, poolID, "Microsoft.NetApp/netAppAccounts/capacityPools/volumes")
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	volumeName := strings.TrimSpace(tail[0])
	if volumeName == "" {
		respondAzureInvalidRequest(w, path, "volume name is required")
		return
	}
	volumeID := azureNetAppVolumeID(subscriptionID, resourceGroup, accountName, poolName, volumeName)
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureNetAppGetResource(w, path, volumeID, false)
		case http.MethodHead:
			s.handleAzureNetAppGetResource(w, path, volumeID, true)
		case http.MethodPut:
			s.handleAzureNetAppCreateOrUpdateVolume(w, r, path, subscriptionID, resourceGroup, accountName, poolName, volumeName, poolID)
		case http.MethodPatch:
			s.handleAzureNetAppPatchResource(w, r, path, volumeID)
		case http.MethodDelete:
			s.handleAzureNetAppDeleteResource(w, volumeID, true)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[1])) {
	case "snapshots":
		s.handleAzureNetAppNamedChildCollectionRoute(w, r, path, volumeID, "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/snapshots", "snapshot", accountName, poolName, volumeName, tail[2:])
	case "subvolumes":
		s.handleAzureNetAppNamedChildCollectionRoute(w, r, path, volumeID, "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/subvolumes", "subvolume", accountName, poolName, volumeName, tail[2:])
	case "volumequotarules":
		s.handleAzureNetAppNamedChildCollectionRoute(w, r, path, volumeID, "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/volumeQuotaRules", "volume quota rule", accountName, poolName, volumeName, tail[2:])
	case "backups":
		s.handleAzureNetAppBackupsRoute(w, r, path, volumeID, subscriptionID, resourceGroup, accountName, poolName, volumeName, tail[2:])
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureNetAppBackupVaultsRoute(w http.ResponseWriter, r *http.Request, path, accountID, subscriptionID, resourceGroup, accountName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureNetAppListChildrenByParent(w, accountID, "Microsoft.NetApp/netAppAccounts/backupVaults")
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	vaultName := strings.TrimSpace(tail[0])
	if vaultName == "" {
		respondAzureInvalidRequest(w, path, "backup vault name is required")
		return
	}
	vaultID := azureNetAppBackupVaultID(subscriptionID, resourceGroup, accountName, vaultName)
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureNetAppGetResource(w, path, vaultID, false)
		case http.MethodHead:
			s.handleAzureNetAppGetResource(w, path, vaultID, true)
		case http.MethodPut:
			s.handleAzureNetAppCreateOrUpdateSimpleChild(w, r, path, accountID, vaultID, vaultName, "Microsoft.NetApp/netAppAccounts/backupVaults", accountName, "", "")
		case http.MethodPatch:
			s.handleAzureNetAppPatchResource(w, r, path, vaultID)
		case http.MethodDelete:
			s.handleAzureNetAppDeleteResource(w, vaultID, true)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if !strings.EqualFold(tail[1], "backups") {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureNetAppBackupsRoute(w, r, path, vaultID, subscriptionID, resourceGroup, accountName, "", "", tail[2:])
}

func (s *Server) handleAzureNetAppBackupsRoute(w http.ResponseWriter, r *http.Request, path, parentID, subscriptionID, resourceGroup, accountName, poolName, volumeName string, tail []string) {
	resourceType := "Microsoft.NetApp/netAppAccounts/backups"
	if strings.Contains(strings.ToLower(parentID), "/backupvaults/") {
		resourceType = "Microsoft.NetApp/netAppAccounts/backupVaults/backups"
	} else if strings.Contains(strings.ToLower(parentID), "/volumes/") {
		resourceType = "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/backups"
	}
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureNetAppListChildrenByParent(w, parentID, resourceType)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	backupName := strings.TrimSpace(tail[0])
	if backupName == "" {
		respondAzureInvalidRequest(w, path, "backup name is required")
		return
	}
	backupID := strings.TrimRight(parentID, "/") + "/backups/" + backupName
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureNetAppGetResource(w, path, backupID, false)
	case http.MethodHead:
		s.handleAzureNetAppGetResource(w, path, backupID, true)
	case http.MethodPut:
		s.handleAzureNetAppCreateOrUpdateSimpleChild(w, r, path, parentID, backupID, backupName, resourceType, accountName, poolName, volumeName)
	case http.MethodPatch:
		s.handleAzureNetAppPatchResource(w, r, path, backupID)
	case http.MethodDelete:
		s.handleAzureNetAppDeleteResource(w, backupID, true)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureNetAppNamedChildCollectionRoute(w http.ResponseWriter, r *http.Request, path, parentID, resourceType, resourceLabel, accountName, poolName, volumeName string, tail []string) {
	if len(tail) == 0 {
		if r.Method == http.MethodGet {
			s.handleAzureNetAppListChildrenByParent(w, parentID, resourceType)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimSpace(tail[0])
	if name == "" {
		respondAzureInvalidRequest(w, path, resourceLabel+" name is required")
		return
	}
	resourceID := strings.TrimRight(parentID, "/") + "/" + azureNetAppCollectionSegment(resourceType) + "/" + name
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureNetAppGetResource(w, path, resourceID, false)
	case http.MethodHead:
		s.handleAzureNetAppGetResource(w, path, resourceID, true)
	case http.MethodPut:
		s.handleAzureNetAppCreateOrUpdateSimpleChild(w, r, path, parentID, resourceID, name, resourceType, accountName, poolName, volumeName)
	case http.MethodPatch:
		s.handleAzureNetAppPatchResource(w, r, path, resourceID)
	case http.MethodDelete:
		s.handleAzureNetAppDeleteResource(w, resourceID, true)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureNetAppCreateOrUpdateAccount(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string) {
	body, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureNetAppString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureNetAppObject(payload["properties"])
	now := time.Now().UTC()
	resourceID := azureNetAppAccountID(subscriptionID, resourceGroup, accountName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	existing := s.azureNetAppResources[resourceID]
	status := http.StatusOK
	createdAt := now
	if existing == nil {
		status = http.StatusCreated
		existing = &azureNetAppResource{
			ID:             resourceID,
			Name:           accountName,
			Type:           "Microsoft.NetApp/netAppAccounts",
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			AccountName:    accountName,
		}
	} else {
		createdAt = existing.CreatedAt
	}

	existing.Location = location
	existing.ParentID = ""
	existing.Tags = azureNetAppStringMap(payload["tags"])
	existing.Properties = azureNetAppNormalizeProperties(existing.Type, accountName, "", "", properties, createdAt, now)
	existing.CreatedAt = createdAt
	existing.UpdatedAt = now
	existing.ETag = azureNetAppETag(resourceID, body, now)
	s.azureNetAppResources[resourceID] = existing
	s.azureNetAppEnsureAccountFixturesLocked(existing)

	respondJSON(w, status, azureNetAppResourceFixture(existing))
}

func (s *Server) handleAzureNetAppCreateOrUpdatePool(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, poolName, accountID string) {
	body, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureNetAppString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureNetAppObject(payload["properties"])
	if azureNetAppInt64(properties["size"], 0) <= 0 {
		respondAzureInvalidRequest(w, path, "properties.size is required")
		return
	}
	if strings.TrimSpace(azureNetAppString(properties["serviceLevel"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.serviceLevel is required")
		return
	}
	now := time.Now().UTC()
	resourceID := azureNetAppPoolID(subscriptionID, resourceGroup, accountName, poolName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	existing := s.azureNetAppResources[resourceID]
	status := http.StatusOK
	createdAt := now
	if existing == nil {
		status = http.StatusCreated
		existing = &azureNetAppResource{
			ID:             resourceID,
			Name:           poolName,
			Type:           "Microsoft.NetApp/netAppAccounts/capacityPools",
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			AccountName:    accountName,
			PoolName:       poolName,
			ParentID:       accountID,
		}
	} else {
		createdAt = existing.CreatedAt
	}
	existing.Location = location
	existing.Tags = azureNetAppStringMap(payload["tags"])
	existing.Properties = azureNetAppNormalizeProperties(existing.Type, accountName, poolName, "", properties, createdAt, now)
	existing.CreatedAt = createdAt
	existing.UpdatedAt = now
	existing.ETag = azureNetAppETag(resourceID, body, now)
	s.azureNetAppResources[resourceID] = existing

	respondJSON(w, status, azureNetAppResourceFixture(existing))
}

func (s *Server) handleAzureNetAppCreateOrUpdateVolume(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, poolName, volumeName, poolID string) {
	body, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureNetAppString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureNetAppObject(payload["properties"])
	if strings.TrimSpace(azureNetAppString(properties["creationToken"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.creationToken is required")
		return
	}
	if strings.TrimSpace(azureNetAppString(properties["serviceLevel"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.serviceLevel is required")
		return
	}
	if azureNetAppInt64(properties["usageThreshold"], 0) <= 0 {
		respondAzureInvalidRequest(w, path, "properties.usageThreshold is required")
		return
	}
	if len(azureNetAppStringSlice(properties["protocolTypes"])) == 0 {
		respondAzureInvalidRequest(w, path, "properties.protocolTypes is required")
		return
	}
	now := time.Now().UTC()
	resourceID := azureNetAppVolumeID(subscriptionID, resourceGroup, accountName, poolName, volumeName)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	existing := s.azureNetAppResources[resourceID]
	status := http.StatusOK
	createdAt := now
	if existing == nil {
		status = http.StatusCreated
		existing = &azureNetAppResource{
			ID:             resourceID,
			Name:           volumeName,
			Type:           "Microsoft.NetApp/netAppAccounts/capacityPools/volumes",
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			AccountName:    accountName,
			PoolName:       poolName,
			VolumeName:     volumeName,
			ParentID:       poolID,
		}
	} else {
		createdAt = existing.CreatedAt
	}
	existing.Location = location
	existing.Tags = azureNetAppStringMap(payload["tags"])
	existing.Properties = azureNetAppNormalizeProperties(existing.Type, accountName, poolName, volumeName, properties, createdAt, now)
	existing.CreatedAt = createdAt
	existing.UpdatedAt = now
	existing.ETag = azureNetAppETag(resourceID, body, now)
	s.azureNetAppResources[resourceID] = existing
	s.azureNetAppEnsureVolumeFixturesLocked(existing)

	respondJSON(w, status, azureNetAppResourceFixture(existing))
}

func (s *Server) handleAzureNetAppCreateOrUpdateSimpleChild(w http.ResponseWriter, r *http.Request, path, parentID, resourceID, name, resourceType, accountName, poolName, volumeName string) {
	body, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureNetAppString(payload["location"]))
	properties := azureNetAppObject(payload["properties"])

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	parent := s.azureNetAppResources[parentID]
	if parent == nil {
		respondJSON(w, http.StatusNotFound, azureNetAppNotFound(path, name))
		return
	}
	if location == "" {
		location = parent.Location
	}
	now := time.Now().UTC()
	existing := s.azureNetAppResources[resourceID]
	status := http.StatusOK
	createdAt := now
	if existing == nil {
		status = http.StatusCreated
		existing = &azureNetAppResource{
			ID:             resourceID,
			Name:           name,
			Type:           resourceType,
			SubscriptionID: parent.SubscriptionID,
			ResourceGroup:  parent.ResourceGroup,
			AccountName:    accountName,
			PoolName:       poolName,
			VolumeName:     volumeName,
			ParentID:       parentID,
		}
	} else {
		createdAt = existing.CreatedAt
	}
	existing.Location = location
	existing.Tags = azureNetAppStringMap(payload["tags"])
	existing.Properties = azureNetAppNormalizeProperties(resourceType, accountName, poolName, volumeName, properties, createdAt, now)
	existing.CreatedAt = createdAt
	existing.UpdatedAt = now
	existing.ETag = azureNetAppETag(resourceID, body, now)
	s.azureNetAppResources[resourceID] = existing
	if strings.EqualFold(resourceType, "Microsoft.NetApp/netAppAccounts/backupVaults") {
		s.azureNetAppEnsureBackupFixturesLocked(existing)
	}
	respondJSON(w, status, azureNetAppResourceFixture(existing))
}

func (s *Server) handleAzureNetAppPatchResource(w http.ResponseWriter, r *http.Request, path, resourceID string) {
	body, payload, ok := azureNetAppReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureNetAppResources[resourceID]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureNetAppNotFound(path, azureNetAppNameFromID(resourceID)))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}
	if _, ok := payload["location"]; ok {
		location := strings.TrimSpace(azureNetAppString(payload["location"]))
		if location == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		resource.Location = location
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureNetAppStringMap(payload["tags"])
	}
	if _, ok := payload["properties"]; ok {
		resource.Properties = azureNetAppMergeObjects(resource.Properties, azureNetAppObject(payload["properties"]))
		resource.Properties = azureNetAppNormalizeProperties(resource.Type, resource.AccountName, resource.PoolName, resource.VolumeName, resource.Properties, resource.CreatedAt, now)
	}
	resource.UpdatedAt = now
	resource.ETag = azureNetAppETag(resourceID, body, now)
	respondJSON(w, http.StatusOK, azureNetAppResourceFixture(resource))
}

func (s *Server) handleAzureNetAppGetResource(w http.ResponseWriter, path, resourceID string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureNetAppResources[resourceID]
	s.providerStorageMu.Unlock()
	if resource == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureNetAppNotFound(path, azureNetAppNameFromID(resourceID)))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureNetAppResourceFixture(resource))
}

func (s *Server) handleAzureNetAppDeleteResource(w http.ResponseWriter, resourceID string, accepted bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	if _, ok := s.azureNetAppResources[resourceID]; !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	delete(s.azureNetAppResources, resourceID)
	prefix := strings.TrimRight(resourceID, "/") + "/"
	for id := range s.azureNetAppResources {
		if strings.HasPrefix(id, prefix) {
			delete(s.azureNetAppResources, id)
		}
	}
	if accepted {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureNetAppListAccountsBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppListResourcesLocked(s.azureNetAppResources, subscriptionID, "", "Microsoft.NetApp/netAppAccounts", "")})
}

func (s *Server) handleAzureNetAppListAccountsByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppListResourcesLocked(s.azureNetAppResources, subscriptionID, resourceGroup, "Microsoft.NetApp/netAppAccounts", "")})
}

func (s *Server) handleAzureNetAppListChildrenByParent(w http.ResponseWriter, parentID, resourceType string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{"value": azureNetAppListResourcesLocked(s.azureNetAppResources, "", "", resourceType, parentID)})
}

func (s *Server) handleAzureNetAppListBackupsByAccount(w http.ResponseWriter, accountID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	values := make([]map[string]any, 0)
	for _, resource := range s.azureNetAppResources {
		if !strings.Contains(strings.ToLower(resource.Type), "/backups") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(resource.ID), strings.ToLower(strings.TrimRight(accountID, "/")+"/")) {
			continue
		}
		values = append(values, azureNetAppResourceFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureNetAppString(values[i]["id"]) < azureNetAppString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func azureNetAppListResourcesLocked(resources map[string]*azureNetAppResource, subscriptionID, resourceGroup, resourceType, parentID string) []map[string]any {
	values := make([]map[string]any, 0)
	for _, resource := range resources {
		if subscriptionID != "" && !strings.EqualFold(resource.SubscriptionID, subscriptionID) {
			continue
		}
		if resourceGroup != "" && !strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			continue
		}
		if resourceType != "" && !strings.EqualFold(resource.Type, resourceType) {
			continue
		}
		if parentID != "" && !strings.EqualFold(resource.ParentID, parentID) {
			continue
		}
		values = append(values, azureNetAppResourceFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureNetAppString(values[i]["id"]) < azureNetAppString(values[j]["id"])
	})
	return values
}

func (s *Server) azureNetAppEnsureAccountFixturesLocked(account *azureNetAppResource) {
	s.azureNetAppEnsureSimpleFixtureLocked(account, account.ID+"/volumeGroups/group-a", "group-a", "Microsoft.NetApp/netAppAccounts/volumeGroups")
	s.azureNetAppEnsureSimpleFixtureLocked(account, account.ID+"/backupPolicies/policy-a", "policy-a", "Microsoft.NetApp/netAppAccounts/backupPolicies")
	s.azureNetAppEnsureSimpleFixtureLocked(account, account.ID+"/snapshotPolicies/snapshot-policy-a", "snapshot-policy-a", "Microsoft.NetApp/netAppAccounts/snapshotPolicies")
	vault := s.azureNetAppEnsureSimpleFixtureLocked(account, account.ID+"/backupVaults/vault-a", "vault-a", "Microsoft.NetApp/netAppAccounts/backupVaults")
	s.azureNetAppEnsureBackupFixturesLocked(vault)
}

func (s *Server) azureNetAppEnsureVolumeFixturesLocked(volume *azureNetAppResource) {
	s.azureNetAppEnsureSimpleFixtureLocked(volume, volume.ID+"/snapshots/snapshot-a", "snapshot-a", "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/snapshots")
	s.azureNetAppEnsureSimpleFixtureLocked(volume, volume.ID+"/subvolumes/subvolume-a", "subvolume-a", "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/subvolumes")
	s.azureNetAppEnsureSimpleFixtureLocked(volume, volume.ID+"/volumeQuotaRules/quota-a", "quota-a", "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/volumeQuotaRules")
	s.azureNetAppEnsureSimpleFixtureLocked(volume, volume.ID+"/backups/backup-a", "backup-a", "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/backups")
}

func (s *Server) azureNetAppEnsureBackupFixturesLocked(parent *azureNetAppResource) {
	if parent == nil {
		return
	}
	s.azureNetAppEnsureSimpleFixtureLocked(parent, parent.ID+"/backups/backup-a", "backup-a", "Microsoft.NetApp/netAppAccounts/backupVaults/backups")
}

func (s *Server) azureNetAppEnsureSimpleFixtureLocked(parent *azureNetAppResource, resourceID, name, resourceType string) *azureNetAppResource {
	if existing := s.azureNetAppResources[resourceID]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	resource := &azureNetAppResource{
		ID:             resourceID,
		Name:           name,
		Type:           resourceType,
		SubscriptionID: parent.SubscriptionID,
		ResourceGroup:  parent.ResourceGroup,
		Location:       parent.Location,
		AccountName:    parent.AccountName,
		PoolName:       parent.PoolName,
		VolumeName:     parent.VolumeName,
		ParentID:       parent.ID,
		Tags:           map[string]string{"seeded": "true"},
		Properties:     azureNetAppNormalizeProperties(resourceType, parent.AccountName, parent.PoolName, parent.VolumeName, map[string]any{}, now, now),
		CreatedAt:      now,
		UpdatedAt:      now,
		ETag:           azureNetAppETag(resourceID, []byte(resourceID), now),
	}
	s.azureNetAppResources[resourceID] = resource
	return resource
}

func azureNetAppResourceFixture(resource *azureNetAppResource) map[string]any {
	if resource == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id":         resource.ID,
		"name":       resource.Name,
		"type":       resource.Type,
		"location":   resource.Location,
		"tags":       azureNetAppAnyMap(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureNetAppSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureNetAppCloneObject(resource.Properties),
	}
}

func azureNetAppNormalizeProperties(resourceType, accountName, poolName, volumeName string, properties map[string]any, createdAt, now time.Time) map[string]any {
	cloned := azureNetAppCloneObject(properties)
	cloned["provisioningState"] = "Succeeded"
	switch resourceType {
	case "Microsoft.NetApp/netAppAccounts":
		if _, ok := cloned["activeDirectories"]; !ok {
			cloned["activeDirectories"] = []any{
				map[string]any{
					"username":      "ad-admin",
					"smbServerName": strings.ToUpper(accountName) + "SMB",
					"dns":           "10.0.0.4",
				},
			}
		}
		if _, ok := cloned["encryption"]; !ok {
			cloned["encryption"] = map[string]any{"keySource": "Microsoft.NetApp"}
		}
		if _, ok := cloned["nfsV4IDDomain"]; !ok {
			cloned["nfsV4IDDomain"] = accountName + ".local"
		}
	case "Microsoft.NetApp/netAppAccounts/capacityPools":
		if _, ok := cloned["size"]; !ok {
			cloned["size"] = int64(4398046511104)
		}
		if _, ok := cloned["serviceLevel"]; !ok {
			cloned["serviceLevel"] = "Premium"
		}
		if _, ok := cloned["qosType"]; !ok {
			cloned["qosType"] = "Auto"
		}
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes":
		if _, ok := cloned["creationToken"]; !ok {
			cloned["creationToken"] = volumeName
		}
		if _, ok := cloned["serviceLevel"]; !ok {
			cloned["serviceLevel"] = "Premium"
		}
		if _, ok := cloned["usageThreshold"]; !ok {
			cloned["usageThreshold"] = int64(1073741824)
		}
		if _, ok := cloned["protocolTypes"]; !ok {
			cloned["protocolTypes"] = []any{"NFSv4.1"}
		}
		if _, ok := cloned["subnetId"]; !ok {
			cloned["subnetId"] = "/subscriptions/" + "00000000-0000-0000-0000-000000000000/resourceGroups/rg-network/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/netapp"
		}
		if _, ok := cloned["mountTargets"]; !ok {
			cloned["mountTargets"] = []any{
				map[string]any{
					"ipAddress":     "10.0.0.10",
					"smbServerFqdn": volumeName + ".file.core.local",
				},
			}
		}
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/snapshots":
		if _, ok := cloned["snapshotId"]; !ok {
			cloned["snapshotId"] = "snapshot-" + volumeName
		}
		if _, ok := cloned["created"]; !ok {
			cloned["created"] = createdAt.Format(time.RFC3339)
		}
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/subvolumes":
		if _, ok := cloned["path"]; !ok {
			cloned["path"] = "/" + accountName + "/" + poolName + "/" + volumeName + "/subvolume-a"
		}
		if _, ok := cloned["size"]; !ok {
			cloned["size"] = int64(5242880)
		}
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/volumeQuotaRules":
		if _, ok := cloned["quotaTarget"]; !ok {
			cloned["quotaTarget"] = "uid:1000"
		}
		if _, ok := cloned["quotaSizeInKiBs"]; !ok {
			cloned["quotaSizeInKiBs"] = int64(1024)
		}
	case "Microsoft.NetApp/netAppAccounts/volumeGroups":
		if _, ok := cloned["groupMetaData"]; !ok {
			cloned["groupMetaData"] = map[string]any{"applicationType": "SAP"}
		}
		if _, ok := cloned["volumes"]; !ok {
			cloned["volumes"] = []any{}
		}
	case "Microsoft.NetApp/netAppAccounts/backupPolicies":
		if _, ok := cloned["enabled"]; !ok {
			cloned["enabled"] = true
		}
		if _, ok := cloned["dailyBackupsToKeep"]; !ok {
			cloned["dailyBackupsToKeep"] = 7
		}
	case "Microsoft.NetApp/netAppAccounts/snapshotPolicies":
		if _, ok := cloned["enabled"]; !ok {
			cloned["enabled"] = true
		}
		if _, ok := cloned["hourlySchedule"]; !ok {
			cloned["hourlySchedule"] = map[string]any{"snapshotsToKeep": 4, "minute": 15}
		}
	case "Microsoft.NetApp/netAppAccounts/backupVaults":
		if _, ok := cloned["vaultState"]; !ok {
			cloned["vaultState"] = "Ready"
		}
	case "Microsoft.NetApp/netAppAccounts/backupVaults/backups", "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/backups", "Microsoft.NetApp/netAppAccounts/backups":
		if _, ok := cloned["backupId"]; !ok {
			cloned["backupId"] = "backup-" + volumeName
		}
		if _, ok := cloned["label"]; !ok {
			cloned["label"] = "daily"
		}
		if _, ok := cloned["creationDate"]; !ok {
			cloned["creationDate"] = createdAt.Format(time.RFC3339)
		}
	}
	cloned["unixPermissions"] = map[string]any{"mode": "0755", "lastEvaluatedAt": now.Format(time.RFC3339)}
	return cloned
}

func azureNetAppOperationsFixture() []map[string]any {
	names := []string{
		"Microsoft.NetApp/netAppAccounts/read",
		"Microsoft.NetApp/netAppAccounts/write",
		"Microsoft.NetApp/netAppAccounts/delete",
		"Microsoft.NetApp/netAppAccounts/capacityPools/read",
		"Microsoft.NetApp/netAppAccounts/capacityPools/write",
		"Microsoft.NetApp/netAppAccounts/capacityPools/delete",
		"Microsoft.NetApp/netAppAccounts/capacityPools/volumes/read",
		"Microsoft.NetApp/netAppAccounts/capacityPools/volumes/write",
		"Microsoft.NetApp/netAppAccounts/capacityPools/volumes/delete",
		"Microsoft.NetApp/netAppAccounts/capacityPools/volumes/snapshots/read",
		"Microsoft.NetApp/netAppAccounts/capacityPools/volumes/subvolumes/read",
		"Microsoft.NetApp/netAppAccounts/capacityPools/volumes/volumeQuotaRules/read",
		"Microsoft.NetApp/netAppAccounts/backupVaults/read",
		"Microsoft.NetApp/netAppAccounts/backupPolicies/read",
		"Microsoft.NetApp/netAppAccounts/volumeGroups/read",
		"Microsoft.NetApp/locations/checkNameAvailability/action",
		"Microsoft.NetApp/locations/checkQuotaAvailability/action",
		"Microsoft.NetApp/locations/checkFilePathAvailability/action",
	}
	values := make([]map[string]any, 0, len(names))
	for _, name := range names {
		values = append(values, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":    "Microsoft NetApp",
				"resource":    "Azure NetApp Files",
				"operation":   name,
				"description": "Mocked NetApp operation",
			},
			"origin": "user,system",
			"serviceSpecification": map[string]any{
				"metricSpecifications": []any{},
			},
		})
	}
	return values
}

func azureNetAppRegionInfosFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.NetApp/locations/" + location + "/regionInfos/default",
			"name": "default",
			"type": "Microsoft.NetApp/locations/regionInfos",
			"properties": map[string]any{
				"availabilityZoneMappings": []any{
					map[string]any{"logicZone": "1", "physicalZone": "zone-a"},
					map[string]any{"logicZone": "2", "physicalZone": "zone-b"},
				},
				"supportedServiceLevels": []any{"Standard", "Premium", "Ultra"},
			},
		},
	}
}

func azureNetAppQuotaLimitsFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.NetApp/locations/" + location + "/quotaLimits/capacity",
			"name": "capacity",
			"type": "Microsoft.NetApp/locations/quotaLimits",
			"properties": map[string]any{
				"currentValue": 64,
				"defaultValue": 128,
				"unit":         "TiB",
			},
		},
	}
}

func azureNetAppUsagesFixture(subscriptionID, location string) []map[string]any {
	return []map[string]any{
		{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.NetApp/locations/" + location + "/usages/volumes",
			"name": "Volumes",
			"type": "Microsoft.NetApp/locations/usages",
			"properties": map[string]any{
				"currentValue": 3,
				"limit":        100,
				"unit":         "Count",
			},
		},
	}
}

func azureNetAppNetworkSiblingFixture(subscriptionID, location string) map[string]any {
	return map[string]any{
		"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.NetApp/locations/" + location + "/networkSiblingSet/default",
		"name": "default",
		"type": "Microsoft.NetApp/locations/networkSiblingSet",
		"properties": map[string]any{
			"delegatedSubnets": []any{
				map[string]any{"id": "/subscriptions/" + subscriptionID + "/resourceGroups/rg-network/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/netapp"},
			},
			"networkFeatures": []any{"Basic", "Standard"},
		},
	}
}

func azureNetAppReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
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

func azureNetAppCloneObject(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureNetAppCloneObject(nested)
			continue
		}
		if values, ok := value.([]any); ok {
			cloned := make([]any, len(values))
			copy(cloned, values)
			out[key] = cloned
			continue
		}
		out[key] = value
	}
	return out
}

func azureNetAppMergeObjects(base, overlay map[string]any) map[string]any {
	if len(base) == 0 {
		return azureNetAppCloneObject(overlay)
	}
	out := azureNetAppCloneObject(base)
	for key, value := range overlay {
		existing, exists := out[key]
		existingObj, existingIsObj := existing.(map[string]any)
		valueObj, valueIsObj := value.(map[string]any)
		if exists && existingIsObj && valueIsObj {
			out[key] = azureNetAppMergeObjects(existingObj, valueObj)
			continue
		}
		out[key] = value
	}
	return out
}

func azureNetAppString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func azureNetAppObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func azureNetAppStringMap(value any) map[string]string {
	object := azureNetAppObject(value)
	out := make(map[string]string, len(object))
	for key, raw := range object {
		out[key] = azureNetAppString(raw)
	}
	return out
}

func azureNetAppAnyMap(in map[string]string) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func azureNetAppStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s := strings.TrimSpace(azureNetAppString(item)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func azureNetAppInt64(value any, fallback int64) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return fallback
	}
}

func azureNetAppETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.UTC().Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureNetAppSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureNetAppAccountID(subscriptionID, resourceGroup, accountName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.NetApp/netAppAccounts/" + accountName
}

func azureNetAppPoolID(subscriptionID, resourceGroup, accountName, poolName string) string {
	return azureNetAppAccountID(subscriptionID, resourceGroup, accountName) + "/capacityPools/" + poolName
}

func azureNetAppVolumeID(subscriptionID, resourceGroup, accountName, poolName, volumeName string) string {
	return azureNetAppPoolID(subscriptionID, resourceGroup, accountName, poolName) + "/volumes/" + volumeName
}

func azureNetAppBackupVaultID(subscriptionID, resourceGroup, accountName, vaultName string) string {
	return azureNetAppAccountID(subscriptionID, resourceGroup, accountName) + "/backupVaults/" + vaultName
}

func azureNetAppCollectionSegment(resourceType string) string {
	switch resourceType {
	case "Microsoft.NetApp/netAppAccounts/volumeGroups":
		return "volumeGroups"
	case "Microsoft.NetApp/netAppAccounts/backupPolicies":
		return "backupPolicies"
	case "Microsoft.NetApp/netAppAccounts/snapshotPolicies":
		return "snapshotPolicies"
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/snapshots":
		return "snapshots"
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/subvolumes":
		return "subvolumes"
	case "Microsoft.NetApp/netAppAccounts/capacityPools/volumes/volumeQuotaRules":
		return "volumeQuotaRules"
	default:
		return ""
	}
}

func azureNetAppNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("The Azure NetApp Files resource %q was not found for %s.", name, path),
		},
	}
}

func azureNetAppNameFromID(resourceID string) string {
	parts := splitPathSegments(resourceID)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
