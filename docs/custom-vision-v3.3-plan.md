# Azure Custom Vision (`custom-vision-v3.3`) Staged Plan

## Objective

Emulate Azure Custom Vision REST API (`v3.3`) with deterministic local behavior for domain discovery, project lifecycle, image/tag operations, iteration lifecycle, and prediction-query management.

Primary reference:
- `https://learn.microsoft.com/en-us/rest/api/customvision/operation-groups?view=rest-customvision-v3.3`

Operation-group references:
- `https://learn.microsoft.com/en-us/rest/api/customvision/domains?view=rest-customvision-v3.3`
- `https://learn.microsoft.com/en-us/rest/api/customvision/images?view=rest-customvision-v3.3`
- `https://learn.microsoft.com/en-us/rest/api/customvision/iterations?view=rest-customvision-v3.3`
- `https://learn.microsoft.com/en-us/rest/api/customvision/predictions?view=rest-customvision-v3.3`
- `https://learn.microsoft.com/en-us/rest/api/customvision/projects?view=rest-customvision-v3.3`

Method references reviewed:
- Domains: `get`, `list`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/domains/get?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/domains/list?view=rest-customvision-v3.3`
- Images: `create-from-data`, `create-from-files`, `create-from-predictions`, `create-from-urls`, `create-regions`, `create-tags`, `delete-regions`, `delete-tags`, `delete`, `get-count`, `get-region-proposals`, `get-suggested-count`, `get-tagged-count`, `get-untagged-count`, `list-by-ids`, `list-suggested`, `list-tagged`, `list-untagged`, `list`, `update-metadata`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/create-from-data?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/create-from-files?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/create-from-predictions?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/create-from-urls?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/create-regions?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/create-tags?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/delete-regions?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/delete-tags?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/delete?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/get-count?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/get-region-proposals?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/get-suggested-count?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/get-tagged-count?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/get-untagged-count?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/list-by-ids?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/list-suggested?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/list-tagged?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/list-untagged?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/list?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/images/update-metadata?view=rest-customvision-v3.3`
- Iterations: `delete`, `export`, `get-performance-image-count`, `get-performance`, `get`, `list-exports`, `list-performance-images`, `list`, `publish`, `unpublish`, `update`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/delete?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/export?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/get-performance-image-count?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/get-performance?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/get?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/list-exports?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/list-performance-images?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/list?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/publish?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/unpublish?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/iterations/update?view=rest-customvision-v3.3`
- Predictions: `delete`, `query`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/predictions/delete?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/predictions/query?view=rest-customvision-v3.3`
- Projects: `create`, `delete`, `export`, `get-artifact`, `get`, `import`, `list`, `suggest-tags-and-regions`, `train`, `update`, `create-tag`, `delete-tag`, `get-tag`, `list-tags`, `update-tag`, `quick-test-image`, `quick-test-image-url`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/create?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/delete?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/export?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/get-artifact?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/get?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/import?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/list?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/suggest-tags-and-regions?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/train?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/update?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/create-tag?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/delete-tag?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/get-tag?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/list-tags?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/update-tag?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/quick-test-image?view=rest-customvision-v3.3`
  - `https://learn.microsoft.com/en-us/rest/api/customvision/projects/quick-test-image-url?view=rest-customvision-v3.3`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full model-training and prediction semantics in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/customvision/v3.3/training/domains*`
- `/azure/customvision/v3.3/training/projects*`

Target API version:
- `api-version=3.3` (query)

## API Surface and Contract Notes

Operation groups and contract expectations:
- Domains: discovery/list operations for domain types.
- Projects: project create/import/get/list/update/delete; train/export; quick test; tag lifecycle.
- Images: image ingest/query/list/count and region/tag mutation endpoints.
- Iterations: iteration list/get/update/delete/export/publish/unpublish and performance endpoints.
- Predictions: prediction result query and delete.

Common error contract characteristics:
- Validation failures produce structured service error payloads.
- Common error classes include invalid request, authentication/authorization failures, not found, conflict, throttling, and internal errors.

## Stage 0: Contract Skeleton

- Add a dedicated Azure Custom Vision router for `/azure/customvision/v3.3/*`.
- Recognize `training/domains*` and `training/projects*` route families.
- Add route-recognition tests covering representative methods across all operation groups.

Exit criteria:
- Route envelope is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape when provided.
- Validate minimum route shape for domains/projects roots.
- Enforce baseline method gating for project and domain route families.

Exit criteria:
- Invalid API-version inputs and unsupported method/root combinations return deterministic validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized Custom Vision routes.
- Preserve stable `provider/path/status` payload semantics used by existing Azure staged services.

Exit criteria:
- Recognized routes return deterministic contracts in provider tests.

## Stage 3: Example Coverage

- Add an Azure Go SDK style example in `examples/azure/ai-services/custom-vision-v3.3`.
- Exercise representative domains/projects/images/iterations/predictions calls against the staged endpoint.
- Keep example compatible with staged/foundation responses.

Exit criteria:
- Example compiles/runs locally against Stackyard and demonstrates routed Custom Vision API calls.

## Stage 4: Coverage Wiring

- Add service aliases for `custom-vision-v3.3` to Azure contract and IO coverage scripts.
- Add plan-doc mapping to Azure doc-contract coverage script.

Exit criteria:
- `azure-contract`, `azure-io-contract`, and `azure-doc-contract` scripts resolve `custom-vision-v3.3` and evaluate strict signals.
