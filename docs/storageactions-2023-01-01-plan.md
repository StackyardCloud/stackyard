# Azure Storage Actions (`storageactions-2023-01-01`) Staged Plan

## Objective

Emulate the Azure Storage Actions control plane (`Microsoft.StorageActions`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): storage tasks. No task runs against any blob; conditions are checked for shape only.

## References

Microsoft Learn pages for the `2023-01-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/storageactions/storage-tasks/create`
- `https://learn.microsoft.com/en-us/rest/api/storageactions/storage-tasks/update`
- `https://learn.microsoft.com/en-us/azure/storage-actions/storage-tasks/storage-task-operations`

## Service Naming

- service key: `storageactions`
- display name: `Azure Storage Actions`
- implementation: `internal/server/provider_azure_storageactions.go`
- tests: `internal/server/provider_azure_storageactions_test.go`
- example: `examples/azure/storageactions-2023-01-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `storageTasks` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- task names are 3-18 lowercase letters or digits
- tasks need a managed identity, an `enabled` flag and a `description`
- `action.if` needs a `condition` written as an expression in `[[...]]` and at least one operation; `action.else`, when set, needs at least one operation
- operation names are `SetBlobTier`, `SetBlobLegalHold`, `SetBlobImmutabilityPolicy`, `SetBlobExpiry`, `SetBlobTags`, `DeleteBlob` or `UndeleteBlob`; `onSuccess` can only be `continue` and `onFailure` only `break`
- `creationTimeInUtc` is server-owned and `taskVersion` stays 1

Out of scope for this stage:
- running tasks: storage task assignments (on `Microsoft.Storage` accounts), reports and `previewActions`
- parsing conditions and checking operation parameters
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureStorageTask -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service storageactions`
- `python3 scripts/azure-io-contract-coverage.py --service storageactions`
- `python3 scripts/azure-doc-contract-coverage.py --service storageactions`
