package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cognitoUserPoolsRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSCognitoIdentityProviderService." + action,
		},
		"cognito-idp",
	)
}

func TestCognitoUserPoolsStage0CatalogCoverage(t *testing.T) {
	if len(cognitoUserPoolsOperations) != 119 {
		t.Fatalf("expected 119 Cognito User Pools operations from docs, got %d", len(cognitoUserPoolsOperations))
	}
	if len(cognitoUserPoolsOperationByName) != len(cognitoUserPoolsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateUserPool",
		"DescribeUserPool",
		"ListUserPools",
		"UpdateUserPool",
		"DeleteUserPool",
		"CreateUserPoolDomain",
		"DescribeUserPoolDomain",
		"DeleteUserPoolDomain",
		"CreateUserPoolClient",
		"DescribeUserPoolClient",
		"ListUserPoolClients",
		"UpdateUserPoolClient",
		"DeleteUserPoolClient",
		"CreateResourceServer",
		"DescribeResourceServer",
		"ListResourceServers",
		"UpdateResourceServer",
		"DeleteResourceServer",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, name := range requiredActions {
		if _, ok := cognitoUserPoolsOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(cognitoUserPoolsDataTypes) != 84 {
		t.Fatalf("expected 84 Cognito User Pools data types from docs, got %d", len(cognitoUserPoolsDataTypes))
	}
	if len(cognitoUserPoolsDataTypeByName) != len(cognitoUserPoolsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"UserPoolType",
		"UserPoolClientType",
		"ResourceServerType",
		"IdentityProviderType",
		"GroupType",
		"DomainDescriptionType",
		"TokenValidityUnitsType",
		"SchemaAttributeType",
	}
	for _, name := range requiredTypes {
		if _, ok := cognitoUserPoolsDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestCognitoUserPoolsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cognitoUserPoolsRequest(t, ts, "TotallyUnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCognitoUserPoolsStage0RoutingForImplementedAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	implementedResp := cognitoUserPoolsRequest(t, ts, "CreateUserPool", []byte(`{"PoolName":"stackyard-stage0-userpool"}`))
	assertStatus(t, implementedResp, http.StatusOK)
	implementedBody := string(mustBody(t, implementedResp))
	if strings.Contains(implementedBody, "NotImplementedException") {
		t.Fatalf("expected implemented action response, got %q", implementedBody)
	}
}
