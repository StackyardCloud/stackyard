package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIAMStage0CatalogCoverage(t *testing.T) {
	if len(iamOperations) != 177 {
		t.Fatalf("expected 177 IAM operations from docs, got %d", len(iamOperations))
	}
	if len(iamOperationByName) != len(iamOperations) {
		t.Fatalf("expected unique IAM operation names")
	}

	requiredActions := []string{
		"ListUsers",
		"ListGroups",
		"ListRoles",
		"CreateUser",
		"CreateRole",
		"CreatePolicy",
		"GetAccountSummary",
		"SimulatePrincipalPolicy",
	}
	for _, action := range requiredActions {
		if _, ok := iamOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iamDataTypes) != 57 {
		t.Fatalf("expected 57 IAM data types from docs, got %d", len(iamDataTypes))
	}
	if len(iamDataTypeByName) != len(iamDataTypes) {
		t.Fatalf("expected unique IAM data type names")
	}
}

func iamRequest(t *testing.T, ts *httptest.Server, action string, params url.Values) *http.Response {
	t.Helper()
	if params == nil {
		params = url.Values{}
	}
	params.Set("Action", action)
	params.Set("Version", "2010-05-08")
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(params.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
		"iam",
	)
}

func TestIAMStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iamRequest(t, ts, "DefinitelyUnknownAction", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestIAMStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iamRequest(t, ts, "ListUsers", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ListUsersResponse") {
		t.Fatalf("expected ListUsersResponse in body, got %q", body)
	}
}

func TestIAMStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range iamOperations {
		resp := iamRequest(t, ts, op.Name, nil)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}

func TestIAMUserGroupPolicyLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iamRequest(t, ts, "CreateUser", url.Values{
		"UserName": []string{"stage1-user"},
		"Path":     []string{"/engineering/"},
	})
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage1-user") {
		t.Fatalf("expected CreateUser response to include stage1-user, got %s", body)
	}

	resp = iamRequest(t, ts, "CreateGroup", url.Values{
		"GroupName": []string{"stage1-group"},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = iamRequest(t, ts, "AddUserToGroup", url.Values{
		"UserName":  []string{"stage1-user"},
		"GroupName": []string{"stage1-group"},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = iamRequest(t, ts, "ListGroupsForUser", url.Values{
		"UserName": []string{"stage1-user"},
	})
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage1-group") {
		t.Fatalf("expected ListGroupsForUser response to include stage1-group, got %s", body)
	}

	resp = iamRequest(t, ts, "CreatePolicy", url.Values{
		"PolicyName":     []string{"stage1-policy"},
		"PolicyDocument": []string{`{"Version":"2012-10-17","Statement":[]}`},
	})
	assertStatus(t, resp, http.StatusOK)
	policyARN := "arn:aws:iam::123456789012:policy/stage1-policy"

	resp = iamRequest(t, ts, "AttachUserPolicy", url.Values{
		"UserName":  []string{"stage1-user"},
		"PolicyArn": []string{policyARN},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = iamRequest(t, ts, "ListAttachedUserPolicies", url.Values{
		"UserName": []string{"stage1-user"},
	})
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage1-policy") {
		t.Fatalf("expected ListAttachedUserPolicies response to include stage1-policy, got %s", body)
	}

	resp = iamRequest(t, ts, "DetachUserPolicy", url.Values{
		"UserName":  []string{"stage1-user"},
		"PolicyArn": []string{policyARN},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = iamRequest(t, ts, "ListAttachedUserPolicies", url.Values{
		"UserName": []string{"stage1-user"},
	})
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, "stage1-policy") {
		t.Fatalf("expected detached policy to be absent, got %s", body)
	}

	resp = iamRequest(t, ts, "RemoveUserFromGroup", url.Values{
		"UserName":  []string{"stage1-user"},
		"GroupName": []string{"stage1-group"},
	})
	assertStatus(t, resp, http.StatusOK)
	resp = iamRequest(t, ts, "DeleteGroup", url.Values{
		"GroupName": []string{"stage1-group"},
	})
	assertStatus(t, resp, http.StatusOK)
	resp = iamRequest(t, ts, "DeletePolicy", url.Values{
		"PolicyArn": []string{policyARN},
	})
	assertStatus(t, resp, http.StatusOK)
	resp = iamRequest(t, ts, "DeleteUser", url.Values{
		"UserName": []string{"stage1-user"},
	})
	assertStatus(t, resp, http.StatusOK)
}
