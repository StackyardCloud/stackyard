package server

import (
	"net/http"
	"testing"
)

func TestAzureKustoLifecycle(t *testing.T) {
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
	resourceGroup := "rg-kusto"
	clusterName := "cluster-a"
	databaseName := "db-a"
	base := "/azure/subscriptions/" + subscriptionID
	clusterPath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "?api-version=2025-02-14"
	databasePath := base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/databases/" + databaseName + "?api-version=2025-02-14"

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Kusto/operations?api-version=2025-02-14", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, clusterPath, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"sku":{"name":"Standard_D13_v2","tier":"Standard"},
		"identity":{"type":"SystemAssigned"},
		"zones":["1","2"],
		"properties":{"optimizedAutoscale":{"isEnabled":true}}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if resp.Header.Get("Azure-AsyncOperation") == "" {
		t.Fatalf("expected Azure-AsyncOperation header on cluster create")
	}
	createdCluster := providerContractJSONMap(t, resp)
	if createdCluster["type"] != "Microsoft.Kusto/clusters" {
		t.Fatalf("unexpected cluster payload: %#v", createdCluster)
	}

	asyncPath := resp.Header.Get("Azure-AsyncOperation")
	resp = providerContractRequest(t, ts, http.MethodGet, asyncPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 polling operation result, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	op := providerContractJSONMap(t, resp)
	if op["status"] != "Succeeded" {
		t.Fatalf("unexpected operation result payload: %#v", op)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, clusterPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	cluster := providerContractJSONMap(t, resp)
	properties, ok := cluster["properties"].(map[string]any)
	if !ok || properties["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected cluster properties: %#v", cluster)
	}

	resp = providerContractRequest(t, ts, http.MethodHead, clusterPath, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading cluster, got %d", resp.StatusCode)
	}

	for _, path := range []string{
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters?api-version=2025-02-14",
		base + "/providers/Microsoft.Kusto/clusters?api-version=2025-02-14",
		base + "/providers/Microsoft.Kusto/skus?api-version=2025-02-14",
		base + "/providers/Microsoft.Kusto/locations/eastus/skus?api-version=2025-02-14",
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/listSkus?api-version=2025-02-14",
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/start?api-version=2025-02-14", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 starting cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	started := providerContractJSONMap(t, resp)
	if props, ok := started["properties"].(map[string]any); !ok || props["state"] != "Running" {
		t.Fatalf("expected started cluster state, got %#v", started)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/diagnoseVirtualNetwork?api-version=2025-02-14", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 diagnosing virtual network, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, databasePath, []byte(`{
		"location":"eastus",
		"kind":"ReadWrite",
		"properties":{"softDeletePeriod":"P365D","hotCachePeriod":"P7D"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating database, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	database := providerContractJSONMap(t, resp)
	if database["type"] != "Microsoft.Kusto/clusters/databases" {
		t.Fatalf("unexpected database payload: %#v", database)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/databases/"+databaseName+"/inviteFollower?api-version=2025-02-14", []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 inviting follower, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	invite := providerContractJSONMap(t, resp)
	if invite["invitationToken"] == "" {
		t.Fatalf("expected invitation token, got %#v", invite)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/databases/"+databaseName+"/dataConnections/conn-a?api-version=2025-02-14", []byte(`{
		"kind":"EventHub",
		"properties":{"consumerGroup":"$Default"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating data connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/databases/"+databaseName+"/scripts/script-a?api-version=2025-02-14", []byte(`{
		"properties":{"scriptUrl":"https://example.invalid/script.kql"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating script, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/principalAssignments/pa-a?api-version=2025-02-14", []byte(`{
		"properties":{"principalId":"11111111-1111-1111-1111-111111111111","role":"AllDatabasesAdmin"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating cluster principal assignment, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.Kusto/clusters/"+clusterName+"/managedPrivateEndpoints/mpe-a?api-version=2025-02-14", []byte(`{
		"properties":{"groupId":"cluster","privateLinkResourceId":"/subscriptions/000/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/sa"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating managed private endpoint, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/databases?api-version=2025-02-14",
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/databases/" + databaseName + "/dataConnections?api-version=2025-02-14",
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/databases/" + databaseName + "/scripts?api-version=2025-02-14",
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/privateLinkResources?api-version=2025-02-14",
		base + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Kusto/clusters/" + clusterName + "/privateLinkResources/cluster?api-version=2025-02-14",
	} {
		resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, clusterPath, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureKustoValidationAndNotFound(t *testing.T) {
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

	path := "/azure/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-kusto/providers/Microsoft.Kusto/clusters/cluster-a?api-version=2025-02-14"

	resp := providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"properties":{}}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, path, []byte(`{"location":`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 getting missing cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 deleting missing cluster, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.Kusto/operations?api-version=", nil, headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid api-version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}
