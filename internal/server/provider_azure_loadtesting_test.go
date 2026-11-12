package server

import (
	"net/http"
	"testing"
)

func TestAzureLoadTestingLifecycle(t *testing.T) {
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
	resourceGroup := "rg-loadtesting"
	loadTestName := "loadtest-a"
	base := "/azure/subscriptions/" + subscriptionID
	resource := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/loadTests/" + loadTestName + "?api-version=" + azureLoadTestingAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.LoadTestService/operations?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"dev"},
		"properties":{
			"description":"stackyard load test",
			"encryption":{"type":"SystemAssigned"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	created := providerContractJSONMap(t, resp)
	if created["type"] != "Microsoft.LoadTestService/loadTests" {
		t.Fatalf("unexpected load test payload: %#v", created)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getPayload := providerContractJSONMap(t, resp)
	properties, ok := getPayload["properties"].(map[string]any)
	if !ok || properties["provisioningState"] != "Succeeded" || properties["dataPlaneURI"] == "" {
		t.Fatalf("unexpected load test properties: %#v", getPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, resource, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading load test resource, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/loadTests?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing load tests by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	byGroup := providerContractJSONMap(t, resp)
	if values, ok := byGroup["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one load test by resource group, got %#v", byGroup)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.LoadTestService/loadTests?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing load tests by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bySubscription := providerContractJSONMap(t, resp)
	if values, ok := bySubscription["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one load test by subscription, got %#v", bySubscription)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/loadTests/"+loadTestName+"/outboundNetworkDependenciesEndpoints?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing outbound dependencies, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	dependencies := providerContractJSONMap(t, resp)
	if values, ok := dependencies["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected outbound dependency endpoints, got %#v", dependencies)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/loadTests/"+loadTestName+"/limits/maxMonthlyVirtualUserHours?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting max monthly virtual user hours, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	limitBefore := providerContractJSONMap(t, resp)
	if props, ok := limitBefore["properties"].(map[string]any); !ok || props["value"] == nil {
		t.Fatalf("expected limit payload, got %#v", limitBefore)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.LoadTestService/loadTests/"+loadTestName+"/limits/maxMonthlyVirtualUserHours/set?api-version="+azureLoadTestingAPIVersion, []byte(`{"properties":{"value":24000}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 setting max monthly virtual user hours, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	limitAfter := providerContractJSONMap(t, resp)
	props, ok := limitAfter["properties"].(map[string]any)
	if !ok || props["value"].(float64) != 24000 {
		t.Fatalf("expected updated limit payload, got %#v", limitAfter)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"description":"patched description"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	tags, ok := patched["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureLoadTestingMappingsLifecycle(t *testing.T) {
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
	resourceGroup := "rg-loadtesting"
	loadTestID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/loadTests/loadtest-a"
	parentResourceURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Web/sites/site-a"
	parentPath := "/azure" + parentResourceURI

	createLoadTestPath := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.LoadTestService/loadTests/loadtest-a?api-version=" + azureLoadTestingAPIVersion
	resp := providerContractRequest(t, ts, http.MethodPut, createLoadTestPath, []byte(`{"location":"eastus","properties":{"description":"for mappings"}}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating referenced load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	mappingPath := parentPath + "/providers/Microsoft.LoadTestService/loadTestMappings/mapping-a?api-version=" + azureLoadTestingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, mappingPath, []byte(`{
		"properties":{
			"sourceResourceId":"`+parentResourceURI+`",
			"azureLoadTestingResourceId":"`+loadTestID+`",
			"testId":"test-a"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating load test mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	createdMapping := providerContractJSONMap(t, resp)
	if createdMapping["type"] != "Microsoft.LoadTestService/loadTestMappings" {
		t.Fatalf("unexpected mapping payload: %#v", createdMapping)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, mappingPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting load test mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, mappingPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading load test mapping, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, parentPath+"/providers/Microsoft.LoadTestService/loadTestMappings?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing load test mappings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listMappings := providerContractJSONMap(t, resp)
	if values, ok := listMappings["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one mapping, got %#v", listMappings)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, mappingPath, []byte(`{
		"properties":{
			"sourceResourceId":"`+parentResourceURI+`",
			"azureLoadTestingResourceId":"`+loadTestID+`",
			"testId":"test-b"
		}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching load test mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patchedMapping := providerContractJSONMap(t, resp)
	mappingProps, ok := patchedMapping["properties"].(map[string]any)
	if !ok || mappingProps["testId"] != "test-b" {
		t.Fatalf("expected patched mapping properties, got %#v", patchedMapping)
	}

	profileMappingPath := parentPath + "/providers/Microsoft.LoadTestService/loadTestProfileMappings/profile-a?api-version=" + azureLoadTestingAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, profileMappingPath, []byte(`{
		"properties":{
			"sourceResourceId":"`+parentResourceURI+`",
			"azureLoadTestingResourceId":"`+loadTestID+`",
			"testProfileId":"profile-a"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating load test profile mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, profileMappingPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting load test profile mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, parentPath+"/providers/Microsoft.LoadTestService/loadTestProfileMappings?api-version="+azureLoadTestingAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing load test profile mappings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listProfiles := providerContractJSONMap(t, resp)
	if values, ok := listProfiles["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one profile mapping, got %#v", listProfiles)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, mappingPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting load test mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, profileMappingPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting load test profile mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureLoadTestingValidationAndNotFound(t *testing.T) {
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

	resource := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-loadtesting/providers/Microsoft.LoadTestService/loadTests/loadtest-a?api-version=" + azureLoadTestingAPIVersion
	parentResourceURI := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-loadtesting/providers/Microsoft.Web/sites/site-a"
	mappingPath := "/azure" + parentResourceURI + "/providers/Microsoft.LoadTestService/loadTestMappings/mapping-a?api-version=" + azureLoadTestingAPIVersion

	resp := providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{"properties":{"description":"missing location"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resource, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resource, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, resource, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, resource, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing load test resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, mappingPath, []byte(`{
		"properties":{
			"sourceResourceId":"`+parentResourceURI+`",
			"azureLoadTestingResourceId":"",
			"testId":"test-a"
		}
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 creating invalid mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.LoadTestService/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
