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

func newBedrockRuntimeTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// brtCall sends a Bedrock-signed request; body may be raw bytes or a value to
// JSON-encode.
func brtCall(t *testing.T, ts *httptest.Server, method, path string, body any, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	var raw []byte
	switch b := body.(type) {
	case nil:
	case []byte:
		raw = b
	default:
		raw, _ = json.Marshal(b)
	}
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok && raw != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "bedrock")
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header, mustBody(t, resp)
}

func brtJSONBody(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return out
}

func brtExpect(t *testing.T, status int, raw []byte, want int, code string) map[string]any {
	t.Helper()
	if status != want {
		t.Fatalf("status %d, want %d: %s", status, want, raw)
	}
	out := brtJSONBody(t, raw)
	if code != "" && out["__type"] != code {
		t.Fatalf("error %v, want %s: %s", out["__type"], code, raw)
	}
	return out
}

func brtModelPath(model, action string) string {
	return "/model/" + url.PathEscape(model) + "/" + action
}

// brtFrames decodes an event-stream response into (event type, payload).
func brtFrames(t *testing.T, raw []byte) ([]string, []map[string]any) {
	t.Helper()
	dec := eventstream.NewDecoder()
	r := bytes.NewReader(raw)
	var types []string
	var payloads []map[string]any
	for r.Len() > 0 {
		msg, err := dec.Decode(r, nil)
		if err != nil {
			t.Fatalf("decode frame: %v", err)
		}
		types = append(types, msg.Headers.Get(":event-type").String())
		payloads = append(payloads, brtJSONBody(t, msg.Payload))
	}
	return types, payloads
}

// brtChunkBodies unwraps InvokeModelWithResponseStream chunk payloads.
func brtChunkBodies(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	types, payloads := brtFrames(t, raw)
	var out []map[string]any
	for i, p := range payloads {
		if types[i] != "chunk" {
			t.Fatalf("event %d type %s, want chunk", i, types[i])
		}
		inner, err := base64.StdEncoding.DecodeString(p["bytes"].(string))
		if err != nil {
			t.Fatalf("chunk bytes: %v", err)
		}
		out = append(out, brtJSONBody(t, inner))
	}
	return out
}

func TestBedrockRuntimeOperations(t *testing.T) {
	if len(bedrockRuntimeOperations) != 11 || len(bedrockRuntimeOperationByName) != 11 {
		t.Fatalf("want 11 unique operations, got %d/%d", len(bedrockRuntimeOperations), len(bedrockRuntimeOperationByName))
	}
	for _, name := range []string{"InvokeModel", "Converse", "ApplyGuardrail", "StartAsyncInvoke", "InvokeModelWithBidirectionalStream"} {
		if _, ok := bedrockRuntimeOperationByName[name]; !ok {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestBedrockRuntimeRoutingLeavesControlPlane(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	status, _, raw := brtCall(t, ts, http.MethodPost, "/", map[string]any{},
		map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Bedrock_20230420.ListFoundationModels"})
	if status != http.StatusOK || !strings.Contains(string(raw), "modelSummaries") {
		t.Fatalf("control plane: %d %s", status, raw)
	}
	// An unknown runtime action is not claimed by the runtime router.
	status, _, _ = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "nope"), map[string]any{}, nil)
	if status == http.StatusOK {
		t.Fatalf("unknown model action answered 200")
	}
}

func TestBedrockRuntimeInvokeModelAnthropic(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	path := brtModelPath("us.anthropic.claude-3-5-sonnet-20241022-v2:0", "invoke")
	body := map[string]any{"anthropic_version": "bedrock-2023-05-31", "max_tokens": 100, "system": "Be brief.",
		"messages": []any{map[string]any{"role": "user", "content": "hello there model"}}}
	status, hdr, raw := brtCall(t, ts, http.MethodPost, path, body, nil)
	out := brtExpect(t, status, raw, http.StatusOK, "")
	content := out["content"].([]any)[0].(map[string]any)
	if out["type"] != "message" || out["stop_reason"] != "end_turn" || !strings.Contains(content["text"].(string), "hello there model") {
		t.Fatalf("unexpected body: %s", raw)
	}
	if hdr.Get("X-Amzn-Bedrock-Input-Token-Count") != "5" || hdr.Get("X-Amzn-Bedrock-Output-Token-Count") == "" {
		t.Fatalf("token headers: %v", hdr)
	}

	body["max_tokens"] = 2
	status, _, raw = brtCall(t, ts, http.MethodPost, path, body, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["stop_reason"] != "max_tokens" {
		t.Fatalf("want max_tokens: %s", raw)
	}
	body["max_tokens"], body["stop_sequences"] = 100, []any{"there"}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, body, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["stop_reason"] != "stop_sequence" || out["stop_sequence"] != "there" {
		t.Fatalf("want stop_sequence: %s", raw)
	}
	body["tools"] = []any{map[string]any{"name": "get_weather", "input_schema": map[string]any{"type": "object"}}}
	body["tool_choice"] = map[string]any{"type": "any"}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, body, nil)
	if out := brtExpect(t, status, raw, http.StatusOK, ""); out["stop_reason"] != "tool_use" || !strings.Contains(string(raw), `"name":"get_weather"`) {
		t.Fatalf("want tool_use: %s", raw)
	}

	for name, bad := range map[string]map[string]any{
		"no version":      {"max_tokens": 10, "messages": []any{map[string]any{"role": "user", "content": "x"}}},
		"no max":          {"anthropic_version": "bedrock-2023-05-31", "messages": []any{map[string]any{"role": "user", "content": "x"}}},
		"extra key":       {"anthropic_version": "bedrock-2023-05-31", "max_tokens": 10, "foo": 1, "messages": []any{map[string]any{"role": "user", "content": "x"}}},
		"assistant first": {"anthropic_version": "bedrock-2023-05-31", "max_tokens": 10, "messages": []any{map[string]any{"role": "assistant", "content": "x"}}},
		"temperature":     {"anthropic_version": "bedrock-2023-05-31", "max_tokens": 10, "temperature": 3, "messages": []any{map[string]any{"role": "user", "content": "x"}}},
	} {
		status, _, raw := brtCall(t, ts, http.MethodPost, path, bad, nil)
		if status != http.StatusBadRequest || !strings.Contains(string(raw), "ValidationException") {
			t.Fatalf("%s: %d %s", name, status, raw)
		}
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("anthropic.claude-9", "invoke"), body, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
	status, _, raw = brtCall(t, ts, http.MethodPost, path, []byte("{not json"), nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeInvokeModelFamilies(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	cases := []struct {
		model string
		body  map[string]any
		check func(map[string]any) bool
	}{
		{"amazon.titan-text-express-v1", map[string]any{"inputText": "summarize this", "textGenerationConfig": map[string]any{"maxTokenCount": 50}},
			func(o map[string]any) bool {
				return o["results"].([]any)[0].(map[string]any)["completionReason"] == "FINISH"
			}},
		{"amazon.titan-embed-text-v2:0", map[string]any{"inputText": "embed me", "dimensions": 256},
			func(o map[string]any) bool { return len(o["embedding"].([]any)) == 256 }},
		{"amazon.nova-lite-v1:0", map[string]any{"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"text": "hi nova"}}}}},
			func(o map[string]any) bool { return o["stopReason"] == "end_turn" && o["usage"] != nil }},
		{"meta.llama3-8b-instruct-v1:0", map[string]any{"prompt": "tell me", "max_gen_len": 1},
			func(o map[string]any) bool { return o["stop_reason"] == "length" }},
		{"mistral.mistral-7b-instruct-v0:2", map[string]any{"prompt": "<s>[INST] hi [/INST]"},
			func(o map[string]any) bool { return o["outputs"].([]any)[0].(map[string]any)["stop_reason"] == "stop" }},
		{"cohere.command-r-v1:0", map[string]any{"message": "hello cohere"},
			func(o map[string]any) bool {
				return o["finish_reason"] == "COMPLETE" && len(o["chat_history"].([]any)) == 2
			}},
		{"cohere.embed-english-v3", map[string]any{"texts": []any{"a", "b"}, "input_type": "search_document"},
			func(o map[string]any) bool { return len(o["embeddings"].([]any)) == 2 }},
		{"cohere.embed-english-v3", map[string]any{"texts": []any{"a"}, "input_type": "clustering", "embedding_types": []any{"int8"}},
			func(o map[string]any) bool { return o["response_type"] == "embeddings_by_type" }},
		{"anthropic.claude-v2:1", map[string]any{"prompt": "\n\nHuman: hi\n\nAssistant:", "max_tokens_to_sample": 20},
			func(o map[string]any) bool { return o["type"] == "completion" && o["stop_reason"] == "stop_sequence" }},
	}
	for _, c := range cases {
		status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath(c.model, "invoke"), c.body, nil)
		if out := brtExpect(t, status, raw, http.StatusOK, ""); !c.check(out) {
			t.Fatalf("%s: unexpected body %s", c.model, raw)
		}
	}
	bad := map[string]map[string]any{
		"amazon.titan-text-express-v1":     {"textGenerationConfig": map[string]any{}},
		"amazon.titan-embed-text-v2:0":     {"inputText": "x", "dimensions": 300},
		"meta.llama3-8b-instruct-v1:0":     {"prompt": "x", "max_gen_len": 5000},
		"cohere.embed-english-v3":          {"texts": []any{"a"}, "input_type": "bogus"},
		"cohere.command-r-v1:0":            {"message": "x", "stop_sequences": []any{"a", "b", "c", "d", "e"}},
		"amazon.nova-lite-v1:0":            {"messages": []any{}},
		"mistral.mistral-7b-instruct-v0:2": {"prompt": "x", "top_k": 500},
		"anthropic.claude-v2":              {"prompt": "hi", "max_tokens_to_sample": 5},
	}
	for model, body := range bad {
		status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath(model, "invoke"), body, nil)
		brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
	}
	status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-reel-v1:0", "invoke"), map[string]any{}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "invoke"), map[string]any{"messages": []any{}},
		map[string]string{"X-Amzn-Bedrock-PerformanceConfig-Latency": "fastest"})
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

func TestBedrockRuntimeInvokeModelWithResponseStream(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	body := map[string]any{"anthropic_version": "bedrock-2023-05-31", "max_tokens": 100,
		"messages": []any{map[string]any{"role": "user", "content": "one two three four five six"}}}
	status, hdr, raw := brtCall(t, ts, http.MethodPost, brtModelPath("anthropic.claude-3-haiku-20240307-v1:0", "invoke-with-response-stream"), body, nil)
	if status != http.StatusOK || hdr.Get("Content-Type") != "application/vnd.amazon.eventstream" || hdr.Get("X-Amzn-Bedrock-Content-Type") != "application/json" {
		t.Fatalf("stream response %d %v", status, hdr)
	}
	chunks := brtChunkBodies(t, raw)
	var types []string
	text := ""
	for _, c := range chunks {
		types = append(types, c["type"].(string))
		if d, ok := c["delta"].(map[string]any); ok && d["text"] != nil {
			text += d["text"].(string)
		}
	}
	if types[0] != "message_start" || types[len(types)-1] != "message_stop" || !strings.Contains(text, "one two three") {
		t.Fatalf("stream events %v text %q", types, text)
	}
	if _, ok := chunks[len(chunks)-1]["amazon-bedrock-invocationMetrics"]; !ok {
		t.Fatalf("last chunk lacks invocation metrics: %v", chunks[len(chunks)-1])
	}
	for model, b := range map[string]map[string]any{
		"amazon.titan-text-lite-v1":          {"inputText": "stream me"},
		"meta.llama3-8b-instruct-v1:0":       {"prompt": "stream me"},
		"mistral.mixtral-8x7b-instruct-v0:1": {"prompt": "stream me"},
		"cohere.command-r-plus-v1:0":         {"message": "stream me"},
		"amazon.nova-micro-v1:0":             {"messages": []any{map[string]any{"role": "user", "content": []any{map[string]any{"text": "stream me"}}}}},
	} {
		status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath(model, "invoke-with-response-stream"), b, nil)
		if status != http.StatusOK || len(brtChunkBodies(t, raw)) < 2 {
			t.Fatalf("%s stream: %d", model, status)
		}
	}
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.titan-embed-text-v1", "invoke-with-response-stream"), map[string]any{"inputText": "x"}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}
