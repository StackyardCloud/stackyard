# Azure Hybrid Network (`hybridnetwork-2023-09-01`) Staged Plan

## Objective

Emulate Azure Operator Service Manager's resource provider (`Microsoft.HybridNetwork`) on the shared ARM handler (`internal/server/azure_resource_manager.go`). Publishers onboard artifacts, network function definitions, network service designs and configuration group schemas; operators deploy them as network functions and site network services with configuration group values. No artifact is stored, no registry or storage account exists, and nothing deploys to an NFVI.

## References

Microsoft Learn pages for the `2023-09-01` API version, the newest Learn documents for these operations:

- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/` (API versions)
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/list-by-artifact-store`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/list-credential`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-manifests/update-state`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-stores/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-stores/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-stores/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-stores/list-by-publisher`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/artifact-stores/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-schemas/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-schemas/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-schemas/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-schemas/list-by-publisher`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-schemas/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-schemas/update-state`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-values/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-values/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-values/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-values/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-values/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/configuration-group-values/update-tags`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-groups/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-groups/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-groups/list-by-publisher`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-groups/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-versions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-versions/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-versions/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-versions/list-by-network-function-definition-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-versions/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-function-definition-versions/update-state`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/execute-request`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-functions/update-tags`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-groups/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-groups/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-groups/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-groups/list-by-publisher`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-groups/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-versions/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-versions/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-versions/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-versions/list-by-network-service-design-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-versions/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/network-service-design-versions/update-state`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/publishers/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/publishers/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/publishers/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/publishers/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/publishers/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/publishers/update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/site-network-services/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/site-network-services/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/site-network-services/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/site-network-services/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/site-network-services/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/site-network-services/update-tags`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/sites/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/sites/delete`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/sites/get`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/sites/list-by-resource-group`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/sites/list-by-subscription`
- `https://learn.microsoft.com/en-us/rest/api/hybridnetwork/sites/update-tags`
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/publisher-resource-preview-management` (Preview, Active and Deprecated states; Uploading and Uploaded manifests; Preview versions deploy only in their own subscription; publisher names unique in a region; a site network service uses a site in its own region)
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/best-practices-onboard-deploy` (an immutable NFDV needs immutable manifests, and an immutable NSDV immutable schemas, at deployment; a VNF on Azure Core needs a VHD image and an ARM template)
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/configuration-guide` (schemas are JSON Schema; `required` properties must be in the values)
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/glossary` (a CGV adheres to its schema; NSD NFVI names match the site's)
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/how-to-create-site` (NFVI types and their details)
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/resource-cleanup-management` (manifest deletion checks start only with `2025-03-30`)
- `https://learn.microsoft.com/en-us/azure/operator-service-manager/how-to-delete-operator-resources` (deletion order)

## Service Naming

- service key: `hybridnetwork`
- display name: `Azure Hybrid Network (Azure Operator Service Manager)`
- implementation: `internal/server/provider_azure_hybridnetwork.go`, `internal/server/provider_azure_hybridnetwork_versions.go`, `internal/server/provider_azure_hybridnetwork_operator.go`
- tests: `internal/server/provider_azure_hybridnetwork_test.go`, `internal/server/provider_azure_hybridnetwork_operator_test.go`
- example: `examples/azure/hybridnetwork-2023-09-01/`

## Operations In Scope

| Resource | Operations |
|---|---|
| `Microsoft.HybridNetwork/publishers` | create, get, update (PATCH tags), delete, list by resource group and subscription |
| `publishers/{name}/artifactStores` | create, get, update (PATCH tags), delete, list by publisher |
| `artifactStores/{name}/artifactManifests` | create, get, update (PATCH tags), delete, list by artifact store; POST `listCredential`, `updateState` |
| `publishers/{name}/networkFunctionDefinitionGroups`, `networkServiceDesignGroups` | create, get, update (PATCH tags), delete, list by publisher |
| `networkFunctionDefinitionGroups/{name}/networkFunctionDefinitionVersions`, `networkServiceDesignGroups/{name}/networkServiceDesignVersions` | create, get, update (PATCH tags), delete, list by group; POST `updateState` |
| `publishers/{name}/configurationGroupSchemas` | create, get, update (PATCH tags), delete, list by publisher; POST `updateState` |
| `Microsoft.HybridNetwork/configurationGroupValues`, `networkFunctions`, `sites`, `siteNetworkServices` | create, get, update tags (PATCH), delete, list by resource group and subscription; POST `executeRequest` on network functions |

## Behavior

CRUD, listing and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- names follow Learn's patterns and are at most 64 characters: letters, digits, `_` and `-` (starting with a letter or digit) for most resources, no whitespace for artifact manifests and network functions, and SemVer 2.0.0 for definition and design versions. Updates (PATCH) take only `tags`
- a publisher's `scope` is `Private` (Learn's enum and the default) or `Public` (Learn's samples) and can't change; publisher names are unique in a region
- an artifact store is an `AzureContainerRegistry` (default) or `AzureStorageAccount` with `SingleReplication`. It reports its backing registry or storage account in its managed resource group (`ArtifactStore-{name}-{hash}` in its location unless given) as `storageResourceId`; the type, replication and managed resource group can't change
- an artifact manifest lists unique artifacts (`OCIArtifact`, `VhdImageFile`, `ArmTemplate`, `ImageFile`) with a name and version; VHDs go in a storage account store and everything else in a registry store. It starts `Uploading`; `updateState` accepts only `Uploaded` (Learn), after which its artifacts are immutable. `listCredential` returns a fresh scoped registry token (with the artifact repositories), or container SAS URIs for a storage store, valid for 24 hours and never stored
- network function definition versions are `ContainerizedNetworkFunction` on `AzureArcKubernetes` with `HelmPackage` applications, or `VirtualNetworkFunction` on `AzureCore` (`ArmTemplate` and `VhdImageFile`, at least one of each) or `AzureOperatorNexus` (`ArmTemplate`, `ImageFile`). Each application names its artifact in its profile and references an existing artifact store of the right kind; application names are unique and `dependsOnProfile` names only sibling applications; `deployParameters` is a JSON schema
- network service design versions reference existing configuration group schemas and artifact stores; their resource element templates are `ArmResourceDefinition` or `NetworkFunctionDefinition` (`ArmTemplate`, JSON `parameterValues`, unique names); `nfvisFromSite` entries have a name and an NFVI type
- configuration group schemas hold a JSON object `schemaDefinition`
- definition versions, design versions and schemas start `Preview`; `updateState` accepts `Active` or `Deprecated` (and back), answering 200 with the state. Active and Deprecated resources are immutable: a PUT that changes them fails, while tags still change
- configuration group values and network functions are `Open` (`configurationValue`, `deploymentValues`) or `Secret` (`secretConfigurationValue`, `secretDeploymentValues`); values are JSON objects and must have every property their schema (the CGS, or the NFDV's `deployParameters`) marks `required`, recursively through nested `properties` (only that subset of JSON Schema: `allOf`/`anyOf`/`$ref`/`items` and property types aren't checked). Secret values are checked but never returned, and an `idType: Secret` reference keeps its ID hidden
- deployments resolve their reference and report `publisherName`, `publisherScope` and the group, version and offering location. A Deprecated version or schema can't be deployed, a Preview one only from its own subscription, and these checks run on every PUT, so re-PUTting a deployment of a Deprecated version fails while tag updates still work. A definition or design version may reference only its own publisher's artifact stores and schemas
- a network function runs on its definition's NFVI type; Arc and Nexus NFVIs need a custom location `nfviId`. `roleOverrideValues` are JSON objects naming the definition's applications. Deploying an Active definition needs each application's artifact in an Uploaded manifest of its store, matched by name and type (a helm package is an `OCIArtifact`) and, for templates and images, by version. `executeRequest` checks `serviceEndpoint` and `requestMetadata` (a relative path starting with `/`, `Post`/`Put`/`Get`/`Patch`/`Delete`, a body that is valid JSON) and answers 200
- a site's NFVIs have unique names: `AzureCore` with a location, `AzureArcKubernetes` or `AzureOperatorNexus` with a custom location reference, or `Unknown`
- a site network service (SKU `Basic` or `Standard`, tier from the name) uses a site in its own region that has each NFVI its design deploys to (name and type), and one configuration group value per schema the design references, made from that schema. An Active design needs Active schemas. It reports the last state's design version and values, and a managed resource group

Out of scope for this stage:
- network function components (get, list), which report Kubernetes deployment status
- proxy publishers and proxy artifacts, devices (`2021-05-01`), the operations list and long-running operation status endpoints
- the `etag` of network functions, `siteNetworkServiceReferences` on sites, and blocking deletion of resources still referenced (Learn's deletion order is guidance; manifest deletion checks start with `2025-03-30`)
- pushing or pulling artifacts and deploying anything

## Quality Gates

- `go test ./internal/server -run TestAzureHybridNetwork -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service hybridnetwork`
- `python3 scripts/azure-io-contract-coverage.py --service hybridnetwork`
- `python3 scripts/azure-doc-contract-coverage.py --service hybridnetwork`
