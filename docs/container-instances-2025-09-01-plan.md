# Azure Container Instances (`container-instances-2025-09-01`) Staged Plan

## Objective

Emulate the Azure Container Instances ARM management API for `2025-09-01` with deterministic local behavior for:

- provider `operations`
- tracked `containerGroups`
- container-group child action routes under `containers`
- subscription-location inventory routes for `cachedImages`, `capabilities`, and `usages`
- tracked `containerGroupProfiles` plus revision inspection
- subnet `serviceAssociationLinks/default` deletion
- an Azure Go SDK example under `examples/azure/container-instances-2025-09-01`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed surface is the ARM management plane under `Microsoft.ContainerInstance`.
- This is not the attach websocket data path, not direct container runtime streaming, and not AKS or Container Apps.
- The `2025-09-01` surface is broader than the long-standing ACI contract:
  - classic `containerGroups` still anchor the service
  - `containerGroupProfiles` add a separate tracked-resource family with revision semantics
  - `ngroups` appears in the reviewed Learn surface as a newer orchestration-style family with materially different behavior and should be staged separately instead of mixed into the first implementation slice
- Many method pages are authorization-gated, but they still expose route shape, parameters, request bodies, response codes, and sample payloads clearly enough to stage the emulator accurately.

## Primary References Reviewed

Primary operation-groups entry point reviewed:

- `https://learn.microsoft.com/en-us/rest/api/container-instances/operation-groups?view=rest-container-instances-2025-09-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/containers?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/location?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/subnet-service-association-link?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/n-groups?view=rest-container-instances-2025-09-01`

Representative linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/container-instances/operations/list?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/create-or-update?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/get?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/list?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/list-by-resource-group?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/update?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/delete?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/start?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/stop?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/restart?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/container-groups/get-outbound-network-dependencies-endpoints?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/containers/attach?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/containers/execute-command?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/containers/list-logs?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/location/list-cached-images?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/location/list-capabilities?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/location/list-usage?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile/create-or-update?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile/get?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile/update?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile/delete?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile/get-by-revision-number?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/cg-profile/list-all-revisions?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/subnet-service-association-link/delete?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/n-groups/create-or-update?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/n-groups/delete?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/n-groups/list-by-resource-group?view=rest-container-instances-2025-09-01`
- `https://learn.microsoft.com/en-us/rest/api/container-instances/n-groups/restart?view=rest-container-instances-2025-09-01`

## SDK Selection Note

- The future example should target the official Azure Go SDK management package for Container Instances, expected to be `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerinstance/armcontainerinstance`.
- The first example should prefer typed ARM clients for:
  - provider `operations`
  - `containerGroups`
  - location inventory routes
- If the exact local package revision does not expose every reviewed `2025-09-01` method cleanly, use narrow `azcore/runtime` fallback only for the missing routes rather than building the whole example around raw HTTP.

## Service Naming

Canonical Stackyard naming:

- service key: `container_instances`
- display name: `Azure Container Instances`
- API version token: `2025-09-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_container_instances.go`
- tests: `internal/server/provider_azure_container_instances_test.go`
- example: `examples/azure/container-instances-2025-09-01`
- plan doc: `docs/container-instances-2025-09-01-plan.md`

Recommended coverage aliases:

- `container_instances`
- `container-instances`
- `containerinstances`
- `azure_container_instances`
- `microsoft_containerinstance`
- `container_instances_2025_09_01`
- `rest_container_instances_2025_09_01`

## API Surface Summary

The reviewed surface breaks into 6 practical families.

Provider:

- `/providers/Microsoft.ContainerInstance/operations`

Tracked `containerGroups`:

- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerInstance/containerGroups/{containerGroupName}`
- subscription and resource-group list routes
- action routes:
  - `/start`
  - `/stop`
  - `/restart`
  - `/outboundNetworkDependenciesEndpoints`

Container child actions:

- `/containerGroups/{containerGroupName}/containers/{containerName}/attach`
- `/containerGroups/{containerGroupName}/containers/{containerName}/exec`
- `/containerGroups/{containerGroupName}/containers/{containerName}/logs`

Location-scoped inventory:

- `/subscriptions/{subscriptionId}/providers/Microsoft.ContainerInstance/locations/{location}/cachedImages`
- `/subscriptions/{subscriptionId}/providers/Microsoft.ContainerInstance/locations/{location}/capabilities`
- `/subscriptions/{subscriptionId}/providers/Microsoft.ContainerInstance/locations/{location}/usages`

Tracked `containerGroupProfiles`:

- `/containerGroupProfiles/{containerGroupProfileName}`
- `/containerGroupProfiles/{containerGroupProfileName}/revisions`
- `/containerGroupProfiles/{containerGroupProfileName}/revisions/{revisionNumber}`

Subnet service-association-link cleanup:

- `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}/providers/Microsoft.ContainerInstance/serviceAssociationLinks/default`

Later reconciliation family:

- `/providers/Microsoft.ContainerInstance/ngroups/...`

Modeling implications:

- `containerGroups` and `containerGroupProfiles` need separate tracked-resource stores.
- `containerGroupProfiles` are not just templates; they have revision history, so updates should append immutable snapshots rather than mutating history in place.
- `containers/*` routes are action-style projections over a parent container group and should be generated from current parent state instead of stored independently.
- regional inventory routes are deterministic read-only fixtures.
- `serviceAssociationLinks/default` is a cross-provider path and should be routed by exact suffix matching rather than generic ARM resource helpers.
- `ngroups` should be staged after the classic ACI families because the request/response model is materially different.

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use `CloudError` or ARM-style error envelopes
- list routes return `value`
- tracked resources carry standard ARM fields such as `id`, `name`, `type`, `location`, `tags`, and `properties`

Container groups:

- `Create Or Update` requires at least:
  - `location`
  - `properties.containers`
  - `properties.osType`
- common optional properties visible in the reviewed request/response model:
  - `ipAddress`
  - `restartPolicy`
  - `sku`
  - `imageRegistryCredentials`
  - `initContainers`
  - `volumes`
  - `subnetIds`
  - `containerGroupProfile`
  - `standbyPoolProfile`
  - `diagnostics`
  - `extensions`
  - `confidentialComputeProperties`
- response examples include:
  - `properties.provisioningState`
  - `properties.instanceView.state`
  - `properties.isCreatedFromStandbyPool`
- `Update` is intentionally narrow in this version:
  - the reviewed PATCH contract updates tags, location, and zones rather than the full container spec
- `Delete` can return:
  - `200`
  - `202` with `Location` and `Retry-After`
  - `204` when the resource does not exist
- `Get Outbound Network Dependencies Endpoints` explicitly states that container groups always return an empty list

Container child actions:

- `Attach` returns a websocket URI plus password
- `Execute Command` is a POST action under `/exec`
- `List Logs` is a GET under `/logs` and supports optional `tail` and `timestamps`

Location inventory:

- `List Capabilities` is the regional CPU/memory/GPU capability catalog
- `List Cached Images` is a regional cached-image catalog
- `List Usage` returns count-style usage data such as current value and limit

Container group profiles:

- `Create Or Update` requires:
  - `location`
  - `properties.containers`
  - `properties.osType`
- the response includes a `revision` property
- `Update` is tag-only in the reviewed PATCH contract
- `List All Revisions` and `Get By Revision Number` expose immutable revision history that the emulator should preserve deterministically
- `Delete` returns:
  - `200` when delete starts
  - `204` when the profile is absent

Subnet service association link:

- only `Delete` is documented in the reviewed surface
- the documented target is always the `default` link under the subnet path

NGroups:

- the reviewed surface exposes tracked CRUD/list/action behavior plus asynchronous delete and restart semantics
- the request model is centered on profile sets, placement, elastic behavior, and update policy rather than classic single container-group execution
- this is large enough to stage after the traditional ACI surface instead of blocking the first implementation

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the service shell and coverage wiring:

- reserve service key `container_instances`
- add provider registration in `internal/server/http.go`
- add plan-doc mapping and aliases in the three Azure coverage scripts
- scaffold `internal/server/provider_azure_container_instances.go`
- scaffold `examples/azure/container-instances-2025-09-01`

Exit criteria:

- routing compiles
- coverage scripts resolve the new service key
- the plan doc is mapped for doc coverage

### Stage 1: Provider Operations and Location Inventory

Implement the low-risk read-only routes first:

- `Operations.List`
- `Location.ListCachedImages`
- `Location.ListCapabilities`
- `Location.ListUsage`

Why first:

- these routes establish provider identity and regional fixture strategy
- they are read-only and cheap to validate

### Stage 2: Core Container Groups

Implement the main tracked resource family:

- `Create Or Update`
- `Get`
- `List`
- `List By Resource Group`
- `Update`
- `Delete`

Implementation notes:

- validate `location`, `properties.containers`, and `properties.osType`
- normalize common optional fields into a deterministic response shape
- synthesize `properties.instanceView.state` and `properties.provisioningState`
- use the repo’s existing ARM LRO conventions only where the contract clearly needs them; otherwise keep the first pass synchronous and deterministic

### Stage 3: Container Group Actions

Add parent-scoped action routes:

- `Start`
- `Stop`
- `Restart`
- `Get Outbound Network Dependencies Endpoints`

Implementation notes:

- model power-state transitions in `instanceView.state`
- keep outbound dependency responses empty as documented

### Stage 4: Container Child Actions

Implement the action routes under `containers`:

- `Attach`
- `Execute Command`
- `List Logs`

Implementation notes:

- these routes should validate parent container-group existence first
- `Attach` should return deterministic websocket credentials
- `Execute Command` should validate a non-empty command payload and return deterministic connection/session material
- `List Logs` should honor simple `tail` and `timestamps` shaping without trying to emulate real streaming

### Stage 5: Container Group Profiles

Implement the separate tracked template family:

- `Create Or Update`
- `Get`
- `Update`
- `Delete`
- `List All Revisions`
- `Get By Revision Number`

Implementation notes:

- maintain a live profile record plus append-only revision snapshots
- increment revisions deterministically on create/update of the full resource
- keep PATCH tag-only, matching the reviewed contract

### Stage 6: Subnet Service Association Link Cleanup

Implement:

- `Subnet Service Association Link.Delete`

Implementation notes:

- this is a narrow, exact-match route on a Microsoft.Network subnet path
- deletion can be modeled as idempotent success keyed by subnet resource ID

### Stage 7: NGroups Reconciliation

Implement the newer orchestration family only after the classic ACI surface is stable:

- `Create Or Update`
- `Get`
- `List`
- `List By Resource Group`
- `Update`
- `Delete`
- `Start`
- `Stop`
- `Restart`

Why later:

- the request and state model differs materially from classic `containerGroups`
- it likely deserves its own internal store and tests rather than being shoehorned into container-group logic

### Stage 8: Example

Add `examples/azure/container-instances-2025-09-01` using the Azure Go SDK.

Suggested example flow:

- list provider operations
- list location usage or capabilities
- create a simple Linux `containerGroup`
- get and list that group
- fetch container logs
- restart or stop/start the group
- create a `containerGroupProfile`
- get the profile and list its revisions

Acceptance target:

- `go build ./examples/azure/container-instances-2025-09-01` succeeds
- the example demonstrates both tracked-resource CRUD and at least one action route

### Stage 9: Tests and Coverage Closure

Add focused provider contract tests for:

- provider operations
- location inventory routes
- container-group lifecycle
- container-group start/stop/restart
- attach/exec/logs
- container-group-profile revision behavior
- subnet service-association-link delete
- validation failures for missing required fields and malformed path/query values

Acceptance target:

- `go test ./internal/server -run 'TestAzureContainerInstances' -count=1` passes
- coverage scripts pass for `container_instances`
- endpoint coverage reports no `NotImplemented` for the implemented surface

## Recommended First Implementation Slice

The first implementation PR should cover enough of the provider to feel complete without taking on `ngroups` yet:

- provider `operations`
- location `cachedImages`, `capabilities`, and `usages`
- `containerGroups` CRUD/list/update
- `containerGroups` `start`, `stop`, `restart`, and outbound-dependencies route
- `containers` attach/exec/logs
- `containerGroupProfiles` create/get/update/delete plus revision inspection
- subnet service-association-link delete
- example and coverage wiring

This gives the repo a credible `Microsoft.ContainerInstance` footprint quickly, while keeping the broader `ngroups` family isolated for a follow-up merge where its different behavior can be modeled cleanly.
