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
	Type    string                        `json:"Type"`
	Next    string                        `json:"Next"`
	End     bool                          `json:"End"`
	Result  any                           `json:"Result"`
	Default string                        `json:"Default"`
	Choices []stepFunctionsASLChoiceState `json:"Choices"`
	Error   string                        `json:"Error"`
	Cause   string                        `json:"Cause"`
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

	current := parseJSONAny(inputJSON, map[string]any{})
	stateName := definition.StartAt
	for steps := 0; steps < 256; steps++ {
		state, ok := definition.States[stateName]
		if !ok {
			failureError, failureCause = "States.Runtime", "state not found: "+stateName
			builder.add("ExecutionFailed", map[string]any{
				"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
			})
			return "FAILED", "", failureError, failureCause, builder.events
		}

		switch state.Type {
		case "Pass":
			builder.add("PassStateEntered", map[string]any{
				"stateEnteredEventDetails": map[string]any{
					"name":  stateName,
					"input": toJSONString(current, "{}"),
				},
			})
			if state.Result != nil {
				current = stepFunctionsCloneAny(state.Result)
			}
			builder.add("PassStateExited", map[string]any{
				"stateExitedEventDetails": map[string]any{
					"name":   stateName,
					"output": toJSONString(current, "{}"),
				},
			})
			if state.End {
				output = toJSONString(current, "{}")
				builder.add("ExecutionSucceeded", map[string]any{
					"executionSucceededEventDetails": map[string]any{"output": output},
				})
				return "SUCCEEDED", output, "", "", builder.events
			}
			stateName = state.Next

		case "Task":
			builder.add("TaskStateEntered", map[string]any{
				"stateEnteredEventDetails": map[string]any{
					"name":  stateName,
					"input": toJSONString(current, "{}"),
				},
			})
			if state.Result != nil {
				current = stepFunctionsCloneAny(state.Result)
			}
			builder.add("TaskStateExited", map[string]any{
				"stateExitedEventDetails": map[string]any{
					"name":   stateName,
					"output": toJSONString(current, "{}"),
				},
			})
			if state.End {
				output = toJSONString(current, "{}")
				builder.add("ExecutionSucceeded", map[string]any{
					"executionSucceededEventDetails": map[string]any{"output": output},
				})
				return "SUCCEEDED", output, "", "", builder.events
			}
			stateName = state.Next

		case "Choice":
			builder.add("ChoiceStateEntered", map[string]any{
				"stateEnteredEventDetails": map[string]any{
					"name":  stateName,
					"input": toJSONString(current, "{}"),
				},
			})
			next := ""
			for _, choice := range state.Choices {
				if matchesChoice(choice, current) {
					next = choice.Next
					break
				}
			}
			if strings.TrimSpace(next) == "" {
				next = state.Default
			}
			if strings.TrimSpace(next) == "" {
				failureError, failureCause = "States.NoChoiceMatched", "no choice matched and no default"
				builder.add("ExecutionFailed", map[string]any{
					"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
				})
				return "FAILED", "", failureError, failureCause, builder.events
			}
			builder.add("ChoiceStateExited", map[string]any{
				"stateExitedEventDetails": map[string]any{
					"name":   stateName,
					"output": toJSONString(current, "{}"),
				},
			})
			stateName = next

		case "Wait":
			builder.add("WaitStateEntered", map[string]any{
				"stateEnteredEventDetails": map[string]any{
					"name":  stateName,
					"input": toJSONString(current, "{}"),
				},
			})
			builder.add("WaitStateExited", map[string]any{
				"stateExitedEventDetails": map[string]any{
					"name":   stateName,
					"output": toJSONString(current, "{}"),
				},
			})
			if state.End {
				output = toJSONString(current, "{}")
				builder.add("ExecutionSucceeded", map[string]any{
					"executionSucceededEventDetails": map[string]any{"output": output},
				})
				return "SUCCEEDED", output, "", "", builder.events
			}
			stateName = state.Next

		case "Succeed":
			output = toJSONString(current, "{}")
			builder.add("ExecutionSucceeded", map[string]any{
				"executionSucceededEventDetails": map[string]any{"output": output},
			})
			return "SUCCEEDED", output, "", "", builder.events

		case "Fail":
			failureError = defaultIfEmpty(state.Error, "States.Failed")
			failureCause = defaultIfEmpty(state.Cause, "execution failed")
			builder.add("ExecutionFailed", map[string]any{
				"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
			})
			return "FAILED", "", failureError, failureCause, builder.events

		default:
			failureError, failureCause = "States.Runtime", "unsupported state type: "+state.Type
			builder.add("ExecutionFailed", map[string]any{
				"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
			})
			return "FAILED", "", failureError, failureCause, builder.events
		}
	}

	failureError, failureCause = "States.Timeout", "state machine exceeded max transitions"
	builder.add("ExecutionFailed", map[string]any{
		"executionFailedEventDetails": map[string]any{"error": failureError, "cause": failureCause},
	})
	return "FAILED", "", failureError, failureCause, builder.events
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
