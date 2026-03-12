package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRecycleBinShard7RuleResponsesOmitExcludeResourceTags(t *testing.T) {
	store := newRecycleBinStore()

	createOut := store.Handle("CreateRule", map[string]any{
		"Description":  "shard7",
		"ResourceType": "EBS_SNAPSHOT",
	}, nil, nil)
	if _, ok := createOut["ExcludeResourceTags"]; ok {
		t.Fatalf("expected CreateRule response to omit ExcludeResourceTags")
	}

	identifier, _ := createOut["Identifier"].(string)
	if identifier == "" {
		t.Fatalf("expected Identifier in CreateRule response")
	}

	for _, action := range []string{"GetRule", "LockRule", "UnlockRule", "UpdateRule"} {
		out := store.Handle(action, map[string]any{}, map[string]string{"identifier": identifier}, url.Values{})
		if _, ok := out["ExcludeResourceTags"]; ok {
			t.Fatalf("expected %s response to omit ExcludeResourceTags", action)
		}
	}

	listOut := store.Handle("ListRules", map[string]any{}, nil, nil)
	items, _ := listOut["Rules"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected at least one recycle bin rule")
	}
	first, _ := items[0].(map[string]any)
	if _, ok := first["ExcludeResourceTags"]; ok {
		t.Fatalf("expected ListRules items to omit ExcludeResourceTags")
	}
}

func TestSingleSignOnShard7ModeledShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := singleSignOnRequest(t, ts, "DescribeInstance", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var describeInstanceOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeInstanceOut); err != nil {
		t.Fatalf("unmarshal describe instance response: %v", err)
	}
	if _, ok := describeInstanceOut["Instance"]; ok {
		t.Fatalf("expected DescribeInstance response to be flattened")
	}
	if _, ok := describeInstanceOut["StatusReason"]; ok {
		t.Fatalf("expected DescribeInstance response to omit StatusReason")
	}
	if describeInstanceOut["InstanceArn"] == nil {
		t.Fatalf("expected DescribeInstance response to include InstanceArn")
	}

	resp = singleSignOnRequest(t, ts, "ListInstances", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var listInstancesOut struct {
		Instances []map[string]any `json:"Instances"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listInstancesOut); err != nil {
		t.Fatalf("unmarshal list instances response: %v", err)
	}
	if len(listInstancesOut.Instances) == 0 {
		t.Fatalf("expected at least one instance")
	}
	if _, ok := listInstancesOut.Instances[0]["StatusReason"]; ok {
		t.Fatalf("expected ListInstances items to omit StatusReason")
	}

	resp = singleSignOnRequest(t, ts, "DescribeApplication", `{
		"ApplicationArn":"arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var describeApplicationOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeApplicationOut); err != nil {
		t.Fatalf("unmarshal describe application response: %v", err)
	}
	if _, ok := describeApplicationOut["Application"]; ok {
		t.Fatalf("expected DescribeApplication response to be flattened")
	}
	if describeApplicationOut["ApplicationArn"] == nil {
		t.Fatalf("expected DescribeApplication response to include ApplicationArn")
	}

	resp = singleSignOnRequest(t, ts, "CreateApplicationAssignment", `{
		"ApplicationArn":"arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001",
		"PrincipalId":"11111111-2222-3333-4444-555555555555",
		"PrincipalType":"USER"
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = singleSignOnRequest(t, ts, "DescribeApplicationAssignment", `{
		"ApplicationArn":"arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001",
		"PrincipalId":"11111111-2222-3333-4444-555555555555",
		"PrincipalType":"USER"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var describeAssignmentOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeAssignmentOut); err != nil {
		t.Fatalf("unmarshal describe application assignment response: %v", err)
	}
	if _, ok := describeAssignmentOut["ApplicationAssignment"]; ok {
		t.Fatalf("expected DescribeApplicationAssignment response to be flattened")
	}
	if describeAssignmentOut["ApplicationArn"] == nil || describeAssignmentOut["PrincipalId"] == nil {
		t.Fatalf("expected DescribeApplicationAssignment response to include modeled fields")
	}

	resp = singleSignOnRequest(t, ts, "DescribeApplicationProvider", `{}`)
	assertStatus(t, resp, http.StatusOK)
	var describeProviderOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeProviderOut); err != nil {
		t.Fatalf("unmarshal describe application provider response: %v", err)
	}
	if _, ok := describeProviderOut["ApplicationProvider"]; ok {
		t.Fatalf("expected DescribeApplicationProvider response to be flattened")
	}
	if describeProviderOut["ApplicationProviderArn"] == nil {
		t.Fatalf("expected DescribeApplicationProvider response to include ApplicationProviderArn")
	}

	resp = singleSignOnRequest(t, ts, "DescribeTrustedTokenIssuer", `{
		"TrustedTokenIssuerArn":"arn:aws:sso::123456789012:trustedTokenIssuer/ssoins-0000000000000000/tti-0000000000000001"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var describeTTIOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &describeTTIOut); err != nil {
		t.Fatalf("unmarshal describe trusted token issuer response: %v", err)
	}
	if _, ok := describeTTIOut["TrustedTokenIssuer"]; ok {
		t.Fatalf("expected DescribeTrustedTokenIssuer response to be flattened")
	}
	if describeTTIOut["TrustedTokenIssuerArn"] == nil {
		t.Fatalf("expected DescribeTrustedTokenIssuer response to include TrustedTokenIssuerArn")
	}

	resp = singleSignOnRequest(t, ts, "GetApplicationAuthenticationMethod", `{
		"ApplicationArn":"arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001",
		"AuthenticationMethodType":"IAM"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var getAuthMethodOut struct {
		AuthenticationMethod struct {
			Iam struct {
				ActorPolicy map[string]any `json:"ActorPolicy"`
			} `json:"Iam"`
		} `json:"AuthenticationMethod"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getAuthMethodOut); err != nil {
		t.Fatalf("unmarshal get application authentication method response: %v", err)
	}
	if getAuthMethodOut.AuthenticationMethod.Iam.ActorPolicy == nil {
		t.Fatalf("expected GetApplicationAuthenticationMethod to return AuthenticationMethod.Iam.ActorPolicy")
	}

	resp = singleSignOnRequest(t, ts, "ListApplicationAccessScopes", `{
		"ApplicationArn":"arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var listScopesOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &listScopesOut); err != nil {
		t.Fatalf("unmarshal list application access scopes response: %v", err)
	}
	if _, ok := listScopesOut["ApplicationAccessScopes"]; ok {
		t.Fatalf("expected ListApplicationAccessScopes to use Scopes")
	}
	if _, ok := listScopesOut["Scopes"]; !ok {
		t.Fatalf("expected ListApplicationAccessScopes response to include Scopes")
	}

	resp = singleSignOnRequest(t, ts, "ListApplicationGrants", `{
		"ApplicationArn":"arn:aws:sso::123456789012:application/ssoins-0000000000000000/apl-0000000000000001"
	}`)
	assertStatus(t, resp, http.StatusOK)
	var listGrantsOut struct {
		Grants []map[string]any `json:"Grants"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listGrantsOut); err != nil {
		t.Fatalf("unmarshal list application grants response: %v", err)
	}
	if len(listGrantsOut.Grants) == 0 {
		t.Fatalf("expected at least one application grant")
	}
	if listGrantsOut.Grants[0]["GrantType"] == nil || listGrantsOut.Grants[0]["Grant"] == nil {
		t.Fatalf("expected ListApplicationGrants items to include GrantType and Grant")
	}
}
