# Azure Provider Foundation Plan

## Objective

Build a maintainable Azure provider foundation in Stackyard so new Azure services can be added with predictable routing, auth behavior, test coverage, and docs/examples support.

## Current State (from repo)

- CLI/provider selection already supports `azure` in `stackyard start --providers ...`.
- Provider routing and auth-mode wiring already include Azure (`shared_key_or_sas`, `shared_key`, `sas`, `disabled`).
- Azure has foundational Blob routes under `/azure/{account}/...` for container/blob create, list, get, and head.
- Provider contract tests exist for the Blob baseline and auth enforcement.
- Gaps: no Azure service catalog, no Azure example suite, no Azure contract coverage scripts, and no explicit per-service scaffolding workflow.

## Non-Goals

- Do not change AWS behavior.
- Do not change GCP behavior.
- Do not attempt full ARM parity in the foundation phase.
- Do not block Azure progress on perfect cross-cloud abstractions.

## Design Principles

- Keep provider-agnostic entry points stable; isolate Azure-specific logic.
- Standardize service onboarding: router + store + contract tests + example + docs.
- Enforce contract quality with fast static gates and focused runtime tests.
- Ship incrementally; each stage must leave main branch usable.

## Target Foundation Layout

- `internal/server/provider_azure_router.go`
- `internal/server/provider_azure_auth.go` (or shared provider auth with Azure-specific helpers)
- `internal/server/provider_azure_blob.go`
- `internal/server/provider_azure_queue.go` (next service)
- `internal/server/provider_azure_keyvault.go` (next service)
- `internal/server/provider_azure_contract_test.go` (shared harness + service tests)
- `scripts/azure-contract-coverage.py`
- `scripts/azure-io-contract-coverage.py`
- `examples/azure/<service>/...`
- `docs/web/azure/...` and `docs/web/assets/azure-catalog.js`

## Staged Plan

### Stage 0: Baseline Lock (1-2 days)

- Freeze current Azure Blob/auth behavior with explicit regression tests.
- Add tests for XML shape, error payloads, and unsupported operation responses.
- Capture baseline routes and auth expectations in docs.

Exit criteria:

- Current Azure Blob behavior is fully regression-protected.

### Stage 1: Provider Management Spine (2-3 days)

- Introduce a single provider metadata registry used by CLI and server.
- Centralize:
  - supported providers,
  - default auth modes,
  - accepted auth mode values,
  - provider route prefixes.
- Remove duplicated provider constants/validation branches where practical.

Exit criteria:

- Adding/changing provider metadata requires one source-of-truth update.

### Stage 2: Azure Module Boundary (3-5 days)

- Extract Azure routing/handlers from mixed files into dedicated Azure files.
- Keep `handleProviderRouter` as thin dispatch only.
- Add internal helper layer for Azure path parsing and common response formatting.
- Keep behavior unchanged while improving maintainability.

Exit criteria:

- Azure code paths are isolated enough for independent service growth.

### Stage 3: Azure Contract Gates (2-4 days)

- Add `scripts/azure-contract-coverage.py` (validation paths, typed success fixtures, negative tests).
- Add `scripts/azure-io-contract-coverage.py` (input/output implementation + tests).
- Add `make` targets:
  - `coverage-azure-contracts`
  - `coverage-azure-io-contracts`
  - strict variants for selected services.

Exit criteria:

- Azure service quality is measurable with the same gate style as GCP.

### Stage 4: Azure Service Scaffolding Workflow (2-3 days)

- Define a standard checklist/template per Azure service:
  - router hook,
  - request parsing/validation,
  - state store model,
  - success/error response fixtures,
  - contract tests,
  - SDK example and compose file.
- Create starter stubs for next services:
  - Queue Storage,
  - Key Vault Secrets.

Exit criteria:

- New Azure services can be added through a repeatable, low-variance workflow.

### Stage 5: First Managed Service Wave (1-2 weeks)

- Harden Blob service beyond baseline:
  - metadata headers,
  - conditional headers (`If-Match`, `If-None-Match`),
  - list pagination markers.
- Implement minimal Queue Storage flow:
  - create queue,
  - enqueue/dequeue message,
  - delete message.
- Implement minimal Key Vault Secrets flow:
  - set/get/list secret versions.

Exit criteria:

- At least three Azure services have end-to-end runnable examples and contract gates.

### Stage 6: Docs, Catalog, and Examples (3-5 days)

- Add Azure docs portal pages (`docs/web/azure/...`) mirroring AWS/GCP structure.
- Add Azure service catalog asset and summary metrics.
- Add `examples/azure/...` and wire to `scripts/run-all-examples.sh`/`Makefile` flows.
- Update README provider support section and Azure run commands.

Exit criteria:

- Contributors can discover, run, and validate Azure services from docs alone.

### Stage 7: CI Integration and Release Criteria (2-3 days)

- Add Azure contract/IO coverage jobs to CI.
- Add Azure example smoke run target to CI (can start as non-blocking, then block).
- Define release gate:
  - tests pass,
  - no regression in AWS/GCP/OCI,
  - Azure strict services pass contract gates.

Exit criteria:

- Azure provider has enforceable quality gates in standard CI.

## Priority Service Order

1. Blob Storage (harden existing baseline)
2. Queue Storage
3. Key Vault Secrets
4. Azure Resource Manager lightweight operation stubs

## Risks and Mitigations

- Risk: monolithic server file slows Azure velocity.
  - Mitigation: early extraction into Azure-scoped files (Stage 2).
- Risk: inconsistent error semantics across Azure services.
  - Mitigation: shared Azure response helpers + fixture tests.
- Risk: docs/examples lag implementation.
  - Mitigation: make example/docs updates required in service checklist.

## Suggested Delivery Order

1. Stage 0 + Stage 1
2. Stage 2
3. Stage 3 + Stage 4
4. Stage 5
5. Stage 6 + Stage 7
