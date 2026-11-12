# Azure Stack Admin (`azurestack-2019-08-08`) Staged Plan

## Objective

Emulate the Azure Stack Admin management API exposed by the reviewed `rest-azurestack-2019-08-08` pages with deterministic local behavior for:

- `Microsoft.Storage.Admin` provider `operations`
- `storageServices` subscription-level discovery
- location-scoped `settings`
- location-scoped `storageAccounts`
- storage-account action routes:
  - `undelete`
  - `reclaimStorageCapacity`
- location-scoped `acquisitions`
- location-scoped `asyncOperations`
- an Azure Go SDK example under `examples/azure/azurestack-2019-08-08`
- Azure coverage-script wiring for the new service key

Important scope note:

- The Microsoft Learn pages under `view=rest-azurestack-2019-08-08` resolve to the Azure Stack Admin storage resource provider surface.
- The reviewed routes are under `Microsoft.Storage.Admin`, with the method pages labeling the API version as `2019-08-08-preview`.
- This is not the broader Azure Stack Hub admin universe across `Microsoft.Fabric.Admin`, `Microsoft.Compute.Admin`, `Microsoft.Subscriptions.Admin`, and similar providers.
- The plan therefore targets the actual reviewed route family rather than the entire Azure Stack admin portfolio.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/azurestack/operation-groups?view=rest-azurestack-2019-08-08`

Operation-group and representative method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/azurestack/acquisitions/list?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/async-operations/get?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/storage-settings/get?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/storage-settings/update?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/storage-services/list-sub?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/storage-accounts/list?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/storage-accounts/get?view=rest-azurestack-2019-08-08`
- `https://learn.microsoft.com/en-us/rest/api/azurestack/storage-accounts/undelete?view=rest-azurestack-2019-08-08`

Azure Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/profiles/latest/azurestack/mgmt/azurestack`

SDK selection note:

- I did not find a modern track-2 `arm...` package for this Azure Stack Admin storage-provider surface.
- The closest official Go SDK is the older track-1 Azure Stack profile package above.
- Plan assumption: prefer the official profile client if the needed types/operations are present enough for a meaningful example; otherwise fall back to an `azcore/runtime` example while keeping the official package note in the implementation.

## Service Naming

Canonical Stackyard naming:

- service key: `azurestack_admin`
- display name: `Azure Stack Admin`
- provider namespace: `Microsoft.Storage.Admin`
- API version used by routes: `2019-08-08-preview`
- Learn view parameter: `rest-azurestack-2019-08-08`

Recommended repo paths:

- implementation: `internal/server/provider_azure_stack_admin.go`
- tests: `internal/server/provider_azure_stack_admin_test.go`
- example: `examples/azure/azurestack-2019-08-08`
- plan doc: `docs/azurestack-2019-08-08-plan.md`

Recommended coverage aliases:

- `azurestack_admin`
- `azurestack`
- `azure_stack`
- `azure-stack`
- `stack_admin`
- `stack-admin`
- `microsoft_storage_admin`
- `rest_azurestack_2019_08_08`
- `azurestack_2019_08_08`

## Surface Summary

The reviewed `2019-08-08` surface is compact and operational rather than deeply nested:

1. `Acquisitions`
2. `Async Operations`
3. `Storage Settings`
4. `Storage Accounts`
5. `Storage Services`

This is a good fit for a staged implementation that starts with discovery/read-only routes, then adds mutable settings and storage-account actions:

- provider-discovery and settings first
- storage-services and storage-accounts next
- action routes and async-operation polling after core resources are stable

## Route Envelope

Documented route family:

- all reviewed routes are ARM-style under `management.azure.com`
- every reviewed route uses `api-version=2019-08-08-preview` in the method pages
- all reviewed routes sit under `Microsoft.Storage.Admin`

Representative Stackyard route targets under `/azure`:

- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/storageServices?api-version=2019-08-08-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/settings?api-version=2019-08-08-preview`
- `PUT /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/settings?api-version=2019-08-08-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/storageAccounts?api-version=2019-08-08-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/storageAccounts/{accountId}?api-version=2019-08-08-preview`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/storageAccounts/{accountId}/undelete?api-version=2019-08-08-preview`
- `POST /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/storageAccounts/{accountId}/reclaimStorageCapacity?api-version=2019-08-08-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/acquisitions?api-version=2019-08-08-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Storage.Admin/locations/{location}/asyncOperations/{asyncOperationId}?api-version=2019-08-08-preview`

Important routing implications from the reviewed pages:

- routes are subscription- and location-scoped, not resource-group-scoped
- `storageAccounts/{accountId}` uses an internal admin account identifier rather than tenant-visible account name
- `undelete` is action-style `POST` with optional `newAccountName`
- mutation-style flows imply async polling support, so the emulator should keep a small `asyncOperations` store even if actions complete immediately

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use a small `ErrorResponse` shape
- most list routes return `value`
- several routes are read-only inventory/report surfaces rather than ARM tracked resources

### Acquisitions

Reviewed page:

- `List`

Key traits from the reviewed page:

- location-scoped list under `acquisitions`
- returns `AcquisitionList`
- acquisition rows contain storage-account/blob/container identifiers and a `status`

Modeling implications:

- expose a deterministic seeded list of acquisitions
- keep the shape small but typed so contract tools see a non-empty, realistic payload

### Async Operations

Reviewed page:

- `Get`

Key traits from the reviewed page:

- location-scoped by `asyncOperationId`
- response shape includes operation metadata, timestamps, HTTP status, and a serialized response payload

Modeling implications:

- maintain a canonical async-operation store keyed by `{subscriptionId,location,asyncOperationId}`
- action routes such as `undelete` and `reclaimStorageCapacity` should register a completed synthetic async operation for follow-up `GET`

### Storage Settings

Reviewed pages:

- `Get`
- `Update`

Key traits from the reviewed pages:

- singleton location-scoped settings resource
- current documented writable field is `retentionPeriodForDeletedStorageAccountsInDays`

Modeling implications:

- persist one settings object per `{subscriptionId,location}`
- support both the seeded default and mutable update flow

### Storage Services

Reviewed page:

- `List Sub`

Key traits from the reviewed page:

- subscription-scoped list, not nested below location
- returns `StorageService[]`

Modeling implications:

- expose a small deterministic subscription inventory
- keep rows stable enough to reference from the example and from follow-on storage-account fixtures

### Storage Accounts

Reviewed pages:

- `Get`
- `List`
- `Undelete`

Key traits from the reviewed pages:

- list is location-scoped and supports `$filter` and `summary`
- resource id/type/name are admin-facing
- properties include tenant cross-links such as:
  - `tenantViewId`
  - `tenantSubscriptionId`
  - `tenantStorageAccountName`
  - `tenantResourceGroupName`
- `undelete` is action-style and can optionally rename the tenant account

Modeling implications:

- persist canonical admin storage accounts keyed by internal `accountId`
- model both active and deleted states
- allow list/get to render either summary or detailed forms from the same underlying record
- implement `undelete` as a state transition plus optional tenant-name change
- include `reclaimStorageCapacity` in implementation even though I only found the operation summary page through the operation-group links rather than a fully indexed search result

## Staged Implementation

### Stage 0: Router and Coverage Wiring

Add the service shell and discovery:

- add `internal/server/provider_azure_stack_admin.go`
- register the new router in `internal/server/http.go`
- add `Server` storage maps for:
  - settings
  - storage services
  - storage accounts
  - acquisitions
  - async operations
- add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Acceptance target:

- service is discoverable by the coverage scripts
- no routed Azure Stack Admin path falls through to `NotImplemented`

### Stage 1: Read-Only Core

Implement the low-risk inventory endpoints first:

- `GET storageServices`
- `GET acquisitions`
- `GET asyncOperations/{asyncOperationId}`
- `GET settings`
- `GET storageAccounts`
- `GET storageAccounts/{accountId}`

Modeling notes:

- seed one subscription, one location, one storage service, and at least two storage-account rows
- include one active and one deleted storage account so action routes have meaningful state to mutate
- honor `summary=true|false` on storage-account list in a pragmatic way

Acceptance target:

- all read-only routes return stable, typed payloads
- tests cover list/get/not-found behavior

### Stage 2: Mutable Settings

Implement:

- `PUT settings`

Modeling notes:

- validate the integer field
- preserve seeded defaults if omitted
- return the updated singleton shape

Acceptance target:

- read-after-write succeeds
- invalid bodies return Azure-style 400s

### Stage 3: Storage Account Actions

Implement:

- `POST storageAccounts/{accountId}/undelete`
- `POST storageAccounts/{accountId}/reclaimStorageCapacity`

Modeling notes:

- `undelete` should:
  - require the account to exist in deleted state
  - optionally update tenant-facing account name from `newAccountName`
  - flip account status back to active
- `reclaimStorageCapacity` should:
  - be modeled as a successful admin action
  - optionally annotate reclaimed timestamp/counter in the account record
- both routes should emit synthetic async-operation records where useful

Acceptance target:

- action routes no longer return `NotImplemented`
- `asyncOperations/{id}` can reflect the most recent mutation

### Stage 4: Example

Add `examples/azure/azurestack-2019-08-08`:

- use the official Azure Go SDK package if practical:
  - `github.com/Azure/azure-sdk-for-go/profiles/latest/azurestack/mgmt/azurestack`
- if the track-1 package does not fit the reviewed storage-admin routes cleanly, use an `azcore/runtime` example and document that the official Azure Stack package is adjacent but not a good typed fit for this exact storage-admin subset

Suggested example flow:

1. list `storageServices`
2. get `settings`
3. update `settings`
4. list `storageAccounts`
5. get one `storageAccount`
6. undelete a deleted account
7. reclaim storage capacity
8. poll `asyncOperations/{id}`

Acceptance target:

- example builds cleanly
- example exercises the main admin routes without custom test-only hooks

### Stage 5: Validation and Coverage

Validation after implementation:

- `go test ./internal/server -run 'TestAzureStackAdmin' -count=1`
- `go build ./examples/azure/azurestack-2019-08-08`
- `python3 scripts/azure-contract-coverage.py --service azurestack_admin --require-service azurestack_admin --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service azurestack_admin --require-service azurestack_admin --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service azurestack_admin --require-service azurestack_admin --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service azurestack_admin --require-service azurestack_admin --fail-on any`

## Test Plan

Focused provider tests should cover:

- provider route detection and bad `api-version`
- settings get/update lifecycle
- storage-services subscription list
- storage-accounts list/get with both summary and detailed responses
- not-found cases for unknown `accountId` and `asyncOperationId`
- undelete success and invalid-state failure
- reclaim capacity success
- async operation retrieval after a mutation

## Risks and Follow-Ups

- The Learn `view=rest-azurestack-2019-08-08` operation-group page fronts a storage-admin subset, so naming must not imply full Azure Stack Hub admin parity across all providers.
- The official Go SDK support appears to be older track-1/profile based; example implementation may need a pragmatic compromise.
- Some pages are partially authorization-gated on Learn, so implementation should prefer the clearly exposed request/response fields and avoid overcommitting to undocumented optional fields.
