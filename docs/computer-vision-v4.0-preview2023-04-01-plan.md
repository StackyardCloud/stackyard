# Azure Computer Vision (`computer-vision-v4.0-preview2023-04-01`) Staged Plan

## Objective

Emulate Azure Computer Vision REST API (`v4.0-preview (2023-04-01)`) with deterministic local behavior for dataset management, image analysis/composition/retrieval, model lifecycle/evaluations, planogram matching, and product recognition runs.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/computervision/operation-groups?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/datasets?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-analysis?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-composition?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-retrieval?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/model-evaluations?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/models?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/planogram-compliance?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/product-recognition?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/datasets/create?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/datasets/delete?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/datasets/get?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/datasets/list?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/datasets/update?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-analysis/analyze-image?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-analysis/analyze-stream?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-analysis/segment-image?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-analysis/segment-stream?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-composition/rectify-image?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-composition/stitch-images?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-retrieval/vectorize-image?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-retrieval/vectorize-stream?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/image-retrieval/vectorize-text?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/model-evaluations/create?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/model-evaluations/delete?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/model-evaluations/get?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/model-evaluations/list?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/models/cancel?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/models/create?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/models/delete?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/models/get?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/models/list?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/planogram-compliance/match?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/product-recognition/create-run?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/product-recognition/delete-run?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/product-recognition/get-run?view=rest-computervision-v4.0-preview%20(2023-04-01)`
- `https://learn.microsoft.com/en-us/rest/api/computervision/product-recognition/list-runs?view=rest-computervision-v4.0-preview%20(2023-04-01)`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full computer vision model training/inference behavior in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/computervision/v4.0-preview/2023-04-01/datasets*`
- `/azure/computervision/v4.0-preview/2023-04-01/imageanalysis:*`
- `/azure/computervision/v4.0-preview/2023-04-01/imagecomposition:*`
- `/azure/computervision/v4.0-preview/2023-04-01/imageretrieval:*`
- `/azure/computervision/v4.0-preview/2023-04-01/modelevaluations*`
- `/azure/computervision/v4.0-preview/2023-04-01/models*`
- `/azure/computervision/v4.0-preview/2023-04-01/planogramcompliance:match`
- `/azure/computervision/v4.0-preview/2023-04-01/productrecognition/runs*`

Target API version:
- `api-version=2023-04-01-preview` (query)

## API Surface and Contract Notes

Operation groups and documented operations reviewed:

- Datasets: Create/Delete/Get/List/Update dataset resources.
- Image Analysis: Analyze (URL/stream) and Segment (URL/stream).
- Image Composition: Rectify and Stitch operations.
- Image Retrieval: Vectorize image/stream/text operations.
- Model Evaluations: Create/Delete/Get/List evaluations.
- Models: Cancel/Create/Delete/Get/List model operations.
- Planogram Compliance: Match operation.
- Product Recognition: Create/Delete/Get/List run operations.

Common error-contract characteristics from method pages:
- Validation and request-shape errors return typed service error payloads.
- Common classes include invalid request, auth/authz failures, not found, conflict, throttling, and internal service failures.

## Stage 0: Contract Skeleton

- Add dedicated Azure Computer Vision router surface.
- Recognize all operation-group envelopes listed above.
- Add route-recognition tests covering every documented method family.

Exit criteria:
- Route envelope is deterministic and contract-tested.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Validate route identifiers where required (`{name}`, `{runName}`).
- Reject unsupported method/resource combinations with deterministic behavior.

Exit criteria:
- Invalid query and unsupported combinations return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success fixtures for recognized routes.
- Preserve stable `provider/path/status` response envelope semantics used across Azure staged services.

Exit criteria:
- All recognized Computer Vision routes are deterministic and contract-tested.

## Stage 3: Example and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/computer-vision-v4.0-preview2023-04-01`.
- Exercise representative calls from each operation group.
- Update Azure contract, IO, and doc coverage scripts with service aliases and plan mapping.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifiers.
