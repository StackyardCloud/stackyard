# Azure Defender for Cloud (`defenderforcloud_2025_06_01`) Staged Plan

## Objective

Emulate Azure Defender for Cloud REST API (`api-version=2025-06-01`) with deterministic local behavior for:

- Defender for Storage setting lifecycle on storage-account resources
- malware scan start/get/cancel flows for storage accounts
- deterministic error and validation behavior matching documented contracts
- an Azure Go SDK example under `examples/azure/defenderforcloud-2025-06-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- For `2025-06-01`, the requested operation-groups page currently exposes one operation group: `Defender For Storage`.
- This service surface is ARM management-plane (`management.azure.com`) and attaches under `Microsoft.Security/defenderForStorageSettings` beneath an existing resource id.

## Primary References Reviewed

Requested operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/operation-groups?view=rest-defenderforcloud-2025-06-01`

Operation-group page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/defender-for-storage?view=rest-defenderforcloud-2025-06-01`

Method pages reviewed directly (all linked methods):

- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/defender-for-storage/create?view=rest-defenderforcloud-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/defender-for-storage/get?view=rest-defenderforcloud-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/defender-for-storage/start-malware-scan?view=rest-defenderforcloud-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/defender-for-storage/get-malware-scan?view=rest-defenderforcloud-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/defenderforcloud/defender-for-storage/cancel-malware-scan?view=rest-defenderforcloud-2025-06-01`

Notes on source behavior in this environment:

- pages intermittently include authorization wrappers
- operation route, request/response contracts, and error-shape sections remain readable
- all linked methods above were opened and reviewed

## Surface Summary

Reviewed operation groups and methods:

- Defender For Storage (5):
  - `Create`
  - `Get`
  - `Start Malware Scan`
  - `Get Malware Scan`
  - `Cancel Malware Scan`

Total reviewed methods: `5`.

## Service Naming

Canonical Stackyard naming:

- service key: `defenderforcloud_2025_06_01`
- display name: `Azure Defender for Cloud`
- API version: `2025-06-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_defenderforcloud_2025_06_01.go`
- tests: `internal/server/provider_azure_defenderforcloud_2025_06_01_test.go`
- example: `examples/azure/defenderforcloud-2025-06-01`
- plan doc: `docs/defenderforcloud-2025-06-01-plan.md`

## Route Envelope

Management-plane route family to emulate:

- `/azure/{resourceId}/providers/Microsoft.Security/defenderForStorageSettings/current`

Representative documented routes:

- `PUT /azure/{resourceId}/providers/Microsoft.Security/defenderForStorageSettings/current?api-version=2025-06-01`
- `GET /azure/{resourceId}/providers/Microsoft.Security/defenderForStorageSettings/current?api-version=2025-06-01`
- `POST /azure/{resourceId}/providers/Microsoft.Security/defenderForStorageSettings/current/startMalwareScan?api-version=2025-06-01`
- `GET /azure/{resourceId}/providers/Microsoft.Security/defenderForStorageSettings/current/malwareScans/{scanId}?api-version=2025-06-01`
- `POST /azure/{resourceId}/providers/Microsoft.Security/defenderForStorageSettings/current/malwareScans/{scanId}/cancelMalwareScan?api-version=2025-06-01`

Where `{resourceId}` is the target storage account ARM id, for example:

- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{storageAccountName}`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- API version is required on all methods (`api-version=2025-06-01`)
- all methods include `Other Status Codes` with `Cloud Error` envelope
- setting name is an enum/pattern constrained to `current`

Observed request/response patterns:

- `Create` (`PUT`) request body uses `properties` with documented fields including:
  - `isEnabled`
  - `malwareScanning` (`onUpload`, `scanResultsEventGridTopicResourceId`, `blobScanResultsOptions`)
  - `sensitiveDataDiscovery`
  - `overrideSubscriptionLevelSettings`
- `Create` responses: `200` or `201` with `DefenderForStorageSetting`
- `Get` response: `200` with `DefenderForStorageSetting`
- `Start Malware Scan` response: `200` with `MalwareScan` (queued scan state)
- `Get Malware Scan` response: `200` with `MalwareScan` (includes scan status/summary)
- `Cancel Malware Scan` response: `200` with `MalwareScan` (canceling state)
- `scanId` parameter for get/cancel supports `latest` or GUID pattern

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `defenderforcloud_2025_06_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Defender for Cloud routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `Microsoft.Security/defenderForStorageSettings` families
- strict `api-version=2025-06-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Defender for Cloud route family is deterministically owned
- version/method errors are stable and testable

### Stage 2: Defender for Storage Setting CRUD Surface

Implement:

- `Create`
- `Get`

Behavior:

- deterministic in-memory setting state keyed by storage `resourceId`
- request-body validation for required and typed fields used by example/doc contracts
- response envelope parity (`id`, `name`, `type`, `properties`)

Exit criteria:

- setting create/update/get routes return deterministic contract-conformant payloads

### Stage 3: Malware Scan Action Surface

Implement:

- `Start Malware Scan`
- `Get Malware Scan`
- `Cancel Malware Scan`

Behavior:

- scan state machine fixtures (`Queued` -> `Completed`; cancel path to `Canceling`)
- support `scanId=latest` and specific GUID lookups
- return `MalwareScan` with `scanSummary` fields consistent with docs

Exit criteria:

- all malware-scan flows return stable responses and correct not-found/validation errors

### Stage 4: Tests and Contract Hardening

Add `internal/server/provider_azure_defenderforcloud_2025_06_01_test.go` for:

- API-version and path ownership validation
- method-not-allowed behavior
- setting create/get roundtrip behavior
- malware scan start/get/cancel behavior
- `latest` scan-id semantics
- `Cloud Error` envelope shape assertions for negative paths

Exit criteria:

- targeted provider tests pass with deterministic contracts

### Stage 5: Example and Coverage Enforcement

- keep example aligned to all 5 reviewed methods
- enforce service discovery through coverage scripts
- retain strict failure gates until implementation/tests are complete

Exit criteria:

- example compiles and demonstrates all method calls
- coverage scripts discover service key and plan-doc mapping

## Coverage Script Updates

Add aliases for `defenderforcloud_2025_06_01` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Minimum alias set:

- `defenderforcloud_2025_06_01`
- `defenderforcloud-2025-06-01`
- `defenderforcloud`
- `defender_for_cloud`
- `defender-for-cloud`
- `defenderforstorage`
- `defender_for_storage`
- `defender-for-storage`
- `azure_defenderforcloud`
- `rest_defenderforcloud_2025_06_01`

## Example Scope

Example path:

- `examples/azure/defenderforcloud-2025-06-01`

Example should:

- use Azure Go SDK core pipeline (`azcore/runtime`, `azcore/policy`) for HTTP execution
- execute representative calls for all 5 methods
- log each request outcome and response JSON
- include cleanup-friendly behavior by using idempotent read/action flows

## Validation Commands

- `go build ./examples/azure/defenderforcloud-2025-06-01`
- `python3 scripts/azure-contract-coverage.py --service defenderforcloud-2025-06-01 --require-service defenderforcloud_2025_06_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service defenderforcloud-2025-06-01 --require-service defenderforcloud_2025_06_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service defenderforcloud-2025-06-01 --require-service defenderforcloud_2025_06_01 --fail-on any`
