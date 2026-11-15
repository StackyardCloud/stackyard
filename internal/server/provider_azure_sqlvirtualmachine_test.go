package server

import (
	"net/http"
	"testing"
)

const azureSQLVirtualMachineTestAPIVersion = "?api-version=2023-10-01"

func TestAzureSQLVirtualMachineLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sqlvm")
	base := sub + "/resourceGroups/rg-sqlvm/providers/Microsoft.SqlVirtualMachine"
	vmID := sub + "/resourceGroups/rg-sqlvm/providers/Microsoft.Compute/virtualMachines/sql01"

	group := base + "/sqlVirtualMachineGroups/ag"
	resp := do(http.MethodPut, group+azureSQLVirtualMachineTestAPIVersion, `{"location":"eastus","properties":{"sqlImageOffer":"SQL2022-WS2022","sqlImageSku":"Enterprise","wsfcDomainProfile":{"domainFqdn":"corp.example.com","storageAccountUrl":"https://witness.blob.core.windows.net/","storageAccountPrimaryKey":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["wsfcDomainProfile"].(map[string]any)["storageAccountPrimaryKey"] != nil || props["scaleType"] != "HA" {
		t.Fatalf("unexpected group properties: %#v", props)
	}

	vm := base + "/sqlVirtualMachines/sql01"
	resp = do(http.MethodPut, vm+azureSQLVirtualMachineTestAPIVersion, `{"location":"eastus","properties":{"virtualMachineResourceId":"`+vmID+`","sqlServerLicenseType":"AHUB","sqlVirtualMachineGroupResourceId":"`+group+`","serverConfigurationsManagementSettings":{"sqlConnectivityUpdateSettings":{"connectivityType":"PRIVATE","port":1433,"sqlAuthUpdateUserName":"sqladmin","sqlAuthUpdatePassword":"secret"}},"autoBackupSettings":{"enable":true,"storageAccessKey":"secret","password":"secret"},"wsfcDomainCredentials":{"clusterBootstrapAccountPassword":"secret"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	connectivity := props["serverConfigurationsManagementSettings"].(map[string]any)["sqlConnectivityUpdateSettings"].(map[string]any)
	backup := props["autoBackupSettings"].(map[string]any)
	if connectivity["sqlAuthUpdatePassword"] != nil || connectivity["sqlAuthUpdateUserName"] != "sqladmin" || backup["storageAccessKey"] != nil || backup["password"] != nil || props["wsfcDomainCredentials"] != nil {
		t.Fatalf("expected secrets stripped, got %#v", props)
	}
	if props["sqlManagement"] != "Full" || props["sqlServerLicenseType"] != "AHUB" {
		t.Fatalf("unexpected defaults: %#v", props)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, vm+"/redeploy"+azureSQLVirtualMachineTestAPIVersion, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodPost, vm+"/startAssessment"+azureSQLVirtualMachineTestAPIVersion, ""), http.StatusAccepted)

	azureResourcesExpectStatus(t, do(http.MethodPut, group+"/availabilityGroupListeners/agl"+azureSQLVirtualMachineTestAPIVersion, `{"properties":{"availabilityGroupName":"ag1","port":1433,"loadBalancerConfigurations":[{"privateIpAddress":{"ipAddress":"10.0.0.10"},"probePort":59999}]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodDelete, group+azureSQLVirtualMachineTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, group+"/availabilityGroupListeners/agl"+azureSQLVirtualMachineTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureSQLVirtualMachineValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-sqlvm")
	base := sub + "/resourceGroups/rg-sqlvm/providers/Microsoft.SqlVirtualMachine"
	vmID := sub + "/resourceGroups/rg-sqlvm/providers/Microsoft.Compute/virtualMachines/sql01"
	azureResourcesExpectStatus(t, do(http.MethodPut, base+"/sqlVirtualMachineGroups/ag"+azureSQLVirtualMachineTestAPIVersion, `{"location":"eastus","properties":{"sqlImageOffer":"SQL2022-WS2022","sqlImageSku":"Developer"}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"no vm", base + "/sqlVirtualMachines/sql01", `{"location":"eastus","properties":{}}`},
		{"bad license", base + "/sqlVirtualMachines/sql01", `{"location":"eastus","properties":{"virtualMachineResourceId":"` + vmID + `","sqlServerLicenseType":"Free"}}`},
		{"bad group id", base + "/sqlVirtualMachines/sql01", `{"location":"eastus","properties":{"virtualMachineResourceId":"` + vmID + `","sqlVirtualMachineGroupResourceId":"/subscriptions/x"}}`},
		{"group offer", base + "/sqlVirtualMachineGroups/g2", `{"location":"eastus","properties":{"sqlImageSku":"Developer"}}`},
		{"group sku", base + "/sqlVirtualMachineGroups/g2", `{"location":"eastus","properties":{"sqlImageOffer":"SQL2022-WS2022","sqlImageSku":"Express"}}`},
		{"listener name", base + "/sqlVirtualMachineGroups/ag/availabilityGroupListeners/l", `{"properties":{"loadBalancerConfigurations":[{}]}}`},
		{"listener both configs", base + "/sqlVirtualMachineGroups/ag/availabilityGroupListeners/l", `{"properties":{"availabilityGroupName":"ag1","loadBalancerConfigurations":[{}],"multiSubnetIpConfigurations":[{}]}}`},
		{"listener no config", base + "/sqlVirtualMachineGroups/ag/availabilityGroupListeners/l", `{"properties":{"availabilityGroupName":"ag1"}}`},
		{"listener port", base + "/sqlVirtualMachineGroups/ag/availabilityGroupListeners/l", `{"properties":{"availabilityGroupName":"ag1","port":70000,"loadBalancerConfigurations":[{}]}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureSQLVirtualMachineTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
