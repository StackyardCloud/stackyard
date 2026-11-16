# NGINXaaS for Azure (`nginx-2025-11-01`) Staged Plan

## Objective

Emulate the NGINXaaS for Azure control plane (`Nginx.NginxPlus`) on the shared ARM handler (`internal/server/azure_resource_manager.go`): NGINX deployments with their configuration, certificates, dataplane API keys and WAF policies. No NGINX runs and no certificate is fetched.

## References

Microsoft Learn pages for the `2025-11-01` API version:

- `https://learn.microsoft.com/en-us/rest/api/nginx/deployments/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/nginx/configurations/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/nginx/certificates/create-or-update`
- `https://learn.microsoft.com/en-us/rest/api/nginx/api-keys/create-or-update`

## Service Naming

- service key: `nginx`
- display name: `NGINXaaS for Azure`
- implementation: `internal/server/provider_azure_nginx.go`
- tests: `internal/server/provider_azure_nginx_test.go`
- example: `examples/azure/nginx-2025-11-01/`

## Operations In Scope

| Resource | Kind | Operations |
|---|---|---|
| `nginxDeployments` | tracked | create/update, get, update (PATCH), delete, list by group and subscription, `listDefaultWafPolicies` |
| `nginxDeployments/configurations` | proxy child | create/update (name `default`), get, delete, list, `analyze` |
| `nginxDeployments/certificates` | proxy child | create/update, get, delete, list |
| `nginxDeployments/apiKeys` | proxy child | create/update, get, delete, list |
| `nginxDeployments/wafPolicies` | proxy child | create/update, get, delete, list, `analyzeWafPolicy` |

## Behavior

CRUD, listing, cascade and error behavior come from the shared ARM handler (see `docs/resources-2021-04-01-plan.md`). Service-specific rules:

- configurations, certificates, API keys and WAF policies have no PATCH (405)
- names are 1-30 letters, digits or hyphens, starting and ending with a letter or digit; the configuration is named `default`
- deployments need a `sku.name`, a subnet ID in `networkInterfaceConfiguration`, and a public IP address ID or private address (Static ones with an address) in the frontend; `scalingProperties` sets an integer `capacity` or autoscale profiles (each with a name and integer `min` no greater than `max`), not both; `upgradeChannel` is `stable` or `preview`, `preferredEmail` must be an email address and WAF `activationState` is `Enabled` or `Disabled`
- deployments report an NGINX Plus version, `ipAddress` (the first private address, or a stable public one), an HTTPS `dataplaneApiEndpoint` on host `{name}-{hash}.{location}.nginxaas.net`, and a WAF status when App Protect is configured
- configurations need a `rootFile` and files, protected files or a package; files need absolute virtual paths and base64 content, and the root file must be one of them unless a package is sent; protected file content is write-only and reported as a SHA-256 `contentHash`
- `analyze` applies the same checks to the `config` sent and reports `SUCCEEDED` or `FAILED` with the error
- certificates need absolute key and certificate paths and a Key Vault secret URL; they report a stable thumbprint and the secret version (from the URL when it names one); an optional `location` is kept
- API keys need a `secretText` of at least 12 characters, which is write-only and reported by its first three characters as `hint`, and an RFC 3339 `endDateTime` when set
- WAF policies need an absolute `filepath` and base64 `content`, and report compiled and applied states; `listDefaultWafPolicies` returns NGINX App Protect's default and strict policies, and `analyzeWafPolicy` checks the content is base64

Out of scope for this stage:
- parsing NGINX configuration, `nginx -t`, and diagnostics beyond the checks above
- fetching certificates from Key Vault and certificate errors
- long-running operation polling: every operation completes synchronously

## Quality Gates

- `go test ./internal/server -run TestAzureNginx -count=1 -race`
- `python3 scripts/azure-contract-coverage.py --service nginx`
- `python3 scripts/azure-io-contract-coverage.py --service nginx`
- `python3 scripts/azure-doc-contract-coverage.py --service nginx`
