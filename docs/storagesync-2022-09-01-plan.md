# Azure File Sync (`storagesync-2022-09-01`) Staged Plan

## Objective

Emulate the Azure File Sync control plane (`Microsoft.StorageSync`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): Storage Sync Services with their sync groups, cloud endpoints, server endpoints and registered servers. No file share is synced and no agent connects: registered servers are records of what the client registers, and endpoints report healthy at once.

## References

Microsoft Learn pages for the `2022-09-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/storagesync/storage-sync-services/create`
- `https://learn.microsoft.com/en-us/rest/api/storagesync/storage-sync-services/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/storagesync/sync-groups/create`
- `https://learn.microsoft.com/en-us/rest/api/storagesync/cloud-endpoints/create`
- `https://learn.microsoft.com/en-us/rest/api/storagesync/server-endpoints/create`
- `https://learn.microsoft.com/en-us/rest/api/storagesync/registered-servers/create`

## Service Naming

- service key: `storagesync`
- display name: `Azure File Sync`
- implementation: `internal/server/provider_azure_storagesync.go`
- tests: `internal/server/provider_azure_storagesync_test.go`
- example: `examples/azure/storagesync-2022-09-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `storageSyncServices` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `locations/{location}/checkNameAvailability` | provider action | POST |
| `storageSyncServices/syncGroups` | proxy child | create, get, delete, list |
| `storageSyncServices/syncGroups/cloudEndpoints` | proxy child | create, get, delete, list, `prebackup`, `postbackup`, `prerestore`, `postrestore`, `restoreheartbeat`, `triggerChangeDetection` |
| `storageSyncServices/syncGroups/serverEndpoints` | proxy child | create, get, update (PATCH), delete, list, `recallAction` |
| `storageSyncServices/registeredServers` | proxy child | create, get, update (PATCH), delete, list, `triggerRollover` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- service names are 1-260 letters, digits, underscores, hyphens or periods, not ending in a period; `checkNameAvailability` reports names in use or invalid
- `incomingTrafficPolicy` is `AllowAllTraffic` (the default) or `AllowVirtualNetworksOnly`; `storageSyncServiceUid` is a stable GUID, `storageSyncServiceStatus` 0, and `useIdentity` is true when the service has an identity
- sync groups report a stable `uniqueId` and `syncGroupStatus` `0`
- cloud endpoints need a storage account `storageAccountResourceId` and an `azureFileShareName`; `storageAccountTenantId`, when set, must be a GUID and defaults to the tenant; `friendlyName` defaults to the share name and the endpoint reports a `partnershipId`; `prebackup`, `prerestore`, `postrestore` and `triggerChangeDetection` return 202, `postbackup` returns backup metadata and `restoreheartbeat` 200
- server endpoints need a `serverLocalPath` and a registered server `serverResourceId`; `cloudTiering` is `on` or `off`, `volumeFreeSpacePercent` 0-100, `tierFilesOlderThanDays` non-negative, and the initial download, initial upload and local cache policies known values; Azure's defaults fill unset fields (`friendlyName` defaults to the registered server ID), and `syncStatus` reports healthy; `recallAction` returns 202
- registered server names are the server's GUID; they need a `serverCertificate` and an `agentVersion`, report the service's `storageSyncServiceUid`, and default `serverRole` to `Standalone` and `friendlyName` to the server ID; `triggerRollover` returns 200

Out of scope for this stage:
- the sync engine: file counts, sync sessions, errors, cloud tiering status and recall progress
- private endpoint connections, workflows and operation status
- cross-resource checks that the share, storage account or registered server exist
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureStorageSync -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service storagesync`
- `python3 scripts/azure-io-contract-coverage.py --service storagesync`
- `python3 scripts/azure-doc-contract-coverage.py --service storagesync`
