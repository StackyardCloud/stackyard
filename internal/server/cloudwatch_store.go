package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type cloudWatchStore struct {
	mu             sync.Mutex
	alarms         map[string]struct{}
	dashboards     map[string]string
	metricStreams  map[string]string
	insightRules   map[string]struct{}
	alarmMuteRules map[string]struct{}
	tags           map[string]map[string]string
}

func newCloudWatchStore() *cloudWatchStore {
	return &cloudWatchStore{
		alarms: map[string]struct{}{
			"stackyard-alarm": {},
		},
		dashboards: map[string]string{
			"stackyard-dashboard": "{}",
		},
		metricStreams: map[string]string{
			"stackyard-stream": "arn:aws:cloudwatch:us-east-1:123456789012:metric-stream/stackyard-stream",
		},
		insightRules: map[string]struct{}{
			"stackyard-insight-rule": {},
		},
		alarmMuteRules: map[string]struct{}{
			"stackyard-mute-rule": {},
		},
		tags: map[string]map[string]string{},
	}
}

func (s *cloudWatchStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "PutMetricAlarm", "PutCompositeAlarm":
		name := cloudWatchFormString(form, "AlarmName", "stackyard-alarm")
		s.alarms[name] = struct{}{}
		return map[string]any{}

	case "DeleteAlarms":
		for _, name := range cloudWatchFormSlice(form, "AlarmNames.member") {
			delete(s.alarms, name)
		}
		if name := cloudWatchFormString(form, "AlarmName", ""); name != "" {
			delete(s.alarms, name)
		}
		return map[string]any{}

	case "DescribeAlarms", "DescribeAlarmsForMetric":
		items := make([]any, 0, len(s.alarms))
		for _, name := range cloudWatchSortedKeys(s.alarms) {
			items = append(items, map[string]any{"AlarmName": name})
		}
		return map[string]any{
			"CompositeAlarms": []any{},
			"MetricAlarms":    items,
			"NextToken":       "",
		}

	case "DescribeAlarmHistory":
		return map[string]any{"AlarmHistoryItems": []any{}, "NextToken": ""}

	case "PutDashboard":
		name := cloudWatchFormString(form, "DashboardName", "stackyard-dashboard")
		body := cloudWatchFormString(form, "DashboardBody", "{}")
		s.dashboards[name] = body
		return map[string]any{"DashboardValidationMessages": []any{}}

	case "GetDashboard":
		name := cloudWatchFormString(form, "DashboardName", "stackyard-dashboard")
		body := s.dashboards[name]
		if body == "" {
			body = "{}"
		}
		return map[string]any{
			"DashboardArn":  fmt.Sprintf("arn:aws:cloudwatch::123456789012:dashboard/%s", name),
			"DashboardBody": body,
			"DashboardName": name,
		}

	case "ListDashboards":
		entries := make([]any, 0, len(s.dashboards))
		for name := range s.dashboards {
			entries = append(entries, map[string]any{"DashboardName": name})
		}
		sort.Slice(entries, func(i, j int) bool {
			left, _ := entries[i].(map[string]any)
			right, _ := entries[j].(map[string]any)
			return strings.Compare(fmt.Sprintf("%v", left["DashboardName"]), fmt.Sprintf("%v", right["DashboardName"])) < 0
		})
		return map[string]any{"DashboardEntries": entries}

	case "DeleteDashboards":
		for _, name := range cloudWatchFormSlice(form, "DashboardNames.member") {
			delete(s.dashboards, name)
		}
		return map[string]any{}

	case "PutMetricData":
		return map[string]any{}

	case "ListMetrics":
		return map[string]any{"Metrics": []any{}, "NextToken": ""}

	case "GetMetricStatistics":
		return map[string]any{"Datapoints": []any{}, "Label": "stackyard"}

	case "GetMetricData":
		return map[string]any{"MetricDataResults": []any{}, "Messages": []any{}, "NextToken": ""}

	case "GetMetricWidgetImage":
		return map[string]any{"MetricWidgetImage": "c3RhY2t5YXJk"}

	case "PutMetricStream":
		name := cloudWatchFormString(form, "Name", "stackyard-stream")
		arn := fmt.Sprintf("arn:aws:cloudwatch:us-east-1:123456789012:metric-stream/%s", name)
		s.metricStreams[name] = arn
		return map[string]any{"Arn": arn}

	case "GetMetricStream":
		name := cloudWatchFormString(form, "Name", "stackyard-stream")
		arn := s.metricStreams[name]
		if arn == "" {
			arn = fmt.Sprintf("arn:aws:cloudwatch:us-east-1:123456789012:metric-stream/%s", name)
		}
		return map[string]any{"Arn": arn, "Name": name}

	case "ListMetricStreams":
		entries := make([]any, 0, len(s.metricStreams))
		for name, arn := range s.metricStreams {
			entries = append(entries, map[string]any{"Arn": arn, "Name": name})
		}
		return map[string]any{"Entries": entries, "NextToken": ""}

	case "DeleteMetricStream":
		delete(s.metricStreams, cloudWatchFormString(form, "Name", "stackyard-stream"))
		return map[string]any{}

	case "StartMetricStreams", "StopMetricStreams":
		return map[string]any{}

	case "PutInsightRule":
		name := cloudWatchFormString(form, "RuleName", "stackyard-insight-rule")
		s.insightRules[name] = struct{}{}
		return map[string]any{}

	case "DescribeInsightRules":
		rules := make([]any, 0, len(s.insightRules))
		for _, name := range cloudWatchSortedKeys(s.insightRules) {
			rules = append(rules, map[string]any{"Name": name, "State": "ENABLED"})
		}
		return map[string]any{"InsightRules": rules, "NextToken": ""}

	case "GetInsightRuleReport":
		return map[string]any{"AggregateValue": 0.0, "Contributors": []any{}, "KeyLabels": []any{}, "MetricDatapoints": []any{}}

	case "DeleteInsightRules", "EnableInsightRules", "DisableInsightRules", "PutManagedInsightRules":
		return map[string]any{}

	case "ListManagedInsightRules":
		return map[string]any{"ManagedRules": []any{}, "NextToken": ""}

	case "PutAnomalyDetector", "DeleteAnomalyDetector":
		return map[string]any{}

	case "DescribeAnomalyDetectors":
		return map[string]any{"AnomalyDetectors": []any{}, "NextToken": ""}

	case "TagResource":
		arn := cloudWatchFormString(form, "ResourceARN", "arn:aws:cloudwatch:us-east-1:123456789012:alarm:stackyard-alarm")
		tags := s.tags[arn]
		if tags == nil {
			tags = map[string]string{}
			s.tags[arn] = tags
		}
		for key, values := range form {
			if !strings.HasPrefix(key, "Tags.member.") || !strings.HasSuffix(key, ".Key") || len(values) == 0 {
				continue
			}
			idx := strings.TrimSuffix(strings.TrimPrefix(key, "Tags.member."), ".Key")
			tagKey := strings.TrimSpace(values[0])
			if tagKey == "" {
				continue
			}
			tagValue := strings.TrimSpace(form.Get("Tags.member." + idx + ".Value"))
			tags[tagKey] = tagValue
		}
		return map[string]any{}

	case "UntagResource":
		arn := cloudWatchFormString(form, "ResourceARN", "arn:aws:cloudwatch:us-east-1:123456789012:alarm:stackyard-alarm")
		tags := s.tags[arn]
		for _, key := range cloudWatchFormSlice(form, "TagKeys.member") {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := cloudWatchFormString(form, "ResourceARN", "arn:aws:cloudwatch:us-east-1:123456789012:alarm:stackyard-alarm")
		tags := s.tags[arn]
		out := make([]any, 0, len(tags))
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": tags[k]})
		}
		return map[string]any{"Tags": out}

	case "PutAlarmMuteRule":
		name := cloudWatchFormString(form, "AlarmMuteRuleName", "stackyard-mute-rule")
		s.alarmMuteRules[name] = struct{}{}
		return map[string]any{}

	case "GetAlarmMuteRule":
		name := cloudWatchFormString(form, "AlarmMuteRuleName", "stackyard-mute-rule")
		return map[string]any{"AlarmMuteRule": map[string]any{"AlarmMuteRuleName": name}}

	case "ListAlarmMuteRules":
		items := make([]any, 0, len(s.alarmMuteRules))
		for _, name := range cloudWatchSortedKeys(s.alarmMuteRules) {
			items = append(items, map[string]any{"AlarmMuteRuleName": name})
		}
		return map[string]any{"AlarmMuteRules": items, "NextToken": ""}

	case "DeleteAlarmMuteRule":
		delete(s.alarmMuteRules, cloudWatchFormString(form, "AlarmMuteRuleName", "stackyard-mute-rule"))
		return map[string]any{}

	case "DescribeAlarmContributors":
		return map[string]any{"AlarmContributor": []any{}, "NextToken": ""}

	case "SetAlarmState":
		return map[string]any{}
	}

	switch {
	case strings.HasPrefix(action, "List"):
		return map[string]any{"NextToken": ""}
	case strings.HasPrefix(action, "Describe"), strings.HasPrefix(action, "Get"):
		return map[string]any{}
	case strings.HasPrefix(action, "Put"), strings.HasPrefix(action, "Start"), strings.HasPrefix(action, "Stop"), strings.HasPrefix(action, "Enable"), strings.HasPrefix(action, "Disable"), strings.HasPrefix(action, "Delete"), strings.HasPrefix(action, "Tag"), strings.HasPrefix(action, "Untag"), strings.HasPrefix(action, "Set"):
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func cloudWatchFormString(form url.Values, key, fallback string) string {
	value := strings.TrimSpace(form.Get(key))
	if value == "" {
		return fallback
	}
	return value
}

func cloudWatchFormSlice(form url.Values, prefix string) []string {
	values := []string{}
	for key, list := range form {
		if key == prefix {
			for _, item := range list {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" {
					values = append(values, trimmed)
				}
			}
			continue
		}
		if !strings.HasPrefix(key, prefix+".") || len(list) == 0 {
			continue
		}
		trimmed := strings.TrimSpace(list[0])
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func cloudWatchSortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
