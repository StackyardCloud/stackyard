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

type azureActiveDirectoryB2CTenant struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	SKU            map[string]any
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type azureActiveDirectoryGuestUsage struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *Server) handleAzureActiveDirectoryRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if !azureIsActiveDirectoryPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureActiveDirectoryAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureActiveDirectory") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureActiveDirectoryOperationsFixture()})
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureActiveDirectoryRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability":
		if resourceGroup == "" && len(tail) == 1 && r.Method == http.MethodPost {
			s.handleAzureActiveDirectoryCheckNameAvailability(w, r, path, subscriptionID)
			return true
		}
	case "b2cdirectories":
		s.handleAzureActiveDirectoryB2CTenants(w, r, path, subscriptionID, resourceGroup, tail[1:])
		return true
	case "guestusages":
		s.handleAzureActiveDirectoryGuestUsages(w, r, path, subscriptionID, resourceGroup, tail[1:])
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureActiveDirectoryCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_, payload, ok := azureActiveDirectoryReadRequestBody(w, r, path)
	if !ok {
		return
	}

	name := strings.TrimSpace(azureActiveDirectoryString(payload["name"]))
	countryCode := strings.TrimSpace(strings.ToUpper(azureActiveDirectoryString(payload["countryCode"])))
	if name == "" {
		respondAzureInvalidRequest(w, path, "name is required")
		return
	}
	if countryCode == "" {
		respondAzureInvalidRequest(w, path, "countryCode is required")
		return
	}

	available, reason, message := s.azureActiveDirectoryNameAvailability(name)
	respondJSON(w, http.StatusOK, map[string]any{
		"nameAvailable": available,
		"reason":        reason,
		"message":       message,
		"subscription":  subscriptionID,
	})
}

func (s *Server) handleAzureActiveDirectoryB2CTenants(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	switch {
	case len(tail) == 0 && resourceGroup == "" && r.Method == http.MethodGet:
		s.handleAzureActiveDirectoryB2CTenantListBySubscription(w, subscriptionID)
	case len(tail) == 0 && resourceGroup != "" && r.Method == http.MethodGet:
		s.handleAzureActiveDirectoryB2CTenantListByResourceGroup(w, subscriptionID, resourceGroup)
	case len(tail) == 1:
		name := strings.TrimSpace(tail[0])
		if name == "" {
			respondAzureInvalidRequest(w, path, "b2c tenant name is required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.handleAzureActiveDirectoryB2CTenantGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureActiveDirectoryB2CTenantGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureActiveDirectoryB2CTenantCreate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureActiveDirectoryB2CTenantUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureActiveDirectoryB2CTenantDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureActiveDirectoryGuestUsages(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	switch {
	case len(tail) == 0 && resourceGroup == "" && r.Method == http.MethodGet:
		s.handleAzureActiveDirectoryGuestUsageListBySubscription(w, subscriptionID)
	case len(tail) == 0 && resourceGroup != "" && r.Method == http.MethodGet:
		s.handleAzureActiveDirectoryGuestUsageListByResourceGroup(w, subscriptionID, resourceGroup)
	case len(tail) == 1:
		name := strings.TrimSpace(tail[0])
		if name == "" {
			respondAzureInvalidRequest(w, path, "guest usage name is required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			s.handleAzureActiveDirectoryGuestUsageGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureActiveDirectoryGuestUsageGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureActiveDirectoryGuestUsageCreate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureActiveDirectoryGuestUsageUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureActiveDirectoryGuestUsageDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureActiveDirectoryB2CTenantCreate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureActiveDirectoryReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureActiveDirectoryString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	sku := azureActiveDirectoryObject(payload["sku"])
	if len(sku) == 0 {
		respondAzureInvalidRequest(w, path, "sku is required")
		return
	}
	properties := azureActiveDirectoryObject(payload["properties"])
	createTenantProperties := azureActiveDirectoryObject(properties["createTenantProperties"])
	if strings.TrimSpace(azureActiveDirectoryString(createTenantProperties["countryCode"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.createTenantProperties.countryCode is required")
		return
	}
	if strings.TrimSpace(azureActiveDirectoryString(createTenantProperties["displayName"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.createTenantProperties.displayName is required")
		return
	}

	available, reason, message := s.azureActiveDirectoryNameAvailability(name)
	if !available {
		azureActiveDirectoryConflict(w, path, reason, message)
		return
	}

	now := time.Now().UTC()
	record := &azureActiveDirectoryB2CTenant{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           name,
		Location:       location,
		Tags:           azureActiveDirectoryStringMap(payload["tags"]),
		SKU:            sku,
		Properties: azureActiveDirectoryMergeObject(map[string]any{
			"tenantId": azureActiveDirectoryTenantUUID(subscriptionID, resourceGroup, name),
			"billingConfig": map[string]any{
				"billingType": "MAU",
			},
			"createTenantProperties": createTenantProperties,
			"provisioningState":      "Succeeded",
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureActiveDirectoryB2CTenants[azureActiveDirectoryStorageKey(azureActiveDirectoryB2CTenantID(subscriptionID, resourceGroup, name))] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureActiveDirectoryB2CTenantFixture(record))
}

func (s *Server) handleAzureActiveDirectoryB2CTenantUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureActiveDirectoryReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureActiveDirectoryStorageKey(azureActiveDirectoryB2CTenantID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureActiveDirectoryB2CTenants[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureActiveDirectoryNotFound(w, path, "B2CTenantNotFound", "b2c tenant not found")
		return
	}
	if tags := azureActiveDirectoryStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if sku := azureActiveDirectoryObject(payload["sku"]); len(sku) != 0 {
		record.SKU = azureActiveDirectoryMergeObject(record.SKU, sku)
	}
	if properties := azureActiveDirectoryObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureActiveDirectoryMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureActiveDirectoryB2CTenantFixture(record))
}

func (s *Server) handleAzureActiveDirectoryB2CTenantGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	key := azureActiveDirectoryStorageKey(azureActiveDirectoryB2CTenantID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureActiveDirectoryB2CTenants[key]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureActiveDirectoryNotFound(w, path, "B2CTenantNotFound", "b2c tenant not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureActiveDirectoryB2CTenantFixture(record))
}

func (s *Server) handleAzureActiveDirectoryB2CTenantDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureActiveDirectoryStorageKey(azureActiveDirectoryB2CTenantID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureActiveDirectoryB2CTenants[key]
	delete(s.azureActiveDirectoryB2CTenants, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureActiveDirectoryB2CTenantFixture(record))
}

func (s *Server) handleAzureActiveDirectoryB2CTenantListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureActiveDirectoryB2CTenants {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) {
			items = append(items, azureActiveDirectoryB2CTenantFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureActiveDirectoryString(items[i]["id"])) < strings.ToLower(azureActiveDirectoryString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureActiveDirectoryB2CTenantListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureActiveDirectoryB2CTenants {
		if record.SubscriptionID == subscriptionID {
			items = append(items, azureActiveDirectoryB2CTenantFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureActiveDirectoryString(items[i]["id"])) < strings.ToLower(azureActiveDirectoryString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureActiveDirectoryGuestUsageCreate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureActiveDirectoryReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureActiveDirectoryString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureActiveDirectoryObject(payload["properties"])
	if strings.TrimSpace(azureActiveDirectoryString(properties["tenantId"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.tenantId is required")
		return
	}

	now := time.Now().UTC()
	record := &azureActiveDirectoryGuestUsage{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           name,
		Location:       location,
		Tags:           azureActiveDirectoryStringMap(payload["tags"]),
		Properties: azureActiveDirectoryMergeObject(map[string]any{
			"provisioningState": "Succeeded",
		}, properties),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureActiveDirectoryGuestUsages[azureActiveDirectoryStorageKey(azureActiveDirectoryGuestUsageID(subscriptionID, resourceGroup, name))] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureActiveDirectoryGuestUsageFixture(record))
}

func (s *Server) handleAzureActiveDirectoryGuestUsageUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureActiveDirectoryReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureActiveDirectoryStorageKey(azureActiveDirectoryGuestUsageID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureActiveDirectoryGuestUsages[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureActiveDirectoryNotFound(w, path, "GuestUsageNotFound", "guest usage not found")
		return
	}
	if tags := azureActiveDirectoryStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if properties := azureActiveDirectoryObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureActiveDirectoryMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureActiveDirectoryGuestUsageFixture(record))
}

func (s *Server) handleAzureActiveDirectoryGuestUsageGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	key := azureActiveDirectoryStorageKey(azureActiveDirectoryGuestUsageID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureActiveDirectoryGuestUsages[key]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureActiveDirectoryNotFound(w, path, "GuestUsageNotFound", "guest usage not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureActiveDirectoryGuestUsageFixture(record))
}

func (s *Server) handleAzureActiveDirectoryGuestUsageDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureActiveDirectoryStorageKey(azureActiveDirectoryGuestUsageID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureActiveDirectoryGuestUsages[key]
	delete(s.azureActiveDirectoryGuestUsages, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureActiveDirectoryGuestUsageFixture(record))
}

func (s *Server) handleAzureActiveDirectoryGuestUsageListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureActiveDirectoryGuestUsages {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) {
			items = append(items, azureActiveDirectoryGuestUsageFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureActiveDirectoryString(items[i]["id"])) < strings.ToLower(azureActiveDirectoryString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureActiveDirectoryGuestUsageListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureActiveDirectoryGuestUsages {
		if record.SubscriptionID == subscriptionID {
			items = append(items, azureActiveDirectoryGuestUsageFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureActiveDirectoryString(items[i]["id"])) < strings.ToLower(azureActiveDirectoryString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) azureActiveDirectoryNameAvailability(name string) (bool, string, string) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if !strings.HasSuffix(normalized, ".onmicrosoft.com") || strings.Contains(normalized, " ") {
		return false, "Invalid", "The tenant name must be an .onmicrosoft.com domain without spaces."
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, record := range s.azureActiveDirectoryB2CTenants {
		if strings.EqualFold(record.Name, normalized) {
			return false, "AlreadyExists", "The tenant name is already in use."
		}
	}
	return true, "", ""
}

func azureIsActiveDirectoryPath(segments []string) bool {
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureActiveDirectory") {
		return true
	}
	if len(segments) < 2 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.AzureActiveDirectory") {
			return true
		}
	}
	return false
}

func azureActiveDirectoryAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureActiveDirectoryRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if len(segments) >= 4 && strings.EqualFold(segments[2], "providers") && strings.EqualFold(segments[3], "Microsoft.AzureActiveDirectory") {
		return subscriptionID, "", segments[4:], true
	}
	if len(segments) >= 6 &&
		strings.EqualFold(segments[2], "resourceGroups") &&
		strings.TrimSpace(segments[3]) != "" &&
		strings.EqualFold(segments[4], "providers") &&
		strings.EqualFold(segments[5], "Microsoft.AzureActiveDirectory") {
		return subscriptionID, strings.TrimSpace(segments[3]), segments[6:], true
	}
	return "", "", nil, false
}

func azureActiveDirectoryOperationsFixture() []map[string]any {
	return []map[string]any{
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/b2cDirectories/read", "Read B2C tenant"),
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/b2cDirectories/write", "Create or update B2C tenant"),
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/b2cDirectories/delete", "Delete B2C tenant"),
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/checkNameAvailability/action", "Check B2C tenant name availability"),
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/guestUsages/read", "Read guest usage"),
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/guestUsages/write", "Create or update guest usage"),
		azureActiveDirectoryOperation("Microsoft.AzureActiveDirectory/guestUsages/delete", "Delete guest usage"),
	}
}

func azureActiveDirectoryOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":  "Microsoft.AzureActiveDirectory",
			"resource":  "Azure AD B2C",
			"operation": description,
		},
	}
}

func azureActiveDirectoryB2CTenantFixture(record *azureActiveDirectoryB2CTenant) map[string]any {
	return map[string]any{
		"id":         azureActiveDirectoryB2CTenantID(record.SubscriptionID, record.ResourceGroup, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.AzureActiveDirectory/b2cDirectories",
		"location":   record.Location,
		"tags":       azureActiveDirectoryAnyStringMap(record.Tags),
		"sku":        record.SKU,
		"properties": record.Properties,
		"systemData": azureActiveDirectorySystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureActiveDirectoryGuestUsageFixture(record *azureActiveDirectoryGuestUsage) map[string]any {
	return map[string]any{
		"id":         azureActiveDirectoryGuestUsageID(record.SubscriptionID, record.ResourceGroup, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.AzureActiveDirectory/guestUsages",
		"location":   record.Location,
		"tags":       azureActiveDirectoryAnyStringMap(record.Tags),
		"properties": record.Properties,
		"systemData": azureActiveDirectorySystemData(record.CreatedAt, record.UpdatedAt),
	}
}

func azureActiveDirectorySystemData(createdAt, updatedAt time.Time) map[string]any {
	if createdAt.IsZero() {
		createdAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return map[string]any{
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"createdAt":          createdAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
	}
}

func azureActiveDirectoryB2CTenantID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AzureActiveDirectory/b2cDirectories/%s", subscriptionID, resourceGroup, name)
}

func azureActiveDirectoryGuestUsageID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AzureActiveDirectory/guestUsages/%s", subscriptionID, resourceGroup, name)
}

func azureActiveDirectoryTenantUUID(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "::")))
	hexed := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexed[0:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32])
}

func azureActiveDirectoryStorageKey(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func azureActiveDirectoryAnyStringMap(in map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func azureActiveDirectoryStringMap(v any) map[string]string {
	if v == nil {
		return nil
	}
	out := map[string]string{}
	switch typed := v.(type) {
	case map[string]string:
		for k, val := range typed {
			out[k] = val
		}
	case map[string]any:
		for k, val := range typed {
			out[k] = azureActiveDirectoryString(val)
		}
	}
	return out
}

func azureActiveDirectoryReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureActiveDirectoryNotFound(w http.ResponseWriter, path, code, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"target":  path,
		},
	})
}

func azureActiveDirectoryConflict(w http.ResponseWriter, path, code, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"target":  path,
		},
	})
}

func azureActiveDirectoryMergeObject(base, overlay map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

func azureActiveDirectoryObject(v any) map[string]any {
	if typed, ok := v.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureActiveDirectoryString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}
