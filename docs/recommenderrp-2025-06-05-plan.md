# Azure Compute Recommender (`recommenderrp-2025-06-05`) Staged Plan

## Objective

Emulate the Azure Compute Recommender ARM management API for `2025-06-05` with deterministic local behavior for:

- provider `operations`
- location-scoped Spot Placement Score metadata
- location-scoped Spot Placement Score generation
- an Azure Go SDK example under `examples/azure/recommenderrp-2025-06-05`
- Azure coverage-script wiring for the new service key

Important scope notes:

- The reviewed Learn surface is an ARM management-plane provider under `Microsoft.Compute`.
- This is not the broad `Microsoft.Compute` resource-management API and not Azure Advisor.
- The `rest-recommenderrp-2025-06-05` operation-groups page is intentionally narrow:
  - `Operations`
  - `Spot Placement Scores`
- The implementation should mirror that narrow surface instead of inventing unrelated recommendation resources.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/recommenderrp/operation-groups?view=rest-recommenderrp-2025-06-05`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/recommenderrp/operations?view=rest-recommenderrp-2025-06-05`
- `https://learn.microsoft.com/en-us/rest/api/recommenderrp/spot-placement-scores?view=rest-recommenderrp-2025-06-05`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/recommenderrp/operations/list?view=rest-recommenderrp-2025-06-05`
- `https://learn.microsoft.com/en-us/rest/api/recommenderrp/spot-placement-scores/get?view=rest-recommenderrp-2025-06-05`
- `https://learn.microsoft.com/en-us/rest/api/recommenderrp/spot-placement-scores/post?view=rest-recommenderrp-2025-06-05`

## SDK Selection Note

- I did not confirm a clearly matching dedicated typed Go SDK package for this exact `recommenderrp` surface.
- The future example should therefore use the official Azure Go SDK core pipeline via `azcore/runtime` and `azcore/policy`.
- If a local typed client is later confirmed to cover these exact routes, it can replace the runtime example without changing the emulator contract.

## Service Naming

Canonical Stackyard naming:

- service key: `compute_recommender`
- display name: `Azure Compute Recommender`
- API version token: `2025-06-05`

Recommended repo paths:

- implementation: `internal/server/provider_azure_compute_recommender.go`
- tests: `internal/server/provider_azure_compute_recommender_test.go`
- example: `examples/azure/recommenderrp-2025-06-05`
- plan doc: `docs/recommenderrp-2025-06-05-plan.md`

Recommended coverage aliases:

- `compute_recommender`
- `compute-recommender`
- `azure_compute_recommender`
- `recommenderrp`
- `rest_recommenderrp_2025_06_05`
- `recommenderrp_2025_06_05`

## API Surface Summary

The reviewed pages expose two route families:

- provider-level operations:
  - `/providers/Microsoft.Compute/operations`
- subscription and location-scoped recommender routes:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Compute/locations/{location}/placementScores/spot`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.Compute/locations/{location}/placementScores/spot/generate`

Modeling implications:

- the provider can be implemented without a persistent tracked-resource hierarchy
- the only mutable-like behavior is request-driven score generation, which can be deterministic and stateless
- subscription ID and location validation matter because both Spot Placement Score routes are path-scoped

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `Error Response`
- `Operations.List` returns `value`
- Spot Placement Score responses are ordinary JSON objects, not tracked ARM resources stored under a resource group

From `Operations.List`:

- route:
  - `GET /providers/Microsoft.Compute/operations?api-version=2025-06-05`
- response shape:
  - `value`
  - each item can include `origin`, `name`, `display`, optional `actionType`, and `isDataAction`

From `Spot Placement Scores.Get`:

- route:
  - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Compute/locations/{location}/placementScores/spot?api-version=2025-06-05`
- response shape:
  - top-level `name`
  - `type`
  - `id`
  - `properties.supportedResourceTypes`
- this is metadata about the diagnostic or recommendation type, not a generated score payload

From `Spot Placement Scores.Post`:

- route:
  - `POST /subscriptions/{subscriptionId}/providers/Microsoft.Compute/locations/{location}/placementScores/spot/generate?api-version=2025-06-05`
- request body:
  - `availabilityZones`
  - `desiredCount`
  - `desiredLocations`
  - `desiredSizes[].sku`
- response shape:
  - echoes `availabilityZones`
  - echoes `desiredCount`
  - echoes `desiredLocations`
  - echoes `desiredSizes`
  - returns `placementScores[]` with:
    - `availabilityZone`
    - `isQuotaAvailable`
    - `region`
    - `score`
    - `sku`
- sample response also includes a `location` header

Validation implications from the reviewed request schema:

- `subscriptionId` must be present and UUID-shaped
- `location` must be non-empty
- `desiredCount` should be present and positive
- `desiredLocations` should be non-empty
- `desiredSizes` should be non-empty
- each desired size should include a non-empty `sku`

## Proposed Stages

### Stage 0: Route Wiring and Coverage Skeleton

Add the new service shell and alias wiring:

- reserve service key `compute_recommender`
- add provider registration in `internal/server/http.go`
- add provider scaffolding in `internal/server/provider_azure_compute_recommender.go`
- add plan-doc mapping and aliases in the three Azure coverage scripts
- create the example directory scaffold under `examples/azure/recommenderrp-2025-06-05`

Exit criteria:

- routing compiles
- coverage scripts can discover the new service key

### Stage 1: Provider Operations

Implement:

- `Operations.List`

Why first:

- it establishes provider identity and is a low-risk read-only route
- it gives the example a simple first call before the score endpoints

### Stage 2: Spot Placement Score Metadata

Implement:

- `Spot Placement Scores.Get`

Behavior goals:

- validate subscription and location path inputs
- return deterministic metadata fixture for supported resource types

Why here:

- it is read-only and smaller than the generation route
- it clarifies the resource shape expected by the service before request-driven score generation

### Stage 3: Spot Placement Score Generation

Implement:

- `Spot Placement Scores.Post`

Behavior goals:

- validate the full request body
- produce deterministic score rows from:
  - requested locations
  - requested SKUs
  - zonal vs regional scope
  - requested count
- include a stable spread of scores such as:
  - `High`
  - `Medium`
  - `Low`
  - `DataNotFoundOrStale`
- set a stable `location` response header matching the documented pattern

Why here:

- this is the only request-driven route in the service
- it is the main contract that users are likely to exercise

### Stage 4: Tests, Example, and Coverage Finalization

Add:

- focused provider tests for:
  - operations success
  - metadata success
  - score-generation success
  - request-validation failures
- the runtime-based example under `examples/azure/recommenderrp-2025-06-05`
- final coverage aliases and doc mappings

## Example Scope

The future example should demonstrate the actual narrow surface:

1. list provider operations
2. fetch Spot Placement Score metadata for one region
3. submit one score-generation request for a small set of locations and SKUs
4. print the returned placement score rows

Recommended example files:

- `examples/azure/recommenderrp-2025-06-05/main.go`
- `examples/azure/recommenderrp-2025-06-05/Dockerfile`
- `examples/azure/recommenderrp-2025-06-05/docker-compose.yml`

## Coverage Script Updates

When implementation begins, update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add:

- alias normalization for `compute_recommender`
- plan-doc mapping to `recommenderrp-2025-06-05-plan.md`
- any example-path or service-name hints needed by the existing coverage tooling

## Validation Plan

After implementation, validate with:

- focused provider tests for `compute_recommender`
- example build for `examples/azure/recommenderrp-2025-06-05`
- Azure contract coverage for `--service compute_recommender`
- Azure IO coverage for `--service compute_recommender`
- Azure doc coverage for `--service compute_recommender`
- Azure endpoint coverage for `--service compute_recommender`
- full `make ci`, reporting unrelated repo failures separately if they remain

## Recommended Implementation Order

If implemented incrementally, merge in this order:

1. Stage 0 and Stage 1
2. Stage 2
3. Stage 3
4. Stage 4

This keeps the service narrow and verifiable, which matches the actual API surface reviewed from Learn.
