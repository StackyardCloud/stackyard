package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type emrServerlessStore struct {
	mu sync.Mutex

	nextApplicationID int64
	nextJobRunID      int64

	applications map[string]*emrServerlessApplication
	jobRuns      map[string]map[string]*emrServerlessJobRun
	tags         map[string]map[string]string
}

type emrServerlessApplication struct {
	ID           string
	ARN          string
	Name         string
	ReleaseLabel string
	Type         string
	State        string
	CreatedAt    string
	UpdatedAt    string
}

type emrServerlessJobRun struct {
	ID               string
	ARN              string
	ApplicationID    string
	Name             string
	State            string
	ExecutionRoleARN string
	ReleaseLabel     string
	Mode             string
	Attempt          int
	CreatedAt        string
	UpdatedAt        string
}

func newEMRServerlessStore() *emrServerlessStore {
	now := time.Now().UTC().Format(time.RFC3339)
	app := &emrServerlessApplication{
		ID:           "app-000001",
		ARN:          emrServerlessApplicationARN("app-000001"),
		Name:         "stackyard-emr-serverless-app",
		ReleaseLabel: "emr-7.0.0",
		Type:         "SPARK",
		State:        "CREATED",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	job := &emrServerlessJobRun{
		ID:               "jobrun-000001",
		ARN:              emrServerlessJobRunARN(app.ID, "jobrun-000001"),
		ApplicationID:    app.ID,
		Name:             "stackyard-emr-serverless-job",
		State:            "SUCCESS",
		ExecutionRoleARN: "arn:aws:iam::123456789012:role/stackyard-emr-serverless-execution-role",
		ReleaseLabel:     app.ReleaseLabel,
		Mode:             "BATCH",
		Attempt:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	tags := map[string]map[string]string{
		app.ARN: {"stackyard": "true", "service": "emrserverless"},
		job.ARN: {"stackyard": "true", "service": "emrserverless"},
	}

	return &emrServerlessStore{
		nextApplicationID: 2,
		nextJobRunID:      2,
		applications: map[string]*emrServerlessApplication{
			app.ID: app,
		},
		jobRuns: map[string]map[string]*emrServerlessJobRun{
			app.ID: {
				job.ID: job,
			},
		},
		tags: tags,
	}
}

func (s *emrServerlessStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := emrServerlessMergeMaps(payload, pathParams, query)
	applicationID := emrServerlessStringAny(ctx, []string{"applicationId", "id"}, "app-000001")
	if applicationID == "" {
		applicationID = "app-000001"
	}
	jobRunID := emrServerlessStringAny(ctx, []string{"jobRunId", "id"}, "jobrun-000001")
	if jobRunID == "" {
		jobRunID = "jobrun-000001"
	}
	resourceARN := emrServerlessStringAny(ctx, []string{"resourceArn"}, emrServerlessApplicationARN(applicationID))

	app := s.ensureApplicationLocked(applicationID)
	_ = app

	switch action {
	case "CreateApplication":
		now := time.Now().UTC().Format(time.RFC3339)
		id := s.nextTokenLocked("app", 6)
		created := &emrServerlessApplication{
			ID:           id,
			ARN:          emrServerlessApplicationARN(id),
			Name:         emrServerlessStringAny(payload, []string{"name"}, "stackyard-emr-serverless-app"),
			ReleaseLabel: emrServerlessStringAny(payload, []string{"releaseLabel"}, "emr-7.0.0"),
			Type:         emrServerlessStringAny(payload, []string{"type"}, "SPARK"),
			State:        "CREATED",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		s.applications[id] = created
		s.ensureTagSetLocked(created.ARN)
		for key, value := range emrServerlessTagsFromAny(payload["tags"]) {
			s.tags[created.ARN][key] = value
		}
		return map[string]any{"applicationId": created.ID, "arn": created.ARN, "name": created.Name}

	case "DeleteApplication":
		application := s.ensureApplicationLocked(applicationID)
		application.State = "TERMINATED"
		application.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "GetApplication":
		application := s.ensureApplicationLocked(applicationID)
		return map[string]any{"application": s.applicationPayload(application)}

	case "ListApplications":
		items := make([]any, 0, len(s.applications))
		for _, id := range s.sortedApplicationIDsLocked() {
			application := s.applications[id]
			items = append(items, map[string]any{
				"id":           application.ID,
				"arn":          application.ARN,
				"name":         application.Name,
				"releaseLabel": application.ReleaseLabel,
				"type":         application.Type,
				"state":        application.State,
				"createdAt":    application.CreatedAt,
				"updatedAt":    application.UpdatedAt,
			})
		}
		return map[string]any{"applications": items, "nextToken": ""}

	case "UpdateApplication":
		application := s.ensureApplicationLocked(applicationID)
		if name := emrServerlessStringAny(payload, []string{"name"}, ""); name != "" {
			application.Name = name
		}
		if releaseLabel := emrServerlessStringAny(payload, []string{"releaseLabel"}, ""); releaseLabel != "" {
			application.ReleaseLabel = releaseLabel
		}
		if appType := emrServerlessStringAny(payload, []string{"type"}, ""); appType != "" {
			application.Type = appType
		}
		application.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "StartApplication":
		application := s.ensureApplicationLocked(applicationID)
		application.State = "STARTED"
		application.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "StopApplication":
		application := s.ensureApplicationLocked(applicationID)
		application.State = "STOPPED"
		application.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "StartJobRun":
		application := s.ensureApplicationLocked(applicationID)
		now := time.Now().UTC().Format(time.RFC3339)
		id := s.nextTokenLocked("jobrun", 6)
		jobRun := &emrServerlessJobRun{
			ID:               id,
			ARN:              emrServerlessJobRunARN(application.ID, id),
			ApplicationID:    application.ID,
			Name:             emrServerlessStringAny(payload, []string{"name"}, "stackyard-emr-serverless-job"),
			State:            "SUBMITTED",
			ExecutionRoleARN: emrServerlessStringAny(payload, []string{"executionRoleArn"}, "arn:aws:iam::123456789012:role/stackyard-emr-serverless-execution-role"),
			ReleaseLabel:     emrServerlessStringAny(payload, []string{"releaseLabel"}, application.ReleaseLabel),
			Mode:             emrServerlessStringAny(payload, []string{"mode"}, "BATCH"),
			Attempt:          1,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		s.ensureJobMapLocked(application.ID)
		s.jobRuns[application.ID][id] = jobRun
		s.ensureTagSetLocked(jobRun.ARN)
		for key, value := range emrServerlessTagsFromAny(payload["tags"]) {
			s.tags[jobRun.ARN][key] = value
		}
		return map[string]any{"applicationId": application.ID, "arn": jobRun.ARN, "jobRunId": jobRun.ID}

	case "GetJobRun":
		jobRun := s.ensureJobRunLocked(applicationID, jobRunID)
		return map[string]any{"jobRun": s.jobRunPayload(jobRun)}

	case "ListJobRuns":
		s.ensureApplicationLocked(applicationID)
		items := make([]any, 0)
		for _, id := range s.sortedJobRunIDsLocked(applicationID) {
			jobRun := s.jobRuns[applicationID][id]
			items = append(items, map[string]any{
				"id":            jobRun.ID,
				"arn":           jobRun.ARN,
				"applicationId": jobRun.ApplicationID,
				"name":          jobRun.Name,
				"state":         jobRun.State,
				"attempt":       jobRun.Attempt,
				"createdAt":     jobRun.CreatedAt,
				"updatedAt":     jobRun.UpdatedAt,
			})
		}
		return map[string]any{"jobRuns": items, "nextToken": ""}

	case "ListJobRunAttempts":
		jobRun := s.ensureJobRunLocked(applicationID, jobRunID)
		attempts := []any{map[string]any{
			"applicationId": jobRun.ApplicationID,
			"jobRunId":      jobRun.ID,
			"attempt":       jobRun.Attempt,
			"state":         jobRun.State,
			"createdAt":     jobRun.CreatedAt,
			"updatedAt":     jobRun.UpdatedAt,
		}}
		return map[string]any{"jobRunAttempts": attempts, "nextToken": ""}

	case "CancelJobRun":
		jobRun := s.ensureJobRunLocked(applicationID, jobRunID)
		jobRun.State = "CANCELLING"
		jobRun.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{}

	case "GetDashboardForJobRun":
		jobRun := s.ensureJobRunLocked(applicationID, jobRunID)
		return map[string]any{"url": "https://console.aws.amazon.com/emr/home?region=us-east-1#serverless-dashboard/" + jobRun.ApplicationID + "/" + jobRun.ID}

	case "TagResource":
		tagSet := s.ensureTagSetLocked(resourceARN)
		for key, value := range emrServerlessTagsFromAny(payload["tags"]) {
			tagSet[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		tagSet := s.ensureTagSetLocked(resourceARN)
		for _, key := range emrServerlessTagKeys(ctx, query) {
			delete(tagSet, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": emrServerlessCloneStringMap(s.ensureTagSetLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *emrServerlessStore) ensureApplicationLocked(applicationID string) *emrServerlessApplication {
	id := strings.TrimSpace(applicationID)
	if id == "" {
		id = "app-000001"
	}
	if existing, ok := s.applications[id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &emrServerlessApplication{
		ID:           id,
		ARN:          emrServerlessApplicationARN(id),
		Name:         "stackyard-emr-serverless-app",
		ReleaseLabel: "emr-7.0.0",
		Type:         "SPARK",
		State:        "CREATED",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	s.applications[id] = created
	s.ensureTagSetLocked(created.ARN)
	return created
}

func (s *emrServerlessStore) ensureJobMapLocked(applicationID string) {
	if s.jobRuns[applicationID] == nil {
		s.jobRuns[applicationID] = map[string]*emrServerlessJobRun{}
	}
}

func (s *emrServerlessStore) ensureJobRunLocked(applicationID, jobRunID string) *emrServerlessJobRun {
	app := s.ensureApplicationLocked(applicationID)
	s.ensureJobMapLocked(app.ID)
	id := strings.TrimSpace(jobRunID)
	if id == "" {
		id = "jobrun-000001"
	}
	if existing, ok := s.jobRuns[app.ID][id]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	created := &emrServerlessJobRun{
		ID:               id,
		ARN:              emrServerlessJobRunARN(app.ID, id),
		ApplicationID:    app.ID,
		Name:             "stackyard-emr-serverless-job",
		State:            "SUCCESS",
		ExecutionRoleARN: "arn:aws:iam::123456789012:role/stackyard-emr-serverless-execution-role",
		ReleaseLabel:     app.ReleaseLabel,
		Mode:             "BATCH",
		Attempt:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.jobRuns[app.ID][id] = created
	s.ensureTagSetLocked(created.ARN)
	return created
}

func (s *emrServerlessStore) ensureTagSetLocked(resourceARN string) map[string]string {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = emrServerlessApplicationARN("app-000001")
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{"stackyard": "true", "service": "emrserverless"}
	}
	return s.tags[arn]
}

func (s *emrServerlessStore) applicationPayload(application *emrServerlessApplication) map[string]any {
	if application == nil {
		application = s.ensureApplicationLocked("app-000001")
	}
	return map[string]any{
		"id":           application.ID,
		"arn":          application.ARN,
		"name":         application.Name,
		"releaseLabel": application.ReleaseLabel,
		"type":         application.Type,
		"state":        application.State,
		"createdAt":    application.CreatedAt,
		"updatedAt":    application.UpdatedAt,
	}
}

func (s *emrServerlessStore) jobRunPayload(jobRun *emrServerlessJobRun) map[string]any {
	if jobRun == nil {
		jobRun = s.ensureJobRunLocked("app-000001", "jobrun-000001")
	}
	return map[string]any{
		"id":            jobRun.ID,
		"arn":           jobRun.ARN,
		"applicationId": jobRun.ApplicationID,
		"name":          jobRun.Name,
		"state":         jobRun.State,
		"attempt":       jobRun.Attempt,
		"executionRole": jobRun.ExecutionRoleARN,
		"releaseLabel":  jobRun.ReleaseLabel,
		"mode":          jobRun.Mode,
		"createdAt":     jobRun.CreatedAt,
		"updatedAt":     jobRun.UpdatedAt,
	}
}

func (s *emrServerlessStore) sortedApplicationIDsLocked() []string {
	keys := make([]string, 0, len(s.applications))
	for key := range s.applications {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrServerlessStore) sortedJobRunIDsLocked(applicationID string) []string {
	jobMap := s.jobRuns[applicationID]
	if jobMap == nil {
		return nil
	}
	keys := make([]string, 0, len(jobMap))
	for key := range jobMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *emrServerlessStore) nextTokenLocked(prefix string, width int) string {
	if prefix == "app" {
		id := s.nextApplicationID
		s.nextApplicationID++
		return fmt.Sprintf("%s-%0*d", prefix, width, id)
	}
	id := s.nextJobRunID
	s.nextJobRunID++
	return fmt.Sprintf("%s-%0*d", prefix, width, id)
}

func emrServerlessMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			out[key] = values[0]
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}
	return out
}

func emrServerlessStringAny(values map[string]any, keys []string, def string) string {
	for _, key := range keys {
		for existingKey, value := range values {
			if !strings.EqualFold(existingKey, key) {
				continue
			}
			switch v := value.(type) {
			case string:
				if out := strings.TrimSpace(v); out != "" {
					return out
				}
			case []string:
				if len(v) > 0 {
					if out := strings.TrimSpace(v[0]); out != "" {
						return out
					}
				}
			case []any:
				if len(v) > 0 {
					if out := strings.TrimSpace(fmt.Sprint(v[0])); out != "" {
						return out
					}
				}
			default:
				if value != nil {
					if out := strings.TrimSpace(fmt.Sprint(value)); out != "" {
						return out
					}
				}
			}
		}
	}
	return def
}

func emrServerlessStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if token := strings.TrimSpace(item); token != "" {
				out = append(out, token)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if token := strings.TrimSpace(fmt.Sprint(item)); token != "" {
				out = append(out, token)
			}
		}
		return out
	case string:
		value := strings.TrimSpace(v)
		if value == "" {
			return nil
		}
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if token := strings.TrimSpace(part); token != "" {
				out = append(out, token)
			}
		}
		return out
	default:
		if raw == nil {
			return nil
		}
		value := strings.TrimSpace(fmt.Sprint(raw))
		if value == "" {
			return nil
		}
		return []string{value}
	}
}

func emrServerlessTagsFromAny(raw any) map[string]string {
	out := map[string]string{}
	switch v := raw.(type) {
	case map[string]string:
		for key, value := range v {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(value)
		}
	case map[string]any:
		for key, value := range v {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprint(value))
		}
	case []any:
		for _, item := range v {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := emrServerlessStringAny(tag, []string{"key", "Key"}, "")
			if key == "" {
				continue
			}
			out[key] = emrServerlessStringAny(tag, []string{"value", "Value"}, "")
		}
	}
	return out
}

func emrServerlessTagKeys(payload map[string]any, query url.Values) []string {
	if keys := emrServerlessStringSlice(payload["tagKeys"]); len(keys) > 0 {
		return keys
	}
	if keys := emrServerlessStringSlice(payload["TagKeys"]); len(keys) > 0 {
		return keys
	}
	if values, ok := query["tagKeys"]; ok && len(values) > 0 {
		keys := []string{}
		for _, value := range values {
			keys = append(keys, emrServerlessStringSlice(value)...)
		}
		if len(keys) > 0 {
			return keys
		}
	}
	if values, ok := query["TagKeys"]; ok && len(values) > 0 {
		keys := []string{}
		for _, value := range values {
			keys = append(keys, emrServerlessStringSlice(value)...)
		}
		if len(keys) > 0 {
			return keys
		}
	}
	return nil
}

func emrServerlessCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func emrServerlessApplicationARN(applicationID string) string {
	id := strings.TrimSpace(applicationID)
	if id == "" {
		id = "app-000001"
	}
	return "arn:aws:emr-serverless:us-east-1:123456789012:/applications/" + id
}

func emrServerlessJobRunARN(applicationID, jobRunID string) string {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-000001"
	}
	jobID := strings.TrimSpace(jobRunID)
	if jobID == "" {
		jobID = "jobrun-000001"
	}
	return "arn:aws:emr-serverless:us-east-1:123456789012:/applications/" + appID + "/jobruns/" + jobID
}
