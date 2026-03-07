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
cd reference-architecture/data-mesh/gcp
docker compose up --build
```

## API Surface

- `POST /api/v1/domains/{domain}/bootstrap`
- `POST /api/v1/domains/{domain}/tenants/{tenantId}/events`
- `POST /api/v1/domains/{domain}/project`
- `POST /api/v1/domains/{domain}/replay-dlq`
- `GET /api/v1/domains/{domain}/products/orders/{tenantId}`
- `GET /api/v1/summary`

## End-to-End Walkthrough

1. Bootstrap a domain (`orders`).

```bash
curl -s -X POST http://localhost:8282/api/v1/domains/orders/bootstrap | jq
```

2. Publish a valid event.

```bash
curl -s -X POST http://localhost:8282/api/v1/domains/orders/tenants/tenant-123/events \
  -H 'Content-Type: application/json' \
  -d '{
    "event_type": "orders.order.created",
    "schema_version": "v1",
    "correlation_id": "trace-001",
    "payload": {
      "entity_id": "ord-1001",
      "amount": 42.75,
      "customer_email": "alice@example.com"
    }
  }' | jq
```

3. Publish a poison event (forced DLQ path).

```bash
curl -s -X POST http://localhost:8282/api/v1/domains/orders/tenants/tenant-123/events \
  -H 'Content-Type: application/json' \
  -d '{
    "event_type": "orders.order.created",
    "schema_version": "v1",
    "poison": true,
    "payload": {
      "entity_id": "ord-poison",
      "amount": 9.99,
      "customer_email": "poison@example.com"
    }
  }' | jq
```

4. Run the projector.

```bash
curl -s -X POST http://localhost:8282/api/v1/domains/orders/project \
  -H 'Content-Type: application/json' \
  -d '{"limit": 50}' | jq
```

5. Query the API-backed product in different access modes.

Full:

```bash
curl -s http://localhost:8282/api/v1/domains/orders/products/orders/tenant-123 \
  -H 'X-Claim-Tenant-Id: tenant-123' \
  -H 'X-Claim-Scopes: read:full' | jq
```

Masked:

```bash
curl -s http://localhost:8282/api/v1/domains/orders/products/orders/tenant-123 \
  -H 'X-Claim-Tenant-Id: tenant-123' \
  -H 'X-Claim-Scopes: read:masked' | jq
```

None:

```bash
curl -s http://localhost:8282/api/v1/domains/orders/products/orders/tenant-123 \
  -H 'X-Claim-Tenant-Id: tenant-123' | jq
```

6. Replay DLQ messages and project again.

```bash
curl -s -X POST http://localhost:8282/api/v1/domains/orders/replay-dlq \
  -H 'Content-Type: application/json' \
  -d '{"max_messages": 10}' | jq

curl -s -X POST http://localhost:8282/api/v1/domains/orders/project \
  -H 'Content-Type: application/json' \
  -d '{"limit": 50}' | jq
```

7. Check runtime summary.

```bash
curl -s http://localhost:8282/api/v1/summary | jq
```
