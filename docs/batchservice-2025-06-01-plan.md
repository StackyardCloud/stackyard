# Azure Batch Service (`batchservice-2025-06-01`) Staged Plan

## Objective

Emulate the Azure Batch Service account-endpoint API for `2025-06-01` with deterministic local behavior for:

- account-scoped `applications` reads
- `jobs` lifecycle and job actions
- `jobSchedules` lifecycle and schedule actions
- `pools` lifecycle, resize, and image/usage helper reads
- `nodes` lifecycle actions, file reads, extension reads, and node-user mutations
- `tasks` lifecycle, collection create, task actions, subtask reads, and task file reads
- an Azure Go SDK example under `examples/azure/batchservice-2025-06-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- This is the Azure Batch runtime data plane rooted at a Batch account endpoint such as `https://{account}.{region}.batch.azure.com`.
- It is not the ARM `Microsoft.Batch` management plane already implemented in this repo under `batch_management`.
- The routes are direct service routes like `/jobs`, `/pools`, and `/jobs/{jobId}/tasks`, not `/subscriptions/.../providers/...`.

Important docs note:

- The reviewed `2025-06-01` pages clearly cover `applications`, `jobs`, `job schedules`, `pools`, `nodes`, and `tasks`.
- During review, certificate-related search results surfaced only on a `2024-07-01` page rather than a clean `2025-06-01` page.
- This plan therefore treats certificates as a later reconciliation item instead of a first implementation stage.

Important SDK note:

- I did not find a clearly matching official typed Azure Go SDK package for the Batch Service data plane during review.
- The official Go SDK package I did confirm is ARM-only: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch`.
- The future example should therefore use the official Azure Go SDK core pipeline packages `azcore/runtime` and `azcore/policy` rather than inventing a fake typed Batch Service client.

## Primary References Reviewed

Primary operation-groups page requested and reviewed:

- `https://learn.microsoft.com/en-us/rest/api/batchservice/operation-groups?view=rest-batchservice-2025-06-01`

Operation-group and linked method pages reviewed from that surface:

- `https://learn.microsoft.com/en-us/rest/api/batchservice/applications/list-applications?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/applications/get-application?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/jobs?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/jobs/create-job?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/jobs/get-job?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/jobs/list-jobs?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/jobs/enable-job?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/jobs/list-jobs-from-schedule?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/get-job-schedule?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/list-job-schedules?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/job-schedule-exists?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/replace-job-schedule?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/update-job-schedule?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/enable-job-schedule?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/job-schedules/disable-job-schedule?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/pools?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/pools/create-pool?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/pools/get-pool?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/pools/list-pools?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/pools/list-pool-usage-metrics?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/pools/list-supported-images?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/list-nodes?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/create-node-user?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/start-node?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/enable-node-scheduling?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/list-node-extensions?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/list-node-files?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/nodes/get-node-file?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/tasks?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/tasks/create-task?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/tasks/list-tasks?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/tasks/list-task-files?view=rest-batchservice-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/batchservice/tasks/get-task-file-properties?view=rest-batchservice-2025-06-01`

Supplemental SDK reference reviewed:

- `https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch`

## Service Naming

Canonical Stackyard naming:

- service key: `batch_service`
- display name: `Azure Batch Service`
- API version token: `2025-06-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_batch_service.go`
- tests: `internal/server/provider_azure_batch_service_test.go`
- example: `examples/azure/batchservice-2025-06-01`
- plan doc: `docs/batchservice-2025-06-01-plan.md`

Recommended coverage aliases:

- `batch_service`
- `batchservice`
- `batch-service`
- `azure_batch_service`
- `azure_batchservice`
- `batch_runtime`
- `batch_data_plane`
- `rest_batchservice_2025_06_01`
- `batchservice_2025_06_01`

## Surface Summary

The reviewed REST surface is a service-endpoint API rooted at a Batch account endpoint:

- `GET {endpoint}/applications`
- `GET {endpoint}/jobs`
- `GET {endpoint}/jobschedules`
- `GET {endpoint}/pools`
- `GET {endpoint}/supportedimages`
- `GET {endpoint}/poolusagemetrics`
- nested `nodes` routes under `/pools/{poolId}/nodes/...`
- nested `tasks` routes under `/jobs/{jobId}/tasks/...`

Operation groups confirmed during review:

1. `Applications`
2. `Jobs`
3. `Job Schedules`
4. `Pools`
5. `Nodes`
6. `Tasks`

Later reconciliation group:

7. `Certificates`

Recommended Stackyard route family:

- `/azure/batchservice/2025-06-01/applications`
- `/azure/batchservice/2025-06-01/applications/{applicationId}`
- `/azure/batchservice/2025-06-01/jobs`
- `/azure/batchservice/2025-06-01/jobs/{jobId}`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/enable`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/disable`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/terminate`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/tasks`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/tasks/{taskId}`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/tasks/{taskId}/terminate`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/tasks/{taskId}/reactivate`
- `/azure/batchservice/2025-06-01/jobs/{jobId}/tasks/{taskId}/files`
- `/azure/batchservice/2025-06-01/jobschedules`
- `/azure/batchservice/2025-06-01/jobschedules/{jobScheduleId}`
- `/azure/batchservice/2025-06-01/jobschedules/{jobScheduleId}/enable`
- `/azure/batchservice/2025-06-01/jobschedules/{jobScheduleId}/disable`
- `/azure/batchservice/2025-06-01/jobschedules/{jobScheduleId}/terminate`
- `/azure/batchservice/2025-06-01/jobschedules/{jobScheduleId}/jobs`
- `/azure/batchservice/2025-06-01/pools`
- `/azure/batchservice/2025-06-01/pools/{poolId}`
- `/azure/batchservice/2025-06-01/pools/{poolId}/resize`
- `/azure/batchservice/2025-06-01/pools/{poolId}/stopresize`
- `/azure/batchservice/2025-06-01/pools/{poolId}/enableautoscale`
- `/azure/batchservice/2025-06-01/pools/{poolId}/disableautoscale`
- `/azure/batchservice/2025-06-01/pools/{poolId}/nodes`
- `/azure/batchservice/2025-06-01/pools/{poolId}/nodes/{nodeId}`
- `/azure/batchservice/2025-06-01/pools/{poolId}/nodes/{nodeId}/files`
- `/azure/batchservice/2025-06-01/pools/{poolId}/nodes/{nodeId}/extensions`
- `/azure/batchservice/2025-06-01/poolusagemetrics`
- `/azure/batchservice/2025-06-01/supportedimages`

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:

- required query parameter: `api-version=2025-06-01`
- account endpoint is service-scoped, not ARM-scoped
- the reviewed pages consistently describe optional `timeOut` request query handling
- list routes commonly support `maxresults` and, for major collections, OData `$filter`, `$select`, and `$expand`
- many routes use status-code-only `HEAD` checks for existence or file properties
- auth in the reviewed pages is Microsoft Entra bearer auth using scope `https://batch.core.windows.net//.default`

Modeling implications for Stackyard:

- keep this emulator separate from `batch_management` even if some resource names overlap
- preserve stable IDs and timestamps, but keep operations synchronous and deterministic
- model paged collections with deterministic ordering and optional `odata.nextLink`
- represent file-content routes separately from JSON metadata routes
- support `HEAD` routes where the main contract is status code and response headers

### Applications

Reviewed method pages:

- `List Applications`
- `Get Application`

Modeled first-pass behavior:

- list available applications and versions visible to compute nodes
- get a specific application summary by `applicationId`
- return simple objects with:
  - `id`
  - `displayName`
  - `versions`

### Jobs

Reviewed method pages:

- `Create Job`
- `Get Job`
- `List Jobs`
- `Enable Job`
- `List Jobs From Schedule`

Operation-group page additionally lists:

- delete
- disable
- get task counts
- list job preparation and release task status
- replace
- terminate

Modeled first-pass behavior:

- create, get, list, replace, patch-like update semantics
- action routes for `enable`, `disable`, and `terminate`
- schedule-linked listing via `/jobschedules/{jobScheduleId}/jobs`
- deterministic counters for task counts and lightweight preparation/release task status

### Job Schedules

Reviewed method pages:

- `Get Job Schedule`
- `List Job Schedules`
- `Job Schedule Exists`
- `Replace Job Schedule`
- `Update Job Schedule`
- `Enable Job Schedule`
- `Disable Job Schedule`

Operation-group page additionally lists:

- create
- delete
- terminate

Modeled first-pass behavior:

- full lifecycle for create/get/list/head/replace/update/delete
- action routes for `enable`, `disable`, and `terminate`
- linkage between schedules and jobs created from the schedule

### Pools

Reviewed method pages:

- `Create Pool`
- `Get Pool`
- `List Pools`
- `List Pool Usage Metrics`
- `List Supported Images`

Operation-group page additionally lists:

- delete
- disable auto scale
- enable auto scale
- evaluate auto scale
- list pool node counts
- pool exists
- remove nodes
- replace pool properties
- resize pool
- stop pool resize
- update pool

Modeled first-pass behavior:

- full pool lifecycle
- action routes for resize, stop resize, enable/disable autoscale, and remove nodes
- seeded supported-image and usage-metric fixtures
- `HEAD` pool-exists route
- pool-scoped node-count summary route

### Nodes

Reviewed method pages:

- `List Nodes`
- `Create Node User`
- `Start Node`
- `Enable Node Scheduling`
- `List Node Extensions`
- `List Node Files`
- `Get Node File`

Operation-group page additionally lists:

- deallocate node
- delete node file
- delete node user
- disable node scheduling
- get node
- get node extension
- get node file properties
- get node remote login settings
- reboot node
- reimage node
- replace node user
- upload node logs

Modeled first-pass behavior:

- represent nodes as pool-scoped runtime records
- support node state transitions for start, deallocate, reboot, and reimage
- support node user create/replace/delete
- expose node file listing, metadata, and content separately
- return deterministic remote-login settings and upload-log acknowledgements

### Tasks

Reviewed method pages:

- `Create Task`
- `List Tasks`
- `List Task Files`
- `Get Task File Properties`

Operation-group page additionally lists:

- create task collection
- delete task
- delete task file
- get task
- get task file
- list sub tasks
- reactivate task
- replace task
- terminate task

Modeled first-pass behavior:

- full task lifecycle under a job
- collection create for bulk task add
- action routes for `terminate` and `reactivate`
- subtask fixture support for multi-instance task reads
- task file content and metadata routes distinct from task JSON routes

### Certificates

Docs note:

- During review, certificate search results surfaced on `2024-07-01` pages instead of a clean `2025-06-01` Batch Service page.
- That version mismatch should be reconciled before implementation claims certificate support for `2025-06-01`.

Plan implication:

- do not block the initial Batch Service rollout on certificates
- add a final reconciliation stage once the exact upstream doc/version relationship is confirmed

## State Model

Recommended in-memory state buckets:

- `applications` keyed by application ID
- `jobs` keyed by job ID
- `jobSchedules` keyed by schedule ID
- `pools` keyed by pool ID
- `nodes` keyed by `{poolId}/{nodeId}`
- `nodeUsers` keyed by `{poolId}/{nodeId}/{userName}`
- `tasks` keyed by `{jobId}/{taskId}`
- `taskFiles` keyed by `{jobId}/{taskId}/{filePath}`
- `nodeFiles` keyed by `{poolId}/{nodeId}/{filePath}`

Recommended seeded relationships:

- jobs may reference either a concrete pool or a schedule-created auto-pool fixture
- tasks belong to jobs and may be assigned to a node
- node files and task files should be independent collections because upstream routes are distinct
- pool usage metrics and supported images can be static fixtures in the first pass

## Error Model

Use deterministic JSON error envelopes aligned with the reviewed Batch Service pages:

- missing required identifiers or malformed request bodies should return `400`
- missing resources should return `404`
- conflicting creates should return `409`
- unsupported or invalid state transitions should return `409`

Representative first-pass error codes:

- `InvalidRequestBody`
- `InvalidPropertyValue`
- `JobNotFound`
- `JobScheduleNotFound`
- `PoolNotFound`
- `ComputeNodeNotFound`
- `TaskNotFound`
- `ApplicationNotFound`
- `ResourceFileNotFound`
- `ResourceExists`
- `InvalidState`

## Staged Implementation Plan

### Stage 1: Routing, Shared Data-Plane Skeleton, And Coverage Wiring

Add:

- `internal/server/provider_azure_batch_service.go`
- `internal/server/provider_azure_batch_service_test.go`
- `examples/azure/batchservice-2025-06-01`

Wire:

- a new canonical service key `batch_service`
- Azure router registration in `internal/server/http.go`
- the three Azure coverage scripts plus plan-doc mapping

Decisions:

- use a dedicated Stackyard path family under `/azure/batchservice/2025-06-01/...`
- treat account endpoint identity as logical emulator state, not ARM state

Exit criteria:

- routes are discoverable by the coverage scripts
- unknown routes do not fall through to `NotImplemented` for the implemented families

### Stage 2: Applications, Jobs, And Job Schedules

Implement:

- `applications` list/get
- `jobs` create/get/list/delete/replace/update
- `jobs` actions: enable, disable, terminate
- `jobs/{jobId}/taskCounts`
- `jobs/{jobId}/jobpreparationandreleasetaskstatus`
- `jobschedules` create/get/head/list/delete/replace/update
- `jobschedules` actions: enable, disable, terminate
- `jobschedules/{jobScheduleId}/jobs`

Modeling rules:

- deterministic timestamps and provisioning-style state transitions
- schedule-created jobs should be materialized in-memory and visible from both sides

Exit criteria:

- example can create a schedule, create a job, and read both back
- tests cover create/get/list/action plus negative missing-resource cases

### Stage 3: Pools And Pool Helper Reads

Implement:

- `pools` create/get/head/list/delete/replace/update
- pool actions: `resize`, `stopresize`, `enableautoscale`, `disableautoscale`, `removenodes`
- helper reads: `poolusagemetrics`, `supportedimages`, `poolnodecounts`
- autoscale evaluation result route

Modeling rules:

- pool size and allocation state changes should be deterministic
- usage metrics and supported images can be static fixtures with realistic shapes

Exit criteria:

- example can create a pool, resize it, and query supported images
- tests cover pool lifecycle and action semantics

### Stage 4: Nodes

Implement:

- list/get nodes
- node actions: start, deallocate, reboot, reimage, enable scheduling, disable scheduling
- node users: create, replace, delete
- remote login settings
- node extensions list/get
- node files list/get/head/delete
- upload node logs

Modeling rules:

- node state transitions should be constrained by current state
- node files should include both metadata and content routes

Exit criteria:

- tests cover node lifecycle actions, node-user mutations, and file reads

### Stage 5: Tasks

Implement:

- task create/get/list/delete/replace
- task collection create
- task actions: terminate and reactivate
- subtask list
- task file list/get/head/delete

Modeling rules:

- tasks should track assignment, state, execution info, retry counts, and timestamps
- task files should be separate from node files even if backed by shared fixture data internally

Exit criteria:

- example can create a job, add tasks, and read task files
- tests cover task lifecycle, bulk create, and task file routes

### Stage 6: Certificate Reconciliation Tail

Revisit:

- whether `Certificates` are officially part of the `2025-06-01` Batch Service surface or effectively carried by a `2024-07-01` doc page

If confirmed:

- add certificate list/get/create/delete support under the Batch Service surface
- add matching example calls and coverage signals

If not confirmed:

- leave certificates out of this service and document the exclusion explicitly

## Example Plan

Create `examples/azure/batchservice-2025-06-01` using Azure Go SDK core packages:

- `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
- `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`

Example flow:

1. list applications
2. create a pool
3. create a job bound to that pool
4. add a task to the job
5. list tasks
6. list pool nodes
7. read supported images

Environment conventions should match other Azure examples in this repo:

- `STACKYARD_ENDPOINT`
- `STACKYARD_AZURE_BATCH_ACCOUNT`
- `STACKYARD_AZURE_BATCH_SUBSCRIPTION_KEY`
- `STACKYARD_AZURE_LOCATION`

## Coverage Script Updates

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for:

- `batch_service`
- `batchservice`
- `batch-service`
- `azure_batch_service`
- `azure_batchservice`
- `batch_runtime`
- `batch_data_plane`
- `rest_batchservice_2025_06_01`
- `batchservice_2025_06_01`

Add plan-doc mapping:

- `batch_service` -> `batchservice-2025-06-01-plan.md`

## Validation Targets For The Future Implementation

Minimum implementation validation:

- `go test ./internal/server -run 'TestAzureBatchService' -count=1`
- `go build ./examples/azure/batchservice-2025-06-01`
- `python3 scripts/azure-contract-coverage.py --service batch_service --require-service batch_service --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service batch_service --require-service batch_service --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service batch_service --require-service batch_service --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service batch_service --require-service batch_service --fail-on any`
- `make ci`

Success criteria:

- implemented Batch Service routes return deterministic success payloads
- tested negative paths return structured Batch-style errors
- coverage scripts treat the service as implemented and documented
- no implemented Batch Service route falls through to `NotImplemented`
