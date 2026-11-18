# Azure Impact Reporting (`impact-2026-01-01-preview`) Staged Plan

## Objective

Emulate Azure Impact Reporting (`Microsoft.Impact`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): workload impact reports, the impact categories they're filed under, the insights Microsoft attaches to each report, Azure Monitor connectors, and the log upload token HPC Guest Health Reporting uses. Nothing is correlated with real platform events: an HPC report is acknowledged with an insight, and other insights exist only when they're created through the (admin) insights API.

## References

Microsoft Learn pages for the `2026-01-01-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/impact/workload-impacts/create`
- `https://learn.microsoft.com/en-us/rest/api/impact/workload-impacts/get`
- `https://learn.microsoft.com/en-us/rest/api/impact/workload-impacts/delete`
- `https://learn.microsoft.com/en-us/rest/api/impact/workload-impacts/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/impact/impact-categories/get`
- `https://learn.microsoft.com/en-us/rest/api/impact/impact-categories/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/impact/insights/create`
- `https://learn.microsoft.com/en-us/rest/api/impact/insights/get`
- `https://learn.microsoft.com/en-us/rest/api/impact/insights/delete`
- `https://learn.microsoft.com/en-us/rest/api/impact/insights/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/impact/connectors/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/impact/connectors/get`
- `https://learn.microsoft.com/en-us/rest/api/impact/connectors/update`
- `https://learn.microsoft.com/en-us/rest/api/impact/connectors/delete`
- `https://learn.microsoft.com/en-us/rest/api/impact/connectors/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/impact/upload-service/get-upload-token`

Behavior references:

- `https://learn.microsoft.com/en-us/azure/azure-impact-reporting/view-impact-categories`
- `https://learn.microsoft.com/en-us/azure/azure-impact-reporting/guest-health-impact-categories`
- `https://learn.microsoft.com/en-us/azure/azure-impact-reporting/guest-health-impact-report`
- `https://learn.microsoft.com/en-us/azure/azure-impact-reporting/guest-health-log-upload`
- `https://learn.microsoft.com/en-us/azure/azure-impact-reporting/view-impact-insights`
- `https://learn.microsoft.com/en-us/azure/azure-impact-reporting/create-azure-monitor-connector`

## Service Naming

- service key: `impact`
- display name: `Impact Reporting`
- implementation: `internal/server/provider_azure_impact.go`, `internal/server/provider_azure_impact_validate.go`
- tests: `internal/server/provider_azure_impact_test.go`
- example: `examples/azure/impact-2026-01-01-preview/`

## Operations In Scope

All resources are at subscription scope, under `/subscriptions/{id}/providers/Microsoft.Impact`.

| Resource | Operations |
|---|---|
| `workloadImpacts` | create, get, delete, list |
| `workloadImpacts/{name}/insights` | create, get, delete, list |
| `impactCategories` | get, list with `categoryName` and `resourceType` |
| `connectors` | create or update, get, update (PATCH), delete, list |
| `getUploadToken` | POST |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names follow the reference's patterns: workload impacts `^[a-zA-Z0-9][a-zA-Z0-9-:]*[a-zA-Z0-9]{3,120}$`, insights `^[a-zA-Z0-9][a-zA-Z0-9.]*[a-zA-Z0-9]{3,180}$`, connectors `^[a-zA-Z0-9-]{3,24}$`, applied as written (so workload impact and insight names end in three letters or digits)
- the category catalog is the 24 categories of the impact categories page plus the 26 `Resource.Hpc.*` Guest Health Reporting categories; category IDs are stable emulator UUIDs derived from the name, and each category's parent is its dotted prefix; HPC categories apply to virtual machines only, the others to every resource type; `categoryName` keeps a category and its subcategories
- a report needs `startDateTime`, an `impactedResourceId` and a catalog `impactCategory` (matched case-insensitively and stored in the catalog's spelling); `endDateTime` can't be before the start; enums, numbers and nested objects are checked against the reference
- `ARMOperation.*` reports need `armCorrelationIds`; HPC reports need `impactDescription` and `additionalProperties.PhysicalHostName`, and a `LogUrl` must be a `getUploadToken` URL with its SAS token trimmed (otherwise `INVALID_FORMAT`)
- reports can't be changed: there's no PATCH, and a PUT to an existing name is rejected
- a report gets `impactUniqueId`, `reportedTimeUtc` and `insightsByCategory`, which groups its insights by category with the status of each category's latest insight and stays current as insights change
- an HPC report is acknowledged with a `MitigationAction` insight whose `statusCode` is `Acknowledged` plus the category kind (`Unhealthy`, `Reset` or `Reboot`)
- an insight needs a `category`; its `impact` is filled from its report and can't name another one; it keeps its `insightUniqueId` across PUTs
- connectors are `AzureMonitor` connectors with an optional user-assigned identity; they get a `connectorId`, the tenant ID and `processingState` `Initialized`; PATCH takes the connector type and identity
- `getUploadToken` runs at subscription scope and returns a URL whose blob is named for the UTC time and an eight-digit hex suffix, on a `.stackyard.invalid` host

Out of scope for this stage:
- correlating reports with outages, changes and deployments, and reading Azure Monitor alerts
- checking that the impacted resource exists, and storing uploaded logs
- topology impacts, which only the `2023-12-01-preview` version has
- long-running operation polling: every operation completes synchronously
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzureImpact -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service impact`
- `python3 scripts/azure-io-contract-coverage.py --service impact`
- `python3 scripts/azure-doc-contract-coverage.py --service impact`
