# Azure Resource Health (`resourcehealth-2022-10-01`) Staged Plan

## Objective

Emulate the Azure Resource Health API (`Microsoft.ResourceHealth`) with read-only views on the shared ARM handler (`internal/server/azure_resource_manager.go`). Nothing in the emulator fails, so every resource is healthy and no service health events are active.

## References

Microsoft Learn pages for the `2022-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/resourcehealth/availability-statuses/list-by-subscription-id`
- `https://learn.microsoft.com/en-us/rest/api/resourcehealth/availability-statuses/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/resourcehealth/availability-statuses/get-by-resource`
- `https://learn.microsoft.com/en-us/rest/api/resourcehealth/events/list-by-subscription-id`

## Service Naming

- service key: `resourcehealth`
- display name: `Azure Resource Health`
- implementation: `internal/server/provider_azure_resourcehealth.go`
- tests: `internal/server/provider_azure_resourcehealth_test.go`
- example: `examples/azure/resourcehealth-2022-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `availabilityStatuses` | read-only view | list by subscription, resource group and resource; get `current` for a resource |
| `events` | read-only view | list, get |

## Behavior

Views answer GET only (other methods return 405 `MethodNotAllowed`), and a name the view doesn't list returns 404 `ResourceNotFound`. Service-specific rules:

- subscription and resource group lists hold one `current` status per tracked resource on the shared ARM handler, with the resource's location. Proxy and extension resources are not listed
- every status is `Available`, with `occuredTime` set to the resource's creation time
- `{resourceId}/providers/Microsoft.ResourceHealth/availabilityStatuses/current` answers for any resource ID; `current` at subscription or resource group scope returns 404
- the event list is always empty

Out of scope for this stage:
- resources kept in a service's own store rather than on the shared ARM handler: they are missing from the subscription and resource group lists, and no resource is checked for existence
- `$filter` and `$expand`, the child resource statuses, impacted resources, emerging issues, metadata and security advisories

## Quality Gates

- `go test ./internal/server -run TestAzureResourceHealth -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service resourcehealth`
- `python3 scripts/azure-io-contract-coverage.py --service resourcehealth`
- `python3 scripts/azure-doc-contract-coverage.py --service resourcehealth`
