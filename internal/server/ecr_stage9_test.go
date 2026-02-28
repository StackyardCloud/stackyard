package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestECRStage9ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage9-raw"
	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	putImagePayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  `{"schemaVersion":2}`,
		"imageTag":       "v1",
	})
	if err != nil {
		t.Fatalf("marshal put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", putImagePayload)
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "PutSigningConfiguration", body: []byte(`{"signingConfiguration":{"rules":[{"signingProfileArn":"arn:aws:signer:us-east-1:123456789012:/signing-profiles/demo","repositoryFilters":[{"filter":"stage9-*","filterType":"WILDCARD_MATCH"}]}]}}`)},
		{name: "GetSigningConfiguration", body: []byte(`{}`)},
		{name: "DescribeImageSigningStatus", body: []byte(`{"repositoryName":"` + repositoryName + `","imageId":{"imageTag":"v1"}}`)},
		{name: "DeleteSigningConfiguration", body: []byte(`{}`)},
	}

	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage9SigningConfigurationNotFound(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := ecrRequest(t, ts, "GetSigningConfiguration", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if got := resp.Header.Get("X-Amzn-ErrorType"); got != "SigningConfigurationNotFoundException" {
		t.Fatalf("expected SigningConfigurationNotFoundException, got %q", got)
	}
}
