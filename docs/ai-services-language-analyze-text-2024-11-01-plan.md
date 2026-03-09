# Azure AI Services - Language - Analyze Text (`ai-services-language-analyze-text-2024-11-01`) Staged Plan

## Objective

Emulate Azure AI Services Language Analyze Text (`2024-11-01`) with deterministic local behavior for synchronous text analysis requests across supported task kinds.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text/analyze-text/analyze-text?view=rest-language-analyze-text-2024-11-01&tabs=HTTP`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full NLP model inference parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/language/:analyze-text`

Target API version:
- `api-version=2024-11-01`

## API Surface and Contract Notes

Documented operation:

- `POST /language/:analyze-text`
  - Query: required `api-version`, optional `showStats`.
  - Body envelope:
    - `kind` selects task type.
    - `analysisInput.documents[]` requires document content (`id`, `text`) with optional `language`.
    - `parameters` shape depends on selected `kind`.
  - Supported task kinds in this API shape:
    - `EntityRecognition`
    - `EntityLinking`
    - `KeyPhraseExtraction`
    - `LanguageDetection`
    - `PiiEntityRecognition`
    - `SentimentAnalysis`
  - Output: `200` with task result envelope containing task-specific `results`, `warnings`, and optional `statistics` when `showStats=true`.
  - Errors: other status codes use Language service error response envelope with `error` and `x-ms-error-code`.

## Stage 0: Contract Skeleton

- Add dedicated Azure language analyze-text router surface.
- Recognize `/azure/language/:analyze-text` route envelope.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route recognition tests.

Exit criteria:
- Route envelope and fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` and basic JSON request envelope.
- Validate required fields:
  - `kind`
  - `analysisInput.documents[]`
  - per-document `id` and `text`
- Validate supported `kind` values and return deterministic `400` for unsupported kinds.

Exit criteria:
- Request validation and negative-path tests are deterministic.

## Stage 2: Deterministic Task Result Fixtures

- Implement deterministic `200` response fixtures for each supported `kind`.
- Return task-specific `results.documents[]` with stable mock outputs:
  - entities/entity links
  - key phrases
  - language detection scores
  - pii entities/redaction
  - sentiment/assessments/opinions skeletons
- Include `warnings` and `modelVersion` fields in response envelope.

Exit criteria:
- All task kinds produce stable typed result payloads.

## Stage 3: Stats, Errors, and Contract Hardening

- Implement `showStats=true` response behavior with deterministic statistics fields.
- Standardize error envelope and `x-ms-error-code` headers for `400/404/429/500` style paths used in emulation.
- Add test coverage for malformed documents, invalid parameters, and unsupported kind transitions.

Exit criteria:
- Success and error response shapes are stable and contract-tested.

## Stage 4: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/ai-services-language-analyze-text-2024-11-01`.
- Update Azure coverage script aliases for this service naming variant.
- Keep strict Azure gate list unchanged until feature parity expands.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
