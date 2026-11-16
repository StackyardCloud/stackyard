# Azure Cost Management (`costmanagement-2023-11-01`) Staged Plan

## Objective

Emulate the Azure Cost Management control plane (`Microsoft.CostManagement`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): exports, saved views and scheduled actions as extension resources on subscriptions and resource groups, plus the query and forecast APIs. Stackyard bills nothing, so queries return their columns with no rows, exports write no files and scheduled actions send no email.

## References

Microsoft Learn pages for the `2023-11-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/cost-management/exports/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/cost-management/exports/execute`
- `https://learn.microsoft.com/en-us/rest/api/cost-management/views/create-or-update-by-scope`
- `https://learn.microsoft.com/en-us/rest/api/cost-management/scheduled-actions/create-or-update-by-scope`
- `https://learn.microsoft.com/en-us/rest/api/cost-management/query/usage`
- `https://learn.microsoft.com/en-us/rest/api/cost-management/forecast/usage`

## Service Naming

- service key: `costmanagement`
- display name: `Azure Cost Management`
- implementation: `internal/server/provider_azure_costmanagement.go`
- tests: `internal/server/provider_azure_costmanagement_test.go`
- example: `examples/azure/costmanagement-2023-11-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `{scope}/providers/Microsoft.CostManagement/exports` | extension | create/update, get, delete, list, `run` |
| `{scope}/providers/Microsoft.CostManagement/views` | extension | create/update, get, delete, list |
| `{scope}/providers/Microsoft.CostManagement/scheduledActions` | extension | create/update, get, delete, list, `execute` |
| `{scope}/providers/Microsoft.CostManagement/query` | scope action | POST |
| `{scope}/providers/Microsoft.CostManagement/forecast` | scope action | POST |

`{scope}` is a subscription or resource group; other scopes return 400 `InvalidScope`.

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- a query definition needs a `type` (query and forecast: `Usage`, `ActualCost` or `AmortizedCost`), a `timeframe` from `MonthToDate`, `BillingMonthToDate`, `TheLastMonth`, `TheLastBillingMonth`, `WeekToDate`, `TheCurrentMonth` or `Custom`, and for `Custom` a `timePeriod` with RFC 3339 `from` and `to`, from first; the dataset's `granularity` is `Daily` or `Monthly`, each `aggregation` needs a column `name` and function `Sum`, and `grouping` allows at most two `Dimension` or `TagKey` entries
- `query` and `forecast` return the requested columns, aggregations then `UsageDate` (daily) or `BillingMonth` (monthly) then groupings then `Currency` (forecast adds `CostStatus` before it), and no rows; invalid definitions return 400 `BadRequest`
- exports take an export definition type (adding `FocusCost`, `PriceSheet` and the reservation types), a storage account `deliveryInfo.destination.resourceId` and `container`, a `format` of `Csv` (the default) or `Parquet`, and an optional schedule whose `status` is `Active` or `Inactive` and whose `recurrence` (`Daily`, `Weekly`, `Monthly`, `Annually`) needs an RFC 3339 `recurrencePeriod.from`; `run` returns 200
- views need a `Usage` query; `chart`, `accumulated` and `metric` must be known values
- scheduled actions take `kind` `Email` or `InsightAlert` and need a `displayName`, a `viewId`, a `status` of `Enabled`, `Disabled` or `Expired`, a notification with at least one recipient and a subject, and a `Daily`, `Weekly` or `Monthly` schedule whose `endDate` follows its `startDate`; `execute` returns 200

Out of scope for this stage:
- billing account, billing profile, invoice section, enrollment and management group scopes, and tenant-level (private) views and scheduled actions
- cost data: query rows, export files, run history, dimensions, alerts, budgets and benefit recommendations
- `checkNameAvailability` for scheduled actions, and long-running operation polling

## Quality Gates

- `go test ./internal/server -run TestAzureCostManagement -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service costmanagement`
- `python3 scripts/azure-io-contract-coverage.py --service costmanagement`
- `python3 scripts/azure-doc-contract-coverage.py --service costmanagement`
