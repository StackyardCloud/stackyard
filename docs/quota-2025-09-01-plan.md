# Azure Quota (`quota-2025-09-01`) Staged Plan

## Objective

Emulate the Azure Quota control plane (`Microsoft.Quota`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): quota limits, usages and quota requests at a resource provider location scope. Every limit change is approved at once.

## References

Microsoft Learn pages for the `2025-09-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/quota/quota/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/quota/quota/list`
- `https://learn.microsoft.com/en-us/rest/api/quota/usages/get`
- `https://learn.microsoft.com/en-us/rest/api/quota/quota-request-status/list`

## Service Naming

- service key: `quota`
- display name: `Azure Quota`
- implementation: `internal/server/provider_azure_quota.go`
- tests: `internal/server/provider_azure_quota_test.go`
- example: `examples/azure/quota-2025-09-01/`

## Operations In Scope

The scope is `/subscriptions/{subscriptionId}/providers/{resourceProvider}/locations/{location}`.

| Resource | Kind | Operations |
|---|---|---|
| `quotas` | extension (provider location) | create/update, get, update (PATCH), list |
| `usages` | read-only view | list, get |
| `quotaRequests` | read-only view | list, get |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- quotas attach only to a provider location scope (400 `InvalidScope` otherwise)
- GET lists default limits (a few common `Microsoft.Compute` and `Microsoft.Network` limits), each replaced by any limit a client set by PUT, followed by limits set for other names
- PUT answers 200, never 201, because the quota always existed; it needs `limit.limitObjectType` `LimitValue` and a non-negative 32-bit integer `limit.value`, and `limit.limitType` is `Independent` or `Shared`
- responses report `unit` `Count`, `isQuotaApplicable` and the limit's display name
- network usages count the emulator's virtual networks, public IP addresses, network security groups, network interfaces, load balancers and route tables in the subscription and location; other usages are 0
- each limit set by PUT appears as one succeeded quota request for its latest change

Out of scope for this stage:
- group quotas, quota allocations and transfers under management groups
- rejecting limits below current usage or above regional capacity, and support tickets
- PATCH of a default limit that was never set by PUT (404)
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureQuota -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service quota`
- `python3 scripts/azure-io-contract-coverage.py --service quota`
- `python3 scripts/azure-doc-contract-coverage.py --service quota`
