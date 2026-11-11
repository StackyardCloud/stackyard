# Azure Quantum Data Plane (`azurequantum-dataplane-2026-01-15-preview`) Staged Plan

## Objective

Emulate the Azure Quantum data-plane API for `2026-01-15-preview` with deterministic local behavior for:

- workspace-scoped job lifecycle under `/jobs`
- workspace-scoped session lifecycle under `/sessions`
- session-scoped job lifecycle under `/sessions/{sessionId}/jobs`
- provider, quota, top-level item, and storage helper reads
- an Azure Go SDK-based example under `examples/azure/azurequantum-dataplane-2026-01-15-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-azurequantum-dataplane-2026-01-15-preview` is not the ARM `Microsoft.Quantum` management plane.
- It is a true Quantum workspace data plane rooted at a workspace endpoint, with jobs and sessions addressed directly on the service URL rather than through `/subscriptions/.../providers/...`.
- This plan intentionally treats it as a separate Stackyard service from any future ARM `quantum` resource-provider implementation.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/operation-groups?view=rest-azurequantum-dataplane-2026-01-15-preview`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/jobs?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/providers?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/quotas?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/session-jobs?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/sessions?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/storage?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/top-level-items?view=rest-azurequantum-dataplane-2026-01-15-preview`

Linked method pages reviewed across those groups:
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/jobs/create?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/jobs/get?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/jobs/list?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/jobs/cancel?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/jobs/delete?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/providers/list?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/quotas/list?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/session-jobs/create?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/session-jobs/get?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/session-jobs/list?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/sessions/open?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/sessions/get?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/sessions/list?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/sessions/close?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/storage/get-sas-uri?view=rest-azurequantum-dataplane-2026-01-15-preview`
- `https://learn.microsoft.com/en-us/rest/api/azurequantum/dataplane/top-level-items/list?view=rest-azurequantum-dataplane-2026-01-15-preview`

SDK selection note:
- I did not find a Quantum data-plane-specific typed Azure Go SDK package during review.
- The official Azure Go SDK package I did find is ARM management-plane only: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/quantum/armquantum`.
- Plan assumption: the example should use the official Azure Go SDK core runtime pipeline (`azcore/runtime` and `azcore/policy`) rather than inventing a non-existent typed data-plane client.

## Service Naming

Canonical Stackyard naming:
- service key: `quantum_data_plane`
- display name: `Azure Quantum Data Plane`
- API version token: `2026-01-15-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_quantum_data_plane.go`
- tests: `internal/server/provider_azure_quantum_data_plane_test.go`
- example: `examples/azure/azurequantum-dataplane-2026-01-15-preview`
- plan doc: `docs/azurequantum-dataplane-2026-01-15-preview-plan.md`

Recommended coverage aliases:
- `quantum_data_plane`
- `quantum_dataplane`
- `quantum-data-plane`
- `quantum-dataplane`
- `azure_quantum_data_plane`
- `azure_quantum_dataplane`
- `azurequantum_data_plane`
- `azurequantum_dataplane`
- `azurequantum-dataplane`
- `rest_azurequantum_dataplane_2026_01_15_preview`
- `azurequantum_2026_01_15_preview_dataplane`

## Surface Summary

The reviewed REST surface is split into 7 operation groups:

1. `Jobs`
2. `Providers`
3. `Quotas`
4. `Session Jobs`
5. `Sessions`
6. `Storage`
7. `Top Level Items`

Method families identified from the reviewed pages:

`Jobs`
- `Create`
- `Get`
- `List`
- `Cancel`
- `Delete`

`Providers`
- `List`

`Quotas`
- `List`

`Session Jobs`
- `Create`
- `Get`
- `List`
- `Cancel`
- `Delete`

`Sessions`
- `Open`
- `Get`
- `List`
- `Close`
- `Delete`

`Storage`
- `Get Sas Uri`

`Top Level Items`
- `List`

## Route Envelope

Documented route family:
- `POST {endpoint}/jobs?api-version=2026-01-15-preview`
- `GET {endpoint}/jobs/{jobId}?api-version=2026-01-15-preview`
- `GET {endpoint}/jobs?api-version=2026-01-15-preview`
- `POST {endpoint}/jobs/{jobId}:cancel?api-version=2026-01-15-preview`
- `DELETE {endpoint}/jobs/{jobId}?api-version=2026-01-15-preview`
- `GET {endpoint}/providers?api-version=2026-01-15-preview`
- `GET {endpoint}/quotas?api-version=2026-01-15-preview`
- `POST {endpoint}/sessions?api-version=2026-01-15-preview`
- `GET {endpoint}/sessions/{sessionId}?api-version=2026-01-15-preview`
- `GET {endpoint}/sessions?api-version=2026-01-15-preview`
- `POST {endpoint}/sessions/{sessionId}:close?api-version=2026-01-15-preview`
- `DELETE {endpoint}/sessions/{sessionId}?api-version=2026-01-15-preview`
- `POST {endpoint}/sessions/{sessionId}/jobs?api-version=2026-01-15-preview`
- `GET {endpoint}/sessions/{sessionId}/jobs/{jobId}?api-version=2026-01-15-preview`
- `GET {endpoint}/sessions/{sessionId}/jobs?api-version=2026-01-15-preview`
- `POST {endpoint}/sessions/{sessionId}/jobs/{jobId}:cancel?api-version=2026-01-15-preview`
- `DELETE {endpoint}/sessions/{sessionId}/jobs/{jobId}?api-version=2026-01-15-preview`
- `POST {endpoint}/storage/sasUri?api-version=2026-01-15-preview`
- `GET {endpoint}/topLevelItems?api-version=2026-01-15-preview`

Stackyard emulation prefix should remain distinct from ARM `quantum`.

Recommended Stackyard route family:
- `/azure/quantum-data-plane/2026-01-15-preview/jobs`
- `/azure/quantum-data-plane/2026-01-15-preview/jobs/{jobId}`
- `/azure/quantum-data-plane/2026-01-15-preview/jobs/{jobId}:cancel`
- `/azure/quantum-data-plane/2026-01-15-preview/providers`
- `/azure/quantum-data-plane/2026-01-15-preview/quotas`
- `/azure/quantum-data-plane/2026-01-15-preview/sessions`
- `/azure/quantum-data-plane/2026-01-15-preview/sessions/{sessionId}`
- `/azure/quantum-data-plane/2026-01-15-preview/sessions/{sessionId}:close`
- `/azure/quantum-data-plane/2026-01-15-preview/sessions/{sessionId}/jobs`
- `/azure/quantum-data-plane/2026-01-15-preview/sessions/{sessionId}/jobs/{jobId}`
- `/azure/quantum-data-plane/2026-01-15-preview/sessions/{sessionId}/jobs/{jobId}:cancel`
- `/azure/quantum-data-plane/2026-01-15-preview/storage/sasUri`
- `/azure/quantum-data-plane/2026-01-15-preview/topLevelItems`

Important routing notes from the reviewed pages:
- This API does not use the Azure ARM `/subscriptions/.../providers/...` envelope.
- The upstream `endpoint` is the Quantum workspace data-plane URL, so Stackyard should emulate a stable service root with workspace-scoped local state.
- `api-version` is still required on every route.
- session and job IDs are service-level identifiers, not ARM child-resource names.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- security is Microsoft Entra ID bearer auth
- errors use a common JSON `error` envelope
- list routes are paged or list-shaped and should preserve deterministic ordering
- most mutable routes are asynchronous in the real service, but Stackyard can model them synchronously while still returning plausible status/provisioning fields

### Jobs

Reviewed method pages:
- `Create`
- `Get`
- `List`
- `Cancel`
- `Delete`

Modeling implications:
- store jobs keyed by `jobId`
- preserve request fields like provider, target, input/output data format, input parameters, container URI, and session linkage when supplied
- return stable computed fields such as:
  - `id`
  - `name`
  - `status`
  - `creationTime`
  - `beginExecutionTime`
  - `endExecutionTime`
  - `sessionId`
  - `target`
  - `providerId`
- model `Cancel` as a status transition like `Waiting` or `Executing` to `Cancelled`
- keep delete idempotent and aligned with the documented success code

### Sessions

Reviewed method pages:
- `Open`
- `Get`
- `List`
- `Close`
- `Delete`

Modeling implications:
- store sessions keyed by `sessionId`
- preserve core fields such as:
  - `id`
  - `name`
  - `status`
  - `providerId`
  - `target`
  - `jobFailurePolicy`
  - `creationTime`
  - `beginExecutionTime`
  - `endExecutionTime`
- treat `Open` as create-or-start, returning an active session
- treat `Close` as a status transition to `Succeeded` or `Closed`
- make `Delete` reject or no-op cleanly when the session is already absent

### Session Jobs

Reviewed method pages:
- `Create`
- `Get`
- `List`
- `Cancel`
- `Delete`

Modeling implications:
- store session jobs in the same canonical job table, but index by `sessionId`
- enforce that the parent session exists before create/list/get
- surface the same base shape as top-level jobs with `sessionId` populated
- let cancel and delete mirror top-level job behavior

### Providers, Quotas, Top Level Items

Reviewed method pages:
- `Providers/List`
- `Quotas/List`
- `Top Level Items/List`

Modeling implications:
- these are good stage-0/stage-1 endpoints because they are read-only and unblock examples and coverage quickly
- return deterministic static fixtures
- recommended starter fixtures:
  - providers: `ionq`, `quantinuum`, `microsoft-qc`, `rigetti`
  - quotas: small set of provider/target quota rows with `dimension`, `scope`, `utilization`, and `limit`
  - top-level items: mixed list entries that reference both sessions and jobs

### Storage

Reviewed method page:
- `Get Sas Uri`

Modeling implications:
- return a deterministic fake SAS URL rooted at Stackyard or a stable mock blob URL
- preserve request fields like container/blob hints when present
- the response should look realistic enough for SDK/client examples without requiring real storage backends

## Error-Handling Expectations To Emulate

High-value validation from the reviewed pages to model early:
- missing or invalid `api-version`
- malformed JSON bodies
- missing required identifiers for create routes
- invalid parent `sessionId` for session-job operations
- unknown `jobId` and `sessionId`
- invalid state transitions:
  - canceling an already terminal job
  - closing an already closed session
- bad pagination tokens if the list routes expose continuation tokens

Recommended first-pass error codes:
- `400 Bad Request` for malformed payloads and invalid transitions
- `404 Not Found` for missing jobs, sessions, or nested parents
- `409 Conflict` for duplicate create semantics if the upstream method is create-only rather than upsert

## State Model

Recommended in-memory structures:
- `workspaces` is not needed if the emulator route already scopes to one endpoint root
- `jobs map[string]*quantumJob`
- `sessions map[string]*quantumSession`
- `sessionJobs map[string]map[string]*quantumJob`
- static `providers []map[string]any`
- static `quotas []map[string]any`

Recommended helper rules:
- when a session job is created, also register it in the canonical `jobs` map
- list routes should sort by `creationTime` then `id` for deterministic output
- timestamps should use UTC RFC3339
- status transitions should be deterministic rather than time-driven

## Recommended Stages

### Stage 0: Route, Naming, Coverage Wiring

Add the service skeleton without implementing the full surface:
- register a new provider file:
  - `internal/server/provider_azure_quantum_data_plane.go`
- reserve a canonical route root:
  - `/azure/quantum-data-plane/2026-01-15-preview/...`
- add coverage-script aliases for `quantum_data_plane`
- add the plan doc mapping in the doc coverage script

Exit criteria:
- service is discoverable in coverage scripts
- unknown routes resolve to explicit Azure `NotImplemented` behavior instead of falling through ambiguously

### Stage 1: Shared Data-Plane Foundation

Implement:
- auth/header parsing aligned with other Azure data-plane services
- `api-version` enforcement
- shared error envelope helpers
- shared JSON body parsing
- ID generation helpers for jobs and sessions

Exit criteria:
- router accepts the documented route families
- validation and not-found responses are deterministic

### Stage 2: Read-Only Discovery Endpoints

Implement first:
- `GET /providers`
- `GET /quotas`
- `GET /topLevelItems`
- `POST /storage/sasUri`

Why this stage first:
- low risk
- useful for examples and smoke tests
- no deep mutable state required

Exit criteria:
- all four routes return stable modeled payloads
- storage helper returns a realistic SAS-style URI

### Stage 3: Top-Level Jobs

Implement:
- create job
- get job
- list jobs
- cancel job
- delete job

Suggested fixtures:
- default status progression:
  - created jobs start at `Waiting`
  - cancel transitions to `Cancelled`
- preserve optional session linkage if the client submits `sessionId`

Exit criteria:
- jobs lifecycle works end to end with both happy-path and negative tests

### Stage 4: Sessions

Implement:
- open session
- get session
- list sessions
- close session
- delete session

Suggested semantics:
- open creates a session if no ID exists, otherwise can be treated as idempotent get/start
- close seals the session and blocks further mutable child operations if the docs imply that restriction

Exit criteria:
- sessions lifecycle works end to end
- invalid close/delete cases are covered

### Stage 5: Session Jobs

Implement:
- create session job
- get session job
- list session jobs
- cancel session job
- delete session job

Important rule:
- parent session existence must be validated consistently

Exit criteria:
- nested job lifecycle works
- session-job routes and top-level job routes stay internally consistent

### Stage 6: Azure Go SDK Example

Add:
- `examples/azure/azurequantum-dataplane-2026-01-15-preview/main.go`
- `examples/azure/azurequantum-dataplane-2026-01-15-preview/Dockerfile`
- `examples/azure/azurequantum-dataplane-2026-01-15-preview/docker-compose.yml`

Recommended example flow:
- list providers
- list quotas
- create a session
- create a session job
- get the session job
- list session jobs
- close the session
- create a top-level job
- cancel the top-level job
- fetch a storage SAS URI

SDK note:
- use `azcore/runtime` and `azcore/policy` with a shared-key or bearer-placeholder policy like the repo’s other preview data-plane examples

Exit criteria:
- example builds with `go build ./examples/azure/azurequantum-dataplane-2026-01-15-preview`
- example exercises the new routes without requiring external Azure dependencies

### Stage 7: Coverage And CI Validation

Update:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended validation commands after implementation:
- `go test ./internal/server -run 'TestAzureQuantumDataPlane' -count=1`
- `go build ./examples/azure/azurequantum-dataplane-2026-01-15-preview`
- `python3 scripts/azure-contract-coverage.py --service quantum_data_plane --require-service quantum_data_plane --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service quantum_data_plane --require-service quantum_data_plane --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service quantum_data_plane --require-service quantum_data_plane --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service quantum_data_plane --require-service quantum_data_plane --fail-on any`
- `make ci`

Success criteria:
- every Quantum data-plane endpoint under the implemented slice returns modeled success or modeled validation/not-found errors
- coverage scripts report the service as implemented
- no Quantum data-plane route falls through to `NotImplemented`

## Testing Strategy

Add focused tests for:
- providers, quotas, top-level items, and storage reads
- job create/get/list/cancel/delete
- session open/get/list/close/delete
- session job create/get/list/cancel/delete
- validation failures:
  - missing body fields
  - malformed JSON
  - invalid `api-version`
  - missing session parent
  - missing job/session IDs
  - invalid cancel/close transitions

Recommended test file:
- `internal/server/provider_azure_quantum_data_plane_test.go`

## Deferred / Nice-To-Have

Safe deferrals if the first implementation needs to stay narrow:
- rich provider-specific job schemas
- full continuation-token semantics
- advanced filtering and ordering on list routes
- real blob-backed storage SAS behavior
- time-driven terminal-state progression for jobs and sessions

Those can be added later without changing the service key, route family, or example path chosen above.
