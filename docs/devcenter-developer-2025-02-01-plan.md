# Azure Dev Center Developer (`devcenter_developer_2025_02_01`) Staged Plan

## Objective

Emulate Azure Dev Center Developer REST API (`api-version=2025-02-01`) with deterministic local behavior for:

- project discovery and project ability lookups
- environment lifecycle, environment actions, and operation logs
- dev box lifecycle, actions, customization surfaces, snapshot flows, and remote connection retrieval
- project-scoped operation-status polling
- an Azure Go SDK example under `examples/azure/devcenter-developer-2025-02-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/devcenter/developer/operation-groups?view=rest-devcenter-developer-2025-02-01`

Operation-group pages reviewed:

- `.../dev-center?view=rest-devcenter-developer-2025-02-01`
- `.../environments?view=rest-devcenter-developer-2025-02-01`
- `.../dev-boxes?view=rest-devcenter-developer-2025-02-01`
- `.../operation-statuses?view=rest-devcenter-developer-2025-02-01`

Method pages reviewed directly (complete operation set, grouped):

- Dev Center: `get-project`, `get-project-abilities`, `list-projects`
- Environments: `create-or-replace-environment`, `delay-action`, `delete-environment`, `get-action`, `get-catalog`, `get-environment-by-user`, `get-environment-definition`, `get-environment-type-abilities`, `get-environment-types`, `get-logs-by-operation`, `get-operation`, `get-outputs`, `list-actions`, `list-catalogs-by-project`, `list-environment-definitions-by-catalog`, `list-environment-definitions-by-project`, `list-environments`, `list-environments-by-user`, `list-environment-types`, `list-operations`, `patch-environment`, `skip-action`
- Dev Boxes: `capture-snapshot`, `create-customization-group`, `create-dev-box`, `delay-action`, `delay-actions`, `delete-dev-box`, `get-action`, `get-customization-group`, `get-customization-task-definitions`, `get-customization-task-log`, `get-dev-box-by-user`, `get-imaging-task-log`, `get-operation`, `get-pool`, `get-remote-connection`, `get-schedule-by-pool`, `get-snapshot`, `list-actions`, `list-all-dev-boxes`, `list-all-dev-boxes-by-user`, `list-customization-groups`, `list-customization-task-definitions-by-project`, `list-dev-boxes-by-user`, `list-operations`, `list-pools`, `list-schedules-by-pool`, `list-schedules-by-project`, `list-snapshots`, `repair-dev-box`, `restart-dev-box`, `restore-snapshot`, `skip-action`, `start-dev-box`, `stop-dev-box`, `validate-customization-tasks-action`
- Operation Statuses: `get`

Notes from source retrieval:

- all 61 method pages were traversed from the operation-group links
- the `delay-actions` page intermittently failed direct page fetch in Learn; the same canonical URL and contract details were confirmed through the operation-group link metadata and cached Learn method snippet

## Surface Summary

Reviewed operation groups and methods:

- Dev Center (3):
  - `Get Project`, `Get Project Abilities`, `List Projects`
- Environments (22):
  - `Create Or Replace Environment`, `Delay Action`, `Delete Environment`, `Get Action`, `Get Catalog`, `Get Environment By User`, `Get Environment Definition`, `Get Environment Type Abilities`, `Get Environment Types`, `Get Logs By Operation`, `Get Operation`, `Get Outputs`, `List Actions`, `List Catalogs By Project`, `List Environment Definitions By Catalog`, `List Environment Definitions By Project`, `List Environments`, `List Environments By User`, `List Environment Types`, `List Operations`, `Patch Environment`, `Skip Action`
- Dev Boxes (35):
  - `Capture Snapshot`, `Create Customization Group`, `Create Dev Box`, `Delay Action`, `Delay Actions`, `Delete Dev Box`, `Get Action`, `Get Customization Group`, `Get Customization Task Definitions`, `Get Customization Task Log`, `Get Dev Box By User`, `Get Imaging Task Log`, `Get Operation`, `Get Pool`, `Get Remote Connection`, `Get Schedule By Pool`, `Get Snapshot`, `List Actions`, `List All Dev Boxes`, `List All Dev Boxes By User`, `List Customization Groups`, `List Customization Task Definitions By Project`, `List Dev Boxes By User`, `List Operations`, `List Pools`, `List Schedules By Pool`, `List Schedules By Project`, `List Snapshots`, `Repair Dev Box`, `Restart Dev Box`, `Restore Snapshot`, `Skip Action`, `Start Dev Box`, `Stop Dev Box`, `Validate Customization Tasks Action`
- Operation Statuses (1):
  - `Get`

Total reviewed methods: `61`.

## Service Naming

Canonical Stackyard naming:

- service key: `devcenter_developer_2025_02_01`
- display name: `Azure Dev Center Developer`
- API version: `2025-02-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_devcenter_developer_2025_02_01.go`
- tests: `internal/server/provider_azure_devcenter_developer_2025_02_01_test.go`
- example: `examples/azure/devcenter-developer-2025-02-01`
- plan doc: `docs/devcenter-developer-2025-02-01-plan.md`

## Route Envelope

Learn route template shape:

- `{endpoint}/projects/...`
- `{endpoint}/users/...`
- `{endpoint}/devboxes...`

Stackyard emulator envelope for this data plane:

- `/azure/devcenter-developer/2025-02-01/...`

Representative routes from reviewed method pages:

- `GET /azure/devcenter-developer/2025-02-01/projects/{projectName}?api-version=2025-02-01`
- `GET /azure/devcenter-developer/2025-02-01/projects?api-version=2025-02-01`
- `GET /azure/devcenter-developer/2025-02-01/projects/{projectName}/users/{userId}/abilities?api-version=2025-02-01`
- `PUT /azure/devcenter-developer/2025-02-01/projects/{projectName}/users/{userId}/environments/{environmentName}?api-version=2025-02-01`
- `DELETE /azure/devcenter-developer/2025-02-01/projects/{projectName}/users/{userId}/environments/{environmentName}?api-version=2025-02-01&force={force}`
- `GET /azure/devcenter-developer/2025-02-01/projects/{projectName}/users/{userId}/environments/{environmentName}/operations?api-version=2025-02-01`
- `GET /azure/devcenter-developer/2025-02-01/projects/{projectName}/users/{userId}/devboxes/{devBoxName}/operations/{operationId}?api-version=2025-02-01`
- `GET /azure/devcenter-developer/2025-02-01/devboxes?api-version=2025-02-01`
- `GET /azure/devcenter-developer/2025-02-01/users/{userId}/devboxes?api-version=2025-02-01`
- `POST /azure/devcenter-developer/2025-02-01/projects/{projectName}/users/{userId}/devboxes/{devBoxName}/actions:delay?api-version=2025-02-01&until={until}`
- `POST /azure/devcenter-developer/2025-02-01/projects/{projectName}/customizationTasks:validateGroup?api-version=2025-02-01`
- `GET /azure/devcenter-developer/2025-02-01/projects/{projectName}/operationstatuses/{operationId}?api-version=2025-02-01`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:

- `api-version` is required on all routes and is `2025-02-01`
- OAuth2 scope shown on method pages: `devcenter.azure.com/.default`
- error envelope is `Azure.Core.Foundations.ErrorResponse`
- error header exposed by pages: `x-ms-error-code`

Observed response patterns:

- list operations use paged envelopes (`value` + optional `nextLink`)
- create/update actions commonly return `200` or `201`
- long-running operations expose `Location` and `Operation-Location` headers for accepted/created flows
- delete/action operations may return `200`, `202`, or `204` depending on operation semantics
- log endpoints (`get-logs-by-operation`, `get-customization-task-log`, `get-imaging-task-log`) return textual payloads (`text/plain` and/or `application/json` media types)

Notable payload/shape considerations from method pages:

- environment create requires `environmentType`, `catalogName`, and `environmentDefinitionName`
- environment patch uses `application/merge-patch+json`
- dev box create supports idempotent create/replace semantics with `200/201`
- operation-status polling is project-scoped: `/projects/{projectName}/operationstatuses/{operationId}`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `devcenter_developer_2025_02_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatch includes a dedicated Dev Center Developer branch

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/azure/devcenter-developer/2025-02-01`
- strict `api-version=2025-02-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `404/405` for unsupported paths and methods

Exit criteria:

- all Developer data-plane requests under prefix are deterministically owned
- invalid version/method/path behavior is stable and testable

### Stage 2: Dev Center and Project Read Surfaces

Implement:

- Dev Center group (3):
  - `Get Project`, `Get Project Abilities`, `List Projects`

Behavior:

- deterministic project fixtures
- deterministic abilities matrix for `me` and UUID-like user IDs
- paged project list shape with stable ordering

Exit criteria:

- project discovery and abilities endpoints are available with documented envelopes

### Stage 3: Environment Lifecycle and Operations

Implement:

- Environments group (22):
  - core CRUD + patch, actions (`Delay`, `Skip`), outputs, catalogs/definitions listings, operation listing/details/logs

Behavior:

- environment state machine (`Creating`, `Succeeded`, `Deleting`, `Failed`) with deterministic transitions
- optional `force` semantics for delete
- deterministic operation records and operation log payloads

Exit criteria:

- environment end-to-end workflow is represented with consistent operation histories

### Stage 4: Dev Box Lifecycle, Actions, and Customization

Implement:

- Dev Boxes group (35)
  - create/get/list/delete, power operations, repair, snapshots, remote connection
  - actions APIs and action-delay flows
  - customization groups/tasks definitions/validation/log retrieval

Behavior:

- deterministic Dev Box provisioning and power states
- action delay/skip semantics with schedule window validation
- snapshot and restore operation records linked into operations list

Exit criteria:

- Dev Box lifecycle/action/customization coverage is complete at route-contract level

### Stage 5: Operation Status Polling and LRO Coherence

Implement:

- Operation Statuses group (1): `Get`

Behavior:

- every asynchronous operation emits stable operation IDs
- `Location` / `Operation-Location` headers resolve to a readable operation-status document

Exit criteria:

- operation status polling works across environment and dev box long-running flows

### Stage 6: Test Matrix, Error Fidelity, and Contract Hardening

Implement:

- comprehensive server tests per operation group
- contract-level checks for:
  - required parameters
  - ID/name regex constraints
  - pagination
  - documented error envelope and headers

Exit criteria:

- provider behavior is deterministic
- contract/error responses are aligned with reviewed method-page contracts
- examples run cleanly once stage implementation moves beyond skeleton
