# Azure Fleet (`fleet_2025_03_01`) Staged Plan

## Objective

Emulate Azure Fleet Resource Manager REST API (`api-version=2025-03-01`) with deterministic local behavior for:

- fleet lifecycle and credentials surfaces
- fleet member lifecycle and grouping surfaces
- fleet update strategy lifecycle surfaces
- auto-upgrade profile lifecycle and update-run generation surfaces
- update-run lifecycle and action surfaces (`start`, `stop`, `skip`)
- provider operations discovery
- an Azure Go SDK example under `examples/azure/fleet-2025-03-01`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/fleet/operation-groups?view=rest-fleet-2025-03-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profile-operations?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profiles?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-members?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-update-strategies?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/operations?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs?view=rest-fleet-2025-03-01`

Method pages reviewed directly: all linked methods from all operation groups (`29` total).

Auto Upgrade Profile Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profile-operations/generate-update-run?view=rest-fleet-2025-03-01`

Auto Upgrade Profiles methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profiles/create-or-update?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profiles/delete?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profiles/get?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/auto-upgrade-profiles/list-by-fleet?view=rest-fleet-2025-03-01`

Fleet Members methods (`5`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-members/create?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-members/delete?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-members/get?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-members/list-by-fleet?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-members/update?view=rest-fleet-2025-03-01`

Fleet Update Strategies methods (`4`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-update-strategies/create-or-update?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-update-strategies/delete?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-update-strategies/get?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleet-update-strategies/list-by-fleet?view=rest-fleet-2025-03-01`

Fleets methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/create-or-update?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/delete?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/get?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/list-by-resource-group?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/list-by-subscription?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/list-credentials?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/fleets/update?view=rest-fleet-2025-03-01`

Operations methods (`1`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/operations/list?view=rest-fleet-2025-03-01`

Update Runs methods (`7`):

- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/create-or-update?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/delete?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/get?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/list-by-fleet?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/skip?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/start?view=rest-fleet-2025-03-01`
- `https://learn.microsoft.com/en-us/rest/api/fleet/update-runs/stop?view=rest-fleet-2025-03-01`

SDK cross-check used:

- official Fleet method pages include Go SDK samples with:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservicefleet/armcontainerservicefleet/v2`

## Surface Summary

Reviewed operation groups and methods:

- Auto Upgrade Profile Operations (1)
- Auto Upgrade Profiles (4)
- Fleet Members (5)
- Fleet Update Strategies (4)
- Fleets (7)
- Operations (1)
- Update Runs (7)

Total reviewed methods: `29`.

## Service Naming

Canonical Stackyard naming:

- service key: `fleet_2025_03_01`
- display name: `Azure Fleet`
- API version: `2025-03-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_fleet_2025_03_01.go`
- tests: `internal/server/provider_azure_fleet_2025_03_01_test.go`
- example: `examples/azure/fleet-2025-03-01`
- plan doc: `docs/fleet-2025-03-01-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.ContainerService/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.ContainerService/fleets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/members`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateStrategies`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns`

Representative routes:

- `GET /azure/providers/Microsoft.ContainerService/operations?api-version=2025-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}?api-version=2025-03-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/listCredentials?api-version=2025-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/members/{fleetMemberName}?api-version=2025-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateStrategies/{updateStrategyName}?api-version=2025-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles/{autoUpgradeProfileName}?api-version=2025-03-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/autoUpgradeProfiles/{autoUpgradeProfileName}/generateUpdateRun?api-version=2025-03-01`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}?api-version=2025-03-01`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ContainerService/fleets/{fleetName}/updateRuns/{updateRunName}/start?api-version=2025-03-01`

## Contract Notes From Reviewed Pages

Cross-cutting traits:

- all reviewed methods require `api-version=2025-03-01`
- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- all reviewed methods define `Other Status Codes` with `ErrorResponse`

Observed response patterns (from all reviewed methods):

- `200 OK` appears in `28` methods
- `201 Created` appears in `5` methods
- `202 Accepted` appears in `11` methods
- `204 No Content` appears in `5` methods

Request-body coverage from reviewed pages:

- `8` methods include request bodies
- body-bearing methods:
  - `auto-upgrade-profiles/create-or-update`
  - `fleet-members/create`
  - `fleet-members/update`
  - `fleet-update-strategies/create-or-update`
  - `fleets/create-or-update`
  - `fleets/update`
  - `update-runs/create-or-update`
  - `update-runs/skip`

LRO-oriented operations and headers:

- several create/update/delete/action methods document async pathways using `202` and headers such as `Location`, `Retry-After`, and/or `Azure-AsyncOperation`

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add `fleet_2025_03_01` aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider router scaffold in `internal/server/provider_azure_fleet_2025_03_01.go`
- wire router dispatch in `internal/server/http.go`

Exit criteria:

- service key resolves via all coverage scripts
- docs coverage resolves to this plan
- Fleet route families are claimed by a dedicated provider router scaffold

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for Fleet path families under `Microsoft.ContainerService/fleets`
- strict `api-version=2025-03-01` validation on owned routes
- deterministic `400` for missing/invalid version on owned routes
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Fleet routes are no longer shadowed by generic ContainerService handling
- version/method failures are stable and testable

### Stage 2: Operations + Read/List Surfaces

Implement:

- `operations/list`
- `fleets/list-by-subscription`
- `fleets/list-by-resource-group`
- `fleets/get`
- `fleet-members/list-by-fleet`, `fleet-members/get`
- `fleet-update-strategies/list-by-fleet`, `fleet-update-strategies/get`
- `auto-upgrade-profiles/list-by-fleet`, `auto-upgrade-profiles/get`
- `update-runs/list-by-fleet`, `update-runs/get`
- `fleets/list-credentials`

Exit criteria:

- all read/list surfaces return deterministic ARM envelopes and list paging shape (`value`, optional `nextLink`)

### Stage 3: Lifecycle Resources (Fleet, Members, Strategies, Profiles)

Implement:

- Fleet lifecycle: `create-or-update`, `update`, `delete`
- Fleet Member lifecycle: `create`, `update`, `delete`
- Fleet Update Strategy lifecycle: `create-or-update`, `delete`
- Auto Upgrade Profile lifecycle: `create-or-update`, `delete`

Behavior:

- deterministic resource identity, `systemData`, `provisioningState`
- idempotent deletes with documented `200`/`202`/`204` behavior
- predictable async headers for accepted operations

### Stage 4: Update Run Actions and Sequencing

Implement:

- Update Run lifecycle: `create-or-update`, `delete`
- Update Run actions: `start`, `stop`, `skip`
- AutoUpgradeProfile action: `generate-update-run`

Behavior:

- deterministic run state transitions (`Pending`/`Running`/`Succeeded`/`Stopped`/`Skipped`-style modeling)
- route-level validation for required action payloads (notably `skip.targets`)
- deterministic action/idempotency semantics across repeated calls

### Stage 5: Tests and Contract Hardening

Add provider tests for:

- route ownership and api-version gating
- per-group happy paths for all 29 methods
- required request-body validation for 8 body-bearing methods
- delete/action async status handling and headers
- list ordering and paging determinism

Exit criteria:

- `internal/server/provider_azure_fleet_2025_03_01_test.go` covers stage 1-4 behavior
- Fleet passes strict contract/IO/doc coverage checks

### Stage 6: Example and Validation

Add `examples/azure/fleet-2025-03-01`:

- `main.go` using Azure Go SDK `azcore/runtime` pipeline
- `Dockerfile`
- `docker-compose.yml`

Validation:

- `go build ./examples/azure/fleet-2025-03-01`
- `python3 scripts/azure-contract-coverage.py --service fleet --require-service fleet_2025_03_01`
- `python3 scripts/azure-io-contract-coverage.py --service fleet --require-service fleet_2025_03_01`
- `python3 scripts/azure-doc-contract-coverage.py --service fleet --require-service fleet_2025_03_01`

