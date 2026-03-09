# Azure Search Service - Data Plane - Documents (`search-service-data-plane-documents-2025-09-01`) Staged Plan

## Objective

Emulate Azure Search Service Data Plane Documents (`2025-09-01`) with deterministic local behavior for indexing, lookup, query, suggest, and autocomplete workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/autocomplete?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/autocomplete-post?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/count?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/get?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/index?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/search-get?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/search-post?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/suggest?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/documents/suggest-post?view=rest-searchservice-2025-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate scoring-profile relevance parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/indexes('{indexName}')/docs`
- `/azure/indexes('{indexName}')/docs/{document-key-shape}`
- `/azure/indexes('{indexName}')/docs/$count`
- `/azure/indexes('{indexName}')/docs/search.autocomplete`
- `/azure/indexes('{indexName}')/docs/search.post.autocomplete`
- `/azure/indexes('{indexName}')/docs/search.index`
- `/azure/indexes('{indexName}')/docs/search.post.search`
- `/azure/indexes('{indexName}')/docs/search.suggest`
- `/azure/indexes('{indexName}')/docs/search.post.suggest`

Target API version:
- `api-version=2025-09-01`

## API Surface and Contract Notes

Documents operations documented for this API version:

- `GET /indexes('{indexName}')/docs/search.autocomplete`
- `POST /indexes('{indexName}')/docs/search.post.autocomplete`
- `GET /indexes('{indexName}')/docs/$count`
- `GET /indexes('{indexName}')/docs('{key}')`
- `POST /indexes('{indexName}')/docs/search.index`
- `GET /indexes('{indexName}')/docs`
- `POST /indexes('{indexName}')/docs/search.post.search`
- `GET /indexes('{indexName}')/docs/search.suggest`
- `POST /indexes('{indexName}')/docs/search.post.suggest`

Common contract characteristics from operation pages:
- Read/query endpoints return typed query envelopes (`200 OK` primary success contract).
- Indexing action endpoint accepts batch actions and can return partial success semantics.
- Error contracts include invalid request, auth/authz failure, throttling, index-not-found, and service failure classes.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Documents router surface.
- Recognize all `/indexes('{indexName}')/docs*` routes under the `/azure` envelope.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate index/document route identifiers.
- Validate baseline payload envelopes for index, search, suggest, and autocomplete POST routes.

Exit criteria:
- Invalid requests return deterministic `400` service-style error envelopes.

## Stage 2: Deterministic Read/Query Fixtures

- Implement deterministic fixtures for:
  - `GET docs('{key}')`
  - `GET docs`
  - `GET docs/$count`
  - autocomplete/suggest/search read responses.
- Keep response shapes stable across runs.

Exit criteria:
- Read/query operations return stable typed envelopes and pass contract tests.

## Stage 3: Indexing and Partial-Failure Fixtures

- Implement deterministic indexing action handling for upload/merge/delete action variants.
- Implement deterministic partial-success behavior for mixed action batches.
- Add consistent document state transitions used by subsequent read/query operations.

Exit criteria:
- Indexing lifecycle and mixed batch semantics are deterministic and contract-tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-service/search-service-data-plane-documents-2025-09-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
