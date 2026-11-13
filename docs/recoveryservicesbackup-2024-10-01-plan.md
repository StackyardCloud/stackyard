# Azure Backup (`recoveryservicesbackup-2024-10-01`) Staged Plan

## Objective

Emulate the Azure Backup control plane inside a Recovery Services vault (`Microsoft.RecoveryServices/vaults/backup*`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). The vault itself is covered by `docs/recoveryservices-2024-04-01-plan.md`.

## References

Microsoft Learn pages for the `2024-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/backup/backup-resource-vault-configs`
- `https://learn.microsoft.com/en-us/rest/api/backup/backup-resource-storage-configs-non-crr`
- `https://learn.microsoft.com/en-us/rest/api/backup/protection-policies`
- `https://learn.microsoft.com/en-us/rest/api/backup/protection-containers/register`
- `https://learn.microsoft.com/en-us/rest/api/backup/protected-items/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/backup/backups/trigger`

## Service Naming

- service key: `recoveryservicesbackup`
- display name: `Azure Backup`
- implementation: `internal/server/provider_azure_recoveryservicesbackup.go`
- tests: `internal/server/provider_azure_recoveryservicesbackup_test.go`
- example: `examples/azure/recoveryservicesbackup-2024-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `vaults/backupconfig/vaultconfig` | singleton | get, update (PUT and PATCH) |
| `vaults/backupstorageconfig/vaultstorageconfig` | singleton | get, update (PUT and PATCH) |
| `vaults/backupPolicies` | proxy child | create/update, get, delete, list |
| `vaults/backupFabrics/Azure/protectionContainers` | proxy child | register (PUT), get, unregister (DELETE), list |
| `.../protectionContainers/{container}/protectedItems` | proxy child | create/update, get, delete, list, `backup` (trigger) |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- every vault starts with `vaultconfig` (soft delete and enhanced security `Enabled`), `vaultstorageconfig` (`GeoRedundant`, `Unlocked`), the `DefaultPolicy` and `EnhancedPolicy` Azure VM policies, the `HourlyLogBackup` SQL workload policy and the `Azure` fabric
- the singletons and the fabric accept only their one name (400 `InvalidResourceName`)
- `softDeleteFeatureState` must be `Enabled`, `Disabled` or `AlwaysON`; `storageModelType` must be `LocallyRedundant`, `GeoRedundant` or `ZoneRedundant`
- `backupManagementType` on policies and containers must be a known management type (`AzureIaasVM`, `AzureSql`, `AzureStorage`, `AzureWorkload`, `MAB`, `DPM`, `AzureBackupServer`, `GenericProtectionPolicy`); policies report `protectedItemsCount` 0
- registered containers report `registrationStatus` `Registered` and `healthStatus` `Healthy`
- protected items need `protectedItemType` and a `policyId`, unless `protectionState` is `ProtectionStopped`; new items report `protectionState` `IRPending` and `protectionStatus` `Healthy`
- `backup` on a protected item returns 202 and starts no job

Out of scope for this stage:
- implicit containers: Azure VM items need their container registered first, where Azure creates it on protect
- the vault-wide `backupProtectedItems`, `backupProtectionContainers`, `backupJobs`, `backupUsageSummaries` and `backupEngines` lists
- checking that `policyId` names an existing policy, counting protected items per policy, and locking storage redundancy once items are protected
- recovery points, restores, item-level recovery, inquiries, resource guard proxies and operation status polling

## Quality Gates

- `go test ./internal/server -run TestAzureBackup -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service recoveryservicesbackup`
- `python3 scripts/azure-io-contract-coverage.py --service recoveryservicesbackup`
- `python3 scripts/azure-doc-contract-coverage.py --service recoveryservicesbackup`
