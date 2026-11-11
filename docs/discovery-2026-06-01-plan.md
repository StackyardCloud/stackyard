# Azure Discovery (`discovery_2026_06_01`) Staged Plan

## Objective

Emulate Azure Discovery Resource Manager REST API (`api-version=2026-06-01`) with deterministic local behavior for:

- workspace, bookshelf, supercomputer, storage-container lifecycle surfaces
- nested resources (projects, tools, node pools, chat model deployments, storage assets)
- private endpoint connection and private link resource read/list surfaces
- provider operations discovery workflow
- an Azure Go SDK example under `examples/azure/discovery-2026-06-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/discovery/operation-groups?view=rest-discovery-2026-06-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-endpoint-connections?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-link-resources?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/chat-model-deployments?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/node-pools?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/operations?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/projects?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-assets?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/tools?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-endpoint-connections?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-link-resources?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces?view=rest-discovery-2026-06-01`

Method pages reviewed directly: all linked methods from all operation groups (`63` total).

Bookshelf Private Endpoint Connections method pages (`4`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-endpoint-connections/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-endpoint-connections/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-endpoint-connections/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-endpoint-connections/list-by-bookshelf?view=rest-discovery-2026-06-01`

Bookshelf Private Link Resources method pages (`2`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-link-resources/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelf-private-link-resources/list-by-bookshelf?view=rest-discovery-2026-06-01`

Bookshelves method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves/list-by-resource-group?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves/list-by-subscription?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/bookshelves/update?view=rest-discovery-2026-06-01`

Chat Model Deployments method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/chat-model-deployments/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/chat-model-deployments/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/chat-model-deployments/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/chat-model-deployments/list-by-workspace?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/chat-model-deployments/update?view=rest-discovery-2026-06-01`

Node Pools method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/node-pools/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/node-pools/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/node-pools/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/node-pools/list-by-supercomputer?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/node-pools/update?view=rest-discovery-2026-06-01`

Operations method pages (`1`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/operations/list?view=rest-discovery-2026-06-01`

Projects method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/projects/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/projects/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/projects/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/projects/list-by-workspace?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/projects/update?view=rest-discovery-2026-06-01`

Storage Assets method pages (`5`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-assets/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-assets/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-assets/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-assets/list-by-storage-container?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-assets/update?view=rest-discovery-2026-06-01`

Storage Containers method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers/list-by-resource-group?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers/list-by-subscription?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/storage-containers/update?view=rest-discovery-2026-06-01`

Supercomputers method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers/list-by-resource-group?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers/list-by-subscription?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/supercomputers/update?view=rest-discovery-2026-06-01`

Tools method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/tools/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/tools/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/tools/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/tools/list-by-resource-group?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/tools/list-by-subscription?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/tools/update?view=rest-discovery-2026-06-01`

Workspace Private Endpoint Connections method pages (`4`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-endpoint-connections/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-endpoint-connections/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-endpoint-connections/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-endpoint-connections/list-by-workspace?view=rest-discovery-2026-06-01`

Workspace Private Link Resources method pages (`2`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-link-resources/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspace-private-link-resources/list-by-workspace?view=rest-discovery-2026-06-01`

Workspaces method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces/create-or-update?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces/delete?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces/get?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces/list-by-resource-group?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces/list-by-subscription?view=rest-discovery-2026-06-01`
- `https://learn.microsoft.com/en-us/rest/api/discovery/workspaces/update?view=rest-discovery-2026-06-01`

## Surface Summary

Reviewed operation groups and methods:

- Bookshelf Private Endpoint Connections (4)
- Bookshelf Private Link Resources (2)
- Bookshelves (6)
- Chat Model Deployments (5)
- Node Pools (5)
- Operations (1)
- Projects (5)
- Storage Assets (5)
- Storage Containers (6)
- Supercomputers (6)
- Tools (6)
- Workspace Private Endpoint Connections (4)
- Workspace Private Link Resources (2)
- Workspaces (6)

Total reviewed methods: `63`.

## Service Naming

Canonical Stackyard naming:

- service key: `discovery_2026_06_01`
- display name: `Azure Discovery`
- API version: `2026-06-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_discovery_2026_06_01.go`
- tests: `internal/server/provider_azure_discovery_2026_06_01_test.go`
- example: `examples/azure/discovery-2026-06-01`
- plan doc: `docs/discovery-2026-06-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Discovery/operations`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}/privateLinkResources`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/tools`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateLinkResources`

Representative routes:

- `GET /azure/providers/Microsoft.Discovery/operations?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}?api-version=2026-06-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/bookshelves/{bookshelfName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/storageContainers/{storageContainerName}/storageAssets/{storageAssetName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/supercomputers/{supercomputerName}/nodePools/{nodePoolName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/projects/{projectName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/tools/{toolName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/chatModelDeployments/{chatModelDeploymentName}?api-version=2026-06-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Discovery/workspaces/{workspaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-06-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2026-06-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- non-success responses are documented as `Other Status Codes | ErrorResponse`
- list-style methods return pageable envelopes with `nextLink` semantics

Observed response patterns:

- `200` appears in `52` methods
- `201` appears in `11` methods
- `202` appears in `20` methods
- `204` appears in `11` methods
- create-or-update methods commonly return `200` (update) or `201` (create)
- delete methods commonly return `202` (accepted) and `204` (idempotent/no-content)
- update (`PATCH`) methods commonly return `200` and may return `202` with `Location` + `Retry-After`

Required create fields inferred from reviewed request-body tables:

- `location` is the only universally required field across top-level create-or-update resources
- private endpoint connection create-or-update methods accept `properties` objects

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add `docs/discovery-2026-06-01-plan.md` with operation-group/method inventory
- add `internal/server/provider_azure_discovery_2026_06_01.go` stage-0 router skeleton
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service is discoverable by coverage scripts as `discovery_2026_06_01`
- doc coverage finds mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- own `Microsoft.Discovery` management-plane route detection
- enforce `api-version=2026-06-01` for handled routes
- return structured ARM-style errors for unsupported versions

Exit criteria:

- discovery routes are intercepted only by discovery handler
- unknown route fallthrough behavior remains unchanged

### Stage 2: Top-Level Resource Families

Implement CRUD/list/update for:

- `workspaces`
- `bookshelves`
- `storageContainers`
- `supercomputers`

Behavior goals:

- deterministic ID/type/name/systemData envelopes
- list-by-resource-group and list-by-subscription support
- `PUT` supports 200/201 semantics; `DELETE` supports 202/204 semantics
- `PATCH` supports 200/202 semantics with basic async header shape when needed

### Stage 3: Nested Resource Families

Implement nested CRUD/list/update for:

- workspace-scoped: `chatModelDeployments`, `projects`, `tools`
- supercomputer-scoped: `nodePools`
- storageContainer-scoped: `storageAssets`

Behavior goals:

- parent existence checks with ARM-style errors
- deterministic parent/child ID and type construction
- 200/201/202/204 parity on create/update/delete paths

### Stage 4: Private Endpoint + Private Link Surfaces

Implement read/list and update-style flows for:

- `bookshelves/{bookshelfName}/privateEndpointConnections`
- `bookshelves/{bookshelfName}/privateLinkResources`
- `workspaces/{workspaceName}/privateEndpointConnections`
- `workspaces/{workspaceName}/privateLinkResources`

Behavior goals:

- consistent connection state envelope fields
- deterministic mock private endpoint IDs and groupIds
- supported create-or-update and delete semantics for private endpoint connections

### Stage 5: Provider Operations + Error Contract

- implement `GET /providers/Microsoft.Discovery/operations`
- centralize common ARM error payload helpers (`ErrorResponse` shape)
- ensure method-not-allowed and not-found branches return deterministic contracts

### Stage 6: Tests, Example, and Coverage

- add provider tests (`internal/server/provider_azure_discovery_2026_06_01_test.go`) covering:
  - api-version validation
  - create/get/update/delete/list across each operation group
  - negative tests for missing parent, invalid body, unknown method/path
- add example under `examples/azure/discovery-2026-06-01` that exercises all reviewed methods
- verify coverage scripts detect aliases and docs mapping

Exit criteria:

- example compiles and runs against Stackyard
- coverage scripts include `discovery_2026_06_01` in `--list-services`
- doc contract coverage resolves `discovery-2026-06-01-plan.md`

## Example Scope for `examples/azure/discovery-2026-06-01`

The example should include representative calls across all operation groups (`63` methods total), using environment-configurable resource names and expected status assertions per method as documented.

## Notes

This staged plan intentionally separates contract-first onboarding (aliases/docs/example/skeleton) from full runtime emulation. Stage 0-1 can land quickly, while stages 2-6 add behavior incrementally with tests.
