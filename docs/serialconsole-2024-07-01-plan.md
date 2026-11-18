# Azure Serial Console (`serialconsole-2024-07-01`) Staged Plan

## Objective

Emulate Azure Serial Console (`Microsoft.SerialConsole`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): serial ports attached to virtual machines and scale set instances, and the subscription switch that turns Serial Console off for all of them. No console runs: connecting returns a WebSocket address that nothing serves.

## References

Microsoft Learn pages for the `2024-07-01` API version (delete is documented for `2018-05-01`):

- `https://learn.microsoft.com/en-us/rest/api/serialconsole/serial-ports/create`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/serial-ports/get`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/serial-ports/list`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/serial-ports/list-by-subscriptions`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/serial-ports/connect`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/serial-ports/delete`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/get-console-status/get-console-status`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/enable-console/enable-console`
- `https://learn.microsoft.com/en-us/rest/api/serialconsole/disable-console/disable-console`

## Service Naming

- service key: `serialconsole`
- display name: `Serial Console`
- implementation: `internal/server/provider_azure_serialconsole.go`
- tests: `internal/server/provider_azure_serialconsole_test.go`
- example: `examples/azure/serialconsole-2024-07-01/`

## Operations In Scope

All paths are under `/subscriptions/{id}`. `{vm}` is `resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachines/{name}` or `resourceGroups/{rg}/providers/Microsoft.Compute/virtualMachineScaleSets/{name}/virtualMachines/{instance}`.

| Resource | Operations |
|---|---|
| `{vm}/providers/Microsoft.SerialConsole/serialPorts` | create (PUT), get, list, delete; POST `connect` |
| `providers/Microsoft.SerialConsole/serialPorts` | list by subscription |
| `providers/Microsoft.SerialConsole/consoleServices/default` | get status; POST `enableConsole`, `disableConsole` |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- serial ports are extension resources of a Microsoft.Compute virtual machine or scale set instance; other scopes get 400 `InvalidScope`, and the resource group must exist
- `properties.state` is `enabled` or `disabled` and `properties.connectionState` is `active` or `inactive` (default `inactive`); there is no PATCH
- a resource's list and GET hold its own ports; the subscription list holds every port in the subscription
- `connect` returns a `wss://` connection string with a container ID derived from the port and a new session ID, or 409 `SerialPortDisabled` while the port is disabled and 409 `SerialConsoleDisabled` while the subscription's Serial Console is off
- Serial Console starts enabled for every subscription; `disableConsole` and `enableConsole` set and return `properties.disabled`, which `consoleServices/default` reports; the switch exists only at subscription scope (400 `InvalidScope` elsewhere) and only as `default`

Out of scope for this stage:
- rejecting GETs at the wrong scope: views answer them, so a list at a non-VM scope (or `consoleServices` below a subscription) is empty and a GET there is 404
- the virtual machine itself: ports don't require a stored Microsoft.Compute resource
- console sessions: the connection string points at a host that serves nothing
- the provider operations list

## Quality Gates

- `go test ./internal/server -run TestAzureSerialConsole -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service serialconsole`
- `python3 scripts/azure-io-contract-coverage.py --service serialconsole`
- `python3 scripts/azure-doc-contract-coverage.py --service serialconsole`
