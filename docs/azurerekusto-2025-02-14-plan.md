# Azure Kusto (`azurerekusto-2025-02-14`) Staged Plan

## Objective

Emulate the Azure Kusto ARM management API for `2025-02-14` with deterministic local behavior for:

- provider discovery and long-running operation result polling
- top-level `clusters` lifecycle plus cluster action routes
- cluster child resources such as `databases`, `attachedDatabaseConfigurations`, `dataConnections`, `scripts`, `managedPrivateEndpoints`, `privateEndpointConnections`, `privateLinkResources`, and `sandboxCustomImages`
- cluster and database principal-assignment resources
- region and subscription SKU discovery
- an Azure Go SDK example under `examples/azure/azurerekusto-2025-02-14`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-azurerekusto-2025-02-14` is the ARM management plane for `Microsoft.Kusto`.
- This is not the Kusto query/data-plane endpoint family used for command/query execution.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2025-02-14`, resource-group scoping, and ARM long-running-operation semantics where documented.

## Primary References Reviewed

Primary operation-groups page reviewed:
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operation-groups?view=rest-azurerekusto-2025-02-14

Representative operation-group pages reviewed across the surface:
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operations?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/clusters?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/database?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/databases?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/attached-database-configurations?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/data-connections?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/cluster-principal-assignments?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/database-principal-assignments?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/managed-private-endpoints?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/private-endpoint-connections?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/private-link-resources?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/sandbox-custom-images?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/scripts?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/skus?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operations-results?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operations-results-location?view=rest-azurerekusto-2025-02-14

Representative linked method pages reviewed:
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operations/list?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/clusters/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/clusters/update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/clusters/list-skus?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/databases/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/databases/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/database/invite-follower?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/attached-database-configurations/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/attached-database-configurations/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/data-connections/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/data-connections/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/data-connections/update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/database-principal-assignments/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/managed-private-endpoints/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/managed-private-endpoints/update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/private-link-resources/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/private-link-resources/list?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/sandbox-custom-images/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/sandbox-custom-images/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/scripts/create-or-update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/scripts/update?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/skus/list?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operations-results/get?view=rest-azurerekusto-2025-02-14
- https://learn.microsoft.com/en-us/rest/api/azurerekusto/operations-results-location/get?view=rest-azurerekusto-2025-02-14

SDK package identified for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/kusto/armkusto`

## Service Naming

Canonical Stackyard naming:
- service key: `kusto`
- display name: `Azure Kusto`
- API version token: `2025-02-14`

Recommended repo paths:
- implementation: `internal/server/provider_azure_kusto.go`
- tests: `internal/server/provider_azure_kusto_test.go`
- example: `examples/azure/azurerekusto-2025-02-14`
- plan doc: `docs/azurerekusto-2025-02-14-plan.md`

Recommended coverage aliases:
- `kusto`
- `azure_kusto`
- `azurerekusto`
- `azure_data_explorer`
- `adx`
- `microsoft_kusto`
- `azurerekusto_2025_02_14`
- `rest_azurerekusto_2025_02_14`

## Surface Summary

The reviewed `2025-02-14` surface exposes 16 operation groups:

1. `Attached Database Configurations`
2. `Cluster Principal Assignments`
3. `Clusters`
4. `Data Connections`
5. `Database`
6. `Database Principal Assignments`
7. `Databases`
8. `Managed Private Endpoints`
9. `Operations`
10. `Operations Results`
11. `Operations Results Location`
12. `Private Endpoint Connections`
13. `Private Link Resources`
14. `Sandbox Custom Images`
15. `Scripts`
16. `Skus`

Broad route families identified from the reviewed method pages:

- provider discovery:
  - `/providers/Microsoft.Kusto/operations`
- subscription-scoped SKU and async status routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Kusto/skus`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Kusto/locations/{location}/skus`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Kusto/locations/{location}/operationResults/{operationId}`
- core ARM resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}`
- cluster child collections:
  - `principalAssignments`
  - `databases`
  - `attachedDatabaseConfigurations`
  - `managedPrivateEndpoints`
  - `privateEndpointConnections`
  - `privateLinkResources`
  - `sandboxCustomImages`
- database child collections:
  - `dataConnections`
  - `scripts`
  - `principalAssignments`
- action routes:
  - cluster actions such as `start`, `stop`, `migrate`, `diagnoseVirtualNetwork`, `addLanguageExtensions`, `removeLanguageExtensions`, `addCalloutPolicies`, `removeCalloutPolicy`, and `detachFollowerDatabases`
  - database action `inviteFollower`
  - database principal mutation routes `addPrincipals` and `removePrincipals`
  - data connection validation route

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host (`management.azure.com`)
- all reviewed routes require `api-version=2025-02-14`
- cluster and nested child resources use `Microsoft.Kusto`
- many create/update/delete methods are documented as ARM LROs using `Azure-AsyncOperation` and/or `Location`

Representative ARM routes:
- `GET /azure/providers/Microsoft.Kusto/operations?api-version=2025-02-14`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}?api-version=2025-02-14`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}?api-version=2025-02-14`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}/start?api-version=2025-02-14`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}/databases/{databaseName}?api-version=2025-02-14`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}/databases/{databaseName}/inviteFollower?api-version=2025-02-14`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}/databases/{databaseName}/dataConnections/{dataConnectionName}?api-version=2025-02-14`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Kusto/clusters/{clusterName}/databases/{databaseName}/scripts/{scriptName}?api-version=2025-02-14`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.Kusto/locations/{location}/operationResults/{operationId}?operationResultResponseType=Location&api-version=2025-02-14`

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- ARM-style `ErrorResponse` is used for non-success responses.
- Many mutating routes return `200`, `201`, or `202` and carry async headers.
- Standard ARM resource fields are present across most resource types:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.provisioningState`
- List routes consistently return `value`.
- Nested child resources generally do not require separate top-level listing beyond their parent scope.

### Operations and Async Polling

Reviewed method pages:
- `Operations - List`
- `Operations Results - Get`
- `Operations Results Location - Get`

Modeling implications:
- the provider should expose a static `operations` payload first
- LRO-capable routes should register synthetic operation results in server state
- support both result polling forms:
  - JSON `OperationResult`
  - empty/202 `Location` style polling variant

### Clusters

Reviewed method pages:
- `Clusters - Create Or Update`
- `Clusters - Update`
- `Clusters - Get`
- `Clusters - List`
- `Clusters - List By Resource Group`
- `Clusters - List Skus`

Notable request/response traits:
- supports standard ARM headers `If-Match` and `If-None-Match`
- cluster payloads include ARM identity, SKU, zones, and rich `properties`
- action methods imply operational state changes such as `Running` and `Stopped`

Modeling implications:
- the first implementation slice should treat `clusters` as the anchor resource
- persist cluster state, SKU, identity, tags, and a compact but typed subset of properties
- expose deterministic action responses for `start`, `stop`, `migrate`, and diagnostics instead of `NotImplemented`

### Databases

Reviewed method pages:
- `Databases - Create Or Update`
- `Databases - Get`
- `Databases - List By Cluster`
- `Database - Invite Follower`

Notable traits:
- database `kind` varies at least between `ReadWrite` and `ReadOnlyFollowing`
- principal mutation routes are action-style rather than child CRUD
- `inviteFollower` returns a generated invitation token payload

Modeling implications:
- keep a typed `kind` field in state even if the first pass only seeds one or two variants
- model principal add/remove/list as deterministic collection transforms on a database record
- implement `inviteFollower` as an action response with stable synthetic token output

### Principal Assignments

Reviewed method pages:
- `Cluster Principal Assignments - Create Or Update`
- `Cluster Principal Assignments - Get/List`
- `Database Principal Assignments - Create Or Update`
- `Database Principal Assignments - Get/List`

Notable traits:
- both cluster and database principal assignments are normal nested ARM child resources
- request bodies include explicit `principalId`, `principalType`, `role`, and `tenantId`

Modeling implications:
- use a shared nested-child resource helper with separate stores for cluster- and database-scoped assignments
- keep role and principal-type enums stable in fixtures so examples can assert on them cleanly

### Data Connections

Reviewed method pages:
- `Data Connections - Create Or Update`
- `Data Connections - Get`
- `Data Connections - Update`

Notable traits:
- union-shaped resource kinds include at least:
  - `EventHub`
  - `EventHubWithManagedIdentity`
  - `IotHub`
  - `EventGrid`
  - `EventGridWithManagedIdentity`
  - `CosmosDb`
- validation is a dedicated action route

Modeling implications:
- implement a typed `kind` discriminator first, with one canonical fixture per major kind family
- support validation as a deterministic success/error action instead of a no-op

### Networking and Security

Reviewed method pages:
- `Managed Private Endpoints - Get/Update/List`
- `Private Endpoint Connections - Get/List`
- `Private Link Resources - Get/List`

Modeling implications:
- split writable managed private endpoints from mostly descriptive private-link resources
- return fixed `requiredMembers` and `requiredZoneNames` for private-link resources
- represent private endpoint connection approval state explicitly in properties

### Scripts and Sandbox Images

Reviewed method pages:
- `Scripts - Create Or Update`
- `Scripts - Update`
- `Sandbox Custom Images - Create Or Update`
- `Sandbox Custom Images - Get/Update`

Modeling implications:
- scripts should carry script text, URL, force-update tag, and continuation behavior in a compact typed subset
- sandbox custom images can be handled as conventional cluster child resources with a stable image URL/tag payload

## SDK Note

The official Azure Go SDK package for this management surface appears to be:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/kusto/armkusto`

That package already exposes client families matching the reviewed operation groups, including:
- `ClustersClient`
- `DatabasesClient`
- `DataConnectionsClient`
- `AttachedDatabaseConfigurationsClient`
- `ClusterPrincipalAssignmentsClient`
- `DatabasePrincipalAssignmentsClient`
- `ManagedPrivateEndpointsClient`

Example implication:
- the future example should prefer `armkusto.NewClientFactory(...)`
- the example should demonstrate at least one top-level cluster operation and one nested database/data-connection operation

## Planned Repo Shape

Implementation files to add:
- `internal/server/provider_azure_kusto.go`
- `internal/server/provider_azure_kusto_test.go`

Example files to add:
- `examples/azure/azurerekusto-2025-02-14/main.go`
- `examples/azure/azurerekusto-2025-02-14/Dockerfile`
- `examples/azure/azurerekusto-2025-02-14/docker-compose.yml`

Coverage mappings to update:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Staged Implementation Plan

### Stage 0: Claim The Service Cleanly

- add `kusto` service aliases to all three Azure coverage scripts
- map `kusto` to this plan doc in `azure-doc-contract-coverage.py`
- wire a new router entry in `internal/server/http.go`
- add server state maps for:
  - clusters
  - databases
  - cluster principal assignments
  - database principal assignments
  - attached database configurations
  - data connections
  - scripts
  - managed private endpoints
  - private endpoint connections
  - sandbox custom images
  - synthetic operation results

Deliverable:
- coverage scripts discover `kusto`
- router claims the service without falling through to `respondAzureImplemented`

### Stage 1: Provider Discovery and Shared ARM/LRO Helpers

- implement:
  - `GET /providers/Microsoft.Kusto/operations`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Kusto/skus`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Kusto/locations/{location}/skus`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Kusto/locations/{location}/operationResults/{operationId}`
  - `GET ...operationResults/{operationId}?operationResultResponseType=Location`
- add shared helpers for:
  - ARM ID construction
  - `systemData`
  - `ETag`
  - synthetic LRO registration and polling

Deliverable:
- the service has a strong ARM foundation before nested resources are added

### Stage 2: Core Clusters

- implement cluster CRUD/list/head routes:
  - create/update/get/delete/list/list-by-resource-group
  - `checkNameAvailability`
  - `listSkus` and `listSkusByResource`
- persist:
  - `location`
  - `sku`
  - `identity`
  - `tags`
  - a stable subset of cluster properties

Deliverable:
- cluster-centric examples and coverage can pass without `NotImplemented`

### Stage 3: Cluster Actions

- implement deterministic action handlers for:
  - `start`
  - `stop`
  - `migrate`
  - `diagnoseVirtualNetwork`
  - `addLanguageExtensions`
  - `removeLanguageExtensions`
  - `listLanguageExtensions`
  - `addCalloutPolicies`
  - `removeCalloutPolicy`
  - `listCalloutPolicies`
  - `listOutboundNetworkDependenciesEndpoints`
  - `detachFollowerDatabases`
  - `listFollowerDatabases`

Deliverable:
- the high-churn action endpoints stop being the major `NotImplemented` risk

### Stage 4: Databases and Database Actions

- implement:
  - database CRUD/list/check-name routes
  - `addPrincipals`
  - `removePrincipals`
  - `listPrincipals`
  - `inviteFollower`
- model at least:
  - `ReadWriteDatabase`
  - `ReadOnlyFollowingDatabase`

Deliverable:
- nested database coverage is solid enough to support scripts and data connections

### Stage 5: Principal Assignments

- implement cluster principal assignment CRUD/list/check-name
- implement database principal assignment CRUD/list/check-name

Deliverable:
- RBAC-style child resources are covered with shared nested-resource helpers

### Stage 6: Data Connections

- implement data connection CRUD/list/check-name/update
- implement validation action route
- support representative `kind` fixtures for:
  - `EventHub`
  - `EventGrid`
  - `CosmosDb`

Deliverable:
- one of the largest polymorphic sub-surfaces is claimed cleanly

### Stage 7: Scripts and Attached Database Configurations

- implement scripts CRUD/list/check-name/update
- implement attached database configuration CRUD/list/check-name

Deliverable:
- database-adjacent child resources are covered with realistic typed outputs

### Stage 8: Networking

- implement managed private endpoints CRUD/list/check-name/update
- implement private endpoint connections get/list/create-or-update/delete
- implement private link resources get/list

Deliverable:
- network and private-access coverage no longer falls through to `NotImplemented`

### Stage 9: Sandbox Images and Remaining Read Tails

- implement sandbox custom images CRUD/list/check-name/update
- sweep any remaining small read-only tails from the reviewed groups

Deliverable:
- all reviewed operation groups are claimed by some handler

### Stage 10: Example and End-to-End Validation

- add Azure Go SDK example under `examples/azure/azurerekusto-2025-02-14`
- use `armkusto` client factory with:
  - one cluster operation
  - one database operation
  - one nested child operation such as data connection or script
- run:
  - `go build ./examples/azure/azurerekusto-2025-02-14`
  - targeted provider tests
  - Azure contract/io/doc coverage scripts for `kusto`

Deliverable:
- example, tests, and coverage all agree on the same service shape

## Recommended First Merge Slice

The highest-value first implementation slice is:

1. Stage 0
2. Stage 1
3. Stage 2
4. a minimal subset of Stage 4

Reason:
- that combination claims the provider, its ARM/LRO foundation, and the central `clusters` plus `databases` hierarchy
- most remaining groups hang off those resources and can then be added incrementally without changing the overall route strategy

## Non-Goals For The First Implementation

- full fidelity for every Kusto union subtype on day one
- exact Azure capacity or SKU logic
- real network diagnostics
- real migration execution
- real invitation-token cryptography
- full parity for every advanced cluster property in the first pass
