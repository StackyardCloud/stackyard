# Multi-Cloud Emulation Foundation

This document tracks the foundational non-AWS emulation surfaces in Stackyard and how to validate them.

## Enabled Providers

Start Stackyard with multiple providers:

```bash
stackyard start --providers aws,gcp,azure,oci
```

Or via Docker Compose:

```bash
docker compose up --build
```

## Foundation Scope

Current non-AWS foundational vertical slices focus on object storage compatibility.

- `gcp`: bucket create/list, object upload/list/get (`/gcp/storage/v1/*`)
- `azure`: container create/list/get/head, blob put/get/head (`/azure/{account}/*`)
- `oci`: namespace get, bucket create/list, object put/get (`/oci/n/*`)

## SDK Endpoint Overrides

Use SDK endpoint/base URL overrides:

- GCP Cloud Storage: `http://localhost:4566/gcp`
- Azure Blob Storage: `http://localhost:4566/azure/<storage-account>`
- OCI Object Storage: `http://localhost:4566/oci`

## Auth Modes

Provider-specific request auth validators are configurable:

- GCP: `emulator` (default), `bearer_tolerant`, `bearer_required`
- Azure: `shared_key_or_sas` (default), `shared_key`, `sas`, `disabled`
- OCI: `signature` (default), `disabled`

## Contract Gate

Run provider contract tests:

```bash
make provider-contracts
```

The CI workflow also executes this contract suite.
