# Azure SignalR Service (`signalr-2024-03-01`) Staged Plan

## Objective

Emulate the Azure SignalR Service control plane (`Microsoft.SignalRService/SignalR`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Web PubSub, in the same namespace, keeps its own handler (`internal/server/provider_azure_webpubsub.go`).

## References

Microsoft Learn pages for the `2024-03-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/signalr/signal-r`
- `https://learn.microsoft.com/en-us/rest/api/signalr/signal-r/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/signalr/signal-r/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/signalr/signal-r/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/signalr/signal-r-replicas`

## Service Naming

- service key: `signalr`
- display name: `Azure SignalR Service`
- implementation: `internal/server/provider_azure_signalr.go`
- tests: `internal/server/provider_azure_signalr_test.go`
- example: `examples/azure/signalr-2024-03-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `SignalR` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listKeys` |
| `SignalR/replicas` | tracked child | create/update, get, update, delete, list |
| `locations/{location}/checkNameAvailability` | action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-63 characters, start with a letter and end with a letter or digit, the same rule as Web PubSub (400 `InvalidResourceName`)
- `sku.name` must be `Free_F1`, `Standard_S1`, `Premium_P1` or `Premium_P2`; `kind` is `SignalR` (default) or `RawWebSockets`
- `hostName` (`{lower(name)}.service.signalr.net`), `hostNamePrefix`, `publicPort`/`serverPort` 443 and `version` are server-owned
- `listKeys` returns keys derived from the resource ID and connection strings of the form `Endpoint=<https host>;AccessKey={key};Version=1.0;`
- replicas are tracked resources and need their own `location`
- the location-scoped `checkNameAvailability` route is shared with Web PubSub: the Web PubSub handler hands `Microsoft.SignalRService/SignalR` requests to SignalR, which compares names across all subscriptions and regions

Out of scope for this stage:
- the SignalR data plane (hub connections and the server REST API)
- `regenerateKey`, `restart`, `listSkus`, custom domains, custom certificates and shared private link resources
- SKU capacity rules and Premium-only checks for replicas

## Quality Gates

- `go test ./internal/server -run TestAzureSignalR -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service signalr`
- `python3 scripts/azure-io-contract-coverage.py --service signalr`
- `python3 scripts/azure-doc-contract-coverage.py --service signalr`
