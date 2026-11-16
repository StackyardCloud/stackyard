# Azure Custom Resource Providers (`customproviders-2018-09-01-preview`) Staged Plan

## Objective

Emulate the Azure custom resource provider control plane (`Microsoft.CustomProviders`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): resource providers and associations. Providers are registered but never called.

## References

Microsoft Learn pages for the `2018-09-01-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/customproviders/custom-resource-provider/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/customproviders/associations/create-or-update`
- `https://learn.microsoft.com/en-us/azure/azure-resource-manager/custom-providers/custom-providers-resources-endpoint-how-to`

## Service Naming

- service key: `customproviders`
- display name: `Azure Custom Resource Providers`
- implementation: `internal/server/provider_azure_customproviders.go`
- tests: `internal/server/provider_azure_customproviders_test.go`
- example: `examples/azure/customproviders-2018-09-01-preview/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `resourceProviders` | tracked | create/update, get, update tags (PATCH, which ignores everything else), delete, list by group and subscription |
| `associations` | extension (any scope) | create/update, get, delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- provider names are 3-64 characters
- actions and resource types each need a unique name without `/` and an HTTPS `endpoint`; actions route only by `Proxy`, resource types by `Proxy` or `Proxy,Cache`; validations are `Swagger` with a `raw.githubusercontent.com` specification
- associations need the ID of an existing custom resource provider as `targetResourceId`; with no PATCH in the API, PATCH returns 405

Out of scope for this stage:
- proxying custom actions and resource types to their endpoints, and Swagger validation of requests
- notifying the provider about associations
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureCustomProviders -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service customproviders`
- `python3 scripts/azure-io-contract-coverage.py --service customproviders`
- `python3 scripts/azure-doc-contract-coverage.py --service customproviders`
