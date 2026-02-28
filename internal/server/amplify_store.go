package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type amplifyStore struct {
	mu sync.Mutex

	nextAppID        int64
	nextWebhookID    int64
	nextJobID        int64
	nextArtifactID   int64
	nextDeploymentID int64

	apps      map[string]map[string]any
	branches  map[string]map[string]map[string]any
	backends  map[string]map[string]map[string]any
	domains   map[string]map[string]map[string]any
	webhooks  map[string]map[string]any
	jobs      map[string]map[string]map[string]map[string]any
	artifacts map[string]map[string]any
	tags      map[string]map[string]string
}

func newAmplifyStore() *amplifyStore {
	s := &amplifyStore{
		nextAppID:        2,
		nextWebhookID:    2,
		nextJobID:        2,
		nextArtifactID:   2,
		nextDeploymentID: 2,
		apps:             map[string]map[string]any{},
		branches:         map[string]map[string]map[string]any{},
		backends:         map[string]map[string]map[string]any{},
		domains:          map[string]map[string]map[string]any{},
		webhooks:         map[string]map[string]any{},
		jobs:             map[string]map[string]map[string]map[string]any{},
		artifacts:        map[string]map[string]any{},
		tags:             map[string]map[string]string{},
	}
	now := time.Now().UTC()
	s.ensureAppLocked("d1234567890", now)
	s.ensureBranchLocked("d1234567890", "main", now)
	s.ensureBackendLocked("d1234567890", "dev", now)
	s.ensureDomainLocked("d1234567890", "example.com", now)
	s.ensureWebhookLocked("webhook-000001", "d1234567890", "main", now)
	s.ensureJobLocked("d1234567890", "main", "0000001", now)
	s.ensureArtifactLocked("artifact-000001", "d1234567890", "main", "0000001", now)
	s.ensureTagsLocked(amplifyAppARN("d1234567890"))
	return s
}

func (s *amplifyStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := amplifyMergeMaps(payload, pathParams, query)
	appID := amplifyString(ctx, "appId", "d1234567890")
	branchName := amplifyString(ctx, "branchName", "main")
	environmentName := amplifyString(ctx, "environmentName", "dev")
	domainName := amplifyString(ctx, "domainName", "example.com")
	jobID := amplifyString(ctx, "jobId", "0000001")
	webhookID := amplifyString(ctx, "webhookId", "webhook-000001")
	artifactID := amplifyString(ctx, "artifactId", "artifact-000001")
	resourceARN := amplifyString(ctx, "resourceArn", amplifyAppARN(appID))

	s.ensureAppLocked(appID, now)
	s.ensureBranchLocked(appID, branchName, now)
	s.ensureBackendLocked(appID, environmentName, now)
	s.ensureDomainLocked(appID, domainName, now)
	s.ensureWebhookLocked(webhookID, appID, branchName, now)
	s.ensureJobLocked(appID, branchName, jobID, now)
	s.ensureArtifactLocked(artifactID, appID, branchName, jobID, now)
	s.ensureTagsLocked(resourceARN)

	switch action {
	case "CreateApp":
		id := amplifyString(payload, "appId", "")
		if id == "" {
			id = fmt.Sprintf("d%010d", s.nextAppID)
			s.nextAppID++
		}
		app := s.ensureAppLocked(id, now)
		if name := amplifyString(payload, "name", ""); name != "" {
			app["name"] = name
		}
		if repo := amplifyString(payload, "repository", ""); repo != "" {
			app["repository"] = repo
		}
		app["updateTime"] = now.Format(time.RFC3339)
		return map[string]any{"app": amplifyCloneMap(app)}

	case "GetApp", "UpdateApp":
		app := s.ensureAppLocked(appID, now)
		if action == "UpdateApp" {
			if name := amplifyString(payload, "name", ""); name != "" {
				app["name"] = name
			}
			if repo := amplifyString(payload, "repository", ""); repo != "" {
				app["repository"] = repo
			}
			app["updateTime"] = now.Format(time.RFC3339)
		}
		return map[string]any{"app": amplifyCloneMap(app)}

	case "DeleteApp":
		app := s.ensureAppLocked(appID, now)
		delete(s.apps, appID)
		delete(s.branches, appID)
		delete(s.backends, appID)
		delete(s.domains, appID)
		for id, wh := range s.webhooks {
			if amplifyString(wh, "appId", "") == appID {
				delete(s.webhooks, id)
			}
		}
		delete(s.jobs, appID)
		delete(s.tags, amplifyString(app, "appArn", ""))
		return map[string]any{"app": amplifyCloneMap(app)}

	case "ListApps":
		items := make([]any, 0, len(s.apps))
		for _, app := range amplifySortedNestedValues(s.apps) {
			items = append(items, amplifyCloneMap(app))
		}
		return map[string]any{"apps": items, "nextToken": ""}

	case "CreateBranch", "GetBranch", "UpdateBranch", "DeleteBranch":
		branch := s.ensureBranchLocked(appID, branchName, now)
		if action == "UpdateBranch" {
			if display := amplifyString(payload, "displayName", ""); display != "" {
				branch["displayName"] = display
			}
			branch["updateTime"] = now.Format(time.RFC3339)
		}
		if action == "DeleteBranch" {
			if appBranches := s.branches[appID]; appBranches != nil {
				delete(appBranches, branchName)
			}
		}
		return map[string]any{"branch": amplifyCloneMap(branch)}

	case "ListBranches":
		items := make([]any, 0)
		for _, branch := range amplifySortedNestedValues(s.branches[appID]) {
			items = append(items, amplifyCloneMap(branch))
		}
		return map[string]any{"branches": items, "nextToken": ""}

	case "CreateBackendEnvironment", "GetBackendEnvironment", "DeleteBackendEnvironment":
		backend := s.ensureBackendLocked(appID, environmentName, now)
		if action == "DeleteBackendEnvironment" {
			if appBackends := s.backends[appID]; appBackends != nil {
				delete(appBackends, environmentName)
			}
		}
		return map[string]any{"backendEnvironment": amplifyCloneMap(backend)}

	case "ListBackendEnvironments":
		items := make([]any, 0)
		for _, backend := range amplifySortedNestedValues(s.backends[appID]) {
			items = append(items, amplifyCloneMap(backend))
		}
		return map[string]any{"backendEnvironments": items, "nextToken": ""}

	case "CreateDomainAssociation", "GetDomainAssociation", "UpdateDomainAssociation", "DeleteDomainAssociation":
		domain := s.ensureDomainLocked(appID, domainName, now)
		if action == "UpdateDomainAssociation" {
			domain["updateTime"] = now.Format(time.RFC3339)
		}
		if action == "DeleteDomainAssociation" {
			if appDomains := s.domains[appID]; appDomains != nil {
				delete(appDomains, domainName)
			}
		}
		return map[string]any{"domainAssociation": amplifyCloneMap(domain)}

	case "ListDomainAssociations":
		items := make([]any, 0)
		for _, domain := range amplifySortedNestedValues(s.domains[appID]) {
			items = append(items, amplifyCloneMap(domain))
		}
		return map[string]any{"domainAssociations": items, "nextToken": ""}

	case "CreateWebhook":
		id := fmt.Sprintf("webhook-%06d", s.nextWebhookID)
		s.nextWebhookID++
		webhook := s.ensureWebhookLocked(id, appID, branchName, now)
		return map[string]any{"webhook": amplifyCloneMap(webhook)}

	case "GetWebhook", "UpdateWebhook", "DeleteWebhook":
		webhook := s.ensureWebhookLocked(webhookID, appID, branchName, now)
		if action == "UpdateWebhook" {
			if desc := amplifyString(payload, "description", ""); desc != "" {
				webhook["description"] = desc
			}
			webhook["updateTime"] = now.Format(time.RFC3339)
		}
		if action == "DeleteWebhook" {
			delete(s.webhooks, webhookID)
		}
		return map[string]any{"webhook": amplifyCloneMap(webhook)}

	case "ListWebhooks":
		items := make([]any, 0)
		for _, webhook := range amplifySortedNestedValues(s.webhooks) {
			if amplifyString(webhook, "appId", "") != appID {
				continue
			}
			items = append(items, amplifyCloneMap(webhook))
		}
		return map[string]any{"webhooks": items, "nextToken": ""}

	case "CreateDeployment":
		id := fmt.Sprintf("%07d", s.nextDeploymentID)
		s.nextDeploymentID++
		job := s.ensureJobLocked(appID, branchName, id, now)
		job["status"] = "PENDING"
		artifact := s.ensureArtifactLocked(fmt.Sprintf("artifact-%06d", s.nextArtifactID), appID, branchName, id, now)
		s.nextArtifactID++
		return map[string]any{
			"jobId":          id,
			"zipUploadUrl":   amplifyString(artifact, "artifactUrl", ""),
			"fileUploadUrls": map[string]any{"artifact.zip": amplifyString(artifact, "artifactUrl", "")},
		}

	case "StartDeployment":
		job := s.ensureJobLocked(appID, branchName, jobID, now)
		job["status"] = "SUCCEED"
		job["endTime"] = now.Format(time.RFC3339)
		return map[string]any{"jobSummary": amplifyJobSummary(job)}

	case "StartJob":
		id := fmt.Sprintf("%07d", s.nextJobID)
		s.nextJobID++
		job := s.ensureJobLocked(appID, branchName, id, now)
		job["jobType"] = amplifyString(payload, "jobType", "RELEASE")
		job["status"] = "SUCCEED"
		job["endTime"] = now.Format(time.RFC3339)
		return map[string]any{"jobSummary": amplifyJobSummary(job)}

	case "StopJob":
		job := s.ensureJobLocked(appID, branchName, jobID, now)
		job["status"] = "CANCELLED"
		job["endTime"] = now.Format(time.RFC3339)
		return map[string]any{"jobSummary": amplifyJobSummary(job)}

	case "DeleteJob":
		job := s.ensureJobLocked(appID, branchName, jobID, now)
		if appJobs := s.jobs[appID]; appJobs != nil {
			if branchJobs := appJobs[branchName]; branchJobs != nil {
				delete(branchJobs, jobID)
			}
		}
		return map[string]any{"jobSummary": amplifyJobSummary(job)}

	case "GetJob":
		job := s.ensureJobLocked(appID, branchName, jobID, now)
		return map[string]any{"job": amplifyCloneMap(job)}

	case "ListJobs":
		items := make([]any, 0)
		if appJobs := s.jobs[appID]; appJobs != nil {
			for _, job := range amplifySortedNestedValues(appJobs[branchName]) {
				items = append(items, amplifyJobSummary(job))
			}
		}
		return map[string]any{"jobSummaries": items, "nextToken": ""}

	case "ListArtifacts":
		items := make([]any, 0)
		for _, artifact := range amplifySortedNestedValues(s.artifacts) {
			if amplifyString(artifact, "appId", "") != appID {
				continue
			}
			if amplifyString(artifact, "branchName", "") != branchName {
				continue
			}
			if amplifyString(artifact, "jobId", "") != jobID {
				continue
			}
			items = append(items, amplifyCloneMap(artifact))
		}
		return map[string]any{"artifacts": items, "nextToken": ""}

	case "GetArtifactUrl":
		artifact := s.ensureArtifactLocked(artifactID, appID, branchName, jobID, now)
		return map[string]any{
			"artifactId":  amplifyString(artifact, "artifactId", artifactID),
			"artifactUrl": amplifyString(artifact, "artifactUrl", ""),
		}

	case "GenerateAccessLogs":
		return map[string]any{"logUrl": "https://stackyard.local/amplify/apps/" + appID + "/access-logs.txt"}

	case "TagResource":
		s.upsertTagsLocked(resourceARN, amplifyMapString(payload["tags"]))
		return map[string]any{}

	case "UntagResource":
		keys := amplifyTagKeys(ctx["tagKeys"])
		if len(keys) == 0 {
			keys = amplifyTagKeys(payload["tagKeys"])
		}
		tagMap := s.ensureTagsLocked(resourceARN)
		for _, key := range keys {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": amplifyCloneStringMap(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{"action": action}
}

func (s *amplifyStore) ensureAppLocked(appID string, now time.Time) map[string]any {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		appID = "d1234567890"
	}
	if app := s.apps[appID]; app != nil {
		return app
	}
	timestamp := now.Format(time.RFC3339)
	app := map[string]any{
		"appId":                 appID,
		"appArn":                amplifyAppARN(appID),
		"name":                  "stackyard-amplify-app",
		"description":           "stackyard amplify app",
		"repository":            "https://example.com/stackyard/amplify.git",
		"platform":              "WEB",
		"enableBranchAutoBuild": true,
		"createTime":            timestamp,
		"updateTime":            timestamp,
	}
	s.apps[appID] = app
	s.ensureTagsLocked(amplifyAppARN(appID))
	return app
}

func (s *amplifyStore) ensureBranchLocked(appID, branchName string, now time.Time) map[string]any {
	branchName = strings.TrimSpace(branchName)
	if branchName == "" {
		branchName = "main"
	}
	if s.branches[appID] == nil {
		s.branches[appID] = map[string]map[string]any{}
	}
	if branch := s.branches[appID][branchName]; branch != nil {
		return branch
	}
	timestamp := now.Format(time.RFC3339)
	branch := map[string]any{
		"branchArn":       fmt.Sprintf("arn:aws:amplify:us-east-1:123456789012:apps/%s/branches/%s", appID, branchName),
		"branchName":      branchName,
		"displayName":     branchName,
		"stage":           "PRODUCTION",
		"activeJobId":     "0000001",
		"enableAutoBuild": true,
		"createTime":      timestamp,
		"updateTime":      timestamp,
	}
	s.branches[appID][branchName] = branch
	return branch
}

func (s *amplifyStore) ensureBackendLocked(appID, environmentName string, now time.Time) map[string]any {
	environmentName = strings.TrimSpace(environmentName)
	if environmentName == "" {
		environmentName = "dev"
	}
	if s.backends[appID] == nil {
		s.backends[appID] = map[string]map[string]any{}
	}
	if backend := s.backends[appID][environmentName]; backend != nil {
		return backend
	}
	timestamp := now.Format(time.RFC3339)
	backend := map[string]any{
		"backendEnvironmentArn": fmt.Sprintf("arn:aws:amplify:us-east-1:123456789012:apps/%s/backendenvironments/%s", appID, environmentName),
		"environmentName":       environmentName,
		"stackName":             "amplify-" + appID + "-" + environmentName,
		"deploymentArtifacts":   "s3://stackyard-amplify/" + appID + "/" + environmentName + "/build.zip",
		"createTime":            timestamp,
		"updateTime":            timestamp,
	}
	s.backends[appID][environmentName] = backend
	return backend
}

func (s *amplifyStore) ensureDomainLocked(appID, domainName string, now time.Time) map[string]any {
	domainName = strings.TrimSpace(domainName)
	if domainName == "" {
		domainName = "example.com"
	}
	if s.domains[appID] == nil {
		s.domains[appID] = map[string]map[string]any{}
	}
	if domain := s.domains[appID][domainName]; domain != nil {
		return domain
	}
	timestamp := now.Format(time.RFC3339)
	domain := map[string]any{
		"domainAssociationArn": fmt.Sprintf("arn:aws:amplify:us-east-1:123456789012:apps/%s/domains/%s", appID, domainName),
		"domainName":           domainName,
		"appId":                appID,
		"enableAutoSubDomain":  false,
		"domainStatus":         "AVAILABLE",
		"createTime":           timestamp,
		"updateTime":           timestamp,
		"subDomains": []any{
			map[string]any{
				"verified": true,
				"subDomainSetting": map[string]any{
					"branchName": "main",
					"prefix":     "",
				},
			},
		},
	}
	s.domains[appID][domainName] = domain
	return domain
}

func (s *amplifyStore) ensureWebhookLocked(webhookID, appID, branchName string, now time.Time) map[string]any {
	webhookID = strings.TrimSpace(webhookID)
	if webhookID == "" {
		webhookID = "webhook-000001"
	}
	if webhook := s.webhooks[webhookID]; webhook != nil {
		return webhook
	}
	timestamp := now.Format(time.RFC3339)
	webhook := map[string]any{
		"webhookArn":  fmt.Sprintf("arn:aws:amplify:us-east-1:123456789012:apps/%s/webhooks/%s", appID, webhookID),
		"webhookId":   webhookID,
		"webhookUrl":  "https://stackyard.local/amplify/webhooks/" + webhookID,
		"branchName":  branchName,
		"description": "stackyard webhook",
		"appId":       appID,
		"createTime":  timestamp,
		"updateTime":  timestamp,
	}
	s.webhooks[webhookID] = webhook
	return webhook
}

func (s *amplifyStore) ensureJobLocked(appID, branchName, jobID string, now time.Time) map[string]any {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		jobID = "0000001"
	}
	if s.jobs[appID] == nil {
		s.jobs[appID] = map[string]map[string]map[string]any{}
	}
	if s.jobs[appID][branchName] == nil {
		s.jobs[appID][branchName] = map[string]map[string]any{}
	}
	if job := s.jobs[appID][branchName][jobID]; job != nil {
		return job
	}
	timestamp := now.Format(time.RFC3339)
	job := map[string]any{
		"jobArn":        fmt.Sprintf("arn:aws:amplify:us-east-1:123456789012:apps/%s/branches/%s/jobs/%s", appID, branchName, jobID),
		"jobId":         jobID,
		"appId":         appID,
		"branchName":    branchName,
		"jobType":       "RELEASE",
		"status":        "SUCCEED",
		"sourceUrl":     "https://example.com/stackyard/amplify/source.zip",
		"commitId":      "commit-000001",
		"commitMessage": "stackyard seeded job",
		"commitTime":    timestamp,
		"startTime":     timestamp,
		"endTime":       timestamp,
	}
	s.jobs[appID][branchName][jobID] = job
	return job
}

func (s *amplifyStore) ensureArtifactLocked(artifactID, appID, branchName, jobID string, now time.Time) map[string]any {
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		artifactID = "artifact-000001"
	}
	if artifact := s.artifacts[artifactID]; artifact != nil {
		return artifact
	}
	artifact := map[string]any{
		"artifactId":       artifactID,
		"artifactFileName": "artifact-" + jobID + ".zip",
		"artifactUrl":      "https://stackyard.local/amplify/artifacts/" + artifactID + ".zip",
		"appId":            appID,
		"branchName":       branchName,
		"jobId":            jobID,
		"createdTime":      now.Format(time.RFC3339),
	}
	s.artifacts[artifactID] = artifact
	return artifact
}

func (s *amplifyStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = amplifyAppARN("d1234567890")
	}
	if tags := s.tags[resourceARN]; tags != nil {
		return tags
	}
	s.tags[resourceARN] = map[string]string{"stackyard": "true"}
	return s.tags[resourceARN]
}

func (s *amplifyStore) upsertTagsLocked(resourceARN string, values map[string]string) {
	if len(values) == 0 {
		return
	}
	tagMap := s.ensureTagsLocked(resourceARN)
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		tagMap[key] = value
	}
}

func amplifyAppARN(appID string) string {
	return "arn:aws:amplify:us-east-1:123456789012:apps/" + strings.TrimSpace(appID)
}

func amplifyMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := amplifyCloneMap(payload)
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		out[key] = values[len(values)-1]
	}
	return out
}

func amplifyString(src map[string]any, key, def string) string {
	if src != nil {
		if value, ok := src[key]; ok && value != nil {
			switch v := value.(type) {
			case string:
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return def
}

func amplifyMapString(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, raw := range v {
			key = strings.TrimSpace(key)
			if key == "" || raw == nil {
				continue
			}
			out[key] = strings.TrimSpace(fmt.Sprintf("%v", raw))
		}
	case map[string]string:
		for key, raw := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(raw)
		}
	}
	return out
}

func amplifyTagKeys(value any) []string {
	switch v := value.(type) {
	case string:
		items := strings.Split(v, ",")
		out := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			str := strings.TrimSpace(fmt.Sprintf("%v", item))
			if str != "" {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

func amplifySortedNestedValues[T any](items map[string]T) []T {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]T, 0, len(keys))
	for _, key := range keys {
		out = append(out, items[key])
	}
	return out
}

func amplifyJobSummary(job map[string]any) map[string]any {
	return map[string]any{
		"jobArn":     amplifyString(job, "jobArn", ""),
		"jobId":      amplifyString(job, "jobId", ""),
		"jobType":    amplifyString(job, "jobType", "RELEASE"),
		"status":     amplifyString(job, "status", "SUCCEED"),
		"commitId":   amplifyString(job, "commitId", ""),
		"commitTime": amplifyString(job, "commitTime", ""),
		"startTime":  amplifyString(job, "startTime", ""),
		"endTime":    amplifyString(job, "endTime", ""),
	}
}

func amplifyCloneMap(src map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range src {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = amplifyCloneMap(typed)
		case map[string]string:
			tmp := map[string]any{}
			for mk, mv := range typed {
				tmp[mk] = mv
			}
			out[key] = tmp
		case []any:
			copied := make([]any, len(typed))
			for i, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					copied[i] = amplifyCloneMap(nested)
				} else {
					copied[i] = item
				}
			}
			out[key] = copied
		default:
			out[key] = value
		}
	}
	return out
}

func amplifyCloneStringMap(src map[string]string) map[string]any {
	out := map[string]any{}
	for key, value := range src {
		out[key] = value
	}
	return out
}
