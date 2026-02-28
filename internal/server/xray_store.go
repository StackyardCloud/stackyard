package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	xrayDefaultRegion    = "us-east-1"
	xrayDefaultAccountID = "123456789012"
)

type xrayTraceRetrieval struct {
	Token     string
	TraceIDs  []string
	Status    string
	StartTime float64
	EndTime   float64
	UpdatedAt float64
}

type xrayStore struct {
	mu sync.Mutex

	nextID int64

	groups                  map[string]map[string]any
	samplingRules           map[string]map[string]any
	resourcePolicies        map[string]map[string]any
	tags                    map[string]map[string]string
	traces                  map[string][]map[string]any
	retrievals              map[string]*xrayTraceRetrieval
	encryptionConfig        map[string]any
	traceSegmentDestination map[string]any
	indexingRules           map[string]map[string]any
	insights                map[string]map[string]any
}

func newXRayStore() *xrayStore {
	now := xrayEpoch(time.Now().UTC())
	s := &xrayStore{
		nextID:                  2,
		groups:                  map[string]map[string]any{},
		samplingRules:           map[string]map[string]any{},
		resourcePolicies:        map[string]map[string]any{},
		tags:                    map[string]map[string]string{},
		traces:                  map[string][]map[string]any{},
		retrievals:              map[string]*xrayTraceRetrieval{},
		encryptionConfig:        map[string]any{"Type": "NONE", "Status": "ACTIVE", "KeyId": ""},
		traceSegmentDestination: map[string]any{"Destination": "XRay", "Status": "ACTIVE"},
		indexingRules:           map[string]map[string]any{},
		insights:                map[string]map[string]any{},
	}

	group := s.ensureGroupLocked("stackyard-group", now)
	s.ensureSamplingRuleLocked("stackyard-default-rule", now)
	s.ensureIndexingRuleLocked("Default", now)
	s.ensureTraceLocked("1-58406520-a006649127e371903a2de979", now)
	s.ensureInsightLocked("insight-000001", now)

	resourceARN := xrayAnyString(group["GroupARN"])
	if resourceARN != "" {
		s.ensureTagsLocked(resourceARN)["service"] = "xray"
		s.ensureTagsLocked(resourceARN)["seed"] = "true"
	}

	s.resourcePolicies["stackyard-default-policy"] = map[string]any{
		"PolicyName":       "stackyard-default-policy",
		"PolicyDocument":   "{}",
		"PolicyRevisionId": "1",
		"LastUpdatedTime":  now,
	}
	return s
}

func (s *xrayStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := xrayEpoch(time.Now().UTC())

	groupName := xrayPayloadString(payload, []string{"GroupName", "groupName"}, "stackyard-group")
	groupARN := xrayPayloadString(payload, []string{"GroupARN", "GroupArn", "groupArn"}, "")
	ruleName := xrayPayloadString(payload, []string{"RuleName", "SamplingRuleName", "ruleName"}, "stackyard-default-rule")
	resourceARN := xrayPayloadString(payload, []string{"ResourceARN", "ResourceArn", "resourceArn"}, xrayGroupARN(groupName))
	insightID := xrayPayloadString(payload, []string{"InsightId", "insightId"}, "insight-000001")
	retrievalToken := xrayPayloadString(payload, []string{"RetrievalToken", "retrievalToken"}, "")
	startTime := xrayPayloadTime(payload, []string{"StartTime", "startTime"}, now-900)
	endTime := xrayPayloadTime(payload, []string{"EndTime", "endTime"}, now)

	if groupARN != "" {
		if name := xrayGroupNameFromARN(groupARN); name != "" {
			groupName = name
		}
	}

	switch action {
	case "CreateGroup":
		if name := xrayPayloadString(payload, []string{"GroupName", "groupName"}, ""); name != "" {
			groupName = name
		} else {
			groupName = fmt.Sprintf("stackyard-group-%06d", s.nextIDLocked())
		}
		group := s.ensureGroupLocked(groupName, now)
		if expr := xrayPayloadString(payload, []string{"FilterExpression", "filterExpression"}, ""); expr != "" {
			group["FilterExpression"] = expr
		}
		if cfg, ok := xrayPayloadMap(payload, []string{"InsightsConfiguration", "insightsConfiguration"}); ok {
			group["InsightsConfiguration"] = xrayCloneMap(cfg)
		}
		group["LastUpdatedTime"] = now
		return map[string]any{"Group": xrayCloneMap(group)}

	case "GetGroup":
		group := s.ensureGroupLocked(groupName, now)
		return map[string]any{"Group": xrayCloneMap(group)}

	case "GetGroups":
		items := []any{}
		for _, name := range s.sortedGroupNamesLocked() {
			items = append(items, xrayCloneMap(s.groups[name]))
		}
		return map[string]any{"Groups": items, "NextToken": ""}

	case "UpdateGroup":
		group := s.ensureGroupLocked(groupName, now)
		if expr := xrayPayloadString(payload, []string{"FilterExpression", "filterExpression"}, ""); expr != "" {
			group["FilterExpression"] = expr
		}
		if cfg, ok := xrayPayloadMap(payload, []string{"InsightsConfiguration", "insightsConfiguration"}); ok {
			group["InsightsConfiguration"] = xrayCloneMap(cfg)
		}
		group["LastUpdatedTime"] = now
		return map[string]any{"Group": xrayCloneMap(group)}

	case "DeleteGroup":
		delete(s.tags, xrayGroupARN(groupName))
		delete(s.groups, groupName)
		return map[string]any{}

	case "CreateSamplingRule":
		in, _ := xrayPayloadMap(payload, []string{"SamplingRule", "samplingRule"})
		if name := xrayStringFromMap(in, []string{"RuleName", "ruleName"}, ""); name != "" {
			ruleName = name
		} else {
			ruleName = fmt.Sprintf("stackyard-rule-%06d", s.nextIDLocked())
		}
		rule := s.ensureSamplingRuleLocked(ruleName, now)
		s.mergeSamplingRuleLocked(rule, in, now)
		return map[string]any{"SamplingRuleRecord": s.samplingRuleRecordLocked(rule)}

	case "GetSamplingRules":
		records := []any{}
		for _, name := range s.sortedSamplingRuleNamesLocked() {
			records = append(records, s.samplingRuleRecordLocked(s.samplingRules[name]))
		}
		return map[string]any{"SamplingRuleRecords": records, "NextToken": ""}

	case "UpdateSamplingRule":
		update, _ := xrayPayloadMap(payload, []string{"SamplingRuleUpdate", "samplingRuleUpdate"})
		if name := xrayStringFromMap(update, []string{"RuleName", "ruleName"}, ""); name != "" {
			ruleName = name
		}
		rule := s.ensureSamplingRuleLocked(ruleName, now)
		s.mergeSamplingRuleLocked(rule, update, now)
		return map[string]any{"SamplingRuleRecord": s.samplingRuleRecordLocked(rule)}

	case "DeleteSamplingRule":
		rule := s.ensureSamplingRuleLocked(ruleName, now)
		delete(s.samplingRules, ruleName)
		return map[string]any{"SamplingRuleRecord": s.samplingRuleRecordLocked(rule)}

	case "GetSamplingStatisticSummaries":
		items := []any{}
		for _, name := range s.sortedSamplingRuleNamesLocked() {
			items = append(items, map[string]any{
				"RuleName":     name,
				"Timestamp":    now,
				"RequestCount": 1.0,
				"BorrowCount":  0.0,
				"SampledCount": 1.0,
			})
		}
		return map[string]any{"SamplingStatisticSummaries": items, "NextToken": ""}

	case "GetSamplingTargets":
		stats := xrayPayloadSlice(payload, "SamplingStatisticsDocuments")
		targets := []any{}
		for _, item := range stats {
			doc, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := xrayStringFromMap(doc, []string{"RuleName", "ruleName"}, ruleName)
			rule := s.ensureSamplingRuleLocked(name, now)
			targets = append(targets, map[string]any{
				"RuleName":          name,
				"FixedRate":         xrayAnyFloat(rule["FixedRate"], 0.05),
				"ReservoirQuota":    1.0,
				"ReservoirQuotaTTL": now + 60,
				"Interval":          10,
			})
		}
		if len(targets) == 0 {
			rule := s.ensureSamplingRuleLocked(ruleName, now)
			targets = append(targets, map[string]any{
				"RuleName":          ruleName,
				"FixedRate":         xrayAnyFloat(rule["FixedRate"], 0.05),
				"ReservoirQuota":    1.0,
				"ReservoirQuotaTTL": now + 60,
				"Interval":          10,
			})
		}
		return map[string]any{
			"SamplingTargetDocuments": targets,
			"LastRuleModification":    now,
			"UnprocessedStatistics":   []any{},
		}

	case "PutTelemetryRecords":
		return map[string]any{}

	case "PutTraceSegments":
		segments := xrayPayloadStringSlice(payload, "TraceSegmentDocuments")
		if len(segments) == 0 {
			segments = []string{xrayDefaultTraceSegmentDocument(now)}
		}
		unprocessed := []any{}
		for _, raw := range segments {
			traceID, segmentID, serviceName := xrayParseTraceSegment(raw)
			if traceID == "" {
				unprocessed = append(unprocessed, map[string]any{
					"Id":        "",
					"ErrorCode": "InvalidTraceSegment",
					"Message":   "trace_id is required",
				})
				continue
			}
			if segmentID == "" {
				segmentID = fmt.Sprintf("%016x", s.nextIDLocked())
			}
			if strings.TrimSpace(serviceName) == "" {
				serviceName = "stackyard-service"
			}
			s.traces[traceID] = append(s.traces[traceID], map[string]any{
				"Id":       segmentID,
				"Document": raw,
			})
			s.ensureInsightLocked(insightID, now)
			s.ensureTraceSummaryDefaultsLocked(traceID, serviceName, now)
		}
		return map[string]any{"UnprocessedTraceSegments": unprocessed}

	case "BatchGetTraces":
		traceIDs := xrayPayloadStringSlice(payload, "TraceIds")
		if len(traceIDs) == 0 {
			traceIDs = s.sortedTraceIDsLocked()
		}
		traces := []any{}
		unprocessed := []string{}
		for _, id := range traceIDs {
			segments, ok := s.traces[id]
			if !ok {
				unprocessed = append(unprocessed, id)
				continue
			}
			traces = append(traces, map[string]any{
				"Id":            id,
				"Duration":      1.0,
				"LimitExceeded": false,
				"Segments":      xrayCloneSliceOfMaps(segments),
			})
		}
		return map[string]any{
			"Traces":              traces,
			"UnprocessedTraceIds": unprocessed,
			"NextToken":           "",
		}

	case "GetTraceGraph":
		ids := xrayPayloadStringSlice(payload, "TraceIds")
		return map[string]any{
			"Services":  s.traceGraphServicesLocked(ids, startTime, endTime),
			"NextToken": "",
		}

	case "GetServiceGraph":
		return map[string]any{
			"StartTime":                startTime,
			"EndTime":                  endTime,
			"Services":                 s.traceGraphServicesLocked(nil, startTime, endTime),
			"ContainsOldGroupVersions": false,
			"NextToken":                "",
		}

	case "GetTimeSeriesServiceStatistics":
		return map[string]any{
			"TimeSeriesServiceStatistics": []any{
				map[string]any{
					"Timestamp": xrayEpoch(time.Now().UTC()),
					"EdgeSummaryStatistics": map[string]any{
						"OkCount":    1.0,
						"TotalCount": 1.0,
					},
					"ServiceSummaryStatistics": map[string]any{
						"OkCount":    1.0,
						"TotalCount": 1.0,
					},
					"ResponseTimeHistogram": []any{},
					"FaultStatistics":       map[string]any{"TotalCount": 0.0},
					"ErrorStatistics":       map[string]any{"TotalCount": 0.0},
					"ThrottleStatistics":    map[string]any{"TotalCount": 0.0},
				},
			},
			"ContainsOldGroupVersions": false,
			"NextToken":                "",
		}

	case "GetTraceSummaries":
		traceIDs := s.sortedTraceIDsLocked()
		if len(traceIDs) == 0 {
			traceIDs = []string{"1-58406520-a006649127e371903a2de979"}
		}
		summaries := make([]any, 0, len(traceIDs))
		for _, id := range traceIDs {
			summaries = append(summaries, map[string]any{
				"Id":           id,
				"Duration":     1.0,
				"ResponseTime": 0.05,
				"HasFault":     false,
				"HasError":     false,
				"HasThrottle":  false,
				"IsPartial":    false,
				"EntryPoint": map[string]any{
					"Name": "stackyard-service",
					"Type": "AWS::EC2::Instance",
				},
				"ServiceIds": []any{
					map[string]any{
						"Name":  "stackyard-service",
						"Names": []any{"stackyard-service"},
						"Type":  "AWS::EC2::Instance",
					},
				},
				"ResourceARNs": []any{},
				"Http": map[string]any{
					"HttpURL":    "https://example.com/stackyard",
					"HttpMethod": "GET",
					"HttpStatus": 200,
					"ClientIp":   "127.0.0.1",
					"UserAgent":  "stackyard",
				},
				"Annotations": map[string]any{},
				"Users":       []any{},
			})
		}
		return map[string]any{
			"TraceSummaries":       summaries,
			"ApproximateTime":      endTime,
			"TracesProcessedCount": float64(len(summaries)),
			"NextToken":            "",
		}

	case "StartTraceRetrieval":
		traceIDs := xrayPayloadStringSlice(payload, "TraceIds")
		if len(traceIDs) == 0 {
			traceIDs = s.sortedTraceIDsLocked()
		}
		token := fmt.Sprintf("retrieval-%06d", s.nextIDLocked())
		s.retrievals[token] = &xrayTraceRetrieval{
			Token:     token,
			TraceIDs:  append([]string(nil), traceIDs...),
			Status:    "SUCCEEDED",
			StartTime: startTime,
			EndTime:   endTime,
			UpdatedAt: now,
		}
		return map[string]any{"RetrievalToken": token}

	case "CancelTraceRetrieval":
		retrieval := s.ensureRetrievalLocked(retrievalToken, now)
		retrieval.Status = "CANCELED"
		retrieval.UpdatedAt = now
		return map[string]any{}

	case "ListRetrievedTraces":
		retrieval := s.ensureRetrievalLocked(retrievalToken, now)
		traces := []any{}
		for _, id := range retrieval.TraceIDs {
			traces = append(traces, map[string]any{
				"Id": id,
				"Spans": []any{
					map[string]any{
						"Id":        "span-000001",
						"Name":      "stackyard-span",
						"StartTime": retrieval.StartTime,
						"EndTime":   retrieval.EndTime,
					},
				},
			})
		}
		return map[string]any{
			"RetrievalStatus": retrieval.Status,
			"TraceFormat":     "XRAY",
			"Traces":          traces,
			"NextToken":       "",
		}

	case "GetRetrievedTracesGraph":
		retrieval := s.ensureRetrievalLocked(retrievalToken, now)
		return map[string]any{
			"RetrievalStatus": retrieval.Status,
			"Services":        s.traceGraphServicesLocked(retrieval.TraceIDs, retrieval.StartTime, retrieval.EndTime),
			"NextToken":       "",
		}

	case "GetInsight":
		insight := s.ensureInsightLocked(insightID, now)
		return map[string]any{"Insight": xrayCloneMap(insight)}

	case "GetInsightEvents":
		insight := s.ensureInsightLocked(insightID, now)
		event := map[string]any{
			"Timestamp": now,
			"Summary":   "Synthetic insight event",
			"ClientRequestImpactStatistics": map[string]any{
				"FaultCount": 0.0,
				"OkCount":    1.0,
				"TotalCount": 1.0,
			},
			"RootCauseServiceRequestImpactStatistics": map[string]any{
				"FaultCount": 0.0,
				"OkCount":    1.0,
				"TotalCount": 1.0,
			},
			"TopAnomalousServices": []any{
				map[string]any{
					"ServiceId": map[string]any{"Name": "stackyard-service", "Type": "AWS::EC2::Instance"},
					"Percent":   100.0,
				},
			},
		}
		_ = insight
		return map[string]any{"InsightEvents": []any{event}, "NextToken": ""}

	case "GetInsightImpactGraph":
		insight := s.ensureInsightLocked(insightID, now)
		return map[string]any{
			"InsightId":             xrayAnyString(insight["InsightId"]),
			"StartTime":             startTime,
			"EndTime":               endTime,
			"ServiceGraphStartTime": startTime,
			"ServiceGraphEndTime":   endTime,
			"Services":              s.traceGraphServicesLocked(nil, startTime, endTime),
			"NextToken":             "",
		}

	case "GetInsightSummaries":
		items := []any{}
		for _, id := range s.sortedInsightIDsLocked() {
			insight := s.ensureInsightLocked(id, now)
			items = append(items, map[string]any{
				"InsightId":          xrayAnyString(insight["InsightId"]),
				"GroupARN":           xrayAnyString(insight["GroupARN"]),
				"GroupName":          xrayAnyString(insight["GroupName"]),
				"RootCauseServiceId": insight["RootCauseServiceId"],
				"Categories":         insight["Categories"],
				"State":              insight["State"],
				"StartTime":          insight["StartTime"],
				"LastUpdateTime":     now,
				"ClientRequestImpactStatistics": map[string]any{
					"FaultCount": 0.0,
					"OkCount":    1.0,
					"TotalCount": 1.0,
				},
			})
		}
		return map[string]any{"InsightSummaries": items, "NextToken": ""}

	case "GetEncryptionConfig":
		return map[string]any{"EncryptionConfig": xrayCloneMap(s.encryptionConfig)}

	case "PutEncryptionConfig":
		encType := strings.ToUpper(xrayPayloadString(payload, []string{"Type", "type"}, xrayAnyString(s.encryptionConfig["Type"])))
		if encType == "" {
			encType = "NONE"
		}
		s.encryptionConfig["Type"] = encType
		s.encryptionConfig["Status"] = "ACTIVE"
		if keyID := xrayPayloadString(payload, []string{"KeyId", "KMSKeyId", "keyId"}, ""); keyID != "" {
			s.encryptionConfig["KeyId"] = keyID
		} else if encType == "NONE" {
			s.encryptionConfig["KeyId"] = ""
		}
		return map[string]any{"EncryptionConfig": xrayCloneMap(s.encryptionConfig)}

	case "GetTraceSegmentDestination":
		return map[string]any{
			"Destination": xrayAnyString(s.traceSegmentDestination["Destination"]),
			"Status":      xrayAnyString(s.traceSegmentDestination["Status"]),
		}

	case "UpdateTraceSegmentDestination":
		destination := xrayPayloadString(payload, []string{"Destination", "destination"}, xrayAnyString(s.traceSegmentDestination["Destination"]))
		if destination == "" {
			destination = "XRay"
		}
		s.traceSegmentDestination["Destination"] = destination
		s.traceSegmentDestination["Status"] = "ACTIVE"
		return map[string]any{
			"Destination": xrayAnyString(s.traceSegmentDestination["Destination"]),
			"Status":      xrayAnyString(s.traceSegmentDestination["Status"]),
		}

	case "GetIndexingRules":
		items := []any{}
		for _, name := range s.sortedIndexingRuleNamesLocked() {
			items = append(items, xrayCloneMap(s.indexingRules[name]))
		}
		return map[string]any{"IndexingRules": items, "NextToken": ""}

	case "UpdateIndexingRule":
		name := xrayPayloadString(payload, []string{"Name", "name"}, "Default")
		rule := s.ensureIndexingRuleLocked(name, now)
		if inputRule, ok := xrayPayloadMap(payload, []string{"Rule", "rule"}); ok {
			rule["Rule"] = xrayCloneMap(inputRule)
		}
		rule["ModifiedAt"] = now
		return map[string]any{"IndexingRule": xrayCloneMap(rule)}

	case "PutResourcePolicy":
		policyName := xrayPayloadString(payload, []string{"PolicyName", "policyName"}, "stackyard-policy")
		policyDoc := xrayPayloadString(payload, []string{"PolicyDocument", "policyDocument"}, "{}")
		policy := s.resourcePolicies[policyName]
		revision := 1
		if policy != nil {
			revision = int(xrayAnyFloat(policy["PolicyRevisionId"], 1)) + 1
		}
		policy = map[string]any{
			"PolicyName":       policyName,
			"PolicyDocument":   policyDoc,
			"PolicyRevisionId": strconv.Itoa(revision),
			"LastUpdatedTime":  now,
		}
		s.resourcePolicies[policyName] = policy
		return map[string]any{"ResourcePolicy": xrayCloneMap(policy)}

	case "ListResourcePolicies":
		items := []any{}
		for _, name := range s.sortedPolicyNamesLocked() {
			items = append(items, xrayCloneMap(s.resourcePolicies[name]))
		}
		return map[string]any{"ResourcePolicies": items, "NextToken": ""}

	case "DeleteResourcePolicy":
		policyName := xrayPayloadString(payload, []string{"PolicyName", "policyName"}, "stackyard-policy")
		delete(s.resourcePolicies, policyName)
		return map[string]any{}

	case "TagResource":
		for k, v := range xrayExtractTags(payload) {
			s.ensureTagsLocked(resourceARN)[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		for _, key := range xrayExtractTagKeys(payload, query) {
			delete(s.ensureTagsLocked(resourceARN), key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{
			"Tags":      xrayTagListFromMap(s.ensureTagsLocked(resourceARN)),
			"NextToken": "",
		}

	default:
		return map[string]any{}
	}
}

func (s *xrayStore) ensureGroupLocked(name string, now float64) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-group"
	}
	if existing, ok := s.groups[name]; ok {
		return existing
	}

	group := map[string]any{
		"GroupName":        name,
		"GroupARN":         xrayGroupARN(name),
		// Use Go string literal quoting to avoid unsafe manual quote interpolation.
		"FilterExpression": "service(" + strconv.Quote(name) + ")",
		"InsightsConfiguration": map[string]any{
			"InsightsEnabled":      true,
			"NotificationsEnabled": false,
		},
		"LastUpdatedTime": now,
	}
	s.groups[name] = group
	return group
}

func (s *xrayStore) ensureSamplingRuleLocked(name string, now float64) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-default-rule"
	}
	if existing, ok := s.samplingRules[name]; ok {
		samplingRuleEnsureDefaults(existing, name, now)
		return existing
	}

	rule := map[string]any{
		"RuleName":      name,
		"ResourceARN":   "*",
		"Priority":      1,
		"FixedRate":     0.05,
		"ReservoirSize": 1,
		"ServiceName":   "*",
		"ServiceType":   "*",
		"Host":          "*",
		"HTTPMethod":    "*",
		"URLPath":       "*",
		"Version":       1,
		"Attributes":    map[string]any{},
		"RuleARN":       xraySamplingRuleARN(name),
		"CreatedAt":     now,
		"ModifiedAt":    now,
	}
	s.samplingRules[name] = rule
	return rule
}

func (s *xrayStore) mergeSamplingRuleLocked(rule map[string]any, incoming map[string]any, now float64) {
	if rule == nil {
		return
	}
	for k, v := range incoming {
		if k == "" {
			continue
		}
		rule[k] = v
	}
	name := xrayAnyString(rule["RuleName"])
	if name == "" {
		name = "stackyard-default-rule"
		rule["RuleName"] = name
	}
	samplingRuleEnsureDefaults(rule, name, now)
	rule["ModifiedAt"] = now
}

func (s *xrayStore) samplingRuleRecordLocked(rule map[string]any) map[string]any {
	if rule == nil {
		return map[string]any{}
	}
	clone := xrayCloneMap(rule)
	created := xrayAnyFloat(clone["CreatedAt"], xrayEpoch(time.Now().UTC()))
	modified := xrayAnyFloat(clone["ModifiedAt"], created)
	delete(clone, "CreatedAt")
	delete(clone, "ModifiedAt")
	return map[string]any{
		"SamplingRule": clone,
		"CreatedAt":    created,
		"ModifiedAt":   modified,
	}
}

func (s *xrayStore) ensureTraceLocked(traceID string, now float64) []map[string]any {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		traceID = "1-58406520-a006649127e371903a2de979"
	}
	if segments, ok := s.traces[traceID]; ok && len(segments) > 0 {
		return segments
	}

	doc := xrayDefaultTraceSegmentDocument(now)
	segments := []map[string]any{
		{
			"Id":       "1111111111111111",
			"Document": doc,
		},
	}
	s.traces[traceID] = segments
	return segments
}

func (s *xrayStore) ensureTraceSummaryDefaultsLocked(traceID, serviceName string, now float64) {
	_ = now
	if _, ok := s.traces[traceID]; !ok {
		s.ensureTraceLocked(traceID, xrayEpoch(time.Now().UTC()))
	}
	if strings.TrimSpace(serviceName) == "" {
		return
	}
}

func (s *xrayStore) ensureRetrievalLocked(token string, now float64) *xrayTraceRetrieval {
	token = strings.TrimSpace(token)
	if token == "" {
		if len(s.retrievals) > 0 {
			token = s.sortedRetrievalTokensLocked()[0]
		} else {
			token = "retrieval-000001"
		}
	}
	if existing, ok := s.retrievals[token]; ok {
		if existing.Status == "" {
			existing.Status = "SUCCEEDED"
		}
		return existing
	}
	traceIDs := s.sortedTraceIDsLocked()
	if len(traceIDs) == 0 {
		traceIDs = []string{"1-58406520-a006649127e371903a2de979"}
	}
	out := &xrayTraceRetrieval{
		Token:     token,
		TraceIDs:  append([]string(nil), traceIDs...),
		Status:    "SUCCEEDED",
		StartTime: now - 900,
		EndTime:   now,
		UpdatedAt: now,
	}
	s.retrievals[token] = out
	return out
}

func (s *xrayStore) ensureIndexingRuleLocked(name string, now float64) map[string]any {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Default"
	}
	if existing, ok := s.indexingRules[name]; ok {
		return existing
	}
	rule := map[string]any{
		"Name": name,
		"Rule": map[string]any{
			"Probabilistic": map[string]any{
				"DesiredSamplingPercentage": 5.0,
			},
		},
		"ModifiedAt": now,
	}
	s.indexingRules[name] = rule
	return rule
}

func (s *xrayStore) ensureInsightLocked(id string, now float64) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "insight-000001"
	}
	if existing, ok := s.insights[id]; ok {
		return existing
	}
	group := s.ensureGroupLocked("stackyard-group", now)
	insight := map[string]any{
		"InsightId": id,
		"GroupARN":  xrayAnyString(group["GroupARN"]),
		"GroupName": xrayAnyString(group["GroupName"]),
		"RootCauseServiceId": map[string]any{
			"Name": "stackyard-service",
			"Type": "AWS::EC2::Instance",
		},
		"Categories": []any{"FAULT"},
		"State":      "ACTIVE",
		"StartTime":  now - 900,
		"EndTime":    now,
	}
	s.insights[id] = insight
	return insight
}

func (s *xrayStore) traceGraphServicesLocked(traceIDs []string, startTime, endTime float64) []any {
	if len(traceIDs) == 0 {
		traceIDs = s.sortedTraceIDsLocked()
	}
	if len(traceIDs) == 0 {
		traceIDs = []string{"1-58406520-a006649127e371903a2de979"}
	}

	services := make([]any, 0, len(traceIDs))
	for i, traceID := range traceIDs {
		name := "stackyard-service"
		if segments, ok := s.traces[traceID]; ok && len(segments) > 0 {
			if raw := xrayAnyString(segments[0]["Document"]); raw != "" {
				if _, _, parsedName := xrayParseTraceSegment(raw); strings.TrimSpace(parsedName) != "" {
					name = parsedName
				}
			}
		}
		services = append(services, map[string]any{
			"ReferenceId": i + 1,
			"Name":        name,
			"Type":        "AWS::EC2::Instance",
			"State":       "active",
			"StartTime":   startTime,
			"EndTime":     endTime,
			"Edges":       []any{},
			"SummaryStatistics": map[string]any{
				"OkCount":    1.0,
				"TotalCount": 1.0,
			},
			"ResponseTimeHistogram": []any{},
		})
	}
	return services
}

func (s *xrayStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = xrayGroupARN("stackyard-group")
	}
	if tags, ok := s.tags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.tags[resourceARN] = tags
	return tags
}

func (s *xrayStore) sortedGroupNamesLocked() []string {
	keys := make([]string, 0, len(s.groups))
	for key := range s.groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) sortedSamplingRuleNamesLocked() []string {
	keys := make([]string, 0, len(s.samplingRules))
	for key := range s.samplingRules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) sortedPolicyNamesLocked() []string {
	keys := make([]string, 0, len(s.resourcePolicies))
	for key := range s.resourcePolicies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) sortedTraceIDsLocked() []string {
	keys := make([]string, 0, len(s.traces))
	for key := range s.traces {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) sortedRetrievalTokensLocked() []string {
	keys := make([]string, 0, len(s.retrievals))
	for key := range s.retrievals {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) sortedIndexingRuleNamesLocked() []string {
	keys := make([]string, 0, len(s.indexingRules))
	for key := range s.indexingRules {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) sortedInsightIDsLocked() []string {
	keys := make([]string, 0, len(s.insights))
	for key := range s.insights {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *xrayStore) nextIDLocked() int64 {
	id := s.nextID
	if id <= 0 {
		id = 1
	}
	s.nextID = id + 1
	return id
}

func xrayGroupARN(groupName string) string {
	name := strings.TrimSpace(groupName)
	if name == "" {
		name = "stackyard-group"
	}
	return fmt.Sprintf("arn:aws:xray:%s:%s:group/%s", xrayDefaultRegion, xrayDefaultAccountID, name)
}

func xraySamplingRuleARN(ruleName string) string {
	name := strings.TrimSpace(ruleName)
	if name == "" {
		name = "stackyard-default-rule"
	}
	return fmt.Sprintf("arn:aws:xray:%s:%s:sampling-rule/%s", xrayDefaultRegion, xrayDefaultAccountID, name)
}

func xrayGroupNameFromARN(arn string) string {
	trimmed := strings.TrimSpace(arn)
	if trimmed == "" {
		return ""
	}
	const needle = ":group/"
	if idx := strings.LastIndex(trimmed, needle); idx >= 0 && idx+len(needle) < len(trimmed) {
		return strings.TrimSpace(trimmed[idx+len(needle):])
	}
	return ""
}

func xrayEpoch(t time.Time) float64 {
	return float64(t.Unix())
}

func xrayPayloadAny(payload map[string]any, keys []string) (any, bool) {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := payload[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func xrayPayloadMap(payload map[string]any, keys []string) (map[string]any, bool) {
	value, ok := xrayPayloadAny(payload, keys)
	if !ok || value == nil {
		return nil, false
	}
	out, ok := value.(map[string]any)
	return out, ok
}

func xrayPayloadString(payload map[string]any, keys []string, def string) string {
	value, ok := xrayPayloadAny(payload, keys)
	if !ok {
		return def
	}
	text := strings.TrimSpace(xrayAnyString(value))
	if text == "" {
		return def
	}
	return text
}

func xrayPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
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
			text := strings.TrimSpace(xrayAnyString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	case string:
		cleaned := strings.TrimSpace(typed)
		if cleaned == "" {
			return nil
		}
		if strings.Contains(cleaned, ",") {
			parts := strings.Split(cleaned, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					out = append(out, part)
				}
			}
			return out
		}
		return []string{cleaned}
	default:
		return nil
	}
}

func xrayPayloadSlice(payload map[string]any, key string) []any {
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func xrayPayloadTime(payload map[string]any, keys []string, def float64) float64 {
	value, ok := xrayPayloadAny(payload, keys)
	if !ok || value == nil {
		return def
	}
	parsed := xrayAnyFloat(value, def)
	if parsed <= 0 {
		return def
	}
	return parsed
}

func xrayAnyString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		raw, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(raw)
	}
}

func xrayAnyFloat(value any, def float64) float64 {
	switch typed := value.(type) {
	case nil:
		return def
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint64:
		return float64(typed)
	case json.Number:
		if f, err := typed.Float64(); err == nil {
			return f
		}
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return parsed
		}
	}
	return def
}

func xrayStringFromMap(in map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if value, ok := in[key]; ok {
			text := strings.TrimSpace(xrayAnyString(value))
			if text != "" {
				return text
			}
		}
	}
	return def
}

func xrayCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func xrayCloneSlice(in []any) []any {
	if in == nil {
		return []any{}
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return []any{}
	}
	out := []any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return []any{}
	}
	return out
}

func xrayCloneSliceOfMaps(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, xrayCloneMap(item))
	}
	return out
}

func xrayExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}

	value, ok := payload["Tags"]
	if !ok || value == nil {
		value = payload["tags"]
	}

	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(xrayAnyString(item))
		}
	case map[string]string:
		for key, item := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(item)
		}
	case []any:
		for _, entry := range typed {
			em, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			key := xrayStringFromMap(em, []string{"Key", "key"}, "")
			if key == "" {
				continue
			}
			out[key] = xrayStringFromMap(em, []string{"Value", "value"}, "")
		}
	}
	return out
}

func xrayExtractTagKeys(payload map[string]any, query url.Values) []string {
	out := []string{}
	seen := map[string]struct{}{}

	appendKey := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	for _, key := range xrayPayloadStringSlice(payload, "TagKeys") {
		appendKey(key)
	}
	for _, key := range xrayPayloadStringSlice(payload, "tagKeys") {
		appendKey(key)
	}
	for _, key := range query["tagKeys"] {
		appendKey(key)
	}
	return out
}

func xrayTagListFromMap(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}

func xrayDefaultTraceSegmentDocument(now float64) string {
	start := now - 1
	if start <= 0 {
		start = now
	}
	return fmt.Sprintf(
		`{"trace_id":"1-58406520-a006649127e371903a2de979","id":"1111111111111111","name":"stackyard-service","start_time":%.0f,"end_time":%.0f}`,
		start,
		now,
	)
}

func xrayParseTraceSegment(raw string) (traceID, segmentID, serviceName string) {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", "", ""
	}
	traceID = xrayStringFromMap(payload, []string{"trace_id", "TraceId"}, "")
	segmentID = xrayStringFromMap(payload, []string{"id", "Id"}, "")
	serviceName = xrayStringFromMap(payload, []string{"name", "Name"}, "")
	return traceID, segmentID, serviceName
}

func samplingRuleEnsureDefaults(rule map[string]any, name string, now float64) {
	if rule == nil {
		return
	}
	if strings.TrimSpace(name) == "" {
		name = "stackyard-default-rule"
	}
	rule["RuleName"] = name
	if strings.TrimSpace(xrayAnyString(rule["ResourceARN"])) == "" {
		rule["ResourceARN"] = "*"
	}
	if xrayAnyFloat(rule["Priority"], 0) == 0 {
		rule["Priority"] = 1
	}
	if xrayAnyFloat(rule["FixedRate"], 0) == 0 {
		rule["FixedRate"] = 0.05
	}
	if xrayAnyFloat(rule["ReservoirSize"], 0) == 0 {
		rule["ReservoirSize"] = 1
	}
	if strings.TrimSpace(xrayAnyString(rule["ServiceName"])) == "" {
		rule["ServiceName"] = "*"
	}
	if strings.TrimSpace(xrayAnyString(rule["ServiceType"])) == "" {
		rule["ServiceType"] = "*"
	}
	if strings.TrimSpace(xrayAnyString(rule["Host"])) == "" {
		rule["Host"] = "*"
	}
	if strings.TrimSpace(xrayAnyString(rule["HTTPMethod"])) == "" {
		rule["HTTPMethod"] = "*"
	}
	if strings.TrimSpace(xrayAnyString(rule["URLPath"])) == "" {
		rule["URLPath"] = "*"
	}
	if xrayAnyFloat(rule["Version"], 0) == 0 {
		rule["Version"] = 1
	}
	if _, ok := rule["Attributes"].(map[string]any); !ok {
		rule["Attributes"] = map[string]any{}
	}
	if strings.TrimSpace(xrayAnyString(rule["RuleARN"])) == "" {
		rule["RuleARN"] = xraySamplingRuleARN(name)
	}
	if xrayAnyFloat(rule["CreatedAt"], 0) == 0 {
		rule["CreatedAt"] = now
	}
	if xrayAnyFloat(rule["ModifiedAt"], 0) == 0 {
		rule["ModifiedAt"] = now
	}
}
