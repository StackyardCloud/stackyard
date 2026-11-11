# Azure Data Replication (`datareplication_2026_05_01`) Staged Plan

## Objective

Emulate Azure Data Replication REST API (`api-version=2026-05-01`) with deterministic local behavior for:

- vault, fabric, and child-resource lifecycle flows
- replication policy, extension, and protected-item workflows
- private endpoint and private link management surfaces
- events, jobs, and recovery point query surfaces
- operation result polling surfaces
- an Azure Go SDK example under `examples/azure/datareplication-2026-05-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/datareplication/operation-groups?view=rest-datareplication-2026-05-01`

Operation-group pages reviewed:

- `.../check-name-availability?view=rest-datareplication-2024-09-01`
- `.../deployment-preflight?view=rest-datareplication-2024-09-01` (operation page reviewed directly)
- `.../email-configuration?view=rest-datareplication-2024-09-01`
- `.../event?view=rest-datareplication-2024-09-01`
- `.../fabric?view=rest-datareplication-2026-05-01`
- `.../fabric-agent?view=rest-datareplication-2024-09-01`
- `.../job?view=rest-datareplication-2024-09-01`
- `.../location-based-operation-results?view=rest-datareplication-2024-09-01` (operation page reviewed directly)
- `.../operation-results?view=rest-datareplication-2024-09-01` (operation page reviewed directly)
- `.../operations/list?view=rest-datareplication-2024-09-01`
- `.../policy?view=rest-datareplication-2024-09-01`
- `.../private-endpoint-connection-proxies?view=rest-datareplication-2024-09-01` (operation pages reviewed directly)
- `.../private-endpoint-connections?view=rest-datareplication-2024-09-01` (operation pages reviewed directly)
- `.../private-link-resources?view=rest-datareplication-2024-09-01`
- `.../protected-item?view=rest-datareplication-2024-09-01`
- `.../recovery-point?view=rest-datareplication-2024-09-01`
- `.../replication-extension?view=rest-datareplication-2024-09-01`
- `.../vault?view=rest-datareplication-2024-09-01`

Representative method pages reviewed directly include:

- `check-name-availability/post`
- `deployment-preflight/post`
- `email-configuration/create`, `email-configuration/get`, `email-configuration/list`
- `event/get`, `event/list`
- `fabric/create`, `fabric/delete`, `fabric/get`, `fabric/list`, `fabric/list-by-subscription`, `fabric/update`
- `fabric-agent/create`, `fabric-agent/delete`, `fabric-agent/get`, `fabric-agent/list`
- `job/get`, `job/list`
- `location-based-operation-results/get`
- `operation-results/get`
- `operations/list`
- `policy/create`, `policy/delete`, `policy/get`, `policy/list`
- `private-endpoint-connection-proxies/create`, `private-endpoint-connection-proxies/delete`, `private-endpoint-connection-proxies/get`, `private-endpoint-connection-proxies/list`
- `private-endpoint-connections/get`, `private-endpoint-connections/list`
- `private-link-resources/list`
- `protected-item/create`, `protected-item/delete`, `protected-item/get`, `protected-item/list`, `protected-item/planned-failover`, `protected-item/update`
- `recovery-point/get`, `recovery-point/list`
- `replication-extension/create`, `replication-extension/delete`, `replication-extension/get`, `replication-extension/list`
- `vault/create`, `vault/delete`, `vault/get`, `vault/list`, `vault/list-by-subscription`, `vault/update`

Notes on source behavior:

- Several `2026-05-01` method pages are available directly (for example `fabric/update`, `fabric-agent/create`, `vault/delete`).
- In this environment, some operation-group index pages intermittently return cache/authorization wrappers; method-level pages were used to complete coverage.
- The Azure SDK spec index confirms package availability for this version:
  - `recoveryservicesdatareplication/resource-manager/Microsoft.DataReplication/DataReplication/README.md-[package-2026-05-01]`

## Surface Summary

Reviewed operation groups and methods:

- Check Name Availability (1):
  - `Post`
- Deployment Preflight (1):
  - `Post`
- Email Configuration (3):
  - `Create`, `Get`, `List`
- Event (2):
  - `Get`, `List`
- Fabric (6):
  - `Create`, `Delete`, `Get`, `List`, `List By Subscription`, `Update`
- Fabric Agent (4):
  - `Create`, `Delete`, `Get`, `List`
- Job (2):
  - `Get`, `List`
- Location Based Operation Results (1):
  - `Get`
- Operation Results (1):
  - `Get`
- Operations (1):
  - `List`
- Policy (4):
  - `Create`, `Delete`, `Get`, `List`
- Private Endpoint Connection Proxies (4):
  - `Create`, `Delete`, `Get`, `List`
- Private Endpoint Connections (2):
  - `Get`, `List`
- Private Link Resources (1):
  - `List`
- Protected Item (6):
  - `Create`, `Delete`, `Get`, `List`, `Planned Failover`, `Update`
- Recovery Point (2):
  - `Get`, `List`
- Replication Extension (4):
  - `Create`, `Delete`, `Get`, `List`
- Vault (6):
  - `Create`, `Delete`, `Get`, `List`, `List By Subscription`, `Update`

Total reviewed methods: `51`.

## Service Naming

Canonical Stackyard naming:

- service key: `datareplication_2026_05_01`
- display name: `Azure Data Replication`
- API version: `2026-05-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_datareplication_2026_05_01.go`
- tests: `internal/server/provider_azure_datareplication_2026_05_01_test.go`
- example: `examples/azure/datareplication-2026-05-01`
- plan doc: `docs/datareplication-2026-05-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DataReplication/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataReplication/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataReplication/...`

Representative routes:

- `POST /azure/subscriptions/{sub}/providers/Microsoft.DataReplication/locations/{location}/checkNameAvailability?api-version=2026-05-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationVaults/{vaultName}?api-version=2026-05-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationFabrics/{fabricName}?api-version=2026-05-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationFabrics/{fabricName}/fabricAgents/{fabricAgentName}?api-version=2026-05-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationVaults/{vaultName}/replicationPolicies/{policyName}?api-version=2026-05-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationVaults/{vaultName}/protectedItems/{protectedItemName}?api-version=2026-05-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationVaults/{vaultName}/protectedItems/{protectedItemName}/plannedFailover?api-version=2026-05-01`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataReplication/replicationVaults/{vaultName}/jobs/{jobName}?api-version=2026-05-01`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.DataReplication/locations/{location}/operationResults/{operationId}?api-version=2026-05-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- operations are ARM OAuth2 (`azure_auth`, `user_impersonation`)
- management-plane naming and envelopes follow ARM conventions
- create/update and many action flows are long-running operation capable
- common error envelope is `ErrorResponse` with nested `ErrorDetail`

Observed response patterns:

- create/update methods frequently return `200` / `201`
- delete methods commonly return `202` and idempotent `204`
- list methods return `value` with optional `nextLink`
- LRO hints include `Azure-AsyncOperation`, `Location`, and `Retry-After`

Notable request/flow specifics:

- `protected-item/planned-failover` is explicit action routing
- `deployment-preflight/post` validates resources pre-deployment
- private endpoint proxy and connection groups have distinct resource paths
- operation-result polling exists in both provider-wide and location-scoped forms

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `datareplication_2026_05_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve the service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Data Replication paths to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- ownership for `/providers/Microsoft.DataReplication/` route families
- strict `api-version=2026-05-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- all Data Replication route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Core Vault/Fabric/Policy CRUD

Implement:

- Vault: create/delete/get/list/list-by-subscription/update
- Fabric: create/delete/get/list/list-by-subscription/update
- Policy: create/delete/get/list

Behavior:

- canonical ARM resource envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic ordering and stable paging in list responses
- idempotent delete semantics

Exit criteria:

- no `NotImplemented` for core vault/fabric/policy endpoints

### Stage 3: Agents, Extensions, and Protected Items

Implement:

- Fabric Agent: create/delete/get/list
- Replication Extension: create/delete/get/list
- Protected Item: create/delete/get/list/update/planned-failover

Behavior:

- deterministic state transitions for protection and failover actions
- stable custom-properties envelopes by provider type discriminators
- consistent LRO headers on action and lifecycle operations

Exit criteria:

- protected-item flow is functionally modeled end-to-end

### Stage 4: Jobs, Events, Recovery, Private Endpoints, and Polling

Implement:

- Job: get/list
- Event: get/list
- Recovery Point: get/list
- Email Configuration: create/get/list
- Private Endpoint Connections: get/list
- Private Endpoint Connection Proxies: create/delete/get/list
- Private Link Resources: list
- Check Name Availability: post
- Deployment Preflight: post
- Operation Results: get
- Location Based Operation Results: get
- Operations: list

Exit criteria:

- full operation-group coverage across all 51 reviewed methods
- asynchronous operation tracking endpoints return deterministic status payloads

### Stage 5: Tests

Add provider tests for:

- API-version and method gating
- vault/fabric/policy/protected-item lifecycle flows
- planned failover action semantics
- job/event/recovery point query behavior
- private endpoint proxy and connection flows
- operation-result polling and idempotent delete behavior

Exit criteria:

- `go test ./internal/server -run 'TestAzureDataReplication20260501' -count=1` passes

### Stage 6: Example and Validation

Add `examples/azure/datareplication-2026-05-01`:

- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example should cover representative calls across:

- provider operations and name availability
- vault/fabric/fabric-agent lifecycle
- policy, extension, and protected item flows
- jobs/events/recovery points
- private endpoint groups
- operation result polling endpoints

Validation commands:

- `go build ./examples/azure/datareplication-2026-05-01`
- `python3 scripts/azure-contract-coverage.py --service datareplication --require-service datareplication_2026_05_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datareplication --require-service datareplication_2026_05_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datareplication --require-service datareplication_2026_05_01 --fail-on any`

Success target:

- coverage scripts resolve and report `datareplication_2026_05_01`
- example compiles and exercises the staged surface contract
