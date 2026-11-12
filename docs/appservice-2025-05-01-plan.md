# Azure App Service Resource Manager (`appservice-2025-05-01`) Staged Plan

## Objective

Emulate Azure App Service Resource Manager (`2025-05-01`) with deterministic local behavior for the highest-value management-plane workflows first, then widen the surface by operation-group family until coverage and example execution are complete.

This Microsoft Learn surface is materially larger than the other Azure services already emulated in Stackyard. The root page exposes 42 operation groups, with `Web Apps` and `Static Sites` carrying the majority of the resource lifecycle and nested child-resource surface. The staged plan below is therefore intentionally incremental.

Primary references reviewed:
- `https://learn.microsoft.com/en-us/rest/api/appservice/operation-groups?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/provider?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/provider/list-operations?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-plans?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-plans/create-or-update?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-plans/get?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/certificates?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/certificates/create-or-update?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/app-service-environments?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/kube-environments?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/create-or-update?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/get?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/web-apps/create-or-update-configuration?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/static-sites?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/static-sites/create-or-update-static-site?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/workflows?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/workflows/regenerate-access-key?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/workflow-runs?view=rest-appservice-2025-05-01`
- `https://learn.microsoft.com/en-us/rest/api/appservice/workflow-runs/cancel?view=rest-appservice-2025-05-01`

## Service Naming

The request called this “Azure Account Management”, but the supplied documentation is for Azure App Service Resource Manager under the `Microsoft.Web` provider. To keep implementation, example, and coverage naming consistent with the API reference, this plan targets:

- service key: `appservice`
- display name: `Azure App Service Resource Manager`
- API version: `2025-05-01`

Recommended repo naming:
- implementation file: `internal/server/provider_azure_appservice.go`
- tests: `internal/server/provider_azure_appservice_test.go`
- strict-signal tests if needed: `internal/server/provider_azure_appservice_strict_signals_test.go`
- example path: `examples/azure/app-service/resource-manager-2025-05-01`
- plan doc: `docs/appservice-2025-05-01-plan.md`

## Route Envelope

Stackyard route envelope for this service:
- `/azure/providers/Microsoft.Web/*`
- `/azure/subscriptions/{subscriptionId}/providers/Microsoft.Web/*`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/*`

Target API version:
- `api-version=2025-05-01`

Representative ARM resource paths from the reviewed method pages:
- provider operations: `/providers/Microsoft.Web/operations`
- app service plans: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/serverfarms/{name}`
- certificates: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/certificates/{name}`
- web apps: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/sites/{name}`
- static sites: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/staticSites/{name}`
- workflow runtime management: `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Web/sites/{name}/hostruntime/runtime/webhooks/workflow/api/management/...`

## API Surface Inventory

Operation groups exposed from the root page:
- `App Service Environments`
- `App Service Plans`
- `Certificates`
- `Check Name Availability`
- `Deleted Web Apps`
- `Diagnostics`
- `Get Publishing User`
- `Get Source Control`
- `Get Subscription Deployment Locations`
- `Get Usages In Location`
- `Global`
- `Kube Environments`
- `List Ase Regions`
- `List Billing Meters`
- `List Custom Host Name Sites`
- `List Geo Regions`
- `List Premier Add On Offers`
- `List Site Identifiers Assigned To Host Name`
- `List Skus`
- `List Source Controls`
- `Move`
- `Provider`
- `Recommendations`
- `Regional Check Name Availability`
- `Resource Health Metadata`
- `Site Certificates`
- `Static Sites`
- `Update Publishing User`
- `Update Source Control`
- `Validate`
- `Validate Move`
- `Verify Hosting Environment Vnet`
- `Web Apps`
- `Workflow Run Action Repetitions`
- `Workflow Run Action Repetitions Request Histories`
- `Workflow Run Action Scope Repetitions`
- `Workflow Run Actions`
- `Workflow Runs`
- `Workflow Trigger Histories`
- `Workflow Triggers`
- `Workflow Versions`
- `Workflows`

High-level contract characteristics from the reviewed method pages:
- The surface is ARM management-plane first and consistently uses `api-version` plus AAD auth.
- Resource create/update methods are generally `PUT` and commonly return `200` or `202`.
- Read/list methods are generally `GET` and return typed resource or collection payloads.
- Action methods use `POST` and often return `200`, `202`, or `204`.
- Typed default errors consistently reference `Default Error Response` or `Error Response`.
- The Go examples on Microsoft Learn use `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6`.

## Non-Goals

- Do not implement the full `Web Apps` and `Static Sites` matrix in the initial stage.
- Do not emulate live runtime behavior of Kudu, MSDeploy, workflow execution, certificate issuance, or network capture internals.
- Do not require live Azure credentials or real `Microsoft.Web` backends.
- Do not expand beyond management-plane `2025-05-01` unless the example or SDK generator forces a minimal compatibility shim.

## Stage 0: Contract Skeleton And Naming

- Add a dedicated Azure App Service router that recognizes the `Microsoft.Web` provider envelope.
- Normalize service aliases in `azure-contract-coverage.py`, `azure-io-contract-coverage.py`, and `azure-doc-contract-coverage.py`:
  - `appservice`
  - `app_service`
  - `appservice_2025_05_01`
  - `app_service_2025_05_01`
  - `azure_appservice`
  - `azure_appservice_2025_05_01`
- Add a plan-document mapping so doc-contract coverage resolves the new service key.
- Add route-recognition tests for provider, serverfarms, certificates, sites, staticSites, and workflow-management subtrees.

Exit criteria:
- Coverage scripts discover the new service key.
- Provider router claims `Microsoft.Web` paths deterministically.

## Stage 1: Foundation Read-Only Surface

- Implement deterministic success fixtures for the low-risk read/list groups first:
  - `Provider`
  - `Check Name Availability`
  - `Regional Check Name Availability`
  - `Get Usages In Location`
  - `Get Subscription Deployment Locations`
  - `List Geo Regions`
  - `List Skus`
  - `List Billing Meters`
  - `List Ase Regions`
  - `List Source Controls`
  - `Get Source Control`
  - `Get Publishing User`
  - `Resource Health Metadata`
  - `Recommendations`
- Return ARM-style collection envelopes with `value` arrays and optional `nextLink`.
- Add validation for missing or empty `api-version`, malformed subscription/resource-group names, and unsupported verbs.

Exit criteria:
- Read-only groups return typed deterministic payloads.
- Azure contract and IO coverage report implemented validation and success signals for the foundation slice.

## Stage 2: Core Resource Families

- Add CRUD/list/get coverage for the main ARM resources that other groups depend on:
  - `App Service Plans` (`serverfarms`)
  - `Certificates`
  - `Site Certificates`
  - `App Service Environments`
  - `Kube Environments`
- Model the minimum reusable resource store for:
  - plans
  - certificates
  - hosting environments
  - kube environments
  - child resources such as worker pools, routes, hybrid connections, and private endpoint connection metadata where contract shape depends on them
- Support representative action methods from the reviewed pages:
  - plan create/update/get/list
  - certificate create/update/get/list/delete
  - ASE get/list/create/update/delete and private endpoint related reads

Exit criteria:
- The reusable state layer exists for `Microsoft.Web` top-level resource families.
- Core resource groups no longer fall back to `NotImplemented`.

## Stage 3: Web Apps Core

- Implement the minimal but stable `Web Apps` subset that unlocks useful examples and broad coverage:
  - create or update app
  - get app
  - delete app
  - list by subscription/resource group
  - create or update configuration
  - get configuration
  - source control get/update
  - host name binding get/create
  - private endpoint connection get/list
  - publishing credential policy reads
- Add deterministic child-resource handling for:
  - `/config/web`
  - `/config/appsettings`
  - `/sourcecontrols/web`
  - `/hostNameBindings/{binding}`
  - `/privateEndpointConnections/*`
- Keep Kudu/MSDeploy/process/runtime endpoints explicitly scoped out until later stages, but return typed validation errors instead of generic `NotImplemented` where feasible.

Exit criteria:
- A typed SDK example can create, read, and configure a web app against Stackyard.
- The core `sites` CRUD and config paths are covered by tests and no longer report `NotImplemented`.

## Stage 4: Static Sites

- Implement the `Static Sites` family as a separate stateful resource tree after the `sites` core is stable.
- Prioritize:
  - create/update static site
  - get static site
  - delete static site
  - list static sites
  - app settings and function app settings
  - custom domain create/get/delete
  - private endpoint connection get/list
- Add typed but deterministic responses for invitation links, zip deployment submission, and database connection stubs.

Exit criteria:
- `staticSites` CRUD and primary child-resource operations are implemented and covered.
- Example can exercise both classic app and static site clients without live Azure.

## Stage 5: Workflow Management Surface

- Implement the Logic Apps Standard management endpoints hosted under the App Service runtime path:
  - `Workflows`
  - `Workflow Versions`
  - `Workflow Triggers`
  - `Workflow Trigger Histories`
  - `Workflow Runs`
  - `Workflow Run Actions`
  - repetition/history groups
- Start with read/list/cancel/resubmit/regenerate-key style actions from the reviewed method pages.
- Model workflows as child resources under a site so they can share `sites/{name}` ownership.

Exit criteria:
- Workflow management endpoints return typed ARM/runtime payloads.
- Action methods like workflow run cancel and workflow access-key regeneration stop returning `NotImplemented`.

## Stage 6: Validation, Move, And Diagnostics Tail

- Implement the long-tail operation groups that are mostly action or validation endpoints:
  - `Validate`
  - `Validate Move`
  - `Move`
  - `Diagnostics`
  - `Verify Hosting Environment Vnet`
  - `Deleted Web Apps`
  - `Global`
- Favor stable, low-entropy success payloads and deterministic error fixtures over deep backend simulation.

Exit criteria:
- Tail groups are recognized and return typed deterministic responses.
- Azure endpoint coverage stops surfacing `NotImplemented` for known `Microsoft.Web` paths in the selected stage scope.

## Stage 7: Example And Coverage Completion

- Add `examples/azure/app-service/resource-manager-2025-05-01` using the Azure Go SDK package shown in the docs:
  - `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v6`
- Follow the existing Azure example pattern:
  - `azsdkshim.StaticTokenCredential{}`
  - `arm.ClientOptions` with `cloud.ResourceManager` pointed at Stackyard
  - `azsdkshim.NewTransport("/azure", account, subscriptionKey)`
- Example should cover a representative slice instead of the full service:
  - provider operations
  - app service plan create/get/list
  - certificate create/get/list
  - web app create/get/config/list
  - static site create/get/list
  - workflow run cancel or workflow key regeneration if the chosen SDK clients exist in `armappservice/v6`
- Add `Dockerfile` and `docker-compose.yml` alongside the example.

Exit criteria:
- Example compiles and runs against Stackyard in local emulation mode.
- `scripts/run-all-examples.sh` picks up the new example automatically through the standard examples layout.

## Coverage Wiring Requirements

- Update `scripts/azure-contract-coverage.py` aliases and file mapping for `appservice`.
- Update `scripts/azure-io-contract-coverage.py` aliases and file mapping for `appservice`.
- Update `scripts/azure-doc-contract-coverage.py` aliases and plan-doc mapping for `appservice`.
- Ensure the implementation file and tests expose the validation/output/negative-signal patterns those scripts expect.

Representative files that will need updates:
- `internal/server/provider_azure_appservice.go`
- `internal/server/provider_azure_appservice_test.go`
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`
- `examples/azure/app-service/resource-manager-2025-05-01/main.go`
- `examples/azure/app-service/resource-manager-2025-05-01/Dockerfile`
- `examples/azure/app-service/resource-manager-2025-05-01/docker-compose.yml`

## Implementation Risks

- `Web Apps` is by far the largest operation group and should not be treated as a single implementation step.
- Several methods under `Web Apps` are Kudu/runtime/process-management flavored and should be explicitly postponed rather than partially faked without contract clarity.
- Workflow management endpoints live under nested runtime webhook paths, so route matching must not accidentally over-capture unrelated `sites/*` children.
- Static Sites has enough secondary resource types that it merits its own state store namespace instead of being folded into generic web app state.

## Recommended First Implementation Slice

If implementation starts immediately after planning, the first mergeable slice should be:
- Stage 0
- Stage 1
- Stage 2 for `App Service Plans`, `Certificates`, and `Provider`
- Stage 3 for `Web Apps` core CRUD/list/get/config only
- Stage 7 example limited to provider + plans + core web app

That slice is small enough to verify with the Azure coverage scripts, but broad enough to establish the `Microsoft.Web` route model and the `armappservice/v6` example pattern before taking on `Static Sites` and workflow-management endpoints.
