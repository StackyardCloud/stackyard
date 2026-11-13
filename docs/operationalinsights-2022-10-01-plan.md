# Azure Log Analytics (`operationalinsights-2022-10-01`) Staged Plan

## Objective

Emulate Log Analytics workspaces (`Microsoft.OperationalInsights`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2022-10-01` API version (saved searches and data exports use `2020-08-01`):

- `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces`
- `https://learn.microsoft.com/en-us/rest/api/loganalytics/workspaces/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/loganalytics/shared-keys/get-shared-keys`
- `https://learn.microsoft.com/en-us/rest/api/loganalytics/tables`
- `https://learn.microsoft.com/en-us/rest/api/loganalytics/saved-searches`
- `https://learn.microsoft.com/en-us/rest/api/loganalytics/data-exports`

## Service Naming

- service key: `operationalinsights`
- display name: `Azure Log Analytics`
- implementation: `internal/server/provider_azure_operationalinsights.go`
- tests: `internal/server/provider_azure_operationalinsights_test.go`
- example: `examples/azure/operationalinsights-2022-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `workspaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `sharedKeys` |
| `workspaces/tables` | proxy child | create/update, get, update, delete, list |
| `workspaces/savedSearches` | proxy child | create/update, get, delete, list |
| `workspaces/dataExports` | proxy child | create/update, get, delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- workspace names are 4-63 letters, digits or hyphens, starting and ending with a letter or digit (400 `InvalidResourceName`)
- `properties.sku.name` must be a Log Analytics tier and defaults to `PerGB2018`; `retentionInDays` must be an integer from 30 to 730 and defaults to 30
- `customerId`, the workspace ID agents send data to, is a stable UUID derived from the resource ID and cannot be set by clients
- `sharedKeys` returns primary and secondary keys derived from the resource ID
- saved searches need `category`, `displayName` and `query`; data exports need `destination.resourceId` and non-empty `tableNames`

Out of scope for this stage:
- ingesting and querying logs (the data plane, including the HTTP Data Collector API)
- soft delete and workspace recovery (`DELETE` removes a workspace immediately)
- the built-in table catalog (tables exist only once created), `regenerateSharedKey`, linked services, linked storage accounts, clusters and query packs
- Free-tier retention rules

## Quality Gates

- `go test ./internal/server -run TestAzureOperationalInsights -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service operationalinsights`
- `python3 scripts/azure-io-contract-coverage.py --service operationalinsights`
- `python3 scripts/azure-doc-contract-coverage.py --service operationalinsights`
