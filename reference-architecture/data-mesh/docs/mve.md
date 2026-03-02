# Minimum Viable Environment (MVE) - Data Mesh

## Accounts
- 1 platform account (org trail, central logs, security services)
- 2 domain accounts (prove cross-account reading + API product)

## Success criteria
- Projection from stream -> DynamoDB within seconds.
- API supports full/masked/none modes based on ABAC claims.
- Poison pill goes to DLQ and can be replayed after fix.
- Dashboards show iterator age, error rate, p95 API latency, DLQ depth.
