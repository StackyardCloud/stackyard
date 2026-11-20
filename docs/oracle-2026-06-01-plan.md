# Azure Oracle Database@Azure (`oracle-2026-06-01`) Staged Plan

## Objective

Emulate the Oracle Database@Azure resource provider (`Oracle.Database`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). This covers several parts:

- **Purchase:** the marketplace Oracle subscription.
- **Exadata:** Exadata infrastructure and its database servers; Exadata VM clusters with their database nodes and virtual network addresses.
- **Autonomous Database:** Autonomous Databases with their backups.
- **Regional catalogs:** the read-only catalogs that size and version these resources.

Nothing runs in OCI. Lifecycle states settle immediately, no OCI tenancy is linked, and no database accepts connections.

## References

Microsoft Learn pages for the `2026-06-01` API version, the newest Learn documents for these operations:

- `https://learn.microsoft.com/en-us/rest/api/oracle/` (API overview)
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/create-or-update` (singleton `default` at subscription scope; `plan`; `intent` Retain/Reset; `termUnit` is an ISO 8601 duration)
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/update` (update model: plan, intent, productCode)
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/delete`
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/add-azure-subscriptions` (202 Accepted)
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/list-activation-links`
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/list-cloud-account-details`
- `https://learn.microsoft.com/en-us/rest/api/oracle/oracle-subscriptions/list-saas-subscription-details`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/create-or-update`. Required fields: `displayName`, `shape` and `zones`. The page also defines the maintenance window: `customActionTimeoutInMins` 15–120, `leadTimeInWeeks` 1–4, 4-hour `hoursOfDay` slots, and weeks of the month. It allows up to 10 customer contacts.
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/update` (update model: counts, contacts, name, maintenance window, zones)
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/delete`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-exadata-infrastructures/add-storage-capacity` (`activatedStorageCount` and `additionalStorageCount`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/db-servers/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/db-servers/list-by-parent`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/create-or-update`
  - Required fields: infrastructure, `cpuCoreCount`, `displayName`, `giVersion`, `hostname`, `sshPublicKeys`, `subnetId` and `vnetId`.
  - The cluster name rule.
  - `dataStoragePercentage` must be 35, 40, 60 or 80, with a default of 80.
  - Defaults: backup subnet `192.168.252.0/22`, SCAN ports 1521/2484, `LicenseIncluded`.
  - `ocpuCount` allows one decimal place.
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/update` (update model)
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/delete`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/add-vms`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/remove-vms`
- `https://learn.microsoft.com/en-us/rest/api/oracle/cloud-vm-clusters/list-private-ip-addresses` (subnet and VNIC OCIDs; answers an array)
- `https://learn.microsoft.com/en-us/rest/api/oracle/db-nodes/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/db-nodes/list-by-parent`
- `https://learn.microsoft.com/en-us/rest/api/oracle/db-nodes/action` (Start, Stop, SoftReset, Reset)
- `https://learn.microsoft.com/en-us/rest/api/oracle/virtual-network-addresses/create-or-update` (name at most 63 characters; `ipAddress`, `vmOcid`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/virtual-network-addresses/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/virtual-network-addresses/delete`
- `https://learn.microsoft.com/en-us/rest/api/oracle/virtual-network-addresses/list-by-parent`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/create-or-update`
  - Name of 1–30 characters.
  - The `dataBaseType` variants: Regular, Clone, CloneFromBackupTimestamp and CrossRegionDisasterRecovery.
  - `adminPassword` of 12–30 characters.
  - Ranges: `computeCount` 0.1–512, `cpuCoreCount` 1–128, `dataStorageSizeInGbs` 20–393216, `dataStorageSizeInTbs` 1–384, `localAdgAutoFailoverMaxDataLossLimit` 0–3600.
  - The workload, compute model, license, permission, open mode and role enums.
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/get`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/update` (update model)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/delete`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/action` (Start, Stop, Restart)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/failover`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/switchover`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/generate-wallet` (password of at least 8 characters; `generateType` Single/All; base64 `walletFiles`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/restore` (`timestamp`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/shrink`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-databases/change-disaster-recovery-configuration` (`disasterRecoveryType` Adg/BackupBased; snapshot standby)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-backups/create-or-update` (`displayName`, `retentionPeriodInDays`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-backups/get` (`isAutomatic`, `backupType` Incremental/Full/LongTerm)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-backups/update` (only `retentionPeriodInDays`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-backups/delete`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-backups/list-by-parent`
- `https://learn.microsoft.com/en-us/rest/api/oracle/db-system-shapes/list-by-location`
- `https://learn.microsoft.com/en-us/rest/api/oracle/gi-versions/list-by-location` (`shape` filter; SystemShapes enum: Exadata.X9M, Exadata.X11M, Exadata.X11MV, ExaDbXS)
- `https://learn.microsoft.com/en-us/rest/api/oracle/system-versions/list-by-location`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-character-sets/list-by-location`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-national-character-sets/list-by-location`
- `https://learn.microsoft.com/en-us/rest/api/oracle/autonomous-database-versions/list-by-location` (served at `autonomousDbVersions`)
- `https://learn.microsoft.com/en-us/rest/api/oracle/dns-private-views/list-by-location`
- `https://learn.microsoft.com/en-us/rest/api/oracle/dns-private-zones/list-by-location`
- `https://learn.microsoft.com/en-us/azure/oracle/oracle-db/faq-oracle-database-azure` (a quarter rack is 2 database and 3 storage servers, up to 32 and 64)
- `https://learn.microsoft.com/en-us/azure/oracle/oracle-db/oracle-database-network-plan` (clusters attach through a subnet delegated to `Oracle.Database/networkAttachment`)
- `https://learn.microsoft.com/en-us/azure/oracle/oracle-db/onboard-oracle-database` (purchase the offer before provisioning)

## Service Naming

- service key: `oracle`
- display name: `Oracle Database@Azure`
- implementation:
  - `internal/server/provider_azure_oracle.go`
  - `internal/server/provider_azure_oracle_exadata.go`
  - `internal/server/provider_azure_oracle_vmcluster.go`
  - `internal/server/provider_azure_oracle_adb.go`
- shared handler: top-level `zones` and `plan` on ARM records
- tests: `internal/server/provider_azure_oracle_test.go`, `internal/server/provider_azure_oracle_adb_test.go`
- example: `examples/azure/oracle-2026-06-01/`
- unrelated to the GCP `oracledatabase` provider

## Operations In Scope

| Resource | Operations |
|---|---|
| `oracleSubscriptions/default` (subscription scope) | create, get, update, delete, list; POST `listActivationLinks`, `listCloudAccountDetails`, `listSaasSubscriptionDetails`, `addAzureSubscriptions` |
| `cloudExadataInfrastructures` | create, get, update, delete, list by resource group and subscription; POST `addStorageCapacity` |
| `cloudExadataInfrastructures/dbServers` | get, list |
| `cloudVmClusters` | create, get, update, delete, list by resource group and subscription; POST `addVms`, `removeVms`, `listPrivateIpAddresses` |
| `cloudVmClusters/dbNodes` | get, list; POST `action` |
| `cloudVmClusters/virtualNetworkAddresses` | create, get, delete, list |
| `autonomousDatabases` | create, get, update, delete, list by resource group and subscription; POST `action`, `failover`, `switchover`, `generateWallet`, `restore`, `shrink`, `changeDisasterRecoveryConfiguration` |
| `autonomousDatabases/autonomousDatabaseBackups` | create, get, update, delete, list |
| `locations/{location}/dbSystemShapes`, `giVersions`, `systemVersions`, `autonomousDatabaseCharacterSets`, `autonomousDatabaseNationalCharacterSets`, `autonomousDbVersions`, `dnsPrivateViews`, `dnsPrivateZones` | get, list |

## Behavior

### Oracle subscription and errors

- Errors use the `{"error": {"code", "message"}}` payload:
  - `InvalidRequestContent` (400) for rejected writes and actions
  - `Conflict` (409) for actions in the wrong state and for changes to automatic backups
  - `MethodNotAllowed` (405) for writes to read-only collections
- The Oracle subscription is the singleton `default`. Its `plan` holds only Learn's string fields, `termUnit` is an ISO 8601 duration, and `intent` is Retain or Reset.
- Updates change only `plan`, `intent` and `productCode`. `azureSubscriptionIds` changes only through `addAzureSubscriptions`, which takes subscription GUIDs, ignores duplicates and answers 202.
- The OCI cloud account is treated as linked (`cloudAccountState` Available).
- Exadata infrastructure, VM clusters and Autonomous Databases are created only in a subscription that holds the Oracle subscription, or that an Oracle subscription added.

### Exadata infrastructure

- `shape` is Exadata.X9M or Exadata.X11M. `zones` names exactly one zone (1, 2 or 3). `computeCount` is 2–32 (default 2) and `storageCount` 3–64 (default 3).
- The maintenance window is checked against Learn's model, and a CustomPreference needs `hoursOfDay`.
- Sizing is derived from the shape and counts; the shape's per-server capacities are the emulator's catalog values.
- `shape` and the server types are fixed after creation.
- A scale-down can't drop a database server a VM cluster uses.
- Raising `storageCount` records `additionalStorageCount`. `addStorageCapacity` activates those servers and answers 409 when there's nothing to add. Activated storage servers can't be removed.
- `dbServers` is computed: one database server per compute server. Each reports the cores, memory and local storage its VM clusters' nodes take, and the clusters and nodes on it.

### VM clusters, nodes and virtual network addresses

- A VM cluster needs:
  - an existing Exadata infrastructure in the same region
  - a Grid Infrastructure version that the infrastructure's shape supports
  - a `vnetId` and one of its subnets
  - OpenSSH public keys
- Its `dbServers` must be distinct servers of the infrastructure; the default is all of them.
- Each cluster's `cpuCoreCount`, `memorySizeInGbs` and `dbNodeStorageSizeInGbs` split evenly across its nodes, and must fit on each chosen server beside the other clusters there; unsized memory and local storage default to 30 GB and 60 GB a node. `addVms` and `removeVms` keep surviving nodes' state, hostname and fault domain, give new nodes the first free host number, and refuse (409) to remove a node a virtual network address is on.
- The infrastructure, network, hostname, cluster name, domain, Grid Infrastructure version, storage split, SCAN ports and database servers are fixed after creation.
- PATCH accepts only Learn's update model, and resizing is checked for capacity again.
- Each database server gets a database node.
  - `addVms` and `removeVms` add and remove nodes, rechecking capacity, and a cluster keeps at least one node.
  - Nodes can't be written directly. `action` follows the node's state: Start needs Stopped, while Stop, SoftReset and Reset need Available.
  - A node's resource share follows the cluster's current size.
- `listPrivateIpAddresses` lists the nodes on the given VNIC, when the subnet OCID is the cluster's.
- Virtual network addresses take a unique IPv4 address in the cluster (assigned when omitted) and, optionally, one of the cluster's node OCIDs. Neither changes afterwards, and there is no PATCH.
- Deleting a cluster deletes its nodes and addresses and frees its servers.

### Autonomous Databases

- An Autonomous Database checks Learn's ranges and enums.
- Its character sets and version must be in the location catalogs. The defaults are AL32UTF8, AL16UTF16 and 19c, along with ECPU, OLTP, LicenseIncluded and a 1 TB, 2-ECPU size.
- `adminPassword` is never stored or returned.
- What the service sets (`ocid`, `lifecycleState`, `source`, `timeCreated`, connection strings and URLs, private endpoint, `peerDbIds`, the disaster recovery configuration, `timeOfLast*` and used-storage values) is read-only: a PUT may echo it back unchanged, and any other value is rejected (400).
- Clones:
  - A Clone needs `cloneType` and an existing `sourceId`.
  - A CloneFromBackupTimestamp needs `sourceId` and either a past `timestamp` (not before the source existed) or `useLatestAvailableBackupTimeStamp`.
  - Clones inherit their source's character sets and version unless they name their own.
- The data type, character sets, compute model, workload, version, network and source are fixed after creation, and PATCH accepts only Learn's update model.
- `action` follows the database's state: Start needs Stopped, while Stop and Restart need Available. `shrink` and new backups need an Available database.
- `failover` and `switchover` need local Data Guard or a named peer, and record the time.
- `restore` needs a timestamp between the database's creation and now.
- `changeDisasterRecoveryConfiguration` checks its enums, and only a Standby may become a snapshot standby.
- `generateWallet` needs a password of at least 8 characters, which is never stored. It returns a base64 zip holding `tnsnames.ora` and `sqlnet.ora`.
- Each database starts with an automatic Full backup, retained for 60 days (an emulator value). Automatic backups can't be changed or deleted (409).
- A manual backup is LongTerm when it has `retentionPeriodInDays`, and Full otherwise. `timeAvailableTil` follows the retention.

### Regional catalogs

- The catalogs are deterministic seeds shaped like Learn's samples:
  - two Exadata shapes
  - Grid Infrastructure versions 19.0.0.0 and 23.0.0.0, filtered by `shape`
  - two system versions
  - character sets
  - Autonomous Database versions 19c and 23ai
  - one DNS private view and zone per location

## Out Of Scope

- **Other resource types:** Exascale (`exadbVmClusters`, `exascaleDbStorageVaults`, `exascaleDbNodes`, `configureExascale`), `dbSystems`, `networkAnchors`, `resourceAnchors`, GoldenGate connections and deployments, `flexComponents`, `giMinorVersions`, `dbVersions`, `databaseEditions`, `databaseSystemShapeResources`, and the operations list.
- **Cross-region:** CrossRegionDisasterRecovery databases, which are rejected, since one emulator has no second region.
- **Delete checks:** blocking infrastructure deletion while VM clusters use it.
- **Timing:** asynchronous provisioning states and `Azure-AsyncOperation` polling, since every write completes immediately.
- **Network checks:** that the virtual network and delegated subnet exist; only their IDs are checked.

## Quality Gates

- `go test ./internal/server -run TestAzureOracle -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service oracle`
- `python3 scripts/azure-io-contract-coverage.py --service oracle`
- `python3 scripts/azure-doc-contract-coverage.py --service oracle`
