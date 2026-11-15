# Azure Policy Insights (`policyinsights-2021-10-01`) Staged Plan

## Objective

Emulate the Azure Policy Insights control plane (`Microsoft.PolicyInsights`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): remediations, attestations and the policy state and event queries. Stackyard evaluates no policy, so remediations deploy nothing and the queries return no records.

## References

Microsoft Learn pages for the `2021-10-01` (remediations), `2022-09-01` (attestations) and `2019-10-01` (policy states and events) API versions:

- `https://learn.microsoft.com/en-us/rest/api/policyinsights/remediations/create-or-update-at-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/policyinsights/remediations/cancel-at-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/policyinsights/remediations/list-deployments-at-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/policyinsights/attestations/create-or-update-at-subscription`
- `https://learn.microsoft.com/en-us/rest/api/policyinsights/policy-states/list-query-results-for-subscription`
- `https://learn.microsoft.com/en-us/rest/api/policyinsights/policy-states/summarize-for-subscription`
- `https://learn.microsoft.com/en-us/rest/api/policyinsights/policy-events/list-query-results-for-subscription`

## Service Naming

- service key: `policyinsights`
- display name: `Azure Policy Insights`
- implementation: `internal/server/provider_azure_policyinsights.go`
- tests: `internal/server/provider_azure_policyinsights_test.go`
- example: `examples/azure/policyinsights-2021-10-01/`

## Operations In Scope

Remediations and attestations are extension resources at a subscription, resource group or resource scope; the queries are POST actions at the same scopes.

| Resource | Kind | Operations |
|---|---|---|
| `remediations` | extension | create/update, get, delete, list, `cancel`, `listDeployments` |
| `attestations` | extension | create/update, get, delete, list |
| `policyStates/{latest,default}/queryResults` | scope action | empty results |
| `policyStates/latest/summarize` | scope action | empty summary |
| `policyEvents/default/queryResults` | scope action | empty results |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- remediations and attestations need a `policyAssignmentId` naming a `Microsoft.Authorization/policyAssignments` resource; the assignment itself is not required to exist
- remediations check `resourceDiscoveryMode` (`ExistingNonCompliant`, the default, or `ReEvaluateCompliance`), `failureThreshold.percentage` (0 to 1) and `resourceCount` and `parallelDeployments` (integers from 1 to 50000)
- `createdOn`, `lastUpdatedOn` and an empty `deploymentStatus` are server-owned; `cancel` sets `provisioningState` to `Canceled`, and `listDeployments` returns no deployments
- attestations check `complianceState` (`Compliant`, `NonCompliant` or `Unknown`, the default) and that `expiresOn` and `assessmentDate` are RFC 3339 times; `lastComplianceStateChangeAt` is server-owned
- the query actions return an OData envelope with `@odata.context`, `@odata.count` 0 and no records; `summarize` returns one summary with zero non-compliant resources and policies

Out of scope for this stage:
- policy evaluation: no compliance records, evaluation triggers or component policy states
- management group scope, policy metadata, policy restrictions, policy tracked resources and `$filter`, `$apply` and `$top` on the queries
- long-running operation polling: create and cancel complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzurePolicyInsights -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service policyinsights`
- `python3 scripts/azure-io-contract-coverage.py --service policyinsights`
- `python3 scripts/azure-doc-contract-coverage.py --service policyinsights`
