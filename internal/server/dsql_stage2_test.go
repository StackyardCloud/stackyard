package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDSQLStage2UpdateAndVpcEndpointServiceName(t *testing.T) {
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

	resp = dsqlRequest(t, ts, http.MethodPost, "/cluster/"+created.Identifier, []byte(`{"deletionProtectionEnabled":true,"clientToken":"stage2-update"}`))
	assertStatus(t, resp, http.StatusOK)
	var updated struct {
		Identifier                string `json:"identifier"`
		DeletionProtectionEnabled bool   `json:"deletionProtectionEnabled"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updated); err != nil {
		t.Fatalf("unmarshal update cluster: %v", err)
	}
	if updated.Identifier != created.Identifier {
		t.Fatalf("expected identifier %q, got %q", created.Identifier, updated.Identifier)
	}
	if !updated.DeletionProtectionEnabled {
		t.Fatalf("expected deletionProtectionEnabled=true after update")
	}

	resp = dsqlRequest(t, ts, http.MethodGet, "/clusters/"+created.Identifier+"/vpc-endpoint-service-name", nil)
	assertStatus(t, resp, http.StatusOK)
	var vpcOut struct {
		VpcEndpointServiceName string `json:"vpcEndpointServiceName"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &vpcOut); err != nil {
		t.Fatalf("unmarshal vpc endpoint service name: %v", err)
	}
	if vpcOut.VpcEndpointServiceName == "" {
		t.Fatalf("expected vpc endpoint service name")
	}
}

func TestDSQLStage2ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dsqlRequest(t, ts, http.MethodPost, "/cluster", nil)
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
		{method: http.MethodPost, path: "/cluster/" + created.Identifier, body: []byte(`{"deletionProtectionEnabled":false}`)},
		{method: http.MethodGet, path: "/clusters/" + created.Identifier + "/vpc-endpoint-service-name"},
	}
	for _, tc := range cases {
		resp := dsqlRequest(t, ts, tc.method, tc.path, tc.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("%s %s returned NotImplemented", tc.method, tc.path)
		}
	}
}
