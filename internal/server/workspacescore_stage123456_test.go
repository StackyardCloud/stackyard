package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWorkSpacesCoreStage12LifecycleAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesCoreRequest(t, ts, "CreateWorkspaces", `{"Workspaces":[{"DirectoryId":"d-000001","BundleId":"wsb-000001","UserName":"core-stage-user"}]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "WorkspaceId") {
		t.Fatalf("expected CreateWorkspaces to include WorkspaceId, got %q", body)
	}

	resp = workspacesCoreRequest(t, ts, "DescribeWorkspaces", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "core-stage-user") {
		t.Fatalf("expected DescribeWorkspaces to include core-stage-user, got %q", body)
	}

	resp = workspacesCoreRequest(t, ts, "StopWorkspaces", `{"StopWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "StartWorkspaces", `{"StartWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "RebootWorkspaces", `{"RebootWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "RebuildWorkspaces", `{"RebuildWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "RestoreWorkspace", `{"WorkspaceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "MigrateWorkspace", `{"SourceWorkspaceId":"ws-000001","BundleId":"wsb-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "TerminateWorkspaces", `{"TerminateWorkspaceRequests":[{"WorkspaceId":"ws-000001"}]}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesCoreStage34DirectoryBundleAndImageSurfaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesCoreRequest(t, ts, "DescribeAccount", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DescribeAccountModifications", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesCoreRequest(t, ts, "RegisterWorkspaceDirectory", `{"DirectoryId":"d-core-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DescribeWorkspaceDirectories", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "d-core-001") {
		t.Fatalf("expected DescribeWorkspaceDirectories to include d-core-001, got %q", body)
	}

	resp = workspacesCoreRequest(t, ts, "CreateWorkspaceBundle", `{"BundleName":"core-stage-bundle"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "BundleId") {
		t.Fatalf("expected CreateWorkspaceBundle response to include BundleId, got %q", body)
	}
	resp = workspacesCoreRequest(t, ts, "DescribeWorkspaceBundles", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "UpdateWorkspaceBundle", `{"BundleId":"wsb-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesCoreRequest(t, ts, "CreateWorkspaceImage", `{"Name":"core-stage-image"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "CopyWorkspaceImage", `{"Name":"core-stage-image-copy","SourceImageId":"wsi-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "ImportWorkspaceImage", `{"Name":"core-stage-image-import"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DescribeWorkspaceImages", `{"MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DescribeWorkspaceImagePermissions", `{"ImageId":"wsi-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "UpdateWorkspaceImagePermission", `{"ImageId":"wsi-000001","AllowCopyImage":true,"SharedAccountId":"123456789012"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesCoreRequest(t, ts, "DescribeWorkspaceSnapshots", `{"WorkspaceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesCoreRequest(t, ts, "DeleteWorkspaceImage", `{"ImageId":"wsi-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DeleteWorkspaceBundle", `{"BundleId":"wsb-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DeregisterWorkspaceDirectory", `{"DirectoryId":"d-core-001"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestWorkSpacesCoreStage56TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesCoreRequest(t, ts, "CreateTags", `{"ResourceId":"ws-000001","Tags":[{"Key":"env","Value":"stage"},{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "CreateTags", `{"ResourceId":"ws-000001","Tags":[{"Key":"owner","Value":"qa"}]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesCoreRequest(t, ts, "DescribeTags", `{"ResourceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected DescribeTags to include owner tag, got %q", body)
	}

	resp = workspacesCoreRequest(t, ts, "DeleteTags", `{"ResourceId":"ws-000001","TagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = workspacesCoreRequest(t, ts, "DescribeTags", `{"ResourceId":"ws-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"Key":"owner"`) {
		t.Fatalf("expected owner tag to be removed, got %q", body)
	}

	resp = workspacesCoreRequest(t, ts, "ListAvailableManagementCidrRanges", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = workspacesCoreRequest(t, ts, "TotallyUnknownCoreAction", `{}`)
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
