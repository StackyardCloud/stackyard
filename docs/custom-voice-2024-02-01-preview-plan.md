# Azure Custom Voice (`custom-voice-2024-02-01-preview`) Staged Plan

## Objective

Emulate Azure Custom Voice REST API (`2024-02-01-preview`) with deterministic local behavior for base models, consents, custom voice endpoints, models, operations tracking, personal voices, projects, and training sets.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/operation-groups?view=rest-aiservices-speechapi-2024-02-01-preview`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/base-models?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/consents?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/models?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/operations?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/personal-voices?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/projects?view=rest-aiservices-speechapi-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/training-sets?view=rest-aiservices-speechapi-2024-02-01-preview`

Method references reviewed:
- Base Models
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/base-models/list?view=rest-aiservices-speechapi-2024-02-01-preview`
- Consents
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/consents/create?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/consents/delete?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/consents/get?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/consents/list?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/consents/post?view=rest-aiservices-speechapi-2024-02-01-preview`
- Endpoints
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints/create?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints/delete?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints/get?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints/list?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints/resume?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/endpoints/suspend?view=rest-aiservices-speechapi-2024-02-01-preview`
- Models
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/models/create?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/models/delete?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/models/get?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/models/list?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/models/list-recipes?view=rest-aiservices-speechapi-2024-02-01-preview`
- Operations
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/operations/get?view=rest-aiservices-speechapi-2024-02-01-preview`
- Personal Voices
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/personal-voices/create?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/personal-voices/delete?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/personal-voices/get?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/personal-voices/list?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/personal-voices/post?view=rest-aiservices-speechapi-2024-02-01-preview`
- Projects
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/projects/create?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/projects/delete?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/projects/get?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/projects/list?view=rest-aiservices-speechapi-2024-02-01-preview`
- Training Sets
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/training-sets/create?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/training-sets/delete?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/training-sets/get?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/training-sets/list?view=rest-aiservices-speechapi-2024-02-01-preview`
  - `https://learn.microsoft.com/en-us/rest/api/aiservices/speechapi/training-sets/upload-data?view=rest-aiservices-speechapi-2024-02-01-preview`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full voice training/inference semantics in this phase.

## Route Envelope

Documented route envelope under the service endpoint:
- `{endpoint}/customvoice/basemodels`
- `{endpoint}/customvoice/consents*`
- `{endpoint}/customvoice/endpoints*`
- `{endpoint}/customvoice/models*`
- `{endpoint}/customvoice/modelrecipes`
- `{endpoint}/customvoice/operations/{id}`
- `{endpoint}/customvoice/personalvoices*`
- `{endpoint}/customvoice/projects*`
- `{endpoint}/customvoice/trainingsets*`

Stackyard emulation prefix:
- `/azure/customvoice/*`

Target API version:
- `api-version=2024-02-01-preview` (query)

## API Surface and Contract Notes

Operation groups and documented operations reviewed:
- Base Models: list available base models.
- Consents: create/get/list/post/delete consent resources.
- Endpoints: create/get/list/delete endpoint resources plus `:resume` and `:suspend`.
- Models: create/get/list/delete models and list model recipes.
- Operations: get operation status.
- Personal Voices: create/get/list/post/delete personal voice resources.
- Projects: create/get/list/delete project resources.
- Training Sets: create/get/list/delete training sets and upload training data.

Common error-contract characteristics:
- Validation failures return structured service error payloads.
- Common classes include invalid request, authorization failures, not found, conflict, throttling, and internal errors.

## Stage 0: Contract Skeleton

- Add dedicated Azure Custom Voice router surface for `/azure/customvoice/*`.
- Recognize all operation-group route families listed above.
- Add route-recognition tests that cover representative methods across all operation groups.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when present.
- Validate top-level route/resource shape for documented operation groups.
- Enforce baseline method gating where required (`GET` list/get, `PUT` create/update, `POST` action, `DELETE` delete).

Exit criteria:
- Invalid query/method combinations return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized routes.
- Preserve stable `provider/path/status` payload semantics used across Azure staged services.

Exit criteria:
- Recognized routes are deterministic and covered by provider contract tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example in `examples/azure/ai-services/custom-voice-2024-02-01-preview`.
- Exercise representative calls across base models, consents, endpoints, models, operations, personal voices, projects, and training sets.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Custom Voice API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `custom-voice-2024-02-01-preview` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc-contract coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and pass strict signals.
