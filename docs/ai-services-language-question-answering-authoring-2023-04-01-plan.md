# Azure AI Services - Language - Question Answering Authoring (`ai-services-language-question-answering-authoring-2023-04-01`) Staged Plan

## Objective

Emulate Azure AI Services Language Question Answering Authoring (`2023-04-01`) with deterministic local behavior for project authoring, import/export, deployment, and knowledge-base asset management workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/language/question-answering-authoring/operation-groups?view=rest-language-question-answering-authoring-2023-04-01`
- `https://learn.microsoft.com/en-us/rest/api/language/question-answering-authoring/question-answering-projects?view=rest-language-question-answering-authoring-2023-04-01`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not implement full ranking-model inference parity in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/language/authoring/query-knowledgebases/*`

Target API version:
- `api-version=2023-04-01`

## API Surface and Contract Notes

Question Answering Authoring operations documented for this API version:

- `POST /language/authoring/query-knowledgebases/projects/{projectName}/feedback`
- `PATCH /language/authoring/query-knowledgebases/projects/{projectName}`
- `DELETE /language/authoring/query-knowledgebases/projects/{projectName}`
- `PUT /language/authoring/query-knowledgebases/projects/{projectName}/deployments/{deploymentName}`
- `POST /language/authoring/query-knowledgebases/projects/{projectName}/:export`
- `GET /language/authoring/query-knowledgebases/projects/deletion-jobs/{jobId}`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/deployments/{deploymentName}/jobs/{jobId}`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/export/jobs/{jobId}`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/import/jobs/{jobId}`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/qnas`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/sources`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/synonyms`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/qnas/jobs/{jobId}`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/sources/jobs/{jobId}`
- `POST /language/authoring/query-knowledgebases/projects/{projectName}/:import`
- `GET /language/authoring/query-knowledgebases/projects/{projectName}/deployments`
- `GET /language/authoring/query-knowledgebases/projects`
- `PATCH /language/authoring/query-knowledgebases/projects/{projectName}/qnas`
- `PATCH /language/authoring/query-knowledgebases/projects/{projectName}/sources`
- `PUT /language/authoring/query-knowledgebases/projects/{projectName}/synonyms`

Common contract characteristics from operation pages:
- Mixed synchronous and long-running style operations (`200/201/204` and `202` acceptance semantics with status polling endpoints).
- Project-centric path hierarchy with dedicated job-status endpoints for delete, import, export, deployment, and update workflows.
- Error envelopes include authorization, validation, not-found, conflict, throttling, and internal-service failure classes.

## Stage 0: Contract Skeleton

- Add dedicated Azure question-answering-authoring router surface.
- Recognize all request paths under `/azure/language/authoring/query-knowledgebases/*`.
- Return deterministic `501 NotImplemented` for recognized-but-unimplemented requests.
- Add route-recognition tests with representative requests across all documented operation groups.

Exit criteria:
- Route envelope and staged fallback behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version`.
- Validate required route identifiers (`projectName`, `deploymentName`, `jobId`).
- Validate minimal payload envelopes for feedback, project patch/create, import/export, deploy, and bulk update operations.

Exit criteria:
- Invalid requests return deterministic `400` service-style error envelopes.

## Stage 2: Project and Asset Fixtures

- Implement deterministic project lifecycle fixtures:
  - create/get/list/delete project.
  - list deployments and get project details.
- Implement deterministic retrieval fixtures:
  - qnas, sources, synonyms.

Exit criteria:
- Core project and asset read/write shapes are deterministic and contract-tested.

## Stage 3: Job Lifecycle Fixtures

- Implement deterministic async lifecycle fixtures for:
  - project deletion jobs.
  - import/export jobs.
  - deployment jobs.
  - qnas/sources update jobs.
- Implement deterministic deployment and feedback behavior.

Exit criteria:
- Job submission/status flows are deterministic and contract-tested.

## Stage 4: Examples and Coverage Wiring

- Add Azure Go SDK style example in `examples/azure/ai-services/language-question-answering-authoring-2023-04-01`.
- Update Azure coverage alias scripts for this service naming variant.

Exit criteria:
- Example compiles/runs in staged mode and coverage scripts resolve the service identifier.
