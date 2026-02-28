package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type quickSightStore struct {
	mu sync.Mutex

	nextID int64

	entities    map[string]map[string]map[string]any
	tags        map[string]map[string]string
	permissions map[string][]any
}

func newQuickSightStore() *quickSightStore {
	s := &quickSightStore{
		nextID:      2,
		entities:    map[string]map[string]map[string]any{},
		tags:        map[string]map[string]string{},
		permissions: map[string][]any{},
	}
	s.seedLocked(time.Now().UTC())
	return s
}

func (s *quickSightStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	s.seedLocked(now)

	ctx := quickSightMergeMaps(payload, pathParams, query)
	accountID := quickSightString(ctx, "AwsAccountId", "123456789012")
	kind := quickSightKindFromContext(ctx)
	entityID := quickSightPrimaryID(ctx, kind)
	entity := s.ensureEntityLocked(kind, entityID, accountID, now)
	actionLC := strings.ToLower(strings.TrimSpace(action))

	// Explicit responses for example coverage and common QuickSight flows.
	switch action {
	case "DescribeAccountSettings":
		return map[string]any{
			"AccountSettings": map[string]any{
				"AccountName":                    "stackyard-quicksight",
				"DefaultNamespace":               "default",
				"NotificationEmail":              "stackyard@example.com",
				"TerminationProtectionEnabled":   false,
				"PublicSharingEnabled":           true,
				"FolderResourceValidations":      "STRICT",
				"AutomaticallyCreateGroups":      true,
				"DisallowUserRegistration":       false,
				"RestrictIdentityType":           false,
				"UseIAMIdentityCenterForAccount": false,
			},
			"Status":    200,
			"RequestId": "stackyard-request",
		}
	case "ListDashboards":
		return quickSightListResponse("DashboardSummaryList", s.listEntitiesLocked("dashboard"), now)
	case "ListAnalyses":
		return quickSightListResponse("AnalysisSummaryList", s.listEntitiesLocked("analysis"), now)
	case "ListDataSets":
		return quickSightListResponse("DataSetSummaries", s.listEntitiesLocked("dataset"), now)
	case "ListDataSources":
		return quickSightListResponse("DataSources", s.listEntitiesLocked("datasource"), now)
	case "ListNamespaces":
		return quickSightListResponse("Namespaces", s.listEntitiesLocked("namespace"), now)
	case "ListUsers":
		return quickSightListResponse("UserList", s.listEntitiesLocked("user"), now)
	case "ListThemes":
		return quickSightListResponse("ThemeSummaryList", s.listEntitiesLocked("theme"), now)
	case "ListTemplates":
		return quickSightListResponse("TemplateSummaryList", s.listEntitiesLocked("template"), now)
	case "ListVPCConnections":
		return quickSightListResponse("VPCConnectionSummaries", s.listEntitiesLocked("vpcconnection"), now)
	case "TagResource":
		arn := quickSightString(ctx, "ResourceArn", quickSightARN(kind, entityID, accountID))
		tagMap := s.ensureTagsLocked(arn)
		for key, value := range quickSightTagsFromPayload(payload) {
			tagMap[key] = value
		}
		return quickSightSuccess(now)
	case "UntagResource":
		arn := quickSightString(ctx, "ResourceArn", quickSightARN(kind, entityID, accountID))
		tagMap := s.ensureTagsLocked(arn)
		for _, key := range quickSightTagKeys(payload, query) {
			delete(tagMap, key)
		}
		return quickSightSuccess(now)
	case "ListTagsForResource":
		arn := quickSightString(ctx, "ResourceArn", quickSightARN(kind, entityID, accountID))
		tagMap := s.ensureTagsLocked(arn)
		return map[string]any{
			"Tags":      quickSightTagsToList(tagMap),
			"TagList":   quickSightTagsToList(tagMap),
			"Status":    200,
			"RequestId": "stackyard-request",
		}
	case "GenerateEmbedUrlForAnonymousUser",
		"GenerateEmbedUrlForRegisteredUser",
		"GenerateEmbedUrlForRegisteredUserWithIdentity",
		"GetDashboardEmbedUrl",
		"GetSessionEmbedUrl":
		return map[string]any{
			"EmbedUrl":  "https://stackyard.local/quicksight/embed/" + action,
			"Status":    200,
			"RequestId": "stackyard-request",
		}
	case "GetIdentityContext":
		return map[string]any{
			"IdentityContext": "stackyard-identity-context",
			"Status":          200,
			"RequestId":       "stackyard-request",
		}
	case "GetFlowMetadata", "GetFlowPermissions":
		flow := s.ensureEntityLocked("flow", quickSightPrimaryID(ctx, "flow"), accountID, now)
		if action == "GetFlowPermissions" {
			return map[string]any{
				"Permissions": s.ensurePermissionsLocked(quickSightARN("flow", quickSightString(flow, "id", "flow-000001"), accountID)),
				"Status":      200,
				"RequestId":   "stackyard-request",
			}
		}
		return map[string]any{
			"FlowId":       quickSightString(flow, "id", "flow-000001"),
			"FlowArn":      quickSightString(flow, "arn", quickSightARN("flow", "flow-000001", accountID)),
			"Status":       "ACTIVE",
			"RequestId":    "stackyard-request",
			"AwsAccountId": accountID,
		}
	}

	if strings.HasSuffix(action, "Permissions") || strings.HasSuffix(action, "Permission") {
		arn := quickSightString(ctx, "ResourceArn", quickSightARN(kind, entityID, accountID))
		perms := s.ensurePermissionsLocked(arn)
		if strings.HasPrefix(action, "Update") {
			perms = s.mergePermissions(payload, arn)
		}
		return map[string]any{
			"Permissions": perms,
			"Status":      200,
			"RequestId":   "stackyard-request",
		}
	}

	if strings.HasPrefix(action, "List") {
		listKey := quickSightListKey(action, kind)
		items := s.listEntitiesLocked(kind)
		return quickSightListResponse(listKey, items, now)
	}

	if strings.HasPrefix(action, "Describe") || strings.HasPrefix(action, "Get") {
		responseKey := quickSightDescribeKey(action, kind)
		return map[string]any{
			responseKey:    quickSightCloneMap(entity),
			"Status":       200,
			"RequestId":    "stackyard-request",
			"AwsAccountId": accountID,
		}
	}

	if strings.HasPrefix(action, "Delete") {
		s.deleteEntityLocked(kind, entityID)
		return quickSightSuccess(now)
	}

	if strings.HasPrefix(action, "BatchDelete") {
		s.deleteEntityLocked(kind, entityID)
		return map[string]any{
			"Errors":     []any{},
			"Successful": []any{quickSightString(ctx, "id", entityID)},
			"Status":     200,
			"RequestId":  "stackyard-request",
		}
	}

	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "BatchCreate") || strings.HasPrefix(action, "BatchUpdate") {
		updated := s.updateEntityLocked(kind, entityID, payload, accountID, now)
		if strings.HasPrefix(action, "Batch") {
			return map[string]any{
				"Errors":     []any{},
				"Successful": []any{quickSightCloneMap(updated)},
				"Status":     200,
				"RequestId":  "stackyard-request",
			}
		}
		return map[string]any{
			"Arn":          quickSightARN(kind, entityID, accountID),
			"Resource":     quickSightCloneMap(updated),
			"Status":       200,
			"RequestId":    "stackyard-request",
			"AwsAccountId": accountID,
		}
	}

	if strings.HasPrefix(actionLC, "start") || strings.HasPrefix(actionLC, "stop") || strings.HasPrefix(actionLC, "cancel") || strings.HasPrefix(actionLC, "put") {
		updated := s.updateEntityLocked(kind, entityID, payload, accountID, now)
		updated["status"] = "SUCCEEDED"
		return map[string]any{
			"Resource":     quickSightCloneMap(updated),
			"Status":       200,
			"RequestId":    "stackyard-request",
			"AwsAccountId": accountID,
		}
	}

	return map[string]any{
		"Operation":    action,
		"Status":       200,
		"RequestId":    "stackyard-request",
		"AwsAccountId": accountID,
	}
}

func (s *quickSightStore) seedLocked(now time.Time) {
	accountID := "123456789012"
	s.ensureEntityLocked("analysis", "analysis-000001", accountID, now)
	s.ensureEntityLocked("dashboard", "dashboard-000001", accountID, now)
	s.ensureEntityLocked("dataset", "dataset-000001", accountID, now)
	s.ensureEntityLocked("datasource", "datasource-000001", accountID, now)
	s.ensureEntityLocked("namespace", "default", accountID, now)
	s.ensureEntityLocked("user", "stackyard-user", accountID, now)
	s.ensureEntityLocked("theme", "theme-000001", accountID, now)
	s.ensureEntityLocked("template", "template-000001", accountID, now)
	s.ensureEntityLocked("vpcconnection", "vpc-connection-000001", accountID, now)
	s.ensureEntityLocked("topic", "topic-000001", accountID, now)
	s.ensureEntityLocked("folder", "folder-000001", accountID, now)
	s.ensureEntityLocked("brand", "brand-000001", accountID, now)
	s.ensureEntityLocked("actionconnector", "action-connector-000001", accountID, now)
	s.ensureEntityLocked("flow", "flow-000001", accountID, now)
	s.ensureEntityLocked("custompermission", "custom-permissions-000001", accountID, now)
	s.ensureEntityLocked("group", "stackyard-group", accountID, now)
}

func (s *quickSightStore) ensureEntityBucketLocked(kind string) map[string]map[string]any {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		kind = "resource"
	}
	if s.entities[kind] == nil {
		s.entities[kind] = map[string]map[string]any{}
	}
	return s.entities[kind]
}

func (s *quickSightStore) ensureEntityLocked(kind, id, accountID string, now time.Time) map[string]any {
	bucket := s.ensureEntityBucketLocked(kind)
	id = strings.TrimSpace(id)
	if id == "" {
		id = quickSightDefaultIDForKind(kind)
	}
	if existing := bucket[id]; existing != nil {
		return existing
	}
	ts := now.Format(time.RFC3339)
	item := map[string]any{
		"id":              id,
		"kind":            kind,
		"name":            id,
		"arn":             quickSightARN(kind, id, accountID),
		"createdTime":     ts,
		"lastUpdatedTime": ts,
		"status":          "ACTIVE",
	}
	bucket[id] = item
	return item
}

func (s *quickSightStore) updateEntityLocked(kind, id string, payload map[string]any, accountID string, now time.Time) map[string]any {
	item := s.ensureEntityLocked(kind, id, accountID, now)
	for key, value := range payload {
		item[key] = value
	}
	item["lastUpdatedTime"] = now.Format(time.RFC3339)
	item["arn"] = quickSightARN(kind, id, accountID)
	return item
}

func (s *quickSightStore) deleteEntityLocked(kind, id string) {
	bucket := s.ensureEntityBucketLocked(kind)
	delete(bucket, id)
}

func (s *quickSightStore) listEntitiesLocked(kind string) []any {
	bucket := s.ensureEntityBucketLocked(kind)
	keys := make([]string, 0, len(bucket))
	for key := range bucket {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, quickSightCloneMap(bucket[key]))
	}
	return out
}

func (s *quickSightStore) ensureTagsLocked(arn string) map[string]string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = "arn:aws:quicksight:us-east-1:123456789012:resource/default"
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"stackyard": "true"}
	}
	return s.tags[arn]
}

func (s *quickSightStore) ensurePermissionsLocked(arn string) []any {
	if perms := s.permissions[arn]; len(perms) > 0 {
		return quickSightCloneAnySlice(perms)
	}
	perms := []any{
		map[string]any{
			"Principal": "arn:aws:iam::123456789012:root",
			"Actions": []any{
				"quicksight:Describe",
				"quicksight:List",
			},
		},
	}
	s.permissions[arn] = perms
	return quickSightCloneAnySlice(perms)
}

func (s *quickSightStore) mergePermissions(payload map[string]any, arn string) []any {
	if payload == nil {
		return s.ensurePermissionsLocked(arn)
	}
	for key, value := range payload {
		if quickSightNormalizeKey(key) != quickSightNormalizeKey("Permissions") {
			continue
		}
		if list, ok := value.([]any); ok {
			s.permissions[arn] = quickSightCloneAnySlice(list)
			return quickSightCloneAnySlice(list)
		}
	}
	return s.ensurePermissionsLocked(arn)
}

func quickSightMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		out[key] = values[len(values)-1]
	}
	return out
}

func quickSightNormalizeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return ""
	}
	out := strings.Builder{}
	for _, ch := range key {
		if ch >= 'a' && ch <= 'z' {
			out.WriteRune(ch)
			continue
		}
		if ch >= '0' && ch <= '9' {
			out.WriteRune(ch)
		}
	}
	return out.String()
}

func quickSightString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	norm := quickSightNormalizeKey(key)
	for candidate, value := range payload {
		if quickSightNormalizeKey(candidate) != norm {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(value))
		if text != "" {
			return text
		}
	}
	return def
}

func quickSightKindFromContext(ctx map[string]any) string {
	switch {
	case quickSightString(ctx, "DashboardId", "") != "":
		return "dashboard"
	case quickSightString(ctx, "AnalysisId", "") != "":
		return "analysis"
	case quickSightString(ctx, "DataSetId", "") != "" || quickSightString(ctx, "DatasetId", "") != "":
		return "dataset"
	case quickSightString(ctx, "DataSourceId", "") != "":
		return "datasource"
	case quickSightString(ctx, "ThemeId", "") != "":
		return "theme"
	case quickSightString(ctx, "TemplateId", "") != "":
		return "template"
	case quickSightString(ctx, "TopicId", "") != "":
		return "topic"
	case quickSightString(ctx, "FolderId", "") != "":
		return "folder"
	case quickSightString(ctx, "VPCConnectionId", "") != "":
		return "vpcconnection"
	case quickSightString(ctx, "BrandId", "") != "":
		return "brand"
	case quickSightString(ctx, "FlowId", "") != "":
		return "flow"
	case quickSightString(ctx, "ActionConnectorId", "") != "":
		return "actionconnector"
	case quickSightString(ctx, "CustomPermissionsName", "") != "":
		return "custompermission"
	case quickSightString(ctx, "GroupName", "") != "":
		return "group"
	case quickSightString(ctx, "UserName", "") != "" || quickSightString(ctx, "PrincipalId", "") != "":
		return "user"
	case quickSightString(ctx, "Namespace", "") != "":
		return "namespace"
	case quickSightString(ctx, "IngestionId", "") != "":
		return "ingestion"
	case quickSightString(ctx, "ScheduleId", "") != "":
		return "schedule"
	case quickSightString(ctx, "RefreshId", "") != "":
		return "refresh"
	case quickSightString(ctx, "SnapshotJobId", "") != "":
		return "snapshotjob"
	case quickSightString(ctx, "AssetBundleExportJobId", "") != "":
		return "assetbundleexportjob"
	case quickSightString(ctx, "AssetBundleImportJobId", "") != "":
		return "assetbundleimportjob"
	case quickSightString(ctx, "AssignmentName", "") != "":
		return "assignment"
	case quickSightString(ctx, "AliasName", "") != "":
		return "alias"
	default:
		return "resource"
	}
}

func quickSightPrimaryID(ctx map[string]any, kind string) string {
	switch kind {
	case "dashboard":
		return quickSightString(ctx, "DashboardId", "dashboard-000001")
	case "analysis":
		return quickSightString(ctx, "AnalysisId", "analysis-000001")
	case "dataset":
		id := quickSightString(ctx, "DataSetId", "")
		if id == "" {
			id = quickSightString(ctx, "DatasetId", "dataset-000001")
		}
		return id
	case "datasource":
		return quickSightString(ctx, "DataSourceId", "datasource-000001")
	case "theme":
		return quickSightString(ctx, "ThemeId", "theme-000001")
	case "template":
		return quickSightString(ctx, "TemplateId", "template-000001")
	case "topic":
		return quickSightString(ctx, "TopicId", "topic-000001")
	case "folder":
		return quickSightString(ctx, "FolderId", "folder-000001")
	case "vpcconnection":
		return quickSightString(ctx, "VPCConnectionId", "vpc-connection-000001")
	case "brand":
		return quickSightString(ctx, "BrandId", "brand-000001")
	case "flow":
		return quickSightString(ctx, "FlowId", "flow-000001")
	case "actionconnector":
		return quickSightString(ctx, "ActionConnectorId", "action-connector-000001")
	case "custompermission":
		return quickSightString(ctx, "CustomPermissionsName", "custom-permissions-000001")
	case "group":
		return quickSightString(ctx, "GroupName", "stackyard-group")
	case "user":
		id := quickSightString(ctx, "UserName", "")
		if id == "" {
			id = quickSightString(ctx, "PrincipalId", "stackyard-user")
		}
		return id
	case "namespace":
		return quickSightString(ctx, "Namespace", "default")
	case "ingestion":
		return quickSightString(ctx, "IngestionId", "ingestion-000001")
	case "schedule":
		return quickSightString(ctx, "ScheduleId", "schedule-000001")
	case "refresh":
		return quickSightString(ctx, "RefreshId", "refresh-000001")
	case "snapshotjob":
		return quickSightString(ctx, "SnapshotJobId", "snapshot-job-000001")
	case "assetbundleexportjob":
		return quickSightString(ctx, "AssetBundleExportJobId", "asset-export-job-000001")
	case "assetbundleimportjob":
		return quickSightString(ctx, "AssetBundleImportJobId", "asset-import-job-000001")
	case "assignment":
		return quickSightString(ctx, "AssignmentName", "assignment-000001")
	case "alias":
		return quickSightString(ctx, "AliasName", "alias-000001")
	default:
		if id := quickSightString(ctx, "id", ""); id != "" {
			return id
		}
		return quickSightDefaultIDForKind(kind)
	}
}

func quickSightDefaultIDForKind(kind string) string {
	switch kind {
	case "namespace":
		return "default"
	case "group":
		return "stackyard-group"
	case "user":
		return "stackyard-user"
	default:
		return kind + "-000001"
	}
}

func quickSightARN(kind, id, accountID string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "resource"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = quickSightDefaultIDForKind(kind)
	}
	return "arn:aws:quicksight:us-east-1:" + accountID + ":" + kind + "/" + id
}

func quickSightListKey(action, kind string) string {
	switch action {
	case "ListDashboardVersions":
		return "DashboardVersionSummaryList"
	case "ListThemeVersions":
		return "ThemeVersionSummaryList"
	case "ListTemplateVersions":
		return "TemplateVersionSummaryList"
	}
	tail := strings.TrimPrefix(action, "List")
	if tail == "" {
		tail = "Resources"
	}
	if !strings.HasSuffix(strings.ToLower(tail), "s") {
		tail += "s"
	}
	return tail
}

func quickSightDescribeKey(action, kind string) string {
	switch action {
	case "DescribeAnalysis":
		return "Analysis"
	case "DescribeDashboard":
		return "Dashboard"
	case "DescribeDataSet":
		return "DataSet"
	case "DescribeDataSource":
		return "DataSource"
	case "DescribeTemplate":
		return "Template"
	case "DescribeTheme":
		return "Theme"
	case "DescribeTopic":
		return "Topic"
	case "DescribeVPCConnection":
		return "VPCConnection"
	case "DescribeUser":
		return "User"
	case "DescribeNamespace":
		return "Namespace"
	}
	for _, prefix := range []string{"Describe", "Get"} {
		if strings.HasPrefix(action, prefix) {
			tail := strings.TrimPrefix(action, prefix)
			if tail != "" {
				return tail
			}
		}
	}
	if kind == "" {
		return "Resource"
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func quickSightListResponse(key string, items []any, now time.Time) map[string]any {
	return map[string]any{
		key:         items,
		"NextToken": "",
		"Status":    200,
		"RequestId": "stackyard-request",
		"Timestamp": now.Format(time.RFC3339),
	}
}

func quickSightSuccess(_ time.Time) map[string]any {
	return map[string]any{
		"Status":    200,
		"RequestId": "stackyard-request",
	}
}

func quickSightTagsFromPayload(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	for key, value := range payload {
		if quickSightNormalizeKey(key) != quickSightNormalizeKey("Tags") {
			continue
		}
		switch typed := value.(type) {
		case map[string]any:
			for k, v := range typed {
				out[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprint(v))
			}
		case []any:
			for _, item := range typed {
				entry, ok := item.(map[string]any)
				if !ok {
					continue
				}
				tagKey := quickSightString(entry, "Key", "")
				if tagKey == "" {
					continue
				}
				out[tagKey] = quickSightString(entry, "Value", "")
			}
		}
	}
	return out
}

func quickSightTagKeys(payload map[string]any, query url.Values) []string {
	keys := []string{}
	isTagKeysField := func(key string) bool {
		norm := quickSightNormalizeKey(key)
		return norm == quickSightNormalizeKey("TagKeys") || norm == quickSightNormalizeKey("Keys")
	}
	if payload != nil {
		for key, value := range payload {
			if !isTagKeysField(key) {
				continue
			}
			switch typed := value.(type) {
			case []any:
				for _, item := range typed {
					tag := strings.TrimSpace(fmt.Sprint(item))
					if tag != "" {
						keys = append(keys, tag)
					}
				}
			case string:
				for _, item := range strings.Split(typed, ",") {
					tag := strings.TrimSpace(item)
					if tag != "" {
						keys = append(keys, tag)
					}
				}
			}
		}
	}
	if len(keys) == 0 {
		for queryKey, values := range query {
			if !isTagKeysField(queryKey) || len(values) == 0 {
				continue
			}
			for _, value := range values {
				for _, item := range strings.Split(value, ",") {
					tag := strings.TrimSpace(item)
					if tag != "" {
						keys = append(keys, tag)
					}
				}
			}
		}
	}
	return keys
}

func quickSightTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{
			"Key":   key,
			"Value": tags[key],
		})
	}
	return out
}

func quickSightCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = quickSightCloneMap(typed)
		case []any:
			out[key] = quickSightCloneAnySlice(typed)
		default:
			out[key] = value
		}
	}
	return out
}

func quickSightCloneAnySlice(in []any) []any {
	if in == nil {
		return []any{}
	}
	out := make([]any, 0, len(in))
	for _, item := range in {
		if nested, ok := item.(map[string]any); ok {
			out = append(out, quickSightCloneMap(nested))
			continue
		}
		if nested, ok := item.([]any); ok {
			out = append(out, quickSightCloneAnySlice(nested))
			continue
		}
		out = append(out, item)
	}
	return out
}
