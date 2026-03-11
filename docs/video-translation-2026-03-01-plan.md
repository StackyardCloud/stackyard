# Azure Video Translation (`video-translation-2026-03-01`) Staged Plan

## Objective

Emulate Azure Video Translation data-plane REST API (`2026-03-01`) with deterministic local behavior for Event Hub configuration management, translation lifecycle, iteration lifecycle, and operation status inspection.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/operation-groups?view=rest-aiservices-videotranslation-2026-03-01`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/event-hub-configuration-operations?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/iteration-operations?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/operation-operations?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/translation-operations?view=rest-aiservices-videotranslation-2026-03-01`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/event-hub-configuration-operations/create-or-replace-event-hub-configuration?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/event-hub-configuration-operations/delete-event-hub-configuration?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/event-hub-configuration-operations/get-event-hub-configuration?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/event-hub-configuration-operations/ping-event-hub?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/iteration-operations/create-iteration?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/iteration-operations/get-iteration?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/iteration-operations/list-iteration?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/operation-operations/get-operation?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/translation-operations/create-translation?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/translation-operations/delete-translation?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/translation-operations/get-translation?view=rest-aiservices-videotranslation-2026-03-01`
- `https://learn.microsoft.com/en-us/rest/api/aiservices/videotranslation/translation-operations/list-translation?view=rest-aiservices-videotranslation-2026-03-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate long-running processing internals in this phase.

## Route Envelope

Documented endpoint route families:
- `{endpoint}/videotranslation/configurations/event-hub?api-version=2026-03-01`
- `{endpoint}/videotranslation/configurations/event-hub:ping?api-version=2026-03-01`
- `{endpoint}/videotranslation/translations/{translationId}?api-version=2026-03-01`
- `{endpoint}/videotranslation/translations?api-version=2026-03-01`
- `{endpoint}/videotranslation/translations/{translationId}/iterations/{iterationId}?api-version=2026-03-01`
- `{endpoint}/videotranslation/translations/{translationId}/iterations?api-version=2026-03-01`
- `{endpoint}/videotranslation/operations/{operationId}?api-version=2026-03-01`

Stackyard emulation prefix:
- `/azure/videotranslation/*`

## API Surface and Contract Notes

Operation groups and documented operations reviewed:
- Event Hub configuration:
  - create or replace configuration
  - get configuration
  - delete configuration
  - ping configuration
- Translation:
  - create/get/list/delete translation
- Iteration:
  - create/get/list iteration
- Operation:
  - get operation

Contract characteristics from method pages:
- Required query parameter: `api-version=2026-03-01`.
- Required request header: `Ocp-Apim-Subscription-Key`.
- Documented security also includes AAD bearer token flow (`cognitiveservices.azure.com/.default` scope).
- Success response patterns:
  - `200 OK`: get/list/ping and idempotent create variants.
  - `201 Created`: create translation and create iteration.
  - `204 No Content`: delete configuration and delete translation.
- Error contract:
  - shared `Azure.Core.Foundations.Error Response` payload with `x-ms-error-code` response header.

## Stage 0: Contract Skeleton

- Add dedicated Azure Video Translation router surface under `/azure/videotranslation/*`.
- Recognize documented operation families for Event Hub configuration, translation, iteration, and operation status.
- Add route-recognition tests across all documented methods.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate required path tokens for:
  - translation routes requiring `translationId` for item-level operations
  - iteration item routes requiring `iterationId`
  - operation item route requiring `operationId`
- Preserve deterministic staged behavior for unknown nested routes.

Exit criteria:
- Invalid path/query forms return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized Video Translation routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and validated by provider tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/video-translation-2026-03-01`.
- Exercise representative workflows for configuration lifecycle, translation lifecycle, iteration lifecycle, and operation get.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Video Translation API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `video-translation-2026-03-01` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and include it in coverage reporting.
