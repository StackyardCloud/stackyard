package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type drsStore struct {
	mu sync.Mutex

	initialized bool

	nextLaunchConfigurationTemplateID      int64
	nextReplicationConfigurationTemplateID int64
	nextSourceNetworkID                    int64
	nextSourceServerID                     int64
	nextRecoveryInstanceID                 int64
	nextJobID                              int64
	nextLaunchActionID                     int64

	launchConfigurationTemplates      map[string]map[string]any
	replicationConfigurationTemplates map[string]map[string]any
	sourceNetworks                    map[string]map[string]any
	sourceServers                     map[string]map[string]any
	recoveryInstances                 map[string]map[string]any
	jobs                              map[string]map[string]any
	launchActions                     map[string]map[string]map[string]any
	tags                              map[string]map[string]string
	stagingAccounts                   []map[string]any
}

func newDRSStore() *drsStore {
	now := time.Now().UTC().Format(time.RFC3339)

	launchTemplateID := "lct-00000001"
	replicationTemplateID := "rct-00000001"
	sourceNetworkID := "sn-00000001"
	sourceServerID := "s-00000001"
	recoveryInstanceID := "i-00000001"
	jobID := "job-00000001"
	launchActionID := "la-00000001"

	sourceServerARN := drsResourceARN("source-server", sourceServerID)

	s := &drsStore{
		initialized: true,

		nextLaunchConfigurationTemplateID:      2,
		nextReplicationConfigurationTemplateID: 2,
		nextSourceNetworkID:                    2,
		nextSourceServerID:                     2,
		nextRecoveryInstanceID:                 2,
		nextJobID:                              2,
		nextLaunchActionID:                     2,

		launchConfigurationTemplates:      map[string]map[string]any{},
		replicationConfigurationTemplates: map[string]map[string]any{},
		sourceNetworks:                    map[string]map[string]any{},
		sourceServers:                     map[string]map[string]any{},
		recoveryInstances:                 map[string]map[string]any{},
		jobs:                              map[string]map[string]any{},
		launchActions:                     map[string]map[string]map[string]any{},
		tags:                              map[string]map[string]string{},
		stagingAccounts: []map[string]any{
			{
				"accountID":        "123456789012",
				"stagingAccountID": "123456789012",
			},
		},
	}

	s.launchConfigurationTemplates[launchTemplateID] = map[string]any{
		"launchConfigurationTemplateID":       launchTemplateID,
		"arn":                                 drsResourceARN("launch-configuration-template", launchTemplateID),
		"copyPrivateIp":                       true,
		"launchDisposition":                   "STOPPED",
		"targetInstanceTypeRightSizingMethod": "NONE",
	}
	s.replicationConfigurationTemplates[replicationTemplateID] = map[string]any{
		"replicationConfigurationTemplateID": replicationTemplateID,
		"arn":                                drsResourceARN("replication-configuration-template", replicationTemplateID),
		"replicationServerInstanceType":      "t3.small",
		"useDedicatedReplicationServer":      false,
		"createPublicIP":                     false,
		"bandwidthThrottling":                0,
	}
	s.sourceNetworks[sourceNetworkID] = map[string]any{
		"sourceNetworkID": sourceNetworkID,
		"arn":             drsResourceARN("source-network", sourceNetworkID),
		"createdDateTime": now,
	}
	s.sourceServers[sourceServerID] = map[string]any{
		"sourceServerID": sourceServerID,
		"arn":            sourceServerARN,
		"isArchived":     false,
		"lifeCycle": map[string]any{
			"state": "READY_FOR_TEST",
		},
		"dataReplicationInfo": map[string]any{
			"dataReplicationState": "CONTINUOUS",
		},
		"lastLaunchResult": "NOT_STARTED",
	}
	s.recoveryInstances[recoveryInstanceID] = map[string]any{
		"recoveryInstanceID": recoveryInstanceID,
		"arn":                drsResourceARN("recovery-instance", recoveryInstanceID),
		"sourceServerID":     sourceServerID,
		"ec2InstanceID":      "i-00000000000000001",
		"dataReplicationInfo": map[string]any{
			"dataReplicationState": "CONTINUOUS",
		},
		"failback": map[string]any{
			"state": "NOT_STARTED",
		},
	}
	s.jobs[jobID] = map[string]any{
		"jobID":            jobID,
		"arn":              drsResourceARN("job", jobID),
		"type":             "START_RECOVERY",
		"status":           "COMPLETED",
		"creationDateTime": now,
		"endDateTime":      now,
	}
	s.launchActions[sourceServerID] = map[string]map[string]any{
		launchActionID: {
			"actionID":       launchActionID,
			"actionCode":     "AWSScript",
			"type":           "CUSTOM_ACTION",
			"active":         true,
			"sourceServerID": sourceServerID,
			"name":           "stackyard-launch-action",
		},
	}
	s.tags[sourceServerARN] = map[string]string{
		"seed": "true",
	}

	return s
}

func (s *drsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceServerID := drsFirstNonEmpty(
		drsString(payload, "sourceServerID", "sourceServerId", "SourceServerID", "SourceServerId"),
		s.firstSourceServerIDLocked(),
	)
	sourceNetworkID := drsFirstNonEmpty(
		drsString(payload, "sourceNetworkID", "sourceNetworkId", "SourceNetworkID", "SourceNetworkId"),
		s.firstSourceNetworkIDLocked(),
	)
	recoveryInstanceID := drsFirstNonEmpty(
		drsString(payload, "recoveryInstanceID", "recoveryInstanceId", "RecoveryInstanceID", "RecoveryInstanceId"),
		s.firstRecoveryInstanceIDLocked(),
	)
	resourceARN := drsFirstNonEmpty(
		drsString(payload, "resourceArn", "resourceARN", "ResourceArn", "ResourceARN", "arn", "Arn"),
		drsResourceARN("source-server", sourceServerID),
	)

	switch action {
	case "InitializeService":
		s.initialized = true
		return map[string]any{}

	case "CreateLaunchConfigurationTemplate":
		id := fmt.Sprintf("lct-%08d", s.nextLaunchConfigurationTemplateIDLocked())
		item := map[string]any{
			"launchConfigurationTemplateID":       id,
			"arn":                                 drsResourceARN("launch-configuration-template", id),
			"copyPrivateIp":                       drsBool(payload, "copyPrivateIp", true),
			"launchDisposition":                   drsFirstNonEmpty(drsString(payload, "launchDisposition"), "STOPPED"),
			"targetInstanceTypeRightSizingMethod": drsFirstNonEmpty(drsString(payload, "targetInstanceTypeRightSizingMethod"), "NONE"),
		}
		s.launchConfigurationTemplates[id] = item
		return drsCloneMap(item)

	case "DescribeLaunchConfigurationTemplates":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.launchConfigurationTemplates),
			"nextToken": "",
		}

	case "UpdateLaunchConfigurationTemplate":
		id := drsFirstNonEmpty(
			drsString(payload, "launchConfigurationTemplateID", "launchConfigurationTemplateId", "LaunchConfigurationTemplateID", "LaunchConfigurationTemplateId"),
			s.firstLaunchConfigurationTemplateIDLocked(),
		)
		item := s.ensureLaunchConfigurationTemplateLocked(id)
		for key, value := range payload {
			item[key] = value
		}
		return drsCloneMap(item)

	case "DeleteLaunchConfigurationTemplate":
		id := drsFirstNonEmpty(
			drsString(payload, "launchConfigurationTemplateID", "launchConfigurationTemplateId", "LaunchConfigurationTemplateID", "LaunchConfigurationTemplateId"),
			s.firstLaunchConfigurationTemplateIDLocked(),
		)
		delete(s.launchConfigurationTemplates, id)
		return map[string]any{}

	case "CreateReplicationConfigurationTemplate":
		id := fmt.Sprintf("rct-%08d", s.nextReplicationConfigurationTemplateIDLocked())
		item := map[string]any{
			"replicationConfigurationTemplateID": id,
			"arn":                                drsResourceARN("replication-configuration-template", id),
			"replicationServerInstanceType":      drsFirstNonEmpty(drsString(payload, "replicationServerInstanceType"), "t3.small"),
			"useDedicatedReplicationServer":      drsBool(payload, "useDedicatedReplicationServer", false),
			"createPublicIP":                     drsBool(payload, "createPublicIP", false),
			"bandwidthThrottling":                drsInt(payload, "bandwidthThrottling", 0),
		}
		s.replicationConfigurationTemplates[id] = item
		return drsCloneMap(item)

	case "DescribeReplicationConfigurationTemplates":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.replicationConfigurationTemplates),
			"nextToken": "",
		}

	case "UpdateReplicationConfigurationTemplate":
		id := drsFirstNonEmpty(
			drsString(payload, "replicationConfigurationTemplateID", "replicationConfigurationTemplateId", "ReplicationConfigurationTemplateID", "ReplicationConfigurationTemplateId"),
			s.firstReplicationConfigurationTemplateIDLocked(),
		)
		item := s.ensureReplicationConfigurationTemplateLocked(id)
		for key, value := range payload {
			item[key] = value
		}
		return drsCloneMap(item)

	case "DeleteReplicationConfigurationTemplate":
		id := drsFirstNonEmpty(
			drsString(payload, "replicationConfigurationTemplateID", "replicationConfigurationTemplateId", "ReplicationConfigurationTemplateID", "ReplicationConfigurationTemplateId"),
			s.firstReplicationConfigurationTemplateIDLocked(),
		)
		delete(s.replicationConfigurationTemplates, id)
		return map[string]any{}

	case "CreateSourceNetwork":
		id := fmt.Sprintf("sn-%08d", s.nextSourceNetworkIDLocked())
		item := map[string]any{
			"sourceNetworkID": id,
			"arn":             drsResourceARN("source-network", id),
			"createdDateTime": time.Now().UTC().Format(time.RFC3339),
		}
		s.sourceNetworks[id] = item
		return drsCloneMap(item)

	case "DescribeSourceNetworks":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.sourceNetworks),
			"nextToken": "",
		}

	case "DeleteSourceNetwork":
		delete(s.sourceNetworks, sourceNetworkID)
		return map[string]any{}

	case "AssociateSourceNetworkStack":
		network := s.ensureSourceNetworkLocked(sourceNetworkID)
		network["stackName"] = drsFirstNonEmpty(drsString(payload, "cfnStackName", "stackName", "CfnStackName"), "stackyard-source-network-stack")
		return map[string]any{
			"sourceNetworkID": sourceNetworkID,
		}

	case "ExportSourceNetworkCfnTemplate":
		_ = s.ensureSourceNetworkLocked(sourceNetworkID)
		return map[string]any{
			"sourceNetworkID":  sourceNetworkID,
			"s3DestinationUrl": "s3://stackyard-drs/source-network-template.yaml",
		}

	case "StartSourceNetworkReplication", "StopSourceNetworkReplication", "StartSourceNetworkRecovery":
		_ = s.ensureSourceNetworkLocked(sourceNetworkID)
		job := s.createJobLocked(action)
		return map[string]any{"job": drsCloneMap(job)}

	case "CreateExtendedSourceServer":
		id := fmt.Sprintf("s-%08d", s.nextSourceServerIDLocked())
		item := map[string]any{
			"sourceServerID": id,
			"arn":            drsResourceARN("source-server", id),
			"isArchived":     false,
			"lifeCycle": map[string]any{
				"state": "READY_FOR_TEST",
			},
			"dataReplicationInfo": map[string]any{
				"dataReplicationState": "CONTINUOUS",
			},
			"lastLaunchResult": "NOT_STARTED",
		}
		s.sourceServers[id] = item
		return drsCloneMap(item)

	case "DescribeSourceServers":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.sourceServers),
			"nextToken": "",
		}

	case "DeleteSourceServer":
		delete(s.sourceServers, sourceServerID)
		return map[string]any{}

	case "DisconnectSourceServer":
		server := s.ensureSourceServerLocked(sourceServerID)
		server["isArchived"] = true
		return map[string]any{}

	case "GetLaunchConfiguration":
		_ = s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID":                      sourceServerID,
			"copyPrivateIp":                       true,
			"launchDisposition":                   "STOPPED",
			"targetInstanceTypeRightSizingMethod": "NONE",
		}

	case "UpdateLaunchConfiguration":
		_ = s.ensureSourceServerLocked(sourceServerID)
		out := map[string]any{
			"sourceServerID":                      sourceServerID,
			"copyPrivateIp":                       drsBool(payload, "copyPrivateIp", true),
			"launchDisposition":                   drsFirstNonEmpty(drsString(payload, "launchDisposition"), "STOPPED"),
			"targetInstanceTypeRightSizingMethod": drsFirstNonEmpty(drsString(payload, "targetInstanceTypeRightSizingMethod"), "NONE"),
		}
		return out

	case "GetReplicationConfiguration":
		_ = s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID":                sourceServerID,
			"replicationServerInstanceType": "t3.small",
			"useDedicatedReplicationServer": false,
			"createPublicIP":                false,
		}

	case "UpdateReplicationConfiguration":
		_ = s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID":                sourceServerID,
			"replicationServerInstanceType": drsFirstNonEmpty(drsString(payload, "replicationServerInstanceType"), "t3.small"),
			"useDedicatedReplicationServer": drsBool(payload, "useDedicatedReplicationServer", false),
			"createPublicIP":                drsBool(payload, "createPublicIP", false),
		}

	case "GetFailbackReplicationConfiguration":
		_ = s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID": sourceServerID,
			"name":           "stackyard-failback",
		}

	case "UpdateFailbackReplicationConfiguration":
		_ = s.ensureSourceServerLocked(sourceServerID)
		return map[string]any{
			"sourceServerID": sourceServerID,
			"name":           drsFirstNonEmpty(drsString(payload, "name"), "stackyard-failback"),
		}

	case "ListExtensibleSourceServers":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.sourceServers),
			"nextToken": "",
		}

	case "StartReplication", "StopReplication", "RetryDataReplication", "ReverseReplication", "StopFailback":
		server := s.ensureSourceServerLocked(sourceServerID)
		switch action {
		case "StartReplication":
			server["dataReplicationInfo"] = map[string]any{"dataReplicationState": "CONTINUOUS"}
		case "StopReplication":
			server["dataReplicationInfo"] = map[string]any{"dataReplicationState": "PAUSED"}
		}
		job := s.createJobLocked(action)
		return map[string]any{"job": drsCloneMap(job)}

	case "StartRecovery", "StartFailbackLaunch":
		serverIDs := drsStringSlice(payload, "sourceServerIDs", "sourceServerIds", "SourceServerIDs", "SourceServerIds")
		if len(serverIDs) == 0 {
			serverIDs = []string{s.firstSourceServerIDLocked()}
		}
		for _, id := range serverIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			_ = s.ensureSourceServerLocked(id)
			recoveryID := fmt.Sprintf("i-%08d", s.nextRecoveryInstanceIDLocked())
			s.recoveryInstances[recoveryID] = map[string]any{
				"recoveryInstanceID": recoveryID,
				"arn":                drsResourceARN("recovery-instance", recoveryID),
				"sourceServerID":     id,
				"ec2InstanceID":      fmt.Sprintf("i-%017d", s.nextRecoveryInstanceIDLocked()+1000),
				"dataReplicationInfo": map[string]any{
					"dataReplicationState": "CONTINUOUS",
				},
				"failback": map[string]any{
					"state": "NOT_STARTED",
				},
			}
		}
		job := s.createJobLocked(action)
		return map[string]any{"job": drsCloneMap(job)}

	case "TerminateRecoveryInstances":
		instanceIDs := drsStringSlice(payload, "recoveryInstanceIDs", "recoveryInstanceIds", "RecoveryInstanceIDs", "RecoveryInstanceIds")
		if len(instanceIDs) == 0 {
			instanceIDs = []string{recoveryInstanceID}
		}
		for _, id := range instanceIDs {
			delete(s.recoveryInstances, strings.TrimSpace(id))
		}
		job := s.createJobLocked(action)
		return map[string]any{"job": drsCloneMap(job)}

	case "DescribeRecoveryInstances":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.recoveryInstances),
			"nextToken": "",
		}

	case "DeleteRecoveryInstance":
		delete(s.recoveryInstances, recoveryInstanceID)
		return map[string]any{}

	case "DisconnectRecoveryInstance":
		instance := s.ensureRecoveryInstanceLocked(recoveryInstanceID)
		instance["isDisconnected"] = true
		return map[string]any{}

	case "DescribeRecoverySnapshots":
		return map[string]any{
			"items": []any{
				map[string]any{
					"recoverySnapshotID": "rs-00000001",
					"sourceServerID":     sourceServerID,
					"expectedTimestamp":  time.Now().UTC().Format(time.RFC3339),
				},
			},
			"nextToken": "",
		}

	case "DescribeJobs":
		return map[string]any{
			"items":     s.sortedMapValuesLocked(s.jobs),
			"nextToken": "",
		}

	case "DescribeJobLogItems":
		jobID := drsFirstNonEmpty(
			drsString(payload, "jobID", "jobId", "JobID", "JobId"),
			s.firstJobIDLocked(),
		)
		_ = s.ensureJobLocked(jobID)
		return map[string]any{
			"items": []any{
				map[string]any{
					"event":            "DRS_JOB_LOG",
					"eventData":        map[string]any{"message": "Stackyard simulated job log"},
					"logDateTime":      time.Now().UTC().Format(time.RFC3339),
					"sourceServerID":   sourceServerID,
					"targetInstanceID": recoveryInstanceID,
				},
			},
			"nextToken": "",
		}

	case "DeleteJob":
		jobID := drsFirstNonEmpty(
			drsString(payload, "jobID", "jobId", "JobID", "JobId"),
			s.firstJobIDLocked(),
		)
		delete(s.jobs, jobID)
		return map[string]any{}

	case "PutLaunchAction":
		id := drsFirstNonEmpty(
			drsString(payload, "actionID", "actionId", "ActionID", "ActionId"),
			fmt.Sprintf("la-%08d", s.nextLaunchActionIDLocked()),
		)
		sourceServerID = drsFirstNonEmpty(sourceServerID, s.firstSourceServerIDLocked())
		if s.launchActions[sourceServerID] == nil {
			s.launchActions[sourceServerID] = map[string]map[string]any{}
		}
		item := map[string]any{
			"actionID":       id,
			"actionCode":     drsFirstNonEmpty(drsString(payload, "actionCode", "ActionCode"), "AWSScript"),
			"type":           drsFirstNonEmpty(drsString(payload, "type", "Type"), "CUSTOM_ACTION"),
			"active":         drsBool(payload, "active", true),
			"name":           drsFirstNonEmpty(drsString(payload, "name", "Name"), id),
			"sourceServerID": sourceServerID,
			"parameters":     drsValue(payload, "parameters", "Parameters"),
		}
		s.launchActions[sourceServerID][id] = item
		return drsCloneMap(item)

	case "ListLaunchActions":
		actions := s.ensureLaunchActionsForSourceServerLocked(sourceServerID)
		return map[string]any{
			"items":     s.sortedMapValuesLocked(actions),
			"nextToken": "",
		}

	case "DeleteLaunchAction":
		actionID := drsFirstNonEmpty(
			drsString(payload, "actionID", "actionId", "ActionID", "ActionId"),
			s.firstLaunchActionIDLocked(sourceServerID),
		)
		if actions, ok := s.launchActions[sourceServerID]; ok {
			delete(actions, actionID)
		}
		return map[string]any{}

	case "ListStagingAccounts":
		items := make([]any, 0, len(s.stagingAccounts))
		for _, item := range s.stagingAccounts {
			items = append(items, drsCloneMap(item))
		}
		return map[string]any{
			"accounts":  items,
			"nextToken": "",
		}

	case "TagResource":
		resourceARN = drsFirstNonEmpty(resourceARN, drsResourceARN("source-server", sourceServerID))
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		rawTags := drsValue(payload, "tags", "Tags")
		for key, value := range drsTagsFromAny(rawTags) {
			s.tags[resourceARN][key] = value
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN = drsFirstNonEmpty(resourceARN, drsResourceARN("source-server", sourceServerID))
		return map[string]any{
			"tags": drsCloneStringMap(s.tags[resourceARN]),
		}

	case "UntagResource":
		resourceARN = drsFirstNonEmpty(resourceARN, drsResourceARN("source-server", sourceServerID))
		keys := drsStringSlice(payload, "tagKeys", "TagKeys")
		if len(keys) == 0 {
			keys = drsStringSlice(payload, "tagkeys")
		}
		for _, key := range keys {
			delete(s.tags[resourceARN], strings.TrimSpace(key))
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *drsStore) ensureLaunchConfigurationTemplateLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstLaunchConfigurationTemplateIDLocked()
	}
	if item, ok := s.launchConfigurationTemplates[id]; ok {
		return item
	}
	item := map[string]any{
		"launchConfigurationTemplateID":       id,
		"arn":                                 drsResourceARN("launch-configuration-template", id),
		"copyPrivateIp":                       true,
		"launchDisposition":                   "STOPPED",
		"targetInstanceTypeRightSizingMethod": "NONE",
	}
	s.launchConfigurationTemplates[id] = item
	return item
}

func (s *drsStore) ensureReplicationConfigurationTemplateLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstReplicationConfigurationTemplateIDLocked()
	}
	if item, ok := s.replicationConfigurationTemplates[id]; ok {
		return item
	}
	item := map[string]any{
		"replicationConfigurationTemplateID": id,
		"arn":                                drsResourceARN("replication-configuration-template", id),
		"replicationServerInstanceType":      "t3.small",
		"useDedicatedReplicationServer":      false,
		"createPublicIP":                     false,
	}
	s.replicationConfigurationTemplates[id] = item
	return item
}

func (s *drsStore) ensureSourceNetworkLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstSourceNetworkIDLocked()
	}
	if item, ok := s.sourceNetworks[id]; ok {
		return item
	}
	item := map[string]any{
		"sourceNetworkID": id,
		"arn":             drsResourceARN("source-network", id),
		"createdDateTime": time.Now().UTC().Format(time.RFC3339),
	}
	s.sourceNetworks[id] = item
	return item
}

func (s *drsStore) ensureSourceServerLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstSourceServerIDLocked()
	}
	if item, ok := s.sourceServers[id]; ok {
		return item
	}
	item := map[string]any{
		"sourceServerID": id,
		"arn":            drsResourceARN("source-server", id),
		"isArchived":     false,
		"lifeCycle": map[string]any{
			"state": "READY_FOR_TEST",
		},
		"dataReplicationInfo": map[string]any{
			"dataReplicationState": "CONTINUOUS",
		},
		"lastLaunchResult": "NOT_STARTED",
	}
	s.sourceServers[id] = item
	return item
}

func (s *drsStore) ensureRecoveryInstanceLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstRecoveryInstanceIDLocked()
	}
	if item, ok := s.recoveryInstances[id]; ok {
		return item
	}
	item := map[string]any{
		"recoveryInstanceID": id,
		"arn":                drsResourceARN("recovery-instance", id),
		"sourceServerID":     s.firstSourceServerIDLocked(),
		"ec2InstanceID":      "i-00000000000000001",
		"dataReplicationInfo": map[string]any{
			"dataReplicationState": "CONTINUOUS",
		},
	}
	s.recoveryInstances[id] = item
	return item
}

func (s *drsStore) ensureJobLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstJobIDLocked()
	}
	if item, ok := s.jobs[id]; ok {
		return item
	}
	item := map[string]any{
		"jobID":            id,
		"arn":              drsResourceARN("job", id),
		"type":             "UNKNOWN",
		"status":           "COMPLETED",
		"creationDateTime": time.Now().UTC().Format(time.RFC3339),
		"endDateTime":      time.Now().UTC().Format(time.RFC3339),
	}
	s.jobs[id] = item
	return item
}

func (s *drsStore) ensureLaunchActionsForSourceServerLocked(sourceServerID string) map[string]map[string]any {
	sourceServerID = strings.TrimSpace(sourceServerID)
	if sourceServerID == "" {
		sourceServerID = s.firstSourceServerIDLocked()
	}
	if s.launchActions[sourceServerID] == nil {
		s.launchActions[sourceServerID] = map[string]map[string]any{}
	}
	return s.launchActions[sourceServerID]
}

func (s *drsStore) firstLaunchConfigurationTemplateIDLocked() string {
	if len(s.launchConfigurationTemplates) == 0 {
		return "lct-00000001"
	}
	return drsFirstSortedKey(s.launchConfigurationTemplates)
}

func (s *drsStore) firstReplicationConfigurationTemplateIDLocked() string {
	if len(s.replicationConfigurationTemplates) == 0 {
		return "rct-00000001"
	}
	return drsFirstSortedKey(s.replicationConfigurationTemplates)
}

func (s *drsStore) firstSourceNetworkIDLocked() string {
	if len(s.sourceNetworks) == 0 {
		return "sn-00000001"
	}
	return drsFirstSortedKey(s.sourceNetworks)
}

func (s *drsStore) firstSourceServerIDLocked() string {
	if len(s.sourceServers) == 0 {
		return "s-00000001"
	}
	return drsFirstSortedKey(s.sourceServers)
}

func (s *drsStore) firstRecoveryInstanceIDLocked() string {
	if len(s.recoveryInstances) == 0 {
		return "i-00000001"
	}
	return drsFirstSortedKey(s.recoveryInstances)
}

func (s *drsStore) firstJobIDLocked() string {
	if len(s.jobs) == 0 {
		return "job-00000001"
	}
	return drsFirstSortedKey(s.jobs)
}

func (s *drsStore) firstLaunchActionIDLocked(sourceServerID string) string {
	actions := s.ensureLaunchActionsForSourceServerLocked(sourceServerID)
	if len(actions) == 0 {
		return "la-00000001"
	}
	return drsFirstSortedKey(actions)
}

func (s *drsStore) nextLaunchConfigurationTemplateIDLocked() int64 {
	id := s.nextLaunchConfigurationTemplateID
	s.nextLaunchConfigurationTemplateID++
	return id
}

func (s *drsStore) nextReplicationConfigurationTemplateIDLocked() int64 {
	id := s.nextReplicationConfigurationTemplateID
	s.nextReplicationConfigurationTemplateID++
	return id
}

func (s *drsStore) nextSourceNetworkIDLocked() int64 {
	id := s.nextSourceNetworkID
	s.nextSourceNetworkID++
	return id
}

func (s *drsStore) nextSourceServerIDLocked() int64 {
	id := s.nextSourceServerID
	s.nextSourceServerID++
	return id
}

func (s *drsStore) nextRecoveryInstanceIDLocked() int64 {
	id := s.nextRecoveryInstanceID
	s.nextRecoveryInstanceID++
	return id
}

func (s *drsStore) nextJobIDLocked() int64 {
	id := s.nextJobID
	s.nextJobID++
	return id
}

func (s *drsStore) nextLaunchActionIDLocked() int64 {
	id := s.nextLaunchActionID
	s.nextLaunchActionID++
	return id
}

func (s *drsStore) createJobLocked(action string) map[string]any {
	id := fmt.Sprintf("job-%08d", s.nextJobIDLocked())
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"jobID":            id,
		"arn":              drsResourceARN("job", id),
		"type":             strings.ToUpper(action),
		"status":           "COMPLETED",
		"creationDateTime": now,
		"endDateTime":      now,
	}
	s.jobs[id] = item
	return item
}

func (s *drsStore) sortedMapValuesLocked(m map[string]map[string]any) []any {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, drsCloneMap(m[key]))
	}
	return out
}

func drsFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func drsValue(payload map[string]any, keys ...string) any {
	if payload == nil {
		return nil
	}
	for _, expected := range keys {
		for key, value := range payload {
			if strings.EqualFold(strings.TrimSpace(key), strings.TrimSpace(expected)) {
				return value
			}
		}
	}
	return nil
}

func drsString(payload map[string]any, keys ...string) string {
	value := drsValue(payload, keys...)
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func drsInt(payload map[string]any, key string, def int64) int64 {
	value := drsValue(payload, key)
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		return def
	}
}

func drsBool(payload map[string]any, key string, def bool) bool {
	value := drsValue(payload, key)
	switch v := value.(type) {
	case bool:
		return v
	default:
		return def
	}
}

func drsStringSlice(payload map[string]any, keys ...string) []string {
	raw := drsValue(payload, keys...)
	switch v := raw.(type) {
	case string:
		value := strings.TrimSpace(v)
		if value == "" {
			return nil
		}
		if strings.Contains(value, ",") {
			parts := strings.Split(value, ",")
			out := make([]string, 0, len(parts))
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" {
					out = append(out, part)
				}
			}
			return out
		}
		return []string{value}
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
			if str, ok := item.(string); ok {
				str = strings.TrimSpace(str)
				if str != "" {
					out = append(out, str)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func drsTagsFromAny(raw any) map[string]string {
	switch v := raw.(type) {
	case map[string]string:
		out := make(map[string]string, len(v))
		for key, value := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = value
		}
		return out
	case map[string]any:
		out := make(map[string]string, len(v))
		for key, value := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			switch typed := value.(type) {
			case string:
				out[key] = typed
			default:
				out[key] = fmt.Sprintf("%v", typed)
			}
		}
		return out
	case []any:
		out := map[string]string{}
		for _, item := range v {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := drsFirstNonEmpty(drsString(entry, "Key", "key"), drsString(entry, "TagKey", "tagKey"))
			if key == "" {
				continue
			}
			out[key] = drsFirstNonEmpty(drsString(entry, "Value", "value"), drsString(entry, "TagValue", "tagValue"))
		}
		return out
	default:
		return nil
	}
}

func drsFirstSortedKey[T any](m map[string]T) string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func drsCloneMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for key, value := range m {
		out[key] = drsCloneAny(value)
	}
	return out
}

func drsCloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for key, value := range m {
		out[key] = value
	}
	return out
}

func drsCloneAny(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return drsCloneMap(typed)
	case map[string]string:
		return drsCloneStringMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, drsCloneAny(item))
		}
		return out
	default:
		return typed
	}
}

func drsResourceARN(resourceType, resourceID string) string {
	resourceType = strings.Trim(strings.TrimSpace(resourceType), "/")
	resourceID = strings.Trim(strings.TrimSpace(resourceID), "/")
	return fmt.Sprintf("arn:aws:drs:us-east-1:123456789012:%s/%s", resourceType, resourceID)
}
