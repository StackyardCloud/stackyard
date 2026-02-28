## Minimum Viable Enterprise

### Core security & identity

* **AWS Identity and Access Management (IAM)** – roles/policies, least privilege, service roles
* **AWS IAM Identity Center** – workforce SSO (admins/operators)
* **Amazon Cognito** – customer/partner identity (multi-tenant JWTs)
* **AWS Key Management Service (KMS)** – per-tenant CMKs + encryption everywhere
* **AWS Secrets Manager** – DB credentials + rotation

### Data lake, processing, and query

* **Amazon S3** – raw/curated/published/quarantine + **Object Lock** for immutable logs + **CRR** for DR
* **AWS Glue** – Spark ETL + Iceberg writes/merges
* **AWS Glue Data Catalog** – table metadata/catalog
* **Amazon Athena** – serverless querying over Iceberg

### Orchestration & eventing

* **Amazon EventBridge** – schedules + file-arrival triggers
* **AWS Step Functions** – pipeline orchestration (retries, branching)
* **AWS Lambda** – lightweight validation + API backend

### Metadata store for UI (business catalog + runs)

* **Amazon Aurora PostgreSQL** – catalog metadata, lineage pointers, pipeline run summaries + **cross-region replica** for DR

### Web UI delivery & API

* **Amazon CloudFront** – UI CDN + edge security posture
* **AWS WAF** – protect UI/API (rate limiting, OWASP rules)
* **Amazon API Gateway** – public API façade for UI backend
* *(UI hosted on S3 — already included above)*

### Audit logging, monitoring, and configuration tracking (with immutability)

* **AWS CloudTrail** – API audit logs (incl. S3 data events for critical buckets)
* **Amazon CloudWatch** – metrics/logs/alarms (Glue, Lambda, API, Step Functions)
* **AWS Config** – resource configuration history / drift tracking

### Networking baseline

* **Amazon VPC** – private subnets for Aurora (and optionally for Glue via VPC access)
* *(Optional but recommended when you harden later: VPC endpoints for S3/KMS/Secrets/CloudWatch)*

---

## Explicitly excluded (still “enterprise”, but not minimum)

These are valuable but not required for the MVE:

* **SQS** (buffering/backpressure) – add when pipelines need decoupling at scale
* **Amazon EMR** – add for heavy Spark tuning or custom runtimes
* **Amazon OpenSearch Service** – add for fast full-text catalog search
* **Amazon Redshift Serverless** – add for BI concurrency/performance
* **Kinesis Data Firehose** – add if you want automated CloudWatch log streaming into immutable S3
* **DataSync / Transfer Family** – add only if you need those ingestion methods
* **Organizations/Control Tower/SCPs** – add when you decide to formalize multi-account governance

---

## What this MVE stack guarantees (mapped to your requirements)

* **TB/day batch + merges:** S3 + Glue (Spark) + Iceberg + Athena
* **Apps + offline ML consumption:** published Iceberg tables in S3 + Athena access (and/or exports)
* **Catalog + monitoring UI:** Glue Data Catalog + Aurora + API Gateway/Lambda + CloudFront
* **Workforce + customer identity:** Identity Center + Cognito
* **Per-tenant encryption keys:** KMS CMK per tenant + enforced by S3 bucket policy/key policy
* **Immutable audit logs:** CloudTrail → S3 Object Lock bucket (WORM)
* **DR focus:** S3 CRR + Aurora cross-region replica (+ replicated secrets/keys as needed)