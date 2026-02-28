package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type migrationHubOrchestratorStore struct {
	mu sync.Mutex

	nextTemplateID      int64
	nextWorkflowID      int64
	nextStepGroupID     int64
	nextStepID          int64
	templates           map[string]map[string]any
	templateStepGroups  map[string]map[string]map[string]any
	templateSteps       map[string]map[string]map[string]any
	workflows           map[string]map[string]any
	workflowStepGroups  map[string]map[string]map[string]any
	workflowSteps       map[string]map[string]map[string]map[string]any
	workflowStepByID    map[string]map[string]any
	tags                map[string]map[string]string
	plugins             []map[string]any
	defaultTemplateID   string
	defaultWorkflowID   string
	defaultStepGroupID  string
	defaultWorkflowStep string
}

func newMigrationHubOrchestratorStore() *migrationHubOrchestratorStore {
	s := &migrationHubOrchestratorStore{
		nextTemplateID:     2,
		nextWorkflowID:     2,
		nextStepGroupID:    2,
		nextStepID:         2,
		templates:          map[string]map[string]any{},
		templateStepGroups: map[string]map[string]map[string]any{},
		templateSteps:      map[string]map[string]map[string]any{},
		workflows:          map[string]map[string]any{},
		workflowStepGroups: map[string]map[string]map[string]any{},
		workflowSteps:      map[string]map[string]map[string]map[string]any{},
		workflowStepByID:   map[string]map[string]any{},
		tags:               map[string]map[string]string{},
		plugins: []map[string]any{
			{
				"name":            "AWSApplicationMigrationService",
				"version":         "1.0.0",
				"description":     "Stackyard seeded plugin",
				"owner":           "AWS",
				"hostname":        "stackyard.local",
				"category":        "MIGRATION",
				"pluginId":        "plugin-00000001",
				"creationTime":    time.Now().UTC().Format(time.RFC3339),
				"lastUpdatedTime": time.Now().UTC().Format(time.RFC3339),
			},
		},
		defaultTemplateID:   "tmpl-00000001",
		defaultWorkflowID:   "mwf-00000001",
		defaultStepGroupID:  "wsg-00000001",
		defaultWorkflowStep: "wstep-00000001",
	}

	template := s.ensureTemplateLocked(s.defaultTemplateID)
	templateGroup := s.ensureTemplateStepGroupLocked(s.defaultTemplateID, "tsg-00000001")
	templateStep := s.ensureTemplateStepLocked(s.defaultTemplateID, "tstep-00000001")
	templateStep["stepGroupId"] = mhoFirstNonEmpty(mhoStringAny(templateGroup, "id"), "tsg-00000001")

	workflow := s.ensureWorkflowLocked(s.defaultWorkflowID)
	workflow["templateId"] = mhoFirstNonEmpty(mhoStringAny(template, "id"), s.defaultTemplateID)
	workflow["status"] = "CREATED"
	s.ensureWorkflowStepGroupLocked(s.defaultWorkflowID, s.defaultStepGroupID)
	workflowStep := s.ensureWorkflowStepLocked(s.defaultWorkflowID, s.defaultStepGroupID, s.defaultWorkflowStep)
	workflowStep["status"] = "READY"
	workflowStep["stepGroupId"] = s.defaultStepGroupID

	s.tags[mhoFirstNonEmpty(mhoStringAny(workflow, "arn"), mhoWorkflowARN(s.defaultWorkflowID))] = map[string]string{
		"seed": "true",
	}
	return s
}

func (s *migrationHubOrchestratorStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := mhoFirstNonEmpty(
		mhoPathParam(pathParams, "id"),
		mhoStringAny(payload, "id", "Id"),
	)
	templateID := mhoFirstNonEmpty(
		mhoPathParam(pathParams, "templateId"),
		mhoStringAny(payload, "templateId", "TemplateId"),
		mhoQueryFirst(query, "templateId"),
		s.defaultTemplateID,
	)
	workflowID := mhoFirstNonEmpty(
		mhoPathParam(pathParams, "workflowId"),
		mhoStringAny(payload, "workflowId", "WorkflowId"),
		mhoQueryFirst(query, "workflowId"),
		s.defaultWorkflowID,
	)
	stepGroupID := mhoFirstNonEmpty(
		mhoPathParam(pathParams, "stepGroupId"),
		mhoStringAny(payload, "stepGroupId", "StepGroupId"),
		mhoQueryFirst(query, "stepGroupId"),
		s.defaultStepGroupID,
	)
	resourceARN := mhoFirstNonEmpty(
		mhoPathParam(pathParams, "resourceArn"),
		mhoStringAny(payload, "resourceArn", "ResourceArn"),
		mhoWorkflowARN(workflowID),
	)

	switch action {
	case "CreateTemplate":
		templateID = fmt.Sprintf("tmpl-%08d", s.nextTemplateIDLocked())
		template := s.ensureTemplateLocked(templateID)
		template["name"] = mhoFirstNonEmpty(mhoStringAny(payload, "name", "templateName"), fmt.Sprintf("stackyard-template-%s", templateID))
		template["description"] = mhoFirstNonEmpty(mhoStringAny(payload, "description"), "Migration Hub Orchestrator template")
		template["status"] = "ACTIVE"
		template["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		templateGroup := s.ensureTemplateStepGroupLocked(templateID, fmt.Sprintf("tsg-%08d", s.nextStepGroupIDLocked()))
		templateStep := s.ensureTemplateStepLocked(templateID, fmt.Sprintf("tstep-%08d", s.nextStepIDLocked()))
		templateStep["stepGroupId"] = mhoFirstNonEmpty(mhoStringAny(templateGroup, "id"), "tsg-00000001")
		return map[string]any{
			"id":              templateID,
			"templateSummary": mhoCloneMap(template),
		}

	case "GetTemplate":
		template := s.ensureTemplateLocked(mhoFirstNonEmpty(id, templateID))
		return map[string]any{
			"template": mhoCloneMap(template),
		}

	case "ListTemplates":
		items := make([]any, 0, len(s.templates))
		for _, key := range mhoSortedKeys(s.templates) {
			items = append(items, mhoCloneMap(s.templates[key]))
		}
		return map[string]any{
			"templateSummary": items,
			"nextToken":       "",
		}

	case "UpdateTemplate":
		template := s.ensureTemplateLocked(mhoFirstNonEmpty(id, templateID))
		for k, v := range payload {
			template[k] = v
		}
		template["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"id":              mhoFirstNonEmpty(mhoStringAny(template, "id"), templateID),
			"templateSummary": mhoCloneMap(template),
		}

	case "DeleteTemplate":
		templateID = mhoFirstNonEmpty(id, templateID)
		delete(s.templates, templateID)
		delete(s.templateStepGroups, templateID)
		delete(s.templateSteps, templateID)
		return map[string]any{"id": templateID}

	case "GetTemplateStep":
		step := s.ensureTemplateStepLocked(templateID, mhoFirstNonEmpty(id, "tstep-00000001"))
		return map[string]any{"templateStep": mhoCloneMap(step)}

	case "ListTemplateSteps":
		templateSteps := s.ensureTemplateStepsForTemplateLocked(templateID)
		items := make([]any, 0, len(templateSteps))
		for _, key := range mhoSortedKeys(templateSteps) {
			items = append(items, mhoCloneMap(templateSteps[key]))
		}
		return map[string]any{"templateStepSummaryList": items, "nextToken": ""}

	case "GetTemplateStepGroup":
		group := s.ensureTemplateStepGroupLocked(templateID, mhoFirstNonEmpty(id, "tsg-00000001"))
		return map[string]any{"templateStepGroup": mhoCloneMap(group)}

	case "ListTemplateStepGroups":
		stepGroups := s.ensureTemplateStepGroupsForTemplateLocked(templateID)
		items := make([]any, 0, len(stepGroups))
		for _, key := range mhoSortedKeys(stepGroups) {
			items = append(items, mhoCloneMap(stepGroups[key]))
		}
		return map[string]any{"templateStepGroupSummaryList": items, "nextToken": ""}

	case "CreateWorkflow":
		workflowID = fmt.Sprintf("mwf-%08d", s.nextWorkflowIDLocked())
		workflow := s.ensureWorkflowLocked(workflowID)
		workflow["name"] = mhoFirstNonEmpty(mhoStringAny(payload, "name", "workflowName"), fmt.Sprintf("stackyard-workflow-%s", workflowID))
		workflow["templateId"] = mhoFirstNonEmpty(mhoStringAny(payload, "templateId", "TemplateId"), templateID)
		workflow["status"] = "CREATED"
		workflow["lastStartTime"] = ""
		workflow["lastStopTime"] = ""
		workflow["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		s.ensureWorkflowStepGroupLocked(workflowID, fmt.Sprintf("wsg-%08d", s.nextStepGroupIDLocked()))
		return map[string]any{
			"id":                      workflowID,
			"workflowSummary":         mhoCloneMap(workflow),
			"workflowArn":             mhoFirstNonEmpty(mhoStringAny(workflow, "arn"), mhoWorkflowARN(workflowID)),
			"workflowStatus":          mhoFirstNonEmpty(mhoStringAny(workflow, "status"), "CREATED"),
			"workflowStepGroupsCount": len(s.ensureWorkflowStepGroupsForWorkflowLocked(workflowID)),
		}

	case "GetWorkflow":
		workflow := s.ensureWorkflowLocked(mhoFirstNonEmpty(id, workflowID))
		return map[string]any{"workflow": mhoCloneMap(workflow)}

	case "ListWorkflows":
		items := make([]any, 0, len(s.workflows))
		for _, key := range mhoSortedKeys(s.workflows) {
			items = append(items, mhoCloneMap(s.workflows[key]))
		}
		return map[string]any{"migrationWorkflowSummary": items, "nextToken": ""}

	case "UpdateWorkflow":
		workflow := s.ensureWorkflowLocked(mhoFirstNonEmpty(id, workflowID))
		for k, v := range payload {
			workflow[k] = v
		}
		workflow["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"id": mhoFirstNonEmpty(mhoStringAny(workflow, "id"), workflowID), "workflowSummary": mhoCloneMap(workflow)}

	case "DeleteWorkflow":
		workflowID = mhoFirstNonEmpty(id, workflowID)
		delete(s.workflows, workflowID)
		delete(s.workflowStepGroups, workflowID)
		delete(s.workflowSteps, workflowID)
		for stepID, step := range s.workflowStepByID {
			if mhoStringAny(step, "workflowId") == workflowID {
				delete(s.workflowStepByID, stepID)
			}
		}
		return map[string]any{"id": workflowID}

	case "StartWorkflow":
		workflow := s.ensureWorkflowLocked(mhoFirstNonEmpty(id, workflowID))
		workflow["status"] = "RUNNING"
		workflow["lastStartTime"] = time.Now().UTC().Format(time.RFC3339)
		workflow["lastUpdatedTime"] = workflow["lastStartTime"]
		return map[string]any{"id": mhoFirstNonEmpty(mhoStringAny(workflow, "id"), workflowID), "status": "RUNNING"}

	case "StopWorkflow":
		workflow := s.ensureWorkflowLocked(mhoFirstNonEmpty(id, workflowID))
		workflow["status"] = "STOPPED"
		workflow["lastStopTime"] = time.Now().UTC().Format(time.RFC3339)
		workflow["lastUpdatedTime"] = workflow["lastStopTime"]
		return map[string]any{"id": mhoFirstNonEmpty(mhoStringAny(workflow, "id"), workflowID), "status": "STOPPED"}

	case "CreateWorkflowStepGroup":
		groupID := fmt.Sprintf("wsg-%08d", s.nextStepGroupIDLocked())
		group := s.ensureWorkflowStepGroupLocked(workflowID, groupID)
		group["name"] = mhoFirstNonEmpty(mhoStringAny(payload, "name"), fmt.Sprintf("stackyard-step-group-%s", groupID))
		group["status"] = "CREATED"
		group["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"id": groupID, "workflowStepGroupSummary": mhoCloneMap(group)}

	case "GetWorkflowStepGroup":
		group := s.ensureWorkflowStepGroupByIDLocked(mhoFirstNonEmpty(id, stepGroupID))
		return map[string]any{"workflowStepGroup": mhoCloneMap(group)}

	case "ListWorkflowStepGroups":
		groups := s.ensureWorkflowStepGroupsForWorkflowLocked(workflowID)
		items := make([]any, 0, len(groups))
		for _, key := range mhoSortedKeys(groups) {
			items = append(items, mhoCloneMap(groups[key]))
		}
		return map[string]any{"workflowStepGroupsSummary": items, "nextToken": ""}

	case "UpdateWorkflowStepGroup":
		group := s.ensureWorkflowStepGroupByIDLocked(mhoFirstNonEmpty(id, stepGroupID))
		for k, v := range payload {
			group[k] = v
		}
		group["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"id": mhoFirstNonEmpty(mhoStringAny(group, "id"), stepGroupID), "workflowStepGroupSummary": mhoCloneMap(group)}

	case "DeleteWorkflowStepGroup":
		groupID := mhoFirstNonEmpty(id, stepGroupID)
		for wfID, groups := range s.workflowStepGroups {
			if _, ok := groups[groupID]; ok {
				delete(groups, groupID)
				if stepsByGroup, ok := s.workflowSteps[wfID]; ok {
					delete(stepsByGroup, groupID)
				}
			}
		}
		for stepID, step := range s.workflowStepByID {
			if mhoStringAny(step, "stepGroupId") == groupID {
				delete(s.workflowStepByID, stepID)
			}
		}
		return map[string]any{"id": groupID}

	case "CreateWorkflowStep":
		stepID := fmt.Sprintf("wstep-%08d", s.nextStepIDLocked())
		step := s.ensureWorkflowStepLocked(workflowID, stepGroupID, stepID)
		step["name"] = mhoFirstNonEmpty(mhoStringAny(payload, "name"), fmt.Sprintf("stackyard-step-%s", stepID))
		step["status"] = "READY"
		step["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"id": stepID, "workflowStepSummary": mhoCloneMap(step)}

	case "GetWorkflowStep":
		step := s.ensureWorkflowStepByIDLocked(mhoFirstNonEmpty(id, s.defaultWorkflowStep))
		return map[string]any{"workflowStep": mhoCloneMap(step)}

	case "ListWorkflowSteps":
		stepsByGroup := s.ensureWorkflowStepsForGroupLocked(workflowID, stepGroupID)
		items := make([]any, 0, len(stepsByGroup))
		for _, key := range mhoSortedKeys(stepsByGroup) {
			items = append(items, mhoCloneMap(stepsByGroup[key]))
		}
		return map[string]any{"workflowStepsSummary": items, "nextToken": ""}

	case "UpdateWorkflowStep":
		step := s.ensureWorkflowStepByIDLocked(mhoFirstNonEmpty(id, s.defaultWorkflowStep))
		for k, v := range payload {
			step[k] = v
		}
		step["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{"id": mhoFirstNonEmpty(mhoStringAny(step, "id"), id), "workflowStepSummary": mhoCloneMap(step)}

	case "DeleteWorkflowStep":
		stepID := mhoFirstNonEmpty(id, s.defaultWorkflowStep)
		if step, ok := s.workflowStepByID[stepID]; ok {
			wfID := mhoStringAny(step, "workflowId")
			sgID := mhoStringAny(step, "stepGroupId")
			if byWF, ok := s.workflowSteps[wfID]; ok {
				if byGroup, ok := byWF[sgID]; ok {
					delete(byGroup, stepID)
				}
			}
		}
		delete(s.workflowStepByID, stepID)
		return map[string]any{"id": stepID}

	case "RetryWorkflowStep":
		step := s.ensureWorkflowStepByIDLocked(mhoFirstNonEmpty(id, s.defaultWorkflowStep))
		attempts := mhoIntAny(step["retryCount"]) + 1
		step["retryCount"] = attempts
		step["status"] = "COMPLETED"
		step["lastUpdatedTime"] = time.Now().UTC().Format(time.RFC3339)
		return map[string]any{
			"id":         mhoFirstNonEmpty(mhoStringAny(step, "id"), s.defaultWorkflowStep),
			"retryCount": attempts,
			"status":     "COMPLETED",
		}

	case "ListPlugins":
		items := make([]any, 0, len(s.plugins))
		for _, plugin := range s.plugins {
			items = append(items, mhoCloneMap(plugin))
		}
		return map[string]any{"plugins": items, "nextToken": ""}

	case "TagResource":
		tagMap := s.ensureTagsLocked(resourceARN)
		for k, v := range mhoStringMapAny(payload, "tags", "Tags") {
			tagMap[k] = v
		}
		return map[string]any{"resourceArn": resourceARN, "tags": mhoCloneStringMap(tagMap)}

	case "UntagResource":
		tagMap := s.ensureTagsLocked(resourceARN)
		for _, key := range mhoStringSliceAny(payload, "tagKeys", "TagKeys") {
			delete(tagMap, key)
		}
		for _, key := range query["tagKeys"] {
			if strings.TrimSpace(key) != "" {
				delete(tagMap, strings.TrimSpace(key))
			}
		}
		return map[string]any{"resourceArn": resourceARN, "tags": mhoCloneStringMap(tagMap)}

	case "ListTagsForResource":
		tagMap := s.ensureTagsLocked(resourceARN)
		return map[string]any{"tags": mhoCloneStringMap(tagMap)}
	}

	return map[string]any{}
}

func (s *migrationHubOrchestratorStore) nextTemplateIDLocked() int64 {
	id := s.nextTemplateID
	s.nextTemplateID++
	return id
}

func (s *migrationHubOrchestratorStore) nextWorkflowIDLocked() int64 {
	id := s.nextWorkflowID
	s.nextWorkflowID++
	return id
}

func (s *migrationHubOrchestratorStore) nextStepGroupIDLocked() int64 {
	id := s.nextStepGroupID
	s.nextStepGroupID++
	return id
}

func (s *migrationHubOrchestratorStore) nextStepIDLocked() int64 {
	id := s.nextStepID
	s.nextStepID++
	return id
}

func (s *migrationHubOrchestratorStore) ensureTemplateLocked(templateID string) map[string]any {
	templateID = mhoFirstNonEmpty(strings.TrimSpace(templateID), s.defaultTemplateID)
	template, ok := s.templates[templateID]
	if ok {
		return template
	}
	now := time.Now().UTC().Format(time.RFC3339)
	template = map[string]any{
		"id":              templateID,
		"arn":             mhoTemplateARN(templateID),
		"name":            fmt.Sprintf("stackyard-template-%s", templateID),
		"description":     "Stackyard seeded template",
		"status":          "ACTIVE",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	s.templates[templateID] = template
	return template
}

func (s *migrationHubOrchestratorStore) ensureTemplateStepGroupsForTemplateLocked(templateID string) map[string]map[string]any {
	templateID = mhoFirstNonEmpty(strings.TrimSpace(templateID), s.defaultTemplateID)
	s.ensureTemplateLocked(templateID)
	if s.templateStepGroups[templateID] == nil {
		s.templateStepGroups[templateID] = map[string]map[string]any{}
	}
	return s.templateStepGroups[templateID]
}

func (s *migrationHubOrchestratorStore) ensureTemplateStepGroupLocked(templateID, groupID string) map[string]any {
	templateID = mhoFirstNonEmpty(strings.TrimSpace(templateID), s.defaultTemplateID)
	groupID = mhoFirstNonEmpty(strings.TrimSpace(groupID), "tsg-00000001")
	groups := s.ensureTemplateStepGroupsForTemplateLocked(templateID)
	if group, ok := groups[groupID]; ok {
		return group
	}
	now := time.Now().UTC().Format(time.RFC3339)
	group := map[string]any{
		"id":              groupID,
		"templateId":      templateID,
		"name":            fmt.Sprintf("stackyard-template-step-group-%s", groupID),
		"description":     "Stackyard seeded template step group",
		"status":          "ACTIVE",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	groups[groupID] = group
	return group
}

func (s *migrationHubOrchestratorStore) ensureTemplateStepsForTemplateLocked(templateID string) map[string]map[string]any {
	templateID = mhoFirstNonEmpty(strings.TrimSpace(templateID), s.defaultTemplateID)
	s.ensureTemplateLocked(templateID)
	if s.templateSteps[templateID] == nil {
		s.templateSteps[templateID] = map[string]map[string]any{}
	}
	return s.templateSteps[templateID]
}

func (s *migrationHubOrchestratorStore) ensureTemplateStepLocked(templateID, stepID string) map[string]any {
	templateID = mhoFirstNonEmpty(strings.TrimSpace(templateID), s.defaultTemplateID)
	stepID = mhoFirstNonEmpty(strings.TrimSpace(stepID), "tstep-00000001")
	steps := s.ensureTemplateStepsForTemplateLocked(templateID)
	if step, ok := steps[stepID]; ok {
		return step
	}
	now := time.Now().UTC().Format(time.RFC3339)
	step := map[string]any{
		"id":              stepID,
		"templateId":      templateID,
		"name":            fmt.Sprintf("stackyard-template-step-%s", stepID),
		"description":     "Stackyard seeded template step",
		"status":          "READY",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	steps[stepID] = step
	return step
}

func (s *migrationHubOrchestratorStore) ensureWorkflowLocked(workflowID string) map[string]any {
	workflowID = mhoFirstNonEmpty(strings.TrimSpace(workflowID), s.defaultWorkflowID)
	if workflow, ok := s.workflows[workflowID]; ok {
		return workflow
	}
	now := time.Now().UTC().Format(time.RFC3339)
	workflow := map[string]any{
		"id":              workflowID,
		"arn":             mhoWorkflowARN(workflowID),
		"name":            fmt.Sprintf("stackyard-workflow-%s", workflowID),
		"templateId":      s.defaultTemplateID,
		"status":          "CREATED",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	s.workflows[workflowID] = workflow
	return workflow
}

func (s *migrationHubOrchestratorStore) ensureWorkflowStepGroupsForWorkflowLocked(workflowID string) map[string]map[string]any {
	workflowID = mhoFirstNonEmpty(strings.TrimSpace(workflowID), s.defaultWorkflowID)
	s.ensureWorkflowLocked(workflowID)
	if s.workflowStepGroups[workflowID] == nil {
		s.workflowStepGroups[workflowID] = map[string]map[string]any{}
	}
	return s.workflowStepGroups[workflowID]
}

func (s *migrationHubOrchestratorStore) ensureWorkflowStepGroupLocked(workflowID, groupID string) map[string]any {
	workflowID = mhoFirstNonEmpty(strings.TrimSpace(workflowID), s.defaultWorkflowID)
	groupID = mhoFirstNonEmpty(strings.TrimSpace(groupID), s.defaultStepGroupID)
	groups := s.ensureWorkflowStepGroupsForWorkflowLocked(workflowID)
	if group, ok := groups[groupID]; ok {
		return group
	}
	now := time.Now().UTC().Format(time.RFC3339)
	group := map[string]any{
		"id":              groupID,
		"workflowId":      workflowID,
		"name":            fmt.Sprintf("stackyard-workflow-step-group-%s", groupID),
		"description":     "Stackyard seeded workflow step group",
		"status":          "CREATED",
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	groups[groupID] = group
	return group
}

func (s *migrationHubOrchestratorStore) ensureWorkflowStepGroupByIDLocked(groupID string) map[string]any {
	groupID = mhoFirstNonEmpty(strings.TrimSpace(groupID), s.defaultStepGroupID)
	for _, groups := range s.workflowStepGroups {
		if group, ok := groups[groupID]; ok {
			return group
		}
	}
	return s.ensureWorkflowStepGroupLocked(s.defaultWorkflowID, groupID)
}

func (s *migrationHubOrchestratorStore) ensureWorkflowStepsForGroupLocked(workflowID, groupID string) map[string]map[string]any {
	workflowID = mhoFirstNonEmpty(strings.TrimSpace(workflowID), s.defaultWorkflowID)
	groupID = mhoFirstNonEmpty(strings.TrimSpace(groupID), s.defaultStepGroupID)
	s.ensureWorkflowStepGroupLocked(workflowID, groupID)
	if s.workflowSteps[workflowID] == nil {
		s.workflowSteps[workflowID] = map[string]map[string]map[string]any{}
	}
	if s.workflowSteps[workflowID][groupID] == nil {
		s.workflowSteps[workflowID][groupID] = map[string]map[string]any{}
	}
	return s.workflowSteps[workflowID][groupID]
}

func (s *migrationHubOrchestratorStore) ensureWorkflowStepLocked(workflowID, groupID, stepID string) map[string]any {
	workflowID = mhoFirstNonEmpty(strings.TrimSpace(workflowID), s.defaultWorkflowID)
	groupID = mhoFirstNonEmpty(strings.TrimSpace(groupID), s.defaultStepGroupID)
	stepID = mhoFirstNonEmpty(strings.TrimSpace(stepID), s.defaultWorkflowStep)
	steps := s.ensureWorkflowStepsForGroupLocked(workflowID, groupID)
	if step, ok := steps[stepID]; ok {
		return step
	}
	now := time.Now().UTC().Format(time.RFC3339)
	step := map[string]any{
		"id":              stepID,
		"workflowId":      workflowID,
		"stepGroupId":     groupID,
		"name":            fmt.Sprintf("stackyard-workflow-step-%s", stepID),
		"description":     "Stackyard seeded workflow step",
		"status":          "READY",
		"retryCount":      0,
		"creationTime":    now,
		"lastUpdatedTime": now,
	}
	steps[stepID] = step
	s.workflowStepByID[stepID] = step
	return step
}

func (s *migrationHubOrchestratorStore) ensureWorkflowStepByIDLocked(stepID string) map[string]any {
	stepID = mhoFirstNonEmpty(strings.TrimSpace(stepID), s.defaultWorkflowStep)
	if step, ok := s.workflowStepByID[stepID]; ok {
		return step
	}
	return s.ensureWorkflowStepLocked(s.defaultWorkflowID, s.defaultStepGroupID, stepID)
}

func (s *migrationHubOrchestratorStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = mhoWorkflowARN(s.defaultWorkflowID)
	}
	if s.tags[resourceARN] == nil {
		s.tags[resourceARN] = map[string]string{}
	}
	return s.tags[resourceARN]
}

func mhoTemplateARN(templateID string) string {
	return fmt.Sprintf("arn:aws:migrationhub-orchestrator:us-east-1:123456789012:template/%s", templateID)
}

func mhoWorkflowARN(workflowID string) string {
	return fmt.Sprintf("arn:aws:migrationhub-orchestrator:us-east-1:123456789012:workflow/%s", workflowID)
}

func mhoFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mhoStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			break
		}
		value, ok := m[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case fmt.Stringer:
			str := strings.TrimSpace(v.String())
			if str != "" {
				return str
			}
		}
	}
	return ""
}

func mhoPathParam(pathParams map[string]string, keys ...string) string {
	for _, key := range keys {
		if pathParams == nil {
			break
		}
		if value, ok := pathParams[key]; ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mhoQueryFirst(query url.Values, keys ...string) string {
	for _, key := range keys {
		if query == nil {
			break
		}
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func mhoIntAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func mhoSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mhoCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch v := value.(type) {
		case map[string]any:
			out[key] = mhoCloneMap(v)
		case []any:
			copied := make([]any, len(v))
			for i := range v {
				item := v[i]
				if vm, ok := item.(map[string]any); ok {
					copied[i] = mhoCloneMap(vm)
					continue
				}
				copied[i] = item
			}
			out[key] = copied
		default:
			out[key] = v
		}
	}
	return out
}

func mhoCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func mhoStringMapAny(payload map[string]any, keys ...string) map[string]string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case map[string]string:
			return mhoCloneStringMap(typed)
		case map[string]any:
			out := map[string]string{}
			for k, v := range typed {
				if str := strings.TrimSpace(fmt.Sprint(v)); str != "" {
					out[k] = str
				}
			}
			return out
		}
	}
	return map[string]string{}
}

func mhoStringSliceAny(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case []string:
			out := make([]string, 0, len(typed))
			for _, value := range typed {
				if trimmed := strings.TrimSpace(value); trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		case []any:
			out := make([]string, 0, len(typed))
			for _, value := range typed {
				trimmed := strings.TrimSpace(fmt.Sprint(value))
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				return nil
			}
			return []string{trimmed}
		}
	}
	return nil
}
