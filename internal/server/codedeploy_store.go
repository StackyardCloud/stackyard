package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type codeDeployStore struct {
	mu                    sync.Mutex
	nextID                int64
	applications          map[string]*codeDeployApplication
	deploymentConfigs     map[string]*codeDeployDeploymentConfig
	deploymentGroups      map[string]*codeDeployDeploymentGroup
	deployments           map[string]*codeDeployDeployment
	onPremInstances       map[string]*codeDeployOnPremInstance
	revisions             map[string]map[string]map[string]any
	resourceTags          map[string]map[string]string
	githubTokens          map[string]bool
	externalResourceByID  map[string][]string
	lifecycleHookStatuses map[string]string
}

type codeDeployApplication struct {
	ID              string
	Name            string
	ComputePlatform string
	CreateTime      time.Time
}

type codeDeployDeploymentConfig struct {
	ID                string
	Name              string
	CreateTime        time.Time
	MinimumHealthy    map[string]any
	TrafficRoutingCfg map[string]any
}

type codeDeployDeploymentGroup struct {
	ID                   string
	ApplicationName      string
	Name                 string
	DeploymentConfigName string
	ServiceRoleArn       string
	CreateTime           time.Time
	LastAttemptedID      string
	LastSuccessfulID     string
	AutoScalingGroups    []string
	EC2TagFilters        []map[string]any
	OnPremisesTagFilters []map[string]any
	DeploymentStyle      map[string]any
}

type codeDeployDeployment struct {
	ID                  string
	ApplicationName     string
	DeploymentGroupName string
	Description         string
	Status              string
	CreateTime          time.Time
	CompleteTime        *time.Time
	Revision            map[string]any
	ErrorInformation    map[string]any
	TargetIDs           []string
}

type codeDeployOnPremInstance struct {
	Name         string
	IamUserArn   string
	RegisterTime time.Time
	Deregistered bool
	Tags         map[string]string
}

func newCodeDeployStore() *codeDeployStore {
	now := time.Now().UTC()
	app := &codeDeployApplication{
		ID:              "app-000000000001",
		Name:            "stackyard-app",
		ComputePlatform: "Server",
		CreateTime:      now,
	}
	cfg := &codeDeployDeploymentConfig{
		ID:         "dc-000000000001",
		Name:       "CodeDeployDefault.AllAtOnce",
		CreateTime: now,
		MinimumHealthy: map[string]any{
			"type":  "HOST_COUNT",
			"value": 1,
		},
		TrafficRoutingCfg: map[string]any{},
	}
	group := &codeDeployDeploymentGroup{
		ID:                   "dg-000000000001",
		ApplicationName:      app.Name,
		Name:                 "stackyard-group",
		DeploymentConfigName: cfg.Name,
		ServiceRoleArn:       "arn:aws:iam::123456789012:role/stackyard-codedeploy",
		CreateTime:           now,
		AutoScalingGroups:    []string{"stackyard-asg"},
		EC2TagFilters:        []map[string]any{},
		OnPremisesTagFilters: []map[string]any{},
		DeploymentStyle: map[string]any{
			"deploymentType":   "IN_PLACE",
			"deploymentOption": "WITHOUT_TRAFFIC_CONTROL",
		},
	}
	deployment := &codeDeployDeployment{
		ID:                  "d-000000000001",
		ApplicationName:     app.Name,
		DeploymentGroupName: group.Name,
		Description:         "seed deployment",
		Status:              "Succeeded",
		CreateTime:          now,
		CompleteTime:        ptrTime(now),
		Revision:            codeDeployDefaultRevision(),
		ErrorInformation:    map[string]any{},
		TargetIDs:           []string{"i-000000000001"},
	}
	group.LastAttemptedID = deployment.ID
	group.LastSuccessfulID = deployment.ID
	onPrem := &codeDeployOnPremInstance{
		Name:         "stackyard-onprem-1",
		IamUserArn:   "arn:aws:iam::123456789012:user/stackyard-codedeploy",
		RegisterTime: now,
		Tags:         map[string]string{"env": "test"},
	}
	revisionKey := codeDeployRevisionKey(deployment.Revision)

	store := &codeDeployStore{
		nextID: 2,
		applications: map[string]*codeDeployApplication{
			app.Name: app,
		},
		deploymentConfigs: map[string]*codeDeployDeploymentConfig{
			cfg.Name: cfg,
		},
		deploymentGroups: map[string]*codeDeployDeploymentGroup{
			codeDeployGroupKey(group.ApplicationName, group.Name): group,
		},
		deployments: map[string]*codeDeployDeployment{
			deployment.ID: deployment,
		},
		onPremInstances: map[string]*codeDeployOnPremInstance{
			onPrem.Name: onPrem,
		},
		revisions: map[string]map[string]map[string]any{
			app.Name: {
				revisionKey: codeDeployCloneMapAny(deployment.Revision),
			},
		},
		resourceTags: map[string]map[string]string{
			codeDeployApplicationARN(app.Name): {"service": "codedeploy", "seed": "true"},
		},
		githubTokens: map[string]bool{
			"stackyard-github-token": true,
		},
		externalResourceByID: map[string][]string{
			"external-seed": {codeDeployDeploymentARN(deployment.ID)},
		},
		lifecycleHookStatuses: map[string]string{},
	}
	return store
}

func (s *codeDeployStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "AddTagsToOnPremisesInstances":
		instanceNames := codeDeployStringSlice(payload["instanceNames"])
		if len(instanceNames) == 0 {
			instanceNames = []string{"stackyard-onprem-1"}
		}
		tags := codeDeployExtractTags(payload["tags"])
		if len(tags) == 0 {
			tags = map[string]string{"coverage": "true"}
		}
		for _, name := range instanceNames {
			inst := s.ensureOnPremInstanceLocked(name)
			for k, v := range tags {
				inst.Tags[k] = v
			}
		}
		return map[string]any{}

	case "BatchGetApplicationRevisions":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		s.ensureApplicationLocked(appName)
		revs := s.revisions[appName]
		if len(revs) == 0 {
			seed := codeDeployDefaultRevision()
			s.ensureRevisionMapLocked(appName)[codeDeployRevisionKey(seed)] = codeDeployCloneMapAny(seed)
			revs = s.revisions[appName]
		}
		out := make([]map[string]any, 0, len(revs))
		for _, rev := range revs {
			out = append(out, map[string]any{
				"applicationName": appName,
				"revision":        codeDeployCloneMapAny(rev),
				"revisionInfo": map[string]any{
					"description":      "Stackyard revision",
					"firstUsedTime":    time.Now().UTC(),
					"lastUsedTime":     time.Now().UTC(),
					"registerTime":     time.Now().UTC(),
					"deploymentGroups": []any{"stackyard-group"},
				},
			})
		}
		return map[string]any{"applicationName": appName, "revisions": out}

	case "BatchGetApplications":
		names := codeDeployStringSlice(payload["applicationNames"])
		if len(names) == 0 {
			names = s.listApplicationNamesLocked()
		}
		apps := make([]map[string]any, 0, len(names))
		for _, name := range names {
			apps = append(apps, s.applicationPayloadLocked(name))
		}
		return map[string]any{"applicationsInfo": apps}

	case "BatchGetDeploymentGroups":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		names := codeDeployStringSlice(payload["deploymentGroupNames"])
		if len(names) == 0 {
			names = s.listDeploymentGroupNamesLocked(appName)
		}
		groups := make([]map[string]any, 0, len(names))
		for _, name := range names {
			groups = append(groups, s.deploymentGroupPayloadLocked(appName, name))
		}
		return map[string]any{"deploymentGroupsInfo": groups}

	case "BatchGetDeploymentInstances":
		deploymentID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(deploymentID)
		instanceIDs := codeDeployStringSlice(payload["instanceIds"])
		if len(instanceIDs) == 0 {
			instanceIDs = append([]string{}, dep.TargetIDs...)
		}
		summaries := make([]map[string]any, 0, len(instanceIDs))
		for _, id := range instanceIDs {
			summaries = append(summaries, codeDeployInstanceSummaryPayload(id, dep.Status))
		}
		return map[string]any{"instancesSummary": summaries}

	case "BatchGetDeployments":
		ids := codeDeployStringSlice(payload["deploymentIds"])
		if len(ids) == 0 {
			ids = s.listDeploymentIDsLocked("", "")
		}
		infos := make([]map[string]any, 0, len(ids))
		for _, id := range ids {
			infos = append(infos, s.deploymentPayloadLocked(id))
		}
		return map[string]any{"deploymentsInfo": infos}

	case "BatchGetDeploymentTargets":
		deploymentID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(deploymentID)
		targetIDs := codeDeployStringSlice(payload["targetIds"])
		if len(targetIDs) == 0 {
			targetIDs = append([]string{}, dep.TargetIDs...)
		}
		targets := make([]map[string]any, 0, len(targetIDs))
		for _, targetID := range targetIDs {
			targets = append(targets, s.deploymentTargetPayloadLocked(dep, targetID))
		}
		return map[string]any{"deploymentTargets": targets}

	case "BatchGetOnPremisesInstances":
		names := codeDeployStringSlice(payload["instanceNames"])
		if len(names) == 0 {
			names = s.listOnPremInstanceNamesLocked()
		}
		infos := make([]map[string]any, 0, len(names))
		for _, name := range names {
			infos = append(infos, s.onPremInstancePayloadLocked(name))
		}
		return map[string]any{"instanceInfos": infos}

	case "ContinueDeployment":
		id := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(id)
		dep.Status = "InProgress"
		dep.CompleteTime = nil
		return map[string]any{}

	case "CreateApplication":
		name := codeDeployDefaultString(payload, "applicationName", fmt.Sprintf("stackyard-app-%06d", s.nextLocked()))
		app := s.ensureApplicationLocked(name)
		platform := codeDeployDefaultString(payload, "computePlatform", "Server")
		app.ComputePlatform = platform
		return map[string]any{"applicationId": app.ID}

	case "CreateDeployment":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		groupName := codeDeployDefaultString(payload, "deploymentGroupName", "stackyard-group")
		description := codeDeployDefaultString(payload, "description", "deployment created by stackyard")
		revision, _ := payload["revision"].(map[string]any)
		if len(revision) == 0 {
			revision = codeDeployDefaultRevision()
		}
		s.ensureApplicationLocked(appName)
		s.ensureDeploymentGroupLocked(appName, groupName)
		depID := fmt.Sprintf("d-%012d", s.nextLocked())
		now := time.Now().UTC()
		targetID := fmt.Sprintf("i-%012d", s.nextLocked())
		dep := &codeDeployDeployment{
			ID:                  depID,
			ApplicationName:     appName,
			DeploymentGroupName: groupName,
			Description:         description,
			Status:              "Created",
			CreateTime:          now,
			Revision:            codeDeployCloneMapAny(revision),
			ErrorInformation:    map[string]any{},
			TargetIDs:           []string{targetID},
		}
		s.deployments[dep.ID] = dep
		grp := s.ensureDeploymentGroupLocked(appName, groupName)
		grp.LastAttemptedID = dep.ID
		s.ensureRevisionMapLocked(appName)[codeDeployRevisionKey(dep.Revision)] = codeDeployCloneMapAny(dep.Revision)
		return map[string]any{"deploymentId": dep.ID}

	case "CreateDeploymentConfig":
		name := codeDeployDefaultString(payload, "deploymentConfigName", fmt.Sprintf("stackyard-config-%06d", s.nextLocked()))
		cfg := s.ensureDeploymentConfigLocked(name)
		if min, ok := payload["minimumHealthyHosts"].(map[string]any); ok && len(min) > 0 {
			cfg.MinimumHealthy = codeDeployCloneMapAny(min)
		}
		return map[string]any{"deploymentConfigId": cfg.ID}

	case "CreateDeploymentGroup":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		name := codeDeployDefaultString(payload, "deploymentGroupName", fmt.Sprintf("stackyard-group-%06d", s.nextLocked()))
		grp := s.ensureDeploymentGroupLocked(appName, name)
		if v := codeDeployString(payload["deploymentConfigName"]); v != "" {
			grp.DeploymentConfigName = v
			s.ensureDeploymentConfigLocked(v)
		}
		if v := codeDeployString(payload["serviceRoleArn"]); v != "" {
			grp.ServiceRoleArn = v
		}
		return map[string]any{"deploymentGroupId": grp.ID}

	case "DeleteApplication":
		name := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		delete(s.applications, name)
		delete(s.revisions, name)
		for key, grp := range s.deploymentGroups {
			if grp.ApplicationName == name {
				delete(s.deploymentGroups, key)
			}
		}
		for id, dep := range s.deployments {
			if dep.ApplicationName == name {
				delete(s.deployments, id)
			}
		}
		return map[string]any{}

	case "DeleteDeploymentConfig":
		name := codeDeployDefaultString(payload, "deploymentConfigName", "CodeDeployDefault.AllAtOnce")
		delete(s.deploymentConfigs, name)
		return map[string]any{}

	case "DeleteDeploymentGroup":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		name := codeDeployDefaultString(payload, "deploymentGroupName", "stackyard-group")
		delete(s.deploymentGroups, codeDeployGroupKey(appName, name))
		for id, dep := range s.deployments {
			if dep.ApplicationName == appName && dep.DeploymentGroupName == name {
				delete(s.deployments, id)
			}
		}
		return map[string]any{}

	case "DeleteGitHubAccountToken":
		tokenName := codeDeployDefaultString(payload, "tokenName", "stackyard-github-token")
		delete(s.githubTokens, tokenName)
		return map[string]any{}

	case "DeleteResourcesByExternalId":
		externalID := codeDeployDefaultString(payload, "externalId", "external-seed")
		delete(s.externalResourceByID, externalID)
		return map[string]any{}

	case "DeregisterOnPremisesInstance":
		name := codeDeployDefaultString(payload, "instanceName", "stackyard-onprem-1")
		if inst := s.ensureOnPremInstanceLocked(name); inst != nil {
			inst.Deregistered = true
		}
		return map[string]any{}

	case "GetApplication":
		name := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		return map[string]any{"application": s.applicationPayloadLocked(name)}

	case "GetApplicationRevision":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		s.ensureApplicationLocked(appName)
		revision, _ := payload["revision"].(map[string]any)
		if len(revision) == 0 {
			revision = codeDeployDefaultRevision()
		}
		revKey := codeDeployRevisionKey(revision)
		revMap := s.ensureRevisionMapLocked(appName)
		if existing, ok := revMap[revKey]; ok {
			revision = existing
		} else {
			revMap[revKey] = codeDeployCloneMapAny(revision)
		}
		return map[string]any{
			"applicationName": appName,
			"revision":        codeDeployCloneMapAny(revision),
			"revisionInfo": map[string]any{
				"description":      "Stackyard revision",
				"firstUsedTime":    time.Now().UTC(),
				"lastUsedTime":     time.Now().UTC(),
				"registerTime":     time.Now().UTC(),
				"deploymentGroups": s.listDeploymentGroupNamesLocked(appName),
			},
		}

	case "GetDeployment":
		id := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		return map[string]any{"deploymentInfo": s.deploymentPayloadLocked(id)}

	case "GetDeploymentConfig":
		name := codeDeployDefaultString(payload, "deploymentConfigName", "CodeDeployDefault.AllAtOnce")
		return map[string]any{"deploymentConfigInfo": s.deploymentConfigPayloadLocked(name)}

	case "GetDeploymentGroup":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		name := codeDeployDefaultString(payload, "deploymentGroupName", "stackyard-group")
		return map[string]any{"deploymentGroupInfo": s.deploymentGroupPayloadLocked(appName, name)}

	case "GetDeploymentInstance":
		depID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		instanceID := codeDeployDefaultString(payload, "instanceId", "i-000000000001")
		dep := s.ensureDeploymentLocked(depID)
		return map[string]any{"instanceSummary": codeDeployInstanceSummaryPayload(instanceID, dep.Status)}

	case "GetDeploymentTarget":
		depID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		targetID := codeDeployDefaultString(payload, "targetId", "i-000000000001")
		dep := s.ensureDeploymentLocked(depID)
		return map[string]any{"deploymentTarget": s.deploymentTargetPayloadLocked(dep, targetID)}

	case "GetOnPremisesInstance":
		name := codeDeployDefaultString(payload, "instanceName", "stackyard-onprem-1")
		return map[string]any{"instanceInfo": s.onPremInstancePayloadLocked(name)}

	case "ListApplicationRevisions":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		revMap := s.ensureRevisionMapLocked(appName)
		if len(revMap) == 0 {
			rev := codeDeployDefaultRevision()
			revMap[codeDeployRevisionKey(rev)] = codeDeployCloneMapAny(rev)
		}
		revisions := make([]map[string]any, 0, len(revMap))
		for _, rev := range revMap {
			revisions = append(revisions, codeDeployCloneMapAny(rev))
		}
		sort.Slice(revisions, func(i, j int) bool {
			return codeDeployRevisionKey(revisions[i]) < codeDeployRevisionKey(revisions[j])
		})
		return map[string]any{"revisions": revisions}

	case "ListApplications":
		return map[string]any{"applications": s.listApplicationNamesLocked()}

	case "ListDeploymentConfigs":
		names := make([]string, 0, len(s.deploymentConfigs))
		for name := range s.deploymentConfigs {
			names = append(names, name)
		}
		sort.Strings(names)
		return map[string]any{"deploymentConfigsList": names}

	case "ListDeploymentGroups":
		appName := codeDeployString(payload["applicationName"])
		if appName == "" {
			appName = "stackyard-app"
		}
		return map[string]any{"applicationName": appName, "deploymentGroups": s.listDeploymentGroupNamesLocked(appName)}

	case "ListDeploymentInstances":
		depID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(depID)
		ids := append([]string{}, dep.TargetIDs...)
		sort.Strings(ids)
		if len(ids) == 0 {
			ids = []string{"i-000000000001"}
		}
		return map[string]any{"instancesList": ids}

	case "ListDeployments":
		appName := codeDeployString(payload["applicationName"])
		groupName := codeDeployString(payload["deploymentGroupName"])
		ids := s.listDeploymentIDsLocked(appName, groupName)
		return map[string]any{"deployments": ids}

	case "ListDeploymentTargets":
		depID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(depID)
		targetIDs := append([]string{}, dep.TargetIDs...)
		sort.Strings(targetIDs)
		if len(targetIDs) == 0 {
			targetIDs = []string{"i-000000000001"}
		}
		return map[string]any{"targetIds": targetIDs}

	case "ListGitHubAccountTokenNames":
		names := make([]string, 0, len(s.githubTokens))
		for token := range s.githubTokens {
			names = append(names, token)
		}
		sort.Strings(names)
		if len(names) == 0 {
			names = []string{"stackyard-github-token"}
		}
		return map[string]any{"tokenNameList": names}

	case "ListOnPremisesInstances":
		return map[string]any{"instanceNames": s.listOnPremInstanceNamesLocked()}

	case "ListTagsForResource":
		arn := codeDeployDefaultString(payload, "resourceArn", codeDeployApplicationARN("stackyard-app"))
		tags := codeDeploySortedTagsPayload(s.resourceTags[arn])
		return map[string]any{"Tags": tags}

	case "PutLifecycleEventHookExecutionStatus":
		deploymentID := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		hookID := codeDeployDefaultString(payload, "lifecycleEventHookExecutionId", "hook-000000000001")
		status := codeDeployDefaultString(payload, "status", "Succeeded")
		key := deploymentID + ":" + hookID
		s.lifecycleHookStatuses[key] = status
		return map[string]any{}

	case "RegisterApplicationRevision":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		revision, _ := payload["revision"].(map[string]any)
		if len(revision) == 0 {
			revision = codeDeployDefaultRevision()
		}
		s.ensureRevisionMapLocked(appName)[codeDeployRevisionKey(revision)] = codeDeployCloneMapAny(revision)
		return map[string]any{}

	case "RegisterOnPremisesInstance":
		name := codeDeployDefaultString(payload, "instanceName", "stackyard-onprem-1")
		inst := s.ensureOnPremInstanceLocked(name)
		if arn := codeDeployString(payload["iamSessionArn"]); arn != "" {
			inst.IamUserArn = arn
		}
		inst.Deregistered = false
		return map[string]any{}

	case "RemoveTagsFromOnPremisesInstances":
		instanceNames := codeDeployStringSlice(payload["instanceNames"])
		if len(instanceNames) == 0 {
			instanceNames = []string{"stackyard-onprem-1"}
		}
		tags := codeDeployExtractTags(payload["tags"])
		if len(tags) == 0 {
			tags = map[string]string{"coverage": "true"}
		}
		for _, name := range instanceNames {
			inst := s.ensureOnPremInstanceLocked(name)
			for k := range tags {
				delete(inst.Tags, k)
			}
		}
		return map[string]any{}

	case "SkipWaitTimeForInstanceTermination":
		id := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(id)
		dep.Status = "Succeeded"
		now := time.Now().UTC()
		dep.CompleteTime = &now
		return map[string]any{}

	case "StopDeployment":
		id := codeDeployDefaultString(payload, "deploymentId", "d-000000000001")
		dep := s.ensureDeploymentLocked(id)
		dep.Status = "Stopped"
		now := time.Now().UTC()
		dep.CompleteTime = &now
		return map[string]any{
			"status":        "Succeeded",
			"statusMessage": "Deployment stop requested",
		}

	case "TagResource":
		arn := codeDeployDefaultString(payload, "resourceArn", codeDeployApplicationARN("stackyard-app"))
		tags := codeDeployExtractTags(payload["tags"])
		if len(tags) == 0 {
			tags = map[string]string{"coverage": "true"}
		}
		existing := s.ensureResourceTagMapLocked(arn)
		for k, v := range tags {
			existing[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		arn := codeDeployDefaultString(payload, "resourceArn", codeDeployApplicationARN("stackyard-app"))
		tagKeys := codeDeployStringSlice(payload["tagKeys"])
		if len(tagKeys) == 0 {
			tagKeys = []string{"coverage"}
		}
		existing := s.ensureResourceTagMapLocked(arn)
		for _, key := range tagKeys {
			delete(existing, key)
		}
		return map[string]any{}

	case "UpdateApplication":
		current := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		newName := codeDeployString(payload["newApplicationName"])
		if newName == "" {
			newName = current
		}
		app := s.ensureApplicationLocked(current)
		if current != newName {
			delete(s.applications, current)
			app.Name = newName
			s.applications[newName] = app
			if revisions, ok := s.revisions[current]; ok {
				delete(s.revisions, current)
				s.revisions[newName] = revisions
			}
			for key, grp := range s.deploymentGroups {
				if grp.ApplicationName == current {
					delete(s.deploymentGroups, key)
					grp.ApplicationName = newName
					s.deploymentGroups[codeDeployGroupKey(newName, grp.Name)] = grp
				}
			}
			for _, dep := range s.deployments {
				if dep.ApplicationName == current {
					dep.ApplicationName = newName
				}
			}
		}
		if platform := codeDeployString(payload["computePlatform"]); platform != "" {
			app.ComputePlatform = platform
		}
		return map[string]any{}

	case "UpdateDeploymentGroup":
		appName := codeDeployDefaultString(payload, "applicationName", "stackyard-app")
		groupName := codeDeployDefaultString(payload, "currentDeploymentGroupName", "stackyard-group")
		if groupName == "" {
			groupName = codeDeployDefaultString(payload, "deploymentGroupName", "stackyard-group")
		}
		grp := s.ensureDeploymentGroupLocked(appName, groupName)
		newName := codeDeployString(payload["newDeploymentGroupName"])
		if newName == "" {
			newName = grp.Name
		}
		if cfg := codeDeployString(payload["deploymentConfigName"]); cfg != "" {
			grp.DeploymentConfigName = cfg
			s.ensureDeploymentConfigLocked(cfg)
		}
		if role := codeDeployString(payload["serviceRoleArn"]); role != "" {
			grp.ServiceRoleArn = role
		}
		if style, ok := payload["deploymentStyle"].(map[string]any); ok && len(style) > 0 {
			grp.DeploymentStyle = codeDeployCloneMapAny(style)
		}
		if newName != grp.Name {
			delete(s.deploymentGroups, codeDeployGroupKey(appName, grp.Name))
			for _, dep := range s.deployments {
				if dep.ApplicationName == appName && dep.DeploymentGroupName == grp.Name {
					dep.DeploymentGroupName = newName
				}
			}
			grp.Name = newName
			s.deploymentGroups[codeDeployGroupKey(appName, newName)] = grp
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *codeDeployStore) ensureApplicationLocked(name string) *codeDeployApplication {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-app"
	}
	if existing, ok := s.applications[name]; ok {
		return existing
	}
	app := &codeDeployApplication{
		ID:              fmt.Sprintf("app-%012d", s.nextLocked()),
		Name:            name,
		ComputePlatform: "Server",
		CreateTime:      time.Now().UTC(),
	}
	s.applications[name] = app
	return app
}

func (s *codeDeployStore) ensureDeploymentConfigLocked(name string) *codeDeployDeploymentConfig {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "CodeDeployDefault.AllAtOnce"
	}
	if existing, ok := s.deploymentConfigs[name]; ok {
		return existing
	}
	cfg := &codeDeployDeploymentConfig{
		ID:         fmt.Sprintf("dc-%012d", s.nextLocked()),
		Name:       name,
		CreateTime: time.Now().UTC(),
		MinimumHealthy: map[string]any{
			"type":  "HOST_COUNT",
			"value": 1,
		},
		TrafficRoutingCfg: map[string]any{},
	}
	s.deploymentConfigs[name] = cfg
	return cfg
}

func (s *codeDeployStore) ensureDeploymentGroupLocked(appName, groupName string) *codeDeployDeploymentGroup {
	app := s.ensureApplicationLocked(appName)
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = "stackyard-group"
	}
	key := codeDeployGroupKey(app.Name, groupName)
	if existing, ok := s.deploymentGroups[key]; ok {
		return existing
	}
	cfg := s.ensureDeploymentConfigLocked("CodeDeployDefault.AllAtOnce")
	grp := &codeDeployDeploymentGroup{
		ID:                   fmt.Sprintf("dg-%012d", s.nextLocked()),
		ApplicationName:      app.Name,
		Name:                 groupName,
		DeploymentConfigName: cfg.Name,
		ServiceRoleArn:       "arn:aws:iam::123456789012:role/stackyard-codedeploy",
		CreateTime:           time.Now().UTC(),
		AutoScalingGroups:    []string{"stackyard-asg"},
		EC2TagFilters:        []map[string]any{},
		OnPremisesTagFilters: []map[string]any{},
		DeploymentStyle: map[string]any{
			"deploymentType":   "IN_PLACE",
			"deploymentOption": "WITHOUT_TRAFFIC_CONTROL",
		},
	}
	s.deploymentGroups[key] = grp
	return grp
}

func (s *codeDeployStore) ensureDeploymentLocked(id string) *codeDeployDeployment {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "d-000000000001"
	}
	if existing, ok := s.deployments[id]; ok {
		return existing
	}
	app := s.ensureApplicationLocked("stackyard-app")
	group := s.ensureDeploymentGroupLocked(app.Name, "stackyard-group")
	dep := &codeDeployDeployment{
		ID:                  id,
		ApplicationName:     app.Name,
		DeploymentGroupName: group.Name,
		Description:         "synthesized deployment",
		Status:              "Succeeded",
		CreateTime:          time.Now().UTC(),
		CompleteTime:        ptrTime(time.Now().UTC()),
		Revision:            codeDeployDefaultRevision(),
		ErrorInformation:    map[string]any{},
		TargetIDs:           []string{"i-000000000001"},
	}
	s.deployments[id] = dep
	group.LastAttemptedID = dep.ID
	group.LastSuccessfulID = dep.ID
	return dep
}

func (s *codeDeployStore) ensureOnPremInstanceLocked(name string) *codeDeployOnPremInstance {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-onprem-1"
	}
	if existing, ok := s.onPremInstances[name]; ok {
		return existing
	}
	inst := &codeDeployOnPremInstance{
		Name:         name,
		IamUserArn:   "arn:aws:iam::123456789012:user/stackyard-codedeploy",
		RegisterTime: time.Now().UTC(),
		Tags:         map[string]string{},
	}
	s.onPremInstances[name] = inst
	return inst
}

func (s *codeDeployStore) ensureRevisionMapLocked(appName string) map[string]map[string]any {
	appName = s.ensureApplicationLocked(appName).Name
	if s.revisions[appName] == nil {
		s.revisions[appName] = map[string]map[string]any{}
	}
	return s.revisions[appName]
}

func (s *codeDeployStore) ensureResourceTagMapLocked(arn string) map[string]string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		arn = codeDeployApplicationARN("stackyard-app")
	}
	if s.resourceTags[arn] == nil {
		s.resourceTags[arn] = map[string]string{}
	}
	return s.resourceTags[arn]
}

func (s *codeDeployStore) listApplicationNamesLocked() []string {
	names := make([]string, 0, len(s.applications))
	for name := range s.applications {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		names = []string{"stackyard-app"}
	}
	return names
}

func (s *codeDeployStore) listDeploymentGroupNamesLocked(appName string) []string {
	appName = strings.TrimSpace(appName)
	names := make([]string, 0, len(s.deploymentGroups))
	for _, grp := range s.deploymentGroups {
		if appName == "" || grp.ApplicationName == appName {
			names = append(names, grp.Name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		defaultGroup := s.ensureDeploymentGroupLocked(codeDeployDefaultString(map[string]any{"applicationName": appName}, "applicationName", "stackyard-app"), "stackyard-group")
		names = []string{defaultGroup.Name}
	}
	return names
}

func (s *codeDeployStore) listDeploymentIDsLocked(appName, groupName string) []string {
	appName = strings.TrimSpace(appName)
	groupName = strings.TrimSpace(groupName)
	ids := make([]string, 0, len(s.deployments))
	for id, dep := range s.deployments {
		if appName != "" && dep.ApplicationName != appName {
			continue
		}
		if groupName != "" && dep.DeploymentGroupName != groupName {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		ids = []string{s.ensureDeploymentLocked("d-000000000001").ID}
	}
	return ids
}

func (s *codeDeployStore) listOnPremInstanceNamesLocked() []string {
	names := make([]string, 0, len(s.onPremInstances))
	for name := range s.onPremInstances {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		names = []string{s.ensureOnPremInstanceLocked("stackyard-onprem-1").Name}
	}
	return names
}

func (s *codeDeployStore) applicationPayloadLocked(name string) map[string]any {
	app := s.ensureApplicationLocked(name)
	return map[string]any{
		"applicationId":   app.ID,
		"applicationName": app.Name,
		"createTime":      app.CreateTime,
		"computePlatform": app.ComputePlatform,
		"linkedToGitHub":  false,
	}
}

func (s *codeDeployStore) deploymentConfigPayloadLocked(name string) map[string]any {
	cfg := s.ensureDeploymentConfigLocked(name)
	return map[string]any{
		"deploymentConfigId":   cfg.ID,
		"deploymentConfigName": cfg.Name,
		"createTime":           cfg.CreateTime,
		"minimumHealthyHosts":  codeDeployCloneMapAny(cfg.MinimumHealthy),
		"trafficRoutingConfig": codeDeployCloneMapAny(cfg.TrafficRoutingCfg),
	}
}

func (s *codeDeployStore) deploymentGroupPayloadLocked(appName, groupName string) map[string]any {
	grp := s.ensureDeploymentGroupLocked(appName, groupName)
	payload := map[string]any{
		"deploymentGroupId":            grp.ID,
		"applicationName":              grp.ApplicationName,
		"deploymentGroupName":          grp.Name,
		"deploymentConfigName":         grp.DeploymentConfigName,
		"serviceRoleArn":               grp.ServiceRoleArn,
		"autoScalingGroups":            append([]string{}, grp.AutoScalingGroups...),
		"ec2TagFilters":                codeDeployCloneSliceMap(grp.EC2TagFilters),
		"onPremisesInstanceTagFilters": codeDeployCloneSliceMap(grp.OnPremisesTagFilters),
		"deploymentStyle":              codeDeployCloneMapAny(grp.DeploymentStyle),
		"targetRevision":               codeDeployDefaultRevision(),
		"triggerConfigurations":        []any{},
		"outdatedInstancesStrategy":    "UPDATE",
		"deploymentGroupArn":           codeDeployDeploymentGroupARN(grp.ApplicationName, grp.Name),
		"computePlatform":              "Server",
	}
	if grp.LastAttemptedID != "" {
		payload["lastAttemptedDeployment"] = map[string]any{
			"deploymentId": grp.LastAttemptedID,
			"status":       s.ensureDeploymentLocked(grp.LastAttemptedID).Status,
			"endTime":      time.Now().UTC(),
			"createTime":   time.Now().UTC(),
		}
	}
	if grp.LastSuccessfulID != "" {
		payload["lastSuccessfulDeployment"] = map[string]any{
			"deploymentId": grp.LastSuccessfulID,
			"status":       s.ensureDeploymentLocked(grp.LastSuccessfulID).Status,
			"endTime":      time.Now().UTC(),
			"createTime":   time.Now().UTC(),
		}
	}
	return payload
}

func (s *codeDeployStore) deploymentPayloadLocked(deploymentID string) map[string]any {
	dep := s.ensureDeploymentLocked(deploymentID)
	payload := map[string]any{
		"applicationName":      dep.ApplicationName,
		"deploymentGroupName":  dep.DeploymentGroupName,
		"deploymentConfigName": s.ensureDeploymentGroupLocked(dep.ApplicationName, dep.DeploymentGroupName).DeploymentConfigName,
		"deploymentId":         dep.ID,
		"status":               dep.Status,
		"errorInformation":     codeDeployCloneMapAny(dep.ErrorInformation),
		"createTime":           dep.CreateTime,
		"creator":              "user",
		"description":          dep.Description,
		"revision":             codeDeployCloneMapAny(dep.Revision),
		"deploymentOverview": map[string]any{
			"Pending":    0,
			"InProgress": 0,
			"Succeeded":  len(dep.TargetIDs),
			"Failed":     0,
			"Skipped":    0,
			"Ready":      0,
		},
		"computePlatform": "Server",
	}
	if dep.CompleteTime != nil {
		payload["completeTime"] = *dep.CompleteTime
	}
	return payload
}

func (s *codeDeployStore) deploymentTargetPayloadLocked(dep *codeDeployDeployment, targetID string) map[string]any {
	if strings.TrimSpace(targetID) == "" {
		targetID = "i-000000000001"
	}
	return map[string]any{
		"deploymentTargetType": "InstanceTarget",
		"instanceTarget": map[string]any{
			"deploymentId": dep.ID,
			"targetId":     targetID,
			"status":       dep.Status,
			"instanceLabel": map[string]any{
				"value": targetID,
				"type":  "InstanceId",
			},
			"lifecycleEvents": []any{},
			"lastUpdatedAt":   time.Now().UTC(),
		},
	}
}

func (s *codeDeployStore) onPremInstancePayloadLocked(name string) map[string]any {
	inst := s.ensureOnPremInstanceLocked(name)
	payload := map[string]any{
		"instanceName":   inst.Name,
		"iamUserArn":     inst.IamUserArn,
		"registerTime":   inst.RegisterTime,
		"instanceArn":    codeDeployOnPremInstanceARN(inst.Name),
		"deregisterTime": nil,
		"tags":           codeDeploySortedTagsPayload(inst.Tags),
	}
	if inst.Deregistered {
		now := time.Now().UTC()
		payload["deregisterTime"] = now
	}
	return payload
}

func (s *codeDeployStore) nextLocked() int64 {
	next := s.nextID
	s.nextID++
	return next
}

func codeDeployDefaultRevision() map[string]any {
	return map[string]any{
		"revisionType": "S3",
		"s3Location": map[string]any{
			"bucket":     "stackyard-codedeploy-revisions",
			"key":        "seed.zip",
			"bundleType": "zip",
			"version":    "1",
			"eTag":       "seed",
		},
	}
}

func codeDeployGroupKey(appName, groupName string) string {
	return strings.TrimSpace(appName) + "|" + strings.TrimSpace(groupName)
}

func codeDeployRevisionKey(revision map[string]any) string {
	if len(revision) == 0 {
		return "S3:stackyard-codedeploy-revisions/seed.zip"
	}
	revType := strings.TrimSpace(codeDeployString(revision["revisionType"]))
	if revType == "" {
		revType = "S3"
	}
	if s3, ok := revision["s3Location"].(map[string]any); ok {
		bucket := strings.TrimSpace(codeDeployString(s3["bucket"]))
		key := strings.TrimSpace(codeDeployString(s3["key"]))
		if bucket != "" || key != "" {
			return revType + ":" + bucket + "/" + key
		}
	}
	if gh, ok := revision["gitHubLocation"].(map[string]any); ok {
		repo := strings.TrimSpace(codeDeployString(gh["repository"]))
		commit := strings.TrimSpace(codeDeployString(gh["commitId"]))
		if repo != "" || commit != "" {
			return revType + ":" + repo + "@" + commit
		}
	}
	return revType + ":default"
}

func codeDeployDefaultString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	if value := strings.TrimSpace(codeDeployString(payload[key])); value != "" {
		return value
	}
	return fallback
}

func codeDeployString(value any) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}

func codeDeployStringSlice(value any) []string {
	list, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]string); ok {
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s := strings.TrimSpace(codeDeployString(item)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func codeDeployExtractTags(value any) map[string]string {
	list, ok := value.([]any)
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, item := range list {
		tag, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.TrimSpace(codeDeployString(tag["Key"]))
		if key == "" {
			key = strings.TrimSpace(codeDeployString(tag["key"]))
		}
		if key == "" {
			continue
		}
		value := strings.TrimSpace(codeDeployString(tag["Value"]))
		if value == "" {
			value = strings.TrimSpace(codeDeployString(tag["value"]))
		}
		out[key] = value
	}
	return out
}

func codeDeploySortedTagsPayload(tags map[string]string) []map[string]string {
	if len(tags) == 0 {
		return []map[string]string{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{"Key": key, "Value": tags[key]})
	}
	return out
}

func codeDeployInstanceSummaryPayload(instanceID, deploymentStatus string) map[string]any {
	if strings.TrimSpace(instanceID) == "" {
		instanceID = "i-000000000001"
	}
	if strings.TrimSpace(deploymentStatus) == "" {
		deploymentStatus = "Succeeded"
	}
	return map[string]any{
		"instanceId":      instanceID,
		"status":          deploymentStatus,
		"lastUpdatedAt":   time.Now().UTC(),
		"lifecycleEvents": []any{},
	}
}

func codeDeployCloneMapAny(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func codeDeployCloneSliceMap(in []map[string]any) []map[string]any {
	if in == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, codeDeployCloneMapAny(item))
	}
	return out
}

func codeDeployApplicationARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-app"
	}
	return "arn:aws:codedeploy:us-east-1:123456789012:application:" + name
}

func codeDeployDeploymentGroupARN(appName, groupName string) string {
	appName = strings.TrimSpace(appName)
	if appName == "" {
		appName = "stackyard-app"
	}
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		groupName = "stackyard-group"
	}
	return "arn:aws:codedeploy:us-east-1:123456789012:deploymentgroup:" + appName + "/" + groupName
}

func codeDeployDeploymentARN(deploymentID string) string {
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		deploymentID = "d-000000000001"
	}
	return "arn:aws:codedeploy:us-east-1:123456789012:deployment:" + deploymentID
}

func codeDeployOnPremInstanceARN(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "stackyard-onprem-1"
	}
	return "arn:aws:codedeploy:us-east-1:123456789012:instance:" + name
}

func ptrTime(t time.Time) *time.Time {
	v := t
	return &v
}
