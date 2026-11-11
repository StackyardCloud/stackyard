# Azure Compute Limit (`computelimit-2026-04-30`) Staged Plan

## Objective

Emulate the Azure Compute Limit ARM management API for `2026-04-30` with deterministic local behavior for:

- provider `operations`
- location-scoped `features`
- location-scoped `guestSubscriptions`
- location-scoped `sharedLimits`
- location-scoped `vmFamilies`
- an Azure Go SDK example under `examples/azure/computelimit-2026-04-30`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed Learn surface is the ARM management plane under `Microsoft.ComputeLimit`, not the broader `Microsoft.Compute` provider and not Azure Quota.
- This API is about host-subscription compute-limit sharing and related metadata, not VM CRUD and not generic quota-request workflows.
- The operation-groups page for `2026-04-30` is centered on:
  - `Features`
  - `Guest Subscriptions`
  - `Operations`
  - `Shared Limits`
  - `Vm Families`
- A few linked method pages surfaced in adjacent search results with older preview or neighboring versions such as `2025-08-15`, but the `2026-04-30` group pages and the `2026-04-30` method pages that were directly reviewed are consistent enough to stage the provider cleanly.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/computelimit/operation-groups?view=rest-computelimit-2026-04-30`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/computelimit/operations?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/features?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/shared-limits?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/vm-families?view=rest-computelimit-2026-04-30`

Representative linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/computelimit/operations/list?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/features/enable?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/shared-limits/create?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/shared-limits/get?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/shared-limits/delete?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/vm-families/get?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/guest-subscriptions/list-by-subscription-location-resource?view=rest-computelimit-2026-04-30`
- `https://learn.microsoft.com/en-us/rest/api/computelimit/guest-subscriptions/create?view=rest-computelimit-2026-04-30`

## SDK Selection Note

- The Azure Go SDK repository lists a dedicated module for this surface: `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/computelimit/armcomputelimit`.
- The future example should prefer typed `armcomputelimit` clients for:
  - `Operations`
  - `SharedLimits`
  - `GuestSubscriptions`
- If the local package version does not expose every reviewed `2026-04-30` route cleanly, a narrow `azcore/runtime` fallback is acceptable for the remaining families such as `Features` or `VmFamilies`.

## Service Naming

Canonical Stackyard naming:

- service key: `compute_limit`
- display name: `Azure Compute Limit`
- API version token: `2026-04-30`

Recommended repo paths:

- implementation: `internal/server/provider_azure_compute_limit.go`
- tests: `internal/server/provider_azure_compute_limit_test.go`
- example: `examples/azure/computelimit-2026-04-30`
- plan doc: `docs/computelimit-2026-04-30-plan.md`

Recommended coverage aliases:

- `compute_limit`
- `compute-limit`
- `computelimit`
- `azure_compute_limit`
- `microsoft_computelimit`
- `computelimit_2026_04_30`
- `rest_computelimit_2026_04_30`

## API Surface Summary

The reviewed pages expose five route families:

- provider-level:
  - `/providers/Microsoft.ComputeLimit/operations`
- location-scoped feature routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeLimit/locations/{location}/features`
  - `.../features/{featureName}`
  - `.../features/{featureName}/enable`
  - `.../features/{featureName}/disable`
- location-scoped guest-subscription routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeLimit/locations/{location}/guestSubscriptions`
  - `.../guestSubscriptions/{guestSubscriptionId}`
- location-scoped shared-limit routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeLimit/locations/{location}/sharedLimits`
  - `.../sharedLimits/{name}`
- location-scoped VM-family routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.ComputeLimit/locations/{location}/vmFamilies`
  - `.../vmFamilies/{vmFamilyName}`

Modeling implications:

- this is a location-scoped ARM surface without resource groups
- all mutable resources are subscription-location keyed
- `sharedLimits`, `guestSubscriptions`, and `features` can be modeled as small, deterministic resource stores keyed by `(subscriptionId, location, name)`
- `vmFamilies` can be a read-only fixture family

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- resources carry ARM-like `id`, `name`, `type`, `properties`, and `systemData`
- path parameters validate `subscriptionId` as UUID and `location` as non-empty

### Operations

From `Operations.List`:

- route:
  - `GET /providers/Microsoft.ComputeLimit/operations?api-version=2026-04-30`
- response:
  - `value`
  - each operation can include `name`, `display`, optional `origin`, `actionType`, and `isDataAction`

The sample operation list also reveals the practical implementation surface:

- guest-subscription read/write/delete
- shared-limit read/write/delete
- feature read and action routes
- VM-family read routes

### Shared Limits

From `Shared Limits` pages:

- routes:
  - `PUT .../sharedLimits/{name}`
  - `GET .../sharedLimits/{name}`
  - `DELETE .../sharedLimits/{name}`
  - `GET .../sharedLimits`
- name pattern:
  - `^[a-zA-Z0-9-]{3,24}$`
- resource shape:
  - `properties.resourceName.{value,localizedValue}`
  - `properties.limit`
  - `properties.unit`
  - `properties.provisioningState`

Behavior implication:

- create/update is effectively enabling sharing for a named compute family limit
- delete disables sharing

### Guest Subscriptions

From `Guest Subscriptions` pages:

- routes:
  - `PUT .../guestSubscriptions/{guestSubscriptionId}`
  - `GET .../guestSubscriptions/{guestSubscriptionId}`
  - `DELETE .../guestSubscriptions/{guestSubscriptionId}`
  - `GET .../guestSubscriptions`
- guest subscription path parameter is UUID-shaped

Behavior implication:

- this is a relationship resource between a host subscription and guest subscriptions in a given location
- deterministic fixtures should expose guest subscription IDs and provisioning state

### Features

From `Features` pages:

- routes:
  - `GET .../features`
  - `GET .../features/{featureName}`
  - `POST .../features/{featureName}/enable`
  - `POST .../features/{featureName}/disable`
- feature names are location-scoped and action-driven
- the action response shape uses operation-status style fields such as `status`

Behavior implication:

- `enable` and `disable` can mutate a small in-memory feature state and return deterministic `Succeeded` operation payloads

### Vm Families

From `Vm Families` pages:

- routes:
  - `GET .../vmFamilies`
  - `GET .../vmFamilies/{vmFamilyName}`
- resource shape:
  - `properties.category`
  - `properties.provisioningState`
- list route accepts optional `$filter`

Behavior implication:

- a first pass can support deterministic read-only fixtures and simple filter matching on family name or category

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the new service shell and alias wiring:

- reserve service key `compute_limit`
- add provider registration in `internal/server/http.go`
- add provider scaffolding in `internal/server/provider_azure_compute_limit.go`
- add plan-doc mapping and aliases in the three Azure coverage scripts
- create the example directory scaffold under `examples/azure/computelimit-2026-04-30`

Exit criteria:

- routing compiles
- coverage scripts can discover the new service key

### Stage 1: Provider Operations and Read-Only Metadata

Implement the smallest read-only routes first:

- `Operations.List`
- `VmFamilies.ListBySubscriptionLocationResource`
- `VmFamilies.Get`

Why first:

- these routes establish the provider identity and location-scoped read behavior
- they are low-risk and unblock the example before mutable flows

### Stage 2: Shared Limits

Implement:

- `SharedLimits.ListBySubscriptionLocationResource`
- `SharedLimits.Get`
- `SharedLimits.Create`
- `SharedLimits.Delete`

Behavior goals:

- validate subscription ID, location, and limit name
- maintain a deterministic store of shared limits
- return `200` or `201` on create/update according to existence
- return either `200` or `204` semantics for delete based on repo convention

Why here:

- shared limits are the core managed resource in this provider
- guest-subscription behavior is easier to reason about once shared limits exist

### Stage 3: Guest Subscriptions

Implement:

- `GuestSubscriptions.ListBySubscriptionLocationResource`
- `GuestSubscriptions.Get`
- `GuestSubscriptions.Create`
- `GuestSubscriptions.Delete`

Behavior goals:

- validate guest-subscription IDs as UUIDs
- keep guest subscriptions keyed by host subscription and location
- optionally validate that at least one shared limit exists before allowing a guest subscription to be created, if that matches the chosen deterministic model

Why here:

- this is the second core mutable hierarchy and closely tied to shared-limit usage

### Stage 4: Features and Action Routes

Implement:

- `Features.ListBySubscriptionLocationResource`
- `Features.Get`
- `Features.Enable`
- `Features.Disable`

Behavior goals:

- maintain a small deterministic set of supported feature names
- toggle enabled state per `(subscriptionId, location, featureName)`
- return operation-status shaped responses with `Succeeded`

Why here:

- these routes are action-based and simpler once the subscription-location parsing and ARM error conventions are already established

### Stage 5: Example and Coverage Finalization

Add:

- focused provider tests for:
  - operations success
  - VM family read success
  - shared-limit lifecycle
  - guest-subscription lifecycle
  - feature enable/disable
  - request-validation failures
- the example under `examples/azure/computelimit-2026-04-30`
- final coverage aliases and doc mappings

## Example Scope

The future example should demonstrate the real provider shape rather than synthetic quota workflows:

1. list provider operations
2. list VM families for one region
3. create a shared limit
4. add a guest subscription
5. enable a feature
6. fetch the shared limit back

Recommended example files:

- `examples/azure/computelimit-2026-04-30/main.go`
- `examples/azure/computelimit-2026-04-30/Dockerfile`
- `examples/azure/computelimit-2026-04-30/docker-compose.yml`

## Coverage Script Updates

When implementation begins, update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add:

- alias normalization for `compute_limit`
- plan-doc mapping to `computelimit-2026-04-30-plan.md`
- any example-path or service-name hints needed by the existing coverage tooling

## Validation Plan

After implementation, validate with:

- focused provider tests for `compute_limit`
- example build for `examples/azure/computelimit-2026-04-30`
- Azure contract coverage for `--service compute_limit`
- Azure IO coverage for `--service compute_limit`
- Azure doc coverage for `--service compute_limit`
- Azure endpoint coverage for `--service compute_limit`
- full `make ci`, reporting unrelated repo failures separately if they remain

## Recommended Implementation Order

If implemented incrementally, merge in this order:

1. Stage 0 and Stage 1
2. Stage 2
3. Stage 3
4. Stage 4
5. Stage 5

This keeps the first slice narrow, then layers the mutable host-sharing relationships in a controlled order that matches the actual operation groups.
