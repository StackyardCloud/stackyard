package server

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newTextractTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func txCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": txTargetPrefix + op,
	}, "textract")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func txOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := txCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func txErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := txCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func txBytesDoc(text string) string {
	b, _ := json.Marshal(map[string]any{"Bytes": base64.StdEncoding.EncodeToString([]byte(text))})
	return string(b)
}

func TestTextractStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.TextractModel()
	if len(textractOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(textractOperations), len(model.Operations))
	}
	for _, op := range textractOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if txHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestTextractStage0Routing(t *testing.T) {
	_, ts := newTextractTestServer(t)
	if out := txOK(t, ts, "ListAdapters", `{}`); out["Adapters"] == nil {
		t.Fatalf("ListAdapters: %v", out)
	}
	// Signed for another service: refused, not served.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": txTargetPrefix + "ListAdapters",
	}, "rekognition")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d %s", resp.StatusCode, body)
	}
	if code := txErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	if code := txErrCode(t, ts, "DetectDocumentText", `{}`); code != "InvalidParameterException" {
		t.Fatalf("missing Document: %s", code)
	}
	// Chunked bodies of unknown length are refused before the body is read.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", txTargetPrefix+"ListAdapters")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	// A request with no Textract target falls through to other routers.
	if code := txErrCode(t, ts, "DetectDocumentText", `{"Document":{"Bytes":"!!!"}}`); code != "InvalidParameterException" {
		t.Fatalf("bad base64: %s", code)
	}
}
