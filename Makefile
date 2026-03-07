.PHONY: fmt tidy test ci install examples-docker examples-docker-provider examples-docker-aws examples-docker-gcp examples-docker-azure examples-docker-oci refarch-examples refarch-examples-aws refarch-examples-gcp refarch-examples-all coverage-all coverage-all-strict coverage-gcp-contracts coverage-gcp-contracts-strict coverage-gcp-io-contracts coverage-gcp-io-contracts-strict provider-contracts up down restart

BUILD ?= 0
VOLUMES ?= 0
PROVIDER ?= aws
COVERAGE_FAIL_ON ?= not_implemented,service_error,client_error,contract_error,transport_error,skeleton_error,unknown_error,auth_error
GCP_STRICT_SERVICES ?= cloudprofiler cloudquotas commerce_consumer_procurement configdelivery apigeeconnect mediatranslation config rapidmigrationassessment

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
