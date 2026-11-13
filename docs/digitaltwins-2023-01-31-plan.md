# Azure Digital Twins (`digitaltwins-2023-01-31`) Staged Plan

## Objective

Emulate the Azure Digital Twins control plane (`Microsoft.DigitalTwins/digitalTwinsInstances`) on the shared ARM handler (`internal/server/azure_resource_manager.go`).

## References

Microsoft Learn pages for the `2023-01-31` API version:

- `https://learn.microsoft.com/en-us/rest/api/digital-twins/controlplane/instance`
- `https://learn.microsoft.com/en-us/rest/api/digital-twins/controlplane/instance/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/digital-twins/controlplane/instance/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/digital-twins/controlplane/endpoints`

## Service Naming

- service key: `digitaltwins`
- display name: `Azure Digital Twins`
- implementation: `internal/server/provider_azure_digitaltwins.go`
- tests: `internal/server/provider_azure_digitaltwins_test.go`
- example: `examples/azure/digitaltwins-2023-01-31/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `digitalTwinsInstances` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `digitalTwinsInstances/endpoints` | proxy child | create/update, get, delete, list |
| `locations/{location}/checkNameAvailability` | action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-63 letters, digits or hyphens and cannot start or end with a hyphen (400 `InvalidResourceName`)
- `hostName` (`{lower(name)}.api.{location}.digitaltwins.azure.net`), `createdTime` and `lastUpdatedTime` are server-owned; Azure uses a short region code where the emulator uses the location name
- endpoints need an `endpointType` of `EventHub`, `EventGrid` or `ServiceBus` and an `authenticationType` of `KeyBased` (default) or `IdentityBased`
- key-based endpoints need the type's key: `connectionStringPrimaryKey` (Event Hubs), `primaryConnectionString` (Service Bus) or `accessKey1` plus `TopicEndpoint` (Event Grid); identity-based endpoints need `endpointUri` and `entityPath`, and Event Grid does not support them
- endpoint connection strings and access keys are accepted but never returned
- the location-scoped `checkNameAvailability` compares names across all subscriptions and regions, because host names are global

Out of scope for this stage:
- the Digital Twins data plane (models, twins, relationships, queries and event routes)
- time series database connections, private endpoints and dead-lettering settings

## Quality Gates

- `go test ./internal/server -run TestAzureDigitalTwins -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service digitaltwins`
- `python3 scripts/azure-io-contract-coverage.py --service digitaltwins`
- `python3 scripts/azure-doc-contract-coverage.py --service digitaltwins`
