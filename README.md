# Stackyard

Stackyard is a Go-based local AWS emulator focused on fast startup, simple local workflows, and clear service boundaries.

## Current status

Implemented services (in-memory):
- S3 subset: create bucket, list buckets, put/get/list/head objects, range reads, metadata, ETags, pre-signed URLs, copy, delete, delete-multi, pagination, multipart, conditional headers, Content-MD5 validation, ACLs, CORS, lifecycle config, tagging, website config, logging config.
- SQS subset: create queue, list queues, send/receive messages.

This is an API-compatible *starting point* for local workflows, not a full AWS protocol clone yet.

AWS S3 (staged compatibility):
- Path-style endpoints with XML responses.
- SigV4 request verification (access key + secret required).

## Run

```bash
go run ./cmd/stackyard -addr :4566
```

Or run with Docker Compose:

```bash
docker compose up --build
```

Credentials for AWS-style endpoints:

```bash
export STACKYARD_ACCESS_KEY=stackyard
export STACKYARD_SECRET_KEY=stackyard
```

Verbose logging:

```bash
export STACKYARD_LOG_LEVEL=debug
```

Health check:

```bash
curl http://localhost:4566/_stackyard/health
```

## Quick API smoke test

Create and list bucket:

```bash
curl -X PUT http://localhost:4566/s3/buckets/demo
curl http://localhost:4566/s3/buckets
```

PowerShell:

```powershell
curl.exe -X PUT "http://localhost:4566/s3/buckets/demo"
curl.exe "http://localhost:4566/s3/buckets"
```

Put and get object:

```bash
curl -X PUT --data 'hello stackyard' \
  -H 'Content-Type: text/plain' \
  http://localhost:4566/s3/objects/demo/notes/hello.txt

curl http://localhost:4566/s3/objects/demo/notes/hello.txt
```

PowerShell:

```powershell
curl.exe -X PUT --data "hello stackyard" -H "Content-Type: text/plain" "http://localhost:4566/s3/objects/demo/notes/hello.txt"
curl.exe "http://localhost:4566/s3/objects/demo/notes/hello.txt"
```

Create queue and send/receive:

```bash
curl -X PUT http://localhost:4566/sqs/queues/jobs

curl -X POST http://localhost:4566/sqs/messages/jobs \
  -H 'Content-Type: application/json' \
  -d '{"body":"run build"}'

curl -X POST http://localhost:4566/sqs/messages/jobs/receive \
  -H 'Content-Type: application/json' \
  -d '{"max_messages":1}'
```

PowerShell:

```powershell
curl.exe -X PUT "http://localhost:4566/sqs/queues/jobs"

@'
{"body":"run build"}
'@ | curl.exe -X POST "http://localhost:4566/sqs/messages/jobs" -H "Content-Type: application/json" --data-binary "@-"

@'
{"max_messages":1}
'@ | curl.exe -X POST "http://localhost:4566/sqs/messages/jobs/receive" -H "Content-Type: application/json" --data-binary "@-"
```

## Smoke test scripts

Run all smoke tests in one go:

```bash
./scripts/smoke-test.sh
```

PowerShell:

```powershell
.\scripts\smoke-test.ps1
```

AWS S3 (XML + SigV4) smoke tests (requires AWS CLI):

```bash
./scripts/smoke-test.sh -a
```

```powershell
.\scripts\smoke-test.ps1 -Aws
```

Reset the Docker container before AWS CLI tests:

```bash
./scripts/smoke-test.sh -a -r
```

```powershell
.\scripts\smoke-test.ps1 -Aws -Reset
```

## AWS-style S3 (XML + SigV4)

Path-style and virtual-host S3 endpoints with SigV4. For AWS CLI/SDKs, force path style if needed:

```bash
export AWS_ACCESS_KEY_ID=stackyard
export AWS_SECRET_ACCESS_KEY=stackyard
export AWS_REGION=us-east-1
export AWS_S3_FORCE_PATH_STYLE=true

aws --endpoint-url http://localhost:4566 s3api create-bucket --bucket demo
aws --endpoint-url http://localhost:4566 s3api list-objects-v2 --bucket demo
aws --endpoint-url http://localhost:4566 s3api put-object --bucket demo --key notes/hello.txt --body ./README.md
aws --endpoint-url http://localhost:4566 s3api get-object --bucket demo --key notes/hello.txt /tmp/hello.txt
```

Enable bucket versioning:

```bash
aws --endpoint-url http://localhost:4566 s3api put-bucket-versioning --bucket demo --versioning-configuration Status=Enabled
aws --endpoint-url http://localhost:4566 s3api get-bucket-versioning --bucket demo
aws --endpoint-url http://localhost:4566 s3api list-object-versions --bucket demo
```

Configure bucket CORS:

```bash
cat > /tmp/cors.xml <<'EOF'
<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <CORSRule>
    <AllowedOrigin>http://example.com</AllowedOrigin>
    <AllowedMethod>GET</AllowedMethod>
    <AllowedHeader>*</AllowedHeader>
    <MaxAgeSeconds>300</MaxAgeSeconds>
  </CORSRule>
</CORSConfiguration>
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-cors --bucket demo --cors-configuration file:///tmp/cors.xml
aws --endpoint-url http://localhost:4566 s3api get-bucket-cors --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-cors --bucket demo
```

Configure bucket lifecycle:

```bash
cat > /tmp/lifecycle.xml <<'EOF'
<LifecycleConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Rule>
    <ID>rule-1</ID>
    <Prefix>logs/</Prefix>
    <Status>Enabled</Status>
  </Rule>
</LifecycleConfiguration>
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-lifecycle-configuration --bucket demo --lifecycle-configuration file:///tmp/lifecycle.xml
aws --endpoint-url http://localhost:4566 s3api get-bucket-lifecycle-configuration --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-lifecycle --bucket demo
```

Configure bucket tagging:

```bash
cat > /tmp/tagging.xml <<'EOF'
<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <TagSet>
    <Tag>
      <Key>env</Key>
      <Value>dev</Value>
    </Tag>
  </TagSet>
</Tagging>
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-tagging --bucket demo --tagging file:///tmp/tagging.xml
aws --endpoint-url http://localhost:4566 s3api get-bucket-tagging --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-tagging --bucket demo
```

Configure bucket website:

```bash
cat > /tmp/website.xml <<'EOF'
<WebsiteConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <IndexDocument>
    <Suffix>index.html</Suffix>
  </IndexDocument>
  <ErrorDocument>
    <Key>error.html</Key>
  </ErrorDocument>
</WebsiteConfiguration>
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-website --bucket demo --website-configuration file:///tmp/website.xml
aws --endpoint-url http://localhost:4566 s3api get-bucket-website --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-website --bucket demo
```

Configure bucket logging:

```bash
cat > /tmp/logging.xml <<'EOF'
<BucketLoggingStatus xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <LoggingEnabled>
    <TargetBucket>demo</TargetBucket>
    <TargetPrefix>logs/</TargetPrefix>
  </LoggingEnabled>
</BucketLoggingStatus>
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-logging --bucket demo --bucket-logging-status file:///tmp/logging.xml
aws --endpoint-url http://localhost:4566 s3api get-bucket-logging --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-logging --bucket demo
```

Stackyard delivers access logs as newline-delimited entries under the target bucket and prefix (for example, `logs/2026-02-06-15-04-05-<id>.log`).

Configure bucket replication:

```bash
cat > /tmp/replication.xml <<'EOF'
<ReplicationConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Role>arn:aws:iam::123456789012:role/replication</Role>
  <Rule>
    <ID>replicate-all</ID>
    <Status>Enabled</Status>
    <Prefix></Prefix>
    <DeleteMarkerReplication>
      <Status>Enabled</Status>
    </DeleteMarkerReplication>
    <Destination>
      <Bucket>replica-bucket</Bucket>
    </Destination>
  </Rule>
</ReplicationConfiguration>
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-replication --bucket demo --replication-configuration file:///tmp/replication.xml
aws --endpoint-url http://localhost:4566 s3api get-bucket-replication --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-replication --bucket demo
```

Get bucket location:

```bash
aws --endpoint-url http://localhost:4566 s3api get-bucket-location --bucket demo
```

Bucket policy:

```bash
cat > /tmp/policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "PublicRead",
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::demo/*"
  }]
}
EOF
aws --endpoint-url http://localhost:4566 s3api put-bucket-policy --bucket demo --policy file:///tmp/policy.json
aws --endpoint-url http://localhost:4566 s3api get-bucket-policy --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-bucket-policy --bucket demo
```

Bucket policy status:

```bash
aws --endpoint-url http://localhost:4566 s3api get-bucket-policy-status --bucket demo
```

Object tagging:

```bash
cat > /tmp/object-tags.xml <<'EOF'
<Tagging xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <TagSet>
    <Tag>
      <Key>team</Key>
      <Value>stackyard</Value>
    </Tag>
  </TagSet>
</Tagging>
EOF
aws --endpoint-url http://localhost:4566 s3api put-object-tagging --bucket demo --key notes/hello.txt --tagging file:///tmp/object-tags.xml
aws --endpoint-url http://localhost:4566 s3api get-object-tagging --bucket demo --key notes/hello.txt
aws --endpoint-url http://localhost:4566 s3api delete-object-tagging --bucket demo --key notes/hello.txt
```

Object attributes:

```bash
aws --endpoint-url http://localhost:4566 s3api get-object-attributes \
  --bucket demo \
  --key notes/hello.txt \
  --object-attributes ETag,ObjectSize,StorageClass,LastModified,VersionId
```

Object retention:

```bash
cat > /tmp/retention.xml <<'EOF'
<Retention xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Mode>GOVERNANCE</Mode>
  <RetainUntilDate>2026-12-31T00:00:00Z</RetainUntilDate>
</Retention>
EOF
aws --endpoint-url http://localhost:4566 s3api put-object-retention --bucket demo --key notes/hello.txt --retention file:///tmp/retention.xml
aws --endpoint-url http://localhost:4566 s3api get-object-retention --bucket demo --key notes/hello.txt
```

Object legal hold:

```bash
cat > /tmp/legal-hold.xml <<'EOF'
<LegalHold xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Status>ON</Status>
</LegalHold>
EOF
aws --endpoint-url http://localhost:4566 s3api put-object-legal-hold --bucket demo --key notes/hello.txt --legal-hold file:///tmp/legal-hold.xml
aws --endpoint-url http://localhost:4566 s3api get-object-legal-hold --bucket demo --key notes/hello.txt
```

Bucket object lock configuration:

```bash
cat > /tmp/object-lock.xml <<'EOF'
<ObjectLockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <ObjectLockEnabled>Enabled</ObjectLockEnabled>
  <Rule>
    <DefaultRetention>
      <Mode>GOVERNANCE</Mode>
      <Days>1</Days>
    </DefaultRetention>
  </Rule>
</ObjectLockConfiguration>
EOF
aws --endpoint-url http://localhost:4566 s3api put-object-lock-configuration --bucket demo --object-lock-configuration file:///tmp/object-lock.xml
aws --endpoint-url http://localhost:4566 s3api get-object-lock-configuration --bucket demo
```

Multipart list operations:

```bash
aws --endpoint-url http://localhost:4566 s3api list-multipart-uploads --bucket demo
aws --endpoint-url http://localhost:4566 s3api list-parts --bucket demo --key notes/multipart.txt --upload-id <upload-id>
aws --endpoint-url http://localhost:4566 s3api list-multipart-uploads --bucket demo --max-uploads 1
aws --endpoint-url http://localhost:4566 s3api list-parts --bucket demo --key notes/multipart.txt --upload-id <upload-id> --max-parts 1
aws --endpoint-url http://localhost:4566 s3api list-multipart-uploads --bucket demo --prefix notes/ --delimiter /
aws --endpoint-url http://localhost:4566 s3api list-multipart-uploads --bucket demo --encoding-type url
aws --endpoint-url http://localhost:4566 s3api list-parts --bucket demo --key notes/multipart.txt --upload-id <upload-id> --encoding-type url
```

Upload part copy:

```bash
aws --endpoint-url http://localhost:4566 s3api upload-part-copy \
  --bucket demo \
  --key notes/multipart-copy.txt \
  --part-number 1 \
  --upload-id <upload-id> \
  --copy-source demo/notes/hello.txt
```

Public access block:

```bash
cat > /tmp/public-access.xml <<'EOF'
<PublicAccessBlockConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <BlockPublicAcls>true</BlockPublicAcls>
  <IgnorePublicAcls>true</IgnorePublicAcls>
  <BlockPublicPolicy>true</BlockPublicPolicy>
  <RestrictPublicBuckets>true</RestrictPublicBuckets>
</PublicAccessBlockConfiguration>
EOF
aws --endpoint-url http://localhost:4566 s3api put-public-access-block --bucket demo --public-access-block-configuration file:///tmp/public-access.xml
aws --endpoint-url http://localhost:4566 s3api get-public-access-block --bucket demo
aws --endpoint-url http://localhost:4566 s3api delete-public-access-block --bucket demo
```

Pre-signed URL example:

```bash
aws --endpoint-url http://localhost:4566 s3 presign s3://demo/notes/hello.txt --expires-in 300
```

Range read example:

```bash
curl -H "Range: bytes=0-9" http://localhost:4566/demo/notes/hello.txt
```

Verbose output:

```bash
./scripts/smoke-test.sh -v
```

```powershell
.\scripts\smoke-test.ps1 -Verbose
```

## Compatibility tests

Local S3 compatibility suite:

```bash
go test ./internal/server -run TestS3CoreDataPlaneCompatibility
```

## CI test target

Run the full test suite (includes compatibility tests):

```bash
./scripts/ci.sh
```

```powershell
.\scripts\ci.ps1
```

## Makefile helpers

```bash
make fmt
make tidy
make test
make ci
make up
make down
make restart
```

Build control:

```bash
make up BUILD=1
```

## CI workflow

GitHub Actions runs the same `scripts/ci.sh` on every push and PR.

## Suggested next milestones

1. Add SigV4 request parsing and request signing validation layer.
2. Add AWS protocol adapters (S3 XML, SQS Query protocol).
3. Persist state with pluggable backends (BoltDB, SQLite, Postgres).
4. Add Lambda, DynamoDB, and SNS emulators behind service interfaces.
5. Add deterministic replay and snapshot tooling for CI.
