# Azure Network Cloud workloads (`networkcloud-2026-07-01`) Staged Plan

## Objective

Emulate the tenant workload resources of the Azure Operator Nexus Network Cloud resource provider (`Microsoft.NetworkCloud`) on the shared ARM handler (`internal/server/azure_resource_manager.go`):

- **Networks:** cloud services networks and layer 2, layer 3 and trunked networks.
- **Storage:** volumes.
- **Virtual machines:** with their consoles and power actions (start, power off, restart, reimage, assign relay).
- **Kubernetes clusters:** with their agent pools, features and node restart.

`networkcloud-2026-07-01-plan.md` describes the infrastructure half: cluster managers, clusters, racks, bare metal machines, storage appliances and the rest. Nothing is scheduled on hardware. Every write completes at once, and resources report the detailed status of a healthy instance.

## References

Microsoft Learn pages for the `2026-07-01` API version (append `?view=rest-networkcloud-2026-07-01`):

- `https://learn.microsoft.com/en-us/rest/api/networkcloud/` (API overview)
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cloud-services-networks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cloud-services-networks/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cloud-services-networks/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cloud-services-networks/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cloud-services-networks/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cloud-services-networks/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l-2-networks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l-2-networks/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l-2-networks/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l-2-networks/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l-2-networks/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l-2-networks/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l3-networks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l3-networks/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l3-networks/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l3-networks/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l3-networks/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/l3-networks/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/trunked-networks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/trunked-networks/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/trunked-networks/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/trunked-networks/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/trunked-networks/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/trunked-networks/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/volumes/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/volumes/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/volumes/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/volumes/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/volumes/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/volumes/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/start`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/power-off`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/restart`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/reimage`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/virtual-machines/assign-relay`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/consoles/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/consoles/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/consoles/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/consoles/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/consoles/list-by-virtual-machine`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-clusters/restart-node`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/agent-pools/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/agent-pools/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/agent-pools/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/agent-pools/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/agent-pools/list-by-kubernetes-cluster`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-cluster-features/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-cluster-features/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-cluster-features/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-cluster-features/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-cluster-features/list-by-kubernetes-cluster`

## Service Naming

- service key: `networkcloud`
- display name: `Azure Operator Nexus - Network Cloud`
- implementation:
  - `internal/server/provider_azure_networkcloud_workloads.go`: networks, volumes, the shared checks and network use tracking
  - `internal/server/provider_azure_networkcloud_workloads_vm.go`: virtual machines and consoles
  - `internal/server/provider_azure_networkcloud_workloads_k8s.go`: Kubernetes clusters, agent pools and features
- shared handler: `azureARMResource` gains a top-level `extendedLocation`. PUT stores it, PATCH keeps it, and responses return it when it's set.
- tests: `internal/server/provider_azure_networkcloud_workloads_test.go`, `internal/server/provider_azure_networkcloud_workloads_k8s_test.go`
- example: `examples/azure/networkcloud-2026-07-01/workloads.go`

## Operations In Scope

| Resource | Operations |
|---|---|
| Cloud services networks | create or update, get, update, delete, list by resource group and subscription |
| L2, L3 and trunked networks | create or update, get, update (tags), delete, list by resource group and subscription |
| Volumes | create or update, get, update (tags), delete, list by resource group and subscription |
| Virtual machines | create or update, get, update, delete, list by resource group and subscription, start, power off, restart, reimage, assign relay |
| Consoles (`default`) | create or update, get, update, delete, list by virtual machine |
| Kubernetes clusters | create or update, get, update, delete, list by resource group and subscription, restart node |
| Agent pools | create or update, get, update, delete, list by Kubernetes cluster |
| Kubernetes cluster features | create or update, get, update, delete, list by Kubernetes cluster |

## Behavior

### Common

- Every resource returns an `etag` that changes with each write.
- **`extendedLocation`:**
  - Required on every type except agent pools, and fixed after creation.
  - `type` is `EdgeZone` or `CustomLocation`.
  - A custom location's `name` must be a `Microsoft.ExtendedLocation/customLocations` ID. The custom location itself isn't looked up.
- **Names** follow the path patterns Learn lists:
  - networks, Kubernetes clusters and agent pools: 2–30 characters
  - virtual machines: 2–64 letters and digits
  - volumes: 2–64 characters
  - consoles: always `default`
  - features: 2–63 letters, digits and hyphens
- **Create-only properties:** those Learn marks create-only can't change on a later PUT (400). A later PUT that omits one keeps the stored value.
- **PATCH** accepts only each type's update model. Anything else gets 400.

### Networks and volumes

- **L2:**
  - `l2IsolationDomainId` is required.
  - `hybridAksPluginType` (`DPDK`, `SRIOV`, `OSDevice`) defaults to `SRIOV`.
  - `interfaceName` is up to 12 letters, digits and `@._-`.
- **L3:**
  - `l3IsolationDomainId` and `vlan` are required.
  - `ipAllocationType` (`IPV4`, `IPV6`, `DualStack`) defaults to `DualStack`, and `hybridAksIpamEnabled` defaults to `True`.
  - `ipv4ConnectedPrefix` is required for `IPV4` and `DualStack`, and `ipv6ConnectedPrefix` for `IPV6` and `DualStack`. Each must be a CIDR of its family.
- **Trunked:** at least one isolation domain ID and one VLAN.
- **Cloud services networks:**
  - Each additional egress endpoint needs a `category` and at least one endpoint with a `domainName` and a port from 1 to 65535.
  - `storageOptions` takes `mode` (`None` or `Standard`), `sizeMiB` (at least 1) and a storage appliance ID. They are reported back in `storageStatus`.
  - PATCH may change the egress endpoints and the storage options.
- **Volumes:** `sizeMiB` (at least 1) is required. `storageApplianceId` is optional and is reported as `assignedStorageApplianceId`.
- **Usage tracking:**
  - Networks list the virtual machines and Kubernetes clusters that attach them in `associatedResourceIds`, `virtualMachinesAssociatedIds` and `hybridAksClustersAssociatedIds`.
  - Volumes list the virtual machines they're attached to in `attachedTo`.
  - A network or volume that is in use can't be deleted (409).

### Virtual machines and consoles

- **Required properties:**
  - `adminUsername` (`^[a-z_][a-z0-9_]{0,31}$`)
  - `cloudServicesNetworkAttachment`
  - `cpuCores` (at least 2) and `memorySizeGB` (at least 1)
  - `storageProfile.osDisk.diskSizeGB`
  - `vmImage`
- **Defaults:** `bootMethod` `UEFI`, `isolateEmulatorThread` `True`, `virtioInterface` `Modern` and `vmDeviceModel` `T2`.
- **Network attachments:**
  - The cloud services network attachment must name an existing cloud services network.
  - Other attachments must name existing L2, L3 or trunked networks.
  - `Dynamic` and `Static` allocation are valid only for an L3 network; other network types need `Disabled`.
  - A `Static` address must be within the L3 network's prefix.
  - Only one attachment may be the default gateway.
  - Attachment names are up to 15 characters and must be unique.
- **Volumes:** volume attachments must name existing volumes and are reported in `volumes`. Each network attachment gets a stable `macAddress`.
- **Write-only:** `vmImageRepositoryCredentials.password`, `userDataContent` and `networkDataContent`.
- **Updates:** everything except the image credentials is fixed at creation. PATCH changes only those credentials, `tags` and `identity`.
- **Power actions:**
  - A new machine is `On` and `Running`.
  - Power off (with `skipShutdown` `True` or `False`) leaves it `Off` and `Stopped`. Start and reimage leave it `On`.
  - Restart needs a running machine (409 otherwise).
  - All of them answer 202.
- **Assign relay:** `machineId` must be a `Microsoft.HybridCompute/machines` ID, and `relayType` is `Platform` or `Public`.
- **Consoles:**
  - `enabled` (`True` or `False`) and `sshPublicKey.keyData` are required, and `expiration` is an RFC 3339 date-time.
  - A console reports `Ready` with a stable `virtualMachineAccessId`.
  - A console is deleted with its virtual machine.

### Kubernetes clusters

- **Required properties:**
  - `kubernetesVersion`
  - `controlPlaneNodeConfiguration`: `count` (at least 1) and `vmSkuName`
  - `initialAgentPoolConfigurations`: at least one pool
  - `networkConfiguration`: `cloudServicesNetworkId` and `cniNetworkId`
- **Referenced networks:** the cloud services network and the CNI L3 network must exist. So must every attached network, and each must be of the type its list names.
- **System pool rule:** every cluster must contain at least one `System` agent pool with at least one node. Creating a cluster without one gets 400. Removing the last one's nodes, or the pool itself, gets 409.
- **Network configuration:**
  - `dnsServiceIp` must be within the service CIDRs.
  - **IP address pools** need a name and addresses. Addresses are CIDRs or start-end ranges; BGP pools take CIDRs only, and not `/32` or `/128` prefixes.
  - **BGP peers** need a name, `peerAddress` and `peerAsn` (0–4294967295). Their `password` (up to 80 letters and digits) is write-only.
  - **Advertisements** must name pools and peers that are defined.
- **Agent pool settings:**
  - `count`, `mode` (`System`, `User`, `NotApplicable`) and `vmSkuName` are required.
  - Upgrade settings take a drain timeout of 60–86400 seconds, plus `maxSurge` and `maxUnavailable` as counts or percentages. At least one of those two must be above 0.
- **After creation:**
  - The initial agent pools become `agentPools` resources, and `initialAgentPoolConfigurations` is empty from then on.
  - The cluster lists its `nodes` (the control plane's and each pool's), `attachedNetworkIds` and `featureStatuses`, and reports `controlPlaneKubernetesVersion`.
- **PATCH:**
  - Accepts `kubernetesVersion` (the agent pools follow it), the control plane `count`, and SSH keys.
  - Fixed after creation: the network, AAD and managed resource group configurations, the control plane SKU and zones, and the admin usernames.
- **Restart node** needs the name of one of the cluster's nodes (400 otherwise) and answers 202.
- **Agent pools:** everything except `count`, `upgradeSettings` and SSH keys is fixed at creation.
- **Features** take `options` key-value pairs and report `Installed`, `GenerallyAvailable` and version `1.0.0`.

## Out Of Scope

- **Timing:** asynchronous provisioning and `Azure-AsyncOperation` polling. Writes complete at once, and actions answer 202 with no body.
- **Placement:** `clusterId`, `bareMetalMachineId` and `availabilityZone` aren't derived, and placement hints aren't enforced.
- **Addresses:** `Dynamic` attachments get no IP address.
- **Egress defaults:** `enabledEgressEndpoints` lists only the additional endpoints.
- **Conditional requests:** `If-Match` and `If-None-Match` on PUT, PATCH and DELETE.
- **Arc integration:** `connectedClusterId` and the managed resource group's contents.

## Quality Gates

- `go test ./internal/server -run TestAzureNetworkCloud -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service networkcloud --require-service networkcloud --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service networkcloud --require-service networkcloud --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service networkcloud --require-service networkcloud --fail-on any`
- the example under `examples/azure/networkcloud-2026-07-01/`, run against a local server
