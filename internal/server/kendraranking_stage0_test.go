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

func kdrCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": kdrContentType, "X-Amz-Target": kdrTargetPrefix + op,
	}, "kendra-ranking")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func kdrOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := kdrCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func kdrErrCode(t *testing.T, ts *httptest.Server, op, body string, want int) string {
	t.Helper()
	status, out := kdrCall(t, ts, op, body)
	if status != want {
		t.Fatalf("%s: want %d, got %d %v", op, want, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestKendraRankingStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.KendraRankingModel()
	if len(kendraRankingOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(kendraRankingOperations), len(model.Operations))
	}
	for _, op := range kendraRankingOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if kdrHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestKendraRankingStage0Routing(t *testing.T) {
	ts := newCSTestServer(t)
	kdrOK(t, ts, "ListRescoreExecutionPlans", `{}`)

	// Signed for kendra: refused, never served as kendra-ranking.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": kdrContentType, "X-Amz-Target": kdrTargetPrefix + "ListRescoreExecutionPlans",
	}, "kendra")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("kendra-signed request: %d %s", resp.StatusCode, body)
	}
	if code := kdrErrCode(t, ts, "Frobnicate", `{}`, http.StatusBadRequest); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	if code := kdrErrCode(t, ts, "CreateRescoreExecutionPlan", `{"Name":`, http.StatusBadRequest); code != "SerializationException" {
		t.Fatalf("bad json: %s", code)
	}

	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", kdrTargetPrefix+"ListRescoreExecutionPlans")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}
}
