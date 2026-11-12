# Azure Container Apps Data Plane (`containerapps-2025-10-02-preview`) Staged Plan

## Objective

Emulate the Azure Container Apps data-plane sessions API for `2025-10-02-preview` with deterministic local behavior for:

- session pool metadata lookup
- session lifecycle lookup and deletion
- session listing
- code execution submission and execution-result retrieval
- session file upload, list, metadata lookup, content download, and deletion
- an Azure Go SDK-based example for `containerapps-2025-10-02-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-data-plane-containerapps-2025-10-02-preview` is not the ARM `Microsoft.App` management plane already implemented in this repo.
- It is a session-oriented data-plane API rooted at a session-pool endpoint, with OAuth scope `dynamicsessions.io/.default`.
- This plan intentionally treats it as a separate Stackyard service from ARM `containerapps`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/operation-groups?view=rest-data-plane-containerapps-2025-10-02-preview`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/code-execution?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/list-sessions?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-management?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-pool-metadata?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-resource-files?view=rest-data-plane-containerapps-2025-10-02-preview`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/code-execution/execute?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/code-execution/get?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/list-sessions/get?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-management/delete?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-management/get?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-pool-metadata/get?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-resource-files/delete?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-resource-files/get?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-resource-files/get-content?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-resource-files/list?view=rest-data-plane-containerapps-2025-10-02-preview`
- `https://learn.microsoft.com/en-us/rest/api/data-plane/containerapps/session-resource-files/upload?view=rest-data-plane-containerapps-2025-10-02-preview`

SDK selection note:
- I did not find a service-specific official Azure Go SDK package for this data-plane surface in the Azure SDK index during review.
- Plan assumption: the example should use the official Azure Go SDK core runtime pipeline (`azcore/runtime` and `azcore/policy`) rather than inventing a non-existent typed client.

## Service Naming

Canonical Stackyard naming:
- service key: `containerapps_data_plane`
- display name: `Azure Container Apps Data Plane`
- API version token: `2025-10-02-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_containerapps_data_plane.go`
- tests: `internal/server/provider_azure_containerapps_data_plane_test.go`
- example: `examples/azure/containerapps-2025-10-02-preview`
- plan doc: `docs/containerapps-data-plane-2025-10-02-preview-plan.md`

Recommended coverage aliases:
- `containerapps_data_plane`
- `container_apps_data_plane`
- `containerapps-data-plane`
- `containerapps_sessions`
- `container_apps_sessions`
- `azure_containerapps_data_plane`
- `rest_data_plane_containerapps_2025_10_02_preview`
- `containerapps_2025_10_02_preview`

## Surface Summary

The reviewed REST surface is split into 5 operation groups:

1. `Code Execution`
2. `List Sessions`
3. `Session Management`
4. `Session Pool Metadata`
5. `Session Resource Files`

Total linked method pages reviewed: `11`

Method inventory:

`Code Execution`
- `Execute`
- `Get`

`List Sessions`
- `Get`

`Session Management`
- `Delete`
- `Get`

`Session Pool Metadata`
- `Get`

`Session Resource Files`
- `Delete`
- `Get`
- `Get Content`
- `List`
- `Upload`

## Route Envelope

Documented route family:
- `POST {endpoint}/executions?api-version=2025-10-02-preview&identifier={identifier}`
- `GET {endpoint}/executions/{executionId}?api-version=2025-10-02-preview&identifier={identifier}`
- `GET {endpoint}/listSessions?api-version=2025-10-02-preview`
- `GET {endpoint}/session?api-version=2025-10-02-preview&identifier={identifier}`
- `DELETE {endpoint}/session?api-version=2025-10-02-preview&identifier={identifier}`
- `GET {endpoint}/metadata?api-version=2025-10-02-preview`
- `POST {endpoint}/files?api-version=2025-10-02-preview&identifier={identifier}`
- `GET {endpoint}/files?api-version=2025-10-02-preview&identifier={identifier}`
- `GET {endpoint}/files/{name}?api-version=2025-10-02-preview&identifier={identifier}`
- `GET {endpoint}/files/{name}/content?api-version=2025-10-02-preview&identifier={identifier}`
- `DELETE {endpoint}/files/{name}?api-version=2025-10-02-preview&identifier={identifier}`

Stackyard emulation prefix should remain distinct from ARM `containerapps`.

Recommended Stackyard route family:
- `/azure/containerapps-data-plane/2025-10-02-preview/metadata`
- `/azure/containerapps-data-plane/2025-10-02-preview/session`
- `/azure/containerapps-data-plane/2025-10-02-preview/listSessions`
- `/azure/containerapps-data-plane/2025-10-02-preview/executions`
- `/azure/containerapps-data-plane/2025-10-02-preview/executions/{executionId}`
- `/azure/containerapps-data-plane/2025-10-02-preview/files`
- `/azure/containerapps-data-plane/2025-10-02-preview/files/{name}`
- `/azure/containerapps-data-plane/2025-10-02-preview/files/{name}/content`

Important routing note:
- This API does not use the Azure ARM `subscriptions/.../providers/...` envelope.
- It does require `api-version` on every method.
- The upstream docs model `{endpoint}` as a pool-specific base URL, so Stackyard should emulate the service as a data-plane root and treat session-pool identity as local emulator state rather than ARM resource identity.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- Security is OAuth2 with scope `dynamicsessions.io/.default`.
- Errors use `Azure.Core.Foundations.ErrorResponse`.
- Most success responses are immediate and synchronous.
- The session identifier query parameter is required on session-bound routes and is constrained by a documented pattern/length range.
- File content download is the only reviewed operation that explicitly returns binary media types.

### Session Pool Metadata

`Session Pool Metadata - Get`
- `GET /metadata`
- response: `Session Pool Metadata Response`
- sample response includes:
  - `runtimes[]`
  - `preInstalledPackages[]`
  - `createdAt`
  - `lastUpdatedAt`
  - `jupyterKernel`

Modeling implications:
- implement this first as the root discovery/readiness endpoint for the data-plane shim
- return deterministic runtime/package/kernel metadata aligned with the example and code-execution fixtures

### Sessions

`Session Management - Get`
- `GET /session?identifier={identifier}`
- response: `Session`
- sample fields:
  - `identifier`
  - `expireAt`
  - `createdAt`
  - `lastAccessedAt`
  - `etag`

`Session Management - Delete`
- `DELETE /session?identifier={identifier}`
- success: `204 No Content`

`List Sessions - Get`
- `GET /listSessions`
- optional query: `skip`
- response: `Sessions List Response`
- sample payload shape:
  - `sessions[]`

Modeling implications:
- persist sessions by user-facing `identifier`
- keep session creation implicit in first-pass execution/file operations if the docs do not expose an explicit create route
- update `lastAccessedAt` when session-bound operations succeed
- support stable list ordering and simple offset pagination via `skip`

### Code Execution

`Code Execution - Execute`
- `POST /executions?identifier={identifier}`
- optional request header: `operation-id`
- required body field:
  - `timeoutInSeconds`
- request body options include:
  - `code`
  - `codeInputType`
  - `executionType`
  - `outputStreamsMaxLength`
  - `returnJupyterFormatExecutionResult`
  - `shellCommand`
  - `execCommandAndArgs`
  - `stdin`
  - `useEmptyKernel`
- success: `200 OK`
- response headers may include:
  - `Operation-Location`
  - `operation-id`
- response type: `Session Code Execution Resource`
- example statuses include:
  - `Running`
  - `Succeeded`

`Code Execution - Get`
- `GET /executions/{executionId}?identifier={identifier}`
- response type: `Session Code Execution Resource`

Modeling implications:
- execution resources need independent storage keyed by `executionId`
- implement both synchronous and asynchronous fixture modes
- `Execute` and `Get` must share the same persisted execution record
- preserve shell-vs-inline request differences, but do not actually execute code
- return deterministic `stdout`, `stderr`, `executionResult`, and `executionTimeInMilliseconds`
- support Jupyter-format result examples for primitive/null-like values

### Session Resource Files

`Upload`
- `POST /files?identifier={identifier}[&path={path}]`
- multipart form-data with required `file`
- success: `200 OK`
- response type: `Session Resource File`

`List`
- `GET /files?identifier={identifier}[&path={path}&recursive={recursive}]`
- success: `200 OK`
- response type: `Paged Session Resource File`
- sample payload:
  - `value[]`
  - `nextLink`

`Get`
- `GET /files/{name}?identifier={identifier}[&path={path}]`
- success: `200 OK`
- response type: `Session Resource File`

`Get Content`
- `GET /files/{name}/content?identifier={identifier}[&path={path}]`
- success: `200 OK`
- media types:
  - `application/octet-stream`
  - `application/json`

`Delete`
- `DELETE /files/{name}?identifier={identifier}[&path={path}]`
- success: `204 No Content`

`Session Resource File` shape includes:
- `name`
- `directory`
- `type`
- `contentType`
- `sizeInBytes`
- `lastModifiedAt`

Modeling implications:
- keep a per-session virtual filesystem rooted by optional `path`
- store both metadata and raw content
- `List` should return both file and directory entries, with deterministic casing and `nextLink: null` in the first slice
- `Get Content` should return bytes, not JSON metadata

## Non-Goals

- Do not implement real code execution, kernels, or shell subprocesses.
- Do not implement real session pool scaling or scheduler behavior.
- Do not implement Azure AD token validation beyond the repo’s existing emulator conventions.
- Do not implement signed URLs or streamed chunk uploads in the first slice.
- Do not merge this surface into the existing ARM `containerapps` provider.

## Planned Repo Shape

Implementation files:
- `internal/server/provider_azure_containerapps_data_plane.go`
- `internal/server/provider_azure_containerapps_data_plane_test.go`

Example files:
- `examples/azure/containerapps-2025-10-02-preview/main.go`
- `examples/azure/containerapps-2025-10-02-preview/Dockerfile`
- `examples/azure/containerapps-2025-10-02-preview/docker-compose.yml`

Coverage updates:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Stage 0: Service Identity And Coverage Wiring

- Reserve canonical service key `containerapps_data_plane`
- Add aliases in all 3 Azure coverage scripts
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add a distinct data-plane router claim so paths do not collide with ARM `containerapps`

Exit criteria:
- coverage scripts discover the new canonical service key
- docs coverage can map the service to this plan doc
- route recognition is separated from management-plane `Microsoft.App`

## Stage 1: Shared Envelope, Validation, And Storage

- Implement data-plane router and storage for:
  - sessions
  - executions
  - session files
- Validate:
  - required `api-version`
  - required `identifier` where documented
  - `identifier` length/pattern rules
  - required multipart `file` on upload
  - required `timeoutInSeconds` on execute
  - mutually exclusive `shellCommand` vs `execCommandAndArgs`
- Return consistent `Azure.Core.Foundations.ErrorResponse` payloads

Exit criteria:
- invalid requests fail deterministically with typed error payloads
- the emulator has stable in-memory stores for sessions, executions, and files

## Stage 2: Metadata And Session Read APIs

- Implement:
  - `GET /metadata`
  - `GET /session`
  - `GET /listSessions`
- Return deterministic metadata payloads:
  - runtimes
  - packages
  - kernel info
- Return stable session payloads and session lists

Exit criteria:
- read-only discovery endpoints are fully covered
- session listing and lookup work without placeholder `NotImplemented`

## Stage 3: Code Execution

- Implement:
  - `POST /executions`
  - `GET /executions/{executionId}`
- Persist execution resources
- Support fixture branches for:
  - synchronous success
  - asynchronous running
  - asynchronous completed
  - shell-command mode
  - Jupyter-format primitive/null result cases
- Emit `Operation-Location` and `operation-id` headers

Exit criteria:
- execute/get routes share persisted execution state
- example responses match documented status/result shapes

## Stage 4: File Operations

- Implement:
  - `POST /files`
  - `GET /files`
  - `GET /files/{name}`
  - `GET /files/{name}/content`
  - `DELETE /files/{name}`
- Support optional `path`
- Support `recursive` for list with shallow deterministic directory traversal
- Return correct binary payloads for content download

Exit criteria:
- file routes round-trip metadata and content cleanly
- list/get/content/delete return modeled payloads instead of generic placeholders

## Stage 5: Example And Coverage Validation

- Add Azure Go SDK example under `examples/azure/containerapps-2025-10-02-preview`
- Use Azure SDK core runtime pipeline:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
- Example flow should cover:
  - metadata get
  - session list/get
  - execute
  - execution get
  - file upload
  - file list/get/get-content/delete
  - session delete
- Update coverage scripts and add focused provider tests

Exit criteria:
- example builds and exercises the modeled data-plane flow
- service-specific contract/io/doc coverage passes cleanly

## Testing Strategy

Add focused tests for:

- route recognition without ARM collisions
- invalid and missing `identifier`
- invalid `api-version`
- missing execute timeout
- conflicting shell execution fields
- missing multipart file upload
- successful metadata/session/execution/file flows
- `GET content` returns bytes with expected media type
- delete operations return `204`

Representative contract-style assertions:
- session lookup returns `identifier`, timestamps, and `etag`
- list sessions returns `sessions[]`
- execute returns execution resource plus operation headers
- execution get can return `Running` and `Succeeded`
- file list returns `value[]` and `nextLink`
- file get returns metadata while file content returns raw bytes

## Risks And Implementation Notes

- The session identifier regex allows a broad character set; the first pass should likely validate non-empty length bounds while being conservative about rejecting edge punctuation until coverage requires exact regex enforcement.
- Because the upstream endpoint is pool-scoped, Stackyard should avoid overfitting to ARM identity concepts.
- File uploads are multipart; tests and example code need to cover encoder behavior, not just JSON calls.
- `Get Content` is the easiest place to accidentally return JSON instead of bytes; it should have explicit test coverage.
- This service overlaps in product name with ARM Container Apps but not in protocol shape, so the router boundary must stay strict.
