#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PROVIDER_NAME="$(printf '%s' "${PROVIDER:-aws}" | tr '[:upper:]' '[:lower:]')"

compose_files=()
if [ -n "${REFARCH_COMPOSE:-}" ]; then
  if [ ! -f "${REFARCH_COMPOSE}" ]; then
    echo "REFARCH_COMPOSE file not found: ${REFARCH_COMPOSE}"
    exit 1
  fi
  compose_files+=("${REFARCH_COMPOSE}")
else
  while IFS= read -r compose_file; do
    compose_files+=("$compose_file")
  done < <(find reference-architecture -mindepth 4 -maxdepth 4 -type f -path "*/${PROVIDER_NAME}/example/docker-compose.yml" | sort)
fi

if [ "${#compose_files[@]}" -eq 0 ]; then
  echo "No reference architecture compose files found for provider '${PROVIDER_NAME}'"
  exit 0
fi

wait_for_stackyard_ready() {
  local compose_file="$1"
  for _ in $(seq 1 90); do
    if docker compose -f "$compose_file" logs --no-color stackyard 2>/dev/null | grep -q "stackyard listening on"; then
      return 0
    fi
    sleep 1
  done
  return 1
}

wait_for_backend_ready() {
  local base_url="$1"
  for _ in $(seq 1 90); do
    if curl -fsS "$base_url/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

run_smoke() {
  local compose_file="$1"
  local backend_url="$2"

  if [[ "$compose_file" == *"/data-pipeline/"* ]]; then
    curl -fsS -X POST "$backend_url/api/v1/tenants/tenant-001/bootstrap" >/dev/null || return 1
    curl -fsS -X POST "$backend_url/api/v1/tenants/tenant-001/ingest" >/dev/null || return 1
    curl -fsS -X POST "$backend_url/api/v1/tenants/tenant-001/run" >/dev/null || return 1
    curl -fsS "$backend_url/api/v1/summary" >/dev/null || return 1
    return
  fi

  if [[ "$compose_file" == *"/data-mesh/"* ]]; then
    curl -fsS -X POST "$backend_url/api/v1/domains/orders/bootstrap" >/dev/null || return 1
    curl -fsS -X POST "$backend_url/api/v1/domains/orders/tenants/tenant-123/events" \
      -H 'Content-Type: application/json' \
      -d '{"event_type":"orders.order.created","schema_version":"v1","correlation_id":"trace-001","payload":{"entity_id":"ord-1001","amount":42.75,"customer_email":"alice@example.com"}}' >/dev/null || return 1
    curl -fsS -X POST "$backend_url/api/v1/domains/orders/project" -H 'Content-Type: application/json' -d '{"limit":50}' >/dev/null || return 1
    curl -fsS "$backend_url/api/v1/domains/orders/products/orders/tenant-123" \
      -H 'X-Claim-Tenant-Id: tenant-123' \
      -H 'X-Claim-Scopes: read:full' >/dev/null || return 1
    curl -fsS "$backend_url/api/v1/summary" >/dev/null || return 1
    return
  fi
}

for compose_file in "${compose_files[@]}"; do
  echo "==> Running reference architecture compose: $compose_file"

  services=()
  while IFS= read -r service; do
    services+=("$service")
  done < <(docker compose -f "$compose_file" config --services)
  if [ "${#services[@]}" -eq 0 ]; then
    echo "No services found in $compose_file"
    exit 1
  fi

  backend_service=""
  for service in "${services[@]}"; do
    if [ "$service" = "backend" ]; then
      backend_service="$service"
      break
    fi
  done
  if [ -z "$backend_service" ]; then
    for service in "${services[@]}"; do
      if [ "$service" != "stackyard" ]; then
        backend_service="$service"
        break
      fi
    done
  fi

  docker compose -f "$compose_file" down --remove-orphans -v >/dev/null 2>&1 || true

  status=0
  if [ -n "$backend_service" ] && printf '%s
' "${services[@]}" | grep -q '^stackyard$'; then
    docker compose -f "$compose_file" up -d --build stackyard "$backend_service" || status=$?
    if [ "$status" -eq 0 ] && ! wait_for_stackyard_ready "$compose_file"; then
      echo "Stackyard health check failed for compose: $compose_file"
      docker compose -f "$compose_file" logs stackyard || true
      status=1
    fi

    backend_host_port=""
    if [ "$status" -eq 0 ] && [ -n "$backend_service" ]; then
      backend_host_port="$(docker compose -f "$compose_file" port "$backend_service" 8080 2>/dev/null | head -n1 | sed 's/^.*://')"
      if [ -n "$backend_host_port" ]; then
        backend_url="http://127.0.0.1:${backend_host_port}"
        if ! wait_for_backend_ready "$backend_url"; then
          echo "Backend health check failed for compose: $compose_file"
          docker compose -f "$compose_file" logs "$backend_service" || true
          status=1
        elif [ "${REFARCH_SKIP_SMOKE:-0}" != "1" ]; then
          run_smoke "$compose_file" "$backend_url" || status=$?
        fi
      fi
    fi
  else
    docker compose -f "$compose_file" up --build --abort-on-container-exit || status=$?
  fi

  docker compose -f "$compose_file" down --remove-orphans -v >/dev/null 2>&1 || true

  if [ "$status" -ne 0 ]; then
    echo "Reference architecture compose failed: $compose_file (exit code $status)"
    exit "$status"
  fi

done

echo "All reference architecture compose runs completed successfully."
