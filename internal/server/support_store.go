package server

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type supportStore struct {
	mu sync.Mutex

	nextCaseID          int64
	nextCommunicationID int64
	nextAttachmentID    int64
	nextAttachmentSetID int64

	cases                map[string]map[string]any
	communicationsByCase map[string][]map[string]any
	attachmentsByID      map[string]map[string]any
	checksByID           map[string]map[string]any
	checkRefreshByID     map[string]map[string]any
}

func newSupportStore() *supportStore {
	now := time.Now().UTC()
	caseID := "case-000001"
	commID := "comm-000001"
	attachmentID := "attachment-000001"
	checkID := "ta-check-ec2-01"

	s := &supportStore{
		nextCaseID:          2,
		nextCommunicationID: 2,
		nextAttachmentID:    2,
		nextAttachmentSetID: 2,
		cases: map[string]map[string]any{
			caseID: {
				"caseId":       caseID,
				"displayId":    "10000000001",
				"subject":      "Stackyard seeded support case",
				"status":       "opened",
				"serviceCode":  "amazon-ec2",
				"categoryCode": "general-guidance",
				"severityCode": "low",
				"submittedBy":  "stackyard@example.com",
				"timeCreated":  now.Add(-10 * time.Minute).Format(time.RFC3339),
				"recentCommunications": map[string]any{
					"communications": []any{
						map[string]any{
							"caseId":        caseID,
							"body":          "Initial seeded communication",
							"submittedBy":   "stackyard@example.com",
							"timeCreated":   now.Add(-10 * time.Minute).Format(time.RFC3339),
							"attachmentSet": []any{},
						},
					},
					"nextToken": "",
				},
			},
		},
		communicationsByCase: map[string][]map[string]any{
			caseID: {
				{
					"communicationId":   commID,
					"caseId":            caseID,
					"body":              "Initial seeded communication",
					"submittedBy":       "stackyard@example.com",
					"timeCreated":       now.Add(-10 * time.Minute).Format(time.RFC3339),
					"attachmentSet":     []any{},
					"displayToCustomer": true,
				},
			},
		},
		attachmentsByID: map[string]map[string]any{
			attachmentID: {
				"fileName": "stackyard.txt",
				"data":     base64.StdEncoding.EncodeToString([]byte("stackyard")),
			},
		},
		checksByID: map[string]map[string]any{
			checkID: {
				"id":          checkID,
				"name":        "EC2 Reserved Instance Optimization",
				"description": "Checks for underutilized EC2 reserved instances",
				"category":    "cost_optimizing",
				"metadata":    []any{"Region", "Instance Type", "Estimated Monthly Savings"},
			},
		},
		checkRefreshByID: map[string]map[string]any{
			checkID: {
				"checkId":                    checkID,
				"status":                     "success",
				"millisUntilNextRefreshable": int64(300000),
			},
		},
	}
	return s
}

func (s *supportStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	language := supportPayloadString(payload, "language", "en")
	if strings.TrimSpace(language) == "" {
		language = "en"
	}

	switch action {
	case "AddAttachmentsToSet":
		attachments := supportPayloadSlice(payload, "attachments")
		added := make([]any, 0, len(attachments))
		for _, item := range attachments {
			asMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			attachmentID := s.nextAttachmentIdentifierLocked()
			fileName := supportPayloadString(asMap, "fileName", "attachment.txt")
			data := supportPayloadString(asMap, "data", "")
			if data == "" {
				data = base64.StdEncoding.EncodeToString([]byte("stackyard"))
			}
			s.attachmentsByID[attachmentID] = map[string]any{
				"fileName": fileName,
				"data":     data,
			}
			added = append(added, attachmentID)
		}
		if len(added) == 0 {
			attachmentID := s.nextAttachmentIdentifierLocked()
			s.attachmentsByID[attachmentID] = map[string]any{
				"fileName": "attachment.txt",
				"data":     base64.StdEncoding.EncodeToString([]byte("stackyard")),
			}
			added = append(added, attachmentID)
		}
		return map[string]any{
			"attachmentSetId": supportAttachmentSetID(s.nextAttachmentSetID),
			"expiryTime":      now.Add(7 * 24 * time.Hour).Format(time.RFC3339),
			"attachmentIds":   added,
		}

	case "AddCommunicationToCase":
		caseID := supportPayloadString(payload, "caseId", s.firstCaseIDLocked())
		cse := s.ensureCaseLocked(caseID)
		communicationID := s.nextCommunicationIdentifierLocked()
		communication := map[string]any{
			"communicationId":   communicationID,
			"caseId":            caseID,
			"body":              supportPayloadString(payload, "communicationBody", "Stackyard communication update"),
			"submittedBy":       "stackyard@example.com",
			"timeCreated":       now.Format(time.RFC3339),
			"displayToCustomer": true,
			"attachmentSet":     supportPayloadSlice(payload, "attachmentSetId"),
		}
		s.communicationsByCase[caseID] = append(s.communicationsByCase[caseID], supportCloneMap(communication))
		cse["recentCommunications"] = map[string]any{
			"communications": s.supportCommunicationListLocked(caseID),
			"nextToken":      "",
		}
		return map[string]any{"result": true}

	case "CreateCase":
		caseID := s.nextCaseIdentifierLocked()
		cse := map[string]any{
			"caseId":       caseID,
			"displayId":    fmt.Sprintf("1%010d", s.nextCaseID+1000),
			"subject":      supportPayloadString(payload, "subject", "Stackyard support case"),
			"status":       "opened",
			"serviceCode":  supportPayloadString(payload, "serviceCode", "amazon-ec2"),
			"categoryCode": supportPayloadString(payload, "categoryCode", "general-guidance"),
			"severityCode": supportPayloadString(payload, "severityCode", "low"),
			"submittedBy":  "stackyard@example.com",
			"timeCreated":  now.Format(time.RFC3339),
		}
		s.cases[caseID] = supportCloneMap(cse)
		initialComm := map[string]any{
			"communicationId":   s.nextCommunicationIdentifierLocked(),
			"caseId":            caseID,
			"body":              supportPayloadString(payload, "communicationBody", "Case created"),
			"submittedBy":       "stackyard@example.com",
			"timeCreated":       now.Format(time.RFC3339),
			"displayToCustomer": true,
			"attachmentSet":     []any{},
		}
		s.communicationsByCase[caseID] = []map[string]any{supportCloneMap(initialComm)}
		s.cases[caseID]["recentCommunications"] = map[string]any{
			"communications": s.supportCommunicationListLocked(caseID),
			"nextToken":      "",
		}
		return map[string]any{"caseId": caseID}

	case "DescribeAttachment":
		attachmentID := supportPayloadString(payload, "attachmentId", s.firstAttachmentIDLocked())
		attachment := s.ensureAttachmentLocked(attachmentID)
		return map[string]any{"attachment": supportCloneMap(attachment)}

	case "DescribeCases":
		return map[string]any{
			"cases":     supportSortedCaseList(s.cases),
			"nextToken": "",
		}

	case "DescribeCommunications":
		caseID := supportPayloadString(payload, "caseId", s.firstCaseIDLocked())
		s.ensureCaseLocked(caseID)
		return map[string]any{
			"communications": s.supportCommunicationListLocked(caseID),
			"nextToken":      "",
		}

	case "DescribeCreateCaseOptions":
		serviceCode := supportPayloadString(payload, "serviceCode", "amazon-ec2")
		categoryCode := supportPayloadString(payload, "categoryCode", "general-guidance")
		issueType := supportPayloadString(payload, "issueType", "technical")
		return map[string]any{
			"communicationTypes": []any{"web", "chat"},
			"languageAvailability": []any{
				map[string]any{"language": language, "display": "Available"},
			},
			"issueTypes": []any{
				map[string]any{"code": issueType, "name": strings.Title(issueType)},
			},
			"supportedHours": map[string]any{
				"type": "business",
				"daysOfWeek": []any{
					"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY",
				},
				"startTime": "09:00",
				"endTime":   "17:00",
				"timeZone":  "UTC",
			},
			"serviceCode":  serviceCode,
			"categoryCode": categoryCode,
		}

	case "DescribeIssueTypes":
		return map[string]any{
			"issueTypes": []any{
				map[string]any{"code": "technical", "name": "Technical support"},
				map[string]any{"code": "account", "name": "Account and billing"},
			},
		}

	case "DescribeServices":
		return map[string]any{
			"services": []any{
				map[string]any{
					"code": "amazon-ec2",
					"name": "Amazon Elastic Compute Cloud",
					"categories": []any{
						map[string]any{"code": "general-guidance", "name": "General guidance"},
						map[string]any{"code": "instance-issue", "name": "Instance issue"},
					},
				},
				map[string]any{
					"code": "amazon-s3",
					"name": "Amazon Simple Storage Service",
					"categories": []any{
						map[string]any{"code": "general-guidance", "name": "General guidance"},
					},
				},
			},
		}

	case "DescribeSeverityLevels":
		return map[string]any{
			"severityLevels": []any{
				map[string]any{"code": "low", "name": "Low"},
				map[string]any{"code": "normal", "name": "Normal"},
				map[string]any{"code": "high", "name": "High"},
				map[string]any{"code": "urgent", "name": "Urgent"},
			},
		}

	case "DescribeSupportedLanguages":
		return map[string]any{
			"supportedLanguages": []any{
				map[string]any{"code": "en", "display": "English"},
				map[string]any{"code": "ja", "display": "Japanese"},
			},
		}

	case "DescribeTrustedAdvisorCheckRefreshStatuses":
		checkIDs := supportPayloadStringSlice(payload, "checkIds")
		if len(checkIDs) == 0 {
			checkIDs = []string{s.firstCheckIDLocked()}
		}
		statuses := make([]any, 0, len(checkIDs))
		for _, checkID := range checkIDs {
			checkID = strings.TrimSpace(checkID)
			if checkID == "" {
				continue
			}
			status := s.ensureCheckRefreshLocked(checkID)
			statuses = append(statuses, supportCloneMap(status))
		}
		if len(statuses) == 0 {
			statuses = append(statuses, supportCloneMap(s.ensureCheckRefreshLocked(s.firstCheckIDLocked())))
		}
		return map[string]any{"statuses": statuses}

	case "DescribeTrustedAdvisorCheckResult":
		checkID := supportPayloadString(payload, "checkId", s.firstCheckIDLocked())
		check := s.ensureCheckLocked(checkID)
		return map[string]any{
			"result": map[string]any{
				"checkId":          checkID,
				"timestamp":        now.Format(time.RFC3339),
				"status":           "ok",
				"resourcesSummary": map[string]any{"resourcesProcessed": 1, "resourcesFlagged": 0, "resourcesIgnored": 0, "resourcesSuppressed": 0},
				"categorySpecificSummary": map[string]any{
					"costOptimizing": map[string]any{
						"estimatedMonthlySavings":        12.34,
						"estimatedPercentMonthlySavings": 3.21,
					},
				},
				"flaggedResources": []any{
					map[string]any{
						"status":       "ok",
						"region":       "us-east-1",
						"resourceId":   "i-00000000000000001",
						"isSuppressed": false,
						"metadata":     supportCloneSlice(check["metadata"]),
					},
				},
			},
		}

	case "DescribeTrustedAdvisorCheckSummaries":
		checkIDs := supportPayloadStringSlice(payload, "checkIds")
		if len(checkIDs) == 0 {
			checkIDs = []string{s.firstCheckIDLocked()}
		}
		summaries := make([]any, 0, len(checkIDs))
		for _, checkID := range checkIDs {
			check := s.ensureCheckLocked(checkID)
			status := s.ensureCheckRefreshLocked(checkID)
			summaries = append(summaries, map[string]any{
				"checkId":             supportPayloadString(check, "id", checkID),
				"timestamp":           now.Format(time.RFC3339),
				"status":              supportPayloadString(status, "status", "success"),
				"hasFlaggedResources": false,
				"resourcesSummary": map[string]any{
					"resourcesProcessed":  1,
					"resourcesFlagged":    0,
					"resourcesIgnored":    0,
					"resourcesSuppressed": 0,
				},
				"categorySpecificSummary": map[string]any{
					"costOptimizing": map[string]any{
						"estimatedMonthlySavings":        12.34,
						"estimatedPercentMonthlySavings": 3.21,
					},
				},
			})
		}
		return map[string]any{"summaries": summaries}

	case "RefreshTrustedAdvisorCheck":
		checkID := supportPayloadString(payload, "checkId", s.firstCheckIDLocked())
		status := s.ensureCheckRefreshLocked(checkID)
		status["status"] = "enqueued"
		status["millisUntilNextRefreshable"] = int64(120000)
		return map[string]any{"status": supportCloneMap(status)}

	case "ResolveCase":
		caseID := supportPayloadString(payload, "caseId", s.firstCaseIDLocked())
		cse := s.ensureCaseLocked(caseID)
		cse["status"] = "resolved"
		return map[string]any{"initialCaseStatus": "opened", "finalCaseStatus": "resolved"}
	}

	return map[string]any{}
}

func (s *supportStore) nextCaseIdentifierLocked() string {
	id := s.nextCaseID
	s.nextCaseID++
	return fmt.Sprintf("case-%06d", id)
}

func (s *supportStore) nextCommunicationIdentifierLocked() string {
	id := s.nextCommunicationID
	s.nextCommunicationID++
	return fmt.Sprintf("comm-%06d", id)
}

func (s *supportStore) nextAttachmentIdentifierLocked() string {
	id := s.nextAttachmentID
	s.nextAttachmentID++
	return fmt.Sprintf("attachment-%06d", id)
}

func supportAttachmentSetID(id int64) string {
	return fmt.Sprintf("attachment-set-%06d", id)
}

func (s *supportStore) firstCaseIDLocked() string {
	return supportFirstMapKey(s.cases)
}

func (s *supportStore) firstAttachmentIDLocked() string {
	return supportFirstMapKey(s.attachmentsByID)
}

func (s *supportStore) firstCheckIDLocked() string {
	return supportFirstMapKey(s.checksByID)
}

func (s *supportStore) ensureCaseLocked(caseID string) map[string]any {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		caseID = s.firstCaseIDLocked()
	}
	if cse := s.cases[caseID]; cse != nil {
		return cse
	}
	now := time.Now().UTC()
	cse := map[string]any{
		"caseId":       caseID,
		"displayId":    fmt.Sprintf("1%010d", s.nextCaseID+1000),
		"subject":      "Stackyard synthetic support case",
		"status":       "opened",
		"serviceCode":  "amazon-ec2",
		"categoryCode": "general-guidance",
		"severityCode": "low",
		"submittedBy":  "stackyard@example.com",
		"timeCreated":  now.Format(time.RFC3339),
	}
	s.cases[caseID] = cse
	if _, ok := s.communicationsByCase[caseID]; !ok {
		s.communicationsByCase[caseID] = []map[string]any{}
	}
	return cse
}

func (s *supportStore) ensureAttachmentLocked(attachmentID string) map[string]any {
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID == "" {
		attachmentID = s.firstAttachmentIDLocked()
	}
	if attachment := s.attachmentsByID[attachmentID]; attachment != nil {
		return attachment
	}
	attachment := map[string]any{
		"fileName": "attachment.txt",
		"data":     base64.StdEncoding.EncodeToString([]byte("stackyard")),
	}
	s.attachmentsByID[attachmentID] = attachment
	return attachment
}

func (s *supportStore) ensureCheckLocked(checkID string) map[string]any {
	checkID = strings.TrimSpace(checkID)
	if checkID == "" {
		checkID = s.firstCheckIDLocked()
	}
	if check := s.checksByID[checkID]; check != nil {
		return check
	}
	check := map[string]any{
		"id":          checkID,
		"name":        "Stackyard Trusted Advisor Check",
		"description": "Synthetic Trusted Advisor check result",
		"category":    "best_practices",
		"metadata":    []any{"Region", "Resource", "Status"},
	}
	s.checksByID[checkID] = check
	return check
}

func (s *supportStore) ensureCheckRefreshLocked(checkID string) map[string]any {
	checkID = strings.TrimSpace(checkID)
	if checkID == "" {
		checkID = s.firstCheckIDLocked()
	}
	if status := s.checkRefreshByID[checkID]; status != nil {
		return status
	}
	status := map[string]any{
		"checkId":                    checkID,
		"status":                     "success",
		"millisUntilNextRefreshable": int64(300000),
	}
	s.checkRefreshByID[checkID] = status
	return status
}

func (s *supportStore) supportCommunicationListLocked(caseID string) []any {
	communications := s.communicationsByCase[caseID]
	if len(communications) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(communications))
	for _, communication := range communications {
		out = append(out, supportCloneMap(communication))
	}
	return out
}

func supportSortedCaseList(cases map[string]map[string]any) []any {
	keys := make([]string, 0, len(cases))
	for caseID := range cases {
		keys = append(keys, caseID)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, caseID := range keys {
		out = append(out, supportCloneMap(cases[caseID]))
	}
	return out
}

func supportFirstMapKey[V any](in map[string]V) string {
	if len(in) == 0 {
		return ""
	}
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func supportPayloadValue(payload map[string]any, key string) (any, bool) {
	for k, value := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return value, true
		}
	}
	return nil, false
}

func supportPayloadString(payload map[string]any, key, fallback string) string {
	value, ok := supportPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return fallback
		}
		return typed
	default:
		asString := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if asString == "" || asString == "%!v(<nil>)" {
			return fallback
		}
		return asString
	}
}

func supportPayloadSlice(payload map[string]any, key string) []any {
	value, ok := supportPayloadValue(payload, key)
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func supportPayloadStringSlice(payload map[string]any, key string) []string {
	raw := supportPayloadSlice(payload, key)
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		asString := strings.TrimSpace(fmt.Sprintf("%v", item))
		if asString != "" && asString != "%!v(<nil>)" {
			out = append(out, asString)
		}
	}
	return out
}

func supportCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = supportCloneAny(value)
	}
	return out
}

func supportCloneSlice(in any) []any {
	switch typed := in.(type) {
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, supportCloneAny(item))
		}
		return out
	default:
		return []any{}
	}
}

func supportCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return supportCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, supportCloneAny(item))
		}
		return out
	default:
		return typed
	}
}
