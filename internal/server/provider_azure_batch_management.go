package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

const azureBatchManagementAPIVersion = "2025-06-01"

var azureBatchManagementAccountNamePattern = regexp.MustCompile(`^[A-Za-z0-9]{3,24}$`)
var azureBatchManagementChildNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var azureBatchManagementVersionNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.-]{0,63}$`)

type azureBatchManagementAccount struct {
	SubscriptionID             string
	ResourceGroup              string
	Name                       string
	Location                   string
	Tags                       map[string]string
	Identity                   map[string]any
	Properties                 map[string]any
	AllowedAuthenticationModes []string
	PrimaryKey                 string
	SecondaryKey               string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	ETag                       string
}

type azureBatchManagementChildResource struct {
	SubscriptionID  string
	ResourceGroup   string
	AccountName     string
	ApplicationName string
	Collection      string
	Name            string
	Location        string
	Tags            map[string]string
	Properties      map[string]any
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ETag            string
}

func (s *Server) handleAzureBatchManagementRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsBatchManagementPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureBatchManagementAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Batch") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureBatchManagementOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureBatchManagementRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		switch {
		case len(tail) == 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "checkNameAvailability"):
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			s.handleAzureBatchManagementCheckNameAvailability(w, r, path, subscriptionID, tail[1])
			return true
		case len(tail) == 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "quotas"):
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			respondJSON(w, http.StatusOK, map[string]any{"value": azureBatchManagementQuotasFixture(subscriptionID, tail[1])})
			return true
		case len(tail) == 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "supportedVirtualMachineSkus"):
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			respondJSON(w, http.StatusOK, map[string]any{"value": azureBatchManagementSupportedVMsFixture(subscriptionID, tail[1])})
			return true
		case len(tail) == 1 && strings.EqualFold(tail[0], "batchAccounts"):
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return true
			}
			s.handleAzureBatchManagementListAccountsBySubscription(w, subscriptionID)
			return true
		default:
			respondAzureImplemented(w, path)
			return true
		}
	}

	if !strings.EqualFold(tail[0], "batchAccounts") {
		respondAzureImplemented(w, path)
		return true
	}
	s.handleAzureBatchManagementAccountsRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsBatchManagementPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Batch") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureBatchManagementRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "locations", "batchaccounts":
		return true
	default:
		return false
	}
}

func azureBatchManagementRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
	for i := 2; i+1 < len(segments); i++ {
		if strings.EqualFold(segments[i], "providers") && strings.EqualFold(segments[i+1], "Microsoft.Batch") {
			return subscriptionID, resourceGroup, segments[i+2:], true
		}
	}
	return "", "", nil, false
}

func azureBatchManagementAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureBatchManagementCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureBatchManagementReadJSONBody(w, r, path)
	if !ok {
		return
	}
	_ = subscriptionID
	name := strings.TrimSpace(azureBatchManagementString(payload["name"]))
	resourceType := strings.TrimSpace(azureBatchManagementString(payload["type"]))
	if name == "" || resourceType == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	if !strings.EqualFold(resourceType, "Microsoft.Batch/batchAccounts") {
		respondAzureInvalidRequest(w, path, "type must target Microsoft.Batch/batchAccounts")
		return
	}
	if !azureBatchManagementValidAccountName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "account name must be 3-24 alphanumeric characters",
		})
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, account := range s.azureBatchManagementAccounts {
		if strings.EqualFold(account.Location, location) && strings.EqualFold(account.Name, name) {
			respondJSON(w, http.StatusOK, map[string]any{
				"nameAvailable": false,
				"reason":        "AlreadyExists",
				"message":       "resource name is already in use",
			})
			return
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *Server) handleAzureBatchManagementAccountsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementListAccountsByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	accountName := strings.TrimSpace(tail[1])
	if !azureBatchManagementValidAccountName(accountName) {
		respondAzureInvalidRequest(w, path, "account name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBatchManagementAccountGet(w, path, subscriptionID, resourceGroup, accountName, false)
		case http.MethodHead:
			s.handleAzureBatchManagementAccountGet(w, path, subscriptionID, resourceGroup, accountName, true)
		case http.MethodPut:
			s.handleAzureBatchManagementAccountCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName)
		case http.MethodPatch:
			s.handleAzureBatchManagementAccountPatch(w, r, path, subscriptionID, resourceGroup, accountName)
		case http.MethodDelete:
			s.handleAzureBatchManagementAccountDelete(w, subscriptionID, resourceGroup, accountName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "listkeys":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureBatchManagementListKeys(w, path, subscriptionID, resourceGroup, accountName)
			return
		}
	case "regeneratekeys":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureBatchManagementRegenerateKeys(w, r, path, subscriptionID, resourceGroup, accountName)
			return
		}
	case "syncautostoragekeys":
		if len(tail) == 3 && r.Method == http.MethodPost {
			s.handleAzureBatchManagementSyncAutoStorageKeys(w, path, subscriptionID, resourceGroup, accountName)
			return
		}
	case "applications":
		s.handleAzureBatchManagementApplicationsRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[3:])
		return
	case "pools":
		s.handleAzureBatchManagementPoolsRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[3:])
		return
	case "privateendpointconnections":
		s.handleAzureBatchManagementPrivateEndpointConnectionsRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[3:])
		return
	case "privatelinkresources":
		s.handleAzureBatchManagementPrivateLinkResourcesRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[3:])
		return
	case "networksecurityperimeterconfigurations":
		s.handleAzureBatchManagementNSPRoute(w, r, path, subscriptionID, resourceGroup, accountName, tail[3:])
		return
	}

	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBatchManagementApplicationsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureBatchManagementAccountExists(subscriptionID, resourceGroup, accountName) {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementListApplications(w, subscriptionID, resourceGroup, accountName)
		return
	}
	applicationName := strings.TrimSpace(tail[0])
	if !azureBatchManagementValidChildName(applicationName) {
		respondAzureInvalidRequest(w, path, "application name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBatchManagementApplicationGet(w, path, subscriptionID, resourceGroup, accountName, applicationName)
		case http.MethodPut:
			s.handleAzureBatchManagementApplicationCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, applicationName)
		case http.MethodPatch:
			s.handleAzureBatchManagementApplicationPatch(w, r, path, subscriptionID, resourceGroup, accountName, applicationName)
		case http.MethodDelete:
			s.handleAzureBatchManagementApplicationDelete(w, subscriptionID, resourceGroup, accountName, applicationName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if !strings.EqualFold(tail[1], "versions") {
		respondAzureImplemented(w, path)
		return
	}
	s.handleAzureBatchManagementApplicationPackagesRoute(w, r, path, subscriptionID, resourceGroup, accountName, applicationName, tail[2:])
}

func (s *Server) handleAzureBatchManagementApplicationPackagesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, applicationName string, tail []string) {
	if !s.azureBatchManagementChildExists(subscriptionID, resourceGroup, accountName, "applications", applicationName, "") {
		azureBatchManagementRespondNotFound(w, path, "ApplicationNotFound", "application not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementListApplicationPackages(w, subscriptionID, resourceGroup, accountName, applicationName)
		return
	}
	versionName := strings.TrimSpace(tail[0])
	if !azureBatchManagementValidVersionName(versionName) {
		respondAzureInvalidRequest(w, path, "version name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBatchManagementApplicationPackageGet(w, path, subscriptionID, resourceGroup, accountName, applicationName, versionName)
		case http.MethodPut:
			s.handleAzureBatchManagementApplicationPackageCreate(w, r, subscriptionID, resourceGroup, accountName, applicationName, versionName)
		case http.MethodDelete:
			s.handleAzureBatchManagementApplicationPackageDelete(w, subscriptionID, resourceGroup, accountName, applicationName, versionName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "activate") && r.Method == http.MethodPost {
		s.handleAzureBatchManagementApplicationPackageActivate(w, path, subscriptionID, resourceGroup, accountName, applicationName, versionName)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBatchManagementPoolsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureBatchManagementAccountExists(subscriptionID, resourceGroup, accountName) {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementListPools(w, subscriptionID, resourceGroup, accountName)
		return
	}
	poolName := strings.TrimSpace(tail[0])
	if !azureBatchManagementValidChildName(poolName) {
		respondAzureInvalidRequest(w, path, "pool name is invalid")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureBatchManagementPoolGet(w, path, subscriptionID, resourceGroup, accountName, poolName)
		case http.MethodPut:
			s.handleAzureBatchManagementPoolCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, poolName)
		case http.MethodPatch:
			s.handleAzureBatchManagementPoolPatch(w, r, path, subscriptionID, resourceGroup, accountName, poolName)
		case http.MethodDelete:
			s.handleAzureBatchManagementPoolDelete(w, subscriptionID, resourceGroup, accountName, poolName)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 2 && r.Method == http.MethodPost {
		switch strings.ToLower(strings.TrimSpace(tail[1])) {
		case "disableautoscale":
			s.handleAzureBatchManagementPoolAction(w, path, subscriptionID, resourceGroup, accountName, poolName, "disableAutoScale")
			return
		case "stopresize":
			s.handleAzureBatchManagementPoolAction(w, path, subscriptionID, resourceGroup, accountName, poolName, "stopResize")
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBatchManagementPrivateEndpointConnectionsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureBatchManagementAccountExists(subscriptionID, resourceGroup, accountName) {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementListChildren(w, subscriptionID, resourceGroup, accountName, "privateEndpointConnections")
		return
	}
	name := strings.TrimSpace(tail[0])
	if !azureBatchManagementValidChildName(name) {
		respondAzureInvalidRequest(w, path, "private endpoint connection name is invalid")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureBatchManagementChildGet(w, path, subscriptionID, resourceGroup, accountName, "privateEndpointConnections", name, "")
	case http.MethodPatch:
		s.handleAzureBatchManagementPrivateEndpointConnectionPatch(w, r, path, subscriptionID, resourceGroup, accountName, name)
	case http.MethodDelete:
		s.handleAzureBatchManagementChildDelete(w, subscriptionID, resourceGroup, accountName, "privateEndpointConnections", name, "")
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureBatchManagementPrivateLinkResourcesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureBatchManagementAccountExists(subscriptionID, resourceGroup, accountName) {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": azureBatchManagementPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName)})
		return
	}
	for _, item := range azureBatchManagementPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName) {
		resource := item.(map[string]any)
		if strings.EqualFold(azureBatchManagementString(resource["name"]), tail[0]) {
			respondJSON(w, http.StatusOK, resource)
			return
		}
	}
	azureBatchManagementRespondNotFound(w, path, "PrivateLinkResourceNotFound", "private link resource not found")
}

func (s *Server) handleAzureBatchManagementNSPRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string, tail []string) {
	if !s.azureBatchManagementAccountExists(subscriptionID, resourceGroup, accountName) {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementListChildren(w, subscriptionID, resourceGroup, accountName, "networkSecurityPerimeterConfigurations")
		return
	}
	name := strings.TrimSpace(tail[0])
	if !azureBatchManagementValidChildName(name) {
		respondAzureInvalidRequest(w, path, "network security perimeter configuration name is invalid")
		return
	}
	if len(tail) == 1 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureBatchManagementChildGet(w, path, subscriptionID, resourceGroup, accountName, "networkSecurityPerimeterConfigurations", name, "")
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "reconcile") && r.Method == http.MethodPost {
		s.handleAzureBatchManagementNSPReconcile(w, path, subscriptionID, resourceGroup, accountName, name)
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureBatchManagementListAccountsBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]*azureBatchManagementAccount, 0)
	for _, account := range s.azureBatchManagementAccounts {
		if account.SubscriptionID == subscriptionID {
			items = append(items, account)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureBatchManagementAccountFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureBatchManagementListAccountsByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]*azureBatchManagementAccount, 0)
	for _, account := range s.azureBatchManagementAccounts {
		if account.SubscriptionID == subscriptionID && strings.EqualFold(account.ResourceGroup, resourceGroup) {
			items = append(items, account)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureBatchManagementAccountFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureBatchManagementAccountGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName string, head bool) {
	s.providerStorageMu.Lock()
	account := s.azureBatchManagementAccounts[azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)]
	s.providerStorageMu.Unlock()
	if account == nil {
		if head {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if head {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureBatchManagementAccountFixture(account))
}

func (s *Server) handleAzureBatchManagementAccountCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string) {
	_, payload, ok := azureBatchManagementReadJSONBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureBatchManagementString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)
	account := s.azureBatchManagementAccounts[key]
	status := http.StatusCreated
	createdAt := now
	primaryKey := azureBatchManagementStableToken(subscriptionID, resourceGroup, accountName, "primary")
	secondaryKey := azureBatchManagementStableToken(subscriptionID, resourceGroup, accountName, "secondary")
	if account != nil {
		status = http.StatusOK
		createdAt = account.CreatedAt
		primaryKey = account.PrimaryKey
		secondaryKey = account.SecondaryKey
	}
	account = &azureBatchManagementAccount{
		SubscriptionID:             subscriptionID,
		ResourceGroup:              resourceGroup,
		Name:                       accountName,
		Location:                   location,
		Tags:                       azureBatchManagementTags(payload["tags"]),
		Identity:                   azureBatchManagementObject(payload["identity"]),
		Properties:                 azureBatchManagementNormalizeAccountProperties(accountName, location, payload["properties"]),
		AllowedAuthenticationModes: azureBatchManagementAuthenticationModes(payload["properties"]),
		PrimaryKey:                 primaryKey,
		SecondaryKey:               secondaryKey,
		CreatedAt:                  createdAt,
		UpdatedAt:                  now,
		ETag:                       azureBatchManagementETag(subscriptionID, resourceGroup, accountName, now),
	}
	s.azureBatchManagementAccounts[key] = account
	s.azureBatchManagementSeedNSPConfigurationLocked(subscriptionID, resourceGroup, accountName, location, now)
	respondJSON(w, status, azureBatchManagementAccountFixture(account))
}

func (s *Server) handleAzureBatchManagementAccountPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string) {
	_, payload, ok := azureBatchManagementReadJSONBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)
	account := s.azureBatchManagementAccounts[key]
	if account == nil {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		account.Tags = azureBatchManagementTags(tags)
	}
	if props := azureBatchManagementObject(payload["properties"]); len(props) > 0 {
		account.Properties = azureBatchManagementMergeObjects(account.Properties, props)
		account.AllowedAuthenticationModes = azureBatchManagementAuthenticationModes(map[string]any{"allowedAuthenticationModes": account.Properties["allowedAuthenticationModes"]})
	}
	account.UpdatedAt = time.Now().UTC()
	account.ETag = azureBatchManagementETag(subscriptionID, resourceGroup, accountName, account.UpdatedAt)
	respondJSON(w, http.StatusOK, azureBatchManagementAccountFixture(account))
}

func (s *Server) handleAzureBatchManagementAccountDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	delete(s.azureBatchManagementAccounts, azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName))
	for key, child := range s.azureBatchManagementResources {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) && strings.EqualFold(child.AccountName, accountName) {
			delete(s.azureBatchManagementResources, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureBatchManagementListKeys(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName string) {
	s.providerStorageMu.Lock()
	account := s.azureBatchManagementAccounts[azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)]
	s.providerStorageMu.Unlock()
	if account == nil {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if !azureBatchManagementAllowsSharedKey(account.AllowedAuthenticationModes) {
		respondAzureInvalidRequest(w, path, "shared key authentication is disabled for this account")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"accountName":  account.Name,
		"primary":      account.PrimaryKey,
		"secondary":    account.SecondaryKey,
		"primaryKey":   account.PrimaryKey,
		"secondaryKey": account.SecondaryKey,
	})
}

func (s *Server) handleAzureBatchManagementRegenerateKeys(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName string) {
	_, payload, ok := azureBatchManagementReadJSONBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	account := s.azureBatchManagementAccounts[azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)]
	if account == nil {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	if !azureBatchManagementAllowsSharedKey(account.AllowedAuthenticationModes) {
		respondAzureInvalidRequest(w, path, "shared key authentication is disabled for this account")
		return
	}
	keyName := strings.ToLower(strings.TrimSpace(azureBatchManagementString(payload["keyName"])))
	if keyName == "" {
		keyName = strings.ToLower(strings.TrimSpace(azureBatchManagementString(payload["keyType"])))
	}
	if keyName == "" {
		respondAzureInvalidRequest(w, path, "keyName is required")
		return
	}
	switch keyName {
	case "primary":
		account.PrimaryKey = azureBatchManagementStableToken(subscriptionID, resourceGroup, accountName, "primary", time.Now().UTC().Format(time.RFC3339Nano))
	case "secondary":
		account.SecondaryKey = azureBatchManagementStableToken(subscriptionID, resourceGroup, accountName, "secondary", time.Now().UTC().Format(time.RFC3339Nano))
	default:
		respondAzureInvalidRequest(w, path, "keyName must be Primary or Secondary")
		return
	}
	account.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, map[string]any{
		"primaryKey":   account.PrimaryKey,
		"secondaryKey": account.SecondaryKey,
	})
}

func (s *Server) handleAzureBatchManagementSyncAutoStorageKeys(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName string) {
	s.providerStorageMu.Lock()
	account := s.azureBatchManagementAccounts[azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)]
	s.providerStorageMu.Unlock()
	if account == nil {
		azureBatchManagementRespondNotFound(w, path, "BatchAccountNotFound", "batch account not found")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":              "Succeeded",
		"batchAccountName":    accountName,
		"autoStorageSyncTime": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleAzureBatchManagementListApplications(w http.ResponseWriter, subscriptionID, resourceGroup, accountName string) {
	s.handleAzureBatchManagementListChildren(w, subscriptionID, resourceGroup, accountName, "applications")
}

func (s *Server) handleAzureBatchManagementApplicationGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, applicationName string) {
	s.handleAzureBatchManagementChildGet(w, path, subscriptionID, resourceGroup, accountName, "applications", applicationName, "")
}

func (s *Server) handleAzureBatchManagementApplicationCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, applicationName string) {
	s.handleAzureBatchManagementChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, "applications", applicationName, "", accountName)
}

func (s *Server) handleAzureBatchManagementApplicationPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, applicationName string) {
	s.handleAzureBatchManagementChildPatch(w, r, path, subscriptionID, resourceGroup, accountName, "applications", applicationName, "")
}

func (s *Server) handleAzureBatchManagementApplicationDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, applicationName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	delete(s.azureBatchManagementResources, azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "applications", applicationName, ""))
	for key, child := range s.azureBatchManagementResources {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) && strings.EqualFold(child.AccountName, accountName) &&
			strings.EqualFold(child.ApplicationName, applicationName) && strings.EqualFold(child.Collection, "versions") {
			delete(s.azureBatchManagementResources, key)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureBatchManagementListApplicationPackages(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, applicationName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]*azureBatchManagementChildResource, 0)
	for _, child := range s.azureBatchManagementResources {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.AccountName, accountName) && strings.EqualFold(child.ApplicationName, applicationName) &&
			strings.EqualFold(child.Collection, "versions") {
			items = append(items, child)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureBatchManagementChildFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureBatchManagementApplicationPackageGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, applicationName, versionName string) {
	s.handleAzureBatchManagementChildGet(w, path, subscriptionID, resourceGroup, accountName, "versions", versionName, applicationName)
}

func (s *Server) handleAzureBatchManagementApplicationPackageCreate(w http.ResponseWriter, r *http.Request, subscriptionID, resourceGroup, accountName, applicationName, versionName string) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "versions", versionName, applicationName)
	status := http.StatusCreated
	if s.azureBatchManagementResources[key] != nil {
		status = http.StatusOK
	}
	s.azureBatchManagementResources[key] = &azureBatchManagementChildResource{
		SubscriptionID:  subscriptionID,
		ResourceGroup:   resourceGroup,
		AccountName:     accountName,
		ApplicationName: applicationName,
		Collection:      "versions",
		Name:            versionName,
		Location:        s.azureBatchManagementAccountLocationLocked(subscriptionID, resourceGroup, accountName),
		Properties: map[string]any{
			"state":              "Pending",
			"format":             "zip",
			"storageURL":         "https://stackyard.local/upload/" + accountName + "/" + applicationName + "/" + versionName,
			"lastActivationTime": nil,
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureBatchManagementETag(subscriptionID, resourceGroup, accountName, now, versionName),
	}
	respondJSON(w, status, azureBatchManagementChildFixture(s.azureBatchManagementResources[key]))
}

func (s *Server) handleAzureBatchManagementApplicationPackageActivate(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, applicationName, versionName string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child := s.azureBatchManagementResources[azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "versions", versionName, applicationName)]
	if child == nil {
		azureBatchManagementRespondNotFound(w, path, "ApplicationPackageNotFound", "application package not found")
		return
	}
	child.Properties["state"] = "Active"
	child.Properties["lastActivationTime"] = time.Now().UTC().Format(time.RFC3339)
	child.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureBatchManagementChildFixture(child))
}

func (s *Server) handleAzureBatchManagementApplicationPackageDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, applicationName, versionName string) {
	s.providerStorageMu.Lock()
	delete(s.azureBatchManagementResources, azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "versions", versionName, applicationName))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureBatchManagementListPools(w http.ResponseWriter, subscriptionID, resourceGroup, accountName string) {
	s.handleAzureBatchManagementListChildren(w, subscriptionID, resourceGroup, accountName, "pools")
}

func (s *Server) handleAzureBatchManagementPoolGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, poolName string) {
	s.handleAzureBatchManagementChildGet(w, path, subscriptionID, resourceGroup, accountName, "pools", poolName, "")
}

func (s *Server) handleAzureBatchManagementPoolCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, poolName string) {
	s.handleAzureBatchManagementChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, "pools", poolName, "", accountName)
}

func (s *Server) handleAzureBatchManagementPoolPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, poolName string) {
	s.handleAzureBatchManagementChildPatch(w, r, path, subscriptionID, resourceGroup, accountName, "pools", poolName, "")
}

func (s *Server) handleAzureBatchManagementPoolDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, poolName string) {
	s.handleAzureBatchManagementChildDelete(w, subscriptionID, resourceGroup, accountName, "pools", poolName, "")
}

func (s *Server) handleAzureBatchManagementPoolAction(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, poolName, action string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child := s.azureBatchManagementResources[azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "pools", poolName, "")]
	if child == nil {
		azureBatchManagementRespondNotFound(w, path, "PoolNotFound", "pool not found")
		return
	}
	switch action {
	case "disableAutoScale":
		child.Properties["enableAutoScale"] = false
		child.Properties["allocationState"] = "Steady"
	case "stopResize":
		child.Properties["resizeOperationStatus"] = "Stopped"
	}
	child.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureBatchManagementChildFixture(child))
}

func (s *Server) handleAzureBatchManagementPrivateEndpointConnectionPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, name string) {
	s.handleAzureBatchManagementChildCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, accountName, "privateEndpointConnections", name, "", accountName)
}

func (s *Server) handleAzureBatchManagementNSPReconcile(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	child := s.azureBatchManagementResources[azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "networkSecurityPerimeterConfigurations", name, "")]
	if child == nil {
		azureBatchManagementRespondNotFound(w, path, "NetworkSecurityPerimeterConfigurationNotFound", "network security perimeter configuration not found")
		return
	}
	child.Properties["reconciliationState"] = "Succeeded"
	child.Properties["lastReconciledAt"] = time.Now().UTC().Format(time.RFC3339)
	child.UpdatedAt = time.Now().UTC()
	respondJSON(w, http.StatusOK, azureBatchManagementChildFixture(child))
}

func (s *Server) handleAzureBatchManagementListChildren(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, collection string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]*azureBatchManagementChildResource, 0)
	for _, child := range s.azureBatchManagementResources {
		if child.SubscriptionID == subscriptionID && strings.EqualFold(child.ResourceGroup, resourceGroup) &&
			strings.EqualFold(child.AccountName, accountName) && strings.EqualFold(child.Collection, collection) {
			items = append(items, child)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureBatchManagementChildFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureBatchManagementChildGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, accountName, collection, name, applicationName string) {
	s.providerStorageMu.Lock()
	child := s.azureBatchManagementResources[azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, collection, name, applicationName)]
	s.providerStorageMu.Unlock()
	if child == nil {
		code := map[string]string{
			"applications":                           "ApplicationNotFound",
			"versions":                               "ApplicationPackageNotFound",
			"pools":                                  "PoolNotFound",
			"privateEndpointConnections":             "PrivateEndpointConnectionNotFound",
			"networkSecurityPerimeterConfigurations": "NetworkSecurityPerimeterConfigurationNotFound",
		}[collection]
		if code == "" {
			code = "ResourceNotFound"
		}
		azureBatchManagementRespondNotFound(w, path, code, "resource not found")
		return
	}
	respondJSON(w, http.StatusOK, azureBatchManagementChildFixture(child))
}

func (s *Server) handleAzureBatchManagementChildCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, collection, name, applicationName, defaultLocationRef string) {
	_, payload, ok := azureBatchManagementReadJSONBody(w, r, path)
	if !ok {
		return
	}
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, collection, name, applicationName)
	status := http.StatusCreated
	existing := s.azureBatchManagementResources[key]
	location := s.azureBatchManagementAccountLocationLocked(subscriptionID, resourceGroup, defaultLocationRef)
	if strings.TrimSpace(azureBatchManagementString(payload["location"])) != "" {
		location = strings.TrimSpace(azureBatchManagementString(payload["location"]))
	}
	if existing != nil {
		status = http.StatusOK
		location = existing.Location
		if strings.TrimSpace(azureBatchManagementString(payload["location"])) != "" {
			location = strings.TrimSpace(azureBatchManagementString(payload["location"]))
		}
	}
	properties := azureBatchManagementNormalizeChildProperties(collection, name, payload["properties"])
	if existing != nil {
		properties = azureBatchManagementMergeObjects(existing.Properties, properties)
	}
	child := &azureBatchManagementChildResource{
		SubscriptionID:  subscriptionID,
		ResourceGroup:   resourceGroup,
		AccountName:     accountName,
		ApplicationName: applicationName,
		Collection:      collection,
		Name:            name,
		Location:        location,
		Tags:            azureBatchManagementTags(payload["tags"]),
		Properties:      properties,
		CreatedAt:       now,
		UpdatedAt:       now,
		ETag:            azureBatchManagementETag(subscriptionID, resourceGroup, accountName, now, collection, name),
	}
	if existing != nil {
		child.CreatedAt = existing.CreatedAt
		if len(child.Tags) == 0 {
			child.Tags = existing.Tags
		}
	}
	s.azureBatchManagementResources[key] = child
	respondJSON(w, status, azureBatchManagementChildFixture(child))
}

func (s *Server) handleAzureBatchManagementChildPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, accountName, collection, name, applicationName string) {
	_, payload, ok := azureBatchManagementReadJSONBody(w, r, path)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, collection, name, applicationName)
	child := s.azureBatchManagementResources[key]
	if child == nil {
		azureBatchManagementRespondNotFound(w, path, "ResourceNotFound", "resource not found")
		return
	}
	if tags := azureBatchManagementTags(payload["tags"]); len(tags) > 0 {
		child.Tags = tags
	}
	if props := azureBatchManagementObject(payload["properties"]); len(props) > 0 {
		child.Properties = azureBatchManagementMergeObjects(child.Properties, props)
	}
	child.UpdatedAt = time.Now().UTC()
	child.ETag = azureBatchManagementETag(subscriptionID, resourceGroup, accountName, child.UpdatedAt, collection, name)
	respondJSON(w, http.StatusOK, azureBatchManagementChildFixture(child))
}

func (s *Server) handleAzureBatchManagementChildDelete(w http.ResponseWriter, subscriptionID, resourceGroup, accountName, collection, name, applicationName string) {
	s.providerStorageMu.Lock()
	delete(s.azureBatchManagementResources, azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, collection, name, applicationName))
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) azureBatchManagementAccountExists(subscriptionID, resourceGroup, accountName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureBatchManagementAccounts[azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)] != nil
}

func (s *Server) azureBatchManagementChildExists(subscriptionID, resourceGroup, accountName, collection, name, applicationName string) bool {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureBatchManagementResources[azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, collection, name, applicationName)] != nil
}

func (s *Server) azureBatchManagementAccountLocationLocked(subscriptionID, resourceGroup, accountName string) string {
	account := s.azureBatchManagementAccounts[azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName)]
	if account == nil {
		return "eastus"
	}
	return account.Location
}

func (s *Server) azureBatchManagementSeedNSPConfigurationLocked(subscriptionID, resourceGroup, accountName, location string, now time.Time) {
	key := azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, "networkSecurityPerimeterConfigurations", "default", "")
	if s.azureBatchManagementResources[key] != nil {
		return
	}
	s.azureBatchManagementResources[key] = &azureBatchManagementChildResource{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		AccountName:    accountName,
		Collection:     "networkSecurityPerimeterConfigurations",
		Name:           "default",
		Location:       location,
		Properties: map[string]any{
			"provisioningState":   "Succeeded",
			"reconciliationState": "Pending",
		},
		CreatedAt: now,
		UpdatedAt: now,
		ETag:      azureBatchManagementETag(subscriptionID, resourceGroup, accountName, now, "nsp"),
	}
}

func azureBatchManagementAccountFixture(account *azureBatchManagementAccount) map[string]any {
	return map[string]any{
		"id":       azureBatchManagementAccountID(account.SubscriptionID, account.ResourceGroup, account.Name),
		"name":     account.Name,
		"type":     "Microsoft.Batch/batchAccounts",
		"location": account.Location,
		"etag":     account.ETag,
		"tags":     azureBatchManagementStringMapAny(account.Tags),
		"identity": account.Identity,
		"properties": azureBatchManagementMergeObjects(account.Properties, map[string]any{
			"accountEndpoint":            fmt.Sprintf("%s.%s.batch.azure.com", strings.ToLower(account.Name), account.Location),
			"provisioningState":          "Succeeded",
			"allowedAuthenticationModes": append([]any(nil), azureBatchManagementStringsToAny(account.AllowedAuthenticationModes)...),
		}),
		"systemData": azureBatchManagementSystemData(account.CreatedAt, account.UpdatedAt),
	}
}

func azureBatchManagementChildFixture(child *azureBatchManagementChildResource) map[string]any {
	return map[string]any{
		"id":       azureBatchManagementChildID(child),
		"name":     child.Name,
		"type":     azureBatchManagementChildType(child.Collection),
		"location": child.Location,
		"etag":     child.ETag,
		"tags":     azureBatchManagementStringMapAny(child.Tags),
		"properties": azureBatchManagementMergeObjects(child.Properties, map[string]any{
			"provisioningState": "Succeeded",
		}),
		"systemData": azureBatchManagementSystemData(child.CreatedAt, child.UpdatedAt),
	}
}

func azureBatchManagementOperationsFixture() []any {
	ops := []string{
		"Microsoft.Batch/operations/read",
		"Microsoft.Batch/locations/checkNameAvailability/action",
		"Microsoft.Batch/locations/quotas/read",
		"Microsoft.Batch/locations/supportedVirtualMachineSkus/read",
		"Microsoft.Batch/batchAccounts/read",
		"Microsoft.Batch/batchAccounts/write",
		"Microsoft.Batch/batchAccounts/delete",
		"Microsoft.Batch/batchAccounts/listKeys/action",
		"Microsoft.Batch/batchAccounts/regenerateKeys/action",
		"Microsoft.Batch/batchAccounts/syncAutoStorageKeys/action",
		"Microsoft.Batch/batchAccounts/applications/read",
		"Microsoft.Batch/batchAccounts/applications/write",
		"Microsoft.Batch/batchAccounts/applications/delete",
		"Microsoft.Batch/batchAccounts/applications/versions/read",
		"Microsoft.Batch/batchAccounts/applications/versions/write",
		"Microsoft.Batch/batchAccounts/applications/versions/delete",
		"Microsoft.Batch/batchAccounts/applications/versions/activate/action",
		"Microsoft.Batch/batchAccounts/pools/read",
		"Microsoft.Batch/batchAccounts/pools/write",
		"Microsoft.Batch/batchAccounts/pools/delete",
		"Microsoft.Batch/batchAccounts/pools/disableAutoScale/action",
		"Microsoft.Batch/batchAccounts/pools/stopResize/action",
		"Microsoft.Batch/batchAccounts/privateEndpointConnections/read",
		"Microsoft.Batch/batchAccounts/privateEndpointConnections/write",
		"Microsoft.Batch/batchAccounts/privateEndpointConnections/delete",
		"Microsoft.Batch/batchAccounts/privateLinkResources/read",
		"Microsoft.Batch/batchAccounts/networkSecurityPerimeterConfigurations/read",
		"Microsoft.Batch/batchAccounts/networkSecurityPerimeterConfigurations/reconcile/action",
	}
	value := make([]any, 0, len(ops))
	for _, name := range ops {
		value = append(value, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":  "Microsoft Batch",
				"resource":  "Batch",
				"operation": name,
			},
		})
	}
	return value
}

func azureBatchManagementQuotasFixture(subscriptionID, location string) []any {
	return []any{
		map[string]any{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Batch/locations/" + location + "/quotas/dedicatedCoreQuota",
			"name": "dedicatedCoreQuota",
			"type": "Microsoft.Batch/locations/quotas",
			"properties": map[string]any{
				"quota":        200,
				"usage":        12,
				"resourceName": "dedicatedCoreQuota",
				"location":     location,
			},
		},
		map[string]any{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Batch/locations/" + location + "/quotas/lowPriorityCoreQuota",
			"name": "lowPriorityCoreQuota",
			"type": "Microsoft.Batch/locations/quotas",
			"properties": map[string]any{
				"quota":        500,
				"usage":        24,
				"resourceName": "lowPriorityCoreQuota",
				"location":     location,
			},
		},
	}
}

func azureBatchManagementSupportedVMsFixture(subscriptionID, location string) []any {
	return []any{
		map[string]any{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Batch/locations/" + location + "/supportedVirtualMachineSkus/Standard_D2s_v3",
			"name": "Standard_D2s_v3",
			"type": "Microsoft.Batch/locations/supportedVirtualMachineSkus",
			"capabilities": []any{
				map[string]any{"name": "vCPUs", "value": "2"},
				map[string]any{"name": "MemoryGB", "value": "8"},
			},
		},
		map[string]any{
			"id":   "/subscriptions/" + subscriptionID + "/providers/Microsoft.Batch/locations/" + location + "/supportedVirtualMachineSkus/Standard_D4s_v3",
			"name": "Standard_D4s_v3",
			"type": "Microsoft.Batch/locations/supportedVirtualMachineSkus",
			"capabilities": []any{
				map[string]any{"name": "vCPUs", "value": "4"},
				map[string]any{"name": "MemoryGB", "value": "16"},
			},
		},
	}
}

func azureBatchManagementPrivateLinkResourcesFixture(subscriptionID, resourceGroup, accountName string) []any {
	base := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Batch/batchAccounts/" + accountName + "/privateLinkResources"
	return []any{
		map[string]any{
			"id":   base + "/batchAccount",
			"name": "batchAccount",
			"type": "Microsoft.Batch/batchAccounts/privateLinkResources",
			"properties": map[string]any{
				"groupId":           "batchAccount",
				"requiredMembers":   []any{"batchAccount"},
				"requiredZoneNames": []any{"privatelink.batch.azure.com"},
			},
		},
	}
}

func azureBatchManagementNormalizeAccountProperties(accountName, location string, value any) map[string]any {
	props := azureBatchManagementObject(value)
	if len(props) == 0 {
		props = map[string]any{}
	}
	if _, ok := props["poolAllocationMode"]; !ok {
		props["poolAllocationMode"] = "BatchService"
	}
	if _, ok := props["publicNetworkAccess"]; !ok {
		props["publicNetworkAccess"] = "Enabled"
	}
	if _, ok := props["autoStorage"]; !ok {
		props["autoStorage"] = map[string]any{
			"storageAccountId": "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/mock-rg/providers/Microsoft.Storage/storageAccounts/" + strings.ToLower(accountName) + "auto",
		}
	}
	if _, ok := props["accountEndpoint"]; !ok {
		props["accountEndpoint"] = fmt.Sprintf("%s.%s.batch.azure.com", strings.ToLower(accountName), location)
	}
	return props
}

func azureBatchManagementNormalizeChildProperties(collection, name string, value any) map[string]any {
	props := azureBatchManagementObject(value)
	if len(props) == 0 {
		props = map[string]any{}
	}
	switch collection {
	case "applications":
		if _, ok := props["displayName"]; !ok {
			props["displayName"] = name
		}
		if _, ok := props["allowUpdates"]; !ok {
			props["allowUpdates"] = true
		}
	case "pools":
		if _, ok := props["vmSize"]; !ok {
			props["vmSize"] = "Standard_D2s_v3"
		}
		if _, ok := props["deploymentConfiguration"]; !ok {
			props["deploymentConfiguration"] = map[string]any{
				"vmConfiguration": map[string]any{
					"imageReference": map[string]any{
						"publisher": "Canonical",
						"offer":     "0001-com-ubuntu-server-jammy",
						"sku":       "22_04-lts",
						"version":   "latest",
					},
					"nodeAgentSkuId": "batch.node.ubuntu 22.04",
				},
			}
		}
		if _, ok := props["enableAutoScale"]; !ok {
			props["enableAutoScale"] = true
		}
		if _, ok := props["allocationState"]; !ok {
			props["allocationState"] = "Steady"
		}
		if _, ok := props["resizeOperationStatus"]; !ok {
			props["resizeOperationStatus"] = "None"
		}
	case "privateEndpointConnections":
		if _, ok := props["privateLinkServiceConnectionState"]; !ok {
			props["privateLinkServiceConnectionState"] = map[string]any{
				"status":      "Approved",
				"description": "Approved by Stackyard",
			}
		}
	case "networkSecurityPerimeterConfigurations":
		if _, ok := props["reconciliationState"]; !ok {
			props["reconciliationState"] = "Pending"
		}
	}
	return props
}

func azureBatchManagementAuthenticationModes(value any) []string {
	props := azureBatchManagementObject(value)
	if raw, ok := props["allowedAuthenticationModes"]; ok {
		switch typed := raw.(type) {
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				if text := strings.TrimSpace(azureBatchManagementString(item)); text != "" {
					out = append(out, text)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			if len(typed) > 0 {
				return typed
			}
		case string:
			if strings.TrimSpace(typed) != "" {
				return []string{typed}
			}
		}
	}
	return []string{"SharedKey"}
}

func azureBatchManagementAllowsSharedKey(modes []string) bool {
	if len(modes) == 0 {
		return true
	}
	for _, mode := range modes {
		if strings.EqualFold(strings.TrimSpace(mode), "SharedKey") {
			return true
		}
	}
	return false
}

func azureBatchManagementAccountKey(subscriptionID, resourceGroup, accountName string) string {
	return strings.ToLower(subscriptionID) + "|" + strings.ToLower(resourceGroup) + "|" + strings.ToLower(accountName)
}

func azureBatchManagementChildKey(subscriptionID, resourceGroup, accountName, collection, name, applicationName string) string {
	return strings.ToLower(subscriptionID) + "|" + strings.ToLower(resourceGroup) + "|" + strings.ToLower(accountName) + "|" + strings.ToLower(collection) + "|" + strings.ToLower(applicationName) + "|" + strings.ToLower(name)
}

func azureBatchManagementAccountID(subscriptionID, resourceGroup, accountName string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Batch/batchAccounts/" + accountName
}

func azureBatchManagementChildID(child *azureBatchManagementChildResource) string {
	base := azureBatchManagementAccountID(child.SubscriptionID, child.ResourceGroup, child.AccountName)
	switch child.Collection {
	case "applications":
		return base + "/applications/" + child.Name
	case "versions":
		return base + "/applications/" + child.ApplicationName + "/versions/" + child.Name
	case "pools":
		return base + "/pools/" + child.Name
	case "privateEndpointConnections":
		return base + "/privateEndpointConnections/" + child.Name
	case "networkSecurityPerimeterConfigurations":
		return base + "/networkSecurityPerimeterConfigurations/" + child.Name
	default:
		return base + "/" + child.Collection + "/" + child.Name
	}
}

func azureBatchManagementChildType(collection string) string {
	switch collection {
	case "applications":
		return "Microsoft.Batch/batchAccounts/applications"
	case "versions":
		return "Microsoft.Batch/batchAccounts/applications/versions"
	case "pools":
		return "Microsoft.Batch/batchAccounts/pools"
	case "privateEndpointConnections":
		return "Microsoft.Batch/batchAccounts/privateEndpointConnections"
	case "networkSecurityPerimeterConfigurations":
		return "Microsoft.Batch/batchAccounts/networkSecurityPerimeterConfigurations"
	default:
		return "Microsoft.Batch/batchAccounts/" + collection
	}
}

func azureBatchManagementValidAccountName(value string) bool {
	return azureBatchManagementAccountNamePattern.MatchString(strings.TrimSpace(value))
}

func azureBatchManagementValidChildName(value string) bool {
	return azureBatchManagementChildNamePattern.MatchString(strings.TrimSpace(value))
}

func azureBatchManagementValidVersionName(value string) bool {
	return azureBatchManagementVersionNamePattern.MatchString(strings.TrimSpace(value))
}

func azureBatchManagementStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func azureBatchManagementETag(parts ...any) string {
	pieces := make([]string, 0, len(parts))
	for _, part := range parts {
		pieces = append(pieces, fmt.Sprint(part))
	}
	return "W/\"" + azureBatchManagementStableToken(pieces...)[:16] + "\""
}

func azureBatchManagementRespondNotFound(w http.ResponseWriter, path, code, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerAzure,
		"path":     path,
	})
}

func azureBatchManagementReadJSONBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureBatchManagementString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func azureBatchManagementObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	switch typed := value.(type) {
	case map[string]any:
		return azureBatchManagementCloneObject(typed)
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return map[string]any{}
	}
}

func azureBatchManagementTags(value any) map[string]string {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			out[key] = azureBatchManagementString(item)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return map[string]string{}
	}
}

func azureBatchManagementMergeObjects(base, overlay map[string]any) map[string]any {
	merged := azureBatchManagementCloneObject(base)
	for key, value := range overlay {
		if current, ok := merged[key].(map[string]any); ok {
			if next, ok := value.(map[string]any); ok {
				merged[key] = azureBatchManagementMergeObjects(current, next)
				continue
			}
		}
		merged[key] = value
	}
	return merged
}

func azureBatchManagementCloneObject(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		switch typed := item.(type) {
		case map[string]any:
			out[key] = azureBatchManagementCloneObject(typed)
		case []any:
			out[key] = append([]any(nil), typed...)
		default:
			out[key] = item
		}
	}
	return out
}

func azureBatchManagementSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":      createdAt.Format(time.RFC3339),
		"lastModifiedAt": updatedAt.Format(time.RFC3339),
	}
}

func azureBatchManagementStringMapAny(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func azureBatchManagementStringsToAny(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
