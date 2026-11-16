# Azure Maintenance (`maintenance-2023-04-01`) Staged Plan

## Objective

Emulate the Azure Maintenance control plane (`Microsoft.Maintenance`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): maintenance configurations, configuration assignments, on-demand update runs, pending updates and public SQL maintenance windows. No update is ever scheduled or installed.

## References

Microsoft Learn pages for the `2023-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/maintenance/maintenance-configurations/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/maintenance/configuration-assignments/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/maintenance/configuration-assignments-for-subscriptions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/maintenance/apply-updates/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/maintenance/public-maintenance-configurations/list`

## Service Naming

- service key: `maintenance`
- display name: `Azure Maintenance`
- implementation: `internal/server/provider_azure_maintenance.go`
- tests: `internal/server/provider_azure_maintenance_test.go`
- example: `examples/azure/maintenance-2023-04-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `maintenanceConfigurations` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `configurationAssignments` | extension (resource, resource group, subscription) | create/update, get, update (PATCH), delete, list |
| `applyUpdates` | extension (resource) | create/update (name `default`), get |
| `updates` | read-only view | list (always empty) |
| `publicMaintenanceConfigurations` | read-only view (subscription) | list, get |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- `maintenanceScope` is required: `Host`, `Resource`, `OSImage`, `Extension`, `InGuestPatch`, `SQLDB` or `SQLManagedInstance`; `visibility` is `Custom` or `Public`
- a `maintenanceWindow`, when set, needs `startDateTime` (and any `expirationDateTime`) as `yyyy-mm-dd hh:mm` and any `duration` as `hh:mm`; a `timeZone` or `recurEvery`, when sent, can't be blank
- `InGuestPatch` configurations need a `maintenanceWindow` with a `timeZone` and `recurEvery`, `installPatches` (with `rebootSetting` `IfRequired`, `Never` or `Always`) and `extensionProperties.InGuestPatchMode` `User` or `Platform`
- assignments need a maintenance configuration ID, keep the `location` the client sends, and report the scope they attach to as `resourceId`; dynamic-scope `filter.osTypes` are `Windows` or `Linux` and `filter.tagSettings.filterOperator` is `All` or `Any`
- update runs attach only to a resource (400 `InvalidScope` otherwise), are named `default`, and complete at once with `status` `Completed`
- public configurations are `SQL_Default` plus weekday (`_DB_1`, `_MI_1`) and weekend (`_DB_2`, `_MI_2`) windows for `eastus`, `westus2` and `westeurope`

Out of scope for this stage:
- installing patches, pending updates, and update history (`applyUpdates` list by subscription)
- checking that an assignment's configuration and resource exist
- naming an update run by a generated GUID: it keeps the name `default`
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureMaintenance -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service maintenance`
- `python3 scripts/azure-io-contract-coverage.py --service maintenance`
- `python3 scripts/azure-doc-contract-coverage.py --service maintenance`
