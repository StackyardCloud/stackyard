# Azure AI Services Data Plane - List Management Image Lists (`ai-services-data-plane-list-management-v1.0`) Staged Plan

## Objective

Emulate Azure AI Services Data Plane List Management Image Lists REST operations in Stackyard with deterministic behavior for local SDK and integration testing.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/list-management-image-lists?view=rest-cognitiveservices-contentmoderator-v1.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not emulate the full data-plane list-management surface outside image lists in this phase.
- Do not require live Azure resources.

## Stackyard Route Envelope

Stackyard will expose Image List Management routes under:
- `/azure/contentmoderator/lists/v1.0/imagelists`
- `/azure/contentmoderator/lists/v1.0/imagelists/{listId}`
- `/azure/contentmoderator/lists/v1.0/imagelists/{listId}/RefreshIndex`

## API Surface (Initial Target)

- `POST /contentmoderator/lists/v1.0/imagelists` (Create)
- `GET /contentmoderator/lists/v1.0/imagelists` (GetAll)
- `GET /contentmoderator/lists/v1.0/imagelists/{listId}` (GetDetails)
- `PUT /contentmoderator/lists/v1.0/imagelists/{listId}` (Update)
- `DELETE /contentmoderator/lists/v1.0/imagelists/{listId}` (Delete)
- `POST /contentmoderator/lists/v1.0/imagelists/{listId}/RefreshIndex` (RefreshIndex)

## Stage 0: Route Contract Lock

- Add Azure List Management router.
- Recognize image-list lifecycle paths.
- Return deterministic not-implemented responses for unsupported methods/routes.
- Add route contract tests.

Exit criteria:
- Route detection and method handling are test-locked.

## Stage 1: Input Handling and Validation

- Validate JSON payload for Create/Update.
- Enforce required fields (`Name`).
- Validate `listId` path segment as positive integer.
- Validate content type for mutating operations (`application/json`).

Exit criteria:
- Invalid inputs return stable `400` envelopes.

## Stage 2: Deterministic Response Fixtures

- Implement deterministic in-memory image-list store.
- Return stable Create/GetAll/GetDetails/Update/Delete shapes.
- Implement RefreshIndex response with deterministic `TrackingId` and success `Status`.

Exit criteria:
- Lifecycle responses are deterministic and shape-stable.

## Stage 3: Contract Test Coverage

- Add positive lifecycle tests across all target operations.
- Add negative tests for invalid JSON, missing Name, invalid listId, unsupported methods.
- Ensure coverage scripts detect validation, fixture, and test signals.

Exit criteria:
- Service passes Azure contract and IO coverage gates.

## Stage 4: SDK Example and Docs Integration

- Add `examples/azure/ai-services/data-plane-list-management-v1.0` using Azure Go SDK primitives.
- Add Dockerfile and docker-compose wiring for runnable example flow.
- Update Azure catalog and README route/SDK notes.
- Update coverage script aliases for requested service naming.

Exit criteria:
- Service is discoverable, runnable, and script-selectable.

## Stage 5: Hardening and Future Work

- Expand fixtures for richer metadata and list indexing behavior.
- Add compatibility toggles if client-specific behavior differences appear.
- Keep service out of strict Azure gate list until parity expands.

Exit criteria:
- Stable baseline with clear extension path.
