package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type codeGuruStore struct {
	mu                  sync.Mutex
	nextID              int64
	associations        map[string]*codeGuruRepositoryAssociation
	associationOrder    []string
	codeReviews         map[string]*codeGuruCodeReview
	codeReviewOrder     []string
	feedbackByReviewARN map[string]map[string]*codeGuruRecommendationFeedback
	tags                map[string]map[string]string
}

type codeGuruRepositoryAssociation struct {
	ID          string
	Arn         string
	Name        string
	Owner       string
	Provider    string
	State       string
	StateReason string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type codeGuruCodeReview struct {
	ID                       string
	Arn                      string
	Name                     string
	Type                     string
	State                    string
	RepositoryAssociationArn string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

type codeGuruRecommendationFeedback struct {
	CodeReviewArn    string
	RecommendationID string
	UserID           string
	Reactions        []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func newCodeGuruStore() *codeGuruStore {
	now := time.Now().UTC()
	association := &codeGuruRepositoryAssociation{
		ID:          "association-000001",
		Arn:         codeGuruAssociationARN("association-000001"),
		Name:        "stackyard-repo",
		Owner:       "stackyard",
		Provider:    "CodeCommit",
		State:       "Associated",
		StateReason: "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	review := &codeGuruCodeReview{
		ID:                       "code-review-000001",
		Arn:                      codeGuruCodeReviewARN("code-review-000001"),
		Name:                     "stackyard-seed-review",
		Type:                     "RepositoryAnalysis",
		State:                    "Completed",
		RepositoryAssociationArn: association.Arn,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	feedback := &codeGuruRecommendationFeedback{
		CodeReviewArn:    review.Arn,
		RecommendationID: "rec-000001",
		UserID:           "stackyard",
		Reactions:        []string{"ThumbsUp"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	return &codeGuruStore{
		nextID: 2,
		associations: map[string]*codeGuruRepositoryAssociation{
			association.Arn: association,
		},
		associationOrder: []string{association.Arn},
		codeReviews: map[string]*codeGuruCodeReview{
			review.Arn: review,
		},
		codeReviewOrder: []string{review.Arn},
		feedbackByReviewARN: map[string]map[string]*codeGuruRecommendationFeedback{
			review.Arn: {
				feedback.RecommendationID: feedback,
			},
		},
		tags: map[string]map[string]string{
			association.Arn: {"seed": "true"},
			review.Arn:      {"seed": "true"},
		},
	}
}

func (s *codeGuruStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "AssociateRepository":
		name, owner, provider := codeGuruRepositoryDetails(payload)
		id := fmt.Sprintf("association-%06d", s.nextLocked())
		association := &codeGuruRepositoryAssociation{
			ID:          id,
			Arn:         codeGuruAssociationARN(id),
			Name:        name,
			Owner:       owner,
			Provider:    provider,
			State:       "Associated",
			StateReason: "",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.associations[association.Arn] = association
		s.associationOrder = append(s.associationOrder, association.Arn)
		return map[string]any{
			"RepositoryAssociation": s.repositoryAssociationPayload(association),
			"Tags":                  s.cloneTags(association.Arn),
		}

	case "CreateCodeReview", "CreateCodeReviewInternal":
		name := codeGuruDefaultString(payload, "Name", fmt.Sprintf("stackyard-code-review-%06d", s.nextLocked()))
		associationArn := codeGuruDefaultString(payload, "RepositoryAssociationArn", s.firstAssociationArnLocked())
		if associationArn == "" {
			associationArn = codeGuruAssociationARN(fmt.Sprintf("association-%06d", s.nextLocked()))
		}
		association := s.ensureAssociationLocked(associationArn)
		reviewID := fmt.Sprintf("code-review-%06d", s.nextLocked())
		review := &codeGuruCodeReview{
			ID:                       reviewID,
			Arn:                      codeGuruCodeReviewARN(reviewID),
			Name:                     name,
			Type:                     codeGuruReviewType(payload),
			State:                    "Completed",
			RepositoryAssociationArn: association.Arn,
			CreatedAt:                now,
			UpdatedAt:                now,
		}
		s.codeReviews[review.Arn] = review
		s.codeReviewOrder = append(s.codeReviewOrder, review.Arn)
		if _, ok := s.feedbackByReviewARN[review.Arn]; !ok {
			s.feedbackByReviewARN[review.Arn] = map[string]*codeGuruRecommendationFeedback{}
		}
		return map[string]any{"CodeReview": s.codeReviewPayload(review)}

	case "CreateConnectionToken":
		token := fmt.Sprintf("stackyard-connection-token-%06d", s.nextLocked())
		return map[string]any{
			"ConnectionToken": token,
			"ExpiresAt":       now.Add(15 * time.Minute),
		}

	case "DescribeCodeReview":
		reviewArn := strings.TrimSpace(pathParams["CodeReviewArn"])
		if reviewArn == "" {
			reviewArn = codeGuruDefaultString(payload, "CodeReviewArn", s.firstCodeReviewArnLocked())
		}
		review := s.ensureCodeReviewLocked(reviewArn)
		return map[string]any{"CodeReview": s.codeReviewPayload(review)}

	case "DescribeRecommendationFeedback":
		reviewArn := strings.TrimSpace(pathParams["CodeReviewArn"])
		if reviewArn == "" {
			reviewArn = codeGuruDefaultString(payload, "CodeReviewArn", s.firstCodeReviewArnLocked())
		}
		recommendationID := codeGuruFirstNonEmpty(
			codeGuruQueryValue(query, "RecommendationId", "recommendationId"),
			codeGuruDefaultString(payload, "RecommendationId", ""),
			codeGuruDefaultString(payload, "RecommendationID", ""),
			"rec-000001",
		)
		feedback := s.ensureFeedbackLocked(reviewArn, recommendationID)
		return map[string]any{"RecommendationFeedback": s.feedbackPayload(feedback)}

	case "DescribeRepositoryAssociation":
		associationArn := strings.TrimSpace(pathParams["AssociationArn"])
		if associationArn == "" {
			associationArn = codeGuruDefaultString(payload, "AssociationArn", s.firstAssociationArnLocked())
		}
		association := s.ensureAssociationLocked(associationArn)
		return map[string]any{"RepositoryAssociation": s.repositoryAssociationPayload(association)}

	case "DisassociateRepository":
		associationArn := strings.TrimSpace(pathParams["AssociationArn"])
		if associationArn == "" {
			associationArn = codeGuruDefaultString(payload, "AssociationArn", s.firstAssociationArnLocked())
		}
		association := s.ensureAssociationLocked(associationArn)
		association.State = "Disassociated"
		association.UpdatedAt = now
		return map[string]any{"RepositoryAssociation": s.repositoryAssociationPayload(association)}

	case "GetMetricsData":
		return map[string]any{
			"MetricQueryResults": []any{
				map[string]any{
					"MetricName": "CodeReviewSuccess",
					"Values": []any{
						map[string]any{
							"Value":     1.0,
							"Timestamp": now,
						},
					},
				},
			},
		}

	case "ListCodeReviews":
		items := make([]any, 0, len(s.codeReviewOrder))
		for _, arn := range s.codeReviewOrder {
			review := s.codeReviews[arn]
			if review == nil {
				continue
			}
			items = append(items, s.codeReviewSummaryPayload(review))
		}
		return map[string]any{"CodeReviewSummaries": items}

	case "ListRecommendationFeedback":
		reviewArn := strings.TrimSpace(pathParams["CodeReviewArn"])
		if reviewArn == "" {
			reviewArn = codeGuruDefaultString(payload, "CodeReviewArn", s.firstCodeReviewArnLocked())
		}
		review := s.ensureCodeReviewLocked(reviewArn)
		feedbackSet := s.feedbackByReviewARN[review.Arn]
		ids := make([]string, 0, len(feedbackSet))
		for id := range feedbackSet {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		items := make([]any, 0, len(ids))
		for _, id := range ids {
			items = append(items, s.feedbackSummaryPayload(feedbackSet[id]))
		}
		return map[string]any{"RecommendationFeedbackSummaries": items}

	case "ListRecommendations":
		reviewArn := strings.TrimSpace(pathParams["CodeReviewArn"])
		if reviewArn == "" {
			reviewArn = codeGuruDefaultString(payload, "CodeReviewArn", s.firstCodeReviewArnLocked())
		}
		review := s.ensureCodeReviewLocked(reviewArn)
		return map[string]any{
			"RecommendationSummaries": []any{
				map[string]any{
					"RecommendationId": "rec-000001",
					"Description":      "Use stronger input validation.",
					"FilePath":         "main.go",
					"StartLine":        int64(1),
					"EndLine":          int64(1),
					"Severity":         "Medium",
					"RuleMetadata": map[string]any{
						"RuleId":           "stackyard.rule.input-validation",
						"RuleName":         "InputValidation",
						"ShortDescription": "Validate external input",
					},
					"CodeReviewArn": review.Arn,
				},
			},
		}

	case "ListRepositoryAssociations":
		items := make([]any, 0, len(s.associationOrder))
		for _, arn := range s.associationOrder {
			association := s.associations[arn]
			if association == nil {
				continue
			}
			items = append(items, s.repositoryAssociationSummaryPayload(association))
		}
		return map[string]any{"RepositoryAssociationSummaries": items}

	case "ListTagsForResource":
		resourceArn := strings.TrimSpace(pathParams["resourceArn"])
		if resourceArn == "" {
			resourceArn = codeGuruDefaultString(payload, "ResourceArn", s.firstAssociationArnLocked())
		}
		return map[string]any{"Tags": s.cloneTags(resourceArn)}

	case "ListThirdPartyRepositories":
		return map[string]any{
			"RepositorySummaries": []any{
				map[string]any{
					"Name":         "stackyard-third-party-repo",
					"Owner":        "stackyard",
					"ProviderType": "GitHub",
				},
			},
			"NextToken": "",
		}

	case "PutRecommendationFeedback":
		reviewArn := codeGuruDefaultString(payload, "CodeReviewArn", s.firstCodeReviewArnLocked())
		recommendationID := codeGuruFirstNonEmpty(
			codeGuruDefaultString(payload, "RecommendationId", ""),
			codeGuruDefaultString(payload, "RecommendationID", ""),
			"rec-000001",
		)
		feedback := s.ensureFeedbackLocked(reviewArn, recommendationID)
		feedback.Reactions = codeGuruStringSlice(codeGuruPayloadValue(payload, "Reactions"))
		if len(feedback.Reactions) == 0 {
			feedback.Reactions = []string{"ThumbsUp"}
		}
		feedback.UserID = codeGuruDefaultString(payload, "UserId", feedback.UserID)
		if feedback.UserID == "" {
			feedback.UserID = "stackyard"
		}
		feedback.UpdatedAt = now
		return map[string]any{}

	case "TagResource":
		resourceArn := strings.TrimSpace(pathParams["resourceArn"])
		if resourceArn == "" {
			resourceArn = codeGuruDefaultString(payload, "ResourceArn", s.firstAssociationArnLocked())
		}
		existing := s.ensureTagsLocked(resourceArn)
		for key, value := range codeGuruStringMap(codeGuruPayloadValue(payload, "Tags")) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := strings.TrimSpace(pathParams["resourceArn"])
		if resourceArn == "" {
			resourceArn = codeGuruDefaultString(payload, "ResourceArn", s.firstAssociationArnLocked())
		}
		tagKeys := codeGuruStringSlice(codeGuruPayloadValue(payload, "TagKeys"))
		if len(tagKeys) == 0 {
			for _, value := range query["tagKeys"] {
				if strings.TrimSpace(value) != "" {
					tagKeys = append(tagKeys, strings.TrimSpace(value))
				}
			}
			for _, value := range query["TagKeys"] {
				if strings.TrimSpace(value) != "" {
					tagKeys = append(tagKeys, strings.TrimSpace(value))
				}
			}
		}
		existing := s.ensureTagsLocked(resourceArn)
		for _, key := range tagKeys {
			delete(existing, key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *codeGuruStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *codeGuruStore) firstAssociationArnLocked() string {
	if len(s.associationOrder) > 0 {
		return s.associationOrder[0]
	}
	return ""
}

func (s *codeGuruStore) firstCodeReviewArnLocked() string {
	if len(s.codeReviewOrder) > 0 {
		return s.codeReviewOrder[0]
	}
	return ""
}

func (s *codeGuruStore) ensureAssociationLocked(arn string) *codeGuruRepositoryAssociation {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = codeGuruAssociationARN(fmt.Sprintf("association-%06d", s.nextLocked()))
	}
	if existing := s.associations[arn]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	id := codeGuruSuffixFromARN(arn)
	if id == "" {
		id = fmt.Sprintf("association-%06d", s.nextLocked())
		arn = codeGuruAssociationARN(id)
	}
	association := &codeGuruRepositoryAssociation{
		ID:          id,
		Arn:         arn,
		Name:        "stackyard-repo",
		Owner:       "stackyard",
		Provider:    "CodeCommit",
		State:       "Associated",
		StateReason: "",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.associations[arn] = association
	s.associationOrder = append(s.associationOrder, arn)
	return association
}

func (s *codeGuruStore) ensureCodeReviewLocked(arn string) *codeGuruCodeReview {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = codeGuruCodeReviewARN(fmt.Sprintf("code-review-%06d", s.nextLocked()))
	}
	if existing := s.codeReviews[arn]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	id := codeGuruSuffixFromARN(arn)
	if id == "" {
		id = fmt.Sprintf("code-review-%06d", s.nextLocked())
		arn = codeGuruCodeReviewARN(id)
	}
	review := &codeGuruCodeReview{
		ID:                       id,
		Arn:                      arn,
		Name:                     "stackyard-code-review",
		Type:                     "RepositoryAnalysis",
		State:                    "Completed",
		RepositoryAssociationArn: s.firstAssociationArnLocked(),
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if review.RepositoryAssociationArn == "" {
		association := s.ensureAssociationLocked("")
		review.RepositoryAssociationArn = association.Arn
	}
	s.codeReviews[arn] = review
	s.codeReviewOrder = append(s.codeReviewOrder, arn)
	if _, ok := s.feedbackByReviewARN[arn]; !ok {
		s.feedbackByReviewARN[arn] = map[string]*codeGuruRecommendationFeedback{}
	}
	return review
}

func (s *codeGuruStore) ensureFeedbackLocked(reviewArn, recommendationID string) *codeGuruRecommendationFeedback {
	review := s.ensureCodeReviewLocked(reviewArn)
	recommendationID = strings.TrimSpace(recommendationID)
	if recommendationID == "" {
		recommendationID = "rec-000001"
	}
	if _, ok := s.feedbackByReviewARN[review.Arn]; !ok {
		s.feedbackByReviewARN[review.Arn] = map[string]*codeGuruRecommendationFeedback{}
	}
	if existing := s.feedbackByReviewARN[review.Arn][recommendationID]; existing != nil {
		return existing
	}
	now := time.Now().UTC()
	feedback := &codeGuruRecommendationFeedback{
		CodeReviewArn:    review.Arn,
		RecommendationID: recommendationID,
		UserID:           "stackyard",
		Reactions:        []string{"ThumbsUp"},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.feedbackByReviewARN[review.Arn][recommendationID] = feedback
	return feedback
}

func (s *codeGuruStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = s.firstAssociationArnLocked()
	}
	if resourceArn == "" {
		resourceArn = codeGuruAssociationARN(fmt.Sprintf("association-%06d", s.nextLocked()))
	}
	existing := s.tags[resourceArn]
	if existing == nil {
		existing = map[string]string{}
		s.tags[resourceArn] = existing
	}
	return existing
}

func (s *codeGuruStore) repositoryAssociationPayload(association *codeGuruRepositoryAssociation) map[string]any {
	return map[string]any{
		"AssociationId":        association.ID,
		"AssociationArn":       association.Arn,
		"ConnectionArn":        "",
		"Name":                 association.Name,
		"Owner":                association.Owner,
		"ProviderType":         association.Provider,
		"State":                association.State,
		"StateReason":          association.StateReason,
		"LastUpdatedTimeStamp": association.UpdatedAt,
		"CreatedTimeStamp":     association.CreatedAt,
		"KMSKeyDetails": map[string]any{
			"KMSKeyId":         "",
			"EncryptionOption": "AWS_OWNED_CMK",
		},
	}
}

func (s *codeGuruStore) repositoryAssociationSummaryPayload(association *codeGuruRepositoryAssociation) map[string]any {
	return map[string]any{
		"AssociationId":        association.ID,
		"AssociationArn":       association.Arn,
		"ConnectionArn":        "",
		"Name":                 association.Name,
		"Owner":                association.Owner,
		"ProviderType":         association.Provider,
		"State":                association.State,
		"LastUpdatedTimeStamp": association.UpdatedAt,
	}
}

func (s *codeGuruStore) codeReviewPayload(review *codeGuruCodeReview) map[string]any {
	association := s.ensureAssociationLocked(review.RepositoryAssociationArn)
	return map[string]any{
		"Name":                 review.Name,
		"CodeReviewArn":        review.Arn,
		"RepositoryName":       association.Name,
		"Owner":                association.Owner,
		"ProviderType":         association.Provider,
		"State":                review.State,
		"CreatedTimeStamp":     review.CreatedAt,
		"LastUpdatedTimeStamp": review.UpdatedAt,
		"Type":                 review.Type,
		"Metrics": map[string]any{
			"MeteredLinesOfCodeCount":    int64(100),
			"SuppressedLinesOfCodeCount": int64(0),
			"FindingsCount":              int64(1),
		},
		"RepositoryAssociationArn": review.RepositoryAssociationArn,
	}
}

func (s *codeGuruStore) codeReviewSummaryPayload(review *codeGuruCodeReview) map[string]any {
	association := s.ensureAssociationLocked(review.RepositoryAssociationArn)
	return map[string]any{
		"Name":                 review.Name,
		"CodeReviewArn":        review.Arn,
		"RepositoryName":       association.Name,
		"Owner":                association.Owner,
		"ProviderType":         association.Provider,
		"State":                review.State,
		"CreatedTimeStamp":     review.CreatedAt,
		"LastUpdatedTimeStamp": review.UpdatedAt,
		"Type":                 review.Type,
		"MetricsSummary": map[string]any{
			"MeteredLinesOfCodeCount":    int64(100),
			"SuppressedLinesOfCodeCount": int64(0),
			"FindingsCount":              int64(1),
		},
	}
}

func (s *codeGuruStore) feedbackPayload(feedback *codeGuruRecommendationFeedback) map[string]any {
	return map[string]any{
		"CodeReviewArn":        feedback.CodeReviewArn,
		"RecommendationId":     feedback.RecommendationID,
		"Reactions":            append([]string{}, feedback.Reactions...),
		"UserId":               feedback.UserID,
		"LastUpdatedTimeStamp": feedback.UpdatedAt,
	}
}

func (s *codeGuruStore) feedbackSummaryPayload(feedback *codeGuruRecommendationFeedback) map[string]any {
	return map[string]any{
		"RecommendationId":     feedback.RecommendationID,
		"Reactions":            append([]string{}, feedback.Reactions...),
		"UserId":               feedback.UserID,
		"LastUpdatedTimeStamp": feedback.UpdatedAt,
	}
}

func (s *codeGuruStore) cloneTags(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		return map[string]string{}
	}
	existing := s.tags[resourceArn]
	if len(existing) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(existing))
	for key, value := range existing {
		out[key] = value
	}
	return out
}

func codeGuruAssociationARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "association-000000"
	}
	return fmt.Sprintf("arn:aws:codeguru-reviewer:us-east-1:123456789012:association:%s", id)
}

func codeGuruCodeReviewARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "code-review-000000"
	}
	return fmt.Sprintf("arn:aws:codeguru-reviewer:us-east-1:123456789012:code-review:%s", id)
}

func codeGuruSuffixFromARN(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	if idx := strings.LastIndex(arn, ":"); idx >= 0 && idx+1 < len(arn) {
		return strings.TrimSpace(arn[idx+1:])
	}
	return ""
}

func codeGuruDefaultString(payload map[string]any, key, def string) string {
	value := strings.TrimSpace(codeGuruAsString(codeGuruPayloadValue(payload, key)))
	if value == "" {
		return def
	}
	return value
}

func codeGuruPayloadValue(payload map[string]any, key string) any {
	for existingKey, value := range payload {
		if strings.EqualFold(strings.TrimSpace(existingKey), strings.TrimSpace(key)) {
			return value
		}
	}
	return nil
}

func codeGuruAsString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", v)
	}
}

func codeGuruStringSlice(value any) []string {
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
			text := strings.TrimSpace(codeGuruAsString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(codeGuruAsString(value))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func codeGuruStringMap(value any) map[string]string {
	out := map[string]string{}
	switch typed := value.(type) {
	case map[string]string:
		for key, item := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = item
		}
	case map[string]any:
		for key, item := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(codeGuruAsString(item))
		}
	}
	return out
}

func codeGuruQueryValue(query url.Values, keys ...string) string {
	for _, key := range keys {
		values := query[key]
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func codeGuruFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func codeGuruReviewType(payload map[string]any) string {
	typeValue := codeGuruPayloadValue(payload, "Type")
	switch typed := typeValue.(type) {
	case string:
		if strings.TrimSpace(typed) != "" {
			return strings.TrimSpace(typed)
		}
	case map[string]any:
		for key := range typed {
			if strings.TrimSpace(key) != "" {
				return strings.TrimSpace(key)
			}
		}
	}
	return "RepositoryAnalysis"
}

func codeGuruRepositoryDetails(payload map[string]any) (name, owner, provider string) {
	name = "stackyard-repo"
	owner = "stackyard"
	provider = "CodeCommit"

	repository := codeGuruPayloadValue(payload, "Repository")
	repoMap, ok := repository.(map[string]any)
	if !ok {
		return name, owner, provider
	}

	if nested, ok := repoMap["CodeCommit"].(map[string]any); ok {
		if v := strings.TrimSpace(codeGuruAsString(nested["Name"])); v != "" {
			name = v
		}
		provider = "CodeCommit"
		return name, owner, provider
	}
	if nested, ok := repoMap["GitHubEnterpriseServer"].(map[string]any); ok {
		if v := strings.TrimSpace(codeGuruAsString(nested["Name"])); v != "" {
			name = v
		}
		if v := strings.TrimSpace(codeGuruAsString(nested["Owner"])); v != "" {
			owner = v
		}
		provider = "GitHubEnterpriseServer"
		return name, owner, provider
	}
	if nested, ok := repoMap["Bitbucket"].(map[string]any); ok {
		if v := strings.TrimSpace(codeGuruAsString(nested["Name"])); v != "" {
			name = v
		}
		if v := strings.TrimSpace(codeGuruAsString(nested["Owner"])); v != "" {
			owner = v
		}
		provider = "Bitbucket"
		return name, owner, provider
	}
	if nested, ok := repoMap["S3Bucket"].(map[string]any); ok {
		if v := strings.TrimSpace(codeGuruAsString(nested["Name"])); v != "" {
			name = v
		}
		provider = "S3Bucket"
		return name, owner, provider
	}

	return name, owner, provider
}
