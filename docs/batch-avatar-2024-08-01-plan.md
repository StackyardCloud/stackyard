# Azure Batch Avatar (`batch-avatar-2024-08-01`) Staged Plan

## Objective

Emulate Azure Batch Avatar REST API (`2024-08-01`) with deterministic local behavior for avatar batch synthesis lifecycle and operation-status retrieval.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/operation-groups?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/avatar-batch-syntheses?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/operations?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/avatar-batch-syntheses/create-avatar-batch-synthesis?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/avatar-batch-syntheses/delete-avatar-batch-synthesis?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/avatar-batch-syntheses/get-avatar-batch-synthesis?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/avatar-batch-syntheses/list-avatar-batch-syntheses?view=rest-batchavatar-2024-08-01`
- `https://learn.microsoft.com/en-us/rest/api/batchavatar/operations/get-operation-status?view=rest-batchavatar-2024-08-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate real avatar rendering or media generation in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/batchavatar/2024-08-01/batchsyntheses`
- `/azure/batchavatar/2024-08-01/batchsyntheses/{id}`
- `/azure/batchavatar/2024-08-01/operations/{id}`

Target API version:
- `api-version=2024-08-01` (query)

## API Surface and Contract Notes

Operation groups and operation pages reviewed:

- Avatar Batch Syntheses
  - `PUT {endpoint}/avatar/batchsyntheses/{id}?api-version=2024-08-01` (Create)
  - `DELETE {endpoint}/avatar/batchsyntheses/{id}?api-version=2024-08-01` (Delete)
  - `GET {endpoint}/avatar/batchsyntheses/{id}?api-version=2024-08-01` (Get)
  - `GET {endpoint}/avatar/batchsyntheses?api-version=2024-08-01` (List)
- Operations
  - `GET {endpoint}/avatar/operations/{id}?api-version=2024-08-01` (Get operation status)

Contract notes from operation pages:
- `Ocp-Apim-Subscription-Key` is required.
- Create can return asynchronous operation metadata and operation resource links.
- Delete returns no-content semantics in the live API.
- Get/List/Operation Get return typed JSON payloads.
- Error responses use typed Azure error envelopes.

## Stage 0: Contract Skeleton

- Add dedicated Azure Batch Avatar router surface.
- Recognize all five documented operation routes.
- Add route-recognition tests for each operation.

Exit criteria:
- Route envelope is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Validate `{id}` route shape at minimum (non-empty).
- Reject unsupported resource/method combinations deterministically.

Exit criteria:
- Invalid query and unsupported method/resource combinations return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized routes.
- Preserve stable `provider/path/status` response envelope semantics used by existing Azure staged services.

Exit criteria:
- Create/Get/List/Delete/Operation routes are deterministic and contract-tested.

## Stage 3: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/openai-2024-08-01`.
- Exercise all documented operation surfaces.
- Update Azure contract, IO, and doc coverage scripts with service aliases and plan mapping.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve service identifiers.
