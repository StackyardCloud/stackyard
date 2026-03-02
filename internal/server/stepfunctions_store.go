package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
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
	executionHist map[string][]any
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
		executionHist: map[string][]any{
			execution["executionArn"].(string): seedExecutionHistory(now, execution["input"].(string), execution["output"].(string), stateMachine["roleArn"].(string)),
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
		if events, ok := s.executionHist[execARN]; ok {
			return map[string]any{
				"events":       stepFunctionsCloneAny(events),
				"nextToken":    "",
				"executionArn": execARN,
			}
		}
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
		input := stepFunctionsPayloadString(payload, "input", "{}")
		definition, roleARN := s.resolveStateMachineExecutionConfigLocked(smARN)
		status, output, failureError, failureCause, events := executeASL(definition, input, roleARN, now)
		item := map[string]any{
			"executionArn":    execARN,
			"stateMachineArn": smARN,
			"name":            name,
			"status":          status,
			"startDate":       now,
			"input":           input,
		}
		if status != "RUNNING" {
			item["stopDate"] = now
		}
		if strings.TrimSpace(output) != "" {
			item["output"] = output
		}
		if strings.TrimSpace(failureError) != "" {
			item["error"] = failureError
		}
		if strings.TrimSpace(failureCause) != "" {
			item["cause"] = failureCause
		}
		s.executions[execARN] = item
		if len(events) > 0 {
			s.executionHist[execARN] = events
		}
		return map[string]any{"executionArn": execARN, "startDate": now}
	case "StartSyncExecution":
		smARN := stepFunctionsPayloadString(payload, "stateMachineArn", defaultStateMachineARN)
		name := stepFunctionsPayloadString(payload, "name", "sync-"+s.nextIdentifierLocked("e"))
		execARN := fmt.Sprintf("arn:aws:states:us-east-1:123456789012:execution:%s:%s", stepFunctionsStateMachineName(smARN), name)
		input := stepFunctionsPayloadString(payload, "input", "{}")
		definition, roleARN := s.resolveStateMachineExecutionConfigLocked(smARN)
		status, output, failureError, failureCause, events := executeASL(definition, input, roleARN, now)
		item := map[string]any{
			"executionArn":    execARN,
			"stateMachineArn": smARN,
			"name":            name,
			"status":          status,
			"startDate":       now,
			"stopDate":        now,
			"input":           input,
		}
		if strings.TrimSpace(output) != "" {
			item["output"] = output
		}
		if strings.TrimSpace(failureError) != "" {
			item["error"] = failureError
		}
		if strings.TrimSpace(failureCause) != "" {
			item["cause"] = failureCause
		}
		s.executions[execARN] = item
		if len(events) > 0 {
			s.executionHist[execARN] = events
		}
		return map[string]any{
			"executionArn":    execARN,
			"stateMachineArn": smARN,
			"status":          status,
			"startDate":       now,
			"stopDate":        now,
			"input":           input,
			"output":          output,
			"error":           failureError,
			"cause":           failureCause,
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

func (s *stepFunctionsStore) resolveStateMachineExecutionConfigLocked(stateMachineARN string) (definition, roleARN string) {
	sm := s.ensureStateMachineLocked(stateMachineARN)
	definition = stepFunctionsPayloadString(sm, "definition", `{"StartAt":"Done","States":{"Done":{"Type":"Succeed"}}}`)
	roleARN = stepFunctionsPayloadString(sm, "roleArn", "arn:aws:iam::123456789012:role/stackyard-step-functions-role")

	if strings.Contains(stateMachineARN, ":stateMachine:") && strings.Count(stateMachineARN, ":") > strings.Count("arn:aws:states:us-east-1:123456789012:stateMachine:stackyard-sm", ":") {
		baseARN := stateMachineARN[:strings.LastIndex(stateMachineARN, ":")]
		if base, ok := s.stateMachines[baseARN]; ok {
			definition = stepFunctionsPayloadString(base, "definition", definition)
			roleARN = stepFunctionsPayloadString(base, "roleArn", roleARN)
		}
	}
	return definition, roleARN
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

type stepFunctionsASLDefinition struct {
	StartAt string                              `json:"StartAt"`
	States  map[string]stepFunctionsASLStateDef `json:"States"`
}

type stepFunctionsASLStateDef struct {
	Type       string                        `json:"Type"`
	Next       string                        `json:"Next"`
	End        bool                          `json:"End"`
	Result     any                           `json:"Result"`
	Default    string                        `json:"Default"`
	Choices    []stepFunctionsASLChoiceState `json:"Choices"`
	Error      string                        `json:"Error"`
	Cause      string                        `json:"Cause"`
	InputPath  string                        `json:"InputPath"`
	OutputPath string                        `json:"OutputPath"`
	ResultPath string                        `json:"ResultPath"`
	Parameters map[string]any                `json:"Parameters"`
	ItemsPath  string                        `json:"ItemsPath"`
	Iterator   *stepFunctionsASLDefinition   `json:"Iterator"`
	Branches   []stepFunctionsASLDefinition  `json:"Branches"`
}

type stepFunctionsASLChoiceState struct {
	Variable      string   `json:"Variable"`
	StringEquals  *string  `json:"StringEquals"`
	NumericEquals *float64 `json:"NumericEquals"`
	BooleanEquals *bool    `json:"BooleanEquals"`
	Next          string   `json:"Next"`
}

func seedExecutionHistory(ts, input, output, roleARN string) []any {
	return []any{
		map[string]any{
			"id":              int64(1),
			"previousEventId": int64(0),
			"timestamp":       ts,
			"type":            "ExecutionStarted",
			"executionStartedEventDetails": map[string]any{
				"input":   defaultJSONString(input, "{}"),
				"roleArn": roleARN,
			},
		},
		map[string]any{
			"id":              int64(2),
			"previousEventId": int64(1),
			"timestamp":       ts,
			"type":            "ExecutionSucceeded",
			"executionSucceededEventDetails": map[string]any{
				"output": defaultJSONString(output, "{}"),
			},
		},
	}
}

func executeASL(definitionJSON, inputJSON, roleARN, ts string) (status, output, failureError, failureCause string, events []any) {
	builder := newStepFunctionsHistoryBuilder(ts)
	builder.add("ExecutionStarted", map[string]any{
		"executionStartedEventDetails": map[string]any{
			"input":   defaultJSONString(inputJSON, "{}"),
			"roleArn": roleARN,
		},
	})

	definition := stepFunctionsASLDefinition{}
	if err := json.Unmarshal([]byte(definitionJSON), &definition); err != nil {
		failureError, failureCause = "States.Runtime", "invalid state machine definition JSON"
		builder.add("ExecutionFailed", map[string]any{
			"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
		})
		return "FAILED", "", failureError, failureCause, builder.events
	}
	if strings.TrimSpace(definition.StartAt) == "" || len(definition.States) == 0 {
		failureError, failureCause = "States.Runtime", "missing StartAt/States"
		builder.add("ExecutionFailed", map[string]any{
			"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
		})
		return "FAILED", "", failureError, failureCause, builder.events
	}

	outputValue, failureError, failureCause := runASLDefinition(definition, parseJSONAny(inputJSON, map[string]any{}), builder)
	if strings.TrimSpace(failureError) != "" {
		builder.add("ExecutionFailed", map[string]any{
			"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
		})
		return "FAILED", "", failureError, failureCause, builder.events
	}

	output = toJSONString(outputValue, "{}")
	builder.add("ExecutionSucceeded", map[string]any{
		"executionSucceededEventDetails": map[string]any{"output": output},
	})
	return "SUCCEEDED", output, "", "", builder.events
}

func runASLDefinition(definition stepFunctionsASLDefinition, input any, builder *stepFunctionsHistoryBuilder) (output any, failureError, failureCause string) {
	current := stepFunctionsCloneAny(input)
	stateName := definition.StartAt
	for steps := 0; steps < 256; steps++ {
		state, ok := definition.States[stateName]
		if !ok {
			return nil, "States.Runtime", "state not found: " + stateName
		}

		stateInput, err := stepFunctionsApplyInputAndParameters(current, state)
		if err != nil {
			return nil, "States.Runtime", err.Error()
		}

		switch state.Type {
		case "Pass":
			stepFunctionsAddStateEnteredEvent(builder, "PassStateEntered", stateName, stateInput)
			stateResult := stateInput
			if state.Result != nil {
				stateResult = stepFunctionsCloneAny(state.Result)
			}
			stateOutput, err := stepFunctionsComposeStateOutput(stateInput, stateResult, state.ResultPath, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			stepFunctionsAddStateExitedEvent(builder, "PassStateExited", stateName, stateOutput)
			if state.End {
				return stateOutput, "", ""
			}
			if strings.TrimSpace(state.Next) == "" {
				return nil, "States.Runtime", "missing Next for state: " + stateName
			}
			current = stateOutput
			stateName = state.Next

		case "Task":
			stepFunctionsAddStateEnteredEvent(builder, "TaskStateEntered", stateName, stateInput)
			stateResult := stateInput
			if state.Result != nil {
				stateResult = stepFunctionsCloneAny(state.Result)
			}
			stateOutput, err := stepFunctionsComposeStateOutput(stateInput, stateResult, state.ResultPath, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			stepFunctionsAddStateExitedEvent(builder, "TaskStateExited", stateName, stateOutput)
			if state.End {
				return stateOutput, "", ""
			}
			if strings.TrimSpace(state.Next) == "" {
				return nil, "States.Runtime", "missing Next for state: " + stateName
			}
			current = stateOutput
			stateName = state.Next

		case "Choice":
			stepFunctionsAddStateEnteredEvent(builder, "ChoiceStateEntered", stateName, stateInput)
			next := ""
			for _, choice := range state.Choices {
				if matchesChoice(choice, stateInput) {
					next = choice.Next
					break
				}
			}
			if strings.TrimSpace(next) == "" {
				next = state.Default
			}
			if strings.TrimSpace(next) == "" {
				return nil, "States.NoChoiceMatched", "no choice matched and no default"
			}
			stateOutput, err := stepFunctionsApplyOutputPath(stateInput, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			stepFunctionsAddStateExitedEvent(builder, "ChoiceStateExited", stateName, stateOutput)
			current = stateOutput
			stateName = next

		case "Wait":
			stepFunctionsAddStateEnteredEvent(builder, "WaitStateEntered", stateName, stateInput)
			stateOutput, err := stepFunctionsApplyOutputPath(stateInput, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			stepFunctionsAddStateExitedEvent(builder, "WaitStateExited", stateName, stateOutput)
			if state.End {
				return stateOutput, "", ""
			}
			if strings.TrimSpace(state.Next) == "" {
				return nil, "States.Runtime", "missing Next for state: " + stateName
			}
			current = stateOutput
			stateName = state.Next

		case "Map":
			stepFunctionsAddStateEnteredEvent(builder, "MapStateEntered", stateName, stateInput)
			itemsPath := defaultIfEmpty(state.ItemsPath, "$")
			rawItems, ok := readJSONPathValue(stateInput, itemsPath)
			if !ok {
				return nil, "States.Runtime", "map itemsPath not found: " + itemsPath
			}
			items, ok := rawItems.([]any)
			if !ok {
				return nil, "States.Runtime", "map itemsPath must resolve to array"
			}
			if state.Iterator == nil || strings.TrimSpace(state.Iterator.StartAt) == "" || len(state.Iterator.States) == 0 {
				return nil, "States.Runtime", "map iterator is required"
			}
			results := make([]any, 0, len(items))
			for _, item := range items {
				itemOutput, itemErr, itemCause := runASLDefinition(*state.Iterator, stepFunctionsCloneAny(item), nil)
				if strings.TrimSpace(itemErr) != "" {
					return nil, itemErr, itemCause
				}
				results = append(results, stepFunctionsCloneAny(itemOutput))
			}
			stateOutput, err := stepFunctionsComposeStateOutput(stateInput, results, state.ResultPath, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			stepFunctionsAddStateExitedEvent(builder, "MapStateExited", stateName, stateOutput)
			if state.End {
				return stateOutput, "", ""
			}
			if strings.TrimSpace(state.Next) == "" {
				return nil, "States.Runtime", "missing Next for state: " + stateName
			}
			current = stateOutput
			stateName = state.Next

		case "Parallel":
			stepFunctionsAddStateEnteredEvent(builder, "ParallelStateEntered", stateName, stateInput)
			if len(state.Branches) == 0 {
				return nil, "States.Runtime", "parallel branches are required"
			}
			results := make([]any, 0, len(state.Branches))
			for _, branch := range state.Branches {
				if strings.TrimSpace(branch.StartAt) == "" || len(branch.States) == 0 {
					return nil, "States.Runtime", "parallel branch definition is invalid"
				}
				branchOutput, branchErr, branchCause := runASLDefinition(branch, stepFunctionsCloneAny(stateInput), nil)
				if strings.TrimSpace(branchErr) != "" {
					return nil, branchErr, branchCause
				}
				results = append(results, stepFunctionsCloneAny(branchOutput))
			}
			stateOutput, err := stepFunctionsComposeStateOutput(stateInput, results, state.ResultPath, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			stepFunctionsAddStateExitedEvent(builder, "ParallelStateExited", stateName, stateOutput)
			if state.End {
				return stateOutput, "", ""
			}
			if strings.TrimSpace(state.Next) == "" {
				return nil, "States.Runtime", "missing Next for state: " + stateName
			}
			current = stateOutput
			stateName = state.Next

		case "Succeed":
			stateOutput, err := stepFunctionsApplyOutputPath(stateInput, state.OutputPath)
			if err != nil {
				return nil, "States.Runtime", err.Error()
			}
			return stateOutput, "", ""

		case "Fail":
			return nil, defaultIfEmpty(state.Error, "States.Failed"), defaultIfEmpty(state.Cause, "execution failed")

		default:
			return nil, "States.Runtime", "unsupported state type: " + state.Type
		}
	}

	return nil, "States.Timeout", "state machine exceeded max transitions"
}

func stepFunctionsAddStateEnteredEvent(builder *stepFunctionsHistoryBuilder, eventType, stateName string, input any) {
	if builder == nil {
		return
	}
	builder.add(eventType, map[string]any{
		"stateEnteredEventDetails": map[string]any{
			"name":  stateName,
			"input": toJSONString(input, "{}"),
		},
	})
}

func stepFunctionsAddStateExitedEvent(builder *stepFunctionsHistoryBuilder, eventType, stateName string, output any) {
	if builder == nil {
		return
	}
	builder.add(eventType, map[string]any{
		"stateExitedEventDetails": map[string]any{
			"name":   stateName,
			"output": toJSONString(output, "{}"),
		},
	})
}

func stepFunctionsApplyInputAndParameters(current any, state stepFunctionsASLStateDef) (any, error) {
	stateInput, err := stepFunctionsApplyInputPath(current, state.InputPath)
	if err != nil {
		return nil, err
	}
	if len(state.Parameters) == 0 {
		return stateInput, nil
	}
	return stepFunctionsResolveParameters(state.Parameters, stateInput)
}

func stepFunctionsApplyInputPath(input any, inputPath string) (any, error) {
	path := strings.TrimSpace(inputPath)
	if path == "" || path == "$" {
		return stepFunctionsCloneAny(input), nil
	}
	value, ok := readJSONPathValue(input, path)
	if !ok {
		return nil, fmt.Errorf("input path not found: %s", path)
	}
	return stepFunctionsCloneAny(value), nil
}

func stepFunctionsApplyOutputPath(output any, outputPath string) (any, error) {
	path := strings.TrimSpace(outputPath)
	if path == "" || path == "$" {
		return stepFunctionsCloneAny(output), nil
	}
	value, ok := readJSONPathValue(output, path)
	if !ok {
		return nil, fmt.Errorf("output path not found: %s", path)
	}
	return stepFunctionsCloneAny(value), nil
}

func stepFunctionsComposeStateOutput(stateInput, stateResult any, resultPath, outputPath string) (any, error) {
	merged, err := stepFunctionsApplyResultPath(stateInput, stateResult, resultPath)
	if err != nil {
		return nil, err
	}
	return stepFunctionsApplyOutputPath(merged, outputPath)
}

func stepFunctionsApplyResultPath(stateInput, stateResult any, resultPath string) (any, error) {
	path := strings.TrimSpace(resultPath)
	if path == "" || path == "$" {
		return stepFunctionsCloneAny(stateResult), nil
	}
	if strings.EqualFold(path, "null") {
		return stepFunctionsCloneAny(stateInput), nil
	}
	clonedInput := stepFunctionsCloneAny(stateInput)
	root, ok := clonedInput.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("result path requires object input: %s", path)
	}
	if err := stepFunctionsSetJSONPathValue(root, path, stepFunctionsCloneAny(stateResult)); err != nil {
		return nil, err
	}
	return root, nil
}

func stepFunctionsSetJSONPathValue(root map[string]any, path string, value any) error {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "$.") {
		return fmt.Errorf("unsupported result path: %s", path)
	}
	parts := strings.Split(path[2:], ".")
	if len(parts) == 0 {
		return fmt.Errorf("invalid result path: %s", path)
	}
	current := root
	for idx, part := range parts {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("invalid result path: %s", path)
		}
		if idx == len(parts)-1 {
			current[part] = value
			return nil
		}
		next, ok := current[part]
		if !ok {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("result path traverses non-object field %q in %s", part, path)
		}
		current = child
	}
	return nil
}

func stepFunctionsResolveParameters(template map[string]any, stateInput any) (map[string]any, error) {
	resolved, err := stepFunctionsResolveParameterNode(template, stateInput)
	if err != nil {
		return nil, err
	}
	out, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parameters must resolve to object")
	}
	return out, nil
}

func stepFunctionsResolveParameterNode(node any, stateInput any) (any, error) {
	switch t := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for key, value := range t {
			if strings.HasSuffix(key, ".$") {
				expr, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("parameter expression for %q must be string", key)
				}
				resolved, ok := readJSONPathValue(stateInput, expr)
				if !ok {
					return nil, fmt.Errorf("parameter path not found for %q: %s", key, expr)
				}
				out[strings.TrimSuffix(key, ".$")] = stepFunctionsCloneAny(resolved)
				continue
			}
			resolved, err := stepFunctionsResolveParameterNode(value, stateInput)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			resolved, err := stepFunctionsResolveParameterNode(item, stateInput)
			if err != nil {
				return nil, err
			}
			out = append(out, resolved)
		}
		return out, nil
	default:
		return stepFunctionsCloneAny(t), nil
	}
}

type stepFunctionsHistoryBuilder struct {
	ts     string
	nextID int64
	events []any
}

func newStepFunctionsHistoryBuilder(ts string) *stepFunctionsHistoryBuilder {
	return &stepFunctionsHistoryBuilder{
		ts:     ts,
		nextID: 1,
		events: make([]any, 0, 8),
	}
}

func (b *stepFunctionsHistoryBuilder) add(eventType string, details map[string]any) {
	id := b.nextID
	b.nextID++
	event := map[string]any{
		"id":        id,
		"timestamp": b.ts,
		"type":      eventType,
	}
	if id == 1 {
		event["previousEventId"] = int64(0)
	} else {
		event["previousEventId"] = id - 1
	}
	for k, v := range details {
		event[k] = stepFunctionsCloneAny(v)
	}
	b.events = append(b.events, event)
}

func matchesChoice(choice stepFunctionsASLChoiceState, data any) bool {
	value, ok := readJSONPathValue(data, choice.Variable)
	if !ok {
		return false
	}
	if choice.StringEquals != nil {
		v, ok := value.(string)
		return ok && v == *choice.StringEquals
	}
	if choice.BooleanEquals != nil {
		v, ok := value.(bool)
		return ok && v == *choice.BooleanEquals
	}
	if choice.NumericEquals != nil {
		n, ok := asFloat(value)
		return ok && n == *choice.NumericEquals
	}
	return false
}

func readJSONPathValue(data any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	if path == "" || path == "$" {
		return data, true
	}
	if !strings.HasPrefix(path, "$.") {
		return nil, false
	}
	parts := strings.Split(path[2:], ".")
	current := data
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := m[part]
		if !ok {
			return nil, false
		}
		current = v
	}
	return current, true
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case int16:
		return float64(n), true
	case int8:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint8:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func parseJSONAny(src string, def any) any {
	src = strings.TrimSpace(src)
	if src == "" {
		return stepFunctionsCloneAny(def)
	}
	var out any
	if err := json.Unmarshal([]byte(src), &out); err != nil {
		return stepFunctionsCloneAny(def)
	}
	return out
}

func toJSONString(v any, def string) string {
	enc, err := json.Marshal(v)
	if err != nil {
		return def
	}
	if len(enc) == 0 {
		return def
	}
	return string(enc)
}

func defaultJSONString(raw, def string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}
	return raw
}
