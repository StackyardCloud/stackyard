# Azure AI Services - Data Plane - Document Classifiers (`ai-services-data-plane-document-classifiers-v4.0`) Staged Plan

## Objective

Emulate Azure AI Services Document Intelligence Document Classifiers data-plane APIs in Stackyard with deterministic local behavior for build/copy/classify/list/get/delete flows.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers?view=rest-aiservices-v4.0%20(2024-11-30)`

Method references (reviewed):
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/authorize-classifier-copy?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/build-classifier?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/classify-document?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/classify-document-from-stream?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/copy-classifier-to?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/delete-classifier?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/get-classifier?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/get-classify-result?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/document-classifiers/list-classifiers?view=rest-aiservices-v4.0%20(2024-11-30)`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full Document Intelligence model-build internals in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/aiservices/documentintelligence/...`

Target API version:
- `api-version=2024-11-30`

## API Surface and Contract Notes

Operations and key contract expectations from the method pages:

- `POST /documentintelligence/documentClassifiers:authorizeCopy`
  - Input: `classifierId` (pattern constrained), optional `description`, optional `tags`.
  - Output: `200` with copy authorization payload (`accessToken`, target resource metadata, expiration).
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentClassifiers:build`
  - Input: `BuildDocumentClassifierRequest` (classifier identity, doc type source mapping, optional tags/description).
  - Output: `202` asynchronous operation with `Operation-Location` and result resource location semantics.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentClassifiers/{classifierId}:analyze`
  - Input: Analyze request body (document URL/payload) plus query options (`features`, `queryFields`, `pages`, `locale`, `stringIndexType`, `split`).
  - Output: `202` asynchronous classify operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentClassifiers/{classifierId}:analyze` (stream variant)
  - Input: Binary stream request body plus classify query options.
  - Output: `202` asynchronous classify operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `POST /documentintelligence/documentClassifiers/{classifierId}:copyTo`
  - Input: Copy authorization token payload from target (`targetResourceId`, `targetResourceRegion`, `targetClassifierId`, `accessToken`, `expirationDateTime`).
  - Output: `202` asynchronous copy operation.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `DELETE /documentintelligence/documentClassifiers/{classifierId}`
  - Output: `204` when deleted.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentClassifiers/{classifierId}`
  - Output: `200` with detailed classifier metadata.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentClassifiers/{classifierId}/analyzeResults/{resultId}`
  - Output: `200` classify result document payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/documentClassifiers`
  - Output: `200` paged list of classifiers.
  - Errors: Other status codes return `Document Intelligence Error Response`.

Shared error envelope focus:
- Model `error.code`, `error.message`, `error.target`, `error.details[]`, and nested `innererror` fields.

## Stage 0: Contract Skeleton

- Add dedicated Azure document classifier router surface and route recognition.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented operations.
- Add route recognition tests per operation family.

Exit criteria:
- Route envelope and status fallback are locked by tests.

## Stage 1: Classifier Resource Lifecycle

- Implement `authorizeCopy`, `build`, `copyTo`, `getClassifier`, `listClassifiers`, `deleteClassifier`.
- Add deterministic in-memory classifier metadata and LRO stubs for build/copy.

Exit criteria:
- Lifecycle operations behave deterministically with stable response shapes.

## Stage 2: Classification Operations

- Implement `classify-document` and `classify-document-from-stream` start operations.
- Implement `get-classify-result` polling surface.
- Support key query parameters (`pages`, `locale`, `stringIndexType`, `split`, `features`, `queryFields`).

Exit criteria:
- Analyze start + result retrieval flow is test-covered and deterministic.

## Stage 3: Error and Validation Hardening

- Enforce classifierId pattern and required payload fields.
- Add explicit `400/404/409` pathways aligned to docs contracts.
- Standardize `Document Intelligence Error Response` shape across handlers.

Exit criteria:
- Negative tests cover malformed requests, missing resources, and invalid transitions.

## Stage 4: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/ai-services-data-plane-document-classifiers-v4.0`.
- Update Azure coverage script aliases for service naming variants.
- Keep strict gate list unchanged until parity expands.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the new service identifier.
