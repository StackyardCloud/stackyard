# Azure Advisor (`advisor-2023-01-01`) Staged Plan

## Objective

Emulate the Azure Advisor control plane (`Microsoft.Advisor`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2023-01-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/advisor/configurations`
- `https://learn.microsoft.com/en-us/rest/api/advisor/recommendations/list`
- `https://learn.microsoft.com/en-us/rest/api/advisor/recommendations/generate`

## Service Naming

- service key: `advisor`
- display name: `Azure Advisor`
- implementation: `internal/server/provider_azure_advisor.go`
- tests: `internal/server/provider_azure_advisor_test.go`
- example: `examples/azure/advisor-2023-01-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `configurations` | extension (subscription and resource group scope) | create/update, get, delete, list |
| `recommendations` | read-only view | list, get |
| `generateRecommendation` | subscription action | generate (202) |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- the only configuration name is `default` (400 `InvalidResourceName`)
- `lowCpuThreshold` must be `5`, `10`, `15`, `20` or `100`, and `exclude` must be a boolean
- each scope lists only the configuration attached to it
- the emulator never generates recommendations: `generateRecommendation` returns 202, the list is empty and getting one returns 404 `ResourceNotFound`

Out of scope for this stage:
- the default configuration Advisor returns before one is saved, and rejecting `lowCpuThreshold` at resource group scope
- suppressions, recommendation metadata, scores, predictions, assessments, workloads and resiliency reviews
- `generateRecommendation/{operationId}` status polling

## Quality Gates

- `go test ./internal/server -run TestAzureAdvisor -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service advisor`
- `python3 scripts/azure-io-contract-coverage.py --service advisor`
- `python3 scripts/azure-doc-contract-coverage.py --service advisor`
