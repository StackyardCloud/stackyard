const SERVICE_CATALOG = [
  {
    id: "acm",
    name: "ACM",
    category: "security",
    summary: "Issue, import, and inspect certificates with AWS-compatible ACM APIs.",
    docsHref: "../../../index.html#acm",
    exampleHref: "../../../../examples/aws/acm/acm/docker-compose.yml"
  },
  {
    id: "apigateway",
    name: "API Gateway",
    category: "integration",
    summary: "Plan staged emulation for REST APIs, resources, methods, integrations, deployments, stages, usage plans, domain names, and tagging APIs.",
    docsHref: "../../../index.html#apigateway",
    exampleHref: "../../../../examples/aws/apigateway/apigateway/docker-compose.yml"
  },
  {
    id: "apigatewayv2",
    name: "API Gateway v2",
    category: "integration",
    summary: "Plan staged emulation for HTTP/WebSocket APIs, routes, integrations, stages, mappings, portals, and tagging APIs.",
    docsHref: "../../../index.html#apigatewayv2",
    exampleHref: "../../../../examples/aws/apigatewayv2/apigatewayv2/docker-compose.yml"
  },
  {
    id: "appconfig",
    name: "AppConfig",
    category: "integration",
    summary: "Plan and validate application, environment, deployment, extension, and hosted-config control plane APIs.",
    docsHref: "../../../index.html#appconfig",
    exampleHref: "../../../../examples/aws/appconfig/appconfig/docker-compose.yml"
  },
  {
    id: "athena",
    name: "Athena",
    category: "database",
    summary: "Query S3-backed datasets with familiar Athena request flows.",
    docsHref: null,
    exampleHref: "../../../../examples/aws/athena/athena/docker-compose.yml"
  },
  {
    id: "aurora-dsql",
    name: "Aurora DSQL",
    category: "database",
    summary: "Use Aurora DSQL endpoints for distributed SQL integration testing.",
    docsHref: "../../../index.html#aurora-dsql",
    exampleHref: "../../../../examples/aws/aurora-dsql/aurora-dsql/docker-compose.yml"
  },
  {
    id: "augmented-ai",
    name: "Amazon Augmented AI",
    category: "integration",
    summary: "Plan and validate human loop runtime workflows for start, stop, describe, list, and delete APIs.",
    docsHref: "../../../index.html#augmented-ai",
    exampleHref: "../../../../examples/aws/augmented-ai/augmented-ai/docker-compose.yml"
  },
  {
    id: "batch",
    name: "Batch",
    category: "compute",
    summary: "Model job queues, job definitions, and execution lifecycles.",
    docsHref: "../../../index.html#batch",
    exampleHref: "../../../../examples/aws/batch/batch/docker-compose.yml"
  },
  {
    id: "bedrock",
    name: "Bedrock",
    category: "integration",
    summary: "Emulate foundation model catalog, guardrails, jobs, and provisioning control-plane APIs.",
    docsHref: "../../../index.html#bedrock",
    exampleHref: "../../../../examples/aws/bedrock/bedrock/docker-compose.yml"
  },
  {
    id: "cloudcontrolapi",
    name: "Cloud Control API",
    category: "integration",
    summary: "Emulate Cloud Control resource CRUD, progress tracking, and request status flows.",
    docsHref: "../../../index.html#cloudcontrolapi",
    exampleHref: "../../../../examples/aws/cloudcontrolapi/cloudcontrolapi/docker-compose.yml"
  },
  {
    id: "controlcatalog",
    name: "Control Catalog",
    category: "integration",
    summary: "Plan staged coverage for control catalog, domains, objectives, and mapping-discovery APIs.",
    docsHref: "../../../index.html#controlcatalog",
    exampleHref: "../../../../examples/aws/controlcatalog/controlcatalog/docker-compose.yml"
  },
  {
    id: "controltower",
    name: "Control Tower",
    category: "integration",
    summary: "Plan staged coverage for landing zone, baseline, control, operation-history, and tagging APIs.",
    docsHref: "../../../index.html#controltower",
    exampleHref: "../../../../examples/aws/controltower/controltower/docker-compose.yml"
  },
  {
    id: "cloudformation",
    name: "CloudFormation",
    category: "integration",
    summary: "Plan and validate stack lifecycle, change sets, stack sets, registry types, and drift workflows.",
    docsHref: "../../../index.html#cloudformation",
    exampleHref: "../../../../examples/aws/cloudformation/cloudformation/docker-compose.yml"
  },
  {
    id: "cloud-map",
    name: "Cloud Map",
    category: "integration",
    summary: "Plan staged coverage for namespace/service lifecycle, instance registration/discovery, operations tracking, and tagging APIs.",
    docsHref: "../../../index.html#cloud-map",
    exampleHref: "../../../../examples/aws/cloud-map/cloud-map/docker-compose.yml"
  },
  {
    id: "cloudfront",
    name: "CloudFront",
    category: "integration",
    summary: "Plan staged coverage for distributions, policies, security primitives, invalidations, edge compute, and tagging APIs.",
    docsHref: "../../../index.html#cloudfront",
    exampleHref: "../../../../examples/aws/cloudfront/cloudfront/docker-compose.yml"
  },
  {
    id: "directconnect",
    name: "Direct Connect",
    category: "integration",
    summary: "Plan staged coverage for physical links, LAGs, virtual interfaces, direct-connect gateways, and routing/security workflows.",
    docsHref: "../../../index.html#directconnect",
    exampleHref: "../../../../examples/aws/directconnect/directconnect/docker-compose.yml"
  },
  {
    id: "config",
    name: "AWS Config",
    category: "integration",
    summary: "Plan staged coverage for recorder, rule, remediation, conformance, aggregation, and query APIs.",
    docsHref: "../../../index.html#config",
    exampleHref: "../../../../examples/aws/config/config/docker-compose.yml"
  },
  {
    id: "cloudtrail",
    name: "CloudTrail",
    category: "integration",
    summary: "Plan and validate trail lifecycle, event data stores, channels, import/query, and policy APIs.",
    docsHref: "../../../index.html#cloudtrail",
    exampleHref: "../../../../examples/aws/cloudtrail/cloudtrail/docker-compose.yml"
  },
  {
    id: "cloudwatch",
    name: "CloudWatch",
    category: "integration",
    summary: "Plan and validate alarms, metrics, dashboards, anomaly detectors, insight rules, and metric stream APIs.",
    docsHref: "../../../index.html#cloudwatch",
    exampleHref: "../../../../examples/aws/cloudwatch/cloudwatch/docker-compose.yml"
  },
  {
    id: "cloudwatchlogs",
    name: "CloudWatch Logs",
    category: "integration",
    summary: "Plan and validate log groups/streams, log event ingestion/search, query/export lifecycle, delivery, policies, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatchlogs",
    exampleHref: "../../../../examples/aws/cloudwatchlogs/cloudwatchlogs/docker-compose.yml"
  },
  {
    id: "cloudwatchrum",
    name: "CloudWatch RUM",
    category: "integration",
    summary: "Plan and validate app monitor lifecycle, RUM metric definitions, policy, event ingestion, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatchrum",
    exampleHref: "../../../../examples/aws/cloudwatchrum/cloudwatchrum/docker-compose.yml"
  },
  {
    id: "cloudwatch-observability-admin",
    name: "CloudWatch Observability Admin",
    category: "integration",
    summary: "Plan telemetry pipeline/rule lifecycle, org-level evaluation/centralization workflows, integrations, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatch-observability-admin",
    exampleHref: "../../../../examples/aws/cloudwatch-observability-admin/cloudwatch-observability-admin/docker-compose.yml"
  },
  {
    id: "oam",
    name: "CloudWatch OAM",
    category: "integration",
    summary: "Plan and validate link/sink lifecycle, sink policies, and cross-account observability tagging APIs.",
    docsHref: "../../../index.html#oam",
    exampleHref: "../../../../examples/aws/oam/oam/docker-compose.yml"
  },
  {
    id: "applicationsignals",
    name: "Application Signals",
    category: "integration",
    summary: "Plan and validate SLO, service graph, grouping, audits, exclusion windows, and tagging APIs.",
    docsHref: "../../../index.html#applicationsignals",
    exampleHref: "../../../../examples/aws/applicationsignals/applicationsignals/docker-compose.yml"
  },
  {
    id: "networkflowmonitor",
    name: "Network Flow Monitor",
    category: "integration",
    summary: "Plan and validate monitor/scope lifecycle, top-contributor queries, and tagging APIs.",
    docsHref: "../../../index.html#networkflowmonitor",
    exampleHref: "../../../../examples/aws/networkflowmonitor/networkflowmonitor/docker-compose.yml"
  },
  {
    id: "internet-monitor",
    name: "Internet Monitor",
    category: "integration",
    summary: "Plan and validate monitor lifecycle, internet/health events, queries, and tagging APIs.",
    docsHref: "../../../index.html#internet-monitor",
    exampleHref: "../../../../examples/aws/internet-monitor/internet-monitor/docker-compose.yml"
  },
  {
    id: "network-synthetic-monitor",
    name: "Network Synthetic Monitor",
    category: "integration",
    summary: "Plan and validate monitor/probe lifecycle and tagging APIs for Network Synthetic Monitor.",
    docsHref: "../../../index.html#network-synthetic-monitor",
    exampleHref: "../../../../examples/aws/network-synthetic-monitor/network-synthetic-monitor/docker-compose.yml"
  },
  {
    id: "cloudwatchsynthetics",
    name: "CloudWatch Synthetics",
    category: "integration",
    summary: "Plan and validate canary lifecycle, dry-run execution, group associations, and tagging APIs.",
    docsHref: "../../../index.html#cloudwatchsynthetics",
    exampleHref: "../../../../examples/aws/cloudwatchsynthetics/cloudwatchsynthetics/docker-compose.yml"
  },
  {
    id: "cloudwatchapplicationinsights",
    name: "CloudWatch Application Insights",
    category: "integration",
    summary: "Stage application/component/workload lifecycle, problem analysis, and log pattern APIs.",
    docsHref: "../../../index.html#cloudwatchapplicationinsights",
    exampleHref: "../../../../examples/aws/cloudwatchapplicationinsights/cloudwatchapplicationinsights/docker-compose.yml"
  },
  {
    id: "cloudwatchinvestigations",
    name: "CloudWatch Investigations",
    category: "integration",
    summary: "Plan and validate investigation-group lifecycle, policy, and tagging API workflows.",
    docsHref: "../../../index.html#cloudwatchinvestigations",
    exampleHref: "../../../../examples/aws/cloudwatchinvestigations/cloudwatchinvestigations/docker-compose.yml"
  },
  {
    id: "codeartifact",
    name: "CodeArtifact",
    category: "integration",
    summary: "Plan and validate package domain, repository, package-group, and version API workflows.",
    docsHref: "../../../index.html#codeartifact",
    exampleHref: "../../../../examples/aws/codeartifact/codeartifact/docker-compose.yml"
  },
  {
    id: "codebuild",
    name: "CodeBuild",
    category: "integration",
    summary: "Plan and validate project, build, report, fleet, sandbox, and webhook API flows.",
    docsHref: "../../../index.html#codebuild",
    exampleHref: "../../../../examples/aws/codebuild/codebuild/docker-compose.yml"
  },
  {
    id: "codecatalyst",
    name: "CodeCatalyst",
    category: "integration",
    summary: "Plan and validate spaces, projects, repositories, dev environments, workflows, and access tokens.",
    docsHref: "../../../index.html#codecatalyst",
    exampleHref: "../../../../examples/aws/codecatalyst/codecatalyst/docker-compose.yml"
  },
  {
    id: "connect",
    name: "Connect",
    category: "integration",
    summary: "Plan staged coverage for instance, user, flow, contact, workspace, search, and tagging APIs.",
    docsHref: "../../../index.html#connect",
    exampleHref: "../../../../examples/aws/connect/connect/docker-compose.yml"
  },
  {
    id: "dlm",
    name: "Data Lifecycle Manager",
    category: "integration",
    summary: "Plan staged coverage for lifecycle policy CRUD, filtering, and tagging APIs.",
    docsHref: "../../../index.html#dlm",
    exampleHref: "../../../../examples/aws/dlm/dlm/docker-compose.yml"
  },
  {
    id: "datazone",
    name: "DataZone",
    category: "integration",
    summary: "Plan staged coverage for domains, projects, environments, governance, subscriptions, lineage, and tagging APIs.",
    docsHref: "../../../index.html#datazone",
    exampleHref: "../../../../examples/aws/datazone/datazone/docker-compose.yml"
  },
  {
    id: "elasticache",
    name: "ElastiCache",
    category: "integration",
    summary: "Plan staged coverage for query actions spanning cluster, replication, serverless, migration, and tagging workflows.",
    docsHref: "../../../index.html#elasticache",
    exampleHref: "../../../../examples/aws/elasticache/elasticache/docker-compose.yml"
  },
  {
    id: "emr",
    name: "EMR",
    category: "integration",
    summary: "Plan staged coverage for cluster lifecycle, step execution, scaling controls, Studio APIs, and tagging workflows.",
    docsHref: "../../../index.html#emr",
    exampleHref: "../../../../examples/aws/emr/emr/docker-compose.yml"
  },
  {
    id: "emr-on-eks",
    name: "EMR on EKS",
    category: "integration",
    summary: "Plan staged coverage for virtual clusters, job runs, managed endpoints, job templates, and tagging APIs.",
    docsHref: "../../../index.html#emr-on-eks",
    exampleHref: "../../../../examples/aws/emr-on-eks/emr-on-eks/docker-compose.yml"
  },
  {
    id: "emr-serverless",
    name: "EMR Serverless",
    category: "integration",
    summary: "Plan staged coverage for application lifecycle, job runs, dashboard access, and tagging APIs.",
    docsHref: "../../../index.html#emr-serverless",
    exampleHref: "../../../../examples/aws/emr-serverless/emr-serverless/docker-compose.yml"
  },
  {
    id: "firehose",
    name: "Data Firehose",
    category: "integration",
    summary: "Plan staged coverage for delivery stream lifecycle, ingestion, encryption, destination updates, and tagging APIs.",
    docsHref: "../../../index.html#firehose",
    exampleHref: "../../../../examples/aws/firehose/firehose/docker-compose.yml"
  },
  {
    id: "finspace",
    name: "FinSpace",
    category: "integration",
    summary: "Plan staged coverage for datasets, changesets, data views, users, permission groups, and credentials APIs.",
    docsHref: "../../../index.html#finspace",
    exampleHref: "../../../../examples/aws/finspace/finspace/docker-compose.yml"
  },
  {
    id: "finspace-management",
    name: "FinSpace Management",
    category: "integration",
    summary: "Plan staged coverage for environments, Kx clusters/databases/dataviews, users, networking, and tagging APIs.",
    docsHref: "../../../index.html#finspace-management",
    exampleHref: "../../../../examples/aws/finspace-management/finspace-management/docker-compose.yml"
  },
  {
    id: "gameliftstreams",
    name: "GameLift Streams",
    category: "integration",
    summary: "Plan staged coverage for applications, stream groups, stream sessions, connection/export workflows, and tagging APIs.",
    docsHref: "../../../index.html#gameliftstreams",
    exampleHref: "../../../../examples/aws/gameliftstreams/gameliftstreams/docker-compose.yml"
  },
  {
    id: "kendra",
    name: "Kendra",
    category: "integration",
    summary: "Plan staged coverage for indices, data sources, FAQs, thesauri, document ingestion, query workflows, and tagging APIs.",
    docsHref: "../../../index.html#kendra",
    exampleHref: "../../../../examples/aws/kendra/kendra/docker-compose.yml"
  },
  {
    id: "kinesis",
    name: "Kinesis Data Streams",
    category: "integration",
    summary: "Plan staged coverage for stream lifecycle, shard scaling, record ingest/read APIs, consumer management, and policy/tagging workflows.",
    docsHref: "../../../index.html#kinesis",
    exampleHref: "../../../../examples/aws/kinesis/kinesis/docker-compose.yml"
  },
  {
    id: "kinesisvideostreams",
    name: "Kinesis Video Streams",
    category: "integration",
    summary: "Plan staged coverage for stream/channel lifecycle, endpoint discovery, edge workflows, retention, and tagging APIs.",
    docsHref: "../../../index.html#kinesis-video-streams",
    exampleHref: "../../../../examples/aws/kinesisvideostreams/kinesisvideostreams/docker-compose.yml"
  },
  {
    id: "mq",
    name: "Amazon MQ",
    category: "integration",
    summary: "Plan staged coverage for broker lifecycle, engine/instance options, configuration revisions, users, reboot/promote, and tagging APIs.",
    docsHref: "../../../index.html#mq",
    exampleHref: "../../../../examples/aws/mq/mq/docker-compose.yml"
  },
  {
    id: "msk",
    name: "Amazon MSK",
    category: "integration",
    summary: "Plan staged coverage for MSK API v2 cluster create/list/describe and cluster-operation read/list workflows.",
    docsHref: "../../../index.html#msk",
    exampleHref: "../../../../examples/aws/msk/msk/docker-compose.yml"
  },
  {
    id: "msk-connect",
    name: "Amazon MSK Connect",
    category: "integration",
    summary: "Plan staged coverage for connector, custom plugin, worker configuration, operation history, and tagging workflows.",
    docsHref: "../../../index.html#msk-connect",
    exampleHref: "../../../../examples/aws/msk-connect/msk-connect/docker-compose.yml"
  },
  {
    id: "mwaa",
    name: "MWAA",
    category: "integration",
    summary: "Plan staged coverage for environment lifecycle, token issuance, Airflow REST proxy, metrics ingestion, and tagging APIs.",
    docsHref: "../../../index.html#mwaa",
    exampleHref: "../../../../examples/aws/mwaa/mwaa/docker-compose.yml"
  },
  {
    id: "mwaa-serverless",
    name: "MWAA Serverless",
    category: "integration",
    summary: "Plan staged coverage for workflow lifecycle, run execution/control, task-instance reads, versioning, and tagging APIs.",
    docsHref: "../../../index.html#mwaa-serverless",
    exampleHref: "../../../../examples/aws/mwaa-serverless/mwaa-serverless/docker-compose.yml"
  },
  {
    id: "qdeveloper",
    name: "AWS Q Developer",
    category: "integration",
    summary: "Plan and validate account preferences, chat channel configurations, custom actions, and associations.",
    docsHref: "../../../index.html#qdeveloper",
    exampleHref: "../../../../examples/aws/qdeveloper/qdeveloper/docker-compose.yml"
  },
  {
    id: "codeguru",
    name: "CodeGuru Reviewer",
    category: "integration",
    summary: "Plan and validate repository associations, code reviews, recommendations, metrics, and tagging APIs.",
    docsHref: "../../../index.html#codeguru",
    exampleHref: "../../../../examples/aws/codeguru/codeguru/docker-compose.yml"
  },
  {
    id: "codeguru-profiler",
    name: "CodeGuru Profiler",
    category: "integration",
    summary: "Plan and validate profiling group, agent profile ingestion, findings, recommendation, and policy APIs.",
    docsHref: "../../../index.html#codeguru-profiler",
    exampleHref: "../../../../examples/aws/codeguru-profiler/codeguru-profiler/docker-compose.yml"
  },
  {
    id: "comprehend",
    name: "Comprehend",
    category: "integration",
    summary: "Emulate text analysis, batch jobs, model endpoints, flywheels, and tagging workflows.",
    docsHref: "../../../index.html#comprehend",
    exampleHref: "../../../../examples/aws/comprehend/comprehend/docker-compose.yml"
  },
  {
    id: "comprehend-medical",
    name: "Comprehend Medical",
    category: "integration",
    summary: "Plan and validate clinical text extraction, ontology inference, and async medical analysis job workflows.",
    docsHref: "../../../index.html#comprehend-medical",
    exampleHref: "../../../../examples/aws/comprehend-medical/comprehend-medical/docker-compose.yml"
  },
  {
    id: "devops-guru",
    name: "DevOps Guru",
    category: "integration",
    summary: "Emulate insights, anomalies, recommendations, resource collections, and integration configuration APIs.",
    docsHref: "../../../index.html#devops-guru",
    exampleHref: "../../../../examples/aws/devops-guru/devops-guru/docker-compose.yml"
  },
  {
    id: "codepipeline",
    name: "CodePipeline",
    category: "integration",
    summary: "Stage and validate pipeline, execution, action type, job, and webhook API flows.",
    docsHref: "../../../index.html#codepipeline",
    exampleHref: "../../../../examples/aws/codepipeline/codepipeline/docker-compose.yml"
  },
  {
    id: "organizations",
    name: "Organizations",
    category: "integration",
    summary: "Stage and validate organization hierarchy, policy, handshake, and delegated-admin control plane APIs.",
    docsHref: "../../../index.html#organizations",
    exampleHref: "../../../../examples/aws/organizations/organizations/docker-compose.yml"
  },
  {
    id: "codedeploy",
    name: "CodeDeploy",
    category: "integration",
    summary: "Plan and validate deployment applications, groups, targets, and revision workflows.",
    docsHref: "../../../index.html#codedeploy",
    exampleHref: "../../../../examples/aws/codedeploy/codedeploy/docker-compose.yml"
  },
  {
    id: "cloudhsm",
    name: "CloudHSM",
    category: "security",
    summary: "Test cluster, HSM, and backup orchestration flows locally.",
    docsHref: "../../../index.html#cloudhsm",
    exampleHref: "../../../../examples/aws/cloudhsm/cloudhsm/docker-compose.yml"
  },
  {
    id: "cognito-user-pools",
    name: "Cognito User Pools",
    category: "security",
    summary: "Plan and validate user pool auth, federation, and identity workflows.",
    docsHref: "../../../index.html#cognito-user-pools",
    exampleHref: "../../../../examples/aws/cognitouserpools/cognitouserpools/docker-compose.yml"
  },
  {
    id: "cognito",
    name: "Cognito Federated Identities",
    category: "security",
    summary: "Emulate identity pool, token, credential, role mapping, and tagging workflows.",
    docsHref: "../../../index.html#cognito",
    exampleHref: "../../../../examples/aws/cognito/cognito/docker-compose.yml"
  },
  {
    id: "cognitosync",
    name: "Cognito Sync",
    category: "security",
    summary: "Plan and validate identity-pool dataset sync, subscriptions, and record flows.",
    docsHref: "../../../index.html#cognitosync",
    exampleHref: "../../../../examples/aws/cognitosync/cognitosync/docker-compose.yml"
  },
  {
    id: "dynamodb",
    name: "DynamoDB",
    category: "database",
    summary: "Exercise table, item, and query-style DynamoDB operations.",
    docsHref: "../../../index.html#dynamodb",
    exampleHref: "../../../../examples/aws/dynamodb/dynamodb/docker-compose.yml"
  },
  {
    id: "ebs",
    name: "EBS",
    category: "compute",
    summary: "Simulate block storage lifecycle calls used by EC2 workflows.",
    docsHref: null,
    exampleHref: "../../../../examples/aws/ebs/ebs/docker-compose.yml"
  },
  {
    id: "ec2",
    name: "EC2",
    category: "compute",
    summary: "Coverage for instance, networking, and lifecycle operations.",
    docsHref: "../../../index.html#ec2",
    exampleHref: "../../../../examples/aws/ec2/ec2/docker-compose.yml"
  },
  {
    id: "ec2-autoscaling",
    name: "EC2 Auto Scaling",
    category: "compute",
    summary: "Plan and validate Auto Scaling group, launch configuration, policy, lifecycle, refresh, and traffic source workflows.",
    docsHref: "../../../index.html#ec2-autoscaling",
    exampleHref: "../../../../examples/aws/ec2-autoscaling/ec2-autoscaling/docker-compose.yml"
  },
  {
    id: "elasticloadbalancing",
    name: "Elastic Load Balancing (ELB)",
    category: "compute",
    summary: "Plan staged coverage for load balancers, listeners, rules, target groups, trust stores, and tagging workflows.",
    docsHref: "../../../index.html#elasticloadbalancing",
    exampleHref: "../../../../examples/aws/elasticloadbalancing/elasticloadbalancing/docker-compose.yml"
  },
  {
    id: "elasticloadbalancingv2",
    name: "Elastic Load Balancing (ELB Classic)",
    category: "compute",
    summary: "Plan staged coverage for Classic ELB load balancers, listeners, health checks, policies, and tags.",
    docsHref: "../../../index.html#elasticloadbalancingv2",
    exampleHref: "../../../../examples/aws/elasticloadbalancingv2/elasticloadbalancingv2/docker-compose.yml"
  },
  {
    id: "aws-autoscaling",
    name: "AWS Auto Scaling",
    category: "compute",
    summary: "Plan and validate scaling plan lifecycle, resource projections, and forecast datapoint workflows.",
    docsHref: "../../../index.html#aws-autoscaling",
    exampleHref: "../../../../examples/aws/aws-autoscaling/aws-autoscaling/docker-compose.yml"
  },
  {
    id: "compute-optimizer",
    name: "Compute Optimizer",
    category: "compute",
    summary: "Plan and validate enrollment, recommendation, export, and automation API workflows.",
    docsHref: "../../../index.html#compute-optimizer",
    exampleHref: "../../../../examples/aws/compute-optimizer/compute-optimizer/docker-compose.yml"
  },
  {
    id: "ecr",
    name: "ECR",
    category: "compute",
    summary: "Work with repositories, image metadata, and container registry APIs.",
    docsHref: "../../../index.html#ecr",
    exampleHref: "../../../../examples/aws/ecr/ecr/docker-compose.yml"
  },
  {
    id: "ecs",
    name: "ECS",
    category: "compute",
    summary: "Validate cluster, task definition, and service control planes.",
    docsHref: "../../../index.html#ecs",
    exampleHref: "../../../../examples/aws/ecs/ecs/docker-compose.yml"
  },
  {
    id: "eks",
    name: "EKS",
    category: "compute",
    summary: "Run Kubernetes cluster management scenarios in local integration tests.",
    docsHref: "../../../index.html#eks",
    exampleHref: "../../../../examples/aws/eks/eks/docker-compose.yml"
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
    exampleHref: "../../../../examples/aws/eventbridge/eventbridge/docker-compose.yml"
  },
  {
    id: "iot",
    name: "IoT",
    category: "integration",
    summary: "Plan and validate IoT Core thing, policy, cert, jobs, rules, audit, and fleet management APIs.",
    docsHref: "../../../index.html#iot",
    exampleHref: "../../../../examples/aws/iot/iot/docker-compose.yml"
  },
  {
    id: "iot-events",
    name: "IoT Events",
    category: "integration",
    summary: "Emulate detector model, alarm model, input, analysis, and tagging API workflows.",
    docsHref: "../../../index.html#iot-events",
    exampleHref: "../../../../examples/aws/iotevents/iotevents/docker-compose.yml"
  },
  {
    id: "iot-greengrass",
    name: "IoT Greengrass",
    category: "integration",
    summary: "Plan and validate edge deployment, component, core-device, connectivity, and tagging API workflows.",
    docsHref: "../../../index.html#iot-greengrass",
    exampleHref: "../../../../examples/aws/iot-greengrass/iot-greengrass/docker-compose.yml"
  },
  {
    id: "iot-wireless",
    name: "IoT Wireless",
    category: "integration",
    summary: "Emulate wireless device, gateway, destination, FUOTA, multicast, and positioning API workflows.",
    docsHref: "../../../index.html#iot-wireless",
    exampleHref: "../../../../examples/aws/iot-wireless/iot-wireless/docker-compose.yml"
  },
  {
    id: "iot-sitewise",
    name: "IoT SiteWise",
    category: "integration",
    summary: "Plan and validate industrial asset, model, portal, project, gateway, and time-series API workflows.",
    docsHref: "../../../index.html#iot-sitewise",
    exampleHref: "../../../../examples/aws/iot-sitewise/iot-sitewise/docker-compose.yml"
  },
  {
    id: "iot-twinmaker",
    name: "IoT TwinMaker",
    category: "integration",
    summary: "Plan and validate workspace, entity, component, scene, and sync APIs for digital twin workflows.",
    docsHref: "../../../index.html#iot-twinmaker",
    exampleHref: "../../../../examples/aws/iot-twinmaker/iot-twinmaker/docker-compose.yml"
  },
  {
    id: "keyspaces",
    name: "Keyspaces",
    category: "database",
    summary: "Use Cassandra-compatible control plane calls through AWS APIs.",
    docsHref: "../../../index.html#keyspaces",
    exampleHref: "../../../../examples/aws/keyspaces/keyspaces/docker-compose.yml"
  },
  {
    id: "kms",
    name: "KMS",
    category: "security",
    summary: "Manage keys, aliases, policies, and cryptographic requests.",
    docsHref: "../../../index.html#kms",
    exampleHref: "../../../../examples/aws/kms/kms/docker-compose.yml"
  },
  {
    id: "lambda",
    name: "Lambda",
    category: "compute",
    summary: "Test function lifecycle plus Invoke in mock or local execution mode with real zip code support.",
    docsHref: "../../../index.html#lambda",
    exampleHref: "../../../../examples/aws/lambda/lambda/docker-compose.yml"
  },
  {
    id: "lightsail",
    name: "Lightsail",
    category: "compute",
    summary: "Exercise simplified VPS and networking APIs for app environments.",
    docsHref: "../../../index.html#lightsail",
    exampleHref: "../../../../examples/aws/lightsail/lightsail/docker-compose.yml"
  },
  {
    id: "license-manager",
    name: "License Manager",
    category: "security",
    summary: "Emulate license configuration, grant, token, conversion task, asset ruleset, and reporting APIs.",
    docsHref: "../../../index.html#license-manager",
    exampleHref: "../../../../examples/aws/license-manager/license-manager/docker-compose.yml"
  },
  {
    id: "license-manager-linux-subscriptions",
    name: "License Manager Linux Subscriptions",
    category: "security",
    summary: "Emulate Linux subscription providers, inventory discovery, service settings, and resource tagging APIs.",
    docsHref: "../../../index.html#license-manager-linux-subscriptions",
    exampleHref: "../../../../examples/aws/license-manager-linux-subscriptions/license-manager-linux-subscriptions/docker-compose.yml"
  },
  {
    id: "license-manager-user-subscriptions",
    name: "License Manager User Subscriptions",
    category: "security",
    summary: "Emulate identity providers, license server endpoints, user associations, product subscriptions, and tagging APIs.",
    docsHref: "../../../index.html#license-manager-user-subscriptions",
    exampleHref: "../../../../examples/aws/license-manager-user-subscriptions/license-manager-user-subscriptions/docker-compose.yml"
  },
  {
    id: "neptune",
    name: "Neptune",
    category: "database",
    summary: "Run graph database control-plane workflows over Neptune query APIs.",
    docsHref: "../../../index.html#neptune",
    exampleHref: "../../../../examples/aws/neptune/neptune/docker-compose.yml"
  },
  {
    id: "omics",
    name: "HealthOmics",
    category: "integration",
    summary: "Plan and validate genomic stores, workflows, runs, import/export jobs, multipart uploads, and sharing APIs.",
    docsHref: "../../../index.html#omics",
    exampleHref: "../../../../examples/aws/omics/omics/docker-compose.yml"
  },
  {
    id: "health",
    name: "AWS Health",
    category: "integration",
    summary: "Plan staged emulation for account and organization health events, entities, aggregates, and service-access APIs.",
    docsHref: "../../../index.html#health",
    exampleHref: "../../../../examples/aws/health/health/docker-compose.yml"
  },
  {
    id: "grafana",
    name: "Managed Grafana",
    category: "integration",
    summary: "Emulate workspace lifecycle, auth/config, service accounts, permissions, licenses, versions, and tagging APIs.",
    docsHref: "../../../index.html#grafana",
    exampleHref: "../../../../examples/aws/grafana/grafana/docker-compose.yml"
  },
  {
    id: "prometheus",
    name: "Managed Prometheus",
    category: "integration",
    summary: "Plan staged emulation for workspaces, rule groups, Alertmanager, scrapers, anomaly detectors, logging, and tagging APIs.",
    docsHref: "../../../index.html#prometheus",
    exampleHref: "../../../../examples/aws/prometheus/prometheus/docker-compose.yml"
  },
  {
    id: "mpa",
    name: "Multi-party Approval",
    category: "security",
    summary: "Plan staged emulation for approval teams, sessions, identity sources, policies, resource policies, and tagging APIs.",
    docsHref: "../../../index.html#mpa",
    exampleHref: "../../../../examples/aws/mpa/mpa/docker-compose.yml"
  },
  {
    id: "proton",
    name: "Proton",
    category: "integration",
    summary: "Plan staged emulation for environments, services, templates, repositories, deployments, sync workflows, and tagging APIs.",
    docsHref: "../../../index.html#proton",
    exampleHref: "../../../../examples/aws/proton/proton/docker-compose.yml"
  },
  {
    id: "healthlake",
    name: "HealthLake",
    category: "integration",
    summary: "Plan and validate FHIR datastore lifecycle, import/export jobs, and resource tagging APIs.",
    docsHref: "../../../index.html#healthlake",
    exampleHref: "../../../../examples/aws/healthlake/healthlake/docker-compose.yml"
  },
  {
    id: "healthimaging",
    name: "HealthImaging",
    category: "integration",
    summary: "Plan and validate medical image datastores, DICOM import workflows, image set lifecycle, and tagging APIs.",
    docsHref: "../../../index.html#healthimaging",
    exampleHref: "../../../../examples/aws/healthimaging/healthimaging/docker-compose.yml"
  },
  {
    id: "neptuneanalytics",
    name: "Neptune Analytics API",
    category: "database",
    summary: "Emulate Neptune Analytics graph lifecycle, query, import/export, and tagging APIs (stages 0-6).",
    docsHref: "../../../index.html#neptuneanalytics",
    exampleHref: "../../../../examples/aws/neptuneanalytics/neptuneanalytics/docker-compose.yml"
  },
  {
    id: "neptunedata",
    name: "Neptune Data API",
    category: "database",
    summary: "Exercise graph data-plane requests for Gremlin, openCypher, loader, statistics/streams, ML workflows, and fast reset APIs (stages 0-6).",
    docsHref: "../../../index.html#neptunedata",
    exampleHref: "../../../../examples/aws/neptunedata/neptunedata/docker-compose.yml"
  },
  {
    id: "opensearch",
    name: "OpenSearch",
    category: "database",
    summary: "Model domain provisioning and service-level OpenSearch controls.",
    docsHref: "../../../index.html#opensearch",
    exampleHref: "../../../../examples/aws/opensearch/opensearch/docker-compose.yml"
  },
  {
    id: "privateca",
    name: "Private CA",
    category: "security",
    summary: "Operate private certificate authorities and issuance workflows.",
    docsHref: "../../../index.html#privateca",
    exampleHref: "../../../../examples/aws/privateca/privateca/docker-compose.yml"
  },
  {
    id: "rds",
    name: "RDS",
    category: "database",
    summary: "Automate relational database instance and cluster management.",
    docsHref: "../../../index.html#rds",
    exampleHref: "../../../../examples/aws/rds/rds/docker-compose.yml"
  },
  {
    id: "redshift",
    name: "Redshift",
    category: "database",
    summary: "Validate warehouse lifecycle and metadata operations.",
    docsHref: "../../../index.html#redshift",
    exampleHref: "../../../../examples/aws/redshift/redshift/docker-compose.yml"
  },
  {
    id: "ram",
    name: "RAM",
    category: "integration",
    summary: "Plan staged emulation for resource shares, invitations, permissions, and association workflows.",
    docsHref: "../../../index.html#ram",
    exampleHref: "../../../../examples/aws/ram/ram/docker-compose.yml"
  },
  {
    id: "resource-groups",
    name: "Resource Groups",
    category: "integration",
    summary: "Plan staged emulation for group lifecycle, group queries, resource membership, and tagging APIs.",
    docsHref: "../../../index.html#resource-groups",
    exampleHref: "../../../../examples/aws/resource-groups/resource-groups/docker-compose.yml"
  },
  {
    id: "resourcegroupstaggingapi",
    name: "Resource Groups Tagging API",
    category: "integration",
    summary: "Plan staged emulation for tag discovery, compliance reporting, and bulk tag mutation APIs.",
    docsHref: "../../../index.html#resourcegroupstaggingapi",
    exampleHref: "../../../../examples/aws/resourcegroupstagging/resourcegroupstagging/docker-compose.yml"
  },
  {
    id: "resource-explorer",
    name: "Resource Explorer",
    category: "integration",
    summary: "Manage index/view lifecycle, search, listing, and tagging APIs with local emulation.",
    docsHref: "../../../index.html#resource-explorer",
    exampleHref: "../../../../examples/aws/resource-explorer/resource-explorer/docker-compose.yml"
  },
  {
    id: "resilience-hub",
    name: "Resilience Hub",
    category: "integration",
    summary: "Plan staged emulation for app lifecycle, resiliency policy workflows, assessments, recommendations, and tagging APIs.",
    docsHref: "../../../index.html#resilience-hub",
    exampleHref: "../../../../examples/aws/resilience-hub/resilience-hub/docker-compose.yml"
  },
  {
    id: "s3",
    name: "S3",
    category: "storage",
    summary: "Exercise bucket/object APIs with AWS-compatible signatures.",
    docsHref: "../../../index.html#s3",
    exampleHref: "../../../../examples/aws/s3/s3/docker-compose.yml"
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
    exampleHref: "../../../../examples/aws/secretsmanager/secretsmanager/docker-compose.yml"
  },
  {
    id: "servicequotas",
    name: "Service Quotas",
    category: "integration",
    summary: "Plan staged emulation for quota discovery, quota requests, templates, auto-management, and tagging APIs.",
    docsHref: "../../../index.html#servicequotas",
    exampleHref: "../../../../examples/aws/servicequotas/servicequotas/docker-compose.yml"
  },
  {
    id: "support",
    name: "Support",
    category: "integration",
    summary: "Plan staged emulation for support cases, communications, attachments, create-case options, and Trusted Advisor APIs.",
    docsHref: "../../../index.html#support",
    exampleHref: "../../../../examples/aws/support/support/docker-compose.yml"
  },
  {
    id: "marketplace",
    name: "Marketplace",
    category: "integration",
    summary: "Plan staged emulation for catalog change sets, agreements, metering, entitlements, deployment parameters, and reporting APIs.",
    docsHref: "../../../index.html#marketplace",
    exampleHref: "../../../../examples/aws/marketplace/marketplace/docker-compose.yml"
  },
  {
    id: "partner-central",
    name: "Partner Central",
    category: "integration",
    summary: "Plan staged emulation for opportunities, engagements, invitations, resource snapshots, settings, and broader Partner Central account/channel workflows.",
    docsHref: "../../../index.html#partner-central",
    exampleHref: "../../../../examples/aws/partner-central/partner-central/docker-compose.yml"
  },
  {
    id: "m2",
    name: "Mainframe Modernization (M2)",
    category: "integration",
    summary: "Plan staged emulation for application, environment, deployment, dataset import/export, and batch-job lifecycle APIs.",
    docsHref: "../../../index.html#m2",
    exampleHref: "../../../../examples/aws/m2/m2/docker-compose.yml"
  },
  {
    id: "ivs-lowlatency",
    name: "Amazon IVS (Low Latency)",
    category: "integration",
    summary: "Plan staged emulation for channels, stream keys, recording configurations, playback key pairs/policies, stream runtime/session, and tagging APIs.",
    docsHref: "../../../index.html#ivs-lowlatency",
    exampleHref: "../../../../examples/aws/ivs-lowlatency/ivs-lowlatency/docker-compose.yml"
  },
  {
    id: "ivs-multitrack",
    name: "Amazon IVS Multitrack",
    category: "integration",
    summary: "Plan staged emulation for multitrack ingest discovery and client configuration APIs used by broadcast integrations.",
    docsHref: "../../../index.html#ivs-multitrack",
    exampleHref: "../../../../examples/aws/ivs-multitrack/ivs-multitrack/docker-compose.yml"
  },
  {
    id: "ivs-realtimesteraming",
    name: "Amazon IVS Real-Time Streaming",
    category: "integration",
    summary: "Plan staged emulation for stage/session orchestration, participant workflows, compositions, configuration APIs, and tagging.",
    docsHref: "../../../index.html#ivs-realtimesteraming",
    exampleHref: "../../../../examples/aws/ivs-realtimesteraming/ivs-realtimesteraming/docker-compose.yml"
  },
  {
    id: "ivs-chat",
    name: "Amazon IVS Chat",
    category: "integration",
    summary: "Plan staged emulation for chat rooms, chat tokens, moderation controls, logging configurations, and tagging APIs.",
    docsHref: "../../../index.html#ivs-chat",
    exampleHref: "../../../../examples/aws/ivs-chat/ivs-chat/docker-compose.yml"
  },
  {
    id: "ivs-chatmessaging",
    name: "Amazon IVS Chat Messaging",
    category: "integration",
    summary: "Plan staged emulation for message publishing, moderation delete/disconnect APIs, and message/event schema parity.",
    docsHref: "../../../index.html#ivs-chatmessaging",
    exampleHref: "../../../../examples/aws/ivs-chatmessaging/ivs-chatmessaging/docker-compose.yml"
  },

  {
    id: "mediapackage",
    name: "Elemental MediaPackage V2",
    category: "integration",
    summary: "Plan staged emulation for channel groups, channels, origin endpoints, harvest jobs, policy controls, resets, and tagging APIs.",
    docsHref: "../../../index.html#mediapackage",
    exampleHref: "../../../../examples/aws/mediapackage/mediapackage/docker-compose.yml"
  },
  {
    id: "mediaconnect",
    name: "Elemental MediaConnect",
    category: "integration",
    summary: "Plan staged emulation for flows, bridges, gateways, router inputs/outputs/interfaces, entitlements, reservations, and tagging APIs.",
    docsHref: "../../../index.html#mediaconnect",
    exampleHref: "../../../../examples/aws/mediaconnect/mediaconnect/docker-compose.yml"
  },
  {
    id: "ground-station",
    name: "Ground Station",
    category: "integration",
    summary: "Plan staged emulation for configs, endpoint groups, ephemerides, mission profiles, contacts, agents, and tagging APIs.",
    docsHref: "../../../index.html#ground-station",
    exampleHref: "../../../../examples/aws/ground-station/ground-station/docker-compose.yml"
  },
  {
    id: "mediatailor",
    name: "Elemental MediaTailor",
    category: "integration",
    summary: "Plan staged emulation for channels, programs, source locations, live and VOD sources, playback configs, prefetch schedules, and tagging APIs.",
    docsHref: "../../../index.html#mediatailor",
    exampleHref: "../../../../examples/aws/mediatailor/mediatailor/docker-compose.yml"
  },
  {
    id: "deadline-cloud",
    name: "Deadline Cloud",
    category: "integration",
    summary: "Plan staged emulation for farms, fleets, queues, workers, jobs, sessions, limits, associations, and tagging APIs.",
    docsHref: "../../../index.html#deadline-cloud",
    exampleHref: "../../../../examples/aws/deadline-cloud/deadline-cloud/docker-compose.yml"
  },
  {
    id: "migrationhub-refactor-spaces",
    name: "Migration Hub Refactor Spaces",
    category: "integration",
    summary: "Plan staged emulation for environment/application/service/route lifecycle, resource policies, VPC mappings, and tagging APIs.",
    docsHref: "../../../index.html#migrationhub-refactor-spaces",
    exampleHref: "../../../../examples/aws/migrationhub-refactor-spaces/migrationhub-refactor-spaces/docker-compose.yml"
  },
  {
    id: "migrationhub-orchestrator",
    name: "Migration Hub Orchestrator",
    category: "integration",
    summary: "Plan staged emulation for migration templates, workflows, step groups, steps, retry/start/stop controls, plugin discovery, and tagging APIs.",
    docsHref: "../../../index.html#migrationhub-orchestrator",
    exampleHref: "../../../../examples/aws/migrationhub-orchestrator/migrationhub-orchestrator/docker-compose.yml"
  },
  {
    id: "migrationhub-strategy",
    name: "Migration Hub Strategy Recommendations",
    category: "integration",
    summary: "Plan staged emulation for assessment orchestration, portfolio preferences, application/server insights, recommendation reports, and collector/import workflows.",
    docsHref: "../../../index.html#migrationhub-strategy",
    exampleHref: "../../../../examples/aws/migrationhub-strategy/migrationhub-strategy/docker-compose.yml"
  },
  {
    id: "mgn",
    name: "Application Migration Service (MGN)",
    category: "integration",
    summary: "Plan staged emulation for source server lifecycle, replication controls, application/wave grouping, import/export workflows, and cutover orchestration.",
    docsHref: "../../../index.html#mgn",
    exampleHref: "../../../../examples/aws/mgn/mgn/docker-compose.yml"
  },
  {
    id: "supportapp",
    name: "Support App in Slack",
    category: "integration",
    summary: "Plan staged emulation for Slack workspace registration, channel configuration, and account alias workflows.",
    docsHref: "../../../index.html#supportapp",
    exampleHref: "../../../../examples/aws/supportapp/supportapp/docker-compose.yml"
  },
  {
    id: "trustedadvisor",
    name: "Trusted Advisor",
    category: "integration",
    summary: "Plan staged emulation for recommendation lifecycle, resource exclusions, checks, and organization recommendation workflows.",
    docsHref: "../../../index.html#trustedadvisor",
    exampleHref: "../../../../examples/aws/trustedadvisor/trustedadvisor/docker-compose.yml"
  },
  {
    id: "usernotifications",
    name: "User Notifications",
    category: "integration",
    summary: "Plan staged emulation for notification hubs, configurations, events, managed notifications, channels, organizational units, and tagging APIs.",
    docsHref: "../../../index.html#usernotifications",
    exampleHref: "../../../../examples/aws/usernotifications/usernotifications/docker-compose.yml"
  },
  {
    id: "systems-manager",
    name: "Systems Manager",
    category: "integration",
    summary: "Plan staged emulation for parameter store, automation, command execution, OpsCenter, session, and maintenance APIs.",
    docsHref: "../../../index.html#systems-manager",
    exampleHref: "../../../../examples/aws/systems-manager/systems-manager/docker-compose.yml"
  },
  {
    id: "ssmsap",
    name: "Systems Manager for SAP",
    category: "integration",
    summary: "Plan staged emulation for SAP application lifecycle, configuration checks, operation telemetry, permission, and tagging APIs.",
    docsHref: "../../../index.html#ssmsap",
    exampleHref: "../../../../examples/aws/ssmsap/ssmsap/docker-compose.yml"
  },
  {
    id: "quick-setup",
    name: "Systems Manager Quick Setup",
    category: "integration",
    summary: "Plan staged emulation for configuration managers, configuration definitions, service settings, type discovery, and tagging APIs.",
    docsHref: "../../../index.html#quick-setup",
    exampleHref: "../../../../examples/aws/quick-setup/quick-setup/docker-compose.yml"
  },
  {
    id: "incident-manager",
    name: "Systems Manager Incident Manager",
    category: "integration",
    summary: "Plan staged emulation for incident records, response plans, timeline events, contacts, engagements, and on-call rotations.",
    docsHref: "../../../index.html#incident-manager",
    exampleHref: "../../../../examples/aws/incident-manager/incident-manager/docker-compose.yml"
  },
  {
    id: "ssm-guiconnect",
    name: "Systems Manager GUI Connect",
    category: "integration",
    summary: "Plan staged emulation for connection recording preference lifecycle APIs.",
    docsHref: "../../../index.html#ssm-guiconnect",
    exampleHref: "../../../../examples/aws/ssm-guiconnect/ssm-guiconnect/docker-compose.yml"
  },
  {
    id: "ses",
    name: "SES",
    category: "integration",
    summary: "Run email identity and sending flows in a local AWS environment.",
    docsHref: "../../../index.html#ses",
    exampleHref: "../../../../examples/aws/ses/ses/docker-compose.yml"
  },
  {
    id: "ses-v2",
    name: "SESv2",
    category: "integration",
    summary: "Use the newer SESv2 API surface for account-level messaging controls.",
    docsHref: "../../../index.html#ses-v2",
    exampleHref: "../../../../examples/aws/sesv2/sesv2/docker-compose.yml"
  },
  {
    id: "signer",
    name: "Signer",
    category: "security",
    summary: "Manage code-signing profiles and signing jobs.",
    docsHref: "../../../index.html#signer",
    exampleHref: "../../../../examples/aws/signer/signer/docker-compose.yml"
  },
  {
    id: "sns",
    name: "SNS",
    category: "integration",
    summary: "Create topics, subscriptions, and publish notifications.",
    docsHref: "../../../index.html#sns",
    exampleHref: "../../../../examples/aws/sns/sns/docker-compose.yml"
  },
  {
    id: "sqs",
    name: "SQS",
    category: "integration",
    summary: "Queue messages with JSON and query-protocol compatible endpoints.",
    docsHref: "../../../index.html#sqs-json",
    exampleHref: "../../../../examples/aws/sqs/sqs/docker-compose.yml"
  },
  {
    id: "swf",
    name: "SWF",
    category: "integration",
    summary: "Emulate workflow domains, activity tasks, and decision flows.",
    docsHref: "../../../index.html#swf",
    exampleHref: "../../../../examples/aws/swf/swf/docker-compose.yml"
  },
  {
    id: "timestream-influxdb",
    name: "Timestream for InfluxDB",
    category: "database",
    summary: "Exercise DB cluster/instance control plane APIs for InfluxDB on Timestream.",
    docsHref: "../../../index.html#timestream-influxdb",
    exampleHref: "../../../../examples/aws/timestream/timestream/docker-compose.yml"
  }
];

const CATEGORY_CONFIG = {
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

const CATEGORY_ORDER = ["compute", "storage", "database", "security", "integration"];

function compareServices(a, b) {
  return a.name.localeCompare(b.name);
}

function categoryCount(services, category) {
  if (category === "all") {
    return services.length;
  }
  return services.filter((service) => service.category === category).length;
}

function normalize(text) {
  return text.toLowerCase().trim();
}

function matchesQuery(service, query) {
  if (!query) {
    return true;
  }

  const haystack = [
    service.name,
    service.summary,
    CATEGORY_CONFIG[service.category].label
  ]
    .join(" ")
    .toLowerCase();

  return haystack.includes(query);
}

function renderSidebarNav(services) {
  const navRoot = document.querySelector("[data-service-nav]");
  if (!navRoot) {
    return;
  }

  const links = [...services]
    .sort(compareServices)
    .map(
      (service) =>
        `<a href="#service-${service.id}" title="Jump to ${service.name}">${service.name}</a>`
    )
    .join("");

  navRoot.innerHTML = links;
}

function renderFilterChips(services, activeCategory) {
  const chipRoot = document.querySelector("[data-filter-chips]");
  if (!chipRoot) {
    return;
  }

  const categories = ["all", ...CATEGORY_ORDER];
  chipRoot.innerHTML = categories
    .map((category) => {
      const config = CATEGORY_CONFIG[category];
      const count = categoryCount(services, category);
      const activeClass = category === activeCategory ? "active" : "";
      return `<button class="filter-chip ${activeClass}" type="button" data-category="${category}">${config.label} (${count})</button>`;
    })
    .join("");
}

function renderCatalog(services, state) {
  const catalogRoot = document.querySelector("[data-services]");
  const countRoot = document.querySelector("[data-visible-count]");

  if (!catalogRoot) {
    return;
  }

  const filtered = services
    .filter((service) => matchesQuery(service, state.query))
    .filter((service) => state.category === "all" || service.category === state.category)
    .sort(compareServices);

  if (countRoot) {
    countRoot.textContent = `Showing ${filtered.length} of ${services.length} services`;
  }

  if (!filtered.length) {
    catalogRoot.innerHTML = `
      <div class="empty-state reveal">
        <strong>No matching services</strong>
        <p>Try a different keyword or switch to another category filter.</p>
      </div>
    `;
    return;
  }

  const categoriesToRender = state.category === "all" ? CATEGORY_ORDER : [state.category];

  const content = categoriesToRender
    .map((category) => {
      const categoryServices = filtered.filter((service) => service.category === category);
      if (!categoryServices.length) {
        return "";
      }

      const config = CATEGORY_CONFIG[category];

      const cards = categoryServices
        .map((service) => {
          const docsAction = service.docsHref
            ? `<a class="btn primary" href="${service.docsHref}">Implementation Details</a>`
            : `<span class="btn disabled">Docs Pending</span>`;

          const exampleAction = service.exampleHref
            ? `<a class="btn ghost" href="${service.exampleHref}">Basic Example</a>`
            : `<span class="btn disabled">No Example Yet</span>`;

          const tags = service.exampleHref
            ? `<span class="pill accent">Basic + advanced examples</span>`
            : `<span class="pill neutral">Core API coverage</span>`;

          return `
            <article class="service-card" id="service-${service.id}">
              <div class="service-head">
                <h3>${service.name}</h3>
                <span class="pill warn">Available</span>
              </div>
              <p class="service-summary">${service.summary}</p>
              <div class="service-tags">${tags}</div>
              <div class="service-actions">${docsAction}${exampleAction}</div>
            </article>
          `;
        })
        .join("");

      return `
        <section class="catalog-group reveal" id="category-${category}">
          <header class="catalog-header">
            <div>
              <h2>${config.label}</h2>
              <p>${config.description}</p>
            </div>
            <span class="pill neutral">${categoryServices.length} services</span>
          </header>
          <div class="card-grid">${cards}</div>
        </section>
      `;
    })
    .join("");

  catalogRoot.innerHTML = content;
}

function renderCatalogStats(services) {
  const totalRoot = document.querySelector("[data-total-services]");
  const examplesRoot = document.querySelector("[data-example-services]");
  const docsRoot = document.querySelector("[data-documented-services]");

  if (totalRoot) {
    totalRoot.textContent = String(services.length);
  }

  if (examplesRoot) {
    examplesRoot.textContent = String(services.filter((service) => Boolean(service.exampleHref)).length);
  }

  if (docsRoot) {
    docsRoot.textContent = String(services.filter((service) => Boolean(service.docsHref)).length);
  }
}

function setupServiceCatalog() {
  const catalogRoot = document.querySelector("[data-services]");
  if (!catalogRoot) {
    return;
  }

  const searchInput = document.querySelector("[data-service-search]");
  const chipRoot = document.querySelector("[data-filter-chips]");

  const state = {
    query: "",
    category: "all"
  };

  renderCatalogStats(SERVICE_CATALOG);
  renderSidebarNav(SERVICE_CATALOG);
  renderFilterChips(SERVICE_CATALOG, state.category);
  renderCatalog(SERVICE_CATALOG, state);

  if (searchInput) {
    searchInput.addEventListener("input", (event) => {
      state.query = normalize(event.target.value);
      renderCatalog(SERVICE_CATALOG, state);
    });
  }

  if (chipRoot) {
    chipRoot.addEventListener("click", (event) => {
      const target = event.target;
      if (!(target instanceof HTMLElement)) {
        return;
      }

      if (!target.dataset.category) {
        return;
      }

      state.category = target.dataset.category;
      renderFilterChips(SERVICE_CATALOG, state.category);
      renderCatalog(SERVICE_CATALOG, state);
    });
  }
}

document.addEventListener("DOMContentLoaded", () => {
  setupServiceCatalog();
});
