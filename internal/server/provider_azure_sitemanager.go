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

type azureSiteManagerSite struct {
	SubscriptionID   string
	ResourceGroup    string
	Name             string
	Location         string
	Kind             string
	Tags             map[string]string
	ExtendedLocation map[string]any
	Identity         map[string]any
	Properties       map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (s *Server) handleAzureSiteManagerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if !azureIsSiteManagerPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureSiteManagerAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Edge") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureSiteManagerOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureSiteManagerRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	if len(tail) == 1 && strings.EqualFold(tail[0], "sites") && resourceGroup == "" && r.Method == http.MethodGet {
		s.handleAzureSiteManagerListBySubscription(w, subscriptionID)
		return true
	}
	if len(tail) == 1 && strings.EqualFold(tail[0], "sites") && resourceGroup != "" && r.Method == http.MethodGet {
		s.handleAzureSiteManagerListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	}
	if resourceGroup != "" && len(tail) == 2 && strings.EqualFold(tail[0], "sites") {
		name := strings.TrimSpace(tail[1])
		if name == "" {
			respondAzureInvalidRequest(w, path, "site name is required")
			return true
		}
		switch r.Method {
		case http.MethodGet:
			s.handleAzureSiteManagerGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureSiteManagerGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureSiteManagerCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureSiteManagerPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureSiteManagerDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func (s *Server) handleAzureSiteManagerCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureSiteManagerReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := strings.TrimSpace(azureSiteManagerString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	record := &azureSiteManagerSite{
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
		Name:           name,
		Location:       location,
		Kind:           strings.TrimSpace(azureSiteManagerString(payload["kind"])),
		Tags:           azureSiteManagerStringMap(payload["tags"]),
		ExtendedLocation: azureSiteManagerObject(
			payload["extendedLocation"],
		),
		Identity: azureSiteManagerObject(payload["identity"]),
		Properties: azureSiteManagerMergeObject(map[string]any{
			"provisioningState": "Succeeded",
			"displayName":       name,
			"siteId":            azureSiteManagerSiteID(subscriptionID, resourceGroup, name),
		}, azureSiteManagerObject(payload["properties"])),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.providerStorageMu.Lock()
	s.azureSiteManagerSites[azureSiteManagerStorageKey(azureSiteManagerSiteID(subscriptionID, resourceGroup, name))] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureSiteManagerSiteFixture(record))
}

func (s *Server) handleAzureSiteManagerPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureSiteManagerReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureSiteManagerStorageKey(azureSiteManagerSiteID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureSiteManagerSites[key]
	if record == nil {
		s.providerStorageMu.Unlock()
		azureSiteManagerNotFound(w, path, "site not found")
		return
	}
	if location := strings.TrimSpace(azureSiteManagerString(payload["location"])); location != "" {
		record.Location = location
	}
	if kind := strings.TrimSpace(azureSiteManagerString(payload["kind"])); kind != "" {
		record.Kind = kind
	}
	if tags := azureSiteManagerStringMap(payload["tags"]); tags != nil {
		record.Tags = tags
	}
	if identity := azureSiteManagerObject(payload["identity"]); len(identity) != 0 {
		record.Identity = identity
	}
	if extendedLocation := azureSiteManagerObject(payload["extendedLocation"]); len(extendedLocation) != 0 {
		record.ExtendedLocation = extendedLocation
	}
	if properties := azureSiteManagerObject(payload["properties"]); len(properties) != 0 {
		record.Properties = azureSiteManagerMergeObject(record.Properties, properties)
	}
	record.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureSiteManagerSiteFixture(record))
}

func (s *Server) handleAzureSiteManagerGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	key := azureSiteManagerStorageKey(azureSiteManagerSiteID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureSiteManagerSites[key]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureSiteManagerNotFound(w, path, "site not found")
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureSiteManagerSiteFixture(record))
}

func (s *Server) handleAzureSiteManagerDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureSiteManagerStorageKey(azureSiteManagerSiteID(subscriptionID, resourceGroup, name))
	s.providerStorageMu.Lock()
	record := s.azureSiteManagerSites[key]
	delete(s.azureSiteManagerSites, key)
	s.providerStorageMu.Unlock()
	if record == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureSiteManagerSiteFixture(record))
}

func (s *Server) handleAzureSiteManagerListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureSiteManagerSites {
		if record.SubscriptionID == subscriptionID && strings.EqualFold(record.ResourceGroup, resourceGroup) {
			items = append(items, azureSiteManagerSiteFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureSiteManagerString(items[i]["id"])) < strings.ToLower(azureSiteManagerString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureSiteManagerListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0)
	for _, record := range s.azureSiteManagerSites {
		if record.SubscriptionID == subscriptionID {
			items = append(items, azureSiteManagerSiteFixture(record))
		}
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureSiteManagerString(items[i]["id"])) < strings.ToLower(azureSiteManagerString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func azureIsSiteManagerPath(segments []string) bool {
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Edge") {
		return true
	}
	if len(segments) < 2 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Edge") {
			return true
		}
	}
	return false
}

func azureSiteManagerAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureSiteManagerRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if len(segments) >= 4 && strings.EqualFold(segments[2], "providers") && strings.EqualFold(segments[3], "Microsoft.Edge") {
		return subscriptionID, "", segments[4:], true
	}
	if len(segments) >= 6 &&
		strings.EqualFold(segments[2], "resourceGroups") &&
		strings.TrimSpace(segments[3]) != "" &&
		strings.EqualFold(segments[4], "providers") &&
		strings.EqualFold(segments[5], "Microsoft.Edge") {
		return subscriptionID, strings.TrimSpace(segments[3]), segments[6:], true
	}
	return "", "", nil, false
}

func azureSiteManagerOperationsFixture() []map[string]any {
	return []map[string]any{
		azureSiteManagerOperation("Microsoft.Edge/sites/read", "Read Azure Arc site"),
		azureSiteManagerOperation("Microsoft.Edge/sites/write", "Create or update Azure Arc site"),
		azureSiteManagerOperation("Microsoft.Edge/sites/delete", "Delete Azure Arc site"),
	}
}

func azureSiteManagerOperation(name, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":  "Microsoft Edge",
			"resource":  "sites",
			"operation": description,
		},
		"isDataAction": false,
	}
}

func azureSiteManagerSiteFixture(record *azureSiteManagerSite) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	properties := azureSiteManagerMergeObject(map[string]any{
		"provisioningState": "Succeeded",
		"siteId":            azureSiteManagerSiteID(record.SubscriptionID, record.ResourceGroup, record.Name),
		"displayName":       record.Name,
	}, record.Properties)
	fixture := map[string]any{
		"id":         azureSiteManagerSiteID(record.SubscriptionID, record.ResourceGroup, record.Name),
		"name":       record.Name,
		"type":       "Microsoft.Edge/sites",
		"location":   record.Location,
		"tags":       azureSiteManagerStringMapAny(record.Tags),
		"properties": properties,
		"systemData": map[string]any{
			"createdBy":          "stackyard",
			"createdByType":      "Application",
			"createdAt":          record.CreatedAt.Format(time.RFC3339),
			"lastModifiedBy":     "stackyard",
			"lastModifiedByType": "Application",
			"lastModifiedAt":     record.UpdatedAt.Format(time.RFC3339),
		},
		"etag": azureSiteManagerETag(record),
	}
	if record.Kind != "" {
		fixture["kind"] = record.Kind
	}
	if len(record.Identity) != 0 {
		fixture["identity"] = azureSiteManagerCloneMap(record.Identity)
	}
	if len(record.ExtendedLocation) != 0 {
		fixture["extendedLocation"] = azureSiteManagerCloneMap(record.ExtendedLocation)
	}
	return fixture
}

func azureSiteManagerSiteID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Edge/sites/%s", subscriptionID, resourceGroup, name)
}

func azureSiteManagerStorageKey(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func azureSiteManagerETag(record *azureSiteManagerSite) string {
	sum := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%s", azureSiteManagerSiteID(record.SubscriptionID, record.ResourceGroup, record.Name), record.Location, record.UpdatedAt.Format(time.RFC3339Nano))))
	return `"` + hex.EncodeToString(sum[:]) + `"`
}

func azureSiteManagerReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureSiteManagerNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": message,
			"target":  path,
		},
	})
}

func azureSiteManagerString(value any) string {
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func azureSiteManagerObject(value any) map[string]any {
	if value == nil {
		return nil
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return azureSiteManagerCloneMap(obj)
}

func azureSiteManagerStringMap(value any) map[string]string {
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

func azureSiteManagerStringMapAny(tags map[string]string) map[string]any {
	if len(tags) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

func azureSiteManagerMergeObject(base, overlay map[string]any) map[string]any {
	out := azureSiteManagerCloneMap(base)
	for key, value := range overlay {
		if nested, ok := value.(map[string]any); ok {
			existing, _ := out[key].(map[string]any)
			out[key] = azureSiteManagerMergeObject(existing, nested)
			continue
		}
		out[key] = value
	}
	return out
}

func azureSiteManagerCloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		if nested, ok := value.(map[string]any); ok {
			out[key] = azureSiteManagerCloneMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}
