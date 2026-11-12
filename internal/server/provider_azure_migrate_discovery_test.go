package server

import (
	"net/http"
	"testing"
)

func TestAzureMigrateDiscoveryLifecycle(t *testing.T) {
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
	resourceGroup := "rg-discovery"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.OffAzure"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.OffAzure/operations?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	vmwareSite := base + "/VMwareSites/vmw-site?api-version=" + azureMigrateDiscoveryAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, vmwareSite, []byte(`{"location":"eastus","tags":{"env":"dev"}}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating vmware site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vmware := providerContractJSONMap(t, resp)
	if vmware["type"] != "Microsoft.OffAzure/VMwareSites" {
		t.Fatalf("unexpected vmware site payload: %#v", vmware)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, vmwareSite, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on vmware HEAD, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/providers/Microsoft.OffAzure/VMwareSites?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vmware sites by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/VMwareSites/vmw-site/summary?api-version="+azureMigrateDiscoveryAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on vmware summary, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	summary := providerContractJSONMap(t, resp)
	if summary["machineCount"] == nil {
		t.Fatalf("expected machineCount in summary, got %#v", summary)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/VMwareSites/vmw-site/refresh?api-version="+azureMigrateDiscoveryAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 on vmware refresh, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if resp.Header.Get("Azure-AsyncOperation") == "" {
		t.Fatalf("expected Azure-AsyncOperation header on refresh")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/VMwareSites/vmw-site/machines?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vmware machines, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	machines := providerContractJSONMap(t, resp)
	values, ok := machines["value"].([]any)
	if !ok || len(values) == 0 {
		t.Fatalf("expected vmware machine fixtures, got %#v", machines)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/VMwareSites/vmw-site/machines/machine-1?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting vmware machine, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/VMwareSites/vmw-site/machines/machine-1/start?api-version="+azureMigrateDiscoveryAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 starting vmware machine, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/VMwareSites/vmw-site/machines/machine-1/stop?api-version="+azureMigrateDiscoveryAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 stopping vmware machine, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/VMwareSites/vmw-site/runAsAccounts?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vmware run as accounts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/VMwareSites/vmw-site/runAsAccounts/runas-default?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting vmware run as account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/VMwareSites/vmw-site/vCenters?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vmware vcenters, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/VMwareSites/vmw-site/vCenters/vcenter-default?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting vmware vcenter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	hypervSite := base + "/HyperVSites/hv-site?api-version=" + azureMigrateDiscoveryAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, hypervSite, []byte(`{"location":"eastus2","tags":{"env":"test"}}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating hyperv site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, hypervSite, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting hyperv site, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/HyperVSites/hv-site/siteUsage?api-version="+azureMigrateDiscoveryAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on hyperv siteUsage, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/HyperVSites/hv-site/refresh?api-version="+azureMigrateDiscoveryAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 on hyperv refresh, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/HyperVSites/hv-site/machines?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing hyperv machines, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/HyperVSites/hv-site/runAsAccounts/runas-default?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting hyperv run as account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/HyperVSites/hv-site/hosts/host-1?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting hyperv host, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/HyperVSites/hv-site/clusters/cluster-1?api-version="+azureMigrateDiscoveryAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting hyperv cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureMigrateDiscoveryValidation(t *testing.T) {
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
			path:       "/azure/providers/Microsoft.OffAzure/operations?api-version=",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location",
			method:     http.MethodPut,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OffAzure/VMwareSites/site-a?api-version=" + azureMigrateDiscoveryAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing site",
			method:     http.MethodGet,
			path:       "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OffAzure/HyperVSites/missing?api-version=" + azureMigrateDiscoveryAPIVersion,
			wantStatus: http.StatusNotFound,
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
}
