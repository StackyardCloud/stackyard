# Azure Speaker Recognition (`speaker-recognition-2021-09-05`) Staged Plan

## Objective

Emulate Azure Speaker Recognition REST API (`2021-09-05`) with deterministic local behavior for profile lifecycle, enrollment, phrase listing, verification, and identification workflows.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/operation-groups?view=rest-speaker-recognition-2021-09-05`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29?view=rest-speaker-recognition-2021-09-05`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/create-enrollment?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/create-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/delete-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/get-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/list-phrases?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/list-profiles?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/reset-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-dependent/verify-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/create-enrollment?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/create-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/delete-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/get-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/list-activation-phrases?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/list-profiles?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/reset-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent%28verification%29/verify-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/create-enrollment?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/create-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/delete-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/get-profile?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/identify-single-speaker?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/list-activation-phrases?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/list-profiles?view=rest-speaker-recognition-2021-09-05`
- `https://learn.microsoft.com/en-us/rest/api/speaker-recognition/text-independent/reset-profile?view=rest-speaker-recognition-2021-09-05`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full biometric model behavior in this phase.

## Route Envelope

Documented endpoint route families:
- `{endpoint}/speaker-recognition/verification/text-dependent/profiles*`
- `{endpoint}/speaker-recognition/verification/text-dependent/phrases/{locale}`
- `{endpoint}/speaker-recognition/verification/text-independent/profiles*`
- `{endpoint}/speaker-recognition/verification/text-independent/phrases/{locale}`
- `{endpoint}/speaker-recognition/identification/text-independent/profiles*`
- `{endpoint}/speaker-recognition/identification/text-independent/profiles:identifySingleSpeaker`
- `{endpoint}/speaker-recognition/identification/text-independent/phrases/{locale}`

Stackyard emulation prefix:
- `/azure/speaker-recognition/*`

## API Surface and Contract Notes

Operation groups and documented operations reviewed:
- Text Dependent:
  - profile create/list/get/delete/reset
  - create enrollment
  - list pass phrases
  - verify profile
- Text Independent:
  - profile create/list/get/delete/reset
  - create enrollment
  - list activation phrases
  - identify single speaker (`profiles:identifySingleSpeaker`)
- Text Independent (Verification):
  - profile create/list/get/delete/reset
  - create enrollment
  - list activation phrases
  - verify profile

Contract characteristics from method pages:
- Required request header: `Ocp-Apim-Subscription-Key`.
- Required query parameter: `api-version=2021-09-05`.
- Audio-driven operations use `audio/wav; codecs=audio/pcm` request media type.
- Success response patterns:
  - `201 Created`: profile/enrollment create operations.
  - `204 No Content`: delete profile operations.
  - `200 OK`: get/list/reset/verify/identify/list-phrases operations.
- Error contract: `SpeakerErrorInfo` payload with `x-ms-error-code` response header.

## Stage 0: Contract Skeleton

- Add dedicated Azure Speaker Recognition router surface.
- Recognize documented route families across verification and identification planes.
- Add route-recognition tests covering representative methods in all operation groups.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate route family shape and method gating for documented operations.
- Preserve deterministic staged behavior for unknown nested routes under `/azure/speaker-recognition/*`.

Exit criteria:
- Invalid query forms return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized Speaker Recognition routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and validated by provider tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/speaker-recognition-2021-09-05`.
- Exercise representative workflows for text-dependent verification, text-independent identification, and text-independent verification.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Speaker Recognition API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `speaker-recognition-2021-09-05` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and include it in coverage reporting.
