package server

import (
	"net/http"
	"testing"
)

func TestAzureComputeScheduleLifecycle(t *testing.T) {
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
	location := "eastus"
	base := "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.ComputeSchedule/locations/" + location
	vmID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-schedule/providers/Microsoft.Compute/virtualMachines/vm-a"
	vmFailID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-schedule/providers/Microsoft.Compute/virtualMachines/vm-fail"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.ComputeSchedule/operations?api-version="+azureComputeScheduleAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) < 8 {
		t.Fatalf("expected compute schedule operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesSubmitStart?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-submit-start",
		"resources":{"ids":["`+vmID+`"]},
		"executionParameters":{"optimizationPreference":"Cost"},
		"schedule":{"deadline":"2026-04-13T15:04:05Z","timezone":"UTC","deadlineType":"InitiateAt"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 submit start, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	submitPayload := providerContractJSONMap(t, resp)
	submitResults, ok := submitPayload["results"].([]any)
	if !ok || len(submitResults) != 1 {
		t.Fatalf("expected one submit result, got %#v", submitPayload)
	}
	submitResult, ok := submitResults[0].(map[string]any)
	if !ok {
		t.Fatalf("expected submit result payload, got %#v", submitPayload)
	}
	submitOperation, ok := submitResult["operation"].(map[string]any)
	if !ok || submitOperation["state"] != "Scheduled" {
		t.Fatalf("expected scheduled operation, got %#v", submitPayload)
	}
	submitOperationID, _ := submitOperation["operationId"].(string)
	if submitOperationID == "" {
		t.Fatalf("expected operation id in submit response, got %#v", submitPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesGetOperationStatus?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-submit-start",
		"operationIds":["`+submitOperationID+`"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting operation status, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	statusPayload := providerContractJSONMap(t, resp)
	statusResults, ok := statusPayload["results"].([]any)
	if !ok || len(statusResults) != 1 {
		t.Fatalf("expected one status result, got %#v", statusPayload)
	}
	statusResult, _ := statusResults[0].(map[string]any)
	statusOperation, _ := statusResult["operation"].(map[string]any)
	if statusOperation["state"] != "InProgress" {
		t.Fatalf("expected in-progress operation after first poll, got %#v", statusPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesGetOperationStatus?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-submit-start",
		"operationIds":["`+submitOperationID+`"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting operation status second time, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	statusPayload = providerContractJSONMap(t, resp)
	statusResults, _ = statusPayload["results"].([]any)
	statusResult, _ = statusResults[0].(map[string]any)
	statusOperation, _ = statusResult["operation"].(map[string]any)
	if statusOperation["state"] != "Succeeded" {
		t.Fatalf("expected succeeded operation after second poll, got %#v", statusPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesExecuteDeallocate?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationId":"corr-execute-deallocate",
		"resources":{"ids":["`+vmID+`","`+vmFailID+`"]},
		"executionParameters":{"retryPolicy":{"retryCount":1,"retryWindowInMinutes":5}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 execute deallocate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	executePayload := providerContractJSONMap(t, resp)
	executeResults, ok := executePayload["results"].([]any)
	if !ok || len(executeResults) != 2 {
		t.Fatalf("expected two execute results, got %#v", executePayload)
	}
	failResult, _ := executeResults[1].(map[string]any)
	failOperation, _ := failResult["operation"].(map[string]any)
	failOperationID, _ := failOperation["operationId"].(string)
	if failOperationID == "" {
		t.Fatalf("expected fail operation id, got %#v", executePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesGetOperationErrors?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-execute-deallocate",
		"operationIds":["`+failOperationID+`"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting operation errors, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	errorPayload := providerContractJSONMap(t, resp)
	errorResults, ok := errorPayload["results"].([]any)
	if !ok || len(errorResults) != 1 {
		t.Fatalf("expected one error result, got %#v", errorPayload)
	}
	errorResult, _ := errorResults[0].(map[string]any)
	errors, ok := errorResult["errors"].([]any)
	if !ok || len(errors) != 1 {
		t.Fatalf("expected one seeded error, got %#v", errorPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesSubmitDeallocate?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-cancel",
		"resources":{"ids":["`+vmID+`"]},
		"schedule":{"deadline":"2026-04-13T16:04:05Z","timezone":"UTC","deadlineType":"InitiateAt"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 submit deallocate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cancelSeedPayload := providerContractJSONMap(t, resp)
	cancelResults, _ := cancelSeedPayload["results"].([]any)
	cancelSeedResult, _ := cancelResults[0].(map[string]any)
	cancelSeedOperation, _ := cancelSeedResult["operation"].(map[string]any)
	cancelOperationID, _ := cancelSeedOperation["operationId"].(string)
	if cancelOperationID == "" {
		t.Fatalf("expected cancelable operation id, got %#v", cancelSeedPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesCancelOperations?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-cancel",
		"operationIds":["`+cancelOperationID+`"]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 canceling operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cancelPayload := providerContractJSONMap(t, resp)
	cancelResults, _ = cancelPayload["results"].([]any)
	cancelResult, _ := cancelResults[0].(map[string]any)
	cancelOperation, _ := cancelResult["operation"].(map[string]any)
	if cancelOperation["state"] != "Canceled" {
		t.Fatalf("expected canceled operation state, got %#v", cancelPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/virtualMachinesExecuteCreate?api-version="+azureComputeScheduleAPIVersion, []byte(`{
		"correlationid":"corr-preview-create",
		"resources":{"ids":["/subscriptions/`+subscriptionID+`/resourceGroups/rg-schedule/providers/Microsoft.Compute/virtualMachines/vm-preview"]},
		"executionParameters":{"optimizationPreference":"Availability"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 execute create preview route, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	previewPayload := providerContractJSONMap(t, resp)
	previewResults, ok := previewPayload["results"].([]any)
	if !ok || len(previewResults) != 1 {
		t.Fatalf("expected preview result, got %#v", previewPayload)
	}
	previewResult, _ := previewResults[0].(map[string]any)
	previewOperation, _ := previewResult["operation"].(map[string]any)
	if previewOperation["preview"] != true {
		t.Fatalf("expected preview flag on preview route, got %#v", previewPayload)
	}
}

func TestAzureComputeScheduleValidation(t *testing.T) {
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

	vmID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-schedule/providers/Microsoft.Compute/virtualMachines/vm-a"

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
			path:       "/azure/providers/Microsoft.ComputeSchedule/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid subscription id",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/not-a-sub/providers/Microsoft.ComputeSchedule/locations/eastus/virtualMachinesSubmitStart?api-version=" + azureComputeScheduleAPIVersion,
			body:       []byte(`{"correlationid":"corr","resources":{"ids":["` + vmID + `"]},"schedule":{"deadline":"2026-04-13T15:04:05Z"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing correlation id",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeSchedule/locations/eastus/virtualMachinesSubmitStart?api-version=" + azureComputeScheduleAPIVersion,
			body:       []byte(`{"resources":{"ids":["` + vmID + `"]},"schedule":{"deadline":"2026-04-13T15:04:05Z"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty resource ids",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeSchedule/locations/eastus/virtualMachinesSubmitStart?api-version=" + azureComputeScheduleAPIVersion,
			body:       []byte(`{"correlationid":"corr","resources":{"ids":[]},"schedule":{"deadline":"2026-04-13T15:04:05Z"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing schedule for submit",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeSchedule/locations/eastus/virtualMachinesSubmitDeallocate?api-version=" + azureComputeScheduleAPIVersion,
			body:       []byte(`{"correlationid":"corr","resources":{"ids":["` + vmID + `"]}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid vm resource id",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeSchedule/locations/eastus/virtualMachinesExecuteStart?api-version=" + azureComputeScheduleAPIVersion,
			body:       []byte(`{"correlationid":"corr","resources":{"ids":["/subscriptions/000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"]}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing operation ids on status",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.ComputeSchedule/locations/eastus/virtualMachinesGetOperationStatus?api-version=" + azureComputeScheduleAPIVersion,
			body:       []byte(`{"correlationid":"corr"}`),
			wantStatus: http.StatusBadRequest,
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
