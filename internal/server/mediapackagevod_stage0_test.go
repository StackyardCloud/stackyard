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

func newMediaPackageVODTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// mpvodCall sends a mediapackage-vod REST call; it fails the test unless the
// status is want, and decodes the reply.
func mpvodCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "mediapackage-vod")
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

func TestMediaPackageVODStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.MediaPackageVODModel()
	if len(mpvodOperations) != len(m.Operations) || len(mpvodOperationByName) != len(m.Operations) {
		t.Fatalf("catalog has %d operations, model has %d", len(mpvodOperations), len(m.Operations))
	}
	for name := range m.Operations {
		if _, ok := mpvodOperationByName[name]; !ok {
			t.Fatalf("model operation %s missing from the catalog", name)
		}
	}
	// The slim model is keyed by wire names.
	if _, ok := m.Shapes["CreateAssetRequest"].Members["sourceArn"]; !ok {
		t.Fatal("model members are not wire names")
	}
}

func TestMediaPackageVODStage0Routing(t *testing.T) {
	ts := newMediaPackageVODTestServer(t)
	mpvodCall(t, ts, http.StatusNotFound, http.MethodPatch, "/assets", "")
	// MediaPackage v2 keeps its traffic.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/channelGroup", nil, nil, "mediapackagev2")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || strings.Contains(string(body), "packagingGroups") {
		t.Fatalf("mediapackagev2 ListChannelGroups: %d %s", resp.StatusCode, body)
	}
	// A VOD path signed for another service isn't claimed.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/packaging_groups", nil, nil, "s3")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "packagingGroups") {
		t.Fatalf("an s3-signed request reached MediaPackage VOD: %s", body)
	}
}
