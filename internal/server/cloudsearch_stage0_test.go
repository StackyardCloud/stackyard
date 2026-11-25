package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newCSTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// csCall posts a Query-protocol action; params are flattened Query keys.
func csCall(t *testing.T, ts *httptest.Server, action string, params map[string]string) (int, string) {
	t.Helper()
	form := url.Values{"Action": {action}, "Version": {csVersion}}
	for k, v := range params {
		form.Set(k, v)
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "cloudsearch")
	return resp.StatusCode, string(mustBody(t, resp))
}

func csOK(t *testing.T, ts *httptest.Server, action string, params map[string]string) string {
	t.Helper()
	status, body := csCall(t, ts, action, params)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %s", action, status, body)
	}
	return body
}

func csErrCode(t *testing.T, ts *httptest.Server, action string, params map[string]string, wantStatus int) string {
	t.Helper()
	status, body := csCall(t, ts, action, params)
	if status != wantStatus {
		t.Fatalf("%s: want %d, got %d %s", action, wantStatus, status, body)
	}
	start, end := strings.Index(body, "<Code>"), strings.Index(body, "</Code>")
	if start < 0 || end < start {
		t.Fatalf("%s: no error code in %s", action, body)
	}
	return body[start+len("<Code>") : end]
}

func TestCloudSearchStage0CatalogMatchesModel(t *testing.T) {
	for _, tc := range []struct {
		model *awsmodels.Model
		names []string
		has   func(string) bool
	}{
		{awsmodels.CloudSearchModel(), csOpNames(), func(op string) bool { return csHandlers[op] != nil }},
		{awsmodels.CloudSearchDomainModel(), csdOpNames(), func(op string) bool { return cloudSearchDomainOperationByName[op] }},
	} {
		if len(tc.names) != len(tc.model.Operations) {
			t.Fatalf("catalog has %d operations, model %d", len(tc.names), len(tc.model.Operations))
		}
		for _, op := range tc.names {
			if _, ok := tc.model.Operations[op]; !ok {
				t.Errorf("%s is not in the model", op)
			}
			if !tc.has(op) {
				t.Errorf("%s has no handler", op)
			}
		}
	}
}

func csOpNames() []string {
	var out []string
	for _, op := range cloudSearchOperations {
		out = append(out, op.Name)
	}
	return out
}

func csdOpNames() []string {
	var out []string
	for _, op := range cloudSearchDomainOperations {
		out = append(out, op.Name)
	}
	return out
}

func TestCloudSearchStage0Routing(t *testing.T) {
	ts := newCSTestServer(t)
	body := csOK(t, ts, "CreateDomain", map[string]string{"DomainName": "route-one"})
	if !strings.Contains(body, "<CreateDomainResponse") || !strings.Contains(body, "<DomainName>route-one</DomainName>") {
		t.Fatalf("CreateDomain body: %s", body)
	}

	// Unknown actions and versions are refused.
	if code := csErrCode(t, ts, "Frobnicate", nil, http.StatusBadRequest); code != "InvalidAction" {
		t.Fatalf("unknown action: %s", code)
	}

	// Signed for another service: never served as CloudSearch.
	form := url.Values{"Action": {"ListDomainNames"}, "Version": {csVersion}}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "es")
	if b := mustBody(t, resp); strings.Contains(string(b), "ListDomainNamesResponse") {
		t.Fatalf("es-signed request was served by CloudSearch: %s", b)
	}

	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(form.Encode())))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=x/20260101/us-east-1/cloudsearch/aws4_request, SignedHeaders=host, Signature=00")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}

	// Kendra keeps its traffic; Kendra Intelligent Ranking gets its own.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSKendraFrontendService.ListIndices",
	}, "kendra")
	raw := mustBody(t, resp)
	if resp.StatusCode != http.StatusOK || strings.Contains(string(raw), "UnknownOperation") {
		t.Fatalf("kendra ListIndices: %d %s", resp.StatusCode, raw)
	}
	status, out := kdrCall(t, ts, "ListRescoreExecutionPlans", `{}`)
	if status != http.StatusOK {
		t.Fatalf("kendra-ranking list: %d %v", status, out)
	}

	// OpenSearch's REST API is untouched.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/2021-01-01/domain", nil, nil, "es")
	raw = mustBody(t, resp)
	var names map[string]any
	if resp.StatusCode != http.StatusOK || json.Unmarshal(raw, &names) != nil {
		t.Fatalf("opensearch ListDomainNames: %d %s", resp.StatusCode, raw)
	}
}
