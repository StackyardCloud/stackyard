package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAzureApplicationGatewayLifecycleAndCapabilities(t *testing.T) {
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
	resourceGroup := "rg-net"
	gatewayName := "appgw-a"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/applicationGateways/" + gatewayName

	createPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"sku":{"name":"Standard_v2","tier":"Standard_v2","capacity":2},
		"properties":{"frontendPorts":[{"name":"port-443","properties":{"port":443}}]}
	}`)
	resp := providerContractRequest(t, ts, http.MethodPut, resource+"?api-version=2025-05-01", createPayload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["name"] != gatewayName || created["type"] != "Microsoft.Network/applicationGateways" {
		t.Fatalf("unexpected create payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource+"?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getBody := providerContractJSONMap(t, resp)
	properties, ok := getBody["properties"].(map[string]any)
	if !ok || properties["operationalState"] != "Running" {
		t.Fatalf("expected running operational state, got %#v", getBody)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Network/applicationGateways?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	var listByGroup struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(providerContractBody(t, resp), &listByGroup); err != nil {
		t.Fatalf("unmarshal list by resource group: %v", err)
	}
	if len(listByGroup.Value) != 1 {
		t.Fatalf("expected one application gateway in resource group list, got %#v", listByGroup.Value)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource+"?api-version=2025-05-01", []byte(`{"tags":{"env":"stage","owner":"stackyard"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	tags, ok := patched["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, resource+"/backendhealth?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from backendhealth, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	backendHealth := providerContractJSONMap(t, resp)
	if _, ok := backendHealth["backendAddressPools"]; !ok {
		t.Fatalf("expected backendAddressPools in backendhealth response, got %#v", backendHealth)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, resource+"/getBackendHealthOnDemand?api-version=2025-05-01", []byte(`{"properties":{"backendAddressPool":{"id":"pool-default"}}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from getBackendHealthOnDemand, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	onDemand := providerContractJSONMap(t, resp)
	if onDemand["health"] != "Healthy" {
		t.Fatalf("expected Healthy on-demand health, got %#v", onDemand)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, resource+"/stop?api-version=2025-05-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 stopping application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, resource+"?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after stop, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	stopped := providerContractJSONMap(t, resp)
	stoppedProperties, _ := stopped["properties"].(map[string]any)
	if stoppedProperties["operationalState"] != "Stopped" {
		t.Fatalf("expected stopped operational state, got %#v", stopped)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, resource+"/start?api-version=2025-05-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 starting application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Network/operations?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if value, ok := operations["value"].([]any); !ok || len(value) == 0 {
		t.Fatalf("expected non-empty operations list, got %#v", operations)
	}

	capabilityPaths := []string{
		base + "/providers/Microsoft.Network/applicationGatewayAvailableRequestHeaders?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGatewayAvailableResponseHeaders?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGatewayAvailableServerVariables?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default/predefinedPolicies?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGatewayAvailableSslOptions/default/predefinedPolicies/AppGwSslPolicy20220101?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGatewayAvailableWafRuleSets?api-version=2025-05-01",
		base + "/providers/Microsoft.Network/applicationGateways?api-version=2025-05-01",
	}
	for _, path := range capabilityPaths {
		path := path
		t.Run(path, func(t *testing.T) {
			resp := providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			body := providerContractBody(t, resp)
			if !json.Valid(body) {
				t.Fatalf("expected valid JSON for %s, got %s", path, string(body))
			}
		})
	}

	resp = providerContractRequest(t, ts, http.MethodHead, resource+"?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from HEAD get, got %d", resp.StatusCode)
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodDelete, resource+"?api-version=2025-05-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureApplicationGatewayValidationAndNotFound(t *testing.T) {
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

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-net/providers/Microsoft.Network/applicationGateways/appgw-a?api-version=2025-05-01"

	resp := providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"sku":{"name":"Standard_v2"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	invalid := providerContractJSONMap(t, resp)
	if invalid["error"] != "InvalidRequest" || !strings.Contains(invalid["message"].(string), "location") {
		t.Fatalf("unexpected validation payload: %#v", invalid)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing application gateway, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	notFound := providerContractJSONMap(t, resp)
	if notFound["error"] != "NotFound" {
		t.Fatalf("expected NotFound payload, got %#v", notFound)
	}
}

func TestAzureApplicationGatewayInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Network/operations?api-version=", nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" || payload["provider"] != providerAzure {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}
