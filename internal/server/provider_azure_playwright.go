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

const azurePlaywrightAPIVersion = "2025-09-01"

type azurePlaywrightWorkspace struct {
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

func (s *Server) handleAzurePlaywrightRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsPlaywrightPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azurePlaywrightAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.LoadTestService") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azurePlaywrightOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, location, tail, ok := azurePlaywrightRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "checkNameAvailability"):
		if r.Method == http.MethodPost {
			s.handleAzurePlaywrightCheckNameAvailability(w, r, path, subscriptionID)
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	case len(tail) >= 1 && strings.EqualFold(tail[0], "playwrightWorkspaces"):
		s.handleAzurePlaywrightWorkspacesRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	case len(tail) >= 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "playwrightQuotas"):
		s.handleAzurePlaywrightSubscriptionQuotasRoute(w, r, path, subscriptionID, tail[1], tail[2:])
		return true
	case len(tail) >= 1 && strings.EqualFold(tail[0], "playwrightQuotas"):
		s.handleAzurePlaywrightSubscriptionQuotasRoute(w, r, path, subscriptionID, location, tail)
		return true
	default:
		return false
	}
}

func azureIsPlaywrightPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.LoadTestService") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, _, tail, ok := azurePlaywrightRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability", "playwrightworkspaces", "playwrightquotas":
		return true
	case "locations":
		return len(tail) >= 3 && strings.EqualFold(tail[2], "playwrightQuotas")
	default:
		return false
	}
}

func azurePlaywrightAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azurePlaywrightRouteContext(segments []string) (subscriptionID, resourceGroup, location string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
		for idx := 4; idx+1 < len(segments); idx++ {
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.LoadTestService") {
				return subscriptionID, resourceGroup, "", segments[idx+2:], true
			}
		}
		return "", "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "locations") && idx+3 < len(segments) &&
			strings.EqualFold(segments[idx+2], "providers") && strings.EqualFold(segments[idx+3], "Microsoft.LoadTestService") {
			location = strings.TrimSpace(segments[idx+1])
			return subscriptionID, "", location, segments[idx+4:], true
		}
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.LoadTestService") {
			return subscriptionID, "", "", segments[idx+2:], true
		}
	}
	return "", "", "", nil, false
}

func (s *Server) handleAzurePlaywrightCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_ = subscriptionID
	_, payload, ok := azurePlaywrightReadRequestBody(w, r, path)
	if !ok {
		return
	}
	name := strings.TrimSpace(azurePlaywrightString(payload["name"]))
	typ := strings.TrimSpace(azurePlaywrightString(payload["type"]))
	if name == "" || typ == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	if !strings.EqualFold(typ, "Microsoft.LoadTestService/playwrightWorkspaces") {
		respondAzureInvalidRequest(w, path, "type must be Microsoft.LoadTestService/playwrightWorkspaces")
		return
	}
	if !azurePlaywrightValidWorkspaceName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "workspace name must match ^[a-zA-Z0-9-]{3,24}$",
		})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	for _, workspace := range s.azurePlaywrightWorkspaces {
		if strings.EqualFold(workspace.Name, name) {
			respondJSON(w, http.StatusOK, map[string]any{
				"nameAvailable": false,
				"reason":        "AlreadyExists",
				"message":       "workspace name is already in use",
			})
			return
		}
	}
	if strings.EqualFold(name, "reserved") {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "workspace name is reserved",
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"nameAvailable": true,
	})
}

func (s *Server) handleAzurePlaywrightWorkspacesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzurePlaywrightListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzurePlaywrightListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azurePlaywrightValidWorkspaceName(name) {
		respondAzureInvalidRequest(w, path, "playwright workspace name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzurePlaywrightGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzurePlaywrightGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzurePlaywrightCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzurePlaywrightPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzurePlaywrightDelete(w, path, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) >= 3 && strings.EqualFold(tail[2], "quotas") {
		s.handleAzurePlaywrightWorkspaceQuotasRoute(w, r, path, subscriptionID, resourceGroup, name, tail[3:])
		return
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzurePlaywrightSubscriptionQuotasRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string, tail []string) {
	if location == "" {
		respondAzureImplemented(w, path)
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 1 {
		respondJSON(w, http.StatusOK, map[string]any{"value": azurePlaywrightSubscriptionQuotaFixtures(subscriptionID, location, s.azurePlaywrightPrimaryWorkspaceID(subscriptionID, location))})
		return
	}
	quotaName := strings.TrimSpace(tail[1])
	quota := azurePlaywrightFindNamedQuota(azurePlaywrightSubscriptionQuotaFixtures(subscriptionID, location, s.azurePlaywrightPrimaryWorkspaceID(subscriptionID, location)), quotaName)
	if quota == nil {
		respondJSON(w, http.StatusNotFound, azurePlaywrightNotFound(path, quotaName))
		return
	}
	respondJSON(w, http.StatusOK, quota)
}

func (s *Server) handleAzurePlaywrightWorkspaceQuotasRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, workspaceName string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s.providerStorageMu.Lock()
	workspace := s.azurePlaywrightWorkspaces[azurePlaywrightWorkspaceKey(subscriptionID, resourceGroup, workspaceName)]
	s.providerStorageMu.Unlock()
	if workspace == nil {
		respondJSON(w, http.StatusNotFound, azurePlaywrightNotFound(path, workspaceName))
		return
	}
	if len(tail) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"value": azurePlaywrightWorkspaceQuotaFixtures(workspace)})
		return
	}
	if len(tail) != 1 {
		respondAzureImplemented(w, path)
		return
	}
	quotaName := strings.TrimSpace(tail[0])
	quota := azurePlaywrightFindNamedQuota(azurePlaywrightWorkspaceQuotaFixtures(workspace), quotaName)
	if quota == nil {
		respondJSON(w, http.StatusNotFound, azurePlaywrightNotFound(path, quotaName))
		return
	}
	respondJSON(w, http.StatusOK, quota)
}

func (s *Server) handleAzurePlaywrightCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azurePlaywrightReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azurePlaywrightString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azurePlaywrightWorkspaceKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	workspace := s.azurePlaywrightWorkspaces[key]
	status := http.StatusOK
	createdAt := now
	if workspace == nil {
		workspace = &azurePlaywrightWorkspace{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = workspace.CreatedAt
	}

	workspace.Location = location
	workspace.Tags = azurePlaywrightStringMap(payload["tags"])
	workspace.Properties = azurePlaywrightNormalizeWorkspaceProperties(name, subscriptionID, resourceGroup, location, azurePlaywrightObject(payload["properties"]), createdAt, now)
	workspace.CreatedAt = createdAt
	workspace.UpdatedAt = now
	workspace.ETag = azurePlaywrightETag(key, body, now)
	s.azurePlaywrightWorkspaces[key] = workspace

	if status == http.StatusCreated {
		w.Header().Set("Azure-AsyncOperation", "/azure/providers/Microsoft.LoadTestService/operations/playwright-"+name+"?api-version="+azurePlaywrightAPIVersion)
	}
	respondJSON(w, status, azurePlaywrightWorkspaceFixture(workspace))
}

func (s *Server) handleAzurePlaywrightPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azurePlaywrightReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azurePlaywrightWorkspaceKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	workspace := s.azurePlaywrightWorkspaces[key]
	if workspace == nil {
		respondJSON(w, http.StatusNotFound, azurePlaywrightNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["tags"]; ok {
		workspace.Tags = azurePlaywrightStringMap(payload["tags"])
		updated = true
	}
	if rawProps, ok := payload["properties"]; ok {
		workspace.Properties = azurePlaywrightMergeObjects(workspace.Properties, azurePlaywrightObject(rawProps))
		workspace.Properties = azurePlaywrightNormalizeWorkspaceProperties(name, subscriptionID, resourceGroup, workspace.Location, workspace.Properties, workspace.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	workspace.UpdatedAt = now
	workspace.ETag = azurePlaywrightETag(key, body, now)
	respondJSON(w, http.StatusOK, azurePlaywrightWorkspaceFixture(workspace))
}

func (s *Server) handleAzurePlaywrightGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	workspace := s.azurePlaywrightWorkspaces[azurePlaywrightWorkspaceKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if workspace == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azurePlaywrightNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azurePlaywrightWorkspaceFixture(workspace))
}

func (s *Server) handleAzurePlaywrightDelete(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	key := azurePlaywrightWorkspaceKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	_, exists := s.azurePlaywrightWorkspaces[key]
	if exists {
		delete(s.azurePlaywrightWorkspaces, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Location", "/azure"+azurePlaywrightWorkspaceID(subscriptionID, resourceGroup, name)+"?api-version="+azurePlaywrightAPIVersion)
	w.Header().Set("Retry-After", "10")
	w.WriteHeader(http.StatusAccepted)
	_ = path
}

func (s *Server) handleAzurePlaywrightListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, workspace := range s.azurePlaywrightWorkspaces {
		if workspace.SubscriptionID != subscriptionID {
			continue
		}
		values = append(values, azurePlaywrightWorkspaceFixture(workspace))
	}
	sort.Slice(values, func(i, j int) bool {
		return azurePlaywrightString(values[i]["id"]) < azurePlaywrightString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzurePlaywrightListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, workspace := range s.azurePlaywrightWorkspaces {
		if workspace.SubscriptionID != subscriptionID || !strings.EqualFold(workspace.ResourceGroup, resourceGroup) {
			continue
		}
		values = append(values, azurePlaywrightWorkspaceFixture(workspace))
	}
	sort.Slice(values, func(i, j int) bool {
		return azurePlaywrightString(values[i]["id"]) < azurePlaywrightString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) azurePlaywrightPrimaryWorkspaceID(subscriptionID, location string) string {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	candidates := make([]*azurePlaywrightWorkspace, 0)
	for _, workspace := range s.azurePlaywrightWorkspaces {
		if workspace.SubscriptionID == subscriptionID && strings.EqualFold(workspace.Location, location) {
			candidates = append(candidates, workspace)
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return azurePlaywrightWorkspaceID(candidates[i].SubscriptionID, candidates[i].ResourceGroup, candidates[i].Name) <
			azurePlaywrightWorkspaceID(candidates[j].SubscriptionID, candidates[j].ResourceGroup, candidates[j].Name)
	})
	if len(candidates) == 0 {
		return ""
	}
	return azurePlaywrightWorkspaceID(candidates[0].SubscriptionID, candidates[0].ResourceGroup, candidates[0].Name)
}

func azurePlaywrightOperationsFixture() []map[string]any {
	operations := []string{
		"Microsoft.LoadTestService/PlaywrightWorkspaces/Read",
		"Microsoft.LoadTestService/PlaywrightWorkspaces/Write",
		"Microsoft.LoadTestService/PlaywrightWorkspaces/Delete",
		"Microsoft.LoadTestService/PlaywrightQuotas/Read",
		"Microsoft.LoadTestService/PlaywrightWorkspaceQuotas/Read",
		"Microsoft.LoadTestService/loadTests/Read",
		"Microsoft.LoadTestService/loadTests/Write",
		"Microsoft.LoadTestService/loadTests/Delete",
	}
	out := make([]map[string]any, 0, len(operations))
	for _, name := range operations {
		out = append(out, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider": "Microsoft.LoadTestService",
				"resource": strings.TrimPrefix(name, "Microsoft.LoadTestService/"),
			},
		})
	}
	return out
}

func azurePlaywrightWorkspaceFixture(workspace *azurePlaywrightWorkspace) map[string]any {
	return map[string]any{
		"id":         azurePlaywrightWorkspaceID(workspace.SubscriptionID, workspace.ResourceGroup, workspace.Name),
		"name":       workspace.Name,
		"type":       "Microsoft.LoadTestService/playwrightWorkspaces",
		"location":   workspace.Location,
		"tags":       azurePlaywrightAnyStringMap(workspace.Tags),
		"systemData": azurePlaywrightSystemData(workspace.CreatedAt, workspace.UpdatedAt),
		"properties": azurePlaywrightMergeObjects(map[string]any{}, workspace.Properties),
	}
}

func azurePlaywrightSubscriptionQuotaFixtures(subscriptionID, location, workspaceID string) []map[string]any {
	idBase := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.LoadTestService/locations/%s/playwrightQuotas", subscriptionID, location)
	state := "NotApplicable"
	if workspaceID != "" {
		state = "Active"
	}
	return []map[string]any{
		{
			"id":   idBase + "/ExecutionMinutes",
			"name": "ExecutionMinutes",
			"type": "Microsoft.LoadTestService/locations/playwrightQuotas",
			"systemData": map[string]any{
				"createdAt":      "2026-01-01T00:00:00Z",
				"lastModifiedAt": "2026-01-01T00:00:00Z",
			},
			"properties": map[string]any{
				"freeTrial": map[string]any{
					"workspaceId": workspaceID,
					"state":       state,
				},
				"provisioningState": "Succeeded",
			},
		},
	}
}

func azurePlaywrightWorkspaceQuotaFixtures(workspace *azurePlaywrightWorkspace) []map[string]any {
	idBase := azurePlaywrightWorkspaceID(workspace.SubscriptionID, workspace.ResourceGroup, workspace.Name) + "/quotas"
	return []map[string]any{
		{
			"id":   idBase + "/ExecutionMinutes",
			"name": "ExecutionMinutes",
			"type": "Microsoft.LoadTestService/playwrightWorkspaces/quotas",
			"properties": map[string]any{
				"limit":             5000,
				"unit":              "Minutes",
				"workspaceId":       azurePlaywrightString(workspace.Properties["workspaceId"]),
				"provisioningState": "Succeeded",
			},
		},
	}
}

func azurePlaywrightFindNamedQuota(items []map[string]any, name string) map[string]any {
	for _, item := range items {
		if strings.EqualFold(azurePlaywrightString(item["name"]), name) {
			return item
		}
	}
	return nil
}

func azurePlaywrightNormalizeWorkspaceProperties(name, subscriptionID, resourceGroup, location string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azurePlaywrightMergeObjects(map[string]any{}, payload)
	if azurePlaywrightString(props["localAuth"]) == "" {
		props["localAuth"] = "Enabled"
	}
	if azurePlaywrightString(props["regionalAffinity"]) == "" {
		props["regionalAffinity"] = "Enabled"
	}
	props["workspaceId"] = azurePlaywrightWorkspaceGUID(subscriptionID, resourceGroup, name)
	props["dataplaneUri"] = fmt.Sprintf("https://%s.%s.playwright.azure.example", strings.ToLower(name), strings.ToLower(location))
	props["provisioningState"] = "Succeeded"
	props["createdDateTime"] = createdAt.Format(time.RFC3339)
	props["lastModifiedDateTime"] = updatedAt.Format(time.RFC3339)
	return props
}

func azurePlaywrightWorkspaceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "::" + resourceGroup + "::" + name)
}

func azurePlaywrightWorkspaceID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.LoadTestService/playwrightWorkspaces/%s", subscriptionID, resourceGroup, name)
}

func azurePlaywrightWorkspaceGUID(subscriptionID, resourceGroup, name string) string {
	sum := sha1.Sum([]byte(azurePlaywrightWorkspaceID(subscriptionID, resourceGroup, name)))
	hexValue := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}

func azurePlaywrightSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azurePlaywrightNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("resource %q was not found for path %s", name, path),
		},
	}
}

func azurePlaywrightReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azurePlaywrightString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azurePlaywrightObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return azurePlaywrightMergeObjects(map[string]any{}, typed)
}

func azurePlaywrightStringMap(value any) map[string]string {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if str := azurePlaywrightString(val); str != "" {
			out[key] = str
		}
	}
	return out
}

func azurePlaywrightAnyStringMap(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, val := range value {
		out[key] = val
	}
	return out
}

func azurePlaywrightMergeObjects(base, overlay map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		if overlayMap, ok := value.(map[string]any); ok {
			if baseMap, ok := merged[key].(map[string]any); ok {
				merged[key] = azurePlaywrightMergeObjects(baseMap, overlayMap)
			} else {
				merged[key] = azurePlaywrightMergeObjects(map[string]any{}, overlayMap)
			}
			continue
		}
		merged[key] = value
	}
	return merged
}

func azurePlaywrightETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azurePlaywrightValidWorkspaceName(name string) bool {
	if len(name) < 3 || len(name) > 24 {
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
