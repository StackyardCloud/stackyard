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

func newElasticsearchServiceTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// eslCall sends an es-signed request and decodes the JSON reply.
func eslCall(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	return eslCallAs(t, ts, method, path, body, "es")
}

func eslCallAs(t *testing.T, ts *httptest.Server, method, path, body, service string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, service)
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	return resp.StatusCode, out
}

func eslWant(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	status, out := eslCall(t, ts, method, path, body)
	if status != want {
		t.Fatalf("%s %s: status %d (want %d) %v", method, path, status, want, out)
	}
	return out
}

func TestElasticsearchServiceStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.ElasticsearchServiceModel()
	if len(elasticsearchserviceOperations) != 52 || len(elasticsearchserviceOperationByName) != 52 {
		t.Fatalf("operations: %d / %d", len(elasticsearchserviceOperations), len(elasticsearchserviceOperationByName))
	}
	for _, op := range elasticsearchserviceOperations {
		if _, ok := m.Operations[op.Name]; !ok {
			t.Fatalf("%s is not in the model", op.Name)
		}
		if !strings.HasPrefix(op.URI, eslPathPrefix) {
			t.Fatalf("%s URI %s is outside %s", op.Name, op.URI, eslPathPrefix)
		}
	}
}

func TestElasticsearchServiceStage0Routing(t *testing.T) {
	ts := newElasticsearchServiceTestServer(t)
	// Unknown /2015-01-01/ route signed for es.
	status, out := eslCall(t, ts, http.MethodGet, "/2015-01-01/es/nope", "")
	if status != http.StatusBadRequest || out["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown route: %d %v", status, out)
	}
	// OpenSearch traffic still reaches OpenSearch.
	if status, out := eslCall(t, ts, http.MethodGet, "/2021-01-01/domain", ""); status != http.StatusOK || out["DomainNames"] == nil {
		t.Fatalf("OpenSearch ListDomainNames: %d %v", status, out)
	}
	// A request signed for another service isn't claimed by es.
	s3 := signedRequestWithService(t, http.MethodGet, ts.URL+"/2015-01-01/domain", nil, nil, "s3")
	s3Body, _ := io.ReadAll(s3.Body)
	s3.Body.Close()
	if strings.Contains(string(s3Body), "DomainNames") {
		t.Fatalf("s3-signed request reached es: %s", s3Body)
	}
	// Bodies of unknown length are refused before signing is checked.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/2015-01-01/es/domain", io.MultiReader(strings.NewReader(`{"DomainName":"chunked"}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/es/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
