package server

import (
	"net/http"
	"testing"
)

func TestAzureStackHCILifecycle(t *testing.T) {
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
	resourceGroup := "rg-stackhci"
	clusterName := "cluster-a"
	clusterBase := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureStackHCI/clusters/" + clusterName
	arcBase := clusterBase + "/arcSettings/default"
	extensionBase := arcBase + "/extensions/telemetry"
	deploymentBase := clusterBase + "/deploymentSettings/default"
	updateBase := clusterBase + "/updates/update-1"
	clusterURI := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureStackHCI/clusters/" + clusterName
	edgeBase := "/azure" + clusterURI + "/providers/Microsoft.AzureStackHCI/edgeDevices/edge-a"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AzureStackHCI/operations?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if values, ok := providerContractJSONMap(t, resp)["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations list")
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"dev"},
		"properties":{"desiredProperties":{"diagnosticLevel":"Enhanced"}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, clusterBase+"?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading cluster, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AzureStackHCI/clusters?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing clusters by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, clusterBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"tags":{"env":"prod"},
		"properties":{"status":"Ready"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, clusterBase+"/configureRemoteSupport?api-version="+azureStackHCIAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 configuring remote support, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, clusterBase+"?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cluster := providerContractJSONMap(t, resp)
	props, ok := cluster["properties"].(map[string]any)
	if !ok || props["remoteSupportEnabled"] != true {
		t.Fatalf("expected cluster action state, got %#v", cluster)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, arcBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"properties":{"connectivityState":"Connected"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating arc setting, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, arcBase+"/createIdentity?api-version="+azureStackHCIAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating arc identity, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, extensionBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"properties":{"version":"1.0.0","autoUpgradeMinorVersion":true}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating extension, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, extensionBase+"/upgrade?api-version="+azureStackHCIAPIVersion, []byte(`{"targetVersion":"1.1.0"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 upgrading extension, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, deploymentBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"properties":{"deploymentMode":"ValidateOnly","deploymentIntent":"Deploy"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating deployment settings, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, updateBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"properties":{"packageVersion":"2404.0","state":"Available"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, updateBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 applying update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, clusterBase+"/updateSummaries/default?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting update summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, updateBase+"/updateRuns/run-latest?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting update run, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, edgeBase+"?api-version="+azureStackHCIAPIVersion, []byte(`{
		"kind":"HCI",
		"properties":{"deviceType":"IndustrialGateway"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating edge device, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, edgeBase+"/validate?api-version="+azureStackHCIAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 validating edge device, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure"+clusterURI+"/providers/Microsoft.AzureStackHCI/edgeDevices?api-version="+azureStackHCIAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing edge devices, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureStackHCIValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-stackhci"
	clusterBase := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureStackHCI/clusters"

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "cluster create missing location",
			method:     http.MethodPut,
			path:       clusterBase + "/invalid?api-version=" + azureStackHCIAPIVersion,
			body:       []byte(`{"identity":{"type":"SystemAssigned"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "cluster create invalid json",
			method:     http.MethodPut,
			path:       clusterBase + "/invalid?api-version=" + azureStackHCIAPIVersion,
			body:       []byte(`{"location":`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing cluster",
			method:     http.MethodGet,
			path:       clusterBase + "/missing?api-version=" + azureStackHCIAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "edge device missing parent",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AzureStackHCI/clusters/missing/providers/Microsoft.AzureStackHCI/edgeDevices/edge-a?api-version=" + azureStackHCIAPIVersion,
			body:       []byte(`{"kind":"HCI","properties":{"deviceType":"IndustrialGateway"}}`),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
