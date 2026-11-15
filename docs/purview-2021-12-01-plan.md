# Microsoft Purview (`purview-2021-12-01`) Staged Plan

## Objective

Emulate the Microsoft Purview control plane (`Microsoft.Purview/accounts`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Accounts report their catalog, scanning and Atlas Kafka endpoints, which Stackyard does not serve.

## References

Microsoft Learn pages for the `2021-12-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/purview/accounts/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/purview/accounts/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/purview/accounts/add-root-collection-admin`
- `https://learn.microsoft.com/en-us/rest/api/purview/accounts/check-name-availability`

## Service Naming

- service key: `purview`
- display name: `Microsoft Purview`
- implementation: `internal/server/provider_azure_purview.go`
- tests: `internal/server/provider_azure_purview_test.go`
- example: `examples/azure/purview-2021-12-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `accounts` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listkeys`, `addRootCollectionAdmin` |
| `checkNameAvailability` | subscription action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-63 letters, digits or hyphens that start and end with a letter or digit (400 `InvalidResourceName`)
- `identity.type`, when set, must be `SystemAssigned`, `UserAssigned` or `None`; `publicNetworkAccess` (default `Enabled`) and `managedResourcesPublicNetworkAccess` must be `Enabled`, `Disabled` or `NotSpecified`
- `endpoints` (`https://{name}.purview.azure.com/{catalog,guardian,scan}`), `friendlyName`, `createdAt`, `accountStatus` and `managedResources` are server-owned; `managedResourceGroupName` defaults to `managed-rg-{name}`, and the managed storage account and Event Hubs namespace are named but not created
- `listkeys` returns stable primary and secondary Atlas Kafka connection strings; `addRootCollectionAdmin` needs an `objectId`
- `checkNameAvailability` answers with `nameAvailable` and compares names across all subscriptions, because account endpoints are DNS names

Out of scope for this stage:
- the Purview data plane: catalog, scanning, collections and policies
- Kafka configurations, private endpoint connections, default accounts, features and usages
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzurePurview -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service purview`
- `python3 scripts/azure-io-contract-coverage.py --service purview`
- `python3 scripts/azure-doc-contract-coverage.py --service purview`
