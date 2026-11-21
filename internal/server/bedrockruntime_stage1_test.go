package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
)

func brtControl(t *testing.T, ts *httptest.Server, action string, payload map[string]any) map[string]any {
	t.Helper()
	status, _, raw := brtCall(t, ts, http.MethodPost, "/", payload,
		map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Bedrock_20230420." + action})
	return brtExpect(t, status, raw, http.StatusOK, "")
}

func brtUser(text string) map[string]any {
	return map[string]any{"role": "user", "content": []any{map[string]any{"text": text}}}
}

var brtWeatherTools = map[string]any{"tools": []any{map[string]any{"toolSpec": map[string]any{"name": "get_weather", "inputSchema": map[string]any{"json": map[string]any{"type": "object"}}}}}}

func TestBedrockRuntimeConverse(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	path := brtModelPath("amazon.nova-pro-v1:0", "converse")
	status, _, raw := brtCall(t, ts, http.MethodPost, path, map[string]any{
		"system": []any{map[string]any{"text": "be kind"}}, "messages": []any{brtUser("hello converse")},
		"inferenceConfig": map[string]any{"maxTokens": 50, "temperature": 0.5}, "additionalModelResponseFieldPaths": []any{"/stop_sequence"},
		"performanceConfig": map[string]any{"latency": "optimized"},
	}, nil)
	out := brtExpect(t, status, raw, http.StatusOK, "")
	usage := out["usage"].(map[string]any)
	if out["stopReason"] != "end_turn" || usage["inputTokens"].(float64) != 4 || out["metrics"] == nil || out["performanceConfig"] == nil {
		t.Fatalf("converse body %s", raw)
	}
	if _, ok := out["additionalModelResponseFields"].(map[string]any)["stop_sequence"]; !ok {
		t.Fatalf("additional fields: %s", raw)
	}

	// The model calls a tool the user names, then answers its result.
	conv := []any{brtUser("what is get_weather in Paris")}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"messages": conv, "toolConfig": brtWeatherTools}, nil)
	out = brtExpect(t, status, raw, http.StatusOK, "")
	msg := out["output"].(map[string]any)["message"].(map[string]any)
	use := msg["content"].([]any)[0].(map[string]any)["toolUse"].(map[string]any)
	if out["stopReason"] != "tool_use" || use["name"] != "get_weather" {
		t.Fatalf("tool use: %s", raw)
	}
	conv = append(conv, msg, map[string]any{"role": "user", "content": []any{map[string]any{"toolResult": map[string]any{
		"toolUseId": use["toolUseId"], "content": []any{map[string]any{"text": "sunny"}}}}}})
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"messages": conv, "toolConfig": brtWeatherTools}, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["stopReason"] != "end_turn" {
		t.Fatalf("after tool result: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"messages": conv}, nil)
	if out := brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException"); !strings.Contains(out["message"].(string), "toolConfig") {
		t.Fatalf("missing toolConfig: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"messages": []any{brtUser("weather")},
		"toolConfig": map[string]any{"tools": brtWeatherTools["tools"], "toolChoice": map[string]any{"tool": map[string]any{"name": "get_weather"}}}}, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["stopReason"] != "tool_use" {
		t.Fatalf("forced tool: %s", raw)
	}

	assistant := map[string]any{"role": "assistant", "content": []any{map[string]any{"text": "ok"}}}
	badResult := map[string]any{"role": "user", "content": []any{map[string]any{"toolResult": map[string]any{"toolUseId": "nope", "content": []any{map[string]any{"text": "x"}}}}}}
	for name, body := range map[string]map[string]any{
		"empty":          {"messages": []any{}},
		"assistant":      {"messages": []any{assistant}},
		"not alternate":  {"messages": []any{brtUser("a"), brtUser("b")}},
		"blank text":     {"messages": []any{brtUser("  ")}},
		"two members":    {"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"text": "a", "image": map[string]any{}}}}}},
		"bad image":      {"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"image": map[string]any{"format": "bmp", "source": map[string]any{"bytes": "AA=="}}}}}}},
		"orphan result":  {"messages": []any{brtUser("a"), assistant, badResult}, "toolConfig": brtWeatherTools},
		"temperature":    {"messages": []any{brtUser("a")}, "inferenceConfig": map[string]any{"temperature": 2}},
		"max tokens":     {"messages": []any{brtUser("a")}, "inferenceConfig": map[string]any{"maxTokens": 0}},
		"dup tools":      {"messages": []any{brtUser("a")}, "toolConfig": map[string]any{"tools": []any{brtWeatherTools["tools"].([]any)[0], brtWeatherTools["tools"].([]any)[0]}}},
		"unknown choice": {"messages": []any{brtUser("a")}, "toolConfig": map[string]any{"tools": brtWeatherTools["tools"], "toolChoice": map[string]any{"tool": map[string]any{"name": "other"}}}},
		"prompt vars":    {"messages": []any{brtUser("a")}, "promptVariables": map[string]any{"x": map[string]any{"text": "y"}}},
		"latency":        {"messages": []any{brtUser("a")}, "performanceConfig": map[string]any{"latency": "warp"}},
	} {
		status, _, raw := brtCall(t, ts, http.MethodPost, path, body, nil)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", name, status, raw)
		}
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.titan-embed-text-v2:0", "converse"), map[string]any{"messages": []any{brtUser("a")}}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeConverseStream(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath("anthropic.claude-3-haiku-20240307-v1:0", "converse-stream"),
		map[string]any{"messages": []any{brtUser("stream this reply please now")}}, nil)
	if status != http.StatusOK {
		t.Fatalf("converse stream: %d %s", status, raw)
	}
	types, payloads := brtFrames(t, raw)
	if types[0] != "messageStart" || types[len(types)-2] != "messageStop" || types[len(types)-1] != "metadata" {
		t.Fatalf("event types %v", types)
	}
	if payloads[len(payloads)-1]["usage"] == nil {
		t.Fatalf("metadata %v", payloads[len(payloads)-1])
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "converse-stream"),
		map[string]any{"messages": []any{brtUser("get_weather now")}, "toolConfig": brtWeatherTools}, nil)
	types, _ = brtFrames(t, raw)
	if status != http.StatusOK || types[1] != "contentBlockStart" {
		t.Fatalf("tool stream %d %v", status, types)
	}
}

func TestBedrockRuntimeCountTokens(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	path := brtModelPath("anthropic.claude-3-5-haiku-20241022-v1:0", "count-tokens")
	status, _, raw := brtCall(t, ts, http.MethodPost, path, map[string]any{"input": map[string]any{"converse": map[string]any{"messages": []any{brtUser("count these four words")}}}}, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["inputTokens"].(float64) != 4 {
		t.Fatalf("count: %s", raw)
	}
	inner, _ := json.Marshal(map[string]any{"anthropic_version": "bedrock-2023-05-31", "max_tokens": 5, "messages": []any{map[string]any{"role": "user", "content": "two words"}}})
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"input": map[string]any{"invokeModel": map[string]any{"body": base64.StdEncoding.EncodeToString(inner)}}}, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["inputTokens"].(float64) != 2 {
		t.Fatalf("count invoke: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"input": map[string]any{}}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "count-tokens"), map[string]any{"input": map[string]any{"converse": map[string]any{"messages": []any{brtUser("x")}}}}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeGuardrails(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	created := brtControl(t, ts, "CreateGuardrail", map[string]any{
		"name": "g", "blockedInputMessaging": "Blocked by Stackyard.", "blockedOutputsMessaging": "Output blocked.",
		"wordPolicyConfig":                 map[string]any{"wordsConfig": []any{map[string]any{"text": "forbidden"}}},
		"topicPolicyConfig":                map[string]any{"topicsConfig": []any{map[string]any{"name": "investing", "definition": "advice", "type": "DENY"}}},
		"sensitiveInformationPolicyConfig": map[string]any{"piiEntitiesConfig": []any{map[string]any{"type": "EMAIL", "action": "ANONYMIZE"}}},
	})
	id := created["guardrailId"].(string)
	apply := func(version, source, text string) (int, map[string]any) {
		status, _, raw := brtCall(t, ts, http.MethodPost, "/guardrail/"+id+"/version/"+version+"/apply",
			map[string]any{"source": source, "content": []any{map[string]any{"text": map[string]any{"text": text}}}}, nil)
		return status, brtJSONBody(t, raw)
	}
	if status, out := apply("DRAFT", "INPUT", "this is forbidden"); status != http.StatusOK || out["action"] != "GUARDRAIL_INTERVENED" ||
		out["outputs"].([]any)[0].(map[string]any)["text"] != "Blocked by Stackyard." {
		t.Fatalf("word block: %d %v", status, out)
	}
	if _, out := apply("DRAFT", "OUTPUT", "talk about investing"); out["outputs"].([]any)[0].(map[string]any)["text"] != "Output blocked." {
		t.Fatalf("topic block: %v", out)
	}
	if _, out := apply("DRAFT", "INPUT", "mail me at a@b.com"); out["outputs"].([]any)[0].(map[string]any)["text"] != "mail me at {EMAIL}" {
		t.Fatalf("anonymize: %v", out)
	}
	if _, out := apply("DRAFT", "INPUT", "hello"); out["action"] != "NONE" || len(out["outputs"].([]any)) != 0 {
		t.Fatalf("clean: %v", out)
	}
	if status, _ := apply("1", "INPUT", "hello"); status != http.StatusNotFound {
		t.Fatalf("missing version: %d", status)
	}
	brtControl(t, ts, "CreateGuardrailVersion", map[string]any{"guardrailIdentifier": id})
	if status, _ := apply("1", "INPUT", "hello"); status != http.StatusOK {
		t.Fatalf("version 1: %d", status)
	}
	status, _, raw := brtCall(t, ts, http.MethodPost, "/guardrail/missing/version/DRAFT/apply", map[string]any{"source": "INPUT", "content": []any{map[string]any{"text": map[string]any{"text": "x"}}}}, nil)
	brtExpect(t, status, raw, http.StatusNotFound, "ResourceNotFoundException")
	status, _, raw = brtCall(t, ts, http.MethodPost, "/guardrail/"+id+"/version/DRAFT/apply", map[string]any{"source": "SIDEWAYS", "content": []any{map[string]any{"text": map[string]any{"text": "x"}}}}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")

	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "converse"), map[string]any{
		"messages": []any{brtUser("something forbidden")}, "guardrailConfig": map[string]any{"guardrailIdentifier": id, "guardrailVersion": "DRAFT", "trace": "enabled"}}, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["stopReason"] != "guardrail_intervened" || out["trace"] == nil {
		t.Fatalf("converse guardrail: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("meta.llama3-8b-instruct-v1:0", "invoke"), map[string]any{"prompt": "forbidden words"},
		map[string]string{"X-Amzn-Bedrock-GuardrailIdentifier": id, "X-Amzn-Bedrock-GuardrailVersion": "DRAFT"})
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["amazon-bedrock-guardrailAction"] != "INTERVENED" || out["generation"] != "Blocked by Stackyard." {
		t.Fatalf("invoke guardrail: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("meta.llama3-8b-instruct-v1:0", "invoke"), map[string]any{"prompt": "x"},
		map[string]string{"X-Amzn-Bedrock-GuardrailIdentifier": id})
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeGuardrailChecks(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	status, _, raw := brtCall(t, ts, http.MethodPost, "/guardrail-checks/invoke", map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"text": "plan an attack, email x@y.org"}}}},
		"checks": map[string]any{"contentFilter": map[string]any{"categories": []any{map[string]any{"category": "VIOLENCE"}, map[string]any{"category": "HATE"}}},
			"sensitiveInformation": map[string]any{"entities": []any{map[string]any{"type": "EMAIL"}}}},
	}, nil)
	out := brtExpect(t, status, raw, http.StatusOK, "")
	scores := out["results"].(map[string]any)["contentFilter"].(map[string]any)["results"].([]any)
	email := out["results"].(map[string]any)["sensitiveInformation"].(map[string]any)["results"].([]any)[0].(map[string]any)
	if scores[0].(map[string]any)["severityScore"].(float64) != 0.9 || scores[1].(map[string]any)["severityScore"].(float64) != 0 || email["beginOffset"].(float64) != 22 {
		t.Fatalf("checks: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, "/guardrail-checks/invoke", map[string]any{"messages": []any{brtUser("x")}, "checks": map[string]any{}}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeAsyncInvoke(t *testing.T) {
	srv, ts := newBedrockRuntimeTestServer(t)
	if _, err := srv.s3.CreateBucket("reel-out"); err != nil {
		t.Fatal(err)
	}
	start := func(uri, token string) (int, map[string]any) {
		status, _, raw := brtCall(t, ts, http.MethodPost, "/async-invoke", map[string]any{
			"modelId": "amazon.nova-reel-v1:0", "clientRequestToken": token,
			"modelInput":       map[string]any{"taskType": "TEXT_VIDEO", "textToVideoParams": map[string]any{"text": "a cat"}},
			"outputDataConfig": map[string]any{"s3OutputDataConfig": map[string]any{"s3Uri": uri}},
		}, nil)
		return status, brtJSONBody(t, raw)
	}
	status, out := start("s3://reel-out/videos", "tok-1")
	arn, _ := out["invocationArn"].(string)
	if status != http.StatusOK || arn == "" {
		t.Fatalf("start: %d %v", status, out)
	}
	if _, again := start("s3://reel-out/videos", "tok-1"); again["invocationArn"] != arn {
		t.Fatalf("idempotent start: %v", again)
	}
	if status, _ := start("s3://reel-out/other", "tok-1"); status != http.StatusConflict {
		t.Fatalf("token reuse: %d", status)
	}
	status, _, raw := brtCall(t, ts, http.MethodGet, "/async-invoke/"+url.PathEscape(arn), nil, nil)
	if got := brtExpect(t, status, raw, http.StatusOK, ""); got["status"] != "Completed" {
		t.Fatalf("get: %s", raw)
	}
	id := arn[strings.LastIndex(arn, "/")+1:]
	if _, err := srv.s3.GetObject("reel-out", "videos/"+id+"/output.mp4"); err != nil {
		t.Fatalf("output object: %v", err)
	}
	if _, failed := start("s3://missing-bucket", ""); failed["invocationArn"] == nil {
		t.Fatalf("failed start: %v", failed)
	}
	status, _, raw = brtCall(t, ts, http.MethodGet, "/async-invoke?statusEquals=Failed", nil, nil)
	list := brtExpect(t, status, raw, http.StatusOK, "")["asyncInvokeSummaries"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["failureMessage"] == nil {
		t.Fatalf("failed list: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodGet, "/async-invoke?maxResults=1&sortOrder=Ascending", nil, nil)
	page := brtExpect(t, status, raw, http.StatusOK, "")
	if page["asyncInvokeSummaries"].([]any)[0].(map[string]any)["invocationArn"] != arn || page["nextToken"] == nil {
		t.Fatalf("page: %s", raw)
	}
	status, _, raw = brtCall(t, ts, http.MethodGet, "/async-invoke?sortOrder=Ascending&maxResults=1&nextToken="+page["nextToken"].(string), nil, nil)
	if next := brtExpect(t, status, raw, http.StatusOK, ""); len(next["asyncInvokeSummaries"].([]any)) != 1 || next["nextToken"] != nil {
		t.Fatalf("second page: %s", raw)
	}
	for name, q := range map[string]string{"status": "statusEquals=Done", "max": "maxResults=0", "order": "sortOrder=Up", "token": "nextToken=bad!"} {
		status, _, raw := brtCall(t, ts, http.MethodGet, "/async-invoke?"+q, nil, nil)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", name, status, raw)
		}
	}
	status, _, raw = brtCall(t, ts, http.MethodGet, "/async-invoke/"+url.PathEscape("arn:aws:bedrock:us-east-1:123456789012:async-invoke/none"), nil, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
	status, _, raw = brtCall(t, ts, http.MethodPost, "/async-invoke", map[string]any{"modelId": "amazon.nova-lite-v1:0", "modelInput": map[string]any{"taskType": "TEXT_VIDEO", "textToVideoParams": map[string]any{"text": "a cat"}},
		"outputDataConfig": map[string]any{"s3OutputDataConfig": map[string]any{"s3Uri": "s3://reel-out"}}}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func brtEventBody(t *testing.T, events ...map[string]any) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := eventstream.NewEncoder()
	for _, ev := range events {
		inner, _ := json.Marshal(map[string]any{"event": ev})
		payload, _ := json.Marshal(map[string]any{"bytes": base64.StdEncoding.EncodeToString(inner)})
		msg := eventstream.Message{Payload: payload}
		msg.Headers.Set(":message-type", eventstream.StringValue("event"))
		msg.Headers.Set(":event-type", eventstream.StringValue("chunk"))
		if err := enc.Encode(&buf, msg); err != nil {
			t.Fatal(err)
		}
	}
	return buf.Bytes()
}

func TestBedrockRuntimeBidirectionalStream(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	path := brtModelPath("amazon.nova-sonic-v1:0", "invoke-with-bidirectional-stream")
	body := brtEventBody(t, map[string]any{"sessionStart": map[string]any{}}, map[string]any{"promptStart": map[string]any{"promptName": "p1"}},
		map[string]any{"textInput": map[string]any{"content": "hello sonic"}}, map[string]any{"sessionEnd": map[string]any{}})
	status, _, raw := brtCall(t, ts, http.MethodPost, path, body, map[string]string{"Content-Type": "application/vnd.amazon.eventstream"})
	if status != http.StatusOK {
		t.Fatalf("bidi: %d %s", status, raw)
	}
	chunks := brtChunkBodies(t, raw)
	text := chunks[2]["event"].(map[string]any)["textOutput"].(map[string]any)["content"]
	if len(chunks) != 6 || text != "Stackyard response: hello sonic" {
		t.Fatalf("bidi output %v", chunks)
	}
	noStart := brtEventBody(t, map[string]any{"textInput": map[string]any{"content": "x"}}, map[string]any{"sessionEnd": map[string]any{}})
	for name, b := range map[string][]byte{"json body": []byte(`{"x":1}`), "no sessionStart": noStart} {
		status, _, raw := brtCall(t, ts, http.MethodPost, path, b, map[string]string{"Content-Type": "application/vnd.amazon.eventstream"})
		if status != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", name, status, raw)
		}
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "invoke-with-bidirectional-stream"), body, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeModelResolution(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	pm := brtControl(t, ts, "CreateProvisionedModelThroughput", map[string]any{"modelId": "arn:aws:bedrock:us-east-1::foundation-model/meta.llama3-8b-instruct-v1:0", "modelUnits": 1})
	aip := brtControl(t, ts, "CreateInferenceProfile", map[string]any{"modelSource": map[string]any{"copyFrom": "arn:aws:bedrock:us-east-1::foundation-model/meta.llama3-8b-instruct-v1:0"}})
	llama := map[string]any{"prompt": "hi"}
	for _, id := range []string{
		pm["provisionedModelArn"].(string), aip["inferenceProfileArn"].(string),
		"arn:aws:bedrock:us-east-1::foundation-model/meta.llama3-8b-instruct-v1:0",
		"arn:aws:bedrock:us-east-1:123456789012:inference-profile/us.meta.llama3-8b-instruct-v1:0",
		"arn:aws:bedrock:us-east-1:123456789012:custom-model/meta.llama3-8b-instruct-v1:0/abc",
		"arn:aws:bedrock:us-east-1:123456789012:imported-model/xyz",
	} {
		status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath(id, "invoke"), llama, nil)
		if out := brtExpect(t, status, raw, http.StatusOK, ""); out["generation"] == nil {
			t.Fatalf("%s: %s", id, raw)
		}
	}
	status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath("arn:aws:bedrock:us-east-1:123456789012:provisioned-model/none", "invoke"), llama, nil)
	brtExpect(t, status, raw, http.StatusNotFound, "ResourceNotFoundException")
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("arn:aws:s3:::bucket", "invoke"), llama, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}
