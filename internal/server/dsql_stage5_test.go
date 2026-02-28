package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDSQLStage5CompatibilityHardening(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dsqlRequest(t, ts, http.MethodGet, "/cluster?max-results=999", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid max-results, got %d", resp.StatusCode)
	}

	resp = dsqlRequest(t, ts, http.MethodGet, "/cluster?next-token=bad", nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid next-token, got %d", resp.StatusCode)
	}

	// CreateCluster idempotency: same clientToken should return the same cluster.
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"clientToken":"stage5-idem","deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	var first struct {
		Identifier string `json:"identifier"`
		ARN        string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &first); err != nil {
		t.Fatalf("unmarshal first create: %v", err)
	}
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"clientToken":"stage5-idem","deletionProtectionEnabled":true}`))
	assertStatus(t, resp, http.StatusOK)
	var second struct {
		Identifier string `json:"identifier"`
		ARN        string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &second); err != nil {
		t.Fatalf("unmarshal second create: %v", err)
	}
	if first.Identifier != second.Identifier || first.ARN != second.ARN {
		t.Fatalf("expected idempotent create, got first=%+v second=%+v", first, second)
	}

	resp = dsqlRequest(t, ts, http.MethodDelete, "/cluster/"+first.Identifier, nil)
	assertStatus(t, resp, http.StatusOK)

	// Create protected cluster and verify conflict until update disables protection.
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"identifier":"dsql0000000000000000000010","deletionProtectionEnabled":true}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dsqlRequest(t, ts, http.MethodDelete, "/cluster/dsql0000000000000000000010", nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 when deletion protection is enabled, got %d", resp.StatusCode)
	}

	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster/dsql0000000000000000000010", []byte(`{"deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	resp = dsqlRequest(t, ts, http.MethodDelete, "/cluster/dsql0000000000000000000010", nil)
	assertStatus(t, resp, http.StatusOK)

	// PutClusterPolicy rejects invalid policy JSON.
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"identifier":"dsql0000000000000000000011","deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster/dsql0000000000000000000011/policy", []byte(`{"policy":"{not-json}"}`))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid policy JSON, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(mustBody(t, resp)), "ValidationException") {
		t.Fatalf("expected ValidationException for invalid policy")
	}
}

func TestDSQLStage5AllCatalogActionsRoutable(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"identifier":"dsql0000000000000000000099","deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	var created struct {
		Identifier string `json:"identifier"`
		ARN        string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &created); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}

	requests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/cluster"},
		{method: http.MethodGet, path: "/cluster/" + created.Identifier},
		{method: http.MethodPost, path: "/cluster/" + created.Identifier, body: []byte(`{"deletionProtectionEnabled":false}`)},
		{method: http.MethodGet, path: "/clusters/" + created.Identifier + "/vpc-endpoint-service-name"},
		{method: http.MethodPost, path: "/cluster/" + created.Identifier + "/policy", body: []byte(`{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)},
		{method: http.MethodGet, path: "/cluster/" + created.Identifier + "/policy"},
		{method: http.MethodDelete, path: "/cluster/" + created.Identifier + "/policy"},
		{method: http.MethodPost, path: "/tags/" + url.PathEscape(created.ARN), body: []byte(`{"tags":{"env":"dev"}}`)},
		{method: http.MethodGet, path: "/tags/" + url.PathEscape(created.ARN)},
		{method: http.MethodDelete, path: "/tags/" + url.PathEscape(created.ARN) + "?tagKeys=env"},
		{method: http.MethodDelete, path: "/cluster/" + created.Identifier},
	}

	for _, req := range requests {
		resp := dsqlRequest(t, ts, req.method, req.path, req.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", req.method, req.path)
		}
	}
}
