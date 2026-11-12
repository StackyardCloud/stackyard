# Azure Authorization (`authorization-2022-04-01`) Staged Plan

## Objective

Emulate Azure Authorization (`2022-04-01`) with deterministic local behavior for the ARM management-plane resource lifecycle and discovery endpoints under `Microsoft.Authorization`, including:

- `Permissions`
- `Role Assignments`
- `Role Definitions`
- a typed Azure Go SDK example using the Azure Authorization SDK
- Azure coverage script wiring for the new service key

This service is a good candidate for a staged implementation because the reviewed surface is compact, centered on standard ARM scope patterns, and heavily reuses a small number of shared models: `RoleAssignment`, `RoleDefinition`, `Permission`, `ErrorResponse`, and list envelopes.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/authorization/operation-groups?view=rest-authorization-2022-04-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/authorization/permissions?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions?view=rest-authorization-2022-04-01`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/authorization/permissions/list-for-resource?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/permissions/list-for-resource-group?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/create?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/create-by-id?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/delete?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/delete-by-id?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/get?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/get-by-id?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/list-for-resource?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/list-for-resource-group?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/list-for-scope?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/list-for-subscription?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/create-or-update?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/delete?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/get?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/get-by-id?view=rest-authorization-2022-04-01`
- `https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list?view=rest-authorization-2022-04-01`

## Service Naming

Canonical Stackyard naming:
- service key: `authorization`
- display name: `Azure Authorization`
- API version: `2022-04-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_authorization.go`
- tests: `internal/server/provider_azure_authorization_test.go`
- example: `examples/azure/authorization-2022-04-01`
- plan doc: `docs/authorization-2022-04-01-plan.md`

## Surface Summary

The reviewed REST surface is split into 3 operation groups:

1. `Permissions`
2. `Role Assignments`
3. `Role Definitions`

Total linked method pages reviewed: `17`

Method inventory:

`Permissions`
- `List For Resource`
- `List For Resource Group`

`Role Assignments`
- `Create`
- `Create By Id`
- `Delete`
- `Delete By Id`
- `Get`
- `Get By Id`
- `List For Resource`
- `List For Resource Group`
- `List For Scope`
- `List For Subscription`

`Role Definitions`
- `Create Or Update`
- `Delete`
- `Get`
- `Get By Id`
- `List`

## Route Envelope

Stackyard route envelope should follow ARM management-plane paths under `/azure`:

- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Authorization/permissions`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/providers/Microsoft.Authorization/permissions`
- `/azure/{scope}/providers/Microsoft.Authorization/roleAssignments`
- `/azure/{scope}/providers/Microsoft.Authorization/roleAssignments/{roleAssignmentName}`
- `/azure/{roleAssignmentId}`
- `/azure/{scope}/providers/Microsoft.Authorization/roleDefinitions`
- `/azure/{scope}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}`
- `/azure/{roleId}`

Target query:
- `api-version=2022-04-01`

Important routing implication:
- this service mixes normal ARM scoped resource paths with fully qualified `By Id` forms, so the router should handle both:
  - scope-based forms such as `/subscriptions/.../providers/Microsoft.Authorization/roleAssignments/...`
  - by-ID forms where the path itself is the fully qualified resource ID under `/azure`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:
- All reviewed methods are standard ARM OAuth2 routes in Learn; local Stackyard examples should continue to use the repo’s shared-key Azure shim transport.
- All methods use `ErrorResponse` for failures.
- Scope may be subscription, resource group, or resource.
- `api-version` is required on every method page in this surface.

### Permissions

`Permissions - List For Resource`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}/providers/Microsoft.Authorization/permissions`
- response: `PermissionGetResult`

`Permissions - List For Resource Group`
- `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Authorization/permissions`
- response: `PermissionGetResult`

Important modeled fields from the reviewed pages:
- `value[]`
- `actions[]`
- `notActions[]`
- `dataActions[]`
- `notDataActions[]`

Modeling implication:
- this group is read-only and deterministic fixtures are sufficient for the first implementation slice.

### Role Assignments

`Role Assignments - Create`
- `PUT {scope}/providers/Microsoft.Authorization/roleAssignments/{roleAssignmentName}`
- required request fields:
  - `properties.principalId`
  - `properties.roleDefinitionId`
- optional request fields:
  - `properties.principalType`
  - `properties.description`
  - `properties.condition`
  - `properties.conditionVersion`
  - `properties.delegatedManagedIdentityResourceId`
- responses: `200` or `201`, returning `RoleAssignment`

`Role Assignments - Create By Id`
- `PUT {roleAssignmentId}`
- same request/response model as `Create`

`Role Assignments - Delete`
- `DELETE {scope}/providers/Microsoft.Authorization/roleAssignments/{roleAssignmentName}`
- responses include `200`, and the reviewed page uses the same `RoleAssignment` model plus `ErrorResponse`

`Role Assignments - Delete By Id`
- `DELETE {roleAssignmentId}`

`Role Assignments - Get`
- `GET {scope}/providers/Microsoft.Authorization/roleAssignments/{roleAssignmentName}`
- optional query:
  - `tenantId`

`Role Assignments - Get By Id`
- `GET {roleAssignmentId}`
- optional query:
  - `tenantId`

List methods:
- `List For Resource`
- `List For Resource Group`
- `List For Scope`
- `List For Subscription`

Common list query parameters from the reviewed pages:
- `$filter`
  - `atScope()`
  - `principalId eq {id}`
- `tenantId`
- `$skipToken` on `List For Scope`

Important modeled fields from the reviewed pages:
- `id`, `name`, `type`
- `properties.roleDefinitionId`
- `properties.principalId`
- `properties.principalType`
- `properties.scope`
- optional metadata:
  - `properties.description`
  - `properties.condition`
  - `properties.conditionVersion`
  - `properties.createdBy`
  - `properties.createdOn`
  - `properties.updatedBy`
  - `properties.updatedOn`
- list envelope:
  - `value`
  - `nextLink`

Modeling implication:
- `RoleAssignment` should be stored as a canonical object keyed by full ID, then projected into both scope-based and by-ID routes.

### Role Definitions

`Role Definitions - Create Or Update`
- `PUT {scope}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}`
- request fields from the reviewed page include:
  - `properties.roleName`
  - `properties.type`
  - `properties.description`
  - `properties.assignableScopes`
  - `properties.permissions`

`Role Definitions - Delete`
- `DELETE {scope}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}`
- responses:
  - `200` with `RoleDefinition`
  - `204` when already deleted or missing

`Role Definitions - Get`
- `GET {scope}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}`

`Role Definitions - Get By Id`
- `GET {roleId}`

`Role Definitions - List`
- `GET {scope}/providers/Microsoft.Authorization/roleDefinitions`
- optional query:
  - `$filter=atScopeAndBelow`

Important modeled fields from the reviewed pages:
- `id`, `name`, `type`
- `properties.roleName`
- `properties.type`
- `properties.description`
- `properties.assignableScopes`
- `properties.permissions[]`
  - `actions[]`
  - `notActions[]`
  - `dataActions[]`
  - `notDataActions[]`
- optional metadata:
  - `properties.createdBy`
  - `properties.createdOn`
  - `properties.updatedBy`
  - `properties.updatedOn`

Modeling implication:
- `RoleDefinition` should also be stored canonically by full ID, with scope-based and by-ID lookup paths sharing the same backing record.

## Non-Goals

- Do not emulate Azure AD directory objects or validate principal existence.
- Do not implement real RBAC permission evaluation against every resource in the repo.
- Do not implement deny assignments, management-group-specific resource providers, or unrelated `Microsoft.Authorization` surfaces not present in `2022-04-01`.
- Do not implement full cross-tenant semantics; accept and echo `tenantId` where relevant without real tenant isolation.

## Planned Repo Shape

Implementation:
- `internal/server/provider_azure_authorization.go`

Tests:
- `internal/server/provider_azure_authorization_test.go`

Example:
- `examples/azure/authorization-2022-04-01/main.go`
- `examples/azure/authorization-2022-04-01/Dockerfile`
- `examples/azure/authorization-2022-04-01/docker-compose.yml`

Coverage wiring:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Expected Azure Go SDK package family for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization`

## Example Strategy

The example should follow the existing repo’s ARM Azure Go SDK pattern:

- `azsdkshim.StaticTokenCredential{}`
- `arm.ClientOptions`
- `cloud.ResourceManager` endpoint override to Stackyard
- `azsdkshim.NewTransport("/azure", account, subscriptionKey)`

Representative client coverage expected in the example:
- `RoleAssignmentsClient`
- `RoleDefinitionsClient`
- `PermissionsClient`

Representative example calls:
- create role definition
- get role definition
- list role definitions
- create role assignment
- get role assignment
- list role assignments for subscription or scope
- list permissions for a resource group
- delete role assignment
- delete role definition

## Coverage Naming And Aliases

Recommended aliases to add:
- `authorization`
- `authorization_2022_04_01`
- `azure_authorization`
- `microsoft_authorization`
- `rest_authorization_2022_04_01`

Recommended doc mapping:
- `authorization -> authorization-2022-04-01-plan.md`

## Staged Implementation Plan

### Stage 0: Contract Skeleton And Coverage Naming

- Add a dedicated Azure Authorization router.
- Add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Map this plan doc in `azure-doc-contract-coverage.py`.
- Add route-recognition tests for:
  - permissions endpoints
  - scope-based role-assignment routes
  - by-ID role-assignment routes
  - scope-based role-definition routes
  - by-ID role-definition routes

Exit criteria:
- Coverage scripts discover the new canonical service key.
- The Azure router claims Authorization paths deterministically.

### Stage 1: Validation Foundation

- Validate `api-version` like other Azure management-plane handlers.
- Enforce supported methods:
  - `GET`, `PUT`, `DELETE`
- Validate required path parts:
  - subscription/resource group names where present
  - `roleAssignmentName`
  - `roleDefinitionId`
  - full by-ID resource ID shapes
- Validate required create/update request fields:
  - role assignment: `principalId`, `roleDefinitionId`
  - role definition: `roleName`, `permissions`, `assignableScopes`
- Return deterministic `InvalidRequest` or typed `ErrorResponse` payloads for malformed state.

Exit criteria:
- Request-validation signals and negative tests satisfy the Azure contract and IO coverage scripts.

### Stage 2: Permissions Read-Only Endpoints

- Implement deterministic `PermissionGetResult` fixtures for:
  - `List For Resource`
  - `List For Resource Group`
- Return stable action/data-action arrays suitable for test assertions.

Exit criteria:
- Permission routes are typed and no longer use generic foundation success.

### Stage 3: Role Definitions Lifecycle

- Implement canonical in-memory storage for `RoleDefinition`.
- Support:
  - `Create Or Update`
  - `Get`
  - `Get By Id`
  - `List`
  - `Delete`
- Honor deterministic `200` / `201` / `204` behavior from the reviewed pages where applicable.
- Support `$filter=atScopeAndBelow` as a shallow deterministic filter flag.

Exit criteria:
- Core role-definition CRUD/list operations return typed payloads and no route returns `NotImplemented`.

### Stage 4: Role Assignments Lifecycle

- Implement canonical in-memory storage for `RoleAssignment`.
- Support:
  - `Create`
  - `Create By Id`
  - `Get`
  - `Get By Id`
  - `Delete`
  - `Delete By Id`
  - `List For Resource`
  - `List For Resource Group`
  - `List For Scope`
  - `List For Subscription`
- Accept optional fields:
  - `principalType`
  - `description`
  - `condition`
  - `conditionVersion`
  - `delegatedManagedIdentityResourceId`
- Provide deterministic `nextLink` only where useful for coverage.

Exit criteria:
- All role-assignment routes return typed data and list envelopes.
- Both scope-based and by-ID forms operate against the same backing records.

### Stage 5: Azure Go SDK Example

- Add `examples/azure/authorization-2022-04-01`.
- Exercise `PermissionsClient`, `RoleAssignmentsClient`, and `RoleDefinitionsClient`.
- Keep the example compile-stable and compatible with Stackyard’s Azure shim transport.

Exit criteria:
- Example compiles and demonstrates representative Authorization calls against Stackyard.

### Stage 6: Coverage Wiring Validation

- Verify:
  - `python3 scripts/azure-contract-coverage.py --service authorization --require-service authorization --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service authorization --require-service authorization --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service authorization --require-service authorization --fail-on any`
- Confirm that no routed Authorization endpoint returns `NotImplemented`.

Exit criteria:
- Coverage scripts pass for the canonical service key.

## Validation Targets For Implementation

When the implementation phase happens, the expected validation loop is:

- `go test ./internal/server -run 'TestAzureAuthorization' -count=1`
- `go build ./examples/azure/authorization-2022-04-01`
- `python3 scripts/azure-contract-coverage.py --service authorization --require-service authorization --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service authorization --require-service authorization --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service authorization --require-service authorization --fail-on any`

Success criteria:
- all recognized Authorization endpoints return typed success or modeled validation failures
- no routed Authorization endpoint returns `NotImplemented`
- coverage discovery can resolve `authorization`
