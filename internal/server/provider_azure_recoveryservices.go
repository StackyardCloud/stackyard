package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
)

const azureRecoveryServicesNamespace = "Microsoft.RecoveryServices"

// Microsoft.RecoveryServices vaults, served by the shared ARM handler. Backup
// and Site Recovery resources live under the vault; see
// provider_azure_recoveryservicesbackup.go and
// provider_azure_recoveryservicessiterecovery.go.
func init() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace:          azureRecoveryServicesNamespace,
		Type:               "vaults",
		ReadOnlyProperties: azureRecoveryServicesVaultProperties,
		ValidateName:       azureRecoveryServicesValidateName,
		Validate:           azureRecoveryServicesValidate,
		DefaultChildren:    azureBackupDefaultChildren,
	})
	registerAzureARMGroupAction(azureRecoveryServicesNamespace, "checkNameAvailability", azureRecoveryServicesCheckName)
}

var (
	azureRecoveryServicesNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{1,49}$`)
	azureRecoveryServicesSKUs        = []string{"Standard", "RS0"}
	azureRecoveryServicesRedundancy  = []string{"LocallyRedundant", "GeoRedundant", "ZoneRedundant"}
)

func azureRecoveryServicesValidateName(name string) string {
	if !azureRecoveryServicesNamePattern.MatchString(name) {
		return "Vault names must be 2-50 letters, digits or hyphens and start with a letter."
	}
	return ""
}

func azureRecoveryServicesValidate(payload map[string]any) string {
	if !slices.Contains(azureRecoveryServicesSKUs, azureARMString(azureARMObject(payload["sku"])["name"])) {
		return "sku.name must be Standard or RS0."
	}
	redundancy := azureARMObject(azureARMObject(payload["properties"])["redundancySettings"])
	if raw, present := redundancy["standardTierStorageRedundancy"]; present && !slices.Contains(azureRecoveryServicesRedundancy, azureARMString(raw)) {
		return "properties.redundancySettings.standardTierStorageRedundancy must be one of " + strings.Join(azureRecoveryServicesRedundancy, ", ") + "."
	}
	return ""
}

// azureRecoveryServicesVaultProperties sets the private endpoint states and
// fills in network access and redundancy when the client leaves them out.
func azureRecoveryServicesVaultProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{
		"privateEndpointStateForBackup":       "None",
		"privateEndpointStateForSiteRecovery": "None",
		"backupStorageVersion":                "V2",
	}
	if _, present := rec.Properties["publicNetworkAccess"]; !present {
		props["publicNetworkAccess"] = "Enabled"
	}
	redundancy := azureARMMerge(map[string]any{"standardTierStorageRedundancy": "GeoRedundant", "crossRegionRestore": "Disabled"},
		azureARMObject(rec.Properties["redundancySettings"]))
	props["redundancySettings"] = redundancy
	return props
}

// azureRecoveryServicesCheckName compares vault names within the resource
// group, as the location-scoped check does.
func azureRecoveryServicesCheckName(s *Server, sub, rg string, payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	if name == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "name is required")
	}
	if typ := azureARMString(payload["type"]); !strings.EqualFold(typ, azureRecoveryServicesNamespace+"/vaults") {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.RecoveryServices/Vaults")
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azureRecoveryServicesNamespace+"/vaults") && strings.EqualFold(rec.SubscriptionID, sub) &&
			strings.EqualFold(rec.ResourceGroup, rg) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		return http.StatusOK, map[string]any{"nameAvailable": false, "reason": "AlreadyExists", "message": "Resource '" + name + "' already exists in resource group '" + rg + "'."}
	}
	return http.StatusOK, map[string]any{"nameAvailable": true}
}
