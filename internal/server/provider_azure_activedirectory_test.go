package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAzureActiveDirectoryLifecycle(t *testing.T) {
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
	resourceGroup := "rg-b2c"
	base := "/azure/subscriptions/" + subscriptionID

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AzureActiveDirectory/operations?api-version=2021-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.AzureActiveDirectory/checkNameAvailability?api-version=2021-04-01", []byte(`{"name":"tenant-a.onmicrosoft.com","countryCode":"US"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	availability := providerContractJSONMap(t, resp)
	if availability["nameAvailable"] != true {
		t.Fatalf("expected nameAvailable=true, got %#v", availability)
	}

	b2cPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureActiveDirectory/b2cDirectories/tenant-a.onmicrosoft.com?api-version=2021-04-01"
	resp = providerContractRequest(t, ts, http.MethodPut, b2cPath, []byte(`{
		"location":"United States",
		"sku":{"name":"Standard","tier":"A0"},
		"tags":{"env":"dev"},
		"properties":{
			"createTenantProperties":{
				"countryCode":"US",
				"displayName":"Tenant A"
			}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	b2cTenant := providerContractJSONMap(t, resp)
	if b2cTenant["type"] != "Microsoft.AzureActiveDirectory/b2cDirectories" {
		t.Fatalf("unexpected b2c tenant payload: %#v", b2cTenant)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, b2cPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, b2cPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading b2c tenant, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AzureActiveDirectory/b2cDirectories?api-version=2021-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing b2c tenants by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.AzureActiveDirectory/b2cDirectories?api-version=2021-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing b2c tenants by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, b2cPath, []byte(`{
		"sku":{"name":"PremiumP1","tier":"A1"},
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{
			"billingConfig":{"billingType":"MAU"},
			"tenantId":"11111111-1111-1111-1111-111111111111"
		}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.AzureActiveDirectory/checkNameAvailability?api-version=2021-04-01", []byte(`{"name":"tenant-a.onmicrosoft.com","countryCode":"US"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking duplicate name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	availability = providerContractJSONMap(t, resp)
	if availability["nameAvailable"] != false {
		t.Fatalf("expected duplicate nameAvailable=false, got %#v", availability)
	}

	guestPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureActiveDirectory/guestUsages/guest-usage-a?api-version=2021-04-01"
	resp = providerContractRequest(t, ts, http.MethodPut, guestPath, []byte(`{
		"location":"global",
		"tags":{"env":"dev"},
		"properties":{"tenantId":"11111111-1111-1111-1111-111111111111"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	guestUsage := providerContractJSONMap(t, resp)
	if guestUsage["type"] != "Microsoft.AzureActiveDirectory/guestUsages" {
		t.Fatalf("unexpected guest usage payload: %#v", guestUsage)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, guestPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, guestPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading guest usage, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AzureActiveDirectory/guestUsages?api-version=2021-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing guest usages by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.AzureActiveDirectory/guestUsages?api-version=2021-04-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing guest usages by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, guestPath, []byte(`{"tags":{"env":"prod","owner":"stackyard"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, guestPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, b2cPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureActiveDirectoryValidationAndNotFound(t *testing.T) {
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

	base := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-b2c/providers/Microsoft.AzureActiveDirectory"
	b2cPath := base + "/b2cDirectories/tenant-b.onmicrosoft.com?api-version=2021-04-01"
	guestPath := base + "/guestUsages/guest-usage-b?api-version=2021-04-01"

	resp := providerContractRequest(t, ts, http.MethodPost, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AzureActiveDirectory/checkNameAvailability?api-version=2021-04-01", []byte(`{"countryCode":"US"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AzureActiveDirectory/checkNameAvailability?api-version=2021-04-01", []byte(`{"name":"bad tenant","countryCode":"US"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for invalid-name availability check, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["nameAvailable"] != false {
		t.Fatalf("expected invalid nameAvailable=false, got %#v", payload)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, b2cPath, []byte(`{"sku":{"name":"Standard"},"properties":{"createTenantProperties":{"countryCode":"US","displayName":"Tenant B"}}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location on b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, guestPath, []byte(`{"location":"global","properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing tenantId on guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, b2cPath, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid json on b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, b2cPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, guestPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, b2cPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing b2c tenant, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, guestPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing guest usage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureActiveDirectoryInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AzureActiveDirectory/operations?api-version=", nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureActiveDirectoryListPayloadsAreJSON(t *testing.T) {
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
	}

	paths := []string{
		"/azure/providers/Microsoft.AzureActiveDirectory/operations?api-version=2021-04-01",
		"/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AzureActiveDirectory/b2cDirectories?api-version=2021-04-01",
		"/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AzureActiveDirectory/guestUsages?api-version=2021-04-01",
	}

	for _, path := range paths {
		resp := providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		var payload map[string]any
		if err := json.Unmarshal(providerContractBody(t, resp), &payload); err != nil {
			t.Fatalf("unmarshal response for %s: %v", path, err)
		}
	}
}
