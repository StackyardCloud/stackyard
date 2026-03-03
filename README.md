# Stackyard

Stackyard is a Go-based local AWS emulator focused on fast startup, deterministic behavior, and contributor-friendly service evolution.

This repository is designed as an open reference implementation for building and testing cloud-integrated systems locally without depending on live AWS infrastructure.

[https://stackyard.cloud](https://stackyard.cloud)

## Project Goals

- Emulate AWS APIs with predictable local behavior
- Keep service implementations readable and independently evolvable
- Support SDK/CLI compatibility testing through staged service coverage
- Make it easy for contributors to add or harden service emulation

## Current Scope

Stackyard includes many service emulations with staged behavior and tests. Coverage and implementation depth vary by service.

For the most up-to-date catalog and operation/type coverage, use:

- Interactive docs: `docs/index.html`
- Coverage runner: `scripts/awscli-endpoint-coverage.py`

## Architecture Principles

- In-memory-first service stores for speed and repeatability
- Explicit request parsing and error envelopes per service protocol
- SigV4-aware routing/validation where required
- Staged implementation strategy to grow compatibility safely

## Repository Layout

- Server entrypoint: `cmd/stackyard`
- Service handlers/stores/tests: `internal/server`
- Runnable examples (basic + advanced): `examples/aws/`
- Endpoint coverage tooling: `scripts/awscli-endpoint-coverage.py`
- Generated docs site: `docs/index.html`
- Multi-cloud foundation notes: `docs/multi-cloud-foundation.md`
- GCP Generative Language staged plan: `docs/gcp-generativelanguage-apiv1-plan.md`
- Reference architecture examples: `reference-architecture/`

## Quickstart

Install the CLI:

```bash
go install ./cmd/stackyard
```

If `stackyard` is not found after install, add your Go bin path to `PATH`.

For the current shell session:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
hash -r
which stackyard
stackyard help
```

For zsh (persist across terminal sessions):

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

Run locally:

```bash
# starts in background (daemon mode)
stackyard start --providers aws
stackyard start --providers aws,gcp,azure,oci
stackyard start --providers aws --port 4566
stackyard start --providers aws --log-level debug

# stop the background process
stackyard stop
```

Provider support status:

- `aws`: broad emulation coverage
- `gcp`, `azure`, `oci`: foundational routing plus object storage SDK paths enabled; broader service emulation in progress

Foundational object storage routes for non-AWS providers:

- `gcp`: JSON API bucket/object create, list, and get via `/gcp/storage/v1/*` and `/gcp/upload/storage/v1/*`
- `azure`: Blob container/blob create, list, get, and head via `/azure/{account}/*`
- `oci`: Object Storage namespace, bucket, and object lifecycle via `/oci/n/*`

Planned non-AWS expansion:

- `gcp generativelanguage/apiv1`: staged implementation for model discovery and generative RPCs (`docs/gcp-generativelanguage-apiv1-plan.md`)

SDK endpoint override base URLs:

- GCP Cloud Storage SDK: `http://localhost:4566/gcp`
- Azure Blob SDK: `http://localhost:4566/azure/<storage-account>`
- OCI Object Storage SDK: `http://localhost:4566/oci`

Provider auth modes (configurable via CLI flags or env vars):

- GCP: `emulator` (default), `bearer_tolerant`, `bearer_required`
- Azure: `shared_key_or_sas` (default), `shared_key`, `sas`, `disabled`
- OCI: `signature` (default), `disabled`

Run with Docker Compose:

```bash
docker compose up --build
```

Default AWS-style credentials for local clients:

```bash
export AWS_ACCESS_KEY_ID=stackyard
export AWS_SECRET_ACCESS_KEY=stackyard
export AWS_REGION=us-east-1
```

Health check:

```bash
curl http://localhost:4566/_stackyard/health
```

## Development Workflow

Common targets:

```bash
make fmt
make tidy
make test
make ci
```

Additional automation:

- Run all Docker examples: `make examples-docker`
- Run endpoint coverage: `make coverage-all`

## Examples

Each service typically includes:

- `*-basic`: minimal happy-path usage
- `*-advanced`: broader lifecycle/workflow coverage

See `examples/aws/` and `examples/gcp/` for service-specific Dockerfiles and compose files.

New GCP Generative Language example scaffolds:

- `examples/gcp/generativelanguage-apiv1/generativelanguage-apiv1-basic`
- `examples/gcp/generativelanguage-apiv1/generativelanguage-apiv1-advanced`

## Testing and Compatibility

Stackyard validates behavior through:

- service-level staged Go tests in `internal/server/*_test.go`
- smoke scripts in `scripts/`
- contract-style endpoint coverage checks using AWS CLI skeletons
- provider contract checks for GCP/Azure/OCI foundation routes (`make provider-contracts`)

This project aims for practical local compatibility, not complete protocol parity for every AWS feature.

## Contribution Model

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full contribution workflow.

Contributions are welcome. A good contribution generally includes:

- clear service/operation scope
- implementation in handler + store layers
- staged tests for new behavior and regressions
- docs updates where user-facing behavior changes
- example updates when the service supports meaningful workflows

Suggested PR structure:

1. Service operation/type catalog update
2. Stage implementation (incremental behavior)
3. Tests for unknown action, known action, and staged lifecycle
4. Documentation and example updates

## Extending a Service

Typical pattern for adding or expanding emulation:

1. Add operation/type catalogs
2. Add protocol-aware candidate/router parsing
3. Add in-memory store behavior by stage
4. Add stage tests
5. Add basic/advanced examples
6. Add service entry to coverage script
7. Update docs index

## Reference Architecture Work

See `reference-architecture/` for architecture-driven examples that integrate multiple emulated services (Go backends + TypeScript frontends).

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting and disclosure guidance.

## Notes

- Stackyard is intended for local development, CI, and architecture prototyping.
- It is not intended as a production AWS replacement.
- Behavior can intentionally differ from AWS where simplification improves local usability, unless compatibility is explicitly required by tests.
