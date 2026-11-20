# Azure Operator Nexus (`networkcloud-2026-07-01`) Staged Plan

## Objective

Emulate the Azure Operator Nexus resource provider (`Microsoft.NetworkCloud`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). This plan covers the infrastructure half:

- **Cluster managers:** create, update, delete, and the relay private endpoint connection action.
- **Clusters:** create, update, delete, and the deploy, update version, continue update version, runtime scan, credential rotation and inspection actions. Child resources are the bare metal machine key sets, BMC key sets and metrics configuration.
- **Hardware:** racks, bare metal machines and storage appliances. A cluster's rack definitions create its own; a client may also create, replace and delete standalone ones directly, PATCH a few fields and run their actions.
- **Access bridges.**
- **Catalogs:** the rack SKU catalog and the Kubernetes version resource.

The workloads half (cloud services networks, L2, L3 and trunked networks, volumes, virtual machines with their consoles, and Kubernetes clusters with their agent pools and features) is in [`networkcloud-2026-07-01-workloads-plan.md`](networkcloud-2026-07-01-workloads-plan.md).

No hardware exists. Every long-running operation completes at once, and actions answer `200` with the finished operation status.

## References

Microsoft Learn pages for the `2026-07-01` API version (`?view=rest-networkcloud-2026-07-01`), the newest stable version Learn documents:

- `https://learn.microsoft.com/en-us/rest/api/networkcloud/` (API overview)
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/operations/list`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/access-bridges/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-versions/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/rack-skus/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/rack-skus/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/racks/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/access-bridges/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/access-bridges/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/access-bridges/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/access-bridges/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/access-bridges/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/cordon`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/power-off`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/reimage`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/replace`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/restart`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/run-command`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/run-data-extracts`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/run-data-extracts-restricted`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/run-read-commands`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/start`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machines/uncordon`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/cluster-managers/update-relay-private-endpoint-connection`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machine-key-sets/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machine-key-sets/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machine-key-sets/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machine-key-sets/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bare-metal-machine-key-sets/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bmc-key-sets/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bmc-key-sets/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bmc-key-sets/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bmc-key-sets/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/bmc-key-sets/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/continue-update-version`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/deploy`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/inspect`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/metrics-configurations/list-by-cluster`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/metrics-configurations/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/metrics-configurations/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/metrics-configurations/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/metrics-configurations/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/rotate-credential`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/scan-runtime`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/clusters/update-version`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-versions/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-versions/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-versions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-versions/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/kubernetes-versions/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/racks/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/racks/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/racks/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/racks/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/racks/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/get`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/update`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/delete`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/disable-remote-vendor-management`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/enable-remote-vendor-management`
- `https://learn.microsoft.com/en-us/rest/api/networkcloud/storage-appliances/run-read-commands`

## Service Naming

- service key: `networkcloud`
- display name: `Azure Operator Nexus (Network Cloud)`
- implementation:
  - `internal/server/provider_azure_networkcloud.go`: shared helpers, cluster managers, Kubernetes versions, key sets, metrics configurations, rack SKUs and operations
  - `internal/server/provider_azure_networkcloud_infra.go`: clusters, their rack definitions and actions
  - `internal/server/provider_azure_networkcloud_machines.go`: racks, bare metal machines, storage appliances and access bridges
- shared handler: `azureARMResource.ExtendedLocation`. PUT stores the top-level `extendedLocation`, PATCH keeps it, and responses return it.
- tests: `internal/server/provider_azure_networkcloud_test.go`, `internal/server/provider_azure_networkcloud_machines_test.go`
- example: `examples/azure/networkcloud-2026-07-01/`

## Operations In Scope

| Resource | Operations |
|---|---|
| Operations | list |
| Cluster managers | create or update, get, update, delete, list by resource group and subscription, update relay private endpoint connection |
| Clusters | create or update, get, update, delete, list by resource group and subscription, deploy, update version, continue update version, scan runtime, rotate credential, inspect |
| Bare metal machine key sets | create or update, get, update, delete, list by cluster |
| BMC key sets | create or update, get, update, delete, list by cluster |
| Metrics configurations (`default`) | create or update, get, update, delete, list by cluster |
| Racks | create or update, get, update, delete, list by resource group and subscription |
| Rack SKUs | get, list by subscription |
| Bare metal machines | get, update, list by resource group and subscription, cordon, uncordon, power off, start, restart, reimage, replace, run command, run read commands, run data extracts, run restricted data extracts; create or update, delete |
| Storage appliances | create or update, get, update, delete, list by resource group and subscription, enable and disable remote vendor management, run read commands |
| Access bridges | create or update, get, update, delete, list by resource group and subscription |
| Kubernetes versions (`default`) | create or update, get, update (tags), delete, list by resource group and subscription |

## Behavior

### Common rules

- Every resource except cluster managers needs `extendedLocation`, with `type` `CustomLocation` or `EdgeZone` and a `name`.
- A re-PUT can't change `extendedLocation`, `kind`, or any property Learn marks create-only. A re-PUT that omits such a property keeps the stored value.
- PATCH accepts only the properties of each resource's patch model. Anything else is rejected as "can't be updated".
- Names follow the path patterns Learn gives:
  - Most resources: 2 to 30 letters, digits, hyphens and underscores.
  - Bare metal machines: 2 to 64 letters and digits.
  - Metrics configurations and Kubernetes versions: `default` only.
- Managed identities:
  - `type` must be `None`, `SystemAssigned`, `UserAssigned` or `SystemAssigned,UserAssigned`.
  - `userAssignedIdentities` is required with a user-assigned type and must name user-assigned identity resource IDs.
  - The service assigns the principal, tenant and client IDs.
- Actions answer `200` with an operation status: `status` `Succeeded`, `resourceId`, and for command runs `properties.exitCode`, `outputHead`, `resultUrl` and `resultRef`.

### Cluster managers

- **Fields:** `fabricControllerId` is required and must be a `Microsoft.ManagedNetworkFabric/networkFabricControllers` ID. `analyticsWorkspaceId`, `availabilityZones`, `fabricControllerId`, `managedResourceGroupConfiguration` and `vmSize` are fixed at creation.
- **PATCH:** changes only tags and identity.
- **Server-set properties:**
  - `managerExtendedLocation`, a custom location named after the manager.
  - `clusterVersions`, the versions clusters may run, with support dates.
  - `relayConfiguration`.
  - The managed resource group defaults to `{name}-managed-{hash}` in the manager's location.
- **Linking clusters:** a cluster whose `extendedLocation` is the manager's `managerExtendedLocation` gets the manager as its `clusterManagerId`. A manager that still manages a cluster can't be deleted (409).
- **Relay connection action:** `updateRelayPrivateEndpointConnection` needs `connectionState` `Approved` or `Rejected`, and a private endpoint `privateEndpointResourceId`.

### Clusters

- **Required fields:**
  - `clusterType` (`SingleRack` or `MultiRack`)
  - `clusterVersion`, one of the versions cluster managers offer
  - `networkFabricId`, a network fabric
  - `aggregatorOrSingleRackDefinition`
- **Rack definitions:**
  - Each needs `networkRackId`, `rackSerialNumber` (1 to 64 characters) and `rackSkuId`, which must name a rack SKU of the catalog. `availabilityZone` is 1 to 10 letters and digits.
  - A `SingleRack` cluster uses a `Single` rack SKU and no compute racks. A `MultiRack` cluster uses an `Aggregator` SKU for the first rack and `Compute` SKUs for its compute racks.
  - Each machine and appliance must sit in a slot the SKU has for it, and each slot is used once per rack.
- **Machine and appliance entries:**
  - A machine needs BMC credentials, a BMC MAC address, a boot MAC address, `rackSlot` (1 to 256) and `serialNumber`. `machineName` is 2 to 64 letters and digits.
  - An appliance needs admin credentials, `rackSlot` and `serialNumber`.
  - Passwords, including the service principal's, are write-only.
- **Optional settings are checked against their enums and ranges:**
  - `computeDeploymentThreshold`, `updateStrategy` (`waitTimeMinutes` 0 to 60) and `secretArchive` (a key vault ID)
  - `runtimeProtectionConfiguration`, `vulnerabilityScanningSettings`
  - `commandOutputSettings` overrides (`commandOutputType`)
  - `clusterServicePrincipal` (all four fields)
- **Identity selectors:** an `associatedIdentity` in the command output, analytics output or secret archive settings must select an identity the cluster has. A system-assigned selector may not name a user identity.
- **Fixed at creation:** `analyticsWorkspaceId`, `networkFabricId` and `managedResourceGroupConfiguration`. A re-PUT may resend part of the managed resource group; the subfields it leaves out keep their stored values. `clusterVersion` changes only through `updateVersion`.
- **Rack definitions:** only a PUT changes `aggregatorOrSingleRackDefinition` and `computeRackDefinitions`, since only a PUT creates hardware. A PATCH that changes them gets 409. Once the cluster leaves `PendingDeployment`, a re-PUT can't change them either (400), so deployed machines are never removed silently.
- **Server-set properties:**
  - `clusterExtendedLocation` and `hybridAksExtendedLocation`, custom locations named after the cluster
  - `availableUpgradeVersions` (the later catalog versions) and `supportExpiryDate`
  - `clusterCapacity`, summed from the rack definitions
  - `managedCredentials`, connection statuses, `workloadResourceIds` and `manualActionCount`
- **Linked hardware:**
  - The rack definitions create one rack per definition, named `{cluster}-rack{n}`. Machines are named after `machineName`, and appliances after `storageApplianceName` or `{cluster}-rack{n}-sa{slot}`.
  - These resources live in the cluster's resource group, on its `clusterExtendedLocation`, and are deleted with the cluster.
  - A re-PUT keeps their state. A name already used by another cluster's hardware, or by standalone hardware, is rejected.
- **Lifecycle:** a new cluster is `PendingDeployment`.
  - `deploy` needs `PendingDeployment` or `Failed`, and `skipValidationsForMachines` must name its machines. It sets the cluster `Running`, its machines `Provisioned`, powered on and ready, and its racks and appliances `Available`.
  - `updateVersion` needs `Running` and a `targetClusterVersion` from `availableUpgradeVersions`. With the `PauseAfterRack` strategy and more than one rack, the cluster stops at `UpdatePaused`. `continueUpdateVersion` needs `UpdatePaused` and finishes the update.
  - `scanRuntime` (`scanActivity`), `rotateCredential` (credentials from `managedCredentials`) and `inspect` (`additionalActions` `ResetHardware`, with `filterDevices` naming the cluster's racks and machines) need a deployed cluster.
  - Each action records its latest run in `actionStates`.

### Key sets and metrics configuration

- **Key sets:**
  - Bare metal machine key sets need `azureGroupId`, `expiration`, `privilegeLevel` (`Standard`, `Superuser` or `Other`, which needs `privilegeLevelName`), `jumpHostsAllowed` (IP addresses) and `userList`.
  - BMC key sets need `azureGroupId`, `expiration`, `privilegeLevel` (`ReadOnly` or `Administrator`) and `userList`.
  - Each user needs `azureUserName`, used once, and `sshPublicKey.keyData`. `osGroupName` is a letter followed by letters and digits, at most 32 characters.
  - PATCH changes `expiration`, `userList` and, on bare metal machine key sets, `jumpHostsAllowed`.
  - Users are `Active` (`AllActive`) until `expiration` passes, then `Invalid` (`AllInvalid`).
- **Metrics configuration:** named `default`, with `collectionInterval` from 1 to 1440 minutes. `enabledMetrics` may name only the cluster's optional metrics, and `disabledMetrics` reports the rest.

### Racks, bare metal machines and storage appliances

Learn says the system manages these resources' lifecycle, but the API defines create or update and delete for them. The emulator handles them this way:

- **Direct PUT:** creates a standalone record outside any cluster, reported ready: racks and appliances `Available`, machines `Provisioned` and powered on. Its actions work at once.
  - Racks need `availabilityZone`, `rackLocation`, `rackSerialNumber` and a catalog `rackSkuId`.
  - Machines need `bmcConnectionString`, BMC credentials, both MAC addresses, `machineDetails`, `machineName`, `machineSkuId`, a rack `rackId`, `rackSlot` and `serialNumber`.
  - Appliances need administrator credentials, a rack `rackId`, `rackSlot`, `serialNumber` and `storageApplianceSkuId`. Of these, credentials, `rackId`, `rackSlot` and `storageApplianceSkuId` are fixed at creation.
  - Passwords are write-only.
- **PUT on a cluster's member:** keeps its cluster (`clusterId`) and server-owned state, such as power, cordon and detailed status.
- **DELETE:** removes a standalone record. A cluster's member gets 409; it goes with its cluster, or when a PUT drops it from the rack definitions before deployment.

- **PATCH:**
  - Racks: `rackLocation` and `rackSerialNumber`.
  - Machines: `machineDetails`.
  - Appliances: `serialNumber`.
- **Machine actions** need a `Provisioned` machine, so its cluster must be deployed.
  - `powerOff` (`skipShutdown`) needs power `On`, `start` needs `Off`, and `restart` needs `On`.
  - `cordon` (`evacuate`) and `uncordon` set `cordonStatus`.
  - `reimage` takes `safeguardMode`.
  - `replace` takes new MAC addresses, serial number, machine name and BMC credentials, plus `safeguardMode` and `storagePolicy`.
- **Command runs:**
  - `runCommand` needs a base64 `script`. The read-command and data-extract actions need one or more `commands`, each with a `command`. All of them need `limitTimeSeconds` from 60 to 14400.
  - Output goes to the cluster's `commandOutputSettings.containerUrl`, or to the override for the command's output type.
- **Appliance actions** need an `Available` appliance. Remote vendor management takes `supportEndpoints` as IP addresses or CIDR ranges and sets `remoteVendorManagementStatus`. `runReadCommands` works as it does on machines.

### Access bridges

- **Fields:** `networkId` (a resource ID) is required.
  - `ipv4ConnectedPrefix` and `ipv6ConnectedPrefix` must be CIDR prefixes of their family.
  - All three are fixed at creation.
- **Security rules:** each needs `direction` (`Inbound` or `Outbound`) and a `port` or port range from 1 to 65535. Addresses are single addresses, CIDR ranges or address ranges of the rule's family.
- **PATCH:** changes only `securityRules`.
- **Server-set properties:** the bridge reports `Running` over TCP, with a `vip` endpoint at the tenth address of each connected prefix.

### Catalogs

- **Rack SKUs:** a subscription-level catalog of three SKUs, one each of `Single`, `Aggregator` and `Compute`, with their control plane, compute and storage slots.
- **Kubernetes versions:** the `default` Kubernetes version resource lists the versions Kubernetes clusters may run. PATCH changes only tags.

## Workloads

Cloud services networks, L2, L3 and trunked networks, volumes, virtual machines (with consoles and their actions) and Kubernetes clusters (with agent pools, features and restart node) are planned in [`networkcloud-2026-07-01-workloads-plan.md`](networkcloud-2026-07-01-workloads-plan.md).

## Out Of Scope

- **Timing:** asynchronous provisioning states and `Location` polling. Long-running operations finish at once and answer `200` with the operation status.
- **Managed resource group:** racks, machines and appliances live in the cluster's own resource group, not its managed resource group.
- **Stored status:** server-set values such as a member's status, addresses and the cluster's capacity are computed when the record is written, not recomputed on each read.
- **Version updates:** `clusterVersion` changes when `updateVersion` starts, not after the last rack.
- **Capacity:** `clusterCapacity` counts every machine and appliance as free.
- **Relay connections:** the relay private endpoint connection decision is validated but not stored, since the relay namespace isn't emulated.
- **Network fabric:** network fabric, network rack and fabric controller IDs are checked for shape only.
- **Identity PATCH merging:** PATCH replaces `identity.userAssignedIdentities` the way PUT does.

## Quality Gates

- `go test ./internal/server -run TestAzureNetworkCloud -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service networkcloud --require-service networkcloud --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service networkcloud --require-service networkcloud --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service networkcloud --require-service networkcloud --fail-on any`
- the example under `examples/azure/networkcloud-2026-07-01/` against a local server
