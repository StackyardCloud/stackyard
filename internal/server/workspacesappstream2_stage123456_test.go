package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkSpacesAppStream2Stage12FleetStackLifecycleAndReads(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesAppStream2Request(t, ts, "CreateFleet", `{"Name":"stage-fleet-001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-fleet-001") {
		t.Fatalf("expected CreateFleet response to include stage-fleet-001, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "CreateStack", `{"Name":"stage-stack-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "AssociateFleet", `{"StackName":"stage-stack-001","FleetName":"stage-fleet-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesAppStream2Request(t, ts, "DescribeFleets", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-fleet-001") {
		t.Fatalf("expected DescribeFleets to include stage-fleet-001, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "ListAssociatedFleets", `{"StackName":"stage-stack-001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-fleet-001") {
		t.Fatalf("expected ListAssociatedFleets to include stage-fleet-001, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "StartFleet", `{"Name":"stage-fleet-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "StopFleet", `{"Name":"stage-fleet-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "DisassociateFleet", `{"StackName":"stage-stack-001","FleetName":"stage-fleet-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "DeleteStack", `{"Name":"stage-stack-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "DeleteFleet", `{"Name":"stage-fleet-001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesAppStream2Stage34ApplicationsUsersAndTasks(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesAppStream2Request(t, ts, "CreateApplication", `{"Name":"stage-app-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "AssociateApplicationFleet", `{"ApplicationName":"stage-app-001","FleetName":"stackyard-fleet"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesAppStream2Request(t, ts, "CreateEntitlement", `{"Name":"stage-entitlement-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "AssociateApplicationToEntitlement", `{"ApplicationIdentifier":"stage-app-001","EntitlementName":"stage-entitlement-001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesAppStream2Request(t, ts, "DescribeApplications", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-app-001") {
		t.Fatalf("expected DescribeApplications to include stage-app-001, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "CreateUser", `{"AuthenticationType":"API","UserName":"stage-user-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(
		t,
		ts,
		"BatchAssociateUserStack",
		`{"UserStackAssociations":[{"StackName":"stackyard-stack","UserName":"stage-user-001","AuthenticationType":"API","SendEmailNotification":false}]}`,
	)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "DescribeUserStackAssociations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-user-001") {
		t.Fatalf("expected DescribeUserStackAssociations to include stage-user-001, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "CreateExportImageTask", `{"ImageName":"stackyard-image"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ExportImageTaskId") {
		t.Fatalf("expected CreateExportImageTask response to include ExportImageTaskId, got %q", body)
	}
	resp = workspacesAppStream2Request(t, ts, "ListExportImageTasks", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "GetExportImageTask", `{"ExportImageTaskId":"export-task-000001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesAppStream2Stage56TaggingValidationAndJSONErrors(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:appstream:us-east-1:123456789012:fleet/stackyard-fleet"

	resp := workspacesAppStream2Request(t, ts, "TagResource", `{"ResourceArn":"`+resourceARN+`","Tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "ListTagsForResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "UntagResource", `{"ResourceArn":"`+resourceARN+`","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesAppStream2Request(t, ts, "ListTagsForResource", `{"ResourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"owner"`) {
		t.Fatalf("expected owner tag removed, got %q", body)
	}

	resp = workspacesAppStream2Request(t, ts, "TotallyUnknownAction", `{}`)
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
			"X-Amz-Target": "PhotonAdminProxyService.DescribeFleets",
		},
		"appstream",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
