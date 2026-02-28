package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type discoveryStore struct {
	mu sync.Mutex

	nextID int64

	applications      map[string]*discoveryApplication
	applicationConfig map[string]map[string]struct{}
	agents            map[string]*discoveryAgent
	configurations    map[string]map[string]any
	tags              map[string]map[string]string

	continuousExports map[string]*discoveryContinuousExport
	exportTasks       map[string]*discoveryExportTask
	importTasks       map[string]*discoveryImportTask
	batchDeleteTasks  map[string]*discoveryBatchDeleteTask
}

type discoveryApplication struct {
	ConfigurationID string
	Name            string
	Description     string
	CreatedAt       string
	UpdatedAt       string
}

type discoveryAgent struct {
	AgentID             string
	HostName            string
	AgentType           string
	AgentVersion        string
	ConfigurationStatus string
	Health              string
	LastHealthPingTime  string
}

type discoveryContinuousExport struct {
	ExportID         string
	Status           string
	StartTime        string
	StopTime         string
	DataSource       string
	SchemaStorageCfg string
}

type discoveryExportTask struct {
	ExportID        string
	Status          string
	RequestedAt     string
	PreferredFormat string
	S3Bucket        string
}

type discoveryImportTask struct {
	ImportTaskID string
	Name         string
	Status       string
	ImportURL    string
	CreatedAt    string
}

type discoveryBatchDeleteTask struct {
	TaskID           string
	Status           string
	StartTime        string
	EndTime          string
	DeletionWarnings []map[string]any
}

func newDiscoveryStore() *discoveryStore {
	now := time.Now().UTC().Format(time.RFC3339)
	appID := "app-000001"

	app := &discoveryApplication{
		ConfigurationID: appID,
		Name:            "stackyard-seed-application",
		Description:     "Stackyard seeded discovery application",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	agent1 := &discoveryAgent{
		AgentID:             "agent-000001",
		HostName:            "stackyard-host-1",
		AgentType:           "AGENT",
		AgentVersion:        "3.2.1",
		ConfigurationStatus: "RUNNING",
		Health:              "HEALTHY",
		LastHealthPingTime:  now,
	}
	agent2 := &discoveryAgent{
		AgentID:             "agent-000002",
		HostName:            "stackyard-host-2",
		AgentType:           "AGENTLESS_COLLECTOR",
		AgentVersion:        "3.2.1",
		ConfigurationStatus: "STOPPED",
		Health:              "UNKNOWN",
		LastHealthPingTime:  now,
	}

	configurations := map[string]map[string]any{
		"srv-000001": {
			"configurationId":   "srv-000001",
			"configurationType": "SERVER",
			"serverName":        "stackyard-srv-1",
			"hostName":          "stackyard-host-1",
			"osName":            "Linux",
		},
		"srv-000002": {
			"configurationId":   "srv-000002",
			"configurationType": "SERVER",
			"serverName":        "stackyard-srv-2",
			"hostName":          "stackyard-host-2",
			"osName":            "Linux",
		},
		appID: {
			"configurationId":   appID,
			"configurationType": "APPLICATION",
			"name":              app.Name,
			"description":       app.Description,
		},
	}

	return &discoveryStore{
		nextID: 2,
		applications: map[string]*discoveryApplication{
			app.ConfigurationID: app,
		},
		applicationConfig: map[string]map[string]struct{}{
			app.ConfigurationID: {"srv-000001": {}},
		},
		agents: map[string]*discoveryAgent{
			agent1.AgentID: agent1,
			agent2.AgentID: agent2,
		},
		configurations: configurations,
		tags: map[string]map[string]string{
			"srv-000001": {"env": "dev", "stackyard": "true"},
			"srv-000002": {"env": "test"},
			appID:        {"app": "seed"},
		},
		continuousExports: map[string]*discoveryContinuousExport{
			"ce-000001": {
				ExportID:         "ce-000001",
				Status:           "STOPPED",
				StartTime:        now,
				StopTime:         now,
				DataSource:       "AGENT",
				SchemaStorageCfg: "S3",
			},
		},
		exportTasks:      map[string]*discoveryExportTask{},
		importTasks:      map[string]*discoveryImportTask{},
		batchDeleteTasks: map[string]*discoveryBatchDeleteTask{},
	}
}

func (s *discoveryStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateApplication":
		name := discoveryPayloadString(payload, "name", "")
		if name == "" {
			name = s.nextTokenLocked("application")
		}
		desc := discoveryPayloadString(payload, "description", "")
		id := discoveryPayloadString(payload, "configurationId", s.nextTokenLocked("app"))

		app := &discoveryApplication{
			ConfigurationID: id,
			Name:            name,
			Description:     desc,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		s.applications[id] = app
		s.configurations[id] = map[string]any{
			"configurationId":   id,
			"configurationType": "APPLICATION",
			"name":              name,
			"description":       desc,
		}
		s.ensureConfigTagsLocked(id)
		return map[string]any{"configurationId": id}

	case "UpdateApplication":
		id := discoveryPayloadString(payload, "configurationId", s.firstApplicationIDLocked())
		app := s.ensureApplicationLocked(id, now)
		if name := discoveryPayloadString(payload, "name", ""); name != "" {
			app.Name = name
		}
		if desc := discoveryPayloadString(payload, "description", ""); desc != "" {
			app.Description = desc
		}
		app.UpdatedAt = now
		s.configurations[id] = map[string]any{
			"configurationId":   id,
			"configurationType": "APPLICATION",
			"name":              app.Name,
			"description":       app.Description,
		}
		return map[string]any{}

	case "DeleteApplications":
		ids := discoveryPayloadStringSlice(payload, "configurationIds")
		if len(ids) == 0 {
			ids = []string{s.firstApplicationIDLocked()}
		}
		for _, id := range ids {
			delete(s.applications, id)
			delete(s.applicationConfig, id)
			delete(s.configurations, id)
			delete(s.tags, id)
		}
		return map[string]any{}

	case "AssociateConfigurationItemsToApplication":
		appID := discoveryPayloadString(payload, "applicationConfigurationId", s.firstApplicationIDLocked())
		cfgIDs := discoveryPayloadStringSlice(payload, "configurationIds")
		if len(cfgIDs) == 0 {
			cfgIDs = []string{"srv-000001"}
		}
		app := s.ensureApplicationLocked(appID, now)
		_ = app
		set := s.applicationConfig[appID]
		if set == nil {
			set = map[string]struct{}{}
			s.applicationConfig[appID] = set
		}
		for _, cfgID := range cfgIDs {
			cfgID = strings.TrimSpace(cfgID)
			if cfgID == "" {
				continue
			}
			set[cfgID] = struct{}{}
			s.ensureConfigLocked(cfgID)
		}
		return map[string]any{}

	case "DisassociateConfigurationItemsFromApplication":
		appID := discoveryPayloadString(payload, "applicationConfigurationId", s.firstApplicationIDLocked())
		cfgIDs := discoveryPayloadStringSlice(payload, "configurationIds")
		if len(cfgIDs) == 0 {
			cfgIDs = []string{"srv-000001"}
		}
		set := s.applicationConfig[appID]
		for _, cfgID := range cfgIDs {
			delete(set, strings.TrimSpace(cfgID))
		}
		return map[string]any{}

	case "CreateTags":
		cfgIDs := discoveryPayloadStringSlice(payload, "configurationIds")
		if len(cfgIDs) == 0 {
			cfgIDs = []string{"srv-000001"}
		}
		tags := discoveryPayloadTags(payload, "tags")
		if len(tags) == 0 {
			tags = []discoveryKV{{Key: "coverage", Value: "true"}}
		}
		for _, cfgID := range cfgIDs {
			tagMap := s.ensureConfigTagsLocked(cfgID)
			for _, tag := range tags {
				if tag.Key == "" {
					continue
				}
				tagMap[tag.Key] = tag.Value
			}
		}
		return map[string]any{}

	case "DeleteTags":
		cfgIDs := discoveryPayloadStringSlice(payload, "configurationIds")
		if len(cfgIDs) == 0 {
			cfgIDs = []string{"srv-000001"}
		}
		keys := discoveryPayloadStringSlice(payload, "tags")
		if len(keys) == 0 {
			keys = []string{"coverage"}
		}
		for _, cfgID := range cfgIDs {
			tagMap := s.ensureConfigTagsLocked(cfgID)
			for _, key := range keys {
				delete(tagMap, key)
			}
		}
		return map[string]any{}

	case "DescribeTags":
		items := make([]any, 0)
		cfgIDs := s.sortedConfigurationIDsLocked()
		for _, cfgID := range cfgIDs {
			tagMap := s.tags[cfgID]
			if len(tagMap) == 0 {
				continue
			}
			keys := make([]string, 0, len(tagMap))
			for k := range tagMap {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			cfg := s.ensureConfigLocked(cfgID)
			cfgType := discoveryAnyToString(cfg["configurationType"], "SERVER")
			for _, key := range keys {
				items = append(items, map[string]any{
					"configurationType": cfgType,
					"configurationId":   cfgID,
					"key":               key,
					"value":             tagMap[key],
					"timeOfCreation":    now,
				})
			}
		}
		return map[string]any{"tags": items}

	case "DescribeAgents":
		items := make([]any, 0, len(s.agents))
		for _, a := range s.sortedAgentsLocked() {
			items = append(items, map[string]any{
				"agentId":               a.AgentID,
				"hostName":              a.HostName,
				"agentType":             a.AgentType,
				"version":               a.AgentVersion,
				"agentHealth":           a.Health,
				"collectionStatus":      a.ConfigurationStatus,
				"lastHealthPingTime":    a.LastHealthPingTime,
				"registeredTime":        a.LastHealthPingTime,
				"agentNetworkInfoList":  []any{},
				"connectorId":           "",
				"connectorProgressInfo": "",
			})
		}
		return map[string]any{"agentsInfo": items}

	case "StartDataCollectionByAgentIds":
		return s.updateAgentsCollectionStatusLocked(payload, "RUNNING", "HEALTHY")

	case "StopDataCollectionByAgentIds":
		return s.updateAgentsCollectionStatusLocked(payload, "STOPPED", "UNKNOWN")

	case "BatchDeleteAgents":
		ids := discoveryDeleteAgentIDs(payload)
		if len(ids) == 0 {
			ids = []string{"agent-000001"}
		}
		errors := make([]any, 0)
		for _, id := range ids {
			if _, ok := s.agents[id]; !ok {
				errors = append(errors, map[string]any{
					"agentId":      id,
					"errorMessage": "agent not found",
					"errorCode":    "NOT_FOUND",
				})
				continue
			}
			delete(s.agents, id)
		}
		return map[string]any{"errors": errors}

	case "StartContinuousExport":
		exportID := discoveryPayloadString(payload, "continuousExportId", "ce-000001")
		ce := s.ensureContinuousExportLocked(exportID, now)
		ce.Status = "STARTED"
		ce.StartTime = now
		ce.StopTime = ""
		return map[string]any{"continuousExportId": ce.ExportID}

	case "StopContinuousExport":
		exportID := discoveryPayloadString(payload, "continuousExportId", "ce-000001")
		ce := s.ensureContinuousExportLocked(exportID, now)
		ce.Status = "STOPPED"
		ce.StopTime = now
		return map[string]any{}

	case "DescribeContinuousExports":
		items := make([]any, 0, len(s.continuousExports))
		for _, ce := range s.sortedContinuousExportsLocked() {
			items = append(items, map[string]any{
				"exportId":                    ce.ExportID,
				"status":                      ce.Status,
				"startTime":                   ce.StartTime,
				"stopTime":                    ce.StopTime,
				"dataSource":                  ce.DataSource,
				"schemaStorageConfig":         ce.SchemaStorageCfg,
				"isTruncated":                 false,
				"statusDetail":                "deterministic local emulation",
				"lastStatusChangeDate":        ce.StartTime,
				"continuousExportDescription": "stackyard discovery export",
			})
		}
		return map[string]any{"descriptions": items}

	case "StartExportTask":
		task := s.createExportTaskLocked(now, discoveryPayloadString(payload, "exportDataFormat", "CSV"))
		return map[string]any{"exportId": task.ExportID}

	case "ExportConfigurations":
		task := s.createExportTaskLocked(now, discoveryPayloadString(payload, "exportDataFormat", "CSV"))
		return map[string]any{"exportId": task.ExportID}

	case "DescribeExportTasks":
		items := make([]any, 0, len(s.exportTasks))
		for _, task := range s.sortedExportTasksLocked() {
			items = append(items, map[string]any{
				"exportId":           task.ExportID,
				"status":             task.Status,
				"requestedStartTime": task.RequestedAt,
				"exportDataFormat":   task.PreferredFormat,
				"s3Bucket":           task.S3Bucket,
				"s3bucket":           task.S3Bucket,
			})
		}
		return map[string]any{"exportsInfo": items}

	case "DescribeExportConfigurations":
		items := make([]any, 0, len(s.exportTasks))
		for _, task := range s.sortedExportTasksLocked() {
			items = append(items, map[string]any{
				"exportId":         task.ExportID,
				"status":           task.Status,
				"exportDataFormat": task.PreferredFormat,
				"s3Bucket":         task.S3Bucket,
			})
		}
		return map[string]any{"exportsInfo": items}

	case "StartImportTask":
		id := discoveryPayloadString(payload, "clientRequestToken", "")
		if id == "" {
			id = s.nextTokenLocked("import")
		}
		task := &discoveryImportTask{
			ImportTaskID: id,
			Name:         discoveryPayloadString(payload, "name", "stackyard-import-task"),
			Status:       "IMPORT_COMPLETE",
			ImportURL:    discoveryPayloadString(payload, "importUrl", "s3://stackyard/import.csv"),
			CreatedAt:    now,
		}
		s.importTasks[task.ImportTaskID] = task
		return map[string]any{
			"task": map[string]any{
				"importTaskId": task.ImportTaskID,
				"name":         task.Name,
				"status":       task.Status,
				"importUrl":    task.ImportURL,
			},
		}

	case "DescribeImportTasks":
		items := make([]any, 0, len(s.importTasks))
		for _, task := range s.sortedImportTasksLocked() {
			items = append(items, map[string]any{
				"importTaskId": task.ImportTaskID,
				"name":         task.Name,
				"status":       task.Status,
				"importUrl":    task.ImportURL,
			})
		}
		return map[string]any{"tasks": items}

	case "BatchDeleteImportData":
		ids := discoveryPayloadStringSlice(payload, "importTaskIds")
		if len(ids) == 0 {
			ids = []string{"import-000002"}
		}
		errors := make([]any, 0)
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := s.importTasks[id]; ok {
				delete(s.importTasks, id)
				continue
			}
			errors = append(errors, map[string]any{
				"importTaskId": id,
				"errorMessage": "import task not found",
			})
		}
		return map[string]any{"errors": errors}

	case "StartBatchDeleteConfigurationTask":
		taskID := s.nextTokenLocked("bdct")
		task := &discoveryBatchDeleteTask{
			TaskID:    taskID,
			Status:    "SUCCEEDED",
			StartTime: now,
			EndTime:   now,
			DeletionWarnings: []map[string]any{
				{"configurationId": "srv-000001", "warningCode": "NONE", "message": "deterministic emulation"},
			},
		}
		s.batchDeleteTasks[taskID] = task
		return map[string]any{"taskId": taskID}

	case "DescribeBatchDeleteConfigurationTask":
		taskID := discoveryPayloadString(payload, "taskId", s.firstBatchDeleteTaskIDLocked())
		task := s.ensureBatchDeleteTaskLocked(taskID, now)
		return map[string]any{
			"task": map[string]any{
				"taskId":           task.TaskID,
				"status":           task.Status,
				"startTime":        task.StartTime,
				"endTime":          task.EndTime,
				"deletionWarnings": task.DeletionWarnings,
			},
		}

	case "DescribeConfigurations":
		ids := discoveryPayloadStringSlice(payload, "configurationIds")
		if len(ids) == 0 {
			ids = []string{"srv-000001"}
		}
		items := make([]any, 0, len(ids))
		for _, id := range ids {
			items = append(items, s.ensureConfigLocked(id))
		}
		return map[string]any{"configurations": items}

	case "ListConfigurations":
		items := make([]any, 0, len(s.configurations))
		for _, id := range s.sortedConfigurationIDsLocked() {
			cfg := s.ensureConfigLocked(id)
			items = append(items, map[string]any{
				"configurationId":   id,
				"configurationType": discoveryAnyToString(cfg["configurationType"], "SERVER"),
				"name":              discoveryAnyToString(cfg["name"], discoveryAnyToString(cfg["serverName"], "stackyard-config")),
			})
		}
		return map[string]any{"configurations": items, "nextToken": ""}

	case "ListServerNeighbors":
		source := discoveryPayloadString(payload, "configurationId", "srv-000001")
		return map[string]any{
			"neighbors": []any{
				map[string]any{
					"sourceServerId":      source,
					"destinationServerId": "srv-000002",
					"destinationPort":     443,
					"transportProtocol":   "TCP",
					"connectionsCount":    12,
				},
			},
			"knownDependencyCount": 1,
			"nextToken":            "",
		}

	case "GetDiscoverySummary":
		serverCount := 0
		for _, cfg := range s.configurations {
			if discoveryAnyToString(cfg["configurationType"], "") == "SERVER" {
				serverCount++
			}
		}
		return map[string]any{
			"servers":                        serverCount,
			"applications":                   len(s.applications),
			"agents":                         len(s.agents),
			"agentlessCollectors":            1,
			"connectorAgents":                0,
			"meCollectors":                   0,
			"startDataCollectionAgents":      len(s.agents),
			"stopDataCollectionAgents":       0,
			"notReceivingHeartBeatAgents":    0,
			"numberOfExportedConfigurations": len(s.exportTasks),
			"numberOfDiscoveredServers":      serverCount,
		}

	default:
		return map[string]any{}
	}
}

func (s *discoveryStore) ensureApplicationLocked(id, now string) *discoveryApplication {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextTokenLocked("app")
	}
	if app, ok := s.applications[id]; ok {
		return app
	}
	app := &discoveryApplication{
		ConfigurationID: id,
		Name:            "stackyard-app-" + id,
		Description:     "generated application",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.applications[id] = app
	s.configurations[id] = map[string]any{
		"configurationId":   id,
		"configurationType": "APPLICATION",
		"name":              app.Name,
		"description":       app.Description,
	}
	return app
}

func (s *discoveryStore) ensureContinuousExportLocked(id, now string) *discoveryContinuousExport {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ce-000001"
	}
	if ce, ok := s.continuousExports[id]; ok {
		return ce
	}
	ce := &discoveryContinuousExport{
		ExportID:         id,
		Status:           "STOPPED",
		StartTime:        now,
		StopTime:         now,
		DataSource:       "AGENT",
		SchemaStorageCfg: "S3",
	}
	s.continuousExports[id] = ce
	return ce
}

func (s *discoveryStore) ensureBatchDeleteTaskLocked(id, now string) *discoveryBatchDeleteTask {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextTokenLocked("bdct")
	}
	if task, ok := s.batchDeleteTasks[id]; ok {
		return task
	}
	task := &discoveryBatchDeleteTask{
		TaskID:    id,
		Status:    "SUCCEEDED",
		StartTime: now,
		EndTime:   now,
	}
	s.batchDeleteTasks[id] = task
	return task
}

func (s *discoveryStore) ensureConfigLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "srv-000001"
	}
	if cfg, ok := s.configurations[id]; ok {
		return cfg
	}
	cfg := map[string]any{
		"configurationId":   id,
		"configurationType": "SERVER",
		"serverName":        "stackyard-" + id,
		"hostName":          "stackyard-" + id,
		"osName":            "Linux",
	}
	s.configurations[id] = cfg
	return cfg
}

func (s *discoveryStore) ensureConfigTagsLocked(id string) map[string]string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "srv-000001"
	}
	if _, ok := s.configurations[id]; !ok {
		s.ensureConfigLocked(id)
	}
	tagMap := s.tags[id]
	if tagMap == nil {
		tagMap = map[string]string{}
		s.tags[id] = tagMap
	}
	return tagMap
}

func (s *discoveryStore) createExportTaskLocked(now, format string) *discoveryExportTask {
	taskID := s.nextTokenLocked("export")
	task := &discoveryExportTask{
		ExportID:        taskID,
		Status:          "SUCCEEDED",
		RequestedAt:     now,
		PreferredFormat: strings.ToUpper(strings.TrimSpace(format)),
		S3Bucket:        "stackyard-discovery-export",
	}
	if task.PreferredFormat == "" {
		task.PreferredFormat = "CSV"
	}
	s.exportTasks[taskID] = task
	return task
}

func (s *discoveryStore) updateAgentsCollectionStatusLocked(payload map[string]any, status, health string) map[string]any {
	ids := discoveryPayloadStringSlice(payload, "agentIds")
	if len(ids) == 0 {
		ids = []string{"agent-000001"}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		a, ok := s.agents[id]
		if !ok {
			a = &discoveryAgent{
				AgentID:            id,
				HostName:           "stackyard-" + id,
				AgentType:          "AGENT",
				AgentVersion:       "3.2.1",
				LastHealthPingTime: now,
			}
			s.agents[id] = a
		}
		a.ConfigurationStatus = status
		a.Health = health
		a.LastHealthPingTime = now
		items = append(items, map[string]any{
			"agentId":            id,
			"operationSucceeded": true,
			"description":        "state updated",
		})
	}
	return map[string]any{"agentsConfigurationStatus": items}
}

func (s *discoveryStore) firstApplicationIDLocked() string {
	for _, id := range s.sortedApplicationIDsLocked() {
		return id
	}
	return "app-000001"
}

func (s *discoveryStore) firstBatchDeleteTaskIDLocked() string {
	for _, id := range s.sortedBatchDeleteTaskIDsLocked() {
		return id
	}
	return ""
}

func (s *discoveryStore) sortedAgentsLocked() []*discoveryAgent {
	keys := make([]string, 0, len(s.agents))
	for id := range s.agents {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]*discoveryAgent, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.agents[id])
	}
	return out
}

func (s *discoveryStore) sortedContinuousExportsLocked() []*discoveryContinuousExport {
	keys := make([]string, 0, len(s.continuousExports))
	for id := range s.continuousExports {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]*discoveryContinuousExport, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.continuousExports[id])
	}
	return out
}

func (s *discoveryStore) sortedExportTasksLocked() []*discoveryExportTask {
	keys := make([]string, 0, len(s.exportTasks))
	for id := range s.exportTasks {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]*discoveryExportTask, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.exportTasks[id])
	}
	return out
}

func (s *discoveryStore) sortedImportTasksLocked() []*discoveryImportTask {
	keys := make([]string, 0, len(s.importTasks))
	for id := range s.importTasks {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]*discoveryImportTask, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.importTasks[id])
	}
	return out
}

func (s *discoveryStore) sortedApplicationIDsLocked() []string {
	keys := make([]string, 0, len(s.applications))
	for id := range s.applications {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func (s *discoveryStore) sortedBatchDeleteTaskIDsLocked() []string {
	keys := make([]string, 0, len(s.batchDeleteTasks))
	for id := range s.batchDeleteTasks {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func (s *discoveryStore) sortedConfigurationIDsLocked() []string {
	keys := make([]string, 0, len(s.configurations))
	for id := range s.configurations {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func (s *discoveryStore) nextTokenLocked(prefix string) string {
	token := fmt.Sprintf("%s-%06d", prefix, s.nextID)
	s.nextID++
	return token
}

func discoveryPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return fallback
	}
	return discoveryAnyToString(raw, fallback)
}

func discoveryPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		if one := strings.TrimSpace(discoveryAnyToString(raw, "")); one != "" {
			return []string{one}
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if value := strings.TrimSpace(discoveryAnyToString(item, "")); value != "" {
			out = append(out, value)
		}
	}
	return out
}

type discoveryKV struct {
	Key   string
	Value string
}

func discoveryPayloadTags(payload map[string]any, key string) []discoveryKV {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]discoveryKV, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k := discoveryPayloadString(m, "key", "")
		if k == "" {
			k = discoveryPayloadString(m, "Key", "")
		}
		v := discoveryPayloadString(m, "value", "")
		if v == "" {
			v = discoveryPayloadString(m, "Value", "")
		}
		if strings.TrimSpace(k) == "" {
			continue
		}
		out = append(out, discoveryKV{Key: strings.TrimSpace(k), Value: strings.TrimSpace(v)})
	}
	return out
}

func discoveryDeleteAgentIDs(payload map[string]any) []string {
	ids := discoveryPayloadStringSlice(payload, "agentIds")
	if len(ids) > 0 {
		return ids
	}
	raw, ok := payload["deleteAgents"]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := discoveryPayloadString(m, "agentId", "")
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func discoveryAnyToString(value any, fallback string) string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		return strings.TrimSpace(v)
	case fmt.Stringer:
		s := strings.TrimSpace(v.String())
		if s == "" {
			return fallback
		}
		return s
	default:
		if value == nil {
			return fallback
		}
		s := strings.TrimSpace(fmt.Sprintf("%v", value))
		if s == "" {
			return fallback
		}
		return s
	}
}
