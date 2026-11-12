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

const azureStackAdminAPIVersion = "2019-08-08-preview"

type azureStackAdminSettings struct {
	SubscriptionID string
	Location       string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureStackAdminService struct {
	SubscriptionID string
	Name           string
	Location       string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureStackAdminStorageAccount struct {
	SubscriptionID string
	Location       string
	AccountID      string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureStackAdminAcquisition struct {
	SubscriptionID string
	Location       string
	Name           string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureStackAdminAsyncOperation struct {
	SubscriptionID string
	Location       string
	ID             string
	Name           string
	Action         string
	ResourceID     string
	StatusCode     int
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Server) handleAzureStackAdminRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsStackAdminPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureStackAdminAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Storage.Admin") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureStackAdminOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, tail, ok := azureStackAdminRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "storageServices"):
		if r.Method == http.MethodGet {
			s.handleAzureStackAdminStorageServicesList(w, r, subscriptionID)
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	case len(tail) >= 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "settings"):
		s.handleAzureStackAdminSettingsRoute(w, r, path, subscriptionID, tail[1], tail[2:])
		return true
	case len(tail) >= 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "storageAccounts"):
		s.handleAzureStackAdminStorageAccountsRoute(w, r, path, subscriptionID, tail[1], tail[2:])
		return true
	case len(tail) >= 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "acquisitions"):
		s.handleAzureStackAdminAcquisitionsRoute(w, r, path, subscriptionID, tail[1], tail[2:])
		return true
	case len(tail) >= 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "asyncOperations"):
		s.handleAzureStackAdminAsyncOperationsRoute(w, r, path, subscriptionID, tail[1], tail[2:])
		return true
	default:
		return false
	}
}

func azureIsStackAdminPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Storage.Admin") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, tail, ok := azureStackAdminRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "storageServices") {
		return true
	}
	return len(tail) >= 3 && strings.EqualFold(tail[0], "locations")
}

func azureStackAdminAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureStackAdminRouteContext(segments []string) (subscriptionID string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Storage.Admin") {
			return subscriptionID, segments[idx+2:], true
		}
	}
	return "", nil, false
}

func (s *Server) handleAzureStackAdminStorageServicesList(w http.ResponseWriter, r *http.Request, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, "local")
	items := make([]map[string]any, 0)
	for _, service := range s.azureStackAdminServices {
		if strings.EqualFold(service.SubscriptionID, subscriptionID) {
			items = append(items, azureStackAdminServiceFixture(service))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureStackAdminString(items[i]["id"])) < strings.ToLower(azureStackAdminString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackAdminSettingsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureStackAdminSettingsGet(w, path, subscriptionID, location)
	case http.MethodHead:
		s.handleAzureStackAdminSettingsHead(w, path, subscriptionID, location)
	case http.MethodPut:
		s.handleAzureStackAdminSettingsUpdate(w, r, path, subscriptionID, location)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureStackAdminSettingsGet(w http.ResponseWriter, path, subscriptionID, location string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	settings, ok := s.azureStackAdminSettings[azureStackAdminSettingsKey(subscriptionID, location)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackAdminNotFound(path, location))
		return
	}
	respondJSON(w, http.StatusOK, azureStackAdminSettingsFixture(settings))
}

func (s *Server) handleAzureStackAdminSettingsHead(w http.ResponseWriter, path, subscriptionID, location string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	if _, ok := s.azureStackAdminSettings[azureStackAdminSettingsKey(subscriptionID, location)]; !ok {
		respondJSON(w, http.StatusNotFound, azureStackAdminNotFound(path, location))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureStackAdminSettingsUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureStackAdminReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureStackAdminObject(payload["properties"])
	daysValue, found := properties["retentionPeriodForDeletedStorageAccountsInDays"]
	if !found {
		respondAzureInvalidRequest(w, path, "retentionPeriodForDeletedStorageAccountsInDays is required")
		return
	}
	days, ok := azureStackAdminInt(daysValue)
	if !ok || days < 0 {
		respondAzureInvalidRequest(w, path, "retentionPeriodForDeletedStorageAccountsInDays must be a non-negative integer")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	now := time.Now().UTC()
	settings := s.azureStackAdminSettings[azureStackAdminSettingsKey(subscriptionID, location)]
	settings.Properties["retentionPeriodForDeletedStorageAccountsInDays"] = days
	settings.Properties["provisioningState"] = "Succeeded"
	settings.UpdatedAt = now
	respondJSON(w, http.StatusOK, azureStackAdminSettingsFixture(settings))
}

func (s *Server) handleAzureStackAdminStorageAccountsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	switch {
	case len(tail) == 1 && r.Method == http.MethodGet:
		s.handleAzureStackAdminStorageAccountsList(w, r, subscriptionID, location)
	case len(tail) == 2:
		switch r.Method {
		case http.MethodGet:
			s.handleAzureStackAdminStorageAccountGet(w, path, subscriptionID, location, tail[1], false)
		case http.MethodHead:
			s.handleAzureStackAdminStorageAccountGet(w, path, subscriptionID, location, tail[1], true)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	case len(tail) == 3 && strings.EqualFold(tail[2], "undelete"):
		if r.Method == http.MethodPost {
			s.handleAzureStackAdminStorageAccountUndelete(w, r, path, subscriptionID, location, tail[1])
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case len(tail) == 3 && strings.EqualFold(tail[2], "reclaimStorageCapacity"):
		if r.Method == http.MethodPost {
			s.handleAzureStackAdminStorageAccountReclaim(w, r, path, subscriptionID, location, tail[1])
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureStackAdminStorageAccountsList(w http.ResponseWriter, r *http.Request, subscriptionID, location string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	items := make([]map[string]any, 0)
	summary := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("summary")), "true")
	filter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("$filter")))
	for _, account := range s.azureStackAdminAccounts {
		if !strings.EqualFold(account.SubscriptionID, subscriptionID) || !strings.EqualFold(account.Location, location) {
			continue
		}
		if filter != "" {
			search := strings.ToLower(account.AccountID + " " + azureStackAdminString(account.Properties["tenantStorageAccountName"]) + " " + azureStackAdminString(account.Properties["status"]))
			if !strings.Contains(search, filter) {
				continue
			}
		}
		items = append(items, azureStackAdminStorageAccountFixture(account, summary))
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureStackAdminString(items[i]["id"])) < strings.ToLower(azureStackAdminString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackAdminStorageAccountGet(w http.ResponseWriter, path, subscriptionID, location, accountID string, head bool) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	account, ok := s.azureStackAdminAccounts[azureStackAdminStorageAccountKey(subscriptionID, location, accountID)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackAdminNotFound(path, accountID))
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureStackAdminStorageAccountFixture(account, false))
}

func (s *Server) handleAzureStackAdminStorageAccountUndelete(w http.ResponseWriter, r *http.Request, path, subscriptionID, location, accountID string) {
	_, payload, ok := azureStackAdminReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	account, ok := s.azureStackAdminAccounts[azureStackAdminStorageAccountKey(subscriptionID, location, accountID)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackAdminNotFound(path, accountID))
		return
	}
	if !strings.EqualFold(azureStackAdminString(account.Properties["status"]), "Deleted") {
		respondAzureInvalidRequest(w, path, "storage account must be in Deleted state to undelete")
		return
	}
	if newName := strings.TrimSpace(azureStackAdminString(payload["newAccountName"])); newName != "" {
		account.Properties["tenantStorageAccountName"] = newName
	}
	now := time.Now().UTC()
	account.Properties["status"] = "Active"
	account.Properties["deleted"] = false
	account.Properties["lastUndeleteAt"] = now.Format(time.RFC3339)
	account.UpdatedAt = now
	op := s.azureStackAdminRegisterAsyncOperation(subscriptionID, location, "undelete", azureStackAdminStorageAccountID(subscriptionID, location, accountID), map[string]any{
		"status":                   "Succeeded",
		"provisioningState":        "Succeeded",
		"tenantStorageAccountName": account.Properties["tenantStorageAccountName"],
	})
	asyncPath := "/azure" + azureStackAdminAsyncOperationID(subscriptionID, location, op.ID) + "?api-version=" + azureStackAdminAPIVersion
	w.Header().Set("Azure-AsyncOperation", asyncPath)
	w.Header().Set("Location", asyncPath)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureStackAdminStorageAccountReclaim(w http.ResponseWriter, r *http.Request, path, subscriptionID, location, accountID string) {
	_, _, ok := azureStackAdminReadRequestBody(w, r, path)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	account, ok := s.azureStackAdminAccounts[azureStackAdminStorageAccountKey(subscriptionID, location, accountID)]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackAdminNotFound(path, accountID))
		return
	}
	now := time.Now().UTC()
	account.Properties["lastReclaimStorageCapacityAt"] = now.Format(time.RFC3339)
	account.Properties["reclaimedCapacityBytes"] = 1073741824
	account.UpdatedAt = now
	op := s.azureStackAdminRegisterAsyncOperation(subscriptionID, location, "reclaimStorageCapacity", azureStackAdminStorageAccountID(subscriptionID, location, accountID), map[string]any{
		"status":                 "Succeeded",
		"reclaimedCapacityBytes": 1073741824,
	})
	asyncPath := "/azure" + azureStackAdminAsyncOperationID(subscriptionID, location, op.ID) + "?api-version=" + azureStackAdminAPIVersion
	w.Header().Set("Azure-AsyncOperation", asyncPath)
	w.Header().Set("Location", asyncPath)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureStackAdminAcquisitionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) != 1 || r.Method != http.MethodGet {
		if len(tail) == 1 {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		respondAzureImplemented(w, path)
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	items := make([]map[string]any, 0)
	for _, acquisition := range s.azureStackAdminAcquisitions {
		if strings.EqualFold(acquisition.SubscriptionID, subscriptionID) && strings.EqualFold(acquisition.Location, location) {
			items = append(items, azureStackAdminAcquisitionFixture(acquisition))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureStackAdminString(items[i]["id"])) < strings.ToLower(azureStackAdminString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureStackAdminAsyncOperationsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	s.azureStackAdminEnsureSeeded(subscriptionID, location)
	op, ok := s.azureStackAdminAsyncOperations[azureStackAdminAsyncOperationKey(subscriptionID, location, tail[1])]
	if !ok {
		respondJSON(w, http.StatusNotFound, azureStackAdminNotFound(path, tail[1]))
		return
	}
	respondJSON(w, http.StatusOK, azureStackAdminAsyncOperationFixture(op))
}

func (s *Server) azureStackAdminEnsureSeeded(subscriptionID, location string) {
	settingsKey := azureStackAdminSettingsKey(subscriptionID, location)
	if _, ok := s.azureStackAdminSettings[settingsKey]; !ok {
		now := time.Now().UTC()
		s.azureStackAdminSettings[settingsKey] = &azureStackAdminSettings{
			SubscriptionID: subscriptionID,
			Location:       location,
			Properties: map[string]any{
				"retentionPeriodForDeletedStorageAccountsInDays": 7,
				"provisioningState": "Succeeded",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	serviceKey := azureStackAdminServiceKey(subscriptionID, "storage-service-"+strings.ToLower(location))
	if _, ok := s.azureStackAdminServices[serviceKey]; !ok {
		now := time.Now().UTC()
		s.azureStackAdminServices[serviceKey] = &azureStackAdminService{
			SubscriptionID: subscriptionID,
			Name:           "storage-service-" + strings.ToLower(location),
			Location:       location,
			Properties: map[string]any{
				"status":                    "Running",
				"healthState":               "Healthy",
				"resourceProviderNamespace": "Microsoft.Storage.Admin",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	activeKey := azureStackAdminStorageAccountKey(subscriptionID, location, "account-active-001")
	if _, ok := s.azureStackAdminAccounts[activeKey]; !ok {
		now := time.Now().UTC()
		s.azureStackAdminAccounts[activeKey] = &azureStackAdminStorageAccount{
			SubscriptionID: subscriptionID,
			Location:       location,
			AccountID:      "account-active-001",
			Properties: map[string]any{
				"status":                   "Active",
				"tenantViewId":             "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/tenant-rg/providers/Microsoft.Storage/storageAccounts/tenantactive001",
				"tenantSubscriptionId":     "11111111-1111-1111-1111-111111111111",
				"tenantStorageAccountName": "tenantactive001",
				"tenantResourceGroupName":  "tenant-rg",
				"creationTime":             now.Format(time.RFC3339),
				"sku":                      "Standard_LRS",
				"usageInBytes":             4294967296,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	deletedKey := azureStackAdminStorageAccountKey(subscriptionID, location, "account-deleted-001")
	if _, ok := s.azureStackAdminAccounts[deletedKey]; !ok {
		now := time.Now().UTC()
		s.azureStackAdminAccounts[deletedKey] = &azureStackAdminStorageAccount{
			SubscriptionID: subscriptionID,
			Location:       location,
			AccountID:      "account-deleted-001",
			Properties: map[string]any{
				"status":                   "Deleted",
				"deleted":                  true,
				"tenantViewId":             "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/tenant-rg/providers/Microsoft.Storage/storageAccounts/tenantdeleted001",
				"tenantSubscriptionId":     "22222222-2222-2222-2222-222222222222",
				"tenantStorageAccountName": "tenantdeleted001",
				"tenantResourceGroupName":  "tenant-rg",
				"creationTime":             now.Add(-48 * time.Hour).Format(time.RFC3339),
				"deletionTime":             now.Add(-24 * time.Hour).Format(time.RFC3339),
				"sku":                      "Standard_GRS",
				"usageInBytes":             2147483648,
			},
			CreatedAt: now.Add(-48 * time.Hour),
			UpdatedAt: now.Add(-24 * time.Hour),
		}
	}

	acquisitionKey := azureStackAdminAcquisitionKey(subscriptionID, location, "acquisition-001")
	if _, ok := s.azureStackAdminAcquisitions[acquisitionKey]; !ok {
		now := time.Now().UTC()
		s.azureStackAdminAcquisitions[acquisitionKey] = &azureStackAdminAcquisition{
			SubscriptionID: subscriptionID,
			Location:       location,
			Name:           "acquisition-001",
			Properties: map[string]any{
				"storageAccountId": "account-active-001",
				"containerName":    "vhds",
				"blobName":         "disk1.vhd",
				"status":           "Completed",
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
}

func (s *Server) azureStackAdminRegisterAsyncOperation(subscriptionID, location, action, resourceID string, payload map[string]any) *azureStackAdminAsyncOperation {
	now := time.Now().UTC()
	id := azureStackAdminStableID(subscriptionID + "::" + location + "::" + action + "::" + resourceID + "::" + now.Format(time.RFC3339Nano))[:12]
	op := &azureStackAdminAsyncOperation{
		SubscriptionID: subscriptionID,
		Location:       location,
		ID:             id,
		Name:           action + "-" + id,
		Action:         action,
		ResourceID:     resourceID,
		StatusCode:     http.StatusOK,
		Properties:     azureStackAdminMergeObjects(map[string]any{}, payload),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.azureStackAdminAsyncOperations[azureStackAdminAsyncOperationKey(subscriptionID, location, id)] = op
	return op
}

func azureStackAdminOperationsFixture() []map[string]any {
	ops := []string{
		"Microsoft.Storage.Admin/storageServices/read",
		"Microsoft.Storage.Admin/settings/read",
		"Microsoft.Storage.Admin/settings/write",
		"Microsoft.Storage.Admin/storageAccounts/read",
		"Microsoft.Storage.Admin/storageAccounts/undelete/action",
		"Microsoft.Storage.Admin/storageAccounts/reclaimStorageCapacity/action",
		"Microsoft.Storage.Admin/acquisitions/read",
		"Microsoft.Storage.Admin/asyncOperations/read",
	}
	out := make([]map[string]any, 0, len(ops))
	for _, name := range ops {
		out = append(out, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":  "Microsoft Azure Stack Admin",
				"resource":  "Storage Admin",
				"operation": name,
			},
		})
	}
	return out
}

func azureStackAdminSettingsFixture(settings *azureStackAdminSettings) map[string]any {
	return map[string]any{
		"id":         azureStackAdminSettingsID(settings.SubscriptionID, settings.Location),
		"name":       "default",
		"type":       "Microsoft.Storage.Admin/locations/settings",
		"location":   settings.Location,
		"systemData": azureStackAdminSystemData(settings.CreatedAt, settings.UpdatedAt),
		"properties": azureStackAdminMergeObjects(map[string]any{}, settings.Properties),
	}
}

func azureStackAdminServiceFixture(service *azureStackAdminService) map[string]any {
	return map[string]any{
		"id":         azureStackAdminServiceID(service.SubscriptionID, service.Name),
		"name":       service.Name,
		"type":       "Microsoft.Storage.Admin/storageServices",
		"location":   service.Location,
		"systemData": azureStackAdminSystemData(service.CreatedAt, service.UpdatedAt),
		"properties": azureStackAdminMergeObjects(map[string]any{}, service.Properties),
	}
}

func azureStackAdminStorageAccountFixture(account *azureStackAdminStorageAccount, summary bool) map[string]any {
	properties := azureStackAdminMergeObjects(map[string]any{}, account.Properties)
	if summary {
		properties = map[string]any{
			"status":                   properties["status"],
			"tenantStorageAccountName": properties["tenantStorageAccountName"],
			"usageInBytes":             properties["usageInBytes"],
		}
	}
	return map[string]any{
		"id":         azureStackAdminStorageAccountID(account.SubscriptionID, account.Location, account.AccountID),
		"name":       account.AccountID,
		"type":       "Microsoft.Storage.Admin/locations/storageAccounts",
		"location":   account.Location,
		"systemData": azureStackAdminSystemData(account.CreatedAt, account.UpdatedAt),
		"properties": properties,
	}
}

func azureStackAdminAcquisitionFixture(acquisition *azureStackAdminAcquisition) map[string]any {
	return map[string]any{
		"id":         azureStackAdminAcquisitionID(acquisition.SubscriptionID, acquisition.Location, acquisition.Name),
		"name":       acquisition.Name,
		"type":       "Microsoft.Storage.Admin/locations/acquisitions",
		"location":   acquisition.Location,
		"systemData": azureStackAdminSystemData(acquisition.CreatedAt, acquisition.UpdatedAt),
		"properties": azureStackAdminMergeObjects(map[string]any{}, acquisition.Properties),
	}
}

func azureStackAdminAsyncOperationFixture(op *azureStackAdminAsyncOperation) map[string]any {
	return map[string]any{
		"id":         azureStackAdminAsyncOperationID(op.SubscriptionID, op.Location, op.ID),
		"name":       op.ID,
		"type":       "Microsoft.Storage.Admin/locations/asyncOperations",
		"location":   op.Location,
		"status":     "Succeeded",
		"statusCode": op.StatusCode,
		"startTime":  op.CreatedAt.Format(time.RFC3339),
		"endTime":    op.UpdatedAt.Format(time.RFC3339),
		"properties": azureStackAdminMergeObjects(map[string]any{
			"action":     op.Action,
			"resourceId": op.ResourceID,
		}, op.Properties),
	}
}

func azureStackAdminSettingsKey(subscriptionID, location string) string {
	return strings.ToLower(subscriptionID + "::" + location)
}

func azureStackAdminServiceKey(subscriptionID, name string) string {
	return strings.ToLower(subscriptionID + "::" + name)
}

func azureStackAdminStorageAccountKey(subscriptionID, location, accountID string) string {
	return strings.ToLower(subscriptionID + "::" + location + "::" + accountID)
}

func azureStackAdminAcquisitionKey(subscriptionID, location, name string) string {
	return strings.ToLower(subscriptionID + "::" + location + "::" + name)
}

func azureStackAdminAsyncOperationKey(subscriptionID, location, id string) string {
	return strings.ToLower(subscriptionID + "::" + location + "::" + id)
}

func azureStackAdminServiceID(subscriptionID, name string) string {
	return fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage.Admin/storageServices/%s", subscriptionID, name)
}

func azureStackAdminSettingsID(subscriptionID, location string) string {
	return fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage.Admin/locations/%s/settings/default", subscriptionID, location)
}

func azureStackAdminStorageAccountID(subscriptionID, location, accountID string) string {
	return fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage.Admin/locations/%s/storageAccounts/%s", subscriptionID, location, accountID)
}

func azureStackAdminAcquisitionID(subscriptionID, location, name string) string {
	return fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage.Admin/locations/%s/acquisitions/%s", subscriptionID, location, name)
}

func azureStackAdminAsyncOperationID(subscriptionID, location, id string) string {
	return fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Storage.Admin/locations/%s/asyncOperations/%s", subscriptionID, location, id)
}

func azureStackAdminSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azureStackAdminNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("resource %q was not found for path %s", name, path),
		},
	}
}

func azureStackAdminReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
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

func azureStackAdminString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureStackAdminObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return azureStackAdminMergeObjects(map[string]any{}, typed)
}

func azureStackAdminMergeObjects(base, overlay map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	out := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		if existing, ok := out[key].(map[string]any); ok {
			if incoming, ok := value.(map[string]any); ok {
				out[key] = azureStackAdminMergeObjects(existing, incoming)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func azureStackAdminInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func azureStackAdminStableID(seed string) string {
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:])
}
