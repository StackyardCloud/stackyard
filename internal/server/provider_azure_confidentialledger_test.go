package server

import (
	"net/http"
	"testing"
)

func TestAzureConfidentialLedgerLifecycle(t *testing.T) {
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
	resourceGroup := "rg-cl"
	ledgerName := "ledger-a"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ConfidentialLedger/ledgers/" + ledgerName + "?api-version=2022-05-13"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ConfidentialLedger/operations?api-version=2022-05-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{
			"ledgerType":"Public",
			"aadBasedSecurityPrincipals":[
				{"principalId":"11111111-1111-1111-1111-111111111111","tenantId":"22222222-2222-2222-2222-222222222222","ledgerRoleName":"Administrator"}
			]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.ConfidentialLedger/ledgers" {
		t.Fatalf("unexpected ledger payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getPayload := providerContractJSONMap(t, resp)
	properties, ok := getPayload["properties"].(map[string]any)
	if !ok || properties["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected ledger properties: %#v", getPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading ledger, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ConfidentialLedger/ledgers?api-version=2022-05-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing ledgers by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	byGroup := providerContractJSONMap(t, resp)
	if values, ok := byGroup["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one ledger by resource group, got %#v", byGroup)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.ConfidentialLedger/ledgers?api-version=2022-05-13", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing ledgers by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bySubscription := providerContractJSONMap(t, resp)
	if values, ok := bySubscription["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one ledger by subscription, got %#v", bySubscription)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"certBasedSecurityPrincipals":[{"cert":"-----BEGIN CERT-----","ledgerRoleName":"Reader"}]}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	tags, ok := patched["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureConfidentialLedgerValidationAndNotFound(t *testing.T) {
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

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-cl/providers/Microsoft.ConfidentialLedger/ledgers/ledger-a?api-version=2022-05-13"

	resp := providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"location":"eastus","properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing required properties, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, path, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing ledger, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ConfidentialLedger/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
