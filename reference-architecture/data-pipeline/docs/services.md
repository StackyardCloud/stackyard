# 1️⃣ Storage & Data Lake

* **Amazon S3**
  Raw, curated (Iceberg), published, ML datasets, quarantine zones
  Cross-Region Replication (DR)
  Object Lock (immutable audit logs)

* **AWS Key Management Service (KMS)**
  Per-tenant CMKs
  Multi-Region keys (for DR)
  Encryption for S3, Aurora, logs

---

# 2️⃣ Data Processing & ETL

* **AWS Glue**
  Spark ETL jobs
  Iceberg table writes
  Data standardization

* **Amazon EMR** (optional heavy Spark workloads)

* **Amazon Athena**
  Query Iceberg tables
  Ad-hoc SQL access

* **Amazon Redshift** (optional – Redshift Serverless for BI acceleration)

---

# 3️⃣ Orchestration & Eventing

* **AWS Step Functions**
  Batch pipeline orchestration

* **Amazon EventBridge**
  File arrival triggers
  Scheduled runs

* **Amazon Simple Queue Service (SQS)**
  Stage buffering & backpressure

* **AWS Lambda**
  Validation logic
  API backend
  Lightweight transforms

---

# 4️⃣ Metadata, Catalog & Databases

* **AWS Glue Data Catalog**
  Table schemas
  Iceberg metadata registration

* **Amazon Aurora (PostgreSQL)**
  Business catalog
  Workflow metadata
  Lineage
  UI query backend
  Cross-region replica for DR

* **Amazon DynamoDB**
  Pipeline run metadata
  Execution state (optional)

* **Amazon OpenSearch Service** (optional for fast dataset search)

---

# 5️⃣ Identity & Access

* **AWS IAM Identity Center**
  Workforce authentication
  Admin/operator roles

* **Amazon Cognito**
  Customer/partner identity
  Multi-tenant JWT tokens

* **AWS Identity and Access Management**
  Roles, policies, least privilege
  Tenant isolation enforcement

---

# 6️⃣ Web UI & API Layer

* **Amazon CloudFront**
  UI delivery
  Edge protection

* **AWS WAF**
  Web security rules
  Rate limiting

* **Amazon API Gateway**
  Backend API for UI

---

# 7️⃣ Logging, Monitoring & Audit (Immutable)

* **AWS CloudTrail**
  API activity logs
  S3 data events

* **Amazon CloudWatch**
  Logs
  Metrics
  Alarms
  Log Insights

* **AWS Config**
  Resource configuration tracking

* **Amazon Kinesis Data Firehose**
  Stream logs to immutable S3 bucket

* **AWS Secrets Manager**
  Database credentials
  Rotation
  Cross-region replication

* **Amazon Virtual Private Cloud (VPC)**
  Private networking
  VPC endpoints
  Flow logs

---

# 8️⃣ Data Transfer & Ingestion (Optional but Included in Architecture)

* **AWS DataSync**
  On-prem → S3 transfers

* **AWS Transfer Family**
  Secure file ingestion

---

# 9️⃣ Disaster Recovery Components

* **S3 Cross-Region Replication** (built into S3)
* **Aurora cross-region read replica**
* **DynamoDB Global Tables** (if used for DR)
* **KMS Multi-Region keys**

---

# 🔎 Total Service Count

**Core required services (minimal production platform): ~20**

If you remove optional components (OpenSearch, EMR, Redshift, DataSync, Transfer Family), the minimal footprint is:

* S3
* KMS
* Glue
* Athena
* Step Functions
* EventBridge
* SQS
* Lambda
* Glue Data Catalog
* Aurora
* IAM
* IAM Identity Center
* Cognito
* CloudFront
* WAF
* API Gateway
* CloudTrail
* CloudWatch
* Config
* Secrets Manager
* VPC