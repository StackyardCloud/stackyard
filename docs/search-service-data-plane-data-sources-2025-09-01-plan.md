# Azure Search Service - Data Plane - Data Sources (`search-service-data-plane-data-sources-2025-09-01`) Staged Plan

## Objective

Emulate Azure Search Service Data Plane Data Sources (`2025-09-01`) with deterministic local behavior for data-source create/update/delete/get/list workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchservice/operation-groups?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/data-sources?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/data-sources/create?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/data-sources/create-or-update?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/data-sources/delete?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/data-sources/get?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/data-sources/list?view=rest-searchservice-2025-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full upstream connector execution semantics in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/datasources`
- `/azure/datasources('{dataSourceName}')`

Target API version:
- `api-version=2025-09-01`

## API Surface and Contract Notes

Data Sources operations documented for this API version:

- `POST /datasources`
- `PUT /datasources('{dataSourceName}')`
- `DELETE /datasources('{dataSourceName}')`
- `GET /datasources('{dataSourceName}')`
- `GET /datasources`

Common contract characteristics from operation pages:
- Conditional update/delete semantics via `If-Match` and `If-None-Match` request headers.
- Standard service error envelopes for validation, auth/authz, not-found, conflict/precondition, throttling, and server failures.
- List endpoint supports selection/paging style query options (`$select`, `$top`, continuation token).

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Data Sources router surface.
- Recognize both collection and named-resource data-source routes under `/azure`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate data-source name identifier and route shape.
- Validate core payload envelope for create/create-or-update.
- Validate conditional request headers (`If-Match`, `If-None-Match`) behavior.

Exit criteria:
- Invalid requests return deterministic `400`/`412` style service contracts.

## Stage 2: Deterministic CRUD Fixtures

- Implement deterministic create/get/list fixtures for data sources.
- Implement deterministic create-or-update semantics with stable ETag behavior.
- Implement deterministic delete semantics including not-found and precondition conflicts.

Exit criteria:
- CRUD flows are stable across repeated runs and contract-tested.

## Stage 3: List Filtering and Continuation Fixtures

- Implement deterministic list shaping for `$select` and `$top`.
- Implement deterministic continuation token behavior for paged listings.
- Ensure stable ordering and ETag surface across list/get operations.

Exit criteria:
- List and pagination contracts are deterministic and tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-service/search-service-data-plane-data-sources-2025-09-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
