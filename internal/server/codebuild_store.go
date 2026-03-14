package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type codeBuildStore struct {
	mu                 sync.Mutex
	nextID             int64
	projects           map[string]*codeBuildProject
	builds             map[string]*codeBuildBuild
	buildOrder         []string
	buildsByProject    map[string][]string
	buildBatches       map[string]*codeBuildBuildBatch
	buildBatchOrder    []string
	buildBatchesByProj map[string][]string
	reportGroups       map[string]*codeBuildReportGroup
	reports            map[string]*codeBuildReport
	reportOrder        []string
	reportsByGroup     map[string][]string
	fleets             map[string]*codeBuildFleet
	sandboxes          map[string]*codeBuildSandbox
	sandboxOrder       []string
	sandboxesByProject map[string][]string
	commandExecs       map[string]*codeBuildCommandExecution
	commandExecOrder   []string
	commandBySandbox   map[string][]string
	webhooks           map[string]*codeBuildWebhook
	sourceCreds        map[string]*codeBuildSourceCredential
	resourcePolicies   map[string]*codeBuildResourcePolicy
}

type codeBuildProject struct {
	Name         string
	Arn          string
	Description  string
	Visibility   string
	ServiceRole  string
	Source       map[string]any
	Environment  map[string]any
	Created      time.Time
	LastModified time.Time
}

type codeBuildBuild struct {
	ID            string
	Arn           string
	ProjectName   string
	Status        string
	Initiator     string
	SourceVersion string
	StartTime     time.Time
	EndTime       *time.Time
}

type codeBuildBuildBatch struct {
	ID            string
	Arn           string
	ProjectName   string
	Status        string
	Initiator     string
	SourceVersion string
	StartTime     time.Time
	EndTime       *time.Time
}

type codeBuildReportGroup struct {
	Arn          string
	Name         string
	Type         string
	ExportConfig map[string]any
	Created      time.Time
	LastModified time.Time
}

type codeBuildReport struct {
	Arn            string
	Name           string
	ReportGroupArn string
	Status         string
	Created        time.Time
	Expired        time.Time
}

type codeBuildFleet struct {
	Arn          string
	Name         string
	StatusCode   string
	StatusCtx    string
	StatusMsg    string
	BaseCapacity int
	Created      time.Time
	LastModified time.Time
}

type codeBuildSandbox struct {
	ID           string
	Arn          string
	ProjectName  string
	Status       string
	StartTime    time.Time
	LastModified time.Time
}

type codeBuildCommandExecution struct {
	ID           string
	SandboxID    string
	Command      string
	Status       string
	StartTime    time.Time
	LastModified time.Time
}

type codeBuildWebhook struct {
	ProjectName string
	URL         string
	PayloadURL  string
	Secret      string
	LastUpdated time.Time
}

type codeBuildSourceCredential struct {
	Arn         string
	ServerType  string
	AuthType    string
	Resource    string
	LastUpdated time.Time
}

type codeBuildResourcePolicy struct {
	ResourceArn string
	Policy      string
	LastUpdated time.Time
}

func newCodeBuildStore() *codeBuildStore {
	now := time.Now().UTC()
	store := &codeBuildStore{
		nextID:             1,
		projects:           map[string]*codeBuildProject{},
		builds:             map[string]*codeBuildBuild{},
		buildOrder:         []string{},
		buildsByProject:    map[string][]string{},
		buildBatches:       map[string]*codeBuildBuildBatch{},
		buildBatchOrder:    []string{},
		buildBatchesByProj: map[string][]string{},
		reportGroups:       map[string]*codeBuildReportGroup{},
		reports:            map[string]*codeBuildReport{},
		reportOrder:        []string{},
		reportsByGroup:     map[string][]string{},
		fleets:             map[string]*codeBuildFleet{},
		sandboxes:          map[string]*codeBuildSandbox{},
		sandboxOrder:       []string{},
		sandboxesByProject: map[string][]string{},
		commandExecs:       map[string]*codeBuildCommandExecution{},
		commandExecOrder:   []string{},
		commandBySandbox:   map[string][]string{},
		webhooks:           map[string]*codeBuildWebhook{},
		sourceCreds:        map[string]*codeBuildSourceCredential{},
		resourcePolicies:   map[string]*codeBuildResourcePolicy{},
	}

	project := &codeBuildProject{
		Name:        "stackyard-codebuild-project",
		Arn:         codeBuildProjectARN("stackyard-codebuild-project"),
		Description: "seed project",
		Visibility:  "PRIVATE",
		ServiceRole: "arn:aws:iam::123456789012:role/stackyard-codebuild",
		Source: map[string]any{
			"type":      "NO_SOURCE",
			"buildspec": "version: 0.2\nphases:\n  build:\n    commands:\n      - echo stackyard\n",
		},
		Environment: map[string]any{
			"type":                       "LINUX_CONTAINER",
			"image":                      "aws/codebuild/standard:7.0",
			"computeType":                "BUILD_GENERAL1_SMALL",
			"privilegedMode":             false,
			"environmentVariables":       []any{},
			"certificate":                "",
			"imagePullCredentialsType":   "CODEBUILD",
			"registryCredential":         nil,
			"dockerServer":               nil,
			"fleet":                      nil,
			"computeConfiguration":       nil,
			"typeDetails":                nil,
			"debugSessionEnabledDefault": false,
		},
		Created:      now,
		LastModified: now,
	}
	store.projects[project.Name] = project
	store.buildsByProject[project.Name] = []string{}
	store.buildBatchesByProj[project.Name] = []string{}
	store.sandboxesByProject[project.Name] = []string{}

	fleet := &codeBuildFleet{
		Arn:          codeBuildFleetARN("stackyard-codebuild-fleet"),
		Name:         "stackyard-codebuild-fleet",
		StatusCode:   "ACTIVE",
		StatusCtx:    "",
		StatusMsg:    "",
		BaseCapacity: 1,
		Created:      now,
		LastModified: now,
	}
	store.fleets[fleet.Name] = fleet

	group := &codeBuildReportGroup{
		Arn:  codeBuildReportGroupARN("stackyard-codebuild-report-group"),
		Name: "stackyard-codebuild-report-group",
		Type: "TEST",
		ExportConfig: map[string]any{
			"exportConfigType": "NO_EXPORT",
		},
		Created:      now,
		LastModified: now,
	}
	store.reportGroups[group.Arn] = group
	store.reportsByGroup[group.Arn] = []string{}

	report := &codeBuildReport{
		Arn:            codeBuildReportARN("stackyard-codebuild-report"),
		Name:           "stackyard-codebuild-report",
		ReportGroupArn: group.Arn,
		Status:         "SUCCEEDED",
		Created:        now,
		Expired:        now.Add(24 * time.Hour),
	}
	store.reports[report.Arn] = report
	store.reportOrder = append(store.reportOrder, report.Arn)
	store.reportsByGroup[group.Arn] = append(store.reportsByGroup[group.Arn], report.Arn)

	cred := &codeBuildSourceCredential{
		Arn:         "arn:aws:codebuild:us-east-1:123456789012:token/stackyard-source-credential",
		ServerType:  "GITHUB",
		AuthType:    "PERSONAL_ACCESS_TOKEN",
		Resource:    "stackyard",
		LastUpdated: now,
	}
	store.sourceCreds[cred.Arn] = cred

	store.resourcePolicies[project.Arn] = &codeBuildResourcePolicy{
		ResourceArn: project.Arn,
		Policy:      `{"Version":"2012-10-17","Statement":[]}`,
		LastUpdated: now,
	}
	return store
}

func (s *codeBuildStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "BatchDeleteBuilds":
		ids := codeBuildStringSlice(codeBuildPayloadValue(payload, "ids"))
		if len(ids) == 0 {
			ids = append([]string{}, s.buildOrder...)
		}
		deleted := make([]any, 0, len(ids))
		for _, id := range ids {
			if build := s.builds[id]; build != nil {
				delete(s.builds, id)
				s.buildOrder = codeBuildRemoveString(s.buildOrder, id)
				s.buildsByProject[build.ProjectName] = codeBuildRemoveString(s.buildsByProject[build.ProjectName], id)
				deleted = append(deleted, id)
			}
		}
		return map[string]any{"buildsDeleted": deleted, "buildsNotDeleted": []any{}}

	case "BatchGetBuildBatches":
		ids := codeBuildStringSlice(codeBuildPayloadValue(payload, "ids"))
		if len(ids) == 0 {
			ids = append([]string{}, s.buildBatchOrder...)
		}
		out := make([]any, 0, len(ids))
		notFound := make([]any, 0)
		for _, id := range ids {
			batch := s.buildBatches[id]
			if batch == nil {
				notFound = append(notFound, id)
				continue
			}
			out = append(out, s.buildBatchPayload(batch))
		}
		return map[string]any{"buildBatches": out, "buildBatchesNotFound": notFound}

	case "BatchGetBuilds":
		ids := codeBuildStringSlice(codeBuildPayloadValue(payload, "ids"))
		if len(ids) == 0 {
			ids = append([]string{}, s.buildOrder...)
		}
		out := make([]any, 0, len(ids))
		notFound := make([]any, 0)
		for _, id := range ids {
			build := s.builds[id]
			if build == nil {
				notFound = append(notFound, id)
				continue
			}
			out = append(out, s.buildPayload(build))
		}
		return map[string]any{"builds": out, "buildsNotFound": notFound}

	case "BatchGetCommandExecutions":
		ids := codeBuildStringSlice(codeBuildPayloadValue(payload, "commandExecutionIds"))
		if len(ids) == 0 {
			ids = codeBuildStringSlice(codeBuildPayloadValue(payload, "ids"))
		}
		if len(ids) == 0 {
			ids = append([]string{}, s.commandExecOrder...)
		}
		out := make([]any, 0, len(ids))
		notFound := make([]any, 0)
		for _, id := range ids {
			cmd := s.commandExecs[id]
			if cmd == nil {
				notFound = append(notFound, id)
				continue
			}
			out = append(out, s.commandExecutionPayload(cmd))
		}
		return map[string]any{"commandExecutions": out, "commandExecutionsNotFound": notFound}

	case "BatchGetFleets":
		names := codeBuildStringSlice(codeBuildPayloadValue(payload, "names"))
		if len(names) == 0 {
			names = codeBuildStringSlice(codeBuildPayloadValue(payload, "fleetNames"))
		}
		if len(names) == 0 {
			names = s.sortedFleetNames()
		}
		out := make([]any, 0, len(names))
		notFound := make([]any, 0)
		for _, name := range names {
			fleet := s.fleets[name]
			if fleet == nil {
				notFound = append(notFound, name)
				continue
			}
			out = append(out, s.fleetPayload(fleet))
		}
		return map[string]any{"fleets": out, "fleetsNotFound": notFound}

	case "BatchGetProjects":
		names := codeBuildStringSlice(codeBuildPayloadValue(payload, "names"))
		if len(names) == 0 {
			names = s.sortedProjectNames()
		}
		out := make([]any, 0, len(names))
		notFound := make([]any, 0)
		for _, name := range names {
			project := s.projects[name]
			if project == nil {
				notFound = append(notFound, name)
				continue
			}
			out = append(out, s.projectPayload(project))
		}
		return map[string]any{"projects": out, "projectsNotFound": notFound}

	case "BatchGetReportGroups":
		arns := codeBuildStringSlice(codeBuildPayloadValue(payload, "reportGroupArns"))
		if len(arns) == 0 {
			arns = s.sortedReportGroupARNs()
		}
		out := make([]any, 0, len(arns))
		notFound := make([]any, 0)
		for _, arn := range arns {
			group := s.reportGroups[arn]
			if group == nil {
				notFound = append(notFound, arn)
				continue
			}
			out = append(out, s.reportGroupPayload(group))
		}
		return map[string]any{"reportGroups": out, "reportGroupsNotFound": notFound}

	case "BatchGetReports":
		arns := codeBuildStringSlice(codeBuildPayloadValue(payload, "reportArns"))
		if len(arns) == 0 {
			arns = append([]string{}, s.reportOrder...)
		}
		out := make([]any, 0, len(arns))
		notFound := make([]any, 0)
		for _, arn := range arns {
			report := s.reports[arn]
			if report == nil {
				notFound = append(notFound, arn)
				continue
			}
			out = append(out, s.reportPayload(report))
		}
		return map[string]any{"reports": out, "reportsNotFound": notFound}

	case "BatchGetSandboxes":
		ids := codeBuildStringSlice(codeBuildPayloadValue(payload, "ids"))
		if len(ids) == 0 {
			ids = codeBuildStringSlice(codeBuildPayloadValue(payload, "sandboxIds"))
		}
		if len(ids) == 0 {
			ids = append([]string{}, s.sandboxOrder...)
		}
		out := make([]any, 0, len(ids))
		notFound := make([]any, 0)
		for _, id := range ids {
			sandbox := s.sandboxes[id]
			if sandbox == nil {
				notFound = append(notFound, id)
				continue
			}
			out = append(out, s.sandboxPayload(sandbox))
		}
		return map[string]any{"sandboxes": out, "sandboxesNotFound": notFound}

	case "CreateFleet":
		name := codeBuildDefaultString(payload, "name", fmt.Sprintf("stackyard-codebuild-fleet-%06d", s.nextLocked()))
		fleet := s.ensureFleetLocked(name)
		fleet.StatusCode = "ACTIVE"
		fleet.StatusCtx = ""
		fleet.StatusMsg = ""
		fleet.BaseCapacity = codeBuildDefaultInt(payload, "baseCapacity", fleet.BaseCapacity)
		fleet.LastModified = time.Now().UTC()
		return map[string]any{"fleet": s.fleetPayload(fleet)}

	case "CreateProject":
		name := codeBuildDefaultString(payload, "name", fmt.Sprintf("stackyard-codebuild-project-%06d", s.nextLocked()))
		project := s.ensureProjectLocked(name)
		project.Description = codeBuildDefaultString(payload, "description", project.Description)
		project.ServiceRole = codeBuildDefaultString(payload, "serviceRole", project.ServiceRole)
		if source, ok := codeBuildPayloadValue(payload, "source").(map[string]any); ok && source != nil {
			project.Source = codeBuildCloneMap(source)
		}
		if env, ok := codeBuildPayloadValue(payload, "environment").(map[string]any); ok && env != nil {
			project.Environment = codeBuildCloneMap(env)
		}
		project.LastModified = time.Now().UTC()
		return map[string]any{"project": s.projectPayload(project)}

	case "CreateReportGroup":
		name := codeBuildDefaultString(payload, "name", fmt.Sprintf("stackyard-codebuild-report-group-%06d", s.nextLocked()))
		arn := codeBuildReportGroupARN(name)
		group := s.reportGroups[arn]
		if group == nil {
			now := time.Now().UTC()
			group = &codeBuildReportGroup{
				Arn:          arn,
				Name:         name,
				Type:         codeBuildDefaultString(payload, "type", "TEST"),
				ExportConfig: map[string]any{"exportConfigType": "NO_EXPORT"},
				Created:      now,
				LastModified: now,
			}
			if exportCfg, ok := codeBuildPayloadValue(payload, "exportConfig").(map[string]any); ok && exportCfg != nil {
				group.ExportConfig = codeBuildCloneMap(exportCfg)
			}
			s.reportGroups[arn] = group
			s.reportsByGroup[arn] = []string{}
		}
		return map[string]any{"reportGroup": s.reportGroupPayload(group)}

	case "CreateWebhook":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		s.ensureProjectLocked(projectName)
		now := time.Now().UTC()
		webhook := &codeBuildWebhook{
			ProjectName: projectName,
			URL:         fmt.Sprintf("https://hooks.example.com/codebuild/%s", projectName),
			PayloadURL:  fmt.Sprintf("https://hooks.example.com/codebuild/%s/payload", projectName),
			Secret:      "stackyard-secret",
			LastUpdated: now,
		}
		s.webhooks[projectName] = webhook
		return map[string]any{"webhook": s.webhookPayload(webhook)}

	case "DeleteBuildBatch":
		id := codeBuildDefaultString(payload, "id", "")
		if id != "" {
			if batch := s.buildBatches[id]; batch != nil {
				delete(s.buildBatches, id)
				s.buildBatchOrder = codeBuildRemoveString(s.buildBatchOrder, id)
				s.buildBatchesByProj[batch.ProjectName] = codeBuildRemoveString(s.buildBatchesByProj[batch.ProjectName], id)
			}
		}
		return map[string]any{"statusCode": "OK"}

	case "DeleteFleet":
		name := codeBuildDefaultString(payload, "name", "")
		if name != "" {
			delete(s.fleets, name)
		}
		return map[string]any{"statusCode": "OK"}

	case "DeleteProject":
		name := codeBuildDefaultString(payload, "name", "")
		if name != "" {
			delete(s.projects, name)
			delete(s.buildsByProject, name)
			delete(s.buildBatchesByProj, name)
			delete(s.sandboxesByProject, name)
			delete(s.webhooks, name)
		}
		return map[string]any{}

	case "DeleteReport":
		arn := codeBuildDefaultString(payload, "arn", "")
		if arn == "" {
			arn = codeBuildDefaultString(payload, "reportArn", "")
		}
		if arn != "" {
			if report := s.reports[arn]; report != nil {
				delete(s.reports, arn)
				s.reportOrder = codeBuildRemoveString(s.reportOrder, arn)
				s.reportsByGroup[report.ReportGroupArn] = codeBuildRemoveString(s.reportsByGroup[report.ReportGroupArn], arn)
			}
		}
		return map[string]any{"statusCode": "OK"}

	case "DeleteReportGroup":
		arn := codeBuildDefaultString(payload, "arn", "")
		if arn == "" {
			arn = codeBuildDefaultString(payload, "reportGroupArn", "")
		}
		if arn != "" {
			delete(s.reportGroups, arn)
			for _, reportArn := range append([]string{}, s.reportsByGroup[arn]...) {
				delete(s.reports, reportArn)
				s.reportOrder = codeBuildRemoveString(s.reportOrder, reportArn)
			}
			delete(s.reportsByGroup, arn)
		}
		return map[string]any{"statusCode": "OK"}

	case "DeleteResourcePolicy":
		resourceArn := codeBuildDefaultString(payload, "resourceArn", "")
		if resourceArn != "" {
			delete(s.resourcePolicies, resourceArn)
		}
		return map[string]any{}

	case "DeleteSourceCredentials":
		arn := codeBuildDefaultString(payload, "arn", "")
		if arn != "" {
			delete(s.sourceCreds, arn)
		}
		return map[string]any{"arn": arn}

	case "DeleteWebhook":
		projectName := codeBuildDefaultString(payload, "projectName", "")
		if projectName != "" {
			delete(s.webhooks, projectName)
		}
		return map[string]any{}

	case "DescribeCodeCoverages":
		return map[string]any{
			"codeCoverages": []any{
				map[string]any{
					"id":                         "coverage-1",
					"reportARN":                  codeBuildReportARN("stackyard-codebuild-report"),
					"filePath":                   "main.go",
					"lineCoveragePercentage":     100.0,
					"linesCovered":               10.0,
					"linesMissed":                0.0,
					"expired":                    false,
					"branchesCovered":            5.0,
					"branchesMissed":             0.0,
					"branchCoveragePercentage":   100.0,
					"lineCoverageStatus":         "PASSED",
					"branchCoverageStatus":       "PASSED",
					"lineCoverageThreshold":      80.0,
					"branchCoverageThreshold":    80.0,
					"lineCoverageRawValue":       100.0,
					"branchCoverageRawValue":     100.0,
					"lineCoverageStatusReason":   "",
					"branchCoverageStatusReason": "",
				},
			},
			"nextToken": "",
		}

	case "DescribeTestCases":
		return map[string]any{
			"testCases": []any{
				map[string]any{
					"reportArn":             codeBuildReportARN("stackyard-codebuild-report"),
					"testRawDataPath":       "",
					"prefix":                "unit",
					"name":                  "TestStackyard",
					"status":                "SUCCEEDED",
					"durationInNanoSeconds": int64(1000000),
					"message":               "",
					"expired":               false,
				},
			},
			"nextToken": "",
		}

	case "GetReportGroupTrend":
		return map[string]any{
			"stats": map[string]any{
				"average": "1.0",
				"max":     "1.0",
				"min":     "1.0",
			},
			"rawData": []any{},
		}

	case "GetResourcePolicy":
		resourceArn := codeBuildDefaultString(payload, "resourceArn", "")
		if resourceArn == "" {
			resourceArn = codeBuildProjectARN("stackyard-codebuild-project")
		}
		policy := s.resourcePolicies[resourceArn]
		if policy == nil {
			policy = &codeBuildResourcePolicy{
				ResourceArn: resourceArn,
				Policy:      `{"Version":"2012-10-17","Statement":[]}`,
				LastUpdated: time.Now().UTC(),
			}
			s.resourcePolicies[resourceArn] = policy
		}
		return map[string]any{
			"resourceArn": policy.ResourceArn,
			"policy":      policy.Policy,
		}

	case "ImportSourceCredentials":
		serverType := codeBuildDefaultString(payload, "serverType", "GITHUB")
		authType := codeBuildDefaultString(payload, "authType", "PERSONAL_ACCESS_TOKEN")
		token := strings.TrimSpace(codeBuildDefaultString(payload, "token", "stackyard"))
		arn := fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:token/%s-%06d", strings.ToLower(serverType), s.nextLocked())
		cred := &codeBuildSourceCredential{
			Arn:         arn,
			ServerType:  serverType,
			AuthType:    authType,
			Resource:    token,
			LastUpdated: time.Now().UTC(),
		}
		s.sourceCreds[arn] = cred
		return map[string]any{
			"arn": cred.Arn,
		}

	case "InvalidateProjectCache":
		return map[string]any{"status": "OK"}

	case "ListBuildBatches":
		ids := append([]string{}, s.buildBatchOrder...)
		sort.Strings(ids)
		return map[string]any{"ids": codeBuildToAnySlice(ids), "nextToken": ""}

	case "ListBuildBatchesForProject":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		ids := append([]string{}, s.buildBatchesByProj[projectName]...)
		sort.Strings(ids)
		return map[string]any{"ids": codeBuildToAnySlice(ids), "nextToken": ""}

	case "ListBuilds":
		ids := append([]string{}, s.buildOrder...)
		sort.Strings(ids)
		return map[string]any{"ids": codeBuildToAnySlice(ids), "nextToken": ""}

	case "ListBuildsForProject":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		ids := append([]string{}, s.buildsByProject[projectName]...)
		sort.Strings(ids)
		return map[string]any{"ids": codeBuildToAnySlice(ids), "nextToken": ""}

	case "ListCommandExecutionsForSandbox":
		sandboxID := codeBuildDefaultString(payload, "sandboxId", "")
		if sandboxID == "" {
			sandboxID = codeBuildDefaultString(payload, "id", "")
		}
		ids := append([]string{}, s.commandBySandbox[sandboxID]...)
		sort.Strings(ids)
		out := make([]any, 0, len(ids))
		for _, id := range ids {
			if cmd := s.commandExecs[id]; cmd != nil {
				out = append(out, s.commandExecutionPayload(cmd))
			}
		}
		return map[string]any{"commandExecutions": out, "nextToken": ""}

	case "ListCuratedEnvironmentImages":
		return map[string]any{
			"platforms": []any{
				map[string]any{
					"platform": "LINUX_CONTAINER",
					"languages": []any{
						map[string]any{
							"language": "Golang",
							"images": []any{
								map[string]any{
									"name":               "aws/codebuild/standard:7.0",
									"description":        "Stackyard curated image",
									"versions":           []any{"1"},
									"supportedPlatforms": []any{"LINUX_CONTAINER"},
								},
							},
						},
					},
				},
			},
		}

	case "ListFleets":
		names := s.sortedFleetNames()
		arns := make([]any, 0, len(names))
		for _, name := range names {
			arns = append(arns, s.fleets[name].Arn)
		}
		return map[string]any{"fleets": arns, "nextToken": ""}

	case "ListProjects":
		return map[string]any{"projects": codeBuildToAnySlice(s.sortedProjectNames()), "nextToken": ""}

	case "ListReportGroups":
		return map[string]any{"reportGroups": codeBuildToAnySlice(s.sortedReportGroupARNs()), "nextToken": ""}

	case "ListReports":
		arns := append([]string{}, s.reportOrder...)
		sort.Strings(arns)
		return map[string]any{"reports": codeBuildToAnySlice(arns), "nextToken": ""}

	case "ListReportsForReportGroup":
		groupArn := codeBuildDefaultString(payload, "reportGroupArn", "")
		if groupArn == "" {
			groupArn = codeBuildReportGroupARN("stackyard-codebuild-report-group")
		}
		arns := append([]string{}, s.reportsByGroup[groupArn]...)
		sort.Strings(arns)
		return map[string]any{"reports": codeBuildToAnySlice(arns), "nextToken": ""}

	case "ListSandboxes":
		arns := make([]string, 0, len(s.sandboxOrder))
		for _, id := range s.sandboxOrder {
			if sandbox := s.sandboxes[id]; sandbox != nil {
				arns = append(arns, sandbox.Arn)
			}
		}
		sort.Strings(arns)
		return map[string]any{"sandboxes": codeBuildToAnySlice(arns), "nextToken": ""}

	case "ListSandboxesForProject":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		ids := append([]string{}, s.sandboxesByProject[projectName]...)
		sort.Strings(ids)
		arns := make([]any, 0, len(ids))
		for _, id := range ids {
			if sandbox := s.sandboxes[id]; sandbox != nil {
				arns = append(arns, sandbox.Arn)
			}
		}
		return map[string]any{"sandboxes": arns, "nextToken": ""}

	case "ListSharedProjects":
		return map[string]any{"projects": codeBuildToAnySlice(s.sortedProjectNames()), "nextToken": ""}

	case "ListSharedReportGroups":
		return map[string]any{"reportGroups": codeBuildToAnySlice(s.sortedReportGroupARNs()), "nextToken": ""}

	case "ListSourceCredentials":
		arns := make([]string, 0, len(s.sourceCreds))
		for arn := range s.sourceCreds {
			arns = append(arns, arn)
		}
		sort.Strings(arns)
		out := make([]any, 0, len(arns))
		for _, arn := range arns {
			cred := s.sourceCreds[arn]
			out = append(out, map[string]any{
				"arn":                cred.Arn,
				"serverType":         cred.ServerType,
				"authType":           cred.AuthType,
				"shouldOverwrite":    true,
				"lastModifiedSecret": cred.LastUpdated,
			})
		}
		return map[string]any{"sourceCredentialsInfos": out}

	case "PutResourcePolicy":
		resourceArn := codeBuildDefaultString(payload, "resourceArn", "")
		if resourceArn == "" {
			resourceArn = codeBuildProjectARN("stackyard-codebuild-project")
		}
		policyDoc := codeBuildDefaultString(payload, "policy", `{"Version":"2012-10-17","Statement":[]}`)
		policy := &codeBuildResourcePolicy{
			ResourceArn: resourceArn,
			Policy:      policyDoc,
			LastUpdated: time.Now().UTC(),
		}
		s.resourcePolicies[resourceArn] = policy
		return map[string]any{"resourceArn": policy.ResourceArn, "policy": policy.Policy}

	case "RetryBuild":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		build := s.createBuildLocked(projectName, "IN_PROGRESS")
		return map[string]any{"build": s.buildPayload(build)}

	case "RetryBuildBatch":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		batch := s.createBuildBatchLocked(projectName, "IN_PROGRESS")
		return map[string]any{"buildBatch": s.buildBatchPayload(batch)}

	case "StartBuild":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		build := s.createBuildLocked(projectName, "IN_PROGRESS")
		return map[string]any{"build": s.buildPayload(build)}

	case "StartBuildBatch":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		batch := s.createBuildBatchLocked(projectName, "IN_PROGRESS")
		return map[string]any{"buildBatch": s.buildBatchPayload(batch)}

	case "StartCommandExecution":
		sandboxID := codeBuildDefaultString(payload, "sandboxId", "")
		if sandboxID == "" {
			sandboxID = codeBuildDefaultString(payload, "id", "")
		}
		if sandboxID == "" {
			projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
			sandbox := s.createSandboxLocked(projectName, "RUNNING")
			sandboxID = sandbox.ID
		}
		cmd := s.createCommandExecutionLocked(sandboxID, codeBuildDefaultString(payload, "command", "echo stackyard"), "IN_PROGRESS")
		return map[string]any{"commandExecution": s.commandExecutionPayload(cmd)}

	case "StartSandbox":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		sandbox := s.createSandboxLocked(projectName, "RUNNING")
		return map[string]any{"sandbox": s.sandboxPayload(sandbox)}

	case "StartSandboxConnection":
		id := codeBuildDefaultString(payload, "id", "")
		if id == "" {
			id = codeBuildDefaultString(payload, "sandboxId", "")
		}
		if id == "" {
			projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
			id = s.createSandboxLocked(projectName, "RUNNING").ID
		}
		sandbox := s.sandboxes[id]
		if sandbox != nil {
			sandbox.Status = "RUNNING"
			sandbox.LastModified = time.Now().UTC()
		}
		return map[string]any{
			"sandboxArn": codeBuildSandboxARN(id),
			"token":      fmt.Sprintf("sandbox-token-%06d", s.nextLocked()),
		}

	case "StopBuild":
		id := codeBuildDefaultString(payload, "id", "")
		if id == "" {
			id = codeBuildDefaultString(payload, "buildId", "")
		}
		build := s.ensureBuildLocked(id)
		now := time.Now().UTC()
		build.Status = "STOPPED"
		build.EndTime = &now
		return map[string]any{"build": s.buildPayload(build)}

	case "StopBuildBatch":
		id := codeBuildDefaultString(payload, "id", "")
		batch := s.ensureBuildBatchLocked(id)
		now := time.Now().UTC()
		batch.Status = "STOPPED"
		batch.EndTime = &now
		return map[string]any{"buildBatch": s.buildBatchPayload(batch)}

	case "StopSandbox":
		id := codeBuildDefaultString(payload, "id", "")
		if id == "" {
			id = codeBuildDefaultString(payload, "sandboxId", "")
		}
		sandbox := s.ensureSandboxLocked(id)
		sandbox.Status = "STOPPED"
		sandbox.LastModified = time.Now().UTC()
		return map[string]any{"sandbox": s.sandboxPayload(sandbox)}

	case "UpdateFleet":
		name := codeBuildDefaultString(payload, "name", "stackyard-codebuild-fleet")
		fleet := s.ensureFleetLocked(name)
		fleet.BaseCapacity = codeBuildDefaultInt(payload, "baseCapacity", fleet.BaseCapacity)
		fleet.LastModified = time.Now().UTC()
		return map[string]any{"fleet": s.fleetPayload(fleet)}

	case "UpdateProject":
		name := codeBuildDefaultString(payload, "name", "stackyard-codebuild-project")
		project := s.ensureProjectLocked(name)
		if desc := strings.TrimSpace(codeBuildDefaultString(payload, "description", "")); desc != "" {
			project.Description = desc
		}
		if source, ok := codeBuildPayloadValue(payload, "source").(map[string]any); ok && source != nil {
			project.Source = codeBuildCloneMap(source)
		}
		if env, ok := codeBuildPayloadValue(payload, "environment").(map[string]any); ok && env != nil {
			project.Environment = codeBuildCloneMap(env)
		}
		project.LastModified = time.Now().UTC()
		return map[string]any{"project": s.projectPayload(project)}

	case "UpdateProjectVisibility":
		name := codeBuildDefaultString(payload, "projectArn", "")
		if name == "" {
			name = codeBuildDefaultString(payload, "projectName", "")
		}
		if strings.HasPrefix(name, "arn:") {
			name = codeBuildNameFromArn(name)
		}
		if name == "" {
			name = "stackyard-codebuild-project"
		}
		project := s.ensureProjectLocked(name)
		project.Visibility = codeBuildDefaultString(payload, "projectVisibility", "PRIVATE")
		project.LastModified = time.Now().UTC()
		return map[string]any{
			"projectArn":        project.Arn,
			"projectVisibility": project.Visibility,
		}

	case "UpdateReportGroup":
		arn := codeBuildDefaultString(payload, "arn", "")
		if arn == "" {
			arn = codeBuildDefaultString(payload, "reportGroupArn", codeBuildReportGroupARN("stackyard-codebuild-report-group"))
		}
		group := s.ensureReportGroupLocked(arn)
		if exportCfg, ok := codeBuildPayloadValue(payload, "exportConfig").(map[string]any); ok && exportCfg != nil {
			group.ExportConfig = codeBuildCloneMap(exportCfg)
		}
		group.LastModified = time.Now().UTC()
		return map[string]any{"reportGroup": s.reportGroupPayload(group)}

	case "UpdateWebhook":
		projectName := codeBuildDefaultString(payload, "projectName", "stackyard-codebuild-project")
		s.ensureProjectLocked(projectName)
		webhook := s.webhooks[projectName]
		if webhook == nil {
			webhook = &codeBuildWebhook{
				ProjectName: projectName,
				URL:         fmt.Sprintf("https://hooks.example.com/codebuild/%s", projectName),
				PayloadURL:  fmt.Sprintf("https://hooks.example.com/codebuild/%s/payload", projectName),
				Secret:      "stackyard-secret",
			}
			s.webhooks[projectName] = webhook
		}
		webhook.LastUpdated = time.Now().UTC()
		return map[string]any{"webhook": s.webhookPayload(webhook)}
	}

	return map[string]any{}
}

func (s *codeBuildStore) sortedProjectNames() []string {
	names := make([]string, 0, len(s.projects))
	for name := range s.projects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *codeBuildStore) sortedFleetNames() []string {
	names := make([]string, 0, len(s.fleets))
	for name := range s.fleets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *codeBuildStore) sortedReportGroupARNs() []string {
	arns := make([]string, 0, len(s.reportGroups))
	for arn := range s.reportGroups {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	return arns
}

func (s *codeBuildStore) ensureProjectLocked(name string) *codeBuildProject {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-codebuild-project"
	}
	project := s.projects[name]
	if project != nil {
		return project
	}
	now := time.Now().UTC()
	project = &codeBuildProject{
		Name:        name,
		Arn:         codeBuildProjectARN(name),
		Description: "stackyard project",
		Visibility:  "PRIVATE",
		ServiceRole: "arn:aws:iam::123456789012:role/stackyard-codebuild",
		Source: map[string]any{
			"type": "NO_SOURCE",
		},
		Environment: map[string]any{
			"type":        "LINUX_CONTAINER",
			"image":       "aws/codebuild/standard:7.0",
			"computeType": "BUILD_GENERAL1_SMALL",
		},
		Created:      now,
		LastModified: now,
	}
	s.projects[name] = project
	s.buildsByProject[name] = []string{}
	s.buildBatchesByProj[name] = []string{}
	s.sandboxesByProject[name] = []string{}
	return project
}

func (s *codeBuildStore) ensureFleetLocked(name string) *codeBuildFleet {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-codebuild-fleet"
	}
	fleet := s.fleets[name]
	if fleet != nil {
		return fleet
	}
	now := time.Now().UTC()
	fleet = &codeBuildFleet{
		Arn:          codeBuildFleetARN(name),
		Name:         name,
		StatusCode:   "ACTIVE",
		StatusCtx:    "",
		StatusMsg:    "",
		BaseCapacity: 1,
		Created:      now,
		LastModified: now,
	}
	s.fleets[name] = fleet
	return fleet
}

func (s *codeBuildStore) ensureReportGroupLocked(arn string) *codeBuildReportGroup {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = codeBuildReportGroupARN("stackyard-codebuild-report-group")
	}
	group := s.reportGroups[arn]
	if group != nil {
		return group
	}
	now := time.Now().UTC()
	group = &codeBuildReportGroup{
		Arn:  arn,
		Name: codeBuildNameFromArn(arn),
		Type: "TEST",
		ExportConfig: map[string]any{
			"exportConfigType": "NO_EXPORT",
		},
		Created:      now,
		LastModified: now,
	}
	s.reportGroups[arn] = group
	s.reportsByGroup[arn] = []string{}
	return group
}

func (s *codeBuildStore) createBuildLocked(projectName, status string) *codeBuildBuild {
	project := s.ensureProjectLocked(projectName)
	now := time.Now().UTC()
	id := fmt.Sprintf("stackyard-build-%06d", s.nextLocked())
	build := &codeBuildBuild{
		ID:            id,
		Arn:           codeBuildBuildARN(id),
		ProjectName:   project.Name,
		Status:        status,
		Initiator:     "stackyard",
		SourceVersion: "main",
		StartTime:     now,
		EndTime:       nil,
	}
	s.builds[id] = build
	s.buildOrder = append(s.buildOrder, id)
	s.buildsByProject[project.Name] = append(s.buildsByProject[project.Name], id)
	return build
}

func (s *codeBuildStore) ensureBuildLocked(id string) *codeBuildBuild {
	id = strings.TrimSpace(id)
	if id == "" {
		if len(s.buildOrder) > 0 {
			id = s.buildOrder[len(s.buildOrder)-1]
		} else {
			return s.createBuildLocked("stackyard-codebuild-project", "IN_PROGRESS")
		}
	}
	if build := s.builds[id]; build != nil {
		return build
	}
	project := s.ensureProjectLocked("stackyard-codebuild-project")
	now := time.Now().UTC()
	build := &codeBuildBuild{
		ID:            id,
		Arn:           codeBuildBuildARN(id),
		ProjectName:   project.Name,
		Status:        "IN_PROGRESS",
		Initiator:     "stackyard",
		SourceVersion: "main",
		StartTime:     now,
	}
	s.builds[id] = build
	s.buildOrder = append(s.buildOrder, id)
	s.buildsByProject[project.Name] = append(s.buildsByProject[project.Name], id)
	return build
}

func (s *codeBuildStore) createBuildBatchLocked(projectName, status string) *codeBuildBuildBatch {
	project := s.ensureProjectLocked(projectName)
	now := time.Now().UTC()
	id := fmt.Sprintf("stackyard-buildbatch-%06d", s.nextLocked())
	batch := &codeBuildBuildBatch{
		ID:            id,
		Arn:           codeBuildBuildBatchARN(id),
		ProjectName:   project.Name,
		Status:        status,
		Initiator:     "stackyard",
		SourceVersion: "main",
		StartTime:     now,
		EndTime:       nil,
	}
	s.buildBatches[id] = batch
	s.buildBatchOrder = append(s.buildBatchOrder, id)
	s.buildBatchesByProj[project.Name] = append(s.buildBatchesByProj[project.Name], id)
	return batch
}

func (s *codeBuildStore) ensureBuildBatchLocked(id string) *codeBuildBuildBatch {
	id = strings.TrimSpace(id)
	if id == "" {
		if len(s.buildBatchOrder) > 0 {
			id = s.buildBatchOrder[len(s.buildBatchOrder)-1]
		} else {
			return s.createBuildBatchLocked("stackyard-codebuild-project", "IN_PROGRESS")
		}
	}
	if batch := s.buildBatches[id]; batch != nil {
		return batch
	}
	project := s.ensureProjectLocked("stackyard-codebuild-project")
	now := time.Now().UTC()
	batch := &codeBuildBuildBatch{
		ID:            id,
		Arn:           codeBuildBuildBatchARN(id),
		ProjectName:   project.Name,
		Status:        "IN_PROGRESS",
		Initiator:     "stackyard",
		SourceVersion: "main",
		StartTime:     now,
	}
	s.buildBatches[id] = batch
	s.buildBatchOrder = append(s.buildBatchOrder, id)
	s.buildBatchesByProj[project.Name] = append(s.buildBatchesByProj[project.Name], id)
	return batch
}

func (s *codeBuildStore) createSandboxLocked(projectName, status string) *codeBuildSandbox {
	project := s.ensureProjectLocked(projectName)
	now := time.Now().UTC()
	id := fmt.Sprintf("sandbox-%06d", s.nextLocked())
	sandbox := &codeBuildSandbox{
		ID:           id,
		Arn:          codeBuildSandboxARN(id),
		ProjectName:  project.Name,
		Status:       status,
		StartTime:    now,
		LastModified: now,
	}
	s.sandboxes[id] = sandbox
	s.sandboxOrder = append(s.sandboxOrder, id)
	s.sandboxesByProject[project.Name] = append(s.sandboxesByProject[project.Name], id)
	return sandbox
}

func (s *codeBuildStore) ensureSandboxLocked(id string) *codeBuildSandbox {
	id = strings.TrimSpace(id)
	if id == "" {
		if len(s.sandboxOrder) > 0 {
			id = s.sandboxOrder[len(s.sandboxOrder)-1]
		} else {
			return s.createSandboxLocked("stackyard-codebuild-project", "RUNNING")
		}
	}
	if sandbox := s.sandboxes[id]; sandbox != nil {
		return sandbox
	}
	project := s.ensureProjectLocked("stackyard-codebuild-project")
	now := time.Now().UTC()
	sandbox := &codeBuildSandbox{
		ID:           id,
		Arn:          codeBuildSandboxARN(id),
		ProjectName:  project.Name,
		Status:       "RUNNING",
		StartTime:    now,
		LastModified: now,
	}
	s.sandboxes[id] = sandbox
	s.sandboxOrder = append(s.sandboxOrder, id)
	s.sandboxesByProject[project.Name] = append(s.sandboxesByProject[project.Name], id)
	return sandbox
}

func (s *codeBuildStore) createCommandExecutionLocked(sandboxID, command, status string) *codeBuildCommandExecution {
	sandbox := s.ensureSandboxLocked(sandboxID)
	now := time.Now().UTC()
	id := fmt.Sprintf("cmd-%06d", s.nextLocked())
	cmd := &codeBuildCommandExecution{
		ID:           id,
		SandboxID:    sandbox.ID,
		Command:      command,
		Status:       status,
		StartTime:    now,
		LastModified: now,
	}
	s.commandExecs[id] = cmd
	s.commandExecOrder = append(s.commandExecOrder, id)
	s.commandBySandbox[sandbox.ID] = append(s.commandBySandbox[sandbox.ID], id)
	return cmd
}

func (s *codeBuildStore) projectPayload(project *codeBuildProject) map[string]any {
	return map[string]any{
		"name":               project.Name,
		"arn":                project.Arn,
		"description":        project.Description,
		"source":             codeBuildCloneMap(project.Source),
		"environment":        codeBuildCloneMap(project.Environment),
		"serviceRole":        project.ServiceRole,
		"created":            project.Created,
		"lastModified":       project.LastModified,
		"badge":              map[string]any{"badgeEnabled": false},
		"visibility":         project.Visibility,
		"projectVisibility":  project.Visibility,
		"secondarySources":   []any{},
		"secondaryArtifacts": []any{},
		"logsConfig": map[string]any{
			"cloudWatchLogs": map[string]any{"status": "ENABLED"},
			"s3Logs":         map[string]any{"status": "DISABLED"},
		},
		"queuedTimeoutInMinutes":   480,
		"timeoutInMinutes":         60,
		"encryptionKey":            "",
		"fileSystemLocations":      []any{},
		"concurrentBuildLimit":     1,
		"vpcConfig":                nil,
		"cache":                    nil,
		"buildBatchConfig":         nil,
		"projectFleet":             nil,
		"resourceAccessRole":       "",
		"publicProjectAlias":       "",
		"webhook":                  nil,
		"sourceVersion":            "",
		"queuedTimeoutInMinutesV2": 480,
	}
}

func (s *codeBuildStore) buildPayload(build *codeBuildBuild) map[string]any {
	resp := map[string]any{
		"id":                      build.ID,
		"arn":                     build.Arn,
		"buildNumber":             1,
		"startTime":               build.StartTime,
		"currentPhase":            "BUILD",
		"buildStatus":             build.Status,
		"projectName":             build.ProjectName,
		"phases":                  []any{},
		"source":                  map[string]any{"type": "NO_SOURCE"},
		"secondarySources":        []any{},
		"secondarySourceVersions": []any{},
		"artifacts":               map[string]any{"location": ""},
		"secondaryArtifacts":      []any{},
		"cache":                   map[string]any{"type": "NO_CACHE"},
		"environment":             map[string]any{"type": "LINUX_CONTAINER", "image": "aws/codebuild/standard:7.0", "computeType": "BUILD_GENERAL1_SMALL"},
		"serviceRole":             "arn:aws:iam::123456789012:role/stackyard-codebuild",
		"logs":                    map[string]any{"deepLink": "", "cloudWatchLogsArn": "", "s3DeepLink": ""},
		"timeoutInMinutes":        60,
		"queuedTimeoutInMinutes":  480,
		"buildComplete":           build.EndTime != nil,
		"initiator":               build.Initiator,
		"sourceVersion":           build.SourceVersion,
	}
	if build.EndTime != nil {
		resp["endTime"] = *build.EndTime
	}
	return resp
}

func (s *codeBuildStore) buildBatchPayload(batch *codeBuildBuildBatch) map[string]any {
	resp := map[string]any{
		"id":                    batch.ID,
		"arn":                   batch.Arn,
		"startTime":             batch.StartTime,
		"currentPhase":          "COMBINE_ARTIFACTS",
		"buildBatchStatus":      batch.Status,
		"sourceVersion":         batch.SourceVersion,
		"initiator":             batch.Initiator,
		"projectName":           batch.ProjectName,
		"buildGroups":           []any{},
		"complete":              batch.EndTime != nil,
		"serviceRole":           "arn:aws:iam::123456789012:role/stackyard-codebuild",
		"resolvedSourceVersion": batch.SourceVersion,
	}
	if batch.EndTime != nil {
		resp["endTime"] = *batch.EndTime
	}
	return resp
}

func (s *codeBuildStore) reportGroupPayload(group *codeBuildReportGroup) map[string]any {
	return map[string]any{
		"arn":          group.Arn,
		"name":         group.Name,
		"type":         group.Type,
		"created":      group.Created,
		"lastModified": group.LastModified,
		"status":       "ACTIVE",
		"exportConfig": codeBuildCloneMap(group.ExportConfig),
		"tags":         []any{},
	}
}

func (s *codeBuildStore) reportPayload(report *codeBuildReport) map[string]any {
	return map[string]any{
		"arn":                 report.Arn,
		"type":                "TEST",
		"name":                report.Name,
		"reportGroupArn":      report.ReportGroupArn,
		"executionId":         "stackyard-execution",
		"status":              report.Status,
		"created":             report.Created,
		"expired":             report.Expired,
		"exportConfig":        map[string]any{"exportConfigType": "NO_EXPORT"},
		"truncated":           false,
		"testSummary":         map[string]any{"total": 1, "statusCounts": map[string]any{"SUCCEEDED": 1}, "durationInNanoSeconds": int64(1000000)},
		"codeCoverageSummary": map[string]any{"lineCoveragePercentage": 100.0, "linesCovered": 1, "linesMissed": 0, "branchCoveragePercentage": 100.0, "branchesCovered": 1, "branchesMissed": 0},
	}
}

func (s *codeBuildStore) fleetPayload(fleet *codeBuildFleet) map[string]any {
	return map[string]any{
		"arn":                          fleet.Arn,
		"name":                         fleet.Name,
		"created":                      fleet.Created,
		"lastModified":                 fleet.LastModified,
		"status":                       map[string]any{"statusCode": fleet.StatusCode, "context": fleet.StatusCtx, "message": fleet.StatusMsg},
		"baseCapacity":                 fleet.BaseCapacity,
		"environmentType":              "LINUX_CONTAINER",
		"computeType":                  "BUILD_GENERAL1_SMALL",
		"vpcConfig":                    nil,
		"proxyConfiguration":           nil,
		"scalingConfiguration":         map[string]any{"maxCapacity": fleet.BaseCapacity},
		"targetTrackingScalingConfigs": []any{},
	}
}

func (s *codeBuildStore) sandboxPayload(sandbox *codeBuildSandbox) map[string]any {
	return map[string]any{
		"id":            sandbox.ID,
		"arn":           sandbox.Arn,
		"projectName":   sandbox.ProjectName,
		"status":        sandbox.Status,
		"created":       sandbox.StartTime,
		"lastModified":  sandbox.LastModified,
		"resolvedImage": "aws/codebuild/standard:7.0",
	}
}

func (s *codeBuildStore) commandExecutionPayload(cmd *codeBuildCommandExecution) map[string]any {
	return map[string]any{
		"id":           cmd.ID,
		"sandboxId":    cmd.SandboxID,
		"command":      cmd.Command,
		"status":       cmd.Status,
		"startTime":    cmd.StartTime,
		"lastModified": cmd.LastModified,
	}
}

func (s *codeBuildStore) webhookPayload(webhook *codeBuildWebhook) map[string]any {
	return map[string]any{
		"url":                webhook.URL,
		"payloadUrl":         webhook.PayloadURL,
		"secret":             webhook.Secret,
		"branchFilter":       ".*",
		"buildType":          "BUILD",
		"lastModifiedSecret": webhook.LastUpdated,
		"scopeConfiguration": nil,
	}
}

func (s *codeBuildStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func codeBuildProjectARN(name string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:project/%s", name)
}

func codeBuildBuildARN(id string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:build/%s", id)
}

func codeBuildBuildBatchARN(id string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:build-batch/%s", id)
}

func codeBuildReportGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:report-group/%s", name)
}

func codeBuildReportARN(name string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:report/%s", name)
}

func codeBuildFleetARN(name string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:fleet/%s", name)
}

func codeBuildSandboxARN(id string) string {
	return fmt.Sprintf("arn:aws:codebuild:us-east-1:123456789012:sandbox/%s", id)
}

func codeBuildNameFromArn(arn string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return ""
	}
	if idx := strings.LastIndex(arn, "/"); idx >= 0 && idx+1 < len(arn) {
		return arn[idx+1:]
	}
	return arn
}

func codeBuildPayloadValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for k, value := range payload {
		if strings.EqualFold(k, key) {
			return value
		}
	}
	return nil
}

func codeBuildDefaultString(payload map[string]any, key, fallback string) string {
	value := codeBuildPayloadValue(payload, key)
	text := strings.TrimSpace(codeBuildToString(value))
	if text == "" {
		return fallback
	}
	return text
}

func codeBuildDefaultInt(payload map[string]any, key string, fallback int) int {
	value := codeBuildPayloadValue(payload, key)
	switch v := value.(type) {
	case int:
		if v > 0 {
			return v
		}
	case int64:
		if v > 0 {
			return int(v)
		}
	case float64:
		if v > 0 {
			return int(v)
		}
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		var parsed int
		_, _ = fmt.Sscanf(strings.TrimSpace(v), "%d", &parsed)
		if parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func codeBuildToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func codeBuildStringSlice(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
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
			text := strings.TrimSpace(codeBuildToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(codeBuildToString(v))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}

func codeBuildCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func codeBuildRemoveString(values []string, target string) []string {
	if len(values) == 0 {
		return values
	}
	out := values[:0]
	for _, value := range values {
		if value == target {
			continue
		}
		out = append(out, value)
	}
	return out
}

func codeBuildToAnySlice(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}
