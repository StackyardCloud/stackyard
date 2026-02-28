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
	cloudWatchObservabilityAdminDefaultTelemetryPipelineID = "stackyard-telemetry-pipeline"
	cloudWatchObservabilityAdminDefaultTelemetryRuleID     = "stackyard-telemetry-rule"
	cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID  = "stackyard-org-telemetry-rule"
	cloudWatchObservabilityAdminDefaultCentralizationID    = "stackyard-centralization-rule"
	cloudWatchObservabilityAdminDefaultS3IntegrationID     = "stackyard-s3-integration"
)

type cloudWatchObservabilityAdminStore struct {
	mu sync.Mutex

	nextID int64

	telemetryPipelines map[string]map[string]any
	telemetryRules     map[string]map[string]any
	orgTelemetryRules  map[string]map[string]any
	centralization     map[string]map[string]any
	s3Integrations     map[string]map[string]any
	tags               map[string]map[string]string

	enrichmentStatus string
	evaluationStatus string
	orgEvalStatus    string
}

func newCloudWatchObservabilityAdminStore() *cloudWatchObservabilityAdminStore {
	s := &cloudWatchObservabilityAdminStore{
		nextID:             2,
		telemetryPipelines: map[string]map[string]any{},
		telemetryRules:     map[string]map[string]any{},
		orgTelemetryRules:  map[string]map[string]any{},
		centralization:     map[string]map[string]any{},
		s3Integrations:     map[string]map[string]any{},
		tags:               map[string]map[string]string{},
		enrichmentStatus:   "STOPPED",
		evaluationStatus:   "STOPPED",
		orgEvalStatus:      "STOPPED",
	}

	pipeline := s.ensureTelemetryPipelineLocked(cloudWatchObservabilityAdminDefaultTelemetryPipelineID)
	rule := s.ensureTelemetryRuleLocked(cloudWatchObservabilityAdminDefaultTelemetryRuleID)
	orgRule := s.ensureOrgTelemetryRuleLocked(cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID)
	centralization := s.ensureCentralizationRuleLocked(cloudWatchObservabilityAdminDefaultCentralizationID)
	integration := s.ensureS3IntegrationLocked(cloudWatchObservabilityAdminDefaultS3IntegrationID)

	s.tags[cloudWatchObservabilityAdminResourceARN(pipeline)] = map[string]string{"seed": "true"}
	s.tags[cloudWatchObservabilityAdminResourceARN(rule)] = map[string]string{"seed": "true"}
	s.tags[cloudWatchObservabilityAdminResourceARN(orgRule)] = map[string]string{"seed": "true"}
	s.tags[cloudWatchObservabilityAdminResourceARN(centralization)] = map[string]string{"seed": "true"}
	s.tags[cloudWatchObservabilityAdminResourceARN(integration)] = map[string]string{"seed": "true"}

	return s
}

func (s *cloudWatchObservabilityAdminStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "CreateTelemetryPipeline":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryPipelineId", "PipelineId", "Name", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryPipelineID)
		item := s.ensureTelemetryPipelineLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"TelemetryPipeline": cloudWatchObservabilityAdminCloneMap(item)}

	case "DeleteTelemetryPipeline":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryPipelineId", "PipelineId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryPipelineID)
		if item := s.telemetryPipelines[id]; item != nil {
			delete(s.tags, cloudWatchObservabilityAdminResourceARN(item))
		}
		delete(s.telemetryPipelines, id)
		return map[string]any{}

	case "GetTelemetryPipeline":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryPipelineId", "PipelineId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryPipelineID)
		item := s.ensureTelemetryPipelineLocked(id)
		return map[string]any{"TelemetryPipeline": cloudWatchObservabilityAdminCloneMap(item)}

	case "ListTelemetryPipelines":
		return map[string]any{"TelemetryPipelines": s.listValuesLocked(s.telemetryPipelines), "NextToken": ""}

	case "UpdateTelemetryPipeline":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryPipelineId", "PipelineId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryPipelineID)
		item := s.ensureTelemetryPipelineLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"TelemetryPipeline": cloudWatchObservabilityAdminCloneMap(item)}

	case "TestTelemetryPipeline":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryPipelineId", "PipelineId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryPipelineID)
		_ = s.ensureTelemetryPipelineLocked(id)
		return map[string]any{
			"Success":              true,
			"PipelineOutput":       []any{},
			"PipelineOutputErrors": []any{},
		}

	case "ValidateTelemetryPipelineConfiguration":
		return map[string]any{"IsValid": true, "ValidationErrors": []any{}}

	case "CreateTelemetryRule":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Name", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryRuleID)
		item := s.ensureTelemetryRuleLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"TelemetryRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "DeleteTelemetryRule":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryRuleID)
		if item := s.telemetryRules[id]; item != nil {
			delete(s.tags, cloudWatchObservabilityAdminResourceARN(item))
		}
		delete(s.telemetryRules, id)
		return map[string]any{}

	case "GetTelemetryRule":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryRuleID)
		item := s.ensureTelemetryRuleLocked(id)
		return map[string]any{"TelemetryRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "ListTelemetryRules":
		return map[string]any{"TelemetryRules": s.listValuesLocked(s.telemetryRules), "NextToken": ""}

	case "UpdateTelemetryRule":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultTelemetryRuleID)
		item := s.ensureTelemetryRuleLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"TelemetryRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "CreateTelemetryRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Name", "Identifier"}, cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID)
		item := s.ensureOrgTelemetryRuleLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"TelemetryRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "DeleteTelemetryRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID)
		if item := s.orgTelemetryRules[id]; item != nil {
			delete(s.tags, cloudWatchObservabilityAdminResourceARN(item))
		}
		delete(s.orgTelemetryRules, id)
		return map[string]any{}

	case "GetTelemetryRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID)
		item := s.ensureOrgTelemetryRuleLocked(id)
		return map[string]any{"TelemetryRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "ListTelemetryRulesForOrganization":
		return map[string]any{"TelemetryRules": s.listValuesLocked(s.orgTelemetryRules), "NextToken": ""}

	case "UpdateTelemetryRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"TelemetryRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID)
		item := s.ensureOrgTelemetryRuleLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"TelemetryRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "CreateCentralizationRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"CentralizationRuleId", "RuleId", "Name", "Identifier"}, cloudWatchObservabilityAdminDefaultCentralizationID)
		item := s.ensureCentralizationRuleLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"CentralizationRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "DeleteCentralizationRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"CentralizationRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultCentralizationID)
		if item := s.centralization[id]; item != nil {
			delete(s.tags, cloudWatchObservabilityAdminResourceARN(item))
		}
		delete(s.centralization, id)
		return map[string]any{}

	case "GetCentralizationRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"CentralizationRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultCentralizationID)
		item := s.ensureCentralizationRuleLocked(id)
		return map[string]any{"CentralizationRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "ListCentralizationRulesForOrganization":
		return map[string]any{"CentralizationRules": s.listValuesLocked(s.centralization), "NextToken": ""}

	case "UpdateCentralizationRuleForOrganization":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"CentralizationRuleId", "RuleId", "Identifier"}, cloudWatchObservabilityAdminDefaultCentralizationID)
		item := s.ensureCentralizationRuleLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"CentralizationRule": cloudWatchObservabilityAdminCloneMap(item)}

	case "CreateS3TableIntegration":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"S3TableIntegrationId", "IntegrationId", "Name", "Identifier"}, cloudWatchObservabilityAdminDefaultS3IntegrationID)
		item := s.ensureS3IntegrationLocked(id)
		s.applyPayloadLocked(item, payload)
		item["LastUpdatedTime"] = now
		return map[string]any{"S3TableIntegration": cloudWatchObservabilityAdminCloneMap(item)}

	case "DeleteS3TableIntegration":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"S3TableIntegrationId", "IntegrationId", "Identifier"}, cloudWatchObservabilityAdminDefaultS3IntegrationID)
		if item := s.s3Integrations[id]; item != nil {
			delete(s.tags, cloudWatchObservabilityAdminResourceARN(item))
		}
		delete(s.s3Integrations, id)
		return map[string]any{}

	case "GetS3TableIntegration":
		id := cloudWatchObservabilityAdminResolveID(payload, []string{"S3TableIntegrationId", "IntegrationId", "Identifier"}, cloudWatchObservabilityAdminDefaultS3IntegrationID)
		item := s.ensureS3IntegrationLocked(id)
		return map[string]any{"S3TableIntegration": cloudWatchObservabilityAdminCloneMap(item)}

	case "ListS3TableIntegrations":
		return map[string]any{"S3TableIntegrations": s.listValuesLocked(s.s3Integrations), "NextToken": ""}

	case "StartTelemetryEvaluation":
		s.evaluationStatus = "RUNNING"
		return map[string]any{"Status": s.evaluationStatus, "LastUpdatedTime": now}

	case "StopTelemetryEvaluation":
		s.evaluationStatus = "STOPPED"
		return map[string]any{"Status": s.evaluationStatus, "LastUpdatedTime": now}

	case "GetTelemetryEvaluationStatus":
		return map[string]any{"Status": s.evaluationStatus, "LastUpdatedTime": now}

	case "StartTelemetryEvaluationForOrganization":
		s.orgEvalStatus = "RUNNING"
		return map[string]any{"Status": s.orgEvalStatus, "LastUpdatedTime": now}

	case "StopTelemetryEvaluationForOrganization":
		s.orgEvalStatus = "STOPPED"
		return map[string]any{"Status": s.orgEvalStatus, "LastUpdatedTime": now}

	case "GetTelemetryEvaluationStatusForOrganization":
		return map[string]any{"Status": s.orgEvalStatus, "LastUpdatedTime": now}

	case "StartTelemetryEnrichment":
		s.enrichmentStatus = "RUNNING"
		return map[string]any{"Status": s.enrichmentStatus, "LastUpdatedTime": now}

	case "StopTelemetryEnrichment":
		s.enrichmentStatus = "STOPPED"
		return map[string]any{"Status": s.enrichmentStatus, "LastUpdatedTime": now}

	case "GetTelemetryEnrichmentStatus":
		return map[string]any{"Status": s.enrichmentStatus, "LastUpdatedTime": now}

	case "ListResourceTelemetry":
		return map[string]any{"ResourceTelemetry": s.buildResourceTelemetry(false), "NextToken": ""}

	case "ListResourceTelemetryForOrganization":
		return map[string]any{"ResourceTelemetry": s.buildResourceTelemetry(true), "NextToken": ""}

	case "TagResource":
		resourceARN := cloudWatchObservabilityAdminResolveID(payload, []string{"ResourceArn", "ResourceARN"}, cloudWatchObservabilityAdminResourceARN(s.ensureTelemetryPipelineLocked(cloudWatchObservabilityAdminDefaultTelemetryPipelineID)))
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range cloudWatchObservabilityAdminMapString(cloudWatchObservabilityAdminAny(payload, "Tags")) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := cloudWatchObservabilityAdminResolveID(payload, []string{"ResourceArn", "ResourceARN"}, cloudWatchObservabilityAdminResourceARN(s.ensureTelemetryPipelineLocked(cloudWatchObservabilityAdminDefaultTelemetryPipelineID)))
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range cloudWatchObservabilityAdminStringSlice(cloudWatchObservabilityAdminAny(payload, "TagKeys")) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := cloudWatchObservabilityAdminResolveID(payload, []string{"ResourceArn", "ResourceARN"}, cloudWatchObservabilityAdminResourceARN(s.ensureTelemetryPipelineLocked(cloudWatchObservabilityAdminDefaultTelemetryPipelineID)))
		return map[string]any{"Tags": cloudWatchObservabilityAdminCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *cloudWatchObservabilityAdminStore) ensureTelemetryPipelineLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = cloudWatchObservabilityAdminDefaultTelemetryPipelineID
	}
	if item := s.telemetryPipelines[id]; item != nil {
		return item
	}
	item := map[string]any{
		"TelemetryPipelineId": id,
		"Name":                id,
		"Arn":                 fmt.Sprintf("arn:aws:observabilityadmin:us-east-1:123456789012:telemetry-pipeline/%s", id),
		"Status":              "ACTIVE",
		"CreatedTime":         time.Now().UTC(),
		"LastUpdatedTime":     time.Now().UTC(),
	}
	s.telemetryPipelines[id] = item
	return item
}

func (s *cloudWatchObservabilityAdminStore) ensureTelemetryRuleLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = cloudWatchObservabilityAdminDefaultTelemetryRuleID
	}
	if item := s.telemetryRules[id]; item != nil {
		return item
	}
	item := map[string]any{
		"TelemetryRuleId": id,
		"Name":            id,
		"Arn":             fmt.Sprintf("arn:aws:observabilityadmin:us-east-1:123456789012:telemetry-rule/%s", id),
		"Status":          "ACTIVE",
		"CreatedTime":     time.Now().UTC(),
		"LastUpdatedTime": time.Now().UTC(),
	}
	s.telemetryRules[id] = item
	return item
}

func (s *cloudWatchObservabilityAdminStore) ensureOrgTelemetryRuleLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID
	}
	if item := s.orgTelemetryRules[id]; item != nil {
		return item
	}
	item := map[string]any{
		"TelemetryRuleId": id,
		"Name":            id,
		"Arn":             fmt.Sprintf("arn:aws:observabilityadmin:us-east-1:123456789012:telemetry-rule-organization/%s", id),
		"Status":          "ACTIVE",
		"CreatedTime":     time.Now().UTC(),
		"LastUpdatedTime": time.Now().UTC(),
	}
	s.orgTelemetryRules[id] = item
	return item
}

func (s *cloudWatchObservabilityAdminStore) ensureCentralizationRuleLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = cloudWatchObservabilityAdminDefaultCentralizationID
	}
	if item := s.centralization[id]; item != nil {
		return item
	}
	item := map[string]any{
		"CentralizationRuleId": id,
		"Name":                 id,
		"Arn":                  fmt.Sprintf("arn:aws:observabilityadmin:us-east-1:123456789012:centralization-rule/%s", id),
		"Status":               "ACTIVE",
		"CreatedTime":          time.Now().UTC(),
		"LastUpdatedTime":      time.Now().UTC(),
	}
	s.centralization[id] = item
	return item
}

func (s *cloudWatchObservabilityAdminStore) ensureS3IntegrationLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = cloudWatchObservabilityAdminDefaultS3IntegrationID
	}
	if item := s.s3Integrations[id]; item != nil {
		return item
	}
	item := map[string]any{
		"S3TableIntegrationId": id,
		"Name":                 id,
		"Arn":                  fmt.Sprintf("arn:aws:observabilityadmin:us-east-1:123456789012:s3-table-integration/%s", id),
		"Status":               "ACTIVE",
		"CreatedTime":          time.Now().UTC(),
		"LastUpdatedTime":      time.Now().UTC(),
	}
	s.s3Integrations[id] = item
	return item
}

func (s *cloudWatchObservabilityAdminStore) listValuesLocked(collection map[string]map[string]any) []any {
	keys := make([]string, 0, len(collection))
	for key := range collection {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, cloudWatchObservabilityAdminCloneMap(collection[key]))
	}
	return items
}

func (s *cloudWatchObservabilityAdminStore) applyPayloadLocked(item map[string]any, payload map[string]any) {
	for key, value := range payload {
		item[key] = value
	}
}

func (s *cloudWatchObservabilityAdminStore) ensureTagsLocked(resourceARN string) map[string]string {
	tags := s.tags[resourceARN]
	if tags == nil {
		tags = map[string]string{}
		s.tags[resourceARN] = tags
	}
	return tags
}

func (s *cloudWatchObservabilityAdminStore) buildResourceTelemetry(isOrg bool) []any {
	resources := []any{
		map[string]any{
			"ResourceArn":  cloudWatchObservabilityAdminResourceARN(s.ensureTelemetryPipelineLocked(cloudWatchObservabilityAdminDefaultTelemetryPipelineID)),
			"ResourceType": "AWS::CloudWatch::TelemetryPipeline",
			"Configurations": []any{
				map[string]any{"Type": "METRICS", "Status": "ENABLED"},
			},
		},
	}
	if isOrg {
		resources = append(resources, map[string]any{
			"ResourceArn":  cloudWatchObservabilityAdminResourceARN(s.ensureOrgTelemetryRuleLocked(cloudWatchObservabilityAdminDefaultOrgTelemetryRuleID)),
			"ResourceType": "AWS::CloudWatch::TelemetryRuleForOrganization",
			"Configurations": []any{
				map[string]any{"Type": "LOGS", "Status": "ENABLED"},
			},
		})
	}
	return resources
}

func cloudWatchObservabilityAdminResolveID(payload map[string]any, keys []string, fallback string) string {
	for _, key := range keys {
		if value := cloudWatchObservabilityAdminString(payload, key); value != "" {
			return value
		}
	}
	return fallback
}

func cloudWatchObservabilityAdminString(payload map[string]any, key string) string {
	value := cloudWatchObservabilityAdminAny(payload, key)
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(str)
}

func cloudWatchObservabilityAdminAny(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return value
		}
	}
	return nil
}

func cloudWatchObservabilityAdminMapString(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, val := range typed {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(val)
		}
	case map[string]any:
		for key, val := range typed {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			if str, ok := val.(string); ok {
				out[k] = strings.TrimSpace(str)
			}
		}
	}
	return out
}

func cloudWatchObservabilityAdminStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	out := []string{}
	switch typed := value.(type) {
	case []string:
		for _, item := range typed {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	case []any:
		for _, item := range typed {
			if str, ok := item.(string); ok {
				trimmed := strings.TrimSpace(str)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
	}
	return out
}

func cloudWatchObservabilityAdminResourceARN(item map[string]any) string {
	for _, key := range []string{"ResourceArn", "ARN", "Arn"} {
		if value := cloudWatchObservabilityAdminString(item, key); value != "" {
			return value
		}
	}
	return "arn:aws:observabilityadmin:us-east-1:123456789012:resource/stackyard"
}

func cloudWatchObservabilityAdminCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	buf, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(buf, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func cloudWatchObservabilityAdminCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
