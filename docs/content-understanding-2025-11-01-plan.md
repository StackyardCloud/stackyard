# Azure Content Understanding (`content-understanding-2025-11-01`) Staged Plan

## Objective

Emulate Azure Content Understanding REST API (`2025-11-01`) with deterministic local behavior for analyzer lifecycle, analysis submission, operation status, result retrieval, and service defaults.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/operation-groups?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers?view=rest-contentunderstanding-2025-11-01`

Method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/analyze?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/analyze-binary?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/copy?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/create-or-replace?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/delete?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/delete-result?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/get?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/get-defaults?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/get-operation-status?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/get-result?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/get-result-file?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/grant-copy-authorization?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/list?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/update?view=rest-contentunderstanding-2025-11-01`
- `https://learn.microsoft.com/en-us/rest/api/contentunderstanding/content-analyzers/update-defaults?view=rest-contentunderstanding-2025-11-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate long-running model internals beyond deterministic staged contracts.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/contentunderstanding/analyzers`
- `/azure/contentunderstanding/analyzers/{analyzerId}`
- `/azure/contentunderstanding/analyzers/{analyzerId}:analyze`
- `/azure/contentunderstanding/analyzers/{analyzerId}:analyzeBinary`
- `/azure/contentunderstanding/analyzers/{analyzerId}:copy`
- `/azure/contentunderstanding/analyzers/{analyzerId}:grantCopyAuthorization`
- `/azure/contentunderstanding/analyzers/{analyzerId}/operations/{operationId}`
- `/azure/contentunderstanding/analyzerResults/{operationId}`
- `/azure/contentunderstanding/analyzerResults/{operationId}/files/{fileId}`
- `/azure/contentunderstanding/defaults`

Target API version:
- `api-version=2025-11-01` (query)

## API Surface and Contract Notes

Operation groups covered:
- Content Analyzers

Common contract characteristics from method pages:
- API uses typed request/response JSON payloads for analyzer resources, operation status, and result retrieval.
- Method set includes read/write/update/delete along with action methods (`:analyze`, `:analyzeBinary`, `:copy`, `:grantCopyAuthorization`).
- Error responses include validation, auth/authz, conflict/not-found, and service-side failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure Content Understanding router with route ownership for all documented operations above.
- Recognize each documented method path and keep deterministic fallback for unknown nested routes under `/azure/contentunderstanding/`.
- Add route-recognition tests covering all method families.

Exit criteria:
- Route ownership is deterministic and complete for documented operation families.

## Stage 1: Request Validation Foundation

- Validate malformed `api-version` query values.
- Preserve deterministic invalid-request behavior for malformed routes.
- Keep unknown nested paths under service prefix deterministic.

Exit criteria:
- Invalid query/version paths return stable `400 InvalidRequest`, with contract tests in place.

## Stage 2: Deterministic Success Fixtures

- Return deterministic staged success fixtures for recognized methods.
- Preserve the shared Azure provider envelope fields (`provider`, `path`, `status`) used by contract tooling.

Exit criteria:
- All recognized routes return stable staged success payloads.

## Stage 3: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/content-understanding-2025-11-01`.
- Exercise representative calls for analyzer CRUD, analysis actions, operation/result retrieval, and defaults.
- Update Azure contract, IO, and doc coverage scripts with aliases and doc-plan mapping for `content_understanding`.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve `content-understanding-2025-11-01`.
