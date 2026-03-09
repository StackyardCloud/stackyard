# Azure AI Services Data Plane - Text Moderation (`ai-services-data-plane-text-moderation-v1.0`) Staged Plan

## Objective

Emulate Azure AI Services Data Plane Text Moderation REST operations in Stackyard with deterministic behavior for local integration and SDK testing.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/cognitiveservices/contentmoderator/text-moderation?view=rest-cognitiveservices-contentmoderator-v1.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not implement custom term list management APIs in this phase.
- Do not require live Azure resources.

## Stackyard Route Envelope

Stackyard will expose Text Moderation routes under:
- `/azure/contentmoderator/moderate/v1.0/ProcessText/*`

## API Surface (Initial Target)

### Detect Language
- `POST /contentmoderator/moderate/v1.0/ProcessText/DetectLanguage`

### Screen Text
- `POST /contentmoderator/moderate/v1.0/ProcessText/Screen/`
- `POST /contentmoderator/moderate/v1.0/ProcessText/Screen/?language={language}&autocorrect={autocorrect}&PII={PII}&listId={listId}&classify={classify}`

## Stage 0: Route Contract Lock

- Add Azure Text Moderation router.
- Recognize DetectLanguage and Screen routes.
- Return deterministic not-implemented responses for unsupported methods/routes.
- Add route contract tests.

Exit criteria:
- Route detection and method handling are test-locked.

## Stage 1: Input Handling and Validation

- Validate accepted content types (`text/plain`, `text/html`, `text/xml`, `text/markdown`).
- Validate text body presence for both operations.
- Validate Screen query parameters:
  - `language` (ISO 639-3 style string),
  - `autocorrect` boolean,
  - `PII` boolean,
  - `listId` positive integer when provided,
  - `classify` boolean.

Exit criteria:
- Invalid inputs return stable `400` envelopes with deterministic error codes/messages.

## Stage 2: Deterministic Response Fixtures

- Implement deterministic `200` response envelopes for:
  - DetectLanguage: `DetectedLanguage`, `Status`, `TrackingId`.
  - Screen: `OriginalText`, `NormalizedText`, `Language`, `Terms`, optional `PII`, optional `Classification`, `Status`, `TrackingId`.
- Include consistent `Status.Code=3000` and stable `TrackingId` generation.

Exit criteria:
- Positive-path responses follow documented shapes and are deterministic.

## Stage 3: Contract Test Coverage

- Add positive tests for DetectLanguage and Screen variants.
- Add negative tests for bad method, missing body, invalid booleans, invalid listId, unsupported routes.
- Ensure coverage scripts detect validation, fixture, and test signals.

Exit criteria:
- Service passes Azure contract and IO coverage gates.

## Stage 4: SDK Example and Docs Integration

- Add `examples/azure/ai-services/data-plane-text-moderation-v1.0` using Azure Go SDK primitives.
- Add Dockerfile and docker-compose wiring for runnable examples.
- Update Azure catalog and README routes/SDK notes.
- Update coverage script aliases for service naming variants.

Exit criteria:
- Service is discoverable, runnable, and covered via scripts.

## Stage 5: Hardening and Future Work

- Add richer profanity, PII, and classification fixtures for broader scenarios.
- Add compatibility toggles if SDK/client differences are identified.
- Keep service out of strict Azure gate list until parity expands.

Exit criteria:
- Stable baseline with clear expansion path.
