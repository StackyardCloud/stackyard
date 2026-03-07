#!/usr/bin/env python3
"""
GCP provider contract coverage analyzer for Stackyard.

This script performs static coverage checks for three quality areas:
1) request validation paths in handler source,
2) typed success response fixtures in handler source,
3) negative contract tests in provider test files.

It is intentionally heuristic-based and fast, so contributors can run it often
while iterating on staged emulation handlers.
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
    "telcoautomation": "telcoautomation",
    "telcoautomation-apiv1": "telcoautomation",
    "telcoautomation_apiv1": "telcoautomation",
    "telco-automation": "telcoautomation",
    "telco_automation": "telcoautomation",
    "gcp-telcoautomation": "telcoautomation",
    "gcp-telco-automation": "telcoautomation",
    "rapid-migration-assessment": "rapidmigrationassessment",
    "rapid_migration_assessment": "rapidmigrationassessment",
    "rma": "rapidmigrationassessment",
    "vmmigration": "vmmigration",
    "vmmigration-apiv1": "vmmigration",
    "vmmigration_apiv1": "vmmigration",
    "vm-migration": "vmmigration",
    "vm_migration": "vmmigration",
    "cloud-vm-migration": "vmmigration",
    "cloud_vm_migration": "vmmigration",
    "gcp-vm-migration": "vmmigration",
    "gcp-vmmigration": "vmmigration",
    "vpcaccess": "vpcaccess",
    "vpcaccess-apiv1": "vpcaccess",
    "vpcaccess_apiv1": "vpcaccess",
    "vpc-access": "vpcaccess",
    "vpc_access": "vpcaccess",
    "serverless-vpc-access": "vpcaccess",
    "serverless_vpc_access": "vpcaccess",
    "gcp-serverless-vpc-access": "vpcaccess",
    "gcp-vpcaccess": "vpcaccess",
    "gcp-vpc-access": "vpcaccess",
    "websecurityscanner": "websecurityscanner",
    "websecurityscanner-apiv1": "websecurityscanner",
    "websecurityscanner_apiv1": "websecurityscanner",
    "web-security-scanner": "websecurityscanner",
    "web_security_scanner": "websecurityscanner",
    "gcp-websecurityscanner": "websecurityscanner",
    "gcp-web-security-scanner": "websecurityscanner",
    "vmwareengine": "vmwareengine",
    "vmwareengine-apiv1": "vmwareengine",
    "vmwareengine_apiv1": "vmwareengine",
    "vmware-engine": "vmwareengine",
    "vmware_engine": "vmwareengine",
    "cloud-vmware-engine": "vmwareengine",
    "cloud_vmware_engine": "vmwareengine",
    "gcp-vmware-engine": "vmwareengine",
    "gcp-vmwareengine": "vmwareengine",
    "recaptcha-enterprise": "recaptchaenterprise",
    "recaptcha_enterprise": "recaptchaenterprise",
    "recaptchaenterprise-v2": "recaptchaenterprise",
    "recaptchaenterprise_v2": "recaptchaenterprise",
    "recaptchaenterprise-v2-apiv1": "recaptchaenterprise",
    "recommendation-engine": "recommendationengine",
    "recommendation_engine": "recommendationengine",
    "recommendations-ai": "recommendationengine",
    "recommendations_ai": "recommendationengine",
    "recommendationengine-v1beta1": "recommendationengine",
    "recommendationengine_v1beta1": "recommendationengine",
    "recommendationengine-apiv1beta1": "recommendationengine",
    "talent": "talent",
    "talent-apiv4": "talent",
    "talent_apiv4": "talent",
    "cloud-talent": "talent",
    "cloud_talent": "talent",
    "talent-solution": "talent",
    "talentsolution": "talent",
    "gcp-talent-solution": "talent",
    "retail": "retail",
    "retail-apiv2": "retail",
    "retail_apiv2": "retail",
    "vertex-ai-search-commerce": "retail",
    "vertex_ai_search_commerce": "retail",
    "vertexaisearchforcommerce": "retail",
    "recommender-apiv1": "recommender",
    "recommender_apiv1": "recommender",
    "gcp-recommender": "recommender",
    "redis-cluster": "redis_cluster",
    "redis_cluster": "redis_cluster",
    "redis-cluster-apiv1": "redis_cluster",
    "memorystore-redis-cluster": "redis_cluster",
    "redis": "redis",
    "redis-apiv1": "redis",
    "redis_apiv1": "redis",
    "memorystore-redis": "redis",
    "cloud-memorystore-redis": "redis",
    "gcp-redis": "redis",
    "resourcemanager": "resourcemanager",
    "resourcemanager-apiv2": "resourcemanager",
    "resourcemanager-apiv3": "resourcemanager",
    "cloudresourcemanager": "resourcemanager",
    "resourcemanager_v2": "resourcemanager",
    "resourcemanager_v3": "resourcemanager",
    "cloudresourcemanager-v3": "resourcemanager",
    "run": "run",
    "run-apiv2": "run",
    "run_apiv2": "run",
    "cloud-run-admin": "run",
    "cloud_run_admin": "run",
    "cloudrunadmin": "run",
    "scheduler": "scheduler",
    "scheduler-apiv1": "scheduler",
    "scheduler_apiv1": "scheduler",
    "cloud-scheduler": "scheduler",
    "cloud_scheduler": "scheduler",
    "cloudscheduler": "scheduler",
    "workflows": "workflows",
    "workflows-apiv1": "workflows",
    "workflows_apiv1": "workflows",
    "workflow": "workflows",
    "gcp-workflows": "workflows",
    "gcp-workflows-apiv1": "workflows",
    "workflow-executions": "workflow_executions",
    "workflow_executions": "workflow_executions",
    "workflow-executions-apiv1": "workflow_executions",
    "workflow_executions_apiv1": "workflow_executions",
    "workflowexecutions": "workflow_executions",
    "workflowexecutions-apiv1": "workflow_executions",
    "workflows-executions": "workflow_executions",
    "workflows_executions": "workflow_executions",
    "workflows-executions-apiv1": "workflow_executions",
    "workflows_executions_apiv1": "workflow_executions",
    "gcp-workflow-executions": "workflow_executions",
    "gcp-workflow-executions-apiv1": "workflow_executions",
    "workstations": "workstations",
    "workstations-apiv1": "workstations",
    "workstations_apiv1": "workstations",
    "workstation": "workstations",
    "cloud-workstations": "workstations",
    "cloud_workstations": "workstations",
    "cloudworkstations": "workstations",
    "gcp-workstations": "workstations",
    "gcp-workstations-apiv1": "workstations",
    "speech": "speech",
    "speech-apiv1": "speech",
    "speech_apiv1": "speech",
    "cloud-speech": "speech",
    "cloud_speech": "speech",
    "cloudspeech": "speech",
    "speech-to-text": "speech",
    "speech_to_text": "speech",
    "cloud-speech-to-text": "speech",
    "gcp-cloud-speech": "speech",
    "texttospeech": "texttospeech",
    "texttospeech-apiv1": "texttospeech",
    "texttospeech_apiv1": "texttospeech",
    "text-to-speech": "texttospeech",
    "text_to_speech": "texttospeech",
    "tts": "texttospeech",
    "gcp-texttospeech": "texttospeech",
    "cloud-texttospeech": "texttospeech",
    "translate": "translate",
    "translate-apiv3": "translate",
    "translate_apiv3": "translate",
    "cloud-translate": "translate",
    "cloud_translate": "translate",
    "cloud-translate-v3": "translate",
    "cloud_translate_v3": "translate",
    "gcp-translate": "translate",
    "gcp-translate-v3": "translate",
    "video_livestream": "video_livestream",
    "video-livestream": "video_livestream",
    "video-livestream-apiv1": "video_livestream",
    "video_livestream_apiv1": "video_livestream",
    "livestream": "video_livestream",
    "livestream-apiv1": "video_livestream",
    "gcp-video-livestream": "video_livestream",
    "gcp-livestream": "video_livestream",
    "video_transcoder": "video_transcoder",
    "video-transcoder": "video_transcoder",
    "video-transcoder-apiv1": "video_transcoder",
    "video_transcoder_apiv1": "video_transcoder",
    "transcoder": "video_transcoder",
    "transcoder-apiv1": "video_transcoder",
    "gcp-video-transcoder": "video_transcoder",
    "gcp-transcoder": "video_transcoder",
    "video_stitcher": "video_stitcher",
    "video-stitcher": "video_stitcher",
    "video-stitcher-apiv1": "video_stitcher",
    "video_stitcher_apiv1": "video_stitcher",
    "stitcher": "video_stitcher",
    "stitcher-apiv1": "video_stitcher",
    "gcp-video-stitcher": "video_stitcher",
    "gcp-stitcher": "video_stitcher",
    "videointelligence": "videointelligence",
    "video-intelligence": "videointelligence",
    "videointelligence-apiv1": "videointelligence",
    "videointelligence_apiv1": "videointelligence",
    "cloud-video-intelligence": "videointelligence",
    "cloud_video_intelligence": "videointelligence",
    "gcp-video-intelligence": "videointelligence",
    "visionai": "visionai",
    "visionai-apiv1": "visionai",
    "visionai_apiv1": "visionai",
    "vision-ai": "visionai",
    "vision_ai": "visionai",
    "gcp-vision-ai": "visionai",
    "vision": "vision",
    "vision-apiv1": "vision",
    "vision_apiv1": "vision",
    "vision-v2": "vision",
    "vision_v2": "vision",
    "vision-v2-apiv1": "vision",
    "vision_v2_apiv1": "vision",
    "cloud-vision": "vision",
    "cloud_vision": "vision",
    "cloud-vision-v2": "vision",
    "cloud_vision_v2": "vision",
    "cloudvision": "vision",
    "gcp-cloud-vision": "vision",
    "gcp-cloud-vision-v2": "vision",
    "trace": "trace",
    "trace-apiv2": "trace",
    "trace_apiv2": "trace",
    "stackdriver-trace": "trace",
    "stackdriver_trace": "trace",
    "cloudtrace": "trace",
    "cloud-trace": "trace",
    "gcp-trace": "trace",
    "trace_v1": "trace_v1",
    "trace-v1": "trace_v1",
    "trace-apiv1": "trace_v1",
    "trace_apiv1": "trace_v1",
    "stackdriver-trace-v1": "trace_v1",
    "stackdriver_trace_v1": "trace_v1",
    "cloudtrace-v1": "trace_v1",
    "gcp-trace-v1": "trace_v1",
    "tpu": "tpu",
    "tpu-apiv1": "tpu",
    "tpu_apiv1": "tpu",
    "cloud-tpu": "tpu",
    "cloud_tpu": "tpu",
    "cloudtpu": "tpu",
    "gcp-tpu": "tpu",
    "tensor-processing-unit": "tpu",
    "speech_v2": "speech_v2",
    "speech-v2": "speech_v2",
    "speech-apiv2": "speech_v2",
    "speech_apiv2": "speech_v2",
    "cloud-speech-v2": "speech_v2",
    "cloud_speech_v2": "speech_v2",
    "cloud-speech-to-text-v2": "speech_v2",
    "cloud_speech_to_text_v2": "speech_v2",
    "gcp-cloud-speech-v2": "speech_v2",
    "spanner": "spanner",
    "spanner-apiv1": "spanner",
    "spanner_apiv1": "spanner",
    "cloud-spanner": "spanner",
    "cloud_spanner": "spanner",
    "cloudspanner": "spanner",
    "gcp-cloud-spanner": "spanner",
    "spanner_executor": "spanner_executor",
    "spanner-executor": "spanner_executor",
    "spanner-executor-apiv1": "spanner_executor",
    "spanner_executor_apiv1": "spanner_executor",
    "cloud-spanner-executor": "spanner_executor",
    "cloud_spanner_executor": "spanner_executor",
    "cloudspannerexecutor": "spanner_executor",
    "gcp-cloud-spanner-executor": "spanner_executor",
    "spanner_admin_database": "spanner_admin_database",
    "spanner-admin-database": "spanner_admin_database",
    "spanner-admin-database-apiv1": "spanner_admin_database",
    "spanner_admin_database_apiv1": "spanner_admin_database",
    "cloud-spanner-admin-database": "spanner_admin_database",
    "cloud_spanner_admin_database": "spanner_admin_database",
    "cloudspanneradmindatabase": "spanner_admin_database",
    "gcp-cloud-spanner-admin-database": "spanner_admin_database",
    "spanner_admin_instance": "spanner_admin_instance",
    "spanner-admin-instance": "spanner_admin_instance",
    "spanner-admin-instance-apiv1": "spanner_admin_instance",
    "spanner_admin_instance_apiv1": "spanner_admin_instance",
    "cloud-spanner-admin-instance": "spanner_admin_instance",
    "cloud_spanner_admin_instance": "spanner_admin_instance",
    "cloudspanneradmininstance": "spanner_admin_instance",
    "gcp-cloud-spanner-admin-instance": "spanner_admin_instance",
    "spanner_adapter": "spanner_adapter",
    "spanner-adapter": "spanner_adapter",
    "spanner-adapter-apiv1": "spanner_adapter",
    "spanner_adapter_apiv1": "spanner_adapter",
    "cloud-spanner-adapter": "spanner_adapter",
    "cloud_spanner_adapter": "spanner_adapter",
    "cloudspanneradapter": "spanner_adapter",
    "gcp-cloud-spanner-adapter": "spanner_adapter",
    "shell": "shell",
    "shell-apiv1": "shell",
    "shell_apiv1": "shell",
    "cloud-shell": "shell",
    "cloud_shell": "shell",
    "cloudshell": "shell",
    "gcp-cloud-shell": "shell",
    "storage": "storage",
    "storage-apiv1": "storage",
    "storage_apiv1": "storage",
    "gcs": "storage",
    "cloud-storage": "storage",
    "cloud_storage": "storage",
    "cloudstorage": "storage",
    "gcp-cloud-storage": "storage",
    "storagebatchoperations": "storagebatchoperations",
    "storagebatchoperations-apiv1": "storagebatchoperations",
    "storagebatchoperations_apiv1": "storagebatchoperations",
    "storage-batch-operations": "storagebatchoperations",
    "storage_batch_operations": "storagebatchoperations",
    "cloud-storage-batch-operations": "storagebatchoperations",
    "cloud_storage_batch_operations": "storagebatchoperations",
    "gcp-storage-batch-operations": "storagebatchoperations",
    "storageinsights": "storageinsights",
    "storageinsights-apiv1": "storageinsights",
    "storageinsights_apiv1": "storageinsights",
    "storage-insights": "storageinsights",
    "storage_insights": "storageinsights",
    "cloud-storage-insights": "storageinsights",
    "cloud_storage_insights": "storageinsights",
    "cloudstorageinsights": "storageinsights",
    "gcp-storage-insights": "storageinsights",
    "storagetransfer": "storagetransfer",
    "storagetransfer-apiv1": "storagetransfer",
    "storagetransfer_apiv1": "storagetransfer",
    "storage-transfer": "storagetransfer",
    "storage_transfer": "storagetransfer",
    "cloud-storage-transfer": "storagetransfer",
    "cloud_storage_transfer": "storagetransfer",
    "cloudstoragetransfer": "storagetransfer",
    "gcp-storage-transfer": "storagetransfer",
    "streetview_publish": "streetview_publish",
    "streetview-publish": "streetview_publish",
    "streetview-publish-apiv1": "streetview_publish",
    "streetview_publish_apiv1": "streetview_publish",
    "streetviewpublish": "streetview_publish",
    "streetviewpublish-apiv1": "streetview_publish",
    "street-view-publish": "streetview_publish",
    "street_view_publish": "streetview_publish",
    "gcp-street-view-publish": "streetview_publish",
    "shopping_css": "shopping_css",
    "shopping-css": "shopping_css",
    "shopping-css-apiv1": "shopping_css",
    "shopping_css_apiv1": "shopping_css",
    "css": "shopping_css",
    "shoppingcss": "shopping_css",
    "gcp-shopping-css": "shopping_css",
    "shopping_merchant_accounts": "shopping_merchant_accounts",
    "shopping-merchant-accounts": "shopping_merchant_accounts",
    "shopping-merchant-accounts-apiv1": "shopping_merchant_accounts",
    "shopping_merchant_accounts_apiv1": "shopping_merchant_accounts",
    "merchant_accounts": "shopping_merchant_accounts",
    "merchant-accounts": "shopping_merchant_accounts",
    "merchantaccounts": "shopping_merchant_accounts",
    "gcp-shopping-merchant-accounts": "shopping_merchant_accounts",
    "shopping_merchant_conversions": "shopping_merchant_conversions",
    "shopping-merchant-conversions": "shopping_merchant_conversions",
    "shopping-merchant-conversions-apiv1": "shopping_merchant_conversions",
    "shopping_merchant_conversions_apiv1": "shopping_merchant_conversions",
    "merchant_conversions": "shopping_merchant_conversions",
    "merchant-conversions": "shopping_merchant_conversions",
    "merchantconversions": "shopping_merchant_conversions",
    "gcp-shopping-merchant-conversions": "shopping_merchant_conversions",
    "shopping_merchant_datasources": "shopping_merchant_datasources",
    "shopping-merchant-datasources": "shopping_merchant_datasources",
    "shopping-merchant-datasources-apiv1": "shopping_merchant_datasources",
    "shopping_merchant_datasources_apiv1": "shopping_merchant_datasources",
    "merchant_datasources": "shopping_merchant_datasources",
    "merchant-datasources": "shopping_merchant_datasources",
    "merchantdatasources": "shopping_merchant_datasources",
    "gcp-shopping-merchant-datasources": "shopping_merchant_datasources",
    "shopping_merchant_inventories": "shopping_merchant_inventories",
    "shopping-merchant-inventories": "shopping_merchant_inventories",
    "shopping-merchant-inventories-apiv1": "shopping_merchant_inventories",
    "shopping_merchant_inventories_apiv1": "shopping_merchant_inventories",
    "merchant_inventories": "shopping_merchant_inventories",
    "merchant-inventories": "shopping_merchant_inventories",
    "merchantinventories": "shopping_merchant_inventories",
    "gcp-shopping-merchant-inventories": "shopping_merchant_inventories",
    "shopping_merchant_issueresolution": "shopping_merchant_issueresolution",
    "shopping-merchant-issueresolution": "shopping_merchant_issueresolution",
    "shopping-merchant-issueresolution-apiv1": "shopping_merchant_issueresolution",
    "shopping_merchant_issueresolution_apiv1": "shopping_merchant_issueresolution",
    "merchant_issueresolution": "shopping_merchant_issueresolution",
    "merchant-issueresolution": "shopping_merchant_issueresolution",
    "merchantissueresolution": "shopping_merchant_issueresolution",
    "gcp-shopping-merchant-issueresolution": "shopping_merchant_issueresolution",
    "shopping_merchant_lfp": "shopping_merchant_lfp",
    "shopping-merchant-lfp": "shopping_merchant_lfp",
    "shopping-merchant-lfp-apiv1": "shopping_merchant_lfp",
    "shopping_merchant_lfp_apiv1": "shopping_merchant_lfp",
    "merchant_lfp": "shopping_merchant_lfp",
    "merchant-lfp": "shopping_merchant_lfp",
    "merchantlfp": "shopping_merchant_lfp",
    "gcp-shopping-merchant-lfp": "shopping_merchant_lfp",
    "shopping_merchant_notifications": "shopping_merchant_notifications",
    "shopping-merchant-notifications": "shopping_merchant_notifications",
    "shopping-merchant-notifications-apiv1": "shopping_merchant_notifications",
    "shopping_merchant_notifications_apiv1": "shopping_merchant_notifications",
    "merchant_notifications": "shopping_merchant_notifications",
    "merchant-notifications": "shopping_merchant_notifications",
    "merchantnotifications": "shopping_merchant_notifications",
    "gcp-shopping-merchant-notifications": "shopping_merchant_notifications",
    "shopping_merchant_ordertracking": "shopping_merchant_ordertracking",
    "shopping-merchant-ordertracking": "shopping_merchant_ordertracking",
    "shopping-merchant-ordertracking-apiv1": "shopping_merchant_ordertracking",
    "shopping_merchant_ordertracking_apiv1": "shopping_merchant_ordertracking",
    "merchant_ordertracking": "shopping_merchant_ordertracking",
    "merchant-ordertracking": "shopping_merchant_ordertracking",
    "merchantordertracking": "shopping_merchant_ordertracking",
    "gcp-shopping-merchant-ordertracking": "shopping_merchant_ordertracking",
    "shopping_merchant_products": "shopping_merchant_products",
    "shopping-merchant-products": "shopping_merchant_products",
    "shopping-merchant-products-apiv1": "shopping_merchant_products",
    "shopping_merchant_products_apiv1": "shopping_merchant_products",
    "merchant_products": "shopping_merchant_products",
    "merchant-products": "shopping_merchant_products",
    "merchantproducts": "shopping_merchant_products",
    "gcp-shopping-merchant-products": "shopping_merchant_products",
    "shopping_merchant_productstudio": "shopping_merchant_productstudio",
    "shopping-merchant-productstudio": "shopping_merchant_productstudio",
    "shopping-merchant-productstudio-apiv1alpha": "shopping_merchant_productstudio",
    "shopping_merchant_productstudio_apiv1alpha": "shopping_merchant_productstudio",
    "merchant_productstudio": "shopping_merchant_productstudio",
    "merchant-productstudio": "shopping_merchant_productstudio",
    "merchantproductstudio": "shopping_merchant_productstudio",
    "gcp-shopping-merchant-productstudio": "shopping_merchant_productstudio",
    "shopping_merchant_promotions": "shopping_merchant_promotions",
    "shopping-merchant-promotions": "shopping_merchant_promotions",
    "shopping-merchant-promotions-apiv1": "shopping_merchant_promotions",
    "shopping_merchant_promotions_apiv1": "shopping_merchant_promotions",
    "merchant_promotions": "shopping_merchant_promotions",
    "merchant-promotions": "shopping_merchant_promotions",
    "merchantpromotions": "shopping_merchant_promotions",
    "gcp-shopping-merchant-promotions": "shopping_merchant_promotions",
    "shopping_merchant_reports": "shopping_merchant_reports",
    "shopping-merchant-reports": "shopping_merchant_reports",
    "shopping-merchant-reports-apiv1": "shopping_merchant_reports",
    "shopping_merchant_reports_apiv1": "shopping_merchant_reports",
    "merchant_reports": "shopping_merchant_reports",
    "merchant-reports": "shopping_merchant_reports",
    "merchantreports": "shopping_merchant_reports",
    "gcp-shopping-merchant-reports": "shopping_merchant_reports",
    "shopping_merchant_reviews": "shopping_merchant_reviews",
    "shopping-merchant-reviews": "shopping_merchant_reviews",
    "shopping-merchant-reviews-apiv1beta": "shopping_merchant_reviews",
    "shopping_merchant_reviews_apiv1beta": "shopping_merchant_reviews",
    "merchant_reviews": "shopping_merchant_reviews",
    "merchant-reviews": "shopping_merchant_reviews",
    "merchantreviews": "shopping_merchant_reviews",
    "gcp-shopping-merchant-reviews": "shopping_merchant_reviews",
    "shopping_merchant_quota": "shopping_merchant_quota",
    "shopping-merchant-quota": "shopping_merchant_quota",
    "shopping-merchant-quota-apiv1": "shopping_merchant_quota",
    "shopping_merchant_quota_apiv1": "shopping_merchant_quota",
    "merchant_quota": "shopping_merchant_quota",
    "merchant-quota": "shopping_merchant_quota",
    "merchantquota": "shopping_merchant_quota",
    "gcp-shopping-merchant-quota": "shopping_merchant_quota",
    "servicecontrol": "servicecontrol",
    "servicecontrol-apiv1": "servicecontrol",
    "servicecontrol_apiv1": "servicecontrol",
    "service-control": "servicecontrol",
    "service_control": "servicecontrol",
    "gcp-service-control": "servicecontrol",
    "webrisk": "webrisk",
    "webrisk-apiv1": "webrisk",
    "webrisk_apiv1": "webrisk",
    "web-risk": "webrisk",
    "web_risk": "webrisk",
    "gcp-webrisk": "webrisk",
    "gcp-web-risk": "webrisk",
    "support": "support",
    "support-apiv2": "support",
    "support_apiv2": "support",
    "cloud-support": "support",
    "cloud_support": "support",
    "cloudsupport": "support",
    "gcp-cloud-support": "support",
    "serviceusage": "serviceusage",
    "serviceusage-apiv1": "serviceusage",
    "serviceusage_apiv1": "serviceusage",
    "service-usage": "serviceusage",
    "service_usage": "serviceusage",
    "gcp-service-usage": "serviceusage",
    "servicemanagement": "servicemanagement",
    "servicemanagement-apiv1": "servicemanagement",
    "servicemanagement_apiv1": "servicemanagement",
    "service-management": "servicemanagement",
    "service_management": "servicemanagement",
    "gcp-service-management": "servicemanagement",
    "servicehealth": "servicehealth",
    "servicehealth-apiv1": "servicehealth",
    "servicehealth_apiv1": "servicehealth",
    "service-health": "servicehealth",
    "service_health": "servicehealth",
    "gcp-service-health": "servicehealth",
    "servicedirectory": "servicedirectory",
    "servicedirectory-apiv1": "servicedirectory",
    "servicedirectory_apiv1": "servicedirectory",
    "service-directory": "servicedirectory",
    "service_directory": "servicedirectory",
    "gcp-service-directory": "servicedirectory",
    "secretmanager": "secretmanager",
    "secretmanager-apiv1": "secretmanager",
    "secretmanager_apiv1": "secretmanager",
    "secret-manager": "secretmanager",
    "secret_manager": "secretmanager",
    "gcp-secret-manager": "secretmanager",
    "securesourcemanager": "securesourcemanager",
    "securesourcemanager-apiv1": "securesourcemanager",
    "securesourcemanager_apiv1": "securesourcemanager",
    "secure-source-manager": "securesourcemanager",
    "secure_source_manager": "securesourcemanager",
    "gcp-secure-source-manager": "securesourcemanager",
    "securitycenter": "securitycenter",
    "securitycenter-apiv1": "securitycenter",
    "securitycenter_apiv1": "securitycenter",
    "securitycenter-apiv2": "securitycenter",
    "securitycenter_apiv2": "securitycenter",
    "securitycenter-v2": "securitycenter",
    "security-center": "securitycenter",
    "security-center-v2": "securitycenter",
    "security_center": "securitycenter",
    "security_center_v2": "securitycenter",
    "scc": "securitycenter",
    "gcp-security-command-center": "securitycenter",
    "scc-v2": "securitycenter",
    "gcp-security-command-center-v2": "securitycenter",
    "securitycentermanagement": "securitycentermanagement",
    "securitycentermanagement-apiv1": "securitycentermanagement",
    "securitycentermanagement_apiv1": "securitycentermanagement",
    "security-center-management": "securitycentermanagement",
    "security_center_management": "securitycentermanagement",
    "scc-management": "securitycentermanagement",
    "gcp-security-center-management": "securitycentermanagement",
    "securityposture": "securityposture",
    "securityposture-apiv1": "securityposture",
    "securityposture_apiv1": "securityposture",
    "security-posture": "securityposture",
    "security_posture": "securityposture",
    "gcp-security-posture": "securityposture",
    "security_privateca": "security_privateca",
    "security-privateca": "security_privateca",
    "security-privateca-apiv1": "security_privateca",
    "security_privateca_apiv1": "security_privateca",
    "privateca": "security_privateca",
    "private-ca": "security_privateca",
    "private_ca": "security_privateca",
    "certificateauthority": "security_privateca",
    "certificate-authority": "security_privateca",
    "security_publicca": "security_publicca",
    "security-publicca": "security_publicca",
    "security-publicca-apiv1": "security_publicca",
    "security_publicca_apiv1": "security_publicca",
    "publicca": "security_publicca",
    "public-ca": "security_publicca",
    "public_ca": "security_publicca",
    "publiccertificateauthority": "security_publicca",
    "public-certificate-authority": "security_publicca",
}


VALIDATION_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("status_bad_request", re.compile(r"http\.StatusBadRequest")),
    ("status_unprocessable_entity", re.compile(r"http\.StatusUnprocessableEntity")),
    ("status_conflict", re.compile(r"http\.StatusConflict")),
    ("error_invalid_argument", re.compile(r'"InvalidArgument"')),
    ("error_failed_precondition", re.compile(r'"FailedPrecondition"')),
    ("error_out_of_range", re.compile(r'"OutOfRange"')),
]

NEGATIVE_TEST_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("expects_bad_request", re.compile(r"http\.StatusBadRequest")),
    ("expects_unprocessable_entity", re.compile(r"http\.StatusUnprocessableEntity")),
    ("expects_conflict", re.compile(r"http\.StatusConflict")),
    ("expects_not_found", re.compile(r"http\.StatusNotFound")),
    ("asserts_invalid_argument", re.compile(r'"InvalidArgument"')),
    ("asserts_failed_precondition", re.compile(r'"FailedPrecondition"')),
    ("asserts_out_of_range", re.compile(r'"OutOfRange"')),
]

EMPTY_SUCCESS_PAYLOAD_PATTERNS = (
    re.compile(r"^map\[string\]any\{\}$"),
    re.compile(r"^map\[string\]interface\{\}\{\}$"),
    re.compile(r"^\[\]any\{\}$"),
    re.compile(r"^\[\]interface\{\}\{\}$"),
    re.compile(r"^nil$"),
)


@dataclass
class ServiceCoverage:
    service: str
    source_file: str
    test_file: str | None
    request_validation: bool
    request_validation_signals: list[str]
    typed_success_fixtures: bool
    ok_response_calls: int
    typed_ok_response_calls: int
    generic_ok_response_calls: int
    negative_contract_tests: bool
    negative_test_functions: int
    negative_test_signals: list[str]
    respond_not_implemented_calls: int

    @property
    def score(self) -> int:
        return int(self.request_validation) + int(self.typed_success_fixtures) + int(self.negative_contract_tests)


def strip_ws(text: str) -> str:
    return re.sub(r"\s+", "", text)


def is_empty_success_payload(payload_expr: str) -> bool:
    compact = strip_ws(payload_expr)
    for pat in EMPTY_SUCCESS_PAYLOAD_PATTERNS:
        if pat.fullmatch(compact):
            return True
    return False


def list_service_sources() -> dict[str, Path]:
    out: dict[str, Path] = {}
    for path in sorted(SERVER_DIR.glob("provider_gcp_*.go")):
        if path.name.endswith("_test.go"):
            continue
        service = path.stem[len("provider_gcp_") :]
        out[service] = path
    return out


def split_top_level_commas(text: str) -> list[str]:
    items: list[str] = []
    start = 0
    depth_paren = 0
    depth_brace = 0
    depth_bracket = 0
    in_string: str | None = None
    escaped = False
    in_line_comment = False
    in_block_comment = False

    i = 0
    while i < len(text):
        ch = text[i]
        nxt = text[i + 1] if i + 1 < len(text) else ""

        if in_line_comment:
            if ch == "\n":
                in_line_comment = False
            i += 1
            continue
        if in_block_comment:
            if ch == "*" and nxt == "/":
                in_block_comment = False
                i += 2
                continue
            i += 1
            continue

        if in_string is not None:
            if in_string == "`":
                if ch == "`":
                    in_string = None
                i += 1
                continue
            if escaped:
                escaped = False
                i += 1
                continue
            if ch == "\\":
                escaped = True
                i += 1
                continue
            if ch == in_string:
                in_string = None
            i += 1
            continue

        if ch == "/" and nxt == "/":
            in_line_comment = True
            i += 2
            continue
        if ch == "/" and nxt == "*":
            in_block_comment = True
            i += 2
            continue
        if ch in ('"', "'", "`"):
            in_string = ch
            i += 1
            continue

        if ch == "(":
            depth_paren += 1
        elif ch == ")":
            depth_paren -= 1
        elif ch == "{":
            depth_brace += 1
        elif ch == "}":
            depth_brace -= 1
        elif ch == "[":
            depth_bracket += 1
        elif ch == "]":
            depth_bracket -= 1
        elif ch == "," and depth_paren == 0 and depth_brace == 0 and depth_bracket == 0:
            items.append(text[start:i].strip())
            start = i + 1

        i += 1

    tail = text[start:].strip()
    if tail:
        items.append(tail)
    return items


def find_function_calls(source: str, function_name: str) -> list[list[str]]:
    token = function_name + "("
    calls: list[list[str]] = []
    idx = 0
    while True:
        start = source.find(token, idx)
        if start < 0:
            break
        i = start + len(token)
        depth = 1
        in_string: str | None = None
        escaped = False
        in_line_comment = False
        in_block_comment = False

        while i < len(source):
            ch = source[i]
            nxt = source[i + 1] if i + 1 < len(source) else ""

            if in_line_comment:
                if ch == "\n":
                    in_line_comment = False
                i += 1
                continue
            if in_block_comment:
                if ch == "*" and nxt == "/":
                    in_block_comment = False
                    i += 2
                    continue
                i += 1
                continue

            if in_string is not None:
                if in_string == "`":
                    if ch == "`":
                        in_string = None
                    i += 1
                    continue
                if escaped:
                    escaped = False
                    i += 1
                    continue
                if ch == "\\":
                    escaped = True
                    i += 1
                    continue
                if ch == in_string:
                    in_string = None
                i += 1
                continue

            if ch == "/" and nxt == "/":
                in_line_comment = True
                i += 2
                continue
            if ch == "/" and nxt == "*":
                in_block_comment = True
                i += 2
                continue
            if ch in ('"', "'", "`"):
                in_string = ch
                i += 1
                continue

            if ch == "(":
                depth += 1
            elif ch == ")":
                depth -= 1
                if depth == 0:
                    args_text = source[start + len(token) : i]
                    calls.append(split_top_level_commas(args_text))
                    idx = i + 1
                    break
            i += 1
        else:
            idx = start + len(token)
    return calls


def extract_test_functions(test_source: str) -> list[tuple[str, str]]:
    starts = list(re.finditer(r"(?m)^func\s+(Test[A-Za-z0-9_]+)\s*\(t\s+\*testing\.T\)\s*\{", test_source))
    if not starts:
        return []

    out: list[tuple[str, str]] = []
    for m in starts:
        name = m.group(1)
        i = m.end() - 1  # points to "{"
        depth = 0
        in_string: str | None = None
        escaped = False
        in_line_comment = False
        in_block_comment = False

        j = i
        while j < len(test_source):
            ch = test_source[j]
            nxt = test_source[j + 1] if j + 1 < len(test_source) else ""

            if in_line_comment:
                if ch == "\n":
                    in_line_comment = False
                j += 1
                continue
            if in_block_comment:
                if ch == "*" and nxt == "/":
                    in_block_comment = False
                    j += 2
                    continue
                j += 1
                continue

            if in_string is not None:
                if in_string == "`":
                    if ch == "`":
                        in_string = None
                    j += 1
                    continue
                if escaped:
                    escaped = False
                    j += 1
                    continue
                if ch == "\\":
                    escaped = True
                    j += 1
                    continue
                if ch == in_string:
                    in_string = None
                j += 1
                continue

            if ch == "/" and nxt == "/":
                in_line_comment = True
                j += 2
                continue
            if ch == "/" and nxt == "*":
                in_block_comment = True
                j += 2
                continue
            if ch in ('"', "'", "`"):
                in_string = ch
                j += 1
                continue

            if ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if depth == 0:
                    body = test_source[i + 1 : j]
                    out.append((name, body))
                    break
            j += 1
    return out


def analyze_service(service: str, source_path: Path, test_path: Path | None) -> ServiceCoverage:
    source = source_path.read_text(encoding="utf-8")
    test_source = test_path.read_text(encoding="utf-8") if test_path and test_path.exists() else ""

    validation_signals = [name for name, pat in VALIDATION_PATTERNS if pat.search(source)]
    has_request_validation = len(validation_signals) > 0

    ok_response_calls = 0
    typed_ok_response_calls = 0
    generic_ok_response_calls = 0
    for args in find_function_calls(source, "respondJSON"):
        if len(args) < 3:
            continue
        status_expr = args[1]
        payload_expr = args[2]
        if "http.StatusOK" not in status_expr:
            continue
        ok_response_calls += 1
        if is_empty_success_payload(payload_expr):
            generic_ok_response_calls += 1
        else:
            typed_ok_response_calls += 1

    typed_success_fixtures = typed_ok_response_calls > 0

    negative_test_signals: list[str] = []
    negative_test_count = 0
    if test_source:
        test_functions = extract_test_functions(test_source)
        for test_name, test_body in test_functions:
            matched = [name for name, pat in NEGATIVE_TEST_PATTERNS if pat.search(test_body)]
            if not matched:
                continue
            negative_test_count += 1
            for match in matched:
                signal = f"{test_name}:{match}"
                if signal not in negative_test_signals:
                    negative_test_signals.append(signal)

    negative_contract_tests = negative_test_count > 0
    respond_not_implemented_calls = source.count("respondProviderNotImplemented(")

    return ServiceCoverage(
        service=service,
        source_file=str(source_path.relative_to(REPO_ROOT)),
        test_file=str(test_path.relative_to(REPO_ROOT)) if test_path and test_path.exists() else None,
        request_validation=has_request_validation,
        request_validation_signals=validation_signals,
        typed_success_fixtures=typed_success_fixtures,
        ok_response_calls=ok_response_calls,
        typed_ok_response_calls=typed_ok_response_calls,
        generic_ok_response_calls=generic_ok_response_calls,
        negative_contract_tests=negative_contract_tests,
        negative_test_functions=negative_test_count,
        negative_test_signals=negative_test_signals,
        respond_not_implemented_calls=respond_not_implemented_calls,
    )


def ratio(numerator: int, denominator: int) -> str:
    if denominator == 0:
        return "0/0 (0.0%)"
    return f"{numerator}/{denominator} ({(numerator / denominator) * 100:.1f}%)"


def print_text_report(services: list[ServiceCoverage], verbose: bool) -> None:
    total = len(services)
    validation_ok = sum(1 for svc in services if svc.request_validation)
    fixtures_ok = sum(1 for svc in services if svc.typed_success_fixtures)
    negative_ok = sum(1 for svc in services if svc.negative_contract_tests)
    strict_ok = sum(1 for svc in services if svc.score == 3)

    print("GCP contract coverage report")
    print(f"- services analyzed: {total}")
    print(f"- request validation: {ratio(validation_ok, total)}")
    print(f"- typed success fixtures: {ratio(fixtures_ok, total)}")
    print(f"- negative contract tests: {ratio(negative_ok, total)}")
    print(f"- strict (all 3): {ratio(strict_ok, total)}")

    missing_validation = [svc.service for svc in services if not svc.request_validation]
    missing_fixtures = [svc.service for svc in services if not svc.typed_success_fixtures]
    missing_negative = [svc.service for svc in services if not svc.negative_contract_tests]

    if missing_validation:
        print("- missing request validation: " + ", ".join(missing_validation))
    if missing_fixtures:
        print("- missing typed success fixtures: " + ", ".join(missing_fixtures))
    if missing_negative:
        print("- missing negative contract tests: " + ", ".join(missing_negative))

    if verbose:
        print("")
        print("Per-service details")
        for svc in services:
            flags = "".join(
                [
                    "V" if svc.request_validation else "-",
                    "F" if svc.typed_success_fixtures else "-",
                    "N" if svc.negative_contract_tests else "-",
                ]
            )
            print(
                f"- {svc.service}: [{flags}] ok_calls={svc.ok_response_calls} "
                f"typed_ok={svc.typed_ok_response_calls} generic_ok={svc.generic_ok_response_calls} "
                f"not_impl_calls={svc.respond_not_implemented_calls}"
            )


def parse_fail_on(raw: str) -> set[str]:
    out: set[str] = set()
    for item in raw.split(","):
        token = item.strip().lower()
        if not token:
            continue
        out.add(token)
    return out


def normalize_service_selector(raw: str) -> str:
    selector = raw.strip()
    if selector == "":
        return "*"
    # Keep fnmatch patterns unchanged.
    if any(ch in selector for ch in "*?["):
        return selector
    return SERVICE_ALIASES.get(selector, selector)


def should_fail(services: list[ServiceCoverage], fail_on: set[str]) -> bool:
    if not fail_on:
        return False

    missing_validation = any(not svc.request_validation for svc in services)
    missing_fixtures = any(not svc.typed_success_fixtures for svc in services)
    missing_negative = any(not svc.negative_contract_tests for svc in services)
    missing_any = any(svc.score < 3 for svc in services)

    checks = {
        "validation": missing_validation,
        "fixtures": missing_fixtures,
        "negative": missing_negative,
        "any": missing_any,
    }
    return any(checks.get(token, False) for token in fail_on)


def main() -> int:
    parser = argparse.ArgumentParser(description="Analyze GCP validation/fixture/negative-test coverage.")
    parser.add_argument(
        "--service",
        default="*",
        help="Filter services by fnmatch pattern (example: 'maps_*' or 'datastore*').",
    )
    parser.add_argument(
        "--require-service",
        default="",
        help="Fail if the normalized exact service is not discovered (aliases supported, e.g. rma).",
    )
    parser.add_argument(
        "--format",
        choices=("text", "json"),
        default="text",
        help="Output format.",
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Include per-service detail rows in text output.",
    )
    parser.add_argument(
        "--fail-on",
        default="",
        help="Comma-separated fail gates: validation,fixtures,negative,any",
    )
    args = parser.parse_args()

    sources = list_service_sources()
    service_selector = normalize_service_selector(args.service)
    required_service = normalize_service_selector(args.require_service) if args.require_service else ""
    matched_services: list[str] = []
    services: list[ServiceCoverage] = []
    for service, source_path in sorted(sources.items()):
        if not fnmatch.fnmatch(service, service_selector):
            continue
        matched_services.append(service)
        test_path = source_path.with_name(source_path.stem + "_test.go")
        services.append(analyze_service(service, source_path, test_path))

    if required_service and required_service not in sources:
        print(f"required service not found: {required_service}", file=sys.stderr)
        return 2
    if required_service and required_service not in matched_services:
        print(f"required service not matched by selector: {required_service}", file=sys.stderr)
        return 2

    if args.format == "json":
        payload = {
            "services": [asdict(svc) for svc in services],
            "summary": {
                "total": len(services),
                "request_validation": sum(1 for svc in services if svc.request_validation),
                "typed_success_fixtures": sum(1 for svc in services if svc.typed_success_fixtures),
                "negative_contract_tests": sum(1 for svc in services if svc.negative_contract_tests),
                "strict_all_three": sum(1 for svc in services if svc.score == 3),
            },
        }
        json.dump(payload, sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        print_text_report(services, verbose=args.verbose)

    fail_on = parse_fail_on(args.fail_on)
    if should_fail(services, fail_on):
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
