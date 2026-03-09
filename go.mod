module github.com/stackyard/stackyard

go 1.25.0

require (
	cloud.google.com/go/ai v0.8.2
	cloud.google.com/go/aiplatform v1.114.0
	cloud.google.com/go/apigateway v1.7.7
	cloud.google.com/go/apigeeconnect v1.7.7
	cloud.google.com/go/apihub v0.1.0
	cloud.google.com/go/apikeys v1.2.0
	cloud.google.com/go/appengine v1.9.7
	cloud.google.com/go/apps v0.5.1
	cloud.google.com/go/artifactregistry v1.19.0
	cloud.google.com/go/automl v1.15.0
	cloud.google.com/go/bigtable v1.41.0
	cloud.google.com/go/certificatemanager v1.9.6
	cloud.google.com/go/channel v1.21.0
	cloud.google.com/go/chat v0.15.0
	cloud.google.com/go/chronicle v0.1.1
	cloud.google.com/go/cloudbuild v1.25.0
	cloud.google.com/go/cloudcontrolspartner v1.4.0
	cloud.google.com/go/clouddms v1.8.8
	cloud.google.com/go/cloudprofiler v0.4.5
	cloud.google.com/go/cloudquotas v1.4.0
	cloud.google.com/go/cloudtasks v1.13.7
	cloud.google.com/go/commerce v1.2.5
	cloud.google.com/go/compute v1.54.0
	cloud.google.com/go/compute/metadata v0.9.0
	cloud.google.com/go/confidentialcomputing v1.9.2
	cloud.google.com/go/config v1.4.0
	cloud.google.com/go/configdelivery v0.1.0
	cloud.google.com/go/container v1.45.0
	cloud.google.com/go/datacatalog v1.26.1
	cloud.google.com/go/dataflow v0.11.1
	cloud.google.com/go/dataform v0.12.1
	cloud.google.com/go/datafusion v1.8.7
	cloud.google.com/go/datalabeling v0.9.7
	cloud.google.com/go/dataplex v1.28.0
	cloud.google.com/go/dataproc v1.12.0
	cloud.google.com/go/dataproc/v2 v2.15.0
	cloud.google.com/go/dataqna v0.9.8
	cloud.google.com/go/datastore v1.21.0
	cloud.google.com/go/datastream v1.15.1
	cloud.google.com/go/deploy v1.27.3
	cloud.google.com/go/developerconnect v0.3.3
	cloud.google.com/go/devicestreaming v0.1.0
	cloud.google.com/go/dialogflow v1.74.0
	cloud.google.com/go/discoveryengine v1.20.0
	cloud.google.com/go/dlp v1.28.0
	cloud.google.com/go/documentai v1.39.0
	cloud.google.com/go/domains v0.10.7
	cloud.google.com/go/edgecontainer v1.4.4
	cloud.google.com/go/edgenetwork v1.2.6
	cloud.google.com/go/errorreporting v0.4.0
	cloud.google.com/go/essentialcontacts v1.7.7
	cloud.google.com/go/eventarc v1.18.0
	cloud.google.com/go/filestore v1.10.3
	cloud.google.com/go/financialservices v0.1.3
	cloud.google.com/go/firestore v1.21.0
	cloud.google.com/go/functions v1.19.7
	cloud.google.com/go/gaming v1.10.1
	cloud.google.com/go/geminidataanalytics v0.2.0
	cloud.google.com/go/gkebackup v1.8.1
	cloud.google.com/go/gkeconnect v0.12.5
	cloud.google.com/go/gkehub v0.16.0
	cloud.google.com/go/gkemulticloud v1.6.0
	cloud.google.com/go/iam v1.5.3
	cloud.google.com/go/iap v1.11.3
	cloud.google.com/go/identitytoolkit v0.2.6
	cloud.google.com/go/ids v1.5.7
	cloud.google.com/go/iot v1.8.7
	cloud.google.com/go/kms v1.25.0
	cloud.google.com/go/language v1.14.6
	cloud.google.com/go/licensemanager v0.1.1
	cloud.google.com/go/lifesciences v0.10.7
	cloud.google.com/go/locationfinder v0.1.1
	cloud.google.com/go/logging v1.13.1
	cloud.google.com/go/longrunning v0.8.0
	cloud.google.com/go/lustre v0.2.1
	cloud.google.com/go/managedidentities v1.7.7
	cloud.google.com/go/managedkafka v0.8.1
	cloud.google.com/go/maps v1.26.0
	cloud.google.com/go/mediatranslation v0.9.7
	cloud.google.com/go/memcache v1.11.7
	cloud.google.com/go/memorystore v0.4.0
	cloud.google.com/go/metastore v1.14.8
	cloud.google.com/go/migrationcenter v1.1.6
	cloud.google.com/go/modelarmor v0.6.1
	cloud.google.com/go/monitoring v1.24.3
	cloud.google.com/go/netapp v1.12.0
	cloud.google.com/go/networkconnectivity v1.20.0
	cloud.google.com/go/networkmanagement v1.22.0
	cloud.google.com/go/networksecurity v0.11.0
	cloud.google.com/go/networkservices v0.6.0
	cloud.google.com/go/notebooks v1.12.7
	cloud.google.com/go/optimization v1.7.7
	cloud.google.com/go/oracledatabase v0.6.0
	cloud.google.com/go/orgpolicy v1.15.1
	cloud.google.com/go/osconfig v1.16.0
	cloud.google.com/go/oslogin v1.14.7
	cloud.google.com/go/parallelstore v0.12.0
	cloud.google.com/go/parametermanager v0.3.1
	cloud.google.com/go/phishingprotection v0.9.7
	cloud.google.com/go/policysimulator v0.4.1
	cloud.google.com/go/policytroubleshooter v1.11.7
	cloud.google.com/go/privatecatalog v0.10.8
	cloud.google.com/go/privilegedaccessmanager v0.3.1
	cloud.google.com/go/pubsub v1.50.1
	cloud.google.com/go/pubsub/v2 v2.0.0
	cloud.google.com/go/pubsublite v1.8.2
	cloud.google.com/go/rapidmigrationassessment v1.1.8
	cloud.google.com/go/recaptchaenterprise/v2 v2.21.0
	cloud.google.com/go/recommendationengine v0.9.7
	cloud.google.com/go/redis v1.18.3
	cloud.google.com/go/resourcemanager v1.10.7
	cloud.google.com/go/retail v1.26.0
	cloud.google.com/go/run v1.15.0
	cloud.google.com/go/scheduler v1.11.8
	cloud.google.com/go/secretmanager v1.16.0
	cloud.google.com/go/securesourcemanager v1.4.1
	cloud.google.com/go/security v1.19.2
	cloud.google.com/go/securitycenter v1.38.1
	cloud.google.com/go/securitycentermanagement v1.1.6
	cloud.google.com/go/securityposture v0.2.6
	cloud.google.com/go/servicecontrol v1.14.6
	cloud.google.com/go/servicedirectory v1.12.7
	cloud.google.com/go/servicehealth v1.2.4
	cloud.google.com/go/servicemanagement v1.10.7
	cloud.google.com/go/serviceusage v1.9.7
	cloud.google.com/go/shell v1.8.7
	cloud.google.com/go/shopping v1.4.0
	cloud.google.com/go/spanner v1.87.0
	cloud.google.com/go/speech v1.30.0
	cloud.google.com/go/storage v1.56.0
	cloud.google.com/go/storagebatchoperations v0.4.0
	cloud.google.com/go/storageinsights v1.2.1
	cloud.google.com/go/storagetransfer v1.13.1
	cloud.google.com/go/streetview v0.2.6
	cloud.google.com/go/support v1.5.0
	cloud.google.com/go/talent v1.8.4
	cloud.google.com/go/telcoautomation v1.1.6
	cloud.google.com/go/texttospeech v1.16.0
	cloud.google.com/go/tpu v1.8.4
	cloud.google.com/go/trace v1.11.7
	cloud.google.com/go/translate v1.12.7
	cloud.google.com/go/video v1.27.1
	cloud.google.com/go/videointelligence v1.12.7
	cloud.google.com/go/vision v1.2.0
	cloud.google.com/go/vision/v2 v2.9.6
	cloud.google.com/go/visionai v0.5.0
	cloud.google.com/go/vmmigration v1.10.0
	cloud.google.com/go/vmwareengine v1.3.6
	cloud.google.com/go/vpcaccess v1.8.7
	cloud.google.com/go/webrisk v1.11.2
	cloud.google.com/go/websecurityscanner v1.7.7
	cloud.google.com/go/workstations v1.1.6
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.27.0
	github.com/aws/aws-sdk-go-v2/credentials v1.17.0
	github.com/aws/aws-sdk-go-v2/service/athena v1.31.0
	github.com/aws/aws-sdk-go-v2/service/batch v1.31.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.245.1
	github.com/aws/aws-sdk-go-v2/service/ecr v1.31.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.71.0
	github.com/aws/aws-sdk-go-v2/service/elasticbeanstalk v1.32.0
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.31.0
	github.com/aws/aws-sdk-go-v2/service/lambda v1.76.1
	github.com/aws/aws-sdk-go-v2/service/lightsail v1.44.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.59.0
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.31.0
	github.com/aws/aws-sdk-go-v2/service/sns v1.31.0
	github.com/aws/aws-sdk-go-v2/service/sqs v1.34.0
	github.com/aws/aws-sdk-go-v2/service/swf v1.31.0
	golang.org/x/net v0.49.0
	google.golang.org/api v0.265.0
	google.golang.org/genproto v0.0.0-20260128011058-8636f8732409
	google.golang.org/genproto/googleapis/api v0.0.0-20260203192932-546029d2fa20
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
)

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.18.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.30.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.55.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.55.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cncf/xds/go v0.0.0-20251022180443-0feb69152e9f // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.35.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.2.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-jose/go-jose/v4 v4.1.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/googleapis/gax-go/v2 v2.17.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.38.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk v1.39.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.14.0 // indirect
)

require (
	cloud.google.com/go/recommender v1.13.6
	cloud.google.com/go/workflows v1.14.3
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/ses v1.31.0
	github.com/aws/aws-sdk-go-v2/service/sso v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.27.0 // indirect
	github.com/aws/smithy-go v1.24.0
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)

replace cloud.google.com/go/gkebackup => cloud.google.com/go/gkebackup v1.7.0
