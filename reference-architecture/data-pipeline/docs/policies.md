## Checklist: “Must-have” policies, configs, and controls

### A) AWS account/service baseline

* [ ] **CloudTrail enabled** for the account (all regions), delivered to centralized log bucket
* [ ] **AWS Config enabled** with key resources recorded (S3, KMS, IAM, API Gateway, CloudFront, Aurora, etc.)
* [ ] Minimum IAM hygiene:

  * [ ] No long-lived access keys for humans (SSO only)
  * [ ] Break-glass role exists + monitored + MFA enforced
  * [ ] Permission boundaries for developers (if you allow IAM authoring)

### B) Identity & authorization

* [ ] **IAM Identity Center**

  * [ ] Separate roles: `PlatformAdmin`, `PlatformOperator`, `ReadOnlyAuditor`
  * [ ] Admin actions (rerun prod jobs, change retention, key ops) restricted to least privilege
* [ ] **Cognito**

  * [ ] Tenant identifier in token claims (e.g., `tenant_id`)
  * [ ] Short-lived tokens, refresh token policies set
  * [ ] MFA policy as appropriate (optional now, easy later)
* [ ] **API authorization rules**

  * [ ] Every API call enforces tenant scope server-side (never only in UI)
  * [ ] Separate admin endpoints gated to workforce roles only

### C) S3 data lake security (critical)

* [ ] **Block Public Access** enabled on all buckets
* [ ] **Bucket policy** denies any `PutObject` unless:

  * [ ] `aws:SecureTransport` = true (TLS required)
  * [ ] `s3:x-amz-server-side-encryption` = `aws:kms`
  * [ ] `s3:x-amz-server-side-encryption-aws-kms-key-id` matches the tenant CMK for the prefix
* [ ] **Prefix isolation**

  * [ ] Roles can only read/write `tenant=<t>/` prefixes they’re authorized for
  * [ ] No wildcard “list all” permissions for external tenants
* [ ] **Versioning** on critical buckets (especially curated/published/logs)
* [ ] **Lifecycle policies** (raw retention, quarantine retention, log retention → Glacier)

### D) KMS per-tenant key controls (your requirement)

* [ ] One **CMK per tenant** (tagged `tenant_id`)
* [ ] **Key policies** restrict:

  * [ ] Key administration to security admins only
  * [ ] Encrypt/Decrypt to platform service roles only (ingest/etl/api) scoped by tenant
* [ ] Consider **KMS Grants** for automated onboarding and tighter scoping
* [ ] **Multi-Region keys** if you require seamless decrypt in DR region (recommended with CRR)

### E) Aurora PostgreSQL security + DR

* [ ] Aurora in **private subnets**, not publicly accessible
* [ ] Encrypted storage (KMS), automated backups enabled
* [ ] Credentials in **Secrets Manager** with rotation
* [ ] **Cross-region read replica** provisioned
* [ ] Tested **promotion** runbook for DR (RTO/RPO documented)

### F) Orchestration and ETL guardrails

* [ ] Step Functions:

  * [ ] Retries/backoff configured
  * [ ] Dead-letter / failure notifications path (SNS/email/Slack optional)
* [ ] Glue jobs:

  * [ ] Write paths restricted to tenant prefixes
  * [ ] DQ checks produce artifacts + quarantine on failure
  * [ ] Iceberg compaction/maintenance jobs scheduled (if needed)

### G) Web edge and API protections

* [ ] CloudFront:

  * [ ] **Origin access control (OAC)** for S3 UI bucket
  * [ ] Default HTTPS, modern TLS
* [ ] WAF:

  * [ ] Managed rule groups enabled (baseline)
  * [ ] Rate limiting configured for API paths
* [ ] API Gateway:

  * [ ] Access logging enabled
  * [ ] Throttling limits set
  * [ ] Authorizers configured (Cognito for external; JWT/OIDC for workforce if used)

### H) Immutable audit logging (non-negotiable)

* [ ] Central log S3 bucket:

  * [ ] **Object Lock enabled** (Compliance mode), retention period set
  * [ ] Versioning enabled
  * [ ] Strict bucket policy prevents deletion/overwrite
* [ ] CloudTrail:

  * [ ] **S3 data events** enabled for curated/published buckets (and logs bucket access)
* [ ] Log integrity monitoring:

  * [ ] Alerts on CloudTrail stop/change events
  * [ ] Alerts on KMS key policy changes

### I) Disaster recovery essentials

* [ ] **S3 CRR** configured and tested (curated/published at minimum)
* [ ] DR region has:

  * [ ] IaC-deployed copies of Glue jobs, Step Functions, API/Lambda, CloudFront/WAF
  * [ ] KMS keys available (multi-region or replicated strategy)
  * [ ] Secrets replicated
* [ ] Regular restore drill: “query published in DR” + “run one pipeline in DR”