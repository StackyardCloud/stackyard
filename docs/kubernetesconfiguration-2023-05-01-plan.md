# Azure Kubernetes Configuration (`kubernetesconfiguration-2023-05-01`) Staged Plan

## Objective

Emulate the Azure Kubernetes configuration control plane (`Microsoft.KubernetesConfiguration`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): cluster extensions and Flux (GitOps) configurations on AKS and Arc-enabled clusters. Nothing is installed and no repository is synced.

## References

Microsoft Learn pages for the `2023-05-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/kubernetesconfiguration/extensions/create`
- `https://learn.microsoft.com/en-us/rest/api/kubernetesconfiguration/extensions/get`
- `https://learn.microsoft.com/en-us/rest/api/kubernetesconfiguration/flux-configurations/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/kubernetesconfiguration/flux-configurations/update`

## Service Naming

- service key: `kubernetesconfiguration`
- display name: `Azure Kubernetes Configuration`
- implementation: `internal/server/provider_azure_kubernetesconfiguration.go`
- tests: `internal/server/provider_azure_kubernetesconfiguration_test.go`
- example: `examples/azure/kubernetesconfiguration-2023-05-01/`

## Operations In Scope

Both types are extension resources at `{cluster}/providers/Microsoft.KubernetesConfiguration/{type}/{name}`.

| Resource | Kind | Operations |
|---|---|---|
| `extensions` | extension | create, get, update (PATCH), delete, list |
| `fluxConfigurations` | extension | create/update, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- the scope must be a `Microsoft.ContainerService/managedClusters`, `Microsoft.Kubernetes/connectedClusters` or `Microsoft.HybridContainerService/provisionedClusters` or `Microsoft.ResourceConnector/appliances` resource (400 `InvalidScope`); the cluster itself is not required to exist, because AKS clusters live in their own store
- extensions need `extensionType`, and `scope` may set `cluster` or `namespace` but not both; `currentVersion` (the requested `version`, else `1.0.0`) and `isSystemExtension` are server-owned, and `autoUpgradeMinorVersion` and `releaseTrain` default to `true` and `Stable`
- Flux configurations check `scope` (`cluster` or `namespace`, default `cluster`) and `sourceKind` (`GitRepository`, `Bucket`, `AzureBlob` or `OCIRepository`, default `GitRepository`) and need the matching source's `url`; `namespace` defaults to `default`, each kustomization must be an object, and `complianceState` (`Compliant`), `statuses` and `sourceUpdatedAt` are server-owned
- `configurationProtectedSettings`, and on Flux configurations the `azureBlob` `accountKey`, `sasToken` and service principal `clientSecret`, `clientCertificate` and `clientCertificatePassword`, are accepted but never returned
- they are deleted with an Arc-enabled cluster they attach to, but not with an AKS cluster, which lives in its own store

Out of scope for this stage:
- installing extensions and syncing repositories: no Helm releases, Flux controllers or per-object statuses
- extension identities and plans, extension type and version discovery, source control configurations (Flux v1), private link scopes and operation status
- removing map entries with `null` in a PATCH: the shared handler stores the `null`
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureKubernetesConfiguration -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service kubernetesconfiguration`
- `python3 scripts/azure-io-contract-coverage.py --service kubernetesconfiguration`
- `python3 scripts/azure-doc-contract-coverage.py --service kubernetesconfiguration`
