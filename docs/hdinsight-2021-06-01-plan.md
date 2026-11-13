# Azure HDInsight (`hdinsight-2021-06-01`) Staged Plan

## Objective

Emulate the Azure HDInsight control plane (`Microsoft.HDInsight/clusters`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Clusters report `Running` at once and run no Hadoop, Spark or Kafka services.

## References

Microsoft Learn pages for the `2021-06-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/hdinsight/clusters/create`
- `https://learn.microsoft.com/en-us/rest/api/hdinsight/clusters/get`
- `https://learn.microsoft.com/en-us/rest/api/hdinsight/clusters/update`
- `https://learn.microsoft.com/en-us/rest/api/hdinsight/locations/check-name-availability`

## Service Naming

- service key: `hdinsight`
- display name: `Azure HDInsight`
- implementation: `internal/server/provider_azure_hdinsight.go`
- tests: `internal/server/provider_azure_hdinsight_test.go`
- example: `examples/azure/hdinsight-2021-06-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `clusters` | tracked | create, get, update (PATCH, tags), delete, list by group and subscription |
| `locations/{location}/checkNameAvailability` | location action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-59 letters, digits or hyphens, start with a letter and end with a letter or digit (400 `InvalidResourceName`)
- `clusterDefinition.kind` must be `hadoop`, `hbase`, `spark`, `kafka` or `interactivehive`; `tier` must be `Standard` or `Premium` (default `Standard`) and `osType` must be `Linux`; `clusterVersion` defaults to `5.1`
- `computeProfile.roles` must include `headnode` and `workernode`; every role name must be `headnode`, `workernode`, `zookeepernode` or `edgenode`, with a positive `targetInstanceCount`
- `clusterDefinition.configurations` (which hold the gateway credentials), each role's SSH password, `securityProfile.domainUserPassword` and each storage account `key` are accepted but never returned; the SSH user name is kept
- `clusterState` (`Running`), `clusterId`, `createdDate` and the HTTPS (`{name}.azurehdinsight.net:443`) and SSH (`{name}-ssh.azurehdinsight.net:22`) endpoints are server-owned
- `checkNameAvailability` compares names across all subscriptions and regions, because cluster endpoints are DNS names

Out of scope for this stage:
- the Hadoop, Spark, Kafka and Ambari endpoints
- role resize, autoscale changes, gateway settings, applications, script actions, extensions, disk encryption and configuration updates
- validating storage profiles, networks and VM sizes, and `validateCreateRequest`
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureHDInsight -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service hdinsight`
- `python3 scripts/azure-io-contract-coverage.py --service hdinsight`
- `python3 scripts/azure-doc-contract-coverage.py --service hdinsight`
