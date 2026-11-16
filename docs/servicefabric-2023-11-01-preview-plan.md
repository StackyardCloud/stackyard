# Azure Service Fabric (`servicefabric-2023-11-01-preview`) Staged Plan

## Objective

Emulate the classic Azure Service Fabric control plane (`Microsoft.ServiceFabric/clusters`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): clusters with their application types, application type versions, applications and services. Clusters report `Ready` at once; no Service Fabric runtime runs and no application package is downloaded. Managed clusters are covered separately in `docs/servicefabricmanagedclusters-2024-04-01-plan.md`.

## References

Microsoft Learn pages for the `2023-11-01-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/servicefabric/application/clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/servicefabric/application/clusters/list-upgradable-versions`
- `https://learn.microsoft.com/en-us/rest/api/servicefabric/application/application-type-versions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/servicefabric/application/applications/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/servicefabric/application/services/create-or-update`

## Service Naming

- service key: `servicefabric`
- display name: `Azure Service Fabric`
- implementation: `internal/server/provider_azure_servicefabric.go`
- tests: `internal/server/provider_azure_servicefabric_test.go`
- example: `examples/azure/servicefabric-2023-11-01-preview/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `clusters` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listUpgradableVersions` |
| `clusters/applicationTypes` | proxy child | create/update, get, update (PATCH), delete, list |
| `clusters/applicationTypes/versions` | proxy child | create/update, get, update (PATCH), delete, list |
| `clusters/applications` | proxy child | create/update, get, update (PATCH), delete, list |
| `clusters/applications/services` | proxy child | create/update, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- clusters need an http or https `managementEndpoint` and at least one node type, one of them primary; each node type needs a `name`, `clientConnectionEndpointPort` and `httpGatewayEndpointPort` ports, a positive `vmInstanceCount` and `isPrimary`; `reliabilityLevel` must be `None`, `Bronze`, `Silver`, `Gold` or `Platinum` and `upgradeMode` `Automatic` or `Manual`
- `clusterId`, `clusterState` (`Ready`), `clusterEndpoint` (host `{location}.servicefabric.azure.com`, path `/runtime/clusters/{clusterId}`) and `availableClusterVersions` are server-owned; `clusterCodeVersion`, `upgradeMode` and `reliabilityLevel` default to the current runtime version, `Automatic` and `Silver`
- `listUpgradableVersions` returns the current runtime version as the only `supportedPath` entry
- application type versions need an http or https `appPackageUrl` and report an empty `defaultParameterList`
- applications need a `typeName` and `typeVersion`
- services need a `serviceTypeName`, a `serviceKind` of `Stateless` or `Stateful` and a `partitionDescription.partitionScheme` of `Singleton`, `UniformInt64Range` or `Named`; stateless `instanceCount` is an integer of at least -1 and stateful `targetReplicaSetSize` and `minReplicaSetSize` are positive integers

Out of scope for this stage:
- the Service Fabric runtime and the cluster's VM scale sets, load balancer and storage accounts
- cross-resource rules such as an application's type version existing, or reliability levels requiring a primary node count
- cluster certificates and Azure AD settings beyond storing what the client sends, the location-scoped `clusterVersions` and `environments` lists, and managed application identities
- long-running operation polling: create, update and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSFCluster -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service servicefabric`
- `python3 scripts/azure-io-contract-coverage.py --service servicefabric`
- `python3 scripts/azure-doc-contract-coverage.py --service servicefabric`
