# Azure Data Share (`datashare_2021_08_01`) Staged Plan

## Objective

Emulate Azure Data Share REST API (`api-version=2021-08-01`) with deterministic local behavior for:

- account, share, invitation, and share-subscription lifecycle flows
- data-set and data-set-mapping lifecycle and list/query surfaces
- trigger and synchronization-setting management surfaces
- consumer/provider invitation and provider-subscription action flows
- synchronization history and details query surfaces
- an Azure Go SDK example under `examples/azure/datashare-2021-08-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/datashare/operation-groups?view=rest-datashare-2021-08-01`

Operation-group pages reviewed directly (when available):

- `.../accounts?view=rest-datashare-2021-08-01`
- `.../consumer-invitations?view=rest-datashare-2021-08-01`
- `.../data-sets?view=rest-datashare-2021-08-01`
- `.../share-subscriptions?view=rest-datashare-2021-08-01`
- `.../shares?view=rest-datashare-2021-08-01`
- `.../provider-share-subscriptions?view=rest-datashare-2021-08-01`

Method pages reviewed directly include:

- Accounts: `create`, `delete`, `get`, `list-by-resource-group`, `list-by-subscription`, `update`
- Consumer Invitations: `get`, `list-invitations`, `reject-invitation`
- Consumer Source Data Sets: `list-by-share-subscription`
- Data Set Mappings: `create`, `delete`, `get`, `list-by-share-subscription`
- Data Sets: `create`, `delete`, `get`, `list-by-share`
- Email Registrations: `register-email`
- Invitations: `create`, `delete`, `get`, `list-by-share`
- Operations: `list`
- Provider Share Subscriptions: `adjust`, `get-by-share`, `list-by-share`, `reinstate`, `revoke`
- Share Subscriptions: `cancel-synchronization`, `create`, `delete`, `get`, `list-by-account`, `list-source-share-synchronization-settings`, `list-synchronization-details`, `list-synchronizations`, `synchronize`
- Shares: `create`, `delete`, `get`, `list-by-account`, `list-synchronization-details`, `list-synchronizations`
- Synchronization Settings: `create`, `delete`, `get`, `list-by-share`
- Triggers: `create`, `delete`, `get`, `list-by-share-subscription`

SDK cross-check used for operation-group parity:

- `armdatashare` (Azure Go SDK resource-manager package)

Notes on source behavior in this environment:

- some operation-group pages are intermittently wrapped by authorization/cache shells
- method-level pages were used to complete full operation coverage
- where locale-specific mirrors surfaced first, method path/verb/shape were cross-checked against the corresponding English method pages and `armdatashare` client methods

## Surface Summary

Reviewed operation groups and methods:

- Accounts (6):
  - `Create`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Consumer Invitations (3):
  - `Get`, `List Invitations`, `Reject Invitation`
- Consumer Source Data Sets (1):
  - `List By Share Subscription`
- Data Set Mappings (4):
  - `Create`, `Delete`, `Get`, `List By Share Subscription`
- Data Sets (4):
  - `Create`, `Delete`, `Get`, `List By Share`
- Email Registrations (1):
  - `Register Email`
- Invitations (4):
  - `Create`, `Delete`, `Get`, `List By Share`
- Operations (1):
  - `List`
- Provider Share Subscriptions (5):
  - `Adjust`, `Get By Share`, `List By Share`, `Reinstate`, `Revoke`
- Share Subscriptions (9):
  - `Cancel Synchronization`, `Create`, `Delete`, `Get`, `List By Account`, `List Source Share Synchronization Settings`, `List Synchronization Details`, `List Synchronizations`, `Synchronize`
- Shares (6):
  - `Create`, `Delete`, `Get`, `List By Account`, `List Synchronization Details`, `List Synchronizations`
- Synchronization Settings (4):
  - `Create`, `Delete`, `Get`, `List By Share`
- Triggers (4):
  - `Create`, `Delete`, `Get`, `List By Share Subscription`

Total reviewed methods: `52`.

## Service Naming

Canonical Stackyard naming:

- service key: `datashare_2021_08_01`
- display name: `Azure Data Share`
- API version: `2021-08-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_datashare_2021_08_01.go`
- tests: `internal/server/provider_azure_datashare_2021_08_01_test.go`
- example: `examples/azure/datashare-2021-08-01`
- plan doc: `docs/datashare-2021-08-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DataShare/...`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataShare/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataShare/...`

Representative routes:

- `GET /azure/providers/Microsoft.DataShare/operations?api-version=2021-08-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataShare/accounts/{accountName}?api-version=2021-08-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataShare/accounts/{accountName}/shares/{shareName}?api-version=2021-08-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataShare/accounts/{accountName}/shares/{shareName}/dataSets/{dataSetName}?api-version=2021-08-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataShare/accounts/{accountName}/shareSubscriptions/{shareSubscriptionName}/dataSetMappings/{dataSetMappingName}?api-version=2021-08-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataShare/accounts/{accountName}/shareSubscriptions/{shareSubscriptionName}/synchronize?api-version=2021-08-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DataShare/accounts/{accountName}/shareSubscriptions/{shareSubscriptionName}/cancelSynchronization?api-version=2021-08-01`
- `POST /azure/providers/Microsoft.DataShare/locations/{location}/registerEmail?api-version=2021-08-01`
- `POST /azure/providers/Microsoft.DataShare/locations/{location}/rejectInvitation?api-version=2021-08-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- operations use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- resource routes follow ARM resource id and envelope patterns
- list responses use `value` and optional `nextLink`
- many list endpoints support optional `$skipToken` and some support `$filter`/`$orderby`
- common error envelope is `DataShareError` containing `DataShareErrorInfo`

Observed response patterns:

- create/update methods generally return `200` / `201`
- delete methods commonly return `200` / `204`
- several long-running flows expose `x-ms-long-running-operation` and `Azure-AsyncOperation`
- synchronization- and trigger-related operations include enum-rich status fields (`synchronizationMode`, `triggerStatus`, etc.)

Notable request/flow specifics:

- consumer invitation rejection uses provider path action route (`/providers/Microsoft.DataShare/locations/{location}/rejectInvitation`)
- provider share subscriptions include action routes (`adjust`, `reinstate`, `revoke`)
- share-subscription synchronization surfaces split between action endpoints and detail/history list endpoints
- data-set and data-set-mapping resources are discriminator-based (`kind`) across many source/sink shapes

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `datashare_2021_08_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve the service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Data Share paths to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for all Data Share path families
- strict `api-version=2021-08-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Data Share route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Accounts, Shares, Invitations, and Operations

Implement:

- Accounts: full CRUD/list set
- Shares: create/delete/get/list + sync list surfaces
- Invitations: create/delete/get/list
- Operations: provider operations list

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `properties`, `systemData`, `tags`)
- deterministic list ordering and stable paging behavior
- idempotent delete semantics

Exit criteria:

- core producer-side account/share/invitation lifecycle modeled end-to-end

### Stage 3: Data Sets and Synchronization Settings

Implement:

- Data Sets: create/delete/get/list
- Synchronization Settings: create/delete/get/list

Behavior:

- discriminator-aware payload validation (`kind`) for data-set resources
- deterministic readback of normalized `kind`/`properties`
- list paging semantics for both groups

Exit criteria:

- share-level data modeling and sync-setting lifecycle are covered

### Stage 4: Share Subscriptions, Triggers, and Data Set Mappings

Implement:

- Share Subscriptions: full CRUD/list/action/detail surface
- Triggers: create/delete/get/list
- Data Set Mappings: create/delete/get/list
- Consumer Source Data Sets: list

Behavior:

- synchronization action transitions (`synchronize`, `cancelSynchronization`)
- list detail/history endpoints for synchronizations
- mapping resource validation across mapping kinds and sinks

Exit criteria:

- consumer-side subscription and mapping flows are functionally modeled

### Stage 5: Consumer/Provider Side Invitation Actions

Implement:

- Consumer Invitations: get/list/reject
- Email Registrations: register email
- Provider Share Subscriptions: adjust/get/list/reinstate/revoke

Behavior:

- deterministic invitation lifecycle state transitions
- deterministic provider-side share-subscription status transitions
- action endpoints with expected status codes and body shapes

Exit criteria:

- invitation acceptance/rejection and provider revocation flows are represented

### Stage 6: Testing, Fixtures, and Contract Hardening

Add tests for:

- happy-path lifecycle across all 13 groups
- validation failures (missing path params, unsupported `kind`, malformed payloads)
- API-version/method enforcement and route ownership
- deterministic pagination and ordering
- enum/state-transition invariants

Exit criteria:

- `provider_azure_datashare_2021_08_01_test.go` covers stage 1-5 behavior
- service passes strict contract/IO/doc coverage checks

## Coverage Script Integration

Add aliases for `datashare_2021_08_01` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended aliases:

- `datashare_2021_08_01`
- `datashare-2021-08-01`
- `datashare`
- `data_share`
- `data-share`
- `azure_datashare`
- `microsoft_datashare`
- `rest_datashare_2021_08_01`

Map plan doc:

- `SERVICE_PLAN_DOCS["datashare_2021_08_01"] = ["datashare-2021-08-01-plan.md"]`

## Example Plan

Add `examples/azure/datashare-2021-08-01`:

- `main.go`: Azure Go SDK (`azcore/runtime`) authenticated client that exercises representative operations from every operation group
- `Dockerfile`: builds and runs the example binary
- `docker-compose.yml`: local stack wiring with Data Share env vars

Example coverage target:

- one representative request per reviewed method family
- lifecycle create/read/list/update/delete coverage for core account/share/subscription resources
- action endpoint coverage for sync, cancel, reject, register, adjust/reinstate/revoke

## Validation Commands

- `go test ./internal/server -run 'TestAzureDataShare20210801' -count=1`
- `go build ./examples/azure/datashare-2021-08-01`
- `python3 scripts/azure-contract-coverage.py --service datashare --require-service datashare_2021_08_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datashare --require-service datashare_2021_08_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datashare --require-service datashare_2021_08_01 --fail-on any`

Expected stage-0 result:

- router and docs/coverage wiring pass service discovery checks
- strict coverage checks fail until implementation/tests are added in later stages
