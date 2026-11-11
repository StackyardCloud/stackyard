# Azure Dell Storage (`dellstorage_2025_03_21`) Staged Plan

## Objective

Emulate Azure Dell Storage REST API (`api-version=2025-03-21`) with deterministic local behavior for:

- provider operations discovery
- Dell filesystem lifecycle and update flows
- list/query surfaces across resource-group and subscription scopes
- an Azure Go SDK example under `examples/azure/dellstorage-2025-03-21`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/dellstorage/operation-groups?view=rest-dellstorage-2025-03-21`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/operations?view=rest-dellstorage-2025-03-21`

Method pages reviewed directly:

- `https://learn.microsoft.com/en-us/rest/api/dellstorage/operations/list?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems/list-by-subscription?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems/list-by-resource-group?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems/get?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems/create-or-update?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems/update?view=rest-dellstorage-2025-03-21`
- `https://learn.microsoft.com/en-us/rest/api/dellstorage/file-systems/delete?view=rest-dellstorage-2025-03-21`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dell/armdellstorage`

Notes on source behavior in this environment:

- several Learn pages are intermittently wrapped by auth/cache shells
- operation-group plus method pages were still usable to validate path/verb, request/response, and error contracts

## Surface Summary

Reviewed operation groups and methods:

- Operations (1):
  - `List`
- File Systems (6):
  - `List By Subscription`, `List By Resource Group`, `Get`, `Create Or Update`, `Update`, `Delete`

Total reviewed methods: `7`.

## Service Naming

Canonical Stackyard naming:

- service key: `dellstorage_2025_03_21`
- display name: `Azure Dell Storage`
- API version: `2025-03-21`

Recommended repo paths:

- implementation: `internal/server/provider_azure_dellstorage_2025_03_21.go`
- tests: `internal/server/provider_azure_dellstorage_2025_03_21_test.go`
- example: `examples/azure/dellstorage-2025-03-21`
- plan doc: `docs/dellstorage-2025-03-21-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Dell.Storage/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Dell.Storage/filesystems`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dell.Storage/filesystems`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Dell.Storage/filesystems/{filesystemName}`

Representative routes:

- `GET /azure/providers/Dell.Storage/operations?api-version=2025-03-21`
- `GET /azure/subscriptions/{sub}/providers/Dell.Storage/filesystems?api-version=2025-03-21`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Dell.Storage/filesystems?api-version=2025-03-21`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Dell.Storage/filesystems/{filesystemName}?api-version=2025-03-21`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Dell.Storage/filesystems/{filesystemName}?api-version=2025-03-21`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Dell.Storage/filesystems/{filesystemName}?api-version=2025-03-21`
- `DELETE /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Dell.Storage/filesystems/{filesystemName}?api-version=2025-03-21`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- list methods are pageable with `value` and optional `nextLink`
- all methods document `ErrorResponse` for non-success paths

Observed response patterns:

- `Create Or Update`: `200` (update) or `201` (create), documented as LRO-capable with `Azure-AsyncOperation` and `Retry-After` headers on create
- `Update`: `200`
- `Delete`: `202` (accepted, LRO via `Location`/`Retry-After`) or `204` (resource absent)
- `Get` / `List*` / `Operations List`: `200`

Notable payload constraints:

- `filesystemName` uses pattern `^[a-zA-Z0-9_-]*$`
- create payload requires `location` and `properties`, and `properties` requires:
  - `marketplace` (`planId`, `offerId`, `planName`)
  - `delegatedSubnetId`
  - `delegatedSubnetCidr` (CIDR pattern with `/16` or `/24`)
  - `user` (`email`)
  - `dellReferenceNumber`
  - `encryption` (`encryptionType`)
- `encryptionType` enum values:
  - `Microsoft-managed keys (MMK)`
  - `Customer-managed keys (CMK)`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `dellstorage_2025_03_21` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Dell Storage paths to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Dell.Storage/` families
- strict `api-version=2025-03-21` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Dell Storage route families are deterministically owned
- version and method failures are stable and testable

### Stage 2: Read/List and Provider Operations

Implement:

- `Operations_List`
- `FileSystems_ListBySubscription`
- `FileSystems_ListByResourceGroup`
- `FileSystems_Get`

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic list ordering and optional `nextLink` paging behavior
- stable projection of required filesystem properties

Exit criteria:

- all read/list surfaces return deterministic modeled payloads

### Stage 3: Create/Update/Delete Lifecycle

Implement:

- `FileSystems_CreateOrUpdate`
- `FileSystems_Update`
- `FileSystems_Delete`

Behavior:

- required create payload validation for marketplace, network, user, reference, and encryption fields
- patch semantics for `tags`, `identity`, and updatable properties
- idempotent delete with `202`/`204` behavior and LRO headers where documented

Exit criteria:

- no `NotImplemented` for filesystem lifecycle methods
- lifecycle transitions are deterministic and contract-testable

### Stage 4: Tests

Add provider tests for:

- api-version and method gating across all owned routes
- filesystem create/get/update/delete flows
- list-by-resource-group and list-by-subscription scoping
- payload validation and deterministic error envelopes
- idempotent delete and LRO-style response headers

Exit criteria:

- `go test ./internal/server -run 'TestAzureDellStorage20250321' -count=1` passes

### Stage 5: Example and Validation

Add `examples/azure/dellstorage-2025-03-21`:

- `main.go` with representative calls for all 7 reviewed methods
- `Dockerfile`
- `docker-compose.yml`

Validation:

- run example against local Stackyard Azure provider endpoint
- verify request/response envelopes and status codes against reviewed method docs

Exit criteria:

- example executes end-to-end against emulated routes
- coverage scripts include the new service key and doc mapping

