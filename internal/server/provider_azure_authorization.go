package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

type azureAuthorizationPermission struct {
	Actions    []string
	NotActions []string
}

type azureAuthorizationRoleAssignment struct {
	ID               string
	Name             string
	Scope            string
	PrincipalID      string
	RoleDefinitionID string
}

type azureAuthorizationRoleDefinition struct {
	ID               string
	Name             string
	Scope            string
	RoleName         string
	Description      string
	RoleType         string
	AssignableScopes []string
	Permissions      []azureAuthorizationPermission
}

func (s *Server) handleAzureAuthorizationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	providerIndex := azureAuthorizationProviderIndex(segments)
	if providerIndex < 0 {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	scope := azureAuthorizationScopeFromSegments(segments[:providerIndex])
	tail := segments[providerIndex+2:]
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}

	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "permissions":
		s.handleAzureAuthorizationPermissions(w, r, path, scope, tail[1:])
		return true
	case "roleassignments":
		s.handleAzureAuthorizationRoleAssignments(w, r, path, scope, tail[1:])
		return true
	case "roledefinitions":
		s.handleAzureAuthorizationRoleDefinitions(w, r, path, scope, tail[1:])
		return true
	default:
		respondAzureImplemented(w, path)
		return true
	}
}

func (s *Server) handleAzureAuthorizationPermissions(w http.ResponseWriter, r *http.Request, path, scope string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) != 0 {
		respondAzureImplemented(w, path)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"actions": []string{
					"*",
					"Microsoft.Authorization/*/read",
					"Microsoft.Authorization/roleAssignments/*",
					"Microsoft.Authorization/roleDefinitions/*",
				},
				"notActions": []string{},
			},
		},
	})
}

func (s *Server) handleAzureAuthorizationRoleAssignments(w http.ResponseWriter, r *http.Request, path, scope string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureAuthorizationRoleAssignmentList(w, r, scope)
	case len(tail) == 1:
		name := strings.TrimSpace(tail[0])
		if name == "" {
			respondAzureInvalidRequest(w, path, "role assignment name is required")
			return
		}
		switch r.Method {
		case http.MethodPut:
			s.handleAzureAuthorizationRoleAssignmentCreateOrUpdate(w, r, path, scope, name)
		case http.MethodGet:
			s.handleAzureAuthorizationRoleAssignmentGet(w, path, scope, name)
		case http.MethodDelete:
			s.handleAzureAuthorizationRoleAssignmentDelete(w, path, scope, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAuthorizationRoleDefinitions(w http.ResponseWriter, r *http.Request, path, scope string, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		s.handleAzureAuthorizationRoleDefinitionList(w, r, scope)
	case len(tail) == 1:
		name := strings.TrimSpace(tail[0])
		if name == "" {
			respondAzureInvalidRequest(w, path, "role definition id is required")
			return
		}
		switch r.Method {
		case http.MethodPut:
			s.handleAzureAuthorizationRoleDefinitionCreateOrUpdate(w, r, path, scope, name)
		case http.MethodGet:
			s.handleAzureAuthorizationRoleDefinitionGet(w, path, scope, name)
		case http.MethodDelete:
			s.handleAzureAuthorizationRoleDefinitionDelete(w, path, scope, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureAuthorizationRoleAssignmentCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, scope, name string) {
	_, payload, ok := azureAuthorizationReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureAuthorizationMap(payload["properties"])
	principalID := strings.TrimSpace(azureAuthorizationString(properties["principalId"]))
	roleDefinitionID := strings.TrimSpace(azureAuthorizationString(properties["roleDefinitionId"]))
	if principalID == "" {
		respondAzureInvalidRequest(w, path, "properties.principalId is required")
		return
	}
	if roleDefinitionID == "" {
		respondAzureInvalidRequest(w, path, "properties.roleDefinitionId is required")
		return
	}

	record := &azureAuthorizationRoleAssignment{
		ID:               azureAuthorizationRoleAssignmentID(scope, name),
		Name:             name,
		Scope:            scope,
		PrincipalID:      principalID,
		RoleDefinitionID: roleDefinitionID,
	}

	s.providerStorageMu.Lock()
	s.azureAuthorizationRoleAssignments[azureAuthorizationStorageKey(record.ID)] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureAuthorizationRoleAssignmentFixture(record))
}

func (s *Server) handleAzureAuthorizationRoleAssignmentGet(w http.ResponseWriter, path, scope, name string) {
	s.providerStorageMu.Lock()
	record := s.azureAuthorizationRoleAssignments[azureAuthorizationStorageKey(azureAuthorizationRoleAssignmentID(scope, name))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureAuthorizationNotFound(w, path, "RoleAssignmentNotFound", "role assignment not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAuthorizationRoleAssignmentFixture(record))
}

func (s *Server) handleAzureAuthorizationRoleAssignmentDelete(w http.ResponseWriter, path, scope, name string) {
	id := azureAuthorizationRoleAssignmentID(scope, name)
	key := azureAuthorizationStorageKey(id)

	s.providerStorageMu.Lock()
	record := s.azureAuthorizationRoleAssignments[key]
	if record == nil {
		record = &azureAuthorizationRoleAssignment{
			ID:    id,
			Name:  name,
			Scope: scope,
		}
	} else {
		delete(s.azureAuthorizationRoleAssignments, key)
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureAuthorizationRoleAssignmentFixture(record))
}

func (s *Server) handleAzureAuthorizationRoleAssignmentList(w http.ResponseWriter, r *http.Request, scope string) {
	filter := strings.TrimSpace(r.URL.Query().Get("$filter"))
	principalFilter := azureAuthorizationPrincipalFilter(filter)
	atScopeOnly := strings.Contains(strings.ToLower(filter), "atscope()")

	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0, len(s.azureAuthorizationRoleAssignments))
	for _, record := range s.azureAuthorizationRoleAssignments {
		if !azureAuthorizationRoleAssignmentMatches(record, scope, principalFilter, atScopeOnly) {
			continue
		}
		items = append(items, azureAuthorizationRoleAssignmentFixture(record))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureAuthorizationString(items[i]["id"])) < strings.ToLower(azureAuthorizationString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureAuthorizationRoleDefinitionCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, scope, name string) {
	_, payload, ok := azureAuthorizationReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureAuthorizationMap(payload["properties"])
	roleName := strings.TrimSpace(azureAuthorizationString(properties["roleName"]))
	if roleName == "" {
		respondAzureInvalidRequest(w, path, "properties.roleName is required")
		return
	}
	permissionsValue, ok := properties["permissions"].([]any)
	if !ok || len(permissionsValue) == 0 {
		respondAzureInvalidRequest(w, path, "properties.permissions is required")
		return
	}
	assignableScopesValue, ok := properties["assignableScopes"].([]any)
	if !ok || len(assignableScopesValue) == 0 {
		respondAzureInvalidRequest(w, path, "properties.assignableScopes is required")
		return
	}

	permissions := make([]azureAuthorizationPermission, 0, len(permissionsValue))
	for _, rawPermission := range permissionsValue {
		permissionMap := azureAuthorizationMap(rawPermission)
		permissions = append(permissions, azureAuthorizationPermission{
			Actions:    azureAuthorizationStringSlice(permissionMap["actions"]),
			NotActions: azureAuthorizationStringSlice(permissionMap["notActions"]),
		})
	}

	assignableScopes := azureAuthorizationStringSlice(properties["assignableScopes"])
	if len(assignableScopes) == 0 {
		respondAzureInvalidRequest(w, path, "properties.assignableScopes is required")
		return
	}

	record := &azureAuthorizationRoleDefinition{
		ID:               azureAuthorizationRoleDefinitionID(scope, name),
		Name:             name,
		Scope:            scope,
		RoleName:         roleName,
		Description:      strings.TrimSpace(azureAuthorizationString(properties["description"])),
		RoleType:         strings.TrimSpace(azureAuthorizationString(properties["type"])),
		AssignableScopes: assignableScopes,
		Permissions:      permissions,
	}
	if record.RoleType == "" {
		record.RoleType = "CustomRole"
	}

	s.providerStorageMu.Lock()
	s.azureAuthorizationRoleDefinitions[azureAuthorizationStorageKey(record.ID)] = record
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusCreated, azureAuthorizationRoleDefinitionFixture(record))
}

func (s *Server) handleAzureAuthorizationRoleDefinitionGet(w http.ResponseWriter, path, scope, name string) {
	s.providerStorageMu.Lock()
	record := s.azureAuthorizationRoleDefinitions[azureAuthorizationStorageKey(azureAuthorizationRoleDefinitionID(scope, name))]
	s.providerStorageMu.Unlock()
	if record == nil {
		azureAuthorizationNotFound(w, path, "RoleDefinitionNotFound", "role definition not found")
		return
	}
	respondJSON(w, http.StatusOK, azureAuthorizationRoleDefinitionFixture(record))
}

func (s *Server) handleAzureAuthorizationRoleDefinitionDelete(w http.ResponseWriter, path, scope, name string) {
	id := azureAuthorizationRoleDefinitionID(scope, name)
	key := azureAuthorizationStorageKey(id)

	s.providerStorageMu.Lock()
	record := s.azureAuthorizationRoleDefinitions[key]
	if record == nil {
		record = &azureAuthorizationRoleDefinition{
			ID:    id,
			Name:  name,
			Scope: scope,
		}
	} else {
		delete(s.azureAuthorizationRoleDefinitions, key)
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, azureAuthorizationRoleDefinitionFixture(record))
}

func (s *Server) handleAzureAuthorizationRoleDefinitionList(w http.ResponseWriter, r *http.Request, scope string) {
	roleNameFilter := azureAuthorizationRoleNameFilter(strings.TrimSpace(r.URL.Query().Get("$filter")))

	s.providerStorageMu.Lock()
	items := make([]map[string]any, 0, len(s.azureAuthorizationRoleDefinitions))
	for _, record := range s.azureAuthorizationRoleDefinitions {
		if roleNameFilter != "" && !strings.EqualFold(record.RoleName, roleNameFilter) {
			continue
		}
		if !azureAuthorizationRoleDefinitionMatches(record, scope) {
			continue
		}
		items = append(items, azureAuthorizationRoleDefinitionFixture(record))
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(azureAuthorizationString(items[i]["id"])) < strings.ToLower(azureAuthorizationString(items[j]["id"]))
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func azureAuthorizationProviderIndex(segments []string) int {
	for i := 0; i+1 < len(segments); i++ {
		if strings.EqualFold(segments[i], "providers") && strings.EqualFold(segments[i+1], "Microsoft.Authorization") {
			return i
		}
	}
	return -1
}

func azureAuthorizationScopeFromSegments(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}
	return azureAuthorizationNormalizeScope("/" + strings.Join(segments, "/"))
}

func azureAuthorizationNormalizeScope(scope string) string {
	scope = "/" + strings.Trim(strings.TrimSpace(scope), "/")
	if scope == "/" {
		return scope
	}
	return strings.TrimRight(scope, "/")
}

func azureAuthorizationRoleAssignmentID(scope, name string) string {
	return azureAuthorizationNormalizeScope(scope) + "/providers/Microsoft.Authorization/roleAssignments/" + strings.TrimSpace(name)
}

func azureAuthorizationRoleDefinitionID(scope, name string) string {
	return azureAuthorizationNormalizeScope(scope) + "/providers/Microsoft.Authorization/roleDefinitions/" + strings.TrimSpace(name)
}

func azureAuthorizationStorageKey(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

func azureAuthorizationRoleAssignmentMatches(record *azureAuthorizationRoleAssignment, scope, principalID string, atScopeOnly bool) bool {
	if record == nil {
		return false
	}
	recordScope := azureAuthorizationNormalizeScope(record.Scope)
	scope = azureAuthorizationNormalizeScope(scope)
	if atScopeOnly {
		if !strings.EqualFold(recordScope, scope) {
			return false
		}
	} else if !azureAuthorizationScopeContains(scope, recordScope) {
		return false
	}
	if principalID != "" && !strings.EqualFold(record.PrincipalID, principalID) {
		return false
	}
	return true
}

func azureAuthorizationRoleDefinitionMatches(record *azureAuthorizationRoleDefinition, scope string) bool {
	if record == nil {
		return false
	}
	scope = azureAuthorizationNormalizeScope(scope)
	for _, assignableScope := range record.AssignableScopes {
		if azureAuthorizationScopeContains(scope, assignableScope) || azureAuthorizationScopeContains(assignableScope, scope) {
			return true
		}
	}
	return strings.EqualFold(azureAuthorizationNormalizeScope(record.Scope), scope)
}

func azureAuthorizationScopeContains(parentScope, childScope string) bool {
	parentScope = azureAuthorizationNormalizeScope(parentScope)
	childScope = azureAuthorizationNormalizeScope(childScope)
	if strings.EqualFold(parentScope, "/") {
		return true
	}
	parentLower := strings.ToLower(parentScope)
	childLower := strings.ToLower(childScope)
	return childLower == parentLower || strings.HasPrefix(childLower, parentLower+"/")
}

func azureAuthorizationPrincipalFilter(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return ""
	}
	lower := strings.ToLower(filter)
	index := strings.Index(lower, "principalid eq")
	if index < 0 {
		return ""
	}
	value := strings.TrimSpace(filter[index+len("principalId eq"):])
	value = strings.Trim(value, "'\"")
	return strings.TrimSpace(value)
}

func azureAuthorizationRoleNameFilter(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return ""
	}
	lower := strings.ToLower(filter)
	index := strings.Index(lower, "rolename eq")
	if index < 0 {
		return ""
	}
	value := strings.TrimSpace(filter[index+len("roleName eq"):])
	value = strings.Trim(value, "'\"")
	return strings.TrimSpace(value)
}

func azureAuthorizationRoleAssignmentFixture(record *azureAuthorizationRoleAssignment) map[string]any {
	return map[string]any{
		"id":   record.ID,
		"name": record.Name,
		"type": "Microsoft.Authorization/roleAssignments",
		"properties": map[string]any{
			"principalId":      record.PrincipalID,
			"roleDefinitionId": record.RoleDefinitionID,
			"scope":            record.Scope,
		},
	}
}

func azureAuthorizationRoleDefinitionFixture(record *azureAuthorizationRoleDefinition) map[string]any {
	permissions := make([]map[string]any, 0, len(record.Permissions))
	for _, permission := range record.Permissions {
		permissions = append(permissions, map[string]any{
			"actions":    permission.Actions,
			"notActions": permission.NotActions,
		})
	}
	return map[string]any{
		"id":   record.ID,
		"name": record.Name,
		"type": "Microsoft.Authorization/roleDefinitions",
		"properties": map[string]any{
			"assignableScopes": record.AssignableScopes,
			"description":      record.Description,
			"permissions":      permissions,
			"roleName":         record.RoleName,
			"type":             record.RoleType,
		},
	}
}

func azureAuthorizationMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureAuthorizationString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureAuthorizationStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(azureAuthorizationString(item))
		if text != "" {
			result = append(result, text)
		}
	}
	return result
}

func azureAuthorizationReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		respondAzureInvalidRequest(w, path, "unable to read request body")
		return nil, nil, false
	}
	var payload map[string]any
	if len(strings.TrimSpace(string(body))) != 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			respondAzureInvalidRequest(w, path, "request body must be valid JSON")
			return nil, nil, false
		}
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return body, payload, true
}

func azureAuthorizationNotFound(w http.ResponseWriter, path, code, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"provider": providerAzure,
		"path":     path,
	})
}
