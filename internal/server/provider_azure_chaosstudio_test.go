package server

import (
	"net/http"
	"testing"
)

func TestAzureChaosStudioLifecycle(t *testing.T) {
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
	resourceGroup := "rg-chaos"
	location := "eastus"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	locationBase := subscriptionBase + "/providers/Microsoft.Chaos/locations/" + location
	parentBase := subscriptionBase + "/resourceGroups/rg-vm/providers/Microsoft.Compute/virtualMachines/vm-chaos"
	targetBase := parentBase + "/providers/Microsoft.Chaos/targets/microsoft-agent"
	capabilityBase := targetBase + "/capabilities/Shutdown-1.0"
	experimentBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Chaos/experiments/exp-a"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Chaos/operations?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) < 10 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, locationBase+"/targetTypes?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing target types, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	targetTypes := providerContractJSONMap(t, resp)
	targetTypeValues, ok := targetTypes["value"].([]any)
	if !ok || len(targetTypeValues) == 0 {
		t.Fatalf("expected target types, got %#v", targetTypes)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, locationBase+"/targetTypes/Microsoft-VirtualMachine?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting target type, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	targetType := providerContractJSONMap(t, resp)
	if targetType["type"] != "Microsoft.Chaos/locations/targetTypes" {
		t.Fatalf("unexpected target type payload: %#v", targetType)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, locationBase+"/targetTypes/Microsoft-VirtualMachine/capabilityTypes?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing capability types, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	capabilityTypes := providerContractJSONMap(t, resp)
	if values, ok := capabilityTypes["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected capability types, got %#v", capabilityTypes)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, locationBase+"/targetTypes/Microsoft-VirtualMachine/capabilityTypes/Shutdown-1.0?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting capability type, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	capabilityType := providerContractJSONMap(t, resp)
	if capabilityType["type"] != "Microsoft.Chaos/locations/targetTypes/capabilityTypes" {
		t.Fatalf("unexpected capability type payload: %#v", capabilityType)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, targetBase+"?api-version="+azureChaosStudioAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"targetType":"Microsoft-VirtualMachine","displayName":"VM Chaos Agent"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating target, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	target := providerContractJSONMap(t, resp)
	if target["type"] != "Microsoft.Chaos/targets" {
		t.Fatalf("unexpected target payload: %#v", target)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, parentBase+"/providers/Microsoft.Chaos/targets?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing targets, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, targetBase+"?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading target, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, capabilityBase+"?api-version="+azureChaosStudioAPIVersion, []byte(`{
		"location":"eastus",
		"properties":{"publisher":"stackyard","version":"1.0"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating capability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	capability := providerContractJSONMap(t, resp)
	if capability["type"] != "Microsoft.Chaos/targets/capabilities" {
		t.Fatalf("unexpected capability payload: %#v", capability)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, targetBase+"/capabilities?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing capabilities, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, experimentBase+"?api-version="+azureChaosStudioAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{
			"selectors":[{"id":"Selector1","type":"List","targets":[{"type":"ChaosTarget","id":"`+targetBase+`"}]}],
			"steps":[{"name":"Step1","branches":[{"name":"Branch1","actions":[{"name":"Action1","type":"continuous","selectorId":"Selector1","duration":"PT1M"}]}]}]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating experiment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	experiment := providerContractJSONMap(t, resp)
	if experiment["type"] != "Microsoft.Chaos/experiments" {
		t.Fatalf("unexpected experiment payload: %#v", experiment)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.Chaos/experiments?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing experiments by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Chaos/experiments?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing experiments by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, experimentBase+"?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading experiment, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, experimentBase+"?api-version="+azureChaosStudioAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"identity":{"type":"None"}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching experiment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, experimentBase+"/start?api-version="+azureChaosStudioAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 starting experiment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	startPayload := providerContractJSONMap(t, resp)
	operationID, _ := startPayload["operationId"].(string)
	executionID, _ := startPayload["executionId"].(string)
	if operationID == "" || executionID == "" {
		t.Fatalf("expected operationId and executionId, got %#v", startPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.Chaos/experiments?api-version="+azureChaosStudioAPIVersion+"&running=true", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing running experiments, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	running := providerContractJSONMap(t, resp)
	if values, ok := running["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one running experiment, got %#v", running)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, experimentBase+"/executions?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing executions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	executions := providerContractJSONMap(t, resp)
	if values, ok := executions["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one execution, got %#v", executions)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, experimentBase+"/executions/"+executionID+"?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting execution, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	execution := providerContractJSONMap(t, resp)
	if execution["type"] != "Microsoft.Chaos/experiments/executions" {
		t.Fatalf("unexpected execution payload: %#v", execution)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, experimentBase+"/executions/"+executionID+"/getExecutionDetails?api-version="+azureChaosStudioAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting execution details, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	details := providerContractJSONMap(t, resp)
	if details["status"] == "" {
		t.Fatalf("expected execution details status, got %#v", details)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, locationBase+"/operationStatuses/"+operationID+"?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting operation status, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operationStatus := providerContractJSONMap(t, resp)
	if operationStatus["status"] == "" {
		t.Fatalf("expected operation status payload, got %#v", operationStatus)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, experimentBase+"/cancel?api-version="+azureChaosStudioAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 canceling experiment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cancelPayload := providerContractJSONMap(t, resp)
	cancelOperationID, _ := cancelPayload["operationId"].(string)
	if cancelOperationID == "" {
		t.Fatalf("expected cancel operationId, got %#v", cancelPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, locationBase+"/operationStatuses/"+cancelOperationID+"?api-version="+azureChaosStudioAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting cancel operation status, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		capabilityBase + "?api-version=" + azureChaosStudioAPIVersion,
		targetBase + "?api-version=" + azureChaosStudioAPIVersion,
		experimentBase + "?api-version=" + azureChaosStudioAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 deleting %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}
}

func TestAzureChaosStudioValidation(t *testing.T) {
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
	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.Chaos/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid target name",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-vm/providers/Microsoft.Compute/virtualMachines/vm-chaos/providers/Microsoft.Chaos/targets/not valid?api-version=" + azureChaosStudioAPIVersion,
			body:       []byte(`{"location":"eastus","properties":{"targetType":"Microsoft-VirtualMachine"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "capability target missing",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-vm/providers/Microsoft.Compute/virtualMachines/vm-chaos/providers/Microsoft.Chaos/targets/microsoft-agent/capabilities/Shutdown-1.0?api-version=" + azureChaosStudioAPIVersion,
			body:       []byte(`{"location":"eastus","properties":{}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "experiment missing selectors",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-chaos/providers/Microsoft.Chaos/experiments/exp-invalid?api-version=" + azureChaosStudioAPIVersion,
			body:       []byte(`{"location":"eastus","properties":{"steps":[{"name":"s1"}]}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cancel missing experiment",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-chaos/providers/Microsoft.Chaos/experiments/missing/cancel?api-version=" + azureChaosStudioAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "execution missing",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/rg-chaos/providers/Microsoft.Chaos/experiments/missing/executions/11111111-1111-1111-1111-111111111111?api-version=" + azureChaosStudioAPIVersion,
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
