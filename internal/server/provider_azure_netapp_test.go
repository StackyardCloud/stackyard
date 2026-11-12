package server

import (
	"net/http"
	"testing"
)

func TestAzureNetAppLifecycle(t *testing.T) {
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
	resourceGroup := "rg-netapp"
	accountName := "accounta"
	poolName := "poola"
	volumeName := "volumea"
	base := "/azure/subscriptions/" + subscriptionID
	accountPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.NetApp/netAppAccounts/" + accountName + "?api-version=" + azureNetAppAPIVersion
	poolBase := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.NetApp/netAppAccounts/" + accountName + "/capacityPools/" + poolName
	poolPath := poolBase + "?api-version=" + azureNetAppAPIVersion
	volumeBase := poolBase + "/volumes/" + volumeName
	volumePath := volumeBase + "?api-version=" + azureNetAppAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.NetApp/operations?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.NetApp/locations/eastus/checkNameAvailability?api-version="+azureNetAppAPIVersion, []byte(`{"name":"`+accountName+`"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, accountPath, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{"nfsV4IDDomain":"example.local"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	account := providerContractJSONMap(t, resp)
	if account["type"] != "Microsoft.NetApp/netAppAccounts" {
		t.Fatalf("unexpected account payload: %#v", account)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, poolPath, []byte(`{
		"location":"eastus",
		"properties":{"size":4398046511104,"serviceLevel":"Premium","qosType":"Auto"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating pool, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, volumePath, []byte(`{
		"location":"eastus",
		"properties":{
			"creationToken":"vol-a",
			"serviceLevel":"Premium",
			"usageThreshold":1073741824,
			"protocolTypes":["NFSv4.1"],
			"subnetId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-network/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/netapp"
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating volume, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, volumePath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading volume, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.NetApp/netAppAccounts?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing accounts by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.NetApp/netAppAccounts?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing accounts by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, poolBase+"/volumes?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing volumes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	volumes := providerContractJSONMap(t, resp)
	if values, ok := volumes["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one volume, got %#v", volumes)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, volumeBase+"/snapshots?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing snapshots, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	snapshots := providerContractJSONMap(t, resp)
	if values, ok := snapshots["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected seeded snapshots, got %#v", snapshots)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.NetApp/netAppAccounts/"+accountName+"/backups?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing account backups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	backups := providerContractJSONMap(t, resp)
	if values, ok := backups["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected seeded backups, got %#v", backups)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.NetApp/netAppAccounts/"+accountName+"/backupVaults/vaultb?api-version="+azureNetAppAPIVersion, []byte(`{"location":"eastus","properties":{"vaultState":"Ready"}}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating backup vault, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.NetApp/netAppAccounts/"+accountName+"/backupVaults/vaultb/backups?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing vault backups, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	vaultBackups := providerContractJSONMap(t, resp)
	if values, ok := vaultBackups["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected seeded vault backups, got %#v", vaultBackups)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, volumePath, []byte(`{"tags":{"env":"stage"},"properties":{"serviceLevel":"Ultra"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching volume, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.NetApp/locations/eastus/quotaLimits?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing quota limits, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.NetApp/locations/eastus/regionInfos?api-version="+azureNetAppAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing region infos, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, accountPath, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureNetAppValidationAndNotFound(t *testing.T) {
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

	accountPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-netapp/providers/Microsoft.NetApp/netAppAccounts/accounta?api-version=" + azureNetAppAPIVersion
	poolPath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-netapp/providers/Microsoft.NetApp/netAppAccounts/accounta/capacityPools/poola?api-version=" + azureNetAppAPIVersion
	volumePath := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-netapp/providers/Microsoft.NetApp/netAppAccounts/accounta/capacityPools/poola/volumes/volumea?api-version=" + azureNetAppAPIVersion

	resp := providerContractRequest(t, ts, http.MethodPut, accountPath, []byte(`{"properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing account location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, poolPath, []byte(`{"location":"eastus","properties":{"serviceLevel":"Premium"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing pool size, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, volumePath, []byte(`{"location":"eastus","properties":{"serviceLevel":"Premium"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing volume fields, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, accountPath, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, accountPath, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 patching missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountPath, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, accountPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, "/azure/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.NetApp/locations/eastus/checkFilePathAvailability?api-version="+azureNetAppAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 checking empty file path, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.NetApp/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
