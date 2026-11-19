# Azure Workloads (`workloads-2023-04-01`) Staged Plan

## Objective

Emulate the Azure Workloads resource provider (`Microsoft.Workloads`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): Azure Center for SAP solutions (Virtual Instances for SAP solutions with their central services, database and application server instances, start and stop, and the SAP metadata actions) and Azure Monitor for SAP solutions (monitors, provider instances and the landscape monitor dashboard). No SAP system, VM or monitoring agent runs.

## References

Microsoft Learn pages for the `2023-04-01` API version, which covers every operation in scope:

- `https://learn.microsoft.com/en-us/rest/api/workloads/` (API versions)
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/update`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/start`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-virtual-instances/stop`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/update`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/list`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/start-instance`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-central-instances/stop-instance`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/update`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/list`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/start-instance`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-database-instances/stop-instance`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/update`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/list`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/start-instance`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-application-server-instances/stop-instance`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-sizing-recommendations/sap-sizing-recommendations`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-supported-sku/sap-supported-sku`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-disk-configurations/sap-disk-configurations`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-availability-zone-details/sap-availability-zone-details`
- `https://learn.microsoft.com/en-us/rest/api/workloads/monitors/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/monitors/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/monitors/update`
- `https://learn.microsoft.com/en-us/rest/api/workloads/monitors/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/monitors/list`
- `https://learn.microsoft.com/en-us/rest/api/workloads/monitors/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/workloads/provider-instances/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/provider-instances/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/provider-instances/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/provider-instances/list`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-landscape-monitor/create`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-landscape-monitor/get`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-landscape-monitor/update`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-landscape-monitor/delete`
- `https://learn.microsoft.com/en-us/rest/api/workloads/sap-landscape-monitor/list`
- `https://learn.microsoft.com/en-us/azure/sap/center-sap-solutions/deploy-s4hana` (the VIS name is the three-character SID)
- `https://learn.microsoft.com/en-us/azure/sap/center-sap-solutions/register-existing-system` (registration needs a user-assigned identity; one ASCS instance per SID)
- `https://learn.microsoft.com/en-us/azure/sap/center-sap-solutions/install-software` (software installed after the infrastructure; HA installs need the fencing service principal; detection needs a user-assigned identity)
- `https://learn.microsoft.com/en-us/azure/sap/center-sap-solutions/start-stop-sap-systems` (start and stop)
- `https://learn.microsoft.com/en-us/azure/sap/automation/naming` (SID length)
- `https://learn.microsoft.com/en-us/azure/sap/monitor/quickstart-portal` (managed resource group, Log Analytics workspace)
- `https://learn.microsoft.com/en-us/azure/sap/monitor/enable-tls-azure-monitor-sap-solutions` (root and server certificates)
- `https://learn.microsoft.com/en-us/azure/sap/monitor/provider-hana`, `provider-netweaver`, `provider-linux`, `provider-ha-pacemaker-cluster`, `provider-ibm-db2`, `provider-sql-server` (provider settings)
- `https://learn.microsoft.com/en-us/azure/sap/workloads/hana-vm-premium-ssd-v1` (HANA volume sizes)
- `https://learn.microsoft.com/en-us/azure/virtual-machines/sizes/memory-optimized/m-family` (M-series memory)

## Service Naming

- service key: `workloads`
- display name: `Azure Workloads (Azure Center for SAP solutions and Azure Monitor for SAP solutions)`
- implementation: `internal/server/provider_azure_workloads.go`, `internal/server/provider_azure_workloads_validate.go`, `internal/server/provider_azure_workloads_metadata.go`
- tests: `internal/server/provider_azure_workloads_test.go`
- example: `examples/azure/workloads-2023-04-01/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `Microsoft.Workloads/sapVirtualInstances` | create, get, update (PATCH), delete, list by resource group and subscription; POST `start`, `stop` |
| `sapVirtualInstances/{name}/centralInstances`, `databaseInstances`, `applicationInstances` | create, get, update (PATCH), delete, list; POST `start`, `stop` |
| `/subscriptions/{id}/providers/Microsoft.Workloads/locations/{location}/sapVirtualInstanceMetadata/default/...` | POST `getSizingRecommendations`, `getSapSupportedSku`, `getDiskConfigurations`, `getAvailabilityZoneDetails` |
| `Microsoft.Workloads/monitors` | create, get, update (PATCH), delete, list by resource group and subscription |
| `monitors/{name}/providerInstances` | create, get, delete, list |
| `monitors/{name}/sapLandscapeMonitor` | create (`default`), get, update (PATCH), delete, list |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- a Virtual Instance for SAP solutions (VIS) is named for its SAP system ID: three letters or digits
- a VIS needs `environment` (`NonProd`, `Prod`), `sapProduct` (`ECC`, `S4HANA`, `Other`) and a `configuration`: `Discovery` (registering an existing system from its central server VM), or `Deployment` / `DeploymentWithOSConfig` (an `appLocation` and an `infrastructureConfiguration`, with an optional `softwareConfiguration`)
- registering a system, or detecting externally installed software, needs a user-assigned identity; the identity type is `None` or `UserAssigned`
- infrastructure is `SingleServer` or `ThreeTier` with an `appResourceGroup` and subnet IDs. A three-tier system has central, application and database layers; with `highAvailabilityConfig` (`AvailabilitySet` or `AvailabilityZone`) its central and database layers have two VMs (ASCS and ERS, primary and secondary), otherwise one. High availability needs a three-tier system, and installing an HA system needs the fencing agent's `fencingClientId` and `fencingClientPassword`
- VMs need a size, a full image reference and an OS profile: Learn's administrator name limits and disallowed names, password lengths, complexity and disallowed values, and for Linux without password sign-in an `ssh-rsa` public key
- database disks name only the HANA volumes (`hana/data`, `hana/log`, `hana/shared`, `usr/sap`, `os`, optional `backup`), each with a positive count and size and a known disk SKU; HANA needs all but `backup`
- custom resource names: VM names up to 64 characters, host names up to 13, one network interface per VM, one name entry per VM, and data disk names for the layer's volumes (`default` outside the database layer)
- the transport file share is `CreateAndMount`, `Mount` (with the share and private endpoint IDs) or `Skip`
- passwords, the fencing password and SSH private keys are never returned
- a registered system, or one deployed with its software, is `RegistrationComplete` and `Running`; infrastructure alone is `SoftwareInstallationPending` and `Unavailable` until a PUT adds its `softwareConfiguration`. `managedResourceGroupConfiguration.name` defaults to `mrg-{sid}-{subscription}`
- Learn's update model takes only tags (and, for the VIS and monitors, identity), so other properties can't change on PUT either, except to add software to a system waiting for it
- SAP instances are added only to a registered system, in its location; a system has one central services instance. A database instance reports the SID as `databaseSid`
- `start` and `stop` answer 200 with a finished operation status. Stopping (with an optional non-negative `softStopTimeoutSeconds`) or starting a system does all its instances; doing one instance leaves the system `PartiallyRunning`, or `Running` / `Offline` once all its instances agree. A system can start or stop only once its software is installed (409 otherwise). Deleting a system deletes its instances
- the metadata actions check the system description they take (`appLocation`, `databaseType`, `sapProduct`, and for all but zone details `deploymentType` and `environment`). Sizing picks the smallest M-series size with `dbMemory` GiB and, as a stand-in for SAP's unpublished sizing model, one `Standard_E8ds_v4` application server per 20,000 SAPS; HA doubles the central and database VMs and needs at least two application servers. Supported SKUs list E-series sizes for application servers and M-series sizes for HANA. Disk configurations return the Premium SSD tiers of Learn's HANA storage tables for the size (four `/hana/data` disks, three `/hana/log`, one `/hana/shared` and one root disk), with a P6 for `usr/sap`; DB2 gets only the OS and `usr/sap` volumes. Zone details return zones 1 and 2
- a monitor's `routingPreference` is `Default` or `RouteAll` (default `Default`), its subnet and Log Analytics workspace are resource IDs, and `appLocation` defaults to its location. It reports the storage account and identity in its managed resource group, and a workspace there unless one is given; the managed resource group can't change after creation
- provider instances are `SapHana`, `SapNetWeaver`, `PrometheusOS`, `PrometheusHaCluster`, `Db2` or `MsSqlServer`, with the settings each provider guide asks for; instance numbers are 00 to 99, ports 1 to 65535, Prometheus endpoints http or https URLs, and database providers need a password or a Key Vault password URI (passwords are never returned). `sslPreference` is `Disabled`, `RootCertificate` or `ServerCertificate`, which needs `sslCertificateUri`. The provider type can't change, and provider instances have no update operation (405)
- the landscape monitor is named `default`; its groupings are named lists of SIDs and its thresholds named numbers. Deleting a monitor deletes its providers and dashboard

Out of scope for this stage:
- deploying VMs, installing SAP or collecting monitoring data
- regional zone pairs and SAP's sizing model beyond the stand-ins above
- the operations list and the long-running operation status endpoints

## Quality Gates

- `go test ./internal/server -run TestAzureWorkloads -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service workloads`
- `python3 scripts/azure-io-contract-coverage.py --service workloads`
- `python3 scripts/azure-doc-contract-coverage.py --service workloads`
