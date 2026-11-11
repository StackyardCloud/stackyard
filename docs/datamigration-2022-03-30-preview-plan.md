# Azure Data Migration (`datamigration_2022_03_30_preview`) Staged Plan

## Objective

Emulate Azure Data Migration REST API (`api-version=2022-03-30-preview`) with deterministic local behavior for:

- service lifecycle and control actions
- service-task lifecycle and service-task commands
- project lifecycle
- task lifecycle and task commands
- project file lifecycle and file transfer helper actions
- provider operations discovery
- an Azure Go SDK example under `examples/azure/datamigration-2022-03-30-preview`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Top-level and operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/datamigration/operation-groups?view=rest-datamigration-2022-03-30-preview`
- `https://learn.microsoft.com/en-us/rest/api/datamigration/services?view=rest-datamigration-2022-03-30-preview`
- `https://learn.microsoft.com/en-us/rest/api/datamigration/service-tasks?view=rest-datamigration-2022-03-30-preview`
- `https://learn.microsoft.com/en-us/rest/api/datamigration/tasks?view=rest-datamigration-2022-03-30-preview`
- `https://learn.microsoft.com/en-us/rest/api/datamigration/files?view=rest-datamigration-2022-03-30-preview`

Method pages reviewed:

- Services:
  - `.../services/check-children-name-availability`
  - `.../services/check-name-availability`
  - `.../services/check-status`
  - `.../services/create-or-update`
  - `.../services/delete`
  - `.../services/get`
  - `.../services/list`
  - `.../services/list-by-resource-group`
  - `.../resource-skus/list-skus` (linked from `List Skus`)
  - `.../services/start`
  - `.../services/stop`
  - `.../services/update`
- Projects:
  - `.../projects/create-or-update`
  - `.../projects/delete`
  - `.../projects/get`
  - `.../projects/list`
  - `.../projects/update`
- Service Tasks:
  - `.../service-tasks/cancel`
  - `.../service-tasks/command`
  - `.../service-tasks/create-or-update`
  - `.../service-tasks/delete`
  - `.../service-tasks/get`
  - `.../service-tasks/list`
  - `.../service-tasks/update`
- Tasks:
  - `.../tasks/cancel`
  - `.../tasks/command`
  - `.../tasks/create-or-update`
  - `.../tasks/delete`
  - `.../tasks/get`
  - `.../tasks/list`
  - `.../tasks/update`
- Files:
  - `.../files/create-or-update`
  - `.../files/delete`
  - `.../files/get`
  - `.../files/list`
  - `.../files/read`
  - `.../files/read-write`
  - `.../files/update`
- Operations:
  - `.../operations/list` (Learn currently serves this as `view=rest-datamigration-2022-01-30-preview`)

Reviewed operation count: `39`.

## Service Naming

Canonical Stackyard naming:

- service key: `datamigration_2022_03_30_preview`
- display name: `Azure Data Migration`
- API version: `2022-03-30-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_datamigration_2022_03_30_preview.go`
- tests: `internal/server/provider_azure_datamigration_2022_03_30_preview_test.go`
- example: `examples/azure/datamigration-2022-03-30-preview`
- plan doc: `docs/datamigration-2022-03-30-preview-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataMigration/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{groupName}/providers/Microsoft.DataMigration/...`
- `/azure/providers/Microsoft.DataMigration/operations`

Representative routes:

- `POST /azure/subscriptions/{sub}/providers/Microsoft.DataMigration/locations/{location}/checkNameAvailability?api-version=2022-03-30-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}?api-version=2022-03-30-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/start?api-version=2022-03-30-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/serviceTasks/{serviceTask}?api-version=2022-03-30-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/serviceTasks/{serviceTask}/command?api-version=2022-03-30-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/projects/{project}?api-version=2022-03-30-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/projects/{project}/tasks/{task}?api-version=2022-03-30-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/projects/{project}/tasks/{task}/command?api-version=2022-03-30-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/projects/{project}/files/{file}?api-version=2022-03-30-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/projects/{project}/files/{file}/read?api-version=2022-03-30-preview`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataMigration/services/{service}/projects/{project}/files/{file}/readwrite?api-version=2022-03-30-preview`
- `GET /azure/providers/Microsoft.DataMigration/operations?api-version=2022-03-30-preview`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed method pages require `api-version=2022-03-30-preview` (except Learn rendering of `operations/list`)
- OAuth2 `azure_auth` is used for authentication in method docs
- standard ARM error envelope is documented as `ApiError`/`ODataError`

Observed status-code patterns:

- create/update/get/list methods are mainly `200 OK` and/or `201 Created`
- deletes commonly include `200`, `202`, and idempotent `204`
- service control methods (`start`/`stop`) expose `200` and `202`
- service-task and task command/cancel endpoints are `POST` actions with command payloads
- task command and file read/read-write helper methods use `POST` and return task/file-specific result payloads

Notable modeling details:

- task list supports `taskType` filtering
- task command request includes `commandType`/properties contract
- services and projects expose nested polymorphic task/project/file models in definitions
- list operations are paged (`value` + optional continuation/`nextLink` semantics)

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `datamigration_2022_03_30_preview` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add a stage-0 provider skeleton and wire it in `internal/server/http.go`

Exit criteria:

- coverage scripts resolve the new service key and aliases
- doc coverage maps this key to this plan doc
- router owns Data Migration route prefix for future stages

### Stage 1: Router Ownership and API-Version Gate

Implement:

- ownership for `/providers/Microsoft.DataMigration/` route families
- strict `api-version=2022-03-30-preview` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- service paths are consistently owned
- invalid API versions fail deterministically

### Stage 2: Services Surface

Implement services operations:

- `CheckChildrenNameAvailability`
- `CheckNameAvailability`
- `CheckStatus`
- `CreateOrUpdate`
- `Delete`
- `Get`
- `List`
- `ListByResourceGroup`
- `ListSkus`
- `Start`
- `Stop`
- `Update`

Behavior:

- stable service resource fixtures (`id`, `name`, `type`, `location`, `properties`, `sku`, `tags`)
- deterministic start/stop transitions
- list ordering + pagination envelope

Exit criteria:

- no `NotImplemented` for services operation set
- deterministic lifecycle + list responses

### Stage 3: Projects, Service Tasks, Tasks, and Files Surfaces

Implement project operations:

- `CreateOrUpdate`, `Delete`, `Get`, `List`, `Update`

Implement service-task operations:

- `Cancel`, `Command`, `CreateOrUpdate`, `Delete`, `Get`, `List`, `Update`

Implement task operations:

- `Cancel`, `Command`, `CreateOrUpdate`, `Delete`, `Get`, `List`, `Update`

Implement file operations:

- `CreateOrUpdate`, `Delete`, `Get`, `List`, `Read`, `ReadWrite`, `Update`

Behavior:

- nested stores keyed by `{subscription, rg, service, project, name}`
- separate service-task and project-task state stores
- deterministic task state transitions (`Queued`, `Running`, `Canceled`, `Succeeded` fixtures)
- deterministic file read/read-write result fixtures

Exit criteria:

- no `NotImplemented` for the reviewed project/service-task/task/file methods
- task command and file read/read-write return stable modeled outputs

### Stage 4: Operations Discovery + Error Fidelity

Implement:

- `Operations_List`
- service-specific validation failures and modeled `ApiError` payloads for invalid input or unsupported transitions

Exit criteria:

- operations metadata endpoint available
- negative-path errors are deterministic and shape-compatible

### Stage 5: Tests

Add provider tests for:

- API-version and method guards
- service lifecycle + start/stop
- project/service-task/task/file CRUD/list
- task cancel/command
- file read/read-write
- operations list and pagination/idempotent delete behavior

Exit criteria:

- `go test ./internal/server -run 'TestAzureDataMigration20220330Preview' -count=1` passes

### Stage 6: Example and Validation

Add `examples/azure/datamigration-2022-03-30-preview`:

- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example flow should cover representative operations from:

- Services
- Service Tasks
- Projects
- Tasks
- Files
- Operations

Validation commands:

- `go build ./examples/azure/datamigration-2022-03-30-preview`
- `python3 scripts/azure-contract-coverage.py --service datamigration --require-service datamigration_2022_03_30_preview --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datamigration --require-service datamigration_2022_03_30_preview --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datamigration --require-service datamigration_2022_03_30_preview --fail-on any`

Success target:

- coverage scripts resolve and report the service key
- example compiles and demonstrates the intended API families
