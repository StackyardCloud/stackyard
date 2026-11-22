package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newSageMakerRuntimeTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func sagemakerRuntimeRequest(t *testing.T, ts *httptest.Server, endpoint, suffix string, body []byte, headers map[string]string) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/endpoints/"+endpoint+"/"+suffix, body, headers, "sagemaker")
}

func TestSageMakerRuntimeStage0Catalog(t *testing.T) {
	if len(sagemakerruntimeOperations) != 3 || len(sagemakerruntimeOperationByName) != 3 {
		t.Fatalf("expected 3 unique SageMaker Runtime operations, got %d", len(sagemakerruntimeOperations))
	}
	for _, op := range []string{"InvokeEndpoint", "InvokeEndpointAsync", "InvokeEndpointWithResponseStream"} {
		if _, ok := sagemakerruntimeOperationByName[op]; !ok {
			t.Fatalf("missing operation %s", op)
		}
		found := false
		for _, mapped := range sagemakerRuntimeOpByPath {
			found = found || mapped == op
		}
		if !found {
			t.Fatalf("operation %s has no route", op)
		}
	}
}

func TestSageMakerRuntimeStage0RoutingKeepsControlPlane(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)

	// The control plane still answers alongside the runtime router.
	resp := sagemakerRequest(t, ts, "ListEndpoints", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var listed struct {
		Endpoints []struct{ EndpointName string } `json:"Endpoints"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listed); err != nil || len(listed.Endpoints) == 0 {
		t.Fatalf("ListEndpoints = %+v, err %v", listed, err)
	}

	// Signed for another service: the runtime router does not claim it.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/endpoints/stackyard-endpoint/invocations", []byte(`{}`), nil, "s3")
	if body := string(mustBody(t, resp)); strings.Contains(body, "ValidationError") || resp.Header.Get("x-Amzn-Invoked-Production-Variant") != "" || resp.Header.Get("X-Amzn-ErrorType") != "" {
		t.Fatalf("runtime router claimed an s3-signed request: %d %s", resp.StatusCode, body)
	}

	// Signed for sagemaker but with a bad signature: rejected by the runtime router.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/endpoints/stackyard-endpoint/invocations", strings.NewReader("x"))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20250101/us-east-1/sagemaker/aws4_request, SignedHeaders=host, Signature=bad")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Fatalf("bad signature status = %d", resp.StatusCode)
	}
	_ = mustBody(t, resp)
}
