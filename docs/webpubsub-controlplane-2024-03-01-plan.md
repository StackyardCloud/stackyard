# Azure Web PubSub Service (`webpubsub-controlplane-2024-03-01`) Staged Plan

## Objective

Emulate the Azure Web PubSub Service ARM management API for `2024-03-01` with deterministic local behavior for:

- provider `operations`
- location-scoped `usages`
- top-level `webPubSub` lifecycle and account-scoped utility routes
- account-scoped `customCertificates`
- account-scoped `customDomains`
- account-scoped `hubs`
- account-scoped `privateEndpointConnections`
- account-scoped `privateLinkResources`
- account-scoped `sharedPrivateLinkResources`
- account-scoped `replicas`
- replica-scoped `sharedPrivateLinkResources`
- an Azure Go SDK example under `examples/azure/webpubsub-controlplane-2024-03-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the ARM management plane for Azure Web PubSub under `Microsoft.SignalRService`.
- This is not the Web PubSub client/data plane used for hub connections or message delivery.
- The plan therefore targets ARM-style `management.azure.com` routes with `api-version=2024-03-01`.

Important naming note:

- Even though the provider is `Microsoft.SignalRService`, the reviewed resource family is specifically `webPubSub`, not generic SignalR Service.
- The Stackyard service key should therefore be `webpubsub`, not `signalr`.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/operation-groups?view=rest-webpubsub-controlplane-2024-03-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/operations?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/usages?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-certificates?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-domains?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-hubs?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-private-endpoint-connections?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-private-link-resources?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-shared-private-link-resources?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-replicas?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-replica-shared-private-link-resources?view=rest-webpubsub-controlplane-2024-03-01`

Representative linked method pages reviewed across the surface:

- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/operations/list?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/usages/list?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub/get?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub/update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub/list-keys?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-certificates/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-certificates/get?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-certificates/list?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-domains/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-custom-domains/delete?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-hubs/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-private-endpoint-connections/get?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-private-endpoint-connections/list?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-private-link-resources/list?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-shared-private-link-resources/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-replicas/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-replicas/get?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-replicas/update?view=rest-webpubsub-controlplane-2024-03-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/controlplane/web-pub-sub-replica-shared-private-link-resources/create-or-update?view=rest-webpubsub-controlplane-2024-03-01`

Azure Go SDK package expected for the future example:

- `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/signalr/armsignalr`

SDK selection note:

- The official `armsignalr` package includes Web PubSub-oriented clients and examples for:
  - root resource lifecycle
  - custom certificates
  - custom domains
  - operations
  - private endpoint connections
  - private link resources
  - shared private link resources
  - usages
- I did not confirm full typed support for every `2024-03-01` Web PubSub child family during this review, especially the deeper replica-scoped branches.
- Plan assumption: prefer `armsignalr` for the example where the client exists, and use `azcore/runtime` only as a targeted fallback for any reviewed routes the package does not expose cleanly.

## Service Naming

Canonical Stackyard naming:

- service key: `webpubsub`
- display name: `Azure Web PubSub Service`
- API version token: `2024-03-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_webpubsub.go`
- tests: `internal/server/provider_azure_webpubsub_test.go`
- example: `examples/azure/webpubsub-controlplane-2024-03-01`
- plan doc: `docs/webpubsub-controlplane-2024-03-01-plan.md`

Recommended coverage aliases:

- `webpubsub`
- `web_pubsub`
- `web-pubsub`
- `webpubsub_controlplane`
- `webpubsub_control_plane`
- `azure_webpubsub`
- `microsoft_signalrservice_webpubsub`
- `rest_webpubsub_controlplane_2024_03_01`

## API Surface Summary

The reviewed pages expose three major route shapes:

- provider and subscription utilities:
  - `/providers/Microsoft.SignalRService/operations`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.SignalRService/locations/{location}/usages`
  - `/subscriptions/{subscriptionId}/providers/Microsoft.SignalRService/checkNameAvailability`
- standard ARM tracked resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.SignalRService/webPubSub/{resourceName}`
- nested account and replica child resources:
  - `customCertificates`
  - `customDomains`
  - `hubs`
  - `privateEndpointConnections`
  - `privateLinkResources`
  - `sharedPrivateLinkResources`
  - `replicas`
  - `replicas/{replicaName}/sharedPrivateLinkResources`

Modeling implications:

- the emulator should keep one canonical Web PubSub account store keyed by ARM ID
- most child resources should validate parent account existence before mutation
- replicas should be modeled as proper ARM child resources, not just a property bag on the parent
- private-link-resource routes are read-only inventory, while private-endpoint and shared-private-link resources are mutable child resources

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- failures use ARM-style `ErrorResponse`
- list routes return `value`
- root and child resources return standard ARM fields such as:
  - `id`
  - `name`
  - `type`
  - `location`
  - `tags`
  - `systemData`
  - `properties.*`

### Operations

Reviewed method page:

- `List`

Key traits from the reviewed page:

- route:
  - `/providers/Microsoft.SignalRService/operations`
- returns `OperationList`

Modeling implications:

- expose a deterministic operation list covering the families implemented in the emulator
- include `webPubSub` CRUD, key-management actions, custom certificate/domain/hub CRUD, private-link families, and replica verbs

### Usages

Reviewed method page:

- `List`

Key traits from the reviewed page:

- route:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.SignalRService/locations/{location}/usages`
- returns location-scoped usage quotas

Modeling implications:

- model usages as read-only seeded inventory
- one or two stable quota rows per location are sufficient for coverage-first emulation

### Root `webPubSub` Resource

Reviewed method pages:

- `Check Name Availability`
- `Create Or Update`
- `Delete`
- `Get`
- `List By Resource Group`
- `List By Subscription`
- `List Keys`
- `List Replica Skus`
- `List Skus`
- `Regenerate Key`
- `Update`

Key traits from the reviewed pages:

- root path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.SignalRService/webPubSub/{resourceName}`
- `resourceName` constraints:
  - min length `3`
  - max length `63`
  - pattern `^[a-zA-Z][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]$`
- create requires:
  - `location`
- root properties in examples and definitions include:
  - `hostName`
  - `externalIP`
  - `publicPort`
  - `serverPort`
  - `publicNetworkAccess`
  - `tls`
  - `networkACLs`
  - `liveTraceConfiguration`
  - `resourceStopped`
  - `provisioningState`
- `listKeys` returns access-key metadata
- `regenerateKey` is an action route
- list-sku routes are read-only discovery helpers

Modeling implications:

- persist canonical root resources keyed by ARM ID
- support `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-resource-group, and list-by-subscription
- implement utility actions:
  - `checkNameAvailability`
  - `listKeys`
  - `regenerateKey`
  - `listSkus`
  - `listReplicaSkus`
- synthesize stable connection properties like `hostName` and `workspace-like` identifiers from the account name

### Custom Certificates

Reviewed method pages:

- `Create Or Update`
- `Get`
- `List`
- operation-group index also lists:
  - `Delete`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/customCertificates/{certificateName}`
- create/update request includes certificate-binding information rather than raw cert bytes

Modeling implications:

- store custom certificates keyed by child ARM ID
- support list/get/put/delete
- keep validation pragmatic around presence of a `properties` block and a few high-signal fields

### Custom Domains

Reviewed method pages:

- `Create Or Update`
- `Delete`
- operation-group page indicates:
  - `Get`
  - `List`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/customDomains/{name}`
- create/update associates the domain with a custom certificate reference

Modeling implications:

- store custom domains keyed by child ARM ID
- validate parent certificate references where practical

### Hubs

Reviewed method pages:

- `Create Or Update`
- operation-group page also lists:
  - `Delete`
  - `Get`
  - `List`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/hubs/{hubName}`
- request body requires `properties`
- response codes include `200` and `201`

Modeling implications:

- hubs should be stored as lightweight child ARM resources
- first-pass validation can require only the `properties` object and preserve the rest as submitted

### Private Endpoint Connections

Reviewed method pages:

- `Get`
- `List`
- operation-group page indicates:
  - `Create Or Update`
  - `Delete`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/privateEndpointConnections/{privateEndpointConnectionName}`
- list/get shapes follow standard private-endpoint-connection ARM models

Modeling implications:

- store private endpoint connections keyed by child ARM ID
- support list/get/put/delete
- preserve `privateLinkServiceConnectionState` fields and `privateEndpoint.id`

### Private Link Resources

Reviewed method pages:

- `List`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/privateLinkResources`
- this family is read-only inventory for Private Link setup

Modeling implications:

- implement as deterministic read-only inventory
- one stable group such as `webpubsub` with required DNS-zone data is enough for first-pass coverage

### Shared Private Link Resources

Reviewed method pages:

- `Create Or Update`
- operation-group page indicates:
  - `Delete`
  - `Get`
  - `List`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}`
- this is a mutable child-resource family distinct from private endpoint connections

Modeling implications:

- store SPLRs keyed by child ARM ID
- keep approval-state fields visible in `properties`

### Replicas

Reviewed method pages:

- `Create Or Update`
- `Get`
- `Update`
- operation-group page also lists:
  - `Delete`
  - `List`
  - `Restart`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/replicas/{replicaName}`
- replica name constraints mirror the root resource name constraints
- create requires `location`
- properties include:
  - `regionEndpointEnabled`
  - `resourceStopped`
- `restart` is an action route

Modeling implications:

- model replicas as first-class child resources with their own `location`, `sku`, `tags`, and state
- `restart` should mutate a visible `lastRestartedAt` or transient state field on the stored replica

### Replica Shared Private Link Resources

Reviewed method pages:

- `Create Or Update`
- operation-group page indicates:
  - `Get`
  - `List`

Key traits from the reviewed pages:

- child path:
  - `/webPubSub/{resourceName}/replicas/{replicaName}/sharedPrivateLinkResources/{sharedPrivateLinkResourceName}`

Modeling implications:

- model these separately from root-level shared-private-link resources
- keep them keyed by the replica child path so resource lookup stays unambiguous

## Staged Implementation

### Stage 0: Router and Coverage Wiring

Add the service shell and discovery:

- add `internal/server/provider_azure_webpubsub.go`
- register the new router in `internal/server/http.go`
- add `Server` storage maps for:
  - root resources
  - custom certificates
  - custom domains
  - hubs
  - private endpoint connections
  - shared private link resources
  - replicas
  - replica shared private link resources
- add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Acceptance target:

- service is discoverable by the coverage scripts
- no routed Web PubSub path falls through to `NotImplemented`

### Stage 1: Provider, Usages, and Root Resource

Implement:

- `GET /providers/Microsoft.SignalRService/operations`
- `GET usages`
- root `checkNameAvailability`
- root `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`
- root list-by-resource-group
- root list-by-subscription
- `listKeys`
- `regenerateKey`
- `listSkus`
- `listReplicaSkus`

Modeling notes:

- seed one Web PubSub resource with stable connection properties and key material
- root delete can be answered synchronously unless the chosen SDK example requires an accepted-style poller path

Acceptance target:

- base lifecycle and utility routes work
- tests cover invalid names, name-availability, list/get/head, and key regeneration

### Stage 2: Core Child Resources

Implement:

- `customCertificates` list/get/put/delete
- `customDomains` list/get/put/delete
- `hubs` list/get/put/delete

Modeling notes:

- enforce parent resource existence
- keep certificate/domain/hub schemas broad but stable

Acceptance target:

- core child-resource routes stop returning `NotImplemented`
- tests cover read-after-write and child not-found behavior

### Stage 3: Private Connectivity

Implement:

- `privateEndpointConnections` list/get/put/delete
- `privateLinkResources` list
- `sharedPrivateLinkResources` list/get/put/delete

Modeling notes:

- seed one approved private endpoint connection and one read-only private link resource on the default service
- shared-private-link resources should expose a visible approval-state field

Acceptance target:

- private connectivity families are fully covered
- list/get payloads are stable and typed

### Stage 4: Replicas

Implement:

- `replicas` list/get/put/patch/delete
- `replicas/restart`
- replica-scoped `sharedPrivateLinkResources` list/get/put

Modeling notes:

- keep at least one seeded replica on the default root resource
- `restart` should update observable status on the replica

Acceptance target:

- replica paths no longer fall through to `NotImplemented`
- tests cover nested child behavior beneath a replica

### Stage 5: Example

Add `examples/azure/webpubsub-controlplane-2024-03-01`:

- prefer `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/signalr/armsignalr`
- if some reviewed `2024-03-01` child families are not exposed cleanly in that package, use the typed SDK for the supported portion and raw `azcore/runtime` requests for the tail routes

Suggested example flow:

1. list provider `operations`
2. list location `usages`
3. check name availability
4. create a Web PubSub resource
5. get, patch, and list the resource
6. list and regenerate keys
7. create a hub
8. create a custom certificate and custom domain
9. list private endpoint connections and private link resources
10. create a shared private link resource
11. create and restart a replica

Acceptance target:

- example builds cleanly
- example exercises the main reviewed surface without custom hooks

### Stage 6: Validation and Coverage

Validation after implementation:

- `go test ./internal/server -run 'TestAzureWebPubSub' -count=1`
- `go build ./examples/azure/webpubsub-controlplane-2024-03-01`
- `python3 scripts/azure-contract-coverage.py --service webpubsub --require-service webpubsub --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service webpubsub --require-service webpubsub --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service webpubsub --require-service webpubsub --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service webpubsub --require-service webpubsub --fail-on any`

## Test Plan

Focused provider tests should cover:

- provider route detection and invalid `api-version`
- root create/get/head/list/update/delete lifecycle
- name-availability and invalid-name validation
- usages list
- list/regenerate keys
- child-resource lifecycle for certificates, domains, and hubs
- private-endpoint and shared-private-link child-resource lifecycle
- read-only private-link-resource list
- replica create/get/update/restart/delete lifecycle
- replica shared-private-link child routes
- not-found behavior for missing root resources and missing child resources

## Risks and Follow-Ups

- Several Microsoft Learn pages in this surface are partially authorization-gated, so implementation should stay grounded in the clearly exposed URIs, field tables, and example response shapes.
- The official `armsignalr` package is clearly related and includes Web PubSub-specific clients, but the exact API-version alignment with `2024-03-01` should be checked at implementation time before depending on it for every child family.
- This service is broad enough that the first implementation should land in slices rather than trying to emulate every child route at once.
