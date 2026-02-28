package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	cloudWatchLogsDefaultLogGroupName  = "/stackyard/default"
	cloudWatchLogsDefaultLogStreamName = "stackyard-stream"
	cloudWatchLogsDefaultQueryID       = "stackyard-query-1"
)

type cloudWatchLogsStore struct {
	mu sync.Mutex

	nextID int64

	logGroups   map[string]map[string]any
	logStreams  map[string]map[string]map[string]any
	logEvents   map[string]map[string][]map[string]any
	tags        map[string]map[string]string
	queries     map[string]map[string]any
	exportTasks map[string]map[string]any
}

func newCloudWatchLogsStore() *cloudWatchLogsStore {
	s := &cloudWatchLogsStore{
		nextID:      2,
		logGroups:   map[string]map[string]any{},
		logStreams:  map[string]map[string]map[string]any{},
		logEvents:   map[string]map[string][]map[string]any{},
		tags:        map[string]map[string]string{},
		queries:     map[string]map[string]any{},
		exportTasks: map[string]map[string]any{},
	}

	group := s.ensureLogGroupLocked(cloudWatchLogsDefaultLogGroupName)
	stream := s.ensureLogStreamLocked(cloudWatchLogsDefaultLogGroupName, cloudWatchLogsDefaultLogStreamName)
	now := time.Now().UTC().UnixMilli()
	s.logEvents[cloudWatchLogsDefaultLogGroupName][cloudWatchLogsDefaultLogStreamName] = []map[string]any{{
		"timestamp": now,
		"message":   "stackyard seed event",
	}}
	stream["firstEventTimestamp"] = now
	stream["lastEventTimestamp"] = now
	stream["lastIngestionTime"] = now
	group["storedBytes"] = int64(64)
	s.tags[group["arn"].(string)] = map[string]string{"seed": "true"}

	return s
}

func (s *cloudWatchLogsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	nowMs := time.Now().UTC().UnixMilli()

	switch action {
	case "CreateLogGroup":
		name := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName", "name", "Name"}, cloudWatchLogsDefaultLogGroupName)
		group := s.ensureLogGroupLocked(name)
		s.applyPayloadLocked(group, payload)
		return map[string]any{}
	case "DeleteLogGroup":
		name := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName", "name", "Name"}, cloudWatchLogsDefaultLogGroupName)
		if group := s.logGroups[name]; group != nil {
			if arn, _ := group["arn"].(string); arn != "" {
				delete(s.tags, arn)
			}
		}
		delete(s.logGroups, name)
		delete(s.logStreams, name)
		delete(s.logEvents, name)
		return map[string]any{}
	case "DescribeLogGroups", "ListLogGroups", "ListAggregateLogGroupSummaries", "ListLogGroupsForQuery":
		return map[string]any{"logGroups": s.listLogGroupsLocked(), "nextToken": ""}
	case "CreateLogStream":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		streamName := cloudWatchLogsResolveString(payload, []string{"logStreamName", "LogStreamName", "name", "Name"}, cloudWatchLogsDefaultLogStreamName)
		s.ensureLogStreamLocked(groupName, streamName)
		return map[string]any{}
	case "DeleteLogStream":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		streamName := cloudWatchLogsResolveString(payload, []string{"logStreamName", "LogStreamName", "name", "Name"}, cloudWatchLogsDefaultLogStreamName)
		if streams := s.logStreams[groupName]; streams != nil {
			delete(streams, streamName)
		}
		if eventsByStream := s.logEvents[groupName]; eventsByStream != nil {
			delete(eventsByStream, streamName)
		}
		return map[string]any{}
	case "DescribeLogStreams":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		return map[string]any{"logStreams": s.listLogStreamsLocked(groupName), "nextToken": ""}
	case "PutLogEvents":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		streamName := cloudWatchLogsResolveString(payload, []string{"logStreamName", "LogStreamName"}, cloudWatchLogsDefaultLogStreamName)
		stream := s.ensureLogStreamLocked(groupName, streamName)
		events := cloudWatchLogsParseInputEvents(cloudWatchLogsAny(payload, "logEvents"))
		if len(events) == 0 {
			events = []map[string]any{{
				"timestamp": nowMs,
				"message":   "stackyard event",
			}}
		}
		if s.logEvents[groupName] == nil {
			s.logEvents[groupName] = map[string][]map[string]any{}
		}
		s.logEvents[groupName][streamName] = append(s.logEvents[groupName][streamName], events...)
		stream["lastIngestionTime"] = nowMs
		stream["lastEventTimestamp"] = nowMs
		stream["storedBytes"] = int64(len(s.logEvents[groupName][streamName])) * 64
		return map[string]any{
			"nextSequenceToken":     fmt.Sprintf("%d", len(s.logEvents[groupName][streamName])+1),
			"rejectedLogEventsInfo": map[string]any{},
		}
	case "GetLogEvents":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		streamName := cloudWatchLogsResolveString(payload, []string{"logStreamName", "LogStreamName"}, cloudWatchLogsDefaultLogStreamName)
		s.ensureLogStreamLocked(groupName, streamName)
		return map[string]any{
			"events":            cloudWatchLogsCloneAny(s.logEvents[groupName][streamName]),
			"nextForwardToken":  "f/00000000000000000000000000000000000000000000000000000000",
			"nextBackwardToken": "b/00000000000000000000000000000000000000000000000000000000",
		}
	case "FilterLogEvents":
		flat := make([]any, 0)
		for groupName, streamEvents := range s.logEvents {
			for streamName, events := range streamEvents {
				for _, evt := range events {
					item := cloudWatchLogsCloneMap(evt)
					item["logGroupName"] = groupName
					item["logStreamName"] = streamName
					flat = append(flat, item)
				}
			}
		}
		return map[string]any{"events": flat, "searchedLogStreams": []any{}, "nextToken": ""}
	case "StartQuery":
		id := s.nextIdentifierLocked("query")
		s.queries[id] = map[string]any{
			"queryId":      id,
			"status":       "Complete",
			"createTime":   nowMs,
			"queryString":  cloudWatchLogsResolveString(payload, []string{"queryString", "QueryString"}, "fields @message | limit 20"),
			"logGroupName": cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName),
		}
		return map[string]any{"queryId": id}
	case "DescribeQueries":
		queries := make([]any, 0, len(s.queries))
		keys := make([]string, 0, len(s.queries))
		for id := range s.queries {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			queries = append(queries, cloudWatchLogsCloneMap(s.queries[id]))
		}
		return map[string]any{"queries": queries, "nextToken": ""}
	case "GetQueryResults":
		id := cloudWatchLogsResolveString(payload, []string{"queryId", "QueryId"}, cloudWatchLogsDefaultQueryID)
		if s.queries[id] == nil {
			s.queries[id] = map[string]any{"queryId": id, "status": "Complete", "createTime": nowMs}
		}
		status := cloudWatchLogsResolveString(s.queries[id], []string{"status"}, "Complete")
		return map[string]any{"status": status, "results": []any{}, "statistics": map[string]any{}}
	case "StopQuery":
		id := cloudWatchLogsResolveString(payload, []string{"queryId", "QueryId"}, cloudWatchLogsDefaultQueryID)
		if q := s.queries[id]; q != nil {
			q["status"] = "Cancelled"
		}
		return map[string]any{"success": true}
	case "CreateExportTask":
		id := s.nextIdentifierLocked("export-task")
		s.exportTasks[id] = map[string]any{
			"taskId":   id,
			"taskName": cloudWatchLogsResolveString(payload, []string{"taskName", "TaskName"}, id),
			"status":   map[string]any{"code": "PENDING", "message": "created"},
		}
		return map[string]any{"taskId": id}
	case "CancelExportTask":
		id := cloudWatchLogsResolveString(payload, []string{"taskId", "TaskId"}, "")
		if task := s.exportTasks[id]; task != nil {
			task["status"] = map[string]any{"code": "CANCELLED", "message": "cancelled"}
		}
		return map[string]any{}
	case "DescribeExportTasks":
		items := make([]any, 0, len(s.exportTasks))
		keys := make([]string, 0, len(s.exportTasks))
		for id := range s.exportTasks {
			keys = append(keys, id)
		}
		sort.Strings(keys)
		for _, id := range keys {
			items = append(items, cloudWatchLogsCloneMap(s.exportTasks[id]))
		}
		return map[string]any{"exportTasks": items, "nextToken": ""}
	case "TagLogGroup":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		group := s.ensureLogGroupLocked(groupName)
		arn := cloudWatchLogsResolveString(group, []string{"arn", "Arn"}, cloudWatchLogsDefaultResourceARN())
		tags := s.ensureTagsLocked(arn)
		for key, value := range cloudWatchLogsMapString(cloudWatchLogsAny(payload, "tags")) {
			tags[key] = value
		}
		return map[string]any{}
	case "UntagLogGroup":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		group := s.ensureLogGroupLocked(groupName)
		arn := cloudWatchLogsResolveString(group, []string{"arn", "Arn"}, cloudWatchLogsDefaultResourceARN())
		tags := s.ensureTagsLocked(arn)
		for _, key := range cloudWatchLogsStringSlice(cloudWatchLogsAny(payload, "tags")) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsLogGroup":
		groupName := cloudWatchLogsResolveString(payload, []string{"logGroupName", "LogGroupName"}, cloudWatchLogsDefaultLogGroupName)
		group := s.ensureLogGroupLocked(groupName)
		arn := cloudWatchLogsResolveString(group, []string{"arn", "Arn"}, cloudWatchLogsDefaultResourceARN())
		return map[string]any{"tags": cloudWatchLogsCloneStringMap(s.tags[arn])}
	case "TagResource":
		resourceArn := cloudWatchLogsResolveString(payload, []string{"resourceArn", "ResourceArn", "resourceARN", "ResourceARN"}, cloudWatchLogsDefaultResourceARN())
		tags := s.ensureTagsLocked(resourceArn)
		for key, value := range cloudWatchLogsMapString(cloudWatchLogsAny(payload, "tags")) {
			tags[key] = value
		}
		return map[string]any{}
	case "UntagResource":
		resourceArn := cloudWatchLogsResolveString(payload, []string{"resourceArn", "ResourceArn", "resourceARN", "ResourceARN"}, cloudWatchLogsDefaultResourceARN())
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range cloudWatchLogsStringSlice(cloudWatchLogsAny(payload, "tagKeys")) {
			delete(tags, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceArn := cloudWatchLogsResolveString(payload, []string{"resourceArn", "ResourceArn", "resourceARN", "ResourceARN"}, cloudWatchLogsDefaultResourceARN())
		return map[string]any{"tags": cloudWatchLogsCloneStringMap(s.tags[resourceArn])}
	case "GetLogFields", "GetLogGroupFields":
		return map[string]any{"logGroupFields": []any{}}
	case "GetLogRecord":
		return map[string]any{"logRecord": map[string]any{}}
	case "GetLogObject":
		return map[string]any{"logEvents": []any{}}
	case "TestMetricFilter":
		return map[string]any{"matches": []any{}}
	case "StartLiveTail":
		return map[string]any{"sessionId": s.nextIdentifierLocked("tail"), "sessionMetadata": map[string]any{}}
	}

	switch {
	case strings.HasPrefix(action, "List"):
		return map[string]any{cloudWatchLogsListResponseKey(action): []any{}, "nextToken": ""}
	case strings.HasPrefix(action, "Describe"):
		return map[string]any{cloudWatchLogsListResponseKey(action): []any{}, "nextToken": ""}
	case strings.HasPrefix(action, "Start"):
		return map[string]any{"status": "STARTED", "requestId": s.nextIdentifierLocked("request")}
	case strings.HasPrefix(action, "Get"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *cloudWatchLogsStore) ensureLogGroupLocked(name string) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = cloudWatchLogsDefaultLogGroupName
	}
	if group := s.logGroups[name]; group != nil {
		return group
	}
	created := time.Now().UTC().UnixMilli()
	group := map[string]any{
		"logGroupName": name,
		"arn":          fmt.Sprintf("arn:aws:logs:us-east-1:123456789012:log-group:%s", strings.TrimPrefix(name, "/")),
		"creationTime": created,
		"storedBytes":  int64(0),
	}
	s.logGroups[name] = group
	if s.logStreams[name] == nil {
		s.logStreams[name] = map[string]map[string]any{}
	}
	if s.logEvents[name] == nil {
		s.logEvents[name] = map[string][]map[string]any{}
	}
	return group
}

func (s *cloudWatchLogsStore) ensureLogStreamLocked(groupName, streamName string) map[string]any {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = cloudWatchLogsDefaultLogGroupName
	}
	streamName = strings.TrimSpace(streamName)
	if streamName == "" {
		streamName = cloudWatchLogsDefaultLogStreamName
	}
	_ = s.ensureLogGroupLocked(groupName)
	if s.logStreams[groupName] == nil {
		s.logStreams[groupName] = map[string]map[string]any{}
	}
	if stream := s.logStreams[groupName][streamName]; stream != nil {
		return stream
	}
	created := time.Now().UTC().UnixMilli()
	stream := map[string]any{
		"logStreamName":       streamName,
		"creationTime":        created,
		"firstEventTimestamp": created,
		"lastEventTimestamp":  created,
		"lastIngestionTime":   created,
		"storedBytes":         int64(0),
		"arn":                 fmt.Sprintf("arn:aws:logs:us-east-1:123456789012:log-group:%s:log-stream:%s", strings.TrimPrefix(groupName, "/"), streamName),
	}
	s.logStreams[groupName][streamName] = stream
	if s.logEvents[groupName] == nil {
		s.logEvents[groupName] = map[string][]map[string]any{}
	}
	if s.logEvents[groupName][streamName] == nil {
		s.logEvents[groupName][streamName] = []map[string]any{}
	}
	return stream
}

func (s *cloudWatchLogsStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = cloudWatchLogsDefaultResourceARN()
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *cloudWatchLogsStore) listLogGroupsLocked() []any {
	names := make([]string, 0, len(s.logGroups))
	for name := range s.logGroups {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloudWatchLogsCloneMap(s.logGroups[name]))
	}
	return out
}

func (s *cloudWatchLogsStore) listLogStreamsLocked(groupName string) []any {
	_ = s.ensureLogGroupLocked(groupName)
	streams := s.logStreams[groupName]
	names := make([]string, 0, len(streams))
	for name := range streams {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloudWatchLogsCloneMap(streams[name]))
	}
	return out
}

func (s *cloudWatchLogsStore) applyPayloadLocked(target map[string]any, payload map[string]any) {
	for key, value := range payload {
		target[key] = cloudWatchLogsCloneAny(value)
	}
}

func (s *cloudWatchLogsStore) nextIdentifierLocked(prefix string) string {
	prefix = strings.Trim(strings.ToLower(prefix), "- ")
	if prefix == "" {
		prefix = "resource"
	}
	id := fmt.Sprintf("stackyard-%s-%06d", prefix, s.nextID)
	s.nextID++
	return id
}

func cloudWatchLogsAny(payload map[string]any, key string) any {
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return v
		}
	}
	return nil
}

func cloudWatchLogsResolveString(payload map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		value := cloudWatchLogsAny(payload, key)
		s, ok := value.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return fallback
}

func cloudWatchLogsStringSlice(value any) []string {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func cloudWatchLogsMapString(value any) map[string]string {
	entries, ok := value.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(entries))
	for k, v := range entries {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if s, ok := v.(string); ok {
			out[key] = s
		}
	}
	return out
}

func cloudWatchLogsParseInputEvents(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		clone := map[string]any{}
		if ts, ok := entry["timestamp"]; ok {
			clone["timestamp"] = ts
		}
		if msg, ok := entry["message"]; ok {
			clone["message"] = msg
		}
		if _, ok := clone["timestamp"]; !ok {
			clone["timestamp"] = time.Now().UTC().UnixMilli()
		}
		if _, ok := clone["message"]; !ok {
			clone["message"] = "stackyard event"
		}
		out = append(out, clone)
	}
	return out
}

func cloudWatchLogsCloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = cloudWatchLogsCloneAny(v)
	}
	return out
}

func cloudWatchLogsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloudWatchLogsCloneAny(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return value
	}
	return out
}

func cloudWatchLogsListResponseKey(action string) string {
	s := strings.TrimSpace(action)
	if strings.HasPrefix(s, "List") {
		s = strings.TrimPrefix(s, "List")
	}
	if strings.HasPrefix(s, "Describe") {
		s = strings.TrimPrefix(s, "Describe")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "items"
	}
	if strings.HasSuffix(s, "s") {
		return strings.ToLower(s[:1]) + s[1:]
	}
	return strings.ToLower(s[:1]) + s[1:] + "s"
}

func cloudWatchLogsDefaultResourceARN() string {
	return "arn:aws:logs:us-east-1:123456789012:log-group:stackyard/default"
}
