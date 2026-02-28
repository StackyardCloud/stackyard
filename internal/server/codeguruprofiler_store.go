package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type codeGuruProfilerStore struct {
	mu     sync.Mutex
	nextID int64
	groups map[string]*codeGuruProfilerProfilingGroup
}

type codeGuruProfilerProfilingGroup struct {
	Name                 string
	Arn                  string
	ComputePlatform      string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	AgentConfiguration   map[string]any
	NotificationChannels []map[string]any
	PermissionPrincipals map[string][]string
	PolicyRevision       int64
	Tags                 map[string]string
	ProfileTimes         []time.Time
	FindingsReports      []map[string]any
}

func newCodeGuruProfilerStore() *codeGuruProfilerStore {
	now := time.Now().UTC()
	groupName := "stackyard-profiling-group"
	groupArn := codeGuruProfilerProfilingGroupARN(groupName)

	return &codeGuruProfilerStore{
		nextID: 2,
		groups: map[string]*codeGuruProfilerProfilingGroup{
			groupName: {
				Name:            groupName,
				Arn:             groupArn,
				ComputePlatform: "Default",
				CreatedAt:       now,
				UpdatedAt:       now,
				AgentConfiguration: map[string]any{
					"agentParameters": map[string]any{},
					"periodInSeconds": int64(60),
					"shouldProfile":   true,
				},
				NotificationChannels: []map[string]any{
					{
						"id":              "channel-000001",
						"uri":             "sns:arn:aws:sns:us-east-1:123456789012:stackyard-codeguru-profiler",
						"eventPublishers": []any{"AnomalyDetection"},
					},
				},
				PermissionPrincipals: map[string][]string{
					"agentPermissions": {"arn:aws:iam::123456789012:role/stackyard-codeguru-profiler"},
				},
				PolicyRevision: 1,
				Tags: map[string]string{
					"seed": "true",
				},
				ProfileTimes: []time.Time{
					now.Add(-1 * time.Hour),
					now.Add(-30 * time.Minute),
				},
				FindingsReports: []map[string]any{
					{
						"id":                    "report-000001",
						"profileStartTime":      now.Add(-24 * time.Hour),
						"profileEndTime":        now.Add(-23 * time.Hour),
						"profilingGroupName":    groupName,
						"totalNumberOfFindings": int64(1),
					},
				},
			},
		},
	}
}

func (s *codeGuruProfilerStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "AddNotificationChannels":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		incoming := codeGuruProfilerChannelSlice(codeGuruProfilerPayloadValue(payload, "channels"))
		if len(incoming) == 0 {
			incoming = []map[string]any{
				{
					"id":              fmt.Sprintf("channel-%06d", s.nextLocked()),
					"uri":             "sns:arn:aws:sns:us-east-1:123456789012:stackyard-codeguru-profiler",
					"eventPublishers": []any{"AnomalyDetection"},
				},
			}
		}
		for _, channel := range incoming {
			group.NotificationChannels = append(group.NotificationChannels, codeGuruProfilerNormalizeChannel(channel))
		}
		group.UpdatedAt = now
		return map[string]any{
			"notificationConfiguration": map[string]any{
				"channels": codeGuruProfilerCloneChannels(group.NotificationChannels),
			},
		}

	case "BatchGetFrameMetricData":
		start := now.Add(-1 * time.Hour)
		end := now
		return map[string]any{
			"startTime":           start,
			"endTime":             end,
			"endTimes":            []any{},
			"frameMetricData":     []any{},
			"resolution":          "PT1H",
			"unprocessedEndTimes": map[string]any{},
		}

	case "ConfigureAgent":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		period := codeGuruProfilerIntValue(codeGuruProfilerPayloadValue(payload, "fleetInstanceId"), 60)
		if period <= 0 {
			period = codeGuruProfilerIntValue(codeGuruProfilerPayloadValue(payload, "periodInSeconds"), 60)
		}
		shouldProfile := codeGuruProfilerBoolValue(codeGuruProfilerPayloadValue(payload, "shouldProfile"), true)
		group.AgentConfiguration = map[string]any{
			"agentParameters": map[string]any{},
			"periodInSeconds": period,
			"shouldProfile":   shouldProfile,
		}
		group.UpdatedAt = now
		return map[string]any{"configuration": codeGuruProfilerCloneMapAny(group.AgentConfiguration)}

	case "CreateProfilingGroup":
		groupName := codeGuruProfilerDefaultString(payload, "profilingGroupName", fmt.Sprintf("stackyard-profiling-group-%06d", s.nextLocked()))
		group := s.ensureGroupLocked(groupName)
		group.ComputePlatform = codeGuruProfilerDefaultString(payload, "computePlatform", group.ComputePlatform)
		group.UpdatedAt = now
		for key, value := range codeGuruProfilerStringMap(codeGuruProfilerPayloadValue(payload, "tags")) {
			group.Tags[key] = value
		}
		return map[string]any{"profilingGroup": s.profilingGroupPayload(group)}

	case "DeleteProfilingGroup":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		delete(s.groups, groupName)
		return map[string]any{}

	case "DescribeProfilingGroup":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		return map[string]any{"profilingGroup": s.profilingGroupPayload(group)}

	case "GetFindingsReportAccountSummary":
		summaries := make([]any, 0, 4)
		for _, group := range s.sortedGroupsLocked() {
			for _, report := range group.FindingsReports {
				summaries = append(summaries, codeGuruProfilerCloneMapAny(report))
			}
		}
		return map[string]any{"reportSummaries": summaries}

	case "GetNotificationConfiguration":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		return map[string]any{
			"notificationConfiguration": map[string]any{
				"channels": codeGuruProfilerCloneChannels(group.NotificationChannels),
			},
		}

	case "GetPolicy":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		return map[string]any{
			"policy":     codeGuruProfilerRenderPolicy(group.PermissionPrincipals),
			"revisionId": fmt.Sprintf("r-%06d", group.PolicyRevision),
		}

	case "GetProfile":
		return map[string]any{
			"contentType":     "application/x-amzn-ion",
			"contentEncoding": "gzip",
			"profile":         "c3RhY2t5YXJk",
		}

	case "GetRecommendations":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		start := now.Add(-2 * time.Hour)
		end := now.Add(-1 * time.Hour)
		return map[string]any{
			"profilingGroupName": group.Name,
			"profileStartTime":   start,
			"profileEndTime":     end,
			"anomalies":          []any{},
			"recommendations":    []any{},
		}

	case "ListFindingsReports":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		out := make([]any, 0, len(group.FindingsReports))
		for _, report := range group.FindingsReports {
			out = append(out, codeGuruProfilerCloneMapAny(report))
		}
		return map[string]any{"findingsReportSummaries": out, "nextToken": ""}

	case "ListProfileTimes":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		items := make([]any, 0, len(group.ProfileTimes))
		for _, ts := range group.ProfileTimes {
			items = append(items, map[string]any{"start": ts})
		}
		if len(items) == 0 {
			items = append(items, map[string]any{"start": now})
		}
		return map[string]any{"profileTimes": items, "nextToken": ""}

	case "ListProfilingGroups":
		groups := s.sortedGroupsLocked()
		names := make([]any, 0, len(groups))
		descriptions := make([]any, 0, len(groups))
		for _, group := range groups {
			names = append(names, group.Name)
			descriptions = append(descriptions, s.profilingGroupPayload(group))
		}
		return map[string]any{
			"profilingGroupNames": names,
			"profilingGroups":     descriptions,
			"nextToken":           "",
		}

	case "ListTagsForResource":
		arn := codeGuruProfilerResolveResourceARN(payload, pathParams, s.defaultGroupARNLocked())
		group := s.findGroupByARNLocked(arn)
		if group == nil {
			return map[string]any{"tags": map[string]any{}}
		}
		return map[string]any{"tags": codeGuruProfilerCloneStringMap(group.Tags)}

	case "PostAgentProfile":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		group.UpdatedAt = now
		group.ProfileTimes = append(group.ProfileTimes, now)
		return map[string]any{}

	case "PutPermission":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		actionGroup := codeGuruProfilerResolveActionGroup(payload, pathParams)
		principals := codeGuruProfilerStringSlice(codeGuruProfilerPayloadValue(payload, "principals"))
		if len(principals) == 0 {
			principals = []string{"arn:aws:iam::123456789012:role/stackyard-codeguru-profiler"}
		}
		group.PermissionPrincipals[actionGroup] = principals
		group.PolicyRevision++
		group.UpdatedAt = now
		return map[string]any{
			"policy":     codeGuruProfilerRenderPolicy(group.PermissionPrincipals),
			"revisionId": fmt.Sprintf("r-%06d", group.PolicyRevision),
		}

	case "RemoveNotificationChannel":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		channelID := strings.TrimSpace(pathParams["channelId"])
		if channelID == "" {
			channelID = codeGuruProfilerDefaultString(payload, "channelId", "")
		}
		if channelID != "" {
			filtered := make([]map[string]any, 0, len(group.NotificationChannels))
			for _, channel := range group.NotificationChannels {
				id := strings.TrimSpace(codeGuruProfilerAsString(channel["id"]))
				if id == channelID {
					continue
				}
				filtered = append(filtered, codeGuruProfilerNormalizeChannel(channel))
			}
			group.NotificationChannels = filtered
		}
		group.UpdatedAt = now
		return map[string]any{
			"notificationConfiguration": map[string]any{
				"channels": codeGuruProfilerCloneChannels(group.NotificationChannels),
			},
		}

	case "RemovePermission":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		actionGroup := codeGuruProfilerResolveActionGroup(payload, pathParams)
		delete(group.PermissionPrincipals, actionGroup)
		group.PolicyRevision++
		group.UpdatedAt = now
		return map[string]any{
			"policy":     codeGuruProfilerRenderPolicy(group.PermissionPrincipals),
			"revisionId": fmt.Sprintf("r-%06d", group.PolicyRevision),
		}

	case "SubmitFeedback":
		return map[string]any{}

	case "TagResource":
		arn := codeGuruProfilerResolveResourceARN(payload, pathParams, s.defaultGroupARNLocked())
		group := s.findGroupByARNLocked(arn)
		if group == nil {
			group = s.ensureGroupLocked(codeGuruProfilerSuffixFromARN(arn))
		}
		for key, value := range codeGuruProfilerStringMap(codeGuruProfilerPayloadValue(payload, "tags")) {
			group.Tags[key] = value
		}
		group.UpdatedAt = now
		return map[string]any{}

	case "UntagResource":
		arn := codeGuruProfilerResolveResourceARN(payload, pathParams, s.defaultGroupARNLocked())
		group := s.findGroupByARNLocked(arn)
		if group == nil {
			return map[string]any{}
		}
		tagKeys := codeGuruProfilerStringSlice(codeGuruProfilerPayloadValue(payload, "tagKeys"))
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				tagKeys = append(tagKeys, key)
			}
		}
		for _, key := range tagKeys {
			delete(group.Tags, key)
		}
		group.UpdatedAt = now
		return map[string]any{}

	case "UpdateProfilingGroup":
		groupName := codeGuruProfilerResolveGroupName(payload, pathParams)
		group := s.ensureGroupLocked(groupName)
		group.ComputePlatform = codeGuruProfilerDefaultString(payload, "computePlatform", group.ComputePlatform)
		group.UpdatedAt = now
		return map[string]any{"profilingGroup": s.profilingGroupPayload(group)}
	}

	return map[string]any{}
}

func (s *codeGuruProfilerStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *codeGuruProfilerStore) ensureGroupLocked(name string) *codeGuruProfilerProfilingGroup {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-profiling-group"
	}
	if existing := s.groups[name]; existing != nil {
		return existing
	}

	now := time.Now().UTC()
	group := &codeGuruProfilerProfilingGroup{
		Name:            name,
		Arn:             codeGuruProfilerProfilingGroupARN(name),
		ComputePlatform: "Default",
		CreatedAt:       now,
		UpdatedAt:       now,
		AgentConfiguration: map[string]any{
			"agentParameters": map[string]any{},
			"periodInSeconds": int64(60),
			"shouldProfile":   true,
		},
		NotificationChannels: []map[string]any{},
		PermissionPrincipals: map[string][]string{},
		PolicyRevision:       1,
		Tags:                 map[string]string{},
		ProfileTimes:         []time.Time{now},
		FindingsReports:      []map[string]any{},
	}
	s.groups[name] = group
	return group
}

func (s *codeGuruProfilerStore) sortedGroupsLocked() []*codeGuruProfilerProfilingGroup {
	names := make([]string, 0, len(s.groups))
	for name := range s.groups {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*codeGuruProfilerProfilingGroup, 0, len(names))
	for _, name := range names {
		out = append(out, s.groups[name])
	}
	return out
}

func (s *codeGuruProfilerStore) defaultGroupARNLocked() string {
	for _, group := range s.sortedGroupsLocked() {
		if group != nil {
			return group.Arn
		}
	}
	return codeGuruProfilerProfilingGroupARN("stackyard-profiling-group")
}

func (s *codeGuruProfilerStore) findGroupByARNLocked(arn string) *codeGuruProfilerProfilingGroup {
	arn = strings.TrimSpace(arn)
	for _, group := range s.groups {
		if group != nil && group.Arn == arn {
			return group
		}
	}
	return nil
}

func (s *codeGuruProfilerStore) profilingGroupPayload(group *codeGuruProfilerProfilingGroup) map[string]any {
	if group == nil {
		return map[string]any{}
	}
	return map[string]any{
		"agentOrchestrationConfig": map[string]any{},
		"arn":                      group.Arn,
		"computePlatform":          group.ComputePlatform,
		"createdAt":                group.CreatedAt,
		"name":                     group.Name,
		"profilingStatus":          map[string]any{},
		"tags":                     codeGuruProfilerCloneStringMap(group.Tags),
		"updatedAt":                group.UpdatedAt,
	}
}

func codeGuruProfilerProfilingGroupARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-profiling-group"
	}
	return fmt.Sprintf("arn:aws:codeguru-profiler:us-east-1:123456789012:profilingGroup:%s", name)
}

func codeGuruProfilerSuffixFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	if idx := strings.LastIndex(arn, ":"); idx >= 0 && idx+1 < len(arn) {
		return strings.TrimSpace(arn[idx+1:])
	}
	return ""
}

func codeGuruProfilerResolveGroupName(payload map[string]any, pathParams map[string]string) string {
	if value := strings.TrimSpace(pathParams["profilingGroupName"]); value != "" {
		return value
	}
	return codeGuruProfilerDefaultString(payload, "profilingGroupName", "stackyard-profiling-group")
}

func codeGuruProfilerResolveResourceARN(payload map[string]any, pathParams map[string]string, def string) string {
	if value := strings.TrimSpace(pathParams["resourceArn"]); value != "" {
		return value
	}
	return codeGuruProfilerDefaultString(payload, "resourceArn", def)
}

func codeGuruProfilerResolveActionGroup(payload map[string]any, pathParams map[string]string) string {
	if value := strings.TrimSpace(pathParams["actionGroup"]); value != "" {
		return value
	}
	return codeGuruProfilerDefaultString(payload, "actionGroup", "agentPermissions")
}

func codeGuruProfilerRenderPolicy(permissions map[string][]string) string {
	statement := make([]map[string]any, 0, len(permissions))
	actionGroups := make([]string, 0, len(permissions))
	for actionGroup := range permissions {
		actionGroups = append(actionGroups, actionGroup)
	}
	sort.Strings(actionGroups)
	for _, actionGroup := range actionGroups {
		principals := append([]string{}, permissions[actionGroup]...)
		sort.Strings(principals)
		statement = append(statement, map[string]any{
			"Sid":       actionGroup,
			"Effect":    "Allow",
			"Principal": map[string]any{"AWS": principals},
			"Action":    actionGroup,
			"Resource":  "*",
		})
	}
	doc := map[string]any{
		"Version":   "2012-10-17",
		"Statement": statement,
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		return `{"Version":"2012-10-17","Statement":[]}`
	}
	return string(raw)
}

func codeGuruProfilerPayloadValue(payload map[string]any, key string) any {
	for existingKey, value := range payload {
		if strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			return value
		}
	}
	return nil
}

func codeGuruProfilerDefaultString(payload map[string]any, key, def string) string {
	value := strings.TrimSpace(codeGuruProfilerAsString(codeGuruProfilerPayloadValue(payload, key)))
	if value == "" {
		return def
	}
	return value
}

func codeGuruProfilerAsString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func codeGuruProfilerBoolValue(value any, def bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return def
}

func codeGuruProfilerIntValue(value any, def int64) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int8:
		return int64(typed)
	case int16:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float32:
		return int64(typed)
	case float64:
		return int64(typed)
	case json.Number:
		if v, err := typed.Int64(); err == nil {
			return v
		}
	case string:
		if parsed, err := json.Number(strings.TrimSpace(typed)).Int64(); err == nil {
			return parsed
		}
	}
	return def
}

func codeGuruProfilerStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(codeGuruProfilerAsString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(codeGuruProfilerAsString(value))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func codeGuruProfilerStringMap(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, item := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(item)
		}
	case map[string]any:
		for key, item := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(codeGuruProfilerAsString(item))
		}
	}
	return out
}

func codeGuruProfilerChannelSlice(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, codeGuruProfilerNormalizeChannel(item))
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if m, ok := item.(map[string]any); ok {
				out = append(out, codeGuruProfilerNormalizeChannel(m))
			}
		}
		return out
	default:
		return nil
	}
}

func codeGuruProfilerNormalizeChannel(channel map[string]any) map[string]any {
	if channel == nil {
		channel = map[string]any{}
	}
	id := strings.TrimSpace(codeGuruProfilerAsString(channel["id"]))
	uri := strings.TrimSpace(codeGuruProfilerAsString(channel["uri"]))
	if id == "" {
		id = "channel-000000"
	}
	if uri == "" {
		uri = "sns:arn:aws:sns:us-east-1:123456789012:stackyard-codeguru-profiler"
	}
	publishers := codeGuruProfilerStringSlice(channel["eventPublishers"])
	if len(publishers) == 0 {
		publishers = []string{"AnomalyDetection"}
	}
	publisherOut := make([]any, 0, len(publishers))
	for _, item := range publishers {
		publisherOut = append(publisherOut, item)
	}
	return map[string]any{
		"id":              id,
		"uri":             uri,
		"eventPublishers": publisherOut,
	}
}

func codeGuruProfilerCloneChannels(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, channel := range in {
		out = append(out, codeGuruProfilerNormalizeChannel(channel))
	}
	return out
}

func codeGuruProfilerCloneMapAny(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func codeGuruProfilerCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
