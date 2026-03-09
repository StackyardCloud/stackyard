# Azure Service Scaffolding Checklist

Use this checklist when adding a new Azure service in Stackyard.

## 1) Router Hook

- Add or update `handleAzure<Service>Router` in `internal/server/provider_azure_<service>.go`.
- Wire it in Azure dispatch order from `internal/server/http.go` before generic blob fallback if needed.

## 2) Request Validation

- Validate required path segments, query params, and body shape.
- Return deterministic errors (`InvalidRequest`, `NotFound`, `NotImplemented`) with stable status codes.

## 3) State Model

- Store service state under `Server` with mutex protection.
- Keep data model deterministic (stable IDs/versioning where practical).

## 4) Typed Responses

- Return explicit response envelopes (JSON or XML as service requires).
- Set relevant headers (`Content-Type`, `ETag`, metadata headers, etc.).

## 5) Contract Tests

- Add positive lifecycle tests.
- Add negative tests for validation/auth/not-found/unsupported operations.
- Ensure coverage scripts detect validation + fixture + negative test signals.

## 6) Example

- Add `examples/azure/<service>/main.go`.
- Add `examples/azure/<service>/Dockerfile`.
- Add `examples/azure/<service>/docker-compose.yml`.

## 7) Docs & Catalog

- Add/update service entry in `docs/web/assets/azure-catalog.js`.
- Update Azure docs pages when category counts or capability claims change.

## 8) Quality Gates

- Run:
  - `go test ./internal/server -run Azure -count=1`
  - `python3 scripts/azure-contract-coverage.py`
  - `python3 scripts/azure-io-contract-coverage.py`
  - `python3 scripts/azure-doc-contract-coverage.py`
  - `make coverage-azure-doc-contracts-strict`
