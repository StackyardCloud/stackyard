package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type serviceQuotasStore struct {
	mu                 sync.Mutex
	nextID             int64
	templateAssociated bool
	autoManagement     string
	reportStatus       string
	reportCreatedAt    string
	resourceTags       map[string]map[string]string
	requestedQuotas    map[string]map[string]any
	templateRequests   map[string]map[string]any
}

func newServiceQuotasStore() *serviceQuotasStore {
	seedArn := "arn:aws:servicequotas:us-east-1:123456789012:quota/ec2/L-1216C47A"
	return &serviceQuotasStore{
		nextID:             1,
		templateAssociated: true,
		autoManagement:     "ENABLED",
		reportStatus:       "NOT_STARTED",
		resourceTags: map[string]map[string]string{
			seedArn: {"stackyard": "true"},
		},
		requestedQuotas:  map[string]map[string]any{},
		templateRequests: map[string]map[string]any{},
	}
}

func (s *serviceQuotasStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	serviceCode := serviceQuotasPayloadString(payload, "ServiceCode", "ec2")
	quotaCode := serviceQuotasPayloadString(payload, "QuotaCode", "L-1216C47A")
	resourceArn := serviceQuotasPayloadString(payload, "ResourceARN", serviceQuotaARN(serviceCode, quotaCode))

	s.ensureSeedQuotaLocked(serviceCode, quotaCode)

	switch action {
	case "AssociateServiceQuotaTemplate":
		s.templateAssociated = true
		return map[string]any{"ServiceQuotaTemplateAssociationStatus": "ASSOCIATED"}
	case "CreateSupportCase":
		return map[string]any{"CaseId": fmt.Sprintf("case-%06d", s.nextIDLocked())}
	case "DeleteServiceQuotaIncreaseRequestFromTemplate":
		key := serviceTemplateKey(serviceCode, quotaCode)
		delete(s.templateRequests, key)
		return map[string]any{}
	case "DisassociateServiceQuotaTemplate":
		s.templateAssociated = false
		return map[string]any{"ServiceQuotaTemplateAssociationStatus": "DISASSOCIATED"}
	case "GetAWSDefaultServiceQuota":
		return map[string]any{"Quota": s.defaultQuotaLocked(serviceCode, quotaCode)}
	case "GetAssociationForServiceQuotaTemplate":
		status := "DISASSOCIATED"
		if s.templateAssociated {
			status = "ASSOCIATED"
		}
		return map[string]any{"ServiceQuotaTemplateAssociationStatus": status}
	case "GetAutoManagementConfiguration":
		return map[string]any{"AutoManagementConfiguration": map[string]any{"ServiceCode": serviceCode, "AutoManagementState": s.autoManagement}}
	case "GetQuotaUtilizationReport":
		return map[string]any{"QuotaUtilizationReport": map[string]any{"QuotaUtilizationReportStatus": s.reportStatus, "Created": s.reportCreatedAt, "LastUpdated": now}}
	case "GetRequestedServiceQuotaChange":
		id := serviceQuotasPayloadString(payload, "RequestId", "")
		if id == "" {
			id = serviceQuotasPayloadString(payload, "RequestedServiceQuotaChangeId", "")
		}
		if id == "" {
			id = s.firstRequestedQuotaIDLocked()
		}
		if rq, ok := s.requestedQuotas[id]; ok {
			return map[string]any{"RequestedQuota": serviceQuotasCloneMap(rq)}
		}
		return map[string]any{"RequestedQuota": s.requestedQuotaSkeletonLocked(serviceCode, quotaCode, now)}
	case "GetServiceQuota":
		return map[string]any{"Quota": s.appliedQuotaLocked(serviceCode, quotaCode)}
	case "GetServiceQuotaIncreaseRequestFromTemplate":
		key := serviceTemplateKey(serviceCode, quotaCode)
		if rq, ok := s.templateRequests[key]; ok {
			return map[string]any{"ServiceQuotaIncreaseRequestInTemplate": serviceQuotasCloneMap(rq)}
		}
		return map[string]any{"ServiceQuotaIncreaseRequestInTemplate": s.templateRequestSkeletonLocked(serviceCode, quotaCode, now)}
	case "ListAWSDefaultServiceQuotas":
		return map[string]any{"Quotas": []any{s.defaultQuotaLocked(serviceCode, quotaCode)}, "NextToken": ""}
	case "ListRequestedServiceQuotaChangeHistory":
		return map[string]any{"RequestedQuotas": s.sortedRequestedQuotasLocked(), "NextToken": ""}
	case "ListRequestedServiceQuotaChangeHistoryByQuota":
		filtered := []any{}
		for _, item := range s.sortedRequestedQuotasLocked() {
			m, _ := item.(map[string]any)
			if strings.EqualFold(fmt.Sprintf("%v", m["ServiceCode"]), serviceCode) && strings.EqualFold(fmt.Sprintf("%v", m["QuotaCode"]), quotaCode) {
				filtered = append(filtered, serviceQuotasCloneMap(m))
			}
		}
		if len(filtered) == 0 {
			filtered = append(filtered, s.requestedQuotaSkeletonLocked(serviceCode, quotaCode, now))
		}
		return map[string]any{"RequestedQuotas": filtered, "NextToken": ""}
	case "ListServiceQuotaIncreaseRequestsInTemplate":
		return map[string]any{"ServiceQuotaIncreaseRequestInTemplateList": s.sortedTemplateRequestsLocked(), "NextToken": ""}
	case "ListServiceQuotas":
		return map[string]any{"Quotas": []any{s.appliedQuotaLocked(serviceCode, quotaCode)}, "NextToken": ""}
	case "ListServices":
		return map[string]any{"Services": []any{
			map[string]any{"ServiceCode": "ec2", "ServiceName": "Amazon Elastic Compute Cloud (Amazon EC2)"},
			map[string]any{"ServiceCode": "s3", "ServiceName": "Amazon Simple Storage Service (Amazon S3)"},
		}, "NextToken": ""}
	case "ListTagsForResource":
		tags := s.tagsListLocked(resourceArn)
		return map[string]any{"Tags": tags}
	case "PutServiceQuotaIncreaseRequestIntoTemplate":
		desired := serviceQuotasPayloadFloat(payload, "DesiredValue", 1)
		entry := map[string]any{
			"ServiceCode":  serviceCode,
			"QuotaCode":    quotaCode,
			"DesiredValue": desired,
			"AwsRegion":    "us-east-1",
			"Unit":         "Count",
			"GlobalQuota":  false,
		}
		s.templateRequests[serviceTemplateKey(serviceCode, quotaCode)] = serviceQuotasCloneMap(entry)
		return map[string]any{"ServiceQuotaIncreaseRequestInTemplate": entry}
	case "RequestServiceQuotaIncrease":
		desired := serviceQuotasPayloadFloat(payload, "DesiredValue", 1)
		id := fmt.Sprintf("sqr-%06d", s.nextIDLocked())
		req := map[string]any{
			"Id":           id,
			"CaseId":       fmt.Sprintf("case-%06d", s.nextIDLocked()),
			"ServiceCode":  serviceCode,
			"ServiceName":  serviceDisplayName(serviceCode),
			"QuotaCode":    quotaCode,
			"QuotaName":    serviceQuotaName(quotaCode),
			"DesiredValue": desired,
			"Status":       "PENDING",
			"Created":      now,
			"LastUpdated":  now,
			"Unit":         "Count",
			"GlobalQuota":  false,
		}
		s.requestedQuotas[id] = serviceQuotasCloneMap(req)
		return map[string]any{"RequestedQuota": req}
	case "StartAutoManagement":
		s.autoManagement = "ENABLED"
		return map[string]any{"AutoManagementConfiguration": map[string]any{"ServiceCode": serviceCode, "AutoManagementState": s.autoManagement}}
	case "StartQuotaUtilizationReport":
		s.reportStatus = "IN_PROGRESS"
		s.reportCreatedAt = now
		return map[string]any{"QuotaUtilizationReport": map[string]any{"QuotaUtilizationReportStatus": s.reportStatus, "Created": s.reportCreatedAt, "LastUpdated": now}}
	case "StopAutoManagement":
		s.autoManagement = "DISABLED"
		return map[string]any{"AutoManagementConfiguration": map[string]any{"ServiceCode": serviceCode, "AutoManagementState": s.autoManagement}}
	case "TagResource":
		s.applyTagsLocked(resourceArn, payload)
		return map[string]any{}
	case "UntagResource":
		s.removeTagsLocked(resourceArn, payload)
		return map[string]any{}
	case "UpdateAutoManagement":
		state := strings.ToUpper(serviceQuotasPayloadString(payload, "AutoManagement", s.autoManagement))
		if state == "" || state == "%!V(<NIL>)" {
			state = s.autoManagement
		}
		s.autoManagement = state
		return map[string]any{"AutoManagementConfiguration": map[string]any{"ServiceCode": serviceCode, "AutoManagementState": s.autoManagement}}
	}

	return map[string]any{}
}

func (s *serviceQuotasStore) ensureSeedQuotaLocked(serviceCode, quotaCode string) {
	arn := serviceQuotaARN(serviceCode, quotaCode)
	if _, ok := s.resourceTags[arn]; !ok {
		s.resourceTags[arn] = map[string]string{"stackyard": "true"}
	}
}

func (s *serviceQuotasStore) defaultQuotaLocked(serviceCode, quotaCode string) map[string]any {
	return map[string]any{
		"ServiceCode":         serviceCode,
		"ServiceName":         serviceDisplayName(serviceCode),
		"QuotaArn":            serviceQuotaARN(serviceCode, quotaCode),
		"QuotaCode":           quotaCode,
		"QuotaName":           serviceQuotaName(quotaCode),
		"Value":               1000.0,
		"Unit":                "Count",
		"Adjustable":          true,
		"GlobalQuota":         false,
		"UsageMetric":         map[string]any{"MetricNamespace": "AWS/Usage", "MetricName": "ResourceCount", "MetricDimensions": map[string]any{"Service": serviceCode}, "MetricStatisticRecommendation": "Maximum"},
		"Period":              map[string]any{"PeriodValue": 1, "PeriodUnit": "DAY"},
		"ErrorReason":         map[string]any{"ErrorCode": "DEPENDENCY_ACCESS_DENIED_ERROR", "ErrorMessage": ""},
		"QuotaAppliedAtLevel": "ACCOUNT",
	}
}

func (s *serviceQuotasStore) appliedQuotaLocked(serviceCode, quotaCode string) map[string]any {
	quota := s.defaultQuotaLocked(serviceCode, quotaCode)
	quota["Value"] = 1200.0
	return quota
}

func (s *serviceQuotasStore) requestedQuotaSkeletonLocked(serviceCode, quotaCode, now string) map[string]any {
	return map[string]any{
		"Id":           fmt.Sprintf("sqr-%06d", s.nextIDLocked()),
		"CaseId":       fmt.Sprintf("case-%06d", s.nextIDLocked()),
		"ServiceCode":  serviceCode,
		"ServiceName":  serviceDisplayName(serviceCode),
		"QuotaCode":    quotaCode,
		"QuotaName":    serviceQuotaName(quotaCode),
		"DesiredValue": 1100.0,
		"Status":       "PENDING",
		"Created":      now,
		"LastUpdated":  now,
		"Unit":         "Count",
		"GlobalQuota":  false,
	}
}

func (s *serviceQuotasStore) templateRequestSkeletonLocked(serviceCode, quotaCode, _ string) map[string]any {
	return map[string]any{
		"ServiceCode":  serviceCode,
		"QuotaCode":    quotaCode,
		"DesiredValue": 1100.0,
		"AwsRegion":    "us-east-1",
		"Unit":         "Count",
		"GlobalQuota":  false,
	}
}

func (s *serviceQuotasStore) sortedRequestedQuotasLocked() []any {
	keys := make([]string, 0, len(s.requestedQuotas))
	for id := range s.requestedQuotas {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		out = append(out, serviceQuotasCloneMap(s.requestedQuotas[id]))
	}
	if len(out) == 0 {
		out = append(out, s.requestedQuotaSkeletonLocked("ec2", "L-1216C47A", time.Now().UTC().Format(time.RFC3339)))
	}
	return out
}

func (s *serviceQuotasStore) sortedTemplateRequestsLocked() []any {
	keys := make([]string, 0, len(s.templateRequests))
	for key := range s.templateRequests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, serviceQuotasCloneMap(s.templateRequests[key]))
	}
	if len(out) == 0 {
		out = append(out, s.templateRequestSkeletonLocked("ec2", "L-1216C47A", time.Now().UTC().Format(time.RFC3339)))
	}
	return out
}

func (s *serviceQuotasStore) firstRequestedQuotaIDLocked() string {
	keys := make([]string, 0, len(s.requestedQuotas))
	for id := range s.requestedQuotas {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func (s *serviceQuotasStore) applyTagsLocked(resourceArn string, payload map[string]any) {
	if s.resourceTags[resourceArn] == nil {
		s.resourceTags[resourceArn] = map[string]string{}
	}
	tagsValue, ok := serviceQuotasPayloadValue(payload, "Tags")
	if !ok {
		return
	}
	if m, ok := tagsValue.(map[string]any); ok {
		for k, v := range m {
			key := strings.TrimSpace(k)
			if key == "" {
				continue
			}
			s.resourceTags[resourceArn][key] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
		return
	}
	if arr, ok := tagsValue.([]any); ok {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := serviceQuotasPayloadString(m, "Key", "")
			if key == "" {
				continue
			}
			s.resourceTags[resourceArn][key] = serviceQuotasPayloadString(m, "Value", "")
		}
	}
}

func (s *serviceQuotasStore) removeTagsLocked(resourceArn string, payload map[string]any) {
	tagKeysValue, ok := serviceQuotasPayloadValue(payload, "TagKeys")
	if !ok {
		return
	}
	arr, ok := tagKeysValue.([]any)
	if !ok {
		return
	}
	for _, item := range arr {
		key := strings.TrimSpace(fmt.Sprintf("%v", item))
		if key == "" {
			continue
		}
		delete(s.resourceTags[resourceArn], key)
	}
}

func (s *serviceQuotasStore) tagsListLocked(resourceArn string) []any {
	tags := s.resourceTags[resourceArn]
	if tags == nil {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}

func (s *serviceQuotasStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func serviceTemplateKey(serviceCode, quotaCode string) string {
	return strings.ToLower(strings.TrimSpace(serviceCode)) + "|" + strings.ToLower(strings.TrimSpace(quotaCode))
}

func serviceQuotaARN(serviceCode, quotaCode string) string {
	return fmt.Sprintf("arn:aws:servicequotas:us-east-1:123456789012:quota/%s/%s", strings.ToLower(strings.TrimSpace(serviceCode)), strings.TrimSpace(quotaCode))
}

func serviceDisplayName(serviceCode string) string {
	switch strings.ToLower(strings.TrimSpace(serviceCode)) {
	case "ec2":
		return "Amazon Elastic Compute Cloud (Amazon EC2)"
	case "s3":
		return "Amazon Simple Storage Service (Amazon S3)"
	default:
		if serviceCode == "" {
			return "Unknown Service"
		}
		return strings.ToUpper(serviceCode)
	}
}

func serviceQuotaName(quotaCode string) string {
	if strings.EqualFold(strings.TrimSpace(quotaCode), "L-1216C47A") {
		return "Running On-Demand Standard (A, C, D, H, I, M, R, T, Z) instances"
	}
	if quotaCode == "" {
		return "Service quota"
	}
	return quotaCode
}

func serviceQuotasPayloadString(payload map[string]any, key, fallback string) string {
	v, ok := serviceQuotasPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "%!v(<nil>)" {
		return fallback
	}
	return s
}

func serviceQuotasPayloadFloat(payload map[string]any, key string, fallback float64) float64 {
	v, ok := serviceQuotasPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", v))
	if s == "" || s == "%!v(<nil>)" {
		return fallback
	}
	var out float64
	if _, err := fmt.Sscanf(s, "%f", &out); err != nil {
		return fallback
	}
	return out
}

func serviceQuotasPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return v, true
		}
	}
	return nil, false
}

func serviceQuotasCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
