# Azure Alerts Management (`alertsmanagement-2021-08-08`) Staged Plan

## Objective

Emulate the Azure Monitor alerts management control plane (`Microsoft.AlertsManagement`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): alert processing rules, Prometheus rule groups and the fired-alerts list.

## References

Microsoft Learn pages for alert processing rules (`2021-08-08`), Prometheus rule groups (`2023-03-01`) and alerts (`2019-05-05-preview`):

- `https://learn.microsoft.com/en-us/rest/api/monitor/alert-processing-rules`
- `https://learn.microsoft.com/en-us/rest/api/monitor/alert-processing-rules/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/monitor/prometheus-rule-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/monitor/alertsmanagement/alerts/get-all`

## Service Naming

- service key: `alertsmanagement`
- display name: `Azure Alerts Management`
- implementation: `internal/server/provider_azure_alertsmanagement.go`
- tests: `internal/server/provider_azure_alertsmanagement_test.go`
- example: `examples/azure/alertsmanagement-2021-08-08/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `actionRules` (alert processing rules) | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `prometheusRuleGroups` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `alerts` | read-only view | list, get |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- alert processing rules need at least one scope and one action, and each `actionType` must be `AddActionGroups` or `RemoveAllActionGroups`. `AddActionGroups` needs at least one `actionGroupIds` entry. Rules use location `Global`, as Azure does
- Prometheus rule groups need at least one scope and one rule; each rule sets exactly one of `record` or `alert` and needs an `expression`
- both kinds of rule report `enabled` `true` unless the client sets it
- the emulator never fires alerts: the alert list is empty and getting one returns 404 `ResourceNotFound`

Out of scope for this stage:
- checking that scopes and action groups exist, and validating schedules, conditions and PromQL
- alert state changes, alert history, the alerts summary, smart groups, smart detector alert rules, alert rule recommendations and issues

## Quality Gates

- `go test ./internal/server -run 'TestAzureAlertProcessingRules|TestAzurePrometheusRuleGroups' -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service alertsmanagement`
- `python3 scripts/azure-io-contract-coverage.py --service alertsmanagement`
- `python3 scripts/azure-doc-contract-coverage.py --service alertsmanagement`
