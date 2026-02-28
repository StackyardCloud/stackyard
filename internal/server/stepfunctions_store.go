package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type stepFunctionsStore struct {
	mu sync.Mutex

	nextID int64

	activities    map[string]map[string]any
	stateMachines map[string]map[string]any
	aliases       map[string]map[string]any
	versions      map[string]map[string]any
	executions    map[string]map[string]any
	mapRuns       map[string]map[string]any
	tags          map[string]map[string]string
}

func newStepFunctionsStore() *stepFunctionsStore {
	now := time.Now().UTC().Format(time.RFC3339)

	activity := map[string]any{
		"activityArn":  "arn:aws:states:us-east-1:123456789012:activity:stackyard-activity",
		"name":         "stackyard-activity",
		"creationDate": now,
	}
	stateMachine := map[string]any{
		"stateMachineArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm",
		"name":            "stackyard-sm",
		"type":            "STANDARD",
		"status":          "ACTIVE",
		"definition":      `{"StartAt":"Done","States":{"Done":{"Type":"Succeed"}}}`,
		"roleArn":         "arn:aws:iam::123456789012:role/stackyard-step-functions-role",
		"creationDate":    now,
	}
	alias := map[string]any{
		"stateMachineAliasArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm:LIVE",
		"name":                 "LIVE",
		"creationDate":         now,
		"updateDate":           now,
		"routingConfiguration": []any{
			map[string]any{
				"stateMachineVersionArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm:1",
				"weight":                 int64(100),
			},
		},
	}
	version := map[string]any{
		"stateMachineVersionArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm:1",
		"creationDate":           now,
	}
	execution := map[string]any{
		"executionArn":    "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec",
		"stateMachineArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm",
		"name":            "stackyard-exec",
		"status":          "SUCCEEDED",
		"startDate":       now,
		"stopDate":        now,
		"input":           "{}",
		"output":          "{}",
	}
	mapRun := map[string]any{
		"mapRunArn":       "arn:aws:states:us-east-1:123456789012:mapRun:stackyard-sm/stackyard-exec:maprun-00000001",
		"executionArn":    "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec",
		"stateMachineArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm",
		"status":          "SUCCEEDED",
		"startDate":       now,
		"stopDate":        now,
	}

	return &stepFunctionsStore{
		nextID: 1,
		activities: map[string]map[string]any{
			activity["activityArn"].(string): activity,
		},
		stateMachines: map[string]map[string]any{
			stateMachine["stateMachineArn"].(string): stateMachine,
		},
		aliases: map[string]map[string]any{
			alias["stateMachineAliasArn"].(string): alias,
		},
		versions: map[string]map[string]any{
			version["stateMachineVersionArn"].(string): version,
		},
		executions: map[string]map[string]any{
			execution["executionArn"].(string): execution,
		},
		mapRuns: map[string]map[string]any{
			mapRun["mapRunArn"].(string): mapRun,
		},
		tags: map[string]map[string]string{
			"arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm": {
				"env": "coverage",
			},
		},
	}
}

func (s *stepFunctionsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	defaultStateMachineARN := "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm"

	switch action {
	case "CreateActivity":
		name := stepFunctionsPayloadString(payload, "name", "stackyard-activity")
		arn := "arn:aws:states:us-east-1:123456789012:activity:" + name
		item := map[string]any{
			"activityArn":  arn,
			"name":         name,
			"creationDate": now,
		}
		s.activities[arn] = item
		return map[string]any{"activityArn": arn, "creationDate": now}
	case "CreateStateMachine":
		name := stepFunctionsPayloadString(payload, "name", "stackyard-sm")
		arn := "arn:aws:states:us-east-1:123456789012:stateMachine:" + name
		item := map[string]any{
			"stateMachineArn": arn,
			"name":            name,
			"type":            stepFunctionsPayloadString(payload, "type", "STANDARD"),
			"status":          "ACTIVE",
			"definition":      stepFunctionsPayloadString(payload, "definition", `{"StartAt":"Done","States":{"Done":{"Type":"Succeed"}}}`),
			"roleArn":         stepFunctionsPayloadString(payload, "roleArn", "arn:aws:iam::123456789012:role/stackyard-step-functions-role"),
			"creationDate":    now,
		}
		s.stateMachines[arn] = item
		s.ensureTagsLocked(arn)
		return map[string]any{"stateMachineArn": arn, "creationDate": now}
	case "CreateStateMachineAlias":
		aliasName := stepFunctionsPayloadString(payload, "name", "LIVE")
		smARN := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		aliasARN := smARN + ":" + aliasName
		item := map[string]any{
			"stateMachineAliasArn": aliasARN,
			"name":                 aliasName,
			"creationDate":         now,
			"updateDate":           now,
			"routingConfiguration": stepFunctionsPayloadAnySlice(payload, "routingConfiguration"),
		}
		if len(item["routingConfiguration"].([]any)) == 0 {
			item["routingConfiguration"] = []any{map[string]any{
				"stateMachineVersionArn": smARN + ":1",
				"weight":                 int64(100),
			}}
		}
		s.aliases[aliasARN] = item
		return map[string]any{"stateMachineAliasArn": aliasARN, "creationDate": now}
	case "DeleteActivity":
		delete(s.activities, stepFunctionsPayloadString(payload, "activityArn", ""))
		return map[string]any{}
	case "DeleteStateMachine":
		delete(s.stateMachines, stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN))
		return map[string]any{}
	case "DeleteStateMachineAlias":
		delete(s.aliases, stepFunctionsPayloadString(payload, "stateMachineAliasArn", defaultStateMachineARN+":LIVE"))
		return map[string]any{}
	case "DeleteStateMachineVersion":
		delete(s.versions, stepFunctionsPayloadString(payload, "stateMachineVersionArn", defaultStateMachineARN+":1"))
		return map[string]any{}
	case "DescribeActivity":
		arn := stepFunctionsPayloadString(payload, "activityArn", "arn:aws:states:us-east-1:123456789012:activity:stackyard-activity")
		item := s.ensureActivityLocked(arn)
		return stepFunctionsCloneMap(item)
	case "DescribeExecution":
		arn := stepFunctionsPayloadString(payload, "executionArn", "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec")
		item := s.ensureExecutionLocked(arn)
		return stepFunctionsCloneMap(item)
	case "DescribeMapRun":
		arn := stepFunctionsPayloadString(payload, "mapRunArn", "arn:aws:states:us-east-1:123456789012:mapRun:stackyard-sm/stackyard-exec:maprun-00000001")
		item := s.ensureMapRunLocked(arn)
		out := stepFunctionsCloneMap(item)
		out["maxConcurrency"] = int64(10)
		out["itemCounts"] = map[string]any{
			"pending":        int64(0),
			"running":        int64(0),
			"succeeded":      int64(1),
			"failed":         int64(0),
			"timedOut":       int64(0),
			"aborted":        int64(0),
			"total":          int64(1),
			"resultsWritten": int64(1),
		}
		out["executionCounts"] = map[string]any{
			"pending":   int64(0),
			"running":   int64(0),
			"succeeded": int64(1),
			"failed":    int64(0),
			"timedOut":  int64(0),
			"aborted":   int64(0),
			"total":     int64(1),
		}
		return out
	case "DescribeStateMachine":
		arn := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		item := s.ensureStateMachineLocked(arn)
		out := stepFunctionsCloneMap(item)
		out["loggingConfiguration"] = map[string]any{"level": "OFF", "includeExecutionData": false, "destinations": []any{}}
		out["tracingConfiguration"] = map[string]any{"enabled": false}
		return out
	case "DescribeStateMachineAlias":
		arn := stepFunctionsPayloadString(payload, "stateMachineAliasArn", defaultStateMachineARN+":LIVE")
		item := s.ensureAliasLocked(arn)
		return stepFunctionsCloneMap(item)
	case "DescribeStateMachineForExecution":
		exec := s.ensureExecutionLocked(stepFunctionsPayloadString(payload, "executionArn", "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec"))
		smARN := stepFunctionsPayloadString(exec, "stateMachineArn", defaultStateMachineARN)
		sm := s.ensureStateMachineLocked(smARN)
		return map[string]any{
			"stateMachineArn": smARN,
			"name":            stepFunctionsPayloadString(sm, "name", "stackyard-sm"),
			"definition":      stepFunctionsPayloadString(sm, "definition", `{"StartAt":"Done","States":{"Done":{"Type":"Succeed"}}}`),
			"roleArn":         stepFunctionsPayloadString(sm, "roleArn", "arn:aws:iam::123456789012:role/stackyard-step-functions-role"),
			"updateDate":      now,
		}
	case "GetActivityTask":
		return map[string]any{
			"taskToken": "task-token-" + s.nextIdentifierLocked("token"),
			"input":     "{}",
		}
	case "GetExecutionHistory":
		execARN := stepFunctionsPayloadString(payload, "executionArn", "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec")
		return map[string]any{
			"events": []any{
				map[string]any{
					"id":              int64(1),
					"previousEventId": int64(0),
					"timestamp":       now,
					"type":            "ExecutionStarted",
					"executionStartedEventDetails": map[string]any{
						"input":   "{}",
						"roleArn": "arn:aws:iam::123456789012:role/stackyard-step-functions-role",
					},
				},
				map[string]any{
					"id":              int64(2),
					"previousEventId": int64(1),
					"timestamp":       now,
					"type":            "ExecutionSucceeded",
					"executionSucceededEventDetails": map[string]any{
						"output": "{}",
					},
				},
			},
			"nextToken":    "",
			"executionArn": execARN,
		}
	case "ListActivities":
		return map[string]any{"activities": s.listActivitiesLocked(), "nextToken": ""}
	case "ListExecutions":
		return map[string]any{"executions": s.listExecutionsLocked(), "nextToken": ""}
	case "ListMapRuns":
		return map[string]any{"mapRuns": s.listMapRunsLocked(), "nextToken": ""}
	case "ListStateMachineAliases":
		return map[string]any{"stateMachineAliases": s.listAliasesLocked(), "nextToken": ""}
	case "ListStateMachines":
		return map[string]any{"stateMachines": s.listStateMachinesLocked(), "nextToken": ""}
	case "ListStateMachineVersions":
		return map[string]any{"stateMachineVersions": s.listVersionsLocked(), "nextToken": ""}
	case "ListTagsForResource":
		resourceARN := stepFunctionsPayloadString(payload, "resourceArn", defaultStateMachineARN)
		return map[string]any{"tags": stepFunctionsTagsToList(s.ensureTagsLocked(resourceARN))}
	case "PublishStateMachineVersion":
		smARN := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		versionARN := smARN + ":" + s.nextIdentifierLocked("v")
		item := map[string]any{"stateMachineVersionArn": versionARN, "creationDate": now}
		s.versions[versionARN] = item
		return map[string]any{"stateMachineVersionArn": versionARN, "creationDate": now}
	case "RedriveExecution":
		return map[string]any{"redriveDate": now}
	case "SendTaskFailure", "SendTaskHeartbeat", "SendTaskSuccess":
		return map[string]any{}
	case "StartExecution":
		smARN := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		name := stepFunctionsPayloadString(payload, "name", "exec-"+s.nextIdentifierLocked("e"))
		execARN := fmt.Sprintf("arn:aws:states:us-east-1:123456789012:execution:%s:%s", stepFunctionsStateMachineName(smARN), name)
		item := map[string]any{
			"executionArn":    execARN,
			"stateMachineArn": smARN,
			"name":            name,
			"status":          "RUNNING",
			"startDate":       now,
			"input":           stepFunctionsPayloadString(payload, "input", "{}"),
		}
		s.executions[execARN] = item
		return map[string]any{"executionArn": execARN, "startDate": now}
	case "StartSyncExecution":
		smARN := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		name := stepFunctionsPayloadString(payload, "name", "sync-"+s.nextIdentifierLocked("e"))
		execARN := fmt.Sprintf("arn:aws:states:us-east-1:123456789012:execution:%s:%s", stepFunctionsStateMachineName(smARN), name)
		return map[string]any{
			"executionArn":    execARN,
			"stateMachineArn": smARN,
			"status":          "SUCCEEDED",
			"startDate":       now,
			"stopDate":        now,
			"input":           stepFunctionsPayloadString(payload, "input", "{}"),
			"output":          "{}",
			"outputDetails":   map[string]any{"included": true},
			"billingDetails":  map[string]any{"billedDurationInMilliseconds": int64(1), "billedMemoryUsedInMB": int64(64)},
		}
	case "StopExecution":
		exec := s.ensureExecutionLocked(stepFunctionsPayloadString(payload, "executionArn", "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec"))
		exec["status"] = "ABORTED"
		exec["stopDate"] = now
		return map[string]any{"stopDate": now}
	case "TagResource":
		resourceARN := stepFunctionsPayloadString(payload, "resourceArn", defaultStateMachineARN)
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range stepFunctionsTagsFromAny(payload["tags"]) {
			tags[k] = v
		}
		return map[string]any{}
	case "TestState":
		return map[string]any{
			"output": "{}",
			"status": "SUCCEEDED",
			"inspectionData": map[string]any{
				"input": map[string]any{
					"provided": true,
				},
			},
		}
	case "UntagResource":
		resourceARN := stepFunctionsPayloadString(payload, "resourceArn", defaultStateMachineARN)
		tags := s.ensureTagsLocked(resourceARN)
		for _, k := range stepFunctionsPayloadStringSlice(payload, "tagKeys") {
			delete(tags, k)
		}
		return map[string]any{}
	case "UpdateMapRun":
		return map[string]any{"mapRunArn": stepFunctionsPayloadString(payload, "mapRunArn", "arn:aws:states:us-east-1:123456789012:mapRun:stackyard-sm/stackyard-exec:maprun-00000001")}
	case "UpdateStateMachine":
		smARN := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		sm := s.ensureStateMachineLocked(smARN)
		sm["definition"] = stepFunctionsPayloadString(payload, "definition", stepFunctionsPayloadString(sm, "definition", `{"StartAt":"Done","States":{"Done":{"Type":"Succeed"}}}`))
		sm["roleArn"] = stepFunctionsPayloadString(payload, "roleArn", stepFunctionsPayloadString(sm, "roleArn", "arn:aws:iam::123456789012:role/stackyard-step-functions-role"))
		return map[string]any{"updateDate": now, "revisionId": "rev-" + s.nextIdentifierLocked("r")}
	case "UpdateStateMachineAlias":
		arn := stepFunctionsPayloadString(payload, "stateMachineAliasArn", defaultStateMachineARN+":LIVE")
		alias := s.ensureAliasLocked(arn)
		alias["updateDate"] = now
		return map[string]any{"updateDate": now}
	case "ValidateStateMachineDefinition":
		return map[string]any{
			"result":      "OK",
			"diagnostics": []any{},
		}
	default:
		return map[string]any{}
	}
}

func (s *stepFunctionsStore) ensureActivityLocked(activityARN string) map[string]any {
	if activityARN == "" {
		activityARN = "arn:aws:states:us-east-1:123456789012:activity:stackyard-activity"
	}
	if existing, ok := s.activities[activityARN]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"activityArn":  activityARN,
		"name":         stepFunctionsActivityName(activityARN),
		"creationDate": now,
	}
	s.activities[activityARN] = item
	return item
}

func (s *stepFunctionsStore) ensureStateMachineLocked(stateMachineARN string) map[string]any {
	if stateMachineARN == "" {
		stateMachineARN = "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm"
	}
	if existing, ok := s.stateMachines[stateMachineARN]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"stateMachineArn": stateMachineARN,
		"name":            stepFunctionsStateMachineName(stateMachineARN),
		"type":            "STANDARD",
		"status":          "ACTIVE",
		"definition":      `{"StartAt":"Done","States":{"Done":{"Type":"Succeed"}}}`,
		"roleArn":         "arn:aws:iam::123456789012:role/stackyard-step-functions-role",
		"creationDate":    now,
	}
	s.stateMachines[stateMachineARN] = item
	return item
}

func (s *stepFunctionsStore) ensureAliasLocked(aliasARN string) map[string]any {
	if aliasARN == "" {
		aliasARN = "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm:LIVE"
	}
	if existing, ok := s.aliases[aliasARN]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"stateMachineAliasArn": aliasARN,
		"name":                 stepFunctionsAliasName(aliasARN),
		"creationDate":         now,
		"updateDate":           now,
		"routingConfiguration": []any{
			map[string]any{
				"stateMachineVersionArn": stepFunctionsAliasStateMachineARN(aliasARN) + ":1",
				"weight":                 int64(100),
			},
		},
	}
	s.aliases[aliasARN] = item
	return item
}

func (s *stepFunctionsStore) ensureExecutionLocked(executionARN string) map[string]any {
	if executionARN == "" {
		executionARN = "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec"
	}
	if existing, ok := s.executions[executionARN]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"executionArn":    executionARN,
		"stateMachineArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm",
		"name":            stepFunctionsExecutionName(executionARN),
		"status":          "SUCCEEDED",
		"startDate":       now,
		"stopDate":        now,
		"input":           "{}",
		"output":          "{}",
	}
	s.executions[executionARN] = item
	return item
}

func (s *stepFunctionsStore) ensureMapRunLocked(mapRunARN string) map[string]any {
	if mapRunARN == "" {
		mapRunARN = "arn:aws:states:us-east-1:123456789012:mapRun:stackyard-sm/stackyard-exec:maprun-00000001"
	}
	if existing, ok := s.mapRuns[mapRunARN]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := map[string]any{
		"mapRunArn":       mapRunARN,
		"executionArn":    "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec",
		"stateMachineArn": "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm",
		"status":          "SUCCEEDED",
		"startDate":       now,
		"stopDate":        now,
	}
	s.mapRuns[mapRunARN] = item
	return item
}

func (s *stepFunctionsStore) ensureTagsLocked(resourceARN string) map[string]string {
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm"
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	s.tags[resourceARN] = map[string]string{}
	return s.tags[resourceARN]
}

func (s *stepFunctionsStore) listActivitiesLocked() []any {
	keys := make([]string, 0, len(s.activities))
	for arn := range s.activities {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		item := s.activities[arn]
		out = append(out, map[string]any{
			"activityArn":  stepFunctionsPayloadString(item, "activityArn", arn),
			"name":         stepFunctionsPayloadString(item, "name", stepFunctionsActivityName(arn)),
			"creationDate": stepFunctionsPayloadString(item, "creationDate", time.Now().UTC().Format(time.RFC3339)),
		})
	}
	return out
}

func (s *stepFunctionsStore) listStateMachinesLocked() []any {
	keys := make([]string, 0, len(s.stateMachines))
	for arn := range s.stateMachines {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		item := s.stateMachines[arn]
		out = append(out, map[string]any{
			"stateMachineArn": stepFunctionsPayloadString(item, "stateMachineArn", arn),
			"name":            stepFunctionsPayloadString(item, "name", stepFunctionsStateMachineName(arn)),
			"type":            stepFunctionsPayloadString(item, "type", "STANDARD"),
			"creationDate":    stepFunctionsPayloadString(item, "creationDate", time.Now().UTC().Format(time.RFC3339)),
		})
	}
	return out
}

func (s *stepFunctionsStore) listAliasesLocked() []any {
	keys := make([]string, 0, len(s.aliases))
	for arn := range s.aliases {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		item := s.aliases[arn]
		out = append(out, map[string]any{
			"stateMachineAliasArn": stepFunctionsPayloadString(item, "stateMachineAliasArn", arn),
			"name":                 stepFunctionsPayloadString(item, "name", stepFunctionsAliasName(arn)),
			"creationDate":         stepFunctionsPayloadString(item, "creationDate", time.Now().UTC().Format(time.RFC3339)),
			"updateDate":           stepFunctionsPayloadString(item, "updateDate", time.Now().UTC().Format(time.RFC3339)),
		})
	}
	return out
}

func (s *stepFunctionsStore) listVersionsLocked() []any {
	keys := make([]string, 0, len(s.versions))
	for arn := range s.versions {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		item := s.versions[arn]
		out = append(out, map[string]any{
			"stateMachineVersionArn": stepFunctionsPayloadString(item, "stateMachineVersionArn", arn),
			"creationDate":           stepFunctionsPayloadString(item, "creationDate", time.Now().UTC().Format(time.RFC3339)),
		})
	}
	return out
}

func (s *stepFunctionsStore) listExecutionsLocked() []any {
	keys := make([]string, 0, len(s.executions))
	for arn := range s.executions {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		item := s.executions[arn]
		out = append(out, map[string]any{
			"executionArn":    stepFunctionsPayloadString(item, "executionArn", arn),
			"stateMachineArn": stepFunctionsPayloadString(item, "stateMachineArn", "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm"),
			"name":            stepFunctionsPayloadString(item, "name", stepFunctionsExecutionName(arn)),
			"status":          stepFunctionsPayloadString(item, "status", "SUCCEEDED"),
			"startDate":       stepFunctionsPayloadString(item, "startDate", time.Now().UTC().Format(time.RFC3339)),
			"stopDate":        stepFunctionsPayloadString(item, "stopDate", time.Now().UTC().Format(time.RFC3339)),
		})
	}
	return out
}

func (s *stepFunctionsStore) listMapRunsLocked() []any {
	keys := make([]string, 0, len(s.mapRuns))
	for arn := range s.mapRuns {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, arn := range keys {
		item := s.mapRuns[arn]
		out = append(out, map[string]any{
			"mapRunArn":       stepFunctionsPayloadString(item, "mapRunArn", arn),
			"executionArn":    stepFunctionsPayloadString(item, "executionArn", "arn:aws:states:us-east-1:123456789012:execution:stackyard-sm:stackyard-exec"),
			"stateMachineArn": stepFunctionsPayloadString(item, "stateMachineArn", "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm"),
			"startDate":       stepFunctionsPayloadString(item, "startDate", time.Now().UTC().Format(time.RFC3339)),
			"stopDate":        stepFunctionsPayloadString(item, "stopDate", time.Now().UTC().Format(time.RFC3339)),
		})
	}
	return out
}

func (s *stepFunctionsStore) nextIdentifierLocked(prefix string) string {
	id := s.nextID
	s.nextID++
	if strings.TrimSpace(prefix) == "" {
		prefix = "id"
	}
	return fmt.Sprintf("%s-%017d", prefix, id)
}

func stepFunctionsPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	if v, ok := payload[key]; ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return def
}

func stepFunctionsPayloadAnySlice(payload map[string]any, key string) []any {
	if payload == nil {
		return []any{}
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return []any{}
	}
	items, ok := raw.([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, stepFunctionsCloneAny(item))
	}
	return out
}

func stepFunctionsPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return []string{}
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return []string{}
	}
	items, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func stepFunctionsTagsFromAny(v any) map[string]string {
	out := map[string]string{}
	items, ok := v.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := stepFunctionsPayloadString(m, "key", "")
		if key == "" {
			key = stepFunctionsPayloadString(m, "Key", "")
		}
		if key == "" {
			continue
		}
		val := stepFunctionsPayloadString(m, "value", "")
		if val == "" {
			val = stepFunctionsPayloadString(m, "Value", "")
		}
		out[key] = val
	}
	return out
}

func stepFunctionsTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"key": k, "value": tags[k]})
	}
	return out
}

func stepFunctionsCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = stepFunctionsCloneAny(v)
	}
	return out
}

func stepFunctionsCloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return stepFunctionsCloneMap(t)
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, stepFunctionsCloneAny(item))
		}
		return out
	default:
		return t
	}
}

func stepFunctionsStateMachineName(arn string) string {
	if idx := strings.LastIndex(arn, ":stateMachine:"); idx >= 0 {
		return arn[idx+len(":stateMachine:"):]
	}
	if idx := strings.LastIndex(arn, ":"); idx >= 0 && idx+1 < len(arn) {
		return arn[idx+1:]
	}
	return "stackyard-sm"
}

func stepFunctionsActivityName(arn string) string {
	if idx := strings.LastIndex(arn, ":activity:"); idx >= 0 {
		return arn[idx+len(":activity:"):]
	}
	if idx := strings.LastIndex(arn, ":"); idx >= 0 && idx+1 < len(arn) {
		return arn[idx+1:]
	}
	return "stackyard-activity"
}

func stepFunctionsAliasName(arn string) string {
	last := strings.LastIndex(arn, ":")
	if last < 0 || last+1 >= len(arn) {
		return "LIVE"
	}
	return arn[last+1:]
}

func stepFunctionsAliasStateMachineARN(aliasARN string) string {
	last := strings.LastIndex(aliasARN, ":")
	if last < 0 {
		return "arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm"
	}
	return aliasARN[:last]
}

func stepFunctionsExecutionName(arn string) string {
	last := strings.LastIndex(arn, ":")
	if last < 0 || last+1 >= len(arn) {
		return "stackyard-exec"
	}
	return arn[last+1:]
}
