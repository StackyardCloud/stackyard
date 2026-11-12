# Azure Spring Apps (`azurespringapps-2023-12-01`) Staged Plan

## Objective

Emulate the Azure Spring Apps ARM management API for `2023-12-01` with deterministic local behavior for:

- provider `operations`
- top-level `services` lifecycle and service-scoped actions
- nested `apps`, `deployments`, and app actions
- app child resources such as `bindings` and `customDomains`
- service child resources such as `certificates`, `apms`, `gateways`, `apiPortals`, `applicationLiveViews`, `configurationServices`, `serviceRegistries`, `devToolPortals`, `containerRegistries`, and `storages`
- build-service families such as `buildServices`, `builds`, `builders`, `buildpackBindings`, and supported stack/buildpack listing routes
- an Azure Go SDK example under `examples/azure/azurespringapps-2023-12-01`
- Azure coverage-script wiring for the new service key

Important scope note:
- The reviewed surface is the ARM management plane for Azure Spring Apps under `Microsoft.AppPlatform`.
- This is not a Spring runtime data plane and not a workload-specific app ingress/data API.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2023-12-01`.

Retirement note from Microsoft Azure product documentation:
- Azure Spring Apps is being retired in March 2028.
- That does not change the implementation target here, but it does mean future example selection should avoid over-investing in fringe tails before the core management surface is green.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/operation-groups?view=rest-azurespringapps-2023-12-01`

Representative operation-group and linked method pages reviewed across the surface:
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/operations/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/enable-test-endpoint?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/disable-test-endpoint?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/disable-apm-globally?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/services/stop?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/apps/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/apps/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/apps/update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/apps/set-active-deployments?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/deployments/update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/deployments/start?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/bindings/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/bindings/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/bindings/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/custom-domains/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/custom-domains/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/custom-domains/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/custom-domains/update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/certificates/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/certificates/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/certificates/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/apms/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/apms/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/gateways/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/gateways/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/api-portals/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/api-portals/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/application-live-views/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/application-live-views/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/configuration-services/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/configuration-services/validate?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/service-registries/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/service-registries/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/dev-tool-portals/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/dev-tool-portals/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/dev-tool-portals/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/container-registries/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/container-registries/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/container-registries/list?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/container-registries/validate?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/storages/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/storages/get?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/create-or-update?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/get-build-service?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/create-or-update-build?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/list-builds?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/list-build-results?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/list-supported-buildpacks?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/build-service/list-supported-stacks?view=rest-azurespringapps-2023-12-01`
- `https://learn.microsoft.com/en-us/rest/api/azurespringapps/buildpack-binding/list?view=rest-azurespringapps-2023-12-01`

Azure Go SDK package expected for the future example:
- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appplatform/armappplatform/v2`

SDK selection note:
- The official `armappplatform/v2` package exists and exposes the major client families needed for the reviewed surface, including `ServicesClient`, `AppsClient`, `BindingsClient`, `CertificatesClient`, `CustomDomainsClient`, `DeploymentsClient`, `GatewaysClient`, `APMsClient`, `ConfigurationServicesClient`, `ServiceRegistriesClient`, `BuildServiceClient`, and related build/buildpack clients.
- The package is marked deprecated because the service itself is retired on a fixed timeline, but it remains the correct official SDK for this API surface.
- Plan assumption: prefer typed `armappplatform/v2` clients for the example and only fall back to raw ARM requests if a reviewed method is missing from the generated client set.

## Service Naming

Canonical Stackyard naming:
- service key: `springapps`
- display name: `Azure Spring Apps`
- provider namespace: `Microsoft.AppPlatform`
- API version: `2023-12-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_springapps.go`
- tests: `internal/server/provider_azure_springapps_test.go`
- example: `examples/azure/azurespringapps-2023-12-01`
- plan doc: `docs/azurespringapps-2023-12-01-plan.md`

Recommended coverage aliases:
- `springapps`
- `spring_apps`
- `spring-apps`
- `azure_spring_apps`
- `azure-spring-apps`
- `azurespringapps`
- `appplatform`
- `app_platform`
- `microsoft_appplatform`
- `rest_azurespringapps_2023_12_01`
- `azurespringapps_2023_12_01`

## Surface Summary

The reviewed `2023-12-01` management surface is broad and nested. The visible operation groups and linked methods indicate at least these implementation families:

1. `Operations`
2. `Services`
3. `Apps`
4. `Deployments`
5. `Bindings`
6. `Custom Domains`
7. `Certificates`
8. `APMs`
9. `Gateways`
10. `Api Portals`
11. `Application Live Views`
12. `Configuration Services`
13. `Service Registries`
14. `Dev Tool Portals`
15. `Container Registries`
16. `Storages`
17. `Build Service`
18. `Buildpack Binding`

This is too large for one flat implementation slice. The emulator should be staged by operator usefulness and nesting depth:

- provider and service roots first
- app lifecycle and deployment flow second
- common app/service children third
- build-service branches after that
- lower-value tails only once the core routes are stable in coverage

## Route Envelope

Documented route family:
- all reviewed routes are ARM-style under the Azure Resource Manager host
- every reviewed route requires `api-version=2023-12-01`
- all routes sit under the `Microsoft.AppPlatform` provider namespace
- the main root resource path remains:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}`

Representative Stackyard route targets under `/azure`:
- `GET /azure/providers/Microsoft.AppPlatform/operations?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}?api-version=2023-12-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/enableTestEndpoint?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/apps/{appName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/apps/{appName}/deployments/{deploymentName}?api-version=2023-12-01`
- `POST /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/apps/{appName}/setActiveDeployments?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/apps/{appName}/bindings/{bindingName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/apps/{appName}/domains/{domainName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/certificates/{certificateName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/gateways/{gatewayName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/apiPortals/{apiPortalName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/buildServices/{buildServiceName}?api-version=2023-12-01`
- `PUT /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/buildServices/{buildServiceName}/builds/{buildName}?api-version=2023-12-01`
- `GET /azure/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.AppPlatform/Spring/{serviceName}/buildServices/{buildServiceName}/builders/{builderName}/buildpackBindings?api-version=2023-12-01`

Important routing implications from the reviewed pages:

- most resources are nested below a single service root and should share parent existence checks
- several service-level features are default-singleton style resources:
  - `gateways/{gatewayName}`
  - `apiPortals/{apiPortalName}`
  - `applicationLiveViews/{applicationLiveViewName}`
  - `configurationServices/{configurationServiceName}`
  - `serviceRegistries/{serviceRegistryName}`
  - `devToolPortals/{devToolPortalName}`
- the build-service hierarchy is deeper and should share one canonical build-service store with nested builders, builds, results, and buildpack bindings
- service and deployment action endpoints use `POST` and should be modeled separately from tracked-resource CRUD
- many create/update operations are long-running in Azure, but coverage-first emulation can usually answer synchronously with stable succeeded payloads unless a specific async contract is required by the SDK path

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use `CloudError` / `CloudErrorBody`
- list routes return `value`
- ARM resources return standard fields such as:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.*`

### Operations

`Operations.List`
- route:
  - `/providers/Microsoft.AppPlatform/operations`
- returns available provider operations for the full Spring Apps surface

Modeling implication:
- expose a deterministic `value` list covering each resource family implemented in the emulator, not just the root service CRUD verbs

### Services

Reviewed method pages:
- `Create Or Update`
- `Get`
- `List`
- `Enable Test Endpoint`
- `Disable Test Endpoint`
- `Disable Apm Globally`
- `Stop`

Key traits from the reviewed pages:
- service resource path:
  - `/providers/Microsoft.AppPlatform/Spring/{serviceName}`
- service properties include network, outbound IP, power state, provisioning state, fqdn, and version fields
- service-scoped actions mutate flags or power state without changing the resource path

Modeling implications:
- persist canonical service state keyed by ARM ID
- implement `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-resource-group, and list-by-subscription
- represent action endpoints as property transitions on the same stored service:
  - test endpoint enabled/disabled
  - global APM disabled
  - power state stopped/running

### Apps and Deployments

Reviewed method pages:
- `Apps.Get`
- `Apps.List`
- `Apps.Update`
- `Apps.SetActiveDeployments`
- `Deployments.Update`
- `Deployments.Start`

Key traits from the reviewed pages:
- apps are nested under services
- app payloads can include ingress, TLS, disk, identity, and public exposure settings
- deployments are nested under apps
- deployment action routes start/stop/promote active deployment behavior

Modeling implications:
- make apps the core workload store under a service
- keep one active deployment pointer per app
- model deployments as child resources with lifecycle/provisioning state and a simple instance summary
- implement enough of app/deployment actions for coverage to validate:
  - create or update app
  - create or update deployment
  - set active deployment
  - start deployment

### Bindings and Custom Domains

Reviewed method pages:
- `Bindings.Create Or Update`
- `Bindings.Get`
- `Bindings.List`
- `Custom Domains.Create Or Update`
- `Custom Domains.Get`
- `Custom Domains.List`
- `Custom Domains.Update`

Key traits from the reviewed pages:
- both are nested under apps
- custom domains return certificate and thumbprint-related properties
- bindings include source resource references and generated connection-like output

Modeling implications:
- store bindings and custom domains as per-app child maps
- keep generated binding output deterministic and non-secret
- model custom-domain provisioning state and certificate reference fields explicitly because SDK examples commonly read them back

### Service Child Resources

Reviewed method pages:
- `Certificates.*`
- `Apms.*`
- `Gateways.*`
- `Api Portals.*`
- `Application Live Views.*`
- `Configuration Services.Get / Validate`
- `Service Registries.Get / List`
- `Dev Tool Portals.*`
- `Container Registries.*`
- `Storages.*`

Key traits from the reviewed pages:
- these resources are mostly service-scoped children
- several of them have `validate` action routes
- list/get/create-or-update patterns recur with small shape variations

Modeling implications:
- implement one generic service-child storage layer keyed by:
  - service ID
  - child collection name
  - child resource name
- start with a narrow successful shape for each family and reuse ARM helper envelopes rather than hand-writing each from scratch
- validation routes should return deterministic success payloads instead of deep semantic checks

### Build Service and Buildpack Families

Reviewed method pages:
- `Build Service.Create Or Update`
- `Build Service.Get Build Service`
- `Build Service.Create Or Update Build`
- `Build Service.List Builds`
- `Build Service.List Build Results`
- `Build Service.List Supported Buildpacks`
- `Build Service.List Supported Stacks`
- `Buildpack Binding.List`

Key traits from the reviewed pages:
- build resources are nested:
  - `buildServices/{buildServiceName}`
  - `builds/{buildName}`
  - `builders/{builderName}`
  - `buildpackBindings`
- the service exposes read-only capability/list routes in addition to tracked resources

Modeling implications:
- canonicalize build-service state first, then nest builds/build results/builders under it
- keep supported stacks/buildpacks as static deterministic fixtures
- implement buildpack bindings as a read-mostly child collection initially unless coverage requires full CRUD

## Recommended Implementation Stages

### Stage 0: Router and Coverage Wiring

Add the service skeleton before filling in resources:

- add a new provider file:
  - `internal/server/provider_azure_springapps.go`
- route `Microsoft.AppPlatform` Spring Apps traffic in:
  - `internal/server/http.go`
- add service state maps to `Server`
- update Azure coverage aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- reserve service key:
  - `springapps`

Deliverable:
- unknown Spring Apps routes no longer fall through to global `NotImplemented`
- provider `operations` can be exercised first

### Stage 1: Provider and Core Services

Implement:

- `GET /providers/Microsoft.AppPlatform/operations`
- `PUT/GET/HEAD/PATCH/DELETE` for root services
- list-by-resource-group and list-by-subscription for services
- basic service actions:
  - `enableTestEndpoint`
  - `disableTestEndpoint`
  - `disableApmGlobally`
  - `stop`

Data model:
- one canonical service object keyed by ARM ID
- service flags for test endpoint, apm state, and power state

Tests:
- CRUD and list coverage
- action endpoints mutate the stored service predictably

### Stage 2: Apps and Deployments

Implement:

- app CRUD/list/get/head
- deployment CRUD/list/get/head
- app action:
  - `setActiveDeployments`
- deployment actions:
  - `start`
  - `stop` if coverage requires it

Data model:
- per-service app map
- per-app deployment map with `activeDeploymentName`

Tests:
- app lifecycle
- deployment lifecycle
- active deployment switching

### Stage 3: Common App and Service Children

Implement the resource families most likely to appear in SDK examples and coverage first:

- `bindings`
- `customDomains`
- `certificates`
- `apms`
- `containerRegistries`
- `storages`

Also implement validation routes where reviewed:

- container registry `validate`

Goal:
- cover the common "attach dependency / ingress / cert / registry" management loop before platform-observability tails

### Stage 4: Platform Add-Ons

Implement singleton or near-singleton service children:

- `gateways`
- `apiPortals`
- `applicationLiveViews`
- `configurationServices`
- `serviceRegistries`
- `devToolPortals`

Recommended scope:
- `PUT/GET/LIST`
- `PATCH/DELETE` where documented and coverage demands it
- `validate` for configuration services

### Stage 5: Build Service Branch

Implement:

- `buildServices`
- nested `builds`
- `listBuildResults`
- `listSupportedBuildpacks`
- `listSupportedStacks`
- builders and `buildpackBindings` list/get if exposed by coverage

Recommended modeling:
- keep build execution deterministic and immediate
- represent build results as stable generated artifacts instead of simulating real build pipelines

### Stage 6: Example and Coverage Completion

Add the example:

- `examples/azure/azurespringapps-2023-12-01`

Recommended example flow:

- create or get a Spring Apps service
- create an app
- create a deployment
- set active deployment
- create one certificate or container-registry child resource
- fetch one gateway or service-registry style singleton

Update coverage scripts:

- add service aliases listed above
- map the service to this plan doc in doc coverage
- ensure endpoint coverage points the service name at the Spring Apps provider namespace

## Example Plan

Recommended future example files:

- `examples/azure/azurespringapps-2023-12-01/main.go`
- `examples/azure/azurespringapps-2023-12-01/Dockerfile`
- `examples/azure/azurespringapps-2023-12-01/docker-compose.yml`

Recommended SDK usage:

- use `armappplatform/v2`
- prefer typed clients such as:
  - `NewServicesClient`
  - `NewAppsClient`
  - `NewDeploymentsClient`
  - `NewCertificatesClient`
  - `NewGatewaysClient`
  - `NewBuildServiceClient`

Example should demonstrate:

- ARM endpoint override to Stackyard
- one service lifecycle
- one app/deployment workflow
- one child-resource workflow

Avoid in the first example:

- deep build-service builder hierarchies
- every singleton feature
- niche action routes with little contract value

## Coverage Script Wiring

During implementation, update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Recommended mappings:

- canonical service: `springapps`
- aliases:
  - `spring_apps`
  - `spring-apps`
  - `azure_spring_apps`
  - `azurespringapps`
  - `appplatform`

Documentation mapping:

- `springapps` -> `azurespringapps-2023-12-01-plan.md`

## Validation Plan

When implementation starts, validate in this order:

1. `go test ./internal/server -run 'TestAzureSpringApps' -count=1`
2. `go build ./examples/azure/azurespringapps-2023-12-01`
3. `python3 scripts/azure-contract-coverage.py --service springapps --require-service springapps --fail-on any`
4. `python3 scripts/azure-io-contract-coverage.py --service springapps --require-service springapps --fail-on any`
5. `python3 scripts/azure-doc-contract-coverage.py --service springapps --require-service springapps --fail-on any`
6. `python3 scripts/azure-endpoint-coverage.py --service springapps --require-service springapps --fail-on any`

Success criteria:

- every Spring Apps endpoint covered by the selected stage returns success or the expected modeled validation error
- no reviewed endpoint falls through to `NotImplemented`
- the example builds cleanly against the chosen SDK package

## Risks and Open Questions

- The surface is large and includes several singleton-style add-on resources that are easy to overfit. The staged plan keeps them behind the core service/app/deployment loop.
- Some Microsoft Learn pages for this surface require sign-in to show the full body, but the route signatures, method names, parameters, and key response shapes are still exposed in the reviewed pages and are sufficient for planning.
- The service retirement means the SDK and docs may see less future churn, but it also means example and coverage work should focus on breadth of supported routes rather than deep simulation fidelity for every fringe add-on.
