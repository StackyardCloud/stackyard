# Azure IoT Hub Device Provisioning Service (`deviceprovisioningservices-2022-02-05`) Staged Plan

## Objective

Emulate the IoT Hub Device Provisioning Service control plane (`Microsoft.Devices/provisioningServices`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Keys and the name check follow IoT Hub (`internal/server/provider_azure_iothub.go`).

## References

Microsoft Learn pages for the `2022-02-05` API version:

- `https://learn.microsoft.com/en-us/rest/api/iot-dps/iot-dps-resource`
- `https://learn.microsoft.com/en-us/rest/api/iot-dps/iot-dps-resource/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/iot-dps/iot-dps-resource/list-keys`
- `https://learn.microsoft.com/en-us/rest/api/iot-dps/iot-dps-resource/check-provisioning-service-name-availability`

## Service Naming

- service key: `deviceprovisioningservices`
- display name: `Azure IoT Hub Device Provisioning Service`
- implementation: `internal/server/provider_azure_deviceprovisioningservices.go`
- tests: `internal/server/provider_azure_deviceprovisioningservices_test.go`
- example: `examples/azure/deviceprovisioningservices-2022-02-05/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `provisioningServices` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listkeys` |
| `checkProvisioningServiceNameAvailability` | action | `Invalid` / `AlreadyExists` / available |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names are 3-64 letters, digits or hyphens and cannot start or end with a hyphen (400 `InvalidResourceName`)
- `sku.name` must be `S1`; `allocationPolicy` must be `Hashed` (default), `GeoLatency` or `Static`
- each linked hub in `iotHubs` needs a `connectionString` with a `HostName` and a `location`; responses mask the hub's `SharedAccessKey` as `****`, as Azure does, and report the host name as the hub's `name`
- `serviceOperationsHostName` (`{lower(name)}.azure-devices-provisioning.net`), `deviceProvisioningHostName` (`global.azure-devices-provisioning.net`), a stable `idScope` (`0ne` and eight hex digits) and `state` are server-owned
- `authorizationPolicies` are accepted but never returned; `listkeys` returns the `provisioningserviceowner` policy with keys derived from the resource ID
- the name check compares names across all subscriptions; hub and provisioning service names are checked separately

Out of scope for this stage:
- the device provisioning data plane (enrollments, registrations and device attestation)
- custom authorization policies, certificates and their verification, private endpoints and `listkeys` for a single key name

## Quality Gates

- `go test ./internal/server -run TestAzureDeviceProvisioningServices -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service deviceprovisioningservices`
- `python3 scripts/azure-io-contract-coverage.py --service deviceprovisioningservices`
- `python3 scripts/azure-doc-contract-coverage.py --service deviceprovisioningservices`
