package server

import (
	"net/http"
	"testing"
)

func TestAzurePlaywrightLifecycle(t *testing.T) {
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
	resourceGroup := "rg-playwright"
	workspaceName := "workspace-a"
	location := "eastus"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces/" + workspaceName + "?api-version=" + azurePlaywrightAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.LoadTestService/operations?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	values, ok := operations["value"].([]any)
	if !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.LoadTestService/checkNameAvailability?api-version="+azurePlaywrightAPIVersion, []byte(`{
		"name":"workspace-a",
		"type":"Microsoft.LoadTestService/playwrightWorkspaces"
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
		"properties":{
			"localAuth":"Enabled",
			"regionalAffinity":"Enabled"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating playwright workspace, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.LoadTestService/playwrightWorkspaces" {
		t.Fatalf("unexpected workspace payload: %#v", created)
	}
	createdProps, ok := created["properties"].(map[string]any)
	if !ok || createdProps["dataplaneUri"] == "" || createdProps["workspaceId"] == "" {
		t.Fatalf("expected dataplaneUri and workspaceId, got %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.LoadTestService/checkNameAvailability?api-version="+azurePlaywrightAPIVersion, []byte(`{
		"name":"workspace-a",
		"type":"Microsoft.LoadTestService/playwrightWorkspaces"
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

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/playwrightWorkspaces?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	byGroup := providerContractJSONMap(t, resp)
	if values, ok := byGroup["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one workspace in resource group, got %#v", byGroup)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.LoadTestService/playwrightWorkspaces?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bySubscription := providerContractJSONMap(t, resp)
	if values, ok := bySubscription["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one workspace in subscription, got %#v", bySubscription)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"localAuth":"Disabled"}
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
	if !ok || patchedProps["localAuth"] != "Disabled" {
		t.Fatalf("expected patched properties, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.LoadTestService/locations/"+location+"/playwrightQuotas?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing subscription quotas, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	subscriptionQuotas := providerContractJSONMap(t, resp)
	if values, ok := subscriptionQuotas["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected quota values, got %#v", subscriptionQuotas)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.LoadTestService/locations/"+location+"/playwrightQuotas/ExecutionMinutes?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting named subscription quota, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/playwrightWorkspaces/"+workspaceName+"/quotas?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing workspace quotas, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	workspaceQuotas := providerContractJSONMap(t, resp)
	if values, ok := workspaceQuotas["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected workspace quota values, got %#v", workspaceQuotas)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/playwrightWorkspaces/"+workspaceName+"/quotas/ExecutionMinutes?api-version="+azurePlaywrightAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting named workspace quota, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
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

func TestAzurePlaywrightValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-playwright"
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
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces/bad_name?api-version=" + azurePlaywrightAPIVersion,
			body:       []byte(`{"location":"eastus"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces/workspace-a?api-version=" + azurePlaywrightAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid name availability type",
			method:     http.MethodPost,
			path:       base + "/providers/Microsoft.LoadTestService/checkNameAvailability?api-version=" + azurePlaywrightAPIVersion,
			body:       []byte(`{"name":"workspace-a","type":"Microsoft.LoadTestService/loadTests"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "get missing workspace",
			method:     http.MethodGet,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces/missing?api-version=" + azurePlaywrightAPIVersion,
			body:       nil,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get missing workspace quota parent",
			method:     http.MethodGet,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/playwrightWorkspaces/missing/quotas?api-version=" + azurePlaywrightAPIVersion,
			body:       nil,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "get missing subscription quota",
			method:     http.MethodGet,
			path:       base + "/providers/Microsoft.LoadTestService/locations/eastus/playwrightQuotas/MissingQuota?api-version=" + azurePlaywrightAPIVersion,
			body:       nil,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			body := providerContractBody(t, resp)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(body))
			}
		})
	}
}
