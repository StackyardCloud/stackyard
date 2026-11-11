# Azure Data Manager for Agriculture Data Plane (`data-manager-for-agri-dataplane-2023-11-01-preview`) Staged Plan

## Objective

Emulate the Azure Data Manager for Agriculture data-plane API for `2023-11-01-preview` with deterministic local behavior for:

- core farm-management records such as `parties`, `farms`, `fields`, `seasonalFields`, `seasons`, `crops`, and `cropProducts`
- farm-operations records such as `application-data`, `tillage-data`, `planting-data`, `harvest-data`, and bulk `farm-operations`
- sensor-family entities such as `device-data-models`, `devices`, `sensor-data-models`, `sensors`, `sensor-mappings`, `sensor-partner-integrations`, and `sensor-events`
- observation and recommendation entities such as `prescriptions`, `prescription-maps`, `plant-tissue-analyses`, `nutrient-analyses`, `insights`, and `insight-attachments`
- dataset families such as `datasets` and `dataset-records`, including access grant/list flows
- provider-style action APIs such as `weather-data:fetch`, `scenes`, `image-processing`, `oauth/tokens/:connect`, and `solutions/{solutionId}:create`
- an Azure Go SDK-based example under `examples/azure/data-manager-for-agri-dataplane-2023-11-01-preview`
- Azure coverage-script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-data-manager-for-agri-dataplane-2023-11-01-preview` is not the ARM `Microsoft.AgFoodPlatform` management plane already implemented in this repo.
- It is a true data-plane API rooted at `{data-manager-for-agri-resource-name}.farmbeats.azure.net`.
- This plan intentionally treats it as a separate Stackyard service from the control-plane `data_manager_for_agri` provider.

## Primary References Reviewed

Primary operation-groups page reviewed:
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/operation-groups?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview

Primary service overview reviewed:
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/

Operation-group pages reviewed:
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/parties?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/farms?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/fields?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/scenes?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/weather-data?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/attachments?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/insights?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview

Representative linked method pages reviewed across the major route families:
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/parties/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/farms/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/farms/list?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/fields/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/fields/search?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/devices/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/devices/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/sensors/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/attachments/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/attachments/download?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/insights/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/insights/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/insight-attachments/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/insight-attachments/download?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/prescriptions/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/prescriptions/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/prescriptions/search?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/datasets/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/datasets/list?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/datasets/authorize-access?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/datasets/list-authorized-access?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/dataset-records/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/dataset-records/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/dataset-records/list?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/weather-data/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/o-auth-providers/get?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/o-auth-tokens/get-o-auth-connection-link?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/o-auth-tokens/list?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/solution-inference/create-or-update?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview
- https://learn.microsoft.com/en-us/rest/api/data-manager-for-agri/dataplane/solution-inference/fetch?view=rest-data-manager-for-agri-dataplane-2023-11-01-preview

SDK selection note:
- I did not find a service-specific official Azure Go SDK package for this full `2023-11-01-preview` data-plane surface.
- Plan assumption: the example should use the official Azure Go SDK core runtime pipeline (`azcore/runtime` and `azcore/policy`) rather than inventing a non-existent typed client.

## Service Naming

Canonical Stackyard naming:
- service key: `data_manager_for_agri_data_plane`
- display name: `Azure Data Manager for Agriculture Data Plane`
- API version token: `2023-11-01-preview`

Recommended repo paths:
- implementation: `internal/server/provider_azure_data_manager_for_agri_data_plane.go`
- tests: `internal/server/provider_azure_data_manager_for_agri_data_plane_test.go`
- example: `examples/azure/data-manager-for-agri-dataplane-2023-11-01-preview`
- plan doc: `docs/data-manager-for-agri-dataplane-2023-11-01-preview-plan.md`

Recommended coverage aliases:
- `data_manager_for_agri_data_plane`
- `data_manager_for_agri_dataplane`
- `data-manager-for-agri-dataplane`
- `agrifood_data_plane`
- `farmbeats_data_plane`
- `rest_data_manager_for_agri_dataplane_2023_11_01_preview`
- `data_manager_for_agri_dataplane_2023_11_01_preview`

## Surface Summary

The reviewed service overview exposes 38 operation groups in `2023-11-01-preview`, grouped into these families:

1. Farm management: `parties`, `farms`, `fields`, `seasonalFields`, `seasons`, `crops`, `cropProducts`, `attachments`, `zones`, `managementZones`, `chemicalProducts`, `farmEquipments`
2. Satellite and imagery: `scenes`, `image-processing`
3. Sensor and partner integrations: `device-data-models`, `devices`, `sensor-data-models`, `sensors`, `sensor-mappings`, `sensor-partner-integrations`, `sensor-events`
4. Weather: `weather-data`
5. Farm operations and connectivity: `application-data`, `tillage-data`, `planting-data`, `harvest-data`, `farm-operations`, `oauth-providers`, `oauth-tokens`
6. Insights and observations: `prescriptions`, `prescription-maps`, `plant-tissue-analyses`, `nutrient-analyses`, `insights`, `insight-attachments`
7. Solution inference: `solution-inference`
8. Reference datasets: `datasets`, `dataset-records`

Important route-shape observations from the reviewed pages:

- This is a data-plane API rooted at `{data-manager-for-agri-resource-name}.farmbeats.azure.net`.
- Every route requires `api-version=2023-11-01-preview`.
- Authentication is bearer-token style on every reviewed method page.
- The API mixes classic entity CRUD routes, global search routes, upload/download binary routes, and action-style verbs such as `:search`, `:fetch`, `:connect`, `:grant`, and `:create`.
- Many farm-management entities use nested party-scoped routes such as `/parties/{partyId}/farms/{farmId}`.
- Some searches are global and not party-scoped, for example `/farms`, `/fields:search`, `/datasets`, or `/oauth/tokens`.
- Sensor entities are rooted under `/sensor-partners/{sensorPartnerId}/...`.
- Insight resources are nested under model/resource context:
  `/parties/{partyId}/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/...`
- Several families support paginated list responses with `value`, `skipToken`, and `nextLink`.
- Multiple groups expose binary payloads or multipart uploads, especially attachments, insight attachments, scene downloads, and image-processing flows.

## Route Envelope

Documented route families reviewed:

- `GET {endpoint}/parties/{partyId}?api-version=2023-11-01-preview`
- `GET {endpoint}/parties/{partyId}/farms/{farmId}?api-version=2023-11-01-preview`
- `GET {endpoint}/farms?api-version=2023-11-01-preview`
- `GET {endpoint}/parties/{partyId}/fields/{fieldId}?api-version=2023-11-01-preview`
- `POST {endpoint}/fields:search?api-version=2023-11-01-preview`
- `PATCH {endpoint}/sensor-partners/{sensorPartnerId}/devices/{deviceId}?api-version=2023-11-01-preview`
- `PATCH {endpoint}/sensor-partners/{sensorPartnerId}/sensors/{sensorId}?api-version=2023-11-01-preview`
- `PATCH {endpoint}/parties/{partyId}/attachments/{attachmentId}?api-version=2023-11-01-preview`
- `GET {endpoint}/parties/{partyId}/attachments/{attachmentId}/file?api-version=2023-11-01-preview`
- `PATCH {endpoint}/parties/{partyId}/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/insights/{insightId}?api-version=2023-11-01-preview`
- `PATCH {endpoint}/parties/{partyId}/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/insight-attachments/{insightAttachmentId}?api-version=2023-11-01-preview`
- `PATCH {endpoint}/parties/{partyId}/prescriptions/{prescriptionId}?api-version=2023-11-01-preview`
- `POST {endpoint}/prescription:search?api-version=2023-11-01-preview`
- `PATCH {endpoint}/datasets/{datasetId}?api-version=2023-11-01-preview`
- `PATCH {endpoint}/datasets/{datasetId}/records/{recordId}?api-version=2023-11-01-preview`
- `POST {endpoint}/datasets/{datasetId}/access:grant?api-version=2023-11-01-preview`
- `POST {endpoint}/weather-data:fetch?api-version=2023-11-01-preview`
- `POST {endpoint}/oauth/tokens/:connect?api-version=2023-11-01-preview`
- `POST {endpoint}/solutions/{solutionId}:create?api-version=2023-11-01-preview`
- `POST {endpoint}/solutions/{solutionId}:fetch?api-version=2023-11-01-preview`

Recommended Stackyard emulation prefix:
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/...`

Recommended route families:
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/parties`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/farms`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/fields`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/seasonalFields`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/seasons`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/crops`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/cropProducts`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/attachments`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/zones`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/managementZones`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/chemicalProducts`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/farmEquipments`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/scenes`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/sensor-partners`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/weather-data:fetch`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/application-data`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/tillage-data`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/planting-data`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/harvest-data`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/farm-operations`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/oauth/providers`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/oauth/tokens`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/prescriptions`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/prescription-maps`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/plant-tissue-analyses`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/nutrient-analyses`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/insights`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/insight-attachments`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/solutions`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/datasets`
- `/azure/data-manager-for-agri/dataplane/2023-11-01-preview/datasets/{datasetId}/records`

Routing note:
- This API does not use the ARM `subscriptions/.../providers/...` envelope.
- It should remain distinct from the control-plane Agriculture provider.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:
- Security is bearer-token based via the `Authorization` header.
- Errors use `ErrorResponse` with `x-ms-error-code`.
- Many entity create/update methods are `PATCH` with `application/merge-patch+json`.
- Upload flows use `multipart/form-data` or file form fields.
- Download flows may return either `application/json` or `application/octet-stream`.
- Search/list responses are generally paginated and return `value`, `skipToken`, and sometimes `nextLink`.

### Farm Management Records

Reviewed method pages show stable entity patterns for `parties`, `farms`, and `fields`:

- `GET /parties/{partyId}`
- `GET /parties/{partyId}/farms/{farmId}`
- `GET /farms`
- `GET /parties/{partyId}/fields/{fieldId}`
- `POST /fields:search`

Common payload traits:
- `id`
- `partyId`
- `name`
- `status`
- `description`
- `source`
- `properties`
- `createdDateTime`
- `modifiedDateTime`
- `createdBy`
- `modifiedBy`
- `eTag`

Modeling implication:
- implement a reusable record store for these entity families
- share the same metadata and paging helpers across the farm-management families
- add a geometry-aware but deterministic stub for `:search` endpoints

### Sensor Family

Reviewed method pages show sensor resources are rooted under a sensor partner:

- `PATCH /sensor-partners/{sensorPartnerId}/devices/{deviceId}`
- `GET /sensor-partners/{sensorPartnerId}/devices/{deviceId}`
- `PATCH /sensor-partners/{sensorPartnerId}/sensors/{sensorId}`

Modeling implication:
- use one hierarchical store keyed by `sensorPartnerId` then collection then resource id
- support the same CRUD/list shape across `devices`, `sensors`, and the model/mapping families
- treat `sensor-events` as append-only ingress/event egress rather than another CRUD collection

### Attachments and Binary Transfers

Reviewed method pages show:
- attachment upload uses form fields and file data
- attachment download returns binary or JSON content
- insight attachments follow the same pattern under the insight route family

Modeling implication:
- add shared binary-object storage for attachments and insight attachments
- keep metadata objects separate from file content
- return stable download bytes for coverage and example validation

### Insights and Observations

Reviewed method pages show nested context-heavy routes:

- `PATCH /parties/{partyId}/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/insights/{insightId}`
- `GET /parties/{partyId}/models/{modelId}/resource-types/{resourceType}/resources/{resourceId}/insights/{insightId}`
- `PATCH /parties/{partyId}/prescriptions/{prescriptionId}`
- `GET /parties/{partyId}/prescriptions/{prescriptionId}`
- `POST /prescription:search`

Important fields surfaced:
- model/resource linkage IDs
- `measurements`
- date windows such as `insightStartDateTime` and `insightEndDateTime`
- attachment links

Modeling implication:
- keep insights and insight attachments in a dedicated nested store keyed by party/model/resource/resource-type
- reuse the same record helpers for prescriptions, maps, and analyses, but preserve nested route identity where the docs require it

### Datasets and Dataset Records

Reviewed method pages show:
- `PATCH /datasets/{datasetId}`
- `GET /datasets`
- `PATCH /datasets/{datasetId}/records/{recordId}`
- `GET /datasets/{datasetId}/records`
- `POST /datasets/{datasetId}/access:grant`
- `GET /datasets/{datasetId}/access`

Important fields surfaced:
- `type`
- `recordsSchema`
- `record`
- dataset access grants via `subscriptionId` and `resourceId`

Modeling implication:
- datasets need a top-level store plus nested records plus nested access grants
- preserve the schema-vs-record split so the example can create a dataset and a record against it
- access grants should be deterministic child resources rather than opaque side effects

### Weather, Scenes, OAuth, and Solution Actions

Reviewed action-style pages show:
- `POST /weather-data:fetch`
- `GET /oauth/tokens`
- `POST /oauth/tokens/:connect`
- `GET /oauth/providers/{oauthProviderId}`
- `POST /solutions/{solutionId}:create`
- `POST /solutions/{solutionId}:fetch`
- `scenes` exposes ingestion-job create/get and list/download patterns

Modeling implication:
- represent these as deterministic request/response actions rather than durable ARM-like resources unless the docs expose a real resource document
- weather fetch should return cached synthetic weather payloads
- OAuth connect should return a deterministic provider redirect URL
- solution inference should persist a small job record so `:fetch` can resolve the previously created trigger
- scenes and image-processing should follow the same job-plus-download pattern

## Azure Go SDK Notes

I did not identify a service-specific official typed Azure Go SDK package for this full preview data-plane surface.

Example implication:
- use the official Azure Go SDK core runtime packages:
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime`
  - `github.com/Azure/azure-sdk-for-go/sdk/azcore/policy`
- build a thin example client around those packages rather than inventing a generated client that does not exist

## Planned Repo Shape

Implementation:
- add a dedicated provider file `internal/server/provider_azure_data_manager_for_agri_data_plane.go`
- register the provider from the main Azure route dispatcher in `internal/server/http.go`
- add dedicated in-memory stores in `Server` for:
  - farm-management records
  - sensor-partner records
  - attachment blobs
  - insight records
  - datasets, records, and access grants
  - job/action results for scenes, image processing, and solution inference

Tests:
- add focused route and shape coverage in `internal/server/provider_azure_data_manager_for_agri_data_plane_test.go`
- include CRUD/list/search coverage across at least one family from each major stage
- include binary upload/download tests
- include action/job polling tests

Example:
- add `examples/azure/data-manager-for-agri-dataplane-2023-11-01-preview/main.go`
- prefer an example that exercises:
  - one CRUD record family
  - one search/list family
  - one dataset family
  - one action-style method such as weather fetch or OAuth connect

Coverage wiring to add during implementation:
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

## Staged Implementation Plan

### Stage 0: Service Claiming and Coverage Wiring

Goal:
- reserve a separate data-plane service key and stop the route family from falling through to generic Azure `NotImplemented`

Scope:
- add service aliases in all Azure coverage scripts
- add plan-doc mapping in the doc coverage script
- add a provider dispatch stub in `http.go`

Exit criteria:
- coverage can target `data_manager_for_agri_data_plane`
- data-plane Agriculture routes are claimed by the new provider

### Stage 1: Shared Data-Plane Envelope and Record Primitives

Goal:
- land the provider skeleton and the shared helpers used by most entity groups

Scope:
- shared route parsing for the data-plane prefix
- bearer-auth header acceptance consistent with other Azure data-plane shims
- reusable record metadata helpers for `id`, `name`, `status`, timestamps, properties, and `eTag`
- common paged list response helpers

Exit criteria:
- the provider can serve trivial health/discovery routes with correct data-plane semantics

### Stage 2: Core Farm Management

Goal:
- cover the most common entity families first

Scope:
- `parties`
- `farms`
- `fields`
- `seasonalFields`
- `seasons`
- `crops`
- `cropProducts`

Implementation notes:
- use one common store shape with per-family route adapters
- include `list` and `search` variants where documented
- support cascade-delete-job placeholders only after the basic CRUD/list/search routes exist

Exit criteria:
- core farm-management endpoints are coverage-complete with deterministic success payloads

### Stage 3: Spatial and Asset Sidecars

Goal:
- finish the farm-management families that add geospatial or asset behavior

Scope:
- `attachments`
- `zones`
- `managementZones`
- `chemicalProducts`
- `farmEquipments`

Implementation notes:
- attachments need metadata plus binary content
- zone families should preserve geometry fields but can use deterministic stub polygons initially

Exit criteria:
- binary upload/download and geometry-bearing resource shapes are stable

### Stage 4: Sensor and Partner Integrations

Goal:
- cover the nested sensor-partner hierarchy

Scope:
- `device-data-models`
- `devices`
- `sensor-data-models`
- `sensors`
- `sensor-mappings`
- `sensor-partner-integrations`
- `sensor-events`

Implementation notes:
- share a common nested store under `sensorPartnerId`
- treat `sensor-events` as ingress/egress documents rather than full CRUD if the docs justify that simplification

Exit criteria:
- sensor family coverage no longer falls through to `NotImplemented`

### Stage 5: Farm Operations and Connectivity

Goal:
- implement operational record types and external-connectivity helpers

Scope:
- `application-data`
- `tillage-data`
- `planting-data`
- `harvest-data`
- `farm-operations`
- `oauth-providers`
- `oauth-tokens`

Implementation notes:
- reuse the record-engine built in earlier stages
- OAuth flows should return deterministic connection URLs and token metadata rather than real authorization

Exit criteria:
- farm operation record flows and OAuth helper routes are stable

### Stage 6: Insights, Recommendations, and Analyses

Goal:
- cover the nested insight/observation route family

Scope:
- `prescriptions`
- `prescription-maps`
- `plant-tissue-analyses`
- `nutrient-analyses`
- `insights`
- `insight-attachments`

Implementation notes:
- preserve the nested model/resource context in the storage key
- insight attachments should reuse the attachment blob helper from Stage 3

Exit criteria:
- the nested insight route family returns correct modeled payloads and file content

### Stage 7: Reference Datasets

Goal:
- cover the largest new `2023-11-01-preview` surface addition

Scope:
- `datasets`
- `dataset-records`
- dataset access grant/list flows

Implementation notes:
- datasets need durable schemas
- records should validate only lightly against the stored schema in the first pass
- access grants should be explicit child objects

Exit criteria:
- dataset and record coverage is green, including access-control routes

### Stage 8: Weather, Satellite, and Image Jobs

Goal:
- implement the action-oriented data retrieval and processing endpoints

Scope:
- `weather-data`
- `scenes`
- `image-processing`

Implementation notes:
- weather fetch returns deterministic synthetic provider data
- scene ingestion and image processing should persist lightweight job documents and expose list/get/download behaviors

Exit criteria:
- action/job flows are usable in examples and no longer return placeholders

### Stage 9: Solution Inference

Goal:
- finish the third-party solution trigger/fetch surface

Scope:
- `solution-inference`

Implementation notes:
- `:create` should mint a stable job trigger payload
- `:fetch` should resolve previously recorded trigger state

Exit criteria:
- solution inference is deterministic and example-ready

### Stage 10: Example and End-to-End Validation

Goal:
- prove the data-plane shim is usable and fully wired into the repo’s coverage gates

Scope:
- add the example under `examples/azure/data-manager-for-agri-dataplane-2023-11-01-preview`
- use `azcore/runtime` plus a thin custom client

Validation target after implementation:
- `go test ./internal/server -run 'TestAzureDataManagerForAgriDataPlane' -count=1`
- `go build ./examples/azure/data-manager-for-agri-dataplane-2023-11-01-preview`
- `python3 scripts/azure-contract-coverage.py --service data_manager_for_agri_data_plane --require-service data_manager_for_agri_data_plane --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service data_manager_for_agri_data_plane --require-service data_manager_for_agri_data_plane --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service data_manager_for_agri_data_plane --require-service data_manager_for_agri_data_plane --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service data_manager_for_agri_data_plane --require-service data_manager_for_agri_data_plane --fail-on any`

## Non-Goals For First Implementation

To keep the first merge tractable, do not attempt all of the following at once:

- real geospatial intersection logic for every `:search` route
- real OAuth handshakes or third-party provider integration
- actual satellite processing or image rasterization
- full schema validation for every dataset record
- complete cascade-delete-job semantics for every family before the underlying CRUD/search paths exist

## Recommended First Merge Slice

The best first merge is:

1. Stage 0
2. Stage 1
3. Stage 2 for `parties`, `farms`, and `fields`
4. Stage 7 for `datasets` and `dataset-records`
5. one action route, preferably `weather-data:fetch`

That sequence claims the service, covers the most reused data-plane patterns, and proves both entity-style and action-style routes before taking on the much larger sensor and insight families.
