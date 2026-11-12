# Azure Text to Speech (`text-to-speech-2024-04-01`) Staged Plan

## Objective

Emulate Azure Batch Text to Speech REST API (`2024-04-01`) with deterministic local behavior for batch synthesis job lifecycle and operation status tracking.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/operation-groups?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/batch-syntheses?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/batch-syntheses/create?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/batch-syntheses/delete?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/batch-syntheses/get?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/batch-syntheses/list?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/operations?view=rest-batchtexttospeech-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/batchtexttospeech/operations/get?view=rest-batchtexttospeech-2024-04-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full synthesis engine parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/batchtexttospeech/2024-04-01/batchsyntheses`
- `/azure/batchtexttospeech/2024-04-01/batchsyntheses/{id}`
- `/azure/batchtexttospeech/2024-04-01/operations/{id}`

Target API version:
- `api-version=2024-04-01` (query)

## API Surface and Contract Notes

Operation groups and operation pages reviewed:

- Batch Syntheses
  - `PUT {endpoint}/texttospeech/batchsyntheses/{id}?api-version=2024-04-01` (Create)
  - `DELETE {endpoint}/texttospeech/batchsyntheses/{id}?api-version=2024-04-01` (Delete)
  - `GET {endpoint}/texttospeech/batchsyntheses/{id}?api-version=2024-04-01` (Get)
  - `GET {endpoint}/texttospeech/batchsyntheses?api-version=2024-04-01` (List)
- Operations
  - `GET {endpoint}/texttospeech/operations/{id}?api-version=2024-04-01` (Get)

Contract notes from operation pages:
- `Ocp-Apim-Subscription-Key` is required.
- Create returns operation tracking headers (`operation-id`, `operation-location`) and task payload.
- Delete returns `204`.
- Get/List/Operation Get return `200` payloads.
- Error responses include a typed `Error Response` envelope and `x-ms-error-code` header.

## Stage 0: Contract Skeleton

- Add dedicated Azure Batch Text-to-Speech router.
- Recognize all operation-group routes listed above.
- Add route-recognition tests across all five documented operations.

Exit criteria:
- Route envelope is locked by tests and deterministic for recognized paths.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Validate `{id}` route shape at minimum (non-empty).
- Reject unsupported method/resource combinations outside documented operations.

Exit criteria:
- Invalid API version and unsupported method/resource patterns return deterministic contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic foundation success payloads for recognized routes.
- Keep stable route/path/provider envelope semantics used by other staged Azure services.

Exit criteria:
- Create/Get/List/Delete/Operation Get routes are deterministic and contract-tested.

## Stage 3: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/text-to-speech-2024-04-01`.
- Exercise all documented operation surfaces.
- Update Azure contract, IO, and doc coverage scripts with service aliases and plan mapping.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve service identifiers.
