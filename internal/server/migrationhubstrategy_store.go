package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type migrationHubStrategyStore struct {
	mu sync.Mutex

	nextAssessmentID int64
	nextImportTaskID int64
	nextReportID     int64

	latestAssessmentID string

	assessments          map[string]map[string]any
	importFileTasks      map[string]map[string]any
	recommendationReport map[string]map[string]any
	portfolioPrefs       map[string]any
	applicationComponent map[string]map[string]any
	componentStrategies  map[string][]map[string]any
	servers              map[string]map[string]any
	serverStrategies     map[string][]map[string]any
	collectors           map[string]map[string]any
}

func newMigrationHubStrategyStore() *migrationHubStrategyStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &migrationHubStrategyStore{
		nextAssessmentID:     2,
		nextImportTaskID:     2,
		nextReportID:         2,
		latestAssessmentID:   "assessment-00000001",
		assessments:          map[string]map[string]any{},
		importFileTasks:      map[string]map[string]any{},
		recommendationReport: map[string]map[string]any{},
		portfolioPrefs: map[string]any{
			"applicationMode": "ALL",
			"managementPreference": map[string]any{
				"awsManagedResources": map[string]any{"targetDestination": []any{"AWS"}},
			},
			"prioritizeBusinessGoals": map[string]any{
				"speedOfMigration":    3,
				"licenseCost":         2,
				"operationalOverhead": 2,
			},
		},
		applicationComponent: map[string]map[string]any{},
		componentStrategies:  map[string][]map[string]any{},
		servers:              map[string]map[string]any{},
		serverStrategies:     map[string][]map[string]any{},
		collectors:           map[string]map[string]any{},
	}

	s.assessments["assessment-00000001"] = map[string]any{
		"id":                    "assessment-00000001",
		"status":                "COMPLETED",
		"dataCollectionDetails": map[string]any{"startTime": now, "completionTime": now},
	}
	s.importFileTasks["import-00000001"] = map[string]any{
		"id":         "import-00000001",
		"status":     "IMPORT_COMPLETED",
		"startTime":  now,
		"updateTime": now,
	}
	s.recommendationReport["report-00000001"] = map[string]any{
		"id":             "report-00000001",
		"status":         "SUCCEEDED",
		"startTime":      now,
		"completionTime": now,
		"s3Bucket":       "stackyard-migrationhubstrategy-reports",
		"s3Keys":         []any{"reports/report-00000001.json"},
	}
	s.applicationComponent["appcomp-00000001"] = map[string]any{
		"applicationComponentId": "appcomp-00000001",
		"applicationName":        "stackyard-app",
		"componentName":          "stackyard-web",
		"osDriver":               "Linux",
		"status":                 "ANALYZED",
	}
	s.componentStrategies["appcomp-00000001"] = []map[string]any{
		{
			"isPreferred": true,
			"recommendation": map[string]any{
				"strategy":          "Replatform",
				"targetDestination": "ECS",
			},
		},
	}
	s.servers["server-00000001"] = map[string]any{
		"serverId":   "server-00000001",
		"hostname":   "stackyard-host-1",
		"osVersion":  "Ubuntu 22.04",
		"serverType": "EC2",
		"status":     "ANALYZED",
	}
	s.serverStrategies["server-00000001"] = []map[string]any{
		{
			"isPreferred": true,
			"recommendation": map[string]any{
				"strategy":          "Relocate",
				"targetDestination": "EC2",
			},
		},
	}
	s.collectors["collector-00000001"] = map[string]any{
		"collectorId":    "collector-00000001",
		"hostName":       "collector.local",
		"status":         "HEALTHY",
		"registeredTime": now,
	}

	return s
}

func (s *migrationHubStrategyStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := mhsFirstNonEmpty(
		mhsPathParam(pathParams, "id"),
		mhsStringAny(payload, "id"),
	)
	componentID := mhsFirstNonEmpty(
		mhsPathParam(pathParams, "applicationComponentId"),
		mhsStringAny(payload, "applicationComponentId"),
		"appcomp-00000001",
	)
	serverID := mhsFirstNonEmpty(
		mhsPathParam(pathParams, "serverId"),
		mhsStringAny(payload, "serverId"),
		"server-00000001",
	)

	switch action {
	case "GetApplicationComponentDetails":
		component := s.ensureApplicationComponentLocked(componentID)
		return map[string]any{"applicationComponentDetail": mhsCloneMap(component)}

	case "GetApplicationComponentStrategies":
		strategies := s.ensureComponentStrategiesLocked(componentID)
		return map[string]any{"applicationComponentStrategies": mhsCloneSlice(strategies)}

	case "GetAssessment":
		assessment := s.ensureAssessmentLocked(mhsFirstNonEmpty(id, s.latestAssessmentID))
		return map[string]any{"assessment": mhsCloneMap(assessment)}

	case "GetImportFileTask":
		task := s.ensureImportFileTaskLocked(mhsFirstNonEmpty(id, "import-00000001"))
		return map[string]any{"task": mhsCloneMap(task)}

	case "GetLatestAssessmentId":
		return map[string]any{"id": s.latestAssessmentID}

	case "GetPortfolioPreferences":
		return map[string]any{"portfolioPreferences": mhsCloneMap(s.portfolioPrefs)}

	case "GetPortfolioSummary":
		return map[string]any{
			"portfolioSummary": map[string]any{
				"serverCount":               len(s.servers),
				"analyzedServerCount":       len(s.servers),
				"applicationComponentCount": len(s.applicationComponent),
				"analyzedApplicationCount":  len(s.applicationComponent),
			},
		}

	case "GetRecommendationReportDetails":
		report := s.ensureRecommendationReportLocked(mhsFirstNonEmpty(id, "report-00000001"))
		return map[string]any{"recommendationReportDetails": mhsCloneMap(report)}

	case "GetServerDetails":
		server := s.ensureServerLocked(serverID)
		return map[string]any{"serverDetail": mhsCloneMap(server)}

	case "GetServerStrategies":
		strategies := s.ensureServerStrategiesLocked(serverID)
		return map[string]any{"serverStrategies": mhsCloneSlice(strategies)}

	case "ListAnalyzableServers":
		items := make([]any, 0, len(s.servers))
		for _, server := range s.sortedServersLocked() {
			items = append(items, map[string]any{
				"serverId":  mhsStringAny(server, "serverId"),
				"hostname":  mhsStringAny(server, "hostname"),
				"osVersion": mhsStringAny(server, "osVersion"),
			})
		}
		return map[string]any{"analyzableServers": items, "nextToken": ""}

	case "ListApplicationComponents":
		items := make([]any, 0, len(s.applicationComponent))
		for _, c := range s.sortedApplicationComponentsLocked() {
			items = append(items, map[string]any{
				"applicationComponentId": mhsStringAny(c, "applicationComponentId"),
				"applicationName":        mhsStringAny(c, "applicationName"),
				"componentName":          mhsStringAny(c, "componentName"),
			})
		}
		return map[string]any{"applicationComponentInfos": items, "nextToken": ""}

	case "ListCollectors":
		items := make([]any, 0, len(s.collectors))
		keys := make([]string, 0, len(s.collectors))
		for key := range s.collectors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			items = append(items, mhsCloneMap(s.collectors[key]))
		}
		return map[string]any{"Collectors": items, "nextToken": ""}

	case "ListImportFileTask":
		items := make([]any, 0, len(s.importFileTasks))
		keys := make([]string, 0, len(s.importFileTasks))
		for key := range s.importFileTasks {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			items = append(items, mhsCloneMap(s.importFileTasks[key]))
		}
		return map[string]any{"taskInfos": items, "nextToken": ""}

	case "ListServers":
		items := make([]any, 0, len(s.servers))
		for _, server := range s.sortedServersLocked() {
			items = append(items, map[string]any{
				"serverId":   mhsStringAny(server, "serverId"),
				"hostname":   mhsStringAny(server, "hostname"),
				"osVersion":  mhsStringAny(server, "osVersion"),
				"serverType": mhsStringAny(server, "serverType"),
				"status":     mhsStringAny(server, "status"),
			})
		}
		return map[string]any{"serverInfos": items, "nextToken": ""}

	case "PutPortfolioPreferences":
		for key, value := range payload {
			s.portfolioPrefs[key] = value
		}
		return map[string]any{"portfolioPreferences": mhsCloneMap(s.portfolioPrefs)}

	case "StartAssessment":
		assessmentID := fmt.Sprintf("assessment-%08d", s.nextAssessmentIDLocked())
		now := time.Now().UTC().Format(time.RFC3339)
		s.assessments[assessmentID] = map[string]any{
			"id":     assessmentID,
			"status": "IN_PROGRESS",
			"dataCollectionDetails": map[string]any{
				"startTime": now,
			},
		}
		s.latestAssessmentID = assessmentID
		return map[string]any{"id": assessmentID}

	case "StartImportFileTask":
		taskID := fmt.Sprintf("import-%08d", s.nextImportFileTaskIDLocked())
		now := time.Now().UTC().Format(time.RFC3339)
		s.importFileTasks[taskID] = map[string]any{
			"id":         taskID,
			"status":     "IMPORT_IN_PROGRESS",
			"startTime":  now,
			"updateTime": now,
		}
		return map[string]any{"id": taskID}

	case "StartRecommendationReportGeneration":
		reportID := fmt.Sprintf("report-%08d", s.nextReportIDLocked())
		now := time.Now().UTC().Format(time.RFC3339)
		s.recommendationReport[reportID] = map[string]any{
			"id":             reportID,
			"status":         "IN_PROGRESS",
			"startTime":      now,
			"s3Bucket":       "stackyard-migrationhubstrategy-reports",
			"s3Keys":         []any{fmt.Sprintf("reports/%s.json", reportID)},
			"completionTime": "",
		}
		return map[string]any{"id": reportID}

	case "StopAssessment":
		targetID := mhsFirstNonEmpty(mhsStringAny(payload, "id"), s.latestAssessmentID)
		assessment := s.ensureAssessmentLocked(targetID)
		assessment["status"] = "STOPPED"
		return map[string]any{"id": targetID}

	case "UpdateApplicationComponentConfig":
		component := s.ensureApplicationComponentLocked(componentID)
		for key, value := range payload {
			component[key] = value
		}
		component["status"] = "CONFIGURED"
		return map[string]any{"applicationComponentId": componentID}

	case "UpdateServerConfig":
		server := s.ensureServerLocked(serverID)
		for key, value := range payload {
			server[key] = value
		}
		server["status"] = "CONFIGURED"
		return map[string]any{"serverId": serverID}
	}

	return map[string]any{}
}

func (s *migrationHubStrategyStore) ensureApplicationComponentLocked(componentID string) map[string]any {
	componentID = strings.TrimSpace(componentID)
	if componentID == "" {
		componentID = "appcomp-00000001"
	}
	if component, ok := s.applicationComponent[componentID]; ok {
		return component
	}
	component := map[string]any{
		"applicationComponentId": componentID,
		"applicationName":        "stackyard-app",
		"componentName":          componentID,
		"osDriver":               "Linux",
		"status":                 "ANALYZED",
	}
	s.applicationComponent[componentID] = component
	return component
}

func (s *migrationHubStrategyStore) ensureComponentStrategiesLocked(componentID string) []map[string]any {
	componentID = strings.TrimSpace(componentID)
	if componentID == "" {
		componentID = "appcomp-00000001"
	}
	if strategies, ok := s.componentStrategies[componentID]; ok {
		return strategies
	}
	s.componentStrategies[componentID] = []map[string]any{
		{
			"isPreferred": true,
			"recommendation": map[string]any{
				"strategy":          "Replatform",
				"targetDestination": "ECS",
			},
		},
	}
	return s.componentStrategies[componentID]
}

func (s *migrationHubStrategyStore) ensureServerLocked(serverID string) map[string]any {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		serverID = "server-00000001"
	}
	if server, ok := s.servers[serverID]; ok {
		return server
	}
	server := map[string]any{
		"serverId":   serverID,
		"hostname":   serverID,
		"osVersion":  "Linux",
		"serverType": "EC2",
		"status":     "ANALYZED",
	}
	s.servers[serverID] = server
	return server
}

func (s *migrationHubStrategyStore) ensureServerStrategiesLocked(serverID string) []map[string]any {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		serverID = "server-00000001"
	}
	if strategies, ok := s.serverStrategies[serverID]; ok {
		return strategies
	}
	s.serverStrategies[serverID] = []map[string]any{
		{
			"isPreferred": true,
			"recommendation": map[string]any{
				"strategy":          "Relocate",
				"targetDestination": "EC2",
			},
		},
	}
	return s.serverStrategies[serverID]
}

func (s *migrationHubStrategyStore) ensureAssessmentLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.latestAssessmentID
	}
	if assessment, ok := s.assessments[id]; ok {
		return assessment
	}
	now := time.Now().UTC().Format(time.RFC3339)
	assessment := map[string]any{
		"id":                    id,
		"status":                "IN_PROGRESS",
		"dataCollectionDetails": map[string]any{"startTime": now},
	}
	s.assessments[id] = assessment
	return assessment
}

func (s *migrationHubStrategyStore) ensureImportFileTaskLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "import-00000001"
	}
	if task, ok := s.importFileTasks[id]; ok {
		return task
	}
	now := time.Now().UTC().Format(time.RFC3339)
	task := map[string]any{
		"id":         id,
		"status":     "IMPORT_IN_PROGRESS",
		"startTime":  now,
		"updateTime": now,
	}
	s.importFileTasks[id] = task
	return task
}

func (s *migrationHubStrategyStore) ensureRecommendationReportLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "report-00000001"
	}
	if report, ok := s.recommendationReport[id]; ok {
		return report
	}
	now := time.Now().UTC().Format(time.RFC3339)
	report := map[string]any{
		"id":             id,
		"status":         "IN_PROGRESS",
		"startTime":      now,
		"s3Bucket":       "stackyard-migrationhubstrategy-reports",
		"s3Keys":         []any{fmt.Sprintf("reports/%s.json", id)},
		"completionTime": "",
	}
	s.recommendationReport[id] = report
	return report
}

func (s *migrationHubStrategyStore) sortedServersLocked() []map[string]any {
	keys := make([]string, 0, len(s.servers))
	for id := range s.servers {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.servers[id])
	}
	return out
}

func (s *migrationHubStrategyStore) sortedApplicationComponentsLocked() []map[string]any {
	keys := make([]string, 0, len(s.applicationComponent))
	for id := range s.applicationComponent {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, id := range keys {
		out = append(out, s.applicationComponent[id])
	}
	return out
}

func (s *migrationHubStrategyStore) nextAssessmentIDLocked() int64 {
	id := s.nextAssessmentID
	s.nextAssessmentID++
	return id
}

func (s *migrationHubStrategyStore) nextImportFileTaskIDLocked() int64 {
	id := s.nextImportTaskID
	s.nextImportTaskID++
	return id
}

func (s *migrationHubStrategyStore) nextReportIDLocked() int64 {
	id := s.nextReportID
	s.nextReportID++
	return id
}

func mhsPathParam(pathParams map[string]string, key string) string {
	for k, v := range pathParams {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func mhsStringAny(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
		for k, v := range payload {
			if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
				if text, ok := v.(string); ok && strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		}
	}
	return ""
}

func mhsFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mhsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mhsCloneSlice(in []map[string]any) []map[string]any {
	if in == nil {
		return nil
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, mhsCloneMap(item))
	}
	return out
}
