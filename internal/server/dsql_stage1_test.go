package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func dsqlRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if body != nil {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "dsql")
}

func TestDSQLStage1ClusterLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBody := []byte(`{"clientToken":"stage1-create","deletionProtectionEnabled":false}`)
	resp := dsqlRequest(t, ts, http.MethodPost, "/cluster", createBody)
	assertStatus(t, resp, http.StatusOK)

	var created struct {
		ARN        string `json:"arn"`
		Identifier string `json:"identifier"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &created); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}
	if created.ARN == "" {
		t.Fatalf("expected cluster arn")
	}
	if created.Identifier == "" {
		t.Fatalf("expected cluster identifier")
	}
	if !regexp.MustCompile(`^[a-z0-9]{26}$`).MatchString(created.Identifier) {
		t.Fatalf("expected 26-char lowercase identifier, got %q", created.Identifier)
	}
	if created.Status == "" {
		t.Fatalf("expected cluster status")
	}

	resp = dsqlRequest(t, ts, http.MethodGet, "/cluster?max-results=1", nil)
	assertStatus(t, resp, http.StatusOK)
	var listOut struct {
		Clusters []struct {
			Identifier string `json:"identifier"`
		} `json:"clusters"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listOut); err != nil {
		t.Fatalf("unmarshal list clusters: %v", err)
	}
	names := make([]string, 0, len(listOut.Clusters))
	for _, item := range listOut.Clusters {
		names = append(names, item.Identifier)
	}
	if !slices.Contains(names, created.Identifier) {
		t.Fatalf("expected cluster list to include %q, got %v", created.Identifier, names)
	}

	resp = dsqlRequest(t, ts, http.MethodGet, "/cluster/"+created.Identifier, nil)
	assertStatus(t, resp, http.StatusOK)
	var described struct {
		Identifier string `json:"identifier"`
		ARN        string `json:"arn"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &described); err != nil {
		t.Fatalf("unmarshal get cluster: %v", err)
	}
	if described.Identifier != created.Identifier {
		t.Fatalf("expected identifier %q, got %q", created.Identifier, described.Identifier)
	}
	if described.ARN != created.ARN {
		t.Fatalf("expected arn %q, got %q", created.ARN, described.ARN)
	}

	resp = dsqlRequest(t, ts, http.MethodDelete, "/cluster/"+created.Identifier+"?client-token=stage1-delete", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = dsqlRequest(t, ts, http.MethodGet, "/cluster/"+created.Identifier, nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException, got %s", body)
	}
}

func TestDSQLStage1ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dsqlRequest(t, ts, http.MethodPost, "/cluster", []byte(`{"deletionProtectionEnabled":false}`))
	assertStatus(t, resp, http.StatusOK)
	var created struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &created); err != nil {
		t.Fatalf("unmarshal create cluster: %v", err)
	}
	if created.Identifier == "" {
		t.Fatalf("expected identifier from create")
	}

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/cluster"},
		{method: http.MethodGet, path: "/cluster/" + created.Identifier},
		{method: http.MethodDelete, path: "/cluster/" + created.Identifier},
	}
	for _, tc := range cases {
		resp := dsqlRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
