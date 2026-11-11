package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAzureAVSLifecycleAndChildResources(t *testing.T) {
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
	resourceGroup := "rg-avs"
	privateCloudName := "pc-a"
	clusterName := "cluster-a"
	base := "/azure/subscriptions/" + subscriptionID
	privateCloudPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AVS/privateClouds/" + privateCloudName
	clusterPath := privateCloudPath + "/clusters/" + clusterName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AVS/operations?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.AVS/locations/eastus/checkQuotaAvailability?api-version=2025-09-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking quota, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	quota := providerContractJSONMap(t, resp)
	if quota["quotaEnabled"] != true {
		t.Fatalf("unexpected quota payload: %#v", quota)
	}

	createPrivateCloud := []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"networkBlock":"10.0.0.0/22"}
	}`)
	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"?api-version=2025-09-01", createPrivateCloud, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	privateCloud := providerContractJSONMap(t, resp)
	if privateCloud["name"] != privateCloudName || privateCloud["type"] != "Microsoft.AVS/privateClouds" {
		t.Fatalf("unexpected private cloud payload: %#v", privateCloud)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, privateCloudPath+"?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from private cloud HEAD, got %d", resp.StatusCode)
	}
	_ = providerContractBody(t, resp)

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.AVS/privateClouds?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private clouds by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.AVS/privateClouds?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private clouds in subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, privateCloudPath+"?api-version=2025-09-01", []byte(`{"tags":{"env":"stage","owner":"stackyard"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	patched := providerContractJSONMap(t, resp)
	tags, ok := patched["tags"].(map[string]any)
	if !ok || tags["owner"] != "stackyard" {
		t.Fatalf("expected patched tags, got %#v", patched)
	}

	for _, suffix := range []string{"/listAdminCredentials", "/getVcfLicense", "/rotateNsxtPassword", "/rotateVcenterPassword"} {
		resp = providerContractRequest(t, ts, http.MethodPost, privateCloudPath+suffix+"?api-version=2025-09-01", []byte(`{}`), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", suffix, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/authorizations/auth-a?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"expressRouteId":"er-a"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating authorization, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/authorizations?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing authorizations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/globalReachConnections/grc-a?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"peerExpressRouteCircuit":"peer-a"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating global reach connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/globalReachConnections?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing global reach connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/licenses/license-a?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"edition":"Datacenter"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating license, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, privateCloudPath+"/licenses/license-a/getProperties?api-version=2025-09-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting license properties, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/addons?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing addons, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/maintenances?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing maintenances, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, privateCloudPath+"/maintenances/default/initiateChecks?api-version=2025-09-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 initiating maintenance checks, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, privateCloudPath+"/maintenances/default/reschedule?api-version=2025-09-01", []byte(`{"rescheduleTime":"2026-04-01T00:00:00Z"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 rescheduling maintenance, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/scriptExecutions/exec-a?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"scriptCmdletId":"Get-Cluster"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating script execution, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, privateCloudPath+"/scriptExecutions/exec-a/getExecutionLogs?api-version=2025-09-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting script execution logs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/iscsiPaths/default?api-version=2025-09-01", []byte(`{"properties":{"networkType":"Static","target":"10.0.0.50"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating iscsi path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/iscsiPaths?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing iscsi paths, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterPath+"?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"clusterSize":4}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, clusterPath+"/listZones?api-version=2025-09-01", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing cluster zones, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, clusterPath+"/hosts?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing hosts, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterPath+"/datastores/ds-a?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"netAppVolumeId":"vol-a"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating datastore, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, clusterPath+"/placementPolicies/policy-a?api-version=2025-09-01", []byte(`{"location":"eastus","properties":{"affinityType":"VmVm"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating placement policy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, clusterPath+"/placementPolicies/policy-a?api-version=2025-09-01", []byte(`{"tags":{"owner":"stackyard"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching placement policy, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, clusterPath+"/virtualMachines?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing cluster virtual machines, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPost, clusterPath+"/virtualMachines/vm-1/restrictMovement?api-version=2025-09-01", []byte(`{"restrictMovement":true}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 restricting movement, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/workloadNetworks/default?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting workload network root, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/workloadNetworks/default/dnsZones/corp.local?api-version=2025-09-01", []byte(`{"properties":{"domain":"corp.local"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating dns zone, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath+"/workloadNetworks/default/vmGroups/apps?api-version=2025-09-01", []byte(`{"properties":{"displayName":"apps","members":["vm-1"]}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 creating vm group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath+"/workloadNetworks/default/virtualMachines/vm-1?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting workload network vm, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, privateCloudPath+"?api-version=2025-09-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureAVSValidationAndNotFound(t *testing.T) {
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

	privateCloudPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-avs/providers/Microsoft.AVS/privateClouds/pc-a?api-version=2025-09-01"

	resp := providerContractRequest(t, ts, http.MethodPut, privateCloudPath, []byte(`{"properties":{"networkBlock":"10.0.0.0/22"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" || !strings.Contains(payload["message"].(string), "location") {
		t.Fatalf("unexpected validation payload: %#v", payload)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, privateCloudPath, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, privateCloudPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing private cloud, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	notFound := providerContractJSONMap(t, resp)
	if notFound["error"] != "NotFound" {
		t.Fatalf("unexpected not found payload: %#v", notFound)
	}
}

func TestAzureAVSInvalidAPIVersionReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.AVS/operations?api-version=", nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	if payload["error"] != "InvalidRequest" || payload["provider"] != providerAzure {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestAzureAVSListPayloadsAreJSON(t *testing.T) {
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
	}

	paths := []string{
		"/azure/providers/Microsoft.AVS/operations?api-version=2025-09-01",
		"/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.AVS/locations/eastus/checkQuotaAvailability?api-version=2025-09-01",
	}

	for _, path := range paths {
		method := http.MethodGet
		var body []byte
		if strings.Contains(path, "checkQuotaAvailability") {
			method = http.MethodPost
			body = []byte(`{}`)
		}
		resp := providerContractRequest(t, ts, method, path, body, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		if !json.Valid(providerContractBody(t, resp)) {
			t.Fatalf("expected valid JSON for %s", path)
		}
	}
}
