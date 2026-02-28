package server

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type resourceGroupsStore struct {
	mu sync.Mutex

	accountSettings map[string]any
	groups          map[string]map[string]any
	groupQueries    map[string]map[string]any
	groupConfigs    map[string]map[string]any
	groupByName     map[string]string
	groupResources  map[string]map[string]struct{}
	tags            map[string]map[string]string
	tagSyncTasks    map[string]map[string]any
	nextTaskID      int64
}

func newResourceGroupsStore() *resourceGroupsStore {
	groupARN := "arn:aws:resource-groups:us-east-1:123456789012:group/stackyard-group"
	resourceARN := "arn:aws:s3:::stackyard-seeded-bucket"
	taskARN := "arn:aws:resource-groups:us-east-1:123456789012:tag-sync-task/task-000001"
	now := time.Now().UTC().Format(time.RFC3339)

	return &resourceGroupsStore{
		accountSettings: map[string]any{
			"GroupLifecycleEventsDesiredStatus": "ACTIVE",
		},
		groups: map[string]map[string]any{
			groupARN: {
				"GroupArn":      groupARN,
				"Name":          "stackyard-group",
				"Description":   "Stackyard seeded resource group",
				"Criticality":   1,
				"Owner":         "123456789012",
				"DisplayName":   "stackyard-group",
				"LastUpdatedAt": now,
			},
		},
		groupQueries: map[string]map[string]any{
			groupARN: {
				"GroupName": "stackyard-group",
				"ResourceQuery": map[string]any{
					"Type":  "TAG_FILTERS_1_0",
					"Query": "{\"ResourceTypeFilters\":[\"AWS::AllSupported\"]}",
				},
			},
		},
		groupConfigs: map[string]map[string]any{
			groupARN: {
				"Group": "AWS::EC2::HostManagement",
			},
		},
		groupByName: map[string]string{
			"stackyard-group": groupARN,
		},
		groupResources: map[string]map[string]struct{}{
			groupARN: {
				resourceARN: {},
			},
		},
		tags: map[string]map[string]string{
			groupARN:    {"stackyard": "true"},
			resourceARN: {"stackyard": "true"},
		},
		tagSyncTasks: map[string]map[string]any{
			taskARN: {
				"TaskArn":   taskARN,
				"Group":     groupARN,
				"TagKey":    "stackyard",
				"TagValue":  "true",
				"RoleArn":   "arn:aws:iam::123456789012:role/stackyard-resource-groups",
				"Status":    "ACTIVE",
				"CreatedAt": now,
			},
		},
		nextTaskID: 2,
	}
}

func (s *resourceGroupsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncPayloadWithQuery(payload, query)

	switch action {
	case "CreateGroup":
		s.createGroup(payload)
		return map[string]any{}
	case "DeleteGroup":
		s.deleteGroup(payload)
		return map[string]any{}
	case "UpdateGroup":
		s.updateGroup(payload)
		return map[string]any{}
	case "UpdateGroupQuery":
		s.updateGroupQuery(payload)
		return map[string]any{}
	case "PutGroupConfiguration":
		s.putGroupConfiguration(payload)
		return map[string]any{}
	case "GroupResources":
		s.groupResourcesOp(payload)
		return map[string]any{}
	case "UngroupResources":
		s.ungroupResourcesOp(payload)
		return map[string]any{}
	case "StartTagSyncTask":
		s.startTagSyncTask(payload)
		return map[string]any{}
	case "CancelTagSyncTask":
		s.cancelTagSyncTask(payload)
		return map[string]any{}
	case "Tag":
		arn := rgResourceARN(pathParams, payload, s.defaultGroupARN())
		s.mergeTagsForARN(arn, payload["Tags"])
		return map[string]any{}
	case "Untag":
		arn := rgResourceARN(pathParams, payload, s.defaultGroupARN())
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		for _, key := range rgTagKeys(payload) {
			delete(s.tags[arn], key)
		}
		return map[string]any{}
	case "GetAccountSettings", "UpdateAccountSettings":
		return map[string]any{"AccountSettings": rgCloneAnyMap(s.accountSettings)}
	case "GetGroupConfiguration":
		return map[string]any{"GroupConfiguration": rgCloneAnyMap(s.groupConfigs[s.resolveGroupARN(payload)])}
	case "GetGroupQuery":
		return map[string]any{"GroupQuery": rgCloneAnyMap(s.groupQueries[s.resolveGroupARN(payload)])}
	case "ListGroupResources":
		arn := s.resolveGroupARN(payload)
		return map[string]any{
			"ResourceIdentifiers": s.listResourceIdentifiers(arn),
			"Resources":           []any{},
			"QueryErrors":         []any{},
			"NextToken":           "",
		}
	case "GetTags":
		arn := rgResourceARN(pathParams, payload, s.defaultGroupARN())
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		return map[string]any{
			"Arn":  arn,
			"Tags": rgCloneTags(s.tags[arn]),
		}
	case "ListTagSyncTasks":
		return map[string]any{"TagSyncTasks": s.listTagSyncTasks(), "NextToken": ""}
	case "GetTagSyncTask":
		return map[string]any{"TagSyncTask": rgCloneAnyMap(s.getTask(payload))}
	case "SearchResources":
		return map[string]any{
			"ResourceIdentifiers": s.searchResourceIdentifiers(),
			"QueryErrors":         []any{},
			"NextToken":           "",
		}
	case "ListGroupingStatuses":
		return map[string]any{
			"GroupingStatuses": s.listGroupingStatuses(payload),
			"NextToken":        "",
		}
	}

	return map[string]any{}
}

func (s *resourceGroupsStore) createGroup(payload map[string]any) {
	name := rgStringAny(payload, "Name", "")
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("stackyard-group-%d", len(s.groups)+1)
	}
	groupARN := rgStringAny(payload, "GroupArn", "")
	if strings.TrimSpace(groupARN) == "" {
		groupARN = fmt.Sprintf("arn:aws:resource-groups:us-east-1:123456789012:group/%s", name)
	}

	s.groups[groupARN] = map[string]any{
		"GroupArn":    groupARN,
		"Name":        name,
		"Description": rgStringAny(payload, "Description", ""),
	}
	s.groupByName[strings.TrimSpace(name)] = groupARN
	if s.groupResources[groupARN] == nil {
		s.groupResources[groupARN] = map[string]struct{}{}
	}
	if rawQuery, ok := payload["ResourceQuery"].(map[string]any); ok {
		s.groupQueries[groupARN] = rgCloneAnyMap(rawQuery)
	}
	if rawConfig, ok := payload["Configuration"].(map[string]any); ok {
		s.groupConfigs[groupARN] = rgCloneAnyMap(rawConfig)
	}
	if s.tags[groupARN] == nil {
		s.tags[groupARN] = map[string]string{}
	}
	s.mergeTagsForARN(groupARN, payload["Tags"])
}

func (s *resourceGroupsStore) deleteGroup(payload map[string]any) {
	arn := s.resolveGroupARN(payload)
	delete(s.groups, arn)
	delete(s.groupQueries, arn)
	delete(s.groupConfigs, arn)
	delete(s.groupResources, arn)
	delete(s.tags, arn)
	for name, candidateARN := range s.groupByName {
		if candidateARN == arn {
			delete(s.groupByName, name)
		}
	}
}

func (s *resourceGroupsStore) updateGroup(payload map[string]any) {
	arn := s.resolveGroupARN(payload)
	group := s.groups[arn]
	if group == nil {
		group = map[string]any{
			"GroupArn": arn,
			"Name":     rgGroupNameFromARN(arn),
		}
		s.groups[arn] = group
	}
	if value := rgStringAny(payload, "Description", ""); value != "" {
		group["Description"] = value
	}
	if value := rgStringAny(payload, "Name", ""); value != "" {
		group["Name"] = value
		s.groupByName[value] = arn
	}
}

func (s *resourceGroupsStore) updateGroupQuery(payload map[string]any) {
	arn := s.resolveGroupARN(payload)
	if s.groupQueries[arn] == nil {
		s.groupQueries[arn] = map[string]any{"GroupName": rgGroupNameFromARN(arn)}
	}
	if raw, ok := payload["ResourceQuery"].(map[string]any); ok {
		s.groupQueries[arn]["ResourceQuery"] = rgCloneAnyMap(raw)
	}
}

func (s *resourceGroupsStore) putGroupConfiguration(payload map[string]any) {
	arn := s.resolveGroupARN(payload)
	if raw, ok := payload["Configuration"].(map[string]any); ok {
		s.groupConfigs[arn] = rgCloneAnyMap(raw)
		return
	}
	if s.groupConfigs[arn] == nil {
		s.groupConfigs[arn] = map[string]any{}
	}
}

func (s *resourceGroupsStore) groupResourcesOp(payload map[string]any) {
	arn := s.resolveGroupARN(payload)
	if s.groupResources[arn] == nil {
		s.groupResources[arn] = map[string]struct{}{}
	}
	for _, resourceARN := range rgStringSlice(payload, "ResourceArns") {
		s.groupResources[arn][resourceARN] = struct{}{}
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
	}
}

func (s *resourceGroupsStore) ungroupResourcesOp(payload map[string]any) {
	arn := s.resolveGroupARN(payload)
	resources := s.groupResources[arn]
	if resources == nil {
		return
	}
	for _, resourceARN := range rgStringSlice(payload, "ResourceArns") {
		delete(resources, resourceARN)
	}
}

func (s *resourceGroupsStore) startTagSyncTask(payload map[string]any) {
	groupARN := s.resolveGroupARN(payload)
	taskARN := fmt.Sprintf("arn:aws:resource-groups:us-east-1:123456789012:tag-sync-task/task-%06d", s.nextTaskID)
	s.nextTaskID++
	s.tagSyncTasks[taskARN] = map[string]any{
		"TaskArn":   taskARN,
		"Group":     groupARN,
		"TagKey":    rgStringAny(payload, "TagKey", "stackyard"),
		"TagValue":  rgStringAny(payload, "TagValue", "true"),
		"RoleArn":   rgStringAny(payload, "RoleArn", "arn:aws:iam::123456789012:role/stackyard-resource-groups"),
		"Status":    "ACTIVE",
		"CreatedAt": time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *resourceGroupsStore) cancelTagSyncTask(payload map[string]any) {
	task := s.getTask(payload)
	if task == nil {
		return
	}
	task["Status"] = "CANCELLED"
	task["UpdatedAt"] = time.Now().UTC().Format(time.RFC3339)
}

func (s *resourceGroupsStore) getTask(payload map[string]any) map[string]any {
	taskARN := rgStringAny(payload, "TaskArn", "")
	if strings.TrimSpace(taskARN) != "" {
		if task := s.tagSyncTasks[taskARN]; task != nil {
			return task
		}
	}
	for _, task := range s.tagSyncTasks {
		return task
	}
	return map[string]any{}
}

func (s *resourceGroupsStore) listTagSyncTasks() []any {
	out := make([]any, 0, len(s.tagSyncTasks))
	for _, task := range s.tagSyncTasks {
		out = append(out, rgCloneAnyMap(task))
	}
	return out
}

func (s *resourceGroupsStore) listGroupingStatuses(payload map[string]any) []any {
	groupARN := s.resolveGroupARN(payload)
	out := make([]any, 0, len(s.groupResources[groupARN]))
	for resourceARN := range s.groupResources[groupARN] {
		out = append(out, map[string]any{
			"Group":       groupARN,
			"ResourceArn": resourceARN,
			"Status":      "SUCCESS",
		})
	}
	return out
}

func (s *resourceGroupsStore) listResourceIdentifiers(groupARN string) []any {
	out := []any{}
	for resourceARN := range s.groupResources[groupARN] {
		out = append(out, map[string]any{
			"ResourceArn":  resourceARN,
			"ResourceType": "AWS::S3::Bucket",
		})
	}
	return out
}

func (s *resourceGroupsStore) searchResourceIdentifiers() []any {
	out := []any{}
	seen := map[string]struct{}{}
	for _, resources := range s.groupResources {
		for resourceARN := range resources {
			if _, ok := seen[resourceARN]; ok {
				continue
			}
			seen[resourceARN] = struct{}{}
			out = append(out, map[string]any{
				"ResourceArn":  resourceARN,
				"ResourceType": "AWS::S3::Bucket",
			})
		}
	}
	if len(out) == 0 {
		out = append(out, map[string]any{
			"ResourceArn":  "arn:aws:s3:::stackyard-seeded-bucket",
			"ResourceType": "AWS::S3::Bucket",
		})
	}
	return out
}

func (s *resourceGroupsStore) resolveGroupARN(payload map[string]any) string {
	if value := strings.TrimSpace(rgStringAny(payload, "Group", "")); value != "" {
		if strings.HasPrefix(value, "arn:") {
			return value
		}
		if arn := strings.TrimSpace(s.groupByName[value]); arn != "" {
			return arn
		}
		return fmt.Sprintf("arn:aws:resource-groups:us-east-1:123456789012:group/%s", value)
	}
	if value := strings.TrimSpace(rgStringAny(payload, "GroupArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(rgStringAny(payload, "GroupName", "")); value != "" {
		if arn := strings.TrimSpace(s.groupByName[value]); arn != "" {
			return arn
		}
		return fmt.Sprintf("arn:aws:resource-groups:us-east-1:123456789012:group/%s", value)
	}
	return s.defaultGroupARN()
}

func (s *resourceGroupsStore) defaultGroupARN() string {
	for arn := range s.groups {
		return arn
	}
	arn := "arn:aws:resource-groups:us-east-1:123456789012:group/stackyard-group"
	s.groups[arn] = map[string]any{"GroupArn": arn, "Name": "stackyard-group"}
	if s.groupResources[arn] == nil {
		s.groupResources[arn] = map[string]struct{}{}
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"stackyard": "true"}
	}
	return arn
}

func (s *resourceGroupsStore) mergeTagsForARN(arn string, raw any) {
	if strings.TrimSpace(arn) == "" {
		return
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{}
	}
	switch tags := raw.(type) {
	case map[string]any:
		for key, value := range tags {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			s.tags[arn][key] = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
	case map[string]string:
		for key, value := range tags {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			s.tags[arn][key] = strings.TrimSpace(value)
		}
	}
}

func (s *resourceGroupsStore) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	if payload == nil {
		return
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if _, exists := payload[key]; exists {
			continue
		}
		payload[key] = values[len(values)-1]
	}
}

func rgStringAny(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case string:
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if text == "" {
		return fallback
	}
	return text
}

func rgStringSlice(payload map[string]any, key string) []string {
	out := []string{}
	if payload == nil {
		return out
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return out
	}
	switch values := raw.(type) {
	case []any:
		for _, value := range values {
			text := strings.TrimSpace(fmt.Sprintf("%v", value))
			if text != "" {
				out = append(out, text)
			}
		}
	case []string:
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				out = append(out, value)
			}
		}
	}
	return out
}

func rgTagKeys(payload map[string]any) []string {
	keys := rgStringSlice(payload, "Keys")
	if len(keys) > 0 {
		return keys
	}
	keys = rgStringSlice(payload, "TagKeys")
	if len(keys) > 0 {
		return keys
	}
	if value := strings.TrimSpace(rgStringAny(payload, "Key", "")); value != "" {
		return []string{value}
	}
	return []string{}
}

func rgResourceARN(pathParams map[string]string, payload map[string]any, fallback string) string {
	if value := strings.TrimSpace(pathParams["Arn"]); value != "" {
		return value
	}
	if value := strings.TrimSpace(pathParams["arn"]); value != "" {
		return value
	}
	if value := strings.TrimSpace(rgStringAny(payload, "Arn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(rgStringAny(payload, "ResourceArn", "")); value != "" {
		return value
	}
	return fallback
}

func rgGroupNameFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return "stackyard-group"
	}
	parts := strings.Split(arn, "/")
	if len(parts) == 0 {
		return "stackyard-group"
	}
	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return "stackyard-group"
	}
	return name
}

func rgCloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func rgCloneTags(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
