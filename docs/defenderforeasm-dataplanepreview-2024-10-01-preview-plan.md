# Azure Defender EASM Data Plane Preview (`defenderforeasm_dataplanepreview_2024_10_01_preview`) Staged Plan

## Objective

Emulate the Azure Defender EASM data-plane REST API (`api-version=2024-10-01-preview`) with deterministic local behavior for:

- assets inventory, export, delta, observation, and bulk-update operations
- CISA CVE lookup/list operations
- data-connection create/get/list/delete/validate operations
- discovery-group lifecycle, run history, validation, and asset-chain actions
- discovery-template get/list operations
- policy create/list operations
- reporting summary/snapshot/export and billable-summary operations
- saved-filter lifecycle operations
- task list/get/run/cancel/download operations
- an Azure Go SDK example under `examples/azure/defenderforeasm-dataplanepreview-2024-10-01-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- This is a service data-plane API rooted at an EASM workspace endpoint (`{region}.easm.defender.microsoft.com/.../workspaces/{workspaceName}`).
- It is not the ARM control-plane `Microsoft.Easm` provider already represented by `defenderforeasm_controlplanepreview_2023_04_01_preview`.

## Primary References Reviewed

Requested operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/operation-groups?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/assets?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/cisa-cves?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/data-connections?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/discovery-groups?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/discovery-templates?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/policies?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/reports?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/saved-filters?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/defenderforeasm/dataplanepreview/tasks?view=rest-defenderforeasm-dataplanepreview-2024-10-01-preview`

Method pages reviewed directly (all linked methods):

- Assets:
  - `assets/get-asset-resource`
  - `assets/get-assets-export`
  - `assets/get-delta-details`
  - `assets/get-delta-summary`
  - `assets/get-observations`
  - `assets/list-asset-resource`
  - `assets/update-assets`
- Cisa Cves:
  - `cisa-cves/get-cisa-cve`
  - `cisa-cves/get-cisa-cves`
- Data Connections:
  - `data-connections/create-or-replace-data-connection`
  - `data-connections/delete-data-connection`
  - `data-connections/get-data-connection`
  - `data-connections/list-data-connection`
  - `data-connections/validate-data-connection`
- Discovery Groups:
  - `discovery-groups/create-or-replace-disco-group`
  - `discovery-groups/delete-disco-group`
  - `discovery-groups/dismiss-asset-chain`
  - `discovery-groups/get-asset-chain-summary`
  - `discovery-groups/get-disco-group`
  - `discovery-groups/list-disco-group`
  - `discovery-groups/list-runs`
  - `discovery-groups/run-disco-group`
  - `discovery-groups/validate-disco-group`
- Discovery Templates:
  - `discovery-templates/get-disco-template`
  - `discovery-templates/list-disco-template`
- Policies:
  - `policies/create-or-replace-policy`
  - `policies/list-policy`
- Reports:
  - `reports/get-billable`
  - `reports/get-snapshot`
  - `reports/get-snapshot-export`
  - `reports/get-summary`
- Saved Filters:
  - `saved-filters/create-or-replace-saved-filter`
  - `saved-filters/delete-saved-filter`
  - `saved-filters/get-saved-filter`
  - `saved-filters/list-saved-filter`
- Tasks:
  - `tasks/cancel-task`
  - `tasks/download-task`
  - `tasks/get-task`
  - `tasks/list-task`
  - `tasks/run-task`

## Surface Summary

Operation groups and methods reviewed:

- Assets (7):
  - `Get Asset Resource`, `Get Assets Export`, `Get Delta Details`, `Get Delta Summary`, `Get Observations`, `List Asset Resource`, `Update Assets`
- Cisa Cves (2):
  - `Get Cisa Cve`, `Get Cisa Cves`
- Data Connections (5):
  - `Create Or Replace Data Connection`, `Delete Data Connection`, `Get Data Connection`, `List Data Connection`, `Validate Data Connection`
- Discovery Groups (9):
  - `Create Or Replace Disco Group`, `Delete Disco Group`, `Dismiss Asset Chain`, `Get Asset Chain Summary`, `Get Disco Group`, `List Disco Group`, `List Runs`, `Run Disco Group`, `Validate Disco Group`
- Discovery Templates (2):
  - `Get Disco Template`, `List Disco Template`
- Policies (2):
  - `Create Or Replace Policy`, `List Policy`
- Reports (4):
  - `Get Billable`, `Get Snapshot`, `Get Snapshot Export`, `Get Summary`
- Saved Filters (4):
  - `Create Or Replace Saved Filter`, `Delete Saved Filter`, `Get Saved Filter`, `List Saved Filter`
- Tasks (5):
  - `Cancel Task`, `Download Task`, `Get Task`, `List Task`, `Run Task`

Total reviewed methods: `40`.

## Service Naming

Canonical Stackyard naming:

- service key: `defenderforeasm_dataplanepreview_2024_10_01_preview`
- display name: `Azure Defender EASM Data Plane Preview`
- API version: `2024-10-01-preview`

Recommended repo paths:

- implementation: `internal/server/provider_azure_defenderforeasm_dataplanepreview_2024_10_01_preview.go`
- tests: `internal/server/provider_azure_defenderforeasm_dataplanepreview_2024_10_01_preview_test.go`
- example: `examples/azure/defenderforeasm-dataplanepreview-2024-10-01-preview`
- plan doc: `docs/defenderforeasm-dataplanepreview-2024-10-01-preview-plan.md`

## Route Envelope

Upstream route shape uses `{endpoint}` and action-style segments, not ARM provider IDs.

Representative documented routes:

- `GET {endpoint}/assets/{assetId}?api-version=2024-10-01-preview`
- `GET {endpoint}/assets?api-version=2024-10-01-preview`
- `POST {endpoint}/assets:export?api-version=2024-10-01-preview&filter={filter}`
- `POST {endpoint}/assets:getDeltaDetails?api-version=2024-10-01-preview`
- `POST {endpoint}/assets:getDeltaSummary?api-version=2024-10-01-preview`
- `POST {endpoint}/assets/{assetId}:getObservations?api-version=2024-10-01-preview`
- `POST {endpoint}/assets?api-version=2024-10-01-preview&filter={filter}`
- `GET {endpoint}/cisaCves?api-version=2024-10-01-preview`
- `GET {endpoint}/cisaCves/{cveId}?api-version=2024-10-01-preview`
- `PUT {endpoint}/dataConnections/{dataConnectionName}?api-version=2024-10-01-preview`
- `POST {endpoint}/dataConnections:validate?api-version=2024-10-01-preview`
- `PUT {endpoint}/discoGroups/{groupName}?api-version=2024-10-01-preview`
- `POST {endpoint}/discoGroups:validate?api-version=2024-10-01-preview`
- `POST {endpoint}/discoGroups:getAssetChainSummary?api-version=2024-10-01-preview`
- `POST {endpoint}/discoGroups:dismissAssetChain?api-version=2024-10-01-preview`
- `GET {endpoint}/discoGroups/{groupName}/runs?api-version=2024-10-01-preview`
- `POST {endpoint}/discoGroups/{groupName}:run?api-version=2024-10-01-preview`
- `PUT {endpoint}/savedFilters/{filterName}?api-version=2024-10-01-preview`
- `PUT {endpoint}/policies/{policyName}?api-version=2024-10-01-preview`
- `POST {endpoint}/reports/assets:getBillable?api-version=2024-10-01-preview`
- `POST {endpoint}/reports/assets:getSummary?api-version=2024-10-01-preview`
- `POST {endpoint}/reports/assets:getSnapshot?api-version=2024-10-01-preview`
- `POST {endpoint}/reports/assets:getSnapshotExport?api-version=2024-10-01-preview`
- `POST {endpoint}/tasks/{taskId}:run?api-version=2024-10-01-preview`
- `POST {endpoint}/tasks/{taskId}:cancel?api-version=2024-10-01-preview`
- `POST {endpoint}/tasks/{taskId}:download?api-version=2024-10-01-preview`

Recommended Stackyard emulation prefix:

- `/azure/defenderforeasm-dataplanepreview/2024-10-01-preview`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:

- OAuth2 security scope shown throughout: `easm.defender.microsoft.com/.default`.
- Every reviewed method includes `Other Status Codes` with `Azure.Core.Foundations.ErrorResponse` and `x-ms-error-code` header.
- List responses consistently use paging shapes (`value`, commonly `totalElements`, and optional `nextLink`).
- Action-style POST operations are common (`:export`, `:validate`, `:run`, `:cancel`, `:download`, `:getSummary`, `:getSnapshot`, `:getSnapshotExport`).

Notable response patterns:

- `Delete Data Connection`, `Delete Disco Group`, and `Delete Saved Filter` return `204`.
- `Run Disco Group` returns `204`.
- `Get Assets Export`, `Update Assets`, and `Get Snapshot Export` return `Task` payloads (`200`).
- Read/list and validation routes generally return `200`.

Request-shape highlights:

- `Update Assets` requires a `filter` query and supports fields such as `labels`, `externalId`, `state`, `transfers`, `remediations`.
- `Validate Data Connection` and `Create Or Replace Data Connection` accept polymorphic connection payloads (`logAnalytics` / `azureDataExplorer`).
- Discovery-group validate/create bodies share the same shape (`description`, `frequencyMilliseconds`, `tier`, `seeds`, etc.).
- Reports snapshot/summary/export calls use request bodies with metric/filter controls.
- Task actions (`run`, `cancel`, `download`) are POST actions over `/tasks/{taskId}`.

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `defenderforeasm_dataplanepreview_2024_10_01_preview` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton and router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router can dispatch dataplane-prefix routes to this provider entrypoint

### Stage 1: Router Ownership and API-Version Gate

Implement:

- ownership for `/azure/defenderforeasm-dataplanepreview/2024-10-01-preview/...`
- strict `api-version=2024-10-01-preview` validation on owned routes
- deterministic `400` for invalid version and `405` for method mismatches

Exit criteria:

- dataplane routes are deterministically owned
- version/method failures are stable and testable

### Stage 2: Lookup and Listing Surfaces

Implement:

- Assets: `Get Asset Resource`, `List Asset Resource`
- Cisa Cves: both methods
- Discovery Templates: both methods
- Saved Filters: `Get`, `List`
- Policies: `List`
- Discovery Groups: `Get`, `List`, `List Runs`
- Tasks: `Get`, `List`

Behavior:

- deterministic fixture-backed stores
- stable ordering and optional pagination (`skip`, `maxpagesize`, `nextLink`)
- common error envelope for missing resources and invalid query parameters

Exit criteria:

- all read/list methods return stable JSON contracts

### Stage 3: Mutations and Action Routes

Implement:

- Assets: `Get Assets Export`, `Get Delta Details`, `Get Delta Summary`, `Get Observations`, `Update Assets`
- Data Connections: all 5 methods
- Discovery Groups: `Create/Replace`, `Delete`, `Validate`, `Run`, `Dismiss Asset Chain`, `Get Asset Chain Summary`
- Saved Filters: `Create/Replace`, `Delete`
- Policies: `Create Or Replace`
- Reports: all 4 methods
- Tasks: `Run`, `Cancel`, `Download`

Behavior:

- deterministic task creation for export/action-style routes
- idempotent deletes returning `204`
- validation endpoints return stable success/error envelopes
- action routes preserve documented status codes (`200`/`204`) where applicable

Exit criteria:

- all 40 methods are routed with deterministic responses and error shapes

### Stage 4: Tests and Contract Hardening

Add `internal/server/provider_azure_defenderforeasm_dataplanepreview_2024_10_01_preview_test.go` for:

- api-version failures
- invalid path/method behavior
- CRUD/list/action flows by operation group
- pagination fields and query handling
- status-code assertions for action/delete routes
- `Azure.Core.Foundations.ErrorResponse` shape assertions

Exit criteria:

- targeted server tests pass
- no owned dataplane route falls through the Azure mux

### Stage 5: Example and Coverage Enforcement

- keep example aligned to documented route shapes across all 9 operation groups
- enforce script discovery with required service checks
- retain strict coverage gates as implementation and tests progress

Exit criteria:

- example compiles and demonstrates route usage
- coverage scripts resolve key, aliases, and plan-doc mapping for this service

## Coverage Script Updates

Add aliases for `defenderforeasm_dataplanepreview_2024_10_01_preview` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Minimum alias set:

- `defenderforeasm_dataplanepreview_2024_10_01_preview`
- `defenderforeasm-dataplanepreview-2024-10-01-preview`
- `defenderforeasm_dataplanepreview`
- `defenderforeasm-dataplanepreview`
- `defenderforeasm_data_plane`
- `defender_easm_data_plane`
- `rest_defenderforeasm_dataplanepreview_2024_10_01_preview`

Keep existing generic aliases (`defenderforeasm`, `defender_easm`, etc.) mapped to control-plane service key to avoid backward-compatibility breaks.

## Example Scope

Example path:

- `examples/azure/defenderforeasm-dataplanepreview-2024-10-01-preview`

Example should execute representative calls spanning all operation groups and include:

- shared-key request signing policy for local emulator auth mode
- deterministic endpoint/api-version wiring
- request/response logging for each operation
- cleanup calls for mutable resources where applicable

## Validation Commands

- `go build ./examples/azure/defenderforeasm-dataplanepreview-2024-10-01-preview`
- `python3 scripts/azure-contract-coverage.py --service defenderforeasm-dataplanepreview-2024-10-01-preview --require-service defenderforeasm_dataplanepreview_2024_10_01_preview --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service defenderforeasm-dataplanepreview-2024-10-01-preview --require-service defenderforeasm_dataplanepreview_2024_10_01_preview --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service defenderforeasm-dataplanepreview-2024-10-01-preview --require-service defenderforeasm_dataplanepreview_2024_10_01_preview --fail-on any`
