# Azure HPC Cache and Managed Lustre (`storagecache-2024-03-01`) Staged Plan

## Objective

Emulate the Azure HPC Cache and Azure Managed Lustre control plane (`Microsoft.StorageCache`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): caches with their storage targets, and Lustre file systems. Nothing is mounted or cached: caches report healthy at stable private addresses, and flushes and archives finish at once.

## References

Microsoft Learn pages for the `2024-03-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/storagecache/caches/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagecache/storage-targets/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagecache/aml-filesystems/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagecache/get-required-aml-fs-subnets-size/get-required-aml-fs-subnets-size`
- `https://learn.microsoft.com/en-us/rest/api/storagecache/check-aml-fs-subnets/check-aml-fs-subnets`

## Service Naming

- service key: `storagecache`
- display name: `Azure HPC Cache and Managed Lustre`
- implementation: `internal/server/provider_azure_storagecache.go`
- tests: `internal/server/provider_azure_storagecache_test.go`
- example: `examples/azure/storagecache-2024-03-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `caches` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `start`, `stop`, `flush`, `upgradeFirmware`, `debugInfo` |
| `caches/storageTargets` | proxy child | create/update, get, delete, list, `suspend`, `resume`, `flush`, `invalidate` |
| `amlFilesystems` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `archive`, `cancelArchive` |
| `getRequiredAmlFSSubnetsSize`, `checkAmlFSSubnets` | provider actions | POST |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- cache, storage target and file system names are 1-80 letters, digits, hyphens or underscores
- caches need an HPC Cache `sku.name` (`Standard_2G`, `Standard_4G`, `Standard_8G`, `Standard_L4_5G`, `Standard_L9G` or `Standard_L16G`) with a `cacheSizeGB` that SKU offers, and a subnet ID; `networkSettings.mtu` is 576-1500
- caches report three stable `mountAddresses`, firmware status and `health`, which only `start` (`Healthy`) and `stop` (`Stopped`) change; `flush`, `upgradeFirmware` and `debugInfo` return 202
- storage targets take `targetType` `nfs3` (needing a host in `nfs3.target`), `clfs` or `blobNfs` (needing a blob container ID) or `unknown`; junction `namespacePath` values start with `/` and are unique, and `nfs3` junctions need an `nfsExport` path; targets start `Ready`, and `suspend` and `resume` change `state`; `flush` and `invalidate` return 202
- Lustre file systems need an `AMLFS-Durable-Premium-40`, `-125`, `-250` or `-500` SKU with a `storageCapacityTiB` that is a multiple of the SKU's step (48, 16, 8 or 4 TiB), up to 4096, a subnet `filesystemSubnet` and a `maintenanceWindow` with a weekday and an `HH:MM` time; HSM containers must be blob container IDs
- file systems report `health` `Available`, `throughputProvisionedMBps` (capacity times the SKU's MB/s per TiB) and `clientInfo` with the MGS address and mount command; `archive` and `cancelArchive` return 200
- `getRequiredAmlFSSubnetsSize` checks the SKU and capacity and returns a subnet size, and `checkAmlFSSubnets` also checks the subnet ID

Out of scope for this stage:
- caching and Lustre data paths, priming jobs, auto-export and auto-import jobs, and space allocation
- Azure's real subnet sizing formula (a flat estimate is returned), firmware upgrades, directory services and encryption settings beyond storing them
- usage models and SKU listings, and long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureStorageCache -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service storagecache`
- `python3 scripts/azure-io-contract-coverage.py --service storagecache`
- `python3 scripts/azure-doc-contract-coverage.py --service storagecache`
