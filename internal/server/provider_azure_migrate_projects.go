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

const azureMigrateProjectsAPIVersion = "2018-09-01-preview"

var azureMigrateProjectsAllowedTools = map[string]struct{}{
	"serverdiscovery":             {},
	"serverassessment":            {},
	"servermigration":             {},
	"cloudamize":                  {},
	"turbonomic":                  {},
	"zerto":                       {},
	"corenttech":                  {},
	"serverassessmentv1":          {},
	"servermigration_replication": {},
	"carbonite":                   {},
	"datamigrationassistant":      {},
	"databasemigrationservice":    {},
}

var azureMigrateProjectsAllowedGoals = map[string]struct{}{
	"servers":   {},
	"databases": {},
}

type azureMigrateProject struct {
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

func (s *Server) handleAzureMigrateProjectsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsMigrateProjectsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureMigrateProjectsAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Migrate") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureMigrateProjectsOperationsFixture()})
			return true
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureMigrateProjectsRouteContext(segments)
	if !ok || len(tail) == 0 {
		respondAzureImplemented(w, path)
		return true
	}
	if !strings.EqualFold(tail[0], "migrateProjects") {
		respondAzureImplemented(w, path)
		return true
	}

	s.handleAzureMigrateProjectsRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsMigrateProjectsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Migrate") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureMigrateProjectsRouteContext(segments)
	return ok && len(tail) > 0 && strings.EqualFold(tail[0], "migrateProjects")
}

func azureMigrateProjectsRouteContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
	if len(segments) < 4 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	if subscriptionID == "" {
		return "", "", nil, false
	}
	if len(segments) >= 4 && strings.EqualFold(segments[2], "resourceGroups") {
		resourceGroup = strings.TrimSpace(segments[3])
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.Migrate") {
			return subscriptionID, resourceGroup, segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func azureMigrateProjectsAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureMigrateProjectsRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureMigrateProjectsListBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureMigrateProjectsListByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if name == "" {
		respondAzureInvalidRequest(w, path, "migrate project name is required")
		return
	}
	if len(tail) == 2 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureMigrateProjectsGet(w, path, subscriptionID, resourceGroup, name, false)
		case http.MethodHead:
			s.handleAzureMigrateProjectsGet(w, path, subscriptionID, resourceGroup, name, true)
		case http.MethodPut:
			s.handleAzureMigrateProjectsCreateOrUpdate(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodPatch:
			s.handleAzureMigrateProjectsPatch(w, r, path, subscriptionID, resourceGroup, name)
		case http.MethodDelete:
			s.handleAzureMigrateProjectsDelete(w, subscriptionID, resourceGroup, name)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	switch strings.ToLower(strings.TrimSpace(tail[2])) {
	case "registertool":
		if r.Method == http.MethodPost {
			s.handleAzureMigrateProjectsRegisterTool(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	case "refreshsummary":
		if r.Method == http.MethodPost {
			s.handleAzureMigrateProjectsRefreshSummary(w, r, path, subscriptionID, resourceGroup, name)
			return
		}
	}
	respondAzureImplemented(w, path)
}

func (s *Server) handleAzureMigrateProjectsCreateOrUpdate(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureMigrateProjectsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	location := azureMigrateProjectsString(payload["location"])
	if location == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return
	}

	now := time.Now().UTC()
	key := azureMigrateProjectsKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	project := s.azureMigrateProjects[key]
	status := http.StatusOK
	createdAt := now
	if project == nil {
		project = &azureMigrateProject{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = project.CreatedAt
	}
	project.Location = location
	project.Tags = azureMigrateProjectsStringMap(payload["tags"])
	project.Properties = azureMigrateProjectsNormalizeProperties(name, project.Properties, azureMigrateProjectsObject(payload["properties"]), createdAt, now)
	project.CreatedAt = createdAt
	project.UpdatedAt = now
	project.ETag = azureMigrateProjectsETag(key, body, now)
	s.azureMigrateProjects[key] = project

	respondJSON(w, status, azureMigrateProjectsFixture(project))
}

func (s *Server) handleAzureMigrateProjectsPatch(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureMigrateProjectsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureMigrateProjectsKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	project := s.azureMigrateProjects[key]
	if project == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateProjectsNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	now := time.Now().UTC()
	updated := false
	if _, ok := payload["location"]; ok {
		location := azureMigrateProjectsString(payload["location"])
		if location == "" {
			respondAzureInvalidRequest(w, path, "location cannot be empty")
			return
		}
		project.Location = location
		updated = true
	}
	if _, ok := payload["tags"]; ok {
		project.Tags = azureMigrateProjectsStringMap(payload["tags"])
		updated = true
	}
	if props, ok := payload["properties"]; ok {
		project.Properties = azureMigrateProjectsNormalizeProperties(name, project.Properties, azureMigrateProjectsObject(props), project.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	project.UpdatedAt = now
	project.ETag = azureMigrateProjectsETag(key, body, now)
	respondJSON(w, http.StatusOK, azureMigrateProjectsFixture(project))
}

func (s *Server) handleAzureMigrateProjectsGet(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	project := s.azureMigrateProjects[azureMigrateProjectsKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if project == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateProjectsNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateProjectsFixture(project))
}

func (s *Server) handleAzureMigrateProjectsDelete(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	key := azureMigrateProjectsKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	project := s.azureMigrateProjects[key]
	delete(s.azureMigrateProjects, key)
	s.providerStorageMu.Unlock()
	if project == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	respondJSON(w, http.StatusOK, azureMigrateProjectsFixture(project))
}

func (s *Server) handleAzureMigrateProjectsRegisterTool(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureMigrateProjectsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	tool := azureMigrateProjectsString(payload["tool"])
	if tool == "" {
		respondAzureInvalidRequest(w, path, "tool is required")
		return
	}
	if _, ok := azureMigrateProjectsAllowedTools[strings.ToLower(tool)]; !ok {
		respondAzureInvalidRequest(w, path, "tool is invalid")
		return
	}

	key := azureMigrateProjectsKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	project := s.azureMigrateProjects[key]
	if project == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateProjectsNotFound(path, name))
		return
	}

	registered := azureMigrateProjectsStringSlice(project.Properties["registeredTools"])
	exists := false
	for _, item := range registered {
		if strings.EqualFold(item, tool) {
			exists = true
			break
		}
	}
	if !exists {
		registered = append(registered, tool)
		sort.Strings(registered)
	}
	project.Properties["registeredTools"] = azureMigrateProjectsStringSliceAny(registered)
	project.UpdatedAt = time.Now().UTC()

	respondJSON(w, http.StatusOK, map[string]any{
		"isRegistered": true,
		"tool":         tool,
		"projectId":    azureMigrateProjectsID(subscriptionID, resourceGroup, name),
	})
}

func (s *Server) handleAzureMigrateProjectsRefreshSummary(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	_, payload, ok := azureMigrateProjectsReadRequestBody(w, r, path)
	if !ok {
		return
	}
	goal := azureMigrateProjectsString(payload["goal"])
	if goal == "" {
		respondAzureInvalidRequest(w, path, "goal is required")
		return
	}
	if _, ok := azureMigrateProjectsAllowedGoals[strings.ToLower(goal)]; !ok {
		respondAzureInvalidRequest(w, path, "goal is invalid")
		return
	}

	key := azureMigrateProjectsKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	project := s.azureMigrateProjects[key]
	if project == nil {
		respondJSON(w, http.StatusNotFound, azureMigrateProjectsNotFound(path, name))
		return
	}

	now := time.Now().UTC()
	project.Properties["refreshSummaryState"] = "Completed"
	project.Properties["lastSummaryRefreshedTime"] = now.Format(time.RFC3339)
	project.Properties["summary"] = azureMigrateProjectsSummary(name, goal, now)
	project.UpdatedAt = now

	respondJSON(w, http.StatusOK, map[string]any{
		"status":                   "Succeeded",
		"goal":                     goal,
		"refreshSummaryState":      "Completed",
		"lastSummaryRefreshedTime": now.Format(time.RFC3339),
		"summary":                  project.Properties["summary"],
	})
}

func (s *Server) handleAzureMigrateProjectsListBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, project := range s.azureMigrateProjects {
		if project.SubscriptionID == subscriptionID {
			items = append(items, azureMigrateProjectsFixture(project))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateProjectsString(items[i]["id"]) < azureMigrateProjectsString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func (s *Server) handleAzureMigrateProjectsListByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	items := make([]map[string]any, 0)
	for _, project := range s.azureMigrateProjects {
		if project.SubscriptionID == subscriptionID && project.ResourceGroup == resourceGroup {
			items = append(items, azureMigrateProjectsFixture(project))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return azureMigrateProjectsString(items[i]["id"]) < azureMigrateProjectsString(items[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": items})
}

func azureMigrateProjectsNormalizeProperties(name string, existing, incoming map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureMigrateProjectsMergeObject(existing, incoming)
	props["provisioningState"] = "Succeeded"
	props["projectStatus"] = "Active"
	if _, ok := props["registeredTools"]; !ok {
		props["registeredTools"] = []any{}
	} else {
		props["registeredTools"] = azureMigrateProjectsStringSliceAny(azureMigrateProjectsStringSlice(props["registeredTools"]))
	}
	if _, ok := props["refreshSummaryState"]; !ok {
		props["refreshSummaryState"] = "NotStarted"
	}
	if _, ok := props["lastSummaryRefreshedTime"]; !ok {
		props["lastSummaryRefreshedTime"] = updatedAt.Format(time.RFC3339)
	}
	if _, ok := props["summary"]; !ok {
		props["summary"] = azureMigrateProjectsSummary(name, "Servers", updatedAt)
	}
	props["lastAssessmentTime"] = updatedAt.Format(time.RFC3339)
	props["createdTimestamp"] = createdAt.Format(time.RFC3339)
	return props
}

func azureMigrateProjectsSummary(name, goal string, now time.Time) map[string]any {
	normalizedGoal := "Servers"
	if strings.EqualFold(goal, "databases") {
		normalizedGoal = "Databases"
	}
	return map[string]any{
		"goal":                     normalizedGoal,
		"instanceType":             normalizedGoal + "ProjectSummary",
		"lastSummaryRefreshedTime": now.Format(time.RFC3339),
		"discoveredCount":          2,
		"assessedCount":            1,
		"applianceCount":           1,
		"displayName":              name + " summary",
	}
}

func azureMigrateProjectsFixture(project *azureMigrateProject) map[string]any {
	return map[string]any{
		"id":       azureMigrateProjectsID(project.SubscriptionID, project.ResourceGroup, project.Name),
		"name":     project.Name,
		"type":     "Microsoft.Migrate/migrateProjects",
		"location": project.Location,
		"tags":     azureMigrateProjectsStringMapAny(project.Tags),
		"eTag":     project.ETag,
		"properties": azureMigrateProjectsMergeObject(
			map[string]any{},
			project.Properties,
		),
	}
}

func azureMigrateProjectsOperationsFixture() []map[string]any {
	return []map[string]any{
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/read", "Migrate Projects", "Read migrate projects"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/write", "Migrate Projects", "Create or update migrate projects"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/delete", "Migrate Projects", "Delete migrate projects"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/registerTool/action", "Migrate Projects", "Register migrate tool"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/refreshSummary/action", "Migrate Projects", "Refresh project summary"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/solutions/read", "Solutions", "Read migrate solutions"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/machines/read", "Machines", "Read migrate machines"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/migrateEvents/read", "Migrate Events", "Read migrate events"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/databases/read", "Databases", "Read databases"),
		azureMigrateProjectsOperation("Microsoft.Migrate/migrateProjects/databaseInstances/read", "Database Instances", "Read database instances"),
	}
}

func azureMigrateProjectsOperation(name, resource, description string) map[string]any {
	return map[string]any{
		"name": name,
		"display": map[string]any{
			"provider":    "Microsoft.Migrate",
			"resource":    resource,
			"operation":   name,
			"description": description,
		},
		"origin": "user,system",
	}
}

func azureMigrateProjectsNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("migrate project %q was not found for %s", name, path),
		},
	}
}

func azureMigrateProjectsID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Migrate/migrateProjects/%s", subscriptionID, resourceGroup, name)
}

func azureMigrateProjectsKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID)) + "::" + strings.ToLower(strings.TrimSpace(resourceGroup)) + "::" + strings.ToLower(strings.TrimSpace(name))
}

func azureMigrateProjectsETag(key string, body []byte, updatedAt time.Time) string {
	sum := sha1.Sum(append([]byte(key+"|"+updatedAt.UTC().Format(time.RFC3339Nano)+"|"), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureMigrateProjectsReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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
	return body, payload, true
}

func azureMigrateProjectsString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func azureMigrateProjectsObject(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func azureMigrateProjectsStringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	switch typed := value.(type) {
	case map[string]string:
		out := make(map[string]string, len(typed))
		for k, v := range typed {
			out[k] = v
		}
		return out
	case map[string]any:
		out := make(map[string]string, len(typed))
		for k, v := range typed {
			out[k] = azureMigrateProjectsString(v)
		}
		return out
	default:
		return map[string]string{}
	}
}

func azureMigrateProjectsStringMapAny(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for k, v := range value {
		out[k] = v
	}
	return out
}

func azureMigrateProjectsMergeObject(objects ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, object := range objects {
		for k, v := range object {
			out[k] = v
		}
	}
	return out
}

func azureMigrateProjectsStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if trimmed := azureMigrateProjectsString(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return []string{}
	}
}

func azureMigrateProjectsStringSliceAny(value []string) []any {
	out := make([]any, 0, len(value))
	for _, item := range value {
		out = append(out, item)
	}
	return out
}
