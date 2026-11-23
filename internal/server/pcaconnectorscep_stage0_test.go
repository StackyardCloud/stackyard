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

func newPCAConnectorSCEPTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// pscCall sends a pca-connector-scep-signed REST call and decodes the reply.
func pscCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "pca-connector-scep")
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
	return out
}

// pscErrCode returns the X-Amzn-ErrorType of a call expected to fail.
func pscErrCode(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "pca-connector-scep")
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: status %d, want an error", method, path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestPCAConnectorSCEPStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.PCAConnectorSCEPModel()
	if len(pcaconnectorscepOperations) != len(m.Operations) || len(pcaconnectorscepOperationByName) != len(m.Operations) {
		t.Fatalf("catalog has %d operations, model has %d", len(pcaconnectorscepOperations), len(m.Operations))
	}
	for name := range m.Operations {
		if _, ok := pcaconnectorscepOperationByName[name]; !ok {
			t.Fatalf("model operation %s missing from the catalog", name)
		}
	}
}

func TestPCAConnectorSCEPStage0Routing(t *testing.T) {
	_, ts := newPCAConnectorSCEPTestServer(t)
	if code := pscErrCode(t, ts, http.MethodPut, "/connectors", "{}"); code != "ValidationException" {
		t.Fatalf("unknown operation: %s", code)
	}
	// Private CA (JSON 1.1, signing name acm-pca) keeps its traffic.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`),
		map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "ACMPrivateCA.ListCertificateAuthorities"}, "acm-pca")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "CertificateAuthorities") {
		t.Fatalf("acm-pca ListCertificateAuthorities: %d %s", resp.StatusCode, body)
	}
	// A connector path signed for another service isn't claimed.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/connectors", nil, nil, "s3")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), `"Connectors"`) {
		t.Fatalf("an s3-signed request reached the SCEP connector API: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/connectors", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/pca-connector-scep/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
