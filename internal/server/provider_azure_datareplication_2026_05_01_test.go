package server

import (
	"net/http"
	"testing"
)

const azureDREPTestAPI = "?api-version=2026-05-01"

func TestAzureDataReplication20260501Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dr")
	group := sub + "/resourceGroups/rg-dr/providers/Microsoft.DataReplication"
	vault := group + "/replicationVaults/v1"
	custom := func(kind string) string { return `{"properties":{"customProperties":{"instanceType":"` + kind + `"}}}` }

	azureResourcesExpectStatus(t, do(http.MethodPut, vault+azureDREPTestAPI, `{"location":"westus2","properties":{"vaultType":"DisasterRecovery"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/replicationFabrics/f1"+azureDREPTestAPI, `{"location":"westus2","properties":{"customProperties":{"instanceType":"FabricModelCustomProperties"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+"/replicationPolicies/p1"+azureDREPTestAPI, custom("PolicyModelCustomProperties")), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+"/replicationExtensions/e1"+azureDREPTestAPI, custom("ReplicationExtensionModelCustomProperties")), http.StatusCreated)

	item := vault + "/protectedItems/i1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, item+azureDREPTestAPI, `{"properties":{"policyName":"missing","replicationExtensionName":"e1","customProperties":{"instanceType":"X"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, item+azureDREPTestAPI, `{"properties":{"policyName":"p1","replicationExtensionName":"e1","customProperties":{"instanceType":"X"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, item+"/plannedFailover"+azureDREPTestAPI, `{"properties":{"customProperties":{"instanceType":"X"}}}`), http.StatusOK)

	jobs := providerContractJSONMap(t, do(http.MethodGet, vault+"/jobs"+azureDREPTestAPI, ""))["value"].([]any)
	if len(jobs) != 2 {
		t.Fatalf("expected protection and failover jobs, got %d", len(jobs))
	}
	jobName := jobs[0].(map[string]any)["name"].(string)
	azureResourcesExpectStatus(t, do(http.MethodGet, vault+"/jobs/"+jobName+azureDREPTestAPI, ""), http.StatusOK)
	if status := providerContractJSONMap(t, do(http.MethodGet, group+"/operations/"+jobName+azureDREPTestAPI, ""))["status"]; status != "Succeeded" {
		t.Fatalf("operation status %v", status)
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, group+"/locations/westus2/operationResults/"+jobName+azureDREPTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, group+"/operations/missing"+azureDREPTestAPI, ""), http.StatusNotFound)
	if n := len(providerContractJSONMap(t, do(http.MethodGet, vault+"/events"+azureDREPTestAPI, ""))["value"].([]any)); n != 2 {
		t.Fatalf("expected 2 events, got %d", n)
	}
	points := providerContractJSONMap(t, do(http.MethodGet, item+"/recoveryPoints"+azureDREPTestAPI, ""))["value"].([]any)
	if len(points) != 1 {
		t.Fatalf("expected 1 recovery point, got %d", len(points))
	}

	azureResourcesExpectStatus(t, do(http.MethodPut, vault+"/privateEndpointConnectionProxies/pe1"+azureDREPTestAPI, `{"properties":{"groupId":"vault"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodGet, vault+"/privateEndpointConnections/pe1"+azureDREPTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+"/privateEndpointConnections/other"+azureDREPTestAPI, `{"properties":{}}`), http.StatusConflict)

	azureResourcesExpectStatus(t, do(http.MethodDelete, vault+"/replicationPolicies/p1"+azureDREPTestAPI, ""), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodDelete, item+azureDREPTestAPI, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodDelete, vault+"/replicationPolicies/p1"+azureDREPTestAPI, ""), http.StatusAccepted)

	resp := do(http.MethodPost, group+"/deployments/d1/preflight"+azureDREPTestAPI, `{"resources":[{"name":"v2","type":"Microsoft.DataReplication/replicationVaults","location":"westus2"}]}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, group+"/deployments/d1/preflight"+azureDREPTestAPI, `{"resources":[{"name":"x","type":"Microsoft.DataReplication/widgets"}]}`), http.StatusBadRequest, "InvalidResourceType")
}

func TestAzureDataReplication20260501Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dr")
	group := sub + "/resourceGroups/rg-dr/providers/Microsoft.DataReplication"
	vault := group + "/replicationVaults/v1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+azureDREPTestAPI, `{"location":"westus2","properties":{"vaultType":"Backup"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+azureDREPTestAPI, `{"location":"westus2","properties":{}}`), http.StatusCreated)
	for _, tc := range []struct{ name, path, body string }{
		{"fabric custom properties", group + "/replicationFabrics/f", `{"location":"westus2","properties":{}}`},
		{"agent identity", group + "/replicationFabrics/f/fabricAgents/a", `{"properties":{"machineId":"m","machineName":"m"}}`},
		{"alert emails", vault + "/alertSettings/a", `{"properties":{"sendToOwners":true,"customEmailAddresses":["nope"]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := do(http.MethodPut, tc.path+azureDREPTestAPI, tc.body)
			if resp.StatusCode != http.StatusBadRequest && resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status %d, want 400 or 404", resp.StatusCode)
			}
		})
	}
	resp := do(http.MethodPost, sub+"/providers/Microsoft.DataReplication/locations/westus2/checkNameAvailability"+azureDREPTestAPI, `{"name":"v1","type":"Microsoft.DataReplication/replicationVaults"}`)
	if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != false {
		t.Fatalf("vault name should be taken, got %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vault+"/widgets"+azureDREPTestAPI, ""), http.StatusNotFound, "InvalidResourceType")
}
