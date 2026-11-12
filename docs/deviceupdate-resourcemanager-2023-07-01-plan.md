# Azure Device Update Resource Manager (`deviceupdate_resourcemanager_2023_07_01`) Staged Plan

## Objective

Emulate Azure Device Update Resource Manager REST API (`api-version=2023-07-01`) with deterministic local behavior for:

- account lifecycle and account discovery surfaces
- instance lifecycle and account-scoped instance enumeration
- provider operation discovery and name-availability checks
- private endpoint connection and private endpoint connection proxy workflows
- private link resource discovery
- an Azure Go SDK example under `examples/azure/deviceupdate-resourcemanager-2023-07-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/operation-groups?view=rest-deviceupdate-resourcemanager-2023-07-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/check-name-availability?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/operations?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connections?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-link-resources?view=rest-deviceupdate-resourcemanager-2023-07-01`

Method pages reviewed directly: all linked methods from all operation groups (`27` total).

Accounts method pages (`7`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/create?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/delete?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/get?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/head?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/list-by-resource-group?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/list-by-subscription?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/update?view=rest-deviceupdate-resourcemanager-2023-07-01`

Check Name Availability method page (`1`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/check-name-availability/check-name-availability?view=rest-deviceupdate-resourcemanager-2023-07-01`

Instances method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances/create?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances/delete?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances/get?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances/head?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances/list-by-account?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/instances/update?view=rest-deviceupdate-resourcemanager-2023-07-01`

Operations method page (`1`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/operations/list?view=rest-deviceupdate-resourcemanager-2023-07-01`

Private Endpoint Connection Proxies method pages (`6`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies/create-or-update?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies/delete?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies/get?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies/list-by-account?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies/update-private-endpoint-properties?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connection-proxies/validate?view=rest-deviceupdate-resourcemanager-2023-07-01`

Private Endpoint Connections method pages (`4`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connections/create-or-update?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connections/delete?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connections/get?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-endpoint-connections/list-by-account?view=rest-deviceupdate-resourcemanager-2023-07-01`

Private Link Resources method pages (`2`):

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-link-resources/get?view=rest-deviceupdate-resourcemanager-2023-07-01`
- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/private-link-resources/list-by-account?view=rest-deviceupdate-resourcemanager-2023-07-01`

SDK parity cross-check used:

- `https://learn.microsoft.com/en-us/rest/api/deviceupdate/resourcemanager/accounts/create?view=rest-deviceupdate-resourcemanager-2023-07-01` (Go tab references `armdeviceupdate`)
- `https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/deviceupdate/armdeviceupdate`

## Surface Summary

Reviewed operation groups and methods:

- Accounts (7)
- Check Name Availability (1)
- Instances (6)
- Operations (1)
- Private Endpoint Connection Proxies (6)
- Private Endpoint Connections (4)
- Private Link Resources (2)

Total reviewed methods: `27`.

## Service Naming

Canonical Stackyard naming:

- service key: `deviceupdate_resourcemanager_2023_07_01`
- display name: `Azure Device Update Resource Manager`
- API version: `2023-07-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_deviceupdate_resourcemanager_2023_07_01.go`
- tests: `internal/server/provider_azure_deviceupdate_resourcemanager_2023_07_01_test.go`
- example: `examples/azure/deviceupdate-resourcemanager-2023-07-01`
- plan doc: `docs/deviceupdate-resourcemanager-2023-07-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DeviceUpdate/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DeviceUpdate/checknameavailability`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DeviceUpdate/accounts`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceUpdate/accounts`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceUpdate/accounts/{accountName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/instances`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/privateEndpointConnectionProxies`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/privateEndpointConnections`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/privateLinkResources`

Representative routes:

- `GET /azure/providers/Microsoft.DeviceUpdate/operations?api-version=2023-07-01`
- `POST /azure/subscriptions/{sub}/providers/Microsoft.DeviceUpdate/checknameavailability?api-version=2023-07-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DeviceUpdate/accounts/{accountName}?api-version=2023-07-01`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DeviceUpdate/accounts/{accountName}?api-version=2023-07-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/instances/{instanceName}?api-version=2023-07-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2023-07-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/privateEndpointConnectionProxies/{privateEndpointConnectionProxyId}?api-version=2023-07-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DeviceUpdate/accounts/{accountName}/privateEndpointConnectionProxies/{privateEndpointConnectionProxyId}/validate?api-version=2023-07-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2023-07-01`
- auth is ARM OAuth2 (`azure_auth`, `user_impersonation` scope)
- non-success responses are documented as `Other Status Codes | ErrorResponse`
- list methods return typed list envelopes (`*List`/`*ListResult`) and are pager-oriented in SDK examples

Observed response patterns:

- dominant success status is `200 OK` for get/list/post action routes
- create/update-style operations often return `201 Created` for async acceptance:
  - `accounts/create`
  - `accounts/update` (also documents `200 OK`)
  - `instances/create`
  - `private-endpoint-connection-proxies/create-or-update`
  - `private-endpoint-connections/create-or-update`
- delete methods for account/instance/private-endpoint resources document:
  - `200 OK` for synchronous completion
  - `202 Accepted` with `Location` header for async completion
  - `204 No Content` for already-absent resource
- HEAD existence checks (`accounts/head`, `instances/head`) document `200 OK` on existence

Selected request-body requirements and constraints:

- `accounts/create`: `location` required
- `instances/create`: `location` required
- `private-endpoint-connections/create-or-update`: `properties.privateLinkServiceConnectionState` required
- `check-name-availability/check-name-availability`: body fields `name` and `type`
- private endpoint proxy action methods require remote/private-endpoint identification fields (`id`, `immutableResourceId`, `immutableSubscriptionId`, `location`) per documented schemas

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `deviceupdate_resourcemanager_2023_07_01` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring
- add Azure Go SDK example scaffold under `examples/azure/deviceupdate-resourcemanager-2023-07-01`

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Device Update management paths to provider stub
- example compiles and covers all reviewed methods

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for all `Microsoft.DeviceUpdate` management-plane families listed above
- strict `api-version=2023-07-01` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Device Update Resource Manager route families are deterministically owned
- version and method failures are stable and testable

### Stage 2: Provider Operations and Account Surfaces

Implement:

- `Operations_List`
- `CheckNameAvailability_CheckNameAvailability`
- all Accounts methods (`Create`, `Delete`, `Get`, `Head`, `ListByResourceGroup`, `ListBySubscription`, `Update`)

Behavior:

- deterministic account persistence keyed by `{subscriptionId, resourceGroupName, accountName}`
- deterministic ARM resource envelopes (`id`, `name`, `type`, `location`, `tags`, `properties`, `systemData`)
- idempotent account delete semantics with documented `200/202/204`

Exit criteria:

- all provider operations/name/account methods return modeled responses
- no `NotImplemented` for these surfaces

### Stage 3: Instance Surfaces

Implement all Instances methods:

- `Create`, `Delete`, `Get`, `Head`, `ListByAccount`, `Update`

Behavior:

- deterministic instance persistence under account scope
- deterministic linkage to IoT hub settings and diagnostics settings
- idempotent delete semantics with documented `200/202/204`

Exit criteria:

- instance lifecycle routes are deterministic and contract-testable

### Stage 4: Private Endpoint and Private Link Surfaces

Implement:

- Private Endpoint Connection Proxies:
  - `CreateOrUpdate`, `Delete`, `Get`, `ListByAccount`, `UpdatePrivateEndpointProperties`, `Validate`
- Private Endpoint Connections:
  - `CreateOrUpdate`, `Delete`, `Get`, `ListByAccount`
- Private Link Resources:
  - `Get`, `ListByAccount`

Behavior:

- deterministic connection/proxy state and approval metadata
- deterministic async-delete response shape with `Location` header where documented
- deterministic private-link resource discovery responses by account scope

Exit criteria:

- all private endpoint/private link methods return modeled responses
- no `NotImplemented` for private connectivity surfaces

### Stage 5: Tests

Add provider tests for:

- api-version and method gating across all owned routes
- account and instance lifecycle behavior
- list scope behavior (subscription vs resource-group vs account)
- private endpoint/proxy create/get/list/update/validate/delete flows
- deterministic delete response envelopes and headers
- deterministic error envelopes for invalid input and not-found resources

Exit criteria:

- `go test ./internal/server -run 'TestAzureDeviceUpdateResourceManager20230701' -count=1` passes

### Stage 6: Example and Validation

Maintain `examples/azure/deviceupdate-resourcemanager-2023-07-01` with representative calls for all 27 reviewed methods using Azure Go SDK core runtime pipeline.

Validation:

- run the example against local Stackyard Azure endpoint
- verify route ownership, status codes, and payload envelopes align with reviewed method docs

Exit criteria:

- example executes end-to-end against emulated routes once Stage 1+ handlers are implemented
- coverage scripts and plan mapping include the new service key
