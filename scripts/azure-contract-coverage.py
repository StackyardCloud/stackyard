#!/usr/bin/env python3
"""
Azure provider contract coverage analyzer for Stackyard.

Checks three quality gates per Azure service implementation:
1) request validation signals in implementation source,
2) typed success fixture signals in implementation source,
3) negative contract test signals in test source.
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
    "contentmoderator": "content_moderator",
    "content_moderator": "content_moderator",
    "content_moderator_v1_0": "content_moderator",
    "contentmoderator_v1_0": "content_moderator",
    "content_moderator_service": "content_moderator",
    "content_moderator_service_v1_0": "content_moderator",
    "azure_content_moderator": "content_moderator",
    "azure_contentmoderator": "content_moderator",
    "azure_content_moderator_v1_0": "content_moderator",
    "azure_contentmoderator_v1_0": "content_moderator",
    "azure_ai_services_content_moderator": "content_moderator",
    "azure_ai_services_content_moderator_v1_0": "content_moderator",
    "ai_services_content_moderator": "content_moderator",
    "ai_services_content_moderator_v1_0": "content_moderator",
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
    "content_safety": "content_safety",
    "contentsafety": "content_safety",
    "content_safety_2024_09_01": "content_safety",
    "contentsafety2024_09_01": "content_safety",
    "azure_content_safety": "content_safety",
    "azure_content_safety_2024_09_01": "content_safety",
    "azure_contentsafety": "content_safety",
    "azure_contentsafety2024_09_01": "content_safety",
    "ai_services_content_safety": "content_safety",
    "ai_services_content_safety_2024_09_01": "content_safety",
    "azure_ai_services_content_safety": "content_safety",
    "azure_ai_services_content_safety_2024_09_01": "content_safety",
    "content_understanding": "content_understanding",
    "contentunderstanding": "content_understanding",
    "content_understanding_2025_11_01": "content_understanding",
    "contentunderstanding2025_11_01": "content_understanding",
    "azure_content_understanding": "content_understanding",
    "azure_content_understanding_2025_11_01": "content_understanding",
    "azure_contentunderstanding": "content_understanding",
    "azure_contentunderstanding2025_11_01": "content_understanding",
    "ai_services_content_understanding": "content_understanding",
    "ai_services_content_understanding_2025_11_01": "content_understanding",
    "azure_ai_services_content_understanding": "content_understanding",
    "azure_ai_services_content_understanding_2025_11_01": "content_understanding",
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
    "speech_to_text": "speech_to_text",
    "speechtotext": "speech_to_text",
    "speech_to_text_v3_2_preview_2": "speech_to_text",
    "azure_speech_to_text": "speech_to_text",
    "azure_speech_to_text_v3_2_preview_2": "speech_to_text",
    "azure_ai_services_speech_to_text": "speech_to_text",
    "azure_ai_services_speech_to_text_v3_2_preview_2": "speech_to_text",
    "text_to_speech": "text_to_speech",
    "texttospeech": "text_to_speech",
    "batch_text_to_speech": "text_to_speech",
    "batchtexttospeech": "text_to_speech",
    "text_to_speech_2024_04_01": "text_to_speech",
    "batch_text_to_speech_2024_04_01": "text_to_speech",
    "batchtexttospeech2024_04_01": "text_to_speech",
    "azure_text_to_speech": "text_to_speech",
    "azure_text_to_speech_2024_04_01": "text_to_speech",
    "azure_batch_text_to_speech": "text_to_speech",
    "azure_batch_text_to_speech_2024_04_01": "text_to_speech",
    "azure_ai_services_text_to_speech": "text_to_speech",
    "azure_ai_services_text_to_speech_2024_04_01": "text_to_speech",
    "azure_ai_services_batch_text_to_speech": "text_to_speech",
    "azure_ai_services_batch_text_to_speech_2024_04_01": "text_to_speech",
    "account_management": "account_management",
    "accountmanagement": "account_management",
    "account_management_2024_10_01": "account_management",
    "azure_account_management": "account_management",
    "azure_account_management_2024_10_01": "account_management",
    "azure_ai_services_account_management": "account_management",
    "azure_ai_services_account_management_2024_10_01": "account_management",
    "ai_services_account_management": "account_management",
    "ai_services_account_management_2024_10_01": "account_management",
    "ai_foundry_account_management": "ai_foundry_account_management",
    "aifoundryaccountmanagement": "ai_foundry_account_management",
    "account_management_2025_06_01": "ai_foundry_account_management",
    "ai_foundry_account_management_2025_06_01": "ai_foundry_account_management",
    "aifoundry_accountmanagement_2025_06_01": "ai_foundry_account_management",
    "aifoundryaccountmanagement2025_06_01": "ai_foundry_account_management",
    "rest_aifoundry_accountmanagement_2025_06_01": "ai_foundry_account_management",
    "azure_ai_foundry_account_management": "ai_foundry_account_management",
    "azure_ai_foundry_account_management_2025_06_01": "ai_foundry_account_management",
    "azure_aifoundry_accountmanagement": "ai_foundry_account_management",
    "azure_aifoundry_accountmanagement_2025_06_01": "ai_foundry_account_management",
    "ai_foundry_model_inference": "ai_foundry_model_inference",
    "aifoundrymodelinference": "ai_foundry_model_inference",
    "model_inference_2024_05_01_preview": "ai_foundry_model_inference",
    "ai_foundry_model_inference_2024_05_01_preview": "ai_foundry_model_inference",
    "aifoundry_model_inference_2024_05_01_preview": "ai_foundry_model_inference",
    "aifoundrymodelinference2024_05_01_preview": "ai_foundry_model_inference",
    "rest_aifoundry_model_inference_2024_05_01_preview": "ai_foundry_model_inference",
    "azure_ai_foundry_model_inference": "ai_foundry_model_inference",
    "azure_ai_foundry_model_inference_2024_05_01_preview": "ai_foundry_model_inference",
    "azure_aifoundry_model_inference": "ai_foundry_model_inference",
    "azure_aifoundry_model_inference_2024_05_01_preview": "ai_foundry_model_inference",
    "analysis_services": "analysis_services",
    "analysisservices": "analysis_services",
    "analysis_services_2017_08_01": "analysis_services",
    "analysisservices2017_08_01": "analysis_services",
    "rest_analysisservices_2017_08_01": "analysis_services",
    "azure_analysis_services": "analysis_services",
    "azure_analysis_services_2017_08_01": "analysis_services",
    "azure_analysisservices": "analysis_services",
    "azure_analysisservices_2017_08_01": "analysis_services",
    "api_center_data_plane": "api_center_data_plane",
    "apicenter_data_plane": "api_center_data_plane",
    "api_center_dataplane": "api_center_data_plane",
    "apicenterdataplane": "api_center_data_plane",
    "api_center_data_plane_2024_02_01_preview": "api_center_data_plane",
    "apicenter_data_plane_2024_02_01_preview": "api_center_data_plane",
    "api_center_dataplane_2024_02_01_preview": "api_center_data_plane",
    "apicenterdataplane2024_02_01_preview": "api_center_data_plane",
    "rest_dataplane_apicenter_2024_02_01_preview": "api_center_data_plane",
    "azure_api_center_data_plane": "api_center_data_plane",
    "azure_api_center_data_plane_2024_02_01_preview": "api_center_data_plane",
    "azure_apicenter_data_plane": "api_center_data_plane",
    "azure_apicenter_data_plane_2024_02_01_preview": "api_center_data_plane",
    "azure_api_center_dataplane": "api_center_data_plane",
    "azure_api_center_dataplane_2024_02_01_preview": "api_center_data_plane",
    "api_center_resource_manager": "api_center_resource_manager",
    "apicenter_resource_manager": "api_center_resource_manager",
    "api_center_resourcemanager": "api_center_resource_manager",
    "apicenterresourcemanager": "api_center_resource_manager",
    "api_center_resource_manager_2024_03_01": "api_center_resource_manager",
    "apicenter_resource_manager_2024_03_01": "api_center_resource_manager",
    "api_center_resourcemanager_2024_03_01": "api_center_resource_manager",
    "apicenterresourcemanager2024_03_01": "api_center_resource_manager",
    "rest_resource_manager_apicenter_2024_03_01": "api_center_resource_manager",
    "azure_api_center_resource_manager": "api_center_resource_manager",
    "azure_api_center_resource_manager_2024_03_01": "api_center_resource_manager",
    "azure_apicenter_resource_manager": "api_center_resource_manager",
    "azure_apicenter_resource_manager_2024_03_01": "api_center_resource_manager",
    "azure_api_center_resourcemanager": "api_center_resource_manager",
    "azure_api_center_resourcemanager_2024_03_01": "api_center_resource_manager",
    "api_management_resource_manager": "api_management_resource_manager",
    "apimanagement_resource_manager": "api_management_resource_manager",
    "api_management_resourcemanager": "api_management_resource_manager",
    "apimanagementresourcemanager": "api_management_resource_manager",
    "api_management_resource_manager_2024_05_01": "api_management_resource_manager",
    "apimanagement_resource_manager_2024_05_01": "api_management_resource_manager",
    "api_management_resourcemanager_2024_05_01": "api_management_resource_manager",
    "apimanagementresourcemanager2024_05_01": "api_management_resource_manager",
    "rest_apimanagement_2024_05_01": "api_management_resource_manager",
    "azure_api_management_resource_manager": "api_management_resource_manager",
    "azure_api_management_resource_manager_2024_05_01": "api_management_resource_manager",
    "azure_apimanagement_resource_manager": "api_management_resource_manager",
    "azure_apimanagement_resource_manager_2024_05_01": "api_management_resource_manager",
    "azure_api_management_resourcemanager": "api_management_resource_manager",
    "azure_api_management_resourcemanager_2024_05_01": "api_management_resource_manager",
    "app_compliance": "app_compliance",
    "appcompliance": "app_compliance",
    "app_compliance_2024_06_27": "app_compliance",
    "appcompliance2024_06_27": "app_compliance",
    "rest_appcompliance_2024_06_27": "app_compliance",
    "app_compliance_automation": "app_compliance",
    "appcomplianceautomation": "app_compliance",
    "azure_app_compliance": "app_compliance",
    "azure_app_compliance_2024_06_27": "app_compliance",
    "azure_appcompliance": "app_compliance",
    "azure_appcompliance_2024_06_27": "app_compliance",
    "azure_app_compliance_automation": "app_compliance",
    "azure_app_compliance_automation_2024_06_27": "app_compliance",
    "app_configuration_resource_manager": "app_configuration_resource_manager",
    "appconfiguration_resource_manager": "app_configuration_resource_manager",
    "app_configuration_resourcemanager": "app_configuration_resource_manager",
    "appconfigurationresourcemanager": "app_configuration_resource_manager",
    "app_configuration_resource_manager_2024_06_01": "app_configuration_resource_manager",
    "appconfiguration_resource_manager_2024_06_01": "app_configuration_resource_manager",
    "app_configuration_resourcemanager_2024_06_01": "app_configuration_resource_manager",
    "appconfigurationresourcemanager2024_06_01": "app_configuration_resource_manager",
    "rest_appconfiguration_2024_06_01": "app_configuration_resource_manager",
    "azure_app_configuration_resource_manager": "app_configuration_resource_manager",
    "azure_app_configuration_resource_manager_2024_06_01": "app_configuration_resource_manager",
    "azure_appconfiguration_resource_manager": "app_configuration_resource_manager",
    "azure_appconfiguration_resource_manager_2024_06_01": "app_configuration_resource_manager",
    "azure_app_configuration_resourcemanager": "app_configuration_resource_manager",
    "azure_app_configuration_resourcemanager_2024_06_01": "app_configuration_resource_manager",
    "app_configuration_data_plane": "app_configuration_data_plane",
    "appconfiguration_data_plane": "app_configuration_data_plane",
    "app_configuration_dataplane": "app_configuration_data_plane",
    "appconfigurationdataplane": "app_configuration_data_plane",
    "app_configuration_data_plane_2024_09_01": "app_configuration_data_plane",
    "appconfiguration_data_plane_2024_09_01": "app_configuration_data_plane",
    "app_configuration_dataplane_2024_09_01": "app_configuration_data_plane",
    "appconfigurationdataplane2024_09_01": "app_configuration_data_plane",
    "rest_data_plane_appconfiguration_2024_09_01": "app_configuration_data_plane",
    "azure_app_configuration_data_plane": "app_configuration_data_plane",
    "azure_app_configuration_data_plane_2024_09_01": "app_configuration_data_plane",
    "azure_appconfiguration_data_plane": "app_configuration_data_plane",
    "azure_appconfiguration_data_plane_2024_09_01": "app_configuration_data_plane",
    "azure_app_configuration_dataplane": "app_configuration_data_plane",
    "azure_app_configuration_dataplane_2024_09_01": "app_configuration_data_plane",
    "aks": "aks",
    "azure_aks": "aks",
    "azure_kubernetes_service": "aks",
    "kubernetes_service": "aks",
    "aks_2025_10_01": "aks",
    "rest_aks_2025_10_01": "aks",
    "azure_aks_2025_10_01": "aks",
    "azure_aks_service": "aks",
    "azure_aks_management": "aks",
    "azure_aks_management_2025_10_01": "aks",
    "batch_avatar": "batch_avatar",
    "batchavatar": "batch_avatar",
    "batch_avatar_2024_08_01": "batch_avatar",
    "azure_batch_avatar": "batch_avatar",
    "azure_batch_avatar_2024_08_01": "batch_avatar",
    "azure_ai_services_batch_avatar": "batch_avatar",
    "azure_ai_services_batch_avatar_2024_08_01": "batch_avatar",
    "ai_services_batch_avatar": "batch_avatar",
    "ai_services_batch_avatar_2024_08_01": "batch_avatar",
    "openai_2024_08_01": "batch_avatar",
    "azure_openai_2024_08_01": "batch_avatar",
    "azure_ai_services_openai_2024_08_01": "batch_avatar",
    "ai_services_openai_2024_08_01": "batch_avatar",
    "custom_vision": "custom_vision",
    "customvision": "custom_vision",
    "custom_vision_v3_3": "custom_vision",
    "customvision_v3_3": "custom_vision",
    "azure_custom_vision": "custom_vision",
    "azure_custom_vision_v3_3": "custom_vision",
    "azure_ai_services_custom_vision": "custom_vision",
    "azure_ai_services_custom_vision_v3_3": "custom_vision",
    "ai_services_custom_vision": "custom_vision",
    "ai_services_custom_vision_v3_3": "custom_vision",
    "custom_voice": "custom_voice",
    "customvoice": "custom_voice",
    "custom_voice_2024_02_01_preview": "custom_voice",
    "customvoice_2024_02_01_preview": "custom_voice",
    "azure_custom_voice": "custom_voice",
    "azure_custom_voice_2024_02_01_preview": "custom_voice",
    "azure_ai_services_custom_voice": "custom_voice",
    "azure_ai_services_custom_voice_2024_02_01_preview": "custom_voice",
    "ai_services_custom_voice": "custom_voice",
    "ai_services_custom_voice_2024_02_01_preview": "custom_voice",
    "face": "face",
    "face_v1_2": "face",
    "facev1_2": "face",
    "rest_face_v1_2": "face",
    "azure_face": "face",
    "azure_face_v1_2": "face",
    "azure_ai_services_face": "face",
    "azure_ai_services_face_v1_2": "face",
    "ai_services_face": "face",
    "ai_services_face_v1_2": "face",
    "health_insights": "health_insights",
    "healthinsights": "health_insights",
    "health_insights_2024_10_01": "health_insights",
    "azure_health_insights": "health_insights",
    "azure_health_insights_2024_10_01": "health_insights",
    "azure_ai_services_health_insights": "health_insights",
    "azure_ai_services_health_insights_2024_10_01": "health_insights",
    "ai_services_health_insights": "health_insights",
    "ai_services_health_insights_2024_10_01": "health_insights",
    "luis": "luis",
    "luis_v3_0": "luis",
    "luisv3_0": "luis",
    "rest_luis_v3_0": "luis",
    "azure_luis": "luis",
    "azure_luis_v3_0": "luis",
    "azure_ai_services_luis": "luis",
    "azure_ai_services_luis_v3_0": "luis",
    "ai_services_luis": "luis",
    "ai_services_luis_v3_0": "luis",
    "personalizer": "personalizer",
    "personalizer_v1_0": "personalizer",
    "personalizerv1_0": "personalizer",
    "rest_personalizer_v1_0": "personalizer",
    "azure_personalizer": "personalizer",
    "azure_personalizer_v1_0": "personalizer",
    "azure_ai_services_personalizer": "personalizer",
    "azure_ai_services_personalizer_v1_0": "personalizer",
    "ai_services_personalizer": "personalizer",
    "ai_services_personalizer_v1_0": "personalizer",
    "speaker_recognition": "speaker_recognition",
    "speakerrecognition": "speaker_recognition",
    "speaker_recognition_2021_09_05": "speaker_recognition",
    "rest_speaker_recognition_2021_09_05": "speaker_recognition",
    "azure_speaker_recognition": "speaker_recognition",
    "azure_speaker_recognition_2021_09_05": "speaker_recognition",
    "azure_ai_services_speaker_recognition": "speaker_recognition",
    "azure_ai_services_speaker_recognition_2021_09_05": "speaker_recognition",
    "ai_services_speaker_recognition": "speaker_recognition",
    "ai_services_speaker_recognition_2021_09_05": "speaker_recognition",
    "translator": "translator",
    "translator_v3_0": "translator",
    "rest_translator_v3_0": "translator",
    "azure_translator": "translator",
    "azure_translator_v3_0": "translator",
    "azure_ai_services_translator": "translator",
    "azure_ai_services_translator_v3_0": "translator",
    "ai_services_translator": "translator",
    "ai_services_translator_v3_0": "translator",
    "video_translation": "video_translation",
    "videotranslation": "video_translation",
    "video_translation_2026_03_01": "video_translation",
    "rest_aiservices_videotranslation_2026_03_01": "video_translation",
    "azure_video_translation": "video_translation",
    "azure_video_translation_2026_03_01": "video_translation",
    "azure_ai_services_video_translation": "video_translation",
    "azure_ai_services_video_translation_2026_03_01": "video_translation",
    "ai_services_video_translation": "video_translation",
    "ai_services_video_translation_2026_03_01": "video_translation",
    "computer_vision": "computer_vision",
    "computervision": "computer_vision",
    "computer_vision_2023_04_01_preview": "computer_vision",
    "computer_vision_v4_0_preview2023_04_01": "computer_vision",
    "computer_vision_v4_0_preview_2023_04_01": "computer_vision",
    "azure_computer_vision": "computer_vision",
    "azure_computer_vision_2023_04_01_preview": "computer_vision",
    "azure_computer_vision_v4_0_preview2023_04_01": "computer_vision",
    "azure_computer_vision_v4_0_preview_2023_04_01": "computer_vision",
    "azure_ai_services_computer_vision": "computer_vision",
    "azure_ai_services_computer_vision_v4_0_preview2023_04_01": "computer_vision",
    "azure_ai_services_computer_vision_v4_0_preview_2023_04_01": "computer_vision",
    "ai_services_computer_vision": "computer_vision",
    "ai_services_computer_vision_v4_0_preview2023_04_01": "computer_vision",
    "ai_services_computer_vision_v4_0_preview_2023_04_01": "computer_vision",
    "openai": "openai",
    "open_ai": "openai",
    "azure_openai": "openai",
    "azure_open_ai": "openai",
    "openai_2024_10_21": "openai",
    "azure_openai_2024_10_21": "openai",
    "azure_ai_services_openai": "openai",
    "azure_ai_services_openai_2024_10_21": "openai",
    "ai_services_openai": "openai",
    "ai_services_openai_2024_10_21": "openai",
}


@dataclass
class ContractCoverage:
    service: str
    request_validation: bool
    typed_success_fixtures: bool
    negative_contract_tests: bool

    @property
    def strict_all_three(self) -> bool:
        return (
            self.request_validation
            and self.typed_success_fixtures
            and self.negative_contract_tests
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
    # Blob baseline tests currently live in provider_objectstorage_test.go.
    if service == "blob":
        legacy = SERVER_DIR / "provider_objectstorage_test.go"
        if legacy.exists():
            files.append(legacy)
    return files


def analyze_service(service: str, source_files: list[Path]) -> ContractCoverage:
    source_text = "\n".join(read_text(path) for path in source_files)
    test_text = "\n".join(read_text(path) for path in discover_test_files(service))

    request_validation = has_any(
        source_text,
        [
            r"StatusBadRequest",
            r"InvalidRequest",
            r"\bmissing\b",
            r"\binvalid\b",
        ],
    )
    typed_success_fixtures = has_any(
        source_text,
        [
            r"respondJSON\(",
            r"respondAzureImplemented\(",
            r"xml\.Marshal",
            r"WriteHeader\(http\.Status",
            r"Content-Type",
        ],
    )
    negative_contract_tests = has_any(
        test_text,
        [
            r"StatusBadRequest",
            r"StatusUnauthorized",
            r"StatusForbidden",
            r"StatusNotFound",
            r"StatusConflict",
            r"StatusNotImplemented",
        ],
    )

    return ContractCoverage(
        service=service,
        request_validation=request_validation,
        typed_success_fixtures=typed_success_fixtures,
        negative_contract_tests=negative_contract_tests,
    )


def summarize(rows: list[ContractCoverage]) -> dict[str, int]:
    return {
        "total": len(rows),
        "request_validation": sum(1 for row in rows if row.request_validation),
        "typed_success_fixtures": sum(1 for row in rows if row.typed_success_fixtures),
        "negative_contract_tests": sum(1 for row in rows if row.negative_contract_tests),
        "strict_all_three": sum(1 for row in rows if row.strict_all_three),
    }


def render_table(rows: list[ContractCoverage]) -> str:
    header = (
        "service".ljust(16)
        + " request_validation  typed_success_fixtures  negative_contract_tests  strict_all_three"
    )
    lines = [header, "-" * len(header)]
    for row in rows:
        lines.append(
            row.service.ljust(16)
            + f" {str(row.request_validation):<18}"
            + f" {str(row.typed_success_fixtures):<22}"
            + f" {str(row.negative_contract_tests):<24}"
            + f" {str(row.strict_all_three):<15}"
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


def should_fail(rows: list[ContractCoverage], fail_on: set[str]) -> bool:
    if not fail_on:
        return False
    checks = {
        "validation": any(not row.request_validation for row in rows),
        "fixtures": any(not row.typed_success_fixtures for row in rows),
        "negative": any(not row.negative_contract_tests for row in rows),
        "any": any(not row.strict_all_three for row in rows),
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
        help="Comma-separated fail gates: validation,fixtures,negative,any",
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
                "mode": "contract",
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
        print(f"azure-contract-coverage: required service {required!r} not discovered", file=sys.stderr)
        return 2
    if required and required not in selected:
        print(f"azure-contract-coverage: required service {required!r} not matched by selector", file=sys.stderr)
        return 2

    rows = [analyze_service(service, files_by_service[service]) for service in selected]
    summary = summarize(rows)

    payload = {
        "provider": "azure",
        "mode": "contract",
        "service_selector": service_filter,
        "required_service": required,
        "summary": summary,
        "services": [asdict(row) | {"strict_all_three": row.strict_all_three} for row in rows],
    }
    write_report_json(args.report_json, payload)

    if args.format == "json":
        print(json.dumps(payload, indent=2, sort_keys=True))
    else:
        print(render_table(rows))
        print()
        print(json.dumps(summary, indent=2, sort_keys=True))
        if args.verbose:
            failing = [row for row in rows if not row.strict_all_three]
            if failing:
                print()
                print("Failing services")
                for row in failing:
                    print(
                        f"- {row.service}: validation={row.request_validation} "
                        f"fixtures={row.typed_success_fixtures} negative={row.negative_contract_tests}"
                    )

    fail_on = parse_fail_on(args.fail_on)
    if should_fail(rows, fail_on):
        failing = [row.service for row in rows if not row.strict_all_three]
        if failing:
            print(
                "azure-contract-coverage: failing services: "
                + ", ".join(failing),
                file=sys.stderr,
            )
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
