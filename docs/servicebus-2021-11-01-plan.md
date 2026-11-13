# Azure Service Bus (`servicebus-2021-11-01`) Staged Plan

## Objective

Emulate the Azure Service Bus control plane (`Microsoft.ServiceBus`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2021-11-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane-stable/namespaces`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane-stable/namespaces/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane-stable/namespaces-authorization-rules`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane-stable/queues`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane-stable/topics`
- `https://learn.microsoft.com/en-us/rest/api/servicebus/controlplane-stable/subscriptions`

## Service Naming

- service key: `servicebus`
- display name: `Azure Service Bus`
- implementation: `internal/server/provider_azure_servicebus.go`
- tests: `internal/server/provider_azure_servicebus_test.go`
- example: `examples/azure/servicebus-2021-11-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `namespaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `namespaces/authorizationRules` | proxy child | create/update, get, delete, list, `listKeys` |
| `namespaces/queues` | proxy child | create/update, get, delete, list |
| `namespaces/topics` | proxy child | create/update, get, delete, list |
| `namespaces/topics/subscriptions` | proxy grandchild | create/update, get, delete, list |
| `providers/Microsoft.ServiceBus/CheckNameAvailability` | action | `None` / `InvalidName` / `NameInUse` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- namespace names are 6-50 letters, digits or hyphens, start with a letter, end with a letter or digit, and may not end in `-sb` or `-mgmt` (400 `InvalidResourceName`); they share the `servicebus.windows.net` DNS space with Relay and Notification Hubs namespaces, and these rules live in `internal/server/azure_sas_namespaces.go`
- `sku.name` must be `Basic`, `Standard` or `Premium`
- `serviceBusEndpoint` (the HTTPS endpoint of `{lower(name)}.servicebus.windows.net` on port 443) and `status` `Active` are server-owned
- creating a namespace also creates its `RootManageSharedAccessKey` rule with `Listen`, `Manage` and `Send`, like Azure
- authorization rules need at least one of `Listen`, `Send`, `Manage`, and `Manage` requires the other two
- `listKeys` returns keys derived from the rule ID plus `Endpoint=sb://...` connection strings
- deleting a topic deletes its subscriptions; deleting a namespace deletes everything under it

Out of scope for this stage:
- the messaging data plane (AMQP or HTTP send/receive), so connection strings point at no listener yet
- `regenerateKeys`, queue- and topic-level authorization rules, subscription rules (filters)
- SKU limits (for example, topics are not available on `Basic`)
- disaster recovery configs, migration configs, network rule sets and private endpoints

## Quality Gates

- `go test ./internal/server -run TestAzureServiceBus -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service servicebus`
- `python3 scripts/azure-io-contract-coverage.py --service servicebus`
- `python3 scripts/azure-doc-contract-coverage.py --service servicebus`
