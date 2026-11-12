package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureBatchManagementLifecycle(t *testing.T) {
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
	resourceGroup := "rg-batch"
	location := "eastus"
	accountName := "batchacct123"
	base := "/azure/subscriptions/" + subscriptionID
	accountPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Batch/batchAccounts/" + accountName + "?api-version=" + azureBatchManagementAPIVersion

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Batch/operations?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing batch operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	ops := providerContractJSONMap(t, resp)
	if values, ok := ops["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", ops)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.Batch/locations/"+location+"/checkNameAvailability?api-version="+azureBatchManagementAPIVersion, []byte(`{
		"name":"`+accountName+`",
		"type":"Microsoft.Batch/batchAccounts"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking batch account name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	nameCheck := providerContractJSONMap(t, resp)
	if available, ok := nameCheck["nameAvailable"].(bool); !ok || !available {
		t.Fatalf("expected available batch account name, got %#v", nameCheck)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Batch/locations/"+location+"/quotas?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting quotas, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	quotas := providerContractJSONMap(t, resp)
	if values, ok := quotas["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected quota entries, got %#v", quotas)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Batch/locations/"+location+"/supportedVirtualMachineSkus?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing supported VM SKUs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	skus := providerContractJSONMap(t, resp)
	if values, ok := skus["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected supported VM SKUs, got %#v", skus)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, accountPath, []byte(`{
		"location":"`+location+`",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"dev"},
		"properties":{
			"publicNetworkAccess":"Enabled",
			"allowedAuthenticationModes":["SharedKey","AAD"]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating batch account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	account := providerContractJSONMap(t, resp)
	if account["type"] != "Microsoft.Batch/batchAccounts" {
		t.Fatalf("unexpected batch account payload: %#v", account)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting batch account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, accountPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading batch account, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Batch/batchAccounts?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing batch accounts by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	byGroup := providerContractJSONMap(t, resp)
	if values, ok := byGroup["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one batch account by resource group, got %#v", byGroup)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/providers/Microsoft.Batch/batchAccounts?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing batch accounts by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bySubscription := providerContractJSONMap(t, resp)
	if values, ok := bySubscription["value"].([]any); !ok || len(values) != 1 {
		t.Fatalf("expected one batch account by subscription, got %#v", bySubscription)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/listKeys?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing batch account keys, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	keys := providerContractJSONMap(t, resp)
	if keys["primary"] == "" || keys["secondary"] == "" {
		t.Fatalf("expected primary and secondary keys, got %#v", keys)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/regenerateKeys?api-version="+azureBatchManagementAPIVersion, []byte(`{"keyName":"Primary"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 regenerating batch account keys, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/syncAutoStorageKeys?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 syncing auto storage keys, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	applicationPath := strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion) + "/applications/appa?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, applicationPath, []byte(`{
		"properties":{"allowUpdates":true},
		"tags":{"kind":"sample"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating application, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/applications?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing applications, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, applicationPath, []byte(`{"properties":{"displayName":"sample-app"}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching application, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	packagePath := strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion) + "/applications/appa/versions/v1?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, packagePath, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating application package, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	pkg := providerContractJSONMap(t, resp)
	pkgProps, ok := pkg["properties"].(map[string]any)
	if !ok || pkgProps["state"] != "Pending" {
		t.Fatalf("expected pending application package, got %#v", pkg)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(packagePath, "?api-version="+azureBatchManagementAPIVersion)+"/activate?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 activating application package, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/applications/appa/versions?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing application packages, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	poolPath := strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion) + "/pools/poola?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, poolPath, []byte(`{
		"properties":{
			"vmSize":"Standard_DS2_v2",
			"deploymentConfiguration":{"virtualMachineConfiguration":{"imageReference":{"publisher":"Canonical"}}}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating pool, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, poolPath, []byte(`{"properties":{"scaleSettings":{"fixedScale":{"targetDedicatedNodes":2}}}}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching pool, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(poolPath, "?api-version="+azureBatchManagementAPIVersion)+"/disableAutoScale?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 disabling pool autoscale, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(poolPath, "?api-version="+azureBatchManagementAPIVersion)+"/stopResize?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 stopping pool resize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/pools?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing pools, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	pecPath := strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion) + "/privateEndpointConnections/peca?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPatch, pecPath, []byte(`{
		"properties":{
			"privateEndpoint":{"id":"/subscriptions/`+subscriptionID+`/resourceGroups/rg-net/providers/Microsoft.Network/privateEndpoints/pe-a"},
			"groupIds":["batchAccount"]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating private endpoint connection via patch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/privateEndpointConnections?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private endpoint connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/privateLinkResources?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	privateLinks := providerContractJSONMap(t, resp)
	if values, ok := privateLinks["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected private link resources, got %#v", privateLinks)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/networkSecurityPerimeterConfigurations/default?api-version="+azureBatchManagementAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting default network security perimeter configuration, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(accountPath, "?api-version="+azureBatchManagementAPIVersion)+"/networkSecurityPerimeterConfigurations/default/reconcile?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 reconciling network security perimeter configuration, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, accountPath, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting batch account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureBatchManagementValidation(t *testing.T) {
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
	resourceGroup := "rg-batch-validation"
	base := "/azure/subscriptions/" + subscriptionID

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Batch/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid batch management api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/providers/Microsoft.Batch/locations/eastus/checkNameAvailability?api-version="+azureBatchManagementAPIVersion, []byte(`{
		"name":"bad",
		"type":"Microsoft.Batch/invalid"
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid check name availability type, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	accountPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Batch/batchAccounts/invalid-name!?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, accountPath, []byte(`{"location":"eastus"}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid batch account name, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	validAccountPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Batch/batchAccounts/batchacctval1?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, validAccountPath, []byte(`{"tags":{"env":"test"}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 creating batch account without location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, validAccountPath, []byte(`{
		"location":"eastus",
		"properties":{"allowedAuthenticationModes":["AAD"]}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating validation batch account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, strings.TrimSuffix(validAccountPath, "?api-version="+azureBatchManagementAPIVersion)+"/listKeys?api-version="+azureBatchManagementAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 listing keys when shared key auth is disabled, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	missingPackagePath := strings.TrimSuffix(validAccountPath, "?api-version="+azureBatchManagementAPIVersion) + "/applications/missing/versions/v1?api-version=" + azureBatchManagementAPIVersion
	resp = providerContractRequest(t, ts, http.MethodPut, missingPackagePath, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 creating package for missing application, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
