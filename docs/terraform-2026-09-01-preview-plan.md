# Azure Export for Terraform (`terraform-2026-09-01-preview`) Staged Plan

## Objective

Emulate Azure Export for Terraform (`Microsoft.AzureTerraform`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): export resource groups and the resources the handler stores as Terraform configuration and import blocks, for the `azurerm` or `azapi` provider.

## References

Microsoft Learn pages for the `2026-09-01-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/terraform/terraform/export-terraform`

## Service Naming

- service key: `terraform`
- display name: `Azure Export for Terraform`
- implementation: `internal/server/provider_azure_terraform.go`
- tests: `internal/server/provider_azure_terraform_test.go`
- example: `examples/azure/terraform-2026-09-01-preview/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `/subscriptions/{id}/providers/Microsoft.AzureTerraform/exportTerraform` | POST |

## Behavior

Service-specific rules:

- `type` selects what to export: `ExportResourceGroup` (`resourceGroupName`, the group and every resource in it, children included; a missing group is 404 `ResourceGroupNotFound`), `ExportResource` (`resourceIds`; IDs naming nothing become entries in `errors`; `recursive` adds the resources under each ID) or `ExportQuery` (`query`)
- queries take Azure Resource Graph predicates of the form `field op "value"` joined by `and`, where field is `id`, `name`, `type`, `location`, `resourceGroup` or `subscriptionId` and op is `==`, `!=` (case-sensitive), `=~` or `!~` (case-insensitive); like Resource Graph, `type`, `location`, `resourceGroup` and `subscriptionId` compare in lower case, and quoted values may contain `and`; anything else is 400 `InvalidQuery`, and `table` must be `Resources`
- `includeResourceGroup` adds the groups of the exported resources; `excludeAzureResource` drops resources whose ID matches a case-insensitive regular expression, and `excludeTerraformResource` drops Terraform resource types
- `targetProvider` `azurerm` (default) maps resource groups, container registries, Logic Apps workflows, user-assigned identities, Maps accounts, application security groups, NAT gateways, network security groups, public IP addresses, route tables, virtual networks, Log Analytics workspaces and Recovery Services vaults; other resources are listed in `skippedResources`
- `azapi` exports every resource as `azapi_resource`, typed with the API version that wrote it (resource groups use `2021-04-01`), its parent ID, and a body holding its properties, SKU and kind
- Terraform names are `namePattern` (default `res-`) plus a counter; `resourceName` and `resourceType` rename and retype the one resource of a single-ID `ExportResource`; names must start with a letter or underscore and hold letters, digits, `_` and `-`, and types must be snake case
- a resource group listed more than once exports once
- the result answers 200 with `status` `Succeeded`, an `operationStatuses` ID, `configuration` and `import` blocks
- the request is validated: booleans, strings, `targetProvider`, `includeExtensions`, `authorizationScopeFilter` and the exclusion lists; the export runs only at subscription scope (400 `InvalidScope` elsewhere)

Out of scope for this stage:
- per-type azurerm attributes beyond name, location, resource group and tags
- resources kept outside the shared ARM handler's store, extension resources from `includeExtensions`, `fullProperties` and `maskSensitive` (write-only values are never stored)
- 202 Accepted and operation status polling: every export completes synchronously
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzureTerraform -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service terraform`
- `python3 scripts/azure-io-contract-coverage.py --service terraform`
- `python3 scripts/azure-doc-contract-coverage.py --service terraform`
