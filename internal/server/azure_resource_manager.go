package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// azureARMResourceType describes an ARM tracked resource served by the shared
// handler. Registering a type gives it PUT/GET/HEAD/PATCH/DELETE plus list by
// resource group and by subscription, backed by one store that cascades with
// resource group deletion.
type azureARMResourceType struct {
	Namespace string // e.g. "Microsoft.ManagedIdentity"
	Type      string // e.g. "userAssignedIdentities"
	// ReadOnlyProperties returns server-owned properties for a resource ID.
	// They override anything the client sends. Optional.
	ReadOnlyProperties func(id string) map[string]any
	// Validate returns a non-empty message to reject a PUT payload. Optional.
	Validate func(payload map[string]any) string
}

func (t azureARMResourceType) fullType() string { return t.Namespace + "/" + t.Type }

// azureARMResource is immutable once stored: updates build a new record.
type azureARMResource struct {
	ID             string
	SubscriptionID string
	ResourceGroup  string
	Name           string
	Type           string
	Location       string
	Kind           string
	Tags           map[string]any
	Identity       map[string]any
	SKU            map[string]any
	Properties     map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ponytail: registration happens in init() only, so the map needs no lock.
var azureARMResourceTypes = map[string]azureARMResourceType{}

func registerAzureARMResourceType(t azureARMResourceType) {
	key := strings.ToLower(t.fullType())
	if _, exists := azureARMResourceTypes[key]; exists {
		panic("azure ARM resource type registered twice: " + t.fullType())
	}
	azureARMResourceTypes[key] = t
}

// azureARMServerProperties are the properties clients can never overwrite.
func azureARMServerProperties(typ azureARMResourceType, id string) map[string]any {
	props := map[string]any{"provisioningState": "Succeeded"}
	if typ.ReadOnlyProperties != nil {
		props = azureARMMerge(props, typ.ReadOnlyProperties(id))
	}
	return props
}

func lookupAzureARMResourceType(namespace, typ string) (azureARMResourceType, bool) {
	t, ok := azureARMResourceTypes[strings.ToLower(namespace+"/"+typ)]
	return t, ok
}

// handleAzureARMRouter serves top-level tracked resources of registered types.
// ponytail: nested child resource types are not routed yet; add a parent
// segment to azureARMResourceType when the first service needs one.
func (s *Server) handleAzureARMRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	var sub, rg, name string
	var typ azureARMResourceType
	var ok bool
	switch {
	case len(seg) == 5 && strings.EqualFold(seg[2], "providers"):
		sub = seg[1]
		typ, ok = lookupAzureARMResourceType(seg[3], seg[4])
	case len(seg) >= 7 && len(seg) <= 8 && strings.EqualFold(seg[2], "resourceGroups") && strings.EqualFold(seg[4], "providers"):
		sub, rg = seg[1], seg[3]
		typ, ok = lookupAzureARMResourceType(seg[5], seg[6])
		if len(seg) == 8 {
			name = seg[7]
		}
	}
	if !ok {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	if name == "" {
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		s.listAzureARMResources(w, typ, sub, rg)
		return true
	}

	id := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", sub, rg, typ.fullType(), name)
	switch r.Method {
	case http.MethodPut:
		s.putAzureARMResource(w, r, typ, id, sub, rg, name)
	case http.MethodPatch:
		s.patchAzureARMResource(w, r, typ, id, sub, rg)
	case http.MethodGet, http.MethodHead:
		record, code := s.getAzureARMResource(id, sub, rg)
		switch {
		case record != nil && r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNoContent)
		case record != nil:
			respondJSON(w, http.StatusOK, record.fixture())
		case r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		default:
			azureARMNotFound(w, code, id)
		}
	case http.MethodDelete:
		s.providerStorageMu.Lock()
		_, existed := s.azureARMResources[azureARMKey(id)]
		delete(s.azureARMResources, azureARMKey(id))
		s.providerStorageMu.Unlock()
		if existed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
	return true
}

func (s *Server) putAzureARMResource(w http.ResponseWriter, r *http.Request, typ azureARMResourceType, id, sub, rg, name string) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	location := strings.TrimSpace(azureARMString(payload["location"]))
	if location == "" {
		azureARMError(w, http.StatusBadRequest, "LocationRequired", "The location property is required for this definition.")
		return
	}
	if typ.Validate != nil {
		if msg := typ.Validate(payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return
		}
	}

	now := time.Now().UTC()
	record := &azureARMResource{
		ID: id, SubscriptionID: sub, ResourceGroup: rg, Name: name, Type: typ.fullType(),
		Location:   location,
		Kind:       azureARMString(payload["kind"]),
		Tags:       azureARMObject(payload["tags"]),
		Identity:   azureARMObject(payload["identity"]),
		SKU:        azureARMObject(payload["sku"]),
		Properties: azureARMMerge(azureARMObject(payload["properties"]), azureARMServerProperties(typ, id)),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.providerStorageMu.Lock()
	group := s.azureResourceGroups[azureResourceGroupKey(sub, rg)]
	if group == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", id)
		return
	}
	existing := s.azureARMResources[azureARMKey(id)]
	switch {
	case existing != nil && !strings.EqualFold(existing.Location, location):
		s.providerStorageMu.Unlock()
		azureARMError(w, http.StatusConflict, "InvalidResourceLocation",
			fmt.Sprintf("The resource '%s' already exists in location '%s'. A resource with the same name cannot be created in location '%s'.", existing.ID, existing.Location, location))
		return
	case existing != nil:
		// Keep the casing the resource was created with, like ARM does.
		record.ID, record.SubscriptionID, record.ResourceGroup, record.Name = existing.ID, existing.SubscriptionID, existing.ResourceGroup, existing.Name
		record.CreatedAt = existing.CreatedAt
	default:
		record.ResourceGroup = group.Name
		record.ID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", group.SubscriptionID, group.Name, typ.fullType(), name)
	}
	s.azureARMResources[azureARMKey(id)] = record
	s.providerStorageMu.Unlock()

	status := http.StatusCreated
	if existing != nil {
		status = http.StatusOK
	}
	respondJSON(w, status, record.fixture())
}

func (s *Server) patchAzureARMResource(w http.ResponseWriter, r *http.Request, typ azureARMResourceType, id, sub, rg string) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	existing, code := s.lookupAzureARMResourceLocked(id, sub, rg)
	if existing == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, code, id)
		return
	}
	updated := *existing
	if tags, present := payload["tags"]; present {
		updated.Tags = azureARMObject(tags)
	}
	if identity := azureARMObject(payload["identity"]); len(identity) != 0 {
		updated.Identity = identity
	}
	if sku := azureARMObject(payload["sku"]); len(sku) != 0 {
		updated.SKU = sku
	}
	updated.Properties = azureARMMerge(
		azureARMMerge(existing.Properties, azureARMObject(payload["properties"])),
		azureARMServerProperties(typ, id),
	)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &updated
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, updated.fixture())
}

func (s *Server) getAzureARMResource(id, sub, rg string) (*azureARMResource, string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.lookupAzureARMResourceLocked(id, sub, rg)
}

// lookupAzureARMResourceLocked returns the record, or nil and the ARM error
// code explaining why it is missing. Caller holds providerStorageMu.
func (s *Server) lookupAzureARMResourceLocked(id, sub, rg string) (*azureARMResource, string) {
	if s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		return nil, "ResourceGroupNotFound"
	}
	if record := s.azureARMResources[azureARMKey(id)]; record != nil {
		return record, ""
	}
	return nil, "ResourceNotFound"
}

func (s *Server) listAzureARMResources(w http.ResponseWriter, typ azureARMResourceType, sub, rg string) {
	s.providerStorageMu.Lock()
	if rg != "" && s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", "/subscriptions/"+sub+"/resourceGroups/"+rg)
		return
	}
	records := s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, typ.fullType()) && strings.EqualFold(rec.SubscriptionID, sub) &&
			(rg == "" || strings.EqualFold(rec.ResourceGroup, rg))
	})
	s.providerStorageMu.Unlock()
	respondAzureARMList(w, records)
}

// azureARMResourcesMatchingLocked returns matching records sorted by ID.
// Caller holds providerStorageMu.
func (s *Server) azureARMResourcesMatchingLocked(match func(*azureARMResource) bool) []*azureARMResource {
	out := make([]*azureARMResource, 0)
	for _, rec := range s.azureARMResources {
		if match(rec) {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return azureARMKey(out[i].ID) < azureARMKey(out[j].ID) })
	return out
}

func respondAzureARMList(w http.ResponseWriter, records []*azureARMResource) {
	items := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		items = append(items, rec.fixture())
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (rec *azureARMResource) fixture() map[string]any {
	out := map[string]any{
		"id":         rec.ID,
		"name":       rec.Name,
		"type":       rec.Type,
		"location":   rec.Location,
		"tags":       azureARMMerge(rec.Tags, nil),
		"properties": rec.Properties,
		"systemData": azureARMSystemData(rec.CreatedAt, rec.UpdatedAt),
	}
	if rec.Kind != "" {
		out["kind"] = rec.Kind
	}
	if len(rec.Identity) != 0 {
		out["identity"] = rec.Identity
	}
	if len(rec.SKU) != 0 {
		out["sku"] = rec.SKU
	}
	return out
}

func azureARMSystemData(created, updated time.Time) map[string]any {
	return map[string]any{
		"createdBy":          "stackyard",
		"createdByType":      "Application",
		"createdAt":          created.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "Application",
		"lastModifiedAt":     updated.Format(time.RFC3339),
	}
}

func azureARMKey(id string) string { return strings.ToLower(id) }

// azureARMReadBody decodes a JSON object body; an empty body is an empty object.
func azureARMReadBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "unable to read request body")
		return nil, false
	}
	payload := map[string]any{}
	if strings.TrimSpace(string(body)) == "" {
		return payload, true
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload == nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid and could not be deserialized.")
		return nil, false
	}
	return payload, true
}

// azureARMError writes the ARM error envelope Azure SDKs parse.
func azureARMError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("x-ms-error-code", code)
	respondJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}

func azureARMNotFound(w http.ResponseWriter, code, id string) {
	if code == "ResourceGroupNotFound" {
		azureARMError(w, http.StatusNotFound, code, "Resource group for '"+id+"' could not be found.")
		return
	}
	azureARMError(w, http.StatusNotFound, code, "The resource '"+id+"' was not found.")
}

func azureARMString(v any) string {
	s, _ := v.(string)
	return s
}

func azureARMObject(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// azureARMMerge returns a new map with overlay applied on top of base,
// recursing into nested objects. Neither input is modified.
func azureARMMerge(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		if nested, ok := v.(map[string]any); ok {
			existing, _ := out[k].(map[string]any)
			out[k] = azureARMMerge(existing, nested)
			continue
		}
		out[k] = v
	}
	return out
}
