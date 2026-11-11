# Azure Compute Schedule (`computeschedule-2025-05-01`) Staged Plan

## Objective

Emulate the Azure Compute Schedule ARM management API for `2025-05-01` with deterministic local behavior for:

- provider `operations`
- location-scoped scheduled-action batch routes for virtual machines
- operation polling and operation-error inspection
- an Azure Go SDK example under `examples/azure/computeschedule-2025-05-01`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed Learn surface is the ARM management plane under `Microsoft.ComputeSchedule`, not the broader `Microsoft.Compute` provider and not a data plane.
- This API is centered on batch VM scheduling and immediate execution requests, not VM CRUD.
- The reviewed `2025-05-01` operation-group pages expose two families:
  - `Operations`
  - `Scheduled Actions`
- The `Scheduled Actions` group includes both generally documented routes and a couple of `[PRIVATE PREVIEW]` routes for VM create/delete. Those preview-only routes should be called out explicitly in the implementation plan rather than silently mixed into the first pass.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/computeschedule/operation-groups?view=rest-computeschedule-2025-05-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/computeschedule/operations?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions?view=rest-computeschedule-2025-05-01`

Representative linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/computeschedule/operations/list?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-submit-start?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-submit-deallocate?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-submit-hibernate?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-execute-start?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-execute-deallocate?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-cancel-operations?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-get-operation-status?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-get-operation-errors?view=rest-computeschedule-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/computeschedule/scheduled-actions/virtual-machines-execute-create?view=rest-computeschedule-2025-05-01`

## SDK Selection Note

- The official Learn Go samples for this surface use `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computeschedule/armcomputeschedule`.
- The future example should prefer typed `armcomputeschedule` clients for:
  - `Operations`
  - `ScheduledActions`
- If the local module version does not expose every reviewed route cleanly, a narrow `azcore/runtime` fallback is acceptable, but the first preference should stay with the typed package because the docs already show generated Go examples for this API version.

## Service Naming

Canonical Stackyard naming:

- service key: `compute_schedule`
- display name: `Azure Compute Schedule`
- API version token: `2025-05-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_compute_schedule.go`
- tests: `internal/server/provider_azure_compute_schedule_test.go`
- example: `examples/azure/computeschedule-2025-05-01`
- plan doc: `docs/computeschedule-2025-05-01-plan.md`

Recommended coverage aliases:

- `compute_schedule`
- `compute-schedule`
- `computeschedule`
- `azure_compute_schedule`
- `microsoft_computeschedule`
- `computeschedule_2025_05_01`
- `rest_computeschedule_2025_05_01`

## API Surface Summary

The reviewed pages expose a provider root plus location-scoped action routes:

- provider-level:
  - `/providers/Microsoft.ComputeSchedule/operations`
- scheduled-action batch routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesSubmitStart`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesSubmitDeallocate`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesSubmitHibernate`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesExecuteStart`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesExecuteDeallocate`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesExecuteHibernate`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesCancelOperations`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesGetOperationStatus`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesGetOperationErrors`

Preview-only routes surfaced on the same group page:

- `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesExecuteCreate`
- `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeSchedule/locations/{locationparameter}/virtualMachinesExecuteDelete`

Modeling implications:

- this is a subscription-location keyed action surface, not a normal ARM resource CRUD tree
- there is no resource group segment in the reviewed routes
- most routes are `POST` action endpoints with structured request bodies and typed operation-result payloads
- the emulator should keep an in-memory operation store keyed by `(subscriptionId, location, operationId)` and optionally by `correlationId`

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- path parameters validate `subscriptionId` as UUID and `locationparameter` as non-empty
- batch request bodies consistently carry `correlationid`
- result payloads are operation-centric, not long-running ARM `Azure-AsyncOperation` resources

### Operations

From `Operations.List`:

- route:
  - `GET /providers/Microsoft.ComputeSchedule/operations?api-version=2025-05-01`
- response:
  - `value`
  - each operation can include `name`, `display`, optional `origin`, `actionType`, and `isDataAction`

The sample list indicates the practical control-plane surface:

- submit start/deallocate/hibernate
- execute start/deallocate/hibernate
- cancel operations
- get operation status/errors
- preview create/delete execution actions

### Scheduled Submit Routes

From the `submit start`, `submit deallocate`, and `submit hibernate` pages:

- routes:
  - `POST .../virtualMachinesSubmitStart`
  - `POST .../virtualMachinesSubmitDeallocate`
  - `POST .../virtualMachinesSubmitHibernate`
- request body shape:
  - `correlationid`
  - `executionParameters`
  - `resources`
  - `schedule`
- `schedule` includes:
  - `deadline`
  - `timezone` or `timeZone` / user-request variants shown in samples
  - `deadlineType`
- `executionParameters` includes:
  - `optimizationPreference`
  - `retryPolicy.retryCount`
  - `retryPolicy.retryWindowInMinutes`

Behavior implication:

- first-pass emulation can validate the required high-level sections and normalize sample-name drift such as `correlationid` vs `correlationId`, `timezone` vs `timeZone`, and `deadline` vs user-request fields
- these routes should create scheduled operations with initial state like `Scheduled`

### Immediate Execute Routes

From `execute start` and `execute deallocate`:

- routes:
  - `POST .../virtualMachinesExecuteStart`
  - `POST .../virtualMachinesExecuteDeallocate`
- request body is similar to submit routes but does not need the future schedule semantics to behave like deferred requests

Behavior implication:

- first-pass emulation can accept the same normalized request body family and return immediate operation results with state progression seeded as `InProgress` or `Succeeded`

### Status and Error Inspection

From `get operation status` and `get operation errors`:

- routes:
  - `POST .../virtualMachinesGetOperationStatus`
  - `POST .../virtualMachinesGetOperationErrors`
- request body shape:
  - `correlationid`
  - `operationIds`
- response shape refers to lists of resource-level operation results

Behavior implication:

- the provider needs a stable operation ledger so later polling requests can return consistent states and any seeded resource-level errors

### Cancellation

From `cancel operations`:

- route:
  - `POST .../virtualMachinesCancelOperations`
- request body shape:
  - `correlationid`
  - `operationIds`
- sample response returns `results` entries with:
  - `resourceId`
  - optional `errorCode`
  - optional `errorDetails`
  - nested `operation`

Behavior implication:

- cancellation can mutate matching scheduled operations into a deterministic cancelled terminal state without needing external async infrastructure

### Preview Create/Delete Execution Actions

From the scheduled-actions group page and `execute create` page:

- `virtualMachinesExecuteCreate` and `virtualMachinesExecuteDelete` are explicitly labeled `[PRIVATE PREVIEW]`

Behavior implication:

- these should be separated into a later optional stage unless the repo’s coverage expectations require them immediately
- if included, keep the implementation obviously synthetic and scoped to the documented request/response contracts rather than pretending to create or delete full VMs

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the new service shell and alias wiring:

- reserve service key `compute_schedule`
- add provider registration in `internal/server/http.go`
- add provider scaffolding in `internal/server/provider_azure_compute_schedule.go`
- add plan-doc mapping and aliases in the three Azure coverage scripts
- create the example directory scaffold under `examples/azure/computeschedule-2025-05-01`

Deliverable:

- the service is discoverable by coverage tooling even before the deeper action routes are filled in

### Stage 1: Provider Operations

Implement the provider metadata route:

- `GET /providers/Microsoft.ComputeSchedule/operations`

Include operation names covering:

- submit start/deallocate/hibernate
- execute start/deallocate/hibernate
- cancel operations
- get operation status/errors
- optionally preview create/delete actions, clearly labeled

Deliverable:

- foundational provider discovery works and coverage can attribute the service cleanly

### Stage 2: Shared Request Parsing and Operation Store

Implement shared helpers for:

- subscription/location route parsing
- UUID validation for `subscriptionId`
- request-body parsing and normalization
- operation-id generation
- in-memory operation persistence keyed by subscription, location, and operation id
- deterministic resource-id extraction from `resources.ids`

Recommended internal model slices:

- operation record:
  - `operationId`
  - `correlationId`
  - `subscriptionId`
  - `location`
  - `opType`
  - `state`
  - `deadline`
  - `deadlineType`
  - `timezone`
  - `resourceIds`
  - `retryPolicy`
  - `createdAt`
  - `completedAt`
  - optional resource-level errors

Deliverable:

- the provider has one normalized state model reused by every scheduled-action endpoint

### Stage 3: Submit Start/Deallocate/Hibernate

Implement the non-preview deferred scheduling routes:

- `virtualMachinesSubmitStart`
- `virtualMachinesSubmitDeallocate`
- `virtualMachinesSubmitHibernate`

Validation targets:

- require `correlationid`
- require at least one resource id
- require valid-looking Microsoft.Compute VM resource ids for first pass
- require schedule shape and parsable deadline semantics

Behavior targets:

- create operation records in `Scheduled`
- return typed per-resource operation responses
- preserve `correlationid` deterministically to support later polling

Deliverable:

- the core Compute Schedule value proposition is present without the preview tails

### Stage 4: Execute Start/Deallocate/Hibernate

Implement immediate execution routes:

- `virtualMachinesExecuteStart`
- `virtualMachinesExecuteDeallocate`
- `virtualMachinesExecuteHibernate`

Behavior targets:

- create operation records with immediate lifecycle state
- optionally seed them as `InProgress` on first response and `Succeeded` on later status poll
- keep response shape aligned with the operation-response families from the reviewed docs

Deliverable:

- immediate and deferred scheduling are both covered for the non-preview public routes

### Stage 5: Status, Errors, and Cancellation

Implement the inspection and mutation routes:

- `virtualMachinesGetOperationStatus`
- `virtualMachinesGetOperationErrors`
- `virtualMachinesCancelOperations`

Behavior targets:

- status polling returns stable per-operation results
- error polling returns seeded resource-level error details only where applicable
- cancellation updates only cancellable records and leaves terminal ones unchanged

Deliverable:

- the action surface becomes stateful and realistic enough for SDK/client polling flows

### Stage 6: Preview Reconciliation

Decide whether to include the preview-only execution routes now or defer them:

- `virtualMachinesExecuteCreate`
- `virtualMachinesExecuteDelete`

Recommendation:

- include them only if the reviewed endpoint-coverage contract expects every route on the operation-group page to resolve
- otherwise defer them and document the choice clearly in implementation notes

Deliverable:

- no ambiguity remains around the preview boundary

### Stage 7: Example Program

Add `examples/azure/computeschedule-2025-05-01`:

- use the official Azure Go SDK package `armcomputeschedule`
- show:
  - listing operations
  - submitting a scheduled VM start
  - polling operation status
  - optionally cancelling a second scheduled operation

Suggested example flow:

1. list provider operations
2. submit start for a small batch of VM resource ids
3. poll `virtualMachinesGetOperationStatus`
4. fetch `virtualMachinesGetOperationErrors`
5. cancel another scheduled operation to demonstrate the mutation path

Deliverable:

- an example that exercises the emulator’s real state transitions, not just a static GET

### Stage 8: Tests and Coverage

Add focused provider tests in `internal/server/provider_azure_compute_schedule_test.go` for:

- operations listing
- submit start/deallocate/hibernate success
- execute start/deallocate success
- status polling lifecycle
- error retrieval
- cancellation
- request validation failures:
  - invalid api-version
  - invalid subscription id
  - missing `correlationid`
  - empty resource list
  - invalid VM resource ids
  - missing schedule on submit routes
  - unknown operation ids on status/errors/cancel

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Validation target after implementation:

- `python3 scripts/azure-contract-coverage.py --service compute_schedule --require-service compute_schedule --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service compute_schedule --require-service compute_schedule --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service compute_schedule --require-service compute_schedule --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service compute_schedule --require-service compute_schedule --fail-on any`

## Recommended First Implementation Slice

If the goal is a mergeable first version with the highest signal-to-effort ratio:

1. provider `operations`
2. shared parsing and operation store
3. `virtualMachinesSubmitStart`
4. `virtualMachinesGetOperationStatus`
5. `virtualMachinesCancelOperations`
6. one Azure Go SDK example call path
7. coverage wiring

That slice establishes the core schedule/poll/cancel lifecycle quickly, and the remaining public routes can then be filled by reusing the same operation model.
