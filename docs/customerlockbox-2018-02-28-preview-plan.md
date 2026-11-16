# Customer Lockbox for Microsoft Azure (`customerlockbox-2018-02-28-preview`) Staged Plan

## Objective

Emulate the Customer Lockbox control plane (`Microsoft.CustomerLockbox`): the tenant's opt-in and its access requests. Microsoft engineers never ask for access in the emulator, so there are never any requests.

## References

Microsoft Learn pages for the `2018-02-28-preview` API version:

- `https://learn.microsoft.com/en-us/rest/api/customerlockbox/post/enable-lockbox`
- `https://learn.microsoft.com/en-us/rest/api/customerlockbox/get/tenant-opted-in`
- `https://learn.microsoft.com/en-us/rest/api/customerlockbox/requests/list`

## Service Naming

- service key: `customerlockbox`
- display name: `Customer Lockbox for Microsoft Azure`
- implementation: `internal/server/provider_azure_customerlockbox.go`
- tests: `internal/server/provider_azure_customerlockbox_test.go`
- example: `examples/azure/customerlockbox-2018-02-28-preview/`

## Operations In Scope

| Operation | Behavior |
|---|---|
| `POST /providers/Microsoft.CustomerLockbox/enableLockbox` | opts the tenant in (200) |
| `POST /providers/Microsoft.CustomerLockbox/disableLockbox` | opts the tenant out (200) |
| `GET /providers/Microsoft.CustomerLockbox/tenantOptedIn/{tenantId}` | `isOptedIn` for the emulator's one tenant, false until enabled; other tenant IDs return 404 |
| `GET /subscriptions/{id}/providers/Microsoft.CustomerLockbox/requests` | always empty; GET of a request returns 404 |

Out of scope for this stage:
- access requests and `updateApproval`: there are no requests to approve or deny

## Quality Gates

- `go test ./internal/server -run TestAzureCustomerLockbox -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service customerlockbox`
- `python3 scripts/azure-io-contract-coverage.py --service customerlockbox`
- `python3 scripts/azure-doc-contract-coverage.py --service customerlockbox`
