# Intelligent Recommendations (`recommendationsservice-2022-02-01`) Staged Plan

## Objective

Emulate Azure Intelligent Recommendations management (`Microsoft.RecommendationsService`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): accounts with their modeling and service endpoint resources, the account status and the name availability check. No models are trained and endpoints serve nothing.

## References

Microsoft Learn pages for the `2022-02-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/get`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/update`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/delete`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/get-status`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/accounts/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/modeling/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/modeling/get`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/modeling/update`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/modeling/delete`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/modeling/list-by-account-resource`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/service-endpoints/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/service-endpoints/get`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/service-endpoints/update`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/service-endpoints/delete`
- `https://learn.microsoft.com/en-us/rest/api/recommendationsservice/service-endpoints/list-by-account-resource`
- `https://learn.microsoft.com/en-us/azure/reliability/regions-paired` (paired locations)

The reference's shared `Properties` definition lists the account's properties on every page; the modeling and service endpoint properties here come from each page's request and response examples and its enum definitions.

## Service Naming

- service key: `recommendationsservice`
- display name: `Intelligent Recommendations`
- implementation: `internal/server/provider_azure_recommendationsservice.go`
- tests: `internal/server/provider_azure_recommendationsservice_test.go`
- example: `examples/azure/recommendationsservice-2022-02-01/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `Microsoft.RecommendationsService/accounts` | create or update, get, update (PATCH), delete, list by resource group and subscription; POST `status` |
| `accounts/{name}/modeling` | create or update, get, update (PATCH), delete, list |
| `accounts/{name}/serviceEndpoints` | create or update, get, update (PATCH, tags), delete, list |
| `/subscriptions/{id}/providers/Microsoft.RecommendationsService/checkNameAvailability` | POST |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- an account's `configuration` (`Free` or `Capacity`) and a modeling resource's `features` (`Basic`, `Standard` or `Premium`) can only be set at creation: a later PUT or PATCH that sets a different value is rejected, and one that omits it keeps it
- account names are unique across subscriptions, and `checkNameAvailability` reports a used name as `AlreadyExists`; it takes the accounts type only
- endpoint authentications take a `principalType` of `Application` or `User`; CORS rules take arrays of strings and a non-negative `maxAgeInSeconds`
- modeling `frequency` is `Low`, `Medium` or `High` and `size` is `Small`, `Medium` or `Large`; input data is a connection string
- account `status` is empty without modeling resources and otherwise lists each modeling resource's `DataAccess`, `DataValidation` and `Model` stages: all `Success` with an input data connection string, or `Failed`, `Pending`, `Pending` without one
- service endpoints take a non-negative whole `preAllocatedCapacity`; `url` is `{account}-{endpoint}.recommendations.stackyard.invalid` and `pairedLocation` is the location's Azure region pair (absent for regions without one); PATCH changes tags only
- deleting an account deletes its modeling and service endpoint resources

Out of scope for this stage:
- training models, reading input data, writing reports and serving recommendations
- the `operationStatuses` resource: every operation completes synchronously, so none is pending
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzureRecommendations -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service recommendationsservice`
- `python3 scripts/azure-io-contract-coverage.py --service recommendationsservice`
- `python3 scripts/azure-doc-contract-coverage.py --service recommendationsservice`
