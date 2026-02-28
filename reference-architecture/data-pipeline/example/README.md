# Data Pipeline Reference Architecture Example

This runnable example implements a reference architecture control plane using Stackyard-emulated AWS services.

## Stack

- Backend service: Go (`reference-architecture/data-pipeline/example/backend`)
- Frontend service: TypeScript + Vite (`reference-architecture/data-pipeline/example/frontend`)
- Emulation endpoint: Stackyard (`http://localhost:4566`)

## Emulated Services Used

- Amazon S3: raw/curated/published/audit objects
- AWS KMS: per-tenant key creation
- AWS Secrets Manager: per-tenant secret bootstrap
- Amazon SQS: ingestion queue
- Amazon EventBridge: tenant event bus, rule, target, and events
- AWS Step Functions: state machine + execution per pipeline run

## Run

```bash
cd reference-architecture/data-pipeline/example
docker compose up --build
```

Open:

- UI: `http://localhost:8080`
- Backend API: `http://localhost:8081`

## Frontend Workflow

1. Set a tenant id (e.g. `tenant-001`).
2. Click `Bootstrap Tenant`.
3. Click `Ingest Raw Batch`.
4. Click `Run Pipeline`.
5. Click `Refresh` to reload summary and tenant details.

## API Endpoints

- `GET /api/v1/summary`
- `GET /api/v1/tenants/{tenantId}`
- `POST /api/v1/tenants/{tenantId}/bootstrap`
- `POST /api/v1/tenants/{tenantId}/ingest`
- `POST /api/v1/tenants/{tenantId}/run`

## Notes

- This is a reference example, not production IaC.
- Data and pipeline state are kept in-memory in the backend service for demo purposes.
- Bucket names are fixed by default and configurable via env vars in `docker-compose.yml`.
