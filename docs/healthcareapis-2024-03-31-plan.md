# Azure Health Data Services (`healthcareapis-2024-03-31`) Staged Plan

## Objective

Emulate the Azure Health Data Services control plane (`Microsoft.HealthcareApis`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): workspaces with their FHIR, DICOM and MedTech services. No FHIR or DICOM server answers at the reported URLs and no device data flows.

## References

Microsoft Learn pages for the `2024-03-31` API version:

- `https://learn.microsoft.com/en-us/rest/api/healthcareapis/workspaces/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/healthcareapis/fhir-services/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/healthcareapis/dicom-services/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/healthcareapis/iot-connectors/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/healthcareapis/iot-connector-fhir-destination/create-or-update`

## Service Naming

- service key: `healthcareapis`
- display name: `Azure Health Data Services`
- implementation: `internal/server/provider_azure_healthcareapis.go`
- tests: `internal/server/provider_azure_healthcareapis_test.go`
- example: `examples/azure/healthcareapis-2024-03-31/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `workspaces` | tracked | create/update, get, update (PATCH), delete, list by group and subscription |
| `workspaces/fhirservices` | tracked child | create/update, get, update (PATCH), delete, list |
| `workspaces/dicomservices` | tracked child | create/update, get, update (PATCH), delete, list |
| `workspaces/iotconnectors` | tracked child | create/update, get, update (PATCH), delete, list |
| `workspaces/iotconnectors/fhirdestinations` | tracked child | create/update, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- workspace names are 3-24 lowercase letters or digits; service, connector and destination names are 3-24 lowercase letters, digits or hyphens that start and end with a letter or digit (400 `InvalidResourceName`)
- `publicNetworkAccess` must be `Enabled` (the default) or `Disabled`
- FHIR services need `kind` `fhir-R4` or `fhir-Stu3`; `authenticationConfiguration.authority`, when set, must be an https URL, and `audience` defaults to the service URL https `{workspace}-{service}.fhir.azurehealthcareapis.com`
- DICOM services report `serviceUrl` https `{workspace}-{service}.dicom.azurehealthcareapis.com`, which is server-owned; FHIR and DICOM services report `eventState` `Disabled`
- MedTech connectors need an `ingestionEndpointConfiguration` with `eventHubName`, `consumerGroup` and `fullyQualifiedEventHubNamespace`, and a `deviceMapping.content` object
- FHIR destinations need `resourceIdentityResolutionType` `Create` or `Lookup`, a `fhirServiceResourceId` naming a workspace FHIR service and a `fhirMapping.content` object; the FHIR service is not required to exist

Out of scope for this stage:
- the FHIR and DICOM data planes and MedTech device data
- the legacy Azure API for FHIR (`services`), private endpoints, CORS, export, import and encryption settings
- long-running operation polling: create and delete complete synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureHealthcare -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service healthcareapis`
- `python3 scripts/azure-io-contract-coverage.py --service healthcareapis`
- `python3 scripts/azure-doc-contract-coverage.py --service healthcareapis`
