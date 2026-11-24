package server

import (
	"net/http"
	"testing"
)

func TestAzureContainerInstancesContainerGroupLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})
	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}
	groupPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-aci/providers/Microsoft.ContainerInstance/containerGroups/cg-a?api-version=2025-09-01"

	resp := providerContractRequest(t, ts, http.MethodGet, groupPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for a missing container group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, groupPath, []byte(`{"location":"eastus","properties":{"osType":"Linux"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 without containers, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, groupPath, []byte(`{
		"location":"eastus",
		"properties":{"osType":"Linux","containers":[{"name":"web","properties":{"image":"nginx","resources":{"requests":{"cpu":1,"memoryInGB":1.5}}}}]}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating container group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	group := providerContractJSONMap(t, resp)
	if group["name"] != "cg-a" || group["location"] != "eastus" {
		t.Fatalf("unexpected container group payload: %#v", group)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, groupPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reading container group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if got := providerContractJSONMap(t, resp); got["name"] != "cg-a" {
		t.Fatalf("unexpected container group read: %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, groupPath, nil, headers)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected success deleting container group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, groupPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}
