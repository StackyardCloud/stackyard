# Azure Desktop Virtualization (`desktopvirtualization_2025_10_10`) Staged Plan

## Objective

Emulate Azure Desktop Virtualization REST API (`api-version=2025-10-10`) with deterministic local behavior for:

- app attach package lifecycle and package-info import workflows
- host pool, workspace, application group, application, and desktop lifecycle flows
- session host and user session administration flows
- scaling plan, pooled schedule, and personal schedule lifecycle flows
- private link resource and private endpoint connection management flows
- provider operations discovery
- an Azure Go SDK example under `examples/azure/desktopvirtualization-2025-10-10`
- Azure coverage-script wiring for the new service key

## Primary References Reviewed

Requested top-level operation-groups page:

- `https://learn.microsoft.com/en-us/rest/api/desktopvirtualization/operation-groups?view=rest-desktopvirtualization-2025-10-10`

Operation-group pages reviewed:

- `.../app-attach-package?view=rest-desktopvirtualization-2025-10-10`
- `.../app-attach-package-info?view=rest-desktopvirtualization-2025-10-10`
- `.../application-groups?view=rest-desktopvirtualization-2025-10-10`
- `.../applications?view=rest-desktopvirtualization-2025-10-10`
- `.../desktops?view=rest-desktopvirtualization-2025-10-10`
- `.../host-pools?view=rest-desktopvirtualization-2025-10-10`
- `.../msix-packages?view=rest-desktopvirtualization-2025-10-10`
- `.../msix-images?view=rest-desktopvirtualization-2025-10-10`
- `.../operations?view=rest-desktopvirtualization-2025-10-10`
- `.../private-endpoint-connections?view=rest-desktopvirtualization-2025-10-10`
- `.../private-link-resources?view=rest-desktopvirtualization-2025-10-10`
- `.../scaling-plan-personal-schedules?view=rest-desktopvirtualization-2025-10-10`
- `.../scaling-plan-pooled-schedules?view=rest-desktopvirtualization-2025-10-10`
- `.../scaling-plans?view=rest-desktopvirtualization-2025-10-10`
- `.../session-hosts?view=rest-desktopvirtualization-2025-10-10`
- `.../start-menu-items/list?view=rest-desktopvirtualization-2025-10-10`
- `.../user-sessions?view=rest-desktopvirtualization-2025-10-10`
- `.../workspaces?view=rest-desktopvirtualization-2025-10-10`

Method pages reviewed directly (complete operation set, grouped):

- `app-attach-package/create-or-update`, `update`, `get`, `delete`, `list-by-resource-group`, `list-by-subscription`
- `app-attach-package-info/import`
- `application-groups/create-or-update`, `update`, `get`, `delete`, `list-by-resource-group`, `list-by-subscription`
- `applications/create-or-update`, `update`, `get`, `delete`, `list`
- `desktops/get`, `list`, `update`
- `host-pools/create-or-update`, `update`, `get`, `delete`, `list`, `list-by-resource-group`, `list-registration-tokens`, `retrieve-registration-token`
- `msix-packages/create-or-update`, `update`, `get`, `delete`, `list`
- `msix-images/expand`
- `operations/list`
- `private-endpoint-connections/get-by-host-pool`, `list-by-host-pool`, `get-by-workspace`, `list-by-workspace`, `update-by-host-pool`, `update-by-workspace`, `delete-by-host-pool`, `delete-by-workspace`
- `private-link-resources/list-by-host-pool`, `list-by-workspace`
- `scaling-plan-personal-schedules/create`, `update`, `get`, `list`, `delete`
- `scaling-plan-pooled-schedules/create`, `update`, `get`, `list`, `delete`
- `scaling-plans/create`, `update`, `get`, `delete`, `list-by-resource-group`, `list-by-subscription`, `list-by-host-pool`
- `session-hosts/get`, `list`, `update`, `delete`
- `start-menu-items/list`
- `user-sessions/get`, `list`, `list-by-host-pool`, `disconnect`, `send-message`, `delete`
- `workspaces/create-or-update`, `update`, `get`, `delete`, `list-by-resource-group`, `list-by-subscription`

SDK cross-check used for operation-group parity:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/desktopvirtualization/armdesktopvirtualization`

Notes on source behavior in this environment:

- several Learn pages intermittently render authorization/cache wrappers
- method route, request/response contracts, and error-shape sections remained readable and were used for contract planning

## Surface Summary

Reviewed operation groups and methods:

- App Attach Package (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- App Attach Package Info (1):
  - `Import`
- Application Groups (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`
- Applications (5):
  - `Create Or Update`, `Delete`, `Get`, `List`, `Update`
- Desktops (3):
  - `Get`, `List`, `Update`
- Host Pools (8):
  - `Create Or Update`, `Delete`, `Get`, `List`, `List By Resource Group`, `List Registration Tokens`, `Retrieve Registration Token`, `Update`
- MSIX Packages (5):
  - `Create Or Update`, `Delete`, `Get`, `List`, `Update`
- Msix Images (1):
  - `Expand`
- Operations (1):
  - `List`
- Private Endpoint Connections (8):
  - `Delete By Host Pool`, `Delete By Workspace`, `Get By Host Pool`, `Get By Workspace`, `List By Host Pool`, `List By Workspace`, `Update By Host Pool`, `Update By Workspace`
- Private Link Resources (2):
  - `List By Host Pool`, `List By Workspace`
- Scaling Plan Personal Schedules (5):
  - `Create`, `Delete`, `Get`, `List`, `Update`
- Scaling Plan Pooled Schedules (5):
  - `Create`, `Delete`, `Get`, `List`, `Update`
- Scaling Plans (7):
  - `Create`, `Delete`, `Get`, `List By Host Pool`, `List By Resource Group`, `List By Subscription`, `Update`
- Session Hosts (4):
  - `Delete`, `Get`, `List`, `Update`
- Start Menu Items (1):
  - `List`
- User Sessions (6):
  - `Delete`, `Disconnect`, `Get`, `List`, `List By Host Pool`, `Send Message`
- Workspaces (6):
  - `Create Or Update`, `Delete`, `Get`, `List By Resource Group`, `List By Subscription`, `Update`

Total reviewed methods: `80`.

## Service Naming

Canonical Stackyard naming:

- service key: `desktopvirtualization_2025_10_10`
- display name: `Azure Desktop Virtualization`
- API version: `2025-10-10`

Recommended repo paths:

- implementation: `internal/server/provider_azure_desktopvirtualization_2025_10_10.go`
- tests: `internal/server/provider_azure_desktopvirtualization_2025_10_10_test.go`
- example: `examples/azure/desktopvirtualization-2025-10-10`
- plan doc: `docs/desktopvirtualization-2025-10-10-plan.md`

## Route Envelope

Management-plane route families to emulate:

- `/azure/providers/Microsoft.DesktopVirtualization/operations`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.DesktopVirtualization/...`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DesktopVirtualization/...`

Representative routes:

- `GET /azure/providers/Microsoft.DesktopVirtualization/operations?api-version=2025-10-10`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/hostPools/{hostPoolName}?api-version=2025-10-10`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/hostPools/{hostPoolName}/listRegistrationTokens?api-version=2025-10-10`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/hostPools/{hostPoolName}/retrieveRegistrationToken?api-version=2025-10-10`
- `POST /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/hostPools/{hostPoolName}/importAppAttachPackageInfo?api-version=2025-10-10`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/applicationGroups/{applicationGroupName}?api-version=2025-10-10`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/applicationGroups/{applicationGroupName}/applications/{applicationName}?api-version=2025-10-10`
- `PATCH /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/applicationGroups/{applicationGroupName}/desktops/{desktopName}?api-version=2025-10-10`
- `GET /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/workspaces/{workspaceName}/privateEndpointConnections?api-version=2025-10-10`
- `PUT /azure/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.DesktopVirtualization/workspaces/{workspaceName}/privateEndpointConnections/{privateEndpointConnectionName}?api-version=2025-10-10`

## Contract Notes From Reviewed Method Pages

Cross-cutting traits:

- all reviewed methods use ARM OAuth2 (`azure_auth`, `user_impersonation`)
- API version is required on all routes (`api-version=2025-10-10`)
- list methods use `value` plus optional `nextLink`
- error envelopes are split by family:
  - many Desktop Virtualization methods use `CloudError`
  - provider operations and some app-attach/private-endpoint methods use `ErrorResponse`

Observed response patterns:

- `PUT` create/update methods generally return `200`/`201`
- `PATCH` update methods return `200`
- action methods (`importAppAttachPackageInfo`, `expandMsixImage`, registration-token actions, user-session actions) return `200`
- delete methods vary by group and commonly return `200`/`204`

Notable payload considerations:

- App Attach import request body includes `path` and `packageArchitecture`
- host pool create/update requires coherent pool type and load-balancer settings
- scaling-plan schedules include day/time windows and ramp-up/down policy fields
- private endpoint connection update uses a connection-state payload (`status`, `description`, `actionsRequired`)
- list-by-host-pool user sessions accepts optional filtering/paging fields

## Staged Implementation Plan

### Stage 0: Coverage, Planning, and Skeleton

- add this plan doc
- add aliases for `desktopvirtualization_2025_10_10` in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `SERVICE_PLAN_DOCS`
- add stage-0 provider skeleton + router wiring

Exit criteria:

- coverage scripts resolve service key and aliases
- doc coverage maps service key to this plan
- Azure router dispatches Desktop Virtualization routes to provider stub

### Stage 1: Router Ownership and API-Version Gate

Implement:

- route ownership for `/providers/Microsoft.DesktopVirtualization/` families
- strict `api-version=2025-10-10` validation
- deterministic `400` for missing/invalid API version
- deterministic `405` for unsupported methods on owned routes

Exit criteria:

- Desktop Virtualization route families are deterministically owned
- version/method errors are stable and testable

### Stage 2: Core Resource Lifecycles

Implement:

- Host Pools (8)
- Workspaces (6)
- Application Groups (6)
- Applications (5)
- Desktops (3)
- Provider Operations (`List`)

Behavior:

- canonical ARM envelopes (`id`, `name`, `type`, `location`, `properties`, `tags`, `systemData`)
- deterministic list ordering/paging
- idempotent delete semantics by resource type

Exit criteria:

- core AVD resource CRUD and read surfaces are available

### Stage 3: App Attach and MSIX Surfaces

Implement:

- App Attach Package (6)
- App Attach Package Info (`Import`)
- MSIX Packages (5)
- Msix Images (`Expand`)
- Start Menu Items (`List`)

Behavior:

- deterministic package import/expand projections from request inputs
- stable package/application list response fixtures
- consistent `CloudError`/`ErrorResponse` shaping on invalid package requests

Exit criteria:

- app attach and MSIX workflows are callable end-to-end

### Stage 4: Private Connectivity Surfaces

Implement:

- Private Endpoint Connections (8)
- Private Link Resources (2)

Behavior:

- deterministic connection-state machine (`Pending`/`Approved`/`Rejected`)
- hostpool/workspace scoped list/get parity
- idempotent delete/update behaviors by scope

Exit criteria:

- private connectivity surfaces are modeled for host pools and workspaces

### Stage 5: Session and Scale Operations

Implement:

- Session Hosts (4)
- User Sessions (6)
- Scaling Plans (7)
- Scaling Plan Personal Schedules (5)
- Scaling Plan Pooled Schedules (5)

Behavior:

- deterministic session state fixtures (`Active`, `Disconnected`, `LogOff`)
- action behavior for `disconnect`/`sendMessage`
- stable schedule readback and update semantics

Exit criteria:

- scale-control and session-management surfaces return contract-conformant payloads

### Stage 6: Tests and Contract Hardening

Add `internal/server/provider_azure_desktopvirtualization_2025_10_10_test.go` for:

- route ownership, method gating, and api-version enforcement
- happy-path and negative-path payload validation across each operation group
- list paging and deterministic ordering assertions
- error envelope shape parity (`CloudError` and `ErrorResponse` families)

Exit criteria:

- deterministic provider behavior is covered with focused tests

### Stage 7: Example and Coverage Enforcement

- keep example aligned to all reviewed operation groups
- enforce service discovery through coverage scripts
- keep strict gates until implementation/test stages are complete

Exit criteria:

- example compiles and demonstrates broad service coverage
- coverage scripts discover service key and plan-doc mapping

## Coverage Script Updates

Add aliases for `desktopvirtualization_2025_10_10` to:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Minimum alias set:

- `desktopvirtualization_2025_10_10`
- `desktopvirtualization-2025-10-10`
- `desktopvirtualization`
- `desktop_virtualization`
- `desktop-virtualization`
- `azure_desktopvirtualization`
- `azure_virtual_desktop`
- `microsoft_desktopvirtualization`
- `rest_desktopvirtualization_2025_10_10`

## Example Scope

Example path:

- `examples/azure/desktopvirtualization-2025-10-10`

Example should:

- use Azure Go SDK core pipeline (`azcore/runtime`, `azcore/policy`) for HTTP execution
- execute representative calls across all operation groups
- print response status and JSON payload for each request
- include cleanup calls for delete-capable resources

## Validation Commands

- `go build ./examples/azure/desktopvirtualization-2025-10-10`
- `python3 scripts/azure-contract-coverage.py --service desktopvirtualization-2025-10-10 --require-service desktopvirtualization_2025_10_10 --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service desktopvirtualization-2025-10-10 --require-service desktopvirtualization_2025_10_10 --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service desktopvirtualization-2025-10-10 --require-service desktopvirtualization_2025_10_10 --fail-on any`
