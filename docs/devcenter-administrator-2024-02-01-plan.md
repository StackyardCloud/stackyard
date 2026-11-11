# Azure Dev Center Administrator (`devcenter_administrator_2024_02_01`) Staged Plan

## Objective

Emulate Azure Dev Center Administrator REST API (`api-version=2024-02-01`) with deterministic local behavior for:

- dev center, project, catalog, attached-network, pool, and schedule lifecycle flows
- network connection lifecycle plus health and outbound dependency inspection actions
- environment type and environment definition surfaces (devcenter and project scopes)
- gallery/image/image-version query surfaces
- name-availability checks, provider operations, operation-status polling, sku usage/listing
- an Azure Go SDK example under `examples/azure/devcenter-administrator-2024-02-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/devcenter/administrator/operation-groups?view=rest-devcenter-administrator-2024-02-01`

Operation-group pages reviewed:

- `.../attached-networks?view=rest-devcenter-administrator-2024-02-01`
- `.../catalogs?view=rest-devcenter-administrator-2024-02-01`
- `.../check-name-availability?view=rest-devcenter-administrator-2024-02-01`
- `.../check-scoped-name-availability?view=rest-devcenter-administrator-2024-02-01`
- `.../dev-box-definitions?view=rest-devcenter-administrator-2024-02-01`
- `.../devcenters?view=rest-devcenter-administrator-2024-02-01`
- `.../environment-definitions?view=rest-devcenter-administrator-2024-02-01`
- `.../environment-types?view=rest-devcenter-administrator-2024-02-01`
- `.../galleries?view=rest-devcenter-administrator-2024-02-01`
- `.../images?view=rest-devcenter-administrator-2024-02-01`
- `.../image-versions?view=rest-devcenter-administrator-2024-02-01`
- `.../network-connections?view=rest-devcenter-administrator-2024-02-01`
- `.../operation-statuses?view=rest-devcenter-administrator-2024-02-01`
- `.../operations?view=rest-devcenter-administrator-2024-02-01`
- `.../pools?view=rest-devcenter-administrator-2024-02-01`
- `.../project-allowed-environment-types?view=rest-devcenter-administrator-2024-02-01`
- `.../project-catalog-environment-definitions?view=rest-devcenter-administrator-2024-02-01`
- `.../project-catalogs?view=rest-devcenter-administrator-2024-02-01`
- `.../project-environment-types?view=rest-devcenter-administrator-2024-02-01`
- `.../projects?view=rest-devcenter-administrator-2024-02-01`
- `.../schedules?view=rest-devcenter-administrator-2024-02-01`
- `.../skus?view=rest-devcenter-administrator-2024-02-01`
- `.../usages?view=rest-devcenter-administrator-2024-02-01`

Method pages reviewed directly (complete operation set, grouped):

- Attached Networks: `create-or-update`, `delete`, `get-by-dev-center`, `get-by-project`, `list-by-dev-center`, `list-by-project`
- Catalogs: `connect`, `create-or-update`, `delete`, `get`, `get-sync-error-details`, `list-by-dev-center`, `sync`, `update`
- Check Name Availability: `execute`
- Check Scoped Name Availability: `execute`
- Dev Box Definitions: `create-or-update`, `delete`, `get`, `get-by-project`, `list-by-dev-center`, `list-by-project`, `update`
- DevCenters: `create-or-update`, `delete`, `get`, `list-by-resource-group`, `list-by-subscription`, `update`
- Environment Definitions: `get`, `get-by-project-catalog`, `get-error-details`, `list-by-catalog`, `list-by-project-catalog`
- Environment Types: `create-or-update`, `delete`, `get`, `list-by-dev-center`, `update`
- Galleries: `create-or-update`, `delete`, `get`, `list-by-dev-center`
- Images: `get`, `list-by-dev-center`, `list-by-gallery`
- Image Versions: `get`, `list-by-image`
- Network Connections: `create-or-update`, `delete`, `get`, `get-health-details`, `list-by-resource-group`, `list-by-subscription`, `list-health-details`, `list-outbound-network-dependencies-endpoints`, `run-health-checks`, `update`
- Operation Statuses: `get`
- Operations: `list`
- Pools: `create-or-update`, `delete`, `get`, `list-by-project`, `run-health-checks`, `update`
- Project Allowed Environment Types: `get`, `list`
- Project Catalog Environment Definitions: `get-error-details`
- Project Catalogs: `connect`, `create-or-update`, `delete`, `get`, `get-sync-error-details`, `list`, `patch`, `sync`
- Project Environment Types: `create-or-update`, `delete`, `get`, `list`, `update`
- Projects: `create-or-update`, `delete`, `get`, `list-by-resource-group`, `list-by-subscription`, `update`
- Schedules: `create-or-update`, `delete`, `get`, `list-by-pool`, `update`
- SKUs: `list-by-subscription`
- Usages: `list-by-location`

SDK cross-check used for operation-group parity:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/devcenter/armdevcenter/v2`

Notes on source behavior in this environment:

- Learn pages intermittently render authorization/cache wrappers
- method route templates, request/response contracts, and error-shape sections remained readable and were used for contract planning

## Surface Summary

Reviewed operation groups and methods:

- Attached Networks (6):
  - `Create Or Update`, `Delete`, `Get By Dev Center`, `Get By Project`, `List By Dev Center`, `List By Project`
- Catalogs (8):
  - `Connect`, `Create Or Update`, `Delete`, `Get`, `Get Sync Error Details`, `List By Dev Center`, `Sync`, `Update`
- Check Name Availability (1):
  - `Execute`
- Check Scoped Name Availability (1):
  - `Execute`
- Dev Box Definitions (7):
  - `Create Or Update`, `Delete`, `Get`, `Get By Project`, `List By Dev Center`, `List By Project`, `Update`
- DevCenters (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Environment Definitions (5):
  - `Get`, `Get By Project Catalog`, `Get Error Details`, `List By Catalog`, `List By Project Catalog`
- Environment Types (5):
  - `Create Or Update`, `Delete`, `Get`, `List By Dev Center`, `Update`
- Galleries (4):
  - `Create Or Update`, `Delete`, `Get`, `List By Dev Center`
- Images (3):
  - `Get`, `List By Dev Center`, `List By Gallery`
- Image Versions (2):
  - `Get`, `List By Image`
- Network Connections (10):
  - `Create Or Update`, `Delete`, `Get`, `Get Health Details`, `List By Resource Group`, `List By Subscription`, `List Health Details`, `List Outbound Network Dependencies Endpoints`, `Run Health Checks`, `Update`
- Operation Statuses (1):
  - `Get`
- Operations (1):
  - `List`
- Pools (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Project`, `Run Health Checks`, `Update`
- Project Allowed Environment Types (2):
  - `Get`, `List`
- Project Catalog Environment Definitions (1):
  - `Get Error Details`
- Project Catalogs (8):
  - `Connect`, `Create Or Update`, `Delete`, `Get`, `Get Sync Error Details`, `List`, `Patch`, `Sync`
- Project Environment Types (5):
  - `Create Or Update`, `Delete`, `Get`, `List`, `Update`
- Projects (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Schedules (5):
  - `Create Or Update`, `Delete`, `Get`, `List By Pool`, `Update`
- SKUs (1):
  - `List By Subscription`
- Usages (1):
  - `List By Location`

Total reviewed methods: `95`.

## Service Naming

Canonical Stackyard naming:

- service key: `devcenter_administrator_2024_02_01`
- display name: `Azure Dev Center Administrator`
- API version: `2024-02-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_devcenter_administrator_2024_02_01.go`
- tests: `internal/server/provider_azure_devcenter_administrator_2024_02_01_test.go`
- example: `examples/azure/devcenter-administrator-2024-02-01`
- plan doc: `docs/devcenter-administrator-2024-02-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DevCenter/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DevCenter/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevCenter/...`

Representative routes:

- `GET /azure/providers/Microsoft.DevCenter/operations?api-version=2024-02-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/devcenters/{devCenterName}?api-version=2024-02-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/projects/{projectName}/pools/{poolName}?api-version=2024-02-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/projects/{projectName}/pools/{poolName}/runHealthChecks?api-version=2024-02-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/networkConnections/{networkConnectionName}/runHealthChecks?api-version=2024-02-01`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/networkConnections/{networkConnectionName}/outboundNetworkDependenciesEndpoints?api-version=2024-02-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/devcenters/{devCenterName}/catalogs/{catalogName}/sync?api-version=2024-02-01`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DevCenter/projects/{projectName}/catalogs/{catalogName}?api-version=2024-02-01`
- `POST /azure/subscriptions/{sub}/providers/Microsoft.DevCenter/checkNameAvailability?api-version=2024-02-01`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- API version is required on all routes (`api-version=2024-02-01`)
- list methods return `value` plus optional `nextLink`
- long-running operations are common for resource create/update/delete/sync/check actions
- errors use `ErrorResponse` envelope in this service version

Observed response patterns:

- `PUT` create/update methods typically return `200`/`201` (with async completion patterns where applicable)
- `PATCH` update/patch methods return `200` or `202` depending on resource family
- action methods (`connect`, `sync`, `runHealthChecks`, name availability checks) return `200`/`202`
- delete methods return `200`/`202`/`204` depending on resource family

Notable payload considerations:

- project resources reference their dev center via `devCenterId`
- attached networks require `networkConnectionId`
- network connections include managed-network and health-check related properties
- project/environment type resources include deployment-target and role-assignment fields
- catalog resources include source settings (`gitHub` / `adoGit`) and sync configuration

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `devcenter_administrator_2024_02_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Dev Center Administrator routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Microsoft.DevCenter/` families
- strict `api-version=2024-02-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Dev Center Administrator route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Core Resource Lifecycles

Implement:

- DevCenters (6)
- Projects (6)
- Pools (6)
- Schedules (5)
- Network Connections (10)
- Operations (`List`) + Operation Statuses (`Get`)

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic list ordering/paging
- idempotent delete semantics by resource type

Exit criteria:

- core Dev Center admin resource CRUD and read surfaces are available

### Stage 3: Catalog and Definition Surfaces

Implement:

- Catalogs (8)
- Project Catalogs (8)
- Environment Definitions (5)
- Project Catalog Environment Definitions (1)
- Dev Box Definitions (7)

Behavior:

- deterministic sync/connect action state transitions
- stable sync and error-details projections
- devcenter and project scoped list/get parity

Exit criteria:

- catalog/definition workflows are callable end-to-end

### Stage 4: Environment and Attached Network Surfaces

Implement:

- Environment Types (5)
- Project Environment Types (5)
- Project Allowed Environment Types (2)
- Attached Networks (6)

Behavior:

- deterministic project-vs-devcenter overlay semantics
- stable environment-type enable/disable projections
- deterministic attached-network association behavior

Exit criteria:

- environment mapping and attached-network surfaces are modeled

### Stage 5: Image and Metadata Surfaces

Implement:

- Galleries (4)
- Images (3)
- Image Versions (2)
- SKUs (1)
- Usages (1)
- Name Availability checks (2)

Behavior:

- deterministic gallery/image/version list/read fixtures
- predictable name-availability decisions for stable test cases
- stable location/subscription metadata fixtures

Exit criteria:

- metadata/image/name-check surfaces return contract-conformant payloads

### Stage 6: Tests and Contract Hardening

Add `internal/server/provider_azure_devcenter_administrator_2024_02_01_test.go` for:

- route ownership, method gating, and api-version enforcement
- happy-path and negative-path payload validation across all operation groups
- list paging and deterministic ordering assertions
- error envelope shape parity (`ErrorResponse`)

Exit criteria:

- deterministic provider behavior is covered with focused tests

### Stage 7: Example and Coverage Enforcement

- keep example aligned to all reviewed operation groups
- enforce service discovery through coverage scripts
- keep strict gates until implementation/test stages are complete

Exit criteria:

- example compiles and demonstrates broad service coverage
- coverage scripts discover service key and plan-doc mapping

## Coverage Script Updates

Add aliases for `devcenter_administrator_2024_02_01` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Minimum alias set:

- `devcenter_administrator_2024_02_01`
- `devcenter-administrator-2024-02-01`
- `devcenter_administrator`
- `devcenter-administrator`
- `devcenter`
- `azure_devcenter_administrator`
- `microsoft_devcenter`
- `rest_devcenter_administrator_2024_02_01`

## Example Scope

Example path:

- `examples/azure/devcenter-administrator-2024-02-01`

Example should:

- use Azure Go SDK core pipeline (`azcore/runtime`, `azcore/policy`) for HTTP execution
- execute representative calls across all operation groups
- print response status and JSON payload for each request
- include cleanup calls for delete-capable resources

## Validation Commands

- `go build ./examples/azure/devcenter-administrator-2024-02-01`
- `python3 scripts/azure-contract-coverage.py --service devcenter-administrator-2024-02-01 --require-service devcenter_administrator_2024_02_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service devcenter-administrator-2024-02-01 --require-service devcenter_administrator_2024_02_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service devcenter-administrator-2024-02-01 --require-service devcenter_administrator_2024_02_01 --fail-on any`
