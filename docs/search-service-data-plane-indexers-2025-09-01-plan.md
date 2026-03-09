# Azure Search Service - Data Plane - Indexers (`search-service-data-plane-indexers-2025-09-01`) Staged Plan

## Objective

Emulate Azure Search Service Data Plane Indexers (`2025-09-01`) with deterministic local behavior for indexer lifecycle, execution, reset, status, and listing workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/create?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/create-or-update?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/delete?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/get?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/get-status?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/list?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/reset?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/indexers/run?view=rest-searchservice-2025-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full connector crawling/indexing throughput in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/indexers`
- `/azure/indexers('{indexerName}')`
- `/azure/indexers('{indexerName}')/search.status`
- `/azure/indexers('{indexerName}')/search.reset`
- `/azure/indexers('{indexerName}')/search.run`

Target API version:
- `api-version=2025-09-01`

## API Surface and Contract Notes

Indexers operations documented for this API version:

- `POST /indexers`
- `PUT /indexers('{indexerName}')`
- `DELETE /indexers('{indexerName}')`
- `GET /indexers('{indexerName}')`
- `GET /indexers('{indexerName}')/search.status`
- `GET /indexers`
- `POST /indexers('{indexerName}')/search.reset`
- `POST /indexers('{indexerName}')/search.run`

Common contract characteristics from operation pages:
- Conditional create-or-update and delete semantics via `If-Match` and `If-None-Match`.
- Runtime control endpoints (`search.run`, `search.reset`) with no-body success status contracts.
- Status endpoint includes execution history/state contracts and service error envelopes.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Indexers router surface.
- Recognize all indexer lifecycle and execution routes under `/azure/indexers*`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate route identifiers (`indexerName`) and conditional headers (`If-Match`, `If-None-Match`) where applicable.
- Validate baseline payload envelopes for create/create-or-update.

Exit criteria:
- Invalid requests return deterministic `400`/`412` style service contracts.

## Stage 2: Deterministic Indexer CRUD Fixtures

- Implement deterministic create/get/list/update/delete indexer fixtures with stable ETag behavior.
- Ensure list supports baseline query shaping (`$select`) and stable ordering.

Exit criteria:
- CRUD behavior is deterministic and contract-tested.

## Stage 3: Execution and Status Fixtures

- Implement deterministic `search.run` and `search.reset` semantics.
- Implement deterministic `search.status` payloads with stable execution state/history transitions.

Exit criteria:
- Run/reset/status flows are deterministic and contract-tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-service/search-service-data-plane-indexers-2025-09-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
