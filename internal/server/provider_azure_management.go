package server

import (
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Management management groups: a tenant hierarchy under the
// tenant root group, with subscriptions placed in it. Subscriptions are
// implicit in Stackyard, so the hierarchy holds only the ones a client has
// placed or moved back to the root.

const (
	azureManagementGroupType  = "Microsoft.Management/managementGroups"
	azureManagementGroupsPath = "/providers/Microsoft.Management/managementGroups/"
)

type azureManagementGroup struct {
	Name        string
	DisplayName string
	Parent      string // lower-cased key of the parent; empty for the root
	Version     int
	UpdatedTime time.Time
}

var azureManagementGroupNamePattern = regexp.MustCompile(`^[a-zA-Z0-9\-_().]{0,89}[a-zA-Z0-9\-_()]$`)

// azureManagementGroupsLocked returns the groups, creating the tenant root
// group on first use. Caller holds providerStorageMu.
func (s *Server) azureManagementGroupsLocked() map[string]*azureManagementGroup {
	if s.azureManagementGroups == nil {
		s.azureManagementGroups = map[string]*azureManagementGroup{
			azureManagementGroupKey(azureDefaultTenantID): {Name: azureDefaultTenantID, DisplayName: "Tenant Root Group", Version: 1, UpdatedTime: time.Now().UTC()},
		}
		s.azureManagementGroupSubscriptions = map[string]string{}
	}
	return s.azureManagementGroups
}

func azureManagementGroupKey(name string) string { return strings.ToLower(name) }

func azureManagementGroupID(name string) string { return azureManagementGroupsPath + name }

func (s *Server) handleAzureManagementRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(strings.ToLower(path), "/azure/providers/microsoft.management/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	switch {
	case len(seg) == 3 && strings.EqualFold(seg[2], "checkNameAvailability") && r.Method == http.MethodPost:
		s.azureManagementGroupNameCheck(w, r)
	case len(seg) == 3 && strings.EqualFold(seg[2], "managementGroups") && r.Method == http.MethodGet:
		s.listAzureManagementGroups(w)
	case len(seg) == 4 && strings.EqualFold(seg[2], "managementGroups"):
		s.serveAzureManagementGroup(w, r, seg[3])
	case len(seg) == 5 && strings.EqualFold(seg[2], "managementGroups") && strings.EqualFold(seg[4], "descendants") && r.Method == http.MethodGet:
		s.listAzureManagementGroupDescendants(w, seg[3])
	case len(seg) == 5 && strings.EqualFold(seg[2], "managementGroups") && strings.EqualFold(seg[4], "subscriptions") && r.Method == http.MethodGet:
		s.listAzureManagementGroupSubscriptions(w, seg[3])
	case len(seg) == 6 && strings.EqualFold(seg[2], "managementGroups") && strings.EqualFold(seg[4], "subscriptions"):
		s.serveAzureManagementGroupSubscription(w, r, seg[3], seg[5])
	case len(seg) >= 3 && len(seg) <= 6 && (strings.EqualFold(seg[2], "managementGroups") || strings.EqualFold(seg[2], "checkNameAvailability")):
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	default:
		return false
	}
	return true
}

func (s *Server) serveAzureManagementGroup(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		payload, ok := azureARMReadBody(w, r)
		if !ok {
			return
		}
		displayName, parent := azureManagementGroupRequest(r.Method, payload)
		s.putAzureManagementGroup(w, r.Method == http.MethodPatch, name, displayName, parent)
	case http.MethodGet:
		expand := strings.EqualFold(r.URL.Query().Get("$expand"), "children")
		recurse := strings.EqualFold(r.URL.Query().Get("$recurse"), "true")
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		group := s.azureManagementGroupsLocked()[azureManagementGroupKey(name)]
		if group == nil {
			azureManagementGroupNotFound(w, name)
			return
		}
		respondJSON(w, http.StatusOK, s.azureManagementGroupFixtureLocked(group, expand, recurse))
	case http.MethodDelete:
		s.deleteAzureManagementGroup(w, name)
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

// azureManagementGroupRequest reads the display name and parent: PUT nests
// them under properties, while PATCH sends displayName and parentGroupId
// at the top level. Empty values leave the current ones in place.
func azureManagementGroupRequest(method string, payload map[string]any) (displayName, parent string) {
	if method == http.MethodPatch {
		return azureARMString(payload["displayName"]), azureARMString(payload["parentGroupId"])
	}
	props := azureARMObject(payload["properties"])
	parentRef := azureARMObject(azureARMObject(props["details"])["parent"])
	return azureARMString(props["displayName"]), azureARMString(parentRef["id"])
}

func (s *Server) putAzureManagementGroup(w http.ResponseWriter, patch bool, name, displayName, parentRef string) {
	if !azureManagementGroupNamePattern.MatchString(name) {
		azureARMError(w, http.StatusBadRequest, "InvalidResourceName", "Management group IDs must be 1-90 letters, digits, hyphens, underscores, periods or parentheses, and cannot end with a period.")
		return
	}
	if len(displayName) > 90 {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "properties.displayName must be at most 90 characters.")
		return
	}
	parentName := parentRef[strings.LastIndex(parentRef, "/")+1:]
	if parentRef != "" && strings.Contains(parentRef, "/") && !strings.HasPrefix(strings.ToLower(parentRef), strings.ToLower(azureManagementGroupsPath)) {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "The parent must be a management group ID.")
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	groups := s.azureManagementGroupsLocked()
	key := azureManagementGroupKey(name)
	existing := groups[key]
	if patch && existing == nil {
		azureManagementGroupNotFound(w, name)
		return
	}
	isRoot := key == azureManagementGroupKey(azureDefaultTenantID)
	if isRoot && parentName != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "The tenant root group cannot be moved.")
		return
	}
	updated := azureManagementGroup{Name: name, DisplayName: name, Version: 1, UpdatedTime: time.Now().UTC()}
	if !isRoot {
		updated.Parent = azureManagementGroupKey(azureDefaultTenantID)
	}
	if existing != nil {
		updated = *existing
		updated.Version++
		updated.UpdatedTime = time.Now().UTC()
	}
	if displayName != "" {
		updated.DisplayName = displayName
	}
	// Clients name the root by their own tenant ID, which may not be
	// Stackyard's: any other GUID that isn't a group means the root.
	if _, err := uuid.Parse(parentName); err == nil && groups[azureManagementGroupKey(parentName)] == nil {
		parentName = azureDefaultTenantID
	}
	if parentName != "" {
		parentKey := azureManagementGroupKey(parentName)
		if groups[parentKey] == nil {
			azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "The parent management group '"+parentName+"' was not found.")
			return
		}
		// A group can't move under itself or one of its descendants.
		for at := parentKey; at != ""; at = groups[at].Parent {
			if at == key {
				azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "A management group cannot be moved under itself or one of its descendants.")
				return
			}
		}
		updated.Parent = parentKey
	}
	groups[key] = &updated
	respondJSON(w, http.StatusOK, s.azureManagementGroupFixtureLocked(&updated, false, false))
}

func (s *Server) deleteAzureManagementGroup(w http.ResponseWriter, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	groups := s.azureManagementGroupsLocked()
	key := azureManagementGroupKey(name)
	if key == azureManagementGroupKey(azureDefaultTenantID) {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "The tenant root group cannot be deleted.")
		return
	}
	if groups[key] == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if len(s.azureManagementGroupChildrenLocked(key)) > 0 {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "The management group '"+name+"' has children and cannot be deleted.")
		return
	}
	delete(groups, key)
	w.WriteHeader(http.StatusOK)
}

// azureManagementGroupChild is a group or subscription directly under a group.
type azureManagementGroupChild struct {
	group *azureManagementGroup
	sub   string
}

// azureManagementGroupChildrenLocked lists a group's children, groups first,
// each sorted by name. Caller holds providerStorageMu.
func (s *Server) azureManagementGroupChildrenLocked(key string) []azureManagementGroupChild {
	var groups []azureManagementGroupChild
	for _, group := range s.azureManagementGroups {
		if group.Parent == key {
			groups = append(groups, azureManagementGroupChild{group: group})
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].group.Name < groups[j].group.Name })
	var subs []azureManagementGroupChild
	for sub, parent := range s.azureManagementGroupSubscriptions {
		if parent == key {
			subs = append(subs, azureManagementGroupChild{sub: sub})
		}
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].sub < subs[j].sub })
	return append(groups, subs...)
}

func (s *Server) azureManagementGroupFixtureLocked(group *azureManagementGroup, expand, recurse bool) map[string]any {
	details := map[string]any{
		"version":     group.Version,
		"updatedTime": group.UpdatedTime.Format(time.RFC3339),
		"updatedBy":   azureDefaultTenantID,
	}
	if parent := s.azureManagementGroups[group.Parent]; parent != nil {
		details["parent"] = map[string]any{"id": azureManagementGroupID(parent.Name), "name": parent.Name, "displayName": parent.DisplayName}
	}
	props := map[string]any{"tenantId": azureDefaultTenantID, "displayName": group.DisplayName, "details": details}
	if expand {
		props["children"] = s.azureManagementGroupChildFixturesLocked(azureManagementGroupKey(group.Name), recurse)
	}
	return map[string]any{"id": azureManagementGroupID(group.Name), "type": azureManagementGroupType, "name": group.Name, "properties": props}
}

func (s *Server) azureManagementGroupChildFixturesLocked(key string, recurse bool) []any {
	children := make([]any, 0)
	for _, child := range s.azureManagementGroupChildrenLocked(key) {
		if child.group == nil {
			children = append(children, map[string]any{"type": "/subscriptions", "id": "/subscriptions/" + child.sub, "name": child.sub, "displayName": azureSubscriptionFixture(child.sub)["displayName"]})
			continue
		}
		item := map[string]any{"type": azureManagementGroupType, "id": azureManagementGroupID(child.group.Name), "name": child.group.Name, "displayName": child.group.DisplayName}
		if recurse {
			item["children"] = s.azureManagementGroupChildFixturesLocked(azureManagementGroupKey(child.group.Name), true)
		}
		children = append(children, item)
	}
	return children
}

func (s *Server) listAzureManagementGroups(w http.ResponseWriter) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	groups := s.azureManagementGroupsLocked()
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]any, 0, len(keys))
	for _, key := range keys {
		group := groups[key]
		items = append(items, map[string]any{"id": azureManagementGroupID(group.Name), "type": azureManagementGroupType, "name": group.Name,
			"properties": map[string]any{"tenantId": azureDefaultTenantID, "displayName": group.DisplayName}})
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) listAzureManagementGroupDescendants(w http.ResponseWriter, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureManagementGroupKey(name)
	if s.azureManagementGroupsLocked()[key] == nil {
		azureManagementGroupNotFound(w, name)
		return
	}
	items := make([]any, 0)
	var walk func(parentKey string)
	walk = func(parentKey string) {
		parentID := azureManagementGroupID(s.azureManagementGroups[parentKey].Name)
		for _, child := range s.azureManagementGroupChildrenLocked(parentKey) {
			if child.group == nil {
				items = append(items, map[string]any{"id": "/subscriptions/" + child.sub, "type": "/subscriptions", "name": child.sub,
					"properties": map[string]any{"displayName": azureSubscriptionFixture(child.sub)["displayName"], "parent": map[string]any{"id": parentID}}})
				continue
			}
			items = append(items, map[string]any{"id": azureManagementGroupID(child.group.Name), "type": azureManagementGroupType, "name": child.group.Name,
				"properties": map[string]any{"displayName": child.group.DisplayName, "parent": map[string]any{"id": parentID}}})
			walk(azureManagementGroupKey(child.group.Name))
		}
	}
	walk(key)
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) serveAzureManagementGroupSubscription(w http.ResponseWriter, r *http.Request, name, sub string) {
	if _, err := uuid.Parse(sub); err != nil {
		azureARMError(w, http.StatusBadRequest, "InvalidSubscriptionId", "The subscription ID '"+sub+"' is not a GUID.")
		return
	}
	sub = strings.ToLower(sub)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureManagementGroupKey(name)
	group := s.azureManagementGroupsLocked()[key]
	if group == nil {
		azureManagementGroupNotFound(w, name)
		return
	}
	switch r.Method {
	case http.MethodPut:
		// A subscription has one parent: placing it here moves it.
		s.azureManagementGroupSubscriptions[sub] = key
		respondJSON(w, http.StatusOK, azureManagementGroupSubscriptionFixture(group, sub))
	case http.MethodGet:
		if s.azureManagementGroupSubscriptions[sub] != key {
			azureARMError(w, http.StatusNotFound, "NotFound", "The subscription '"+sub+"' is not in management group '"+name+"'.")
			return
		}
		respondJSON(w, http.StatusOK, azureManagementGroupSubscriptionFixture(group, sub))
	case http.MethodDelete:
		// Removing a subscription from a group moves it back to the root.
		if s.azureManagementGroupSubscriptions[sub] == key {
			s.azureManagementGroupSubscriptions[sub] = azureManagementGroupKey(azureDefaultTenantID)
		}
		w.WriteHeader(http.StatusOK)
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}

func (s *Server) listAzureManagementGroupSubscriptions(w http.ResponseWriter, name string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	key := azureManagementGroupKey(name)
	group := s.azureManagementGroupsLocked()[key]
	if group == nil {
		azureManagementGroupNotFound(w, name)
		return
	}
	items := make([]any, 0)
	for _, child := range s.azureManagementGroupChildrenLocked(key) {
		if child.group == nil {
			items = append(items, azureManagementGroupSubscriptionFixture(group, child.sub))
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func azureManagementGroupSubscriptionFixture(group *azureManagementGroup, sub string) map[string]any {
	return map[string]any{
		"id":   azureManagementGroupID(group.Name) + "/subscriptions/" + sub,
		"type": "Microsoft.Management/managementGroups/subscriptions",
		"name": sub,
		"properties": map[string]any{
			"tenant":      azureDefaultTenantID,
			"displayName": azureSubscriptionFixture(sub)["displayName"],
			"parent":      map[string]any{"id": azureManagementGroupID(group.Name)},
			"state":       "Active",
		},
	}
}

func (s *Server) azureManagementGroupNameCheck(w http.ResponseWriter, r *http.Request) {
	payload, ok := azureARMReadBody(w, r)
	if !ok {
		return
	}
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		azureARMError(w, http.StatusBadRequest, "InvalidRequestContent", "name is required")
		return
	}
	if !azureManagementGroupNamePattern.MatchString(name) {
		respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": false, "reason": "Invalid", "message": "The management group ID '" + name + "' is invalid."})
		return
	}
	s.providerStorageMu.Lock()
	taken := s.azureManagementGroupsLocked()[azureManagementGroupKey(name)] != nil
	s.providerStorageMu.Unlock()
	if taken {
		respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The management group ID '" + name + "' is already in use."})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"nameAvailable": true})
}

func azureManagementGroupNotFound(w http.ResponseWriter, name string) {
	azureARMError(w, http.StatusNotFound, "NotFound", "The management group '"+name+"' was not found.")
}
