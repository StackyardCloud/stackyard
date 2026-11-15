# Azure IoT Central (`iotcentral-2021-06-01`) Staged Plan

## Objective

Emulate the Azure IoT Central control plane (`Microsoft.IoTCentral/iotApps`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Applications report `created` at once; no IoT Central application is served at the subdomain.

## References

Microsoft Learn pages for the `2021-06-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/iotcentral/apps/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/iotcentral/apps/get`
- `https://learn.microsoft.com/en-us/rest/api/iotcentral/apps/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/iotcentral/apps/check-subdomain-availability`

## Service Naming

- service key: `iotcentral`
- display name: `Azure IoT Central`
- implementation: `internal/server/provider_azure_iotcentral.go`
- tests: `internal/server/provider_azure_iotcentral_test.go`
- example: `examples/azure/iotcentral-2021-06-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `iotApps` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `checkNameAvailability` | subscription action | `Invalid` / `AlreadyExists` / available |
| `checkSubdomainAvailability` | subscription action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names and subdomains are 2-63 lowercase letters, digits or hyphens that start and end with a letter or digit (400 `InvalidResourceName` for a name)
- applications need `sku.name` `ST0`, `ST1` or `ST2` and a `subdomain`; `identity.type`, when set, must be `SystemAssigned` or `None`, and `publicNetworkAccess` `Enabled` (the default) or `Disabled`
- `applicationId` and `state` (`created`) are server-owned; `displayName` defaults to the name and `template` to `iotc-pnp-preview@1.0.0`
- both availability checks answer with `nameAvailable` and compare across all subscriptions; `checkSubdomainAvailability` compares `subdomain` values

Out of scope for this stage:
- the IoT Central application and its REST API
- rejecting a PUT whose subdomain another application already uses
- network rule sets, private endpoint connections, private link resources and templates
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureIoTCentral -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service iotcentral`
- `python3 scripts/azure-io-contract-coverage.py --service iotcentral`
- `python3 scripts/azure-doc-contract-coverage.py --service iotcentral`
