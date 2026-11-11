# Azure Arc-enabled VMware (`arc-vmware-2023-12-01`) Staged Plan

## Objective

Emulate Azure Arc-enabled VMware for API version `2023-12-01` with deterministic ARM-style behavior under `Microsoft.ConnectedVMwarevSphere`, including:

- provider `operations`
- top-level fabric resources such as `vcenters`, `clusters`, `hosts`, `resourcePools`, `datastores`, `virtualNetworks`, and `virtualMachineTemplates`
- nested `inventoryItems` under `vcenters`
- `{resourceUri}`-based `virtualMachineInstances`
- `guestAgents` and `hybridIdentityMetadata` under virtual machine instances
- an Azure Go SDK example under `examples/azure/arc-vmware-2023-12-01`
- Azure coverage-script wiring for the new service key

The surface is broad enough that implementation should be staged around shared route families and reusable resource fixtures instead of a single large drop.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/operation-groups?view=rest-azure-arc-vmware-2023-12-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/operations?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/v-centers?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/clusters?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/hosts?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/resource-pools?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/datastores?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-networks?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-templates?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/inventory-items?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances?view=rest-azure-arc-vmware-2023-12-01`

Linked method pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/operations/list?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/v-centers/get?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/clusters/update?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/hosts/get?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/resource-pools/create?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/resource-pools/get?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/datastores/get?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-networks/create?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/inventory-items/create?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances/create-or-update?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances/get?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances/list?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances/start?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances/stop?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/virtual-machine-instances/restart?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/vm-instance-guest-agents/create?view=rest-azure-arc-vmware-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azure-arc-vmware/vm-instance-hybrid-identity-metadata/get?view=rest-azure-arc-vmware-2023-12-01`

Azure Go SDK package selected for implementation:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/connectedvmware/armconnectedvmware`

## Service Naming

Canonical Stackyard naming:
- service key: `arc_vmware`
- display name: `Azure Arc-enabled VMware`
- provider namespace: `Microsoft.ConnectedVMwarevSphere`
- API version: `2023-12-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_arc_vmware.go`
- tests: `internal/server/provider_azure_arc_vmware_test.go`
- example: `examples/azure/arc-vmware-2023-12-01`
- plan doc: `docs/arc-vmware-2023-12-01-plan.md`

## Surface Summary

The reviewed operation-groups page exposes these families:

1. `Operations`
2. `V Centers`
3. `Clusters`
4. `Hosts`
5. `Resource Pools`
6. `Datastores`
7. `Virtual Networks`
8. `Virtual Machine Templates`
9. `Inventory Items`
10. `Virtual Machine Instances`
11. `VM Instance Guest Agents`
12. `Vm Instance Hybrid Identity Metadata`

Important route-shape observations from the reviewed pages:

- Top-level resources use standard ARM subscription/resource-group paths under `Microsoft.ConnectedVMwarevSphere`.
- Most top-level resources follow standard create/get/list/delete/patch semantics with `location`, optional `extendedLocation`, `tags`, and resource-specific `properties`.
- `inventoryItems` are nested under `vcenters/{vcenterName}` and use a typed `properties.inventoryType`.
- `virtualMachineInstances`, `guestAgents`, and `hybridIdentityMetadata` are anchored under `{resourceUri}` for a Hybrid Compute machine resource, not under a resource-group collection.
- VM lifecycle operations such as `start`, `stop`, and `restart` are `POST` action routes under `virtualMachineInstances/default`.

## Route Envelope

Primary top-level ARM routes:

- `/azure/providers/Microsoft.ConnectedVMwarevSphere/operations`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/vcenters`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/clusters`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/hosts`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/resourcePools`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/datastores`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/virtualNetworks`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineTemplates`
- `/azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ConnectedVMwarevSphere/vcenters/{vcenterName}/inventoryItems`

`resourceUri`-based VM routes:

- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/start`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/stop`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/restart`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/guestAgents/default`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/hybridIdentityMetadata`
- `/azure/{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/hybridIdentityMetadata/default`

Target query:
- `api-version=2023-12-01`

## Contract Notes From Reviewed Pages

### Operations

`Operations.List`
- `GET /providers/Microsoft.ConnectedVMwarevSphere/operations`
- returns a provider operation catalog

Modeling implication:
- implement first as a deterministic `value` list covering every resource family the emulator exposes

### V Centers

Reviewed pages show standard ARM CRUD/list/update semantics.

Important fields visible from the reviewed `get` and SDK pages:
- `location`
- `extendedLocation`
- `tags`
- `properties.credentials.username`
- `properties.fqdn`
- `properties.port`
- `properties.version`
- `properties.instanceUuid`
- `properties.provisioningState`

Modeling implication:
- `vcenters` should be the anchor resource for `inventoryItems`
- initial validation should require `location` plus the key connection properties such as `fqdn`

### Clusters, Hosts, Resource Pools, Datastores, Virtual Networks, Virtual Machine Templates

These families follow the same ARM pattern:
- create or update
- get
- list by resource group
- list by subscription
- patch update
- delete for mutable collections

Important modeled properties seen across the reviewed pages:
- `properties.inventoryItemId`
- `properties.vCenterId`
- `properties.moRefId`
- `properties.moName`
- `properties.uuid`
- `properties.provisioningState`

Modeling implication:
- these families can share one generic ARM resource store with small per-collection validation rules
- each fixture should keep the VMware-specific identifying properties stable so the example can inspect them

### Inventory Items

Reviewed pages show:
- create
- get
- list by vCenter
- delete

Important fields:
- `kind`
- `properties.inventoryType`
- `properties.inventoryItemName`
- `properties.managedResourceId`
- `properties.uuid`

Modeling implication:
- `inventoryItems` should be staged after `vcenters`
- implement a shallow discriminator on `properties.inventoryType` instead of full per-type payload fidelity in the first pass

### Virtual Machine Instances

Reviewed pages show:
- create or update
- get
- list
- delete
- `start`
- `stop`
- `restart`

Route pattern:
- `{resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default`

Important fields:
- `extendedLocation`
- `properties.infrastructureProfile`
- `properties.osProfile`
- `properties.provisioningState`
- `properties.powerState`

Modeling implication:
- use a separate store keyed by normalized `resourceUri`
- the emulator can expose one canonical VM instance named `default` per `{resourceUri}`
- lifecycle actions should mutate a stable `powerState` field and return deterministic operation results

### VM Instance Guest Agents

Reviewed create page shows:
- `PUT {resourceUri}/providers/Microsoft.ConnectedVMwarevSphere/virtualMachineInstances/default/guestAgents/default`
- request uses `properties.credentials`
- response is a `GuestAgent` resource

Modeling implication:
- require the parent VM instance to exist first
- validate at least `properties.credentials.username`
- keep the guest agent as a simple singleton child resource named `default`

### Vm Instance Hybrid Identity Metadata

Reviewed get page shows:
- list/get under `hybridIdentityMetadata`
- payload includes `properties.resourceUid` and `properties.publicKey`

Modeling implication:
- create deterministic singleton hybrid-identity metadata automatically when the VM instance is created
- surface both list and get routes with the same backing record

## Staged Implementation Plan

### Stage 0: Provider Wiring and Canonical Service Mapping

Files:
- `internal/server/http.go`
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Tasks:
- add the canonical Azure service alias block for `arc_vmware`
- wire a new Azure provider router entry
- reserve server-side state maps for Arc VMware resources
- map the plan doc in the doc coverage script

Exit criteria:
- service aliases resolve `arc_vmware`, `arc-vmware`, `vmware`, `azure_arc_vmware`, `rest_azure_arc_vmware_2023_12_01`, and `microsoft_connectedvmwarevsphere`

### Stage 1: Operations and Shared ARM Resource Helpers

Files:
- `internal/server/provider_azure_arc_vmware.go`

Tasks:
- implement provider path recognition for `Microsoft.ConnectedVMwarevSphere`
- add `Operations.List`
- add shared helpers for:
  - top-level ARM CRUD/list routing
  - stable ARM IDs
  - storage-key normalization
  - system data / etag generation
  - request-body parsing and validation errors

Exit criteria:
- provider operations succeed
- unsupported Arc VMware routes fall through cleanly to `respondAzureImplemented`

### Stage 2: Top-level Fabric Resources

Initial families:
- `vcenters`
- `clusters`
- `hosts`
- `resourcePools`
- `datastores`
- `virtualNetworks`
- `virtualMachineTemplates`

Tasks:
- implement create/get/head/list-by-resource-group/list-by-subscription/patch/delete
- validate minimal required inputs by family
- keep deterministic VMware-specific properties in fixtures

Recommended minimum validations:
- `vcenters`: require `location`, `properties.fqdn`
- other top-level resources: require `location`; for child-of-vCenter style resources require `properties.vCenterId` or `properties.inventoryItemId` when appropriate

Exit criteria:
- coverage can exercise all top-level ARM routes without `NotImplemented`

### Stage 3: VCenter Inventory Items

Tasks:
- implement `inventoryItems` under `vcenters/{vcenterName}`
- support create/get/list/delete
- validate `properties.inventoryType`
- generate deterministic `managedResourceId`, `uuid`, and provisioning state

Exit criteria:
- a created vCenter can host one or more inventory items
- inventory item list and get return stable typed fixtures

### Stage 4: Virtual Machine Instances and Actions

Tasks:
- implement `{resourceUri}` route parsing
- implement `virtualMachineInstances/default` create/get/list/patch/delete
- implement `start`, `stop`, and `restart`
- track power state transitions

Exit criteria:
- lifecycle action routes mutate VM state predictably
- repeated get/list after actions shows the updated `powerState`

### Stage 5: Guest Agents and Hybrid Identity Metadata

Tasks:
- implement `guestAgents/default` create/get/delete
- implement `hybridIdentityMetadata` list/get
- auto-seed default hybrid metadata when a VM instance is created

Exit criteria:
- VM instance coverage can walk through guest-agent and hybrid-identity routes end to end

### Stage 6: Azure Go SDK Example

Example path:
- `examples/azure/arc-vmware-2023-12-01`

SDK:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/connectedvmware/armconnectedvmware`

Recommended example scope:
- `OperationsClient` or equivalent provider list call
- create/get/list a `vcenter`
- create/get a `virtualMachineInstance`
- invoke one VM action such as `start`

Implementation guidance:
- follow the existing Azure examples that use the shared Stackyard transport shim when a typed client path is straightforward
- if one or two route families are awkward in the typed client because of `{resourceUri}` signatures, it is acceptable to combine typed ARM clients for top-level resources with `azcore/runtime` requests for the VM-instance branch, but keep the example Azure-SDK-based

Exit criteria:
- `go build ./examples/azure/arc-vmware-2023-12-01` succeeds
- example exercises the new provider with stable output

### Stage 7: Coverage and Validation

Required checks:
- `go test ./internal/server -run 'TestAzureArcVMware' -count=1`
- `go build ./examples/azure/arc-vmware-2023-12-01`
- `python3 scripts/azure-contract-coverage.py --service arc_vmware --require-service arc_vmware --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service arc_vmware --require-service arc_vmware --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service arc_vmware --require-service arc_vmware --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service arc_vmware --require-service arc_vmware --fail-on any`

Goal:
- all Arc VMware endpoints exercised by the repo coverage scripts succeed
- no Arc VMware endpoint returns `NotImplemented`

## Testing Strategy

Add focused tests in:
- `internal/server/provider_azure_arc_vmware_test.go`

Recommended test slices:
- operations list
- vCenter lifecycle
- one representative lifecycle each for `clusters`, `hosts`, `resourcePools`, `datastores`, `virtualNetworks`, and `virtualMachineTemplates`
- inventory item create/get/list/delete under a vCenter
- VM instance create/get/list/patch/delete and `start`/`stop`/`restart`
- guest-agent create/get/delete
- hybrid-identity metadata list/get
- invalid JSON
- invalid API version
- not-found responses for missing resources

Coverage-specific expectations:
- tests should assert response JSON shapes, not just status codes
- include explicit negative assertions for bad requests and missing resources so contract/doc coverage marks the service as fully covered

## Risk Notes

- The Microsoft Learn operation-groups page is partially gated, but the linked method pages and SDK package clearly establish the route families and signatures.
- The `{resourceUri}` branch is the most specialized part of the service and should be isolated early to avoid mixing it into the standard ARM resource-group router.
- The selected Go SDK package currently documents generated clients for `2023-10-01`; that is still the closest official Azure Go SDK package for this service family and should be acceptable for Stackyard examples if the route signatures align closely with `2023-12-01`.

## Recommended Implementation Order

1. Stage 0 and Stage 1
2. Stage 2 for `vcenters`, then the remaining top-level collections
3. Stage 3 `inventoryItems`
4. Stage 4 VM instances and actions
5. Stage 5 guest agents and hybrid metadata
6. Stage 6 example
7. Stage 7 coverage and CI validation
