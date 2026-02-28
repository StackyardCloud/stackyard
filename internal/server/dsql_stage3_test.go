package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDSQLStage3ClusterPolicyLifecycle(t *testing.T) {
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

	policy1 := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":"dsql:GetCluster","Resource":"*"}]}`
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster/"+created.Identifier+"/policy", []byte(`{"policy":`+quoteJSONString(policy1)+`}`))
	assertStatus(t, resp, http.StatusOK)
	var put1 struct {
		Policy        string `json:"policy"`
		PolicyVersion string `json:"policyVersion"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &put1); err != nil {
		t.Fatalf("unmarshal put policy: %v", err)
	}
	if put1.PolicyVersion != "v1" {
		t.Fatalf("expected policyVersion v1, got %q", put1.PolicyVersion)
	}

	resp = dsqlRequest(t, ts, http.MethodGet, "/cluster/"+created.Identifier+"/policy", nil)
	assertStatus(t, resp, http.StatusOK)
	var got struct {
		Policy        string `json:"policy"`
		PolicyVersion string `json:"policyVersion"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &got); err != nil {
		t.Fatalf("unmarshal get policy: %v", err)
	}
	if got.PolicyVersion != "v1" {
		t.Fatalf("expected policyVersion v1, got %q", got.PolicyVersion)
	}
	if got.Policy != policy1 {
		t.Fatalf("expected policy %q, got %q", policy1, got.Policy)
	}

	policy2 := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":"dsql:ListTagsForResource","Resource":"*"}]}`
	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster/"+created.Identifier+"/policy", []byte(`{"clientToken":"stage3-put","policy":`+quoteJSONString(policy2)+`}`))
	assertStatus(t, resp, http.StatusOK)
	var put2 struct {
		PolicyVersion string `json:"policyVersion"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &put2); err != nil {
		t.Fatalf("unmarshal put policy v2: %v", err)
	}
	if put2.PolicyVersion != "v2" {
		t.Fatalf("expected policyVersion v2, got %q", put2.PolicyVersion)
	}

	resp = dsqlRequest(t, ts, http.MethodDelete, "/cluster/"+created.Identifier+"/policy?expectedPolicyVersion=v1", nil)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 on version mismatch, got %d", resp.StatusCode)
	}

	resp = dsqlRequest(t, ts, http.MethodDelete, "/cluster/"+created.Identifier+"/policy?expectedPolicyVersion=v2&clientToken=stage3-delete", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = dsqlRequest(t, ts, http.MethodGet, "/cluster/"+created.Identifier+"/policy", nil)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete policy, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(mustBody(t, resp)), "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException")
	}
}

func TestDSQLStage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
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

	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodPost, path: "/cluster/" + created.Identifier + "/policy", body: []byte(`{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)},
		{method: http.MethodGet, path: "/cluster/" + created.Identifier + "/policy"},
		{method: http.MethodDelete, path: "/cluster/" + created.Identifier + "/policy"},
	}
	for _, tc := range cases {
		resp := dsqlRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}

func quoteJSONString(in string) string {
	encoded, _ := json.Marshal(in)
	return string(encoded)
}
