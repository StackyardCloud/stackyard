# Azure Notification Hubs (`notificationhubs-2023-09-01`) Staged Plan

## Objective

Emulate the Azure Notification Hubs control plane (`Microsoft.NotificationHubs`) on the shared ARM handler (`internal/server/azure_resource_manager.go`), reusing the Service Bus family rules in `internal/server/azure_sas_namespaces.go`.

## References

Microsoft Learn pages for the `2023-09-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/notificationhubs/namespaces`
- `https://learn.microsoft.com/en-us/rest/api/notificationhubs/namespaces/check-availability`
- `https://learn.microsoft.com/en-us/rest/api/notificationhubs/notification-hubs`
- `https://learn.microsoft.com/en-us/rest/api/notificationhubs/notification-hubs/list-keys`

## Service Naming

- service key: `notificationhubs`
- display name: `Azure Notification Hubs`
- implementation: `internal/server/provider_azure_notificationhubs.go`
- tests: `internal/server/provider_azure_notificationhubs_test.go`
- example: `examples/azure/notificationhubs-2023-09-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `namespaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `namespaces/authorizationRules` | proxy child | create/update, get, delete, list, `listKeys` |
| `namespaces/notificationHubs` | tracked child | create/update, get, update, delete, list |
| `namespaces/notificationHubs/authorizationRules` | proxy grandchild | create/update, get, delete, list, `listKeys` |
| `providers/Microsoft.NotificationHubs/checkNamespaceAvailability` | action | `isAvailiable` true / false |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- namespace names follow the Service Bus family rule (6-50 characters, letter first, no `-sb` or `-mgmt` suffix) and share the `servicebus.windows.net` DNS space with Service Bus and Relay namespaces
- `sku.name` must be `Free`, `Basic` or `Standard`; `properties.namespaceType` is `NotificationHub` (default) or `Messaging`
- creating a namespace also creates its `RootManageSharedAccessKey` rule
- notification hubs are tracked resources, so they require a `location`
- hub `listKeys` connection strings never carry `EntityPath`, matching Azure; clients pass the hub name separately
- `checkNamespaceAvailability` answers with the API's own misspelled field `isAvailiable`, which SDKs deserialize

Out of scope for this stage:
- sending notifications and device registrations (the data plane)
- platform credentials (APNs, FCM, WNS) and `getPnsCredentials`
- `checkNotificationHubAvailability`, `regenerateKeys` and `debugSend`
- default hub policies (`DefaultListenSharedAccessSignature` and `DefaultFullSharedAccessSignature` must be created explicitly)

## Quality Gates

- `go test ./internal/server -run TestAzureNotificationHubs -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service notificationhubs`
- `python3 scripts/azure-io-contract-coverage.py --service notificationhubs`
- `python3 scripts/azure-doc-contract-coverage.py --service notificationhubs`
