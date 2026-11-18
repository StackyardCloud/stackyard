# Microsoft Help (`help-2024-03-01-preview`) Staged Plan

## Objective

Emulate Azure Help (`Microsoft.Help`, the self-help API) on the shared ARM handler (`internal/server/azure_resource_manager.go`): discovery of the diagnostics, solutions, troubleshooters and self-help articles for a problem, and the diagnostics, solutions, simplified solutions and troubleshooters that run them at any scope. Everything runs against a small catalog the emulator ships, with IDs starting `stackyard-`; diagnostics report the catalog's insights without inspecting the scope.

## References

Microsoft Learn pages for the `2024-03-01-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/help/discovery-solution/list`
- `https://learn.microsoft.com/en-us/rest/api/help/discovery-solution-nlp/discover-solutions`
- `https://learn.microsoft.com/en-us/rest/api/help/discovery-solution-nlp/discover-solutions-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/help/solution-self-help/get`
- `https://learn.microsoft.com/en-us/rest/api/help/check-name-availability/check-availability`
- `https://learn.microsoft.com/en-us/rest/api/help/diagnostics/create`
- `https://learn.microsoft.com/en-us/rest/api/help/diagnostics/get`
- `https://learn.microsoft.com/en-us/rest/api/help/solution/create`
- `https://learn.microsoft.com/en-us/rest/api/help/solution/get`
- `https://learn.microsoft.com/en-us/rest/api/help/solution/update`
- `https://learn.microsoft.com/en-us/rest/api/help/solution/warm-up`
- `https://learn.microsoft.com/en-us/rest/api/help/simplified-solutions/create`
- `https://learn.microsoft.com/en-us/rest/api/help/simplified-solutions/get`
- `https://learn.microsoft.com/en-us/rest/api/help/troubleshooters/create`
- `https://learn.microsoft.com/en-us/rest/api/help/troubleshooters/get`
- `https://learn.microsoft.com/en-us/rest/api/help/troubleshooters/continue`
- `https://learn.microsoft.com/en-us/rest/api/help/troubleshooters/end`
- `https://learn.microsoft.com/en-us/rest/api/help/troubleshooters/restart`

## Service Naming

- service key: `help`
- display name: `Help`
- implementation: `internal/server/provider_azure_help.go`, `internal/server/provider_azure_help_catalog.go`
- tests: `internal/server/provider_azure_help_test.go`
- example: `examples/azure/help-2024-03-01-preview/`

## Operations In Scope

`{scope}` is a subscription, resource group or resource.

| Resource | Operations |
|---|---|
| `/providers/Microsoft.Help/discoverySolutions` | list with `$filter` |
| `/providers/Microsoft.Help/discoverSolutions`, `/subscriptions/{id}/providers/Microsoft.Help/discoverSolutions` | POST |
| `/providers/Microsoft.Help/selfHelp/{solutionId}` | get |
| `{scope}/providers/Microsoft.Help/checkNameAvailability` | POST |
| `{scope}/providers/Microsoft.Help/diagnostics` | create, get |
| `{scope}/providers/Microsoft.Help/solutions` | create, get, update (PATCH); POST `warmup` |
| `{scope}/providers/Microsoft.Help/simplifiedSolutions` | create, get |
| `{scope}/providers/Microsoft.Help/troubleshooters` | create, get; POST `continue`, `end`, `restart` |

## Behavior

Service-specific rules:

- the catalog has two problem classifications, `stackyard-vm-connectivity` (a diagnostic, a solution and a troubleshooter for virtual machines) and `stackyard-keyvault-recovery` (a solution, a troubleshooter and a self-help article for Key Vault)
- discovery requires a `ProblemClassificationId` term in every `or` clause of `$filter` and also takes `ResourceType` and `SolutionType`, as `eq` terms joined by `and` and `or`; natural language discovery matches the issue summary and context against each problem's keywords, and `resourceId` against its resource types
- diagnostics run each requested insight: a diagnostic needing `ResourceUri` runs at a resource scope or with a `ResourceUri` parameter and otherwise reports `MissingInputs`; unknown IDs fail; `provisioningState` is `Succeeded`, `PartialComplete` or `Failed` accordingly
- solutions take one `SolutionId` or `ProblemClassificationId` trigger and return the solution's content, sections and replacement maps: the diagnostics embedded in the content (by replacement key) and the problem's troubleshooters; PATCH takes parameters and `ReplacementKey` triggers and reruns the diagnostics
- simplified solutions return a solution's title and plain content
- troubleshooters start at the workflow's first step (`Running`); `continue` answers the current decision step with one of its response options and moves to the next step, ending `Succeeded` at a solution step; `end` cancels a running troubleshooter, `restart` returns to the first step, and continuing an ended one is 409; PUT keeps the workflow and can't change the troubleshooter
- names are checked per scope and type

Out of scope for this stage:
- Microsoft's catalog and real diagnostics against the scope's resources
- metrics charts, videos and web results in replacement maps
- long-running operation polling: every operation completes synchronously
- the provider operations list and the older `2023-09-01-preview` name check

## Quality Gates

- `go test ./internal/server -run TestAzureHelp -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service help`
- `python3 scripts/azure-io-contract-coverage.py --service help`
- `python3 scripts/azure-doc-contract-coverage.py --service help`
