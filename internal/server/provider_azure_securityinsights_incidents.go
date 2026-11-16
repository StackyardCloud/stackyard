package server

import (
	"encoding/csv"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	azureSentinelIncidentStatuses      = []string{"New", "Active", "Closed"}
	azureSentinelClassifications       = []string{"Undetermined", "TruePositive", "BenignPositive", "FalsePositive"}
	azureSentinelClassificationReasons = []string{"SuspiciousActivity", "SuspiciousButExpected", "IncorrectAlertLogic", "InaccurateData"}
	// azureSentinelReasonsFor maps each classification to the reasons it allows.
	azureSentinelReasonsFor = map[string][]string{
		"TruePositive":   {"SuspiciousActivity"},
		"BenignPositive": {"SuspiciousButExpected"},
		"FalsePositive":  {"IncorrectAlertLogic", "InaccurateData"},
	}
	// Watchlist names and aliases are 3-64 characters, alphanumeric at
	// both ends, with spaces, hyphens and underscores between.
	azureSentinelWatchlistName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 _-]{1,62}[A-Za-z0-9]$`)
	azureSentinelRelationName  = regexp.MustCompile(`^[a-zA-Z0-9-]{3,63}$`)
)

// Local watchlist uploads are limited to 3.8 MB.
const azureSentinelMaxWatchlistContent = 3_800_000

func azureSentinelRegisterIncidents() {
	registerAzureARMResourceType(azureSentinelType("incidents", azureARMResourceType{
		Validate:           azureSentinelIncidentValidate,
		ReadOnlyProperties: azureSentinelIncidentProperties,
		Linked:             azureSentinelOnboarded(azureSentinelIncidentNumber),
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"alerts": func(*Server, *azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"value": []any{}}
			},
			"entities": func(*Server, *azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"entities": []any{}, "metaData": []any{}}
			},
			"bookmarks": azureSentinelIncidentBookmarks,
		},
	}))
	registerAzureARMResourceType(azureSentinelType("incidents/comments", azureARMResourceType{
		Validate: func(payload map[string]any) string {
			return azureSentinelRequireString(azureARMObject(payload["properties"]), "message")
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return azureARMMerge(azureSentinelTimes(rec), map[string]any{"author": azureSentinelClient})
		},
	}))
	registerAzureARMResourceType(azureSentinelType("incidents/relations", azureARMResourceType{
		ValidateName: func(name string) string {
			if !azureSentinelRelationName.MatchString(name) {
				return "Relation names must be 3-63 letters, digits or hyphens."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			return azureSentinelRequireString(azureARMObject(payload["properties"]), "relatedResourceId")
		},
		Linked: azureSentinelRelationLinked,
	}))
	registerAzureARMResourceType(azureSentinelType("incidents/tasks", azureARMResourceType{
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return azureSentinelFirst(azureSentinelRequireString(props, "title"), azureSentinelOneOf(props, "status", true, []string{"New", "Completed"}))
		},
		ReadOnlyProperties: azureSentinelTimes,
	}))
	registerAzureARMResourceType(azureSentinelType("bookmarks", azureARMResourceType{
		Validate: azureSentinelBookmarkValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"created": rec.CreatedAt.Format(time.RFC3339), "updated": rec.UpdatedAt.Format(time.RFC3339)}
		},
		Linked: azureSentinelOnboarded(nil),
	}))
	items := azureSentinelType("watchlists/watchlistItems", azureARMResourceType{
		Validate: func(payload map[string]any) string {
			if len(azureARMObject(azureARMObject(payload["properties"])["itemsKeyValue"])) == 0 {
				return "properties.itemsKeyValue is required and must name at least one column."
			}
			return ""
		},
		ReadOnlyProperties: azureSentinelWatchlistItemProperties,
	})
	registerAzureARMResourceType(azureSentinelType("watchlists", azureARMResourceType{
		ValidateName: func(name string) string {
			if !azureSentinelWatchlistName.MatchString(name) {
				return "Watchlist aliases must be 3-64 characters, start and end with a letter or digit, and contain only letters, digits, spaces, hyphens and underscores."
			}
			return ""
		},
		Validate:           azureSentinelWatchlistValidate,
		ReadOnlyProperties: azureSentinelWatchlistProperties,
		Linked:             azureSentinelOnboarded(nil),
		// A new watchlist's rawContent becomes its items.
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			rows, _ := azureSentinelWatchlistRows(parent.Properties)
			children := make([]*azureARMResource, 0, len(rows))
			for i, row := range rows {
				name := uuid.NewSHA1(uuid.NameSpaceURL, []byte(parent.ID+"#"+strconv.Itoa(i))).String()
				children = append(children, azureARMChild(parent, items, name, map[string]any{"itemsKeyValue": row}))
			}
			return children
		},
		// Deleting a watchlist is a long-running operation that answers 202
		// or 204; 204 is final for any poller.
		DeleteStatus: http.StatusNoContent,
	}))
	registerAzureARMResourceType(items)
}

func azureSentinelTimes(rec *azureARMResource) map[string]any {
	return map[string]any{"createdTimeUtc": rec.CreatedAt.Format(time.RFC3339), "lastModifiedTimeUtc": rec.UpdatedAt.Format(time.RFC3339)}
}

func azureSentinelIncidentValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	msg := azureSentinelFirst(
		azureSentinelRequireString(props, "title"),
		azureSentinelOneOf(props, "severity", true, azureSentinelSeverities),
		azureSentinelOneOf(props, "status", true, azureSentinelIncidentStatuses),
		azureSentinelOneOf(props, "classification", false, azureSentinelClassifications),
		azureSentinelOneOf(props, "classificationReason", false, azureSentinelClassificationReasons),
		azureSentinelOneOf(azureARMObject(props["owner"]), "ownerType", false, []string{"Unknown", "User", "Group"}),
	)
	if msg != "" {
		return msg
	}
	classification := azureARMString(props["classification"])
	if props["status"] == "Closed" && classification == "" {
		return "properties.classification is required to close an incident."
	}
	if reason, present := props["classificationReason"]; present {
		if allowed := azureSentinelReasonsFor[classification]; !slices.Contains(allowed, azureARMString(reason)) {
			return "properties.classificationReason " + azureARMString(reason) + " doesn't apply to classification " + classification + "."
		}
	}
	for _, raw := range azureARMSlice(props["labels"]) {
		if strings.TrimSpace(azureARMString(azureARMObject(raw)["labelName"])) == "" {
			return "properties.labels items need a labelName."
		}
	}
	return ""
}

func azureSentinelIncidentProperties(rec *azureARMResource) map[string]any {
	labels := []any{}
	for _, raw := range azureARMSlice(rec.Properties["labels"]) {
		labels = append(labels, azureARMMerge(azureARMObject(raw), map[string]any{"labelType": "User"}))
	}
	return azureARMMerge(azureSentinelTimes(rec), map[string]any{
		"labels": labels, "providerName": "Azure Sentinel", "relatedAnalyticRuleIds": []any{},
		"incidentUrl": "https://portal.azure.com/#asset/Microsoft_Azure_Security_Insights/Incident" + rec.ID,
		"additionalData": map[string]any{
			"alertsCount": 0, "bookmarksCount": 0, "commentsCount": 0, "alertProductNames": []any{}, "tactics": []any{},
		},
	})
}

// azureSentinelIncidentNumber keeps an incident's number, or gives a new
// incident the workspace's next number, which is never reused. The last
// number lives in a record of an unregistered type, so no route serves it
// and it is deleted with the workspace. Caller holds providerStorageMu.
func azureSentinelIncidentNumber(s *Server, rec *azureARMResource) map[string]any {
	number := 0
	if existing := s.azureARMResources[azureARMKey(rec.ID)]; existing != nil {
		number, _ = existing.Properties["incidentNumber"].(int)
	} else {
		counterID := azureSentinelWorkspace(rec.ID) + "/providers/" + azureSentinelNamespace + "/incidentCounters/default"
		if counter := s.azureARMResources[azureARMKey(counterID)]; counter != nil {
			number, _ = counter.Properties["last"].(int)
		}
		number++
		s.azureARMResources[azureARMKey(counterID)] = &azureARMResource{
			ID: counterID, Type: azureSentinelNamespace + "/incidentCounters", Properties: map[string]any{"last": number},
		}
	}
	return map[string]any{"incidentNumber": number, "providerIncidentId": strconv.Itoa(number)}
}

// azureSentinelIncidentBookmarks lists the bookmarks an incident's
// relations name.
func azureSentinelIncidentBookmarks(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	prefix := azureARMKey(rec.ID + "/relations/")
	relations := s.azureARMResourcesWhere(func(other *azureARMResource) bool { return strings.HasPrefix(azureARMKey(other.ID), prefix) })
	value := make([]any, 0, len(relations))
	for _, relation := range relations {
		if bookmark, _ := s.getAzureARMResource(azureARMString(relation.Properties["relatedResourceId"]), "", "", ""); bookmark != nil {
			value = append(value, bookmark.fixture())
		}
	}
	return http.StatusOK, map[string]any{"value": value}
}

// azureSentinelRelationLinked requires a relation to name a bookmark in
// the incident's workspace and describes it. Caller holds providerStorageMu.
func azureSentinelRelationLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	related := azureARMString(rec.Properties["relatedResourceId"])
	bookmark := s.azureARMResources[azureARMKey(related)]
	prefix := azureARMKey(azureSentinelWorkspace(rec.ID) + "/providers/" + azureSentinelNamespace + "/bookmarks/")
	if bookmark == nil || !strings.HasPrefix(azureARMKey(related), prefix) {
		return nil, nil, "properties.relatedResourceId must be the ID of a bookmark in the incident's workspace."
	}
	return map[string]any{
		"relatedResourceName": bookmark.Name, "relatedResourceType": bookmark.Type, "relatedResourceKind": bookmark.Kind,
	}, nil, ""
}

func azureSentinelBookmarkValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := azureSentinelRequireString(props, "displayName", "query"); msg != "" {
		return msg
	}
	times := map[string]time.Time{}
	for _, key := range []string{"eventTime", "queryStartTime", "queryEndTime"} {
		raw, present := props[key]
		if !present {
			continue
		}
		t, err := time.Parse(time.RFC3339, azureARMString(raw))
		if err != nil {
			return "properties." + key + " must be an RFC 3339 date and time."
		}
		times[key] = t
	}
	start, hasStart := times["queryStartTime"]
	if end, hasEnd := times["queryEndTime"]; hasStart && hasEnd && end.Before(start) {
		return "properties.queryEndTime must not be before properties.queryStartTime."
	}
	return ""
}

func azureSentinelWatchlistValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	msg := azureSentinelFirst(
		azureSentinelRequireString(props, "displayName", "provider", "itemsSearchKey"),
		azureSentinelOneOf(props, "sourceType", false, []string{"Local", "AzureStorage"}),
	)
	if msg != "" {
		return msg
	}
	if !azureSentinelWatchlistName.MatchString(azureARMString(props["displayName"])) {
		return "properties.displayName must be 3-64 characters, start and end with a letter or digit, and contain only letters, digits, spaces, hyphens and underscores."
	}
	if raw, present := props["defaultDuration"]; present {
		if _, ok := azureSentinelDurationOf(raw); !ok {
			return "properties.defaultDuration must be an ISO 8601 duration."
		}
	}
	if _, present := props["rawContent"]; !present {
		return ""
	}
	if !strings.EqualFold(azureARMString(props["contentType"]), "text/csv") {
		return "properties.contentType must be text/csv when properties.rawContent is set."
	}
	if len(azureARMString(props["rawContent"])) > azureSentinelMaxWatchlistContent {
		return "properties.rawContent can be at most 3.8 MB."
	}
	if raw, present := props["numberOfLinesToSkip"]; present && !azureWholeNumber(raw, 0, 2147483647) {
		return "properties.numberOfLinesToSkip must be a whole number of at least 0."
	}
	_, msg = azureSentinelWatchlistRows(props)
	return msg
}

// azureSentinelWatchlistRows parses a watchlist's rawContent: after
// numberOfLinesToSkip lines, a CSV header that includes itemsSearchKey,
// then one item per row.
func azureSentinelWatchlistRows(props map[string]any) ([]map[string]any, string) {
	raw, present := props["rawContent"]
	if !present {
		return nil, ""
	}
	skip, _ := props["numberOfLinesToSkip"].(float64)
	lines := strings.SplitN(azureARMString(raw), "\n", int(skip)+1)
	if len(lines) <= int(skip) {
		return nil, "properties.rawContent has no header after properties.numberOfLinesToSkip lines."
	}
	records, err := csv.NewReader(strings.NewReader(lines[len(lines)-1])).ReadAll()
	if err != nil || len(records) == 0 {
		return nil, "properties.rawContent must be CSV with a header row and the same number of fields in every row."
	}
	header := records[0]
	if !slices.Contains(header, azureARMString(props["itemsSearchKey"])) {
		return nil, "properties.itemsSearchKey must be a column of the rawContent header."
	}
	rows := make([]map[string]any, 0, len(records)-1)
	for _, record := range records[1:] {
		row := make(map[string]any, len(header))
		for i, column := range header {
			row[column] = record[i]
		}
		rows = append(rows, row)
	}
	return rows, ""
}

func azureSentinelWatchlistProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"watchlistId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(rec.ID)).String(), "watchlistAlias": rec.Name,
		"watchlistType": "watchlist", "tenantId": azureDefaultTenantID, "isDeleted": false,
		"created": rec.CreatedAt.Format(time.RFC3339), "updated": rec.UpdatedAt.Format(time.RFC3339),
		"createdBy": azureSentinelClient, "updatedBy": azureSentinelClient,
	}
}

func azureSentinelWatchlistItemProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"watchlistItemId": rec.Name, "watchlistItemType": "watchlist-item", "tenantId": azureDefaultTenantID, "isDeleted": false,
		"created": rec.CreatedAt.Format(time.RFC3339), "updated": rec.UpdatedAt.Format(time.RFC3339),
		"createdBy": azureSentinelClient, "updatedBy": azureSentinelClient,
	}
}
