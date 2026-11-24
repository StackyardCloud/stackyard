package server

import (
	"net/http"
	"strings"
)

// Event Hubs routes outside the shared ARM handler's resource model:
//
//   - a geo-DR alias's authorization rules, which are the primary
//     namespace's rules seen through the alias, with alias connection
//     strings;
//   - reconciling a network security perimeter configuration.

// evhServeExtraRoutes handles the routes above, returning false for anything
// else.
func (s *Server) evhServeExtraRoutes(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	// subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.EventHub/namespaces/{ns}/...
	if len(seg) < 10 || !strings.EqualFold(seg[2], "resourceGroups") || !strings.EqualFold(seg[4], "providers") ||
		!strings.EqualFold(seg[5], evhNamespace) || !strings.EqualFold(seg[6], "namespaces") {
		return false
	}
	namespaceID := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/" + evhNamespace + "/namespaces/" + seg[7]
	rest := seg[8:]
	switch {
	case len(rest) >= 3 && strings.EqualFold(rest[0], "disasterRecoveryConfigs") && strings.EqualFold(rest[2], "authorizationRules"):
		if hasAzureInvalidAPIVersion(r) {
			azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
			return true
		}
		s.evhServeAliasRules(w, r, namespaceID, rest[1], rest[3:])
		return true
	case len(rest) == 3 && strings.EqualFold(rest[0], "networkSecurityPerimeterConfigurations") && strings.EqualFold(rest[2], "reconcile"):
		s.serveAzureARMPost(w, r, func(map[string]any) (int, any) {
			if s.azureARMLookup(namespaceID) == nil {
				return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The namespace "+namespaceID+" was not found.")
			}
			return http.StatusAccepted, nil
		})
		return true
	}
	return false
}

// evhServeAliasRules serves GET {alias}/authorizationRules[/{rule}] and POST
// {alias}/authorizationRules/{rule}/listKeys; tail is what follows
// authorizationRules.
func (s *Server) evhServeAliasRules(w http.ResponseWriter, r *http.Request, namespaceID, aliasName string, tail []string) {
	alias := s.azureARMLookup(namespaceID + "/disasterRecoveryConfigs/" + aliasName)
	if alias == nil {
		azureARMNotFound(w, "ParentResourceNotFound", namespaceID+"/disasterRecoveryConfigs/"+aliasName, namespaceID)
		return
	}
	rules := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, evhNamespace+"/namespaces/authorizationRules") && strings.HasPrefix(azureARMKey(rec.ID), azureARMKey(namespaceID)+"/")
	})
	view := func(rule *azureARMResource) map[string]any {
		out := rule.fixture()
		out["id"] = alias.ID + "/authorizationRules/" + rule.Name
		out["type"] = evhNamespace + "/namespaces/disasterRecoveryConfigs/authorizationRules"
		return out
	}
	find := func(name string) *azureARMResource {
		for _, rule := range rules {
			if strings.EqualFold(rule.Name, name) {
				return rule
			}
		}
		return nil
	}
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		items := make([]map[string]any, 0, len(rules))
		for _, rule := range rules {
			items = append(items, view(rule))
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": items})
	case len(tail) == 1 && r.Method == http.MethodGet:
		if rule := find(tail[0]); rule != nil {
			respondJSON(w, http.StatusOK, view(rule))
			return
		}
		azureARMNotFound(w, "ResourceNotFound", alias.ID+"/authorizationRules/"+tail[0], "")
	case len(tail) == 2 && strings.EqualFold(tail[1], "listKeys") && r.Method == http.MethodPost:
		rule := find(tail[0])
		if rule == nil {
			azureARMNotFound(w, "ResourceNotFound", alias.ID+"/authorizationRules/"+tail[0], "")
			return
		}
		primary, secondary := s.azureResourceKey(rule, "primary"), s.azureResourceKey(rule, "secondary")
		conn := func(key string) string {
			return "Endpoint=sb://" + strings.ToLower(alias.Name) + ".servicebus.windows.net/;SharedAccessKeyName=" + rule.Name + ";SharedAccessKey=" + key
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"keyName": rule.Name, "primaryKey": primary, "secondaryKey": secondary,
			"aliasPrimaryConnectionString": conn(primary), "aliasSecondaryConnectionString": conn(secondary),
		})
	default:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
	}
}
