// Generated from local GCP provider coverage + examples + plan docs.
window.GCP_SERVICE_CATALOG = {
  "categoryOrder": [
    "compute",
    "storage",
    "database",
    "security",
    "networking",
    "ai",
    "integration"
  ],
  "categories": {
    "all": {
      "label": "All",
      "description": "Complete Stackyard GCP service catalog."
    },
    "compute": {
      "label": "Compute & Runtimes",
      "description": "VMs, managed runtimes, containers, and execution platforms."
    },
    "storage": {
      "label": "Storage",
      "description": "Object, file, and storage transfer capabilities."
    },
    "database": {
      "label": "Data & Databases",
      "description": "Datastores, analytics pipelines, streaming, and catalog services."
    },
    "security": {
      "label": "Security & Identity",
      "description": "IAM, policy controls, key management, and security posture services."
    },
    "networking": {
      "label": "Networking & Edge",
      "description": "Connectivity, routing, API gateways, maps, and edge access services."
    },
    "ai": {
      "label": "AI & ML",
      "description": "Vision, language, recommendations, and applied AI products."
    },
    "integration": {
      "label": "Integration & Operations",
      "description": "Events, messaging, scheduling, developer tooling, and platform operations."
    }
  },
  "summary": {
    "providerServices": 197,
    "servicesListed": 200,
    "examplesAvailable": 200,
    "plansAvailable": 70,
    "contractStrictAllThree": 196,
    "ioStrictAllFour": 196,
    "contractSummary": {
      "total": 197,
      "request_validation": 196,
      "typed_success_fixtures": 197,
      "negative_contract_tests": 196,
      "strict_all_three": 196
    },
    "ioSummary": {
      "total": 197,
      "input_validation_impl": 196,
      "output_fixture_impl": 197,
      "input_validation_tests": 196,
      "output_shape_tests": 196,
      "strict_all_four": 196
    }
  },
  "services": [
    {
      "id": "apigateway-apiv1",
      "name": "API Gateway (API v1)",
      "category": "networking",
      "summary": "Local emulation for API Gateway (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/apigateway-apiv1/docker-compose.yml",
      "canonicalService": "apigateway",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "apihub-apiv1",
      "name": "API Hub (API v1)",
      "category": "integration",
      "summary": "Local emulation for API Hub (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/apihub-apiv1/docker-compose.yml",
      "canonicalService": "apihub",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "apikeys-apiv2",
      "name": "API Keys (API v2)",
      "category": "integration",
      "summary": "Local emulation for API Keys (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/apikeys-apiv2/docker-compose.yml",
      "canonicalService": "apikeys",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "apigeeconnect-apiv1",
      "name": "Apigee Connect (API v1)",
      "category": "integration",
      "summary": "Local emulation for Apigee Connect (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/apigeeconnect-apiv1/docker-compose.yml",
      "canonicalService": "apigeeconnect",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "appengine-apiv1",
      "name": "App Engine (API v1)",
      "category": "compute",
      "summary": "Local emulation for App Engine (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/appengine-apiv1/docker-compose.yml",
      "canonicalService": "appengine",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "artifactregistry-apiv1",
      "name": "Artifact Registry (API v1)",
      "category": "integration",
      "summary": "Local emulation for Artifact Registry (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/artifactregistry-apiv1/docker-compose.yml",
      "canonicalService": "artifactregistry",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "automl-apiv1",
      "name": "AutoML (API v1)",
      "category": "ai",
      "summary": "Local emulation for AutoML (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/automl-apiv1/docker-compose.yml",
      "canonicalService": "automl",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "bigtable-apiv2",
      "name": "Bigtable (API v2)",
      "category": "database",
      "summary": "Local emulation for Bigtable (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/bigtable-apiv2/docker-compose.yml",
      "canonicalService": "bigtable",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "security-privateca-apiv1",
      "name": "Certificate Authority Service",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Certificate Authority Service (Private CA) with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-security-privateca-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/security-privateca-apiv1/docker-compose.yml",
      "canonicalService": "security_privateca",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "certificatemanager-apiv1",
      "name": "Certificate Manager (API v1)",
      "category": "security",
      "summary": "Local emulation for Certificate Manager (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/certificatemanager-apiv1/docker-compose.yml",
      "canonicalService": "certificatemanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "channel-apiv1",
      "name": "Channel (API v1)",
      "category": "integration",
      "summary": "Local emulation for Channel (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/channel-apiv1/docker-compose.yml",
      "canonicalService": "channel",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "chat-apiv1",
      "name": "Chat (API v1)",
      "category": "ai",
      "summary": "Local emulation for Chat (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/chat-apiv1/docker-compose.yml",
      "canonicalService": "chat",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "chronicle-apiv1",
      "name": "Chronicle (API v1)",
      "category": "integration",
      "summary": "Local emulation for Chronicle (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/chronicle-apiv1/docker-compose.yml",
      "canonicalService": "chronicle",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "cloudbuild-apiv2",
      "name": "Cloud Build (API v2)",
      "category": "integration",
      "summary": "Local emulation for Cloud Build (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/cloudbuild-apiv2/docker-compose.yml",
      "canonicalService": "cloudbuild",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "cloudcontrolspartner-apiv1",
      "name": "Cloud Controls Partner (API v1)",
      "category": "integration",
      "summary": "Local emulation for Cloud Controls Partner (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/cloudcontrolspartner-apiv1/docker-compose.yml",
      "canonicalService": "cloudcontrolspartner",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "clouddms-apiv1",
      "name": "Cloud DMS (API v1)",
      "category": "integration",
      "summary": "Local emulation for Cloud DMS (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/clouddms-apiv1/docker-compose.yml",
      "canonicalService": "clouddms",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "cloudprofiler-apiv2",
      "name": "Cloud Profiler (API v2)",
      "category": "integration",
      "summary": "Local emulation for Cloud Profiler (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/cloudprofiler-apiv2/docker-compose.yml",
      "canonicalService": "cloudprofiler",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "cloudquotas-apiv1",
      "name": "Cloud Quotas (API v1)",
      "category": "integration",
      "summary": "Local emulation for Cloud Quotas (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/cloudquotas-apiv1/docker-compose.yml",
      "canonicalService": "cloudquotas",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "resourcemanager-apiv2",
      "name": "Cloud Resource Manager V2",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Resource Manager V2 (Folders API surface) with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-resourcemanager-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/resourcemanager-apiv2/docker-compose.yml",
      "canonicalService": "resourcemanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "resourcemanager-apiv3",
      "name": "Cloud Resource Manager V3",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Resource Manager V3 with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-resourcemanager-apiv3-plan.md",
      "exampleHref": "../../../../examples/gcp/resourcemanager-apiv3/docker-compose.yml",
      "canonicalService": "resourcemanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "run-apiv2",
      "name": "Cloud Run Admin",
      "category": "compute",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Run Admin v2 with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-run-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/run-apiv2/docker-compose.yml",
      "canonicalService": "run",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "scheduler-apiv1",
      "name": "Cloud Scheduler",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Scheduler with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-scheduler-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/scheduler-apiv1/docker-compose.yml",
      "canonicalService": "scheduler",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shell-apiv1",
      "name": "Cloud Shell",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Shell with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shell-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shell-apiv1/docker-compose.yml",
      "canonicalService": "shell",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "spanner-apiv1",
      "name": "Cloud Spanner",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Spanner with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-spanner-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/spanner-apiv1/docker-compose.yml",
      "canonicalService": "spanner",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "spanner-adapter-apiv1",
      "name": "Cloud Spanner Adapter",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Spanner Adapter with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-spanner-adapter-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/spanner-adapter-apiv1/docker-compose.yml",
      "canonicalService": "spanner_adapter",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "spanner-admin-database-apiv1",
      "name": "Cloud Spanner Admin Database",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Spanner Admin Database with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-spanner-admin-database-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/spanner-admin-database-apiv1/docker-compose.yml",
      "canonicalService": "spanner_admin_database",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "spanner-admin-instance-apiv1",
      "name": "Cloud Spanner Admin Instance",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Spanner Admin Instance with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-spanner-admin-instance-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/spanner-admin-instance-apiv1/docker-compose.yml",
      "canonicalService": "spanner_admin_instance",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "spanner-executor-apiv1",
      "name": "Cloud Spanner Executor",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Spanner Executor with stream-aware request/response handling, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-spanner-executor-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/spanner-executor-apiv1/docker-compose.yml",
      "canonicalService": "spanner_executor",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "speech-apiv1",
      "name": "Cloud Speech-to-Text V1",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Speech-to-Text V1, including Speech RPCs, Adaptation RPCs, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-speech-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/speech-apiv1/docker-compose.yml",
      "canonicalService": "speech",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "speech-apiv2",
      "name": "Cloud Speech-to-Text V2",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Speech-to-Text V2, including Recognizer/Adaptation resource lifecycle, recognition RPCs, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-speech-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/speech-apiv2/docker-compose.yml",
      "canonicalService": "speech_v2",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "storage-apiv1",
      "name": "Cloud Storage",
      "category": "storage",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Storage (GCS) with SDK-compatible behavior, REST transport coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-storage-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/storage-apiv1/docker-compose.yml",
      "canonicalService": "storage",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "support-apiv2",
      "name": "Cloud Support V2",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Support V2 with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-support-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/support-apiv2/docker-compose.yml",
      "canonicalService": "support",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "cloudtasks-apiv2",
      "name": "Cloud Tasks (API v2)",
      "category": "integration",
      "summary": "Local emulation for Cloud Tasks (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/cloudtasks-apiv2/docker-compose.yml",
      "canonicalService": "cloudtasks",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "texttospeech-apiv1",
      "name": "Cloud Text-to-Speech V1",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Text-to-Speech V1, including TextToSpeech RPCs, Long Audio Synthesis RPCs, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-texttospeech-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/texttospeech-apiv1/docker-compose.yml",
      "canonicalService": "texttospeech",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "tpu-apiv1",
      "name": "Cloud TPU V1",
      "category": "compute",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud TPU V1, including TPU node/resource RPCs, long-running operation behavior, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-tpu-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/tpu-apiv1/docker-compose.yml",
      "canonicalService": "tpu",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "translate-apiv3",
      "name": "Cloud Translation Advanced V3",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Translation Advanced V3, including TranslationService RPCs, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-translate-apiv3-plan.md",
      "exampleHref": "../../../../examples/gcp/translate-apiv3/docker-compose.yml",
      "canonicalService": "translate",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "videointelligence-apiv1",
      "name": "Cloud Video Intelligence",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Video Intelligence (`cloud.google.com/go/videointelligence/apiv1`), including provider routing, stage-4 gRPC handling, contract tests, SDK docker examples, and coverage script integration.",
      "docsHref": "../../../gcp-videointelligence-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/videointelligence-apiv1/docker-compose.yml",
      "canonicalService": "videointelligence",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "vision-v2-apiv1",
      "name": "Cloud Vision V2",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation coverage for Cloud Vision V2 Go SDK clients (`cloud.google.com/go/vision/v2/apiv1`), including SDK example coverage and contract-coverage integration.",
      "docsHref": "../../../gcp-vision-v2-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/vision-v2-apiv1/docker-compose.yml",
      "canonicalService": "vision",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "commerce-consumer-procurement-apiv1",
      "name": "Commerce Consumer Procurement (API v1)",
      "category": "integration",
      "summary": "Local emulation for Commerce Consumer Procurement (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/commerce-consumer-procurement-apiv1/docker-compose.yml",
      "canonicalService": "commerce_consumer_procurement",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "compute-apiv1",
      "name": "Compute (API v1)",
      "category": "compute",
      "summary": "Local emulation for Compute (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/compute-apiv1/docker-compose.yml",
      "canonicalService": "compute",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "compute-metadata-apiv1",
      "name": "Compute Metadata (API v1)",
      "category": "compute",
      "summary": "Local emulation for Compute Metadata (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/compute-metadata-apiv1/docker-compose.yml",
      "canonicalService": "compute_metadata",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "confidentialcomputing-apiv1",
      "name": "Confidential Computing (API v1)",
      "category": "integration",
      "summary": "Local emulation for Confidential Computing (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/confidentialcomputing-apiv1/docker-compose.yml",
      "canonicalService": "confidentialcomputing",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "config-apiv1",
      "name": "Config (API v1)",
      "category": "integration",
      "summary": "Local emulation for Config (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/config-apiv1/docker-compose.yml",
      "canonicalService": "config",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "configdelivery-apiv1",
      "name": "Config Delivery (API v1)",
      "category": "integration",
      "summary": "Local emulation for Config Delivery (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/configdelivery-apiv1/docker-compose.yml",
      "canonicalService": "configdelivery",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "container-apiv1",
      "name": "Container (API v1)",
      "category": "compute",
      "summary": "Local emulation for Container (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/container-apiv1/docker-compose.yml",
      "canonicalService": "container",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-css-apiv1",
      "name": "CSS API",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Google Shopping CSS API v1 with REST + gRPC parity, SDK Docker examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-css-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-css-apiv1/docker-compose.yml",
      "canonicalService": "shopping_css",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datacatalog-apiv1",
      "name": "Data Catalog (API v1)",
      "category": "security",
      "summary": "Local emulation for Data Catalog (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datacatalog-apiv1/docker-compose.yml",
      "canonicalService": "datacatalog",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datacatalog-lineage-apiv1",
      "name": "Data Catalog Lineage (API v1)",
      "category": "security",
      "summary": "Local emulation for Data Catalog Lineage (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datacatalog-lineage-apiv1/docker-compose.yml",
      "canonicalService": "datacatalog_lineage",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datafusion-apiv1",
      "name": "Data Fusion (API v1)",
      "category": "database",
      "summary": "Local emulation for Data Fusion (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datafusion-apiv1/docker-compose.yml",
      "canonicalService": "datafusion",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datalabeling-apiv1",
      "name": "Data Labeling (API v1)",
      "category": "ai",
      "summary": "Local emulation for Data Labeling (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datalabeling-apiv1/docker-compose.yml",
      "canonicalService": "datalabeling",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dataqna-apiv1alpha",
      "name": "Data QnA (API v1alpha)",
      "category": "database",
      "summary": "Local emulation for Data QnA (API v1alpha) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dataqna-apiv1alpha/docker-compose.yml",
      "canonicalService": "dataqna",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dataflow-apiv1beta3",
      "name": "Dataflow (API v1beta3)",
      "category": "database",
      "summary": "Local emulation for Dataflow (API v1beta3) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dataflow-apiv1beta3/docker-compose.yml",
      "canonicalService": "dataflow",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dataform-apiv1",
      "name": "Dataform (API v1)",
      "category": "database",
      "summary": "Local emulation for Dataform (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dataform-apiv1/docker-compose.yml",
      "canonicalService": "dataform",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dataplex-apiv1",
      "name": "Dataplex (API v1)",
      "category": "database",
      "summary": "Local emulation for Dataplex (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dataplex-apiv1/docker-compose.yml",
      "canonicalService": "dataplex",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dataproc-apiv1",
      "name": "Dataproc (API v1)",
      "category": "database",
      "summary": "Local emulation for Dataproc (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dataproc-apiv1/docker-compose.yml",
      "canonicalService": "dataproc",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dataproc-v2-apiv1",
      "name": "Dataproc v2 (API v1)",
      "category": "database",
      "summary": "Local emulation for Dataproc v2 (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dataproc-v2-apiv1/docker-compose.yml",
      "canonicalService": "dataproc_v2",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datastore-apiv1",
      "name": "Datastore (API v1)",
      "category": "database",
      "summary": "Local emulation for Datastore (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datastore-apiv1/docker-compose.yml",
      "canonicalService": "datastore",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datastore-admin-apiv1",
      "name": "Datastore Admin (API v1)",
      "category": "database",
      "summary": "Local emulation for Datastore Admin (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datastore-admin-apiv1/docker-compose.yml",
      "canonicalService": "datastore_admin",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "datastream-apiv1",
      "name": "Datastream (API v1)",
      "category": "database",
      "summary": "Local emulation for Datastream (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/datastream-apiv1/docker-compose.yml",
      "canonicalService": "datastream",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "deploy-apiv1",
      "name": "Deploy (API v1)",
      "category": "integration",
      "summary": "Local emulation for Deploy (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/deploy-apiv1/docker-compose.yml",
      "canonicalService": "deploy",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "developerconnect-apiv1",
      "name": "Developer Connect (API v1)",
      "category": "integration",
      "summary": "Local emulation for Developer Connect (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/developerconnect-apiv1/docker-compose.yml",
      "canonicalService": "developerconnect",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "devicestreaming-apiv1",
      "name": "Device Streaming (API v1)",
      "category": "integration",
      "summary": "Local emulation for Device Streaming (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/devicestreaming-apiv1/docker-compose.yml",
      "canonicalService": "devicestreaming",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dialogflow-apiv1",
      "name": "Dialogflow (API v1)",
      "category": "ai",
      "summary": "Local emulation for Dialogflow (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dialogflow-apiv1/docker-compose.yml",
      "canonicalService": "dialogflow",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dialogflow-cx-apiv1",
      "name": "Dialogflow Cx (API v1)",
      "category": "ai",
      "summary": "Local emulation for Dialogflow Cx (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dialogflow-cx-apiv1/docker-compose.yml",
      "canonicalService": "dialogflow_cx",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "discoveryengine-apiv1",
      "name": "Discovery Engine (API v1)",
      "category": "integration",
      "summary": "Local emulation for Discovery Engine (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/discoveryengine-apiv1/docker-compose.yml",
      "canonicalService": "discoveryengine",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "dlp-apiv1",
      "name": "DLP (API v1)",
      "category": "integration",
      "summary": "Local emulation for DLP (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/dlp-apiv1/docker-compose.yml",
      "canonicalService": "dlp",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "documentai-apiv1",
      "name": "Document AI (API v1)",
      "category": "ai",
      "summary": "Local emulation for Document AI (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/documentai-apiv1/docker-compose.yml",
      "canonicalService": "documentai",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "domains-apiv1",
      "name": "Domains (API v1)",
      "category": "ai",
      "summary": "Local emulation for Domains (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/domains-apiv1/docker-compose.yml",
      "canonicalService": "domains",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "edgecontainer-apiv1",
      "name": "Edge Container (API v1)",
      "category": "compute",
      "summary": "Local emulation for Edge Container (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/edgecontainer-apiv1/docker-compose.yml",
      "canonicalService": "edgecontainer",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "edgenetwork-apiv1",
      "name": "Edge Network (API v1)",
      "category": "networking",
      "summary": "Local emulation for Edge Network (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/edgenetwork-apiv1/docker-compose.yml",
      "canonicalService": "edgenetwork",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "errorreporting-apiv1",
      "name": "Error Reporting (API v1)",
      "category": "integration",
      "summary": "Local emulation for Error Reporting (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/errorreporting-apiv1/docker-compose.yml",
      "canonicalService": "errorreporting",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "essentialcontacts-apiv1",
      "name": "Essential Contacts (API v1)",
      "category": "integration",
      "summary": "Local emulation for Essential Contacts (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/essentialcontacts-apiv1/docker-compose.yml",
      "canonicalService": "essentialcontacts",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "eventarc-apiv1",
      "name": "Eventarc (API v1)",
      "category": "integration",
      "summary": "Local emulation for Eventarc (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/eventarc-apiv1/docker-compose.yml",
      "canonicalService": "eventarc",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "eventarc-publishing-apiv1",
      "name": "Eventarc Publishing (API v1)",
      "category": "integration",
      "summary": "Local emulation for Eventarc Publishing (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/eventarc-publishing-apiv1/docker-compose.yml",
      "canonicalService": "eventarc_publishing",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "filestore-apiv1",
      "name": "Filestore (API v1)",
      "category": "storage",
      "summary": "Local emulation for Filestore (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/filestore-apiv1/docker-compose.yml",
      "canonicalService": "filestore",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "financialservices-apiv1",
      "name": "Financial Services (API v1)",
      "category": "integration",
      "summary": "Local emulation for Financial Services (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/financialservices-apiv1/docker-compose.yml",
      "canonicalService": "financialservices",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "firestore-apiv1",
      "name": "Firestore (API v1)",
      "category": "database",
      "summary": "Local emulation for Firestore (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/firestore-apiv1/docker-compose.yml",
      "canonicalService": "firestore",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "functions-apiv1",
      "name": "Functions (API v1)",
      "category": "compute",
      "summary": "Local emulation for Functions (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/functions-apiv1/docker-compose.yml",
      "canonicalService": "functions",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "functions-apiv2",
      "name": "Functions (API v2)",
      "category": "compute",
      "summary": "Local emulation for Functions (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/functions-apiv2/docker-compose.yml",
      "canonicalService": "functions",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "gaming-apiv1",
      "name": "Gaming (API v1)",
      "category": "integration",
      "summary": "Local emulation for Gaming (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/gaming-apiv1/docker-compose.yml",
      "canonicalService": "gaming",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "geminidataanalytics-apiv1",
      "name": "Gemini Data Analytics (API v1)",
      "category": "ai",
      "summary": "Local emulation for Gemini Data Analytics (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/geminidataanalytics-apiv1/docker-compose.yml",
      "canonicalService": "geminidataanalytics",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "generativelanguage-apiv1",
      "name": "Generative Language (API v1)",
      "category": "ai",
      "summary": "Local emulation for Generative Language (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/generativelanguage-apiv1/docker-compose.yml",
      "canonicalService": "generativelanguage",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "gkebackup-apiv1",
      "name": "GKE Backup (API v1)",
      "category": "compute",
      "summary": "Local emulation for GKE Backup (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/gkebackup-apiv1/docker-compose.yml",
      "canonicalService": "gkebackup",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "gkeconnect-apiv1",
      "name": "GKE Connect (API v1)",
      "category": "compute",
      "summary": "Local emulation for GKE Connect (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/gkeconnect-apiv1/docker-compose.yml",
      "canonicalService": "gkeconnect_gateway",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "gkehub-apiv1",
      "name": "GKE Hub (API v1)",
      "category": "compute",
      "summary": "Local emulation for GKE Hub (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/gkehub-apiv1/docker-compose.yml",
      "canonicalService": "gkehub",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "gkemulticloud-apiv1",
      "name": "GKE Multi-Cloud (API v1)",
      "category": "compute",
      "summary": "Local emulation for GKE Multi-Cloud (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/gkemulticloud-apiv1/docker-compose.yml",
      "canonicalService": "gkemulticloud",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iam-apiv1",
      "name": "IAM (API v1)",
      "category": "security",
      "summary": "Local emulation for IAM (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iam-apiv1/docker-compose.yml",
      "canonicalService": "iam",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iam-apiv2",
      "name": "IAM (API v2)",
      "category": "security",
      "summary": "Local emulation for IAM (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iam-apiv2/docker-compose.yml",
      "canonicalService": "iam",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iam-apiv3",
      "name": "IAM (API v3)",
      "category": "security",
      "summary": "Local emulation for IAM (API v3) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iam-apiv3/docker-compose.yml",
      "canonicalService": "iam",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iam-admin-apiv1",
      "name": "IAM Admin (API v1)",
      "category": "security",
      "summary": "Local emulation for IAM Admin (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iam-admin-apiv1/docker-compose.yml",
      "canonicalService": "iam_admin",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iam-credentials-apiv1",
      "name": "IAM Credentials (API v1)",
      "category": "security",
      "summary": "Local emulation for IAM Credentials (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iam-credentials-apiv1/docker-compose.yml",
      "canonicalService": "iam_credentials",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iap-apiv1",
      "name": "IAP (API v1)",
      "category": "security",
      "summary": "Local emulation for IAP (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iap-apiv1/docker-compose.yml",
      "canonicalService": "iap",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "identitytoolkit-apiv2",
      "name": "Identity Toolkit (API v2)",
      "category": "security",
      "summary": "Local emulation for Identity Toolkit (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/identitytoolkit-apiv2/docker-compose.yml",
      "canonicalService": "identitytoolkit",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "ids-apiv1",
      "name": "IDS (API v1)",
      "category": "security",
      "summary": "Local emulation for IDS (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/ids-apiv1/docker-compose.yml",
      "canonicalService": "ids",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "iot-apiv1",
      "name": "IoT (API v1)",
      "category": "integration",
      "summary": "Local emulation for IoT (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/iot-apiv1/docker-compose.yml",
      "canonicalService": "iot",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "kms-apiv1",
      "name": "KMS (API v1)",
      "category": "security",
      "summary": "Local emulation for KMS (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/kms-apiv1/docker-compose.yml",
      "canonicalService": "kms",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "kms-inventory-apiv1",
      "name": "KMS Inventory (API v1)",
      "category": "security",
      "summary": "Local emulation for KMS Inventory (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/kms-inventory-apiv1/docker-compose.yml",
      "canonicalService": "kms_inventory",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "language-apiv1",
      "name": "Language (API v1)",
      "category": "ai",
      "summary": "Local emulation for Language (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/language-apiv1/docker-compose.yml",
      "canonicalService": "language",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "language-apiv2",
      "name": "Language (API v2)",
      "category": "ai",
      "summary": "Local emulation for Language (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/language-apiv2/docker-compose.yml",
      "canonicalService": "language",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "licensemanager-apiv1",
      "name": "License Manager (API v1)",
      "category": "security",
      "summary": "Local emulation for License Manager (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/licensemanager-apiv1/docker-compose.yml",
      "canonicalService": "licensemanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "lifesciences-apiv2beta",
      "name": "Life Sciences (API v2beta)",
      "category": "integration",
      "summary": "Local emulation for Life Sciences (API v2beta) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/lifesciences-apiv2beta/docker-compose.yml",
      "canonicalService": "lifesciences",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "video-livestream-apiv1",
      "name": "Live Stream",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Live Stream (`cloud.google.com/go/video/livestream/apiv1`), including REST+gRPC routing, typed fixtures, contract tests, SDK docker examples, and coverage script integration.",
      "docsHref": "../../../gcp-video-livestream-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/video-livestream-apiv1/docker-compose.yml",
      "canonicalService": "video_livestream",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "locationfinder-apiv1",
      "name": "Location Finder (API v1)",
      "category": "security",
      "summary": "Local emulation for Location Finder (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/locationfinder-apiv1/docker-compose.yml",
      "canonicalService": "locationfinder",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "logging-apiv2",
      "name": "Logging (API v2)",
      "category": "integration",
      "summary": "Local emulation for Logging (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/logging-apiv2/docker-compose.yml",
      "canonicalService": "logging",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "lustre-apiv1",
      "name": "Lustre (API v1)",
      "category": "storage",
      "summary": "Local emulation for Lustre (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/lustre-apiv1/docker-compose.yml",
      "canonicalService": "lustre",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "managedidentities-apiv1",
      "name": "Managed Identities (API v1)",
      "category": "security",
      "summary": "Local emulation for Managed Identities (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/managedidentities-apiv1/docker-compose.yml",
      "canonicalService": "managedidentities",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "managedkafka-apiv1",
      "name": "Managed Kafka (API v1)",
      "category": "integration",
      "summary": "Local emulation for Managed Kafka (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/managedkafka-apiv1/docker-compose.yml",
      "canonicalService": "managedkafka",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "managedkafka-schemaregistry-apiv1",
      "name": "Managed Kafka Schema Registry (API v1)",
      "category": "integration",
      "summary": "Local emulation for Managed Kafka Schema Registry (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/managedkafka-schemaregistry-apiv1/docker-compose.yml",
      "canonicalService": "managedkafka_schemaregistry",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-addressvalidation-apiv1",
      "name": "Maps Address Validation (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Address Validation (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-addressvalidation-apiv1/docker-compose.yml",
      "canonicalService": "maps_addressvalidation",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-areainsights-apiv1",
      "name": "Maps Area Insights (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Area Insights (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-areainsights-apiv1/docker-compose.yml",
      "canonicalService": "maps_areainsights",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-fleetengine-apiv1",
      "name": "Maps Fleet Engine (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Fleet Engine (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-fleetengine-apiv1/docker-compose.yml",
      "canonicalService": "maps_fleetengine",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-fleetengine-delivery-apiv1",
      "name": "Maps Fleet Engine Delivery (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Fleet Engine Delivery (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-fleetengine-delivery-apiv1/docker-compose.yml",
      "canonicalService": "maps_fleetengine_delivery",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-places-apiv1",
      "name": "Maps Places (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Places (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-places-apiv1/docker-compose.yml",
      "canonicalService": "maps_places",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-routeoptimization-apiv1",
      "name": "Maps Route Optimization (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Route Optimization (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-routeoptimization-apiv1/docker-compose.yml",
      "canonicalService": "maps_routeoptimization",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-routing-apiv2",
      "name": "Maps Routing (API v2)",
      "category": "networking",
      "summary": "Local emulation for Maps Routing (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-routing-apiv2/docker-compose.yml",
      "canonicalService": "maps_routing",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "maps-solar-apiv1",
      "name": "Maps Solar (API v1)",
      "category": "networking",
      "summary": "Local emulation for Maps Solar (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/maps-solar-apiv1/docker-compose.yml",
      "canonicalService": "maps_solar",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "mediatranslation-apiv1beta1",
      "name": "Media Translation (API v1beta1)",
      "category": "ai",
      "summary": "Local emulation for Media Translation (API v1beta1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/mediatranslation-apiv1beta1/docker-compose.yml",
      "canonicalService": "mediatranslation",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "meet-apiv2",
      "name": "Meet (API v2)",
      "category": "ai",
      "summary": "Local emulation for Meet (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/meet-apiv2/docker-compose.yml",
      "canonicalService": "meet",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "memcache-apiv1",
      "name": "Memcache (API v1)",
      "category": "security",
      "summary": "Local emulation for Memcache (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/memcache-apiv1/docker-compose.yml",
      "canonicalService": "memcache",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "memorystore-apiv1",
      "name": "Memorystore (API v1)",
      "category": "database",
      "summary": "Local emulation for Memorystore (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/memorystore-apiv1/docker-compose.yml",
      "canonicalService": "memorystore",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "redis-apiv1",
      "name": "Memorystore For Redis",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Google Cloud Memorystore for Redis with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-redis-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/redis-apiv1/docker-compose.yml",
      "canonicalService": "redis",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "redis-cluster-apiv1",
      "name": "Memorystore For Redis Cluster",
      "category": "database",
      "summary": "This plan defines how to add Stackyard GCP emulation for Google Cloud Memorystore for Redis Cluster with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-redis-cluster-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/redis-cluster-apiv1/docker-compose.yml",
      "canonicalService": "redis_cluster",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-accounts-apiv1",
      "name": "Merchant Accounts",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Accounts with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-accounts-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-accounts-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_accounts",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-conversions-apiv1",
      "name": "Merchant Conversions",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Conversions with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-conversions-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-conversions-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_conversions",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-datasources-apiv1",
      "name": "Merchant Data Sources",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Data Sources with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-datasources-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-datasources-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_datasources",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-inventories-apiv1",
      "name": "Merchant Inventories",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Inventories with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-inventories-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-inventories-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_inventories",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-issueresolution-apiv1",
      "name": "Merchant Issues Resolution",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Issues Resolution with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-issueresolution-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-issueresolution-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_issueresolution",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-lfp-apiv1",
      "name": "Merchant LFP",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant LFP with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-lfp-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-lfp-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_lfp",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-notifications-apiv1",
      "name": "Merchant Notifications",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Notifications with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-notifications-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-notifications-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_notifications",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-ordertracking-apiv1",
      "name": "Merchant Order Tracking",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Order Tracking with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-ordertracking-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-ordertracking-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_ordertracking",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-productstudio-apiv1alpha",
      "name": "Merchant Product Studio",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Product Studio with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-productstudio-apiv1alpha-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-productstudio-apiv1alpha/docker-compose.yml",
      "canonicalService": "shopping_merchant_productstudio",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-products-apiv1",
      "name": "Merchant Products",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Products with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-products-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-products-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_products",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-promotions-apiv1",
      "name": "Merchant Promotions",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Promotions with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-promotions-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-promotions-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_promotions",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-quota-apiv1",
      "name": "Merchant Quota",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Quota with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-quota-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-quota-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_quota",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-reports-apiv1",
      "name": "Merchant Reports",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Reports with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-reports-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-reports-apiv1/docker-compose.yml",
      "canonicalService": "shopping_merchant_reports",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "shopping-merchant-reviews-apiv1beta",
      "name": "Merchant Reviews",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Merchant Reviews with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-shopping-merchant-reviews-apiv1beta-plan.md",
      "exampleHref": "../../../../examples/gcp/shopping-merchant-reviews-apiv1beta/docker-compose.yml",
      "canonicalService": "shopping_merchant_reviews",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "metastore-apiv1",
      "name": "Metastore (API v1)",
      "category": "database",
      "summary": "Local emulation for Metastore (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/metastore-apiv1/docker-compose.yml",
      "canonicalService": "metastore",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "metricsscope-apiv1",
      "name": "Metrics Scope (API v1)",
      "category": "database",
      "summary": "Local emulation for Metrics Scope (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/metricsscope-apiv1/docker-compose.yml",
      "canonicalService": "metricsscope",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "migrationcenter-apiv1",
      "name": "Migration Center (API v1)",
      "category": "integration",
      "summary": "Local emulation for Migration Center (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/migrationcenter-apiv1/docker-compose.yml",
      "canonicalService": "migrationcenter",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "modelarmor-apiv1",
      "name": "Model Armor (API v1)",
      "category": "ai",
      "summary": "Local emulation for Model Armor (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/modelarmor-apiv1/docker-compose.yml",
      "canonicalService": "modelarmor",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "monitoring-apiv3",
      "name": "Monitoring (API v3)",
      "category": "integration",
      "summary": "Local emulation for Monitoring (API v3) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/monitoring-apiv3/docker-compose.yml",
      "canonicalService": "monitoring",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "monitoring-dashboard-apiv1",
      "name": "Monitoring Dashboard (API v1)",
      "category": "integration",
      "summary": "Local emulation for Monitoring Dashboard (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/monitoring-dashboard-apiv1/docker-compose.yml",
      "canonicalService": "monitoring_dashboard",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "netapp-apiv1",
      "name": "NetApp (API v1)",
      "category": "storage",
      "summary": "Local emulation for NetApp (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/netapp-apiv1/docker-compose.yml",
      "canonicalService": "netapp",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "networkconnectivity-apiv1",
      "name": "Network Connectivity (API v1)",
      "category": "networking",
      "summary": "Local emulation for Network Connectivity (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/networkconnectivity-apiv1/docker-compose.yml",
      "canonicalService": "networkconnectivity",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "networkmanagement-apiv1",
      "name": "Network Management (API v1)",
      "category": "networking",
      "summary": "Local emulation for Network Management (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/networkmanagement-apiv1/docker-compose.yml",
      "canonicalService": "networkmanagement",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "networksecurity-apiv1beta1",
      "name": "Network Security (API v1beta1)",
      "category": "security",
      "summary": "Local emulation for Network Security (API v1beta1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/networksecurity-apiv1beta1/docker-compose.yml",
      "canonicalService": "networksecurity",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "networkservices-apiv1",
      "name": "Network Services (API v1)",
      "category": "networking",
      "summary": "Local emulation for Network Services (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/networkservices-apiv1/docker-compose.yml",
      "canonicalService": "networkservices",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "notebooks-apiv1",
      "name": "Notebooks (API v1)",
      "category": "compute",
      "summary": "Local emulation for Notebooks (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/notebooks-apiv1/docker-compose.yml",
      "canonicalService": "notebooks",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "notebooks-apiv2",
      "name": "Notebooks (API v2)",
      "category": "compute",
      "summary": "Local emulation for Notebooks (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/notebooks-apiv2/docker-compose.yml",
      "canonicalService": "notebooks",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "optimization-apiv1",
      "name": "Optimization (API v1)",
      "category": "ai",
      "summary": "Local emulation for Optimization (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/optimization-apiv1/docker-compose.yml",
      "canonicalService": "optimization",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "oracledatabase-apiv1",
      "name": "Oracle Database (API v1)",
      "category": "database",
      "summary": "Local emulation for Oracle Database (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/oracledatabase-apiv1/docker-compose.yml",
      "canonicalService": "oracledatabase",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "orgpolicy-apiv2",
      "name": "Org Policy (API v2)",
      "category": "security",
      "summary": "Local emulation for Org Policy (API v2) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/orgpolicy-apiv2/docker-compose.yml",
      "canonicalService": "orgpolicy",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "osconfig-apiv1",
      "name": "OS Config (API v1)",
      "category": "integration",
      "summary": "Local emulation for OS Config (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/osconfig-apiv1/docker-compose.yml",
      "canonicalService": "osconfig",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "osconfig-agentendpoint-apiv1",
      "name": "OS Config Agent Endpoint (API v1)",
      "category": "integration",
      "summary": "Local emulation for OS Config Agent Endpoint (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/osconfig-agentendpoint-apiv1/docker-compose.yml",
      "canonicalService": "osconfig_agentendpoint",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "oslogin-apiv1",
      "name": "OS Login (API v1)",
      "category": "integration",
      "summary": "Local emulation for OS Login (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/oslogin-apiv1/docker-compose.yml",
      "canonicalService": "oslogin",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "parallelstore-apiv1",
      "name": "Parallelstore (API v1)",
      "category": "storage",
      "summary": "Local emulation for Parallelstore (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/parallelstore-apiv1/docker-compose.yml",
      "canonicalService": "parallelstore",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "parametermanager-apiv1",
      "name": "Parameter Manager (API v1)",
      "category": "integration",
      "summary": "Local emulation for Parameter Manager (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/parametermanager-apiv1/docker-compose.yml",
      "canonicalService": "parametermanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "phishingprotection-apiv1beta1",
      "name": "Phishing Protection (API v1beta1)",
      "category": "security",
      "summary": "Local emulation for Phishing Protection (API v1beta1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/phishingprotection-apiv1beta1/docker-compose.yml",
      "canonicalService": "phishingprotection",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "policysimulator-apiv1",
      "name": "Policy Simulator (API v1)",
      "category": "security",
      "summary": "Local emulation for Policy Simulator (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/policysimulator-apiv1/docker-compose.yml",
      "canonicalService": "policysimulator",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "policytroubleshooter-apiv1",
      "name": "Policy Troubleshooter (API v1)",
      "category": "security",
      "summary": "Local emulation for Policy Troubleshooter (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/policytroubleshooter-apiv1/docker-compose.yml",
      "canonicalService": "policytroubleshooter",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "policytroubleshooter-iam-apiv3",
      "name": "Policy Troubleshooter IAM (API v3)",
      "category": "security",
      "summary": "Local emulation for Policy Troubleshooter IAM (API v3) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/policytroubleshooter-iam-apiv3/docker-compose.yml",
      "canonicalService": "policytroubleshooter_iam",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "privatecatalog-apiv1beta1",
      "name": "Private Catalog (API v1beta1)",
      "category": "security",
      "summary": "Local emulation for Private Catalog (API v1beta1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/privatecatalog-apiv1beta1/docker-compose.yml",
      "canonicalService": "privatecatalog",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "privilegedaccessmanager-apiv1",
      "name": "Privileged Access Manager (API v1)",
      "category": "security",
      "summary": "Local emulation for Privileged Access Manager (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/privilegedaccessmanager-apiv1/docker-compose.yml",
      "canonicalService": "privilegedaccessmanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "pubsub-apiv1",
      "name": "Pub/Sub (API v1)",
      "category": "database",
      "summary": "Local emulation for Pub/Sub (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/pubsub-apiv1/docker-compose.yml",
      "canonicalService": "pubsub",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "pubsublite-apiv1",
      "name": "Pub/Sub Lite (API v1)",
      "category": "database",
      "summary": "Local emulation for Pub/Sub Lite (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/pubsublite-apiv1/docker-compose.yml",
      "canonicalService": "pubsublite",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "pubsub-v2-apiv1",
      "name": "Pub/Sub v2 (API v1)",
      "category": "database",
      "summary": "Local emulation for Pub/Sub v2 (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/pubsub-v2-apiv1/docker-compose.yml",
      "canonicalService": "pubsub",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "security-publicca-apiv1",
      "name": "Public Certificate Authority",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Public Certificate Authority with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-security-publicca-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/security-publicca-apiv1/docker-compose.yml",
      "canonicalService": "security_publicca",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "rapidmigrationassessment-apiv1",
      "name": "Rapid Migration Assessment",
      "category": "integration",
      "summary": "This plan defines how to add a new Stackyard GCP emulation for Rapid Migration Assessment with REST + gRPC parity, SDK examples, and coverage integration.",
      "docsHref": "../../../gcp-rapidmigrationassessment-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/rapidmigrationassessment-apiv1/docker-compose.yml",
      "canonicalService": "rapidmigrationassessment",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "recaptchaenterprise-v2-apiv1",
      "name": "reCAPTCHA Enterprise",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for reCAPTCHA Enterprise with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-recaptchaenterprise-v2-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/recaptchaenterprise-v2-apiv1/docker-compose.yml",
      "canonicalService": "recaptchaenterprise",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "recommendationengine-apiv1beta1",
      "name": "Recommendations AI",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Recommendations AI with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-recommendationengine-apiv1beta1-plan.md",
      "exampleHref": "../../../../examples/gcp/recommendationengine-apiv1beta1/docker-compose.yml",
      "canonicalService": "recommendationengine",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "recommender-apiv1",
      "name": "Recommender",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Recommender with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-recommender-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/recommender-apiv1/docker-compose.yml",
      "canonicalService": "recommender",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "secretmanager-apiv1",
      "name": "Secret Manager",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Secret Manager with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-secretmanager-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/secretmanager-apiv1/docker-compose.yml",
      "canonicalService": "secretmanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "securesourcemanager-apiv1",
      "name": "Secure Source Manager",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Secure Source Manager with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-securesourcemanager-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/securesourcemanager-apiv1/docker-compose.yml",
      "canonicalService": "securesourcemanager",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "securitycenter-apiv1",
      "name": "Security Command Center",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Security Command Center (SCC) with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-securitycenter-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/securitycenter-apiv1/docker-compose.yml",
      "canonicalService": "securitycenter",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "securitycentermanagement-apiv1",
      "name": "Security Command Center Management",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Security Command Center Management with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-securitycentermanagement-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/securitycentermanagement-apiv1/docker-compose.yml",
      "canonicalService": "securitycentermanagement",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "securitycenter-apiv2",
      "name": "Security Command Center V2",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Security Command Center (SCC) V2 with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-securitycenter-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/securitycenter-apiv2/docker-compose.yml",
      "canonicalService": "securitycenter",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "securityposture-apiv1",
      "name": "Security Posture",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Security Posture with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-securityposture-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/securityposture-apiv1/docker-compose.yml",
      "canonicalService": "securityposture",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "vpcaccess-apiv1",
      "name": "Serverless VPC Access",
      "category": "security",
      "summary": "This plan defines how to add Stackyard GCP emulation for Serverless VPC Access (`cloud.google.com/go/vpcaccess/apiv1`) with REST + gRPC parity, SDK Docker examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-vpcaccess-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/vpcaccess-apiv1/docker-compose.yml",
      "canonicalService": "vpcaccess",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "servicecontrol-apiv1",
      "name": "Service Control",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Service Control with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-servicecontrol-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/servicecontrol-apiv1/docker-compose.yml",
      "canonicalService": "servicecontrol",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "servicedirectory-apiv1",
      "name": "Service Directory",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Service Directory with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-servicedirectory-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/servicedirectory-apiv1/docker-compose.yml",
      "canonicalService": "servicedirectory",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "servicehealth-apiv1",
      "name": "Service Health",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Service Health with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-servicehealth-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/servicehealth-apiv1/docker-compose.yml",
      "canonicalService": "servicehealth",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "servicemanagement-apiv1",
      "name": "Service Management",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Service Management with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-servicemanagement-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/servicemanagement-apiv1/docker-compose.yml",
      "canonicalService": "servicemanagement",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "serviceusage-apiv1",
      "name": "Service Usage",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Service Usage with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-serviceusage-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/serviceusage-apiv1/docker-compose.yml",
      "canonicalService": "serviceusage",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "trace-apiv1",
      "name": "Stackdriver Trace V1",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Stackdriver Trace V1, including trace query/write RPCs, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-trace-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/trace-apiv1/docker-compose.yml",
      "canonicalService": "trace_v1",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "trace-apiv2",
      "name": "Stackdriver Trace V2",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Stackdriver Trace V2, including span-ingest RPCs, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-trace-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/trace-apiv2/docker-compose.yml",
      "canonicalService": "trace",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "storagebatchoperations-apiv1",
      "name": "Storage Batch Operations",
      "category": "storage",
      "summary": "This plan defines how to add Stackyard GCP emulation for Storage Batch Operations with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-storagebatchoperations-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/storagebatchoperations-apiv1/docker-compose.yml",
      "canonicalService": "storagebatchoperations",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "storageinsights-apiv1",
      "name": "Storage Insights",
      "category": "storage",
      "summary": "This plan defines how to add Stackyard GCP emulation for Storage Insights with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-storageinsights-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/storageinsights-apiv1/docker-compose.yml",
      "canonicalService": "storageinsights",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "storagetransfer-apiv1",
      "name": "Storage Transfer",
      "category": "storage",
      "summary": "This plan defines how to add Stackyard GCP emulation for Storage Transfer with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-storagetransfer-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/storagetransfer-apiv1/docker-compose.yml",
      "canonicalService": "storagetransfer",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "streetview-publish-apiv1",
      "name": "Street View Publish",
      "category": "networking",
      "summary": "This plan defines how to add Stackyard GCP emulation for Street View Publish with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-streetview-publish-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/streetview-publish-apiv1/docker-compose.yml",
      "canonicalService": "streetview_publish",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "subscriptions-apiv1",
      "name": "Subscriptions (API v1)",
      "category": "integration",
      "summary": "Local emulation for Subscriptions (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/subscriptions-apiv1/docker-compose.yml",
      "canonicalService": "workspaceevents_subscriptions",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "talent-apiv4",
      "name": "Talent Solution V4",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Cloud Talent Solution V4 with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-talent-apiv4-plan.md",
      "exampleHref": "../../../../examples/gcp/talent-apiv4/docker-compose.yml",
      "canonicalService": "talent",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "telcoautomation-apiv1",
      "name": "Telco Automation V1",
      "category": "integration",
      "summary": "This plan defines how to add Stackyard GCP emulation for Telco Automation V1 with REST + gRPC parity, SDK example coverage, and contract-coverage integration.",
      "docsHref": "../../../gcp-telcoautomation-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/telcoautomation-apiv1/docker-compose.yml",
      "canonicalService": "telcoautomation",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "video-transcoder-apiv1",
      "name": "Transcoder",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Transcoder (`cloud.google.com/go/video/transcoder/apiv1`), including provider routing, stage-4 gRPC handling, contract tests, SDK docker examples, and coverage script integration.",
      "docsHref": "../../../gcp-video-transcoder-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/video-transcoder-apiv1/docker-compose.yml",
      "canonicalService": "video_transcoder",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "aiplatform-apiv1",
      "name": "Vertex AI Platform (API v1)",
      "category": "ai",
      "summary": "Local emulation for Vertex AI Platform (API v1) with full contract gates for integration and SDK workflow testing.",
      "docsHref": null,
      "exampleHref": "../../../../examples/gcp/aiplatform-apiv1/docker-compose.yml",
      "canonicalService": "aiplatform",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "retail-apiv2",
      "name": "Vertex AI Search for commerce",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Vertex AI Search for commerce with REST + gRPC parity, SDK examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-retail-apiv2-plan.md",
      "exampleHref": "../../../../examples/gcp/retail-apiv2/docker-compose.yml",
      "canonicalService": "retail",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "video-stitcher-apiv1",
      "name": "Video Stitcher",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Video Stitcher (`cloud.google.com/go/video/stitcher/apiv1`), including provider routing, stage-4 gRPC handling, contract tests, SDK docker examples, and coverage script integration.",
      "docsHref": "../../../gcp-video-stitcher-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/video-stitcher-apiv1/docker-compose.yml",
      "canonicalService": "video_stitcher",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "vision-apiv1",
      "name": "Vision",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Vision (`cloud.google.com/go/vision/apiv1`), including provider routing, stage-4 gRPC handling, contract tests, SDK docker examples, and coverage script integration.",
      "docsHref": "../../../gcp-vision-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/vision-apiv1/docker-compose.yml",
      "canonicalService": "vision",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "visionai-apiv1",
      "name": "Vision AI",
      "category": "ai",
      "summary": "This plan defines how to add Stackyard GCP emulation for Vision AI (`cloud.google.com/go/visionai/apiv1`), including provider routing, stage-4 gRPC parity, contract tests, SDK docker example coverage, and coverage-script integration.",
      "docsHref": "../../../gcp-visionai-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/visionai-apiv1/docker-compose.yml",
      "canonicalService": "visionai",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "vmmigration-apiv1",
      "name": "VM Migration",
      "category": "compute",
      "summary": "This plan defines how to add Stackyard GCP emulation for VM Migration (`cloud.google.com/go/vmmigration/apiv1`) with REST + gRPC parity, SDK Docker examples, and coverage-script integration.",
      "docsHref": "../../../gcp-vmmigration-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/vmmigration-apiv1/docker-compose.yml",
      "canonicalService": "vmmigration",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    },
    {
      "id": "vmwareengine-apiv1",
      "name": "VMware Engine",
      "category": "compute",
      "summary": "This plan defines how to add Stackyard GCP emulation for VMware Engine (`cloud.google.com/go/vmwareengine/apiv1`) with REST + gRPC parity, SDK Docker examples, and contract-coverage integration.",
      "docsHref": "../../../gcp-vmwareengine-apiv1-plan.md",
      "exampleHref": "../../../../examples/gcp/vmwareengine-apiv1/docker-compose.yml",
      "canonicalService": "vmwareengine",
      "capabilities": {
        "contractScore": 3,
        "ioScore": 4,
        "requestValidation": true,
        "typedFixtures": true,
        "negativeTests": true,
        "ioValidationImpl": true,
        "ioValidationTests": true,
        "ioShapeTests": true
      }
    }
  ]
};
