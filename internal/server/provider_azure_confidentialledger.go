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

type azureConfidentialLedgerResource struct {
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

func (s *Server) handleAzureConfidentialLedgerRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsConfidentialLedgerPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureConfidentialLedgerAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ConfidentialLedger") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureConfidentialLedgerOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureConfidentialLedgerRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		if len(tail) == 1 && strings.EqualFold(tail[0], "ledgers") && r.Method == http.MethodGet {
			s.handleAzureConfidentialLedgerListBySubscription(w, subscriptionID)
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	if len(tail) == 1 && strings.EqualFold(tail[0], "ledgers") && r.Method == http.MethodGet {
		s.handleAzureConfidentialLedgerListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	}
	if len(tail) < 2 || !strings.EqualFold(tail[0], "ledgers") {
		respondAzureImplemented(w, path)
		return true
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "ledger name is required")
		return true
	}
	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return true
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureConfidentialLedgerGet(w, path, subscriptionID, resourceGroup, name, false)
	case http.MethodHead:
		s.handleAzureConfidentialLedgerGet(w, path, subscriptionID, resourceGroup, name, true)
	case http.MethodPut:
		s.handleAzureConfidentialLedgerCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
	case http.MethodPatch:
		s.handleAzureConfidentialLedgerPatch(w, r, path, subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		s.handleAzureConfidentialLedgerDelete(w, path, subscriptionID, resourceGroup, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	return true
}

func azureIsConfidentialLedgerPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ConfidentialLedger") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureConfidentialLedgerRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "ledgers")
}

func azureConfidentialLedgerRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.ConfidentialLedger") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureConfidentialLedgerAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureConfidentialLedgerCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureConfidentialLedgerReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := azureConfidentialLedgerString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureConfidentialLedgerObject(payload["properties"])
	if len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}
	if azureConfidentialLedgerString(properties["ledgerType"]) == "" {
		respondAzureInvalidRequest(w, path, "properties.ledgerType is required")
		return
	}

	now := time.Now().UTC()
	key := azureConfidentialLedgerKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureConfidentialLedgers[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureConfidentialLedgerResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}

	resource.Location = location
	resource.Tags = azureConfidentialLedgerStringMap(payload["tags"])
	resource.Properties = azureConfidentialLedgerNormalizeProperties(name, properties, createdAt, now)
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureConfidentialLedgerETag(key, body, now)
	s.azureConfidentialLedgers[key] = resource

	respondJSON(w, status, azureConfidentialLedgerResourceFixture(resource))
}

func (s *Server) handleAzureConfidentialLedgerPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureConfidentialLedgerReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureConfidentialLedgerKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureConfidentialLedgers[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureConfidentialLedgerNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["location"]; ok {
		location := azureConfidentialLedgerString(payload["location"])
		if location == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		resource.Location = location
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureConfidentialLedgerStringMap(payload["tags"])
		updated = true
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureConfidentialLedgerMergeObjects(resource.Properties, azureConfidentialLedgerObject(properties))
		resource.Properties = azureConfidentialLedgerNormalizeProperties(name, resource.Properties, resource.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	resource.UpdatedAt = now
	resource.ETag = azureConfidentialLedgerETag(key, body, now)
	respondJSON(w, http.StatusOK, azureConfidentialLedgerResourceFixture(resource))
}

func (s *Server) handleAzureConfidentialLedgerGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureConfidentialLedgers[azureConfidentialLedgerKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureConfidentialLedgerNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureConfidentialLedgerResourceFixture(resource))
}

func (s *Server) handleAzureConfidentialLedgerDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureConfidentialLedgerKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureConfidentialLedgers[key]
	if exists {
		delete(s.azureConfidentialLedgers, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAzureConfidentialLedgerListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]*azureConfidentialLedgerResource, 0)
	for _, resource := range s.azureConfidentialLedgers {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) && strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureConfidentialLedgerResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureConfidentialLedgerListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureConfidentialLedgerResource, 0)
	for _, resource := range s.azureConfidentialLedgers {
		if strings.EqualFold(resource.SubscriptionID, subscriptionID) {
			items = append(items, resource)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		if strings.EqualFold(items[i].ResourceGroup, items[j].ResourceGroup) {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		return strings.ToLower(items[i].ResourceGroup) < strings.ToLower(items[j].ResourceGroup)
	})

	value := make([]any, 0, len(items))
	for _, item := range items {
		value = append(value, azureConfidentialLedgerResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func azureConfidentialLedgerOperationsFixture() []map[string]any {
	return []map[string]any{
		azureConfidentialLedgerOperation("Microsoft.ConfidentialLedger/ledgers/read", "Read", "Read Confidential Ledger resources"),
		azureConfidentialLedgerOperation("Microsoft.ConfidentialLedger/ledgers/write", "Write", "Create or update Confidential Ledger resources"),
		azureConfidentialLedgerOperation("Microsoft.ConfidentialLedger/ledgers/delete", "Delete", "Delete Confidential Ledger resources"),
	}
}

func azureConfidentialLedgerOperation(name, operation, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":    "Microsoft Confidential Ledger",
			"resource":    "ledgers",
			"operation":   operation,
			"description": description,
		},
		"origin":       "user",
		"isDataAction": false,
	}
}

func azureConfidentialLedgerResourceFixture(resource *azureConfidentialLedgerResource) map[string]any {
	properties := azureConfidentialLedgerCloneObject(resource.Properties)
	properties["ledgerName"] = resource.Name
	properties["ledgerUri"] = "https://" + strings.ToLower(resource.Name) + ".confidential-ledger.azure.com"
	properties["identityServiceUri"] = "https://" + strings.ToLower(resource.Name) + ".identity.confidential-ledger.azure.com"
	properties["ledgerInternalNamespace"] = "cl-" + azureConfidentialLedgerStableToken(resource.SubscriptionID, resource.ResourceGroup, resource.Name)
	properties["provisioningState"] = azureConfidentialLedgerStringOr(properties["provisioningState"], "Succeeded")

	fixture := map[string]any{
		"id":         azureConfidentialLedgerID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ConfidentialLedger/ledgers",
		"location":   resource.Location,
		"tags":       azureConfidentialLedgerStringMapToAny(resource.Tags),
		"etag":       resource.ETag,
		"systemData": azureConfidentialLedgerSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": properties,
	}
	return azureConfidentialLedgerPruneNil(fixture)
}

func azureConfidentialLedgerNotFound(path, name string) map[string]any {
	return map[string]any{
		"error":    "NotFound",
		"message":  fmt.Sprintf("Confidential Ledger %q was not found", name),
		"provider": providerAzure,
		"path":     path,
	}
}

func azureConfidentialLedgerReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureConfidentialLedgerKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID) + "|" + strings.TrimSpace(resourceGroup) + "|" + strings.TrimSpace(name))
}

func azureConfidentialLedgerID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ConfidentialLedger/ledgers/%s", subscriptionID, resourceGroup, name)
}

func azureConfidentialLedgerETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureConfidentialLedgerSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureConfidentialLedgerNormalizeProperties(name string, properties map[string]any, _ time.Time, _ time.Time) map[string]any {
	normalized := azureConfidentialLedgerCloneObject(properties)
	normalized["ledgerType"] = azureConfidentialLedgerStringOr(normalized["ledgerType"], "Public")
	normalized["ledgerName"] = name
	normalized["ledgerUri"] = "https://" + strings.ToLower(name) + ".confidential-ledger.azure.com"
	normalized["identityServiceUri"] = "https://" + strings.ToLower(name) + ".identity.confidential-ledger.azure.com"
	normalized["ledgerInternalNamespace"] = "cl-" + azureConfidentialLedgerStableToken(name)
	normalized["provisioningState"] = "Succeeded"
	return normalized
}

func azureConfidentialLedgerStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func azureConfidentialLedgerPruneNil(value map[string]any) map[string]any {
	out := make(map[string]any, len(value))
	for k, v := range value {
		switch typed := v.(type) {
		case nil:
			continue
		case map[string]any:
			if len(typed) == 0 {
				continue
			}
		case []any:
			if len(typed) == 0 {
				continue
			}
		}
		out[k] = v
	}
	return out
}

func azureConfidentialLedgerMergeObjects(base, patch map[string]any) map[string]any {
	if len(base) == 0 && len(patch) == 0 {
		return map[string]any{}
	}
	out := azureConfidentialLedgerCloneObject(base)
	for key, value := range patch {
		if existing, ok := out[key].(map[string]any); ok {
			if patchMap, ok := value.(map[string]any); ok {
				out[key] = azureConfidentialLedgerMergeObjects(existing, patchMap)
				continue
			}
		}
		out[key] = azureConfidentialLedgerCloneAny(value)
	}
	return out
}

func azureConfidentialLedgerCloneObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = azureConfidentialLedgerCloneAny(item)
	}
	return cloned
}

func azureConfidentialLedgerCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return azureConfidentialLedgerCloneObject(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = azureConfidentialLedgerCloneAny(item)
		}
		return out
	default:
		return typed
	}
}

func azureConfidentialLedgerObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return azureConfidentialLedgerCloneObject(typed)
	}
	return map[string]any{}
}

func azureConfidentialLedgerString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureConfidentialLedgerStringOr(value any, fallback string) string {
	if str := azureConfidentialLedgerString(value); str != "" {
		return str
	}
	return fallback
}

func azureConfidentialLedgerStringMap(value any) map[string]string {
	items, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(items))
	for key, item := range items {
		if str := azureConfidentialLedgerString(item); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureConfidentialLedgerStringMapToAny(value map[string]string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}
