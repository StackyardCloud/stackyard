package server

import (
	"net/http"
	"testing"
)

func TestAzureIntegrationEnvironmentLifecycle(t *testing.T) {
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
	resourceGroup := "rg-integration"
	spaceName := "space-a"
	applicationName := "app-a"
	resourceName := "resource-a"
	processName := "process-a"
	location := "eastus"
	base := "/azure/subscriptions/" + subscriptionID
	spacePath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.IntegrationSpaces/spaces/" + spaceName
	applicationPath := spacePath + "/applications/" + applicationName
	resourcePath := applicationPath + "/resources/" + resourceName
	processPath := applicationPath + "/businessProcesses/" + processName
	api := "?api-version=" + azureIntegrationEnvironmentAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.IntegrationSpaces/operations"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected operation values, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, spacePath+api, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"description":"integration space"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating space, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	space := providerContractJSONMap(t, resp)
	if space["type"] != "Microsoft.IntegrationSpaces/spaces" || space["name"] != spaceName {
		t.Fatalf("unexpected space payload: %#v", space)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, spacePath+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting space, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, spacePath+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading space, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.IntegrationSpaces/spaces"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing spaces by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	spaceList := providerContractJSONMap(t, resp)
	if values, ok := spaceList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one space, got %#v", spaceList)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.IntegrationSpaces/spaces"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing spaces by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, spacePath+api, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"},
		"properties":{"description":"patched"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching space, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, applicationPath+api, []byte(`{
		"tags":{"tier":"core"},
		"properties":{"description":"application"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating application, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	application := providerContractJSONMap(t, resp)
	if application["type"] != "Microsoft.IntegrationSpaces/spaces/applications" {
		t.Fatalf("unexpected application payload: %#v", application)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, applicationPath+api, []byte(`{
		"tags":{"tier":"updated"},
		"properties":{"description":"patched app"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching application, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, spacePath+"/applications"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing applications, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, resourcePath+api, []byte(`{
		"tags":{"kind":"connector"},
		"properties":{
			"resourceId":"/subscriptions/mock/resourceGroups/mock/providers/Microsoft.ServiceBus/namespaces/ns1",
			"resourceType":"Microsoft.ServiceBus/namespaces"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating application resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, resourcePath+api, []byte(`{
		"properties":{"resourceKind":"Messaging"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching application resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, applicationPath+"/resources"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing application resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resourceList := providerContractJSONMap(t, resp)
	if values, ok := resourceList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one application resource, got %#v", resourceList)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, processPath+api, []byte(`{
		"tags":{"domain":"orders"},
		"properties":{
			"tableName":"orders",
			"identifier":"orderId"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating business process, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, processPath+api, []byte(`{
		"properties":{"description":"patched process"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching business process, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, applicationPath+"/businessProcesses"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing business processes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, processPath+"/versions"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing business process versions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	versions := providerContractJSONMap(t, resp)
	versionValues, ok := versions["value"].([]any)
	if !ok || len(versionValues) != 1 {
		t.Fatalf("expected seeded version payload, got %#v", versions)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, processPath+"/versions/1.0.0"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting seeded version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	version := providerContractJSONMap(t, resp)
	versionProps, ok := version["properties"].(map[string]any)
	if !ok || versionProps["version"] != "1.0.0" {
		t.Fatalf("unexpected version payload: %#v", version)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, applicationPath+"/saveBusinessProcessDevelopmentArtifact"+api, []byte(`{
		"artifactName":"artifact-a",
		"content":"{\"definition\":true}"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 saving artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, applicationPath+"/getBusinessProcessDevelopmentArtifact"+api, []byte(`{
		"artifactName":"artifact-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, applicationPath+"/listBusinessProcessDevelopmentArtifacts"+api, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing artifacts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	artifactList := providerContractJSONMap(t, resp)
	if values, ok := artifactList["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one artifact, got %#v", artifactList)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, applicationPath+"/validateBusinessProcessDevelopmentArtifact"+api, []byte(`{
		"artifactName":"artifact-a",
		"content":"valid-content"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	validation := providerContractJSONMap(t, resp)
	if validation["isValid"] != true {
		t.Fatalf("expected valid artifact response, got %#v", validation)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, spacePath+"/infrastructureResources"+api, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing infrastructure resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	infrastructure := providerContractJSONMap(t, resp)
	infraValues, ok := infrastructure["value"].([]any)
	if !ok || len(infraValues) == 0 {
		t.Fatalf("expected seeded infrastructure resource, got %#v", infrastructure)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, spacePath+"/infrastructureResources/default-runtime"+api, []byte(`{
		"tags":{"seeded":"true","managed":"yes"},
		"properties":{"status":"Updating"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching infrastructure resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	infrastructureResource := providerContractJSONMap(t, resp)
	if infrastructureResource["location"] != location {
		t.Fatalf("unexpected infrastructure resource payload: %#v", infrastructureResource)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, applicationPath+"/deleteBusinessProcessDevelopmentArtifact"+api, []byte(`{
		"artifactName":"artifact-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting artifact, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, spacePath+api, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting space, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, spacePath+api, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing space, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureIntegrationEnvironmentValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-integration"
	base := "/azure/subscriptions/" + subscriptionID
	spacePath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.IntegrationSpaces/spaces/space-a"
	applicationPath := spacePath + "/applications/app-a"
	api := "?api-version=" + azureIntegrationEnvironmentAPIVersion

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid json",
			method:     http.MethodPut,
			path:       spacePath + api,
			body:       []byte(`{`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location",
			method:     http.MethodPut,
			path:       spacePath + api,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "create application under missing space",
			method:     http.MethodPut,
			path:       applicationPath + api,
			body:       []byte(`{"properties":{"description":"app"}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing application resource fields",
			method:     http.MethodPut,
			path:       applicationPath + "/resources/resource-a" + api,
			body:       []byte(`{"properties":{"resourceId":"id-only"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing business process fields",
			method:     http.MethodPut,
			path:       applicationPath + "/businessProcesses/process-a" + api,
			body:       []byte(`{"properties":{"tableName":"orders"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing artifact name",
			method:     http.MethodPost,
			path:       applicationPath + "/saveBusinessProcessDevelopmentArtifact" + api,
			body:       []byte(`{"content":"{}"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing business process version",
			method:     http.MethodGet,
			path:       applicationPath + "/businessProcesses/process-a/versions/9.9.9" + api,
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
