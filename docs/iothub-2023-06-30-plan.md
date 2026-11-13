# Azure IoT Hub (`iothub-2023-06-30`) Staged Plan

## Objective

Emulate the Azure IoT Hub control plane (`Microsoft.Devices/IotHubs`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). The Device Provisioning Service, in the same namespace, is covered by `docs/deviceprovisioningservices-2022-02-05-plan.md`.

## References

Microsoft Learn pages for the `2023-06-30` API version:

- `https://learn.microsoft.com/en-us/rest/api/iothub/iot-hub-resource`
- `https://learn.microsoft.com/en-us/rest/api/iothub/iot-hub-resource/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/iothub/iot-hub-resource/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/iothub/iot-hub-resource/check-name-availability`
- `https://learn.microsoft.com/en-us/rest/api/iothub/iot-hub-resource/create-event-hub-consumer-group`

## Service Naming

- service key: `iothub`
- display name: `Azure IoT Hub`
- implementation: `internal/server/provider_azure_iothub.go`
- tests: `internal/server/provider_azure_iothub_test.go`
- example: `examples/azure/iothub-2023-06-30/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `IotHubs` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listkeys` |
| `IotHubs/eventHubEndpoints/events/ConsumerGroups` | proxy child | create, get, delete, list |
| `checkNameAvailability` | action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-50 letters, digits or hyphens and cannot start or end with a hyphen (400 `InvalidResourceName`)
- `sku.name` must be `F1`, `S1`, `S2`, `S3`, `B1`, `B2` or `B3`; the built-in endpoint's `partitionCount` must be 2 to 128 (default 4) and `retentionTimeInDays` 1 to 7 (default 1)
- `hostName` (`{lower(name)}.azure-devices.net`), `state` and the built-in endpoint's `partitionIds`, `path` and Service Bus `endpoint` are server-owned
- `authorizationPolicies` are accepted but never returned; `listkeys` returns the five default policies (`iothubowner`, `service`, `device`, `registryRead`, `registryReadWrite`) with keys derived from the resource ID
- every hub starts with the `events` endpoint and its `$Default` consumer group; the only endpoint name is `events`, and consumer groups need `properties.name`
- `checkNameAvailability` compares names across all subscriptions and regions, because host names are global

Out of scope for this stage:
- the device and service data planes (device identities, messaging, twins and direct methods)
- custom authorization policies, `IotHubKeys/{keyName}/listkeys`, routing endpoints and routes, certificates, private endpoints, failover, import/export jobs and quota and stats operations

## Quality Gates

- `go test ./internal/server -run TestAzureIoTHub -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service iothub`
- `python3 scripts/azure-io-contract-coverage.py --service iothub`
- `python3 scripts/azure-doc-contract-coverage.py --service iothub`
