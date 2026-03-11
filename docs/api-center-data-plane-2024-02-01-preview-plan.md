# Azure API Center Data Plane (`data-plane-2024-02-01-preview`) Staged Plan

## Objective

Emulate Azure API Center data-plane APIs (`2024-02-01-preview`) with deterministic local behavior for API catalog listing/reads, versioning, definitions, deployments, environments, and definition export operation-status workflows.

Primary references:
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/operation-groups?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-definitions?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-deployments?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-versions?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/apis?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/environments?view=rest-dataplane-apicenter-2024-02-01-preview`

Method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-definitions/export-specification?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-definitions/get-definition?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-definitions/get-export-specification-operation-status?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-definitions/list-definitions?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-deployments/get-deployment?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-deployments/list-deployments?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-versions/get-version?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/api-versions/list-versions?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/apis/get?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/apis/list?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/apis/list-all?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/environments/get?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/environments/list?view=rest-dataplane-apicenter-2024-02-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/dataplane/apicenter/environments/list-all?view=rest-dataplane-apicenter-2024-02-01-preview`

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not require live Azure resources for local examples.
- Do not emulate full API Center governance lifecycle, authoring, or permission systems in this phase.

## Route Envelope

Stackyard route envelope for this service:
- `/azure/apicenter/workspaces/{workspaceName}/apis/*`
- `/azure/apicenter/workspaces/{workspaceName}/environments/*`
- `/azure/apicenter/apis`
- `/azure/apicenter/environments`

Target API version:
- `api-version=2024-02-01-preview`

## API Surface and Contract Notes

Operation groups reviewed:
- 5 operation-group pages (`api-definitions`, `api-deployments`, `api-versions`, `apis`, `environments`).

Method pages reviewed:
- 14 method pages linked from those operation groups.

Common contract characteristics from method pages:
- Data-plane endpoint host pattern: `{serviceName}.data.{region}.azure-apicenter.ms`.
- Workspace-scoped and global list endpoints.
- Success status families include `200` and `202` (export specification).
- Non-success responses use documented error schemas in `Other Status Codes`.

## Stage 0: Contract Skeleton

- Add dedicated Azure API Center data-plane router surface.
- Recognize API Center data-plane path envelope for workspaces, APIs, and environments.
- Add route-recognition tests spanning all operation groups.

Exit criteria:
- Route ownership and deterministic staged behavior are locked by tests.

## Stage 1: Request Validation Foundation

- Validate `api-version` query shape.
- Enforce baseline method gating for API Center data-plane routes.
- Return deterministic invalid-request payloads for malformed API version input.

Exit criteria:
- Invalid query/method patterns return stable validation contracts.

## Stage 2: Deterministic Foundation Fixtures

- Return deterministic staged success payloads for recognized API Center routes.
- Preserve stable response shape (`provider`, `path`, `status`) used by Azure staged services.

Exit criteria:
- Recognized routes are deterministic and contract-tested.

## Stage 3: Example Coverage

- Add Azure Go SDK style example under `examples/azure/api-center/data-plane-2024-02-01-preview`.
- Exercise representative workflows for APIs, versions, definitions, deployments, environments, and export operation status.

Exit criteria:
- Example compiles and runs against Stackyard in staged mode.

## Stage 4: Coverage Wiring

- Add API Center aliases to Azure contract, IO-contract, and doc-contract coverage scripts.
- Map this plan doc in doc-contract coverage lookup for the `api_center_data_plane` service key.

Exit criteria:
- Coverage tooling resolves `api_center_data_plane` and versioned aliases consistently.
