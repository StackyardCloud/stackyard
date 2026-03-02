Below is the full Markdown version of the PDF architecture pack.

You can copy this directly into your repo as:

```
/docs/event-driven-serverless-data-mesh-aws.md
```

---

# Event-Driven Serverless Data Mesh (Kinesis-First)

## AWS Reference Architecture Pack

**Date:** March 2, 2026
**Audience:** Principal Cloud Architects, Platform Engineering, Security Engineering, Domain Teams

---

# 1. Executive Summary

This architecture defines a **Kinesis-first, event-driven, serverless data mesh** on AWS with:

* One AWS account per domain
* API-backed data products
* Seconds-level latency
* ≤500 events/sec at ~2KB per event
* Eventual ordering
* Per-tenant encryption keys
* PII controls and masking
* No DR requirement (yet)

Each domain publishes:

1. A canonical event stream (Amazon Kinesis Data Streams)
2. One or more API-backed data products (materialized read models)

Consumers can:

* Subscribe to events cross-account, or
* Query current state via API

---

# 2. Architecture Overview

## 2.1 Account Model

### Shared Platform Account

Provides centralized guardrails:

* AWS Organizations / Control Tower
* CloudTrail (org trail → S3 log archive)
* Security Hub, GuardDuty, Config
* IAM Identity Center (workforce)
* CloudWatch cross-account dashboards

---

### Domain Account (One per Domain)

Each domain owns:

| Layer               | AWS Services                                     |
| ------------------- | ------------------------------------------------ |
| Ingestion           | Kinesis Data Streams                             |
| Processing          | Lambda (Kinesis consumer)                        |
| Workflow (optional) | Step Functions                                   |
| Read Models         | DynamoDB                                         |
| API Surface         | API Gateway + Lambda                             |
| Identity            | Cognito (partners) + Identity Center (workforce) |
| Encryption          | KMS                                              |
| Failure Handling    | SQS DLQ                                          |
| Observability       | CloudWatch + X-Ray                               |
| Audit               | CloudTrail                                       |

---

## 2.2 Core Data Flow

```
Producers
   ↓
Kinesis Data Streams (Domain-Ingress)
   ↓
Lambda Projectors (Idempotent)
   ↓
DynamoDB Read Models
   ↓
API Gateway + Lambda
```

---

# 3. Event Contract

## Standard Event Envelope

```json
{
  "event_id": "uuid",
  "event_type": "domain.entity.action",
  "schema_version": "v1",
  "occurred_at": "RFC3339 timestamp",
  "producer": {
    "domain": "orders",
    "service": "orders-api"
  },
  "tenant_id": "tenant-123",
  "correlation_id": "trace-id",
  "payload": {}
}
```

### Partition Key Strategy

Preferred:

```
tenant_id
```

For hot tenants:

```
tenant_id#entity_id
```

Total ordering is NOT guaranteed or required.

---

# 4. Data Product Pattern (API-Backed)

Each domain publishes one or more **materialized read models**.

## CQRS Model

### Write Path

* Events land in Kinesis
* Lambda projector validates schema
* Idempotency check
* Update DynamoDB read model

### Read Path

* API Gateway → Lambda
* Authorization + masking
* Query DynamoDB
* Return filtered response

---

# 5. Security Architecture

## 5.1 Identity

| Identity Type        | Service             |
| -------------------- | ------------------- |
| Workforce            | IAM Identity Center |
| Partners / Customers | Amazon Cognito      |

---

## 5.2 Authorization (ABAC)

Authorization is Attribute-Based Access Control.

Claims used:

* `tenant_id`
* `scopes` (read:full, read:masked)
* `role`
* optional `purpose_of_use`

Enforced in:

1. API authorizer
2. Lambda handler (defense in depth)

---

## 5.3 Per-Tenant Encryption Strategy

Sensitive fields are encrypted at rest in DynamoDB.

### Recommended Pattern: Envelope Encryption

### Write Path

1. Determine `tenant_id`
2. Resolve KMS key (e.g., `alias/tenant/tenant-123`)
3. Call `GenerateDataKey`
4. Encrypt sensitive fields
5. Store:

   * Ciphertext
   * Encrypted Data Key (EDK)

---

### Read Path

| Access Level | Behavior                        |
| ------------ | ------------------------------- |
| Full         | Decrypt EDK → decrypt fields    |
| Masked       | Do NOT decrypt → return partial |
| None         | Omit fields                     |

---

### Key Strategy Options

**Option A — CMK per tenant**

* Strong isolation
* Operational overhead

**Option B — Domain CMK + derived DEKs**

* Scales better
* Still cryptographically isolated

Recommended for scale: Option B.

---

# 6. Idempotency Pattern

Each projector Lambda must be idempotent.

## Process

* Every event contains `event_id`
* Maintain DynamoDB table:

```
ProcessedEvents
  PK: event_id
  TTL: 7–30 days
```

* Use conditional writes
* Reject duplicates safely

---

# 7. Failure Handling

## DLQ Pattern

* Lambda failures → SQS DLQ
* Alarm on DLQ depth
* Replay Lambda re-injects events

---

# 8. Observability & SLOs

## Required Metrics

| Metric              | Purpose           |
| ------------------- | ----------------- |
| Kinesis IteratorAge | Detect lag        |
| Lambda Errors       | Consumer health   |
| Lambda Duration     | Timeout risk      |
| DLQ Depth           | Failure detection |
| API p95 latency     | Product SLO       |
| KMS Errors          | Encryption issues |

---

# 9. Minimum Viable Environment (MVE)

## Smallest Production-Ready Stack

| Layer         | Choice                         |
| ------------- | ------------------------------ |
| Accounts      | 1 platform + 2 domain accounts |
| Streams       | 1 per domain (1–2 shards)      |
| Projectors    | 1 Lambda per read model        |
| Data Store    | DynamoDB                       |
| API           | API Gateway + Lambda           |
| Auth          | Cognito + Identity Center      |
| Encryption    | KMS + envelope encryption      |
| Failure       | SQS DLQ                        |
| Observability | CloudWatch dashboards          |
| Audit         | CloudTrail org trail           |

---

## MVE Acceptance Criteria

* Event projected into DynamoDB within seconds
* API supports full / masked / none modes
* DLQ catches poison events
* Replay tool works
* Metrics visible in dashboard
* KMS usage logged in CloudTrail

---

# 10. Production Readiness Checklist

## Infrastructure

* IaC modules per domain
* Parameterized shard count
* Environment isolation (dev/stage/prod)

---

## Contracts

* Versioned schema
* Backward-compatible evolution
* Validation in projector

---

## Security

* Least privilege IAM
* KMS key policies validated
* Masking regression tests
* PII inventory documented

---

## Operations

* Runbook documented
* Replay procedure tested
* On-call ownership defined
* Alarms configured

---

# 11. Runbook Excerpts

## Consumer Lag

**Signal:** Increasing `IteratorAgeMilliseconds`

**Actions:**

* Check shard hot keys
* Increase shard count
* Adjust Lambda parallelization

---

## DLQ Messages

**Signal:** SQS depth > 0

**Actions:**

* Inspect error
* Patch projector
* Replay messages

---

## KMS Errors

**Signal:** Access denied / throttling

**Actions:**

* Validate key policy
* Check quotas
* Consider reducing CMK sprawl

---

# 12. IaC Module Structure (Recommended)

```
modules/
  domain-mesh/
    kinesis_stream/
    lambda_projector/
    dynamodb_read_model/
    api_product/
    dlq_and_replay/
    observability/
    kms_tenant_keys/

environments/
  platform/
  domains/
    orders/
    billing/
```

---

# 13. Standard Naming & Tagging

**Resource Name Pattern**

```
{env}-{domain}-{component}
```

Example:

```
prod-orders-stream
```

**Required Tags**

* env
* domain
* owner
* data_sensitivity
* cost_center

---

# 14. Future Expansion (When Needed)

* Multi-region active-active
* Analytics products (S3 + Lake Formation)
* DataZone for business discovery
* Cross-region replication
* Dedicated event egress streams

---

# End State Characteristics

When implemented correctly, this architecture provides:

* Domain autonomy
* Strong tenant isolation
* Cryptographic data protection
* Real-time consumption
* Operational resilience
* Clear ownership boundaries
* Low operational overhead