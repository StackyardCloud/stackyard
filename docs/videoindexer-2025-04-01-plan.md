# Azure Video Indexer (`videoindexer-2025-04-01`) Staged Plan

## Objective

Emulate the Azure Video Indexer ARM management API exposed by the reviewed `rest-videoindexer-2025-04-01` pages with deterministic local behavior for:

- provider `operations`
- subscription-scoped `checkNameAvailability`
- top-level `accounts` lifecycle
- account-scoped token-generation actions:
  - `generateAccessToken`
  - `generateRestrictedViewerAccessToken`
  - `generateExtensionAccessToken`
  - `generateExtensionRestrictedViewerAccessToken`
- account-scoped `privateEndpointConnections`
- account-scoped `privateLinkResources`
- an Azure Go SDK example under `examples/azure/videoindexer-2025-04-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the Azure ARM management plane for Azure Video Indexer under `Microsoft.VideoIndexer`.
- This is not the legacy Video Indexer data plane rooted at regional account endpoints.
- The plan therefore targets `management.azure.com` routes with `api-version=2025-04-01`.

Important product note:

- Even though token-generation routes return access tokens used against Video Indexer experiences, the reviewed methods are still ARM management-plane `POST` actions under the account resource.
- The Stackyard implementation should therefore keep `videoindexer` separate from any future direct Video Indexer content-processing or playback data-plane emulation.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/videoindexer/operation-groups?view=rest-videoindexer-2025-04-01`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/videoindexer/operations?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/generate?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-endpoint-connections?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-link-resources?view=rest-videoindexer-2025-04-01`

Every linked method page in those reviewed groups was also examined:

- `https://learn.microsoft.com/en-us/rest/api/videoindexer/operations/list?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/check-name-availability?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/create-or-update?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/delete?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/get?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/list?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/list-by-resource-group?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/accounts/update?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/generate/access-token?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/generate/restricted-viewer-access-token?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/generate/extension-access-token?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/generate/extension-restricted-viewer-access-token?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-endpoint-connections/create-or-update?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-endpoint-connections/delete?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-endpoint-connections/get?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-endpoint-connections/list-by-account?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-link-resources/get?view=rest-videoindexer-2025-04-01`
- `https://learn.microsoft.com/en-us/rest/api/videoindexer/private-link-resources/list-by-account?view=rest-videoindexer-2025-04-01`

SDK selection note:

- During review I did not find a modern official Go package that is clearly targeted at `Microsoft.VideoIndexer` management for `2025-04-01`.
- Search results surfaced Azure Video Analyzer packages instead, which are a different service family.
- Plan assumption: the example should use the official Azure Go SDK core runtime pipeline (`azcore/runtime` and `azcore/policy`) rather than inventing a non-existent typed `armvideoindexer` client.

## Service Naming

Canonical Stackyard naming:

- service key: `videoindexer`
- display name: `Azure Video Indexer`
- API version token: `2025-04-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_videoindexer.go`
- tests: `internal/server/provider_azure_videoindexer_test.go`
- example: `examples/azure/videoindexer-2025-04-01`
- plan doc: `docs/videoindexer-2025-04-01-plan.md`

Recommended coverage aliases:

- `videoindexer`
- `video_indexer`
- `video-indexer`
- `azure_video_indexer`
- `microsoft_videoindexer`
- `rest_videoindexer_2025_04_01`
- `videoindexer_2025_04_01`

## API Surface Summary

The reviewed pages expose five management-plane route families:

- provider-level discovery:
  - `/providers/Microsoft.VideoIndexer/operations`
- subscription-scoped account name validation:
  - `/subscriptions/{subscriptionId}/providers/Microsoft.VideoIndexer/checkNameAvailability`
- standard ARM tracked resources:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.VideoIndexer/accounts/{accountName}`
- account action routes:
  - `/accounts/{accountName}/generateAccessToken`
  - `/accounts/{accountName}/generateRestrictedViewerAccessToken`
  - `/accounts/{accountName}/generateExtensionAccessToken`
  - `/accounts/{accountName}/generateExtensionRestrictedViewerAccessToken`
- account child resources:
  - `/accounts/{accountName}/privateEndpointConnections/{privateEndpointConnectionName}`
  - `/accounts/{accountName}/privateLinkResources/{groupId}`

Modeling implications:

- the emulator should maintain a canonical account store keyed by ARM ID
- token-generation routes should be modeled as deterministic action endpoints rather than as stored resources
- `privateEndpointConnections` should be mutable child resources backed by a real store
- `privateLinkResources` should be read-only inventory derived from the account and network configuration model

## Contract Notes From Reviewed Method Pages

Shared management-plane traits:

- errors use `ErrorResponse`
- list routes return `value`
- tracked resources return standard ARM fields such as:
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
  - `/providers/Microsoft.VideoIndexer/operations`
- sample response includes at least:
  - `accounts/read`
  - `accounts/write`
  - `accounts/delete`

Modeling implications:

- expose a deterministic operation list covering the families implemented in the emulator
- include account CRUD, token-generation actions, private-endpoint-connection CRUD, and private-link-resource reads

### Accounts

Reviewed method pages:

- `Check Name Availability`
- `Create Or Update`
- `Delete`
- `Get`
- `List`
- `List By Resource Group`
- `Update`

Key traits from the reviewed pages:

- resource path:
  - `/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.VideoIndexer/accounts/{accountName}`
- `accountName` constraints:
  - min length `3`
  - max length `50`
  - pattern `^[A-Za-z0-9-]+$`
- `checkNameAvailability` requires:
  - `name`
  - `type = Microsoft.VideoIndexer/accounts`
- create and patch definitions reference:
  - managed identity
  - private endpoint connection details
  - OpenAI service details

Modeling implications:

- persist canonical account state keyed by ARM ID
- support `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`, list-by-resource-group, and list-by-subscription
- validate the documented account-name rules and name-availability request body
- keep the account `properties` shape broad enough for stable read-after-write behavior without over-modeling every nested optional field on the first pass

### Generate Token Actions

Reviewed method pages:

- `Access Token`
- `Restricted Viewer Access Token`
- `Extension Access Token`
- `Extension Restricted Viewer Access Token`

Key traits from the reviewed pages:

- all routes are `POST` beneath an account
- all return:
  - `200 OK`
  - body shape `{ "accessToken": "<jwt token...>" }`
- common parameter families include:
  - `permissionType`
  - `scope`
  - `projectId`
  - `videoId`
  - `tokenLifetimeInSeconds`
  - `extensionId`
- documented enum families include:
  - `permissionType`: `Reader`, `Contributor`
  - `scope`: `Account`, `Project`, `Video`
- extension token routes require `extensionId`
- restricted viewer variants do not accept `permissionType`
- `tokenLifetimeInSeconds` defaults to `3600` and caps at `43200` on the documented extension restricted viewer page, and the same lifetime pattern appears across the token pages

Modeling implications:

- implement these as pure action endpoints that do not create persistent ARM child resources
- validate the documented combinations at a pragmatic level:
  - required account exists
  - required fields are present
  - `permissionType` only on the contributor/reader token endpoints
  - `extensionId` required on extension token endpoints
  - lifetime is positive and capped
- return deterministic token-shaped strings that clearly vary by action, scope, and target identifiers

### Private Endpoint Connections

Reviewed method pages:

- `Create Or Update`
- `Delete`
- `Get`
- `List By Account`

Key traits from the reviewed pages:

- route:
  - `/accounts/{accountName}/privateEndpointConnections/{privateEndpointConnectionName}`
- resource type:
  - `Microsoft.VideoIndexer/accounts/privateEndpointConnections`
- properties include:
  - `provisioningState`
  - `privateEndpoint.id`
  - `privateLinkServiceConnectionState.status`
  - `privateLinkServiceConnectionState.description`
  - `privateLinkServiceConnectionState.actionsRequired`
  - `groupIds`

Modeling implications:

- store private endpoint connections keyed by account child ARM ID
- support list/get/put/delete
- keep update semantics pragmatic:
  - primarily mutate connection-state fields and private-endpoint references
- seed at least one connection per seeded account so list/get coverage is meaningful

### Private Link Resources

Reviewed method pages:

- `Get`
- `List By Account`

Key traits from the reviewed pages:

- route:
  - `/accounts/{accountName}/privateLinkResources/{groupId}`
- resource type:
  - `Microsoft.VideoIndexer/accounts/privateLinkResources`
- properties include:
  - `groupId`
  - `requiredMembers`
  - `requiredZoneNames`
- sample response uses:
  - `groupId = avi`
  - `requiredZoneNames = ["privatelink.avi.microsoft.com"]`

Modeling implications:

- model this family as deterministic read-only inventory derived from the account
- a single seeded group such as `avi` is enough for first-pass coverage

## Staged Implementation

### Stage 0: Router and Coverage Wiring

Add the service shell and discovery:

- add `internal/server/provider_azure_videoindexer.go`
- register the new router in `internal/server/http.go`
- add `Server` storage maps for:
  - accounts
  - private endpoint connections
- add service aliases to:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Acceptance target:

- service is discoverable by the coverage scripts
- no routed Video Indexer path falls through to `NotImplemented`

### Stage 1: Provider and Account Foundation

Implement:

- `GET /providers/Microsoft.VideoIndexer/operations`
- `POST checkNameAvailability`
- account `PUT`, `GET`, `HEAD`, `PATCH`, `DELETE`
- list-by-resource-group
- list-by-subscription

Modeling notes:

- seed one account with stable ARM metadata and representative account properties
- support name validation and duplicate-name rejection

Acceptance target:

- base resource lifecycle works
- tests cover bad `api-version`, invalid account names, name-availability responses, and read-after-write

### Stage 2: Token Generation Actions

Implement:

- `generateAccessToken`
- `generateRestrictedViewerAccessToken`
- `generateExtensionAccessToken`
- `generateExtensionRestrictedViewerAccessToken`

Modeling notes:

- keep the returned token deterministic but opaque-looking
- validate the documented required-field differences between the four endpoints
- derive token content from:
  - account name
  - action
  - scope
  - video/project/extension identifiers

Acceptance target:

- all token action routes stop returning `NotImplemented`
- validation tests cover missing fields and invalid enum/lifetime cases

### Stage 3: Private Endpoint Connections

Implement:

- `privateEndpointConnections` create-or-update
- `privateEndpointConnections` get
- `privateEndpointConnections` list-by-account
- `privateEndpointConnections` delete

Modeling notes:

- seed one approved connection on the default account
- keep `groupIds`, `privateEndpoint.id`, and `privateLinkServiceConnectionState` stable and visible on read-after-write

Acceptance target:

- child lifecycle works under an existing account
- tests cover list/get/mutate/delete and not-found behavior

### Stage 4: Private Link Resources

Implement:

- `privateLinkResources` get
- `privateLinkResources` list-by-account

Modeling notes:

- treat this family as read-only inventory
- seed `avi` as the default group ID with:
  - `requiredMembers = ["avi"]`
  - `requiredZoneNames = ["privatelink.avi.microsoft.com"]`

Acceptance target:

- private-link-resource routes are no longer unimplemented
- list/get shapes match ARM child-resource expectations

### Stage 5: Example

Add `examples/azure/videoindexer-2025-04-01`:

- use `azcore/runtime` and `azcore/policy`
- do not depend on a speculative typed `armvideoindexer` client

Suggested example flow:

1. list provider `operations`
2. check account name availability
3. create an account
4. get and patch the account
5. list accounts by resource group
6. generate account access token
7. generate restricted viewer token
8. generate extension token
9. list private endpoint connections
10. get private link resources

Acceptance target:

- example builds cleanly
- example exercises the main reviewed surface without test-only hooks

### Stage 6: Validation and Coverage

Validation after implementation:

- `go test ./internal/server -run 'TestAzureVideoIndexer' -count=1`
- `go build ./examples/azure/videoindexer-2025-04-01`
- `python3 scripts/azure-contract-coverage.py --service videoindexer --require-service videoindexer --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service videoindexer --require-service videoindexer --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service videoindexer --require-service videoindexer --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service videoindexer --require-service videoindexer --fail-on any`

## Test Plan

Focused provider tests should cover:

- provider route detection and invalid `api-version`
- account create/get/head/list/update/delete lifecycle
- name-availability success, duplicate-name, and invalid-request cases
- token-generation success for all four action routes
- token-generation validation failures for missing `permissionType`, missing `extensionId`, invalid `scope`, and invalid lifetime
- private-endpoint-connection list/get/update/delete
- private-link-resource list/get
- not-found behavior for unknown account and child resources

## Risks and Follow-Ups

- Some Learn pages for this surface are partially authorization-gated, so implementation should stick closely to the clearly surfaced URI parameters, response shapes, examples, and definition tables rather than overcommitting to hidden optional fields.
- The token-generation routes are management-plane actions returning opaque strings; tests should validate shape and determinism, not JWT semantics.
- I did not find a clearly correct typed Go SDK package for `Microsoft.VideoIndexer` management `2025-04-01`, so the example should use the Azure core runtime directly unless a matching official client is later confirmed.
