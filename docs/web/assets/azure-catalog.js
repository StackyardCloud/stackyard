window.AZURE_SERVICE_CATALOG = {
  categoryOrder: ["storage", "integration", "security"],
  categories: {
    all: {
      label: "All",
      description: "Complete Stackyard Azure service catalog."
    },
    storage: {
      label: "Storage",
      description: "Blob and queue-oriented storage services."
    },
    integration: {
      label: "Integration",
      description: "Messaging, bot conversation, and service-to-service integration capabilities."
    },
    security: {
      label: "Security",
      description: "Secrets, key management, and AI Services data-plane moderation services."
    }
  },
  summary: {
    providerServices: 10,
    servicesListed: 10,
    examplesAvailable: 10,
    plansAvailable: 8,
    contractStrictAllThree: 8,
    ioStrictAllFour: 8
  },
  services: [
    {
      id: "blob-storage",
      name: "Blob Storage",
      category: "storage",
      summary: "Container/blob lifecycle with metadata, conditional reads, and paginated listing support.",
      docsHref: "../../../azure-provider-foundation-plan.md",
      exampleHref: "../../../../examples/azure/blob-storage/docker-compose.yml",
      canonicalService: "blob",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "queue-storage",
      name: "Queue Storage",
      category: "integration",
      summary: "Queue lifecycle and message enqueue/dequeue/delete flows for local workflow testing.",
      docsHref: "../../../azure-provider-foundation-plan.md",
      exampleHref: "../../../../examples/azure/queue-storage/docker-compose.yml",
      canonicalService: "queue",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "azure-bot-service-4.0",
      name: "Azure Bot Framework",
      category: "integration",
      summary: "Connector API conversation/activity/member workflows for local bot integration testing.",
      docsHref: "../../../azure-bot-service-4.0-plan.md",
      exampleHref: "../../../../examples/azure/ai-bot-service/bot-service-4.0/docker-compose.yml",
      canonicalService: "botframework",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "bot-framework-bot-connector-v3.1",
      name: "Azure AI Bot Service - Bot Connector",
      category: "integration",
      summary: "Bot Connector v3.1 conversation/activity/member workflows aligned to botframework-channel Swagger.",
      docsHref: "../../../bot-framework-bot-connector-v3.1-plan.md",
      exampleHref: "../../../../examples/azure/ai-bot-service/bot-framework-bot-connector-v3.1/docker-compose.yml",
      canonicalService: "botframework",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "bot-framework-direct-line-v3.0",
      name: "Azure AI Bot Service - Direct Line (v3.0)",
      category: "integration",
      summary: "Direct Line v3.0 token, conversation, activity, and reconnect workflows for bot clients.",
      docsHref: "../../../bot-framework-direct-line-v3.0-plan.md",
      exampleHref: "../../../../examples/azure/ai-bot-service/bot-framework-direct-line-v3.0/docker-compose.yml",
      canonicalService: "botframework",
      capabilities: {
        contractScore: 0,
        ioScore: 0,
        requestValidation: false,
        typedFixtures: false,
        negativeTests: false,
        ioValidationImpl: false,
        ioValidationTests: false,
        ioShapeTests: false
      }
    },
    {
      id: "bot-framework-direct-line-v1.1",
      name: "Azure AI Bot Service - Direct Line (v1.1)",
      category: "integration",
      summary: "Direct Line v1.1 token, conversation, and message workflows aligned to directline-1.1 Swagger.",
      docsHref: "../../../bot-framework-direct-line-v1.1-plan.md",
      exampleHref: "../../../../examples/azure/ai-bot-service/bot-framework-direct-line-v1.1/docker-compose.yml",
      canonicalService: "botframework",
      capabilities: {
        contractScore: 0,
        ioScore: 0,
        requestValidation: false,
        typedFixtures: false,
        negativeTests: false,
        ioValidationImpl: false,
        ioValidationTests: false,
        ioShapeTests: false
      }
    },
    {
      id: "keyvault-secrets",
      name: "Key Vault Secrets",
      category: "security",
      summary: "Secret set/get/list-version workflows with deterministic local versioning semantics.",
      docsHref: "../../../azure-provider-foundation-plan.md",
      exampleHref: "../../../../examples/azure/keyvault-secrets/docker-compose.yml",
      canonicalService: "keyvault",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "data-plane-image-moderation",
      name: "AI Services Data Plane - Image Moderation",
      category: "security",
      summary: "Evaluate/find-faces/match/OCR image moderation workflows for URL and stream overloads.",
      docsHref: "../../../ai-services-data-plane-image-moderation-v1.0-plan.md",
      exampleHref: "../../../../examples/azure/ai-services/data-plane-image-moderation-v1.0/docker-compose.yml",
      canonicalService: "data_plane_image_moderation",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "data-plane-text-moderation",
      name: "AI Services Data Plane - Text Moderation",
      category: "security",
      summary: "DetectLanguage and Screen text moderation workflows with optional PII and classification signals.",
      docsHref: "../../../ai-services-data-plane-text-moderation-v1.0-plan.md",
      exampleHref: "../../../../examples/azure/ai-services/data-plane-text-moderation-v1.0/docker-compose.yml",
      canonicalService: "data_plane_text_moderation",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    },
    {
      id: "data-plane-list-management",
      name: "AI Services Data Plane - List Management",
      category: "security",
      summary: "Image list create/list/get/update/refresh/delete workflows for local moderation list orchestration.",
      docsHref: "../../../ai-services-data-plane-list-management-v1.0-plan.md",
      exampleHref: "../../../../examples/azure/ai-services/data-plane-list-management-v1.0/docker-compose.yml",
      canonicalService: "data_plane_list_management",
      capabilities: {
        contractScore: 3,
        ioScore: 4,
        requestValidation: true,
        typedFixtures: true,
        negativeTests: true,
        ioValidationImpl: true,
        ioValidationTests: true,
        ioShapeTests: true
      }
    }
  ]
};
