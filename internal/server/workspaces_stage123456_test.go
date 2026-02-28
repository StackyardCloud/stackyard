package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkSpacesStage12WorkspaceLifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesRequest(t, ts, "CreateWorkspaces", `{"Workspaces":[{"DirectoryId":"d-000001","BundleId":"wsb-000001","UserName":"stage-workspace-user"}]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "WorkspaceId") {
		t.Fatalf("expected CreateWorkspaces to include WorkspaceId, got %q", body)
	}

	resp = workspacesRequest(t, ts, "DescribeWorkspaces", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-workspace-user") {
		t.Fatalf("expected DescribeWorkspaces to include stage-workspace-user, got %q", body)
	}

	resp = workspacesRequest(t, ts, "DescribeWorkspacesConnectionStatus", `{"WorkspaceIds":["ws-000001"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesRequest(t, ts, "StopWorkspaces", `{"StopWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "StartWorkspaces", `{"StartWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "RebootWorkspaces", `{"RebootWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "RebuildWorkspaces", `{"RebuildWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "TerminateWorkspaces", `{"TerminateWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesStage34BundleImagePoolAndAssociationSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesRequest(t, ts, "CreateWorkspaceBundle", `{"BundleName":"stage-workspaces-bundle"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeWorkspaceBundles", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "BundleId") {
		t.Fatalf("expected DescribeWorkspaceBundles to include BundleId, got %q", body)
	}

	resp = workspacesRequest(t, ts, "CreateWorkspaceImage", `{"Name":"stage-workspaces-image"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeWorkspaceImages", `{"MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesRequest(t, ts, "CreateConnectionAlias", `{"ConnectionString":"stage-workspaces-alias"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeConnectionAliases", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesRequest(t, ts, "CreateWorkspacesPool", `{"PoolName":"stage-workspaces-pool"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeWorkspacesPools", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeWorkspacesPoolSessions", `{"WorkspacesPoolId":"wspool-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesRequest(t, ts, "DescribeApplications", `{"MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeApplicationAssociations", `{"ApplicationId":"wsapp-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeWorkspaceAssociations", `{"WorkspaceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesRequest(t, ts, "CreateTags", `{"ResourceId":"ws-000001","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesRequest(t, ts, "DescribeTags", `{"ResourceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected DescribeTags to include owner tag, got %q", body)
	}

	resp = workspacesRequest(t, ts, "DeleteTags", `{"ResourceId":"ws-000001","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesRequest(t, ts, "DescribeTags", `{"ResourceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"Key":"owner"`) {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = workspacesRequest(t, ts, "TotallyUnknownAction", `{}`)
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
			"X-Amz-Target": "WorkspacesService.DescribeWorkspaces",
		},
		"workspaces",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
