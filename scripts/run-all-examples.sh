#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

wait_for_stackyard_health() {
  local attempts=60
  local delay=1
  local url="http://localhost:4566/_stackyard/health"
  local i
  for i in $(seq 1 "$attempts"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep "$delay"
  done
  return 1
}

compose_files=()
if [ "${RUN_ALL_EXAMPLES:-0}" = "1" ]; then
  while IFS= read -r compose_file; do
    compose_files+=("$compose_file")
  done < <(find examples -mindepth 2 -maxdepth 3 -type f -name 'docker-compose.yml' | sort)
else
  service_dirs=()
  while IFS= read -r service_dir; do
    service_dirs+=("$service_dir")
  done < <(find examples -mindepth 1 -maxdepth 1 -type d | sort)

  preferred_variant="${EXAMPLE_VARIANT:-advanced}"
  for service_dir in "${service_dirs[@]}"; do
    service_name="$(basename "$service_dir")"
    preferred_compose="$service_dir/$service_name-$preferred_variant/docker-compose.yml"
    if [ -f "$preferred_compose" ]; then
      compose_files+=("$preferred_compose")
      continue
    fi

    if [ "$preferred_variant" = "advanced" ]; then
      fallback_variant="basic"
    else
      fallback_variant="advanced"
    fi
    fallback_compose="$service_dir/$service_name-$fallback_variant/docker-compose.yml"
    if [ -f "$fallback_compose" ]; then
      compose_files+=("$fallback_compose")
      continue
    fi

    first_compose=""
    while IFS= read -r compose_file; do
      first_compose="$compose_file"
      break
    done < <(find "$service_dir" -mindepth 2 -maxdepth 2 -type f -name 'docker-compose.yml' | sort)

    if [ -n "$first_compose" ]; then
      compose_files+=("$first_compose")
    fi
  done
fi

if [ "${#compose_files[@]}" -eq 0 ]; then
  echo "No example docker-compose files found under examples/*/*/docker-compose.yml"
  exit 0
fi

for compose_file in "${compose_files[@]}"; do
  echo "==> Running example compose: $compose_file"

  compose_images=()
  while IFS= read -r image; do
    if [ -n "$image" ]; then
      compose_images+=("$image")
    fi
  done < <(docker compose -f "$compose_file" config --images 2>/dev/null || true)

  services=()
  while IFS= read -r service; do
    services+=("$service")
  done < <(docker compose -f "$compose_file" config --services)
  if [ "${#services[@]}" -eq 0 ]; then
    echo "No services found in $compose_file"
    exit 1
  fi

  exit_service=""
  for service in "${services[@]}"; do
    if [ "$service" != "stackyard" ]; then
      exit_service="$service"
      break
    fi
  done
  if [ -z "$exit_service" ]; then
    exit_service="${services[0]}"
  fi

  has_stackyard=0
  non_stackyard_count=0
  for service in "${services[@]}"; do
    if [ "$service" = "stackyard" ]; then
      has_stackyard=1
    else
      non_stackyard_count=$((non_stackyard_count + 1))
    fi
  done

  docker compose -f "$compose_file" down --remove-orphans -v >/dev/null 2>&1 || true

  status=0
  if [ "$has_stackyard" -eq 1 ] && [ "$non_stackyard_count" -eq 1 ] && [ "$exit_service" != "stackyard" ]; then
    docker rm -f stackyard >/dev/null 2>&1 || true
    docker compose -f "$compose_file" up -d --build stackyard || status=$?
    if [ "$status" -eq 0 ] && ! wait_for_stackyard_health; then
      echo "Stackyard health check failed for compose: $compose_file"
      docker compose -f "$compose_file" logs stackyard || true
      status=1
    fi
    if [ "$status" -eq 0 ]; then
      docker compose -f "$compose_file" up --build --no-deps --abort-on-container-exit --exit-code-from "$exit_service" "$exit_service" || status=$?
    fi
  else
    docker compose -f "$compose_file" up --build --abort-on-container-exit --exit-code-from "$exit_service" || status=$?
  fi

  docker compose -f "$compose_file" down --remove-orphans -v >/dev/null 2>&1 || true

  if [ "$status" -ne 0 ]; then
    echo "Example compose failed: $compose_file (exit code $status)"
    exit "$status"
  fi

  for image in "${compose_images[@]}"; do
    if [ -n "$image" ]; then
      docker image rm -f "$image" >/dev/null 2>&1 || true
    fi
  done

done

echo "All example docker-compose runs completed successfully."
