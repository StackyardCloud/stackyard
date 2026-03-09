# Azure AI Services - Language - Question Answering (`ai-services-language-question-answering-2023-04-01`) Staged Plan

## Objective

Emulate Azure AI Services Language Question Answering (`2023-04-01`) with deterministic local behavior for knowledge-base query and query-from-text workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/language/question-answering/operation-groups?view=rest-language-question-answering-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/question-answering/question-answering/get-answers?view=rest-language-question-answering-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/question-answering/question-answering/get-answers-from-text?view=rest-language-question-answering-2023-04-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement retrieval-model quality parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/language/:query-knowledgebases`
- `/azure/language/:query-text`

Target API version:
- `api-version=2023-04-01`

## API Surface and Contract Notes

Operations documented for this API version:

- `POST /language/:query-knowledgebases`
  - Required query: `api-version`, `projectName`, `deploymentName`.
  - Request body includes question query payload (for example: `question`, optional ranking/top thresholds and answer-context flags).
  - Success: `200 OK` with answer list and metadata.
- `POST /language/:query-text`
  - Required query: `api-version`.
  - Request body includes inline text records and question payload.
  - Success: `200 OK` with generated answers from provided text.

Common contract characteristics from operation pages:
- Runtime inference style endpoints (no resource CRUD lifecycle).
- Shared language-service error envelope for validation, auth, throttling, not-found, and server failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure question-answering router surface.
- Recognize both runtime paths listed above.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests for both operations.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate required query parameters for `:query-knowledgebases` (`projectName`, `deploymentName`).
- Validate minimal request payload envelope for both operations (`question` and text record shape where required).

Exit criteria:
- Invalid requests return deterministic `400` service-style error envelopes.

## Stage 2: Deterministic Response Fixtures

- Implement deterministic response fixture for `POST /language/:query-knowledgebases`.
- Implement deterministic response fixture for `POST /language/:query-text`.
- Include stable answer IDs, confidence scores, and source metadata fields.

Exit criteria:
- Both operations return stable `200` response contracts and are contract-tested.

## Stage 3: Error and Edge-Case Fixtures

- Add deterministic not-found semantics for missing projects/deployments.
- Add deterministic throttling and service-error fixtures for resilience testing.
- Add edge-case coverage for empty text records, unsupported language hints, and low-confidence thresholds.

Exit criteria:
- Negative path contracts are deterministic and covered by tests.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/language-question-answering-2023-04-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
