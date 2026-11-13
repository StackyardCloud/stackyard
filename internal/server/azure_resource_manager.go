package server

import (
	"crypto/sha256"
	"encoding/base64"
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
	Namespace string // e.g. "Microsoft.ContainerRegistry"
	// Type is the resource type path; nested child types include their
	// parents, e.g. "registries" or "registries/webhooks".
	Type string
	// Proxy marks ARM proxy resources, which have no location.
	Proxy bool
	// Extension marks extension resources, which attach to any scope (a
	// subscription, resource group or resource) at
	// {scope}/providers/{Namespace}/{Type}/{name}. They are proxy resources,
	// and Type must be a single segment.
	Extension bool
	// ReadOnlyProperties returns server-owned properties for a record whose
	// Properties still hold only what the client sent. They override anything
	// the client sends. Optional.
	ReadOnlyProperties func(rec *azureARMResource) map[string]any
	// ValidateName returns a non-empty message to reject a resource name. Optional.
	ValidateName func(name string) string
	// Validate returns a non-empty message to reject a PUT payload. Optional.
	Validate func(payload map[string]any) string
	// Actions are POST operations on an existing resource, keyed by the
	// lower-cased action segment, e.g. "listcredentials". Optional.
	Actions map[string]azureARMAction
	// Transitions are POST operations that merge server-owned properties
	// into the stored resource and return 200 with no body, keyed like
	// Actions, e.g. "stop": {"state": "Stopped"}. Optional.
	Transitions map[string]map[string]any
	// WriteOnlyProperties are dotted property paths accepted on PUT and
	// PATCH but never stored or returned, such as passwords. Optional.
	WriteOnlyProperties []string
	// DefaultChildren returns child resources ARM creates along with a new
	// resource, e.g. a default authorization rule. Optional.
	DefaultChildren func(parent *azureARMResource) []*azureARMResource
}

// azureARMAction handles POST {resourceId}/{action} and returns the status
// and JSON body. The record is immutable and must not be modified.
type azureARMAction func(record *azureARMResource, payload map[string]any) (int, any)

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
	if t.Extension {
		t.Proxy = true
	}
	key := strings.ToLower(t.fullType())
	if _, exists := azureARMResourceTypes[key]; exists {
		panic("azure ARM resource type registered twice: " + t.fullType())
	}
	azureARMResourceTypes[key] = t
}

// azureARMProperties returns the client properties with the server-owned
// ones, which clients can never overwrite, applied on top.
func azureARMProperties(typ azureARMResourceType, rec *azureARMResource) map[string]any {
	server := map[string]any{"provisioningState": "Succeeded"}
	if typ.ReadOnlyProperties != nil {
		server = azureARMMerge(server, typ.ReadOnlyProperties(rec))
	}
	props := azureARMMerge(rec.Properties, server)
	for _, path := range typ.WriteOnlyProperties {
		props = azureARMWithout(props, path)
	}
	return props
}

// azureARMWithout returns props without the dotted path, copying the maps
// along it so props itself is not modified.
func azureARMWithout(props map[string]any, path string) map[string]any {
	key, rest, nested := strings.Cut(path, ".")
	child, present := props[key]
	if !present {
		return props
	}
	out := azureARMMerge(props, nil)
	if !nested {
		delete(out, key)
	} else if childMap, ok := child.(map[string]any); ok {
		out[key] = azureARMWithout(childMap, rest)
	}
	return out
}

// azureARMProviderAction handles POST .../providers/{namespace}/{action},
// such as a name availability check. Actions registered as
// "locations/{action}" are served at
// /subscriptions/{sub}/providers/{namespace}/locations/{location}/{action}.
type azureARMProviderAction func(s *Server, payload map[string]any) (int, any)

// ponytail: registered in init() only, like azureARMResourceTypes.
var azureARMProviderActions = map[string]azureARMProviderAction{}

func registerAzureARMProviderAction(namespace, action string, fn azureARMProviderAction) {
	key := strings.ToLower(namespace + "/" + action)
	if _, exists := azureARMProviderActions[key]; exists {
		panic("azure ARM provider action registered twice: " + namespace + "/" + action)
	}
	azureARMProviderActions[key] = fn
}

func lookupAzureARMResourceType(namespace, typ string) (azureARMResourceType, bool) {
	t, ok := azureARMResourceTypes[strings.ToLower(namespace+"/"+typ)]
	return t, ok
}

// handleAzureARMRouter serves registered resource types, including nested
// child types, at subscription, resource group and parent-resource scope.
func (s *Server) handleAzureARMRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/subscriptions/") && !strings.HasPrefix(path, "/azure/providers/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if s.handleAzureARMExtension(w, r, seg) {
		return true
	}
	var sub, rg, namespace string
	var rest []string
	switch {
	case len(seg) == 3 && strings.EqualFold(seg[0], "providers"):
		return s.invokeAzureARMProviderAction(w, r, seg[1], seg[2])
	case len(seg) == 5 && strings.EqualFold(seg[2], "providers") && azureARMProviderActions[strings.ToLower(seg[3]+"/"+seg[4])] != nil:
		return s.invokeAzureARMProviderAction(w, r, seg[3], seg[4])
	case len(seg) == 7 && strings.EqualFold(seg[2], "providers") && strings.EqualFold(seg[4], "locations"):
		return s.invokeAzureARMProviderAction(w, r, seg[3], "locations/"+seg[6])
	case len(seg) < 2 || !strings.EqualFold(seg[0], "subscriptions"):
		return false
	case len(seg) == 5 && strings.EqualFold(seg[2], "providers"):
		sub, namespace, rest = seg[1], seg[3], seg[4:]
	case len(seg) >= 7 && strings.EqualFold(seg[2], "resourceGroups") && strings.EqualFold(seg[4], "providers"):
		sub, rg, namespace, rest = seg[1], seg[3], seg[5], seg[6:]
	default:
		return false
	}

	// POST {resourceId}/{action}: the trailing segment is an action, not a type.
	var action azureARMAction
	var transition map[string]any
	if r.Method == http.MethodPost && len(rest) >= 3 && len(rest)%2 == 1 {
		typ, ok := lookupAzureARMResourceType(namespace, azureARMTypePath(rest[:len(rest)-1]))
		verb := strings.ToLower(rest[len(rest)-1])
		if !ok || (typ.Actions[verb] == nil && typ.Transitions[verb] == nil) {
			return false
		}
		action, transition = typ.Actions[verb], typ.Transitions[verb]
		rest = rest[:len(rest)-1]
	}
	typ, ok := lookupAzureARMResourceType(namespace, azureARMTypePath(rest))
	if !ok {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}

	// Rebuild the ID with the registered type casing; names keep client casing.
	scope := "/subscriptions/" + sub
	if rg != "" {
		scope += "/resourceGroups/" + rg
	}
	// ids[i] is the resource ID after the i-th type/name pair.
	typeParts := strings.Split(typ.Type, "/")
	ids := make([]string, 0, len(typeParts))
	id := scope + "/providers/" + typ.Namespace
	for i := 1; i < len(rest); i += 2 {
		id += "/" + typeParts[i/2] + "/" + rest[i]
		ids = append(ids, id)
	}

	if len(rest)%2 == 1 {
		if r.Method != http.MethodGet {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return true
		}
		parentID := ""
		if len(ids) > 0 {
			parentID = ids[len(ids)-1]
		}
		s.listAzureARMResources(w, typ, sub, rg, parentID, scope)
		return true
	}
	parentID := ""
	if len(ids) > 1 {
		parentID = ids[len(ids)-2]
	}
	name := rest[len(rest)-1]

	if action != nil {
		s.invokeAzureARMAction(w, r, action, id, sub, rg, parentID)
		return true
	}
	if transition != nil {
		s.transitionAzureARMResource(w, transition, id, sub, rg, parentID)
		return true
	}
	s.serveAzureARMItem(w, r, typ, id, sub, rg, parentID, name)
	return true
}

// serveAzureARMItem handles the CRUD verbs on one resource. rg is empty only
// for extension resources at subscription scope.
func (s *Server) serveAzureARMItem(w http.ResponseWriter, r *http.Request, typ azureARMResourceType, id, sub, rg, parentID, name string) {
	switch r.Method {
	case http.MethodPut:
		s.putAzureARMResource(w, r, typ, id, sub, rg, parentID, name)
	case http.MethodPatch:
		s.patchAzureARMResource(w, r, typ, id, sub, rg, parentID)
	case http.MethodGet, http.MethodHead:
		record, code := s.getAzureARMResource(id, sub, rg, parentID)
		switch {
		case record != nil && r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNoContent)
		case record != nil:
			respondJSON(w, http.StatusOK, record.fixture())
		case r.Method == http.MethodHead:
			w.WriteHeader(http.StatusNotFound)
		default:
			azureARMNotFound(w, code, id, parentID)
		}
	case http.MethodDelete:
		s.providerStorageMu.Lock()
		_, existed := s.azureARMResources[azureARMKey(id)]
		delete(s.azureARMResources, azureARMKey(id))
		// Child and extension resources go with their parent.
		prefix := azureARMKey(id + "/")
		for k := range s.azureARMResources {
			if strings.HasPrefix(k, prefix) {
				delete(s.azureARMResources, k)
			}
		}
		s.providerStorageMu.Unlock()
		if existed {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

// handleAzureARMExtension serves extension resources: the segments after the
// last "providers" name a registered extension type, and everything before
// it is the scope. The scope resource itself is not required to exist,
// because it may belong to a service with its own store.
func (s *Server) handleAzureARMExtension(w http.ResponseWriter, r *http.Request, seg []string) bool {
	i := len(seg) - 1
	for i >= 0 && !strings.EqualFold(seg[i], "providers") {
		i--
	}
	if i < 2 || !strings.EqualFold(seg[0], "subscriptions") || len(seg)-i < 3 || len(seg)-i > 4 {
		return false
	}
	typ, ok := lookupAzureARMResourceType(seg[i+1], seg[i+2])
	if !ok || !typ.Extension {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	sub, rg := seg[1], ""
	if i >= 4 && strings.EqualFold(seg[2], "resourceGroups") {
		rg = seg[3]
	}
	collection := "/" + strings.Join(seg[:i], "/") + "/providers/" + typ.Namespace + "/" + typ.Type
	if len(seg)-i == 4 {
		s.serveAzureARMItem(w, r, typ, collection+"/"+seg[i+3], sub, rg, "", seg[i+3])
		return true
	}
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	s.providerStorageMu.Lock()
	if rg != "" && s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", collection, "")
		return true
	}
	// Only settings attached directly to this scope, not to resources under it.
	prefix := azureARMKey(collection + "/")
	records := s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		key := azureARMKey(rec.ID)
		return strings.EqualFold(rec.Type, typ.fullType()) && strings.HasPrefix(key, prefix) && !strings.Contains(key[len(prefix):], "/")
	})
	s.providerStorageMu.Unlock()
	respondAzureARMList(w, records)
	return true
}

// azureARMTypePath returns the type segments (even indexes) joined by "/".
func azureARMTypePath(rest []string) string {
	types := make([]string, 0, (len(rest)+1)/2)
	for i := 0; i < len(rest); i += 2 {
		types = append(types, rest[i])
	}
	return strings.Join(types, "/")
}

func (s *Server) invokeAzureARMAction(w http.ResponseWriter, r *http.Request, action azureARMAction, id, sub, rg, parentID string) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	record, code := s.getAzureARMResource(id, sub, rg, parentID)
	if record == nil {
		azureARMNotFound(w, code, id, parentID)
		return
	}
	status, body := action(record, payload)
	respondAzureARMAction(w, status, body)
}

// transitionAzureARMResource stores a copy of the resource with the
// transition's properties merged in.
func (s *Server) transitionAzureARMResource(w http.ResponseWriter, transition map[string]any, id, sub, rg, parentID string) {
	s.providerStorageMu.Lock()
	existing, code := s.lookupAzureARMResourceLocked(id, sub, rg, parentID)
	if existing == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, code, id, parentID)
		return
	}
	updated := *existing
	updated.Properties = azureARMMerge(existing.Properties, transition)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &updated
	s.providerStorageMu.Unlock()
	respondAzureARMAction(w, http.StatusOK, nil)
}

func (s *Server) invokeAzureARMProviderAction(w http.ResponseWriter, r *http.Request, namespace, name string) bool {
	action := azureARMProviderActions[strings.ToLower(namespace+"/"+name)]
	if action == nil {
		return false
	}
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return true
	}
	status, body := action(s, payload)
	respondAzureARMAction(w, status, body)
	return true
}

// azureARMResourcesWhere returns matching records across all subscriptions,
// for checks of globally unique names.
func (s *Server) azureARMResourcesWhere(match func(*azureARMResource) bool) []*azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azureARMResourcesMatchingLocked(match)
}

// azureARMChild builds a child record of parent, for DefaultChildren hooks.
func azureARMChild(parent *azureARMResource, typ azureARMResourceType, name string, properties map[string]any) *azureARMResource {
	id := parent.ID + "/" + typ.Type[strings.LastIndex(typ.Type, "/")+1:] + "/" + name
	child := &azureARMResource{
		ID: id, SubscriptionID: parent.SubscriptionID, ResourceGroup: parent.ResourceGroup, Name: name, Type: typ.fullType(),
		Properties: properties,
		CreatedAt:  parent.CreatedAt,
		UpdatedAt:  parent.UpdatedAt,
	}
	child.Properties = azureARMProperties(typ, child)
	return child
}

func (s *Server) putAzureARMResource(w http.ResponseWriter, r *http.Request, typ azureARMResourceType, id, sub, rg, parentID, name string) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	if typ.ValidateName != nil {
		if msg := typ.ValidateName(name); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidResourceName", msg)
			return
		}
	}
	location := strings.TrimSpace(azureARMString(payload["location"]))
	if typ.Proxy {
		location = ""
	} else if location == "" {
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
		Properties: azureARMObject(payload["properties"]),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	record.Properties = azureARMProperties(typ, record)

	s.providerStorageMu.Lock()
	group := s.azureResourceGroups[azureResourceGroupKey(sub, rg)]
	if group == nil && rg != "" {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", id, "")
		return
	}
	var parent *azureARMResource
	if parentID != "" {
		if parent = s.azureARMResources[azureARMKey(parentID)]; parent == nil {
			s.providerStorageMu.Unlock()
			azureARMNotFound(w, "ParentResourceNotFound", id, parentID)
			return
		}
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
	case typ.Extension:
		// The ID keeps the scope as the client sent it.
	case parent != nil:
		record.ResourceGroup = parent.ResourceGroup
		record.ID = parent.ID + "/" + typ.Type[strings.LastIndex(typ.Type, "/")+1:] + "/" + name
	default:
		record.ResourceGroup = group.Name
		record.ID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", group.SubscriptionID, group.Name, typ.fullType(), name)
	}
	s.azureARMResources[azureARMKey(id)] = record
	if existing == nil && typ.DefaultChildren != nil {
		for _, child := range typ.DefaultChildren(record) {
			s.azureARMResources[azureARMKey(child.ID)] = child
		}
	}
	s.providerStorageMu.Unlock()

	status := http.StatusCreated
	if existing != nil {
		status = http.StatusOK
	}
	respondJSON(w, status, record.fixture())
}

func (s *Server) patchAzureARMResource(w http.ResponseWriter, r *http.Request, typ azureARMResourceType, id, sub, rg, parentID string) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	existing, code := s.lookupAzureARMResourceLocked(id, sub, rg, parentID)
	if existing == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, code, id, parentID)
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
	updated.Properties = azureARMMerge(existing.Properties, azureARMObject(payload["properties"]))
	updated.Properties = azureARMProperties(typ, &updated)
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(id)] = &updated
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, updated.fixture())
}

func (s *Server) getAzureARMResource(id, sub, rg, parentID string) (*azureARMResource, string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.lookupAzureARMResourceLocked(id, sub, rg, parentID)
}

// lookupAzureARMResourceLocked returns the record, or nil and the ARM error
// code explaining why it is missing. Caller holds providerStorageMu.
func (s *Server) lookupAzureARMResourceLocked(id, sub, rg, parentID string) (*azureARMResource, string) {
	if rg != "" && s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		return nil, "ResourceGroupNotFound"
	}
	if parentID != "" && s.azureARMResources[azureARMKey(parentID)] == nil {
		return nil, "ParentResourceNotFound"
	}
	if record := s.azureARMResources[azureARMKey(id)]; record != nil {
		return record, ""
	}
	return nil, "ResourceNotFound"
}

// listAzureARMResources lists typ under parentID when set, else under scope
// (a subscription or resource group ID).
func (s *Server) listAzureARMResources(w http.ResponseWriter, typ azureARMResourceType, sub, rg, parentID, scope string) {
	s.providerStorageMu.Lock()
	if rg != "" && s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", scope, "")
		return
	}
	if parentID != "" {
		if s.azureARMResources[azureARMKey(parentID)] == nil {
			s.providerStorageMu.Unlock()
			azureARMNotFound(w, "ParentResourceNotFound", parentID, parentID)
			return
		}
		scope = parentID
	}
	prefix := azureARMKey(scope + "/")
	records := s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, typ.fullType()) && strings.HasPrefix(azureARMKey(rec.ID), prefix)
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
		"tags":       azureARMMerge(rec.Tags, nil),
		"properties": rec.Properties,
		"systemData": azureARMSystemData(rec.CreatedAt, rec.UpdatedAt),
	}
	if rec.Location != "" {
		out["location"] = rec.Location
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

// azureARMDerivedKey returns a stable 44-character base64 access key derived
// from a resource ID, so keys survive restarts without being stored.
func azureARMDerivedKey(id, label string) string {
	sum := sha256.Sum256([]byte(azureARMKey(id) + "#" + label))
	return base64.StdEncoding.EncodeToString(sum[:])
}

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
	respondAzureARMAction(w, status, azureARMErrorBody(code, message))
}

func azureARMErrorBody(code, message string) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "message": message}}
}

// respondAzureARMAction writes an action result, mirroring an error body's
// code into the x-ms-error-code header. A nil body writes the status only.
func respondAzureARMAction(w http.ResponseWriter, status int, body any) {
	if errBody, _ := body.(map[string]any); errBody != nil {
		if code := azureARMString(azureARMObject(errBody["error"])["code"]); code != "" {
			w.Header().Set("x-ms-error-code", code)
		}
	}
	if body == nil {
		w.WriteHeader(status)
		return
	}
	respondJSON(w, status, body)
}

func azureARMNotFound(w http.ResponseWriter, code, id, parentID string) {
	switch code {
	case "ResourceGroupNotFound":
		azureARMError(w, http.StatusNotFound, code, "Resource group for '"+id+"' could not be found.")
		return
	case "ParentResourceNotFound":
		azureARMError(w, http.StatusNotFound, code, "Can not perform requested operation on nested resource. Parent resource '"+parentID+"' not found.")
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
