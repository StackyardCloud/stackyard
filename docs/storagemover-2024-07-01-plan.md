# Azure Storage Mover (`storagemover-2024-07-01`) Staged Plan

## Objective

Emulate the Azure Storage Mover control plane (`Microsoft.StorageMover`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): storage movers with their agents, endpoints, projects and job definitions. No agent runs and no data moves: starting a job returns the ID of a job run that is never stored.

## References

Microsoft Learn pages for the `2024-07-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/storagemover/storage-movers/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagemover/agents/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagemover/endpoints/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagemover/projects/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagemover/job-definitions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/storagemover/job-definitions/start-job`

## Service Naming

- service key: `storagemover`
- display name: `Azure Storage Mover`
- implementation: `internal/server/provider_azure_storagemover.go`
- tests: `internal/server/provider_azure_storagemover_test.go`
- example: `examples/azure/storagemover-2024-07-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `storageMovers` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `storageMovers/agents` | proxy child | create/update, get, update (PATCH), delete, list |
| `storageMovers/endpoints` | proxy child | create/update, get, update (PATCH), delete, list |
| `storageMovers/projects` | proxy child | create/update, get, update (PATCH), delete, list |
| `storageMovers/projects/jobDefinitions` | proxy child | create/update, get, update (PATCH), delete, list, `startJob`, `stopJob` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- agents need an Arc-enabled machine `arcResourceId` and a GUID `arcVmUuid`, and report `agentStatus` `Online` with fixed VM resources and a stable local address
- endpoints take an `endpointType` and its required fields: `AzureStorageBlobContainer` (`storageAccountResourceId`, `blobContainerName`), `AzureStorageSmbFileShare` (`storageAccountResourceId`, `fileShareName`), `NfsMount` (`host`, `export`, and `nfsVersion` `NFSauto`, `NFSv3` or `NFSv4`) and `SmbMount` (`host`, `shareName`); storage account IDs must be storage accounts, and SMB credential `usernameUri` and `passwordUri` must be Key Vault secret URIs
- job definitions need a `copyMode` of `Additive` or `Mirror`, a `sourceName` and a `targetName`; `sourceResourceId`, `targetResourceId` and `agentResourceId` are resolved under the storage mover
- `startJob` and `stopJob` return a stable `jobRunResourceId` under the job definition; `startJob` needs an `agentName`

Out of scope for this stage:
- job runs (`jobRuns` listing and their status), agent registration and heartbeats
- cross-resource checks that the named endpoints and agent exist or match the copy direction
- connections and private endpoints, and long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureStorageMover -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service storagemover`
- `python3 scripts/azure-io-contract-coverage.py --service storagemover`
- `python3 scripts/azure-doc-contract-coverage.py --service storagemover`
