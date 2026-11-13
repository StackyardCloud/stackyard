# Azure Stream Analytics (`streamanalytics-2020-03-01`) Staged Plan

## Objective

Emulate the Azure Stream Analytics control plane (`Microsoft.StreamAnalytics/streamingjobs`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Jobs start and stop but process no events.

## References

Microsoft Learn pages for the `2020-03-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/streamanalytics/stream-analytics-job/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/streamanalytics/stream-analytics-job/start`
- `https://learn.microsoft.com/en-us/rest/api/streamanalytics/stream-analytics-input/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/streamanalytics/stream-analytics-output/create-or-replace`
- `https://learn.microsoft.com/en-us/rest/api/streamanalytics/stream-analytics-transformation/create-or-replace`

## Service Naming

- service key: `streamanalytics`
- display name: `Azure Stream Analytics`
- implementation: `internal/server/provider_azure_streamanalytics.go`
- tests: `internal/server/provider_azure_streamanalytics_test.go`
- example: `examples/azure/streamanalytics-2020-03-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `streamingjobs` | tracked | create/replace, get, update (PATCH), delete, list by group and subscription, `start`, `stop` |
| `streamingjobs/inputs` | proxy child | create/replace, get, update (PATCH), delete, list |
| `streamingjobs/outputs` | proxy child | create/replace, get, update (PATCH), delete, list |
| `streamingjobs/transformations` | proxy child | create/replace, get, update (PATCH) |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- `properties.sku.name` must be `Standard` or `StandardV2`; `eventsOutOfOrderPolicy` (`Adjust` or `Drop`), `outputErrorPolicy` (`Stop` or `Drop`), `compatibilityLevel` (`1.0`, `1.1` or `1.2`) and `jobType` (`Cloud` or `Edge`) are checked and default to `Adjust`, `Stop`, `1.2` and `Cloud`
- `jobId` and `createdDate` are server-owned; `jobState` is `Created` for a new job, and `start` and `stop` set it to `Running` and `Stopped`
- inputs need `type` `Stream` or `Reference` and a `datasource.type`; outputs need a `datasource.type`
- data source credentials (`sharedAccessPolicyKey`, `password`, `accountKey`, `refreshToken`, `apiKey` and each `storageAccounts[].accountKey` under `datasource.properties`) and the job's `jobStorageAccount.accountKey` and `externals.storageAccount.accountKey` are accepted but never returned
- PUT and PATCH never change `jobState`
- transformations need a `query`; `streamingUnits`, when set, must be 1, 3 or a multiple of 6 up to 396

Out of scope for this stage:
- running queries: no events are read or written
- checking on `start` that the job has inputs, outputs and a transformation, and refusing changes while a job runs
- functions, clusters, private endpoints, `$expand` on job GET, input and output `test`, `scale`, query testing and compilation
- long-running operation polling: create, start and stop complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureStreamAnalytics -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service streamanalytics`
- `python3 scripts/azure-io-contract-coverage.py --service streamanalytics`
- `python3 scripts/azure-doc-contract-coverage.py --service streamanalytics`
