package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cognitoIdentityRequestPayload(t *testing.T, ts *httptest.Server, action string, payload map[string]any) *http.Response {
	t.Helper()
	body := []byte(`{}`)
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = encoded
	}
	return cognitoIdentityRequest(t, ts, action, body)
}

func decodeCognitoIdentityBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := mustBody(t, resp)
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	out := map[string]any{}
	if err := decoder.Decode(&out); err != nil {
		t.Fatalf("decode JSON body: %v body=%s", err, string(raw))
	}
	return out
}

func TestCognitoIdentityStage12IdentityPoolLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cognitoIdentityRequestPayload(t, ts, "CreateIdentityPool", map[string]any{
		"IdentityPoolName":               "stackyard-stage12-pool",
		"AllowUnauthenticatedIdentities": true,
		"IdentityPoolTags": map[string]any{
			"env": "test",
		},
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := decodeCognitoIdentityBody(t, createResp)
	poolID, _ := createBody["IdentityPoolId"].(string)
	if strings.TrimSpace(poolID) == "" {
		t.Fatalf("expected IdentityPoolId in create response: %#v", createBody)
	}
	if got, _ := createBody["IdentityPoolName"].(string); got != "stackyard-stage12-pool" {
		t.Fatalf("expected pool name from create response, got %#v", createBody["IdentityPoolName"])
	}

	describeResp := cognitoIdentityRequestPayload(t, ts, "DescribeIdentityPool", map[string]any{
		"IdentityPoolId": poolID,
	})
	assertStatus(t, describeResp, http.StatusOK)
	describeBody := decodeCognitoIdentityBody(t, describeResp)
	if got, _ := describeBody["IdentityPoolId"].(string); got != poolID {
		t.Fatalf("expected described pool id %q, got %#v", poolID, describeBody["IdentityPoolId"])
	}

	listResp := cognitoIdentityRequestPayload(t, ts, "ListIdentityPools", map[string]any{
		"MaxResults": 10,
	})
	assertStatus(t, listResp, http.StatusOK)
	listBody := decodeCognitoIdentityBody(t, listResp)
	poolsRaw, ok := listBody["IdentityPools"].([]any)
	if !ok || len(poolsRaw) == 0 {
		t.Fatalf("expected IdentityPools list, got %#v", listBody["IdentityPools"])
	}

	updateResp := cognitoIdentityRequestPayload(t, ts, "UpdateIdentityPool", map[string]any{
		"IdentityPoolId":                 poolID,
		"IdentityPoolName":               "stackyard-stage12-pool-updated",
		"AllowUnauthenticatedIdentities": false,
		"AllowClassicFlow":               true,
	})
	assertStatus(t, updateResp, http.StatusOK)
	updateBody := decodeCognitoIdentityBody(t, updateResp)
	if got, _ := updateBody["IdentityPoolName"].(string); got != "stackyard-stage12-pool-updated" {
		t.Fatalf("expected updated identity pool name, got %#v", updateBody["IdentityPoolName"])
	}
	if got, _ := updateBody["AllowClassicFlow"].(bool); !got {
		t.Fatalf("expected AllowClassicFlow=true in update response")
	}

	deleteResp := cognitoIdentityRequestPayload(t, ts, "DeleteIdentityPool", map[string]any{
		"IdentityPoolId": poolID,
	})
	assertStatus(t, deleteResp, http.StatusOK)

	describeAfterDeleteResp := cognitoIdentityRequestPayload(t, ts, "DescribeIdentityPool", map[string]any{
		"IdentityPoolId": poolID,
	})
	assertStatus(t, describeAfterDeleteResp, http.StatusBadRequest)
	describeAfterDeleteBody := string(mustBody(t, describeAfterDeleteResp))
	if !strings.Contains(describeAfterDeleteBody, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException after delete, got %q", describeAfterDeleteBody)
	}
}

func TestCognitoIdentityStage12IdentityAndCredentialFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cognitoIdentityRequestPayload(t, ts, "CreateIdentityPool", map[string]any{
		"IdentityPoolName":               "stackyard-stage12-identities",
		"AllowUnauthenticatedIdentities": true,
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := decodeCognitoIdentityBody(t, createResp)
	poolID, _ := createBody["IdentityPoolId"].(string)
	if strings.TrimSpace(poolID) == "" {
		t.Fatalf("expected identity pool id from create response: %#v", createBody)
	}

	getIDResp := cognitoIdentityRequestPayload(t, ts, "GetId", map[string]any{
		"IdentityPoolId": poolID,
		"Logins": map[string]any{
			"cognito-idp.us-east-1.amazonaws.com/us-east-1_stage12": "user-1",
		},
	})
	assertStatus(t, getIDResp, http.StatusOK)
	getIDBody := decodeCognitoIdentityBody(t, getIDResp)
	identityID, _ := getIDBody["IdentityId"].(string)
	if strings.TrimSpace(identityID) == "" {
		t.Fatalf("expected identity id in GetId response: %#v", getIDBody)
	}

	getIDAgainResp := cognitoIdentityRequestPayload(t, ts, "GetId", map[string]any{
		"IdentityPoolId": poolID,
		"Logins": map[string]any{
			"cognito-idp.us-east-1.amazonaws.com/us-east-1_stage12": "user-1",
		},
	})
	assertStatus(t, getIDAgainResp, http.StatusOK)
	getIDAgainBody := decodeCognitoIdentityBody(t, getIDAgainResp)
	if got, _ := getIDAgainBody["IdentityId"].(string); got != identityID {
		t.Fatalf("expected stable identity id for same login map, got %q (wanted %q)", got, identityID)
	}

	describeIdentityResp := cognitoIdentityRequestPayload(t, ts, "DescribeIdentity", map[string]any{
		"IdentityId": identityID,
	})
	assertStatus(t, describeIdentityResp, http.StatusOK)
	describeIdentityBody := decodeCognitoIdentityBody(t, describeIdentityResp)
	if got, _ := describeIdentityBody["IdentityId"].(string); got != identityID {
		t.Fatalf("expected describe identity id %q, got %#v", identityID, describeIdentityBody["IdentityId"])
	}

	listIdentitiesResp := cognitoIdentityRequestPayload(t, ts, "ListIdentities", map[string]any{
		"IdentityPoolId": poolID,
		"MaxResults":     10,
	})
	assertStatus(t, listIdentitiesResp, http.StatusOK)
	listIdentitiesBody := decodeCognitoIdentityBody(t, listIdentitiesResp)
	identities, ok := listIdentitiesBody["Identities"].([]any)
	if !ok || len(identities) == 0 {
		t.Fatalf("expected identities in list response, got %#v", listIdentitiesBody["Identities"])
	}

	getCredentialsResp := cognitoIdentityRequestPayload(t, ts, "GetCredentialsForIdentity", map[string]any{
		"IdentityId": identityID,
	})
	assertStatus(t, getCredentialsResp, http.StatusOK)
	getCredentialsBody := decodeCognitoIdentityBody(t, getCredentialsResp)
	credentials, ok := getCredentialsBody["Credentials"].(map[string]any)
	if !ok {
		t.Fatalf("expected credentials object in response, got %#v", getCredentialsBody["Credentials"])
	}
	if accessKeyID, _ := credentials["AccessKeyId"].(string); strings.TrimSpace(accessKeyID) == "" {
		t.Fatalf("expected AccessKeyId in credentials response")
	}

	getOpenIDResp := cognitoIdentityRequestPayload(t, ts, "GetOpenIdToken", map[string]any{
		"IdentityId": identityID,
	})
	assertStatus(t, getOpenIDResp, http.StatusOK)
	getOpenIDBody := decodeCognitoIdentityBody(t, getOpenIDResp)
	if token, _ := getOpenIDBody["Token"].(string); strings.TrimSpace(token) == "" {
		t.Fatalf("expected token in GetOpenIdToken response")
	}
}

func TestCognitoIdentityStage12ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cognitoIdentityRequestPayload(t, ts, "CreateIdentityPool", map[string]any{
		"IdentityPoolName":               "stackyard-stage12-implemented",
		"AllowUnauthenticatedIdentities": true,
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := decodeCognitoIdentityBody(t, createResp)
	poolID, _ := createBody["IdentityPoolId"].(string)

	getIDResp := cognitoIdentityRequestPayload(t, ts, "GetId", map[string]any{
		"IdentityPoolId": poolID,
	})
	assertStatus(t, getIDResp, http.StatusOK)
	getIDBody := decodeCognitoIdentityBody(t, getIDResp)
	identityID, _ := getIDBody["IdentityId"].(string)

	cases := []struct {
		action  string
		payload map[string]any
	}{
		{action: "DescribeIdentityPool", payload: map[string]any{"IdentityPoolId": poolID}},
		{action: "ListIdentityPools", payload: map[string]any{"MaxResults": 10}},
		{action: "UpdateIdentityPool", payload: map[string]any{
			"IdentityPoolId":                 poolID,
			"IdentityPoolName":               "stackyard-stage12-implemented-updated",
			"AllowUnauthenticatedIdentities": true,
		}},
		{action: "DescribeIdentity", payload: map[string]any{"IdentityId": identityID}},
		{action: "ListIdentities", payload: map[string]any{"IdentityPoolId": poolID, "MaxResults": 10}},
		{action: "GetCredentialsForIdentity", payload: map[string]any{"IdentityId": identityID}},
		{action: "GetOpenIdToken", payload: map[string]any{"IdentityId": identityID}},
	}

	for _, tc := range cases {
		resp := cognitoIdentityRequestPayload(t, ts, tc.action, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", tc.action, resp.StatusCode, body)
		}
	}
}
