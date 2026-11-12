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

type azureComputeFleetResource struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Identity       map[string]any
	Plan           map[string]any
	Tags           map[string]string
	Zones          []string
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

func (s *Server) handleAzureComputeFleetRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsComputeFleetPath(segments) {
		return false
	}

	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureComputeFleetAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureFleet") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeFleetOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureComputeFleetRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}

	if resourceGroup == "" {
		if len(tail) == 1 && strings.EqualFold(tail[0], "fleets") && r.Method == http.MethodGet {
			s.handleAzureComputeFleetListBySubscription(w, subscriptionID)
			return true
		}
		respondAzureImplemented(w, path)
		return true
	}

	if len(tail) == 1 && strings.EqualFold(tail[0], "fleets") && r.Method == http.MethodGet {
		s.handleAzureComputeFleetListByResourceGroup(w, subscriptionID, resourceGroup)
		return true
	}
	if len(tail) < 2 || !strings.EqualFold(tail[0], "fleets") {
		respondAzureImplemented(w, path)
		return true
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "fleet name is required")
		return true
	}

	if len(tail) == 3 && strings.EqualFold(tail[2], "virtualMachineScaleSets") {
		if r.Method == http.MethodGet {
			s.handleAzureComputeFleetListVirtualMachineScaleSets(w, path, subscriptionID, resourceGroup, name)
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return true
	}

	switch r.Method {
	case http.MethodGet:
		s.handleAzureComputeFleetGet(w, path, subscriptionID, resourceGroup, name, false)
	case http.MethodHead:
		s.handleAzureComputeFleetGet(w, path, subscriptionID, resourceGroup, name, true)
	case http.MethodPut:
		s.handleAzureComputeFleetCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
	case http.MethodPatch:
		s.handleAzureComputeFleetPatch(w, r, path, subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		s.handleAzureComputeFleetDelete(w, path, subscriptionID, resourceGroup, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	return true
}

func azureIsComputeFleetPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.AzureFleet") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureComputeFleetRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "fleets")
}

func azureComputeFleetRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") || strings.TrimSpace(segments[1]) == "" {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+1 < len(segments); idx++ {
		if !strings.EqualFold(segments[idx], "providers") || !strings.EqualFold(segments[idx+1], "Microsoft.AzureFleet") {
			continue
		}
		if idx >= 4 && strings.EqualFold(segments[2], "resourceGroups") && strings.TrimSpace(segments[3]) != "" {
			resourceGroup = strings.TrimSpace(segments[3])
		}
		return subscriptionID, resourceGroup, segments[idx+2:], true
	}
	return "", "", nil, false
}

func azureComputeFleetAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureComputeFleetCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureComputeFleetReadRequestBody(w, r, path)
	if !ok {
		return
	}

	location := azureComputeFleetString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}
	properties := azureComputeFleetObject(payload["properties"])
	if len(properties) == 0 {
		respondAzureInvalidRequest(w, path, "properties is required")
		return
	}
	if len(azureComputeFleetObject(properties["computeProfile"])) == 0 {
		respondAzureInvalidRequest(w, path, "properties.computeProfile is required")
		return
	}
	if len(azureComputeFleetObject(properties["vmSizesProfile"])) == 0 {
		respondAzureInvalidRequest(w, path, "properties.vmSizesProfile is required")
		return
	}

	now := time.Now().UTC()
	key := azureComputeFleetKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureComputeFleets[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureComputeFleetResource{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}

	resource.Location = location
	resource.Identity = azureComputeFleetObject(payload["identity"])
	resource.Plan = azureComputeFleetObject(payload["plan"])
	resource.Tags = azureComputeFleetStringMap(payload["tags"])
	resource.Zones = azureComputeFleetStringSlice(payload["zones"])
	resource.Properties = azureComputeFleetNormalizeProperties(properties, createdAt, now)
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureComputeFleetETag(key, body, now)
	s.azureComputeFleets[key] = resource

	respondJSON(w, status, azureComputeFleetResourceFixture(resource))
}

func (s *Server) handleAzureComputeFleetPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureComputeFleetReadRequestBody(w, r, path)
	if !ok {
		return
	}

	key := azureComputeFleetKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureComputeFleets[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeFleetNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["identity"]; ok {
		resource.Identity = azureComputeFleetObject(payload["identity"])
		updated = true
	}
	if _, ok := payload["plan"]; ok {
		resource.Plan = azureComputeFleetObject(payload["plan"])
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureComputeFleetStringMap(payload["tags"])
		updated = true
	}
	if zones, ok := payload["zones"]; ok {
		resource.Zones = azureComputeFleetStringSlice(zones)
		updated = true
	}
	if properties, ok := payload["properties"]; ok {
		resource.Properties = azureComputeFleetMergeObjects(resource.Properties, azureComputeFleetObject(properties))
		resource.Properties = azureComputeFleetNormalizeProperties(resource.Properties, resource.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	resource.UpdatedAt = now
	resource.ETag = azureComputeFleetETag(key, body, now)
	respondJSON(w, http.StatusOK, azureComputeFleetResourceFixture(resource))
}

func (s *Server) handleAzureComputeFleetGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureComputeFleets[azureComputeFleetKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeFleetNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureComputeFleetResourceFixture(resource))
}

func (s *Server) handleAzureComputeFleetDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azureComputeFleetKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureComputeFleets[key]
	if exists {
		delete(s.azureComputeFleets, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Location", "/azure/operations/computefleet/"+name)
	w.Header().Set("Retry-After", "0")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureComputeFleetListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	items := make([]*azureComputeFleetResource, 0)
	for _, resource := range s.azureComputeFleets {
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
		value = append(value, azureComputeFleetResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureComputeFleetListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	items := make([]*azureComputeFleetResource, 0)
	for _, resource := range s.azureComputeFleets {
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
		value = append(value, azureComputeFleetResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureComputeFleetListVirtualMachineScaleSets(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	resource := s.azureComputeFleets[azureComputeFleetKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()

	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureComputeFleetNotFound(path, name))
		return
	}

	vmssList := azureComputeFleetVirtualMachineScaleSetsFixture(resource)
	respondJSON(w, http.StatusOK, map[string]any{"value": vmssList})
}

func azureComputeFleetOperationsFixture() []map[string]any {
	return []map[string]any{
		azureComputeFleetOperation("Microsoft.AzureFleet/fleets/read", "Read", "Read Compute Fleet resources"),
		azureComputeFleetOperation("Microsoft.AzureFleet/fleets/write", "Write", "Create or update Compute Fleet resources"),
		azureComputeFleetOperation("Microsoft.AzureFleet/fleets/delete", "Delete", "Delete Compute Fleet resources"),
		azureComputeFleetOperation("Microsoft.AzureFleet/fleets/virtualMachineScaleSets/read", "Read VMSS", "Read Compute Fleet VM scale set projections"),
	}
}

func azureComputeFleetOperation(name, operation, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":    "Microsoft Azure Fleet",
			"resource":    "fleets",
			"operation":   operation,
			"description": description,
		},
		"origin":       "user",
		"isDataAction": false,
	}
}

func azureComputeFleetVirtualMachineScaleSetsFixture(resource *azureComputeFleetResource) []any {
	properties := azureComputeFleetObject(resource.Properties)
	regular := azureComputeFleetObject(properties["regularPriorityProfile"])
	spot := azureComputeFleetObject(properties["spotPriorityProfile"])

	names := []string{resource.Name + "-vmss-1"}
	if len(regular) > 0 {
		names[0] = resource.Name + "-regular-vmss"
	}
	if len(spot) > 0 {
		names = append(names, resource.Name+"-spot-vmss")
	}

	status := azureComputeFleetString(properties["provisioningState"])
	if status == "" {
		status = "Succeeded"
	}

	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, map[string]any{
			"id":              azureComputeFleetVirtualMachineScaleSetID(resource.SubscriptionID, resource.ResourceGroup, resource.Name, name),
			"name":            name,
			"type":            "Microsoft.AzureFleet/fleets/virtualMachineScaleSets",
			"operationStatus": status,
		})
	}
	return items
}

func azureComputeFleetResourceFixture(resource *azureComputeFleetResource) map[string]any {
	properties := azureComputeFleetCloneObject(resource.Properties)
	properties["provisioningState"] = azureComputeFleetStringOr(properties["provisioningState"], "Succeeded")
	properties["timeCreated"] = resource.CreatedAt.UTC().Format(time.RFC3339)
	properties["uniqueId"] = azureComputeFleetStableToken("fleet", resource.SubscriptionID, resource.ResourceGroup, resource.Name)

	fixture := map[string]any{
		"id":         azureComputeFleetID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.AzureFleet/fleets",
		"location":   resource.Location,
		"identity":   azureComputeFleetCloneObject(resource.Identity),
		"plan":       azureComputeFleetCloneObject(resource.Plan),
		"tags":       azureComputeFleetStringMapToAny(resource.Tags),
		"zones":      azureComputeFleetStringsToAny(resource.Zones),
		"etag":       resource.ETag,
		"systemData": azureComputeFleetSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": properties,
	}
	return azureComputeFleetPruneNil(fixture)
}

func azureComputeFleetNotFound(path, name string) map[string]any {
	return map[string]any{
		"error":    "NotFound",
		"message":  fmt.Sprintf("Compute Fleet %q was not found", name),
		"provider": providerAzure,
		"path":     path,
	}
}

func azureComputeFleetReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureComputeFleetKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID) + "|" + strings.TrimSpace(resourceGroup) + "|" + strings.TrimSpace(name))
}

func azureComputeFleetID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.AzureFleet/fleets/%s", subscriptionID, resourceGroup, name)
}

func azureComputeFleetVirtualMachineScaleSetID(subscriptionID, resourceGroup, fleetName, vmssName string) string {
	return fmt.Sprintf("%s/virtualMachineScaleSets/%s", azureComputeFleetID(subscriptionID, resourceGroup, fleetName), vmssName)
}

func azureComputeFleetETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+now.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureComputeFleetSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.UTC().Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"lastModifiedAt":     updatedAt.UTC().Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
	}
}

func azureComputeFleetNormalizeProperties(properties map[string]any, createdAt, updatedAt time.Time) map[string]any {
	normalized := azureComputeFleetCloneObject(properties)
	normalized["provisioningState"] = "Succeeded"
	if _, ok := normalized["timeCreated"]; !ok {
		normalized["timeCreated"] = createdAt.UTC().Format(time.RFC3339)
	}
	if _, ok := normalized["lastModifiedAt"]; !ok {
		normalized["lastModifiedAt"] = updatedAt.UTC().Format(time.RFC3339)
	}
	return normalized
}

func azureComputeFleetStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func azureComputeFleetPruneNil(value map[string]any) map[string]any {
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

func azureComputeFleetMergeObjects(base, patch map[string]any) map[string]any {
	if len(base) == 0 && len(patch) == 0 {
		return map[string]any{}
	}
	out := azureComputeFleetCloneObject(base)
	for key, value := range patch {
		if existing, ok := out[key].(map[string]any); ok {
			if patchMap, ok := value.(map[string]any); ok {
				out[key] = azureComputeFleetMergeObjects(existing, patchMap)
				continue
			}
		}
		out[key] = azureComputeFleetCloneAny(value)
	}
	return out
}

func azureComputeFleetCloneObject(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = azureComputeFleetCloneAny(item)
	}
	return cloned
}

func azureComputeFleetCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return azureComputeFleetCloneObject(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = azureComputeFleetCloneAny(item)
		}
		return out
	case []string:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = item
		}
		return out
	default:
		return typed
	}
}

func azureComputeFleetObject(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return azureComputeFleetCloneObject(typed)
	}
	return map[string]any{}
}

func azureComputeFleetString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureComputeFleetStringOr(value any, fallback string) string {
	if str := azureComputeFleetString(value); str != "" {
		return str
	}
	return fallback
}

func azureComputeFleetStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str := azureComputeFleetString(item); str != "" {
			out = append(out, str)
		}
	}
	return out
}

func azureComputeFleetStringMap(value any) map[string]string {
	items, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(items))
	for key, item := range items {
		if str := azureComputeFleetString(item); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureComputeFleetStringMapToAny(value map[string]string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func azureComputeFleetStringsToAny(values []string) []any {
	if len(values) == 0 {
		return []any{}
	}
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}
