package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecurityPostureRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)

	parent := "/gcp/v1/organizations/123456/locations/global"
	postureName := parent + "/postures/posture-1"
	deploymentName := parent + "/postureDeployments/deployment-1"
	templateName := parent + "/postureTemplates/template-1"
	operationName := parent + "/operations/op-1"

	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/locations?pageSize=1", nil, "locations")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/locations/global", nil, "global")

	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, parent+"/postures?pageSize=1", nil, "postures")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, postureName+":listRevisions?pageSize=1", nil, "revisions")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, postureName+"?revisionId=00000009", nil, "postures/posture-1")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodPost, parent+"/postures?postureId=posture-created", []byte(`{"name":"organizations/123456/locations/global/postures/posture-created","state":"ACTIVE","policySets":[{"policySetId":"baseline","policies":[{"policyId":"sha-001","constraint":{"securityHealthAnalyticsModule":{"moduleName":"BIGQUERY_TABLE_CMEK_DISABLED","moduleEnablementState":"ENABLED"}}}]}]}`), "operations/createPosture.posture-created")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodPatch, postureName+"?revisionId=0000000a&updateMask=description", []byte(`{"name":"organizations/123456/locations/global/postures/posture-1","description":"updated posture"}`), "operations/updatePosture.posture-1.0000000a")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodDelete, postureName, nil, "operations/deletePosture.posture-1")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodPost, parent+"/postures:extract", []byte(`{"parent":"organizations/123456/locations/global","postureId":"posture-extracted","workload":"project/123456789"}`), "operations/extractPosture.posture-extracted")

	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, parent+"/postureDeployments?pageSize=1", nil, "postureDeployments")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, deploymentName, nil, "postureDeployments/deployment-1")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodPost, parent+"/postureDeployments?postureDeploymentId=deployment-created", []byte(`{"name":"organizations/123456/locations/global/postureDeployments/deployment-created","targetResource":"projects/123456789","postureId":"organizations/123456/locations/global/postures/posture-1","postureRevisionId":"0000000a"}`), "operations/createPostureDeployment.deployment-created")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodPatch, deploymentName+"?updateMask=description", []byte(`{"name":"organizations/123456/locations/global/postureDeployments/deployment-1","description":"updated deployment"}`), "operations/updatePostureDeployment.deployment-1")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodDelete, deploymentName, nil, "operations/deletePostureDeployment.deployment-1")

	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, parent+"/postureTemplates?pageSize=1", nil, "postureTemplates")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, templateName+"?revisionId=00000001", nil, "postureTemplates/template-1")

	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, parent+"/operations?pageSize=1", nil, "operations")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodGet, operationName, nil, "operations/op-1")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodPost, operationName+":cancel", []byte(`{}`), "{}")
	assertGCPSecurityPostureSuccess(t, ts, http.MethodDelete, operationName, nil, "{}")
}

func TestGCPSecurityPostureRouter_ListInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/locations/global/postures?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securityposture router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPostureRouter_CreatePostureRequiresPostureID(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/organizations/123456/locations/global/postures", []byte(`{"state":"ACTIVE","policySets":[{"policySetId":"baseline","policies":[{"policyId":"sha-1"}]}]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securityposture router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPostureRouter_UpdatePostureRequiresRevisionID(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/organizations/123456/locations/global/postures/posture-1?updateMask=description", []byte(`{"name":"organizations/123456/locations/global/postures/posture-1","description":"updated"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securityposture router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPostureRouter_UpdatePostureRejectsInvalidUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/organizations/123456/locations/global/postures/posture-1?revisionId=0000000a&updateMask=badField", []byte(`{"name":"organizations/123456/locations/global/postures/posture-1","description":"updated"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securityposture router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPostureRouter_UpdatePostureRejectsEtagMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/organizations/123456/locations/global/postures/posture-1?revisionId=0000000a&updateMask=description", []byte(`{"name":"organizations/123456/locations/global/postures/posture-1","etag":"etag-mismatch","description":"updated"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp securityposture router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"Aborted"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPostureRouter_DeletePostureRejectsActiveDeployment(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPostureContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/v1/organizations/123456/locations/global/postures/posture-deployed", nil, map[string]string{
		"X-Stackyard-GCP-Service": "securityposture",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp securityposture router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPostureRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/organizations/123456/locations/global/securityposture?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securityposture contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "securityposture" {
		t.Fatalf("expected service=securityposture, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSecurityPostureContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecurityPostureSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "securityposture",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp securityposture router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
