# Azure Communication - Identity (`communication-identity-2025-06-30`) Staged Plan

## Objective

Emulate the Azure Communication Services Identity data-plane API for `2025-06-30` with deterministic local behavior for:

- core ACS identity creation and deletion
- ACS access-token issuance and revocation
- Teams user access-token exchange
- Teams extension assignment upsert, get, and delete
- Teams extension token exchange
- an Azure Go SDK-based example under `examples/azure/communication-identity-2025-06-30`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Identity `2025-06-30` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed client is later confirmed in the repo or official package index.

Important documentation note:

- The `2025-06-30` operation-groups page exposes 3 operation groups:
  - `Communication Identity`
  - `Teams Extension Assignment`
  - `Teams Extension Token`
- Some of the method pages are authorization-gated on Learn, but the linked pages and search-visible extracts were still sufficient to confirm the route surface, request and response shapes, and primary success/error semantics.
- The plan below stays at that verified scope and does not invent unrelated Communication-management or phone-number-management routes.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/identity/operation-groups?view=rest-communication-identity-2025-06-30`

Operation-group pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/identity/communication-identity?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/teams-extension-assignment?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/teams-extension-token?view=rest-communication-identity-2025-06-30`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/identity/communication-identity/create?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/communication-identity/delete?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/communication-identity/issue-access-token?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/communication-identity/revoke-access-tokens?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/communication-identity/exchange-teams-user-access-token?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/teams-extension-assignment/upsert?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/teams-extension-assignment/get?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/teams-extension-assignment/delete?view=rest-communication-identity-2025-06-30`
- `https://learn.microsoft.com/en-us/rest/api/communication/identity/teams-extension-token/exchange?view=rest-communication-identity-2025-06-30`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_identity`
- display name: `Azure Communication - Identity`
- API version token: `2025-06-30`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_identity.go`
- tests: `internal/server/provider_azure_communication_identity_test.go`
- example: `examples/azure/communication-identity-2025-06-30`
- plan doc: `docs/communication-identity-2025-06-30-plan.md`

Recommended coverage aliases:

- `communication_identity`
- `communication-identity`
- `identity`
- `azure_communication_identity`
- `acs_identity`
- `rest_communication_identity_2025_06_30`
- `communication_identity_2025_06_30`

## Surface Summary

The reviewed REST surface contains 3 operation groups and 9 concrete methods:

1. `Communication Identity`
2. `Teams Extension Assignment`
3. `Teams Extension Token`

Documented route family:

- `POST {endpoint}/identities?api-version=2025-06-30`
- `DELETE {endpoint}/identities/{id}?api-version=2025-06-30`
- `POST {endpoint}/identities/{id}/:issueAccessToken?api-version=2025-06-30`
- `POST {endpoint}/identities/{id}/:revokeAccessTokens?api-version=2025-06-30`
- `POST {endpoint}/teamsUser/:exchangeAccessToken?api-version=2025-06-30`
- `PUT {endpoint}/access/teamsExtension/tenants/{tenantId}/assignments/{objectId}?api-version=2025-06-30`
- `GET {endpoint}/access/teamsExtension/tenants/{tenantId}/assignments/{objectId}?api-version=2025-06-30`
- `DELETE {endpoint}/access/teamsExtension/tenants/{tenantId}/assignments/{objectId}?api-version=2025-06-30`
- `POST {endpoint}/access/teamsExtension/:exchangeAccessToken?api-version=2025-06-30`

Recommended Stackyard route family:

- `/azure/communication-identity/2025-06-30/identities`
- `/azure/communication-identity/2025-06-30/identities/{id}`
- `/azure/communication-identity/2025-06-30/identities/{id}/:issueAccessToken`
- `/azure/communication-identity/2025-06-30/identities/{id}/:revokeAccessTokens`
- `/azure/communication-identity/2025-06-30/teamsUser/:exchangeAccessToken`
- `/azure/communication-identity/2025-06-30/access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`
- `/azure/communication-identity/2025-06-30/access/teamsExtension/:exchangeAccessToken`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other Communication endpoint-rooted services.
- Every route still requires `api-version=2025-06-30`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- security is documented for ACS endpoint authentication rather than ARM bearer flows
- errors use `CommunicationErrorResponse`
- success semantics are standard HTTP rather than ARM async polling
- route parameters are endpoint-rooted and identity-centric

### Communication Identity

Reviewed method pages:

- `Create`
- `Delete`
- `Issue Access Token`
- `Revoke Access Tokens`
- `Exchange Teams User Access Token`

High-value route behavior from the reviewed pages:

- `Create`
  - `POST /identities`
  - optional body:
    - `createTokenWithScopes`
    - `expiresInMinutes`
  - success status: `201 Created`
  - response type: `CommunicationIdentityAccessTokenResult`
  - response includes:
    - `identity.id`
    - optional `accessToken.token`
    - optional `accessToken.expiresOn`
- `Delete`
  - `DELETE /identities/{id}`
  - success status: `204 No Content`
- `Issue Access Token`
  - `POST /identities/{id}/:issueAccessToken`
  - required body:
    - `scopes`
  - optional body:
    - `expiresInMinutes`
  - success status: `200 OK`
  - response type: `CommunicationIdentityAccessToken`
- `Revoke Access Tokens`
  - `POST /identities/{id}/:revokeAccessTokens`
  - success status: `204 No Content`
- `Exchange Teams User Access Token`
  - `POST /teamsUser/:exchangeAccessToken`
  - required body:
    - `token`
    - `appId`
    - `userId`
  - success status: `200 OK`
  - response type: `CommunicationIdentityAccessToken`

Reviewed token-scope values:

- `chat`
- `voip`
- `chat.join`
- `chat.join.limited`
- `voip.join`

Modeling implications:

- first-pass emulation should maintain a deterministic ACS identity store keyed by `identity.id`
- `Create` should optionally mint an access token when `createTokenWithScopes` is present
- `Issue Access Token` should validate scopes and return a deterministic token payload
- `Revoke Access Tokens` should mark all existing tokens for that identity as revoked without needing deep token introspection
- `Exchange Teams User Access Token` can validate only the high-value shape from the reviewed page and return a deterministic ACS-style access token

### Teams Extension Assignment

Reviewed method pages:

- `Upsert`
- `Get`
- `Delete`

High-value route behavior from the reviewed pages:

- `Upsert`
  - `PUT /access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`
  - required body:
    - `principalType`
  - optional body:
    - `clientIds`
  - success status:
    - `201 Created` on first create
    - `200 OK` on replacement
  - response type: `TeamsExtensionAssignmentResponse`
- `Get`
  - `GET /access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`
  - success status: `200 OK`
  - response type: `TeamsExtensionAssignmentResponse`
- `Delete`
  - `DELETE /access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`
  - success status: `204 No Content`

Reviewed assignment fields:

- `objectId`
- `tenantId`
- `principalType`
- optional `clientIds`

Reviewed principal-type values:

- `resourceAccount`
- `user`

Modeling implications:

- store assignments keyed by normalized `{tenantId}|{objectId}`
- `Upsert` should distinguish first-create from replace so status code behavior is deterministic
- `Get` and `Delete` should return consistent not-found errors instead of falling through to `NotImplemented`

### Teams Extension Token

Reviewed method page:

- `Exchange`

High-value route behavior from the reviewed page:

- `POST /access/teamsExtension/:exchangeAccessToken`
  - sample request body shown as `{}` on the reviewed page
  - success status: `200 OK`
  - response type: `CommunicationIdentityAccessTokenResult`
  - response includes:
    - `identity.id`
    - `accessToken.token`
    - `accessToken.expiresOn`

Modeling implications:

- first-pass emulation should treat this as a deterministic token-exchange route that returns an ACS identity plus token
- because the reviewed page did not expose a richer required-body contract in the visible content, the implementation should avoid over-validating beyond basic JSON acceptance unless later docs in the repo establish more strict requirements

## Recommended Emulator Shape

State model:

- `identities` keyed by ACS identity id
- `identity tokens` keyed by identity id plus token digest or issue sequence
- `teams extension assignments` keyed by normalized tenant and object ids

Each identity record should retain:

- `id`
- `createdAt`
- `updatedAt`
- `deletedAt` if removed
- issued token summaries
- revoked state or revoked timestamp

Suggested helper behaviors:

- generate deterministic ACS-looking identity ids
- generate deterministic token strings and expirations
- normalize tenant ids, object ids, and identity ids for stable lookups
- keep token issuance and exchange deterministic so repeated tests are stable

Error-handling expectations:

- reject wrong or missing `api-version`
- reject invalid JSON bodies
- reject missing required token scopes or exchange fields
- reject unknown identity ids and assignment ids with documented-style communication errors
- return method errors or path-not-found errors rather than `NotImplemented`

## Staged Implementation Plan

### Stage 1: Route Skeleton And Shared Validation

Add the new data-plane route family and service registration:

- implement `internal/server/provider_azure_communication_identity.go`
- wire the router into `internal/server/http.go`
- enforce `api-version=2025-06-30`
- add shared helpers for JSON decoding, error envelopes, identity generation, token generation, and assignment lookup

Exit criteria:

- all 9 documented routes resolve into the new provider
- unsupported methods return proper method errors rather than `NotImplemented`

### Stage 2: Core Identity Lifecycle

Implement:

- `POST /identities`
- `DELETE /identities/{id}`
- `POST /identities/{id}/:issueAccessToken`
- `POST /identities/{id}/:revokeAccessTokens`

Behavior goals:

- create deterministic identities
- optionally mint access tokens during create
- issue additional access tokens with validated scopes
- revoke all tokens for an identity
- remove an identity cleanly on delete

Exit criteria:

- core identity flows are deterministic and never fall through to `NotImplemented`

### Stage 3: Teams Token Exchange Routes

Implement:

- `POST /teamsUser/:exchangeAccessToken`
- `POST /access/teamsExtension/:exchangeAccessToken`

Behavior goals:

- validate the high-value documented request shape for Teams user exchange
- accept the minimal reviewed request shape for Teams extension token exchange
- return deterministic ACS identity and token payloads

Exit criteria:

- both exchange routes return the documented response family and do not rely on external identity providers

### Stage 4: Teams Extension Assignment Routes

Implement:

- `PUT /access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`
- `GET /access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`
- `DELETE /access/teamsExtension/tenants/{tenantId}/assignments/{objectId}`

Behavior goals:

- support create-versus-update status code differences
- echo principal type and client ids
- keep assignment storage keyed by tenant and object id

Exit criteria:

- assignment CRUD works deterministically and every reviewed assignment route is covered

### Stage 5: Provider Tests

Add focused tests in `internal/server/provider_azure_communication_identity_test.go` for:

- create identity without token
- create identity with token scopes
- issue access token with valid scopes
- revoke tokens
- delete identity
- Teams user token exchange
- Teams extension assignment create, update, get, and delete
- Teams extension token exchange
- invalid body validation
- missing required fields
- unknown identity and assignment ids
- incorrect `api-version`

Exit criteria:

- the new tests cover every documented route and main validation paths

### Stage 6: Azure Go SDK Example

Add `examples/azure/communication-identity-2025-06-30`:

- use `azcore/runtime` and `azcore/policy`
- authenticate with the same ACS endpoint-signing pattern used by the repo’s existing Communication data-plane examples
- create an identity
- issue a token
- upsert and get a Teams extension assignment
- exchange a Teams user or Teams extension token

The example should stay focused on a minimal but representative flow:

1. create an ACS identity with chat scope
2. issue another token for that identity
3. upsert and get a Teams extension assignment
4. exchange a Teams token route and print the resulting identity/token payload

Exit criteria:

- the example builds cleanly
- the example exercises both the core identity and Teams-extension route families

### Stage 7: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Required coverage additions:

- map the canonical service key `communication_identity`
- add the alias set listed above
- map the plan doc in the doc-coverage script

Exit criteria:

- coverage tooling recognizes `communication_identity`
- the service can be targeted explicitly by the Azure coverage scripts

## Risks And Edge Cases To Handle Deliberately

- token scope validation should stay constrained to the reviewed enum values rather than broad free-form acceptance
- revoked identities and revoked tokens should produce consistent not-found or invalid-state behavior in later calls
- `expiresInMinutes` should enforce the reviewed bounds of `60` through `1440`
- Teams extension assignment upsert should preserve deterministic create-versus-update status codes
- Teams extension token exchange should not over-assume hidden request fields that were not visible on the reviewed page

## Definition Of Done

The service is ready when all of the following are true:

- all 9 reviewed `2025-06-30` Identity routes are implemented
- no reviewed Identity route returns `NotImplemented`
- deterministic provider tests pass
- the example under `examples/azure/communication-identity-2025-06-30` builds and runs against Stackyard
- the Azure coverage scripts recognize and validate the new `communication_identity` service
