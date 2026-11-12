package server

import (
	"net/http"
	"testing"
)

func TestAzureSiteManagerLifecycle(t *testing.T) {
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

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-site-manager"
	base := "/azure/subscriptions/" + subscriptionID

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Edge/operations?api-version=2023-07-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	sitePath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Edge/sites/site-a?api-version=2023-07-01-preview"
	resp = providerContractRequest(t, ts, http.MethodPut, sitePath, []byte(`{
		"location":"eastus",
		"kind":"Arc",
		"tags":{"env":"dev"},
		"extendedLocation":{"type":"CustomLocation","name":"custom-eastus"},
		"identity":{"type":"SystemAssigned"},
		"properties":{
			"displayName":"Site A",
			"description":"stackyard site",
			"servicePrincipalApplicationId":"11111111-1111-1111-1111-111111111111"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	site := providerContractJSONMap(t, resp)
	if site["type"] != "Microsoft.Edge/sites" {
		t.Fatalf("unexpected site payload: %#v", site)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, sitePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, sitePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading site, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Edge/sites?api-version=2023-07-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing sites by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Edge/sites?api-version=2023-07-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing sites by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, sitePath, []byte(`{
		"tags":{"env":"prod","owner":"stackyard"},
		"properties":{"description":"updated site"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, sitePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureSiteManagerValidationAndNotFound(t *testing.T) {
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

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-site-manager/providers/Microsoft.Edge/sites/site-b?api-version=2023-07-01-preview"

	resp := providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"properties":{"displayName":"Site B"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, path, []byte(`{"tags":{"env":"stage"}}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureSiteManagerInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Edge/operations?api-version=", nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
