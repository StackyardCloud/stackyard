# Event Contract (Domain Standard)

## Envelope (stable)
```json
{
  "event_id": "uuid",
  "event_type": "domain.entity.action",
  "schema_version": "v1",
  "occurred_at": "RFC3339 timestamp",
  "producer": {"domain": "orders", "service": "orders-api"},
  "tenant_id": "tenant-123",
  "correlation_id": "trace-id",
  "payload": { }
}
```

## Evolution rules
- Prefer additive changes to payload.
- Never change meaning of an existing field.
- Deprecate fields with an overlap window (keep producing old + new).
- Validate schema_version in every consumer.
- Maintain contract tests shared across domains.
