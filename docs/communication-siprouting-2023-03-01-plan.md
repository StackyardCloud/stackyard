# Azure Communication - Sip Routing (`communication-siprouting-2023-03-01`) Staged Plan

## Objective

Emulate the Azure Communication Services Sip Routing data-plane API for `2023-03-01` with deterministic local behavior for:

- SIP configuration retrieval
- SIP configuration merge-patch updates
- trunk and route validation within a single SIP configuration document
- an Azure Go SDK-based example under `examples/azure/communication-siprouting-2023-03-01`
- Azure coverage-script wiring for the new service key

Important scope note:

- The reviewed Learn surface is an ACS endpoint-rooted data plane, not an ARM `subscriptions/.../providers/...` resource provider.
- The documented routes are rooted at the Communication resource endpoint, for example `{resource}.communication.azure.com/sip?...`.
- This should be modeled as a separate Stackyard Azure service from Communication management-plane APIs and from the other ACS data-plane services already in the repo.

Important SDK note:

- I did not confirm a clearly matching current typed official Azure Go SDK package dedicated to this exact Sip Routing `2023-03-01` REST surface during review.
- The future example should therefore use the official Azure Go SDK core runtime pipeline, specifically `azcore/runtime` and `azcore/policy`, unless a typed Sip Routing client is later confirmed in the repo or official Azure package index.

Important documentation note:

- The reviewed `2023-03-01` operation-group page exposes a single operation group:
  - `Sip Routing`
- The route surface for this version is intentionally small: `GET /sip` and `PATCH /sip`.
- The plan below stays at that verified scope and does not invent extra SIP-routing validation, health, domain, or preview-only operations from other API versions.

## Primary References Reviewed

Primary operation-groups page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/siprouting/operation-groups?view=rest-communication-siprouting-2023-03-01`

Operation-group page reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/siprouting/sip-routing?view=rest-communication-siprouting-2023-03-01`

Linked method pages reviewed:

- `https://learn.microsoft.com/en-us/rest/api/communication/siprouting/sip-routing/get?view=rest-communication-siprouting-2023-03-01`
- `https://learn.microsoft.com/en-us/rest/api/communication/siprouting/sip-routing/update?view=rest-communication-siprouting-2023-03-01`

## Service Naming

Canonical Stackyard naming:

- service key: `communication_siprouting`
- display name: `Azure Communication - Sip Routing`
- API version token: `2023-03-01`

Recommended repo paths:

- implementation: `internal/server/provider_azure_communication_siprouting.go`
- tests: `internal/server/provider_azure_communication_siprouting_test.go`
- example: `examples/azure/communication-siprouting-2023-03-01`
- plan doc: `docs/communication-siprouting-2023-03-01-plan.md`

Recommended coverage aliases:

- `communication_siprouting`
- `communication-siprouting`
- `siprouting`
- `sip_routing`
- `azure_communication_siprouting`
- `acs_siprouting`
- `rest_communication_siprouting_2023_03_01`
- `communication_siprouting_2023_03_01`

## Surface Summary

The reviewed REST surface contains 1 operation group and 2 concrete methods:

1. `Sip Routing`

Documented route family:

- `GET {endpoint}/sip?api-version=2023-03-01`
- `PATCH {endpoint}/sip?api-version=2023-03-01`

Recommended Stackyard route family:

- `/azure/communication-siprouting/2023-03-01/sip`

Important routing note:

- This API should not be mixed into the ARM Azure router chain.
- It should be implemented as a data-plane style Azure route family similar to the repo’s other Communication endpoint-rooted services.
- Every route still requires `api-version=2023-03-01`.

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:

- security is documented for ACS endpoint authentication rather than ARM bearer flows
- errors use `Communication Error Response`
- success semantics are synchronous standard HTTP
- the SIP configuration is treated as a single document containing:
  - `trunks`
  - `routes`

### Get SIP Configuration

Reviewed method page:

- `Get`

High-value route behavior from the reviewed page:

- `GET /sip`
- success status: `200 OK`
- response type: `Sip Configuration`
- response shape includes:
  - `trunks` as a map keyed by trunk FQDN
  - `routes` as an ordered array

Reviewed response field details:

- trunk map key:
  - FQDN, 1-249 characters
- `Trunk`
  - `sipSignalingPort`
- `Trunk Route`
  - `name`
  - optional `description`
  - `numberPattern`
  - optional `trunks`

Modeling implications:

- first-pass emulation should maintain one SIP configuration document per Stackyard server instance
- route order matters and should be preserved as stored
- a deterministic default configuration should be seeded so `GET /sip` succeeds even before any updates

### Update SIP Configuration

Reviewed method page:

- `Update`

High-value route behavior from the reviewed page:

- `PATCH /sip`
- request media type:
  - `application/merge-patch+json`
- request body type:
  - `Sip Configuration Update`
- request body may include:
  - `trunks`
  - `routes`
- success status: `200 OK`
- response type:
  - updated `Sip Configuration`

Reviewed request field details:

- `trunks` is a map keyed by FQDN to `Trunk Update`
- `Trunk Update`
  - `sipSignalingPort`
- `routes` is a full ordered list of `Trunk Route`
- `Trunk Route`
  - `name` max length 256
  - `description` max length 1024
  - `numberPattern` max length 1024
  - `trunks` as list of FQDN strings

Modeling implications:

- first-pass update semantics should be:
  - if `trunks` is present, replace the stored trunk map with the provided map
  - if `routes` is present, replace the stored route list with the provided ordered list
  - omitted sections remain unchanged
- validation should be deterministic and local:
  - trunk keys must be non-empty and within documented max length
  - `sipSignalingPort` must be a positive integer in a reasonable SIP port range
  - route names and number patterns must be non-empty
  - route trunk references should resolve against the post-patch trunk map

## Stage 1: Route Skeleton And Shared Models

Add:

- `internal/server/provider_azure_communication_siprouting.go`

Implement:

- service constants:
  - API version `2023-03-01`
  - route prefix `/azure/communication-siprouting/2023-03-01`
- router entrypoint:
  - `handleAzureCommunicationSipRoutingRouter`
- shared helpers for:
  - JSON body decoding
  - merge-patch-style field presence checks
  - ACS-style error payloads

Add provider state in `internal/server/http.go` for:

- a single SIP configuration document

Exit criteria:

- `/azure/communication-siprouting/2023-03-01/sip` is recognized
- missing or wrong `api-version` returns deterministic validation errors
- unsupported paths return `PathNotFound`

## Stage 2: Implement Deterministic Configuration Get

Implement:

- `GET /sip`

Seed a default deterministic configuration such as:

- trunks:
  - `vendor.xyz.biz` -> `sipSignalingPort: 5061`
  - `vendor.abc.biz` -> `sipSignalingPort: 5061`
- routes:
  - `LocalRoute` with a US-style `numberPattern`
  - `VendorXYZ` with `description`, `numberPattern`, and trunk reference

Exit criteria:

- `GET /sip` returns a valid `Sip Configuration`
- route ordering is stable
- the response matches the reviewed shape closely enough for typed fixture coverage

## Stage 3: Implement Merge-Patch Update

Implement:

- `PATCH /sip`

Recommended validation:

- body must be valid JSON object
- if present, `trunks` must be an object keyed by FQDN strings
- FQDN keys must be non-empty and no longer than 249 characters
- `sipSignalingPort`, when present, must be a positive integer
- if present, `routes` must be an array
- each route must include:
  - non-empty `name`
  - non-empty `numberPattern`
- `description`, when present, must not exceed documented max length
- route trunk references must point to known trunks after the patch is applied

Recommended deterministic semantics:

- omitted top-level fields do not change current state
- provided `trunks` replace the entire trunk map
- provided `routes` replace the entire ordered route list
- return the final stored configuration after the patch

Exit criteria:

- patching trunks only works
- patching routes only works
- patching both together works
- invalid route-to-trunk references are rejected

## Stage 4: Provider Tests

Add:

- `internal/server/provider_azure_communication_siprouting_test.go`

Cover at minimum:

- default `GET /sip`
- `PATCH /sip` updating trunks only
- `PATCH /sip` updating routes only
- `PATCH /sip` updating both trunks and routes
- route ordering preserved after update
- validation failures for:
  - invalid API version
  - invalid JSON
  - empty trunk key
  - invalid `sipSignalingPort`
  - missing route `name`
  - missing route `numberPattern`
  - route referencing unknown trunk
  - unknown path
  - wrong method

Exit criteria:

- deterministic positive fixtures exist for both reviewed methods
- negative tests prove no Sip Routing route falls through to `NotImplemented`

## Stage 5: Azure Go SDK Example

Add:

- `examples/azure/communication-siprouting-2023-03-01/main.go`
- `examples/azure/communication-siprouting-2023-03-01/Dockerfile`
- `examples/azure/communication-siprouting-2023-03-01/docker-compose.yml`

Recommended example flow:

1. Get the current SIP configuration.
2. Patch the configuration with one updated trunk and one updated route list.
3. Get the SIP configuration again.

Recommended SDK approach:

- use `azcore/runtime` and `azcore/policy`
- implement a small shared-key signing policy that mirrors the repo’s other ACS data-plane examples
- do not invent an `arm...` client because this is not ARM

Exit criteria:

- the example is runnable against Stackyard
- the example demonstrates both `GET` and `PATCH`

## Stage 6: Coverage Script Wiring

Update:

- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `scripts/azure-doc-contract-coverage.py`

Add aliases for `communication_siprouting`, including:

- `communication_siprouting`
- `communication-siprouting`
- `siprouting`
- `sip_routing`
- `azure_communication_siprouting`
- `acs_siprouting`
- `rest_communication_siprouting_2023_03_01`
- `communication_siprouting_2023_03_01`

Map the plan doc in `azure-doc-contract-coverage.py`:

- `communication_siprouting` -> `communication-siprouting-2023-03-01-plan.md`

Exit criteria:

- the service is discoverable by all Azure coverage scripts
- the plan doc is associated with the service key

## Stage 7: Final Validation

Run:

- `go test ./internal/server -run 'TestAzureCommunicationSipRouting' -count=1`
- `go build ./examples/azure/communication-siprouting-2023-03-01`
- `python3 scripts/azure-contract-coverage.py --service communication_siprouting --require-service communication_siprouting --fail-on any`
- `python3 scripts/azure-io-contract-coverage.py --service communication_siprouting --require-service communication_siprouting --fail-on any`
- `python3 scripts/azure-doc-contract-coverage.py --service communication_siprouting --require-service communication_siprouting --fail-on any`
- `python3 scripts/azure-endpoint-coverage.py --service communication_siprouting --require-service communication_siprouting --fail-on any`
- `git diff --check -- ...`

Success criteria:

- both reviewed Sip Routing endpoints are covered and successful
- no Sip Routing endpoint returns `NotImplemented`
- the example builds cleanly
- the new service is addressable through the repo’s standard Azure coverage aliases
