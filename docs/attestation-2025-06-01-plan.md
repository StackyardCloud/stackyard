# Azure Attestation (`attestation-2025-06-01`) Staged Plan

## Objective

Emulate Azure Attestation (`2025-06-01`) with deterministic local behavior for the documented data-plane surface, including:

- attestation evidence submission for all documented attestation types
- OpenID metadata and signing-certificate discovery
- attestation policy get/set/reset
- policy management certificate get/add/remove
- a typed Azure Go SDK example for `attestation-2025-06-01`
- Azure coverage script wiring for the new service key

Important scope note:
- The Microsoft Learn surface at `rest-attestation-2025-06-01` is the Azure Attestation data plane hosted per service endpoint.
- It is not the ARM management-plane `Microsoft.Attestation` resource-provider API.
- Management-plane provider CRUD exists as a separate service surface and should be planned separately if needed.

## Primary References Reviewed

Primary operation-groups page reviewed:
- `https://learn.microsoft.com/en-us/rest/api/attestation/operation-groups?view=rest-attestation-2025-06-01`

Operation-group pages reviewed:
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/metadata-configuration/get?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy-certificates?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/signing-certificates?view=rest-attestation-2025-06-01`

Every linked method page in those groups was also reviewed:
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation/attest-azure-guest?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation/attest-open-enclave?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation/attest-sev-snp-vm?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation/attest-sgx-enclave?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation/attest-tdx-vm?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/attestation/attest-tpm?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/metadata-configuration/get?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy/get?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy/set?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy/reset?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy-certificates/get?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy-certificates/add?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/policy-certificates/remove?view=rest-attestation-2025-06-01`
- `https://learn.microsoft.com/en-us/rest/api/attestation/signing-certificates/get?view=rest-attestation-2025-06-01`

SDK selection note:
- I did not find a current `2025-06-01` Go data-plane package on `pkg.go.dev`.
- The official Go SDK package I could verify for the data plane is the older track-1 package:
  - `github.com/Azure/azure-sdk-for-go/services/attestation/2020-10-01/attestation`
- That package exposes `Client`, `MetadataConfigurationClient`, `PolicyClient`, `PolicyCertificatesClient`, and `SigningCertificatesClient`.
- This is an inference from the SDK docs, so the example stage should be planned against the verified official package unless a newer package appears locally during implementation.

## Service Naming

Canonical Stackyard naming:
- service key: `attestation`
- display name: `Azure Attestation`
- API version: `2025-06-01`

Recommended repo paths:
- implementation: `internal/server/provider_azure_attestation.go`
- tests: `internal/server/provider_azure_attestation_test.go`
- example: `examples/azure/attestation-2025-06-01`
- plan doc: `docs/attestation-2025-06-01-plan.md`

## Surface Summary

The reviewed surface is split into 5 operation groups:

1. `Attestation`
2. `Metadata Configuration`
3. `Policy`
4. `Policy Certificates`
5. `Signing Certificates`

Total linked method pages reviewed: `14`

Method inventory:

`Attestation`
- `Attest Azure Guest`
- `Attest Open Enclave`
- `Attest Sev Snp Vm`
- `Attest Sgx Enclave`
- `Attest Tdx Vm`
- `Attest Tpm`

`Metadata Configuration`
- `Get`

`Policy`
- `Get`
- `Set`
- `Reset`

`Policy Certificates`
- `Get`
- `Add`
- `Remove`

`Signing Certificates`
- `Get`

## Route Envelope

Documented route family:
- `GET {endpoint}/.well-known/openid-configuration?api-version=2025-06-01`
- `GET {endpoint}/certs?api-version=2025-06-01`
- `GET {endpoint}/certificates?api-version=2025-06-01`
- `POST {endpoint}/certificates:add?api-version=2025-06-01`
- `POST {endpoint}/certificates:remove?api-version=2025-06-01`
- `GET {endpoint}/policies/{attestationType}?api-version=2025-06-01`
- `PUT {endpoint}/policies/{attestationType}?api-version=2025-06-01`
- `POST {endpoint}/policies/{attestationType}:reset?api-version=2025-06-01`
- `POST {endpoint}/attest/AzureGuest?api-version=2025-06-01`
- `POST {endpoint}/attest/OpenEnclave?api-version=2025-06-01`
- `POST {endpoint}/attest/SevSnpVm?api-version=2025-06-01`
- `POST {endpoint}/attest/SgxEnclave?api-version=2025-06-01`
- `POST {endpoint}/attest/TdxVm?api-version=2025-06-01`
- `POST {endpoint}/attest/Tpm?api-version=2025-06-01`

Recommended Stackyard emulation prefix:
- `/azure/attestation`

Recommended Stackyard routes:
- `/azure/attestation/.well-known/openid-configuration`
- `/azure/attestation/certs`
- `/azure/attestation/certificates`
- `/azure/attestation/certificates:add`
- `/azure/attestation/certificates:remove`
- `/azure/attestation/policies/{attestationType}`
- `/azure/attestation/policies/{attestationType}:reset`
- `/azure/attestation/attest/{attestationType}`

Important routing notes:
- this service is endpoint-based and does not use the ARM `/subscriptions/.../providers/...` envelope
- unlike some other Azure data-plane services in the repo, this API still uses an `api-version` query parameter
- `attestationType` is an enum-like path segment and should be handled case-sensitively in emitted fixtures while accepting case-insensitive routing if practical

## Contract Notes From Reviewed Method Pages

Shared traits across the reviewed pages:
- all reviewed methods require `api-version=2025-06-01`
- auth scope is `attest.azure.net/.default`
- success responses are immediate; no ARM-style long-running operation behavior is documented
- non-success responses use `Azure.Core.Foundations.ErrorResponse`
- most success payloads are JSON; signing certificates can use `application/jwk+json`

### Metadata Configuration

`Metadata Configuration - Get`
- `GET /.well-known/openid-configuration`
- response: `OpenIDConfigurationResponse`
- important modeled fields from the reviewed page:
  - `claims_supported[]`
  - `id_token_signing_alg_values_supported[]`
  - `issuer`
  - `jwks_uri`
  - `response_types_supported[]`
  - `revocation_endpoint`

Modeling implication:
- deterministic OpenID metadata is sufficient and should be implemented first because other certificate endpoints can reference it.

### Signing Certificates

`Signing Certificates - Get`
- `GET /certs`
- response: `JsonWebKeySet`
- media types:
  - `application/jwk+json`
  - `application/json`

Modeling implication:
- this endpoint should return a stable RFC7517 JWK set that matches the `jwks_uri` emitted by metadata.

### Policy Certificates

`Policy Certificates - Get`
- `GET /certificates`
- response: `PolicyCertificatesResponse`

`Policy Certificates - Add`
- `POST /certificates:add`
- request field:
  - `policyCertificateToAdd`
- response: `PolicyCertificatesModifyResponse`

`Policy Certificates - Remove`
- `POST /certificates:remove`
- request field:
  - `policyCertificateToRemove`
- response: `PolicyCertificatesModifyResponse`
- reviewed note:
  - the final policy management certificate cannot be removed

Modeling implications:
- maintain a small in-memory certificate set
- enforce the "last certificate cannot be removed" rule
- return deterministic token-shaped response values for modify operations

### Policy

`Policy - Get`
- `GET /policies/{attestationType}`
- response: `PolicyResponse`

`Policy - Set`
- `PUT /policies/{attestationType}`
- request body is plain text:
  - `newAttestationPolicy`
- response: `PolicyResponse`

`Policy - Reset`
- `POST /policies/{attestationType}:reset`
- request body is plain text:
  - `policyJws`
- response: `PolicyResponse`

Reviewed `AttestationType` values:
- `SgxEnclave`
- `OpenEnclave`
- `AzureGuest`
- `SevSnpVm`
- `Tpm`
- `TdxVm`

Modeling implications:
- store policy per attestation type
- return deterministic JWT-shaped `token` strings for `Get`, `Set`, and `Reset`
- treat request bodies as raw strings rather than JSON

### Attestation

All 6 attestation operations are `POST /attest/{attestationType}` and return signed attestation-result tokens.

`Attest Azure Guest`
- request type: `AttestAzureGuestRequest`
- reviewed field:
  - `attestationInfo`
- response: `SealedAttestationResponse`

`Attest Open Enclave`
- reviewed fields include:
  - `draftPolicyForAttestation`
  - `initTimeData`
  - `nonce`
- response: attestation token response

`Attest Sgx Enclave`
- reviewed fields include:
  - `draftPolicyForAttestation`
  - `initTimeData`
  - `nonce`
  - `quote`
  - `runtimeData`
- response: attestation token response

`Attest Sev Snp Vm`
- reviewed fields include:
  - `draftPolicyForAttestation`
  - `initTimeData`
  - `nonce`
  - `report`
- response: `AttestationResponse`

`Attest Tdx Vm`
- reviewed fields include:
  - `initTimeData`
  - `nonce`
- response: `AttestationResponse`

`Attest Tpm`
- reviewed field:
  - `data`
- response: `TpmAttestationResponse`

Common modeled response trait:
- attestation responses carry a `token` string in JWT format

Modeling implications:
- deterministic token emission is enough for the first useful emulator slice
- evidence bytes need not be cryptographically validated in the first implementation
- response payload should still differ slightly by attestation type so tests and examples can confirm route-specific behavior

## Suggested In-Memory Model

Recommended provider state on `Server`:
- `azureAttestationPolicies map[string]string`
- `azureAttestationPolicyCertificates []string`

Recommended deterministic fixtures:
- one OpenID metadata document
- one JWK set for signing certificates
- one initial policy-certificate set
- one default policy token per attestation type

Recommended helper patterns:
- `attestationType` validation helper
- shared `api-version` validation helper
- raw string body reader for policy set/reset
- shared token builder for policy and attestation responses
- shared `ErrorResponse` helper

## Implementation Stages

### Stage 0: Routing And Coverage Wiring

Deliverables:
- add a dedicated Azure Attestation router under `/azure/attestation`
- reserve service key `attestation`
- add service aliases in:
  - `scripts/azure-contract-coverage.py`
  - `scripts/azure-io-contract-coverage.py`
  - `scripts/azure-doc-contract-coverage.py`
- map the plan doc for doc coverage

Acceptance target:
- coverage scripts discover the service
- doc coverage resolves the new plan

### Stage 1: Discovery And Certificate Read Endpoints

Deliverables:
- implement:
  - `Metadata Configuration - Get`
  - `Signing Certificates - Get`
  - `Policy Certificates - Get`
- add focused tests for:
  - required `api-version`
  - metadata shape
  - JWK set shape
  - certificate list shape

Acceptance target:
- all read-only discovery routes return deterministic typed fixtures

### Stage 2: Policy Certificates Mutation

Deliverables:
- implement:
  - `Policy Certificates - Add`
  - `Policy Certificates - Remove`
- validate request shapes:
  - `policyCertificateToAdd`
  - `policyCertificateToRemove`
- enforce:
  - duplicate-add stability
  - final-certificate removal rejection

Acceptance target:
- policy certificate lifecycle is modeled and negative cases are contract-tested

### Stage 3: Policy Lifecycle

Deliverables:
- implement:
  - `Policy - Get`
  - `Policy - Set`
  - `Policy - Reset`
- persist policy per `attestationType`
- treat `Set` and `Reset` request bodies as plain text

Acceptance target:
- all policy routes return deterministic `PolicyResponse` token payloads
- no policy route returns `NotImplemented`

### Stage 4: Attestation Execution Endpoints

Deliverables:
- implement:
  - `Attest Azure Guest`
  - `Attest Open Enclave`
  - `Attest Sev Snp Vm`
  - `Attest Sgx Enclave`
  - `Attest Tdx Vm`
  - `Attest Tpm`
- validate minimum documented request fields for each method
- emit deterministic JWT-shaped result tokens that encode:
  - attestation type
  - nonce when provided
  - a stable pseudo-issuer

Acceptance target:
- all documented attestation endpoints return modeled success payloads
- missing required request fields produce stable validation errors

### Stage 5: Azure Go SDK Example

Deliverables:
- add `examples/azure/attestation-2025-06-01`
- prefer the official Azure Go SDK package verified during planning:
  - `github.com/Azure/azure-sdk-for-go/services/attestation/2020-10-01/attestation`
- example should exercise:
  - metadata configuration
  - signing certificates
  - policy get/set/reset
  - policy certificate get/add/remove
  - representative attestation calls supported by the verified SDK package

Implementation note:
- because I did not find a visible `2025-06-01` Go data-plane package, this stage likely requires using the older official package while keeping the endpoint/query behavior aligned to the `2025-06-01` docs. This is an inference from SDK discovery, not from the REST docs themselves.

Acceptance target:
- `go build ./examples/azure/attestation-2025-06-01`

### Stage 6: Validation And Coverage Closure

Deliverables:
- add focused provider tests for:
  - metadata and JWK read paths
  - policy certificates add/remove rules
  - policy get/set/reset
  - attestation request validation and success responses
- run:
  - `go test ./internal/server -run 'TestAzureAttestation' -count=1`
  - `python3 scripts/azure-contract-coverage.py --service attestation --require-service attestation --fail-on any`
  - `python3 scripts/azure-io-contract-coverage.py --service attestation --require-service attestation --fail-on any`
  - `python3 scripts/azure-doc-contract-coverage.py --service attestation --require-service attestation --fail-on any`
  - `python3 scripts/azure-endpoint-coverage.py --service attestation --require-service attestation --fail-on any`

Acceptance target:
- all new service endpoints are covered successfully
- no `NotImplemented` responses remain for `attestation`

## Coverage Script Updates To Plan For

Add service aliases for:
- `attestation`
- `azure_attestation`
- `attestation_2025_06_01`
- `rest_attestation_2025_06_01`
- `azure_attestation_2025_06_01`

Doc coverage mapping:
- `attestation -> attestation-2025-06-01-plan.md`

## Testing Strategy

Provider tests should cover:
- metadata endpoint output shape
- JWK signing certificate endpoint output shape
- policy certificate add/remove/get lifecycle
- "last certificate cannot be removed" validation
- policy get/set/reset across all attestation types
- one happy-path request for each `attest/*` route
- `400` for invalid `api-version`
- `400` for malformed or missing request bodies
- deterministic token shape in policy and attestation responses

Example validation target:
- example builds cleanly and uses an official Azure Go SDK package

## Risks And Decisions

### Data Plane Versus ARM

The reviewed docs are clearly data-plane endpoint routes, while Azure Attestation also has a separate ARM management surface.

Decision:
- scope this plan strictly to the reviewed data-plane API
- do not mix provider-management CRUD into this implementation

### SDK Surface Mismatch

The reviewed docs are `2025-06-01`, but the official Go SDK discovery I could verify publicly appears to expose older data-plane packages.

Decision:
- treat the REST docs as the contract source of truth
- use the verified official SDK package for the example if no newer package exists in the repo during implementation
- keep this limitation explicit in the implementation PR

### JWT And JWK Semantics

Real Azure Attestation performs cryptographic policy and evidence validation.

Decision:
- first implementation should emit deterministic JWT-shaped and JWK-shaped fixtures without full cryptographic semantics
- preserve route, field, and envelope compatibility as the main contract target

## Definition Of Done

The Azure Attestation data-plane emulation is complete when:

- all documented `2025-06-01` routes above are implemented under `/azure/attestation`
- metadata, certificates, policy, and attestation flows return deterministic typed responses
- the Azure Go SDK example builds successfully
- Azure contract, IO, doc, and aggregate endpoint coverage all pass for `attestation`
- no endpoint for the new service returns `NotImplemented`
