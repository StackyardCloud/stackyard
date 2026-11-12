package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureResourceGraphAPIVersion = "2024-04-01"
const azureResourceGraphHistoryAPIVersion = "2021-06-01-preview"
const azureResourceGraphChangesAPIVersion = "2020-09-01-preview"

type azureResourceGraphQuery struct {
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

type azureResourceGraphResourceRecord struct {
	ID             string
	Name           string
	Type           string
	SubscriptionID string
	ResourceGroup  string
	Location       string
	Tags           map[string]string
	Properties     map[string]any
}

func (s *Server) handleAzureResourceGraphRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureIsResourceGraphPath(segments) {
		return false
	}
	expectedVersion, ok := azureResourceGraphExpectedAPIVersion(segments)
	if !ok {
		return false
	}
	if !azureResourceGraphHasExpectedAPIVersion(r, expectedVersion) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureResourceGraphAllowedMethod(r.Method) {
		return false
	}

	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ResourceGraph") {
		s.handleAzureResourceGraphProviderRoute(w, r, path, segments[2:])
		return true
	}

	subscriptionID, resourceGroup, tail, ok := azureResourceGraphSubscriptionContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	s.handleAzureResourceGraphQueriesRoute(w, r, path, subscriptionID, resourceGroup, tail)
	return true
}

func azureIsResourceGraphPath(segments []string) bool {
	if len(segments) >= 2 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ResourceGraph") {
		if len(segments) < 3 {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(segments[2])) {
		case "operations", "resources", "resourceshistory", "resourcechanges", "resourcechangedetails":
			return true
		default:
			return false
		}
	}

	_, _, tail, ok := azureResourceGraphSubscriptionContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	return strings.EqualFold(tail[0], "queries")
}

func azureResourceGraphExpectedAPIVersion(segments []string) (string, bool) {
	if len(segments) >= 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ResourceGraph") {
		switch strings.ToLower(strings.TrimSpace(segments[2])) {
		case "operations", "resources":
			return azureResourceGraphAPIVersion, true
		case "resourceshistory":
			return azureResourceGraphHistoryAPIVersion, true
		case "resourcechanges", "resourcechangedetails":
			return azureResourceGraphChangesAPIVersion, true
		default:
			return "", false
		}
	}
	_, _, tail, ok := azureResourceGraphSubscriptionContext(segments)
	if !ok || len(tail) == 0 {
		return "", false
	}
	if strings.EqualFold(tail[0], "queries") {
		return azureResourceGraphAPIVersion, true
	}
	return "", false
}

func azureResourceGraphHasExpectedAPIVersion(r *http.Request, expected string) bool {
	values, ok := r.URL.Query()["api-version"]
	if !ok || len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) != expected {
			return false
		}
	}
	return true
}

func azureResourceGraphAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodPost, http.MethodHead:
		return true
	default:
		return false
	}
}

func azureResourceGraphSubscriptionContext(segments []string) (subscriptionID, resourceGroup string, tail []string, ok bool) {
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
			if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ResourceGraph") {
				return subscriptionID, resourceGroup, segments[idx+2:], true
			}
		}
		return "", "", nil, false
	}
	for idx := 2; idx+1 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") && strings.EqualFold(segments[idx+1], "Microsoft.ResourceGraph") {
			return subscriptionID, "", segments[idx+2:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureResourceGraphProviderRoute(w http.ResponseWriter, r *http.Request, path string, tail []string) {
	if len(tail) == 0 {
		respondAzureImplemented(w, path)
		return
	}
	switch strings.ToLower(strings.TrimSpace(tail[0])) {
	case "operations":
		if len(tail) == 1 && r.Method == http.MethodGet {
			respondJSON(w, http.StatusOK, map[string]any{"value": azureResourceGraphOperationsFixture()})
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "resources":
		if len(tail) == 1 && r.Method == http.MethodPost {
			s.handleAzureResourceGraphResources(w, r, path)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "resourceshistory":
		if len(tail) == 1 && r.Method == http.MethodPost {
			s.handleAzureResourceGraphResourcesHistory(w, r, path)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "resourcechanges":
		if len(tail) == 1 && r.Method == http.MethodPost {
			s.handleAzureResourceGraphResourceChanges(w, r, path)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	case "resourcechangedetails":
		if len(tail) == 1 && r.Method == http.MethodPost {
			s.handleAzureResourceGraphResourceChangeDetails(w, r, path)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	default:
		respondAzureImplemented(w, path)
	}
}

func (s *Server) handleAzureResourceGraphQueriesRoute(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup string, tail []string) {
	if len(tail) == 1 && strings.EqualFold(tail[0], "queries") && r.Method == http.MethodGet {
		if resourceGroup == "" {
			s.handleAzureResourceGraphListQueriesBySubscription(w, subscriptionID)
			return
		}
		s.handleAzureResourceGraphListQueriesByResourceGroup(w, subscriptionID, resourceGroup)
		return
	}
	if resourceGroup == "" || len(tail) < 2 || !strings.EqualFold(tail[0], "queries") {
		respondAzureImplemented(w, path)
		return
	}

	name := strings.TrimSpace(tail[1])
	if !azureResourceGraphValidQueryName(name) {
		respondAzureInvalidRequest(w, path, "query resource name is invalid")
		return
	}

	if len(tail) != 2 {
		respondAzureImplemented(w, path)
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleAzureResourceGraphGetQuery(w, path, subscriptionID, resourceGroup, name, false)
	case http.MethodHead:
		s.handleAzureResourceGraphGetQuery(w, path, subscriptionID, resourceGroup, name, true)
	case http.MethodPut:
		s.handleAzureResourceGraphCreateOrUpdateQuery(w, r, path, subscriptionID, resourceGroup, name)
	case http.MethodPatch:
		s.handleAzureResourceGraphPatchQuery(w, r, path, subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		s.handleAzureResourceGraphDeleteQuery(w, subscriptionID, resourceGroup, name)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureResourceGraphResources(w http.ResponseWriter, r *http.Request, path string) {
	_, payload, ok := azureResourceGraphReadRequestBody(w, r, path)
	if !ok {
		return
	}
	query := strings.TrimSpace(azureResourceGraphString(payload["query"]))
	if query == "" {
		respondAzureInvalidRequest(w, path, "query is required")
		return
	}
	top, skip, err := azureResourceGraphPagination(payload["options"])
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}
	items := azureResourceGraphApplyQuery(azureResourceGraphFilterByScopes(azureResourceGraphSampleResources(), payload), query)
	respondJSON(w, http.StatusOK, azureResourceGraphQueryResponse(items, top, skip, payload["facets"]))
}

func (s *Server) handleAzureResourceGraphResourcesHistory(w http.ResponseWriter, r *http.Request, path string) {
	_, payload, ok := azureResourceGraphReadRequestBody(w, r, path)
	if !ok {
		return
	}
	query := strings.TrimSpace(azureResourceGraphString(payload["query"]))
	if query == "" {
		respondAzureInvalidRequest(w, path, "query is required")
		return
	}
	top, skip, err := azureResourceGraphPagination(payload["options"])
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}
	baseItems := azureResourceGraphApplyQuery(azureResourceGraphFilterByScopes(azureResourceGraphSampleResources(), payload), query)
	history := make([]map[string]any, 0, len(baseItems)*2)
	for _, item := range baseItems {
		history = append(history, map[string]any{
			"resourceId":    item["id"],
			"snapshotTime":  "2026-03-01T00:00:00Z",
			"changeType":    "Snapshot",
			"properties":    map[string]any{"provisioningState": "Succeeded", "tags": item["tags"]},
			"resourceType":  item["type"],
			"resourceGroup": item["resourceGroup"],
		})
		history = append(history, map[string]any{
			"resourceId":    item["id"],
			"snapshotTime":  "2026-04-01T00:00:00Z",
			"changeType":    "Snapshot",
			"properties":    map[string]any{"provisioningState": "Succeeded", "tags": item["tags"]},
			"resourceType":  item["type"],
			"resourceGroup": item["resourceGroup"],
		})
	}
	respondJSON(w, http.StatusOK, azureResourceGraphQueryResponse(history, top, skip, nil))
}

func (s *Server) handleAzureResourceGraphResourceChanges(w http.ResponseWriter, r *http.Request, path string) {
	_, payload, ok := azureResourceGraphReadRequestBody(w, r, path)
	if !ok {
		return
	}
	if azureResourceGraphObject(payload["interval"]) == nil {
		respondAzureInvalidRequest(w, path, "interval is required")
		return
	}
	top, skip, err := azureResourceGraphPreviewPagination(payload)
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}
	fetchPropertyChanges := azureResourceGraphBool(payload["fetchPropertyChanges"])
	fetchSnapshots := azureResourceGraphBool(payload["fetchSnapshots"])
	items := azureResourceGraphFilterByScopes(azureResourceGraphSampleResources(), payload)

	changes := make([]map[string]any, 0, len(items))
	for _, item := range items {
		changeID := azureResourceGraphStableToken("change", azureResourceGraphString(item["id"]))
		change := map[string]any{
			"changeId":      changeID,
			"resourceId":    item["id"],
			"resourceType":  item["type"],
			"changeType":    "Update",
			"changeTime":    "2026-04-01T00:00:00Z",
			"resourceGroup": item["resourceGroup"],
		}
		if fetchSnapshots {
			change["beforeSnapshot"] = map[string]any{"tags": map[string]any{"env": "dev"}}
			change["afterSnapshot"] = map[string]any{"tags": item["tags"]}
		}
		if fetchPropertyChanges {
			change["propertyChanges"] = []any{
				map[string]any{
					"propertyName": "tags.env",
					"beforeValue":  "dev",
					"afterValue":   azureResourceGraphTagValue(item["tags"], "env"),
				},
			}
		}
		changes = append(changes, change)
	}
	total := len(changes)
	if skip > total {
		skip = total
	}
	end := total
	nextToken := ""
	if top > 0 && skip+top < total {
		end = skip + top
		nextToken = strconv.Itoa(end)
	}
	if top > 0 && skip+top >= total {
		end = total
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"changes":    changes[skip:end],
		"count":      end - skip,
		"total":      total,
		"skipToken":  nextToken,
		"resultType": "ResourceChanges",
	})
}

func (s *Server) handleAzureResourceGraphResourceChangeDetails(w http.ResponseWriter, r *http.Request, path string) {
	_, payload, ok := azureResourceGraphReadRequestBody(w, r, path)
	if !ok {
		return
	}
	changeIDs := azureResourceGraphStringSlice(payload["changeIds"])
	resourceIDs := azureResourceGraphStringSlice(payload["resourceIds"])
	if len(changeIDs) == 0 || len(resourceIDs) == 0 {
		respondAzureInvalidRequest(w, path, "changeIds and resourceIds are required")
		return
	}
	details := make([]any, 0, len(changeIDs))
	for idx, changeID := range changeIDs {
		resourceID := resourceIDs[idx%len(resourceIDs)]
		details = append(details, map[string]any{
			"changeId":   changeID,
			"resourceId": resourceID,
			"changes": []any{
				map[string]any{
					"propertyName": "tags.env",
					"beforeValue":  "dev",
					"afterValue":   "stage",
				},
			},
			"beforeSnapshot": map[string]any{"tags": map[string]any{"env": "dev"}},
			"afterSnapshot":  map[string]any{"tags": map[string]any{"env": "stage"}},
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"changes": details})
}

func (s *Server) handleAzureResourceGraphCreateOrUpdateQuery(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureResourceGraphReadRequestBody(w, r, path)
	if !ok {
		return
	}
	properties := azureResourceGraphObject(payload["properties"])
	if strings.TrimSpace(azureResourceGraphString(properties["query"])) == "" {
		respondAzureInvalidRequest(w, path, "properties.query is required")
		return
	}
	location := strings.TrimSpace(azureResourceGraphString(payload["location"]))
	if location == "" {
		location = "global"
	}

	now := time.Now().UTC()
	key := azureResourceGraphQueryKey(subscriptionID, resourceGroup, name)

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureResourceGraphQueries[key]
	status := http.StatusOK
	createdAt := now
	if resource == nil {
		resource = &azureResourceGraphQuery{
			SubscriptionID: subscriptionID,
			ResourceGroup:  resourceGroup,
			Name:           name,
		}
		status = http.StatusCreated
	} else {
		createdAt = resource.CreatedAt
	}

	resource.Location = location
	resource.Tags = azureResourceGraphStringMap(payload["tags"])
	resource.Properties = azureResourceGraphNormalizeQueryProperties(properties, createdAt, now)
	resource.CreatedAt = createdAt
	resource.UpdatedAt = now
	resource.ETag = azureResourceGraphETag(key, body, now)
	s.azureResourceGraphQueries[key] = resource

	respondJSON(w, status, azureResourceGraphQueryFixture(resource))
}

func (s *Server) handleAzureResourceGraphPatchQuery(w http.ResponseWriter, r *http.Request, path, subscriptionID, resourceGroup, name string) {
	body, payload, ok := azureResourceGraphReadRequestBody(w, r, path)
	if !ok {
		return
	}
	key := azureResourceGraphQueryKey(subscriptionID, resourceGroup, name)
	now := time.Now().UTC()

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	resource := s.azureResourceGraphQueries[key]
	if resource == nil {
		respondJSON(w, http.StatusNotFound, azureResourceGraphNotFound(path, name))
		return
	}
	if len(payload) == 0 {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	updated := false
	if _, ok := payload["tags"]; ok {
		resource.Tags = azureResourceGraphStringMap(payload["tags"])
		updated = true
	}
	if rawProps, ok := payload["properties"]; ok {
		resource.Properties = azureResourceGraphMergeObjects(resource.Properties, azureResourceGraphObject(rawProps))
		resource.Properties = azureResourceGraphNormalizeQueryProperties(resource.Properties, resource.CreatedAt, now)
		updated = true
	}
	if !updated {
		respondAzureInvalidRequest(w, path, "patch body must include at least one updatable field")
		return
	}

	resource.UpdatedAt = now
	resource.ETag = azureResourceGraphETag(key, body, now)
	respondJSON(w, http.StatusOK, azureResourceGraphQueryFixture(resource))
}

func (s *Server) handleAzureResourceGraphGetQuery(w http.ResponseWriter, path, subscriptionID, resourceGroup, name string, headOnly bool) {
	s.providerStorageMu.Lock()
	resource := s.azureResourceGraphQueries[azureResourceGraphQueryKey(subscriptionID, resourceGroup, name)]
	s.providerStorageMu.Unlock()
	if resource == nil {
		if headOnly {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusNotFound, azureResourceGraphNotFound(path, name))
		return
	}
	if headOnly {
		w.WriteHeader(http.StatusOK)
		return
	}
	respondJSON(w, http.StatusOK, azureResourceGraphQueryFixture(resource))
}

func (s *Server) handleAzureResourceGraphDeleteQuery(w http.ResponseWriter, subscriptionID, resourceGroup, name string) {
	key := azureResourceGraphQueryKey(subscriptionID, resourceGroup, name)
	s.providerStorageMu.Lock()
	_, exists := s.azureResourceGraphQueries[key]
	if exists {
		delete(s.azureResourceGraphQueries, key)
	}
	s.providerStorageMu.Unlock()
	if exists {
		respondJSON(w, http.StatusOK, map[string]any{})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureResourceGraphListQueriesBySubscription(w http.ResponseWriter, subscriptionID string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, resource := range s.azureResourceGraphQueries {
		if resource.SubscriptionID != subscriptionID {
			continue
		}
		values = append(values, azureResourceGraphQueryFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureResourceGraphString(values[i]["id"]) < azureResourceGraphString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func (s *Server) handleAzureResourceGraphListQueriesByResourceGroup(w http.ResponseWriter, subscriptionID, resourceGroup string) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	values := make([]map[string]any, 0)
	for _, resource := range s.azureResourceGraphQueries {
		if resource.SubscriptionID != subscriptionID || !strings.EqualFold(resource.ResourceGroup, resourceGroup) {
			continue
		}
		values = append(values, azureResourceGraphQueryFixture(resource))
	}
	sort.Slice(values, func(i, j int) bool {
		return azureResourceGraphString(values[i]["id"]) < azureResourceGraphString(values[j]["id"])
	})
	respondJSON(w, http.StatusOK, map[string]any{"value": values})
}

func azureResourceGraphOperationsFixture() []map[string]any {
	ops := []string{
		"Microsoft.ResourceGraph/resources/read",
		"Microsoft.ResourceGraph/resourcesHistory/read",
		"Microsoft.ResourceGraph/resourceChanges/read",
		"Microsoft.ResourceGraph/resourceChangeDetails/read",
		"Microsoft.ResourceGraph/queries/read",
		"Microsoft.ResourceGraph/queries/write",
		"Microsoft.ResourceGraph/queries/delete",
	}
	out := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		out = append(out, map[string]any{
			"name": op,
			"display": map[string]any{
				"provider": "Microsoft.ResourceGraph",
				"resource": strings.TrimPrefix(op, "Microsoft.ResourceGraph/"),
			},
		})
	}
	return out
}

func azureResourceGraphQueryResponse(items []map[string]any, top, skip int, facetsRaw any) map[string]any {
	total := len(items)
	if skip > total {
		skip = total
	}
	end := total
	nextToken := ""
	if top > 0 && skip+top < total {
		end = skip + top
		nextToken = strconv.Itoa(end)
	}
	if top > 0 && skip+top >= total {
		end = total
	}
	page := make([]any, 0, end-skip)
	for _, item := range items[skip:end] {
		page = append(page, item)
	}
	response := map[string]any{
		"count":           end - skip,
		"totalRecords":    total,
		"resultTruncated": nextToken != "",
		"data":            page,
		"facets":          azureResourceGraphFacetResponse(facetsRaw, items),
	}
	if nextToken != "" {
		response["skipToken"] = nextToken
	}
	return response
}

func azureResourceGraphFacetResponse(facetsRaw any, items []map[string]any) []any {
	facets, _ := facetsRaw.([]any)
	if len(facets) == 0 {
		return []any{}
	}
	return []any{
		map[string]any{
			"expression": "type",
			"resultType": "Facet",
			"count":      len(items),
			"data": []any{
				map[string]any{"value": "Microsoft.Compute/virtualMachines", "count": 1},
				map[string]any{"value": "Microsoft.Storage/storageAccounts", "count": 1},
			},
		},
	}
}

func azureResourceGraphSampleResources() []map[string]any {
	out := make([]map[string]any, 0, 2)
	for _, item := range []azureResourceGraphResourceRecord{
		{
			ID:             "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-app/providers/Microsoft.Compute/virtualMachines/vm-a",
			Name:           "vm-a",
			Type:           "Microsoft.Compute/virtualMachines",
			SubscriptionID: "00000000-0000-0000-0000-000000000000",
			ResourceGroup:  "rg-app",
			Location:       "eastus",
			Tags:           map[string]string{"env": "stage", "owner": "stackyard"},
			Properties:     map[string]any{"provisioningState": "Succeeded", "vmSize": "Standard_B2s"},
		},
		{
			ID:             "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-data/providers/Microsoft.Storage/storageAccounts/storea",
			Name:           "storea",
			Type:           "Microsoft.Storage/storageAccounts",
			SubscriptionID: "00000000-0000-0000-0000-000000000000",
			ResourceGroup:  "rg-data",
			Location:       "eastus2",
			Tags:           map[string]string{"env": "dev", "owner": "stackyard"},
			Properties:     map[string]any{"provisioningState": "Succeeded", "primaryLocation": "eastus2"},
		},
	} {
		out = append(out, map[string]any{
			"id":             item.ID,
			"name":           item.Name,
			"type":           item.Type,
			"subscriptionId": item.SubscriptionID,
			"resourceGroup":  item.ResourceGroup,
			"location":       item.Location,
			"tags":           azureResourceGraphAnyStringMap(item.Tags),
			"properties":     azureResourceGraphMergeObjects(map[string]any{}, item.Properties),
		})
	}
	return out
}

func azureResourceGraphFilterByScopes(items []map[string]any, payload map[string]any) []map[string]any {
	subscriptions := azureResourceGraphStringSlice(payload["subscriptions"])
	if len(subscriptions) == 0 {
		return items
	}
	allowed := map[string]struct{}{}
	for _, sub := range subscriptions {
		allowed[strings.ToLower(sub)] = struct{}{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if _, ok := allowed[strings.ToLower(azureResourceGraphString(item["subscriptionId"]))]; ok {
			out = append(out, item)
		}
	}
	return out
}

func azureResourceGraphApplyQuery(items []map[string]any, query string) []map[string]any {
	lowered := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		itemType := strings.ToLower(azureResourceGraphString(item["type"]))
		if strings.Contains(lowered, "where type") {
			if strings.Contains(lowered, "storage") && !strings.Contains(itemType, "storage") {
				continue
			}
			if strings.Contains(lowered, "compute") && !strings.Contains(itemType, "compute") {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	if strings.Contains(lowered, "summarize") {
		return []map[string]any{{"count_": len(filtered)}}
	}
	return filtered
}

func azureResourceGraphQueryFixture(resource *azureResourceGraphQuery) map[string]any {
	return map[string]any{
		"id":         azureResourceGraphQueryID(resource.SubscriptionID, resource.ResourceGroup, resource.Name),
		"name":       resource.Name,
		"type":       "Microsoft.ResourceGraph/queries",
		"etag":       resource.ETag,
		"location":   resource.Location,
		"tags":       azureResourceGraphAnyStringMap(resource.Tags),
		"systemData": azureResourceGraphSystemData(resource.CreatedAt, resource.UpdatedAt),
		"properties": azureResourceGraphMergeObjects(map[string]any{}, resource.Properties),
	}
}

func azureResourceGraphNormalizeQueryProperties(payload map[string]any, createdAt, updatedAt time.Time) map[string]any {
	props := azureResourceGraphMergeObjects(map[string]any{}, payload)
	if azureResourceGraphString(props["description"]) == "" {
		props["description"] = "Stackyard Resource Graph query"
	}
	if azureResourceGraphString(props["resultKind"]) == "" {
		props["resultKind"] = "basic"
	}
	props["timeModified"] = updatedAt.Format(time.RFC3339)
	props["createdTime"] = createdAt.Format(time.RFC3339)
	return props
}

func azureResourceGraphQueryKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID + "::" + resourceGroup + "::" + name)
}

func azureResourceGraphQueryID(subscriptionID, resourceGroup, name string) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ResourceGraph/queries/%s", subscriptionID, resourceGroup, name)
}

func azureResourceGraphSystemData(createdAt, updatedAt time.Time) map[string]any {
	return map[string]any{
		"createdAt":          createdAt.Format(time.RFC3339),
		"createdBy":          "stackyard",
		"createdByType":      "User",
		"lastModifiedAt":     updatedAt.Format(time.RFC3339),
		"lastModifiedBy":     "stackyard",
		"lastModifiedByType": "User",
	}
}

func azureResourceGraphNotFound(path, name string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    "ResourceNotFound",
			"message": fmt.Sprintf("resource %q was not found for path %s", name, path),
		},
	}
}

func azureResourceGraphReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
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

func azureResourceGraphPagination(raw any) (top, skip int, err error) {
	options := azureResourceGraphObject(raw)
	if options == nil {
		return 0, 0, nil
	}
	if top, err = azureResourceGraphIntValue(options["$top"], options["top"]); err != nil {
		return 0, 0, fmt.Errorf("options.$top must be a non-negative integer")
	}
	skipToken := strings.TrimSpace(azureResourceGraphStringOr(options["$skipToken"], azureResourceGraphString(options["skipToken"])))
	if skipToken != "" {
		skip, err = strconv.Atoi(skipToken)
		if err != nil || skip < 0 {
			return 0, 0, fmt.Errorf("options.$skipToken must be a non-negative integer string")
		}
	}
	return top, skip, nil
}

func azureResourceGraphPreviewPagination(payload map[string]any) (top, skip int, err error) {
	if top, err = azureResourceGraphIntValue(payload["$top"], payload["top"]); err != nil {
		return 0, 0, fmt.Errorf("$top must be a non-negative integer")
	}
	if skipToken := strings.TrimSpace(azureResourceGraphStringOr(payload["$skipToken"], azureResourceGraphString(payload["skipToken"]))); skipToken != "" {
		skip, err = strconv.Atoi(skipToken)
		if err != nil || skip < 0 {
			return 0, 0, fmt.Errorf("$skipToken must be a non-negative integer string")
		}
	}
	return top, skip, nil
}

func azureResourceGraphIntValue(values ...any) (int, error) {
	for _, value := range values {
		switch typed := value.(type) {
		case nil:
			continue
		case float64:
			if typed < 0 || typed != float64(int(typed)) {
				return 0, fmt.Errorf("invalid")
			}
			return int(typed), nil
		case int:
			if typed < 0 {
				return 0, fmt.Errorf("invalid")
			}
			return typed, nil
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err != nil || parsed < 0 {
				return 0, fmt.Errorf("invalid")
			}
			return parsed, nil
		}
	}
	return 0, nil
}

func azureResourceGraphString(value any) string {
	if typed, ok := value.(string); ok {
		return strings.TrimSpace(typed)
	}
	return ""
}

func azureResourceGraphStringOr(value any, fallback string) string {
	if str := azureResourceGraphString(value); str != "" {
		return str
	}
	return fallback
}

func azureResourceGraphBool(value any) bool {
	typed, _ := value.(bool)
	return typed
}

func azureResourceGraphObject(value any) map[string]any {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return nil
	}
	return azureResourceGraphMergeObjects(map[string]any{}, typed)
}

func azureResourceGraphStringSlice(value any) []string {
	raw, _ := value.([]any)
	if raw == nil {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if str := azureResourceGraphString(item); str != "" {
			out = append(out, str)
		}
	}
	return out
}

func azureResourceGraphStringMap(value any) map[string]string {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(raw))
	for key, val := range raw {
		if str := azureResourceGraphString(val); str != "" {
			out[key] = str
		}
	}
	return out
}

func azureResourceGraphAnyStringMap(value map[string]string) map[string]any {
	out := make(map[string]any, len(value))
	for key, val := range value {
		out[key] = val
	}
	return out
}

func azureResourceGraphMergeObjects(base, overlay map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		if overlayMap, ok := value.(map[string]any); ok {
			if baseMap, ok := merged[key].(map[string]any); ok {
				merged[key] = azureResourceGraphMergeObjects(baseMap, overlayMap)
			} else {
				merged[key] = azureResourceGraphMergeObjects(map[string]any{}, overlayMap)
			}
			continue
		}
		merged[key] = value
	}
	return merged
}

func azureResourceGraphETag(key string, body []byte, now time.Time) string {
	sum := sha1.Sum(append([]byte(key+now.Format(time.RFC3339Nano)), body...))
	return `"` + hex.EncodeToString(sum[:8]) + `"`
}

func azureResourceGraphStableToken(parts ...string) string {
	sum := sha1.Sum([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(sum[:8])
}

func azureResourceGraphValidQueryName(name string) bool {
	if len(name) < 1 || len(name) > 260 {
		return false
	}
	for _, r := range name {
		switch r {
		case '<', '>', '&', ':', '?':
			return false
		}
	}
	return true
}

func azureResourceGraphTagValue(value any, key string) string {
	typed, _ := value.(map[string]any)
	if typed == nil {
		return ""
	}
	return azureResourceGraphString(typed[key])
}
