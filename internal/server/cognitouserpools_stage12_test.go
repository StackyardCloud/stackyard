package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cognitoUserPoolsRequestPayload(t *testing.T, ts *httptest.Server, action string, payload map[string]any) *http.Response {
	t.Helper()
	body := []byte(`{}`)
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = encoded
	}
	return cognitoUserPoolsRequest(t, ts, action, body)
}

func decodeCognitoUserPoolsBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := mustBody(t, resp)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	out := map[string]any{}
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decode JSON body: %v body=%s", err, string(raw))
	}
	return out
}

func TestCognitoUserPoolsStage12UserPoolAndDomainLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPool", map[string]any{
		"PoolName":         "stackyard-stage12-pool",
		"MfaConfiguration": "OFF",
		"UserPoolTags": map[string]any{
			"env": "test",
		},
	})
	assertStatus(t, createResp, http.StatusOK)
	createBody := decodeCognitoUserPoolsBody(t, createResp)
	userPoolObj, ok := createBody["UserPool"].(map[string]any)
	if !ok {
		t.Fatalf("expected UserPool object, got %#v", createBody["UserPool"])
	}
	userPoolID, _ := userPoolObj["Id"].(string)
	if strings.TrimSpace(userPoolID) == "" {
		t.Fatalf("expected user pool id in create response: %#v", userPoolObj)
	}

	describeResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeUserPool", map[string]any{"UserPoolId": userPoolID})
	assertStatus(t, describeResp, http.StatusOK)
	describeBody := decodeCognitoUserPoolsBody(t, describeResp)
	describedPool, ok := describeBody["UserPool"].(map[string]any)
	if !ok {
		t.Fatalf("expected UserPool object in describe response, got %#v", describeBody["UserPool"])
	}
	if got, _ := describedPool["Id"].(string); got != userPoolID {
		t.Fatalf("expected described user pool id %q, got %#v", userPoolID, describedPool["Id"])
	}

	listResp := cognitoUserPoolsRequestPayload(t, ts, "ListUserPools", map[string]any{"MaxResults": 10})
	assertStatus(t, listResp, http.StatusOK)
	listBody := decodeCognitoUserPoolsBody(t, listResp)
	userPools, ok := listBody["UserPools"].([]any)
	if !ok || len(userPools) == 0 {
		t.Fatalf("expected UserPools list with at least one entry, got %#v", listBody["UserPools"])
	}

	updateResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateUserPool", map[string]any{
		"UserPoolId":       userPoolID,
		"MfaConfiguration": "OPTIONAL",
		"UserPoolTags": map[string]any{
			"env":  "stage12",
			"team": "identity",
		},
	})
	assertStatus(t, updateResp, http.StatusOK)

	domain := "stackyard-stage12-domain"
	createDomainResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     domain,
	})
	assertStatus(t, createDomainResp, http.StatusOK)

	describeDomainResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeUserPoolDomain", map[string]any{"Domain": domain})
	assertStatus(t, describeDomainResp, http.StatusOK)
	describeDomainBody := decodeCognitoUserPoolsBody(t, describeDomainResp)
	domainDescription, ok := describeDomainBody["DomainDescription"].(map[string]any)
	if !ok {
		t.Fatalf("expected DomainDescription object, got %#v", describeDomainBody["DomainDescription"])
	}
	if got, _ := domainDescription["UserPoolId"].(string); got != userPoolID {
		t.Fatalf("expected domain to reference user pool id %q, got %#v", userPoolID, domainDescription["UserPoolId"])
	}

	deleteDomainResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     domain,
	})
	assertStatus(t, deleteDomainResp, http.StatusOK)

	describeDeletedDomainResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeUserPoolDomain", map[string]any{"Domain": domain})
	assertStatus(t, describeDeletedDomainResp, http.StatusBadRequest)
	describeDeletedDomainBody := string(mustBody(t, describeDeletedDomainResp))
	if !strings.Contains(describeDeletedDomainBody, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException for deleted domain, got %q", describeDeletedDomainBody)
	}

	deletePoolResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteUserPool", map[string]any{"UserPoolId": userPoolID})
	assertStatus(t, deletePoolResp, http.StatusOK)

	describeDeletedPoolResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeUserPool", map[string]any{"UserPoolId": userPoolID})
	assertStatus(t, describeDeletedPoolResp, http.StatusBadRequest)
	describeDeletedPoolBody := string(mustBody(t, describeDeletedPoolResp))
	if !strings.Contains(describeDeletedPoolBody, "ResourceNotFoundException") {
		t.Fatalf("expected ResourceNotFoundException for deleted user pool, got %q", describeDeletedPoolBody)
	}
}

func TestCognitoUserPoolsStage12ClientResourceServerAndTagFlows(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createPoolResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPool", map[string]any{
		"PoolName": "stackyard-stage12-client-pool",
	})
	assertStatus(t, createPoolResp, http.StatusOK)
	createPoolBody := decodeCognitoUserPoolsBody(t, createPoolResp)
	poolObj, _ := createPoolBody["UserPool"].(map[string]any)
	userPoolID, _ := poolObj["Id"].(string)
	userPoolARN, _ := poolObj["Arn"].(string)
	if strings.TrimSpace(userPoolID) == "" || strings.TrimSpace(userPoolARN) == "" {
		t.Fatalf("expected user pool id and arn in create response: %#v", poolObj)
	}

	createClientResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolClient", map[string]any{
		"UserPoolId":     userPoolID,
		"ClientName":     "stackyard-stage12-client",
		"GenerateSecret": true,
		"ExplicitAuthFlows": []string{
			"ALLOW_USER_PASSWORD_AUTH",
			"ALLOW_REFRESH_TOKEN_AUTH",
		},
		"RefreshTokenValidity": 10,
	})
	assertStatus(t, createClientResp, http.StatusOK)
	createClientBody := decodeCognitoUserPoolsBody(t, createClientResp)
	clientObj, _ := createClientBody["UserPoolClient"].(map[string]any)
	clientID, _ := clientObj["ClientId"].(string)
	if strings.TrimSpace(clientID) == "" {
		t.Fatalf("expected client id in create response: %#v", clientObj)
	}

	describeClientResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, describeClientResp, http.StatusOK)

	listClientsResp := cognitoUserPoolsRequestPayload(t, ts, "ListUserPoolClients", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	})
	assertStatus(t, listClientsResp, http.StatusOK)

	updateClientResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateUserPoolClient", map[string]any{
		"UserPoolId":           userPoolID,
		"ClientId":             clientID,
		"ClientName":           "stackyard-stage12-client-updated",
		"RefreshTokenValidity": 14,
	})
	assertStatus(t, updateClientResp, http.StatusOK)

	createResourceServerResp := cognitoUserPoolsRequestPayload(t, ts, "CreateResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": "https://api.stackyard.local",
		"Name":       "stackyard-api",
		"Scopes": []map[string]any{
			{
				"ScopeName":        "read",
				"ScopeDescription": "Read access",
			},
		},
	})
	assertStatus(t, createResourceServerResp, http.StatusOK)

	describeResourceServerResp := cognitoUserPoolsRequestPayload(t, ts, "DescribeResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": "https://api.stackyard.local",
	})
	assertStatus(t, describeResourceServerResp, http.StatusOK)

	listResourceServersResp := cognitoUserPoolsRequestPayload(t, ts, "ListResourceServers", map[string]any{
		"UserPoolId": userPoolID,
		"MaxResults": 10,
	})
	assertStatus(t, listResourceServersResp, http.StatusOK)

	updateResourceServerResp := cognitoUserPoolsRequestPayload(t, ts, "UpdateResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": "https://api.stackyard.local",
		"Name":       "stackyard-api-v2",
		"Scopes": []map[string]any{
			{
				"ScopeName":        "read",
				"ScopeDescription": "Read access",
			},
			{
				"ScopeName":        "write",
				"ScopeDescription": "Write access",
			},
		},
	})
	assertStatus(t, updateResourceServerResp, http.StatusOK)

	tagResp := cognitoUserPoolsRequestPayload(t, ts, "TagResource", map[string]any{
		"ResourceArn": userPoolARN,
		"Tags": map[string]any{
			"env":  "stage12",
			"team": "identity",
		},
	})
	assertStatus(t, tagResp, http.StatusOK)

	listTagsResp := cognitoUserPoolsRequestPayload(t, ts, "ListTagsForResource", map[string]any{
		"ResourceArn": userPoolARN,
	})
	assertStatus(t, listTagsResp, http.StatusOK)
	listTagsBody := decodeCognitoUserPoolsBody(t, listTagsResp)
	tags, ok := listTagsBody["Tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected tags object, got %#v", listTagsBody["Tags"])
	}
	if got, _ := tags["env"].(string); got != "stage12" {
		t.Fatalf("expected env=stage12 tag, got %#v", tags["env"])
	}

	untagResp := cognitoUserPoolsRequestPayload(t, ts, "UntagResource", map[string]any{
		"ResourceArn": userPoolARN,
		"TagKeys":     []string{"team"},
	})
	assertStatus(t, untagResp, http.StatusOK)

	listTagsAfterUntagResp := cognitoUserPoolsRequestPayload(t, ts, "ListTagsForResource", map[string]any{
		"ResourceArn": userPoolARN,
	})
	assertStatus(t, listTagsAfterUntagResp, http.StatusOK)
	listTagsAfterUntagBody := decodeCognitoUserPoolsBody(t, listTagsAfterUntagResp)
	tagsAfterUntag, ok := listTagsAfterUntagBody["Tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected tags object after untag, got %#v", listTagsAfterUntagBody["Tags"])
	}
	if _, exists := tagsAfterUntag["team"]; exists {
		t.Fatalf("expected team tag to be removed, got %#v", tagsAfterUntag)
	}

	deleteResourceServerResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": "https://api.stackyard.local",
	})
	assertStatus(t, deleteResourceServerResp, http.StatusOK)

	deleteClientResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, deleteClientResp, http.StatusOK)

	deletePoolResp := cognitoUserPoolsRequestPayload(t, ts, "DeleteUserPool", map[string]any{
		"UserPoolId": userPoolID,
	})
	assertStatus(t, deletePoolResp, http.StatusOK)
}

func TestCognitoUserPoolsStage12ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createPoolResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPool", map[string]any{"PoolName": "stackyard-stage12-implemented"})
	assertStatus(t, createPoolResp, http.StatusOK)
	createPoolBody := decodeCognitoUserPoolsBody(t, createPoolResp)
	poolObj, _ := createPoolBody["UserPool"].(map[string]any)
	userPoolID, _ := poolObj["Id"].(string)
	userPoolARN, _ := poolObj["Arn"].(string)

	createClientResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolClient", map[string]any{
		"UserPoolId": userPoolID,
		"ClientName": "stackyard-stage12-implemented-client",
	})
	assertStatus(t, createClientResp, http.StatusOK)
	createClientBody := decodeCognitoUserPoolsBody(t, createClientResp)
	clientObj, _ := createClientBody["UserPoolClient"].(map[string]any)
	clientID, _ := clientObj["ClientId"].(string)

	domain := "stackyard-stage12-implemented-domain"
	createDomainResp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     domain,
	})
	assertStatus(t, createDomainResp, http.StatusOK)

	createResourceServerResp := cognitoUserPoolsRequestPayload(t, ts, "CreateResourceServer", map[string]any{
		"UserPoolId": userPoolID,
		"Identifier": "https://implemented.stackyard.local",
		"Name":       "implemented-api",
		"Scopes": []map[string]any{{
			"ScopeName":        "read",
			"ScopeDescription": "Read scope",
		}},
	})
	assertStatus(t, createResourceServerResp, http.StatusOK)

	cases := []struct {
		action  string
		payload map[string]any
	}{
		{action: "DescribeUserPool", payload: map[string]any{"UserPoolId": userPoolID}},
		{action: "ListUserPools", payload: map[string]any{"MaxResults": 10}},
		{action: "UpdateUserPool", payload: map[string]any{"UserPoolId": userPoolID, "MfaConfiguration": "OPTIONAL"}},
		{action: "DescribeUserPoolDomain", payload: map[string]any{"Domain": domain}},
		{action: "CreateUserPoolClient", payload: map[string]any{"UserPoolId": userPoolID, "ClientName": "stage12-extra-client"}},
		{action: "DescribeUserPoolClient", payload: map[string]any{"UserPoolId": userPoolID, "ClientId": clientID}},
		{action: "ListUserPoolClients", payload: map[string]any{"UserPoolId": userPoolID, "MaxResults": 10}},
		{action: "UpdateUserPoolClient", payload: map[string]any{"UserPoolId": userPoolID, "ClientId": clientID, "ClientName": "updated"}},
		{action: "TagResource", payload: map[string]any{"ResourceArn": userPoolARN, "Tags": map[string]any{"env": "test"}}},
		{action: "ListTagsForResource", payload: map[string]any{"ResourceArn": userPoolARN}},
		{action: "UntagResource", payload: map[string]any{"ResourceArn": userPoolARN, "TagKeys": []string{"env"}}},
		{action: "DescribeResourceServer", payload: map[string]any{"UserPoolId": userPoolID, "Identifier": "https://implemented.stackyard.local"}},
		{action: "ListResourceServers", payload: map[string]any{"UserPoolId": userPoolID, "MaxResults": 10}},
		{action: "UpdateResourceServer", payload: map[string]any{"UserPoolId": userPoolID, "Identifier": "https://implemented.stackyard.local", "Name": "implemented-api-v2", "Scopes": []map[string]any{{"ScopeName": "read", "ScopeDescription": "Read scope"}}}},
		{action: "DeleteResourceServer", payload: map[string]any{"UserPoolId": userPoolID, "Identifier": "https://implemented.stackyard.local"}},
		{action: "DeleteUserPoolDomain", payload: map[string]any{"UserPoolId": userPoolID, "Domain": domain}},
		{action: "DeleteUserPoolClient", payload: map[string]any{"UserPoolId": userPoolID, "ClientId": clientID}},
		{action: "DeleteUserPool", payload: map[string]any{"UserPoolId": userPoolID}},
	}

	for _, tc := range cases {
		resp := cognitoUserPoolsRequestPayload(t, ts, tc.action, tc.payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", tc.action, resp.StatusCode, body)
		}
	}
}
