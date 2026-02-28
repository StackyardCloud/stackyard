package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type cloudWatchApplicationInsightsStore struct {
	mu              sync.Mutex
	nextID          int64
	applications    map[string]map[string]any
	components      map[string]map[string]map[string]any
	workloads       map[string]map[string]map[string]any
	logPatterns     map[string]map[string]map[string]map[string]any
	tags            map[string]map[string]string
	problems        map[string]map[string]any
	observations    map[string]map[string]any
	configEvents    map[string][]map[string]any
	problemMappings map[string][]map[string]any
}

func newCloudWatchApplicationInsightsStore() *cloudWatchApplicationInsightsStore {
	now := time.Now().UTC()
	resourceGroupName := "stackyard-appinsights-rg"
	arn := cloudWatchApplicationInsightsApplicationARN(resourceGroupName)
	problemID := "problem-stackyard-000001"
	observationID := "obs-stackyard-000001"

	store := &cloudWatchApplicationInsightsStore{
		nextID:       2,
		applications: map[string]map[string]any{},
		components:   map[string]map[string]map[string]any{},
		workloads:    map[string]map[string]map[string]any{},
		logPatterns:  map[string]map[string]map[string]map[string]any{},
		tags: map[string]map[string]string{
			arn: {"stackyard": "true"},
		},
		problems: map[string]map[string]any{
			problemID: {
				"Id":                    problemID,
				"Title":                 "Stackyard sample problem",
				"Insights":              "Synthetic insight for local contract tests",
				"ResourceGroupName":     resourceGroupName,
				"StartTime":             now.Add(-5 * time.Minute),
				"EndTime":               now,
				"SeverityLevel":         "Medium",
				"Status":                "RESOLVED",
				"AffectedResource":      "stackyard-component",
				"Feedback":              map[string]any{},
				"RecurringCount":        1,
				"Visibility":            "PUBLIC",
				"OpsItemArn":            "",
				"AccountId":             "123456789012",
				"Region":                "us-east-1",
				"LastRecurrenceTime":    now,
				"LastUpdatedTime":       now,
				"SourceService":         "CloudWatch",
				"Subcategory":           "Application",
				"ProblemType":           "APPLICATION",
				"ResourceArn":           arn,
				"MetricName":            "ApplicationErrors",
				"MetricNamespace":       "AWS/ApplicationInsights",
				"MetricsAnalyzerResult": "HEALTHY",
			},
		},
		observations: map[string]map[string]any{
			observationID: {
				"Id":                observationID,
				"StartTime":         now.Add(-10 * time.Minute),
				"EndTime":           now,
				"SourceType":        "CW_METRIC",
				"SourceArn":         arn,
				"LogGroup":          "/stackyard/applicationinsights",
				"MetricName":        "ApplicationErrors",
				"MetricNamespace":   "AWS/ApplicationInsights",
				"Unit":              "Count",
				"Value":             1,
				"CloudWatchEventId": "evt-stackyard-000001",
			},
		},
		configEvents: map[string][]map[string]any{
			resourceGroupName: {
				{
					"MonitoredResourceARN": arn,
					"EventStatus":          "INFO",
					"EventResourceType":    "Application",
					"EventTime":            now,
					"EventDetail":          "Application configured",
					"EventResourceName":    resourceGroupName,
				},
			},
		},
		problemMappings: map[string][]map[string]any{
			problemID: {
				{
					"ObservationId": observationID,
				},
			},
		},
	}
	store.applications[resourceGroupName] = map[string]any{
		"ResourceGroupName":                resourceGroupName,
		"LifeCycle":                        "ACTIVE",
		"OpsItemSNSTopicArn":               "",
		"SNSNotificationArn":               "",
		"OpsCenterEnabled":                 false,
		"CWEMonitorEnabled":                true,
		"Remarks":                          "seeded by Stackyard",
		"AutoConfigEnabled":                true,
		"AttachMissingPermission":          true,
		"AccountId":                        "123456789012",
		"ResourceGroupArn":                 arn,
		"OpsItemEnabled":                   false,
		"AlarmArns":                        []any{},
		"OpsItemsResources":                []any{},
		"ComponentMonitoringSettings":      []any{},
		"CustomComponentConfigurationList": []any{},
	}
	store.components[resourceGroupName] = map[string]map[string]any{
		"stackyard-component": {
			"ComponentName":     "stackyard-component",
			"ResourceType":      "AWS::EC2::Instance",
			"Tier":              "DEFAULT",
			"Monitor":           true,
			"ResourceGroupName": resourceGroupName,
		},
	}
	store.workloads[resourceGroupName] = map[string]map[string]any{
		"stackyard-workload": {
			"WorkloadName":      "stackyard-workload",
			"Tier":              "DEFAULT",
			"ComponentName":     "stackyard-component",
			"WorkloadId":        "workload-000001",
			"ResourceGroupName": resourceGroupName,
		},
	}
	store.logPatterns[resourceGroupName] = map[string]map[string]map[string]any{
		"default": {
			"stackyard-pattern": {
				"ResourceGroupName": resourceGroupName,
				"PatternSetName":    "default",
				"PatternName":       "stackyard-pattern",
				"Pattern":           "ERROR",
				"Rank":              1,
			},
		},
	}

	return store
}

func (s *cloudWatchApplicationInsightsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateApplication":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		app := s.ensureApplicationLocked(rg)
		s.applyApplicationFieldsLocked(app, payload)
		return map[string]any{"ApplicationInfo": cloudWatchApplicationInsightsCloneMap(app)}
	case "DeleteApplication":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		delete(s.applications, rg)
		delete(s.components, rg)
		delete(s.workloads, rg)
		delete(s.logPatterns, rg)
		delete(s.configEvents, rg)
		return map[string]any{}
	case "DescribeApplication":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		return map[string]any{"ApplicationInfo": cloudWatchApplicationInsightsCloneMap(s.ensureApplicationLocked(rg))}
	case "ListApplications":
		return map[string]any{"ApplicationInfoList": s.listApplicationsLocked(), "NextToken": ""}
	case "UpdateApplication":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		app := s.ensureApplicationLocked(rg)
		s.applyApplicationFieldsLocked(app, payload)
		return map[string]any{"ApplicationInfo": cloudWatchApplicationInsightsCloneMap(app)}

	case "CreateComponent":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		componentName := cloudWatchApplicationInsightsDefaultString(payload, "ComponentName", fmt.Sprintf("stackyard-component-%06d", s.nextLocked()))
		component := s.ensureComponentLocked(rg, componentName)
		s.applyComponentFieldsLocked(component, payload)
		return map[string]any{"ComponentName": componentName, "ResourceGroupName": rg}
	case "DeleteComponent":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		componentName := cloudWatchApplicationInsightsDefaultString(payload, "ComponentName", "stackyard-component")
		if s.components[rg] != nil {
			delete(s.components[rg], componentName)
		}
		return map[string]any{}
	case "DescribeComponent":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		componentName := cloudWatchApplicationInsightsDefaultString(payload, "ComponentName", "stackyard-component")
		return map[string]any{"ApplicationComponent": cloudWatchApplicationInsightsCloneMap(s.ensureComponentLocked(rg, componentName))}
	case "ListComponents":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		return map[string]any{"ApplicationComponentList": s.listComponentsLocked(rg), "NextToken": ""}
	case "UpdateComponent":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		componentName := cloudWatchApplicationInsightsDefaultString(payload, "ComponentName", "stackyard-component")
		component := s.ensureComponentLocked(rg, componentName)
		s.applyComponentFieldsLocked(component, payload)
		return map[string]any{"ComponentName": componentName, "ResourceGroupName": rg}
	case "DescribeComponentConfiguration":
		componentName := cloudWatchApplicationInsightsDefaultString(payload, "ComponentName", "stackyard-component")
		return map[string]any{
			"Monitor":                true,
			"Tier":                   "DEFAULT",
			"ComponentName":          componentName,
			"ComponentConfiguration": `{"logs":{"default":{"logGroupName":"/stackyard/applicationinsights"}}}`,
		}
	case "DescribeComponentConfigurationRecommendation":
		componentName := cloudWatchApplicationInsightsDefaultString(payload, "ComponentName", "stackyard-component")
		return map[string]any{
			"ComponentName":          componentName,
			"Tier":                   "DEFAULT",
			"ComponentConfiguration": `{"metrics":{"enabled":true},"logs":{"default":{"logGroupName":"/stackyard/applicationinsights"}}}`,
		}
	case "UpdateComponentConfiguration":
		return map[string]any{}

	case "AddWorkload":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		workloadName := cloudWatchApplicationInsightsDefaultString(payload, "WorkloadName", fmt.Sprintf("stackyard-workload-%06d", s.nextLocked()))
		workload := s.ensureWorkloadLocked(rg, workloadName)
		s.applyWorkloadFieldsLocked(workload, payload)
		return map[string]any{"Workload": cloudWatchApplicationInsightsCloneMap(workload)}
	case "DescribeWorkload":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		workloadName := cloudWatchApplicationInsightsDefaultString(payload, "WorkloadName", "stackyard-workload")
		return map[string]any{"Workload": cloudWatchApplicationInsightsCloneMap(s.ensureWorkloadLocked(rg, workloadName))}
	case "ListWorkloads":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		return map[string]any{"WorkloadList": s.listWorkloadsLocked(rg), "NextToken": ""}
	case "RemoveWorkload":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		workloadName := cloudWatchApplicationInsightsDefaultString(payload, "WorkloadName", "stackyard-workload")
		if s.workloads[rg] != nil {
			delete(s.workloads[rg], workloadName)
		}
		return map[string]any{}
	case "UpdateWorkload":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		workloadName := cloudWatchApplicationInsightsDefaultString(payload, "WorkloadName", "stackyard-workload")
		workload := s.ensureWorkloadLocked(rg, workloadName)
		s.applyWorkloadFieldsLocked(workload, payload)
		return map[string]any{"Workload": cloudWatchApplicationInsightsCloneMap(workload)}

	case "CreateLogPattern":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		patternSetName := cloudWatchApplicationInsightsDefaultString(payload, "PatternSetName", "default")
		patternName := cloudWatchApplicationInsightsDefaultString(payload, "PatternName", fmt.Sprintf("stackyard-pattern-%06d", s.nextLocked()))
		pattern := s.ensureLogPatternLocked(rg, patternSetName, patternName)
		s.applyLogPatternFieldsLocked(pattern, payload)
		return map[string]any{"LogPattern": cloudWatchApplicationInsightsCloneMap(pattern)}
	case "DeleteLogPattern":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		patternSetName := cloudWatchApplicationInsightsDefaultString(payload, "PatternSetName", "default")
		patternName := cloudWatchApplicationInsightsDefaultString(payload, "PatternName", "stackyard-pattern")
		if s.logPatterns[rg] != nil && s.logPatterns[rg][patternSetName] != nil {
			delete(s.logPatterns[rg][patternSetName], patternName)
		}
		return map[string]any{}
	case "DescribeLogPattern":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		patternSetName := cloudWatchApplicationInsightsDefaultString(payload, "PatternSetName", "default")
		patternName := cloudWatchApplicationInsightsDefaultString(payload, "PatternName", "stackyard-pattern")
		return map[string]any{"LogPattern": cloudWatchApplicationInsightsCloneMap(s.ensureLogPatternLocked(rg, patternSetName, patternName))}
	case "ListLogPatternSets":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		return map[string]any{"ResourceGroupName": rg, "LogPatternSets": s.listLogPatternSetsLocked(rg), "NextToken": ""}
	case "ListLogPatterns":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		patternSetName := cloudWatchApplicationInsightsDefaultString(payload, "PatternSetName", "default")
		return map[string]any{"ResourceGroupName": rg, "LogPatterns": s.listLogPatternsLocked(rg, patternSetName), "NextToken": ""}
	case "UpdateLogPattern":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		patternSetName := cloudWatchApplicationInsightsDefaultString(payload, "PatternSetName", "default")
		patternName := cloudWatchApplicationInsightsDefaultString(payload, "PatternName", "stackyard-pattern")
		pattern := s.ensureLogPatternLocked(rg, patternSetName, patternName)
		s.applyLogPatternFieldsLocked(pattern, payload)
		return map[string]any{"LogPattern": cloudWatchApplicationInsightsCloneMap(pattern)}

	case "DescribeObservation":
		observationID := cloudWatchApplicationInsightsDefaultString(payload, "ObservationId", "obs-stackyard-000001")
		return map[string]any{"Observation": cloudWatchApplicationInsightsCloneMap(s.ensureObservationLocked(observationID))}
	case "DescribeProblem":
		problemID := cloudWatchApplicationInsightsDefaultString(payload, "ProblemId", "problem-stackyard-000001")
		return map[string]any{"Problem": cloudWatchApplicationInsightsCloneMap(s.ensureProblemLocked(problemID))}
	case "DescribeProblemObservations":
		problemID := cloudWatchApplicationInsightsDefaultString(payload, "ProblemId", "problem-stackyard-000001")
		return map[string]any{"RelatedObservations": map[string]any{"ObservationList": cloudWatchApplicationInsightsCloneListOfMaps(s.ensureProblemObservationsLocked(problemID))}}
	case "ListConfigurationHistory":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		return map[string]any{"EventList": cloudWatchApplicationInsightsCloneListOfMaps(s.ensureConfigEventsLocked(rg)), "NextToken": ""}
	case "ListProblems":
		rg := cloudWatchApplicationInsightsDefaultString(payload, "ResourceGroupName", "stackyard-appinsights-rg")
		return map[string]any{"ResourceGroupName": rg, "ProblemList": s.listProblemsLocked(rg), "NextToken": ""}
	case "UpdateProblem":
		problemID := cloudWatchApplicationInsightsDefaultString(payload, "ProblemId", "problem-stackyard-000001")
		problem := s.ensureProblemLocked(problemID)
		if visibility := cloudWatchApplicationInsightsPayloadString(payload, "Visibility"); visibility != "" {
			problem["Visibility"] = visibility
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := cloudWatchApplicationInsightsDefaultString(payload, "ResourceARN", cloudWatchApplicationInsightsApplicationARN("stackyard-appinsights-rg"))
		return map[string]any{"Tags": s.tagsForResourceLocked(resourceARN)}
	case "TagResource":
		resourceARN := cloudWatchApplicationInsightsDefaultString(payload, "ResourceARN", cloudWatchApplicationInsightsApplicationARN("stackyard-appinsights-rg"))
		tags := cloudWatchApplicationInsightsMapValue(payload, "Tags")
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for k, v := range tags {
			s.tags[resourceARN][k] = v
		}
		return map[string]any{}
	case "UntagResource":
		resourceARN := cloudWatchApplicationInsightsDefaultString(payload, "ResourceARN", cloudWatchApplicationInsightsApplicationARN("stackyard-appinsights-rg"))
		tagKeys := cloudWatchApplicationInsightsStringSlice(payload, "TagKeys")
		for _, key := range tagKeys {
			if s.tags[resourceARN] != nil {
				delete(s.tags[resourceARN], key)
			}
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *cloudWatchApplicationInsightsStore) nextLocked() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *cloudWatchApplicationInsightsStore) ensureApplicationLocked(resourceGroupName string) map[string]any {
	key := strings.TrimSpace(resourceGroupName)
	if key == "" {
		key = "stackyard-appinsights-rg"
	}
	app := s.applications[key]
	if app != nil {
		return app
	}
	app = map[string]any{
		"ResourceGroupName":       key,
		"LifeCycle":               "ACTIVE",
		"CWEMonitorEnabled":       true,
		"OpsCenterEnabled":        false,
		"AutoConfigEnabled":       true,
		"AttachMissingPermission": true,
		"Remarks":                 "",
		"ResourceGroupArn":        cloudWatchApplicationInsightsApplicationARN(key),
	}
	s.applications[key] = app
	return app
}

func (s *cloudWatchApplicationInsightsStore) ensureComponentLocked(resourceGroupName, componentName string) map[string]any {
	rg := strings.TrimSpace(resourceGroupName)
	if rg == "" {
		rg = "stackyard-appinsights-rg"
	}
	s.ensureApplicationLocked(rg)
	if s.components[rg] == nil {
		s.components[rg] = map[string]map[string]any{}
	}
	name := strings.TrimSpace(componentName)
	if name == "" {
		name = "stackyard-component"
	}
	component := s.components[rg][name]
	if component != nil {
		return component
	}
	component = map[string]any{
		"ResourceGroupName": rg,
		"ComponentName":     name,
		"ResourceType":      "AWS::EC2::Instance",
		"Tier":              "DEFAULT",
		"Monitor":           true,
	}
	s.components[rg][name] = component
	return component
}

func (s *cloudWatchApplicationInsightsStore) ensureWorkloadLocked(resourceGroupName, workloadName string) map[string]any {
	rg := strings.TrimSpace(resourceGroupName)
	if rg == "" {
		rg = "stackyard-appinsights-rg"
	}
	s.ensureApplicationLocked(rg)
	if s.workloads[rg] == nil {
		s.workloads[rg] = map[string]map[string]any{}
	}
	name := strings.TrimSpace(workloadName)
	if name == "" {
		name = "stackyard-workload"
	}
	workload := s.workloads[rg][name]
	if workload != nil {
		return workload
	}
	workload = map[string]any{
		"ResourceGroupName": rg,
		"WorkloadName":      name,
		"WorkloadId":        fmt.Sprintf("workload-%06d", s.nextLocked()),
		"Tier":              "DEFAULT",
		"ComponentName":     "stackyard-component",
	}
	s.workloads[rg][name] = workload
	return workload
}

func (s *cloudWatchApplicationInsightsStore) ensureLogPatternLocked(resourceGroupName, patternSetName, patternName string) map[string]any {
	rg := strings.TrimSpace(resourceGroupName)
	if rg == "" {
		rg = "stackyard-appinsights-rg"
	}
	if s.logPatterns[rg] == nil {
		s.logPatterns[rg] = map[string]map[string]map[string]any{}
	}
	setName := strings.TrimSpace(patternSetName)
	if setName == "" {
		setName = "default"
	}
	if s.logPatterns[rg][setName] == nil {
		s.logPatterns[rg][setName] = map[string]map[string]any{}
	}
	name := strings.TrimSpace(patternName)
	if name == "" {
		name = "stackyard-pattern"
	}
	pattern := s.logPatterns[rg][setName][name]
	if pattern != nil {
		return pattern
	}
	pattern = map[string]any{
		"ResourceGroupName": rg,
		"PatternSetName":    setName,
		"PatternName":       name,
		"Pattern":           "ERROR",
		"Rank":              1,
	}
	s.logPatterns[rg][setName][name] = pattern
	return pattern
}

func (s *cloudWatchApplicationInsightsStore) ensureProblemLocked(problemID string) map[string]any {
	id := strings.TrimSpace(problemID)
	if id == "" {
		id = "problem-stackyard-000001"
	}
	problem := s.problems[id]
	if problem != nil {
		return problem
	}
	now := time.Now().UTC()
	problem = map[string]any{
		"Id":                id,
		"Title":             "Stackyard generated problem",
		"Insights":          "",
		"ResourceGroupName": "stackyard-appinsights-rg",
		"StartTime":         now.Add(-1 * time.Minute),
		"EndTime":           now,
		"SeverityLevel":     "Low",
		"Status":            "OPEN",
		"Visibility":        "PUBLIC",
		"ResourceArn":       cloudWatchApplicationInsightsApplicationARN("stackyard-appinsights-rg"),
	}
	s.problems[id] = problem
	return problem
}

func (s *cloudWatchApplicationInsightsStore) ensureObservationLocked(observationID string) map[string]any {
	id := strings.TrimSpace(observationID)
	if id == "" {
		id = "obs-stackyard-000001"
	}
	observation := s.observations[id]
	if observation != nil {
		return observation
	}
	now := time.Now().UTC()
	observation = map[string]any{
		"Id":                id,
		"StartTime":         now.Add(-2 * time.Minute),
		"EndTime":           now,
		"SourceType":        "CW_METRIC",
		"MetricName":        "ApplicationErrors",
		"MetricNamespace":   "AWS/ApplicationInsights",
		"SourceArn":         cloudWatchApplicationInsightsApplicationARN("stackyard-appinsights-rg"),
		"CloudWatchEventId": fmt.Sprintf("evt-%06d", s.nextLocked()),
	}
	s.observations[id] = observation
	return observation
}

func (s *cloudWatchApplicationInsightsStore) ensureProblemObservationsLocked(problemID string) []map[string]any {
	id := strings.TrimSpace(problemID)
	if id == "" {
		id = "problem-stackyard-000001"
	}
	observations := s.problemMappings[id]
	if len(observations) != 0 {
		return observations
	}
	defaultObs := map[string]any{"ObservationId": "obs-stackyard-000001"}
	s.problemMappings[id] = []map[string]any{defaultObs}
	return s.problemMappings[id]
}

func (s *cloudWatchApplicationInsightsStore) ensureConfigEventsLocked(resourceGroupName string) []map[string]any {
	key := strings.TrimSpace(resourceGroupName)
	if key == "" {
		key = "stackyard-appinsights-rg"
	}
	events := s.configEvents[key]
	if len(events) != 0 {
		return events
	}
	now := time.Now().UTC()
	s.configEvents[key] = []map[string]any{
		{
			"MonitoredResourceARN": cloudWatchApplicationInsightsApplicationARN(key),
			"EventStatus":          "INFO",
			"EventResourceType":    "Application",
			"EventTime":            now,
			"EventDetail":          "Configuration queried",
			"EventResourceName":    key,
		},
	}
	return s.configEvents[key]
}

func (s *cloudWatchApplicationInsightsStore) applyApplicationFieldsLocked(app map[string]any, payload map[string]any) {
	if value, ok := cloudWatchApplicationInsightsPayloadBool(payload, "AutoConfigEnabled"); ok {
		app["AutoConfigEnabled"] = value
	}
	if value, ok := cloudWatchApplicationInsightsPayloadBool(payload, "CWEMonitorEnabled"); ok {
		app["CWEMonitorEnabled"] = value
	}
	if value, ok := cloudWatchApplicationInsightsPayloadBool(payload, "OpsCenterEnabled"); ok {
		app["OpsCenterEnabled"] = value
	}
	if value, ok := cloudWatchApplicationInsightsPayloadBool(payload, "AttachMissingPermission"); ok {
		app["AttachMissingPermission"] = value
	}
	if remarks := cloudWatchApplicationInsightsPayloadString(payload, "Remarks"); remarks != "" {
		app["Remarks"] = remarks
	}
}

func (s *cloudWatchApplicationInsightsStore) applyComponentFieldsLocked(component map[string]any, payload map[string]any) {
	if resourceType := cloudWatchApplicationInsightsPayloadString(payload, "ResourceType"); resourceType != "" {
		component["ResourceType"] = resourceType
	}
	if tier := cloudWatchApplicationInsightsPayloadString(payload, "Tier"); tier != "" {
		component["Tier"] = tier
	}
	if monitor, ok := cloudWatchApplicationInsightsPayloadBool(payload, "Monitor"); ok {
		component["Monitor"] = monitor
	}
}

func (s *cloudWatchApplicationInsightsStore) applyWorkloadFieldsLocked(workload map[string]any, payload map[string]any) {
	if componentName := cloudWatchApplicationInsightsPayloadString(payload, "ComponentName"); componentName != "" {
		workload["ComponentName"] = componentName
	}
	if tier := cloudWatchApplicationInsightsPayloadString(payload, "Tier"); tier != "" {
		workload["Tier"] = tier
	}
}

func (s *cloudWatchApplicationInsightsStore) applyLogPatternFieldsLocked(pattern map[string]any, payload map[string]any) {
	if value := cloudWatchApplicationInsightsPayloadString(payload, "Pattern"); value != "" {
		pattern["Pattern"] = value
	}
	if value, ok := cloudWatchApplicationInsightsPayloadNumber(payload, "Rank"); ok {
		pattern["Rank"] = value
	}
}

func (s *cloudWatchApplicationInsightsStore) listApplicationsLocked() []any {
	names := make([]string, 0, len(s.applications))
	for name := range s.applications {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloudWatchApplicationInsightsCloneMap(s.applications[name]))
	}
	return out
}

func (s *cloudWatchApplicationInsightsStore) listComponentsLocked(resourceGroupName string) []any {
	rg := strings.TrimSpace(resourceGroupName)
	components := s.components[rg]
	if components == nil {
		return []any{}
	}
	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloudWatchApplicationInsightsCloneMap(components[name]))
	}
	return out
}

func (s *cloudWatchApplicationInsightsStore) listWorkloadsLocked(resourceGroupName string) []any {
	rg := strings.TrimSpace(resourceGroupName)
	workloads := s.workloads[rg]
	if workloads == nil {
		return []any{}
	}
	names := make([]string, 0, len(workloads))
	for name := range workloads {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloudWatchApplicationInsightsCloneMap(workloads[name]))
	}
	return out
}

func (s *cloudWatchApplicationInsightsStore) listLogPatternSetsLocked(resourceGroupName string) []any {
	rg := strings.TrimSpace(resourceGroupName)
	patternSets := s.logPatterns[rg]
	if patternSets == nil {
		return []any{}
	}
	names := make([]string, 0, len(patternSets))
	for setName := range patternSets {
		names = append(names, setName)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, setName := range names {
		out = append(out, map[string]any{"PatternSetName": setName})
	}
	return out
}

func (s *cloudWatchApplicationInsightsStore) listLogPatternsLocked(resourceGroupName, patternSetName string) []any {
	rg := strings.TrimSpace(resourceGroupName)
	setName := strings.TrimSpace(patternSetName)
	if setName == "" {
		setName = "default"
	}
	patterns := s.logPatterns[rg][setName]
	if patterns == nil {
		return []any{}
	}
	names := make([]string, 0, len(patterns))
	for name := range patterns {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, cloudWatchApplicationInsightsCloneMap(patterns[name]))
	}
	return out
}

func (s *cloudWatchApplicationInsightsStore) listProblemsLocked(resourceGroupName string) []any {
	rg := strings.TrimSpace(resourceGroupName)
	if rg == "" {
		rg = "stackyard-appinsights-rg"
	}
	problemIDs := make([]string, 0, len(s.problems))
	for problemID, problem := range s.problems {
		if strings.EqualFold(cloudWatchApplicationInsightsPayloadString(problem, "ResourceGroupName"), rg) {
			problemIDs = append(problemIDs, problemID)
		}
	}
	sort.Strings(problemIDs)
	out := make([]any, 0, len(problemIDs))
	for _, problemID := range problemIDs {
		out = append(out, cloudWatchApplicationInsightsCloneMap(s.problems[problemID]))
	}
	return out
}

func (s *cloudWatchApplicationInsightsStore) tagsForResourceLocked(resourceARN string) map[string]string {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = cloudWatchApplicationInsightsApplicationARN("stackyard-appinsights-rg")
	}
	entries := s.tags[arn]
	if entries == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(entries))
	for k, v := range entries {
		out[k] = v
	}
	return out
}

func cloudWatchApplicationInsightsApplicationARN(resourceGroupName string) string {
	name := strings.TrimSpace(resourceGroupName)
	if name == "" {
		name = "stackyard-appinsights-rg"
	}
	return fmt.Sprintf("arn:aws:applicationinsights:us-east-1:123456789012:application/%s", name)
}

func cloudWatchApplicationInsightsPayloadString(payload map[string]any, key string) string {
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return ""
}

func cloudWatchApplicationInsightsDefaultString(payload map[string]any, key, fallback string) string {
	if value := cloudWatchApplicationInsightsPayloadString(payload, key); value != "" {
		return value
	}
	return fallback
}

func cloudWatchApplicationInsightsPayloadBool(payload map[string]any, key string) (bool, bool) {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch typed := v.(type) {
		case bool:
			return typed, true
		case string:
			lower := strings.ToLower(strings.TrimSpace(typed))
			if lower == "true" {
				return true, true
			}
			if lower == "false" {
				return false, true
			}
		}
	}
	return false, false
}

func cloudWatchApplicationInsightsPayloadNumber(payload map[string]any, key string) (int64, bool) {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch typed := v.(type) {
		case int:
			return int64(typed), true
		case int64:
			return typed, true
		case float64:
			return int64(typed), true
		case json.Number:
			if value, err := typed.Int64(); err == nil {
				return value, true
			}
			if value, err := typed.Float64(); err == nil {
				return int64(value), true
			}
		case string:
			trimmed := strings.TrimSpace(typed)
			if trimmed == "" {
				return 0, false
			}
			var parsed int64
			_, err := fmt.Sscan(trimmed, &parsed)
			if err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func cloudWatchApplicationInsightsMapValue(payload map[string]any, key string) map[string]string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if typed, ok := v.(map[string]any); ok {
			out := map[string]string{}
			for mk, mv := range typed {
				out[mk] = strings.TrimSpace(fmt.Sprintf("%v", mv))
			}
			return out
		}
		if typed, ok := v.(map[string]string); ok {
			out := map[string]string{}
			for mk, mv := range typed {
				out[mk] = mv
			}
			return out
		}
	}
	return map[string]string{}
}

func cloudWatchApplicationInsightsStringSlice(payload map[string]any, key string) []string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch typed := v.(type) {
		case []string:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		case []any:
			out := make([]string, 0, len(typed))
			for _, item := range typed {
				trimmed := strings.TrimSpace(fmt.Sprintf("%v", item))
				if trimmed != "" {
					out = append(out, trimmed)
				}
			}
			return out
		}
	}
	return []string{}
}

func cloudWatchApplicationInsightsCloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloudWatchApplicationInsightsCloneListOfMaps(src []map[string]any) []any {
	out := make([]any, 0, len(src))
	for _, entry := range src {
		out = append(out, cloudWatchApplicationInsightsCloneMap(entry))
	}
	return out
}
