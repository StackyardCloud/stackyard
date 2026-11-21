package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestBedrockRuntimeGuardrailRegexes(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	regexGuardrail := func(pattern, action string) (int, []byte) {
		status, _, raw := brtCall(t, ts, http.MethodPost, "/", map[string]any{"name": "r", "sensitiveInformationPolicyConfig": map[string]any{
			"regexesConfig": []any{map[string]any{"name": "num", "pattern": pattern, "action": action}}}},
			map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Bedrock_20230420.CreateGuardrail"})
		return status, raw
	}
	for _, c := range []struct{ pattern, action string }{{`\d*`, "ANONYMIZE"}, {`\d*`, "BLOCK"}, {"(", "BLOCK"}, {"", "BLOCK"}} {
		if status, raw := regexGuardrail(c.pattern, c.action); status != http.StatusBadRequest || !strings.Contains(string(raw), "ValidationException") {
			t.Fatalf("pattern %q %s: %d %s", c.pattern, c.action, status, raw)
		}
	}
	status, raw := regexGuardrail(`\d+`, "ANONYMIZE")
	id := brtExpect(t, status, raw, http.StatusOK, "")["guardrailId"].(string)
	status, _, raw = brtCall(t, ts, http.MethodPost, "/guardrail/"+id+"/version/DRAFT/apply",
		map[string]any{"source": "INPUT", "content": []any{map[string]any{"text": map[string]any{"text": "a1b22c333"}}}}, nil)
	out := brtExpect(t, status, raw, http.StatusOK, "")
	regexes := out["assessments"].([]any)[0].(map[string]any)["sensitiveInformationPolicy"].(map[string]any)["regexes"].([]any)
	if out["outputs"].([]any)[0].(map[string]any)["text"] != "a{num}b{num}c{num}" || len(regexes) != 3 {
		t.Fatalf("multiple matches: %s", raw)
	}

	// Even if an empty-matching pattern reached evaluation, it must not grow
	// or block the text.
	cfg := map[string]any{"sensitiveInformationPolicyConfig": map[string]any{"regexesConfig": []any{map[string]any{"name": "e", "pattern": `\d*`, "action": "ANONYMIZE"}}}}
	for _, action := range []string{"ANONYMIZE", "BLOCK"} {
		cfg["sensitiveInformationPolicyConfig"].(map[string]any)["regexesConfig"].([]any)[0].(map[string]any)["action"] = action
		res := brtEvaluateGuardrail(cfg, []*regexp.Regexp{regexp.MustCompile(`\d*`)}, []string{"no digits here"}, "INPUT")
		if res.intervened || len(res.outputs) != 0 {
			t.Fatalf("%s empty matches intervened: %+v", action, res)
		}
	}
	res := brtEvaluateGuardrail(cfg, []*regexp.Regexp{regexp.MustCompile(`\d*`)}, []string{"x7y"}, "INPUT")
	if !res.intervened || res.outputs[0] != "Sorry, the model cannot answer this question." {
		t.Fatalf("BLOCK on a real match: %+v", res)
	}
	cfg["sensitiveInformationPolicyConfig"].(map[string]any)["regexesConfig"].([]any)[0].(map[string]any)["action"] = "ANONYMIZE"
	res = brtEvaluateGuardrail(cfg, []*regexp.Regexp{regexp.MustCompile(`\d*`)}, []string{"x7y"}, "INPUT")
	if res.outputs[0] != "x{e}y" {
		t.Fatalf("ANONYMIZE masks only the real match: %+v", res)
	}
}

func TestBedrockGuardrailControlPlaneRecords(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	call := func(action string, payload map[string]any) (int, map[string]any) {
		status, _, raw := brtCall(t, ts, http.MethodPost, "/", payload,
			map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "Bedrock_20230420." + action})
		return status, brtJSONBody(t, raw)
	}
	_, created := call("CreateGuardrail", map[string]any{"name": "g1", "blockedInputMessaging": "in", "blockedOutputsMessaging": "out",
		"wordPolicyConfig": map[string]any{"wordsConfig": []any{map[string]any{"text": "x"}}}})
	id := created["guardrailId"].(string)
	status, got := call("GetGuardrail", map[string]any{"guardrailIdentifier": created["guardrailArn"]})
	if status != http.StatusOK || got["guardrailId"] != id || got["name"] != "g1" || got["wordPolicy"] == nil {
		t.Fatalf("get: %d %v", status, got)
	}
	call("CreateGuardrailVersion", map[string]any{"guardrailIdentifier": id})
	status, list := call("ListGuardrails", map[string]any{"guardrailIdentifier": id})
	versions := list["guardrails"].([]any)
	if status != http.StatusOK || len(versions) != 2 || versions[0].(map[string]any)["version"] != "DRAFT" || versions[1].(map[string]any)["version"] != "1" {
		t.Fatalf("list versions: %v", list)
	}
	if _, all := call("ListGuardrails", map[string]any{}); len(all["guardrails"].([]any)) != 1 {
		t.Fatalf("list all: %v", all)
	}
	if status, upd := call("UpdateGuardrail", map[string]any{"guardrailIdentifier": id, "name": "g2"}); status != http.StatusOK || upd["guardrailArn"] != created["guardrailArn"] {
		t.Fatalf("update: %d %v", status, upd)
	}
	if _, got := call("GetGuardrail", map[string]any{"guardrailIdentifier": id}); got["name"] != "g2" {
		t.Fatalf("renamed: %v", got)
	}
	for _, c := range []struct {
		action  string
		payload map[string]any
		want    int
	}{
		{"UpdateGuardrail", map[string]any{"guardrailIdentifier": "missing"}, http.StatusNotFound},
		{"DeleteGuardrail", map[string]any{"guardrailIdentifier": "missing"}, http.StatusNotFound},
		{"DeleteGuardrail", map[string]any{"guardrailIdentifier": id, "guardrailVersion": "7"}, http.StatusNotFound},
		{"GetGuardrail", map[string]any{"guardrailIdentifier": id, "guardrailVersion": "9"}, http.StatusNotFound},
		{"GetGuardrail", map[string]any{}, http.StatusBadRequest},
		{"UpdateGuardrail", map[string]any{"guardrailIdentifier": id, "sensitiveInformationPolicyConfig": map[string]any{"regexesConfig": []any{map[string]any{"pattern": "a*"}}}}, http.StatusBadRequest},
	} {
		if status, body := call(c.action, c.payload); status != c.want {
			t.Fatalf("%s %v: %d %v", c.action, c.payload, status, body)
		}
	}
	if status, _ := call("DeleteGuardrail", map[string]any{"guardrailIdentifier": id}); status != http.StatusOK {
		t.Fatalf("delete: %d", status)
	}
	if status, _ := call("GetGuardrail", map[string]any{"guardrailIdentifier": id}); status != http.StatusNotFound {
		t.Fatalf("get after delete: %d", status)
	}
}

func TestBedrockRuntimeBodyLimitAndConverseRules(t *testing.T) {
	_, ts := newBedrockRuntimeTestServer(t)
	big := bytes.Repeat([]byte(" "), brtMaxBody+1)
	status, _, raw := brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "invoke"), big, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("not.a-model", "invoke"), big, nil)
	if out := brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException"); !strings.Contains(out["message"].(string), "model identifier") {
		t.Fatalf("model checked before size: %s", raw)
	}

	// A declared body beyond what SigV4 buffers is still a 400, not a signature error.
	status, _, raw = brtCall(t, ts, http.MethodPost, brtModelPath("amazon.nova-lite-v1:0", "invoke"), bytes.Repeat([]byte(" "), 27_000_000), nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")

	// A chunked body (no Content-Length) is caught by the length of what was read.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+brtModelPath("amazon.nova-lite-v1:0", "invoke"), io.MultiReader(bytes.NewReader(big)))
	brtSign(req, "us-east-1", "UNSIGNED-PAYLOAD")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw = mustBody(t, resp)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(raw), "exceeds the maximum size") {
		t.Fatalf("chunked oversize: %d %s", resp.StatusCode, raw)
	}

	path := brtModelPath("amazon.nova-lite-v1:0", "converse")
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"messages": []any{brtUser("hi")}, "inferenceConfig": map[string]any{"maxTokens": nil}}, nil)
	brtExpect(t, status, raw, http.StatusOK, "")
	use := func(id string) map[string]any {
		return map[string]any{"toolUse": map[string]any{"toolUseId": id, "name": "get_weather", "input": map[string]any{}}}
	}
	result := func(id string) map[string]any {
		return map[string]any{"toolResult": map[string]any{"toolUseId": id, "content": []any{map[string]any{"text": "ok"}}}}
	}
	conv := []any{brtUser("a"), map[string]any{"role": "assistant", "content": []any{use("t1")}},
		map[string]any{"role": "user", "content": []any{result("t1")}}, map[string]any{"role": "assistant", "content": []any{use("t1")}}}
	status, _, raw = brtCall(t, ts, http.MethodPost, path, map[string]any{"messages": conv, "toolConfig": brtWeatherTools}, nil)
	brtExpect(t, status, raw, http.StatusBadRequest, "ValidationException")
}

// brtSign signs req for the bedrock service in region.
func brtSign(req *http.Request, region, payloadHash string) {
	req.Header.Set("Content-Type", "application/json")
	req.Host = req.URL.Host
	amzDate := time.Now().UTC().Format("20060102T150405Z")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)
	signed := buildSignedHeaders(req)
	canonical, _ := buildCanonicalRequest(req, signed, payloadHash, false)
	signature := signString(buildStringToSign(amzDate, region, "bedrock", canonical), region, "bedrock", testSecretKey)
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/bedrock/aws4_request, SignedHeaders=%s, Signature=%s",
		testAccessKey, amzDate[:8], region, signed, signature))
}

// brtSignedInRegion sends a bedrock-signed request scoped to another region.
func brtSignedInRegion(t *testing.T, ts *httptest.Server, method, path, region string, payload any) []byte {
	t.Helper()
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, ts.URL+path, bytes.NewReader(body))
	brtSign(req, region, sha256Hex(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return mustBody(t, resp)
}

func TestBedrockRuntimeAsyncStore(t *testing.T) {
	srv, ts := newBedrockRuntimeTestServer(t)
	if _, err := srv.s3.CreateBucket("eu-out"); err != nil {
		t.Fatal(err)
	}
	start := map[string]any{"modelId": "amazon.nova-reel-v1:0",
		"modelInput":       map[string]any{"taskType": "TEXT_VIDEO", "textToVideoParams": map[string]any{"text": "x"}},
		"outputDataConfig": map[string]any{"s3OutputDataConfig": map[string]any{"s3Uri": "s3://eu-out"}}}
	arn := brtJSONBody(t, brtSignedInRegion(t, ts, http.MethodPost, "/async-invoke", "eu-west-1", start))["invocationArn"].(string)
	if !strings.HasPrefix(arn, "arn:aws:bedrock:eu-west-1:123456789012:async-invoke/") {
		t.Fatalf("arn region: %s", arn)
	}
	for i := 0; i < 2; i++ {
		brtCall(t, ts, http.MethodPost, "/async-invoke", start, nil)
	}
	var seen []string
	next := ""
	for page := 0; page < 4; page++ {
		path := "/async-invoke?maxResults=1&sortOrder=Ascending"
		if next != "" {
			path += "&nextToken=" + next
		}
		status, _, raw := brtCall(t, ts, http.MethodGet, path, nil, nil)
		out := brtExpect(t, status, raw, http.StatusOK, "")
		for _, s := range out["asyncInvokeSummaries"].([]any) {
			seen = append(seen, s.(map[string]any)["invocationArn"].(string))
		}
		next, _ = out["nextToken"].(string)
		if next == "" {
			break
		}
	}
	if len(seen) != 3 || seen[0] != arn || seen[1] == seen[2] {
		t.Fatalf("cursor pages: %v", seen)
	}

	st := newBedrockRuntimeStore()
	for i := 0; i <= brtMaxAsyncInvokes; i++ {
		st.insertLocked(&brtAsync{arn: fmt.Sprintf("a%d", i), token: fmt.Sprintf("t%d", i)})
	}
	if len(st.invokes) != brtMaxAsyncInvokes || st.byARN["a0"] != nil || st.byToken["t0"] != nil || st.byARN["a1"] == nil {
		t.Fatalf("cap: %d invokes", len(st.invokes))
	}
}

func TestBedrockRuntimeRouteNonInterference(t *testing.T) {
	auth := "AWS4-HMAC-SHA256 Credential=stackyard/20260101/us-east-1/bedrock/aws4_request, SignedHeaders=host, Signature=x"
	for _, c := range []struct{ method, path, target string }{
		{http.MethodPost, "/guardrails", ""},
		{http.MethodGet, "/agents/agent-1", ""},
		{http.MethodPost, "/knowledgebases/kb-1/retrieve", ""},
		{http.MethodPost, "/agents/a/agentAliases/b/sessions/c/text", ""},
		{http.MethodPost, "/", "Bedrock_20230420.ListGuardrails"},
		{http.MethodGet, "/model/amazon.nova-lite-v1:0/invoke", ""},
	} {
		r := httptest.NewRequest(c.method, c.path, nil)
		r.Header.Set("Authorization", auth)
		if c.target != "" {
			r.Header.Set("X-Amz-Target", c.target)
		}
		if op, _, ok := bedrockRuntimeRoute(r); ok {
			t.Fatalf("%s %s claimed as %s", c.method, c.path, op)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/model/amazon.nova-lite-v1:0/converse", nil)
	if _, _, ok := bedrockRuntimeRoute(r); ok {
		t.Fatalf("unsigned non-runtime-host request claimed")
	}
}
