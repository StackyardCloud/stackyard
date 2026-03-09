# Azure AI Services Data Plane - Image Moderation (`ai-services-data-plane-image-moderation-v1.0`) Staged Plan

## Objective

Emulate Azure AI Services Data Plane Image Moderation REST operations in Stackyard with deterministic behavior for local integration and SDK testing.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/image-moderation?view=rest-cognitiveservices-contentmoderator-v1.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not implement the full data-plane moderation surface outside Image Moderation in this phase.
- Do not require live Azure resources.

## Stackyard Route Envelope

Stackyard will expose Image Moderation routes under:
- `/azure/contentmoderator/moderate/v1.0/ProcessImage/*`

## API Surface (Initial Target)

### Evaluate
- `POST /contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=stream&CacheImage={CacheImage}`
- `POST /contentmoderator/moderate/v1.0/ProcessImage/Evaluate?overload=url&CacheImage={CacheImage}`

### Find Faces
- `POST /contentmoderator/moderate/v1.0/ProcessImage/FindFaces?overload=stream&CacheImage={CacheImage}`
- `POST /contentmoderator/moderate/v1.0/ProcessImage/FindFaces?overload=url&CacheImage={CacheImage}`

### Match
- `POST /contentmoderator/moderate/v1.0/ProcessImage/Match?overload=stream&listId={listId}&CacheImage={CacheImage}`
- `POST /contentmoderator/moderate/v1.0/ProcessImage/Match?overload=url&listId={listId}&CacheImage={CacheImage}`

### OCR
- `POST /contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=stream&language={language}&CacheImage={CacheImage}&enhanced={enhanced}`
- `POST /contentmoderator/moderate/v1.0/ProcessImage/OCR?overload=url&language={language}&CacheImage={CacheImage}&enhanced={enhanced}`

## Stage 0: Route Contract Lock

- Add Azure AI Services Data Plane Image Moderation router.
- Recognize Evaluate/FindFaces/Match/OCR route patterns.
- Return deterministic not-implemented responses for unsupported methods/routes.
- Add route contract tests.

Exit criteria:
- Route detection and method handling are test-locked.

## Stage 1: Input Handling and Validation

- Add overload parsing (`stream`, `url`) and validation.
- For URL overload, validate JSON body (`DataRepresentation`, `Value`).
- For stream overload, validate binary body presence.
- Validate operation-specific parameters:
  - `listId` for Match,
  - `language` for OCR,
  - optional `CacheImage`/`enhanced` booleans.

Exit criteria:
- Invalid inputs return stable `400` envelopes with deterministic error codes/messages.

## Stage 2: Deterministic Response Fixtures

- Implement deterministic `200` response envelopes for:
  - Evaluate classification fields,
  - FindFaces coordinates/count,
  - Match result/match list,
  - OCR text/candidates/metadata.
- Include stable `Status` and `TrackingId` fields.

Exit criteria:
- Positive-path responses follow documented shapes and are deterministic.

## Stage 3: Contract Test Coverage

- Add positive lifecycle tests for URL and stream overloads.
- Add negative tests for bad query/body/missing required params and unsupported methods.
- Ensure coverage scripts detect validation, fixture, and test signals.

Exit criteria:
- Service passes Azure contract and IO coverage gates.

## Stage 4: SDK Example and Docs Integration

- Add `examples/azure/ai-services/data-plane-image-moderation-v1.0` using Azure Go SDK primitives.
- Add Dockerfile and docker-compose wiring for runnable examples.
- Update Azure catalog and README routes/SDK notes.
- Update coverage script aliases for service naming variants.

Exit criteria:
- Service is discoverable, runnable, and covered via scripts.

## Stage 5: Hardening and Future Work

- Add additional fixtures for broader OCR/match scenarios.
- Add optional compatibility toggles if behavior differences are found with clients.
- Keep service out of strict Azure gate list until expanded parity is complete.

Exit criteria:
- Stable baseline with clear expansion path.
