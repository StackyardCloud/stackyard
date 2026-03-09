package server

import (
	"net/http"
	"testing"
)

func TestAzureKeyVaultSecretVersionLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
	authHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	}

	resp := providerContractRequest(t, ts, http.MethodPut, "/azure/keyvault/demo-vault/secrets/api-token", []byte(`{"value":"token-v1"}`), authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 setting first secret version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	first := providerContractJSONMap(t, resp)
	if first["version"] != "v1" {
		t.Fatalf("expected version v1, got %#v", first["version"])
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/keyvault/demo-vault/secrets/api-token", []byte(`{"value":"token-v2"}`), authHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 setting second secret version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	second := providerContractJSONMap(t, resp)
	if second["version"] != "v2" {
		t.Fatalf("expected version v2, got %#v", second["version"])
	}

	getHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/keyvault/demo-vault/secrets/api-token", nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting latest secret version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	latest := providerContractJSONMap(t, resp)
	if latest["version"] != "v2" || latest["value"] != "token-v2" {
		t.Fatalf("expected latest secret version v2 token-v2, got %#v", latest)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/keyvault/demo-vault/secrets/api-token/versions", nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing secret versions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	versions := providerContractJSONMap(t, resp)
	items, ok := versions["value"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected two secret versions, got %#v", versions["value"])
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/keyvault/demo-vault/secrets/api-token/versions/v1", nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting secret version v1, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	versionOne := providerContractJSONMap(t, resp)
	if versionOne["value"] != "token-v1" {
		t.Fatalf("expected token-v1 for version v1, got %#v", versionOne["value"])
	}
}

func TestAzureKeyVaultValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
	authHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
		"Content-Type":  "application/json",
	}

	resp := providerContractRequest(t, ts, http.MethodPut, "/azure/keyvault/demo-vault/secrets/empty", []byte(`{"value":""}`), authHeaders)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty secret value, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)

	getHeaders := map[string]string{
		"Authorization": "SharedKey devstoreaccount1:signature",
	}
	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/keyvault/demo-vault/secrets/missing", nil, getHeaders)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing secret, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	_ = providerContractBody(t, resp)
}
