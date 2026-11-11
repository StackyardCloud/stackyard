# Azure Databricks (`databricks_2026_01_01`) Staged Plan

## Objective

Emulate Azure Databricks REST API (`api-version=2026-01-01`) with deterministic local behavior for:

- workspace and access-connector lifecycle flows
- workspace virtual network peering lifecycle flows
- private endpoint connection management flows
- private link resource and outbound dependency discovery surfaces
- provider operations discovery
- an Azure Go SDK example under `examples/azure/databricks-2026-01-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/databricks/operation-groups?view=rest-databricks-2026-01-01`

Operation-group pages reviewed:

- `.../access-connectors?view=rest-databricks-2026-01-01`
- `.../workspaces?view=rest-databricks-2026-01-01`
- `.../v-net-peering?view=rest-databricks-2026-01-01`
- `.../private-endpoint-connections?view=rest-databricks-2026-01-01`
- `.../private-link-resources?view=rest-databricks-2026-01-01`
- `.../outbound-network-dependencies-endpoints?view=rest-databricks-2026-01-01`
- `.../operations?view=rest-databricks-2026-01-01`

Method pages reviewed directly include:

- Access Connectors:
  - `access-connectors/create-or-update`
  - `access-connectors/delete`
  - `access-connectors/get`
  - `access-connectors/list-by-resource-group`
  - `access-connectors/list-by-subscription`
  - `access-connectors/update`
- Workspaces:
  - `workspaces/create-or-update`
  - `workspaces/delete`
  - `workspaces/get`
  - `workspaces/list-by-resource-group`
  - `workspaces/list-by-subscription`
  - `workspaces/update`
- v Net Peering:
  - `v-net-peering/create-or-update`
  - `v-net-peering/delete`
  - `v-net-peering/get`
  - `v-net-peering/list-by-workspace`
- Private Endpoint Connections:
  - `private-endpoint-connections/create`
  - `private-endpoint-connections/delete`
  - `private-endpoint-connections/get`
  - `private-endpoint-connections/list`
- Private Link Resources:
  - `private-link-resources/get`
  - `private-link-resources/list`
- Outbound Network Dependencies Endpoints:
  - `outbound-network-dependencies-endpoints/list`
- Operations:
  - `operations/list`

SDK cross-check used for operation-group parity:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/databricks/armdatabricks`
  - `NewAccessConnectorsClient`
  - `NewWorkspacesClient`
  - `NewVNetPeeringClient`
  - `NewPrivateEndpointConnectionsClient`
  - `NewPrivateLinkResourcesClient`
  - `NewOutboundNetworkDependenciesEndpointsClient`
  - `NewOperationsClient`

Notes on source behavior in this environment:

- the operation-groups index page intermittently wraps behind authorization/cache UX
- direct method pages under `view=rest-databricks-2026-01-01` were available and used for complete method coverage
- method pages consistently exposed verbs, routes, request/response schema, and `ErrorResponse` details

## Surface Summary

Reviewed operation groups and methods:

- Access Connectors (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Workspaces (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- v Net Peering (4):
  - `Create Or Update`, `Delete`, `Get`, `List By Workspace`
- Private Endpoint Connections (4):
  - `Create`, `Delete`, `Get`, `List`
- Private Link Resources (2):
  - `Get`, `List`
- Outbound Network Dependencies Endpoints (1):
  - `List`
- Operations (1):
  - `List`

Total reviewed methods: `24`.

## Service Naming

Canonical Stackyard naming:

- service key: `databricks_2026_01_01`
- display name: `Azure Databricks`
- API version: `2026-01-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_databricks_2026_01_01.go`
- tests: `internal/server/provider_azure_databricks_2026_01_01_test.go`
- example: `examples/azure/databricks-2026-01-01`
- plan doc: `docs/databricks-2026-01-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Databricks/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Databricks/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Databricks/...`

Representative routes:

- `GET /azure/providers/Microsoft.Databricks/operations?api-version=2026-01-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Databricks/accessConnectors/{connectorName}?api-version=2026-01-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Databricks/workspaces/{workspaceName}?api-version=2026-01-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Databricks/workspaces/{workspaceName}/virtualNetworkPeerings/{peeringName}?api-version=2026-01-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Databricks/workspaces/{workspaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-01-01`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Databricks/workspaces/{workspaceName}/privateLinkResources?api-version=2026-01-01`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Databricks/workspaces/{workspaceName}/outboundNetworkDependenciesEndpoints?api-version=2026-01-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- API paths follow ARM management-plane envelope conventions
- list methods return `value` and optional `nextLink`
- common error envelope is `ErrorResponse` with nested error detail structures

Observed response patterns:

- create/update methods return sync or async results:
  - access connector update supports `200` or `202`
  - workspace update supports `200` or `202`
  - vNet peering create/update supports `200` / `201`
  - private endpoint connection create supports `200` / `202`
- delete methods are idempotent and support async polling:
  - access connector/workspace/vNet peering/private endpoint delete support `200`, `202`, and `204`
- read/list methods typically return `200`
- long-running responses document `Location` and `Retry-After` headers on async operations

Notable request/flow specifics:

- workspace create requires `location` and documents required `properties.computeMode`
- private endpoint create is an update-style status operation for connection state
- vNet peering and private endpoint operations are workspace-child routes
- outbound dependency endpoint listing documents network egress dependencies for workspace configurations

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `databricks_2026_01_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Databricks routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Microsoft.Databricks/` path families
- strict `api-version=2026-01-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Databricks route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Access Connectors and Workspaces

Implement:

- Access Connectors: all 6 methods
- Workspaces: all 6 methods
- Operations: `List`

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic provisioning-state transitions and async delete/update behavior
- stable list ordering and paging semantics

Exit criteria:

- core workspace and access-connector management no longer returns `NotImplemented`

### Stage 3: Workspace Networking Resources

Implement:

- v Net Peering: all 4 methods
- Private Endpoint Connections: all 4 methods
- Private Link Resources: both methods
- Outbound Network Dependencies Endpoints: list method

Behavior:

- deterministic child-resource lifecycle under workspace scope
- modeled private endpoint connection states and polling-friendly async behavior
- stable private link/outbound endpoint listing contracts

Exit criteria:

- workspace networking configuration surface is functionally represented

### Stage 4: Testing and Contract Hardening

Add tests for:

- happy-path lifecycle across all 7 operation groups
- API-version/method enforcement and route ownership
- validation failures for malformed route/body combinations
- deterministic pagination and ordering
- async response header and idempotent-delete semantics

Exit criteria:

- `provider_azure_databricks_2026_01_01_test.go` covers stage 1-3 behavior
- service passes strict contract/IO/doc coverage checks

## Coverage Script Integration

Add aliases for `databricks_2026_01_01` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:

- `databricks_2026_01_01`
- `databricks-2026-01-01`
- `databricks`
- `azure_databricks`
- `microsoft_databricks`
- `rest_databricks_2026_01_01`

Map plan doc:

- `SERVICE_PLAN_DOCS["databricks_2026_01_01"] = ["databricks-2026-01-01-plan.md"]`

## Example Plan

Add `examples/azure/databricks-2026-01-01`:

- `main.go`: Azure Go SDK (`azcore/runtime`) authenticated client that exercises representative operations from every reviewed operation group
- `Dockerfile`: builds and runs the example binary
- `docker-compose.yml`: local stack wiring with Databricks env vars

Example coverage target:

- one representative request per reviewed method family
- create/read/list/update/delete lifecycle coverage for workspace and access connector surfaces
- networking child-resource coverage for vNet peering, private endpoint connections, private link resources, and outbound dependencies

## Validation Commands

- `go test ./internal/server -run 'TestAzureDatabricks20260101' -count=1`
- `go build ./examples/azure/databricks-2026-01-01`
- `python3 scripts/azure-contract-coverage.py --service databricks --require-service databricks_2026_01_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service databricks --require-service databricks_2026_01_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service databricks --require-service databricks_2026_01_01 --fail-on any`

Expected stage-0 result:

- router and docs/coverage wiring pass service discovery checks
- strict coverage checks fail until implementation/tests are added in later stages
