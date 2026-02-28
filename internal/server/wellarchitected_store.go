package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	wellArchitectedDefaultRegion    = "us-east-1"
	wellArchitectedDefaultAccountID = "123456789012"
)

type wellArchitectedStore struct {
	mu sync.Mutex

	nextID int64

	workloads        map[string]map[string]any
	lenses           map[string]map[string]any
	profiles         map[string]map[string]any
	reviewTemplates  map[string]map[string]any
	workloadShares   map[string]map[string]any
	lensShares       map[string]map[string]any
	profileShares    map[string]map[string]any
	templateShares   map[string]map[string]any
	shareInvitations map[string]map[string]any
	milestones       map[string]map[string]map[string]any
	answers          map[string]map[string]map[string]any // workloadId -> lensAlias -> questionId -> answer
	tags             map[string]map[string]string

	globalSettings map[string]any
}

func newWellArchitectedStore() *wellArchitectedStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &wellArchitectedStore{
		nextID:           2,
		workloads:        map[string]map[string]any{},
		lenses:           map[string]map[string]any{},
		profiles:         map[string]map[string]any{},
		reviewTemplates:  map[string]map[string]any{},
		workloadShares:   map[string]map[string]any{},
		lensShares:       map[string]map[string]any{},
		profileShares:    map[string]map[string]any{},
		templateShares:   map[string]map[string]any{},
		shareInvitations: map[string]map[string]any{},
		milestones:       map[string]map[string]map[string]any{},
		answers:          map[string]map[string]map[string]any{},
		tags:             map[string]map[string]string{},
		globalSettings: map[string]any{
			"JiraConfigurationStatus": "DISABLED",
			"UpdatedAt":               now,
		},
	}

	workload := s.ensureWorkloadLocked("workload-000001", now)
	s.ensureLensLocked("wellarchitected", now)
	s.ensureProfileLocked(wellArchitectedProfileARN("profile-000001"), now)
	s.ensureReviewTemplateLocked(wellArchitectedReviewTemplateARN("template-000001"), now)
	s.ensureWorkloadShareLocked("share-000001", wellArchitectedString(workload, "WorkloadId", "workload-000001"), now)
	s.ensureShareInvitationLocked("share-invitation-000001", now)
	s.ensureMilestoneLocked(wellArchitectedString(workload, "WorkloadId", "workload-000001"), "1", now)
	s.ensureAnswerLocked(wellArchitectedString(workload, "WorkloadId", "workload-000001"), "wellarchitected", "security_1", now)
	s.ensureTagsLocked(wellArchitectedString(workload, "WorkloadArn", wellArchitectedWorkloadARN("workload-000001")))["stackyard"] = "true"

	return s
}

func (s *wellArchitectedStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	workloadID := wellArchitectedLookupString(pathParams, payload, query, "WorkloadId", "workloadId")
	if workloadID == "" {
		workloadID = "workload-000001"
	}
	lensAlias := wellArchitectedLookupString(pathParams, payload, query, "LensAlias", "lensAlias")
	if lensAlias == "" {
		lensAlias = "wellarchitected"
	}
	questionID := wellArchitectedLookupString(pathParams, payload, query, "QuestionId", "questionId")
	if questionID == "" {
		questionID = "security_1"
	}
	milestoneNumber := wellArchitectedLookupString(pathParams, payload, query, "MilestoneNumber", "milestoneNumber")
	if milestoneNumber == "" {
		milestoneNumber = "1"
	}
	profileARN := wellArchitectedLookupString(pathParams, payload, query, "ProfileArn", "profileArn")
	if profileARN == "" {
		profileARN = wellArchitectedProfileARN("profile-000001")
	}
	templateARN := wellArchitectedLookupString(pathParams, payload, query, "TemplateArn", "templateArn", "ReviewTemplateArn", "reviewTemplateArn")
	if templateARN == "" {
		templateARN = wellArchitectedReviewTemplateARN("template-000001")
	}
	shareID := wellArchitectedLookupString(pathParams, payload, query, "ShareId", "shareId")
	if shareID == "" {
		shareID = "share-000001"
	}
	shareInvitationID := wellArchitectedLookupString(pathParams, payload, query, "ShareInvitationId", "shareInvitationId")
	if shareInvitationID == "" {
		shareInvitationID = "share-invitation-000001"
	}
	workloadARN := wellArchitectedLookupString(pathParams, payload, query, "WorkloadArn", "workloadArn", "ResourceArn", "resourceArn")
	if workloadARN == "" {
		workloadARN = wellArchitectedWorkloadARN(workloadID)
	}

	workload := s.ensureWorkloadLocked(workloadID, now)
	lens := s.ensureLensLocked(lensAlias, now)
	profile := s.ensureProfileLocked(profileARN, now)
	reviewTemplate := s.ensureReviewTemplateLocked(templateARN, now)
	workloadShare := s.ensureWorkloadShareLocked(shareID, workloadID, now)
	s.ensureLensShareLocked(shareID, lensAlias, now)
	s.ensureProfileShareLocked(shareID, profileARN, now)
	s.ensureTemplateShareLocked(shareID, templateARN, now)
	shareInvitation := s.ensureShareInvitationLocked(shareInvitationID, now)
	milestone := s.ensureMilestoneLocked(workloadID, milestoneNumber, now)
	answer := s.ensureAnswerLocked(workloadID, lensAlias, questionID, now)
	s.ensureTagsLocked(workloadARN)

	switch action {
	case "CreateWorkload":
		id := wellArchitectedLookupString(pathParams, payload, query, "WorkloadId", "workloadId")
		if id == "" {
			id = s.nextIdentifierLocked("workload")
		}
		item := s.ensureWorkloadLocked(id, now)
		wellArchitectedMergeMap(item, payload)
		item["WorkloadId"] = id
		item["WorkloadArn"] = wellArchitectedWorkloadARN(id)
		item["UpdatedAt"] = now
		s.ensureTagsLocked(wellArchitectedString(item, "WorkloadArn", wellArchitectedWorkloadARN(id)))
		return map[string]any{"WorkloadId": id, "WorkloadArn": item["WorkloadArn"], "Workload": wellArchitectedCloneMap(item)}

	case "GetWorkload":
		return map[string]any{"Workload": wellArchitectedCloneMap(workload)}
	case "UpdateWorkload":
		wellArchitectedMergeMap(workload, payload)
		workload["UpdatedAt"] = now
		return map[string]any{"Workload": wellArchitectedCloneMap(workload)}
	case "DeleteWorkload":
		delete(s.workloads, workloadID)
		delete(s.milestones, workloadID)
		delete(s.answers, workloadID)
		delete(s.tags, workloadARN)
		return map[string]any{}
	case "ListWorkloads":
		return map[string]any{"WorkloadSummaries": s.listWorkloadSummariesLocked(), "NextToken": ""}

	case "CreateWorkloadShare":
		createdShareID := wellArchitectedLookupString(pathParams, payload, query, "ShareId", "shareId")
		if createdShareID == "" {
			createdShareID = s.nextIdentifierLocked("share")
		}
		item := s.ensureWorkloadShareLocked(createdShareID, workloadID, now)
		wellArchitectedMergeMap(item, payload)
		item["UpdatedAt"] = now
		return map[string]any{"WorkloadShare": wellArchitectedCloneMap(item), "ShareId": createdShareID}
	case "UpdateWorkloadShare":
		wellArchitectedMergeMap(workloadShare, payload)
		workloadShare["UpdatedAt"] = now
		return map[string]any{"WorkloadShare": wellArchitectedCloneMap(workloadShare), "ShareId": shareID}
	case "DeleteWorkloadShare":
		delete(s.workloadShares, shareID)
		return map[string]any{}
	case "ListWorkloadShares":
		return map[string]any{"WorkloadShareSummaries": s.listSharesForWorkloadLocked(workloadID), "NextToken": ""}

	case "AssociateLenses", "DisassociateLenses":
		return map[string]any{"WorkloadId": workloadID, "LensAlias": lensAlias}
	case "AssociateProfiles", "DisassociateProfiles":
		return map[string]any{"WorkloadId": workloadID, "ProfileArn": profileARN}

	case "ListLenses":
		return map[string]any{"LensSummaries": s.listLensSummariesLocked(), "NextToken": ""}
	case "GetLens":
		return map[string]any{"Lens": wellArchitectedCloneMap(lens)}
	case "CreateLensVersion":
		return map[string]any{"LensAlias": lensAlias, "LensVersion": "1", "Status": "COMPLETE"}
	case "ImportLens":
		return map[string]any{"LensAlias": lensAlias, "Status": "IMPORTED"}
	case "ExportLens":
		return map[string]any{"LensJSON": "{}", "Status": "EXPORTED"}
	case "DeleteLens":
		delete(s.lenses, lensAlias)
		return map[string]any{}
	case "GetLensVersionDifference":
		return map[string]any{"VersionDifferences": map[string]any{"PillarDifferences": []any{}, "QuestionDifferences": []any{}}}
	case "CreateLensShare":
		createdShareID := wellArchitectedLookupString(pathParams, payload, query, "ShareId", "shareId")
		if createdShareID == "" {
			createdShareID = s.nextIdentifierLocked("share")
		}
		item := s.ensureLensShareLocked(createdShareID, lensAlias, now)
		wellArchitectedMergeMap(item, payload)
		item["UpdatedAt"] = now
		return map[string]any{"LensShare": wellArchitectedCloneMap(item), "ShareId": createdShareID}
	case "DeleteLensShare":
		delete(s.lensShares, shareID)
		return map[string]any{}
	case "ListLensShares":
		return map[string]any{"LensShareSummaries": s.listLensSharesLocked(lensAlias), "NextToken": ""}

	case "GetLensReview":
		return map[string]any{"LensReview": map[string]any{"WorkloadId": workloadID, "LensAlias": lensAlias, "UpdatedAt": now}}
	case "ListLensReviews":
		return map[string]any{"LensReviewSummaries": []any{map[string]any{"WorkloadId": workloadID, "LensAlias": lensAlias, "UpdatedAt": now}}, "NextToken": ""}
	case "UpdateLensReview":
		return map[string]any{"LensReview": map[string]any{"WorkloadId": workloadID, "LensAlias": lensAlias, "UpdatedAt": now}}
	case "UpgradeLensReview":
		return map[string]any{"WorkloadId": workloadID, "LensAlias": lensAlias, "Status": "UPGRADED"}
	case "GetLensReviewReport":
		return map[string]any{"LensReviewReport": map[string]any{"WorkloadId": workloadID, "LensAlias": lensAlias, "GeneratedAt": now}}
	case "ListLensReviewImprovements":
		return map[string]any{"ImprovementSummaries": []any{}, "NextToken": ""}

	case "GetAnswer":
		return map[string]any{"Answer": wellArchitectedCloneMap(answer)}
	case "ListAnswers":
		return map[string]any{"AnswerSummaries": s.listAnswerSummariesLocked(workloadID, lensAlias), "NextToken": ""}
	case "UpdateAnswer":
		wellArchitectedMergeMap(answer, payload)
		answer["UpdatedAt"] = now
		return map[string]any{"Answer": wellArchitectedCloneMap(answer)}

	case "CreateMilestone":
		item := s.ensureMilestoneLocked(workloadID, milestoneNumber, now)
		wellArchitectedMergeMap(item, payload)
		item["UpdatedAt"] = now
		return map[string]any{"MilestoneNumber": item["MilestoneNumber"], "MilestoneName": item["MilestoneName"]}
	case "GetMilestone":
		return map[string]any{"Milestone": wellArchitectedCloneMap(milestone)}
	case "ListMilestones":
		return map[string]any{"MilestoneSummaries": s.listMilestonesLocked(workloadID), "NextToken": ""}

	case "CreateProfile":
		arn := wellArchitectedLookupString(pathParams, payload, query, "ProfileArn", "profileArn")
		if arn == "" {
			arn = wellArchitectedProfileARN(s.nextIdentifierLocked("profile"))
		}
		item := s.ensureProfileLocked(arn, now)
		wellArchitectedMergeMap(item, payload)
		item["ProfileArn"] = arn
		item["UpdatedAt"] = now
		return map[string]any{"ProfileArn": arn, "Profile": wellArchitectedCloneMap(item)}
	case "GetProfile":
		return map[string]any{"Profile": wellArchitectedCloneMap(profile)}
	case "UpdateProfile":
		wellArchitectedMergeMap(profile, payload)
		profile["UpdatedAt"] = now
		return map[string]any{"Profile": wellArchitectedCloneMap(profile)}
	case "DeleteProfile":
		delete(s.profiles, profileARN)
		return map[string]any{}
	case "ListProfiles":
		return map[string]any{"ProfileSummaries": s.listProfileSummariesLocked(), "NextToken": ""}
	case "GetProfileTemplate":
		return map[string]any{"ProfileTemplate": map[string]any{"TemplateName": "stackyard-profile-template", "ProfileTemplateQuestions": []any{}}}
	case "UpgradeProfileVersion":
		return map[string]any{"WorkloadId": workloadID, "ProfileArn": profileARN, "Status": "UPGRADED"}

	case "CreateProfileShare":
		createdShareID := wellArchitectedLookupString(pathParams, payload, query, "ShareId", "shareId")
		if createdShareID == "" {
			createdShareID = s.nextIdentifierLocked("share")
		}
		item := s.ensureProfileShareLocked(createdShareID, profileARN, now)
		wellArchitectedMergeMap(item, payload)
		item["UpdatedAt"] = now
		return map[string]any{"ProfileShare": wellArchitectedCloneMap(item), "ShareId": createdShareID}
	case "DeleteProfileShare":
		delete(s.profileShares, shareID)
		return map[string]any{}
	case "ListProfileShares":
		return map[string]any{"ProfileShareSummaries": s.listProfileSharesLocked(profileARN), "NextToken": ""}

	case "CreateReviewTemplate":
		arn := wellArchitectedLookupString(pathParams, payload, query, "TemplateArn", "templateArn")
		if arn == "" {
			arn = wellArchitectedReviewTemplateARN(s.nextIdentifierLocked("template"))
		}
		item := s.ensureReviewTemplateLocked(arn, now)
		wellArchitectedMergeMap(item, payload)
		item["TemplateArn"] = arn
		item["UpdatedAt"] = now
		return map[string]any{"TemplateArn": arn, "ReviewTemplate": wellArchitectedCloneMap(item)}
	case "GetReviewTemplate":
		return map[string]any{"ReviewTemplate": wellArchitectedCloneMap(reviewTemplate)}
	case "UpdateReviewTemplate":
		wellArchitectedMergeMap(reviewTemplate, payload)
		reviewTemplate["UpdatedAt"] = now
		return map[string]any{"ReviewTemplate": wellArchitectedCloneMap(reviewTemplate)}
	case "DeleteReviewTemplate":
		delete(s.reviewTemplates, templateARN)
		return map[string]any{}
	case "ListReviewTemplates":
		return map[string]any{"ReviewTemplates": s.listReviewTemplatesLocked(), "NextToken": ""}

	case "GetReviewTemplateLensReview":
		return map[string]any{"ReviewTemplateLensReview": map[string]any{"TemplateArn": templateARN, "LensAlias": lensAlias, "UpdatedAt": now}}
	case "UpdateReviewTemplateLensReview":
		return map[string]any{"ReviewTemplateLensReview": map[string]any{"TemplateArn": templateARN, "LensAlias": lensAlias, "UpdatedAt": now}}
	case "UpgradeReviewTemplateLensReview":
		return map[string]any{"TemplateArn": templateARN, "LensAlias": lensAlias, "Status": "UPGRADED"}
	case "GetReviewTemplateAnswer":
		return map[string]any{"ReviewTemplateAnswer": map[string]any{"TemplateArn": templateARN, "LensAlias": lensAlias, "QuestionId": questionID, "UpdatedAt": now}}
	case "UpdateReviewTemplateAnswer":
		return map[string]any{"ReviewTemplateAnswer": map[string]any{"TemplateArn": templateARN, "LensAlias": lensAlias, "QuestionId": questionID, "UpdatedAt": now}}
	case "ListReviewTemplateAnswers":
		return map[string]any{"ReviewTemplateAnswerSummaries": []any{map[string]any{"TemplateArn": templateARN, "LensAlias": lensAlias, "QuestionId": questionID}}, "NextToken": ""}

	case "CreateTemplateShare":
		createdShareID := wellArchitectedLookupString(pathParams, payload, query, "ShareId", "shareId")
		if createdShareID == "" {
			createdShareID = s.nextIdentifierLocked("share")
		}
		item := s.ensureTemplateShareLocked(createdShareID, templateARN, now)
		wellArchitectedMergeMap(item, payload)
		item["UpdatedAt"] = now
		return map[string]any{"TemplateShare": wellArchitectedCloneMap(item), "ShareId": createdShareID}
	case "DeleteTemplateShare":
		delete(s.templateShares, shareID)
		return map[string]any{}
	case "ListTemplateShares":
		return map[string]any{"TemplateShareSummaries": s.listTemplateSharesLocked(templateARN), "NextToken": ""}

	case "ListNotifications":
		return map[string]any{
			"NotificationSummaries": []any{
				map[string]any{"Type": "REVIEW_UPDATE", "WorkloadId": workloadID, "CreatedAt": now},
			},
			"NextToken": "",
		}
	case "ListProfileNotifications":
		return map[string]any{
			"ProfileNotificationSummaries": []any{
				map[string]any{"Type": "PROFILE_SHARE", "ProfileArn": profileARN, "CreatedAt": now},
			},
			"NextToken": "",
		}
	case "ListShareInvitations":
		return map[string]any{"ShareInvitationSummaries": s.listShareInvitationsLocked(), "NextToken": ""}
	case "UpdateShareInvitation":
		wellArchitectedMergeMap(shareInvitation, payload)
		shareInvitation["UpdatedAt"] = now
		return map[string]any{"ShareInvitation": wellArchitectedCloneMap(shareInvitation)}

	case "GetConsolidatedReport":
		return map[string]any{
			"WorkloadId": workloadID,
			"Metrics": []any{
				map[string]any{"MetricType": "QUESTIONS_ANSWERED", "Value": 1},
			},
		}
	case "ListCheckDetails":
		return map[string]any{"CheckDetails": []any{}, "NextToken": ""}
	case "ListCheckSummaries":
		return map[string]any{"CheckSummaries": []any{}, "NextToken": ""}

	case "GetGlobalSettings":
		return map[string]any{"GlobalSettings": wellArchitectedCloneMap(s.globalSettings)}
	case "UpdateGlobalSettings":
		wellArchitectedMergeMap(s.globalSettings, payload)
		s.globalSettings["UpdatedAt"] = now
		return map[string]any{"GlobalSettings": wellArchitectedCloneMap(s.globalSettings)}

	case "UpdateIntegration":
		return map[string]any{"WorkloadId": workloadID, "Status": "UPDATED"}

	case "ListTagsForResource":
		tagMap := s.ensureTagsLocked(workloadARN)
		return map[string]any{"Tags": wellArchitectedCloneStringMap(tagMap)}
	case "TagResource":
		target := s.ensureTagsLocked(workloadARN)
		for k, v := range wellArchitectedExtractTags(payload) {
			target[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		target := s.ensureTagsLocked(workloadARN)
		for _, key := range wellArchitectedExtractTagKeys(payload, query) {
			delete(target, key)
		}
		return map[string]any{}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"NextToken": ""}
	}
	if strings.HasPrefix(action, "Get") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") || strings.HasPrefix(action, "Associate") || strings.HasPrefix(action, "Disassociate") || strings.HasPrefix(action, "Upgrade") || strings.HasPrefix(action, "Import") || strings.HasPrefix(action, "Export") || strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}
	return map[string]any{"Operation": action}
}

func (s *wellArchitectedStore) nextIdentifierLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("%s-%06d", prefix, id)
}

func (s *wellArchitectedStore) ensureWorkloadLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "workload-000001"
	}
	if existing := s.workloads[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"WorkloadId":   id,
		"WorkloadArn":  wellArchitectedWorkloadARN(id),
		"WorkloadName": "stackyard-workload-" + id,
		"Description":  "Stackyard Well-Architected workload",
		"Environment":  "PREPRODUCTION",
		"ReviewOwner":  "stackyard",
		"Lenses":       []any{"wellarchitected"},
		"AwsRegions":   []any{wellArchitectedDefaultRegion},
		"CreatedAt":    now,
		"UpdatedAt":    now,
	}
	s.workloads[id] = item
	return item
}

func (s *wellArchitectedStore) ensureLensLocked(alias, now string) map[string]any {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = "wellarchitected"
	}
	if existing := s.lenses[alias]; existing != nil {
		return existing
	}
	item := map[string]any{
		"LensAlias":   alias,
		"LensName":    "AWS Well-Architected Framework",
		"LensVersion": "1.0",
		"LensStatus":  "PUBLISHED",
		"CreatedAt":   now,
		"UpdatedAt":   now,
	}
	s.lenses[alias] = item
	return item
}

func (s *wellArchitectedStore) ensureProfileLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = wellArchitectedProfileARN("profile-000001")
	}
	if existing := s.profiles[arn]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ProfileArn":  arn,
		"ProfileName": wellArchitectedResourceIDFromARN(arn, "profile-000001"),
		"Version":     "1.0",
		"Owner":       "stackyard",
		"CreatedAt":   now,
		"UpdatedAt":   now,
	}
	s.profiles[arn] = item
	return item
}

func (s *wellArchitectedStore) ensureReviewTemplateLocked(arn, now string) map[string]any {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = wellArchitectedReviewTemplateARN("template-000001")
	}
	if existing := s.reviewTemplates[arn]; existing != nil {
		return existing
	}
	item := map[string]any{
		"TemplateArn":  arn,
		"TemplateName": wellArchitectedResourceIDFromARN(arn, "template-000001"),
		"Description":  "Stackyard review template",
		"Owner":        "stackyard",
		"CreatedAt":    now,
		"UpdatedAt":    now,
	}
	s.reviewTemplates[arn] = item
	return item
}

func (s *wellArchitectedStore) ensureWorkloadShareLocked(shareID, workloadID, now string) map[string]any {
	shareID = strings.TrimSpace(shareID)
	if shareID == "" {
		shareID = "share-000001"
	}
	if existing := s.workloadShares[shareID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ShareId":    shareID,
		"WorkloadId": workloadID,
		"Status":     "ACCEPTED",
		"CreatedAt":  now,
		"UpdatedAt":  now,
	}
	s.workloadShares[shareID] = item
	return item
}

func (s *wellArchitectedStore) ensureLensShareLocked(shareID, lensAlias, now string) map[string]any {
	shareID = strings.TrimSpace(shareID)
	if shareID == "" {
		shareID = "share-000001"
	}
	if existing := s.lensShares[shareID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ShareId":   shareID,
		"LensAlias": lensAlias,
		"Status":    "ACCEPTED",
		"CreatedAt": now,
		"UpdatedAt": now,
	}
	s.lensShares[shareID] = item
	return item
}

func (s *wellArchitectedStore) ensureProfileShareLocked(shareID, profileARN, now string) map[string]any {
	shareID = strings.TrimSpace(shareID)
	if shareID == "" {
		shareID = "share-000001"
	}
	if existing := s.profileShares[shareID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ShareId":    shareID,
		"ProfileArn": profileARN,
		"Status":     "ACCEPTED",
		"CreatedAt":  now,
		"UpdatedAt":  now,
	}
	s.profileShares[shareID] = item
	return item
}

func (s *wellArchitectedStore) ensureTemplateShareLocked(shareID, templateARN, now string) map[string]any {
	shareID = strings.TrimSpace(shareID)
	if shareID == "" {
		shareID = "share-000001"
	}
	if existing := s.templateShares[shareID]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ShareId":     shareID,
		"TemplateArn": templateARN,
		"Status":      "ACCEPTED",
		"CreatedAt":   now,
		"UpdatedAt":   now,
	}
	s.templateShares[shareID] = item
	return item
}

func (s *wellArchitectedStore) ensureShareInvitationLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "share-invitation-000001"
	}
	if existing := s.shareInvitations[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"ShareInvitationId": id,
		"Status":            "PENDING",
		"CreatedAt":         now,
		"UpdatedAt":         now,
	}
	s.shareInvitations[id] = item
	return item
}

func (s *wellArchitectedStore) ensureMilestoneLocked(workloadID, milestoneNumber, now string) map[string]any {
	workloadID = strings.TrimSpace(workloadID)
	if workloadID == "" {
		workloadID = "workload-000001"
	}
	milestoneNumber = strings.TrimSpace(milestoneNumber)
	if milestoneNumber == "" {
		milestoneNumber = "1"
	}
	if s.milestones[workloadID] == nil {
		s.milestones[workloadID] = map[string]map[string]any{}
	}
	if existing := s.milestones[workloadID][milestoneNumber]; existing != nil {
		return existing
	}
	item := map[string]any{
		"MilestoneNumber": milestoneNumber,
		"MilestoneName":   "Milestone " + milestoneNumber,
		"WorkloadId":      workloadID,
		"RecordedAt":      now,
	}
	s.milestones[workloadID][milestoneNumber] = item
	return item
}

func (s *wellArchitectedStore) ensureAnswerLocked(workloadID, lensAlias, questionID, now string) map[string]any {
	workloadID = strings.TrimSpace(workloadID)
	if workloadID == "" {
		workloadID = "workload-000001"
	}
	lensAlias = strings.TrimSpace(lensAlias)
	if lensAlias == "" {
		lensAlias = "wellarchitected"
	}
	questionID = strings.TrimSpace(questionID)
	if questionID == "" {
		questionID = "security_1"
	}
	if s.answers[workloadID] == nil {
		s.answers[workloadID] = map[string]map[string]any{}
	}
	key := lensAlias + "|" + questionID
	if existing := s.answers[workloadID][key]; existing != nil {
		return existing
	}
	item := map[string]any{
		"WorkloadId":       workloadID,
		"LensAlias":        lensAlias,
		"QuestionId":       questionID,
		"QuestionTitle":    "Stackyard Question",
		"SelectedChoices":  []any{},
		"Risk":             "UNANSWERED",
		"IsApplicable":     true,
		"Reason":           "",
		"HelpfulResource":  map[string]any{},
		"ImprovementPlan":  map[string]any{},
		"LastUpdatedBy":    "stackyard",
		"LastUpdatedAt":    now,
		"Notes":            "",
		"AdditionalInfo":   map[string]any{},
		"AnswerSourceType": "MANUAL",
	}
	s.answers[workloadID][key] = item
	return item
}

func (s *wellArchitectedStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = wellArchitectedWorkloadARN("workload-000001")
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceARN] = created
	return created
}

func (s *wellArchitectedStore) listWorkloadSummariesLocked() []any {
	keys := make([]string, 0, len(s.workloads))
	for key := range s.workloads {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		item := s.workloads[key]
		out = append(out, map[string]any{
			"WorkloadId":   wellArchitectedString(item, "WorkloadId", key),
			"WorkloadArn":  wellArchitectedString(item, "WorkloadArn", wellArchitectedWorkloadARN(key)),
			"WorkloadName": wellArchitectedString(item, "WorkloadName", key),
			"UpdatedAt":    wellArchitectedString(item, "UpdatedAt", ""),
		})
	}
	return out
}

func (s *wellArchitectedStore) listLensSummariesLocked() []any {
	keys := make([]string, 0, len(s.lenses))
	for key := range s.lenses {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		item := s.lenses[key]
		out = append(out, map[string]any{
			"LensAlias":   wellArchitectedString(item, "LensAlias", key),
			"LensName":    wellArchitectedString(item, "LensName", key),
			"LensVersion": wellArchitectedString(item, "LensVersion", "1.0"),
			"LensStatus":  wellArchitectedString(item, "LensStatus", "PUBLISHED"),
		})
	}
	return out
}

func (s *wellArchitectedStore) listProfileSummariesLocked() []any {
	keys := make([]string, 0, len(s.profiles))
	for key := range s.profiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		item := s.profiles[key]
		out = append(out, map[string]any{
			"ProfileArn":  wellArchitectedString(item, "ProfileArn", key),
			"ProfileName": wellArchitectedString(item, "ProfileName", ""),
			"UpdatedAt":   wellArchitectedString(item, "UpdatedAt", ""),
		})
	}
	return out
}

func (s *wellArchitectedStore) listReviewTemplatesLocked() []any {
	keys := make([]string, 0, len(s.reviewTemplates))
	for key := range s.reviewTemplates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		item := s.reviewTemplates[key]
		out = append(out, map[string]any{
			"TemplateArn":  wellArchitectedString(item, "TemplateArn", key),
			"TemplateName": wellArchitectedString(item, "TemplateName", ""),
			"UpdatedAt":    wellArchitectedString(item, "UpdatedAt", ""),
		})
	}
	return out
}

func (s *wellArchitectedStore) listSharesForWorkloadLocked(workloadID string) []any {
	keys := make([]string, 0, len(s.workloadShares))
	for key := range s.workloadShares {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		share := s.workloadShares[key]
		if wellArchitectedString(share, "WorkloadId", "") != workloadID {
			continue
		}
		out = append(out, wellArchitectedCloneMap(share))
	}
	return out
}

func (s *wellArchitectedStore) listLensSharesLocked(lensAlias string) []any {
	keys := make([]string, 0, len(s.lensShares))
	for key := range s.lensShares {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		share := s.lensShares[key]
		if wellArchitectedString(share, "LensAlias", "") != lensAlias {
			continue
		}
		out = append(out, wellArchitectedCloneMap(share))
	}
	return out
}

func (s *wellArchitectedStore) listProfileSharesLocked(profileARN string) []any {
	keys := make([]string, 0, len(s.profileShares))
	for key := range s.profileShares {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		share := s.profileShares[key]
		if wellArchitectedString(share, "ProfileArn", "") != profileARN {
			continue
		}
		out = append(out, wellArchitectedCloneMap(share))
	}
	return out
}

func (s *wellArchitectedStore) listTemplateSharesLocked(templateARN string) []any {
	keys := make([]string, 0, len(s.templateShares))
	for key := range s.templateShares {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		share := s.templateShares[key]
		if wellArchitectedString(share, "TemplateArn", "") != templateARN {
			continue
		}
		out = append(out, wellArchitectedCloneMap(share))
	}
	return out
}

func (s *wellArchitectedStore) listShareInvitationsLocked() []any {
	keys := make([]string, 0, len(s.shareInvitations))
	for key := range s.shareInvitations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, wellArchitectedCloneMap(s.shareInvitations[key]))
	}
	return out
}

func (s *wellArchitectedStore) listMilestonesLocked(workloadID string) []any {
	bucket := s.milestones[workloadID]
	if bucket == nil {
		return []any{}
	}
	keys := make([]string, 0, len(bucket))
	for key := range bucket {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, wellArchitectedCloneMap(bucket[key]))
	}
	return out
}

func (s *wellArchitectedStore) listAnswerSummariesLocked(workloadID, lensAlias string) []any {
	bucket := s.answers[workloadID]
	if bucket == nil {
		return []any{}
	}
	keys := make([]string, 0, len(bucket))
	for key := range bucket {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		answer := bucket[key]
		if wellArchitectedString(answer, "LensAlias", "") != lensAlias {
			continue
		}
		out = append(out, map[string]any{
			"QuestionId":      wellArchitectedString(answer, "QuestionId", "security_1"),
			"QuestionTitle":   wellArchitectedString(answer, "QuestionTitle", "Stackyard Question"),
			"Risk":            wellArchitectedString(answer, "Risk", "UNANSWERED"),
			"SelectedChoices": wellArchitectedAnySlice(answer["SelectedChoices"]),
		})
	}
	return out
}

func wellArchitectedLookupString(pathParams map[string]string, payload map[string]any, query url.Values, keys ...string) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if pathParams != nil {
			if v := strings.TrimSpace(pathParams[key]); v != "" {
				return v
			}
		}
		if payload != nil {
			if raw, ok := payload[key]; ok && raw != nil {
				if str, ok := raw.(string); ok {
					if v := strings.TrimSpace(str); v != "" {
						return v
					}
				}
			}
		}
		if query != nil {
			if values, ok := query[key]; ok {
				for _, value := range values {
					if v := strings.TrimSpace(value); v != "" {
						return v
					}
				}
			}
		}
	}
	return ""
}

func wellArchitectedExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	for _, key := range []string{"Tags", "tags"} {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		if m, ok := raw.(map[string]any); ok {
			for mk, mv := range m {
				if str, ok := mv.(string); ok {
					out[mk] = str
				}
			}
		}
		if arr, ok := raw.([]any); ok {
			for _, entry := range arr {
				item, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				k := wellArchitectedLookupMapString(item, "Key", "key")
				if k == "" {
					continue
				}
				out[k] = wellArchitectedLookupMapString(item, "Value", "value")
			}
		}
	}
	return out
}

func wellArchitectedExtractTagKeys(payload map[string]any, query url.Values) []string {
	out := []string{}
	if payload != nil {
		for _, key := range []string{"TagKeys", "tagKeys"} {
			raw, ok := payload[key]
			if !ok || raw == nil {
				continue
			}
			if arr, ok := raw.([]any); ok {
				for _, entry := range arr {
					str, ok := entry.(string)
					if !ok {
						continue
					}
					str = strings.TrimSpace(str)
					if str != "" {
						out = append(out, str)
					}
				}
			}
			if str, ok := raw.(string); ok {
				for _, token := range strings.Split(str, ",") {
					token = strings.TrimSpace(token)
					if token != "" {
						out = append(out, token)
					}
				}
			}
		}
	}
	for _, key := range []string{"tagKeys", "TagKeys"} {
		if query == nil {
			continue
		}
		for _, raw := range query[key] {
			for _, token := range strings.Split(raw, ",") {
				token = strings.TrimSpace(token)
				if token != "" {
					out = append(out, token)
				}
			}
		}
	}
	return out
}

func wellArchitectedLookupMapString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		str, ok := raw.(string)
		if !ok {
			continue
		}
		str = strings.TrimSpace(str)
		if str != "" {
			return str
		}
	}
	return ""
}

func wellArchitectedString(m map[string]any, key, def string) string {
	if m == nil {
		return strings.TrimSpace(def)
	}
	raw, ok := m[key]
	if !ok || raw == nil {
		return strings.TrimSpace(def)
	}
	str, ok := raw.(string)
	if !ok {
		return strings.TrimSpace(def)
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return strings.TrimSpace(def)
	}
	return str
}

func wellArchitectedAnySlice(v any) []any {
	if v == nil {
		return []any{}
	}
	if arr, ok := v.([]any); ok {
		return wellArchitectedCloneAnySlice(arr)
	}
	return []any{}
}

func wellArchitectedCloneAnySlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, wellArchitectedCloneValue(item))
	}
	return out
}

func wellArchitectedMergeMap(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for key, value := range src {
		dst[key] = wellArchitectedCloneValue(value)
	}
}

func wellArchitectedCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = wellArchitectedCloneValue(value)
	}
	return out
}

func wellArchitectedCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func wellArchitectedCloneValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return wellArchitectedCloneMap(typed)
	case []any:
		return wellArchitectedCloneAnySlice(typed)
	default:
		return typed
	}
}

func wellArchitectedWorkloadARN(workloadID string) string {
	workloadID = strings.TrimSpace(workloadID)
	if workloadID == "" {
		workloadID = "workload-000001"
	}
	return fmt.Sprintf("arn:aws:wellarchitected:%s:%s:workload/%s", wellArchitectedDefaultRegion, wellArchitectedDefaultAccountID, workloadID)
}

func wellArchitectedProfileARN(profileID string) string {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		profileID = "profile-000001"
	}
	return fmt.Sprintf("arn:aws:wellarchitected:%s:%s:profile/%s", wellArchitectedDefaultRegion, wellArchitectedDefaultAccountID, profileID)
}

func wellArchitectedReviewTemplateARN(templateID string) string {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		templateID = "template-000001"
	}
	return fmt.Sprintf("arn:aws:wellarchitected:%s:%s:reviewTemplate/%s", wellArchitectedDefaultRegion, wellArchitectedDefaultAccountID, templateID)
}

func wellArchitectedResourceIDFromARN(arn, def string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return def
	}
	if idx := strings.LastIndex(arn, "/"); idx >= 0 && idx+1 < len(arn) {
		id := strings.TrimSpace(arn[idx+1:])
		if id != "" {
			return id
		}
	}
	return def
}
