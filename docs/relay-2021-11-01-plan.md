# Azure Relay (`relay-2021-11-01`) Staged Plan

## Objective

Emulate the Azure Relay control plane (`Microsoft.Relay`) on the shared ARM handler (`internal/server/azure_resource_manager.go`), reusing the Service Bus family rules in `internal/server/azure_sas_namespaces.go`.

## References

Microsoft Learn pages for the `2021-11-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/relay/namespaces`
- `https://learn.microsoft.com/en-us/rest/api/relay/namespaces/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/relay/hybrid-connections`
- `https://learn.microsoft.com/en-us/rest/api/relay/wcf-relays`

## Service Naming

- service key: `relay`
- display name: `Azure Relay`
- implementation: `internal/server/provider_azure_relay.go`
- tests: `internal/server/provider_azure_relay_test.go`
- example: `examples/azure/relay-2021-11-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `namespaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `namespaces/authorizationRules` | proxy child | create/update, get, delete, list, `listKeys` |
| `namespaces/hybridConnections` | proxy child | create/update, get, delete, list |
| `namespaces/hybridConnections/authorizationRules` | proxy grandchild | create/update, get, delete, list, `listKeys` |
| `namespaces/wcfRelays` | proxy child | create/update, get, delete, list |
| `namespaces/wcfRelays/authorizationRules` | proxy grandchild | create/update, get, delete, list, `listKeys` |
| `providers/Microsoft.Relay/checkNameAvailability` | action | `None` / `InvalidName` / `NameInUse` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- namespace names follow the Service Bus family rule and share the `servicebus.windows.net` DNS space with Service Bus and Notification Hubs namespaces, so `checkNameAvailability` reports `NameInUse` for a name any of them holds
- `sku.name` must be `Standard`, the only Relay tier
- creating a namespace also creates its `RootManageSharedAccessKey` rule
- WCF relays require `properties.relayType` of `NetTcp` or `Http`
- rules on a hybrid connection or WCF relay add `;EntityPath={entity}` to their connection strings; namespace-level rules do not

Out of scope for this stage:
- the relay data plane (listeners and senders), so `listenerCount` is not reported
- `regenerateKeys` and network rule sets
- private endpoint connections

## Quality Gates

- `go test ./internal/server -run TestAzureRelay -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service relay`
- `python3 scripts/azure-io-contract-coverage.py --service relay`
- `python3 scripts/azure-doc-contract-coverage.py --service relay`
