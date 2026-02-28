## Reference architecture

### 1) Data plane: S3 lakehouse with mixed formats + merges

**S3 buckets (or prefixes)**

* `raw/` → *mixed formats allowed* (CSV/JSON/Parquet, etc.), append-only
* `curated/` → **Iceberg tables (Parquet)** for merges/upserts + governed datasets
* `published/` → Iceberg tables optimized for consumption + offline ML datasets
* `quarantine/` → rejected data + DQ artifacts

**Table format**

* Curated/published: **Apache Iceberg** (ACID merges + snapshots/time travel)
* Raw: keep “as received” + optional “standardized raw parquet” for performance

**Query**

* **Athena** as default (works well for Iceberg + ad hoc)
* Optional: **Redshift Serverless** later if BI concurrency becomes painful

---

### 2) Control plane: orchestration + catalog + UI

**Orchestration**

* **EventBridge** (file arrival + schedules)
* **Step Functions** (pipeline orchestration: validate → transform → merge → publish)
* **SQS** (buffering/backpressure; decouple heavy stages)

**Metadata**

* **Glue Data Catalog** for technical metadata
* **Aurora PostgreSQL** for “business catalog” + workflow + lineage pointers + run summaries
* (Optional) **OpenSearch** for fast full-text dataset search

**Web UI**

* Frontend: **S3 + CloudFront + WAF** (OAC locked origin)
* Backend: **API Gateway + Lambda**
* UI focuses on:

  * Catalog browsing (Glue + Aurora metadata)
  * Pipeline monitoring/run control (Step Functions + run DB + CloudWatch links)

---

### 3) Identity: workforce + customer/partner (recommended split)

**Workforce**

* **IAM Identity Center** (SSO to AWS + platform admin/operator UI roles)

**Customer/Partner**

* **Cognito User Pools** (or federated IdPs) for external identities
* API layer enforces **tenant isolation** (tenant_id in JWT claims)

This gives clean separation: AWS-account access for workforce, app-first auth for external users.

---

## Per-tenant encryption keys (your requirement)

You want **each customer/partner to have their own encryption key**. On AWS, the most scalable, operable pattern is:

### Tenant KMS key model (recommended)

* Create **one KMS CMK per tenant** (in the platform account), tagged with `tenant_id`.
* Store tenant data under `s3://lake-*/tenant=<tenant_id>/...`
* Enforce **SSE-KMS with that tenant’s CMK** for objects in that tenant prefix.

**How to enforce**

1. **S3 bucket policy** denies `PutObject` unless:

   * `x-amz-server-side-encryption` = `aws:kms`
   * `x-amz-server-side-encryption-aws-kms-key-id` matches the tenant key
   * and the caller principal is authorized for that tenant

2. **KMS key policy + grants**:

   * Only the platform ingestion/processing roles can `Encrypt/Decrypt` for that tenant key.
   * Optionally use **KMS Grants** dynamically when onboarding a tenant (nice for automation).

**Important operational note**

* KMS has per-account limits (number of keys, request rates). If you expect *many* tenants (hundreds/thousands), you may need to:

  * request quota increases, and/or
  * use the “key-per-tenant group” fallback (below).

### Fallback when tenant count is huge (still meets intent)

If you anticipate **very large tenant counts**, a pragmatic alternative:

* **Key-per-tenant-segment** (e.g., per 50 tenants) + strict prefix policy + app-layer isolation
* Or use **envelope encryption at the app layer** with a tenant DEK wrapped by a smaller set of CMKs
  But: if your requirement is explicitly “a key per tenant,” start with CMK-per-tenant and monitor scale.

---

## Golden path pipeline (batch, TB/day, with merges)

### Stage 0 — Ingest (raw)

1. Producer drops to `raw/tenant=<t>/source=<s>/date=YYYY-MM-DD/...`
2. S3 event → **EventBridge** → **Step Functions** execution created

### Stage 1 — Validate (cheap gate)

* **Lambda** checks:

  * expected naming convention
  * file type allowed
  * checksum/size sanity
  * basic schema sniff (optional)

Failures go to `quarantine/` + an event emitted for UI visibility.

### Stage 2 — Standardize (optional but recommended)

* **Glue Spark** job converts raw formats to standardized Parquet (still “raw standardized”) to reduce downstream pain.
* Write to `raw_standardized/` (still tenant-prefixed, tenant-key encrypted)

### Stage 3 — Curate + Merge (Iceberg)

* Glue/EMR Spark reads standardized raw, applies transforms, then:

  * Writes to curated **Iceberg** tables
  * Performs **MERGE INTO** for upserts (keys defined per dataset)
* Record run metadata + inputs/outputs in Aurora/DynamoDB

### Stage 4 — Publish (consumption-ready)

* Build published Iceberg tables optimized for consumers (partitioning, compaction)
* Emit catalog updates (Glue crawlers only if needed; better to register tables directly)

### Stage 5 — Offline ML datasets

* Materialize training datasets to `published/ml/tenant=<t>/dataset=<d>/...` as Iceberg or Parquet (your call)
* Track dataset versions via Iceberg snapshots + metadata in Aurora

---

## Catalog + monitoring in the UI (exactly aligned to your scope)

### Catalog view

* Dataset list (tenant-filtered for external users; global for workforce)
* Table schema + schema history (Iceberg)
* Owners, tags, “DQ score,” retention hints
* Links to sample queries (Athena) and partitions/snapshots

### Pipeline monitoring view

* Pipeline definitions and schedules
* Run history with:

  * status, duration, row counts
  * DQ results
  * direct links to CloudWatch logs + Step Functions execution

---

## Immutable audit logs (kept, as requested)

Even without compliance requirements, do this:

* **Org-wide CloudTrail** (or account-wide) → centralized **S3 log bucket**
* Central log bucket has:

  * **S3 Object Lock (Compliance mode)** + Versioning
  * lifecycle to Glacier
* CloudWatch Logs streamed to the same bucket via **Firehose**
* Include: CloudTrail, Config, VPC Flow Logs, WAF logs, API Gateway access logs, Step Functions/Glue logs

This gives you tamper-resistant auditability.

---

## DR-only focus (recommended posture)

* **S3 Cross-Region Replication** for curated/published (+ raw if you want reprocessing in DR)
* **Aurora PostgreSQL**: cross-region read replica + promotion runbook (or Aurora Global DB if RTO/RPO must be tight later)
* **Secrets Manager** replicated
* Deploy control plane stack in DR region (warm standby) via IaC
