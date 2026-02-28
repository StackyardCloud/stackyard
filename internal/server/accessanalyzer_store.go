package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type accessAnalyzerStore struct {
	mu sync.Mutex

	accountID string
	region    string
	nextID    int64

	analyzers    map[string]map[string]any
	archiveRules map[string]map[string]map[string]any
	policyJobs   map[string]map[string]any
	tags         map[string]map[string]string
}

func newAccessAnalyzerStore() *accessAnalyzerStore {
	s := &accessAnalyzerStore{
		accountID:    "123456789012",
		region:       "us-east-1",
		nextID:       1,
		analyzers:    map[string]map[string]any{},
		archiveRules: map[string]map[string]map[string]any{},
		policyJobs:   map[string]map[string]any{},
		tags:         map[string]map[string]string{},
	}
	now := time.Now().UTC().Format(time.RFC3339)
	s.analyzers["stackyard-analyzer"] = s.newAnalyzer("stackyard-analyzer", now)
	return s
}

func (s *accessAnalyzerStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	analyzerName := s.resolveAnalyzerName(payload, pathParams, query)
	analyzerARN := s.resolveAnalyzerARN(payload, pathParams, query, analyzerName)
	ruleName := accessAnalyzerFirstNonEmpty(
		accessAnalyzerPathParam(pathParams, "ruleName"),
		accessAnalyzerPayloadString(payload, "ruleName"),
		"stackyard-rule",
	)
	findingID := accessAnalyzerFirstNonEmpty(accessAnalyzerPathParam(pathParams, "id"), "finding-000001")
	jobID := accessAnalyzerFirstNonEmpty(accessAnalyzerPathParam(pathParams, "jobId"), "job-000001")
	accessPreviewID := accessAnalyzerFirstNonEmpty(accessAnalyzerPathParam(pathParams, "accessPreviewId"), "ap-000001")
	resourceARN := accessAnalyzerFirstNonEmpty(
		accessAnalyzerPathParam(pathParams, "resourceArn"),
		accessAnalyzerPayloadString(payload, "resourceArn"),
		"arn:aws:s3:::stackyard-bucket",
	)

	switch action {
	case "CreateAnalyzer":
		analyzerName = accessAnalyzerFirstNonEmpty(accessAnalyzerPayloadString(payload, "analyzerName"), analyzerName)
		analyzer := s.ensureAnalyzerLocked(analyzerName, now)
		if t := accessAnalyzerPayloadString(payload, "type"); t != "" {
			analyzer["type"] = t
		}
		analyzer["lastResourceAnalyzed"] = now
		return accessAnalyzerCloneMap(analyzer)

	case "DeleteAnalyzer":
		delete(s.analyzers, analyzerName)
		delete(s.archiveRules, analyzerName)
		return map[string]any{}

	case "GetAnalyzer":
		return accessAnalyzerCloneMap(s.ensureAnalyzerLocked(analyzerName, now))

	case "ListAnalyzers":
		return map[string]any{
			"analyzers": s.listAnalyzersLocked(now),
			"nextToken": "",
		}

	case "UpdateAnalyzer":
		analyzer := s.ensureAnalyzerLocked(analyzerName, now)
		analyzer["lastResourceAnalyzed"] = now
		return accessAnalyzerCloneMap(analyzer)

	case "CreateArchiveRule", "UpdateArchiveRule":
		rule := s.ensureArchiveRuleLocked(analyzerName, ruleName, now)
		rule["updatedAt"] = now
		return map[string]any{"archiveRule": accessAnalyzerCloneMap(rule)}

	case "DeleteArchiveRule":
		rule := s.ensureArchiveRuleLocked(analyzerName, ruleName, now)
		delete(s.ensureArchiveRuleMapLocked(analyzerName), ruleName)
		return map[string]any{"archiveRule": accessAnalyzerCloneMap(rule)}

	case "GetArchiveRule":
		return map[string]any{"archiveRule": accessAnalyzerCloneMap(s.ensureArchiveRuleLocked(analyzerName, ruleName, now))}

	case "ListArchiveRules":
		return map[string]any{
			"archiveRules": s.listArchiveRulesLocked(analyzerName, now),
			"nextToken":    "",
		}

	case "ApplyArchiveRule":
		return map[string]any{}

	case "CreateAccessPreview":
		return map[string]any{"id": accessPreviewID}

	case "GetAccessPreview":
		return map[string]any{
			"id":          accessPreviewID,
			"analyzerArn": analyzerARN,
			"status":      "COMPLETED",
		}

	case "ListAccessPreviews":
		return map[string]any{
			"accessPreviews": []any{
				map[string]any{
					"id":          accessPreviewID,
					"analyzerArn": analyzerARN,
					"status":      "COMPLETED",
					"createdAt":   now,
				},
			},
			"nextToken": "",
		}

	case "ListAccessPreviewFindings":
		return map[string]any{
			"findings": []any{
				map[string]any{
					"id":          findingID,
					"resourceArn": resourceARN,
					"status":      "ACTIVE",
				},
			},
			"nextToken": "",
		}

	case "GetAnalyzedResource":
		return map[string]any{
			"resource": map[string]any{
				"resourceArn":  resourceARN,
				"resourceType": "AWS::S3::Bucket",
				"createdAt":    now,
				"status":       "ACTIVE",
			},
		}

	case "ListAnalyzedResources":
		return map[string]any{
			"analyzedResources": []any{
				map[string]any{
					"resourceArn":  resourceARN,
					"resourceType": "AWS::S3::Bucket",
					"status":       "ACTIVE",
				},
			},
			"nextToken": "",
		}

	case "GetFinding", "GetFindingV2":
		return map[string]any{
			"id":          findingID,
			"analyzerArn": analyzerARN,
			"resourceArn": resourceARN,
			"status":      "ACTIVE",
		}

	case "ListFindings", "ListFindingsV2":
		return map[string]any{
			"findings": []any{
				map[string]any{
					"id":          findingID,
					"analyzerArn": analyzerARN,
					"resourceArn": resourceARN,
					"status":      "ACTIVE",
				},
			},
			"nextToken": "",
		}

	case "UpdateFindings":
		return map[string]any{}

	case "GetFindingsStatistics":
		return map[string]any{
			"findingsStatistics": map[string]any{
				"active": 1,
			},
		}

	case "GenerateFindingRecommendation":
		return map[string]any{}

	case "GetFindingRecommendation":
		return map[string]any{
			"recommendedSteps": []any{
				map[string]any{
					"unusedPermissionsRecommendedStep": map[string]any{
						"recommendedAction": "REMOVE",
					},
				},
			},
		}

	case "StartPolicyGeneration":
		s.nextID++
		jobID = fmt.Sprintf("job-%06d", s.nextID)
		s.policyJobs[jobID] = map[string]any{
			"jobId":       jobID,
			"status":      "SUCCEEDED",
			"startedOn":   now,
			"completedOn": now,
		}
		return map[string]any{"jobId": jobID}

	case "CancelPolicyGeneration":
		job := s.ensurePolicyJobLocked(jobID, now)
		job["status"] = "CANCELLED"
		return map[string]any{}

	case "GetGeneratedPolicy":
		job := s.ensurePolicyJobLocked(jobID, now)
		return map[string]any{
			"jobDetails":            accessAnalyzerCloneMap(job),
			"generatedPolicyResult": map[string]any{"generatedPolicies": []any{}},
		}

	case "ListPolicyGenerations":
		return map[string]any{
			"policyGenerations": s.listPolicyJobsLocked(now),
			"nextToken":         "",
		}

	case "CheckAccessNotGranted", "CheckNoNewAccess", "CheckNoPublicAccess":
		return map[string]any{
			"result": "PASS",
		}

	case "ValidatePolicy":
		return map[string]any{
			"findings":  []any{},
			"nextToken": "",
		}

	case "StartResourceScan":
		return map[string]any{}

	case "TagResource":
		tagSet := s.tags[resourceARN]
		if tagSet == nil {
			tagSet = map[string]string{}
			s.tags[resourceARN] = tagSet
		}
		if tags, ok := payload["tags"].(map[string]any); ok {
			for k, v := range tags {
				tagSet[k] = accessAnalyzerAnyString(v)
			}
		}
		return map[string]any{}

	case "UntagResource":
		tagSet := s.tags[resourceARN]
		if tagSet != nil {
			for _, key := range query["tagKeys"] {
				delete(tagSet, strings.TrimSpace(key))
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		out := map[string]string{}
		for k, v := range s.tags[resourceARN] {
			out[k] = v
		}
		return map[string]any{"tags": out}

	default:
		return map[string]any{}
	}
}

func (s *accessAnalyzerStore) resolveAnalyzerName(payload map[string]any, pathParams map[string]string, query url.Values) string {
	return accessAnalyzerFirstNonEmpty(
		accessAnalyzerPathParam(pathParams, "analyzerName"),
		accessAnalyzerPayloadString(payload, "analyzerName"),
		strings.TrimSpace(query.Get("analyzerName")),
		"stackyard-analyzer",
	)
}

func (s *accessAnalyzerStore) resolveAnalyzerARN(payload map[string]any, pathParams map[string]string, query url.Values, analyzerName string) string {
	return accessAnalyzerFirstNonEmpty(
		strings.TrimSpace(query.Get("analyzerArn")),
		accessAnalyzerPayloadString(payload, "analyzerArn"),
		accessAnalyzerPathParam(pathParams, "analyzerArn"),
		accessAnalyzerAnalyzerARN(s.region, s.accountID, analyzerName),
	)
}

func (s *accessAnalyzerStore) ensureAnalyzerLocked(name, now string) map[string]any {
	name = accessAnalyzerFirstNonEmpty(strings.TrimSpace(name), "stackyard-analyzer")
	if existing, ok := s.analyzers[name]; ok {
		return existing
	}
	analyzer := s.newAnalyzer(name, now)
	s.analyzers[name] = analyzer
	return analyzer
}

func (s *accessAnalyzerStore) newAnalyzer(name, now string) map[string]any {
	return map[string]any{
		"name":                 name,
		"arn":                  accessAnalyzerAnalyzerARN(s.region, s.accountID, name),
		"status":               "ACTIVE",
		"type":                 "ACCOUNT",
		"createdAt":            now,
		"lastResourceAnalyzed": now,
	}
}

func (s *accessAnalyzerStore) listAnalyzersLocked(now string) []any {
	names := make([]string, 0, len(s.analyzers))
	for name := range s.analyzers {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, accessAnalyzerCloneMap(s.ensureAnalyzerLocked(name, now)))
	}
	return out
}

func (s *accessAnalyzerStore) ensureArchiveRuleMapLocked(analyzerName string) map[string]map[string]any {
	analyzerName = accessAnalyzerFirstNonEmpty(strings.TrimSpace(analyzerName), "stackyard-analyzer")
	rules := s.archiveRules[analyzerName]
	if rules == nil {
		rules = map[string]map[string]any{}
		s.archiveRules[analyzerName] = rules
	}
	return rules
}

func (s *accessAnalyzerStore) ensureArchiveRuleLocked(analyzerName, ruleName, now string) map[string]any {
	rules := s.ensureArchiveRuleMapLocked(analyzerName)
	ruleName = accessAnalyzerFirstNonEmpty(strings.TrimSpace(ruleName), "stackyard-rule")
	if existing, ok := rules[ruleName]; ok {
		return existing
	}
	rule := map[string]any{
		"ruleName":     ruleName,
		"createdAt":    now,
		"updatedAt":    now,
		"filter":       map[string]any{},
		"analyzerName": accessAnalyzerFirstNonEmpty(strings.TrimSpace(analyzerName), "stackyard-analyzer"),
	}
	rules[ruleName] = rule
	return rule
}

func (s *accessAnalyzerStore) listArchiveRulesLocked(analyzerName, now string) []any {
	rules := s.ensureArchiveRuleMapLocked(analyzerName)
	names := make([]string, 0, len(rules))
	for name := range rules {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, accessAnalyzerCloneMap(s.ensureArchiveRuleLocked(analyzerName, name, now)))
	}
	return out
}

func (s *accessAnalyzerStore) ensurePolicyJobLocked(jobID, now string) map[string]any {
	jobID = accessAnalyzerFirstNonEmpty(strings.TrimSpace(jobID), "job-000001")
	if existing, ok := s.policyJobs[jobID]; ok {
		return existing
	}
	job := map[string]any{
		"jobId":       jobID,
		"status":      "SUCCEEDED",
		"startedOn":   now,
		"completedOn": now,
	}
	s.policyJobs[jobID] = job
	return job
}

func (s *accessAnalyzerStore) listPolicyJobsLocked(now string) []any {
	ids := make([]string, 0, len(s.policyJobs))
	for id := range s.policyJobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, accessAnalyzerCloneMap(s.ensurePolicyJobLocked(id, now)))
	}
	return out
}

func accessAnalyzerAnalyzerARN(region, accountID, analyzerName string) string {
	return fmt.Sprintf("arn:aws:access-analyzer:%s:%s:analyzer/%s", region, accountID, analyzerName)
}

func accessAnalyzerPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if v, ok := pathParams[key]; ok {
		return strings.TrimSpace(v)
	}
	for k, v := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func accessAnalyzerPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if v, ok := payload[key]; ok {
		return strings.TrimSpace(accessAnalyzerAnyString(v))
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(accessAnalyzerAnyString(v))
		}
	}
	return ""
}

func accessAnalyzerAnyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

func accessAnalyzerFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func accessAnalyzerCloneMap(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
