# Azure Arc-enabled Kubernetes (`hybridkubernetes-2024-01-01`) Staged Plan

## Objective

Emulate the Azure Arc-enabled Kubernetes control plane (`Microsoft.Kubernetes/connectedClusters`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Clusters report `Connected` at once; no Arc agents run and no API server listens at the kubeconfig address.

## References

Microsoft Learn pages for the `2024-01-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/hybridkubernetes/connected-cluster/create`
- `https://learn.microsoft.com/en-us/rest/api/hybridkubernetes/connected-cluster/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridkubernetes/connected-cluster/list-cluster-user-credential`

## Service Naming

- service key: `hybridkubernetes`
- display name: `Azure Arc-enabled Kubernetes`
- implementation: `internal/server/provider_azure_hybridkubernetes.go`
- tests: `internal/server/provider_azure_hybridkubernetes_test.go`
- example: `examples/azure/hybridkubernetes-2024-01-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `connectedClusters` | tracked | create, get, update (PATCH), delete, list by group and subscription, `listClusterUserCredential` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- clusters need `identity.type` `SystemAssigned` or `None` and `agentPublicKeyCertificate`; `privateLinkState`, when set, must be `Enabled` or `Disabled`
- `connectivityStatus` (`Connected`) and `lastConnectivityTime` are server-owned; `distribution`, `infrastructure` and `privateLinkState` default to `generic`, `generic` and `Disabled`
- `listClusterUserCredential` needs `authenticationMethod` `Token` or `AAD` and returns one base64 kubeconfig, `credentialUser`, whose server is `https://{name}.arc.stackyard.local`
- extensions and Flux configurations on a cluster are served by `kubernetesconfiguration` and are deleted with it

Out of scope for this stage:
- agent-reported state: Kubernetes version, node and core counts, agent version and certificate expiry
- the cluster connect proxy (`hybridConnectionConfig`) and OIDC issuer, workload identity, gateway and Arc agent profile settings
- `principalId` and `tenantId` on the system-assigned identity
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureConnectedCluster -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service hybridkubernetes`
- `python3 scripts/azure-io-contract-coverage.py --service hybridkubernetes`
- `python3 scripts/azure-doc-contract-coverage.py --service hybridkubernetes`
