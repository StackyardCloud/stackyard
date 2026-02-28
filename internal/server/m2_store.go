package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type m2Store struct {
	mu sync.Mutex

	nextApplication int64
	nextEnvironment int64
	nextDeployment  int64
	nextTask        int64
	nextExecution   int64

	applications    map[string]map[string]any
	applicationVers map[string][]map[string]any
	environments    map[string]map[string]any
	deployments     map[string]map[string]map[string]any
	dataSets        map[string]map[string]map[string]any
	importTasks     map[string]map[string]map[string]any
	exportTasks     map[string]map[string]map[string]any
	batchExecutions map[string]map[string]map[string]any
	batchDefs       []map[string]any
	tags            map[string]map[string]string
}

func newM2Store() *m2Store {
	s := &m2Store{
		nextApplication: 2,
		nextEnvironment: 2,
		nextDeployment:  2,
		nextTask:        2,
		nextExecution:   2,
		applications:    map[string]map[string]any{},
		applicationVers: map[string][]map[string]any{},
		environments:    map[string]map[string]any{},
		deployments:     map[string]map[string]map[string]any{},
		dataSets:        map[string]map[string]map[string]any{},
		importTasks:     map[string]map[string]map[string]any{},
		exportTasks:     map[string]map[string]map[string]any{},
		batchExecutions: map[string]map[string]map[string]any{},
		tags:            map[string]map[string]string{},
		batchDefs: []map[string]any{
			{"name": "DAILY-RECON", "type": "JCL", "status": "AVAILABLE"},
			{"name": "NIGHTLY-CLOSE", "type": "JCL", "status": "AVAILABLE"},
		},
	}

	app := s.ensureApplicationLocked("app-00000001")
	env := s.ensureEnvironmentLocked("env-00000001")
	env["applicationId"] = m2StringAny(app, "applicationId", "app-00000001")
	_ = s.ensureDeploymentLocked("app-00000001", "dep-00000001")
	_ = s.ensureDataSetLocked("app-00000001", "SYS1.PARMLIB")
	_ = s.ensureImportTaskLocked("app-00000001", "task-00000001")
	_ = s.ensureExportTaskLocked("app-00000001", "task-00000001")
	_ = s.ensureBatchExecutionLocked("app-00000001", "exec-00000001")
	s.tags[m2ApplicationARN("app-00000001")] = map[string]string{"seed": "true"}

	return s
}

func (s *m2Store) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultAppID := "app-00000001"
	defaultEnvID := "env-00000001"

	applicationID := m2DefaultString(pathParams, "applicationId", "")
	if applicationID == "" {
		applicationID = m2DefaultStringAny(payload, "applicationId", defaultAppID)
	}
	if strings.TrimSpace(applicationID) == "" {
		applicationID = defaultAppID
	}

	environmentID := m2DefaultString(pathParams, "environmentId", "")
	if environmentID == "" {
		environmentID = m2DefaultStringAny(payload, "environmentId", defaultEnvID)
	}
	if strings.TrimSpace(environmentID) == "" {
		environmentID = defaultEnvID
	}

	switch action {
	case "CreateApplication":
		applicationID = fmt.Sprintf("app-%08d", s.nextApplicationIDLocked())
		app := s.ensureApplicationLocked(applicationID)
		app["name"] = m2DefaultStringAny(payload, "name", fmt.Sprintf("stackyard-m2-%s", applicationID))
		app["description"] = m2DefaultStringAny(payload, "description", "Mainframe application")
		app["engineType"] = m2DefaultStringAny(payload, "engineType", "microfocus")
		app["status"] = "CREATED"
		app["updatedAt"] = time.Now().UTC()
		s.ensureVersionLocked(applicationID, "1")
		return map[string]any{"applicationId": applicationID, "application": m2CloneMap(app)}

	case "DeleteApplication":
		app := s.ensureApplicationLocked(applicationID)
		app["status"] = "DELETING"
		app["updatedAt"] = time.Now().UTC()
		delete(s.applications, applicationID)
		delete(s.applicationVers, applicationID)
		delete(s.deployments, applicationID)
		delete(s.dataSets, applicationID)
		delete(s.importTasks, applicationID)
		delete(s.exportTasks, applicationID)
		delete(s.batchExecutions, applicationID)
		return map[string]any{"applicationId": applicationID}

	case "GetApplication":
		return map[string]any{"application": m2CloneMap(s.ensureApplicationLocked(applicationID))}

	case "UpdateApplication":
		app := s.ensureApplicationLocked(applicationID)
		if v := m2DefaultStringAny(payload, "name", ""); v != "" {
			app["name"] = v
		}
		if v := m2DefaultStringAny(payload, "description", ""); v != "" {
			app["description"] = v
		}
		if v := m2DefaultStringAny(payload, "engineType", ""); v != "" {
			app["engineType"] = v
		}
		app["updatedAt"] = time.Now().UTC()
		return map[string]any{"application": m2CloneMap(app)}

	case "ListApplications":
		items := make([]any, 0, len(s.applications))
		keys := make([]string, 0, len(s.applications))
		for k := range s.applications {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, id := range keys {
			app := s.applications[id]
			items = append(items, map[string]any{
				"applicationId": m2StringAny(app, "applicationId", id),
				"name":          m2StringAny(app, "name", id),
				"engineType":    m2StringAny(app, "engineType", "microfocus"),
				"status":        m2StringAny(app, "status", "AVAILABLE"),
			})
		}
		return map[string]any{"applications": items, "nextToken": ""}

	case "GetApplicationVersion":
		applicationVersion := m2DefaultString(pathParams, "applicationVersion", m2DefaultStringAny(payload, "applicationVersion", "1"))
		version := s.ensureVersionLocked(applicationID, applicationVersion)
		return map[string]any{"applicationVersion": m2CloneMap(version)}

	case "ListApplicationVersions":
		versions := s.applicationVers[applicationID]
		if len(versions) == 0 {
			_ = s.ensureVersionLocked(applicationID, "1")
			versions = s.applicationVers[applicationID]
		}
		out := make([]any, 0, len(versions))
		for _, v := range versions {
			out = append(out, m2CloneMap(v))
		}
		return map[string]any{"applicationVersions": out, "nextToken": ""}

	case "CreateEnvironment":
		environmentID = fmt.Sprintf("env-%08d", s.nextEnvironmentIDLocked())
		env := s.ensureEnvironmentLocked(environmentID)
		env["name"] = m2DefaultStringAny(payload, "name", fmt.Sprintf("stackyard-env-%s", environmentID))
		env["description"] = m2DefaultStringAny(payload, "description", "Mainframe environment")
		env["engineType"] = m2DefaultStringAny(payload, "engineType", "microfocus")
		env["status"] = "AVAILABLE"
		if v := m2DefaultStringAny(payload, "applicationId", ""); v != "" {
			env["applicationId"] = v
		}
		env["updatedAt"] = time.Now().UTC()
		return map[string]any{"environment": m2CloneMap(env)}

	case "GetEnvironment":
		return map[string]any{"environment": m2CloneMap(s.ensureEnvironmentLocked(environmentID))}

	case "UpdateEnvironment":
		env := s.ensureEnvironmentLocked(environmentID)
		if v := m2DefaultStringAny(payload, "name", ""); v != "" {
			env["name"] = v
		}
		if v := m2DefaultStringAny(payload, "description", ""); v != "" {
			env["description"] = v
		}
		if v := m2DefaultStringAny(payload, "applicationId", ""); v != "" {
			env["applicationId"] = v
		}
		if v := m2DefaultStringAny(payload, "engineType", ""); v != "" {
			env["engineType"] = v
		}
		env["updatedAt"] = time.Now().UTC()
		return map[string]any{"environment": m2CloneMap(env)}

	case "DeleteEnvironment":
		env := s.ensureEnvironmentLocked(environmentID)
		env["status"] = "DELETING"
		env["updatedAt"] = time.Now().UTC()
		delete(s.environments, environmentID)
		return map[string]any{"environmentId": environmentID}

	case "ListEnvironments":
		engineTypeFilter := strings.TrimSpace(query.Get("engineType"))
		items := make([]any, 0, len(s.environments))
		keys := make([]string, 0, len(s.environments))
		for k := range s.environments {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, id := range keys {
			env := s.environments[id]
			engineType := m2StringAny(env, "engineType", "microfocus")
			if engineTypeFilter != "" && !strings.EqualFold(engineTypeFilter, engineType) {
				continue
			}
			items = append(items, map[string]any{
				"environmentId": m2StringAny(env, "environmentId", id),
				"name":          m2StringAny(env, "name", id),
				"engineType":    engineType,
				"status":        m2StringAny(env, "status", "AVAILABLE"),
			})
		}
		return map[string]any{"environments": items, "nextToken": ""}

	case "DeleteApplicationFromEnvironment":
		env := s.ensureEnvironmentLocked(environmentID)
		env["applicationId"] = ""
		env["updatedAt"] = time.Now().UTC()
		return map[string]any{}

	case "CreateDeployment":
		deploymentID := fmt.Sprintf("dep-%08d", s.nextDeploymentIDLocked())
		deployment := s.ensureDeploymentLocked(applicationID, deploymentID)
		deployment["environmentId"] = m2DefaultStringAny(payload, "environmentId", environmentID)
		deployment["status"] = "DEPLOYED"
		deployment["createdAt"] = time.Now().UTC()
		return map[string]any{"deployment": m2CloneMap(deployment)}

	case "GetDeployment":
		deploymentID := m2DefaultString(pathParams, "deploymentId", m2DefaultStringAny(payload, "deploymentId", "dep-00000001"))
		return map[string]any{"deployment": m2CloneMap(s.ensureDeploymentLocked(applicationID, deploymentID))}

	case "ListDeployments":
		items := s.listDeploymentsLocked(applicationID)
		return map[string]any{"deployments": items, "nextToken": ""}

	case "CreateDataSetImportTask":
		taskID := fmt.Sprintf("task-%08d", s.nextTaskIDLocked())
		task := s.ensureImportTaskLocked(applicationID, taskID)
		task["status"] = "COMPLETED"
		task["createdAt"] = time.Now().UTC()
		return map[string]any{"taskId": taskID, "task": m2CloneMap(task)}

	case "GetDataSetImportTask":
		taskID := m2DefaultString(pathParams, "taskId", m2DefaultStringAny(payload, "taskId", "task-00000001"))
		return map[string]any{"task": m2CloneMap(s.ensureImportTaskLocked(applicationID, taskID))}

	case "ListDataSetImportHistory":
		items := s.listImportTasksLocked(applicationID)
		return map[string]any{"dataSetImportTasks": items, "nextToken": ""}

	case "CreateDataSetExportTask":
		taskID := fmt.Sprintf("task-%08d", s.nextTaskIDLocked())
		task := s.ensureExportTaskLocked(applicationID, taskID)
		task["status"] = "COMPLETED"
		task["createdAt"] = time.Now().UTC()
		return map[string]any{"taskId": taskID, "task": m2CloneMap(task)}

	case "GetDataSetExportTask":
		taskID := m2DefaultString(pathParams, "taskId", m2DefaultStringAny(payload, "taskId", "task-00000001"))
		return map[string]any{"task": m2CloneMap(s.ensureExportTaskLocked(applicationID, taskID))}

	case "ListDataSetExportHistory":
		items := s.listExportTasksLocked(applicationID)
		return map[string]any{"dataSetExportTasks": items, "nextToken": ""}

	case "GetDataSetDetails":
		dataSetName := m2DefaultString(pathParams, "dataSetName", m2DefaultStringAny(payload, "dataSetName", "SYS1.PARMLIB"))
		return map[string]any{"dataSet": m2CloneMap(s.ensureDataSetLocked(applicationID, dataSetName))}

	case "ListDataSets":
		items := s.listDataSetsLocked(applicationID)
		return map[string]any{"dataSets": items, "nextToken": ""}

	case "StartBatchJob":
		executionID := fmt.Sprintf("exec-%08d", s.nextExecutionIDLocked())
		exec := s.ensureBatchExecutionLocked(applicationID, executionID)
		exec["status"] = "RUNNING"
		exec["startedAt"] = time.Now().UTC()
		if name := m2DefaultStringAny(payload, "jobName", ""); name != "" {
			exec["jobName"] = name
		}
		return map[string]any{"executionId": executionID, "batchJobExecution": m2CloneMap(exec)}

	case "CancelBatchJobExecution":
		executionID := m2DefaultString(pathParams, "executionId", m2DefaultStringAny(payload, "executionId", "exec-00000001"))
		exec := s.ensureBatchExecutionLocked(applicationID, executionID)
		exec["status"] = "CANCELLED"
		exec["updatedAt"] = time.Now().UTC()
		return map[string]any{"batchJobExecution": m2CloneMap(exec)}

	case "GetBatchJobExecution":
		executionID := m2DefaultString(pathParams, "executionId", m2DefaultStringAny(payload, "executionId", "exec-00000001"))
		return map[string]any{"batchJobExecution": m2CloneMap(s.ensureBatchExecutionLocked(applicationID, executionID))}

	case "ListBatchJobExecutions":
		items := s.listBatchExecutionsLocked(applicationID)
		return map[string]any{"batchJobExecutions": items, "nextToken": ""}

	case "ListBatchJobDefinitions":
		out := make([]any, 0, len(s.batchDefs))
		for _, item := range s.batchDefs {
			out = append(out, m2CloneMap(item))
		}
		return map[string]any{"batchJobDefinitions": out, "nextToken": ""}

	case "ListBatchJobRestartPoints":
		executionID := m2DefaultString(pathParams, "executionId", m2DefaultStringAny(payload, "executionId", "exec-00000001"))
		exec := s.ensureBatchExecutionLocked(applicationID, executionID)
		steps, _ := exec["steps"].([]any)
		if len(steps) == 0 {
			steps = []any{
				map[string]any{"stepName": "STEP0001", "restartAllowed": true},
				map[string]any{"stepName": "STEP0002", "restartAllowed": true},
			}
		}
		return map[string]any{"restartPoints": steps}

	case "StartApplication":
		app := s.ensureApplicationLocked(applicationID)
		app["status"] = "RUNNING"
		app["updatedAt"] = time.Now().UTC()
		return map[string]any{"application": m2CloneMap(app)}

	case "StopApplication":
		app := s.ensureApplicationLocked(applicationID)
		app["status"] = "STOPPED"
		app["updatedAt"] = time.Now().UTC()
		return map[string]any{"application": m2CloneMap(app)}

	case "ListEngineVersions":
		engineTypeFilter := strings.TrimSpace(query.Get("engineType"))
		versions := []map[string]any{
			{"engineType": "microfocus", "version": "8.0"},
			{"engineType": "bluage", "version": "4.2"},
		}
		out := make([]any, 0, len(versions))
		for _, v := range versions {
			engineType := m2StringAny(v, "engineType", "")
			if engineTypeFilter != "" && !strings.EqualFold(engineTypeFilter, engineType) {
				continue
			}
			out = append(out, m2CloneMap(v))
		}
		return map[string]any{"engineVersions": out, "nextToken": ""}

	case "GetSignedBluinsightsUrl":
		return map[string]any{"signedBiUrl": "https://example.com/bluinsights/signed-url", "expiresInSeconds": 900}

	case "TagResource":
		resourceARN := m2DefaultString(pathParams, "resourceArn", m2DefaultStringAny(payload, "resourceArn", m2ApplicationARN(applicationID)))
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for key, value := range m2StringMap(payload, "tags") {
			s.tags[resourceARN][key] = value
		}
		if len(s.tags[resourceARN]) == 0 {
			for key, value := range m2StringMap(payload, "Tags") {
				s.tags[resourceARN][key] = value
			}
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := m2DefaultString(pathParams, "resourceArn", m2DefaultStringAny(payload, "resourceArn", m2ApplicationARN(applicationID)))
		tagKeys := m2StringSlice(payload, "tagKeys")
		if len(tagKeys) == 0 {
			tagKeys = m2StringSlice(payload, "TagKeys")
		}
		if len(tagKeys) == 0 {
			tagKeys = query["tagKeys"]
		}
		for _, key := range tagKeys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := m2DefaultString(pathParams, "resourceArn", m2DefaultStringAny(payload, "resourceArn", m2ApplicationARN(applicationID)))
		return map[string]any{"tags": m2CloneStringMap(s.tags[resourceARN])}
	}

	return map[string]any{}
}

func (s *m2Store) ensureApplicationLocked(applicationID string) map[string]any {
	id := strings.TrimSpace(applicationID)
	if id == "" {
		id = "app-00000001"
	}
	if app := s.applications[id]; app != nil {
		return app
	}
	app := map[string]any{
		"applicationId":  id,
		"name":           fmt.Sprintf("stackyard-%s", id),
		"description":    "Mainframe modernization application",
		"engineType":     "microfocus",
		"status":         "AVAILABLE",
		"applicationArn": m2ApplicationARN(id),
		"createdAt":      time.Now().UTC(),
	}
	s.applications[id] = app
	s.ensureVersionLocked(id, "1")
	return app
}

func (s *m2Store) ensureVersionLocked(applicationID, version string) map[string]any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	v := strings.TrimSpace(version)
	if v == "" {
		v = "1"
	}
	if s.applicationVers[appID] == nil {
		s.applicationVers[appID] = []map[string]any{}
	}
	for _, item := range s.applicationVers[appID] {
		if m2StringAny(item, "applicationVersion", "") == v {
			return item
		}
	}
	item := map[string]any{
		"applicationId":      appID,
		"applicationVersion": v,
		"status":             "AVAILABLE",
		"description":        "Initial version",
		"createdAt":          time.Now().UTC(),
	}
	s.applicationVers[appID] = append(s.applicationVers[appID], item)
	return item
}

func (s *m2Store) ensureEnvironmentLocked(environmentID string) map[string]any {
	id := strings.TrimSpace(environmentID)
	if id == "" {
		id = "env-00000001"
	}
	if env := s.environments[id]; env != nil {
		return env
	}
	env := map[string]any{
		"environmentId":  id,
		"name":           fmt.Sprintf("stackyard-%s", id),
		"description":    "Mainframe modernization environment",
		"engineType":     "microfocus",
		"status":         "AVAILABLE",
		"environmentArn": m2EnvironmentARN(id),
		"createdAt":      time.Now().UTC(),
	}
	s.environments[id] = env
	return env
}

func (s *m2Store) ensureDeploymentLocked(applicationID, deploymentID string) map[string]any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	_ = s.ensureApplicationLocked(appID)
	if s.deployments[appID] == nil {
		s.deployments[appID] = map[string]map[string]any{}
	}
	id := strings.TrimSpace(deploymentID)
	if id == "" {
		id = "dep-00000001"
	}
	if dep := s.deployments[appID][id]; dep != nil {
		return dep
	}
	dep := map[string]any{
		"deploymentId":  id,
		"applicationId": appID,
		"status":        "COMPLETED",
		"createdAt":     time.Now().UTC(),
	}
	s.deployments[appID][id] = dep
	return dep
}

func (s *m2Store) ensureDataSetLocked(applicationID, dataSetName string) map[string]any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	_ = s.ensureApplicationLocked(appID)
	if s.dataSets[appID] == nil {
		s.dataSets[appID] = map[string]map[string]any{}
	}
	name := strings.TrimSpace(dataSetName)
	if name == "" {
		name = "SYS1.PARMLIB"
	}
	if ds := s.dataSets[appID][name]; ds != nil {
		return ds
	}
	ds := map[string]any{
		"applicationId": appID,
		"name":          name,
		"type":          "PO",
		"status":        "AVAILABLE",
	}
	s.dataSets[appID][name] = ds
	return ds
}

func (s *m2Store) ensureImportTaskLocked(applicationID, taskID string) map[string]any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	_ = s.ensureApplicationLocked(appID)
	if s.importTasks[appID] == nil {
		s.importTasks[appID] = map[string]map[string]any{}
	}
	id := strings.TrimSpace(taskID)
	if id == "" {
		id = "task-00000001"
	}
	if task := s.importTasks[appID][id]; task != nil {
		return task
	}
	task := map[string]any{
		"taskId":        id,
		"applicationId": appID,
		"status":        "COMPLETED",
		"createdAt":     time.Now().UTC(),
	}
	s.importTasks[appID][id] = task
	return task
}

func (s *m2Store) ensureExportTaskLocked(applicationID, taskID string) map[string]any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	_ = s.ensureApplicationLocked(appID)
	if s.exportTasks[appID] == nil {
		s.exportTasks[appID] = map[string]map[string]any{}
	}
	id := strings.TrimSpace(taskID)
	if id == "" {
		id = "task-00000001"
	}
	if task := s.exportTasks[appID][id]; task != nil {
		return task
	}
	task := map[string]any{
		"taskId":        id,
		"applicationId": appID,
		"status":        "COMPLETED",
		"createdAt":     time.Now().UTC(),
	}
	s.exportTasks[appID][id] = task
	return task
}

func (s *m2Store) ensureBatchExecutionLocked(applicationID, executionID string) map[string]any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	_ = s.ensureApplicationLocked(appID)
	if s.batchExecutions[appID] == nil {
		s.batchExecutions[appID] = map[string]map[string]any{}
	}
	id := strings.TrimSpace(executionID)
	if id == "" {
		id = "exec-00000001"
	}
	if exec := s.batchExecutions[appID][id]; exec != nil {
		return exec
	}
	exec := map[string]any{
		"executionId":   id,
		"applicationId": appID,
		"jobName":       "DAILY-RECON",
		"status":        "COMPLETED",
		"steps": []any{
			map[string]any{"stepName": "STEP0001", "restartAllowed": true},
			map[string]any{"stepName": "STEP0002", "restartAllowed": true},
		},
		"createdAt": time.Now().UTC(),
	}
	s.batchExecutions[appID][id] = exec
	return exec
}

func (s *m2Store) listDeploymentsLocked(applicationID string) []any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	if s.deployments[appID] == nil {
		_ = s.ensureDeploymentLocked(appID, "dep-00000001")
	}
	keys := make([]string, 0, len(s.deployments[appID]))
	for k := range s.deployments[appID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, m2CloneMap(s.deployments[appID][key]))
	}
	return out
}

func (s *m2Store) listDataSetsLocked(applicationID string) []any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	if s.dataSets[appID] == nil {
		_ = s.ensureDataSetLocked(appID, "SYS1.PARMLIB")
	}
	keys := make([]string, 0, len(s.dataSets[appID]))
	for k := range s.dataSets[appID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, m2CloneMap(s.dataSets[appID][key]))
	}
	return out
}

func (s *m2Store) listImportTasksLocked(applicationID string) []any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	if s.importTasks[appID] == nil {
		_ = s.ensureImportTaskLocked(appID, "task-00000001")
	}
	keys := make([]string, 0, len(s.importTasks[appID]))
	for k := range s.importTasks[appID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, m2CloneMap(s.importTasks[appID][key]))
	}
	return out
}

func (s *m2Store) listExportTasksLocked(applicationID string) []any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	if s.exportTasks[appID] == nil {
		_ = s.ensureExportTaskLocked(appID, "task-00000001")
	}
	keys := make([]string, 0, len(s.exportTasks[appID]))
	for k := range s.exportTasks[appID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, m2CloneMap(s.exportTasks[appID][key]))
	}
	return out
}

func (s *m2Store) listBatchExecutionsLocked(applicationID string) []any {
	appID := strings.TrimSpace(applicationID)
	if appID == "" {
		appID = "app-00000001"
	}
	if s.batchExecutions[appID] == nil {
		_ = s.ensureBatchExecutionLocked(appID, "exec-00000001")
	}
	keys := make([]string, 0, len(s.batchExecutions[appID]))
	for k := range s.batchExecutions[appID] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, m2CloneMap(s.batchExecutions[appID][key]))
	}
	return out
}

func (s *m2Store) nextApplicationIDLocked() int64 {
	id := s.nextApplication
	s.nextApplication++
	return id
}

func (s *m2Store) nextEnvironmentIDLocked() int64 {
	id := s.nextEnvironment
	s.nextEnvironment++
	return id
}

func (s *m2Store) nextDeploymentIDLocked() int64 {
	id := s.nextDeployment
	s.nextDeployment++
	return id
}

func (s *m2Store) nextTaskIDLocked() int64 {
	id := s.nextTask
	s.nextTask++
	return id
}

func (s *m2Store) nextExecutionIDLocked() int64 {
	id := s.nextExecution
	s.nextExecution++
	return id
}

func m2ApplicationARN(applicationID string) string {
	id := strings.TrimSpace(applicationID)
	if id == "" {
		id = "app-00000001"
	}
	return "arn:aws:m2:us-east-1:123456789012:application/" + id
}

func m2EnvironmentARN(environmentID string) string {
	id := strings.TrimSpace(environmentID)
	if id == "" {
		id = "env-00000001"
	}
	return "arn:aws:m2:us-east-1:123456789012:environment/" + id
}

func m2CloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func m2CloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func m2DefaultString(values map[string]string, key, fallback string) string {
	if values == nil {
		return fallback
	}
	v := strings.TrimSpace(values[key])
	if v == "" {
		return fallback
	}
	return v
}

func m2DefaultStringAny(values map[string]any, key, fallback string) string {
	if values == nil {
		return fallback
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return fallback
	}
	if s, ok := raw.(string); ok {
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return fallback
}

func m2StringAny(values map[string]any, key, fallback string) string {
	if values == nil {
		return fallback
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return fallback
	}
	if s, ok := raw.(string); ok {
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return fallback
}

func m2StringMap(values map[string]any, key string) map[string]string {
	out := map[string]string{}
	if values == nil {
		return out
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return out
	}
	typed, ok := raw.(map[string]any)
	if !ok {
		return out
	}
	for k, v := range typed {
		keyName := strings.TrimSpace(k)
		if keyName == "" {
			continue
		}
		if s, ok := v.(string); ok {
			out[keyName] = s
		}
	}
	return out
}

func m2StringSlice(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return nil
	}
	typed, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(typed))
	for _, item := range typed {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
