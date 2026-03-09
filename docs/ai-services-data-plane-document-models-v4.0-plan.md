# Azure AI Services - Data Plane - Document Models (`ai-services-data-plane-document-models-v4.0`) Staged Plan

## Objective

Emulate Azure AI Services Document Intelligence document-model data-plane APIs with deterministic local behavior for model build/compose/copy, analyze, analyze batch, and model/result retrieval flows.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models?view=rest-aiservices-v4.0%20(2024-11-30)`

Method references (reviewed):
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/analyze-batch-documents?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/analyze-document?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/analyze-document-from-stream?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/authorize-model-copy?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/build-model?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/compose-model?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/copy-model-to?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/delete-analyze-batch-result?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/delete-analyze-result?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/delete-model?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/get-analyze-batch-result?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/get-analyze-result?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/get-analyze-result-figure?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/get-analyze-result-pdf?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/get-model?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/list-analyze-batch-results?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-models/list-models?view=rest-aiservices-v4.0%20(2024-11-30)`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full model training/classification internals in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/aiservices/documentintelligence/...`

Target API version:
- `api-version=2024-11-30`

## API Surface and Contract Notes

Operations and key contract expectations from the method pages:

- `POST /documentintelligence/documentModels/{modelId}:analyzeBatch`
  - Input: batch analyze request with source and destination containers.
  - Output: `202` asynchronous analyze-batch operation with operation-location style headers.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentModels/{modelId}:analyze`
  - Input: JSON analyze request (`urlSource`/content source) with query options (`features`, `queryFields`, `pages`, `locale`, `stringIndexType`, `outputContentFormat`, etc.).
  - Output: `202` asynchronous analyze operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentModels/{modelId}:analyze` (stream variant)
  - Input: binary stream request body plus analyze query options.
  - Output: `202` asynchronous analyze operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentModels:authorizeCopy`
  - Input: target model metadata (`modelId`, optional `description`, optional `tags`).
  - Output: `200` copy authorization payload (`accessToken`, target resource metadata, expiration).
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentModels:build`
  - Input: build request (`modelId`, `buildMode`, training source, optional description/tags).
  - Output: `202` asynchronous model build operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentModels:compose`
  - Input: compose request (`modelId`, component models, optional description/tags).
  - Output: `202` asynchronous compose operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentModels/{modelId}:copyTo`
  - Input: copy authorization payload from target resource.
  - Output: `202` asynchronous copy operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `DELETE /documentintelligence/documentModels/{modelId}/analyzeBatchResults/{resultId}`
  - Output: `204` when analyze-batch result is deleted.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `DELETE /documentintelligence/documentModels/{modelId}/analyzeResults/{resultId}`
  - Output: `204` when analyze result is deleted.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `DELETE /documentintelligence/documentModels/{modelId}`
  - Output: `204` when model is deleted.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels/{modelId}/analyzeBatchResults/{resultId}`
  - Output: `200` with analyze-batch operation details/result envelope.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels/{modelId}/analyzeResults/{resultId}`
  - Output: `200` with analyze result payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels/{modelId}/analyzeResults/{resultId}/figures/{figureId}`
  - Output: `200` figure binary payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels/{modelId}/analyzeResults/{resultId}/pdf`
  - Output: `200` searchable PDF payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels/{modelId}`
  - Output: `200` model details payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels/{modelId}/analyzeBatchResults`
  - Output: `200` paged analyze-batch results.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentModels`
  - Output: `200` paged list of models.
  - Errors: Other status codes return `Document Intelligence Error Response`.

Shared error envelope focus:
- `error.code`, `error.message`, `error.target`, `error.details[]`, and nested `innererror`.

## Stage 0: Contract Skeleton

- Add dedicated Azure document-model router surface.
- Recognize all documented route envelopes for build/compose/copy/analyze/list/get/delete.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented operations.
- Add route recognition tests covering every method page route.

Exit criteria:
- Route envelope and `501` fallback behavior are locked by tests.

## Stage 1: Model Lifecycle Foundations

- Implement deterministic in-memory model resource state.
- Implement `authorizeCopy`, `build`, `compose`, `copyTo`, `getModel`, `listModels`, and `deleteModel`.
- Provide deterministic asynchronous operation stubs and operation-location semantics.

Exit criteria:
- Core model lifecycle routes are deterministic and shape-compatible.

## Stage 2: Analyze and Result Retrieval

- Implement `analyze` (JSON and stream) and `analyzeBatch` start operations.
- Implement `get-analyze-result`, `get-analyze-batch-result`, `list-analyze-batch-results`, `delete-analyze-result`, and `delete-analyze-batch-result`.
- Support key query options for analyze flows (pages/features/locale/string index/output format).

Exit criteria:
- Analyze start + retrieval/delete workflows are deterministic and test-covered.

## Stage 3: Binary Result Artifacts and Validation Hardening

- Implement `get-analyze-result-figure` and `get-analyze-result-pdf` binary responses.
- Enforce modelId/resultId/figureId and payload validation.
- Add explicit `400/404/409` pathways aligned to docs contracts and standardize error envelope.

Exit criteria:
- Binary artifact and negative-path behavior align with staged contract expectations.

## Stage 4: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/ai-services-data-plane-document-models-v4.0`.
- Update Azure coverage script aliases for service naming variants.
- Keep strict Azure gate list unchanged until parity expands.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the new service identifier.
