#!/usr/bin/env python3
"""
Azure provider documentation-backed contract coverage analyzer for Stackyard.

This analyzer merges two concerns:
1) every discovered Azure service maps to official documentation sources,
2) implementation and tests expose baseline request/output contract signals.

It is intentionally conservative and reports gaps rather than assuming parity.
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


# Service aliases intentionally mirror the existing Azure coverage tools.
SERVICE_ALIASES: dict[str, str] = {
    "blob": "blob",
    "blobstorage": "blob",
    "blob_storage": "blob",
    "storage": "blob",
    "objectstorage": "blob",
    "object_storage": "blob",
    "queue": "queue",
    "queue_storage": "queue",
    "queuestorage": "queue",
    "keyvault": "keyvault",
    "key_vault": "keyvault",
    "vault": "keyvault",
    "secrets": "keyvault",
    "botframework": "botframework",
    "bot_framework": "botframework",
    "bot_service": "botframework",
    "azure_bot_service_4_0": "botframework",
    "azurebotservice4_0": "botframework",
    "azure_botframework": "botframework",
    "bot_framework_bot_connector_v3_1": "botframework",
    "botframeworkbotconnectorv3_1": "botframework",
    "bot_framework_direct_line_v3_0": "botframework",
    "botframeworkdirectlinev3_0": "botframework",
    "bot_framework_direct_line_v1_1": "botframework",
    "botframeworkdirectlinev1_1": "botframework",
    "direct_line_v3_0": "botframework",
    "direct_line_v1_1": "botframework",
    "directline": "botframework",
    "direct_line": "botframework",
    "contentmoderator": "contentmoderator_image_moderation",
    "content_moderator": "contentmoderator_image_moderation",
    "data_plane": "contentmoderator_image_moderation",
    "data_plane_image_moderation": "contentmoderator_image_moderation",
    "contentmoderatorimagemoderation": "contentmoderator_image_moderation",
    "contentmoderator_image_moderation": "contentmoderator_image_moderation",
    "contentmoderatortextmoderation": "contentmoderator_text_moderation",
    "data_plane_text_moderation": "contentmoderator_text_moderation",
    "contentmoderator_text_moderation": "contentmoderator_text_moderation",
    "contentmoderatorlistmanagement": "contentmoderator_list_management",
    "data_plane_list_management": "contentmoderator_list_management",
    "contentmoderator_list_management": "contentmoderator_list_management",
    "ai_services_document_classifiers": "ai_services_document_classifiers",
    "aiservicesdocumentclassifiers": "ai_services_document_classifiers",
    "ai_services_document_models": "ai_services_document_models",
    "aiservicesdocumentmodels": "ai_services_document_models",
    "ai_services_miscellaneous_operations": "ai_services_miscellaneous_operations",
    "aiservicesmiscellaneousoperations": "ai_services_miscellaneous_operations",
    "ai_services_language_analyze_text": "ai_services_language_analyze_text",
    "aiserviceslanguageanalyzetext": "ai_services_language_analyze_text",
    "ai_services_language_analyze_conversations": "ai_services_language_analyze_conversations",
    "aiserviceslanguageanalyzeconversations": "ai_services_language_analyze_conversations",
    "ai_services_language_analyze_text_authoring": "ai_services_language_analyze_text_authoring",
    "aiserviceslanguageanalyzetextauthoring": "ai_services_language_analyze_text_authoring",
    "ai_services_language_question_answering": "ai_services_language_question_answering",
    "aiserviceslanguagequestionanswering": "ai_services_language_question_answering",
    "ai_services_language_question_answering_authoring": "ai_services_language_question_answering_authoring",
    "aiserviceslanguagequestionansweringauthoring": "ai_services_language_question_answering_authoring",
    "search_service_data_plane_documents": "search_service_data_plane_documents",
    "search_service_data_plane_data_sources": "search_service_data_plane_data_sources",
    "search_service_data_plane_get_service_statistics": "search_service_data_plane_get_service_statistics",
    "search_service_data_plane_indexers": "search_service_data_plane_indexers",
    "search_service_data_plane_indexes": "search_service_data_plane_indexes",
    "search_service_data_plane_skillsets": "search_service_data_plane_skillsets",
    "search_service_data_plane_synonym_maps": "search_service_data_plane_synonym_maps",
    "search_management_resource_manager_admin_keys": "search_management_resource_manager_admin_keys",
    "search_management_resource_manager_network_security_perimeter_configurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "search_management_resource_manager_operations": "search_management_resource_manager_operations",
    "search_management_resource_manager_private_endpoint_connections": "search_management_resource_manager_private_endpoint_connections",
    "search_management_resource_manager_private_link_resources": "search_management_resource_manager_private_link_resources",
    "search_management_resource_manager_query_keys": "search_management_resource_manager_query_keys",
    "search_management_resource_manager_services": "search_management_resource_manager_services",
    "search_management_resource_manager_shared_private_link_resources": "search_management_resource_manager_shared_private_link_resources",
    "search_management_resource_manager_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
}


# Service -> plan docs that should contain primary/method references.
SERVICE_PLAN_DOCS: dict[str, list[str]] = {
    "blob": ["azure-provider-foundation-plan.md"],
    "queue": ["azure-provider-foundation-plan.md"],
    "keyvault": ["azure-provider-foundation-plan.md"],
    "botframework": [
        "azure-bot-service-4.0-plan.md",
        "bot-framework-bot-connector-v3.1-plan.md",
        "bot-framework-direct-line-v3.0-plan.md",
        "bot-framework-direct-line-v1.1-plan.md",
    ],
    "contentmoderator_image_moderation": ["ai-services-data-plane-image-moderation-v1.0-plan.md"],
    "contentmoderator_text_moderation": ["ai-services-data-plane-text-moderation-v1.0-plan.md"],
    "contentmoderator_list_management": ["ai-services-data-plane-list-management-v1.0-plan.md"],
    "ai_services_document_classifiers": ["ai-services-data-plane-document-classifiers-v4.0-plan.md"],
    "ai_services_document_models": ["ai-services-data-plane-document-models-v4.0-plan.md"],
    "ai_services_miscellaneous_operations": ["ai-services-data-plane-miscellaneous-operations-v4.0-plan.md"],
    "ai_services_language_analyze_text": ["ai-services-language-analyze-text-2024-11-01-plan.md"],
    "ai_services_language_analyze_conversations": ["ai-services-language-analyze-conversations-2024-11-01-plan.md"],
    "ai_services_language_analyze_text_authoring": ["ai-services-language-analyze-text-authoring-2023-04-01-plan.md"],
    "ai_services_language_question_answering": ["ai-services-language-question-answering-2023-04-01-plan.md"],
    "ai_services_language_question_answering_authoring": ["ai-services-language-question-answering-authoring-2023-04-01-plan.md"],
    "search_service_data_plane_documents": ["search-service-data-plane-documents-2025-09-01-plan.md"],
    "search_service_data_plane_data_sources": ["search-service-data-plane-data-sources-2025-09-01-plan.md"],
    "search_service_data_plane_get_service_statistics": ["search-service-data-plane-get-service-statistics-2025-09-01-plan.md"],
    "search_service_data_plane_indexers": ["search-service-data-plane-indexers-2025-09-01-plan.md"],
    "search_service_data_plane_indexes": ["search-service-data-plane-indexes-2025-09-01-plan.md"],
    "search_service_data_plane_skillsets": ["search-service-data-plane-skillsets-2025-09-01-plan.md"],
    "search_service_data_plane_synonym_maps": ["search-service-data-plane-synonym-maps-2025-09-01-plan.md"],
    "search_management_resource_manager_admin_keys": ["search-management-resource-manager-admin-keys-2025-05-01-plan.md"],
    "search_management_resource_manager_network_security_perimeter_configurations": [
        "search-management-resource-manager-network-security-parameter-configurations-2025-05-01-plan.md"
    ],
    "search_management_resource_manager_operations": ["search-management-resource-manager-operations-2025-05-01-plan.md"],
    "search_management_resource_manager_private_endpoint_connections": [
        "search-management-resource-manager-private-endpoint-connections-2025-05-01-plan.md"
    ],
    "search_management_resource_manager_private_link_resources": [
        "search-management-resource-manager-private-link-resources-2025-05-01-plan.md"
    ],
    "search_management_resource_manager_query_keys": ["search-management-resource-manager-query-keys-2025-05-01-plan.md"],
    "search_management_resource_manager_services": ["search-management-resource-manager-services-2025-05-01-plan.md"],
    "search_management_resource_manager_shared_private_link_resources": [
        "search-management-resource-manager-shared-private-link-resources-2025-05-01-plan.md"
    ],
    "search_management_resource_manager_usage_by_subscription_sku": [
        "search-management-resource-manager-shared-usage-by-subscription-sku-2025-05-01-plan.md"
    ],
}


# Explicit references for foundational services whose plan doc is provider-level.
SERVICE_FALLBACK_SOURCES: dict[str, list[str]] = {
    "blob": [
        "https://learn.microsoft.com/en-us/rest/api/storageservices/blob-service-rest-api",
    ],
    "queue": [
        "https://learn.microsoft.com/en-us/rest/api/storageservices/queue-service-rest-api",
    ],
    "keyvault": [
        "https://learn.microsoft.com/en-us/rest/api/keyvault/secrets",
    ],
}

ALLOWED_DOC_HOST_MARKERS = (
    "learn.microsoft.com",
    "docs.azure.cn",
    "github.com/microsoft/",
    "raw.githubusercontent.com/microsoft/",
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
    if not value:
        return ""
    return SERVICE_ALIASES.get(value, value)


def read_text(path: Path) -> str:
    try:
        return path.read_text(encoding="utf-8")
    except OSError:
        return ""


def has_any(text: str, patterns: list[str]) -> bool:
    return any(re.search(pattern, text, flags=re.IGNORECASE) for pattern in patterns)


def discover_service_files() -> dict[str, list[Path]]:
    out: dict[str, list[Path]] = {}
    for path in SERVER_DIR.glob("provider_azure_*.go"):
        if path.name.endswith("_test.go"):
            continue
        service = path.stem.removeprefix("provider_azure_")
        if service in {"auth", "router"}:
            continue
        out.setdefault(service, []).append(path)
    return out


def discover_test_files(service: str) -> list[Path]:
    files = sorted(SERVER_DIR.glob(f"provider_azure_{service}*_test.go"))
    if service == "blob":
        legacy = SERVER_DIR / "provider_objectstorage_test.go"
        if legacy.exists():
            files.append(legacy)
    return files


def extract_urls_from_docs(plan_docs: list[str]) -> list[str]:
    urls: list[str] = []
    seen: set[str] = set()
    for name in plan_docs:
        path = DOCS_DIR / name
        text = read_text(path)
        for found in URL_RE.findall(text):
            cleaned = found.rstrip(".,")
            if cleaned not in seen:
                seen.add(cleaned)
                urls.append(cleaned)
    return urls


def analyze_service(service: str, source_files: list[Path]) -> DocContractCoverage:
    source_text = "\n".join(read_text(path) for path in source_files)
    test_text = "\n".join(read_text(path) for path in discover_test_files(service))

    plan_docs = SERVICE_PLAN_DOCS.get(service, [])
    docs_urls = extract_urls_from_docs(plan_docs)
    docs_urls.extend(url for url in SERVICE_FALLBACK_SOURCES.get(service, []) if url not in docs_urls)

    official_doc_sources_allowed_hosts = bool(docs_urls) and all(
        any(marker in url.lower() for marker in ALLOWED_DOC_HOST_MARKERS)
        for url in docs_urls
    )

    input_validation_impl = has_any(
        source_text,
        [
            r"StatusBadRequest",
            r"InvalidRequest",
            r"\bmissing\b",
            r"\binvalid\b",
        ],
    )
    output_fixture_impl = has_any(
        source_text,
        [
            r"respondJSON\(",
            r"WriteHeader\(http\.Status",
            r"xml\.Marshal",
            r"Content-Type",
            r"respondAzureImplemented\(",
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
            r"StatusOK",
        ],
    )
    output_shape_tests = has_any(
        test_text,
        [
            r"json\.Unmarshal",
            r"xml\.Unmarshal",
            r"strings\.Contains",
            r"providerContractJSONMap\(",
            r"\[\"error\"\]",
            r"\[\"message\"\]",
            r"\[\"status\"\]",
            r"\[\"provider\"\]",
        ],
    )

    return DocContractCoverage(
        service=service,
        plan_docs_mapped=service in SERVICE_PLAN_DOCS,
        plan_docs_exist=bool(plan_docs) and all((DOCS_DIR / name).exists() for name in plan_docs),
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
        "service".ljust(52)
        + " docs  hosts  in_impl  out_impl  in_tests  out_tests  strict"
    )
    lines = [header, "-" * len(header)]
    for row in rows:
        lines.append(
            row.service.ljust(52)
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
    parser.add_argument(
        "--service",
        default="*",
        help="Service filter (name or glob). Example: blob, keyvault, '*'.",
    )
    parser.add_argument(
        "--require-service",
        default="",
        help="Fail if the specified service is not discovered after filtering.",
    )
    parser.add_argument(
        "--format",
        choices=["text", "json"],
        default="text",
        help="Output format.",
    )
    parser.add_argument(
        "--fail-on",
        default="none",
        help="Comma-separated fail gates: docs,hosts,input_impl,output_impl,input_tests,output_tests,any",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Alias for --format json.",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Include per-service failure details in text output.",
    )
    parser.add_argument(
        "--report-json",
        default="",
        help="Optional path to write JSON output payload.",
    )
    parser.add_argument(
        "--list-services",
        action="store_true",
        help="List discovered services and exit.",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.json:
        args.format = "json"

    service_filter = canonical_service(args.service)
    if not service_filter or service_filter == "_":
        service_filter = "*"

    files_by_service = discover_service_files()
    discovered = sorted(files_by_service)
    required = canonical_service(args.require_service) if args.require_service else ""

    if args.list_services:
        if args.format == "json":
            payload = {
                "provider": "azure",
                "mode": "doc_contract",
                "services": discovered,
                "count": len(discovered),
            }
            print(json.dumps(payload, indent=2, sort_keys=True))
            write_report_json(args.report_json, payload)
        else:
            for service in discovered:
                print(service)
            print(f"total: {len(discovered)}")
        return 0

    selected = sorted(
        service
        for service in files_by_service
        if fnmatch.fnmatch(service, service_filter)
    )
    if required and required not in files_by_service:
        print(f"azure-doc-contract-coverage: required service {required!r} not discovered", file=sys.stderr)
        return 2
    if required and required not in selected:
        print(f"azure-doc-contract-coverage: required service {required!r} not matched by selector", file=sys.stderr)
        return 2

    rows = [analyze_service(service, files_by_service[service]) for service in selected]
    summary = summarize(rows)

    payload = {
        "provider": "azure",
        "mode": "doc_contract",
        "service_selector": service_filter,
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
                print()
                print("Failing services")
                for row in failing:
                    print(
                        f"- {row.service}: docs={row.official_doc_sources > 0} hosts={row.official_doc_sources_allowed_hosts} "
                        f"input_impl={row.input_validation_impl} output_impl={row.output_fixture_impl} "
                        f"input_tests={row.input_validation_tests} output_tests={row.output_shape_tests}"
                    )

    fail_on = parse_fail_on(args.fail_on)
    if should_fail(rows, fail_on):
        failing = [row.service for row in rows if not row.strict_all]
        if failing:
            print(
                "azure-doc-contract-coverage: failing services: " + ", ".join(failing),
                file=sys.stderr,
            )
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
