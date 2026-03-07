#!/usr/bin/env python3
"""
GCP input/output contract coverage analyzer for Stackyard.

This script checks four dimensions per GCP provider service:
1) input validation implementation signals in provider source,
2) output fixture implementation signals in provider source,
3) input validation test signals in provider tests,
4) output shape assertion test signals in provider tests.

The analyzer is static and heuristic-based. It is designed to be stricter than
scripts/gcp-contract-coverage.py for IO contract checks while staying fast.
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


INPUT_STATUS_PATTERN = re.compile(
    r"http\.Status(BadRequest|UnprocessableEntity|Conflict|Forbidden|NotFound|Unauthorized)"
)
INPUT_ERROR_TOKEN_PATTERN = re.compile(
    r'"(InvalidArgument|FailedPrecondition|OutOfRange|PermissionDenied|MetadataFlavorRequired|NotFound|Unauthorized)"'
)
INPUT_ERROR_FIELD_PATTERN = re.compile(r'"error"\s*:')
OUTPUT_OK_WRITE_PATTERN = re.compile(r"w\.WriteHeader\(\s*http\.StatusOK\s*\)")
OUTPUT_BODY_WRITE_PATTERN = re.compile(r"w\.Write\(")
ASSERT_SUCCESS_CALL_PATTERN = re.compile(r"\bassert[A-Za-z0-9_]*Success\(")
TYPE_ASSERT_PATTERN = re.compile(r"\.\([A-Za-z0-9_\[\]\*\.]+\)")
NEGATIVE_TEST_NAME_PATTERN = re.compile(
    r"(Invalid|Requires|Missing|Malformed|OutOfRange|BadRequest|Forbidden|Denied|Reject|Error|MustMatch|Required)"
)

EMPTY_SUCCESS_PAYLOAD_PATTERNS = (
    re.compile(r"^map\[string\]any\{\}$"),
    re.compile(r"^map\[string\]interface\{\}\{\}$"),
    re.compile(r"^\[\]any\{\}$"),
    re.compile(r"^\[\]interface\{\}\{\}$"),
    re.compile(r"^nil$"),
)


@dataclass
class ServiceIOCoverage:
    service: str
    source_file: str
    test_file: str | None
    input_validation_impl: bool
    output_fixture_impl: bool
    input_validation_tests: bool
    output_shape_tests: bool
    input_validation_impl_signals: list[str]
    output_fixture_impl_signals: list[str]
    input_validation_test_signals: list[str]
    output_shape_test_signals: list[str]
    ok_response_calls: int
    typed_ok_response_calls: int
    generic_ok_response_calls: int

    @property
    def strict(self) -> bool:
        return (
            self.input_validation_impl
            and self.output_fixture_impl
            and self.input_validation_tests
            and self.output_shape_tests
        )


def strip_ws(text: str) -> str:
    return re.sub(r"\s+", "", text)


def is_empty_success_payload(payload_expr: str) -> bool:
    compact = strip_ws(payload_expr)
    return any(pat.fullmatch(compact) for pat in EMPTY_SUCCESS_PAYLOAD_PATTERNS)


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
        i = m.end() - 1
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
                    out.append((name, test_source[i + 1 : j]))
                    break
            j += 1
    return out


def analyze_service(service: str, source_path: Path, test_path: Path | None) -> ServiceIOCoverage:
    source = source_path.read_text(encoding="utf-8")
    test_source = test_path.read_text(encoding="utf-8") if test_path and test_path.exists() else ""

    impl_input_signals: list[str] = []
    if INPUT_STATUS_PATTERN.search(source):
        impl_input_signals.append("has_4xx_status")
    if INPUT_ERROR_FIELD_PATTERN.search(source):
        impl_input_signals.append("has_error_field")
    if INPUT_ERROR_TOKEN_PATTERN.search(source):
        impl_input_signals.append("has_contract_error_token")
    input_validation_impl = "has_4xx_status" in impl_input_signals and (
        "has_error_field" in impl_input_signals or "has_contract_error_token" in impl_input_signals
    )

    ok_response_calls = 0
    typed_ok_response_calls = 0
    generic_ok_response_calls = 0
    for args in find_function_calls(source, "respondJSON"):
        if len(args) < 3:
            continue
        if "http.StatusOK" not in args[1]:
            continue
        ok_response_calls += 1
        if is_empty_success_payload(args[2]):
            generic_ok_response_calls += 1
        else:
            typed_ok_response_calls += 1

    impl_output_signals: list[str] = []
    if typed_ok_response_calls > 0:
        impl_output_signals.append("typed_json_success")
    if OUTPUT_OK_WRITE_PATTERN.search(source) and OUTPUT_BODY_WRITE_PATTERN.search(source):
        impl_output_signals.append("raw_success_body_write")
    output_fixture_impl = len(impl_output_signals) > 0

    input_test_signals: list[str] = []
    output_test_signals: list[str] = []
    if test_source:
        for test_name, body in extract_test_functions(test_source):
            has_4xx_assert = bool(INPUT_STATUS_PATTERN.search(body))
            has_error_assert = bool(INPUT_ERROR_TOKEN_PATTERN.search(body) or INPUT_ERROR_FIELD_PATTERN.search(body))
            is_negative_named = bool(NEGATIVE_TEST_NAME_PATTERN.search(test_name))

            if has_4xx_assert and (has_error_assert or is_negative_named):
                input_test_signals.append(test_name)

            has_success_helper_call = bool(ASSERT_SUCCESS_CALL_PATTERN.search(body))
            has_ok_assert = "http.StatusOK" in body
            has_output_shape_assert = (
                "providerContractJSONMap" in body
                or bool(TYPE_ASSERT_PATTERN.search(body))
                or ("providerContractBody" in body and "strings.Contains(" in body)
            )
            if has_success_helper_call or (has_ok_assert and has_output_shape_assert):
                output_test_signals.append(test_name)

    input_validation_tests = len(input_test_signals) > 0
    output_shape_tests = len(output_test_signals) > 0

    return ServiceIOCoverage(
        service=service,
        source_file=str(source_path.relative_to(REPO_ROOT)),
        test_file=str(test_path.relative_to(REPO_ROOT)) if test_path and test_path.exists() else None,
        input_validation_impl=input_validation_impl,
        output_fixture_impl=output_fixture_impl,
        input_validation_tests=input_validation_tests,
        output_shape_tests=output_shape_tests,
        input_validation_impl_signals=impl_input_signals,
        output_fixture_impl_signals=impl_output_signals,
        input_validation_test_signals=input_test_signals,
        output_shape_test_signals=output_test_signals,
        ok_response_calls=ok_response_calls,
        typed_ok_response_calls=typed_ok_response_calls,
        generic_ok_response_calls=generic_ok_response_calls,
    )


def ratio(numerator: int, denominator: int) -> str:
    if denominator == 0:
        return "0/0 (0.0%)"
    return f"{numerator}/{denominator} ({(numerator / denominator) * 100:.1f}%)"


def parse_fail_on(raw: str) -> set[str]:
    out: set[str] = set()
    for item in raw.split(","):
        token = item.strip().lower()
        if token:
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


def should_fail(services: list[ServiceIOCoverage], fail_on: set[str]) -> bool:
    if not fail_on:
        return False
    checks = {
        "input_impl": any(not s.input_validation_impl for s in services),
        "output_impl": any(not s.output_fixture_impl for s in services),
        "input_tests": any(not s.input_validation_tests for s in services),
        "output_tests": any(not s.output_shape_tests for s in services),
        "strict": any(not s.strict for s in services),
        "any": any(not s.strict for s in services),
    }
    return any(checks.get(token, False) for token in fail_on)


def print_text_report(services: list[ServiceIOCoverage], verbose: bool) -> None:
    total = len(services)
    input_impl_ok = sum(1 for s in services if s.input_validation_impl)
    output_impl_ok = sum(1 for s in services if s.output_fixture_impl)
    input_tests_ok = sum(1 for s in services if s.input_validation_tests)
    output_tests_ok = sum(1 for s in services if s.output_shape_tests)
    strict_ok = sum(1 for s in services if s.strict)

    print("GCP IO contract coverage report")
    print(f"- services analyzed: {total}")
    print(f"- input validation implementation: {ratio(input_impl_ok, total)}")
    print(f"- output fixture implementation: {ratio(output_impl_ok, total)}")
    print(f"- input validation tests: {ratio(input_tests_ok, total)}")
    print(f"- output shape tests: {ratio(output_tests_ok, total)}")
    print(f"- strict (all 4): {ratio(strict_ok, total)}")

    missing_input_impl = [s.service for s in services if not s.input_validation_impl]
    missing_output_impl = [s.service for s in services if not s.output_fixture_impl]
    missing_input_tests = [s.service for s in services if not s.input_validation_tests]
    missing_output_tests = [s.service for s in services if not s.output_shape_tests]

    if missing_input_impl:
        print("- missing input validation implementation: " + ", ".join(missing_input_impl))
    if missing_output_impl:
        print("- missing output fixture implementation: " + ", ".join(missing_output_impl))
    if missing_input_tests:
        print("- missing input validation tests: " + ", ".join(missing_input_tests))
    if missing_output_tests:
        print("- missing output shape tests: " + ", ".join(missing_output_tests))

    if verbose:
        print("")
        print("Per-service details")
        for s in services:
            flags = "".join(
                [
                    "I" if s.input_validation_impl else "-",
                    "O" if s.output_fixture_impl else "-",
                    "i" if s.input_validation_tests else "-",
                    "o" if s.output_shape_tests else "-",
                ]
            )
            print(
                f"- {s.service}: [{flags}] "
                f"ok_calls={s.ok_response_calls} typed_ok={s.typed_ok_response_calls} generic_ok={s.generic_ok_response_calls}"
            )


def main() -> int:
    parser = argparse.ArgumentParser(description="Analyze GCP input/output contract coverage.")
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
    parser.add_argument("--format", choices=("text", "json"), default="text", help="Output format.")
    parser.add_argument("--verbose", action="store_true", help="Include per-service detail in text output.")
    parser.add_argument(
        "--fail-on",
        default="",
        help="Comma-separated fail gates: input_impl,output_impl,input_tests,output_tests,strict,any",
    )
    args = parser.parse_args()

    service_selector = normalize_service_selector(args.service)
    required_service = normalize_service_selector(args.require_service) if args.require_service else ""
    matched_services: list[str] = []
    sources = list_service_sources()

    services: list[ServiceIOCoverage] = []
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
            "services": [asdict(s) for s in services],
            "summary": {
                "total": len(services),
                "input_validation_impl": sum(1 for s in services if s.input_validation_impl),
                "output_fixture_impl": sum(1 for s in services if s.output_fixture_impl),
                "input_validation_tests": sum(1 for s in services if s.input_validation_tests),
                "output_shape_tests": sum(1 for s in services if s.output_shape_tests),
                "strict_all_four": sum(1 for s in services if s.strict),
            },
        }
        json.dump(payload, sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        print_text_report(services, verbose=args.verbose)

    if should_fail(services, parse_fail_on(args.fail_on)):
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
