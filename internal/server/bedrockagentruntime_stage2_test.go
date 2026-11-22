package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestBedrockAgentRuntimeStage2Sessions(t *testing.T) {
	ts := newBedrockAgentTestServer(t)

	created := barJSON(t, ts, http.StatusCreated, http.MethodPut, "/sessions/", map[string]any{"sessionMetadata": map[string]any{"user": "u1"}, "tags": map[string]any{"team": "core"}})
	id, arn := created["sessionId"].(string), created["sessionArn"].(string)
	if created["sessionStatus"] != "ACTIVE" || !barSessionARN.MatchString(arn) {
		t.Fatalf("unexpected session: %v", created)
	}
	sess := "/sessions/" + id + "/"
	if got := barJSON(t, ts, http.StatusOK, http.MethodGet, "/sessions/"+url.PathEscape(arn)+"/", nil); got["sessionMetadata"].(map[string]any)["user"] != "u1" {
		t.Fatalf("GetSession by ARN: %v", got)
	}
	barJSON(t, ts, http.StatusOK, http.MethodPut, sess, map[string]any{"sessionMetadata": map[string]any{"user": "u2"}})

	inv := barJSON(t, ts, http.StatusCreated, http.MethodPut, sess+"invocations/", map[string]any{"description": "first"})["invocationId"].(string)
	barJSON(t, ts, http.StatusConflict, http.MethodPut, sess+"invocations/", map[string]any{"invocationId": inv})
	step := barJSON(t, ts, http.StatusCreated, http.MethodPut, sess+"invocationSteps/", map[string]any{
		"invocationIdentifier": inv, "invocationStepTime": "2025-01-01T00:00:00Z",
		"payload": map[string]any{"contentBlocks": []any{map[string]any{"text": "hello"}}},
	})["invocationStepId"].(string)
	barJSON(t, ts, http.StatusNotFound, http.MethodPut, sess+"invocationSteps/", map[string]any{
		"invocationIdentifier": "00000000-0000-0000-0000-000000000000", "invocationStepTime": "2025-01-01T00:00:00Z",
		"payload": map[string]any{"contentBlocks": []any{map[string]any{"text": "x"}}},
	})
	got := barJSON(t, ts, http.StatusOK, http.MethodPost, sess+"invocationSteps/"+step, map[string]any{"invocationIdentifier": inv})["invocationStep"].(map[string]any)
	if got["payload"].(map[string]any)["contentBlocks"].([]any)[0].(map[string]any)["text"] != "hello" {
		t.Fatalf("unexpected step: %v", got)
	}
	if steps := barJSON(t, ts, http.StatusOK, http.MethodPost, sess+"invocationSteps/", map[string]any{"invocationIdentifier": inv})["invocationStepSummaries"].([]any); len(steps) != 1 {
		t.Fatalf("expected one step, got %v", steps)
	}
	if invs := barJSON(t, ts, http.StatusOK, http.MethodPost, sess+"invocations/", map[string]any{})["invocationSummaries"].([]any); len(invs) != 1 {
		t.Fatalf("expected one invocation, got %v", invs)
	}

	// Tags on the session ARN.
	tags := "/tags/" + url.PathEscape(arn)
	barJSON(t, ts, http.StatusOK, http.MethodPost, tags, map[string]any{"tags": map[string]any{"env": "test"}})
	if got := barJSON(t, ts, http.StatusOK, http.MethodGet, tags, nil)["tags"].(map[string]any); got["team"] != "core" || got["env"] != "test" {
		t.Fatalf("unexpected tags: %v", got)
	}
	barJSON(t, ts, http.StatusOK, http.MethodDelete, tags+"?tagKeys=env", nil)
	if got := barJSON(t, ts, http.StatusOK, http.MethodGet, tags, nil)["tags"].(map[string]any); len(got) != 1 {
		t.Fatalf("env should be untagged: %v", got)
	}
	barJSON(t, ts, http.StatusNotFound, http.MethodGet, "/tags/"+url.PathEscape("arn:aws:bedrock:us-east-1:123456789012:session/00000000-0000-0000-0000-000000000000"), nil)

	// An ACTIVE session can't be deleted; an ENDED one takes no new invocations.
	barJSON(t, ts, http.StatusConflict, http.MethodDelete, sess, nil)
	if ended := barJSON(t, ts, http.StatusOK, http.MethodPatch, "/sessions/"+id, nil); ended["sessionStatus"] != "ENDED" {
		t.Fatalf("EndSession: %v", ended)
	}
	barJSON(t, ts, http.StatusConflict, http.MethodPatch, "/sessions/"+id, nil)
	barJSON(t, ts, http.StatusConflict, http.MethodPut, sess+"invocations/", map[string]any{})
	if list := barJSON(t, ts, http.StatusOK, http.MethodPost, "/sessions/", map[string]any{})["sessionSummaries"].([]any); len(list) != 1 {
		t.Fatalf("expected one session, got %v", list)
	}
	barJSON(t, ts, http.StatusOK, http.MethodDelete, sess, nil)
	barJSON(t, ts, http.StatusNotFound, http.MethodGet, sess, nil)
	barJSON(t, ts, http.StatusBadRequest, http.MethodGet, "/sessions/not-a-uuid/", nil)
}

// A flow: Input -> Condition; "big" numbers go to a Prompt then Output A,
// everything else straight to Output B.
const barTestFlow = `{"nodes":[
 {"name":"in","type":"Input","outputs":[{"name":"document","type":"Number"}]},
 {"name":"check","type":"Condition","configuration":{"condition":{"conditions":[{"name":"big","expression":"value > 10"},{"name":"default"}]}}},
 {"name":"say","type":"Prompt","configuration":{"prompt":{"sourceConfiguration":{"inline":{"modelId":"anthropic.claude-v2","templateType":"TEXT","templateConfiguration":{"text":{"text":"Big number: {{value}}"}}}}}}},
 {"name":"outA","type":"Output"},
 {"name":"outB","type":"Output"}],
 "connections":[
 {"name":"c1","type":"Data","source":"in","target":"check"},
 {"name":"c2","type":"Conditional","source":"check","target":"say","configuration":{"conditional":{"condition":"big"}}},
 {"name":"c3","type":"Conditional","source":"check","target":"outB","configuration":{"conditional":{"condition":"default"}}},
 {"name":"c4","type":"Data","source":"say","target":"outA"}]}`

func TestBedrockAgentRuntimeStage2Flows(t *testing.T) {
	ts := newBedrockAgentTestServer(t)
	flow := baWant(t, ts, http.StatusCreated, http.MethodPost, "/flows/", fmt.Sprintf(`{"name":"router","executionRoleArn":"arn:aws:iam::123456789012:role/flow","definition":%s}`, barTestFlow))
	flowID := flow["id"].(string)
	invoke := "/flows/" + flowID + "/aliases/TSTALIASID"
	input := func(v any) map[string]any {
		return map[string]any{"inputs": []any{map[string]any{"nodeName": "in", "nodeOutputName": "document", "content": map[string]any{"document": v}}}, "enableTrace": true}
	}

	// Not prepared yet.
	barJSON(t, ts, http.StatusConflict, http.MethodPost, invoke, input(1))
	baWant(t, ts, http.StatusAccepted, http.MethodPost, "/flows/"+flowID+"/", "")

	hdr, types, payloads := barStream(t, ts, http.MethodPost, invoke, input(42))
	if hdr.Get("x-amz-bedrock-flow-execution-id") == "" || types[len(types)-1] != "flowCompletionEvent" {
		t.Fatalf("unexpected flow stream: %v %v", hdr, types)
	}
	var outputs []map[string]any
	for i, ty := range types {
		if ty == "flowOutputEvent" {
			outputs = append(outputs, payloads[i])
		}
	}
	if len(outputs) != 1 || outputs[0]["nodeName"] != "outA" || outputs[0]["content"].(map[string]any)["document"] != "Big number: 42" {
		t.Fatalf("42 should take the prompt branch: %v", outputs)
	}
	_, types, payloads = barStream(t, ts, http.MethodPost, invoke, map[string]any{"inputs": input(3)["inputs"]})
	if strings.Join(types, ",") != "flowOutputEvent,flowCompletionEvent" || payloads[0]["nodeName"] != "outB" {
		t.Fatalf("3 should take the default branch: %v %v", types, payloads)
	}
	barJSON(t, ts, http.StatusBadRequest, http.MethodPost, invoke, map[string]any{"inputs": []any{map[string]any{"nodeName": "say", "content": map[string]any{"document": 1}}}})
	barJSON(t, ts, http.StatusNotFound, http.MethodPost, "/flows/"+flowID+"/aliases/NOPE000000", input(1))

	// Asynchronous executions.
	start := invoke + "/executions"
	arn := barJSON(t, ts, http.StatusOK, http.MethodPost, start, map[string]any{"inputs": input(50)["inputs"], "flowExecutionName": "run-1"})["executionArn"].(string)
	barJSON(t, ts, http.StatusConflict, http.MethodPost, start, map[string]any{"inputs": input(50)["inputs"], "flowExecutionName": "run-1"})
	exec := start + "/run-1"
	if got := barJSON(t, ts, http.StatusOK, http.MethodGet, exec, nil); got["status"] != "Succeeded" || got["executionArn"] != arn {
		t.Fatalf("unexpected execution: %v", got)
	}
	barJSON(t, ts, http.StatusConflict, http.MethodPost, exec+"/stop", nil)
	flowEvents := barJSON(t, ts, http.StatusOK, http.MethodGet, exec+"/events?eventType=Flow", nil)["flowExecutionEvents"].([]any)
	if len(flowEvents) != 2 {
		t.Fatalf("expected flow input and output events, got %v", flowEvents)
	}
	if nodeEvents := barJSON(t, ts, http.StatusOK, http.MethodGet, exec+"/events?eventType=Node", nil)["flowExecutionEvents"].([]any); len(nodeEvents) == 0 {
		t.Fatal("expected node events")
	}
	barJSON(t, ts, http.StatusBadRequest, http.MethodGet, exec+"/events?eventType=Other", nil)
	snap := barJSON(t, ts, http.StatusOK, http.MethodGet, exec+"/flowsnapshot", nil)
	var def map[string]any
	if err := json.Unmarshal([]byte(snap["definition"].(string)), &def); err != nil || len(def["nodes"].([]any)) != 5 {
		t.Fatalf("snapshot definition: %v %v", err, snap)
	}

	// A fresh execution can be stopped once.
	barJSON(t, ts, http.StatusOK, http.MethodPost, start, map[string]any{"inputs": input(1)["inputs"], "flowExecutionName": "run-2"})
	if stopped := barJSON(t, ts, http.StatusOK, http.MethodPost, start+"/run-2/stop", nil); stopped["status"] != "Aborted" {
		t.Fatalf("StopFlowExecution: %v", stopped)
	}
	list := barJSON(t, ts, http.StatusOK, http.MethodGet, "/flows/"+flowID+"/executions", nil)["flowExecutionSummaries"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected two executions, got %v", list)
	}
	barJSON(t, ts, http.StatusNotFound, http.MethodGet, start+"/missing", nil)
}
