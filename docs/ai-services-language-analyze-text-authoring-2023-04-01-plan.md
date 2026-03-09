# Azure AI Services - Language - Analyze Text Authoring (`ai-services-language-analyze-text-authoring-2023-04-01`) Staged Plan

## Objective

Emulate Azure AI Services Language Analyze Text Authoring (`2023-04-01`) with deterministic local behavior for project authoring, training, deployment, trained-model, and language-resource workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/operation-groups?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/text-authoring-project?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/text-authoring-trained-model?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/text-authoring-deployment?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/project-training-jobs?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/project-training-config?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/language-resource-group?view=rest-language-analyze-text-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/analyze-text-authoring/get-supported-languages?view=rest-language-analyze-text-authoring-2023-04-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full portal parity for labeling UX and runtime intelligence quality in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/language/authoring/analyze-text/*`

Target API version:
- `api-version=2023-04-01`

## API Surface and Contract Notes

Operation groups and methods documented in `2023-04-01`:

- Get Supported Languages
  - `Get Supported Languages`
- Text Authoring Project
  - `Create Project`
  - `Delete Project`
  - `Export`
  - `Get Project`
  - `Import`
  - `List Projects`
  - `List Projects Metadata`
  - `Update Metadata`
- Project Training Config
  - `Export Project`
  - `Import Project`
- Project Training Jobs
  - `Train`
  - `Cancel Training Job`
- Text Authoring Deployment
  - `Assign Deployment Resources`
  - `Delete Deployment`
  - `Deploy Model`
  - `Get Deployment`
  - `Swap Deployments`
- Text Authoring Trained Model
  - `Delete Trained Model`
  - `Get Trained Model`
  - `List Trained Models`
  - `Load Snapshot`
  - `Validate Training Data`
- Language Resource Group
  - `Delete Text Authoring Resource`
  - `Get Resource State`
  - `List Text Authoring Resources`
  - `New Text Authoring Resource`
  - `Unassign Deployment Resources`
  - `Unassign Project Resources`
  - `Update Text Authoring Resource`

Contract characteristics captured from operation pages:
- Surface is project-centric and uses authoring routes beneath `/language/authoring/analyze-text`.
- Includes synchronous CRUD responses (`200/201/204`) and long-running-style training/deployment paths with `202` acceptance semantics.
- Error payloads use service error envelopes with model-state, validation, authorization, not-found, conflict, throttling, and server-failure classes.

## Stage 0: Contract Skeleton

- Add dedicated Azure language analyze-text-authoring router surface.
- Recognize authoring route envelope under `/azure/language/authoring/analyze-text/*`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests covering representative methods from each operation group.

Exit criteria:
- Authoring route envelope and fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate required path identifiers (`projectName`, `deploymentName`, `modelLabel`, `jobId`, `resourceName`) where applicable.
- Validate core request envelopes for project create/import/export, train, deploy, metadata update, and resource assignment.

Exit criteria:
- Invalid requests return deterministic `400` envelopes with stable service-style error contracts.

## Stage 2: Authoring Resource and Project Fixtures

- Implement deterministic fixtures for language resources and project lifecycle:
  - create/get/list/update/delete project and metadata.
  - create/get/list/update/delete text authoring resources.
  - supported languages response fixture.

Exit criteria:
- Project/resource CRUD shapes are deterministic and contract-tested.

## Stage 3: Training, Model, and Deployment Fixtures

- Implement deterministic training job submit/cancel lifecycle.
- Implement trained-model list/get/delete/load-snapshot/validate flows.
- Implement deployment create/get/swap/delete and assignment/unassignment semantics.

Exit criteria:
- Model/deployment/training flows are deterministic and contract-tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/language-analyze-text-authoring-2023-04-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
