package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newPollyTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// pollyRaw sends a polly-signed request and returns status, headers and body.
func pollyRaw(t *testing.T, ts *httptest.Server, method, path, body string, headers map[string]string) (int, http.Header, []byte) {
	t.Helper()
	if headers == nil {
		headers = map[string]string{}
	}
	if body != "" && headers["Content-Type"] == "" {
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), headers, "polly")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, data
}

// pollyCall expects want and decodes a JSON reply.
func pollyCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	status, hdr, data := pollyRaw(t, ts, method, path, body, nil)
	if status != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, status, want, data)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	if want >= 400 && hdr.Get("X-Amzn-ErrorType") == "" {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType", method, path)
	}
	return out
}

// pollyErrCode returns the error type of a call expected to fail.
func pollyErrCode(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	status, hdr, data := pollyRaw(t, ts, method, path, body, nil)
	if status < 400 {
		t.Fatalf("%s %s: status %d, want an error (%s)", method, path, status, data)
	}
	return hdr.Get("X-Amzn-ErrorType")
}

func TestPollyStage0CatalogMatchesModel(t *testing.T) {
	var model, catalog []string
	for name := range awsmodels.PollyModel().Operations {
		model = append(model, name)
	}
	for _, op := range pollyOperations {
		catalog = append(catalog, op.Name)
	}
	sort.Strings(model)
	sort.Strings(catalog)
	if strings.Join(model, ",") != strings.Join(catalog, ",") {
		t.Fatalf("catalog %v != model %v", catalog, model)
	}
	if len(pollyVoices) != 116 {
		t.Fatalf("voice catalog has %d voices, want the model's 116", len(pollyVoices))
	}
	for _, v := range pollyVoices {
		if problems := awsmodels.PollyModel().Validate("VoiceId", v.id); len(problems) > 0 {
			t.Fatalf("voice %s isn't in the model: %v", v.id, problems)
		}
		if pollyLanguageNames[v.lang] == "" {
			t.Fatalf("voice %s has unknown language %s", v.id, v.lang)
		}
	}
}

func TestPollyStage0RoutingAndBodies(t *testing.T) {
	_, ts := newPollyTestServer(t)
	// Route 53 Profiles and S3 traffic isn't claimed.
	if status, _, data := func() (int, http.Header, []byte) {
		resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/profiles", nil, nil, "route53profiles")
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header, b
	}(); status != http.StatusOK {
		t.Fatalf("route53profiles ListProfiles: %d %s", status, data)
	}
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/v1/voices", nil, nil, "s3")
	s3body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(s3body), `"Voices"`) {
		t.Fatalf("an s3-signed request reached Polly: %s", s3body)
	}
	// Unknown Polly route.
	if code := pollyErrCode(t, ts, http.MethodGet, "/v1/speech", ""); code != "UnknownOperationException" {
		t.Fatalf("unknown route: %s", code)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/speech", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/polly/aws4_request, SignedHeaders=host, Signature=00")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
}
