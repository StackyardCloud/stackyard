# Data Mesh Reference Architecture Example (GCP)

Runnable GCP variant of the data-mesh reference architecture against Stackyard.

## Stack

- Stackyard (`http://localhost:4566/gcp`)
- Backend API (`http://localhost:8282`)

## Emulated Services Used

- Cloud Pub/Sub: ingress topic and DLQ topic workflow
- Cloud KMS: per-tenant key bootstrap metadata
- Cloud Firestore: projected read-model and processed-event documents

## Run

```bash
cd reference-architecture/data-mesh/gcp/example
docker compose up --build
```

## API Surface

- `POST /api/v1/domains/{domain}/bootstrap`
- `POST /api/v1/domains/{domain}/tenants/{tenantId}/events`
- `POST /api/v1/domains/{domain}/project`
- `POST /api/v1/domains/{domain}/replay-dlq`
- `GET /api/v1/domains/{domain}/products/orders/{tenantId}`
- `GET /api/v1/summary`
