package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecureSourceManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecureSourceManagerContractServer(t)

	baseLocation := "/gcp/v1/projects/stackyard/locations/us-central1"
	repoName := baseLocation + "/repositories/repository-1"
	pullRequestName := repoName + "/pullRequests/pull-request-1"
	issueName := repoName + "/issues/issue-1"

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `"locations":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `"service":"securesourcemanager.googleapis.com"`)

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, baseLocation+"/instances?pageSize=1", nil, `"instances":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, baseLocation+"/instances/instance-1", nil, `"ssm-instance-instance-1"`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, baseLocation+"/instances?instanceId=instance-3", []byte(`{"instance":{"displayName":"instance 3"}}`), "createInstance.instance-3")

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, baseLocation+"/repositories?pageSize=1", nil, `"repositories":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName, nil, "repositories/repository-1")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, baseLocation+"/repositories?repositoryId=repository-3", []byte(`{"repository":{"description":"repo 3"}}`), "createRepository.repository-3")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPatch, repoName+"?updateMask=description", []byte(`{"repository":{"name":"projects/stackyard/locations/us-central1/repositories/repository-1","description":"updated"}}`), "updateRepository.repository-1")

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+"/hooks?pageSize=1", nil, `"hooks":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+"/hooks/hook-1", nil, "hooks/hook-1")

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+":getIamPolicy", nil, `"bindings":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, repoName+":setIamPolicy", []byte(`{"policy":{"version":1,"bindings":[{"role":"roles/securesourcemanager.reader","members":["user:stackyard@example.com"]}]}}`), `"resource":"projects/stackyard/locations/us-central1/repositories/repository-1"`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, repoName+":testIamPermissions", []byte(`{"permissions":["securesourcemanager.repositories.get"]}`), `"permissions":["securesourcemanager.repositories.get"]`)

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+"/branchRules?pageSize=1", nil, `"branchRules":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+"/branchRules/main", nil, "branchRules/main")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, repoName+"/branchRules?branchRuleId=release-candidate", []byte(`{"branchRule":{"includePattern":"release/*"}}`), "createBranchRule.release-candidate")

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+"/pullRequests?pageSize=1", nil, `"pullRequests":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, pullRequestName, nil, "pullRequests/pull-request-1")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, pullRequestName+":close", []byte(`{}`), "closePullRequest.pull-request-1")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, pullRequestName+":listFileDiffs?pageSize=1", nil, `"fileDiffs":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+":fetchTree?pageSize=1&ref=main", nil, `"treeEntries":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+":fetchBlob?sha=abc123", nil, "abc123")

	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, repoName+"/issues?pageSize=1", nil, `"issues":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, issueName, nil, "issues/issue-1")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodPost, issueName+":close", []byte(`{}`), "closeIssue.issue-1")
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, pullRequestName+"/pullRequestComments?pageSize=1", nil, `"pullRequestComments":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, issueName+"/issueComments?pageSize=1", nil, `"issueComments":[`)
	assertGCPSecureSourceManagerSuccess(t, ts, http.MethodGet, baseLocation+"/operations?pageSize=1", nil, `"operations":[`)
}

func TestGCPSecureSourceManagerRouter_ListRepositoriesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSecureSourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securesourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securesourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecureSourceManagerRouter_CreateRepositoryRequiresDescription(t *testing.T) {
	t.Parallel()

	ts := newGCPSecureSourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?repositoryId=repository-1", []byte(`{"repository":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securesourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securesourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecureSourceManagerRouter_CloseMergedPullRequestFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSecureSourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/repository-1/pullRequests/pull-request-merged:close", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securesourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securesourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecureSourceManagerRouter_FetchBlobRequiresSHA(t *testing.T) {
	t.Parallel()

	ts := newGCPSecureSourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/repository-1:fetchBlob", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securesourcemanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securesourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecureSourceManagerRouter_OutputShapeRepository(t *testing.T) {
	t.Parallel()

	ts := newGCPSecureSourceManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/repository-1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securesourcemanager",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securesourcemanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["name"].(string); got != "projects/stackyard/locations/us-central1/repositories/repository-1" {
		t.Fatalf("expected repository name, got %#v", body["name"])
	}
	if _, ok := body["description"].(string); !ok {
		t.Fatalf("expected string description, got %#v", body["description"])
	}
	if _, ok := body["createTime"].(string); !ok {
		t.Fatalf("expected string createTime, got %#v", body["createTime"])
	}
	uris, ok := body["uris"].(map[string]any)
	if !ok {
		t.Fatalf("expected object uris, got %#v", body["uris"])
	}
	if _, ok := uris["https"].(string); !ok {
		t.Fatalf("expected string uris.https, got %#v", uris["https"])
	}
}

func TestGCPSecureSourceManagerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/securesourcemanager?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securesourcemanager contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "securesourcemanager" {
		t.Fatalf("expected service=securesourcemanager, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSecureSourceManagerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecureSourceManagerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "securesourcemanager",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securesourcemanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
