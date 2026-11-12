# Azure Communication - Job Router (`communication-jobrouter-2023-11-01`) Staged Plan

## Objective

Emulate the Azure Communication Services Job Router data-plane API for `2023-11-01` with deterministic local behavior for:

- Job Router administration resources:
  - classification policies
  - distribution policies
  - exception policies
  - queues
- Job Router operational resources:
  - jobs
  - workers
  - offers
  - assignments
  - queue statistics and in-queue position
- an Azure Go SDK-based example under `examples/azure/communication-jobrouter-2023-11-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/routing/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Job Router `2023-11-01` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official Azure package index.

Important documentation note:

- Many of the method pages are authorization-gated on Learn, but the operation-group pages and linked method pages were still reviewable enough to confirm the route surface, resource families, response codes, ETag behavior, paging controls, and representative request and response payloads.
- The plan below is intentionally scoped to the documented `2023-11-01` Job Router surface and avoids inventing unrelated Communication services routes.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/operation-groups?view=rest-communication-jobrouter-2023-11-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations?view=rest-communication-jobrouter-2023-11-01`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/delete-classification-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/delete-distribution-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/delete-exception-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/delete-queue?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/get-classification-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/get-distribution-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/get-exception-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/get-queue?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/list-classification-policies?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/list-distribution-policies?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/list-exception-policies?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/list-queues?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/upsert-classification-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/upsert-distribution-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/upsert-exception-policy?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-administration-operations/upsert-queue?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/accept-job-offer?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/cancel-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/close-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/complete-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/decline-job-offer?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/delete-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/delete-worker?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/get-in-queue-position?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/get-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/get-queue-statistics?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/get-worker?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/list-jobs?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/list-workers?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/reclassify-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/unassign-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/upsert-job?view=rest-communication-jobrouter-2023-11-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/jobrouter/job-router-operations/upsert-worker?view=rest-communication-jobrouter-2023-11-01`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_jobrouter`
- display name: `Azure Communication - Job Router`
- API version token: `2023-11-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_jobrouter.go`
- tests: `internal/server/provider_azure_communication_jobrouter_test.go`
- example: `examples/azure/communication-jobrouter-2023-11-01`
- plan doc: `docs/communication-jobrouter-2023-11-01-plan.md`

Recommended coverage aliases:

- `communication_jobrouter`
- `communication-jobrouter`
- `jobrouter`
- `job_router`
- `azure_communication_jobrouter`
- `acs_jobrouter`
- `rest_communication_jobrouter_2023_11_01`
- `communication_jobrouter_2023_11_01`

## Surface Summary

The reviewed REST surface contains 2 operation groups and 33 concrete methods:

1. `Job Router Administration Operations`
2. `Job Router Operations`

Documented route families:

- `PATCH {endpoint}/routing/classificationPolicies/{classificationPolicyId}?api-version=2023-11-01`
- `GET {endpoint}/routing/classificationPolicies/{classificationPolicyId}?api-version=2023-11-01`
- `GET {endpoint}/routing/classificationPolicies?api-version=2023-11-01`
- `DELETE {endpoint}/routing/classificationPolicies/{classificationPolicyId}?api-version=2023-11-01`
- `PATCH {endpoint}/routing/distributionPolicies/{distributionPolicyId}?api-version=2023-11-01`
- `GET {endpoint}/routing/distributionPolicies/{distributionPolicyId}?api-version=2023-11-01`
- `GET {endpoint}/routing/distributionPolicies?api-version=2023-11-01`
- `DELETE {endpoint}/routing/distributionPolicies/{distributionPolicyId}?api-version=2023-11-01`
- `PATCH {endpoint}/routing/exceptionPolicies/{exceptionPolicyId}?api-version=2023-11-01`
- `GET {endpoint}/routing/exceptionPolicies/{exceptionPolicyId}?api-version=2023-11-01`
- `GET {endpoint}/routing/exceptionPolicies?api-version=2023-11-01`
- `DELETE {endpoint}/routing/exceptionPolicies/{exceptionPolicyId}?api-version=2023-11-01`
- `PATCH {endpoint}/routing/queues/{queueId}?api-version=2023-11-01`
- `GET {endpoint}/routing/queues/{queueId}?api-version=2023-11-01`
- `GET {endpoint}/routing/queues?api-version=2023-11-01`
- `DELETE {endpoint}/routing/queues/{queueId}?api-version=2023-11-01`
- `PATCH {endpoint}/routing/jobs/{jobId}?api-version=2023-11-01`
- `GET {endpoint}/routing/jobs/{jobId}?api-version=2023-11-01`
- `GET {endpoint}/routing/jobs?api-version=2023-11-01`
- `DELETE {endpoint}/routing/jobs/{jobId}?api-version=2023-11-01`
- `POST {endpoint}/routing/jobs/{jobId}:cancel?api-version=2023-11-01`
- `POST {endpoint}/routing/jobs/{jobId}:complete?api-version=2023-11-01`
- `POST {endpoint}/routing/jobs/{jobId}:close?api-version=2023-11-01`
- `POST {endpoint}/routing/jobs/{jobId}:reclassify?api-version=2023-11-01`
- `POST {endpoint}/routing/jobs/{jobId}:unassign?api-version=2023-11-01`
- `GET {endpoint}/routing/jobs/{jobId}/position?api-version=2023-11-01`
- `PATCH {endpoint}/routing/workers/{workerId}?api-version=2023-11-01`
- `GET {endpoint}/routing/workers/{workerId}?api-version=2023-11-01`
- `GET {endpoint}/routing/workers?api-version=2023-11-01`
- `DELETE {endpoint}/routing/workers/{workerId}?api-version=2023-11-01`
- `POST {endpoint}/routing/workers/{workerId}/offers/{offerId}:accept?api-version=2023-11-01`
- `POST {endpoint}/routing/workers/{workerId}/offers/{offerId}:decline?api-version=2023-11-01`
- `GET {endpoint}/routing/queues/{queueId}/statistics?api-version=2023-11-01`

Common optional query traits visible from reviewed pages:

- `maxpagesize` on list routes
- filter parameters on list jobs and list workers

Recommended Stackyard route family:

- `/azure/communication-jobrouter/2023-11-01/routing/...`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other Communication endpoint-rooted services.
- Every route still requires `api-version=2023-11-01`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- security is documented with the Communication Services default scope `communication.azure.com/.default`
- errors use `Azure.Core.Foundations.ErrorResponse`
- many `GET` routes return `ETag` and `Last-Modified`
- many `PATCH` upsert routes accept `application/merge-patch+json`
- list routes return paged result envelopes

### Administration Resources

Reviewed administration families:

- classification policies
- distribution policies
- exception policies
- queues

High-value shared behavior from reviewed method pages:

- Upsert routes use `PATCH`
- Get routes return the resource and response headers such as:
  - `ETag`
  - `Last-Modified`
- List routes return paged collections, commonly with optional `maxpagesize`
- Delete routes return `204 No Content`

Modeling implications:

- first-pass emulation should maintain deterministic stores for each administration resource family
- every stored resource should carry:
  - `id`
  - `name`
  - typed `properties`
  - `etag`
  - `lastModified`
- first-pass pagination can be simple but should preserve `value` plus optional paging token or `nextLink` shape

Specific high-value payload elements visible from reviewed pages:

- classification policies contain prioritization, queue selection, worker selectors, and fallback labeling rules
- distribution policies contain:
  - `mode.kind`
  - mode-specific settings such as offer TTL or worker selectors
- exception policies contain:
  - `exceptionRules`
  - trigger kinds such as `waitTime`
  - actions such as `reclassify`
- queues contain:
  - distribution policy linkage
  - optional exception policy linkage
  - labels

Implementation guidance:

- do not try to fully execute Job Router scheduling semantics in the first pass
- instead, store and echo the core policy objects faithfully enough for clients and coverage scripts to interact with the surface

### Jobs

Reviewed job routes:

- `Upsert Job`
- `Get Job`
- `List Jobs`
- `Delete Job`
- `Cancel Job`
- `Complete Job`
- `Close Job`
- `Reclassify Job`
- `Unassign Job`
- `Get In Queue Position`

High-value route behavior from reviewed pages:

- `Upsert Job`
  - `PATCH /routing/jobs/{jobId}`
  - creates or updates a router job
- `Get Job`
  - `GET /routing/jobs/{jobId}`
- `List Jobs`
  - `GET /routing/jobs`
  - supports filter-oriented retrieval
- `Delete Job`
  - `DELETE /routing/jobs/{jobId}`
- `Cancel Job`
  - `POST /routing/jobs/{jobId}:cancel`
  - accepts free-form cancellation reason
- `Complete Job`
  - `POST /routing/jobs/{jobId}:complete`
- `Close Job`
  - `POST /routing/jobs/{jobId}:close`
- `Reclassify Job`
  - `POST /routing/jobs/{jobId}:reclassify`
- `Unassign Job`
  - `POST /routing/jobs/{jobId}:unassign`
- `Get In Queue Position`
  - `GET /routing/jobs/{jobId}/position`

Modeling implications:

- jobs need a deterministic lifecycle state machine, at minimum:
  - queued
  - assigned
  - completed
  - closed
  - canceled
- first-pass operational actions can mutate stored lifecycle flags instead of re-running full matching logic
- `Get In Queue Position` can return deterministic synthetic ranking data derived from queue membership and creation order

### Workers And Offers

Reviewed worker and offer routes:

- `Upsert Worker`
- `Get Worker`
- `List Workers`
- `Delete Worker`
- `Accept Job Offer`
- `Decline Job Offer`

High-value route behavior from reviewed pages:

- `Upsert Worker`
  - `PATCH /routing/workers/{workerId}`
  - creates or updates a worker
- `Get Worker`
  - `GET /routing/workers/{workerId}`
- `List Workers`
  - `GET /routing/workers`
- `Delete Worker`
  - `DELETE /routing/workers/{workerId}`
- `Accept Job Offer`
  - `POST /routing/workers/{workerId}/offers/{offerId}:accept`
  - returns `200 OK`
  - response includes:
    - `assignmentId`
    - `jobId`
    - `workerId`
  - documented conflict behavior:
    - `409 Conflict` if another agent already accepted
- `Decline Job Offer`
  - `POST /routing/workers/{workerId}/offers/{offerId}:decline`

Modeling implications:

- worker state should include:
  - labels
  - capacity
  - channel configurations
  - offers
  - assignments
- offers can be modeled as synthetic child records seeded when jobs are created or reclassified
- first-pass scheduling does not need to be smart:
  - deterministic offer generation is enough
  - accept and decline flows must be stable and conflict-aware

### Queue Statistics

Reviewed route:

- `GET /routing/queues/{queueId}/statistics`

Modeling implications:

- this route should aggregate deterministic counts from the in-memory job store
- first-pass stats can include queued, assigned, and waiting counts derived from stored job state

## Recommended Emulator Shape

State model:

- `classification policies` keyed by id
- `distribution policies` keyed by id
- `exception policies` keyed by id
- `queues` keyed by id
- `jobs` keyed by id
- `workers` keyed by id
- `offers` keyed by `{workerId}|{offerId}`
- `assignments` keyed by `{jobId}|{assignmentId}`

Suggested helper behaviors:

- store `etag` and `lastModified` on all persisted resources
- generate deterministic IDs for offers and assignments
- keep synthetic matching simple and deterministic:
  - jobs in a queue can produce offers for the lexicographically earliest available worker in that queue
- preserve `PATCH` merge behavior for mutable resource fields
- support stable paging order for all list routes

Error-handling expectations:

- reject wrong or missing `api-version`
- reject invalid JSON bodies
- reject missing required IDs and body fields
- return consistent not-found errors for missing resources and offers
- return `409 Conflict` on double-accept for the same offer/job
- return method or path errors rather than `NotImplemented`

## Staged Implementation Plan

### Stage 1: Route Skeleton And Shared Data-Plane Validation

Add the new data-plane route family and service registration:

- implement `internal/server/provider_azure_communication_jobrouter.go`
- wire the router into `internal/server/http.go`
- enforce `api-version=2023-11-01`
- add shared helpers for JSON decoding, error envelopes, ETag handling, paging, and deterministic timestamps

Exit criteria:

- all reviewed Job Router routes resolve into the new provider
- unsupported methods return proper method errors rather than `NotImplemented`

### Stage 2: Administration Resources

Implement CRUD and list flows for:

- classification policies
- distribution policies
- exception policies
- queues

Behavior goals:

- support `PATCH` upsert semantics
- return `ETag` and `Last-Modified` on `GET`
- support `maxpagesize` on list routes
- return `204` on deletes

Exit criteria:

- the administration surface is complete and deterministic

### Stage 3: Core Operational Resources

Implement:

- jobs upsert/get/list/delete
- workers upsert/get/list/delete
- queue statistics
- in-queue position

Behavior goals:

- jobs and workers persist strongly typed but flexible payloads
- queue statistics derive from stored jobs
- in-queue position is deterministic based on queued order

Exit criteria:

- basic operational state exists for follow-on actions

### Stage 4: Job Lifecycle Actions

Implement:

- cancel job
- complete job
- close job
- reclassify job
- unassign job

Behavior goals:

- mutate job state consistently
- preserve optional action notes such as cancellation reason
- keep traces simple and local rather than simulating external eventing

Exit criteria:

- all reviewed job action routes return typed success responses and update stored state

### Stage 5: Offers And Assignments

Implement:

- accept job offer
- decline job offer

Behavior goals:

- seed deterministic offers for assignable jobs
- accept returns typed `assignmentId`, `jobId`, and `workerId`
- duplicate acceptance on a claimed offer returns `409 Conflict`
- declining an offer preserves job viability for later reassignment

Exit criteria:

- offer flows are deterministic and conflict-aware

### Stage 6: Provider Tests

Add focused tests in `internal/server/provider_azure_communication_jobrouter_test.go` for:

- each administration resource family upsert/get/list/delete
- job upsert/get/list/delete
- worker upsert/get/list/delete
- cancel, complete, close, reclassify, and unassign job actions
- queue statistics and in-queue position
- accept and decline offer behavior
- conflict on double accept
- invalid body validation
- not-found behavior
- incorrect `api-version`

Exit criteria:

- all reviewed routes are exercised
- no reviewed endpoint falls through to `NotImplemented`

### Stage 7: Azure Go SDK Example

Add `examples/azure/communication-jobrouter-2023-11-01`:

- use `azcore/runtime` and `azcore/policy`
- authenticate with the same ACS endpoint-signing pattern used by the repo’s existing Communication data-plane examples
- create:
  - a distribution policy
  - a queue
  - a worker
  - a job
- list jobs or workers
- accept a deterministic offer or query queue statistics

The example should stay intentionally small and prove the main emulator value:

1. administration object creation
2. operational object creation
3. one end-to-end offer or queue-statistics interaction

Exit criteria:

- the example builds cleanly
- it exercises both administration and operational route families

### Stage 8: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Required coverage additions:

- map the canonical service key `communication_jobrouter`
- add the alias set listed above
- map the plan doc in the doc-coverage script

Exit criteria:

- coverage tooling recognizes `communication_jobrouter`
- the service can be targeted explicitly by the Azure coverage scripts

## Risks And Edge Cases To Handle Deliberately

- upsert routes use merge-patch semantics, so the emulator should not accidentally treat omitted fields as clears unless explicitly requested
- ETag and `If-Match` handling may matter for some clients; a first pass can echo ETags even if strict conditional-update enforcement is deferred
- offer generation should stay deterministic and test-friendly rather than attempting a full production-grade routing engine
- `List Jobs` and `List Workers` expose filter behavior in the real service; a first pass can support only basic documented filters if the full expression grammar is too large
- queue statistics should be derived from stored state and not hard-coded

## Definition Of Done

The service is ready when all of the following are true:

- all 33 reviewed `2023-11-01` Job Router routes are implemented
- no reviewed Job Router route returns `NotImplemented`
- deterministic provider tests pass
- the example under `examples/azure/communication-jobrouter-2023-11-01` builds and runs against Stackyard
- the Azure coverage scripts recognize and validate the new `communication_jobrouter` service
