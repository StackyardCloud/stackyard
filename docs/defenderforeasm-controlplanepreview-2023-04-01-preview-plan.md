# Azure Defender EASM Control Plane Preview (`defenderforeasm_controlplanepreview_2023_04_01_preview`) Staged Plan

## Objective

Emulate Azure Defender EASM control-plane REST API (`api-version=2023-04-01-preview`) with deterministic local behavior for:

- label lifecycle under workspaces
- workspace lifecycle under subscriptions/resource groups
- workspace task retrieval
- provider operations discovery
- an Azure Go SDK example under `examples/azure/defenderforeasm-controlplanepreview-2023-04-01-preview`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/controlplanepreview/operation-groups?view=rest-defenderforeasm-controlplanepreview-2023-04-01-preview`

Service overview page (used to confirm control-plane groups):

- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/controlplanepreview/labels?view=rest-defenderforeasm-controlplanepreview-2023-04-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/controlplanepreview/operations?view=rest-defenderforeasm-controlplanepreview-2023-04-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/controlplanepreview/workspaces?view=rest-defenderforeasm-controlplanepreview-2023-04-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/controlplanepreview/tasks?view=rest-defenderforeasm-controlplanepreview-2023-04-01-preview`

Method pages reviewed directly:

- Labels:
  - `labels/create-and-update`
  - `labels/delete`
  - `labels/get-by-workspace`
  - `labels/list-by-workspace`
  - `labels/update`
- Operations:
  - `operations/list`
- Workspaces:
  - `workspaces/create-and-update`
  - `workspaces/delete`
  - `workspaces/get`
  - `workspaces/list-by-resource-group`
  - `workspaces/list-by-subscription`
  - `workspaces/update`
- Tasks:
  - `tasks/get-by-workspace`

Notes on source behavior in this environment:

- pages intermittently include authorization wrappers
- route, request body, response codes, and error-shape sections remain readable
- all operation pages above were opened and verified

## Surface Summary

Reviewed operation groups and methods:

- Labels (5):
  - `Create And Update`, `Delete`, `Get By Workspace`, `List By Workspace`, `Update`
- Operations (1):
  - `List`
- Workspaces (6):
  - `Create And Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Tasks (1):
  - `Get By Workspace`

Total reviewed methods: `13`.

## Service Naming

Canonical Stackyard naming:

- service key: `defenderforeasm_controlplanepreview_2023_04_01_preview`
- display name: `Azure Defender EASM (Control Plane Preview)`
- API version: `2023-04-01-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_defenderforeasm_controlplanepreview_2023_04_01_preview.go`
- tests: `internal/server/provider_azure_defenderforeasm_controlplanepreview_2023_04_01_preview_test.go`
- example: `examples/azure/defenderforeasm-controlplanepreview-2023-04-01-preview`
- plan doc: `docs/defenderforeasm-controlplanepreview-2023-04-01-preview-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Easm/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Easm/workspaces`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Easm/workspaces`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Easm/workspaces/{workspaceName}/labels`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Easm/workspaces/{workspaceName}/tasks/{taskId}`

Representative routes:

- `GET /azure/providers/Microsoft.Easm/operations?api-version=2023-04-01-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}?api-version=2023-04-01-preview`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}?api-version=2023-04-01-preview`
- `DELETE /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}?api-version=2023-04-01-preview`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}/labels/{labelName}?api-version=2023-04-01-preview`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}/labels/{labelName}?api-version=2023-04-01-preview`
- `DELETE /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}/labels/{labelName}?api-version=2023-04-01-preview`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Easm/workspaces/{workspaceName}/tasks/{taskId}?api-version=2023-04-01-preview`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all methods use ARM OAuth2 (`azure_auth` with `user_impersonation`)
- common error envelope is `ErrorResponse` with nested ARM error detail
- list responses use `value` and optional `nextLink`
- ARM resource envelopes include `id`, `name`, `type`, `systemData`, and `properties`

Observed response patterns:

- create/update operations return `200` and/or `201`
- delete operations return `202` or `204` (idempotent/async delete semantics)
- read/list operations return `200`
- all methods define `Other Status Codes` with `Error Response`

Notable request specifics:

- `workspaces/create-and-update` requires `location` in request body
- `workspaces/update` uses patch body focused on mutable fields (`tags`)
- label write methods accept `properties.color` and `properties.displayName`
- task retrieval requires a concrete `{taskId}` under workspace scope

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `defenderforeasm_controlplanepreview_2023_04_01_preview` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches EASM paths to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Microsoft.Easm/` path families
- strict `api-version=2023-04-01-preview` validation
- deterministic `400` for missing/invalid API version on owned routes
- deterministic `405` for unsupported methods

Exit criteria:

- EASM route families are deterministically owned
- version/method failures are stable and testable

### Stage 2: Workspaces + Provider Operations

Implement:

- Workspaces: `Create And Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Operations: `List`

Behavior:

- deterministic workspace storage keyed by subscription/resourceGroup/workspaceName
- stable list ordering and pagination stubs (`nextLink` optional)
- `DELETE` returns `202` when deleting an existing workspace and `204` when already absent

Exit criteria:

- workspace lifecycle and provider operation discovery are fully routed and deterministic

### Stage 3: Labels + Tasks

Implement:

- Labels: all 5 methods
- Tasks: `Get By Workspace`

Behavior:

- label resources scoped beneath workspace identity
- label PUT/PATCH merge behavior for properties (`color`, `displayName`)
- task fixture readback with deterministic `200` for known fixture IDs and ARM `404` for unknown IDs

Exit criteria:

- workspace-child resources (labels/tasks) are operational with stable shapes and errors

### Stage 4: Tests and Contract Hardening

Add `provider_azure_defenderforeasm_controlplanepreview_2023_04_01_preview_test.go` to cover:

- API-version validation failures
- method-not-allowed and path-not-found behavior
- workspace CRUD/list behavior and delete idempotency (`202` vs `204`)
- label CRUD/list behavior scoped to workspace
- operations list shape
- task get success and not-found
- ARM error envelope shape assertions

Exit criteria:

- targeted server tests pass
- no EASM route falls through as unowned

### Stage 5: Example and Coverage Enforcement

- ensure example exercises all 4 operation groups and all 13 methods
- verify coverage scripts recognize aliases and plan-doc mapping
- keep stage progression explicit by allowing strict coverage failures until implementation/tests are added

Exit criteria:

- example compiles and runs against emulator
- coverage scripts discover service key and doc mapping

## Coverage Script Updates

Add aliases for `defenderforeasm_controlplanepreview_2023_04_01_preview` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Minimum alias set:

- `defenderforeasm_controlplanepreview_2023_04_01_preview`
- `defenderforeasm-controlplanepreview-2023-04-01-preview`
- `defenderforeasm`
- `defenderforeasm-controlplanepreview`
- `defender_easm`
- `defender-easm`
- `azure_defenderforeasm`
- `microsoft_easm`
- `rest_defenderforeasm_controlplanepreview_2023_04_01_preview`

Plan-doc mapping:

- `SERVICE_PLAN_DOCS["defenderforeasm_controlplanepreview_2023_04_01_preview"] = ["defenderforeasm-controlplanepreview-2023-04-01-preview-plan.md"]`

## Example Plan

Add `examples/azure/defenderforeasm-controlplanepreview-2023-04-01-preview`:

- `main.go`: Azure Go SDK (`azcore/runtime`) based client exercising all 13 methods
- `Dockerfile`: compiles the example binary
- `docker-compose.yml`: starts Stackyard + the example with required env vars

Example operation coverage target:

- Operations: 1/1
- Workspaces: 6/6
- Labels: 5/5
- Tasks: 1/1

## Validation Commands

- `go test ./internal/server -run 'TestAzureDefenderForEasmControlPlanePreview20230401' -count=1`
- `go build ./examples/azure/defenderforeasm-controlplanepreview-2023-04-01-preview`
- `python3 scripts/azure-contract-coverage.py --service defenderforeasm --require-service defenderforeasm_controlplanepreview_2023_04_01_preview --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service defenderforeasm --require-service defenderforeasm_controlplanepreview_2023_04_01_preview --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service defenderforeasm --require-service defenderforeasm_controlplanepreview_2023_04_01_preview --fail-on any`
