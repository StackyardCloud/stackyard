package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type mgnStore struct {
	mu sync.Mutex

	initialized bool

	nextApplicationID                 int64
	nextWaveID                        int64
	nextConnectorID                   int64
	nextLaunchConfigurationTemplateID int64
	nextReplicationTemplateID         int64
	nextSourceServerID                int64
	nextVcenterClientID               int64
	nextJobID                         int64
	nextExportID                      int64
	nextImportID                      int64

	applications                  map[string]map[string]any
	waves                         map[string]map[string]any
	connectors                    map[string]map[string]any
	launchConfigurationTemplates  map[string]map[string]any
	replicationConfigurationTemps map[string]map[string]any
	sourceServers                 map[string]map[string]any
	vcenterClients                map[string]map[string]any
	jobs                          map[string]map[string]any
	exports                       map[string]map[string]any
	imports                       map[string]map[string]any
	sourceServerActions           map[string]map[string]map[string]any
	templateActions               map[string]map[string]map[string]any
	tags                          map[string]map[string]string
	managedAccounts               []map[string]any
	sourceServerToApplication     map[string]string
	applicationToWave             map[string]string
}

func newMGNStore() *mgnStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &mgnStore{
		initialized:                       true,
		nextApplicationID:                 2,
		nextWaveID:                        2,
		nextConnectorID:                   2,
		nextLaunchConfigurationTemplateID: 2,
		nextReplicationTemplateID:         2,
		nextSourceServerID:                2,
		nextVcenterClientID:               2,
		nextJobID:                         2,
		nextExportID:                      2,
		nextImportID:                      2,
		applications:                      map[string]map[string]any{},
		waves:                             map[string]map[string]any{},
		connectors:                        map[string]map[string]any{},
		launchConfigurationTemplates:      map[string]map[string]any{},
		replicationConfigurationTemps:     map[string]map[string]any{},
		sourceServers:                     map[string]map[string]any{},
		vcenterClients:                    map[string]map[string]any{},
		jobs:                              map[string]map[string]any{},
		exports:                           map[string]map[string]any{},
		imports:                           map[string]map[string]any{},
		sourceServerActions:               map[string]map[string]map[string]any{},
		templateActions:                   map[string]map[string]map[string]any{},
		tags:                              map[string]map[string]string{},
		managedAccounts: []map[string]any{
			{
				"accountID":  "123456789012",
				"ec2Service": "ENABLED",
			},
		},
		sourceServerToApplication: map[string]string{},
		applicationToWave:         map[string]string{},
	}

	appID := "app-00000001"
	waveID := "wave-00000001"
	connectorID := "connector-00000001"
	launchTemplateID := "lct-00000001"
	replTemplateID := "rct-00000001"
	sourceServerID := "s-00000001"
	vcenterID := "vcenter-00000001"
	jobID := "job-00000001"
	exportID := "export-00000001"
	importID := "import-00000001"

	s.applications[appID] = map[string]any{
		"applicationID":    appID,
		"accountID":        "123456789012",
		"name":             "stackyard-app",
		"description":      "seed application",
		"isArchived":       false,
		"creationDateTime": now,
		"arn":              mgnResourceARN("application", appID),
	}
	s.waves[waveID] = map[string]any{
		"waveID":           waveID,
		"accountID":        "123456789012",
		"name":             "stackyard-wave",
		"description":      "seed wave",
		"isArchived":       false,
		"creationDateTime": now,
		"arn":              mgnResourceARN("wave", waveID),
	}
	s.applicationToWave[appID] = waveID
	s.connectors[connectorID] = map[string]any{
		"connectorID": connectorID,
		"name":        "stackyard-connector",
		"arn":         mgnResourceARN("connector", connectorID),
	}
	s.launchConfigurationTemplates[launchTemplateID] = map[string]any{
		"launchConfigurationTemplateID": launchTemplateID,
		"copyTags":                      true,
		"launchDisposition":             "STOPPED",
		"createdBy":                     "stackyard",
		"arn":                           mgnResourceARN("launch-configuration-template", launchTemplateID),
	}
	s.replicationConfigurationTemps[replTemplateID] = map[string]any{
		"replicationConfigurationTemplateID": replTemplateID,
		"associateDefaultSecurityGroup":      true,
		"replicationServerInstanceType":      "t3.small",
		"arn":                                mgnResourceARN("replication-configuration-template", replTemplateID),
	}
	s.sourceServers[sourceServerID] = map[string]any{
		"sourceServerID":      sourceServerID,
		"accountID":           "123456789012",
		"arn":                 mgnResourceARN("source-server", sourceServerID),
		"isArchived":          false,
		"tags":                map[string]any{"env": "test"},
		"lifeCycle":           map[string]any{"state": "READY_FOR_TEST"},
		"lastUpdatedDateTime": now,
	}
	s.sourceServerToApplication[sourceServerID] = appID
	s.vcenterClients[vcenterID] = map[string]any{
		"vcenterClientID": vcenterID,
		"hostname":        "vcenter.local",
		"arn":             mgnResourceARN("vcenter-client", vcenterID),
	}
	s.jobs[jobID] = map[string]any{
		"jobID":            jobID,
		"status":           "COMPLETED",
		"type":             "LAUNCH",
		"creationDateTime": now,
	}
	s.exports[exportID] = map[string]any{
		"exportID": exportID,
		"s3Bucket": "stackyard-mgn-export",
		"s3Key":    "export.json",
		"status":   "SUCCEEDED",
	}
	s.imports[importID] = map[string]any{
		"importID": importID,
		"s3Bucket": "stackyard-mgn-import",
		"s3Key":    "import.json",
		"status":   "SUCCEEDED",
	}
	s.tags[mgnResourceARN("application", appID)] = map[string]string{"env": "test"}

	return s
}

func (s *mgnStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedDataLocked()

	switch action {
	case "InitializeService":
		s.initialized = true
		return map[string]any{"status": "INITIALIZED"}

	case "CreateApplication":
		id := fmt.Sprintf("app-%08d", s.nextApplicationID)
		s.nextApplicationID++
		item := map[string]any{
			"applicationID":    id,
			"accountID":        mgnFirstNonEmpty(mgnString(payload, "accountID"), "123456789012"),
			"name":             mgnFirstNonEmpty(mgnString(payload, "name"), id),
			"description":      mgnString(payload, "description"),
			"isArchived":       false,
			"creationDateTime": time.Now().UTC().Format(time.RFC3339),
			"arn":              mgnResourceARN("application", id),
		}
		s.applications[id] = item
		return mgnCloneMap(item)

	case "UpdateApplication", "ArchiveApplication", "UnarchiveApplication", "DeleteApplication":
		id := mgnFirstNonEmpty(mgnString(payload, "applicationID"), s.firstApplicationIDLocked())
		item := s.ensureApplicationLocked(id)
		switch action {
		case "UpdateApplication":
			if v := mgnString(payload, "name"); v != "" {
				item["name"] = v
			}
			if v := mgnString(payload, "description"); v != "" {
				item["description"] = v
			}
			return mgnCloneMap(item)
		case "ArchiveApplication":
			item["isArchived"] = true
			return mgnCloneMap(item)
		case "UnarchiveApplication":
			item["isArchived"] = false
			return mgnCloneMap(item)
		default:
			delete(s.applications, id)
			delete(s.applicationToWave, id)
			return map[string]any{"applicationID": id}
		}

	case "CreateWave":
		id := fmt.Sprintf("wave-%08d", s.nextWaveID)
		s.nextWaveID++
		item := map[string]any{
			"waveID":           id,
			"accountID":        mgnFirstNonEmpty(mgnString(payload, "accountID"), "123456789012"),
			"name":             mgnFirstNonEmpty(mgnString(payload, "name"), id),
			"description":      mgnString(payload, "description"),
			"isArchived":       false,
			"creationDateTime": time.Now().UTC().Format(time.RFC3339),
			"arn":              mgnResourceARN("wave", id),
		}
		s.waves[id] = item
		return mgnCloneMap(item)

	case "UpdateWave", "ArchiveWave", "UnarchiveWave", "DeleteWave":
		id := mgnFirstNonEmpty(mgnString(payload, "waveID"), s.firstWaveIDLocked())
		item := s.ensureWaveLocked(id)
		switch action {
		case "UpdateWave":
			if v := mgnString(payload, "name"); v != "" {
				item["name"] = v
			}
			if v := mgnString(payload, "description"); v != "" {
				item["description"] = v
			}
			return mgnCloneMap(item)
		case "ArchiveWave":
			item["isArchived"] = true
			return mgnCloneMap(item)
		case "UnarchiveWave":
			item["isArchived"] = false
			return mgnCloneMap(item)
		default:
			delete(s.waves, id)
			for appID, waveID := range s.applicationToWave {
				if waveID == id {
					delete(s.applicationToWave, appID)
				}
			}
			return map[string]any{"waveID": id}
		}

	case "AssociateApplications", "DisassociateApplications":
		waveID := mgnFirstNonEmpty(mgnString(payload, "waveID"), s.firstWaveIDLocked())
		_ = s.ensureWaveLocked(waveID)
		applicationIDs := mgnStringSlice(payload, "applicationIDs")
		if len(applicationIDs) == 0 {
			applicationIDs = []string{s.firstApplicationIDLocked()}
		}
		for _, applicationID := range applicationIDs {
			if applicationID == "" {
				continue
			}
			_ = s.ensureApplicationLocked(applicationID)
			if action == "AssociateApplications" {
				s.applicationToWave[applicationID] = waveID
			} else {
				delete(s.applicationToWave, applicationID)
			}
		}
		return map[string]any{"waveID": waveID, "applicationIDs": applicationIDs}

	case "AssociateSourceServers", "DisassociateSourceServers":
		appID := mgnFirstNonEmpty(mgnString(payload, "applicationID"), s.firstApplicationIDLocked())
		_ = s.ensureApplicationLocked(appID)
		serverIDs := mgnStringSlice(payload, "sourceServerIDs")
		if len(serverIDs) == 0 {
			serverIDs = []string{s.firstSourceServerIDLocked()}
		}
		for _, serverID := range serverIDs {
			if serverID == "" {
				continue
			}
			_ = s.ensureSourceServerLocked(serverID)
			if action == "AssociateSourceServers" {
				s.sourceServerToApplication[serverID] = appID
			} else {
				delete(s.sourceServerToApplication, serverID)
			}
		}
		return map[string]any{"applicationID": appID, "sourceServerIDs": serverIDs}

	case "CreateConnector":
		id := fmt.Sprintf("connector-%08d", s.nextConnectorID)
		s.nextConnectorID++
		item := map[string]any{
			"connectorID": id,
			"name":        mgnFirstNonEmpty(mgnString(payload, "name"), id),
			"arn":         mgnResourceARN("connector", id),
		}
		s.connectors[id] = item
		return mgnCloneMap(item)

	case "UpdateConnector", "DeleteConnector":
		id := mgnFirstNonEmpty(mgnString(payload, "connectorID"), s.firstConnectorIDLocked())
		item := s.ensureConnectorLocked(id)
		if action == "DeleteConnector" {
			delete(s.connectors, id)
			return map[string]any{"connectorID": id}
		}
		if v := mgnString(payload, "name"); v != "" {
			item["name"] = v
		}
		return mgnCloneMap(item)

	case "CreateLaunchConfigurationTemplate":
		id := fmt.Sprintf("lct-%08d", s.nextLaunchConfigurationTemplateID)
		s.nextLaunchConfigurationTemplateID++
		item := map[string]any{
			"launchConfigurationTemplateID": id,
			"copyTags":                      mgnBool(payload, "copyTags", true),
			"launchDisposition":             mgnFirstNonEmpty(mgnString(payload, "launchDisposition"), "STOPPED"),
			"arn":                           mgnResourceARN("launch-configuration-template", id),
		}
		s.launchConfigurationTemplates[id] = item
		return mgnCloneMap(item)

	case "UpdateLaunchConfigurationTemplate", "DeleteLaunchConfigurationTemplate":
		id := mgnFirstNonEmpty(mgnString(payload, "launchConfigurationTemplateID"), s.firstLaunchTemplateIDLocked())
		item := s.ensureLaunchTemplateLocked(id)
		if action == "DeleteLaunchConfigurationTemplate" {
			delete(s.launchConfigurationTemplates, id)
			delete(s.templateActions, id)
			return map[string]any{"launchConfigurationTemplateID": id}
		}
		if v := mgnString(payload, "launchDisposition"); v != "" {
			item["launchDisposition"] = v
		}
		item["copyTags"] = mgnBool(payload, "copyTags", mgnBoolFromAny(item["copyTags"], true))
		return mgnCloneMap(item)

	case "CreateReplicationConfigurationTemplate":
		id := fmt.Sprintf("rct-%08d", s.nextReplicationTemplateID)
		s.nextReplicationTemplateID++
		item := map[string]any{
			"replicationConfigurationTemplateID": id,
			"associateDefaultSecurityGroup":      mgnBool(payload, "associateDefaultSecurityGroup", true),
			"replicationServerInstanceType":      mgnFirstNonEmpty(mgnString(payload, "replicationServerInstanceType"), "t3.small"),
			"arn":                                mgnResourceARN("replication-configuration-template", id),
		}
		s.replicationConfigurationTemps[id] = item
		return mgnCloneMap(item)

	case "UpdateReplicationConfigurationTemplate", "DeleteReplicationConfigurationTemplate":
		id := mgnFirstNonEmpty(mgnString(payload, "replicationConfigurationTemplateID"), s.firstReplicationTemplateIDLocked())
		item := s.ensureReplicationTemplateLocked(id)
		if action == "DeleteReplicationConfigurationTemplate" {
			delete(s.replicationConfigurationTemps, id)
			return map[string]any{"replicationConfigurationTemplateID": id}
		}
		if v := mgnString(payload, "replicationServerInstanceType"); v != "" {
			item["replicationServerInstanceType"] = v
		}
		item["associateDefaultSecurityGroup"] = mgnBool(payload, "associateDefaultSecurityGroup", mgnBoolFromAny(item["associateDefaultSecurityGroup"], true))
		return mgnCloneMap(item)

	case "GetLaunchConfiguration":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		server := s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID":    sourceServerID,
			"launchDisposition": "STOPPED",
			"copyTags":          true,
			"name":              server["sourceServerID"],
		}

	case "GetReplicationConfiguration":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		_ = s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID":                sourceServerID,
			"replicationServerInstanceType": "t3.small",
			"bandwidthThrottling":           0,
		}

	case "UpdateLaunchConfiguration", "UpdateReplicationConfiguration", "UpdateSourceServer", "UpdateSourceServerReplicationType", "PauseReplication", "ResumeReplication", "RetryDataReplication", "StartReplication", "StopReplication", "DisconnectFromService", "FinalizeCutover", "MarkAsArchived":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		server := s.ensureSourceServerLocked(sourceServerID)
		now := time.Now().UTC().Format(time.RFC3339)
		server["lastUpdatedDateTime"] = now
		switch action {
		case "MarkAsArchived":
			server["isArchived"] = true
		case "PauseReplication":
			server["replicationStatus"] = "PAUSED"
		case "ResumeReplication", "StartReplication":
			server["replicationStatus"] = "CONTINUOUS"
		case "StopReplication":
			server["replicationStatus"] = "STOPPED"
		case "RetryDataReplication":
			server["replicationStatus"] = "RETRYING"
		case "DisconnectFromService":
			server["disconnected"] = true
		case "FinalizeCutover":
			server["lifeCycle"] = map[string]any{"state": "CUTOVER"}
		case "UpdateSourceServerReplicationType":
			server["replicationType"] = mgnFirstNonEmpty(mgnString(payload, "replicationType"), "AGENT_BASED")
		}
		return mgnCloneMap(server)

	case "ChangeServerLifeCycleState":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		server := s.ensureSourceServerLocked(sourceServerID)
		if lc, ok := payload["lifeCycle"].(map[string]any); ok {
			if state := mgnString(lc, "state"); state != "" {
				server["lifeCycle"] = map[string]any{"state": state}
			}
		}
		return mgnCloneMap(server)

	case "StartCutover", "StartTest", "TerminateTargetInstances":
		ids := mgnStringSlice(payload, "sourceServerIDs")
		if len(ids) == 0 {
			ids = []string{s.firstSourceServerIDLocked()}
		}
		jobID := fmt.Sprintf("job-%08d", s.nextJobID)
		s.nextJobID++
		job := map[string]any{
			"jobID":            jobID,
			"status":           "STARTED",
			"type":             strings.TrimPrefix(action, "Start"),
			"sourceServerIDs":  ids,
			"creationDateTime": time.Now().UTC().Format(time.RFC3339),
		}
		s.jobs[jobID] = job
		return map[string]any{"job": mgnCloneMap(job)}

	case "StartExport":
		exportID := fmt.Sprintf("export-%08d", s.nextExportID)
		s.nextExportID++
		item := map[string]any{
			"exportID": exportID,
			"status":   "STARTED",
			"s3Bucket": mgnString(payload, "s3Bucket"),
			"s3Key":    mgnString(payload, "s3Key"),
		}
		s.exports[exportID] = item
		return map[string]any{"exportTask": mgnCloneMap(item)}

	case "StartImport":
		importID := fmt.Sprintf("import-%08d", s.nextImportID)
		s.nextImportID++
		bucket := ""
		key := ""
		if src, ok := payload["s3BucketSource"].(map[string]any); ok {
			bucket = mgnString(src, "s3Bucket")
			key = mgnString(src, "s3Key")
		}
		item := map[string]any{
			"importID": importID,
			"status":   "STARTED",
			"s3Bucket": bucket,
			"s3Key":    key,
		}
		s.imports[importID] = item
		return map[string]any{"importTask": mgnCloneMap(item)}

	case "PutSourceServerAction":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		actionID := mgnFirstNonEmpty(mgnString(payload, "actionID"), "action-00000001")
		if _, ok := s.sourceServerActions[sourceServerID]; !ok {
			s.sourceServerActions[sourceServerID] = map[string]map[string]any{}
		}
		stored := mgnCloneMap(payload)
		stored["sourceServerID"] = sourceServerID
		stored["actionID"] = actionID
		s.sourceServerActions[sourceServerID][actionID] = stored
		return mgnCloneMap(stored)

	case "PutTemplateAction":
		templateID := mgnFirstNonEmpty(mgnString(payload, "launchConfigurationTemplateID"), s.firstLaunchTemplateIDLocked())
		actionID := mgnFirstNonEmpty(mgnString(payload, "actionID"), "action-00000001")
		if _, ok := s.templateActions[templateID]; !ok {
			s.templateActions[templateID] = map[string]map[string]any{}
		}
		stored := mgnCloneMap(payload)
		stored["launchConfigurationTemplateID"] = templateID
		stored["actionID"] = actionID
		s.templateActions[templateID][actionID] = stored
		return mgnCloneMap(stored)

	case "RemoveSourceServerAction":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		actionID := mgnFirstNonEmpty(mgnString(payload, "actionID"), "action-00000001")
		if actions, ok := s.sourceServerActions[sourceServerID]; ok {
			delete(actions, actionID)
		}
		return map[string]any{"sourceServerID": sourceServerID, "actionID": actionID}

	case "RemoveTemplateAction":
		templateID := mgnFirstNonEmpty(mgnString(payload, "launchConfigurationTemplateID"), s.firstLaunchTemplateIDLocked())
		actionID := mgnFirstNonEmpty(mgnString(payload, "actionID"), "action-00000001")
		if actions, ok := s.templateActions[templateID]; ok {
			delete(actions, actionID)
		}
		return map[string]any{"launchConfigurationTemplateID": templateID, "actionID": actionID}

	case "TagResource":
		arn := mgnFirstNonEmpty(mgnString(payload, "resourceArn"), mgnResourceARN("application", s.firstApplicationIDLocked()))
		if _, ok := s.tags[arn]; !ok {
			s.tags[arn] = map[string]string{}
		}
		if tagMap, ok := payload["tags"].(map[string]any); ok {
			for k, v := range tagMap {
				s.tags[arn][k] = fmt.Sprint(v)
			}
		}
		return map[string]any{"resourceArn": arn}

	case "UntagResource":
		arn := mgnFirstNonEmpty(mgnString(payload, "resourceArn"), mgnResourceARN("application", s.firstApplicationIDLocked()))
		tagKeys := mgnStringSlice(payload, "tagKeys")
		if tags, ok := s.tags[arn]; ok {
			for _, key := range tagKeys {
				delete(tags, key)
			}
		}
		return map[string]any{"resourceArn": arn}

	case "DeleteSourceServer":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		delete(s.sourceServers, sourceServerID)
		delete(s.sourceServerToApplication, sourceServerID)
		delete(s.sourceServerActions, sourceServerID)
		return map[string]any{"sourceServerID": sourceServerID}

	case "DeleteVcenterClient":
		id := mgnFirstNonEmpty(mgnString(payload, "vcenterClientID"), s.firstVcenterClientIDLocked())
		delete(s.vcenterClients, id)
		return map[string]any{"vcenterClientID": id}

	case "DeleteJob":
		id := mgnFirstNonEmpty(mgnString(payload, "jobID"), s.firstJobIDLocked())
		delete(s.jobs, id)
		return map[string]any{"jobID": id}

	case "DescribeLaunchConfigurationTemplates":
		return map[string]any{"items": s.sortedValuesLocked(s.launchConfigurationTemplates), "nextToken": ""}
	case "DescribeReplicationConfigurationTemplates":
		return map[string]any{"items": s.sortedValuesLocked(s.replicationConfigurationTemps), "nextToken": ""}
	case "DescribeSourceServers":
		return map[string]any{"items": s.sortedValuesLocked(s.sourceServers), "nextToken": ""}
	case "DescribeVcenterClients":
		return map[string]any{"items": s.sortedValuesLocked(s.vcenterClients), "nextToken": ""}
	case "DescribeJobs":
		return map[string]any{"items": s.sortedValuesLocked(s.jobs), "nextToken": ""}
	case "DescribeJobLogItems":
		return map[string]any{"items": []any{}, "nextToken": ""}
	case "ListApplications":
		items := s.sortedValuesLocked(s.applications)
		for _, item := range items {
			itemMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if appID := mgnString(itemMap, "applicationID"); appID != "" {
				if waveID := s.applicationToWave[appID]; waveID != "" {
					itemMap["waveID"] = waveID
				}
			}
		}
		return map[string]any{"items": items, "nextToken": ""}
	case "ListWaves":
		return map[string]any{"items": s.sortedValuesLocked(s.waves), "nextToken": ""}
	case "ListConnectors":
		return map[string]any{"items": s.sortedValuesLocked(s.connectors), "nextToken": ""}
	case "ListManagedAccounts":
		items := make([]any, 0, len(s.managedAccounts))
		for _, item := range s.managedAccounts {
			items = append(items, mgnCloneMap(item))
		}
		return map[string]any{"items": items, "nextToken": ""}
	case "ListSourceServerActions":
		sourceServerID := mgnFirstNonEmpty(mgnString(payload, "sourceServerID"), s.firstSourceServerIDLocked())
		items := []any{}
		if actions, ok := s.sourceServerActions[sourceServerID]; ok {
			ids := make([]string, 0, len(actions))
			for id := range actions {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				items = append(items, mgnCloneMap(actions[id]))
			}
		}
		return map[string]any{"items": items, "nextToken": ""}
	case "ListTemplateActions":
		templateID := mgnFirstNonEmpty(mgnString(payload, "launchConfigurationTemplateID"), s.firstLaunchTemplateIDLocked())
		items := []any{}
		if actions, ok := s.templateActions[templateID]; ok {
			ids := make([]string, 0, len(actions))
			for id := range actions {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				items = append(items, mgnCloneMap(actions[id]))
			}
		}
		return map[string]any{"items": items, "nextToken": ""}
	case "ListExports":
		return map[string]any{"items": s.sortedValuesLocked(s.exports), "nextToken": ""}
	case "ListImports":
		return map[string]any{"items": s.sortedValuesLocked(s.imports), "nextToken": ""}
	case "ListExportErrors", "ListImportErrors":
		return map[string]any{"items": []any{}, "nextToken": ""}
	case "ListTagsForResource":
		arn := mgnFirstNonEmpty(mgnString(payload, "resourceArn"), mgnResourceARN("application", s.firstApplicationIDLocked()))
		tags := map[string]any{}
		if existing, ok := s.tags[arn]; ok {
			keys := make([]string, 0, len(existing))
			for key := range existing {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				tags[key] = existing[key]
			}
		}
		return map[string]any{"tags": tags}
	default:
		return map[string]any{}
	}
}

func (s *mgnStore) ensureSeedDataLocked() {
	if len(s.applications) == 0 {
		s.applications["app-00000001"] = map[string]any{
			"applicationID": "app-00000001",
			"accountID":     "123456789012",
			"name":          "stackyard-app",
			"isArchived":    false,
			"arn":           mgnResourceARN("application", "app-00000001"),
		}
	}
	if len(s.waves) == 0 {
		s.waves["wave-00000001"] = map[string]any{
			"waveID":     "wave-00000001",
			"accountID":  "123456789012",
			"name":       "stackyard-wave",
			"isArchived": false,
			"arn":        mgnResourceARN("wave", "wave-00000001"),
		}
	}
	if len(s.sourceServers) == 0 {
		s.sourceServers["s-00000001"] = map[string]any{
			"sourceServerID": "s-00000001",
			"accountID":      "123456789012",
			"lifeCycle":      map[string]any{"state": "READY_FOR_TEST"},
			"isArchived":     false,
			"arn":            mgnResourceARN("source-server", "s-00000001"),
		}
	}
}

func (s *mgnStore) ensureApplicationLocked(id string) map[string]any {
	if id == "" {
		id = s.firstApplicationIDLocked()
	}
	if item, ok := s.applications[id]; ok {
		return item
	}
	item := map[string]any{
		"applicationID": id,
		"accountID":     "123456789012",
		"name":          id,
		"isArchived":    false,
		"arn":           mgnResourceARN("application", id),
	}
	s.applications[id] = item
	return item
}

func (s *mgnStore) ensureWaveLocked(id string) map[string]any {
	if id == "" {
		id = s.firstWaveIDLocked()
	}
	if item, ok := s.waves[id]; ok {
		return item
	}
	item := map[string]any{
		"waveID":     id,
		"accountID":  "123456789012",
		"name":       id,
		"isArchived": false,
		"arn":        mgnResourceARN("wave", id),
	}
	s.waves[id] = item
	return item
}

func (s *mgnStore) ensureSourceServerLocked(id string) map[string]any {
	if id == "" {
		id = s.firstSourceServerIDLocked()
	}
	if item, ok := s.sourceServers[id]; ok {
		return item
	}
	item := map[string]any{
		"sourceServerID": id,
		"accountID":      "123456789012",
		"isArchived":     false,
		"lifeCycle":      map[string]any{"state": "READY_FOR_TEST"},
		"arn":            mgnResourceARN("source-server", id),
	}
	s.sourceServers[id] = item
	return item
}

func (s *mgnStore) ensureConnectorLocked(id string) map[string]any {
	if id == "" {
		id = s.firstConnectorIDLocked()
	}
	if item, ok := s.connectors[id]; ok {
		return item
	}
	item := map[string]any{"connectorID": id, "name": id, "arn": mgnResourceARN("connector", id)}
	s.connectors[id] = item
	return item
}

func (s *mgnStore) ensureLaunchTemplateLocked(id string) map[string]any {
	if id == "" {
		id = s.firstLaunchTemplateIDLocked()
	}
	if item, ok := s.launchConfigurationTemplates[id]; ok {
		return item
	}
	item := map[string]any{
		"launchConfigurationTemplateID": id,
		"copyTags":                      true,
		"launchDisposition":             "STOPPED",
		"arn":                           mgnResourceARN("launch-configuration-template", id),
	}
	s.launchConfigurationTemplates[id] = item
	return item
}

func (s *mgnStore) ensureReplicationTemplateLocked(id string) map[string]any {
	if id == "" {
		id = s.firstReplicationTemplateIDLocked()
	}
	if item, ok := s.replicationConfigurationTemps[id]; ok {
		return item
	}
	item := map[string]any{
		"replicationConfigurationTemplateID": id,
		"associateDefaultSecurityGroup":      true,
		"replicationServerInstanceType":      "t3.small",
		"arn":                                mgnResourceARN("replication-configuration-template", id),
	}
	s.replicationConfigurationTemps[id] = item
	return item
}

func (s *mgnStore) sortedValuesLocked(items map[string]map[string]any) []any {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, mgnCloneMap(items[key]))
	}
	return out
}

func (s *mgnStore) firstApplicationIDLocked() string {
	return mgnFirstSortedKey(s.applications)
}

func (s *mgnStore) firstWaveIDLocked() string {
	return mgnFirstSortedKey(s.waves)
}

func (s *mgnStore) firstConnectorIDLocked() string {
	return mgnFirstSortedKey(s.connectors)
}

func (s *mgnStore) firstLaunchTemplateIDLocked() string {
	return mgnFirstSortedKey(s.launchConfigurationTemplates)
}

func (s *mgnStore) firstReplicationTemplateIDLocked() string {
	return mgnFirstSortedKey(s.replicationConfigurationTemps)
}

func (s *mgnStore) firstSourceServerIDLocked() string {
	return mgnFirstSortedKey(s.sourceServers)
}

func (s *mgnStore) firstVcenterClientIDLocked() string {
	return mgnFirstSortedKey(s.vcenterClients)
}

func (s *mgnStore) firstJobIDLocked() string {
	return mgnFirstSortedKey(s.jobs)
}

func mgnFirstSortedKey[T any](items map[string]T) string {
	if len(items) == 0 {
		return ""
	}
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func mgnResourceARN(kind, id string) string {
	kind = strings.TrimSpace(kind)
	id = strings.TrimSpace(id)
	if kind == "" {
		kind = "resource"
	}
	if id == "" {
		id = "unknown"
	}
	return fmt.Sprintf("arn:aws:mgn:us-east-1:123456789012:%s/%s", kind, id)
}

func mgnString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func mgnStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		value := strings.TrimSpace(fmt.Sprint(item))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func mgnBool(payload map[string]any, key string, def bool) bool {
	if payload == nil {
		return def
	}
	return mgnBoolFromAny(payload[key], def)
}

func mgnBoolFromAny(raw any, def bool) bool {
	if raw == nil {
		return def
	}
	if v, ok := raw.(bool); ok {
		return v
	}
	if v, ok := raw.(string); ok {
		v = strings.TrimSpace(strings.ToLower(v))
		if v == "true" {
			return true
		}
		if v == "false" {
			return false
		}
	}
	return def
}

func mgnFirstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mgnCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = mgnCloneMap(typed)
		case []any:
			out[key] = mgnCloneSlice(typed)
		case map[string]string:
			cloned := make(map[string]any, len(typed))
			for nestedKey, nestedValue := range typed {
				cloned[nestedKey] = nestedValue
			}
			out[key] = cloned
		default:
			out[key] = typed
		}
	}
	return out
}

func mgnCloneSlice(in []any) []any {
	if in == nil {
		return nil
	}
	out := make([]any, 0, len(in))
	for _, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out = append(out, mgnCloneMap(typed))
		case []any:
			out = append(out, mgnCloneSlice(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}
