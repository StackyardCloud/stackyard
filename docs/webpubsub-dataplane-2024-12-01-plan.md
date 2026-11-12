# Azure Web PubSub Service Data Plane (`webpubsub-dataplane-2024-12-01`) Staged Plan

## Objective

Emulate the Azure Web PubSub Service data-plane API for `2024-12-01` with deterministic local behavior for:

- service health status
- client token generation
- connection, user, and group existence checks
- direct group-membership mutations for connections and users
- filtered bulk group-membership mutations
- permission grant, revoke, and existence checks
- message send routes for all, group, user, and connection targets
- connection-closing routes for all, group, user, and single-connection targets
- connection listing within a group
- an Azure Go SDK example under `examples/azure/webpubsub-dataplane-2024-12-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is the Web PubSub service endpoint data plane, not the ARM `Microsoft.SignalRService/webPubSub` control plane already implemented in this repo.
- The routes are rooted at the service endpoint itself and use `/api/...`, not the Azure ARM `subscriptions/.../providers/...` envelope.
- This plan therefore treats the service as a separate Stackyard emulator surface from control-plane `webpubsub`.

Important SDK note:

- The official Azure Go SDK package for this data-plane surface is `github.com/Azure/azure-sdk-for-go/sdk/messaging/azwebpubsub`.
- The package documents typed client support for nearly the full reviewed route family:
  - token generation
  - connection, group, and user existence checks
  - send operations
  - close operations
  - membership operations
  - permission grant/revoke/check
- I did not confirm typed support for `Get Service Status` or `List Connections In Group` from the package index during review, so the future example may need a small `azcore/runtime` fallback for those two routes.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/operation-groups?view=rest-webpubsub-dataplane-2024-12-01`

Method pages reviewed from that index:

- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/add-connection-to-group/add-connection-to-group?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/add-connections-to-groups/add-connections-to-groups?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/add-user-to-group/add-user-to-group?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/check-permission/check-permission?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/close-all-connections/close-all-connections?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/close-connection/close-connection?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/close-group-connections/close-group-connections?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/close-user-connections/close-user-connections?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/connection-exists/connection-exists?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/generate-client-token/generate-client-token?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/get-service-status/get-service-status?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/grant-permission/grant-permission?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/group-exists/group-exists?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/list-connections-in-group/list-connections-in-group?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/remove-connection-from-all-groups/remove-connection-from-all-groups?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/remove-connection-from-group/remove-connection-from-group?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/remove-connections-from-groups/remove-connections-from-groups?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/remove-user-from-all-groups/remove-user-from-all-groups?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/remove-user-from-group/remove-user-from-group?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/revoke-permission/revoke-permission?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/send-to-all/send-to-all?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/send-to-connection/send-to-connection?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/send-to-group/send-to-group?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/send-to-user/send-to-user?view=rest-webpubsub-dataplane-2024-12-01`
- `https://learn.microsoft.com/en-us/rest/api/webpubsub/dataplane/user-exists/user-exists?view=rest-webpubsub-dataplane-2024-12-01`

## Service Naming

Canonical Stackyard naming:

- service key: `webpubsub_data_plane`
- display name: `Azure Web PubSub Service Data Plane`
- API version token: `2024-12-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_webpubsub_data_plane.go`
- tests: `internal/server/provider_azure_webpubsub_data_plane_test.go`
- example: `examples/azure/webpubsub-dataplane-2024-12-01`
- plan doc: `docs/webpubsub-dataplane-2024-12-01-plan.md`

Recommended coverage aliases:

- `webpubsub_data_plane`
- `webpubsub_dataplane`
- `webpubsub-data-plane`
- `webpubsub-dataplane`
- `azure_webpubsub_data_plane`
- `azure_webpubsub_dataplane`
- `web_pubsub_data_plane`
- `web_pubsub_dataplane`
- `rest_webpubsub_dataplane_2024_12_01`
- `webpubsub_2024_12_01_dataplane`

## Surface Summary

The reviewed REST surface is a single service-endpoint route family rooted at:

- `HEAD {endpoint}/api/health`
- `/api/hubs/{hub}/...`

Operation families exposed by the reviewed pages:

1. health and token bootstrap
2. existence checks
3. direct membership operations
4. filtered bulk membership operations
5. permission operations
6. send operations
7. close operations
8. group connection listing

Route families documented on the method pages:

- `HEAD {endpoint}/api/health?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/:generateToken?api-version=2024-12-01`
- `HEAD {endpoint}/api/hubs/{hub}/connections/{connectionId}?api-version=2024-12-01`
- `HEAD {endpoint}/api/hubs/{hub}/groups/{group}?api-version=2024-12-01`
- `HEAD {endpoint}/api/hubs/{hub}/users/{userId}?api-version=2024-12-01`
- `PUT {endpoint}/api/hubs/{hub}/groups/{group}/connections/{connectionId}?api-version=2024-12-01`
- `PUT {endpoint}/api/hubs/{hub}/users/{userId}/groups/{group}?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/:addToGroups?api-version=2024-12-01`
- `DELETE {endpoint}/api/hubs/{hub}/connections/{connectionId}/groups?api-version=2024-12-01`
- `DELETE {endpoint}/api/hubs/{hub}/groups/{group}/connections/{connectionId}?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/:removeFromGroups?api-version=2024-12-01`
- `DELETE {endpoint}/api/hubs/{hub}/users/{userId}/groups?api-version=2024-12-01`
- `DELETE {endpoint}/api/hubs/{hub}/users/{userId}/groups/{group}?api-version=2024-12-01`
- `HEAD {endpoint}/api/hubs/{hub}/permissions/{permission}/connections/{connectionId}?api-version=2024-12-01`
- `PUT {endpoint}/api/hubs/{hub}/permissions/{permission}/connections/{connectionId}?api-version=2024-12-01`
- `DELETE {endpoint}/api/hubs/{hub}/permissions/{permission}/connections/{connectionId}?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/:send?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/connections/{connectionId}/:send?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/groups/{group}/:send?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/users/{userId}/:send?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/:closeConnections?api-version=2024-12-01`
- `DELETE {endpoint}/api/hubs/{hub}/connections/{connectionId}?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/groups/{group}/:closeConnections?api-version=2024-12-01`
- `POST {endpoint}/api/hubs/{hub}/users/{userId}/:closeConnections?api-version=2024-12-01`
- `GET {endpoint}/api/hubs/{hub}/groups/{group}/connections?api-version=2024-12-01`

Recommended Stackyard route family:

- `/azure/webpubsub-data-plane/2024-12-01/api/health`
- `/azure/webpubsub-data-plane/2024-12-01/api/hubs/{hub}/...`

Modeling implications:

- this service should maintain data-plane state by `hub`
- connections, users, groups, and granted permissions should be tracked independently from the control-plane account model
- group membership needs both direct mutation and filtered bulk mutation support
- existence checks are `HEAD` routes whose primary contract is status code, not response JSON
- send and close routes mostly return `204 No Content`
- `List Connections In Group` is the only reviewed read route with a structured collection response and continuation support

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed method pages:

- security is OAuth2 with scope `webpubsub.azure.com/.default`
- hub names use pattern `^[A-Za-z][A-Za-z0-9_`,.[\]]{0,127}$`
- group names require length `1..1024` and pattern `^(?!\\s+$).+$`
- connection IDs and user IDs require minimum length `1`
- errors use `Azure.Core.Foundations.ErrorResponse`
- mutation routes mostly return `200` or `204`

### Health And Token Bootstrap

Reviewed method pages:

- `Get Service Status`
- `Generate Client Token`

Key traits from the reviewed pages:

- `HEAD /api/health` returns service health via status code
- `POST /api/hubs/{hub}/:generateToken`
- optional token query parameters:
  - `userId`
  - `roles[]`
  - `minutesToExpire`
  - `groups[]`
  - `clientProtocol`
- token response payload contains:
  - `token`
- `clientProtocol` supports Web PubSub client-type semantics, with docs noting `Default` and `MQTT`

Modeling implications:

- implement health first as the readiness route for the data-plane shim
- generate deterministic mock tokens from hub, user, role, group, and expiry inputs
- keep token generation stateless for first-pass emulation

### Existence Checks

Reviewed method pages:

- `Connection Exists`
- `Group Exists`
- `User Exists`
- `Check Permission`

Key traits from the reviewed pages:

- existence and permission checks are `HEAD` routes
- success is `200`
- missing target is `404`
- permission routes use enum:
  - `sendToGroup`
  - `joinLeaveGroup`
- permission checks may require `targetName` for group-scoped permissions

Modeling implications:

- represent existence mostly via status code
- persist explicit granted permissions per connection so `Check Permission` is not just inferred
- validate `targetName` when permission scope requires it

### Direct Membership Operations

Reviewed method pages:

- `Add Connection To Group`
- `Add User To Group`
- `Remove Connection From Group`
- `Remove User From Group`
- `Remove Connection From All Groups`
- `Remove User From All Groups`

Key traits from the reviewed pages:

- direct add/remove routes are `PUT` or `DELETE`
- add routes return `200`
- remove routes generally return `204`
- connection-wide and user-wide removal from all groups use the `/groups` collection route without a group segment

Modeling implications:

- store memberships both by group and reverse-index by connection/user for efficient “remove from all groups”
- keep operations idempotent
- support mixed membership sources: a user can imply multiple active connections

### Filtered Bulk Membership Operations

Reviewed method pages:

- `Add Connections To Groups`
- `Remove Connections From Groups`

Key traits from the reviewed pages:

- bulk routes use:
  - `POST /api/hubs/{hub}/:addToGroups`
  - `POST /api/hubs/{hub}/:removeFromGroups`
- request bodies include:
  - `filter`
  - `groups[]`
- docs explicitly call out OData-style filter syntax
- success is `204`

Modeling implications:

- first-pass emulator should support a small deterministic subset of filter behavior
- recommended subset:
  - `startswith(userId, 'prefix')`
  - `userId eq 'value'`
  - `connectionId eq 'value'`
- any unsupported filter should return a typed validation error rather than silently matching everything

### Permission Operations

Reviewed method pages:

- `Grant Permission`
- `Revoke Permission`
- `Check Permission`

Key traits from the reviewed pages:

- permission routes are scoped to a connection
- `Grant Permission` uses `PUT`
- `Revoke Permission` uses `DELETE`
- `Check Permission` uses `HEAD`
- `targetName` is required for `joinLeaveGroup` and `sendToGroup`

Modeling implications:

- keep permissions as explicit records keyed by `(hub, connectionId, permission, targetName)`
- use the same permission store for grant, revoke, and check

### Send Operations

Reviewed method pages:

- `Send To All`
- `Send To Connection`
- `Send To Group`
- `Send To User`

Key traits from the reviewed pages:

- send routes use `POST`
- accepted request media types include:
  - `application/json`
  - `application/octet-stream`
  - `text/plain`
- TTL query parameter:
  - `messageTtlSeconds`
  - maximum `300`
- route-specific query options include:
  - `excluded[]` on broadcast and group-send
  - `filter` on broadcast, group-send, and user-send
- success is `204`

Modeling implications:

- emulator should not actually fan out messages, but it should persist a last-message record per hub for deterministic inspection if needed
- validate content type, TTL range, and target existence where the docs imply it
- preserve body bytes for binary and text payloads

### Close Operations

Reviewed method pages:

- `Close All Connections`
- `Close Connection`
- `Close Group Connections`
- `Close User Connections`

Key traits from the reviewed pages:

- close routes return `200` or `204` depending on route
- close routes support optional query/body options such as exclusion or reason fields
- they are control-like mutations over current live connection state

Modeling implications:

- represent closed connections with a simple state transition instead of deleting them outright
- keep per-connection metadata such as close reason and closed time for deterministic follow-up behavior

### Group Connection Listing

Reviewed method page:

- `List Connections In Group`

Key traits from the reviewed page:

- `GET /api/hubs/{hub}/groups/{group}/connections`
- optional query parameters:
  - `maxpagesize`
  - `top`
  - `continuationToken`
- returns a structured collection page

Modeling implications:

- implement deterministic ordering by connection ID
- support simple continuation-token pagination
- expose enough shape for coverage to prove this route is implemented rather than falling through

## Staged Implementation Plan

### Stage 0: Route Skeleton And Coverage Wiring

Goal:

- reserve the data-plane service as distinct from control-plane `webpubsub`

Changes:

- add route entry in `internal/server/http.go`
- add new provider file `internal/server/provider_azure_webpubsub_data_plane.go`
- add empty-but-routed coverage aliases for `webpubsub_data_plane`
- add plan-doc mapping in `scripts/azure-doc-contract-coverage.py`

Exit criteria:

- the router claims only `/azure/webpubsub-data-plane/2024-12-01/...`
- coverage scripts discover canonical service `webpubsub_data_plane`

### Stage 1: Shared Data-Plane Model, Health, And Token Generation

Goal:

- establish the core hub/connection/user/group state model and readiness route

Changes:

- add in-memory stores for:
  - hubs
  - connections
  - user-to-connection membership
  - group membership
  - granted permissions
  - last-send records
- implement:
  - `HEAD /api/health`
  - `POST /api/hubs/{hub}/:generateToken`

Exit criteria:

- deterministic `204` or `200` health response
- token generation returns stable `token` JSON and validates hub/query inputs

### Stage 2: Existence And Direct Membership Routes

Goal:

- cover the simplest operational routes first

Changes:

- implement:
  - `Connection Exists`
  - `Group Exists`
  - `User Exists`
  - `Add Connection To Group`
  - `Add User To Group`
  - `Remove Connection From Group`
  - `Remove User From Group`
  - `Remove Connection From All Groups`
  - `Remove User From All Groups`

Exit criteria:

- direct group mutation round-trips via `HEAD` existence checks
- idempotent delete semantics are covered in tests

### Stage 3: Permissions

Goal:

- model connection-scoped permission state cleanly before send routes depend on it

Changes:

- implement:
  - `Grant Permission`
  - `Revoke Permission`
  - `Check Permission`
- validate:
  - permission enum
  - required `targetName` for group-scoped permissions

Exit criteria:

- grant, head-check, revoke flow is deterministic
- missing permission returns `404`

### Stage 4: Send And Close Operations

Goal:

- cover the high-value messaging and connection-control routes

Changes:

- implement:
  - `Send To All`
  - `Send To Connection`
  - `Send To Group`
  - `Send To User`
  - `Close All Connections`
  - `Close Connection`
  - `Close Group Connections`
  - `Close User Connections`
- validate:
  - content type
  - TTL limit
  - optional excluded/filter parameters

Exit criteria:

- all send routes return correct no-content responses
- close routes mutate connection state in a verifiable way

### Stage 5: Filtered Bulk Membership And Group Connection Listing

Goal:

- finish the remaining data-plane collection semantics

Changes:

- implement:
  - `Add Connections To Groups`
  - `Remove Connections From Groups`
  - `List Connections In Group`
- support deterministic subset of filter parsing and continuation tokens

Exit criteria:

- group listing paginates deterministically
- unsupported filters fail with typed validation responses

### Stage 6: Example And Validation

Goal:

- prove the surface using the official Go SDK and the repo coverage gates

Changes:

- add example under `examples/azure/webpubsub-dataplane-2024-12-01`
- prefer `azwebpubsub` typed client for:
  - token generation
  - membership operations
  - permission operations
  - send operations
  - close operations
  - existence checks
- use `azcore/runtime` only if needed for:
  - `Get Service Status`
  - `List Connections In Group`
- add focused tests in `internal/server/provider_azure_webpubsub_data_plane_test.go`
- update:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`

Validation commands to run during implementation:

- `go test ./internal/server -run 'TestAzureWebPubSubDataPlane' -count=1`
- `go build ./examples/azure/webpubsub-dataplane-2024-12-01`
- `python3 scripts/azure-contract-coverage.py --service webpubsub_data_plane --require-service webpubsub_data_plane --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service webpubsub_data_plane --require-service webpubsub_data_plane --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service webpubsub_data_plane --require-service webpubsub_data_plane --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service webpubsub_data_plane --require-service webpubsub_data_plane --fail-on any`
- `make ci`

## Suggested Test Matrix

- healthy service responds on `HEAD /api/health`
- token generation validates TTL and returns deterministic token JSON
- connection/group/user existence routes return `200` vs `404`
- direct add/remove operations are idempotent
- bulk membership routes validate filter syntax and target groups
- permission grant/check/revoke flow works for both permission enum values
- send routes validate content types and TTL, then return `204`
- close routes alter follow-up existence/list behavior predictably
- group connection listing returns stable pagination and continuation

## Risks And Follow-Ups

- The docs use authorization-gated Learn pages, but the route and schema details were still visible during review. Implementation should stick closely to those visible contracts and avoid inventing unsupported extras.
- Filter semantics could sprawl quickly; keep first-pass support narrow and documented.
- The official Go package appears to cover most server operations already, so the future example should stay typed wherever possible instead of defaulting to raw HTTP.
