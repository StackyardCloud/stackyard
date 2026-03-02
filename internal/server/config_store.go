package server

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type configStore struct {
	mu               sync.Mutex
	recorders        map[string]map[string]any
	recorderStatuses map[string]map[string]any
	deliveryChannels map[string]map[string]any
	configRules      map[string]map[string]any
	storedQueries    map[string]map[string]any
}

func newConfigStore() *configStore {
	return &configStore{
		recorders:        map[string]map[string]any{},
		recorderStatuses: map[string]map[string]any{},
		deliveryChannels: map[string]map[string]any{},
		configRules:      map[string]map[string]any{},
		storedQueries:    map[string]map[string]any{},
	}
}

func (s *configStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := configOperationByName[action]; !known {
		return map[string]any{}
	}

	switch action {
	case "PutConfigurationRecorder":
		return s.putConfigurationRecorder(payload)
	case "DeleteConfigurationRecorder":
		return s.deleteConfigurationRecorder(payload)
	case "DescribeConfigurationRecorders":
		return map[string]any{"ConfigurationRecorders": mapValuesSorted(s.recorders, "name")}
	case "StartConfigurationRecorder":
		return s.startConfigurationRecorder(payload)
	case "StopConfigurationRecorder":
		return s.stopConfigurationRecorder(payload)
	case "DescribeConfigurationRecorderStatus":
		return map[string]any{"ConfigurationRecordersStatus": mapValuesSorted(s.recorderStatuses, "name")}

	case "PutDeliveryChannel":
		return s.putDeliveryChannel(payload)
	case "DeleteDeliveryChannel":
		return s.deleteDeliveryChannel(payload)
	case "DescribeDeliveryChannels":
		return map[string]any{"DeliveryChannels": mapValuesSorted(s.deliveryChannels, "name")}
	case "DescribeDeliveryChannelStatus":
		return map[string]any{"DeliveryChannelsStatus": s.describeDeliveryChannelStatus()}

	case "PutConfigRule":
		return s.putConfigRule(payload)
	case "DeleteConfigRule":
		return s.deleteConfigRule(payload)
	case "DescribeConfigRules":
		return map[string]any{"ConfigRules": mapValuesSorted(s.configRules, "name")}
	case "DescribeConfigRuleEvaluationStatus":
		return map[string]any{"ConfigRulesEvaluationStatus": s.describeConfigRuleEvaluationStatus()}

	case "PutStoredQuery":
		return s.putStoredQuery(payload)
	case "GetStoredQuery":
		return s.getStoredQuery(payload)
	case "ListStoredQueries":
		return map[string]any{"StoredQueryMetadata": s.listStoredQueryMetadata(), "NextToken": ""}
	case "DeleteStoredQuery":
		return s.deleteStoredQuery(payload)

	case "ListConfigurationRecorders":
		return map[string]any{"ConfigurationRecorderNames": mapKeysSorted(s.recorders), "NextToken": ""}
	case "ListTagsForResource":
		return map[string]any{"Tags": []any{}}
	case "TagResource", "UntagResource":
		return map[string]any{}
	}

	return configDefaultResponse(action)
}

func (s *configStore) putConfigurationRecorder(payload map[string]any) map[string]any {
	recorder, _ := payload["ConfigurationRecorder"].(map[string]any)
	name := configMapString(recorder, "name", "Name")
	if name == "" {
		name = "default"
	}
	roleARN := configMapString(recorder, "roleARN", "RoleARN")
	if roleARN == "" {
		roleARN = "arn:aws:iam::123456789012:role/stackyard-config-recorder"
	}
	item := map[string]any{
		"name":    name,
		"Name":    name,
		"roleARN": roleARN,
		"RoleARN": roleARN,
	}
	if recordingGroup, ok := recorder["recordingGroup"]; ok {
		item["recordingGroup"] = recordingGroup
		item["RecordingGroup"] = recordingGroup
	}
	if recordingGroup, ok := recorder["RecordingGroup"]; ok {
		item["recordingGroup"] = recordingGroup
		item["RecordingGroup"] = recordingGroup
	}
	s.recorders[name] = item
	if _, ok := s.recorderStatuses[name]; !ok {
		now := time.Now().UTC().Format(time.RFC3339)
		s.recorderStatuses[name] = map[string]any{
			"name":                 name,
			"Name":                 name,
			"recording":            false,
			"Recording":            false,
			"lastStatusChangeTime": now,
			"LastStatusChangeTime": now,
		}
	}
	return map[string]any{}
}

func (s *configStore) deleteConfigurationRecorder(payload map[string]any) map[string]any {
	name := configMapString(payload, "ConfigurationRecorderName", "configurationRecorderName")
	if name != "" {
		delete(s.recorders, name)
		delete(s.recorderStatuses, name)
	}
	return map[string]any{}
}

func (s *configStore) startConfigurationRecorder(payload map[string]any) map[string]any {
	name := configMapString(payload, "ConfigurationRecorderName", "configurationRecorderName")
	if name == "" {
		name = "default"
	}
	status := s.recorderStatuses[name]
	if status == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		status = map[string]any{
			"name":                 name,
			"Name":                 name,
			"lastStatusChangeTime": now,
			"LastStatusChangeTime": now,
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	status["recording"] = true
	status["Recording"] = true
	status["lastStatusChangeTime"] = now
	status["LastStatusChangeTime"] = now
	s.recorderStatuses[name] = status
	return map[string]any{}
}

func (s *configStore) stopConfigurationRecorder(payload map[string]any) map[string]any {
	name := configMapString(payload, "ConfigurationRecorderName", "configurationRecorderName")
	if name == "" {
		name = "default"
	}
	status := s.recorderStatuses[name]
	if status == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		status = map[string]any{
			"name":                 name,
			"Name":                 name,
			"lastStatusChangeTime": now,
			"LastStatusChangeTime": now,
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	status["recording"] = false
	status["Recording"] = false
	status["lastStatusChangeTime"] = now
	status["LastStatusChangeTime"] = now
	s.recorderStatuses[name] = status
	return map[string]any{}
}

func (s *configStore) putDeliveryChannel(payload map[string]any) map[string]any {
	channel, _ := payload["DeliveryChannel"].(map[string]any)
	name := configMapString(channel, "name", "Name")
	if name == "" {
		name = "default"
	}
	item := map[string]any{
		"name": name,
		"Name": name,
	}
	for _, key := range []string{"s3BucketName", "S3BucketName", "s3KeyPrefix", "S3KeyPrefix", "snsTopicARN", "SnsTopicARN", "SNSChannelARN", "snsTopicArn"} {
		if v, ok := channel[key]; ok {
			item[key] = v
		}
	}
	s.deliveryChannels[name] = item
	return map[string]any{}
}

func (s *configStore) deleteDeliveryChannel(payload map[string]any) map[string]any {
	name := configMapString(payload, "DeliveryChannelName", "deliveryChannelName")
	if name != "" {
		delete(s.deliveryChannels, name)
	}
	return map[string]any{}
}

func (s *configStore) describeDeliveryChannelStatus() []any {
	out := make([]any, 0, len(s.deliveryChannels))
	now := time.Now().UTC().Format(time.RFC3339)
	for _, name := range mapKeysSorted(s.deliveryChannels) {
		out = append(out, map[string]any{
			"name":                      name,
			"Name":                      name,
			"configHistoryDeliveryInfo": map[string]any{"lastStatus": "SUCCESS", "lastSuccessfulTime": now},
			"snapshotDeliveryInfo":      map[string]any{"lastStatus": "SUCCESS", "lastSuccessfulTime": now},
		})
	}
	return out
}

func (s *configStore) putConfigRule(payload map[string]any) map[string]any {
	rule, _ := payload["ConfigRule"].(map[string]any)
	name := configMapString(rule, "ConfigRuleName", "configRuleName")
	if name == "" {
		name = fmt.Sprintf("config-rule-%d", len(s.configRules)+1)
	}
	item := cloneAnyMap(rule)
	item["ConfigRuleName"] = name
	item["configRuleName"] = name
	if _, ok := item["ConfigRuleArn"]; !ok {
		item["ConfigRuleArn"] = "arn:aws:config:us-east-1:123456789012:config-rule/" + name
	}
	if _, ok := item["ConfigRuleState"]; !ok {
		item["ConfigRuleState"] = "ACTIVE"
	}
	s.configRules[name] = item
	return map[string]any{}
}

func (s *configStore) deleteConfigRule(payload map[string]any) map[string]any {
	name := configMapString(payload, "ConfigRuleName", "configRuleName")
	if name != "" {
		delete(s.configRules, name)
	}
	return map[string]any{}
}

func (s *configStore) describeConfigRuleEvaluationStatus() []any {
	out := make([]any, 0, len(s.configRules))
	now := time.Now().UTC().Format(time.RFC3339)
	for _, name := range mapKeysSorted(s.configRules) {
		out = append(out, map[string]any{
			"ConfigRuleName":               name,
			"FirstActivatedTime":           now,
			"LastSuccessfulInvocationTime": now,
			"LastSuccessfulEvaluationTime": now,
		})
	}
	return out
}

func (s *configStore) putStoredQuery(payload map[string]any) map[string]any {
	query, _ := payload["StoredQuery"].(map[string]any)
	name := configMapString(query, "QueryName", "queryName")
	if name == "" {
		name = fmt.Sprintf("stored-query-%d", len(s.storedQueries)+1)
	}
	desc := configMapString(query, "Description", "description")
	expr := configMapString(query, "Expression", "expression")
	item := map[string]any{
		"QueryName":   name,
		"queryName":   name,
		"Description": desc,
		"description": desc,
		"Expression":  expr,
		"expression":  expr,
		"QueryArn":    "arn:aws:config:us-east-1:123456789012:stored-query/" + name,
	}
	s.storedQueries[name] = item
	return map[string]any{
		"QueryArn": item["QueryArn"],
	}
}

func (s *configStore) getStoredQuery(payload map[string]any) map[string]any {
	name := configMapString(payload, "QueryName", "queryName")
	if name == "" {
		return map[string]any{}
	}
	item := s.storedQueries[name]
	if item == nil {
		return map[string]any{}
	}
	return map[string]any{"StoredQuery": item}
}

func (s *configStore) listStoredQueryMetadata() []any {
	out := make([]any, 0, len(s.storedQueries))
	for _, name := range mapKeysSorted(s.storedQueries) {
		item := s.storedQueries[name]
		out = append(out, map[string]any{
			"QueryName":   item["QueryName"],
			"Description": item["Description"],
			"QueryArn":    item["QueryArn"],
		})
	}
	return out
}

func (s *configStore) deleteStoredQuery(payload map[string]any) map[string]any {
	name := configMapString(payload, "QueryName", "queryName")
	if name != "" {
		delete(s.storedQueries, name)
	}
	return map[string]any{}
}

func configDefaultResponse(action string) map[string]any {
	switch action {
	case "DescribeAggregationAuthorizations":
		return map[string]any{"AggregationAuthorizations": []any{}}
	case "DescribeConfigurationAggregators":
		return map[string]any{"ConfigurationAggregators": []any{}}
	case "DescribeConfigurationAggregatorSourcesStatus":
		return map[string]any{"AggregatedSourceStatusList": []any{}}
	case "DescribeConformancePacks":
		return map[string]any{"ConformancePackDetails": []any{}}
	case "DescribeConformancePackStatus":
		return map[string]any{"ConformancePackStatusDetails": []any{}}
	case "DescribeConformancePackCompliance":
		return map[string]any{"ConformancePackRuleComplianceList": []any{}, "NextToken": ""}
	case "DescribeOrganizationConfigRules":
		return map[string]any{"OrganizationConfigRules": []any{}, "NextToken": ""}
	case "DescribeOrganizationConfigRuleStatuses":
		return map[string]any{"OrganizationConfigRuleStatuses": []any{}, "NextToken": ""}
	case "DescribeOrganizationConformancePacks":
		return map[string]any{"OrganizationConformancePackDetails": []any{}, "NextToken": ""}
	case "DescribeOrganizationConformancePackStatuses":
		return map[string]any{"OrganizationConformancePackStatuses": []any{}, "NextToken": ""}
	case "DescribePendingAggregationRequests":
		return map[string]any{"PendingAggregationRequests": []any{}}
	case "DescribeRemediationConfigurations":
		return map[string]any{"RemediationConfigurations": []any{}}
	case "DescribeRemediationExceptions":
		return map[string]any{"RemediationExceptions": []any{}}
	case "DescribeRemediationExecutionStatus":
		return map[string]any{"RemediationExecutionStatuses": []any{}, "NextToken": ""}
	case "DescribeRetentionConfigurations":
		return map[string]any{"RetentionConfigurations": []any{}}

	case "GetStoredQuery":
		return map[string]any{"StoredQuery": map[string]any{}}
	case "GetResourceEvaluationSummary":
		return map[string]any{"ResourceEvaluation": map[string]any{}}
	case "GetConformancePackComplianceSummary":
		return map[string]any{"ConformancePackComplianceSummaryList": []any{}}
	case "GetConformancePackComplianceDetails":
		return map[string]any{"ConformancePackRuleEvaluationResults": []any{}, "NextToken": ""}
	case "GetComplianceSummaryByConfigRule":
		return map[string]any{"ComplianceSummary": map[string]any{}}
	case "GetComplianceSummaryByResourceType":
		return map[string]any{"ComplianceSummariesByResourceType": []any{}}
	case "GetComplianceDetailsByConfigRule":
		return map[string]any{"EvaluationResults": []any{}, "NextToken": ""}
	case "GetComplianceDetailsByResource":
		return map[string]any{"EvaluationResults": []any{}, "NextToken": ""}
	case "GetDiscoveredResourceCounts":
		return map[string]any{"TotalDiscoveredResources": 0, "ResourceCounts": []any{}, "NextToken": ""}
	case "GetResourceConfigHistory":
		return map[string]any{"ConfigurationItems": []any{}, "NextToken": ""}
	case "GetAggregateResourceConfig":
		return map[string]any{"ConfigurationItem": map[string]any{}}
	case "GetAggregateDiscoveredResourceCounts":
		return map[string]any{"TotalDiscoveredResources": 0, "GroupByKey": "", "GroupByCounts": []any{}, "NextToken": ""}
	case "GetAggregateComplianceDetailsByConfigRule":
		return map[string]any{"AggregateEvaluationResults": []any{}, "NextToken": ""}
	case "GetAggregateConfigRuleComplianceSummary":
		return map[string]any{"AggregateComplianceByConfigRules": []any{}}
	case "GetAggregateConformancePackComplianceSummary":
		return map[string]any{"AggregateConformancePackComplianceSummaries": []any{}, "NextToken": ""}

	case "SelectResourceConfig", "SelectAggregateResourceConfig":
		return map[string]any{"Results": []any{}, "QueryInfo": map[string]any{}, "NextToken": ""}
	case "ListAggregateDiscoveredResources", "ListDiscoveredResources":
		return map[string]any{"ResourceIdentifiers": []any{}, "NextToken": ""}
	case "ListConformancePackComplianceScores":
		return map[string]any{"ConformancePackComplianceScores": []any{}, "NextToken": ""}
	case "ListResourceEvaluations":
		return map[string]any{"ResourceEvaluations": []any{}, "NextToken": ""}
	}

	switch {
	case strings.HasPrefix(action, "Delete"),
		strings.HasPrefix(action, "Put"),
		strings.HasPrefix(action, "Start"),
		strings.HasPrefix(action, "Stop"),
		strings.HasPrefix(action, "Deliver"),
		strings.HasPrefix(action, "Associate"),
		strings.HasPrefix(action, "Disassociate"),
		strings.HasPrefix(action, "Batch"):
		return map[string]any{}
	case strings.HasPrefix(action, "Describe"):
		return map[string]any{}
	case strings.HasPrefix(action, "Get"):
		return map[string]any{}
	case strings.HasPrefix(action, "List"):
		return map[string]any{"NextToken": ""}
	default:
		return map[string]any{}
	}
}

func mapValuesSorted(values map[string]map[string]any, keyField string) []any {
	keys := mapKeysSorted(values)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		item := cloneAnyMap(values[key])
		if keyField != "" {
			item[keyField] = key
		}
		out = append(out, item)
	}
	return out
}

func mapKeysSorted[V any](values map[string]V) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sortStrings(out)
	return out
}

func configMapString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := values[key]
		if !ok || raw == nil {
			continue
		}
		if s, ok := raw.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func sortStrings(items []string) {
	if len(items) < 2 {
		return
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j] < items[j-1]; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
