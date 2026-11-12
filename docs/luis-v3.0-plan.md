# Azure LUIS (`luis-v3.0`) Staged Plan

## Objective

Emulate Azure LUIS REST API (`v3.0`) with deterministic local behavior for prediction requests across application slots and versions.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/luis/operation-groups?view=rest-luis-v3.0`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/luis/prediction?view=rest-luis-v3.0`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/luis/prediction/get-slot-prediction?view=rest-luis-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/luis/prediction/get-slot-prediction-get?view=rest-luis-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/luis/prediction/get-version-prediction?view=rest-luis-v3.0`
- `https://learn.microsoft.com/en-us/rest/api/luis/prediction/get-version-prediction-get?view=rest-luis-v3.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full LUIS model lifecycle/authoring capabilities in this phase.

## Route Envelope

Documented endpoint route families:
- `{Endpoint}/luis/prediction/v3.0/apps/{appId}/slots/{slotName}/predict` (`POST`)
- `{Endpoint}/luis/prediction/v3.0/apps/{appId}/slots/{slotName}/predict?query={query}` (`GET`)
- `{Endpoint}/luis/prediction/v3.0/apps/{appId}/versions/{versionId}/predict` (`POST`)
- `{Endpoint}/luis/prediction/v3.0/apps/{appId}/versions/{versionId}/predict?query={query}` (`GET`)

Stackyard emulation prefix:
- `/azure/luis/prediction/v3.0/*`

## API Surface and Contract Notes

Operation group and documented operations reviewed:
- Prediction
  - Get Slot Prediction (`POST`): predicts intents/entities from JSON body containing `query` and optional `dynamicLists`, `externalEntities`, `options`.
  - Get Slot Prediction GET (`GET`): predicts intents/entities from required `query` query parameter.
  - Get Version Prediction (`POST`): same body contract targeting specific app version.
  - Get Version Prediction GET (`GET`): same query contract targeting specific app version.

Contract characteristics from method pages:
- Required header: `Ocp-Apim-Subscription-Key`.
- Common optional query toggles: `verbose`, `show-all-intents`, `log`.
- Success response: `200 OK` with `PredictionResponse`.
- Error response contract: `Error` object (`error.code`, `error.message`) for other status codes.

## Stage 0: Contract Skeleton

- Add dedicated Azure LUIS router surface.
- Recognize documented prediction route families for slot/version + GET/POST variants.
- Add route-recognition tests covering all four documented operations.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate expected top-level route structure (`apps/{id}/slots|versions/{id}/predict`).
- Preserve deterministic staged behavior for unknown nested routes under `/azure/luis/prediction/v3.0/`.

Exit criteria:
- Invalid query forms return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized LUIS routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and validated by provider tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/luis-v3.0`.
- Exercise slot/version prediction via both `POST` and `GET` operation variants.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed LUIS API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `luis-v3.0` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and include it in coverage reporting.
