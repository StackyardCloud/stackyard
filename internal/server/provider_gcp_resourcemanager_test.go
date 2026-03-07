package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPResourceManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v2/folders?parent=organizations/123456&pageSize=1", nil, "folders/1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v2/folders:search?query=parent=organizations/123456&pageSize=1", nil, "folders/1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v2/folders/1001", nil, "folders/1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/folders?parent=organizations/123456", []byte(`{"folder":{"displayName":"Team Folder"}}`), "operations/create-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPatch, "/gcp/v2/folders/1001?updateMask=display_name", []byte(`{"folder":{"name":"folders/1001","displayName":"Team Folder Updated"}}`), "Team Folder Updated")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/folders/1001:move", []byte(`{"destinationParent":"folders/2000"}`), "operations/move-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v2/folders/1001", nil, "DELETE_REQUESTED")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/folders/1001:undelete", []byte(`{}`), "ACTIVE")

	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/folders/1001:getIamPolicy", []byte(`{}`), "bindings")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/folders/1001:setIamPolicy", []byte(`{"policy":{"bindings":[{"role":"roles/viewer","members":["user:bob@example.com"]}]}}`), "roles/viewer")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/folders/1001:testIamPermissions", []byte(`{"permissions":["resourcemanager.folders.get"]}`), "resourcemanager.folders.get")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v2/operations?pageSize=1", nil, "operations/create-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v2/operations/create-folder-1001", nil, "operations/create-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v2/operations/create-folder-1001:cancel", []byte(`{}`), "{}")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v2/operations/create-folder-1001", nil, "{}")
}

func TestGCPResourceManagerRouter_V3RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/folders?parent=organizations/123456&pageSize=1", nil, "folders/1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/folders:search?query=lifecyclestate=active&pageSize=1", nil, "folders/1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/folders", []byte(`{"displayName":"Team Folder","parent":"organizations/123456"}`), "operations/create-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPatch, "/gcp/v3/folders/1001?updateMask=display_name", []byte(`{"name":"folders/1001","displayName":"Team Folder Updated"}`), "operations/update-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/folders/1001:move", []byte(`{"destinationParent":"folders/2000"}`), "operations/move-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/folders/1001", nil, "operations/delete-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/folders/1001:undelete", []byte(`{}`), "operations/undelete-folder-1001")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/projects?parent=organizations/123456&pageSize=1", nil, "projects/415104041262")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/projects:search?query=state=active&pageSize=1", nil, "projects/415104041262")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/projects/415104041262", nil, "projects/415104041262")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/projects", []byte(`{"projectId":"stackyard-prod","displayName":"Stackyard Prod","parent":"organizations/123456"}`), "operations/create-project-stackyard-prod")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPatch, "/gcp/v3/projects/415104041262?updateMask=display_name", []byte(`{"name":"projects/415104041262","displayName":"Stackyard Prod Updated"}`), "operations/update-project-415104041262")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/415104041262:move", []byte(`{"destinationParent":"folders/1001"}`), "operations/move-project-415104041262")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/projects/415104041262", nil, "operations/delete-project-415104041262")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/415104041262:undelete", []byte(`{}`), "operations/undelete-project-415104041262")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/organizations:search?query=domain:example.com&pageSize=1", nil, "organizations/123456")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/organizations/123456", nil, "organizations/123456")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagKeys?parent=organizations/123456&pageSize=1", nil, "tagKeys/2001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagKeys/2001", nil, "tagKeys/2001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagKeys/namespaced?name=123456/env", nil, "\"namespacedName\":\"123456/env\"")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/tagKeys", []byte(`{"parent":"organizations/123456","shortName":"env","description":"environment key"}`), "operations/create-tagkey-2001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPatch, "/gcp/v3/tagKeys/2001?updateMask=description", []byte(`{"name":"tagKeys/2001","description":"updated description"}`), "operations/update-tagkey-2001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/tagKeys/2001", nil, "operations/delete-tagkey-2001")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagValues?parent=tagKeys/2001&pageSize=1", nil, "tagValues/3001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagValues/3001", nil, "tagValues/3001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagValues/namespaced?name=123456/env/prod", nil, "\"namespacedName\":\"123456/env/prod\"")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/tagValues", []byte(`{"parent":"tagKeys/2001","shortName":"prod","description":"production value"}`), "operations/create-tagvalue-3001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPatch, "/gcp/v3/tagValues/3001?updateMask=description", []byte(`{"name":"tagValues/3001","description":"updated value description"}`), "operations/update-tagvalue-3001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/tagValues/3001", nil, "operations/delete-tagvalue-3001")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagBindings?parent=//cloudresourcemanager.googleapis.com/projects/415104041262&pageSize=1", nil, "tagBindings/%2F%2Fcloudresourcemanager.googleapis.com")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/tagBindings", []byte(`{"parent":"//cloudresourcemanager.googleapis.com/projects/415104041262","tagValue":"tagValues/3001"}`), "operations/create-tagbinding-3001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/tagBindings/%2F%2Fcloudresourcemanager.googleapis.com%2Fprojects%2F415104041262/tagValues/3001", nil, "operations/delete-tagbinding-")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/effectiveTags?parent=//cloudresourcemanager.googleapis.com/projects/415104041262&pageSize=1", nil, "\"effectiveTags\"")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/tagValues/3001/tagHolds?pageSize=1", nil, "tagValues/3001/tagHolds/hold-1")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/tagValues/3001/tagHolds", []byte(`{"holder":"//cloudresourcemanager.googleapis.com/projects/415104041262","origin":"stackyard"}`), "operations/create-taghold-3001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/tagValues/3001/tagHolds/hold-1", nil, "operations/delete-taghold-3001-hold-1")

	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/415104041262:getIamPolicy", []byte(`{}`), "\"bindings\"")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/415104041262:setIamPolicy", []byte(`{"policy":{"bindings":[{"role":"roles/viewer","members":["user:bob@example.com"]}]}}`), "roles/viewer")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/projects/415104041262:testIamPermissions", []byte(`{"permissions":["resourcemanager.projects.get"]}`), "resourcemanager.projects.get")

	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/operations?pageSize=1", nil, "operations/create-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v3/operations/create-folder-1001", nil, "operations/create-folder-1001")
	assertGCPResourceManagerSuccess(t, ts, http.MethodPost, "/gcp/v3/operations/create-folder-1001:cancel", []byte(`{}`), "{}")
	assertGCPResourceManagerSuccess(t, ts, http.MethodDelete, "/gcp/v3/operations/create-folder-1001", nil, "{}")
}

func TestGCPResourceManagerRouter_ListFoldersRequiresParent(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/folders?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_CreateFolderValidatesDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/folders?parent=organizations/123456", []byte(`{"folder":{"displayName":"Team*"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_UpdateFolderRejectsUnsupportedUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v2/folders/1001?updateMask=parent", []byte(`{"folder":{"name":"folders/1001","displayName":"Team"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_MoveFolderRequiresDestinationParent(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/folders/1001:move", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_SetIAMPolicyRequiresPolicy(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/folders/1001:setIamPolicy", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_UndeleteFolderFailsPreconditionWhenActive(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/folders/active-folder:undelete", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_V3CreateProjectValidatesProjectID(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects", []byte(`{"projectId":"BAD","displayName":"Stackyard Prod","parent":"organizations/123456"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager v3 create project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_V3UpdateTagValueRejectsBadMask(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v3/tagValues/3001?updateMask=short_name", []byte(`{"name":"tagValues/3001","description":"updated"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager v3 update tagValue, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_V3CreateTagBindingRejectsMutuallyExclusiveFields(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/tagBindings", []byte(`{"parent":"//cloudresourcemanager.googleapis.com/projects/415104041262","tagValue":"tagValues/3001","tagValueNamespacedName":"123456/env/prod"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager v3 create tagBinding, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_V3UndeleteProjectFailsPreconditionWhenActive(t *testing.T) {
	t.Parallel()

	ts := newGCPResourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/projects/active-project:undelete", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "resourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp resourcemanager v3 undelete project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPResourceManagerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/resourcemanager?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp resourcemanager contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "resourcemanager" {
		t.Fatalf("expected service=resourcemanager, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPResourceManagerContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPResourceManagerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "resourcemanager",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp resourcemanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
