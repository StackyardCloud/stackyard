# Azure Subscription (`subscription-2021-10-01`) Staged Plan

## Objective

Emulate the Azure Subscription control plane (`Microsoft.Subscription`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): subscription aliases at tenant scope and the subscription `cancel`, `rename` and `enable` actions. Every subscription ID already exists in Stackyard, so an alias only names one and the actions change nothing.

## References

Microsoft Learn pages for the `2021-10-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/subscription/alias/create`
- `https://learn.microsoft.com/en-us/rest/api/subscription/alias/get`
- `https://learn.microsoft.com/en-us/rest/api/subscription/subscription/cancel`
- `https://learn.microsoft.com/en-us/rest/api/subscription/subscription/rename`
- `https://learn.microsoft.com/en-us/rest/api/subscription/subscription/enable`

## Service Naming

- service key: `subscription`
- display name: `Azure Subscription`
- implementation: `internal/server/provider_azure_subscription.go`
- tests: `internal/server/provider_azure_subscription_test.go`
- example: `examples/azure/subscription-2021-10-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `/providers/Microsoft.Subscription/aliases` | tenant | create (PUT), get, delete, list |
| `/subscriptions/{id}/providers/Microsoft.Subscription/cancel` | subscription action | returns the subscription ID |
| `/subscriptions/{id}/providers/Microsoft.Subscription/rename` | subscription action | returns the subscription ID |
| `/subscriptions/{id}/providers/Microsoft.Subscription/enable` | subscription action | returns the subscription ID |

## Behavior

Tenant resources use the shared ARM handler's `Tenant` types; CRUD, listing and error behavior come from it (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- an alias for an existing subscription needs only a GUID `subscriptionId`
- an alias that creates a subscription needs `displayName`, `workload` (`Production` or `DevTest`) and a `billingScope` under `/providers/Microsoft.Billing/billingAccounts/`; its `subscriptionId` is server-owned and stable for the alias name
- `cancel`, `rename` (which needs `subscriptionName`) and `enable` apply only at subscription scope and return `{"subscriptionId": ...}`

Out of scope for this stage:
- creating, renaming or cancelling subscriptions: subscription GETs are unchanged, so Terraform `azurerm_subscription` sees a renamed subscription keep its old name and a cancelled one stay `Enabled`
- placing the new subscription in `additionalProperties.managementGroupId`
- subscription policies, ownership transfer (`acceptOwnership`), billing accounts and tenant moves
- long-running operation polling: alias creation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureSubscriptionAlias -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service subscription`
- `python3 scripts/azure-io-contract-coverage.py --service subscription`
- `python3 scripts/azure-doc-contract-coverage.py --service subscription`
