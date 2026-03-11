#!/usr/bin/env python3
"""
AWS provider input/output contract coverage wrapper for Stackyard.

This wrapper aligns AWS IO contract checks with GCP/Azure provider coverage
scripts while delegating execution to scripts/awscli-endpoint-coverage.py.
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

from aws_service_shards import (
    aws_endpoint_service_weights,
    discover_services,
    filter_services,
    select_shard,
    validate_shard_args,
)


SCRIPT_DIR = Path(__file__).resolve().parent
AWSCLI_COVERAGE_SCRIPT = SCRIPT_DIR / "awscli-endpoint-coverage.py"

STRICT_FAIL_ON = (
    "not_implemented,service_error,client_error,contract_error,"
    "transport_error,skeleton_error,unknown_error,auth_error"
)
KNOWN_STATUS_TOKENS = {
    "not_implemented",
    "service_error",
    "client_error",
    "contract_error",
    "transport_error",
    "skeleton_error",
    "unknown_error",
    "auth_error",
    "unavailable_in_cli",
    "precondition_tolerated",
}


def parse_args() -> tuple[argparse.Namespace, list[str]]:
    parser = argparse.ArgumentParser(description="Analyze AWS input/output contract coverage.")
    parser.add_argument(
        "--service",
        default="*",
        help="Service filter passed to AWS endpoint coverage include-services.",
    )
    parser.add_argument(
        "--require-service",
        default="",
        help="Fail if the specified service resolves to zero endpoints.",
    )
    parser.add_argument(
        "--fail-on",
        default="none",
        help=(
            "Comma-separated fail gates/statuses. Aliases: none,any,strict. "
            "Statuses: not_implemented,service_error,client_error,contract_error,"
            "transport_error,skeleton_error,unknown_error,auth_error,unavailable_in_cli,precondition_tolerated"
        ),
    )
    parser.add_argument(
        "--format",
        choices=("text", "json"),
        default="text",
        help="Output format.",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Alias for --format json.",
    )
    parser.add_argument(
        "--quiet",
        action="store_true",
        help="Forward quiet mode to the AWS endpoint coverage runner.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="List selected endpoints without executing calls.",
    )
    parser.add_argument(
        "--endpoint-url",
        default=os.getenv("STACKYARD_URL", "http://localhost:4566"),
        help="Stackyard/AWS endpoint URL.",
    )
    parser.add_argument(
        "--region",
        default=os.getenv("AWS_REGION", "us-east-1"),
        help="AWS region for CLI calls.",
    )
    parser.add_argument(
        "--aws-bin",
        default=os.getenv("AWS_BIN", "aws"),
        help="AWS CLI binary path.",
    )
    parser.add_argument(
        "--report-json",
        default="",
        help="Optional path for the underlying AWS coverage JSON report.",
    )
    parser.add_argument(
        "--no-start-stackyard",
        action="store_true",
        help="Do not auto-start Stackyard before checks.",
    )
    parser.add_argument(
        "--no-rebuild-stackyard",
        action="store_true",
        help="When auto-starting Stackyard, do not rebuild the image.",
    )
    parser.add_argument(
        "--strict-ec2-stateful-errors",
        action="store_true",
        help="Treat tolerated EC2 stateful precondition errors as hard failures.",
    )
    parser.add_argument(
        "--list-services",
        action="store_true",
        help="List discovered services and exit.",
    )
    parser.add_argument(
        "--shard-count",
        type=int,
        default=1,
        help="Split the selected services into N balanced shards.",
    )
    parser.add_argument(
        "--shard-index",
        type=int,
        default=0,
        help="Zero-based shard index when --shard-count is greater than 1.",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Include extra context in text output.",
    )
    return parser.parse_known_args()


def build_base_command(args: argparse.Namespace, service_override: str | None = None) -> list[str]:
    service_selector = args.service if service_override is None else service_override
    command = [
        sys.executable,
        str(AWSCLI_COVERAGE_SCRIPT),
        "--contract-mode",
        "--endpoint-url",
        args.endpoint_url,
        "--region",
        args.region,
        "--aws-bin",
        args.aws_bin,
    ]
    if service_selector and service_selector != "*":
        command.extend(["--include-services", service_selector])
    if args.no_start_stackyard:
        command.append("--no-start-stackyard")
    if args.no_rebuild_stackyard:
        command.append("--no-rebuild-stackyard")
    if args.quiet:
        command.append("--quiet")
    return command


def extract_selected_endpoint_count(output: str) -> int | None:
    match = re.search(r"Selected\s+(\d+)\s+endpoints\.", output)
    if match:
        return int(match.group(1))
    if "No endpoints selected." in output:
        return 0
    return None


def ensure_required_service(args: argparse.Namespace) -> int:
    required_service = args.require_service.strip()
    if not required_service:
        return 0

    command = build_base_command(args, service_override=required_service)
    command.extend(["--dry-run", "--max-endpoints", "1", "--fail-on", ""])
    cp = subprocess.run(command, check=False, text=True, capture_output=True)
    combined = (cp.stdout or "") + "\n" + (cp.stderr or "")

    if cp.returncode != 0:
        print(
            f"aws-io-contract-coverage: failed to validate required service {required_service!r}",
            file=sys.stderr,
        )
        if combined.strip():
            print(combined.strip(), file=sys.stderr)
        return 2

    selected = extract_selected_endpoint_count(combined)
    if selected is None:
        print(
            "aws-io-contract-coverage: unable to determine selected endpoint count while checking "
            f"required service {required_service!r}",
            file=sys.stderr,
        )
        if combined.strip():
            print(combined.strip(), file=sys.stderr)
        return 2
    if selected == 0:
        print(
            f"aws-io-contract-coverage: required service {required_service!r} not found",
            file=sys.stderr,
        )
        return 2
    return 0


def resolve_services(args: argparse.Namespace) -> list[str]:
    discovered = discover_services()
    selected = filter_services(discovered, args.service)
    if args.require_service and args.require_service not in discovered:
        raise ValueError(f"required service {args.require_service!r} not discovered")
    if args.require_service and args.require_service not in selected:
        raise ValueError(f"required service {args.require_service!r} not matched by selector")
    if args.shard_count == 1:
        return selected
    weights = aws_endpoint_service_weights(selected)
    sharded = select_shard(selected, weights, args.shard_count, args.shard_index)
    if args.require_service and args.require_service not in sharded:
        raise ValueError(f"required service {args.require_service!r} not matched by shard")
    return sharded


def load_report(path: Path) -> object | None:
    if not path.exists():
        return None
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None


def strip_passthrough_report_json(args: list[str]) -> list[str]:
    out: list[str] = []
    skip_next = False
    for token in args:
        if skip_next:
            skip_next = False
            continue
        if token == "--report-json":
            skip_next = True
            continue
        if token.startswith("--report-json="):
            continue
        out.append(token)
    return out


def parse_fail_on(raw: str) -> tuple[str, bool]:
    ordered: list[str] = []
    seen: set[str] = set()
    strict_stateful = False
    for item in raw.split(","):
        token = item.strip().lower()
        if not token or token == "none":
            continue
        if token in {"any", "strict"}:
            strict_stateful = True
            for status in STRICT_FAIL_ON.split(","):
                status = status.strip()
                if status and status not in seen:
                    seen.add(status)
                    ordered.append(status)
            continue
        if token in KNOWN_STATUS_TOKENS and token not in seen:
            seen.add(token)
            ordered.append(token)
    return ",".join(ordered), strict_stateful


def run_with_json_output(command: list[str], report_path: Path) -> int:
    cp = subprocess.run(command, check=False, text=True, capture_output=True)
    report_payload = load_report(report_path)

    payload = {
        "provider": "aws",
        "mode": "io_contract",
        "exit_code": cp.returncode,
        "report_path": str(report_path),
        "report": report_payload,
    }
    print(json.dumps(payload, indent=2, sort_keys=True))

    if cp.returncode != 0:
        if cp.stderr:
            print(cp.stderr.strip(), file=sys.stderr)
        elif cp.stdout:
            print(cp.stdout.strip(), file=sys.stderr)
    return cp.returncode


def main() -> int:
    if not AWSCLI_COVERAGE_SCRIPT.exists():
        print(f"missing script: {AWSCLI_COVERAGE_SCRIPT}", file=sys.stderr)
        return 2

    args, passthrough = parse_args()
    if args.json:
        args.format = "json"
    try:
        validate_shard_args(args.shard_count, args.shard_index)
        selected_services = resolve_services(args)
    except ValueError as err:
        print(f"aws-io-contract-coverage: {err}", file=sys.stderr)
        return 2

    if args.list_services:
        rc = 0
        services = selected_services
        detail = ""
        payload = {
            "provider": "aws",
            "mode": "io_contract",
            "service_selector": args.service,
            "selected_services": services,
            "services": services,
            "count": len(services),
            "shard_count": args.shard_count,
            "shard_index": args.shard_index,
        }
        if args.format == "json":
            print(json.dumps(payload, indent=2, sort_keys=True))
            if args.report_json:
                report_path = Path(args.report_json)
                report_path.parent.mkdir(parents=True, exist_ok=True)
                report_path.write_text(json.dumps(payload, indent=2, sort_keys=True), encoding="utf-8")
        else:
            for service in services:
                print(service)
            print(f"total: {len(services)}")
            if args.verbose and detail:
                print("")
                print(detail)
        return 0 if rc == 0 else rc

    if not selected_services:
        payload = {
            "provider": "aws",
            "mode": "io_contract",
            "service_selector": args.service,
            "selected_services": [],
            "count": 0,
            "shard_count": args.shard_count,
            "shard_index": args.shard_index,
        }
        if args.format == "json":
            print(json.dumps(payload, indent=2, sort_keys=True))
        else:
            print("No services selected for this shard.")
            print(json.dumps(payload, indent=2, sort_keys=True))
        return 0

    require_status = ensure_required_service(args)
    if require_status != 0:
        return require_status

    command = build_base_command(args, service_override=",".join(selected_services))
    fail_on_csv, strict_from_mode = parse_fail_on(args.fail_on)
    command.extend(["--fail-on", fail_on_csv])
    if strict_from_mode or args.strict_ec2_stateful_errors:
        command.append("--strict-ec2-stateful-errors")

    if args.dry_run:
        command.append("--dry-run")

    report_path: Path | None = None
    if args.report_json:
        report_path = Path(args.report_json)
        command.extend(["--report-json", str(report_path)])
    elif args.format == "json":
        tmp = tempfile.NamedTemporaryFile(
            prefix="stackyard-aws-io-contract-coverage-",
            suffix=".json",
            delete=False,
        )
        tmp.close()
        report_path = Path(tmp.name)
        command.extend(["--report-json", str(report_path)])

    if report_path is not None:
        passthrough = strip_passthrough_report_json(passthrough)
    command.extend(passthrough)

    try:
        if args.format == "json" and report_path is not None:
            return run_with_json_output(command, report_path)
        cp = subprocess.run(command, check=False)
        return cp.returncode
    finally:
        if args.format == "json" and args.report_json == "" and report_path is not None:
            try:
                report_path.unlink()
            except OSError:
                pass


if __name__ == "__main__":
    raise SystemExit(main())
