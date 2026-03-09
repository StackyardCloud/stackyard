# Azure Search Service - Data Plane - Indexes (`search-service-data-plane-indexes-2025-09-01`) Staged Plan

## Objective

Emulate Azure Search Service Data Plane Indexes (`2025-09-01`) with deterministic local behavior for index lifecycle, analysis, statistics, and listing workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/analyze?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/create?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/create-or-update?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/delete?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/get?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/get-statistics?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexes/list?view=rest-searchservice-2025-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full analyzers/tokenizers behavior parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/indexes`
- `/azure/indexes('{indexName}')`
- `/azure/indexes('{indexName}')/search.analyze`
- `/azure/indexes('{indexName}')/search.stats`

Target API version:
- `api-version=2025-09-01`

## API Surface and Contract Notes

Indexes operations documented for this API version:

- `POST /indexes('{indexName}')/search.analyze`
- `POST /indexes`
- `PUT /indexes('{indexName}')`
- `DELETE /indexes('{indexName}')`
- `GET /indexes('{indexName}')`
- `GET /indexes('{indexName}')/search.stats`
- `GET /indexes`

Common contract characteristics from operation pages:
- Conditional create-or-update and delete semantics via `If-Match` and `If-None-Match`.
- Optional `allowIndexDowntime` query on create-or-update for schema evolution semantics.
- Standard Search service error envelopes for validation, auth/authz, conflict, throttling, and server failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Indexes router surface.
- Recognize all index lifecycle/analysis/statistics routes under `/azure/indexes*`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate route identifiers (`indexName`) and conditional headers (`If-Match`, `If-None-Match`) where applicable.
- Validate baseline request envelopes for create/create-or-update and analyze operations.

Exit criteria:
- Invalid requests return deterministic `400`/`412` style service contracts.

## Stage 2: Deterministic Index CRUD Fixtures

- Implement deterministic create/get/list/update/delete index fixtures with stable ETag behavior.
- Implement deterministic list shaping for `$select` with stable ordering.

Exit criteria:
- CRUD/index metadata behavior is deterministic and contract-tested.

## Stage 3: Analyze and Statistics Fixtures

- Implement deterministic `search.analyze` response fixtures for token stream shape.
- Implement deterministic `search.stats` usage/size metrics fixtures.

Exit criteria:
- Analyze/statistics routes are deterministic and contract-tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-service/search-service-data-plane-indexes-2025-09-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
