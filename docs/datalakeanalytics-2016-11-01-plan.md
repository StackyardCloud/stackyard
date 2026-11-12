# Azure Data Lake Analytics (`datalakeanalytics_2016_11_01`) Staged Plan

## Objective

Emulate Azure Data Lake Analytics REST API (`api-version=2016-11-01`) with deterministic local behavior for:

- management-plane account lifecycle and subresources
- catalog operations
- job operations
- pipeline and recurrence read/query operations
- an Azure Go SDK example under `examples/azure/datalakeanalytics-2016-11-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Top-level references reviewed:

- `https://learn.microsoft.com/en-us/rest/api/datalakeanalytics/`
- `https://learn.microsoft.com/en-us/rest/api/datalakeanalytics/accounts?view=rest-datalakeanalytics-2016-11-01`
- `https://learn.microsoft.com/en-us/rest/api/datalakeanalytics/storage-accounts?view=rest-datalakeanalytics-2016-11-01`
- `https://learn.microsoft.com/en-us/rest/api/datalakeanalytics/catalog?view=rest-datalakeanalytics-2016-11-01`
- `https://learn.microsoft.com/en-us/rest/api/datalakeanalytics/job?view=rest-datalakeanalytics-2016-11-01`

Additional operation/method pages reviewed:

- Accounts:
  - `.../accounts/check-name-availability`
  - `.../accounts/create`
  - `.../accounts/delete`
  - `.../accounts/get`
  - `.../accounts/list`
  - `.../accounts/list-by-resource-group`
  - `.../accounts/update`
- Storage Accounts:
  - `.../storage-accounts/add`
  - `.../storage-accounts/delete`
  - `.../storage-accounts/get`
  - `.../storage-accounts/get-storage-container`
  - `.../storage-accounts/list-by-account`
  - `.../storage-accounts/list-sas-tokens`
  - `.../storage-accounts/list-storage-containers`
  - `.../storage-accounts/update`
- Data Lake Store Accounts:
  - `.../data-lake-store-accounts/add`
  - `.../data-lake-store-accounts/delete`
  - `.../data-lake-store-accounts/get`
  - `.../data-lake-store-accounts/list-by-account`
- Compute Policies:
  - `.../compute-policies/create-or-update`
  - `.../compute-policies/delete`
  - `.../compute-policies/get`
  - `.../compute-policies/list-by-account`
  - `.../compute-policies/update`
- Firewall Rules:
  - `.../firewall-rules/create-or-update`
  - `.../firewall-rules/delete`
  - `.../firewall-rules/get`
  - `.../firewall-rules/list-by-account`
  - `.../firewall-rules/update`
- Job:
  - `.../job/build`
  - `.../job/cancel`
  - `.../job/create`
  - `.../job/get`
  - `.../job/get-debug-data-path`
  - `.../job/get-statistics`
  - `.../job/list`
- Pipeline:
  - `.../pipeline/get`
  - `.../pipeline/list`
- Recurrence:
  - `.../recurrence/get`
  - `.../recurrence/list`
- Catalog (full operation-group list reviewed, including all linked methods):
  - `Create Credential`, `Create Secret`, `Delete All Secrets`, `Delete Credential`, `Delete Secret`
  - `Get Assembly`, `Get Credential`, `Get Database`, `Get External Data Source`, `Get Package`, `Get Procedure`, `Get Schema`, `Get Secret`, `Get Table`, `Get Table Partition`, `Get Table Statistic`, `Get Table Type`, `Get Table Valued Function`, `Get View`
  - `Grant Acl`, `Grant Acl To Database`
  - `List Acls`, `List Acls By Database`, `List Assemblies`, `List Credentials`, `List Databases`, `List External Data Sources`, `List Packages`, `List Procedures`, `List Schemas`, `List Table Fragments`, `List Table Partitions`, `List Tables`, `List Tables By Database`, `List Table Statistics`, `List Table Statistics By Database`, `List Table Statistics By Database And Schema`, `List Table Types`, `List Table Valued Functions`, `List Table Valued Functions By Database`, `List Types`, `List Views`, `List Views By Database`
  - `Preview Table`, `Preview Table Partition`
  - `Revoke Acl`, `Revoke Acl From Database`
  - `Update Credential`, `Update Secret`

## Service Naming

Canonical Stackyard naming:

- service key: `datalakeanalytics_2016_11_01`
- display name: `Azure Data Lake Analytics`
- API version: `2016-11-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_datalakeanalytics_2016_11_01.go`
- tests: `internal/server/provider_azure_datalakeanalytics_2016_11_01_test.go`
- example: `examples/azure/datalakeanalytics-2016-11-01`
- plan doc: `docs/datalakeanalytics-2016-11-01-plan.md`

## Route Envelope

Management plane (ARM) patterns:

- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DataLakeAnalytics/locations/{location}/checkNameAvailability?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeAnalytics/accounts/{accountName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeAnalytics/accounts/{accountName}/storageAccounts/{storageAccountName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeAnalytics/accounts/{accountName}/dataLakeStoreAccounts/{dataLakeStoreAccountName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeAnalytics/accounts/{accountName}/computePolicies/{computePolicyName}?api-version=2016-11-01`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DataLakeAnalytics/accounts/{accountName}/firewallRules/{firewallRuleName}?api-version=2016-11-01`
- `/azure/providers/Microsoft.DataLakeAnalytics/operations?api-version=2016-11-01`

Data plane (job/catalog/pipeline/recurrence) patterns from method pages:

- `{accountName}.{adlaJobDnsSuffix}/Jobs...`
- `{accountName}.{adlaJobDnsSuffix}/pipelines...`
- `{accountName}.{adlaJobDnsSuffix}/recurrences...`
- `{accountName}.{adlaCatalogDnsSuffix}/catalog/usql/...`

Stackyard convention should expose these through `/azure` with the same path suffixes after host normalization.

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- All reviewed methods use `api-version=2016-11-01`.
- `azure_auth` (AAD OAuth2) is used on both management and data-plane method pages.
- Top-level reference explicitly calls out token audience split:
  - management-plane requests use ARM audience/token
  - job/pipeline/recurrence/catalog require `datalake.azure.net` audience

Response/status characteristics to model:

- Management create/update methods are mostly `200`/`201`/`202`.
- Delete methods commonly return `200` and idempotent `204` for absent resources.
- List methods return paged envelopes with `nextLink`.
- Data-plane list/get/search-style methods return typed JSON payloads and optional OData query controls.
- Catalog surfaces include deprecated secret methods alongside credential methods and ACL grant/revoke operations.

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `datalakeanalytics_2016_11_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton file

Exit criteria:

- coverage scripts resolve the new service key and aliases
- doc coverage maps service key to this plan doc
- service is discoverable by `provider_azure_*.go` scans

### Stage 1: Router Ownership and Version Gate

Implement:

- ownership for Data Lake Analytics ARM and data-plane route families
- strict `api-version=2016-11-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned paths

Exit criteria:

- requests for DLA route envelopes are consistently owned by this provider
- invalid version behavior is stable across management/data-plane paths

### Stage 2: Core Account Lifecycle

Implement:

- `Accounts_CheckNameAvailability`
- `Accounts_Create`, `Accounts_Get`, `Accounts_List`, `Accounts_ListByResourceGroup`, `Accounts_Update`, `Accounts_Delete`

Behavior:

- stable account fixture shape (`id`, `name`, `type`, `location`, `properties`, `provisioningState`)
- deterministic transitions for create/update/delete
- paged list responses with stable sorting and optional continuation token

Exit criteria:

- account create/get/list/update/delete flows work end-to-end
- idempotent delete semantics modeled (`200`/`202`/`204` where applicable)

### Stage 3: Account Subresource Surfaces

Implement:

- Storage Accounts: all 8 methods
- Data Lake Store Accounts: all 4 methods
- Compute Policies: create/get/list/update/delete
- Firewall Rules: create/get/list/update/delete
- Operations: list available operations metadata

Behavior:

- subresource stores keyed by `{subscription, rg, account, subresource-name}`
- paged list responses and update/replace semantics per method docs
- SAS and container listing methods return deterministic fixtures

Exit criteria:

- all management-plane subresource methods return modeled responses
- no `NotImplemented` for the reviewed management-plane operation set

### Stage 4: Data-Plane Read Surfaces

Implement first-class data-plane read/query support for:

- Job: `Get`, `GetStatistics`, `GetDebugDataPath`, `List`
- Pipeline: `Get`, `List`
- Recurrence: `Get`, `List`
- Catalog read/list families (`Get*`, `List*`, `Preview*`, ACL list)

Behavior:

- deterministic identifiers and timestamps
- consistent pagination (`nextLink`) for list-heavy groups
- optional filter/top/skip/query options parsed where documented

Exit criteria:

- read/query operations across job/pipeline/recurrence/catalog return deterministic fixtures

### Stage 5: Data-Plane Mutation Surfaces

Implement mutable data-plane operations:

- Job: `Build`, `Create`, `Cancel`
- Catalog: create/update/delete credential/secret operations, grant/revoke ACL operations

Behavior:

- deterministic create/update idempotency rules
- deprecated secret methods still supported with explicit compatibility notes
- ACL grant/revoke effects reflected by subsequent ACL/list/get reads

Exit criteria:

- all reviewed data-plane mutation methods return modeled non-`NotImplemented` responses

### Stage 6: Tests

Add provider tests covering:

- API-version and method validation
- account lifecycle with subresources
- representative job lifecycle (`Build`/`Create`/`Get`/`Cancel`)
- catalog credential + ACL lifecycle
- pipeline/recurrence list/get behavior
- pagination and idempotent-delete semantics

Exit criteria:

- `go test ./internal/server -run 'TestAzureDataLakeAnalytics' -count=1` passes

### Stage 7: Example and Validation

Add `examples/azure/datalakeanalytics-2016-11-01`:

- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Validation commands:

- `go build ./examples/azure/datalakeanalytics-2016-11-01`
- `python3 scripts/azure-contract-coverage.py --service datalakeanalytics_2016_11_01 --require-service datalakeanalytics_2016_11_01 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service datalakeanalytics_2016_11_01 --require-service datalakeanalytics_2016_11_01 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service datalakeanalytics_2016_11_01 --require-service datalakeanalytics_2016_11_01 --fail-on any`

Success target:

- new service key is fully wired for plan/example/coverage discovery
- example builds and exercises representative Data Lake Analytics surfaces via Azure Go SDK pipeline calls
