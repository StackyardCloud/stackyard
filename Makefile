.PHONY: fmt tidy test ci install examples-docker examples-docker-provider examples-docker-aws examples-docker-gcp examples-docker-azure examples-docker-oci refarch-examples refarch-examples-aws refarch-examples-gcp refarch-examples-all coverage-all coverage-all-strict coverage-aws-contracts coverage-aws-contracts-strict coverage-aws-contracts-all-strict coverage-aws-io-contracts coverage-aws-io-contracts-strict coverage-aws-io-contracts-all-strict coverage-aws-doc-contracts coverage-aws-doc-contracts-strict coverage-aws-doc-contracts-all-strict coverage-aws-endpoints-smoke coverage-gcp-contracts coverage-gcp-contracts-strict coverage-gcp-contracts-all-strict coverage-gcp-io-contracts coverage-gcp-io-contracts-strict coverage-gcp-io-contracts-all-strict coverage-gcp-doc-contracts coverage-gcp-doc-contracts-strict coverage-gcp-doc-contracts-all-strict coverage-gcp-endpoints coverage-gcp-endpoints-strict coverage-azure-contracts coverage-azure-contracts-strict coverage-azure-contracts-all-strict coverage-azure-io-contracts coverage-azure-io-contracts-strict coverage-azure-io-contracts-all-strict coverage-azure-doc-contracts coverage-azure-doc-contracts-strict coverage-azure-doc-contracts-all-strict coverage-azure-endpoints coverage-azure-endpoints-strict provider-contracts provider-contracts-pr provider-contracts-nightly provider-contracts-all-strict aws-provider-contracts aws-provider-contracts-pr aws-provider-contracts-nightly aws-provider-contracts-all-strict azure-provider-contracts azure-provider-contracts-pr azure-provider-contracts-nightly azure-provider-contracts-all-strict gcp-provider-contracts gcp-provider-contracts-pr gcp-provider-contracts-nightly gcp-provider-contracts-all-strict up down restart

BUILD ?= 0
VOLUMES ?= 0
PROVIDER ?= aws
COVERAGE_FAIL_ON ?= not_implemented,service_error,client_error,contract_error,transport_error,skeleton_error,unknown_error,auth_error
AWS_STRICT_SERVICES ?= sts sqs
AWS_PR_SMOKE_SERVICES ?= dynamodb,sqs,ecr,lambda,sns,s3,sts,ses,athena,bedrock,cloudfront,ec2,ecs,eks,elasticache,elasticloadbalancing,eventbridge,iam,kinesis,kms,neptune,opensearch,rds,redshift,route53,waf
AWS_PR_SMOKE_ENDPOINTS ?= sqs.CreateQueue,sqs.GetQueueAttributes,sqs.SendMessage,sqs.ReceiveMessage,sqs.DeleteMessage,dynamodb.CreateTable,dynamodb.DescribeTable,dynamodb.GetItem,dynamodb.ListTables,dynamodb.DeleteTable,ecr.CreateRepository,ecr.DescribeRepositories,ecr.DeleteRepository,lambda.CreateFunction,lambda.Invoke,lambda.DeleteFunction,sns.CreateTopic,sns.ListTopics,s3.CreateBucket,sts.GetCallerIdentity,ses.ListConfigurationSets,athena.ListWorkGroups,bedrock.CreateGuardrail,cloudfront.CreateCachePolicy,ec2.AllocateAddress,ecs.ListClusters,eks.CreateCluster,elasticache.CreateCacheParameterGroup,elasticloadbalancing.CreateTargetGroup,eventbridge.CreateEventBus,iam.CreateGroup,kinesis.CreateStream,kms.CreateKey,neptune.CreateDBParameterGroup,opensearch.DescribeDomain,rds.CreateDbParameterGroup,redshift.DescribeClusters,route53.GetHostedZoneCount,wafv2.CreateIPSet
GCP_STRICT_SERVICES ?= cloudprofiler cloudquotas commerce_consumer_procurement configdelivery apigeeconnect mediatranslation config rapidmigrationassessment
AZURE_STRICT_SERVICES ?= blob queue keyvault

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy

test:
	go test ./...
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -Aws -Reset
else
	bash ./scripts/smoke-test.sh -a -r
endif

ci:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$ciStart=Get-Date; function Log-CI([string]$$msg){ Write-Host $$msg -ForegroundColor Cyan }; function Run-Step([string]$$name,[scriptblock]$$block){ $$stepStart=Get-Date; Log-CI ('[ci] step=' + $$name + ' start'); & $$block; $$elapsed=[int]((Get-Date)-$$stepStart).TotalSeconds; Log-CI ('[ci] step=' + $$name + ' complete elapsed=' + $$elapsed + 's') }; Run-Step 'fmt' { & '$(MAKE)' fmt }; Run-Step 'tidy' { & '$(MAKE)' tidy }; Run-Step 'test-packages' { $$packages = go list ./... | Where-Object { $$_ -notmatch '/examples(/|$)' }; Write-Host 'Running CI tests (excluding examples/)...'; go test $$packages }; Run-Step 'provider-contracts' { & '$(MAKE)' provider-contracts }; Run-Step 'act' { if (Test-Path .github/workflows) { if (Get-Command act -ErrorAction SilentlyContinue) { act -j test -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest } else { Log-CI '[ci] step=act skipped (not found)' } } else { Log-CI '[ci] step=act skipped (.github/workflows not found)' } }; Run-Step 'examples-docker' { & '$(MAKE)' examples-docker }; Run-Step 'coverage-all' { & '$(MAKE)' coverage-all }; $$total=[int]((Get-Date)-$$ciStart).TotalSeconds; Log-CI ('[ci] total elapsed=' + $$total + 's')"
else
	@set -e; \
	ci_start=$$(date +%s); \
	cyan='\033[36m'; \
	reset='\033[0m'; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=fmt start"; \
	$(MAKE) fmt; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=fmt complete elapsed=$$((step_end-step_start))s"; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=tidy start"; \
	$(MAKE) tidy; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=tidy complete elapsed=$$((step_end-step_start))s"; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=test-packages start"; \
	echo "Running CI tests (excluding examples/)..."; \
	go list ./... | grep -vE '^github.com/stackyard/stackyard/examples(/|$$)' | xargs go test; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=test-packages complete elapsed=$$((step_end-step_start))s"; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=provider-contracts start"; \
	$(MAKE) provider-contracts; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=provider-contracts complete elapsed=$$((step_end-step_start))s"; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=act start"; \
	if [ -d .github/workflows ]; then \
		if command -v act >/dev/null 2>&1; then \
			act -j test -P ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest; \
		else \
			printf "$${cyan}%s$${reset}\n" "[ci] step=act skipped (not found)"; \
		fi; \
	else \
		printf "$${cyan}%s$${reset}\n" "[ci] step=act skipped (.github/workflows not found)"; \
	fi; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=act complete elapsed=$$((step_end-step_start))s"; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=examples-docker start"; \
	$(MAKE) examples-docker; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=examples-docker complete elapsed=$$((step_end-step_start))s"; \
	step_start=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=coverage-all start"; \
	$(MAKE) coverage-all; \
	step_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] step=coverage-all complete elapsed=$$((step_end-step_start))s"; \
	ci_end=$$(date +%s); \
	printf "$${cyan}%s$${reset}\n" "[ci] total elapsed=$$((ci_end-ci_start))s"
endif

install:
	go install ./cmd/stackyard

examples-docker:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$providers=@('aws','gcp','azure','oci'); foreach($$p in $$providers){ Write-Host ('==> Running provider examples: ' + $$p) -ForegroundColor Cyan; & '$(MAKE)' examples-docker-provider PROVIDER=$$p ALLOW_MISSING_PROVIDER_DIR=1 }"
else
	@set -e; \
	for provider in aws gcp azure oci; do \
		echo "==> Running provider examples: $$provider"; \
		$(MAKE) examples-docker-provider PROVIDER=$$provider ALLOW_MISSING_PROVIDER_DIR=1; \
	done
endif

examples-docker-provider:
ifeq ($(OS),Windows_NT)
	powershell -NoProfile -ExecutionPolicy Bypass -Command "$$env:ALLOW_MISSING_PROVIDER_DIR='$(ALLOW_MISSING_PROVIDER_DIR)'; & '.\scripts\run-all-examples.ps1' -Provider '$(PROVIDER)'"
else
	PROVIDER=$(PROVIDER) ALLOW_MISSING_PROVIDER_DIR=$(ALLOW_MISSING_PROVIDER_DIR) bash ./scripts/run-all-examples.sh
endif

examples-docker-aws:
	$(MAKE) examples-docker-provider PROVIDER=aws

examples-docker-gcp:
	$(MAKE) examples-docker-provider PROVIDER=gcp

examples-docker-azure:
	$(MAKE) examples-docker-provider PROVIDER=azure

examples-docker-oci:
	$(MAKE) examples-docker-provider PROVIDER=oci

refarch-examples:
ifeq ($(OS),Windows_NT)
	$(MAKE) refarch-examples-aws
else
	$(MAKE) refarch-examples-aws
endif

refarch-examples-aws:
ifeq ($(OS),Windows_NT)
	PROVIDER=aws powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\run-reference-architecture-examples.ps1
else
	PROVIDER=aws bash ./scripts/run-reference-architecture-examples.sh
endif

refarch-examples-gcp:
ifeq ($(OS),Windows_NT)
	PROVIDER=gcp powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\run-reference-architecture-examples.ps1
else
	PROVIDER=gcp bash ./scripts/run-reference-architecture-examples.sh
endif

refarch-examples-all:
	$(MAKE) refarch-examples-aws
	$(MAKE) refarch-examples-gcp

coverage-all:
ifeq ($(OS),Windows_NT)
	py -3 scripts/awscli-endpoint-coverage.py --contract-mode --fail-on $(COVERAGE_FAIL_ON)
else
	python3 scripts/awscli-endpoint-coverage.py --contract-mode --fail-on $(COVERAGE_FAIL_ON)
endif

coverage-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/awscli-endpoint-coverage.py --contract-mode --fail-on $(COVERAGE_FAIL_ON)
else
	python3 scripts/awscli-endpoint-coverage.py --contract-mode --fail-on $(COVERAGE_FAIL_ON)
endif

coverage-aws-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/aws-contract-coverage.py
else
	python3 scripts/aws-contract-coverage.py
endif

coverage-aws-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(AWS_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> aws contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/aws-contract-coverage.py --service $$svc --require-service $$svc --fail-on any --no-rebuild-stackyard }"
else
	@set -e; \
	for svc in $(AWS_STRICT_SERVICES); do \
		echo "==> aws contract gate: $$svc"; \
		python3 scripts/aws-contract-coverage.py --service "$$svc" --require-service "$$svc" --fail-on any --no-rebuild-stackyard; \
	done
endif

coverage-aws-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/aws-contract-coverage.py --service "*" --fail-on any --no-rebuild-stackyard
else
	python3 scripts/aws-contract-coverage.py --service "*" --fail-on any --no-rebuild-stackyard
endif

coverage-aws-io-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/aws-io-contract-coverage.py
else
	python3 scripts/aws-io-contract-coverage.py
endif

coverage-aws-io-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(AWS_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> aws io contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/aws-io-contract-coverage.py --service $$svc --require-service $$svc --fail-on strict --no-rebuild-stackyard }"
else
	@set -e; \
	for svc in $(AWS_STRICT_SERVICES); do \
		echo "==> aws io contract gate: $$svc"; \
		python3 scripts/aws-io-contract-coverage.py --service "$$svc" --require-service "$$svc" --fail-on strict --no-rebuild-stackyard; \
	done
endif

coverage-aws-io-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/aws-io-contract-coverage.py --service "*" --fail-on strict --no-rebuild-stackyard
else
	python3 scripts/aws-io-contract-coverage.py --service "*" --fail-on strict --no-rebuild-stackyard
endif

coverage-aws-doc-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/aws-doc-contract-coverage.py
else
	python3 scripts/aws-doc-contract-coverage.py
endif

coverage-aws-doc-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(AWS_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> aws doc contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/aws-doc-contract-coverage.py --service $$svc --require-service $$svc --fail-on any }"
else
	@set -e; \
	for svc in $(AWS_STRICT_SERVICES); do \
		echo "==> aws doc contract gate: $$svc"; \
		python3 scripts/aws-doc-contract-coverage.py --service "$$svc" --require-service "$$svc" --fail-on any; \
	done
endif

coverage-aws-doc-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/aws-doc-contract-coverage.py --service "*" --fail-on any
else
	python3 scripts/aws-doc-contract-coverage.py --service "*" --fail-on any
endif

coverage-aws-endpoints-smoke:
ifeq ($(OS),Windows_NT)
	py -3 scripts/awscli-endpoint-coverage.py --contract-mode --include-services "$(AWS_PR_SMOKE_SERVICES)" --include-endpoints "$(AWS_PR_SMOKE_ENDPOINTS)" --fail-on "$(COVERAGE_FAIL_ON)" --no-rebuild-stackyard --quiet
else
	python3 scripts/awscli-endpoint-coverage.py --contract-mode --include-services "$(AWS_PR_SMOKE_SERVICES)" --include-endpoints "$(AWS_PR_SMOKE_ENDPOINTS)" --fail-on "$(COVERAGE_FAIL_ON)" --no-rebuild-stackyard --quiet
endif

coverage-gcp-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-contract-coverage.py
else
	python3 scripts/gcp-contract-coverage.py
endif

coverage-gcp-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(GCP_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> gcp contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/gcp-contract-coverage.py --service $$svc --fail-on any }"
else
	@set -e; \
	for svc in $(GCP_STRICT_SERVICES); do \
		echo "==> gcp contract gate: $$svc"; \
		python3 scripts/gcp-contract-coverage.py --service "$$svc" --fail-on any; \
	done
endif

coverage-gcp-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-contract-coverage.py --service "*" --fail-on any
else
	python3 scripts/gcp-contract-coverage.py --service "*" --fail-on any
endif

coverage-gcp-io-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-io-contract-coverage.py
else
	python3 scripts/gcp-io-contract-coverage.py
endif

coverage-gcp-io-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(GCP_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> gcp io contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/gcp-io-contract-coverage.py --service $$svc --require-service $$svc --fail-on strict }"
else
	@set -e; \
	for svc in $(GCP_STRICT_SERVICES); do \
		echo "==> gcp io contract gate: $$svc"; \
		python3 scripts/gcp-io-contract-coverage.py --service "$$svc" --require-service "$$svc" --fail-on strict; \
	done
endif

coverage-gcp-io-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-io-contract-coverage.py --service "*" --fail-on strict
else
	python3 scripts/gcp-io-contract-coverage.py --service "*" --fail-on strict
endif

coverage-gcp-doc-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-doc-contract-coverage.py
else
	python3 scripts/gcp-doc-contract-coverage.py
endif

coverage-gcp-doc-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(GCP_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> gcp doc contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/gcp-doc-contract-coverage.py --service $$svc --require-service $$svc --fail-on any }"
else
	@set -e; \
	for svc in $(GCP_STRICT_SERVICES); do \
		echo "==> gcp doc contract gate: $$svc"; \
		python3 scripts/gcp-doc-contract-coverage.py --service "$$svc" --require-service "$$svc" --fail-on any; \
	done
endif

coverage-gcp-doc-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-doc-contract-coverage.py --service "*" --fail-on any
else
	python3 scripts/gcp-doc-contract-coverage.py --service "*" --fail-on any
endif

coverage-gcp-endpoints:
ifeq ($(OS),Windows_NT)
	py -3 scripts/gcp-endpoint-coverage.py
else
	python3 scripts/gcp-endpoint-coverage.py
endif

coverage-gcp-endpoints-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$docs = Get-ChildItem -Path docs -Filter 'gcp-*-plan.md' -File -ErrorAction SilentlyContinue; if ($$docs.Count -gt 0) { py -3 scripts/gcp-endpoint-coverage.py --service '*' --fail-on any --live-smoke } else { Write-Host '==> running gcp endpoint coverage without doc gate (no docs/gcp-*-plan.md files present)' -ForegroundColor Yellow; py -3 scripts/gcp-endpoint-coverage.py --service '*' --fail-on contract,io --live-smoke }"
else
	@if find docs -maxdepth 1 -type f -name 'gcp-*-plan.md' | grep -q .; then \
		python3 scripts/gcp-endpoint-coverage.py --service "*" --fail-on any --live-smoke; \
	else \
		echo "==> running gcp endpoint coverage without doc gate (no docs/gcp-*-plan.md files present)"; \
		python3 scripts/gcp-endpoint-coverage.py --service "*" --fail-on contract,io --live-smoke; \
	fi
endif

coverage-azure-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-contract-coverage.py
else
	python3 scripts/azure-contract-coverage.py
endif

coverage-azure-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(AZURE_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> azure contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/azure-contract-coverage.py --service $$svc --fail-on any }"
else
	@set -e; \
	for svc in $(AZURE_STRICT_SERVICES); do \
		echo "==> azure contract gate: $$svc"; \
		python3 scripts/azure-contract-coverage.py --service "$$svc" --fail-on any; \
	done
endif

coverage-azure-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-contract-coverage.py --service "*" --fail-on any
else
	python3 scripts/azure-contract-coverage.py --service "*" --fail-on any
endif

coverage-azure-io-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-io-contract-coverage.py
else
	python3 scripts/azure-io-contract-coverage.py
endif

coverage-azure-io-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(AZURE_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> azure io contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/azure-io-contract-coverage.py --service $$svc --require-service $$svc --fail-on strict }"
else
	@set -e; \
	for svc in $(AZURE_STRICT_SERVICES); do \
		echo "==> azure io contract gate: $$svc"; \
		python3 scripts/azure-io-contract-coverage.py --service "$$svc" --require-service "$$svc" --fail-on strict; \
	done
endif

coverage-azure-io-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-io-contract-coverage.py --service "*" --fail-on strict
else
	python3 scripts/azure-io-contract-coverage.py --service "*" --fail-on strict
endif

coverage-azure-doc-contracts:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-doc-contract-coverage.py
else
	python3 scripts/azure-doc-contract-coverage.py
endif

coverage-azure-doc-contracts-strict:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; $$services='$(AZURE_STRICT_SERVICES)'.Split(' ',[System.StringSplitOptions]::RemoveEmptyEntries); foreach($$svc in $$services){ Write-Host ('==> azure doc contract gate: ' + $$svc) -ForegroundColor Cyan; py -3 scripts/azure-doc-contract-coverage.py --service $$svc --fail-on any }"
else
	@set -e; \
	for svc in $(AZURE_STRICT_SERVICES); do \
		echo "==> azure doc contract gate: $$svc"; \
		python3 scripts/azure-doc-contract-coverage.py --service "$$svc" --fail-on any; \
	done
endif

coverage-azure-doc-contracts-all-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-doc-contract-coverage.py --service "*" --fail-on any
else
	python3 scripts/azure-doc-contract-coverage.py --service "*" --fail-on any
endif

coverage-azure-endpoints:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-endpoint-coverage.py
else
	python3 scripts/azure-endpoint-coverage.py
endif

coverage-azure-endpoints-strict:
ifeq ($(OS),Windows_NT)
	py -3 scripts/azure-endpoint-coverage.py --service "*" --fail-on any --live-smoke
else
	python3 scripts/azure-endpoint-coverage.py --service "*" --fail-on any --live-smoke
endif

provider-contracts: provider-contracts-pr

provider-contracts-pr: aws-provider-contracts-pr azure-provider-contracts-pr gcp-provider-contracts-pr

provider-contracts-nightly: aws-provider-contracts-nightly azure-provider-contracts-nightly gcp-provider-contracts-nightly

provider-contracts-all-strict: provider-contracts-nightly

aws-provider-contracts: aws-provider-contracts-pr

aws-provider-contracts-pr:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$docs = Get-ChildItem -Path docs -Filter 'aws-*-plan.md' -File -ErrorAction SilentlyContinue; if ($$docs.Count -gt 0) { & '$(MAKE)' coverage-aws-doc-contracts-strict } else { Write-Host '==> skipping aws doc contract gate (no docs/aws-*-plan.md files present)' -ForegroundColor Yellow }"
else
	@if find docs -maxdepth 1 -type f -name 'aws-*-plan.md' | grep -q .; then \
		$(MAKE) coverage-aws-doc-contracts-strict; \
	else \
		echo "==> skipping aws doc contract gate (no docs/aws-*-plan.md files present)"; \
	fi
endif
	$(MAKE) coverage-aws-endpoints-smoke

aws-provider-contracts-nightly: aws-provider-contracts-all-strict

aws-provider-contracts-all-strict:
	$(MAKE) coverage-aws-contracts-all-strict
	$(MAKE) coverage-aws-io-contracts-all-strict
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$docs = Get-ChildItem -Path docs -Filter 'aws-*-plan.md' -File -ErrorAction SilentlyContinue; if ($$docs.Count -gt 0) { & '$(MAKE)' coverage-aws-doc-contracts-all-strict } else { Write-Host '==> skipping aws doc contract gate (no docs/aws-*-plan.md files present)' -ForegroundColor Yellow }"
else
	@if find docs -maxdepth 1 -type f -name 'aws-*-plan.md' | grep -q .; then \
		$(MAKE) coverage-aws-doc-contracts-all-strict; \
	else \
		echo "==> skipping aws doc contract gate (no docs/aws-*-plan.md files present)"; \
	fi
endif

azure-provider-contracts: azure-provider-contracts-pr

azure-provider-contracts-pr:
	$(MAKE) coverage-azure-contracts-strict
	$(MAKE) coverage-azure-io-contracts-strict
	$(MAKE) coverage-azure-doc-contracts-strict

azure-provider-contracts-nightly: azure-provider-contracts-all-strict

azure-provider-contracts-all-strict:
	$(MAKE) coverage-azure-contracts-all-strict
	$(MAKE) coverage-azure-io-contracts-all-strict
	$(MAKE) coverage-azure-doc-contracts-all-strict
	$(MAKE) coverage-azure-endpoints-strict

gcp-provider-contracts: gcp-provider-contracts-pr

gcp-provider-contracts-pr:
	$(MAKE) coverage-gcp-contracts-strict
	$(MAKE) coverage-gcp-io-contracts-strict
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$docs = Get-ChildItem -Path docs -Filter 'gcp-*-plan.md' -File -ErrorAction SilentlyContinue; if ($$docs.Count -gt 0) { & '$(MAKE)' coverage-gcp-doc-contracts-strict } else { Write-Host '==> skipping gcp doc contract gate (no docs/gcp-*-plan.md files present)' -ForegroundColor Yellow }"
else
	@if find docs -maxdepth 1 -type f -name 'gcp-*-plan.md' | grep -q .; then \
		$(MAKE) coverage-gcp-doc-contracts-strict; \
	else \
		echo "==> skipping gcp doc contract gate (no docs/gcp-*-plan.md files present)"; \
	fi
endif

gcp-provider-contracts-nightly: gcp-provider-contracts-all-strict

gcp-provider-contracts-all-strict:
	$(MAKE) coverage-gcp-contracts-all-strict
	$(MAKE) coverage-gcp-io-contracts-all-strict
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$docs = Get-ChildItem -Path docs -Filter 'gcp-*-plan.md' -File -ErrorAction SilentlyContinue; if ($$docs.Count -gt 0) { & '$(MAKE)' coverage-gcp-doc-contracts-all-strict } else { Write-Host '==> skipping gcp doc contract gate (no docs/gcp-*-plan.md files present)' -ForegroundColor Yellow }"
else
	@if find docs -maxdepth 1 -type f -name 'gcp-*-plan.md' | grep -q .; then \
		$(MAKE) coverage-gcp-doc-contracts-all-strict; \
	else \
		echo "==> skipping gcp doc contract gate (no docs/gcp-*-plan.md files present)"; \
	fi
endif
	$(MAKE) coverage-gcp-endpoints-strict

up:
	if [ "$(BUILD)" = "1" ]; then \
		docker compose -f examples/docker-compose.yml up --build; \
	else \
		docker compose -f examples/docker-compose.yml up; \
	fi

down:
	if [ "$(VOLUMES)" = "1" ]; then \
		docker compose -f examples/docker-compose.yml down -v; \
	else \
		docker compose -f examples/docker-compose.yml down \
	fi

restart: down up
