package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newCodeCommitTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func ccRequest(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": codecommitTargetPrefix + op,
	}, "codecommit")
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %s: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

// ccOK calls op and fails the test unless it returns 200.
func ccOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := ccRequest(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", op, body, status, out)
	}
	return out
}

// ccFail calls op, expects a 400 and returns the error code.
func ccFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := ccRequest(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s %s: status %d (want 400) %v", op, body, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestCodeCommitStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.CodeCommitModel()
	if len(codecommitOperations) != len(model.Operations) || len(codecommitOperations) != 80 {
		t.Fatalf("operations: %d listed, %d in the model", len(codecommitOperations), len(model.Operations))
	}
	for _, op := range codecommitOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if _, ok := ccHandlers[op.Name]; !ok {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestCodeCommitStage0RoutingAndValidation(t *testing.T) {
	ts := newCodeCommitTestServer(t)
	if code := ccFail(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown operation: %s", code)
	}
	if code := ccFail(t, ts, "GetRepository", `{}`); code != "RepositoryNameRequiredException" {
		t.Fatalf("missing repositoryName: %s", code)
	}
	if code := ccFail(t, ts, "CreateRepository", `{"repositoryName":"bad name!"}`); code != "InvalidRepositoryNameException" {
		t.Fatalf("bad repository name: %s", code)
	}
	if code := ccFail(t, ts, "GetRepository", `not json`); code != "SerializationException" {
		t.Fatalf("bad json: %s", code)
	}
	// Signed for another service.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": codecommitTargetPrefix + "ListRepositories",
	}, "codebuild")
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	// A body of unknown length is refused before signing is checked.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", codecommitTargetPrefix+"ListRepositories")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/codecommit/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
