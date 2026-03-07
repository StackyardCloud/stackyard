# Data Mesh Reference Architecture Example

Canonical provider-scoped path for this AWS variant:

- `reference-architecture/data-mesh/aws/example`

This example implements a runnable local version of the `reference-architecture/data-mesh/docs` pack against Stackyard.

It focuses on the documented minimum viable environment behaviors:

- Kinesis-first event ingestion
- Projector flow into DynamoDB read models
- Idempotency table for processed events
- SQS DLQ + replay path
- API-backed data product with ABAC-style access modes (`full`, `masked`, `none`)
- Per-tenant KMS key creation and key association metadata

## Stack

- Stackyard (`http://localhost:4566`)
- Backend API (`http://localhost:8082`)

## Run

```bash
cd reference-architecture/data-mesh/example
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
curl -s -X POST http://localhost:8082/api/v1/domains/orders/bootstrap | jq
```

2. Publish a valid event.

```bash
curl -s -X POST http://localhost:8082/api/v1/domains/orders/tenants/tenant-123/events \
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
curl -s -X POST http://localhost:8082/api/v1/domains/orders/tenants/tenant-123/events \
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
curl -s -X POST http://localhost:8082/api/v1/domains/orders/project \
  -H 'Content-Type: application/json' \
  -d '{"limit": 50}' | jq
```

5. Query the API-backed product in different access modes.

Full:

```bash
curl -s http://localhost:8082/api/v1/domains/orders/products/orders/tenant-123 \
  -H 'X-Claim-Tenant-Id: tenant-123' \
  -H 'X-Claim-Scopes: read:full' | jq
```

Masked:

```bash
curl -s http://localhost:8082/api/v1/domains/orders/products/orders/tenant-123 \
  -H 'X-Claim-Tenant-Id: tenant-123' \
  -H 'X-Claim-Scopes: read:masked' | jq
```

None:

```bash
curl -s http://localhost:8082/api/v1/domains/orders/products/orders/tenant-123 \
  -H 'X-Claim-Tenant-Id: tenant-123' | jq
```

6. Replay DLQ messages and project again.

```bash
curl -s -X POST http://localhost:8082/api/v1/domains/orders/replay-dlq \
  -H 'Content-Type: application/json' \
  -d '{"max_messages": 10}' | jq

curl -s -X POST http://localhost:8082/api/v1/domains/orders/project \
  -H 'Content-Type: application/json' \
  -d '{"limit": 50}' | jq
```

7. Check runtime summary.

```bash
curl -s http://localhost:8082/api/v1/summary | jq
```

## Mapping To `docs/`

- `design.md`: Kinesis ingress, projector, DynamoDB read model, API product, KMS + DLQ are all represented.
- `event-contract.md`: event envelope shape and `schema_version` handling are implemented.
- `mve.md`: ingestion -> projection -> API access path, poison-to-DLQ, and replay workflow are covered.
- `runbook.md`: operations are reflected by explicit project/replay actions and metrics in `/api/v1/summary`.

## Notes / Local Limits

- This is a reference demo, not production IaC.
- Stream consumption is modeled by the backend projector endpoint (deterministic local workflow).
- Encryption in the read model is simulated ciphertext with tenant scoping; KMS keys are still created per tenant and tracked.
- Cross-account federation and full policy engine behavior are out of scope for this local runnable example.
