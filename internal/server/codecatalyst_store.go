package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type codeCatalystStore struct {
	mu sync.Mutex

	nextAccessTokenID int64
	nextProjectID     int64
	nextRepoID        int64
	nextDevEnvID      int64
	nextSessionID     int64
	nextWorkflowRunID int64

	spaces       map[string]map[string]any
	projects     map[string]map[string]any
	repositories map[string]map[string]any
	branches     map[string]map[string]any
	devEnvs      map[string]map[string]any
	sessions     map[string]map[string]any
	workflows    map[string]map[string]any
	workflowRuns map[string]map[string]any
	accessTokens map[string]map[string]any
	eventLogs    map[string][]map[string]any
}

func newCodeCatalystStore() *codeCatalystStore {
	s := &codeCatalystStore{
		nextAccessTokenID: 2,
		nextProjectID:     2,
		nextRepoID:        2,
		nextDevEnvID:      2,
		nextSessionID:     2,
		nextWorkflowRunID: 2,
		spaces:            map[string]map[string]any{},
		projects:          map[string]map[string]any{},
		repositories:      map[string]map[string]any{},
		branches:          map[string]map[string]any{},
		devEnvs:           map[string]map[string]any{},
		sessions:          map[string]map[string]any{},
		workflows:         map[string]map[string]any{},
		workflowRuns:      map[string]map[string]any{},
		accessTokens:      map[string]map[string]any{},
		eventLogs:         map[string][]map[string]any{},
	}
	s.ensureSeedDataLocked()
	return s
}

func (s *codeCatalystStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedDataLocked()
	now := time.Now().UTC()
	ctx := codeCatalystMergeMaps(payload, pathParams, query)

	spaceName := codeCatalystString(ctx, "spaceName", "stackyard-space")
	projectName := codeCatalystString(ctx, "projectName", codeCatalystString(ctx, "name", "stackyard-project"))
	repoName := codeCatalystString(ctx, "sourceRepositoryName", codeCatalystString(ctx, "name", "stackyard-repository"))
	devEnvID := codeCatalystString(ctx, "id", codeCatalystString(ctx, "devEnvironmentId", "dev-env-000001"))
	sessionID := codeCatalystString(ctx, "sessionId", "session-000001")
	workflowID := codeCatalystString(ctx, "id", "workflow-000001")
	workflowRunID := codeCatalystString(ctx, "id", "workflow-run-000001")
	accessTokenID := codeCatalystString(ctx, "id", "at-000001")

	s.ensureSpaceLocked(spaceName, now)
	s.ensureProjectLocked(spaceName, projectName, now)

	s.appendEventLocked(spaceName, action, now)

	switch action {
	case "ListSpaces":
		items := make([]any, 0, len(s.spaces))
		for _, name := range codeCatalystSortedKeys(s.spaces) {
			space := s.spaces[name]
			items = append(items, map[string]any{
				"name":            codeCatalystString(space, "name", name),
				"displayName":     codeCatalystString(space, "displayName", name),
				"description":     codeCatalystString(space, "description", ""),
				"createdTime":     codeCatalystString(space, "createdTime", now.Format(time.RFC3339)),
				"lastUpdatedTime": codeCatalystString(space, "lastUpdatedTime", now.Format(time.RFC3339)),
			})
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "GetSpace":
		space := s.ensureSpaceLocked(spaceName, now)
		return codeCatalystCloneMap(space)

	case "UpdateSpace":
		space := s.ensureSpaceLocked(spaceName, now)
		if v := codeCatalystString(ctx, "displayName", ""); v != "" {
			space["displayName"] = v
		}
		if v := codeCatalystString(ctx, "description", ""); v != "" {
			space["description"] = v
		}
		space["lastUpdatedTime"] = now.Format(time.RFC3339)
		return codeCatalystCloneMap(space)

	case "DeleteSpace":
		delete(s.spaces, spaceName)
		for key := range s.projects {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.projects, key)
			}
		}
		for key := range s.repositories {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.repositories, key)
			}
		}
		for key := range s.branches {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.branches, key)
			}
		}
		for key := range s.devEnvs {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.devEnvs, key)
			}
		}
		for key := range s.sessions {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.sessions, key)
			}
		}
		for key := range s.workflows {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.workflows, key)
			}
		}
		for key := range s.workflowRuns {
			if strings.HasPrefix(key, codeCatalystSpacePrefix(spaceName)) {
				delete(s.workflowRuns, key)
			}
		}
		delete(s.eventLogs, spaceName)
		return map[string]any{}

	case "CreateProject":
		name := codeCatalystString(ctx, "name", fmt.Sprintf("project-%06d", s.nextProjectID))
		project := s.ensureProjectLocked(spaceName, name, now)
		if v := codeCatalystString(ctx, "displayName", ""); v != "" {
			project["displayName"] = v
		}
		if v := codeCatalystString(ctx, "description", ""); v != "" {
			project["description"] = v
		}
		project["lastUpdatedTime"] = now.Format(time.RFC3339)
		s.nextProjectID++
		return map[string]any{"spaceName": spaceName, "name": name}

	case "GetProject":
		name := codeCatalystString(ctx, "name", projectName)
		project := s.ensureProjectLocked(spaceName, name, now)
		return codeCatalystCloneMap(project)

	case "ListProjects":
		items := []any{}
		prefix := codeCatalystProjectPrefix(spaceName)
		for _, key := range codeCatalystSortedKeys(s.projects) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.projects[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "UpdateProject":
		name := codeCatalystString(ctx, "name", projectName)
		project := s.ensureProjectLocked(spaceName, name, now)
		if v := codeCatalystString(ctx, "displayName", ""); v != "" {
			project["displayName"] = v
		}
		if v := codeCatalystString(ctx, "description", ""); v != "" {
			project["description"] = v
		}
		project["lastUpdatedTime"] = now.Format(time.RFC3339)
		return codeCatalystCloneMap(project)

	case "DeleteProject":
		name := codeCatalystString(ctx, "name", projectName)
		projectKey := codeCatalystProjectKey(spaceName, name)
		delete(s.projects, projectKey)

		resourcePrefix := codeCatalystProjectResourcePrefix(spaceName, name)
		for key := range s.repositories {
			if strings.HasPrefix(key, resourcePrefix) {
				delete(s.repositories, key)
			}
		}
		for key := range s.branches {
			if strings.HasPrefix(key, resourcePrefix) {
				delete(s.branches, key)
			}
		}
		for key := range s.devEnvs {
			if strings.HasPrefix(key, resourcePrefix) {
				delete(s.devEnvs, key)
			}
		}
		for key := range s.sessions {
			if strings.HasPrefix(key, resourcePrefix) {
				delete(s.sessions, key)
			}
		}
		for key := range s.workflows {
			if strings.HasPrefix(key, resourcePrefix) {
				delete(s.workflows, key)
			}
		}
		for key := range s.workflowRuns {
			if strings.HasPrefix(key, resourcePrefix) {
				delete(s.workflowRuns, key)
			}
		}
		return map[string]any{}

	case "CreateSourceRepository":
		name := codeCatalystString(ctx, "name", repoName)
		repo := s.ensureRepositoryLocked(spaceName, projectName, name, now)
		if v := codeCatalystString(ctx, "description", ""); v != "" {
			repo["description"] = v
		}
		repo["lastUpdatedTime"] = now.Format(time.RFC3339)
		s.nextRepoID++
		return map[string]any{"spaceName": spaceName, "projectName": projectName, "name": name}

	case "GetSourceRepository":
		name := codeCatalystString(ctx, "name", repoName)
		repo := s.ensureRepositoryLocked(spaceName, projectName, name, now)
		return codeCatalystCloneMap(repo)

	case "ListSourceRepositories":
		items := []any{}
		prefix := codeCatalystRepositoryPrefix(spaceName, projectName)
		for _, key := range codeCatalystSortedKeys(s.repositories) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.repositories[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "DeleteSourceRepository":
		name := codeCatalystString(ctx, "name", repoName)
		repoKey := codeCatalystRepositoryKey(spaceName, projectName, name)
		delete(s.repositories, repoKey)
		branchPrefix := codeCatalystBranchPrefix(spaceName, projectName, name)
		for key := range s.branches {
			if strings.HasPrefix(key, branchPrefix) {
				delete(s.branches, key)
			}
		}
		return map[string]any{}

	case "CreateSourceRepositoryBranch":
		repositoryName := codeCatalystString(ctx, "sourceRepositoryName", repoName)
		name := codeCatalystString(ctx, "name", "main")
		branch := s.ensureBranchLocked(spaceName, projectName, repositoryName, name, now)
		branch["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{"name": name, "sourceRepositoryName": repositoryName}

	case "ListSourceRepositoryBranches":
		repositoryName := codeCatalystString(ctx, "sourceRepositoryName", repoName)
		items := []any{}
		prefix := codeCatalystBranchPrefix(spaceName, projectName, repositoryName)
		for _, key := range codeCatalystSortedKeys(s.branches) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.branches[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "GetSourceRepositoryCloneUrls":
		repositoryName := codeCatalystString(ctx, "sourceRepositoryName", repoName)
		s.ensureRepositoryLocked(spaceName, projectName, repositoryName, now)
		base := fmt.Sprintf("https://codecatalyst.aws/spaces/%s/projects/%s/source-repositories/%s", spaceName, projectName, repositoryName)
		return map[string]any{
			"https": base,
			"ssh":   fmt.Sprintf("ssh://git@codecatalyst.aws/v1/spaces/%s/projects/%s/sourceRepositories/%s", spaceName, projectName, repositoryName),
		}

	case "CreateDevEnvironment":
		id := codeCatalystString(ctx, "id", fmt.Sprintf("dev-env-%06d", s.nextDevEnvID))
		devEnv := s.ensureDevEnvironmentLocked(spaceName, projectName, id, now)
		if v := codeCatalystString(ctx, "alias", ""); v != "" {
			devEnv["alias"] = v
		}
		if v := codeCatalystString(ctx, "instanceType", ""); v != "" {
			devEnv["instanceType"] = v
		}
		devEnv["lastUpdatedTime"] = now.Format(time.RFC3339)
		s.nextDevEnvID++
		return map[string]any{"id": id, "spaceName": spaceName, "projectName": projectName, "status": codeCatalystString(devEnv, "status", "STOPPED")}

	case "GetDevEnvironment":
		devEnv := s.ensureDevEnvironmentLocked(spaceName, projectName, devEnvID, now)
		return codeCatalystCloneMap(devEnv)

	case "ListDevEnvironments":
		items := []any{}
		prefix := codeCatalystSpacePrefix(spaceName)
		for _, key := range codeCatalystSortedKeys(s.devEnvs) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.devEnvs[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "UpdateDevEnvironment":
		devEnv := s.ensureDevEnvironmentLocked(spaceName, projectName, devEnvID, now)
		if v := codeCatalystString(ctx, "alias", ""); v != "" {
			devEnv["alias"] = v
		}
		if v := codeCatalystString(ctx, "instanceType", ""); v != "" {
			devEnv["instanceType"] = v
		}
		devEnv["lastUpdatedTime"] = now.Format(time.RFC3339)
		return codeCatalystCloneMap(devEnv)

	case "DeleteDevEnvironment":
		id := codeCatalystString(ctx, "id", devEnvID)
		delete(s.devEnvs, codeCatalystDevEnvironmentKey(spaceName, projectName, id))
		sessionPrefix := codeCatalystSessionPrefix(spaceName, projectName, id)
		for key := range s.sessions {
			if strings.HasPrefix(key, sessionPrefix) {
				delete(s.sessions, key)
			}
		}
		return map[string]any{}

	case "StartDevEnvironment":
		devEnv := s.ensureDevEnvironmentLocked(spaceName, projectName, devEnvID, now)
		devEnv["status"] = "RUNNING"
		devEnv["startedTime"] = now.Format(time.RFC3339)
		devEnv["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{"id": codeCatalystString(devEnv, "id", devEnvID), "status": "RUNNING"}

	case "StopDevEnvironment":
		devEnv := s.ensureDevEnvironmentLocked(spaceName, projectName, devEnvID, now)
		devEnv["status"] = "STOPPED"
		devEnv["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{"id": codeCatalystString(devEnv, "id", devEnvID), "status": "STOPPED"}

	case "StartDevEnvironmentSession":
		devEnv := s.ensureDevEnvironmentLocked(spaceName, projectName, devEnvID, now)
		sessionID = fmt.Sprintf("session-%06d", s.nextSessionID)
		s.nextSessionID++
		session := s.ensureSessionLocked(spaceName, projectName, codeCatalystString(devEnv, "id", devEnvID), sessionID, now)
		session["status"] = "ACTIVE"
		session["startedTime"] = now.Format(time.RFC3339)
		session["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{"id": sessionID, "status": "ACTIVE", "sessionUrl": codeCatalystString(session, "sessionUrl", "")}

	case "StopDevEnvironmentSession":
		id := codeCatalystString(ctx, "id", devEnvID)
		session := s.ensureSessionLocked(spaceName, projectName, id, sessionID, now)
		session["status"] = "STOPPED"
		session["endedTime"] = now.Format(time.RFC3339)
		session["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{}

	case "ListDevEnvironmentSessions":
		id := codeCatalystString(ctx, "devEnvironmentId", devEnvID)
		s.ensureDevEnvironmentLocked(spaceName, projectName, id, now)
		items := []any{}
		prefix := codeCatalystSessionPrefix(spaceName, projectName, id)
		for _, key := range codeCatalystSortedKeys(s.sessions) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.sessions[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "ListWorkflows":
		items := []any{}
		prefix := codeCatalystWorkflowPrefix(spaceName, projectName)
		for _, key := range codeCatalystSortedKeys(s.workflows) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.workflows[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "GetWorkflow":
		workflow := s.ensureWorkflowLocked(spaceName, projectName, workflowID, now)
		return codeCatalystCloneMap(workflow)

	case "StartWorkflowRun":
		runID := fmt.Sprintf("workflow-run-%06d", s.nextWorkflowRunID)
		s.nextWorkflowRunID++
		workflowIDFromReq := codeCatalystString(ctx, "workflowId", "workflow-000001")
		s.ensureWorkflowLocked(spaceName, projectName, workflowIDFromReq, now)
		run := s.ensureWorkflowRunLocked(spaceName, projectName, runID, now)
		run["workflowId"] = workflowIDFromReq
		run["status"] = "SUCCEEDED"
		run["startedTime"] = now.Add(-5 * time.Second).Format(time.RFC3339)
		run["endedTime"] = now.Format(time.RFC3339)
		run["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{"id": runID, "status": "SUCCEEDED", "workflowId": workflowIDFromReq}

	case "ListWorkflowRuns":
		items := []any{}
		prefix := codeCatalystWorkflowRunPrefix(spaceName, projectName)
		for _, key := range codeCatalystSortedKeys(s.workflowRuns) {
			if !strings.HasPrefix(key, prefix) {
				continue
			}
			items = append(items, codeCatalystCloneMap(s.workflowRuns[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "GetWorkflowRun":
		run := s.ensureWorkflowRunLocked(spaceName, projectName, workflowRunID, now)
		return codeCatalystCloneMap(run)

	case "ListEventLogs":
		entries := s.eventLogs[spaceName]
		if len(entries) == 0 {
			s.appendEventLocked(spaceName, "ListEventLogs", now)
			entries = s.eventLogs[spaceName]
		}
		items := make([]any, 0, len(entries))
		for _, entry := range entries {
			items = append(items, codeCatalystCloneMap(entry))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "CreateAccessToken":
		id := fmt.Sprintf("at-%06d", s.nextAccessTokenID)
		s.nextAccessTokenID++
		token := s.ensureAccessTokenLocked(id, now)
		if v := codeCatalystString(ctx, "name", ""); v != "" {
			token["name"] = v
		}
		token["lastUpdatedTime"] = now.Format(time.RFC3339)
		return map[string]any{"id": id, "name": codeCatalystString(token, "name", id), "secret": "stk_" + strings.ReplaceAll(id, "-", "")}

	case "ListAccessTokens":
		items := []any{}
		for _, key := range codeCatalystSortedKeys(s.accessTokens) {
			items = append(items, codeCatalystCloneMap(s.accessTokens[key]))
		}
		return map[string]any{"items": items, "nextToken": ""}

	case "DeleteAccessToken":
		delete(s.accessTokens, accessTokenID)
		return map[string]any{}

	case "GetSubscription":
		return map[string]any{
			"spaceName":        spaceName,
			"subscriptionType": "FREE",
			"awsAccountId":     "123456789012",
			"isFreeTier":       true,
		}

	case "GetUserDetails":
		return map[string]any{
			"userId":       "user-000001",
			"userName":     "stackyard-user",
			"displayName":  "Stackyard User",
			"primaryEmail": "user@example.com",
		}

	case "VerifySession":
		return map[string]any{
			"identity": map[string]any{
				"userId":      "user-000001",
				"userName":    "stackyard-user",
				"displayName": "Stackyard User",
			},
			"valid": true,
		}
	}

	return map[string]any{}
}

func (s *codeCatalystStore) ensureSeedDataLocked() {
	now := time.Now().UTC()
	s.ensureSpaceLocked("stackyard-space", now)
	s.ensureProjectLocked("stackyard-space", "stackyard-project", now)
	s.ensureRepositoryLocked("stackyard-space", "stackyard-project", "stackyard-repository", now)
	s.ensureBranchLocked("stackyard-space", "stackyard-project", "stackyard-repository", "main", now)
	s.ensureDevEnvironmentLocked("stackyard-space", "stackyard-project", "dev-env-000001", now)
	s.ensureSessionLocked("stackyard-space", "stackyard-project", "dev-env-000001", "session-000001", now)
	s.ensureWorkflowLocked("stackyard-space", "stackyard-project", "workflow-000001", now)
	s.ensureWorkflowRunLocked("stackyard-space", "stackyard-project", "workflow-run-000001", now)
	s.ensureAccessTokenLocked("at-000001", now)
	if len(s.eventLogs["stackyard-space"]) == 0 {
		s.appendEventLocked("stackyard-space", "SeedData", now)
	}
}

func (s *codeCatalystStore) ensureSpaceLocked(spaceName string, now time.Time) map[string]any {
	name := strings.TrimSpace(spaceName)
	if name == "" {
		name = "stackyard-space"
	}
	if space := s.spaces[name]; space != nil {
		return space
	}
	space := map[string]any{
		"name":            name,
		"displayName":     strings.Title(strings.ReplaceAll(name, "-", " ")),
		"description":     "Stackyard CodeCatalyst space " + name,
		"regionName":      "us-east-1",
		"createdTime":     now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
	}
	s.spaces[name] = space
	return space
}

func (s *codeCatalystStore) ensureProjectLocked(spaceName, projectName string, now time.Time) map[string]any {
	s.ensureSpaceLocked(spaceName, now)
	name := strings.TrimSpace(projectName)
	if name == "" {
		name = "stackyard-project"
	}
	key := codeCatalystProjectKey(spaceName, name)
	if project := s.projects[key]; project != nil {
		return project
	}
	project := map[string]any{
		"spaceName":       spaceName,
		"name":            name,
		"displayName":     strings.Title(strings.ReplaceAll(name, "-", " ")),
		"description":     "Stackyard CodeCatalyst project " + name,
		"createdTime":     now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
	}
	s.projects[key] = project
	return project
}

func (s *codeCatalystStore) ensureRepositoryLocked(spaceName, projectName, repositoryName string, now time.Time) map[string]any {
	s.ensureProjectLocked(spaceName, projectName, now)
	name := strings.TrimSpace(repositoryName)
	if name == "" {
		name = "stackyard-repository"
	}
	key := codeCatalystRepositoryKey(spaceName, projectName, name)
	if repo := s.repositories[key]; repo != nil {
		return repo
	}
	repo := map[string]any{
		"spaceName":         spaceName,
		"projectName":       projectName,
		"name":              name,
		"description":       "Stackyard CodeCatalyst source repository " + name,
		"defaultBranchName": "main",
		"createdTime":       now.Format(time.RFC3339),
		"lastUpdatedTime":   now.Format(time.RFC3339),
	}
	s.repositories[key] = repo
	s.ensureBranchLocked(spaceName, projectName, name, "main", now)
	return repo
}

func (s *codeCatalystStore) ensureBranchLocked(spaceName, projectName, repositoryName, branchName string, now time.Time) map[string]any {
	s.ensureRepositoryLocked(spaceName, projectName, repositoryName, now)
	name := strings.TrimSpace(branchName)
	if name == "" {
		name = "main"
	}
	key := codeCatalystBranchKey(spaceName, projectName, repositoryName, name)
	if branch := s.branches[key]; branch != nil {
		return branch
	}
	branch := map[string]any{
		"spaceName":            spaceName,
		"projectName":          projectName,
		"sourceRepositoryName": repositoryName,
		"name":                 name,
		"headCommitId":         "commit-000001",
		"createdTime":          now.Format(time.RFC3339),
		"lastUpdatedTime":      now.Format(time.RFC3339),
	}
	s.branches[key] = branch
	return branch
}

func (s *codeCatalystStore) ensureDevEnvironmentLocked(spaceName, projectName, devEnvID string, now time.Time) map[string]any {
	s.ensureProjectLocked(spaceName, projectName, now)
	id := strings.TrimSpace(devEnvID)
	if id == "" {
		id = "dev-env-000001"
	}
	key := codeCatalystDevEnvironmentKey(spaceName, projectName, id)
	if env := s.devEnvs[key]; env != nil {
		return env
	}
	env := map[string]any{
		"spaceName":                spaceName,
		"projectName":              projectName,
		"id":                       id,
		"alias":                    "stackyard-dev-environment",
		"status":                   "STOPPED",
		"instanceType":             "dev.standard1.small",
		"inactivityTimeoutMinutes": 15,
		"repositories": []any{
			map[string]any{
				"name":       "stackyard-repository",
				"branchName": "main",
			},
		},
		"createdTime":     now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
	}
	s.devEnvs[key] = env
	return env
}

func (s *codeCatalystStore) ensureSessionLocked(spaceName, projectName, devEnvID, sessionID string, now time.Time) map[string]any {
	s.ensureDevEnvironmentLocked(spaceName, projectName, devEnvID, now)
	id := strings.TrimSpace(sessionID)
	if id == "" {
		id = "session-000001"
	}
	key := codeCatalystSessionKey(spaceName, projectName, devEnvID, id)
	if session := s.sessions[key]; session != nil {
		return session
	}
	session := map[string]any{
		"spaceName":        spaceName,
		"projectName":      projectName,
		"devEnvironmentId": devEnvID,
		"id":               id,
		"sessionType":      "SSH",
		"status":           "ACTIVE",
		"sessionUrl":       fmt.Sprintf("wss://codecatalyst.aws/session/%s", id),
		"startedTime":      now.Format(time.RFC3339),
		"lastUpdatedTime":  now.Format(time.RFC3339),
	}
	s.sessions[key] = session
	return session
}

func (s *codeCatalystStore) ensureWorkflowLocked(spaceName, projectName, workflowID string, now time.Time) map[string]any {
	s.ensureProjectLocked(spaceName, projectName, now)
	id := strings.TrimSpace(workflowID)
	if id == "" {
		id = "workflow-000001"
	}
	key := codeCatalystWorkflowKey(spaceName, projectName, id)
	if workflow := s.workflows[key]; workflow != nil {
		return workflow
	}
	workflow := map[string]any{
		"spaceName":            spaceName,
		"projectName":          projectName,
		"id":                   id,
		"name":                 "build-and-test",
		"sourceRepositoryName": "stackyard-repository",
		"sourceBranchName":     "main",
		"path":                 ".codecatalyst/workflows/build.yaml",
		"createdTime":          now.Format(time.RFC3339),
		"lastUpdatedTime":      now.Format(time.RFC3339),
	}
	s.workflows[key] = workflow
	return workflow
}

func (s *codeCatalystStore) ensureWorkflowRunLocked(spaceName, projectName, runID string, now time.Time) map[string]any {
	s.ensureWorkflowLocked(spaceName, projectName, "workflow-000001", now)
	id := strings.TrimSpace(runID)
	if id == "" {
		id = "workflow-run-000001"
	}
	key := codeCatalystWorkflowRunKey(spaceName, projectName, id)
	if run := s.workflowRuns[key]; run != nil {
		return run
	}
	run := map[string]any{
		"spaceName":       spaceName,
		"projectName":     projectName,
		"id":              id,
		"workflowId":      "workflow-000001",
		"status":          "SUCCEEDED",
		"triggeredBy":     "stackyard-user",
		"startedTime":     now.Add(-1 * time.Minute).Format(time.RFC3339),
		"endedTime":       now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
	}
	s.workflowRuns[key] = run
	return run
}

func (s *codeCatalystStore) ensureAccessTokenLocked(tokenID string, now time.Time) map[string]any {
	id := strings.TrimSpace(tokenID)
	if id == "" {
		id = "at-000001"
	}
	if token := s.accessTokens[id]; token != nil {
		return token
	}
	token := map[string]any{
		"id":              id,
		"name":            "stackyard-access-token",
		"status":          "ACTIVE",
		"createdTime":     now.Format(time.RFC3339),
		"lastUpdatedTime": now.Format(time.RFC3339),
		"expiresTime":     now.Add(24 * time.Hour).Format(time.RFC3339),
	}
	s.accessTokens[id] = token
	return token
}

func (s *codeCatalystStore) appendEventLocked(spaceName, operation string, now time.Time) {
	eventID := fmt.Sprintf("evt-%06d", len(s.eventLogs[spaceName])+1)
	entry := map[string]any{
		"id":            eventID,
		"eventType":     "API_CALL",
		"operationName": operation,
		"message":       "Processed " + operation,
		"eventTime":     now.Format(time.RFC3339),
	}
	s.eventLogs[spaceName] = append(s.eventLogs[spaceName], entry)
}

func codeCatalystMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := codeCatalystCloneMap(payload)
	for k, v := range pathParams {
		out[k] = strings.TrimSpace(v)
	}
	for k, values := range query {
		if len(values) == 0 {
			continue
		}
		out[k] = strings.TrimSpace(values[len(values)-1])
	}
	return out
}

func codeCatalystString(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	for k, v := range m {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		s := strings.TrimSpace(fmt.Sprint(v))
		if s != "" && s != "<nil>" {
			return s
		}
		break
	}
	return def
}

func codeCatalystCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = codeCatalystCloneAny(v)
	}
	return out
}

func codeCatalystCloneAny(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return codeCatalystCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = codeCatalystCloneAny(typed[i])
		}
		return out
	default:
		return typed
	}
}

func codeCatalystSortedKeys[V any](items map[string]V) []string {
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func codeCatalystSpacePrefix(spaceName string) string {
	return strings.TrimSpace(spaceName) + "/"
}

func codeCatalystProjectKey(spaceName, projectName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName)
}

func codeCatalystProjectPrefix(spaceName string) string {
	return strings.TrimSpace(spaceName) + "/"
}

func codeCatalystProjectResourcePrefix(spaceName, projectName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/"
}

func codeCatalystRepositoryKey(spaceName, projectName, repositoryName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(repositoryName)
}

func codeCatalystRepositoryPrefix(spaceName, projectName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/"
}

func codeCatalystBranchKey(spaceName, projectName, repositoryName, branchName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(repositoryName) + "/" + strings.TrimSpace(branchName)
}

func codeCatalystBranchPrefix(spaceName, projectName, repositoryName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(repositoryName) + "/"
}

func codeCatalystDevEnvironmentKey(spaceName, projectName, devEnvID string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(devEnvID)
}

func codeCatalystSessionKey(spaceName, projectName, devEnvID, sessionID string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(devEnvID) + "/" + strings.TrimSpace(sessionID)
}

func codeCatalystSessionPrefix(spaceName, projectName, devEnvID string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(devEnvID) + "/"
}

func codeCatalystWorkflowKey(spaceName, projectName, workflowID string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(workflowID)
}

func codeCatalystWorkflowPrefix(spaceName, projectName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/"
}

func codeCatalystWorkflowRunKey(spaceName, projectName, runID string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/" + strings.TrimSpace(runID)
}

func codeCatalystWorkflowRunPrefix(spaceName, projectName string) string {
	return strings.TrimSpace(spaceName) + "/" + strings.TrimSpace(projectName) + "/"
}
