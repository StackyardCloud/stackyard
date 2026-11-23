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

func newOSISTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// osisCall sends an osis-signed REST call and decodes the JSON reply.
func osisCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+osisPathPrefix+path, raw, headers, "osis")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, resp.StatusCode, want, data)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	if want >= 400 && resp.Header.Get("X-Amzn-ErrorType") == "" {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType", method, path)
	}
	return out
}

// osisErr returns the error type of a call expected to fail.
func osisErr(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+osisPathPrefix+path, raw, map[string]string{"Content-Type": "application/json"}, "osis")
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: %d, want an error", method, path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestOSISStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.OSISModel()
	if len(osisOperations) != len(model.Operations) || len(osisOperationByName) != len(osisOperations) {
		t.Fatalf("catalog has %d operations (%d unique), model has %d", len(osisOperations), len(osisOperationByName), len(model.Operations))
	}
	for name := range model.Operations {
		if _, ok := osisOperationByName[name]; !ok {
			t.Fatalf("model operation %s missing from the catalog", name)
		}
	}
}

func TestOSISStage0Routing(t *testing.T) {
	_, ts := newOSISTestServer(t)
	osisCall(t, ts, http.StatusOK, http.MethodGet, "listPipelines", "")
	osisCall(t, ts, http.StatusNotFound, http.MethodGet, "nopePipeline", "")
	// OpenSearch Service (signing name es) keeps its own paths.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/2021-01-01/domain", nil, nil, "es")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "OpenSearch Ingestion") {
		t.Fatalf("an es request reached osis: %s", body)
	}
	// Signed for another service, the osis path isn't claimed.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+osisPathPrefix+"listPipelines", nil, nil, "s3")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "Pipelines") {
		t.Fatalf("an s3-signed request reached osis: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+osisPathPrefix+"validatePipeline", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/osis/aws4_request, SignedHeaders=host, Signature=00")
	r2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r2.StatusCode)
	}
}

func TestOSISStage0ConfigCheck(t *testing.T) {
	good := "version: \"2\"\nlog-pipeline:\n  source:\n    http:\n      path: /logs\n  sink:\n    - opensearch:\n        hosts: [\"https://search-x.us-east-1.es.amazonaws.com\"]\n"
	if p := osisCheckConfig(good); len(p) != 0 {
		t.Fatalf("valid config rejected: %v", p)
	}
	if d := osisDestinations(good); len(d) != 1 || d[0].endpoint != "https://search-x.us-east-1.es.amazonaws.com" {
		t.Fatalf("destinations: %v", d)
	}
	for name, body := range map[string]string{
		"no version": "p:\n  source:\n    http:\n  sink:\n    - stdout:\n",
		"no sink":    "version: \"2\"\np:\n  source:\n    http:\n",
		"no source":  "version: \"2\"\np:\n  sink:\n    - stdout:\n",
		"no pipe":    "version: \"2\"\n",
		"tabs":       "version: \"2\"\np:\n\tsource:\n",
		"json":       `{"version":"2","p":{"source":{"http":{}}}}`,
		"bad json":   `{"version":`,
	} {
		if len(osisCheckConfig(body)) == 0 {
			t.Fatalf("%s: config accepted", name)
		}
	}
	if p := osisCheckConfig(`{"version":"2","p":{"source":{"http":{}},"sink":[{"stdout":{}}]}}`); len(p) != 0 {
		t.Fatalf("valid JSON config rejected: %v", p)
	}
	// Every catalog blueprint is itself a valid configuration, and its JSON
	// form parses.
	for _, b := range osisBlueprints {
		if p := osisCheckConfig(b.yaml); len(p) != 0 {
			t.Fatalf("blueprint %s invalid: %v", b.name, p)
		}
		var doc map[string]any
		if err := json.Unmarshal([]byte(osisYAMLToJSON(b.yaml)), &doc); err != nil || doc["version"] != "2" {
			t.Fatalf("blueprint %s JSON: %v %v", b.name, err, doc)
		}
		if p := osisCheckConfig(osisYAMLToJSON(b.yaml)); len(p) != 0 {
			t.Fatalf("blueprint %s JSON form invalid: %v", b.name, p)
		}
	}
}
