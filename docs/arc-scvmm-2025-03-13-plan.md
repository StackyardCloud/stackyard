# Azure Arc-enabled SCVMM (`arc-scvmm-2025-03-13`) Staged Plan

## Objective

Emulate Azure Arc-enabled SCVMM for API version `2025-03-13` with deterministic ARM-style behavior under `Microsoft.ScVmm`, including:

- provider `operations`
- onboarding and inventory resources such as `vmmServers`, `clouds`, `availabilitySets`, `virtualNetworks`, and `virtualMachineTemplates`
- nested `inventoryItems` under `vmmServers`
- `virtualMachineInstances` and their action routes
- VM guest-agent and hybrid-identity metadata subresources
- a typed Azure Go SDK example using the Azure Arc-enabled SCVMM SDK
- Azure coverage-script wiring for the new service key

The surface is broad enough that it should be implemented in staged slices around shared route patterns and resource shapes rather than as one large provider drop.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/operation-groups?view=rest-azure-arc-scvmm-2025-03-13`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/operations?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/vmm-servers?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/clouds?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/availability-sets?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-networks?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-templates?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances?view=rest-azure-arc-scvmm-2025-03-13`

Linked method pages reviewed across those groups and the nested SCVMM subresource families:
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/operations/list?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/vmm-servers/create-or-update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/vmm-servers/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/vmm-servers/update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/clouds/create-or-update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/clouds/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/availability-sets/create-or-update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/availability-sets/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-networks/create-or-update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-networks/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-templates/create-or-update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-templates/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/inventory-items/create?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/inventory-items/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances/create-or-update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances/list?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances/update?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances/start?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/virtual-machine-instances/stop?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/guest-agents/create?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/guest-agents/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/guest-agents/delete?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/vm-instance-hybrid-identity-metadatas/get?view=rest-azure-arc-scvmm-2025-03-13`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-scvmm/vm-instance-hybrid-identity-metadatas/list-by-virtual-machine-instance?view=rest-azure-arc-scvmm-2025-03-13`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/scvmm/armscvmm`

## Service Naming

Canonical Stackyard naming:
- service key: `arc_scvmm`
- display name: `Azure Arc-enabled SCVMM`
- provider namespace: `Microsoft.ScVmm`
- API version: `2025-03-13`

Recommended repo paths:
- implementation: `internal/server/provider_azure_arc_scvmm.go`
- tests: `internal/server/provider_azure_arc_scvmm_test.go`
- example: `examples/azure/arc-scvmm-2025-03-13`
- plan doc: `docs/arc-scvmm-2025-03-13-plan.md`

## Surface Summary

From the reviewed pages, the SCVMM ARM surface clusters into these major areas:

1. Provider discovery
2. Fabric onboarding resources
3. Fabric inventory resources
4. Virtual machine instance lifecycle
5. Guest-agent and hybrid-identity subresources

High-value operation groups and families for implementation:
- `Operations`
- `Vmm Servers`
- `Clouds`
- `Availability Sets`
- `Virtual Networks`
- `Virtual Machine Templates`
- `Inventory Items`
- `Virtual Machine Instances`
- `Guest Agents`
- `Vm Instance Hybrid Identity Metadatas`

Important shape observations from the reviewed pages:
- Most top-level resources use standard ARM subscription/resource-group paths under `Microsoft.ScVmm`.
- Top-level onboarding resources consistently use standard ARM CRUD semantics with `location`, `extendedLocation`, tags, and provider-specific `properties`.
- `inventoryItems` are nested under `vmmServers` and have discriminated property shapes for cloud, VM, VM template, and virtual network inventory.
- `virtualMachineInstances`, `guestAgents`, and hybrid-identity metadata use `{resourceUri}`-based routes rather than standard resource-group collection paths.
- VM instance actions such as `start` and `stop` use operation-style `POST` routes and return accepted/operation-style responses rather than regular resource bodies.

## Route Envelope

Primary route families under `/azure`:

- `/azure/providers/Microsoft.ScVmm/operations`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/vmmServers`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/vmmServers/{vmmServerName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/vmmServers/{vmmServerName}/inventoryItems`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/vmmServers/{vmmServerName}/inventoryItems/{inventoryItemResourceName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/clouds/{cloudResourceName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/availabilitySets/{availabilitySetResourceName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/virtualNetworks/{virtualNetworkName}`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ScVmm/virtualMachineTemplates/{virtualMachineTemplateName}`

`resourceUri`-based route families:

- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/start`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/stop`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/restart`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/createCheckpoint`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/deleteCheckpoint`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/restoreCheckpoint`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/guestAgents/default`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/hybridIdentityMetadata`
- `/azure/{resourceUri}/providers/Microsoft.ScVmm/virtualMachineInstances/default/hybridIdentityMetadata/default`

Target query:
- `api-version=2025-03-13`

Important routing implications:
- top-level resources are standard ARM collections keyed by subscription, resource group, and resource name
- `virtualMachineInstances` and their children hang under an existing `{resourceUri}` and should be routed as a separate branch
- `inventoryItems` use UUID-like resource names and should be modeled as typed child resources under a parent `vmmServer`

## Contract Notes From Reviewed Pages

### Operations

`Operations.List`
- `GET /providers/Microsoft.ScVmm/operations`
- returns the provider operation catalog

Modeling implication:
- implement first as a deterministic `value` list covering the resource families the emulator exposes

### Vmm Servers

Reviewed methods show standard ARM CRUD/list:
- create or update
- delete
- get
- list by resource group
- list by subscription
- patch update

The reviewed create page shows key request fields:
- `location`
- `extendedLocation`
- `properties.fqdn`
- optional `properties.credentials`
- optional `properties.port`
- `tags`

Modeling implication:
- `vmmServers` should be a canonical parent store because many other resources reference `properties.vmmServerId`
- first implementation should validate `location`, `extendedLocation`, and `properties.fqdn`

### Clouds, Availability Sets, Virtual Networks, Virtual Machine Templates

These top-level resources all follow the same ARM pattern:
- create or update
- delete
- get
- list by resource group
- list by subscription
- patch update

The reviewed create/get pages show consistent core fields:
- `location`
- `extendedLocation`
- `tags`
- resource-specific names and IDs inside `properties`
- `properties.vmmServerId` on resources that belong to a VMM server
- provider-generated `provisioningState`

Modeling implication:
- these resources can share a generic ARM collection helper with small per-type validation rules
- deterministic fixtures should preserve each resource’s identifying SCVMM properties so the Azure SDK example has stable response fields to inspect

### Inventory Items

Reviewed methods show:
- create
- get
- list under a `vmmServer`
- delete semantics are implied by the resource family and should be checked during implementation if coverage requires them

Important modeled fields from the reviewed pages:
- `kind`
- `properties.inventoryType`
- `properties.managedResourceId`
- `properties.uuid`
- `properties.inventoryItemName`
- discriminated inventory shapes for:
  - cloud inventory
  - virtual machine inventory
  - virtual machine template inventory
  - virtual network inventory

Modeling implication:
- `inventoryItems` should be staged after `vmmServers`
- implement a shallow discriminator based on `properties.inventoryType` rather than fully reproducing every inventory shape up front

### Virtual Machine Instances

Reviewed methods include:
- `Create Or Update`
- `Delete`
- `Get`
- `List`
- `Update`
- `Start`
- `Stop`
- `Restart`
- `Create Checkpoint`
- `Delete Checkpoint`
- `Restore Checkpoint`

The reviewed create/update pages show core request fields such as:
- `extendedLocation`
- `properties.hardwareProfile`
- `properties.infrastructureProfile`
- `properties.networkProfile`
- `properties.storageProfile`
- `properties.availabilitySets`

The reviewed action pages show:
- `start`, `stop`, and similar action routes use `POST`
- action responses can be `202 Accepted`
- `stop` accepts `skipShutdown`

Modeling implication:
- treat VM instances as a separate `{resourceUri}`-keyed store from top-level ARM collections
- model power state and checkpoint state explicitly
- action routes can still complete deterministically in-process while returning accepted-style status and headers if needed by examples/coverage

### Guest Agents

Reviewed methods include:
- `Create`
- `Get`
- `Delete`

The reviewed create page shows request properties:
- `properties.credentials`
- `properties.httpProxyConfig`
- `properties.privateLinkScopeResourceId`
- `properties.provisioningAction`

Modeling implication:
- this is a compact singleton-style subresource at `/guestAgents/default`
- it should store a small status-bearing object tied to a VM instance resource URI

### Vm Instance Hybrid Identity Metadatas

Reviewed methods include:
- `Get`
- `List By Virtual Machine Instance`

Important modeled fields from the reviewed pages:
- `properties.resourceUid`
- `properties.publicKey`
- `properties.provisioningState`
- standard `value` list envelope for the list route

Modeling implication:
- this is a read-heavy singleton/listable child family and should be implemented after VM instance creation exists

## Staged Implementation

### Stage 0: Planning And Coverage Wiring

Deliverables:
- add `docs/arc-scvmm-2025-03-13-plan.md`
- add `arc_scvmm` aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map this plan doc in `azure-doc-contract-coverage.py`

Success criteria:
- the service key `arc_scvmm` is discoverable by all Azure coverage scripts
- doc coverage can tie the new service to official Learn references

### Stage 1: Provider Skeleton And Operations

Implement first:
- `handleAzureArcSCVMMRouter`
- path detection for `Microsoft.ScVmm`
- `Operations.List`
- shared request-body parsing helpers
- standard Azure invalid `api-version` handling

Success criteria:
- provider operations route is deterministic
- unknown SCVMM routes fail with the repo’s standard Azure not-implemented behavior

### Stage 2: Core ARM Onboarding Resources

Implement CRUD/list/patch for:
- `vmmServers`
- `clouds`
- `availabilitySets`
- `virtualNetworks`
- `virtualMachineTemplates`

State model:
- separate in-memory maps per resource family keyed by full ARM ID
- shared fixture helpers for `id`, `name`, `type`, `location`, `extendedLocation`, `tags`, `systemData`, and `etag`

Validation baseline:
- `location` required on `PUT`
- `extendedLocation` required where documented on create
- `properties.fqdn` required for `vmmServers`

Success criteria:
- all top-level onboarding resources return non-`NotImplemented`
- list-by-resource-group and list-by-subscription behave consistently

### Stage 3: Inventory Items Under VMM Servers

Implement:
- list inventory items under a `vmmServer`
- get inventory item
- create or update inventory item
- delete inventory item if coverage/examples require it

State model:
- nested store keyed by full inventory item ARM ID
- shallow discriminator on `properties.inventoryType`

Success criteria:
- inventory routes reflect the parent `vmmServer`
- at least one typed inventory shape exists for each high-value family:
  - cloud
  - virtual machine
  - virtual machine template
  - virtual network

### Stage 4: Virtual Machine Instances And Actions

Implement:
- VM instance create/get/list/update/delete
- `start`
- `stop`
- `restart`
- checkpoint action routes

State model:
- `{resourceUri}`-keyed VM instance store with:
  - provisioning state
  - power state
  - checkpoint metadata

Success criteria:
- all VM instance methods return non-`NotImplemented`
- action routes mutate deterministic state and expose stable action results

### Stage 5: Guest Agents And Hybrid Identity Metadata

Implement:
- guest agent create/get/delete
- hybrid identity metadata get/list

Success criteria:
- subresource routes under `/guestAgents/default` and `/hybridIdentityMetadata` resolve against an existing VM instance
- coverage sees typed success fixtures and negative tests for these nested routes

### Stage 6: Tests

Add focused provider tests covering:
- operations listing
- one happy-path CRUD/list flow for each top-level resource family
- inventory-item nesting under `vmmServers`
- VM instance lifecycle and power actions
- guest-agent and hybrid-identity subresources
- invalid JSON, missing required fields, not-found routes, and invalid `api-version`

Success criteria:
- targeted provider tests pass with a command like:
  - `go test ./internal/server -run 'TestAzureArcSCVMM' -count=1`

### Stage 7: Example

Add `examples/azure/arc-scvmm-2025-03-13`:
- `main.go`
- `Dockerfile`
- `docker-compose.yml`

Example flow should exercise:
- provider operations
- create a `vmmServer`
- create one or more dependent onboarding resources such as a `cloud` or `virtualNetwork`
- create a VM instance through the `{resourceUri}` route family
- run at least one VM action such as `start` or `stop`
- get guest-agent or hybrid-identity metadata

SDK approach:
- use `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/scvmm/armscvmm`

Success criteria:
- `go build ./examples/azure/arc-scvmm-2025-03-13` passes

### Stage 8: Validation

Run:
- `go test ./internal/server -run 'TestAzureArcSCVMM' -count=1`
- `go build ./examples/azure/arc-scvmm-2025-03-13`
- `python3 scripts/azure-contract-coverage.py --service arc_scvmm --require-service arc_scvmm --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service arc_scvmm --require-service arc_scvmm --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service arc_scvmm --require-service arc_scvmm --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service arc_scvmm --require-service arc_scvmm --fail-on any`

Success target:
- all `arc_scvmm` endpoints return successful or contract-valid modeled responses
- no `arc_scvmm` endpoint returns `NotImplemented`
- if full repo CI still fails, the remaining failures should be unrelated existing repo issues
