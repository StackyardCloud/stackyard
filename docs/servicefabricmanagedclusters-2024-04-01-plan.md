# Azure Service Fabric Managed Clusters (`servicefabricmanagedclusters-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Service Fabric managed clusters control plane (`Microsoft.ServiceFabric/managedClusters`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): clusters and their node types. Clusters report `Ready` at once; no Service Fabric runtime or VM scale sets run.

## References

Microsoft Learn pages for the `2024-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/servicefabric/managedclusters/managed-clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/servicefabric/managedclusters/node-types/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/servicefabric/managedclusters/node-types/restart`

## Service Naming

- service key: `servicefabricmanagedclusters`
- display name: `Azure Service Fabric Managed Clusters`
- implementation: `internal/server/provider_azure_servicefabricmanagedclusters.go`
- tests: `internal/server/provider_azure_servicefabricmanagedclusters_test.go`
- example: `examples/azure/servicefabricmanagedclusters-2024-04-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `managedClusters` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `managedClusters/nodeTypes` | proxy child | create/update, get, update (PATCH), delete, list, `restart`, `reimage`, `deleteNode` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- clusters need `sku.name` `Basic` or `Standard`, a `dnsName` of 1-63 letters, digits or hyphens and an `adminUserName`; `clientConnectionPort` and `httpGatewayConnectionPort` must be ports and `clusterUpgradeMode` `Automatic` or `Manual`
- `adminPassword` is accepted but never returned
- `clusterId`, `clusterState` (`Ready`), `fqdn` (`{dnsName}.{location}.cloudapp.azure.com`, lower-cased) and `clusterCertificateThumbprints` are server-owned; the ports default to 19000 and 19080 and the upgrade mode and cadence to `Automatic` and `Wave0`
- node types need `isPrimary` and a `vmInstanceCount` of at least -1 (autoscale); `dataDiskSizeGB`, when set, must be positive and `dataDiskType` a managed disk type; `vmSize`, `dataDiskType` and `dataDiskSizeGB` default to `Standard_D2s_v3`, `StandardSSD_LRS` and 256
- `restart`, `reimage` and `deleteNode` need a non-empty `nodes` list and return 202

Out of scope for this stage:
- the Service Fabric runtime: applications, application types, services, and the cluster's VM scale sets, load balancer and public IP
- cross-resource rules such as the primary node type's instance count for each SKU
- cluster versions, upgrade policies, node type SKUs, `start`, `deallocate` and `redeploy`
- long-running operation polling: create, delete and node actions complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSFManagedCluster -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service servicefabricmanagedclusters`
- `python3 scripts/azure-io-contract-coverage.py --service servicefabricmanagedclusters`
- `python3 scripts/azure-doc-contract-coverage.py --service servicefabricmanagedclusters`
