package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecurityPrivateCARouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	caPools := base + "/caPools"
	caPoolName := caPools + "/pool-1"
	certificateAuthorities := caPoolName + "/certificateAuthorities"
	certificateAuthorityName := certificateAuthorities + "/ca-1"
	certificates := caPoolName + "/certificates"
	certificateName := certificates + "/cert-1"
	revocationLists := certificateAuthorityName + "/certificateRevocationLists"
	revocationListName := revocationLists + "/crl-1"
	certificateTemplates := base + "/certificateTemplates"
	certificateTemplateName := certificateTemplates + "/template-1"
	operationName := base + "/operations/operation-1"

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, caPools+"?pageSize=1", nil, "caPools")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, caPoolName, nil, "caPools/pool-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, caPools+"?caPoolId=pool-1", []byte(`{"caPool":{"tier":"ENTERPRISE","publishingOptions":{"publishCaCert":true}}}`), "operations/create-ca-pool-pool-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPatch, caPoolName+"?updateMask=labels", []byte(`{"caPool":{"name":"projects/stackyard/locations/us-central1/caPools/pool-1","labels":{"team":"platform"}}}`), "operations/update-ca-pool-pool-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodDelete, caPoolName, nil, "operations/delete-ca-pool-pool-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, caPoolName+":fetchCaCerts", nil, "caCerts")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, certificateAuthorities+"?pageSize=1", nil, "certificateAuthorities")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, certificateAuthorityName, nil, "certificateAuthorities/ca-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateAuthorities+"?certificateAuthorityId=ca-1", []byte(`{"certificateAuthority":{"type":"SELF_SIGNED","config":{"subjectConfig":{"subject":{"commonName":"Stackyard CA"}}}}}`), "operations/create-certificate-authority-ca-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPatch, certificateAuthorityName+"?updateMask=labels", []byte(`{"certificateAuthority":{"name":"projects/stackyard/locations/us-central1/caPools/pool-1/certificateAuthorities/ca-1","labels":{"team":"security"}}}`), "operations/update-certificate-authority-ca-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateAuthorityName+":disable", []byte(`{}`), "operations/disable-certificate-authority-ca-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateAuthorities+"/ca-disabled:enable", []byte(`{}`), "operations/enable-certificate-authority-ca-disabled")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateAuthorities+"/ca-awaiting:activate", []byte(`{}`), "operations/activate-certificate-authority-ca-awaiting")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateAuthorities+"/ca-awaiting:fetch", []byte(`{}`), "pemCsr")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateAuthorities+"/ca-deleted:undelete", []byte(`{}`), "operations/undelete-certificate-authority-ca-deleted")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodDelete, certificateAuthorityName, nil, "operations/delete-certificate-authority-ca-1")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, certificates+"?pageSize=1", nil, "certificates")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, certificateName, nil, "certificates/cert-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificates+"?certificateId=cert-1", []byte(`{"certificate":{"lifetime":"86400s","pemCsr":"-----BEGIN CERTIFICATE REQUEST-----\nSTACKYARD\n-----END CERTIFICATE REQUEST-----"}}`), "certificates/cert-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPatch, certificateName+"?updateMask=labels", []byte(`{"certificate":{"name":"projects/stackyard/locations/us-central1/caPools/pool-1/certificates/cert-1","labels":{"env":"staged"}}}`), "certificates/cert-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateName+":revoke", []byte(`{"reason":"KEY_COMPROMISE"}`), "revocationDetails")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, revocationLists+"?pageSize=1", nil, "certificateRevocationLists")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, revocationListName, nil, "certificateRevocationLists/crl-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPatch, revocationListName+"?updateMask=labels", []byte(`{"certificateRevocationList":{"name":"projects/stackyard/locations/us-central1/caPools/pool-1/certificateAuthorities/ca-1/certificateRevocationLists/crl-1","labels":{"env":"prod"}}}`), "operations/update-certificate-revocation-list-crl-1")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, certificateTemplates+"?pageSize=1", nil, "certificateTemplates")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, certificateTemplateName, nil, "certificateTemplates/template-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, certificateTemplates+"?certificateTemplateId=template-1", []byte(`{"certificateTemplate":{"description":"Stackyard template"}}`), "operations/create-certificate-template-template-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPatch, certificateTemplateName+"?updateMask=description", []byte(`{"certificateTemplate":{"name":"projects/stackyard/locations/us-central1/certificateTemplates/template-1","description":"updated"}}`), "operations/update-certificate-template-template-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodDelete, certificateTemplateName, nil, "operations/delete-certificate-template-template-1")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, caPoolName+":getIamPolicy", []byte(`{}`), "bindings")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, caPoolName+":setIamPolicy", []byte(`{"policy":{"version":1,"bindings":[{"role":"roles/privateca.admin","members":["user:stackyard@example.com"]}]}}`), "privateca.admin")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, caPoolName+":testIamPermissions", []byte(`{"permissions":["privateca.caPools.get"]}`), "permissions")

	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodGet, operationName, nil, "operations/operation-1")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodPost, operationName+":cancel", []byte(`{}`), "{}")
	assertGCPSecurityPrivateCASuccess(t, ts, http.MethodDelete, operationName, nil, "{}")
}

func TestGCPSecurityPrivateCARouter_ListCaPoolsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/caPools?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_CreateCaPoolRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/caPools", []byte(`{"caPool":{"tier":"ENTERPRISE"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_UpdateCaPoolRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1", []byte(`{"caPool":{"name":"projects/stackyard/locations/us-central1/caPools/pool-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_UpdateCaPoolNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1?updateMask=labels", []byte(`{"caPool":{"name":"projects/stackyard/locations/us-central1/caPools/pool-2"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_ActivateRequiresAwaitingState(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1/certificateAuthorities/ca-1:activate", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_RevokeAlreadyRevokedFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1/certificates/cert-revoked:revoke", []byte(`{"reason":"KEY_COMPROMISE"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_SetIAMPolicyRequiresPolicy(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1:setIamPolicy", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_TestIAMPermissionsRequiresPermissions(t *testing.T) {
	t.Parallel()

	ts := newGCPSecurityPrivateCAContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1:testIamPermissions", []byte(`{"permissions":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "security-privateca",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp security_privateca router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecurityPrivateCARouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/security_privateca?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp security_privateca contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "security_privateca" {
		t.Fatalf("expected service=security_privateca, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPSecurityPrivateCARouter_DisableActionDirect(t *testing.T) {
	t.Parallel()

	path := "/gcp/v1/projects/stackyard/locations/us-central1/caPools/pool-1/certificateAuthorities/ca-1:disable"
	if !isGCPSecurityPrivateCAPath(path) {
		t.Fatalf("expected privateca path recognizer to match: %s", path)
	}
	_, _, caPoolID, caID, action, ok := parseGCPSecurityPrivateCACertificateAuthorityActionPath(path)
	if !ok || caPoolID != "pool-1" || caID != "ca-1" || action != "disable" {
		t.Fatalf("unexpected ca action parse result: ok=%v caPoolID=%q caID=%q action=%q", ok, caPoolID, caID, action)
	}

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Stackyard-GCP-Service", "security-privateca")
	recorder := httptest.NewRecorder()
	server := &Server{}
	if !server.handleGCPSecurityPrivateCARouter(recorder, req) {
		t.Fatalf("expected router to handle disable action path")
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from disable action direct path, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func newGCPSecurityPrivateCAContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecurityPrivateCASuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "security-privateca",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp security_privateca router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
