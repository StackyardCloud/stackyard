# Azure Machine Learning (`machinelearningservices-2024-10-01`) Staged Plan

## Objective

Emulate the Azure Machine Learning control plane (`Microsoft.MachineLearningServices`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): workspaces (including hubs and projects) with their computes, datastores, managed online endpoints and deployments. No compute nodes start, no model is served at the scoring URI and no job runs.

## References

Microsoft Learn pages for the `2024-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/azureml/workspaces/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/azureml/workspaces/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/azureml/compute/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/azureml/datastores/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/azureml/online-endpoints/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/azureml/online-deployments/create-or-update`

## Service Naming

- service key: `machinelearningservices`
- display name: `Azure Machine Learning`
- implementation: `internal/server/provider_azure_machinelearningservices.go`
- tests: `internal/server/provider_azure_machinelearningservices_test.go`
- example: `examples/azure/machinelearningservices-2024-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `workspaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listKeys`, `listStorageAccountKeys`, `resyncKeys` |
| `workspaces/computes` | tracked child | create/update, get, update (PATCH), delete, list, `start`, `stop`, `restart`, `listNodes` |
| `workspaces/datastores` | proxy child | create/update, get, update (PATCH), delete, list |
| `workspaces/onlineEndpoints` | tracked child | create/update, get, update (PATCH), delete, list, `listKeys`, `regenerateKeys`, `token` |
| `workspaces/onlineEndpoints/deployments` | tracked child | create/update, get, update (PATCH), delete, list, `getLogs` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- workspace `kind` is `Default`, `Hub`, `Project` or `FeatureStore`; projects need a workspace `hubResourceId`, and other kinds need `storageAccount` and `keyVault` resource IDs; `applicationInsights` and `containerRegistry`, when set, must be resource IDs of those types
- `workspaceId`, `tenantId`, `discoveryUrl` (host `{location}.api.azureml.ms`, path `/discovery`) and `mlFlowTrackingUri` are server-owned and `publicNetworkAccess` defaults to `Enabled`
- `listKeys` and `listStorageAccountKeys` return the workspace storage account ID, a stable derived storage key and an Application Insights key; `resyncKeys` returns 200
- compute names are 3-24 letters, digits or hyphens starting with a letter; `computeType` is `AmlCompute`, `ComputeInstance` or an attachable type (`AKS`, `Kubernetes`, `VirtualMachine`, `Databricks`, `DataFactory`, `HDInsight`, `SynapseSpark`, `DataLakeAnalytics`), which needs a `resourceId`; AmlCompute needs a non-negative `maxNodeCount` and a `minNodeCount` no greater
- computes report `computeLocation`, `isAttachedCompute` and `provisioningErrors`; created computes default `vmSize` to `STANDARD_DS3_V2`, compute instances report `state` `Running` and clusters `currentNodeCount` 0; admin passwords, private keys, Databricks access tokens and AKS SSL keys are never returned
- `start`, `stop` and `restart` return 202 for compute instances and 400 for other computes, and the instance stays `Running`; `listNodes` returns no nodes
- datastores need a `datastoreType` with its account fields (`AzureBlob`: `accountName`, `containerName`; `AzureFile`: `accountName`, `fileShareName`; `AzureDataLakeGen2`: `accountName`, `filesystem`; `AzureDataLakeGen1`: `storeName`; `OneLake`: `oneLakeWorkspaceName`) and a `credentials.credentialsType`; `credentials.secrets` is never returned, storage datastores default `endpoint` and `protocol`, and `isDefault` is false
- endpoint and deployment names are 3-32 letters, digits or hyphens starting with a letter; endpoints need an `authMode` of `AMLToken`, `Key` or `AADToken` and report `scoringUri` and `swaggerUri` under `{name}.{location}.inference.ml.azure.com`, with empty `traffic` by default
- `listKeys` and `regenerateKeys` (with `keyType` `Primary` or `Secondary`) work only for `Key` endpoints and `token` only for `AMLToken` endpoints; keys are stable derived values that regeneration does not change
- deployments need an `endpointComputeType` of `Managed`, `Kubernetes` or `AzureMLCompute`; `getLogs` returns empty content

Out of scope for this stage:
- assets and jobs: models, environments, code, data, components, jobs, schedules and registries
- connections, outbound rules, private endpoints, serverless and batch endpoints, feature stores' feature sets, quotas and usages
- datastore `listSecrets`, since secrets are not stored, and instance state changes from `start` and `stop`
- long-running operation polling: create, delete and compute actions complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureML -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service machinelearningservices`
- `python3 scripts/azure-io-contract-coverage.py --service machinelearningservices`
- `python3 scripts/azure-doc-contract-coverage.py --service machinelearningservices`
