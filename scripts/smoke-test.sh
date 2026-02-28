#!/usr/bin/env bash
set -euo pipefail

base_url="${STACKYARD_URL:-http://localhost:4566}"
verbose=0
run_aws=0
reset_container=0
health_attempts="${STACKYARD_HEALTH_ATTEMPTS:-30}"
health_delay_sec="${STACKYARD_HEALTH_DELAY_SEC:-1}"

while getopts ":var" opt; do
  case "$opt" in
    v) verbose=1 ;;
    a) run_aws=1 ;;
    r) reset_container=1 ;;
    *) echo "Usage: $0 [-v] [-a] [-r]" >&2; exit 1 ;;
  esac
done
shift $((OPTIND - 1))

check_status() {
  local status="$1"
  shift
  for allowed in "$@"; do
    if [[ "${status}" == "${allowed}" ]]; then
      return 0
    fi
  done
  echo "Unexpected HTTP status: ${status}" >&2
  return 1
}

curl_check() {
  local label="$1"
  shift
  local allowed=()
  while [[ "$1" =~ ^[0-9]{3}$ ]]; do
    allowed+=("$1")
    shift
  done

  if [[ "$verbose" -eq 1 ]]; then
    echo "+ curl $*"
  fi

  local output
  output=$(curl -sS -w "\n%{http_code}" "$@")
  local status
  status=$(printf '%s' "$output" | tail -n 1)
  local body
  body=$(printf '%s' "$output" | sed '$d')

  if [[ "$verbose" -eq 1 && -n "$body" ]]; then
    echo "$body"
  fi

  echo "$label"
  check_status "$status" "${allowed[@]}"
}

wait_for_health() {
  local attempt=1
  while [[ "$attempt" -le "$health_attempts" ]]; do
    if [[ "$verbose" -eq 1 ]]; then
      if curl -fsS "${base_url}/_stackyard/health" >/dev/null; then
        echo "Health check..."
        return 0
      fi
      echo "Waiting for health (${attempt}/${health_attempts})..."
    else
      if curl -fsS "${base_url}/_stackyard/health" >/dev/null 2>&1; then
        echo "Health check..."
        return 0
      fi
    fi
    sleep "$health_delay_sec"
    attempt=$((attempt + 1))
  done

  echo "Health check failed after ${health_attempts} attempts." >&2
  return 1
}

echo "Stackyard smoke test against ${base_url}"

cleanup_on_exit() {
  local status=$?
  if [[ "$reset_container" -eq 1 && "$status" -eq 0 ]]; then
    if command -v docker >/dev/null 2>&1; then
      if [[ "$verbose" -eq 1 ]]; then
        echo "+ docker compose down"
      fi
      docker compose down --remove-orphans
    else
      echo "Docker not found; cannot clean up container." >&2
    fi
  fi
  exit "$status"
}

if [[ "$reset_container" -eq 1 ]]; then
  if command -v docker >/dev/null 2>&1; then
    if [[ "$verbose" -eq 1 ]]; then
      echo "+ docker compose down"
      echo "+ docker compose up -d --build"
    fi
    docker compose down --remove-orphans
    docker compose up -d --build
    trap cleanup_on_exit EXIT
  else
    echo "Docker not found; cannot reset container." >&2
    exit 1
  fi
fi

wait_for_health

curl_check "Create bucket..." 201 409 -X PUT "${base_url}/s3/buckets/demo"

curl_check "List buckets..." 200 "${base_url}/s3/buckets"

curl_check "Put object..." 201 -X PUT --data "hello stackyard" \
  -H "Content-Type: text/plain" \
  "${base_url}/s3/objects/demo/notes/hello.txt"

curl_check "Get object..." 200 "${base_url}/s3/objects/demo/notes/hello.txt"

curl_check "Create queue..." 201 409 -X PUT "${base_url}/sqs/queues/jobs"

curl_check "Send message..." 201 -X POST "${base_url}/sqs/messages/jobs" \
  -H "Content-Type: application/json" \
  -d '{"body":"run build"}'

curl_check "Receive message..." 200 -X POST "${base_url}/sqs/messages/jobs/receive" \
  -H "Content-Type: application/json" \
  -d '{"max_messages":1}'

if [[ "$run_aws" -eq 1 ]]; then
  if ! command -v aws >/dev/null 2>&1; then
    echo "AWS CLI not found; skipping SigV4/XML smoke tests." >&2
  else
    export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-stackyard}"
    export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-stackyard}"
    export AWS_REGION="${AWS_REGION:-us-east-1}"
    export AWS_S3_FORCE_PATH_STYLE=true

    bucket="demo-aws-$(date +%s)"
    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api create-bucket --bucket ${bucket}"
    fi
    aws --endpoint-url "${base_url}" s3api create-bucket --bucket "${bucket}" >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api list-objects-v2 --bucket ${bucket}"
    fi
    aws --endpoint-url "${base_url}" s3api list-objects-v2 --bucket "${bucket}" >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api put-object --bucket ${bucket} --key notes/hello.txt --body ./README.md"
    fi
    aws --endpoint-url "${base_url}" s3api put-object --bucket "${bucket}" --key notes/hello.txt --body ./README.md >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api get-object --bucket ${bucket} --key notes/hello.txt /tmp/stackyard-hello.txt"
    fi
    aws --endpoint-url "${base_url}" s3api get-object --bucket "${bucket}" --key notes/hello.txt /tmp/stackyard-hello.txt >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api head-bucket --bucket ${bucket}"
    fi
    aws --endpoint-url "${base_url}" s3api head-bucket --bucket "${bucket}" >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api head-object --bucket ${bucket} --key notes/hello.txt"
    fi
    aws --endpoint-url "${base_url}" s3api head-object --bucket "${bucket}" --key notes/hello.txt >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api copy-object --copy-source ${bucket}/notes/hello.txt --bucket ${bucket} --key notes/hello-copy.txt"
    fi
    aws --endpoint-url "${base_url}" s3api copy-object --copy-source "${bucket}/notes/hello.txt" --bucket "${bucket}" --key notes/hello-copy.txt >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3 presign s3://${bucket}/notes/hello.txt --expires-in 300"
    fi
    presigned=$(aws --endpoint-url "${base_url}" s3 presign "s3://${bucket}/notes/hello.txt" --expires-in 300)

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ curl -H 'Range: bytes=0-9' ${presigned}"
    fi
    range_status=$(curl -sS -o /dev/null -w "%{http_code}" -H "Range: bytes=0-9" "${presigned}")
    if [[ "$range_status" != "206" ]]; then
      echo "Unexpected range status: ${range_status}" >&2
      exit 1
    fi

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api delete-object --bucket ${bucket} --key notes/hello-copy.txt"
    fi
    aws --endpoint-url "${base_url}" s3api delete-object --bucket "${bucket}" --key notes/hello-copy.txt >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api delete-objects --bucket ${bucket} --delete file:///tmp/stackyard-delete-objects.json"
    fi
    cat >/tmp/stackyard-delete-objects.json <<EOF
{
  "Objects": [
    { "Key": "notes/hello.txt" }
  ],
  "Quiet": true
}
EOF
    aws --endpoint-url "${base_url}" s3api delete-objects --bucket "${bucket}" --delete file:///tmp/stackyard-delete-objects.json >/dev/null

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api create-multipart-upload --bucket ${bucket} --key notes/multipart.txt"
    fi
    upload_id=$(aws --endpoint-url "${base_url}" s3api create-multipart-upload --bucket "${bucket}" --key notes/multipart.txt --query UploadId --output text)

    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api upload-part --bucket ${bucket} --key notes/multipart.txt --part-number 1 --upload-id ${upload_id} --body ./README.md"
    fi
    part_etag=$(aws --endpoint-url "${base_url}" s3api upload-part --bucket "${bucket}" --key notes/multipart.txt --part-number 1 --upload-id "${upload_id}" --body ./README.md --query ETag --output text)

    cat >/tmp/stackyard-complete-mpu.json <<EOF
{
  "Parts": [
    { "ETag": ${part_etag}, "PartNumber": 1 }
  ]
}
EOF
    if [[ "$verbose" -eq 1 ]]; then
      echo "+ aws --endpoint-url ${base_url} s3api complete-multipart-upload --bucket ${bucket} --key notes/multipart.txt --upload-id ${upload_id} --multipart-upload file:///tmp/stackyard-complete-mpu.json"
    fi
    aws --endpoint-url "${base_url}" s3api complete-multipart-upload --bucket "${bucket}" --key notes/multipart.txt --upload-id "${upload_id}" --multipart-upload file:///tmp/stackyard-complete-mpu.json >/dev/null
  fi
fi

echo "Smoke tests passed."
