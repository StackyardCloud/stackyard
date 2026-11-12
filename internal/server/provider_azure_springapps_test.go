package server

import (
	"net/http"
	"testing"
)

func TestAzureSpringAppsLifecycle(t *testing.T) {
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
	resourceGroup := "rg-spring"
	serviceName := "spring-a"
	appName := "app-a"
	deploymentName := "deployment-a"
	base := "/azure/subscriptions/" + subscriptionID
	serviceBase := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/" + serviceName
	appBase := serviceBase + "/apps/" + appName
	deploymentBase := appBase + "/deployments/" + deploymentName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AppPlatform/operations?api-version="+azureSpringAppsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, serviceBase+"?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"networkProfile":"default"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating spring service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	service := providerContractJSONMap(t, resp)
	if service["type"] != "Microsoft.AppPlatform/Spring" {
		t.Fatalf("unexpected service payload: %#v", service)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, serviceBase+"/enableTestEndpoint?api-version="+azureSpringAppsAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 enabling test endpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, appBase+"?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"tags":{"tier":"core"},
		"properties":{"public":true}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating app, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, deploymentBase+"?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"source":{"type":"Jar","relativePath":"app.jar"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating deployment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, appBase+"/setActiveDeployments?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"deploymentName":"deployment-a"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 setting active deployment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, appBase+"/bindings/binding-a?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"resourceId":"/subscriptions/mock/resourceGroups/mock/providers/Microsoft.Storage/storageAccounts/storagea"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating binding, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, appBase+"/domains/domain-a?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"certName":"cert-a"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating domain, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, serviceBase+"/certificates/cert-a?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"issuer":"Stackyard CA"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating certificate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, serviceBase+"/containerRegistries/registry-a?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"server":"registry.azurecr.io"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating container registry, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, serviceBase+"/containerRegistries/registry-a/validate?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"server":"registry.azurecr.io"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating container registry, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		serviceBase + "/gateways/gateway-a?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/apiPortals/portal-a?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/applicationLiveViews/live-a?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/configurationServices/config-a?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/serviceRegistries/registry-a?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/devToolPortals/devportal-a?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/storages/storage-a?api-version=" + azureSpringAppsAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"properties":{"enabled":true}}`), headers)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 creating child resource for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, serviceBase+"/configurationServices/config-a/validate?api-version="+azureSpringAppsAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating configuration service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	buildServiceBase := serviceBase + "/buildServices/default"
	resp = providerContractRequest(t, ts, http.MethodPut, buildServiceBase+"?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"resourceRequests":{"cpu":"1","memory":"2Gi"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating build service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, buildServiceBase+"/builds/build-a?api-version="+azureSpringAppsAPIVersion, []byte(`{
		"properties":{"builderName":"default"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating build, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		buildServiceBase + "/buildResults?api-version=" + azureSpringAppsAPIVersion,
		buildServiceBase + "/supportedBuildpacks?api-version=" + azureSpringAppsAPIVersion,
		buildServiceBase + "/supportedStacks?api-version=" + azureSpringAppsAPIVersion,
		buildServiceBase + "/builders/default/buildpackBindings?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/apps?api-version=" + azureSpringAppsAPIVersion,
		appBase + "/deployments?api-version=" + azureSpringAppsAPIVersion,
		appBase + "/bindings?api-version=" + azureSpringAppsAPIVersion,
		appBase + "/domains?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/certificates?api-version=" + azureSpringAppsAPIVersion,
		serviceBase + "/containerRegistries?api-version=" + azureSpringAppsAPIVersion,
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring?api-version=" + azureSpringAppsAPIVersion,
		base + "/providers/Microsoft.AppPlatform/Spring?api-version=" + azureSpringAppsAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, serviceBase+"?api-version="+azureSpringAppsAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting service, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureSpringAppsValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-spring"
	base := "/azure/subscriptions/" + subscriptionID

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid service name",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/bad name?api-version=" + azureSpringAppsAPIVersion,
			body:       []byte(`{"location":"eastus"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing service location",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/spring-a?api-version=" + azureSpringAppsAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing service get",
			method:     http.MethodGet,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/missing?api-version=" + azureSpringAppsAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "binding missing parent",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/spring-a/apps/app-a/bindings/binding-a?api-version=" + azureSpringAppsAPIVersion,
			body:       []byte(`{"properties":{"resourceId":"/subscriptions/mock/resourceGroups/mock/providers/Microsoft.Storage/storageAccounts/storagea"}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "binding missing resource id",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/spring-a/apps/app-a/bindings/binding-a?api-version=" + azureSpringAppsAPIVersion,
			body:       []byte(`{"properties":{}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing active deployment name",
			method:     http.MethodPost,
			path:       base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppPlatform/Spring/spring-a/apps/app-a/setActiveDeployments?api-version=" + azureSpringAppsAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid api version empty",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.AppPlatform/operations?api-version=",
			wantStatus: http.StatusBadRequest,
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
