package server

import (
	"slices"
	"strings"
)

// Azure Site Recovery resources under a Recovery Services vault: fabrics,
// their protection containers and container mappings, and replication
// policies. Provider-specific inputs come back as provider-specific details.
func init() {
	const ns = azureRecoveryServicesNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "vaults/replicationFabrics", Proxy: true,
		Validate: azureSiteRecoveryValidateFabric,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"friendlyName":       rec.Name,
				"health":             "Normal",
				"fabricType":         azureARMString(azureARMObject(rec.Properties["customDetails"])["instanceType"]),
				"internalIdentifier": azureARMDerivedKey(rec.ID, "fabric")[:32],
			}
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "vaults/replicationFabrics/replicationProtectionContainers", Proxy: true,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"friendlyName": rec.Name, "role": "Primary", "protectedItemCount": 0}
		},
		WriteOnlyProperties: []string{"providerSpecificInput"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "vaults/replicationFabrics/replicationProtectionContainers/replicationProtectionContainerMappings", Proxy: true,
		Validate: azureSiteRecoveryValidateMapping,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := azureSiteRecoveryDetails(rec)
			props["state"], props["health"] = "Paired", "Normal"
			return props
		},
		WriteOnlyProperties: []string{"providerSpecificInput"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "vaults/replicationPolicies", Proxy: true,
		Validate: func(payload map[string]any) string {
			if azureARMString(azureARMObject(azureARMObject(payload["properties"])["providerSpecificInput"])["instanceType"]) == "" {
				return "properties.providerSpecificInput.instanceType is required."
			}
			return ""
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := azureSiteRecoveryDetails(rec)
			props["friendlyName"] = rec.Name
			return props
		},
		WriteOnlyProperties: []string{"providerSpecificInput"},
	})
}

var azureSiteRecoveryFabricTypes = []string{"Azure", "HyperVSite", "VMM", "VMware", "VMwareV2", "InMageRcm", "AzureStackHCI"}

func azureSiteRecoveryValidateFabric(payload map[string]any) string {
	custom := azureARMObject(azureARMObject(payload["properties"])["customDetails"])
	if !slices.Contains(azureSiteRecoveryFabricTypes, azureARMString(custom["instanceType"])) {
		return "properties.customDetails.instanceType must be one of " + strings.Join(azureSiteRecoveryFabricTypes, ", ") + "."
	}
	if azureARMString(custom["instanceType"]) == "Azure" && strings.TrimSpace(azureARMString(custom["location"])) == "" {
		return "properties.customDetails.location is required for an Azure fabric."
	}
	return ""
}

func azureSiteRecoveryValidateMapping(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["policyId"])) == "" {
		return "properties.policyId is required."
	}
	if azureARMString(azureARMObject(props["providerSpecificInput"])["instanceType"]) == "" {
		return "properties.providerSpecificInput.instanceType is required."
	}
	return ""
}

// azureSiteRecoveryDetails echoes the provider-specific input of a PUT as
// providerSpecificDetails. A PATCH without input keeps the stored details.
func azureSiteRecoveryDetails(rec *azureARMResource) map[string]any {
	if input := azureARMObject(rec.Properties["providerSpecificInput"]); len(input) != 0 {
		return map[string]any{"providerSpecificDetails": input}
	}
	return map[string]any{}
}
