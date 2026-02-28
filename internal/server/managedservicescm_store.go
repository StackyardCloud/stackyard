package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type managedServicesCMStore struct {
	mu sync.Mutex

	nextRFC            int64
	nextAttachment     int64
	nextCorrespondence int64

	changeTypeVersions      map[string]map[string]any
	rfcs                    map[string]map[string]any
	rfcByClientToken        map[string]string
	attachmentsByRFC        map[string]map[string]map[string]any
	correspondencesByRFC    map[string][]map[string]any
	restrictedExecutionTime []map[string]any
}

func newManagedServicesCMStore() *managedServicesCMStore {
	now := time.Now().UTC().Format(time.RFC3339)

	changeTypeID := "ct-ec2-patch"
	changeTypeVersion := "1.0"
	changeTypeKey := changeTypeID + "|" + changeTypeVersion
	rfcID := "rfc-000001"
	attachmentID := "att-000001"
	correspondenceID := "corr-000001"

	return &managedServicesCMStore{
		nextRFC:            2,
		nextAttachment:     2,
		nextCorrespondence: 2,
		changeTypeVersions: map[string]map[string]any{
			changeTypeKey: {
				"ChangeTypeId":      changeTypeID,
				"Version":           changeTypeVersion,
				"Category":          "Infrastructure",
				"Subcategory":       "Compute",
				"Title":             "EC2 Managed Patching",
				"Description":       "Seeded change type version for deterministic AMS CM emulation",
				"AutomationStatus":  "Supported",
				"ApprovalRequired":  true,
				"LastModifiedTime":  now,
				"ChangeTypeItemId":  "cti-000001",
				"ExecutionType":     "Manual",
				"AccessLevel":       "Customer",
				"TemplateReference": "template://stackyard/ct-ec2-patch",
			},
		},
		rfcs: map[string]map[string]any{
			rfcID: {
				"RfcId":                       rfcID,
				"Title":                       "Seeded RFC",
				"Description":                 "Seeded request for change for AMS CM coverage",
				"ChangeTypeId":                changeTypeID,
				"ChangeTypeVersion":           changeTypeVersion,
				"Status":                      "Draft",
				"ApprovalStatus":              "NotRequested",
				"ActionState":                 "New",
				"Impact":                      "LOW",
				"RequestedExecutionStartTime": now,
				"RequestedExecutionEndTime":   now,
				"CreatedTime":                 now,
				"LastModifiedTime":            now,
				"CreatedBy":                   "stackyard@example.com",
			},
		},
		rfcByClientToken: map[string]string{},
		attachmentsByRFC: map[string]map[string]map[string]any{
			rfcID: {
				attachmentID: {
					"AttachmentId": attachmentID,
					"RfcId":        rfcID,
					"FileName":     "seeded-change-plan.txt",
					"S3Location":   "s3://stackyard-managedservices-cm/seeded-change-plan.txt",
					"Description":  "Seeded RFC attachment",
					"CreatedTime":  now,
				},
			},
		},
		correspondencesByRFC: map[string][]map[string]any{
			rfcID: {
				{
					"CorrespondenceId": correspondenceID,
					"RfcId":            rfcID,
					"Message":          "Seeded correspondence entry",
					"CreatedTime":      now,
					"CreatedBy":        "stackyard@example.com",
				},
			},
		},
		restrictedExecutionTime: []map[string]any{
			{
				"StartTime": now,
				"EndTime":   now,
				"Reason":    "Seeded maintenance window",
			},
		},
	}
}

func (s *managedServicesCMStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	rfcID := s.resolveRfcID(payload)

	switch action {
	case "ListChangeTypeCategories":
		return map[string]any{
			"Categories": []any{"Infrastructure", "Application"},
			"NextToken":  "",
		}
	case "ListChangeTypeSubcategories":
		category := managedServicesCMPayloadString(payload, "Category", "category")
		if category == "" {
			category = "Infrastructure"
		}
		return map[string]any{
			"Category":      category,
			"Subcategories": []any{"Compute", "Database"},
			"NextToken":     "",
		}
	case "ListChangeTypeItems":
		return map[string]any{
			"ChangeTypeItems": []any{
				map[string]any{
					"ChangeTypeId": changeTypeFromVersion(s.firstChangeTypeVersionLocked(), "ChangeTypeId"),
					"Title":        "EC2 Managed Patching",
					"Category":     "Infrastructure",
					"Subcategory":  "Compute",
				},
			},
			"NextToken": "",
		}
	case "ListChangeTypeOperations":
		return map[string]any{
			"ChangeTypeOperations": []any{
				map[string]any{"Name": "Patch", "Description": "Apply approved patch baseline"},
				map[string]any{"Name": "Reboot", "Description": "Perform controlled reboot"},
			},
			"NextToken": "",
		}
	case "ListChangeTypeClassificationSummaries":
		return map[string]any{
			"ChangeTypeClassificationSummaries": []any{
				map[string]any{"Category": "Infrastructure", "Subcategory": "Compute", "ItemCount": 1},
			},
			"NextToken": "",
		}
	case "ListChangeTypeVersionSummaries":
		items := make([]any, 0, len(s.changeTypeVersions))
		for _, version := range s.sortedChangeTypeVersionsLocked() {
			items = append(items, map[string]any{
				"ChangeTypeId":     managedServicesCMMapString(version, "ChangeTypeId"),
				"Version":          managedServicesCMMapString(version, "Version"),
				"Title":            managedServicesCMMapString(version, "Title"),
				"AutomationStatus": managedServicesCMMapString(version, "AutomationStatus"),
				"LastModifiedTime": managedServicesCMMapString(version, "LastModifiedTime"),
			})
		}
		return map[string]any{"ChangeTypeVersionSummaries": items, "NextToken": ""}
	case "GetChangeTypeVersion":
		changeTypeID := managedServicesCMPayloadString(payload, "ChangeTypeId", "changeTypeId")
		if changeTypeID == "" {
			changeTypeID = managedServicesCMMapString(s.firstChangeTypeVersionLocked(), "ChangeTypeId")
		}
		version := managedServicesCMPayloadString(payload, "Version", "version", "ChangeTypeVersion")
		if version == "" {
			version = managedServicesCMMapString(s.firstChangeTypeVersionLocked(), "Version")
		}
		return map[string]any{"ChangeTypeVersion": managedServicesCMCloneMap(s.ensureChangeTypeVersionLocked(changeTypeID, version, now))}

	case "CreateRfc":
		clientToken := managedServicesCMPayloadString(payload, "ClientToken", "clientToken")
		if clientToken != "" {
			if existing := strings.TrimSpace(s.rfcByClientToken[clientToken]); existing != "" {
				return map[string]any{"RfcId": existing}
			}
		}
		id := fmt.Sprintf("rfc-%06d", s.nextRFC)
		s.nextRFC++
		rfc := map[string]any{
			"RfcId":                       id,
			"Title":                       managedServicesCMPayloadString(payload, "Title", "title"),
			"Description":                 managedServicesCMPayloadString(payload, "Description", "description"),
			"ChangeTypeId":                managedServicesCMPayloadString(payload, "ChangeTypeId", "changeTypeId"),
			"ChangeTypeVersion":           managedServicesCMPayloadString(payload, "ChangeTypeVersion", "changeTypeVersion", "Version", "version"),
			"Status":                      "Draft",
			"ApprovalStatus":              "NotRequested",
			"ActionState":                 "New",
			"Impact":                      managedServicesCMPayloadString(payload, "Impact", "impact"),
			"RequestedExecutionStartTime": managedServicesCMPayloadString(payload, "RequestedExecutionStartTime", "requestedExecutionStartTime"),
			"RequestedExecutionEndTime":   managedServicesCMPayloadString(payload, "RequestedExecutionEndTime", "requestedExecutionEndTime"),
			"CreatedTime":                 now,
			"LastModifiedTime":            now,
			"CreatedBy":                   "stackyard@example.com",
		}
		if managedServicesCMMapString(rfc, "Title") == "" {
			rfc["Title"] = "stackyard-rfc-" + id
		}
		if managedServicesCMMapString(rfc, "ChangeTypeId") == "" {
			rfc["ChangeTypeId"] = managedServicesCMMapString(s.firstChangeTypeVersionLocked(), "ChangeTypeId")
		}
		if managedServicesCMMapString(rfc, "ChangeTypeVersion") == "" {
			rfc["ChangeTypeVersion"] = managedServicesCMMapString(s.firstChangeTypeVersionLocked(), "Version")
		}
		if managedServicesCMMapString(rfc, "Impact") == "" {
			rfc["Impact"] = "LOW"
		}
		if managedServicesCMMapString(rfc, "RequestedExecutionStartTime") == "" {
			rfc["RequestedExecutionStartTime"] = now
		}
		if managedServicesCMMapString(rfc, "RequestedExecutionEndTime") == "" {
			rfc["RequestedExecutionEndTime"] = now
		}
		s.rfcs[id] = rfc
		if clientToken != "" {
			s.rfcByClientToken[clientToken] = id
		}
		return map[string]any{"RfcId": id}

	case "GetRfc":
		return map[string]any{"Rfc": managedServicesCMCloneMap(s.ensureRFCLocked(rfcID, now))}

	case "UpdateRfc":
		rfc := s.ensureRFCLocked(rfcID, now)
		managedServicesCMApplyRFCUpdates(rfc, payload)
		rfc["LastModifiedTime"] = now
		return map[string]any{}

	case "SubmitRfc":
		rfc := s.ensureRFCLocked(rfcID, now)
		rfc["Status"] = "Submitted"
		rfc["ApprovalStatus"] = "PendingApproval"
		rfc["ActionState"] = "AwaitingApproval"
		rfc["LastModifiedTime"] = now
		return map[string]any{}

	case "ApproveRfc":
		rfc := s.ensureRFCLocked(rfcID, now)
		rfc["Status"] = "Approved"
		rfc["ApprovalStatus"] = "Approved"
		rfc["ActionState"] = "ReadyForExecution"
		rfc["LastModifiedTime"] = now
		return map[string]any{}

	case "RejectRfc":
		rfc := s.ensureRFCLocked(rfcID, now)
		rfc["Status"] = "Rejected"
		rfc["ApprovalStatus"] = "Rejected"
		rfc["ActionState"] = "Closed"
		rfc["LastModifiedTime"] = now
		return map[string]any{}

	case "CancelRfc":
		rfc := s.ensureRFCLocked(rfcID, now)
		rfc["Status"] = "Canceled"
		rfc["ActionState"] = "Closed"
		rfc["LastModifiedTime"] = now
		return map[string]any{}

	case "ListRfcSummaries":
		items := make([]any, 0, len(s.rfcs))
		for _, rfc := range s.sortedRFCsLocked() {
			items = append(items, map[string]any{
				"RfcId":                       managedServicesCMMapString(rfc, "RfcId"),
				"Title":                       managedServicesCMMapString(rfc, "Title"),
				"Status":                      managedServicesCMMapString(rfc, "Status"),
				"ApprovalStatus":              managedServicesCMMapString(rfc, "ApprovalStatus"),
				"ChangeTypeId":                managedServicesCMMapString(rfc, "ChangeTypeId"),
				"ChangeTypeVersion":           managedServicesCMMapString(rfc, "ChangeTypeVersion"),
				"RequestedExecutionStartTime": managedServicesCMMapString(rfc, "RequestedExecutionStartTime"),
				"RequestedExecutionEndTime":   managedServicesCMMapString(rfc, "RequestedExecutionEndTime"),
				"LastModifiedTime":            managedServicesCMMapString(rfc, "LastModifiedTime"),
			})
		}
		return map[string]any{"RfcSummaries": items, "NextToken": ""}

	case "CreateRfcAttachment":
		rfc := s.ensureRFCLocked(rfcID, now)
		attachmentID := fmt.Sprintf("att-%06d", s.nextAttachment)
		s.nextAttachment++
		attachment := map[string]any{
			"AttachmentId": attachmentID,
			"RfcId":        managedServicesCMMapString(rfc, "RfcId"),
			"FileName":     managedServicesCMPayloadString(payload, "FileName", "fileName"),
			"S3Location":   managedServicesCMPayloadString(payload, "S3Location", "s3Location", "AttachmentS3Uri"),
			"Description":  managedServicesCMPayloadString(payload, "Description", "description"),
			"CreatedTime":  now,
		}
		if managedServicesCMMapString(attachment, "FileName") == "" {
			attachment["FileName"] = "stackyard-" + attachmentID + ".txt"
		}
		if managedServicesCMMapString(attachment, "S3Location") == "" {
			attachment["S3Location"] = "s3://stackyard-managedservices-cm/" + attachmentID + ".txt"
		}
		s.ensureAttachmentMapLocked(managedServicesCMMapString(rfc, "RfcId"))[attachmentID] = attachment
		return map[string]any{"AttachmentId": attachmentID}

	case "GetRfcAttachment":
		attachmentID := managedServicesCMPayloadString(payload, "AttachmentId", "attachmentId")
		attachment := s.ensureAttachmentLocked(rfcID, attachmentID, now)
		return map[string]any{"RfcAttachment": managedServicesCMCloneMap(attachment)}

	case "ListRfcAttachmentSummaries":
		rfc := s.ensureRFCLocked(rfcID, now)
		attachments := s.ensureAttachmentMapLocked(managedServicesCMMapString(rfc, "RfcId"))
		items := make([]any, 0, len(attachments))
		ids := make([]string, 0, len(attachments))
		for id := range attachments {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			attachment := attachments[id]
			items = append(items, map[string]any{
				"AttachmentId": managedServicesCMMapString(attachment, "AttachmentId"),
				"FileName":     managedServicesCMMapString(attachment, "FileName"),
				"CreatedTime":  managedServicesCMMapString(attachment, "CreatedTime"),
			})
		}
		return map[string]any{"RfcAttachmentSummaries": items, "NextToken": ""}

	case "CreateRfcCorrespondence":
		rfc := s.ensureRFCLocked(rfcID, now)
		correspondence := map[string]any{
			"CorrespondenceId": fmt.Sprintf("corr-%06d", s.nextCorrespondence),
			"RfcId":            managedServicesCMMapString(rfc, "RfcId"),
			"Message":          managedServicesCMPayloadString(payload, "Message", "message"),
			"CreatedBy":        "stackyard@example.com",
			"CreatedTime":      now,
		}
		s.nextCorrespondence++
		if managedServicesCMMapString(correspondence, "Message") == "" {
			correspondence["Message"] = "RFC correspondence entry"
		}
		s.correspondencesByRFC[managedServicesCMMapString(rfc, "RfcId")] = append(
			s.correspondencesByRFC[managedServicesCMMapString(rfc, "RfcId")],
			correspondence,
		)
		return map[string]any{}

	case "ListRfcCorrespondences":
		rfc := s.ensureRFCLocked(rfcID, now)
		items := s.correspondencesByRFC[managedServicesCMMapString(rfc, "RfcId")]
		if len(items) == 0 {
			items = []map[string]any{
				{
					"CorrespondenceId": fmt.Sprintf("corr-%06d", s.nextCorrespondence),
					"RfcId":            managedServicesCMMapString(rfc, "RfcId"),
					"Message":          "No correspondence entries",
					"CreatedBy":        "stackyard@example.com",
					"CreatedTime":      now,
				},
			}
			s.nextCorrespondence++
			s.correspondencesByRFC[managedServicesCMMapString(rfc, "RfcId")] = items
		}
		return map[string]any{"RfcCorrespondences": managedServicesCMCloneListOfMaps(items), "NextToken": ""}

	case "ListRestrictedExecutionTimes":
		return map[string]any{
			"RestrictedExecutionTimes": managedServicesCMCloneListOfMaps(s.restrictedExecutionTime),
			"NextToken":                "",
		}

	case "UpdateRestrictedExecutionTimes":
		parsed := managedServicesCMExtractRestrictedTimes(payload)
		if len(parsed) == 0 {
			parsed = []map[string]any{
				{
					"StartTime": now,
					"EndTime":   now,
					"Reason":    "Updated by UpdateRestrictedExecutionTimes",
				},
			}
		}
		s.restrictedExecutionTime = parsed
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *managedServicesCMStore) resolveRfcID(payload map[string]any) string {
	id := managedServicesCMPayloadString(payload, "RfcId", "rfcId")
	if strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id)
	}
	for candidate := range s.rfcs {
		return candidate
	}
	return "rfc-000001"
}

func (s *managedServicesCMStore) ensureRFCLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "rfc-000001"
	}
	if existing, ok := s.rfcs[id]; ok {
		return existing
	}
	seed := s.firstChangeTypeVersionLocked()
	rfc := map[string]any{
		"RfcId":                       id,
		"Title":                       "stackyard-rfc-" + id,
		"Description":                 "Auto-created RFC",
		"ChangeTypeId":                managedServicesCMMapString(seed, "ChangeTypeId"),
		"ChangeTypeVersion":           managedServicesCMMapString(seed, "Version"),
		"Status":                      "Draft",
		"ApprovalStatus":              "NotRequested",
		"ActionState":                 "New",
		"Impact":                      "LOW",
		"RequestedExecutionStartTime": now,
		"RequestedExecutionEndTime":   now,
		"CreatedTime":                 now,
		"LastModifiedTime":            now,
		"CreatedBy":                   "stackyard@example.com",
	}
	s.rfcs[id] = rfc
	return rfc
}

func (s *managedServicesCMStore) ensureChangeTypeVersionLocked(changeTypeID, version, now string) map[string]any {
	changeTypeID = strings.TrimSpace(changeTypeID)
	version = strings.TrimSpace(version)
	if changeTypeID == "" {
		changeTypeID = "ct-ec2-patch"
	}
	if version == "" {
		version = "1.0"
	}
	key := changeTypeID + "|" + version
	if existing, ok := s.changeTypeVersions[key]; ok {
		return existing
	}
	v := map[string]any{
		"ChangeTypeId":      changeTypeID,
		"Version":           version,
		"Category":          "Infrastructure",
		"Subcategory":       "Compute",
		"Title":             "Auto-created change type",
		"Description":       "Auto-created change type version",
		"AutomationStatus":  "Supported",
		"ApprovalRequired":  true,
		"LastModifiedTime":  now,
		"ChangeTypeItemId":  "cti-auto-000001",
		"ExecutionType":     "Manual",
		"AccessLevel":       "Customer",
		"TemplateReference": "template://stackyard/" + changeTypeID,
	}
	s.changeTypeVersions[key] = v
	return v
}

func (s *managedServicesCMStore) firstChangeTypeVersionLocked() map[string]any {
	for _, v := range s.changeTypeVersions {
		return v
	}
	return map[string]any{
		"ChangeTypeId": "ct-ec2-patch",
		"Version":      "1.0",
	}
}

func (s *managedServicesCMStore) sortedChangeTypeVersionsLocked() []map[string]any {
	keys := make([]string, 0, len(s.changeTypeVersions))
	for key := range s.changeTypeVersions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.changeTypeVersions[key])
	}
	return out
}

func (s *managedServicesCMStore) sortedRFCsLocked() []map[string]any {
	keys := make([]string, 0, len(s.rfcs))
	for key := range s.rfcs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, s.rfcs[key])
	}
	return out
}

func (s *managedServicesCMStore) ensureAttachmentMapLocked(rfcID string) map[string]map[string]any {
	rfcID = strings.TrimSpace(rfcID)
	if rfcID == "" {
		rfcID = s.resolveRfcID(map[string]any{})
	}
	existing := s.attachmentsByRFC[rfcID]
	if existing == nil {
		existing = map[string]map[string]any{}
		s.attachmentsByRFC[rfcID] = existing
	}
	return existing
}

func (s *managedServicesCMStore) ensureAttachmentLocked(rfcID, attachmentID, now string) map[string]any {
	attachments := s.ensureAttachmentMapLocked(rfcID)
	attachmentID = strings.TrimSpace(attachmentID)
	if attachmentID != "" {
		if existing := attachments[attachmentID]; existing != nil {
			return existing
		}
	}
	if attachmentID == "" {
		for _, existing := range attachments {
			return existing
		}
		attachmentID = fmt.Sprintf("att-%06d", s.nextAttachment)
		s.nextAttachment++
	}
	created := map[string]any{
		"AttachmentId": attachmentID,
		"RfcId":        strings.TrimSpace(rfcID),
		"FileName":     "stackyard-" + attachmentID + ".txt",
		"S3Location":   "s3://stackyard-managedservices-cm/" + attachmentID + ".txt",
		"Description":  "Auto-created RFC attachment",
		"CreatedTime":  now,
	}
	attachments[attachmentID] = created
	return created
}

func managedServicesCMPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if payload == nil {
			continue
		}
		if raw, ok := payload[key]; ok {
			switch v := raw.(type) {
			case string:
				if strings.TrimSpace(v) != "" {
					return strings.TrimSpace(v)
				}
			}
		}
	}
	return ""
}

func managedServicesCMMapString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	s, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func managedServicesCMCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func managedServicesCMCloneListOfMaps(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, managedServicesCMCloneMap(item))
	}
	return out
}

func managedServicesCMApplyRFCUpdates(rfc map[string]any, payload map[string]any) {
	if rfc == nil || payload == nil {
		return
	}

	for _, key := range []string{
		"Title",
		"Description",
		"ChangeTypeId",
		"ChangeTypeVersion",
		"Impact",
		"RequestedExecutionStartTime",
		"RequestedExecutionEndTime",
	} {
		if value := managedServicesCMPayloadString(payload, key, lowerFirst(key)); value != "" {
			rfc[key] = value
		}
	}

	if nestedRaw, ok := payload["UpdateRfc"]; ok {
		if nested, ok := nestedRaw.(map[string]any); ok {
			for _, key := range []string{
				"Title",
				"Description",
				"ChangeTypeId",
				"ChangeTypeVersion",
				"Impact",
				"RequestedExecutionStartTime",
				"RequestedExecutionEndTime",
			} {
				if value := managedServicesCMPayloadString(nested, key, lowerFirst(key)); value != "" {
					rfc[key] = value
				}
			}
		}
	}
}

func managedServicesCMExtractRestrictedTimes(payload map[string]any) []map[string]any {
	var rawList []any
	if payload != nil {
		if raw, ok := payload["RestrictedExecutionTimes"]; ok {
			if list, ok := raw.([]any); ok {
				rawList = list
			}
		}
		if len(rawList) == 0 {
			if raw, ok := payload["restrictedExecutionTimes"]; ok {
				if list, ok := raw.([]any); ok {
					rawList = list
				}
			}
		}
	}
	if len(rawList) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(rawList))
	for _, item := range rawList {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{
			"StartTime": managedServicesCMPayloadString(m, "StartTime", "startTime"),
			"EndTime":   managedServicesCMPayloadString(m, "EndTime", "endTime"),
			"Reason":    managedServicesCMPayloadString(m, "Reason", "reason"),
		}
		if managedServicesCMMapString(entry, "StartTime") == "" && managedServicesCMMapString(entry, "EndTime") == "" {
			continue
		}
		if managedServicesCMMapString(entry, "Reason") == "" {
			entry["Reason"] = "Restricted window"
		}
		out = append(out, entry)
	}
	return out
}

func changeTypeFromVersion(version map[string]any, key string) string {
	if version == nil {
		return ""
	}
	value, _ := version[key].(string)
	return strings.TrimSpace(value)
}
