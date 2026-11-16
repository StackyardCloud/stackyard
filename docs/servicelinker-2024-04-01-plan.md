# Azure Service Connector (`servicelinker-2024-04-01`) Staged Plan

## Objective

Emulate the Azure Service Connector control plane (`Microsoft.ServiceLinker`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): linkers and dry runs on any source resource. No identity, firewall, network or app setting change is made; validation always passes and configurations are computed from the target's resource type.

## References

Microsoft Learn pages for the `2024-04-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/servicelinker/linker/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/servicelinker/linker/list-configurations`
- `https://learn.microsoft.com/en-us/rest/api/servicelinker/linkers/generate-configurations`
- `https://learn.microsoft.com/en-us/rest/api/servicelinker/linkers/create-dryrun`
- `https://learn.microsoft.com/en-us/azure/service-connector/reference-client-types`

## Service Naming

- service key: `servicelinker`
- display name: `Azure Service Connector`
- implementation: `internal/server/provider_azure_servicelinker.go`
- tests: `internal/server/provider_azure_servicelinker_test.go`
- example: `examples/azure/servicelinker-2024-04-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `linkers` | extension (resource) | create/update, get, update (PATCH), delete, list, `validateLinker`, `listConfigurations`, `generateConfigurations` |
| `dryruns` | extension (resource) | create/update, get, update (PATCH), delete, list |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- linkers and dry runs attach only to a resource (400 `InvalidScope` otherwise); names are up to 60 letters, digits, periods and underscores
- `targetService.type` is `AzureResource` (with a resource `id`), `ConfluentBootstrapServer`, `ConfluentSchemaRegistry` or `SelfHostedServer` (with an `endpoint`)
- `authInfo.authType` is one of the eight Azure auth types; service principals need `clientId`, `principalId` and a `secret` or `certificate`; `secretInfo.secretType` is `rawValue`, `keyVaultSecretUri` or `keyVaultSecretReference`
- `clientType` and `vNetSolution.type` are checked against Azure's values
- secrets (`authInfo.secret`, `authInfo.certificate`, `authInfo.secretInfo.value`) are accepted but never stored or returned
- `listConfigurations` and `generateConfigurations` return the target endpoint setting (for example `AZURE_STORAGEBLOB_RESOURCEENDPOINT`) plus `_CLIENTID` for a user-assigned identity, or an empty `_CONNECTIONSTRING` for secret and access key auth; `customizedKeys` (stored or sent) rename settings
- dry runs need `parameters.actionName` `createOrUpdate` and valid linker parameters, and preview one `configConnection` operation on the source

Out of scope for this stage:
- location-scoped connectors and dry runs (`resourceGroups/{rg}/providers/Microsoft.ServiceLinker/locations/{location}/...`)
- `configurationNames` and `daprConfigurations`
- checking that the target exists and is reachable, and connection string values
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureServiceLinker -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service servicelinker`
- `python3 scripts/azure-io-contract-coverage.py --service servicelinker`
- `python3 scripts/azure-doc-contract-coverage.py --service servicelinker`
