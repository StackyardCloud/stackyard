# Azure Content Safety (`content-safety-2024-09-01`) Staged Plan

## Objective

Emulate Azure Content Safety REST API (`2024-09-01`) with deterministic local behavior for image analysis, text analysis, and text blocklist management.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/operation-groups?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/image-analysis?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-analysis?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists?view=rest-contentsafety-2024-09-01`

Method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/image-analysis/analyze-image?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-analysis/analyze-text?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-analysis/detect-protected-material?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-analysis/shield-prompt?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/add-or-update-text-blocklist-items?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/create-or-update-text-blocklist?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/delete-text-blocklist?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/get-text-blocklist?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/get-text-blocklist-item?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/list-text-blocklist-items?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/list-text-blocklists?view=rest-contentsafety-2024-09-01`
- `https://learn.microsoft.com/en-us/rest/api/contentsafety/text-blocklists/remove-text-blocklist-items?view=rest-contentsafety-2024-09-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement model-side moderation logic in this stage.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/contentsafety/image:analyze`
- `/azure/contentsafety/text:analyze`
- `/azure/contentsafety/text:detectProtectedMaterial`
- `/azure/contentsafety/text:shieldPrompt`
- `/azure/contentsafety/text/blocklists*`

Target API version:
- `api-version=2024-09-01` (query)

## API Surface and Contract Notes

Operation groups covered:
- Image Analysis
- Text Analysis
- Text Blocklists

Common contract characteristics from method pages:
- Required auth headers include subscription-key style credentials for Azure AI endpoints.
- Success responses are primarily `200` with typed JSON bodies.
- Error responses include typed request/auth/permission/not-found/conflict/service failures.

## Stage 0: Contract Skeleton

- Add a dedicated Azure Content Safety router with route ownership for all operation groups above.
- Recognize each documented route pattern and preserve deterministic handling for unknown nested paths under `/azure/contentsafety/`.
- Add route-recognition tests for each documented method family.

Exit criteria:
- All documented route families are recognized and return deterministic provider payloads.

## Stage 1: Request Validation Foundation

- Validate malformed `api-version` query values.
- Keep deterministic envelope behavior on unknown nested routes.
- Ensure invalid requests return stable `400 InvalidRequest`.

Exit criteria:
- Input validation and negative path behavior are contract-tested.

## Stage 2: Deterministic Success Fixtures

- Return deterministic staged success fixtures for recognized operations.
- Preserve provider/path/status response envelope for contract tooling.

Exit criteria:
- Recognized operations return stable success fixtures.

## Stage 3: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/content-safety-2024-09-01`.
- Exercise representative calls across all operation groups.
- Update Azure contract, IO, and doc coverage scripts with aliases and plan mapping for `content_safety`.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve `content-safety-2024-09-01`.
