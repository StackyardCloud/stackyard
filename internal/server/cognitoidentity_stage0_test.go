package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cognitoIdentityRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "com.amazonaws.cognito.identity.model.AWSCognitoIdentityService." + action,
		},
		"cognito-identity",
	)
}

func TestCognitoIdentityStage0CatalogCoverage(t *testing.T) {
	if len(cognitoIdentityOperations) != 23 {
		t.Fatalf("expected 23 Cognito Federated Identities operations from docs, got %d", len(cognitoIdentityOperations))
	}
	if len(cognitoIdentityOperationByName) != len(cognitoIdentityOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"GetId",
		"GetCredentialsForIdentity",
		"CreateIdentityPool",
		"DeleteIdentityPool",
		"ListIdentityPools",
		"TagResource",
		"UntagResource",
	}
	for _, name := range requiredActions {
		if _, ok := cognitoIdentityOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(cognitoIdentityDataTypes) != 8 {
		t.Fatalf("expected 8 Cognito Federated Identities data types from docs, got %d", len(cognitoIdentityDataTypes))
	}
	if len(cognitoIdentityDataTypeByName) != len(cognitoIdentityDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CognitoIdentityProvider",
		"Credentials",
		"IdentityDescription",
		"IdentityPoolShortDescription",
		"RoleMapping",
		"RulesConfigurationType",
	}
	for _, name := range requiredTypes {
		if _, ok := cognitoIdentityDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestCognitoIdentityStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cognitoIdentityRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCognitoIdentityStage0KnownActionIsRouted(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cognitoIdentityRequest(t, ts, "CreateIdentityPool", []byte(`{"IdentityPoolName":"stackyard-stage0-known","AllowUnauthenticatedIdentities":true}`))
	assertStatus(t, createResp, http.StatusOK)
	createBody := string(mustBody(t, createResp))
	if !strings.Contains(createBody, "IdentityPoolId") {
		t.Fatalf("expected IdentityPoolId in create response body, got %q", createBody)
	}

	resp := cognitoIdentityRequest(t, ts, "GetIdentityPoolRoles", []byte(`{"IdentityPoolId":"us-east-1:11111111-2222-3333-4444-555555555555"}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected implemented route response body, got %q", body)
	}
}
