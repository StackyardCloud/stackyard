package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type auditManagerStore struct {
	mu sync.Mutex

	accountID string
	region    string
	nextID    int64

	accountStatus string

	assessments map[string]map[string]any
	frameworks  map[string]map[string]any
	controls    map[string]map[string]any
	reports     map[string]map[string]any
	delegations map[string]map[string]any
	tags        map[string]map[string]string
}

func newAuditManagerStore() *auditManagerStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &auditManagerStore{
		accountID:     "123456789012",
		region:        "us-east-1",
		nextID:        1,
		accountStatus: "ACTIVE",
		assessments:   map[string]map[string]any{},
		frameworks:    map[string]map[string]any{},
		controls:      map[string]map[string]any{},
		reports:       map[string]map[string]any{},
		delegations:   map[string]map[string]any{},
		tags:          map[string]map[string]string{},
	}

	s.assessments["assessment-000001"] = map[string]any{
		"id":                 "assessment-000001",
		"name":               "stackyard-assessment",
		"awsAccount":         map[string]any{"id": s.accountID},
		"status":             "ACTIVE",
		"creationTime":       now,
		"lastUpdated":        now,
		"assessmentMetadata": map[string]any{"id": "assessment-000001"},
	}
	s.frameworks["framework-000001"] = map[string]any{
		"id":           "framework-000001",
		"name":         "stackyard-framework",
		"type":         "Custom",
		"creationTime": now,
		"lastUpdated":  now,
	}
	s.controls["control-000001"] = map[string]any{
		"id":           "control-000001",
		"name":         "stackyard-control",
		"type":         "Custom",
		"creationTime": now,
		"lastUpdated":  now,
	}
	s.reports["report-000001"] = map[string]any{
		"id":           "report-000001",
		"name":         "stackyard-report",
		"status":       "COMPLETE",
		"creationTime": now,
	}
	s.delegations["delegation-000001"] = map[string]any{
		"id":           "delegation-000001",
		"status":       "IN_PROGRESS",
		"creationTime": now,
	}
	return s
}

func (s *auditManagerStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	assessmentID := auditManagerFirstNonEmpty(
		auditManagerPathParam(pathParams, "assessmentId"),
		auditManagerPayloadString(payload, "assessmentId"),
		"assessment-000001",
	)
	frameworkID := auditManagerFirstNonEmpty(
		auditManagerPathParam(pathParams, "frameworkId"),
		auditManagerPayloadString(payload, "frameworkId"),
		"framework-000001",
	)
	controlID := auditManagerFirstNonEmpty(
		auditManagerPathParam(pathParams, "controlId"),
		auditManagerPayloadString(payload, "controlId"),
		"control-000001",
	)
	reportID := auditManagerFirstNonEmpty(
		auditManagerPathParam(pathParams, "assessmentReportId"),
		auditManagerPayloadString(payload, "assessmentReportId"),
		"report-000001",
	)
	requestID := auditManagerFirstNonEmpty(
		auditManagerPathParam(pathParams, "requestId"),
		auditManagerPayloadString(payload, "requestId"),
		"request-000001",
	)
	resourceARN := auditManagerFirstNonEmpty(
		auditManagerPathParam(pathParams, "resourceArn"),
		auditManagerPayloadString(payload, "resourceArn"),
		fmt.Sprintf("arn:aws:auditmanager:%s:%s:assessment/%s", s.region, s.accountID, assessmentID),
	)

	switch action {
	case "GetAccountStatus":
		return map[string]any{"status": s.accountStatus}
	case "RegisterAccount":
		s.accountStatus = "ACTIVE"
		return map[string]any{"status": s.accountStatus}
	case "DeregisterAccount":
		s.accountStatus = "INACTIVE"
		return map[string]any{"status": s.accountStatus}
	case "RegisterOrganizationAdminAccount", "DeregisterOrganizationAdminAccount":
		return map[string]any{"adminAccountId": s.accountID}
	case "GetOrganizationAdminAccount":
		return map[string]any{"adminAccountId": s.accountID}
	case "GetSettings":
		return map[string]any{"settings": map[string]any{"defaultProcessOwners": []any{}, "kmsKey": ""}}
	case "UpdateSettings":
		return map[string]any{"settings": map[string]any{"lastUpdated": now}}
	case "GetServicesInScope":
		return map[string]any{
			"serviceMetadata": []any{
				map[string]any{"name": "s3"},
				map[string]any{"name": "ec2"},
			},
		}

	case "CreateAssessment":
		id := fmt.Sprintf("assessment-%06d", s.nextID)
		s.nextID++
		item := map[string]any{
			"id":           id,
			"name":         auditManagerFirstNonEmpty(auditManagerPayloadString(payload, "name"), id),
			"status":       "ACTIVE",
			"creationTime": now,
			"lastUpdated":  now,
		}
		s.assessments[id] = item
		return map[string]any{"assessment": auditManagerCloneMap(item)}
	case "GetAssessment":
		return map[string]any{"assessment": auditManagerCloneMap(s.ensureAssessmentLocked(assessmentID, now))}
	case "ListAssessments":
		return map[string]any{"assessmentMetadata": s.listByNameLocked(s.assessments), "nextToken": ""}
	case "UpdateAssessment", "UpdateAssessmentStatus":
		item := s.ensureAssessmentLocked(assessmentID, now)
		item["lastUpdated"] = now
		if action == "UpdateAssessmentStatus" {
			item["status"] = auditManagerFirstNonEmpty(auditManagerPayloadString(payload, "status"), "ACTIVE")
		}
		if v := auditManagerPayloadString(payload, "name"); v != "" {
			item["name"] = v
		}
		return map[string]any{"assessment": auditManagerCloneMap(item)}
	case "DeleteAssessment":
		delete(s.assessments, assessmentID)
		return map[string]any{}

	case "CreateAssessmentFramework":
		id := fmt.Sprintf("framework-%06d", s.nextID)
		s.nextID++
		item := map[string]any{
			"id":           id,
			"name":         auditManagerFirstNonEmpty(auditManagerPayloadString(payload, "name"), id),
			"type":         "Custom",
			"creationTime": now,
			"lastUpdated":  now,
		}
		s.frameworks[id] = item
		return map[string]any{"framework": auditManagerCloneMap(item)}
	case "GetAssessmentFramework":
		return map[string]any{"framework": auditManagerCloneMap(s.ensureFrameworkLocked(frameworkID, now))}
	case "ListAssessmentFrameworks":
		return map[string]any{"frameworkMetadataList": s.listByNameLocked(s.frameworks), "nextToken": ""}
	case "UpdateAssessmentFramework":
		item := s.ensureFrameworkLocked(frameworkID, now)
		item["lastUpdated"] = now
		if v := auditManagerPayloadString(payload, "name"); v != "" {
			item["name"] = v
		}
		return map[string]any{"framework": auditManagerCloneMap(item)}
	case "DeleteAssessmentFramework":
		delete(s.frameworks, frameworkID)
		return map[string]any{}

	case "CreateControl":
		id := fmt.Sprintf("control-%06d", s.nextID)
		s.nextID++
		item := map[string]any{
			"id":           id,
			"name":         auditManagerFirstNonEmpty(auditManagerPayloadString(payload, "name"), id),
			"type":         "Custom",
			"creationTime": now,
			"lastUpdated":  now,
		}
		s.controls[id] = item
		return map[string]any{"control": auditManagerCloneMap(item)}
	case "GetControl":
		return map[string]any{"control": auditManagerCloneMap(s.ensureControlLocked(controlID, now))}
	case "ListControls":
		return map[string]any{"controlMetadataList": s.listByNameLocked(s.controls), "nextToken": ""}
	case "UpdateControl":
		item := s.ensureControlLocked(controlID, now)
		item["lastUpdated"] = now
		if v := auditManagerPayloadString(payload, "name"); v != "" {
			item["name"] = v
		}
		return map[string]any{"control": auditManagerCloneMap(item)}
	case "DeleteControl":
		delete(s.controls, controlID)
		return map[string]any{}

	case "CreateAssessmentReport":
		id := fmt.Sprintf("report-%06d", s.nextID)
		s.nextID++
		item := map[string]any{
			"id":           id,
			"assessmentId": assessmentID,
			"status":       "IN_PROGRESS",
			"creationTime": now,
		}
		s.reports[id] = item
		return map[string]any{"assessmentReport": auditManagerCloneMap(item)}
	case "DeleteAssessmentReport":
		delete(s.reports, reportID)
		return map[string]any{}
	case "ListAssessmentReports":
		return map[string]any{"assessmentReports": s.listByNameLocked(s.reports), "nextToken": ""}
	case "GetAssessmentReportUrl":
		return map[string]any{
			"url": fmt.Sprintf("https://example.com/auditmanager/reports/%s", reportID),
		}
	case "ValidateAssessmentReportIntegrity":
		return map[string]any{"signatureValid": true}

	case "BatchCreateDelegationByAssessment":
		return map[string]any{"delegations": s.listByNameLocked(s.delegations), "errors": []any{}}
	case "BatchDeleteDelegationByAssessment":
		return map[string]any{"errors": []any{}}
	case "GetDelegations":
		return map[string]any{"delegations": s.listByNameLocked(s.delegations), "nextToken": ""}

	case "StartAssessmentFrameworkShare":
		return map[string]any{"requestId": requestID}
	case "UpdateAssessmentFrameworkShare":
		return map[string]any{"requestId": requestID}
	case "DeleteAssessmentFrameworkShare":
		return map[string]any{}
	case "ListAssessmentFrameworkShareRequests":
		return map[string]any{
			"assessmentFrameworkShareRequests": []any{
				map[string]any{"requestId": requestID, "status": "ACTIVE"},
			},
			"nextToken": "",
		}

	case "GetInsights":
		return map[string]any{"insights": map[string]any{"activeAssessmentsCount": len(s.assessments)}}
	case "GetInsightsByAssessment":
		return map[string]any{"insights": map[string]any{"assessmentId": assessmentID}}
	case "ListControlDomainInsights":
		return map[string]any{"controlDomainInsights": []any{}, "nextToken": ""}
	case "ListControlDomainInsightsByAssessment":
		return map[string]any{"controlDomainInsights": []any{}, "nextToken": ""}
	case "ListAssessmentControlInsightsByControlDomain":
		return map[string]any{"controlInsightsByAssessment": []any{}, "nextToken": ""}
	case "ListControlInsightsByControlDomain":
		return map[string]any{"controlInsightsMetadata": []any{}, "nextToken": ""}
	case "GetChangeLogs":
		return map[string]any{"changeLogs": []any{}, "nextToken": ""}
	case "ListNotifications":
		return map[string]any{"notifications": []any{}, "nextToken": ""}
	case "ListKeywordsForDataSource":
		return map[string]any{"keywords": []any{"S3_BUCKET", "IAM_ROLE"}}

	case "GetEvidence", "GetEvidenceByEvidenceFolder", "GetEvidenceFolder", "GetEvidenceFoldersByAssessment", "GetEvidenceFoldersByAssessmentControl":
		return map[string]any{"evidence": []any{}, "nextToken": ""}
	case "GetEvidenceFileUploadUrl":
		return map[string]any{"evidenceFileUploadUrl": "https://example.com/auditmanager/upload"}
	case "BatchImportEvidenceToAssessmentControl", "AssociateAssessmentReportEvidenceFolder", "DisassociateAssessmentReportEvidenceFolder", "BatchAssociateAssessmentReportEvidence", "BatchDisassociateAssessmentReportEvidence", "UpdateAssessmentControl", "UpdateAssessmentControlSetStatus":
		return map[string]any{}

	case "TagResource":
		tagSet := s.tags[resourceARN]
		if tagSet == nil {
			tagSet = map[string]string{}
			s.tags[resourceARN] = tagSet
		}
		switch t := payload["tags"].(type) {
		case map[string]any:
			for k, v := range t {
				tagSet[k] = auditManagerAnyString(v)
			}
		case []any:
			for _, item := range t {
				obj, ok := item.(map[string]any)
				if !ok {
					continue
				}
				key := auditManagerFirstNonEmpty(auditManagerPayloadMapString(obj, "key"), auditManagerPayloadMapString(obj, "Key"))
				if key == "" {
					continue
				}
				tagSet[key] = auditManagerFirstNonEmpty(auditManagerPayloadMapString(obj, "value"), auditManagerPayloadMapString(obj, "Value"))
			}
		}
		return map[string]any{}
	case "UntagResource":
		tagSet := s.tags[resourceARN]
		if tagSet == nil {
			return map[string]any{}
		}
		for _, key := range auditManagerStringSlice(payload, "tagKeys") {
			delete(tagSet, key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		return map[string]any{"tags": auditManagerCloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *auditManagerStore) ensureAssessmentLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "assessment-000001"
	}
	if item, ok := s.assessments[id]; ok {
		return item
	}
	item := map[string]any{"id": id, "name": id, "status": "ACTIVE", "creationTime": now, "lastUpdated": now}
	s.assessments[id] = item
	return item
}

func (s *auditManagerStore) ensureFrameworkLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "framework-000001"
	}
	if item, ok := s.frameworks[id]; ok {
		return item
	}
	item := map[string]any{"id": id, "name": id, "type": "Custom", "creationTime": now, "lastUpdated": now}
	s.frameworks[id] = item
	return item
}

func (s *auditManagerStore) ensureControlLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "control-000001"
	}
	if item, ok := s.controls[id]; ok {
		return item
	}
	item := map[string]any{"id": id, "name": id, "type": "Custom", "creationTime": now, "lastUpdated": now}
	s.controls[id] = item
	return item
}

func (s *auditManagerStore) listByNameLocked(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, auditManagerCloneMap(items[key]))
	}
	return out
}

func auditManagerPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if value, ok := pathParams[key]; ok {
		return strings.TrimSpace(value)
	}
	for k, v := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func auditManagerPayloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		return strings.TrimSpace(auditManagerAnyString(value))
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(auditManagerAnyString(v))
		}
	}
	return ""
}

func auditManagerPayloadMapString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	if value, ok := payload[key]; ok {
		return strings.TrimSpace(auditManagerAnyString(value))
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(auditManagerAnyString(v))
		}
	}
	return ""
}

func auditManagerAnyString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func auditManagerFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func auditManagerStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok {
		for k, v := range payload {
			if strings.EqualFold(k, key) {
				raw = v
				ok = true
				break
			}
		}
	}
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			value := strings.TrimSpace(auditManagerAnyString(item))
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(typed))
		for _, value := range typed {
			value = strings.TrimSpace(value)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return nil
		}
		return []string{value}
	default:
		value := strings.TrimSpace(auditManagerAnyString(typed))
		if value == "" {
			return nil
		}
		return []string{value}
	}
}

func auditManagerCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = auditManagerCloneMap(typed)
		case map[string]string:
			m := make(map[string]any, len(typed))
			for k, v := range typed {
				m[k] = v
			}
			out[key] = m
		case []any:
			c := make([]any, len(typed))
			for i, item := range typed {
				if inner, ok := item.(map[string]any); ok {
					c[i] = auditManagerCloneMap(inner)
				} else {
					c[i] = item
				}
			}
			out[key] = c
		default:
			out[key] = typed
		}
	}
	return out
}

func auditManagerCloneStringMap(input map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range input {
		out[key] = value
	}
	return out
}
