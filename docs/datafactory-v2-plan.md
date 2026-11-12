# Azure Data Factory V2 (`datafactory_v2`) Staged Plan

## Objective

Emulate Azure Data Factory REST API v2 (`api-version=2018-06-01`) with deterministic local behavior for:

- factory lifecycle
- linked-service, dataset, pipeline, and trigger CRUD
- pipeline-run and activity-run control/query surfaces
- integration-runtime management/control operations
- operations discovery (`providers/Microsoft.DataFactory/operations`)
- an Azure Go SDK example under `examples/azure/datafactory`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Top-level page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/datafactory/v2`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/datafactory/factories`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/linked-services`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/datasets`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/pipelines`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/pipeline-runs`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/activity-runs`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/integration-runtimes`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/triggers`
- `https://learn.microsoft.com/en-us/rest/api/datafactory/operations`

Method pages reviewed (all linked methods across all operation groups):

- Factories (9):
  - `factories/configure-factory-repo`
  - `factories/create-or-update`
  - `factories/delete`
  - `factories/get`
  - `factories/get-data-plane-access`
  - `factories/get-git-hub-access-token`
  - `factories/list`
  - `factories/list-by-resource-group`
  - `factories/update`
- Linked Services (4):
  - `linked-services/create-or-update`
  - `linked-services/delete`
  - `linked-services/get`
  - `linked-services/list-by-factory`
- Datasets (4):
  - `datasets/create-or-update`
  - `datasets/delete`
  - `datasets/get`
  - `datasets/list-by-factory`
- Pipelines (5):
  - `pipelines/create-or-update`
  - `pipelines/create-run`
  - `pipelines/delete`
  - `pipelines/get`
  - `pipelines/list-by-factory`
- Pipeline Runs (3):
  - `pipeline-runs/cancel`
  - `pipeline-runs/get`
  - `pipeline-runs/query-by-factory`
- Activity Runs (1):
  - `activity-runs/query-by-pipeline-run`
- Integration Runtimes (17):
  - `integration-runtimes/create-linked-integration-runtime`
  - `integration-runtimes/create-or-update`
  - `integration-runtimes/delete`
  - `integration-runtimes/get`
  - `integration-runtimes/get-connection-info`
  - `integration-runtimes/get-monitoring-data`
  - `integration-runtimes/get-status`
  - `integration-runtimes/list-auth-keys`
  - `integration-runtimes/list-by-factory`
  - `integration-runtimes/list-outbound-network-dependencies-endpoints`
  - `integration-runtimes/regenerate-auth-key`
  - `integration-runtimes/remove-links`
  - `integration-runtimes/start`
  - `integration-runtimes/stop`
  - `integration-runtimes/sync-credentials`
  - `integration-runtimes/update`
  - `integration-runtimes/upgrade`
- Triggers (10):
  - `triggers/create-or-update`
  - `triggers/delete`
  - `triggers/get`
  - `triggers/get-event-subscription-status`
  - `triggers/list-by-factory`
  - `triggers/query-by-factory`
  - `triggers/start`
  - `triggers/stop`
  - `triggers/subscribe-to-events`
  - `triggers/unsubscribe-from-events`
- Operations (1):
  - `operations/list`

Total methods reviewed: `54`.

## Service Naming

Canonical Stackyard naming:
- service key: `datafactory_v2`
- display name: `Azure Data Factory V2`
- API version: `2018-06-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_datafactory_v2.go`
- tests: `internal/server/provider_azure_datafactory_v2_test.go`
- example: `examples/azure/datafactory`
- plan doc: `docs/datafactory-v2-plan.md`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:
- all methods use `api-version=2018-06-01`
- management-plane resource paths under `management.azure.com`
- security model: `azure_auth` (AAD OAuth2) documented on all methods
- error envelope type: `CloudError` for unexpected errors

Observed success/response patterns:
- create/update CRUD methods return `200 OK` with typed resource payload
- delete methods return `200 OK` (deleted) or `204 NoContent` (already absent)
- query/list methods return `200 OK` with paged/list payloads
- some control operations return `202 Accepted` with `Location` and `Retry-After`:
  - `integration-runtimes/start`
  - `integration-runtimes/stop`
  - `triggers/subscribe-to-events`
  - `triggers/unsubscribe-from-events`
- several GETs document conditional request behavior (`resource not modified`) in other status codes

Notable input/behavior details to model:
- optional `If-Match` on create/update for several entity types
- trigger start/stop returns `Location` header on success
- pipeline/activity query methods are POST with filter/time-window body contracts
- pipeline run cancel supports optional `isRecursive` query parameter

## Route Envelope For Emulator

Management route base to emulate:
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataFactory/...`
- `/azure/providers/Microsoft.DataFactory/operations`

Representative routes:
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}?api-version=2018-06-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/linkedservices/{name}?api-version=2018-06-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/datasets/{name}?api-version=2018-06-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/pipelines/{name}?api-version=2018-06-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/pipelines/{name}/createRun?api-version=2018-06-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/pipelineruns/queryByFactory?api-version=2018-06-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/pipelineruns/{runId}/queryActivityruns?api-version=2018-06-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/triggers/{name}?api-version=2018-06-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataFactory/factories/{factory}/triggers/{name}/start?api-version=2018-06-01`
- `GET /azure/providers/Microsoft.DataFactory/operations?api-version=2018-06-01`

## Staged Implementation Plan

### Stage 0: Coverage And Planning Wiring

- add this plan doc
- add aliases for `datafactory_v2` to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this doc in `SERVICE_PLAN_DOCS`

Exit criteria:
- coverage scripts can resolve `datafactory_v2` and common aliases
- doc coverage maps `datafactory_v2` to this plan doc

### Stage 1: Router Skeleton + API-Version Gate

Implement:
- route ownership for `providers/Microsoft.DataFactory/*`
- strict `api-version=2018-06-01` validation
- deterministic `400` on missing/invalid API version
- `405` for unsupported methods on owned routes

Exit criteria:
- Data Factory routes are owned by Azure provider router
- unsupported operations return deterministic method/version errors

### Stage 2: Core Resource CRUD

Implement first-class CRUD for:
- Factories
- Linked Services
- Datasets
- Pipelines
- Triggers

Behavior:
- `PUT` create/update returns `200` with canonical fixture
- `GET` returns stored fixture
- `LIST` by scope returns stable ordering + optional continuation support
- `DELETE` returns `200` or `204` idempotently

Exit criteria:
- all CRUD/list methods in these groups return non-`NotImplemented`
- fixtures include stable `id`, `name`, `type`, `etag`, timestamps

### Stage 3: Pipeline Execution Surfaces

Implement:
- `Pipelines_CreateRun`
- `PipelineRuns_Get`
- `PipelineRuns_QueryByFactory`
- `PipelineRuns_Cancel`
- `ActivityRuns_QueryByPipelineRun`

Behavior:
- run creation returns deterministic run IDs
- query responses filter by time window and status fields
- cancel updates run status and records recursive intent

Exit criteria:
- end-to-end pipeline run lifecycle is emulated and queryable

### Stage 4: Trigger Control And Subscriptions

Implement:
- `Triggers_Start`, `Triggers_Stop`
- `Triggers_SubscribeToEvents`, `Triggers_UnsubscribeFromEvents`
- `Triggers_GetEventSubscriptionStatus`
- `Triggers_QueryByFactory`

Behavior:
- trigger runtime state transitions
- asynchronous-style responses where documented (`202` + headers)
- subscription status fixture lifecycle

Exit criteria:
- trigger control endpoints model documented synchronous/asynchronous behavior

### Stage 5: Integration Runtime Surface

Implement integration-runtime resource + control methods:
- create/update/get/delete/list
- auth key operations
- connection/monitoring/status endpoints
- start/stop/sync/upgrade/remove-links/create-linked-IR
- outbound dependency endpoints

Behavior:
- managed/self-hosted status fixtures
- long-running style status for start/stop
- deterministic auth key rotation semantics

Exit criteria:
- all 17 integration-runtime methods return modeled responses

### Stage 6: Provider Operations Discovery

Implement:
- `Operations_List`

Behavior:
- return static+extensible operation metadata set aligned with implemented methods

Exit criteria:
- operations list reflects emulator-supported Data Factory actions

### Stage 7: Tests

Add focused provider contract tests:
- CRUD/list per core entity
- run/query/cancel/activity-run flows
- trigger and integration-runtime control state transitions
- version/validation/negative-path coverage
- async (`202`) header behavior where applicable

Exit criteria:
- `go test ./internal/server -run 'TestAzureDataFactoryV2' -count=1` passes

### Stage 8: Example

Add `examples/azure/datafactory`:
- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example flow:
- list provider operations
- create factory + linked service + dataset + pipeline
- create pipeline run + query runs/activity-runs
- create/start/stop trigger
- cleanup resources

SDK approach:
- use Azure Go SDK core runtime pipeline (`azcore/runtime` + `azcore/policy`)

Exit criteria:
- `go build ./examples/azure/datafactory` passes

### Stage 9: Validation

Run once implementation stages are complete:
- `go test ./internal/server -run 'TestAzureDataFactoryV2' -count=1`
- `go build ./examples/azure/datafactory`
- `python3 scripts/azure-contract-coverage.py --service datafactory_v2 --require-service datafactory_v2 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datafactory_v2 --require-service datafactory_v2 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datafactory_v2 --require-service datafactory_v2 --fail-on any`

Success target:
- no `NotImplemented` for the 54 reviewed Data Factory v2 methods
- coverage gates pass for `datafactory_v2`
