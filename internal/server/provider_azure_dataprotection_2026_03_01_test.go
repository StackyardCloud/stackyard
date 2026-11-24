package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureDPPTestAPI = "?api-version=2026-03-01"

func TestAzureDataProtection20260301Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dpp")
	vault := sub + "/resourceGroups/rg-dpp/providers/Microsoft.DataProtection/backupVaults/v1"
	policy := vault + "/backupPolicies/p1"
	instance := vault + "/backupInstances/bi1"
	policyID := strings.TrimPrefix(policy, "/azure")

	azureResourcesExpectStatus(t, do(http.MethodPut, vault+azureDPPTestAPI, `{"location":"westus","properties":{"storageSettings":[{"type":"LocallyRedundant","datastoreType":"VaultStore"}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, policy+azureDPPTestAPI, `{"properties":{"objectType":"BackupPolicy","datasourceTypes":["Microsoft.Compute/disks"],"policyRules":[]}}`), http.StatusCreated)
	instanceBody := func(datasourceType string) string {
		return `{"properties":{"objectType":"BackupInstance","friendlyName":"disk","dataSourceInfo":{"datasourceType":"` + datasourceType + `","resourceID":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Compute/disks/d"},"policyInfo":{"policyId":"` + policyID + `"}}}`
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, instance+azureDPPTestAPI, instanceBody("Microsoft.Storage/storageAccounts")), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, instance+azureDPPTestAPI, instanceBody("Microsoft.Compute/disks")), http.StatusCreated)

	azureResourcesExpectErrorCode(t, do(http.MethodPost, instance+"/backup"+azureDPPTestAPI, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	jobID := providerContractJSONMap(t, do(http.MethodPost, instance+"/backup"+azureDPPTestAPI, `{"backupRuleOptions":{"ruleName":"Default"}}`))["jobId"].(string)
	job := providerContractJSONMap(t, do(http.MethodGet, vault+"/backupJobs/"+jobID+azureDPPTestAPI, ""))
	if job["properties"].(map[string]any)["status"] != "Completed" {
		t.Fatalf("unexpected job %#v", job)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, instance+"/stopProtection"+azureDPPTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, instance+"/backup"+azureDPPTestAPI, `{"backupRuleOptions":{"ruleName":"Default"}}`), http.StatusConflict)

	resp := do(http.MethodPost, vault+"/exportBackupJobs"+azureDPPTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusAccepted)
	location := resp.Header.Get("Location")
	opID := location[strings.LastIndex(location, "/")+1 : strings.Index(location, "?")]
	if blob := providerContractJSONMap(t, do(http.MethodGet, vault+"/backupJobs/operations/"+opID+azureDPPTestAPI, ""))["blobUrl"]; blob == nil {
		t.Fatal("export result should have a blobUrl")
	}
	for _, path := range []string{
		sub + "/providers/Microsoft.DataProtection/locations/westus/operationStatus/" + opID,
		sub + "/resourceGroups/rg-dpp/providers/Microsoft.DataProtection/locations/westus/operationStatus/" + opID,
		vault + "/operationStatus/" + opID,
	} {
		if status := providerContractJSONMap(t, do(http.MethodGet, path+azureDPPTestAPI, ""))["status"]; status != "Succeeded" {
			t.Fatalf("%s: status %v", path, status)
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.DataProtection/locations/eastus/operationStatus/"+opID+azureDPPTestAPI, ""), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodGet, vault+"/operationStatus/missing"+azureDPPTestAPI, ""), http.StatusNotFound)

	azureResourcesExpectStatus(t, do(http.MethodDelete, policy+azureDPPTestAPI, ""), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodDelete, vault+azureDPPTestAPI, ""), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodDelete, instance+azureDPPTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, vault+azureDPPTestAPI, ""), http.StatusOK)

	deleted := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.DataProtection/locations/westus/deletedVaults"+azureDPPTestAPI, ""))["value"].([]any)
	if len(deleted) != 1 || deleted[0].(map[string]any)["properties"].(map[string]any)["originalBackupVaultName"] != "v1" {
		t.Fatalf("expected the deleted vault, got %#v", deleted)
	}
	name := deleted[0].(map[string]any)["name"].(string)
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.DataProtection/locations/westus/deletedVaults/"+name+azureDPPTestAPI, ""), http.StatusOK)
}

func TestAzureDataProtection20260301Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dpp")
	vault := sub + "/resourceGroups/rg-dpp/providers/Microsoft.DataProtection/backupVaults/v1"
	for _, body := range []string{
		`{"location":"westus","properties":{}}`,
		`{"location":"westus","properties":{"storageSettings":[{"type":"Everywhere","datastoreType":"VaultStore"}]}}`,
	} {
		azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+azureDPPTestAPI, body), http.StatusBadRequest, "InvalidRequestContent")
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+azureDPPTestAPI, `{"location":"westus","properties":{"storageSettings":[{"type":"GeoRedundant","datastoreType":"VaultStore"}]}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/backupPolicies/p"+azureDPPTestAPI, `{"properties":{"objectType":"BackupPolicy","datasourceTypes":[]}}`), http.StatusBadRequest, "InvalidRequestContent")
	resp := do(http.MethodPost, sub+"/resourceGroups/rg-dpp/providers/Microsoft.DataProtection/locations/westus/checkNameAvailability"+azureDPPTestAPI, `{"name":"v1","type":"Microsoft.DataProtection/BackupVaults"}`)
	if got := providerContractJSONMap(t, resp)["nameAvailable"]; got != false {
		t.Fatalf("vault name should be taken, got %v", got)
	}
	resp = do(http.MethodPost, sub+"/providers/Microsoft.DataProtection/locations/westus/checkFeatureSupport"+azureDPPTestAPI, `{"objectType":"FeatureValidationRequest","featureType":"DataSourceType"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/providers/Microsoft.DataProtection/locations/westus/checkFeatureSupport"+azureDPPTestAPI, `{}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodGet, vault+"/exportBackupJobs"+azureDPPTestAPI, ""), http.StatusMethodNotAllowed)
}
