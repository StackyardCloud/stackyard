package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPSecretManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)

	secret := "/gcp/v1/projects/stackyard/secrets/secret-1"
	version := secret + "/versions/1"

	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global", nil, "global")

	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/secrets?pageSize=1", nil, "secrets")
	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, secret, nil, "secrets/secret-1")
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/secrets?secretId=secret-1", []byte(`{"replication":{"automatic":{}},"labels":{"env":"test"}}`), "secrets/secret-1")
	assertGCPSecretManagerSuccess(t, ts, http.MethodPatch, secret+"?updateMask=labels,rotation", []byte(`{"name":"projects/stackyard/secrets/secret-1","replication":{"automatic":{}},"labels":{"team":"platform"}}`), "secrets/secret-1")
	assertGCPSecretManagerSuccess(t, ts, http.MethodDelete, secret, nil, "{}")

	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, secret+"/versions?pageSize=1", nil, "versions")
	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, version, nil, "versions/1")
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, secret+"/versions", []byte(`{"payload":{"data":"c3RhY2t5YXJkLXNlY3JldA=="}}`), "versions/4")
	assertGCPSecretManagerSuccess(t, ts, http.MethodGet, version+":access", nil, "payload")
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, version+":disable", []byte(`{}`), `"state":"DISABLED"`)
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, secret+"/versions/2:enable", []byte(`{}`), `"state":"ENABLED"`)
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, version+":destroy", []byte(`{}`), `"state":"DESTROYED"`)

	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, secret+":setIamPolicy", []byte(`{"policy":{"version":1,"bindings":[{"role":"roles/secretmanager.secretAccessor","members":["user:stackyard@example.com"]}]}}`), "bindings")
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, secret+":getIamPolicy", []byte(`{}`), "policy-etag")
	assertGCPSecretManagerSuccess(t, ts, http.MethodPost, secret+":testIamPermissions", []byte(`{"permissions":["secretmanager.secrets.get"]}`), "permissions")
}

func TestGCPSecretManagerRouter_ListSecretsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/secrets?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "secretmanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp secretmanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecretManagerRouter_CreateSecretRequiresReplication(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/secrets?secretId=secret-1", []byte(`{"labels":{"env":"test"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "secretmanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp secretmanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecretManagerRouter_UpdateSecretRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/secrets/secret-1", []byte(`{"name":"projects/stackyard/secrets/secret-1","replication":{"automatic":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "secretmanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp secretmanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecretManagerRouter_UpdateSecretNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/secrets/secret-1?updateMask=labels", []byte(`{"name":"projects/stackyard/secrets/secret-2","replication":{"automatic":{}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "secretmanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp secretmanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecretManagerRouter_AddSecretVersionRequiresPayload(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/secrets/secret-1/versions", []byte(`{"payload":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "secretmanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp secretmanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecretManagerRouter_EnableDestroyedVersionFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPSecretManagerContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/secrets/secret-1/versions/3:enable", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "secretmanager",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp secretmanager router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPSecretManagerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/secretmanager?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp secretmanager contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "secretmanager" {
		t.Fatalf("expected service=secretmanager, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPSecretManagerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPSecretManagerSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "secretmanager",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp secretmanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
