# Azure App Testing Load Testing (`loadtesting-resourcemanager-2024-12-01-preview`) Staged Plan

## Objective

Emulate the Azure App Testing Load Testing ARM management API for `2024-12-01-preview` with deterministic local behavior for:

- provider `operations`
- top-level `loadTests` lifecycle and discovery routes
- resource-scoped `loadTestMappings`
- resource-scoped `loadTestProfileMappings`
- `loadTests/{loadTestName}/outboundNetworkDependenciesEndpoints`
- `loadTests/{loadTestName}/limits/maxMonthlyVirtualUserHours` get and set routes
- an Azure Go SDK example under `examples/azure/loadtesting-resourcemanager-2024-12-01-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for `Microsoft.LoadTestService`.
- This is not the Load Testing data-plane API used to upload tests, create test runs, or fetch metrics from the service endpoint itself.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2024-12-01-preview`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/operation-groups?view=rest-loadtesting-resourcemanager-2024-12-01-preview`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-mappings?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-profile-mappings?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-tests-max-monthly-virtual-user-hours-limits-resource/get?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-tests-max-monthly-virtual-user-hours-limits-resource/set?view=rest-loadtesting-resourcemanager-2024-12-01-preview`

Representative linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/create-or-update?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/get?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/list-by-resource-group?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/list-by-subscription?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/update?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/delete?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-resource/list-outbound-network-dependencies-endpoints?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-mappings/create-or-update?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-mappings/get?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-mappings/list?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-mappings/update?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-mappings/delete?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-profile-mappings/create-or-update?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-profile-mappings/get?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-profile-mappings/list?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-test-profile-mappings/delete?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-tests-max-monthly-virtual-user-hours-limits-resource/get?view=rest-loadtesting-resourcemanager-2024-12-01-preview`
- `https://learn.microsoft.com/en-us/rest/api/loadtesting/resourcemanager/load-tests-max-monthly-virtual-user-hours-limits-resource/set?view=rest-loadtesting-resourcemanager-2024-12-01-preview`

SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/loadtesting/armloadtesting`

If the preview surface is ahead of the currently released typed client:
- keep the example typed where possible
- use `azcore/arm` plus raw runtime requests only for preview-only gaps

## Service Naming

Canonical Stackyard naming:
- service key: `loadtesting`
- display name: `Azure App Testing Load Testing`
- provider namespace: `Microsoft.LoadTestService`
- API version: `2024-12-01-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_loadtesting.go`
- tests: `internal/server/provider_azure_loadtesting_test.go`
- example: `examples/azure/loadtesting-resourcemanager-2024-12-01-preview`
- plan doc: `docs/loadtesting-resourcemanager-2024-12-01-preview-plan.md`

Recommended coverage aliases:
- `loadtesting`
- `load_testing`
- `load-testing`
- `azure_loadtesting`
- `azure_load_testing`
- `app_testing_load_testing`
- `microsoft_loadtestservice`
- `loadtesting_resourcemanager_2024_12_01_preview`
- `rest_loadtesting_resourcemanager_2024_12_01_preview`

## Surface Summary

The reviewed `2024-12-01-preview` management surface exposes 5 operation groups:

1. `Operations`
2. `Load Test Resource`
3. `Load Test Mappings`
4. `Load Test Profile Mappings`
5. `Load Tests Max Monthly Virtual User Hours Limits Resource`

Broad route families identified from the reviewed pages:

- provider discovery:
  - `/providers/Microsoft.LoadTestService/operations`
- top-level ARM resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}`
- subscription and resource-group listing:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/loadTests`
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests`
- load-test child routes:
  - `/outboundNetworkDependenciesEndpoints`
  - `/limits/maxMonthlyVirtualUserHours`
  - `/limits/maxMonthlyVirtualUserHours/set`
- arbitrary parent-resource child mappings:
  - `/{resourceUri}/providers/Microsoft.LoadTestService/loadTestMappings`
  - `/{resourceUri}/providers/Microsoft.LoadTestService/loadTestProfileMappings`

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2024-12-01-preview`
- top-level resources use `Microsoft.LoadTestService`
- mapping resources are parented by an arbitrary ARM `resourceUri`, not by the load test resource itself

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.LoadTestService/operations?api-version=2024-12-01-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}?api-version=2024-12-01-preview`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}?api-version=2024-12-01-preview`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/loadTests?api-version=2024-12-01-preview`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}/outboundNetworkDependenciesEndpoints?api-version=2024-12-01-preview`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}/limits/maxMonthlyVirtualUserHours?api-version=2024-12-01-preview`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}/limits/maxMonthlyVirtualUserHours/set?api-version=2024-12-01-preview`
- `PUT /azure/{resourceUri}/providers/Microsoft.LoadTestService/loadTestMappings/{loadTestMappingName}?api-version=2024-12-01-preview`
- `PATCH /azure/{resourceUri}/providers/Microsoft.LoadTestService/loadTestMappings/{loadTestMappingName}?api-version=2024-12-01-preview`
- `PUT /azure/{resourceUri}/providers/Microsoft.LoadTestService/loadTestProfileMappings/{loadTestProfileMappingName}?api-version=2024-12-01-preview`

Important routing implications from the reviewed pages:

- `resourceUri` is a fully qualified ARM resource ID and must remain path-safe through the router
- mapping resources need a generic nested-resource router, similar to other parent-resource-based Azure services
- `loadTests` is the only top-level persisted ARM resource in this surface
- the VUH limit routes are modeled as actions/resources attached to a `loadTests` instance rather than independent top-level resources

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- top-level `loadTests` returns standard ARM resource fields:
  - `id`
  - `name`
  - `type`
  - `location`
  - `identity`
  - `tags`
  - `systemData`
  - `properties.provisioningState`
- create and update pages document async headers for some responses

### Operations

`Operations.List`
- provider route under `/providers/Microsoft.LoadTestService/operations`
- standard provider operation list result

Modeling implication:
- expose a deterministic `value` list including at least:
  - `Microsoft.LoadTestService/loadTests/read`
  - `Microsoft.LoadTestService/loadTests/write`
  - `Microsoft.LoadTestService/loadTests/delete`
  - `Microsoft.LoadTestService/loadTests/outboundNetworkDependenciesEndpoints/read`
  - `Microsoft.LoadTestService/loadTests/limits/maxMonthlyVirtualUserHours/read`
  - `Microsoft.LoadTestService/loadTests/limits/maxMonthlyVirtualUserHours/set/action`
  - `Microsoft.LoadTestService/loadTestMappings/read`
  - `Microsoft.LoadTestService/loadTestMappings/write`
  - `Microsoft.LoadTestService/loadTestMappings/delete`
  - `Microsoft.LoadTestService/loadTestProfileMappings/read`
  - `Microsoft.LoadTestService/loadTestProfileMappings/write`
  - `Microsoft.LoadTestService/loadTestProfileMappings/delete`

### Load Test Resource

Reviewed method pages:
- `Create Or Update`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `Update`
- `Delete`
- `List Outbound Network Dependencies Endpoints`

Key traits from the reviewed pages:

- `Create Or Update` requires:
  - `location`
- optional create/update fields include:
  - `identity`
  - `properties.description`
  - `properties.encryption`
  - `tags`
- response fields include:
  - `properties.dataPlaneURI`
  - `properties.description`
  - `properties.encryption`
  - `properties.provisioningState`
- create may return `201` with `Azure-AsyncOperation` and `Retry-After`
- patch may return `200` or `202`
- delete is ARM-style and should tolerate repeated deletes
- outbound dependencies route returns `PagedOutboundEnvironmentEndpoint`

Modeling implications:
- persist one canonical `LoadTestResource` store keyed by ARM ID
- generate deterministic `dataPlaneURI` from subscription, resource group, and resource name
- synthesize stable outbound dependency endpoints instead of leaving the route unimplemented
- preserve caller-provided `identity`, `tags`, and `properties.description` with light validation only in the first pass

### Load Test Mappings

Reviewed method pages:
- `Create Or Update`
- `Get`
- `List`
- `Update`
- `Delete`

Key traits from the reviewed pages:

- parent route is `{resourceUri}/providers/Microsoft.LoadTestService/loadTestMappings`
- mapping resource name pattern is constrained to alphanumeric plus internal dashes
- payload fields include:
  - `properties.sourceResourceId`
  - `properties.azureLoadTestingResourceId`
  - `properties.testId`

Modeling implications:
- keep mappings in a dedicated store keyed by `(resourceUri, mappingName)`
- validate that `sourceResourceId` and the parent `resourceUri` are coherent in a lightweight way
- treat `testId` as an opaque string
- return standard ARM resource wrappers for mapping items

### Load Test Profile Mappings

Reviewed method pages:
- `Create Or Update`
- `Get`
- `List`
- `Delete`

Key traits from the reviewed pages:

- same parent-resource model as `loadTestMappings`
- payload fields include:
  - `properties.sourceResourceId`
  - `properties.azureLoadTestingResourceId`
  - `properties.testProfileId`
- there is no separate patch route in the reviewed surface

Modeling implications:
- reuse the same parent-resource nested routing helper used for `loadTestMappings`
- store profile mappings separately from test mappings because their payload contracts differ
- allow idempotent create-overwrite semantics on `PUT`

### Max Monthly Virtual User Hours Limits

Reviewed method pages:
- `Get`
- `Set`

Key traits from the reviewed pages:

- limit route is nested beneath a `loadTests` resource
- get route returns `MaxMonthlyVirtualUserHoursResource`
- set route is an action route under `/set`
- set uses `POST`

Modeling implications:
- store a per-load-test integer or structured limit record
- seed a default limit during `loadTests` creation so the get route always has deterministic output
- implement `POST .../set` as a state mutation on the parent `loadTests` record rather than a separate resource

## Non-Goals

- do not emulate the Load Testing data-plane test-run lifecycle in this service
- do not implement full customer-managed-key or managed-identity authorization behavior
- do not perform cross-resource existence checks for every mapped `resourceUri` in the first pass
- do not implement real ARM async operation polling unless coverage/examples require it
- do not model quota enforcement beyond returning and mutating a deterministic max-monthly-VUH record

## Stage 0: Coverage Naming And Route Claiming

- Add a dedicated Azure Load Testing router for `Microsoft.LoadTestService`
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add the aliases listed above
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route recognition tests for:
  - provider operations
  - subscription and resource-group `loadTests` listings
  - item CRUD routes
  - mapping child routes using `{resourceUri}`
  - outbound dependency and VUH limit routes

Exit criteria:
- coverage scripts can target the new service key without alias misses
- router tests prove requests resolve into the new provider instead of `NotImplemented`

## Stage 1: Provider Operations And Core `loadTests`

- Implement:
  - `GET /providers/Microsoft.LoadTestService/operations`
  - `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}`
  - `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}`
  - `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.LoadTestService/loadTests`
  - `PATCH /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}`
  - `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.LoadTestService/loadTests/{loadTestName}`
- Persist deterministic ARM metadata:
  - `id`
  - `type`
  - `systemData`
  - `properties.provisioningState`
  - `properties.dataPlaneURI`
- Seed default nested limit state for each created load test

Exit criteria:
- all core `loadTests` methods return modeled success shapes
- repeated get/list/patch/delete flows are deterministic
- no `loadTests` route falls through to `NotImplemented`

## Stage 2: Outbound Dependencies And Limits

- Implement:
  - `GET /.../loadTests/{loadTestName}/outboundNetworkDependenciesEndpoints`
  - `GET /.../loadTests/{loadTestName}/limits/maxMonthlyVirtualUserHours`
  - `POST /.../loadTests/{loadTestName}/limits/maxMonthlyVirtualUserHours/set`
- Return stable synthetic outbound endpoint pages with:
  - endpoint category/grouping
  - endpoint URIs or hostnames
  - paging shape if documented by coverage/examples
- Implement deterministic VUH limit get/set semantics

Exit criteria:
- auxiliary `loadTests` child routes are fully covered
- set followed by get reflects the mutated limit

## Stage 3: Generic Parent-Resource Mapping Infrastructure

- Add a shared helper for routes shaped like:
  - `/{resourceUri}/providers/Microsoft.LoadTestService/{childCollection}`
- Implement `loadTestMappings`:
  - create or update
  - get
  - list
  - patch
  - delete
- Implement `loadTestProfileMappings`:
  - create or update
  - get
  - list
  - delete
- Keep each child collection in a dedicated store with deterministic list ordering

Exit criteria:
- mapping collections behave consistently for arbitrary ARM parent resources
- both mapping families return modeled ARM resource envelopes
- no mapping route returns `NotImplemented`

## Stage 4: Tests And Contract Tightening

- Add focused tests in `internal/server/provider_azure_loadtesting_test.go` covering:
  - `loadTests` CRUD/list flows
  - outbound dependency listing
  - VUH get/set behavior
  - `loadTestMappings` create/get/list/patch/delete
  - `loadTestProfileMappings` create/get/list/delete
  - 404 behavior for missing resources
  - lightweight validation failures for malformed names or missing required fields
- Add deterministic assertions for:
  - `systemData`
  - ARM IDs and types
  - list ordering
  - nested route behavior with encoded parent `resourceUri` values

Exit criteria:
- focused server tests cover the full reviewed surface implemented in prior stages
- coverage scripts can exercise the service without route misses or empty placeholder payloads

## Stage 5: Azure Go SDK Example

- Add example under:
  - `examples/azure/loadtesting-resourcemanager-2024-12-01-preview`
- Prefer the official typed management SDK package:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/loadtesting/armloadtesting`
- Example flow should cover:
  - create a `loadTests` resource
  - get/list/update/delete the resource
  - list outbound dependencies
  - get/set max monthly virtual user hours
  - create/get/list/delete either a `loadTestMapping` or `loadTestProfileMapping`
- If the typed SDK does not expose every preview method, keep the example mostly typed and use a small raw ARM request helper only for the missing preview route

Exit criteria:
- example builds cleanly in repo CI conventions
- example demonstrates at least one top-level resource flow and one nested mapping or limit flow

## Validation Plan

After implementation, validate with:

- targeted server tests:
  - `go test ./internal/server -run 'TestAzureLoadTesting' -count=1`
- example build:
  - `go build ./examples/azure/loadtesting-resourcemanager-2024-12-01-preview`
- coverage checks:
  - `python3 scripts/azure-contract-coverage.py --service loadtesting --require-service loadtesting --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service loadtesting --require-service loadtesting --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service loadtesting --require-service loadtesting --fail-on any`
  - `python3 scripts/azure-endpoint-coverage.py --service loadtesting --require-service loadtesting --fail-on any`

Success target:
- all reviewed routes return modeled success or modeled service errors
- no reviewed Load Testing route returns `NotImplemented`
- the example compiles and exercises the new provider cleanly
