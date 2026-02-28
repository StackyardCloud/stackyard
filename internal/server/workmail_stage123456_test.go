package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkMailStage12OrganizationIdentityAndMembershipLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workmailRequest(t, ts, "CreateOrganization", `{"Alias":"stage-workmail"}`)
	assertStatus(t, resp, http.StatusOK)
	createBody := string(mustBody(t, resp))
	if !strings.Contains(createBody, "OrganizationId") {
		t.Fatalf("expected CreateOrganization to include OrganizationId, got %q", createBody)
	}

	resp = workmailRequest(t, ts, "ListOrganizations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-workmail") {
		t.Fatalf("expected ListOrganizations to include stage-workmail alias, got %q", body)
	}

	resp = workmailRequest(t, ts, "RegisterMailDomain", `{"OrganizationId":"m-000001","DomainName":"stage-workmail.example.com"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "UpdateDefaultMailDomain", `{"OrganizationId":"m-000001","DomainName":"stage-workmail.example.com"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListMailDomains", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "CreateUser", `{"OrganizationId":"m-000001","Name":"stage-user","DisplayName":"Stage User"}`)
	assertStatus(t, resp, http.StatusOK)
	createUser := string(mustBody(t, resp))
	if !strings.Contains(createUser, "UserId") {
		t.Fatalf("expected CreateUser to include UserId, got %q", createUser)
	}

	resp = workmailRequest(t, ts, "CreateGroup", `{"OrganizationId":"m-000001","Name":"stage-group"}`)
	assertStatus(t, resp, http.StatusOK)
	createGroup := string(mustBody(t, resp))
	if !strings.Contains(createGroup, "GroupId") {
		t.Fatalf("expected CreateGroup to include GroupId, got %q", createGroup)
	}

	resp = workmailRequest(t, ts, "AssociateMemberToGroup", `{"OrganizationId":"m-000001","GroupId":"g-000001","MemberId":"u-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListGroupMembers", `{"OrganizationId":"m-000001","GroupId":"g-000001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkMailStage34MailboxPolicyAndMobileAccessLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workmailRequest(t, ts, "PutMailboxPermissions", `{"OrganizationId":"m-000001","EntityId":"u-000001","GranteeId":"u-000001","PermissionValues":["FULL_ACCESS"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListMailboxPermissions", `{"OrganizationId":"m-000001","EntityId":"u-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "PutAccessControlRule", `{"OrganizationId":"m-000001","Name":"stage-acl","Effect":"ALLOW"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListAccessControlRules", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "GetAccessControlEffect", `{"OrganizationId":"m-000001","IpAddress":"10.0.0.10","Action":"GetUser"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "DeleteAccessControlRule", `{"OrganizationId":"m-000001","Name":"stage-acl"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "CreateMobileDeviceAccessRule", `{"OrganizationId":"m-000001","Name":"stage-mobile","Effect":"ALLOW"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListMobileDeviceAccessRules", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "PutMobileDeviceAccessOverride", `{"OrganizationId":"m-000001","UserId":"u-000001","DeviceId":"device-stage","Effect":"ALLOW"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListMobileDeviceAccessOverrides", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "GetMobileDeviceAccessEffect", `{"OrganizationId":"m-000001","DeviceType":"IOS"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkMailStage5AvailabilityImpersonationAndIdpLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workmailRequest(t, ts, "CreateAvailabilityConfiguration", `{"OrganizationId":"m-000001","DomainName":"stage-workmail.example.com"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListAvailabilityConfigurations", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "TestAvailabilityConfiguration", `{"OrganizationId":"m-000001","DomainName":"stage-workmail.example.com"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "CreateImpersonationRole", `{"OrganizationId":"m-000001","Name":"stage-role","Type":"FULL_ACCESS"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListImpersonationRoles", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "GetImpersonationRoleEffect", `{"OrganizationId":"m-000001","TargetUser":"u-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "PutIdentityProviderConfiguration", `{"OrganizationId":"m-000001","AuthenticationMode":"IDENTITY_PROVIDER_ONLY"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "DescribeIdentityProviderConfiguration", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "PutEmailMonitoringConfiguration", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "DescribeEmailMonitoringConfiguration", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkMailStage6ExportTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workmailRequest(t, ts, "StartMailboxExportJob", `{"OrganizationId":"m-000001","EntityId":"u-000001","Description":"stage export"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "JobId") {
		t.Fatalf("expected StartMailboxExportJob to include JobId, got %q", body)
	}

	resp = workmailRequest(t, ts, "ListMailboxExportJobs", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "DescribeMailboxExportJob", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "CancelMailboxExportJob", `{"OrganizationId":"m-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:workmail:us-east-1:123456789012:organization/m-000001"
	resp = workmailRequest(t, ts, "TagResource", `{"ResourceARN":"`+resourceARN+`","Tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workmailRequest(t, ts, "ListTagsForResource", `{"ResourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = workmailRequest(t, ts, "UntagResource", `{"ResourceARN":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workmailRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "WorkMailService.ListOrganizations",
		},
		"workmail",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
