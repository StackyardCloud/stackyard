# Azure Monitor (`monitor-2023-01-01`) Staged Plan

## Objective

Emulate the Azure Monitor control plane resources most used by infrastructure-as-code (`Microsoft.Insights`) on the shared ARM handler (`internal/server/azure_resource_manager.go`), including diagnostic settings as the first extension resource.

## References

Microsoft Learn pages (each resource type has its own API version):

- `https://learn.microsoft.com/en-us/rest/api/monitor/action-groups` (`2023-01-01`)
- `https://learn.microsoft.com/en-us/rest/api/monitor/metric-alerts` (`2018-03-01`)
- `https://learn.microsoft.com/en-us/rest/api/monitor/activity-log-alerts` (`2020-10-01`)
- `https://learn.microsoft.com/en-us/rest/api/monitor/scheduled-query-rules` (`2023-12-01`)
- `https://learn.microsoft.com/en-us/rest/api/monitor/autoscale-settings` (`2022-10-01`)
- `https://learn.microsoft.com/en-us/rest/api/monitor/diagnostic-settings` (`2021-05-01-preview`)

## Service Naming

- service key: `monitor`
- display name: `Azure Monitor`
- implementation: `internal/server/provider_azure_monitor.go`
- tests: `internal/server/provider_azure_monitor_test.go`
- example: `examples/azure/monitor-2023-01-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `actionGroups` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `metricAlerts` | tracked | same |
| `activityLogAlerts` | tracked | same |
| `scheduledQueryRules` | tracked | same |
| `autoscalesettings` | tracked | same |
| `diagnosticSettings` | extension | create/update, get, delete, list on a subscription, resource group or any resource |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules (400 `InvalidRequestContent` otherwise):

- action groups need `groupShortName` (at most 12 characters) and `enabled`
- metric alerts need non-empty `scopes`, integer `severity` 0-4, `enabled`, ISO 8601 `evaluationFrequency` and `windowSize`, and `criteria` with an `odata.type`
- activity log alerts need `scopes` and a non-empty `condition.allOf`; scheduled query rules need `scopes` and a non-empty `criteria.allOf`
- autoscale settings need 1 to 20 `profiles`
- diagnostic settings need at least one destination (`workspaceId`, `storageAccountId`, `eventHubAuthorizationRuleId` or `marketplacePartnerId`); they carry no location, a list returns only the settings attached directly to that scope, and settings on a shared-handler resource are deleted with it
- a diagnostic setting's target resource is not required to exist, because it may belong to a service with its own store

Out of scope for this stage:
- evaluating alerts, sending notifications and scaling anything
- metrics and activity log data (`metrics`, `metricDefinitions`, `eventtypes`), and `diagnosticSettingsCategories`
- action group `createNotifications` tests, alert processing rules (`Microsoft.AlertsManagement`, issue #258)
- data collection rules and endpoints, private link scopes, workbooks, and Application Insights components
- checking that alert `scopes` and action group IDs refer to existing resources

## Quality Gates

- `go test ./internal/server -run TestAzureMonitor -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service monitor`
- `python3 scripts/azure-io-contract-coverage.py --service monitor`
- `python3 scripts/azure-doc-contract-coverage.py --service monitor`
