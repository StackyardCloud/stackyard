# Azure Data Lake Storage Gen1 (`datalakestore_2016_11_01`) Staged Plan

## Objective

Emulate Azure Data Lake Storage Gen1 REST API (`api-version=2016-11-01`) with deterministic local behavior for:

- management-plane account lifecycle and subresources
- location and operations discovery surfaces
- an Azure Go SDK example under `examples/azure/datalakestore-2016-11-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Top-level references reviewed:

- `https://learn.microsoft.com/en-us/rest/api/datalakestore/`
- `https://learn.microsoft.com/en-us/rest/api/datalakestore/operation-groups?view=rest-datalakestore-2016-11-01`
- `https://learn.microsoft.com/en-us/rest/api/datalakestore/locations?view=rest-datalakestore-2016-11-01`

Operation-group pages and method pages reviewed:

- Accounts operation group:
  - `https://learn.microsoft.com/id-id/rest/api/datalakestore/accounts?view=rest-datalakestore-2016-11-01` (operation list includes `Enable Key Vault`)
  - `.../accounts/check-name-availability`
  - `.../accounts/create`
  - `.../accounts/delete`
  - `.../accounts/get`
  - `.../accounts/list`
  - `.../accounts/list-by-resource-group`
  - `.../accounts/update`
  - `.../accounts/enable-key-vault` (link target identified from operation-group listing; page was intermittently unavailable via Learn cache in this environment)
- Firewall Rules:
  - `.../firewall-rules/create-or-update`
  - `.../firewall-rules/get`
  - `.../firewall-rules/list-by-account`
  - `.../firewall-rules/delete`
  - `.../firewall-rules/update`
- Trusted Id Providers:
  - `.../trusted-id-providers/create-or-update`
  - `.../trusted-id-providers/get`
  - `.../trusted-id-providers/list-by-account`
  - `.../trusted-id-providers/delete`
  - `.../trusted-id-providers/update`
- Virtual Network Rules:
  - `.../virtual-network-rules/create-or-update`
  - `.../virtual-network-rules/get`
  - `.../virtual-network-rules/list-by-account`
  - `.../virtual-network-rules/delete`
  - `.../virtual-network-rules/update`
- Locations:
  - `.../locations/get-capability`
  - `.../locations/get-usage`
- Operations:
  - `.../operations/list`

## Service Naming

Canonical Stackyard naming:

- service key: `datalakestore_2016_11_01`
- display name: `Azure Data Lake Storage Gen1`
- API version: `2016-11-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_datalakestore_2016_11_01.go`
- tests: `internal/server/provider_azure_datalakestore_2016_11_01_test.go`
- example: `examples/azure/datalakestore-2016-11-01`
- plan doc: `docs/datalakestore-2016-11-01-plan.md`

## Route Envelope

Management-plane (ARM) patterns:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataLakeStore/locations/{location}/checkNameAvailability?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeStore/accounts/{accountName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataLakeStore/accounts?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeStore/accounts?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeStore/accounts/{accountName}/firewallRules/{firewallRuleName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeStore/accounts/{accountName}/trustedIdProviders/{trustedIdProviderName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeStore/accounts/{accountName}/virtualNetworkRules/{virtualNetworkRuleName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataLakeStore/locations/{location}/capability?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataLakeStore/locations/{location}/usages?api-version=2016-11-01`
- `/azure/providers/Microsoft.DataLakeStore/operations?api-version=2016-11-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- All reviewed methods use `api-version=2016-11-01`.
- Authentication is Azure AD OAuth2 (`azure_auth`, `user_impersonation`).
- Standard ARM resource identifiers and envelopes are used.

Response/status characteristics to model:

- Account create returns `200`/`201`.
- Account update returns `200`/`201`/`202`.
- Account delete returns `200`/`202`/`204`.
- Subresource create/update/get/list operations return `200` with typed payloads.
- Subresource deletes return `200` and idempotent `204` on missing resources (firewall/trusted-id-provider/virtual-network-rule delete pages).
- Location capability returns `200` and documented `404` for missing subscription context.
- List operations return `value` arrays and commonly include `nextLink`.
- Account list supports OData query controls (`$filter`, `$top`, `$skip`, `$select`, `$orderby`, `$count`).

Known operation groups and methods from the reviewed operation pages:

- Accounts:
  - `CheckNameAvailability`, `Create`, `Delete`, `EnableKeyVault`, `Get`, `List`, `ListByResourceGroup`, `Update`
- Firewall Rules:
  - `CreateOrUpdate`, `Delete`, `Get`, `ListByAccount`, `Update`
- Trusted Id Providers:
  - `CreateOrUpdate`, `Delete`, `Get`, `ListByAccount`, `Update`
- Virtual Network Rules:
  - `CreateOrUpdate`, `Delete`, `Get`, `ListByAccount`, `Update`
- Locations:
  - `GetCapability`, `GetUsage`
- Operations:
  - `List`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `datalakestore_2016_11_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton file and wire router in `internal/server/http.go`

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps key to this plan doc
- Azure router chain includes Data Lake Store Gen1 handler

### Stage 1: Router Ownership and Version Gate

Implement:

- ownership for Data Lake Store ARM route families
- strict `api-version=2016-11-01` validation
- deterministic `400` for missing/invalid version
- deterministic `405` for unsupported methods on owned paths

Exit criteria:

- requests for Data Lake Store Gen1 envelopes are consistently owned
- invalid version behavior is stable across all envelopes

### Stage 2: Core Account Lifecycle

Implement:

- `Accounts_CheckNameAvailability`
- `Accounts_Create`, `Accounts_Get`, `Accounts_List`, `Accounts_ListByResourceGroup`, `Accounts_Update`, `Accounts_Delete`

Behavior:

- stable account fixture shape (`id`, `name`, `type`, `location`, `properties`, `tags`)
- deterministic create/update/delete transitions
- paged list envelopes with stable ordering and `nextLink`
- OData parameter handling for list routes

Exit criteria:

- account create/get/list/update/delete flows work end-to-end
- idempotent delete semantics modeled (`204` for absent resources)

### Stage 3: Account Subresource Surfaces

Implement:

- Firewall Rules: `CreateOrUpdate`, `Get`, `ListByAccount`, `Update`, `Delete`
- Trusted Id Providers: `CreateOrUpdate`, `Get`, `ListByAccount`, `Update`, `Delete`
- Virtual Network Rules: `CreateOrUpdate`, `Get`, `ListByAccount`, `Update`, `Delete`

Behavior:

- subresource stores keyed by `{subscription, rg, account, subresource-name}`
- replace semantics for `PUT`, partial semantics for `PATCH` where documented
- idempotent delete behavior with `200`/`204`

Exit criteria:

- all reviewed subresource methods return modeled responses
- list envelopes include stable `value` ordering and optional `nextLink`

### Stage 4: Locations and Provider Metadata

Implement:

- `Locations_GetCapability`
- `Locations_GetUsage`
- `Operations_List`

Behavior:

- deterministic capability/usage fixtures by `{subscription, location}`
- documented `404` capability path where subscription context is missing/invalid
- operations listing fixture with pagination-compatible envelope

Exit criteria:

- location and operations endpoints satisfy SDK list/get patterns
- pagination and metadata fields are stable for contract tests

### Stage 5: Enable Key Vault and Polishing

Implement:

- `Accounts_EnableKeyVault` (operation listed in Accounts group)

Behavior:

- operation state transition from service-managed to user-managed encryption modeling
- realistic failure modes where account or encryption prerequisites are invalid

Validation:

- add strict contract tests per method family (version, path, method, required payload, negative cases)
- verify emulator fixtures against examples and coverage scripts

Exit criteria:

- no `NotImplemented` for operation-group method inventory
- deterministic request/response contracts for all documented methods

## Example Scope (`examples/azure/datalakestore-2016-11-01`)

The example should demonstrate representative end-to-end Azure Go SDK calls covering:

- account name availability check
- account create/get/update/list/list-by-rg/delete
- firewall rule create/get/update/list/delete
- trusted id provider create/get/update/list/delete
- virtual network rule create/get/update/list/delete
- location capability and usage
- operations list

## Coverage Script Updates Required

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases mapping all common forms to `datalakestore_2016_11_01`, including:

- `datalakestore_2016_11_01`
- `datalakestore-2016-11-01`
- `datalakestore`
- `data_lake_store`
- `data-lake-store`
- `azure_datalakestore`
- `azure_data_lake_store`
- `microsoft_datalakestore`
- `rest_datalakestore_2016_11_01`
