# Azure Speech to Text (`speech-to-text-v3.2-preview.2`) Staged Plan

## Objective

Emulate Azure Speech to Text REST API (`v3.2-preview.2`) with deterministic local behavior for dataset, endpoint, evaluation, model, transcription, project, webhook, and operation-tracking workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/operation-groups?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/datasets?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/endpoints?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/evaluations?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/models?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/operations?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/projects?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/service-health?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/transcriptions?view=rest-speechtotext-v3.2-preview.2`
- `https://learn.microsoft.com/en-us/rest/api/speechtotext/web-hooks?view=rest-speechtotext-v3.2-preview.2`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full speech model training/inference parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/speechtotext/v3.2-preview.2/datasets*`
- `/azure/speechtotext/v3.2-preview.2/endpoints*`
- `/azure/speechtotext/v3.2-preview.2/evaluations*`
- `/azure/speechtotext/v3.2-preview.2/models*`
- `/azure/speechtotext/v3.2-preview.2/operations*`
- `/azure/speechtotext/v3.2-preview.2/projects*`
- `/azure/speechtotext/v3.2-preview.2/healthstatus`
- `/azure/speechtotext/v3.2-preview.2/transcriptions*`
- `/azure/speechtotext/v3.2-preview.2/webhooks*`

Target API version:
- `v3.2-preview.2` in path, with optional `api-version` query gate checks when provided.

## API Surface and Contract Notes

Operation groups and documented operations reviewed:

- Datasets
  - Create/Delete/Get/List datasets.
  - List files, get file, upload block, commit blocks.
  - Copy dataset, update dataset, get supported locales, get uploads.
- Endpoints
  - Create/Delete/Get/List/Update endpoints.
  - Copy endpoint.
  - List/get/delete endpoint logs and audio files.
  - Get supported locales and base model endpoint.
- Evaluations
  - Create/Delete/Get/List/Update evaluations.
  - List/get evaluation files.
  - Get supported locales.
- Models
  - Create/Delete/Get/List/Update models.
  - Copy model and authorize model copy.
  - List/get model files and manifest.
  - Get supported locales.
  - List/get model SKUs.
  - List base models and get base model.
- Operations
  - Get operation status.
- Projects
  - Create/Delete/Get/List/Update projects.
  - Copy project.
  - List project datasets/endpoints/evaluations/models.
  - Get supported locales.
- Service Health
  - Get health status.
- Transcriptions
  - Create/Delete/Get/List/Update transcriptions.
  - List/get transcription files.
  - Get supported locales.
- Webhooks
  - Create/Delete/Get/List/Update webhooks.
  - Ping and test webhooks.

Common error contract characteristics:
- Validation failures return a structured service error payload.
- Common classes include invalid request, auth/authz failures, not found, conflict, throttling, and server failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure Speech to Text router surface.
- Recognize all operation-group envelopes under `/azure/speechtotext/v3.2-preview.2/*`.
- Add route-recognition tests that cover representative methods across every operation group.

Exit criteria:
- Route envelope is locked by tests and routed requests are deterministic.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when present.
- Validate known top-level resources and reject unknown roots.
- Enforce baseline method gating for operation-only and health-status endpoints.

Exit criteria:
- Invalid query/method combinations return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic success fixtures for recognized Speech to Text routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/ai-services/speech-to-text-v3.2-preview.2`.
- Exercise representative calls from multiple operation groups.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally and demonstrates routed Speech to Text API calls.

## Stage 4: Coverage Wiring

- Add service aliases to Azure contract and IO coverage scripts.
- Add plan-doc mapping for doc coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve the new service identifiers.
