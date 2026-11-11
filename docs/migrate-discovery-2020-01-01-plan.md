# Azure Migrate Discovery (`migrate-discovery-2020-01-01`) Staged Plan

## Objective

Emulate the Azure Migrate Discovery ARM management API for `2020-01-01` with deterministic local behavior for:

- provider `operations`
- VMware discovery site lifecycle under `Microsoft.OffAzure/VMwareSites`
- Hyper-V discovery site lifecycle under `Microsoft.OffAzure/HyperVSites`
- nested inventory and credential families such as `machines`, `runAsAccounts`, `vCenters`, `hosts`, and `clusters`
- action routes such as site `refresh`, site `summary`, site health summary, and machine `start` / `stop`
- an Azure Go SDK example under `examples/azure/migrate-discovery-2020-01-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure Migrate Discovery under `Microsoft.OffAzure`.
- This is not the Azure Migrate Assessment API and not a migration appliance data plane.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2020-01-01`.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/operation-groups?view=rest-migrate-discovery-2020-01-01`

Representative operation-group and method pages reviewed across the surface:
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/operations/list?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/sites/put-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/sites/get-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/sites/patch-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/sites/get-site-usage?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-sites/get-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-sites/put-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-sites/get-site-usage?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-sites/refresh-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/machines?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/machines/get-machine?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/machines/get-all-machines-in-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/machines/start-machine?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/run-as-accounts/get-run-as-account?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/run-as-accounts/get-all-run-as-accounts-in-site?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-run-as-accounts/get-run-as-account?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-host/get-host?view=rest-migrate-discovery-2020-01-01`
- `https://learn.microsoft.com/en-us/rest/api/migrate/discovery/hyper-v-cluster/get-cluster?view=rest-migrate-discovery-2020-01-01`

SDK selection note:
- The official package Microsoft currently exposes for this API family is the older track-1 `migrate` management client.
- It is deprecated upstream, but it is still the closest official Azure Go SDK match for the documented `2020-01-01` Discovery surface.
- Plan assumption: prefer the typed track-1 clients where available, and use raw `autorest` request construction only for route gaps if the generated package does not expose a reviewed operation directly.

## Service Naming

Canonical Stackyard naming:
- service key: `migrate_discovery`
- display name: `Azure Migrate Discovery`
- provider namespace: `Microsoft.OffAzure`
- API version: `2020-01-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_migrate_discovery.go`
- tests: `internal/server/provider_azure_migrate_discovery_test.go`
- example: `examples/azure/migrate-discovery-2020-01-01`
- plan doc: `docs/migrate-discovery-2020-01-01-plan.md`

Recommended coverage aliases:
- `migrate_discovery`
- `migrate-discovery`
- `azure_migrate_discovery`
- `microsoft_offazure`
- `offazure`
- `rest_migrate_discovery_2020_01_01`
- `migrate_discovery_2020_01_01`

## Surface Summary

The reviewed `2020-01-01` management surface is centered on two parallel discovery trees:

1. `Operations`
2. `Sites` for VMware discovery
3. `Machines` under VMware sites
4. `Run As Accounts` under VMware sites
5. `Hyper V Sites`
6. `Hyper V Machines`
7. `Hyper V Run As Accounts`
8. `Hyper V Host`
9. `Hyper V Cluster`

The site families are the core of the surface. Both provider trees follow the same overall shape:

- top-level site resource
- site mutation routes
- site summary or health routes
- nested discovery inventory
- small action-style routes that often return `202 Accepted`

The API is much narrower than Migrate Assessment, but it still benefits from staged implementation because VMware and Hyper-V resources have similar structure with different type names and nested children.

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2020-01-01`
- all routes sit under the `Microsoft.OffAzure` provider namespace

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.OffAzure/operations?api-version=2020-01-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}?api-version=2020-01-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}?api-version=2020-01-01`
- `PATCH /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}?api-version=2020-01-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}/summary?api-version=2020-01-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}/refresh?api-version=2020-01-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}/machines?api-version=2020-01-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}/runAsAccounts?api-version=2020-01-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/HyperVSites/{siteName}?api-version=2020-01-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/HyperVSites/{siteName}/hosts/{hostName}?api-version=2020-01-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/HyperVSites/{siteName}/clusters/{clusterName}?api-version=2020-01-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}/machines/{machineName}/start?api-version=2020-01-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.OffAzure/VMwareSites/{siteName}/machines/{machineName}/stop?api-version=2020-01-01`

Important routing implications from the reviewed pages:
- resource type casing is significant in response payloads even if inbound matching is case-insensitive
- VMware and Hyper-V paths are structurally parallel but not identical
- several action routes are modeled as asynchronous ARM operations with `202 Accepted`
- nested resource names are effectively opaque ARM child names and should be stored canonically rather than normalized into display fields

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style error payloads
- list routes return `value`
- top-level and child resources return standard ARM fields:
  - `id`
  - `name`
  - `type`
  - `location` for sites
  - `tags` where applicable
  - `properties.*`

### Operations

`Operations.List`
- route: `/providers/Microsoft.OffAzure/operations`
- returns `OperationResultList`
- sample payload includes:
  - `name`
  - `display.provider`
  - `display.resource`
  - `display.operation`
  - `display.description`
  - `origin`

Modeling implication:
- return a deterministic operation list covering each implemented resource family and action route

### VMware Sites

Reviewed method pages:
- `Put Site`
- `Get Site`
- `Patch Site`
- `Get Site Usage`

Key traits from the reviewed pages:
- route root is `/providers/Microsoft.OffAzure/VMwareSites/{siteName}`
- `PUT` requires `location`
- response model includes:
  - `servicePrincipalIdentityDetails`
  - `agentDetails`
  - `serviceEndpoint`
  - `discoverySolutionId`
  - `applianceName`
- summary route is a `POST` returning a compact usage document

Modeling implications:
- persist VMware sites as first-class ARM resources
- synthesize stable agent/SPN details so nested resources can reference their parent site cleanly
- support `HEAD` alongside `GET` for standard ARM coverage expectations

### Hyper-V Sites

Reviewed method pages:
- `Put Site`
- `Get Site`
- `Get Site Usage`
- `Refresh Site`

Key traits from the reviewed pages:
- route root is `/providers/Microsoft.OffAzure/HyperVSites/{siteName}`
- top-level shape mirrors VMware site responses
- summary includes `machineCount`, `runAsAccountCount`, `hostCount`, and `clusterCount`
- refresh route returns `202 Accepted`

Modeling implications:
- reuse a shared site-storage pattern with provider-specific type strings and summary fields
- model refresh as an accepted async-style success, with deterministic operation headers if needed by coverage

### Machines

Reviewed method pages:
- `Get All Machines In Site`
- `Get Machine`
- `Start Machine`

Key traits from the reviewed pages:
- VMware machine routes sit under `/VMwareSites/{siteName}/machines/{machineName}`
- Hyper-V machine routes sit under `/HyperVSites/{siteName}/machines/{machineName}`
- machine payloads are large, but the contract-critical parts are:
  - `id`
  - `name`
  - `type`
  - `properties.displayName`
  - `properties.powerStatus`
  - `properties.operatingSystemDetails`
  - `properties.disks`
  - `properties.networkAdapters`
  - `properties.errors`
- start and stop are asynchronous `POST` actions

Modeling implications:
- seed a small deterministic machine collection when a site is created
- expose realistic but compact machine properties rather than trying to fully mirror every optional field
- model `start` and `stop` as state transitions with `202 Accepted`

### Run As Accounts

Reviewed method pages:
- VMware run-as account get/list
- Hyper-V run-as account get

Key traits from the reviewed pages:
- routes are nested under the relevant site family
- list returns a collection
- payload type names differ between VMware and Hyper-V

Modeling implications:
- keep provider-specific type strings but reuse a single internal credential shape
- auto-seed one run-as account per site to satisfy list/get coverage without extra setup

### Hyper-V Hosts and Clusters

Reviewed method pages:
- `Hyper V Host - Get Host`
- `Hyper V Cluster - Get Cluster`

Key traits from the reviewed pages:
- these resources are read-only inventory under a Hyper-V site
- routes are nested under `/hosts/{hostName}` and `/clusters/{clusterName}`

Modeling implications:
- treat hosts and clusters as deterministic fixtures derived from site creation
- implement list routes if they are present in the operation groups, but prioritize `get` if coverage starts from linked methods first

## Recommended Stages

### Stage 0: Wiring and Discovery Skeleton

Goal:
- make the new service discoverable in coverage and reserve the route family

Work:
- add `migrate_discovery` aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- add a provider entry point in `internal/server/http.go`
- add an empty focused test file scaffold
- add the plan doc itself

Exit criteria:
- coverage scripts recognize `migrate_discovery`
- requests under `/azure/.../providers/Microsoft.OffAzure/...` no longer fall through to generic not-found once the provider is wired

### Stage 1: Operations + VMware Site Foundation

Goal:
- cover the smallest useful vertical slice of the provider

Work:
- implement `GET /providers/Microsoft.OffAzure/operations`
- implement VMware site:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
- implement VMware site action routes:
  - `POST .../summary`
  - `POST .../refresh`
  - site health summary if present in the linked methods
- persist canonical VMware site state in memory

Exit criteria:
- provider discovery and the full VMware site lifecycle are green

### Stage 2: VMware Nested Inventory

Goal:
- cover the most likely contract paths under `VMwareSites`

Work:
- implement VMware:
  - `machines` list/get
  - `machines/{machineName}/start`
  - `machines/{machineName}/stop`
  - `runAsAccounts` list/get
  - `vCenters` get and list if present
- seed deterministic nested resources on site create

Exit criteria:
- VMware nested coverage is successful without `NotImplemented`

### Stage 3: Hyper-V Site Foundation

Goal:
- mirror the proven site patterns for the second provider tree

Work:
- implement Hyper-V site:
  - `PUT`
  - `GET`
  - `HEAD`
  - `PATCH`
  - `DELETE`
  - `summary`
  - `refresh`
  - health summary if present
- reuse shared site helpers while preserving Hyper-V-specific type names and summary fields

Exit criteria:
- Hyper-V site lifecycle and action routes are green

### Stage 4: Hyper-V Nested Inventory

Goal:
- complete the main nested resources under `HyperVSites`

Work:
- implement Hyper-V:
  - `machines` list/get
  - machine `start` / `stop` if documented for this tree
  - `runAsAccounts` list/get
  - `hosts` list/get
  - `clusters` list/get
- derive host and cluster fixtures from each site

Exit criteria:
- both provider trees have deterministic nested inventory coverage

### Stage 5: Example and Coverage Hardening

Goal:
- make the new emulator easy to validate and consume

Work:
- add `examples/azure/migrate-discovery-2020-01-01`
- use the official Azure Go SDK package:
  - `github.com/Azure/azure-sdk-for-go/services/migrate/mgmt/2020-01-01/migrate`
- demonstrate:
  - operations list
  - VMware site create/get
  - VMware machine list or get
  - Hyper-V site create/get
- finalize coverage aliases and doc mappings
- run targeted service coverage plus repo-level validation expected for new Azure services

Exit criteria:
- example builds cleanly
- all service endpoints implemented for the chosen slice pass the contract, IO, doc, and endpoint coverage scripts

## Implementation Notes

Suggested in-memory layout:

- top-level maps keyed by ARM ID for:
  - VMware sites
  - Hyper-V sites
- child maps keyed by parent site ARM ID plus child name for:
  - VMware machines
  - VMware run-as accounts
  - VMware vCenters
  - Hyper-V machines
  - Hyper-V run-as accounts
  - Hyper-V hosts
  - Hyper-V clusters

Suggested helper structure:

- shared ARM helpers:
  - validate `api-version`
  - parse subscription / resource group / site names
  - emit ARM errors
- shared site helpers:
  - canonical ARM ID and type construction
  - patch merging
  - summary generation
  - fixture seeding
- async helpers:
  - accepted response builder for `refresh`, `start`, and `stop`

Response-shape strategy:

- prioritize contract-stable required fields over exhaustive schema completion
- use deterministic timestamps, IDs, and summary counters
- keep large nested arrays compact but non-empty where the model implies collections

## Testing Strategy

Add focused tests in `internal/server/provider_azure_migrate_discovery_test.go` for:

- operations list
- VMware site create/get/head/patch/delete
- VMware site summary and refresh
- VMware machines list/get/start/stop
- VMware run-as accounts list/get
- Hyper-V site create/get/head/patch/delete
- Hyper-V site summary and refresh
- Hyper-V machines list/get
- Hyper-V run-as accounts list/get
- Hyper-V hosts get
- Hyper-V clusters get
- invalid API version handling
- missing site and missing nested resource handling

Coverage validation target after implementation:

- `python3 scripts/azure-contract-coverage.py --service migrate_discovery --require-service migrate_discovery --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service migrate_discovery --require-service migrate_discovery --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service migrate_discovery --require-service migrate_discovery --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service migrate_discovery --require-service migrate_discovery --fail-on any`

## Risks and Gaps

- Several Microsoft Learn pages for this older API family are intermittently gated or only partially indexable in search results, so some tail operation groups may need confirmation during implementation.
- The older official Go SDK package is track-1 and deprecated, which is acceptable for the example but may require more manual request setup than newer ARM packages.
- The full Discovery surface is broader than the representative method pages reviewed here; implementation should begin with the listed core slices and extend only as coverage demonstrates more linked routes are required.

## Definition of Done

The staged implementation should be considered complete when:

- `migrate_discovery` is recognized by the Azure coverage scripts
- provider routes under `Microsoft.OffAzure` return modeled responses instead of `NotImplemented`
- VMware and Hyper-V site lifecycles are deterministic and test-covered
- the main nested inventory routes are implemented for both trees
- the Azure Go SDK example builds and exercises the local emulator
- the targeted Azure coverage scripts for `migrate_discovery` pass cleanly
