# Azure Portal Dashboards (`portal-2020-09-01-preview`) Staged Plan

## Objective

Emulate the Azure portal dashboard control plane (`Microsoft.Portal`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): shared dashboards and the tenant configuration. Nothing renders a dashboard.

## References

Microsoft Learn pages for the `2020-09-01-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/portal/dashboards/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/portal/tenant-configurations/create`
- `https://learn.microsoft.com/en-us/azure/azure-portal/azure-portal-dashboards-structure`

## Service Naming

- service key: `portal`
- display name: `Azure Portal Dashboards`
- implementation: `internal/server/provider_azure_portal.go`
- tests: `internal/server/provider_azure_portal_test.go`
- example: `examples/azure/portal-2020-09-01-preview/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `dashboards` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `tenantConfigurations` | tenant | create/update (name `default`), get, delete, list |
| `listTenantConfigurationViolations` | tenant action | POST (always empty) |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- dashboard names are 3-24 letters, digits or hyphens
- `lenses` may be an array or, as older clients and Terraform send it, an object keyed by index, and so may each lens's `parts`; each lens needs an integer `order` and parts, and each part an integer `position` `x`, `y`, `rowSpan` and `colSpan`; part metadata and dashboard metadata are stored as sent
- the tenant configuration is named `default` and `enforcePrivateMarkdownStorage` must be a boolean; it has no PATCH (405)

Out of scope for this stage:
- rendering dashboards and checking part types or their inputs
- finding tenant configuration violations: the list is always empty

## Quality Gates

- `go test ./internal/server -run TestAzurePortal -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service portal`
- `python3 scripts/azure-io-contract-coverage.py --service portal`
- `python3 scripts/azure-doc-contract-coverage.py --service portal`
