# Azure AI Services - Language - Analyze Conversations (`ai-services-language-analyze-conversations-2024-11-01`) Staged Plan

## Objective

Emulate Azure AI Services Language Analyze Conversations (`2024-11-01`) with deterministic local behavior for synchronous and long-running conversation analysis workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-conversations/operation-groups?view=rest-language-analyze-conversations-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-conversations/conversation-analysis/analyze-conversations?view=rest-language-analyze-conversations-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-conversations/submit-analysis-job/submit-analysis-job?view=rest-language-analyze-conversations-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-conversations/job-status/get-analysis-status?view=rest-language-analyze-conversations-2024-11-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-conversations/cancel-analysis-job/cancel-analysis-job?view=rest-language-analyze-conversations-2024-11-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full model-inference parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/language/:analyze-conversations`
- `/azure/language/analyze-conversations/jobs`
- `/azure/language/analyze-conversations/jobs/{jobId}`
- `/azure/language/analyze-conversations/jobs/{jobId}:cancel`

Target API version:
- `api-version=2024-11-01`

## API Surface and Contract Notes

Operation groups and operations documented for this API version:

- Conversation Analysis
  - `POST /language/:analyze-conversations`
  - Query: required `api-version`, optional `showStats`.
  - Body: synchronous conversation payload (`kind`, `analysisInput.conversationItem`, `parameters` including `projectName` and `deploymentName`).
  - Success: `200 OK` with analysis result envelope.
- Submit Analysis Job
  - `POST /language/analyze-conversations/jobs`
  - Body: batch/LRO payload (`analysisInput.conversations[]`, `tasks[]`, optional metadata like `displayName`).
  - Success: `202 Accepted`, with job tracking via response headers (including operation location).
- Job Status
  - `GET /language/analyze-conversations/jobs/{jobId}`
  - Query: required `api-version`, optional `showStats`.
  - Success: `200 OK` with job status, task progress, and result references.
- Cancel Analysis Job
  - `POST /language/analyze-conversations/jobs/{jobId}:cancel`
  - Success: `202 Accepted` for cancellation request acceptance.

Common error contract:
- Non-success responses use the language service error envelope (`ErrorResponse`) with service error codes and nested details.
- Typical failure classes include invalid request payloads/parameters, authorization failures, not found, throttling, and server-side failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure language analyze-conversations router surface.
- Recognize all four route envelopes listed above.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering each documented operation.

Exit criteria:
- Route envelope and fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate minimal request shapes:
  - synchronous analyze request envelope.
  - submit-job request envelope with `analysisInput.conversations[]` and `tasks[]`.
- Validate path-bound `jobId` requirements for status and cancel operations.

Exit criteria:
- Invalid/missing fields return deterministic `400` envelopes with stable error codes.

## Stage 2: Synchronous Analyze Fixture

- Implement deterministic `200` fixture for `POST /language/:analyze-conversations`.
- Return stable `result`, `warnings`, and optional `statistics` when `showStats=true`.
- Ensure response body schema remains stable across repeated runs.

Exit criteria:
- Sync analyze response is deterministic and contract-tested.

## Stage 3: LRO Job Lifecycle Fixture

- Implement deterministic `202` submission with stable job IDs and operation-location behavior.
- Implement deterministic `GET job status` transitions and terminal payload shape.
- Implement deterministic `POST ...:cancel` semantics for cancellable/completed jobs.

Exit criteria:
- Submit/get/cancel workflow is deterministic and contract-tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/ai-services-language-analyze-conversations-2024-11-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
