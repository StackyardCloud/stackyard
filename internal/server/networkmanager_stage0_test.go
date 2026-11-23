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

func newNetworkManagerTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// nmCall sends a networkmanager-signed REST call and decodes the reply.
func nmCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "networkmanager")
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

// nmErrCode returns the X-Amzn-ErrorType of a call expected to fail.
func nmErrCode(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "networkmanager")
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: status %d, want an error", method, path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestNetworkManagerStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.NetworkManagerModel()
	if len(networkmanagerOperations) != len(m.Operations) || len(networkmanagerOperationByName) != len(m.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(networkmanagerOperations), len(m.Operations))
	}
	for _, op := range networkmanagerOperations {
		if _, ok := m.Operations[op.Name]; !ok {
			t.Errorf("%s isn't in the model", op.Name)
		}
		if nmHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestNetworkManagerStage0Routing(t *testing.T) {
	_, ts := newNetworkManagerTestServer(t)
	if code := nmErrCode(t, ts, http.MethodPut, "/global-networks", ""); code != "UnknownOperationException" {
		t.Fatalf("unknown route: %s", code)
	}
	// An ARN label routes whether its slashes are percent-encoded or raw.
	nmCall(t, ts, http.StatusNotFound, http.MethodGet, "/tags/"+url.PathEscape("arn:aws:networkmanager::123456789012:global-network/global-network-0123456789abcdef0"), "")
	nmCall(t, ts, http.StatusNotFound, http.MethodGet, "/tags/arn:aws:networkmanager::123456789012:global-network/global-network-0123456789abcdef0", "")

	// A Network Manager path signed for another service isn't claimed.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/global-networks", nil, nil, "s3")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "GlobalNetworks") {
		t.Fatalf("an s3-signed request reached Network Manager: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/global-networks", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/networkmanager/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}

func TestNetworkManagerStage0InputValidation(t *testing.T) {
	_, ts := newNetworkManagerTestServer(t)
	for name, c := range map[string]struct{ method, path, body string }{
		"unknown member":    {http.MethodPost, "/global-networks", `{"Bogus":1}`},
		"bad core id label": {http.MethodGet, "/core-networks/not-a-core-network", ""},
		"maxResults range":  {http.MethodGet, "/global-networks?maxResults=0", ""},
		"maxResults type":   {http.MethodGet, "/global-networks?maxResults=abc", ""},
		"missing required":  {http.MethodPost, "/core-networks", `{}`},
		"bad next token":    {http.MethodGet, "/global-networks?nextToken=zz", ""},
	} {
		if code := nmErrCode(t, ts, c.method, c.path, c.body); code != "ValidationException" {
			t.Errorf("%s: %s", name, code)
		}
	}
}
