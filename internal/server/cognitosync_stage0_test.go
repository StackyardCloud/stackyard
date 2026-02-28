package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCognitoSyncStage0CatalogCoverage(t *testing.T) {
	if len(cognitoSyncOperations) != 17 {
		t.Fatalf("expected 17 Cognito Sync operations from docs, got %d", len(cognitoSyncOperations))
	}
	if len(cognitoSyncOperationByName) != len(cognitoSyncOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredOps := []string{
		"DescribeIdentityPoolUsage",
		"DescribeIdentityUsage",
		"ListIdentityPoolUsage",
		"ListDatasets",
		"ListRecords",
		"SetCognitoEvents",
		"SetIdentityPoolConfiguration",
		"RegisterDevice",
		"UpdateRecords",
	}
	for _, name := range requiredOps {
		if _, ok := cognitoSyncOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(cognitoSyncDataTypes) != 7 {
		t.Fatalf("expected 7 Cognito Sync data types from docs, got %d", len(cognitoSyncDataTypes))
	}
	if len(cognitoSyncDataTypeByName) != len(cognitoSyncDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CognitoStreams",
		"Dataset",
		"IdentityPoolUsage",
		"IdentityUsage",
		"PushSync",
		"Record",
		"RecordPatch",
	}
	for _, name := range requiredTypes {
		if _, ok := cognitoSyncDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestCognitoSyncStage0KnownUnimplementedRouteReturnsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/identitypools/us-east-1%3Astage0/__stackyard_unimplemented_cognitosync_route__",
		[]byte(`{}`),
		map[string]string{"Content-Type": "application/json"},
		"cognito-sync",
	)
	assertStatus(t, resp, http.StatusNotImplemented)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected NotImplementedException response body, got %q", body)
	}
}
