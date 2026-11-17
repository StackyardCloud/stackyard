# Azure Data Box (`databox-2025-07-01`) Staged Plan

## Objective

Emulate Azure Data Box (`Microsoft.DataBox`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): checking SKUs and inputs before an order, then ordering, updating, cancelling and deleting Data Box jobs and reading their device credentials. The emulator never prepares or ships a device, so a job stays `DeviceOrdered` until it is cancelled.

## References

Microsoft Learn pages for the `2025-07-01` API version and the order rules it enforces:

- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/create`
- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/update`
- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/cancel`
- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/delete`
- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/list-credentials`
- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/book-shipment-pick-up`
- `https://learn.microsoft.com/en-us/rest/api/databox/jobs/mark-devices-shipped`
- `https://learn.microsoft.com/en-us/rest/api/databox/service/list-available-skus-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/databox/service/validate-inputs`
- `https://learn.microsoft.com/en-us/rest/api/databox/service/region-configuration`
- `https://learn.microsoft.com/en-us/azure/databox/data-box-portal-admin`

## Service Naming

- service key: `databox`
- display name: `Azure Data Box`
- implementation: `internal/server/provider_azure_databox.go`, `internal/server/provider_azure_databox_validate.go`
- tests: `internal/server/provider_azure_databox_test.go`
- example: `examples/azure/databox-2025-07-01/`

## Operations In Scope

All paths are under `/subscriptions/{id}` unless noted.

| Resource | Operations |
|---|---|
| `resourceGroups/{rg}/providers/Microsoft.DataBox/jobs` | create (PUT, 200), get, list by resource group and subscription, update (PATCH), delete; POST `cancel`, `listCredentials`, `bookShipmentPickUp`, `markDevicesShipped` |
| `resourceGroups/{rg}/providers/Microsoft.DataBox/locations/{location}/availableSkus` | POST: SKUs available for a transfer |
| `providers/Microsoft.DataBox/locations/{location}/validateInputs` | POST, also at resource group scope: pre-order checks |
| `providers/Microsoft.DataBox/locations/{location}/regionConfiguration` | POST, also at resource group scope: transport, schedule and device capability options |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- job names are 3 to 24 letters, digits, underscores, hyphens or periods; a job needs a SKU (`DataBox`, `DataBoxDisk`, `DataBoxHeavy`, `DataBoxCustomerDisk`, with the `AzureDataBox120` and `AzureDataBox525` models for `DataBox`), a transfer type, details whose `jobDetailsType` matches the SKU, a contact (name, phone, email list) and a shipping address (street and country)
- imports take `dataImportDetails` (storage accounts, or managed disks with a resource group and staging account); exports take `dataExportDetails` with a `TransferAll` or `TransferUsingFilter` configuration and no device or share passwords
- device and share passwords are 12 to 64 characters with an uppercase letter, a digit and a special character, without `IilLoO0`; they are never returned, and `listCredentials` returns them, or passwords derived from the job ID when none were set
- a `passkey` applies to `DataBoxDisk` jobs only; a null password counts as unset
- a PUT to an existing job is rejected (update with PATCH); PATCH takes tags, identity and the contact, shipping, preference and key details, each an object, and only tags once the job is cancelled; the cancelled stage keeps its time through later updates
- a job can be cancelled (with a reason) only before it is processed, and deleted (200, or 204 when absent) only once cancelled or completed; other attempts answer 409 Conflict
- `bookShipmentPickUp` applies to Microsoft-managed shipping and `markDevicesShipped` to customer-managed shipping; both check their input and then answer 409, since the device never reaches the customer
- every SKU is available everywhere with sample capacities; `validateInputs` checks addresses, transfer details and preferences, and passes the subscription, order limit and SKU checks

Out of scope for this stage:
- device preparation, shipping, data copy and completion: jobs never advance past `DeviceOrdered`
- `$expand=details` (details are always returned), `If-Match` on PATCH, datacenter addresses in `regionConfiguration` and the deprecated `validateAddress`
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureDataBox -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service databox`
- `python3 scripts/azure-io-contract-coverage.py --service databox`
- `python3 scripts/azure-doc-contract-coverage.py --service databox`
