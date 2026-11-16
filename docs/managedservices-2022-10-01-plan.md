# Azure Lighthouse (`managedservices-2022-10-01`) Staged Plan

## Objective

Emulate the Azure Lighthouse control plane (`Microsoft.ManagedServices`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): registration definitions and registration assignments at subscription and resource group scope. No access is delegated to the managing tenant.

## References

Microsoft Learn pages for the `2022-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/managedservices/registration-definitions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/managedservices/registration-assignments/create-or-update`
- `https://learn.microsoft.com/en-us/azure/lighthouse/concepts/tenants-users-roles`

## Service Naming

- service key: `managedservices`
- display name: `Azure Lighthouse`
- implementation: `internal/server/provider_azure_managedservices.go`
- tests: `internal/server/provider_azure_managedservices_test.go`
- example: `examples/azure/managedservices-2022-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `registrationDefinitions` | extension (subscription, resource group) | create/update, get, delete, list |
| `registrationAssignments` | extension (subscription, resource group) | create/update, get, delete, list |
| `marketplaceRegistrationDefinitions` | read-only view (tenant and scope) | list (always empty) |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- definitions and assignments attach only to a subscription or resource group (400 `InvalidScope` otherwise), and their names are GUIDs
- definitions need a `managedByTenantId` GUID other than the customer's tenant and at least one authorization; each authorization and eligible authorization needs a `principalId` and `roleDefinitionId` GUID, can't use the Owner role, and may list `delegatedRoleDefinitionIds` only with User Access Administrator; just-in-time `multiFactorAuthProvider` is `Azure` or `None`
- definitions report the customer tenant as `manageeTenantId`
- assignments need the ID of an existing registration definition; with no PATCH in the API, PATCH returns 405

Out of scope for this stage:
- delegating access: role assignments for the managing tenant, and listing delegations from the managing tenant's side
- `$expandRegistrationDefinition` and `$filter` on lists, marketplace offers, and the marketplace `plan` on a definition, which is not stored
- checking that roles exist or have no data actions

## Quality Gates

- `go test ./internal/server -run TestAzureLighthouse -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service managedservices`
- `python3 scripts/azure-io-contract-coverage.py --service managedservices`
- `python3 scripts/azure-doc-contract-coverage.py --service managedservices`
