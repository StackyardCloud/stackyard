package server

import (
	"net/http"
	"slices"
	"strings"
)

// Azure Backup resources under a Recovery Services vault: the vault and
// storage configuration singletons, backup policies, and protection
// containers and protected items under the Azure backup fabric.
var (
	azureBackupConfigType = azureARMResourceType{
		Namespace: azureRecoveryServicesNamespace, Type: "vaults/backupconfig", Proxy: true,
		ValidateName: azureSingletonName("vaultconfig"),
		Validate:     azureEnumProperty("softDeleteFeatureState", "Enabled", "Disabled", "AlwaysON"),
	}
	azureBackupStorageConfigType = azureARMResourceType{
		Namespace: azureRecoveryServicesNamespace, Type: "vaults/backupstorageconfig", Proxy: true,
		ValidateName: azureSingletonName("vaultstorageconfig"),
		Validate:     azureEnumProperty("storageModelType", azureRecoveryServicesRedundancy...),
	}
	azureBackupPolicyType = azureARMResourceType{
		Namespace: azureRecoveryServicesNamespace, Type: "vaults/backupPolicies", Proxy: true,
		Validate: azureEnumProperty("backupManagementType", azureBackupManagementTypes...),
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			// ponytail: no item counts policies yet; count protected items when policy deletion checks them.
			return map[string]any{"protectedItemsCount": 0}
		},
	}
	azureBackupFabricType = azureARMResourceType{
		Namespace: azureRecoveryServicesNamespace, Type: "vaults/backupFabrics", Proxy: true,
		ValidateName: azureSingletonName("Azure"),
	}
	azureBackupContainerType = azureARMResourceType{
		Namespace: azureRecoveryServicesNamespace, Type: "vaults/backupFabrics/protectionContainers", Proxy: true,
		Validate: azureEnumProperty("backupManagementType", azureBackupManagementTypes...),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"registrationStatus": "Registered", "healthStatus": "Healthy", "friendlyName": rec.Name}
		},
	}
	azureBackupProtectedItemType = azureARMResourceType{
		Namespace: azureRecoveryServicesNamespace, Type: "vaults/backupFabrics/protectionContainers/protectedItems", Proxy: true,
		Validate:           azureBackupValidateProtectedItem,
		ReadOnlyProperties: azureBackupProtectedItemProperties,
		// An on-demand backup is accepted and never runs.
		Actions: map[string]azureARMAction{"backup": func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil }},
	}

	azureBackupManagementTypes = []string{"AzureIaasVM", "AzureSql", "AzureStorage", "AzureWorkload", "MAB", "DPM", "AzureBackupServer", "GenericProtectionPolicy"}
)

func init() {
	for _, typ := range []azureARMResourceType{azureBackupConfigType, azureBackupStorageConfigType, azureBackupPolicyType, azureBackupFabricType, azureBackupContainerType, azureBackupProtectedItemType} {
		registerAzureARMResourceType(typ)
	}
}

// azureBackupDefaultChildren are the configuration, built-in policies and
// Azure fabric every new vault has.
func azureBackupDefaultChildren(vault *azureARMResource) []*azureARMResource {
	daily := func(count int) map[string]any {
		return map[string]any{
			"retentionPolicyType": "LongTermRetentionPolicy",
			"dailySchedule": map[string]any{
				"retentionTimes":    []any{"2020-01-01T02:00:00Z"},
				"retentionDuration": map[string]any{"count": count, "durationType": "Days"},
			},
		}
	}
	return []*azureARMResource{
		azureARMChild(vault, azureBackupConfigType, "vaultconfig", map[string]any{
			"softDeleteFeatureState": "Enabled", "enhancedSecurityState": "Enabled", "isSoftDeleteFeatureStateEditable": true,
		}),
		azureARMChild(vault, azureBackupStorageConfigType, "vaultstorageconfig", map[string]any{
			"storageModelType": "GeoRedundant", "storageType": "GeoRedundant", "storageTypeState": "Unlocked", "crossRegionRestoreFlag": false,
		}),
		azureARMChild(vault, azureBackupPolicyType, "DefaultPolicy", map[string]any{
			"backupManagementType":          "AzureIaasVM",
			"instantRpRetentionRangeInDays": 2,
			"timeZone":                      "UTC",
			"schedulePolicy":                map[string]any{"schedulePolicyType": "SimpleSchedulePolicy", "scheduleRunFrequency": "Daily", "scheduleRunTimes": []any{"2020-01-01T02:00:00Z"}},
			"retentionPolicy":               daily(30),
		}),
		azureARMChild(vault, azureBackupPolicyType, "EnhancedPolicy", map[string]any{
			"backupManagementType":          "AzureIaasVM",
			"policyType":                    "V2",
			"instantRpRetentionRangeInDays": 7,
			"timeZone":                      "UTC",
			"schedulePolicy": map[string]any{
				"schedulePolicyType": "SimpleSchedulePolicyV2", "scheduleRunFrequency": "Hourly",
				"hourlySchedule": map[string]any{"interval": 4, "scheduleWindowStartTime": "2020-01-01T08:00:00Z", "scheduleWindowDuration": 12},
			},
			"retentionPolicy": daily(30),
		}),
		azureARMChild(vault, azureBackupPolicyType, "HourlyLogBackup", map[string]any{
			"backupManagementType": "AzureWorkload",
			"workLoadType":         "SQLDataBase",
			"settings":             map[string]any{"timeZone": "UTC", "issqlcompression": false, "isCompression": false},
			"subProtectionPolicy": []any{
				map[string]any{"policyType": "Full", "schedulePolicy": map[string]any{"schedulePolicyType": "SimpleSchedulePolicy", "scheduleRunFrequency": "Daily", "scheduleRunTimes": []any{"2020-01-01T02:00:00Z"}}, "retentionPolicy": daily(30)},
				map[string]any{"policyType": "Log", "schedulePolicy": map[string]any{"schedulePolicyType": "LogSchedulePolicy", "scheduleFrequencyInMins": 60}, "retentionPolicy": map[string]any{"retentionPolicyType": "SimpleRetentionPolicy", "retentionDuration": map[string]any{"count": 30, "durationType": "Days"}}},
			},
		}),
		azureARMChild(vault, azureBackupFabricType, "Azure", map[string]any{}),
	}
}

func azureBackupValidateProtectedItem(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if strings.TrimSpace(azureARMString(props["protectedItemType"])) == "" {
		return "properties.protectedItemType is required."
	}
	// Stopping protection clears the policy.
	if azureARMString(props["protectionState"]) != "ProtectionStopped" && strings.TrimSpace(azureARMString(props["policyId"])) == "" {
		return "properties.policyId is required."
	}
	return ""
}

// azureBackupProtectedItemProperties reports a new item as waiting for its
// initial backup unless the client set a protection state.
func azureBackupProtectedItemProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"protectionStatus": "Healthy"}
	if _, present := rec.Properties["protectionState"]; !present {
		props["protectionState"] = "IRPending"
	}
	return props
}

// azureSingletonName accepts only the one name a singleton child can have.
func azureSingletonName(only string) func(string) string {
	return func(name string) string {
		if !strings.EqualFold(name, only) {
			return "The only name allowed here is '" + only + "'."
		}
		return ""
	}
}

// azureEnumProperty requires properties.{key}, when present, to be one of
// values.
func azureEnumProperty(key string, values ...string) func(map[string]any) string {
	return func(payload map[string]any) string {
		raw, present := azureARMObject(payload["properties"])[key]
		if present && !slices.Contains(values, azureARMString(raw)) {
			return "properties." + key + " must be one of " + strings.Join(values, ", ") + "."
		}
		return ""
	}
}
