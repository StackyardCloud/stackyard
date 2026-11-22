package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	cxnTestTarget = "com.amazonaws.codeconnections.CodeConnections_20231201."
	csTestTarget  = "com.amazonaws.codestar.connections.CodeStar_connections_20191201."
)

func newCodeConnectionsTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// cxnCall sends one awsJson1_0 call signed for service and decodes the reply.
func cxnCall(t *testing.T, ts *httptest.Server, service, target, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": target + op,
	}, service)
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func cxnOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := cxnCall(t, ts, "codeconnections", cxnTestTarget, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func cxnErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := cxnCall(t, ts, "codeconnections", cxnTestTarget, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestCodeConnectionsStage0Catalog(t *testing.T) {
	if len(codeconnectionsOperations) != 27 || len(codestarconnectionsOperations) != 27 {
		t.Fatalf("operation counts: %d %d", len(codeconnectionsOperations), len(codestarconnectionsOperations))
	}
	for _, api := range []*cxnAPI{codeConnectionsAPI, codeStarConnectionsAPI} {
		model := api.model()
		for op := range api.knownOps {
			if _, ok := model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", api.service, op)
			}
		}
		if len(model.Operations) != len(api.knownOps) {
			t.Errorf("%s: model has %d operations, catalog %d", api.service, len(model.Operations), len(api.knownOps))
		}
	}
}

func TestCodeConnectionsStage0Routing(t *testing.T) {
	_, ts := newCodeConnectionsTestServer(t)
	// Short (SDK v1 style) target prefixes are accepted too.
	if status, out := cxnCall(t, ts, "codeconnections", "CodeConnections_20231201.", "ListConnections", `{}`); status != http.StatusOK {
		t.Fatalf("short target: %d %v", status, out)
	}
	if code := cxnErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	// Signed for the other name: SigV4 rejects the service mismatch.
	if status, _ := cxnCall(t, ts, "codestar-connections", cxnTestTarget, "ListConnections", `{}`); status != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", status)
	}
	if code := cxnErrCode(t, ts, "ListConnections", `{"MaxResults":"x"}`); code != "ValidationException" {
		t.Fatalf("bad MaxResults type: %s", code)
	}
	if code := cxnErrCode(t, ts, "ListConnections", `{"Bogus":1}`); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
	if code := cxnErrCode(t, ts, "ListConnections", `{`); code != "SerializationException" {
		t.Fatalf("bad JSON: %s", code)
	}
	// Bodies of unknown length are refused before anything reads them.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", cxnTestTarget+"ListConnections")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Content-Length") {
		t.Fatalf("chunked body: %d %s", resp.StatusCode, body)
	}
	// A plain S3 request still reaches S3.
	if resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/cc-bucket", nil, nil, "s3"); resp.StatusCode != http.StatusOK {
		t.Fatalf("S3 CreateBucket: %d", resp.StatusCode)
	}
}

func TestCodeConnectionsStage0StackyardEndpointsNeedSignature(t *testing.T) {
	_, ts := newCodeConnectionsTestServer(t)
	resp, err := http.Post(ts.URL+"/_stackyard/codeconnections/connections/x/activate", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("unsigned activate: %d", resp.StatusCode)
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/_stackyard/codeconnections/nope", []byte(`{}`), nil, "codeconnections")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown Stackyard endpoint: %d", resp.StatusCode)
	}
}
