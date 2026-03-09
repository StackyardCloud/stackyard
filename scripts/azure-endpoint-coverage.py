#!/usr/bin/env python3
"""
Azure endpoint coverage aggregator for Stackyard.

This script aggregates Azure contract/io/doc coverage signals into a single
provider-level endpoint validation report with a consistent CLI surface.
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

from endpoint_live_smoke import HTTPCheck, run_compose_cleanup, run_live_smoke_suite


SCRIPT_DIR = Path(__file__).resolve().parent
CONTRACT_SCRIPT = SCRIPT_DIR / "azure-contract-coverage.py"
IO_SCRIPT = SCRIPT_DIR / "azure-io-contract-coverage.py"
DOC_SCRIPT = SCRIPT_DIR / "azure-doc-contract-coverage.py"
REPO_ROOT = SCRIPT_DIR.parent
AZURE_SHARED_KEY = "SharedKey devstoreaccount1:signature"


def _json_payload(body: bytes) -> dict[str, object]:
    if not body.strip():
        return {}
    payload = json.loads(body.decode("utf-8"))
    return payload if isinstance(payload, dict) else {}


def _blob_create_validator(status: int, _headers: dict[str, str], _body: bytes) -> tuple[bool, str]:
    if status in {201, 409}:
        return True, f"container route returned HTTP {status}"
    return False, f"expected 201 or 409, got {status}"


def _blob_put_validator(status: int, headers: dict[str, str], _body: bytes) -> tuple[bool, str]:
    if status != 201:
        return False, f"expected 201, got {status}"
    if not headers.get("etag"):
        return False, "missing ETag header"
    return True, "blob write returned Created with ETag"


def _blob_get_validator(status: int, headers: dict[str, str], body: bytes) -> tuple[bool, str]:
    if status != 200:
        return False, f"expected 200, got {status}"
    if headers.get("x-ms-meta-env") != "smoke":
        return False, f"expected x-ms-meta-env=smoke, got {headers.get('x-ms-meta-env')!r}"
    if not headers.get("content-type", "").lower().startswith("application/json"):
        return False, f"expected application/json content type, got {headers.get('content-type')!r}"
    if body.decode("utf-8") != '{"version":"live"}':
        return False, f"unexpected blob body: {body.decode('utf-8')!r}"
    return True, "blob read preserved body and metadata"


def _queue_create_validator(status: int, _headers: dict[str, str], _body: bytes) -> tuple[bool, str]:
    if status in {201, 409}:
        return True, f"queue route returned HTTP {status}"
    return False, f"expected 201 or 409, got {status}"


def _queue_enqueue_validator(status: int, _headers: dict[str, str], body: bytes) -> tuple[bool, str]:
    if status != 201:
        return False, f"expected 201, got {status}"
    payload = _json_payload(body)
    if not payload.get("messageId") or not payload.get("popReceipt"):
        return False, f"missing messageId/popReceipt in payload: {payload!r}"
    return True, "queue enqueue returned message metadata"


def _queue_invalid_dequeue_validator(status: int, _headers: dict[str, str], body: bytes) -> tuple[bool, str]:
    if status != 400:
        return False, f"expected 400, got {status}"
    payload = _json_payload(body)
    if payload.get("error") != "InvalidQueryParameterValue":
        return False, f"expected InvalidQueryParameterValue, got {payload!r}"
    return True, "queue validation rejected invalid numofmessages"


def _keyvault_set_validator(status: int, _headers: dict[str, str], body: bytes) -> tuple[bool, str]:
    if status != 200:
        return False, f"expected 200, got {status}"
    payload = _json_payload(body)
    if payload.get("name") != "api-token" or payload.get("value") != "live-smoke-token":
        return False, f"unexpected keyvault payload: {payload!r}"
    return True, "keyvault set secret returned latest value"


def _keyvault_get_validator(status: int, _headers: dict[str, str], body: bytes) -> tuple[bool, str]:
    if status != 200:
        return False, f"expected 200, got {status}"
    payload = _json_payload(body)
    if payload.get("value") != "live-smoke-token":
        return False, f"expected live-smoke-token, got {payload!r}"
    return True, "keyvault get secret returned stored value"


def _keyvault_missing_validator(status: int, _headers: dict[str, str], body: bytes) -> tuple[bool, str]:
    if status != 404:
        return False, f"expected 404, got {status}"
    payload = _json_payload(body)
    if payload.get("error") != "SecretNotFound":
        return False, f"expected SecretNotFound, got {payload!r}"
    return True, "keyvault missing secret returned not found"


AZURE_LIVE_SMOKE_CHECKS: tuple[HTTPCheck, ...] = (
    HTTPCheck(
        service="blob",
        name="create_container",
        method="PUT",
        path="/azure/devstoreaccount1/live-smoke-artifacts?restype=container",
        headers={"Authorization": AZURE_SHARED_KEY},
        validator=_blob_create_validator,
    ),
    HTTPCheck(
        service="blob",
        name="put_blob",
        method="PUT",
        path="/azure/devstoreaccount1/live-smoke-artifacts/releases/live.json",
        headers={
            "Authorization": AZURE_SHARED_KEY,
            "Content-Type": "application/json",
            "x-ms-meta-env": "smoke",
        },
        body=b'{"version":"live"}',
        validator=_blob_put_validator,
    ),
    HTTPCheck(
        service="blob",
        name="get_blob",
        method="GET",
        path="/azure/devstoreaccount1/live-smoke-artifacts/releases/live.json",
        headers={"Authorization": AZURE_SHARED_KEY},
        validator=_blob_get_validator,
    ),
    HTTPCheck(
        service="queue",
        name="create_queue",
        method="PUT",
        path="/azure/queue/devstoreaccount1/live-smoke-queue",
        headers={"Authorization": AZURE_SHARED_KEY},
        validator=_queue_create_validator,
    ),
    HTTPCheck(
        service="queue",
        name="enqueue_message",
        method="POST",
        path="/azure/queue/devstoreaccount1/live-smoke-queue/messages",
        headers={"Authorization": AZURE_SHARED_KEY},
        body=b"smoke-task",
        validator=_queue_enqueue_validator,
    ),
    HTTPCheck(
        service="queue",
        name="invalid_dequeue_query",
        method="POST",
        path="/azure/queue/devstoreaccount1/live-smoke-queue/messages/dequeue?numofmessages=bad",
        headers={"Authorization": AZURE_SHARED_KEY},
        validator=_queue_invalid_dequeue_validator,
    ),
    HTTPCheck(
        service="keyvault",
        name="set_secret",
        method="PUT",
        path="/azure/keyvault/live-smoke-vault/secrets/api-token",
        headers={
            "Authorization": AZURE_SHARED_KEY,
            "Content-Type": "application/json",
        },
        body=b'{"value":"live-smoke-token"}',
        validator=_keyvault_set_validator,
    ),
    HTTPCheck(
        service="keyvault",
        name="get_secret",
        method="GET",
        path="/azure/keyvault/live-smoke-vault/secrets/api-token",
        headers={"Authorization": AZURE_SHARED_KEY},
        validator=_keyvault_get_validator,
    ),
    HTTPCheck(
        service="keyvault",
        name="missing_secret",
        method="GET",
        path="/azure/keyvault/live-smoke-vault/secrets/missing",
        headers={"Authorization": AZURE_SHARED_KEY},
        validator=_keyvault_missing_validator,
    ),
)


def parse_fail_on(raw: str) -> set[str]:
    out: set[str] = set()
    for item in raw.split(","):
        token = item.strip().lower()
        if not token or token == "none":
            continue
        if token == "strict":
            token = "any"
        out.add(token)
    return out


def run_json_script(script: Path, service: str, require_service: str) -> tuple[int, dict[str, object], str]:
    command = [sys.executable, str(script), "--service", service, "--format", "json", "--fail-on", "none"]
    if require_service:
        command.extend(["--require-service", require_service])
    cp = subprocess.run(command, check=False, text=True, capture_output=True)
    payload: dict[str, object] = {}
    if cp.stdout.strip():
        try:
            payload = json.loads(cp.stdout)
        except json.JSONDecodeError:
            payload = {}
    detail = (cp.stderr or cp.stdout or "").strip()
    return cp.returncode, payload, detail


def write_report_json(path: str, payload: dict[str, object]) -> None:
    if not path:
        return
    report_path = Path(path)
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(json.dumps(payload, indent=2, sort_keys=True), encoding="utf-8")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--service", default="*", help="Service filter (name or glob).")
    parser.add_argument(
        "--require-service",
        default="",
        help="Fail if the specified service is not discovered after filtering.",
    )
    parser.add_argument("--format", choices=["text", "json"], default="text", help="Output format.")
    parser.add_argument("--json", action="store_true", help="Alias for --format json.")
    parser.add_argument("--verbose", action="store_true", help="Include per-service detail in text output.")
    parser.add_argument("--report-json", default="", help="Optional path to write JSON output payload.")
    parser.add_argument("--list-services", action="store_true", help="List discovered services and exit.")
    parser.add_argument(
        "--fail-on",
        default="none",
        help="Comma-separated fail gates: contract,io,docs,live,any",
    )
    parser.add_argument("--live-smoke", action="store_true", help="Run curated live HTTP smoke checks against Stackyard.")
    parser.add_argument("--endpoint-url", default="http://127.0.0.1:4566", help="Base URL for live smoke checks.")
    parser.add_argument("--no-start-stackyard", action="store_true", help="Do not auto-start Stackyard via Docker Compose.")
    parser.add_argument(
        "--rebuild-stackyard",
        dest="rebuild_stackyard",
        action="store_true",
        help="Rebuild Stackyard before starting for live smoke checks (default).",
    )
    parser.add_argument(
        "--no-rebuild-stackyard",
        dest="rebuild_stackyard",
        action="store_false",
        help="Start Stackyard without rebuilding for live smoke checks.",
    )
    parser.set_defaults(rebuild_stackyard=True)
    parser.add_argument(
        "--stackyard-compose-file",
        default=str(REPO_ROOT / "docker-compose.yml"),
        help="Compose file used to start Stackyard for live smoke checks.",
    )
    parser.add_argument("--stackyard-service", default="stackyard", help="Compose service name for Stackyard.")
    parser.add_argument("--stackyard-health-path", default="/_stackyard/health", help="Health path to wait on.")
    parser.add_argument(
        "--stackyard-health-attempts",
        type=int,
        default=30,
        help="Max health check attempts after starting Stackyard.",
    )
    parser.add_argument(
        "--stackyard-health-delay-sec",
        type=float,
        default=1.0,
        help="Seconds to wait between Stackyard health checks.",
    )
    parser.add_argument(
        "--live-smoke-timeout-sec",
        type=float,
        default=5.0,
        help="Per-request timeout for live smoke checks.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.json:
        args.format = "json"

    if args.list_services:
        command = [sys.executable, str(CONTRACT_SCRIPT), "--service", args.service, "--format", args.format, "--list-services"]
        if args.require_service:
            command.extend(["--require-service", args.require_service])
        cp = subprocess.run(command, check=False)
        return cp.returncode

    rc_contract, payload_contract, detail_contract = run_json_script(CONTRACT_SCRIPT, args.service, args.require_service)
    rc_io, payload_io, detail_io = run_json_script(IO_SCRIPT, args.service, args.require_service)
    rc_doc, payload_doc, detail_doc = run_json_script(DOC_SCRIPT, args.service, args.require_service)

    for code in (rc_contract, rc_io, rc_doc):
        if code == 2:
            if detail_contract:
                print(detail_contract, file=sys.stderr)
            if detail_io:
                print(detail_io, file=sys.stderr)
            if detail_doc:
                print(detail_doc, file=sys.stderr)
            return 2

    contract_rows = payload_contract.get("services", []) if isinstance(payload_contract, dict) else []
    io_rows = payload_io.get("services", []) if isinstance(payload_io, dict) else []
    doc_rows = payload_doc.get("services", []) if isinstance(payload_doc, dict) else []

    contract_map: dict[str, bool] = {}
    for row in contract_rows if isinstance(contract_rows, list) else []:
        if not isinstance(row, dict):
            continue
        service = str(row.get("service", "")).strip()
        if not service:
            continue
        contract_map[service] = bool(
            row.get("strict_all_three")
            if "strict_all_three" in row
            else row.get("request_validation")
            and row.get("typed_success_fixtures")
            and row.get("negative_contract_tests")
        )

    io_map: dict[str, bool] = {}
    for row in io_rows if isinstance(io_rows, list) else []:
        if not isinstance(row, dict):
            continue
        service = str(row.get("service", "")).strip()
        if not service:
            continue
        io_map[service] = bool(
            row.get("strict_all_four")
            if "strict_all_four" in row
            else row.get("input_validation_impl")
            and row.get("output_fixture_impl")
            and row.get("input_validation_tests")
            and row.get("output_shape_tests")
        )

    doc_map: dict[str, bool] = {}
    for row in doc_rows if isinstance(doc_rows, list) else []:
        if not isinstance(row, dict):
            continue
        service = str(row.get("service", "")).strip()
        if not service:
            continue
        doc_map[service] = bool(row.get("strict_all", False))

    services = sorted(set(contract_map) | set(io_map) | set(doc_map))
    service_rows: list[dict[str, object]] = []
    for service in services:
        contract_ok = contract_map.get(service, False)
        io_ok = io_map.get(service, False)
        doc_ok = doc_map.get(service, False)
        service_rows.append(
            {
                "service": service,
                "contract_ok": contract_ok,
                "io_ok": io_ok,
                "doc_ok": doc_ok,
                "strict_all": contract_ok and io_ok and doc_ok,
            }
        )

    summary = {
        "total": len(service_rows),
        "contract_ok": sum(1 for row in service_rows if row["contract_ok"]),
        "io_ok": sum(1 for row in service_rows if row["io_ok"]),
        "doc_ok": sum(1 for row in service_rows if row["doc_ok"]),
        "strict_all": sum(1 for row in service_rows if row["strict_all"]),
    }

    live_smoke = {
        "provider": "azure",
        "enabled": False,
        "endpoint_url": args.endpoint_url,
        "selected_services": [],
        "selected_checks": 0,
        "passed_checks": 0,
        "failed_checks": 0,
        "strict": True,
        "skipped": True,
        "startup_error": "",
        "checks": [],
    }
    cleanup_stackyard = False
    if args.live_smoke:
        selected_checks = [check for check in AZURE_LIVE_SMOKE_CHECKS if check.service in services]
        live_smoke, cleanup_stackyard = run_live_smoke_suite(
            provider="azure",
            endpoint_url=args.endpoint_url,
            checks=selected_checks,
            start_stackyard=not args.no_start_stackyard,
            rebuild_stackyard=args.rebuild_stackyard,
            compose_file=args.stackyard_compose_file,
            stackyard_service=args.stackyard_service,
            health_path=args.stackyard_health_path,
            health_attempts=args.stackyard_health_attempts,
            health_delay_sec=args.stackyard_health_delay_sec,
            timeout_sec=args.live_smoke_timeout_sec,
            env=os.environ.copy(),
        )

    payload = {
        "provider": "azure",
        "mode": "endpoint_coverage",
        "service_selector": args.service,
        "required_service": args.require_service,
        "summary": summary,
        "services": service_rows,
        "component_exit_codes": {
            "contract": rc_contract,
            "io": rc_io,
            "doc": rc_doc,
        },
        "live_smoke": live_smoke,
    }
    try:
        write_report_json(args.report_json, payload)

        if args.format == "json":
            print(json.dumps(payload, indent=2, sort_keys=True))
        else:
            print("Azure endpoint coverage (aggregated)")
            print(json.dumps(summary, indent=2, sort_keys=True))
            if args.live_smoke:
                print("")
                print("Live smoke")
                print(
                    json.dumps(
                        {
                            "selected_checks": live_smoke["selected_checks"],
                            "passed_checks": live_smoke["passed_checks"],
                            "failed_checks": live_smoke["failed_checks"],
                            "strict": live_smoke["strict"],
                            "selected_services": live_smoke["selected_services"],
                            "startup_error": live_smoke["startup_error"],
                        },
                        indent=2,
                        sort_keys=True,
                    )
                )
            if args.verbose:
                print("")
                print("Per-service details")
                for row in service_rows:
                    print(
                        f"- {row['service']}: contract={row['contract_ok']} io={row['io_ok']} "
                        f"doc={row['doc_ok']} strict_all={row['strict_all']}"
                    )
                if args.live_smoke and live_smoke["checks"]:
                    print("")
                    print("Live smoke checks")
                    for row in live_smoke["checks"]:
                        print(
                            f"- {row['service']}/{row['name']}: ok={row['ok']} "
                            f"http_status={row['http_status']} detail={row['detail']}"
                        )

        fail_on = parse_fail_on(args.fail_on)
        live_failed = args.live_smoke and not bool(live_smoke.get("strict", True))
        if not fail_on:
            return 1 if live_failed or any(code == 1 for code in (rc_contract, rc_io, rc_doc)) else 0

        checks = {
            "contract": any(not row["contract_ok"] for row in service_rows),
            "io": any(not row["io_ok"] for row in service_rows),
            "docs": any(not row["doc_ok"] for row in service_rows),
            "live": live_failed,
            "any": any(not row["strict_all"] for row in service_rows) or live_failed,
        }
        should_fail = any(checks.get(token, False) for token in fail_on)
        return 1 if should_fail else 0
    finally:
        if cleanup_stackyard:
            ok, detail = run_compose_cleanup(args.stackyard_compose_file, args.stackyard_service, os.environ.copy())
            if not ok:
                print(f"warning: failed to remove Stackyard container: {detail}", file=sys.stderr)


if __name__ == "__main__":
    raise SystemExit(main())
