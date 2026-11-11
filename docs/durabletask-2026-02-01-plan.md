# Azure Durable Task Scheduler (`durabletask_2026_02_01`) Staged Plan

## Objective

Emulate Azure Durable Task Scheduler Resource Manager REST API (`api-version=2026-02-01`) with deterministic local behavior for:

- scheduler lifecycle and resource listings
- scheduler private endpoint connection lifecycle and private link resource listings
- task hub lifecycle per scheduler
- retention policy lifecycle per scheduler
- operations discovery endpoint
- an Azure Go SDK example under `examples/azure/durabletask-2026-02-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/durabletask/operation-groups?view=rest-durabletask-2026-02-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/durabletask/operations?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/retention-policies?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/task-hubs?view=rest-durabletask-2026-02-01`

Method pages reviewed directly: all linked methods from all operation groups (`23` total).

operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/durabletask/operations/list?view=rest-durabletask-2026-02-01`

retention-policies methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/durabletask/retention-policies/create-or-replace?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/retention-policies/delete?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/retention-policies/get?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/retention-policies/list-by-scheduler?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/retention-policies/update?view=rest-durabletask-2026-02-01`

schedulers methods (`13`):

- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/create-or-update?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/create-or-update-private-endpoint-connection?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/delete?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/delete-private-endpoint-connection?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/get?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/get-private-endpoint-connection?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/get-private-link?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/list-by-resource-group?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/list-by-subscription?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/list-private-endpoint-connections?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/list-private-links?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/update?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/schedulers/update-private-endpoint-connection?view=rest-durabletask-2026-02-01`

task-hubs methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/durabletask/task-hubs/create-or-update?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/task-hubs/delete?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/task-hubs/get?view=rest-durabletask-2026-02-01`
- `https://learn.microsoft.com/en-us/rest/api/durabletask/task-hubs/list-by-scheduler?view=rest-durabletask-2026-02-01`

## Surface Summary

Reviewed operation groups and methods:

- operations (1)
- retention-policies (5)
- schedulers (13)
- task-hubs (4)

Total reviewed methods: `23`.

## Service Naming

Canonical Stackyard naming:

- service key: `durabletask_2026_02_01`
- display name: `Azure Durable Task Scheduler`
- API version: `2026-02-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_durabletask_2026_02_01.go`
- tests: `internal/server/provider_azure_durabletask_2026_02_01_test.go`
- example: `examples/azure/durabletask-2026-02-01`
- plan doc: `docs/durabletask-2026-02-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DurableTask/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DurableTask/schedulers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateLinkResources`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies`

Representative routes:

- `GET /azure/providers/Microsoft.DurableTask/operations?api-version=2026-02-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies?api-version=2026-02-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/retentionPolicies/default?api-version=2026-02-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateLinkResources/{privateLinkResourceName}?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.DurableTask/schedulers?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateLinkResources?api-version=2026-02-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}?api-version=2026-02-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2026-02-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs/{taskHubName}?api-version=2026-02-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs/{taskHubName}?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs/{taskHubName}?api-version=2026-02-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DurableTask/schedulers/{schedulerName}/taskHubs?api-version=2026-02-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2026-02-01`
- auth model is ARM OAuth2 (`azure_auth`, `user_impersonation`)
- non-success responses are documented as `Other Status Codes | ErrorResponse`
- several create/update/delete methods include asynchronous patterns (`202` and async headers on some methods)

Observed response patterns:

- `200` appears in `19` methods
- `201` appears in `4` methods
- `202` appears in `7` methods
- `204` appears in `4` methods

Request-body coverage from reviewed pages:

- `7` methods include a request body
- body-bearing methods include scheduler create/update, private endpoint connection create/update, task hub create/update, and retention policy create/update

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add `docs/durabletask-2026-02-01-plan.md` with operation-group/method inventory
- add `internal/server/provider_azure_durabletask_2026_02_01.go` stage-0 router skeleton
- wire router in `internal/server/http.go`
- add coverage aliases and docs mapping in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- service key is discoverable as `durabletask_2026_02_01` in coverage tooling
- doc coverage resolves mapped plan doc

### Stage 1: Router Ownership and API-Version Gate

- claim `Microsoft.DurableTask` route families for this API version
- enforce `api-version=2026-02-01` for handled routes
- preserve behavior for unrelated providers

### Stage 2: Scheduler Surface

- implement `operations/list` for durabletask provider metadata
- implement scheduler CRUD/list operations
- support private link resource listings (`list-private-links`, `get-private-link`)

### Stage 3: Private Endpoint Connection Surface

- implement private endpoint connection create/get/list/update/delete under schedulers
- preserve connection-state payload fields (`status`, `description`, `actionsRequired`)
- support synchronous and asynchronous status patterns where documented

### Stage 4: Task Hub + Retention Policy Surfaces

- implement task hub create/get/list/delete methods
- implement retention policy create/get/list/update/delete methods
- include deterministic retention policy structures (`retentionPeriodInDays`, `orchestrationState`)

### Stage 5: Tests

- add provider tests (`internal/server/provider_azure_durabletask_2026_02_01_test.go`) covering:
  - api-version validation
  - positive lifecycle flows for schedulers, task hubs, private endpoint connections, and retention policies
  - negative paths (missing params, malformed names, unknown resources)
  - async response/header assertions for representative `202` endpoints

### Stage 6: Example and Validation

- add `examples/azure/durabletask-2026-02-01` using Azure Go SDK core pipeline
- include one call per reviewed method with expected status assertions
- verify build and coverage script discoverability

Exit criteria:

- example compiles and runs against emulated routes once handlers are implemented
- coverage scripts include `durabletask_2026_02_01` and mapped plan doc

## Example Scope for `examples/azure/durabletask-2026-02-01`

The example should exercise all reviewed methods (`23` total) with environment-configurable subscription, resource group, scheduler, task-hub, private endpoint connection, private link resource, and retention-policy identifiers.
