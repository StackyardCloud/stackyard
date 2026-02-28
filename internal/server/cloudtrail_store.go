package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type cloudTrailStore struct {
	mu             sync.Mutex
	nextID         int64
	trails         map[string]map[string]any
	channels       map[string]map[string]any
	dashboards     map[string]map[string]any
	eventStores    map[string]map[string]any
	queries        map[string]map[string]any
	imports        map[string]map[string]any
	resourcePolicy string
}

func newCloudTrailStore() *cloudTrailStore {
	now := time.Now().UTC()
	store := &cloudTrailStore{
		nextID:      1,
		trails:      map[string]map[string]any{},
		channels:    map[string]map[string]any{},
		dashboards:  map[string]map[string]any{},
		eventStores: map[string]map[string]any{},
		queries:     map[string]map[string]any{},
		imports:     map[string]map[string]any{},
		resourcePolicy: `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},` +
			`"Action":"cloudtrail:LookupEvents","Resource":"*"}]}`,
	}

	trailName := "stackyard-trail"
	store.trails[trailName] = map[string]any{
		"Name":                       trailName,
		"S3BucketName":               "stackyard-cloudtrail-logs",
		"IncludeGlobalServiceEvents": true,
		"IsMultiRegionTrail":         true,
		"HomeRegion":                 "us-east-1",
		"TrailARN":                   cloudTrailARN("trail", trailName),
		"LogFileValidationEnabled":   true,
		"HasCustomEventSelectors":    false,
		"HasInsightSelectors":        true,
		"IsOrganizationTrail":        false,
	}

	channelName := "stackyard-channel"
	store.channels[channelName] = map[string]any{
		"ChannelArn": cloudTrailARN("channel", channelName),
		"Name":       channelName,
		"Source":     "Custom",
		"Destinations": []any{
			map[string]any{
				"Location": "arn:aws:s3:::stackyard-cloudtrail-logs",
				"Type":     "S3",
			},
		},
	}

	dashboardName := "stackyard-dashboard"
	store.dashboards[dashboardName] = map[string]any{
		"DashboardArn": cloudTrailARN("dashboard", dashboardName),
		"Name":         dashboardName,
		"Widgets":      []any{},
		"RefreshSchedule": map[string]any{
			"Frequency": map[string]any{
				"Unit":  "HOURS",
				"Value": 1,
			},
		},
	}

	eventStoreName := "stackyard-event-data-store"
	store.eventStores[eventStoreName] = map[string]any{
		"EventDataStoreArn":             cloudTrailARN("eventdatastore", eventStoreName),
		"Name":                          eventStoreName,
		"Status":                        "ENABLED",
		"RetentionPeriod":               2557,
		"TerminationProtectionEnabled":  true,
		"MultiRegionEnabled":            true,
		"OrganizationEnabled":           false,
		"BillingMode":                   "EXTENDABLE_RETENTION_PRICING",
		"CreatedTimestamp":              now,
		"UpdatedTimestamp":              now,
		"AdvancedEventSelectors":        []any{},
		"FederationStatus":              "DISABLED",
		"FederationRoleArn":             "",
		"IngestionEnabled":              true,
		"KmsKeyId":                      "",
		"StartIngestionStatus":          map[string]any{"LatestIngestionSuccessEventID": "evt-seed"},
		"IngestionStatus":               map[string]any{"LatestIngestionSuccessEventID": "evt-seed"},
		"FederationEventDataStoreRole":  "",
		"FederationEventDataStoreOwner": "123456789012",
	}

	queryID := store.nextTokenLocked("query", 6)
	store.queries[queryID] = map[string]any{
		"QueryId":        queryID,
		"QueryString":    "SELECT eventTime, eventName FROM stackyard LIMIT 25",
		"QueryStatus":    "FINISHED",
		"CreationTime":   now,
		"DeliveryStatus": "SUCCESS",
		"QueryStatistics": map[string]any{
			"ResultsCount":      0,
			"TotalResultsCount": 0,
			"BytesScanned":      0,
		},
	}

	importID := store.nextTokenLocked("import", 6)
	store.imports[importID] = map[string]any{
		"ImportId":       importID,
		"ImportStatus":   "COMPLETED",
		"CreatedTime":    now,
		"UpdatedTime":    now,
		"Destinations":   []any{map[string]any{"Location": cloudTrailARN("eventdatastore", eventStoreName), "Type": "EventDataStore"}},
		"ImportSource":   map[string]any{"S3": map[string]any{"S3BucketAccessRoleArn": "arn:aws:iam::123456789012:role/stackyard-cloudtrail", "S3LocationUri": "s3://stackyard-cloudtrail-import"}},
		"StartEventTime": now.Add(-1 * time.Hour),
		"EndEventTime":   now,
	}

	return store
}

func (s *cloudTrailStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "AddTags", "RemoveTags",
		"DeleteChannel", "DeleteDashboard", "DeleteEventDataStore", "DeleteTrail", "DeleteResourcePolicy",
		"DeregisterOrganizationDelegatedAdmin", "DisableFederation", "EnableFederation",
		"PutEventConfiguration", "PutEventSelectors", "PutInsightSelectors", "RegisterOrganizationDelegatedAdmin",
		"StartEventDataStoreIngestion", "StopEventDataStoreIngestion", "StartLogging", "StopLogging",
		"StopImport":
		return map[string]any{}

	case "CreateTrail":
		name := cloudTrailDefaultString(payload, "Name", "stackyard-trail")
		trail := s.ensureTrailLocked(name)
		if bucket := cloudTrailDefaultString(payload, "S3BucketName", "stackyard-cloudtrail-logs"); bucket != "" {
			trail["S3BucketName"] = bucket
		}
		return cloudTrailCloneMap(trail)
	case "UpdateTrail":
		name := cloudTrailDefaultString(payload, "Name", "stackyard-trail")
		trail := s.ensureTrailLocked(name)
		if bucket := cloudTrailPayloadString(payload, "S3BucketName"); bucket != "" {
			trail["S3BucketName"] = bucket
		}
		if homeRegion := cloudTrailPayloadString(payload, "HomeRegion"); homeRegion != "" {
			trail["HomeRegion"] = homeRegion
		}
		return cloudTrailCloneMap(trail)
	case "GetTrail":
		name := cloudTrailNameFromARN(cloudTrailPayloadString(payload, "Name"))
		trail := s.ensureTrailLocked(name)
		return map[string]any{"Trail": cloudTrailCloneMap(trail)}
	case "DescribeTrails":
		return map[string]any{"trailList": s.listTrailsLocked()}
	case "ListTrails":
		return map[string]any{"Trails": s.listTrailsLocked(), "NextToken": ""}
	case "GetTrailStatus":
		return map[string]any{
			"IsLogging":                true,
			"LatestDeliveryError":      "",
			"LatestNotificationError":  "",
			"StartLoggingTime":         time.Now().UTC().Add(-1 * time.Hour),
			"StopLoggingTime":          time.Time{},
			"LatestCloudWatchLogsTime": time.Now().UTC(),
			"LatestDigestDeliveryTime": time.Now().UTC(),
			"LatestDeliveryTime":       time.Now().UTC(),
		}
	case "GetEventSelectors":
		return map[string]any{
			"TrailARN": cloudTrailARN("trail", cloudTrailDefaultString(payload, "TrailName", "stackyard-trail")),
			"EventSelectors": []any{
				map[string]any{
					"ReadWriteType":           "All",
					"IncludeManagementEvents": true,
					"DataResources":           []any{},
				},
			},
			"AdvancedEventSelectors": []any{},
		}
	case "GetInsightSelectors":
		return map[string]any{
			"TrailARN": cloudTrailARN("trail", cloudTrailDefaultString(payload, "TrailName", "stackyard-trail")),
			"InsightSelectors": []any{
				map[string]any{"InsightType": "ApiCallRateInsight"},
			},
		}

	case "CreateChannel", "UpdateChannel":
		name := cloudTrailDefaultString(payload, "Name", "stackyard-channel")
		channel := s.ensureChannelLocked(name)
		return map[string]any{"ChannelArn": channel["ChannelArn"]}
	case "GetChannel":
		name := cloudTrailNameFromARN(cloudTrailPayloadString(payload, "Channel"))
		return map[string]any{"Channel": cloudTrailCloneMap(s.ensureChannelLocked(name))}
	case "ListChannels":
		return map[string]any{"Channels": s.listChannelsLocked(), "NextToken": ""}

	case "CreateDashboard", "UpdateDashboard":
		name := cloudTrailDefaultString(payload, "Name", "stackyard-dashboard")
		dashboard := s.ensureDashboardLocked(name)
		return map[string]any{"DashboardArn": dashboard["DashboardArn"]}
	case "GetDashboard":
		name := cloudTrailNameFromARN(cloudTrailPayloadString(payload, "DashboardId"))
		dashboard := s.ensureDashboardLocked(name)
		return map[string]any{
			"DashboardArn":    dashboard["DashboardArn"],
			"RefreshSchedule": dashboard["RefreshSchedule"],
			"Widgets":         dashboard["Widgets"],
		}
	case "ListDashboards":
		return map[string]any{"Dashboards": s.listDashboardsLocked(), "NextToken": ""}
	case "StartDashboardRefresh":
		dashboardName := cloudTrailDefaultString(payload, "DashboardId", "stackyard-dashboard")
		dashboard := s.ensureDashboardLocked(dashboardName)
		return map[string]any{
			"DashboardArn": dashboard["DashboardArn"],
			"RefreshId":    s.nextTokenLocked("refresh", 6),
		}

	case "CreateEventDataStore", "UpdateEventDataStore":
		name := cloudTrailDefaultString(payload, "Name", "stackyard-event-data-store")
		store := s.ensureEventStoreLocked(name)
		return map[string]any{"EventDataStoreArn": store["EventDataStoreArn"]}
	case "GetEventDataStore":
		name := cloudTrailNameFromARN(cloudTrailPayloadString(payload, "EventDataStore"))
		return cloudTrailCloneMap(s.ensureEventStoreLocked(name))
	case "ListEventDataStores":
		return map[string]any{"EventDataStores": s.listEventStoresLocked(), "NextToken": ""}
	case "RestoreEventDataStore":
		name := cloudTrailNameFromARN(cloudTrailPayloadString(payload, "EventDataStore"))
		store := s.ensureEventStoreLocked(name)
		store["Status"] = "ENABLED"
		return map[string]any{"EventDataStoreArn": store["EventDataStoreArn"]}

	case "StartQuery":
		queryID := s.nextTokenLocked("query", 6)
		query := map[string]any{
			"QueryId":      queryID,
			"QueryString":  cloudTrailDefaultString(payload, "QueryStatement", "SELECT * FROM stackyard LIMIT 25"),
			"QueryStatus":  "RUNNING",
			"CreationTime": time.Now().UTC(),
			"QueryStatistics": map[string]any{
				"ResultsCount":      0,
				"TotalResultsCount": 0,
				"BytesScanned":      0,
			},
		}
		s.queries[queryID] = query
		return map[string]any{"QueryId": queryID}
	case "CancelQuery":
		queryID := cloudTrailDefaultString(payload, "QueryId", s.firstQueryIDLocked())
		query := s.ensureQueryLocked(queryID)
		query["QueryStatus"] = "CANCELLED"
		return map[string]any{"QueryId": queryID, "QueryStatus": "CANCELLED"}
	case "DescribeQuery":
		queryID := cloudTrailDefaultString(payload, "QueryId", s.firstQueryIDLocked())
		query := s.ensureQueryLocked(queryID)
		return cloudTrailCloneMap(query)
	case "GetQueryResults":
		queryID := cloudTrailDefaultString(payload, "QueryId", s.firstQueryIDLocked())
		query := s.ensureQueryLocked(queryID)
		query["QueryStatus"] = "FINISHED"
		return map[string]any{
			"QueryStatus":     query["QueryStatus"],
			"QueryStatistics": query["QueryStatistics"],
			"QueryResultRows": []any{},
			"NextToken":       "",
		}
	case "ListQueries":
		return map[string]any{"Queries": s.listQueriesLocked(), "NextToken": ""}
	case "GenerateQuery":
		return map[string]any{"QueryStatement": "SELECT eventTime, eventName FROM stackyard LIMIT 25"}
	case "SearchSampleQueries":
		return map[string]any{
			"SearchResults": []any{
				map[string]any{
					"SearchResultId": "stackyard-sample-query",
					"Query":          "SELECT eventSource, eventName FROM stackyard LIMIT 100",
				},
			},
		}

	case "StartImport":
		importID := s.nextTokenLocked("import", 6)
		s.imports[importID] = map[string]any{
			"ImportId":     importID,
			"ImportStatus": "INITIALIZING",
			"CreatedTime":  time.Now().UTC(),
			"UpdatedTime":  time.Now().UTC(),
		}
		return map[string]any{"ImportId": importID}
	case "GetImport":
		importID := cloudTrailDefaultString(payload, "ImportId", s.firstImportIDLocked())
		return cloudTrailCloneMap(s.ensureImportLocked(importID))
	case "ListImports":
		return map[string]any{"Imports": s.listImportsLocked(), "NextToken": ""}
	case "ListImportFailures":
		return map[string]any{"Failures": []any{}, "NextToken": ""}

	case "GetResourcePolicy":
		return map[string]any{"ResourcePolicy": s.resourcePolicy}
	case "PutResourcePolicy":
		s.resourcePolicy = cloudTrailDefaultString(payload, "ResourcePolicy", s.resourcePolicy)
		return map[string]any{}

	case "GetEventConfiguration":
		return map[string]any{
			"EventSelectors": []any{},
			"InsightSelectors": []any{
				map[string]any{"InsightType": "ApiCallRateInsight"},
			},
		}

	case "ListPublicKeys":
		return map[string]any{
			"PublicKeyList": []any{
				map[string]any{
					"Value":               "seed-public-key",
					"Fingerprint":         "AA:BB:CC:DD",
					"ValidityStartTime":   time.Now().UTC().Add(-24 * time.Hour),
					"ValidityEndTime":     time.Now().UTC().Add(24 * time.Hour),
					"ValidatingAccountId": "123456789012",
				},
			},
			"NextToken": "",
		}
	case "ListTags":
		return map[string]any{
			"ResourceTagList": []any{
				map[string]any{
					"ResourceId": cloudTrailARN("trail", "stackyard-trail"),
					"TagsList": []any{
						map[string]any{"Key": "service", "Value": "cloudtrail"},
						map[string]any{"Key": "seed", "Value": "true"},
					},
				},
			},
		}
	case "LookupEvents":
		return map[string]any{
			"Events": []any{
				map[string]any{
					"EventId":            "stackyard-event-000001",
					"EventName":          "CreateTrail",
					"EventSource":        "cloudtrail.amazonaws.com",
					"EventTime":          time.Now().UTC().Add(-1 * time.Hour),
					"Username":           "stackyard",
					"CloudTrailEvent":    "{}",
					"Resources":          []any{},
					"AccessKeyId":        "stackyard",
					"ReadOnly":           "false",
					"EventVersion":       "1.09",
					"RecipientAccountId": "123456789012",
				},
			},
			"NextToken": "",
		}
	case "ListInsightsData":
		return map[string]any{
			"EventSources": []any{
				map[string]any{
					"EventSource": "cloudtrail.amazonaws.com",
					"Insights":    []any{},
				},
			},
		}
	case "ListInsightsMetricData":
		return map[string]any{
			"EventSource": "cloudtrail.amazonaws.com",
			"ErrorCode":   "",
			"Values":      []any{},
		}
	}

	return map[string]any{}
}

func (s *cloudTrailStore) ensureTrailLocked(name string) map[string]any {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		normalized = "stackyard-trail"
	}
	if existing, ok := s.trails[normalized]; ok {
		return existing
	}
	trail := map[string]any{
		"Name":                       normalized,
		"S3BucketName":               "stackyard-cloudtrail-logs",
		"IncludeGlobalServiceEvents": true,
		"IsMultiRegionTrail":         true,
		"HomeRegion":                 "us-east-1",
		"TrailARN":                   cloudTrailARN("trail", normalized),
		"LogFileValidationEnabled":   true,
		"HasCustomEventSelectors":    false,
		"HasInsightSelectors":        true,
		"IsOrganizationTrail":        false,
	}
	s.trails[normalized] = trail
	return trail
}

func (s *cloudTrailStore) ensureChannelLocked(name string) map[string]any {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		normalized = "stackyard-channel"
	}
	if existing, ok := s.channels[normalized]; ok {
		return existing
	}
	channel := map[string]any{
		"ChannelArn": cloudTrailARN("channel", normalized),
		"Name":       normalized,
		"Source":     "Custom",
		"Destinations": []any{
			map[string]any{
				"Location": cloudTrailARN("eventdatastore", "stackyard-event-data-store"),
				"Type":     "EventDataStore",
			},
		},
	}
	s.channels[normalized] = channel
	return channel
}

func (s *cloudTrailStore) ensureDashboardLocked(name string) map[string]any {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		normalized = "stackyard-dashboard"
	}
	if existing, ok := s.dashboards[normalized]; ok {
		return existing
	}
	dashboard := map[string]any{
		"DashboardArn": cloudTrailARN("dashboard", normalized),
		"Name":         normalized,
		"Widgets":      []any{},
		"RefreshSchedule": map[string]any{
			"Frequency": map[string]any{
				"Unit":  "HOURS",
				"Value": 1,
			},
		},
	}
	s.dashboards[normalized] = dashboard
	return dashboard
}

func (s *cloudTrailStore) ensureEventStoreLocked(name string) map[string]any {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		normalized = "stackyard-event-data-store"
	}
	if existing, ok := s.eventStores[normalized]; ok {
		return existing
	}
	eventStore := map[string]any{
		"EventDataStoreArn":            cloudTrailARN("eventdatastore", normalized),
		"Name":                         normalized,
		"Status":                       "ENABLED",
		"RetentionPeriod":              2557,
		"TerminationProtectionEnabled": true,
		"MultiRegionEnabled":           true,
		"OrganizationEnabled":          false,
		"BillingMode":                  "EXTENDABLE_RETENTION_PRICING",
		"CreatedTimestamp":             time.Now().UTC(),
		"UpdatedTimestamp":             time.Now().UTC(),
		"AdvancedEventSelectors":       []any{},
	}
	s.eventStores[normalized] = eventStore
	return eventStore
}

func (s *cloudTrailStore) ensureQueryLocked(queryID string) map[string]any {
	id := strings.TrimSpace(queryID)
	if id == "" {
		id = s.firstQueryIDLocked()
	}
	if existing, ok := s.queries[id]; ok {
		return existing
	}
	query := map[string]any{
		"QueryId":      id,
		"QueryString":  "SELECT * FROM stackyard LIMIT 25",
		"QueryStatus":  "FINISHED",
		"CreationTime": time.Now().UTC(),
		"QueryStatistics": map[string]any{
			"ResultsCount":      0,
			"TotalResultsCount": 0,
			"BytesScanned":      0,
		},
	}
	s.queries[id] = query
	return query
}

func (s *cloudTrailStore) ensureImportLocked(importID string) map[string]any {
	id := strings.TrimSpace(importID)
	if id == "" {
		id = s.firstImportIDLocked()
	}
	if existing, ok := s.imports[id]; ok {
		return existing
	}
	imp := map[string]any{
		"ImportId":     id,
		"ImportStatus": "COMPLETED",
		"CreatedTime":  time.Now().UTC(),
		"UpdatedTime":  time.Now().UTC(),
	}
	s.imports[id] = imp
	return imp
}

func (s *cloudTrailStore) firstQueryIDLocked() string {
	if len(s.queries) == 0 {
		id := s.nextTokenLocked("query", 6)
		s.ensureQueryLocked(id)
		return id
	}
	keys := make([]string, 0, len(s.queries))
	for key := range s.queries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *cloudTrailStore) firstImportIDLocked() string {
	if len(s.imports) == 0 {
		id := s.nextTokenLocked("import", 6)
		s.ensureImportLocked(id)
		return id
	}
	keys := make([]string, 0, len(s.imports))
	for key := range s.imports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *cloudTrailStore) listTrailsLocked() []any {
	keys := make([]string, 0, len(s.trails))
	for key := range s.trails {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, cloudTrailCloneMap(s.trails[key]))
	}
	return out
}

func (s *cloudTrailStore) listChannelsLocked() []any {
	keys := make([]string, 0, len(s.channels))
	for key := range s.channels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		ch := s.channels[key]
		out = append(out, map[string]any{
			"ChannelArn": ch["ChannelArn"],
			"Name":       ch["Name"],
		})
	}
	return out
}

func (s *cloudTrailStore) listDashboardsLocked() []any {
	keys := make([]string, 0, len(s.dashboards))
	for key := range s.dashboards {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		d := s.dashboards[key]
		out = append(out, map[string]any{
			"DashboardArn": d["DashboardArn"],
			"Name":         d["Name"],
		})
	}
	return out
}

func (s *cloudTrailStore) listEventStoresLocked() []any {
	keys := make([]string, 0, len(s.eventStores))
	for key := range s.eventStores {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		store := s.eventStores[key]
		out = append(out, map[string]any{
			"EventDataStoreArn":            store["EventDataStoreArn"],
			"Name":                         store["Name"],
			"Status":                       store["Status"],
			"TerminationProtectionEnabled": store["TerminationProtectionEnabled"],
			"RetentionPeriod":              store["RetentionPeriod"],
		})
	}
	return out
}

func (s *cloudTrailStore) listQueriesLocked() []any {
	keys := make([]string, 0, len(s.queries))
	for key := range s.queries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		query := s.queries[key]
		out = append(out, map[string]any{
			"QueryId":      query["QueryId"],
			"QueryStatus":  query["QueryStatus"],
			"CreationTime": query["CreationTime"],
		})
	}
	return out
}

func (s *cloudTrailStore) listImportsLocked() []any {
	keys := make([]string, 0, len(s.imports))
	for key := range s.imports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		imp := s.imports[key]
		out = append(out, map[string]any{
			"ImportId":     imp["ImportId"],
			"ImportStatus": imp["ImportStatus"],
			"CreatedTime":  imp["CreatedTime"],
			"UpdatedTime":  imp["UpdatedTime"],
		})
	}
	return out
}

func (s *cloudTrailStore) nextTokenLocked(prefix string, width int) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%0*d", prefix, width, id)
}

func cloudTrailPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	}
	return ""
}

func cloudTrailDefaultString(payload map[string]any, key, fallback string) string {
	if value := cloudTrailPayloadString(payload, key); value != "" {
		return value
	}
	return fallback
}

func cloudTrailNameFromARN(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "arn:") {
		return trimmed
	}
	if slash := strings.LastIndex(trimmed, "/"); slash >= 0 && slash+1 < len(trimmed) {
		return trimmed[slash+1:]
	}
	if colon := strings.LastIndex(trimmed, ":"); colon >= 0 && colon+1 < len(trimmed) {
		return trimmed[colon+1:]
	}
	return trimmed
}

func cloudTrailARN(resource, name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		trimmed = "stackyard"
	}
	if strings.HasPrefix(trimmed, "arn:") {
		return trimmed
	}
	return fmt.Sprintf("arn:aws:cloudtrail:us-east-1:123456789012:%s/%s", resource, trimmed)
}

func cloudTrailCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = cloudTrailCloneMap(typed)
		case []any:
			out[key] = cloudTrailCloneSlice(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func cloudTrailCloneSlice(in []any) []any {
	if in == nil {
		return []any{}
	}
	out := make([]any, len(in))
	for i, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[i] = cloudTrailCloneMap(typed)
		case []any:
			out[i] = cloudTrailCloneSlice(typed)
		default:
			out[i] = typed
		}
	}
	return out
}
