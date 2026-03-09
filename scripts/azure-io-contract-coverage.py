#!/usr/bin/env python3
"""
Azure provider input/output contract coverage analyzer for Stackyard.

Checks four quality gates per Azure service implementation:
1) input validation implementation signals,
2) output fixture implementation signals,
3) input validation test signals,
4) output shape assertion test signals.
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
    "bot_framework_bot_connector": "botframework",
    "bot_connector_v3_1": "botframework",
    "azure_ai_bot_service_bot_connector_v3_1": "botframework",
    "ai_bot_service_bot_connector_v3_1": "botframework",
    "bot_framework_direct_line_v3_0": "botframework",
    "botframeworkdirectlinev3_0": "botframework",
    "bot_framework_direct_line": "botframework",
    "direct_line_v3_0": "botframework",
    "bot_framework_direct_line_v1_1": "botframework",
    "botframeworkdirectlinev1_1": "botframework",
    "direct_line_v1_1": "botframework",
    "directline": "botframework",
    "direct_line": "botframework",
    "azure_ai_bot_service_direct_line_v3_0": "botframework",
    "ai_bot_service_direct_line_v3_0": "botframework",
    "azure_ai_bot_service_direct_line_v1_1": "botframework",
    "ai_bot_service_direct_line_v1_1": "botframework",
    "contentmoderator": "contentmoderator_image_moderation",
    "content_moderator": "contentmoderator_image_moderation",
    "data_plane": "contentmoderator_image_moderation",
    "data_plane_image_moderation": "contentmoderator_image_moderation",
    "data_plane_image_moderation_v1_0": "contentmoderator_image_moderation",
    "ai_services_data_plane_image_moderation": "contentmoderator_image_moderation",
    "ai_services_data_plane_image_moderation_v1_0": "contentmoderator_image_moderation",
    "azure_ai_services_data_plane_image_moderation": "contentmoderator_image_moderation",
    "azure_ai_services_data_plane_image_moderation_v1_0": "contentmoderator_image_moderation",
    "contentmoderatorimagemoderation": "contentmoderator_image_moderation",
    "contentmoderator_image_moderation": "contentmoderator_image_moderation",
    "azure_contentmoderator_image_moderation_v1_0": "contentmoderator_image_moderation",
    "azure_contentmoderator_image_moderation": "contentmoderator_image_moderation",
    "azure_content_moderator_image_moderation_v1_0": "contentmoderator_image_moderation",
    "contentmoderatortextmoderation": "contentmoderator_text_moderation",
    "data_plane_text_moderation": "contentmoderator_text_moderation",
    "data_plane_text_moderation_v1_0": "contentmoderator_text_moderation",
    "ai_services_data_plane_text_moderation": "contentmoderator_text_moderation",
    "ai_services_data_plane_text_moderation_v1_0": "contentmoderator_text_moderation",
    "azure_ai_services_data_plane_text_moderation": "contentmoderator_text_moderation",
    "azure_ai_services_data_plane_text_moderation_v1_0": "contentmoderator_text_moderation",
    "contentmoderator_text_moderation": "contentmoderator_text_moderation",
    "azure_contentmoderator_text_moderation_v1_0": "contentmoderator_text_moderation",
    "azure_contentmoderator_text_moderation": "contentmoderator_text_moderation",
    "azure_content_moderator_text_moderation_v1_0": "contentmoderator_text_moderation",
    "contentmoderatorlistmanagement": "contentmoderator_list_management",
    "data_plane_list_management": "contentmoderator_list_management",
    "data_plane_list_management_v1_0": "contentmoderator_list_management",
    "ai_services_data_plane_list_management": "contentmoderator_list_management",
    "ai_services_data_plane_list_management_v1_0": "contentmoderator_list_management",
    "azure_ai_services_data_plane_list_management": "contentmoderator_list_management",
    "azure_ai_services_data_plane_list_management_v1_0": "contentmoderator_list_management",
    "contentmoderator_list_management": "contentmoderator_list_management",
    "azure_contentmoderator_list_management_v1_0": "contentmoderator_list_management",
    "azure_contentmoderator_list_management": "contentmoderator_list_management",
    "azure_content_moderator_list_management_v1_0": "contentmoderator_list_management",
    "ai_services_document_classifiers": "ai_services_document_classifiers",
    "aiservicesdocumentclassifiers": "ai_services_document_classifiers",
    "ai_services_data_plane_document_classifiers": "ai_services_document_classifiers",
    "ai_services_data_plane_document_classifiers_v4_0": "ai_services_document_classifiers",
    "azure_ai_services_document_classifiers": "ai_services_document_classifiers",
    "azure_ai_services_data_plane_document_classifiers": "ai_services_document_classifiers",
    "azure_ai_services_data_plane_document_classifiers_v4_0": "ai_services_document_classifiers",
    "ai_services_miscellaneous_operations": "ai_services_miscellaneous_operations",
    "aiservicesmiscellaneousoperations": "ai_services_miscellaneous_operations",
    "ai_services_data_plane_miscellaneous_operations": "ai_services_miscellaneous_operations",
    "ai_services_data_plane_miscellaneous_operations_v4_0": "ai_services_miscellaneous_operations",
    "azure_ai_services_miscellaneous_operations": "ai_services_miscellaneous_operations",
    "azure_ai_services_data_plane_miscellaneous_operations": "ai_services_miscellaneous_operations",
    "azure_ai_services_data_plane_miscellaneous_operations_v4_0": "ai_services_miscellaneous_operations",
    "ai_services_document_models": "ai_services_document_models",
    "aiservicesdocumentmodels": "ai_services_document_models",
    "ai_services_data_plane_document_models": "ai_services_document_models",
    "ai_services_data_plane_document_models_v4_0": "ai_services_document_models",
    "azure_ai_services_document_models": "ai_services_document_models",
    "azure_ai_services_data_plane_document_models": "ai_services_document_models",
    "azure_ai_services_data_plane_document_models_v4_0": "ai_services_document_models",
    "ai_services_language_analyze_text": "ai_services_language_analyze_text",
    "aiserviceslanguageanalyzetext": "ai_services_language_analyze_text",
    "ai_services_language_analyze_text_2024_11_01": "ai_services_language_analyze_text",
    "azure_ai_services_language_analyze_text": "ai_services_language_analyze_text",
    "azure_ai_services_language_analyze_text_2024_11_01": "ai_services_language_analyze_text",
    "ai_services_language_analyze_text_authoring": "ai_services_language_analyze_text_authoring",
    "aiserviceslanguageanalyzetextauthoring": "ai_services_language_analyze_text_authoring",
    "ai_services_language_analyze_text_authoring_2023_04_01": "ai_services_language_analyze_text_authoring",
    "azure_ai_services_language_analyze_text_authoring": "ai_services_language_analyze_text_authoring",
    "azure_ai_services_language_analyze_text_authoring_2023_04_01": "ai_services_language_analyze_text_authoring",
    "ai_services_language_question_answering_authoring": "ai_services_language_question_answering_authoring",
    "aiserviceslanguagequestionansweringauthoring": "ai_services_language_question_answering_authoring",
    "ai_services_language_question_answering_authoring_2023_04_01": "ai_services_language_question_answering_authoring",
    "azure_ai_services_language_question_answering_authoring": "ai_services_language_question_answering_authoring",
    "azure_ai_services_language_question_answering_authoring_2023_04_01": "ai_services_language_question_answering_authoring",
    "ai_services_language_question_answering": "ai_services_language_question_answering",
    "aiserviceslanguagequestionanswering": "ai_services_language_question_answering",
    "ai_services_language_question_answering_2023_04_01": "ai_services_language_question_answering",
    "azure_ai_services_language_question_answering": "ai_services_language_question_answering",
    "azure_ai_services_language_question_answering_2023_04_01": "ai_services_language_question_answering",
    "search_service_data_plane_documents": "search_service_data_plane_documents",
    "search_service_data_plane_documents_2025_09_01": "search_service_data_plane_documents",
    "search_service_documents": "search_service_data_plane_documents",
    "searchservicedataplanedocuments": "search_service_data_plane_documents",
    "azure_search_service_data_plane_documents": "search_service_data_plane_documents",
    "azure_search_service_data_plane_documents_2025_09_01": "search_service_data_plane_documents",
    "search_service_data_plane_data_sources": "search_service_data_plane_data_sources",
    "search_service_data_plane_data_sources_2025_09_01": "search_service_data_plane_data_sources",
    "search_service_data_sources": "search_service_data_plane_data_sources",
    "searchservicedataplanedatasources": "search_service_data_plane_data_sources",
    "azure_search_service_data_plane_data_sources": "search_service_data_plane_data_sources",
    "azure_search_service_data_plane_data_sources_2025_09_01": "search_service_data_plane_data_sources",
    "search_service_data_plane_get_service_statistics": "search_service_data_plane_get_service_statistics",
    "search_service_data_plane_get_service_statistics_2025_09_01": "search_service_data_plane_get_service_statistics",
    "search_service_get_service_statistics": "search_service_data_plane_get_service_statistics",
    "searchservicedataplanegetservicestatistics": "search_service_data_plane_get_service_statistics",
    "azure_search_service_data_plane_get_service_statistics": "search_service_data_plane_get_service_statistics",
    "azure_search_service_data_plane_get_service_statistics_2025_09_01": "search_service_data_plane_get_service_statistics",
    "search_service_data_plane_indexers": "search_service_data_plane_indexers",
    "search_service_data_plane_indexers_2025_09_01": "search_service_data_plane_indexers",
    "search_service_indexers": "search_service_data_plane_indexers",
    "searchservicedataplaneindexers": "search_service_data_plane_indexers",
    "azure_search_service_data_plane_indexers": "search_service_data_plane_indexers",
    "azure_search_service_data_plane_indexers_2025_09_01": "search_service_data_plane_indexers",
    "search_service_data_plane_indexes": "search_service_data_plane_indexes",
    "search_service_data_plane_indexes_2025_09_01": "search_service_data_plane_indexes",
    "search_service_indexes": "search_service_data_plane_indexes",
    "searchservicedataplaneindexes": "search_service_data_plane_indexes",
    "azure_search_service_data_plane_indexes": "search_service_data_plane_indexes",
    "azure_search_service_data_plane_indexes_2025_09_01": "search_service_data_plane_indexes",
    "search_service_data_plane_skillsets": "search_service_data_plane_skillsets",
    "search_service_data_plane_skillsets_2025_09_01": "search_service_data_plane_skillsets",
    "search_service_skillsets": "search_service_data_plane_skillsets",
    "searchservicedataplaneskillsets": "search_service_data_plane_skillsets",
    "azure_search_service_data_plane_skillsets": "search_service_data_plane_skillsets",
    "azure_search_service_data_plane_skillsets_2025_09_01": "search_service_data_plane_skillsets",
    "search_service_data_plane_synonym_maps": "search_service_data_plane_synonym_maps",
    "search_service_data_plane_synonym_maps_2025_09_01": "search_service_data_plane_synonym_maps",
    "search_service_synonym_maps": "search_service_data_plane_synonym_maps",
    "searchservicedataplanesynonymmaps": "search_service_data_plane_synonym_maps",
    "azure_search_service_data_plane_synonym_maps": "search_service_data_plane_synonym_maps",
    "azure_search_service_data_plane_synonym_maps_2025_09_01": "search_service_data_plane_synonym_maps",
    "search_management_resource_manager_admin_keys": "search_management_resource_manager_admin_keys",
    "search_management_resource_manager_admin_keys_2025_05_01": "search_management_resource_manager_admin_keys",
    "search_management_admin_keys": "search_management_resource_manager_admin_keys",
    "searchmanagementresourcemanageradminkeys": "search_management_resource_manager_admin_keys",
    "azure_search_management_resource_manager_admin_keys": "search_management_resource_manager_admin_keys",
    "azure_search_management_resource_manager_admin_keys_2025_05_01": "search_management_resource_manager_admin_keys",
    "search_management_resource_manager_network_security_perimeter_configurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "search_management_resource_manager_network_security_perimeter_configurations_2025_05_01": "search_management_resource_manager_network_security_perimeter_configurations",
    "search_management_resource_manager_network_security_parameter_configurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "search_management_resource_manager_network_security_parameter_configurations_2025_05_01": "search_management_resource_manager_network_security_perimeter_configurations",
    "searchmanagementresourcemanagernetworksecurityperimeterconfigurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "searchmanagementresourcemanagernetworksecurityparameterconfigurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "azure_search_management_resource_manager_network_security_perimeter_configurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "azure_search_management_resource_manager_network_security_perimeter_configurations_2025_05_01": "search_management_resource_manager_network_security_perimeter_configurations",
    "azure_search_management_resource_manager_network_security_parameter_configurations": "search_management_resource_manager_network_security_perimeter_configurations",
    "azure_search_management_resource_manager_network_security_parameter_configurations_2025_05_01": "search_management_resource_manager_network_security_perimeter_configurations",
    "search_management_resource_manager_operations": "search_management_resource_manager_operations",
    "search_management_resource_manager_operations_2025_05_01": "search_management_resource_manager_operations",
    "search_management_operations": "search_management_resource_manager_operations",
    "searchmanagementresourcemanageroperations": "search_management_resource_manager_operations",
    "azure_search_management_resource_manager_operations": "search_management_resource_manager_operations",
    "azure_search_management_resource_manager_operations_2025_05_01": "search_management_resource_manager_operations",
    "search_management_resource_manager_private_endpoint_connections": "search_management_resource_manager_private_endpoint_connections",
    "search_management_resource_manager_private_endpoint_connections_2025_05_01": "search_management_resource_manager_private_endpoint_connections",
    "search_management_private_endpoint_connections": "search_management_resource_manager_private_endpoint_connections",
    "searchmanagementresourcemanagerprivateendpointconnections": "search_management_resource_manager_private_endpoint_connections",
    "azure_search_management_resource_manager_private_endpoint_connections": "search_management_resource_manager_private_endpoint_connections",
    "azure_search_management_resource_manager_private_endpoint_connections_2025_05_01": "search_management_resource_manager_private_endpoint_connections",
    "search_management_resource_manager_private_link_resources": "search_management_resource_manager_private_link_resources",
    "search_management_resource_manager_private_link_resources_2025_05_01": "search_management_resource_manager_private_link_resources",
    "search_management_private_link_resources": "search_management_resource_manager_private_link_resources",
    "searchmanagementresourcemanagerprivatelinkresources": "search_management_resource_manager_private_link_resources",
    "azure_search_management_resource_manager_private_link_resources": "search_management_resource_manager_private_link_resources",
    "azure_search_management_resource_manager_private_link_resources_2025_05_01": "search_management_resource_manager_private_link_resources",
    "search_management_resource_manager_query_keys": "search_management_resource_manager_query_keys",
    "search_management_resource_manager_query_keys_2025_05_01": "search_management_resource_manager_query_keys",
    "search_management_query_keys": "search_management_resource_manager_query_keys",
    "searchmanagementresourcemanagerquerykeys": "search_management_resource_manager_query_keys",
    "azure_search_management_resource_manager_query_keys": "search_management_resource_manager_query_keys",
    "azure_search_management_resource_manager_query_keys_2025_05_01": "search_management_resource_manager_query_keys",
    "search_management_resource_manager_services": "search_management_resource_manager_services",
    "search_management_resource_manager_services_2025_05_01": "search_management_resource_manager_services",
    "search_management_services": "search_management_resource_manager_services",
    "searchmanagementresourcemanagerservices": "search_management_resource_manager_services",
    "azure_search_management_resource_manager_services": "search_management_resource_manager_services",
    "azure_search_management_resource_manager_services_2025_05_01": "search_management_resource_manager_services",
    "search_management_resource_manager_shared_private_link_resources": "search_management_resource_manager_shared_private_link_resources",
    "search_management_resource_manager_shared_private_link_resources_2025_05_01": "search_management_resource_manager_shared_private_link_resources",
    "search_management_shared_private_link_resources": "search_management_resource_manager_shared_private_link_resources",
    "searchmanagementresourcemanagersharedprivatelinkresources": "search_management_resource_manager_shared_private_link_resources",
    "azure_search_management_resource_manager_shared_private_link_resources": "search_management_resource_manager_shared_private_link_resources",
    "azure_search_management_resource_manager_shared_private_link_resources_2025_05_01": "search_management_resource_manager_shared_private_link_resources",
    "search_management_resource_manager_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
    "search_management_resource_manager_usage_by_subscription_sku_2025_05_01": "search_management_resource_manager_usage_by_subscription_sku",
    "search_management_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
    "searchmanagementresourcemanagerusagebysubscriptionsku": "search_management_resource_manager_usage_by_subscription_sku",
    "azure_search_management_resource_manager_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
    "azure_search_management_resource_manager_usage_by_subscription_sku_2025_05_01": "search_management_resource_manager_usage_by_subscription_sku",
    "search_management_resource_manager_shared_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
    "search_management_resource_manager_shared_usage_by_subscription_sku_2025_05_01": "search_management_resource_manager_usage_by_subscription_sku",
    "search_management_shared_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
    "searchmanagementresourcemanagersharedusagebysubscriptionsku": "search_management_resource_manager_usage_by_subscription_sku",
    "azure_search_management_resource_manager_shared_usage_by_subscription_sku": "search_management_resource_manager_usage_by_subscription_sku",
    "azure_search_management_resource_manager_shared_usage_by_subscription_sku_2025_05_01": "search_management_resource_manager_usage_by_subscription_sku",
    "ai_services_language_analyze_conversations": "ai_services_language_analyze_conversations",
    "aiserviceslanguageanalyzeconversations": "ai_services_language_analyze_conversations",
    "ai_services_language_analyze_conversations_2024_11_01": "ai_services_language_analyze_conversations",
    "azure_ai_services_language_analyze_conversations": "ai_services_language_analyze_conversations",
    "azure_ai_services_language_analyze_conversations_2024_11_01": "ai_services_language_analyze_conversations",
}


@dataclass
class IOCoverage:
    service: str
    input_validation_impl: bool
    output_fixture_impl: bool
    input_validation_tests: bool
    output_shape_tests: bool

    @property
    def strict_all_four(self) -> bool:
        return (
            self.input_validation_impl
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


def analyze_service(service: str, source_files: list[Path]) -> IOCoverage:
    source_text = "\n".join(read_text(path) for path in source_files)
    test_text = "\n".join(read_text(path) for path in discover_test_files(service))

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
            r"respondAzureImplemented\(",
            r"WriteHeader\(http\.Status",
            r"xml\.Marshal",
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
            r"xml\.Unmarshal",
            r"strings\.Contains",
            r"providerContractJSONMap\(",
            r"\[\"error\"\]",
            r"\[\"message\"\]",
        ],
    )

    return IOCoverage(
        service=service,
        input_validation_impl=input_validation_impl,
        output_fixture_impl=output_fixture_impl,
        input_validation_tests=input_validation_tests,
        output_shape_tests=output_shape_tests,
    )


def summarize(rows: list[IOCoverage]) -> dict[str, int]:
    return {
        "total": len(rows),
        "input_validation_impl": sum(1 for row in rows if row.input_validation_impl),
        "output_fixture_impl": sum(1 for row in rows if row.output_fixture_impl),
        "input_validation_tests": sum(1 for row in rows if row.input_validation_tests),
        "output_shape_tests": sum(1 for row in rows if row.output_shape_tests),
        "strict_all_four": sum(1 for row in rows if row.strict_all_four),
    }


def render_table(rows: list[IOCoverage]) -> str:
    header = (
        "service".ljust(16)
        + " input_validation_impl  output_fixture_impl  input_validation_tests  output_shape_tests  strict_all_four"
    )
    lines = [header, "-" * len(header)]
    for row in rows:
        lines.append(
            row.service.ljust(16)
            + f" {str(row.input_validation_impl):<21}"
            + f" {str(row.output_fixture_impl):<20}"
            + f" {str(row.input_validation_tests):<22}"
            + f" {str(row.output_shape_tests):<19}"
            + f" {str(row.strict_all_four):<15}"
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


def should_fail(rows: list[IOCoverage], fail_on: set[str]) -> bool:
    if not fail_on:
        return False
    checks = {
        "input_impl": any(not row.input_validation_impl for row in rows),
        "output_impl": any(not row.output_fixture_impl for row in rows),
        "input_tests": any(not row.input_validation_tests for row in rows),
        "output_tests": any(not row.output_shape_tests for row in rows),
        "any": any(not row.strict_all_four for row in rows),
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
        help="Service filter (name or glob). Examples: blob, keyvault, '*'.",
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
        help="Comma-separated fail gates: input_impl,output_impl,input_tests,output_tests,any",
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

    required = canonical_service(args.require_service) if args.require_service else ""

    files_by_service = discover_service_files()
    discovered = sorted(files_by_service)

    if args.list_services:
        if args.format == "json":
            payload = {
                "provider": "azure",
                "mode": "io_contract",
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
        print(
            f"azure-io-contract-coverage: required service {required!r} not discovered",
            file=sys.stderr,
        )
        return 2
    if required and required not in selected:
        print(
            f"azure-io-contract-coverage: required service {required!r} not found in selection",
            file=sys.stderr,
        )
        return 2

    rows = [analyze_service(service, files_by_service[service]) for service in selected]
    summary = summarize(rows)
    payload = {
        "provider": "azure",
        "mode": "io_contract",
        "service_selector": service_filter,
        "required_service": required,
        "summary": summary,
        "services": [asdict(row) | {"strict_all_four": row.strict_all_four} for row in rows],
    }
    write_report_json(args.report_json, payload)

    if args.format == "json":
        print(json.dumps(payload, indent=2, sort_keys=True))
    else:
        print(render_table(rows))
        print()
        print(json.dumps(summary, indent=2, sort_keys=True))
        if args.verbose:
            failing = [row for row in rows if not row.strict_all_four]
            if failing:
                print()
                print("Failing services")
                for row in failing:
                    print(
                        f"- {row.service}: input_impl={row.input_validation_impl} "
                        f"output_impl={row.output_fixture_impl} input_tests={row.input_validation_tests} "
                        f"output_tests={row.output_shape_tests}"
                    )

    fail_on = parse_fail_on(args.fail_on)
    if should_fail(rows, fail_on):
        failing = [row.service for row in rows if not row.strict_all_four]
        if failing:
            print(
                "azure-io-contract-coverage: failing services: " + ", ".join(failing),
                file=sys.stderr,
            )
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
