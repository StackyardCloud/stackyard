# Azure AI Services - Data Plane - Miscellaneous Operations (`ai-services-data-plane-miscellaneous-operations-v4.0`) Staged Plan

## Objective

Emulate Azure AI Services Document Intelligence miscellaneous data-plane operations with deterministic local behavior for operation status polling, operation listing, and resource details retrieval.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations?view=rest-aiservices-v4.0%20(2024-11-30)`

Method references (reviewed):
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-document-classifier-build-operation?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-document-classifier-copy-to-operation?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-document-model-build-operation?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-document-model-compose-operation?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-document-model-copy-to-operation?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-operation?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/get-resource-details?view=rest-aiservices-v4.0%20(2024-11-30)`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/miscellaneous-operations/list-operations?view=rest-aiservices-v4.0%20(2024-11-30)`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full backend model-training internals in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/aiservices/documentintelligence/...`

Target API version:
- `api-version=2024-11-30`

## API Surface and Contract Notes

Operations and key contract expectations from the method pages:

- `GET /documentintelligence/operations/{operationId}?_overload=getDocumentClassifierBuildOperation`
  - Output: `200` with `DocumentClassifierBuildOperationDetails`.
  - Key shape: operation metadata (`operationId`, `status`, `percentCompleted`, timestamps, `kind`, `resourceLocation`) plus classifier result.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/operations/{operationId}?_overload=getDocumentClassifierCopyToOperation`
  - Output: `200` with `DocumentClassifierCopyToOperationDetails`.
  - Key shape: operation metadata and copied classifier result payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/operations/{operationId}`
  - Output: `200` with `DocumentModelBuildOperationDetails`.
  - Key shape: operation metadata and model result payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/operations/{operationId}?_overload=getDocumentModelComposeOperation`
  - Output: `200` with `DocumentModelComposeOperationDetails`.
  - Key shape: compose operation metadata and model result payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/operations/{operationId}?_overload=getDocumentModelCopyToOperation`
  - Output: `200` with `DocumentModelCopyToOperationDetails`.
  - Key shape: copy operation metadata and model result payload.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/operations/{operationId}?_overload=getOperation`
  - Output: `200` with `DocumentIntelligenceOperationDetails` (discriminated union over build/compose/copy variants).
  - Key shape: `kind` drives the operation-details payload type.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/info`
  - Output: `200` with `DocumentIntelligenceResourceDetails`.
  - Key shape: resource quotas/counters, including `customDocumentModels.count` and `customDocumentModels.limit`.
  - Errors: Other status codes return `Document Intelligence Error Response`.

- `GET /documentintelligence/operations`
  - Output: `200` with `PagedDocumentIntelligenceOperationDetails`.
  - Key shape: `value[]` operation summaries and optional `nextLink`.
  - Errors: Other status codes return `Document Intelligence Error Response`.

Shared error envelope focus:
- `error.code`, `error.message`, `error.target`, `error.details[]`, and nested `innererror`.

## Stage 0: Contract Skeleton

- Add dedicated Azure miscellaneous-operations router surface.
- Recognize all documented operation paths and method envelopes.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented operations.
- Add route recognition tests per method route.

Exit criteria:
- Route envelope and `501` fallback behavior are locked by tests.

## Stage 1: Deterministic Read Fixtures

- Add deterministic fixtures for resource details and operation read/list payloads.
- Provide stable operation records keyed by `operationId`.
- Return typed payloads with operation metadata fields and expected `kind` values.

Exit criteria:
- `GET /info`, `GET /operations`, and `GET /operations/{operationId}` return stable typed success fixtures.

## Stage 2: Overload-Aware Operation Shapes

- Implement overload-aware response selection for operation detail variants:
  - `getDocumentClassifierBuildOperation`
  - `getDocumentClassifierCopyToOperation`
  - `getDocumentModelComposeOperation`
  - `getDocumentModelCopyToOperation`
  - `getOperation`
- Ensure `kind` and result payloads align with the selected operation type.

Exit criteria:
- Operation detail routes return correct variant shape and deterministic `kind`.

## Stage 3: Validation and Error Hardening

- Validate required `api-version` and `operationId` presence/shape.
- Implement explicit `400` (invalid query), `404` (unknown operation), and conflict pathways where relevant.
- Standardize `Document Intelligence Error Response` shape across handlers.

Exit criteria:
- Negative tests cover malformed requests and missing operation/resource scenarios.

## Stage 4: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/ai-services-data-plane-miscellaneous-operations-v4.0`.
- Update Azure coverage script aliases for service naming variants.
- Keep strict gate list unchanged until parity expands.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the new service identifier.
