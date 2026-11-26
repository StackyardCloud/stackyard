package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Stage 0 for MTurk, WorkDocs and re:Post Private: the catalogs match the CLI
// models, every operation has a handler, and routing leaves neighbouring
// services their traffic.

func collabCall(t *testing.T, ts *httptest.Server, service, method, path, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	h := map[string]string{"Content-Type": "application/json"}
	for k, v := range headers {
		h[k] = v
	}
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, h, service)
	return resp.StatusCode, msgDecode(t, resp)
}

func mtkCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	return collabCall(t, ts, mtkSigningName, http.MethodPost, "/", body, map[string]string{"Content-Type": mtkContentType, "X-Amz-Target": mtkTargetPrefix + op})
}

func mtkOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := mtkCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func mtkFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := mtkCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["TurkErrorCode"].(string)
	if code == "" {
		code, _ = out["__type"].(string)
	}
	return code
}

// mtkWorkerDo posts a simulated worker action.
func mtkWorkerDo(t *testing.T, ts *httptest.Server, action string, req map[string]any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(req)
	resp, err := http.Post(ts.URL+mtkLinkPrefix+action, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, msgDecode(t, resp)
}

func TestCollabStage0CatalogsMatchModels(t *testing.T) {
	check := func(name string, model *awsmodels.Model, ops []string, handled func(string) bool) {
		if len(ops) != len(model.Operations) {
			t.Fatalf("%s: catalog has %d operations, model %d", name, len(ops), len(model.Operations))
		}
		for _, op := range ops {
			if _, ok := model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", name, op)
			}
			if !handled(op) {
				t.Errorf("%s: %s has no handler", name, op)
			}
		}
	}
	var ops []string
	for _, op := range mturkOperations {
		ops = append(ops, op.Name)
	}
	check("mturk", awsmodels.MTurkModel(), ops, func(op string) bool { return mtkHandlers[op] != nil })
	ops = nil
	for _, op := range workDocsOperations {
		ops = append(ops, op.Name)
	}
	check("workdocs", awsmodels.WorkDocsModel(), ops, func(op string) bool { return wdocHandlers[op] != nil })
	ops = nil
	for _, op := range repostSpaceOperations {
		ops = append(ops, op.Name)
	}
	check("repostspace", awsmodels.RepostSpaceModel(), ops, func(op string) bool { return rpsHandlers[op] != nil })
}

func TestCollabStage0Routing(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	if out := mtkOK(t, ts, "GetAccountBalance", `{}`); out["AvailableBalance"] != mtkSandboxBalance {
		t.Fatalf("GetAccountBalance: %v", out)
	}
	if status, out := collabCall(t, ts, "workdocs", http.MethodGet, "/api/v1/users", "", nil); status != http.StatusOK || out["Users"] == nil {
		t.Fatalf("WorkDocs DescribeUsers: %d %v", status, out)
	}
	if status, out := collabCall(t, ts, "repostspace", http.MethodGet, "/spaces", "", nil); status != http.StatusOK || out["spaces"] == nil {
		t.Fatalf("re:Post Private ListSpaces: %d %v", status, out)
	}
	// Neighbours keep their traffic: other services' /tags routes, and
	// JSON services posting to /.
	if status, out := collabCall(t, ts, "mobiletargeting", http.MethodGet, "/v1/apps", "", nil); status != http.StatusOK || out["Item"] == nil {
		t.Fatalf("Pinpoint GetApps: %d %v", status, out)
	}
	if status, out := collabCall(t, ts, "ses", http.MethodGet, "/v1/email/configuration-sets", "", nil); status != http.StatusOK || out["ConfigurationSets"] == nil {
		t.Fatalf("Pinpoint Email ListConfigurationSets: %d %v", status, out)
	}
	if out := mmgrOK(t, ts, "ListArchives", `{}`); out["Archives"] == nil {
		t.Fatalf("Mail Manager ListArchives: %v", out)
	}
	arn := "arn:aws:mediastore:us-east-1:123456789012:container/none"
	if status, _ := collabCall(t, ts, "repostspace", http.MethodGet, "/tags/"+strings.ReplaceAll(arn, "/", "%2F"), "", nil); status != http.StatusNotFound {
		t.Fatalf("re:Post Private claims only its own ARNs: %d", status)
	}
	// Requests signed for another service aren't claimed.
	for _, path := range []string{"/api/v1/users", "/spaces"} {
		resp := signedRequestWithService(t, http.MethodGet, ts.URL+path, nil, nil, "s3")
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusOK || strings.Contains(body, "Users") || strings.Contains(body, "spaces") {
			t.Fatalf("s3-signed %s was served by WorkDocs or re:Post Private: %d %s", path, resp.StatusCode, body)
		}
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": mtkContentType, "X-Amz-Target": mtkTargetPrefix + "GetAccountBalance",
	}, "s3")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("s3-signed MTurk request: %d", resp.StatusCode)
	}
	// Chunked bodies are refused.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", mtkTargetPrefix+"GetAccountBalance")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	if status, out := mtkCall(t, ts, "NoSuchOperation", `{}`); status != http.StatusBadRequest || out["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown op: %d %v", status, out)
	}
	if code := mtkFail(t, ts, "GetHIT", `{"HITId":"ABC","Smuggled":1}`); code != "AWS.MechanicalTurk.ParameterValidationError" {
		t.Fatalf("unknown member: %s", code)
	}
	if status, out := collabCall(t, ts, "repostspace", http.MethodPost, "/spaces", `{"name":"x"}`, nil); status != http.StatusBadRequest || out["reason"] == nil {
		t.Fatalf("missing members: %d %v", status, out)
	}
}
