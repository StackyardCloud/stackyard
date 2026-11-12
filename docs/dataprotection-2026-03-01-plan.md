# Azure Data Protection (`dataprotection_2026_03_01`) Staged Plan

## Objective

Emulate Azure Data Protection REST API (`api-version=2026-03-01`) with deterministic local behavior for:

- backup vault lifecycle and settings
- backup policy and backup instance lifecycle
- backup job/query/status/result endpoints
- deleted resource and recovery-point surfaces
- resource-guard and critical-operation proxy flows
- provider operations discovery
- an Azure Go SDK example under `examples/azure/dataprotection-2026-03-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Top-level operation-group index reviewed:

- `https://learn.microsoft.com/en-us/rest/api/dataprotection/operation-groups?view=rest-dataprotection-2026-03-01`

Operation-group pages reviewed:

- `.../backup-instances?view=rest-dataprotection-2026-03-01`
- `.../backup-instances-extension-routing?view=rest-dataprotection-2026-03-01`
- `.../backup-policies?view=rest-dataprotection-2026-03-01`
- `.../backup-vault-operation-results?view=rest-dataprotection-2026-03-01`
- `.../backup-vaults?view=rest-dataprotection-2026-03-01`
- `.../data-protection?view=rest-dataprotection-2026-03-01`
- `.../data-protection-operations?view=rest-dataprotection-2026-03-01`
- `.../deleted-backup-instances?view=rest-dataprotection-2026-03-01`
- `.../deleted-backup-vaults?view=rest-dataprotection-2026-03-01`
- `.../dpp-resource-guard-proxy?view=rest-dataprotection-2026-03-01`
- `.../export-jobs?view=rest-dataprotection-2026-03-01`
- `.../export-jobs-operation-result?view=rest-dataprotection-2026-03-01`
- `.../fetch-cross-region-restore-job?view=rest-dataprotection-2026-03-01`
- `.../fetch-cross-region-restore-jobs?view=rest-dataprotection-2026-03-01`
- `.../fetch-secondary-recovery-points?view=rest-dataprotection-2026-03-01`
- `.../jobs?view=rest-dataprotection-2026-03-01`
- `.../operation-result?view=rest-dataprotection-2026-03-01`
- `.../operation-status?view=rest-dataprotection-2026-03-01`
- `.../operation-status-backup-vault-context?view=rest-dataprotection-2026-03-01`
- `.../operation-status-resource-group-context?view=rest-dataprotection-2026-03-01`
- `.../recovery-points?view=rest-dataprotection-2026-03-01`
- `.../resource-guards?view=rest-dataprotection-2026-03-01`
- `.../restorable-time-ranges?view=rest-dataprotection-2026-03-01`

Representative method pages reviewed directly include:

- `backup-instances/adhoc-backup`
- `backup-instances/create-or-update`
- `backup-instances/delete`
- `backup-instances/get`
- `backup-instances/get-backup-instance-operation-result`
- `backup-instances/list`
- `backup-instances/suspend-backups`
- `backup-instances/trigger-cross-region-restore`
- `backup-instances/trigger-rehydrate`
- `backup-instances/trigger-restore`
- `backup-instances/validate-cross-region-restore`
- `backup-instances/validate-for-backup`
- `backup-instances/validate-for-modify-backup`
- `backup-instances/validate-for-restore`
- `backup-instances-extension-routing/list`
- `backup-policies/create-or-update`
- `backup-policies/delete`
- `backup-policies/get`
- `backup-policies/list`
- `backup-vault-operation-results/get`
- `backup-vaults/check-name-availability`
- `backup-vaults/create-or-update`
- `backup-vaults/delete`
- `backup-vaults/get`
- `backup-vaults/get-in-resource-group`
- `backup-vaults/get-in-subscription`
- `backup-vaults/update`
- `data-protection/check-feature-support`
- `data-protection-operations/list`
- `deleted-backup-instances/get`
- `deleted-backup-instances/list`
- `deleted-backup-instances/undelete`
- `deleted-backup-vaults/get`
- `deleted-backup-vaults/list-by-location`
- `dpp-resource-guard-proxy/create-or-update`
- `dpp-resource-guard-proxy/delete`
- `dpp-resource-guard-proxy/get`
- `dpp-resource-guard-proxy/list`
- `dpp-resource-guard-proxy/unlock-delete`
- `export-jobs/trigger`
- `export-jobs-operation-result/get`
- `fetch-cross-region-restore-job/get`
- `fetch-cross-region-restore-jobs/list`
- `fetch-secondary-recovery-points/list`
- `jobs/get`
- `jobs/list`
- `operation-result/get`
- `operation-status/get`
- `operation-status-backup-vault-context/get`
- `operation-status-resource-group-context/get`
- `recovery-points/get`
- `recovery-points/list`
- `resource-guards/get-resources-in-resource-group`
- `resource-guards/get-resources-in-subscription`
- `resource-guards/patch`
- `resource-guards/put`
- `restorable-time-ranges/find`

Note on Learn behavior:
- a subset of method pages intermittently rendered as access-gated or cache-miss in this environment, so operation/method names were additionally confirmed from operation-group pages and method search indices for the same `2026-03-01` API version.

## Surface Summary

Reviewed operation groups and methods:

- Backup Instances (18):
  - `Adhoc Backup`, `Create Or Update`, `Delete`, `Get`, `Get Backup Instance Operation Result`, `List`, `Resume Backups`, `Resume Protection`, `Stop Protection`, `Suspend Backups`, `Sync Backup Instance`, `Trigger Cross Region Restore`, `Trigger Rehydrate`, `Trigger Restore`, `Validate Cross Region Restore`, `Validate For Backup`, `Validate For Modify Backup`, `Validate For Restore`
- Backup Instances Extension Routing (1):
  - `List`
- Backup Policies (4):
  - `Create Or Update`, `Delete`, `Get`, `List`
- Backup Vault Operation Results (1):
  - `Get`
- Backup Vaults (7):
  - `Check Name Availability`, `Create Or Update`, `Delete`, `Get`, `Get In Resource Group`, `Get In Subscription`, `Update`
- Data Protection (1):
  - `Check Feature Support`
- Data Protection Operations (1):
  - `List`
- Deleted Backup Instances (3):
  - `Get`, `List`, `Undelete`
- Deleted Backup Vaults (2):
  - `Get`, `List By Location`
- Dpp Resource Guard Proxy (5):
  - `Create Or Update`, `Delete`, `Get`, `List`, `Unlock Delete`
- Export Jobs (1):
  - `Trigger`
- Export Jobs Operation Result (1):
  - `Get`
- Fetch Cross Region Restore Job (1):
  - `Get`
- Fetch Cross Region Restore Jobs (1):
  - `List`
- Fetch Secondary Recovery Points (1):
  - `List`
- Jobs (2):
  - `Get`, `List`
- Operation Result (1):
  - `Get`
- Operation Status (1):
  - `Get`
- Operation Status Backup Vault Context (1):
  - `Get`
- Operation Status Resource Group Context (1):
  - `Get`
- Recovery Points (2):
  - `Get`, `List`
- Resource Guards (18):
  - `Delete`, `Get`, `Get Backup Security PIN Requests Objects`, `Get Default Backup Security PIN Requests Object`, `Get Default Delete Protected Item Requests Object`, `Get Default Delete Resource Guard Proxy Requests Object`, `Get Default Disable Soft Delete Requests Object`, `Get Default Update Protected Item Requests Object`, `Get Default Update Protection Policy Requests Object`, `Get Delete Protected Item Requests Objects`, `Get Delete Resource Guard Proxy Requests Objects`, `Get Disable Soft Delete Requests Objects`, `Get Resources In Resource Group`, `Get Resources In Subscription`, `Get Update Protected Item Requests Objects`, `Get Update Protection Policy Requests Objects`, `Patch`, `Put`
- Restorable Time Ranges (1):
  - `Find`

Total reviewed methods: `75`.

## Service Naming

Canonical Stackyard naming:

- service key: `dataprotection_2026_03_01`
- display name: `Azure Data Protection`
- API version: `2026-03-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_dataprotection_2026_03_01.go`
- tests: `internal/server/provider_azure_dataprotection_2026_03_01_test.go`
- example: `examples/azure/dataprotection-2026-03-01`
- plan doc: `docs/dataprotection-2026-03-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DataProtection/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataProtection/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataProtection/...`

Representative routes:

- `POST /azure/subscriptions/{sub}/providers/Microsoft.DataProtection/locations/{location}/checkFeatureSupport?api-version=2026-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataProtection/backupVaults/{vaultName}?api-version=2026-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataProtection/backupVaults/{vaultName}/backupPolicies/{policyName}?api-version=2026-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataProtection/backupVaults/{vaultName}/backupInstances/{backupInstanceName}?api-version=2026-03-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataProtection/backupVaults/{vaultName}/backupInstances/{backupInstanceName}/backup?api-version=2026-03-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataProtection/backupVaults/{vaultName}/exportBackupJobs?api-version=2026-03-01`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataProtection/backupVaults/{vaultName}/backupJobs/operations/{operationId}?api-version=2026-03-01`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.DataProtection/locations/{location}/operationStatus/{operationId}?api-version=2026-03-01`
- `GET /azure/subscriptions/{sub}/providers/Microsoft.DataProtection/locations/{location}/operationResults/{operationId}?api-version=2026-03-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed operations use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- expected API version is `2026-03-01`
- operation surfaces are strongly LRO-oriented in create/update/delete/action endpoints
- common error envelopes observed: `CloudError`, `ErrorResponse`/`OData` variants

Observed response patterns:

- create/update surfaces commonly return `200`/`201`
- action surfaces commonly return `200` and/or `202` with `Azure-AsyncOperation`, `Location`, `Retry-After`
- delete surfaces often include `202` and idempotent `204`
- list responses generally follow `value` + optional continuation

Notable request/flow specifics:

- backup instance actions include backup/restore/rehydrate/suspend/resume/validate flows
- resource-guard proxy has explicit `unlockDelete` flow before deletion
- export jobs are two-step (`trigger` then operation-result fetch for blob links)
- operation status/result are available across provider, backup-vault context, and resource-group context

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `dataprotection_2026_03_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve the service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Data Protection paths to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- ownership for `/providers/Microsoft.DataProtection/` route families
- strict `api-version=2026-03-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- all Data Protection route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Core Vault/Policy/Instance CRUD

Implement:

- Backup Vaults: full CRUD/list/update + name availability
- Backup Policies: create/get/list/delete
- Backup Instances: create/get/list/delete + operation-result get

Behavior:

- canonical ARM resource envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic ordering for list responses
- idempotent delete behavior where documented

Exit criteria:

- no `NotImplemented` for core vault/policy/instance CRUD methods

### Stage 3: Actions, Jobs, and Recovery Surfaces

Implement:

- Backup Instance action APIs (backup, suspend/resume, stop/resume protection, sync, validate*, restore*)
- Jobs (`Get`, `List`)
- Recovery Points (`Get`, `List`)
- Restorable Time Ranges (`Find`)
- Fetch Secondary Recovery Points

Behavior:

- deterministic LRO envelopes for action endpoints
- deterministic job/protection state transitions
- stable payload fixtures for validation/query endpoints

Exit criteria:

- action/job/recovery families return modeled non-`NotImplemented` responses

### Stage 4: Guard, Deleted, Export, and Operation-Tracking Surfaces

Implement:

- Dpp Resource Guard Proxy (create/get/list/unlock-delete/delete)
- Resource Guards operation family (all 18 methods)
- Deleted Backup Instances and Deleted Backup Vaults families
- Export Jobs trigger/result
- Operation status/result families (provider, vault context, rg context)
- Data Protection check-feature-support + provider operations list

Exit criteria:

- full operation-group coverage for all 75 reviewed methods
- critical flow dependencies (unlock-delete, export trigger->result, LRO polling) are modeled

### Stage 5: Tests

Add provider tests for:

- API-version/method gating and ownership
- vault/policy/instance lifecycle
- action/LRO behavior and operation status/result endpoints
- resource-guard + proxy flows including unlock-delete precondition
- deleted-resource and export job flows
- pagination and idempotent delete semantics

Exit criteria:

- `go test ./internal/server -run 'TestAzureDataProtection20260301' -count=1` passes

### Stage 6: Example and Validation

Add `examples/azure/dataprotection-2026-03-01`:

- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example should cover representative calls across:

- Data Protection Operations
- Data Protection (feature support)
- Backup Vaults
- Backup Policies
- Backup Instances
- Jobs
- Export Jobs + Export Jobs Operation Result
- Operation status/result endpoints

Validation commands:

- `go build ./examples/azure/dataprotection-2026-03-01`
- `python3 scripts/azure-contract-coverage.py --service dataprotection --require-service dataprotection_2026_03_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service dataprotection --require-service dataprotection_2026_03_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service dataprotection --require-service dataprotection_2026_03_01 --fail-on any`

Success target:

- coverage scripts resolve and report `dataprotection_2026_03_01`
- example compiles and exercises the staged surface contract
