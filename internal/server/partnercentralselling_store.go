package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type partnerCentralSellingStore struct {
	mu sync.Mutex

	nextOpportunityID int64
	nextEngagementID  int64
	nextInvitationID  int64
	nextSnapshotID    int64
	nextSnapshotJobID int64
	nextTaskID        int64
	nextGenericID     int64

	opportunities map[string]map[string]any
	engagements   map[string]map[string]any
	invitations   map[string]map[string]any
	snapshots     map[string]map[string]any
	snapshotJobs  map[string]map[string]any
	tasks         map[string]map[string]any
	tags          map[string]map[string]string
	settings      map[string]any
}

func newPartnerCentralSellingStore() *partnerCentralSellingStore {
	s := &partnerCentralSellingStore{
		nextOpportunityID: 2,
		nextEngagementID:  2,
		nextInvitationID:  2,
		nextSnapshotID:    2,
		nextSnapshotJobID: 2,
		nextTaskID:        2,
		nextGenericID:     2,
		opportunities:     map[string]map[string]any{},
		engagements:       map[string]map[string]any{},
		invitations:       map[string]map[string]any{},
		snapshots:         map[string]map[string]any{},
		snapshotJobs:      map[string]map[string]any{},
		tasks:             map[string]map[string]any{},
		tags:              map[string]map[string]string{},
		settings: map[string]any{
			"Catalog":                         "AWS",
			"SellingSystem":                   "Stackyard",
			"OpportunityVisibility":           "ALL",
			"AutoAcceptEngagementInvitations": false,
		},
	}

	opportunity := s.ensureOpportunityLocked("opty-0000000000001")
	engagement := s.ensureEngagementLocked("engi-0000000000001", pcsStringAny(opportunity, "Identifier"))
	invitation := s.ensureInvitationLocked("eginv-0000000000001", pcsStringAny(engagement, "Identifier"), pcsStringAny(opportunity, "Identifier"))
	snapshot := s.ensureSnapshotLocked("rsnp-0000000000001")
	snapshotJob := s.ensureSnapshotJobLocked("rsj-0000000000001", pcsStringAny(snapshot, "Identifier"))
	task := s.ensureTaskLocked("task-0000000000001", "EngagementByAcceptingInvitation", pcsStringAny(invitation, "Identifier"), pcsStringAny(engagement, "Identifier"), pcsStringAny(opportunity, "Identifier"))

	s.tags[pcsResourceARN("engagement", pcsStringAny(engagement, "Identifier"))] = map[string]string{"seed": "true"}
	s.tags[pcsResourceARN("opportunity", pcsStringAny(opportunity, "Identifier"))] = map[string]string{"seed": "true"}
	s.tags[pcsResourceARN("snapshot", pcsStringAny(snapshot, "Identifier"))] = map[string]string{"seed": "true"}
	_ = snapshotJob
	_ = task

	return s
}

func (s *partnerCentralSellingStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	catalog := pcsFirstNonEmpty(pcsStringAny(payload, "Catalog", "catalog"), "AWS")
	identifier := pcsFirstNonEmpty(
		pcsStringAny(payload, "Identifier", "identifier", "OpportunityIdentifier", "EngagementIdentifier", "EngagementInvitationIdentifier", "ResourceSnapshotIdentifier", "ResourceSnapshotJobIdentifier"),
		"",
	)

	switch action {
	case "Partner":
		return map[string]any{"Partner": map[string]any{"Identifier": "partner-00000001", "Catalog": catalog, "Name": "Stackyard Partner"}}

	case "CreateOpportunity":
		id := fmt.Sprintf("opty-%013d", s.nextOpportunityIDLocked())
		opp := s.ensureOpportunityLocked(id)
		opp["Catalog"] = catalog
		opp["Title"] = pcsFirstNonEmpty(pcsStringAny(payload, "Title", "title"), "Stackyard Opportunity")
		opp["Status"] = "Draft"
		opp["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id}
	case "GetOpportunity":
		id := pcsFirstNonEmpty(identifier, "opty-0000000000001")
		return pcsCloneMap(s.ensureOpportunityLocked(id))
	case "ListOpportunities":
		return map[string]any{"OpportunitySummaries": s.listItemsLocked(s.opportunities), "NextToken": ""}
	case "UpdateOpportunity":
		id := pcsFirstNonEmpty(identifier, "opty-0000000000001")
		opp := s.ensureOpportunityLocked(id)
		for k, v := range payload {
			opp[k] = v
		}
		opp["Status"] = pcsFirstNonEmpty(pcsStringAny(payload, "Status"), pcsStringAny(opp, "Status"), "Draft")
		opp["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id}
	case "SubmitOpportunity":
		id := pcsFirstNonEmpty(identifier, "opty-0000000000001")
		opp := s.ensureOpportunityLocked(id)
		opp["Status"] = "Submitted"
		opp["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id}
	case "AssignOpportunity", "AssociateOpportunity", "DisassociateOpportunity":
		id := pcsFirstNonEmpty(identifier, "opty-0000000000001")
		s.ensureOpportunityLocked(id)
		return map[string]any{"Identifier": id}
	case "GetAwsOpportunitySummary":
		id := pcsFirstNonEmpty(identifier, "opty-0000000000001")
		opp := s.ensureOpportunityLocked(id)
		return map[string]any{"Identifier": id, "AwsOpportunitySummary": map[string]any{"Identifier": id, "Catalog": catalog, "Status": pcsFirstNonEmpty(pcsStringAny(opp, "Status"), "Draft")}}

	case "CreateEngagement":
		id := fmt.Sprintf("engi-%013d", s.nextEngagementIDLocked())
		oppID := pcsFirstNonEmpty(pcsStringAny(payload, "OpportunityIdentifier", "opportunityIdentifier"), "opty-0000000000001")
		eng := s.ensureEngagementLocked(id, oppID)
		eng["Catalog"] = catalog
		eng["Status"] = "Active"
		eng["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id}
	case "GetEngagement":
		id := pcsFirstNonEmpty(identifier, "engi-0000000000001")
		return pcsCloneMap(s.ensureEngagementLocked(id, "opty-0000000000001"))
	case "ListEngagements":
		return map[string]any{"EngagementSummaries": s.listItemsLocked(s.engagements), "NextToken": ""}
	case "CreateEngagementInvitation":
		id := fmt.Sprintf("eginv-%013d", s.nextInvitationIDLocked())
		engID := pcsFirstNonEmpty(pcsStringAny(payload, "EngagementIdentifier", "engagementIdentifier"), "engi-0000000000001")
		oppID := pcsFirstNonEmpty(pcsStringAny(payload, "OpportunityIdentifier", "opportunityIdentifier"), "opty-0000000000001")
		s.ensureEngagementLocked(engID, oppID)
		inv := s.ensureInvitationLocked(id, engID, oppID)
		inv["Catalog"] = catalog
		inv["Status"] = "Pending"
		inv["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id}
	case "GetEngagementInvitation":
		id := pcsFirstNonEmpty(identifier, "eginv-0000000000001")
		return pcsCloneMap(s.ensureInvitationLocked(id, "engi-0000000000001", "opty-0000000000001"))
	case "ListEngagementInvitations":
		return map[string]any{"EngagementInvitationSummaries": s.listItemsLocked(s.invitations), "NextToken": ""}
	case "AcceptEngagementInvitation":
		id := pcsFirstNonEmpty(identifier, "eginv-0000000000001")
		inv := s.ensureInvitationLocked(id, "engi-0000000000001", "opty-0000000000001")
		inv["Status"] = "Accepted"
		inv["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id, "Status": "Accepted"}
	case "RejectEngagementInvitation":
		id := pcsFirstNonEmpty(identifier, "eginv-0000000000001")
		inv := s.ensureInvitationLocked(id, "engi-0000000000001", "opty-0000000000001")
		inv["Status"] = "Rejected"
		inv["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id, "Status": "Rejected"}

	case "StartEngagementByAcceptingInvitationTask":
		taskID := fmt.Sprintf("task-%013d", s.nextTaskIDLocked())
		invID := pcsFirstNonEmpty(identifier, "eginv-0000000000001")
		inv := s.ensureInvitationLocked(invID, "engi-0000000000001", "opty-0000000000001")
		task := s.ensureTaskLocked(taskID, "EngagementByAcceptingInvitation", invID, pcsStringAny(inv, "EngagementIdentifier"), pcsStringAny(inv, "OpportunityIdentifier"))
		task["Catalog"] = catalog
		return map[string]any{"TaskId": taskID, "Status": pcsStringAny(task, "Status")}
	case "StartEngagementFromOpportunityTask":
		taskID := fmt.Sprintf("task-%013d", s.nextTaskIDLocked())
		oppID := pcsFirstNonEmpty(identifier, "opty-0000000000001")
		task := s.ensureTaskLocked(taskID, "EngagementFromOpportunity", "", "engi-0000000000001", oppID)
		task["Catalog"] = catalog
		return map[string]any{"TaskId": taskID, "Status": pcsStringAny(task, "Status")}
	case "ListEngagementByAcceptingInvitationTasks":
		return map[string]any{"TaskSummaries": s.listTasksByTypeLocked("EngagementByAcceptingInvitation"), "NextToken": ""}
	case "ListEngagementFromOpportunityTasks":
		return map[string]any{"TaskSummaries": s.listTasksByTypeLocked("EngagementFromOpportunity"), "NextToken": ""}
	case "ListOpportunityFromEngagementTasks":
		return map[string]any{"TaskSummaries": s.listTasksByTypeLocked("OpportunityFromEngagement"), "NextToken": ""}
	case "ListEngagementMembers":
		return map[string]any{"EngagementMemberSummaries": []any{map[string]any{"Identifier": "member-00000001", "Role": "OWNER"}}, "NextToken": ""}
	case "ListEngagementResourceAssociations":
		return map[string]any{"EngagementResourceAssociationSummaries": []any{}, "NextToken": ""}

	case "CreateResourceSnapshot":
		id := fmt.Sprintf("rsnp-%013d", s.nextSnapshotIDLocked())
		snap := s.ensureSnapshotLocked(id)
		snap["Catalog"] = catalog
		snap["Status"] = "ACTIVE"
		snap["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id}
	case "GetResourceSnapshot":
		id := pcsFirstNonEmpty(identifier, "rsnp-0000000000001")
		return pcsCloneMap(s.ensureSnapshotLocked(id))
	case "ListResourceSnapshots":
		return map[string]any{"ResourceSnapshotSummaries": s.listItemsLocked(s.snapshots), "NextToken": ""}
	case "CreateResourceSnapshotJob":
		id := fmt.Sprintf("rsj-%013d", s.nextSnapshotJobIDLocked())
		snapshotID := pcsFirstNonEmpty(pcsStringAny(payload, "ResourceSnapshotIdentifier", "resourceSnapshotIdentifier"), "rsnp-0000000000001")
		job := s.ensureSnapshotJobLocked(id, snapshotID)
		job["Status"] = "CREATED"
		job["Catalog"] = catalog
		return map[string]any{"Identifier": id}
	case "GetResourceSnapshotJob":
		id := pcsFirstNonEmpty(identifier, "rsj-0000000000001")
		return pcsCloneMap(s.ensureSnapshotJobLocked(id, "rsnp-0000000000001"))
	case "ListResourceSnapshotJobs":
		return map[string]any{"ResourceSnapshotJobSummaries": s.listItemsLocked(s.snapshotJobs), "NextToken": ""}
	case "StartResourceSnapshotJob":
		id := pcsFirstNonEmpty(identifier, "rsj-0000000000001")
		job := s.ensureSnapshotJobLocked(id, "rsnp-0000000000001")
		job["Status"] = "IN_PROGRESS"
		job["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id, "Status": "IN_PROGRESS"}
	case "StopResourceSnapshotJob":
		id := pcsFirstNonEmpty(identifier, "rsj-0000000000001")
		job := s.ensureSnapshotJobLocked(id, "rsnp-0000000000001")
		job["Status"] = "STOPPED"
		job["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"Identifier": id, "Status": "STOPPED"}
	case "DeleteResourceSnapshotJob":
		id := pcsFirstNonEmpty(identifier, "rsj-0000000000001")
		delete(s.snapshotJobs, id)
		return map[string]any{"Identifier": id}

	case "GetSellingSystemSettings":
		return pcsCloneMap(s.settings)
	case "PutSellingSystemSettings":
		for k, v := range payload {
			s.settings[k] = v
		}
		s.settings["LastModifiedDate"] = time.Now().UTC().Format(time.RFC3339)
		return pcsCloneMap(s.settings)

	case "ListSolutions":
		return map[string]any{"SolutionSummaries": []any{map[string]any{"Identifier": "sol-0000000000001", "Name": "Stackyard Solution", "Catalog": catalog}}, "NextToken": ""}
	case "TagResource":
		arn := pcsFirstNonEmpty(pcsStringAny(payload, "ResourceArn", "resourceArn", "Arn", "arn"), pcsResourceARN("opportunity", "opty-0000000000001"))
		tags := pcsStringMapAny(payload, "Tags", "tags")
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		for k, v := range tags {
			s.tags[arn][k] = v
		}
		return map[string]any{}
	case "UntagResource":
		arn := pcsFirstNonEmpty(pcsStringAny(payload, "ResourceArn", "resourceArn", "Arn", "arn"), pcsResourceARN("opportunity", "opty-0000000000001"))
		keys := pcsStringSliceAny(payload, "TagKeys", "tagKeys")
		if s.tags[arn] == nil {
			s.tags[arn] = map[string]string{}
		}
		for _, k := range keys {
			delete(s.tags[arn], k)
		}
		return map[string]any{}
	case "ListTagsForResource":
		arn := pcsFirstNonEmpty(pcsStringAny(payload, "ResourceArn", "resourceArn", "Arn", "arn"), pcsResourceARN("opportunity", "opty-0000000000001"))
		return map[string]any{"Tags": pcsAnyMapFromStringMap(s.tags[arn])}
	}

	if strings.HasPrefix(action, "Create") {
		return map[string]any{"Identifier": fmt.Sprintf("pc-%013d", s.nextGenericIDLocked())}
	}
	if strings.HasPrefix(action, "Get") {
		id := pcsFirstNonEmpty(identifier, "pc-0000000000001")
		return map[string]any{"Identifier": id, "Status": "ACTIVE", "Catalog": catalog}
	}
	if strings.HasPrefix(action, "List") {
		return map[string]any{"Items": []any{}, "NextToken": ""}
	}
	if strings.HasPrefix(action, "Start") {
		return map[string]any{"TaskId": fmt.Sprintf("task-%013d", s.nextTaskIDLocked()), "Status": "IN_PROGRESS"}
	}
	if strings.HasPrefix(action, "Accept") || strings.HasPrefix(action, "Reject") || strings.HasPrefix(action, "Assign") || strings.HasPrefix(action, "Associate") || strings.HasPrefix(action, "Disassociate") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Submit") || strings.HasPrefix(action, "Recall") || strings.HasPrefix(action, "Amend") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Send") || strings.HasPrefix(action, "Cancel") || strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Stop") {
		if identifier != "" {
			return map[string]any{"Identifier": identifier}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *partnerCentralSellingStore) ensureOpportunityLocked(id string) map[string]any {
	if id == "" {
		id = "opty-0000000000001"
	}
	if existing, ok := s.opportunities[id]; ok {
		return existing
	}
	entry := map[string]any{
		"Identifier":       id,
		"Catalog":          "AWS",
		"Title":            "Stackyard Opportunity",
		"Status":           "Draft",
		"LastModifiedDate": time.Now().UTC().Format(time.RFC3339),
	}
	s.opportunities[id] = entry
	return entry
}

func (s *partnerCentralSellingStore) ensureEngagementLocked(id, opportunityID string) map[string]any {
	if id == "" {
		id = "engi-0000000000001"
	}
	if existing, ok := s.engagements[id]; ok {
		return existing
	}
	entry := map[string]any{
		"Identifier":            id,
		"Catalog":               "AWS",
		"OpportunityIdentifier": pcsFirstNonEmpty(opportunityID, "opty-0000000000001"),
		"Status":                "Active",
		"LastModifiedDate":      time.Now().UTC().Format(time.RFC3339),
	}
	s.engagements[id] = entry
	return entry
}

func (s *partnerCentralSellingStore) ensureInvitationLocked(id, engagementID, opportunityID string) map[string]any {
	if id == "" {
		id = "eginv-0000000000001"
	}
	if existing, ok := s.invitations[id]; ok {
		return existing
	}
	entry := map[string]any{
		"Identifier":            id,
		"Catalog":               "AWS",
		"EngagementIdentifier":  pcsFirstNonEmpty(engagementID, "engi-0000000000001"),
		"OpportunityIdentifier": pcsFirstNonEmpty(opportunityID, "opty-0000000000001"),
		"Status":                "Pending",
		"LastModifiedDate":      time.Now().UTC().Format(time.RFC3339),
	}
	s.invitations[id] = entry
	return entry
}

func (s *partnerCentralSellingStore) ensureSnapshotLocked(id string) map[string]any {
	if id == "" {
		id = "rsnp-0000000000001"
	}
	if existing, ok := s.snapshots[id]; ok {
		return existing
	}
	entry := map[string]any{
		"Identifier":       id,
		"Catalog":          "AWS",
		"Status":           "ACTIVE",
		"LastModifiedDate": time.Now().UTC().Format(time.RFC3339),
	}
	s.snapshots[id] = entry
	return entry
}

func (s *partnerCentralSellingStore) ensureSnapshotJobLocked(id, snapshotID string) map[string]any {
	if id == "" {
		id = "rsj-0000000000001"
	}
	if existing, ok := s.snapshotJobs[id]; ok {
		return existing
	}
	entry := map[string]any{
		"Identifier":                 id,
		"Catalog":                    "AWS",
		"ResourceSnapshotIdentifier": pcsFirstNonEmpty(snapshotID, "rsnp-0000000000001"),
		"Status":                     "CREATED",
		"LastModifiedDate":           time.Now().UTC().Format(time.RFC3339),
	}
	s.snapshotJobs[id] = entry
	return entry
}

func (s *partnerCentralSellingStore) ensureTaskLocked(id, taskType, invitationID, engagementID, opportunityID string) map[string]any {
	if id == "" {
		id = "task-0000000000001"
	}
	if existing, ok := s.tasks[id]; ok {
		return existing
	}
	entry := map[string]any{
		"TaskId":                id,
		"TaskType":              taskType,
		"Catalog":               "AWS",
		"Status":                "IN_PROGRESS",
		"InvitationIdentifier":  invitationID,
		"EngagementIdentifier":  engagementID,
		"OpportunityIdentifier": opportunityID,
		"LastModifiedDate":      time.Now().UTC().Format(time.RFC3339),
	}
	s.tasks[id] = entry
	return entry
}

func (s *partnerCentralSellingStore) nextOpportunityIDLocked() int64 {
	id := s.nextOpportunityID
	s.nextOpportunityID++
	return id
}

func (s *partnerCentralSellingStore) nextEngagementIDLocked() int64 {
	id := s.nextEngagementID
	s.nextEngagementID++
	return id
}

func (s *partnerCentralSellingStore) nextInvitationIDLocked() int64 {
	id := s.nextInvitationID
	s.nextInvitationID++
	return id
}

func (s *partnerCentralSellingStore) nextSnapshotIDLocked() int64 {
	id := s.nextSnapshotID
	s.nextSnapshotID++
	return id
}

func (s *partnerCentralSellingStore) nextSnapshotJobIDLocked() int64 {
	id := s.nextSnapshotJobID
	s.nextSnapshotJobID++
	return id
}

func (s *partnerCentralSellingStore) nextTaskIDLocked() int64 {
	id := s.nextTaskID
	s.nextTaskID++
	return id
}

func (s *partnerCentralSellingStore) nextGenericIDLocked() int64 {
	id := s.nextGenericID
	s.nextGenericID++
	return id
}

func (s *partnerCentralSellingStore) listItemsLocked(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, pcsCloneMap(items[key]))
	}
	return out
}

func (s *partnerCentralSellingStore) listTasksByTypeLocked(taskType string) []any {
	keys := make([]string, 0, len(s.tasks))
	for key := range s.tasks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		entry := s.tasks[key]
		if !strings.EqualFold(pcsStringAny(entry, "TaskType"), taskType) {
			continue
		}
		out = append(out, pcsCloneMap(entry))
	}
	return out
}

func pcsResourceARN(kind, id string) string {
	resourceKind := strings.ToLower(strings.TrimSpace(kind))
	if resourceKind == "" {
		resourceKind = "resource"
	}
	resourceID := strings.TrimSpace(id)
	if resourceID == "" {
		resourceID = "pc-0000000000001"
	}
	if strings.HasPrefix(resourceID, "arn:") {
		return resourceID
	}
	return fmt.Sprintf("arn:aws:partnercentral-selling:us-east-1:123456789012:%s/%s", resourceKind, resourceID)
}

func pcsFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func pcsStringAny(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, key := range keys {
		for existingKey, value := range m {
			if strings.EqualFold(existingKey, key) {
				return strings.TrimSpace(fmt.Sprintf("%v", value))
			}
		}
	}
	return ""
}

func pcsStringMapAny(m map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		for existingKey, value := range m {
			if !strings.EqualFold(existingKey, key) {
				continue
			}
			if typed, ok := value.(map[string]any); ok {
				out := map[string]string{}
				for k, v := range typed {
					out[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", v))
				}
				return out
			}
		}
	}
	return map[string]string{}
}

func pcsStringSliceAny(m map[string]any, keys ...string) []string {
	for _, key := range keys {
		for existingKey, value := range m {
			if !strings.EqualFold(existingKey, key) {
				continue
			}
			if typed, ok := value.([]any); ok {
				out := make([]string, 0, len(typed))
				for _, item := range typed {
					trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
					if trimmed != "" {
						out = append(out, trimmed)
					}
				}
				return out
			}
		}
	}
	return nil
}

func pcsCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func pcsAnyMapFromStringMap(in map[string]string) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
