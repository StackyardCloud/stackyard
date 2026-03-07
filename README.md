# Stackyard

Stackyard is a Go-based local AWS emulator focused on fast startup, deterministic behavior, and contributor-friendly service evolution.

This repository is designed as an open reference implementation for building and testing cloud-integrated systems locally without depending on live AWS infrastructure.

[https://stackyard.cloud](https://stackyard.cloud)

## Project Goals

- Emulate AWS APIs with predictable local behavior
- Keep service implementations readable and independently evolvable
- Support SDK/CLI compatibility testing through staged service coverage
- Make it easy for contributors to add or harden service emulation

## Current Scope

Stackyard includes many service emulations with staged behavior and tests. Coverage and implementation depth vary by service.

For the most up-to-date catalog and operation/type coverage, use:

- Interactive docs: `docs/index.html`
- Coverage runner: `scripts/awscli-endpoint-coverage.py`
- GCP contract coverage runner (validation/fixtures/negative tests): `scripts/gcp-contract-coverage.py`

## Architecture Principles

- In-memory-first service stores for speed and repeatability
- Explicit request parsing and error envelopes per service protocol
- SigV4-aware routing/validation where required
- Staged implementation strategy to grow compatibility safely

## Repository Layout

- Server entrypoint: `cmd/stackyard`
- Service handlers/stores/tests: `internal/server`
- Runnable examples: `examples/aws/`
- Endpoint coverage tooling: `scripts/awscli-endpoint-coverage.py`
- GCP contract coverage tooling: `scripts/gcp-contract-coverage.py`
- GCP contract hardening roadmap: `docs/gcp-contract-hardening-plan.md`
- Generated docs site: `docs/index.html`
- Multi-cloud foundation notes: `docs/multi-cloud-foundation.md`
- GCP Generative Language staged plan: `docs/gcp-generativelanguage-apiv1-plan.md`
- GCP Vertex AI staged plan: `docs/gcp-aiplatform-apiv1-plan.md`
- GCP Gateway API staged plan: `docs/gcp-apigateway-apiv1-plan.md`
- GCP Apigee Connect staged plan: `docs/gcp-apigeeconnect-apiv1-plan.md`
- GCP API Hub staged plan: `docs/gcp-apihub-apiv1-plan.md`
- GCP API Keys staged plan: `docs/gcp-apikeys-apiv2-plan.md`
- GCP App Engine Admin staged plan: `docs/gcp-appengine-apiv1-plan.md`
- GCP Workspace Events Subscriptions staged plan: `docs/gcp-subscriptions-apiv1-plan.md`
- GCP Artifact Registry staged plan: `docs/gcp-artifactregistry-apiv1-plan.md`
- GCP AutoML staged plan: `docs/gcp-automl-apiv1-plan.md`
- GCP Google Meet staged plan: `docs/gcp-meet-apiv2-plan.md`
- GCP Google Chat staged plan: `docs/gcp-chat-apiv1-plan.md`
- GCP Chronicle staged plan: `docs/gcp-chronicle-apiv1-plan.md`
- GCP Cloud Bigtable staged plan: `docs/gcp-bigtable-apiv2-plan.md`
- GCP Certificate Manager staged plan: `docs/gcp-certificatemanager-apiv1-plan.md`
- GCP Cloud Channel staged plan: `docs/gcp-channel-apiv1-plan.md`
- GCP Cloud Controls Partner staged plan: `docs/gcp-cloudcontrolspartner-apiv1-plan.md`
- GCP Database Migration staged plan: `docs/gcp-clouddms-apiv1-plan.md`
- GCP Cloud Profiler staged plan: `docs/gcp-cloudprofiler-apiv2-plan.md`
- GCP Cloud Quotas staged plan: `docs/gcp-cloudquotas-apiv1-plan.md`
- GCP Cloud Tasks staged plan: `docs/gcp-cloudtasks-apiv2-plan.md`
- GCP Cloud Build staged plan: `docs/gcp-cloudbuild-apiv2-plan.md`
- GCP Google Compute Engine staged plan: `docs/gcp-compute-apiv1-plan.md`
- GCP Compute Metadata staged plan: `docs/gcp-compute-metadata-apiv1-plan.md`
- GCP Cloud Commerce Consumer Procurement staged plan: `docs/gcp-commerce-consumer-procurement-apiv1-plan.md`
- GCP Confidential Computing staged plan: `docs/gcp-confidentialcomputing-apiv1-plan.md`
- GCP Infrastructure Manager (Config API) staged plan: `docs/gcp-config-apiv1-plan.md`
- GCP Config Delivery staged plan: `docs/gcp-configdelivery-apiv1-plan.md`
- GCP Kubernetes Engine staged plan: `docs/gcp-container-apiv1-plan.md`
- GCP Data Catalog staged plan: `docs/gcp-datacatalog-apiv1-plan.md`
- GCP Data Lineage staged plan: `docs/gcp-datacatalog-lineage-apiv1-plan.md`
- GCP Dataflow staged plan: `docs/gcp-dataflow-apiv1beta3-plan.md`
- GCP Dataform staged plan: `docs/gcp-dataform-apiv1-plan.md`
- GCP Cloud Datastore staged plan: `docs/gcp-datastore-apiv1-plan.md`
- GCP Cloud Datastore Admin staged plan: `docs/gcp-datastore-admin-apiv1-plan.md`
- GCP Datastream staged plan: `docs/gcp-datastream-apiv1-plan.md`
- GCP Cloud Deploy staged plan: `docs/gcp-deploy-apiv1-plan.md`
- GCP Dialogflow staged plan: `docs/gcp-dialogflow-apiv2-plan.md`
- GCP Dialogflow CX staged plan: `docs/gcp-dialogflow-cx-apiv3-plan.md`
- GCP Discovery Engine staged plan: `docs/gcp-discoveryengine-apiv1-plan.md`
- GCP Cloud Document AI staged plan: `docs/gcp-documentai-apiv1-plan.md`
- GCP Sensitive Data Protection (DLP) staged plan: `docs/gcp-dlp-apiv2-plan.md`
- GCP Device Streaming staged plan: `docs/gcp-devicestreaming-apiv1-plan.md`
- GCP Developer Connect staged plan: `docs/gcp-developerconnect-apiv1-plan.md`
- GCP Data QnA staged plan: `docs/gcp-dataqna-apiv1alpha-plan.md`
- GCP Cloud Domains staged plan: `docs/gcp-domains-apiv1beta1-plan.md`
- GCP Distributed Cloud Edge Container staged plan: `docs/gcp-edgecontainer-apiv1-plan.md`
- GCP Distributed Cloud Edge Network staged plan: `docs/gcp-edgenetwork-apiv1-plan.md`
- GCP Data Fusion staged plan: `docs/gcp-datafusion-apiv1-plan.md`
- GCP Data Labeling staged plan: `docs/gcp-datalabeling-apiv1-plan.md`
- GCP Cloud Dataplex staged plan: `docs/gcp-dataplex-apiv1-plan.md`
- GCP Cloud Dataproc staged plan: `docs/gcp-dataproc-apiv1-plan.md`
- GCP Cloud Dataproc v2 staged plan: `docs/gcp-dataproc-v2-apiv1-plan.md`
- GCP Error Reporting staged plan: `docs/gcp-errorreporting-apiv1beta1-plan.md`
- GCP Essential Contacts staged plan: `docs/gcp-essentialcontacts-apiv1-plan.md`
- GCP Eventarc staged plan: `docs/gcp-eventarc-apiv1-plan.md`
- GCP Eventarc Publishing staged plan: `docs/gcp-eventarc-publishing-apiv1-plan.md`
- GCP Cloud Filestore staged plan: `docs/gcp-filestore-apiv1-plan.md`
- GCP Financial Services staged plan: `docs/gcp-financialservices-apiv1-plan.md`
- GCP Game Services staged plan: `docs/gcp-gaming-apiv1-plan.md`
- GCP Cloud Functions v1 staged plan: `docs/gcp-functions-apiv1-plan.md`
- GCP Cloud Functions v2 staged plan: `docs/gcp-functions-apiv2-plan.md`
- GCP Cloud Firestore staged plan: `docs/gcp-firestore-apiv1-plan.md`
- GCP Data Analytics with Gemini staged plan: `docs/gcp-geminidataanalytics-apiv1beta-plan.md`
- GCP GKE Backup staged plan: `docs/gcp-gkebackup-apiv1-plan.md`
- GCP Connect Gateway staged plan: `docs/gcp-gkeconnect-apiv1-plan.md`
- GCP GKE Hub staged plan: `docs/gcp-gkehub-apiv1-plan.md`
- GCP GKE Multi-Cloud staged plan: `docs/gcp-gkemulticloud-apiv1-plan.md`
- GCP Cloud IAM staged plan: `docs/gcp-iam-apiv1-plan.md`
- GCP Cloud IAM v2 staged plan: `docs/gcp-iam-apiv2-plan.md`
- GCP Cloud IAM v3 staged plan: `docs/gcp-iam-apiv3-plan.md`
- GCP Cloud IAM Admin staged plan: `docs/gcp-iam-admin-apiv1-plan.md`
- GCP Cloud IAM Credentials staged plan: `docs/gcp-iam-credentials-apiv1-plan.md`
- GCP Cloud Identity-Aware Proxy staged plan: `docs/gcp-iap-apiv1-plan.md`
- GCP Cloud IDS staged plan: `docs/gcp-ids-apiv1-plan.md`
- GCP Cloud IoT staged plan: `docs/gcp-iot-apiv1-plan.md`
- GCP Cloud Key Management Service staged plan: `docs/gcp-kms-apiv1-plan.md`
- GCP Cloud Key Management Service Inventory staged plan: `docs/gcp-kms-inventory-apiv1-plan.md`
- GCP Cloud Natural Language staged plan: `docs/gcp-language-apiv1-plan.md`
- GCP Cloud Natural Language v2 staged plan: `docs/gcp-language-apiv2-plan.md`
- GCP Cloud Location Finder staged plan: `docs/gcp-locationfinder-apiv1-plan.md`
- GCP Maps Places Aggregate staged plan: `docs/gcp-maps-areainsights-apiv1-plan.md`
- GCP Maps Address Validation staged plan: `docs/gcp-maps-addressvalidation-apiv1-plan.md`
- GCP Local Rides and Deliveries (Fleet Engine) staged plan: `docs/gcp-maps-fleetengine-apiv1-plan.md`
- GCP Last Mile Fleet Delivery Solution staged plan: `docs/gcp-maps-fleetengine-delivery-apiv1-plan.md`
- GCP Maps Places staged plan: `docs/gcp-maps-places-apiv1-plan.md`
- GCP Maps Route Optimization staged plan: `docs/gcp-maps-routeoptimization-apiv1-plan.md`
- GCP Maps Routes staged plan: `docs/gcp-maps-routing-apiv2-plan.md`
- GCP Maps Solar staged plan: `docs/gcp-maps-solar-apiv1-plan.md`
- GCP Media Translation staged plan: `docs/gcp-mediatranslation-apiv1beta1-plan.md`
- GCP Cloud Memorystore for Memcached staged plan: `docs/gcp-memcache-apiv1-plan.md`
- GCP Memorystore staged plan: `docs/gcp-memorystore-apiv1-plan.md`
- GCP Dataproc Metastore staged plan: `docs/gcp-metastore-apiv1-plan.md`
- GCP Migration Center staged plan: `docs/gcp-migrationcenter-apiv1-plan.md`
- GCP Model Armor staged plan: `docs/gcp-modelarmor-apiv1-plan.md`
- GCP Cloud Monitoring Dashboard staged plan: `docs/gcp-monitoring-dashboard-apiv1-plan.md`
- GCP Cloud Monitoring Metrics Scope staged plan: `docs/gcp-metricsscope-apiv1-plan.md`
- GCP NetApp staged plan: `docs/gcp-netapp-apiv1-plan.md`
- GCP Notebooks staged plan: `docs/gcp-notebooks-apiv1-plan.md`
- GCP Notebooks V2 staged plan: `docs/gcp-notebooks-apiv2-plan.md`
- GCP Cloud Optimization staged plan: `docs/gcp-optimization-apiv1-plan.md`
- GCP OS Config staged plan: `docs/gcp-osconfig-apiv1-plan.md`
- GCP OS Config Agent Endpoint staged plan: `docs/gcp-osconfig-agentendpoint-apiv1-plan.md`
- GCP Cloud OS Login staged plan: `docs/gcp-oslogin-apiv1-plan.md`
- GCP Parallelstore staged plan: `docs/gcp-parallelstore-apiv1-plan.md`
- GCP Parameter Manager staged plan: `docs/gcp-parametermanager-apiv1-plan.md`
- GCP Cloud Private Catalog staged plan: `docs/gcp-privatecatalog-apiv1beta1-plan.md`
- GCP Privileged Access Manager staged plan: `docs/gcp-privilegedaccessmanager-apiv1-plan.md`
- GCP Cloud Pub/Sub staged plan: `docs/gcp-pubsub-apiv1-plan.md`
- GCP Cloud Pub/Sub v2 staged plan: `docs/gcp-pubsub-v2-apiv1-plan.md`
- GCP Cloud Pub/Sub Lite staged plan: `docs/gcp-pubsublite-apiv1-plan.md`
- GCP Phishing Protection staged plan: `docs/gcp-phishingprotection-apiv1beta1-plan.md`
- GCP Policy Simulator staged plan: `docs/gcp-policysimulator-apiv1-plan.md`
- GCP Policy Troubleshooter staged plan: `docs/gcp-policytroubleshooter-apiv1-plan.md`
- GCP Policy Troubleshooter IAM staged plan: `docs/gcp-policytroubleshooter-iam-apiv3-plan.md`
- GCP Organization Policy staged plan: `docs/gcp-orgpolicy-apiv2-plan.md`
- GCP Oracle Database staged plan: `docs/gcp-oracledatabase-apiv1-plan.md`
- GCP Cloud Monitoring staged plan: `docs/gcp-monitoring-apiv3-plan.md`
- GCP Network Services staged plan: `docs/gcp-networkservices-apiv1-plan.md`
- GCP Network Connectivity staged plan: `docs/gcp-networkconnectivity-apiv1-plan.md`
- GCP Network Management staged plan: `docs/gcp-networkmanagement-apiv1-plan.md`
- GCP Network Security staged plan: `docs/gcp-networksecurity-apiv1beta1-plan.md`
- GCP Logging staged plan: `docs/gcp-logging-apiv2-plan.md`
- GCP Lustre staged plan: `docs/gcp-lustre-apiv1-plan.md`
- GCP Managed Kafka staged plan: `docs/gcp-managedkafka-apiv1-plan.md`
- GCP Managed Kafka Schema Registry staged plan: `docs/gcp-managedkafka-schemaregistry-apiv1-plan.md`
- GCP Managed Identities staged plan: `docs/gcp-managedidentities-apiv1-plan.md`
- GCP Life Sciences staged plan: `docs/gcp-lifesciences-apiv2beta-plan.md`
- GCP License Manager staged plan: `docs/gcp-licensemanager-apiv1-plan.md`
- GCP Identity Toolkit v2 staged plan: `docs/gcp-identitytoolkit-apiv2-plan.md`
- Reference architecture examples: `reference-architecture/`

## Quickstart

Install the CLI:

```bash
go install ./cmd/stackyard
```

If `stackyard` is not found after install, add your Go bin path to `PATH`.

For the current shell session:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
hash -r
which stackyard
stackyard help
```

For zsh (persist across terminal sessions):

```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

Run locally:

```bash
# starts in background (daemon mode)
stackyard start --providers aws
stackyard start --providers aws,gcp,azure,oci
stackyard start --providers aws --port 4566
stackyard start --providers aws --log-level debug

# stop the background process
stackyard stop
```

Provider support status:

- `aws`: broad emulation coverage
- `gcp`, `azure`, `oci`: foundational routing plus object storage SDK paths enabled; broader service emulation in progress

Foundational object storage routes for non-AWS providers:

- `gcp`: JSON API bucket/object create, list, and get via `/gcp/storage/v1/*` and `/gcp/upload/storage/v1/*`
- `azure`: Blob container/blob create, list, get, and head via `/azure/{account}/*`
- `oci`: Object Storage namespace, bucket, and object lifecycle via `/oci/n/*`

Planned non-AWS expansion:

- `gcp generativelanguage/apiv1`: staged implementation for model discovery and generative RPCs (`docs/gcp-generativelanguage-apiv1-plan.md`)
- `gcp geminidataanalytics/apiv1beta`: staged implementation for data agent lifecycle, conversational analytics workflows, and operation controls (`docs/gcp-geminidataanalytics-apiv1beta-plan.md`)
- `gcp aiplatform/apiv1`: staged implementation for Vertex AI resources, jobs, and prediction flows (`docs/gcp-aiplatform-apiv1-plan.md`)
- `gcp apigateway/apiv1`: staged implementation for API/config/gateway lifecycle and transport compatibility (`docs/gcp-apigateway-apiv1-plan.md`)
- `gcp apigeeconnect/apiv1`: staged implementation for connection listing and tether stream emulation (`docs/gcp-apigeeconnect-apiv1-plan.md`)
- `gcp apihub/apiv1`: staged implementation for API metadata governance resources and search flows (`docs/gcp-apihub-apiv1-plan.md`)
- `gcp apikeys/apiv2`: staged implementation for API key lifecycle, lookup, and restriction flows (`docs/gcp-apikeys-apiv2-plan.md`)
- `gcp appengine/apiv1`: staged implementation for App Engine applications, services, versions, and instances (`docs/gcp-appengine-apiv1-plan.md`)
- `gcp subscriptions/apiv1`: staged implementation for Google Workspace Events subscription lifecycle and renewal flows (`docs/gcp-subscriptions-apiv1-plan.md`)
- `gcp artifactregistry/apiv1`: staged implementation for repository/package/version/tag lifecycle and artifact discovery flows (`docs/gcp-artifactregistry-apiv1-plan.md`)
- `gcp automl/apiv1`: staged implementation for dataset/model lifecycle and prediction flows (`docs/gcp-automl-apiv1-plan.md`)
- `gcp meet/apiv2`: staged implementation for space lifecycle and conference record discovery flows (`docs/gcp-meet-apiv2-plan.md`)
- `gcp chat/apiv1`: staged implementation for spaces, messages, memberships, reactions, and read-state workflows (`docs/gcp-chat-apiv1-plan.md`)
- `gcp chronicle/apiv1`: staged implementation for detections, watchlists, reference lists, and data-access governance workflows (`docs/gcp-chronicle-apiv1-plan.md`)
- `gcp bigtable/apiv2`: staged implementation for instance/table admin and row data-plane emulation (`docs/gcp-bigtable-apiv2-plan.md`)
- `gcp certificatemanager/apiv1`: staged implementation for certificate lifecycle, map/entry routing, and trust management flows (`docs/gcp-certificatemanager-apiv1-plan.md`)
- `gcp channel/apiv1`: staged implementation for customer lifecycle, entitlement actions, and channel reports workflows (`docs/gcp-channel-apiv1-plan.md`)
- `gcp cloudcontrolspartner/apiv1`: staged implementation for customer/workload governance and monitoring violation workflows (`docs/gcp-cloudcontrolspartner-apiv1-plan.md`)
- `gcp clouddms/apiv1`: staged implementation for migration-job lifecycle, conversion workspace orchestration, and connectivity workflows (`docs/gcp-clouddms-apiv1-plan.md`)
- `gcp cloudprofiler/apiv2`: staged implementation for profile session lifecycle, offline ingestion, and profile export discovery workflows (`docs/gcp-cloudprofiler-apiv2-plan.md`)
- `gcp cloudquotas/apiv1`: staged implementation for quota observability and quota-preference lifecycle workflows (`docs/gcp-cloudquotas-apiv1-plan.md`)
- `gcp cloudtasks/apiv2`: staged implementation for queue lifecycle controls and task dispatch workflows (`docs/gcp-cloudtasks-apiv2-plan.md`)
- `gcp cloudbuild/apiv2`: staged implementation for connection/repository lifecycle, token access, and IAM workflows (`docs/gcp-cloudbuild-apiv2-plan.md`)
- `gcp compute/apiv1`: staged implementation for zone/network discovery, instance lifecycle controls, and operation polling workflows (`docs/gcp-compute-apiv1-plan.md`)
- `gcp compute/metadata`: staged implementation for metadata-server flavor semantics, instance/project metadata reads, and subscription workflows (`docs/gcp-compute-metadata-apiv1-plan.md`)
- `gcp commerce consumer procurement/apiv1`: staged implementation for order lifecycle and long-running procurement operation workflows (`docs/gcp-commerce-consumer-procurement-apiv1-plan.md`)
- `gcp confidentialcomputing/apiv1`: staged implementation for challenge issuance, attestation verification flows, and location discovery workflows (`docs/gcp-confidentialcomputing-apiv1-plan.md`)
- `gcp config/apiv1`: staged implementation for deployment and preview lifecycle workflows in Infrastructure Manager (`docs/gcp-config-apiv1-plan.md`)
- `gcp configdelivery/apiv1`: staged implementation for resource bundle, fleet package, release, variant, and rollout workflows (`docs/gcp-configdelivery-apiv1-plan.md`)
- `gcp container/apiv1`: staged implementation for GKE cluster/node-pool lifecycle, operations, and upgrade compatibility workflows (`docs/gcp-container-apiv1-plan.md`)
- `gcp datacatalog/apiv1`: staged implementation for catalog discovery, entry/tag lifecycle, and governance workflows (`docs/gcp-datacatalog-apiv1-plan.md`)
- `gcp datacatalog/lineage/apiv1`: staged implementation for process/run/event lineage lifecycle and link discovery workflows (`docs/gcp-datacatalog-lineage-apiv1-plan.md`)
- `gcp dataflow/apiv1beta3`: staged implementation for job lifecycle, metrics/messages observability, templates, and snapshots (`docs/gcp-dataflow-apiv1beta3-plan.md`)
- `gcp dataform/apiv1`: staged implementation for repository/workspace lifecycle, compilation/workflow orchestration, and config management (`docs/gcp-dataform-apiv1-plan.md`)
- `gcp datastore/apiv1`: staged implementation for entity lookup/query, transaction controls, and ID allocation workflows (`docs/gcp-datastore-apiv1-plan.md`)
- `gcp datastore/admin apiv1`: staged implementation for export/import, composite index lifecycle, and long-running operation workflows (`docs/gcp-datastore-admin-apiv1-plan.md`)
- `gcp datastream/apiv1`: staged implementation for connection profile, stream/object, private connectivity, and backfill workflows (`docs/gcp-datastream-apiv1-plan.md`)
- `gcp deploy/apiv1`: staged implementation for delivery pipeline lifecycle, rollout orchestration, job controls, and operations workflows (`docs/gcp-deploy-apiv1-plan.md`)
- `gcp dialogflow/apiv2`: staged implementation for agent, intent, session runtime, and operations workflows (`docs/gcp-dialogflow-apiv2-plan.md`)
- `gcp dialogflow/cx apiv3`: staged implementation for agent/flow lifecycle, session runtime, and operations workflows (`docs/gcp-dialogflow-cx-apiv3-plan.md`)
- `gcp discoveryengine/apiv1`: staged implementation for engine/data-store lifecycle, search/completion runtime, and conversational workflows (`docs/gcp-discoveryengine-apiv1-plan.md`)
- `gcp documentai/apiv1`: staged implementation for processor/version lifecycle, processing/review runtime, and evaluation workflows (`docs/gcp-documentai-apiv1-plan.md`)
- `gcp dlp/apiv2`: staged implementation for content inspection/transformation, template/trigger/job lifecycle, and stored-info-type workflows (`docs/gcp-dlp-apiv2-plan.md`)
- `gcp devicestreaming/apiv1`: staged implementation for device session lifecycle and direct ADB stream workflows (`docs/gcp-devicestreaming-apiv1-plan.md`)
- `gcp developerconnect/apiv1`: staged implementation for connection/resource lifecycle, OAuth/token actions, insights config, and deployment-event workflows (`docs/gcp-developerconnect-apiv1-plan.md`)
- `gcp dataqna/apiv1alpha`: staged implementation for natural-language suggestion, question execution, and user feedback workflows (`docs/gcp-dataqna-apiv1alpha-plan.md`)
- `gcp domains/apiv1beta1`: staged implementation for domain search, registration/transfer lifecycle, registration configuration, and authorization-code workflows (`docs/gcp-domains-apiv1beta1-plan.md`)
- `gcp edgecontainer/apiv1`: staged implementation for cluster/node pool lifecycle, machine and VPN workflows, credential actions, and operation controls (`docs/gcp-edgecontainer-apiv1-plan.md`)
- `gcp edgenetwork/apiv1`: staged implementation for zone initialization, network/subnet/interconnect/router lifecycle, diagnostics, and operation workflows (`docs/gcp-edgenetwork-apiv1-plan.md`)
- `gcp datafusion/apiv1`: staged implementation for version discovery and Data Fusion instance lifecycle orchestration (`docs/gcp-datafusion-apiv1-plan.md`)
- `gcp datalabeling/apiv1beta1`: staged implementation for dataset lifecycle, annotation workflows, and evaluation job orchestration (`docs/gcp-datalabeling-apiv1-plan.md`)
- `gcp dataplex/apiv1`: staged implementation for lake/zone/asset lifecycle, task/job execution, and environment/session workflows (`docs/gcp-dataplex-apiv1-plan.md`)
- `gcp dataproc/apiv1`: staged implementation for cluster/job/workflow/autoscaling/batch orchestration workflows (`docs/gcp-dataproc-apiv1-plan.md`)
- `gcp dataproc/v2 apiv1`: staged implementation for serverless batches, interactive sessions, session templates, and operations workflows (`docs/gcp-dataproc-v2-apiv1-plan.md`)
- `gcp errorreporting/apiv1beta1`: staged implementation for group/event statistics, error-group lifecycle, and error-event reporting workflows (`docs/gcp-errorreporting-apiv1beta1-plan.md`)
- `gcp essentialcontacts/apiv1`: staged implementation for contact lifecycle, hierarchy-aware contact computation, and notification test messaging workflows (`docs/gcp-essentialcontacts-apiv1-plan.md`)
- `gcp eventarc/apiv1`: staged implementation for trigger/channel/message-bus lifecycle, IAM controls, and long-running operation workflows (`docs/gcp-eventarc-apiv1-plan.md`)
- `gcp eventarc publishing/apiv1`: staged implementation for partner channel publishing, channel publish events, and message-bus publish workflows (`docs/gcp-eventarc-publishing-apiv1-plan.md`)
- `gcp filestore/apiv1`: staged implementation for instance/snapshot/backup lifecycle, replica actions, and operation controls (`docs/gcp-filestore-apiv1-plan.md`)
- `gcp financialservices/apiv1`: staged implementation for AML instance/dataset/model/engine lifecycle, prediction/backtest workflows, and governance export actions (`docs/gcp-financialservices-apiv1-plan.md`)
- `gcp gaming/apiv1`: staged implementation for realms, game server cluster/config/deployment lifecycle, rollout previews, and deployment-state workflows (`docs/gcp-gaming-apiv1-plan.md`)
- `gcp functions/apiv1`: staged implementation for function lifecycle, invocation/source-url flows, IAM controls, and operation discovery (`docs/gcp-functions-apiv1-plan.md`)
- `gcp functions/apiv2`: staged implementation for function lifecycle, runtime/source-url flows, IAM controls, and operation discovery (`docs/gcp-functions-apiv2-plan.md`)
- `gcp firestore/apiv1`: staged implementation for document lifecycle, query/transaction workflows, streaming changes, and operation controls (`docs/gcp-firestore-apiv1-plan.md`)
- `gcp gkebackup/apiv1`: staged implementation for backup and restore plan lifecycle, backup/restore execution workflows, and policy/operation controls (`docs/gcp-gkebackup-apiv1-plan.md`)
- `gcp gkeconnect/gateway/apiv1`: staged implementation for gateway credential generation, membership validation, and REST/gRPC transport parity (`docs/gcp-gkeconnect-apiv1-plan.md`)
- `gcp gkehub/apiv1beta1`: staged implementation for membership lifecycle, connect/exclusivity workflows, and IAM/operation controls (`docs/gcp-gkehub-apiv1-plan.md`)
- `gcp gkemulticloud/apiv1`: staged implementation for attached/AWS/Azure cluster lifecycle, token and manifest actions, and long-running operation controls (`docs/gcp-gkemulticloud-apiv1-plan.md`)
- `gcp iam/apiv1`: staged implementation for IAM policy get/set/test workflows and REST/gRPC transport parity (`docs/gcp-iam-apiv1-plan.md`)
- `gcp iam/apiv2`: staged implementation for deny policy lifecycle, rule validation, and REST/gRPC transport parity (`docs/gcp-iam-apiv2-plan.md`)
- `gcp iam/apiv3`: staged implementation for policy binding and principal access boundary policy lifecycle, search workflows, and REST/gRPC transport parity (`docs/gcp-iam-apiv3-plan.md`)
- `gcp iam/admin apiv1`: staged implementation for service account and key lifecycle, role administration, policy workflows, and REST/gRPC transport parity (`docs/gcp-iam-admin-apiv1-plan.md`)
- `gcp iam/credentials apiv1`: staged implementation for service-account token/signature issuance workflows and REST/gRPC transport parity (`docs/gcp-iam-credentials-apiv1-plan.md`)
- `gcp iap/apiv1`: staged implementation for admin settings/policy controls, tunnel destination group lifecycle, and OAuth brand/client workflows (`docs/gcp-iap-apiv1-plan.md`)
- `gcp ids/apiv1`: staged implementation for IDS endpoint lifecycle, long-running operation semantics, and REST/gRPC transport parity (`docs/gcp-ids-apiv1-plan.md`)
- `gcp iot/apiv1`: staged implementation for registry/device lifecycle, config/state controls, IAM methods, and gateway command workflows (`docs/gcp-iot-apiv1-plan.md`)
- `gcp kms/apiv1`: staged implementation for key hierarchy lifecycle, cryptographic operation workflows, import-job orchestration, and IAM/location controls (`docs/gcp-kms-apiv1-plan.md`)
- `gcp kms/inventory apiv1`: staged implementation for cross-project key visibility and protected-resource tracking/search workflows (`docs/gcp-kms-inventory-apiv1-plan.md`)
- `gcp language/apiv1`: staged implementation for sentiment/entity/syntax analysis and document classification/moderation workflows (`docs/gcp-language-apiv1-plan.md`)
- `gcp language/apiv2`: staged implementation for v2 sentiment/entity analysis and document classification/moderation workflows (`docs/gcp-language-apiv2-plan.md`)
- `gcp locationfinder/apiv1`: staged implementation for cloud location catalog discovery, search workflows, and location metadata endpoints (`docs/gcp-locationfinder-apiv1-plan.md`)
- `gcp maps/areainsights apiv1`: staged implementation for places aggregate insights and filter compatibility across REST and gRPC transports (`docs/gcp-maps-areainsights-apiv1-plan.md`)
- `gcp maps/addressvalidation apiv1`: staged implementation for address validation and feedback workflows across REST and gRPC transports (`docs/gcp-maps-addressvalidation-apiv1-plan.md`)
- `gcp maps/fleetengine apiv1`: staged implementation for trip and vehicle lifecycle, billable-trip reporting, and search workflows across REST and gRPC transports (`docs/gcp-maps-fleetengine-apiv1-plan.md`)
- `gcp maps/fleetengine/delivery apiv1`: staged implementation for delivery-vehicle lifecycle, task orchestration, and task-tracking workflows across REST and gRPC transports (`docs/gcp-maps-fleetengine-delivery-apiv1-plan.md`)
- `gcp maps/places apiv1`: staged implementation for text/nearby/autocomplete search workflows plus place-details and photo-media retrieval (`docs/gcp-maps-places-apiv1-plan.md`)
- `gcp maps/routeoptimization apiv1`: staged implementation for optimize and batch tour workflows with long-running operation polling support (`docs/gcp-maps-routeoptimization-apiv1-plan.md`)
- `gcp maps/routing apiv2`: staged implementation for route and route-matrix computation workflows across REST and gRPC transports (`docs/gcp-maps-routing-apiv2-plan.md`)
- `gcp maps/solar apiv1`: staged implementation for building insights, solar data-layer retrieval, and GeoTIFF asset access workflows (`docs/gcp-maps-solar-apiv1-plan.md`)
- `gcp mediatranslation/apiv1beta1`: staged implementation for bidirectional streaming speech translation workflows and stream-session lifecycle handling (`docs/gcp-mediatranslation-apiv1beta1-plan.md`)
- `gcp memcache/apiv1`: staged implementation for instance lifecycle, parameter and maintenance controls, and operations/location discovery workflows (`docs/gcp-memcache-apiv1-plan.md`)
- `gcp memorystore/apiv1`: staged implementation for instance lifecycle, backup/export actions, maintenance controls, and operations/location discovery workflows (`docs/gcp-memorystore-apiv1-plan.md`)
- `gcp metastore/apiv1`: staged implementation for service/import/backup lifecycle, metadata actions, IAM controls, and operations/location discovery workflows (`docs/gcp-metastore-apiv1-plan.md`)
- `gcp migrationcenter/apiv1`: staged implementation for asset/import/group/source lifecycle, report workflows, and location/operation controls (`docs/gcp-migrationcenter-apiv1-plan.md`)
- `gcp modelarmor/apiv1`: staged implementation for template/floor-setting lifecycle and runtime prompt/response sanitization workflows (`docs/gcp-modelarmor-apiv1-plan.md`)
- `gcp monitoring dashboard/apiv1`: staged implementation for dashboard lifecycle, layout/widget compatibility, and REST/gRPC transport parity (`docs/gcp-monitoring-dashboard-apiv1-plan.md`)
- `gcp metricsscope/apiv1`: staged implementation for metrics-scope discovery and monitored-project link lifecycle workflows (`docs/gcp-metricsscope-apiv1-plan.md`)
- `gcp netapp/apiv1`: staged implementation for storage pool and volume lifecycle, data-protection workflows, and REST/gRPC transport parity (`docs/gcp-netapp-apiv1-plan.md`)
- `gcp notebooks/apiv1`: staged implementation for notebook instance/environment/schedule/execution workflows and managed runtime control across gRPC and REST path compatibility (`docs/gcp-notebooks-apiv1-plan.md`)
- `gcp notebooks/apiv2`: staged implementation for notebook instance lifecycle and control actions, IAM and operation discovery workflows, and REST/gRPC transport parity (`docs/gcp-notebooks-apiv2-plan.md`)
- `gcp optimization/apiv1`: staged implementation for synchronous and batch tour optimization workflows plus long-running operation compatibility across REST/gRPC (`docs/gcp-optimization-apiv1-plan.md`)
- `gcp osconfig/apiv1`: staged implementation for patch job/deployment lifecycle plus zonal policy-assignment, inventory, and vulnerability workflows across REST/gRPC (`docs/gcp-osconfig-apiv1-plan.md`)
- `gcp osconfig/agentendpoint apiv1`: staged implementation for agent task orchestration, registration, inventory reporting, and streaming notification compatibility (`docs/gcp-osconfig-agentendpoint-apiv1-plan.md`)
- `gcp oslogin/apiv1`: staged implementation for SSH key lifecycle, login profile shaping, and POSIX account workflows across REST/gRPC (`docs/gcp-oslogin-apiv1-plan.md`)
- `gcp parallelstore/apiv1`: staged implementation for instance lifecycle, import/export action workflows, and operation/location discovery with REST/gRPC transport parity (`docs/gcp-parallelstore-apiv1-plan.md`)
- `gcp parametermanager/apiv1`: staged implementation for parameter and parameter-version lifecycle workflows, render semantics, and REST/gRPC transport parity (`docs/gcp-parametermanager-apiv1-plan.md`)
- `gcp privatecatalog/apiv1beta1`: staged implementation for catalog/product/version search workflows and REST/gRPC transport parity (`docs/gcp-privatecatalog-apiv1beta1-plan.md`)
- `gcp privilegedaccessmanager/apiv1`: staged implementation for onboarding checks, entitlement/grant lifecycle workflows, and REST/gRPC transport parity (`docs/gcp-privilegedaccessmanager-apiv1-plan.md`)
- `gcp pubsub/apiv1`: staged implementation for topic/subscription/snapshot/schema workflows and REST/gRPC transport parity (`docs/gcp-pubsub-apiv1-plan.md`)
- `gcp pubsub/v2/apiv1`: staged implementation for TopicAdmin/SubscriptionAdmin/Schema workflows using the Pub/Sub v2 Go SDK transport surface (`docs/gcp-pubsub-v2-apiv1-plan.md`)
- `gcp pubsublite/apiv1`: staged implementation for Admin/Cursor/PartitionAssignment/Publisher/Subscriber/TopicStats workflows with gRPC transport coverage (`docs/gcp-pubsublite-apiv1-plan.md`)
- `gcp phishingprotection/apiv1beta1`: staged implementation for phishing report submission workflows with deterministic validation, idempotency semantics, and REST/gRPC transport parity (`docs/gcp-phishingprotection-apiv1beta1-plan.md`)
- `gcp policysimulator/apiv1`: staged implementation for replay and org-policy violations preview workflows with deterministic lifecycle semantics and REST/gRPC transport parity (`docs/gcp-policysimulator-apiv1-plan.md`)
- `gcp policytroubleshooter/apiv1`: staged implementation for IAM access troubleshoot workflows with deterministic validation, explanation shaping, and REST/gRPC transport parity (`docs/gcp-policytroubleshooter-apiv1-plan.md`)
- `gcp policytroubleshooter/iam apiv3`: staged implementation for IAM v3 access troubleshooting workflows with deterministic validation, allow/deny explanation shaping, and REST/gRPC transport parity (`docs/gcp-policytroubleshooter-iam-apiv3-plan.md`)
- `gcp orgpolicy/apiv2`: staged implementation for org policy and custom constraint lifecycle workflows plus effective-policy evaluation compatibility across REST/gRPC (`docs/gcp-orgpolicy-apiv2-plan.md`)
- `gcp oracledatabase/apiv1`: staged implementation for Exadata/VM cluster/autonomous database/ODB network lifecycle workflows plus control actions and operations/location compatibility (`docs/gcp-oracledatabase-apiv1-plan.md`)
- `gcp monitoring/apiv3`: staged implementation for alerting, metrics, service monitoring, uptime checks, and time-series query workflows (`docs/gcp-monitoring-apiv3-plan.md`)
- `gcp networkservices/apiv1`: staged implementation for core network resource lifecycle, route views, IAM controls, and long-running operation workflows (`docs/gcp-networkservices-apiv1-plan.md`)
- `gcp networkconnectivity/apiv1`: staged implementation for hub/spoke, service-connection, internal-range, policy-routing, and data-transfer workflows (`docs/gcp-networkconnectivity-apiv1-plan.md`)
- `gcp networkmanagement/apiv1`: staged implementation for reachability testing, VPC flow logs config lifecycle, and REST/gRPC transport parity (`docs/gcp-networkmanagement-apiv1-plan.md`)
- `gcp networksecurity/apiv1beta1`: staged implementation for authorization and TLS policy lifecycle workflows with REST/gRPC transport parity (`docs/gcp-networksecurity-apiv1beta1-plan.md`)
- `gcp logging/apiv2`: staged implementation for log ingestion/query, sink/exclusion/metric lifecycle workflows, and REST/gRPC transport parity (`docs/gcp-logging-apiv2-plan.md`)
- `gcp lustre/apiv1`: staged implementation for instance lifecycle, import/export action workflows, and REST/gRPC transport parity (`docs/gcp-lustre-apiv1-plan.md`)
- `gcp managedkafka/apiv1`: staged implementation for cluster/topic/consumer-group/ACL lifecycle workflows and REST/gRPC transport parity (`docs/gcp-managedkafka-apiv1-plan.md`)
- `gcp managedkafka/schemaregistry/apiv1`: staged implementation for registry/context/schema/version/config/mode workflows and REST/gRPC transport parity (`docs/gcp-managedkafka-schemaregistry-apiv1-plan.md`)
- `gcp managedidentities/apiv1`: staged implementation for domain/trust lifecycle workflows and REST/gRPC transport parity (`docs/gcp-managedidentities-apiv1-plan.md`)
- `gcp lifesciences/apiv2beta`: staged implementation for pipeline execution workflows with location discovery and long-running operation controls (`docs/gcp-lifesciences-apiv2beta-plan.md`)
- `gcp licensemanager/apiv1`: staged implementation for configuration lifecycle, instance/product discovery, usage analytics, and location/operation workflows (`docs/gcp-licensemanager-apiv1-plan.md`)
- `gcp identitytoolkit/apiv2`: staged implementation for MFA enrollment/sign-in lifecycle flows and transport parity across account/authentication services (`docs/gcp-identitytoolkit-apiv2-plan.md`)

SDK endpoint override base URLs:

- GCP Cloud Storage SDK: `http://localhost:4566/gcp`
- Azure Blob SDK: `http://localhost:4566/azure/<storage-account>`
- OCI Object Storage SDK: `http://localhost:4566/oci`

Provider auth modes (configurable via CLI flags or env vars):

- GCP: `emulator` (default), `bearer_tolerant`, `bearer_required`
- Azure: `shared_key_or_sas` (default), `shared_key`, `sas`, `disabled`
- OCI: `signature` (default), `disabled`

Run with Docker Compose:

```bash
docker compose up --build
```

Default AWS-style credentials for local clients:

```bash
export AWS_ACCESS_KEY_ID=stackyard
export AWS_SECRET_ACCESS_KEY=stackyard
export AWS_REGION=us-east-1
```

Health check:

```bash
curl http://localhost:4566/_stackyard/health
```

## Development Workflow

Common targets:

```bash
make fmt
make tidy
make test
make ci
```

Additional automation:

- Run all provider Docker examples (`aws`, `gcp`, `azure`, `oci`): `make examples-docker`
- Run one provider only: `make examples-docker-aws` (also `-gcp`, `-azure`, `-oci`)
- Run endpoint coverage: `make coverage-all`

## Examples

Each service typically includes one runnable compose example.

See `examples/aws/` and `examples/gcp/` for service-specific Dockerfiles and compose files.

GCP Generative Language example scaffold:

- `examples/gcp/generativelanguage-apiv1`

GCP Data Analytics with Gemini example scaffold:

- `examples/gcp/geminidataanalytics-apiv1`

GCP Vertex AI example scaffold:

- `examples/gcp/aiplatform-apiv1`

GCP Gateway API example scaffold:

- `examples/gcp/apigateway-apiv1`

GCP Apigee Connect example scaffold:

- `examples/gcp/apigeeconnect-apiv1`

GCP API Hub example scaffold:

- `examples/gcp/apihub-apiv1`

GCP API Keys example scaffold:

- `examples/gcp/apikeys-apiv2`

GCP App Engine Admin example scaffold:

- `examples/gcp/appengine-apiv1`

GCP Workspace Events Subscriptions example scaffold:

- `examples/gcp/subscriptions-apiv1`

GCP Artifact Registry example scaffold:

- `examples/gcp/artifactregistry-apiv1`

GCP AutoML example scaffold:

- `examples/gcp/automl-apiv1`

GCP Google Meet example scaffold:

- `examples/gcp/meet-apiv2`

GCP Google Chat example scaffold:

- `examples/gcp/chat-apiv1`

GCP Chronicle example scaffold:

- `examples/gcp/chronicle-apiv1`

GCP Cloud Bigtable example scaffold:

- `examples/gcp/bigtable-apiv2`

GCP Certificate Manager example scaffold:

- `examples/gcp/certificatemanager-apiv1`

GCP Cloud Channel example scaffold:

- `examples/gcp/channel-apiv1`

GCP Cloud Controls Partner example scaffold:

- `examples/gcp/cloudcontrolspartner-apiv1`

GCP Database Migration example scaffold:

- `examples/gcp/clouddms-apiv1`

GCP Cloud Profiler example scaffold:

- `examples/gcp/cloudprofiler-apiv2`

GCP Cloud Quotas example scaffold:

- `examples/gcp/cloudquotas-apiv1`

GCP Cloud Tasks example scaffold:

- `examples/gcp/cloudtasks-apiv2`

GCP Cloud Build example scaffold:

- `examples/gcp/cloudbuild-apiv2`

GCP Google Compute Engine example scaffold:

- `examples/gcp/compute-apiv1`

GCP Compute Metadata example scaffold:

- `examples/gcp/compute-metadata-apiv1`

GCP Cloud Commerce Consumer Procurement example scaffold:

- `examples/gcp/commerce-consumer-procurement-apiv1`

GCP Confidential Computing example scaffold:

- `examples/gcp/confidentialcomputing-apiv1`

GCP Infrastructure Manager (Config API) example scaffold:

- `examples/gcp/config-apiv1`

GCP Config Delivery example scaffold:

- `examples/gcp/configdelivery-apiv1`

GCP Kubernetes Engine example scaffold:

- `examples/gcp/container-apiv1`

GCP Data Catalog example scaffold:

- `examples/gcp/datacatalog-apiv1`

GCP Data Lineage example scaffold:

- `examples/gcp/datacatalog-lineage-apiv1`

GCP Dataflow example scaffold:

- `examples/gcp/dataflow-apiv1beta3`

GCP Dataform example scaffold:

- `examples/gcp/dataform-apiv1`

GCP Cloud Datastore example scaffold:

- `examples/gcp/datastore-apiv1`

GCP Cloud Datastore Admin example scaffold:

- `examples/gcp/datastore-admin-apiv1`

GCP Datastream example scaffold:

- `examples/gcp/datastream-apiv1`

GCP Cloud Deploy example scaffold:

- `examples/gcp/deploy-apiv1`

GCP Dialogflow example scaffold:

- `examples/gcp/dialogflow-apiv1`

GCP Dialogflow CX example scaffold:

- `examples/gcp/dialogflow-cx-apiv1`

GCP Discovery Engine example scaffold:

- `examples/gcp/discoveryengine-apiv1`

GCP Cloud Document AI example scaffold:

- `examples/gcp/documentai-apiv1`

GCP Sensitive Data Protection (DLP) example scaffold:

- `examples/gcp/dlp-apiv1`

GCP Device Streaming example scaffold:

- `examples/gcp/devicestreaming-apiv1`

GCP Developer Connect example scaffold:

- `examples/gcp/developerconnect-apiv1`

GCP Data QnA example scaffold:

- `examples/gcp/dataqna-apiv1alpha`

GCP Cloud Domains example scaffold:

- `examples/gcp/domains-apiv1`

GCP Distributed Cloud Edge Container example scaffold:

- `examples/gcp/edgecontainer-apiv1`

GCP Data Fusion example scaffold:

- `examples/gcp/datafusion-apiv1`

GCP Data Labeling example scaffold:

- `examples/gcp/datalabeling-apiv1`

GCP Cloud Dataplex example scaffold:

- `examples/gcp/dataplex-apiv1`

GCP Cloud Dataproc example scaffold:

- `examples/gcp/dataproc-apiv1`

GCP Cloud Dataproc v2 example scaffold:

- `examples/gcp/dataproc-v2-apiv1`

GCP Error Reporting example scaffold:

- `examples/gcp/errorreporting-apiv1`

GCP Distributed Cloud Edge Network example scaffold:

- `examples/gcp/edgenetwork-apiv1`

GCP Essential Contacts example scaffold:

- `examples/gcp/essentialcontacts-apiv1`

GCP Eventarc example scaffold:

- `examples/gcp/eventarc-apiv1`

GCP Eventarc Publishing example scaffold:

- `examples/gcp/eventarc-publishing-apiv1`

GCP Cloud Filestore example scaffold:

- `examples/gcp/filestore-apiv1`

GCP Financial Services example scaffold:

- `examples/gcp/financialservices-apiv1`

GCP Game Services example scaffold:

- `examples/gcp/gaming-apiv1`

GCP Cloud Functions v1 example scaffold:

- `examples/gcp/functions-apiv1`

GCP Cloud Functions v2 example scaffold:

- `examples/gcp/functions-apiv2`

GCP Cloud Firestore example scaffold:

- `examples/gcp/firestore-apiv1`

GCP GKE Backup example scaffold:

- `examples/gcp/gkebackup-apiv1`

GCP Connect Gateway example scaffold:

- `examples/gcp/gkeconnect-apiv1`

GCP GKE Hub example scaffold:

- `examples/gcp/gkehub-apiv1`

GCP GKE Multi-Cloud example scaffold:

- `examples/gcp/gkemulticloud-apiv1`

GCP Cloud IAM example scaffold:

- `examples/gcp/iam-apiv1`

GCP Cloud IAM v2 example scaffold:

- `examples/gcp/iam-apiv2`

GCP Cloud IAM v3 example scaffold:

- `examples/gcp/iam-apiv3`

GCP Cloud IAM Admin example scaffold:

- `examples/gcp/iam-admin-apiv1`

GCP Cloud IAM Credentials example scaffold:

- `examples/gcp/iam-credentials-apiv1`

GCP Cloud Identity-Aware Proxy example scaffold:

- `examples/gcp/iap-apiv1`

GCP Cloud IDS example scaffold:

- `examples/gcp/ids-apiv1`

GCP Cloud IoT example scaffold:

- `examples/gcp/iot-apiv1`

GCP Cloud Key Management Service example scaffold:

- `examples/gcp/kms-apiv1`

GCP Cloud Key Management Service Inventory example scaffold:

- `examples/gcp/kms-inventory-apiv1`

GCP Cloud Natural Language example scaffold:

- `examples/gcp/language-apiv1`

GCP Cloud Natural Language v2 example scaffold:

- `examples/gcp/language-apiv2`

GCP Cloud Location Finder example scaffold:

- `examples/gcp/locationfinder-apiv1`

GCP Maps Places Aggregate example scaffold:

- `examples/gcp/maps-areainsights-apiv1`

GCP Maps Address Validation example scaffold:

- `examples/gcp/maps-addressvalidation-apiv1`

GCP Local Rides and Deliveries (Fleet Engine) example scaffold:

- `examples/gcp/maps-fleetengine-apiv1`

GCP Last Mile Fleet Delivery Solution example scaffold:

- `examples/gcp/maps-fleetengine-delivery-apiv1`

GCP Maps Places example scaffold:

- `examples/gcp/maps-places-apiv1`

GCP Maps Route Optimization example scaffold:

- `examples/gcp/maps-routeoptimization-apiv1`

GCP Maps Routes example scaffold:

- `examples/gcp/maps-routing-apiv2`

GCP Maps Solar example scaffold:

- `examples/gcp/maps-solar-apiv1`

GCP Media Translation example scaffold:

- `examples/gcp/mediatranslation-apiv1beta1`

GCP Cloud Memorystore for Memcached example scaffold:

- `examples/gcp/memcache-apiv1`

GCP Memorystore example scaffold:

- `examples/gcp/memorystore-apiv1`

GCP Dataproc Metastore example scaffold:

- `examples/gcp/metastore-apiv1`

GCP Migration Center example scaffold:

- `examples/gcp/migrationcenter-apiv1`

GCP Model Armor example scaffold:

- `examples/gcp/modelarmor-apiv1`

GCP Cloud Monitoring example scaffold:

- `examples/gcp/monitoring-apiv3`

GCP Cloud Monitoring Dashboard example scaffold:

- `examples/gcp/monitoring-dashboard-apiv1`

GCP Cloud Monitoring Metrics Scope example scaffold:

- `examples/gcp/metricsscope-apiv1`

GCP NetApp example scaffold:

- `examples/gcp/netapp-apiv1`

GCP Notebooks example scaffold:

- `examples/gcp/notebooks-apiv1`
- `examples/gcp/notebooks-apiv2`

GCP Cloud Optimization example scaffold:

- `examples/gcp/optimization-apiv1`

GCP OS Config example scaffold:

- `examples/gcp/osconfig-apiv1`

GCP OS Config Agent Endpoint example scaffold:

- `examples/gcp/osconfig-agentendpoint-apiv1`

GCP Cloud OS Login example scaffold:

- `examples/gcp/oslogin-apiv1`

GCP Parallelstore example scaffold:

- `examples/gcp/parallelstore-apiv1`

GCP Parameter Manager example scaffold:

- `examples/gcp/parametermanager-apiv1`

GCP Cloud Private Catalog example scaffold:

- `examples/gcp/privatecatalog-apiv1beta1`

GCP Privileged Access Manager example scaffold:

- `examples/gcp/privilegedaccessmanager-apiv1`

GCP Cloud Pub/Sub example scaffold:

- `examples/gcp/pubsub-apiv1`
- `examples/gcp/pubsub-v2-apiv1`

GCP Cloud Pub/Sub Lite example scaffold:

- `examples/gcp/pubsublite-apiv1`

GCP Phishing Protection example scaffold:

- `examples/gcp/phishingprotection-apiv1beta1`

GCP Policy Simulator example scaffold:

- `examples/gcp/policysimulator-apiv1`

GCP Policy Troubleshooter example scaffold:

- `examples/gcp/policytroubleshooter-apiv1`

GCP Policy Troubleshooter IAM example scaffold:

- `examples/gcp/policytroubleshooter-iam-apiv3`

GCP Organization Policy example scaffold:

- `examples/gcp/orgpolicy-apiv2`

GCP Oracle Database example scaffold:

- `examples/gcp/oracledatabase-apiv1`

GCP Network Services example scaffold:

- `examples/gcp/networkservices-apiv1`

GCP Network Connectivity example scaffold:

- `examples/gcp/networkconnectivity-apiv1`

GCP Network Management example scaffold:

- `examples/gcp/networkmanagement-apiv1`

GCP Network Security example scaffold:

- `examples/gcp/networksecurity-apiv1beta1`

GCP Logging example scaffold:

- `examples/gcp/logging-apiv2`

GCP Lustre example scaffold:

- `examples/gcp/lustre-apiv1`

GCP Managed Kafka example scaffold:

- `examples/gcp/managedkafka-apiv1`

GCP Managed Kafka Schema Registry example scaffold:

- `examples/gcp/managedkafka-schemaregistry-apiv1`

GCP Managed Identities example scaffold:

- `examples/gcp/managedidentities-apiv1`

GCP Life Sciences example scaffold:

- `examples/gcp/lifesciences-apiv2beta`

GCP License Manager example scaffold:

- `examples/gcp/licensemanager-apiv1`

GCP Identity Toolkit v2 example scaffold:

- `examples/gcp/identitytoolkit-apiv2`

## Testing and Compatibility

Stackyard validates behavior through:

- service-level staged Go tests in `internal/server/*_test.go`
- smoke scripts in `scripts/`
- contract-style endpoint coverage checks using AWS CLI skeletons
- provider contract checks for GCP/Azure/OCI foundation routes (`make provider-contracts`)

This project aims for practical local compatibility, not complete protocol parity for every AWS feature.

## Contribution Model

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full contribution workflow.

Contributions are welcome. A good contribution generally includes:

- clear service/operation scope
- implementation in handler + store layers
- staged tests for new behavior and regressions
- docs updates where user-facing behavior changes
- example updates when the service supports meaningful workflows

Suggested PR structure:

1. Service operation/type catalog update
2. Stage implementation (incremental behavior)
3. Tests for unknown action, known action, and staged lifecycle
4. Documentation and example updates

## Extending a Service

Typical pattern for adding or expanding emulation:

1. Add operation/type catalogs
2. Add protocol-aware candidate/router parsing
3. Add in-memory store behavior by stage
4. Add stage tests
5. Add or update example compose
6. Add service entry to coverage script
7. Update docs index

## Reference Architecture Work

See `reference-architecture/` for architecture-driven examples that integrate multiple emulated services (Go backends + TypeScript frontends).

Canonical provider-scoped layout:

- `reference-architecture/data-pipeline/aws/example`
- `reference-architecture/data-pipeline/gcp/example`
- `reference-architecture/data-mesh/aws/example`
- `reference-architecture/data-mesh/gcp/example`

Run targets:

- `make refarch-examples-aws`
- `make refarch-examples-gcp`
- `make refarch-examples-all`

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting and disclosure guidance.

## Notes

- Stackyard is intended for local development, CI, and architecture prototyping.
- It is not intended as a production AWS replacement.
- Behavior can intentionally differ from AWS where simplification improves local usability, unless compatibility is explicitly required by tests.
