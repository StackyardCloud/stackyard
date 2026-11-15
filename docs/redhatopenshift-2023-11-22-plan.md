# Azure Red Hat OpenShift (`redhatopenshift-2023-11-22`) Staged Plan

## Objective

Emulate the Azure Red Hat OpenShift control plane (`Microsoft.RedHatOpenShift/openShiftClusters`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Clusters report their console and API server addresses at once; no OpenShift control plane or workers run.

## References

Microsoft Learn pages for the `2023-11-22` API version:

- `https://learn.microsoft.com/en-us/rest/api/openshift/open-shift-clusters/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/openshift/open-shift-clusters/list-credentials`
- `https://learn.microsoft.com/en-us/rest/api/openshift/open-shift-clusters/list-admin-credentials`

## Service Naming

- service key: `redhatopenshift`
- display name: `Azure Red Hat OpenShift`
- implementation: `internal/server/provider_azure_redhatopenshift.go`
- tests: `internal/server/provider_azure_redhatopenshift_test.go`
- example: `examples/azure/redhatopenshift-2023-11-22/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `openShiftClusters` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listCredentials`, `listAdminCredentials` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- clusters need a lowercase DNS `clusterProfile.domain`, a `clusterProfile.resourceGroupId` naming a resource group, a `servicePrincipalProfile.clientId`, a `masterProfile.subnetId` and at least one worker profile, each with a `subnetId` and a `count` of at least 3; subnet IDs must name virtual network subnets, which are not required to exist
- `apiserverProfile` and `ingressProfiles` visibility must be `Public` or `Private`
- `clusterProfile.pullSecret` and `servicePrincipalProfile.clientSecret` are accepted but never returned
- the console URL (`https://console-openshift-console.apps.{domain}/`) and API server URL (`https://api.{domain}:6443/`) are server-owned; a domain without a dot is a prefix of `{location}.aroapp.io`
- the version defaults to `4.14.16` and the network to pod CIDR `10.128.0.0/14`, service CIDR `172.30.0.0/16` and outbound type `Loadbalancer`
- `listCredentials` returns the `kubeadmin` user with a stable password; `listAdminCredentials` returns a base64 kubeconfig for the API server URL, at which nothing listens

Out of scope for this stage:
- the OpenShift cluster: no control plane, workers, ingress or API server
- creating the cluster resource group and checking the networks and service principal
- ingress and API server IP addresses, worker profile status, OpenShift versions, machine pools, sync sets and identity providers
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureOpenShift -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service redhatopenshift`
- `python3 scripts/azure-io-contract-coverage.py --service redhatopenshift`
- `python3 scripts/azure-doc-contract-coverage.py --service redhatopenshift`
