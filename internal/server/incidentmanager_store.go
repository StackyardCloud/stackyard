package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type incidentManagerStore struct {
	mu               sync.Mutex
	nextID           int64
	incidents        map[string]map[string]any
	responsePlans    map[string]map[string]any
	timelineEvents   map[string]map[string]any
	contacts         map[string]map[string]any
	contactChannels  map[string]map[string]any
	rotations        map[string]map[string]any
	engagements      map[string]map[string]any
	resourcePolicies map[string]string
	replicationSet   map[string]any
	resourceTags     map[string]map[string]string
}

func newIncidentManagerStore() *incidentManagerStore {
	incidentArn := "arn:aws:ssm-incidents:us-east-1:123456789012:incident-record/ir-000001"
	responsePlanArn := "arn:aws:ssm-incidents:us-east-1:123456789012:response-plan/stackyard-response-plan"
	contactArn := "arn:aws:ssm-contacts:us-east-1:123456789012:contact/stackyard-primary"
	rotationArn := "arn:aws:ssm-contacts:us-east-1:123456789012:rotation/stackyard-rotation"
	channelArn := "arn:aws:ssm-contacts:us-east-1:123456789012:contact-channel/cc-000001"
	now := time.Now().UTC().Format(time.RFC3339)
	return &incidentManagerStore{
		nextID: 2,
		incidents: map[string]map[string]any{
			incidentArn: {
				"Arn":                  incidentArn,
				"Title":                "Seeded Stackyard incident",
				"Status":               "OPEN",
				"Impact":               3,
				"CreationTime":         now,
				"LastModifiedTime":     now,
				"IncidentRecordSource": map[string]any{"CreatedBy": "stackyard"},
			},
		},
		responsePlans: map[string]map[string]any{
			responsePlanArn: {
				"Arn":                   responsePlanArn,
				"Name":                  "stackyard-response-plan",
				"DisplayName":           "Stackyard Response Plan",
				"IncidentTemplate":      map[string]any{"Title": "Stackyard incident template", "Impact": 3},
				"Engagements":           []any{contactArn},
				"Actions":               []any{},
				"ChatChannel":           map[string]any{},
				"IncidentTemplateTitle": "Stackyard incident template",
			},
		},
		timelineEvents: map[string]map[string]any{},
		contacts: map[string]map[string]any{
			contactArn: {
				"ContactArn":  contactArn,
				"Alias":       "stackyard-primary",
				"DisplayName": "Stackyard Primary Contact",
				"Type":        "PERSONAL",
				"Plan":        map[string]any{"Stages": []any{}},
			},
		},
		contactChannels: map[string]map[string]any{
			channelArn: {
				"ContactChannelArn": channelArn,
				"ContactArn":        contactArn,
				"Name":              "primary-email",
				"Type":              "EMAIL",
				"DeliveryAddress":   map[string]any{"SimpleAddress": "stackyard@example.com"},
				"ActivationStatus":  "ACTIVATED",
			},
		},
		rotations: map[string]map[string]any{
			rotationArn: {
				"RotationArn": rotationArn,
				"Name":        "stackyard-rotation",
				"ContactIds":  []any{contactArn},
				"StartTime":   now,
			},
		},
		engagements: map[string]map[string]any{},
		resourcePolicies: map[string]string{
			responsePlanArn: "{\"Version\":\"2012-10-17\",\"Statement\":[]}",
		},
		replicationSet: map[string]any{
			"Arn":               "arn:aws:ssm-incidents::123456789012:replication-set/stackyard",
			"DeletionProtected": true,
			"RegionMap": map[string]any{
				"us-east-1": map[string]any{"SseKmsKeyId": "alias/stackyard-incident-manager"},
			},
		},
		resourceTags: map[string]map[string]string{
			incidentArn:     {"stackyard": "true"},
			responsePlanArn: {"stackyard": "true"},
			contactArn:      {"stackyard": "true"},
		},
	}
}

func (s *incidentManagerStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "ListIncidentRecords":
		return map[string]any{"IncidentRecordSummaries": s.incidentSummariesLocked(), "NextToken": ""}
	case "GetIncidentRecord":
		arn := incidentManagerPayloadString(payload, "Arn", s.firstIncidentArnLocked())
		return map[string]any{"IncidentRecord": s.incidentByArnLocked(arn)}
	case "StartIncident":
		arn := fmt.Sprintf("arn:aws:ssm-incidents:us-east-1:123456789012:incident-record/ir-%06d", s.nextIDLocked())
		record := map[string]any{
			"Arn":              arn,
			"Title":            incidentManagerPayloadString(payload, "Title", "Stackyard incident"),
			"Status":           "OPEN",
			"Impact":           3,
			"CreationTime":     now,
			"LastModifiedTime": now,
		}
		s.incidents[arn] = record
		return map[string]any{"IncidentRecordArn": arn}
	case "UpdateIncidentRecord":
		arn := incidentManagerPayloadString(payload, "Arn", s.firstIncidentArnLocked())
		if record, ok := s.incidents[arn]; ok {
			record["LastModifiedTime"] = now
			if title := incidentManagerPayloadString(payload, "Title", ""); title != "" {
				record["Title"] = title
			}
		}
		return map[string]any{}
	case "DeleteIncidentRecord":
		delete(s.incidents, incidentManagerPayloadString(payload, "Arn", ""))
		return map[string]any{}
	case "CreateResponsePlan":
		name := incidentManagerPayloadString(payload, "Name", fmt.Sprintf("stackyard-response-plan-%06d", s.nextIDLocked()))
		arn := "arn:aws:ssm-incidents:us-east-1:123456789012:response-plan/" + name
		s.responsePlans[arn] = map[string]any{
			"Arn":              arn,
			"Name":             name,
			"DisplayName":      incidentManagerPayloadString(payload, "DisplayName", name),
			"IncidentTemplate": map[string]any{"Title": incidentManagerPayloadString(payload, "DisplayName", name), "Impact": 3},
		}
		return map[string]any{"Arn": arn}
	case "GetResponsePlan":
		arn := incidentManagerPayloadString(payload, "Arn", s.firstResponsePlanArnLocked())
		return map[string]any{"ResponsePlan": incidentManagerCloneMap(s.responsePlans[arn])}
	case "ListResponsePlans":
		return map[string]any{"ResponsePlanSummaries": s.responsePlanSummariesLocked(), "NextToken": ""}
	case "UpdateResponsePlan":
		arn := incidentManagerPayloadString(payload, "Arn", s.firstResponsePlanArnLocked())
		if plan, ok := s.responsePlans[arn]; ok {
			plan["DisplayName"] = incidentManagerPayloadString(payload, "DisplayName", fmt.Sprintf("%v", plan["DisplayName"]))
		}
		return map[string]any{}
	case "DeleteResponsePlan":
		delete(s.responsePlans, incidentManagerPayloadString(payload, "Arn", ""))
		return map[string]any{}
	case "CreateTimelineEvent":
		eventID := fmt.Sprintf("te-%06d", s.nextIDLocked())
		incidentArn := incidentManagerPayloadString(payload, "IncidentRecordArn", s.firstIncidentArnLocked())
		event := map[string]any{
			"EventId":           eventID,
			"IncidentRecordArn": incidentArn,
			"EventTime":         now,
			"EventType":         incidentManagerPayloadString(payload, "EventType", "CUSTOM"),
			"EventData":         incidentManagerCloneMap(payload),
		}
		s.timelineEvents[eventID] = event
		return map[string]any{"EventId": eventID}
	case "GetTimelineEvent":
		eventID := incidentManagerPayloadString(payload, "EventId", s.firstTimelineEventIDLocked())
		if event, ok := s.timelineEvents[eventID]; ok {
			return map[string]any{"Event": incidentManagerCloneMap(event)}
		}
		return map[string]any{"Event": map[string]any{"EventId": eventID, "EventTime": now}}
	case "ListTimelineEvents":
		return map[string]any{"EventSummaries": s.timelineEventSummariesLocked(), "NextToken": ""}
	case "UpdateTimelineEvent":
		eventID := incidentManagerPayloadString(payload, "EventId", s.firstTimelineEventIDLocked())
		if event, ok := s.timelineEvents[eventID]; ok {
			event["EventData"] = incidentManagerCloneMap(payload)
		}
		return map[string]any{}
	case "DeleteTimelineEvent":
		delete(s.timelineEvents, incidentManagerPayloadString(payload, "EventId", ""))
		return map[string]any{}
	case "GetReplicationSet":
		return map[string]any{"ReplicationSet": incidentManagerCloneMap(s.replicationSet)}
	case "ListReplicationSets":
		arn := incidentManagerPayloadString(s.replicationSet, "Arn", "arn:aws:ssm-incidents::123456789012:replication-set/stackyard")
		return map[string]any{"ReplicationSetArns": []any{arn}, "NextToken": ""}
	case "CreateReplicationSet", "UpdateReplicationSet", "UpdateDeletionProtection":
		s.replicationSet["LastModifiedTime"] = now
		return map[string]any{}
	case "DeleteReplicationSet":
		s.replicationSet = map[string]any{"Arn": "arn:aws:ssm-incidents::123456789012:replication-set/deleted"}
		return map[string]any{}
	case "BatchGetIncidentFindings":
		return map[string]any{"Findings": []any{}, "Errors": []any{}}
	case "ListIncidentFindings":
		return map[string]any{"Findings": []any{}, "NextToken": ""}
	case "ListRelatedItems":
		return map[string]any{"RelatedItems": []any{}, "NextToken": ""}
	case "UpdateRelatedItems":
		return map[string]any{}
	case "PutResourcePolicy":
		resourceArn := incidentManagerPayloadString(payload, "ResourceArn", s.firstResponsePlanArnLocked())
		policy := incidentManagerPayloadString(payload, "Policy", "{\"Version\":\"2012-10-17\",\"Statement\":[]}")
		s.resourcePolicies[resourceArn] = policy
		return map[string]any{}
	case "GetResourcePolicies":
		entries := make([]any, 0, len(s.resourcePolicies))
		for resourceArn, policy := range s.resourcePolicies {
			entries = append(entries, map[string]any{"ResourceArn": resourceArn, "Policy": policy})
		}
		sort.Slice(entries, func(i, j int) bool {
			a := entries[i].(map[string]any)
			b := entries[j].(map[string]any)
			return fmt.Sprintf("%v", a["ResourceArn"]) < fmt.Sprintf("%v", b["ResourceArn"])
		})
		return map[string]any{"ResourcePolicies": entries, "NextToken": ""}
	case "DeleteResourcePolicy":
		delete(s.resourcePolicies, incidentManagerPayloadString(payload, "ResourceArn", ""))
		return map[string]any{}
	case "TagResource":
		arn := incidentManagerPayloadString(payload, "ResourceArn", "")
		s.applyTagsLocked(arn, payload)
		return map[string]any{}
	case "UntagResource":
		arn := incidentManagerPayloadString(payload, "ResourceArn", "")
		s.removeTagsLocked(arn, payload)
		return map[string]any{}
	case "ListTagsForResource":
		arn := incidentManagerPayloadString(payload, "ResourceArn", s.firstIncidentArnLocked())
		return map[string]any{"Tags": s.tagsMapLocked(arn)}
	case "CreateContact":
		alias := incidentManagerPayloadString(payload, "Alias", fmt.Sprintf("stackyard-contact-%06d", s.nextIDLocked()))
		arn := "arn:aws:ssm-contacts:us-east-1:123456789012:contact/" + alias
		s.contacts[arn] = map[string]any{
			"ContactArn":  arn,
			"Alias":       alias,
			"DisplayName": incidentManagerPayloadString(payload, "DisplayName", alias),
			"Type":        incidentManagerPayloadString(payload, "Type", "PERSONAL"),
			"Plan":        map[string]any{"Stages": []any{}},
		}
		return map[string]any{"ContactArn": arn}
	case "GetContact":
		arn := incidentManagerPayloadString(payload, "ContactId", s.firstContactArnLocked())
		return map[string]any{"ContactArn": arn, "Alias": incidentManagerPayloadString(s.contacts[arn], "Alias", "stackyard-contact"), "DisplayName": incidentManagerPayloadString(s.contacts[arn], "DisplayName", "Stackyard Contact"), "Type": incidentManagerPayloadString(s.contacts[arn], "Type", "PERSONAL"), "Plan": map[string]any{"Stages": []any{}}}
	case "UpdateContact":
		return map[string]any{}
	case "DeleteContact":
		delete(s.contacts, incidentManagerPayloadString(payload, "ContactId", ""))
		return map[string]any{}
	case "ListContacts":
		items := make([]any, 0, len(s.contacts))
		for _, c := range s.contacts {
			items = append(items, map[string]any{
				"ContactArn":  c["ContactArn"],
				"Alias":       c["Alias"],
				"DisplayName": c["DisplayName"],
				"Type":        c["Type"],
			})
		}
		return map[string]any{"Contacts": items, "NextToken": ""}
	case "CreateContactChannel":
		arn := fmt.Sprintf("arn:aws:ssm-contacts:us-east-1:123456789012:contact-channel/cc-%06d", s.nextIDLocked())
		contactArn := incidentManagerPayloadString(payload, "ContactId", s.firstContactArnLocked())
		s.contactChannels[arn] = map[string]any{
			"ContactChannelArn": arn,
			"ContactArn":        contactArn,
			"Name":              incidentManagerPayloadString(payload, "Name", "channel"),
			"Type":              incidentManagerPayloadString(payload, "Type", "EMAIL"),
			"DeliveryAddress":   map[string]any{"SimpleAddress": "stackyard@example.com"},
			"ActivationStatus":  "NOT_ACTIVATED",
		}
		return map[string]any{"ContactChannelArn": arn}
	case "GetContactChannel":
		arn := incidentManagerPayloadString(payload, "ContactChannelId", s.firstContactChannelArnLocked())
		if ch, ok := s.contactChannels[arn]; ok {
			return incidentManagerCloneMap(ch)
		}
		return map[string]any{"ContactChannelArn": arn}
	case "ListContactChannels":
		contactArn := incidentManagerPayloadString(payload, "ContactId", s.firstContactArnLocked())
		items := []any{}
		for _, ch := range s.contactChannels {
			if fmt.Sprintf("%v", ch["ContactArn"]) == contactArn {
				items = append(items, map[string]any{
					"ContactChannelArn": ch["ContactChannelArn"],
					"Name":              ch["Name"],
					"Type":              ch["Type"],
				})
			}
		}
		return map[string]any{"ContactChannels": items, "NextToken": ""}
	case "ActivateContactChannel":
		arn := incidentManagerPayloadString(payload, "ContactChannelId", s.firstContactChannelArnLocked())
		if ch, ok := s.contactChannels[arn]; ok {
			ch["ActivationStatus"] = "ACTIVATED"
		}
		return map[string]any{}
	case "DeactivateContactChannel":
		arn := incidentManagerPayloadString(payload, "ContactChannelId", s.firstContactChannelArnLocked())
		if ch, ok := s.contactChannels[arn]; ok {
			ch["ActivationStatus"] = "NOT_ACTIVATED"
		}
		return map[string]any{}
	case "UpdateContactChannel", "DeleteContactChannel", "SendActivationCode":
		if action == "DeleteContactChannel" {
			delete(s.contactChannels, incidentManagerPayloadString(payload, "ContactChannelId", ""))
		}
		return map[string]any{}
	case "PutContactPolicy":
		contactArn := incidentManagerPayloadString(payload, "ContactArn", s.firstContactArnLocked())
		s.resourcePolicies[contactArn] = incidentManagerPayloadString(payload, "Policy", "{\"Version\":\"2012-10-17\",\"Statement\":[]}")
		return map[string]any{}
	case "GetContactPolicy":
		contactArn := incidentManagerPayloadString(payload, "ContactArn", s.firstContactArnLocked())
		policy := s.resourcePolicies[contactArn]
		if strings.TrimSpace(policy) == "" {
			policy = "{\"Version\":\"2012-10-17\",\"Statement\":[]}"
		}
		return map[string]any{"Policy": policy}
	case "CreateRotation":
		name := incidentManagerPayloadString(payload, "Name", fmt.Sprintf("stackyard-rotation-%06d", s.nextIDLocked()))
		arn := "arn:aws:ssm-contacts:us-east-1:123456789012:rotation/" + name
		s.rotations[arn] = map[string]any{
			"RotationArn": arn,
			"Name":        name,
			"StartTime":   now,
			"ContactIds":  []any{s.firstContactArnLocked()},
		}
		return map[string]any{"RotationArn": arn}
	case "GetRotation":
		arn := incidentManagerPayloadString(payload, "RotationId", s.firstRotationArnLocked())
		if rot, ok := s.rotations[arn]; ok {
			return incidentManagerCloneMap(rot)
		}
		return map[string]any{"RotationArn": arn}
	case "ListRotations":
		items := make([]any, 0, len(s.rotations))
		for _, rot := range s.rotations {
			items = append(items, map[string]any{
				"RotationArn": rot["RotationArn"],
				"Name":        rot["Name"],
			})
		}
		return map[string]any{"Rotations": items, "NextToken": ""}
	case "UpdateRotation", "DeleteRotation":
		if action == "DeleteRotation" {
			delete(s.rotations, incidentManagerPayloadString(payload, "RotationId", ""))
		}
		return map[string]any{}
	case "CreateRotationOverride":
		overrideID := fmt.Sprintf("ro-%06d", s.nextIDLocked())
		return map[string]any{"RotationOverrideId": overrideID}
	case "DeleteRotationOverride":
		return map[string]any{}
	case "GetRotationOverride":
		return map[string]any{"RotationOverrideId": incidentManagerPayloadString(payload, "RotationOverrideId", "ro-000001")}
	case "ListRotationOverrides":
		return map[string]any{"RotationOverrides": []any{}, "NextToken": ""}
	case "ListRotationShifts", "ListPreviewRotationShifts":
		return map[string]any{"RotationShifts": []any{}, "NextToken": ""}
	case "StartEngagement":
		engagementArn := fmt.Sprintf("arn:aws:ssm-contacts:us-east-1:123456789012:engagement/e-%06d", s.nextIDLocked())
		s.engagements[engagementArn] = map[string]any{
			"EngagementArn": engagementArn,
			"ContactArn":    incidentManagerPayloadString(payload, "ContactId", s.firstContactArnLocked()),
			"StartTime":     now,
			"StopTime":      "",
		}
		return map[string]any{"EngagementArn": engagementArn}
	case "StopEngagement":
		engagementArn := incidentManagerPayloadString(payload, "EngagementId", "")
		if engagement, ok := s.engagements[engagementArn]; ok {
			engagement["StopTime"] = now
		}
		return map[string]any{}
	case "DescribeEngagement":
		engagementArn := incidentManagerPayloadString(payload, "EngagementId", s.firstEngagementArnLocked())
		engagement := s.engagements[engagementArn]
		if engagement == nil {
			engagement = map[string]any{"EngagementArn": engagementArn, "ContactArn": s.firstContactArnLocked(), "StartTime": now}
		}
		return map[string]any{"EngagementArn": engagement["EngagementArn"], "ContactArn": engagement["ContactArn"], "StartTime": engagement["StartTime"]}
	case "AcceptPage", "DescribePage":
		return map[string]any{"PageId": incidentManagerPayloadString(payload, "PageId", "page-000001")}
	case "ListEngagements":
		items := []any{}
		for _, e := range s.engagements {
			items = append(items, map[string]any{
				"EngagementArn": e["EngagementArn"],
				"ContactArn":    e["ContactArn"],
				"StartTime":     e["StartTime"],
			})
		}
		return map[string]any{"Engagements": items, "NextToken": ""}
	case "ListPageReceipts":
		return map[string]any{"Receipts": []any{}, "NextToken": ""}
	case "ListPageResolutions":
		return map[string]any{"PageResolutions": []any{}, "NextToken": ""}
	case "ListPagesByContact", "ListPagesByEngagement":
		return map[string]any{"Pages": []any{}, "NextToken": ""}
	}

	if strings.HasPrefix(action, "List") {
		key := incidentManagerListKey(action)
		return map[string]any{key: []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") {
		key := strings.TrimPrefix(action, "Get")
		if key == "" {
			key = "Result"
		}
		return map[string]any{key: map[string]any{}}
	}
	if strings.HasPrefix(action, "Describe") {
		key := strings.TrimPrefix(action, "Describe")
		if key == "" {
			key = "Result"
		}
		return map[string]any{key: map[string]any{}}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Start") {
		return map[string]any{"Arn": fmt.Sprintf("arn:aws:incident-manager:us-east-1:123456789012:%s/%06d", strings.ToLower(action), s.nextIDLocked())}
	}
	return map[string]any{}
}

func (s *incidentManagerStore) nextIDLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *incidentManagerStore) firstIncidentArnLocked() string {
	keys := make([]string, 0, len(s.incidents))
	for arn := range s.incidents {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "arn:aws:ssm-incidents:us-east-1:123456789012:incident-record/ir-000001"
	}
	return keys[0]
}

func (s *incidentManagerStore) firstResponsePlanArnLocked() string {
	keys := make([]string, 0, len(s.responsePlans))
	for arn := range s.responsePlans {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "arn:aws:ssm-incidents:us-east-1:123456789012:response-plan/stackyard-response-plan"
	}
	return keys[0]
}

func (s *incidentManagerStore) firstTimelineEventIDLocked() string {
	keys := make([]string, 0, len(s.timelineEvents))
	for id := range s.timelineEvents {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "te-000001"
	}
	return keys[0]
}

func (s *incidentManagerStore) firstContactArnLocked() string {
	keys := make([]string, 0, len(s.contacts))
	for arn := range s.contacts {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "arn:aws:ssm-contacts:us-east-1:123456789012:contact/stackyard-primary"
	}
	return keys[0]
}

func (s *incidentManagerStore) firstContactChannelArnLocked() string {
	keys := make([]string, 0, len(s.contactChannels))
	for arn := range s.contactChannels {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "arn:aws:ssm-contacts:us-east-1:123456789012:contact-channel/cc-000001"
	}
	return keys[0]
}

func (s *incidentManagerStore) firstRotationArnLocked() string {
	keys := make([]string, 0, len(s.rotations))
	for arn := range s.rotations {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "arn:aws:ssm-contacts:us-east-1:123456789012:rotation/stackyard-rotation"
	}
	return keys[0]
}

func (s *incidentManagerStore) firstEngagementArnLocked() string {
	keys := make([]string, 0, len(s.engagements))
	for arn := range s.engagements {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return "arn:aws:ssm-contacts:us-east-1:123456789012:engagement/e-000001"
	}
	return keys[0]
}

func (s *incidentManagerStore) incidentByArnLocked(arn string) map[string]any {
	if incident, ok := s.incidents[arn]; ok {
		return incidentManagerCloneMap(incident)
	}
	return map[string]any{
		"Arn":              arn,
		"Title":            "Stackyard incident",
		"Status":           "OPEN",
		"Impact":           3,
		"CreationTime":     time.Now().UTC().Format(time.RFC3339),
		"LastModifiedTime": time.Now().UTC().Format(time.RFC3339),
	}
}

func (s *incidentManagerStore) incidentSummariesLocked() []any {
	keys := make([]string, 0, len(s.incidents))
	for arn := range s.incidents {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		incident := s.incidents[arn]
		out = append(out, map[string]any{
			"Arn":              incident["Arn"],
			"Title":            incident["Title"],
			"Status":           incident["Status"],
			"Impact":           incident["Impact"],
			"CreationTime":     incident["CreationTime"],
			"LastModifiedTime": incident["LastModifiedTime"],
		})
	}
	return out
}

func (s *incidentManagerStore) responsePlanSummariesLocked() []any {
	keys := make([]string, 0, len(s.responsePlans))
	for arn := range s.responsePlans {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		plan := s.responsePlans[arn]
		out = append(out, map[string]any{
			"Arn":         plan["Arn"],
			"Name":        plan["Name"],
			"DisplayName": plan["DisplayName"],
		})
	}
	return out
}

func (s *incidentManagerStore) timelineEventSummariesLocked() []any {
	keys := make([]string, 0, len(s.timelineEvents))
	for id := range s.timelineEvents {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, id := range keys {
		event := s.timelineEvents[id]
		out = append(out, map[string]any{
			"EventId":   event["EventId"],
			"EventTime": event["EventTime"],
			"EventType": event["EventType"],
		})
	}
	return out
}

func (s *incidentManagerStore) applyTagsLocked(resourceArn string, payload map[string]any) {
	if resourceArn == "" {
		return
	}
	if _, ok := s.resourceTags[resourceArn]; !ok {
		s.resourceTags[resourceArn] = map[string]string{}
	}
	for _, key := range []string{"Tags", "tags"} {
		value, ok := payload[key]
		if !ok {
			continue
		}
		tagMap, ok := value.(map[string]any)
		if !ok {
			continue
		}
		for k, v := range tagMap {
			s.resourceTags[resourceArn][k] = fmt.Sprintf("%v", v)
		}
	}
}

func (s *incidentManagerStore) removeTagsLocked(resourceArn string, payload map[string]any) {
	tagMap, ok := s.resourceTags[resourceArn]
	if !ok {
		return
	}
	keys := incidentManagerPayloadStrings(payload, "TagKeys")
	if len(keys) == 0 {
		keys = incidentManagerPayloadStrings(payload, "tagKeys")
	}
	for _, key := range keys {
		delete(tagMap, key)
	}
}

func (s *incidentManagerStore) tagsMapLocked(resourceArn string) map[string]string {
	source, ok := s.resourceTags[resourceArn]
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(source))
	for k, v := range source {
		out[k] = v
	}
	return out
}

func incidentManagerPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			value := strings.TrimSpace(fmt.Sprintf("%v", v))
			if value != "" {
				return value
			}
		}
	}
	return fallback
}

func incidentManagerPayloadStrings(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		items, ok := v.([]any)
		if !ok {
			return nil
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			val := strings.TrimSpace(fmt.Sprintf("%v", item))
			if val != "" {
				out = append(out, val)
			}
		}
		return out
	}
	return nil
}

func incidentManagerListKey(action string) string {
	switch action {
	case "ListIncidentRecords":
		return "IncidentRecordSummaries"
	case "ListResponsePlans":
		return "ResponsePlanSummaries"
	case "ListTimelineEvents":
		return "EventSummaries"
	case "ListReplicationSets":
		return "ReplicationSetArns"
	case "ListIncidentFindings":
		return "Findings"
	case "ListRelatedItems":
		return "RelatedItems"
	case "ListContactChannels":
		return "ContactChannels"
	case "ListContacts":
		return "Contacts"
	case "ListEngagements":
		return "Engagements"
	case "ListRotationOverrides":
		return "RotationOverrides"
	case "ListRotations":
		return "Rotations"
	case "ListRotationShifts", "ListPreviewRotationShifts":
		return "RotationShifts"
	case "ListPagesByContact", "ListPagesByEngagement":
		return "Pages"
	case "ListPageReceipts":
		return "Receipts"
	case "ListPageResolutions":
		return "PageResolutions"
	default:
		key := strings.TrimPrefix(action, "List")
		if key == "" {
			key = "Items"
		}
		return key
	}
}

func incidentManagerCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for k, v := range input {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = incidentManagerCloneMap(typed)
		case []any:
			cloned := make([]any, len(typed))
			copy(cloned, typed)
			out[k] = cloned
		default:
			out[k] = v
		}
	}
	return out
}
