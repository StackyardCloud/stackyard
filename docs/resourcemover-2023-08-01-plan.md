# Azure Resource Mover (`resourcemover-2023-08-01`) Staged Plan

## Objective

Emulate the Azure Resource Mover control plane (`Microsoft.Migrate` move collections) on the shared ARM handler (`internal/server/azure_resource_manager.go`): move collections and move resources, with the prepare, move, commit and discard steps. Moves are bookkeeping; nothing is copied or deleted.

## References

Microsoft Learn pages for the `2023-08-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/resourcemover/move-collections/create`
- `https://learn.microsoft.com/en-us/rest/api/resourcemover/move-resources/create`
- `https://learn.microsoft.com/en-us/rest/api/resourcemover/move-collections/prepare`
- `https://learn.microsoft.com/en-us/azure/resource-mover/about-move-process`

## Service Naming

- service key: `resourcemover`
- display name: `Azure Resource Mover`
- implementation: `internal/server/provider_azure_resourcemover.go`
- tests: `internal/server/provider_azure_resourcemover_test.go`
- example: `examples/azure/resourcemover-2023-08-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `moveCollections` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `prepare`, `initiateMove`, `commit`, `discard`, `bulkRemove`, `resolveDependencies` |
| `moveCollections/moveResources` | proxy child | create/update, get, delete, list |
| `moveCollections/unresolvedDependencies` | read-only nested view | list (always empty) |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- `moveType` is `RegionToRegion` (the default, needing different `sourceRegion` and `targetRegion`) or `RegionToZone` (needing `moveRegion`)
- move resources need a resource or resource group `sourceId`, and a `resourceSettings.resourceType` Resource Mover supports when settings are sent; creates answer 200, never 201, like the API, and PATCH returns 405
- a new move resource is `PreparePending`; clients can't set `moveStatus`
- steps list move resources by ID or, with `moveResourceInputType` `MoveResourceSourceId`, by source ID: `prepare` takes `PreparePending` or `PrepareFailed` to `MovePending`, `initiateMove` takes `MovePending` or `MoveFailed` to `CommitPending`, `commit` takes `CommitPending` or `CommitFailed` to `DeleteSourcePending`, and `discard` takes `CommitPending` or `DiscardFailed` back to `MovePending`; `bulkRemove` removes the listed resources (an empty or missing list removes none)
- every listed resource must exist in the collection and be in a starting state, or the step fails with 400 and nothing changes; `validateOnly` checks without changing
- steps answer 200 with a succeeded operation status

Out of scope for this stage:
- dependency analysis: `dependsOn` is empty, `isResolveRequired` is false and `resolveDependencies` does nothing
- `requiredFor`, `targetId`, source resource settings, and failed states
- copying resources to the target region or deleting the source
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureResourceMover -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service resourcemover`
- `python3 scripts/azure-io-contract-coverage.py --service resourcemover`
- `python3 scripts/azure-doc-contract-coverage.py --service resourcemover`
