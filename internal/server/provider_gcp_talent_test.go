package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTalentRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)

	project := "stackyard"
	tenantName := "projects/stackyard/tenants/tenant-1"
	companyName := tenantName + "/companies/company-1"
	jobName := tenantName + "/jobs/job-1"

	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants?pageSize=1", nil, "tenants")
	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1", nil, "tenant-1")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants", []byte(`{"externalId":"tenant-ext-new"}`), "tenant-ext-new")
	assertGCPTalentSuccess(t, ts, http.MethodPatch, "/gcp/v4/projects/"+project+"/tenants/tenant-1?updateMask=externalId", []byte(`{"name":"projects/stackyard/tenants/tenant-1","externalId":"tenant-ext-updated"}`), "tenant-ext-updated")
	assertGCPTalentSuccess(t, ts, http.MethodDelete, "/gcp/v4/projects/"+project+"/tenants/tenant-1", nil, "{}")

	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1/companies?pageSize=1&requireOpenJobs=true", nil, "companies")
	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1/companies/company-1", nil, "company-1")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/companies", []byte(`{"displayName":"Stackyard Inc","externalId":"company-ext-new"}`), "company-ext-new")
	assertGCPTalentSuccess(t, ts, http.MethodPatch, "/gcp/v4/projects/"+project+"/tenants/tenant-1/companies/company-1?updateMask=displayName,externalId", []byte(`{"name":"projects/stackyard/tenants/tenant-1/companies/company-1","displayName":"Updated Stackyard Inc","externalId":"company-ext-updated"}`), "Updated Stackyard Inc")
	assertGCPTalentSuccess(t, ts, http.MethodDelete, "/gcp/v4/projects/"+project+"/tenants/tenant-1/companies/company-1", nil, "{}")

	filter := "companyName%20=%20%22projects/stackyard/tenants/tenant-1/companies/company-1%22"
	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs?filter="+filter+"&pageSize=1", nil, "jobs")
	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs/job-1", nil, "job-1")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs", []byte(`{"company":"projects/stackyard/tenants/tenant-1/companies/company-1","requisitionId":"req-created-1","title":"Platform Engineer","description":"Build platforms"}`), "req-created-1")
	assertGCPTalentSuccess(t, ts, http.MethodPatch, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs/job-1?updateMask=company,requisitionId,title,description", []byte(`{"name":"projects/stackyard/tenants/tenant-1/jobs/job-1","company":"projects/stackyard/tenants/tenant-1/companies/company-1","requisitionId":"req-updated-1","title":"Updated Platform Engineer","description":"Build and operate platforms"}`), "Updated Platform Engineer")
	assertGCPTalentSuccess(t, ts, http.MethodDelete, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs/job-1", nil, "{}")

	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs:batchCreate", []byte(`{"parent":"projects/stackyard/tenants/tenant-1","jobs":[{"company":"projects/stackyard/tenants/tenant-1/companies/company-1","requisitionId":"req-batch-create-1","title":"Batch Create Job","description":"Batch create description"}]}`), "batchCreateJobs-1")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs:batchUpdate", []byte(`{"parent":"projects/stackyard/tenants/tenant-1","jobs":[{"name":"projects/stackyard/tenants/tenant-1/jobs/job-1","company":"projects/stackyard/tenants/tenant-1/companies/company-1","requisitionId":"req-batch-update-1","title":"Batch Update Job","description":"Batch update description"}]}`), "batchUpdateJobs-1")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs:batchDelete", []byte(`{"parent":"projects/stackyard/tenants/tenant-1","names":["projects/stackyard/tenants/tenant-1/jobs/job-1"]}`), "batchDeleteJobs-1")

	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs:search", []byte(`{"parent":"projects/stackyard/tenants/tenant-1","requestMetadata":{"domain":"example.com","sessionId":"session-1","userId":"user-1"},"jobQuery":{"query":"Engineer"},"maxPageSize":1}`), "matchingJobs")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/jobs:searchForAlert", []byte(`{"parent":"projects/stackyard/tenants/tenant-1","requestMetadata":{"domain":"example.com","sessionId":"session-1","userId":"user-1"},"jobQuery":{"query":"Engineer"},"maxPageSize":1}`), "matchingJobs")

	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1:completeQuery?query=stack&pageSize=2&company="+companyName, nil, "completionResults")
	assertGCPTalentSuccess(t, ts, http.MethodPost, "/gcp/v4/projects/"+project+"/tenants/tenant-1/clientEvents", []byte(`{"requestId":"req-1","eventId":"event-1","createTime":"2026-01-01T00:00:00Z","jobEvent":{"type":"VIEW","jobs":["projects/stackyard/tenants/tenant-1/jobs/job-1"]}}`), "event-1")

	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1/operations?pageSize=1", nil, "operations")
	assertGCPTalentSuccess(t, ts, http.MethodGet, "/gcp/v4/projects/"+project+"/tenants/tenant-1/operations/batchCreateJobs-1", nil, "batchCreateJobs-1")

	_ = companyName
	_ = jobName
}

func TestGCPTalentRouter_ListTenantsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v4/projects/stackyard/tenants?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_CreateTenantRequiresExternalID(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v4/projects/stackyard/tenants", []byte(`{"name":"projects/stackyard/tenants/tenant-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_CreateCompanyRequiresDisplayNameAndExternalID(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v4/projects/stackyard/tenants/tenant-1/companies", []byte(`{"displayName":"Stackyard"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_CreateJobRequiresRequiredFields(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v4/projects/stackyard/tenants/tenant-1/jobs", []byte(`{"company":"projects/stackyard/tenants/tenant-1/companies/company-1"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_SearchJobsRequiresRequestMetadata(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v4/projects/stackyard/tenants/tenant-1/jobs:search", []byte(`{"parent":"projects/stackyard/tenants/tenant-1","jobQuery":{"query":"Engineer"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_CompleteQueryPageSizeOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v4/projects/stackyard/tenants/tenant-1:completeQuery?query=stack&pageSize=11", nil, map[string]string{
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"OutOfRange"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_CreateClientEventRequiresJobEvent(t *testing.T) {
	t.Parallel()

	ts := newGCPTalentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v4/projects/stackyard/tenants/tenant-1/clientEvents", []byte(`{"eventId":"event-1","createTime":"2026-01-01T00:00:00Z"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "talent",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp talent router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTalentRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v4/projects/stackyard/tenants/tenant-1/talent?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp talent contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "talent" {
		t.Fatalf("expected service=talent, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPTalentContractServer(t *testing.T) *httptest.Server {
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

func assertGCPTalentSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "talent",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp talent router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
