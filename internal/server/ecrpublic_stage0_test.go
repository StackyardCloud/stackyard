package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newECRPublicTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func ecrpHeaders(op string) map[string]string {
	return map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": ecrPublicTargetPrefix + op}
}

// ecrpRequest sends a signed ECR Public call and returns status and body.
func ecrpRequest(t *testing.T, ts *httptest.Server, op, body string) (int, []byte) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), ecrpHeaders(op), ecrPublicSigningName)
	return resp.StatusCode, mustBody(t, resp)
}

func ecrpOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, raw := ecrpRequest(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %s", op, body, status, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: decode %s: %v", op, raw, err)
	}
	return out
}

// ecrpFail expects a 4xx and returns the error code.
func ecrpFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, raw := ecrpRequest(t, ts, op, body)
	if status < 400 || status >= 500 {
		t.Fatalf("%s %s: status %d (want 4xx) %s", op, body, status, raw)
	}
	var e struct {
		Type string `json:"__type"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("decode error %s: %v", raw, err)
	}
	return e.Type
}

func TestECRPublicStage0CatalogMatchesModel(t *testing.T) {
	if len(ecrPublicOperations) != 23 || len(ecrPublicOperationByName) != 23 {
		t.Fatalf("operations: %d unique %d, want 23", len(ecrPublicOperations), len(ecrPublicOperationByName))
	}
	for _, op := range []string{"CreateRepository", "PutImage", "UploadLayerPart", "DescribeRegistries", "PutRegistryCatalogData"} {
		if _, ok := ecrPublicOperationByName[op]; !ok {
			t.Fatalf("missing %s", op)
		}
	}
}

func TestECRPublicStage0Routing(t *testing.T) {
	ts := newECRPublicTestServer(t)

	if code := ecrpFail(t, ts, "NoSuchOperation", `{}`); code != "InvalidParameterException" {
		t.Fatalf("unknown action: %s", code)
	}
	if code := ecrpFail(t, ts, "DescribeRegistries", `{"bogus":1}`); code != "InvalidParameterException" {
		t.Fatalf("unknown member: %s", code)
	}
	if code := ecrpFail(t, ts, "DescribeRegistries", `{`); code != "InvalidParameterException" {
		t.Fatalf("bad JSON: %s", code)
	}

	// Signed for another service: rejected, not handed to private ECR.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), ecrpHeaders("DescribeRegistries"), "ecr")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// ECR Public only exists in us-east-1.
	resp = signRequestWithHeaders(t, http.MethodPost, ts.URL+"/", []byte(`{}`), ecrpHeaders("DescribeRegistries"), ecrPublicSigningName, "eu-west-1", "")
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "UnsupportedCommandException") {
		t.Fatalf("other region: %d %s", resp.StatusCode, body)
	}

	// Private ECR still routes next to the public router.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AmazonEC2ContainerRegistry_V20150921.DescribeRepositories",
	}, "ecr")
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	// A body of unknown length is refused before anything reads it.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range ecrpHeaders("DescribeRegistries") {
		req.Header.Set(k, v)
	}
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	if ct := chunked.Header.Get("Content-Type"); ct != "application/x-amz-json-1.1" {
		t.Fatalf("content type: %q", ct)
	}
}
