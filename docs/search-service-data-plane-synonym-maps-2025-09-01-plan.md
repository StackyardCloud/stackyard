# Azure Search Service - Data Plane - Synonym Maps (`search-service-data-plane-synonym-maps-2025-09-01`) Staged Plan

## Objective

Emulate Azure Search Service Data Plane Synonym Maps (`2025-09-01`) with deterministic local behavior for synonym map lifecycle and listing workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/searchservice/synonym-maps?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/synonym-maps/create?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/synonym-maps/create-or-update?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/synonym-maps/delete?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/synonym-maps/get?view=rest-searchservice-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/searchservice/synonym-maps/list?view=rest-searchservice-2025-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full query-time synonym expansion semantics in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/synonymmaps`
- `/azure/synonymmaps('{synonymMapName}')`

Target API version:
- `api-version=2025-09-01`

## API Surface and Contract Notes

Synonym Maps operations documented for this API version:

- `POST /synonymmaps`
- `PUT /synonymmaps('{synonymMapName}')`
- `DELETE /synonymmaps('{synonymMapName}')`
- `GET /synonymmaps('{synonymMapName}')`
- `GET /synonymmaps`

Common contract characteristics from operation pages:
- Conditional create-or-update and delete semantics via `If-Match` and `If-None-Match`.
- List endpoint supports selection and pagination style query options (`$select`, `$top`).
- Standard Search service error envelopes for validation, auth/authz, conflict, throttling, and server failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure Search Synonym Maps router surface.
- Recognize synonym map collection and named-resource routes under `/azure/synonymmaps*`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering all documented operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate route identifiers (`synonymMapName`) and conditional headers (`If-Match`, `If-None-Match`) where applicable.
- Validate baseline request envelopes for create/create-or-update.

Exit criteria:
- Invalid requests return deterministic `400`/`412` style service contracts.

## Stage 2: Deterministic Synonym Map CRUD Fixtures

- Implement deterministic create/get/list/update/delete synonym map fixtures with stable ETag behavior.
- Implement deterministic list shaping for `$select` with stable ordering.

Exit criteria:
- CRUD/metadata behavior is deterministic and contract-tested.

## Stage 3: Negative-Path and Concurrency Fixtures

- Add deterministic not-found and conflict/precondition fixtures for update/delete flows.
- Add deterministic throttling/service-failure fixtures for resilience testing.

Exit criteria:
- Negative path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/search-service/search-service-data-plane-synonym-maps-2025-09-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
