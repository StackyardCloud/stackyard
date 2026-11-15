package server

import (
	"net/http"
	"strings"
)

// Microsoft.SqlVirtualMachine registrations of SQL Server on Azure VMs, the
// availability group clusters they join and the clusters' listeners, served
// by the shared ARM handler. No SQL IaaS agent runs.
func init() {
	const ns = "Microsoft.SqlVirtualMachine"
	accepted := func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil }
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "sqlVirtualMachines",
		ReadOnlyProperties: azureSQLVirtualMachineProperties,
		Validate:           azureSQLVirtualMachineValidate,
		Actions:            map[string]azureARMAction{"redeploy": accepted, "startassessment": accepted},
		WriteOnlyProperties: []string{
			"serverConfigurationsManagementSettings.sqlConnectivityUpdateSettings.sqlAuthUpdatePassword",
			"keyVaultCredentialSettings.servicePrincipalSecret",
			"autoBackupSettings.storageAccessKey",
			"autoBackupSettings.password",
			"wsfcDomainCredentials",
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          ns,
		Type:               "sqlVirtualMachineGroups",
		ReadOnlyProperties: azureSQLVirtualMachineGroupProperties,
		Validate:           azureSQLVirtualMachineGroupValidate,
		// The witness storage account key is never returned.
		WriteOnlyProperties: []string{"wsfcDomainProfile.storageAccountPrimaryKey"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "sqlVirtualMachineGroups/availabilityGroupListeners", Proxy: true,
		Validate: azureSQLVirtualMachineListenerValidate,
	})
}

var azureSQLVirtualMachineEnums = map[string][]string{
	"sqlServerLicenseType": {"PAYG", "AHUB", "DR"},
	"sqlManagement":        {"Full", "LightWeight", "NoAgent"},
	"sqlImageSku":          {"Developer", "Express", "Standard", "Enterprise", "Web"},
	"leastPrivilegeMode":   {"Enabled", "NotSet"},
}

func azureSQLVirtualMachineValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.Contains(strings.ToLower(azureARMString(props["virtualMachineResourceId"])), "/providers/microsoft.compute/virtualmachines/") {
		return "properties.virtualMachineResourceId must be a virtual machine ID."
	}
	for key, values := range azureSQLVirtualMachineEnums {
		if msg := azureEnumProperty(key, values...)(payload); msg != "" {
			return msg
		}
	}
	if group := azureARMString(props["sqlVirtualMachineGroupResourceId"]); group != "" && !strings.Contains(strings.ToLower(group), "/providers/microsoft.sqlvirtualmachine/sqlvirtualmachinegroups/") {
		return "properties.sqlVirtualMachineGroupResourceId must be a SQL virtual machine group ID."
	}
	return ""
}

// azureSQLVirtualMachineProperties defaults the license and management mode
// and reports the VM's SQL image, which the SQL IaaS agent would detect.
func azureSQLVirtualMachineProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{}
	for key, value := range map[string]string{"sqlServerLicenseType": "PAYG", "sqlManagement": "Full", "sqlImageOffer": "SQL2022-WS2022", "sqlImageSku": "Developer"} {
		if _, present := rec.Properties[key]; !present {
			props[key] = value
		}
	}
	return props
}

func azureSQLVirtualMachineGroupValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["sqlImageOffer"])) == "" {
		return "properties.sqlImageOffer is required."
	}
	return azureEnumProperty("sqlImageSku", "Developer", "Enterprise")(payload)
}

func azureSQLVirtualMachineGroupProperties(*azureARMResource) map[string]any {
	return map[string]any{"scaleType": "HA", "clusterManagerType": "WSFC", "clusterConfiguration": "Domainful"}
}

func azureSQLVirtualMachineListenerValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["availabilityGroupName"])) == "" {
		return "properties.availabilityGroupName is required."
	}
	balancers, _ := props["loadBalancerConfigurations"].([]any)
	subnets, _ := props["multiSubnetIpConfigurations"].([]any)
	if (len(balancers) == 0) == (len(subnets) == 0) {
		return "Set exactly one of properties.loadBalancerConfigurations and properties.multiSubnetIpConfigurations."
	}
	if raw, present := props["port"]; present {
		if port, ok := raw.(float64); !ok || port != float64(int(port)) || port < 1 || port > 65535 {
			return "properties.port must be an integer from 1 to 65535."
		}
	}
	return ""
}
