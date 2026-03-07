# Data Pipeline Reference Architecture Example (GCP)

Runnable GCP variant of the data-pipeline reference architecture against Stackyard.

## Stack

- Backend service: Go (`reference-architecture/data-pipeline/gcp/example/backend`)
- Frontend service: TypeScript + Vite (`reference-architecture/data-pipeline/gcp/example/frontend`)
- Emulation endpoint: Stackyard (`http://localhost:4566/gcp`)

## Emulated Services Used

- Cloud Storage: raw/curated/published/audit objects
- Cloud KMS: per-tenant key creation
- Secret Manager: per-tenant secret bootstrap
- Cloud Pub/Sub: ingestion topic/subscription and run events
- Workflows + Workflow Executions: pipeline run orchestration

## Run

```bash
cd reference-architecture/data-pipeline/gcp/example
docker compose up --build
```

Open:

- UI: `http://localhost:8180`
- Backend API: `http://localhost:8181`

## Frontend Workflow

1. Set a tenant id (for example `tenant-001`).
2. Click `Bootstrap Tenant`.
3. Click `Ingest Raw Batch`.
4. Click `Run Pipeline`.
5. Click `Refresh`.
