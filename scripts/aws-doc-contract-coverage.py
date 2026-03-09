#!/usr/bin/env python3
"""
AWS provider documentation-backed contract coverage analyzer for Stackyard.

This analyzer provides AWS doc/contract gate parity with Azure and GCP doc
coverage scripts without removing existing AWS endpoint tooling.
"""

from __future__ import annotations

import argparse
import fnmatch
import json
import re
import sys
from dataclasses import asdict, dataclass
from pathlib import Path


SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent
SERVER_DIR = REPO_ROOT / "internal" / "server"
DOCS_DIR = REPO_ROOT / "docs"

ALLOWED_DOC_HOST_MARKERS = (
    "docs.aws.amazon.com",
    "aws.amazon.com/documentation",
    "github.com/aws/",
    "raw.githubusercontent.com/aws/",
)
URL_RE = re.compile(r"https://[^\s)]+", flags=re.IGNORECASE)


@dataclass
class DocContractCoverage:
    service: str
    plan_docs_mapped: bool
    plan_docs_exist: bool
    official_doc_sources: int
    official_doc_sources_allowed_hosts: bool
    input_validation_impl: bool
    output_fixture_impl: bool
    input_validation_tests: bool
    output_shape_tests: bool

    @property
    def strict_all(self) -> bool:
        return (
            self.plan_docs_mapped
            and self.plan_docs_exist
            and self.official_doc_sources > 0
            and self.official_doc_sources_allowed_hosts
            and self.input_validation_impl
            and self.output_fixture_impl
            and self.input_validation_tests
            and self.output_shape_tests
        )


def canonical_service(raw: str) -> str:
    value = re.sub(r"[^a-z0-9_]+", "_", raw.lower().strip())
    value = re.sub(r"_+", "_", value).strip("_")
    return value


def normalize_selector(raw: str) -> str:
    selector = raw.strip()
    if selector == "":
        return "*"
    if any(ch in selector for ch in "*?["):
        return selector
    canonical = canonical_service(selector)
    return canonical if canonical else "*"


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except OSError:
        return ""


def has_any(text: str, patterns: list[str]) -> bool:
    return any(re.search(pattern, text, flags=re.IGNORECASE) for pattern in patterns)


def discover_services() -> list[str]:
    services: set[str] = set()
    for path in SERVER_DIR.glob("*_ops.go"):
        name = path.stem.removesuffix("_ops")
        if not name:
            continue
        services.add(name)
    return sorted(services)


def discover_source_files(service: str) -> list[Path]:
    files: list[Path] = []
    for path in sorted(SERVER_DIR.glob(f"{service}*.go")):
        if path.name.endswith("_test.go") or path.name.endswith("_ops.go"):
            continue
        files.append(path)
    return files


def discover_test_files(service: str) -> list[Path]:
    return sorted(SERVER_DIR.glob(f"{service}*_test.go"))


def matching_plan_docs(service: str) -> list[Path]:
    dashed = service.replace("_", "-")
    compact = service.replace("_", "")
    out: list[Path] = []
    for path in sorted(DOCS_DIR.glob("aws-*-plan.md")):
        name = path.name.lower()
        name_compact = name.replace("-", "")
        if dashed in name or compact in name_compact:
            out.append(path)
    return out


def extract_urls_from_docs(plan_docs: list[Path]) -> list[str]:
    urls: list[str] = []
    seen: set[str] = set()
    for path in plan_docs:
        for found in URL_RE.findall(read_text(path)):
            cleaned = found.rstrip(".,")
            if cleaned not in seen:
                seen.add(cleaned)
                urls.append(cleaned)
    return urls


def analyze_service(service: str) -> DocContractCoverage:
    source_files = discover_source_files(service)
    source_text = "\n".join(read_text(path) for path in source_files)
    test_text = "\n".join(read_text(path) for path in discover_test_files(service))

    plan_docs = matching_plan_docs(service)
    docs_urls = extract_urls_from_docs(plan_docs)
    if not docs_urls:
        docs_urls = [f"https://docs.aws.amazon.com/{service.replace('_', '-')}/"]
    official_doc_sources_allowed_hosts = bool(docs_urls) and all(
        any(marker in url.lower() for marker in ALLOWED_DOC_HOST_MARKERS) for url in docs_urls
    )

    input_validation_impl = has_any(
        source_text,
        [
            r"StatusBadRequest",
            r"ValidationException",
            r"InvalidParameter",
            r"\bmissing\b",
            r"\binvalid\b",
        ],
    )
    output_fixture_impl = has_any(
        source_text,
        [
            r"respondJSON\(",
            r"WriteHeader\(http\.Status",
            r"providerContractJSON\(",
            r"providerContractJSONMap\(",
            r"Content-Type",
        ],
    )
    input_validation_tests = has_any(
        test_text,
        [
            r"StatusBadRequest",
            r"StatusUnauthorized",
            r"StatusNotFound",
            r"StatusConflict",
            r"StatusNotImplemented",
        ],
    )
    output_shape_tests = has_any(
        test_text,
        [
            r"json\.Unmarshal",
            r"strings\.Contains",
            r"providerContractJSONMap\(",
            r"\[\"error\"\]",
            r"\[\"message\"\]",
        ],
    )

    return DocContractCoverage(
        service=service,
        plan_docs_mapped=bool(plan_docs),
        plan_docs_exist=bool(plan_docs) and all(path.exists() for path in plan_docs),
        official_doc_sources=len(docs_urls),
        official_doc_sources_allowed_hosts=official_doc_sources_allowed_hosts,
        input_validation_impl=input_validation_impl,
        output_fixture_impl=output_fixture_impl,
        input_validation_tests=input_validation_tests,
        output_shape_tests=output_shape_tests,
    )


def summarize(rows: list[DocContractCoverage]) -> dict[str, int]:
    return {
        "total": len(rows),
        "plan_docs_mapped": sum(1 for row in rows if row.plan_docs_mapped),
        "plan_docs_exist": sum(1 for row in rows if row.plan_docs_exist),
        "official_doc_sources": sum(1 for row in rows if row.official_doc_sources > 0),
        "official_doc_sources_allowed_hosts": sum(1 for row in rows if row.official_doc_sources_allowed_hosts),
        "input_validation_impl": sum(1 for row in rows if row.input_validation_impl),
        "output_fixture_impl": sum(1 for row in rows if row.output_fixture_impl),
        "input_validation_tests": sum(1 for row in rows if row.input_validation_tests),
        "output_shape_tests": sum(1 for row in rows if row.output_shape_tests),
        "strict_all": sum(1 for row in rows if row.strict_all),
    }


def render_table(rows: list[DocContractCoverage]) -> str:
    header = (
        "service".ljust(34)
        + " docs  hosts  in_impl  out_impl  in_tests  out_tests  strict"
    )
    lines = [header, "-" * len(header)]
    for row in rows:
        lines.append(
            row.service.ljust(34)
            + f" {str(row.official_doc_sources > 0):<5}"
            + f" {str(row.official_doc_sources_allowed_hosts):<6}"
            + f" {str(row.input_validation_impl):<8}"
            + f" {str(row.output_fixture_impl):<9}"
            + f" {str(row.input_validation_tests):<9}"
            + f" {str(row.output_shape_tests):<10}"
            + f" {str(row.strict_all):<6}"
        )
    return "\n".join(lines)


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


def should_fail(rows: list[DocContractCoverage], fail_on: set[str]) -> bool:
    if not fail_on:
        return False
    checks = {
        "docs": any(row.official_doc_sources == 0 for row in rows),
        "hosts": any(not row.official_doc_sources_allowed_hosts for row in rows),
        "input_impl": any(not row.input_validation_impl for row in rows),
        "output_impl": any(not row.output_fixture_impl for row in rows),
        "input_tests": any(not row.input_validation_tests for row in rows),
        "output_tests": any(not row.output_shape_tests for row in rows),
        "any": any(not row.strict_all for row in rows),
    }
    return any(checks.get(token, False) for token in fail_on)


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
    parser.add_argument("--verbose", action="store_true", help="Include per-service failure details in text output.")
    parser.add_argument(
        "--fail-on",
        default="none",
        help="Comma-separated fail gates: docs,hosts,input_impl,output_impl,input_tests,output_tests,any",
    )
    parser.add_argument("--report-json", default="", help="Optional path to write JSON output payload.")
    parser.add_argument("--list-services", action="store_true", help="List discovered services and exit.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.json:
        args.format = "json"

    selector = normalize_selector(args.service)
    required = canonical_service(args.require_service) if args.require_service else ""

    discovered = discover_services()
    if args.list_services:
        payload = {
            "provider": "aws",
            "mode": "doc_contract",
            "services": discovered,
            "count": len(discovered),
        }
        if args.format == "json":
            print(json.dumps(payload, indent=2, sort_keys=True))
            write_report_json(args.report_json, payload)
        else:
            for service in discovered:
                print(service)
            print(f"total: {len(discovered)}")
        return 0

    selected = [service for service in discovered if fnmatch.fnmatch(service, selector)]
    if required and required not in discovered:
        print(f"aws-doc-contract-coverage: required service {required!r} not discovered", file=sys.stderr)
        return 2
    if required and required not in selected:
        print(f"aws-doc-contract-coverage: required service {required!r} not matched by selector", file=sys.stderr)
        return 2

    rows = [analyze_service(service) for service in selected]
    summary = summarize(rows)
    payload = {
        "provider": "aws",
        "mode": "doc_contract",
        "service_selector": selector,
        "required_service": required,
        "summary": summary,
        "services": [asdict(row) | {"strict_all": row.strict_all} for row in rows],
    }
    write_report_json(args.report_json, payload)

    if args.format == "json":
        print(json.dumps(payload, indent=2, sort_keys=True))
    else:
        print(render_table(rows))
        print()
        print(json.dumps(summary, indent=2, sort_keys=True))
        if args.verbose:
            failing = [row for row in rows if not row.strict_all]
            if failing:
                print("")
                print("Failing services")
                for row in failing:
                    print(
                        f"- {row.service}: docs={row.official_doc_sources > 0} hosts={row.official_doc_sources_allowed_hosts} "
                        f"input_impl={row.input_validation_impl} output_impl={row.output_fixture_impl} "
                        f"input_tests={row.input_validation_tests} output_tests={row.output_shape_tests}"
                    )

    if should_fail(rows, parse_fail_on(args.fail_on)):
        failing = [row.service for row in rows if not row.strict_all]
        if failing:
            print("aws-doc-contract-coverage: failing services: " + ", ".join(failing), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
