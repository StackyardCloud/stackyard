package server

import (
	"net/http"
	"testing"
)

func TestAzureDataManagerForAgriLifecycle(t *testing.T) {
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
	resourceGroup := "rg-agri"
	farmBeatName := "farm-a"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + farmBeatName + "?api-version=2023-06-01-preview"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AgFoodPlatform/operations?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.AgFoodPlatform/checkNameAvailability?api-version=2023-06-01-preview", []byte(`{
		"name":"farm-a",
		"type":"Microsoft.AgFoodPlatform/farmBeats"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	nameCheck := providerContractJSONMap(t, resp)
	if available, ok := nameCheck["nameAvailable"].(bool); !ok || !available {
		t.Fatalf("expected name to be available before create, got %#v", nameCheck)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"identity":{"type":"SystemAssigned"},
		"properties":{"publicNetworkAccess":"Enabled"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.AgFoodPlatform/farmBeats" {
		t.Fatalf("unexpected farmBeats payload: %#v", created)
	}
	asyncPath := resp.Header.Get("Azure-AsyncOperation")
	if asyncPath == "" {
		t.Fatalf("expected Azure-AsyncOperation header on create")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, asyncPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting operation result, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operationResult := providerContractJSONMap(t, resp)
	if operationResult["status"] != "Succeeded" {
		t.Fatalf("unexpected operation result payload: %#v", operationResult)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resourcePayload := providerContractJSONMap(t, resp)
	properties, ok := resourcePayload["properties"].(map[string]any)
	if !ok || properties["instanceUri"] == nil {
		t.Fatalf("expected instanceUri in payload, got %#v", resourcePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading farmBeats resource, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AgFoodPlatform/farmBeats?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing farmBeats by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	byGroup := providerContractJSONMap(t, resp)
	if values, ok := byGroup["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one farmBeats resource by group, got %#v", byGroup)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.AgFoodPlatform/farmBeats?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing farmBeats by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.AgFoodPlatform/checkNameAvailability?api-version=2023-06-01-preview", []byte(`{
		"name":"farm-a",
		"type":"Microsoft.AgFoodPlatform/farmBeats"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability after create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	unavailableCheck := providerContractJSONMap(t, resp)
	if available, ok := unavailableCheck["nameAvailable"].(bool); !ok || available {
		t.Fatalf("expected name to be unavailable after create, got %#v", unavailableCheck)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AgFoodPlatform/farmBeats/"+farmBeatName+"/farmBeatsExtensionDefinitions?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing extension definitions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	extensionDefs := providerContractJSONMap(t, resp)
	if values, ok := extensionDefs["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected extension definition values, got %#v", extensionDefs)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AgFoodPlatform/farmBeats/"+farmBeatName+"/farmBeatsSolutionDefinitions?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing solution definitions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	dataConnectorPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + farmBeatName + "/dataConnectors/connector-a?api-version=2023-06-01-preview"
	resp = providerContractRequest(t, ts, http.MethodPut, dataConnectorPath, []byte(`{
		"properties":{
			"credentials":{"kind":"OAuth","clientId":"client-a","clientSecret":"secret-a"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating data connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	dataConnector := providerContractJSONMap(t, resp)
	if dataConnector["type"] != "Microsoft.AgFoodPlatform/farmBeats/dataConnectors" {
		t.Fatalf("unexpected data connector payload: %#v", dataConnector)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, dataConnectorPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting data connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AgFoodPlatform/farmBeats/"+farmBeatName+"/dataConnectors?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing data connectors, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	extensionPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + farmBeatName + "/extensions/weather-insights?api-version=2023-06-01-preview"
	resp = providerContractRequest(t, ts, http.MethodPut, extensionPath, []byte(`{
		"properties":{"version":"1.0.0"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating extension, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AgFoodPlatform/farmBeats/"+farmBeatName+"/extensions?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing extensions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	privateEndpointPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + farmBeatName + "/privateEndpointConnections/pec-a?api-version=2023-06-01-preview"
	resp = providerContractRequest(t, ts, http.MethodPut, privateEndpointPath, []byte(`{
		"properties":{"privateLinkServiceConnectionState":{"status":"Approved","description":"Approved by test"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AgFoodPlatform/farmBeats/"+farmBeatName+"/privateLinkResources?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	solutionPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AgFoodPlatform/farmBeats/" + farmBeatName + "/solutions/crop-analytics?api-version=2023-06-01-preview"
	resp = providerContractRequest(t, ts, http.MethodPut, solutionPath, []byte(`{
		"properties":{"version":"1.0.0"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating solution, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"sensorIntegration":{"enabled":true}}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	if tags, ok := patched["tags"].(map[string]any); !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, solutionPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting solution, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, privateEndpointPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, extensionPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting extension, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, dataConnectorPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting data connector, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureDataManagerForAgriValidationAndNotFound(t *testing.T) {
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

	basePath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-agri/providers/Microsoft.AgFoodPlatform/farmBeats/farm-a?api-version=2023-06-01-preview"

	resp := providerContractRequest(t, ts, http.MethodPut, basePath, []byte(`{"properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, basePath, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AgFoodPlatform/checkNameAvailability?api-version=2023-06-01-preview", []byte(`{"type":"Microsoft.AgFoodPlatform/farmBeats"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name in name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, basePath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, basePath, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AgFoodPlatform/locations/eastus/operationResults/missing?api-version=2023-06-01-preview", nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing operation result, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-agri/providers/Microsoft.AgFoodPlatform/farmBeats/farm-a/dataConnectors/connector-a?api-version=2023-06-01-preview", []byte(`{"properties":{"credentials":{"kind":"OAuth"}}}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 creating child resource under missing farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, basePath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing farmBeats resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AgFoodPlatform/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
