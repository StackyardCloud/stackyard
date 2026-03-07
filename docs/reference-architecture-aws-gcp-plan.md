# Reference Architecture Multi-Cloud Expansion Plan (AWS + GCP)

## Objective

Keep all existing AWS reference-architecture examples intact, and add equivalent GCP reference-architecture examples with the same user workflows and comparable outputs.

## Current State Review

The repository currently has two runnable reference-architecture examples, both AWS-only:

- `reference-architecture/data-pipeline/example`
- `reference-architecture/data-mesh/example`

Observed AWS service dependencies:

- Data Pipeline example backend:
  - S3, KMS, Secrets Manager, SQS, EventBridge, Step Functions
- Data Mesh example backend:
  - Kinesis, DynamoDB, SQS (DLQ), KMS

Stackyard already has GCP emulation support for the key equivalents needed for a first GCP pass:

- `storage`, `kms`, `secretmanager`, `pubsub`, `eventarc`, `workflows`, `workflow_executions`, `firestore`

## Non-Goals

- Do not remove or rename current AWS example paths in this phase.
- Do not change AWS runtime behavior or API responses beyond bug fixes.
- Do not attempt full production parity between AWS and GCP services in v1.

## Design Principles

- Preserve backward compatibility for existing AWS users.
- Add GCP examples side-by-side, not as replacements.
- Share domain logic where possible; isolate cloud SDK wiring in provider adapters.
- Keep API contracts stable between AWS and GCP variants so frontend flows remain consistent.

## Target Layout

Canonical provider-scoped layout:

- `reference-architecture/data-pipeline/aws/example` (AWS)
- `reference-architecture/data-pipeline/gcp/example` (GCP)
- `reference-architecture/data-mesh/aws/example` (AWS)
- `reference-architecture/data-mesh/gcp/example` (GCP)

Compatibility migration guidance:

- preserve existing `.../example` paths initially as wrappers/symlinks or documented redirects.
- update CI/scripts/docs to point at the canonical provider-scoped paths.

## Service Mapping (AWS -> GCP)

Data Pipeline:

- S3 -> Cloud Storage
- KMS -> Cloud KMS
- Secrets Manager -> Secret Manager
- SQS -> Pub/Sub (queue/topic semantics for demo)
- EventBridge -> Eventarc (or Pub/Sub topic trigger for v1)
- Step Functions -> Workflows + Workflow Executions

Data Mesh:

- Kinesis -> Pub/Sub (or Pub/Sub Lite in later stage)
- DynamoDB read model -> Firestore
- DynamoDB idempotency table -> Firestore processed-events collection
- SQS DLQ -> Pub/Sub dead-letter topic/subscription
- KMS -> Cloud KMS

## Staged Implementation Plan

### Stage 1: Baseline and Contracts

- Freeze current AWS behavior with smoke tests for:
  - Data Pipeline: bootstrap, ingest, run, summary
  - Data Mesh: bootstrap, publish, project, replay-dlq, product reads, summary
- Record request/response contract fixtures for both examples.
- Add a short compatibility matrix in each example README.

Exit criteria:

- AWS examples pass unchanged in CI and local compose runs.

### Stage 2: Shared Core Extraction

- Refactor each backend into:
  - domain workflows (provider-agnostic)
  - cloud adapter interface
  - AWS adapter (existing logic)
  - GCP adapter (new)
- Keep HTTP handlers and payload schemas unchanged.
- Add adapter-level unit tests with fake implementations.

Exit criteria:

- AWS backend uses the adapter layer and all existing smoke tests still pass.

### Stage 3: Data Pipeline GCP Example

- Create `reference-architecture/data-pipeline/gcp/example`:
  - backend using GCP adapter
  - frontend reusing current UI with provider-specific labels/env
  - docker compose with Stackyard started as `--providers aws,gcp`
- Implement GCP adapter operations:
  - bucket bootstrap (Storage)
  - per-tenant key bootstrap (KMS)
  - tenant secret bootstrap (Secret Manager)
  - ingestion queue publish (Pub/Sub)
  - orchestration execution (Workflows/Executions)
  - optional event emission path (Eventarc) if needed by current flow

Exit criteria:

- GCP data-pipeline compose completes end-to-end and mirrors AWS user journey.

### Stage 4: Data Mesh GCP Example

- Create `reference-architecture/data-mesh/gcp/example`.
- Implement GCP adapter operations:
  - domain bootstrap resources
  - event ingest to Pub/Sub
  - projector reads from subscription and writes Firestore read model
  - processed-events idempotency in Firestore
  - poison events to dead-letter topic and replay path
  - per-tenant KMS key association metadata

Exit criteria:

- GCP data-mesh compose completes end-to-end including DLQ replay and ABAC mode checks.

### Stage 5: Tooling and CI

- Add dedicated targets:
  - `make refarch-examples-aws`
  - `make refarch-examples-gcp`
  - `make refarch-examples-all`
- Add a script (similar to `scripts/run-all-examples.sh`) for reference-architecture compose runs.
- Wire jobs into CI without regressing existing `examples/*` runs.

Exit criteria:

- AWS and GCP reference-architecture compose suites run independently and in aggregate.

### Stage 6: Documentation and Developer Experience

- Update:
  - `README.md` reference-architecture section
  - `reference-architecture/*/*/example/README.md` with provider-specific run instructions
  - architecture docs with AWS/GCP mapping notes
- Add troubleshooting for endpoint overrides and provider headers.

Exit criteria:

- New contributors can run AWS and GCP reference architectures from docs alone.

## Validation Strategy

- Contract-level API smoke tests for each example workflow.
- Compose-based end-to-end checks for AWS and GCP variants.
- Adapter unit tests to verify deterministic behavior on validation and error mapping.
- Golden output checks for summary endpoints to ensure cross-provider parity where intended.

## Risks and Mitigations

- Risk: GCP service behavior differs from AWS assumptions in current workflow.
  - Mitigation: enforce workflow-level contracts and isolate provider-specific semantics in adapters.
- Risk: duplicated logic between AWS and GCP examples.
  - Mitigation: shared domain package + thin adapters.
- Risk: CI runtime increase from additional compose suites.
  - Mitigation: split AWS/GCP jobs and allow targeted local runs.

## Suggested Delivery Order

1. Data Pipeline GCP first (simpler to mirror current architecture).
2. Data Mesh GCP second (more stateful and replay-heavy).
3. CI + docs consolidation after both are stable.
