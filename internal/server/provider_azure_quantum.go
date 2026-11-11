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

const azureQuantumAPIVersion = "2025-12-15-preview"

type azureQuantumWorkspace struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Location       string
	Identity       map[string]any
	Tags           map[string]string
	Properties     map[string]any
	PrimaryKey     string
	SecondaryKey   string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ETag           string
}

func (s *Server) handleAzureQuantumRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsQuantumPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureQuantumAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Quantum") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureQuantumOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureQuantumRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}

	switch {
	case len(tail) == 1 && strings.EqualFold(tail[0], "checkNameAvailability"):
		if r.Method == http.MethodPost {
			s.handleAzureQuantumCheckNameAvailability(w, r, path, subscriptionID)
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	case len(tail) == 3 && strings.EqualFold(tail[0], "locations") && strings.EqualFold(tail[2], "offerings"):
		if r.Method == http.MethodGet {
			s.handleAzureQuantumListOfferings(w, subscriptionID, tail[1])
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	case len(tail) >= 1 && strings.EqualFold(tail[0], "workspaces"):
		s.handleAzureQuantumWorkspacesRoute(w, r, path, subscriptionID, resourceGroup, tail)
		return true
	default:
		return false
	}
}

func azureIsQuantumPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Quantum") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureQuantumRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "checknameavailability", "workspaces":
		return true
	case "locations":
		return len(tail) >= 3 && strings.EqualFold(tail[2], "offerings")
	default:
		return false
	}
}

func azureQuantumAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureQuantumRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Quantum") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Quantum") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureQuantumCheckNameAvailability(w http.ResponseWriter, r *http.Request, path, subscriptionID string) {
	_ = subscriptionID
	_, payload, ok := azureQuantumReadRequestBody(w, r, path)
	if !ok {
		return
	}
	name := strings.TrimSpace(azureQuantumString(payload["name"]))
	typ := strings.TrimSpace(azureQuantumString(payload["type"]))
	if name == "" || typ == "" {
		respondAzureInvalidRequest(w, path, "name and type are required")
		return
	}
	if !strings.EqualFold(typ, "Microsoft.Quantum/workspaces") {
		respondAzureInvalidRequest(w, path, "type must be Microsoft.Quantum/workspaces")
		return
	}
	if !azureQuantumValidWorkspaceName(name) {
		respondJSON(w, http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "workspace name must be 3-63 characters of letters, digits, or hyphens",
		})
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	for _, workspace := range s.azureQuantumWorkspaces {
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
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *Server) handleAzureQuantumListOfferings(w http.ResponseWriter, subscriptionID, location string) {
	respondJSON(w, http.StatusOK, map[string]any{"value": azureQuantumOfferingsFixture(subscriptionID, location)})
}

func (s *Server) handleAzureQuantumWorkspacesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureQuantumListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureQuantumListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureQuantumValidWorkspaceName(name) {
		respondAzureInvalidRequest(w, path, "workspace name is invalid")
		return
	}

	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureQuantumGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureQuantumGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureQuantumCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureQuantumPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureQuantumDelete(w, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	if len(tail) == 3 {
		switch strings.ToLower(strings.TrimSpace(tail[2])) {
		case "listkeys":
			if r.Method == http.MethodPost {
				s.handleAzureQuantumListKeys(w, path, subscriptionID, resourceGroup, name)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		case "regeneratekeys":
			if r.Method == http.MethodPost {
				s.handleAzureQuantumRegenerateKeys(w, r, path, subscriptionID, resourceGroup, name)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureQuantumCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureQuantumReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureQuantumString(payload["location"]))
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	workspace := s.azureQuantumWorkspaces[key]
	status := http.StatusOK
	createdAt := now
	primaryKey := azureQuantumStableKey("primary", subscriptionID, resourceGroup, name, now)
	secondaryKey := azureQuantumStableKey("secondary", subscriptionID, resourceGroup, name, now)
	if workspace == nil {
		workspace = &azureQuantumWorkspace{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
			PrimaryKey:     primaryKey,
			SecondaryKey:   secondaryKey,
		}
		status = http.StatusCreated
	} else {
		createdAt = workspace.CreatedAt
		primaryKey = workspace.PrimaryKey
		secondaryKey = workspace.SecondaryKey
	}

	workspace.Location = location
	workspace.Identity = azureQuantumObject(payload["identity"])
	workspace.Tags = azureQuantumStringMap(payload["tags"])
	workspace.Properties = azureQuantumNormalizeWorkspaceProperties(name, subscriptionID, resourceGroup, location, azureQuantumObject(payload["properties"]), createdAt, now)
	workspace.PrimaryKey = primaryKey
	workspace.SecondaryKey = secondaryKey
	workspace.CreatedAt = createdAt
	workspace.UpdatedAt = now
	workspace.ETag = azureQuantumETag(key, body, now)
	s.azureQuantumWorkspaces[key] = workspace

	if status == http.StatusCreated {
		w.Header().Set("Azure-AsyncOperation", "/azure/providers/Microsoft.Quantum/operations/workspace-"+name+"?api-version="+azureQuantumAPIVersion)
	}
	respondJSON(w, status, azureQuantumWorkspaceFixture(workspace))
}

func (s *Server) handleAzureQuantumPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureQuantumReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	workspace := s.azureQuantumWorkspaces[key]
	if workspace == nil {
		respondJSON(w, http.StatusNotFound, azureQuantumNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["tags"]; ok {
		workspace.Tags = azureQuantumStringMap(payload["tags"])
		updated = true
	}
	if rawIdentity, ok := payload["identity"]; ok {
		workspace.Identity = azureQuantumObject(rawIdentity)
		updated = true
	}
	if rawProps, ok := payload["properties"]; ok {
		workspace.Properties = azureQuantumMergeObjects(workspace.Properties, azureQuantumObject(rawProps))
		workspace.Properties = azureQuantumNormalizeWorkspaceProperties(name, subscriptionID, resourceGroup, workspace.Location, workspace.Properties, workspace.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	workspace.UpdatedAt = now
	workspace.ETag = azureQuantumETag(key, body, now)
	respondJSON(w, http.StatusOK, azureQuantumWorkspaceFixture(workspace))
}

func (s *Server) handleAzureQuantumGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	workspace := s.azureQuantumWorkspaces[azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if workspace == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureQuantumNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureQuantumWorkspaceFixture(workspace))
}

func (s *Server) handleAzureQuantumDelete(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	key := azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	_, exists := s.azureQuantumWorkspaces[key]
	if exists {
		delete(s.azureQuantumWorkspaces, key)
	}
	s.providerStorageMu.Unlock()

	if !exists {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Location", "/azure"+azureQuantumWorkspaceID(subscriptionID, resourceGroup, name)+"?api-version="+azureQuantumAPIVersion)
	w.Header().Set("Retry-After", "10")
	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) handleAzureQuantumListKeys(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string) {
	s.providerStorageMu.Lock()
	workspace := s.azureQuantumWorkspaces[azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if workspace == nil {
		respondJSON(w, http.StatusNotFound, azureQuantumNotFound(path, name))
		return
	}
	respondJSON(w, http.StatusOK, azureQuantumWorkspaceKeysFixture(workspace))
}

func (s *Server) handleAzureQuantumRegenerateKeys(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureQuantumReadRequestBody(w, r, path)
	if !ok {
		return
	}

	keyName := strings.TrimSpace(azureQuantumString(payload["keyName"]))
	if keyName == "" {
		keyName = strings.TrimSpace(azureQuantumString(payload["keyType"]))
	}
	if keyName == "" {
		respondAzureInvalidRequest(w, path, "keyName is required")
		return
	}

	key := azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	workspace := s.azureQuantumWorkspaces[key]
	if workspace == nil {
		respondJSON(w, http.StatusNotFound, azureQuantumNotFound(path, name))
		return
	}

	switch strings.ToLower(keyName) {
	case "primary":
		workspace.PrimaryKey = azureQuantumStableKey("primary", subscriptionID, resourceGroup, name, now)
	case "secondary":
		workspace.SecondaryKey = azureQuantumStableKey("secondary", subscriptionID, resourceGroup, name, now)
	default:
		respondAzureInvalidRequest(w, path, "keyName must be Primary or Secondary")
		return
	}
	workspace.UpdatedAt = now
	workspace.ETag = azureQuantumETag(key, body, now)
	respondJSON(w, http.StatusOK, azureQuantumWorkspaceKeysFixture(workspace))
}

func (s *Server) handleAzureQuantumListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, workspace := range s.azureQuantumWorkspaces {
		if workspace.SubscriptionID != subscriptionID {
			continue
		}
		values = append(values, azureQuantumWorkspaceFixture(workspace))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureQuantumString(values[i]["id"]) < azureQuantumString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureQuantumListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, workspace := range s.azureQuantumWorkspaces {
		if workspace.SubscriptionID != subscriptionID || !strings.EqualFold(workspace.ResourceGroup, resourceGroup) {
			continue
		}
		values = append(values, azureQuantumWorkspaceFixture(workspace))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureQuantumString(values[i]["id"]) < azureQuantumString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func azureQuantumOperationsFixture() []map[string]any {
	operations := []string{
		"Microsoft.Quantum/Workspaces/Read",
		"Microsoft.Quantum/Workspaces/Write",
		"Microsoft.Quantum/Workspaces/Delete",
		"Microsoft.Quantum/Workspaces/ListKeys/Action",
		"Microsoft.Quantum/Workspaces/RegenerateKeys/Action",
		"Microsoft.Quantum/Locations/Offerings/Read",
	}
	out := make([]map[string]any, 0, len(operations))
	for _, name := range operations {
		out = append(out, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider": "Microsoft.Quantum",
				"resource": strings.TrimPrefix(name, "Microsoft.Quantum/"),
			},
		})
	}
	return out
}

func azureQuantumOfferingsFixture(subscriptionID, location string) []map[string]any {
	base := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Quantum/locations/%s/offerings", subscriptionID, location)
	return []map[string]any{
		{
			"id":          base + "/ionq-simulator",
			"name":        "ionq-simulator",
			"displayName": "IonQ Simulator",
			"providerId":  "ionq",
			"providerSku": "simulator",
			"targets": []any{
				map[string]any{"id": "ionq.simulator", "currentAvailability": "Available"},
			},
			"quotaDimensions": []any{
				map[string]any{"id": "shots", "unit": "Count", "limit": 100000},
			},
			"pricingDimensions": []any{
				map[string]any{"id": "execution", "unit": "Job", "estimatedPrice": 0},
			},
		},
		{
			"id":          base + "/microsoft-estimator",
			"name":        "microsoft-estimator",
			"displayName": "Microsoft Resource Estimator",
			"providerId":  "microsoft-qc",
			"providerSku": "estimator",
			"targets": []any{
				map[string]any{"id": "microsoft.estimator", "currentAvailability": "Available"},
			},
			"quotaDimensions": []any{
				map[string]any{"id": "jobs", "unit": "Count", "limit": 1000},
			},
			"pricingDimensions": []any{
				map[string]any{"id": "execution", "unit": "Job", "estimatedPrice": 0},
			},
		},
	}
}

func azureQuantumWorkspaceFixture(workspace *azureQuantumWorkspace) map[string]any {
	return map[string]any{
		"id":         azureQuantumWorkspaceID(workspace.SubscriptionID, workspace.ResourceGroup, workspace.Name),
		"name":       workspace.Name,
		"type":       "Microsoft.Quantum/workspaces",
		"location":   workspace.Location,
		"identity":   azureQuantumMergeObjects(map[string]any{}, workspace.Identity),
		"tags":       azureQuantumAnyStringMap(workspace.Tags),
		"systemData": azureQuantumSystemData(workspace.CreatedAt, workspace.UpdatedAt),
		"properties": azureQuantumMergeObjects(map[string]any{}, workspace.Properties),
	}
}

func azureQuantumWorkspaceKeysFixture(workspace *azureQuantumWorkspace) map[string]any {
	endpointURI := azureQuantumString(workspace.Properties["endpointUri"])
	workspaceID := azureQuantumString(workspace.Properties["workspaceId"])
	return map[string]any{
		"primaryKey":                workspace.PrimaryKey,
		"secondaryKey":              workspace.SecondaryKey,
		"primaryConnectionString":   fmt.Sprintf("Endpoint=%s;WorkspaceId=%s;AccessKey=%s", endpointURI, workspaceID, workspace.PrimaryKey),
		"secondaryConnectionString": fmt.Sprintf("Endpoint=%s;WorkspaceId=%s;AccessKey=%s", endpointURI, workspaceID, workspace.SecondaryKey),
	}
}

func azureQuantumNormalizeWorkspaceProperties(name, subscriptionID, resourceGroup, location string, payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureQuantumMergeObjects(map[string]any{}, payload)
	if _, ok := props["apiKeyEnabled"]; !ok {
		props["apiKeyEnabled"] = true
	}
	if azureQuantumString(props["workspaceDescription"]) == "" {
		props["workspaceDescription"] = "Stackyard Azure Quantum workspace"
	}
	if _, ok := props["providers"]; !ok {
		props["providers"] = []any{
			map[string]any{"providerId": "ionq", "providerSku": "simulator"},
			map[string]any{"providerId": "microsoft-qc", "providerSku": "estimator"},
		}
	}
	props["workspaceId"] = azureQuantumWorkspaceGUID(subscriptionID, resourceGroup, name)
	props["endpointUri"] = fmt.Sprintf("https://%s.%s.quantum.azure.example", strings.ToLower(name), strings.ToLower(location))
	props["provisioningState"] = "Succeeded"
	props["createdDateTime"] = createdAt.Format(time.RFC3339)
	props["lastModifiedDateTime"] = updatedAt.Format(time.RFC3339)
	return props
}

func azureQuantumWorkspaceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "::" + resourceGroup + "::" + name)
}

func azureQuantumWorkspaceID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Quantum/workspaces/%s", subscriptionID, resourceGroup, name)
}

func azureQuantumWorkspaceGUID(subscriptionID, resourceGroup, name string) string {
	sum := sha1.Sum([]byte(azureQuantumWorkspaceID(subscriptionID, resourceGroup, name)))
	hexValue := hex.EncodeToString(sum[:16])
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}

func azureQuantumStableKey(prefix, subscriptionID, resourceGroup, name string, now time.Time) string {
	sum := sha1.Sum([]byte(prefix + "::" + azureQuantumWorkspaceID(subscriptionID, resourceGroup, name) + "::" + now.Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:])
}

func azureQuantumSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azureQuantumNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("resource %q was not found for path %s", name, path),
		},
	}
}

func azureQuantumReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureQuantumString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureQuantumObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return map[string]any{}
	}
	return azureQuantumMergeObjects(map[string]any{}, typed)
}

func azureQuantumStringMap(value any) map[string]string {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if str := azureQuantumString(val); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureQuantumAnyStringMap(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, val := range value {
		out[key] = val
	}
	return out
}

func azureQuantumMergeObjects(base, overlay map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		if overlayMap, ok := value.(map[string]any); ok {
			if baseMap, ok := merged[key].(map[string]any); ok {
				merged[key] = azureQuantumMergeObjects(baseMap, overlayMap)
			} else {
				merged[key] = azureQuantumMergeObjects(map[string]any{}, overlayMap)
			}
			continue
		}
		merged[key] = value
	}
	return merged
}

func azureQuantumETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureQuantumValidWorkspaceName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	for idx, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
			if idx == 0 || idx == len(name)-1 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
