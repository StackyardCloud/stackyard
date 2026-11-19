package server

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
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
	// {scope}/providers/{Namespace}/{Type}/{name}. They are proxy
	// resources. Child types nested under one must set it too.
	Extension bool
	// ScopeStored makes an extension type's scope a resource that must be
	// in the store, such as a Log Analytics workspace: requests under a
	// missing one get 404 ParentResourceNotFound. Its resources are deleted
	// with it either way.
	ScopeStored bool
	// Tenant marks tenant resources, served at /providers/{Namespace}/{Type}/
	// {name} outside any subscription, such as subscription aliases. They
	// are proxy resources. Child types nested under a tenant type must set
	// it too.
	Tenant bool
	// Subscription marks types served at /subscriptions/{id}/providers/
	// {Namespace}/{Type}/{name}, outside any resource group, such as
	// support tickets. They are proxy resources, and child types nested
	// under one must set it too. A type may also be Tenant, to be served
	// at both scopes.
	Subscription bool
	// ValidateScope returns a non-empty message to reject the scope an
	// extension resource attaches to, such as a resource of the wrong type.
	// Optional; only for Extension types.
	ValidateScope func(scope string) string
	// ReadOnlySKU returns server-owned sku fields merged over the sku the
	// client sent on PUT, such as a tier derived from the sku name. Optional.
	ReadOnlySKU func(sku map[string]any) map[string]any
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
	// ServerActions are Actions given the server, for operations that read
	// or change other records, such as moving a collection's resources.
	// They run without the store lock held. Optional.
	ServerActions map[string]func(s *Server, rec *azureARMResource, payload map[string]any) (int, any)
	// StringBodyActions are the Actions or ServerActions, keyed like them,
	// whose request body is a bare JSON string; it arrives as
	// payload["body"]. Optional.
	StringBodyActions map[string]bool
	// Transitions are POST operations that merge server-owned properties
	// into the stored resource and return 200 with no body, keyed like
	// Actions, e.g. "stop": {"state": "Stopped"}. Optional.
	Transitions map[string]map[string]any
	// WriteOnlyProperties are dotted property paths accepted on PUT and
	// PATCH but never stored or returned, such as passwords. A segment
	// ending in "[]" applies the rest of the path to every element of that
	// array, e.g. "roles[].password". Optional.
	WriteOnlyProperties []string
	// StateProperties are properties only Transitions change, mapped to
	// the value a new resource starts with, e.g. {"state": "Ready"}, or
	// nil for a property a new resource lacks until a transition sets it.
	// Clients can't set them, and PUT and PATCH keep the stored value.
	// Optional.
	StateProperties map[string]any
	// DefaultChildren returns child resources ARM creates along with a new
	// resource, e.g. a default authorization rule. Optional.
	DefaultChildren func(parent *azureARMResource) []*azureARMResource
	// InlineChildren maps a property to the child type it embeds, e.g.
	// {"subnets": "virtualNetworks/subnets"}. A PUT that sets the property
	// replaces the child resources with its items; one that omits it (or
	// sends null) keeps them, rechecked with ValidateInParent. Responses
	// list the current children there, so inline and child resources stay
	// in sync.
	// Optional; see azure_resource_manager_inline.go.
	InlineChildren map[string]string
	// ChildProperties returns properties derived from the record's child
	// resources, such as a summary of them. Like inline properties, they
	// are recomputed whenever a child changes. Caller holds the store
	// lock. Optional.
	ChildProperties func(s *Server, rec *azureARMResource) map[string]any
	// ValidateInParent returns a non-empty message to reject a child PUT,
	// given its parent's properties (which list the child's siblings when
	// the parent embeds them) and the child's name, such as a subnet
	// outside its network's address space. Inline items are checked the
	// same way against the parent payload. Like Validate, it does not run
	// on PATCH. Optional.
	ValidateInParent func(parent map[string]any, name string, payload map[string]any) string
	// KeepLocation keeps the optional location a client sends to a proxy
	// or extension type, such as a maintenance configuration assignment.
	KeepLocation bool
	// CreateStatus is the status of a PUT that creates the resource, for
	// APIs that never answer 201, such as quota limits. Default 201.
	CreateStatus int
	// Linked runs on PUT, after validation and under the store lock (so it
	// may read the store but not lock it). It returns server-owned
	// properties for the record and resources ARM creates alongside it
	// outside its own ID, such as a private endpoint's network interface.
	// Those are stored with the record, dropped when a later PUT no longer
	// returns them, and deleted with it. A non-empty message rejects the
	// PUT with 400 InvalidRequestContent. The record it gets also holds the
	// WriteOnlyProperties the client sent, such as the purchase details a
	// reservation order turns into reservations. Optional.
	Linked func(s *Server, rec *azureARMResource) (props map[string]any, linked []*azureARMResource, msg string)
	// DeleteStatus is the status of a DELETE that removes the resource, for
	// APIs that never answer 200, such as Sentinel watchlists. Default 200.
	DeleteStatus int
	// NoDelete rejects DELETE with 405 for APIs that have none, such as
	// reservation orders, which are returned rather than deleted.
	NoDelete bool
	// ValidatePatch and ValidateDelete return a non-empty message to reject
	// a PATCH or DELETE with 409 Conflict, given the stored resource, such
	// as a Data Box job past the stage where it can change. They run under
	// the store lock, so ValidatePatch may read the store but not lock it.
	// Optional.
	ValidatePatch  func(s *Server, existing *azureARMResource, payload map[string]any) string
	ValidateDelete func(existing *azureARMResource) string
	// NoPatch rejects PATCH with 405 for APIs that have none. Types with
	// Linked don't accept PATCH either, unless they set PatchBody.
	NoPatch bool
	// PatchBody rewrites and checks a PATCH body for APIs whose update
	// model differs from the resource, such as support tickets' top-level
	// fields. It returns the body to merge, the StateProperties values the
	// update sets, or a non-empty message to reject it (400
	// InvalidRequestContent). Optional.
	PatchBody func(payload map[string]any) (body, state map[string]any, msg string)
	// Reconcile runs on PUT and PATCH under the store lock, once the
	// write's properties are final and before anything is stored, given
	// the stored record (nil on create). It returns server-owned
	// properties merged into the record, such as values derived from what
	// changed, or a non-empty message to reject the write with 400
	// InvalidRequestContent, such as a limit counted across the store. It
	// may read the store but not lock it. Optional.
	Reconcile func(s *Server, existing, rec *azureARMResource) (props map[string]any, msg string)
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
	// APIVersion is the api-version of the PUT that wrote the record, or
	// of its parent's for children ARM creates with it, such as the version
	// a Terraform export names. Empty for records custom routers store.
	APIVersion string
	// LinkedIDs are the resources a Linked hook created with this one.
	LinkedIDs []string
}

// ponytail: registration happens in init() only, so the map needs no lock.
var azureARMResourceTypes = map[string]azureARMResourceType{}

func registerAzureARMResourceType(t azureARMResourceType) {
	if t.Extension || t.Tenant || t.Subscription {
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
	key, each := strings.CutSuffix(key, "[]")
	child, present := props[key]
	if !present {
		return props
	}
	out := azureARMMerge(props, nil)
	items, isList := child.([]any)
	switch {
	case each && isList && nested:
		copied := make([]any, len(items))
		for i, item := range items {
			if itemMap, ok := item.(map[string]any); ok {
				item = azureARMWithout(itemMap, rest)
			}
			copied[i] = item
		}
		out[key] = copied
	case each:
		return props
	case !nested:
		delete(out, key)
	default:
		if childMap, ok := child.(map[string]any); ok {
			out[key] = azureARMWithout(childMap, rest)
		}
	}
	return out
}

// azureARMWithSent returns props with the value at the dotted path (as in
// WriteOnlyProperties) taken from sent, matching array items by index (so
// the server must keep the client's item order) and copying the maps along
// the path so props itself is not modified. Records never hold their
// write-only values, so taking the sent value whole loses nothing.
func azureARMWithSent(props, sent map[string]any, path string) map[string]any {
	key, rest, nested := strings.Cut(path, ".")
	key, each := strings.CutSuffix(key, "[]")
	value, present := sent[key]
	if !present {
		return props
	}
	out := azureARMMerge(props, nil)
	sentItems, sentList := value.([]any)
	items, isList := props[key].([]any)
	switch {
	case !nested:
		out[key] = value
	case each && sentList && isList:
		copied := make([]any, len(items))
		for i, item := range items {
			itemMap, ok := item.(map[string]any)
			sentMap, sentOK := azureARMIndex(sentItems, i).(map[string]any)
			if ok && sentOK {
				item = azureARMWithSent(itemMap, sentMap, rest)
			}
			copied[i] = item
		}
		out[key] = copied
	case !each:
		if sentMap, ok := value.(map[string]any); ok {
			own, _ := props[key].(map[string]any)
			out[key] = azureARMWithSent(own, sentMap, rest)
		}
	}
	return out
}

func azureARMIndex(items []any, i int) any {
	if i < len(items) {
		return items[i]
	}
	return nil
}

// azureARMWithState returns props with each state property taken from the
// existing record, or set to its initial value for a new resource.
func azureARMWithState(typ azureARMResourceType, props map[string]any, existing *azureARMResource) map[string]any {
	if len(typ.StateProperties) == 0 {
		return props
	}
	out := azureARMMerge(props, nil)
	for key, initial := range typ.StateProperties {
		delete(out, key)
		if initial != nil {
			out[key] = initial
		}
		if existing != nil {
			if stored, ok := existing.Properties[key]; ok {
				out[key] = stored
			}
		}
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

// azureARMLocationAction handles POST /subscriptions/{sub}/providers/
// {namespace}/locations/{location}/{action} for actions that need the
// subscription and location, such as per-region usage. Registered as
// "{action}", which may span segments, such as
// "sapVirtualInstanceMetadata/default/getSizingRecommendations"; provider
// actions registered as "locations/{action}" take precedence.
type azureARMLocationAction func(s *Server, sub, location string, payload map[string]any) (int, any)

// ponytail: registered in init() only, like azureARMResourceTypes.
var azureARMLocationActions = map[string]azureARMLocationAction{}

func registerAzureARMLocationAction(namespace, action string, fn azureARMLocationAction) {
	key := strings.ToLower(namespace + "/locations/" + action)
	if _, exists := azureARMLocationActions[key]; exists {
		panic("azure ARM location action registered twice: " + namespace + "/" + action)
	}
	azureARMLocationActions[key] = fn
}

// azureARMGroupAction handles POST /subscriptions/{sub}/resourceGroups/{rg}/
// providers/{namespace}/locations/{location}/{action}, such as a name check
// scoped to one resource group. The resource group exists.
type azureARMGroupAction func(s *Server, sub, rg string, payload map[string]any) (int, any)

// ponytail: registered in init() only, like azureARMResourceTypes.
var azureARMGroupActions = map[string]azureARMGroupAction{}

func registerAzureARMGroupAction(namespace, action string, fn azureARMGroupAction) {
	key := strings.ToLower(namespace + "/locations/" + action)
	if _, exists := azureARMGroupActions[key]; exists {
		panic("azure ARM group action registered twice: " + namespace + "/" + action)
	}
	azureARMGroupActions[key] = fn
}

// azureARMScopeAction handles POST {scope}/providers/{namespace}/{path},
// where scope is a subscription, resource group or resource ID and path
// may have several segments, such as "policyStates/latest/queryResults".
// A resource group in the scope exists.
type azureARMScopeAction func(s *Server, scope string, payload map[string]any) (int, any)

// ponytail: registered in init() only, like azureARMResourceTypes.
var azureARMScopeActions = map[string]azureARMScopeAction{}

func registerAzureARMScopeAction(namespace, path string, fn azureARMScopeAction) {
	key := strings.ToLower(namespace + "/" + path)
	if _, exists := azureARMScopeActions[key]; exists {
		panic("azure ARM scope action registered twice: " + namespace + "/" + path)
	}
	azureARMScopeActions[key] = fn
}

// handleAzureARMScopeAction serves registered scope actions; the segments
// after the last "providers" name the action and everything before it is
// the scope.
func (s *Server) handleAzureARMScopeAction(w http.ResponseWriter, r *http.Request, seg []string) bool {
	i := len(seg) - 1
	for i >= 0 && !strings.EqualFold(seg[i], "providers") {
		i--
	}
	if i < 2 || !strings.EqualFold(seg[0], "subscriptions") || len(seg)-i < 3 {
		return false
	}
	action := azureARMScopeActions[strings.ToLower(strings.Join(seg[i+1:], "/"))]
	if action == nil {
		return false
	}
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return true
	}
	scope := "/" + strings.Join(seg[:i], "/")
	if i >= 4 && strings.EqualFold(seg[2], "resourceGroups") {
		s.providerStorageMu.Lock()
		group := s.azureResourceGroups[azureResourceGroupKey(seg[1], seg[3])]
		s.providerStorageMu.Unlock()
		if group == nil {
			azureARMNotFound(w, "ResourceGroupNotFound", scope, "")
			return true
		}
	}
	status, body := action(s, scope, payload)
	respondAzureARMAction(w, status, body)
	return true
}

// azureARMView is a read-only collection computed from emulator state, such
// as health statuses or generated recommendations. GET is served at
// {scope}/providers/{namespace}/{collection} and, for one item found by
// name, at .../{collection}/{name}. scope is a subscription, resource group
// or resource ID, or empty at tenant scope. A collection registered as
// "{type}/{collection}" is nested under a stored resource of that type
// instead, at {resourceId}/{collection}, and scope is the resource ID.
// List returns items with an "id" key; GET by name finds the item whose
// id is the collection path plus "/{name}".
type azureARMView func(s *Server, scope string) []map[string]any

// azureARMQueryView is an azureARMView that also reads the request's
// query parameters, such as optional list filters.
type azureARMQueryView func(s *Server, scope string, query url.Values) []map[string]any

// ponytail: registered in init() only, like azureARMResourceTypes.
var azureARMViews = map[string]azureARMQueryView{}

func registerAzureARMView(namespace, collection string, list azureARMView) {
	registerAzureARMQueryView(namespace, collection, func(s *Server, scope string, _ url.Values) []map[string]any {
		return list(s, scope)
	})
}

func registerAzureARMQueryView(namespace, collection string, list azureARMQueryView) {
	key := strings.ToLower(namespace + "/" + collection)
	if _, exists := azureARMViews[key]; exists {
		panic("azure ARM view registered twice: " + namespace + "/" + collection)
	}
	azureARMViews[key] = list
}

// handleAzureARMView serves registered views; the segments after the last
// "providers" name the view and everything before it is the scope.
func (s *Server) handleAzureARMView(w http.ResponseWriter, r *http.Request, seg []string) bool {
	i := len(seg) - 1
	for i >= 0 && !strings.EqualFold(seg[i], "providers") {
		i--
	}
	var key, scope, collection, root string
	n := len(seg) - i
	nested := n >= 5
	// c is the index of the collection segment in a nested path, which may
	// sit under another view's item, as in {resourceId}/servers/{name}/
	// configurations.
	c := len(seg) - 1 - (n+1)%2
	switch {
	case i < 0:
		return false
	case n == 3 || n == 4:
		key = seg[i+1] + "/" + seg[i+2]
		if i > 0 {
			scope = "/" + strings.Join(seg[:i], "/")
		}
		collection = scope + "/providers/" + key
	case nested:
		key = seg[i+1] + "/" + seg[i+2]
		for j := i + 4; j <= c; j += 2 {
			key += "/" + seg[j]
		}
		scope = "/" + strings.Join(seg[:c], "/")
		collection = scope + "/" + seg[c]
		root = "/" + strings.Join(seg[:i+4], "/")
	default:
		return false
	}
	list := azureARMViews[strings.ToLower(key)]
	if list == nil {
		return false
	}
	if r.Method != http.MethodGet {
		// Writes go to an extension type sharing the collection, if any.
		if typ, ok := lookupAzureARMResourceType(seg[i+1], seg[i+2]); ok && typ.Extension && !nested {
			return false
		}
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	s.providerStorageMu.Lock()
	groupMissing := i >= 4 && strings.EqualFold(seg[2], "resourceGroups") && s.azureResourceGroups[azureResourceGroupKey(seg[1], seg[3])] == nil
	// Only stored parents can be missing; others, such as Azure support
	// services, are views themselves.
	_, stored := lookupAzureARMResourceType(seg[i+1], seg[i+2])
	parentMissing := nested && stored && s.azureARMResources[azureARMKey(root)] == nil
	s.providerStorageMu.Unlock()
	switch {
	case groupMissing:
		azureARMNotFound(w, "ResourceGroupNotFound", scope, "")
		return true
	case parentMissing:
		azureARMNotFound(w, "ParentResourceNotFound", collection, root)
		return true
	case nested && c > i+4 && !s.azureARMViewHas(key[:strings.LastIndex(key, "/")], seg[:c], r.URL.Query()):
		azureARMNotFound(w, "ParentResourceNotFound", collection, scope)
		return true
	}
	items := list(s, scope, r.URL.Query())
	if items == nil {
		items = []map[string]any{}
	}
	if n%2 == 1 {
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
		return true
	}
	// Only an item directly under this scope: a subscription list may hold
	// items of the same name for many resources.
	id := collection + "/" + seg[len(seg)-1]
	for _, item := range items {
		if strings.EqualFold(azureARMString(item["id"]), id) {
			respondJSON(w, http.StatusOK, item)
			return true
		}
	}
	azureARMNotFound(w, "ResourceNotFound", id, "")
	return true
}

// azureARMViewHas reports whether the item at path, the parent of a view
// nested under another collection, exists: an item the view registered as
// key lists, or a stored resource when no view is registered there.
func (s *Server) azureARMViewHas(key string, path []string, query url.Values) bool {
	id := "/" + strings.Join(path, "/")
	list := azureARMViews[strings.ToLower(key)]
	if list == nil {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		return s.azureARMResources[azureARMKey(id)] != nil
	}
	return slices.ContainsFunc(list(s, "/"+strings.Join(path[:len(path)-2], "/"), query), func(item map[string]any) bool {
		return strings.EqualFold(azureARMString(item["id"]), id)
	})
}

// azureARMNameChecks maps a lower-cased "{namespace}/{collection}" to the
// resource type whose names HEAD /providers/{namespace}/{collection}/{name}
// checks: 200 when a resource of that type has the name in any
// subscription, 404 otherwise.
// ponytail: registered in init() only, like azureARMResourceTypes.
var azureARMNameChecks = map[string]string{}

func registerAzureARMNameCheck(namespace, collection, resourceType string) {
	azureARMNameChecks[strings.ToLower(namespace+"/"+collection)] = namespace + "/" + resourceType
}

func (s *Server) serveAzureARMNameCheck(w http.ResponseWriter, r *http.Request, fullType, name string) {
	if r.Method != http.MethodHead {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, fullType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.WriteHeader(http.StatusNotFound)
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
	if s.handleAzureARMView(w, r, seg) || s.handleAzureARMScopeAction(w, r, seg) || s.handleAzureARMExtension(w, r, seg) {
		return true
	}
	var sub, rg, namespace string
	var rest []string
	switch {
	case len(seg) >= 3 && strings.EqualFold(seg[0], "providers") && azureARMIsTenantType(seg[1], seg[2]):
		namespace, rest = seg[1], seg[2:]
	case len(seg) == 3 && strings.EqualFold(seg[0], "providers"):
		return s.invokeAzureARMProviderAction(w, r, seg[1], seg[2])
	case len(seg) == 4 && strings.EqualFold(seg[0], "providers") && azureARMNameChecks[strings.ToLower(seg[1]+"/"+seg[2])] != "":
		s.serveAzureARMNameCheck(w, r, azureARMNameChecks[strings.ToLower(seg[1]+"/"+seg[2])], seg[3])
		return true
	case len(seg) == 5 && strings.EqualFold(seg[2], "providers") && azureARMProviderActions[strings.ToLower(seg[3]+"/"+seg[4])] != nil:
		return s.invokeAzureARMProviderAction(w, r, seg[3], seg[4])
	case len(seg) >= 7 && strings.EqualFold(seg[0], "subscriptions") && strings.EqualFold(seg[2], "providers") && strings.EqualFold(seg[4], "locations") && azureARMProviderActions[strings.ToLower(seg[3]+"/locations/"+seg[6])] == nil && azureARMLocationActions[strings.ToLower(seg[3]+"/locations/"+strings.Join(seg[6:], "/"))] != nil:
		sub, location := seg[1], seg[5]
		action := azureARMLocationActions[strings.ToLower(seg[3]+"/locations/"+strings.Join(seg[6:], "/"))]
		s.serveAzureARMPost(w, r, func(payload map[string]any) (int, any) { return action(s, sub, location, payload) })
		return true
	case len(seg) == 7 && strings.EqualFold(seg[2], "providers") && strings.EqualFold(seg[4], "locations"):
		return s.invokeAzureARMProviderAction(w, r, seg[3], "locations/"+seg[6])
	case len(seg) < 2 || !strings.EqualFold(seg[0], "subscriptions"):
		return false
	case len(seg) == 9 && strings.EqualFold(seg[2], "resourceGroups") && strings.EqualFold(seg[4], "providers") && strings.EqualFold(seg[6], "locations") && azureARMGroupActions[strings.ToLower(seg[5]+"/locations/"+seg[8])] != nil:
		s.invokeAzureARMGroupAction(w, r, azureARMGroupActions[strings.ToLower(seg[5]+"/locations/"+seg[8])], seg[1], seg[3])
		return true
	case len(seg) == 5 && strings.EqualFold(seg[2], "providers"),
		len(seg) > 5 && strings.EqualFold(seg[2], "providers") && azureARMIsSubscriptionType(seg[3], seg[4]):
		sub, namespace, rest = seg[1], seg[3], seg[4:]
	case len(seg) >= 7 && strings.EqualFold(seg[2], "resourceGroups") && strings.EqualFold(seg[4], "providers"):
		sub, rg, namespace, rest = seg[1], seg[3], seg[5], seg[6:]
	default:
		return false
	}

	// POST {resourceId}/{action}: the trailing segment is an action, not a type.
	var action azureARMAction
	var transition map[string]any
	var stringBody bool
	if r.Method == http.MethodPost && len(rest) >= 3 && len(rest)%2 == 1 {
		typ, ok := lookupAzureARMResourceType(namespace, azureARMTypePath(rest[:len(rest)-1]))
		verb := strings.ToLower(rest[len(rest)-1])
		if !ok {
			return false
		}
		action, transition = s.azureARMActionFor(typ, verb), typ.Transitions[verb]
		stringBody = typ.StringBodyActions[verb]
		if action == nil && transition == nil {
			return false
		}
		rest = rest[:len(rest)-1]
	}
	typ, ok := lookupAzureARMResourceType(namespace, azureARMTypePath(rest))
	// Tenant types live only at tenant scope, and others only under a subscription.
	// Tenant types live only at tenant scope (unless also Subscription),
	// Subscription types never in a resource group, and others only under
	// a subscription.
	if !ok || (sub == "" && !typ.Tenant) || (sub != "" && typ.Tenant && !typ.Subscription) || (rg != "" && typ.Subscription) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}

	// Rebuild the ID with the registered type casing; names keep client casing.
	scope := ""
	if sub != "" {
		scope = "/subscriptions/" + sub
	}
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
		s.invokeAzureARMAction(w, r, action, stringBody, id, sub, rg, parentID)
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
		if (typ.Linked != nil && typ.PatchBody == nil) || typ.NoPatch {
			// Linked resources follow PUT only, unless the type's PatchBody
			// limits what PATCH changes; these APIs have no PATCH.
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return
		}
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
		if typ.NoDelete {
			azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
			return
		}
		s.providerStorageMu.Lock()
		if existing := s.azureARMResources[azureARMKey(id)]; existing != nil && typ.ValidateDelete != nil {
			if msg := typ.ValidateDelete(existing); msg != "" {
				s.providerStorageMu.Unlock()
				azureARMError(w, http.StatusConflict, "Conflict", msg)
				return
			}
		}
		// Child and extension resources go with their parent.
		existed := s.deleteAzureARMTreeLocked(id)
		if parentID != "" {
			s.refreshAzureARMParentLocked(parentID)
		}
		s.providerStorageMu.Unlock()
		switch {
		case existed && typ.DeleteStatus != 0:
			w.WriteHeader(typ.DeleteStatus)
		case existed:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

// handleAzureARMExtension serves extension resources and their child
// types: the segments after the last "providers" name a registered
// extension type, and everything before it is the scope. The scope resource
// itself is not required to exist, unless the type sets ScopeStored,
// because it may belong to a service with its own store.
func (s *Server) handleAzureARMExtension(w http.ResponseWriter, r *http.Request, seg []string) bool {
	i := len(seg) - 1
	for i >= 0 && !strings.EqualFold(seg[i], "providers") {
		i--
	}
	if i < 2 || !strings.EqualFold(seg[0], "subscriptions") || len(seg)-i < 3 {
		return false
	}
	rest := seg[i+2:]
	// POST {extension}/{action}
	verb := ""
	if r.Method == http.MethodPost && len(rest) >= 3 && len(rest)%2 == 1 {
		verb, rest = strings.ToLower(rest[len(rest)-1]), rest[:len(rest)-1]
	}
	typ, ok := lookupAzureARMResourceType(seg[i+1], azureARMTypePath(rest))
	if !ok || !typ.Extension {
		return false
	}
	var action azureARMAction
	var transition map[string]any
	if verb != "" {
		action, transition = s.azureARMActionFor(typ, verb), typ.Transitions[verb]
		if action == nil && transition == nil {
			return false
		}
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	scope := "/" + strings.Join(seg[:i], "/")
	if typ.ValidateScope != nil {
		if msg := typ.ValidateScope(scope); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidScope", msg)
			return true
		}
	}
	if typ.ScopeStored {
		if stored, _ := s.getAzureARMResource(scope, "", "", ""); stored == nil {
			azureARMNotFound(w, "ParentResourceNotFound", scope+"/providers/"+typ.fullType(), scope)
			return true
		}
	}
	sub, rg := seg[1], ""
	if i >= 4 && strings.EqualFold(seg[2], "resourceGroups") {
		rg = seg[3]
	}
	// Rebuild the ID with the registered type casing; names keep client casing.
	typeParts := strings.Split(typ.Type, "/")
	id, parentID := scope+"/providers/"+typ.Namespace, ""
	for j := 1; j < len(rest); j += 2 {
		parentID = id
		id += "/" + typeParts[j/2] + "/" + rest[j]
	}
	if len(rest)%2 == 1 {
		// A collection: id names its parent, if any.
		parentID, id = id, id+"/"+typeParts[len(typeParts)-1]
	}
	if len(rest) <= 2 {
		parentID = ""
	}
	switch {
	case action != nil:
		s.invokeAzureARMAction(w, r, action, typ.StringBodyActions[verb], id, sub, rg, parentID)
		return true
	case transition != nil:
		s.transitionAzureARMResource(w, transition, id, sub, rg, parentID)
		return true
	case len(rest)%2 == 0:
		s.serveAzureARMItem(w, r, typ, id, sub, rg, parentID, rest[len(rest)-1])
		return true
	case r.Method != http.MethodGet:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	case parentID != "":
		s.listAzureARMResources(w, typ, sub, rg, parentID, scope)
		return true
	}
	s.providerStorageMu.Lock()
	if rg != "" && s.azureResourceGroups[azureResourceGroupKey(sub, rg)] == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ResourceGroupNotFound", id, "")
		return true
	}
	// Only settings attached directly to this scope, not to resources under it.
	prefix := azureARMKey(id + "/")
	records := s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		key := azureARMKey(rec.ID)
		return strings.EqualFold(rec.Type, typ.fullType()) && strings.HasPrefix(key, prefix) && !strings.Contains(key[len(prefix):], "/")
	})
	s.providerStorageMu.Unlock()
	respondAzureARMList(w, records)
	return true
}

// azureARMIsSubscriptionType reports whether typ, a top-level type, is a
// registered subscription type.
func azureARMIsSubscriptionType(namespace, typ string) bool {
	t, ok := lookupAzureARMResourceType(namespace, typ)
	return ok && t.Subscription
}

// azureARMActionFor returns the action, or server action, named verb.
func (s *Server) azureARMActionFor(typ azureARMResourceType, verb string) azureARMAction {
	if action := typ.Actions[verb]; action != nil {
		return action
	}
	if action := typ.ServerActions[verb]; action != nil {
		return func(rec *azureARMResource, payload map[string]any) (int, any) { return action(s, rec, payload) }
	}
	return nil
}

// azureARMIsTenantType reports whether typ, a top-level type, is a
// registered tenant type.
func azureARMIsTenantType(namespace, typ string) bool {
	t, ok := lookupAzureARMResourceType(namespace, typ)
	return ok && t.Tenant
}

// azureARMTypePath returns the type segments (even indexes) joined by "/".
func azureARMTypePath(rest []string) string {
	types := make([]string, 0, (len(rest)+1)/2)
	for i := 0; i < len(rest); i += 2 {
		types = append(types, rest[i])
	}
	return strings.Join(types, "/")
}

func (s *Server) invokeAzureARMAction(w http.ResponseWriter, r *http.Request, action azureARMAction, stringBody bool, id, sub, rg, parentID string) {
	payload, ok := azureARMReadBodyAs(w, r, stringBody)
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
	if parentID != "" {
		s.refreshAzureARMParentLocked(parentID)
	}
	s.providerStorageMu.Unlock()
	respondAzureARMAction(w, http.StatusOK, nil)
}

func (s *Server) invokeAzureARMProviderAction(w http.ResponseWriter, r *http.Request, namespace, name string) bool {
	action := azureARMProviderActions[strings.ToLower(namespace+"/"+name)]
	if action == nil {
		return false
	}
	s.serveAzureARMPost(w, r, func(payload map[string]any) (int, any) { return action(s, payload) })
	return true
}

// serveAzureARMPost checks the verb and api-version of an action request,
// reads its body and writes what run returns.
func (s *Server) serveAzureARMPost(w http.ResponseWriter, r *http.Request, run func(payload map[string]any) (int, any)) {
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	status, body := run(payload)
	respondAzureARMAction(w, status, body)
}

func (s *Server) invokeAzureARMGroupAction(w http.ResponseWriter, r *http.Request, action azureARMGroupAction, sub, rg string) {
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return
	}
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	group := s.azureResourceGroups[azureResourceGroupKey(sub, rg)]
	s.providerStorageMu.Unlock()
	if group == nil {
		azureARMNotFound(w, "ResourceGroupNotFound", "/subscriptions/"+sub+"/resourceGroups/"+rg, "")
		return
	}
	status, body := action(s, sub, rg, payload)
	respondAzureARMAction(w, status, body)
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
		APIVersion: parent.APIVersion,
	}
	child.Properties = azureARMWithState(typ, azureARMProperties(typ, child), nil)
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
	switch {
	case typ.Proxy && !typ.KeepLocation:
		location = ""
	case !typ.Proxy && location == "":
		azureARMError(w, http.StatusBadRequest, "LocationRequired", "The location property is required for this definition.")
		return
	}
	if typ.Validate != nil {
		if msg := typ.Validate(payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return
		}
	}
	if msg := validateAzureARMInline(typ, payload); msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
		return
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
		APIVersion: r.URL.Query().Get("api-version"),
	}
	if typ.ReadOnlySKU != nil {
		record.SKU = azureARMMerge(record.SKU, typ.ReadOnlySKU(record.SKU))
	}

	// Hooks run under the lock, so unlock with defer: a panicking hook must
	// not leave the store locked.
	existing, ok := func() (*azureARMResource, bool) {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		group := s.azureResourceGroups[azureResourceGroupKey(sub, rg)]
		if group == nil && rg != "" {
			azureARMNotFound(w, "ResourceGroupNotFound", id, "")
			return nil, false
		}
		var parent *azureARMResource
		if parentID != "" {
			if parent = s.azureARMResources[azureARMKey(parentID)]; parent == nil {
				azureARMNotFound(w, "ParentResourceNotFound", id, parentID)
				return nil, false
			}
		}
		if parent != nil && typ.ValidateInParent != nil {
			if msg := typ.ValidateInParent(parent.Properties, name, payload); msg != "" {
				azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
				return nil, false
			}
		}
		if msg := s.validateAzureARMKeptInlineLocked(typ, id, payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return nil, false
		}
		existing := s.azureARMResources[azureARMKey(id)]
		switch {
		case existing != nil && !strings.EqualFold(existing.Location, location):
			azureARMError(w, http.StatusConflict, "InvalidResourceLocation",
				fmt.Sprintf("The resource '%s' already exists in location '%s'. A resource with the same name cannot be created in location '%s'.", existing.ID, existing.Location, location))
			return nil, false
		case existing != nil:
			// Keep the casing the resource was created with, like ARM does.
			record.ID, record.SubscriptionID, record.ResourceGroup, record.Name = existing.ID, existing.SubscriptionID, existing.ResourceGroup, existing.Name
			record.CreatedAt = existing.CreatedAt
		case typ.Extension, (typ.Tenant || typ.Subscription) && parent == nil:
			// The ID keeps the scope as the client sent it.
		case parent != nil:
			record.ResourceGroup = parent.ResourceGroup
			record.ID = parent.ID + "/" + typ.Type[strings.LastIndex(typ.Type, "/")+1:] + "/" + name
		default:
			record.ResourceGroup = group.Name
			record.ID = fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s", group.SubscriptionID, group.Name, typ.fullType(), name)
		}
		// Server-owned properties see the final ID and creation time.
		record.Properties = azureARMWithState(typ, azureARMProperties(typ, record), existing)
		if !s.reconcileAzureARMLocked(w, typ, existing, record) {
			return nil, false
		}
		record.Properties = s.syncAzureARMInlineLocked(typ, record, azureARMObject(payload["properties"]))
		if typ.Linked != nil && !s.storeAzureARMLinkedLocked(w, typ, record, existing, azureARMObject(payload["properties"])) {
			return nil, false
		}
		s.azureARMResources[azureARMKey(id)] = record
		if parent != nil {
			s.refreshAzureARMParentLocked(parent.ID)
		}
		if existing == nil && typ.DefaultChildren != nil {
			for _, child := range typ.DefaultChildren(record) {
				s.azureARMResources[azureARMKey(child.ID)] = child
			}
			s.refreshAzureARMParentLocked(record.ID)
			record = s.azureARMResources[azureARMKey(id)]
		}
		return existing, true
	}()
	if !ok {
		return
	}

	status := http.StatusCreated
	if typ.CreateStatus != 0 {
		status = typ.CreateStatus
	}
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
	var state map[string]any
	if typ.PatchBody != nil {
		var msg string
		if payload, state, msg = typ.PatchBody(payload); msg != "" {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
			return
		}
	}
	updated, code, conflict, invalid := s.patchAzureARMRecord(typ, id, sub, rg, parentID, payload, state)
	if conflict != "" {
		azureARMError(w, http.StatusConflict, "Conflict", conflict)
		return
	}
	if invalid != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", invalid)
		return
	}
	if updated == nil {
		azureARMNotFound(w, code, id, parentID)
		return
	}
	respondJSON(w, http.StatusOK, updated.fixture())
}

// patchAzureARMRecord merges payload, and any state values from the type's
// PatchBody, into the stored record. It runs the type's hooks under the
// lock, so it unlocks with defer: a panicking hook must not leave the
// store locked. It returns the updated record, or the ARM error code for a
// missing one, or the type's ValidatePatch or Reconcile message.
func (s *Server) patchAzureARMRecord(typ azureARMResourceType, id, sub, rg, parentID string, payload, state map[string]any) (updated *azureARMResource, code, conflict, invalid string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	existing, code := s.lookupAzureARMResourceLocked(id, sub, rg, parentID)
	if existing == nil {
		return nil, code, "", ""
	}
	if typ.ValidatePatch != nil {
		if msg := typ.ValidatePatch(s, existing, payload); msg != "" {
			return nil, "", msg, ""
		}
	}
	copied := *existing
	updated = &copied
	if tags, present := payload["tags"]; present {
		updated.Tags = azureARMObject(tags)
	}
	if identity := azureARMObject(payload["identity"]); len(identity) != 0 {
		updated.Identity = identity
	}
	if sku := azureARMObject(payload["sku"]); len(sku) != 0 {
		updated.SKU = sku
	}
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(existing.Properties, azureARMObject(payload["properties"]))
	updated.Properties = azureARMMerge(azureARMWithState(typ, azureARMProperties(typ, updated), existing), state)
	if typ.Reconcile != nil {
		props, msg := typ.Reconcile(s, existing, updated)
		if msg != "" {
			return nil, "", "", msg
		}
		updated.Properties = azureARMMerge(updated.Properties, props)
	}
	// PATCH never changes inline children; it lists them as they are.
	updated.Properties = s.azureARMInlinePropertiesLocked(typ, updated)
	s.azureARMResources[azureARMKey(id)] = updated
	if parentID != "" {
		s.refreshAzureARMParentLocked(parentID)
	}
	return updated, "", "", ""
}

// reconcileAzureARMLocked merges the type's Reconcile properties into a PUT
// record, or writes the 400 that rejects it. Caller holds providerStorageMu.
func (s *Server) reconcileAzureARMLocked(w http.ResponseWriter, typ azureARMResourceType, existing, record *azureARMResource) bool {
	if typ.Reconcile == nil {
		return true
	}
	props, msg := typ.Reconcile(s, existing, record)
	if msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", msg)
		return false
	}
	record.Properties = azureARMMerge(record.Properties, props)
	return true
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
	if scope == "" {
		// Tenant scope lists only tenant resources, not those of a type
		// that is also served under subscriptions.
		scope = "/providers"
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
	return azureARMReadBodyAs(w, r, false)
}

// azureARMReadBodyAs is azureARMReadBody that, with allowString, also takes
// a bare JSON string as payload["body"], the name REST references give it.
func azureARMReadBodyAs(w http.ResponseWriter, r *http.Request, allowString bool) (map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "unable to read request body")
		return nil, false
	}
	payload := map[string]any{}
	if strings.TrimSpace(string(body)) == "" {
		return payload, true
	}
	var text string
	if allowString && strings.HasPrefix(strings.TrimSpace(string(body)), `"`) && json.Unmarshal(body, &text) == nil {
		return map[string]any{"body": text}, true
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
