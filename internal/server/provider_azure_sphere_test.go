package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureSphereLifecycle(t *testing.T) {
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
	resourceGroup := "rg-sphere"
	catalogName := "catalog-a"
	productName := "product-a"
	deviceGroupName := "group-a"
	deviceName := strings.Repeat("0", 128)
	deploymentName := "deployment-a"
	imageName := "image-a"
	location := "eastus"

	base := "/azure/subscriptions/" + subscriptionID
	catalogBase := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureSphere/catalogs/" + catalogName
	catalogPath := catalogBase + "?api-version=" + azureSphereAPIVersion
	productBase := catalogBase + "/products/" + productName
	productPath := productBase + "?api-version=" + azureSphereAPIVersion
	deviceGroupBase := productBase + "/deviceGroups/" + deviceGroupName
	deviceGroupPath := deviceGroupBase + "?api-version=" + azureSphereAPIVersion
	deviceBase := deviceGroupBase + "/devices/" + deviceName
	devicePath := deviceBase + "?api-version=" + azureSphereAPIVersion
	deploymentPath := deviceGroupBase + "/deployments/" + deploymentName + "?api-version=" + azureSphereAPIVersion
	imagePath := catalogBase + "/images/" + imageName + "?api-version=" + azureSphereAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AzureSphere/operations?api-version="+azureSphereAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected operations value payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, catalogPath, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"description":"catalog"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating catalog, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	catalog := providerContractJSONMap(t, resp)
	if catalog["type"] != "Microsoft.AzureSphere/catalogs" {
		t.Fatalf("unexpected catalog payload: %#v", catalog)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, catalogPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting catalog, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, catalogPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading catalog, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AzureSphere/catalogs?api-version="+azureSphereAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing catalogs by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.AzureSphere/catalogs?api-version="+azureSphereAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing catalogs by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, productPath, []byte(`{
		"properties":{"description":"product"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating product, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, productBase+"/generateDefaultDeviceGroups?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 generating default device groups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	defaultGroups := providerContractJSONMap(t, resp)
	if values, ok := defaultGroups["value"].([]any); !ok || len(values) < 2 {
		t.Fatalf("expected generated device groups, got %#v", defaultGroups)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, deviceGroupPath, []byte(`{
		"properties":{"description":"group"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating device group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, deviceGroupBase+"/claimDevices?api-version="+azureSphereAPIVersion, []byte(`{
		"deviceIdentifiers":["`+deviceName+`"]
	}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 claiming devices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, devicePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting device, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, productBase+"/countDevices?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 counting product devices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	productCount := providerContractJSONMap(t, resp)
	if productCount["value"] == nil {
		t.Fatalf("expected count value, got %#v", productCount)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, deviceGroupBase+"/countDevices?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 counting device group devices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, deviceGroupBase+"/devices?api-version="+azureSphereAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing devices by group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	devices := providerContractJSONMap(t, resp)
	if values, ok := devices["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one device in group, got %#v", devices)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, deploymentPath, []byte(`{
		"properties":{"description":"deployment"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating deployment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, imagePath, []byte(`{
		"properties":{"imageName":"image-a","description":"image"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, catalogBase+"/uploadImage?api-version="+azureSphereAPIVersion, []byte(`{
		"name":"uploaded-image",
		"properties":{"description":"uploaded image"}
	}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 uploading image, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, catalogBase+"/listDevices?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing catalog devices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, catalogBase+"/listDeviceGroups?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing catalog device groups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, catalogBase+"/listDeployments?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing catalog deployments, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, catalogBase+"/listDeviceInsights?api-version="+azureSphereAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing device insights, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	insights := providerContractJSONMap(t, resp)
	if values, ok := insights["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected device insight rows, got %#v", insights)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, catalogPath, []byte(`{
		"tags":{"env":"stage","owner":"stackyard"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching catalog, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	tags, ok := patched["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, catalogPath, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting catalog, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, catalogPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing catalog, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	_ = location
}

func TestAzureSphereValidationAndNotFound(t *testing.T) {
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

	base := "/azure/subscriptions/00000000-0000-0000-0000-000000000000"
	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "missing catalog location",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/rg-sphere/providers/Microsoft.AzureSphere/catalogs/catalog-a?api-version=" + azureSphereAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid catalog json",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/rg-sphere/providers/Microsoft.AzureSphere/catalogs/catalog-a?api-version=" + azureSphereAPIVersion,
			body:       []byte(`{"location":`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing catalog get",
			method:     http.MethodGet,
			path:       base + "/resourceGroups/rg-sphere/providers/Microsoft.AzureSphere/catalogs/missing?api-version=" + azureSphereAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing parent product create",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/rg-sphere/providers/Microsoft.AzureSphere/catalogs/catalog-a/products/product-a?api-version=" + azureSphereAPIVersion,
			body:       []byte(`{"properties":{"description":"product"}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing parent device group create",
			method:     http.MethodPut,
			path:       base + "/resourceGroups/rg-sphere/providers/Microsoft.AzureSphere/catalogs/catalog-a/products/product-a/deviceGroups/group-a?api-version=" + azureSphereAPIVersion,
			body:       []byte(`{"properties":{"description":"group"}}`),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/providers/Microsoft.AzureSphere/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tc.method, tc.path, tc.body, headers)
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tc.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
