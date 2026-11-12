package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAzureApplicationGatewayContainersLifecycle(t *testing.T) {
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
	resourceGroup := "rg-servicenetworking"
	trafficControllerName := "tc-a"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ServiceNetworking/trafficControllers/" + trafficControllerName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ServiceNetworking/operations?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"?api-version=2025-01-01", []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"securityPolicyConfigurations":{"wafSecurityPolicy":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.ServiceNetworking/trafficControllers/tc-a/securityPolicies/sp-a"}}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	tc := providerContractJSONMap(t, resp)
	if tc["name"] != trafficControllerName || tc["type"] != "Microsoft.ServiceNetworking/trafficControllers" {
		t.Fatalf("unexpected traffic controller payload: %#v", tc)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.ServiceNetworking/trafficControllers?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing traffic controllers by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	var listByGroup struct {
		Value []map[string]any `json:"value"`
	}
	if err := json.Unmarshal(providerContractBody(t, resp), &listByGroup); err != nil {
		t.Fatalf("unmarshal list by group: %v", err)
	}
	if len(listByGroup.Value) != 1 {
		t.Fatalf("expected one traffic controller, got %#v", listByGroup.Value)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/providers/Microsoft.ServiceNetworking/trafficControllers?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing traffic controllers by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"?api-version=2025-01-01", []byte(`{"tags":{"env":"stage","owner":"stackyard"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patchedTC := providerContractJSONMap(t, resp)
	if tags, ok := patchedTC["tags"].(map[string]any); !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched traffic controller tags, got %#v", patchedTC)
	}

	frontendPath := base + "/frontends/fe-a"
	resp = providerContractRequest(t, ts, http.MethodPut, frontendPath+"?api-version=2025-01-01", []byte(`{"location":"eastus","tags":{"tier":"public"},"properties":{}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating frontend, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	frontend := providerContractJSONMap(t, resp)
	if frontend["type"] != "Microsoft.ServiceNetworking/trafficControllers/frontends" {
		t.Fatalf("unexpected frontend payload: %#v", frontend)
	}
	if properties, _ := frontend["properties"].(map[string]any); properties["fqdn"] == "" {
		t.Fatalf("expected frontend fqdn, got %#v", frontend)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, frontendPath+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting frontend, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/frontends?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing frontends, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, frontendPath+"?api-version=2025-01-01", []byte(`{"tags":{"tier":"edge"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching frontend, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	associationPath := base + "/associations/as-a"
	resp = providerContractRequest(t, ts, http.MethodPut, associationPath+"?api-version=2025-01-01", []byte(`{
		"location":"eastus",
		"tags":{"scope":"cluster"},
		"properties":{"associationType":"subnets","subnet":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating association, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, associationPath+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting association, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/associations?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing associations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, associationPath+"?api-version=2025-01-01", []byte(`{"properties":{"subnet":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-b"}}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching association, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	securityPolicyPath := base + "/securityPolicies/sp-a"
	resp = providerContractRequest(t, ts, http.MethodPut, securityPolicyPath+"?api-version=2025-01-01", []byte(`{
		"location":"eastus",
		"tags":{"tier":"waf"},
		"properties":{"wafPolicy":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/waf-a"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating security policy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	securityPolicy := providerContractJSONMap(t, resp)
	if properties, _ := securityPolicy["properties"].(map[string]any); properties["policyType"] != "waf" {
		t.Fatalf("expected waf policy type, got %#v", securityPolicy)
	}
	resp = providerContractRequest(t, ts, http.MethodGet, securityPolicyPath+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting security policy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/securityPolicies?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing security policies, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, securityPolicyPath+"?api-version=2025-01-01", []byte(`{"tags":{"tier":"waf-updated"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching security policy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 re-getting traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	reloadedTC := providerContractJSONMap(t, resp)
	properties, _ := reloadedTC["properties"].(map[string]any)
	if frontends, _ := properties["frontends"].([]any); len(frontends) != 1 {
		t.Fatalf("expected one frontend reference, got %#v", reloadedTC)
	}
	if associations, _ := properties["associations"].([]any); len(associations) != 1 {
		t.Fatalf("expected one association reference, got %#v", reloadedTC)
	}
	if securityPolicies, _ := properties["securityPolicies"].([]any); len(securityPolicies) != 1 {
		t.Fatalf("expected one security policy reference, got %#v", reloadedTC)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, frontendPath+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting frontend, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodDelete, associationPath+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting association, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodDelete, securityPolicyPath+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting security policy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodDelete, base+"?api-version=2025-01-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureApplicationGatewayContainersValidationAndNotFound(t *testing.T) {
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

	parentPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.ServiceNetworking/trafficControllers/tc-a?api-version=2025-01-01"
	resp := providerContractRequest(t, ts, http.MethodPut, parentPath, []byte(`{"tags":{"env":"dev"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing traffic controller location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, parentPath, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	associationPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.ServiceNetworking/trafficControllers/tc-a/associations/as-a?api-version=2025-01-01"
	resp = providerContractRequest(t, ts, http.MethodPut, associationPath, []byte(`{"location":"eastus","properties":{"subnet":{"id":"subnet-a"}}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing association type, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	securityPolicyPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.ServiceNetworking/trafficControllers/tc-a/securityPolicies/sp-a?api-version=2025-01-01"
	resp = providerContractRequest(t, ts, http.MethodPut, securityPolicyPath, []byte(`{"location":"eastus","properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing waf policy id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, parentPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "NotFound" {
		t.Fatalf("expected NotFound payload, got %#v", payload)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, parentPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing traffic controller, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureApplicationGatewayContainersInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ServiceNetworking/operations?api-version=", nil, map[string]string{
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

func TestAzureApplicationGatewayContainersUpdateRequiresParent(t *testing.T) {
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

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-servicenetworking/providers/Microsoft.ServiceNetworking/trafficControllers/tc-a/frontends/fe-a?api-version=2025-01-01"
	resp := providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"location":"eastus","properties":{}}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when creating child under missing parent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "NotFound" || !strings.Contains(payload["message"].(string), "traffic controller") {
		t.Fatalf("unexpected missing parent payload: %#v", payload)
	}
}
