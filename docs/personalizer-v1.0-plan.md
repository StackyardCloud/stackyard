# Azure Personalizer (`personalizer-v1.0`) Staged Plan

## Objective

Emulate Azure Personalizer REST API (`v1.0`) with deterministic local behavior for ranking and event-feedback workflows.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/personalizer/operation-groups?view=rest-personalizer-v1.0`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/personalizer/events?view=rest-personalizer-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/personalizer/rank?view=rest-personalizer-v1.0`

Method references reviewed (all linked method pages were browsed):
- `https://learn.microsoft.com/en-us/rest/api/personalizer/events/activate?view=rest-personalizer-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/personalizer/events/reward?view=rest-personalizer-v1.0`
- `https://learn.microsoft.com/en-us/rest/api/personalizer/rank/rank?view=rest-personalizer-v1.0`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full reinforcement-learning behavior in this phase.

## Route Envelope

Documented endpoint route families:
- `{Endpoint}/personalizer/v1.0/rank` (`POST`)
- `{Endpoint}/personalizer/v1.0/events/{eventId}/reward` (`POST`)
- `{Endpoint}/personalizer/v1.0/events/{eventId}/activate` (`POST`)

Stackyard emulation prefix:
- `/azure/personalizer/v1.0/*`

## API Surface and Contract Notes

Operation groups and documented operations reviewed:
- Rank
  - `Rank`: sends context/actions and receives ranked action output.
- Events
  - `Reward`: reports reward value for a previously ranked event.
  - `Activate`: confirms event was shown and reward is expected.

Contract characteristics from method pages:
- Required header: `Ocp-Apim-Subscription-Key`.
- Success responses:
  - Rank: `201 Created` with `RankResponse`.
  - Reward: `204 No Content`.
  - Activate: `204 No Content`.
- Error contract for other status codes: `ErrorResponse` containing service error details.

## Stage 0: Contract Skeleton

- Add dedicated Azure Personalizer router surface.
- Recognize all documented route families under `/azure/personalizer/v1.0/*`.
- Add route-recognition tests covering rank, reward, and activate.

Exit criteria:
- Route envelope ownership is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate route shape and method gating for documented operations.
- Preserve deterministic staged behavior for unknown nested routes under Personalizer prefix.

Exit criteria:
- Invalid query forms return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized Personalizer routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and validated by provider tests.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/personalizer-v1.0`.
- Exercise rank -> reward -> activate flow with representative payloads.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Personalizer API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `personalizer-v1.0` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifier and include it in coverage reporting.
