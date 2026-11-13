# Azure Recovery Services vaults (`recoveryservices-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Recovery Services vault control plane (`Microsoft.RecoveryServices/vaults`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Backup and Site Recovery resources inside a vault are covered by `docs/recoveryservicesbackup-2024-10-01-plan.md` and `docs/recoveryservicessiterecovery-2024-10-01-plan.md`.

## References

Microsoft Learn pages for the `2024-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/recoveryservices/vaults`
- `https://learn.microsoft.com/en-us/rest/api/recoveryservices/vaults/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/recoveryservices/vaults/update`
- `https://learn.microsoft.com/en-us/rest/api/recoveryservices/recovery-services/check-name-availability`

## Service Naming

- service key: `recoveryservices`
- display name: `Azure Recovery Services vaults`
- implementation: `internal/server/provider_azure_recoveryservices.go`
- tests: `internal/server/provider_azure_recoveryservices_test.go`
- example: `examples/azure/recoveryservices-2024-04-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `vaults` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `locations/{location}/checkNameAvailability` | resource group action | `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 2-50 letters, digits or hyphens and start with a letter (400 `InvalidResourceName`)
- `sku.name` must be `Standard` or `RS0`
- `properties.redundancySettings.standardTierStorageRedundancy` must be `LocallyRedundant`, `GeoRedundant` or `ZoneRedundant`; it defaults to `GeoRedundant`, `crossRegionRestore` defaults to `Disabled`, and `publicNetworkAccess` defaults to `Enabled`
- `privateEndpointStateForBackup`, `privateEndpointStateForSiteRecovery` (`None`) and `backupStorageVersion` (`V2`) are server-owned
- every vault starts with the Backup vault and storage configuration, the `DefaultPolicy`, `EnhancedPolicy` and `HourlyLogBackup` policies and the `Azure` backup fabric; deleting the vault removes everything under it
- `checkNameAvailability` lives at resource group scope and compares names within that group; `type` must be `Microsoft.RecoveryServices/Vaults`, and a missing group returns 404 `ResourceGroupNotFound`

Out of scope for this stage:
- refusing to delete a vault that still holds protected or replicated items
- vault extended information, certificates, usages, private link resources, monitoring settings and immutability settings
- long-running operation polling: create, update and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureRecoveryServices -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service recoveryservices`
- `python3 scripts/azure-io-contract-coverage.py --service recoveryservices`
- `python3 scripts/azure-doc-contract-coverage.py --service recoveryservices`
