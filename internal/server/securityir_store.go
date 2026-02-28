package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type securityIRStore struct {
	mu sync.Mutex

	nextCaseID       int64
	nextCommentID    int64
	nextMembershipID int64

	cases          map[string]map[string]any
	caseComments   map[string]map[string]map[string]any
	memberships    map[string]map[string]any
	tags           map[string]map[string]string
	investigations map[string][]map[string]any
}

func newSecurityIRStore() *securityIRStore {
	s := &securityIRStore{
		nextCaseID:       2,
		nextCommentID:    2,
		nextMembershipID: 2,
		cases:            map[string]map[string]any{},
		caseComments:     map[string]map[string]map[string]any{},
		memberships:      map[string]map[string]any{},
		tags:             map[string]map[string]string{},
		investigations:   map[string][]map[string]any{},
	}

	now := time.Now().UTC().Format(time.RFC3339)
	seedCaseID := "case-00000001"
	seedMembershipID := "membership-00000001"
	seedCase := map[string]any{
		"caseId":          seedCaseID,
		"title":           "stackyard-seeded-case",
		"description":     "Seeded Security IR case",
		"status":          "OPEN",
		"resolverType":    "CUSTOMER",
		"createdDate":     now,
		"lastUpdatedDate": now,
		"caseArn":         securityIRCaseARN(seedCaseID),
	}
	s.cases[seedCaseID] = seedCase
	s.caseComments[seedCaseID] = map[string]map[string]any{
		"comment-00000001": {
			"commentId":       "comment-00000001",
			"caseId":          seedCaseID,
			"body":            "Seeded case comment",
			"createdDate":     now,
			"lastUpdatedDate": now,
		},
	}
	s.memberships[seedMembershipID] = map[string]any{
		"membershipId":    seedMembershipID,
		"accountId":       "123456789012",
		"region":          "us-east-1",
		"status":          "ACTIVE",
		"createdDate":     now,
		"lastUpdatedDate": now,
		"membershipArn":   securityIRMembershipARN(seedMembershipID),
	}
	s.investigations[seedCaseID] = []map[string]any{
		{
			"investigationId": "inv-00000001",
			"caseId":          seedCaseID,
			"status":          "OPEN",
			"updatedDate":     now,
		},
	}
	s.tags[seedCase["caseArn"].(string)] = map[string]string{"seed": "true"}

	return s
}

func (s *securityIRStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	caseID := securityIRFirstNonEmpty(
		securityIRPathParam(pathParams, "caseId"),
		securityIRStringAny(payload, "caseId", "caseID"),
		"case-00000001",
	)
	commentID := securityIRFirstNonEmpty(
		securityIRPathParam(pathParams, "commentId"),
		securityIRStringAny(payload, "commentId", "commentID"),
		"comment-00000001",
	)
	membershipID := securityIRFirstNonEmpty(
		securityIRPathParam(pathParams, "membershipId"),
		securityIRStringAny(payload, "membershipId", "membershipID"),
		"membership-00000001",
	)
	resourceARN := securityIRFirstNonEmpty(
		securityIRPathParam(pathParams, "resourceArn"),
		securityIRStringAny(payload, "resourceArn", "resourceARN"),
		securityIRCaseARN(caseID),
	)

	switch action {
	case "CreateCase":
		caseID = fmt.Sprintf("case-%08d", s.nextCaseID)
		s.nextCaseID++
		created := map[string]any{
			"caseId":          caseID,
			"title":           securityIRFirstNonEmpty(securityIRStringAny(payload, "title"), fmt.Sprintf("stackyard-case-%s", caseID)),
			"description":     securityIRFirstNonEmpty(securityIRStringAny(payload, "description"), "Stackyard Security IR case"),
			"status":          "OPEN",
			"resolverType":    "CUSTOMER",
			"createdDate":     now,
			"lastUpdatedDate": now,
			"caseArn":         securityIRCaseARN(caseID),
		}
		s.cases[caseID] = created
		return map[string]any{"caseId": caseID}
	case "GetCase":
		return securityIRCloneMap(s.ensureCaseLocked(caseID))
	case "ListCases":
		return map[string]any{"items": s.listCasesLocked(), "nextToken": ""}
	case "UpdateCase":
		c := s.ensureCaseLocked(caseID)
		for k, v := range payload {
			c[k] = v
		}
		c["lastUpdatedDate"] = now
		return map[string]any{"caseId": caseID}
	case "UpdateCaseStatus":
		c := s.ensureCaseLocked(caseID)
		c["status"] = securityIRFirstNonEmpty(securityIRStringAny(payload, "status"), "OPEN")
		c["lastUpdatedDate"] = now
		return map[string]any{"caseId": caseID}
	case "UpdateResolverType":
		c := s.ensureCaseLocked(caseID)
		c["resolverType"] = securityIRFirstNonEmpty(securityIRStringAny(payload, "resolverType"), "CUSTOMER")
		c["lastUpdatedDate"] = now
		return map[string]any{"caseId": caseID}
	case "CloseCase":
		c := s.ensureCaseLocked(caseID)
		c["status"] = "CLOSED"
		c["lastUpdatedDate"] = now
		return map[string]any{"caseId": caseID}

	case "CreateCaseComment":
		commentID = fmt.Sprintf("comment-%08d", s.nextCommentID)
		s.nextCommentID++
		comments := s.ensureCaseCommentsLocked(caseID)
		comments[commentID] = map[string]any{
			"commentId":       commentID,
			"caseId":          caseID,
			"body":            securityIRFirstNonEmpty(securityIRStringAny(payload, "body", "comment"), "Stackyard comment"),
			"createdDate":     now,
			"lastUpdatedDate": now,
		}
		return map[string]any{"commentId": commentID}
	case "UpdateCaseComment":
		comments := s.ensureCaseCommentsLocked(caseID)
		comment := comments[commentID]
		if comment == nil {
			comment = map[string]any{"commentId": commentID, "caseId": caseID, "createdDate": now}
			comments[commentID] = comment
		}
		for k, v := range payload {
			comment[k] = v
		}
		comment["lastUpdatedDate"] = now
		return map[string]any{"commentId": commentID}
	case "ListComments":
		return map[string]any{"items": s.listCommentsLocked(caseID), "nextToken": ""}
	case "ListCaseEdits":
		return map[string]any{"items": []any{}, "nextToken": ""}

	case "GetCaseAttachmentUploadUrl":
		return map[string]any{
			"url":               "https://example.invalid/security-ir/upload",
			"attachmentId":      securityIRFirstNonEmpty(securityIRStringAny(payload, "attachmentId"), "attachment-00000001"),
			"expirationDate":    now,
			"httpMethod":        "PUT",
			"contentType":       "application/octet-stream",
			"contentLength":     0,
			"additionalHeaders": map[string]any{},
		}
	case "GetCaseAttachmentDownloadUrl":
		return map[string]any{
			"url":            "https://example.invalid/security-ir/download",
			"expirationDate": now,
		}
	case "ListInvestigations":
		return map[string]any{"items": securityIRCloneListOfMaps(s.ensureInvestigationsLocked(caseID)), "nextToken": ""}
	case "SendFeedback":
		return map[string]any{}

	case "CreateMembership":
		membershipID = fmt.Sprintf("membership-%08d", s.nextMembershipID)
		s.nextMembershipID++
		created := map[string]any{
			"membershipId":    membershipID,
			"accountId":       securityIRFirstNonEmpty(securityIRStringAny(payload, "accountId"), "123456789012"),
			"region":          securityIRFirstNonEmpty(securityIRStringAny(payload, "region"), "us-east-1"),
			"status":          "ACTIVE",
			"createdDate":     now,
			"lastUpdatedDate": now,
			"membershipArn":   securityIRMembershipARN(membershipID),
		}
		s.memberships[membershipID] = created
		return map[string]any{"membershipId": membershipID}
	case "GetMembership":
		return securityIRCloneMap(s.ensureMembershipLocked(membershipID))
	case "ListMemberships":
		return map[string]any{"items": s.listMembershipsLocked(), "nextToken": ""}
	case "UpdateMembership":
		m := s.ensureMembershipLocked(membershipID)
		for k, v := range payload {
			m[k] = v
		}
		m["lastUpdatedDate"] = now
		return map[string]any{"membershipId": membershipID}
	case "CancelMembership":
		m := s.ensureMembershipLocked(membershipID)
		m["status"] = "CANCELLED"
		m["lastUpdatedDate"] = now
		return map[string]any{"membershipId": membershipID}
	case "BatchGetMemberAccountDetails":
		return map[string]any{"items": []any{}, "errors": []any{}}

	case "ListTagsForResource":
		out := map[string]any{}
		for k, v := range s.ensureTagsLocked(resourceARN) {
			out[k] = v
		}
		return map[string]any{"tags": out}
	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		if raw, ok := payload["tags"].(map[string]any); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		if raw, ok := payload["Tags"].(map[string]any); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range securityIRStringSlicePayload(payload, "tagKeys", "TagKeys") {
			delete(tags, key)
		}
		for _, key := range strings.Split(securityIRQueryString(query, "tagKeys"), ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}

	default:
		return map[string]any{}
	}
}

func (s *securityIRStore) ensureCaseLocked(caseID string) map[string]any {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		caseID = "case-00000001"
	}
	if existing, ok := s.cases[caseID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := map[string]any{
		"caseId":          caseID,
		"title":           fmt.Sprintf("stackyard-case-%s", caseID),
		"description":     "Stackyard Security IR case",
		"status":          "OPEN",
		"resolverType":    "CUSTOMER",
		"createdDate":     now,
		"lastUpdatedDate": now,
		"caseArn":         securityIRCaseARN(caseID),
	}
	s.cases[caseID] = created
	return created
}

func (s *securityIRStore) ensureCaseCommentsLocked(caseID string) map[string]map[string]any {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		caseID = "case-00000001"
	}
	if existing, ok := s.caseComments[caseID]; ok {
		return existing
	}
	created := map[string]map[string]any{}
	s.caseComments[caseID] = created
	return created
}

func (s *securityIRStore) ensureMembershipLocked(membershipID string) map[string]any {
	membershipID = strings.TrimSpace(membershipID)
	if membershipID == "" {
		membershipID = "membership-00000001"
	}
	if existing, ok := s.memberships[membershipID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := map[string]any{
		"membershipId":    membershipID,
		"accountId":       "123456789012",
		"region":          "us-east-1",
		"status":          "ACTIVE",
		"createdDate":     now,
		"lastUpdatedDate": now,
		"membershipArn":   securityIRMembershipARN(membershipID),
	}
	s.memberships[membershipID] = created
	return created
}

func (s *securityIRStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = securityIRCaseARN("case-00000001")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceARN] = created
	return created
}

func (s *securityIRStore) ensureInvestigationsLocked(caseID string) []map[string]any {
	caseID = strings.TrimSpace(caseID)
	if caseID == "" {
		caseID = "case-00000001"
	}
	if existing, ok := s.investigations[caseID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := []map[string]any{
		{
			"investigationId": "inv-00000001",
			"caseId":          caseID,
			"status":          "OPEN",
			"updatedDate":     now,
		},
	}
	s.investigations[caseID] = created
	return created
}

func (s *securityIRStore) listCasesLocked() []any {
	keys := make([]string, 0, len(s.cases))
	for k := range s.cases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityIRCloneMap(s.cases[k]))
	}
	return out
}

func (s *securityIRStore) listCommentsLocked(caseID string) []any {
	comments := s.ensureCaseCommentsLocked(caseID)
	keys := make([]string, 0, len(comments))
	for k := range comments {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityIRCloneMap(comments[k]))
	}
	return out
}

func (s *securityIRStore) listMembershipsLocked() []any {
	keys := make([]string, 0, len(s.memberships))
	for k := range s.memberships {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, securityIRCloneMap(s.memberships[k]))
	}
	return out
}

func securityIRCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func securityIRCloneListOfMaps(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, securityIRCloneMap(item))
	}
	return out
}

func securityIRPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if v := strings.TrimSpace(pathParams[key]); v != "" {
		return v
	}
	for k, v := range pathParams {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func securityIRStringAny(payload map[string]any, keys ...string) string {
	if payload == nil {
		return ""
	}
	for _, key := range keys {
		for k, v := range payload {
			if !strings.EqualFold(strings.TrimSpace(k), key) {
				continue
			}
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func securityIRStringSlicePayload(payload map[string]any, keys ...string) []string {
	if payload == nil {
		return nil
	}
	for _, key := range keys {
		for k, v := range payload {
			if !strings.EqualFold(strings.TrimSpace(k), key) {
				continue
			}
			switch vv := v.(type) {
			case []string:
				out := make([]string, 0, len(vv))
				for _, item := range vv {
					if s := strings.TrimSpace(item); s != "" {
						out = append(out, s)
					}
				}
				return out
			case []any:
				out := make([]string, 0, len(vv))
				for _, item := range vv {
					if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
						out = append(out, s)
					}
				}
				return out
			case string:
				raw := strings.TrimSpace(vv)
				if raw == "" {
					return nil
				}
				parts := strings.Split(raw, ",")
				out := make([]string, 0, len(parts))
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						out = append(out, p)
					}
				}
				return out
			}
		}
	}
	return nil
}

func securityIRQueryString(values url.Values, key string) string {
	if values == nil {
		return ""
	}
	if v := strings.TrimSpace(values.Get(key)); v != "" {
		return v
	}
	for k, vals := range values {
		if !strings.EqualFold(k, key) || len(vals) == 0 {
			continue
		}
		if v := strings.TrimSpace(vals[0]); v != "" {
			return v
		}
	}
	return ""
}

func securityIRFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func securityIRCaseARN(caseID string) string {
	return fmt.Sprintf("arn:aws:security-ir:us-east-1:123456789012:case/%s", strings.TrimSpace(caseID))
}

func securityIRMembershipARN(membershipID string) string {
	return fmt.Sprintf("arn:aws:security-ir:us-east-1:123456789012:membership/%s", strings.TrimSpace(membershipID))
}
