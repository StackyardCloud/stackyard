package server

import (
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Helpers shared by the Database Watcher, Databricks, Datadog, Data Factory
// and Data Lake emulations.

// azureDataPlatAll runs checks in order and returns the first failure message.
func azureDataPlatAll(checks ...func(map[string]any) string) func(map[string]any) string {
	return func(payload map[string]any) string {
		for _, check := range checks {
			if msg := check(payload); msg != "" {
				return msg
			}
		}
		return ""
	}
}

// azureDataPlatRequired requires each properties.{key} to be present and not
// empty. A dotted key reaches into nested objects, e.g. "userInfo.emailAddress".
func azureDataPlatRequired(keys ...string) func(map[string]any) string {
	return func(payload map[string]any) string {
		for _, key := range keys {
			if azureDataPlatEmpty(azureDataPlatLookup(azureARMObject(payload["properties"]), key)) {
				return "properties." + key + " is required."
			}
		}
		return ""
	}
}

// azureDataPlatLookup returns the value at a dotted path in m, or nil.
func azureDataPlatLookup(m map[string]any, path string) any {
	var cur any = m
	for _, part := range strings.Split(path, ".") {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[part]
	}
	return cur
}

func azureDataPlatEmpty(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	}
	return false
}

// azureDataPlatEnum requires the value at a dotted properties path, when present,
// to be one of values.
func azureDataPlatEnum(path string, values ...string) func(map[string]any) string {
	return func(payload map[string]any) string {
		raw := azureDataPlatLookup(azureARMObject(payload["properties"]), path)
		if raw != nil && !slices.Contains(values, azureARMString(raw)) {
			return "properties." + path + " must be one of " + strings.Join(values, ", ") + "."
		}
		return ""
	}
}

// azureDataPlatIPv4Range requires properties.startIpAddress and endIpAddress to be
// IPv4 addresses with start not after end, as firewall rules need.
func azureDataPlatIPv4Range(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	start := net.ParseIP(azureARMString(props["startIpAddress"])).To4()
	end := net.ParseIP(azureARMString(props["endIpAddress"])).To4()
	switch {
	case start == nil:
		return "properties.startIpAddress must be an IPv4 address."
	case end == nil:
		return "properties.endIpAddress must be an IPv4 address."
	case slices.Compare(start, end) > 0:
		return "properties.startIpAddress must not be after properties.endIpAddress."
	}
	return ""
}

// azureDataPlatOnlyName returns a ValidateName hook for singleton resources whose
// only valid name is name, such as Datadog's "default" configurations.
func azureDataPlatOnlyName(name string) func(string) string {
	return func(got string) string {
		if got != name {
			return "The only name allowed here is '" + name + "'."
		}
		return ""
	}
}

// azureDataPlatOperations returns an operations view listing each op, a
// "{resource}/{verb}" path such as "watchers/read", under namespace.
func azureDataPlatOperations(namespace, provider string, ops ...string) azureARMView {
	return func(*Server, string) []map[string]any {
		out := make([]map[string]any, 0, len(ops))
		for _, op := range ops {
			resource, _, _ := strings.Cut(op, "/")
			out = append(out, map[string]any{
				"name": namespace + "/" + op, "isDataAction": false, "origin": "user,system",
				"display": map[string]any{"provider": provider, "resource": resource, "operation": op, "description": op},
			})
		}
		return out
	}
}

// azureDataPlatChildrenLocked returns the stored records of fullType directly
// under parentID, oldest first. Caller holds providerStorageMu.
func (s *Server) azureDataPlatChildrenLocked(parentID, fullType string) []*azureARMResource {
	prefix := azureARMKey(parentID + "/")
	var out []*azureARMResource
	for key, rec := range s.azureARMResources {
		// A direct child's key ends in "{collection}/{name}".
		if strings.HasPrefix(key, prefix) && strings.EqualFold(rec.Type, fullType) && strings.Count(key[len(prefix):], "/") == 1 {
			out = append(out, rec)
		}
	}
	slices.SortFunc(out, func(a, b *azureARMResource) int {
		if c := a.CreatedAt.Compare(b.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

// azureDataPlatPutChildLocked stores a record of fullType named name under
// parent, keeping at most limit of that type there by dropping the oldest.
// Caller holds providerStorageMu.
func (s *Server) azureDataPlatPutChildLocked(parent *azureARMResource, fullType, collection, name string, props map[string]any, limit int) *azureARMResource {
	id := parent.ID + "/" + collection + "/" + name
	now := time.Now().UTC()
	created := now
	if existing := s.azureARMResources[azureARMKey(id)]; existing != nil {
		created = existing.CreatedAt
	} else if siblings := s.azureDataPlatChildrenLocked(parent.ID, fullType); len(siblings) >= limit {
		for _, old := range siblings[:len(siblings)-limit+1] {
			delete(s.azureARMResources, azureARMKey(old.ID))
		}
	}
	rec := &azureARMResource{
		ID: id, SubscriptionID: parent.SubscriptionID, ResourceGroup: parent.ResourceGroup, Name: name, Type: fullType,
		Properties: props, CreatedAt: created, UpdatedAt: now,
	}
	s.azureARMResources[azureARMKey(id)] = rec
	return rec
}

// azureDataPlatStoredLocked returns the stored record with id when it has
// fullType, or nil. Caller holds providerStorageMu.
func (s *Server) azureDataPlatStoredLocked(id, fullType string) *azureARMResource {
	rec := s.azureARMResources[azureARMKey(id)]
	if rec == nil || !strings.EqualFold(rec.Type, fullType) {
		return nil
	}
	return rec
}

// azureDataPlatCheckName returns a checkNameAvailability location action for
// fullType, whose names must pass valid (a non-empty message rejects).
func azureDataPlatCheckName(fullType string, valid func(string) string) azureARMLocationAction {
	return func(s *Server, sub, _ string, payload map[string]any) (int, any) {
		name := azureARMString(payload["name"])
		if !strings.EqualFold(azureARMString(payload["type"]), fullType) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be "+fullType+".")
		}
		if msg := valid(name); msg != "" {
			return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "InvalidName", "message": msg}
		}
		taken := len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
			return strings.EqualFold(rec.Type, fullType) && strings.EqualFold(rec.Name, name)
		})) > 0
		if taken {
			return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "The name '" + name + "' is already in use."}
		}
		return http.StatusOK, map[string]any{"nameAvailable": true}
	}
}

// azureDataPlatAccountNameError checks the 3 to 24 lowercase letters and digits
// Data Lake account names allow.
func azureDataPlatAccountNameError(name string) string {
	if len(name) < 3 || len(name) > 24 || strings.IndexFunc(name, func(r rune) bool { return (r < 'a' || r > 'z') && (r < '0' || r > '9') }) >= 0 {
		return "The account name '" + name + "' must be 3 to 24 lowercase letters and digits."
	}
	return ""
}

// azureDataPlatGUID returns a stable GUID derived from seed, such as a
// resource ID, for server-owned identifiers.
func azureDataPlatGUID(seed string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.ToLower(seed))).String()
}

// azureDataPlatLocationScope splits a ".../subscriptions/{sub}/providers/
// {namespace}/locations/{location}" scope into its subscription and location.
func azureDataPlatLocationScope(scope string) (sub, location string) {
	seg := splitPathSegments(scope)
	if len(seg) >= 6 && strings.EqualFold(seg[0], "subscriptions") {
		return seg[1], seg[5]
	}
	return "", ""
}

// azureDataPlatAccountCount counts the subscription's accounts of fullType
// in location.
func (s *Server) azureDataPlatAccountCount(sub, location, fullType string) int {
	return len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, fullType) && strings.EqualFold(rec.SubscriptionID, sub) &&
			strings.EqualFold(strings.ReplaceAll(rec.Location, " ", ""), strings.ReplaceAll(location, " ", ""))
	}))
}

// azureDataPlatAccountUsage is the account-count usage item a regional
// usages list returns.
func azureDataPlatAccountUsage(s *Server, scope, fullType string, limit int) map[string]any {
	sub, location := azureDataPlatLocationScope(scope)
	return map[string]any{
		"id": scope + "/usages/AccountCount", "unit": "Count",
		"currentValue": s.azureDataPlatAccountCount(sub, location, fullType), "limit": limit,
		"name": map[string]any{"value": "AccountCount", "localizedValue": "Account Count"},
	}
}

// serveAzureDataPlatCapability serves GET /subscriptions/{sub}/providers/
// {namespace}/locations/{location}/capability, a single object.
func (s *Server) serveAzureDataPlatCapability(w http.ResponseWriter, r *http.Request, namespace, fullType string, limit int) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) != 7 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "providers") ||
		!strings.EqualFold(seg[3], namespace) || !strings.EqualFold(seg[4], "locations") || !strings.EqualFold(seg[6], "capability") {
		return false
	}
	if r.Method != http.MethodGet {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"subscriptionId": seg[1], "state": "Registered", "maxAccountCount": limit,
		"accountCount": s.azureDataPlatAccountCount(seg[1], seg[5], fullType), "migrationState": false,
	})
	return true
}
