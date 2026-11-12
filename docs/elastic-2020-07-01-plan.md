# Azure Elastic (`elastic_2020_07_01`) Staged Plan

## Objective

Emulate Azure Elastic Resource Manager REST API (`api-version=2020-07-01`) with deterministic local behavior for:

- monitor lifecycle and list surfaces
- deployment info and monitored-resource action surfaces
- monitor tag-rule lifecycle
- VM host and VM ingestion action surfaces
- VM collection update action surface
- provider operations discovery
- an Azure Go SDK example under `examples/azure/elastic-2020-07-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/elastic/operation-groups?view=rest-elastic-2020-07-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/elastic/deployment-info?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitored-resources?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/operations?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/tag-rules?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/vm-collection?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/vm-host?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/vm-ingestion?view=rest-elastic-2020-07-01`

Method pages reviewed directly: all linked methods from all operation groups (`16` total).

Deployment Info methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/deployment-info/list?view=rest-elastic-2020-07-01`

Monitored Resources methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/monitored-resources/list?view=rest-elastic-2020-07-01`

Monitors methods (`6`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors/create?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors/delete?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors/get?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors/list?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors/list-by-resource-group?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/monitors/update?view=rest-elastic-2020-07-01`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/operations/list?view=rest-elastic-2020-07-01`

Tag Rules methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/tag-rules/create-or-update?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/tag-rules/delete?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/tag-rules/get?view=rest-elastic-2020-07-01`
- `https://learn.microsoft.com/en-us/rest/api/elastic/tag-rules/list?view=rest-elastic-2020-07-01`

VM Collection methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/vm-collection/update?view=rest-elastic-2020-07-01`

VM Host methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/vm-host/list?view=rest-elastic-2020-07-01`

VM Ingestion methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/elastic/vm-ingestion/details?view=rest-elastic-2020-07-01`

SDK parity cross-check used:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elastic/armelastic`

## Surface Summary

Reviewed operation groups and methods:

- Deployment Info (1)
- Monitored Resources (1)
- Monitors (6)
- Operations (1)
- Tag Rules (4)
- VM Collection (1)
- VM Host (1)
- VM Ingestion (1)

Total reviewed methods: `16`.

## Service Naming

Canonical Stackyard naming:

- service key: `elastic_2020_07_01`
- display name: `Azure Elastic`
- API version: `2020-07-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_elastic_2020_07_01.go`
- tests: `internal/server/provider_azure_elastic_2020_07_01_test.go`
- example: `examples/azure/elastic-2020-07-01`
- plan doc: `docs/elastic-2020-07-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.Elastic/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Elastic/monitors`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listDeploymentInfo`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listMonitoredResources`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules/{ruleSetName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/vmCollectionUpdate`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listVMHost`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/vmIngestionDetails`

Representative routes:

- `GET /azure/providers/Microsoft.Elastic/operations?api-version=2020-07-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}?api-version=2020-07-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}?api-version=2020-07-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listDeploymentInfo?api-version=2020-07-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listMonitoredResources?api-version=2020-07-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/tagRules/{ruleSetName}?api-version=2020-07-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/vmCollectionUpdate?api-version=2020-07-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/listVMHost?api-version=2020-07-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Elastic/monitors/{monitorName}/vmIngestionDetails?api-version=2020-07-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2020-07-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods document non-success paths as `Other Status Codes | ResourceProviderDefaultErrorResponse`

Observed response patterns:

- `200` appears in `16` methods
- `201` appears in `1` method
- `202` appears in `2` methods

Request-body coverage from reviewed pages:

- `4` methods include request bodies
- body-bearing methods:
  - `Monitors_Create`
  - `Monitors_Update`
  - `TagRules_CreateOrUpdate`
  - `VMCollection_Update`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add stage-0 provider skeleton at `internal/server/provider_azure_elastic_2020_07_01.go`
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key resolves as `elastic_2020_07_01`
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.Elastic` route families for this API version
- enforce `api-version=2020-07-01` for handled routes
- preserve behavior for unrelated Azure providers

### Stage 2: Read/List and Action Surfaces

- implement `Operations_List`
- implement monitor get/list/list-by-resource-group
- implement deployment-info, monitored-resources, vm-host, and vm-ingestion action routes
- implement tag-rule get/list routes

### Stage 3: Mutating Surfaces

- implement `Monitors_Create`, `Monitors_Update`, `Monitors_Delete`
- implement `TagRules_CreateOrUpdate`, `TagRules_Delete`
- implement `VMCollection_Update`
- support deterministic documented `200`/`201`/`202` response status behavior

### Stage 4: Payload and Query Validation

- validate required payload elements for body-bearing methods:
  - monitor create (`location` and optional resource fields)
  - monitor update (`tags`)
  - tag-rule properties/log-rule structure
  - VM collection update (`operationName`, `vmResourceId`)
- validate required path identifiers (`monitorName`, `ruleSetName`) and api-version consistency

### Stage 5: Tests

- add provider tests `internal/server/provider_azure_elastic_2020_07_01_test.go` covering:
  - route ownership and API-version validation
  - monitor lifecycle and list behavior
  - deployment-info/monitored-resources/vm-host/vm-ingestion action routes
  - tag-rule lifecycle behavior
  - VM collection update action behavior
  - deterministic error envelopes for contract violations
