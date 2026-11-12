# Azure Resource Graph (`azureresourcegraph-resourcegraph-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Resource Graph service for the `2024-04-01` REST surface with deterministic local behavior for:

- provider `operations`
- provider-scoped `resources` query execution
- ARM-tracked `graph query` resources under `Microsoft.ResourceGraph/queries`
- provider-scoped `resourcesHistory`
- provider-scoped `resourceChanges`
- provider-scoped `resourceChangeDetails`
- an Azure Go SDK example under `examples/azure/azureresourcegraph-resourcegraph-2024-04-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is a mixed service shape.
- `Graph Query` is an ARM management resource under `Microsoft.ResourceGraph/queries`.
- `Resources`, `Resources History`, `Resource Changes`, and `Resource Change Details` are provider-scoped POST routes under `/providers/Microsoft.ResourceGraph/...`, not ARM tracked resources.
- The Microsoft Learn operation-groups page is anchored on `2024-04-01`, but some linked method pages still use older preview API versions:
  - `resourcesHistory`: `2021-06-01-preview`
  - `resourceChanges`: `2020-09-01-preview`
  - `resourceChangeDetails`: `2020-09-01-preview`
- This plan intentionally keeps those version splits explicit so implementation and coverage do not flatten incompatible route contracts into one fake version.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/operation-groups?view=rest-azureresourcegraph-resourcegraph-2024-04-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/operations?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resource-change-details?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resource-changes?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resources?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resources-history?view=rest-azureresourcegraph-resourcegraph-2024-04-01`

Representative linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/operations/list?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query/create-or-update?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query/get?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query/list?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query/list-by-subscription?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query/update?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/graph-query/delete?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resources/resources?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resources-history/resources-history?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resource-changes/resource-changes?view=rest-azureresourcegraph-resourcegraph-2024-04-01`
- `https://learn.microsoft.com/en-us/rest/api/azureresourcegraph/resourcegraph/resource-change-details/resource-change-details?view=rest-azureresourcegraph-resourcegraph-2024-04-01`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph`

SDK selection note:
- The official track-2 Go package `armresourcegraph` exposes typed clients for:
  - `Resources`
  - `ResourcesHistory`
  - provider `Operations`
- It does not appear to expose typed `Graph Query` ARM CRUD clients or the preview `resourceChanges` / `resourceChangeDetails` routes.
- Plan assumption: the example should use typed `armresourcegraph` clients where available, and a small `azcore/arm` raw request helper for the ARM `queries` resource and preview change endpoints.

## Service Naming

Canonical Stackyard naming:
- service key: `resource_graph`
- display name: `Azure Resource Graph`
- provider namespace: `Microsoft.ResourceGraph`
- primary API version: `2024-04-01`

Secondary version tokens required by reviewed method pages:
- `resourcesHistory`: `2021-06-01-preview`
- `resourceChanges`: `2020-09-01-preview`
- `resourceChangeDetails`: `2020-09-01-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_resource_graph.go`
- tests: `internal/server/provider_azure_resource_graph_test.go`
- example: `examples/azure/azureresourcegraph-resourcegraph-2024-04-01`
- plan doc: `docs/azureresourcegraph-resourcegraph-2024-04-01-plan.md`

Recommended coverage aliases:
- `resource_graph`
- `resource-graph`
- `azure_resource_graph`
- `azureresourcegraph`
- `graph_query`
- `graph_queries`
- `microsoft_resourcegraph`
- `azureresourcegraph_resourcegraph_2024_04_01`
- `rest_azureresourcegraph_resourcegraph_2024_04_01`

## Surface Summary

The reviewed surface exposes 6 operation groups:

1. `Graph Query`
2. `Operations`
3. `Resource Change Details`
4. `Resource Changes`
5. `Resources`
6. `Resources History`

Broad route families identified from the reviewed pages:

- provider discovery:
  - `/providers/Microsoft.ResourceGraph/operations`
- provider query endpoints:
  - `/providers/Microsoft.ResourceGraph/resources`
  - `/providers/Microsoft.ResourceGraph/resourcesHistory`
  - `/providers/Microsoft.ResourceGraph/resourceChanges`
  - `/providers/Microsoft.ResourceGraph/resourceChangeDetails`
- ARM tracked resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
- ARM query listings:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ResourceGraph/queries`

## Route Envelope

Documented route family:
- all reviewed routes live under the Azure Resource Manager host
- `Graph Query`, `Operations`, and `Resources` use `api-version=2024-04-01`
- `Resources History` uses `api-version=2021-06-01-preview`
- `Resource Changes` and `Resource Change Details` use `api-version=2020-09-01-preview`

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.ResourceGraph/operations?api-version=2024-04-01`
- `POST /azure/providers/Microsoft.ResourceGraph/resources?api-version=2024-04-01`
- `POST /azure/providers/Microsoft.ResourceGraph/resourcesHistory?api-version=2021-06-01-preview`
- `POST /azure/providers/Microsoft.ResourceGraph/resourceChanges?api-version=2020-09-01-preview`
- `POST /azure/providers/Microsoft.ResourceGraph/resourceChangeDetails?api-version=2020-09-01-preview`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}?api-version=2024-04-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}?api-version=2024-04-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}?api-version=2024-04-01`
- `DELETE /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}?api-version=2024-04-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries?api-version=2024-04-01`
- `GET /azure/subscriptions/{subscriptionId}/providers/Microsoft.ResourceGraph/queries?api-version=2024-04-01`

Important routing implications from the reviewed pages:

- `queries` is an ARM tracked resource collection, while `resources*` and `resourceChange*` are provider-scoped POST actions
- `operations` is provider-rooted and does not require a subscription segment
- the service must accept multiple API versions at once for different route families
- the provider-scoped POST routes are query-style request/response contracts, not long-lived persisted resources
- `resources` and `resourcesHistory` requests support scopes via `subscriptions` and `managementGroups`

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- errors use a common Resource Graph JSON error envelope
- provider query routes return structured result sets rather than ARM resource envelopes
- graph-query ARM resources return standard ARM fields:
  - `id`
  - `name`
  - `type`
  - `etag`
  - `location`
  - `tags`
  - `systemData`
  - `properties.*`

### Operations

Reviewed method page:
- `Operations.List`

Key traits:
- route:
  - `/providers/Microsoft.ResourceGraph/operations`
- response returns provider operation metadata

Modeling implications:
- expose a deterministic `value` list including at least:
  - `Microsoft.ResourceGraph/resources/read`
  - `Microsoft.ResourceGraph/resourcesHistory/read`
  - `Microsoft.ResourceGraph/resourceChanges/read`
  - `Microsoft.ResourceGraph/resourceChangeDetails/read`
  - `Microsoft.ResourceGraph/queries/read`
  - `Microsoft.ResourceGraph/queries/write`
  - `Microsoft.ResourceGraph/queries/delete`

### Graph Query

Reviewed method pages:
- `Create Or Update`
- `Delete`
- `Get`
- `List`
- `List By Subscription`
- `Update`

Key traits:
- resource route:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
- name pattern excludes `<`, `>`, `&`, `:`, and `?`
- create requires:
  - `properties.query`
- optional create/update fields include:
  - `etag`
  - `location`
  - `properties.description`
  - `tags`
- response fields include:
  - `properties.query`
  - `properties.description`
  - `properties.resultKind`
  - `properties.timeModified`
- delete returns:
  - `200 OK` when a resource is deleted
  - `204 No Content` when the resource does not exist

Modeling implications:
- persist canonical graph-query resources keyed by ARM ID
- support `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list by resource group, and list by subscription
- preserve caller-provided query text, description, location, and tags
- synthesize stable `etag`, `properties.resultKind`, `properties.timeModified`, and `systemData`
- treat delete as idempotent with the documented `200` or `204` semantics

### Resources

Reviewed method page:
- `Resources`

Key traits:
- route:
  - `/providers/Microsoft.ResourceGraph/resources`
- request requires:
  - `query`
- optional request fields include:
  - `subscriptions`
  - `managementGroups`
  - `facets`
  - `options`
- response is a `QueryResponse` with fields such as:
  - `count`
  - `data`
  - `facets`
  - `resultTruncated`
  - `totalRecords`
  - `skipToken`

Modeling implications:
- emulate deterministic KQL-like results for a small useful subset instead of a real Kusto engine
- preserve request scopes and echo them into synthetic results where helpful
- support both table-like and object-array response data in a minimal, deterministic way
- include facet handling and skip-token handling in a lightweight manner because they are first-class request fields in the reviewed contract

### Resources History

Reviewed method page:
- `Resources History`

Key traits:
- route:
  - `/providers/Microsoft.ResourceGraph/resourcesHistory`
- request resembles `Resources` but is history-oriented
- method page is versioned at `2021-06-01-preview`
- returns a result set of historical snapshots for a time interval

Modeling implications:
- implement this as a provider-scoped POST route with preview-version enforcement
- return deterministic historical snapshots for known resource IDs and scopes
- preserve interval and pagination options without implementing a full historical store

### Resource Changes

Reviewed method page:
- `Resource Changes`

Key traits:
- route:
  - `/providers/Microsoft.ResourceGraph/resourceChanges`
- method page is versioned at `2020-09-01-preview`
- request requires:
  - `interval`
- optional request fields include:
  - `$skipToken`
  - `$top`
  - `fetchPropertyChanges`
  - `fetchSnapshots`
  - resource scope filters

Modeling implications:
- implement deterministic change-record results for known resource IDs
- support `$top` and `$skipToken` in a minimal paging model
- optionally enrich responses when `fetchPropertyChanges` or `fetchSnapshots` is true
- keep returned `changeId` values stable so `resourceChangeDetails` can consume them

### Resource Change Details

Reviewed method page:
- `Resource Change Details`

Key traits:
- route:
  - `/providers/Microsoft.ResourceGraph/resourceChangeDetails`
- method page is versioned at `2020-09-01-preview`
- request requires:
  - `changeIds`
  - `resourceIds`
- response returns an array of resource change data with property-level diffs

Modeling implications:
- derive details from the stable synthetic change IDs emitted by `resourceChanges`
- return deterministic before/after property changes for a small set of mocked resources
- validate that at least one `changeId` and `resourceId` is present and coherent

## Non-Goals

- do not implement a real KQL parser or evaluation engine
- do not model tenant-wide or management-group-wide authorization checks beyond lightweight request-shape validation
- do not emulate every possible facet type or query option in the first pass
- do not build a persistent historical store beyond deterministic fixtures seeded from in-memory state
- do not attempt to support undocumented API-version aliases beyond the reviewed method pages

## Stage 0: Coverage Naming And Route Claiming

- Add a dedicated Azure Resource Graph router
- Normalize service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- Add the aliases listed above
- Map this plan doc in `azure-doc-contract-coverage.py`
- Add route recognition tests for:
  - provider operations
  - provider POST query routes
  - graph-query ARM resource routes
  - the preview-version route split for history and change endpoints

Exit criteria:
- coverage scripts can target the new service key without alias misses
- router tests prove requests resolve into the new provider instead of `NotImplemented`

## Stage 1: Provider Operations And `Resources`

- Implement:
  - `GET /providers/Microsoft.ResourceGraph/operations`
  - `POST /providers/Microsoft.ResourceGraph/resources`
- Model `QueryRequest` validation for:
  - required `query`
  - acceptable scope fields
  - lightweight options and facets structure
- Return deterministic `QueryResponse` payloads with:
  - `count`
  - `data`
  - `facets`
  - `resultTruncated`
  - `totalRecords`
  - optional `skipToken`

Exit criteria:
- provider operations and the core resource query route return modeled success shapes
- `resources` no longer falls through to `NotImplemented`

## Stage 2: Graph Query ARM Resources

- Implement:
  - `PUT /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
  - `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
  - `HEAD /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
  - `PATCH /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
  - `DELETE /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries/{resourceName}`
  - `GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ResourceGraph/queries`
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.ResourceGraph/queries`
- Persist deterministic ARM metadata:
  - `id`
  - `type`
  - `etag`
  - `systemData`
  - `properties.resultKind`
  - `properties.timeModified`

Exit criteria:
- all graph-query CRUD and list routes return modeled resource envelopes
- repeated get/list/patch/delete flows are deterministic
- no `queries` route returns `NotImplemented`

## Stage 3: History And Change Preview Routes

- Implement:
  - `POST /providers/Microsoft.ResourceGraph/resourcesHistory?api-version=2021-06-01-preview`
  - `POST /providers/Microsoft.ResourceGraph/resourceChanges?api-version=2020-09-01-preview`
  - `POST /providers/Microsoft.ResourceGraph/resourceChangeDetails?api-version=2020-09-01-preview`
- Enforce the reviewed preview versions explicitly rather than accepting `2024-04-01` on those routes
- Return deterministic historical snapshots and change records tied to stable synthetic resources
- Implement minimal paging for `resourceChanges` with `$top` and `$skipToken`

Exit criteria:
- all reviewed preview routes return modeled success shapes
- `resourceChanges` and `resourceChangeDetails` interoperate through stable synthetic `changeId` values

## Stage 4: Tests And Contract Tightening

- Add focused tests in `internal/server/provider_azure_resource_graph_test.go` covering:
  - provider operations
  - `resources` query success and validation failures
  - graph-query create/get/head/list/patch/delete
  - graph-query list ordering
  - `resourcesHistory` preview route behavior
  - `resourceChanges` paging and optional flags
  - `resourceChangeDetails` lookups and validation errors
  - 404 behavior for missing graph-query resources
  - version mismatch failures for preview routes

Exit criteria:
- focused server tests cover the full reviewed surface implemented in prior stages
- coverage scripts can exercise the service without route misses or placeholder empty payloads

## Stage 5: Azure Go SDK Example

- Add example under:
  - `examples/azure/azureresourcegraph-resourcegraph-2024-04-01`
- Prefer the official typed package:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph`
- Example flow should cover:
  - list provider operations
  - execute a `resources` query
  - execute a `resourcesHistory` query
  - create/get/list/update/delete a graph-query resource using raw ARM helper requests if needed
  - call one preview change route using a raw helper if no typed SDK surface exists
- Keep typed SDK usage primary where the official package supports the route

Exit criteria:
- example builds cleanly in repo CI conventions
- example demonstrates at least one typed query flow and one ARM graph-query lifecycle flow

## Validation Plan

After implementation, validate with:

- targeted server tests:
  - `go test ./internal/server -run 'TestAzureResourceGraph' -count=1`
- example build:
  - `go build ./examples/azure/azureresourcegraph-resourcegraph-2024-04-01`
- coverage checks:
  - `python3 scripts/azure-contract-coverage.py --service resource_graph --require-service resource_graph --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service resource_graph --require-service resource_graph --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service resource_graph --require-service resource_graph --fail-on any`
  - `python3 scripts/azure-endpoint-coverage.py --service resource_graph --require-service resource_graph --fail-on any`

Success target:
- all reviewed routes return modeled success or modeled service errors
- no reviewed Azure Resource Graph route returns `NotImplemented`
