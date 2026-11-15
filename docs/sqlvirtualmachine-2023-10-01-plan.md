# SQL Server on Azure VMs (`sqlvirtualmachine-2023-10-01`) Staged Plan

## Objective

Emulate the SQL Server on Azure Virtual Machines control plane (`Microsoft.SqlVirtualMachine`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): SQL virtual machine registrations, the availability group clusters they join and the clusters' listeners. No SQL IaaS agent runs.

## References

Microsoft Learn pages for the `2023-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/sqlvm/sql-virtual-machines/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/sqlvm/sql-virtual-machines/redeploy`
- `https://learn.microsoft.com/en-us/rest/api/sqlvm/sql-virtual-machines/start-assessment`
- `https://learn.microsoft.com/en-us/rest/api/sqlvm/sql-virtual-machine-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/sqlvm/availability-group-listeners/create-or-update`

## Service Naming

- service key: `sqlvirtualmachine`
- display name: `SQL Server on Azure VMs`
- implementation: `internal/server/provider_azure_sqlvirtualmachine.go`
- tests: `internal/server/provider_azure_sqlvirtualmachine_test.go`
- example: `examples/azure/sqlvirtualmachine-2023-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `sqlVirtualMachines` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `redeploy`, `startAssessment` |
| `sqlVirtualMachineGroups` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `sqlVirtualMachineGroups/availabilityGroupListeners` | proxy child | create/update, get, delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- SQL virtual machines need a `virtualMachineResourceId` naming a `Microsoft.Compute/virtualMachines` resource, which is not required to exist; `sqlVirtualMachineGroupResourceId`, when set, must name a SQL virtual machine group
- `sqlServerLicenseType` (`PAYG`, `AHUB` or `DR`), `sqlManagement` (`Full`, `LightWeight` or `NoAgent`), `sqlImageSku` (`Developer`, `Express`, `Standard`, `Enterprise` or `Web`) and `leastPrivilegeMode` (`Enabled` or `NotSet`) are checked; the license, management mode, image offer and SKU default to `PAYG`, `Full`, `SQL2022-WS2022` and `Developer`
- the SQL authentication password, the Key Vault service principal secret, the backup storage key and password, and all `wsfcDomainCredentials` are accepted but never returned
- `redeploy` and `startAssessment` return 202 and do nothing
- groups need `sqlImageOffer`, and `sqlImageSku` must be `Developer` or `Enterprise`; `scaleType`, `clusterManagerType` and `clusterConfiguration` are server-owned, and the witness `storageAccountPrimaryKey` is never returned
- listeners need `availabilityGroupName` and exactly one of `loadBalancerConfigurations` and `multiSubnetIpConfigurations`; `port`, when set, must be from 1 to 65535

Out of scope for this stage:
- the SQL IaaS agent: no SQL Server detection, storage configuration, patching, backups or assessments
- checking that the VM exists and that the registration name matches it
- troubleshooting, `fetchDCAssessment` and the cluster's `clusterSubnetType` and health
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSQLVirtualMachine -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service sqlvirtualmachine`
- `python3 scripts/azure-io-contract-coverage.py --service sqlvirtualmachine`
- `python3 scripts/azure-doc-contract-coverage.py --service sqlvirtualmachine`
