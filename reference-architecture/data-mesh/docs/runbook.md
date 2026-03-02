# Operations Runbook (Domain)

## Consumer lag rising
**Signals:** IteratorAgeMilliseconds increasing, delayed projections  
**Actions:** identify hot partition keys; increase shards; tune Lambda batch/parallelization; check throttles.

## DLQ non-empty
**Signals:** DLQ depth > 0  
**Actions:** inspect message; patch consumer; replay; verify idempotency.

## KMS errors/throttling
**Signals:** KMS AccessDenied/Throttling in Lambda logs  
**Actions:** validate key policy; review quotas; reduce CMK-per-tenant sprawl if needed; consider short-lived DEK caching per request.

## Masking regression
**Signals:** policy test failures, unexpected data exposure paths  
**Actions:** roll back; fix ABAC logic; add regression tests; audit CloudTrail + access logs.
