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
	fisDefaultRegion    = "us-east-1"
	fisDefaultAccountID = "123456789012"
)

type fisStore struct {
	mu sync.Mutex

	nextTemplateID int64
	nextExperiment int64

	actions                     map[string]map[string]any
	targetResourceTypes         map[string]map[string]any
	experimentTemplates         map[string]map[string]any
	experiments                 map[string]map[string]any
	targetAccountByTemplate     map[string]map[string]map[string]any
	targetAccountByExperiment   map[string]map[string]map[string]any
	resolvedTargetsByExperiment map[string][]map[string]any
	safetyLevers                map[string]map[string]any
	tags                        map[string]map[string]string
	createTemplateTokens        map[string]string
	startExperimentTokens       map[string]string
}

func newFISStore() *fisStore {
	now := time.Now().UTC().Format(time.RFC3339)

	templateID := "ext-000001"
	templateArn := fisTemplateARN(templateID)
	experimentID := "exp-000001"
	experimentArn := fisExperimentARN(experimentID)
	accountID := fisDefaultAccountID
	actionID := "aws:ec2:stop-instances"
	targetResourceTypeID := "aws:ec2:instance"

	s := &fisStore{
		nextTemplateID: 2,
		nextExperiment: 2,
		actions: map[string]map[string]any{
			actionID: {
				"id":          actionID,
				"description": "Stop EC2 instances",
				"tags":        map[string]any{"stackyard": "true"},
			},
		},
		targetResourceTypes: map[string]map[string]any{
			targetResourceTypeID: {
				"id":          targetResourceTypeID,
				"description": "EC2 instance resource target",
				"parameters": []any{
					map[string]any{"key": "resourceTags", "description": "Resource tag filter", "required": false},
				},
			},
		},
		experimentTemplates: map[string]map[string]any{
			templateID: {
				"id":          templateID,
				"arn":         templateArn,
				"description": "Seeded experiment template",
				"roleArn":     "arn:aws:iam::123456789012:role/stackyard-fis-role",
				"state":       map[string]any{"status": "active"},
				"createdAt":   now,
				"updatedAt":   now,
				"tags":        map[string]any{"stackyard": "true"},
			},
		},
		experiments: map[string]map[string]any{
			experimentID: {
				"id":                   experimentID,
				"arn":                  experimentArn,
				"experimentTemplateId": templateID,
				"state":                map[string]any{"status": "completed"},
				"createdAt":            now,
				"updatedAt":            now,
				"tags":                 map[string]any{"stackyard": "true"},
			},
		},
		targetAccountByTemplate: map[string]map[string]map[string]any{
			templateID: {
				accountID: {
					"accountId": accountID,
					"roleArn":   "arn:aws:iam::123456789012:role/stackyard-fis-role",
					"status":    "enabled",
				},
			},
		},
		targetAccountByExperiment: map[string]map[string]map[string]any{
			experimentID: {
				accountID: {
					"accountId": accountID,
					"roleArn":   "arn:aws:iam::123456789012:role/stackyard-fis-role",
					"status":    "enabled",
				},
			},
		},
		resolvedTargetsByExperiment: map[string][]map[string]any{
			experimentID: {
				{
					"targetName":      "instances",
					"targetType":      targetResourceTypeID,
					"resolvedTargets": []any{"i-00000000000000001"},
				},
			},
		},
		safetyLevers: map[string]map[string]any{
			"default": {
				"id":          "default",
				"state":       map[string]any{"status": "enabled", "reason": "Seeded default safety lever"},
				"description": "Seeded safety lever",
				"updatedAt":   now,
			},
		},
		tags: map[string]map[string]string{
			templateArn:   {"stackyard": "true"},
			experimentArn: {"stackyard": "true"},
		},
		createTemplateTokens:  map[string]string{},
		startExperimentTokens: map[string]string{},
	}
	return s
}

func (s *fisStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateExperimentTemplate":
		clientToken := fisLookupString(payload, pathParams, query, "clientToken", "ClientToken")
		if existingID := strings.TrimSpace(s.createTemplateTokens[clientToken]); existingID != "" {
			return map[string]any{"experimentTemplate": fisCloneMap(s.ensureTemplateLocked(existingID, now))}
		}

		templateID := fmt.Sprintf("ext-%06d", s.nextTemplateID)
		s.nextTemplateID++
		template := s.ensureTemplateLocked(templateID, now)
		if v := fisLookupString(payload, pathParams, query, "description", "Description"); v != "" {
			template["description"] = v
		}
		if v := fisLookupString(payload, pathParams, query, "roleArn", "RoleArn"); v != "" {
			template["roleArn"] = v
		}
		if clientToken != "" {
			s.createTemplateTokens[clientToken] = templateID
		}
		return map[string]any{"experimentTemplate": fisCloneMap(template)}

	case "UpdateExperimentTemplate":
		templateID := fisLookupString(payload, pathParams, query, "id", "experimentTemplateId")
		template := s.ensureTemplateLocked(templateID, now)
		if v := fisLookupString(payload, pathParams, query, "description", "Description"); v != "" {
			template["description"] = v
		}
		if v := fisLookupString(payload, pathParams, query, "roleArn", "RoleArn"); v != "" {
			template["roleArn"] = v
		}
		template["updatedAt"] = now
		return map[string]any{"experimentTemplate": fisCloneMap(template)}

	case "DeleteExperimentTemplate":
		templateID := fisLookupString(payload, pathParams, query, "id", "experimentTemplateId")
		template := s.ensureTemplateLocked(templateID, now)
		delete(s.experimentTemplates, templateID)
		delete(s.targetAccountByTemplate, templateID)
		delete(s.tags, fisString(template, "arn", fisTemplateARN(templateID)))
		return map[string]any{}

	case "GetExperimentTemplate":
		templateID := fisLookupString(payload, pathParams, query, "id", "experimentTemplateId")
		return map[string]any{"experimentTemplate": fisCloneMap(s.ensureTemplateLocked(templateID, now))}

	case "ListExperimentTemplates":
		items := make([]any, 0, len(s.experimentTemplates))
		for _, template := range fisSortedMaps(s.experimentTemplates) {
			items = append(items, map[string]any{
				"id":          fisString(template, "id", ""),
				"arn":         fisString(template, "arn", ""),
				"description": fisString(template, "description", ""),
			})
		}
		return map[string]any{"experimentTemplates": items, "nextToken": ""}

	case "StartExperiment":
		clientToken := fisLookupString(payload, pathParams, query, "clientToken", "ClientToken")
		if existingID := strings.TrimSpace(s.startExperimentTokens[clientToken]); existingID != "" {
			return map[string]any{"experiment": fisCloneMap(s.ensureExperimentLocked(existingID, now))}
		}

		templateID := fisLookupString(payload, pathParams, query, "experimentTemplateId")
		template := s.ensureTemplateLocked(templateID, now)
		experimentID := fmt.Sprintf("exp-%06d", s.nextExperiment)
		s.nextExperiment++
		experiment := s.ensureExperimentLocked(experimentID, now)
		experiment["experimentTemplateId"] = fisString(template, "id", templateID)
		experiment["state"] = map[string]any{"status": "running"}
		experiment["updatedAt"] = now
		s.targetAccountByExperiment[experimentID] = fisCloneNestedMap(s.ensureTargetAccountTemplateMapLocked(fisString(template, "id", templateID)))
		s.resolvedTargetsByExperiment[experimentID] = []map[string]any{
			{
				"targetName":      "instances",
				"targetType":      "aws:ec2:instance",
				"resolvedTargets": []any{"i-00000000000000001"},
			},
		}
		s.ensureTagsLocked(fisString(experiment, "arn", fisExperimentARN(experimentID)))["stackyard"] = "true"
		if clientToken != "" {
			s.startExperimentTokens[clientToken] = experimentID
		}
		return map[string]any{"experiment": fisCloneMap(experiment)}

	case "StopExperiment":
		experimentID := fisLookupString(payload, pathParams, query, "id", "experimentId")
		experiment := s.ensureExperimentLocked(experimentID, now)
		experiment["state"] = map[string]any{"status": "stopped"}
		experiment["updatedAt"] = now
		return map[string]any{"experiment": fisCloneMap(experiment)}

	case "GetExperiment":
		experimentID := fisLookupString(payload, pathParams, query, "id", "experimentId")
		return map[string]any{"experiment": fisCloneMap(s.ensureExperimentLocked(experimentID, now))}

	case "ListExperiments":
		items := make([]any, 0, len(s.experiments))
		for _, experiment := range fisSortedMaps(s.experiments) {
			items = append(items, map[string]any{
				"id":                   fisString(experiment, "id", ""),
				"arn":                  fisString(experiment, "arn", ""),
				"experimentTemplateId": fisString(experiment, "experimentTemplateId", ""),
				"state":                experiment["state"],
			})
		}
		return map[string]any{"experiments": items, "nextToken": ""}

	case "ListExperimentResolvedTargets":
		experimentID := fisLookupString(payload, pathParams, query, "id", "experimentId")
		_ = s.ensureExperimentLocked(experimentID, now)
		items := s.resolvedTargetsByExperiment[experimentID]
		if len(items) == 0 {
			items = []map[string]any{
				{
					"targetName":      "instances",
					"targetType":      "aws:ec2:instance",
					"resolvedTargets": []any{"i-00000000000000001"},
				},
			}
			s.resolvedTargetsByExperiment[experimentID] = items
		}
		return map[string]any{"resolvedTargets": fisCloneListOfMaps(items), "nextToken": ""}

	case "CreateTargetAccountConfiguration":
		templateID := fisLookupString(payload, pathParams, query, "experimentTemplateId", "id")
		template := s.ensureTemplateLocked(templateID, now)
		templateID = fisString(template, "id", templateID)
		accountID := fisLookupString(payload, pathParams, query, "accountId", "AccountId")
		if accountID == "" {
			accountID = fisDefaultAccountID
		}
		config := s.ensureTargetAccountConfigurationLocked(templateID, accountID, now)
		if v := fisLookupString(payload, pathParams, query, "roleArn", "RoleArn"); v != "" {
			config["roleArn"] = v
		}
		return map[string]any{"targetAccountConfiguration": fisCloneMap(config)}

	case "UpdateTargetAccountConfiguration":
		templateID := fisLookupString(payload, pathParams, query, "experimentTemplateId", "id")
		template := s.ensureTemplateLocked(templateID, now)
		templateID = fisString(template, "id", templateID)
		accountID := fisLookupString(payload, pathParams, query, "accountId", "AccountId")
		if accountID == "" {
			accountID = fisDefaultAccountID
		}
		config := s.ensureTargetAccountConfigurationLocked(templateID, accountID, now)
		if v := fisLookupString(payload, pathParams, query, "status", "Status"); v != "" {
			config["status"] = strings.ToLower(v)
		}
		if v := fisLookupString(payload, pathParams, query, "roleArn", "RoleArn"); v != "" {
			config["roleArn"] = v
		}
		return map[string]any{"targetAccountConfiguration": fisCloneMap(config)}

	case "DeleteTargetAccountConfiguration":
		templateID := fisLookupString(payload, pathParams, query, "experimentTemplateId", "id")
		template := s.ensureTemplateLocked(templateID, now)
		templateID = fisString(template, "id", templateID)
		accountID := fisLookupString(payload, pathParams, query, "accountId", "AccountId")
		if accountID == "" {
			accountID = fisDefaultAccountID
		}
		delete(s.ensureTargetAccountTemplateMapLocked(templateID), accountID)
		return map[string]any{}

	case "GetTargetAccountConfiguration":
		templateID := fisLookupString(payload, pathParams, query, "experimentTemplateId", "id")
		template := s.ensureTemplateLocked(templateID, now)
		templateID = fisString(template, "id", templateID)
		accountID := fisLookupString(payload, pathParams, query, "accountId", "AccountId")
		if accountID == "" {
			accountID = fisDefaultAccountID
		}
		return map[string]any{
			"targetAccountConfiguration": fisCloneMap(s.ensureTargetAccountConfigurationLocked(templateID, accountID, now)),
		}

	case "ListTargetAccountConfigurations":
		templateID := fisLookupString(payload, pathParams, query, "experimentTemplateId", "id")
		template := s.ensureTemplateLocked(templateID, now)
		configs := s.ensureTargetAccountTemplateMapLocked(fisString(template, "id", templateID))
		items := make([]any, 0, len(configs))
		for _, cfg := range fisSortedMaps(configs) {
			items = append(items, fisCloneMap(cfg))
		}
		return map[string]any{"targetAccountConfigurations": items, "nextToken": ""}

	case "GetExperimentTargetAccountConfiguration":
		experimentID := fisLookupString(payload, pathParams, query, "experimentId", "id")
		experiment := s.ensureExperimentLocked(experimentID, now)
		experimentID = fisString(experiment, "id", experimentID)
		accountID := fisLookupString(payload, pathParams, query, "accountId", "AccountId")
		if accountID == "" {
			accountID = fisDefaultAccountID
		}
		return map[string]any{
			"experimentTargetAccountConfiguration": fisCloneMap(s.ensureExperimentTargetAccountConfigurationLocked(experimentID, accountID, now)),
		}

	case "ListExperimentTargetAccountConfigurations":
		experimentID := fisLookupString(payload, pathParams, query, "experimentId", "id")
		experiment := s.ensureExperimentLocked(experimentID, now)
		configs := s.ensureTargetAccountExperimentMapLocked(fisString(experiment, "id", experimentID))
		items := make([]any, 0, len(configs))
		for _, cfg := range fisSortedMaps(configs) {
			items = append(items, fisCloneMap(cfg))
		}
		return map[string]any{"experimentTargetAccountConfigurations": items, "nextToken": ""}

	case "GetAction":
		actionID := fisLookupString(payload, pathParams, query, "id", "actionId")
		return map[string]any{"action": fisCloneMap(s.ensureActionLocked(actionID))}

	case "ListActions":
		items := make([]any, 0, len(s.actions))
		for _, actionItem := range fisSortedMaps(s.actions) {
			items = append(items, fisCloneMap(actionItem))
		}
		return map[string]any{"actions": items, "nextToken": ""}

	case "GetTargetResourceType":
		targetTypeID := fisLookupString(payload, pathParams, query, "id", "targetResourceTypeId")
		return map[string]any{"targetResourceType": fisCloneMap(s.ensureTargetResourceTypeLocked(targetTypeID))}

	case "ListTargetResourceTypes":
		items := make([]any, 0, len(s.targetResourceTypes))
		for _, targetType := range fisSortedMaps(s.targetResourceTypes) {
			items = append(items, fisCloneMap(targetType))
		}
		return map[string]any{"targetResourceTypes": items, "nextToken": ""}

	case "GetSafetyLever":
		safetyLeverID := fisLookupString(payload, pathParams, query, "id", "safetyLeverId")
		return map[string]any{"safetyLever": fisCloneMap(s.ensureSafetyLeverLocked(safetyLeverID, now))}

	case "UpdateSafetyLeverState":
		safetyLeverID := fisLookupString(payload, pathParams, query, "id", "safetyLeverId")
		lever := s.ensureSafetyLeverLocked(safetyLeverID, now)
		state := fisLookupString(payload, pathParams, query, "state", "status")
		if state == "" {
			state = "enabled"
		}
		reason := fisLookupString(payload, pathParams, query, "reason")
		if reason == "" {
			reason = "updated by UpdateSafetyLeverState"
		}
		lever["state"] = map[string]any{"status": strings.ToLower(state), "reason": reason}
		lever["updatedAt"] = now
		return map[string]any{"safetyLever": fisCloneMap(lever)}

	case "TagResource":
		resourceArn := fisLookupString(payload, pathParams, query, "resourceArn", "ResourceArn")
		if resourceArn == "" {
			resourceArn = fisTemplateARN("ext-000001")
		}
		tags := s.ensureTagsLocked(resourceArn)
		for key, value := range fisExtractTags(payload) {
			tags[key] = value
		}
		if len(tags) == 0 {
			tags["stackyard"] = "true"
		}
		return map[string]any{}

	case "UntagResource":
		resourceArn := fisLookupString(payload, pathParams, query, "resourceArn", "ResourceArn")
		if resourceArn == "" {
			resourceArn = fisTemplateARN("ext-000001")
		}
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range fisExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceArn := fisLookupString(payload, pathParams, query, "resourceArn", "ResourceArn")
		if resourceArn == "" {
			resourceArn = fisTemplateARN("ext-000001")
		}
		return map[string]any{"tags": fisCloneTags(s.ensureTagsLocked(resourceArn))}
	}

	return map[string]any{}
}

func (s *fisStore) ensureActionLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "aws:ec2:stop-instances"
	}
	if existing, ok := s.actions[id]; ok {
		return existing
	}
	action := map[string]any{
		"id":          id,
		"description": "Auto-created FIS action",
	}
	s.actions[id] = action
	return action
}

func (s *fisStore) ensureTargetResourceTypeLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "aws:ec2:instance"
	}
	if existing, ok := s.targetResourceTypes[id]; ok {
		return existing
	}
	resourceType := map[string]any{
		"id":          id,
		"description": "Auto-created target resource type",
		"parameters":  []any{},
	}
	s.targetResourceTypes[id] = resourceType
	return resourceType
}

func (s *fisStore) ensureTemplateLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ext-000001"
	}
	if existing, ok := s.experimentTemplates[id]; ok {
		return existing
	}
	template := map[string]any{
		"id":          id,
		"arn":         fisTemplateARN(id),
		"description": "Auto-created experiment template",
		"roleArn":     "arn:aws:iam::123456789012:role/stackyard-fis-role",
		"state":       map[string]any{"status": "active"},
		"createdAt":   now,
		"updatedAt":   now,
		"tags":        map[string]any{"stackyard": "true"},
	}
	s.experimentTemplates[id] = template
	s.ensureTargetAccountConfigurationLocked(id, fisDefaultAccountID, now)
	s.ensureTagsLocked(fisTemplateARN(id))["stackyard"] = "true"
	return template
}

func (s *fisStore) ensureExperimentLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "exp-000001"
	}
	if existing, ok := s.experiments[id]; ok {
		return existing
	}
	template := s.ensureTemplateLocked("ext-000001", now)
	experiment := map[string]any{
		"id":                   id,
		"arn":                  fisExperimentARN(id),
		"experimentTemplateId": fisString(template, "id", "ext-000001"),
		"state":                map[string]any{"status": "running"},
		"createdAt":            now,
		"updatedAt":            now,
		"tags":                 map[string]any{"stackyard": "true"},
	}
	s.experiments[id] = experiment
	s.targetAccountByExperiment[id] = fisCloneNestedMap(s.ensureTargetAccountTemplateMapLocked(fisString(template, "id", "ext-000001")))
	s.resolvedTargetsByExperiment[id] = []map[string]any{
		{
			"targetName":      "instances",
			"targetType":      "aws:ec2:instance",
			"resolvedTargets": []any{"i-00000000000000001"},
		},
	}
	s.ensureTagsLocked(fisExperimentARN(id))["stackyard"] = "true"
	return experiment
}

func (s *fisStore) ensureTargetAccountTemplateMapLocked(templateID string) map[string]map[string]any {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		templateID = "ext-000001"
	}
	existing := s.targetAccountByTemplate[templateID]
	if existing == nil {
		existing = map[string]map[string]any{}
		s.targetAccountByTemplate[templateID] = existing
	}
	return existing
}

func (s *fisStore) ensureTargetAccountExperimentMapLocked(experimentID string) map[string]map[string]any {
	experimentID = strings.TrimSpace(experimentID)
	if experimentID == "" {
		experimentID = "exp-000001"
	}
	existing := s.targetAccountByExperiment[experimentID]
	if existing == nil {
		existing = map[string]map[string]any{}
		s.targetAccountByExperiment[experimentID] = existing
	}
	return existing
}

func (s *fisStore) ensureTargetAccountConfigurationLocked(templateID, accountID, now string) map[string]any {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		templateID = "ext-000001"
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		accountID = fisDefaultAccountID
	}
	configs := s.ensureTargetAccountTemplateMapLocked(templateID)
	if existing := configs[accountID]; existing != nil {
		return existing
	}
	config := map[string]any{
		"accountId": accountID,
		"roleArn":   "arn:aws:iam::123456789012:role/stackyard-fis-role",
		"status":    "enabled",
		"createdAt": now,
		"updatedAt": now,
	}
	configs[accountID] = config
	return config
}

func (s *fisStore) ensureExperimentTargetAccountConfigurationLocked(experimentID, accountID, now string) map[string]any {
	experimentID = strings.TrimSpace(experimentID)
	if experimentID == "" {
		experimentID = "exp-000001"
	}
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		accountID = fisDefaultAccountID
	}
	configs := s.ensureTargetAccountExperimentMapLocked(experimentID)
	if existing := configs[accountID]; existing != nil {
		return existing
	}
	templateID := fisString(s.ensureExperimentLocked(experimentID, now), "experimentTemplateId", "ext-000001")
	base := s.ensureTargetAccountConfigurationLocked(templateID, accountID, now)
	config := fisCloneMap(base)
	config["updatedAt"] = now
	configs[accountID] = config
	return config
}

func (s *fisStore) ensureSafetyLeverLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "default"
	}
	if existing := s.safetyLevers[id]; existing != nil {
		return existing
	}
	lever := map[string]any{
		"id":          id,
		"state":       map[string]any{"status": "enabled", "reason": "Auto-created safety lever"},
		"description": "Auto-created safety lever",
		"updatedAt":   now,
	}
	s.safetyLevers[id] = lever
	return lever
}

func (s *fisStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = fisTemplateARN("ext-000001")
	}
	existing := s.tags[resourceArn]
	if existing == nil {
		existing = map[string]string{}
		s.tags[resourceArn] = existing
	}
	return existing
}

func fisTemplateARN(templateID string) string {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		templateID = "ext-000001"
	}
	return fmt.Sprintf("arn:aws:fis:%s:%s:experiment-template/%s", fisDefaultRegion, fisDefaultAccountID, templateID)
}

func fisExperimentARN(experimentID string) string {
	experimentID = strings.TrimSpace(experimentID)
	if experimentID == "" {
		experimentID = "exp-000001"
	}
	return fmt.Sprintf("arn:aws:fis:%s:%s:experiment/%s", fisDefaultRegion, fisDefaultAccountID, experimentID)
}

func fisLookupString(payload map[string]any, pathParams map[string]string, query url.Values, keys ...string) string {
	for _, key := range keys {
		if pathParams != nil {
			if v := strings.TrimSpace(pathParams[key]); v != "" {
				return v
			}
			if v := strings.TrimSpace(pathParams[lowerFirst(key)]); v != "" {
				return v
			}
		}
		if payload != nil {
			if raw, ok := payload[key]; ok {
				if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
			if raw, ok := payload[lowerFirst(key)]; ok {
				if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		if query != nil {
			if v := strings.TrimSpace(query.Get(key)); v != "" {
				return v
			}
			if v := strings.TrimSpace(query.Get(lowerFirst(key))); v != "" {
				return v
			}
		}
	}
	return ""
}

func fisExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}

	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		for k, v := range m {
			ks := strings.TrimSpace(k)
			vs, _ := v.(string)
			vs = strings.TrimSpace(vs)
			if ks != "" {
				out[ks] = vs
			}
		}
	}

	if len(out) == 0 {
		out["stackyard"] = "true"
	}
	return out
}

func fisExtractTagKeys(payload map[string]any, query url.Values) []string {
	var keys []string
	appendKey := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		keys = append(keys, v)
	}

	if payload != nil {
		for _, field := range []string{"tagKeys", "TagKeys"} {
			if raw, ok := payload[field]; ok {
				switch x := raw.(type) {
				case []any:
					for _, item := range x {
						if s, ok := item.(string); ok {
							appendKey(s)
						}
					}
				case []string:
					for _, s := range x {
						appendKey(s)
					}
				case string:
					for _, part := range strings.Split(x, ",") {
						appendKey(part)
					}
				}
			}
		}
	}

	if query != nil {
		for _, candidate := range []string{"tagKeys", "TagKeys"} {
			for _, value := range query[candidate] {
				for _, part := range strings.Split(value, ",") {
					appendKey(part)
				}
			}
		}
	}

	uniq := map[string]struct{}{}
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := uniq[key]; ok {
			continue
		}
		uniq[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func fisCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func fisCloneListOfMaps(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, fisCloneMap(item))
	}
	return out
}

func fisCloneNestedMap(in map[string]map[string]any) map[string]map[string]any {
	out := make(map[string]map[string]any, len(in))
	for k, v := range in {
		out[k] = fisCloneMap(v)
	}
	return out
}

func fisCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func fisString(m map[string]any, key, fallback string) string {
	if m == nil {
		return fallback
	}
	raw, ok := m[key]
	if !ok {
		return fallback
	}
	s, ok := raw.(string)
	if !ok {
		return fallback
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func fisSortedMaps[T ~map[string]map[string]any](source T) []map[string]any {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, source[key])
	}
	return out
}
