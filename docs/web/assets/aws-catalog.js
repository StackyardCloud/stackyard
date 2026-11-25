// Extracted from app.js for cross-page data consumption.
window.AWS_SERVICE_CATALOG = [
  {
    id: "acm",
    name: "ACM",
    category: "security",
    summary: "Issue, import, and inspect certificates with AWS-compatible ACM APIs.",
    docsHref: "../../../index.html#acm",
    exampleHref: "../../../../examples/aws/acm/docker-compose.yml"
  },
  {
    id: "apigateway",
    name: "API Gateway",
    category: "integration",
    summary: "Plan staged emulation for REST APIs, resources, methods, integrations, deployments, stages, usage plans, domain names, and tagging APIs.",
    docsHref: "../../../index.html#apigateway",
    exampleHref: "../../../../examples/aws/apigateway/docker-compose.yml"
  },
  {
    id: "apigatewayv2",
    name: "API Gateway v2",
    category: "integration",
    summary: "Plan staged emulation for HTTP/WebSocket APIs, routes, integrations, stages, mappings, portals, and tagging APIs.",
    docsHref: "../../../index.html#apigatewayv2",
    exampleHref: "../../../../examples/aws/apigatewayv2/docker-compose.yml"
  },
  {
    id: "apigatewaymanagementapi",
    name: "API Gateway Management API",
    category: "integration",
    summary: "Push messages to, inspect, and disconnect WebSocket API client connections through the @connections callback API.",
    docsHref: "../../../index.html#apigatewaymanagementapi",
    exampleHref: "../../../../examples/aws/apigatewaymanagementapi/docker-compose.yml"
  },
  {
    id: "appconfig",
    name: "AppConfig",
    category: "integration",
    summary: "Plan and validate application, environment, deployment, extension, and hosted-config control plane APIs.",
    docsHref: "../../../index.html#appconfig",
    exampleHref: "../../../../examples/aws/appconfig/docker-compose.yml"
  },
  {
    id: "appconfigdata",
    name: "AppConfig Data",
    category: "integration",
    summary: "Start configuration sessions and poll the latest deployed AppConfig configuration with single-use tokens.",
    docsHref: "../../../index.html#appconfigdata",
    exampleHref: "../../../../examples/aws/appconfig-data/docker-compose.yml"
  },
  {
    id: "athena",
    name: "Athena",
    category: "database",
    summary: "Query S3-backed datasets with familiar Athena request flows.",
    docsHref: null,
    exampleHref: "../../../../examples/aws/athena/docker-compose.yml"
  },
  {
    id: "glue",
    name: "AWS Glue",
    category: "database",
    summary: "Emulate the Glue Data Catalog (databases, tables with versions, partitions, statistics), crawlers, jobs and runs, triggers, workflows, Schema Registry, sessions and Data Quality.",
    docsHref: "../../../index.html#glue",
    exampleHref: "../../../../examples/aws/glue/docker-compose.yml"
  },
  {
    id: "aurora-dsql",
    name: "Aurora DSQL",
    category: "database",
    summary: "Use Aurora DSQL endpoints for distributed SQL integration testing.",
    docsHref: "../../../index.html#aurora-dsql",
    exampleHref: "../../../../examples/aws/aurora-dsql/docker-compose.yml"
  },
  {
    id: "rds-data",
    name: "RDS Data API",
    category: "database",
    summary: "Run SQL over HTTP against Aurora clusters with the RDS Data API.",
    docsHref: "../../../index.html#rds-data",
    exampleHref: "../../../../examples/aws/rds-data/docker-compose.yml"
  },
  {
    id: "augmented-ai",
    name: "Amazon Augmented AI",
    category: "integration",
    summary: "Plan and validate human loop runtime workflows for start, stop, describe, list, and delete APIs.",
    docsHref: "../../../index.html#augmented-ai",
    exampleHref: "../../../../examples/aws/augmented-ai/docker-compose.yml"
  },
  {
    id: "sagemaker-runtime",
    name: "Amazon SageMaker Runtime",
    category: "integration",
    summary: "Invoke SageMaker endpoints in real time, asynchronously through S3, and with event-stream responses.",
    docsHref: "../../../index.html#sagemaker-runtime",
    exampleHref: "../../../../examples/aws/sagemaker-runtime/docker-compose.yml"
  },
  {
    id: "sagemaker-featurestore-runtime",
    name: "Amazon SageMaker Feature Store Runtime",
    category: "integration",
    summary: "Put, update, get, list, delete and batch-read/write records in the online store of feature groups created with SageMaker, checked against the group's feature definitions, event times and TTLs.",
    docsHref: "../../../index.html#sagemaker-featurestore-runtime",
    exampleHref: "../../../../examples/aws/sagemaker-featurestore-runtime/docker-compose.yml"
  },
  {
    id: "sagemaker-metrics",
    name: "Amazon SageMaker Metrics",
    category: "integration",
    summary: "Log training metrics to SageMaker trial components (including a training job's) and query them by iteration or time window.",
    docsHref: "../../../index.html#sagemaker-metrics",
    exampleHref: "../../../../examples/aws/sagemaker-metrics/docker-compose.yml"
  },
  {
    id: "sagemaker-edge",
    name: "Amazon SageMaker Edge Manager",
    category: "integration",
    summary: "Edge agent API for devices registered with SageMaker: registration checks, deployments from in-progress edge deployment stages, and heartbeats that record agent versions, models and deployment results.",
    docsHref: "../../../index.html#sagemaker-edge",
    exampleHref: "../../../../examples/aws/sagemaker-edge/docker-compose.yml"
  },
  {
    id: "batch",
    name: "Batch",
    category: "compute",
    summary: "Model job queues, job definitions, and execution lifecycles.",
    docsHref: "../../../index.html#batch",
    exampleHref: "../../../../examples/aws/batch/docker-compose.yml"
  },
  {
    id: "bedrock",
    name: "Bedrock",
    category: "integration",
    summary: "Emulate foundation model catalog, guardrails, jobs, and provisioning control-plane APIs.",
    docsHref: "../../../index.html#bedrock",
    exampleHref: "../../../../examples/aws/bedrock/docker-compose.yml"
  },
  {
    id: "bedrockruntime",
    name: "Bedrock Runtime",
    category: "integration",
    summary: "Emulate model invocation, Converse, streaming, token counting, guardrails, and async invoke with deterministic responses.",
    docsHref: "../../../index.html#bedrockruntime",
    exampleHref: "../../../../examples/aws/bedrock-runtime/docker-compose.yml"
  },
  {
    id: "bedrockagent",
    name: "Agents for Amazon Bedrock",
    category: "integration",
    summary: "Emulate Bedrock agents, aliases and versions, knowledge bases with data sources and ingestion, flows, prompts and retrieval instructions.",
    docsHref: "../../../index.html#bedrockagent",
    exampleHref: "../../../../examples/aws/bedrock-agent/docker-compose.yml"
  },
  {
    id: "bedrockagentruntime",
    name: "Agents for Amazon Bedrock Runtime",
    category: "integration",
    summary: "Emulate InvokeAgent, inline agents, knowledge-base Retrieve and RetrieveAndGenerate, InvokeFlow and flow executions, Bedrock sessions, rerank and agent memory.",
    docsHref: "../../../index.html#bedrockagentruntime",
    exampleHref: "../../../../examples/aws/bedrock-agent-runtime/docker-compose.yml"
  },
  {
    id: "cloudcontrolapi",
    name: "Cloud Control API",
    category: "integration",
    summary: "Emulate Cloud Control resource CRUD, progress tracking, and request status flows.",
    docsHref: "../../../index.html#cloudcontrolapi",
    exampleHref: "../../../../examples/aws/cloudcontrolapi/docker-compose.yml"
  },
  {
    id: "controlcatalog",
    name: "Control Catalog",
    category: "integration",
    summary: "Plan staged coverage for control catalog, domains, objectives, and mapping-discovery APIs.",
    docsHref: "../../../index.html#controlcatalog",
    exampleHref: "../../../../examples/aws/controlcatalog/docker-compose.yml"
  },
  {
    id: "controltower",
    name: "Control Tower",
    category: "integration",
    summary: "Plan staged coverage for landing zone, baseline, control, operation-history, and tagging APIs.",
    docsHref: "../../../index.html#controltower",
    exampleHref: "../../../../examples/aws/controltower/docker-compose.yml"
  },
  {
    id: "cloudformation",
    name: "CloudFormation",
    category: "integration",
    summary: "Plan and validate stack lifecycle, change sets, stack sets, registry types, and drift workflows.",
    docsHref: "../../../index.html#cloudformation",
    exampleHref: "../../../../examples/aws/cloudformation/docker-compose.yml"
  },
  {
    id: "cloud-map",
    name: "Cloud Map",
    category: "integration",
    summary: "Plan staged coverage for namespace/service lifecycle, instance registration/discovery, operations tracking, and tagging APIs.",
    docsHref: "../../../index.html#cloud-map",
    exampleHref: "../../../../examples/aws/cloud-map/docker-compose.yml"
  },
  {
    id: "route53domains",
    name: "Route 53 Domains",
    category: "integration",
    summary: "Emulate domain registration, renewal, transfers, contacts and privacy, name servers, DNSSEC keys, operations tracking, billing and tags.",
    docsHref: "../../../index.html#route53domains",
    exampleHref: "../../../../examples/aws/route53domains/docker-compose.yml"
  },
  {
    id: "textract",
    name: "Textract",
    category: "integration",
    summary: "Emulate text detection, forms, tables, queries, layout, signatures, expense, identity and lending analysis over text and PDF documents, with async jobs, adapters and tags.",
    docsHref: "../../../index.html#textract",
    exampleHref: "../../../../examples/aws/textract/docker-compose.yml"
  },  {
    id: "forecast",
    name: "Forecast",
    category: "integration",
    summary: "Emulate Forecast datasets, import jobs, dataset groups, predictors and auto predictors with accuracy metrics, forecasts, exports, explainability, monitors with evaluations, what-if analyses and forecasts, resource trees and tags.",
    docsHref: "../../../index.html#forecast",
    exampleHref: "../../../../examples/aws/forecast/docker-compose.yml"
  },

  {
    id: "forecastquery",
    name: "Forecast Query",
    category: "integration",
    summary: "Query predictions from Forecast forecasts and what-if forecasts: one series per forecast type over the predictor's horizon and frequency, filtered by target time series attributes and a date window.",
    docsHref: "../../../index.html#forecastquery",
    exampleHref: "../../../../examples/aws/forecastquery/docker-compose.yml"
  },

  {
    id: "frauddetector",
    name: "Fraud Detector",
    category: "integration",
    summary: "Emulate Fraud Detector variables, entity types, labels, outcomes, event types, detectors with versioned rules, models and model versions, external models, lists, events, predictions with rule evaluation, batch jobs, KMS settings and tags.",
    docsHref: "../../../index.html#frauddetector",
    exampleHref: "../../../../examples/aws/frauddetector/docker-compose.yml"
  },

  {
    id: "wafregional",
    name: "WAF Classic Regional",
    category: "security",
    summary: "Emulate WAF Classic regional conditions, rules, rate-based rules, rule groups and web ACLs with change tokens, resource associations, logging, rule group policies and tags.",
    docsHref: "../../../index.html#wafregional",
    exampleHref: "../../../../examples/aws/waf-regional/docker-compose.yml"
  },

  {
    id: "budgets",
    name: "Budgets",
    category: "integration",
    summary: "Emulate budgets with limits, planned limits, filters and cost types, notifications with email and SNS subscribers, budget actions with approve/reverse/reset and history, performance history and tags.",
    docsHref: "../../../index.html#budgets",
    exampleHref: "../../../../examples/aws/budgets/docker-compose.yml"
  },
  {
    id: "costoptimizationhub",
    name: "Cost Optimization Hub",
    category: "integration",
    summary: "Emulate enrollment and preferences, a deterministic catalog of rightsizing, Graviton, Savings Plans and Reserved Instance recommendations with filters, ordering and summaries, and efficiency metrics.",
    docsHref: "../../../index.html#costoptimizationhub",
    exampleHref: "../../../../examples/aws/cost-optimization-hub/docker-compose.yml"
  },
  {
    id: "bcmdataexports",
    name: "Data Exports",
    category: "integration",
    summary: "Emulate billing data exports: SQL-checked queries over CUR 2.0, FOCUS, recommendation and carbon tables, executions that deliver manifests and CSV files to S3, table schemas and tags.",
    docsHref: "../../../index.html#bcmdataexports",
    exampleHref: "../../../../examples/aws/bcm-data-exports/docker-compose.yml"
  },
  {
    id: "account",
    name: "Account",
    category: "integration",
    summary: "Emulate account name and state, alternate and contact information with phone verification, primary email updates with one-time codes, and Region opt-in status for the management and member accounts.",
    docsHref: "../../../index.html#account",
    exampleHref: "../../../../examples/aws/account/docker-compose.yml"
  },
  {
    id: "freetier",
    name: "Free Tier",
    category: "integration",
    summary: "Emulate the free account plan with credits and upgrade to paid, sign-up activities (the Budgets activity completes from Stackyard's Budgets), and a filterable free-tier usage catalog.",
    docsHref: "../../../index.html#freetier",
    exampleHref: "../../../../examples/aws/freetier/docker-compose.yml"
  },
  {
    id: "taxsettings",
    name: "Tax Settings",
    category: "integration",
    summary: "Emulate tax registrations for the caller and member accounts with S3-backed documents, supplemental VAT registrations, tax exemptions with Pending then Valid status, and tax inheritance.",
    docsHref: "../../../index.html#taxsettings",
    exampleHref: "../../../../examples/aws/taxsettings/docker-compose.yml"
  },
  {
    id: "geomaps",
    name: "Location Maps V2",
    category: "integration",
    summary: "Emulate MapLibre style descriptors, vector, satellite and DEM tiles, sprites, glyphs and static map images as small, valid placeholder payloads with model-checked parameters.",
    docsHref: "../../../index.html#geomaps",
    exampleHref: "../../../../examples/aws/geo-maps/docker-compose.yml"
  },
  {
    id: "geoplaces",
    name: "Location Places V2",
    category: "integration",
    summary: "Emulate geocoding, reverse geocoding, text and nearby search, autocomplete and suggestions over a deterministic built-in gazetteer of cities, landmarks and addresses.",
    docsHref: "../../../index.html#geoplaces",
    exampleHref: "../../../../examples/aws/geo-places/docker-compose.yml"
  },
  {
    id: "georoutes",
    name: "Location Routes V2",
    category: "integration",
    summary: "Emulate routes, route matrices, isolines, waypoint optimization and road snapping with deterministic great-circle geometry and flexible polylines.",
    docsHref: "../../../index.html#georoutes",
    exampleHref: "../../../../examples/aws/geo-routes/docker-compose.yml"
  },
  {
    id: "pi",
    name: "Performance Insights",
    category: "integration",
    summary: "Emulate Performance Insights for RDS and DocumentDB instances: deterministic db.load and OS/DB counter metrics, dimension keys by SQL, wait event, user and host, analysis reports with insights and recommendations, and tags.",
    docsHref: "../../../index.html#pi",
    exampleHref: "../../../../examples/aws/performance-insights/docker-compose.yml"
  },
  {
    id: "cloudsearch",
    name: "CloudSearch",
    category: "integration",
    summary: "Emulate CloudSearch domains: index fields of every type, expressions, analysis schemes and suggesters deployed by IndexDocuments, plus scaling, availability, endpoint and access-policy options.",
    docsHref: "../../../index.html#cloudsearch",
    exampleHref: "../../../../examples/aws/cloudsearch/docker-compose.yml"
  },
  {
    id: "cloudsearchdomain",
    name: "CloudSearch Domain",
    category: "integration",
    summary: "Emulate CloudSearch document upload, search and suggest against domains created with CloudSearch: simple and structured queries, filters, sorting, cursors, facets, stats, highlights, expressions and fuzzy suggestions.",
    docsHref: "../../../index.html#cloudsearchdomain",
    exampleHref: "../../../../examples/aws/cloudsearch-domain/docker-compose.yml"
  },
  {
    id: "kendraranking",
    name: "Kendra Intelligent Ranking",
    category: "integration",
    summary: "Emulate Kendra Intelligent Ranking rescore execution plans with deterministic term-overlap rescoring, document groups and tags.",
    docsHref: "../../../index.html#kendraranking",
    exampleHref: "../../../../examples/aws/kendra-ranking/docker-compose.yml"
  },
  {
    id: "servicecatalog",
    name: "Service Catalog",
    category: "integration",
    summary: "Emulate portfolios, products and provisioning artifacts from S3 templates, sharing, constraints, TagOptions, service actions, provisioning with real CloudFormation stacks, plans and records.",
    docsHref: "../../../index.html#servicecatalog",
    exampleHref: "../../../../examples/aws/servicecatalog/docker-compose.yml"
  },
  {
    id: "cloudfront",
    name: "CloudFront",
    category: "integration",
    summary: "Plan staged coverage for distributions, policies, security primitives, invalidations, edge compute, and tagging APIs.",
    docsHref: "../../../index.html#cloudfront",
    exampleHref: "../../../../examples/aws/cloudfront/docker-compose.yml"
  },
  {
    id: "cloudfrontkeyvaluestore",
    name: "CloudFront KeyValueStore",
    category: "integration",
    summary: "Read and write keys in CloudFront KeyValueStores with ETag concurrency, atomic batch updates, size quotas, and SigV4A signing.",
    docsHref: "../../../index.html#cloudfrontkeyvaluestore",
    exampleHref: "../../../../examples/aws/cloudfront-keyvaluestore/docker-compose.yml"
  },
  {
    id: "directconnect",
    name: "Direct Connect",
    category: "integration",
    summary: "Plan staged coverage for physical links, LAGs, virtual interfaces, direct-connect gateways, and routing/security workflows.",
    docsHref: "../../../index.html#directconnect",
    exampleHref: "../../../../examples/aws/directconnect/docker-compose.yml"
  },
  {
    id: "networkmanager",
    name: "Network Manager (Cloud WAN)",
    category: "integration",
    summary: "Global networks with sites, devices and links, Cloud WAN core networks driven by validated policy documents and change sets, and VPC, VPN, Connect, peering and Direct Connect attachments.",
    docsHref: "../../../index.html#networkmanager",
    exampleHref: "../../../../examples/aws/networkmanager/docker-compose.yml"
  },
  {
    id: "route53profiles",
    name: "Route 53 Profiles",
    category: "integration",
    summary: "Bundle private hosted zones, Resolver rules and DNS Firewall rule groups into Profiles and associate them with VPCs, one Profile per VPC.",
    docsHref: "../../../index.html#route53profiles",
    exampleHref: "../../../../examples/aws/route53profiles/docker-compose.yml"
  },
  {
    id: "serverlessrepo",
    name: "Serverless Application Repository",
    category: "integration",
    summary: "Publish SAM applications and versions with parameters, capabilities and nested-app dependencies read from the template; share them with policies and deploy them as CloudFormation change sets.",
    docsHref: "../../../index.html#serverlessrepo",
    exampleHref: "../../../../examples/aws/serverlessrepo/docker-compose.yml"
  },
  {
    id: "config",
    name: "AWS Config",
    category: "integration",
    summary: "Plan staged coverage for recorder, rule, remediation, conformance, aggregation, and query APIs.",
    docsHref: "../../../index.html#config",
    exampleHref: "../../../../examples/aws/config/docker-compose.yml"
  },
  {
    id: "cloudtrail",
    name: "CloudTrail",
    category: "integration",
    summary: "Plan and validate trail lifecycle, event data stores, channels, import/query, and policy APIs.",
    docsHref: "../../../index.html#cloudtrail",
    exampleHref: "../../../../examples/aws/cloudtrail/docker-compose.yml"
  },
  {
    id: "cloudtraildata",
    name: "CloudTrail Data",
    category: "integration",
    summary: "Ingest external audit events into CloudTrail Lake channels with PutAuditEvents and query them back.",
    docsHref: "../../../index.html#cloudtraildata",
    exampleHref: "../../../../examples/aws/cloudtrail-data/docker-compose.yml"
  },
  {
    id: "cloudwatch",
    name: "CloudWatch",
    category: "integration",
    summary: "Plan and validate alarms, metrics, dashboards, anomaly detectors, insight rules, and metric stream APIs.",
    docsHref: "../../../index.html#cloudwatch",
    exampleHref: "../../../../examples/aws/cloudwatch/docker-compose.yml"
  },
  {
    id: "cloudwatchlogs",
    name: "CloudWatch Logs",
    category: "integration",
    summary: "Plan and validate log groups/streams, log event ingestion/search, query/export lifecycle, delivery, policies, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatchlogs",
    exampleHref: "../../../../examples/aws/cloudwatchlogs/docker-compose.yml"
  },
  {
    id: "cloudwatchrum",
    name: "CloudWatch RUM",
    category: "integration",
    summary: "Plan and validate app monitor lifecycle, RUM metric definitions, policy, event ingestion, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatchrum",
    exampleHref: "../../../../examples/aws/cloudwatchrum/docker-compose.yml"
  },
  {
    id: "cloudwatch-observability-admin",
    name: "CloudWatch Observability Admin",
    category: "integration",
    summary: "Plan telemetry pipeline/rule lifecycle, org-level evaluation/centralization workflows, integrations, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatch-observability-admin",
    exampleHref: "../../../../examples/aws/cloudwatch-observability-admin/docker-compose.yml"
  },
  {
    id: "oam",
    name: "CloudWatch OAM",
    category: "integration",
    summary: "Plan and validate link/sink lifecycle, sink policies, and cross-account observability tagging APIs.",
    docsHref: "../../../index.html#oam",
    exampleHref: "../../../../examples/aws/oam/docker-compose.yml"
  },
  {
    id: "applicationsignals",
    name: "Application Signals",
    category: "integration",
    summary: "Plan and validate SLO, service graph, grouping, audits, exclusion windows, and tagging APIs.",
    docsHref: "../../../index.html#applicationsignals",
    exampleHref: "../../../../examples/aws/applicationsignals/docker-compose.yml"
  },
  {
    id: "networkflowmonitor",
    name: "Network Flow Monitor",
    category: "integration",
    summary: "Plan and validate monitor/scope lifecycle, top-contributor queries, and tagging APIs.",
    docsHref: "../../../index.html#networkflowmonitor",
    exampleHref: "../../../../examples/aws/networkflowmonitor/docker-compose.yml"
  },
  {
    id: "internet-monitor",
    name: "Internet Monitor",
    category: "integration",
    summary: "Plan and validate monitor lifecycle, internet/health events, queries, and tagging APIs.",
    docsHref: "../../../index.html#internet-monitor",
    exampleHref: "../../../../examples/aws/internet-monitor/docker-compose.yml"
  },
  {
    id: "network-synthetic-monitor",
    name: "Network Synthetic Monitor",
    category: "integration",
    summary: "Plan and validate monitor/probe lifecycle and tagging APIs for Network Synthetic Monitor.",
    docsHref: "../../../index.html#network-synthetic-monitor",
    exampleHref: "../../../../examples/aws/network-synthetic-monitor/docker-compose.yml"
  },
  {
    id: "cloudwatchsynthetics",
    name: "CloudWatch Synthetics",
    category: "integration",
    summary: "Plan and validate canary lifecycle, dry-run execution, group associations, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatchsynthetics",
    exampleHref: "../../../../examples/aws/cloudwatchsynthetics/docker-compose.yml"
  },
  {
    id: "cloudwatchapplicationinsights",
    name: "CloudWatch Application Insights",
    category: "integration",
    summary: "Stage application/component/workload lifecycle, problem analysis, and log pattern APIs.",
    docsHref: "../../../index.html#cloudwatchapplicationinsights",
    exampleHref: "../../../../examples/aws/cloudwatchapplicationinsights/docker-compose.yml"
  },
  {
    id: "cloudwatchinvestigations",
    name: "CloudWatch Investigations",
    category: "integration",
    summary: "Plan and validate investigation-group lifecycle, policy, and tagging API workflows.",
    docsHref: "../../../index.html#cloudwatchinvestigations",
    exampleHref: "../../../../examples/aws/cloudwatchinvestigations/docker-compose.yml"
  },
  {
    id: "codeartifact",
    name: "CodeArtifact",
    category: "integration",
    summary: "Plan and validate package domain, repository, package-group, and version API workflows.",
    docsHref: "../../../index.html#codeartifact",
    exampleHref: "../../../../examples/aws/codeartifact/docker-compose.yml"
  },
  {
    id: "codeconnections",
    name: "CodeConnections",
    category: "integration",
    summary: "Emulate source-provider connections, self-managed hosts, repository links and CloudFormation Git sync configurations.",
    docsHref: "../../../index.html#codeconnections",
    exampleHref: "../../../../examples/aws/codeconnections/docker-compose.yml"
  },
  {
    id: "codestarconnections",
    name: "CodeStar Connections",
    category: "integration",
    summary: "Emulate source-provider connections, self-managed hosts, repository links and CloudFormation Git sync configurations (shared store with CodeConnections).",
    docsHref: "../../../index.html#codestarconnections",
    exampleHref: "../../../../examples/aws/codeconnections/docker-compose.yml"
  },
  {
    id: "codebuild",
    name: "CodeBuild",
    category: "integration",
    summary: "Plan and validate project, build, report, fleet, sandbox, and webhook API flows.",
    docsHref: "../../../index.html#codebuild",
    exampleHref: "../../../../examples/aws/codebuild/docker-compose.yml"
  },
  {
    id: "codecatalyst",
    name: "CodeCatalyst",
    category: "integration",
    summary: "Plan and validate spaces, projects, repositories, dev environments, workflows, and access tokens.",
    docsHref: "../../../index.html#codecatalyst",
    exampleHref: "../../../../examples/aws/codecatalyst/docker-compose.yml"
  },
  {
    id: "connect",
    name: "Connect",
    category: "integration",
    summary: "Plan staged coverage for instance, user, flow, contact, workspace, search, and tagging APIs.",
    docsHref: "../../../index.html#connect",
    exampleHref: "../../../../examples/aws/connect/docker-compose.yml"
  },
  {
    id: "connectcases",
    name: "Connect Cases",
    category: "integration",
    summary: "Emulate Connect Cases domains, fields, layouts, templates, case rules, cases with audit history, related items, search and merges.",
    docsHref: "../../../index.html#connectcases",
    exampleHref: "../../../../examples/aws/connect-cases/docker-compose.yml"
  },
  {
    id: "connectparticipant",
    name: "Connect Participant",
    category: "integration",
    summary: "Emulate Connect customer chat: participant connections, messages, events, read receipts, attachments, transcripts and disconnects.",
    docsHref: "../../../index.html#connectparticipant",
    exampleHref: "../../../../examples/aws/connect-participant/docker-compose.yml"
  },
  {
    id: "connectcontactlens",
    name: "Connect Contact Lens",
    category: "integration",
    summary: "Emulate Contact Lens real-time analysis segments over Connect chat transcripts, with sentiment and issue detection.",
    docsHref: "../../../index.html#connectcontactlens",
    exampleHref: "../../../../examples/aws/connect-contact-lens/docker-compose.yml"
  },
  {
    id: "dlm",
    name: "Data Lifecycle Manager",
    category: "integration",
    summary: "Plan staged coverage for lifecycle policy CRUD, filtering, and tagging APIs.",
    docsHref: "../../../index.html#dlm",
    exampleHref: "../../../../examples/aws/dlm/docker-compose.yml"
  },
  {
    id: "datazone",
    name: "DataZone",
    category: "integration",
    summary: "Plan staged coverage for domains, projects, environments, governance, subscriptions, lineage, and tagging APIs.",
    docsHref: "../../../index.html#datazone",
    exampleHref: "../../../../examples/aws/datazone/docker-compose.yml"
  },
  {
    id: "elasticache",
    name: "ElastiCache",
    category: "integration",
    summary: "Plan staged coverage for query actions spanning cluster, replication, serverless, migration, and tagging workflows.",
    docsHref: "../../../index.html#elasticache",
    exampleHref: "../../../../examples/aws/elasticache/docker-compose.yml"
  },
  {
    id: "emr",
    name: "EMR",
    category: "integration",
    summary: "Plan staged coverage for cluster lifecycle, step execution, scaling controls, Studio APIs, and tagging workflows.",
    docsHref: "../../../index.html#emr",
    exampleHref: "../../../../examples/aws/emr/docker-compose.yml"
  },
  {
    id: "emr-on-eks",
    name: "EMR on EKS",
    category: "integration",
    summary: "Plan staged coverage for virtual clusters, job runs, managed endpoints, job templates, and tagging APIs.",
    docsHref: "../../../index.html#emr-on-eks",
    exampleHref: "../../../../examples/aws/emr-on-eks/docker-compose.yml"
  },
  {
    id: "emr-serverless",
    name: "EMR Serverless",
    category: "integration",
    summary: "Plan staged coverage for application lifecycle, job runs, dashboard access, and tagging APIs.",
    docsHref: "../../../index.html#emr-serverless",
    exampleHref: "../../../../examples/aws/emr-serverless/docker-compose.yml"
  },
  {
    id: "firehose",
    name: "Data Firehose",
    category: "integration",
    summary: "Plan staged coverage for delivery stream lifecycle, ingestion, encryption, destination updates, and tagging APIs.",
    docsHref: "../../../index.html#firehose",
    exampleHref: "../../../../examples/aws/firehose/docker-compose.yml"
  },
  {
    id: "finspace",
    name: "FinSpace",
    category: "integration",
    summary: "Plan staged coverage for datasets, changesets, data views, users, permission groups, and credentials APIs.",
    docsHref: "../../../index.html#finspace",
    exampleHref: "../../../../examples/aws/finspace/docker-compose.yml"
  },
  {
    id: "finspace-management",
    name: "FinSpace Management",
    category: "integration",
    summary: "Plan staged coverage for environments, Kx clusters/databases/dataviews, users, networking, and tagging APIs.",
    docsHref: "../../../index.html#finspace-management",
    exampleHref: "../../../../examples/aws/finspace-management/docker-compose.yml"
  },
  {
    id: "gameliftstreams",
    name: "GameLift Streams",
    category: "integration",
    summary: "Plan staged coverage for applications, stream groups, stream sessions, connection/export workflows, and tagging APIs.",
    docsHref: "../../../index.html#gameliftstreams",
    exampleHref: "../../../../examples/aws/gameliftstreams/docker-compose.yml"
  },
  {
    id: "kendra",
    name: "Kendra",
    category: "integration",
    summary: "Plan staged coverage for indices, data sources, FAQs, thesauri, document ingestion, query workflows, and tagging APIs.",
    docsHref: "../../../index.html#kendra",
    exampleHref: "../../../../examples/aws/kendra/docker-compose.yml"
  },
  {
    id: "kinesis",
    name: "Kinesis Data Streams",
    category: "integration",
    summary: "Plan staged coverage for stream lifecycle, shard scaling, record ingest/read APIs, consumer management, and policy/tagging workflows.",
    docsHref: "../../../index.html#kinesis",
    exampleHref: "../../../../examples/aws/kinesis/docker-compose.yml"
  },
  {
    id: "kinesisvideostreams",
    name: "Kinesis Video Streams",
    category: "integration",
    summary: "Plan staged coverage for stream/channel lifecycle, endpoint discovery, edge workflows, retention, and tagging APIs.",
    docsHref: "../../../index.html#kinesis-video-streams",
    exampleHref: "../../../../examples/aws/kinesisvideostreams/docker-compose.yml"
  },
  {
    id: "kinesisvideomedia",
    name: "Kinesis Video Streams Media",
    category: "integration",
    summary: "PutMedia ingest of MKV fragments with acknowledgements, and GetMedia reads by fragment number, timestamp, continuation token, EARLIEST or NOW.",
    docsHref: "../../../index.html#kinesisvideomedia",
    exampleHref: "../../../../examples/aws/kinesis-video-media/docker-compose.yml"
  },
  {
    id: "kinesisvideoarchivedmedia",
    name: "Kinesis Video Streams Archived Media",
    category: "integration",
    summary: "ListFragments, GetMediaForFragmentList, GetClip, GetImages and HLS/DASH session URLs that serve playlists of the stored fragments.",
    docsHref: "../../../index.html#kinesisvideoarchivedmedia",
    exampleHref: "../../../../examples/aws/kinesis-video-media/docker-compose.yml"
  },
  {
    id: "kinesisvideosignaling",
    name: "Kinesis Video Signaling Channels",
    category: "integration",
    summary: "GetIceServerConfig with deterministic TURN credentials, SendAlexaOfferToMaster, and a WSS signaling relay between a channel's master and viewers.",
    docsHref: "../../../index.html#kinesisvideosignaling",
    exampleHref: "../../../../examples/aws/kinesis-video-signaling/docker-compose.yml"
  },
  {
    id: "kinesisvideowebrtcstorage",
    name: "Kinesis Video WebRTC Storage",
    category: "integration",
    summary: "JoinStorageSession and JoinStorageSessionAsViewer for channels with media storage linked to an active stream.",
    docsHref: "../../../index.html#kinesisvideowebrtcstorage",
    exampleHref: "../../../../examples/aws/kinesis-video-signaling/docker-compose.yml"
  },
  {
    id: "mq",
    name: "Amazon MQ",
    category: "integration",
    summary: "Plan staged coverage for broker lifecycle, engine/instance options, configuration revisions, users, reboot/promote, and tagging APIs.",
    docsHref: "../../../index.html#mq",
    exampleHref: "../../../../examples/aws/mq/docker-compose.yml"
  },
  {
    id: "msk",
    name: "Amazon MSK",
    category: "integration",
    summary: "Plan staged coverage for MSK API v2 cluster create/list/describe and cluster-operation read/list workflows.",
    docsHref: "../../../index.html#msk",
    exampleHref: "../../../../examples/aws/msk/docker-compose.yml"
  },
  {
    id: "msk-connect",
    name: "Amazon MSK Connect",
    category: "integration",
    summary: "Plan staged coverage for connector, custom plugin, worker configuration, operation history, and tagging workflows.",
    docsHref: "../../../index.html#msk-connect",
    exampleHref: "../../../../examples/aws/msk-connect/docker-compose.yml"
  },
  {
    id: "mwaa",
    name: "MWAA",
    category: "integration",
    summary: "Plan staged coverage for environment lifecycle, token issuance, Airflow REST proxy, metrics ingestion, and tagging APIs.",
    docsHref: "../../../index.html#mwaa",
    exampleHref: "../../../../examples/aws/mwaa/docker-compose.yml"
  },
  {
    id: "mwaa-serverless",
    name: "MWAA Serverless",
    category: "integration",
    summary: "Plan staged coverage for workflow lifecycle, run execution/control, task-instance reads, versioning, and tagging APIs.",
    docsHref: "../../../index.html#mwaa-serverless",
    exampleHref: "../../../../examples/aws/mwaa-serverless/docker-compose.yml"
  },
  {
    id: "qdeveloper",
    name: "AWS Q Developer",
    category: "integration",
    summary: "Plan and validate account preferences, chat channel configurations, custom actions, and associations.",
    docsHref: "../../../index.html#qdeveloper",
    exampleHref: "../../../../examples/aws/qdeveloper/docker-compose.yml"
  },
  {
    id: "codeguru",
    name: "CodeGuru Reviewer",
    category: "integration",
    summary: "Plan and validate repository associations, code reviews, recommendations, metrics, and tagging APIs.",
    docsHref: "../../../index.html#codeguru",
    exampleHref: "../../../../examples/aws/codeguru/docker-compose.yml"
  },
  {
    id: "codeguru-profiler",
    name: "CodeGuru Profiler",
    category: "integration",
    summary: "Plan and validate profiling group, agent profile ingestion, findings, recommendation, and policy APIs.",
    docsHref: "../../../index.html#codeguru-profiler",
    exampleHref: "../../../../examples/aws/codeguru-profiler/docker-compose.yml"
  },
  {
    id: "comprehend",
    name: "Comprehend",
    category: "integration",
    summary: "Emulate text analysis, batch jobs, model endpoints, flywheels, and tagging workflows.",
    docsHref: "../../../index.html#comprehend",
    exampleHref: "../../../../examples/aws/comprehend/docker-compose.yml"
  },
  {
    id: "comprehend-medical",
    name: "Comprehend Medical",
    category: "integration",
    summary: "Plan and validate clinical text extraction, ontology inference, and async medical analysis job workflows.",
    docsHref: "../../../index.html#comprehend-medical",
    exampleHref: "../../../../examples/aws/comprehend-medical/docker-compose.yml"
  },
  {
    id: "devops-guru",
    name: "DevOps Guru",
    category: "integration",
    summary: "Emulate insights, anomalies, recommendations, resource collections, and integration configuration APIs.",
    docsHref: "../../../index.html#devops-guru",
    exampleHref: "../../../../examples/aws/devops-guru/docker-compose.yml"
  },
  {
    id: "codecommit",
    name: "CodeCommit",
    category: "integration",
    summary: "Emulate Git repositories with branches, commits, files, diffs, merges with conflict detection, pull requests, approval rules, comments, and triggers.",
    docsHref: "../../../index.html#codecommit",
    exampleHref: "../../../../examples/aws/codecommit/docker-compose.yml"
  },
  {
    id: "codepipeline",
    name: "CodePipeline",
    category: "integration",
    summary: "Stage and validate pipeline, execution, action type, job, and webhook API flows.",
    docsHref: "../../../index.html#codepipeline",
    exampleHref: "../../../../examples/aws/codepipeline/docker-compose.yml"
  },
  {
    id: "codestarnotifications",
    name: "CodeStar Notifications",
    category: "integration",
    summary: "Notification rules on CodeBuild, CodeDeploy, CodePipeline and CodeCommit resources with SNS and AWS Chatbot targets.",
    docsHref: "../../../index.html#codestarnotifications",
    exampleHref: "../../../../examples/aws/codestar-notifications/docker-compose.yml"
  },
  {
    id: "organizations",
    name: "Organizations",
    category: "integration",
    summary: "Stage and validate organization hierarchy, policy, handshake, and delegated-admin control plane APIs.",
    docsHref: "../../../index.html#organizations",
    exampleHref: "../../../../examples/aws/organizations/docker-compose.yml"
  },
  {
    id: "codedeploy",
    name: "CodeDeploy",
    category: "integration",
    summary: "Plan and validate deployment applications, groups, targets, and revision workflows.",
    docsHref: "../../../index.html#codedeploy",
    exampleHref: "../../../../examples/aws/codedeploy/docker-compose.yml"
  },
  {
    id: "cloudhsm",
    name: "CloudHSM",
    category: "security",
    summary: "Test cluster, HSM, and backup orchestration flows locally.",
    docsHref: "../../../index.html#cloudhsm",
    exampleHref: "../../../../examples/aws/cloudhsm/docker-compose.yml"
  },
  {
    id: "cognito-user-pools",
    name: "Cognito User Pools",
    category: "security",
    summary: "Plan and validate user pool auth, federation, and identity workflows.",
    docsHref: "../../../index.html#cognito-user-pools",
    exampleHref: "../../../../examples/aws/cognitouserpools/docker-compose.yml"
  },
  {
    id: "cognito",
    name: "Cognito Federated Identities",
    category: "security",
    summary: "Emulate identity pool, token, credential, role mapping, and tagging workflows.",
    docsHref: "../../../index.html#cognito",
    exampleHref: "../../../../examples/aws/cognito/docker-compose.yml"
  },
  {
    id: "cognitosync",
    name: "Cognito Sync",
    category: "security",
    summary: "Plan and validate identity-pool dataset sync, subscriptions, and record flows.",
    docsHref: "../../../index.html#cognitosync",
    exampleHref: "../../../../examples/aws/cognitosync/docker-compose.yml"
  },
  {
    id: "dynamodb",
    name: "DynamoDB",
    category: "database",
    summary: "Exercise table, item, and query-style DynamoDB operations.",
    docsHref: "../../../index.html#dynamodb",
    exampleHref: "../../../../examples/aws/dynamodb/docker-compose.yml"
  },
  {
    id: "databrew",
    name: "Glue DataBrew",
    category: "database",
    summary: "Run DataBrew datasets, versioned recipes, project sessions and recipe or profile jobs that read CSV/JSON from S3 and write results back, plus rulesets, schedules and tags.",
    docsHref: "../../../index.html#databrew",
    exampleHref: "../../../../examples/aws/databrew/docker-compose.yml"
  },
  {
    id: "dax",
    name: "DynamoDB Accelerator (DAX)",
    category: "database",
    summary: "Manage DAX clusters, node replication, parameter groups, subnet groups, events and tags (control plane only).",
    docsHref: "../../../index.html#dax",
    exampleHref: "../../../../examples/aws/dax/docker-compose.yml"
  },
  {
    id: "route53resolver",
    name: "Route 53 Resolver",
    category: "integration",
    summary: "Emulate Resolver endpoints, forwarding rules, query logging, DNSSEC and resolver settings, and DNS Firewall domain lists, rule groups and VPC associations.",
    docsHref: "../../../index.html#route53resolver",
    exampleHref: "../../../../examples/aws/route53resolver/docker-compose.yml"
  },
  {
    id: "dynamodbstreams",
    name: "DynamoDB Streams",
    category: "database",
    summary: "Read table change records with shard iterators and GetRecords.",
    docsHref: "../../../index.html#dynamodbstreams",
    exampleHref: "../../../../examples/aws/dynamodb-streams/docker-compose.yml"
  },
  {
    id: "ebs",
    name: "EBS",
    category: "compute",
    summary: "Simulate block storage lifecycle calls used by EC2 workflows.",
    docsHref: null,
    exampleHref: "../../../../examples/aws/ebs/docker-compose.yml"
  },
  {
    id: "ec2",
    name: "EC2",
    category: "compute",
    summary: "Coverage for instance, networking, and lifecycle operations.",
    docsHref: "../../../index.html#ec2",
    exampleHref: "../../../../examples/aws/ec2/docker-compose.yml"
  },
  {
    id: "ec2instanceconnect",
    name: "EC2 Instance Connect",
    category: "compute",
    summary: "Emulate pushing SSH and serial console public keys to running EC2 instances, validated against the EC2 store, with 60-second key expiry.",
    docsHref: "../../../index.html#ec2instanceconnect",
    exampleHref: "../../../../examples/aws/ec2-instance-connect/docker-compose.yml"
  },
  {
    id: "ec2-autoscaling",
    name: "EC2 Auto Scaling",
    category: "compute",
    summary: "Plan and validate Auto Scaling group, launch configuration, policy, lifecycle, refresh, and traffic source workflows.",
    docsHref: "../../../index.html#ec2-autoscaling",
    exampleHref: "../../../../examples/aws/ec2-autoscaling/docker-compose.yml"
  },
  {
    id: "elasticloadbalancing",
    name: "Elastic Load Balancing (ELB)",
    category: "compute",
    summary: "Plan staged coverage for load balancers, listeners, rules, target groups, trust stores, and tagging workflows.",
    docsHref: "../../../index.html#elasticloadbalancing",
    exampleHref: "../../../../examples/aws/elasticloadbalancing/docker-compose.yml"
  },
  {
    id: "elasticloadbalancingv2",
    name: "Elastic Load Balancing (ELB Classic)",
    category: "compute",
    summary: "Plan staged coverage for Classic ELB load balancers, listeners, health checks, policies, and tags.",
    docsHref: "../../../index.html#elasticloadbalancingv2",
    exampleHref: "../../../../examples/aws/elasticloadbalancingv2/docker-compose.yml"
  },
  {
    id: "applicationautoscaling",
    name: "Application Auto Scaling",
    category: "compute",
    summary: "Emulate scalable targets checked against ECS, DynamoDB and Lambda, step, target tracking and predictive policies with CloudWatch alarms, scheduled actions, activities and tags.",
    docsHref: "../../../index.html#applicationautoscaling",
    exampleHref: "../../../../examples/aws/application-autoscaling/docker-compose.yml"
  },
  {
    id: "aws-autoscaling",
    name: "AWS Auto Scaling",
    category: "compute",
    summary: "Plan and validate scaling plan lifecycle, resource projections, and forecast datapoint workflows.",
    docsHref: "../../../index.html#aws-autoscaling",
    exampleHref: "../../../../examples/aws/aws-autoscaling/docker-compose.yml"
  },
  {
    id: "compute-optimizer",
    name: "Compute Optimizer",
    category: "compute",
    summary: "Plan and validate enrollment, recommendation, export, and automation API workflows.",
    docsHref: "../../../index.html#compute-optimizer",
    exampleHref: "../../../../examples/aws/compute-optimizer/docker-compose.yml"
  },
  {
    id: "ecr",
    name: "ECR",
    category: "compute",
    summary: "Work with repositories, image metadata, and container registry APIs.",
    docsHref: "../../../index.html#ecr",
    exampleHref: "../../../../examples/aws/ecr/docker-compose.yml"
  },
  {
    id: "ecrpublic",
    name: "ECR Public",
    category: "compute",
    summary: "Run a public container registry: repositories with gallery catalog data, layer uploads, image manifests, tags and the registry alias.",
    docsHref: "../../../index.html#ecrpublic",
    exampleHref: "../../../../examples/aws/ecr-public/docker-compose.yml"
  },
  {
    id: "ecs",
    name: "ECS",
    category: "compute",
    summary: "Validate cluster, task definition, and service control planes.",
    docsHref: "../../../index.html#ecs",
    exampleHref: "../../../../examples/aws/ecs/docker-compose.yml"
  },
  {
    id: "eks",
    name: "EKS",
    category: "compute",
    summary: "Run Kubernetes cluster management scenarios in local integration tests.",
    docsHref: "../../../index.html#eks",
    exampleHref: "../../../../examples/aws/eks/docker-compose.yml"
  },
  {
    id: "eksauth",
    name: "EKS Auth",
    category: "compute",
    summary: "Exchange EKS Pod Identity service-account tokens for temporary role credentials in local integration tests.",
    docsHref: "../../../index.html#eksauth",
    exampleHref: "../../../../examples/aws/eks-auth/docker-compose.yml"
  },
  {
    id: "elasticbeanstalk",
    name: "Elastic Beanstalk",
    category: "compute",
    summary: "Mock application environment and deployment lifecycle endpoints.",
    docsHref: "../../../index.html#elasticbeanstalk",
    exampleHref: null
  },
  {
    id: "eventbridge",
    name: "EventBridge",
    category: "integration",
    summary: "Create rules, buses, and event targets for routing workflows.",
    docsHref: "../../../index.html#eventbridge",
    exampleHref: "../../../../examples/aws/eventbridge/docker-compose.yml"
  },
  {
    id: "scheduler",
    name: "EventBridge Scheduler",
    category: "integration",
    summary: "Emulate at(), rate() and cron() schedules with timezones, schedule groups, templated and universal targets, retry and dead-letter settings, plus on-demand delivery to SQS, SNS and EventBridge.",
    docsHref: "../../../index.html#scheduler",
    exampleHref: "../../../../examples/aws/scheduler/docker-compose.yml"
  },
  {
    id: "schemas",
    name: "EventBridge Schemas",
    category: "integration",
    summary: "Emulate schema registries with versioned OpenAPI 3 and JSON Schema Draft 4 schemas, search and export, schema inference from sample events, generated Java, Python, TypeScript and Go code bindings, discoverers on event buses, resource policies and tags.",
    docsHref: "../../../index.html#schemas",
    exampleHref: "../../../../examples/aws/schemas/docker-compose.yml"
  },
  {
    id: "pipes",
    name: "EventBridge Pipes",
    category: "integration",
    summary: "Create point-to-point pipes from SQS, Kinesis, DynamoDB streams, MQ and Kafka sources to targets, with filters, enrichment and start/stop state.",
    docsHref: "../../../index.html#pipes",
    exampleHref: "../../../../examples/aws/pipes/docker-compose.yml"
  },
  {
    id: "personalize",
    name: "Amazon Personalize",
    category: "integration",
    summary: "Dataset groups (custom or ECOMMERCE / VIDEO_ON_DEMAND domain), Avro schemas, datasets fed by S3 imports and Personalize Events, solutions and versions, campaigns, recommenders, filters, metric attributions, and export, data deletion, batch inference and batch segment jobs.",
    docsHref: "../../../index.html#personalize",
    exampleHref: "../../../../examples/aws/personalize/docker-compose.yml"
  },
  {
    id: "personalizeevents",
    name: "Amazon Personalize Events",
    category: "integration",
    summary: "Stream interaction events and action interactions through an event tracker, and add items, users and actions to Personalize datasets.",
    docsHref: "../../../index.html#personalizeevents",
    exampleHref: "../../../../examples/aws/personalize-events/docker-compose.yml"
  },
  {
    id: "personalizeruntime",
    name: "Amazon Personalize Runtime",
    category: "integration",
    summary: "Deterministic recommendations, personalized rankings and next-best actions from Personalize campaigns and recommenders over the dataset group's current data.",
    docsHref: "../../../index.html#personalizeruntime",
    exampleHref: "../../../../examples/aws/personalize-runtime/docker-compose.yml"
  },
  {
    id: "pinpointemail",
    name: "Amazon Pinpoint Email",
    category: "integration",
    summary: "Pinpoint Email on SESv2's account state: identities with DKIM, feedback and MAIL FROM settings, configuration sets with delivery, reputation, sending and tracking options and event destinations, dedicated IPs and pools, the deliverability dashboard with domain statistics and campaigns from sent mail, test reports, SendEmail (Simple, Raw or Template) with the send quota, and tags.",
    docsHref: "../../../index.html#pinpointemail",
    exampleHref: "../../../../examples/aws/pinpoint-email/docker-compose.yml"
  },
  {
    id: "pinpointsmsvoice",
    name: "Amazon Pinpoint SMS and Voice",
    category: "integration",
    summary: "Version 1 voice API: configuration sets with CloudWatch Logs, Kinesis Data Firehose or SNS event destinations, and voice messages (plain text, call instructions or SSML) validated against E.164 numbers and their configuration set.",
    docsHref: "../../../index.html#pinpointsmsvoice",
    exampleHref: "../../../../examples/aws/pinpoint-sms-voice/docker-compose.yml"
  },
  {
    id: "mailmanager",
    name: "Amazon SES Mail Manager",
    category: "integration",
    summary: "Archives filled by SESv2 and Pinpoint Email sends through configuration sets with archiving options, with searches, message content and downloads, and exports to S3; address lists with members and CSV/JSON imports through a pre-signed upload URL; ingress points, SMTP relays, rule sets and traffic policies with checked references; add-on subscriptions and instances; tags.",
    docsHref: "../../../index.html#mailmanager",
    exampleHref: "../../../../examples/aws/mailmanager/docker-compose.yml"
  },
  {
    id: "polly",
    name: "Amazon Polly",
    category: "integration",
    summary: "Synthesize speech (silent but valid MP3, Ogg, PCM and mu-law/A-law audio sized to the text), speech marks, SSML, lexicons, S3 synthesis tasks and custom voices.",
    docsHref: "../../../index.html#polly",
    exampleHref: "../../../../examples/aws/polly/docker-compose.yml"
  },
  {
    id: "iot",
    name: "IoT",
    category: "integration",
    summary: "Plan and validate IoT Core thing, policy, cert, jobs, rules, audit, and fleet management APIs.",
    docsHref: "../../../index.html#iot",
    exampleHref: "../../../../examples/aws/iot/docker-compose.yml"
  },
  {
    id: "iot-events",
    name: "IoT Events",
    category: "integration",
    summary: "Emulate detector model, alarm model, input, analysis, and tagging API workflows.",
    docsHref: "../../../index.html#iot-events",
    exampleHref: "../../../../examples/aws/iotevents/docker-compose.yml"
  },
  {
    id: "iot-greengrass",
    name: "IoT Greengrass",
    category: "integration",
    summary: "Plan and validate edge deployment, component, core-device, connectivity, and tagging API workflows.",
    docsHref: "../../../index.html#iot-greengrass",
    exampleHref: "../../../../examples/aws/iot-greengrass/docker-compose.yml"
  },
  {
    id: "iot-wireless",
    name: "IoT Wireless",
    category: "integration",
    summary: "Emulate wireless device, gateway, destination, FUOTA, multicast, and positioning API workflows.",
    docsHref: "../../../index.html#iot-wireless",
    exampleHref: "../../../../examples/aws/iot-wireless/docker-compose.yml"
  },
  {
    id: "iot-sitewise",
    name: "IoT SiteWise",
    category: "integration",
    summary: "Plan and validate industrial asset, model, portal, project, gateway, and time-series API workflows.",
    docsHref: "../../../index.html#iot-sitewise",
    exampleHref: "../../../../examples/aws/iot-sitewise/docker-compose.yml"
  },
  {
    id: "iot-twinmaker",
    name: "IoT TwinMaker",
    category: "integration",
    summary: "Plan and validate workspace, entity, component, scene, and sync APIs for digital twin workflows.",
    docsHref: "../../../index.html#iot-twinmaker",
    exampleHref: "../../../../examples/aws/iot-twinmaker/docker-compose.yml"
  },
  {
    id: "iot-secure-tunneling",
    name: "IoT Secure Tunneling",
    category: "integration",
    summary: "Emulate tunnels with one-time source and destination access tokens, token rotation, lifetime-based auto-close, close and delete, listing by thing and tags.",
    docsHref: "../../../index.html#iot-secure-tunneling",
    exampleHref: "../../../../examples/aws/iot-secure-tunneling/docker-compose.yml"
  },
  {
    id: "iot-device-advisor",
    name: "IoT Device Advisor",
    category: "integration",
    summary: "Emulate versioned suite definitions, suite runs that progress to PASS with per-test results, stop, qualification reports, the test endpoint and tags.",
    docsHref: "../../../index.html#iot-device-advisor",
    exampleHref: "../../../../examples/aws/iot-device-advisor/docker-compose.yml"
  },
  {
    id: "iot-things-graph",
    name: "IoT Things Graph",
    category: "integration",
    summary: "Emulate GraphQL entity uploads into a versioned namespace, thing associations, flow and system templates with revisions, system instance deployments, flow executions and tags.",
    docsHref: "../../../index.html#iot-things-graph",
    exampleHref: "../../../../examples/aws/iot-things-graph/docker-compose.yml"
  },
  {
    id: "keyspaces",
    name: "Keyspaces",
    category: "database",
    summary: "Use Cassandra-compatible control plane calls through AWS APIs.",
    docsHref: "../../../index.html#keyspaces",
    exampleHref: "../../../../examples/aws/keyspaces/docker-compose.yml"
  },
  {
    id: "inspector",
    name: "Inspector Classic",
    category: "security",
    summary: "Run Inspector Classic assessments over tagged EC2 instances: resource groups, targets, templates, runs that settle into findings, exclusions, reports and SNS event subscriptions.",
    docsHref: "../../../index.html#inspector",
    exampleHref: "../../../../examples/aws/inspector/docker-compose.yml"
  },
  {
    id: "inspectorscan",
    name: "Inspector Scan",
    category: "security",
    summary: "Scan CycloneDX 1.4/1.5 and SPDX 2.3 SBOMs against an embedded sample advisory list and return a CycloneDX or Inspector-format vulnerability report.",
    docsHref: "../../../index.html#inspectorscan",
    exampleHref: "../../../../examples/aws/inspector/docker-compose.yml"
  },
  {
    id: "kms",
    name: "KMS",
    category: "security",
    summary: "Manage keys, aliases, policies, and cryptographic requests.",
    docsHref: "../../../index.html#kms",
    exampleHref: "../../../../examples/aws/kms/docker-compose.yml"
  },
  {
    id: "lexruntimev2",
    name: "Lex Runtime V2",
    category: "integration",
    summary: "Converse with Lex V2 bots: recognize text and utterances, elicit slots, manage sessions, and stream StartConversation events.",
    docsHref: "../../../index.html#lexruntimev2",
    exampleHref: "../../../../examples/aws/lex-runtime-v2/docker-compose.yml"
  },
  {
    id: "transcribestreaming",
    name: "Transcribe Streaming",
    category: "integration",
    summary: "Stream audio over full-duplex event streams for live, medical, Call Analytics and Medical Scribe transcription with deterministic transcripts.",
    docsHref: "../../../index.html#transcribestreaming",
    exampleHref: "../../../../examples/aws/transcribe-streaming/docker-compose.yml"
  },
  {
    id: "lambda",
    name: "Lambda",
    category: "compute",
    summary: "Test function lifecycle plus Invoke in mock or local execution mode with real zip code support.",
    docsHref: "../../../index.html#lambda",
    exampleHref: "../../../../examples/aws/lambda/docker-compose.yml"
  },
  {
    id: "lightsail",
    name: "Lightsail",
    category: "compute",
    summary: "Exercise simplified VPS and networking APIs for app environments.",
    docsHref: "../../../index.html#lightsail",
    exampleHref: "../../../../examples/aws/lightsail/docker-compose.yml"
  },
  {
    id: "license-manager",
    name: "License Manager",
    category: "security",
    summary: "Emulate license configuration, grant, token, conversion task, asset ruleset, and reporting APIs.",
    docsHref: "../../../index.html#license-manager",
    exampleHref: "../../../../examples/aws/license-manager/docker-compose.yml"
  },
  {
    id: "license-manager-linux-subscriptions",
    name: "License Manager Linux Subscriptions",
    category: "security",
    summary: "Emulate Linux subscription providers, inventory discovery, service settings, and resource tagging APIs.",
    docsHref: "../../../index.html#license-manager-linux-subscriptions",
    exampleHref: "../../../../examples/aws/license-manager-linux-subscriptions/docker-compose.yml"
  },
  {
    id: "license-manager-user-subscriptions",
    name: "License Manager User Subscriptions",
    category: "security",
    summary: "Emulate identity providers, license server endpoints, user associations, product subscriptions, and tagging APIs.",
    docsHref: "../../../index.html#license-manager-user-subscriptions",
    exampleHref: "../../../../examples/aws/license-manager-user-subscriptions/docker-compose.yml"
  },
  {
    id: "docdb",
    name: "DocumentDB",
    category: "database",
    summary: "Run DocumentDB clusters, instances, cluster parameter groups, cluster snapshots and global clusters on the shared RDS control plane.",
    docsHref: "../../../index.html#docdb",
    exampleHref: "../../../../examples/aws/docdb/docker-compose.yml"
  },
  {
    id: "docdbelastic",
    name: "DocumentDB Elastic Clusters",
    category: "database",
    summary: "Run sharded DocumentDB elastic clusters with settle-on-read lifecycles, manual and automated snapshots, copy and restore, maintenance opt-in and tags.",
    docsHref: "../../../index.html#docdbelastic",
    exampleHref: "../../../../examples/aws/docdb-elastic/docker-compose.yml"
  },
  {
    id: "neptune",
    name: "Neptune",
    category: "database",
    summary: "Run graph database control-plane workflows over Neptune query APIs.",
    docsHref: "../../../index.html#neptune",
    exampleHref: "../../../../examples/aws/neptune/docker-compose.yml"
  },
  {
    id: "omics",
    name: "HealthOmics",
    category: "integration",
    summary: "Plan and validate genomic stores, workflows, runs, import/export jobs, multipart uploads, and sharing APIs.",
    docsHref: "../../../index.html#omics",
    exampleHref: "../../../../examples/aws/omics/docker-compose.yml"
  },
  {
    id: "health",
    name: "AWS Health",
    category: "integration",
    summary: "Plan staged emulation for account and organization health events, entities, aggregates, and service-access APIs.",
    docsHref: "../../../index.html#health",
    exampleHref: "../../../../examples/aws/health/docker-compose.yml"
  },
  {
    id: "grafana",
    name: "Managed Grafana",
    category: "integration",
    summary: "Emulate workspace lifecycle, auth/config, service accounts, permissions, licenses, versions, and tagging APIs.",
    docsHref: "../../../index.html#grafana",
    exampleHref: "../../../../examples/aws/grafana/docker-compose.yml"
  },
  {
    id: "prometheus",
    name: "Managed Prometheus",
    category: "integration",
    summary: "Plan staged emulation for workspaces, rule groups, Alertmanager, scrapers, anomaly detectors, logging, and tagging APIs.",
    docsHref: "../../../index.html#prometheus",
    exampleHref: "../../../../examples/aws/prometheus/docker-compose.yml"
  },
  {
    id: "mpa",
    name: "Multi-party Approval",
    category: "security",
    summary: "Plan staged emulation for approval teams, sessions, identity sources, policies, resource policies, and tagging APIs.",
    docsHref: "../../../index.html#mpa",
    exampleHref: "../../../../examples/aws/mpa/docker-compose.yml"
  },
  {
    id: "proton",
    name: "Proton",
    category: "integration",
    summary: "Plan staged emulation for environments, services, templates, repositories, deployments, sync workflows, and tagging APIs.",
    docsHref: "../../../index.html#proton",
    exampleHref: "../../../../examples/aws/proton/docker-compose.yml"
  },
  {
    id: "healthlake",
    name: "HealthLake",
    category: "integration",
    summary: "Plan and validate FHIR datastore lifecycle, import/export jobs, and resource tagging APIs.",
    docsHref: "../../../index.html#healthlake",
    exampleHref: "../../../../examples/aws/healthlake/docker-compose.yml"
  },
  {
    id: "healthimaging",
    name: "HealthImaging",
    category: "integration",
    summary: "Plan and validate medical image datastores, DICOM import workflows, image set lifecycle, and tagging APIs.",
    docsHref: "../../../index.html#healthimaging",
    exampleHref: "../../../../examples/aws/healthimaging/docker-compose.yml"
  },
  {
    id: "neptuneanalytics",
    name: "Neptune Analytics API",
    category: "database",
    summary: "Emulate Neptune Analytics graph lifecycle, query, import/export, and tagging APIs (stages 0-6).",
    docsHref: "../../../index.html#neptuneanalytics",
    exampleHref: "../../../../examples/aws/neptuneanalytics/docker-compose.yml"
  },
  {
    id: "neptunedata",
    name: "Neptune Data API",
    category: "database",
    summary: "Exercise graph data-plane requests for Gremlin, openCypher, loader, statistics/streams, ML workflows, and fast reset APIs (stages 0-6).",
    docsHref: "../../../index.html#neptunedata",
    exampleHref: "../../../../examples/aws/neptunedata/docker-compose.yml"
  },
  {
    id: "opensearch",
    name: "OpenSearch",
    category: "database",
    summary: "Model domain provisioning and service-level OpenSearch controls.",
    docsHref: "../../../index.html#opensearch",
    exampleHref: "../../../../examples/aws/opensearch/docker-compose.yml"
  },
  {
    id: "opensearchserverless",
    name: "OpenSearch Serverless",
    category: "database",
    summary: "Serverless collections gated by encryption policies, network, data access and retention policies, security configs, VPC endpoints, indexes and ingestion.",
    docsHref: "../../../index.html#opensearchserverless",
    exampleHref: "../../../../examples/aws/opensearch-serverless/docker-compose.yml"
  },
  {
    id: "osis",
    name: "OpenSearch Ingestion",
    category: "database",
    summary: "Manage OpenSearch Ingestion pipelines with checked Data Prepper configurations, settle-on-read lifecycles, VPC endpoints, change progress, blueprints, resource policies and tags.",
    docsHref: "../../../index.html#osis",
    exampleHref: "../../../../examples/aws/osis/docker-compose.yml"
  },
  {
    id: "elasticsearchservice",
    name: "Elasticsearch Service",
    category: "database",
    summary: "Legacy es 2015-01-01 API over the OpenSearch domain store: domains, config, upgrades, packages, VPC endpoints and cross-cluster connections.",
    docsHref: "../../../index.html#elasticsearchservice",
    exampleHref: "../../../../examples/aws/elasticsearch/docker-compose.yml"
  },
  {
    id: "privateca",
    name: "Private CA",
    category: "security",
    summary: "Operate private certificate authorities and issuance workflows.",
    docsHref: "../../../index.html#privateca",
    exampleHref: "../../../../examples/aws/privateca/docker-compose.yml"
  },
  {
    id: "pcaconnectorad",
    name: "Private CA Connector for Active Directory",
    category: "security",
    summary: "Emulate AD connectors, directory registrations, service principal names, certificate templates and group access control entries backed by Private CA.",
    docsHref: "../../../index.html#pcaconnectorad",
    exampleHref: "../../../../examples/aws/pca-connector-ad/docker-compose.yml"
  },
  {
    id: "pcaconnectorscep",
    name: "Private CA Connector for SCEP",
    category: "security",
    summary: "Issue SCEP connectors on ACTIVE Private CAs, with challenge passwords and Intune configurations.",
    docsHref: "../../../index.html#pcaconnectorscep",
    exampleHref: "../../../../examples/aws/pca-connector-scep/docker-compose.yml"
  },
  {
    id: "rds",
    name: "RDS",
    category: "database",
    summary: "Automate relational database instance and cluster management.",
    docsHref: "../../../index.html#rds",
    exampleHref: "../../../../examples/aws/rds/docker-compose.yml"
  },
  {
    id: "redshift",
    name: "Redshift",
    category: "database",
    summary: "Validate warehouse lifecycle and metadata operations.",
    docsHref: "../../../index.html#redshift",
    exampleHref: "../../../../examples/aws/redshift/docker-compose.yml"
  },
  {
    id: "redshift-data",
    name: "Redshift Data API",
    category: "database",
    summary: "Run SQL statements against Redshift clusters and Serverless workgroups over HTTP.",
    docsHref: "../../../index.html#redshift-data",
    exampleHref: "../../../../examples/aws/redshift-data/docker-compose.yml"
  },
  {
    id: "ram",
    name: "RAM",
    category: "integration",
    summary: "Plan staged emulation for resource shares, invitations, permissions, and association workflows.",
    docsHref: "../../../index.html#ram",
    exampleHref: "../../../../examples/aws/ram/docker-compose.yml"
  },
  {
    id: "resource-groups",
    name: "Resource Groups",
    category: "integration",
    summary: "Plan staged emulation for group lifecycle, group queries, resource membership, and tagging APIs.",
    docsHref: "../../../index.html#resource-groups",
    exampleHref: "../../../../examples/aws/resource-groups/docker-compose.yml"
  },
  {
    id: "resourcegroupstaggingapi",
    name: "Resource Groups Tagging API",
    category: "integration",
    summary: "Plan staged emulation for tag discovery, compliance reporting, and bulk tag mutation APIs.",
    docsHref: "../../../index.html#resourcegroupstaggingapi",
    exampleHref: "../../../../examples/aws/resourcegroupstagging/docker-compose.yml"
  },
  {
    id: "resource-explorer",
    name: "Resource Explorer",
    category: "integration",
    summary: "Manage index/view lifecycle, search, listing, and tagging APIs with local emulation.",
    docsHref: "../../../index.html#resource-explorer",
    exampleHref: "../../../../examples/aws/resource-explorer/docker-compose.yml"
  },
  {
    id: "resilience-hub",
    name: "Resilience Hub",
    category: "integration",
    summary: "Plan staged emulation for app lifecycle, resiliency policy workflows, assessments, recommendations, and tagging APIs.",
    docsHref: "../../../index.html#resilience-hub",
    exampleHref: "../../../../examples/aws/resilience-hub/docker-compose.yml"
  },
  {
    id: "datasync",
    name: "AWS DataSync",
    category: "storage",
    summary: "Emulate DataSync agents, every location type, tasks with filters and schedules, and executions that really copy S3-to-S3 objects.",
    docsHref: "../../../index.html#datasync",
    exampleHref: "../../../../examples/aws/datasync/docker-compose.yml"
  },
  {
    id: "efs",
    name: "Elastic File System",
    category: "storage",
    summary: "Emulate EFS file systems, mount targets in EC2 subnets, access points, policies, lifecycle, backup and replication.",
    docsHref: "../../../index.html#efs",
    exampleHref: "../../../../examples/aws/efs/docker-compose.yml"
  },
  {
    id: "glacier",
    name: "S3 Glacier",
    category: "storage",
    summary: "Emulate Glacier vaults, tree-hash-checked archive and multipart uploads, retrieval and inventory jobs, vault locks, policies and notifications.",
    docsHref: "../../../index.html#glacier",
    exampleHref: "../../../../examples/aws/glacier/docker-compose.yml"
  },
  {
    id: "backup",
    name: "AWS Backup",
    category: "storage",
    summary: "Emulate backup vaults (standard, logically air-gapped, restore access) with locks and policies, backup plans and selections, backup/copy/restore/scan jobs over real DynamoDB tables, legal holds, audit frameworks and reports, and restore testing.",
    docsHref: "../../../index.html#backup",
    exampleHref: "../../../../examples/aws/backup/docker-compose.yml"
  },
  {
    id: "s3",
    name: "S3",
    category: "storage",
    summary: "Exercise bucket/object APIs with AWS-compatible signatures.",
    docsHref: "../../../index.html#s3",
    exampleHref: "../../../../examples/aws/s3/docker-compose.yml"
  },
  {
    id: "s3-control",
    name: "S3 Control",
    category: "storage",
    summary: "Configure access points, jobs, and account-level S3 controls.",
    docsHref: "../../../index.html#s3-control",
    exampleHref: null
  },
  {
    id: "s3-outposts",
    name: "S3 Outposts",
    category: "storage",
    summary: "Test outpost endpoint management and lifecycle APIs.",
    docsHref: "../../../index.html#s3-outposts",
    exampleHref: null
  },
  {
    id: "s3-tables",
    name: "S3 Tables",
    category: "storage",
    summary: "Use table-bucket and namespace APIs for tabular data layouts.",
    docsHref: "../../../index.html#s3-tables",
    exampleHref: null
  },
  {
    id: "s3-vectors",
    name: "S3 Vectors",
    category: "storage",
    summary: "Work with vector buckets, indexes, and retrieval operations.",
    docsHref: "../../../index.html#s3-vectors",
    exampleHref: null
  },
  {
    id: "secretsmanager",
    name: "Secrets Manager",
    category: "security",
    summary: "Store, rotate, and retrieve secret values during local testing.",
    docsHref: "../../../index.html#secretsmanager",
    exampleHref: "../../../../examples/aws/secretsmanager/docker-compose.yml"
  },
  {
    id: "servicecatalogappregistry",
    name: "Service Catalog AppRegistry",
    category: "integration",
    summary: "Emulate applications, attribute groups, CloudFormation stack and tag-value associations, configuration, and tagging.",
    docsHref: "../../../index.html#servicecatalogappregistry",
    exampleHref: "../../../../examples/aws/servicecatalog-appregistry/docker-compose.yml"
  },
  {
    id: "servicequotas",
    name: "Service Quotas",
    category: "integration",
    summary: "Plan staged emulation for quota discovery, quota requests, templates, auto-management, and tagging APIs.",
    docsHref: "../../../index.html#servicequotas",
    exampleHref: "../../../../examples/aws/servicequotas/docker-compose.yml"
  },
  {
    id: "support",
    name: "Support",
    category: "integration",
    summary: "Plan staged emulation for support cases, communications, attachments, create-case options, and Trusted Advisor APIs.",
    docsHref: "../../../index.html#support",
    exampleHref: "../../../../examples/aws/support/docker-compose.yml"
  },
  {
    id: "marketplace",
    name: "Marketplace",
    category: "integration",
    summary: "Plan staged emulation for catalog change sets, agreements, metering, entitlements, deployment parameters, and reporting APIs.",
    docsHref: "../../../index.html#marketplace",
    exampleHref: "../../../../examples/aws/marketplace/docker-compose.yml"
  },
  {
    id: "partner-central",
    name: "Partner Central",
    category: "integration",
    summary: "Plan staged emulation for opportunities, engagements, invitations, resource snapshots, settings, and broader Partner Central account/channel workflows.",
    docsHref: "../../../index.html#partner-central",
    exampleHref: "../../../../examples/aws/partner-central/docker-compose.yml"
  },
  {
    id: "m2",
    name: "Mainframe Modernization (M2)",
    category: "integration",
    summary: "Plan staged emulation for application, environment, deployment, dataset import/export, and batch-job lifecycle APIs.",
    docsHref: "../../../index.html#m2",
    exampleHref: "../../../../examples/aws/m2/docker-compose.yml"
  },
  {
    id: "ivs-lowlatency",
    name: "Amazon IVS (Low Latency)",
    category: "integration",
    summary: "Plan staged emulation for channels, stream keys, recording configurations, playback key pairs/policies, stream runtime/session, and tagging APIs.",
    docsHref: "../../../index.html#ivs-lowlatency",
    exampleHref: "../../../../examples/aws/ivs-lowlatency/docker-compose.yml"
  },
  {
    id: "ivs-multitrack",
    name: "Amazon IVS Multitrack",
    category: "integration",
    summary: "Plan staged emulation for multitrack ingest discovery and client configuration APIs used by broadcast integrations.",
    docsHref: "../../../index.html#ivs-multitrack",
    exampleHref: "../../../../examples/aws/ivs-multitrack/docker-compose.yml"
  },
  {
    id: "ivs-realtimesteraming",
    name: "Amazon IVS Real-Time Streaming",
    category: "integration",
    summary: "Plan staged emulation for stage/session orchestration, participant workflows, compositions, configuration APIs, and tagging.",
    docsHref: "../../../index.html#ivs-realtimesteraming",
    exampleHref: "../../../../examples/aws/ivs-realtimesteraming/docker-compose.yml"
  },
  {
    id: "ivs-chat",
    name: "Amazon IVS Chat",
    category: "integration",
    summary: "Plan staged emulation for chat rooms, chat tokens, moderation controls, logging configurations, and tagging APIs.",
    docsHref: "../../../index.html#ivs-chat",
    exampleHref: "../../../../examples/aws/ivs-chat/docker-compose.yml"
  },
  {
    id: "ivs-chatmessaging",
    name: "Amazon IVS Chat Messaging",
    category: "integration",
    summary: "Plan staged emulation for message publishing, moderation delete/disconnect APIs, and message/event schema parity.",
    docsHref: "../../../index.html#ivs-chatmessaging",
    exampleHref: "../../../../examples/aws/ivs-chatmessaging/docker-compose.yml"
  },

  {
    id: "mediapackage",
    name: "Elemental MediaPackage V2",
    category: "integration",
    summary: "Plan staged emulation for channel groups, channels, origin endpoints, harvest jobs, policy controls, resets, and tagging APIs.",
    docsHref: "../../../index.html#mediapackage",
    exampleHref: "../../../../examples/aws/mediapackage/docker-compose.yml"
  },
  {
    id: "mediastore",
    name: "Elemental MediaStore",
    category: "integration",
    summary: "Containers with endpoints, container/CORS/lifecycle/metric policies, access logging and tags.",
    docsHref: "../../../index.html#mediastore",
    exampleHref: "../../../../examples/aws/mediastore/docker-compose.yml"
  },
  {
    id: "mediastoredata",
    name: "Elemental MediaStore Data",
    category: "integration",
    summary: "Store and serve objects in MediaStore containers with byte ranges and folder listings.",
    docsHref: "../../../index.html#mediastoredata",
    exampleHref: "../../../../examples/aws/mediastore-data/docker-compose.yml"
  },
  {
    id: "mediapackagevod",
    name: "Elemental MediaPackage VOD",
    category: "integration",
    summary: "Packaging groups, CMAF/DASH/HLS/MSS packaging configurations and S3-sourced assets with egress endpoints.",
    docsHref: "../../../index.html#mediapackagevod",
    exampleHref: "../../../../examples/aws/mediapackage-vod/docker-compose.yml"
  },
  {
    id: "mediaconnect",
    name: "Elemental MediaConnect",
    category: "integration",
    summary: "Plan staged emulation for flows, bridges, gateways, router inputs/outputs/interfaces, entitlements, reservations, and tagging APIs.",
    docsHref: "../../../index.html#mediaconnect",
    exampleHref: "../../../../examples/aws/mediaconnect/docker-compose.yml"
  },
  {
    id: "ground-station",
    name: "Ground Station",
    category: "integration",
    summary: "Plan staged emulation for configs, endpoint groups, ephemerides, mission profiles, contacts, agents, and tagging APIs.",
    docsHref: "../../../index.html#ground-station",
    exampleHref: "../../../../examples/aws/ground-station/docker-compose.yml"
  },
  {
    id: "mediatailor",
    name: "Elemental MediaTailor",
    category: "integration",
    summary: "Plan staged emulation for channels, programs, source locations, live and VOD sources, playback configs, prefetch schedules, and tagging APIs.",
    docsHref: "../../../index.html#mediatailor",
    exampleHref: "../../../../examples/aws/mediatailor/docker-compose.yml"
  },
  {
    id: "deadline-cloud",
    name: "Deadline Cloud",
    category: "integration",
    summary: "Plan staged emulation for farms, fleets, queues, workers, jobs, sessions, limits, associations, and tagging APIs.",
    docsHref: "../../../index.html#deadline-cloud",
    exampleHref: "../../../../examples/aws/deadline-cloud/docker-compose.yml"
  },
  {
    id: "migrationhub-refactor-spaces",
    name: "Migration Hub Refactor Spaces",
    category: "integration",
    summary: "Plan staged emulation for environment/application/service/route lifecycle, resource policies, VPC mappings, and tagging APIs.",
    docsHref: "../../../index.html#migrationhub-refactor-spaces",
    exampleHref: "../../../../examples/aws/migrationhub-refactor-spaces/docker-compose.yml"
  },
  {
    id: "migrationhub",
    name: "Migration Hub",
    category: "integration",
    summary: "Track migrations with progress update streams, migration task state and history, created artifacts, discovered and source resources, and application states.",
    docsHref: "../../../index.html#migrationhub",
    exampleHref: "../../../../examples/aws/migrationhub/docker-compose.yml"
  },
  {
    id: "migrationhub-orchestrator",
    name: "Migration Hub Orchestrator",
    category: "integration",
    summary: "Plan staged emulation for migration templates, workflows, step groups, steps, retry/start/stop controls, plugin discovery, and tagging APIs.",
    docsHref: "../../../index.html#migrationhub-orchestrator",
    exampleHref: "../../../../examples/aws/migrationhub-orchestrator/docker-compose.yml"
  },
  {
    id: "migrationhub-strategy",
    name: "Migration Hub Strategy Recommendations",
    category: "integration",
    summary: "Plan staged emulation for assessment orchestration, portfolio preferences, application/server insights, recommendation reports, and collector/import workflows.",
    docsHref: "../../../index.html#migrationhub-strategy",
    exampleHref: "../../../../examples/aws/migrationhub-strategy/docker-compose.yml"
  },
  {
    id: "mgn",
    name: "Application Migration Service (MGN)",
    category: "integration",
    summary: "Plan staged emulation for source server lifecycle, replication controls, application/wave grouping, import/export workflows, and cutover orchestration.",
    docsHref: "../../../index.html#mgn",
    exampleHref: "../../../../examples/aws/mgn/docker-compose.yml"
  },
  {
    id: "supportapp",
    name: "Support App in Slack",
    category: "integration",
    summary: "Plan staged emulation for Slack workspace registration, channel configuration, and account alias workflows.",
    docsHref: "../../../index.html#supportapp",
    exampleHref: "../../../../examples/aws/supportapp/docker-compose.yml"
  },
  {
    id: "trustedadvisor",
    name: "Trusted Advisor",
    category: "integration",
    summary: "Plan staged emulation for recommendation lifecycle, resource exclusions, checks, and organization recommendation workflows.",
    docsHref: "../../../index.html#trustedadvisor",
    exampleHref: "../../../../examples/aws/trustedadvisor/docker-compose.yml"
  },
  {
    id: "usernotifications",
    name: "User Notifications",
    category: "integration",
    summary: "Plan staged emulation for notification hubs, configurations, events, managed notifications, channels, organizational units, and tagging APIs.",
    docsHref: "../../../index.html#usernotifications",
    exampleHref: "../../../../examples/aws/usernotifications/docker-compose.yml"
  },
  {
    id: "systems-manager",
    name: "Systems Manager",
    category: "integration",
    summary: "Plan staged emulation for parameter store, automation, command execution, OpsCenter, session, and maintenance APIs.",
    docsHref: "../../../index.html#systems-manager",
    exampleHref: "../../../../examples/aws/systems-manager/docker-compose.yml"
  },
  {
    id: "ssmsap",
    name: "Systems Manager for SAP",
    category: "integration",
    summary: "Plan staged emulation for SAP application lifecycle, configuration checks, operation telemetry, permission, and tagging APIs.",
    docsHref: "../../../index.html#ssmsap",
    exampleHref: "../../../../examples/aws/ssmsap/docker-compose.yml"
  },
  {
    id: "quick-setup",
    name: "Systems Manager Quick Setup",
    category: "integration",
    summary: "Plan staged emulation for configuration managers, configuration definitions, service settings, type discovery, and tagging APIs.",
    docsHref: "../../../index.html#quick-setup",
    exampleHref: "../../../../examples/aws/quick-setup/docker-compose.yml"
  },
  {
    id: "incident-manager",
    name: "Systems Manager Incident Manager",
    category: "integration",
    summary: "Plan staged emulation for incident records, response plans, timeline events, contacts, engagements, and on-call rotations.",
    docsHref: "../../../index.html#incident-manager",
    exampleHref: "../../../../examples/aws/incident-manager/docker-compose.yml"
  },
  {
    id: "ssm-guiconnect",
    name: "Systems Manager GUI Connect",
    category: "integration",
    summary: "Plan staged emulation for connection recording preference lifecycle APIs.",
    docsHref: "../../../index.html#ssm-guiconnect",
    exampleHref: "../../../../examples/aws/ssm-guiconnect/docker-compose.yml"
  },
  {
    id: "ses",
    name: "SES",
    category: "integration",
    summary: "Run email identity and sending flows in a local AWS environment.",
    docsHref: "../../../index.html#ses",
    exampleHref: "../../../../examples/aws/ses/docker-compose.yml"
  },
  {
    id: "ses-v2",
    name: "SESv2",
    category: "integration",
    summary: "Use the newer SESv2 API surface for account-level messaging controls.",
    docsHref: "../../../index.html#ses-v2",
    exampleHref: "../../../../examples/aws/sesv2/docker-compose.yml"
  },
  {
    id: "signer",
    name: "Signer",
    category: "security",
    summary: "Manage code-signing profiles and signing jobs.",
    docsHref: "../../../index.html#signer",
    exampleHref: "../../../../examples/aws/signer/docker-compose.yml"
  },
  {
    id: "verifiedpermissions",
    name: "Verified Permissions",
    category: "security",
    summary: "Cedar policy stores with schemas, STRICT validation, static and template-linked policies, aliases, identity sources and real IsAuthorized decisions.",
    docsHref: "../../../index.html#verifiedpermissions",
    exampleHref: "../../../../examples/aws/verifiedpermissions/docker-compose.yml"
  },
  {
    id: "sns",
    name: "SNS",
    category: "integration",
    summary: "Create topics, subscriptions, and publish notifications.",
    docsHref: "../../../index.html#sns",
    exampleHref: "../../../../examples/aws/sns/docker-compose.yml"
  },
  {
    id: "sqs",
    name: "SQS",
    category: "integration",
    summary: "Queue messages with JSON and query-protocol compatible endpoints.",
    docsHref: "../../../index.html#sqs-json",
    exampleHref: "../../../../examples/aws/sqs/docker-compose.yml"
  },
  {
    id: "swf",
    name: "SWF",
    category: "integration",
    summary: "Emulate workflow domains, activity tasks, and decision flows.",
    docsHref: "../../../index.html#swf",
    exampleHref: "../../../../examples/aws/swf/docker-compose.yml"
  },
  {
    id: "timestream-influxdb",
    name: "Timestream for InfluxDB",
    category: "database",
    summary: "Exercise DB cluster/instance control plane APIs for InfluxDB on Timestream.",
    docsHref: "../../../index.html#timestream-influxdb",
    exampleHref: "../../../../examples/aws/timestream/docker-compose.yml"
  },
  {
    id: "timestream-write",
    name: "Timestream Write",
    category: "database",
    summary: "Emulate Timestream for LiveAnalytics databases, tables, WriteRecords with version upserts and retention windows, and CSV batch loads from S3.",
    docsHref: "../../../index.html#timestream-write",
    exampleHref: "../../../../examples/aws/timestream-liveanalytics/docker-compose.yml"
  },
  {
    id: "timestream-query",
    name: "Timestream Query",
    category: "database",
    summary: "Run a practical Timestream SQL subset (filters, bin(), aggregates, ORDER BY, LIMIT) over written records, plus scheduled queries and account settings.",
    docsHref: "../../../index.html#timestream-query",
    exampleHref: "../../../../examples/aws/timestream-liveanalytics/docker-compose.yml"
  }
];

window.AWS_CATEGORY_CONFIG = {
  all: {
    label: "All",
    description: "Complete Stackyard AWS service catalog."
  },
  compute: {
    label: "Compute & Containers",
    description: "Instances, container platforms, and application runtimes."
  },
  storage: {
    label: "Storage",
    description: "Object, table, vector, and outpost-oriented storage APIs."
  },
  database: {
    label: "Databases & Analytics",
    description: "SQL, NoSQL, search, and query service coverage."
  },
  security: {
    label: "Security & Identity",
    description: "Certificates, keys, signing, and secret-management services."
  },
  integration: {
    label: "Integration & Messaging",
    description: "Events, queues, topics, email, and workflow orchestration."
  }
};

window.AWS_CATEGORY_ORDER = ["compute", "storage", "database", "security", "integration"];
