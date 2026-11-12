# Azure Face (`face-v1.2`) Staged Plan

## Objective

Emulate Azure Face REST API (`v1.2`) with deterministic local behavior for face detection, face/large-face lists, recognition operations, liveness sessions, and person-group resources.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/face/operation-groups?view=rest-face-v1.2`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/face/face-detection-operations?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations?view=rest-face-v1.2`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/face/face-detection-operations/detect-from-session-image-id?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-detection-operations/detect-from-url?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-detection-operations/detect?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/add-face-list-face-from-url?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/add-face-list-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/add-large-face-list-face-from-url?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/add-large-face-list-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/create-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/create-large-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/delete-face-list-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/delete-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/delete-large-face-list-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/delete-large-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-face-lists?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-large-face-list-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-large-face-list-faces?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-large-face-list-training-status?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-large-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/get-large-face-lists?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/train-large-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/update-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/update-large-face-list-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-list-operations/update-large-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/find-similar-from-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/find-similar-from-large-face-list?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/find-similar?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/identify-from-dynamic-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/identify-from-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/identify-from-person-directory?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/identify-from-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/verify-face-to-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/verify-from-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/verify-from-person-directory?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/face-recognition-operations/verify-from-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/create-liveness-session?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/create-liveness-with-verify-session?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/delete-liveness-session?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/delete-liveness-with-verify-session?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/get-liveness-session-result?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/get-liveness-with-verify-session-result?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/liveness-session-operations/get-session-image?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/add-large-person-group-person-face-from-url?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/add-large-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/add-person-group-person-face-from-url?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/add-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/create-large-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/create-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/create-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/create-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/delete-large-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/delete-large-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/delete-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/delete-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/delete-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/delete-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-large-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-large-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-large-person-group-persons?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-large-person-group-training-status?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-large-person-groups?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-person-group-persons?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-person-group-training-status?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/get-person-groups?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/train-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/train-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/update-large-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/update-large-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/update-large-person-group?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/update-person-group-person-face?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/update-person-group-person?view=rest-face-v1.2`
- `https://learn.microsoft.com/en-us/rest/api/face/person-group-operations/update-person-group?view=rest-face-v1.2`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full face-recognition model semantics in this phase.

## Route Envelope

Documented endpoint route families:
- `{endpoint}/face/{apiVersion}/detect`
- `{endpoint}/face/{apiVersion}/facelists*`
- `{endpoint}/face/{apiVersion}/largefacelists*`
- `{endpoint}/face/{apiVersion}/findsimilars`
- `{endpoint}/face/{apiVersion}/group`
- `{endpoint}/face/{apiVersion}/identify`
- `{endpoint}/face/{apiVersion}/verify`
- `{endpoint}/face/{apiVersion}/detectLiveness-sessions*`
- `{endpoint}/face/{apiVersion}/detectLivenessWithVerify-sessions*`
- `{endpoint}/face/{apiVersion}/sessionImages/{sessionImageId}`
- `{endpoint}/face/{apiVersion}/persongroups*`
- `{endpoint}/face/{apiVersion}/largepersongroups*`

Stackyard emulation prefix:
- `/azure/face/v1.2/*`

## API Surface and Contract Notes

Operation groups and documented operations reviewed:
- Face detection: detect from binary input, URL, and liveness session image.
- Face list operations: create/update/delete/get/list for face lists and large face lists, add/update/delete/get persisted faces, train large face list, get training status.
- Recognition operations: find similar, group, identify, verify across supported target resources.
- Liveness sessions: create/get/delete liveness and liveness-with-verify sessions, retrieve session images.
- Person group operations: create/update/delete/get/list for person groups and large person groups, person lifecycle, persisted-face lifecycle, and train/training status operations.

Common error-contract characteristics from method pages:
- Validation failures return structured service error payloads.
- Common classes include invalid request, auth/authz failures, not found, conflict, throttling, and internal service failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure Face router surface.
- Recognize documented route families listed above.
- Add route-recognition tests covering representative operations across every operation group.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate top-level route/resource shapes and action-route suffixes.
- Enforce baseline method gating for detection, recognition, liveness, and list/group resources.

Exit criteria:
- Invalid query/method combinations return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and validated by provider tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example in `examples/azure/ai-services/face-v1.2`.
- Exercise representative calls across all operation groups.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Face API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `face-v1.2` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and pass strict signals.
