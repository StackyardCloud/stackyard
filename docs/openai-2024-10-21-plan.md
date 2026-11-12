# Azure OpenAI (`openai-2024-10-21`) Staged Plan

## Objective

Emulate Azure OpenAI REST API (`2024-10-21`) with deterministic local behavior for batch jobs, file management, fine-tuning workflows, model discovery, and upload lifecycle operations.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/operation-groups?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/batch?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/models?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/upload-file?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/batch/create?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/batch/cancel?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/batch/get?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/batch/list?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files/upload?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files/list?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files/get?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files/delete?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files/download?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/files/import?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/create?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/list?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/get?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/cancel?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/list-checkpoints?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/list-events?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/fine-tuning/delete?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/models/list?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/models/get?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/upload-file/add-upload-part?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/upload-file/complete-upload?view=rest-azureopenai-2024-10-21`
- `https://learn.microsoft.com/en-us/rest/api/azureopenai/upload-file/cancel-upload?view=rest-azureopenai-2024-10-21`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full model inference behavior or token-level semantics in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/openai/batches`
- `/azure/openai/batches/{batch_id}`
- `/azure/openai/batches/{batch_id}/cancel`
- `/azure/openai/files`
- `/azure/openai/files/import`
- `/azure/openai/files/{file_id}`
- `/azure/openai/files/{file_id}/content`
- `/azure/openai/fine_tuning/jobs`
- `/azure/openai/fine_tuning/jobs/{fine_tuning_job_id}`
- `/azure/openai/fine_tuning/jobs/{fine_tuning_job_id}/cancel`
- `/azure/openai/fine_tuning/jobs/{fine_tuning_job_id}/checkpoints`
- `/azure/openai/fine_tuning/jobs/{fine_tuning_job_id}/events`
- `/azure/openai/models`
- `/azure/openai/models/{model}`
- `/azure/openai/uploads/{upload_id}/parts`
- `/azure/openai/uploads/{upload_id}/complete`
- `/azure/openai/uploads/{upload_id}/cancel`

Target API version:
- `api-version=2024-10-21`

## API Surface and Contract Notes

Operation groups and method pages reviewed:

- Batch:
  - Create/List/Get/Cancel batch operations.
- Files:
  - Upload/List/Get/Delete/Download/Import file operations.
- Fine-Tuning:
  - Create/List/Get/Cancel/Delete fine-tuning job operations plus list checkpoints/events.
- Models:
  - List and get model metadata.
- Upload File:
  - Add upload part, complete upload, cancel upload.

Common contract characteristics from method pages:
- `Ocp-Apim-Subscription-Key` is required for requests.
- Common success classes are `200`, `201`, `202`, and `204` depending on operation.
- Error responses use typed error envelopes (`error`, `code`, `message`, details) with common classes for invalid request, auth/authz, not found, conflict, and service-side failure.

## Stage 0: Contract Skeleton

- Add dedicated Azure OpenAI router surface.
- Recognize all operation-group routes listed above.
- Add route-recognition tests across all method families.

Exit criteria:
- Route envelope and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Validate route IDs for nested resources (batch/file/job/upload/model IDs).
- Reject unsupported combinations with deterministic staged behavior.

Exit criteria:
- Invalid query/method combinations return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized routes.
- Preserve stable `provider/path/status` envelope semantics used by existing Azure staged services.

Exit criteria:
- OpenAI data-plane route handling is deterministic and contract-tested.

## Stage 3: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/openai-2024-10-21`.
- Exercise representative calls from every operation group.
- Update Azure contract, IO, and doc coverage scripts with service aliases and plan mapping.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
