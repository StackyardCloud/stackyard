package server

import (
	"net/http"
	"testing"
)

func TestAzureQuantumLifecycle(t *testing.T) {
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
	resourceGroup := "rg-quantum"
	workspaceName := "workspace-a"
	location := "eastus"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/" + workspaceName + "?api-version=" + azureQuantumAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Quantum/operations?api-version="+azureQuantumAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	values, ok := operations["value"].([]any)
	if !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.Quantum/checkNameAvailability?api-version="+azureQuantumAPIVersion, []byte(`{
		"name":"workspace-a",
		"type":"Microsoft.Quantum/workspaces"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	nameAvailability := providerContractJSONMap(t, resp)
	if nameAvailability["nameAvailable"] != true {
		t.Fatalf("expected name to be available, got %#v", nameAvailability)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"identity":{"type":"SystemAssigned"},
		"properties":{
			"workspaceDescription":"Local workspace",
			"apiKeyEnabled":true
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.Quantum/workspaces" {
		t.Fatalf("unexpected workspace payload: %#v", created)
	}
	createdProps, ok := created["properties"].(map[string]any)
	if !ok || createdProps["endpointUri"] == "" || createdProps["workspaceId"] == "" {
		t.Fatalf("expected endpointUri and workspaceId, got %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.Quantum/checkNameAvailability?api-version="+azureQuantumAPIVersion, []byte(`{
		"name":"workspace-a",
		"type":"Microsoft.Quantum/workspaces"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking duplicate name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	nameTaken := providerContractJSONMap(t, resp)
	if nameTaken["nameAvailable"] != false {
		t.Fatalf("expected name collision payload, got %#v", nameTaken)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	got := providerContractJSONMap(t, resp)
	if got["name"] != workspaceName || got["location"] != location {
		t.Fatalf("unexpected get payload: %#v", got)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading workspace, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Quantum/workspaces?api-version="+azureQuantumAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	byGroup := providerContractJSONMap(t, resp)
	if values, ok := byGroup["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one workspace in resource group, got %#v", byGroup)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Quantum/workspaces?api-version="+azureQuantumAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bySubscription := providerContractJSONMap(t, resp)
	if values, ok := bySubscription["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one workspace in subscription, got %#v", bySubscription)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Quantum/locations/"+location+"/offerings?api-version="+azureQuantumAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing offerings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	offerings := providerContractJSONMap(t, resp)
	if values, ok := offerings["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected offering values, got %#v", offerings)
	}

	keysPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/" + workspaceName + "/listKeys?api-version=" + azureQuantumAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPost, keysPath, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing keys, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	keys := providerContractJSONMap(t, resp)
	primaryBefore, _ := keys["primaryKey"].(string)
	secondaryBefore, _ := keys["secondaryKey"].(string)
	if primaryBefore == "" || secondaryBefore == "" {
		t.Fatalf("expected primary and secondary keys, got %#v", keys)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Quantum/workspaces/"+workspaceName+"/regenerateKeys?api-version="+azureQuantumAPIVersion, []byte(`{"keyName":"Primary"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 regenerating primary key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	regen := providerContractJSONMap(t, resp)
	primaryAfter, _ := regen["primaryKey"].(string)
	secondaryAfter, _ := regen["secondaryKey"].(string)
	if primaryAfter == "" || primaryAfter == primaryBefore {
		t.Fatalf("expected primary key to change, got %#v", regen)
	}
	if secondaryAfter != secondaryBefore {
		t.Fatalf("expected secondary key to stay stable, got %#v", regen)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"workspaceDescription":"Updated workspace"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	tags, ok := patched["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}
	patchedProps, ok := patched["properties"].(map[string]any)
	if !ok || patchedProps["workspaceDescription"] != "Updated workspace" {
		t.Fatalf("expected patched properties, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting already missing workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureQuantumValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-quantum"
	base := "/azure/subscriptions/" + subscriptionID

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid workspace name",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/bad_name?api-version=" + azureQuantumAPIVersion,
			body:       []byte(`{"location":"eastus"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/workspace-a?api-version=" + azureQuantumAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid name availability type",
			method:     http.MethodPost,
			path:       base + "/providers/Microsoft.Quantum/checkNameAvailability?api-version=" + azureQuantumAPIVersion,
			body:       []byte(`{"name":"workspace-a","type":"Microsoft.Quantum/providers"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing workspace get",
			method:     http.MethodGet,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/missing?api-version=" + azureQuantumAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing key route",
			method:     http.MethodPost,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/missing/listKeys?api-version=" + azureQuantumAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid regenerate key payload",
			method:     http.MethodPost,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/workspace-a/regenerateKeys?api-version=" + azureQuantumAPIVersion,
			body:       []byte(`{"keyName":"Tertiary"}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.Quantum/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}

	createPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Quantum/workspaces/workspace-a?api-version=" + azureQuantumAPIVersion
	resp := providerContractRequest(t, ts, http.MethodPut, createPath, []byte(`{"location":"eastus"}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Quantum/workspaces/workspace-a/regenerateKeys?api-version="+azureQuantumAPIVersion, []byte(`{"keyName":"Tertiary"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid keyName, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
