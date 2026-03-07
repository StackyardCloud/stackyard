package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	apigeeconnectpb "cloud.google.com/go/apigeeconnect/apiv1/apigeeconnectpb"
	cloudprofilerpb "cloud.google.com/go/cloudprofiler/apiv2/cloudprofilerpb"
	cloudquotaspb "cloud.google.com/go/cloudquotas/apiv1/cloudquotaspb"
	procurementpb "cloud.google.com/go/commerce/consumer/procurement/apiv1/procurementpb"
	configpb "cloud.google.com/go/config/apiv1/configpb"
	configdeliverypb "cloud.google.com/go/configdelivery/apiv1/configdeliverypb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	mediatranslationpb "cloud.google.com/go/mediatranslation/apiv1beta1/mediatranslationpb"
	rapidmigrationassessmentpb "cloud.google.com/go/rapidmigrationassessment/apiv1/rapidmigrationassessmentpb"
	recaptchaenterprisepb "cloud.google.com/go/recaptchaenterprise/v2/apiv1/recaptchaenterprisepb"
	recommendationenginepb "cloud.google.com/go/recommendationengine/apiv1beta1/recommendationenginepb"
	recommenderpb "cloud.google.com/go/recommender/apiv1/recommenderpb"
	redispb "cloud.google.com/go/redis/apiv1/redispb"
	clusterpb "cloud.google.com/go/redis/cluster/apiv1/clusterpb"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv2/resourcemanagerpb"
	retailpb "cloud.google.com/go/retail/apiv2/retailpb"
	runpb "cloud.google.com/go/run/apiv2/runpb"
	schedulerpb "cloud.google.com/go/scheduler/apiv1/schedulerpb"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	securesourcemanagerpb "cloud.google.com/go/securesourcemanager/apiv1/securesourcemanagerpb"
	privatecapb "cloud.google.com/go/security/privateca/apiv1/privatecapb"
	publiccapb "cloud.google.com/go/security/publicca/apiv1/publiccapb"
	securitycenterpb "cloud.google.com/go/securitycenter/apiv1/securitycenterpb"
	securitycenterv2pb "cloud.google.com/go/securitycenter/apiv2/securitycenterpb"
	securitycentermanagementpb "cloud.google.com/go/securitycentermanagement/apiv1/securitycentermanagementpb"
	securityposturepb "cloud.google.com/go/securityposture/apiv1/securityposturepb"
	servicecontrolpb "cloud.google.com/go/servicecontrol/apiv1/servicecontrolpb"
	servicedirectorypb "cloud.google.com/go/servicedirectory/apiv1/servicedirectorypb"
	servicehealthpb "cloud.google.com/go/servicehealth/apiv1/servicehealthpb"
	servicemanagementpb "cloud.google.com/go/servicemanagement/apiv1/servicemanagementpb"
	serviceusagepb "cloud.google.com/go/serviceusage/apiv1/serviceusagepb"
	shellpb "cloud.google.com/go/shell/apiv1/shellpb"
	shoppingcsspb "cloud.google.com/go/shopping/css/apiv1/csspb"
	accountspb "cloud.google.com/go/shopping/merchant/accounts/apiv1/accountspb"
	conversionspb "cloud.google.com/go/shopping/merchant/conversions/apiv1/conversionspb"
	datasourcespb "cloud.google.com/go/shopping/merchant/datasources/apiv1/datasourcespb"
	inventoriespb "cloud.google.com/go/shopping/merchant/inventories/apiv1/inventoriespb"
	issueresolutionpb "cloud.google.com/go/shopping/merchant/issueresolution/apiv1/issueresolutionpb"
	lfppb "cloud.google.com/go/shopping/merchant/lfp/apiv1/lfppb"
	notificationspb "cloud.google.com/go/shopping/merchant/notifications/apiv1/notificationspb"
	ordertrackingpb "cloud.google.com/go/shopping/merchant/ordertracking/apiv1/ordertrackingpb"
	productspb "cloud.google.com/go/shopping/merchant/products/apiv1/productspb"
	productstudiopb "cloud.google.com/go/shopping/merchant/productstudio/apiv1alpha/productstudiopb"
	promotionspb "cloud.google.com/go/shopping/merchant/promotions/apiv1/promotionspb"
	quotapb "cloud.google.com/go/shopping/merchant/quota/apiv1/quotapb"
	reportspb "cloud.google.com/go/shopping/merchant/reports/apiv1/reportspb"
	reviewspb "cloud.google.com/go/shopping/merchant/reviews/apiv1beta/reviewspb"
	shoppingtypepb "cloud.google.com/go/shopping/type/typepb"
	adapterpb "cloud.google.com/go/spanner/adapter/apiv1/adapterpb"
	spanneradminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	spanneradmininstancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	spannerpb "cloud.google.com/go/spanner/apiv1/spannerpb"
	executorpb "cloud.google.com/go/spanner/executor/apiv1/executorpb"
	speechpb "cloud.google.com/go/speech/apiv1/speechpb"
	speechv2pb "cloud.google.com/go/speech/apiv2/speechpb"
	storagebatchoperationspb "cloud.google.com/go/storagebatchoperations/apiv1/storagebatchoperationspb"
	storageinsightspb "cloud.google.com/go/storageinsights/apiv1/storageinsightspb"
	storagetransferpb "cloud.google.com/go/storagetransfer/apiv1/storagetransferpb"
	publishpb "cloud.google.com/go/streetview/publish/apiv1/publishpb"
	supportpb "cloud.google.com/go/support/apiv2/supportpb"
	talentpb "cloud.google.com/go/talent/apiv4/talentpb"
	vmmigrationpb "cloud.google.com/go/vmmigration/apiv1/vmmigrationpb"
	webriskpb "cloud.google.com/go/webrisk/apiv1/webriskpb"
	httpbodypb "google.golang.org/genproto/googleapis/api/httpbody"
	serviceconfigpb "google.golang.org/genproto/googleapis/api/serviceconfig"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	moneypb "google.golang.org/genproto/googleapis/type/money"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpApigeeConnectListConnectionsMethod = "/google.cloud.apigeeconnect.v1.ConnectionService/ListConnections"
	gcpApigeeConnectTetherEgressMethod    = "/google.cloud.apigeeconnect.v1.Tether/Egress"

	gcpMediaTranslationStreamingTranslateSpeechMethod = "/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/StreamingTranslateSpeech"

	gcpCloudProfilerCreateProfileMethod        = "/google.devtools.cloudprofiler.v2.ProfilerService/CreateProfile"
	gcpCloudProfilerCreateOfflineProfileMethod = "/google.devtools.cloudprofiler.v2.ProfilerService/CreateOfflineProfile"
	gcpCloudProfilerUpdateProfileMethod        = "/google.devtools.cloudprofiler.v2.ProfilerService/UpdateProfile"
	gcpCloudProfilerListProfilesMethod         = "/google.devtools.cloudprofiler.v2.ExportService/ListProfiles"

	gcpCloudQuotasListQuotaInfosMethod        = "/google.api.cloudquotas.v1.CloudQuotas/ListQuotaInfos"
	gcpCloudQuotasGetQuotaInfoMethod          = "/google.api.cloudquotas.v1.CloudQuotas/GetQuotaInfo"
	gcpCloudQuotasListQuotaPreferencesMethod  = "/google.api.cloudquotas.v1.CloudQuotas/ListQuotaPreferences"
	gcpCloudQuotasGetQuotaPreferenceMethod    = "/google.api.cloudquotas.v1.CloudQuotas/GetQuotaPreference"
	gcpCloudQuotasCreateQuotaPreferenceMethod = "/google.api.cloudquotas.v1.CloudQuotas/CreateQuotaPreference"
	gcpCloudQuotasUpdateQuotaPreferenceMethod = "/google.api.cloudquotas.v1.CloudQuotas/UpdateQuotaPreference"

	gcpProcurementPlaceOrderMethod  = "/google.cloud.commerce.consumer.procurement.v1.ConsumerProcurementService/PlaceOrder"
	gcpProcurementGetOrderMethod    = "/google.cloud.commerce.consumer.procurement.v1.ConsumerProcurementService/GetOrder"
	gcpProcurementListOrdersMethod  = "/google.cloud.commerce.consumer.procurement.v1.ConsumerProcurementService/ListOrders"
	gcpProcurementModifyOrderMethod = "/google.cloud.commerce.consumer.procurement.v1.ConsumerProcurementService/ModifyOrder"
	gcpProcurementCancelOrderMethod = "/google.cloud.commerce.consumer.procurement.v1.ConsumerProcurementService/CancelOrder"

	gcpConfigListDeploymentsMethod       = "/google.cloud.config.v1.Config/ListDeployments"
	gcpConfigGetDeploymentMethod         = "/google.cloud.config.v1.Config/GetDeployment"
	gcpConfigCreateDeploymentMethod      = "/google.cloud.config.v1.Config/CreateDeployment"
	gcpConfigDeleteDeploymentMethod      = "/google.cloud.config.v1.Config/DeleteDeployment"
	gcpConfigLockDeploymentMethod        = "/google.cloud.config.v1.Config/LockDeployment"
	gcpConfigUnlockDeploymentMethod      = "/google.cloud.config.v1.Config/UnlockDeployment"
	gcpConfigExportLockInfoMethod        = "/google.cloud.config.v1.Config/ExportLockInfo"
	gcpConfigCreatePreviewMethod         = "/google.cloud.config.v1.Config/CreatePreview"
	gcpConfigGetPreviewMethod            = "/google.cloud.config.v1.Config/GetPreview"
	gcpConfigListPreviewsMethod          = "/google.cloud.config.v1.Config/ListPreviews"
	gcpConfigDeletePreviewMethod         = "/google.cloud.config.v1.Config/DeletePreview"
	gcpConfigExportPreviewResultMethod   = "/google.cloud.config.v1.Config/ExportPreviewResult"
	gcpConfigListTerraformVersionsMethod = "/google.cloud.config.v1.Config/ListTerraformVersions"
	gcpConfigGetTerraformVersionMethod   = "/google.cloud.config.v1.Config/GetTerraformVersion"
	gcpConfigListResourceChangesMethod   = "/google.cloud.config.v1.Config/ListResourceChanges"
	gcpConfigGetResourceChangeMethod     = "/google.cloud.config.v1.Config/GetResourceChange"
	gcpConfigListResourceDriftsMethod    = "/google.cloud.config.v1.Config/ListResourceDrifts"
	gcpConfigGetResourceDriftMethod      = "/google.cloud.config.v1.Config/GetResourceDrift"

	gcpConfigDeliveryListResourceBundlesMethod  = "/google.cloud.configdelivery.v1.ConfigDelivery/ListResourceBundles"
	gcpConfigDeliveryGetResourceBundleMethod    = "/google.cloud.configdelivery.v1.ConfigDelivery/GetResourceBundle"
	gcpConfigDeliveryCreateResourceBundleMethod = "/google.cloud.configdelivery.v1.ConfigDelivery/CreateResourceBundle"
	gcpConfigDeliveryDeleteResourceBundleMethod = "/google.cloud.configdelivery.v1.ConfigDelivery/DeleteResourceBundle"
	gcpConfigDeliveryListFleetPackagesMethod    = "/google.cloud.configdelivery.v1.ConfigDelivery/ListFleetPackages"
	gcpConfigDeliveryGetFleetPackageMethod      = "/google.cloud.configdelivery.v1.ConfigDelivery/GetFleetPackage"
	gcpConfigDeliveryCreateFleetPackageMethod   = "/google.cloud.configdelivery.v1.ConfigDelivery/CreateFleetPackage"
	gcpConfigDeliveryDeleteFleetPackageMethod   = "/google.cloud.configdelivery.v1.ConfigDelivery/DeleteFleetPackage"
	gcpConfigDeliveryListReleasesMethod         = "/google.cloud.configdelivery.v1.ConfigDelivery/ListReleases"
	gcpConfigDeliveryGetReleaseMethod           = "/google.cloud.configdelivery.v1.ConfigDelivery/GetRelease"
	gcpConfigDeliveryCreateReleaseMethod        = "/google.cloud.configdelivery.v1.ConfigDelivery/CreateRelease"
	gcpConfigDeliveryDeleteReleaseMethod        = "/google.cloud.configdelivery.v1.ConfigDelivery/DeleteRelease"
	gcpConfigDeliveryListVariantsMethod         = "/google.cloud.configdelivery.v1.ConfigDelivery/ListVariants"
	gcpConfigDeliveryGetVariantMethod           = "/google.cloud.configdelivery.v1.ConfigDelivery/GetVariant"
	gcpConfigDeliveryCreateVariantMethod        = "/google.cloud.configdelivery.v1.ConfigDelivery/CreateVariant"
	gcpConfigDeliveryDeleteVariantMethod        = "/google.cloud.configdelivery.v1.ConfigDelivery/DeleteVariant"
	gcpConfigDeliveryListRolloutsMethod         = "/google.cloud.configdelivery.v1.ConfigDelivery/ListRollouts"
	gcpConfigDeliveryGetRolloutMethod           = "/google.cloud.configdelivery.v1.ConfigDelivery/GetRollout"
	gcpConfigDeliverySuspendRolloutMethod       = "/google.cloud.configdelivery.v1.ConfigDelivery/SuspendRollout"
	gcpConfigDeliveryResumeRolloutMethod        = "/google.cloud.configdelivery.v1.ConfigDelivery/ResumeRollout"
	gcpConfigDeliveryAbortRolloutMethod         = "/google.cloud.configdelivery.v1.ConfigDelivery/AbortRollout"

	gcpRapidMigrationAssessmentListCollectorsMethod    = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/ListCollectors"
	gcpRapidMigrationAssessmentGetCollectorMethod      = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/GetCollector"
	gcpRapidMigrationAssessmentCreateCollectorMethod   = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/CreateCollector"
	gcpRapidMigrationAssessmentUpdateCollectorMethod   = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/UpdateCollector"
	gcpRapidMigrationAssessmentDeleteCollectorMethod   = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/DeleteCollector"
	gcpRapidMigrationAssessmentPauseCollectorMethod    = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/PauseCollector"
	gcpRapidMigrationAssessmentResumeCollectorMethod   = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/ResumeCollector"
	gcpRapidMigrationAssessmentRegisterCollectorMethod = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/RegisterCollector"
	gcpRapidMigrationAssessmentCreateAnnotationMethod  = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/CreateAnnotation"
	gcpRapidMigrationAssessmentGetAnnotationMethod     = "/google.cloud.rapidmigrationassessment.v1.RapidMigrationAssessment/GetAnnotation"

	gcpVMMigrationListSourcesMethod    = "/google.cloud.vmmigration.v1.VmMigration/ListSources"
	gcpVMMigrationGetSourceMethod      = "/google.cloud.vmmigration.v1.VmMigration/GetSource"
	gcpVMMigrationCreateSourceMethod   = "/google.cloud.vmmigration.v1.VmMigration/CreateSource"
	gcpVMMigrationPauseMigrationMethod = "/google.cloud.vmmigration.v1.VmMigration/PauseMigration"

	gcpResourceManagerListFoldersMethod        = "/google.cloud.resourcemanager.v2.Folders/ListFolders"
	gcpResourceManagerSearchFoldersMethod      = "/google.cloud.resourcemanager.v2.Folders/SearchFolders"
	gcpResourceManagerGetFolderMethod          = "/google.cloud.resourcemanager.v2.Folders/GetFolder"
	gcpResourceManagerCreateFolderMethod       = "/google.cloud.resourcemanager.v2.Folders/CreateFolder"
	gcpResourceManagerUpdateFolderMethod       = "/google.cloud.resourcemanager.v2.Folders/UpdateFolder"
	gcpResourceManagerMoveFolderMethod         = "/google.cloud.resourcemanager.v2.Folders/MoveFolder"
	gcpResourceManagerDeleteFolderMethod       = "/google.cloud.resourcemanager.v2.Folders/DeleteFolder"
	gcpResourceManagerUndeleteFolderMethod     = "/google.cloud.resourcemanager.v2.Folders/UndeleteFolder"
	gcpResourceManagerGetIAMPolicyMethod       = "/google.cloud.resourcemanager.v2.Folders/GetIamPolicy"
	gcpResourceManagerSetIAMPolicyMethod       = "/google.cloud.resourcemanager.v2.Folders/SetIamPolicy"
	gcpResourceManagerTestIAMPermissionsMethod = "/google.cloud.resourcemanager.v2.Folders/TestIamPermissions"

	gcpRedisListInstancesMethod         = "/google.cloud.redis.v1.CloudRedis/ListInstances"
	gcpRedisGetInstanceMethod           = "/google.cloud.redis.v1.CloudRedis/GetInstance"
	gcpRedisGetInstanceAuthStringMethod = "/google.cloud.redis.v1.CloudRedis/GetInstanceAuthString"
	gcpRedisCreateInstanceMethod        = "/google.cloud.redis.v1.CloudRedis/CreateInstance"
	gcpRedisUpdateInstanceMethod        = "/google.cloud.redis.v1.CloudRedis/UpdateInstance"
	gcpRedisUpgradeInstanceMethod       = "/google.cloud.redis.v1.CloudRedis/UpgradeInstance"
	gcpRedisImportInstanceMethod        = "/google.cloud.redis.v1.CloudRedis/ImportInstance"
	gcpRedisExportInstanceMethod        = "/google.cloud.redis.v1.CloudRedis/ExportInstance"
	gcpRedisFailoverInstanceMethod      = "/google.cloud.redis.v1.CloudRedis/FailoverInstance"
	gcpRedisDeleteInstanceMethod        = "/google.cloud.redis.v1.CloudRedis/DeleteInstance"
	gcpRedisRescheduleMaintenanceMethod = "/google.cloud.redis.v1.CloudRedis/RescheduleMaintenance"

	gcpRedisClusterListClustersMethod                   = "/google.cloud.redis.cluster.v1.CloudRedisCluster/ListClusters"
	gcpRedisClusterGetClusterMethod                     = "/google.cloud.redis.cluster.v1.CloudRedisCluster/GetCluster"
	gcpRedisClusterUpdateClusterMethod                  = "/google.cloud.redis.cluster.v1.CloudRedisCluster/UpdateCluster"
	gcpRedisClusterDeleteClusterMethod                  = "/google.cloud.redis.cluster.v1.CloudRedisCluster/DeleteCluster"
	gcpRedisClusterCreateClusterMethod                  = "/google.cloud.redis.cluster.v1.CloudRedisCluster/CreateCluster"
	gcpRedisClusterGetClusterCertificateAuthorityMethod = "/google.cloud.redis.cluster.v1.CloudRedisCluster/GetClusterCertificateAuthority"
	gcpRedisClusterRescheduleClusterMaintenanceMethod   = "/google.cloud.redis.cluster.v1.CloudRedisCluster/RescheduleClusterMaintenance"
	gcpRedisClusterListBackupCollectionsMethod          = "/google.cloud.redis.cluster.v1.CloudRedisCluster/ListBackupCollections"
	gcpRedisClusterGetBackupCollectionMethod            = "/google.cloud.redis.cluster.v1.CloudRedisCluster/GetBackupCollection"
	gcpRedisClusterListBackupsMethod                    = "/google.cloud.redis.cluster.v1.CloudRedisCluster/ListBackups"
	gcpRedisClusterGetBackupMethod                      = "/google.cloud.redis.cluster.v1.CloudRedisCluster/GetBackup"
	gcpRedisClusterDeleteBackupMethod                   = "/google.cloud.redis.cluster.v1.CloudRedisCluster/DeleteBackup"
	gcpRedisClusterExportBackupMethod                   = "/google.cloud.redis.cluster.v1.CloudRedisCluster/ExportBackup"
	gcpRedisClusterBackupClusterMethod                  = "/google.cloud.redis.cluster.v1.CloudRedisCluster/BackupCluster"

	gcpRecommendationEngineCreateCatalogItemMethod                  = "/google.cloud.recommendationengine.v1beta1.CatalogService/CreateCatalogItem"
	gcpRecommendationEngineGetCatalogItemMethod                     = "/google.cloud.recommendationengine.v1beta1.CatalogService/GetCatalogItem"
	gcpRecommendationEngineListCatalogItemsMethod                   = "/google.cloud.recommendationengine.v1beta1.CatalogService/ListCatalogItems"
	gcpRecommendationEngineUpdateCatalogItemMethod                  = "/google.cloud.recommendationengine.v1beta1.CatalogService/UpdateCatalogItem"
	gcpRecommendationEngineDeleteCatalogItemMethod                  = "/google.cloud.recommendationengine.v1beta1.CatalogService/DeleteCatalogItem"
	gcpRecommendationEngineImportCatalogItemsMethod                 = "/google.cloud.recommendationengine.v1beta1.CatalogService/ImportCatalogItems"
	gcpRecommendationEngineWriteUserEventMethod                     = "/google.cloud.recommendationengine.v1beta1.UserEventService/WriteUserEvent"
	gcpRecommendationEngineCollectUserEventMethod                   = "/google.cloud.recommendationengine.v1beta1.UserEventService/CollectUserEvent"
	gcpRecommendationEngineListUserEventsMethod                     = "/google.cloud.recommendationengine.v1beta1.UserEventService/ListUserEvents"
	gcpRecommendationEnginePurgeUserEventsMethod                    = "/google.cloud.recommendationengine.v1beta1.UserEventService/PurgeUserEvents"
	gcpRecommendationEngineImportUserEventsMethod                   = "/google.cloud.recommendationengine.v1beta1.UserEventService/ImportUserEvents"
	gcpRecommendationEnginePredictMethod                            = "/google.cloud.recommendationengine.v1beta1.PredictionService/Predict"
	gcpRecommendationEngineCreatePredictionAPIKeyRegistrationMethod = "/google.cloud.recommendationengine.v1beta1.PredictionApiKeyRegistry/CreatePredictionApiKeyRegistration"
	gcpRecommendationEngineListPredictionAPIKeyRegistrationsMethod  = "/google.cloud.recommendationengine.v1beta1.PredictionApiKeyRegistry/ListPredictionApiKeyRegistrations"
	gcpRecommendationEngineDeletePredictionAPIKeyRegistrationMethod = "/google.cloud.recommendationengine.v1beta1.PredictionApiKeyRegistry/DeletePredictionApiKeyRegistration"

	gcpRetailListProductsMethod        = "/google.cloud.retail.v2.ProductService/ListProducts"
	gcpRetailCreateProductMethod       = "/google.cloud.retail.v2.ProductService/CreateProduct"
	gcpRetailSearchMethod              = "/google.cloud.retail.v2.SearchService/Search"
	gcpRetailCreateServingConfigMethod = "/google.cloud.retail.v2.ServingConfigService/CreateServingConfig"
	gcpRetailGetServingConfigMethod    = "/google.cloud.retail.v2.ServingConfigService/GetServingConfig"

	gcpRunListServicesMethod   = "/google.cloud.run.v2.Services/ListServices"
	gcpRunGetServiceMethod     = "/google.cloud.run.v2.Services/GetService"
	gcpRunCreateServiceMethod  = "/google.cloud.run.v2.Services/CreateService"
	gcpRunListJobsMethod       = "/google.cloud.run.v2.Jobs/ListJobs"
	gcpRunGetJobMethod         = "/google.cloud.run.v2.Jobs/GetJob"
	gcpRunCreateJobMethod      = "/google.cloud.run.v2.Jobs/CreateJob"
	gcpRunRunJobMethod         = "/google.cloud.run.v2.Jobs/RunJob"
	gcpRunListExecutionsMethod = "/google.cloud.run.v2.Executions/ListExecutions"
	gcpRunGetExecutionMethod   = "/google.cloud.run.v2.Executions/GetExecution"
	gcpRunListTasksMethod      = "/google.cloud.run.v2.Tasks/ListTasks"
	gcpRunGetTaskMethod        = "/google.cloud.run.v2.Tasks/GetTask"
	gcpRunListRevisionsMethod  = "/google.cloud.run.v2.Revisions/ListRevisions"
	gcpRunGetRevisionMethod    = "/google.cloud.run.v2.Revisions/GetRevision"

	gcpSchedulerListJobsMethod  = "/google.cloud.scheduler.v1.CloudScheduler/ListJobs"
	gcpSchedulerGetJobMethod    = "/google.cloud.scheduler.v1.CloudScheduler/GetJob"
	gcpSchedulerCreateJobMethod = "/google.cloud.scheduler.v1.CloudScheduler/CreateJob"
	gcpSchedulerUpdateJobMethod = "/google.cloud.scheduler.v1.CloudScheduler/UpdateJob"
	gcpSchedulerDeleteJobMethod = "/google.cloud.scheduler.v1.CloudScheduler/DeleteJob"
	gcpSchedulerPauseJobMethod  = "/google.cloud.scheduler.v1.CloudScheduler/PauseJob"
	gcpSchedulerResumeJobMethod = "/google.cloud.scheduler.v1.CloudScheduler/ResumeJob"
	gcpSchedulerRunJobMethod    = "/google.cloud.scheduler.v1.CloudScheduler/RunJob"

	gcpStorageBatchOperationsListJobsMethod             = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/ListJobs"
	gcpStorageBatchOperationsGetJobMethod               = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/GetJob"
	gcpStorageBatchOperationsCreateJobMethod            = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/CreateJob"
	gcpStorageBatchOperationsDeleteJobMethod            = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/DeleteJob"
	gcpStorageBatchOperationsCancelJobMethod            = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/CancelJob"
	gcpStorageBatchOperationsListBucketOperationsMethod = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/ListBucketOperations"
	gcpStorageBatchOperationsGetBucketOperationMethod   = "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/GetBucketOperation"

	gcpStorageInsightsListReportConfigsMethod   = "/google.cloud.storageinsights.v1.StorageInsights/ListReportConfigs"
	gcpStorageInsightsGetReportConfigMethod     = "/google.cloud.storageinsights.v1.StorageInsights/GetReportConfig"
	gcpStorageInsightsCreateReportConfigMethod  = "/google.cloud.storageinsights.v1.StorageInsights/CreateReportConfig"
	gcpStorageInsightsUpdateReportConfigMethod  = "/google.cloud.storageinsights.v1.StorageInsights/UpdateReportConfig"
	gcpStorageInsightsDeleteReportConfigMethod  = "/google.cloud.storageinsights.v1.StorageInsights/DeleteReportConfig"
	gcpStorageInsightsListReportDetailsMethod   = "/google.cloud.storageinsights.v1.StorageInsights/ListReportDetails"
	gcpStorageInsightsGetReportDetailMethod     = "/google.cloud.storageinsights.v1.StorageInsights/GetReportDetail"
	gcpStorageInsightsListDatasetConfigsMethod  = "/google.cloud.storageinsights.v1.StorageInsights/ListDatasetConfigs"
	gcpStorageInsightsGetDatasetConfigMethod    = "/google.cloud.storageinsights.v1.StorageInsights/GetDatasetConfig"
	gcpStorageInsightsCreateDatasetConfigMethod = "/google.cloud.storageinsights.v1.StorageInsights/CreateDatasetConfig"
	gcpStorageInsightsUpdateDatasetConfigMethod = "/google.cloud.storageinsights.v1.StorageInsights/UpdateDatasetConfig"
	gcpStorageInsightsDeleteDatasetConfigMethod = "/google.cloud.storageinsights.v1.StorageInsights/DeleteDatasetConfig"
	gcpStorageInsightsLinkDatasetMethod         = "/google.cloud.storageinsights.v1.StorageInsights/LinkDataset"
	gcpStorageInsightsUnlinkDatasetMethod       = "/google.cloud.storageinsights.v1.StorageInsights/UnlinkDataset"

	gcpStorageTransferGetGoogleServiceAccountMethod = "/google.storagetransfer.v1.StorageTransferService/GetGoogleServiceAccount"
	gcpStorageTransferCreateTransferJobMethod       = "/google.storagetransfer.v1.StorageTransferService/CreateTransferJob"
	gcpStorageTransferUpdateTransferJobMethod       = "/google.storagetransfer.v1.StorageTransferService/UpdateTransferJob"
	gcpStorageTransferGetTransferJobMethod          = "/google.storagetransfer.v1.StorageTransferService/GetTransferJob"
	gcpStorageTransferListTransferJobsMethod        = "/google.storagetransfer.v1.StorageTransferService/ListTransferJobs"
	gcpStorageTransferPauseTransferOperationMethod  = "/google.storagetransfer.v1.StorageTransferService/PauseTransferOperation"
	gcpStorageTransferResumeTransferOperationMethod = "/google.storagetransfer.v1.StorageTransferService/ResumeTransferOperation"
	gcpStorageTransferRunTransferJobMethod          = "/google.storagetransfer.v1.StorageTransferService/RunTransferJob"
	gcpStorageTransferDeleteTransferJobMethod       = "/google.storagetransfer.v1.StorageTransferService/DeleteTransferJob"
	gcpStorageTransferCreateAgentPoolMethod         = "/google.storagetransfer.v1.StorageTransferService/CreateAgentPool"
	gcpStorageTransferUpdateAgentPoolMethod         = "/google.storagetransfer.v1.StorageTransferService/UpdateAgentPool"
	gcpStorageTransferGetAgentPoolMethod            = "/google.storagetransfer.v1.StorageTransferService/GetAgentPool"
	gcpStorageTransferListAgentPoolsMethod          = "/google.storagetransfer.v1.StorageTransferService/ListAgentPools"
	gcpStorageTransferDeleteAgentPoolMethod         = "/google.storagetransfer.v1.StorageTransferService/DeleteAgentPool"

	gcpStreetViewPublishStartUploadMethod              = "/google.streetview.publish.v1.StreetViewPublishService/StartUpload"
	gcpStreetViewPublishCreatePhotoMethod              = "/google.streetview.publish.v1.StreetViewPublishService/CreatePhoto"
	gcpStreetViewPublishGetPhotoMethod                 = "/google.streetview.publish.v1.StreetViewPublishService/GetPhoto"
	gcpStreetViewPublishBatchGetPhotosMethod           = "/google.streetview.publish.v1.StreetViewPublishService/BatchGetPhotos"
	gcpStreetViewPublishListPhotosMethod               = "/google.streetview.publish.v1.StreetViewPublishService/ListPhotos"
	gcpStreetViewPublishUpdatePhotoMethod              = "/google.streetview.publish.v1.StreetViewPublishService/UpdatePhoto"
	gcpStreetViewPublishBatchUpdatePhotosMethod        = "/google.streetview.publish.v1.StreetViewPublishService/BatchUpdatePhotos"
	gcpStreetViewPublishDeletePhotoMethod              = "/google.streetview.publish.v1.StreetViewPublishService/DeletePhoto"
	gcpStreetViewPublishBatchDeletePhotosMethod        = "/google.streetview.publish.v1.StreetViewPublishService/BatchDeletePhotos"
	gcpStreetViewPublishStartPhotoSequenceUploadMethod = "/google.streetview.publish.v1.StreetViewPublishService/StartPhotoSequenceUpload"
	gcpStreetViewPublishCreatePhotoSequenceMethod      = "/google.streetview.publish.v1.StreetViewPublishService/CreatePhotoSequence"
	gcpStreetViewPublishGetPhotoSequenceMethod         = "/google.streetview.publish.v1.StreetViewPublishService/GetPhotoSequence"
	gcpStreetViewPublishListPhotoSequencesMethod       = "/google.streetview.publish.v1.StreetViewPublishService/ListPhotoSequences"
	gcpStreetViewPublishDeletePhotoSequenceMethod      = "/google.streetview.publish.v1.StreetViewPublishService/DeletePhotoSequence"

	gcpSpeechRecognizeMethod            = "/google.cloud.speech.v1.Speech/Recognize"
	gcpSpeechLongRunningRecognizeMethod = "/google.cloud.speech.v1.Speech/LongRunningRecognize"
	gcpSpeechStreamingRecognizeMethod   = "/google.cloud.speech.v1.Speech/StreamingRecognize"

	gcpSpeechCreatePhraseSetMethod   = "/google.cloud.speech.v1.Adaptation/CreatePhraseSet"
	gcpSpeechGetPhraseSetMethod      = "/google.cloud.speech.v1.Adaptation/GetPhraseSet"
	gcpSpeechListPhraseSetMethod     = "/google.cloud.speech.v1.Adaptation/ListPhraseSet"
	gcpSpeechUpdatePhraseSetMethod   = "/google.cloud.speech.v1.Adaptation/UpdatePhraseSet"
	gcpSpeechDeletePhraseSetMethod   = "/google.cloud.speech.v1.Adaptation/DeletePhraseSet"
	gcpSpeechCreateCustomClassMethod = "/google.cloud.speech.v1.Adaptation/CreateCustomClass"
	gcpSpeechGetCustomClassMethod    = "/google.cloud.speech.v1.Adaptation/GetCustomClass"
	gcpSpeechListCustomClassesMethod = "/google.cloud.speech.v1.Adaptation/ListCustomClasses"
	gcpSpeechUpdateCustomClassMethod = "/google.cloud.speech.v1.Adaptation/UpdateCustomClass"
	gcpSpeechDeleteCustomClassMethod = "/google.cloud.speech.v1.Adaptation/DeleteCustomClass"

	gcpSpeechV2CreateRecognizerMethod    = "/google.cloud.speech.v2.Speech/CreateRecognizer"
	gcpSpeechV2ListRecognizersMethod     = "/google.cloud.speech.v2.Speech/ListRecognizers"
	gcpSpeechV2GetRecognizerMethod       = "/google.cloud.speech.v2.Speech/GetRecognizer"
	gcpSpeechV2UpdateRecognizerMethod    = "/google.cloud.speech.v2.Speech/UpdateRecognizer"
	gcpSpeechV2DeleteRecognizerMethod    = "/google.cloud.speech.v2.Speech/DeleteRecognizer"
	gcpSpeechV2UndeleteRecognizerMethod  = "/google.cloud.speech.v2.Speech/UndeleteRecognizer"
	gcpSpeechV2RecognizeMethod           = "/google.cloud.speech.v2.Speech/Recognize"
	gcpSpeechV2StreamingRecognizeMethod  = "/google.cloud.speech.v2.Speech/StreamingRecognize"
	gcpSpeechV2BatchRecognizeMethod      = "/google.cloud.speech.v2.Speech/BatchRecognize"
	gcpSpeechV2GetConfigMethod           = "/google.cloud.speech.v2.Speech/GetConfig"
	gcpSpeechV2UpdateConfigMethod        = "/google.cloud.speech.v2.Speech/UpdateConfig"
	gcpSpeechV2CreateCustomClassMethod   = "/google.cloud.speech.v2.Speech/CreateCustomClass"
	gcpSpeechV2ListCustomClassesMethod   = "/google.cloud.speech.v2.Speech/ListCustomClasses"
	gcpSpeechV2GetCustomClassMethod      = "/google.cloud.speech.v2.Speech/GetCustomClass"
	gcpSpeechV2UpdateCustomClassMethod   = "/google.cloud.speech.v2.Speech/UpdateCustomClass"
	gcpSpeechV2DeleteCustomClassMethod   = "/google.cloud.speech.v2.Speech/DeleteCustomClass"
	gcpSpeechV2UndeleteCustomClassMethod = "/google.cloud.speech.v2.Speech/UndeleteCustomClass"
	gcpSpeechV2CreatePhraseSetMethod     = "/google.cloud.speech.v2.Speech/CreatePhraseSet"
	gcpSpeechV2ListPhraseSetsMethod      = "/google.cloud.speech.v2.Speech/ListPhraseSets"
	gcpSpeechV2GetPhraseSetMethod        = "/google.cloud.speech.v2.Speech/GetPhraseSet"
	gcpSpeechV2UpdatePhraseSetMethod     = "/google.cloud.speech.v2.Speech/UpdatePhraseSet"
	gcpSpeechV2DeletePhraseSetMethod     = "/google.cloud.speech.v2.Speech/DeletePhraseSet"
	gcpSpeechV2UndeletePhraseSetMethod   = "/google.cloud.speech.v2.Speech/UndeletePhraseSet"

	gcpSpannerCreateSessionMethod                        = "/google.spanner.v1.Spanner/CreateSession"
	gcpSpannerBatchCreateSessionsMethod                  = "/google.spanner.v1.Spanner/BatchCreateSessions"
	gcpSpannerGetSessionMethod                           = "/google.spanner.v1.Spanner/GetSession"
	gcpSpannerListSessionsMethod                         = "/google.spanner.v1.Spanner/ListSessions"
	gcpSpannerDeleteSessionMethod                        = "/google.spanner.v1.Spanner/DeleteSession"
	gcpSpannerExecuteSQLMethod                           = "/google.spanner.v1.Spanner/ExecuteSql"
	gcpSpannerExecuteStreamingSQLMethod                  = "/google.spanner.v1.Spanner/ExecuteStreamingSql"
	gcpSpannerExecuteBatchDMLMethod                      = "/google.spanner.v1.Spanner/ExecuteBatchDml"
	gcpSpannerReadMethod                                 = "/google.spanner.v1.Spanner/Read"
	gcpSpannerStreamingReadMethod                        = "/google.spanner.v1.Spanner/StreamingRead"
	gcpSpannerBeginTransactionMethod                     = "/google.spanner.v1.Spanner/BeginTransaction"
	gcpSpannerCommitMethod                               = "/google.spanner.v1.Spanner/Commit"
	gcpSpannerRollbackMethod                             = "/google.spanner.v1.Spanner/Rollback"
	gcpSpannerPartitionQueryMethod                       = "/google.spanner.v1.Spanner/PartitionQuery"
	gcpSpannerPartitionReadMethod                        = "/google.spanner.v1.Spanner/PartitionRead"
	gcpSpannerBatchWriteMethod                           = "/google.spanner.v1.Spanner/BatchWrite"
	gcpSpannerAdapterCreateSessionMethod                 = "/google.spanner.adapter.v1.Adapter/CreateSession"
	gcpSpannerAdapterAdaptMessageMethod                  = "/google.spanner.adapter.v1.Adapter/AdaptMessage"
	gcpSpannerExecutorExecuteActionAsyncMethod           = "/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync"
	gcpSpannerAdminDatabaseListDatabasesMethod           = "/google.spanner.admin.database.v1.DatabaseAdmin/ListDatabases"
	gcpSpannerAdminDatabaseCreateDatabaseMethod          = "/google.spanner.admin.database.v1.DatabaseAdmin/CreateDatabase"
	gcpSpannerAdminDatabaseGetDatabaseMethod             = "/google.spanner.admin.database.v1.DatabaseAdmin/GetDatabase"
	gcpSpannerAdminDatabaseUpdateDatabaseMethod          = "/google.spanner.admin.database.v1.DatabaseAdmin/UpdateDatabase"
	gcpSpannerAdminDatabaseUpdateDatabaseDDLMethod       = "/google.spanner.admin.database.v1.DatabaseAdmin/UpdateDatabaseDdl"
	gcpSpannerAdminDatabaseDropDatabaseMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/DropDatabase"
	gcpSpannerAdminDatabaseGetDatabaseDDLMethod          = "/google.spanner.admin.database.v1.DatabaseAdmin/GetDatabaseDdl"
	gcpSpannerAdminDatabaseSetIAMPolicyMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/SetIamPolicy"
	gcpSpannerAdminDatabaseGetIAMPolicyMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/GetIamPolicy"
	gcpSpannerAdminDatabaseTestIAMPermissionsMethod      = "/google.spanner.admin.database.v1.DatabaseAdmin/TestIamPermissions"
	gcpSpannerAdminDatabaseCreateBackupMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/CreateBackup"
	gcpSpannerAdminDatabaseCopyBackupMethod              = "/google.spanner.admin.database.v1.DatabaseAdmin/CopyBackup"
	gcpSpannerAdminDatabaseGetBackupMethod               = "/google.spanner.admin.database.v1.DatabaseAdmin/GetBackup"
	gcpSpannerAdminDatabaseUpdateBackupMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/UpdateBackup"
	gcpSpannerAdminDatabaseDeleteBackupMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/DeleteBackup"
	gcpSpannerAdminDatabaseListBackupsMethod             = "/google.spanner.admin.database.v1.DatabaseAdmin/ListBackups"
	gcpSpannerAdminDatabaseRestoreDatabaseMethod         = "/google.spanner.admin.database.v1.DatabaseAdmin/RestoreDatabase"
	gcpSpannerAdminDatabaseListDatabaseOperationsMethod  = "/google.spanner.admin.database.v1.DatabaseAdmin/ListDatabaseOperations"
	gcpSpannerAdminDatabaseListBackupOperationsMethod    = "/google.spanner.admin.database.v1.DatabaseAdmin/ListBackupOperations"
	gcpSpannerAdminDatabaseListDatabaseRolesMethod       = "/google.spanner.admin.database.v1.DatabaseAdmin/ListDatabaseRoles"
	gcpSpannerAdminDatabaseAddSplitPointsMethod          = "/google.spanner.admin.database.v1.DatabaseAdmin/AddSplitPoints"
	gcpSpannerAdminDatabaseCreateBackupScheduleMethod    = "/google.spanner.admin.database.v1.DatabaseAdmin/CreateBackupSchedule"
	gcpSpannerAdminDatabaseGetBackupScheduleMethod       = "/google.spanner.admin.database.v1.DatabaseAdmin/GetBackupSchedule"
	gcpSpannerAdminDatabaseUpdateBackupScheduleMethod    = "/google.spanner.admin.database.v1.DatabaseAdmin/UpdateBackupSchedule"
	gcpSpannerAdminDatabaseDeleteBackupScheduleMethod    = "/google.spanner.admin.database.v1.DatabaseAdmin/DeleteBackupSchedule"
	gcpSpannerAdminDatabaseListBackupSchedulesMethod     = "/google.spanner.admin.database.v1.DatabaseAdmin/ListBackupSchedules"
	gcpSpannerAdminDatabaseInternalUpdateGraphOpMethod   = "/google.spanner.admin.database.v1.DatabaseAdmin/InternalUpdateGraphOperation"
	gcpSpannerAdminDatabaseCancelOperationMethod         = "/google.spanner.admin.database.v1.DatabaseAdmin/CancelOperation"
	gcpSpannerAdminDatabaseDeleteOperationMethod         = "/google.spanner.admin.database.v1.DatabaseAdmin/DeleteOperation"
	gcpSpannerAdminDatabaseGetOperationMethod            = "/google.spanner.admin.database.v1.DatabaseAdmin/GetOperation"
	gcpSpannerAdminDatabaseListOperationsMethod          = "/google.spanner.admin.database.v1.DatabaseAdmin/ListOperations"
	gcpSpannerAdminInstanceListInstanceConfigsMethod     = "/google.spanner.admin.instance.v1.InstanceAdmin/ListInstanceConfigs"
	gcpSpannerAdminInstanceGetInstanceConfigMethod       = "/google.spanner.admin.instance.v1.InstanceAdmin/GetInstanceConfig"
	gcpSpannerAdminInstanceCreateInstanceConfigMethod    = "/google.spanner.admin.instance.v1.InstanceAdmin/CreateInstanceConfig"
	gcpSpannerAdminInstanceUpdateInstanceConfigMethod    = "/google.spanner.admin.instance.v1.InstanceAdmin/UpdateInstanceConfig"
	gcpSpannerAdminInstanceDeleteInstanceConfigMethod    = "/google.spanner.admin.instance.v1.InstanceAdmin/DeleteInstanceConfig"
	gcpSpannerAdminInstanceListInstanceConfigOpsMethod   = "/google.spanner.admin.instance.v1.InstanceAdmin/ListInstanceConfigOperations"
	gcpSpannerAdminInstanceListInstancesMethod           = "/google.spanner.admin.instance.v1.InstanceAdmin/ListInstances"
	gcpSpannerAdminInstanceGetInstanceMethod             = "/google.spanner.admin.instance.v1.InstanceAdmin/GetInstance"
	gcpSpannerAdminInstanceCreateInstanceMethod          = "/google.spanner.admin.instance.v1.InstanceAdmin/CreateInstance"
	gcpSpannerAdminInstanceUpdateInstanceMethod          = "/google.spanner.admin.instance.v1.InstanceAdmin/UpdateInstance"
	gcpSpannerAdminInstanceDeleteInstanceMethod          = "/google.spanner.admin.instance.v1.InstanceAdmin/DeleteInstance"
	gcpSpannerAdminInstanceListInstancePartitionsMethod  = "/google.spanner.admin.instance.v1.InstanceAdmin/ListInstancePartitions"
	gcpSpannerAdminInstanceGetInstancePartitionMethod    = "/google.spanner.admin.instance.v1.InstanceAdmin/GetInstancePartition"
	gcpSpannerAdminInstanceCreateInstancePartitionMethod = "/google.spanner.admin.instance.v1.InstanceAdmin/CreateInstancePartition"
	gcpSpannerAdminInstanceUpdateInstancePartitionMethod = "/google.spanner.admin.instance.v1.InstanceAdmin/UpdateInstancePartition"
	gcpSpannerAdminInstanceDeleteInstancePartitionMethod = "/google.spanner.admin.instance.v1.InstanceAdmin/DeleteInstancePartition"
	gcpSpannerAdminInstanceListPartitionOpsMethod        = "/google.spanner.admin.instance.v1.InstanceAdmin/ListInstancePartitionOperations"
	gcpSpannerAdminInstanceMoveInstanceMethod            = "/google.spanner.admin.instance.v1.InstanceAdmin/MoveInstance"
	gcpSpannerAdminInstanceSetIAMPolicyMethod            = "/google.spanner.admin.instance.v1.InstanceAdmin/SetIamPolicy"
	gcpSpannerAdminInstanceGetIAMPolicyMethod            = "/google.spanner.admin.instance.v1.InstanceAdmin/GetIamPolicy"
	gcpSpannerAdminInstanceTestIAMPermissionsMethod      = "/google.spanner.admin.instance.v1.InstanceAdmin/TestIamPermissions"
	gcpSpannerAdminInstanceCancelOperationMethod         = "/google.spanner.admin.instance.v1.InstanceAdmin/CancelOperation"
	gcpSpannerAdminInstanceDeleteOperationMethod         = "/google.spanner.admin.instance.v1.InstanceAdmin/DeleteOperation"
	gcpSpannerAdminInstanceGetOperationMethod            = "/google.spanner.admin.instance.v1.InstanceAdmin/GetOperation"
	gcpSpannerAdminInstanceListOperationsMethod          = "/google.spanner.admin.instance.v1.InstanceAdmin/ListOperations"

	gcpShellGetEnvironmentMethod       = "/google.cloud.shell.v1.CloudShellService/GetEnvironment"
	gcpShellStartEnvironmentMethod     = "/google.cloud.shell.v1.CloudShellService/StartEnvironment"
	gcpShellAuthorizeEnvironmentMethod = "/google.cloud.shell.v1.CloudShellService/AuthorizeEnvironment"
	gcpShellAddPublicKeyMethod         = "/google.cloud.shell.v1.CloudShellService/AddPublicKey"
	gcpShellRemovePublicKeyMethod      = "/google.cloud.shell.v1.CloudShellService/RemovePublicKey"

	gcpShoppingCSSListChildAccountsMethod     = "/google.shopping.css.v1.AccountsService/ListChildAccounts"
	gcpShoppingCSSGetAccountMethod            = "/google.shopping.css.v1.AccountsService/GetAccount"
	gcpShoppingCSSUpdateLabelsMethod          = "/google.shopping.css.v1.AccountsService/UpdateLabels"
	gcpShoppingCSSListAccountLabelsMethod     = "/google.shopping.css.v1.AccountLabelsService/ListAccountLabels"
	gcpShoppingCSSCreateAccountLabelMethod    = "/google.shopping.css.v1.AccountLabelsService/CreateAccountLabel"
	gcpShoppingCSSUpdateAccountLabelMethod    = "/google.shopping.css.v1.AccountLabelsService/UpdateAccountLabel"
	gcpShoppingCSSDeleteAccountLabelMethod    = "/google.shopping.css.v1.AccountLabelsService/DeleteAccountLabel"
	gcpShoppingCSSGetCssProductMethod         = "/google.shopping.css.v1.CssProductsService/GetCssProduct"
	gcpShoppingCSSListCssProductsMethod       = "/google.shopping.css.v1.CssProductsService/ListCssProducts"
	gcpShoppingCSSInsertCssProductInputMethod = "/google.shopping.css.v1.CssProductInputsService/InsertCssProductInput"
	gcpShoppingCSSUpdateCssProductInputMethod = "/google.shopping.css.v1.CssProductInputsService/UpdateCssProductInput"
	gcpShoppingCSSDeleteCssProductInputMethod = "/google.shopping.css.v1.CssProductInputsService/DeleteCssProductInput"
	gcpShoppingCSSListQuotaGroupsMethod       = "/google.shopping.css.v1.QuotaService/ListQuotaGroups"

	gcpShoppingMerchantConversionsCreateConversionSourceMethod   = "/google.shopping.merchant.conversions.v1.ConversionSourcesService/CreateConversionSource"
	gcpShoppingMerchantConversionsUpdateConversionSourceMethod   = "/google.shopping.merchant.conversions.v1.ConversionSourcesService/UpdateConversionSource"
	gcpShoppingMerchantConversionsDeleteConversionSourceMethod   = "/google.shopping.merchant.conversions.v1.ConversionSourcesService/DeleteConversionSource"
	gcpShoppingMerchantConversionsUndeleteConversionSourceMethod = "/google.shopping.merchant.conversions.v1.ConversionSourcesService/UndeleteConversionSource"
	gcpShoppingMerchantConversionsGetConversionSourceMethod      = "/google.shopping.merchant.conversions.v1.ConversionSourcesService/GetConversionSource"
	gcpShoppingMerchantConversionsListConversionSourcesMethod    = "/google.shopping.merchant.conversions.v1.ConversionSourcesService/ListConversionSources"

	gcpShoppingMerchantDatasourcesGetDataSourceMethod    = "/google.shopping.merchant.datasources.v1.DataSourcesService/GetDataSource"
	gcpShoppingMerchantDatasourcesListDataSourcesMethod  = "/google.shopping.merchant.datasources.v1.DataSourcesService/ListDataSources"
	gcpShoppingMerchantDatasourcesCreateDataSourceMethod = "/google.shopping.merchant.datasources.v1.DataSourcesService/CreateDataSource"
	gcpShoppingMerchantDatasourcesUpdateDataSourceMethod = "/google.shopping.merchant.datasources.v1.DataSourcesService/UpdateDataSource"
	gcpShoppingMerchantDatasourcesDeleteDataSourceMethod = "/google.shopping.merchant.datasources.v1.DataSourcesService/DeleteDataSource"
	gcpShoppingMerchantDatasourcesFetchDataSourceMethod  = "/google.shopping.merchant.datasources.v1.DataSourcesService/FetchDataSource"
	gcpShoppingMerchantDatasourcesGetFileUploadMethod    = "/google.shopping.merchant.datasources.v1.FileUploadsService/GetFileUpload"

	gcpShoppingMerchantInventoriesListLocalInventoriesMethod    = "/google.shopping.merchant.inventories.v1.LocalInventoryService/ListLocalInventories"
	gcpShoppingMerchantInventoriesInsertLocalInventoryMethod    = "/google.shopping.merchant.inventories.v1.LocalInventoryService/InsertLocalInventory"
	gcpShoppingMerchantInventoriesDeleteLocalInventoryMethod    = "/google.shopping.merchant.inventories.v1.LocalInventoryService/DeleteLocalInventory"
	gcpShoppingMerchantInventoriesListRegionalInventoriesMethod = "/google.shopping.merchant.inventories.v1.RegionalInventoryService/ListRegionalInventories"
	gcpShoppingMerchantInventoriesInsertRegionalInventoryMethod = "/google.shopping.merchant.inventories.v1.RegionalInventoryService/InsertRegionalInventory"
	gcpShoppingMerchantInventoriesDeleteRegionalInventoryMethod = "/google.shopping.merchant.inventories.v1.RegionalInventoryService/DeleteRegionalInventory"

	gcpShoppingMerchantIssueresolutionRenderAccountIssuesMethod          = "/google.shopping.merchant.issueresolution.v1.IssueResolutionService/RenderAccountIssues"
	gcpShoppingMerchantIssueresolutionRenderProductIssuesMethod          = "/google.shopping.merchant.issueresolution.v1.IssueResolutionService/RenderProductIssues"
	gcpShoppingMerchantIssueresolutionTriggerActionMethod                = "/google.shopping.merchant.issueresolution.v1.IssueResolutionService/TriggerAction"
	gcpShoppingMerchantIssueresolutionListAggregateProductStatusesMethod = "/google.shopping.merchant.issueresolution.v1.AggregateProductStatusesService/ListAggregateProductStatuses"

	gcpShoppingMerchantNotificationsGetNotificationSubscriptionMethod              = "/google.shopping.merchant.notifications.v1.NotificationsApiService/GetNotificationSubscription"
	gcpShoppingMerchantNotificationsCreateNotificationSubscriptionMethod           = "/google.shopping.merchant.notifications.v1.NotificationsApiService/CreateNotificationSubscription"
	gcpShoppingMerchantNotificationsUpdateNotificationSubscriptionMethod           = "/google.shopping.merchant.notifications.v1.NotificationsApiService/UpdateNotificationSubscription"
	gcpShoppingMerchantNotificationsDeleteNotificationSubscriptionMethod           = "/google.shopping.merchant.notifications.v1.NotificationsApiService/DeleteNotificationSubscription"
	gcpShoppingMerchantNotificationsListNotificationSubscriptionsMethod            = "/google.shopping.merchant.notifications.v1.NotificationsApiService/ListNotificationSubscriptions"
	gcpShoppingMerchantNotificationsGetNotificationSubscriptionHealthMetricsMethod = "/google.shopping.merchant.notifications.v1.NotificationsApiService/GetNotificationSubscriptionHealthMetrics"
	gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod                = "/google.shopping.merchant.ordertracking.v1.OrderTrackingSignalsService/CreateOrderTrackingSignal"
	gcpShoppingMerchantPromotionsInsertPromotionMethod                             = "/google.shopping.merchant.promotions.v1.PromotionsService/InsertPromotion"
	gcpShoppingMerchantPromotionsGetPromotionMethod                                = "/google.shopping.merchant.promotions.v1.PromotionsService/GetPromotion"
	gcpShoppingMerchantPromotionsListPromotionsMethod                              = "/google.shopping.merchant.promotions.v1.PromotionsService/ListPromotions"
	gcpShoppingMerchantReportsSearchMethod                                         = "/google.shopping.merchant.reports.v1.ReportService/Search"
	gcpShoppingMerchantReviewsGetMerchantReviewMethod                              = "/google.shopping.merchant.reviews.v1beta.MerchantReviewsService/GetMerchantReview"
	gcpShoppingMerchantReviewsListMerchantReviewsMethod                            = "/google.shopping.merchant.reviews.v1beta.MerchantReviewsService/ListMerchantReviews"
	gcpShoppingMerchantReviewsInsertMerchantReviewMethod                           = "/google.shopping.merchant.reviews.v1beta.MerchantReviewsService/InsertMerchantReview"
	gcpShoppingMerchantReviewsDeleteMerchantReviewMethod                           = "/google.shopping.merchant.reviews.v1beta.MerchantReviewsService/DeleteMerchantReview"
	gcpShoppingMerchantReviewsGetProductReviewMethod                               = "/google.shopping.merchant.reviews.v1beta.ProductReviewsService/GetProductReview"
	gcpShoppingMerchantReviewsListProductReviewsMethod                             = "/google.shopping.merchant.reviews.v1beta.ProductReviewsService/ListProductReviews"
	gcpShoppingMerchantReviewsInsertProductReviewMethod                            = "/google.shopping.merchant.reviews.v1beta.ProductReviewsService/InsertProductReview"
	gcpShoppingMerchantReviewsDeleteProductReviewMethod                            = "/google.shopping.merchant.reviews.v1beta.ProductReviewsService/DeleteProductReview"
	gcpShoppingMerchantQuotaListQuotaGroupsMethod                                  = "/google.shopping.merchant.quota.v1.QuotaService/ListQuotaGroups"
	gcpShoppingMerchantProductsGetProductMethod                                    = "/google.shopping.merchant.products.v1.ProductsService/GetProduct"
	gcpShoppingMerchantProductsListProductsMethod                                  = "/google.shopping.merchant.products.v1.ProductsService/ListProducts"
	gcpShoppingMerchantProductsInsertProductInputMethod                            = "/google.shopping.merchant.products.v1.ProductInputsService/InsertProductInput"
	gcpShoppingMerchantProductsUpdateProductInputMethod                            = "/google.shopping.merchant.products.v1.ProductInputsService/UpdateProductInput"
	gcpShoppingMerchantProductsDeleteProductInputMethod                            = "/google.shopping.merchant.products.v1.ProductInputsService/DeleteProductInput"
	gcpShoppingMerchantProductstudioGenerateProductImageBackgroundMethod           = "/google.shopping.merchant.productstudio.v1alpha.ImageService/GenerateProductImageBackground"
	gcpShoppingMerchantProductstudioRemoveProductImageBackgroundMethod             = "/google.shopping.merchant.productstudio.v1alpha.ImageService/RemoveProductImageBackground"
	gcpShoppingMerchantProductstudioUpscaleProductImageMethod                      = "/google.shopping.merchant.productstudio.v1alpha.ImageService/UpscaleProductImage"
	gcpShoppingMerchantProductstudioGenerateProductTextSuggestionsMethod           = "/google.shopping.merchant.productstudio.v1alpha.TextSuggestionsService/GenerateProductTextSuggestions"

	gcpShoppingMerchantLFPGetLfpStoreMethod         = "/google.shopping.merchant.lfp.v1.LfpStoreService/GetLfpStore"
	gcpShoppingMerchantLFPInsertLfpStoreMethod      = "/google.shopping.merchant.lfp.v1.LfpStoreService/InsertLfpStore"
	gcpShoppingMerchantLFPDeleteLfpStoreMethod      = "/google.shopping.merchant.lfp.v1.LfpStoreService/DeleteLfpStore"
	gcpShoppingMerchantLFPListLfpStoresMethod       = "/google.shopping.merchant.lfp.v1.LfpStoreService/ListLfpStores"
	gcpShoppingMerchantLFPInsertLfpInventoryMethod  = "/google.shopping.merchant.lfp.v1.LfpInventoryService/InsertLfpInventory"
	gcpShoppingMerchantLFPInsertLfpSaleMethod       = "/google.shopping.merchant.lfp.v1.LfpSaleService/InsertLfpSale"
	gcpShoppingMerchantLFPGetLfpMerchantStateMethod = "/google.shopping.merchant.lfp.v1.LfpMerchantStateService/GetLfpMerchantState"

	gcpServiceControlCheckMethod         = "/google.api.servicecontrol.v1.ServiceController/Check"
	gcpServiceControlReportMethod        = "/google.api.servicecontrol.v1.ServiceController/Report"
	gcpServiceControlAllocateQuotaMethod = "/google.api.servicecontrol.v1.QuotaController/AllocateQuota"

	gcpWebRiskComputeThreatListDiffMethod = "/google.cloud.webrisk.v1.WebRiskService/ComputeThreatListDiff"
	gcpWebRiskSearchUrisMethod            = "/google.cloud.webrisk.v1.WebRiskService/SearchUris"
	gcpWebRiskSearchHashesMethod          = "/google.cloud.webrisk.v1.WebRiskService/SearchHashes"
	gcpWebRiskCreateSubmissionMethod      = "/google.cloud.webrisk.v1.WebRiskService/CreateSubmission"
	gcpWebRiskSubmitURIMethod             = "/google.cloud.webrisk.v1.WebRiskService/SubmitUri"

	gcpServiceUsageEnableServiceMethod       = "/google.api.serviceusage.v1.ServiceUsage/EnableService"
	gcpServiceUsageDisableServiceMethod      = "/google.api.serviceusage.v1.ServiceUsage/DisableService"
	gcpServiceUsageGetServiceMethod          = "/google.api.serviceusage.v1.ServiceUsage/GetService"
	gcpServiceUsageListServicesMethod        = "/google.api.serviceusage.v1.ServiceUsage/ListServices"
	gcpServiceUsageBatchEnableServicesMethod = "/google.api.serviceusage.v1.ServiceUsage/BatchEnableServices"
	gcpServiceUsageBatchGetServicesMethod    = "/google.api.serviceusage.v1.ServiceUsage/BatchGetServices"

	gcpSupportGetCaseMethod                   = "/google.cloud.support.v2.CaseService/GetCase"
	gcpSupportListCasesMethod                 = "/google.cloud.support.v2.CaseService/ListCases"
	gcpSupportSearchCasesMethod               = "/google.cloud.support.v2.CaseService/SearchCases"
	gcpSupportCreateCaseMethod                = "/google.cloud.support.v2.CaseService/CreateCase"
	gcpSupportUpdateCaseMethod                = "/google.cloud.support.v2.CaseService/UpdateCase"
	gcpSupportEscalateCaseMethod              = "/google.cloud.support.v2.CaseService/EscalateCase"
	gcpSupportCloseCaseMethod                 = "/google.cloud.support.v2.CaseService/CloseCase"
	gcpSupportSearchCaseClassificationsMethod = "/google.cloud.support.v2.CaseService/SearchCaseClassifications"
	gcpSupportListCommentsMethod              = "/google.cloud.support.v2.CommentService/ListComments"
	gcpSupportCreateCommentMethod             = "/google.cloud.support.v2.CommentService/CreateComment"
	gcpSupportListAttachmentsMethod           = "/google.cloud.support.v2.CaseAttachmentService/ListAttachments"

	gcpTalentCreateCompanyMethod      = "/google.cloud.talent.v4.CompanyService/CreateCompany"
	gcpTalentGetCompanyMethod         = "/google.cloud.talent.v4.CompanyService/GetCompany"
	gcpTalentUpdateCompanyMethod      = "/google.cloud.talent.v4.CompanyService/UpdateCompany"
	gcpTalentDeleteCompanyMethod      = "/google.cloud.talent.v4.CompanyService/DeleteCompany"
	gcpTalentListCompaniesMethod      = "/google.cloud.talent.v4.CompanyService/ListCompanies"
	gcpTalentCreateTenantMethod       = "/google.cloud.talent.v4.TenantService/CreateTenant"
	gcpTalentGetTenantMethod          = "/google.cloud.talent.v4.TenantService/GetTenant"
	gcpTalentUpdateTenantMethod       = "/google.cloud.talent.v4.TenantService/UpdateTenant"
	gcpTalentDeleteTenantMethod       = "/google.cloud.talent.v4.TenantService/DeleteTenant"
	gcpTalentListTenantsMethod        = "/google.cloud.talent.v4.TenantService/ListTenants"
	gcpTalentCreateJobMethod          = "/google.cloud.talent.v4.JobService/CreateJob"
	gcpTalentBatchCreateJobsMethod    = "/google.cloud.talent.v4.JobService/BatchCreateJobs"
	gcpTalentGetJobMethod             = "/google.cloud.talent.v4.JobService/GetJob"
	gcpTalentUpdateJobMethod          = "/google.cloud.talent.v4.JobService/UpdateJob"
	gcpTalentBatchUpdateJobsMethod    = "/google.cloud.talent.v4.JobService/BatchUpdateJobs"
	gcpTalentDeleteJobMethod          = "/google.cloud.talent.v4.JobService/DeleteJob"
	gcpTalentBatchDeleteJobsMethod    = "/google.cloud.talent.v4.JobService/BatchDeleteJobs"
	gcpTalentListJobsMethod           = "/google.cloud.talent.v4.JobService/ListJobs"
	gcpTalentSearchJobsMethod         = "/google.cloud.talent.v4.JobService/SearchJobs"
	gcpTalentSearchJobsForAlertMethod = "/google.cloud.talent.v4.JobService/SearchJobsForAlert"
	gcpTalentCompleteQueryMethod      = "/google.cloud.talent.v4.Completion/CompleteQuery"
	gcpTalentCreateClientEventMethod  = "/google.cloud.talent.v4.EventService/CreateClientEvent"

	gcpServiceManagementListServicesMethod         = "/google.api.servicemanagement.v1.ServiceManager/ListServices"
	gcpServiceManagementGetServiceMethod           = "/google.api.servicemanagement.v1.ServiceManager/GetService"
	gcpServiceManagementCreateServiceMethod        = "/google.api.servicemanagement.v1.ServiceManager/CreateService"
	gcpServiceManagementDeleteServiceMethod        = "/google.api.servicemanagement.v1.ServiceManager/DeleteService"
	gcpServiceManagementUndeleteServiceMethod      = "/google.api.servicemanagement.v1.ServiceManager/UndeleteService"
	gcpServiceManagementListServiceConfigsMethod   = "/google.api.servicemanagement.v1.ServiceManager/ListServiceConfigs"
	gcpServiceManagementGetServiceConfigMethod     = "/google.api.servicemanagement.v1.ServiceManager/GetServiceConfig"
	gcpServiceManagementCreateServiceConfigMethod  = "/google.api.servicemanagement.v1.ServiceManager/CreateServiceConfig"
	gcpServiceManagementSubmitConfigSourceMethod   = "/google.api.servicemanagement.v1.ServiceManager/SubmitConfigSource"
	gcpServiceManagementListServiceRolloutsMethod  = "/google.api.servicemanagement.v1.ServiceManager/ListServiceRollouts"
	gcpServiceManagementGetServiceRolloutMethod    = "/google.api.servicemanagement.v1.ServiceManager/GetServiceRollout"
	gcpServiceManagementCreateServiceRolloutMethod = "/google.api.servicemanagement.v1.ServiceManager/CreateServiceRollout"
	gcpServiceManagementGenerateConfigReportMethod = "/google.api.servicemanagement.v1.ServiceManager/GenerateConfigReport"

	gcpServiceHealthListEventsMethod              = "/google.cloud.servicehealth.v1.ServiceHealth/ListEvents"
	gcpServiceHealthGetEventMethod                = "/google.cloud.servicehealth.v1.ServiceHealth/GetEvent"
	gcpServiceHealthListOrganizationEventsMethod  = "/google.cloud.servicehealth.v1.ServiceHealth/ListOrganizationEvents"
	gcpServiceHealthGetOrganizationEventMethod    = "/google.cloud.servicehealth.v1.ServiceHealth/GetOrganizationEvent"
	gcpServiceHealthListOrganizationImpactsMethod = "/google.cloud.servicehealth.v1.ServiceHealth/ListOrganizationImpacts"
	gcpServiceHealthGetOrganizationImpactMethod   = "/google.cloud.servicehealth.v1.ServiceHealth/GetOrganizationImpact"

	gcpServiceDirectoryCreateNamespaceMethod    = "/google.cloud.servicedirectory.v1.RegistrationService/CreateNamespace"
	gcpServiceDirectoryListNamespacesMethod     = "/google.cloud.servicedirectory.v1.RegistrationService/ListNamespaces"
	gcpServiceDirectoryGetNamespaceMethod       = "/google.cloud.servicedirectory.v1.RegistrationService/GetNamespace"
	gcpServiceDirectoryUpdateNamespaceMethod    = "/google.cloud.servicedirectory.v1.RegistrationService/UpdateNamespace"
	gcpServiceDirectoryDeleteNamespaceMethod    = "/google.cloud.servicedirectory.v1.RegistrationService/DeleteNamespace"
	gcpServiceDirectoryCreateServiceMethod      = "/google.cloud.servicedirectory.v1.RegistrationService/CreateService"
	gcpServiceDirectoryListServicesMethod       = "/google.cloud.servicedirectory.v1.RegistrationService/ListServices"
	gcpServiceDirectoryGetServiceMethod         = "/google.cloud.servicedirectory.v1.RegistrationService/GetService"
	gcpServiceDirectoryUpdateServiceMethod      = "/google.cloud.servicedirectory.v1.RegistrationService/UpdateService"
	gcpServiceDirectoryDeleteServiceMethod      = "/google.cloud.servicedirectory.v1.RegistrationService/DeleteService"
	gcpServiceDirectoryCreateEndpointMethod     = "/google.cloud.servicedirectory.v1.RegistrationService/CreateEndpoint"
	gcpServiceDirectoryListEndpointsMethod      = "/google.cloud.servicedirectory.v1.RegistrationService/ListEndpoints"
	gcpServiceDirectoryGetEndpointMethod        = "/google.cloud.servicedirectory.v1.RegistrationService/GetEndpoint"
	gcpServiceDirectoryUpdateEndpointMethod     = "/google.cloud.servicedirectory.v1.RegistrationService/UpdateEndpoint"
	gcpServiceDirectoryDeleteEndpointMethod     = "/google.cloud.servicedirectory.v1.RegistrationService/DeleteEndpoint"
	gcpServiceDirectoryGetIAMPolicyMethod       = "/google.cloud.servicedirectory.v1.RegistrationService/GetIamPolicy"
	gcpServiceDirectorySetIAMPolicyMethod       = "/google.cloud.servicedirectory.v1.RegistrationService/SetIamPolicy"
	gcpServiceDirectoryTestIAMPermissionsMethod = "/google.cloud.servicedirectory.v1.RegistrationService/TestIamPermissions"
	gcpServiceDirectoryResolveServiceMethod     = "/google.cloud.servicedirectory.v1.LookupService/ResolveService"

	gcpSecretManagerListSecretsMethod          = "/google.cloud.secretmanager.v1.SecretManagerService/ListSecrets"
	gcpSecretManagerCreateSecretMethod         = "/google.cloud.secretmanager.v1.SecretManagerService/CreateSecret"
	gcpSecretManagerAddSecretVersionMethod     = "/google.cloud.secretmanager.v1.SecretManagerService/AddSecretVersion"
	gcpSecretManagerGetSecretMethod            = "/google.cloud.secretmanager.v1.SecretManagerService/GetSecret"
	gcpSecretManagerUpdateSecretMethod         = "/google.cloud.secretmanager.v1.SecretManagerService/UpdateSecret"
	gcpSecretManagerDeleteSecretMethod         = "/google.cloud.secretmanager.v1.SecretManagerService/DeleteSecret"
	gcpSecretManagerListSecretVersionsMethod   = "/google.cloud.secretmanager.v1.SecretManagerService/ListSecretVersions"
	gcpSecretManagerGetSecretVersionMethod     = "/google.cloud.secretmanager.v1.SecretManagerService/GetSecretVersion"
	gcpSecretManagerAccessSecretVersionMethod  = "/google.cloud.secretmanager.v1.SecretManagerService/AccessSecretVersion"
	gcpSecretManagerDisableSecretVersionMethod = "/google.cloud.secretmanager.v1.SecretManagerService/DisableSecretVersion"
	gcpSecretManagerEnableSecretVersionMethod  = "/google.cloud.secretmanager.v1.SecretManagerService/EnableSecretVersion"
	gcpSecretManagerDestroySecretVersionMethod = "/google.cloud.secretmanager.v1.SecretManagerService/DestroySecretVersion"
	gcpSecretManagerSetIAMPolicyMethod         = "/google.cloud.secretmanager.v1.SecretManagerService/SetIamPolicy"
	gcpSecretManagerGetIAMPolicyMethod         = "/google.cloud.secretmanager.v1.SecretManagerService/GetIamPolicy"
	gcpSecretManagerTestIAMPermissionsMethod   = "/google.cloud.secretmanager.v1.SecretManagerService/TestIamPermissions"

	gcpSecurityPrivateCAListCaPoolsMethod                = "/google.cloud.security.privateca.v1.CertificateAuthorityService/ListCaPools"
	gcpSecurityPrivateCAGetCaPoolMethod                  = "/google.cloud.security.privateca.v1.CertificateAuthorityService/GetCaPool"
	gcpSecurityPrivateCACreateCaPoolMethod               = "/google.cloud.security.privateca.v1.CertificateAuthorityService/CreateCaPool"
	gcpSecurityPrivateCAListCertificateAuthoritiesMethod = "/google.cloud.security.privateca.v1.CertificateAuthorityService/ListCertificateAuthorities"
	gcpSecurityPrivateCAGetCertificateAuthorityMethod    = "/google.cloud.security.privateca.v1.CertificateAuthorityService/GetCertificateAuthority"
	gcpSecurityPrivateCACreateCertificateAuthorityMethod = "/google.cloud.security.privateca.v1.CertificateAuthorityService/CreateCertificateAuthority"
	gcpSecurityPrivateCAListCertificatesMethod           = "/google.cloud.security.privateca.v1.CertificateAuthorityService/ListCertificates"
	gcpSecurityPrivateCAGetCertificateMethod             = "/google.cloud.security.privateca.v1.CertificateAuthorityService/GetCertificate"
	gcpSecurityPrivateCACreateCertificateMethod          = "/google.cloud.security.privateca.v1.CertificateAuthorityService/CreateCertificate"
	gcpSecurityPrivateCARevokeCertificateMethod          = "/google.cloud.security.privateca.v1.CertificateAuthorityService/RevokeCertificate"
	gcpSecurityPublicCACreateExternalAccountKeyMethod    = "/google.cloud.security.publicca.v1.PublicCertificateAuthorityService/CreateExternalAccountKey"

	gcpSecurityCenterListSourcesMethod    = "/google.cloud.securitycenter.v1.SecurityCenter/ListSources"
	gcpSecurityCenterGetSourceMethod      = "/google.cloud.securitycenter.v1.SecurityCenter/GetSource"
	gcpSecurityCenterCreateSourceMethod   = "/google.cloud.securitycenter.v1.SecurityCenter/CreateSource"
	gcpSecurityCenterSetMuteMethod        = "/google.cloud.securitycenter.v1.SecurityCenter/SetMute"
	gcpSecurityCenterV2ListSourcesMethod  = "/google.cloud.securitycenter.v2.SecurityCenter/ListSources"
	gcpSecurityCenterV2GetSourceMethod    = "/google.cloud.securitycenter.v2.SecurityCenter/GetSource"
	gcpSecurityCenterV2CreateSourceMethod = "/google.cloud.securitycenter.v2.SecurityCenter/CreateSource"
	gcpSecurityCenterV2SetMuteMethod      = "/google.cloud.securitycenter.v2.SecurityCenter/SetMute"

	gcpSecurityCenterManagementListEffectiveSHAModulesMethod  = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListEffectiveSecurityHealthAnalyticsCustomModules"
	gcpSecurityCenterManagementGetEffectiveSHAModuleMethod    = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/GetEffectiveSecurityHealthAnalyticsCustomModule"
	gcpSecurityCenterManagementListSHAModulesMethod           = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListSecurityHealthAnalyticsCustomModules"
	gcpSecurityCenterManagementListDescendantSHAModulesMethod = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListDescendantSecurityHealthAnalyticsCustomModules"
	gcpSecurityCenterManagementGetSHAModuleMethod             = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/GetSecurityHealthAnalyticsCustomModule"
	gcpSecurityCenterManagementCreateSHAModuleMethod          = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/CreateSecurityHealthAnalyticsCustomModule"
	gcpSecurityCenterManagementUpdateSHAModuleMethod          = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/UpdateSecurityHealthAnalyticsCustomModule"
	gcpSecurityCenterManagementDeleteSHAModuleMethod          = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/DeleteSecurityHealthAnalyticsCustomModule"
	gcpSecurityCenterManagementSimulateSHAModuleMethod        = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/SimulateSecurityHealthAnalyticsCustomModule"
	gcpSecurityCenterManagementListEffectiveETDModulesMethod  = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListEffectiveEventThreatDetectionCustomModules"
	gcpSecurityCenterManagementGetEffectiveETDModuleMethod    = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/GetEffectiveEventThreatDetectionCustomModule"
	gcpSecurityCenterManagementListETDModulesMethod           = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListEventThreatDetectionCustomModules"
	gcpSecurityCenterManagementListDescendantETDModulesMethod = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListDescendantEventThreatDetectionCustomModules"
	gcpSecurityCenterManagementGetETDModuleMethod             = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/GetEventThreatDetectionCustomModule"
	gcpSecurityCenterManagementCreateETDModuleMethod          = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/CreateEventThreatDetectionCustomModule"
	gcpSecurityCenterManagementUpdateETDModuleMethod          = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/UpdateEventThreatDetectionCustomModule"
	gcpSecurityCenterManagementDeleteETDModuleMethod          = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/DeleteEventThreatDetectionCustomModule"
	gcpSecurityCenterManagementValidateETDModuleMethod        = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ValidateEventThreatDetectionCustomModule"
	gcpSecurityCenterManagementGetServiceMethod               = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/GetSecurityCenterService"
	gcpSecurityCenterManagementListServicesMethod             = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/ListSecurityCenterServices"
	gcpSecurityCenterManagementUpdateServiceMethod            = "/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/UpdateSecurityCenterService"

	gcpSecurityPostureListPosturesMethod            = "/google.cloud.securityposture.v1.SecurityPosture/ListPostures"
	gcpSecurityPostureListPostureRevisionsMethod    = "/google.cloud.securityposture.v1.SecurityPosture/ListPostureRevisions"
	gcpSecurityPostureGetPostureMethod              = "/google.cloud.securityposture.v1.SecurityPosture/GetPosture"
	gcpSecurityPostureCreatePostureMethod           = "/google.cloud.securityposture.v1.SecurityPosture/CreatePosture"
	gcpSecurityPostureUpdatePostureMethod           = "/google.cloud.securityposture.v1.SecurityPosture/UpdatePosture"
	gcpSecurityPostureDeletePostureMethod           = "/google.cloud.securityposture.v1.SecurityPosture/DeletePosture"
	gcpSecurityPostureExtractPostureMethod          = "/google.cloud.securityposture.v1.SecurityPosture/ExtractPosture"
	gcpSecurityPostureListPostureDeploymentsMethod  = "/google.cloud.securityposture.v1.SecurityPosture/ListPostureDeployments"
	gcpSecurityPostureGetPostureDeploymentMethod    = "/google.cloud.securityposture.v1.SecurityPosture/GetPostureDeployment"
	gcpSecurityPostureCreatePostureDeploymentMethod = "/google.cloud.securityposture.v1.SecurityPosture/CreatePostureDeployment"
	gcpSecurityPostureUpdatePostureDeploymentMethod = "/google.cloud.securityposture.v1.SecurityPosture/UpdatePostureDeployment"
	gcpSecurityPostureDeletePostureDeploymentMethod = "/google.cloud.securityposture.v1.SecurityPosture/DeletePostureDeployment"
	gcpSecurityPostureListPostureTemplatesMethod    = "/google.cloud.securityposture.v1.SecurityPosture/ListPostureTemplates"
	gcpSecurityPostureGetPostureTemplateMethod      = "/google.cloud.securityposture.v1.SecurityPosture/GetPostureTemplate"

	gcpSecureSourceManagerListRepositoriesMethod = "/google.cloud.securesourcemanager.v1.SecureSourceManager/ListRepositories"
	gcpSecureSourceManagerGetRepositoryMethod    = "/google.cloud.securesourcemanager.v1.SecureSourceManager/GetRepository"
	gcpSecureSourceManagerGetIAMPolicyRepoMethod = "/google.cloud.securesourcemanager.v1.SecureSourceManager/GetIamPolicyRepo"
	gcpSecureSourceManagerListPullRequestsMethod = "/google.cloud.securesourcemanager.v1.SecureSourceManager/ListPullRequests"
	gcpSecureSourceManagerClosePullRequestMethod = "/google.cloud.securesourcemanager.v1.SecureSourceManager/ClosePullRequest"

	gcpRecommenderListInsightsMethod                = "/google.cloud.recommender.v1.Recommender/ListInsights"
	gcpRecommenderGetInsightMethod                  = "/google.cloud.recommender.v1.Recommender/GetInsight"
	gcpRecommenderMarkInsightAcceptedMethod         = "/google.cloud.recommender.v1.Recommender/MarkInsightAccepted"
	gcpRecommenderListRecommendationsMethod         = "/google.cloud.recommender.v1.Recommender/ListRecommendations"
	gcpRecommenderGetRecommendationMethod           = "/google.cloud.recommender.v1.Recommender/GetRecommendation"
	gcpRecommenderMarkRecommendationDismissedMethod = "/google.cloud.recommender.v1.Recommender/MarkRecommendationDismissed"
	gcpRecommenderMarkRecommendationClaimedMethod   = "/google.cloud.recommender.v1.Recommender/MarkRecommendationClaimed"
	gcpRecommenderMarkRecommendationSucceededMethod = "/google.cloud.recommender.v1.Recommender/MarkRecommendationSucceeded"
	gcpRecommenderMarkRecommendationFailedMethod    = "/google.cloud.recommender.v1.Recommender/MarkRecommendationFailed"
	gcpRecommenderGetRecommenderConfigMethod        = "/google.cloud.recommender.v1.Recommender/GetRecommenderConfig"
	gcpRecommenderUpdateRecommenderConfigMethod     = "/google.cloud.recommender.v1.Recommender/UpdateRecommenderConfig"
	gcpRecommenderGetInsightTypeConfigMethod        = "/google.cloud.recommender.v1.Recommender/GetInsightTypeConfig"
	gcpRecommenderUpdateInsightTypeConfigMethod     = "/google.cloud.recommender.v1.Recommender/UpdateInsightTypeConfig"

	gcpRecaptchaEnterpriseCreateAssessmentMethod                     = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/CreateAssessment"
	gcpRecaptchaEnterpriseAnnotateAssessmentMethod                   = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/AnnotateAssessment"
	gcpRecaptchaEnterpriseCreateKeyMethod                            = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/CreateKey"
	gcpRecaptchaEnterpriseListKeysMethod                             = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/ListKeys"
	gcpRecaptchaEnterpriseRetrieveLegacySecretKeyMethod              = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/RetrieveLegacySecretKey"
	gcpRecaptchaEnterpriseGetKeyMethod                               = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/GetKey"
	gcpRecaptchaEnterpriseUpdateKeyMethod                            = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/UpdateKey"
	gcpRecaptchaEnterpriseDeleteKeyMethod                            = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/DeleteKey"
	gcpRecaptchaEnterpriseMigrateKeyMethod                           = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/MigrateKey"
	gcpRecaptchaEnterpriseAddIpOverrideMethod                        = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/AddIpOverride"
	gcpRecaptchaEnterpriseRemoveIpOverrideMethod                     = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/RemoveIpOverride"
	gcpRecaptchaEnterpriseListIpOverridesMethod                      = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/ListIpOverrides"
	gcpRecaptchaEnterpriseGetMetricsMethod                           = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/GetMetrics"
	gcpRecaptchaEnterpriseCreateFirewallPolicyMethod                 = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/CreateFirewallPolicy"
	gcpRecaptchaEnterpriseListFirewallPoliciesMethod                 = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/ListFirewallPolicies"
	gcpRecaptchaEnterpriseGetFirewallPolicyMethod                    = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/GetFirewallPolicy"
	gcpRecaptchaEnterpriseUpdateFirewallPolicyMethod                 = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/UpdateFirewallPolicy"
	gcpRecaptchaEnterpriseDeleteFirewallPolicyMethod                 = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/DeleteFirewallPolicy"
	gcpRecaptchaEnterpriseReorderFirewallPoliciesMethod              = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/ReorderFirewallPolicies"
	gcpRecaptchaEnterpriseListRelatedAccountGroupsMethod             = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/ListRelatedAccountGroups"
	gcpRecaptchaEnterpriseListRelatedAccountGroupMembershipsMethod   = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/ListRelatedAccountGroupMemberships"
	gcpRecaptchaEnterpriseSearchRelatedAccountGroupMembershipsMethod = "/google.cloud.recaptchaenterprise.v1.RecaptchaEnterpriseService/SearchRelatedAccountGroupMemberships"

	gcpLocationsGetLocationMethod   = "/google.cloud.location.Locations/GetLocation"
	gcpLocationsListLocationsMethod = "/google.cloud.location.Locations/ListLocations"

	gcpLongrunningGetOpMethod    = "/google.longrunning.Operations/GetOperation"
	gcpLongrunningListOpsMethod  = "/google.longrunning.Operations/ListOperations"
	gcpLongrunningCancelOpMethod = "/google.longrunning.Operations/CancelOperation"
	gcpLongrunningDeleteOpMethod = "/google.longrunning.Operations/DeleteOperation"
)

var gcpStage4ReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func knownGCPStage4GRPCResponse(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpApigeeConnectListConnectionsMethod:
		return gcpStage4GRPCApigeeConnectListConnections(grpcReqBody)
	case gcpApigeeConnectTetherEgressMethod:
		return grpcUnimplemented("streaming-not-supported")
	case gcpMediaTranslationStreamingTranslateSpeechMethod:
		return gcpStage4GRPCMediaTranslationStreamingTranslateSpeech(grpcReqBody)
	case gcpCloudProfilerListProfilesMethod:
		return gcpStage4GRPCCloudProfilerListProfiles(grpcReqBody)
	case gcpCloudProfilerCreateProfileMethod:
		return gcpStage4GRPCCloudProfilerCreateProfile(grpcReqBody)
	case gcpCloudProfilerCreateOfflineProfileMethod:
		return gcpStage4GRPCCloudProfilerCreateOfflineProfile(grpcReqBody)
	case gcpCloudProfilerUpdateProfileMethod:
		return gcpStage4GRPCCloudProfilerUpdateProfile(grpcReqBody)
	case gcpCloudQuotasListQuotaInfosMethod:
		return gcpStage4GRPCCloudQuotasListQuotaInfos(grpcReqBody)
	case gcpCloudQuotasGetQuotaInfoMethod:
		return gcpStage4GRPCCloudQuotasGetQuotaInfo(grpcReqBody)
	case gcpCloudQuotasListQuotaPreferencesMethod:
		return gcpStage4GRPCCloudQuotasListQuotaPreferences(grpcReqBody)
	case gcpCloudQuotasGetQuotaPreferenceMethod:
		return gcpStage4GRPCCloudQuotasGetQuotaPreference(grpcReqBody)
	case gcpCloudQuotasCreateQuotaPreferenceMethod:
		return gcpStage4GRPCCloudQuotasCreateQuotaPreference(grpcReqBody)
	case gcpCloudQuotasUpdateQuotaPreferenceMethod:
		return gcpStage4GRPCCloudQuotasUpdateQuotaPreference(grpcReqBody)
	case gcpProcurementListOrdersMethod:
		return gcpStage4GRPCProcurementListOrders(grpcReqBody)
	case gcpProcurementGetOrderMethod:
		return gcpStage4GRPCProcurementGetOrder(grpcReqBody)
	case gcpProcurementPlaceOrderMethod:
		return gcpStage4GRPCProcurementPlaceOrder(grpcReqBody)
	case gcpProcurementModifyOrderMethod:
		return gcpStage4GRPCProcurementModifyOrder(grpcReqBody)
	case gcpProcurementCancelOrderMethod:
		return gcpStage4GRPCProcurementCancelOrder(grpcReqBody)
	case gcpConfigListDeploymentsMethod:
		return gcpStage4GRPCConfigListDeployments(grpcReqBody)
	case gcpConfigGetDeploymentMethod:
		return gcpStage4GRPCConfigGetDeployment(grpcReqBody)
	case gcpConfigCreateDeploymentMethod:
		return gcpStage4GRPCConfigCreateDeployment(grpcReqBody)
	case gcpConfigDeleteDeploymentMethod:
		return gcpStage4GRPCConfigDeleteDeployment(grpcReqBody)
	case gcpConfigLockDeploymentMethod:
		return gcpStage4GRPCConfigLockDeployment(grpcReqBody)
	case gcpConfigUnlockDeploymentMethod:
		return gcpStage4GRPCConfigUnlockDeployment(grpcReqBody)
	case gcpConfigExportLockInfoMethod:
		return gcpStage4GRPCConfigExportLockInfo(grpcReqBody)
	case gcpConfigCreatePreviewMethod:
		return gcpStage4GRPCConfigCreatePreview(grpcReqBody)
	case gcpConfigGetPreviewMethod:
		return gcpStage4GRPCConfigGetPreview(grpcReqBody)
	case gcpConfigListPreviewsMethod:
		return gcpStage4GRPCConfigListPreviews(grpcReqBody)
	case gcpConfigDeletePreviewMethod:
		return gcpStage4GRPCConfigDeletePreview(grpcReqBody)
	case gcpConfigExportPreviewResultMethod:
		return gcpStage4GRPCConfigExportPreviewResult(grpcReqBody)
	case gcpConfigListTerraformVersionsMethod:
		return gcpStage4GRPCConfigListTerraformVersions(grpcReqBody)
	case gcpConfigGetTerraformVersionMethod:
		return gcpStage4GRPCConfigGetTerraformVersion(grpcReqBody)
	case gcpConfigListResourceChangesMethod:
		return gcpStage4GRPCConfigListResourceChanges(grpcReqBody)
	case gcpConfigGetResourceChangeMethod:
		return gcpStage4GRPCConfigGetResourceChange(grpcReqBody)
	case gcpConfigListResourceDriftsMethod:
		return gcpStage4GRPCConfigListResourceDrifts(grpcReqBody)
	case gcpConfigGetResourceDriftMethod:
		return gcpStage4GRPCConfigGetResourceDrift(grpcReqBody)
	case gcpConfigDeliveryListResourceBundlesMethod:
		return gcpStage4GRPCConfigDeliveryListResourceBundles(grpcReqBody)
	case gcpConfigDeliveryGetResourceBundleMethod:
		return gcpStage4GRPCConfigDeliveryGetResourceBundle(grpcReqBody)
	case gcpConfigDeliveryCreateResourceBundleMethod:
		return gcpStage4GRPCConfigDeliveryCreateResourceBundle(grpcReqBody)
	case gcpConfigDeliveryDeleteResourceBundleMethod:
		return gcpStage4GRPCConfigDeliveryDeleteResourceBundle(grpcReqBody)
	case gcpConfigDeliveryListFleetPackagesMethod:
		return gcpStage4GRPCConfigDeliveryListFleetPackages(grpcReqBody)
	case gcpConfigDeliveryGetFleetPackageMethod:
		return gcpStage4GRPCConfigDeliveryGetFleetPackage(grpcReqBody)
	case gcpConfigDeliveryCreateFleetPackageMethod:
		return gcpStage4GRPCConfigDeliveryCreateFleetPackage(grpcReqBody)
	case gcpConfigDeliveryDeleteFleetPackageMethod:
		return gcpStage4GRPCConfigDeliveryDeleteFleetPackage(grpcReqBody)
	case gcpConfigDeliveryListReleasesMethod:
		return gcpStage4GRPCConfigDeliveryListReleases(grpcReqBody)
	case gcpConfigDeliveryGetReleaseMethod:
		return gcpStage4GRPCConfigDeliveryGetRelease(grpcReqBody)
	case gcpConfigDeliveryCreateReleaseMethod:
		return gcpStage4GRPCConfigDeliveryCreateRelease(grpcReqBody)
	case gcpConfigDeliveryDeleteReleaseMethod:
		return gcpStage4GRPCConfigDeliveryDeleteRelease(grpcReqBody)
	case gcpConfigDeliveryListVariantsMethod:
		return gcpStage4GRPCConfigDeliveryListVariants(grpcReqBody)
	case gcpConfigDeliveryGetVariantMethod:
		return gcpStage4GRPCConfigDeliveryGetVariant(grpcReqBody)
	case gcpConfigDeliveryCreateVariantMethod:
		return gcpStage4GRPCConfigDeliveryCreateVariant(grpcReqBody)
	case gcpConfigDeliveryDeleteVariantMethod:
		return gcpStage4GRPCConfigDeliveryDeleteVariant(grpcReqBody)
	case gcpConfigDeliveryListRolloutsMethod:
		return gcpStage4GRPCConfigDeliveryListRollouts(grpcReqBody)
	case gcpConfigDeliveryGetRolloutMethod:
		return gcpStage4GRPCConfigDeliveryGetRollout(grpcReqBody)
	case gcpConfigDeliverySuspendRolloutMethod:
		return gcpStage4GRPCConfigDeliveryRolloutAction(grpcReqBody, "suspend")
	case gcpConfigDeliveryResumeRolloutMethod:
		return gcpStage4GRPCConfigDeliveryRolloutAction(grpcReqBody, "resume")
	case gcpConfigDeliveryAbortRolloutMethod:
		return gcpStage4GRPCConfigDeliveryRolloutAction(grpcReqBody, "abort")
	case gcpRapidMigrationAssessmentListCollectorsMethod:
		return gcpStage4GRPCRapidMigrationAssessmentListCollectors(grpcReqBody)
	case gcpRapidMigrationAssessmentGetCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentGetCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentCreateCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentCreateCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentUpdateCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentUpdateCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentDeleteCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentDeleteCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentPauseCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentPauseCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentResumeCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentResumeCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentRegisterCollectorMethod:
		return gcpStage4GRPCRapidMigrationAssessmentRegisterCollector(grpcReqBody)
	case gcpRapidMigrationAssessmentCreateAnnotationMethod:
		return gcpStage4GRPCRapidMigrationAssessmentCreateAnnotation(grpcReqBody)
	case gcpRapidMigrationAssessmentGetAnnotationMethod:
		return gcpStage4GRPCRapidMigrationAssessmentGetAnnotation(grpcReqBody)
	case gcpVMMigrationListSourcesMethod,
		gcpVMMigrationGetSourceMethod,
		gcpVMMigrationCreateSourceMethod,
		gcpVMMigrationPauseMigrationMethod:
		return gcpStage4GRPCVMMigration(path, grpcReqBody)
	case gcpTelcoAutomationListOrchestrationClustersMethod,
		gcpTelcoAutomationGetOrchestrationClusterMethod,
		gcpTelcoAutomationCreateOrchestrationClusterMethod,
		gcpTelcoAutomationDeleteOrchestrationClusterMethod,
		gcpTelcoAutomationListEdgeSlmsMethod,
		gcpTelcoAutomationGetEdgeSlmMethod,
		gcpTelcoAutomationCreateEdgeSlmMethod,
		gcpTelcoAutomationDeleteEdgeSlmMethod,
		gcpTelcoAutomationCreateBlueprintMethod,
		gcpTelcoAutomationUpdateBlueprintMethod,
		gcpTelcoAutomationGetBlueprintMethod,
		gcpTelcoAutomationDeleteBlueprintMethod,
		gcpTelcoAutomationListBlueprintsMethod,
		gcpTelcoAutomationApproveBlueprintMethod,
		gcpTelcoAutomationProposeBlueprintMethod,
		gcpTelcoAutomationRejectBlueprintMethod,
		gcpTelcoAutomationListBlueprintRevisionsMethod,
		gcpTelcoAutomationSearchBlueprintRevisionsMethod,
		gcpTelcoAutomationSearchDeploymentRevisionsMethod,
		gcpTelcoAutomationDiscardBlueprintChangesMethod,
		gcpTelcoAutomationListPublicBlueprintsMethod,
		gcpTelcoAutomationGetPublicBlueprintMethod,
		gcpTelcoAutomationCreateDeploymentMethod,
		gcpTelcoAutomationUpdateDeploymentMethod,
		gcpTelcoAutomationGetDeploymentMethod,
		gcpTelcoAutomationRemoveDeploymentMethod,
		gcpTelcoAutomationListDeploymentsMethod,
		gcpTelcoAutomationListDeploymentRevisionsMethod,
		gcpTelcoAutomationDiscardDeploymentChangesMethod,
		gcpTelcoAutomationApplyDeploymentMethod,
		gcpTelcoAutomationComputeDeploymentStatusMethod,
		gcpTelcoAutomationRollbackDeploymentMethod,
		gcpTelcoAutomationGetHydratedDeploymentMethod,
		gcpTelcoAutomationListHydratedDeploymentsMethod,
		gcpTelcoAutomationUpdateHydratedDeploymentMethod,
		gcpTelcoAutomationApplyHydratedDeploymentMethod:
		return gcpStage4GRPCTelcoAutomation(path, grpcReqBody)
	case gcpResourceManagerListFoldersMethod:
		return gcpStage4GRPCResourceManagerListFolders(grpcReqBody)
	case gcpResourceManagerSearchFoldersMethod:
		return gcpStage4GRPCResourceManagerSearchFolders(grpcReqBody)
	case gcpResourceManagerGetFolderMethod:
		return gcpStage4GRPCResourceManagerGetFolder(grpcReqBody)
	case gcpResourceManagerCreateFolderMethod:
		return gcpStage4GRPCResourceManagerCreateFolder(grpcReqBody)
	case gcpResourceManagerUpdateFolderMethod:
		return gcpStage4GRPCResourceManagerUpdateFolder(grpcReqBody)
	case gcpResourceManagerMoveFolderMethod:
		return gcpStage4GRPCResourceManagerMoveFolder(grpcReqBody)
	case gcpResourceManagerDeleteFolderMethod:
		return gcpStage4GRPCResourceManagerDeleteFolder(grpcReqBody)
	case gcpResourceManagerUndeleteFolderMethod:
		return gcpStage4GRPCResourceManagerUndeleteFolder(grpcReqBody)
	case gcpResourceManagerGetIAMPolicyMethod:
		return gcpStage4GRPCResourceManagerGetIAMPolicy(grpcReqBody)
	case gcpResourceManagerSetIAMPolicyMethod:
		return gcpStage4GRPCResourceManagerSetIAMPolicy(grpcReqBody)
	case gcpResourceManagerTestIAMPermissionsMethod:
		return gcpStage4GRPCResourceManagerTestIAMPermissions(grpcReqBody)
	case gcpResourceManagerV3ListFoldersMethod:
		return gcpStage4GRPCResourceManagerV3ListFolders(grpcReqBody)
	case gcpResourceManagerV3SearchFoldersMethod:
		return gcpStage4GRPCResourceManagerV3SearchFolders(grpcReqBody)
	case gcpResourceManagerV3GetFolderMethod:
		return gcpStage4GRPCResourceManagerV3GetFolder(grpcReqBody)
	case gcpResourceManagerV3CreateFolderMethod:
		return gcpStage4GRPCResourceManagerV3CreateFolder(grpcReqBody)
	case gcpResourceManagerV3UpdateFolderMethod:
		return gcpStage4GRPCResourceManagerV3UpdateFolder(grpcReqBody)
	case gcpResourceManagerV3MoveFolderMethod:
		return gcpStage4GRPCResourceManagerV3MoveFolder(grpcReqBody)
	case gcpResourceManagerV3DeleteFolderMethod:
		return gcpStage4GRPCResourceManagerV3DeleteFolder(grpcReqBody)
	case gcpResourceManagerV3UndeleteFolderMethod:
		return gcpStage4GRPCResourceManagerV3UndeleteFolder(grpcReqBody)
	case gcpResourceManagerV3FoldersGetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3GetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3FoldersSetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3SetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3FoldersTestIAMPermission:
		return gcpStage4GRPCResourceManagerV3TestIAMPermissions(grpcReqBody)
	case gcpResourceManagerV3ListProjectsMethod:
		return gcpStage4GRPCResourceManagerV3ListProjects(grpcReqBody)
	case gcpResourceManagerV3SearchProjectsMethod:
		return gcpStage4GRPCResourceManagerV3SearchProjects(grpcReqBody)
	case gcpResourceManagerV3GetProjectMethod:
		return gcpStage4GRPCResourceManagerV3GetProject(grpcReqBody)
	case gcpResourceManagerV3CreateProjectMethod:
		return gcpStage4GRPCResourceManagerV3CreateProject(grpcReqBody)
	case gcpResourceManagerV3UpdateProjectMethod:
		return gcpStage4GRPCResourceManagerV3UpdateProject(grpcReqBody)
	case gcpResourceManagerV3MoveProjectMethod:
		return gcpStage4GRPCResourceManagerV3MoveProject(grpcReqBody)
	case gcpResourceManagerV3DeleteProjectMethod:
		return gcpStage4GRPCResourceManagerV3DeleteProject(grpcReqBody)
	case gcpResourceManagerV3UndeleteProjectMethod:
		return gcpStage4GRPCResourceManagerV3UndeleteProject(grpcReqBody)
	case gcpResourceManagerV3ProjectsGetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3GetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3ProjectsSetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3SetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3ProjectsTestIAMPermission:
		return gcpStage4GRPCResourceManagerV3TestIAMPermissions(grpcReqBody)
	case gcpResourceManagerV3GetOrganizationMethod:
		return gcpStage4GRPCResourceManagerV3GetOrganization(grpcReqBody)
	case gcpResourceManagerV3SearchOrganizationsMethod:
		return gcpStage4GRPCResourceManagerV3SearchOrganizations(grpcReqBody)
	case gcpResourceManagerV3OrganizationsGetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3GetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3OrganizationsSetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3SetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3OrganizationsTestIAMPermission:
		return gcpStage4GRPCResourceManagerV3TestIAMPermissions(grpcReqBody)
	case gcpResourceManagerV3ListTagKeysMethod:
		return gcpStage4GRPCResourceManagerV3ListTagKeys(grpcReqBody)
	case gcpResourceManagerV3GetTagKeyMethod:
		return gcpStage4GRPCResourceManagerV3GetTagKey(grpcReqBody)
	case gcpResourceManagerV3GetNamespacedTagKey:
		return gcpStage4GRPCResourceManagerV3GetNamespacedTagKey(grpcReqBody)
	case gcpResourceManagerV3CreateTagKeyMethod:
		return gcpStage4GRPCResourceManagerV3CreateTagKey(grpcReqBody)
	case gcpResourceManagerV3UpdateTagKeyMethod:
		return gcpStage4GRPCResourceManagerV3UpdateTagKey(grpcReqBody)
	case gcpResourceManagerV3DeleteTagKeyMethod:
		return gcpStage4GRPCResourceManagerV3DeleteTagKey(grpcReqBody)
	case gcpResourceManagerV3TagKeysGetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3GetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3TagKeysSetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3SetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3TagKeysTestIAMPermission:
		return gcpStage4GRPCResourceManagerV3TestIAMPermissions(grpcReqBody)
	case gcpResourceManagerV3ListTagValuesMethod:
		return gcpStage4GRPCResourceManagerV3ListTagValues(grpcReqBody)
	case gcpResourceManagerV3GetTagValueMethod:
		return gcpStage4GRPCResourceManagerV3GetTagValue(grpcReqBody)
	case gcpResourceManagerV3GetNamespacedTagValue:
		return gcpStage4GRPCResourceManagerV3GetNamespacedTagValue(grpcReqBody)
	case gcpResourceManagerV3CreateTagValueMethod:
		return gcpStage4GRPCResourceManagerV3CreateTagValue(grpcReqBody)
	case gcpResourceManagerV3UpdateTagValueMethod:
		return gcpStage4GRPCResourceManagerV3UpdateTagValue(grpcReqBody)
	case gcpResourceManagerV3DeleteTagValueMethod:
		return gcpStage4GRPCResourceManagerV3DeleteTagValue(grpcReqBody)
	case gcpResourceManagerV3TagValuesGetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3GetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3TagValuesSetIAMPolicy:
		return gcpStage4GRPCResourceManagerV3SetIAMPolicy(grpcReqBody)
	case gcpResourceManagerV3TagValuesTestIAMPermission:
		return gcpStage4GRPCResourceManagerV3TestIAMPermissions(grpcReqBody)
	case gcpResourceManagerV3ListTagBindingsMethod:
		return gcpStage4GRPCResourceManagerV3ListTagBindings(grpcReqBody)
	case gcpResourceManagerV3CreateTagBindingMethod:
		return gcpStage4GRPCResourceManagerV3CreateTagBinding(grpcReqBody)
	case gcpResourceManagerV3DeleteTagBindingMethod:
		return gcpStage4GRPCResourceManagerV3DeleteTagBinding(grpcReqBody)
	case gcpResourceManagerV3ListEffectiveTags:
		return gcpStage4GRPCResourceManagerV3ListEffectiveTags(grpcReqBody)
	case gcpResourceManagerV3ListTagHoldsMethod:
		return gcpStage4GRPCResourceManagerV3ListTagHolds(grpcReqBody)
	case gcpResourceManagerV3CreateTagHoldMethod:
		return gcpStage4GRPCResourceManagerV3CreateTagHold(grpcReqBody)
	case gcpResourceManagerV3DeleteTagHoldMethod:
		return gcpStage4GRPCResourceManagerV3DeleteTagHold(grpcReqBody)
	case gcpRedisListInstancesMethod:
		return gcpStage4GRPCRedisListInstances(grpcReqBody)
	case gcpRedisGetInstanceMethod:
		return gcpStage4GRPCRedisGetInstance(grpcReqBody)
	case gcpRedisGetInstanceAuthStringMethod:
		return gcpStage4GRPCRedisGetInstanceAuthString(grpcReqBody)
	case gcpRedisCreateInstanceMethod:
		return gcpStage4GRPCRedisCreateInstance(grpcReqBody)
	case gcpRedisUpdateInstanceMethod:
		return gcpStage4GRPCRedisUpdateInstance(grpcReqBody)
	case gcpRedisUpgradeInstanceMethod:
		return gcpStage4GRPCRedisUpgradeInstance(grpcReqBody)
	case gcpRedisImportInstanceMethod:
		return gcpStage4GRPCRedisImportInstance(grpcReqBody)
	case gcpRedisExportInstanceMethod:
		return gcpStage4GRPCRedisExportInstance(grpcReqBody)
	case gcpRedisFailoverInstanceMethod:
		return gcpStage4GRPCRedisFailoverInstance(grpcReqBody)
	case gcpRedisDeleteInstanceMethod:
		return gcpStage4GRPCRedisDeleteInstance(grpcReqBody)
	case gcpRedisRescheduleMaintenanceMethod:
		return gcpStage4GRPCRedisRescheduleMaintenance(grpcReqBody)
	case gcpRedisClusterListClustersMethod:
		return gcpStage4GRPCRedisClusterListClusters(grpcReqBody)
	case gcpRedisClusterGetClusterMethod:
		return gcpStage4GRPCRedisClusterGetCluster(grpcReqBody)
	case gcpRedisClusterUpdateClusterMethod:
		return gcpStage4GRPCRedisClusterUpdateCluster(grpcReqBody)
	case gcpRedisClusterDeleteClusterMethod:
		return gcpStage4GRPCRedisClusterDeleteCluster(grpcReqBody)
	case gcpRedisClusterCreateClusterMethod:
		return gcpStage4GRPCRedisClusterCreateCluster(grpcReqBody)
	case gcpRedisClusterGetClusterCertificateAuthorityMethod:
		return gcpStage4GRPCRedisClusterGetClusterCertificateAuthority(grpcReqBody)
	case gcpRedisClusterRescheduleClusterMaintenanceMethod:
		return gcpStage4GRPCRedisClusterRescheduleClusterMaintenance(grpcReqBody)
	case gcpRedisClusterListBackupCollectionsMethod:
		return gcpStage4GRPCRedisClusterListBackupCollections(grpcReqBody)
	case gcpRedisClusterGetBackupCollectionMethod:
		return gcpStage4GRPCRedisClusterGetBackupCollection(grpcReqBody)
	case gcpRedisClusterListBackupsMethod:
		return gcpStage4GRPCRedisClusterListBackups(grpcReqBody)
	case gcpRedisClusterGetBackupMethod:
		return gcpStage4GRPCRedisClusterGetBackup(grpcReqBody)
	case gcpRedisClusterDeleteBackupMethod:
		return gcpStage4GRPCRedisClusterDeleteBackup(grpcReqBody)
	case gcpRedisClusterExportBackupMethod:
		return gcpStage4GRPCRedisClusterExportBackup(grpcReqBody)
	case gcpRedisClusterBackupClusterMethod:
		return gcpStage4GRPCRedisClusterBackupCluster(grpcReqBody)
	case gcpRecommendationEngineCreateCatalogItemMethod:
		return gcpStage4GRPCRecommendationEngineCreateCatalogItem(grpcReqBody)
	case gcpRecommendationEngineGetCatalogItemMethod:
		return gcpStage4GRPCRecommendationEngineGetCatalogItem(grpcReqBody)
	case gcpRecommendationEngineListCatalogItemsMethod:
		return gcpStage4GRPCRecommendationEngineListCatalogItems(grpcReqBody)
	case gcpRecommendationEngineUpdateCatalogItemMethod:
		return gcpStage4GRPCRecommendationEngineUpdateCatalogItem(grpcReqBody)
	case gcpRecommendationEngineDeleteCatalogItemMethod:
		return gcpStage4GRPCRecommendationEngineDeleteCatalogItem(grpcReqBody)
	case gcpRecommendationEngineImportCatalogItemsMethod:
		return gcpStage4GRPCRecommendationEngineImportCatalogItems(grpcReqBody)
	case gcpRecommendationEngineWriteUserEventMethod:
		return gcpStage4GRPCRecommendationEngineWriteUserEvent(grpcReqBody)
	case gcpRecommendationEngineCollectUserEventMethod:
		return gcpStage4GRPCRecommendationEngineCollectUserEvent(grpcReqBody)
	case gcpRecommendationEngineListUserEventsMethod:
		return gcpStage4GRPCRecommendationEngineListUserEvents(grpcReqBody)
	case gcpRecommendationEnginePurgeUserEventsMethod:
		return gcpStage4GRPCRecommendationEnginePurgeUserEvents(grpcReqBody)
	case gcpRecommendationEngineImportUserEventsMethod:
		return gcpStage4GRPCRecommendationEngineImportUserEvents(grpcReqBody)
	case gcpRecommendationEnginePredictMethod:
		return gcpStage4GRPCRecommendationEnginePredict(grpcReqBody)
	case gcpRecommendationEngineCreatePredictionAPIKeyRegistrationMethod:
		return gcpStage4GRPCRecommendationEngineCreatePredictionAPIKeyRegistration(grpcReqBody)
	case gcpRecommendationEngineListPredictionAPIKeyRegistrationsMethod:
		return gcpStage4GRPCRecommendationEngineListPredictionAPIKeyRegistrations(grpcReqBody)
	case gcpRecommendationEngineDeletePredictionAPIKeyRegistrationMethod:
		return gcpStage4GRPCRecommendationEngineDeletePredictionAPIKeyRegistration(grpcReqBody)
	case gcpRetailListProductsMethod:
		return gcpStage4GRPCRetailListProducts(grpcReqBody)
	case gcpRetailCreateProductMethod:
		return gcpStage4GRPCRetailCreateProduct(grpcReqBody)
	case gcpRetailSearchMethod:
		return gcpStage4GRPCRetailSearch(grpcReqBody)
	case gcpRetailCreateServingConfigMethod:
		return gcpStage4GRPCRetailCreateServingConfig(grpcReqBody)
	case gcpRetailGetServingConfigMethod:
		return gcpStage4GRPCRetailGetServingConfig(grpcReqBody)
	case gcpRunListServicesMethod:
		return gcpStage4GRPCRunListServices(grpcReqBody)
	case gcpRunGetServiceMethod:
		return gcpStage4GRPCRunGetService(grpcReqBody)
	case gcpRunCreateServiceMethod:
		return gcpStage4GRPCRunCreateService(grpcReqBody)
	case gcpRunListJobsMethod:
		return gcpStage4GRPCRunListJobs(grpcReqBody)
	case gcpRunGetJobMethod:
		return gcpStage4GRPCRunGetJob(grpcReqBody)
	case gcpRunCreateJobMethod:
		return gcpStage4GRPCRunCreateJob(grpcReqBody)
	case gcpRunRunJobMethod:
		return gcpStage4GRPCRunRunJob(grpcReqBody)
	case gcpRunListExecutionsMethod:
		return gcpStage4GRPCRunListExecutions(grpcReqBody)
	case gcpRunGetExecutionMethod:
		return gcpStage4GRPCRunGetExecution(grpcReqBody)
	case gcpRunListTasksMethod:
		return gcpStage4GRPCRunListTasks(grpcReqBody)
	case gcpRunGetTaskMethod:
		return gcpStage4GRPCRunGetTask(grpcReqBody)
	case gcpRunListRevisionsMethod:
		return gcpStage4GRPCRunListRevisions(grpcReqBody)
	case gcpRunGetRevisionMethod:
		return gcpStage4GRPCRunGetRevision(grpcReqBody)
	case gcpSchedulerListJobsMethod:
		return gcpStage4GRPCSchedulerListJobs(grpcReqBody)
	case gcpSchedulerGetJobMethod:
		return gcpStage4GRPCSchedulerGetJob(grpcReqBody)
	case gcpSchedulerCreateJobMethod:
		return gcpStage4GRPCSchedulerCreateJob(grpcReqBody)
	case gcpSchedulerUpdateJobMethod:
		return gcpStage4GRPCSchedulerUpdateJob(grpcReqBody)
	case gcpSchedulerDeleteJobMethod:
		return gcpStage4GRPCSchedulerDeleteJob(grpcReqBody)
	case gcpSchedulerPauseJobMethod:
		return gcpStage4GRPCSchedulerPauseJob(grpcReqBody)
	case gcpSchedulerResumeJobMethod:
		return gcpStage4GRPCSchedulerResumeJob(grpcReqBody)
	case gcpSchedulerRunJobMethod:
		return gcpStage4GRPCSchedulerRunJob(grpcReqBody)
	case gcpWorkflowExecutionsListExecutionsMethod,
		gcpWorkflowExecutionsCreateExecutionMethod,
		gcpWorkflowExecutionsGetExecutionMethod,
		gcpWorkflowExecutionsCancelExecutionMethod:
		return gcpStage4GRPCWorkflowExecutions(path, grpcReqBody)
	case gcpWorkflowsListWorkflowsMethod,
		gcpWorkflowsGetWorkflowMethod,
		gcpWorkflowsCreateWorkflowMethod,
		gcpWorkflowsDeleteWorkflowMethod,
		gcpWorkflowsUpdateWorkflowMethod,
		gcpWorkflowsListWorkflowRevisionsMethod:
		return gcpStage4GRPCWorkflows(path, grpcReqBody)
	case gcpWorkstationsListWorkstationClustersMethod,
		gcpWorkstationsGetWorkstationClusterMethod,
		gcpWorkstationsCreateWorkstationClusterMethod,
		gcpWorkstationsUpdateWorkstationClusterMethod,
		gcpWorkstationsDeleteWorkstationClusterMethod,
		gcpWorkstationsListWorkstationConfigsMethod,
		gcpWorkstationsListUsableWorkstationConfigsMethod,
		gcpWorkstationsGetWorkstationConfigMethod,
		gcpWorkstationsCreateWorkstationConfigMethod,
		gcpWorkstationsUpdateWorkstationConfigMethod,
		gcpWorkstationsDeleteWorkstationConfigMethod,
		gcpWorkstationsListWorkstationsMethod,
		gcpWorkstationsListUsableWorkstationsMethod,
		gcpWorkstationsGetWorkstationMethod,
		gcpWorkstationsCreateWorkstationMethod,
		gcpWorkstationsUpdateWorkstationMethod,
		gcpWorkstationsDeleteWorkstationMethod,
		gcpWorkstationsStartWorkstationMethod,
		gcpWorkstationsStopWorkstationMethod,
		gcpWorkstationsGenerateAccessTokenMethod:
		return gcpStage4GRPCWorkstations(path, grpcReqBody)
	case gcpStorageBatchOperationsListJobsMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageBatchOperationsGetJobMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageBatchOperationsCreateJobMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageBatchOperationsDeleteJobMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageBatchOperationsCancelJobMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageBatchOperationsListBucketOperationsMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageBatchOperationsGetBucketOperationMethod:
		return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
	case gcpStorageTransferGetGoogleServiceAccountMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferCreateTransferJobMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferUpdateTransferJobMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferGetTransferJobMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferListTransferJobsMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferPauseTransferOperationMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferResumeTransferOperationMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferRunTransferJobMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferDeleteTransferJobMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferCreateAgentPoolMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferUpdateAgentPoolMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferGetAgentPoolMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferListAgentPoolsMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStorageTransferDeleteAgentPoolMethod:
		return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
	case gcpStreetViewPublishStartUploadMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishCreatePhotoMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishGetPhotoMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishBatchGetPhotosMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishListPhotosMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishUpdatePhotoMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishBatchUpdatePhotosMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishDeletePhotoMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishBatchDeletePhotosMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishStartPhotoSequenceUploadMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishCreatePhotoSequenceMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishGetPhotoSequenceMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishListPhotoSequencesMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpStreetViewPublishDeletePhotoSequenceMethod:
		return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
	case gcpSpeechRecognizeMethod:
		return gcpStage4GRPCSpeech(path, grpcReqBody)
	case gcpSpeechLongRunningRecognizeMethod:
		return gcpStage4GRPCSpeech(path, grpcReqBody)
	case gcpSpeechStreamingRecognizeMethod:
		return gcpStage4GRPCSpeech(path, grpcReqBody)
	case gcpSpeechCreatePhraseSetMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechGetPhraseSetMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechListPhraseSetMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechUpdatePhraseSetMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechDeletePhraseSetMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechCreateCustomClassMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechGetCustomClassMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechListCustomClassesMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechUpdateCustomClassMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpSpeechDeleteCustomClassMethod:
		return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
	case gcpTextToSpeechListVoicesMethod,
		gcpTextToSpeechSynthesizeSpeechMethod,
		gcpTextToSpeechStreamingSynthesizeMethod,
		gcpTextToSpeechSynthesizeLongAudioMethod,
		gcpTextToSpeechGetOperationMethod,
		gcpTextToSpeechListOperationsMethod:
		return gcpStage4GRPCTextToSpeech(path, grpcReqBody)
	case gcpTraceV1ListTracesMethod,
		gcpTraceV1GetTraceMethod,
		gcpTraceV1PatchTracesMethod:
		return gcpStage4GRPCTraceV1(path, grpcReqBody)
	case gcpTraceBatchWriteSpansMethod,
		gcpTraceCreateSpanMethod:
		return gcpStage4GRPCTrace(path, grpcReqBody)
	case gcpTPUListNodesMethod,
		gcpTPUGetNodeMethod,
		gcpTPUCreateNodeMethod,
		gcpTPUDeleteNodeMethod,
		gcpTPUReimageNodeMethod,
		gcpTPUStopNodeMethod,
		gcpTPUStartNodeMethod,
		gcpTPUListTensorFlowVersionsMethod,
		gcpTPUGetTensorFlowVersionMethod,
		gcpTPUListAcceleratorTypesMethod,
		gcpTPUGetAcceleratorTypeMethod:
		return gcpStage4GRPCTPU(path, grpcReqBody)
	case gcpSpeechV2CreateRecognizerMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2ListRecognizersMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2GetRecognizerMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UpdateRecognizerMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2DeleteRecognizerMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UndeleteRecognizerMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2RecognizeMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2StreamingRecognizeMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2BatchRecognizeMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2GetConfigMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UpdateConfigMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2CreateCustomClassMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2ListCustomClassesMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2GetCustomClassMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UpdateCustomClassMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2DeleteCustomClassMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UndeleteCustomClassMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2CreatePhraseSetMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2ListPhraseSetsMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2GetPhraseSetMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UpdatePhraseSetMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2DeletePhraseSetMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpeechV2UndeletePhraseSetMethod:
		return gcpStage4GRPCSpeechV2(path, grpcReqBody)
	case gcpSpannerCreateSessionMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerBatchCreateSessionsMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerGetSessionMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerListSessionsMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerDeleteSessionMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerExecuteSQLMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerExecuteStreamingSQLMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerExecuteBatchDMLMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerReadMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerStreamingReadMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerBeginTransactionMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerCommitMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerRollbackMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerPartitionQueryMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerPartitionReadMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerBatchWriteMethod:
		return gcpStage4GRPCSpanner(path, grpcReqBody)
	case gcpSpannerAdapterCreateSessionMethod:
		return gcpStage4GRPCSpannerAdapter(path, grpcReqBody)
	case gcpSpannerAdapterAdaptMessageMethod:
		return gcpStage4GRPCSpannerAdapter(path, grpcReqBody)
	case gcpSpannerExecutorExecuteActionAsyncMethod:
		return gcpStage4GRPCSpannerExecutor(grpcReqBody)
	case gcpSpannerAdminDatabaseListDatabasesMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseCreateDatabaseMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseGetDatabaseMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseUpdateDatabaseMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseUpdateDatabaseDDLMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseDropDatabaseMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseGetDatabaseDDLMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseSetIAMPolicyMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseGetIAMPolicyMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseTestIAMPermissionsMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseCreateBackupMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseCopyBackupMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseGetBackupMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseUpdateBackupMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseDeleteBackupMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseListBackupsMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseRestoreDatabaseMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseListDatabaseOperationsMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseListBackupOperationsMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseListDatabaseRolesMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseAddSplitPointsMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseCreateBackupScheduleMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseGetBackupScheduleMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseUpdateBackupScheduleMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseDeleteBackupScheduleMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseListBackupSchedulesMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseInternalUpdateGraphOpMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseCancelOperationMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseDeleteOperationMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseGetOperationMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminDatabaseListOperationsMethod:
		return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
	case gcpSpannerAdminInstanceListInstanceConfigsMethod,
		gcpSpannerAdminInstanceGetInstanceConfigMethod,
		gcpSpannerAdminInstanceCreateInstanceConfigMethod,
		gcpSpannerAdminInstanceUpdateInstanceConfigMethod,
		gcpSpannerAdminInstanceDeleteInstanceConfigMethod,
		gcpSpannerAdminInstanceListInstanceConfigOpsMethod,
		gcpSpannerAdminInstanceListInstancesMethod,
		gcpSpannerAdminInstanceGetInstanceMethod,
		gcpSpannerAdminInstanceCreateInstanceMethod,
		gcpSpannerAdminInstanceUpdateInstanceMethod,
		gcpSpannerAdminInstanceDeleteInstanceMethod,
		gcpSpannerAdminInstanceListInstancePartitionsMethod,
		gcpSpannerAdminInstanceGetInstancePartitionMethod,
		gcpSpannerAdminInstanceCreateInstancePartitionMethod,
		gcpSpannerAdminInstanceUpdateInstancePartitionMethod,
		gcpSpannerAdminInstanceDeleteInstancePartitionMethod,
		gcpSpannerAdminInstanceListPartitionOpsMethod,
		gcpSpannerAdminInstanceMoveInstanceMethod,
		gcpSpannerAdminInstanceSetIAMPolicyMethod,
		gcpSpannerAdminInstanceGetIAMPolicyMethod,
		gcpSpannerAdminInstanceTestIAMPermissionsMethod,
		gcpSpannerAdminInstanceCancelOperationMethod,
		gcpSpannerAdminInstanceDeleteOperationMethod,
		gcpSpannerAdminInstanceGetOperationMethod,
		gcpSpannerAdminInstanceListOperationsMethod:
		return gcpStage4GRPCSpannerAdminInstance(path, grpcReqBody)
	case gcpShellGetEnvironmentMethod:
		return gcpStage4GRPCShellGetEnvironment(grpcReqBody)
	case gcpShellStartEnvironmentMethod:
		return gcpStage4GRPCShellStartEnvironment(grpcReqBody)
	case gcpShellAuthorizeEnvironmentMethod:
		return gcpStage4GRPCShellAuthorizeEnvironment(grpcReqBody)
	case gcpShellAddPublicKeyMethod:
		return gcpStage4GRPCShellAddPublicKey(grpcReqBody)
	case gcpShellRemovePublicKeyMethod:
		return gcpStage4GRPCShellRemovePublicKey(grpcReqBody)
	case gcpShoppingCSSListChildAccountsMethod:
		return gcpStage4GRPCShoppingCSSListChildAccounts(grpcReqBody)
	case gcpShoppingCSSGetAccountMethod:
		return gcpStage4GRPCShoppingCSSGetAccount(grpcReqBody)
	case gcpShoppingCSSUpdateLabelsMethod:
		return gcpStage4GRPCShoppingCSSUpdateLabels(grpcReqBody)
	case gcpShoppingCSSListAccountLabelsMethod:
		return gcpStage4GRPCShoppingCSSListAccountLabels(grpcReqBody)
	case gcpShoppingCSSCreateAccountLabelMethod:
		return gcpStage4GRPCShoppingCSSCreateAccountLabel(grpcReqBody)
	case gcpShoppingCSSUpdateAccountLabelMethod:
		return gcpStage4GRPCShoppingCSSUpdateAccountLabel(grpcReqBody)
	case gcpShoppingCSSDeleteAccountLabelMethod:
		return gcpStage4GRPCShoppingCSSDeleteAccountLabel(grpcReqBody)
	case gcpShoppingCSSGetCssProductMethod:
		return gcpStage4GRPCShoppingCSSGetCssProduct(grpcReqBody)
	case gcpShoppingCSSListCssProductsMethod:
		return gcpStage4GRPCShoppingCSSListCssProducts(grpcReqBody)
	case gcpShoppingCSSInsertCssProductInputMethod:
		return gcpStage4GRPCShoppingCSSInsertCssProductInput(grpcReqBody)
	case gcpShoppingCSSUpdateCssProductInputMethod:
		return gcpStage4GRPCShoppingCSSUpdateCssProductInput(grpcReqBody)
	case gcpShoppingCSSDeleteCssProductInputMethod:
		return gcpStage4GRPCShoppingCSSDeleteCssProductInput(grpcReqBody)
	case gcpShoppingCSSListQuotaGroupsMethod:
		return gcpStage4GRPCShoppingCSSListQuotaGroups(grpcReqBody)
	case gcpShoppingMerchantConversionsCreateConversionSourceMethod:
		return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
	case gcpShoppingMerchantConversionsUpdateConversionSourceMethod:
		return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
	case gcpShoppingMerchantConversionsDeleteConversionSourceMethod:
		return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
	case gcpShoppingMerchantConversionsUndeleteConversionSourceMethod:
		return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
	case gcpShoppingMerchantConversionsGetConversionSourceMethod:
		return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
	case gcpShoppingMerchantConversionsListConversionSourcesMethod:
		return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesGetDataSourceMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesListDataSourcesMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesCreateDataSourceMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesUpdateDataSourceMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesDeleteDataSourceMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesFetchDataSourceMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantDatasourcesGetFileUploadMethod:
		return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
	case gcpShoppingMerchantInventoriesListLocalInventoriesMethod:
		return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
	case gcpShoppingMerchantInventoriesInsertLocalInventoryMethod:
		return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
	case gcpShoppingMerchantInventoriesDeleteLocalInventoryMethod:
		return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
	case gcpShoppingMerchantInventoriesListRegionalInventoriesMethod:
		return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
	case gcpShoppingMerchantInventoriesInsertRegionalInventoryMethod:
		return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
	case gcpShoppingMerchantInventoriesDeleteRegionalInventoryMethod:
		return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
	case gcpShoppingMerchantIssueresolutionRenderAccountIssuesMethod:
		return gcpStage4GRPCShoppingMerchantIssueresolution(path, grpcReqBody)
	case gcpShoppingMerchantIssueresolutionRenderProductIssuesMethod:
		return gcpStage4GRPCShoppingMerchantIssueresolution(path, grpcReqBody)
	case gcpShoppingMerchantIssueresolutionTriggerActionMethod:
		return gcpStage4GRPCShoppingMerchantIssueresolution(path, grpcReqBody)
	case gcpShoppingMerchantIssueresolutionListAggregateProductStatusesMethod:
		return gcpStage4GRPCShoppingMerchantIssueresolution(path, grpcReqBody)
	case gcpShoppingMerchantNotificationsGetNotificationSubscriptionMethod:
		return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
	case gcpShoppingMerchantNotificationsCreateNotificationSubscriptionMethod:
		return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
	case gcpShoppingMerchantNotificationsUpdateNotificationSubscriptionMethod:
		return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
	case gcpShoppingMerchantNotificationsDeleteNotificationSubscriptionMethod:
		return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
	case gcpShoppingMerchantNotificationsListNotificationSubscriptionsMethod:
		return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
	case gcpShoppingMerchantNotificationsGetNotificationSubscriptionHealthMetricsMethod:
		return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
	case gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod:
		return gcpStage4GRPCShoppingMerchantOrdertracking(path, grpcReqBody)
	case gcpShoppingMerchantPromotionsInsertPromotionMethod:
		return gcpStage4GRPCShoppingMerchantPromotions(path, grpcReqBody)
	case gcpShoppingMerchantPromotionsGetPromotionMethod:
		return gcpStage4GRPCShoppingMerchantPromotions(path, grpcReqBody)
	case gcpShoppingMerchantPromotionsListPromotionsMethod:
		return gcpStage4GRPCShoppingMerchantPromotions(path, grpcReqBody)
	case gcpShoppingMerchantReportsSearchMethod:
		return gcpStage4GRPCShoppingMerchantReports(path, grpcReqBody)
	case gcpShoppingMerchantReviewsGetMerchantReviewMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsListMerchantReviewsMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsInsertMerchantReviewMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsDeleteMerchantReviewMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsGetProductReviewMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsListProductReviewsMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsInsertProductReviewMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantReviewsDeleteProductReviewMethod:
		return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
	case gcpShoppingMerchantQuotaListQuotaGroupsMethod:
		return gcpStage4GRPCShoppingMerchantQuota(path, grpcReqBody)
	case gcpShoppingMerchantProductsGetProductMethod:
		return gcpStage4GRPCShoppingMerchantProducts(path, grpcReqBody)
	case gcpShoppingMerchantProductsListProductsMethod:
		return gcpStage4GRPCShoppingMerchantProducts(path, grpcReqBody)
	case gcpShoppingMerchantProductsInsertProductInputMethod:
		return gcpStage4GRPCShoppingMerchantProducts(path, grpcReqBody)
	case gcpShoppingMerchantProductsUpdateProductInputMethod:
		return gcpStage4GRPCShoppingMerchantProducts(path, grpcReqBody)
	case gcpShoppingMerchantProductsDeleteProductInputMethod:
		return gcpStage4GRPCShoppingMerchantProducts(path, grpcReqBody)
	case gcpShoppingMerchantProductstudioGenerateProductImageBackgroundMethod:
		return gcpStage4GRPCShoppingMerchantProductstudio(path, grpcReqBody)
	case gcpShoppingMerchantProductstudioRemoveProductImageBackgroundMethod:
		return gcpStage4GRPCShoppingMerchantProductstudio(path, grpcReqBody)
	case gcpShoppingMerchantProductstudioUpscaleProductImageMethod:
		return gcpStage4GRPCShoppingMerchantProductstudio(path, grpcReqBody)
	case gcpShoppingMerchantProductstudioGenerateProductTextSuggestionsMethod:
		return gcpStage4GRPCShoppingMerchantProductstudio(path, grpcReqBody)
	case gcpServiceControlCheckMethod:
		return gcpStage4GRPCServiceControlCheck(grpcReqBody)
	case gcpServiceControlReportMethod:
		return gcpStage4GRPCServiceControlReport(grpcReqBody)
	case gcpServiceControlAllocateQuotaMethod:
		return gcpStage4GRPCServiceControlAllocateQuota(grpcReqBody)
	case gcpWebRiskComputeThreatListDiffMethod:
		return gcpStage4GRPCWebRisk(path, grpcReqBody)
	case gcpWebRiskSearchUrisMethod:
		return gcpStage4GRPCWebRisk(path, grpcReqBody)
	case gcpWebRiskSearchHashesMethod:
		return gcpStage4GRPCWebRisk(path, grpcReqBody)
	case gcpWebRiskCreateSubmissionMethod:
		return gcpStage4GRPCWebRisk(path, grpcReqBody)
	case gcpWebRiskSubmitURIMethod:
		return gcpStage4GRPCWebRisk(path, grpcReqBody)
	case gcpWebSecurityScannerCreateScanConfigMethod,
		gcpWebSecurityScannerDeleteScanConfigMethod,
		gcpWebSecurityScannerGetScanConfigMethod,
		gcpWebSecurityScannerListScanConfigsMethod,
		gcpWebSecurityScannerUpdateScanConfigMethod,
		gcpWebSecurityScannerStartScanRunMethod,
		gcpWebSecurityScannerGetScanRunMethod,
		gcpWebSecurityScannerListScanRunsMethod,
		gcpWebSecurityScannerStopScanRunMethod,
		gcpWebSecurityScannerListCrawledURLsMethod,
		gcpWebSecurityScannerGetFindingMethod,
		gcpWebSecurityScannerListFindingsMethod,
		gcpWebSecurityScannerListFindingTypeStatsMethod:
		return gcpStage4GRPCWebSecurityScanner(path, grpcReqBody)
	case gcpServiceUsageEnableServiceMethod:
		return gcpStage4GRPCServiceUsageEnableService(grpcReqBody)
	case gcpServiceUsageDisableServiceMethod:
		return gcpStage4GRPCServiceUsageDisableService(grpcReqBody)
	case gcpServiceUsageGetServiceMethod:
		return gcpStage4GRPCServiceUsageGetService(grpcReqBody)
	case gcpServiceUsageListServicesMethod:
		return gcpStage4GRPCServiceUsageListServices(grpcReqBody)
	case gcpServiceUsageBatchEnableServicesMethod:
		return gcpStage4GRPCServiceUsageBatchEnableServices(grpcReqBody)
	case gcpServiceUsageBatchGetServicesMethod:
		return gcpStage4GRPCServiceUsageBatchGetServices(grpcReqBody)
	case gcpSupportGetCaseMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportListCasesMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportSearchCasesMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportCreateCaseMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportUpdateCaseMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportEscalateCaseMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportCloseCaseMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportSearchCaseClassificationsMethod:
		return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
	case gcpSupportListCommentsMethod:
		return gcpStage4GRPCSupportCommentService(path, grpcReqBody)
	case gcpSupportCreateCommentMethod:
		return gcpStage4GRPCSupportCommentService(path, grpcReqBody)
	case gcpSupportListAttachmentsMethod:
		return gcpStage4GRPCSupportCaseAttachmentService(path, grpcReqBody)
	case gcpTalentCreateCompanyMethod:
		return gcpStage4GRPCTalentCompanyService(path, grpcReqBody)
	case gcpTalentGetCompanyMethod:
		return gcpStage4GRPCTalentCompanyService(path, grpcReqBody)
	case gcpTalentUpdateCompanyMethod:
		return gcpStage4GRPCTalentCompanyService(path, grpcReqBody)
	case gcpTalentDeleteCompanyMethod:
		return gcpStage4GRPCTalentCompanyService(path, grpcReqBody)
	case gcpTalentListCompaniesMethod:
		return gcpStage4GRPCTalentCompanyService(path, grpcReqBody)
	case gcpTalentCreateTenantMethod:
		return gcpStage4GRPCTalentTenantService(path, grpcReqBody)
	case gcpTalentGetTenantMethod:
		return gcpStage4GRPCTalentTenantService(path, grpcReqBody)
	case gcpTalentUpdateTenantMethod:
		return gcpStage4GRPCTalentTenantService(path, grpcReqBody)
	case gcpTalentDeleteTenantMethod:
		return gcpStage4GRPCTalentTenantService(path, grpcReqBody)
	case gcpTalentListTenantsMethod:
		return gcpStage4GRPCTalentTenantService(path, grpcReqBody)
	case gcpTalentCreateJobMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentBatchCreateJobsMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentGetJobMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentUpdateJobMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentBatchUpdateJobsMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentDeleteJobMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentBatchDeleteJobsMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentListJobsMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentSearchJobsMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentSearchJobsForAlertMethod:
		return gcpStage4GRPCTalentJobService(path, grpcReqBody)
	case gcpTalentCompleteQueryMethod:
		return gcpStage4GRPCTalentCompletionService(path, grpcReqBody)
	case gcpTalentCreateClientEventMethod:
		return gcpStage4GRPCTalentEventService(path, grpcReqBody)
	case gcpServiceManagementListServicesMethod:
		return gcpStage4GRPCServiceManagementListServices(grpcReqBody)
	case gcpServiceManagementGetServiceMethod:
		return gcpStage4GRPCServiceManagementGetService(grpcReqBody)
	case gcpServiceManagementCreateServiceMethod:
		return gcpStage4GRPCServiceManagementCreateService(grpcReqBody)
	case gcpServiceManagementDeleteServiceMethod:
		return gcpStage4GRPCServiceManagementDeleteService(grpcReqBody)
	case gcpServiceManagementUndeleteServiceMethod:
		return gcpStage4GRPCServiceManagementUndeleteService(grpcReqBody)
	case gcpServiceManagementListServiceConfigsMethod:
		return gcpStage4GRPCServiceManagementListServiceConfigs(grpcReqBody)
	case gcpServiceManagementGetServiceConfigMethod:
		return gcpStage4GRPCServiceManagementGetServiceConfig(grpcReqBody)
	case gcpServiceManagementCreateServiceConfigMethod:
		return gcpStage4GRPCServiceManagementCreateServiceConfig(grpcReqBody)
	case gcpServiceManagementSubmitConfigSourceMethod:
		return gcpStage4GRPCServiceManagementSubmitConfigSource(grpcReqBody)
	case gcpServiceManagementListServiceRolloutsMethod:
		return gcpStage4GRPCServiceManagementListServiceRollouts(grpcReqBody)
	case gcpServiceManagementGetServiceRolloutMethod:
		return gcpStage4GRPCServiceManagementGetServiceRollout(grpcReqBody)
	case gcpServiceManagementCreateServiceRolloutMethod:
		return gcpStage4GRPCServiceManagementCreateServiceRollout(grpcReqBody)
	case gcpServiceManagementGenerateConfigReportMethod:
		return gcpStage4GRPCServiceManagementGenerateConfigReport(grpcReqBody)
	case gcpServiceHealthListEventsMethod:
		return gcpStage4GRPCServiceHealthListEvents(grpcReqBody)
	case gcpServiceHealthGetEventMethod:
		return gcpStage4GRPCServiceHealthGetEvent(grpcReqBody)
	case gcpServiceHealthListOrganizationEventsMethod:
		return gcpStage4GRPCServiceHealthListOrganizationEvents(grpcReqBody)
	case gcpServiceHealthGetOrganizationEventMethod:
		return gcpStage4GRPCServiceHealthGetOrganizationEvent(grpcReqBody)
	case gcpServiceHealthListOrganizationImpactsMethod:
		return gcpStage4GRPCServiceHealthListOrganizationImpacts(grpcReqBody)
	case gcpServiceHealthGetOrganizationImpactMethod:
		return gcpStage4GRPCServiceHealthGetOrganizationImpact(grpcReqBody)
	case gcpServiceDirectoryCreateNamespaceMethod:
		return gcpStage4GRPCServiceDirectoryCreateNamespace(grpcReqBody)
	case gcpServiceDirectoryListNamespacesMethod:
		return gcpStage4GRPCServiceDirectoryListNamespaces(grpcReqBody)
	case gcpServiceDirectoryGetNamespaceMethod:
		return gcpStage4GRPCServiceDirectoryGetNamespace(grpcReqBody)
	case gcpServiceDirectoryUpdateNamespaceMethod:
		return gcpStage4GRPCServiceDirectoryUpdateNamespace(grpcReqBody)
	case gcpServiceDirectoryDeleteNamespaceMethod:
		return gcpStage4GRPCServiceDirectoryDeleteNamespace(grpcReqBody)
	case gcpServiceDirectoryCreateServiceMethod:
		return gcpStage4GRPCServiceDirectoryCreateService(grpcReqBody)
	case gcpServiceDirectoryListServicesMethod:
		return gcpStage4GRPCServiceDirectoryListServices(grpcReqBody)
	case gcpServiceDirectoryGetServiceMethod:
		return gcpStage4GRPCServiceDirectoryGetService(grpcReqBody)
	case gcpServiceDirectoryUpdateServiceMethod:
		return gcpStage4GRPCServiceDirectoryUpdateService(grpcReqBody)
	case gcpServiceDirectoryDeleteServiceMethod:
		return gcpStage4GRPCServiceDirectoryDeleteService(grpcReqBody)
	case gcpServiceDirectoryCreateEndpointMethod:
		return gcpStage4GRPCServiceDirectoryCreateEndpoint(grpcReqBody)
	case gcpServiceDirectoryListEndpointsMethod:
		return gcpStage4GRPCServiceDirectoryListEndpoints(grpcReqBody)
	case gcpServiceDirectoryGetEndpointMethod:
		return gcpStage4GRPCServiceDirectoryGetEndpoint(grpcReqBody)
	case gcpServiceDirectoryUpdateEndpointMethod:
		return gcpStage4GRPCServiceDirectoryUpdateEndpoint(grpcReqBody)
	case gcpServiceDirectoryDeleteEndpointMethod:
		return gcpStage4GRPCServiceDirectoryDeleteEndpoint(grpcReqBody)
	case gcpServiceDirectoryGetIAMPolicyMethod:
		return gcpStage4GRPCServiceDirectoryGetIAMPolicy(grpcReqBody)
	case gcpServiceDirectorySetIAMPolicyMethod:
		return gcpStage4GRPCServiceDirectorySetIAMPolicy(grpcReqBody)
	case gcpServiceDirectoryTestIAMPermissionsMethod:
		return gcpStage4GRPCServiceDirectoryTestIAMPermissions(grpcReqBody)
	case gcpServiceDirectoryResolveServiceMethod:
		return gcpStage4GRPCServiceDirectoryResolveService(grpcReqBody)
	case gcpSecretManagerListSecretsMethod:
		return gcpStage4GRPCSecretManagerListSecrets(grpcReqBody)
	case gcpSecretManagerCreateSecretMethod:
		return gcpStage4GRPCSecretManagerCreateSecret(grpcReqBody)
	case gcpSecretManagerAddSecretVersionMethod:
		return gcpStage4GRPCSecretManagerAddSecretVersion(grpcReqBody)
	case gcpSecretManagerGetSecretMethod:
		return gcpStage4GRPCSecretManagerGetSecret(grpcReqBody)
	case gcpSecretManagerUpdateSecretMethod:
		return gcpStage4GRPCSecretManagerUpdateSecret(grpcReqBody)
	case gcpSecretManagerDeleteSecretMethod:
		return gcpStage4GRPCSecretManagerDeleteSecret(grpcReqBody)
	case gcpSecretManagerListSecretVersionsMethod:
		return gcpStage4GRPCSecretManagerListSecretVersions(grpcReqBody)
	case gcpSecretManagerGetSecretVersionMethod:
		return gcpStage4GRPCSecretManagerGetSecretVersion(grpcReqBody)
	case gcpSecretManagerAccessSecretVersionMethod:
		return gcpStage4GRPCSecretManagerAccessSecretVersion(grpcReqBody)
	case gcpSecretManagerDisableSecretVersionMethod:
		return gcpStage4GRPCSecretManagerDisableSecretVersion(grpcReqBody)
	case gcpSecretManagerEnableSecretVersionMethod:
		return gcpStage4GRPCSecretManagerEnableSecretVersion(grpcReqBody)
	case gcpSecretManagerDestroySecretVersionMethod:
		return gcpStage4GRPCSecretManagerDestroySecretVersion(grpcReqBody)
	case gcpSecretManagerSetIAMPolicyMethod:
		return gcpStage4GRPCSecretManagerSetIAMPolicy(grpcReqBody)
	case gcpSecretManagerGetIAMPolicyMethod:
		return gcpStage4GRPCSecretManagerGetIAMPolicy(grpcReqBody)
	case gcpSecretManagerTestIAMPermissionsMethod:
		return gcpStage4GRPCSecretManagerTestIAMPermissions(grpcReqBody)
	case gcpSecurityPrivateCAListCaPoolsMethod:
		return gcpStage4GRPCSecurityPrivateCAListCaPools(grpcReqBody)
	case gcpSecurityPrivateCAGetCaPoolMethod:
		return gcpStage4GRPCSecurityPrivateCAGetCaPool(grpcReqBody)
	case gcpSecurityPrivateCACreateCaPoolMethod:
		return gcpStage4GRPCSecurityPrivateCACreateCaPool(grpcReqBody)
	case gcpSecurityPrivateCAListCertificateAuthoritiesMethod:
		return gcpStage4GRPCSecurityPrivateCAListCertificateAuthorities(grpcReqBody)
	case gcpSecurityPrivateCAGetCertificateAuthorityMethod:
		return gcpStage4GRPCSecurityPrivateCAGetCertificateAuthority(grpcReqBody)
	case gcpSecurityPrivateCACreateCertificateAuthorityMethod:
		return gcpStage4GRPCSecurityPrivateCACreateCertificateAuthority(grpcReqBody)
	case gcpSecurityPrivateCAListCertificatesMethod:
		return gcpStage4GRPCSecurityPrivateCAListCertificates(grpcReqBody)
	case gcpSecurityPrivateCAGetCertificateMethod:
		return gcpStage4GRPCSecurityPrivateCAGetCertificate(grpcReqBody)
	case gcpSecurityPrivateCACreateCertificateMethod:
		return gcpStage4GRPCSecurityPrivateCACreateCertificate(grpcReqBody)
	case gcpSecurityPrivateCARevokeCertificateMethod:
		return gcpStage4GRPCSecurityPrivateCARevokeCertificate(grpcReqBody)
	case gcpSecurityPublicCACreateExternalAccountKeyMethod:
		return gcpStage4GRPCSecurityPublicCACreateExternalAccountKey(grpcReqBody)
	case gcpSecurityCenterListSourcesMethod:
		return gcpStage4GRPCSecurityCenterListSources(grpcReqBody)
	case gcpSecurityCenterGetSourceMethod:
		return gcpStage4GRPCSecurityCenterGetSource(grpcReqBody)
	case gcpSecurityCenterCreateSourceMethod:
		return gcpStage4GRPCSecurityCenterCreateSource(grpcReqBody)
	case gcpSecurityCenterSetMuteMethod:
		return gcpStage4GRPCSecurityCenterSetMute(grpcReqBody)
	case gcpSecurityCenterV2ListSourcesMethod:
		return gcpStage4GRPCSecurityCenterV2ListSources(grpcReqBody)
	case gcpSecurityCenterV2GetSourceMethod:
		return gcpStage4GRPCSecurityCenterV2GetSource(grpcReqBody)
	case gcpSecurityCenterV2CreateSourceMethod:
		return gcpStage4GRPCSecurityCenterV2CreateSource(grpcReqBody)
	case gcpSecurityCenterV2SetMuteMethod:
		return gcpStage4GRPCSecurityCenterV2SetMute(grpcReqBody)
	case gcpSecurityCenterManagementListEffectiveSHAModulesMethod:
		return gcpStage4GRPCSecurityCenterManagementListEffectiveSHAModules(grpcReqBody)
	case gcpSecurityCenterManagementGetEffectiveSHAModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementGetEffectiveSHAModule(grpcReqBody)
	case gcpSecurityCenterManagementListSHAModulesMethod:
		return gcpStage4GRPCSecurityCenterManagementListSHAModules(grpcReqBody)
	case gcpSecurityCenterManagementListDescendantSHAModulesMethod:
		return gcpStage4GRPCSecurityCenterManagementListDescendantSHAModules(grpcReqBody)
	case gcpSecurityCenterManagementGetSHAModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementGetSHAModule(grpcReqBody)
	case gcpSecurityCenterManagementCreateSHAModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementCreateSHAModule(grpcReqBody)
	case gcpSecurityCenterManagementUpdateSHAModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementUpdateSHAModule(grpcReqBody)
	case gcpSecurityCenterManagementDeleteSHAModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementDeleteSHAModule(grpcReqBody)
	case gcpSecurityCenterManagementSimulateSHAModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementSimulateSHAModule(grpcReqBody)
	case gcpSecurityCenterManagementListEffectiveETDModulesMethod:
		return gcpStage4GRPCSecurityCenterManagementListEffectiveETDModules(grpcReqBody)
	case gcpSecurityCenterManagementGetEffectiveETDModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementGetEffectiveETDModule(grpcReqBody)
	case gcpSecurityCenterManagementListETDModulesMethod:
		return gcpStage4GRPCSecurityCenterManagementListETDModules(grpcReqBody)
	case gcpSecurityCenterManagementListDescendantETDModulesMethod:
		return gcpStage4GRPCSecurityCenterManagementListDescendantETDModules(grpcReqBody)
	case gcpSecurityCenterManagementGetETDModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementGetETDModule(grpcReqBody)
	case gcpSecurityCenterManagementCreateETDModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementCreateETDModule(grpcReqBody)
	case gcpSecurityCenterManagementUpdateETDModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementUpdateETDModule(grpcReqBody)
	case gcpSecurityCenterManagementDeleteETDModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementDeleteETDModule(grpcReqBody)
	case gcpSecurityCenterManagementValidateETDModuleMethod:
		return gcpStage4GRPCSecurityCenterManagementValidateETDModule(grpcReqBody)
	case gcpSecurityCenterManagementGetServiceMethod:
		return gcpStage4GRPCSecurityCenterManagementGetService(grpcReqBody)
	case gcpSecurityCenterManagementListServicesMethod:
		return gcpStage4GRPCSecurityCenterManagementListServices(grpcReqBody)
	case gcpSecurityCenterManagementUpdateServiceMethod:
		return gcpStage4GRPCSecurityCenterManagementUpdateService(grpcReqBody)
	case gcpSecurityPostureListPosturesMethod:
		return gcpStage4GRPCSecurityPostureListPostures(grpcReqBody)
	case gcpSecurityPostureListPostureRevisionsMethod:
		return gcpStage4GRPCSecurityPostureListPostureRevisions(grpcReqBody)
	case gcpSecurityPostureGetPostureMethod:
		return gcpStage4GRPCSecurityPostureGetPosture(grpcReqBody)
	case gcpSecurityPostureCreatePostureMethod:
		return gcpStage4GRPCSecurityPostureCreatePosture(grpcReqBody)
	case gcpSecurityPostureUpdatePostureMethod:
		return gcpStage4GRPCSecurityPostureUpdatePosture(grpcReqBody)
	case gcpSecurityPostureDeletePostureMethod:
		return gcpStage4GRPCSecurityPostureDeletePosture(grpcReqBody)
	case gcpSecurityPostureExtractPostureMethod:
		return gcpStage4GRPCSecurityPostureExtractPosture(grpcReqBody)
	case gcpSecurityPostureListPostureDeploymentsMethod:
		return gcpStage4GRPCSecurityPostureListPostureDeployments(grpcReqBody)
	case gcpSecurityPostureGetPostureDeploymentMethod:
		return gcpStage4GRPCSecurityPostureGetPostureDeployment(grpcReqBody)
	case gcpSecurityPostureCreatePostureDeploymentMethod:
		return gcpStage4GRPCSecurityPostureCreatePostureDeployment(grpcReqBody)
	case gcpSecurityPostureUpdatePostureDeploymentMethod:
		return gcpStage4GRPCSecurityPostureUpdatePostureDeployment(grpcReqBody)
	case gcpSecurityPostureDeletePostureDeploymentMethod:
		return gcpStage4GRPCSecurityPostureDeletePostureDeployment(grpcReqBody)
	case gcpSecurityPostureListPostureTemplatesMethod:
		return gcpStage4GRPCSecurityPostureListPostureTemplates(grpcReqBody)
	case gcpSecurityPostureGetPostureTemplateMethod:
		return gcpStage4GRPCSecurityPostureGetPostureTemplate(grpcReqBody)
	case gcpSecureSourceManagerListRepositoriesMethod:
		return gcpStage4GRPCSecureSourceManagerListRepositories(grpcReqBody)
	case gcpSecureSourceManagerGetRepositoryMethod:
		return gcpStage4GRPCSecureSourceManagerGetRepository(grpcReqBody)
	case gcpSecureSourceManagerGetIAMPolicyRepoMethod:
		return gcpStage4GRPCSecureSourceManagerGetIAMPolicyRepo(grpcReqBody)
	case gcpSecureSourceManagerListPullRequestsMethod:
		return gcpStage4GRPCSecureSourceManagerListPullRequests(grpcReqBody)
	case gcpSecureSourceManagerClosePullRequestMethod:
		return gcpStage4GRPCSecureSourceManagerClosePullRequest(grpcReqBody)
	case gcpRecommenderListInsightsMethod:
		return gcpStage4GRPCRecommenderListInsights(grpcReqBody)
	case gcpRecommenderGetInsightMethod:
		return gcpStage4GRPCRecommenderGetInsight(grpcReqBody)
	case gcpRecommenderMarkInsightAcceptedMethod:
		return gcpStage4GRPCRecommenderMarkInsightAccepted(grpcReqBody)
	case gcpRecommenderListRecommendationsMethod:
		return gcpStage4GRPCRecommenderListRecommendations(grpcReqBody)
	case gcpRecommenderGetRecommendationMethod:
		return gcpStage4GRPCRecommenderGetRecommendation(grpcReqBody)
	case gcpRecommenderMarkRecommendationDismissedMethod:
		return gcpStage4GRPCRecommenderMarkRecommendationDismissed(grpcReqBody)
	case gcpRecommenderMarkRecommendationClaimedMethod:
		return gcpStage4GRPCRecommenderMarkRecommendationClaimed(grpcReqBody)
	case gcpRecommenderMarkRecommendationSucceededMethod:
		return gcpStage4GRPCRecommenderMarkRecommendationSucceeded(grpcReqBody)
	case gcpRecommenderMarkRecommendationFailedMethod:
		return gcpStage4GRPCRecommenderMarkRecommendationFailed(grpcReqBody)
	case gcpRecommenderGetRecommenderConfigMethod:
		return gcpStage4GRPCRecommenderGetRecommenderConfig(grpcReqBody)
	case gcpRecommenderUpdateRecommenderConfigMethod:
		return gcpStage4GRPCRecommenderUpdateRecommenderConfig(grpcReqBody)
	case gcpRecommenderGetInsightTypeConfigMethod:
		return gcpStage4GRPCRecommenderGetInsightTypeConfig(grpcReqBody)
	case gcpRecommenderUpdateInsightTypeConfigMethod:
		return gcpStage4GRPCRecommenderUpdateInsightTypeConfig(grpcReqBody)
	case gcpRecaptchaEnterpriseCreateAssessmentMethod:
		return gcpStage4GRPCRecaptchaEnterpriseCreateAssessment(grpcReqBody)
	case gcpRecaptchaEnterpriseAnnotateAssessmentMethod:
		return gcpStage4GRPCRecaptchaEnterpriseAnnotateAssessment(grpcReqBody)
	case gcpRecaptchaEnterpriseCreateKeyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseCreateKey(grpcReqBody)
	case gcpRecaptchaEnterpriseListKeysMethod:
		return gcpStage4GRPCRecaptchaEnterpriseListKeys(grpcReqBody)
	case gcpRecaptchaEnterpriseRetrieveLegacySecretKeyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseRetrieveLegacySecretKey(grpcReqBody)
	case gcpRecaptchaEnterpriseGetKeyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseGetKey(grpcReqBody)
	case gcpRecaptchaEnterpriseUpdateKeyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseUpdateKey(grpcReqBody)
	case gcpRecaptchaEnterpriseDeleteKeyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseDeleteKey(grpcReqBody)
	case gcpRecaptchaEnterpriseMigrateKeyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseMigrateKey(grpcReqBody)
	case gcpRecaptchaEnterpriseAddIpOverrideMethod:
		return gcpStage4GRPCRecaptchaEnterpriseAddIpOverride(grpcReqBody)
	case gcpRecaptchaEnterpriseRemoveIpOverrideMethod:
		return gcpStage4GRPCRecaptchaEnterpriseRemoveIpOverride(grpcReqBody)
	case gcpRecaptchaEnterpriseListIpOverridesMethod:
		return gcpStage4GRPCRecaptchaEnterpriseListIpOverrides(grpcReqBody)
	case gcpRecaptchaEnterpriseGetMetricsMethod:
		return gcpStage4GRPCRecaptchaEnterpriseGetMetrics(grpcReqBody)
	case gcpRecaptchaEnterpriseCreateFirewallPolicyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseCreateFirewallPolicy(grpcReqBody)
	case gcpRecaptchaEnterpriseListFirewallPoliciesMethod:
		return gcpStage4GRPCRecaptchaEnterpriseListFirewallPolicies(grpcReqBody)
	case gcpRecaptchaEnterpriseGetFirewallPolicyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseGetFirewallPolicy(grpcReqBody)
	case gcpRecaptchaEnterpriseUpdateFirewallPolicyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseUpdateFirewallPolicy(grpcReqBody)
	case gcpRecaptchaEnterpriseDeleteFirewallPolicyMethod:
		return gcpStage4GRPCRecaptchaEnterpriseDeleteFirewallPolicy(grpcReqBody)
	case gcpRecaptchaEnterpriseReorderFirewallPoliciesMethod:
		return gcpStage4GRPCRecaptchaEnterpriseReorderFirewallPolicies(grpcReqBody)
	case gcpRecaptchaEnterpriseListRelatedAccountGroupsMethod:
		return gcpStage4GRPCRecaptchaEnterpriseListRelatedAccountGroups(grpcReqBody)
	case gcpRecaptchaEnterpriseListRelatedAccountGroupMembershipsMethod:
		return gcpStage4GRPCRecaptchaEnterpriseListRelatedAccountGroupMemberships(grpcReqBody)
	case gcpRecaptchaEnterpriseSearchRelatedAccountGroupMembershipsMethod:
		return gcpStage4GRPCRecaptchaEnterpriseSearchRelatedAccountGroupMemberships(grpcReqBody)
	case gcpLocationsGetLocationMethod:
		return gcpStage4GRPCLocationsGetLocation(grpcReqBody)
	case gcpLocationsListLocationsMethod:
		return gcpStage4GRPCLocationsListLocations(grpcReqBody)
	case gcpLongrunningGetOpMethod:
		return gcpStage4GRPCLongrunningGetOperation(grpcReqBody)
	case gcpLongrunningListOpsMethod:
		return gcpStage4GRPCLongrunningListOperations(grpcReqBody)
	case gcpLongrunningCancelOpMethod:
		return gcpStage4GRPCLongrunningCancelOperation(grpcReqBody)
	case gcpLongrunningDeleteOpMethod:
		return gcpStage4GRPCLongrunningDeleteOperation(grpcReqBody)
	default:
		if strings.HasPrefix(path, "/google.cloud.workstations.v1.Workstations/") {
			return gcpStage4GRPCWorkstations(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.vpcaccess.v1.VpcAccessService/") {
			return gcpStage4GRPCVPCAccess(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.webrisk.v1.WebRiskService/") {
			return gcpStage4GRPCWebRisk(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.websecurityscanner.v1.WebSecurityScanner/") {
			return gcpStage4GRPCWebSecurityScanner(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.vmwareengine.v1.VmwareEngine/") {
			return gcpStage4GRPCVMwareEngine(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.vmmigration.v1.VmMigration/") {
			return gcpStage4GRPCVMMigration(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.storagebatchoperations.v1.StorageBatchOperations/") {
			return gcpStage4GRPCStorageBatchOperations(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.storageinsights.v1.StorageInsights/") {
			return gcpStage4GRPCStorageInsights(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.storagetransfer.v1.StorageTransferService/") {
			return gcpStage4GRPCStorageTransfer(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.streetview.publish.v1.StreetViewPublishService/") {
			return gcpStage4GRPCStreetViewPublish(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.support.v2.CaseService/") {
			return gcpStage4GRPCSupportCaseService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.support.v2.CommentService/") {
			return gcpStage4GRPCSupportCommentService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.support.v2.CaseAttachmentService/") {
			return gcpStage4GRPCSupportCaseAttachmentService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.talent.v4.CompanyService/") {
			return gcpStage4GRPCTalentCompanyService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.talent.v4.TenantService/") {
			return gcpStage4GRPCTalentTenantService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.talent.v4.JobService/") {
			return gcpStage4GRPCTalentJobService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.talent.v4.Completion/") {
			return gcpStage4GRPCTalentCompletionService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.talent.v4.EventService/") {
			return gcpStage4GRPCTalentEventService(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.speech.v1.Speech/") {
			return gcpStage4GRPCSpeech(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.speech.v1.Adaptation/") {
			return gcpStage4GRPCSpeechAdaptation(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.texttospeech.v1.TextToSpeech/") {
			return gcpStage4GRPCTextToSpeech(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.texttospeech.v1.TextToSpeechLongAudioSynthesize/") {
			return gcpStage4GRPCTextToSpeech(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.video.livestream.v1.LivestreamService/") {
			return gcpStage4GRPCVideoLivestream(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.videointelligence.v1.VideoIntelligenceService/") {
			return gcpStage4GRPCVideoIntelligence(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.vision.v1.ImageAnnotator/") {
			return gcpStage4GRPCVision(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.vision.v1.ProductSearch/") {
			return gcpStage4GRPCVision(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.visionai.v1.HealthCheckService/") {
			return gcpStage4GRPCVisionAI(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.visionai.v1.StreamsService/") {
			return gcpStage4GRPCVisionAI(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.visionai.v1.AppPlatform/") {
			return gcpStage4GRPCVisionAI(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.visionai.v1.LiveVideoAnalytics/") {
			return gcpStage4GRPCVisionAI(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.visionai.v1.Warehouse/") {
			return gcpStage4GRPCVisionAI(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.visionai.v1.StreamingService/") {
			return gcpStage4GRPCVisionAI(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.video.transcoder.v1.TranscoderService/") {
			return gcpStage4GRPCVideoTranscoder(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.video.stitcher.v1.VideoStitcherService/") {
			return gcpStage4GRPCVideoStitcher(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.translation.v3.TranslationService/") {
			return gcpStage4GRPCTranslate(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.devtools.cloudtrace.v1.TraceService/") {
			return gcpStage4GRPCTraceV1(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.tpu.v1.Tpu/") {
			return gcpStage4GRPCTPU(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.devtools.cloudtrace.v2.TraceService/") {
			return gcpStage4GRPCTrace(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.speech.v2.Speech/") {
			return gcpStage4GRPCSpeechV2(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.cloud.telcoautomation.v1.TelcoAutomation/") {
			return gcpStage4GRPCTelcoAutomation(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.spanner.v1.Spanner/") {
			return gcpStage4GRPCSpanner(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.spanner.adapter.v1.Adapter/") {
			return gcpStage4GRPCSpannerAdapter(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.spanner.executor.v1.SpannerExecutorProxy/") {
			return gcpStage4GRPCSpannerExecutor(grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.spanner.admin.database.v1.DatabaseAdmin/") {
			return gcpStage4GRPCSpannerAdminDatabase(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.spanner.admin.instance.v1.InstanceAdmin/") {
			return gcpStage4GRPCSpannerAdminInstance(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.accounts.v1.") {
			return gcpStage4GRPCShoppingMerchantAccounts(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.conversions.v1.") {
			return gcpStage4GRPCShoppingMerchantConversions(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.datasources.v1.") {
			return gcpStage4GRPCShoppingMerchantDatasources(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.inventories.v1.") {
			return gcpStage4GRPCShoppingMerchantInventories(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.issueresolution.v1.") {
			return gcpStage4GRPCShoppingMerchantIssueresolution(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.notifications.v1.") {
			return gcpStage4GRPCShoppingMerchantNotifications(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.ordertracking.v1.") {
			return gcpStage4GRPCShoppingMerchantOrdertracking(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.promotions.v1.") {
			return gcpStage4GRPCShoppingMerchantPromotions(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.reports.v1.") {
			return gcpStage4GRPCShoppingMerchantReports(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.reviews.v1beta.") {
			return gcpStage4GRPCShoppingMerchantReviews(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.quota.v1.") {
			return gcpStage4GRPCShoppingMerchantQuota(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.products.v1.") {
			return gcpStage4GRPCShoppingMerchantProducts(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.productstudio.v1alpha.") {
			return gcpStage4GRPCShoppingMerchantProductstudio(path, grpcReqBody)
		}
		if strings.HasPrefix(path, "/google.shopping.merchant.lfp.v1.") {
			return gcpStage4GRPCShoppingMerchantLFP(path, grpcReqBody)
		}
		return nil, "", "", false
	}
}

func gcpStage4GRPCShoppingMerchantAccounts(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case "/google.shopping.merchant.accounts.v1.AccountsService/GetAccount":
		req := &accountspb.GetAccountRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name := strings.TrimSpace(req.GetName())
		account, ok := parseGCPShoppingMerchantAccountsAccountName(name)
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		return grpcProtoSuccess(&accountspb.Account{
			Name:         name,
			AccountId:    gcpShoppingMerchantAccountsNumericID(account, 123456),
			AccountName:  "Stackyard Merchant " + account,
			LanguageCode: "en-US",
		})
	case "/google.shopping.merchant.accounts.v1.UserService/ListUsers":
		req := &accountspb.ListUsersRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantAccountsAccountName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*accountspb.User{
			{
				Name:         fmt.Sprintf("accounts/%s/users/owner@example.com", account),
				State:        accountspb.User_VERIFIED,
				AccessRights: []accountspb.AccessRight{accountspb.AccessRight_ADMIN},
			},
			{
				Name:         fmt.Sprintf("accounts/%s/users/analyst@example.com", account),
				State:        accountspb.User_VERIFIED,
				AccessRights: []accountspb.AccessRight{accountspb.AccessRight_STANDARD},
			},
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&accountspb.ListUsersResponse{
			Users:         items[start:end],
			NextPageToken: next,
		})
	case "/google.shopping.merchant.accounts.v1.ProgramsService/EnableProgram":
		req := &accountspb.EnableProgramRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, _, ok := parseGCPShoppingMerchantAccountsProgramName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		return grpcProtoSuccess(&accountspb.Program{
			Name:              req.GetName(),
			State:             accountspb.Program_ENABLED,
			DocumentationUri:  "https://support.google.com/merchants/answer/13889434",
			ActiveRegionCodes: []string{"001"},
			UnmetRequirements: nil,
		})
	case "/google.shopping.merchant.accounts.v1.UserService/CreateUser":
		req := &accountspb.CreateUserRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantAccountsAccountName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		userID := strings.TrimSpace(req.GetUserId())
		if userID == "" {
			return grpcInvalidArgument("user_id-required")
		}
		if strings.Contains(strings.ToLower(userID), "existing") {
			return grpcAlreadyExists("user-already-exists")
		}
		return grpcProtoSuccess(&accountspb.User{
			Name:         fmt.Sprintf("accounts/%s/users/%s", account, userID),
			State:        accountspb.User_VERIFIED,
			AccessRights: []accountspb.AccessRight{accountspb.AccessRight_ADMIN},
		})
	}
	return gcpStage4GRPCShoppingMerchantAccountsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantAccountsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4GRPCShoppingMerchantConversions(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantConversionsCreateConversionSourceMethod:
		req := &conversionspb.CreateConversionSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantConversionsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		source := req.GetConversionSource()
		if source == nil {
			return grpcInvalidArgument("conversion_source-required")
		}
		switch {
		case source.GetGoogleAnalyticsLink() != nil:
			ga := source.GetGoogleAnalyticsLink()
			if ga.GetPropertyId() <= 0 {
				return grpcInvalidArgument("google_analytics_link.property_id-required")
			}
			if strings.HasPrefix(strconv.FormatInt(ga.GetPropertyId(), 10), "9") {
				return grpcAlreadyExists("conversion-source-already-exists")
			}
			return grpcProtoSuccess(gcpStage4ShoppingMerchantConversionsGALinkFixture(account, "galk:"+strconv.FormatInt(ga.GetPropertyId(), 10), conversionspb.ConversionSource_ACTIVE))
		case source.GetMerchantCenterDestination() != nil:
			mcd := source.GetMerchantCenterDestination()
			if strings.TrimSpace(mcd.GetDisplayName()) == "" {
				return grpcInvalidArgument("merchant_center_destination.display_name-required")
			}
			if !gcpShoppingMerchantConversionsCurrencyRe.MatchString(strings.ToUpper(strings.TrimSpace(mcd.GetCurrencyCode()))) {
				return grpcInvalidArgument("merchant_center_destination.currency_code-required")
			}
			if mcd.GetAttributionSettings() == nil {
				return grpcInvalidArgument("merchant_center_destination.attribution_settings-required")
			}
			if strings.Contains(strings.ToLower(mcd.GetDisplayName()), "existing") {
				return grpcAlreadyExists("conversion-source-already-exists")
			}
			return grpcProtoSuccess(gcpStage4ShoppingMerchantConversionsMCDFixture(account, "mcdn:1001", conversionspb.ConversionSource_ACTIVE, mcd.GetDisplayName(), strings.ToUpper(mcd.GetCurrencyCode())))
		default:
			return grpcInvalidArgument("source_data-required")
		}
	case gcpShoppingMerchantConversionsUpdateConversionSourceMethod:
		req := &conversionspb.UpdateConversionSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		source := req.GetConversionSource()
		if source == nil {
			return grpcInvalidArgument("conversion_source-required")
		}
		account, sourceID, ok := parseGCPShoppingMerchantConversionsName(strings.TrimSpace(source.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		if strings.HasPrefix(sourceID, "galk:") {
			return grpcFailedPrecondition("google-analytics-link-update-not-supported")
		}
		mcd := source.GetMerchantCenterDestination()
		if mcd == nil {
			return grpcInvalidArgument("merchant_center_destination-required")
		}
		if strings.TrimSpace(mcd.GetDisplayName()) == "" && strings.TrimSpace(mcd.GetCurrencyCode()) == "" {
			return grpcInvalidArgument("merchant_center_destination-update-payload-required")
		}
		displayName := strings.TrimSpace(mcd.GetDisplayName())
		if displayName == "" {
			displayName = "Stackyard Destination"
		}
		currency := strings.ToUpper(strings.TrimSpace(mcd.GetCurrencyCode()))
		if currency == "" {
			currency = "USD"
		}
		if !gcpShoppingMerchantConversionsCurrencyRe.MatchString(currency) {
			return grpcInvalidArgument("currency_code-invalid")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantConversionsMCDFixture(account, sourceID, conversionspb.ConversionSource_ACTIVE, displayName, currency))
	case gcpShoppingMerchantConversionsDeleteConversionSourceMethod:
		req := &conversionspb.DeleteConversionSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, sourceID, ok := parseGCPShoppingMerchantConversionsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(sourceID), "missing") {
			return grpcNotFound("conversion-source-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantConversionsUndeleteConversionSourceMethod:
		req := &conversionspb.UndeleteConversionSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, sourceID, ok := parseGCPShoppingMerchantConversionsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(sourceID), "missing") {
			return grpcNotFound("conversion-source-not-found")
		}
		if strings.HasPrefix(sourceID, "galk:") {
			return grpcFailedPrecondition("google-analytics-link-undelete-not-supported")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantConversionsMCDFixture(account, sourceID, conversionspb.ConversionSource_ACTIVE, "Primary Destination", "USD"))
	case gcpShoppingMerchantConversionsGetConversionSourceMethod:
		req := &conversionspb.GetConversionSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, sourceID, ok := parseGCPShoppingMerchantConversionsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(sourceID), "missing") {
			return grpcNotFound("conversion-source-not-found")
		}
		state := conversionspb.ConversionSource_ACTIVE
		if sourceID == "mcdn:1002" {
			state = conversionspb.ConversionSource_ARCHIVED
		}
		if strings.HasPrefix(sourceID, "galk:") {
			return grpcProtoSuccess(gcpStage4ShoppingMerchantConversionsGALinkFixture(account, sourceID, state))
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantConversionsMCDFixture(account, sourceID, state, "Primary Destination", "USD"))
	case gcpShoppingMerchantConversionsListConversionSourcesMethod:
		req := &conversionspb.ListConversionSourcesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantConversionsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*conversionspb.ConversionSource{
			gcpStage4ShoppingMerchantConversionsMCDFixture(account, "mcdn:1001", conversionspb.ConversionSource_ACTIVE, "Primary Destination", "USD"),
			gcpStage4ShoppingMerchantConversionsGALinkFixture(account, "galk:2001", conversionspb.ConversionSource_ACTIVE),
			gcpStage4ShoppingMerchantConversionsMCDFixture(account, "mcdn:1002", conversionspb.ConversionSource_ARCHIVED, "Archived Destination", "USD"),
		}
		filtered := make([]*conversionspb.ConversionSource, 0, len(items))
		for _, item := range items {
			if !req.GetShowDeleted() && item.GetState() == conversionspb.ConversionSource_ARCHIVED {
				continue
			}
			filtered = append(filtered, item)
		}
		if start > len(filtered) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(filtered)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(filtered) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&conversionspb.ListConversionSourcesResponse{
			ConversionSources: filtered[start:end],
			NextPageToken:     next,
		})
	}
	return gcpStage4GRPCShoppingMerchantConversionsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantConversionsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4GRPCShoppingMerchantDatasources(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantDatasourcesGetDataSourceMethod:
		req := &datasourcespb.GetDataSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, dataSourceID, ok := parseGCPShoppingMerchantDatasourcesName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(dataSourceID), "missing") {
			return grpcNotFound("data-source-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantDatasourcesDataSourceFixture(account, dataSourceID, "Stackyard Data Source "+dataSourceID))
	case gcpShoppingMerchantDatasourcesListDataSourcesMethod:
		req := &datasourcespb.ListDataSourcesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantDatasourcesParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*datasourcespb.DataSource{
			gcpStage4ShoppingMerchantDatasourcesDataSourceFixture(account, "1001", "Stackyard Data Source 1001"),
			gcpStage4ShoppingMerchantDatasourcesDataSourceFixture(account, "1002", "Stackyard Data Source 1002"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&datasourcespb.ListDataSourcesResponse{
			DataSources:   items[start:end],
			NextPageToken: next,
		})
	case gcpShoppingMerchantDatasourcesCreateDataSourceMethod:
		req := &datasourcespb.CreateDataSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantDatasourcesParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		source := req.GetDataSource()
		if source == nil {
			return grpcInvalidArgument("data_source-required")
		}
		if strings.TrimSpace(source.GetDisplayName()) == "" {
			return grpcInvalidArgument("display_name-required")
		}
		if source.GetType() == nil {
			return grpcInvalidArgument("data_source.type-required")
		}
		if strings.Contains(strings.ToLower(source.GetDisplayName()), "existing") {
			return grpcAlreadyExists("data-source-already-exists")
		}
		if name := strings.TrimSpace(source.GetName()); name != "" {
			expectedPrefix := fmt.Sprintf("accounts/%s/dataSources/", account)
			if !strings.HasPrefix(name, expectedPrefix) {
				return grpcInvalidArgument("name-parent-mismatch")
			}
		}
		dataSourceID := "1001"
		if parsedName := strings.TrimSpace(source.GetName()); parsedName != "" {
			if _, parsedID, parsed := parseGCPShoppingMerchantDatasourcesName(parsedName); parsed {
				dataSourceID = parsedID
			}
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantDatasourcesDataSourceFixture(account, dataSourceID, source.GetDisplayName()))
	case gcpShoppingMerchantDatasourcesUpdateDataSourceMethod:
		req := &datasourcespb.UpdateDataSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		source := req.GetDataSource()
		if source == nil {
			return grpcInvalidArgument("data_source-required")
		}
		account, dataSourceID, ok := parseGCPShoppingMerchantDatasourcesName(strings.TrimSpace(source.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		displayName := strings.TrimSpace(source.GetDisplayName())
		if displayName == "" {
			return grpcInvalidArgument("display_name-required")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantDatasourcesDataSourceFixture(account, dataSourceID, displayName))
	case gcpShoppingMerchantDatasourcesDeleteDataSourceMethod:
		req := &datasourcespb.DeleteDataSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, dataSourceID, ok := parseGCPShoppingMerchantDatasourcesName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(dataSourceID), "missing") {
			return grpcNotFound("data-source-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantDatasourcesFetchDataSourceMethod:
		req := &datasourcespb.FetchDataSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, dataSourceID, ok := parseGCPShoppingMerchantDatasourcesName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(dataSourceID), "missing") {
			return grpcNotFound("data-source-not-found")
		}
		if strings.Contains(strings.ToLower(dataSourceID), "nofile") {
			return grpcFailedPrecondition("fetch-requires-file-input")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantDatasourcesGetFileUploadMethod:
		req := &datasourcespb.GetFileUploadRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, dataSourceID, alias, ok := parseGCPShoppingMerchantDatasourcesFileUploadName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if alias != "latest" {
			return grpcInvalidArgument("file_upload_alias-latest-required")
		}
		if strings.Contains(strings.ToLower(dataSourceID), "missing") {
			return grpcNotFound("data-source-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantDatasourcesFileUploadFixture(account, dataSourceID))
	}
	return gcpStage4GRPCShoppingMerchantDatasourcesDynamic(path)
}

func gcpStage4GRPCShoppingMerchantDatasourcesDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4GRPCShoppingMerchantIssueresolution(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantIssueresolutionRenderAccountIssuesMethod:
		req := &issueresolutionpb.RenderAccountIssuesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantIssueresolutionAccountName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if languageCode := strings.TrimSpace(req.GetLanguageCode()); languageCode != "" && !gcpShoppingMerchantIssueresolutionLanguageRe.MatchString(languageCode) {
			return grpcInvalidArgument("language_code-invalid")
		}
		if timeZone := strings.TrimSpace(req.GetTimeZone()); timeZone != "" && !gcpShoppingMerchantIssueresolutionTimeZoneRe.MatchString(timeZone) {
			return grpcInvalidArgument("time_zone-invalid")
		}
		return grpcProtoSuccess(&issueresolutionpb.RenderAccountIssuesResponse{
			RenderedIssues: []*issueresolutionpb.RenderedIssue{
				gcpStage4ShoppingMerchantIssueresolutionRenderedIssueFixture("account", account, ""),
			},
		})
	case gcpShoppingMerchantIssueresolutionRenderProductIssuesMethod:
		req := &issueresolutionpb.RenderProductIssuesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, product, ok := parseGCPShoppingMerchantIssueresolutionProductName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if languageCode := strings.TrimSpace(req.GetLanguageCode()); languageCode != "" && !gcpShoppingMerchantIssueresolutionLanguageRe.MatchString(languageCode) {
			return grpcInvalidArgument("language_code-invalid")
		}
		if timeZone := strings.TrimSpace(req.GetTimeZone()); timeZone != "" && !gcpShoppingMerchantIssueresolutionTimeZoneRe.MatchString(timeZone) {
			return grpcInvalidArgument("time_zone-invalid")
		}
		return grpcProtoSuccess(&issueresolutionpb.RenderProductIssuesResponse{
			RenderedIssues: []*issueresolutionpb.RenderedIssue{
				gcpStage4ShoppingMerchantIssueresolutionRenderedIssueFixture("product", account, product),
			},
		})
	case gcpShoppingMerchantIssueresolutionTriggerActionMethod:
		req := &issueresolutionpb.TriggerActionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if _, ok := parseGCPShoppingMerchantIssueresolutionAccountName(strings.TrimSpace(req.GetName())); !ok {
			return grpcInvalidArgument("name-required")
		}
		if languageCode := strings.TrimSpace(req.GetLanguageCode()); languageCode != "" && !gcpShoppingMerchantIssueresolutionLanguageRe.MatchString(languageCode) {
			return grpcInvalidArgument("language_code-invalid")
		}
		payload := req.GetPayload()
		if payload == nil {
			return grpcInvalidArgument("payload-required")
		}
		actionContext := strings.TrimSpace(payload.GetActionContext())
		if actionContext == "" {
			return grpcInvalidArgument("payload.action_context-required")
		}
		if strings.Contains(strings.ToLower(actionContext), "missing") {
			return grpcNotFound("action-context-not-found")
		}
		if strings.Contains(strings.ToLower(actionContext), "locked") {
			return grpcFailedPrecondition("action-context-locked")
		}
		if actionContext != "ctx-account-review" && actionContext != "ctx-product-review" {
			return grpcNotFound("action-context-not-found")
		}
		actionInput := payload.GetActionInput()
		if actionInput == nil {
			return grpcInvalidArgument("payload.action_input-required")
		}
		actionFlowID := strings.TrimSpace(actionInput.GetActionFlowId())
		if actionFlowID == "" {
			return grpcInvalidArgument("payload.action_input.action_flow_id-required")
		}
		if actionFlowID != "flow-review" {
			return grpcInvalidArgument("payload.action_input.action_flow_id-unsupported")
		}
		inputValues := actionInput.GetInputValues()
		if len(inputValues) == 0 {
			return grpcInvalidArgument("payload.action_input.input_values-required")
		}
		hasExplanation := false
		for _, inputValue := range inputValues {
			if inputValue == nil {
				return grpcInvalidArgument("payload.action_input.input_values-invalid")
			}
			inputFieldID := strings.TrimSpace(inputValue.GetInputFieldId())
			if inputFieldID == "" {
				return grpcInvalidArgument("payload.action_input.input_values.input_field_id-required")
			}
			switch typed := inputValue.GetValue().(type) {
			case *issueresolutionpb.InputValue_TextInputValue_:
				if typed.TextInputValue == nil {
					return grpcInvalidArgument("payload.action_input.input_values.text_input_value-required")
				}
				text := strings.TrimSpace(typed.TextInputValue.GetValue())
				if inputFieldID == "explanation" && text == "" {
					return grpcInvalidArgument("payload.action_input.input_values.explanation-required")
				}
				if inputFieldID == "explanation" && text != "" {
					hasExplanation = true
				}
			case *issueresolutionpb.InputValue_ChoiceInputValue_:
				if typed.ChoiceInputValue == nil || strings.TrimSpace(typed.ChoiceInputValue.GetChoiceInputOptionId()) == "" {
					return grpcInvalidArgument("payload.action_input.input_values.choice_input_value.choice_input_option_id-required")
				}
			case *issueresolutionpb.InputValue_CheckboxInputValue_:
				if typed.CheckboxInputValue == nil {
					return grpcInvalidArgument("payload.action_input.input_values.checkbox_input_value-required")
				}
			default:
				return grpcInvalidArgument("payload.action_input.input_values.typed_value-required")
			}
		}
		if !hasExplanation {
			return grpcInvalidArgument("payload.action_input.input_values.explanation-required")
		}
		return grpcProtoSuccess(&issueresolutionpb.TriggerActionResponse{
			Message: fmt.Sprintf("action started for context %s with flow %s", actionContext, actionFlowID),
		})
	case gcpShoppingMerchantIssueresolutionListAggregateProductStatusesMethod:
		req := &issueresolutionpb.ListAggregateProductStatusesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantIssueresolutionAccountName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize := int(req.GetPageSize())
		if pageSize < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		if pageSize == 0 {
			pageSize = 25
		}
		if pageSize > 250 {
			pageSize = 250
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		filter := strings.TrimSpace(req.GetFilter())
		if !gcpShoppingMerchantIssueresolutionHasSupportedFilter(filter) {
			return grpcInvalidArgument("filter-unsupported-field")
		}
		filterReportingContext, filterCountry := gcpShoppingMerchantIssueresolutionExtractFilterTerms(filter)
		items := []*issueresolutionpb.AggregateProductStatus{
			gcpStage4ShoppingMerchantIssueresolutionAggregateStatusFixture(account, "shopping_ads-us", shoppingtypepb.ReportingContext_SHOPPING_ADS, "US"),
			gcpStage4ShoppingMerchantIssueresolutionAggregateStatusFixture(account, "free_listings-us", shoppingtypepb.ReportingContext_FREE_LISTINGS, "US"),
			gcpStage4ShoppingMerchantIssueresolutionAggregateStatusFixture(account, "shopping_ads-ca", shoppingtypepb.ReportingContext_SHOPPING_ADS, "CA"),
		}
		filtered := make([]*issueresolutionpb.AggregateProductStatus, 0, len(items))
		for _, item := range items {
			if filterReportingContext != "" && strings.ToUpper(strings.TrimSpace(item.GetReportingContext().String())) != filterReportingContext {
				continue
			}
			if filterCountry != "" && strings.ToUpper(strings.TrimSpace(item.GetCountry())) != filterCountry {
				continue
			}
			filtered = append(filtered, item)
		}
		if start > len(filtered) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(filtered)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(filtered) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&issueresolutionpb.ListAggregateProductStatusesResponse{
			AggregateProductStatuses: filtered[start:end],
			NextPageToken:            next,
		})
	}
	return gcpStage4GRPCShoppingMerchantIssueresolutionDynamic(path)
}

func gcpStage4GRPCShoppingMerchantIssueresolutionDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4GRPCShoppingMerchantNotifications(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantNotificationsGetNotificationSubscriptionMethod:
		req := &notificationspb.GetNotificationSubscriptionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, subscriptionID, ok := parseGCPShoppingMerchantNotificationsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(subscriptionID), "missing") {
			return grpcNotFound("notification-subscription-not-found")
		}
		event := int32(notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE)
		callbackURI := "https://example.com/hooks/merchant-notifications"
		allManagedAccounts := true
		targetAccount := ""
		if strings.Contains(subscriptionID, "target-") {
			callbackURI = "https://example.com/hooks/merchant-notifications-target"
			allManagedAccounts = false
			targetAccount = "accounts/567890"
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID, event, callbackURI, allManagedAccounts, targetAccount))
	case gcpShoppingMerchantNotificationsCreateNotificationSubscriptionMethod:
		req := &notificationspb.CreateNotificationSubscriptionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantNotificationsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		subscription := req.GetNotificationSubscription()
		if subscription == nil {
			return grpcInvalidArgument("notification_subscription-required")
		}
		event := int32(subscription.GetRegisteredEvent())
		if event <= 0 {
			return grpcInvalidArgument("registered_event-required")
		}
		callbackURI := strings.TrimSpace(subscription.GetCallBackUri())
		if !isGCPShoppingMerchantNotificationsCallbackURI(callbackURI) {
			return grpcInvalidArgument("call_back_uri-invalid")
		}
		allManagedAccounts, targetAccount, valid := gcpStage4ShoppingMerchantNotificationsInterestedIn(subscription)
		if !valid {
			return grpcInvalidArgument("interested_in-required")
		}
		subscriptionID := gcpShoppingMerchantNotificationsSubscriptionIDFor(event, allManagedAccounts, targetAccount)
		if strings.Contains(strings.ToLower(callbackURI), "existing") || strings.Contains(strings.ToLower(subscriptionID), "existing") {
			return grpcAlreadyExists("notification-subscription-already-exists")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID, event, callbackURI, allManagedAccounts, targetAccount))
	case gcpShoppingMerchantNotificationsUpdateNotificationSubscriptionMethod:
		req := &notificationspb.UpdateNotificationSubscriptionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		subscription := req.GetNotificationSubscription()
		if subscription == nil {
			return grpcInvalidArgument("notification_subscription-required")
		}
		account, subscriptionID, ok := parseGCPShoppingMerchantNotificationsName(strings.TrimSpace(subscription.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(subscriptionID), "missing") {
			return grpcNotFound("notification-subscription-not-found")
		}
		updateMaskPaths := gcpStage4ShoppingMerchantNotificationsNormalizeUpdateMaskPaths(req.GetUpdateMask().GetPaths())
		if len(updateMaskPaths) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		for _, path := range updateMaskPaths {
			switch path {
			case "call_back_uri", "registered_event", "all_managed_accounts", "target_account":
				continue
			default:
				return grpcFailedPrecondition("update_mask-unsupported")
			}
		}
		event := int32(subscription.GetRegisteredEvent())
		if event <= 0 {
			return grpcInvalidArgument("registered_event-required")
		}
		callbackURI := strings.TrimSpace(subscription.GetCallBackUri())
		if !isGCPShoppingMerchantNotificationsCallbackURI(callbackURI) {
			return grpcInvalidArgument("call_back_uri-invalid")
		}
		allManagedAccounts, targetAccount, valid := gcpStage4ShoppingMerchantNotificationsInterestedIn(subscription)
		if !valid {
			return grpcInvalidArgument("interested_in-required")
		}
		for _, path := range updateMaskPaths {
			switch path {
			case "all_managed_accounts":
				if !allManagedAccounts {
					return grpcInvalidArgument("all_managed_accounts-must-be-true")
				}
			case "target_account":
				if targetAccount == "" {
					return grpcInvalidArgument("target_account-required")
				}
			}
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID, event, callbackURI, allManagedAccounts, targetAccount))
	case gcpShoppingMerchantNotificationsDeleteNotificationSubscriptionMethod:
		req := &notificationspb.DeleteNotificationSubscriptionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, subscriptionID, ok := parseGCPShoppingMerchantNotificationsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(subscriptionID), "missing") {
			return grpcNotFound("notification-subscription-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantNotificationsListNotificationSubscriptionsMethod:
		req := &notificationspb.ListNotificationSubscriptionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantNotificationsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize := int(req.GetPageSize())
		if pageSize < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		if pageSize == 0 {
			pageSize = 100
		}
		if pageSize > 200 {
			pageSize = 200
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*notificationspb.NotificationSubscription{
			gcpStage4ShoppingMerchantNotificationsSubscriptionFixture(account, "all-managed-product-status-change", int32(notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE), "https://example.com/hooks/merchant-notifications", true, ""),
			gcpStage4ShoppingMerchantNotificationsSubscriptionFixture(account, "target-567890-product-status-change", int32(notificationspb.NotificationSubscription_PRODUCT_STATUS_CHANGE), "https://example.com/hooks/merchant-notifications-target", false, "accounts/567890"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&notificationspb.ListNotificationSubscriptionsResponse{
			NotificationSubscriptions: items[start:end],
			NextPageToken:             next,
		})
	case gcpShoppingMerchantNotificationsGetNotificationSubscriptionHealthMetricsMethod:
		req := &notificationspb.GetNotificationSubscriptionHealthMetricsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, subscriptionID, ok := parseGCPShoppingMerchantNotificationsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(subscriptionID), "missing") {
			return grpcNotFound("notification-subscription-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantNotificationsHealthMetricsFixture(account, subscriptionID))
	}
	return gcpStage4GRPCShoppingMerchantNotificationsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantNotificationsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4GRPCShoppingMerchantOrdertracking(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantOrdertrackingCreateOrderTrackingSignalMethod:
		req := &ordertrackingpb.CreateOrderTrackingSignalRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantOrdertrackingParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetOrderTrackingSignal() == nil {
			return grpcInvalidArgument("order_tracking_signal-required")
		}

		body := map[string]any{
			"orderTrackingSignal": gcpStage4ShoppingMerchantOrdertrackingSignalToMap(req.GetOrderTrackingSignal()),
		}
		if orderTrackingSignalID := strings.TrimSpace(req.GetOrderTrackingSignalId()); orderTrackingSignalID != "" {
			body["orderTrackingSignalId"] = orderTrackingSignalID
		}
		fixture, errType, errMessage := buildGCPShoppingMerchantOrdertrackingCreateFixture(account, body)
		if errType != "" {
			switch errType {
			case "AlreadyExists":
				return grpcAlreadyExists("order_tracking_signal-already-exists")
			case "NotFound":
				return grpcNotFound("merchant-account-not-found")
			case "FailedPrecondition":
				return grpcFailedPrecondition("shipment_line_item_mapping-invalid")
			default:
				return grpcInvalidArgument(gcpStage4ShoppingMerchantOrdertrackingInvalidReason(errMessage))
			}
		}

		out := &ordertrackingpb.OrderTrackingSignal{}
		if !gcpStage4ShoppingMerchantOrdertrackingFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	}
	return gcpStage4GRPCShoppingMerchantOrdertrackingDynamic(path)
}

func gcpStage4GRPCShoppingMerchantOrdertrackingDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantOrdertrackingInvalidReason(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(normalized, "ordertrackingsignal is required"):
		return "order_tracking_signal-required"
	case strings.Contains(normalized, "ordertrackingsignal.ordercreatedtime is required"):
		return "order_created_time-required"
	case strings.Contains(normalized, "ordertrackingsignal.orderid is required"):
		return "order_id-required"
	case strings.Contains(normalized, "ordertrackingsignal.shippinginfo must include at least one shipment"):
		return "shipping_info-required"
	case strings.Contains(normalized, "ordertrackingsignal.lineitems must include at least one line item"):
		return "line_items-required"
	case strings.Contains(normalized, "shipmentlineitemmapping"):
		return "shipment_line_item_mapping-invalid"
	default:
		return "order_tracking_signal-invalid"
	}
}

func gcpStage4ShoppingMerchantOrdertrackingSignalToMap(signal *ordertrackingpb.OrderTrackingSignal) map[string]any {
	if signal == nil {
		return nil
	}
	raw, err := (protojson.MarshalOptions{}).Marshal(signal)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func gcpStage4ShoppingMerchantOrdertrackingFixtureToProto(src map[string]any, dst proto.Message) bool {
	raw, err := json.Marshal(src)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, dst); err != nil {
		return false
	}
	return true
}

func gcpStage4GRPCShoppingMerchantProducts(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantProductsGetProductMethod:
		req := &productspb.GetProductRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, productID, ok := parseGCPShoppingMerchantProductsProductName(strings.TrimSpace(req.GetName()), "products")
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(productID), "missing") {
			return grpcNotFound("product-not-found")
		}
		fixture := gcpShoppingMerchantProductsProductFixture(account, productID, fmt.Sprintf("accounts/%s/dataSources/104628", account))
		out := &productspb.Product{}
		if !gcpStage4ShoppingMerchantProductsFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantProductsListProductsMethod:
		req := &productspb.ListProductsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantProductsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 25
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		items := []map[string]any{
			gcpShoppingMerchantProductsProductFixture(account, "en~US~sku-1001", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpShoppingMerchantProductsProductFixture(account, "en~US~sku-1002", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpShoppingMerchantProductsProductFixture(account, "local~en~US~sku-local-1003", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		products := make([]*productspb.Product, 0, end-start)
		for _, item := range items[start:end] {
			out := &productspb.Product{}
			if !gcpStage4ShoppingMerchantProductsFixtureToProto(item, out) {
				return nil, "", "", false
			}
			products = append(products, out)
		}
		return grpcProtoSuccess(&productspb.ListProductsResponse{
			Products:      products,
			NextPageToken: next,
		})
	case gcpShoppingMerchantProductsInsertProductInputMethod:
		req := &productspb.InsertProductInputRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantProductsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetProductInput() == nil {
			return grpcInvalidArgument("product_input-required")
		}
		dataSource, errType, _ := parseGCPShoppingMerchantProductsDataSourceQuery(account, req.GetDataSource())
		if errType != "" {
			return gcpStage4ShoppingMerchantProductsError(errType)
		}
		body := gcpStage4ShoppingMerchantProductsProtoToMap(req.GetProductInput())
		fixture, fixtureErrType, _ := buildGCPShoppingMerchantProductsProductInputFixture(account, dataSource, body, "insert", "", nil)
		if fixtureErrType != "" {
			return gcpStage4ShoppingMerchantProductsError(fixtureErrType)
		}
		out := &productspb.ProductInput{}
		if !gcpStage4ShoppingMerchantProductsFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantProductsUpdateProductInputMethod:
		req := &productspb.UpdateProductInputRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetProductInput() == nil {
			return grpcInvalidArgument("product_input-required")
		}
		account, productID, ok := parseGCPShoppingMerchantProductsProductName(strings.TrimSpace(req.GetProductInput().GetName()), "productInputs")
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		dataSource, errType, _ := parseGCPShoppingMerchantProductsDataSourceQuery(account, req.GetDataSource())
		if errType != "" {
			return gcpStage4ShoppingMerchantProductsError(errType)
		}
		maskFields := parseGCPStage4ShoppingMerchantProductsUpdateMaskPaths(req.GetUpdateMask().GetPaths())
		expectedName := fmt.Sprintf("accounts/%s/productInputs/%s", account, productID)
		body := gcpStage4ShoppingMerchantProductsProtoToMap(req.GetProductInput())
		fixture, fixtureErrType, _ := buildGCPShoppingMerchantProductsProductInputFixture(account, dataSource, body, "update", expectedName, maskFields)
		if fixtureErrType != "" {
			return gcpStage4ShoppingMerchantProductsError(fixtureErrType)
		}
		out := &productspb.ProductInput{}
		if !gcpStage4ShoppingMerchantProductsFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantProductsDeleteProductInputMethod:
		req := &productspb.DeleteProductInputRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, productID, ok := parseGCPShoppingMerchantProductsProductName(strings.TrimSpace(req.GetName()), "productInputs")
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		_, errType, _ := parseGCPShoppingMerchantProductsDataSourceQuery(account, req.GetDataSource())
		if errType != "" {
			return gcpStage4ShoppingMerchantProductsError(errType)
		}
		if strings.Contains(strings.ToLower(productID), "missing") {
			return grpcNotFound("product_input-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	}
	return gcpStage4GRPCShoppingMerchantProductsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantPromotions(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantPromotionsInsertPromotionMethod:
		req := &promotionspb.InsertPromotionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantPromotionsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPromotion() == nil {
			return grpcInvalidArgument("promotion-required")
		}
		dataSource, errType, _ := parseGCPShoppingMerchantPromotionsDataSource(account, req.GetDataSource())
		if errType != "" {
			return gcpStage4ShoppingMerchantPromotionsError(errType)
		}
		body := gcpStage4ShoppingMerchantPromotionsProtoToMap(req.GetPromotion())
		fixture, fixtureErrType, _ := buildGCPShoppingMerchantPromotionsPromotionFixture(account, dataSource, body)
		if fixtureErrType != "" {
			return gcpStage4ShoppingMerchantPromotionsError(fixtureErrType)
		}
		out := &promotionspb.Promotion{}
		if !gcpStage4ShoppingMerchantPromotionsFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantPromotionsGetPromotionMethod:
		req := &promotionspb.GetPromotionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, promotionToken, ok := parseGCPShoppingMerchantPromotionsName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(promotionToken), "missing") {
			return grpcNotFound("promotion-not-found")
		}
		fixture := gcpShoppingMerchantPromotionsPromotionFixture(account, promotionToken, fmt.Sprintf("accounts/%s/dataSources/104628", account))
		out := &promotionspb.Promotion{}
		if !gcpStage4ShoppingMerchantPromotionsFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantPromotionsListPromotionsMethod:
		req := &promotionspb.ListPromotionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantPromotionsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 50
		}
		if pageSize > 250 {
			pageSize = 250
		}
		items := []map[string]any{
			gcpShoppingMerchantPromotionsPromotionFixture(account, "en~US~promo-1001", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpShoppingMerchantPromotionsPromotionFixture(account, "en~US~promo-1002", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpShoppingMerchantPromotionsPromotionFixture(account, "en~CA~promo-1003", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		promotions := make([]*promotionspb.Promotion, 0, end-start)
		for _, item := range items[start:end] {
			out := &promotionspb.Promotion{}
			if !gcpStage4ShoppingMerchantPromotionsFixtureToProto(item, out) {
				return nil, "", "", false
			}
			promotions = append(promotions, out)
		}
		return grpcProtoSuccess(&promotionspb.ListPromotionsResponse{
			Promotions:    promotions,
			NextPageToken: next,
		})
	}
	return gcpStage4GRPCShoppingMerchantPromotionsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantPromotionsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantPromotionsError(errType string) ([]byte, string, string, bool) {
	switch errType {
	case "NotFound":
		return grpcNotFound("resource-not-found")
	case "FailedPrecondition":
		return grpcFailedPrecondition("request-failed-precondition")
	case "Aborted":
		return grpcAborted("version_number-stale")
	default:
		return grpcInvalidArgument("request-invalid")
	}
}

func gcpStage4ShoppingMerchantPromotionsProtoToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}
	raw, err := (protojson.MarshalOptions{}).Marshal(msg)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func gcpStage4ShoppingMerchantPromotionsFixtureToProto(src map[string]any, dst proto.Message) bool {
	raw, err := json.Marshal(src)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, dst); err != nil {
		return false
	}
	return true
}

func gcpStage4GRPCShoppingMerchantReports(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantReportsSearchMethod:
		req := &reportspb.SearchRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantReportsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") {
			return grpcNotFound("account-not-found")
		}
		if reason := validateGCPShoppingMerchantReportsQuery(req.GetQuery()); reason != "" {
			return grpcInvalidArgument("query-invalid")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 1000
		}
		if pageSize > 5000 {
			pageSize = 5000
		}
		items := []*reportspb.ReportRow{
			gcpStage4ShoppingMerchantReportsRow("online~en~US~sku-1001", "Stackyard Tee", 42, 420),
			gcpStage4ShoppingMerchantReportsRow("online~en~US~sku-1002", "Stackyard Hoodie", 21, 280),
			gcpStage4ShoppingMerchantReportsRow("online~en~US~sku-1003", "Stackyard Cap", 9, 120),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&reportspb.SearchResponse{
			Results:       items[start:end],
			NextPageToken: next,
		})
	}
	return gcpStage4GRPCShoppingMerchantReportsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantReportsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantReportsRow(id, title string, clicks, impressions int64) *reportspb.ReportRow {
	offerID := id
	if parts := strings.Split(strings.TrimSpace(id), "~"); len(parts) > 0 {
		offerID = parts[len(parts)-1]
	}
	ctr := 0.0
	if impressions > 0 {
		ctr = float64(clicks) / float64(impressions)
	}
	channel := shoppingtypepb.Channel_ONLINE
	return &reportspb.ReportRow{
		ProductView: &reportspb.ProductView{
			Id:           proto.String(id),
			Channel:      &channel,
			LanguageCode: proto.String("en"),
			FeedLabel:    proto.String("US"),
			OfferId:      proto.String(offerID),
			Title:        proto.String(title),
			Brand:        proto.String("Stackyard"),
			Availability: proto.String("IN_STOCK"),
		},
		ProductPerformanceView: &reportspb.ProductPerformanceView{
			OfferId:          proto.String(offerID),
			Title:            proto.String(title),
			Clicks:           proto.Int64(clicks),
			Impressions:      proto.Int64(impressions),
			ClickThroughRate: proto.Float64(ctr),
		},
		NonProductPerformanceView: &reportspb.NonProductPerformanceView{
			Clicks:           proto.Int64(4),
			Impressions:      proto.Int64(40),
			ClickThroughRate: proto.Float64(0.1),
		},
	}
}

func gcpStage4GRPCShoppingMerchantReviews(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantReviewsGetMerchantReviewMethod:
		req := &reviewspb.GetMerchantReviewRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, reviewID, ok := parseGCPShoppingMerchantReviewsMerchantName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") || strings.Contains(strings.ToLower(reviewID), "missing") {
			return grpcNotFound("review-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantReviewFixture(account, reviewID, fmt.Sprintf("accounts/%s/dataSources/104628", account)))
	case gcpShoppingMerchantReviewsListMerchantReviewsMethod:
		req := &reviewspb.ListMerchantReviewsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantReviewsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") {
			return grpcNotFound("account-not-found")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 100
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		items := []*reviewspb.MerchantReview{
			gcpStage4ShoppingMerchantReviewFixture(account, "merchant-review-1001", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpStage4ShoppingMerchantReviewFixture(account, "merchant-review-1002", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpStage4ShoppingMerchantReviewFixture(account, "merchant-review-1003", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&reviewspb.ListMerchantReviewsResponse{
			MerchantReviews: items[start:end],
			NextPageToken:   next,
		})
	case gcpShoppingMerchantReviewsInsertMerchantReviewMethod:
		req := &reviewspb.InsertMerchantReviewRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantReviewsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") {
			return grpcNotFound("account-not-found")
		}
		dsAccount, dsID, ok := parseGCPShoppingMerchantReviewsDataSource(strings.TrimSpace(req.GetDataSource()))
		if !ok {
			return grpcInvalidArgument("data_source-invalid")
		}
		if dsAccount != account {
			return grpcFailedPrecondition("data_source-account-mismatch")
		}
		in := req.GetMerchantReview()
		if in == nil {
			return grpcInvalidArgument("merchant_review-required")
		}
		reviewID := strings.TrimSpace(in.GetMerchantReviewId())
		if reviewID == "" {
			return grpcInvalidArgument("merchant_review_id-required")
		}
		if strings.Contains(strings.ToLower(reviewID), "missing") {
			return grpcNotFound("review-not-found")
		}
		if name := strings.TrimSpace(in.GetName()); name != "" {
			nameAccount, nameID, ok := parseGCPShoppingMerchantReviewsMerchantName(name)
			if !ok {
				return grpcInvalidArgument("merchant_review_name-invalid")
			}
			if nameAccount != account || nameID != reviewID {
				return grpcFailedPrecondition("merchant_review-name-mismatch")
			}
		}
		dataSource := fmt.Sprintf("accounts/%s/dataSources/%s", account, dsID)
		return grpcProtoSuccess(gcpStage4ShoppingMerchantReviewFromInsert(account, reviewID, dataSource, in))
	case gcpShoppingMerchantReviewsDeleteMerchantReviewMethod:
		req := &reviewspb.DeleteMerchantReviewRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, reviewID, ok := parseGCPShoppingMerchantReviewsMerchantName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") || strings.Contains(strings.ToLower(reviewID), "missing") {
			return grpcNotFound("review-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantReviewsGetProductReviewMethod:
		req := &reviewspb.GetProductReviewRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, reviewID, ok := parseGCPShoppingMerchantReviewsProductName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") || strings.Contains(strings.ToLower(reviewID), "missing") {
			return grpcNotFound("review-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingProductReviewFixture(account, reviewID, fmt.Sprintf("accounts/%s/dataSources/104628", account)))
	case gcpShoppingMerchantReviewsListProductReviewsMethod:
		req := &reviewspb.ListProductReviewsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantReviewsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") {
			return grpcNotFound("account-not-found")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 100
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		items := []*reviewspb.ProductReview{
			gcpStage4ShoppingProductReviewFixture(account, "product-review-1001", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpStage4ShoppingProductReviewFixture(account, "product-review-1002", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
			gcpStage4ShoppingProductReviewFixture(account, "product-review-1003", fmt.Sprintf("accounts/%s/dataSources/104628", account)),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&reviewspb.ListProductReviewsResponse{
			ProductReviews: items[start:end],
			NextPageToken:  next,
		})
	case gcpShoppingMerchantReviewsInsertProductReviewMethod:
		req := &reviewspb.InsertProductReviewRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantReviewsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") {
			return grpcNotFound("account-not-found")
		}
		dsAccount, dsID, ok := parseGCPShoppingMerchantReviewsDataSource(strings.TrimSpace(req.GetDataSource()))
		if !ok {
			return grpcInvalidArgument("data_source-invalid")
		}
		if dsAccount != account {
			return grpcFailedPrecondition("data_source-account-mismatch")
		}
		in := req.GetProductReview()
		if in == nil {
			return grpcInvalidArgument("product_review-required")
		}
		reviewID := strings.TrimSpace(in.GetProductReviewId())
		if reviewID == "" {
			return grpcInvalidArgument("product_review_id-required")
		}
		if strings.Contains(strings.ToLower(reviewID), "missing") {
			return grpcNotFound("review-not-found")
		}
		if name := strings.TrimSpace(in.GetName()); name != "" {
			nameAccount, nameID, ok := parseGCPShoppingMerchantReviewsProductName(name)
			if !ok {
				return grpcInvalidArgument("product_review_name-invalid")
			}
			if nameAccount != account || nameID != reviewID {
				return grpcFailedPrecondition("product_review-name-mismatch")
			}
		}
		dataSource := fmt.Sprintf("accounts/%s/dataSources/%s", account, dsID)
		return grpcProtoSuccess(gcpStage4ShoppingProductReviewFromInsert(account, reviewID, dataSource, in))
	case gcpShoppingMerchantReviewsDeleteProductReviewMethod:
		req := &reviewspb.DeleteProductReviewRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, reviewID, ok := parseGCPShoppingMerchantReviewsProductName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") || strings.Contains(strings.ToLower(reviewID), "missing") {
			return grpcNotFound("review-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	}
	return gcpStage4GRPCShoppingMerchantReviewsDynamic(path)
}

func gcpStage4GRPCShoppingMerchantReviewsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantReviewFixture(account, reviewID, dataSource string) *reviewspb.MerchantReview {
	collectionMethod := reviewspb.MerchantReviewAttributes_AFTER_FULFILLMENT
	return &reviewspb.MerchantReview{
		Name:             fmt.Sprintf("accounts/%s/merchantReviews/%s", account, reviewID),
		MerchantReviewId: reviewID,
		DataSource:       dataSource,
		MerchantReviewAttributes: &reviewspb.MerchantReviewAttributes{
			MerchantId:          proto.String("merchant-" + account),
			MerchantDisplayName: proto.String("Stackyard Merchant " + account),
			MerchantLink:        proto.String(fmt.Sprintf("https://merchant.stackyard.example/%s", account)),
			MerchantRatingLink:  proto.String(fmt.Sprintf("https://merchant.stackyard.example/%s/reviews", account)),
			MinRating:           proto.Int64(1),
			MaxRating:           proto.Int64(5),
			Rating:              proto.Float64(4.8),
			Title:               proto.String("Great merchant experience"),
			Content:             proto.String("Order arrived quickly and as described."),
			ReviewerId:          proto.String("reviewer-" + reviewID),
			ReviewerUsername:    proto.String("stackyard-user"),
			CollectionMethod:    &collectionMethod,
			ReviewTime:          timestamppb.New(time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)),
			ReviewLanguage:      proto.String("en"),
			ReviewCountry:       proto.String("US"),
		},
	}
}

func gcpStage4ShoppingMerchantReviewFromInsert(account, reviewID, dataSource string, in *reviewspb.MerchantReview) *reviewspb.MerchantReview {
	out := gcpStage4ShoppingMerchantReviewFixture(account, reviewID, dataSource)
	if in == nil {
		return out
	}
	if attrs := in.GetMerchantReviewAttributes(); attrs != nil {
		if value := strings.TrimSpace(attrs.GetTitle()); value != "" {
			out.MerchantReviewAttributes.Title = proto.String(value)
		}
		if value := strings.TrimSpace(attrs.GetContent()); value != "" {
			out.MerchantReviewAttributes.Content = proto.String(value)
		}
		if value := strings.TrimSpace(attrs.GetMerchantDisplayName()); value != "" {
			out.MerchantReviewAttributes.MerchantDisplayName = proto.String(value)
		}
		if value := strings.TrimSpace(attrs.GetMerchantLink()); value != "" {
			out.MerchantReviewAttributes.MerchantLink = proto.String(value)
		}
	}
	if len(in.GetCustomAttributes()) > 0 {
		out.CustomAttributes = in.GetCustomAttributes()
	}
	return out
}

func gcpStage4ShoppingProductReviewFixture(account, reviewID, dataSource string) *reviewspb.ProductReview {
	return &reviewspb.ProductReview{
		Name:            fmt.Sprintf("accounts/%s/productReviews/%s", account, reviewID),
		ProductReviewId: reviewID,
		DataSource:      dataSource,
		ProductReviewAttributes: &reviewspb.ProductReviewAttributes{
			AggregatorName:   proto.String("Stackyard Reviews Aggregator"),
			PublisherName:    proto.String("Stackyard"),
			ReviewerId:       proto.String("reviewer-" + reviewID),
			ReviewerUsername: proto.String("stackyard-user"),
			ReviewLanguage:   proto.String("en"),
			ReviewCountry:    proto.String("US"),
			ReviewTime:       timestamppb.New(time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)),
			Title:            proto.String("Excellent product quality"),
			Content:          proto.String("Fabric and fit exceeded expectations."),
			Pros:             []string{"Great quality", "Fast shipping"},
			Cons:             []string{"Limited colors"},
			ReviewLink: &reviewspb.ProductReviewAttributes_ReviewLink{
				Type: reviewspb.ProductReviewAttributes_ReviewLink_SINGLETON,
				Link: "https://merchant.stackyard.example/reviews/" + reviewID,
			},
			MinRating:        proto.Int64(1),
			MaxRating:        proto.Int64(5),
			Rating:           proto.Float64(4.6),
			ProductNames:     []string{"Stackyard Tee"},
			ProductLinks:     []string{"https://merchant.stackyard.example/products/stackyard-tee"},
			Skus:             []string{"sku-1001"},
			Brands:           []string{"Stackyard"},
			CollectionMethod: reviewspb.ProductReviewAttributes_POST_FULFILLMENT,
			TransactionId:    "txn-" + reviewID,
		},
	}
}

func gcpStage4ShoppingProductReviewFromInsert(account, reviewID, dataSource string, in *reviewspb.ProductReview) *reviewspb.ProductReview {
	out := gcpStage4ShoppingProductReviewFixture(account, reviewID, dataSource)
	if in == nil {
		return out
	}
	if attrs := in.GetProductReviewAttributes(); attrs != nil {
		if value := strings.TrimSpace(attrs.GetTitle()); value != "" {
			out.ProductReviewAttributes.Title = proto.String(value)
		}
		if value := strings.TrimSpace(attrs.GetContent()); value != "" {
			out.ProductReviewAttributes.Content = proto.String(value)
		}
		if value := strings.TrimSpace(attrs.GetPublisherName()); value != "" {
			out.ProductReviewAttributes.PublisherName = proto.String(value)
		}
		if value := strings.TrimSpace(attrs.GetReviewLanguage()); value != "" {
			out.ProductReviewAttributes.ReviewLanguage = proto.String(value)
		}
	}
	if len(in.GetCustomAttributes()) > 0 {
		out.CustomAttributes = in.GetCustomAttributes()
	}
	return out
}

func gcpStage4GRPCShoppingMerchantQuota(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantQuotaListQuotaGroupsMethod:
		req := &quotapb.ListQuotaGroupsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantQuotaParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if strings.Contains(strings.ToLower(account), "missing") {
			return grpcNotFound("account-not-found")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 500
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		items := []*quotapb.QuotaGroup{
			gcpStage4ShoppingMerchantQuotaGroup(account, "product-read", 15, 3000, 120, "products.list"),
			gcpStage4ShoppingMerchantQuotaGroup(account, "product-write", 3, 800, 45, "products.insert"),
			gcpStage4ShoppingMerchantQuotaGroup(account, "promotion-write", 2, 400, 20, "promotions.insert"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&quotapb.ListQuotaGroupsResponse{
			QuotaGroups:   items[start:end],
			NextPageToken: next,
		})
	}
	return gcpStage4GRPCShoppingMerchantQuotaDynamic(path)
}

func gcpStage4GRPCShoppingMerchantQuotaDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantQuotaGroup(account, group string, usage, limit, minuteLimit int64, method string) *quotapb.QuotaGroup {
	return &quotapb.QuotaGroup{
		Name:             gcpShoppingMerchantQuotaGroupName(account, group),
		QuotaUsage:       usage,
		QuotaLimit:       limit,
		QuotaMinuteLimit: minuteLimit,
		MethodDetails: []*quotapb.MethodDetails{
			{
				Method:  method,
				Version: "v1",
				Subapi:  "merchant-quota",
				Path:    "quota/v1/" + method,
			},
		},
	}
}

func gcpStage4GRPCShoppingMerchantProductsDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantProductsError(errType string) ([]byte, string, string, bool) {
	switch errType {
	case "NotFound":
		return grpcNotFound("resource-not-found")
	case "FailedPrecondition":
		return grpcFailedPrecondition("request-failed-precondition")
	case "Aborted":
		return grpcAborted("version_number-stale")
	default:
		return grpcInvalidArgument("request-invalid")
	}
}

func gcpStage4ShoppingMerchantProductsProtoToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}
	raw, err := (protojson.MarshalOptions{}).Marshal(msg)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func gcpStage4ShoppingMerchantProductsFixtureToProto(src map[string]any, dst proto.Message) bool {
	raw, err := json.Marshal(src)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, dst); err != nil {
		return false
	}
	return true
}

func parseGCPStage4ShoppingMerchantProductsUpdateMaskPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		field := normalizeGCPShoppingMerchantProductsUpdateMaskField(path)
		if field == "" {
			continue
		}
		normalized = append(normalized, field)
	}
	return normalized
}

func gcpStage4GRPCShoppingMerchantProductstudio(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantProductstudioGenerateProductImageBackgroundMethod:
		req := &productstudiopb.GenerateProductImageBackgroundRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantProductstudioName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		body := gcpStage4ShoppingMerchantProductstudioProtoToMap(req)
		fixture, errType, _ := buildGCPShoppingMerchantProductstudioImageFixture(account, "generateProductImageBackground", body)
		if errType != "" {
			return gcpStage4ShoppingMerchantProductstudioError(errType)
		}
		out := &productstudiopb.GenerateProductImageBackgroundResponse{}
		if !gcpStage4ShoppingMerchantProductstudioFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantProductstudioRemoveProductImageBackgroundMethod:
		req := &productstudiopb.RemoveProductImageBackgroundRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantProductstudioName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		body := gcpStage4ShoppingMerchantProductstudioProtoToMap(req)
		fixture, errType, _ := buildGCPShoppingMerchantProductstudioImageFixture(account, "removeProductImageBackground", body)
		if errType != "" {
			return gcpStage4ShoppingMerchantProductstudioError(errType)
		}
		out := &productstudiopb.RemoveProductImageBackgroundResponse{}
		if !gcpStage4ShoppingMerchantProductstudioFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantProductstudioUpscaleProductImageMethod:
		req := &productstudiopb.UpscaleProductImageRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantProductstudioName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		body := gcpStage4ShoppingMerchantProductstudioProtoToMap(req)
		fixture, errType, _ := buildGCPShoppingMerchantProductstudioImageFixture(account, "upscaleProductImage", body)
		if errType != "" {
			return gcpStage4ShoppingMerchantProductstudioError(errType)
		}
		out := &productstudiopb.UpscaleProductImageResponse{}
		if !gcpStage4ShoppingMerchantProductstudioFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	case gcpShoppingMerchantProductstudioGenerateProductTextSuggestionsMethod:
		req := &productstudiopb.GenerateProductTextSuggestionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantProductstudioName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		body := gcpStage4ShoppingMerchantProductstudioProtoToMap(req)
		fixture, errType, _ := buildGCPShoppingMerchantProductstudioTextFixture(account, body)
		if errType != "" {
			return gcpStage4ShoppingMerchantProductstudioError(errType)
		}
		out := &productstudiopb.GenerateProductTextSuggestionsResponse{}
		if !gcpStage4ShoppingMerchantProductstudioFixtureToProto(fixture, out) {
			return nil, "", "", false
		}
		return grpcProtoSuccess(out)
	}
	return gcpStage4GRPCShoppingMerchantProductstudioDynamic(path)
}

func gcpStage4GRPCShoppingMerchantProductstudioDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantProductstudioError(errType string) ([]byte, string, string, bool) {
	switch errType {
	case "NotFound":
		return grpcNotFound("resource-not-found")
	case "FailedPrecondition":
		return grpcFailedPrecondition("request-failed-precondition")
	case "Aborted":
		return grpcAborted("request-aborted")
	default:
		return grpcInvalidArgument("request-invalid")
	}
}

func gcpStage4ShoppingMerchantProductstudioProtoToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}
	raw, err := (protojson.MarshalOptions{}).Marshal(msg)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func gcpStage4ShoppingMerchantProductstudioFixtureToProto(src map[string]any, dst proto.Message) bool {
	raw, err := json.Marshal(src)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, dst); err != nil {
		return false
	}
	return true
}

func gcpStage4ShoppingMerchantNotificationsInterestedIn(subscription *notificationspb.NotificationSubscription) (allManagedAccounts bool, targetAccount string, ok bool) {
	switch interestedIn := subscription.GetInterestedIn().(type) {
	case *notificationspb.NotificationSubscription_AllManagedAccounts:
		if !interestedIn.AllManagedAccounts {
			return false, "", false
		}
		return true, "", true
	case *notificationspb.NotificationSubscription_TargetAccount:
		targetAccount = strings.TrimSpace(interestedIn.TargetAccount)
		if _, valid := parseGCPShoppingMerchantNotificationsParent(targetAccount); !valid {
			return false, "", false
		}
		return false, targetAccount, true
	default:
		return false, "", false
	}
}

func gcpStage4ShoppingMerchantNotificationsNormalizeUpdateMaskPaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		field := normalizeGCPShoppingMerchantNotificationsMaskField(path)
		if field == "" {
			continue
		}
		normalized = append(normalized, field)
	}
	return normalized
}

func gcpStage4ShoppingMerchantNotificationsSubscriptionFixture(account, subscriptionID string, event int32, callbackURI string, allManagedAccounts bool, targetAccount string) *notificationspb.NotificationSubscription {
	subscription := &notificationspb.NotificationSubscription{
		Name:            fmt.Sprintf("accounts/%s/notificationsubscriptions/%s", account, subscriptionID),
		RegisteredEvent: notificationspb.NotificationSubscription_NotificationEventType(event),
		CallBackUri:     callbackURI,
	}
	if allManagedAccounts {
		subscription.InterestedIn = &notificationspb.NotificationSubscription_AllManagedAccounts{
			AllManagedAccounts: true,
		}
	} else {
		subscription.InterestedIn = &notificationspb.NotificationSubscription_TargetAccount{
			TargetAccount: targetAccount,
		}
	}
	return subscription
}

func gcpStage4ShoppingMerchantNotificationsHealthMetricsFixture(account, subscriptionID string) *notificationspb.NotificationSubscriptionHealthMetrics {
	return &notificationspb.NotificationSubscriptionHealthMetrics{
		Name:                                   fmt.Sprintf("accounts/%s/notificationsubscriptions/%s", account, subscriptionID),
		AcknowledgedMessagesCount:              42,
		UndeliveredMessagesCount:               3,
		OldestUnacknowledgedMessageWaitingTime: 3600,
	}
}

func gcpStage4GRPCShoppingMerchantLFP(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantLFPGetLfpStoreMethod:
		req := &lfppb.GetLfpStoreRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, target, storeCode, ok := parseGCPShoppingMerchantLFPStoreName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(target), "missing") || strings.Contains(strings.ToLower(storeCode), "missing") {
			return grpcNotFound("lfp-store-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantLFPStoreFixture(account, gcpShoppingMerchantLFPTargetID(target, 567890), storeCode))
	case gcpShoppingMerchantLFPInsertLfpStoreMethod:
		req := &lfppb.InsertLfpStoreRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantLFPParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		store := req.GetLfpStore()
		if store == nil {
			return grpcInvalidArgument("lfp_store-required")
		}
		if store.GetTargetAccount() <= 0 {
			return grpcInvalidArgument("target_account-required")
		}
		storeCode := strings.TrimSpace(store.GetStoreCode())
		if storeCode == "" {
			return grpcInvalidArgument("store_code-required")
		}
		if !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
			return grpcInvalidArgument("store_code-invalid")
		}
		if strings.TrimSpace(store.GetStoreAddress()) == "" {
			return grpcInvalidArgument("store_address-required")
		}
		if name := strings.TrimSpace(store.GetName()); name != "" {
			expected := fmt.Sprintf("accounts/%s/lfpStores/%d~%s", account, store.GetTargetAccount(), storeCode)
			if name != expected {
				return grpcInvalidArgument("name-parent-or-store_code-mismatch")
			}
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantLFPStoreFixture(account, store.GetTargetAccount(), storeCode))
	case gcpShoppingMerchantLFPDeleteLfpStoreMethod:
		req := &lfppb.DeleteLfpStoreRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, target, storeCode, ok := parseGCPShoppingMerchantLFPStoreName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(target), "missing") || strings.Contains(strings.ToLower(storeCode), "missing") {
			return grpcNotFound("lfp-store-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantLFPListLfpStoresMethod:
		req := &lfppb.ListLfpStoresRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantLFPParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetTargetAccount() <= 0 {
			return grpcInvalidArgument("target_account-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize == 0 {
			pageSize = 250
		}
		if pageSize > 1000 {
			pageSize = 1000
		}
		items := []*lfppb.LfpStore{
			gcpStage4ShoppingMerchantLFPStoreFixture(account, req.GetTargetAccount(), "store-nyc"),
			gcpStage4ShoppingMerchantLFPStoreFixture(account, req.GetTargetAccount(), "store-sfo"),
			gcpStage4ShoppingMerchantLFPStoreFixture(account, req.GetTargetAccount(), "store-bos"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&lfppb.ListLfpStoresResponse{
			LfpStores:     items[start:end],
			NextPageToken: next,
		})
	case gcpShoppingMerchantLFPInsertLfpInventoryMethod:
		req := &lfppb.InsertLfpInventoryRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantLFPParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		inv := req.GetLfpInventory()
		if inv == nil {
			return grpcInvalidArgument("lfp_inventory-required")
		}
		if inv.GetTargetAccount() <= 0 {
			return grpcInvalidArgument("target_account-required")
		}
		storeCode := strings.TrimSpace(inv.GetStoreCode())
		if storeCode == "" {
			return grpcInvalidArgument("store_code-required")
		}
		if !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
			return grpcInvalidArgument("store_code-invalid")
		}
		if strings.Contains(strings.ToLower(storeCode), "missing") || strings.Contains(strings.ToLower(storeCode), "unknown") {
			return grpcFailedPrecondition("store_code-not-linked")
		}
		offerID := strings.TrimSpace(inv.GetOfferId())
		if offerID == "" {
			return grpcInvalidArgument("offer_id-required")
		}
		if !gcpShoppingMerchantLFPOfferIDRe.MatchString(offerID) {
			return grpcInvalidArgument("offer_id-invalid")
		}
		regionCode := strings.ToUpper(strings.TrimSpace(inv.GetRegionCode()))
		if !gcpShoppingMerchantLFPRegionCodeRe.MatchString(regionCode) {
			return grpcInvalidArgument("region_code-invalid")
		}
		contentLanguage := strings.TrimSpace(inv.GetContentLanguage())
		if !gcpShoppingMerchantLFPLanguageRe.MatchString(contentLanguage) {
			return grpcInvalidArgument("content_language-invalid")
		}
		if strings.TrimSpace(inv.GetAvailability()) == "" {
			return grpcInvalidArgument("availability-required")
		}
		if inv.Quantity != nil && inv.GetQuantity() < 0 {
			return grpcInvalidArgument("quantity-invalid")
		}
		if inv.GetPrice() != nil {
			currency := strings.ToUpper(strings.TrimSpace(inv.GetPrice().GetCurrencyCode()))
			if !gcpShoppingMerchantLFPCurrencyCode.MatchString(currency) {
				return grpcInvalidArgument("price.currency_code-invalid")
			}
			if inv.GetPrice().AmountMicros == nil || inv.GetPrice().GetAmountMicros() < 0 {
				return grpcInvalidArgument("price.amount_micros-invalid")
			}
		}
		if hasPickupMethod := inv.PickupMethod != nil; hasPickupMethod != (inv.PickupSla != nil) {
			return grpcFailedPrecondition("pickup_method-and-pickup_sla-required-together")
		}
		if name := strings.TrimSpace(inv.GetName()); name != "" {
			expected := fmt.Sprintf("accounts/%s/lfpInventories/%d~%s~%s", account, inv.GetTargetAccount(), storeCode, offerID)
			if name != expected {
				return grpcInvalidArgument("name-parent-or-resource-id-mismatch")
			}
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantLFPInventoryFixture(account, inv.GetTargetAccount(), storeCode, offerID, regionCode, contentLanguage, inv.GetAvailability()))
	case gcpShoppingMerchantLFPInsertLfpSaleMethod:
		req := &lfppb.InsertLfpSaleRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, ok := parseGCPShoppingMerchantLFPParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		sale := req.GetLfpSale()
		if sale == nil {
			return grpcInvalidArgument("lfp_sale-required")
		}
		if sale.GetTargetAccount() <= 0 {
			return grpcInvalidArgument("target_account-required")
		}
		storeCode := strings.TrimSpace(sale.GetStoreCode())
		if storeCode == "" {
			return grpcInvalidArgument("store_code-required")
		}
		if !gcpShoppingMerchantLFPStoreCodeRe.MatchString(storeCode) {
			return grpcInvalidArgument("store_code-invalid")
		}
		if strings.Contains(strings.ToLower(storeCode), "missing") || strings.Contains(strings.ToLower(storeCode), "unknown") {
			return grpcFailedPrecondition("store_code-not-linked")
		}
		offerID := strings.TrimSpace(sale.GetOfferId())
		if offerID == "" {
			return grpcInvalidArgument("offer_id-required")
		}
		if !gcpShoppingMerchantLFPOfferIDRe.MatchString(offerID) {
			return grpcInvalidArgument("offer_id-invalid")
		}
		if strings.Contains(strings.ToLower(offerID), "duplicate") {
			return grpcAlreadyExists("sale-already-exists")
		}
		regionCode := strings.ToUpper(strings.TrimSpace(sale.GetRegionCode()))
		if !gcpShoppingMerchantLFPRegionCodeRe.MatchString(regionCode) {
			return grpcInvalidArgument("region_code-invalid")
		}
		contentLanguage := strings.TrimSpace(sale.GetContentLanguage())
		if !gcpShoppingMerchantLFPLanguageRe.MatchString(contentLanguage) {
			return grpcInvalidArgument("content_language-invalid")
		}
		gtin := strings.TrimSpace(sale.GetGtin())
		if !gcpShoppingMerchantLFPGTINRe.MatchString(gtin) {
			return grpcInvalidArgument("gtin-invalid")
		}
		if sale.GetPrice() == nil {
			return grpcInvalidArgument("price-required")
		}
		currency := strings.ToUpper(strings.TrimSpace(sale.GetPrice().GetCurrencyCode()))
		if !gcpShoppingMerchantLFPCurrencyCode.MatchString(currency) {
			return grpcInvalidArgument("price.currency_code-invalid")
		}
		if sale.GetPrice().AmountMicros == nil || sale.GetPrice().GetAmountMicros() <= 0 {
			return grpcInvalidArgument("price.amount_micros-invalid")
		}
		if sale.GetSaleTime() == nil {
			return grpcInvalidArgument("sale_time-required")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantLFPSaleFixture(account, sale.GetTargetAccount(), storeCode, offerID, regionCode, contentLanguage, gtin, sale.GetSaleTime()))
	case gcpShoppingMerchantLFPGetLfpMerchantStateMethod:
		req := &lfppb.GetLfpMerchantStateRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, target, ok := parseGCPShoppingMerchantLFPMerchantStateName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(target), "missing") {
			return grpcNotFound("lfp-merchant-state-not-found")
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantLFPMerchantStateFixture(account, gcpShoppingMerchantLFPTargetID(target, 567890)))
	}
	return gcpStage4GRPCShoppingMerchantLFPDynamic(path)
}

func gcpStage4GRPCShoppingMerchantLFPDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantLFPStoreFixture(account string, targetAccount int64, storeCode string) *lfppb.LfpStore {
	return &lfppb.LfpStore{
		Name:          fmt.Sprintf("accounts/%s/lfpStores/%d~%s", account, targetAccount, storeCode),
		TargetAccount: targetAccount,
		StoreCode:     storeCode,
		StoreAddress:  "1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA",
		StoreName:     proto.String("Stackyard Downtown"),
		PhoneNumber:   proto.String("+15555550123"),
		WebsiteUri:    proto.String("https://example.com/store/" + storeCode),
		GcidCategory:  []string{"gcid:department_store"},
		PlaceId:       proto.String("ChIJ2eUgeAK6j4ARbn5u_wAGqWA"),
		MatchingState: lfppb.LfpStore_StoreMatchingState(2),
	}
}

func gcpStage4ShoppingMerchantLFPInventoryFixture(account string, targetAccount int64, storeCode, offerID, regionCode, contentLanguage, availability string) *lfppb.LfpInventory {
	return &lfppb.LfpInventory{
		Name:            fmt.Sprintf("accounts/%s/lfpInventories/%d~%s~%s", account, targetAccount, storeCode, offerID),
		TargetAccount:   targetAccount,
		StoreCode:       storeCode,
		OfferId:         offerID,
		RegionCode:      strings.ToUpper(regionCode),
		ContentLanguage: contentLanguage,
		Availability:    availability,
		Price: &shoppingtypepb.Price{
			CurrencyCode: proto.String("USD"),
			AmountMicros: proto.Int64(12990000),
		},
		Quantity:       proto.Int64(7),
		CollectionTime: timestamppb.New(time.Date(2026, time.January, 1, 15, 4, 5, 0, time.UTC)),
		PickupMethod:   proto.String("buy"),
		PickupSla:      proto.String("same day"),
		FeedLabel:      proto.String(strings.ToUpper(regionCode)),
	}
}

func gcpStage4ShoppingMerchantLFPSaleFixture(account string, targetAccount int64, storeCode, offerID, regionCode, contentLanguage, gtin string, saleTime *timestamppb.Timestamp) *lfppb.LfpSale {
	if saleTime == nil {
		saleTime = timestamppb.New(gcpStage4ReferenceTime)
	}
	return &lfppb.LfpSale{
		Name:            fmt.Sprintf("accounts/%s/lfpSales/%d~%s~%s", account, targetAccount, storeCode, offerID),
		TargetAccount:   targetAccount,
		StoreCode:       storeCode,
		OfferId:         offerID,
		RegionCode:      strings.ToUpper(regionCode),
		ContentLanguage: contentLanguage,
		Gtin:            gtin,
		Price: &shoppingtypepb.Price{
			CurrencyCode: proto.String("USD"),
			AmountMicros: proto.Int64(14990000),
		},
		Quantity:  saleTime.GetSeconds()%3 + 1,
		SaleTime:  saleTime,
		Uid:       proto.String(fmt.Sprintf("uid-%d-%s-%s", targetAccount, storeCode, offerID)),
		FeedLabel: proto.String(strings.ToUpper(regionCode)),
	}
}

func gcpStage4ShoppingMerchantLFPMerchantStateFixture(account string, targetAccount int64) *lfppb.LfpMerchantState {
	return &lfppb.LfpMerchantState{
		Name:       fmt.Sprintf("accounts/%s/lfpMerchantStates/%d", account, targetAccount),
		LinkedGbps: 2,
		StoreStates: []*lfppb.LfpMerchantState_LfpStoreState{
			{StoreCode: "store-nyc", MatchingState: lfppb.LfpMerchantState_LfpStoreState_StoreMatchingState(2)},
			{StoreCode: "store-sfo", MatchingState: lfppb.LfpMerchantState_LfpStoreState_StoreMatchingState(2)},
		},
		InventoryStats: &lfppb.LfpMerchantState_InventoryStats{
			SubmittedEntries:        42,
			SubmittedInStockEntries: 37,
			UnsubmittedEntries:      5,
			SubmittedProducts:       15,
		},
		CountrySettings: []*lfppb.LfpMerchantState_CountrySettings{
			{
				RegionCode:                      "US",
				FreeLocalListingsEnabled:        true,
				LocalInventoryAdsEnabled:        true,
				InventoryVerificationState:      lfppb.LfpMerchantState_CountrySettings_VerificationState(2),
				ProductPageType:                 lfppb.LfpMerchantState_CountrySettings_ProductPageType(1),
				InstockServingVerificationState: lfppb.LfpMerchantState_CountrySettings_VerificationState(2),
				PickupServingVerificationState:  lfppb.LfpMerchantState_CountrySettings_VerificationState(2),
			},
		},
	}
}

func gcpStage4ShoppingMerchantIssueresolutionRenderedIssueFixture(scope, account, product string) *issueresolutionpb.RenderedIssue {
	title := "Account issue: business information missing"
	context := "ctx-account-review"
	content := "<p>Update your business profile information to restore product serving.</p>"
	impactMessage := "Disapproves 25 offers in 2 countries"
	if scope == "product" {
		title = fmt.Sprintf("Product issue: policy warning for %s", product)
		context = "ctx-product-review"
		content = fmt.Sprintf("<p>Fix product %s data and request another review.</p>", product)
		impactMessage = "Disapproves 5 offers in 1 country"
	}

	return &issueresolutionpb.RenderedIssue{
		Title: title,
		Impact: &issueresolutionpb.Impact{
			Message:  impactMessage,
			Severity: issueresolutionpb.Severity_ERROR,
			Breakdowns: []*issueresolutionpb.Breakdown{
				{
					Regions: []*issueresolutionpb.Breakdown_Region{
						{Code: "US", Name: "United States"},
					},
					Details: []string{
						"Products not showing in Shopping ads",
						"Products not showing organically",
					},
				},
			},
		},
		Content: &issueresolutionpb.RenderedIssue_PrerenderedContent{
			PrerenderedContent: content,
		},
		Actions: []*issueresolutionpb.Action{
			{
				Action: &issueresolutionpb.Action_ExternalAction{
					ExternalAction: &issueresolutionpb.ExternalAction{
						Type: issueresolutionpb.ExternalAction_EXTERNAL_ACTION_TYPE_UNSPECIFIED,
						Uri:  fmt.Sprintf("https://merchants.google.com/mc/account/%s/issues", account),
					},
				},
				ButtonLabel: "Open Merchant Center",
				IsAvailable: true,
			},
			{
				Action: &issueresolutionpb.Action_BuiltinUserInputAction{
					BuiltinUserInputAction: &issueresolutionpb.BuiltInUserInputAction{
						ActionContext: context,
						Flows: []*issueresolutionpb.ActionFlow{
							{
								Id:                "flow-review",
								Label:             "I fixed the issue",
								DialogTitle:       "Request another review",
								DialogButtonLabel: "Request review",
								Inputs: []*issueresolutionpb.InputField{
									{
										Id:       "explanation",
										Required: true,
										Label: &issueresolutionpb.TextWithTooltip{
											Value: &issueresolutionpb.TextWithTooltip_SimpleValue{
												SimpleValue: "Explain what changed",
											},
										},
										ValueInput: &issueresolutionpb.InputField_TextInput_{
											TextInput: &issueresolutionpb.InputField_TextInput{
												Type:       issueresolutionpb.InputField_TextInput_TEXT_INPUT_TYPE_UNSPECIFIED,
												FormatInfo: proto.String("Provide details of the fix."),
											},
										},
									},
								},
							},
						},
					},
				},
				ButtonLabel: "Request review",
				IsAvailable: true,
			},
		},
	}
}

func gcpStage4ShoppingMerchantIssueresolutionAggregateStatusFixture(account, id string, reportingContext shoppingtypepb.ReportingContext_ReportingContextEnum, country string) *issueresolutionpb.AggregateProductStatus {
	return &issueresolutionpb.AggregateProductStatus{
		Name:             fmt.Sprintf("accounts/%s/aggregateProductStatuses/%s", account, id),
		ReportingContext: reportingContext,
		Country:          strings.ToUpper(strings.TrimSpace(country)),
		Stats: &issueresolutionpb.AggregateProductStatus_Stats{
			ActiveCount:      120,
			PendingCount:     5,
			DisapprovedCount: 7,
			ExpiringCount:    3,
		},
		ItemLevelIssues: []*issueresolutionpb.AggregateProductStatus_ItemLevelIssue{
			{
				Code:             "MISSING_IDENTIFIER",
				Severity:         issueresolutionpb.AggregateProductStatus_ItemLevelIssue_DISAPPROVED,
				Resolution:       issueresolutionpb.AggregateProductStatus_ItemLevelIssue_MERCHANT_ACTION,
				Attribute:        "gtin",
				Description:      "Missing product identifier",
				Detail:           "Provide GTIN to improve product quality and eligibility.",
				DocumentationUri: "https://support.google.com/merchants/answer/7052112",
				ProductCount:     5,
			},
		},
	}
}

func gcpStage4ShoppingMerchantDatasourcesDataSourceFixture(account, dataSourceID, displayName string) *datasourcespb.DataSource {
	if strings.TrimSpace(displayName) == "" {
		displayName = "Stackyard Data Source " + dataSourceID
	}
	numericID := int64(1001)
	if parsed, err := strconv.ParseInt(dataSourceID, 10, 64); err == nil && parsed > 0 {
		numericID = parsed
	}
	return &datasourcespb.DataSource{
		Name:         fmt.Sprintf("accounts/%s/dataSources/%s", account, dataSourceID),
		DataSourceId: numericID,
		DisplayName:  displayName,
		Input:        datasourcespb.DataSource_FILE,
		FileInput: &datasourcespb.FileInput{
			FileName:      "products.csv",
			FileInputType: datasourcespb.FileInput_UPLOAD,
		},
		Type: &datasourcespb.DataSource_PrimaryProductDataSource{
			PrimaryProductDataSource: &datasourcespb.PrimaryProductDataSource{
				FeedLabel:       proto.String("US"),
				ContentLanguage: proto.String("en"),
				Countries:       []string{"US"},
			},
		},
	}
}

func gcpStage4ShoppingMerchantDatasourcesFileUploadFixture(account, dataSourceID string) *datasourcespb.FileUpload {
	numericID := int64(1001)
	if parsed, err := strconv.ParseInt(dataSourceID, 10, 64); err == nil && parsed > 0 {
		numericID = parsed
	}
	return &datasourcespb.FileUpload{
		Name:            fmt.Sprintf("accounts/%s/dataSources/%s/fileUploads/latest", account, dataSourceID),
		DataSourceId:    numericID,
		ProcessingState: datasourcespb.FileUpload_SUCCEEDED,
		Issues: []*datasourcespb.FileUpload_Issue{
			{
				Title:            "Missing gtin",
				Description:      "Some products are missing gtin",
				Code:             "validation/missing_gtin",
				Count:            1,
				Severity:         datasourcespb.FileUpload_Issue_WARNING,
				DocumentationUri: "https://support.google.com/merchants/answer/7052112",
			},
		},
		ItemsTotal:   10,
		ItemsCreated: 6,
		ItemsUpdated: 4,
		UploadTime:   timestamppb.New(time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)),
	}
}

func gcpStage4GRPCShoppingMerchantInventories(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpShoppingMerchantInventoriesListLocalInventoriesMethod:
		req := &inventoriespb.ListLocalInventoriesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, product, ok := parseGCPShoppingMerchantInventoriesParentName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize > 25000 {
			pageSize = 25000
		}
		items := []*inventoriespb.LocalInventory{
			gcpStage4ShoppingMerchantInventoriesLocalFixture(account, product, "store-nyc"),
			gcpStage4ShoppingMerchantInventoriesLocalFixture(account, product, "store-sfo"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&inventoriespb.ListLocalInventoriesResponse{
			LocalInventories: items[start:end],
			NextPageToken:    next,
		})
	case gcpShoppingMerchantInventoriesInsertLocalInventoryMethod:
		req := &inventoriespb.InsertLocalInventoryRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, product, ok := parseGCPShoppingMerchantInventoriesParentName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		inv := req.GetLocalInventory()
		if inv == nil {
			return grpcInvalidArgument("local_inventory-required")
		}
		storeCode := strings.TrimSpace(inv.GetStoreCode())
		if storeCode == "" {
			return grpcInvalidArgument("store_code-required")
		}
		if !gcpShoppingMerchantInventoriesStoreCodeRe.MatchString(storeCode) {
			return grpcInvalidArgument("store_code-invalid")
		}
		if name := strings.TrimSpace(inv.GetName()); name != "" {
			expected := fmt.Sprintf("accounts/%s/products/%s/localInventories/%s", account, product, storeCode)
			if name != expected {
				return grpcInvalidArgument("name-parent-or-store_code-mismatch")
			}
		}
		attrs := inv.GetLocalInventoryAttributes()
		if attrs != nil {
			if attrs.Quantity != nil && attrs.GetQuantity() < 0 {
				return grpcInvalidArgument("quantity-invalid")
			}
			hasPickupMethod := attrs.PickupMethod != nil
			hasPickupSLA := attrs.PickupSla != nil
			if hasPickupMethod != hasPickupSLA {
				return grpcFailedPrecondition("pickup_method-and-pickup_sla-required-together")
			}
			hasSalePriceEffectiveDate := attrs.GetSalePriceEffectiveDate() != nil
			hasSalePrice := attrs.GetSalePrice() != nil
			if hasSalePriceEffectiveDate && !hasSalePrice {
				return grpcFailedPrecondition("sale_price-required-when-sale_price_effective_date-set")
			}
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantInventoriesLocalFixture(account, product, storeCode))
	case gcpShoppingMerchantInventoriesDeleteLocalInventoryMethod:
		req := &inventoriespb.DeleteLocalInventoryRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, _, storeCode, ok := parseGCPShoppingMerchantInventoriesLocalName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(storeCode), "missing") {
			return grpcNotFound("local-inventory-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpShoppingMerchantInventoriesListRegionalInventoriesMethod:
		req := &inventoriespb.ListRegionalInventoriesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, product, ok := parseGCPShoppingMerchantInventoriesParentName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		pageSize := int(req.GetPageSize())
		if pageSize > 100000 {
			pageSize = 100000
		}
		items := []*inventoriespb.RegionalInventory{
			gcpStage4ShoppingMerchantInventoriesRegionalFixture(account, product, "us-east1"),
			gcpStage4ShoppingMerchantInventoriesRegionalFixture(account, product, "us-west1"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&inventoriespb.ListRegionalInventoriesResponse{
			RegionalInventories: items[start:end],
			NextPageToken:       next,
		})
	case gcpShoppingMerchantInventoriesInsertRegionalInventoryMethod:
		req := &inventoriespb.InsertRegionalInventoryRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		account, product, ok := parseGCPShoppingMerchantInventoriesParentName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		inv := req.GetRegionalInventory()
		if inv == nil {
			return grpcInvalidArgument("regional_inventory-required")
		}
		region := strings.TrimSpace(inv.GetRegion())
		if region == "" {
			return grpcInvalidArgument("region-required")
		}
		if !gcpShoppingMerchantInventoriesRegionRe.MatchString(region) {
			return grpcInvalidArgument("region-invalid")
		}
		if name := strings.TrimSpace(inv.GetName()); name != "" {
			expected := fmt.Sprintf("accounts/%s/products/%s/regionalInventories/%s", account, product, region)
			if name != expected {
				return grpcInvalidArgument("name-parent-or-region-mismatch")
			}
		}
		attrs := inv.GetRegionalInventoryAttributes()
		if attrs != nil {
			hasSalePriceEffectiveDate := attrs.GetSalePriceEffectiveDate() != nil
			hasSalePrice := attrs.GetSalePrice() != nil
			if hasSalePriceEffectiveDate && !hasSalePrice {
				return grpcFailedPrecondition("sale_price-required-when-sale_price_effective_date-set")
			}
		}
		return grpcProtoSuccess(gcpStage4ShoppingMerchantInventoriesRegionalFixture(account, product, region))
	case gcpShoppingMerchantInventoriesDeleteRegionalInventoryMethod:
		req := &inventoriespb.DeleteRegionalInventoryRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, _, region, ok := parseGCPShoppingMerchantInventoriesRegionalName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(region), "missing") {
			return grpcNotFound("regional-inventory-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	}
	return gcpStage4GRPCShoppingMerchantInventoriesDynamic(path)
}

func gcpStage4GRPCShoppingMerchantInventoriesDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4ShoppingMerchantInventoriesLocalFixture(account, product, storeCode string) *inventoriespb.LocalInventory {
	return &inventoriespb.LocalInventory{
		Name:      fmt.Sprintf("accounts/%s/products/%s/localInventories/%s", account, product, storeCode),
		Account:   gcpStage4ShoppingMerchantInventoriesAccountID(account),
		StoreCode: storeCode,
		LocalInventoryAttributes: &inventoriespb.LocalInventoryAttributes{
			Quantity: proto.Int64(8),
		},
	}
}

func gcpStage4ShoppingMerchantInventoriesRegionalFixture(account, product, region string) *inventoriespb.RegionalInventory {
	return &inventoriespb.RegionalInventory{
		Name:    fmt.Sprintf("accounts/%s/products/%s/regionalInventories/%s", account, product, region),
		Account: gcpStage4ShoppingMerchantInventoriesAccountID(account),
		Region:  region,
		RegionalInventoryAttributes: &inventoriespb.RegionalInventoryAttributes{
			Availability: inventoriespb.RegionalInventoryAttributes_REGIONAL_INVENTORY_AVAILABILITY_UNSPECIFIED.Enum(),
		},
	}
}

func gcpStage4ShoppingMerchantInventoriesAccountID(account string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(account), 10, 64)
	if err != nil || parsed < 0 {
		return 123456
	}
	return parsed
}

func gcpStage4ShoppingMerchantConversionsGALinkFixture(account, sourceID string, state conversionspb.ConversionSource_State) *conversionspb.ConversionSource {
	propertyID := int64(2001)
	if strings.HasPrefix(sourceID, "galk:") {
		if parsed, err := strconv.ParseInt(strings.TrimPrefix(sourceID, "galk:"), 10, 64); err == nil && parsed > 0 {
			propertyID = parsed
		}
	}
	out := &conversionspb.ConversionSource{
		Name:  fmt.Sprintf("accounts/%s/conversionSources/%s", account, sourceID),
		State: state,
		SourceData: &conversionspb.ConversionSource_GoogleAnalyticsLink{
			GoogleAnalyticsLink: &conversionspb.GoogleAnalyticsLink{
				PropertyId:          propertyID,
				Property:            fmt.Sprintf("properties/%d", propertyID),
				AttributionSettings: gcpStage4ShoppingMerchantConversionsAttributionSettingsFixture(),
			},
		},
		Controller: conversionspb.ConversionSource_MERCHANT,
	}
	if state == conversionspb.ConversionSource_ARCHIVED {
		out.ExpireTime = timestamppb.New(time.Now().UTC().Add(30 * 24 * time.Hour))
	}
	return out
}

func gcpStage4ShoppingMerchantConversionsMCDFixture(account, sourceID string, state conversionspb.ConversionSource_State, displayName, currency string) *conversionspb.ConversionSource {
	if strings.TrimSpace(displayName) == "" {
		displayName = "Primary Destination"
	}
	if strings.TrimSpace(currency) == "" {
		currency = "USD"
	}
	out := &conversionspb.ConversionSource{
		Name:  fmt.Sprintf("accounts/%s/conversionSources/%s", account, sourceID),
		State: state,
		SourceData: &conversionspb.ConversionSource_MerchantCenterDestination{
			MerchantCenterDestination: &conversionspb.MerchantCenterDestination{
				Destination:         fmt.Sprintf("accounts/%s/conversionSources/%s/destination", account, sourceID),
				AttributionSettings: gcpStage4ShoppingMerchantConversionsAttributionSettingsFixture(),
				DisplayName:         displayName,
				CurrencyCode:        strings.ToUpper(currency),
			},
		},
		Controller: conversionspb.ConversionSource_MERCHANT,
	}
	if state == conversionspb.ConversionSource_ARCHIVED {
		out.ExpireTime = timestamppb.New(time.Now().UTC().Add(30 * 24 * time.Hour))
	}
	return out
}

func gcpStage4ShoppingMerchantConversionsAttributionSettingsFixture() *conversionspb.AttributionSettings {
	return &conversionspb.AttributionSettings{
		AttributionLookbackWindowDays: 30,
		AttributionModel:              conversionspb.AttributionSettings_CROSS_CHANNEL_LAST_CLICK,
		ConversionType: []*conversionspb.AttributionSettings_ConversionType{
			{Name: "purchase", Report: true},
		},
	}
}

func parseGCPShoppingMerchantConversionsParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.TrimSpace(parent), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantConversionsAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantConversionsName(name string) (account, sourceID string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "conversionSources" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	sourceID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantConversionsAccountRe.MatchString(account) || !gcpShoppingMerchantConversionsSourceIDRe.MatchString(sourceID) {
		return "", "", false
	}
	return account, sourceID, true
}

func parseGCPShoppingMerchantDatasourcesParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.TrimSpace(parent), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantDatasourcesAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantDatasourcesName(name string) (account, dataSourceID string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "dataSources" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	dataSourceID = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantDatasourcesAccountRe.MatchString(account) || !gcpShoppingMerchantDatasourcesDatasourceIDRe.MatchString(dataSourceID) {
		return "", "", false
	}
	return account, dataSourceID, true
}

func parseGCPShoppingMerchantDatasourcesFileUploadName(name string) (account, dataSourceID, alias string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "accounts" || parts[2] != "dataSources" || parts[4] != "fileUploads" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	dataSourceID = strings.TrimSpace(parts[3])
	alias = strings.TrimSpace(parts[5])
	if !gcpShoppingMerchantDatasourcesAccountRe.MatchString(account) || !gcpShoppingMerchantDatasourcesDatasourceIDRe.MatchString(dataSourceID) || alias == "" {
		return "", "", "", false
	}
	return account, dataSourceID, alias, true
}

func parseGCPShoppingMerchantIssueresolutionAccountName(name string) (account string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantIssueresolutionAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantIssueresolutionProductName(name string) (account, product string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "products" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantIssueresolutionAccountRe.MatchString(account) || !gcpShoppingMerchantIssueresolutionProductRe.MatchString(product) {
		return "", "", false
	}
	return account, product, true
}

func parseGCPShoppingMerchantInventoriesParentName(parent string) (account, product string, ok bool) {
	parts := strings.Split(strings.TrimSpace(parent), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "products" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) || !gcpShoppingMerchantInventoriesProductRe.MatchString(product) {
		return "", "", false
	}
	return account, product, true
}

func parseGCPShoppingMerchantInventoriesLocalName(name string) (account, product, storeCode string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "accounts" || parts[2] != "products" || parts[4] != "localInventories" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	storeCode = strings.TrimSpace(parts[5])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) ||
		!gcpShoppingMerchantInventoriesProductRe.MatchString(product) ||
		!gcpShoppingMerchantInventoriesStoreCodeRe.MatchString(storeCode) {
		return "", "", "", false
	}
	return account, product, storeCode, true
}

func parseGCPShoppingMerchantInventoriesRegionalName(name string) (account, product, region string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 6 || parts[0] != "accounts" || parts[2] != "products" || parts[4] != "regionalInventories" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	product = strings.TrimSpace(parts[3])
	region = strings.TrimSpace(parts[5])
	if !gcpShoppingMerchantInventoriesAccountRe.MatchString(account) ||
		!gcpShoppingMerchantInventoriesProductRe.MatchString(product) ||
		!gcpShoppingMerchantInventoriesRegionRe.MatchString(region) {
		return "", "", "", false
	}
	return account, product, region, true
}

func parseGCPShoppingMerchantLFPParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.TrimSpace(parent), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantLFPAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantLFPStoreName(name string) (account, target, storeCode string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "lfpStores" {
		return "", "", "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantLFPAccountRe.MatchString(account) {
		return "", "", "", false
	}
	target, storeCode, ok = parseGCPShoppingMerchantLFPStoreKey(strings.TrimSpace(parts[3]))
	if !ok {
		return "", "", "", false
	}
	return account, target, storeCode, true
}

func parseGCPShoppingMerchantLFPMerchantStateName(name string) (account, target string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "lfpMerchantStates" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	target = strings.TrimSpace(parts[3])
	if !gcpShoppingMerchantLFPAccountRe.MatchString(account) || !gcpShoppingMerchantLFPTargetRe.MatchString(target) {
		return "", "", false
	}
	return account, target, true
}

func parseGCPStage4GRPCServiceAndMethod(path string) (serviceName, methodName string, ok bool) {
	parts := strings.Split(strings.TrimSpace(path), "/")
	if len(parts) != 3 {
		return "", "", false
	}
	serviceName = strings.TrimSpace(parts[1])
	methodName = strings.TrimSpace(parts[2])
	if serviceName == "" || methodName == "" {
		return "", "", false
	}
	return serviceName, methodName, true
}

func parseGCPShoppingMerchantAccountsAccountName(name string) (account string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !isGCPShoppingMerchantAccountsID(account) {
		return "", false
	}
	return account, true
}

func parseGCPShoppingMerchantAccountsProgramName(name string) (account, programID string, ok bool) {
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) != 4 || parts[0] != "accounts" || parts[2] != "programs" {
		return "", "", false
	}
	account = strings.TrimSpace(parts[1])
	programID = strings.TrimSpace(parts[3])
	if !isGCPShoppingMerchantAccountsID(account) || !gcpShoppingMerchantAccountsProgramRe.MatchString(programID) {
		return "", "", false
	}
	return account, programID, true
}

func gcpStage4GRPCApigeeConnectListConnections(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &apigeeconnectpb.ListConnectionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, endpoint, ok := parseGCPStage4ApigeeConnectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*apigeeconnectpb.Connection{
		{
			Endpoint: fmt.Sprintf("projects/%s/endpoints/%s", project, endpoint),
			Cluster: &apigeeconnectpb.Cluster{
				Name:   "stackyard-cluster",
				Region: "us-central1",
			},
			StreamCount: 1,
		},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&apigeeconnectpb.ListConnectionsResponse{
		Connections:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCMediaTranslationStreamingTranslateSpeech(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &mediatranslationpb.StreamingTranslateSpeechRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	config := req.GetStreamingConfig()
	if config == nil {
		return grpcInvalidArgument("streaming_config-required")
	}
	audioConfig := config.GetAudioConfig()
	if audioConfig == nil || strings.TrimSpace(audioConfig.GetSourceLanguageCode()) == "" || strings.TrimSpace(audioConfig.GetTargetLanguageCode()) == "" {
		return grpcInvalidArgument("audio_config-required")
	}
	return grpcProtoSuccess(&mediatranslationpb.StreamingTranslateSpeechResponse{
		Result: &mediatranslationpb.StreamingTranslateSpeechResult{
			Result: &mediatranslationpb.StreamingTranslateSpeechResult_TextTranslationResult_{
				TextTranslationResult: &mediatranslationpb.StreamingTranslateSpeechResult_TextTranslationResult{
					Translation: "hola stackyard",
					IsFinal:     true,
				},
			},
		},
		SpeechEventType: mediatranslationpb.StreamingTranslateSpeechResponse_END_OF_SINGLE_UTTERANCE,
	})
}

func gcpStage4GRPCCloudProfilerListProfiles(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudprofilerpb.ListProfilesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPStage4ProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*cloudprofilerpb.Profile{
		gcpStage4CloudProfilerProfile(project, "stackyard-profile"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&cloudprofilerpb.ListProfilesResponse{
		Profiles:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCCloudProfilerCreateProfile(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudprofilerpb.CreateProfileRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPStage4ProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetDeployment() == nil {
		return grpcInvalidArgument("deployment-required")
	}
	if len(req.GetProfileType()) == 0 {
		return grpcInvalidArgument("profile_type-required")
	}
	profile := gcpStage4CloudProfilerProfile(project, "stackyard-profile")
	profile.ProfileType = req.GetProfileType()[0]
	profile.Deployment = req.GetDeployment()
	if profile.Deployment.GetProjectId() == "" {
		profile.Deployment.ProjectId = project
	}
	if profile.Deployment.GetTarget() == "" {
		profile.Deployment.Target = "stackyard-service"
	}
	return grpcProtoSuccess(profile)
}

func gcpStage4GRPCCloudProfilerCreateOfflineProfile(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudprofilerpb.CreateOfflineProfileRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPStage4ProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetProfile() == nil {
		return grpcInvalidArgument("profile-required")
	}
	if len(req.GetProfile().GetProfileBytes()) == 0 {
		return grpcInvalidArgument("profile_bytes-required")
	}
	profileID := "offline-profile"
	if name := strings.TrimSpace(req.GetProfile().GetName()); name != "" {
		parsedID, parsedProject, valid := parseGCPStage4CloudProfilerProfileName(name)
		if !valid {
			return grpcInvalidArgument("profile_name-invalid")
		}
		profileID = parsedID
		project = parsedProject
	}
	response := gcpStage4CloudProfilerProfile(project, profileID)
	if req.GetProfile().GetProfileType() != cloudprofilerpb.ProfileType_PROFILE_TYPE_UNSPECIFIED {
		response.ProfileType = req.GetProfile().GetProfileType()
	}
	response.ProfileBytes = req.GetProfile().GetProfileBytes()
	if req.GetProfile().GetDeployment() != nil {
		response.Deployment = req.GetProfile().GetDeployment()
	}
	if len(req.GetProfile().GetLabels()) > 0 {
		response.Labels = req.GetProfile().GetLabels()
	}
	return grpcProtoSuccess(response)
}

func gcpStage4GRPCCloudProfilerUpdateProfile(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudprofilerpb.UpdateProfileRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetProfile() == nil {
		return grpcInvalidArgument("profile-required")
	}
	profileID, project, ok := parseGCPStage4CloudProfilerProfileName(req.GetProfile().GetName())
	if !ok {
		return grpcInvalidArgument("profile_name-invalid")
	}
	response := gcpStage4CloudProfilerProfile(project, profileID)
	if len(req.GetProfile().GetLabels()) > 0 {
		response.Labels = req.GetProfile().GetLabels()
	}
	return grpcProtoSuccess(response)
}

func gcpStage4CloudProfilerProfile(project, profileID string) *cloudprofilerpb.Profile {
	return &cloudprofilerpb.Profile{
		Name:        fmt.Sprintf("projects/%s/profiles/%s", project, profileID),
		ProfileType: cloudprofilerpb.ProfileType_CPU,
		Deployment: &cloudprofilerpb.Deployment{
			ProjectId: project,
			Target:    "stackyard-service",
			Labels: map[string]string{
				"language": "go",
				"region":   "us-central1",
			},
		},
		Duration: durationpb.New(60 * time.Second),
		Labels: map[string]string{
			"source": "stackyard",
		},
		StartTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func parseGCPStage4CloudProfilerProfileName(name string) (profileID, project string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "profiles" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	profileID = strings.TrimSpace(parts[3])
	if project == "" || profileID == "" {
		return "", "", false
	}
	return profileID, project, true
}

func gcpStage4GRPCCloudQuotasListQuotaInfos(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudquotaspb.ListQuotaInfosRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, service, ok := parseGCPStage4QuotaInfosParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*cloudquotaspb.QuotaInfo{
		gcpStage4QuotaInfo(scope, scopeID, location, service, "CpusPerProjectPerRegion"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&cloudquotaspb.ListQuotaInfosResponse{
		QuotaInfos:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCCloudQuotasGetQuotaInfo(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudquotaspb.GetQuotaInfoRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, service, quotaID, ok := parseGCPStage4QuotaInfoName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4QuotaInfo(scope, scopeID, location, service, quotaID))
}

func gcpStage4GRPCCloudQuotasListQuotaPreferences(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudquotaspb.ListQuotaPreferencesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPStage4QuotaPreferencesParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*cloudquotaspb.QuotaPreference{
		gcpStage4QuotaPreference(scope, scopeID, location, "team-config", "compute.googleapis.com", "CpusPerProjectPerRegion", 16),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&cloudquotaspb.ListQuotaPreferencesResponse{
		QuotaPreferences: items[start:end],
		NextPageToken:    next,
	})
}

func gcpStage4GRPCCloudQuotasGetQuotaPreference(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudquotaspb.GetQuotaPreferenceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, preferenceID, ok := parseGCPStage4QuotaPreferenceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4QuotaPreference(scope, scopeID, location, preferenceID, "compute.googleapis.com", "CpusPerProjectPerRegion", 16))
}

func gcpStage4GRPCCloudQuotasCreateQuotaPreference(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudquotaspb.CreateQuotaPreferenceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPStage4QuotaPreferencesParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	preference := req.GetQuotaPreference()
	if preference == nil {
		return grpcInvalidArgument("quota_preference-required")
	}
	if strings.TrimSpace(preference.GetService()) == "" || strings.TrimSpace(preference.GetQuotaId()) == "" {
		return grpcInvalidArgument("service-and-quota_id-required")
	}
	if preference.GetQuotaConfig() == nil || preference.GetQuotaConfig().GetPreferredValue() <= 0 {
		return grpcInvalidArgument("preferred_value-invalid")
	}
	preferenceID := strings.TrimSpace(req.GetQuotaPreferenceId())
	if preferenceID == "" {
		preferenceID = "team-config"
	}
	return grpcProtoSuccess(gcpStage4QuotaPreference(scope, scopeID, location, preferenceID, preference.GetService(), preference.GetQuotaId(), preference.GetQuotaConfig().GetPreferredValue()))
}

func gcpStage4GRPCCloudQuotasUpdateQuotaPreference(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &cloudquotaspb.UpdateQuotaPreferenceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	preference := req.GetQuotaPreference()
	if preference == nil {
		return grpcInvalidArgument("quota_preference-required")
	}
	scope, scopeID, location, preferenceID, ok := parseGCPStage4QuotaPreferenceName(preference.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(preference.GetService()) == "" || strings.TrimSpace(preference.GetQuotaId()) == "" {
		return grpcInvalidArgument("service-and-quota_id-required")
	}
	if preference.GetQuotaConfig() == nil || preference.GetQuotaConfig().GetPreferredValue() <= 0 {
		return grpcInvalidArgument("preferred_value-invalid")
	}
	return grpcProtoSuccess(gcpStage4QuotaPreference(scope, scopeID, location, preferenceID, preference.GetService(), preference.GetQuotaId(), preference.GetQuotaConfig().GetPreferredValue()))
}

func gcpStage4QuotaInfo(scope, scopeID, location, service, quotaID string) *cloudquotaspb.QuotaInfo {
	return &cloudquotaspb.QuotaInfo{
		Name:                   fmt.Sprintf("%s/%s/locations/%s/services/%s/quotaInfos/%s", scope, scopeID, location, service, quotaID),
		QuotaId:                quotaID,
		Metric:                 "compute.googleapis.com/cpus",
		Service:                service,
		IsPrecise:              true,
		RefreshInterval:        "60s",
		ContainerType:          cloudquotaspb.QuotaInfo_PROJECT,
		MetricDisplayName:      "CPU quota",
		QuotaDisplayName:       quotaID,
		MetricUnit:             "1",
		IsFixed:                false,
		IsConcurrent:           false,
		ServiceRequestQuotaUri: "https://console.cloud.google.com/iam-admin/quotas",
	}
}

func gcpStage4QuotaPreference(scope, scopeID, location, preferenceID, service, quotaID string, preferredValue int64) *cloudquotaspb.QuotaPreference {
	return &cloudquotaspb.QuotaPreference{
		Name:    fmt.Sprintf("%s/%s/locations/%s/quotaPreferences/%s", scope, scopeID, location, preferenceID),
		Service: service,
		QuotaId: quotaID,
		QuotaConfig: &cloudquotaspb.QuotaConfig{
			PreferredValue: preferredValue,
		},
		Dimensions: map[string]string{
			"region": "us-central1",
		},
		Justification: "stackyard staged quota request",
		ContactEmail:  "stackyard@example.com",
	}
}

func gcpStage4GRPCProcurementListOrders(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &procurementpb.ListOrdersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	billingAccount, ok := parseGCPStage4ProcurementParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*procurementpb.Order{
		gcpStage4ProcurementOrder(billingAccount, "order-1"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&procurementpb.ListOrdersResponse{
		Orders:        items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCProcurementGetOrder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &procurementpb.GetOrderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	billingAccount, orderID, ok := parseGCPStage4ProcurementOrderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ProcurementOrder(billingAccount, orderID))
}

func gcpStage4GRPCProcurementPlaceOrder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &procurementpb.PlaceOrderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	billingAccount, ok := parseGCPStage4ProcurementParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetDisplayName()) == "" {
		return grpcInvalidArgument("display_name-required")
	}
	return grpcProtoSuccess(gcpStage4ProcurementOperation(billingAccount, "order-1", "placeOrder"))
}

func gcpStage4GRPCProcurementModifyOrder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &procurementpb.ModifyOrderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	billingAccount, orderID, ok := parseGCPStage4ProcurementOrderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(req.GetDisplayName()) == "" {
		return grpcInvalidArgument("display_name-required")
	}
	if len(req.GetModifications()) == 0 {
		return grpcInvalidArgument("modifications-required")
	}
	return grpcProtoSuccess(gcpStage4ProcurementOperation(billingAccount, orderID, "modifyOrder"))
}

func gcpStage4GRPCProcurementCancelOrder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &procurementpb.CancelOrderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	billingAccount, orderID, ok := parseGCPStage4ProcurementOrderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetCancellationPolicy() == procurementpb.CancelOrderRequest_CANCELLATION_POLICY_UNSPECIFIED {
		return grpcInvalidArgument("cancellation_policy-required")
	}
	return grpcProtoSuccess(gcpStage4ProcurementOperation(billingAccount, orderID, "cancelOrder"))
}

func gcpStage4ProcurementOrder(billingAccount, orderID string) *procurementpb.Order {
	return &procurementpb.Order{
		Name:        fmt.Sprintf("billingAccounts/%s/orders/%s", billingAccount, orderID),
		DisplayName: "Team commitment order",
		Etag:        "stackyard-order-etag",
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ProcurementOperation(billingAccount, orderID, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("billingAccounts/%s/orders/%s/operations/%s", billingAccount, orderID, operationID),
		Done: false,
	}
}

func gcpStage4GRPCConfigListDeployments(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ListDeploymentsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*configpb.Deployment{}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configpb.ListDeploymentsResponse{
		Deployments:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCConfigGetDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.GetDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, deploymentID, ok := parseGCPStage4ConfigDeploymentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeployment(project, location, deploymentID))
}

func gcpStage4GRPCConfigCreateDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.CreateDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetDeploymentId()) == "" {
		return grpcInvalidArgument("deployment_id-required")
	}
	if req.GetDeployment() == nil {
		return grpcInvalidArgument("deployment-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigOperation(project, location, "create-deployment"))
}

func gcpStage4GRPCConfigDeleteDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.DeleteDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPStage4ConfigDeploymentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigOperation(project, location, "delete-deployment"))
}

func gcpStage4GRPCConfigLockDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.LockDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPStage4ConfigDeploymentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigOperation(project, location, "lock-deployment"))
}

func gcpStage4GRPCConfigUnlockDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.UnlockDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPStage4ConfigDeploymentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetLockId() <= 0 {
		return grpcInvalidArgument("lock_id-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigOperation(project, location, "unlock-deployment"))
}

func gcpStage4GRPCConfigExportLockInfo(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ExportLockInfoRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4ConfigDeploymentName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&configpb.LockInfo{})
}

func gcpStage4GRPCConfigCreatePreview(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.CreatePreviewRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPreview() == nil {
		return grpcInvalidArgument("preview-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigOperation(project, location, "create-preview"))
}

func gcpStage4GRPCConfigGetPreview(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.GetPreviewRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, previewID, ok := parseGCPStage4ConfigPreviewName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigPreview(project, location, previewID))
}

func gcpStage4GRPCConfigListPreviews(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ListPreviewsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*configpb.Preview{}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configpb.ListPreviewsResponse{
		Previews:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCConfigDeletePreview(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.DeletePreviewRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPStage4ConfigPreviewName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigOperation(project, location, "delete-preview"))
}

func gcpStage4GRPCConfigExportPreviewResult(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ExportPreviewResultRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4ConfigPreviewName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	return grpcProtoSuccess(&configpb.ExportPreviewResultResponse{})
}

func gcpStage4GRPCConfigListTerraformVersions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ListTerraformVersionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*configpb.TerraformVersion{}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configpb.ListTerraformVersionsResponse{
		TerraformVersions: items[start:end],
		NextPageToken:     next,
	})
}

func gcpStage4GRPCConfigGetTerraformVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.GetTerraformVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, versionID, ok := parseGCPStage4ConfigTerraformVersionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigTerraformVersion(project, location, versionID))
}

func gcpStage4GRPCConfigListResourceChanges(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ListResourceChangesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4ConfigPreviewName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*configpb.ResourceChange{}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configpb.ListResourceChangesResponse{
		ResourceChanges: items[start:end],
		NextPageToken:   next,
	})
}

func gcpStage4GRPCConfigGetResourceChange(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.GetResourceChangeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, previewID, resourceChangeID, ok := parseGCPStage4ConfigResourceChangeName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigResourceChange(project, location, previewID, resourceChangeID))
}

func gcpStage4GRPCConfigListResourceDrifts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.ListResourceDriftsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4ConfigPreviewName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*configpb.ResourceDrift{}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configpb.ListResourceDriftsResponse{
		ResourceDrifts: items[start:end],
		NextPageToken:  next,
	})
}

func gcpStage4GRPCConfigGetResourceDrift(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configpb.GetResourceDriftRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, previewID, resourceDriftID, ok := parseGCPStage4ConfigResourceDriftName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigResourceDrift(project, location, previewID, resourceDriftID))
}

func gcpStage4ConfigDeployment(project, location, deploymentID string) *configpb.Deployment {
	return &configpb.Deployment{
		Name:           fmt.Sprintf("projects/%s/locations/%s/deployments/%s", project, location, deploymentID),
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime),
		State:          configpb.Deployment_ACTIVE,
		LatestRevision: fmt.Sprintf("projects/%s/locations/%s/deployments/%s/revisions/r-1", project, location, deploymentID),
	}
}

func gcpStage4ConfigPreview(project, location, previewID string) *configpb.Preview {
	return &configpb.Preview{
		Name:           fmt.Sprintf("projects/%s/locations/%s/previews/%s", project, location, previewID),
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime),
		State:          configpb.Preview_SUCCEEDED,
		ServiceAccount: fmt.Sprintf("projects/%s/serviceAccounts/stackyard@%s.iam.gserviceaccount.com", project, project),
	}
}

func gcpStage4ConfigTerraformVersion(project, location, terraformVersionID string) *configpb.TerraformVersion {
	return &configpb.TerraformVersion{
		Name:        fmt.Sprintf("projects/%s/locations/%s/terraformVersions/%s", project, location, terraformVersionID),
		State:       configpb.TerraformVersion_ACTIVE,
		SupportTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ConfigResourceChange(project, location, previewID, resourceChangeID string) *configpb.ResourceChange {
	return &configpb.ResourceChange{
		Name:   fmt.Sprintf("projects/%s/locations/%s/previews/%s/resourceChanges/%s", project, location, previewID, resourceChangeID),
		Intent: configpb.ResourceChange_CREATE,
	}
}

func gcpStage4ConfigResourceDrift(project, location, previewID, resourceDriftID string) *configpb.ResourceDrift {
	return &configpb.ResourceDrift{
		Name: fmt.Sprintf("projects/%s/locations/%s/previews/%s/resourceDrifts/%s", project, location, previewID, resourceDriftID),
	}
}

func gcpStage4ConfigOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s-op-1", project, location, operationID),
		Done: true,
	}
}

func gcpStage4GRPCConfigDeliveryListResourceBundles(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.ListResourceBundlesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid || req.GetPageSize() < 0 {
		return grpcInvalidArgument("pagination-invalid")
	}
	items := []*configdeliverypb.ResourceBundle{
		gcpStage4ConfigDeliveryResourceBundle(project, location, "platform-bundle"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configdeliverypb.ListResourceBundlesResponse{
		ResourceBundles: items[start:end],
		NextPageToken:   next,
	})
}

func gcpStage4GRPCConfigDeliveryGetResourceBundle(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.GetResourceBundleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, ok := parseGCPStage4ConfigDeliveryResourceBundleName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryResourceBundle(project, location, resourceBundleID))
}

func gcpStage4GRPCConfigDeliveryCreateResourceBundle(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.CreateResourceBundleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetResourceBundleId()) == "" {
		return grpcInvalidArgument("resource_bundle_id-required")
	}
	if req.GetResourceBundle() == nil {
		return grpcInvalidArgument("resource_bundle-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "createResourceBundle."+req.GetResourceBundleId()))
}

func gcpStage4GRPCConfigDeliveryDeleteResourceBundle(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.DeleteResourceBundleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, ok := parseGCPStage4ConfigDeliveryResourceBundleName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "deleteResourceBundle."+resourceBundleID))
}

func gcpStage4GRPCConfigDeliveryListFleetPackages(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.ListFleetPackagesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid || req.GetPageSize() < 0 {
		return grpcInvalidArgument("pagination-invalid")
	}
	items := []*configdeliverypb.FleetPackage{
		gcpStage4ConfigDeliveryFleetPackage(project, location, "platform-package"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configdeliverypb.ListFleetPackagesResponse{
		FleetPackages: items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCConfigDeliveryGetFleetPackage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.GetFleetPackageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, fleetPackageID, ok := parseGCPStage4ConfigDeliveryFleetPackageName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryFleetPackage(project, location, fleetPackageID))
}

func gcpStage4GRPCConfigDeliveryCreateFleetPackage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.CreateFleetPackageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ConfigDeliveryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetFleetPackageId()) == "" {
		return grpcInvalidArgument("fleet_package_id-required")
	}
	if req.GetFleetPackage() == nil {
		return grpcInvalidArgument("fleet_package-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "createFleetPackage."+req.GetFleetPackageId()))
}

func gcpStage4GRPCConfigDeliveryDeleteFleetPackage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.DeleteFleetPackageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, fleetPackageID, ok := parseGCPStage4ConfigDeliveryFleetPackageName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "deleteFleetPackage."+fleetPackageID))
}

func gcpStage4GRPCConfigDeliveryListReleases(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.ListReleasesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, ok := parseGCPStage4ConfigDeliveryResourceBundleName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid || req.GetPageSize() < 0 {
		return grpcInvalidArgument("pagination-invalid")
	}
	items := []*configdeliverypb.Release{
		gcpStage4ConfigDeliveryRelease(project, location, resourceBundleID, "r-1"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configdeliverypb.ListReleasesResponse{
		Releases:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCConfigDeliveryGetRelease(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.GetReleaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, releaseID, ok := parseGCPStage4ConfigDeliveryReleaseName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryRelease(project, location, resourceBundleID, releaseID))
}

func gcpStage4GRPCConfigDeliveryCreateRelease(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.CreateReleaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, ok := parseGCPStage4ConfigDeliveryResourceBundleName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetReleaseId()) == "" {
		return grpcInvalidArgument("release_id-required")
	}
	if req.GetRelease() == nil {
		return grpcInvalidArgument("release-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "createRelease."+resourceBundleID+"."+req.GetReleaseId()))
}

func gcpStage4GRPCConfigDeliveryDeleteRelease(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.DeleteReleaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, releaseID, ok := parseGCPStage4ConfigDeliveryReleaseName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "deleteRelease."+releaseID))
}

func gcpStage4GRPCConfigDeliveryListVariants(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.ListVariantsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, releaseID, ok := parseGCPStage4ConfigDeliveryReleaseName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid || req.GetPageSize() < 0 {
		return grpcInvalidArgument("pagination-invalid")
	}
	items := []*configdeliverypb.Variant{
		gcpStage4ConfigDeliveryVariant(project, location, resourceBundleID, releaseID, "default"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configdeliverypb.ListVariantsResponse{
		Variants:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCConfigDeliveryGetVariant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.GetVariantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, resourceBundleID, releaseID, variantID, ok := parseGCPStage4ConfigDeliveryVariantName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryVariant(project, location, resourceBundleID, releaseID, variantID))
}

func gcpStage4GRPCConfigDeliveryCreateVariant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.CreateVariantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, _, ok := parseGCPStage4ConfigDeliveryReleaseName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetVariantId()) == "" {
		return grpcInvalidArgument("variant_id-required")
	}
	if req.GetVariant() == nil {
		return grpcInvalidArgument("variant-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "createVariant."+req.GetVariantId()))
}

func gcpStage4GRPCConfigDeliveryDeleteVariant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.DeleteVariantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, _, variantID, ok := parseGCPStage4ConfigDeliveryVariantName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, "deleteVariant."+variantID))
}

func gcpStage4GRPCConfigDeliveryListRollouts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.ListRolloutsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, fleetPackageID, ok := parseGCPStage4ConfigDeliveryFleetPackageName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid || req.GetPageSize() < 0 {
		return grpcInvalidArgument("pagination-invalid")
	}
	items := []*configdeliverypb.Rollout{
		gcpStage4ConfigDeliveryRollout(project, location, fleetPackageID, "rollout-1"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&configdeliverypb.ListRolloutsResponse{
		Rollouts:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCConfigDeliveryGetRollout(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &configdeliverypb.GetRolloutRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, fleetPackageID, rolloutID, ok := parseGCPStage4ConfigDeliveryRolloutName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryRollout(project, location, fleetPackageID, rolloutID))
}

func gcpStage4GRPCConfigDeliveryRolloutAction(grpcReqBody []byte, action string) ([]byte, string, string, bool) {
	var name string
	var reason string

	switch action {
	case "suspend":
		req := &configdeliverypb.SuspendRolloutRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name = req.GetName()
		reason = req.GetReason()
	case "resume":
		req := &configdeliverypb.ResumeRolloutRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name = req.GetName()
		reason = req.GetReason()
	case "abort":
		req := &configdeliverypb.AbortRolloutRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name = req.GetName()
		reason = req.GetReason()
	default:
		return nil, "", "", false
	}

	project, location, _, rolloutID, ok := parseGCPStage4ConfigDeliveryRolloutName(name)
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(reason) == "" {
		return grpcInvalidArgument("reason-required")
	}
	return grpcProtoSuccess(gcpStage4ConfigDeliveryOperation(project, location, action+"Rollout."+rolloutID))
}

func gcpStage4ConfigDeliveryResourceBundle(project, location, resourceBundleID string) *configdeliverypb.ResourceBundle {
	return &configdeliverypb.ResourceBundle{
		Name:        fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s", project, location, resourceBundleID),
		Description: "Stackyard sample resource bundle",
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ConfigDeliveryFleetPackage(project, location, fleetPackageID string) *configdeliverypb.FleetPackage {
	return &configdeliverypb.FleetPackage{
		Name:       fmt.Sprintf("projects/%s/locations/%s/fleetPackages/%s", project, location, fleetPackageID),
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ConfigDeliveryRelease(project, location, resourceBundleID, releaseID string) *configdeliverypb.Release {
	return &configdeliverypb.Release{
		Name:       fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s/releases/%s", project, location, resourceBundleID, releaseID),
		Version:    "v1.0.0",
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ConfigDeliveryVariant(project, location, resourceBundleID, releaseID, variantID string) *configdeliverypb.Variant {
	return &configdeliverypb.Variant{
		Name: fmt.Sprintf("projects/%s/locations/%s/resourceBundles/%s/releases/%s/variants/%s", project, location, resourceBundleID, releaseID, variantID),
		Resources: []string{
			"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sample\n",
		},
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ConfigDeliveryRollout(project, location, fleetPackageID, rolloutID string) *configdeliverypb.Rollout {
	return &configdeliverypb.Rollout{
		Name:       fmt.Sprintf("projects/%s/locations/%s/fleetPackages/%s/rollouts/%s", project, location, fleetPackageID, rolloutID),
		Release:    fmt.Sprintf("projects/%s/locations/%s/resourceBundles/platform-bundle/releases/r-1", project, location),
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ConfigDeliveryOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}

func gcpStage4GRPCRapidMigrationAssessmentListCollectors(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.ListCollectorsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RapidMigrationAssessmentParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*rapidmigrationassessmentpb.Collector{
		gcpStage4RapidMigrationAssessmentCollector(project, location, "collector-1", rapidmigrationassessmentpb.Collector_STATE_ACTIVE),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&rapidmigrationassessmentpb.ListCollectorsResponse{
		Collectors:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRapidMigrationAssessmentGetCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.GetCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectorID, ok := parseGCPStage4RapidMigrationAssessmentCollectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := rapidmigrationassessmentpb.Collector_STATE_ACTIVE
	if strings.Contains(collectorID, "paused") {
		state = rapidmigrationassessmentpb.Collector_STATE_PAUSED
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentCollector(project, location, collectorID, state))
}

func gcpStage4GRPCRapidMigrationAssessmentCreateCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.CreateCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RapidMigrationAssessmentParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	collectorID := strings.TrimSpace(req.GetCollectorId())
	if collectorID == "" {
		return grpcInvalidArgument("collector_id-required")
	}
	collector := req.GetCollector()
	if collector == nil {
		return grpcInvalidArgument("collector-required")
	}
	if strings.TrimSpace(collector.GetDisplayName()) == "" {
		return grpcInvalidArgument("collector_display_name-required")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/collectors/%s", project, location, collectorID)
	if strings.TrimSpace(collector.GetName()) != "" && strings.TrimSpace(collector.GetName()) != expectedName {
		return grpcInvalidArgument("collector_name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "createCollector."+collectorID))
}

func gcpStage4GRPCRapidMigrationAssessmentUpdateCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.UpdateCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	collector := req.GetCollector()
	if collector == nil {
		return grpcInvalidArgument("collector-required")
	}
	project, location, collectorID, ok := parseGCPStage4RapidMigrationAssessmentCollectorName(collector.GetName())
	if !ok {
		return grpcInvalidArgument("collector_name-required")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "updateCollector."+collectorID))
}

func gcpStage4GRPCRapidMigrationAssessmentDeleteCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.DeleteCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectorID, ok := parseGCPStage4RapidMigrationAssessmentCollectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "deleteCollector."+collectorID))
}

func gcpStage4GRPCRapidMigrationAssessmentPauseCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.PauseCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectorID, ok := parseGCPStage4RapidMigrationAssessmentCollectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(collectorID, "paused") {
		return grpcFailedPrecondition("collector-already-paused")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "pauseCollector."+collectorID))
}

func gcpStage4GRPCRapidMigrationAssessmentResumeCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.ResumeCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectorID, ok := parseGCPStage4RapidMigrationAssessmentCollectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(collectorID, "active") {
		return grpcFailedPrecondition("collector-already-active")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "resumeCollector."+collectorID))
}

func gcpStage4GRPCRapidMigrationAssessmentRegisterCollector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.RegisterCollectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectorID, ok := parseGCPStage4RapidMigrationAssessmentCollectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(collectorID, "registered") {
		return grpcFailedPrecondition("collector-already-registered")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "registerCollector."+collectorID))
}

func gcpStage4GRPCRapidMigrationAssessmentCreateAnnotation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.CreateAnnotationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RapidMigrationAssessmentParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	annotation := req.GetAnnotation()
	if annotation == nil {
		return grpcInvalidArgument("annotation-required")
	}
	if annotation.GetType() == rapidmigrationassessmentpb.Annotation_TYPE_UNSPECIFIED {
		return grpcInvalidArgument("annotation_type-required")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentOperation(project, location, "createAnnotation.annotation-1"))
}

func gcpStage4GRPCRapidMigrationAssessmentGetAnnotation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &rapidmigrationassessmentpb.GetAnnotationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, annotationID, ok := parseGCPStage4RapidMigrationAssessmentAnnotationName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RapidMigrationAssessmentAnnotation(project, location, annotationID, rapidmigrationassessmentpb.Annotation_TYPE_LEGACY_EXPORT_CONSENT))
}

func gcpStage4RapidMigrationAssessmentCollector(project, location, collectorID string, state rapidmigrationassessmentpb.Collector_State) *rapidmigrationassessmentpb.Collector {
	return &rapidmigrationassessmentpb.Collector{
		Name:               fmt.Sprintf("projects/%s/locations/%s/collectors/%s", project, location, collectorID),
		DisplayName:        "Stackyard Collector " + collectorID,
		Description:        "Stackyard staged Rapid Migration Assessment collector",
		ServiceAccount:     fmt.Sprintf("collector-%s@%s.iam.gserviceaccount.com", collectorID, project),
		ExpectedAssetCount: 42,
		State:              state,
		CollectionDays:     7,
		EulaUri:            "https://example.com/stackyard/eula",
		Labels: map[string]string{
			"env": "staged",
		},
		Bucket:        "stackyard-rma-bucket",
		ClientVersion: "1.0.0",
		CreateTime:    timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:    timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4RapidMigrationAssessmentAnnotation(project, location, annotationID string, annotationType rapidmigrationassessmentpb.Annotation_Type) *rapidmigrationassessmentpb.Annotation {
	return &rapidmigrationassessmentpb.Annotation{
		Name: fmt.Sprintf("projects/%s/locations/%s/annotations/%s", project, location, annotationID),
		Type: annotationType,
		Labels: map[string]string{
			"source": "stackyard",
		},
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4RapidMigrationAssessmentOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}

func gcpStage4GRPCVMMigration(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVMMigrationListSourcesMethod:
		req := &vmmigrationpb.ListSourcesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStage4VMMigrationParent(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*vmmigrationpb.Source{
			gcpStage4VMMigrationSource(project, location, "source-1"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&vmmigrationpb.ListSourcesResponse{
			Sources:       items[start:end],
			NextPageToken: next,
			Unreachable:   []string{},
		})
	case gcpVMMigrationGetSourceMethod:
		req := &vmmigrationpb.GetSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, sourceID, ok := parseGCPStage4VMMigrationSourceName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPVMMigrationMissingID(sourceID) {
			return grpcNotFound("source-not-found")
		}
		return grpcProtoSuccess(gcpStage4VMMigrationSource(project, location, sourceID))
	case gcpVMMigrationCreateSourceMethod:
		req := &vmmigrationpb.CreateSourceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStage4VMMigrationParent(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		sourceID := strings.TrimSpace(req.GetSourceId())
		if sourceID == "" {
			return grpcInvalidArgument("source_id-required")
		}
		if req.GetSource() == nil {
			return grpcInvalidArgument("source-required")
		}
		if strings.Contains(strings.ToLower(sourceID), "existing") {
			return grpcAlreadyExists("source-already-exists")
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/sources/%s", project, location, sourceID)
		if sourceName := strings.TrimSpace(req.GetSource().GetName()); sourceName != "" && sourceName != expectedName {
			return grpcInvalidArgument("source_name-mismatch")
		}
		return grpcProtoSuccess(gcpStage4VMMigrationOperation(project, location, "createSource."+sourceID))
	case gcpVMMigrationPauseMigrationMethod:
		req := &vmmigrationpb.PauseMigrationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, migratingVMID, ok := parseGCPStage4VMMigrationMigratingVMName(req.GetMigratingVm())
		if !ok {
			return grpcInvalidArgument("migrating_vm-required")
		}
		if strings.Contains(strings.ToLower(migratingVMID), "paused") {
			return grpcFailedPrecondition("migration-already-paused")
		}
		return grpcProtoSuccess(gcpStage4VMMigrationOperation(project, location, "pauseMigration."+migratingVMID))
	default:
		return gcpStage4GRPCVMMigrationDynamic(path)
	}
}

func gcpStage4VMMigrationSource(project, location, sourceID string) *vmmigrationpb.Source {
	return &vmmigrationpb.Source{
		Name:        fmt.Sprintf("projects/%s/locations/%s/sources/%s", project, location, sourceID),
		Description: "Stackyard VM Migration source fixture",
		Labels: map[string]string{
			"env": "staged",
		},
		SourceDetails: &vmmigrationpb.Source_Vmware{
			Vmware: &vmmigrationpb.VmwareSourceDetails{},
		},
	}
}

func gcpStage4VMMigrationOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}

func parseGCPStage4VMMigrationParent(parent string) (project, location string, ok bool) {
	project, location, tail, parsed := parseGCPVMMigrationResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 0 {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4VMMigrationSourceName(name string) (project, location, sourceID string, ok bool) {
	project, location, tail, parsed := parseGCPVMMigrationResourceName(strings.TrimSpace(name))
	if !parsed || len(tail) != 2 || tail[0] != "sources" {
		return "", "", "", false
	}
	sourceID = strings.TrimSpace(tail[1])
	if sourceID == "" {
		return "", "", "", false
	}
	return project, location, sourceID, true
}

func parseGCPStage4VMMigrationMigratingVMName(name string) (project, location, migratingVMID string, ok bool) {
	project, location, tail, parsed := parseGCPVMMigrationResourceName(strings.TrimSpace(name))
	if !parsed || len(tail) != 4 || tail[0] != "sources" || tail[2] != "migratingVms" {
		return "", "", "", false
	}
	migratingVMID = strings.TrimSpace(tail[3])
	if migratingVMID == "" {
		return "", "", "", false
	}
	return project, location, migratingVMID, true
}

func gcpStage4GRPCVMMigrationDynamic(path string) ([]byte, string, string, bool) {
	serviceName, methodName, ok := parseGCPStage4GRPCServiceAndMethod(path)
	if !ok {
		return nil, "", "", false
	}
	serviceDesc, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
	if err != nil {
		return nil, "", "", false
	}
	service, ok := serviceDesc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, "", "", false
	}
	method := service.Methods().ByName(protoreflect.Name(methodName))
	if method == nil {
		return nil, "", "", false
	}
	msg := dynamicpb.NewMessage(method.Output())
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func gcpStage4GRPCLongrunningGetOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.GetOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&longrunningpb.Operation{
		Name: req.GetName(),
		Done: true,
	})
}

func gcpStage4GRPCResourceManagerListFolders(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.ListFoldersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPResourceManagerParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerpb.Folder{
		gcpStage4ResourceManagerFolder("1001", parent, "Team Folder", resourcemanagerpb.Folder_ACTIVE),
		gcpStage4ResourceManagerFolder("1002", parent, "Archive Folder", resourcemanagerpb.Folder_DELETE_REQUESTED),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerpb.ListFoldersResponse{
		Folders:       items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCResourceManagerSearchFolders(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.SearchFoldersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return grpcInvalidArgument("query-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*resourcemanagerpb.Folder{
		gcpStage4ResourceManagerFolder("1001", "organizations/123456", "Team Folder", resourcemanagerpb.Folder_ACTIVE),
		gcpStage4ResourceManagerFolder("1002", "folders/1001", "Archive Folder", resourcemanagerpb.Folder_DELETE_REQUESTED),
	}
	lowerQuery := strings.ToLower(query)
	switch {
	case strings.Contains(lowerQuery, "lifecyclestate=active"):
		items = items[:1]
	case strings.Contains(lowerQuery, "lifecyclestate=delete_requested"):
		items = items[1:]
	case strings.Contains(lowerQuery, "parent=folders/1001"):
		items = items[1:]
	case strings.Contains(lowerQuery, "parent=organizations/123456"):
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&resourcemanagerpb.SearchFoldersResponse{
		Folders:       items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCResourceManagerGetFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.GetFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := resourcemanagerpb.Folder_ACTIVE
	if strings.Contains(strings.ToLower(folderID), "deleted") {
		state = resourcemanagerpb.Folder_DELETE_REQUESTED
	}
	parent := "organizations/123456"
	if strings.Contains(strings.ToLower(folderID), "child") {
		parent = "folders/1001"
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerFolder(folderID, parent, "Folder "+folderID, state))
}

func gcpStage4GRPCResourceManagerCreateFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.CreateFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPResourceManagerParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	folder := req.GetFolder()
	if folder == nil {
		return grpcInvalidArgument("folder-required")
	}
	displayName := strings.TrimSpace(folder.GetDisplayName())
	if !isGCPResourceManagerDisplayName(displayName) {
		return grpcInvalidArgument("folder.display_name-invalid")
	}
	folderID := "1001"
	if strings.Contains(strings.ToLower(displayName), "archive") {
		folderID = "1002"
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerOperation("create-folder-"+folderID, false))
}

func gcpStage4GRPCResourceManagerUpdateFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.UpdateFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folder := req.GetFolder()
	if folder == nil {
		return grpcInvalidArgument("folder-required")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(folder.GetName())
	if !ok {
		return grpcInvalidArgument("folder.name-required")
	}
	if !isGCPResourceManagerDisplayName(strings.TrimSpace(folder.GetDisplayName())) {
		return grpcInvalidArgument("folder.display_name-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range req.GetUpdateMask().GetPaths() {
		switch strings.TrimSpace(path) {
		case "display_name", "displayName":
		default:
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerFolder(folderID, "organizations/123456", strings.TrimSpace(folder.GetDisplayName()), resourcemanagerpb.Folder_ACTIVE))
}

func gcpStage4GRPCResourceManagerMoveFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.MoveFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !isGCPResourceManagerParent(strings.TrimSpace(req.GetDestinationParent())) {
		return grpcInvalidArgument("destination_parent-required")
	}
	if strings.TrimSpace(req.GetDestinationParent()) == "folders/"+folderID {
		return grpcFailedPrecondition("destination_parent-cannot-equal-folder")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerOperation("move-folder-"+folderID, false))
}

func gcpStage4GRPCResourceManagerDeleteFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.DeleteFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerFolder(folderID, "organizations/123456", "Folder "+folderID, resourcemanagerpb.Folder_DELETE_REQUESTED))
}

func gcpStage4GRPCResourceManagerUndeleteFolder(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &resourcemanagerpb.UndeleteFolderRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(folderID), "active") {
		return grpcFailedPrecondition("folder-not-delete-requested")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerFolder(folderID, "organizations/123456", "Folder "+folderID, resourcemanagerpb.Folder_ACTIVE))
}

func gcpStage4GRPCResourceManagerGetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.GetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(req.GetResource())
	if !ok {
		return grpcInvalidArgument("resource-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerPolicy(folderID, nil))
}

func gcpStage4GRPCResourceManagerSetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.SetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	folderID, ok := parseGCPStage4ResourceManagerFolderName(req.GetResource())
	if !ok {
		return grpcInvalidArgument("resource-required")
	}
	if req.GetPolicy() == nil {
		return grpcInvalidArgument("policy-required")
	}
	return grpcProtoSuccess(gcpStage4ResourceManagerPolicy(folderID, req.GetPolicy()))
}

func gcpStage4GRPCResourceManagerTestIAMPermissions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.TestIamPermissionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPStage4ResourceManagerFolderName(req.GetResource()); !ok {
		return grpcInvalidArgument("resource-required")
	}
	if len(req.GetPermissions()) == 0 {
		return grpcInvalidArgument("permissions-required")
	}
	return grpcProtoSuccess(&iampb.TestIamPermissionsResponse{Permissions: req.GetPermissions()})
}

func gcpStage4ResourceManagerFolder(folderID, parent, displayName string, state resourcemanagerpb.Folder_LifecycleState) *resourcemanagerpb.Folder {
	return &resourcemanagerpb.Folder{
		Name:           "folders/" + folderID,
		Parent:         parent,
		DisplayName:    displayName,
		LifecycleState: state,
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4ResourceManagerOperation(operationID string, done bool) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: "operations/" + operationID,
		Done: done,
	}
}

func gcpStage4ResourceManagerPolicy(folderID string, policy *iampb.Policy) *iampb.Policy {
	if policy == nil {
		return &iampb.Policy{
			Version: 1,
			Etag:    []byte("resourcemanager-etag"),
			Bindings: []*iampb.Binding{
				{
					Role:    "roles/resourcemanager.folderViewer",
					Members: []string{"user:alice@example.com"},
				},
			},
		}
	}
	cloned, ok := proto.Clone(policy).(*iampb.Policy)
	if !ok || cloned == nil {
		return policy
	}
	if len(cloned.GetEtag()) == 0 {
		cloned.Etag = []byte("resourcemanager-etag")
	}
	if cloned.GetVersion() == 0 {
		cloned.Version = 1
	}
	_ = folderID
	return cloned
}

func parseGCPStage4ResourceManagerFolderName(name string) (folderID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 2 || parts[0] != "folders" {
		return "", false
	}
	folderID = strings.TrimSpace(parts[1])
	if folderID == "" {
		return "", false
	}
	return folderID, true
}

func gcpStage4GRPCRedisListInstances(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.ListInstancesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RedisLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*redispb.Instance{
		gcpStage4RedisInstance(project, location, "redis-1"),
		gcpStage4RedisInstance(project, location, "redis-2"),
	}
	if location == "-" {
		items = []*redispb.Instance{
			gcpStage4RedisInstance(project, "us-central1", "redis-1"),
			gcpStage4RedisInstance(project, "us-east1", "redis-2"),
		}
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&redispb.ListInstancesResponse{
		Instances:     items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRedisGetInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.GetInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisInstance(project, location, instanceID))
}

func gcpStage4GRPCRedisGetInstanceAuthString(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.GetInstanceAuthStringRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&redispb.InstanceAuthString{AuthString: "stackyard-auth-" + instanceID})
}

func gcpStage4GRPCRedisCreateInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.CreateInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RedisLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	instanceID := strings.TrimSpace(req.GetInstanceId())
	if instanceID == "" {
		return grpcInvalidArgument("instance_id-required")
	}
	if !gcpRedisInstanceIDRegex.MatchString(instanceID) {
		return grpcInvalidArgument("instance_id-invalid")
	}
	instance := req.GetInstance()
	if instance == nil {
		return grpcInvalidArgument("instance-required")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, location, instanceID)
	if strings.TrimSpace(instance.GetName()) != "" && strings.TrimSpace(instance.GetName()) != expectedName {
		return grpcInvalidArgument("instance.name-must-match-parent-and-instance_id")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "createInstance."+instanceID, expectedName))
}

func gcpStage4GRPCRedisUpdateInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.UpdateInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	instance := req.GetInstance()
	if instance == nil {
		return grpcInvalidArgument("instance-required")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(instance.GetName())
	if !ok {
		return grpcInvalidArgument("instance.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4RedisUpdateMaskValid(req.GetUpdateMask().GetPaths()) {
		return grpcInvalidArgument("update_mask-unsupported")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "updateInstance."+instanceID, instance.GetName()))
}

func gcpStage4GRPCRedisUpgradeInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.UpgradeInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(req.GetRedisVersion()) == "" {
		return grpcInvalidArgument("redis_version-required")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "upgradeInstance."+instanceID, req.GetName()))
}

func gcpStage4GRPCRedisImportInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.ImportInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetInputConfig() == nil || req.GetInputConfig().GetGcsSource() == nil || strings.TrimSpace(req.GetInputConfig().GetGcsSource().GetUri()) == "" {
		return grpcInvalidArgument("input_config-required")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "importInstance."+instanceID, req.GetName()))
}

func gcpStage4GRPCRedisExportInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.ExportInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetOutputConfig() == nil || req.GetOutputConfig().GetGcsDestination() == nil || strings.TrimSpace(req.GetOutputConfig().GetGcsDestination().GetUri()) == "" {
		return grpcInvalidArgument("output_config-required")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "exportInstance."+instanceID, req.GetName()))
}

func gcpStage4GRPCRedisFailoverInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.FailoverInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(instanceID, "basic") {
		return grpcFailedPrecondition("instance-tier-basic")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "failoverInstance."+instanceID, req.GetName()))
}

func gcpStage4GRPCRedisDeleteInstance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.DeleteInstanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "deleteInstance."+instanceID, req.GetName()))
}

func gcpStage4GRPCRedisRescheduleMaintenance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &redispb.RescheduleMaintenanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, instanceID, ok := parseGCPStage4RedisInstanceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetRescheduleType() == redispb.RescheduleMaintenanceRequest_RESCHEDULE_TYPE_UNSPECIFIED {
		return grpcInvalidArgument("reschedule_type-required")
	}
	if req.GetRescheduleType() == redispb.RescheduleMaintenanceRequest_SPECIFIC_TIME && req.GetScheduleTime() == nil {
		return grpcInvalidArgument("schedule_time-required")
	}
	if req.GetRescheduleType() == redispb.RescheduleMaintenanceRequest_IMMEDIATE && strings.Contains(instanceID, "locked") {
		return grpcFailedPrecondition("instance-maintenance-locked")
	}
	return grpcProtoSuccess(gcpStage4RedisOperation(project, location, "rescheduleMaintenance."+instanceID, req.GetName()))
}

func gcpStage4RedisInstance(project, location, instanceID string) *redispb.Instance {
	return &redispb.Instance{
		Name:         fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, location, instanceID),
		DisplayName:  "Stackyard Redis " + instanceID,
		Labels:       map[string]string{"env": "test", "service": "redis"},
		Host:         "10.0.0.11",
		Port:         6379,
		State:        redispb.Instance_READY,
		Tier:         redispb.Instance_STANDARD_HA,
		MemorySizeGb: 4,
		RedisVersion: "REDIS_7_0",
		AuthorizedNetwork: fmt.Sprintf(
			"projects/%s/global/networks/default",
			project,
		),
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		MaintenancePolicy: &redispb.MaintenancePolicy{
			Description: "Weekly maintenance window",
		},
		PersistenceConfig: &redispb.PersistenceConfig{
			PersistenceMode:   redispb.PersistenceConfig_RDB,
			RdbSnapshotPeriod: redispb.PersistenceConfig_SIX_HOURS,
		},
	}
}

func gcpStage4RedisOperation(project, location, operationID, target string) *longrunningpb.Operation {
	_ = target
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}

func gcpStage4RedisUpdateMaskValid(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(path)), "_", "")
		switch normalized {
		case "displayname", "labels", "redisconfig", "redisversion", "memorysizegb", "replicacount":
			continue
		default:
			return false
		}
	}
	return true
}

func gcpStage4GRPCRedisClusterListClusters(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.ListClustersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RedisClusterLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*clusterpb.Cluster{
		gcpStage4RedisCluster(project, location, "cluster-1"),
		gcpStage4RedisCluster(project, location, "cluster-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&clusterpb.ListClustersResponse{
		Clusters:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRedisClusterGetCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.GetClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPStage4RedisClusterName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisCluster(project, location, clusterID))
}

func gcpStage4GRPCRedisClusterCreateCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.CreateClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RedisClusterLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	clusterID := strings.TrimSpace(req.GetClusterId())
	if clusterID == "" {
		return grpcInvalidArgument("cluster_id-required")
	}
	if !gcpRedisClusterIDPattern.MatchString(clusterID) {
		return grpcInvalidArgument("cluster_id-invalid")
	}
	cluster := req.GetCluster()
	if cluster == nil {
		return grpcInvalidArgument("cluster-required")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterID)
	if strings.TrimSpace(cluster.GetName()) != "" && strings.TrimSpace(cluster.GetName()) != expectedName {
		return grpcInvalidArgument("cluster.name-must-match-parent-and-cluster_id")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "createCluster."+clusterID, expectedName))
}

func gcpStage4GRPCRedisClusterUpdateCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.UpdateClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	cluster := req.GetCluster()
	if cluster == nil {
		return grpcInvalidArgument("cluster-required")
	}
	project, location, clusterID, ok := parseGCPStage4RedisClusterName(cluster.GetName())
	if !ok {
		return grpcInvalidArgument("cluster.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4RedisClusterUpdateMaskValid(req.GetUpdateMask().GetPaths()) {
		return grpcInvalidArgument("update_mask-unsupported")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "updateCluster."+clusterID, cluster.GetName()))
}

func gcpStage4GRPCRedisClusterDeleteCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.DeleteClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPStage4RedisClusterName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "deleteCluster."+clusterID, req.GetName()))
}

func gcpStage4GRPCRedisClusterGetClusterCertificateAuthority(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.GetClusterCertificateAuthorityRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPStage4RedisClusterCertificateAuthorityName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterCertificateAuthority(project, location, clusterID))
}

func gcpStage4GRPCRedisClusterRescheduleClusterMaintenance(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.RescheduleClusterMaintenanceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPStage4RedisClusterName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetRescheduleType() == clusterpb.RescheduleClusterMaintenanceRequest_RESCHEDULE_TYPE_UNSPECIFIED {
		return grpcInvalidArgument("reschedule_type-required")
	}
	if req.GetRescheduleType() == clusterpb.RescheduleClusterMaintenanceRequest_SPECIFIC_TIME && req.GetScheduleTime() == nil {
		return grpcInvalidArgument("schedule_time-required")
	}
	if req.GetRescheduleType() == clusterpb.RescheduleClusterMaintenanceRequest_IMMEDIATE && strings.Contains(clusterID, "locked") {
		return grpcFailedPrecondition("cluster-maintenance-locked")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "rescheduleClusterMaintenance."+clusterID, req.GetName()))
}

func gcpStage4GRPCRedisClusterListBackupCollections(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.ListBackupCollectionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RedisClusterLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*clusterpb.BackupCollection{
		gcpStage4RedisBackupCollection(project, location, "collection-1", "cluster-1"),
		gcpStage4RedisBackupCollection(project, location, "collection-2", "cluster-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&clusterpb.ListBackupCollectionsResponse{
		BackupCollections: items[start:end],
		NextPageToken:     next,
	})
}

func gcpStage4GRPCRedisClusterGetBackupCollection(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.GetBackupCollectionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectionID, ok := parseGCPStage4RedisBackupCollectionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisBackupCollection(project, location, collectionID, "cluster-1"))
}

func gcpStage4GRPCRedisClusterListBackups(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.ListBackupsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectionID, ok := parseGCPStage4RedisBackupCollectionName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*clusterpb.Backup{
		gcpStage4RedisBackup(project, location, collectionID, "backup-1", "cluster-1"),
		gcpStage4RedisBackup(project, location, collectionID, "backup-2", "cluster-1"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&clusterpb.ListBackupsResponse{
		Backups:       items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRedisClusterGetBackup(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.GetBackupRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, collectionID, backupID, ok := parseGCPStage4RedisBackupName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisBackup(project, location, collectionID, backupID, "cluster-1"))
}

func gcpStage4GRPCRedisClusterDeleteBackup(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.DeleteBackupRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, backupID, ok := parseGCPStage4RedisBackupName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "deleteBackup."+backupID, req.GetName()))
}

func gcpStage4GRPCRedisClusterExportBackup(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.ExportBackupRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, backupID, ok := parseGCPStage4RedisBackupName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(req.GetGcsBucket()) == "" {
		return grpcInvalidArgument("gcs_bucket-required")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "exportBackup."+backupID, req.GetName()))
}

func gcpStage4GRPCRedisClusterBackupCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &clusterpb.BackupClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPStage4RedisClusterName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if backupID := strings.TrimSpace(req.GetBackupId()); backupID != "" && !gcpRedisClusterIDPattern.MatchString(backupID) {
		return grpcInvalidArgument("backup_id-invalid")
	}
	return grpcProtoSuccess(gcpStage4RedisClusterOperation(project, location, "backupCluster."+clusterID, req.GetName()))
}

func gcpStage4GRPCLocationsGetLocation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &locationpb.GetLocationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4LocationName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&locationpb.Location{
		Name:        req.GetName(),
		LocationId:  location,
		DisplayName: "Redis Cluster " + location,
		Labels: map[string]string{
			"service":  "redis_cluster",
			"project":  project,
			"provider": providerGCP,
		},
	})
}

func gcpStage4GRPCLocationsListLocations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &locationpb.ListLocationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPStage4ProjectParent(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*locationpb.Location{
		{
			Name:        fmt.Sprintf("projects/%s/locations/us-central1", project),
			LocationId:  "us-central1",
			DisplayName: "Redis Cluster us-central1",
		},
		{
			Name:        fmt.Sprintf("projects/%s/locations/global", project),
			LocationId:  "global",
			DisplayName: "Redis Cluster global",
		},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&locationpb.ListLocationsResponse{
		Locations:     items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCLongrunningListOperations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.ListOperationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return grpcInvalidArgument("name-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*longrunningpb.Operation{
		{Name: name + "/operations/op-1", Done: false},
		{Name: name + "/operations/op-2", Done: true},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&longrunningpb.ListOperationsResponse{
		Operations:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCLongrunningCancelOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.CancelOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCLongrunningDeleteOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.DeleteOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4RedisCluster(project, location, clusterID string) *clusterpb.Cluster {
	replicaCount := int32(1)
	shardCount := int32(3)
	sizeGB := int32(12)
	backupCollection := fmt.Sprintf("projects/%s/locations/%s/backupCollections/collection-1", project, location)
	return &clusterpb.Cluster{
		Name:         fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterID),
		CreateTime:   timestamppb.New(gcpStage4ReferenceTime),
		State:        clusterpb.Cluster_ACTIVE,
		Uid:          "redis-cluster-" + clusterID,
		ReplicaCount: &replicaCount,
		ShardCount:   &shardCount,
		SizeGb:       &sizeGB,
		NodeType:     clusterpb.NodeType_REDIS_SHARED_CORE_NANO,
		PscConfigs: []*clusterpb.PscConfig{
			{Network: fmt.Sprintf("projects/%s/global/networks/default", project)},
		},
		DiscoveryEndpoints: []*clusterpb.DiscoveryEndpoint{
			{
				Address: "10.0.0.5",
				Port:    6379,
				PscConfig: &clusterpb.PscConfig{
					Network: fmt.Sprintf("projects/%s/global/networks/default", project),
				},
			},
		},
		RedisConfigs:     map[string]string{"maxmemory-policy": "allkeys-lru"},
		BackupCollection: &backupCollection,
	}
}

func gcpStage4RedisClusterCertificateAuthority(project, location, clusterID string) *clusterpb.CertificateAuthority {
	return &clusterpb.CertificateAuthority{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s/certificateAuthority", project, location, clusterID),
		ServerCa: &clusterpb.CertificateAuthority_ManagedServerCa{
			ManagedServerCa: &clusterpb.CertificateAuthority_ManagedCertificateAuthority{
				CaCerts: []*clusterpb.CertificateAuthority_ManagedCertificateAuthority_CertChain{
					{
						Certificates: []string{
							"-----BEGIN CERTIFICATE-----",
							"STACKYARD-REDIS-CLUSTER-CA",
							"-----END CERTIFICATE-----",
						},
					},
				},
			},
		},
	}
}

func gcpStage4RedisBackupCollection(project, location, collectionID, clusterID string) *clusterpb.BackupCollection {
	return &clusterpb.BackupCollection{
		Name:       fmt.Sprintf("projects/%s/locations/%s/backupCollections/%s", project, location, collectionID),
		ClusterUid: "redis-cluster-" + clusterID,
		Cluster:    fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterID),
		KmsKey:     fmt.Sprintf("projects/%s/locations/%s/keyRings/stackyard/cryptoKeys/redis-cluster", project, location),
		Uid:        "backup-collection-" + collectionID,
	}
}

func gcpStage4RedisBackup(project, location, collectionID, backupID, clusterID string) *clusterpb.Backup {
	return &clusterpb.Backup{
		Name:           fmt.Sprintf("projects/%s/locations/%s/backupCollections/%s/backups/%s", project, location, collectionID, backupID),
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime),
		Cluster:        fmt.Sprintf("projects/%s/locations/%s/clusters/%s", project, location, clusterID),
		ClusterUid:     "redis-cluster-" + clusterID,
		TotalSizeBytes: 2147483648,
		ExpireTime:     timestamppb.New(gcpStage4ReferenceTime.Add(24 * time.Hour)),
		EngineVersion:  "redis-7.2",
		NodeType:       clusterpb.NodeType_REDIS_SHARED_CORE_NANO,
		ReplicaCount:   1,
		ShardCount:     3,
		BackupType:     clusterpb.Backup_ON_DEMAND,
		State:          clusterpb.Backup_ACTIVE,
		Uid:            "backup-" + backupID,
	}
}

func gcpStage4RedisClusterOperation(project, location, operationID, target string) *longrunningpb.Operation {
	_ = target
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}

func gcpStage4RedisClusterUpdateMaskValid(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(path)), "_", "")
		switch normalized {
		case "sizegb", "replicacount":
			continue
		default:
			return false
		}
	}
	return true
}

func parseGCPStage4RedisLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4RedisInstanceName(name string) (project, location, instanceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "instances" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || instanceID == "" {
		return "", "", "", false
	}
	return project, location, instanceID, true
}

func parseGCPStage4RedisClusterLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4RedisClusterName(name string) (project, location, clusterID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "clusters" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || clusterID == "" {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPStage4RedisClusterCertificateAuthorityName(name string) (project, location, clusterID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 7 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "clusters" || parts[6] != "certificateAuthority" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || clusterID == "" {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPStage4RedisBackupCollectionName(name string) (project, location, collectionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "backupCollections" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	collectionID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || collectionID == "" {
		return "", "", "", false
	}
	return project, location, collectionID, true
}

func parseGCPStage4RedisBackupName(name string) (project, location, collectionID, backupID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "backupCollections" || parts[6] != "backups" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	collectionID = strings.TrimSpace(parts[5])
	backupID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || collectionID == "" || backupID == "" {
		return "", "", "", "", false
	}
	return project, location, collectionID, backupID, true
}

func parseGCPStage4LocationName(name string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4ApigeeConnectParent(parent string) (project, endpoint string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "endpoints" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	endpoint = strings.TrimSpace(parts[3])
	if project == "" || endpoint == "" {
		return "", "", false
	}
	return project, endpoint, true
}

func parseGCPStage4ProjectParent(parent string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" {
		return "", false
	}
	project = strings.TrimSpace(parts[1])
	return project, project != ""
}

func parseGCPStage4PageToken(token string) (int, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, true
	}
	value, err := strconv.Atoi(token)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}

func parseGCPStage4QuotaInfosParent(parent string) (scope, scopeID, location, service string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 6 || parts[2] != "locations" || parts[4] != "services" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	service = strings.TrimSpace(parts[5])
	if (scope != "projects" && scope != "folders" && scope != "organizations") || scopeID == "" || location == "" || service == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, service, true
}

func parseGCPStage4QuotaInfoName(name string) (scope, scopeID, location, service, quotaID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[2] != "locations" || parts[4] != "services" || parts[6] != "quotaInfos" {
		return "", "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	service = strings.TrimSpace(parts[5])
	quotaID = strings.TrimSpace(parts[7])
	if (scope != "projects" && scope != "folders" && scope != "organizations") || scopeID == "" || location == "" || service == "" || quotaID == "" {
		return "", "", "", "", "", false
	}
	return scope, scopeID, location, service, quotaID, true
}

func parseGCPStage4QuotaPreferencesParent(parent string) (scope, scopeID, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[2] != "locations" {
		return "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if (scope != "projects" && scope != "folders" && scope != "organizations") || scopeID == "" || location == "" {
		return "", "", "", false
	}
	return scope, scopeID, location, true
}

func parseGCPStage4QuotaPreferenceName(name string) (scope, scopeID, location, preferenceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[2] != "locations" || parts[4] != "quotaPreferences" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	preferenceID = strings.TrimSpace(parts[5])
	if (scope != "projects" && scope != "folders" && scope != "organizations") || scopeID == "" || location == "" || preferenceID == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, preferenceID, true
}

func parseGCPStage4ProcurementParent(parent string) (billingAccount string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 || parts[0] != "billingAccounts" {
		return "", false
	}
	billingAccount = strings.TrimSpace(parts[1])
	return billingAccount, billingAccount != ""
}

func parseGCPStage4ProcurementOrderName(name string) (billingAccount, orderID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "billingAccounts" || parts[2] != "orders" {
		return "", "", false
	}
	billingAccount = strings.TrimSpace(parts[1])
	orderID = strings.TrimSpace(parts[3])
	if billingAccount == "" || orderID == "" {
		return "", "", false
	}
	return billingAccount, orderID, true
}

func parseGCPStage4ConfigDeploymentName(name string) (project, location, deploymentID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "deployments" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	deploymentID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || deploymentID == "" {
		return "", "", "", false
	}
	return project, location, deploymentID, true
}

func parseGCPStage4ConfigPreviewName(name string) (project, location, previewID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "previews" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	previewID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || previewID == "" {
		return "", "", "", false
	}
	return project, location, previewID, true
}

func parseGCPStage4ConfigTerraformVersionName(name string) (project, location, terraformVersionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "terraformVersions" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	terraformVersionID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || terraformVersionID == "" {
		return "", "", "", false
	}
	return project, location, terraformVersionID, true
}

func parseGCPStage4ConfigResourceChangeName(name string) (project, location, previewID, resourceChangeID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "previews" || parts[6] != "resourceChanges" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	previewID = strings.TrimSpace(parts[5])
	resourceChangeID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || previewID == "" || resourceChangeID == "" {
		return "", "", "", "", false
	}
	return project, location, previewID, resourceChangeID, true
}

func parseGCPStage4ConfigResourceDriftName(name string) (project, location, previewID, resourceDriftID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "previews" || parts[6] != "resourceDrifts" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	previewID = strings.TrimSpace(parts[5])
	resourceDriftID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || previewID == "" || resourceDriftID == "" {
		return "", "", "", "", false
	}
	return project, location, previewID, resourceDriftID, true
}

func parseGCPStage4ConfigDeliveryLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4ConfigDeliveryResourceBundleName(name string) (project, location, resourceBundleID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "resourceBundles" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	resourceBundleID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || resourceBundleID == "" {
		return "", "", "", false
	}
	return project, location, resourceBundleID, true
}

func parseGCPStage4ConfigDeliveryFleetPackageName(name string) (project, location, fleetPackageID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "fleetPackages" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	fleetPackageID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || fleetPackageID == "" {
		return "", "", "", false
	}
	return project, location, fleetPackageID, true
}

func parseGCPStage4ConfigDeliveryReleaseName(name string) (project, location, resourceBundleID, releaseID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "resourceBundles" || parts[6] != "releases" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	resourceBundleID = strings.TrimSpace(parts[5])
	releaseID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || resourceBundleID == "" || releaseID == "" {
		return "", "", "", "", false
	}
	return project, location, resourceBundleID, releaseID, true
}

func parseGCPStage4ConfigDeliveryVariantName(name string) (project, location, resourceBundleID, releaseID, variantID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "resourceBundles" || parts[6] != "releases" || parts[8] != "variants" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	resourceBundleID = strings.TrimSpace(parts[5])
	releaseID = strings.TrimSpace(parts[7])
	variantID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || resourceBundleID == "" || releaseID == "" || variantID == "" {
		return "", "", "", "", "", false
	}
	return project, location, resourceBundleID, releaseID, variantID, true
}

func parseGCPStage4ConfigDeliveryRolloutName(name string) (project, location, fleetPackageID, rolloutID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "fleetPackages" || parts[6] != "rollouts" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	fleetPackageID = strings.TrimSpace(parts[5])
	rolloutID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || fleetPackageID == "" || rolloutID == "" {
		return "", "", "", "", false
	}
	return project, location, fleetPackageID, rolloutID, true
}

func parseGCPStage4RapidMigrationAssessmentParent(parent string) (project, location string, ok bool) {
	return parseGCPStage4ConfigDeliveryLocationParent(parent)
}

func parseGCPStage4RapidMigrationAssessmentCollectorName(name string) (project, location, collectorID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "collectors" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	collectorID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || collectorID == "" {
		return "", "", "", false
	}
	return project, location, collectorID, true
}

func parseGCPStage4RapidMigrationAssessmentAnnotationName(name string) (project, location, annotationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "annotations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	annotationID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || annotationID == "" {
		return "", "", "", false
	}
	return project, location, annotationID, true
}

func parseGCPStage4RetailCatalogName(name string) (project, location, catalogID, catalogName string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || catalogID == "" {
		return "", "", "", "", false
	}
	catalogName = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s", project, location, catalogID)
	return project, location, catalogID, catalogName, true
}

func parseGCPStage4RetailBranchParent(parent string) (project, location, catalogID, branchID, branchName string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != "branches" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	branchID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || catalogID == "" || branchID == "" {
		return "", "", "", "", "", false
	}
	branchName = fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s", project, location, catalogID, branchID)
	return project, location, catalogID, branchID, branchName, true
}

func parseGCPStage4RetailPlacementName(name string) (project, location, catalogID, placementID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != "placements" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	placementID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || catalogID == "" || placementID == "" {
		return "", "", "", "", false
	}
	return project, location, catalogID, placementID, true
}

func parseGCPStage4RetailServingConfigName(name string) (project, location, catalogID, servingConfigID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "catalogs" || parts[6] != "servingConfigs" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	catalogID = strings.TrimSpace(parts[5])
	servingConfigID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || catalogID == "" || servingConfigID == "" {
		return "", "", "", "", false
	}
	return project, location, catalogID, servingConfigID, true
}

func parseGCPStage4RunLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4RunServiceName(name string) (project, location, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "services" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || serviceID == "" {
		return "", "", "", false
	}
	return project, location, serviceID, true
}

func parseGCPStage4RunJobName(name string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPStage4RunExecutionName(name string) (project, location, jobID, executionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" || parts[6] != "executions" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	executionID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" || executionID == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, executionID, true
}

func parseGCPStage4RunTaskName(name string) (project, location, jobID, executionID, taskID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" || parts[6] != "executions" || parts[8] != "tasks" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	executionID = strings.TrimSpace(parts[7])
	taskID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || jobID == "" || executionID == "" || taskID == "" {
		return "", "", "", "", "", false
	}
	return project, location, jobID, executionID, taskID, true
}

func parseGCPStage4RunRevisionName(name string) (project, location, serviceID, revisionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "services" || parts[6] != "revisions" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	revisionID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || serviceID == "" || revisionID == "" {
		return "", "", "", "", false
	}
	return project, location, serviceID, revisionID, true
}

func parseGCPStage4SchedulerLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4SchedulerJobName(name string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPSchedulerJobID(jobID) {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPStage4ServiceControlServiceName(name string) (serviceName string, ok bool) {
	serviceName = strings.TrimSpace(name)
	if serviceName == "" {
		return "", false
	}
	parts := strings.Split(serviceName, ".")
	if len(parts) < 2 {
		return "", false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", false
		}
	}
	return serviceName, true
}

func parseGCPStage4ServiceHealthProjectParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) == 5 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "events" {
		parts = parts[:4]
	}
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4ServiceHealthEventName(name string) (project, location, eventID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "events" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	eventID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || eventID == "" {
		return "", "", "", false
	}
	return project, location, eventID, true
}

func parseGCPStage4ServiceHealthOrganizationParent(parent string) (orgID, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) == 5 && parts[0] == "organizations" && parts[2] == "locations" && (parts[4] == "organizationEvents" || parts[4] == "organizationImpacts") {
		parts = parts[:4]
	}
	if len(parts) != 4 || parts[0] != "organizations" || parts[2] != "locations" {
		return "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if orgID == "" || location == "" {
		return "", "", false
	}
	return orgID, location, true
}

func parseGCPStage4ServiceHealthOrganizationEventName(name string) (orgID, location, eventID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "organizations" || parts[2] != "locations" || parts[4] != "organizationEvents" {
		return "", "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	eventID = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" || eventID == "" {
		return "", "", "", false
	}
	return orgID, location, eventID, true
}

func parseGCPStage4ServiceHealthOrganizationImpactName(name string) (orgID, location, impactID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "organizations" || parts[2] != "locations" || parts[4] != "organizationImpacts" {
		return "", "", "", false
	}
	orgID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	impactID = strings.TrimSpace(parts[5])
	if orgID == "" || location == "" || impactID == "" {
		return "", "", "", false
	}
	return orgID, location, impactID, true
}

func parseGCPStage4ServiceDirectoryLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4ServiceDirectoryNamespaceName(name string) (project, location, namespaceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "namespaces" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	namespaceID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPServiceDirectoryID(namespaceID) {
		return "", "", "", false
	}
	return project, location, namespaceID, true
}

func parseGCPStage4ServiceDirectoryServiceParent(parent string) (project, location, namespaceID string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "namespaces" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	namespaceID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPServiceDirectoryID(namespaceID) {
		return "", "", "", false
	}
	return project, location, namespaceID, true
}

func parseGCPStage4ServiceDirectoryServiceName(name string) (project, location, namespaceID, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "namespaces" || parts[6] != "services" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	namespaceID = strings.TrimSpace(parts[5])
	serviceID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || !isGCPServiceDirectoryID(namespaceID) || !isGCPServiceDirectoryID(serviceID) {
		return "", "", "", "", false
	}
	return project, location, namespaceID, serviceID, true
}

func parseGCPStage4ServiceDirectoryEndpointParent(parent string) (project, location, namespaceID, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "namespaces" || parts[6] != "services" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	namespaceID = strings.TrimSpace(parts[5])
	serviceID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || !isGCPServiceDirectoryID(namespaceID) || !isGCPServiceDirectoryID(serviceID) {
		return "", "", "", "", false
	}
	return project, location, namespaceID, serviceID, true
}

func parseGCPStage4ServiceDirectoryEndpointName(name string) (project, location, namespaceID, serviceID, endpointID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "namespaces" || parts[6] != "services" || parts[8] != "endpoints" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	namespaceID = strings.TrimSpace(parts[5])
	serviceID = strings.TrimSpace(parts[7])
	endpointID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || !isGCPServiceDirectoryID(namespaceID) || !isGCPServiceDirectoryID(serviceID) || !isGCPServiceDirectoryID(endpointID) {
		return "", "", "", "", "", false
	}
	return project, location, namespaceID, serviceID, endpointID, true
}

func parseGCPStage4ServiceDirectoryIAMResource(resource string) (project, location, namespaceID, serviceID string, namespaceOnly bool, ok bool) {
	if p, l, n, ok := parseGCPStage4ServiceDirectoryNamespaceName(resource); ok {
		return p, l, n, "", true, true
	}
	p, l, n, s, ok := parseGCPStage4ServiceDirectoryServiceName(resource)
	if !ok {
		return "", "", "", "", false, false
	}
	return p, l, n, s, false, true
}

func parseGCPStage4SecretManagerParent(parent string) (project, location string, hasLocation bool, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) == 2 && parts[0] == "projects" {
		project = strings.TrimSpace(parts[1])
		if project == "" {
			return "", "", false, false
		}
		return project, "", false, true
	}
	if len(parts) == 4 && parts[0] == "projects" && parts[2] == "locations" {
		project = strings.TrimSpace(parts[1])
		location = strings.TrimSpace(parts[3])
		if project == "" || location == "" {
			return "", "", false, false
		}
		return project, location, true, true
	}
	return "", "", false, false
}

func parseGCPStage4SecretManagerSecretName(name string) (project, location, secretID string, hasLocation bool, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) == 4 && parts[0] == "projects" && parts[2] == "secrets" {
		project = strings.TrimSpace(parts[1])
		secretID = strings.TrimSpace(parts[3])
		if project == "" || !isGCPSecretManagerID(secretID) {
			return "", "", "", false, false
		}
		return project, "", secretID, false, true
	}
	if len(parts) == 6 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "secrets" {
		project = strings.TrimSpace(parts[1])
		location = strings.TrimSpace(parts[3])
		secretID = strings.TrimSpace(parts[5])
		if project == "" || location == "" || !isGCPSecretManagerID(secretID) {
			return "", "", "", false, false
		}
		return project, location, secretID, true, true
	}
	return "", "", "", false, false
}

func parseGCPStage4SecretManagerVersionName(name string) (project, location, secretID, versionID string, hasLocation bool, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) == 6 && parts[0] == "projects" && parts[2] == "secrets" && parts[4] == "versions" {
		project = strings.TrimSpace(parts[1])
		secretID = strings.TrimSpace(parts[3])
		versionID = strings.TrimSpace(parts[5])
		if project == "" || !isGCPSecretManagerID(secretID) || !isGCPSecretManagerVersionID(versionID) {
			return "", "", "", "", false, false
		}
		return project, "", secretID, versionID, false, true
	}
	if len(parts) == 8 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "secrets" && parts[6] == "versions" {
		project = strings.TrimSpace(parts[1])
		location = strings.TrimSpace(parts[3])
		secretID = strings.TrimSpace(parts[5])
		versionID = strings.TrimSpace(parts[7])
		if project == "" || location == "" || !isGCPSecretManagerID(secretID) || !isGCPSecretManagerVersionID(versionID) {
			return "", "", "", "", false, false
		}
		return project, location, secretID, versionID, true, true
	}
	return "", "", "", "", false, false
}

func parseGCPStage4SecureSourceManagerLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4SecureSourceManagerRepositoryName(name string) (project, location, repositoryID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "repositories" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	repositoryID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPSecureSourceManagerID(repositoryID) {
		return "", "", "", false
	}
	return project, location, repositoryID, true
}

func parseGCPStage4SecureSourceManagerPullRequestName(name string) (project, location, repositoryID, pullRequestID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "repositories" || parts[6] != "pullRequests" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	repositoryID = strings.TrimSpace(parts[5])
	pullRequestID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || !isGCPSecureSourceManagerID(repositoryID) || !isGCPSecureSourceManagerID(pullRequestID) {
		return "", "", "", "", false
	}
	return project, location, repositoryID, pullRequestID, true
}

func gcpStage4GRPCRecommendationEngineCreateCatalogItem(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.CreateCatalogItemRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, parent, ok := parseGCPRecommendationCatalogParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetCatalogItem() == nil || strings.TrimSpace(req.GetCatalogItem().GetId()) == "" || strings.TrimSpace(req.GetCatalogItem().GetTitle()) == "" {
		return grpcInvalidArgument("catalog_item.id-and-title-required")
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineCatalogItem(project, catalog, parent, req.GetCatalogItem().GetId(), req.GetCatalogItem().GetTitle()))
}

func gcpStage4GRPCRecommendationEngineGetCatalogItem(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.GetCatalogItemRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, parent, itemID, ok := parseGCPRecommendationCatalogItemName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineCatalogItem(project, catalog, parent, itemID, "Stackyard Item "+itemID))
}

func gcpStage4GRPCRecommendationEngineListCatalogItems(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.ListCatalogItemsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, parent, ok := parseGCPRecommendationCatalogParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recommendationenginepb.CatalogItem{
		gcpStage4RecommendationEngineCatalogItem(project, catalog, parent, "item-1", "Stackyard Item 1"),
		gcpStage4RecommendationEngineCatalogItem(project, catalog, parent, "item-2", "Stackyard Item 2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recommendationenginepb.ListCatalogItemsResponse{
		CatalogItems:  items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRecommendationEngineUpdateCatalogItem(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.UpdateCatalogItemRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetCatalogItem() == nil {
		return grpcInvalidArgument("catalog_item-required")
	}
	project, catalog, parent, itemID, ok := parseGCPRecommendationCatalogItemName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if strings.TrimSpace(req.GetCatalogItem().GetId()) == "" || req.GetCatalogItem().GetId() != itemID {
		return grpcInvalidArgument("catalog_item.id-must-match-name")
	}
	title := strings.TrimSpace(req.GetCatalogItem().GetTitle())
	if title == "" {
		title = "Stackyard Item " + itemID
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineCatalogItem(project, catalog, parent, itemID, title))
}

func gcpStage4GRPCRecommendationEngineDeleteCatalogItem(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.DeleteCatalogItemRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPRecommendationCatalogItemName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCRecommendationEngineImportCatalogItems(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.ImportCatalogItemsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, _, ok := parseGCPRecommendationCatalogParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetInputConfig() == nil || req.GetInputConfig().GetSource() == nil {
		return grpcInvalidArgument("input_config-required")
	}
	opID := "importCatalogItems-1"
	if rid := strings.TrimSpace(req.GetRequestId()); rid != "" {
		opID = "importCatalogItems-" + rid
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineOperation(project, catalog, opID))
}

func gcpStage4GRPCRecommendationEngineWriteUserEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.WriteUserEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, eventStore, parent, ok := parseGCPRecommendationEventStoreParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if !gcpStage4RecommendationEngineValidUserEventProto(req.GetUserEvent()) {
		return grpcInvalidArgument("user_event.event_type-and-user_info.visitor_id-required")
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineUserEvent(project, catalog, eventStore, parent, req.GetUserEvent()))
}

func gcpStage4GRPCRecommendationEngineCollectUserEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.CollectUserEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPRecommendationEventStoreParentName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetUserEvent()) == "" {
		return grpcInvalidArgument("user_event-required")
	}
	return grpcProtoSuccess(&httpbodypb.HttpBody{
		ContentType: "application/json",
		Data:        []byte(`{"status":"ok","provider":"gcp"}`),
	})
}

func gcpStage4GRPCRecommendationEngineListUserEvents(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.ListUserEventsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, eventStore, parent, ok := parseGCPRecommendationEventStoreParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recommendationenginepb.UserEvent{
		gcpStage4RecommendationEngineUserEvent(project, catalog, eventStore, parent, &recommendationenginepb.UserEvent{}),
		gcpStage4RecommendationEngineUserEvent(project, catalog, eventStore, parent, &recommendationenginepb.UserEvent{
			EventType: "search",
			UserInfo:  &recommendationenginepb.UserInfo{VisitorId: "visitor-2", UserId: "user-2"},
		}),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recommendationenginepb.ListUserEventsResponse{
		UserEvents:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRecommendationEnginePurgeUserEvents(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.PurgeUserEventsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, _, _, ok := parseGCPRecommendationEventStoreParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetFilter()) == "" {
		return grpcInvalidArgument("filter-required")
	}
	opID := "purgeUserEvents-1"
	if req.GetForce() {
		opID = "purgeUserEvents-force"
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineOperation(project, catalog, opID))
}

func gcpStage4GRPCRecommendationEngineImportUserEvents(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.ImportUserEventsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, _, _, ok := parseGCPRecommendationEventStoreParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetInputConfig() == nil || req.GetInputConfig().GetSource() == nil {
		return grpcInvalidArgument("input_config-required")
	}
	opID := "importUserEvents-1"
	if rid := strings.TrimSpace(req.GetRequestId()); rid != "" {
		opID = "importUserEvents-" + rid
	}
	return grpcProtoSuccess(gcpStage4RecommendationEngineOperation(project, catalog, opID))
}

func gcpStage4GRPCRecommendationEnginePredict(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.PredictRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, catalog, eventStore, placement, ok := parseGCPRecommendationPlacementName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !gcpStage4RecommendationEngineValidUserEventProto(req.GetUserEvent()) {
		return grpcInvalidArgument("user_event.event_type-and-user_info.visitor_id-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recommendationenginepb.PredictResponse_PredictionResult{
		gcpStage4RecommendationEnginePredictionResult(project, catalog, eventStore, placement, "item-1", 0.97),
		gcpStage4RecommendationEnginePredictionResult(project, catalog, eventStore, placement, "item-2", 0.91),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recommendationenginepb.PredictResponse{
		Results:               items[start:end],
		RecommendationToken:   "rec-token-1",
		ItemsMissingInCatalog: []string{},
		DryRun:                req.GetDryRun(),
		Metadata: map[string]*structpb.Value{
			"placement": structpb.NewStringValue(placement),
		},
		NextPageToken: next,
	})
}

func gcpStage4GRPCRecommendationEngineCreatePredictionAPIKeyRegistration(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.CreatePredictionApiKeyRegistrationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPRecommendationEventStoreParentName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	reg := req.GetPredictionApiKeyRegistration()
	if reg == nil || strings.TrimSpace(reg.GetApiKey()) == "" {
		return grpcInvalidArgument("prediction_api_key_registration.api_key-required")
	}
	return grpcProtoSuccess(&recommendationenginepb.PredictionApiKeyRegistration{ApiKey: reg.GetApiKey()})
}

func gcpStage4GRPCRecommendationEngineListPredictionAPIKeyRegistrations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.ListPredictionApiKeyRegistrationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPRecommendationEventStoreParentName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recommendationenginepb.PredictionApiKeyRegistration{
		{ApiKey: "stackyard-api-key"},
		{ApiKey: "stackyard-api-key-2"},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recommendationenginepb.ListPredictionApiKeyRegistrationsResponse{
		PredictionApiKeyRegistrations: items[start:end],
		NextPageToken:                 next,
	})
}

func gcpStage4GRPCRecommendationEngineDeletePredictionAPIKeyRegistration(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommendationenginepb.DeletePredictionApiKeyRegistrationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPRecommendationPredictionAPIKeyRegistrationName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4RecommendationEngineCatalogItem(project, catalog, parent, itemID, title string) *recommendationenginepb.CatalogItem {
	if strings.TrimSpace(title) == "" {
		title = "Stackyard Item " + itemID
	}
	return &recommendationenginepb.CatalogItem{
		Id:    itemID,
		Title: title,
		Description: fmt.Sprintf(
			"Stackyard recommendation item %s in %s", itemID, parent,
		),
		CategoryHierarchies: []*recommendationenginepb.CatalogItem_CategoryHierarchy{
			{Categories: []string{"Books", "Fiction"}},
		},
		ItemAttributes: &recommendationenginepb.FeatureMap{
			CategoricalFeatures: map[string]*recommendationenginepb.FeatureMap_StringList{
				"brand": {Value: []string{"Stackyard"}},
			},
		},
		LanguageCode: "en",
		Tags:         []string{"staged", "recommendationengine"},
		ItemGroupId:  "group-1",
		RecommendationType: &recommendationenginepb.CatalogItem_ProductMetadata{
			ProductMetadata: &recommendationenginepb.ProductCatalogItem{
				StockState:          recommendationenginepb.ProductCatalogItem_IN_STOCK,
				AvailableQuantity:   100,
				CurrencyCode:        "USD",
				CanonicalProductUri: fmt.Sprintf("https://example.com/%s/%s", catalog, itemID),
			},
		},
	}
}

func gcpStage4RecommendationEngineUserEvent(project, catalog, eventStore, parent string, event *recommendationenginepb.UserEvent) *recommendationenginepb.UserEvent {
	if event == nil {
		event = &recommendationenginepb.UserEvent{}
	}
	eventType := strings.TrimSpace(event.GetEventType())
	if eventType == "" {
		eventType = "detail-page-view"
	}
	visitorID := "visitor-1"
	userID := "user-1"
	if event.GetUserInfo() != nil {
		if v := strings.TrimSpace(event.GetUserInfo().GetVisitorId()); v != "" {
			visitorID = v
		}
		if u := strings.TrimSpace(event.GetUserInfo().GetUserId()); u != "" {
			userID = u
		}
	}
	return &recommendationenginepb.UserEvent{
		EventType: eventType,
		UserInfo: &recommendationenginepb.UserInfo{
			VisitorId: visitorID,
			UserId:    userID,
		},
		ProductEventDetail: &recommendationenginepb.ProductEventDetail{
			ProductDetails: []*recommendationenginepb.ProductDetail{
				{
					Id:         "item-1",
					Quantity:   1,
					StockState: recommendationenginepb.ProductCatalogItem_IN_STOCK,
				},
			},
		},
		EventTime:   timestamppb.New(gcpStage4ReferenceTime),
		EventSource: recommendationenginepb.UserEvent_EVENT_SOURCE_UNSPECIFIED,
		EventDetail: &recommendationenginepb.EventDetail{
			Uri: fmt.Sprintf("https://example.com/%s/%s/%s", project, catalog, eventStore),
		},
	}
}

func gcpStage4RecommendationEnginePredictionResult(project, catalog, eventStore, placement, itemID string, score float64) *recommendationenginepb.PredictResponse_PredictionResult {
	return &recommendationenginepb.PredictResponse_PredictionResult{
		Id: itemID,
		ItemMetadata: map[string]*structpb.Value{
			"score":      structpb.NewNumberValue(score),
			"placement":  structpb.NewStringValue(placement),
			"eventStore": structpb.NewStringValue(eventStore),
			"project":    structpb.NewStringValue(project),
			"catalog":    structpb.NewStringValue(catalog),
		},
	}
}

func gcpStage4RecommendationEngineOperation(project, catalog, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/global/catalogs/%s/operations/%s", project, catalog, operationID),
		Done: false,
	}
}

func gcpStage4RecommendationEngineValidUserEventProto(event *recommendationenginepb.UserEvent) bool {
	if event == nil || strings.TrimSpace(event.GetEventType()) == "" || event.GetUserInfo() == nil {
		return false
	}
	return strings.TrimSpace(event.GetUserInfo().GetVisitorId()) != ""
}

func gcpStage4GRPCRetailListProducts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &retailpb.ListProductsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, catalogID, branchID, _, ok := parseGCPStage4RetailBranchParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*retailpb.Product{
		gcpStage4RetailProduct(project, location, catalogID, branchID, "product-1", "Stackyard Product 1"),
		gcpStage4RetailProduct(project, location, catalogID, branchID, "product-2", "Stackyard Product 2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&retailpb.ListProductsResponse{
		Products:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRetailCreateProduct(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &retailpb.CreateProductRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, catalogID, branchID, _, ok := parseGCPStage4RetailBranchParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetProductId()) == "" {
		return grpcInvalidArgument("product_id-required")
	}
	if req.GetProduct() == nil || strings.TrimSpace(req.GetProduct().GetTitle()) == "" {
		return grpcInvalidArgument("product.title-required")
	}
	return grpcProtoSuccess(gcpStage4RetailProduct(project, location, catalogID, branchID, req.GetProductId(), req.GetProduct().GetTitle()))
}

func gcpStage4GRPCRetailSearch(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &retailpb.SearchRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, catalogID, _, ok := parseGCPStage4RetailPlacementName(req.GetPlacement())
	if !ok {
		return grpcInvalidArgument("placement-required")
	}
	if strings.TrimSpace(req.GetQuery()) == "" {
		return grpcInvalidArgument("query-required")
	}
	product := gcpStage4RetailProduct(project, location, catalogID, "default_branch", "product-1", "Search Product")
	return grpcProtoSuccess(&retailpb.SearchResponse{
		Results: []*retailpb.SearchResponse_SearchResult{
			{
				Id:                   "product-1",
				Product:              product,
				MatchingVariantCount: 1,
			},
		},
		TotalSize:                  1,
		CorrectedQuery:             req.GetQuery(),
		AttributionToken:           "stackyard-search-token",
		AppliedControls:            []string{},
		NextPageToken:              "",
		RedirectUri:                "",
		ExperimentInfo:             []*retailpb.ExperimentInfo{},
		InvalidConditionBoostSpecs: []*retailpb.SearchRequest_BoostSpec_ConditionBoostSpec{},
		ConversationalSearchResult: nil,
		TileNavigationResult:       nil,
		PinControlMetadata:         nil,
		QueryExpansionInfo:         nil,
	})
}

func gcpStage4GRPCRetailCreateServingConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &retailpb.CreateServingConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, catalogID, _, ok := parseGCPStage4RetailCatalogName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetServingConfigId()) == "" {
		return grpcInvalidArgument("serving_config_id-required")
	}
	if req.GetServingConfig() == nil || strings.TrimSpace(req.GetServingConfig().GetDisplayName()) == "" {
		return grpcInvalidArgument("serving_config.display_name-required")
	}
	return grpcProtoSuccess(gcpStage4RetailServingConfig(project, location, catalogID, req.GetServingConfigId(), req.GetServingConfig().GetDisplayName()))
}

func gcpStage4GRPCRetailGetServingConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &retailpb.GetServingConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, catalogID, servingConfigID, ok := parseGCPStage4RetailServingConfigName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RetailServingConfig(project, location, catalogID, servingConfigID, "Serving Config "+servingConfigID))
}

func gcpStage4RetailProduct(project, location, catalogID, branchID, productID, title string) *retailpb.Product {
	if strings.TrimSpace(title) == "" {
		title = "Product " + productID
	}
	return &retailpb.Product{
		Name:         fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/branches/%s/products/%s", project, location, catalogID, branchID, productID),
		Id:           productID,
		Type:         retailpb.Product_PRIMARY,
		Title:        title,
		Description:  "Stackyard retail product",
		Categories:   []string{"Apparel > Tops"},
		LanguageCode: "en-US",
		Availability: retailpb.Product_IN_STOCK,
		PriceInfo: &retailpb.PriceInfo{
			CurrencyCode: "USD",
			Price:        19.99,
		},
		PublishTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4RetailServingConfig(project, location, catalogID, servingConfigID, displayName string) *retailpb.ServingConfig {
	if strings.TrimSpace(displayName) == "" {
		displayName = "Serving Config " + servingConfigID
	}
	return &retailpb.ServingConfig{
		Name:                fmt.Sprintf("projects/%s/locations/%s/catalogs/%s/servingConfigs/%s", project, location, catalogID, servingConfigID),
		DisplayName:         displayName,
		ModelId:             "recommended-for-you",
		PriceRerankingLevel: "no-price-reranking",
		FacetControlIds:     []string{},
		BoostControlIds:     []string{},
		FilterControlIds:    []string{},
		RedirectControlIds:  []string{},
	}
}

func gcpStage4GRPCRunListServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.ListServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RunLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*runpb.Service{
		gcpStage4RunService(project, location, "service-1"),
		gcpStage4RunService(project, location, "service-2"),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&runpb.ListServicesResponse{
		Services:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRunGetService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.GetServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, serviceID, ok := parseGCPStage4RunServiceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RunService(project, location, serviceID))
}

func gcpStage4GRPCRunCreateService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.CreateServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RunLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	serviceID := strings.TrimSpace(req.GetServiceId())
	if serviceID == "" {
		return grpcInvalidArgument("service_id-required")
	}
	if !isGCPRunResourceID(serviceID) {
		return grpcInvalidArgument("service_id-invalid")
	}
	service := req.GetService()
	if service == nil || service.GetTemplate() == nil || len(service.GetTemplate().GetContainers()) == 0 {
		return grpcInvalidArgument("service.template.containers-required")
	}
	expectedName := gcpRunServiceName(project, location, serviceID)
	if got := strings.TrimSpace(service.GetName()); got != "" && got != expectedName {
		return grpcInvalidArgument("service.name-must-match-parent-and-service_id")
	}
	return grpcProtoSuccess(gcpStage4RunOperation(project, location, "createService."+serviceID, expectedName, false))
}

func gcpStage4GRPCRunListJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.ListJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RunLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*runpb.Job{
		gcpStage4RunJob(project, location, "job-1"),
		gcpStage4RunJob(project, location, "job-2"),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&runpb.ListJobsResponse{
		Jobs:          items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRunGetJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.GetJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4RunJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RunJob(project, location, jobID))
}

func gcpStage4GRPCRunCreateJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.CreateJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4RunLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	jobID := strings.TrimSpace(req.GetJobId())
	if jobID == "" {
		return grpcInvalidArgument("job_id-required")
	}
	if !isGCPRunResourceID(jobID) {
		return grpcInvalidArgument("job_id-invalid")
	}
	job := req.GetJob()
	if job == nil || job.GetTemplate() == nil || job.GetTemplate().GetTemplate() == nil || len(job.GetTemplate().GetTemplate().GetContainers()) == 0 {
		return grpcInvalidArgument("job.template.template.containers-required")
	}
	expectedName := gcpRunJobName(project, location, jobID)
	if got := strings.TrimSpace(job.GetName()); got != "" && got != expectedName {
		return grpcInvalidArgument("job.name-must-match-parent-and-job_id")
	}
	return grpcProtoSuccess(gcpStage4RunOperation(project, location, "createJob."+jobID, expectedName, false))
}

func gcpStage4GRPCRunRunJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.RunJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4RunJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpRunEtag(jobID) {
		return grpcFailedPrecondition("etag-mismatch")
	}
	executionName := gcpRunExecutionName(project, location, jobID, "execution-1")
	return grpcProtoSuccess(gcpStage4RunOperation(project, location, "runJob."+jobID, executionName, false))
}

func gcpStage4GRPCRunListExecutions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.ListExecutionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4RunJobName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*runpb.Execution{
		gcpStage4RunExecution(project, location, jobID, "execution-1"),
		gcpStage4RunExecution(project, location, jobID, "execution-2"),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&runpb.ListExecutionsResponse{
		Executions:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRunGetExecution(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.GetExecutionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, executionID, ok := parseGCPStage4RunExecutionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RunExecution(project, location, jobID, executionID))
}

func gcpStage4GRPCRunListTasks(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.ListTasksRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, executionID, ok := parseGCPStage4RunExecutionName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*runpb.Task{
		gcpStage4RunTask(project, location, jobID, executionID, "task-1"),
		gcpStage4RunTask(project, location, jobID, executionID, "task-2"),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&runpb.ListTasksResponse{
		Tasks:         items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRunGetTask(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.GetTaskRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, executionID, taskID, ok := parseGCPStage4RunTaskName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RunTask(project, location, jobID, executionID, taskID))
}

func gcpStage4GRPCRunListRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.ListRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, serviceID, ok := parseGCPStage4RunServiceName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*runpb.Revision{
		gcpStage4RunRevision(project, location, serviceID, serviceID+"-00001"),
		gcpStage4RunRevision(project, location, serviceID, serviceID+"-00002"),
	}
	if !req.GetShowDeleted() {
		items = items[:1]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&runpb.ListRevisionsResponse{
		Revisions:     items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRunGetRevision(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &runpb.GetRevisionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, serviceID, revisionID, ok := parseGCPStage4RunRevisionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RunRevision(project, location, serviceID, revisionID))
}

func gcpStage4RunService(project, location, serviceID string) *runpb.Service {
	name := gcpRunServiceName(project, location, serviceID)
	revisionID := serviceID + "-00001"
	return &runpb.Service{
		Name:                  name,
		Uid:                   "run-service-" + serviceID,
		Generation:            2,
		ObservedGeneration:    2,
		Labels:                map[string]string{"env": "staged"},
		CreateTime:            timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:            timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
		LatestReadyRevision:   gcpRunRevisionName(project, location, serviceID, revisionID),
		LatestCreatedRevision: gcpRunRevisionName(project, location, serviceID, revisionID),
		Uri:                   "https://" + serviceID + "-" + location + ".a.run.app",
		Template: &runpb.RevisionTemplate{
			Containers: []*runpb.Container{{
				Image: "us-docker.pkg.dev/cloudrun/container/hello",
				Name:  "app",
			}},
		},
		Etag: gcpRunEtag(serviceID),
	}
}

func gcpStage4RunJob(project, location, jobID string) *runpb.Job {
	name := gcpRunJobName(project, location, jobID)
	executionName := gcpRunExecutionName(project, location, jobID, "execution-1")
	return &runpb.Job{
		Name:               name,
		Uid:                "run-job-" + jobID,
		Generation:         3,
		ObservedGeneration: 3,
		Labels:             map[string]string{"env": "staged"},
		CreateTime:         timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:         timestamppb.New(gcpStage4ReferenceTime.Add(3 * time.Hour)),
		Template: &runpb.ExecutionTemplate{
			Parallelism: 1,
			TaskCount:   1,
			Template: &runpb.TaskTemplate{
				Containers: []*runpb.Container{{
					Image: "us-docker.pkg.dev/cloudrun/container/job",
					Name:  "job",
				}},
			},
		},
		LatestCreatedExecution: &runpb.ExecutionReference{
			Name: executionName,
		},
		ExecutionCount: 1,
		Etag:           gcpRunEtag(jobID),
	}
}

func gcpStage4RunExecution(project, location, jobID, executionID string) *runpb.Execution {
	name := gcpRunExecutionName(project, location, jobID, executionID)
	jobName := gcpRunJobName(project, location, jobID)
	return &runpb.Execution{
		Name:               name,
		Uid:                "run-execution-" + executionID,
		Generation:         1,
		ObservedGeneration: 1,
		Job:                jobName,
		CreateTime:         timestamppb.New(gcpStage4ReferenceTime.Add(4 * time.Hour)),
		StartTime:          timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 30*time.Second)),
		CompletionTime:     timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 5*time.Minute)),
		UpdateTime:         timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 5*time.Minute)),
		TaskCount:          1,
		Parallelism:        1,
		SucceededCount:     1,
		RunningCount:       0,
		FailedCount:        0,
		Etag:               gcpRunEtag(executionID),
	}
}

func gcpStage4RunTask(project, location, jobID, executionID, taskID string) *runpb.Task {
	name := gcpRunTaskName(project, location, jobID, executionID, taskID)
	jobName := gcpRunJobName(project, location, jobID)
	executionName := gcpRunExecutionName(project, location, jobID, executionID)
	return &runpb.Task{
		Name:           name,
		Uid:            "run-task-" + taskID,
		Generation:     1,
		Job:            jobName,
		Execution:      executionName,
		Index:          0,
		CreateTime:     timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 10*time.Second)),
		ScheduledTime:  timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 20*time.Second)),
		StartTime:      timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 30*time.Second)),
		CompletionTime: timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 3*time.Minute)),
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime.Add(4*time.Hour + 3*time.Minute)),
		Containers: []*runpb.Container{{
			Image: "us-docker.pkg.dev/cloudrun/container/job",
			Name:  "job",
		}},
		Etag: gcpRunEtag(taskID),
	}
}

func gcpStage4RunRevision(project, location, serviceID, revisionID string) *runpb.Revision {
	name := gcpRunRevisionName(project, location, serviceID, revisionID)
	serviceName := gcpRunServiceName(project, location, serviceID)
	return &runpb.Revision{
		Name:               name,
		Uid:                "run-revision-" + revisionID,
		Generation:         1,
		ObservedGeneration: 1,
		Service:            serviceName,
		CreateTime:         timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:         timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
		Containers: []*runpb.Container{{
			Image: "us-docker.pkg.dev/cloudrun/container/hello",
			Name:  "app",
		}},
		Etag: gcpRunEtag(revisionID),
	}
}

func gcpStage4RunOperation(project, location, operationID, target string, done bool) *longrunningpb.Operation {
	_ = target
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: done,
	}
}

func gcpStage4GRPCSchedulerListJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.ListJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4SchedulerLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 500 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*schedulerpb.Job{
		gcpStage4SchedulerJob(project, location, "job-1", schedulerpb.Job_ENABLED),
		gcpStage4SchedulerJob(project, location, "job-paused", schedulerpb.Job_PAUSED),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&schedulerpb.ListJobsResponse{
		Jobs:          items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSchedulerGetJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.GetJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4SchedulerJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SchedulerJob(project, location, jobID, gcpStage4SchedulerState(jobID)))
}

func gcpStage4GRPCSchedulerCreateJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.CreateJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4SchedulerLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if reason := gcpStage4SchedulerValidateJob(req.GetJob(), false); reason != "" {
		return grpcInvalidArgument(reason)
	}

	job := req.GetJob()
	jobID := "job-1"
	if name := strings.TrimSpace(job.GetName()); name != "" {
		p, l, id, ok := parseGCPStage4SchedulerJobName(name)
		if !ok {
			return grpcInvalidArgument("job.name-invalid")
		}
		if p != project || l != location {
			return grpcInvalidArgument("job.name-must-match-parent")
		}
		jobID = id
	}
	return grpcProtoSuccess(gcpStage4SchedulerJobFromRequest(project, location, jobID, schedulerpb.Job_ENABLED, job))
}

func gcpStage4GRPCSchedulerUpdateJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.UpdateJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if reason := gcpStage4SchedulerValidateJob(req.GetJob(), true); reason != "" {
		return grpcInvalidArgument(reason)
	}
	project, location, jobID, ok := parseGCPStage4SchedulerJobName(req.GetJob().GetName())
	if !ok {
		return grpcInvalidArgument("job.name-required")
	}
	return grpcProtoSuccess(gcpStage4SchedulerJobFromRequest(project, location, jobID, gcpStage4SchedulerState(jobID), req.GetJob()))
}

func gcpStage4GRPCSchedulerDeleteJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.DeleteJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4SchedulerJobName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCSchedulerPauseJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.PauseJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4SchedulerJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpStage4SchedulerState(jobID) != schedulerpb.Job_ENABLED {
		return grpcFailedPrecondition("job-already-paused")
	}
	return grpcProtoSuccess(gcpStage4SchedulerJob(project, location, jobID, schedulerpb.Job_PAUSED))
}

func gcpStage4GRPCSchedulerResumeJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.ResumeJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4SchedulerJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpStage4SchedulerState(jobID) != schedulerpb.Job_PAUSED {
		return grpcFailedPrecondition("job-not-paused")
	}
	return grpcProtoSuccess(gcpStage4SchedulerJob(project, location, jobID, schedulerpb.Job_ENABLED))
}

func gcpStage4GRPCSchedulerRunJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &schedulerpb.RunJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, jobID, ok := parseGCPStage4SchedulerJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	job := gcpStage4SchedulerJob(project, location, jobID, gcpStage4SchedulerState(jobID))
	job.LastAttemptTime = timestamppb.New(gcpStage4ReferenceTime.Add(45 * time.Second))
	job.ScheduleTime = timestamppb.New(gcpStage4ReferenceTime.Add(15 * time.Minute))
	return grpcProtoSuccess(job)
}

func gcpStage4SchedulerValidateJob(job *schedulerpb.Job, requireName bool) string {
	if job == nil {
		return "job-required"
	}
	if requireName && strings.TrimSpace(job.GetName()) == "" {
		return "job.name-required"
	}
	if strings.TrimSpace(job.GetSchedule()) == "" {
		return "job.schedule-required"
	}
	if strings.TrimSpace(job.GetTimeZone()) == "" {
		return "job.time_zone-required"
	}
	if job.GetHttpTarget() == nil && job.GetPubsubTarget() == nil && job.GetAppEngineHttpTarget() == nil {
		return "job.target-required"
	}
	if httpTarget := job.GetHttpTarget(); httpTarget != nil && strings.TrimSpace(httpTarget.GetUri()) == "" {
		return "job.http_target.uri-required"
	}
	if pubsubTarget := job.GetPubsubTarget(); pubsubTarget != nil && strings.TrimSpace(pubsubTarget.GetTopicName()) == "" {
		return "job.pubsub_target.topic_name-required"
	}
	return ""
}

func gcpStage4SchedulerState(jobID string) schedulerpb.Job_State {
	if strings.Contains(strings.ToLower(strings.TrimSpace(jobID)), "paused") {
		return schedulerpb.Job_PAUSED
	}
	return schedulerpb.Job_ENABLED
}

func gcpStage4SchedulerJobFromRequest(project, location, jobID string, state schedulerpb.Job_State, req *schedulerpb.Job) *schedulerpb.Job {
	job := gcpStage4SchedulerJob(project, location, jobID, state)
	if req == nil {
		return job
	}
	if v := strings.TrimSpace(req.GetDescription()); v != "" {
		job.Description = v
	}
	if v := strings.TrimSpace(req.GetSchedule()); v != "" {
		job.Schedule = v
	}
	if v := strings.TrimSpace(req.GetTimeZone()); v != "" {
		job.TimeZone = v
	}
	if req.GetRetryConfig() != nil {
		job.RetryConfig = req.GetRetryConfig()
	}
	if req.GetAttemptDeadline() != nil {
		job.AttemptDeadline = req.GetAttemptDeadline()
	}
	switch target := req.GetTarget().(type) {
	case *schedulerpb.Job_HttpTarget:
		job.Target = &schedulerpb.Job_HttpTarget{HttpTarget: target.HttpTarget}
	case *schedulerpb.Job_PubsubTarget:
		job.Target = &schedulerpb.Job_PubsubTarget{PubsubTarget: target.PubsubTarget}
	case *schedulerpb.Job_AppEngineHttpTarget:
		job.Target = &schedulerpb.Job_AppEngineHttpTarget{AppEngineHttpTarget: target.AppEngineHttpTarget}
	}
	return job
}

func gcpStage4SchedulerJob(project, location, jobID string, state schedulerpb.Job_State) *schedulerpb.Job {
	return &schedulerpb.Job{
		Name:            gcpSchedulerJobName(project, location, jobID),
		Description:     "Stackyard Scheduler job " + jobID,
		Schedule:        "*/15 * * * *",
		TimeZone:        "UTC",
		UserUpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
		State:           state,
		ScheduleTime:    timestamppb.New(gcpStage4ReferenceTime.Add(15 * time.Minute)),
		LastAttemptTime: timestamppb.New(gcpStage4ReferenceTime.Add(30 * time.Second)),
		RetryConfig: &schedulerpb.RetryConfig{
			RetryCount:         3,
			MaxRetryDuration:   durationpb.New(60 * time.Second),
			MinBackoffDuration: durationpb.New(5 * time.Second),
			MaxBackoffDuration: durationpb.New(time.Hour),
			MaxDoublings:       5,
		},
		AttemptDeadline: durationpb.New(180 * time.Second),
		Target: &schedulerpb.Job_HttpTarget{
			HttpTarget: &schedulerpb.HttpTarget{
				Uri:        "https://example.com/stackyard-scheduler",
				HttpMethod: schedulerpb.HttpMethod_POST,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
		},
	}
}

func gcpStage4GRPCStorageBatchOperations(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpStorageBatchOperationsListJobsMethod:
		req := &storagebatchoperationspb.ListJobsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStorageBatchOperationsParent(strings.TrimSpace(req.GetParent()))
		if !ok || location != "global" {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		orderBy := normalizeGCPStorageBatchOperationsOrderBy(req.GetOrderBy())
		if !isGCPStorageBatchOperationsOrderBy(orderBy) {
			return grpcInvalidArgument("order_by-invalid")
		}
		filterState, filterValid := parseGCPStorageBatchOperationsStateFilter(req.GetFilter())
		if !filterValid {
			return grpcInvalidArgument("filter-invalid")
		}
		items := []*storagebatchoperationspb.Job{
			gcpStage4StorageBatchOperationsJob(project, "job-1", "RUNNING"),
			gcpStage4StorageBatchOperationsJob(project, "job-succeeded", "SUCCEEDED"),
			gcpStage4StorageBatchOperationsJob(project, "job-canceled", "CANCELED"),
		}
		if filterState != "" {
			filtered := make([]*storagebatchoperationspb.Job, 0, len(items))
			for _, item := range items {
				if strings.EqualFold(item.GetState().String(), filterState) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		gcpStage4StorageBatchOperationsSortJobs(items, orderBy)
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&storagebatchoperationspb.ListJobsResponse{
			Jobs:          items[start:end],
			NextPageToken: next,
			Unreachable:   []string{},
		})
	case gcpStorageBatchOperationsGetJobMethod:
		req := &storagebatchoperationspb.GetJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, jobID, ok := parseGCPStorageBatchOperationsJobName(strings.TrimSpace(req.GetName()))
		if !ok || location != "global" {
			return grpcInvalidArgument("name-required")
		}
		if !isGCPStorageBatchOperationsJobID(jobID) {
			return grpcInvalidArgument("name-invalid")
		}
		if isGCPStorageBatchOperationsMissingID(jobID) {
			return grpcNotFound("job-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageBatchOperationsJob(project, jobID, gcpStorageBatchOperationsStateForJobID(jobID)))
	case gcpStorageBatchOperationsCreateJobMethod:
		req := &storagebatchoperationspb.CreateJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStorageBatchOperationsParent(strings.TrimSpace(req.GetParent()))
		if !ok || location != "global" {
			return grpcInvalidArgument("parent-required")
		}
		jobID := strings.TrimSpace(req.GetJobId())
		if jobID == "" {
			return grpcInvalidArgument("job_id-required")
		}
		if !isGCPStorageBatchOperationsJobID(jobID) {
			return grpcInvalidArgument("job_id-invalid")
		}
		if isGCPStorageBatchOperationsMissingID(jobID) || strings.Contains(strings.ToLower(jobID), "existing") {
			return grpcAlreadyExists("job-already-exists")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageBatchOperationsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		job := req.GetJob()
		if job == nil {
			return grpcInvalidArgument("job-required")
		}
		if reason := gcpStage4StorageBatchOperationsValidateJob(job); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if name := strings.TrimSpace(job.GetName()); name != "" {
			p, l, id, parsed := parseGCPStorageBatchOperationsJobName(name)
			if !parsed || l != "global" || p != project || id != jobID {
				return grpcInvalidArgument("job.name-must-match-parent-and-job_id")
			}
		}
		created := gcpStage4StorageBatchOperationsJobFromRequest(project, jobID, "RUNNING", job)
		return grpcProtoSuccess(gcpStage4StorageBatchOperationsOperation(project, "createJob."+jobID, created, false, false))
	case gcpStorageBatchOperationsDeleteJobMethod:
		req := &storagebatchoperationspb.DeleteJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, location, jobID, ok := parseGCPStorageBatchOperationsJobName(strings.TrimSpace(req.GetName()))
		if !ok || location != "global" {
			return grpcInvalidArgument("name-required")
		}
		if !isGCPStorageBatchOperationsJobID(jobID) {
			return grpcInvalidArgument("name-invalid")
		}
		if isGCPStorageBatchOperationsMissingID(jobID) {
			return grpcNotFound("job-not-found")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageBatchOperationsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpStorageBatchOperationsCancelJobMethod:
		req := &storagebatchoperationspb.CancelJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, location, jobID, ok := parseGCPStorageBatchOperationsJobName(strings.TrimSpace(req.GetName()))
		if !ok || location != "global" {
			return grpcInvalidArgument("name-required")
		}
		if !isGCPStorageBatchOperationsJobID(jobID) {
			return grpcInvalidArgument("name-invalid")
		}
		if isGCPStorageBatchOperationsMissingID(jobID) {
			return grpcNotFound("job-not-found")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageBatchOperationsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		state := gcpStorageBatchOperationsStateForJobID(jobID)
		if state == "SUCCEEDED" || state == "FAILED" || state == "CANCELED" {
			return grpcFailedPrecondition("job-terminal-state")
		}
		return grpcProtoSuccess(&storagebatchoperationspb.CancelJobResponse{})
	case gcpStorageBatchOperationsListBucketOperationsMethod:
		req := &storagebatchoperationspb.ListBucketOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, jobID, ok := parseGCPStorageBatchOperationsJobName(strings.TrimSpace(req.GetParent()))
		if !ok || location != "global" {
			return grpcInvalidArgument("parent-required")
		}
		if !isGCPStorageBatchOperationsJobID(jobID) {
			return grpcInvalidArgument("parent-invalid")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		orderBy := normalizeGCPStorageBatchOperationsOrderBy(req.GetOrderBy())
		if !isGCPStorageBatchOperationsOrderBy(orderBy) {
			return grpcInvalidArgument("order_by-invalid")
		}
		filterState, filterValid := parseGCPStorageBatchOperationsStateFilter(req.GetFilter())
		if !filterValid {
			return grpcInvalidArgument("filter-invalid")
		}
		items := []*storagebatchoperationspb.BucketOperation{
			gcpStage4StorageBatchOperationsBucketOperation(project, jobID, "bucket-op-1", "RUNNING"),
			gcpStage4StorageBatchOperationsBucketOperation(project, jobID, "bucket-op-2", "SUCCEEDED"),
		}
		if filterState != "" {
			filtered := make([]*storagebatchoperationspb.BucketOperation, 0, len(items))
			for _, item := range items {
				if strings.EqualFold(item.GetState().String(), filterState) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		gcpStage4StorageBatchOperationsSortBucketOperations(items, orderBy)
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&storagebatchoperationspb.ListBucketOperationsResponse{
			BucketOperations: items[start:end],
			NextPageToken:    next,
			Unreachable:      []string{},
		})
	case gcpStorageBatchOperationsGetBucketOperationMethod:
		req := &storagebatchoperationspb.GetBucketOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, jobID, bucketOperationID, ok := parseGCPStorageBatchOperationsBucketOperationName(strings.TrimSpace(req.GetName()))
		if !ok || location != "global" {
			return grpcInvalidArgument("name-required")
		}
		if !isGCPStorageBatchOperationsJobID(jobID) {
			return grpcInvalidArgument("name-invalid")
		}
		if isGCPStorageBatchOperationsMissingID(jobID) || isGCPStorageBatchOperationsMissingID(bucketOperationID) {
			return grpcNotFound("bucket_operation-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageBatchOperationsBucketOperation(project, jobID, bucketOperationID, "SUCCEEDED"))
	default:
		return nil, "", "", false
	}
}

func gcpStage4GRPCStorageInsights(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpStorageInsightsListReportConfigsMethod:
		req := &storageinsightspb.ListReportConfigsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStorageInsightsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		orderBy := normalizeGCPStorageInsightsOrderBy(req.GetOrderBy())
		if !isGCPStorageInsightsOrderBy(orderBy) {
			return grpcInvalidArgument("order_by-invalid")
		}
		if !isGCPStorageInsightsListFilter(req.GetFilter()) {
			return grpcInvalidArgument("filter-invalid")
		}
		items := []map[string]any{
			gcpStorageInsightsReportConfig(project, location, "reportconfig1"),
			gcpStorageInsightsReportConfig(project, location, "reportconfig2"),
		}
		sortGCPStorageInsightsItems(items, orderBy)
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		out := make([]*storageinsightspb.ReportConfig, 0, end-start)
		for _, item := range items[start:end] {
			out = append(out, gcpStage4StorageInsightsReportConfigFromMap(item))
		}
		return grpcProtoSuccess(&storageinsightspb.ListReportConfigsResponse{
			ReportConfigs: out,
			NextPageToken: next,
			Unreachable:   []string{},
		})
	case gcpStorageInsightsGetReportConfigMethod:
		req := &storageinsightspb.GetReportConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, reportConfigID, ok := parseGCPStorageInsightsReportConfigName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(reportConfigID) {
			return grpcNotFound("report_config-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageInsightsReportConfigFromMap(gcpStorageInsightsReportConfig(project, location, reportConfigID)))
	case gcpStorageInsightsCreateReportConfigMethod:
		req := &storageinsightspb.CreateReportConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStorageInsightsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		reportConfig := req.GetReportConfig()
		if reportConfig == nil {
			return grpcInvalidArgument("report_config-required")
		}
		reportConfigMap := gcpStage4StorageInsightsProtoToMap(reportConfig)
		if reason := gcpStage4StorageInsightsValidateReportConfig(reportConfigMap, false); reason != "" {
			return grpcInvalidArgument(reason)
		}
		reportConfigID := "reportconfig1"
		if providedName := strings.TrimSpace(gcpStorageInsightsString(reportConfigMap, "name")); providedName != "" {
			p, l, id, parsed := parseGCPStorageInsightsReportConfigName(providedName)
			if !parsed || p != project || l != location {
				return grpcInvalidArgument("report_config.name-must-match-parent")
			}
			reportConfigID = id
		}
		if strings.Contains(strings.ToLower(reportConfigID), "existing") {
			return grpcAlreadyExists("report_config-already-exists")
		}
		created := gcpStorageInsightsReportConfig(project, location, reportConfigID)
		applyGCPStorageInsightsReportConfigOverrides(created, reportConfigMap)
		return grpcProtoSuccess(gcpStage4StorageInsightsReportConfigFromMap(created))
	case gcpStorageInsightsUpdateReportConfigMethod:
		req := &storageinsightspb.UpdateReportConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		reportConfig := req.GetReportConfig()
		if reportConfig == nil {
			return grpcInvalidArgument("report_config-required")
		}
		reportConfigMap := gcpStage4StorageInsightsProtoToMap(reportConfig)
		project, location, reportConfigID, ok := parseGCPStorageInsightsReportConfigName(strings.TrimSpace(gcpStorageInsightsString(reportConfigMap, "name")))
		if !ok {
			return grpcInvalidArgument("report_config.name-required")
		}
		if isGCPStorageInsightsMissingID(reportConfigID) {
			return grpcNotFound("report_config-not-found")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		if mask := req.GetUpdateMask(); mask != nil && len(mask.GetPaths()) > 0 {
			if !validateGCPStorageInsightsUpdateMask(strings.Join(mask.GetPaths(), ","), []string{
				"display_name", "displayName", "labels", "frequency_options", "frequencyOptions", "csv_options", "csvOptions", "parquet_options", "parquetOptions", "object_metadata_report_options", "objectMetadataReportOptions",
			}) {
				return grpcInvalidArgument("update_mask-invalid")
			}
		}
		if reason := gcpStage4StorageInsightsValidateReportConfig(reportConfigMap, true); reason != "" {
			return grpcInvalidArgument(reason)
		}
		expectedName := gcpStorageInsightsReportConfigName(project, location, reportConfigID)
		if strings.TrimSpace(gcpStorageInsightsString(reportConfigMap, "name")) != expectedName {
			return grpcInvalidArgument("report_config.name-must-match-requested-resource")
		}
		updated := gcpStorageInsightsReportConfig(project, location, reportConfigID)
		applyGCPStorageInsightsReportConfigOverrides(updated, reportConfigMap)
		return grpcProtoSuccess(gcpStage4StorageInsightsReportConfigFromMap(updated))
	case gcpStorageInsightsDeleteReportConfigMethod:
		req := &storageinsightspb.DeleteReportConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, _, reportConfigID, ok := parseGCPStorageInsightsReportConfigName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(reportConfigID) {
			return grpcNotFound("report_config-not-found")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpStorageInsightsListReportDetailsMethod:
		req := &storageinsightspb.ListReportDetailsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, reportConfigID, ok := parseGCPStorageInsightsReportConfigName(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPStorageInsightsMissingID(reportConfigID) {
			return grpcNotFound("report_config-not-found")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		orderBy := normalizeGCPStorageInsightsOrderBy(req.GetOrderBy())
		if !isGCPStorageInsightsOrderBy(orderBy) {
			return grpcInvalidArgument("order_by-invalid")
		}
		if !isGCPStorageInsightsListFilter(req.GetFilter()) {
			return grpcInvalidArgument("filter-invalid")
		}
		items := []map[string]any{
			gcpStorageInsightsReportDetail(project, location, reportConfigID, "reportdetail1"),
			gcpStorageInsightsReportDetail(project, location, reportConfigID, "reportdetail2"),
		}
		sortGCPStorageInsightsItems(items, orderBy)
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		out := make([]*storageinsightspb.ReportDetail, 0, end-start)
		for _, item := range items[start:end] {
			out = append(out, gcpStage4StorageInsightsReportDetailFromMap(item))
		}
		return grpcProtoSuccess(&storageinsightspb.ListReportDetailsResponse{
			ReportDetails: out,
			NextPageToken: next,
			Unreachable:   []string{},
		})
	case gcpStorageInsightsGetReportDetailMethod:
		req := &storageinsightspb.GetReportDetailRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, reportConfigID, reportDetailID, ok := parseGCPStorageInsightsReportDetailName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(reportConfigID) || isGCPStorageInsightsMissingID(reportDetailID) {
			return grpcNotFound("report_detail-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageInsightsReportDetailFromMap(gcpStorageInsightsReportDetail(project, location, reportConfigID, reportDetailID)))
	case gcpStorageInsightsListDatasetConfigsMethod:
		req := &storageinsightspb.ListDatasetConfigsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStorageInsightsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		orderBy := normalizeGCPStorageInsightsOrderBy(req.GetOrderBy())
		if !isGCPStorageInsightsOrderBy(orderBy) {
			return grpcInvalidArgument("order_by-invalid")
		}
		if !isGCPStorageInsightsListFilter(req.GetFilter()) {
			return grpcInvalidArgument("filter-invalid")
		}
		items := []map[string]any{
			gcpStorageInsightsDatasetConfig(project, location, "datasetconfig1"),
			gcpStorageInsightsDatasetConfig(project, location, "datasetconfig2"),
		}
		sortGCPStorageInsightsItems(items, orderBy)
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		out := make([]*storageinsightspb.DatasetConfig, 0, end-start)
		for _, item := range items[start:end] {
			out = append(out, gcpStage4StorageInsightsDatasetConfigFromMap(item))
		}
		return grpcProtoSuccess(&storageinsightspb.ListDatasetConfigsResponse{
			DatasetConfigs: out,
			NextPageToken:  next,
			Unreachable:    []string{},
		})
	case gcpStorageInsightsGetDatasetConfigMethod:
		req := &storageinsightspb.GetDatasetConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(datasetConfigID) {
			return grpcNotFound("dataset_config-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageInsightsDatasetConfigFromMap(gcpStorageInsightsDatasetConfig(project, location, datasetConfigID)))
	case gcpStorageInsightsCreateDatasetConfigMethod:
		req := &storageinsightspb.CreateDatasetConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPStorageInsightsParent(strings.TrimSpace(req.GetParent()))
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		datasetConfigID := strings.TrimSpace(req.GetDatasetConfigId())
		if datasetConfigID == "" {
			return grpcInvalidArgument("dataset_config_id-required")
		}
		if !isGCPStorageInsightsDatasetConfigID(datasetConfigID) {
			return grpcInvalidArgument("dataset_config_id-invalid")
		}
		if strings.Contains(strings.ToLower(datasetConfigID), "existing") {
			return grpcAlreadyExists("dataset_config-already-exists")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		datasetConfig := req.GetDatasetConfig()
		if datasetConfig == nil {
			return grpcInvalidArgument("dataset_config-required")
		}
		datasetConfigMap := gcpStage4StorageInsightsProtoToMap(datasetConfig)
		if reason := gcpStage4StorageInsightsValidateDatasetConfig(datasetConfigMap, false); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if providedName := strings.TrimSpace(gcpStorageInsightsString(datasetConfigMap, "name")); providedName != "" {
			p, l, id, parsed := parseGCPStorageInsightsDatasetConfigName(providedName)
			if !parsed || p != project || l != location || id != datasetConfigID {
				return grpcInvalidArgument("dataset_config.name-must-match-parent-and-dataset_config_id")
			}
		}
		created := gcpStorageInsightsDatasetConfig(project, location, datasetConfigID)
		applyGCPStorageInsightsDatasetConfigOverrides(created, datasetConfigMap)
		return grpcProtoSuccess(gcpStage4StorageInsightsOperationFromMap(gcpStorageInsightsOperationForAction(project, location, "createDatasetConfig."+datasetConfigID, created, false)))
	case gcpStorageInsightsUpdateDatasetConfigMethod:
		req := &storageinsightspb.UpdateDatasetConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		if !validateGCPStorageInsightsUpdateMask(strings.Join(req.GetUpdateMask().GetPaths(), ","), []string{
			"description", "labels", "retention_period_days", "retentionPeriodDays", "include_newly_created_buckets", "includeNewlyCreatedBuckets", "identity.type", "identity_type",
		}) {
			return grpcInvalidArgument("update_mask-invalid")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		datasetConfig := req.GetDatasetConfig()
		if datasetConfig == nil {
			return grpcInvalidArgument("dataset_config-required")
		}
		datasetConfigMap := gcpStage4StorageInsightsProtoToMap(datasetConfig)
		project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigName(strings.TrimSpace(gcpStorageInsightsString(datasetConfigMap, "name")))
		if !ok {
			return grpcInvalidArgument("dataset_config.name-required")
		}
		if isGCPStorageInsightsMissingID(datasetConfigID) {
			return grpcNotFound("dataset_config-not-found")
		}
		if reason := gcpStage4StorageInsightsValidateDatasetConfig(datasetConfigMap, true); reason != "" {
			return grpcInvalidArgument(reason)
		}
		expectedName := gcpStorageInsightsDatasetConfigName(project, location, datasetConfigID)
		if strings.TrimSpace(gcpStorageInsightsString(datasetConfigMap, "name")) != expectedName {
			return grpcInvalidArgument("dataset_config.name-must-match-requested-resource")
		}
		updated := gcpStorageInsightsDatasetConfig(project, location, datasetConfigID)
		applyGCPStorageInsightsDatasetConfigOverrides(updated, datasetConfigMap)
		return grpcProtoSuccess(gcpStage4StorageInsightsOperationFromMap(gcpStorageInsightsOperationForAction(project, location, "updateDatasetConfig."+datasetConfigID, updated, false)))
	case gcpStorageInsightsDeleteDatasetConfigMethod:
		req := &storageinsightspb.DeleteDatasetConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(datasetConfigID) {
			return grpcNotFound("dataset_config-not-found")
		}
		if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPStorageInsightsRequestID(requestID) {
			return grpcInvalidArgument("request_id-invalid")
		}
		return grpcProtoSuccess(gcpStage4StorageInsightsOperationFromMap(gcpStorageInsightsOperationForAction(project, location, "deleteDatasetConfig."+datasetConfigID, nil, false)))
	case gcpStorageInsightsLinkDatasetMethod:
		req := &storageinsightspb.LinkDatasetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(datasetConfigID) {
			return grpcNotFound("dataset_config-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageInsightsOperationFromMap(gcpStorageInsightsOperationForAction(project, location, "linkDataset."+datasetConfigID, nil, false)))
	case gcpStorageInsightsUnlinkDatasetMethod:
		req := &storageinsightspb.UnlinkDatasetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, datasetConfigID, ok := parseGCPStorageInsightsDatasetConfigName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageInsightsMissingID(datasetConfigID) {
			return grpcNotFound("dataset_config-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageInsightsOperationFromMap(gcpStorageInsightsOperationForAction(project, location, "unlinkDataset."+datasetConfigID, nil, false)))
	default:
		return nil, "", "", false
	}
}

func gcpStage4StorageInsightsReportConfigFromMap(source map[string]any) *storageinsightspb.ReportConfig {
	out := &storageinsightspb.ReportConfig{}
	if !gcpStage4StorageInsightsMapToProto(source, out) {
		return &storageinsightspb.ReportConfig{}
	}
	return out
}

func gcpStage4StorageInsightsReportDetailFromMap(source map[string]any) *storageinsightspb.ReportDetail {
	out := &storageinsightspb.ReportDetail{}
	if !gcpStage4StorageInsightsMapToProto(source, out) {
		return &storageinsightspb.ReportDetail{}
	}
	return out
}

func gcpStage4StorageInsightsDatasetConfigFromMap(source map[string]any) *storageinsightspb.DatasetConfig {
	out := &storageinsightspb.DatasetConfig{}
	if !gcpStage4StorageInsightsMapToProto(source, out) {
		return &storageinsightspb.DatasetConfig{}
	}
	return out
}

func gcpStage4StorageInsightsOperationFromMap(source map[string]any) *longrunningpb.Operation {
	out := &longrunningpb.Operation{}
	if !gcpStage4StorageInsightsMapToProto(source, out) {
		return &longrunningpb.Operation{}
	}
	return out
}

func gcpStage4StorageInsightsProtoToMap(message proto.Message) map[string]any {
	if message == nil {
		return map[string]any{}
	}
	payload, err := protojson.Marshal(message)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func gcpStage4StorageInsightsMapToProto(source map[string]any, message proto.Message) bool {
	if message == nil || source == nil {
		return false
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, message); err != nil {
		return false
	}
	return true
}

func gcpStage4StorageInsightsValidateReportConfig(reportConfig map[string]any, requireName bool) string {
	if requireName && strings.TrimSpace(gcpStorageInsightsString(reportConfig, "name")) == "" {
		return "report_config.name-required"
	}
	if displayName := strings.TrimSpace(gcpStorageInsightsString(reportConfig, "displayName")); len(displayName) > 256 {
		return "report_config.display_name-too-long"
	}
	if _, ok := reportConfig["frequencyOptions"].(map[string]any); !ok {
		return "report_config.frequency_options-required"
	}
	reportFormatCount := 0
	if _, ok := reportConfig["csvOptions"].(map[string]any); ok {
		reportFormatCount++
	}
	if _, ok := reportConfig["parquetOptions"].(map[string]any); ok {
		reportFormatCount++
	}
	if reportFormatCount == 0 {
		return "report_config.report_format-required"
	}
	if reportFormatCount > 1 {
		return "report_config.report_format-oneof-invalid"
	}
	objectOptions, ok := reportConfig["objectMetadataReportOptions"].(map[string]any)
	if !ok || len(objectOptions) == 0 {
		return "report_config.object_metadata_report_options-required"
	}
	if fields, ok := objectOptions["metadataFields"].([]any); !ok || len(fields) == 0 {
		return "report_config.object_metadata_report_options.metadata_fields-required"
	}
	return ""
}

func gcpStage4StorageInsightsValidateDatasetConfig(datasetConfig map[string]any, requireName bool) string {
	if requireName && strings.TrimSpace(gcpStorageInsightsString(datasetConfig, "name")) == "" {
		return "dataset_config.name-required"
	}
	if description := strings.TrimSpace(gcpStorageInsightsString(datasetConfig, "description")); len(description) > 256 {
		return "dataset_config.description-too-long"
	}
	if retentionRaw, ok := datasetConfig["retentionPeriodDays"]; ok {
		if n, ok := asInt64GCPStorageInsights(retentionRaw); !ok || n < 0 {
			return "dataset_config.retention_period_days-invalid"
		}
	}
	sourceFields := 0
	for _, key := range []string{"sourceProjects", "sourceFolders", "organizationScope", "cloudStorageObjectPath"} {
		if _, ok := datasetConfig[key]; ok {
			sourceFields++
		}
	}
	if sourceFields == 0 {
		return "dataset_config.source_option-required"
	}
	if sourceFields > 1 {
		return "dataset_config.source_option-oneof-invalid"
	}
	locationFields := 0
	for _, key := range []string{"includeCloudStorageLocations", "excludeCloudStorageLocations"} {
		if _, ok := datasetConfig[key]; ok {
			locationFields++
		}
	}
	if locationFields > 1 {
		return "dataset_config.cloud_storage_locations-oneof-invalid"
	}
	bucketFields := 0
	for _, key := range []string{"includeCloudStorageBuckets", "excludeCloudStorageBuckets"} {
		if _, ok := datasetConfig[key]; ok {
			bucketFields++
		}
	}
	if bucketFields > 1 {
		return "dataset_config.cloud_storage_buckets-oneof-invalid"
	}
	return ""
}

func gcpStage4GRPCStorageTransfer(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpStorageTransferGetGoogleServiceAccountMethod:
		req := &storagetransferpb.GetGoogleServiceAccountRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		return grpcProtoSuccess(&storagetransferpb.GoogleServiceAccount{
			AccountEmail: fmt.Sprintf("project-%s@storage-transfer-service.iam.gserviceaccount.com", projectID),
			SubjectId:    fmt.Sprintf("subject-%s", projectID),
		})
	case gcpStorageTransferCreateTransferJobMethod:
		req := &storagetransferpb.CreateTransferJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		transferJob := req.GetTransferJob()
		if transferJob == nil {
			return grpcInvalidArgument("transfer_job-required")
		}
		projectID := strings.TrimSpace(transferJob.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("transfer_job.project_id-required")
		}
		if transferJob.GetTransferSpec() == nil {
			return grpcInvalidArgument("transfer_job.transfer_spec-required")
		}
		jobID := "job-1"
		if providedName := strings.TrimSpace(transferJob.GetName()); providedName != "" {
			parsedJobID, ok := parseGCPStorageTransferJobResourceName(providedName)
			if !ok {
				return grpcInvalidArgument("transfer_job.name-invalid")
			}
			jobID = parsedJobID
		}
		if !isGCPStorageTransferJobID(jobID) {
			return grpcInvalidArgument("transfer_job.name-invalid")
		}
		if isGCPStorageTransferMissingID(jobID) || strings.Contains(strings.ToLower(jobID), "existing") {
			return grpcAlreadyExists("transfer_job-already-exists")
		}
		status := transferJob.GetStatus().String()
		if transferJob.GetStatus() == storagetransferpb.TransferJob_STATUS_UNSPECIFIED {
			status = "ENABLED"
		}
		created := gcpStage4StorageTransferJob(projectID, jobID, status)
		if description := strings.TrimSpace(transferJob.GetDescription()); description != "" {
			created.Description = description
		}
		if transferJob.GetTransferSpec() != nil {
			created.TransferSpec = transferJob.GetTransferSpec()
		}
		return grpcProtoSuccess(created)
	case gcpStorageTransferUpdateTransferJobMethod:
		req := &storagetransferpb.UpdateTransferJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		jobID, ok := parseGCPStorageTransferJobResourceName(strings.TrimSpace(req.GetJobName()))
		if !ok {
			return grpcInvalidArgument("job_name-required")
		}
		if isGCPStorageTransferMissingID(jobID) {
			return grpcNotFound("transfer_job-not-found")
		}
		if !validateGCPStorageTransferTransferJobUpdateMask(strings.Join(req.GetUpdateTransferJobFieldMask().GetPaths(), ",")) {
			return grpcInvalidArgument("update_transfer_job_field_mask-invalid")
		}
		transferJob := req.GetTransferJob()
		if transferJob == nil {
			return grpcInvalidArgument("transfer_job-required")
		}
		if strings.TrimSpace(transferJob.GetName()) != gcpStorageTransferTransferJobName(jobID) {
			return grpcInvalidArgument("transfer_job.name-mismatch")
		}
		if transferSpec := transferJob.GetTransferSpec(); transferSpec != nil {
			if transferSpec.GetDataSource() == nil && transferSpec.GetDataSink() == nil {
				return grpcInvalidArgument("transfer_job.transfer_spec-invalid")
			}
		}
		updated := gcpStage4StorageTransferJob(projectID, jobID, gcpStorageTransferJobStatusFromID(jobID))
		if description := strings.TrimSpace(transferJob.GetDescription()); description != "" {
			updated.Description = description
		}
		if transferJob.GetStatus() != storagetransferpb.TransferJob_STATUS_UNSPECIFIED {
			updated.Status = transferJob.GetStatus()
		}
		if transferJob.GetTransferSpec() != nil {
			updated.TransferSpec = transferJob.GetTransferSpec()
		}
		return grpcProtoSuccess(updated)
	case gcpStorageTransferGetTransferJobMethod:
		req := &storagetransferpb.GetTransferJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		jobID, ok := parseGCPStorageTransferJobResourceName(strings.TrimSpace(req.GetJobName()))
		if !ok {
			return grpcInvalidArgument("job_name-required")
		}
		if isGCPStorageTransferMissingID(jobID) {
			return grpcNotFound("transfer_job-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageTransferJob(projectID, jobID, gcpStorageTransferJobStatusFromID(jobID)))
	case gcpStorageTransferListTransferJobsMethod:
		req := &storagetransferpb.ListTransferJobsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 256 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		filter, err := parseGCPStorageTransferFilterJSON(strings.TrimSpace(req.GetFilter()))
		if err != nil {
			return grpcInvalidArgument("filter-invalid")
		}
		projectID := strings.TrimSpace(gcpStorageTransferString(filter, "projectId"))
		if projectID == "" {
			projectID = strings.TrimSpace(gcpStorageTransferString(filter, "project_id"))
		}
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("filter.project_id-required")
		}
		jobNames := gcpStorageTransferAnyStringSet(filter["jobNames"])
		if len(jobNames) == 0 {
			jobNames = gcpStorageTransferAnyStringSet(filter["job_names"])
		}
		jobStatuses := gcpStorageTransferAnyStringSet(filter["jobStatuses"])
		if len(jobStatuses) == 0 {
			jobStatuses = gcpStorageTransferAnyStringSet(filter["job_statuses"])
		}

		items := []*storagetransferpb.TransferJob{
			gcpStage4StorageTransferJob(projectID, "job-1", "ENABLED"),
			gcpStage4StorageTransferJob(projectID, "job-disabled", "DISABLED"),
		}
		if len(jobNames) > 0 {
			filtered := make([]*storagetransferpb.TransferJob, 0, len(items))
			for _, item := range items {
				name := strings.TrimSpace(item.GetName())
				jobID := strings.TrimPrefix(name, "transferJobs/")
				if _, ok := jobNames[name]; ok {
					filtered = append(filtered, item)
					continue
				}
				if _, ok := jobNames[jobID]; ok {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		if len(jobStatuses) > 0 {
			filtered := make([]*storagetransferpb.TransferJob, 0, len(items))
			for _, item := range items {
				status := strings.ToUpper(strings.TrimSpace(item.GetStatus().String()))
				if _, ok := jobStatuses[status]; ok {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].GetName() < items[j].GetName()
		})
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&storagetransferpb.ListTransferJobsResponse{
			TransferJobs:  items[start:end],
			NextPageToken: next,
		})
	case gcpStorageTransferPauseTransferOperationMethod:
		req := &storagetransferpb.PauseTransferOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		operationID, ok := parseGCPStage4StorageTransferOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageTransferMissingID(operationID) {
			return grpcNotFound("transfer_operation-not-found")
		}
		if strings.Contains(strings.ToLower(operationID), "paused") {
			return grpcFailedPrecondition("transfer_operation-already-paused")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpStorageTransferResumeTransferOperationMethod:
		req := &storagetransferpb.ResumeTransferOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		operationID, ok := parseGCPStage4StorageTransferOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageTransferMissingID(operationID) {
			return grpcNotFound("transfer_operation-not-found")
		}
		if strings.Contains(strings.ToLower(operationID), "running") || strings.Contains(strings.ToLower(operationID), "active") {
			return grpcFailedPrecondition("transfer_operation-not-paused")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpStorageTransferRunTransferJobMethod:
		req := &storagetransferpb.RunTransferJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		jobID, ok := parseGCPStorageTransferJobResourceName(strings.TrimSpace(req.GetJobName()))
		if !ok {
			return grpcInvalidArgument("job_name-required")
		}
		if isGCPStorageTransferMissingID(jobID) {
			return grpcNotFound("transfer_job-not-found")
		}
		if strings.Contains(strings.ToLower(jobID), "running") {
			return grpcFailedPrecondition("transfer_job-active-run-conflict")
		}
		return grpcProtoSuccess(gcpStage4StorageTransferOperation(projectID, "run."+jobID, jobID, "IN_PROGRESS", false))
	case gcpStorageTransferDeleteTransferJobMethod:
		req := &storagetransferpb.DeleteTransferJobRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		jobID, ok := parseGCPStorageTransferJobResourceName(strings.TrimSpace(req.GetJobName()))
		if !ok {
			return grpcInvalidArgument("job_name-required")
		}
		if isGCPStorageTransferMissingID(jobID) {
			return grpcNotFound("transfer_job-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpStorageTransferCreateAgentPoolMethod:
		req := &storagetransferpb.CreateAgentPoolRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		agentPoolID := strings.TrimSpace(req.GetAgentPoolId())
		if !isGCPStorageTransferAgentPoolID(agentPoolID) {
			return grpcInvalidArgument("agent_pool_id-required")
		}
		if isGCPStorageTransferMissingID(agentPoolID) || strings.Contains(strings.ToLower(agentPoolID), "existing") {
			return grpcAlreadyExists("agent_pool-already-exists")
		}
		agentPool := req.GetAgentPool()
		if agentPool == nil {
			return grpcInvalidArgument("agent_pool-required")
		}
		expectedName := gcpStorageTransferAgentPoolName(projectID, agentPoolID)
		if got := strings.TrimSpace(agentPool.GetName()); got != "" && got != expectedName {
			return grpcInvalidArgument("agent_pool.name-mismatch")
		}
		created := gcpStage4StorageTransferAgentPool(projectID, agentPoolID, "CREATED")
		if displayName := strings.TrimSpace(agentPool.GetDisplayName()); displayName != "" {
			created.DisplayName = displayName
		}
		if agentPool.GetBandwidthLimit() != nil {
			created.BandwidthLimit = agentPool.GetBandwidthLimit()
		}
		return grpcProtoSuccess(created)
	case gcpStorageTransferUpdateAgentPoolMethod:
		req := &storagetransferpb.UpdateAgentPoolRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if !validateGCPStorageTransferAgentPoolUpdateMask(strings.Join(req.GetUpdateMask().GetPaths(), ",")) {
			return grpcInvalidArgument("update_mask-invalid")
		}
		agentPool := req.GetAgentPool()
		if agentPool == nil {
			return grpcInvalidArgument("agent_pool-required")
		}
		projectID, agentPoolID, ok := parseGCPStorageTransferAgentPoolResourceName(strings.TrimSpace(agentPool.GetName()))
		if !ok {
			return grpcInvalidArgument("agent_pool.name-required")
		}
		if isGCPStorageTransferMissingID(agentPoolID) {
			return grpcNotFound("agent_pool-not-found")
		}
		updated := gcpStage4StorageTransferAgentPool(projectID, agentPoolID, gcpStorageTransferAgentPoolStateFromID(agentPoolID))
		if displayName := strings.TrimSpace(agentPool.GetDisplayName()); displayName != "" {
			updated.DisplayName = displayName
		}
		if agentPool.GetBandwidthLimit() != nil {
			updated.BandwidthLimit = agentPool.GetBandwidthLimit()
		}
		return grpcProtoSuccess(updated)
	case gcpStorageTransferGetAgentPoolMethod:
		req := &storagetransferpb.GetAgentPoolRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID, agentPoolID, ok := parseGCPStorageTransferAgentPoolResourceName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageTransferMissingID(agentPoolID) {
			return grpcNotFound("agent_pool-not-found")
		}
		return grpcProtoSuccess(gcpStage4StorageTransferAgentPool(projectID, agentPoolID, gcpStorageTransferAgentPoolStateFromID(agentPoolID)))
	case gcpStorageTransferListAgentPoolsMethod:
		req := &storagetransferpb.ListAgentPoolsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		projectID := strings.TrimSpace(req.GetProjectId())
		if !isGCPStorageTransferProjectID(projectID) {
			return grpcInvalidArgument("project_id-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 256 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		filter := map[string]any{}
		if raw := strings.TrimSpace(req.GetFilter()); raw != "" {
			parsed, err := parseGCPStorageTransferFilterJSON(raw)
			if err != nil {
				return grpcInvalidArgument("filter-invalid")
			}
			filter = parsed
		}
		nameFilter := gcpStorageTransferAnyStringSet(filter["agentPoolNames"])
		if len(nameFilter) == 0 {
			nameFilter = gcpStorageTransferAnyStringSet(filter["agent_pool_names"])
		}

		items := []*storagetransferpb.AgentPool{
			gcpStage4StorageTransferAgentPool(projectID, "agentpool-1", "CREATED"),
			gcpStage4StorageTransferAgentPool(projectID, "agentpool-2", "CREATING"),
		}
		if len(nameFilter) > 0 {
			filtered := make([]*storagetransferpb.AgentPool, 0, len(items))
			for _, item := range items {
				name := strings.TrimSpace(item.GetName())
				agentPoolID := strings.TrimPrefix(name, fmt.Sprintf("projects/%s/agentPools/", projectID))
				if _, ok := nameFilter[name]; ok {
					filtered = append(filtered, item)
					continue
				}
				if _, ok := nameFilter[agentPoolID]; ok {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&storagetransferpb.ListAgentPoolsResponse{
			AgentPools:    items[start:end],
			NextPageToken: next,
		})
	case gcpStorageTransferDeleteAgentPoolMethod:
		req := &storagetransferpb.DeleteAgentPoolRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, agentPoolID, ok := parseGCPStorageTransferAgentPoolResourceName(strings.TrimSpace(req.GetName()))
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPStorageTransferMissingID(agentPoolID) {
			return grpcNotFound("agent_pool-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	default:
		return nil, "", "", false
	}
}

func gcpStage4StorageTransferJob(projectID, jobID, status string) *storagetransferpb.TransferJob {
	return &storagetransferpb.TransferJob{
		Name:                 gcpStorageTransferTransferJobName(jobID),
		Description:          "Stackyard Storage Transfer job " + jobID,
		ProjectId:            projectID,
		Status:               gcpStage4StorageTransferJobStatus(status),
		CreationTime:         timestamppb.New(gcpStorageTransferReferenceTime),
		LastModificationTime: timestamppb.New(gcpStorageTransferReferenceTime.Add(2 * time.Hour)),
		LatestOperationName:  "transferOperations/run." + jobID,
		TransferSpec: &storagetransferpb.TransferSpec{
			DataSource: &storagetransferpb.TransferSpec_GcsDataSource{
				GcsDataSource: &storagetransferpb.GcsData{
					BucketName: "stackyard-source-bucket",
				},
			},
			DataSink: &storagetransferpb.TransferSpec_GcsDataSink{
				GcsDataSink: &storagetransferpb.GcsData{
					BucketName: "stackyard-destination-bucket",
				},
			},
		},
	}
}

func gcpStage4StorageTransferAgentPool(projectID, agentPoolID, state string) *storagetransferpb.AgentPool {
	return &storagetransferpb.AgentPool{
		Name:        gcpStorageTransferAgentPoolName(projectID, agentPoolID),
		DisplayName: "Stackyard Agent Pool " + agentPoolID,
		State:       gcpStage4StorageTransferAgentPoolState(state),
		BandwidthLimit: &storagetransferpb.AgentPool_BandwidthLimit{
			LimitMbps: 100,
		},
	}
}

func gcpStage4StorageTransferOperation(projectID, operationID, jobID, status string, done bool) *longrunningpb.Operation {
	transferOperation := &storagetransferpb.TransferOperation{
		Name:            "transferOperations/" + operationID,
		ProjectId:       projectID,
		TransferJobName: gcpStorageTransferTransferJobName(jobID),
		Status:          gcpStage4StorageTransferOperationStatus(status),
		StartTime:       timestamppb.New(gcpStorageTransferReferenceTime.Add(5 * time.Minute)),
	}
	if done {
		transferOperation.EndTime = timestamppb.New(gcpStorageTransferReferenceTime.Add(35 * time.Minute))
	}
	metadataAny, err := anypb.New(transferOperation)
	if err != nil {
		return &longrunningpb.Operation{
			Name: "transferOperations/" + operationID,
			Done: done,
		}
	}
	operation := &longrunningpb.Operation{
		Name:     "transferOperations/" + operationID,
		Done:     done,
		Metadata: metadataAny,
	}
	if done {
		responseAny, err := anypb.New(&emptypb.Empty{})
		if err == nil {
			operation.Result = &longrunningpb.Operation_Response{Response: responseAny}
		}
	}
	return operation
}

func gcpStage4StorageTransferJobStatus(status string) storagetransferpb.TransferJob_Status {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ENABLED":
		return storagetransferpb.TransferJob_ENABLED
	case "DISABLED":
		return storagetransferpb.TransferJob_DISABLED
	case "DELETED":
		return storagetransferpb.TransferJob_DELETED
	default:
		return storagetransferpb.TransferJob_STATUS_UNSPECIFIED
	}
}

func gcpStage4StorageTransferAgentPoolState(state string) storagetransferpb.AgentPool_State {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "ACTIVE", "CREATED":
		return storagetransferpb.AgentPool_CREATED
	case "CREATING":
		return storagetransferpb.AgentPool_CREATING
	case "DELETING":
		return storagetransferpb.AgentPool_DELETING
	default:
		return storagetransferpb.AgentPool_STATE_UNSPECIFIED
	}
}

func gcpStage4StorageTransferOperationStatus(status string) storagetransferpb.TransferOperation_Status {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "IN_PROGRESS":
		return storagetransferpb.TransferOperation_IN_PROGRESS
	case "PAUSED":
		return storagetransferpb.TransferOperation_PAUSED
	case "SUCCESS":
		return storagetransferpb.TransferOperation_SUCCESS
	case "FAILED":
		return storagetransferpb.TransferOperation_FAILED
	case "ABORTED":
		return storagetransferpb.TransferOperation_ABORTED
	default:
		return storagetransferpb.TransferOperation_STATUS_UNSPECIFIED
	}
}

func parseGCPStage4StorageTransferOperationName(name string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 2 || parts[0] != "transferOperations" {
		return "", false
	}
	operationID = strings.TrimSpace(parts[1])
	if operationID == "" {
		return "", false
	}
	return operationID, true
}

func gcpStage4GRPCStreetViewPublish(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpStreetViewPublishStartUploadMethod:
		req := &emptypb.Empty{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		return grpcProtoSuccess(gcpStage4StreetViewPublishUploadRef("photo-upload-1"))
	case gcpStreetViewPublishCreatePhotoMethod:
		req := &publishpb.CreatePhotoRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		photo := req.GetPhoto()
		if photo == nil {
			return grpcInvalidArgument("photo-required")
		}
		photoID := strings.TrimSpace(photo.GetPhotoId().GetId())
		if photoID == "" {
			photoID = "photo-1"
		}
		if !isGCPStreetViewPublishPhotoID(photoID) {
			return grpcInvalidArgument("photo.photo_id.id-invalid")
		}
		if strings.Contains(strings.ToLower(photoID), "existing") {
			return grpcAlreadyExists("photo-already-exists")
		}
		uploadURL := strings.TrimSpace(photo.GetUploadReference().GetUploadUrl())
		if !isGCPStreetViewPublishUploadURL(uploadURL) {
			return grpcInvalidArgument("photo.upload_reference.upload_url-required")
		}
		if strings.Contains(strings.ToLower(uploadURL), "missing") {
			return grpcNotFound("upload_reference-not-found")
		}
		return grpcProtoSuccess(gcpStage4StreetViewPublishPhoto(photoID, false))
	case gcpStreetViewPublishGetPhotoMethod:
		req := &publishpb.GetPhotoRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		photoID := strings.TrimSpace(req.GetPhotoId())
		if !isGCPStreetViewPublishPhotoID(photoID) {
			return grpcInvalidArgument("photo_id-required")
		}
		if !isGCPStage4StreetViewPublishPhotoView(req.GetView()) {
			return grpcInvalidArgument("view-invalid")
		}
		if strings.Contains(strings.ToLower(photoID), "missing") {
			return grpcNotFound("photo-not-found")
		}
		return grpcProtoSuccess(gcpStage4StreetViewPublishPhoto(photoID, req.GetView() == publishpb.PhotoView_INCLUDE_DOWNLOAD_URL))
	case gcpStreetViewPublishBatchGetPhotosMethod:
		req := &publishpb.BatchGetPhotosRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if len(req.GetPhotoIds()) == 0 {
			return grpcInvalidArgument("photo_ids-required")
		}
		if !isGCPStage4StreetViewPublishPhotoView(req.GetView()) {
			return grpcInvalidArgument("view-invalid")
		}
		results := make([]*publishpb.PhotoResponse, 0, len(req.GetPhotoIds()))
		for _, rawID := range req.GetPhotoIds() {
			photoID := strings.TrimSpace(rawID)
			if !isGCPStreetViewPublishPhotoID(photoID) {
				return grpcInvalidArgument("photo_ids-invalid")
			}
			if strings.Contains(strings.ToLower(photoID), "missing") {
				results = append(results, &publishpb.PhotoResponse{
					Status: gcpStage4StreetViewPublishStatus(5, "photo not found"),
				})
				continue
			}
			results = append(results, &publishpb.PhotoResponse{
				Status: gcpStage4StreetViewPublishStatus(0, "OK"),
				Photo:  gcpStage4StreetViewPublishPhoto(photoID, req.GetView() == publishpb.PhotoView_INCLUDE_DOWNLOAD_URL),
			})
		}
		return grpcProtoSuccess(&publishpb.BatchGetPhotosResponse{Results: results})
	case gcpStreetViewPublishListPhotosMethod:
		req := &publishpb.ListPhotosRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if !isGCPStage4StreetViewPublishPhotoView(req.GetView()) {
			return grpcInvalidArgument("view-invalid")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		filter, err := parseGCPStreetViewPublishFilter(req.GetFilter(), map[string]string{
			"placeId":       "string",
			"min_latitude":  "float",
			"max_latitude":  "float",
			"min_longitude": "float",
			"max_longitude": "float",
		})
		if err != nil {
			return grpcInvalidArgument("filter-invalid")
		}
		includeDownload := req.GetView() == publishpb.PhotoView_INCLUDE_DOWNLOAD_URL
		items := []*publishpb.Photo{
			gcpStage4StreetViewPublishPhoto("photo-1", includeDownload),
			gcpStage4StreetViewPublishPhoto("photo-2", includeDownload),
		}
		if placeID := strings.TrimSpace(filter["placeId"]); placeID != "" {
			filtered := make([]*publishpb.Photo, 0, len(items))
			for _, item := range items {
				for _, place := range item.GetPlaces() {
					if strings.EqualFold(strings.TrimSpace(place.GetPlaceId()), placeID) {
						filtered = append(filtered, item)
						break
					}
				}
			}
			items = filtered
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&publishpb.ListPhotosResponse{
			Photos:        items[start:end],
			NextPageToken: next,
		})
	case gcpStreetViewPublishUpdatePhotoMethod:
		req := &publishpb.UpdatePhotoRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		photo := req.GetPhoto()
		if photo == nil {
			return grpcInvalidArgument("photo-required")
		}
		photoID := strings.TrimSpace(photo.GetPhotoId().GetId())
		if !isGCPStreetViewPublishPhotoID(photoID) {
			return grpcInvalidArgument("photo.photo_id.id-required")
		}
		mask, err := parseGCPStreetViewPublishUpdateMask(strings.Join(req.GetUpdateMask().GetPaths(), ","))
		if err != nil {
			return grpcInvalidArgument("update_mask-invalid")
		}
		if hasGCPStreetViewPublishMask(mask, "pose.altitude") && photo.GetPose() != nil && photo.GetPose().GetLatLngPair() == nil {
			return grpcInvalidArgument("pose.altitude-requires-pose.lat_lng_pair")
		}
		if strings.Contains(strings.ToLower(photoID), "missing") {
			return grpcNotFound("photo-not-found")
		}
		updated := gcpStage4StreetViewPublishPhoto(photoID, false)
		if photo.GetPose() != nil {
			updated.Pose = photo.GetPose()
		}
		if len(photo.GetConnections()) > 0 {
			updated.Connections = photo.GetConnections()
		}
		if len(photo.GetPlaces()) > 0 {
			updated.Places = photo.GetPlaces()
		}
		return grpcProtoSuccess(updated)
	case gcpStreetViewPublishBatchUpdatePhotosMethod:
		req := &publishpb.BatchUpdatePhotosRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if len(req.GetUpdatePhotoRequests()) == 0 {
			return grpcInvalidArgument("update_photo_requests-required")
		}
		if len(req.GetUpdatePhotoRequests()) > 20 {
			return grpcInvalidArgument("update_photo_requests-too-many")
		}
		results := make([]*publishpb.PhotoResponse, 0, len(req.GetUpdatePhotoRequests()))
		for _, updateReq := range req.GetUpdatePhotoRequests() {
			photo := updateReq.GetPhoto()
			if photo == nil {
				results = append(results, &publishpb.PhotoResponse{
					Status: gcpStage4StreetViewPublishStatus(3, "photo is required"),
				})
				continue
			}
			photoID := strings.TrimSpace(photo.GetPhotoId().GetId())
			if !isGCPStreetViewPublishPhotoID(photoID) {
				results = append(results, &publishpb.PhotoResponse{
					Status: gcpStage4StreetViewPublishStatus(3, "photoId is invalid"),
				})
				continue
			}
			mask, err := parseGCPStreetViewPublishUpdateMask(strings.Join(updateReq.GetUpdateMask().GetPaths(), ","))
			if err != nil {
				results = append(results, &publishpb.PhotoResponse{
					Status: gcpStage4StreetViewPublishStatus(3, "updateMask is invalid"),
				})
				continue
			}
			if hasGCPStreetViewPublishMask(mask, "pose.altitude") && photo.GetPose() != nil && photo.GetPose().GetLatLngPair() == nil {
				results = append(results, &publishpb.PhotoResponse{
					Status: gcpStage4StreetViewPublishStatus(3, "pose.altitude requires pose.lat_lng_pair"),
				})
				continue
			}
			if strings.Contains(strings.ToLower(photoID), "missing") {
				results = append(results, &publishpb.PhotoResponse{
					Status: gcpStage4StreetViewPublishStatus(5, "photo not found"),
				})
				continue
			}
			updated := gcpStage4StreetViewPublishPhoto(photoID, false)
			if photo.GetPose() != nil {
				updated.Pose = photo.GetPose()
			}
			if len(photo.GetConnections()) > 0 {
				updated.Connections = photo.GetConnections()
			}
			if len(photo.GetPlaces()) > 0 {
				updated.Places = photo.GetPlaces()
			}
			results = append(results, &publishpb.PhotoResponse{
				Status: gcpStage4StreetViewPublishStatus(0, "OK"),
				Photo:  updated,
			})
		}
		return grpcProtoSuccess(&publishpb.BatchUpdatePhotosResponse{Results: results})
	case gcpStreetViewPublishDeletePhotoMethod:
		req := &publishpb.DeletePhotoRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		photoID := strings.TrimSpace(req.GetPhotoId())
		if !isGCPStreetViewPublishPhotoID(photoID) {
			return grpcInvalidArgument("photo_id-required")
		}
		if strings.Contains(strings.ToLower(photoID), "missing") {
			return grpcNotFound("photo-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpStreetViewPublishBatchDeletePhotosMethod:
		req := &publishpb.BatchDeletePhotosRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if len(req.GetPhotoIds()) == 0 {
			return grpcInvalidArgument("photo_ids-required")
		}
		statuses := make([]*statuspb.Status, 0, len(req.GetPhotoIds()))
		for _, rawID := range req.GetPhotoIds() {
			photoID := strings.TrimSpace(rawID)
			if !isGCPStreetViewPublishPhotoID(photoID) {
				return grpcInvalidArgument("photo_ids-invalid")
			}
			if strings.Contains(strings.ToLower(photoID), "missing") {
				statuses = append(statuses, gcpStage4StreetViewPublishStatus(5, "photo not found"))
				continue
			}
			statuses = append(statuses, gcpStage4StreetViewPublishStatus(0, "OK"))
		}
		return grpcProtoSuccess(&publishpb.BatchDeletePhotosResponse{Status: statuses})
	case gcpStreetViewPublishStartPhotoSequenceUploadMethod:
		req := &emptypb.Empty{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		return grpcProtoSuccess(gcpStage4StreetViewPublishUploadRef("photo-sequence-upload-1"))
	case gcpStreetViewPublishCreatePhotoSequenceMethod:
		req := &publishpb.CreatePhotoSequenceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		photoSequence := req.GetPhotoSequence()
		if photoSequence == nil {
			return grpcInvalidArgument("photo_sequence-required")
		}
		if !isGCPStage4StreetViewPublishInputType(req.GetInputType()) {
			return grpcInvalidArgument("input_type-required")
		}
		sequenceID := strings.TrimSpace(photoSequence.GetId())
		if sequenceID == "" {
			sequenceID = "sequence-1"
		}
		if !isGCPStreetViewPublishSequenceID(sequenceID) {
			return grpcInvalidArgument("photo_sequence.id-invalid")
		}
		if strings.Contains(strings.ToLower(sequenceID), "existing") {
			return grpcAlreadyExists("photo_sequence-already-exists")
		}
		uploadURL := strings.TrimSpace(photoSequence.GetUploadReference().GetUploadUrl())
		if !isGCPStreetViewPublishUploadURL(uploadURL) {
			return grpcInvalidArgument("photo_sequence.upload_reference.upload_url-required")
		}
		done := !isGCPStreetViewPublishProcessingID(sequenceID)
		return grpcProtoSuccess(gcpStage4StreetViewPublishOperation(gcpStreetViewPublishOperationID(sequenceID), sequenceID, done, req.GetInputType()))
	case gcpStreetViewPublishGetPhotoSequenceMethod:
		req := &publishpb.GetPhotoSequenceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		sequenceID := strings.TrimSpace(req.GetSequenceId())
		if !isGCPStreetViewPublishSequenceID(sequenceID) {
			return grpcInvalidArgument("sequence_id-required")
		}
		if strings.Contains(strings.ToLower(sequenceID), "missing") {
			return grpcNotFound("photo_sequence-not-found")
		}
		_, err := parseGCPStreetViewPublishFilter(req.GetFilter(), map[string]string{
			"published_status": "string",
		})
		if err != nil {
			return grpcInvalidArgument("filter-invalid")
		}
		done := !isGCPStreetViewPublishProcessingID(sequenceID)
		return grpcProtoSuccess(gcpStage4StreetViewPublishOperation(gcpStreetViewPublishOperationID(sequenceID), sequenceID, done, publishpb.CreatePhotoSequenceRequest_VIDEO))
	case gcpStreetViewPublishListPhotoSequencesMethod:
		req := &publishpb.ListPhotoSequencesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		filter, err := parseGCPStreetViewPublishFilter(req.GetFilter(), map[string]string{
			"imagery_type":             "string",
			"processing_state":         "string",
			"min_latitude":             "float",
			"max_latitude":             "float",
			"min_longitude":            "float",
			"max_longitude":            "float",
			"filename_query":           "string",
			"min_capture_time_seconds": "int",
			"max_capture_time_seconds": "int",
		})
		if err != nil {
			return grpcInvalidArgument("filter-invalid")
		}
		items := []*longrunningpb.Operation{
			gcpStage4StreetViewPublishOperation(gcpStreetViewPublishOperationID("sequence-1"), "sequence-1", true, publishpb.CreatePhotoSequenceRequest_VIDEO),
			gcpStage4StreetViewPublishOperation(gcpStreetViewPublishOperationID("sequence-processing"), "sequence-processing", false, publishpb.CreatePhotoSequenceRequest_VIDEO),
		}
		if state := strings.ToUpper(strings.TrimSpace(filter["processing_state"])); state != "" {
			filtered := make([]*longrunningpb.Operation, 0, len(items))
			for _, item := range items {
				sequenceID := gcpStreetViewPublishSequenceIDFromOperationID(strings.TrimPrefix(item.GetName(), "operations/"))
				current := "PROCESSING"
				if !isGCPStreetViewPublishProcessingID(sequenceID) {
					current = "PROCESSED"
				}
				if current == state {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&publishpb.ListPhotoSequencesResponse{
			PhotoSequences: items[start:end],
			NextPageToken:  next,
		})
	case gcpStreetViewPublishDeletePhotoSequenceMethod:
		req := &publishpb.DeletePhotoSequenceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		sequenceID := strings.TrimSpace(req.GetSequenceId())
		if !isGCPStreetViewPublishSequenceID(sequenceID) {
			return grpcInvalidArgument("sequence_id-required")
		}
		if strings.Contains(strings.ToLower(sequenceID), "missing") {
			return grpcNotFound("photo_sequence-not-found")
		}
		if isGCPStreetViewPublishProcessingID(sequenceID) {
			return grpcFailedPrecondition("photo_sequence-processing")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	default:
		return nil, "", "", false
	}
}

func gcpStage4StreetViewPublishUploadRef(uploadID string) *publishpb.UploadRef {
	return &publishpb.UploadRef{
		FileSource: &publishpb.UploadRef_UploadUrl{
			UploadUrl: gcpStreetViewPublishUploadURL(uploadID),
		},
	}
}

func gcpStage4StreetViewPublishPhoto(photoID string, includeDownloadURL bool) *publishpb.Photo {
	photo := &publishpb.Photo{
		PhotoId: &publishpb.PhotoId{
			Id: photoID,
		},
		UploadReference:   gcpStage4StreetViewPublishUploadRef(photoID),
		ThumbnailUrl:      fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/%s/thumbnail", photoID),
		ShareLink:         fmt.Sprintf("https://maps.google.com/?q=stackyard-photo-%s", photoID),
		CaptureTime:       timestamppb.New(gcpStreetViewPublishReferenceTime.Add(5 * time.Minute)),
		UploadTime:        timestamppb.New(gcpStreetViewPublishReferenceTime.Add(30 * time.Minute)),
		ViewCount:         101,
		TransferStatus:    publishpb.Photo_NEVER_TRANSFERRED,
		MapsPublishStatus: publishpb.Photo_PUBLISHED,
		Places: []*publishpb.Place{
			{PlaceId: "ChIJj61dQgK6j4AR4GeTYWZsKWw"},
		},
	}
	if includeDownloadURL {
		photo.DownloadUrl = fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/%s/download", photoID)
	}
	return photo
}

func gcpStage4StreetViewPublishPhotoSequence(sequenceID string, inputType publishpb.CreatePhotoSequenceRequest_InputType) *publishpb.PhotoSequence {
	processingState := publishpb.ProcessingState_PROCESSED
	if isGCPStreetViewPublishProcessingID(sequenceID) {
		processingState = publishpb.ProcessingState_PROCESSING
	}
	_ = inputType
	return &publishpb.PhotoSequence{
		Id:              sequenceID,
		UploadReference: gcpStage4StreetViewPublishUploadRef("sequence-" + sequenceID),
		UploadTime:      timestamppb.New(gcpStreetViewPublishReferenceTime.Add(2 * time.Hour)),
		ProcessingState: processingState,
		Photos: []*publishpb.Photo{
			gcpStage4StreetViewPublishPhoto("photo-1", false),
			gcpStage4StreetViewPublishPhoto("photo-2", false),
		},
		DistanceMeters: 12.5,
		ViewCount:      12,
		Filename:       sequenceID + ".mp4",
	}
}

func gcpStage4StreetViewPublishOperation(operationID, sequenceID string, done bool, inputType publishpb.CreatePhotoSequenceRequest_InputType) *longrunningpb.Operation {
	operation := &longrunningpb.Operation{
		Name: "operations/" + operationID,
		Done: done,
	}
	metadataAny, err := anypb.New(&emptypb.Empty{})
	if err == nil {
		operation.Metadata = metadataAny
	}
	if done {
		responseAny, err := anypb.New(gcpStage4StreetViewPublishPhotoSequence(sequenceID, inputType))
		if err == nil {
			operation.Result = &longrunningpb.Operation_Response{Response: responseAny}
		}
	}
	return operation
}

func gcpStage4StreetViewPublishStatus(code int32, message string) *statuspb.Status {
	return &statuspb.Status{
		Code:    code,
		Message: message,
	}
}

func isGCPStage4StreetViewPublishPhotoView(view publishpb.PhotoView) bool {
	return view == publishpb.PhotoView_BASIC || view == publishpb.PhotoView_INCLUDE_DOWNLOAD_URL
}

func isGCPStage4StreetViewPublishInputType(inputType publishpb.CreatePhotoSequenceRequest_InputType) bool {
	return inputType == publishpb.CreatePhotoSequenceRequest_VIDEO || inputType == publishpb.CreatePhotoSequenceRequest_XDM
}

func gcpStage4StorageBatchOperationsValidateJob(job *storagebatchoperationspb.Job) string {
	if job.GetBucketList() == nil {
		return "job.bucket_list-required"
	}
	buckets := job.GetBucketList().GetBuckets()
	if len(buckets) == 0 {
		return "job.bucket_list.buckets-required"
	}
	for _, bucket := range buckets {
		if strings.TrimSpace(bucket.GetBucket()) == "" {
			return "job.bucket_list.buckets.bucket-required"
		}
	}

	transformationCount := 0
	if job.GetPutObjectHold() != nil {
		transformationCount++
	}
	if job.GetDeleteObject() != nil {
		transformationCount++
	}
	if job.GetPutMetadata() != nil {
		transformationCount++
	}
	if job.GetRewriteObject() != nil {
		transformationCount++
	}
	if job.GetUpdateObjectCustomContext() != nil {
		transformationCount++
		return "job.update_object_custom_context-not-supported"
	}
	if transformationCount == 0 {
		return "job.transformation-required"
	}
	if transformationCount > 1 {
		return "job.transformation-oneof-invalid"
	}
	return ""
}

func gcpStage4StorageBatchOperationsSortJobs(items []*storagebatchoperationspb.Job, orderBy string) {
	switch orderBy {
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return items[i].GetName() < items[j].GetName()
		})
	case "create_time", "create_time asc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStage4StorageBatchOperationsTime(items[i].GetCreateTime()).Before(gcpStage4StorageBatchOperationsTime(items[j].GetCreateTime()))
		})
	case "create_time desc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStage4StorageBatchOperationsTime(items[i].GetCreateTime()).After(gcpStage4StorageBatchOperationsTime(items[j].GetCreateTime()))
		})
	}
}

func gcpStage4StorageBatchOperationsSortBucketOperations(items []*storagebatchoperationspb.BucketOperation, orderBy string) {
	switch orderBy {
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return items[i].GetName() < items[j].GetName()
		})
	case "create_time", "create_time asc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStage4StorageBatchOperationsTime(items[i].GetCreateTime()).Before(gcpStage4StorageBatchOperationsTime(items[j].GetCreateTime()))
		})
	case "create_time desc":
		sort.Slice(items, func(i, j int) bool {
			return gcpStage4StorageBatchOperationsTime(items[i].GetCreateTime()).After(gcpStage4StorageBatchOperationsTime(items[j].GetCreateTime()))
		})
	}
}

func gcpStage4StorageBatchOperationsTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func gcpStage4StorageBatchOperationsJob(project, jobID, state string) *storagebatchoperationspb.Job {
	return gcpStage4StorageBatchOperationsJobFromMap(gcpStorageBatchOperationsJob(project, jobID, state))
}

func gcpStage4StorageBatchOperationsJobFromRequest(project, jobID, state string, req *storagebatchoperationspb.Job) *storagebatchoperationspb.Job {
	return gcpStage4StorageBatchOperationsJobFromMap(gcpStorageBatchOperationsJobFromRequest(project, jobID, state, gcpStage4StorageBatchOperationsProtoToMap(req)))
}

func gcpStage4StorageBatchOperationsBucketOperation(project, jobID, bucketOperationID, state string) *storagebatchoperationspb.BucketOperation {
	return gcpStage4StorageBatchOperationsBucketOperationFromMap(gcpStorageBatchOperationsBucketOperation(project, jobID, bucketOperationID, state))
}

func gcpStage4StorageBatchOperationsJobFromMap(source map[string]any) *storagebatchoperationspb.Job {
	out := &storagebatchoperationspb.Job{}
	gcpStage4StorageBatchOperationsMapToProto(source, out)
	if out.GetState() == storagebatchoperationspb.Job_STATE_UNSPECIFIED {
		out.State = gcpStage4StorageBatchOperationsJobState(gcpStorageBatchOperationsString(source, "state"))
	}
	return out
}

func gcpStage4StorageBatchOperationsBucketOperationFromMap(source map[string]any) *storagebatchoperationspb.BucketOperation {
	out := &storagebatchoperationspb.BucketOperation{}
	gcpStage4StorageBatchOperationsMapToProto(source, out)
	if out.GetState() == storagebatchoperationspb.BucketOperation_STATE_UNSPECIFIED {
		out.State = gcpStage4StorageBatchOperationsBucketOperationState(gcpStorageBatchOperationsString(source, "state"))
	}
	return out
}

func gcpStage4StorageBatchOperationsProtoToMap(message proto.Message) map[string]any {
	if message == nil {
		return map[string]any{}
	}
	payload, err := protojson.Marshal(message)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(payload, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func gcpStage4StorageBatchOperationsMapToProto(source map[string]any, message proto.Message) {
	if message == nil || source == nil {
		return
	}
	payload, err := json.Marshal(source)
	if err != nil {
		return
	}
	_ = protojson.Unmarshal(payload, message)
}

func gcpStage4StorageBatchOperationsJobState(state string) storagebatchoperationspb.Job_State {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "RUNNING":
		return storagebatchoperationspb.Job_RUNNING
	case "SUCCEEDED":
		return storagebatchoperationspb.Job_SUCCEEDED
	case "FAILED":
		return storagebatchoperationspb.Job_FAILED
	case "CANCELED":
		return storagebatchoperationspb.Job_CANCELED
	case "QUEUED":
		return storagebatchoperationspb.Job_QUEUED
	default:
		return storagebatchoperationspb.Job_STATE_UNSPECIFIED
	}
}

func gcpStage4StorageBatchOperationsBucketOperationState(state string) storagebatchoperationspb.BucketOperation_State {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "RUNNING":
		return storagebatchoperationspb.BucketOperation_RUNNING
	case "SUCCEEDED":
		return storagebatchoperationspb.BucketOperation_SUCCEEDED
	case "FAILED":
		return storagebatchoperationspb.BucketOperation_FAILED
	case "CANCELED":
		return storagebatchoperationspb.BucketOperation_CANCELED
	case "QUEUED":
		return storagebatchoperationspb.BucketOperation_QUEUED
	default:
		return storagebatchoperationspb.BucketOperation_STATE_UNSPECIFIED
	}
}

func gcpStage4StorageBatchOperationsOperation(project, operationID string, job *storagebatchoperationspb.Job, done, requestedCancellation bool) *longrunningpb.Operation {
	operationName := fmt.Sprintf("projects/%s/locations/global/operations/%s", project, operationID)
	metadata := &storagebatchoperationspb.OperationMetadata{
		Operation:             operationName,
		CreateTime:            timestamppb.New(gcpStorageBatchOperationsReferenceTime),
		RequestedCancellation: requestedCancellation,
		ApiVersion:            "v1",
		Job:                   job,
	}
	if done {
		metadata.EndTime = timestamppb.New(gcpStorageBatchOperationsReferenceTime.Add(2 * time.Minute))
	}
	metadataAny, err := anypb.New(metadata)
	if err != nil {
		metadataAny = nil
	}

	operation := &longrunningpb.Operation{
		Name:     operationName,
		Done:     done,
		Metadata: metadataAny,
	}
	if done {
		responseAny, err := anypb.New(job)
		if err == nil {
			operation.Result = &longrunningpb.Operation_Response{Response: responseAny}
		}
	}
	return operation
}

func gcpStage4GRPCSpeech(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpeechRecognizeMethod:
		req := &speechpb.RecognizeRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetConfig() == nil {
			return grpcInvalidArgument("config-required")
		}
		if req.GetAudio() == nil {
			return grpcInvalidArgument("audio-required")
		}
		if strings.TrimSpace(req.GetConfig().GetLanguageCode()) == "" {
			return grpcInvalidArgument("config.language_code-required")
		}
		if req.GetConfig().GetEncoding() == speechpb.RecognitionConfig_ENCODING_UNSPECIFIED {
			return grpcInvalidArgument("config.encoding-invalid")
		}
		if req.GetConfig().GetSampleRateHertz() < 0 {
			return grpcInvalidArgument("config.sample_rate_hertz-invalid")
		}
		hasContent := len(req.GetAudio().GetContent()) > 0
		hasURI := strings.TrimSpace(req.GetAudio().GetUri()) != ""
		if !hasContent && !hasURI {
			return grpcInvalidArgument("audio-source-required")
		}
		if hasContent && hasURI {
			return grpcInvalidArgument("audio-source-multiple")
		}
		if hasURI && strings.Contains(strings.ToLower(req.GetAudio().GetUri()), "missing") {
			return grpcNotFound("audio-uri-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpeechRecognizeResponse(req.GetConfig().GetLanguageCode(), "stackyard recognized speech"))
	case gcpSpeechLongRunningRecognizeMethod:
		req := &speechpb.LongRunningRecognizeRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetConfig() == nil {
			return grpcInvalidArgument("config-required")
		}
		if req.GetAudio() == nil {
			return grpcInvalidArgument("audio-required")
		}
		if strings.TrimSpace(req.GetConfig().GetLanguageCode()) == "" {
			return grpcInvalidArgument("config.language_code-required")
		}
		hasContent := len(req.GetAudio().GetContent()) > 0
		hasURI := strings.TrimSpace(req.GetAudio().GetUri()) != ""
		if !hasContent && !hasURI {
			return grpcInvalidArgument("audio-source-required")
		}
		if hasContent && hasURI {
			return grpcInvalidArgument("audio-source-multiple")
		}
		if hasURI && strings.Contains(strings.ToLower(req.GetAudio().GetUri()), "missing") {
			return grpcNotFound("audio-uri-not-found")
		}

		responseAny, err := anypb.New(&speechpb.LongRunningRecognizeResponse{
			Results:         gcpStage4SpeechRecognizeResponse(req.GetConfig().GetLanguageCode(), "stackyard long running recognized speech").GetResults(),
			TotalBilledTime: durationpb.New(1200 * time.Millisecond),
		})
		if err != nil {
			return grpcInvalidArgument("response-any-encode-failed")
		}
		metadataAny, err := anypb.New(&speechpb.LongRunningRecognizeMetadata{
			ProgressPercent: 100,
			StartTime:       timestamppb.New(gcpStage4ReferenceTime.Add(-2 * time.Second)),
			LastUpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
		})
		if err != nil {
			return grpcInvalidArgument("metadata-any-encode-failed")
		}
		return grpcProtoSuccess(&longrunningpb.Operation{
			Name:     "projects/stackyard/locations/global/operations/speech.longRunningRecognize.op-1",
			Metadata: metadataAny,
			Done:     true,
			Result: &longrunningpb.Operation_Response{
				Response: responseAny,
			},
		})
	case gcpSpeechStreamingRecognizeMethod:
		req := &speechpb.StreamingRecognizeRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		streamingConfig := req.GetStreamingConfig()
		if streamingConfig == nil {
			return grpcInvalidArgument("streaming_config-required")
		}
		if streamingConfig.GetConfig() == nil {
			return grpcInvalidArgument("streaming_config.config-required")
		}
		if strings.TrimSpace(streamingConfig.GetConfig().GetLanguageCode()) == "" {
			return grpcInvalidArgument("streaming_config.config.language_code-required")
		}
		if streamingConfig.GetConfig().GetSampleRateHertz() < 0 {
			return grpcInvalidArgument("streaming_config.config.sample_rate_hertz-invalid")
		}
		return grpcProtoSuccess(gcpStage4SpeechStreamingRecognizeResponse(streamingConfig.GetConfig().GetLanguageCode(), "stackyard streaming recognized speech"))
	default:
		return nil, "", "", false
	}
}

func gcpStage4GRPCSpeechAdaptation(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpeechCreatePhraseSetMethod:
		req := &speechpb.CreatePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		phraseSetID := strings.TrimSpace(req.GetPhraseSetId())
		if !isGCPSpeechResourceID(phraseSetID) {
			return grpcInvalidArgument("phrase_set_id-required")
		}
		if req.GetPhraseSet() == nil {
			return grpcInvalidArgument("phrase_set-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "existing") {
			return grpcAlreadyExists("phrase_set-already-exists")
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID)
		if providedName := strings.TrimSpace(req.GetPhraseSet().GetName()); providedName != "" && providedName != expectedName {
			return grpcInvalidArgument("phrase_set.name-must-match-parent")
		}
		phraseSet := gcpStage4SpeechPhraseSet(project, location, phraseSetID)
		if req.GetPhraseSet().GetBoost() != 0 {
			phraseSet.Boost = req.GetPhraseSet().GetBoost()
		}
		if len(req.GetPhraseSet().GetPhrases()) > 0 {
			phraseSet.Phrases = req.GetPhraseSet().GetPhrases()
		}
		return grpcProtoSuccess(phraseSet)
	case gcpSpeechGetPhraseSetMethod:
		req := &speechpb.GetPhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") {
			return grpcNotFound("phrase_set-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpeechPhraseSet(project, location, phraseSetID))
	case gcpSpeechListPhraseSetMethod:
		req := &speechpb.ListPhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*speechpb.PhraseSet{
			gcpStage4SpeechPhraseSet(project, location, "phrase-set-1"),
			gcpStage4SpeechPhraseSet(project, location, "phrase-set-2"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		nextToken := ""
		if end < len(items) {
			nextToken = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&speechpb.ListPhraseSetResponse{
			PhraseSets:    items[start:end],
			NextPageToken: nextToken,
		})
	case gcpSpeechUpdatePhraseSetMethod:
		req := &speechpb.UpdatePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetPhraseSet() == nil {
			return grpcInvalidArgument("phrase_set-required")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetPhraseSet().GetName())
		if !ok {
			return grpcInvalidArgument("phrase_set.name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") {
			return grpcNotFound("phrase_set-not-found")
		}
		for _, path := range req.GetUpdateMask().GetPaths() {
			if strings.Contains(strings.ToLower(path), "name") {
				return grpcFailedPrecondition("phrase_set.name-immutable")
			}
		}
		phraseSet := gcpStage4SpeechPhraseSet(project, location, phraseSetID)
		if req.GetPhraseSet().GetBoost() != 0 {
			phraseSet.Boost = req.GetPhraseSet().GetBoost()
		}
		if len(req.GetPhraseSet().GetPhrases()) > 0 {
			phraseSet.Phrases = req.GetPhraseSet().GetPhrases()
		}
		return grpcProtoSuccess(phraseSet)
	case gcpSpeechDeletePhraseSetMethod:
		req := &speechpb.DeletePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, _, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") {
			return grpcNotFound("phrase_set-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpeechCreateCustomClassMethod:
		req := &speechpb.CreateCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		customClassID := strings.TrimSpace(req.GetCustomClassId())
		if !isGCPSpeechResourceID(customClassID) {
			return grpcInvalidArgument("custom_class_id-required")
		}
		if req.GetCustomClass() == nil {
			return grpcInvalidArgument("custom_class-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "existing") {
			return grpcAlreadyExists("custom_class-already-exists")
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID)
		if providedName := strings.TrimSpace(req.GetCustomClass().GetName()); providedName != "" && providedName != expectedName {
			return grpcInvalidArgument("custom_class.name-must-match-parent")
		}
		customClass := gcpStage4SpeechCustomClass(project, location, customClassID)
		if len(req.GetCustomClass().GetItems()) > 0 {
			customClass.Items = req.GetCustomClass().GetItems()
		}
		return grpcProtoSuccess(customClass)
	case gcpSpeechGetCustomClassMethod:
		req := &speechpb.GetCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, customClassID, ok := parseGCPSpeechCustomClassName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") {
			return grpcNotFound("custom_class-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpeechCustomClass(project, location, customClassID))
	case gcpSpeechListCustomClassesMethod:
		req := &speechpb.ListCustomClassesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*speechpb.CustomClass{
			gcpStage4SpeechCustomClass(project, location, "custom-class-1"),
			gcpStage4SpeechCustomClass(project, location, "custom-class-2"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		nextToken := ""
		if end < len(items) {
			nextToken = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&speechpb.ListCustomClassesResponse{
			CustomClasses: items[start:end],
			NextPageToken: nextToken,
		})
	case gcpSpeechUpdateCustomClassMethod:
		req := &speechpb.UpdateCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetCustomClass() == nil {
			return grpcInvalidArgument("custom_class-required")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		project, location, customClassID, ok := parseGCPSpeechCustomClassName(req.GetCustomClass().GetName())
		if !ok {
			return grpcInvalidArgument("custom_class.name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") {
			return grpcNotFound("custom_class-not-found")
		}
		for _, path := range req.GetUpdateMask().GetPaths() {
			if strings.Contains(strings.ToLower(path), "name") {
				return grpcFailedPrecondition("custom_class.name-immutable")
			}
		}
		customClass := gcpStage4SpeechCustomClass(project, location, customClassID)
		if len(req.GetCustomClass().GetItems()) > 0 {
			customClass.Items = req.GetCustomClass().GetItems()
		}
		return grpcProtoSuccess(customClass)
	case gcpSpeechDeleteCustomClassMethod:
		req := &speechpb.DeleteCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		_, _, customClassID, ok := parseGCPSpeechCustomClassName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") {
			return grpcNotFound("custom_class-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	default:
		return nil, "", "", false
	}
}

func gcpStage4SpeechRecognizeResponse(languageCode, transcript string) *speechpb.RecognizeResponse {
	return &speechpb.RecognizeResponse{
		Results: []*speechpb.SpeechRecognitionResult{
			{
				Alternatives: []*speechpb.SpeechRecognitionAlternative{
					{
						Transcript: transcript,
						Confidence: 0.98,
					},
				},
				ChannelTag:    1,
				ResultEndTime: durationpb.New(1200 * time.Millisecond),
				LanguageCode:  languageCode,
			},
		},
		TotalBilledTime: durationpb.New(1200 * time.Millisecond),
		RequestId:       1,
	}
}

func gcpStage4SpeechStreamingRecognizeResponse(languageCode, transcript string) *speechpb.StreamingRecognizeResponse {
	return &speechpb.StreamingRecognizeResponse{
		Results: []*speechpb.StreamingRecognitionResult{
			{
				Alternatives: []*speechpb.SpeechRecognitionAlternative{
					{
						Transcript: transcript,
						Confidence: 0.93,
					},
				},
				IsFinal:       true,
				Stability:     0.91,
				ResultEndTime: durationpb.New(900 * time.Millisecond),
				LanguageCode:  languageCode,
			},
		},
		SpeechEventType: speechpb.StreamingRecognizeResponse_END_OF_SINGLE_UTTERANCE,
		SpeechEventTime: durationpb.New(900 * time.Millisecond),
		TotalBilledTime: durationpb.New(900 * time.Millisecond),
		RequestId:       1,
	}
}

func gcpStage4SpeechPhraseSet(project, location, phraseSetID string) *speechpb.PhraseSet {
	return &speechpb.PhraseSet{
		Name: fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID),
		Phrases: []*speechpb.PhraseSet_Phrase{
			{Value: "stackyard"},
			{Value: "cloud emulation"},
		},
		Boost: 12.5,
	}
}

func gcpStage4SpeechCustomClass(project, location, customClassID string) *speechpb.CustomClass {
	return &speechpb.CustomClass{
		Name:          fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID),
		CustomClassId: customClassID,
		Items: []*speechpb.CustomClass_ClassItem{
			{Value: "stackyard"},
			{Value: "speech"},
		},
	}
}

func gcpStage4GRPCSpeechV2(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpeechV2CreateRecognizerMethod:
		req := &speechv2pb.CreateRecognizerRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		recognizerID := strings.TrimSpace(req.GetRecognizerId())
		if !isGCPSpeechV2ResourceID(recognizerID) {
			return grpcInvalidArgument("recognizer_id-required")
		}
		if req.GetRecognizer() == nil {
			return grpcInvalidArgument("recognizer-required")
		}
		if strings.Contains(strings.ToLower(recognizerID), "existing") {
			return grpcAlreadyExists("recognizer-already-exists")
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/recognizers/%s", project, location, recognizerID)
		if provided := strings.TrimSpace(req.GetRecognizer().GetName()); provided != "" && provided != expectedName {
			return grpcInvalidArgument("recognizer.name-must-match-parent")
		}
		recognizer := gcpStage4SpeechV2Recognizer(project, location, recognizerID, false)
		gcpStage4SpeechV2ApplyRecognizerOverrides(recognizer, req.GetRecognizer())
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "createRecognizer."+recognizerID, path, expectedName, recognizer))
	case gcpSpeechV2ListRecognizersMethod:
		req := &speechv2pb.ListRecognizersRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize, start, reason, ok := gcpStage4SpeechV2ListPage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*speechv2pb.Recognizer{
			gcpStage4SpeechV2Recognizer(project, location, "recognizer-1", false),
			gcpStage4SpeechV2Recognizer(project, location, "recognizer-deleted", true),
		}
		if !req.GetShowDeleted() {
			items = items[:1]
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&speechv2pb.ListRecognizersResponse{
			Recognizers:   items[start:end],
			NextPageToken: next,
		})
	case gcpSpeechV2GetRecognizerMethod:
		req := &speechv2pb.GetRecognizerRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(recognizerID), "missing") {
			return grpcNotFound("recognizer-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpeechV2Recognizer(project, location, recognizerID, strings.Contains(strings.ToLower(recognizerID), "deleted")))
	case gcpSpeechV2UpdateRecognizerMethod:
		req := &speechv2pb.UpdateRecognizerRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetRecognizer() == nil {
			return grpcInvalidArgument("recognizer-required")
		}
		project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(req.GetRecognizer().GetName())
		if !ok {
			return grpcInvalidArgument("recognizer.name-required")
		}
		if strings.Contains(strings.ToLower(recognizerID), "missing") {
			return grpcNotFound("recognizer-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		for _, p := range req.GetUpdateMask().GetPaths() {
			if strings.EqualFold(strings.TrimSpace(p), "name") {
				return grpcFailedPrecondition("recognizer.name-immutable")
			}
		}
		if etag := strings.TrimSpace(req.GetRecognizer().GetEtag()); etag != "" && etag != gcpSpeechV2Etag(recognizerID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		recognizer := gcpStage4SpeechV2Recognizer(project, location, recognizerID, false)
		gcpStage4SpeechV2ApplyRecognizerOverrides(recognizer, req.GetRecognizer())
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "updateRecognizer."+recognizerID, path, recognizer.GetName(), recognizer))
	case gcpSpeechV2DeleteRecognizerMethod:
		req := &speechv2pb.DeleteRecognizerRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(recognizerID), "missing") && !req.GetAllowMissing() {
			return grpcNotFound("recognizer-not-found")
		}
		if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSpeechV2Etag(recognizerID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		recognizer := gcpStage4SpeechV2Recognizer(project, location, recognizerID, true)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "deleteRecognizer."+recognizerID, path, recognizer.GetName(), recognizer))
	case gcpSpeechV2UndeleteRecognizerMethod:
		req := &speechv2pb.UndeleteRecognizerRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, recognizerID, ok := parseGCPSpeechV2RecognizerName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(recognizerID), "missing") {
			return grpcNotFound("recognizer-not-found")
		}
		if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSpeechV2Etag(recognizerID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		recognizer := gcpStage4SpeechV2Recognizer(project, location, recognizerID, false)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "undeleteRecognizer."+recognizerID, path, recognizer.GetName(), recognizer))
	case gcpSpeechV2RecognizeMethod:
		req := &speechv2pb.RecognizeRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, recognizerID, ok := parseGCPSpeechV2RecognizerNameOrImplicit(req.GetRecognizer())
		if !ok {
			return grpcInvalidArgument("recognizer-required")
		}
		hasContent := len(req.GetContent()) > 0
		hasURI := strings.TrimSpace(req.GetUri()) != ""
		if !hasContent && !hasURI {
			return grpcInvalidArgument("audio-source-required")
		}
		if hasContent && hasURI {
			return grpcInvalidArgument("audio-source-multiple")
		}
		if hasURI && strings.Contains(strings.ToLower(req.GetUri()), "missing") {
			return grpcNotFound("audio-uri-not-found")
		}
		languageCode := "en-US"
		if req.GetConfig() != nil && len(req.GetConfig().GetLanguageCodes()) > 0 {
			languageCode = strings.TrimSpace(req.GetConfig().GetLanguageCodes()[0])
		}
		if recognizerID == "_" && languageCode == "" {
			return grpcInvalidArgument("config.language_codes-required-for-implicit-recognizer")
		}
		return grpcProtoSuccess(gcpStage4SpeechV2RecognizeResponse(project, location, languageCode, "stackyard speech v2 recognized audio"))
	case gcpSpeechV2StreamingRecognizeMethod:
		req := &speechv2pb.StreamingRecognizeRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, _, ok := parseGCPSpeechV2RecognizerNameOrImplicit(req.GetRecognizer())
		if !ok {
			return grpcInvalidArgument("recognizer-required")
		}
		if req.GetStreamingConfig() == nil {
			return grpcInvalidArgument("streaming_config-required")
		}
		if req.GetStreamingConfig().GetConfig() == nil {
			return grpcInvalidArgument("streaming_config.config-required")
		}
		languageCode := ""
		if langs := req.GetStreamingConfig().GetConfig().GetLanguageCodes(); len(langs) > 0 {
			languageCode = strings.TrimSpace(langs[0])
		}
		if languageCode == "" {
			return grpcInvalidArgument("streaming_config.config.language_codes-required")
		}
		if len(req.GetAudio()) > 0 {
			return grpcInvalidArgument("audio-not-supported-in-staged-unary-streaming")
		}
		return grpcProtoSuccess(gcpStage4SpeechV2StreamingRecognizeResponse(project, location, languageCode, "stackyard speech v2 streaming transcript"))
	case gcpSpeechV2BatchRecognizeMethod:
		req := &speechv2pb.BatchRecognizeRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, _, ok := parseGCPSpeechV2RecognizerNameOrImplicit(req.GetRecognizer())
		if !ok {
			return grpcInvalidArgument("recognizer-required")
		}
		if len(req.GetFiles()) == 0 {
			return grpcInvalidArgument("files-required")
		}
		if len(req.GetFiles()) > 15 {
			return grpcInvalidArgument("files-too-many")
		}
		uris := make([]string, 0, len(req.GetFiles()))
		for _, file := range req.GetFiles() {
			uri := strings.TrimSpace(file.GetUri())
			if uri == "" {
				return grpcInvalidArgument("files.uri-required")
			}
			if strings.Contains(strings.ToLower(uri), "missing") {
				return grpcNotFound("audio-uri-not-found")
			}
			uris = append(uris, uri)
		}
		if req.GetRecognitionOutputConfig() == nil {
			return grpcInvalidArgument("recognition_output_config-required")
		}
		if req.GetRecognitionOutputConfig().GetGcsOutputConfig() == nil && req.GetRecognitionOutputConfig().GetInlineResponseConfig() == nil {
			return grpcInvalidArgument("recognition_output_config.output-required")
		}
		response := gcpStage4SpeechV2BatchRecognizeResponse(uris)
		resource := fmt.Sprintf("projects/%s/locations/%s/recognizers/_", project, location)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "batchRecognize.stackyard", path, resource, response))
	case gcpSpeechV2GetConfigMethod:
		req := &speechv2pb.GetConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechV2ConfigName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		return grpcProtoSuccess(gcpStage4SpeechV2Config(project, location))
	case gcpSpeechV2UpdateConfigMethod:
		req := &speechv2pb.UpdateConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetConfig() == nil {
			return grpcInvalidArgument("config-required")
		}
		project, location, ok := parseGCPSpeechV2ConfigName(req.GetConfig().GetName())
		if !ok {
			return grpcInvalidArgument("config.name-required")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		for _, p := range req.GetUpdateMask().GetPaths() {
			if strings.EqualFold(strings.TrimSpace(p), "name") {
				return grpcFailedPrecondition("config.name-immutable")
			}
		}
		config := gcpStage4SpeechV2Config(project, location)
		if kms := strings.TrimSpace(req.GetConfig().GetKmsKeyName()); kms != "" {
			config.KmsKeyName = kms
		}
		return grpcProtoSuccess(config)
	case gcpSpeechV2CreateCustomClassMethod:
		req := &speechv2pb.CreateCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		customClassID := strings.TrimSpace(req.GetCustomClassId())
		if !isGCPSpeechV2ResourceID(customClassID) {
			return grpcInvalidArgument("custom_class_id-required")
		}
		if req.GetCustomClass() == nil {
			return grpcInvalidArgument("custom_class-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "existing") {
			return grpcAlreadyExists("custom_class-already-exists")
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID)
		if provided := strings.TrimSpace(req.GetCustomClass().GetName()); provided != "" && provided != expectedName {
			return grpcInvalidArgument("custom_class.name-must-match-parent")
		}
		customClass := gcpStage4SpeechV2CustomClass(project, location, customClassID, false)
		gcpStage4SpeechV2ApplyCustomClassOverrides(customClass, req.GetCustomClass())
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "createCustomClass."+customClassID, path, expectedName, customClass))
	case gcpSpeechV2ListCustomClassesMethod:
		req := &speechv2pb.ListCustomClassesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize, start, reason, ok := gcpStage4SpeechV2ListPage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*speechv2pb.CustomClass{
			gcpStage4SpeechV2CustomClass(project, location, "custom-class-1", false),
			gcpStage4SpeechV2CustomClass(project, location, "custom-class-deleted", true),
		}
		if !req.GetShowDeleted() {
			items = items[:1]
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&speechv2pb.ListCustomClassesResponse{
			CustomClasses: items[start:end],
			NextPageToken: next,
		})
	case gcpSpeechV2GetCustomClassMethod:
		req := &speechv2pb.GetCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, customClassID, ok := parseGCPSpeechCustomClassName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") {
			return grpcNotFound("custom_class-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpeechV2CustomClass(project, location, customClassID, strings.Contains(strings.ToLower(customClassID), "deleted")))
	case gcpSpeechV2UpdateCustomClassMethod:
		req := &speechv2pb.UpdateCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetCustomClass() == nil {
			return grpcInvalidArgument("custom_class-required")
		}
		project, location, customClassID, ok := parseGCPSpeechCustomClassName(req.GetCustomClass().GetName())
		if !ok {
			return grpcInvalidArgument("custom_class.name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") {
			return grpcNotFound("custom_class-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		for _, p := range req.GetUpdateMask().GetPaths() {
			if strings.EqualFold(strings.TrimSpace(p), "name") {
				return grpcFailedPrecondition("custom_class.name-immutable")
			}
		}
		if etag := strings.TrimSpace(req.GetCustomClass().GetEtag()); etag != "" && etag != gcpSpeechV2Etag(customClassID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		customClass := gcpStage4SpeechV2CustomClass(project, location, customClassID, false)
		gcpStage4SpeechV2ApplyCustomClassOverrides(customClass, req.GetCustomClass())
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "updateCustomClass."+customClassID, path, customClass.GetName(), customClass))
	case gcpSpeechV2DeleteCustomClassMethod:
		req := &speechv2pb.DeleteCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, customClassID, ok := parseGCPSpeechCustomClassName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") && !req.GetAllowMissing() {
			return grpcNotFound("custom_class-not-found")
		}
		if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSpeechV2Etag(customClassID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		customClass := gcpStage4SpeechV2CustomClass(project, location, customClassID, true)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "deleteCustomClass."+customClassID, path, customClass.GetName(), customClass))
	case gcpSpeechV2UndeleteCustomClassMethod:
		req := &speechv2pb.UndeleteCustomClassRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, customClassID, ok := parseGCPSpeechCustomClassName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(customClassID), "missing") {
			return grpcNotFound("custom_class-not-found")
		}
		if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSpeechV2Etag(customClassID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		customClass := gcpStage4SpeechV2CustomClass(project, location, customClassID, false)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "undeleteCustomClass."+customClassID, path, customClass.GetName(), customClass))
	case gcpSpeechV2CreatePhraseSetMethod:
		req := &speechv2pb.CreatePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		phraseSetID := strings.TrimSpace(req.GetPhraseSetId())
		if !isGCPSpeechV2ResourceID(phraseSetID) {
			return grpcInvalidArgument("phrase_set_id-required")
		}
		if req.GetPhraseSet() == nil {
			return grpcInvalidArgument("phrase_set-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "existing") {
			return grpcAlreadyExists("phrase_set-already-exists")
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID)
		if provided := strings.TrimSpace(req.GetPhraseSet().GetName()); provided != "" && provided != expectedName {
			return grpcInvalidArgument("phrase_set.name-must-match-parent")
		}
		phraseSet := gcpStage4SpeechV2PhraseSet(project, location, phraseSetID, false)
		gcpStage4SpeechV2ApplyPhraseSetOverrides(phraseSet, req.GetPhraseSet())
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "createPhraseSet."+phraseSetID, path, expectedName, phraseSet))
	case gcpSpeechV2ListPhraseSetsMethod:
		req := &speechv2pb.ListPhraseSetsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, ok := parseGCPSpeechParentName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize, start, reason, ok := gcpStage4SpeechV2ListPage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*speechv2pb.PhraseSet{
			gcpStage4SpeechV2PhraseSet(project, location, "phrase-set-1", false),
			gcpStage4SpeechV2PhraseSet(project, location, "phrase-set-deleted", true),
		}
		if !req.GetShowDeleted() {
			items = items[:1]
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&speechv2pb.ListPhraseSetsResponse{
			PhraseSets:    items[start:end],
			NextPageToken: next,
		})
	case gcpSpeechV2GetPhraseSetMethod:
		req := &speechv2pb.GetPhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") {
			return grpcNotFound("phrase_set-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpeechV2PhraseSet(project, location, phraseSetID, strings.Contains(strings.ToLower(phraseSetID), "deleted")))
	case gcpSpeechV2UpdatePhraseSetMethod:
		req := &speechv2pb.UpdatePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetPhraseSet() == nil {
			return grpcInvalidArgument("phrase_set-required")
		}
		project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetPhraseSet().GetName())
		if !ok {
			return grpcInvalidArgument("phrase_set.name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") {
			return grpcNotFound("phrase_set-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		for _, p := range req.GetUpdateMask().GetPaths() {
			if strings.EqualFold(strings.TrimSpace(p), "name") {
				return grpcFailedPrecondition("phrase_set.name-immutable")
			}
		}
		if etag := strings.TrimSpace(req.GetPhraseSet().GetEtag()); etag != "" && etag != gcpSpeechV2Etag(phraseSetID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		phraseSet := gcpStage4SpeechV2PhraseSet(project, location, phraseSetID, false)
		gcpStage4SpeechV2ApplyPhraseSetOverrides(phraseSet, req.GetPhraseSet())
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "updatePhraseSet."+phraseSetID, path, phraseSet.GetName(), phraseSet))
	case gcpSpeechV2DeletePhraseSetMethod:
		req := &speechv2pb.DeletePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") && !req.GetAllowMissing() {
			return grpcNotFound("phrase_set-not-found")
		}
		if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSpeechV2Etag(phraseSetID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		phraseSet := gcpStage4SpeechV2PhraseSet(project, location, phraseSetID, true)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "deletePhraseSet."+phraseSetID, path, phraseSet.GetName(), phraseSet))
	case gcpSpeechV2UndeletePhraseSetMethod:
		req := &speechv2pb.UndeletePhraseSetRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, location, phraseSetID, ok := parseGCPSpeechPhraseSetName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if strings.Contains(strings.ToLower(phraseSetID), "missing") {
			return grpcNotFound("phrase_set-not-found")
		}
		if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSpeechV2Etag(phraseSetID) {
			return grpcFailedPrecondition("etag-mismatch")
		}
		phraseSet := gcpStage4SpeechV2PhraseSet(project, location, phraseSetID, false)
		return grpcProtoSuccess(gcpStage4SpeechV2Operation(project, location, "undeletePhraseSet."+phraseSetID, path, phraseSet.GetName(), phraseSet))
	default:
		return nil, "", "", false
	}
}

func gcpStage4SpeechV2ListPage(pageSize int32, pageToken string) (limit, start int, reason string, ok bool) {
	if pageSize < 0 || pageSize > 100 {
		return 0, 0, "page_size-invalid", false
	}
	limit = int(pageSize)
	if limit == 0 {
		limit = 5
	}
	token := strings.TrimSpace(pageToken)
	if token == "" {
		return limit, 0, "", true
	}
	start, ok = parseGCPStage4PageToken(token)
	if !ok {
		return 0, 0, "page_token-invalid", false
	}
	return limit, start, "", true
}

func gcpStage4SpeechV2Operation(project, location, operationID, method, resource string, response proto.Message) *longrunningpb.Operation {
	responseAny, err := anypb.New(response)
	if err != nil {
		return &longrunningpb.Operation{
			Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &statuspb.Status{Code: 3, Message: "response-any-encode-failed"},
			},
		}
	}
	metadataAny, err := anypb.New(&speechv2pb.OperationMetadata{
		CreateTime:      timestamppb.New(gcpStage4ReferenceTime.Add(-2 * time.Second)),
		UpdateTime:      timestamppb.New(gcpStage4ReferenceTime),
		Resource:        resource,
		Method:          method,
		ProgressPercent: 100,
	})
	if err != nil {
		return &longrunningpb.Operation{
			Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &statuspb.Status{Code: 3, Message: "metadata-any-encode-failed"},
			},
		}
	}
	return &longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Metadata: metadataAny,
		Done:     true,
		Result: &longrunningpb.Operation_Response{
			Response: responseAny,
		},
	}
}

func gcpStage4SpeechV2Recognizer(project, location, recognizerID string, deleted bool) *speechv2pb.Recognizer {
	state := speechv2pb.Recognizer_ACTIVE
	var deleteTime *timestamppb.Timestamp
	var expireTime *timestamppb.Timestamp
	if deleted {
		state = speechv2pb.Recognizer_DELETED
		deleteTime = timestamppb.New(gcpStage4ReferenceTime.Add(-5 * time.Minute))
		expireTime = timestamppb.New(gcpStage4ReferenceTime.Add(24 * time.Hour))
	}
	return &speechv2pb.Recognizer{
		Name:        fmt.Sprintf("projects/%s/locations/%s/recognizers/%s", project, location, recognizerID),
		Uid:         "uid-" + recognizerID,
		DisplayName: "Stackyard Recognizer " + recognizerID,
		Model:       "latest_long",
		LanguageCodes: []string{
			"en-US",
		},
		DefaultRecognitionConfig: &speechv2pb.RecognitionConfig{
			Model:         "latest_long",
			LanguageCodes: []string{"en-US"},
			Features: &speechv2pb.RecognitionFeatures{
				EnableAutomaticPunctuation: true,
			},
		},
		Annotations: map[string]string{
			"env": "staged",
		},
		State:       state,
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(-30 * time.Minute)),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime),
		DeleteTime:  deleteTime,
		ExpireTime:  expireTime,
		Etag:        gcpSpeechV2Etag(recognizerID),
		Reconciling: false,
	}
}

func gcpStage4SpeechV2ApplyRecognizerOverrides(out, req *speechv2pb.Recognizer) {
	if out == nil || req == nil {
		return
	}
	if displayName := strings.TrimSpace(req.GetDisplayName()); displayName != "" {
		out.DisplayName = displayName
	}
	if model := strings.TrimSpace(req.GetModel()); model != "" {
		out.Model = model
	}
	if len(req.GetLanguageCodes()) > 0 {
		out.LanguageCodes = req.GetLanguageCodes()
	}
	if req.GetDefaultRecognitionConfig() != nil {
		out.DefaultRecognitionConfig = req.GetDefaultRecognitionConfig()
	}
	if len(req.GetAnnotations()) > 0 {
		out.Annotations = req.GetAnnotations()
	}
}

func gcpStage4SpeechV2RecognizeResponse(project, location, languageCode, transcript string) *speechv2pb.RecognizeResponse {
	return &speechv2pb.RecognizeResponse{
		Results: []*speechv2pb.SpeechRecognitionResult{
			{
				Alternatives: []*speechv2pb.SpeechRecognitionAlternative{
					{
						Transcript: transcript,
						Confidence: 0.98,
					},
				},
				ChannelTag:      1,
				ResultEndOffset: durationpb.New(1200 * time.Millisecond),
				LanguageCode:    languageCode,
			},
		},
		Metadata: &speechv2pb.RecognitionResponseMetadata{
			RequestId:           fmt.Sprintf("speech-v2-%s-%s-req-1", project, location),
			TotalBilledDuration: durationpb.New(1200 * time.Millisecond),
		},
	}
}

func gcpStage4SpeechV2StreamingRecognizeResponse(project, location, languageCode, transcript string) *speechv2pb.StreamingRecognizeResponse {
	return &speechv2pb.StreamingRecognizeResponse{
		Results: []*speechv2pb.StreamingRecognitionResult{
			{
				Alternatives: []*speechv2pb.SpeechRecognitionAlternative{
					{
						Transcript: transcript,
						Confidence: 0.93,
					},
				},
				IsFinal:         true,
				Stability:       0.91,
				ResultEndOffset: durationpb.New(900 * time.Millisecond),
				ChannelTag:      1,
				LanguageCode:    languageCode,
			},
		},
		SpeechEventType:   speechv2pb.StreamingRecognizeResponse_END_OF_SINGLE_UTTERANCE,
		SpeechEventOffset: durationpb.New(900 * time.Millisecond),
		Metadata: &speechv2pb.RecognitionResponseMetadata{
			RequestId:           fmt.Sprintf("speech-v2-%s-%s-stream-1", project, location),
			TotalBilledDuration: durationpb.New(900 * time.Millisecond),
		},
	}
}

func gcpStage4SpeechV2BatchRecognizeResponse(uris []string) *speechv2pb.BatchRecognizeResponse {
	results := make(map[string]*speechv2pb.BatchRecognizeFileResult, len(uris))
	for i, uri := range uris {
		results[uri] = &speechv2pb.BatchRecognizeFileResult{
			Metadata: &speechv2pb.RecognitionResponseMetadata{
				RequestId:           fmt.Sprintf("speech-v2-batch-%d", i+1),
				TotalBilledDuration: durationpb.New(1200 * time.Millisecond),
			},
			Result: &speechv2pb.BatchRecognizeFileResult_InlineResult{
				InlineResult: &speechv2pb.InlineResult{
					Transcript: &speechv2pb.BatchRecognizeResults{
						Results: []*speechv2pb.SpeechRecognitionResult{
							{
								Alternatives: []*speechv2pb.SpeechRecognitionAlternative{
									{
										Transcript: fmt.Sprintf("stackyard transcript for %s", uri),
										Confidence: 0.95,
									},
								},
								ChannelTag:      1,
								ResultEndOffset: durationpb.New(1200 * time.Millisecond),
								LanguageCode:    "en-US",
							},
						},
						Metadata: &speechv2pb.RecognitionResponseMetadata{
							RequestId:           fmt.Sprintf("speech-v2-batch-inline-%d", i+1),
							TotalBilledDuration: durationpb.New(1200 * time.Millisecond),
						},
					},
				},
			},
		}
	}
	return &speechv2pb.BatchRecognizeResponse{
		Results:             results,
		TotalBilledDuration: durationpb.New(1200 * time.Millisecond),
	}
}

func gcpStage4SpeechV2Config(project, location string) *speechv2pb.Config {
	return &speechv2pb.Config{
		Name:       fmt.Sprintf("projects/%s/locations/%s/config", project, location),
		KmsKeyName: fmt.Sprintf("projects/%s/locations/%s/keyRings/stackyard/cryptoKeys/speech-v2", project, location),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4SpeechV2CustomClass(project, location, customClassID string, deleted bool) *speechv2pb.CustomClass {
	state := speechv2pb.CustomClass_ACTIVE
	var deleteTime *timestamppb.Timestamp
	var expireTime *timestamppb.Timestamp
	if deleted {
		state = speechv2pb.CustomClass_DELETED
		deleteTime = timestamppb.New(gcpStage4ReferenceTime.Add(-5 * time.Minute))
		expireTime = timestamppb.New(gcpStage4ReferenceTime.Add(24 * time.Hour))
	}
	return &speechv2pb.CustomClass{
		Name:        fmt.Sprintf("projects/%s/locations/%s/customClasses/%s", project, location, customClassID),
		Uid:         "uid-" + customClassID,
		DisplayName: "Stackyard CustomClass " + customClassID,
		Items: []*speechv2pb.CustomClass_ClassItem{
			{Value: "stackyard"},
			{Value: "speech"},
		},
		State:      state,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(-30 * time.Minute)),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
		DeleteTime: deleteTime,
		ExpireTime: expireTime,
		Annotations: map[string]string{
			"env": "staged",
		},
		Etag:        gcpSpeechV2Etag(customClassID),
		Reconciling: false,
	}
}

func gcpStage4SpeechV2ApplyCustomClassOverrides(out, req *speechv2pb.CustomClass) {
	if out == nil || req == nil {
		return
	}
	if displayName := strings.TrimSpace(req.GetDisplayName()); displayName != "" {
		out.DisplayName = displayName
	}
	if len(req.GetItems()) > 0 {
		out.Items = req.GetItems()
	}
	if len(req.GetAnnotations()) > 0 {
		out.Annotations = req.GetAnnotations()
	}
}

func gcpStage4SpeechV2PhraseSet(project, location, phraseSetID string, deleted bool) *speechv2pb.PhraseSet {
	state := speechv2pb.PhraseSet_ACTIVE
	var deleteTime *timestamppb.Timestamp
	var expireTime *timestamppb.Timestamp
	if deleted {
		state = speechv2pb.PhraseSet_DELETED
		deleteTime = timestamppb.New(gcpStage4ReferenceTime.Add(-5 * time.Minute))
		expireTime = timestamppb.New(gcpStage4ReferenceTime.Add(24 * time.Hour))
	}
	return &speechv2pb.PhraseSet{
		Name:        fmt.Sprintf("projects/%s/locations/%s/phraseSets/%s", project, location, phraseSetID),
		Uid:         "uid-" + phraseSetID,
		DisplayName: "Stackyard PhraseSet " + phraseSetID,
		Phrases: []*speechv2pb.PhraseSet_Phrase{
			{Value: "stackyard"},
			{Value: "cloud emulation"},
		},
		Boost:      12.5,
		State:      state,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(-30 * time.Minute)),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
		DeleteTime: deleteTime,
		ExpireTime: expireTime,
		Annotations: map[string]string{
			"env": "staged",
		},
		Etag:        gcpSpeechV2Etag(phraseSetID),
		Reconciling: false,
	}
}

func gcpStage4SpeechV2ApplyPhraseSetOverrides(out, req *speechv2pb.PhraseSet) {
	if out == nil || req == nil {
		return
	}
	if displayName := strings.TrimSpace(req.GetDisplayName()); displayName != "" {
		out.DisplayName = displayName
	}
	if req.GetBoost() != 0 {
		out.Boost = req.GetBoost()
	}
	if len(req.GetPhrases()) > 0 {
		out.Phrases = req.GetPhrases()
	}
	if len(req.GetAnnotations()) > 0 {
		out.Annotations = req.GetAnnotations()
	}
}

func gcpStage4GRPCSpanner(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpannerCreateSessionMethod:
		req := &spannerpb.CreateSessionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerMissingResource(project, instance, database) {
			return grpcNotFound("database-not-found")
		}
		sessionID := gcpSpannerDefaultSessionID
		multiplexed := false
		if session := req.GetSession(); session != nil && strings.TrimSpace(session.GetName()) != "" {
			p, i, d, s, valid := parseGCPSpannerSessionName(session.GetName())
			if !valid {
				return grpcInvalidArgument("session.name-invalid")
			}
			if p != project || i != instance || d != database {
				return grpcInvalidArgument("session.name-must-match-database")
			}
			sessionID = s
		}
		if session := req.GetSession(); session != nil {
			multiplexed = session.GetMultiplexed()
		}
		return grpcProtoSuccess(gcpStage4SpannerSession(project, instance, database, sessionID, multiplexed))
	case gcpSpannerBatchCreateSessionsMethod:
		req := &spannerpb.BatchCreateSessionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerMissingResource(project, instance, database) {
			return grpcNotFound("database-not-found")
		}
		if req.GetSessionCount() <= 0 || req.GetSessionCount() > gcpSpannerMaxSessionCount {
			return grpcInvalidArgument("session_count-invalid")
		}
		multiplexed := false
		if req.GetSessionTemplate() != nil {
			multiplexed = req.GetSessionTemplate().GetMultiplexed()
			if name := strings.TrimSpace(req.GetSessionTemplate().GetName()); name != "" {
				p, i, d, _, valid := parseGCPSpannerSessionName(name)
				if !valid {
					return grpcInvalidArgument("session_template.name-invalid")
				}
				if p != project || i != instance || d != database {
					return grpcInvalidArgument("session_template.name-must-match-database")
				}
			}
		}
		sessions := make([]*spannerpb.Session, 0, req.GetSessionCount())
		for i := int32(1); i <= req.GetSessionCount(); i++ {
			sessions = append(sessions, gcpStage4SpannerSession(project, instance, database, fmt.Sprintf("s-%d", i), multiplexed))
		}
		return grpcProtoSuccess(&spannerpb.BatchCreateSessionsResponse{Session: sessions})
	case gcpSpannerGetSessionMethod:
		req := &spannerpb.GetSessionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerSession(project, instance, database, sessionID, strings.Contains(sessionID, "mux")))
	case gcpSpannerListSessionsMethod:
		req := &spannerpb.ListSessionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerMissingResource(project, instance, database) {
			return grpcNotFound("database-not-found")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*spannerpb.Session{
			gcpStage4SpannerSession(project, instance, database, "s-1", false),
			gcpStage4SpannerSession(project, instance, database, "s-2", false),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spannerpb.ListSessionsResponse{
			Sessions:      items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerDeleteSessionMethod:
		req := &spannerpb.DeleteSessionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerExecuteSQLMethod:
		req := &spannerpb.ExecuteSqlRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if strings.TrimSpace(req.GetSql()) == "" {
			return grpcInvalidArgument("sql-required")
		}
		if reason := gcpStage4SpannerValidateTransactionSelector(req.GetTransaction(), false); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if req.GetDataBoostEnabled() && len(req.GetPartitionToken()) == 0 {
			return grpcInvalidArgument("partition_token-required")
		}
		return grpcProtoSuccess(gcpStage4SpannerResultSet(sessionID))
	case gcpSpannerExecuteStreamingSQLMethod:
		req := &spannerpb.ExecuteSqlRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if strings.TrimSpace(req.GetSql()) == "" {
			return grpcInvalidArgument("sql-required")
		}
		if reason := gcpStage4SpannerValidateTransactionSelector(req.GetTransaction(), false); reason != "" {
			return grpcInvalidArgument(reason)
		}
		return grpcProtoSuccess(gcpStage4SpannerPartialResultSet(sessionID))
	case gcpSpannerExecuteBatchDMLMethod:
		req := &spannerpb.ExecuteBatchDmlRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if reason := gcpStage4SpannerValidateTransactionSelector(req.GetTransaction(), true); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if len(req.GetStatements()) == 0 {
			return grpcInvalidArgument("statements-required")
		}
		for _, statement := range req.GetStatements() {
			if strings.TrimSpace(statement.GetSql()) == "" {
				return grpcInvalidArgument("statement.sql-required")
			}
			if strings.Contains(strings.ToLower(statement.GetSql()), "abort") {
				return grpcAborted("transaction-aborted")
			}
		}
		return grpcProtoSuccess(gcpStage4SpannerExecuteBatchDMLResponse())
	case gcpSpannerReadMethod:
		req := &spannerpb.ReadRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if reason := gcpStage4SpannerValidateReadRequest(req, false); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if req.GetDataBoostEnabled() && len(req.GetPartitionToken()) == 0 {
			return grpcInvalidArgument("partition_token-required")
		}
		return grpcProtoSuccess(gcpStage4SpannerResultSet(sessionID))
	case gcpSpannerStreamingReadMethod:
		req := &spannerpb.ReadRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if reason := gcpStage4SpannerValidateReadRequest(req, false); reason != "" {
			return grpcInvalidArgument(reason)
		}
		return grpcProtoSuccess(gcpStage4SpannerPartialResultSet(sessionID))
	case gcpSpannerBeginTransactionMethod:
		req := &spannerpb.BeginTransactionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if req.GetOptions() == nil || req.GetOptions().GetMode() == nil {
			return grpcInvalidArgument("options-required")
		}
		return grpcProtoSuccess(gcpStage4SpannerTransaction(gcpSpannerTransactionIDForSession(sessionID)))
	case gcpSpannerCommitMethod:
		req := &spannerpb.CommitRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		txID := req.GetTransactionId()
		singleUse := req.GetSingleUseTransaction()
		if len(txID) == 0 && singleUse == nil {
			return grpcInvalidArgument("transaction-required")
		}
		if len(txID) > 0 && singleUse != nil {
			return grpcInvalidArgument("transaction-selector-conflict")
		}
		txToken := strings.ToLower(string(txID))
		if strings.Contains(txToken, "stale") {
			return grpcFailedPrecondition("transaction-stale")
		}
		if strings.Contains(txToken, "abort") {
			return grpcAborted("transaction-aborted")
		}
		return grpcProtoSuccess(gcpStage4SpannerCommitResponse(int64(len(req.GetMutations()))))
	case gcpSpannerRollbackMethod:
		req := &spannerpb.RollbackRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if len(req.GetTransactionId()) == 0 {
			return grpcInvalidArgument("transaction_id-required")
		}
		if strings.Contains(strings.ToLower(string(req.GetTransactionId())), "stale") {
			return grpcFailedPrecondition("transaction-stale")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerPartitionQueryMethod:
		req := &spannerpb.PartitionQueryRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if strings.TrimSpace(req.GetSql()) == "" {
			return grpcInvalidArgument("sql-required")
		}
		if reason := gcpStage4SpannerValidateTransactionSelector(req.GetTransaction(), true); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if reason := gcpStage4SpannerValidatePartitionOptions(req.GetPartitionOptions()); reason != "" {
			return grpcInvalidArgument(reason)
		}
		return grpcProtoSuccess(gcpStage4SpannerPartitionResponse(sessionID))
	case gcpSpannerPartitionReadMethod:
		req := &spannerpb.PartitionReadRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if strings.TrimSpace(req.GetTable()) == "" {
			return grpcInvalidArgument("table-required")
		}
		if len(req.GetColumns()) == 0 {
			return grpcInvalidArgument("columns-required")
		}
		if !gcpStage4SpannerValidKeySet(req.GetKeySet()) {
			return grpcInvalidArgument("key_set-required")
		}
		if reason := gcpStage4SpannerValidateTransactionSelector(req.GetTransaction(), true); reason != "" {
			return grpcInvalidArgument(reason)
		}
		if reason := gcpStage4SpannerValidatePartitionOptions(req.GetPartitionOptions()); reason != "" {
			return grpcInvalidArgument(reason)
		}
		return grpcProtoSuccess(gcpStage4SpannerPartitionResponse(sessionID))
	case gcpSpannerBatchWriteMethod:
		req := &spannerpb.BatchWriteRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetSession())
		if !ok {
			return grpcInvalidArgument("session-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		if len(req.GetMutationGroups()) == 0 {
			return grpcInvalidArgument("mutation_groups-required")
		}
		for _, group := range req.GetMutationGroups() {
			if group == nil || len(group.GetMutations()) == 0 {
				return grpcInvalidArgument("mutation_group.mutations-required")
			}
			for _, mutation := range group.GetMutations() {
				if !gcpStage4SpannerValidMutation(mutation) {
					return grpcInvalidArgument("mutation-operation-required")
				}
			}
		}
		if strings.Contains(strings.ToLower(sessionID), "abort") {
			return grpcAborted("batch-write-aborted")
		}
		return grpcProtoSuccess(gcpStage4SpannerBatchWriteResponse())
	default:
		return nil, "", "", false
	}
}

func gcpStage4GRPCSpannerAdapter(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpannerAdapterCreateSessionMethod:
		req := &adapterpb.CreateSessionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, ok := parseGCPSpannerAdapterDatabaseName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerMissingResource(project, instance, database) {
			return grpcNotFound("database-not-found")
		}
		if req.GetSession() == nil {
			return grpcInvalidArgument("session-required")
		}

		sessionID := gcpSpannerAdapterDefaultSessionID
		if name := strings.TrimSpace(req.GetSession().GetName()); name != "" {
			p, i, d, s, valid := parseGCPSpannerSessionName(name)
			if !valid {
				return grpcInvalidArgument("session.name-invalid")
			}
			if p != project || i != instance || d != database {
				return grpcInvalidArgument("session.name-must-match-parent")
			}
			sessionID = s
		}
		return grpcProtoSuccess(&adapterpb.Session{
			Name: fmt.Sprintf("projects/%s/instances/%s/databases/%s/sessions/%s", project, instance, database, sessionID),
		})
	case gcpSpannerAdapterAdaptMessageMethod:
		req := &adapterpb.AdaptMessageRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, database, sessionID, ok := parseGCPSpannerSessionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerMissingResource(project, instance, database, sessionID) {
			return grpcNotFound("session-not-found")
		}
		protocol := strings.TrimSpace(req.GetProtocol())
		if protocol == "" {
			return grpcInvalidArgument("protocol-required")
		}
		if strings.Contains(strings.ToLower(protocol), "unsupported") {
			return grpcFailedPrecondition("protocol-unsupported")
		}
		payload := req.GetPayload()
		if len(payload) == 0 {
			payload = []byte("stackyard-adapted-" + strings.ToLower(protocol))
		}
		return grpcProtoSuccess(&adapterpb.AdaptMessageResponse{
			Payload: payload,
			StateUpdates: map[string]string{
				"adapter":  "stackyard",
				"protocol": protocol,
				"session":  sessionID,
			},
			Last: true,
		})
	default:
		return nil, "", "", false
	}
}

func gcpStage4GRPCSpannerExecutor(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &executorpb.SpannerAsyncActionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetActionId() <= 0 {
		return grpcInvalidArgument("action_id-required")
	}
	action := req.GetAction()
	if action == nil || action.GetAction() == nil {
		return grpcInvalidArgument("action-required")
	}
	if databasePath := strings.TrimSpace(action.GetDatabasePath()); databasePath != "" {
		project, instance, database, ok := parseGCPSpannerDatabaseName(databasePath)
		if !ok {
			return grpcInvalidArgument("database_path-invalid")
		}
		if isGCPSpannerMissingResource(project, instance, database) {
			return grpcNotFound("database-not-found")
		}
	}

	outcome := &executorpb.SpannerActionOutcome{
		Status: &statuspb.Status{Code: 0, Message: "OK"},
	}

	switch act := action.GetAction().(type) {
	case *executorpb.SpannerAction_Start:
		// no-op
	case *executorpb.SpannerAction_Finish:
		if act.Finish == nil || act.Finish.GetMode() == executorpb.FinishTransactionAction_MODE_UNSPECIFIED {
			return grpcInvalidArgument("finish.mode-required")
		}
		outcome.CommitTime = timestamppb.New(gcpStage4ReferenceTime)
	case *executorpb.SpannerAction_Read:
		read := act.Read
		if read == nil || strings.TrimSpace(read.GetTable()) == "" {
			return grpcInvalidArgument("read.table-required")
		}
		if len(read.GetColumn()) == 0 {
			return grpcInvalidArgument("read.column-required")
		}
		if read.GetKeys() == nil {
			return grpcInvalidArgument("read.keys-required")
		}
		if read.GetLimit() < 0 {
			return grpcInvalidArgument("read.limit-invalid")
		}
		outcome.ReadResult = &executorpb.ReadResult{Table: read.GetTable()}
	case *executorpb.SpannerAction_Query:
		query := act.Query
		if query == nil || strings.TrimSpace(query.GetSql()) == "" {
			return grpcInvalidArgument("query.sql-required")
		}
		outcome.QueryResult = &executorpb.QueryResult{}
	case *executorpb.SpannerAction_Mutation:
		if act.Mutation == nil || len(act.Mutation.GetMod()) == 0 {
			return grpcInvalidArgument("mutation.mod-required")
		}
	case *executorpb.SpannerAction_Dml:
		if act.Dml == nil || act.Dml.GetUpdate() == nil || strings.TrimSpace(act.Dml.GetUpdate().GetSql()) == "" {
			return grpcInvalidArgument("dml.update.sql-required")
		}
		outcome.DmlRowsModified = []int64{1}
	case *executorpb.SpannerAction_BatchDml:
		if act.BatchDml == nil || len(act.BatchDml.GetUpdates()) == 0 {
			return grpcInvalidArgument("batch_dml.updates-required")
		}
		rows := make([]int64, 0, len(act.BatchDml.GetUpdates()))
		for _, update := range act.BatchDml.GetUpdates() {
			if strings.TrimSpace(update.GetSql()) == "" {
				return grpcInvalidArgument("batch_dml.update.sql-required")
			}
			rows = append(rows, 1)
		}
		outcome.DmlRowsModified = rows
	case *executorpb.SpannerAction_Write:
		if act.Write == nil || act.Write.GetMutation() == nil || len(act.Write.GetMutation().GetMod()) == 0 {
			return grpcInvalidArgument("write.mutation.mod-required")
		}
	case *executorpb.SpannerAction_PartitionedUpdate:
		if act.PartitionedUpdate == nil || act.PartitionedUpdate.GetUpdate() == nil || strings.TrimSpace(act.PartitionedUpdate.GetUpdate().GetSql()) == "" {
			return grpcInvalidArgument("partitioned_update.update.sql-required")
		}
		outcome.DmlRowsModified = []int64{7}
	case *executorpb.SpannerAction_Admin:
		if act.Admin == nil || act.Admin.GetAction() == nil {
			return grpcInvalidArgument("admin.action-required")
		}
		adminResult, grpcCode, reason := gcpStage4SpannerExecutorAdminResult(act.Admin)
		if grpcCode != "" {
			switch grpcCode {
			case "3":
				return grpcInvalidArgument(reason)
			case "5":
				return grpcNotFound(reason)
			case "9":
				return grpcFailedPrecondition(reason)
			case "6":
				return grpcAlreadyExists(reason)
			default:
				return grpcUnimplemented(reason)
			}
		}
		outcome.AdminResult = adminResult
	case *executorpb.SpannerAction_StartBatchTxn:
		if act.StartBatchTxn == nil || (len(act.StartBatchTxn.GetTid()) == 0 && act.StartBatchTxn.GetBatchTxnTime() == nil) {
			return grpcInvalidArgument("start_batch_txn.param-required")
		}
		outcome.BatchTxnId = []byte("batch-tx-1")
	case *executorpb.SpannerAction_CloseBatchTxn:
		// no-op
	case *executorpb.SpannerAction_GenerateDbPartitionsRead:
		if act.GenerateDbPartitionsRead == nil || act.GenerateDbPartitionsRead.GetRead() == nil {
			return grpcInvalidArgument("generate_db_partitions_read.read-required")
		}
		read := act.GenerateDbPartitionsRead.GetRead()
		if strings.TrimSpace(read.GetTable()) == "" {
			return grpcInvalidArgument("generate_db_partitions_read.read.table-required")
		}
		if len(read.GetColumn()) == 0 {
			return grpcInvalidArgument("generate_db_partitions_read.read.column-required")
		}
		table := read.GetTable()
		outcome.DbPartition = []*executorpb.BatchPartition{
			{
				Partition:      []byte("partition-1"),
				PartitionToken: []byte("token-1"),
				Table:          &table,
			},
		}
	case *executorpb.SpannerAction_GenerateDbPartitionsQuery:
		if act.GenerateDbPartitionsQuery == nil || act.GenerateDbPartitionsQuery.GetQuery() == nil || strings.TrimSpace(act.GenerateDbPartitionsQuery.GetQuery().GetSql()) == "" {
			return grpcInvalidArgument("generate_db_partitions_query.query.sql-required")
		}
		outcome.DbPartition = []*executorpb.BatchPartition{
			{
				Partition:      []byte("partition-query-1"),
				PartitionToken: []byte("token-query-1"),
			},
		}
	case *executorpb.SpannerAction_ExecutePartition:
		if act.ExecutePartition == nil || act.ExecutePartition.GetPartition() == nil {
			return grpcInvalidArgument("execute_partition.partition-required")
		}
		partition := act.ExecutePartition.GetPartition()
		if len(partition.GetPartition()) == 0 && len(partition.GetPartitionToken()) == 0 {
			return grpcInvalidArgument("execute_partition.partition-content-required")
		}
		table := partition.GetTable()
		if strings.TrimSpace(table) == "" {
			table = "Users"
		}
		outcome.ReadResult = &executorpb.ReadResult{Table: table}
	case *executorpb.SpannerAction_ExecuteChangeStreamQuery:
		if act.ExecuteChangeStreamQuery == nil {
			return grpcInvalidArgument("execute_change_stream_query-required")
		}
		if strings.TrimSpace(act.ExecuteChangeStreamQuery.GetName()) == "" {
			return grpcInvalidArgument("execute_change_stream_query.name-required")
		}
		if act.ExecuteChangeStreamQuery.GetStartTime() == nil {
			return grpcInvalidArgument("execute_change_stream_query.start_time-required")
		}
		outcome.ChangeStreamRecords = []*executorpb.ChangeStreamRecord{
			{
				Record: &executorpb.ChangeStreamRecord_Heartbeat{
					Heartbeat: &executorpb.HeartbeatRecord{
						HeartbeatTime: timestamppb.New(gcpStage4ReferenceTime),
					},
				},
			},
		}
	case *executorpb.SpannerAction_QueryCancellation:
		if act.QueryCancellation == nil {
			return grpcInvalidArgument("query_cancellation-required")
		}
		longRunningSQL := strings.TrimSpace(act.QueryCancellation.GetLongRunningSql())
		if longRunningSQL == "" {
			return grpcInvalidArgument("query_cancellation.long_running_sql-required")
		}
		if strings.TrimSpace(act.QueryCancellation.GetCancelQuery()) == "" {
			return grpcInvalidArgument("query_cancellation.cancel_query-required")
		}
		if strings.Contains(strings.ToLower(longRunningSQL), "already-cancelled") {
			return grpcFailedPrecondition("query-already-cancelled")
		}
		outcome.QueryResult = &executorpb.QueryResult{}
	default:
		return grpcUnimplemented("action-kind-not-implemented")
	}

	return grpcProtoSuccess(&executorpb.SpannerAsyncActionResponse{
		ActionId: req.GetActionId(),
		Outcome:  outcome,
	})
}

func gcpStage4SpannerExecutorAdminResult(action *executorpb.AdminAction) (*executorpb.AdminResult, string, string) {
	const (
		project    = "stackyard"
		instanceID = "stackyard-instance"
		databaseID = "stackyard-db"
		backupID   = "backup-1"
		configID   = "custom-stackyard-primary"
	)

	operationFixture := gcpStage4SpannerAdminInstanceOperation(project, instanceID, "spanner-executor-op-1", &emptypb.Empty{})
	switch action.GetAction().(type) {
	case *executorpb.AdminAction_CreateUserInstanceConfig,
		*executorpb.AdminAction_UpdateUserInstanceConfig,
		*executorpb.AdminAction_DeleteUserInstanceConfig,
		*executorpb.AdminAction_GetCloudInstanceConfig,
		*executorpb.AdminAction_ListInstanceConfigs:
		instanceConfig := gcpStage4SpannerAdminInstanceConfig(project, configID)
		return &executorpb.AdminResult{
			InstanceConfigResponse: &executorpb.CloudInstanceConfigResponse{
				ListedInstanceConfigs: []*spanneradmininstancepb.InstanceConfig{instanceConfig},
				InstanceConfig:        instanceConfig,
				NextPageToken:         "",
			},
		}, "", ""
	case *executorpb.AdminAction_CreateCloudInstance,
		*executorpb.AdminAction_UpdateCloudInstance,
		*executorpb.AdminAction_DeleteCloudInstance,
		*executorpb.AdminAction_ListCloudInstances,
		*executorpb.AdminAction_GetCloudInstance:
		instance := gcpStage4SpannerAdminInstanceInstance(project, instanceID)
		return &executorpb.AdminResult{
			InstanceResponse: &executorpb.CloudInstanceResponse{
				ListedInstances: []*spanneradmininstancepb.Instance{instance},
				Instance:        instance,
				NextPageToken:   "",
			},
		}, "", ""
	case *executorpb.AdminAction_CreateCloudDatabase,
		*executorpb.AdminAction_UpdateCloudDatabaseDdl,
		*executorpb.AdminAction_UpdateCloudDatabase,
		*executorpb.AdminAction_DropCloudDatabase,
		*executorpb.AdminAction_ListCloudDatabases,
		*executorpb.AdminAction_ListCloudDatabaseOperations,
		*executorpb.AdminAction_RestoreCloudDatabase,
		*executorpb.AdminAction_GetCloudDatabase,
		*executorpb.AdminAction_ChangeQuorumCloudDatabase:
		database := gcpStage4SpannerAdminDatabaseDatabase(project, instanceID, databaseID)
		return &executorpb.AdminResult{
			DatabaseResponse: &executorpb.CloudDatabaseResponse{
				ListedDatabases:          []*spanneradminpb.Database{database},
				ListedDatabaseOperations: []*longrunningpb.Operation{operationFixture},
				Database:                 database,
				NextPageToken:            "",
			},
		}, "", ""
	case *executorpb.AdminAction_CreateCloudBackup,
		*executorpb.AdminAction_CopyCloudBackup,
		*executorpb.AdminAction_GetCloudBackup,
		*executorpb.AdminAction_UpdateCloudBackup,
		*executorpb.AdminAction_DeleteCloudBackup,
		*executorpb.AdminAction_ListCloudBackups,
		*executorpb.AdminAction_ListCloudBackupOperations:
		backup := gcpStage4SpannerAdminDatabaseBackup(project, instanceID, backupID, databaseID)
		return &executorpb.AdminResult{
			BackupResponse: &executorpb.CloudBackupResponse{
				ListedBackups:          []*spanneradminpb.Backup{backup},
				ListedBackupOperations: []*longrunningpb.Operation{operationFixture},
				Backup:                 backup,
				NextPageToken:          "",
			},
		}, "", ""
	case *executorpb.AdminAction_GetOperation, *executorpb.AdminAction_CancelOperation:
		return &executorpb.AdminResult{
			OperationResponse: &executorpb.OperationResponse{
				ListedOperations: []*longrunningpb.Operation{operationFixture},
				Operation:        operationFixture,
				NextPageToken:    "",
			},
		}, "", ""
	default:
		return nil, "12", "admin-action-not-implemented"
	}
}

func gcpStage4GRPCSpannerAdminDatabase(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpannerAdminDatabaseListDatabasesMethod:
		req := &spanneradminpb.ListDatabasesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*spanneradminpb.Database{
			gcpStage4SpannerAdminDatabaseDatabase(project, instance, "stackyard-db"),
			gcpStage4SpannerAdminDatabaseDatabase(project, instance, "analytics-db"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradminpb.ListDatabasesResponse{
			Databases:     items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminDatabaseCreateDatabaseMethod:
		req := &spanneradminpb.CreateDatabaseRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		databaseID, ok := gcpSpannerAdminDatabaseIDFromCreateStatement(strings.TrimSpace(req.GetCreateStatement()))
		if !ok {
			return grpcInvalidArgument("create_statement-required")
		}
		if isGCPSpannerAdminDatabaseAlreadyExists(databaseID) {
			return grpcAlreadyExists("database-already-exists")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, "create-database-"+databaseID, gcpStage4SpannerAdminDatabaseDatabase(project, instance, databaseID)))
	case gcpSpannerAdminDatabaseGetDatabaseMethod:
		req := &spanneradminpb.GetDatabaseRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseDatabase(project, instance, databaseID))
	case gcpSpannerAdminDatabaseUpdateDatabaseMethod:
		req := &spanneradminpb.UpdateDatabaseRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetDatabase() == nil {
			return grpcInvalidArgument("database-required")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetDatabase().GetName())
		if !ok {
			return grpcInvalidArgument("database.name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		response := gcpStage4SpannerAdminDatabaseDatabase(project, instance, databaseID)
		response.EnableDropProtection = req.GetDatabase().GetEnableDropProtection()
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, "update-database-"+databaseID, response))
	case gcpSpannerAdminDatabaseUpdateDatabaseDDLMethod:
		req := &spanneradminpb.UpdateDatabaseDdlRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if len(req.GetStatements()) == 0 {
			return grpcInvalidArgument("statements-required")
		}
		operationID := strings.TrimSpace(req.GetOperationId())
		if operationID == "" {
			operationID = "update-ddl-" + databaseID
		}
		if isGCPSpannerAdminDatabaseAlreadyExists(operationID) {
			return grpcAlreadyExists("operation-already-exists")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, operationID, &emptypb.Empty{}))
	case gcpSpannerAdminDatabaseDropDatabaseMethod:
		req := &spanneradminpb.DropDatabaseRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminDatabaseGetDatabaseDDLMethod:
		req := &spanneradminpb.GetDatabaseDdlRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		return grpcProtoSuccess(&spanneradminpb.GetDatabaseDdlResponse{
			Statements: []string{
				fmt.Sprintf("CREATE TABLE Users_%s (UserId STRING(36) NOT NULL, Name STRING(256)) PRIMARY KEY (UserId)", databaseID),
				"CREATE INDEX UsersByName ON Users (Name)",
			},
		})
	case gcpSpannerAdminDatabaseSetIAMPolicyMethod:
		req := &iampb.SetIamPolicyRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		resource := strings.TrimSpace(req.GetResource())
		if resource == "" {
			return grpcInvalidArgument("resource-required")
		}
		if _, _, _, ok := parseGCPSpannerDatabaseName(resource); !ok {
			if _, _, _, ok := parseGCPSpannerAdminDatabaseBackupName(resource); !ok {
				return grpcInvalidArgument("resource-invalid")
			}
		}
		if isGCPSpannerAdminDatabaseMissingResource(resource) {
			return grpcNotFound("resource-not-found")
		}
		if req.GetPolicy() == nil {
			return grpcInvalidArgument("policy-required")
		}
		response := gcpStage4SpannerAdminDatabasePolicy(resource)
		response.Bindings = req.GetPolicy().GetBindings()
		if len(req.GetPolicy().GetEtag()) > 0 {
			response.Etag = req.GetPolicy().GetEtag()
		}
		return grpcProtoSuccess(response)
	case gcpSpannerAdminDatabaseGetIAMPolicyMethod:
		req := &iampb.GetIamPolicyRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		resource := strings.TrimSpace(req.GetResource())
		if resource == "" {
			return grpcInvalidArgument("resource-required")
		}
		if _, _, _, ok := parseGCPSpannerDatabaseName(resource); !ok {
			if _, _, _, ok := parseGCPSpannerAdminDatabaseBackupName(resource); !ok {
				return grpcInvalidArgument("resource-invalid")
			}
		}
		if isGCPSpannerAdminDatabaseMissingResource(resource) {
			return grpcNotFound("resource-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabasePolicy(resource))
	case gcpSpannerAdminDatabaseTestIAMPermissionsMethod:
		req := &iampb.TestIamPermissionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		resource := strings.TrimSpace(req.GetResource())
		if resource == "" {
			return grpcInvalidArgument("resource-required")
		}
		if _, _, _, ok := parseGCPSpannerDatabaseName(resource); !ok {
			if _, _, _, ok := parseGCPSpannerAdminDatabaseBackupName(resource); !ok {
				return grpcInvalidArgument("resource-invalid")
			}
		}
		if isGCPSpannerAdminDatabaseMissingResource(resource) {
			return grpcNotFound("resource-not-found")
		}
		if len(req.GetPermissions()) == 0 {
			return grpcInvalidArgument("permissions-required")
		}
		filtered := make([]string, 0, len(req.GetPermissions()))
		for _, permission := range req.GetPermissions() {
			if strings.Contains(permission, "spanner") {
				filtered = append(filtered, permission)
			}
		}
		return grpcProtoSuccess(&iampb.TestIamPermissionsResponse{Permissions: filtered})
	case gcpSpannerAdminDatabaseCreateBackupMethod:
		req := &spanneradminpb.CreateBackupRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		backupID := strings.TrimSpace(req.GetBackupId())
		if !isGCPSpannerAdminDatabaseIdentifier(backupID, 60) {
			return grpcInvalidArgument("backup_id-required")
		}
		if req.GetBackup() == nil {
			return grpcInvalidArgument("backup-required")
		}
		dbProject, dbInstance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetBackup().GetDatabase())
		if !ok {
			return grpcInvalidArgument("backup.database-required")
		}
		if dbProject != project || dbInstance != instance {
			return grpcInvalidArgument("backup.database-must-match-parent")
		}
		if req.GetBackup().GetExpireTime() == nil {
			return grpcInvalidArgument("backup.expire_time-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if isGCPSpannerAdminDatabaseAlreadyExists(backupID) {
			return grpcAlreadyExists("backup-already-exists")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, "create-backup-"+backupID, gcpStage4SpannerAdminDatabaseBackup(project, instance, backupID, databaseID)))
	case gcpSpannerAdminDatabaseCopyBackupMethod:
		req := &spanneradminpb.CopyBackupRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		backupID := strings.TrimSpace(req.GetBackupId())
		if !isGCPSpannerAdminDatabaseIdentifier(backupID, 60) {
			return grpcInvalidArgument("backup_id-required")
		}
		sourceProject, sourceInstance, sourceBackupID, ok := parseGCPSpannerAdminDatabaseBackupName(req.GetSourceBackup())
		if !ok {
			return grpcInvalidArgument("source_backup-required")
		}
		if sourceProject != project || sourceInstance != instance {
			return grpcInvalidArgument("source_backup-must-match-parent")
		}
		if req.GetExpireTime() == nil {
			return grpcInvalidArgument("expire_time-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(sourceBackupID) {
			return grpcNotFound("source-backup-not-found")
		}
		if strings.Contains(strings.ToLower(sourceBackupID), "creating") {
			return grpcFailedPrecondition("source-backup-not-ready")
		}
		if isGCPSpannerAdminDatabaseAlreadyExists(backupID) {
			return grpcAlreadyExists("backup-already-exists")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, "copy-backup-"+backupID, gcpStage4SpannerAdminDatabaseBackup(project, instance, backupID, "stackyard-db")))
	case gcpSpannerAdminDatabaseGetBackupMethod:
		req := &spanneradminpb.GetBackupRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, backupID, ok := parseGCPSpannerAdminDatabaseBackupName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, backupID) {
			return grpcNotFound("backup-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseBackup(project, instance, backupID, "stackyard-db"))
	case gcpSpannerAdminDatabaseUpdateBackupMethod:
		req := &spanneradminpb.UpdateBackupRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetBackup() == nil {
			return grpcInvalidArgument("backup-required")
		}
		project, instance, backupID, ok := parseGCPSpannerAdminDatabaseBackupName(req.GetBackup().GetName())
		if !ok {
			return grpcInvalidArgument("backup.name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, backupID) {
			return grpcNotFound("backup-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		response := gcpStage4SpannerAdminDatabaseBackup(project, instance, backupID, "stackyard-db")
		if req.GetBackup().GetExpireTime() != nil {
			response.ExpireTime = req.GetBackup().GetExpireTime()
		}
		return grpcProtoSuccess(response)
	case gcpSpannerAdminDatabaseDeleteBackupMethod:
		req := &spanneradminpb.DeleteBackupRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, backupID, ok := parseGCPSpannerAdminDatabaseBackupName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, backupID) {
			return grpcNotFound("backup-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminDatabaseListBackupsMethod:
		req := &spanneradminpb.ListBackupsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*spanneradminpb.Backup{
			gcpStage4SpannerAdminDatabaseBackup(project, instance, "backup-1", "stackyard-db"),
			gcpStage4SpannerAdminDatabaseBackup(project, instance, "backup-2", "analytics-db"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradminpb.ListBackupsResponse{
			Backups:       items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminDatabaseRestoreDatabaseMethod:
		req := &spanneradminpb.RestoreDatabaseRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		databaseID := strings.TrimSpace(req.GetDatabaseId())
		if !isGCPSpannerAdminDatabaseIdentifier(databaseID, 30) {
			return grpcInvalidArgument("database_id-required")
		}
		_, _, sourceBackupID, ok := parseGCPSpannerAdminDatabaseBackupName(req.GetBackup())
		if !ok {
			return grpcInvalidArgument("backup-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(sourceBackupID) {
			return grpcNotFound("backup-not-found")
		}
		if strings.Contains(strings.ToLower(sourceBackupID), "creating") {
			return grpcFailedPrecondition("backup-not-ready")
		}
		if isGCPSpannerAdminDatabaseAlreadyExists(databaseID) {
			return grpcAlreadyExists("database-already-exists")
		}
		response := gcpStage4SpannerAdminDatabaseDatabase(project, instance, databaseID)
		response.RestoreInfo = &spanneradminpb.RestoreInfo{
			SourceType: spanneradminpb.RestoreSourceType_BACKUP,
			SourceInfo: &spanneradminpb.RestoreInfo_BackupInfo{
				BackupInfo: &spanneradminpb.BackupInfo{
					Backup:         req.GetBackup(),
					SourceDatabase: fmt.Sprintf("projects/%s/instances/%s/databases/stackyard-db", project, instance),
					CreateTime:     timestamppb.New(gcpStage4ReferenceTime.Add(4 * time.Minute)),
					VersionTime:    timestamppb.New(gcpStage4ReferenceTime.Add(3 * time.Minute)),
				},
			},
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, "restore-database-"+databaseID, response))
	case gcpSpannerAdminDatabaseListDatabaseOperationsMethod:
		req := &spanneradminpb.ListDatabaseOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*longrunningpb.Operation{
			gcpStage4SpannerAdminDatabaseOperation(project, instance, "create-database-stackyard-db", &emptypb.Empty{}),
			gcpStage4SpannerAdminDatabaseOperation(project, instance, "update-ddl-stackyard-db", &emptypb.Empty{}),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradminpb.ListDatabaseOperationsResponse{
			Operations:    items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminDatabaseListBackupOperationsMethod:
		req := &spanneradminpb.ListBackupOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*longrunningpb.Operation{
			gcpStage4SpannerAdminDatabaseOperation(project, instance, "create-backup-backup-1", &emptypb.Empty{}),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradminpb.ListBackupOperationsResponse{
			Operations:    items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminDatabaseListDatabaseRolesMethod:
		req := &spanneradminpb.ListDatabaseRolesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*spanneradminpb.DatabaseRole{
			gcpStage4SpannerAdminDatabaseRole(project, instance, databaseID, "reader"),
			gcpStage4SpannerAdminDatabaseRole(project, instance, databaseID, "writer"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradminpb.ListDatabaseRolesResponse{
			DatabaseRoles: items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminDatabaseAddSplitPointsMethod:
		req := &spanneradminpb.AddSplitPointsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if len(req.GetSplitPoints()) == 0 {
			return grpcInvalidArgument("split_points-required")
		}
		return grpcProtoSuccess(&spanneradminpb.AddSplitPointsResponse{})
	case gcpSpannerAdminDatabaseCreateBackupScheduleMethod:
		req := &spanneradminpb.CreateBackupScheduleRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		scheduleID := strings.TrimSpace(req.GetBackupScheduleId())
		if !isGCPSpannerAdminDatabaseIdentifier(scheduleID, 60) {
			return grpcInvalidArgument("backup_schedule_id-required")
		}
		if req.GetBackupSchedule() == nil {
			return grpcInvalidArgument("backup_schedule-required")
		}
		if isGCPSpannerAdminDatabaseAlreadyExists(scheduleID) {
			return grpcAlreadyExists("backup-schedule-already-exists")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseBackupSchedule(project, instance, databaseID, scheduleID))
	case gcpSpannerAdminDatabaseGetBackupScheduleMethod:
		req := &spanneradminpb.GetBackupScheduleRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, scheduleID, ok := parseGCPSpannerAdminDatabaseBackupScheduleName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID, scheduleID) {
			return grpcNotFound("backup-schedule-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseBackupSchedule(project, instance, databaseID, scheduleID))
	case gcpSpannerAdminDatabaseUpdateBackupScheduleMethod:
		req := &spanneradminpb.UpdateBackupScheduleRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetBackupSchedule() == nil {
			return grpcInvalidArgument("backup_schedule-required")
		}
		project, instance, databaseID, scheduleID, ok := parseGCPSpannerAdminDatabaseBackupScheduleName(req.GetBackupSchedule().GetName())
		if !ok {
			return grpcInvalidArgument("backup_schedule.name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID, scheduleID) {
			return grpcNotFound("backup-schedule-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		response := gcpStage4SpannerAdminDatabaseBackupSchedule(project, instance, databaseID, scheduleID)
		if req.GetBackupSchedule().GetRetentionDuration() != nil {
			response.RetentionDuration = req.GetBackupSchedule().GetRetentionDuration()
		}
		return grpcProtoSuccess(response)
	case gcpSpannerAdminDatabaseDeleteBackupScheduleMethod:
		req := &spanneradminpb.DeleteBackupScheduleRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, scheduleID, ok := parseGCPSpannerAdminDatabaseBackupScheduleName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID, scheduleID) {
			return grpcNotFound("backup-schedule-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminDatabaseListBackupSchedulesMethod:
		req := &spanneradminpb.ListBackupSchedulesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*spanneradminpb.BackupSchedule{
			gcpStage4SpannerAdminDatabaseBackupSchedule(project, instance, databaseID, "daily-full"),
			gcpStage4SpannerAdminDatabaseBackupSchedule(project, instance, databaseID, "hourly-incremental"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradminpb.ListBackupSchedulesResponse{
			BackupSchedules: items[start:end],
			NextPageToken:   next,
		})
	case gcpSpannerAdminDatabaseInternalUpdateGraphOpMethod:
		req := &spanneradminpb.InternalUpdateGraphOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, databaseID, ok := parseGCPSpannerDatabaseName(req.GetDatabase())
		if !ok {
			return grpcInvalidArgument("database-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, databaseID) {
			return grpcNotFound("database-not-found")
		}
		if strings.TrimSpace(req.GetOperationId()) == "" {
			return grpcInvalidArgument("operation_id-required")
		}
		if strings.TrimSpace(req.GetVmIdentityToken()) == "" {
			return grpcInvalidArgument("vm_identity_token-required")
		}
		if req.GetProgress() < 0 || req.GetProgress() > 100 {
			return grpcInvalidArgument("progress-invalid")
		}
		return grpcProtoSuccess(&spanneradminpb.InternalUpdateGraphOperationResponse{})
	case gcpSpannerAdminDatabaseCancelOperationMethod:
		req := &longrunningpb.CancelOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, operationID, ok := parseGCPSpannerAdminDatabaseOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, operationID) {
			return grpcNotFound("operation-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminDatabaseDeleteOperationMethod:
		req := &longrunningpb.DeleteOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, operationID, ok := parseGCPSpannerAdminDatabaseOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, operationID) {
			return grpcNotFound("operation-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminDatabaseGetOperationMethod:
		req := &longrunningpb.GetOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, operationID, ok := parseGCPSpannerAdminDatabaseOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminDatabaseMissingResource(project, instance, operationID) {
			return grpcNotFound("operation-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminDatabaseOperation(project, instance, operationID, &emptypb.Empty{}))
	case gcpSpannerAdminDatabaseListOperationsMethod:
		req := &longrunningpb.ListOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instance, ok := parseGCPSpannerAdminDatabaseOperationsCollectionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if req.GetPageSize() < 0 || req.GetPageSize() > gcpSpannerAdminDatabaseMaxPageSize {
			return grpcInvalidArgument("page_size-invalid")
		}
		start, valid := parseGCPStage4PageToken(req.GetPageToken())
		if !valid {
			return grpcInvalidArgument("page_token-invalid")
		}
		items := []*longrunningpb.Operation{
			gcpStage4SpannerAdminDatabaseOperation(project, instance, "create-database-stackyard-db", &emptypb.Empty{}),
			gcpStage4SpannerAdminDatabaseOperation(project, instance, "create-backup-backup-1", &emptypb.Empty{}),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
			end = start + int(req.GetPageSize())
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&longrunningpb.ListOperationsResponse{
			Operations:    items[start:end],
			NextPageToken: next,
		})
	default:
		return nil, "", "", false
	}
}

func gcpStage4GRPCSpannerAdminInstance(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSpannerAdminInstanceListInstanceConfigsMethod:
		req := &spanneradmininstancepb.ListInstanceConfigsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, ok := parseGCPSpannerAdminInstanceProjectName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize, start, reason, ok := gcpStage4SpannerAdminInstanceParsePage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*spanneradmininstancepb.InstanceConfig{
			gcpStage4SpannerAdminInstanceConfig(project, "custom-stackyard-primary"),
			gcpStage4SpannerAdminInstanceConfig(project, "custom-stackyard-analytics"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradmininstancepb.ListInstanceConfigsResponse{
			InstanceConfigs: items[start:end],
			NextPageToken:   next,
		})
	case gcpSpannerAdminInstanceGetInstanceConfigMethod:
		req := &spanneradmininstancepb.GetInstanceConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, configID, ok := parseGCPSpannerAdminInstanceConfigName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, configID) {
			return grpcNotFound("instance-config-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceConfig(project, configID))
	case gcpSpannerAdminInstanceCreateInstanceConfigMethod:
		req := &spanneradmininstancepb.CreateInstanceConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, ok := parseGCPSpannerAdminInstanceProjectName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		configID := strings.TrimSpace(req.GetInstanceConfigId())
		if !isGCPSpannerAdminInstanceIdentifier(configID, 64) {
			return grpcInvalidArgument("instance_config_id-required")
		}
		if req.GetInstanceConfig() == nil {
			return grpcInvalidArgument("instance_config-required")
		}
		if isGCPSpannerAdminInstanceAlreadyExists(configID) {
			return grpcAlreadyExists("instance-config-already-exists")
		}
		response := gcpStage4SpannerAdminInstanceConfig(project, configID)
		if displayName := strings.TrimSpace(req.GetInstanceConfig().GetDisplayName()); displayName != "" {
			response.DisplayName = displayName
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, "stackyard-instance", "create-instance-config-"+configID, response))
	case gcpSpannerAdminInstanceUpdateInstanceConfigMethod:
		req := &spanneradmininstancepb.UpdateInstanceConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetInstanceConfig() == nil {
			return grpcInvalidArgument("instance_config-required")
		}
		project, configID, ok := parseGCPSpannerAdminInstanceConfigName(req.GetInstanceConfig().GetName())
		if !ok {
			return grpcInvalidArgument("instance_config.name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, configID) {
			return grpcNotFound("instance-config-not-found")
		}
		if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
			return grpcInvalidArgument("update_mask-required")
		}
		response := gcpStage4SpannerAdminInstanceConfig(project, configID)
		if displayName := strings.TrimSpace(req.GetInstanceConfig().GetDisplayName()); displayName != "" {
			response.DisplayName = displayName
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, "stackyard-instance", "update-instance-config-"+configID, response))
	case gcpSpannerAdminInstanceDeleteInstanceConfigMethod:
		req := &spanneradmininstancepb.DeleteInstanceConfigRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, configID, ok := parseGCPSpannerAdminInstanceConfigName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, configID) {
			return grpcNotFound("instance-config-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminInstanceListInstanceConfigOpsMethod:
		req := &spanneradmininstancepb.ListInstanceConfigOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, ok := parseGCPSpannerAdminInstanceProjectName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize, start, reason, ok := gcpStage4SpannerAdminInstanceParsePage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*longrunningpb.Operation{
			gcpStage4SpannerAdminInstanceOperation(project, "stackyard-instance", "create-instance-config", &emptypb.Empty{}),
			gcpStage4SpannerAdminInstanceOperation(project, "stackyard-instance", "update-instance-config", &emptypb.Empty{}),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradmininstancepb.ListInstanceConfigOperationsResponse{
			Operations:    items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminInstanceListInstancesMethod:
		req := &spanneradmininstancepb.ListInstancesRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, ok := parseGCPSpannerAdminInstanceProjectName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		pageSize, start, reason, ok := gcpStage4SpannerAdminInstanceParsePage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*spanneradmininstancepb.Instance{
			gcpStage4SpannerAdminInstanceInstance(project, "stackyard-instance"),
			gcpStage4SpannerAdminInstanceInstance(project, "analytics-instance"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradmininstancepb.ListInstancesResponse{
			Instances:     items[start:end],
			NextPageToken: next,
			Unreachable:   nil,
		})
	case gcpSpannerAdminInstanceGetInstanceMethod:
		req := &spanneradmininstancepb.GetInstanceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceInstance(project, instanceID))
	case gcpSpannerAdminInstanceCreateInstanceMethod:
		req := &spanneradmininstancepb.CreateInstanceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, ok := parseGCPSpannerAdminInstanceProjectName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		instanceID := strings.TrimSpace(req.GetInstanceId())
		if !isGCPSpannerAdminInstanceIdentifier(instanceID, 64) {
			return grpcInvalidArgument("instance_id-required")
		}
		if req.GetInstance() == nil {
			return grpcInvalidArgument("instance-required")
		}
		if strings.TrimSpace(req.GetInstance().GetConfig()) == "" {
			return grpcInvalidArgument("instance.config-required")
		}
		if strings.TrimSpace(req.GetInstance().GetDisplayName()) == "" {
			return grpcInvalidArgument("instance.display_name-required")
		}
		if isGCPSpannerAdminInstanceAlreadyExists(instanceID) {
			return grpcAlreadyExists("instance-already-exists")
		}
		response := gcpStage4SpannerAdminInstanceInstance(project, instanceID)
		response.DisplayName = req.GetInstance().GetDisplayName()
		if req.GetInstance().GetNodeCount() > 0 {
			response.NodeCount = req.GetInstance().GetNodeCount()
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, instanceID, "create-instance-"+instanceID, response))
	case gcpSpannerAdminInstanceUpdateInstanceMethod:
		req := &spanneradmininstancepb.UpdateInstanceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetInstance() == nil {
			return grpcInvalidArgument("instance-required")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetInstance().GetName())
		if !ok {
			return grpcInvalidArgument("instance.name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		if req.GetFieldMask() == nil || len(req.GetFieldMask().GetPaths()) == 0 {
			return grpcInvalidArgument("field_mask-required")
		}
		response := gcpStage4SpannerAdminInstanceInstance(project, instanceID)
		if displayName := strings.TrimSpace(req.GetInstance().GetDisplayName()); displayName != "" {
			response.DisplayName = displayName
		}
		if req.GetInstance().GetNodeCount() > 0 {
			response.NodeCount = req.GetInstance().GetNodeCount()
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, instanceID, "update-instance-"+instanceID, response))
	case gcpSpannerAdminInstanceDeleteInstanceMethod:
		req := &spanneradmininstancepb.DeleteInstanceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminInstanceListInstancePartitionsMethod:
		req := &spanneradmininstancepb.ListInstancePartitionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		pageSize, start, reason, ok := gcpStage4SpannerAdminInstanceParsePage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*spanneradmininstancepb.InstancePartition{
			gcpStage4SpannerAdminInstancePartition(project, instanceID, "partition-a"),
			gcpStage4SpannerAdminInstancePartition(project, instanceID, "partition-b"),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradmininstancepb.ListInstancePartitionsResponse{
			InstancePartitions: items[start:end],
			NextPageToken:      next,
		})
	case gcpSpannerAdminInstanceGetInstancePartitionMethod:
		req := &spanneradmininstancepb.GetInstancePartitionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, partitionID, ok := parseGCPSpannerAdminInstancePartitionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID, partitionID) {
			return grpcNotFound("instance-partition-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstancePartition(project, instanceID, partitionID))
	case gcpSpannerAdminInstanceCreateInstancePartitionMethod:
		req := &spanneradmininstancepb.CreateInstancePartitionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		partitionID := strings.TrimSpace(req.GetInstancePartitionId())
		if !isGCPSpannerAdminInstanceIdentifier(partitionID, 64) {
			return grpcInvalidArgument("instance_partition_id-required")
		}
		if req.GetInstancePartition() == nil {
			return grpcInvalidArgument("instance_partition-required")
		}
		if isGCPSpannerAdminInstanceAlreadyExists(partitionID) {
			return grpcAlreadyExists("instance-partition-already-exists")
		}
		response := gcpStage4SpannerAdminInstancePartition(project, instanceID, partitionID)
		if displayName := strings.TrimSpace(req.GetInstancePartition().GetDisplayName()); displayName != "" {
			response.DisplayName = displayName
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, instanceID, "create-instance-partition-"+partitionID, response))
	case gcpSpannerAdminInstanceUpdateInstancePartitionMethod:
		req := &spanneradmininstancepb.UpdateInstancePartitionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		if req.GetInstancePartition() == nil {
			return grpcInvalidArgument("instance_partition-required")
		}
		project, instanceID, partitionID, ok := parseGCPSpannerAdminInstancePartitionName(req.GetInstancePartition().GetName())
		if !ok {
			return grpcInvalidArgument("instance_partition.name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID, partitionID) {
			return grpcNotFound("instance-partition-not-found")
		}
		if req.GetFieldMask() == nil || len(req.GetFieldMask().GetPaths()) == 0 {
			return grpcInvalidArgument("field_mask-required")
		}
		response := gcpStage4SpannerAdminInstancePartition(project, instanceID, partitionID)
		if displayName := strings.TrimSpace(req.GetInstancePartition().GetDisplayName()); displayName != "" {
			response.DisplayName = displayName
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, instanceID, "update-instance-partition-"+partitionID, response))
	case gcpSpannerAdminInstanceDeleteInstancePartitionMethod:
		req := &spanneradmininstancepb.DeleteInstancePartitionRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, partitionID, ok := parseGCPSpannerAdminInstancePartitionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID, partitionID) {
			return grpcNotFound("instance-partition-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminInstanceListPartitionOpsMethod:
		req := &spanneradmininstancepb.ListInstancePartitionOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetParent())
		if !ok {
			return grpcInvalidArgument("parent-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		pageSize, start, reason, ok := gcpStage4SpannerAdminInstanceParsePage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*longrunningpb.Operation{
			gcpStage4SpannerAdminInstanceOperation(project, instanceID, "create-partition-"+instanceID, &emptypb.Empty{}),
			gcpStage4SpannerAdminInstanceOperation(project, instanceID, "update-partition-"+instanceID, &emptypb.Empty{}),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&spanneradmininstancepb.ListInstancePartitionOperationsResponse{
			Operations:    items[start:end],
			NextPageToken: next,
		})
	case gcpSpannerAdminInstanceMoveInstanceMethod:
		req := &spanneradmininstancepb.MoveInstanceRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, ok := parseGCPSpannerInstanceName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
			return grpcNotFound("instance-not-found")
		}
		targetConfig := strings.TrimSpace(req.GetTargetConfig())
		if targetConfig == "" {
			return grpcInvalidArgument("target_config-required")
		}
		if strings.HasSuffix(targetConfig, "/custom-stackyard-primary") {
			return grpcFailedPrecondition("instance-already-uses-target-config")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, instanceID, "move-instance-"+instanceID, &spanneradmininstancepb.MoveInstanceResponse{}))
	case gcpSpannerAdminInstanceSetIAMPolicyMethod:
		req := &iampb.SetIamPolicyRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		resource := strings.TrimSpace(req.GetResource())
		if resource == "" {
			return grpcInvalidArgument("resource-required")
		}
		if _, _, ok := parseGCPSpannerInstanceName(resource); !ok {
			return grpcInvalidArgument("resource-invalid")
		}
		if isGCPSpannerAdminInstanceMissingResource(resource) {
			return grpcNotFound("resource-not-found")
		}
		if req.GetPolicy() == nil {
			return grpcInvalidArgument("policy-required")
		}
		response := gcpStage4SpannerAdminInstancePolicy(resource)
		response.Bindings = req.GetPolicy().GetBindings()
		if len(req.GetPolicy().GetEtag()) > 0 {
			response.Etag = req.GetPolicy().GetEtag()
		}
		return grpcProtoSuccess(response)
	case gcpSpannerAdminInstanceGetIAMPolicyMethod:
		req := &iampb.GetIamPolicyRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		resource := strings.TrimSpace(req.GetResource())
		if resource == "" {
			return grpcInvalidArgument("resource-required")
		}
		if _, _, ok := parseGCPSpannerInstanceName(resource); !ok {
			return grpcInvalidArgument("resource-invalid")
		}
		if isGCPSpannerAdminInstanceMissingResource(resource) {
			return grpcNotFound("resource-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstancePolicy(resource))
	case gcpSpannerAdminInstanceTestIAMPermissionsMethod:
		req := &iampb.TestIamPermissionsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		resource := strings.TrimSpace(req.GetResource())
		if resource == "" {
			return grpcInvalidArgument("resource-required")
		}
		if _, _, ok := parseGCPSpannerInstanceName(resource); !ok {
			return grpcInvalidArgument("resource-invalid")
		}
		if isGCPSpannerAdminInstanceMissingResource(resource) {
			return grpcNotFound("resource-not-found")
		}
		if len(req.GetPermissions()) == 0 {
			return grpcInvalidArgument("permissions-required")
		}
		filtered := make([]string, 0, len(req.GetPermissions()))
		for _, permission := range req.GetPermissions() {
			if strings.Contains(permission, "spanner") {
				filtered = append(filtered, permission)
			}
		}
		return grpcProtoSuccess(&iampb.TestIamPermissionsResponse{Permissions: filtered})
	case gcpSpannerAdminInstanceCancelOperationMethod:
		req := &longrunningpb.CancelOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, operationID, ok := parseGCPSpannerAdminInstanceOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID, operationID) {
			return grpcNotFound("operation-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminInstanceDeleteOperationMethod:
		req := &longrunningpb.DeleteOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, operationID, ok := parseGCPSpannerAdminInstanceOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID, operationID) {
			return grpcNotFound("operation-not-found")
		}
		return grpcProtoSuccess(&emptypb.Empty{})
	case gcpSpannerAdminInstanceGetOperationMethod:
		req := &longrunningpb.GetOperationRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, instanceID, operationID, ok := parseGCPSpannerAdminInstanceOperationName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		if isGCPSpannerAdminInstanceMissingResource(project, instanceID, operationID) {
			return grpcNotFound("operation-not-found")
		}
		return grpcProtoSuccess(gcpStage4SpannerAdminInstanceOperation(project, instanceID, operationID, &emptypb.Empty{}))
	case gcpSpannerAdminInstanceListOperationsMethod:
		req := &longrunningpb.ListOperationsRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		project, ok := parseGCPSpannerAdminInstanceOperationsCollectionName(req.GetName())
		if !ok {
			return grpcInvalidArgument("name-required")
		}
		pageSize, start, reason, ok := gcpStage4SpannerAdminInstanceParsePage(req.GetPageSize(), req.GetPageToken())
		if !ok {
			return grpcInvalidArgument(reason)
		}
		items := []*longrunningpb.Operation{
			gcpStage4SpannerAdminInstanceOperation(project, "stackyard-instance", "create-project", &emptypb.Empty{}),
			gcpStage4SpannerAdminInstanceOperation(project, "stackyard-instance", "update-project", &emptypb.Empty{}),
		}
		if start > len(items) {
			return grpcInvalidArgument("page_token-out-of-range")
		}
		end := len(items)
		if pageSize > 0 && start+pageSize < end {
			end = start + pageSize
		}
		next := ""
		if end < len(items) {
			next = strconv.Itoa(end)
		}
		return grpcProtoSuccess(&longrunningpb.ListOperationsResponse{
			Operations:    items[start:end],
			NextPageToken: next,
		})
	default:
		return nil, "", "", false
	}
}

func gcpStage4SpannerAdminDatabaseOperation(project, instance, operationID string, response proto.Message) *longrunningpb.Operation {
	responseAny := &anypb.Any{}
	if response == nil {
		response = &emptypb.Empty{}
	}
	if packed, err := anypb.New(response); err == nil {
		responseAny = packed
	}
	metadataAny := &anypb.Any{}
	if packed, err := anypb.New(&emptypb.Empty{}); err == nil {
		metadataAny = packed
	}
	return &longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/instances/%s/operations/%s", project, instance, operationID),
		Done:     true,
		Metadata: metadataAny,
		Result: &longrunningpb.Operation_Response{
			Response: responseAny,
		},
	}
}

func gcpStage4SpannerAdminDatabaseDatabase(project, instance, databaseID string) *spanneradminpb.Database {
	return &spanneradminpb.Database{
		Name:                   fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID),
		State:                  spanneradminpb.Database_READY,
		CreateTime:             timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Minute)),
		VersionRetentionPeriod: "604800s",
		DefaultLeader:          "regional-us-central1",
		DatabaseDialect:        spanneradminpb.DatabaseDialect_GOOGLE_STANDARD_SQL,
		EnableDropProtection:   true,
		Reconciling:            false,
	}
}

func gcpStage4SpannerAdminDatabaseBackup(project, instance, backupID, databaseID string) *spanneradminpb.Backup {
	if strings.TrimSpace(databaseID) == "" {
		databaseID = "stackyard-db"
	}
	return &spanneradminpb.Backup{
		Name:              fmt.Sprintf("projects/%s/instances/%s/backups/%s", project, instance, backupID),
		Database:          fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, databaseID),
		State:             spanneradminpb.Backup_READY,
		CreateTime:        timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute)),
		ExpireTime:        timestamppb.New(gcpStage4ReferenceTime.Add(24 * time.Hour)),
		VersionTime:       timestamppb.New(gcpStage4ReferenceTime.Add(4 * time.Minute)),
		SizeBytes:         1024,
		FreeableSizeBytes: 1024,
		DatabaseDialect:   spanneradminpb.DatabaseDialect_GOOGLE_STANDARD_SQL,
	}
}

func gcpStage4SpannerAdminDatabaseBackupSchedule(project, instance, databaseID, scheduleID string) *spanneradminpb.BackupSchedule {
	return &spanneradminpb.BackupSchedule{
		Name:              fmt.Sprintf("projects/%s/instances/%s/databases/%s/backupSchedules/%s", project, instance, databaseID, scheduleID),
		RetentionDuration: durationpb.New(48 * time.Hour),
		Spec: &spanneradminpb.BackupScheduleSpec{
			ScheduleSpec: &spanneradminpb.BackupScheduleSpec_CronSpec{
				CronSpec: &spanneradminpb.CrontabSpec{
					Text: "0 */6 * * *",
				},
			},
		},
		BackupTypeSpec: &spanneradminpb.BackupSchedule_FullBackupSpec{
			FullBackupSpec: &spanneradminpb.FullBackupSpec{},
		},
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(6 * time.Minute)),
	}
}

func gcpStage4SpannerAdminDatabaseRole(project, instance, databaseID, roleID string) *spanneradminpb.DatabaseRole {
	return &spanneradminpb.DatabaseRole{
		Name: fmt.Sprintf("projects/%s/instances/%s/databases/%s/databaseRoles/%s", project, instance, databaseID, roleID),
	}
}

func gcpStage4SpannerAdminDatabasePolicy(resource string) *iampb.Policy {
	return &iampb.Policy{
		Version: 1,
		Etag:    []byte("etag-spanner-admin-database"),
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/spanner.databaseAdmin",
				Members: []string{"user:stackyard@example.com"},
			},
		},
		AuditConfigs: []*iampb.AuditConfig{
			{
				Service: resource,
			},
		},
	}
}

func gcpStage4SpannerAdminInstanceParsePage(pageSize int32, pageToken string) (int, int, string, bool) {
	if pageSize < 0 || pageSize > gcpSpannerAdminInstanceMaxPageSize {
		return 0, 0, "page_size-invalid", false
	}
	start, valid := parseGCPStage4PageToken(pageToken)
	if !valid {
		return 0, 0, "page_token-invalid", false
	}
	return int(pageSize), start, "", true
}

func gcpStage4SpannerAdminInstanceOperation(project, instanceID, operationID string, response proto.Message) *longrunningpb.Operation {
	responseAny := &anypb.Any{}
	if response == nil {
		response = &emptypb.Empty{}
	}
	if packed, err := anypb.New(response); err == nil {
		responseAny = packed
	}
	metadataAny := &anypb.Any{}
	if packed, err := anypb.New(&emptypb.Empty{}); err == nil {
		metadataAny = packed
	}
	return &longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/instances/%s/operations/%s", project, instanceID, operationID),
		Done:     true,
		Metadata: metadataAny,
		Result: &longrunningpb.Operation_Response{
			Response: responseAny,
		},
	}
}

func gcpStage4SpannerAdminInstanceConfig(project, configID string) *spanneradmininstancepb.InstanceConfig {
	if strings.TrimSpace(configID) == "" {
		configID = "custom-stackyard-primary"
	}
	return &spanneradmininstancepb.InstanceConfig{
		Name:                          fmt.Sprintf("projects/%s/instanceConfigs/%s", project, configID),
		DisplayName:                   "Stackyard User Managed Config",
		ConfigType:                    spanneradmininstancepb.InstanceConfig_USER_MANAGED,
		BaseConfig:                    fmt.Sprintf("projects/%s/instanceConfigs/regional-us-central1", project),
		Etag:                          "etag-instance-config-" + configID,
		LeaderOptions:                 []string{"default", "regional-us-central1"},
		Reconciling:                   false,
		State:                         spanneradmininstancepb.InstanceConfig_READY,
		QuorumType:                    spanneradmininstancepb.InstanceConfig_REGION,
		StorageLimitPerProcessingUnit: 4096,
		Replicas: []*spanneradmininstancepb.ReplicaInfo{
			{
				Location: "us-central1",
				Type:     spanneradmininstancepb.ReplicaInfo_READ_WRITE,
			},
		},
	}
}

func gcpStage4SpannerAdminInstanceInstance(project, instanceID string) *spanneradmininstancepb.Instance {
	if strings.TrimSpace(instanceID) == "" {
		instanceID = "stackyard-instance"
	}
	return &spanneradmininstancepb.Instance{
		Name:                      fmt.Sprintf("projects/%s/instances/%s", project, instanceID),
		Config:                    fmt.Sprintf("projects/%s/instanceConfigs/custom-stackyard-primary", project),
		DisplayName:               "Stackyard Instance",
		NodeCount:                 1,
		State:                     spanneradmininstancepb.Instance_READY,
		InstanceType:              spanneradmininstancepb.Instance_PROVISIONED,
		EndpointUris:              []string{"spanner.googleapis.com"},
		CreateTime:                timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Minute)),
		UpdateTime:                timestamppb.New(gcpStage4ReferenceTime.Add(4 * time.Minute)),
		Edition:                   spanneradmininstancepb.Instance_ENTERPRISE,
		DefaultBackupScheduleType: spanneradmininstancepb.Instance_AUTOMATIC,
	}
}

func gcpStage4SpannerAdminInstancePartition(project, instanceID, partitionID string) *spanneradmininstancepb.InstancePartition {
	if strings.TrimSpace(partitionID) == "" {
		partitionID = "partition-a"
	}
	return &spanneradmininstancepb.InstancePartition{
		Name:        fmt.Sprintf("projects/%s/instances/%s/instancePartitions/%s", project, instanceID, partitionID),
		Config:      fmt.Sprintf("projects/%s/instanceConfigs/custom-stackyard-primary", project),
		DisplayName: "Stackyard Partition",
		ComputeCapacity: &spanneradmininstancepb.InstancePartition_ProcessingUnits{
			ProcessingUnits: 1000,
		},
		State:      spanneradmininstancepb.InstancePartition_READY,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(3 * time.Minute)),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute)),
		Etag:       "etag-instance-partition-" + partitionID,
	}
}

func gcpStage4SpannerAdminInstancePolicy(resource string) *iampb.Policy {
	return &iampb.Policy{
		Version: 1,
		Etag:    []byte("etag-spanner-admin-instance-policy"),
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/spanner.admin",
				Members: []string{"user:stackyard@example.com"},
			},
		},
		AuditConfigs: []*iampb.AuditConfig{
			{
				Service: resource,
			},
		},
	}
}

func parseGCPSpannerAdminDatabaseOperationsCollectionName(name string) (project, instance string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 5 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instance = strings.TrimSpace(parts[3])
	if project == "" || instance == "" {
		return "", "", false
	}
	return project, instance, true
}

func gcpStage4SpannerValidateTransactionSelector(selector *spannerpb.TransactionSelector, required bool) string {
	if selector == nil {
		if required {
			return "transaction-required"
		}
		return ""
	}
	count := 0
	if id := selector.GetId(); len(id) > 0 {
		count++
	}
	if singleUse := selector.GetSingleUse(); singleUse != nil {
		count++
	}
	if begin := selector.GetBegin(); begin != nil {
		count++
	}
	if count == 0 {
		return "transaction-selector-required"
	}
	if count > 1 {
		return "transaction-selector-must-be-exclusive"
	}
	return ""
}

func gcpStage4SpannerValidateReadRequest(req *spannerpb.ReadRequest, requireTransaction bool) string {
	if strings.TrimSpace(req.GetTable()) == "" {
		return "table-required"
	}
	if len(req.GetColumns()) == 0 {
		return "columns-required"
	}
	if !gcpStage4SpannerValidKeySet(req.GetKeySet()) {
		return "key_set-required"
	}
	if reason := gcpStage4SpannerValidateTransactionSelector(req.GetTransaction(), requireTransaction); reason != "" {
		return reason
	}
	return ""
}

func gcpStage4SpannerValidatePartitionOptions(options *spannerpb.PartitionOptions) string {
	if options == nil {
		return ""
	}
	if options.GetMaxPartitions() < 0 || options.GetMaxPartitions() > gcpSpannerMaxPartitions {
		return "partition_options.max_partitions-invalid"
	}
	if options.GetPartitionSizeBytes() < 0 {
		return "partition_options.partition_size_bytes-invalid"
	}
	return ""
}

func gcpStage4SpannerValidKeySet(keySet *spannerpb.KeySet) bool {
	if keySet == nil {
		return false
	}
	return keySet.GetAll() || len(keySet.GetKeys()) > 0 || len(keySet.GetRanges()) > 0
}

func gcpStage4SpannerValidMutation(mutation *spannerpb.Mutation) bool {
	if mutation == nil {
		return false
	}
	return mutation.GetInsert() != nil ||
		mutation.GetUpdate() != nil ||
		mutation.GetInsertOrUpdate() != nil ||
		mutation.GetReplace() != nil ||
		mutation.GetDelete() != nil
}

func gcpStage4SpannerSession(project, instance, database, sessionID string, multiplexed bool) *spannerpb.Session {
	return &spannerpb.Session{
		Name: gcpSpannerSessionResourceName(project, instance, database, sessionID),
		Labels: map[string]string{
			"env": "staged",
		},
		CreateTime:             timestamppb.New(gcpStage4ReferenceTime),
		ApproximateLastUseTime: timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Minute)),
		CreatorRole:            "roles/spanner.databaseUser",
		Multiplexed:            multiplexed,
	}
}

func gcpStage4SpannerTransaction(txID string) *spannerpb.Transaction {
	return &spannerpb.Transaction{
		Id:            []byte(txID),
		ReadTimestamp: timestamppb.New(gcpStage4ReferenceTime.Add(10 * time.Second)),
	}
}

func gcpStage4SpannerResultSetMetadata(txID string) *spannerpb.ResultSetMetadata {
	return &spannerpb.ResultSetMetadata{
		RowType: &spannerpb.StructType{
			Fields: []*spannerpb.StructType_Field{
				{Name: "id", Type: &spannerpb.Type{Code: spannerpb.TypeCode_INT64}},
				{Name: "value", Type: &spannerpb.Type{Code: spannerpb.TypeCode_STRING}},
			},
		},
		Transaction: gcpStage4SpannerTransaction(txID),
	}
}

func gcpStage4SpannerResultSet(sessionID string) *spannerpb.ResultSet {
	return &spannerpb.ResultSet{
		Metadata: gcpStage4SpannerResultSetMetadata(gcpSpannerTransactionIDForSession(sessionID)),
		Rows: []*structpb.ListValue{
			{Values: []*structpb.Value{
				structpb.NewStringValue("1"),
				structpb.NewStringValue("stackyard"),
			}},
		},
		Stats: &spannerpb.ResultSetStats{
			RowCount: &spannerpb.ResultSetStats_RowCountExact{RowCountExact: 1},
		},
	}
}

func gcpStage4SpannerPartialResultSet(sessionID string) *spannerpb.PartialResultSet {
	return &spannerpb.PartialResultSet{
		Metadata: gcpStage4SpannerResultSetMetadata(gcpSpannerTransactionIDForSession(sessionID)),
		Values: []*structpb.Value{
			structpb.NewStringValue("1"),
			structpb.NewStringValue("stackyard"),
		},
		ChunkedValue: false,
		ResumeToken:  []byte("resume-1"),
		Last:         true,
	}
}

func gcpStage4SpannerExecuteBatchDMLResponse() *spannerpb.ExecuteBatchDmlResponse {
	return &spannerpb.ExecuteBatchDmlResponse{
		ResultSets: []*spannerpb.ResultSet{
			{
				Metadata: &spannerpb.ResultSetMetadata{
					RowType: &spannerpb.StructType{
						Fields: []*spannerpb.StructType_Field{
							{Name: "row_count", Type: &spannerpb.Type{Code: spannerpb.TypeCode_INT64}},
						},
					},
				},
				Stats: &spannerpb.ResultSetStats{
					RowCount: &spannerpb.ResultSetStats_RowCountExact{RowCountExact: 1},
				},
			},
		},
		Status: &statuspb.Status{
			Code:    0,
			Message: "OK",
		},
	}
}

func gcpStage4SpannerCommitResponse(mutationCount int64) *spannerpb.CommitResponse {
	if mutationCount < 0 {
		mutationCount = 0
	}
	return &spannerpb.CommitResponse{
		CommitTimestamp: timestamppb.New(gcpStage4ReferenceTime.Add(30 * time.Second)),
		CommitStats: &spannerpb.CommitResponse_CommitStats{
			MutationCount: mutationCount,
		},
	}
}

func gcpStage4SpannerPartitionResponse(sessionID string) *spannerpb.PartitionResponse {
	return &spannerpb.PartitionResponse{
		Partitions: []*spannerpb.Partition{
			{PartitionToken: []byte("partition-1")},
			{PartitionToken: []byte("partition-2")},
		},
		Transaction: gcpStage4SpannerTransaction(gcpSpannerTransactionIDForSession(sessionID)),
	}
}

func gcpStage4SpannerBatchWriteResponse() *spannerpb.BatchWriteResponse {
	return &spannerpb.BatchWriteResponse{
		Indexes: []int32{0},
		Status: &statuspb.Status{
			Code:    0,
			Message: "OK",
		},
		CommitTimestamp: timestamppb.New(gcpStage4ReferenceTime.Add(45 * time.Second)),
	}
}

func gcpStage4GRPCShellGetEnvironment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shellpb.GetEnvironmentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	owner, environmentID, ok := parseGCPShellEnvironmentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ShellEnvironment(owner, environmentID, []string{gcpShellDefaultPublicKey()}))
}

func gcpStage4GRPCShellStartEnvironment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shellpb.StartEnvironmentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	owner, environmentID, ok := parseGCPShellEnvironmentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	for _, key := range req.GetPublicKeys() {
		if !isGCPShellPublicKey(key) {
			return grpcInvalidArgument("public_keys-invalid")
		}
	}
	environmentName := gcpShellEnvironmentName(owner, environmentID)
	return grpcProtoSuccess(gcpStage4ShellOperation("start", environmentName, ""))
}

func gcpStage4GRPCShellAuthorizeEnvironment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shellpb.AuthorizeEnvironmentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	owner, environmentID, ok := parseGCPShellEnvironmentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(req.GetAccessToken()) == "" && strings.TrimSpace(req.GetIdToken()) == "" {
		return grpcInvalidArgument("token-required")
	}
	if expireTime := req.GetExpireTime(); expireTime != nil {
		if err := expireTime.CheckValid(); err != nil {
			return grpcInvalidArgument("expire_time-invalid")
		}
	}
	environmentName := gcpShellEnvironmentName(owner, environmentID)
	return grpcProtoSuccess(gcpStage4ShellOperation("authorize", environmentName, ""))
}

func gcpStage4GRPCShellAddPublicKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shellpb.AddPublicKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	owner, environmentID, ok := parseGCPShellEnvironmentName(req.GetEnvironment())
	if !ok {
		return grpcInvalidArgument("environment-required")
	}
	key := strings.TrimSpace(req.GetKey())
	if key == "" {
		return grpcInvalidArgument("key-required")
	}
	if !isGCPShellPublicKey(key) {
		return grpcInvalidArgument("key-invalid")
	}
	if isGCPShellDuplicateKey(key) {
		return grpcAlreadyExists("key-already-exists")
	}
	environmentName := gcpShellEnvironmentName(owner, environmentID)
	return grpcProtoSuccess(gcpStage4ShellOperation("add-public-key", environmentName, key))
}

func gcpStage4GRPCShellRemovePublicKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shellpb.RemovePublicKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	owner, environmentID, ok := parseGCPShellEnvironmentName(req.GetEnvironment())
	if !ok {
		return grpcInvalidArgument("environment-required")
	}
	key := strings.TrimSpace(req.GetKey())
	if key == "" {
		return grpcInvalidArgument("key-required")
	}
	if !isGCPShellPublicKey(key) {
		return grpcInvalidArgument("key-invalid")
	}
	if isGCPShellMissingKey(key) {
		return grpcNotFound("key-not-found")
	}
	environmentName := gcpShellEnvironmentName(owner, environmentID)
	return grpcProtoSuccess(gcpStage4ShellOperation("remove-public-key", environmentName, key))
}

func gcpStage4ShellOperation(action, environmentName, key string) *longrunningpb.Operation {
	op := &longrunningpb.Operation{
		Name: "operations/" + gcpShellOperationID(action, environmentName, key),
		Done: true,
	}

	owner, environmentID, ok := parseGCPShellEnvironmentName(environmentName)
	if !ok {
		owner = "me"
		environmentID = "default"
	}

	var metadata proto.Message
	var response proto.Message
	switch action {
	case "start":
		metadata = &shellpb.StartEnvironmentMetadata{
			State: shellpb.StartEnvironmentMetadata_FINISHED,
		}
		response = &shellpb.StartEnvironmentResponse{
			Environment: gcpStage4ShellEnvironment(owner, environmentID, []string{gcpShellDefaultPublicKey()}),
		}
	case "authorize":
		metadata = &shellpb.AuthorizeEnvironmentMetadata{}
		response = &shellpb.AuthorizeEnvironmentResponse{}
	case "add-public-key":
		if strings.TrimSpace(key) == "" {
			key = gcpShellDefaultPublicKey()
		}
		metadata = &shellpb.AddPublicKeyMetadata{}
		response = &shellpb.AddPublicKeyResponse{Key: key}
	case "remove-public-key":
		metadata = &shellpb.RemovePublicKeyMetadata{}
		response = &shellpb.RemovePublicKeyResponse{}
	}

	if metadataAny, err := anypb.New(metadata); err == nil {
		op.Metadata = metadataAny
	}
	if responseAny, err := anypb.New(response); err == nil {
		op.Result = &longrunningpb.Operation_Response{
			Response: responseAny,
		}
	}
	return op
}

func gcpStage4ShellEnvironment(owner, environmentID string, publicKeys []string) *shellpb.Environment {
	if len(publicKeys) == 0 {
		publicKeys = []string{gcpShellDefaultPublicKey()}
	}
	return &shellpb.Environment{
		Name:        gcpShellEnvironmentName(owner, environmentID),
		Id:          environmentID,
		DockerImage: "gcr.io/dev-con/cloud-devshell:latest",
		State:       shellpb.Environment_RUNNING,
		WebHost:     "ssh.cloud.google.com",
		SshUsername: "stackyard",
		SshHost:     "34.1.2.3",
		SshPort:     6000,
		PublicKeys:  publicKeys,
	}
}

func gcpStage4GRPCShoppingCSSListChildAccounts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.ListChildAccountsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*shoppingcsspb.Account{
		gcpStage4ShoppingCSSAccount(account+"-child-1", []int64{1001}),
		gcpStage4ShoppingCSSAccount(account+"-child-2", []int64{1002}),
	}

	if req.FullName != nil && strings.TrimSpace(req.GetFullName()) != "" {
		filtered := make([]*shoppingcsspb.Account, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(item.GetFullName()), strings.TrimSpace(req.GetFullName())) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if req.LabelId != nil && req.GetLabelId() > 0 {
		filtered := make([]*shoppingcsspb.Account, 0, len(items))
		for _, item := range items {
			for _, labelID := range item.GetLabelIds() {
				if labelID == req.GetLabelId() {
					filtered = append(filtered, item)
					break
				}
			}
		}
		items = filtered
	}

	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&shoppingcsspb.ListChildAccountsResponse{
		Accounts:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCShoppingCSSGetAccount(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.GetAccountRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if parent := strings.TrimSpace(req.GetParent()); parent != "" {
		if _, valid := parseGCPShoppingCSSAccountName(parent); !valid {
			return grpcInvalidArgument("parent-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSAccount(account, []int64{1001, 1002}))
}

func gcpStage4GRPCShoppingCSSUpdateLabels(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.UpdateAccountLabelsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if parent := strings.TrimSpace(req.GetParent()); parent != "" {
		if _, valid := parseGCPShoppingCSSAccountName(parent); !valid {
			return grpcInvalidArgument("parent-invalid")
		}
	}
	if len(req.GetLabelIds()) == 0 {
		return grpcInvalidArgument("label_ids-required")
	}
	for _, id := range req.GetLabelIds() {
		if id < 0 {
			return grpcInvalidArgument("label_ids-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSAccount(account, req.GetLabelIds()))
}

func gcpStage4GRPCShoppingCSSListAccountLabels(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.ListAccountLabelsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*shoppingcsspb.AccountLabel{
		gcpStage4ShoppingCSSAccountLabel(account, "label-1", "Stackyard Label 1", "Default staged label"),
		gcpStage4ShoppingCSSAccountLabel(account, "label-2", "Stackyard Label 2", "Secondary staged label"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&shoppingcsspb.ListAccountLabelsResponse{
		AccountLabels: items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCShoppingCSSCreateAccountLabel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.CreateAccountLabelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	label := req.GetAccountLabel()
	if label == nil {
		return grpcInvalidArgument("account_label-required")
	}
	displayName := strings.TrimSpace(label.GetDisplayName())
	if displayName == "" {
		return grpcInvalidArgument("display_name-required")
	}
	labelID := gcpShoppingCSSLabelIDFromBody(map[string]any{
		"name":        strings.TrimSpace(label.GetName()),
		"displayName": displayName,
	}, displayName)
	if !isGCPShoppingCSSLabelID(labelID) {
		return grpcInvalidArgument("account_label.name-invalid")
	}
	if gcpShoppingCSSLabelDuplicate(displayName, labelID) {
		return grpcAlreadyExists("label-already-exists")
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSAccountLabel(account, labelID, displayName, label.GetDescription()))
}

func gcpStage4GRPCShoppingCSSUpdateAccountLabel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.UpdateAccountLabelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	label := req.GetAccountLabel()
	if label == nil {
		return grpcInvalidArgument("account_label-required")
	}
	account, labelID, ok := parseGCPShoppingCSSLabelName(label.GetName())
	if !ok {
		return grpcInvalidArgument("account_label.name-required")
	}
	if gcpShoppingCSSLabelMissing(labelID) {
		return grpcNotFound("label-not-found")
	}
	displayName := strings.TrimSpace(label.GetDisplayName())
	if displayName == "" {
		return grpcInvalidArgument("display_name-required")
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSAccountLabel(account, labelID, displayName, label.GetDescription()))
}

func gcpStage4GRPCShoppingCSSDeleteAccountLabel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.DeleteAccountLabelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, labelID, ok := parseGCPShoppingCSSLabelName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpShoppingCSSLabelMissing(labelID) {
		return grpcNotFound("label-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCShoppingCSSGetCssProduct(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.GetCssProductRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, productID, ok := parseGCPShoppingCSSCssProductName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSCssProduct(account, productID, "Stackyard Product "+productID))
}

func gcpStage4GRPCShoppingCSSListCssProducts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.ListCssProductsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*shoppingcsspb.CssProduct{
		gcpStage4ShoppingCSSCssProduct(account, "en~US~sku-1", "Stackyard Tee"),
		gcpStage4ShoppingCSSCssProduct(account, "en~US~sku-2", "Stackyard Hoodie"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&shoppingcsspb.ListCssProductsResponse{
		CssProducts:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCShoppingCSSInsertCssProductInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.InsertCssProductInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetFeedId() < 0 {
		return grpcInvalidArgument("feed_id-invalid")
	}
	input := req.GetCssProductInput()
	if reason := gcpStage4ShoppingCSSValidateInput(input, false); reason != "" {
		return grpcInvalidArgument(reason)
	}
	if name := strings.TrimSpace(input.GetName()); name != "" {
		parsedAccount, _, valid := parseGCPShoppingCSSCssProductInputName(name)
		if !valid {
			return grpcInvalidArgument("css_product_input.name-invalid")
		}
		if parsedAccount != account {
			return grpcInvalidArgument("css_product_input.name-must-match-parent")
		}
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(input.GetRawProvidedId())), "existing") {
		return grpcAlreadyExists("input-already-exists")
	}
	inputID := strings.TrimSpace(input.GetName())
	if _, parsedInputID, valid := parseGCPShoppingCSSCssProductInputName(inputID); valid {
		inputID = parsedInputID
	} else {
		inputID = fmt.Sprintf("%s~%s~%s", strings.TrimSpace(input.GetContentLanguage()), strings.ToUpper(strings.TrimSpace(input.GetFeedLabel())), strings.TrimSpace(input.GetRawProvidedId()))
	}
	title := ""
	if attrs := input.GetAttributes(); attrs != nil {
		title = strings.TrimSpace(attrs.GetTitle())
	}
	if title == "" {
		title = "Stackyard Product " + strings.TrimSpace(input.GetRawProvidedId())
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSCssProductInput(account, inputID, title))
}

func gcpStage4GRPCShoppingCSSUpdateCssProductInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.UpdateCssProductInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	input := req.GetCssProductInput()
	if reason := gcpStage4ShoppingCSSValidateInput(input, true); reason != "" {
		return grpcInvalidArgument(reason)
	}
	account, inputID, ok := parseGCPShoppingCSSCssProductInputName(input.GetName())
	if !ok {
		return grpcInvalidArgument("css_product_input.name-required")
	}
	if gcpShoppingCSSInputMissing(inputID) {
		return grpcNotFound("input-not-found")
	}
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	title := ""
	if attrs := input.GetAttributes(); attrs != nil {
		title = strings.TrimSpace(attrs.GetTitle())
	}
	if title == "" {
		title = "Stackyard Product " + strings.TrimSpace(input.GetRawProvidedId())
	}
	return grpcProtoSuccess(gcpStage4ShoppingCSSCssProductInput(account, inputID, title))
}

func gcpStage4GRPCShoppingCSSDeleteCssProductInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.DeleteCssProductInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, inputID, ok := parseGCPShoppingCSSCssProductInputName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.SupplementalFeedId != nil && req.GetSupplementalFeedId() < 0 {
		return grpcInvalidArgument("supplemental_feed_id-invalid")
	}
	if gcpShoppingCSSInputMissing(inputID) {
		return grpcNotFound("input-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCShoppingCSSListQuotaGroups(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &shoppingcsspb.ListQuotaGroupsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	account, ok := parseGCPShoppingCSSAccountName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*shoppingcsspb.QuotaGroup{
		gcpStage4ShoppingCSSQuotaGroup(account, "css-products-read", 12, 1000, 60, "cssproductsservice.listcssproducts"),
		gcpStage4ShoppingCSSQuotaGroup(account, "css-products-write", 4, 300, 30, "cssproductinputsservice.insertcssproductinput"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&shoppingcsspb.ListQuotaGroupsResponse{
		QuotaGroups:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4ShoppingCSSValidateInput(input *shoppingcsspb.CssProductInput, requireName bool) string {
	if input == nil {
		return "css_product_input-required"
	}
	if requireName && strings.TrimSpace(input.GetName()) == "" {
		return "css_product_input.name-required"
	}
	if strings.TrimSpace(input.GetRawProvidedId()) == "" {
		return "css_product_input.raw_provided_id-required"
	}
	if !gcpShoppingCSSRawIDRe.MatchString(strings.TrimSpace(input.GetRawProvidedId())) {
		return "css_product_input.raw_provided_id-invalid"
	}
	if !gcpShoppingCSSLanguageRe.MatchString(strings.TrimSpace(input.GetContentLanguage())) {
		return "css_product_input.content_language-invalid"
	}
	if !gcpShoppingCSSFeedLabelRe.MatchString(strings.ToUpper(strings.TrimSpace(input.GetFeedLabel()))) {
		return "css_product_input.feed_label-invalid"
	}
	return ""
}

func gcpStage4ShoppingCSSAccount(account string, labelIDs []int64) *shoppingcsspb.Account {
	if len(labelIDs) == 0 {
		labelIDs = []int64{1001}
	}
	sort.Slice(labelIDs, func(i, j int) bool { return labelIDs[i] < labelIDs[j] })
	parent := "accounts/999999"
	displayName := "Stackyard CSS " + account
	homepage := fmt.Sprintf("https://merchant.stackyard.example/%s", account)
	return &shoppingcsspb.Account{
		Name:              gcpShoppingCSSAccountName(account),
		FullName:          "Stackyard CSS Account " + account,
		DisplayName:       proto.String(displayName),
		HomepageUri:       proto.String(homepage),
		Parent:            proto.String(parent),
		LabelIds:          labelIDs,
		AutomaticLabelIds: []int64{9001},
		AccountType:       shoppingcsspb.Account_CSS_DOMAIN,
	}
}

func gcpStage4ShoppingCSSAccountLabel(account, labelID, displayName, description string) *shoppingcsspb.AccountLabel {
	if displayName == "" {
		displayName = "Stackyard Label " + labelID
	}
	if description == "" {
		description = "Staged account label " + labelID
	}
	return &shoppingcsspb.AccountLabel{
		Name:        gcpShoppingCSSLabelName(account, labelID),
		LabelId:     gcpShoppingCSSNumericID(labelID, 1001),
		AccountId:   gcpShoppingCSSNumericID(account, 123456),
		DisplayName: proto.String(displayName),
		Description: proto.String(description),
		LabelType:   shoppingcsspb.AccountLabel_MANUAL,
	}
}

func gcpStage4ShoppingCSSCssProduct(account, productID, title string) *shoppingcsspb.CssProduct {
	contentLanguage, feedLabel, rawProvidedID := gcpShoppingCSSParseInputID(productID)
	if title == "" {
		title = "Stackyard Product " + rawProvidedID
	}
	return &shoppingcsspb.CssProduct{
		Name:            gcpShoppingCSSCssProductName(account, productID),
		RawProvidedId:   rawProvidedID,
		ContentLanguage: contentLanguage,
		FeedLabel:       feedLabel,
		Attributes: &shoppingcsspb.Attributes{
			Title: proto.String(title),
		},
	}
}

func gcpStage4ShoppingCSSCssProductInput(account, inputID, title string) *shoppingcsspb.CssProductInput {
	contentLanguage, feedLabel, rawProvidedID := gcpShoppingCSSParseInputID(inputID)
	if title == "" {
		title = "Stackyard Product " + rawProvidedID
	}
	return &shoppingcsspb.CssProductInput{
		Name:            gcpShoppingCSSCssProductInputName(account, inputID),
		FinalName:       gcpShoppingCSSCssProductName(account, inputID),
		RawProvidedId:   rawProvidedID,
		ContentLanguage: contentLanguage,
		FeedLabel:       feedLabel,
		Attributes: &shoppingcsspb.Attributes{
			Title: proto.String(title),
		},
	}
}

func gcpStage4ShoppingCSSQuotaGroup(account, group string, usage, limit, minuteLimit int64, method string) *shoppingcsspb.QuotaGroup {
	return &shoppingcsspb.QuotaGroup{
		Name:             gcpShoppingCSSQuotaGroupName(account, group),
		QuotaUsage:       usage,
		QuotaLimit:       limit,
		QuotaMinuteLimit: minuteLimit,
		MethodDetails: []*shoppingcsspb.MethodDetails{
			{
				Method:  method,
				Version: "v1",
				Subapi:  "css",
				Path:    "v1/" + method,
			},
		},
	}
}

func gcpStage4GRPCWebRisk(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpWebRiskComputeThreatListDiffMethod:
		return gcpStage4GRPCWebRiskComputeThreatListDiff(grpcReqBody)
	case gcpWebRiskSearchUrisMethod:
		return gcpStage4GRPCWebRiskSearchUris(grpcReqBody)
	case gcpWebRiskSearchHashesMethod:
		return gcpStage4GRPCWebRiskSearchHashes(grpcReqBody)
	case gcpWebRiskCreateSubmissionMethod:
		return gcpStage4GRPCWebRiskCreateSubmission(grpcReqBody)
	case gcpWebRiskSubmitURIMethod:
		return gcpStage4GRPCWebRiskSubmitURI(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCWebRiskComputeThreatListDiff(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &webriskpb.ComputeThreatListDiffRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !gcpStage4WebRiskThreatTypeValid(req.GetThreatType()) {
		return grpcInvalidArgument("threat_type-required")
	}
	constraints := req.GetConstraints()
	if constraints == nil {
		return grpcInvalidArgument("constraints-required")
	}
	if !gcpStage4WebRiskConstraintValueValid(constraints.GetMaxDiffEntries()) {
		return grpcInvalidArgument("constraints.max_diff_entries-invalid")
	}
	if !gcpStage4WebRiskConstraintValueValid(constraints.GetMaxDatabaseEntries()) {
		return grpcInvalidArgument("constraints.max_database_entries-invalid")
	}
	resp := &webriskpb.ComputeThreatListDiffResponse{
		ResponseType:        webriskpb.ComputeThreatListDiffResponse_DIFF,
		NewVersionToken:     []byte(fmt.Sprintf("stage4-token-%d", req.GetThreatType())),
		Checksum:            &webriskpb.ComputeThreatListDiffResponse_Checksum{Sha256: gcpStage4WebRiskChecksum()},
		RecommendedNextDiff: timestamppb.New(gcpStage4ReferenceTime.Add(30 * time.Minute)),
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCWebRiskSearchUris(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &webriskpb.SearchUrisRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	uri := strings.TrimSpace(req.GetUri())
	if uri == "" {
		return grpcInvalidArgument("uri-required")
	}
	if !isGCPWebRiskURI(uri) {
		return grpcInvalidArgument("uri-invalid")
	}
	if !gcpStage4WebRiskThreatTypesValid(req.GetThreatTypes()) {
		return grpcInvalidArgument("threat_types-required")
	}
	resp := &webriskpb.SearchUrisResponse{
		Threat: &webriskpb.SearchUrisResponse_ThreatUri{
			ThreatTypes: req.GetThreatTypes(),
			ExpireTime:  timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
		},
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCWebRiskSearchHashes(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &webriskpb.SearchHashesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if len(req.GetHashPrefix()) < 4 || len(req.GetHashPrefix()) > 32 {
		return grpcInvalidArgument("hash_prefix-length-invalid")
	}
	if !gcpStage4WebRiskThreatTypesValid(req.GetThreatTypes()) {
		return grpcInvalidArgument("threat_types-required")
	}
	hash := make([]byte, 32)
	copy(hash, req.GetHashPrefix())
	for i := len(req.GetHashPrefix()); i < len(hash); i++ {
		hash[i] = byte(i + 1)
	}
	resp := &webriskpb.SearchHashesResponse{
		Threats: []*webriskpb.SearchHashesResponse_ThreatHash{
			{
				ThreatTypes: req.GetThreatTypes(),
				Hash:        hash,
				ExpireTime:  timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
			},
		},
		NegativeExpireTime: timestamppb.New(gcpStage4ReferenceTime.Add(30 * time.Minute)),
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCWebRiskCreateSubmission(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &webriskpb.CreateSubmissionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPWebRiskProjectParent(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	submission := req.GetSubmission()
	if submission == nil {
		return grpcInvalidArgument("submission-required")
	}
	uri := strings.TrimSpace(submission.GetUri())
	if uri == "" {
		return grpcInvalidArgument("submission.uri-required")
	}
	if !isGCPWebRiskURI(uri) {
		return grpcInvalidArgument("submission.uri-invalid")
	}
	resp := &webriskpb.Submission{
		Uri:         uri,
		ThreatTypes: gcpStage4WebRiskThreatTypesForURI(uri),
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCWebRiskSubmitURI(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &webriskpb.SubmitUriRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPWebRiskProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	submission := req.GetSubmission()
	if submission == nil {
		return grpcInvalidArgument("submission-required")
	}
	uri := strings.TrimSpace(submission.GetUri())
	if uri == "" {
		return grpcInvalidArgument("submission.uri-required")
	}
	if !isGCPWebRiskURI(uri) {
		return grpcInvalidArgument("submission.uri-invalid")
	}
	metadataAny, _ := anypb.New(&webriskpb.SubmitUriMetadata{
		State:      webriskpb.SubmitUriMetadata_RUNNING,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(45 * time.Second)),
	})
	return grpcProtoSuccess(&longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/operations/submitUri.grpc-op-1", project),
		Done:     false,
		Metadata: metadataAny,
	})
}

func gcpStage4WebRiskThreatTypeValid(threatType webriskpb.ThreatType) bool {
	return threatType != webriskpb.ThreatType_THREAT_TYPE_UNSPECIFIED
}

func gcpStage4WebRiskThreatTypesValid(threatTypes []webriskpb.ThreatType) bool {
	if len(threatTypes) == 0 {
		return false
	}
	for _, threatType := range threatTypes {
		if !gcpStage4WebRiskThreatTypeValid(threatType) {
			return false
		}
	}
	return true
}

func gcpStage4WebRiskConstraintValueValid(value int32) bool {
	if value == 0 {
		return true
	}
	if value < (1<<10) || value > (1<<20) {
		return false
	}
	return isGCPWebRiskPowerOfTwo(int(value))
}

func gcpStage4WebRiskChecksum() []byte {
	checksum := make([]byte, 32)
	for i := range checksum {
		checksum[i] = byte(i + 1)
	}
	return checksum
}

func gcpStage4WebRiskThreatTypesForURI(uri string) []webriskpb.ThreatType {
	lowered := strings.ToLower(strings.TrimSpace(uri))
	switch {
	case strings.Contains(lowered, "malware"):
		return []webriskpb.ThreatType{webriskpb.ThreatType_MALWARE}
	case strings.Contains(lowered, "phish"):
		return []webriskpb.ThreatType{webriskpb.ThreatType_SOCIAL_ENGINEERING}
	default:
		return []webriskpb.ThreatType{webriskpb.ThreatType_SOCIAL_ENGINEERING}
	}
}

func gcpStage4GRPCServiceControlCheck(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicecontrolpb.CheckRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPStage4ServiceControlServiceName(req.GetServiceName()); !ok {
		return grpcInvalidArgument("service_name-required")
	}
	operation := req.GetOperation()
	if operation == nil {
		return grpcInvalidArgument("operation-required")
	}
	if strings.TrimSpace(operation.GetOperationId()) == "" {
		return grpcInvalidArgument("operation.operation_id-required")
	}
	if operation.GetStartTime() == nil {
		return grpcInvalidArgument("operation.start_time-required")
	}
	if err := operation.GetStartTime().CheckValid(); err != nil {
		return grpcInvalidArgument("operation.start_time-invalid")
	}
	denied := gcpStage4ServiceControlDeniedConsumer(operation.GetConsumerId())
	resp := &servicecontrolpb.CheckResponse{
		OperationId:      operation.GetOperationId(),
		ServiceConfigId:  "2026-01-01r0",
		ServiceRolloutId: "2026-01-01r0",
		CheckInfo: &servicecontrolpb.CheckResponse_CheckInfo{
			UnusedArguments: []string{},
			ConsumerInfo: &servicecontrolpb.CheckResponse_ConsumerInfo{
				ProjectNumber: 1234567890,
			},
		},
	}
	if denied {
		resp.CheckErrors = []*servicecontrolpb.CheckError{
			{
				Code:    servicecontrolpb.CheckError_PERMISSION_DENIED,
				Subject: "operation.consumer_id",
				Detail:  "permission denied by staged emulation",
			},
		}
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCServiceControlReport(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicecontrolpb.ReportRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPStage4ServiceControlServiceName(req.GetServiceName()); !ok {
		return grpcInvalidArgument("service_name-required")
	}
	operations := req.GetOperations()
	if len(operations) == 0 {
		return grpcInvalidArgument("operations-required")
	}
	for _, operation := range operations {
		if operation == nil {
			return grpcInvalidArgument("operations.operation-required")
		}
		if strings.TrimSpace(operation.GetOperationId()) == "" {
			return grpcInvalidArgument("operations.operation_id-required")
		}
		if strings.TrimSpace(operation.GetConsumerId()) == "" {
			return grpcInvalidArgument("operations.consumer_id-required")
		}
		if operation.GetStartTime() == nil {
			return grpcInvalidArgument("operations.start_time-required")
		}
		if err := operation.GetStartTime().CheckValid(); err != nil {
			return grpcInvalidArgument("operations.start_time-invalid")
		}
		if operation.GetEndTime() != nil {
			if err := operation.GetEndTime().CheckValid(); err != nil {
				return grpcInvalidArgument("operations.end_time-invalid")
			}
		}
	}
	return grpcProtoSuccess(&servicecontrolpb.ReportResponse{
		ServiceConfigId:  "2026-01-01r0",
		ServiceRolloutId: "2026-01-01r0",
		ReportErrors:     []*servicecontrolpb.ReportResponse_ReportError{},
	})
}

func gcpStage4GRPCServiceControlAllocateQuota(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicecontrolpb.AllocateQuotaRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPStage4ServiceControlServiceName(req.GetServiceName()); !ok {
		return grpcInvalidArgument("service_name-required")
	}
	allocateOperation := req.GetAllocateOperation()
	if allocateOperation == nil {
		return grpcInvalidArgument("allocate_operation-required")
	}
	if strings.TrimSpace(allocateOperation.GetOperationId()) == "" {
		return grpcInvalidArgument("allocate_operation.operation_id-required")
	}
	if strings.TrimSpace(allocateOperation.GetConsumerId()) == "" {
		return grpcInvalidArgument("allocate_operation.consumer_id-required")
	}
	hasMethodName := strings.TrimSpace(allocateOperation.GetMethodName()) != ""
	hasQuotaMetrics := len(allocateOperation.GetQuotaMetrics()) > 0
	if hasMethodName && hasQuotaMetrics {
		return grpcInvalidArgument("allocate_operation.method_name-and-quota_metrics-mutually-exclusive")
	}
	if !hasMethodName && !hasQuotaMetrics {
		return grpcInvalidArgument("allocate_operation.method_name-or-quota_metrics-required")
	}
	for _, metricSet := range allocateOperation.GetQuotaMetrics() {
		if strings.TrimSpace(metricSet.GetMetricName()) == "" {
			return grpcInvalidArgument("allocate_operation.quota_metrics.metric_name-required")
		}
		if len(metricSet.GetMetricValues()) == 0 {
			return grpcInvalidArgument("allocate_operation.quota_metrics.metric_values-required")
		}
		for _, value := range metricSet.GetMetricValues() {
			if value == nil || value.GetValue() == nil {
				return grpcInvalidArgument("allocate_operation.quota_metrics.metric_values.value-required")
			}
		}
	}

	resp := &servicecontrolpb.AllocateQuotaResponse{
		OperationId:     allocateOperation.GetOperationId(),
		ServiceConfigId: "2026-01-01r0",
		AllocateErrors:  []*servicecontrolpb.QuotaError{},
		QuotaMetrics: []*servicecontrolpb.MetricValueSet{
			{
				MetricName: "serviceruntime.googleapis.com/api/consumer/quota_used_count",
				MetricValues: []*servicecontrolpb.MetricValue{
					{
						Value: &servicecontrolpb.MetricValue_Int64Value{
							Int64Value: 1,
						},
						StartTime: timestamppb.New(gcpStage4ReferenceTime),
						EndTime:   timestamppb.New(gcpStage4ReferenceTime.Add(time.Minute)),
					},
				},
			},
		},
	}
	if gcpStage4ServiceControlDeniedConsumer(allocateOperation.GetConsumerId()) {
		resp.AllocateErrors = []*servicecontrolpb.QuotaError{
			{
				Code:        servicecontrolpb.QuotaError_RESOURCE_EXHAUSTED,
				Subject:     "allocate_operation.consumer_id",
				Description: "quota exhausted by staged emulation",
			},
		}
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4ServiceControlDeniedConsumer(consumerID string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(consumerID)), "deny")
}

func gcpStage4GRPCTalentCompanyService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTalentCreateCompanyMethod:
		return gcpStage4GRPCTalentCreateCompany(grpcReqBody)
	case gcpTalentGetCompanyMethod:
		return gcpStage4GRPCTalentGetCompany(grpcReqBody)
	case gcpTalentUpdateCompanyMethod:
		return gcpStage4GRPCTalentUpdateCompany(grpcReqBody)
	case gcpTalentDeleteCompanyMethod:
		return gcpStage4GRPCTalentDeleteCompany(grpcReqBody)
	case gcpTalentListCompaniesMethod:
		return gcpStage4GRPCTalentListCompanies(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTalentTenantService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTalentCreateTenantMethod:
		return gcpStage4GRPCTalentCreateTenant(grpcReqBody)
	case gcpTalentGetTenantMethod:
		return gcpStage4GRPCTalentGetTenant(grpcReqBody)
	case gcpTalentUpdateTenantMethod:
		return gcpStage4GRPCTalentUpdateTenant(grpcReqBody)
	case gcpTalentDeleteTenantMethod:
		return gcpStage4GRPCTalentDeleteTenant(grpcReqBody)
	case gcpTalentListTenantsMethod:
		return gcpStage4GRPCTalentListTenants(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTalentJobService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTalentCreateJobMethod:
		return gcpStage4GRPCTalentCreateJob(grpcReqBody)
	case gcpTalentBatchCreateJobsMethod:
		return gcpStage4GRPCTalentBatchCreateJobs(grpcReqBody)
	case gcpTalentGetJobMethod:
		return gcpStage4GRPCTalentGetJob(grpcReqBody)
	case gcpTalentUpdateJobMethod:
		return gcpStage4GRPCTalentUpdateJob(grpcReqBody)
	case gcpTalentBatchUpdateJobsMethod:
		return gcpStage4GRPCTalentBatchUpdateJobs(grpcReqBody)
	case gcpTalentDeleteJobMethod:
		return gcpStage4GRPCTalentDeleteJob(grpcReqBody)
	case gcpTalentBatchDeleteJobsMethod:
		return gcpStage4GRPCTalentBatchDeleteJobs(grpcReqBody)
	case gcpTalentListJobsMethod:
		return gcpStage4GRPCTalentListJobs(grpcReqBody)
	case gcpTalentSearchJobsMethod:
		return gcpStage4GRPCTalentSearchJobs(grpcReqBody)
	case gcpTalentSearchJobsForAlertMethod:
		return gcpStage4GRPCTalentSearchJobsForAlert(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTalentCompletionService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTalentCompleteQueryMethod:
		return gcpStage4GRPCTalentCompleteQuery(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTalentEventService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTalentCreateClientEventMethod:
		return gcpStage4GRPCTalentCreateClientEvent(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTalentCreateTenant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.CreateTenantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPStage4TalentProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	tenant := req.GetTenant()
	if tenant == nil {
		return grpcInvalidArgument("tenant-required")
	}
	externalID := strings.TrimSpace(tenant.GetExternalId())
	if externalID == "" {
		return grpcInvalidArgument("tenant.external_id-required")
	}
	tenantID := "tenant-created-1"
	if name := strings.TrimSpace(tenant.GetName()); name != "" {
		nameProject, parsedTenantID, parsed := parseGCPStage4TalentTenantName(name)
		if !parsed || nameProject != project {
			return grpcInvalidArgument("tenant.name-invalid")
		}
		tenantID = parsedTenantID
	}
	return grpcProtoSuccess(gcpStage4TalentTenant(project, tenantID, externalID))
}

func gcpStage4GRPCTalentGetTenant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.GetTenantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(tenantID), "missing") {
		return grpcNotFound("tenant-not-found")
	}
	return grpcProtoSuccess(gcpStage4TalentTenant(project, tenantID, "tenant-ext-"+tenantID))
}

func gcpStage4GRPCTalentUpdateTenant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.UpdateTenantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	tenant := req.GetTenant()
	if tenant == nil {
		return grpcInvalidArgument("tenant-required")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(tenant.GetName())
	if !ok {
		return grpcInvalidArgument("tenant.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4TalentMaskValid(req.GetUpdateMask().GetPaths(), map[string]struct{}{
		"external_id": {},
		"externalId":  {},
	}) {
		return grpcInvalidArgument("update_mask-unsupported")
	}
	externalID := strings.TrimSpace(tenant.GetExternalId())
	if externalID == "" {
		return grpcInvalidArgument("tenant.external_id-required")
	}
	return grpcProtoSuccess(gcpStage4TalentTenant(project, tenantID, externalID))
}

func gcpStage4GRPCTalentDeleteTenant(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.DeleteTenantRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPStage4TalentTenantName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCTalentListTenants(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.ListTenantsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPStage4TalentProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	pageSize, start, reason, ok := gcpStage4TalentParsePage(req.GetPageSize(), req.GetPageToken(), 100, 100)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*talentpb.Tenant{
		gcpStage4TalentTenant(project, "tenant-1", "tenant-ext-1"),
		gcpStage4TalentTenant(project, "tenant-2", "tenant-ext-2"),
	}
	end, next, ok := gcpStage4TalentPageRange(len(items), pageSize, start)
	if !ok {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	return grpcProtoSuccess(&talentpb.ListTenantsResponse{
		Tenants:       items[start:end],
		NextPageToken: next,
		Metadata:      &talentpb.ResponseMetadata{RequestId: "talent-listtenants-req-1"},
	})
}

func gcpStage4GRPCTalentCreateCompany(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.CreateCompanyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	company := req.GetCompany()
	if company == nil {
		return grpcInvalidArgument("company-required")
	}
	displayName := strings.TrimSpace(company.GetDisplayName())
	externalID := strings.TrimSpace(company.GetExternalId())
	if displayName == "" || externalID == "" {
		return grpcInvalidArgument("company.display_name-and-external_id-required")
	}
	companyID := "company-created-1"
	if name := strings.TrimSpace(company.GetName()); name != "" {
		nameProject, nameTenant, parsedCompanyID, parsed := parseGCPStage4TalentCompanyName(name)
		if !parsed || nameProject != project || nameTenant != tenantID {
			return grpcInvalidArgument("company.name-invalid")
		}
		companyID = parsedCompanyID
	}
	created := gcpStage4TalentCompany(project, tenantID, companyID, displayName, externalID)
	if website := strings.TrimSpace(company.GetWebsiteUri()); website != "" {
		created.WebsiteUri = website
	}
	return grpcProtoSuccess(created)
}

func gcpStage4GRPCTalentGetCompany(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.GetCompanyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, companyID, ok := parseGCPStage4TalentCompanyName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(companyID), "missing") {
		return grpcNotFound("company-not-found")
	}
	return grpcProtoSuccess(gcpStage4TalentCompany(project, tenantID, companyID, "Company "+companyID, "company-ext-"+companyID))
}

func gcpStage4GRPCTalentUpdateCompany(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.UpdateCompanyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	company := req.GetCompany()
	if company == nil {
		return grpcInvalidArgument("company-required")
	}
	project, tenantID, companyID, ok := parseGCPStage4TalentCompanyName(company.GetName())
	if !ok {
		return grpcInvalidArgument("company.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4TalentMaskValid(req.GetUpdateMask().GetPaths(), map[string]struct{}{
		"display_name": {},
		"displayName":  {},
		"external_id":  {},
		"externalId":   {},
		"website_uri":  {},
		"websiteUri":   {},
	}) {
		return grpcInvalidArgument("update_mask-unsupported")
	}
	displayName := strings.TrimSpace(company.GetDisplayName())
	externalID := strings.TrimSpace(company.GetExternalId())
	if displayName == "" || externalID == "" {
		return grpcInvalidArgument("company.display_name-and-external_id-required")
	}
	updated := gcpStage4TalentCompany(project, tenantID, companyID, displayName, externalID)
	if website := strings.TrimSpace(company.GetWebsiteUri()); website != "" {
		updated.WebsiteUri = website
	}
	return grpcProtoSuccess(updated)
}

func gcpStage4GRPCTalentDeleteCompany(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.DeleteCompanyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4TalentCompanyName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCTalentListCompanies(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.ListCompaniesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	pageSize, start, reason, ok := gcpStage4TalentParsePage(req.GetPageSize(), req.GetPageToken(), 100, 100)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*talentpb.Company{
		gcpStage4TalentCompany(project, tenantID, "company-1", "Stackyard Inc", "company-ext-1"),
		gcpStage4TalentCompany(project, tenantID, "company-2", "Example Corp", "company-ext-2"),
	}
	if req.GetRequireOpenJobs() {
		items = items[:1]
	}
	end, next, ok := gcpStage4TalentPageRange(len(items), pageSize, start)
	if !ok {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	return grpcProtoSuccess(&talentpb.ListCompaniesResponse{
		Companies:     items[start:end],
		NextPageToken: next,
		Metadata:      &talentpb.ResponseMetadata{RequestId: "talent-listcompanies-req-1"},
	})
}

func gcpStage4GRPCTalentCreateJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.CreateJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	job, _, reason := gcpStage4TalentParseJobMutation(project, tenantID, req.GetJob(), false)
	if reason != "" {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(job)
}

func gcpStage4GRPCTalentGetJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.GetJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, jobID, ok := parseGCPStage4TalentJobName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(jobID), "missing") {
		return grpcNotFound("job-not-found")
	}
	return grpcProtoSuccess(gcpStage4TalentJob(project, tenantID, jobID, "company-1", "req-"+jobID, "Job "+jobID, "Deterministic staged job"))
}

func gcpStage4GRPCTalentUpdateJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.UpdateJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4TalentMaskValid(req.GetUpdateMask().GetPaths(), map[string]struct{}{
		"company":        {},
		"requisition_id": {},
		"requisitionId":  {},
		"title":          {},
		"description":    {},
		"addresses":      {},
		"language_code":  {},
		"languageCode":   {},
	}) {
		return grpcInvalidArgument("update_mask-unsupported")
	}
	project, tenantID, ok := "", "", false
	if req.GetJob() != nil {
		project, tenantID, _, ok = parseGCPStage4TalentJobName(req.GetJob().GetName())
	}
	if !ok {
		return grpcInvalidArgument("job.name-required")
	}
	job, _, reason := gcpStage4TalentParseJobMutation(project, tenantID, req.GetJob(), true)
	if reason != "" {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(job)
}

func gcpStage4GRPCTalentDeleteJob(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.DeleteJobRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4TalentJobName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCTalentListJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.ListJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	filter := strings.TrimSpace(req.GetFilter())
	if filter == "" {
		return grpcInvalidArgument("filter-required")
	}
	parent := fmt.Sprintf("projects/%s/tenants/%s", project, tenantID)
	filterSpec, valid := parseGCPTalentListJobsFilter(filter, parent)
	if !valid {
		return grpcInvalidArgument("filter-invalid")
	}
	maxPageSize := 100
	if req.GetJobView() == talentpb.JobView_JOB_VIEW_ID_ONLY {
		maxPageSize = 1000
	}
	pageSize, start, reason, ok := gcpStage4TalentParsePage(req.GetPageSize(), req.GetPageToken(), maxPageSize, 100)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*talentpb.Job{
		gcpStage4TalentJob(project, tenantID, "job-1", "company-1", "req-1", "Software Engineer", "Build distributed systems"),
		gcpStage4TalentJob(project, tenantID, "job-2", "company-2", "req-2", "Site Reliability Engineer", "Operate reliable infrastructure"),
	}
	items = gcpStage4TalentFilterJobs(items, filterSpec)
	end, next, ok := gcpStage4TalentPageRange(len(items), pageSize, start)
	if !ok {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	return grpcProtoSuccess(&talentpb.ListJobsResponse{
		Jobs:          items[start:end],
		NextPageToken: next,
		Metadata:      &talentpb.ResponseMetadata{RequestId: "talent-listjobs-req-1"},
	})
}

func gcpStage4GRPCTalentBatchCreateJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.BatchCreateJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if len(req.GetJobs()) == 0 {
		return grpcInvalidArgument("jobs-required")
	}
	if len(req.GetJobs()) > 200 {
		return grpcInvalidArgument("jobs-too-many")
	}
	results := make([]*talentpb.JobResult, 0, len(req.GetJobs()))
	resourceNames := make([]string, 0, len(req.GetJobs()))
	for _, item := range req.GetJobs() {
		job, resourceName, reason := gcpStage4TalentParseJobMutation(project, tenantID, item, false)
		if reason != "" {
			return grpcInvalidArgument(reason)
		}
		results = append(results, &talentpb.JobResult{
			Job:    job,
			Status: &statuspb.Status{Code: 0},
		})
		resourceNames = append(resourceNames, resourceName)
	}
	response := &talentpb.BatchCreateJobsResponse{JobResults: results}
	return grpcProtoSuccess(gcpStage4TalentBatchOperation(project, tenantID, "batchCreateJobs-1", resourceNames, response))
}

func gcpStage4GRPCTalentBatchUpdateJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.BatchUpdateJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if len(req.GetJobs()) == 0 {
		return grpcInvalidArgument("jobs-required")
	}
	if len(req.GetJobs()) > 200 {
		return grpcInvalidArgument("jobs-too-many")
	}
	if mask := req.GetUpdateMask(); mask != nil && len(mask.GetPaths()) > 0 {
		if !gcpStage4TalentMaskValid(mask.GetPaths(), map[string]struct{}{
			"company":        {},
			"requisition_id": {},
			"requisitionId":  {},
			"title":          {},
			"description":    {},
			"addresses":      {},
			"language_code":  {},
			"languageCode":   {},
		}) {
			return grpcInvalidArgument("update_mask-unsupported")
		}
	}
	results := make([]*talentpb.JobResult, 0, len(req.GetJobs()))
	resourceNames := make([]string, 0, len(req.GetJobs()))
	for _, item := range req.GetJobs() {
		job, resourceName, reason := gcpStage4TalentParseJobMutation(project, tenantID, item, true)
		if reason != "" {
			return grpcInvalidArgument(reason)
		}
		results = append(results, &talentpb.JobResult{
			Job:    job,
			Status: &statuspb.Status{Code: 0},
		})
		resourceNames = append(resourceNames, resourceName)
	}
	response := &talentpb.BatchUpdateJobsResponse{JobResults: results}
	return grpcProtoSuccess(gcpStage4TalentBatchOperation(project, tenantID, "batchUpdateJobs-1", resourceNames, response))
}

func gcpStage4GRPCTalentBatchDeleteJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.BatchDeleteJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	names := req.GetNames()
	if len(names) == 0 {
		return grpcInvalidArgument("names-required")
	}
	if len(names) > 200 {
		return grpcInvalidArgument("names-too-many")
	}
	results := make([]*talentpb.JobResult, 0, len(names))
	resourceNames := make([]string, 0, len(names))
	for _, name := range names {
		nameProject, nameTenant, jobID, parsed := parseGCPStage4TalentJobName(name)
		if !parsed || nameProject != project || nameTenant != tenantID {
			return grpcInvalidArgument("name-invalid")
		}
		job := gcpStage4TalentJob(project, tenantID, jobID, "company-1", "req-"+jobID, "Job "+jobID, "Deterministic staged job")
		results = append(results, &talentpb.JobResult{
			Job:    job,
			Status: &statuspb.Status{Code: 0},
		})
		resourceNames = append(resourceNames, name)
	}
	response := &talentpb.BatchDeleteJobsResponse{JobResults: results}
	return grpcProtoSuccess(gcpStage4TalentBatchOperation(project, tenantID, "batchDeleteJobs-1", resourceNames, response))
}

func gcpStage4GRPCTalentSearchJobs(grpcReqBody []byte) ([]byte, string, string, bool) {
	return gcpStage4GRPCTalentSearchJobsCommon(grpcReqBody, false)
}

func gcpStage4GRPCTalentSearchJobsForAlert(grpcReqBody []byte) ([]byte, string, string, bool) {
	return gcpStage4GRPCTalentSearchJobsCommon(grpcReqBody, true)
}

func gcpStage4GRPCTalentSearchJobsCommon(grpcReqBody []byte, alert bool) ([]byte, string, string, bool) {
	req := &talentpb.SearchJobsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if !gcpStage4TalentValidRequestMetadata(req.GetRequestMetadata()) {
		return grpcInvalidArgument("request_metadata-required")
	}
	if req.GetMaxPageSize() < 0 || req.GetMaxPageSize() > 100 {
		return grpcInvalidArgument("max_page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	if req.GetOffset() < 0 || req.GetOffset() > 5000 {
		return grpcInvalidArgument("offset-invalid")
	}
	if strings.TrimSpace(req.GetPageToken()) == "" && req.GetOffset() > 0 {
		start = int(req.GetOffset())
	}
	pageSize := int(req.GetMaxPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}

	items := []*talentpb.Job{
		gcpStage4TalentJob(project, tenantID, "job-1", "company-1", "req-1", "Software Engineer", "Build distributed systems"),
		gcpStage4TalentJob(project, tenantID, "job-2", "company-2", "req-2", "Site Reliability Engineer", "Operate reliable infrastructure"),
	}
	query := strings.TrimSpace(req.GetJobQuery().GetQuery())
	if query != "" {
		queryLower := strings.ToLower(query)
		filtered := make([]*talentpb.Job, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.GetTitle()), queryLower) || strings.Contains(strings.ToLower(item.GetDescription()), queryLower) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if strings.Contains(strings.ToLower(query), "nomatch") {
		items = []*talentpb.Job{}
	}
	if alert && len(items) > 1 {
		items = items[:1]
	}
	matches := make([]*talentpb.SearchJobsResponse_MatchingJob, 0, len(items))
	for _, item := range items {
		matches = append(matches, &talentpb.SearchJobsResponse_MatchingJob{
			Job:               item,
			JobSummary:        "Summary for " + item.GetTitle(),
			JobTitleSnippet:   "<b>" + item.GetTitle() + "</b>",
			SearchTextSnippet: item.GetDescription(),
		})
	}
	totalSize := int32(len(matches))
	end, next, ok := gcpStage4TalentPageRange(len(matches), pageSize, start)
	if !ok {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	requestID := "talent-search-req-1"
	if alert {
		requestID = "talent-search-alert-req-1"
	}
	return grpcProtoSuccess(&talentpb.SearchJobsResponse{
		MatchingJobs:  matches[start:end],
		NextPageToken: next,
		TotalSize:     totalSize,
		Metadata:      &talentpb.ResponseMetadata{RequestId: requestID},
	})
}

func gcpStage4GRPCTalentCompleteQuery(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.CompleteQueryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetTenant())
	if !ok {
		return grpcInvalidArgument("tenant-required")
	}
	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return grpcInvalidArgument("query-required")
	}
	if req.GetPageSize() <= 0 || req.GetPageSize() > 10 {
		return grpcInvalidArgument("page_size-range")
	}
	if company := strings.TrimSpace(req.GetCompany()); company != "" {
		companyProject, companyTenant, _, parsed := parseGCPStage4TalentCompanyName(company)
		if !parsed || companyProject != project || companyTenant != tenantID {
			return grpcInvalidArgument("company-invalid")
		}
	}
	results := []*talentpb.CompleteQueryResponse_CompletionResult{
		{
			Suggestion: "Software Engineer",
			Type:       talentpb.CompleteQueryRequest_JOB_TITLE,
		},
		{
			Suggestion: "Stackyard Inc",
			Type:       talentpb.CompleteQueryRequest_COMPANY_NAME,
			ImageUri:   "https://example.com/logo.png",
		},
		{
			Suggestion: "Stackyard Engineer",
			Type:       talentpb.CompleteQueryRequest_COMBINED,
		},
	}
	queryLower := strings.ToLower(query)
	filtered := make([]*talentpb.CompleteQueryResponse_CompletionResult, 0, len(results))
	for _, item := range results {
		if strings.Contains(strings.ToLower(item.GetSuggestion()), queryLower) {
			filtered = append(filtered, item)
		}
	}
	results = filtered
	if req.GetType() != talentpb.CompleteQueryRequest_COMPLETION_TYPE_UNSPECIFIED && req.GetType() != talentpb.CompleteQueryRequest_COMBINED {
		filtered = filtered[:0]
		for _, item := range results {
			if item.GetType() == req.GetType() {
				filtered = append(filtered, item)
			}
		}
		results = filtered
	}
	if req.GetScope() == talentpb.CompleteQueryRequest_TENANT && len(results) > 1 {
		results = results[:1]
	}
	if int(req.GetPageSize()) < len(results) {
		results = results[:int(req.GetPageSize())]
	}
	return grpcProtoSuccess(&talentpb.CompleteQueryResponse{
		CompletionResults: results,
		Metadata:          &talentpb.ResponseMetadata{RequestId: "talent-complete-req-1"},
	})
}

func gcpStage4GRPCTalentCreateClientEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &talentpb.CreateClientEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, tenantID, ok := parseGCPStage4TalentTenantName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	clientEvent := req.GetClientEvent()
	if clientEvent == nil {
		return grpcInvalidArgument("client_event-required")
	}
	if strings.TrimSpace(clientEvent.GetEventId()) == "" {
		return grpcInvalidArgument("client_event.event_id-required")
	}
	if clientEvent.GetCreateTime() == nil {
		return grpcInvalidArgument("client_event.create_time-required")
	}
	jobEvent := clientEvent.GetJobEvent()
	if jobEvent == nil {
		return grpcInvalidArgument("client_event.job_event-required")
	}
	if jobEvent.GetType() == talentpb.JobEvent_JOB_EVENT_TYPE_UNSPECIFIED {
		return grpcInvalidArgument("client_event.job_event.type-required")
	}
	if len(jobEvent.GetJobs()) == 0 {
		return grpcInvalidArgument("client_event.job_event.jobs-required")
	}
	jobs := make([]string, 0, len(jobEvent.GetJobs()))
	for _, jobName := range jobEvent.GetJobs() {
		nameProject, nameTenant, _, parsed := parseGCPStage4TalentJobName(jobName)
		if !parsed || nameProject != project || nameTenant != tenantID {
			return grpcInvalidArgument("client_event.job_event.jobs-invalid")
		}
		jobs = append(jobs, jobName)
	}
	requestID := strings.TrimSpace(clientEvent.GetRequestId())
	if requestID == "" {
		requestID = "talent-event-req-1"
	}
	return grpcProtoSuccess(&talentpb.ClientEvent{
		RequestId:  requestID,
		EventId:    clientEvent.GetEventId(),
		CreateTime: clientEvent.GetCreateTime(),
		Event: &talentpb.ClientEvent_JobEvent{
			JobEvent: &talentpb.JobEvent{
				Type: jobEvent.GetType(),
				Jobs: jobs,
			},
		},
		EventNotes: clientEvent.GetEventNotes(),
	})
}

func gcpStage4TalentTenant(project, tenantID, externalID string) *talentpb.Tenant {
	if strings.TrimSpace(externalID) == "" {
		externalID = "tenant-ext-" + tenantID
	}
	return &talentpb.Tenant{
		Name:       fmt.Sprintf("projects/%s/tenants/%s", project, tenantID),
		ExternalId: externalID,
	}
}

func gcpStage4TalentCompany(project, tenantID, companyID, displayName, externalID string) *talentpb.Company {
	if strings.TrimSpace(displayName) == "" {
		displayName = "Company " + companyID
	}
	if strings.TrimSpace(externalID) == "" {
		externalID = "company-ext-" + companyID
	}
	return &talentpb.Company{
		Name:        fmt.Sprintf("projects/%s/tenants/%s/companies/%s", project, tenantID, companyID),
		DisplayName: displayName,
		ExternalId:  externalID,
		Size:        talentpb.CompanySize_SMALL,
		WebsiteUri:  "https://example.com",
	}
}

func gcpStage4TalentJob(project, tenantID, jobID, companyID, requisitionID, title, description string) *talentpb.Job {
	created := gcpStage4ReferenceTime.Add(5 * time.Minute)
	updated := created.Add(15 * time.Minute)
	expire := created.Add(30 * 24 * time.Hour)
	return &talentpb.Job{
		Name:               fmt.Sprintf("projects/%s/tenants/%s/jobs/%s", project, tenantID, jobID),
		Company:            fmt.Sprintf("projects/%s/tenants/%s/companies/%s", project, tenantID, companyID),
		RequisitionId:      requisitionID,
		Title:              title,
		Description:        description,
		Addresses:          []string{"1600 Amphitheatre Parkway, Mountain View, CA, USA"},
		LanguageCode:       "en-US",
		CompanyDisplayName: "Stackyard Inc",
		PostingCreateTime:  timestamppb.New(created),
		PostingUpdateTime:  timestamppb.New(updated),
		PostingExpireTime:  timestamppb.New(expire),
	}
}

func gcpStage4TalentBatchOperation(project, tenantID, operationID string, resourceNames []string, response proto.Message) *longrunningpb.Operation {
	if response == nil {
		response = &emptypb.Empty{}
	}
	responseAny := &anypb.Any{}
	if packed, err := anypb.New(response); err == nil {
		responseAny = packed
	}
	successCount := int32(len(resourceNames))
	metadata := &talentpb.BatchOperationMetadata{
		State:            talentpb.BatchOperationMetadata_SUCCEEDED,
		StateDescription: "completed",
		SuccessCount:     successCount,
		FailureCount:     0,
		TotalCount:       successCount,
		CreateTime:       timestamppb.New(gcpStage4ReferenceTime.Add(10 * time.Minute)),
		UpdateTime:       timestamppb.New(gcpStage4ReferenceTime.Add(11 * time.Minute)),
		EndTime:          timestamppb.New(gcpStage4ReferenceTime.Add(11 * time.Minute)),
	}
	metadataAny := &anypb.Any{}
	if packed, err := anypb.New(metadata); err == nil {
		metadataAny = packed
	}
	return &longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/tenants/%s/operations/%s", project, tenantID, operationID),
		Done:     true,
		Metadata: metadataAny,
		Result: &longrunningpb.Operation_Response{
			Response: responseAny,
		},
	}
}

func gcpStage4TalentParseJobMutation(project, tenantID string, job *talentpb.Job, requireName bool) (*talentpb.Job, string, string) {
	if job == nil {
		return nil, "", "job-required"
	}
	companyProject, companyTenant, companyID, ok := parseGCPStage4TalentCompanyName(job.GetCompany())
	if !ok || companyProject != project || companyTenant != tenantID {
		return nil, "", "job.company-invalid"
	}
	requisitionID := strings.TrimSpace(job.GetRequisitionId())
	title := strings.TrimSpace(job.GetTitle())
	description := strings.TrimSpace(job.GetDescription())
	if requisitionID == "" || title == "" || description == "" {
		return nil, "", "job.requisition_id-title-description-required"
	}
	jobID := "job-created-1"
	if name := strings.TrimSpace(job.GetName()); name != "" {
		nameProject, nameTenant, parsedJobID, parsed := parseGCPStage4TalentJobName(name)
		if !parsed || nameProject != project || nameTenant != tenantID {
			return nil, "", "job.name-invalid"
		}
		jobID = parsedJobID
	} else if requireName {
		return nil, "", "job.name-required"
	}
	parsed := gcpStage4TalentJob(project, tenantID, jobID, companyID, requisitionID, title, description)
	if len(job.GetAddresses()) > 0 {
		parsed.Addresses = append([]string{}, job.GetAddresses()...)
	}
	if languageCode := strings.TrimSpace(job.GetLanguageCode()); languageCode != "" {
		parsed.LanguageCode = languageCode
	}
	return parsed, parsed.GetName(), ""
}

func gcpStage4TalentValidRequestMetadata(metadata *talentpb.RequestMetadata) bool {
	if metadata == nil {
		return false
	}
	if metadata.GetAllowMissingIds() {
		return true
	}
	return strings.TrimSpace(metadata.GetDomain()) != "" &&
		strings.TrimSpace(metadata.GetSessionId()) != "" &&
		strings.TrimSpace(metadata.GetUserId()) != ""
}

func gcpStage4TalentFilterJobs(items []*talentpb.Job, filterSpec gcpTalentListJobsFilterSpec) []*talentpb.Job {
	filtered := make([]*talentpb.Job, 0, len(items))
	for _, item := range items {
		if filterSpec.CompanyName != "" && item.GetCompany() != filterSpec.CompanyName {
			continue
		}
		if filterSpec.RequisitionID != "" && item.GetRequisitionId() != filterSpec.RequisitionID {
			continue
		}
		if filterSpec.Status != "" && filterSpec.Status != "ALL" {
			// Talent v4 Job has no explicit status field in the proto; staged jobs are treated as OPEN.
			status := "OPEN"
			if status != filterSpec.Status {
				continue
			}
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func gcpStage4TalentMaskValid(paths []string, allowed map[string]struct{}) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return false
		}
		if _, ok := allowed[trimmed]; !ok {
			return false
		}
	}
	return true
}

func gcpStage4TalentParsePage(pageSize int32, pageToken string, maxPageSize, defaultPageSize int) (int, int, string, bool) {
	if pageSize < 0 || int(pageSize) > maxPageSize {
		return 0, 0, "page_size-invalid", false
	}
	start, ok := parseGCPStage4PageToken(pageToken)
	if !ok {
		return 0, 0, "page_token-invalid", false
	}
	size := int(pageSize)
	if size <= 0 {
		size = defaultPageSize
	}
	return size, start, "", true
}

func gcpStage4TalentPageRange(total, pageSize, start int) (int, string, bool) {
	if start > total {
		return 0, "", false
	}
	end := total
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < total {
		next = strconv.Itoa(end)
	}
	return end, next, true
}

func parseGCPStage4TalentProjectParent(parent string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" {
		return "", false
	}
	project = strings.TrimSpace(parts[1])
	if !isGCPTalentProjectID(project) {
		return "", false
	}
	return project, true
}

func parseGCPStage4TalentTenantName(name string) (project, tenantID string, ok bool) {
	if !isGCPTalentTenantName(name) {
		return "", "", false
	}
	parts := strings.Split(strings.Trim(name, "/"), "/")
	return parts[1], parts[3], true
}

func parseGCPStage4TalentCompanyName(name string) (project, tenantID, companyID string, ok bool) {
	if !isGCPTalentCompanyName(name) {
		return "", "", "", false
	}
	parts := strings.Split(strings.Trim(name, "/"), "/")
	return parts[1], parts[3], parts[5], true
}

func parseGCPStage4TalentJobName(name string) (project, tenantID, jobID string, ok bool) {
	if !isGCPTalentJobName(name) {
		return "", "", "", false
	}
	parts := strings.Split(strings.Trim(name, "/"), "/")
	return parts[1], parts[3], parts[5], true
}

func gcpStage4GRPCSupportCaseService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSupportGetCaseMethod:
		return gcpStage4GRPCSupportGetCase(grpcReqBody)
	case gcpSupportListCasesMethod:
		return gcpStage4GRPCSupportListCases(grpcReqBody)
	case gcpSupportSearchCasesMethod:
		return gcpStage4GRPCSupportSearchCases(grpcReqBody)
	case gcpSupportCreateCaseMethod:
		return gcpStage4GRPCSupportCreateCase(grpcReqBody)
	case gcpSupportUpdateCaseMethod:
		return gcpStage4GRPCSupportUpdateCase(grpcReqBody)
	case gcpSupportEscalateCaseMethod:
		return gcpStage4GRPCSupportEscalateCase(grpcReqBody)
	case gcpSupportCloseCaseMethod:
		return gcpStage4GRPCSupportCloseCase(grpcReqBody)
	case gcpSupportSearchCaseClassificationsMethod:
		return gcpStage4GRPCSupportSearchCaseClassifications(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCSupportCommentService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSupportListCommentsMethod:
		return gcpStage4GRPCSupportListComments(grpcReqBody)
	case gcpSupportCreateCommentMethod:
		return gcpStage4GRPCSupportCreateComment(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCSupportCaseAttachmentService(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpSupportListAttachmentsMethod:
		return gcpStage4GRPCSupportListAttachments(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCSupportGetCase(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.GetCaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	name := strings.TrimSpace(req.GetName())
	if !isGCPSupportCaseName(name) {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(name), "missing") {
		return grpcNotFound("case-not-found")
	}
	return grpcProtoSuccess(gcpStage4SupportCase(name, strings.Contains(strings.ToLower(name), "closed"), strings.Contains(strings.ToLower(name), "escalated")))
}

func gcpStage4GRPCSupportListCases(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.ListCasesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPSupportCaseParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, ok := parseGCPStage4PageToken(req.GetPageToken())
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	filter := strings.TrimSpace(req.GetFilter())
	if filter != "" && !isGCPSupportCaseFilter(filter) {
		return grpcInvalidArgument("filter-invalid")
	}

	items := []*supportpb.Case{
		gcpStage4SupportCase(fmt.Sprintf("%s/cases/case-open-1", parent), false, false),
		gcpStage4SupportCase(fmt.Sprintf("%s/cases/case-closed-1", parent), true, false),
	}
	items = gcpStage4SupportFilterCases(items, filter)
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&supportpb.ListCasesResponse{
		Cases:         items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSupportSearchCases(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.SearchCasesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPSupportCaseParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, ok := parseGCPStage4PageToken(req.GetPageToken())
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	query := strings.TrimSpace(req.GetQuery())
	if query != "" && !isGCPSupportSearchQuery(query) {
		return grpcInvalidArgument("query-invalid")
	}

	items := []*supportpb.Case{
		gcpStage4SupportCase(fmt.Sprintf("%s/cases/case-open-1", parent), false, false),
		gcpStage4SupportCase(fmt.Sprintf("%s/cases/case-open-2", parent), false, true),
		gcpStage4SupportCase(fmt.Sprintf("%s/cases/case-closed-1", parent), true, false),
	}
	items = gcpStage4SupportFilterSearchCases(items, query)
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&supportpb.SearchCasesResponse{
		Cases:         items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSupportCreateCase(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.CreateCaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPSupportCaseParent(parent) {
		return grpcInvalidArgument("parent-required")
	}
	caseReq := req.GetCase()
	if caseReq == nil {
		return grpcInvalidArgument("case-required")
	}
	if strings.TrimSpace(caseReq.GetDisplayName()) == "" {
		return grpcInvalidArgument("case.display_name-required")
	}
	if strings.TrimSpace(caseReq.GetDescription()) == "" {
		return grpcInvalidArgument("case.description-required")
	}
	if caseReq.GetClassification() == nil || strings.TrimSpace(caseReq.GetClassification().GetId()) == "" {
		return grpcInvalidArgument("case.classification.id-required")
	}
	if caseReq.GetPriority() == supportpb.Case_PRIORITY_UNSPECIFIED {
		return grpcInvalidArgument("case.priority-required")
	}

	name := strings.TrimSpace(caseReq.GetName())
	caseID := "case-created-1"
	if name != "" {
		if !strings.HasPrefix(name, parent+"/cases/") || !isGCPSupportCaseName(name) {
			return grpcInvalidArgument("case.name-invalid")
		}
		caseID = pathBase(name)
	}
	createdName := fmt.Sprintf("%s/cases/%s", parent, caseID)
	created := gcpStage4SupportCase(createdName, false, false)
	created.DisplayName = caseReq.GetDisplayName()
	created.Description = caseReq.GetDescription()
	created.Classification = &supportpb.CaseClassification{
		Id:          caseReq.GetClassification().GetId(),
		DisplayName: caseReq.GetClassification().GetDisplayName(),
	}
	created.Priority = caseReq.GetPriority()
	created.SubscriberEmailAddresses = append([]string{}, caseReq.GetSubscriberEmailAddresses()...)
	if strings.TrimSpace(caseReq.GetTimeZone()) != "" {
		created.TimeZone = caseReq.GetTimeZone()
	}
	if strings.TrimSpace(caseReq.GetLanguageCode()) != "" {
		created.LanguageCode = caseReq.GetLanguageCode()
	}
	created.TestCase = caseReq.GetTestCase()
	return grpcProtoSuccess(created)
}

func gcpStage4GRPCSupportUpdateCase(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.UpdateCaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	caseReq := req.GetCase()
	if caseReq == nil {
		return grpcInvalidArgument("case-required")
	}
	name := strings.TrimSpace(caseReq.GetName())
	if !isGCPSupportCaseName(name) {
		return grpcInvalidArgument("case.name-required")
	}
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	allowed := map[string]struct{}{
		"priority":                   {},
		"display_name":               {},
		"subscriber_email_addresses": {},
	}
	for _, path := range mask.GetPaths() {
		path = strings.TrimSpace(path)
		if _, ok := allowed[path]; !ok {
			return grpcInvalidArgument("update_mask-unsupported")
		}
	}

	closed := strings.Contains(strings.ToLower(name), "closed")
	escalated := strings.Contains(strings.ToLower(name), "escalated")
	updated := gcpStage4SupportCase(name, closed, escalated)
	for _, path := range mask.GetPaths() {
		switch strings.TrimSpace(path) {
		case "priority":
			if caseReq.GetPriority() == supportpb.Case_PRIORITY_UNSPECIFIED {
				return grpcInvalidArgument("case.priority-required")
			}
			updated.Priority = caseReq.GetPriority()
		case "display_name":
			if strings.TrimSpace(caseReq.GetDisplayName()) == "" {
				return grpcInvalidArgument("case.display_name-required")
			}
			updated.DisplayName = caseReq.GetDisplayName()
		case "subscriber_email_addresses":
			if len(caseReq.GetSubscriberEmailAddresses()) == 0 {
				return grpcInvalidArgument("case.subscriber_email_addresses-required")
			}
			updated.SubscriberEmailAddresses = append([]string{}, caseReq.GetSubscriberEmailAddresses()...)
		}
	}
	return grpcProtoSuccess(updated)
}

func gcpStage4GRPCSupportEscalateCase(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.EscalateCaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	name := strings.TrimSpace(req.GetName())
	if !isGCPSupportCaseName(name) {
		return grpcInvalidArgument("name-required")
	}
	escalation := req.GetEscalation()
	if escalation == nil {
		return grpcInvalidArgument("escalation-required")
	}
	if escalation.GetReason() == supportpb.Escalation_REASON_UNSPECIFIED {
		return grpcInvalidArgument("escalation.reason-required")
	}
	if strings.TrimSpace(escalation.GetJustification()) == "" {
		return grpcInvalidArgument("escalation.justification-required")
	}
	if strings.Contains(strings.ToLower(name), "closed") {
		return grpcFailedPrecondition("case-closed")
	}
	return grpcProtoSuccess(gcpStage4SupportCase(name, false, true))
}

func gcpStage4GRPCSupportCloseCase(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.CloseCaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	name := strings.TrimSpace(req.GetName())
	if !isGCPSupportCaseName(name) {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(name), "closed") {
		return grpcFailedPrecondition("case-already-closed")
	}
	return grpcProtoSuccess(gcpStage4SupportCase(name, true, strings.Contains(strings.ToLower(name), "escalated")))
}

func gcpStage4GRPCSupportSearchCaseClassifications(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.SearchCaseClassificationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, ok := parseGCPStage4PageToken(req.GetPageToken())
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	query := strings.TrimSpace(req.GetQuery())
	if query != "" && !isGCPSupportClassificationQuery(query) {
		return grpcInvalidArgument("query-invalid")
	}

	items := []*supportpb.CaseClassification{
		{Id: "technical-issue/compute-engine", DisplayName: "Technical Issue > Compute > Compute Engine"},
		{Id: "technical-issue/storage", DisplayName: "Technical Issue > Storage"},
		{Id: "billing-issue/invoice", DisplayName: "Billing Issue > Invoice"},
	}
	if query != "" {
		filtered := make([]*supportpb.CaseClassification, 0, len(items))
		queryLower := strings.ToLower(query)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.GetId()), queryLower) || strings.Contains(strings.ToLower(item.GetDisplayName()), queryLower) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&supportpb.SearchCaseClassificationsResponse{
		CaseClassifications: items[start:end],
		NextPageToken:       next,
	})
}

func gcpStage4GRPCSupportListComments(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.ListCommentsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPSupportCaseName(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, ok := parseGCPStage4PageToken(req.GetPageToken())
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*supportpb.Comment{
		gcpStage4SupportComment(parent, "comment-1", "Initial case triage completed."),
		gcpStage4SupportComment(parent, "comment-2", "Please provide additional logs."),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&supportpb.ListCommentsResponse{
		Comments:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSupportCreateComment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.CreateCommentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPSupportCaseName(parent) {
		return grpcInvalidArgument("parent-required")
	}
	comment := req.GetComment()
	if comment == nil {
		return grpcInvalidArgument("comment-required")
	}
	if strings.TrimSpace(comment.GetBody()) == "" {
		return grpcInvalidArgument("comment.body-required")
	}
	commentID := "comment-created-1"
	if name := strings.TrimSpace(comment.GetName()); name != "" {
		if !strings.HasPrefix(name, parent+"/comments/") {
			return grpcInvalidArgument("comment.name-invalid")
		}
		commentID = pathBase(name)
		if !isGCPSupportCommentID(commentID) {
			return grpcInvalidArgument("comment.name-invalid")
		}
	}
	return grpcProtoSuccess(gcpStage4SupportComment(parent, commentID, comment.GetBody()))
}

func gcpStage4GRPCSupportListAttachments(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &supportpb.ListAttachmentsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent := strings.TrimSpace(req.GetParent())
	if !isGCPSupportCaseName(parent) {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, ok := parseGCPStage4PageToken(req.GetPageToken())
	if !ok {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*supportpb.Attachment{
		gcpStage4SupportAttachment(parent, "attachment-1", "stacktrace.txt", "text/plain", 2048),
		gcpStage4SupportAttachment(parent, "attachment-2", "screenshot.png", "image/png", 65536),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&supportpb.ListAttachmentsResponse{
		Attachments:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4SupportFilterCases(items []*supportpb.Case, filter string) []*supportpb.Case {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return items
	}
	parts := splitGCPSupportFilterByAnd(filter)
	filtered := make([]*supportpb.Case, 0, len(items))
	for _, item := range items {
		if gcpStage4SupportCaseMatchesFilter(item, parts) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func gcpStage4SupportFilterSearchCases(items []*supportpb.Case, query string) []*supportpb.Case {
	query = strings.TrimSpace(query)
	if query == "" {
		return items
	}
	queryLower := strings.ToLower(query)
	if strings.HasPrefix(queryLower, "state=") || strings.HasPrefix(queryLower, "priority=") {
		return gcpStage4SupportFilterCases(items, query)
	}
	filtered := make([]*supportpb.Case, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.GetDisplayName()), queryLower) || strings.Contains(strings.ToLower(item.GetDescription()), queryLower) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func gcpStage4SupportCaseMatchesFilter(item *supportpb.Case, parts []string) bool {
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "state=") {
			expected := strings.TrimSpace(strings.TrimPrefix(part, "state="))
			if gcpStage4SupportStateToken(item.GetState()) != expected {
				return false
			}
			continue
		}
		if strings.HasPrefix(part, "priority=") {
			expr := strings.TrimSpace(strings.TrimPrefix(part, "priority="))
			if strings.Contains(expr, " OR ") {
				tokens := strings.Split(expr, " OR ")
				match := false
				for _, token := range tokens {
					if strings.TrimSpace(token) == item.GetPriority().String() {
						match = true
						break
					}
				}
				if !match {
					return false
				}
				continue
			}
			if expr != item.GetPriority().String() {
				return false
			}
			continue
		}
		if strings.HasPrefix(part, "creator.email=") {
			expected := strings.TrimSpace(strings.TrimPrefix(part, "creator.email="))
			expected = strings.Trim(expected, `"`+"\t ")
			if strings.TrimSpace(item.GetCreator().GetEmail()) != expected {
				return false
			}
			continue
		}
		return false
	}
	return true
}

func gcpStage4SupportStateToken(state supportpb.Case_State) string {
	if state == supportpb.Case_CLOSED {
		return "CLOSED"
	}
	return "OPEN"
}

func gcpStage4SupportCase(name string, closed, escalated bool) *supportpb.Case {
	state := supportpb.Case_NEW
	if closed {
		state = supportpb.Case_CLOSED
	}
	caseID := pathBase(name)
	created := gcpSupportReferenceTime.Add(5 * time.Minute)
	updated := created.Add(30 * time.Minute)
	if closed {
		updated = updated.Add(30 * time.Minute)
	}
	return &supportpb.Case{
		Name:        name,
		DisplayName: "Stackyard Support Case " + caseID,
		Description: "Deterministic staged support case fixture",
		Classification: &supportpb.CaseClassification{
			Id:          "technical-issue/compute-engine",
			DisplayName: "Technical Issue > Compute > Compute Engine",
		},
		TimeZone:                 "America/New_York",
		SubscriberEmailAddresses: []string{"ops@example.com"},
		State:                    state,
		CreateTime:               timestamppb.New(created),
		UpdateTime:               timestamppb.New(updated),
		Creator: &supportpb.Actor{
			DisplayName: "Stackyard Operator",
			Email:       "operator@example.com",
		},
		Escalated:    escalated,
		TestCase:     true,
		LanguageCode: "en",
		Priority:     supportpb.Case_P2,
	}
}

func gcpStage4SupportComment(parent, commentID, body string) *supportpb.Comment {
	return &supportpb.Comment{
		Name:       fmt.Sprintf("%s/comments/%s", parent, commentID),
		CreateTime: timestamppb.New(gcpSupportReferenceTime.Add(20 * time.Minute)),
		Creator: &supportpb.Actor{
			DisplayName: "Stackyard Support",
			Email:       "support@example.com",
		},
		Body: body,
	}
}

func gcpStage4SupportAttachment(parent, attachmentID, filename, mimeType string, sizeBytes int64) *supportpb.Attachment {
	return &supportpb.Attachment{
		Name:       fmt.Sprintf("%s/attachments/%s", parent, attachmentID),
		CreateTime: timestamppb.New(gcpSupportReferenceTime.Add(15 * time.Minute)),
		Creator: &supportpb.Actor{
			DisplayName: "Stackyard Support",
			Email:       "support@example.com",
		},
		Filename:  filename,
		MimeType:  mimeType,
		SizeBytes: sizeBytes,
	}
}

func gcpStage4GRPCServiceUsageEnableService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &serviceusagepb.EnableServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, serviceID, ok := parseGCPServiceUsageServiceResourceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	service := gcpStage4ServiceUsageService(project, serviceID, serviceusagepb.State_ENABLED)
	resp := &serviceusagepb.EnableServiceResponse{Service: service}
	return grpcProtoSuccess(gcpStage4ServiceUsageOperation("serviceusage-enable-"+serviceID, []string{req.GetName()}, resp))
}

func gcpStage4GRPCServiceUsageDisableService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &serviceusagepb.DisableServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, serviceID, ok := parseGCPServiceUsageServiceResourceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	switch req.GetCheckIfServiceHasUsage() {
	case serviceusagepb.DisableServiceRequest_CHECK_IF_SERVICE_HAS_USAGE_UNSPECIFIED,
		serviceusagepb.DisableServiceRequest_SKIP,
		serviceusagepb.DisableServiceRequest_CHECK:
	default:
		return grpcInvalidArgument("check_if_service_has_usage-invalid")
	}
	service := gcpStage4ServiceUsageService(project, serviceID, serviceusagepb.State_DISABLED)
	resp := &serviceusagepb.DisableServiceResponse{Service: service}
	return grpcProtoSuccess(gcpStage4ServiceUsageOperation("serviceusage-disable-"+serviceID, []string{req.GetName()}, resp))
}

func gcpStage4GRPCServiceUsageGetService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &serviceusagepb.GetServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, serviceID, ok := parseGCPServiceUsageServiceResourceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceUsageService(project, serviceID, gcpStage4ServiceUsageDefaultState(serviceID)))
}

func gcpStage4GRPCServiceUsageListServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &serviceusagepb.ListServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPServiceUsageParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 200 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	filter := strings.TrimSpace(req.GetFilter())
	if filter != "" && !isGCPServiceUsageFilter(filter) {
		return grpcInvalidArgument("filter-invalid")
	}

	items := gcpStage4ServiceUsageDefaultServices(project)
	if filter == "state:ENABLED" {
		filtered := make([]*serviceusagepb.Service, 0, len(items))
		for _, item := range items {
			if item.GetState() == serviceusagepb.State_ENABLED {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if filter == "state:DISABLED" {
		filtered := make([]*serviceusagepb.Service, 0, len(items))
		for _, item := range items {
			if item.GetState() == serviceusagepb.State_DISABLED {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&serviceusagepb.ListServicesResponse{
		Services:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCServiceUsageBatchEnableServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &serviceusagepb.BatchEnableServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPServiceUsageParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	serviceIDs := req.GetServiceIds()
	if len(serviceIDs) == 0 {
		return grpcInvalidArgument("service_ids-required")
	}
	if len(serviceIDs) > 20 {
		return grpcInvalidArgument("service_ids-too-many")
	}
	services := make([]*serviceusagepb.Service, 0, len(serviceIDs))
	resourceNames := make([]string, 0, len(serviceIDs))
	for _, serviceID := range serviceIDs {
		serviceID = strings.TrimSpace(serviceID)
		if !isGCPServiceUsageServiceID(serviceID) {
			return grpcInvalidArgument("service_ids-invalid")
		}
		serviceName := fmt.Sprintf("projects/%s/services/%s", project, serviceID)
		resourceNames = append(resourceNames, serviceName)
		services = append(services, gcpStage4ServiceUsageService(project, serviceID, serviceusagepb.State_ENABLED))
	}
	resp := &serviceusagepb.BatchEnableServicesResponse{
		Services: services,
	}
	return grpcProtoSuccess(gcpStage4ServiceUsageOperation("serviceusage-batch-enable-"+project, resourceNames, resp))
}

func gcpStage4GRPCServiceUsageBatchGetServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &serviceusagepb.BatchGetServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPServiceUsageParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	names := req.GetNames()
	if len(names) == 0 {
		return grpcInvalidArgument("names-required")
	}
	if len(names) > 30 {
		return grpcInvalidArgument("names-too-many")
	}
	services := make([]*serviceusagepb.Service, 0, len(names))
	for _, name := range names {
		nameProject, serviceID, valid := parseGCPServiceUsageServiceResourceName(name)
		if !valid {
			return grpcInvalidArgument("names-invalid")
		}
		if nameProject != project {
			return grpcInvalidArgument("names-parent-mismatch")
		}
		services = append(services, gcpStage4ServiceUsageService(project, serviceID, gcpStage4ServiceUsageDefaultState(serviceID)))
	}
	return grpcProtoSuccess(&serviceusagepb.BatchGetServicesResponse{
		Services: services,
	})
}

func gcpStage4ServiceUsageDefaultServices(project string) []*serviceusagepb.Service {
	return []*serviceusagepb.Service{
		gcpStage4ServiceUsageService(project, "serviceusage.googleapis.com", serviceusagepb.State_ENABLED),
		gcpStage4ServiceUsageService(project, "stackyard.googleapis.com", serviceusagepb.State_ENABLED),
		gcpStage4ServiceUsageService(project, "compute.googleapis.com", serviceusagepb.State_DISABLED),
	}
}

func gcpStage4ServiceUsageDefaultState(serviceID string) serviceusagepb.State {
	if strings.TrimSpace(serviceID) == "compute.googleapis.com" {
		return serviceusagepb.State_DISABLED
	}
	return serviceusagepb.State_ENABLED
}

func gcpStage4ServiceUsageService(project, serviceID string, state serviceusagepb.State) *serviceusagepb.Service {
	return &serviceusagepb.Service{
		Name:   fmt.Sprintf("projects/%s/services/%s", project, serviceID),
		Parent: "projects/" + project,
		State:  state,
		Config: &serviceusagepb.ServiceConfig{
			Name:  serviceID,
			Title: "Stackyard " + serviceID,
		},
	}
}

func gcpStage4ServiceUsageOperation(operationID string, resourceNames []string, response proto.Message) *longrunningpb.Operation {
	op := &longrunningpb.Operation{
		Name: "operations/" + operationID,
		Done: true,
	}
	if metadata, err := anypb.New(&serviceusagepb.OperationMetadata{ResourceNames: resourceNames}); err == nil {
		op.Metadata = metadata
	}
	if response != nil {
		if responseAny, err := anypb.New(response); err == nil {
			op.Result = &longrunningpb.Operation_Response{
				Response: responseAny,
			}
		}
	}
	return op
}

func gcpStage4GRPCServiceManagementListServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.ListServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 500 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	if consumerID := strings.TrimSpace(req.GetConsumerId()); consumerID != "" && !strings.HasPrefix(consumerID, "project:") {
		return grpcInvalidArgument("consumer_id-invalid")
	}
	if producerProjectID := strings.TrimSpace(req.GetProducerProjectId()); producerProjectID != "" && !gcpServiceManagementProjectIDRegex.MatchString(producerProjectID) {
		return grpcInvalidArgument("producer_project_id-invalid")
	}

	items := []*servicemanagementpb.ManagedService{
		gcpStage4ServiceManagementManagedService("stackyard.googleapis.com"),
		gcpStage4ServiceManagementManagedService("aux.stackyard.googleapis.com"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicemanagementpb.ListServicesResponse{
		Services:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCServiceManagementGetService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.GetServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	serviceName := strings.TrimSpace(req.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service_name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementManagedService(serviceName))
}

func gcpStage4GRPCServiceManagementCreateService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.CreateServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	service := req.GetService()
	if service == nil {
		return grpcInvalidArgument("service-required")
	}
	serviceName := strings.TrimSpace(service.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service.service_name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementOperation("servicemanagement-create-service"))
}

func gcpStage4GRPCServiceManagementDeleteService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.DeleteServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPServiceManagementServiceName(req.GetServiceName()) {
		return grpcInvalidArgument("service_name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementOperation("servicemanagement-delete-service"))
}

func gcpStage4GRPCServiceManagementUndeleteService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.UndeleteServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPServiceManagementServiceName(req.GetServiceName()) {
		return grpcInvalidArgument("service_name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementOperation("servicemanagement-undelete-service"))
}

func gcpStage4GRPCServiceManagementListServiceConfigs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.ListServiceConfigsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	serviceName := strings.TrimSpace(req.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service_name-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*serviceconfigpb.Service{
		gcpStage4ServiceManagementServiceConfig(serviceName, "2026-01-01r0"),
		gcpStage4ServiceManagementServiceConfig(serviceName, "2026-01-02r0"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicemanagementpb.ListServiceConfigsResponse{
		ServiceConfigs: items[start:end],
		NextPageToken:  next,
	})
}

func gcpStage4GRPCServiceManagementGetServiceConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.GetServiceConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	serviceName := strings.TrimSpace(req.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service_name-required")
	}
	configID := strings.TrimSpace(req.GetConfigId())
	if configID == "" {
		return grpcInvalidArgument("config_id-required")
	}
	switch req.GetView() {
	case servicemanagementpb.GetServiceConfigRequest_BASIC,
		servicemanagementpb.GetServiceConfigRequest_FULL:
	default:
		return grpcInvalidArgument("view-invalid")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementServiceConfig(serviceName, configID))
}

func gcpStage4GRPCServiceManagementCreateServiceConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.CreateServiceConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	serviceName := strings.TrimSpace(req.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service_name-required")
	}
	serviceConfig := req.GetServiceConfig()
	if serviceConfig == nil {
		return grpcInvalidArgument("service_config-required")
	}
	if cfgName := strings.TrimSpace(serviceConfig.GetName()); cfgName != "" && cfgName != serviceName {
		return grpcInvalidArgument("service_config.name-must-match-service_name")
	}
	configID := strings.TrimSpace(serviceConfig.GetId())
	if configID == "" {
		configID = "2026-01-03r0"
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementServiceConfig(serviceName, configID))
}

func gcpStage4GRPCServiceManagementSubmitConfigSource(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.SubmitConfigSourceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	serviceName := strings.TrimSpace(req.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service_name-required")
	}
	configSource := req.GetConfigSource()
	if configSource == nil {
		return grpcInvalidArgument("config_source-required")
	}
	for _, file := range configSource.GetFiles() {
		if file == nil || strings.TrimSpace(file.GetFilePath()) == "" {
			return grpcInvalidArgument("config_source.files.file_path-required")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementOperation("servicemanagement-submit-config-source"))
}

func gcpStage4GRPCServiceManagementListServiceRollouts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.ListServiceRolloutsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	serviceName := strings.TrimSpace(req.GetServiceName())
	if !isGCPServiceManagementServiceName(serviceName) {
		return grpcInvalidArgument("service_name-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	filter := strings.TrimSpace(req.GetFilter())
	if filter == "" {
		return grpcInvalidArgument("filter-required")
	}
	if !isGCPServiceManagementRolloutFilter(filter) {
		return grpcInvalidArgument("filter-invalid")
	}

	items := []*servicemanagementpb.Rollout{
		gcpStage4ServiceManagementRollout("2026-01-01r0"),
		gcpStage4ServiceManagementRollout("2026-01-02r0"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicemanagementpb.ListServiceRolloutsResponse{
		Rollouts:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCServiceManagementGetServiceRollout(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.GetServiceRolloutRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPServiceManagementServiceName(req.GetServiceName()) {
		return grpcInvalidArgument("service_name-required")
	}
	if strings.TrimSpace(req.GetRolloutId()) == "" {
		return grpcInvalidArgument("rollout_id-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementRollout(req.GetRolloutId()))
}

func gcpStage4GRPCServiceManagementCreateServiceRollout(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.CreateServiceRolloutRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if !isGCPServiceManagementServiceName(req.GetServiceName()) {
		return grpcInvalidArgument("service_name-required")
	}
	rollout := req.GetRollout()
	if rollout == nil {
		return grpcInvalidArgument("rollout-required")
	}
	if strings.TrimSpace(rollout.GetRolloutId()) == "" {
		return grpcInvalidArgument("rollout.rollout_id-required")
	}
	if rollout.GetTrafficPercentStrategy() == nil && rollout.GetDeleteServiceStrategy() == nil {
		return grpcInvalidArgument("rollout.strategy-required")
	}
	if rollout.GetTrafficPercentStrategy() != nil && rollout.GetDeleteServiceStrategy() != nil {
		return grpcInvalidArgument("rollout.strategy-must-be-single")
	}
	return grpcProtoSuccess(gcpStage4ServiceManagementOperation("servicemanagement-create-rollout"))
}

func gcpStage4GRPCServiceManagementGenerateConfigReport(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicemanagementpb.GenerateConfigReportRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetNewConfig() == nil {
		return grpcInvalidArgument("new_config-required")
	}
	return grpcProtoSuccess(&servicemanagementpb.GenerateConfigReportResponse{
		ServiceName:   "stackyard.googleapis.com",
		Id:            "report-2026-01-01",
		ChangeReports: []*servicemanagementpb.ChangeReport{{}},
		Diagnostics: []*servicemanagementpb.Diagnostic{
			{
				Location: "service.yaml:1",
				Kind:     servicemanagementpb.Diagnostic_WARNING,
				Message:  "staged configuration warning",
			},
		},
	})
}

func gcpStage4ServiceManagementManagedService(serviceName string) *servicemanagementpb.ManagedService {
	return &servicemanagementpb.ManagedService{
		ServiceName:       serviceName,
		ProducerProjectId: "stackyard-project",
	}
}

func gcpStage4ServiceManagementServiceConfig(serviceName, configID string) *serviceconfigpb.Service {
	return &serviceconfigpb.Service{
		Name:              serviceName,
		Id:                configID,
		Title:             "Stackyard Service Config",
		ProducerProjectId: "stackyard-project",
		Documentation: &serviceconfigpb.Documentation{
			Summary: "Stackyard staged service configuration",
		},
	}
}

func gcpStage4ServiceManagementRollout(rolloutID string) *servicemanagementpb.Rollout {
	return &servicemanagementpb.Rollout{
		RolloutId:  rolloutID,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(10 * time.Minute)),
		CreatedBy:  "stackyard@example.com",
		Status:     servicemanagementpb.Rollout_SUCCESS,
		Strategy: &servicemanagementpb.Rollout_TrafficPercentStrategy_{
			TrafficPercentStrategy: &servicemanagementpb.Rollout_TrafficPercentStrategy{
				Percentages: map[string]float64{
					"2026-01-01r0": 100,
				},
			},
		},
	}
}

func gcpStage4ServiceManagementOperation(operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", operationID),
		Done: false,
	}
}

func gcpStage4GRPCServiceHealthListEvents(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicehealthpb.ListEventsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ServiceHealthProjectParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	switch req.GetView() {
	case servicehealthpb.EventView_EVENT_VIEW_UNSPECIFIED,
		servicehealthpb.EventView_EVENT_VIEW_BASIC,
		servicehealthpb.EventView_EVENT_VIEW_FULL:
	default:
		return grpcInvalidArgument("view-invalid")
	}
	if isGCPServiceHealthMalformedFilter(req.GetFilter()) {
		return grpcInvalidArgument("filter-invalid")
	}
	includeFull := req.GetView() != servicehealthpb.EventView_EVENT_VIEW_BASIC
	items := []*servicehealthpb.Event{
		gcpStage4ServiceHealthEvent(project, location, "event-1", includeFull),
		gcpStage4ServiceHealthEvent(project, location, "event-2", includeFull),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicehealthpb.ListEventsResponse{
		Events:        items[start:end],
		NextPageToken: next,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCServiceHealthGetEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicehealthpb.GetEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, eventID, ok := parseGCPStage4ServiceHealthEventName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceHealthEvent(project, location, eventID, true))
}

func gcpStage4GRPCServiceHealthListOrganizationEvents(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicehealthpb.ListOrganizationEventsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPStage4ServiceHealthOrganizationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	switch req.GetView() {
	case servicehealthpb.OrganizationEventView_ORGANIZATION_EVENT_VIEW_UNSPECIFIED,
		servicehealthpb.OrganizationEventView_ORGANIZATION_EVENT_VIEW_BASIC,
		servicehealthpb.OrganizationEventView_ORGANIZATION_EVENT_VIEW_FULL:
	default:
		return grpcInvalidArgument("view-invalid")
	}
	if isGCPServiceHealthMalformedFilter(req.GetFilter()) {
		return grpcInvalidArgument("filter-invalid")
	}
	includeFull := req.GetView() != servicehealthpb.OrganizationEventView_ORGANIZATION_EVENT_VIEW_BASIC
	items := []*servicehealthpb.OrganizationEvent{
		gcpStage4ServiceHealthOrganizationEvent(orgID, location, "org-event-1", includeFull),
		gcpStage4ServiceHealthOrganizationEvent(orgID, location, "org-event-2", includeFull),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicehealthpb.ListOrganizationEventsResponse{
		OrganizationEvents: items[start:end],
		NextPageToken:      next,
		Unreachable:        []string{},
	})
}

func gcpStage4GRPCServiceHealthGetOrganizationEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicehealthpb.GetOrganizationEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, eventID, ok := parseGCPStage4ServiceHealthOrganizationEventName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceHealthOrganizationEvent(orgID, location, eventID, true))
}

func gcpStage4GRPCServiceHealthListOrganizationImpacts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicehealthpb.ListOrganizationImpactsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPStage4ServiceHealthOrganizationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 100 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	if isGCPServiceHealthMalformedFilter(req.GetFilter()) {
		return grpcInvalidArgument("filter-invalid")
	}
	items := []*servicehealthpb.OrganizationImpact{
		gcpStage4ServiceHealthOrganizationImpact(orgID, location, "impact-1"),
		gcpStage4ServiceHealthOrganizationImpact(orgID, location, "impact-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicehealthpb.ListOrganizationImpactsResponse{
		OrganizationImpacts: items[start:end],
		NextPageToken:       next,
		Unreachable:         []string{},
	})
}

func gcpStage4GRPCServiceHealthGetOrganizationImpact(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicehealthpb.GetOrganizationImpactRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, impactID, ok := parseGCPStage4ServiceHealthOrganizationImpactName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceHealthOrganizationImpact(orgID, location, impactID))
}

func gcpStage4ServiceHealthEvent(project, location, eventID string, includeFull bool) *servicehealthpb.Event {
	event := &servicehealthpb.Event{
		Name:             fmt.Sprintf("projects/%s/locations/%s/events/%s", project, location, eventID),
		Title:            "Service disruption " + eventID,
		Description:      "Stackyard staged project service health event",
		Category:         servicehealthpb.Event_INCIDENT,
		DetailedCategory: servicehealthpb.Event_CONFIRMED_INCIDENT,
		State:            servicehealthpb.Event_ACTIVE,
		DetailedState:    servicehealthpb.Event_CONFIRMED,
		EventImpacts: []*servicehealthpb.EventImpact{
			{
				Product: &servicehealthpb.Product{
					ProductName: "Google Kubernetes Engine",
					Id:          "gke",
				},
				Location: &servicehealthpb.Location{
					LocationName: location,
				},
			},
		},
		Relevance:      servicehealthpb.Event_RELATED,
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime.Add(20 * time.Minute)),
		StartTime:      timestamppb.New(gcpStage4ReferenceTime),
		EndTime:        timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
		NextUpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(30 * time.Minute)),
	}
	if includeFull {
		event.Updates = []*servicehealthpb.EventUpdate{
			{
				UpdateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(10 * time.Minute)),
				Title:       "Investigating",
				Description: "Google is investigating elevated error rates.",
				Symptom:     "Intermittent request failures",
				Workaround:  "Retry with exponential backoff.",
			},
		}
	}
	return event
}

func gcpStage4ServiceHealthOrganizationEvent(orgID, location, eventID string, includeFull bool) *servicehealthpb.OrganizationEvent {
	event := &servicehealthpb.OrganizationEvent{
		Name:             fmt.Sprintf("organizations/%s/locations/%s/organizationEvents/%s", orgID, location, eventID),
		Title:            "Organization incident " + eventID,
		Description:      "Stackyard staged organization service health event",
		Category:         servicehealthpb.OrganizationEvent_INCIDENT,
		DetailedCategory: servicehealthpb.OrganizationEvent_CONFIRMED_INCIDENT,
		State:            servicehealthpb.OrganizationEvent_ACTIVE,
		DetailedState:    servicehealthpb.OrganizationEvent_CONFIRMED,
		EventImpacts: []*servicehealthpb.EventImpact{
			{
				Product: &servicehealthpb.Product{
					ProductName: "Cloud Storage",
					Id:          "storage",
				},
				Location: &servicehealthpb.Location{
					LocationName: location,
				},
			},
		},
		UpdateTime:     timestamppb.New(gcpStage4ReferenceTime.Add(25 * time.Minute)),
		StartTime:      timestamppb.New(gcpStage4ReferenceTime),
		EndTime:        timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
		NextUpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(35 * time.Minute)),
	}
	if includeFull {
		event.Updates = []*servicehealthpb.EventUpdate{
			{
				UpdateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(12 * time.Minute)),
				Title:       "Mitigation in progress",
				Description: "Traffic engineering changes are being rolled out.",
				Symptom:     "Increased latency in regional endpoints",
				Workaround:  "Use alternate region where possible.",
			},
		}
	}
	return event
}

func gcpStage4ServiceHealthOrganizationImpact(orgID, location, impactID string) *servicehealthpb.OrganizationImpact {
	return &servicehealthpb.OrganizationImpact{
		Name: fmt.Sprintf("organizations/%s/locations/%s/organizationImpacts/%s", orgID, location, impactID),
		Events: []string{
			fmt.Sprintf("organizations/%s/locations/%s/organizationEvents/org-event-1", orgID, location),
		},
		Asset: &servicehealthpb.Asset{
			AssetName: "//cloudresourcemanager.googleapis.com/projects/123456789",
			AssetType: "cloudresourcemanager.googleapis.com/Project",
		},
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(40 * time.Minute)),
	}
}

func gcpStage4GRPCServiceDirectoryCreateNamespace(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.CreateNamespaceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ServiceDirectoryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	namespaceID := strings.TrimSpace(req.GetNamespaceId())
	if !isGCPServiceDirectoryID(namespaceID) {
		return grpcInvalidArgument("namespace_id-required")
	}
	namespace := req.GetNamespace()
	if namespace == nil {
		return grpcInvalidArgument("namespace-required")
	}
	if name := strings.TrimSpace(namespace.GetName()); name != "" {
		bodyProject, bodyLocation, bodyNamespaceID, valid := parseGCPStage4ServiceDirectoryNamespaceName(name)
		if !valid {
			return grpcInvalidArgument("namespace.name-invalid")
		}
		if bodyProject != project || bodyLocation != location || bodyNamespaceID != namespaceID {
			return grpcInvalidArgument("namespace.name-must-match-parent-and-namespace_id")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryNamespaceFromRequest(project, location, namespaceID, namespace))
}

func gcpStage4GRPCServiceDirectoryListNamespaces(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.ListNamespacesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4ServiceDirectoryLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*servicedirectorypb.Namespace{
		gcpStage4ServiceDirectoryNamespace(project, location, "ns-1"),
		gcpStage4ServiceDirectoryNamespace(project, location, "ns-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicedirectorypb.ListNamespacesResponse{
		Namespaces:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCServiceDirectoryGetNamespace(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.GetNamespaceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, ok := parseGCPStage4ServiceDirectoryNamespaceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryNamespace(project, location, namespaceID))
}

func gcpStage4GRPCServiceDirectoryUpdateNamespace(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.UpdateNamespaceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	namespace := req.GetNamespace()
	if namespace == nil {
		return grpcInvalidArgument("namespace-required")
	}
	project, location, namespaceID, ok := parseGCPStage4ServiceDirectoryNamespaceName(namespace.GetName())
	if !ok {
		return grpcInvalidArgument("namespace.name-required")
	}
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range mask.GetPaths() {
		if strings.TrimSpace(path) != "labels" {
			return grpcInvalidArgument("update_mask-unsupported")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryNamespaceFromRequest(project, location, namespaceID, namespace))
}

func gcpStage4GRPCServiceDirectoryDeleteNamespace(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.DeleteNamespaceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4ServiceDirectoryNamespaceName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCServiceDirectoryCreateService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.CreateServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, ok := parseGCPStage4ServiceDirectoryServiceParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	serviceID := strings.TrimSpace(req.GetServiceId())
	if !isGCPServiceDirectoryID(serviceID) {
		return grpcInvalidArgument("service_id-required")
	}
	service := req.GetService()
	if service == nil {
		return grpcInvalidArgument("service-required")
	}
	if name := strings.TrimSpace(service.GetName()); name != "" {
		bodyProject, bodyLocation, bodyNamespaceID, bodyServiceID, valid := parseGCPStage4ServiceDirectoryServiceName(name)
		if !valid {
			return grpcInvalidArgument("service.name-invalid")
		}
		if bodyProject != project || bodyLocation != location || bodyNamespaceID != namespaceID || bodyServiceID != serviceID {
			return grpcInvalidArgument("service.name-must-match-parent-and-service_id")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryServiceFromRequest(project, location, namespaceID, serviceID, service))
}

func gcpStage4GRPCServiceDirectoryListServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.ListServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, ok := parseGCPStage4ServiceDirectoryNamespaceName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*servicedirectorypb.Service{
		gcpStage4ServiceDirectoryService(project, location, namespaceID, "svc-1"),
		gcpStage4ServiceDirectoryService(project, location, namespaceID, "svc-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicedirectorypb.ListServicesResponse{
		Services:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCServiceDirectoryGetService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.GetServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, serviceID, ok := parseGCPStage4ServiceDirectoryServiceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryService(project, location, namespaceID, serviceID))
}

func gcpStage4GRPCServiceDirectoryUpdateService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.UpdateServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	service := req.GetService()
	if service == nil {
		return grpcInvalidArgument("service-required")
	}
	project, location, namespaceID, serviceID, ok := parseGCPStage4ServiceDirectoryServiceName(service.GetName())
	if !ok {
		return grpcInvalidArgument("service.name-required")
	}
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range mask.GetPaths() {
		if strings.TrimSpace(path) != "annotations" {
			return grpcInvalidArgument("update_mask-unsupported")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryServiceFromRequest(project, location, namespaceID, serviceID, service))
}

func gcpStage4GRPCServiceDirectoryDeleteService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.DeleteServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPStage4ServiceDirectoryServiceName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCServiceDirectoryCreateEndpoint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.CreateEndpointRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, serviceID, ok := parseGCPStage4ServiceDirectoryEndpointParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	endpointID := strings.TrimSpace(req.GetEndpointId())
	if !isGCPServiceDirectoryID(endpointID) {
		return grpcInvalidArgument("endpoint_id-required")
	}
	endpoint := req.GetEndpoint()
	if reason := gcpStage4ServiceDirectoryValidateEndpoint(endpoint, false); reason != "" {
		return grpcInvalidArgument(reason)
	}
	if name := strings.TrimSpace(endpoint.GetName()); name != "" {
		bodyProject, bodyLocation, bodyNamespaceID, bodyServiceID, bodyEndpointID, valid := parseGCPStage4ServiceDirectoryEndpointName(name)
		if !valid {
			return grpcInvalidArgument("endpoint.name-invalid")
		}
		if bodyProject != project || bodyLocation != location || bodyNamespaceID != namespaceID || bodyServiceID != serviceID || bodyEndpointID != endpointID {
			return grpcInvalidArgument("endpoint.name-must-match-parent-and-endpoint_id")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryEndpointFromRequest(project, location, namespaceID, serviceID, endpointID, endpoint))
}

func gcpStage4GRPCServiceDirectoryListEndpoints(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.ListEndpointsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, serviceID, ok := parseGCPStage4ServiceDirectoryServiceName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*servicedirectorypb.Endpoint{
		gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-1"),
		gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&servicedirectorypb.ListEndpointsResponse{
		Endpoints:     items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCServiceDirectoryGetEndpoint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.GetEndpointRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, serviceID, endpointID, ok := parseGCPStage4ServiceDirectoryEndpointName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, endpointID))
}

func gcpStage4GRPCServiceDirectoryUpdateEndpoint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.UpdateEndpointRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	endpoint := req.GetEndpoint()
	if reason := gcpStage4ServiceDirectoryValidateEndpoint(endpoint, true); reason != "" {
		return grpcInvalidArgument(reason)
	}
	project, location, namespaceID, serviceID, endpointID, ok := parseGCPStage4ServiceDirectoryEndpointName(endpoint.GetName())
	if !ok {
		return grpcInvalidArgument("endpoint.name-required")
	}
	mask := req.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, path := range mask.GetPaths() {
		switch strings.TrimSpace(path) {
		case "address", "port", "annotations", "network":
		default:
			return grpcInvalidArgument("update_mask-unsupported")
		}
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryEndpointFromRequest(project, location, namespaceID, serviceID, endpointID, endpoint))
}

func gcpStage4GRPCServiceDirectoryDeleteEndpoint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.DeleteEndpointRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, _, ok := parseGCPStage4ServiceDirectoryEndpointName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCServiceDirectoryGetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.GetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, _, ok := parseGCPStage4ServiceDirectoryIAMResource(req.GetResource()); !ok {
		return grpcInvalidArgument("resource-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryPolicy(req.GetResource(), nil))
}

func gcpStage4GRPCServiceDirectorySetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.SetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, _, ok := parseGCPStage4ServiceDirectoryIAMResource(req.GetResource()); !ok {
		return grpcInvalidArgument("resource-required")
	}
	if req.GetPolicy() == nil {
		return grpcInvalidArgument("policy-required")
	}
	return grpcProtoSuccess(gcpStage4ServiceDirectoryPolicy(req.GetResource(), req.GetPolicy()))
}

func gcpStage4GRPCServiceDirectoryTestIAMPermissions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.TestIamPermissionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, _, ok := parseGCPStage4ServiceDirectoryIAMResource(req.GetResource()); !ok {
		return grpcInvalidArgument("resource-required")
	}
	if len(req.GetPermissions()) == 0 {
		return grpcInvalidArgument("permissions-required")
	}
	for _, permission := range req.GetPermissions() {
		if strings.TrimSpace(permission) == "" {
			return grpcInvalidArgument("permissions-invalid")
		}
	}
	return grpcProtoSuccess(&iampb.TestIamPermissionsResponse{Permissions: req.GetPermissions()})
}

func gcpStage4GRPCServiceDirectoryResolveService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &servicedirectorypb.ResolveServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, namespaceID, serviceID, ok := parseGCPStage4ServiceDirectoryServiceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetMaxEndpoints() < 0 || req.GetMaxEndpoints() > 1000 {
		return grpcInvalidArgument("max_endpoints-invalid")
	}
	if endpointFilter := strings.TrimSpace(req.GetEndpointFilter()); strings.Contains(endpointFilter, "!!") || strings.Contains(endpointFilter, "\n") {
		return grpcInvalidArgument("endpoint_filter-invalid")
	}
	service := gcpStage4ServiceDirectoryService(project, location, namespaceID, serviceID)
	endpoints := []*servicedirectorypb.Endpoint{
		gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-1"),
		gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-2"),
	}
	if req.GetMaxEndpoints() > 0 && int(req.GetMaxEndpoints()) < len(endpoints) {
		endpoints = endpoints[:int(req.GetMaxEndpoints())]
	}
	service.Endpoints = endpoints
	return grpcProtoSuccess(&servicedirectorypb.ResolveServiceResponse{
		Service: service,
	})
}

func gcpStage4ServiceDirectoryValidateEndpoint(endpoint *servicedirectorypb.Endpoint, requireName bool) string {
	if endpoint == nil {
		return "endpoint-required"
	}
	if requireName && strings.TrimSpace(endpoint.GetName()) == "" {
		return "endpoint.name-required"
	}
	if endpoint.GetPort() < 0 || endpoint.GetPort() > 65535 {
		return "endpoint.port-invalid"
	}
	if strings.Contains(endpoint.GetAddress(), "/") {
		return "endpoint.address-invalid"
	}
	return ""
}

func gcpStage4ServiceDirectoryNamespace(project, location, namespaceID string) *servicedirectorypb.Namespace {
	return &servicedirectorypb.Namespace{
		Name: gcpServiceDirectoryNamespaceName(project, location, namespaceID),
		Labels: map[string]string{
			"env": "stackyard",
		},
		Uid: "namespace-" + namespaceID,
	}
}

func gcpStage4ServiceDirectoryNamespaceFromRequest(project, location, namespaceID string, req *servicedirectorypb.Namespace) *servicedirectorypb.Namespace {
	namespace := gcpStage4ServiceDirectoryNamespace(project, location, namespaceID)
	if req == nil {
		return namespace
	}
	if len(req.GetLabels()) > 0 {
		namespace.Labels = req.GetLabels()
	}
	return namespace
}

func gcpStage4ServiceDirectoryService(project, location, namespaceID, serviceID string) *servicedirectorypb.Service {
	return &servicedirectorypb.Service{
		Name: gcpServiceDirectoryServiceName(project, location, namespaceID, serviceID),
		Annotations: map[string]string{
			"owner": "stackyard",
		},
		Uid: "service-" + serviceID,
	}
}

func gcpStage4ServiceDirectoryServiceFromRequest(project, location, namespaceID, serviceID string, req *servicedirectorypb.Service) *servicedirectorypb.Service {
	service := gcpStage4ServiceDirectoryService(project, location, namespaceID, serviceID)
	if req == nil {
		return service
	}
	if len(req.GetAnnotations()) > 0 {
		service.Annotations = req.GetAnnotations()
	}
	return service
}

func gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, endpointID string) *servicedirectorypb.Endpoint {
	return &servicedirectorypb.Endpoint{
		Name:    gcpServiceDirectoryEndpointName(project, location, namespaceID, serviceID, endpointID),
		Address: "10.10.0.8",
		Port:    8080,
		Annotations: map[string]string{
			"backend": "primary",
		},
		Network: "projects/1234567890/locations/global/networks/default",
		Uid:     "endpoint-" + endpointID,
	}
}

func gcpStage4ServiceDirectoryEndpointFromRequest(project, location, namespaceID, serviceID, endpointID string, req *servicedirectorypb.Endpoint) *servicedirectorypb.Endpoint {
	endpoint := gcpStage4ServiceDirectoryEndpoint(project, location, namespaceID, serviceID, endpointID)
	if req == nil {
		return endpoint
	}
	if address := strings.TrimSpace(req.GetAddress()); address != "" {
		endpoint.Address = address
	}
	if req.GetPort() != 0 {
		endpoint.Port = req.GetPort()
	}
	if len(req.GetAnnotations()) > 0 {
		endpoint.Annotations = req.GetAnnotations()
	}
	if network := strings.TrimSpace(req.GetNetwork()); network != "" {
		endpoint.Network = network
	}
	return endpoint
}

func gcpStage4ServiceDirectoryPolicy(resource string, in *iampb.Policy) *iampb.Policy {
	policy := &iampb.Policy{
		Version: 1,
		Etag:    []byte("etag-" + strings.ReplaceAll(resource, "/", "-")),
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/servicedirectory.viewer",
				Members: []string{"user:stackyard@example.com"},
			},
		},
	}
	if in != nil {
		if in.GetVersion() != 0 {
			policy.Version = in.GetVersion()
		}
		if len(in.GetBindings()) > 0 {
			policy.Bindings = in.GetBindings()
		}
		if len(in.GetEtag()) > 0 {
			policy.Etag = in.GetEtag()
		}
	}
	return policy
}

func gcpStage4GRPCSecretManagerListSecrets(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.ListSecretsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPStage4SecretManagerParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 25000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*secretmanagerpb.Secret{
		gcpStage4SecretManagerSecret(project, location, "secret-1"),
		gcpStage4SecretManagerSecret(project, location, "secret-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&secretmanagerpb.ListSecretsResponse{
		Secrets:       items[start:end],
		NextPageToken: next,
		TotalSize:     int32(len(items)),
	})
}

func gcpStage4GRPCSecretManagerCreateSecret(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.CreateSecretRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPStage4SecretManagerParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	secretID := strings.TrimSpace(req.GetSecretId())
	if !isGCPSecretManagerID(secretID) {
		return grpcInvalidArgument("secret_id-required")
	}
	secret := req.GetSecret()
	if secret == nil {
		return grpcInvalidArgument("secret-required")
	}
	if secret.GetReplication() == nil {
		return grpcInvalidArgument("secret.replication-required")
	}
	if name := strings.TrimSpace(secret.GetName()); name != "" {
		bodyProject, bodyLocation, bodySecretID, _, valid := parseGCPStage4SecretManagerSecretName(name)
		if !valid {
			return grpcInvalidArgument("secret.name-invalid")
		}
		if bodyProject != project || bodyLocation != location || bodySecretID != secretID {
			return grpcInvalidArgument("secret.name-must-match-parent-and-secret_id")
		}
	}
	return grpcProtoSuccess(gcpStage4SecretManagerSecretFromRequest(project, location, secretID, secret))
}

func gcpStage4GRPCSecretManagerAddSecretVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.AddSecretVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, _, ok := parseGCPStage4SecretManagerSecretName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPayload() == nil || len(req.GetPayload().GetData()) == 0 {
		return grpcInvalidArgument("payload.data-required")
	}
	version := gcpStage4SecretManagerVersion(project, location, secretID, "4")
	version.ClientSpecifiedPayloadChecksum = req.GetPayload().DataCrc32C != nil
	return grpcProtoSuccess(version)
}

func gcpStage4GRPCSecretManagerGetSecret(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.GetSecretRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, _, ok := parseGCPStage4SecretManagerSecretName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecretManagerSecret(project, location, secretID))
}

func gcpStage4GRPCSecretManagerUpdateSecret(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.UpdateSecretRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	secret := req.GetSecret()
	if secret == nil {
		return grpcInvalidArgument("secret-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	project, location, secretID, _, ok := parseGCPStage4SecretManagerSecretName(secret.GetName())
	if !ok {
		return grpcInvalidArgument("secret.name-required")
	}
	if secret.GetReplication() == nil {
		return grpcInvalidArgument("secret.replication-required")
	}
	return grpcProtoSuccess(gcpStage4SecretManagerSecretFromRequest(project, location, secretID, secret))
}

func gcpStage4GRPCSecretManagerDeleteSecret(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.DeleteSecretRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPStage4SecretManagerSecretName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCSecretManagerListSecretVersions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.ListSecretVersionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, _, ok := parseGCPStage4SecretManagerSecretName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 25000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*secretmanagerpb.SecretVersion{
		gcpStage4SecretManagerVersion(project, location, secretID, "1"),
		gcpStage4SecretManagerVersion(project, location, secretID, "2"),
		gcpStage4SecretManagerVersion(project, location, secretID, "3"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&secretmanagerpb.ListSecretVersionsResponse{
		Versions:      items[start:end],
		NextPageToken: next,
		TotalSize:     int32(len(items)),
	})
}

func gcpStage4GRPCSecretManagerGetSecretVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.GetSecretVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, versionID, _, ok := parseGCPStage4SecretManagerVersionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecretManagerVersion(project, location, secretID, versionID))
}

func gcpStage4GRPCSecretManagerAccessSecretVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.AccessSecretVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, versionID, _, ok := parseGCPStage4SecretManagerVersionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	if gcpSecretManagerVersionState(versionID) == "DESTROYED" {
		return grpcFailedPrecondition("version-destroyed")
	}
	crc := int64(4077700407)
	return grpcProtoSuccess(&secretmanagerpb.AccessSecretVersionResponse{
		Name: gcpStage4SecretManagerVersionResourceName(project, location, secretID, versionID),
		Payload: &secretmanagerpb.SecretPayload{
			Data:       []byte("stackyard-secret"),
			DataCrc32C: &crc,
		},
	})
}

func gcpStage4GRPCSecretManagerDisableSecretVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.DisableSecretVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, versionID, _, ok := parseGCPStage4SecretManagerVersionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	if gcpSecretManagerVersionState(versionID) == "DESTROYED" {
		return grpcFailedPrecondition("version-destroyed")
	}
	version := gcpStage4SecretManagerVersion(project, location, secretID, versionID)
	version.State = secretmanagerpb.SecretVersion_DISABLED
	return grpcProtoSuccess(version)
}

func gcpStage4GRPCSecretManagerEnableSecretVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.EnableSecretVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, versionID, _, ok := parseGCPStage4SecretManagerVersionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	if gcpSecretManagerVersionState(versionID) == "DESTROYED" {
		return grpcFailedPrecondition("version-destroyed")
	}
	version := gcpStage4SecretManagerVersion(project, location, secretID, versionID)
	version.State = secretmanagerpb.SecretVersion_ENABLED
	return grpcProtoSuccess(version)
}

func gcpStage4GRPCSecretManagerDestroySecretVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &secretmanagerpb.DestroySecretVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, secretID, versionID, _, ok := parseGCPStage4SecretManagerVersionName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	version := gcpStage4SecretManagerVersion(project, location, secretID, versionID)
	version.State = secretmanagerpb.SecretVersion_DESTROYED
	version.DestroyTime = timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour))
	return grpcProtoSuccess(version)
}

func gcpStage4GRPCSecretManagerSetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.SetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, secretID, _, ok := parseGCPStage4SecretManagerSecretName(req.GetResource())
	if !ok {
		return grpcInvalidArgument("resource-required")
	}
	if req.GetPolicy() == nil {
		return grpcInvalidArgument("policy-required")
	}
	return grpcProtoSuccess(gcpStage4SecretManagerPolicy(secretID, req.GetPolicy()))
}

func gcpStage4GRPCSecretManagerGetIAMPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.GetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, secretID, _, ok := parseGCPStage4SecretManagerSecretName(req.GetResource())
	if !ok {
		return grpcInvalidArgument("resource-required")
	}
	return grpcProtoSuccess(gcpStage4SecretManagerPolicy(secretID, nil))
}

func gcpStage4GRPCSecretManagerTestIAMPermissions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.TestIamPermissionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPStage4SecretManagerSecretName(req.GetResource()); !ok {
		return grpcInvalidArgument("resource-required")
	}
	permissions := req.GetPermissions()
	if len(permissions) == 0 {
		permissions = []string{"secretmanager.secrets.get"}
	}
	return grpcProtoSuccess(&iampb.TestIamPermissionsResponse{Permissions: permissions})
}

func gcpStage4GRPCSecurityCenterListSources(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterpb.ListSourcesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, ok := parseGCPStage4SecurityCenterParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycenterpb.Source{
		gcpStage4SecurityCenterSource(scope, scopeID, "source-1"),
		gcpStage4SecurityCenterSource(scope, scopeID, "source-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycenterpb.ListSourcesResponse{
		Sources:       items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecurityCenterGetSource(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterpb.GetSourceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, sourceID, ok := parseGCPStage4SecurityCenterSourceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterSource(scope, scopeID, sourceID))
}

func gcpStage4GRPCSecurityCenterCreateSource(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterpb.CreateSourceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, ok := parseGCPStage4SecurityCenterParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	source := req.GetSource()
	if source == nil {
		return grpcInvalidArgument("source-required")
	}
	if strings.TrimSpace(source.GetDisplayName()) == "" {
		return grpcInvalidArgument("source.display_name-required")
	}
	sourceID := "source-1"
	if strings.TrimSpace(source.GetName()) != "" {
		bodyScope, bodyScopeID, bodySourceID, valid := parseGCPStage4SecurityCenterSourceName(source.GetName())
		if !valid {
			return grpcInvalidArgument("source.name-invalid")
		}
		if bodyScope != scope || bodyScopeID != scopeID {
			return grpcInvalidArgument("source.name-must-match-parent")
		}
		sourceID = bodySourceID
	}
	resp := gcpStage4SecurityCenterSource(scope, scopeID, sourceID)
	resp.DisplayName = strings.TrimSpace(source.GetDisplayName())
	if d := strings.TrimSpace(source.GetDescription()); d != "" {
		resp.Description = d
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterSetMute(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterpb.SetMuteRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, sourceID, findingID, ok := parseGCPStage4SecurityCenterFindingName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetMute() == securitycenterpb.Finding_MUTE_UNSPECIFIED {
		return grpcInvalidArgument("mute-required")
	}
	if strings.Contains(strings.ToLower(findingID), "already-muted") && req.GetMute() == securitycenterpb.Finding_MUTED {
		return grpcFailedPrecondition("finding-already-muted")
	}
	finding := gcpStage4SecurityCenterFinding(scope, scopeID, sourceID, findingID)
	finding.Mute = req.GetMute()
	finding.MuteUpdateTime = timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute))
	return grpcProtoSuccess(finding)
}

func gcpStage4GRPCSecurityCenterV2ListSources(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterv2pb.ListSourcesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, ok := parseGCPStage4SecurityCenterParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycenterv2pb.Source{
		gcpStage4SecurityCenterV2Source(scope, scopeID, "source-1"),
		gcpStage4SecurityCenterV2Source(scope, scopeID, "source-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycenterv2pb.ListSourcesResponse{
		Sources:       items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecurityCenterV2GetSource(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterv2pb.GetSourceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, sourceID, ok := parseGCPStage4SecurityCenterSourceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterV2Source(scope, scopeID, sourceID))
}

func gcpStage4GRPCSecurityCenterV2CreateSource(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterv2pb.CreateSourceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, ok := parseGCPStage4SecurityCenterParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	source := req.GetSource()
	if source == nil {
		return grpcInvalidArgument("source-required")
	}
	if strings.TrimSpace(source.GetDisplayName()) == "" {
		return grpcInvalidArgument("source.display_name-required")
	}
	sourceID := "source-1"
	if strings.TrimSpace(source.GetName()) != "" {
		bodyScope, bodyScopeID, bodySourceID, valid := parseGCPStage4SecurityCenterSourceName(source.GetName())
		if !valid {
			return grpcInvalidArgument("source.name-invalid")
		}
		if bodyScope != scope || bodyScopeID != scopeID {
			return grpcInvalidArgument("source.name-must-match-parent")
		}
		sourceID = bodySourceID
	}
	resp := gcpStage4SecurityCenterV2Source(scope, scopeID, sourceID)
	resp.DisplayName = strings.TrimSpace(source.GetDisplayName())
	if d := strings.TrimSpace(source.GetDescription()); d != "" {
		resp.Description = d
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterV2SetMute(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycenterv2pb.SetMuteRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, sourceID, findingID, ok := parseGCPStage4SecurityCenterFindingName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetMute() == securitycenterv2pb.Finding_MUTE_UNSPECIFIED {
		return grpcInvalidArgument("mute-required")
	}
	if strings.Contains(strings.ToLower(findingID), "already-muted") && req.GetMute() == securitycenterv2pb.Finding_MUTED {
		return grpcFailedPrecondition("finding-already-muted")
	}
	finding := gcpStage4SecurityCenterV2Finding(scope, scopeID, sourceID, findingID)
	finding.Mute = req.GetMute()
	finding.MuteUpdateTime = timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute))
	return grpcProtoSuccess(finding)
}

func gcpStage4SecurityCenterSource(scope, scopeID, sourceID string) *securitycenterpb.Source {
	name := fmt.Sprintf("%s/%s/sources/%s", scope, scopeID, sourceID)
	return &securitycenterpb.Source{
		Name:          name,
		DisplayName:   "Stackyard Source " + sourceID,
		Description:   "Stackyard staged Security Command Center source",
		CanonicalName: name,
	}
}

func gcpStage4SecurityCenterFinding(scope, scopeID, sourceID, findingID string) *securitycenterpb.Finding {
	sourceName := fmt.Sprintf("%s/%s/sources/%s", scope, scopeID, sourceID)
	name := sourceName + "/findings/" + findingID
	return &securitycenterpb.Finding{
		Name:        name,
		Parent:      sourceName,
		Category:    "OPEN_FIREWALL",
		State:       securitycenterpb.Finding_ACTIVE,
		Severity:    securitycenterpb.Finding_HIGH,
		Mute:        securitycenterpb.Finding_UNMUTED,
		EventTime:   timestamppb.New(gcpStage4ReferenceTime),
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		Description: "Stackyard staged Security Command Center finding",
		SecurityMarks: &securitycenterpb.SecurityMarks{
			Name: name + "/securityMarks",
			Marks: map[string]string{
				"owner": "stackyard",
			},
		},
	}
}

func gcpStage4SecurityCenterV2Source(scope, scopeID, sourceID string) *securitycenterv2pb.Source {
	name := fmt.Sprintf("%s/%s/sources/%s", scope, scopeID, sourceID)
	return &securitycenterv2pb.Source{
		Name:          name,
		DisplayName:   "Stackyard Source " + sourceID,
		Description:   "Stackyard staged Security Command Center source",
		CanonicalName: name,
	}
}

func gcpStage4SecurityCenterV2Finding(scope, scopeID, sourceID, findingID string) *securitycenterv2pb.Finding {
	sourceName := fmt.Sprintf("%s/%s/sources/%s", scope, scopeID, sourceID)
	name := sourceName + "/findings/" + findingID
	return &securitycenterv2pb.Finding{
		Name:        name,
		Parent:      sourceName,
		Category:    "OPEN_FIREWALL",
		State:       securitycenterv2pb.Finding_ACTIVE,
		Severity:    securitycenterv2pb.Finding_HIGH,
		Mute:        securitycenterv2pb.Finding_UNMUTED,
		EventTime:   timestamppb.New(gcpStage4ReferenceTime),
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		Description: "Stackyard staged Security Command Center finding",
		SecurityMarks: &securitycenterv2pb.SecurityMarks{
			Name: name + "/securityMarks",
			Marks: map[string]string{
				"owner": "stackyard",
			},
		},
	}
}

func parseGCPStage4SecurityCenterParent(parent string) (scope, scopeID string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	if scopeID == "" {
		return "", "", false
	}
	switch scope {
	case "organizations", "folders", "projects":
		return scope, scopeID, true
	default:
		return "", "", false
	}
}

func parseGCPStage4SecurityCenterSourceName(name string) (scope, scopeID, sourceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[2] != "sources" {
		return "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	sourceID = strings.TrimSpace(parts[3])
	if scopeID == "" || sourceID == "" {
		return "", "", "", false
	}
	switch scope {
	case "organizations", "folders", "projects":
		return scope, scopeID, sourceID, true
	default:
		return "", "", "", false
	}
}

func parseGCPStage4SecurityCenterFindingName(name string) (scope, scopeID, sourceID, findingID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[2] != "sources" || parts[4] != "findings" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	sourceID = strings.TrimSpace(parts[3])
	findingID = strings.TrimSpace(parts[5])
	if scopeID == "" || sourceID == "" || findingID == "" {
		return "", "", "", "", false
	}
	switch scope {
	case "organizations", "folders", "projects":
		return scope, scopeID, sourceID, findingID, true
	default:
		return "", "", "", "", false
	}
}

func gcpStage4GRPCSecurityCenterManagementListEffectiveSHAModules(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListEffectiveSecurityHealthAnalyticsCustomModulesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycentermanagementpb.EffectiveSecurityHealthAnalyticsCustomModule{
		gcpStage4SecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, "effective-sha-module-1"),
		gcpStage4SecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, "effective-sha-module-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListEffectiveSecurityHealthAnalyticsCustomModulesResponse{
		EffectiveSecurityHealthAnalyticsCustomModules: items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecurityCenterManagementGetEffectiveSHAModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.GetEffectiveSecurityHealthAnalyticsCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementEffectiveSHAModuleName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, moduleID))
}

func gcpStage4GRPCSecurityCenterManagementListSHAModules(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListSecurityHealthAnalyticsCustomModulesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycentermanagementpb.SecurityHealthAnalyticsCustomModule{
		gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, "sha-module-1"),
		gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, "sha-module-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListSecurityHealthAnalyticsCustomModulesResponse{
		SecurityHealthAnalyticsCustomModules: items[start:end],
		NextPageToken:                        next,
	})
}

func gcpStage4GRPCSecurityCenterManagementListDescendantSHAModules(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListDescendantSecurityHealthAnalyticsCustomModulesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	item := gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, "descendant-sha-module-1")
	item.AncestorModule = gcpSecurityCenterManagementSHAModuleName(scope, scopeID, location, "sha-module-1")
	items := []*securitycentermanagementpb.SecurityHealthAnalyticsCustomModule{item}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListDescendantSecurityHealthAnalyticsCustomModulesResponse{
		SecurityHealthAnalyticsCustomModules: items[start:end],
		NextPageToken:                        next,
	})
}

func gcpStage4GRPCSecurityCenterManagementGetSHAModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.GetSecurityHealthAnalyticsCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementSHAModuleName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, moduleID))
}

func gcpStage4GRPCSecurityCenterManagementCreateSHAModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.CreateSecurityHealthAnalyticsCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	module := req.GetSecurityHealthAnalyticsCustomModule()
	if module == nil {
		return grpcInvalidArgument("security_health_analytics_custom_module-required")
	}
	if module.GetCustomConfig() == nil {
		return grpcInvalidArgument("security_health_analytics_custom_module.custom_config-required")
	}
	moduleID := "sha-module-created"
	if name := strings.TrimSpace(module.GetName()); name != "" {
		nameScope, nameScopeID, nameLocation, nameID, valid := parseGCPSecurityCenterManagementSHAModuleName(name)
		if !valid {
			return grpcInvalidArgument("security_health_analytics_custom_module.name-invalid")
		}
		if nameScope != scope || nameScopeID != scopeID || nameLocation != location {
			return grpcInvalidArgument("security_health_analytics_custom_module.name-must-match-parent")
		}
		moduleID = nameID
	}
	resp := gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, moduleID)
	if displayName := strings.TrimSpace(module.GetDisplayName()); displayName != "" {
		resp.DisplayName = displayName
	}
	if module.GetEnablementState() != securitycentermanagementpb.SecurityHealthAnalyticsCustomModule_ENABLEMENT_STATE_UNSPECIFIED {
		resp.EnablementState = module.GetEnablementState()
	}
	resp.CustomConfig = module.GetCustomConfig()
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterManagementUpdateSHAModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.UpdateSecurityHealthAnalyticsCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4SecurityCenterManagementValidateMask(req.GetUpdateMask().GetPaths(), map[string]struct{}{
		"custom_config":    {},
		"enablement_state": {},
		"display_name":     {},
		"customConfig":     {},
		"enablementState":  {},
		"displayName":      {},
	}) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	module := req.GetSecurityHealthAnalyticsCustomModule()
	if module == nil {
		return grpcInvalidArgument("security_health_analytics_custom_module-required")
	}
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementSHAModuleName(module.GetName())
	if !ok {
		return grpcInvalidArgument("security_health_analytics_custom_module.name-required")
	}
	if gcpStage4SecurityCenterManagementMaskHas(req.GetUpdateMask().GetPaths(), "custom_config", "customConfig") && module.GetCustomConfig() == nil {
		return grpcInvalidArgument("security_health_analytics_custom_module.custom_config-required")
	}
	resp := gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, moduleID)
	if displayName := strings.TrimSpace(module.GetDisplayName()); displayName != "" {
		resp.DisplayName = displayName
	}
	if module.GetEnablementState() != securitycentermanagementpb.SecurityHealthAnalyticsCustomModule_ENABLEMENT_STATE_UNSPECIFIED {
		resp.EnablementState = module.GetEnablementState()
	}
	if module.GetCustomConfig() != nil {
		resp.CustomConfig = module.GetCustomConfig()
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterManagementDeleteSHAModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.DeleteSecurityHealthAnalyticsCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPSecurityCenterManagementSHAModuleName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCSecurityCenterManagementSimulateSHAModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.SimulateSecurityHealthAnalyticsCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, _, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetCustomConfig() == nil {
		return grpcInvalidArgument("custom_config-required")
	}
	if req.GetResource() == nil || strings.TrimSpace(req.GetResource().GetResourceType()) == "" || req.GetResource().GetResourceData() == nil {
		return grpcInvalidArgument("resource-required")
	}
	return grpcProtoSuccess(&securitycentermanagementpb.SimulateSecurityHealthAnalyticsCustomModuleResponse{
		Result: &securitycentermanagementpb.SimulateSecurityHealthAnalyticsCustomModuleResponse_SimulatedResult{
			Result: &securitycentermanagementpb.SimulateSecurityHealthAnalyticsCustomModuleResponse_SimulatedResult_Finding{
				Finding: &securitycentermanagementpb.SimulatedFinding{
					Name:         fmt.Sprintf("%s/%s/sources/source-1/findings/simulated-finding-1", scope, scopeID),
					Parent:       fmt.Sprintf("%s/%s/sources/source-1", scope, scopeID),
					ResourceName: fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s/%s", scope, scopeID),
					Category:     "stackyard_custom_sha_violation",
					State:        securitycentermanagementpb.SimulatedFinding_ACTIVE,
					Severity:     securitycentermanagementpb.SimulatedFinding_HIGH,
					FindingClass: securitycentermanagementpb.SimulatedFinding_MISCONFIGURATION,
					EventTime:    timestamppb.New(gcpStage4ReferenceTime),
					SourceProperties: map[string]*structpb.Value{
						"simulated": structpb.NewBoolValue(true),
					},
				},
			},
		},
	})
}

func gcpStage4GRPCSecurityCenterManagementListEffectiveETDModules(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListEffectiveEventThreatDetectionCustomModulesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycentermanagementpb.EffectiveEventThreatDetectionCustomModule{
		gcpStage4SecurityCenterManagementEffectiveETDModule(scope, scopeID, location, "effective-etd-module-1"),
		gcpStage4SecurityCenterManagementEffectiveETDModule(scope, scopeID, location, "effective-etd-module-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListEffectiveEventThreatDetectionCustomModulesResponse{
		EffectiveEventThreatDetectionCustomModules: items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecurityCenterManagementGetEffectiveETDModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.GetEffectiveEventThreatDetectionCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementEffectiveETDModuleName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterManagementEffectiveETDModule(scope, scopeID, location, moduleID))
}

func gcpStage4GRPCSecurityCenterManagementListETDModules(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListEventThreatDetectionCustomModulesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycentermanagementpb.EventThreatDetectionCustomModule{
		gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, "etd-module-1"),
		gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, "etd-module-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListEventThreatDetectionCustomModulesResponse{
		EventThreatDetectionCustomModules: items[start:end],
		NextPageToken:                     next,
	})
}

func gcpStage4GRPCSecurityCenterManagementListDescendantETDModules(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListDescendantEventThreatDetectionCustomModulesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	item := gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, "descendant-etd-module-1")
	item.AncestorModule = gcpSecurityCenterManagementETDModuleName(scope, scopeID, location, "etd-module-1")
	items := []*securitycentermanagementpb.EventThreatDetectionCustomModule{item}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListDescendantEventThreatDetectionCustomModulesResponse{
		EventThreatDetectionCustomModules: items[start:end],
		NextPageToken:                     next,
	})
}

func gcpStage4GRPCSecurityCenterManagementGetETDModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.GetEventThreatDetectionCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementETDModuleName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, moduleID))
}

func gcpStage4GRPCSecurityCenterManagementCreateETDModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.CreateEventThreatDetectionCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	module := req.GetEventThreatDetectionCustomModule()
	if module == nil {
		return grpcInvalidArgument("event_threat_detection_custom_module-required")
	}
	if strings.TrimSpace(module.GetType()) == "" {
		return grpcInvalidArgument("event_threat_detection_custom_module.type-required")
	}
	if module.GetConfig() == nil {
		return grpcInvalidArgument("event_threat_detection_custom_module.config-required")
	}
	moduleID := "etd-module-created"
	if name := strings.TrimSpace(module.GetName()); name != "" {
		nameScope, nameScopeID, nameLocation, nameID, valid := parseGCPSecurityCenterManagementETDModuleName(name)
		if !valid {
			return grpcInvalidArgument("event_threat_detection_custom_module.name-invalid")
		}
		if nameScope != scope || nameScopeID != scopeID || nameLocation != location {
			return grpcInvalidArgument("event_threat_detection_custom_module.name-must-match-parent")
		}
		moduleID = nameID
	}
	resp := gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, moduleID)
	resp.Type = module.GetType()
	if module.GetConfig() != nil {
		resp.Config = module.GetConfig()
	}
	if displayName := strings.TrimSpace(module.GetDisplayName()); displayName != "" {
		resp.DisplayName = displayName
	}
	if description := strings.TrimSpace(module.GetDescription()); description != "" {
		resp.Description = description
	}
	if module.GetEnablementState() != securitycentermanagementpb.EventThreatDetectionCustomModule_ENABLEMENT_STATE_UNSPECIFIED {
		resp.EnablementState = module.GetEnablementState()
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterManagementUpdateETDModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.UpdateEventThreatDetectionCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4SecurityCenterManagementValidateMask(req.GetUpdateMask().GetPaths(), map[string]struct{}{
		"config":           {},
		"enablement_state": {},
		"display_name":     {},
		"description":      {},
		"enablementState":  {},
		"displayName":      {},
	}) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	module := req.GetEventThreatDetectionCustomModule()
	if module == nil {
		return grpcInvalidArgument("event_threat_detection_custom_module-required")
	}
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementETDModuleName(module.GetName())
	if !ok {
		return grpcInvalidArgument("event_threat_detection_custom_module.name-required")
	}
	if gcpStage4SecurityCenterManagementMaskHas(req.GetUpdateMask().GetPaths(), "config") && module.GetConfig() == nil {
		return grpcInvalidArgument("event_threat_detection_custom_module.config-required")
	}
	resp := gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, moduleID)
	if module.GetConfig() != nil {
		resp.Config = module.GetConfig()
	}
	if displayName := strings.TrimSpace(module.GetDisplayName()); displayName != "" {
		resp.DisplayName = displayName
	}
	if description := strings.TrimSpace(module.GetDescription()); description != "" {
		resp.Description = description
	}
	if strings.TrimSpace(module.GetType()) != "" {
		resp.Type = module.GetType()
	}
	if module.GetEnablementState() != securitycentermanagementpb.EventThreatDetectionCustomModule_ENABLEMENT_STATE_UNSPECIFIED {
		resp.EnablementState = module.GetEnablementState()
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterManagementDeleteETDModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.DeleteEventThreatDetectionCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, ok := parseGCPSecurityCenterManagementETDModuleName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCSecurityCenterManagementValidateETDModule(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ValidateEventThreatDetectionCustomModuleRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPSecurityCenterManagementParentName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetRawText()) == "" {
		return grpcInvalidArgument("raw_text-required")
	}
	if strings.TrimSpace(req.GetType()) == "" {
		return grpcInvalidArgument("type-required")
	}
	resp := &securitycentermanagementpb.ValidateEventThreatDetectionCustomModuleResponse{}
	if strings.Contains(strings.ToLower(req.GetRawText()), "invalid") || strings.Contains(strings.ToLower(req.GetRawText()), "error") {
		resp.Errors = append(resp.Errors, &securitycentermanagementpb.ValidateEventThreatDetectionCustomModuleResponse_CustomModuleValidationError{
			Description: "Raw module text failed validation",
			FieldPath:   "/rawText",
			Start:       &securitycentermanagementpb.ValidateEventThreatDetectionCustomModuleResponse_Position{LineNumber: 1, ColumnNumber: 1},
			End:         &securitycentermanagementpb.ValidateEventThreatDetectionCustomModuleResponse_Position{LineNumber: 1, ColumnNumber: 8},
		})
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCSecurityCenterManagementGetService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.GetSecurityCenterServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, serviceID, ok := parseGCPSecurityCenterManagementServiceName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !isGCPSecurityCenterManagementServiceID(serviceID) {
		return grpcInvalidArgument("name-service-invalid")
	}
	return grpcProtoSuccess(gcpStage4SecurityCenterManagementService(scope, scopeID, location, serviceID))
}

func gcpStage4GRPCSecurityCenterManagementListServices(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.ListSecurityCenterServicesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scope, scopeID, location, ok := parseGCPSecurityCenterManagementParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securitycentermanagementpb.SecurityCenterService{
		gcpStage4SecurityCenterManagementService(scope, scopeID, location, "security-health-analytics"),
		gcpStage4SecurityCenterManagementService(scope, scopeID, location, "event-threat-detection"),
		gcpStage4SecurityCenterManagementService(scope, scopeID, location, "vm-threat-detection"),
	}
	if req.GetShowEligibleModulesOnly() {
		items = items[:2]
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securitycentermanagementpb.ListSecurityCenterServicesResponse{
		SecurityCenterServices: items[start:end],
		NextPageToken:          next,
	})
}

func gcpStage4GRPCSecurityCenterManagementUpdateService(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securitycentermanagementpb.UpdateSecurityCenterServiceRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpStage4SecurityCenterManagementValidateMask(req.GetUpdateMask().GetPaths(), map[string]struct{}{
		"intended_enablement_state": {},
		"modules":                   {},
		"service_config":            {},
		"intendedEnablementState":   {},
		"serviceConfig":             {},
	}) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	service := req.GetSecurityCenterService()
	if service == nil {
		return grpcInvalidArgument("security_center_service-required")
	}
	scope, scopeID, location, serviceID, ok := parseGCPSecurityCenterManagementServiceName(service.GetName())
	if !ok {
		return grpcInvalidArgument("security_center_service.name-required")
	}
	if !isGCPSecurityCenterManagementServiceID(serviceID) {
		return grpcInvalidArgument("security_center_service.name-service-invalid")
	}
	if service.GetIntendedEnablementState() == securitycentermanagementpb.SecurityCenterService_INGEST_ONLY {
		return grpcFailedPrecondition("security_center_service.intended_enablement_state-read-only")
	}
	if gcpStage4SecurityCenterManagementMaskHas(req.GetUpdateMask().GetPaths(), "modules") && len(service.GetModules()) == 0 {
		return grpcInvalidArgument("security_center_service.modules-required")
	}
	resp := gcpStage4SecurityCenterManagementService(scope, scopeID, location, serviceID)
	if service.GetIntendedEnablementState() != securitycentermanagementpb.SecurityCenterService_ENABLEMENT_STATE_UNSPECIFIED {
		resp.IntendedEnablementState = service.GetIntendedEnablementState()
	}
	if len(service.GetModules()) > 0 {
		resp.Modules = service.GetModules()
	}
	if service.GetServiceConfig() != nil {
		resp.ServiceConfig = service.GetServiceConfig()
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4SecurityCenterManagementFixtureToProto(src map[string]any, dst proto.Message) bool {
	raw, err := json.Marshal(src)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, dst); err != nil {
		return false
	}
	return true
}

func gcpStage4SecurityCenterManagementService(scope, scopeID, location, serviceID string) *securitycentermanagementpb.SecurityCenterService {
	resp := &securitycentermanagementpb.SecurityCenterService{}
	if !gcpStage4SecurityCenterManagementFixtureToProto(gcpSecurityCenterManagementService(scope, scopeID, location, serviceID), resp) {
		return &securitycentermanagementpb.SecurityCenterService{}
	}
	return resp
}

func gcpStage4SecurityCenterManagementSHAModule(scope, scopeID, location, moduleID string) *securitycentermanagementpb.SecurityHealthAnalyticsCustomModule {
	resp := &securitycentermanagementpb.SecurityHealthAnalyticsCustomModule{}
	if !gcpStage4SecurityCenterManagementFixtureToProto(gcpSecurityCenterManagementSHAModule(scope, scopeID, location, moduleID), resp) {
		return &securitycentermanagementpb.SecurityHealthAnalyticsCustomModule{}
	}
	return resp
}

func gcpStage4SecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, moduleID string) *securitycentermanagementpb.EffectiveSecurityHealthAnalyticsCustomModule {
	resp := &securitycentermanagementpb.EffectiveSecurityHealthAnalyticsCustomModule{}
	if !gcpStage4SecurityCenterManagementFixtureToProto(gcpSecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, moduleID), resp) {
		return &securitycentermanagementpb.EffectiveSecurityHealthAnalyticsCustomModule{}
	}
	return resp
}

func gcpStage4SecurityCenterManagementETDModule(scope, scopeID, location, moduleID string) *securitycentermanagementpb.EventThreatDetectionCustomModule {
	resp := &securitycentermanagementpb.EventThreatDetectionCustomModule{}
	if !gcpStage4SecurityCenterManagementFixtureToProto(gcpSecurityCenterManagementETDModule(scope, scopeID, location, moduleID), resp) {
		return &securitycentermanagementpb.EventThreatDetectionCustomModule{}
	}
	return resp
}

func gcpStage4SecurityCenterManagementEffectiveETDModule(scope, scopeID, location, moduleID string) *securitycentermanagementpb.EffectiveEventThreatDetectionCustomModule {
	resp := &securitycentermanagementpb.EffectiveEventThreatDetectionCustomModule{}
	if !gcpStage4SecurityCenterManagementFixtureToProto(gcpSecurityCenterManagementEffectiveETDModule(scope, scopeID, location, moduleID), resp) {
		return &securitycentermanagementpb.EffectiveEventThreatDetectionCustomModule{}
	}
	return resp
}

func gcpStage4SecurityCenterManagementValidateMask(paths []string, allowed map[string]struct{}) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return false
		}
		if _, ok := allowed[trimmed]; !ok {
			return false
		}
	}
	return true
}

func gcpStage4SecurityCenterManagementMaskHas(paths []string, candidates ...string) bool {
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		for _, candidate := range candidates {
			if trimmed == candidate {
				return true
			}
		}
	}
	return false
}

func gcpStage4GRPCSecurityPostureListPostures(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.ListPosturesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPSecurityPostureParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	pageSize, start, errReason, ok := gcpStage4SecurityPostureParsePage(req.GetPageSize(), req.GetPageToken())
	if !ok {
		return grpcInvalidArgument(errReason)
	}
	items := []*securityposturepb.Posture{
		gcpStage4SecurityPosturePosture(orgID, location, "posture-1", "0000000a", "ACTIVE"),
		gcpStage4SecurityPosturePosture(orgID, location, "posture-draft", "0000000b", "DRAFT"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securityposturepb.ListPosturesResponse{
		Postures:      items[start:end],
		NextPageToken: next,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCSecurityPostureListPostureRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.ListPostureRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, postureID, ok := parseGCPSecurityPosturePostureName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	pageSize, start, errReason, ok := gcpStage4SecurityPostureParsePage(req.GetPageSize(), req.GetPageToken())
	if !ok {
		return grpcInvalidArgument(errReason)
	}
	state := gcpSecurityPostureStateForID(postureID)
	items := []*securityposturepb.Posture{
		gcpStage4SecurityPosturePosture(orgID, location, postureID, "00000009", state),
		gcpStage4SecurityPosturePosture(orgID, location, postureID, "0000000a", state),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securityposturepb.ListPostureRevisionsResponse{
		Revisions:     items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecurityPostureGetPosture(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.GetPostureRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, postureID, ok := parseGCPSecurityPosturePostureName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	revisionID := strings.TrimSpace(req.GetRevisionId())
	if revisionID == "" {
		revisionID = "0000000a"
	}
	return grpcProtoSuccess(gcpStage4SecurityPosturePosture(orgID, location, postureID, revisionID, gcpSecurityPostureStateForID(postureID)))
}

func gcpStage4GRPCSecurityPostureCreatePosture(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.CreatePostureRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPSecurityPostureParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	postureID := strings.TrimSpace(req.GetPostureId())
	if postureID == "" {
		return grpcInvalidArgument("posture_id-required")
	}
	posture := req.GetPosture()
	if posture == nil {
		return grpcInvalidArgument("posture-required")
	}
	if len(posture.GetPolicySets()) == 0 {
		return grpcInvalidArgument("posture.policy_sets-required")
	}
	expectedName := gcpSecurityPosturePostureName(orgID, location, postureID)
	if got := strings.TrimSpace(posture.GetName()); got != "" && got != expectedName {
		return grpcInvalidArgument("posture.name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "createPosture."+postureID, expectedName, "create", false))
}

func gcpStage4GRPCSecurityPostureUpdatePosture(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.UpdatePostureRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if strings.TrimSpace(req.GetRevisionId()) == "" {
		return grpcInvalidArgument("revision_id-required")
	}
	paths := req.GetUpdateMask().GetPaths()
	if !gcpStage4SecurityPostureAllowedMask(paths, map[string]struct{}{
		"state":       {},
		"description": {},
		"policySets":  {},
		"policy_sets": {},
		"annotations": {},
		"etag":        {},
	}) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	posture := req.GetPosture()
	if posture == nil {
		return grpcInvalidArgument("posture-required")
	}
	orgID, location, postureID, ok := parseGCPSecurityPosturePostureName(posture.GetName())
	if !ok {
		return grpcInvalidArgument("posture.name-required")
	}
	if gcpStage4SecurityPostureMaskHas(paths, "policySets", "policy_sets") && len(posture.GetPolicySets()) == 0 {
		return grpcInvalidArgument("posture.policy_sets-required")
	}
	if gcpStage4SecurityPostureMaskHas(paths, "state") && gcpStage4SecurityPostureMaskHas(paths, "description", "policySets", "policy_sets", "annotations") {
		return grpcInvalidArgument("state-mask-conflict")
	}
	if etag := strings.TrimSpace(posture.GetEtag()); etag != "" && etag != gcpSecurityPostureEtag(postureID) {
		return grpcAborted("etag-mismatch")
	}
	if gcpStage4SecurityPostureMaskHas(paths, "state") {
		requested := gcpStage4SecurityPostureStateString(posture.GetState())
		if requested == "" {
			return grpcInvalidArgument("posture.state-required")
		}
		if !gcpSecurityPostureStateTransitionAllowed(gcpSecurityPostureStateForID(postureID), requested) {
			return grpcFailedPrecondition("state-transition-not-allowed")
		}
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "updatePosture."+postureID+"."+strings.TrimSpace(req.GetRevisionId()), posture.GetName(), "update", false))
}

func gcpStage4GRPCSecurityPostureDeletePosture(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.DeletePostureRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, postureID, ok := parseGCPSecurityPosturePostureName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSecurityPostureEtag(postureID) {
		return grpcAborted("etag-mismatch")
	}
	if strings.Contains(strings.ToLower(postureID), "active") {
		return grpcFailedPrecondition("posture-has-active-deployments")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "deletePosture."+postureID, req.GetName(), "delete", false))
}

func gcpStage4GRPCSecurityPostureExtractPosture(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.ExtractPostureRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPSecurityPostureParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	postureID := strings.TrimSpace(req.GetPostureId())
	if postureID == "" {
		return grpcInvalidArgument("posture_id-required")
	}
	if strings.TrimSpace(req.GetWorkload()) == "" {
		return grpcInvalidArgument("workload-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "extractPosture."+postureID, gcpSecurityPosturePostureName(orgID, location, postureID), "extract", false))
}

func gcpStage4GRPCSecurityPostureListPostureDeployments(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.ListPostureDeploymentsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPSecurityPostureParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	pageSize, start, errReason, ok := gcpStage4SecurityPostureParsePage(req.GetPageSize(), req.GetPageToken())
	if !ok {
		return grpcInvalidArgument(errReason)
	}
	items := []*securityposturepb.PostureDeployment{
		gcpStage4SecurityPostureDeployment(orgID, location, "deployment-1"),
		gcpStage4SecurityPostureDeployment(orgID, location, "deployment-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securityposturepb.ListPostureDeploymentsResponse{
		PostureDeployments: items[start:end],
		NextPageToken:      next,
		Unreachable:        []string{},
	})
}

func gcpStage4GRPCSecurityPostureGetPostureDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.GetPostureDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, deploymentID, ok := parseGCPSecurityPostureDeploymentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureDeployment(orgID, location, deploymentID))
}

func gcpStage4GRPCSecurityPostureCreatePostureDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.CreatePostureDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPSecurityPostureParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	deploymentID := strings.TrimSpace(req.GetPostureDeploymentId())
	if deploymentID == "" {
		return grpcInvalidArgument("posture_deployment_id-required")
	}
	deployment := req.GetPostureDeployment()
	if deployment == nil {
		return grpcInvalidArgument("posture_deployment-required")
	}
	expectedName := gcpSecurityPostureDeploymentName(orgID, location, deploymentID)
	if got := strings.TrimSpace(deployment.GetName()); got != "" && got != expectedName {
		return grpcInvalidArgument("posture_deployment.name-mismatch")
	}
	if strings.TrimSpace(deployment.GetTargetResource()) == "" {
		return grpcInvalidArgument("posture_deployment.target_resource-required")
	}
	if strings.TrimSpace(deployment.GetPostureId()) == "" {
		return grpcInvalidArgument("posture_deployment.posture_id-required")
	}
	if strings.TrimSpace(deployment.GetPostureRevisionId()) == "" {
		return grpcInvalidArgument("posture_deployment.posture_revision_id-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "createPostureDeployment."+deploymentID, expectedName, "create", false))
}

func gcpStage4GRPCSecurityPostureUpdatePostureDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.UpdatePostureDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	paths := req.GetUpdateMask().GetPaths()
	if !gcpStage4SecurityPostureAllowedMask(paths, map[string]struct{}{
		"description":         {},
		"postureId":           {},
		"posture_id":          {},
		"postureRevisionId":   {},
		"posture_revision_id": {},
		"annotations":         {},
		"targetResource":      {},
		"target_resource":     {},
		"etag":                {},
	}) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	deployment := req.GetPostureDeployment()
	if deployment == nil {
		return grpcInvalidArgument("posture_deployment-required")
	}
	orgID, location, deploymentID, ok := parseGCPSecurityPostureDeploymentName(deployment.GetName())
	if !ok {
		return grpcInvalidArgument("posture_deployment.name-required")
	}
	if etag := strings.TrimSpace(deployment.GetEtag()); etag != "" && etag != gcpSecurityPostureEtag(deploymentID) {
		return grpcAborted("etag-mismatch")
	}
	if gcpStage4SecurityPostureMaskHas(paths, "targetResource", "target_resource") && strings.TrimSpace(deployment.GetTargetResource()) == "" {
		return grpcInvalidArgument("posture_deployment.target_resource-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "updatePostureDeployment."+deploymentID, deployment.GetName(), "update", false))
}

func gcpStage4GRPCSecurityPostureDeletePostureDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.DeletePostureDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, deploymentID, ok := parseGCPSecurityPostureDeploymentName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if etag := strings.TrimSpace(req.GetEtag()); etag != "" && etag != gcpSecurityPostureEtag(deploymentID) {
		return grpcAborted("etag-mismatch")
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureOperation(orgID, location, "deletePostureDeployment."+deploymentID, req.GetName(), "delete", false))
}

func gcpStage4GRPCSecurityPostureListPostureTemplates(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.ListPostureTemplatesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, ok := parseGCPSecurityPostureParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	pageSize, start, errReason, ok := gcpStage4SecurityPostureParsePage(req.GetPageSize(), req.GetPageToken())
	if !ok {
		return grpcInvalidArgument(errReason)
	}
	items := []*securityposturepb.PostureTemplate{
		gcpStage4SecurityPostureTemplate(orgID, location, "template-1", "00000001"),
		gcpStage4SecurityPostureTemplate(orgID, location, "template-2", "00000002"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securityposturepb.ListPostureTemplatesResponse{
		PostureTemplates: items[start:end],
		NextPageToken:    next,
	})
}

func gcpStage4GRPCSecurityPostureGetPostureTemplate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securityposturepb.GetPostureTemplateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	orgID, location, templateID, ok := parseGCPSecurityPostureTemplateName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	revisionID := strings.TrimSpace(req.GetRevisionId())
	if revisionID == "" {
		revisionID = "00000001"
	}
	return grpcProtoSuccess(gcpStage4SecurityPostureTemplate(orgID, location, templateID, revisionID))
}

func gcpStage4SecurityPosturePosture(orgID, location, postureID, revisionID, state string) *securityposturepb.Posture {
	resp := &securityposturepb.Posture{}
	if !gcpStage4SecurityPostureFixtureToProto(gcpSecurityPosturePosture(orgID, location, postureID, revisionID, state), resp) {
		return &securityposturepb.Posture{
			Name:        gcpSecurityPosturePostureName(orgID, location, postureID),
			State:       securityposturepb.Posture_ACTIVE,
			RevisionId:  revisionID,
			Description: "Fallback posture fixture",
			PolicySets: []*securityposturepb.PolicySet{
				{
					PolicySetId: "baseline",
					Policies: []*securityposturepb.Policy{
						{
							PolicyId: "sha-001",
							Constraint: &securityposturepb.Constraint{
								Implementation: &securityposturepb.Constraint_SecurityHealthAnalyticsModule{
									SecurityHealthAnalyticsModule: &securityposturepb.SecurityHealthAnalyticsModule{
										ModuleName:            "BIGQUERY_TABLE_CMEK_DISABLED",
										ModuleEnablementState: securityposturepb.EnablementState_ENABLED,
									},
								},
							},
						},
					},
				},
			},
			CreateTime: timestamppb.New(gcpStage4ReferenceTime),
			UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(15 * time.Minute)),
			Etag:       gcpSecurityPostureEtag(postureID),
		}
	}
	return resp
}

func gcpStage4SecurityPostureDeployment(orgID, location, deploymentID string) *securityposturepb.PostureDeployment {
	resp := &securityposturepb.PostureDeployment{}
	if !gcpStage4SecurityPostureFixtureToProto(gcpSecurityPostureDeployment(orgID, location, deploymentID), resp) {
		return &securityposturepb.PostureDeployment{
			Name:                     gcpSecurityPostureDeploymentName(orgID, location, deploymentID),
			TargetResource:           "projects/123456789",
			State:                    securityposturepb.PostureDeployment_ACTIVE,
			PostureId:                gcpSecurityPosturePostureName(orgID, location, "posture-1"),
			PostureRevisionId:        "0000000a",
			CreateTime:               timestamppb.New(gcpStage4ReferenceTime),
			UpdateTime:               timestamppb.New(gcpStage4ReferenceTime.Add(20 * time.Minute)),
			Etag:                     gcpSecurityPostureEtag(deploymentID),
			DesiredPostureId:         gcpSecurityPosturePostureName(orgID, location, "posture-1"),
			DesiredPostureRevisionId: "0000000a",
		}
	}
	return resp
}

func gcpStage4SecurityPostureTemplate(orgID, location, templateID, revisionID string) *securityposturepb.PostureTemplate {
	resp := &securityposturepb.PostureTemplate{}
	if !gcpStage4SecurityPostureFixtureToProto(gcpSecurityPostureTemplate(orgID, location, templateID, revisionID), resp) {
		return &securityposturepb.PostureTemplate{
			Name:        gcpSecurityPostureTemplateName(orgID, location, templateID),
			RevisionId:  revisionID,
			Description: "Fallback posture template fixture",
			State:       securityposturepb.PostureTemplate_ACTIVE,
		}
	}
	return resp
}

func gcpStage4SecurityPostureOperation(orgID, location, operationID, target, verb string, done bool) *longrunningpb.Operation {
	resp := &longrunningpb.Operation{}
	if !gcpStage4SecurityPostureFixtureToProto(gcpSecurityPostureOperation(orgID, location, operationID, target, verb, done), resp) {
		return &longrunningpb.Operation{
			Name: gcpSecurityPostureOperationName(orgID, location, operationID),
			Done: done,
		}
	}
	return resp
}

func gcpStage4SecurityPostureFixtureToProto(src map[string]any, dst proto.Message) bool {
	raw, err := json.Marshal(src)
	if err != nil {
		return false
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, dst); err != nil {
		return false
	}
	return true
}

func gcpStage4SecurityPostureParsePage(pageSize int32, pageToken string) (int, int, string, bool) {
	if pageSize < 0 || pageSize > 1000 {
		return 0, 0, "page_size-invalid", false
	}
	start, valid := parseGCPStage4PageToken(pageToken)
	if !valid {
		return 0, 0, "page_token-invalid", false
	}
	return int(pageSize), start, "", true
}

func gcpStage4SecurityPostureAllowedMask(paths []string, allowed map[string]struct{}) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			return false
		}
		if _, ok := allowed[trimmed]; !ok {
			return false
		}
	}
	return true
}

func gcpStage4SecurityPostureMaskHas(paths []string, candidates ...string) bool {
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		for _, candidate := range candidates {
			if trimmed == candidate {
				return true
			}
		}
	}
	return false
}

func gcpStage4SecurityPostureStateString(state securityposturepb.Posture_State) string {
	switch state {
	case securityposturepb.Posture_ACTIVE:
		return "ACTIVE"
	case securityposturepb.Posture_DRAFT:
		return "DRAFT"
	case securityposturepb.Posture_DEPRECATED:
		return "DEPRECATED"
	default:
		return ""
	}
}

func gcpStage4GRPCSecureSourceManagerListRepositories(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securesourcemanagerpb.ListRepositoriesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4SecureSourceManagerLocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securesourcemanagerpb.Repository{
		gcpStage4SecureSourceManagerRepository(project, location, "repository-1"),
		gcpStage4SecureSourceManagerRepository(project, location, "repository-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securesourcemanagerpb.ListRepositoriesResponse{
		Repositories:  items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecureSourceManagerGetRepository(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securesourcemanagerpb.GetRepositoryRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, repositoryID, ok := parseGCPStage4SecureSourceManagerRepositoryName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecureSourceManagerRepository(project, location, repositoryID))
}

func gcpStage4GRPCSecureSourceManagerGetIAMPolicyRepo(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &iampb.GetIamPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, ok := parseGCPStage4SecureSourceManagerRepositoryName(req.GetResource()); !ok {
		return grpcInvalidArgument("resource-required")
	}
	return grpcProtoSuccess(&iampb.Policy{
		Version: 1,
		Bindings: []*iampb.Binding{
			{
				Role:    "roles/securesourcemanager.reader",
				Members: []string{"user:alice@example.com"},
			},
		},
		Etag: []byte("c3RhY2t5YXJk"),
	})
}

func gcpStage4GRPCSecureSourceManagerListPullRequests(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securesourcemanagerpb.ListPullRequestsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, repositoryID, ok := parseGCPStage4SecureSourceManagerRepositoryName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*securesourcemanagerpb.PullRequest{
		gcpStage4SecureSourceManagerPullRequest(project, location, repositoryID, "pull-request-1"),
		gcpStage4SecureSourceManagerPullRequest(project, location, repositoryID, "pull-request-closed"),
		gcpStage4SecureSourceManagerPullRequest(project, location, repositoryID, "pull-request-merged"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&securesourcemanagerpb.ListPullRequestsResponse{
		PullRequests:  items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCSecureSourceManagerClosePullRequest(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &securesourcemanagerpb.ClosePullRequestRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, repositoryID, pullRequestID, ok := parseGCPStage4SecureSourceManagerPullRequestName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpSecureSourceManagerPullRequestState(pullRequestID) != "OPEN" {
		return grpcFailedPrecondition("pull_request-open-required")
	}
	return grpcProtoSuccess(gcpStage4SecureSourceManagerOperation(project, location, "closePullRequest."+pullRequestID, gcpSecureSourceManagerPullRequestName(project, location, repositoryID, pullRequestID), false))
}

func gcpStage4SecureSourceManagerRepository(project, location, repositoryID string) *securesourcemanagerpb.Repository {
	name := gcpSecureSourceManagerRepositoryName(project, location, repositoryID)
	return &securesourcemanagerpb.Repository{
		Name:        name,
		Uid:         "ssm-repo-" + repositoryID,
		Description: "Stackyard repository " + repositoryID,
		Instance:    gcpSecureSourceManagerInstanceName(project, location, "instance-1"),
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(90 * time.Minute)),
		Etag:        gcpSecureSourceManagerETag(repositoryID),
		Uris: &securesourcemanagerpb.Repository_URIs{
			Html:     fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", project, repositoryID),
			GitHttps: fmt.Sprintf("https://source.developers.google.com/p/%s/r/%s", project, repositoryID),
			Api:      "https://securesourcemanager.googleapis.com/v1/" + name,
		},
	}
}

func gcpStage4SecureSourceManagerPullRequest(project, location, repositoryID, pullRequestID string) *securesourcemanagerpb.PullRequest {
	state := gcpStage4SecureSourceManagerPullRequestState(pullRequestID)
	pr := &securesourcemanagerpb.PullRequest{
		Name:  gcpSecureSourceManagerPullRequestName(project, location, repositoryID, pullRequestID),
		Title: "Stackyard pull request " + pullRequestID,
		Body:  "Generated pull request fixture",
		Base: &securesourcemanagerpb.PullRequest_Branch{
			Ref: "refs/heads/main",
			Sha: "c0ffee111",
		},
		Head: &securesourcemanagerpb.PullRequest_Branch{
			Ref: "refs/heads/feature/" + pullRequestID,
			Sha: "abc123def",
		},
		State:      state,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(15 * time.Minute)),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(140 * time.Minute)),
	}
	if state != securesourcemanagerpb.PullRequest_OPEN {
		pr.CloseTime = timestamppb.New(gcpStage4ReferenceTime.Add(150 * time.Minute))
	}
	return pr
}

func gcpStage4SecureSourceManagerPullRequestState(pullRequestID string) securesourcemanagerpb.PullRequest_State {
	switch gcpSecureSourceManagerPullRequestState(pullRequestID) {
	case "MERGED":
		return securesourcemanagerpb.PullRequest_MERGED
	case "CLOSED":
		return securesourcemanagerpb.PullRequest_CLOSED
	default:
		return securesourcemanagerpb.PullRequest_OPEN
	}
}

func gcpStage4SecureSourceManagerOperation(project, location, operationID, _ string, done bool) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: done,
	}
}

func gcpStage4SecretManagerSecret(project, location, secretID string) *secretmanagerpb.Secret {
	return &secretmanagerpb.Secret{
		Name: gcpStage4SecretManagerSecretResourceName(project, location, secretID),
		Replication: &secretmanagerpb.Replication{
			Replication: &secretmanagerpb.Replication_Automatic_{
				Automatic: &secretmanagerpb.Replication_Automatic{},
			},
		},
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		Labels: map[string]string{
			"env": "staged",
		},
		Rotation: &secretmanagerpb.Rotation{
			NextRotationTime: timestamppb.New(gcpStage4ReferenceTime.Add(24 * time.Hour)),
			RotationPeriod:   durationpb.New(24 * time.Hour),
		},
		Etag: "etag-" + secretID,
	}
}

func gcpStage4SecretManagerSecretFromRequest(project, location, secretID string, req *secretmanagerpb.Secret) *secretmanagerpb.Secret {
	secret := gcpStage4SecretManagerSecret(project, location, secretID)
	if req == nil {
		return secret
	}
	if req.GetReplication() != nil {
		secret.Replication = req.GetReplication()
	}
	if len(req.GetLabels()) > 0 {
		secret.Labels = req.GetLabels()
	}
	if req.GetRotation() != nil {
		secret.Rotation = req.GetRotation()
	}
	if etag := strings.TrimSpace(req.GetEtag()); etag != "" {
		secret.Etag = etag
	}
	return secret
}

func gcpStage4SecretManagerVersion(project, location, secretID, versionID string) *secretmanagerpb.SecretVersion {
	versionID = normalizeGCPSecretManagerVersionID(versionID)
	state := gcpStage4SecretManagerVersionState(versionID)
	version := &secretmanagerpb.SecretVersion{
		Name:       gcpStage4SecretManagerVersionResourceName(project, location, secretID, versionID),
		State:      state,
		Etag:       "etag-version-" + versionID,
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(30 * time.Second)),
	}
	if state == secretmanagerpb.SecretVersion_DESTROYED {
		version.DestroyTime = timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour))
	}
	return version
}

func gcpStage4SecretManagerVersionState(versionID string) secretmanagerpb.SecretVersion_State {
	switch gcpSecretManagerVersionState(normalizeGCPSecretManagerVersionID(versionID)) {
	case "DISABLED":
		return secretmanagerpb.SecretVersion_DISABLED
	case "DESTROYED":
		return secretmanagerpb.SecretVersion_DESTROYED
	default:
		return secretmanagerpb.SecretVersion_ENABLED
	}
}

func gcpStage4SecretManagerPolicy(secretID string, in *iampb.Policy) *iampb.Policy {
	if in == nil {
		return &iampb.Policy{
			Version: 1,
			Etag:    []byte("policy-etag-" + secretID),
			Bindings: []*iampb.Binding{
				{
					Role:    "roles/secretmanager.secretAccessor",
					Members: []string{"user:stackyard@example.com"},
				},
			},
		}
	}
	cloned, ok := proto.Clone(in).(*iampb.Policy)
	if !ok || cloned == nil {
		return in
	}
	if cloned.GetVersion() == 0 {
		cloned.Version = 1
	}
	if len(cloned.GetEtag()) == 0 {
		cloned.Etag = []byte("policy-etag-" + secretID)
	}
	return cloned
}

func gcpStage4SecretManagerSecretResourceName(project, location, secretID string) string {
	if strings.TrimSpace(location) != "" {
		return fmt.Sprintf("projects/%s/locations/%s/secrets/%s", project, location, secretID)
	}
	return fmt.Sprintf("projects/%s/secrets/%s", project, secretID)
}

func gcpStage4SecretManagerVersionResourceName(project, location, secretID, versionID string) string {
	if strings.TrimSpace(location) != "" {
		return fmt.Sprintf("projects/%s/locations/%s/secrets/%s/versions/%s", project, location, secretID, versionID)
	}
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", project, secretID, versionID)
}

func gcpStage4GRPCSecurityPrivateCAListCaPools(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.ListCaPoolsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4SecurityPrivateCALocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*privatecapb.CaPool{
		gcpStage4SecurityPrivateCACaPool(project, location, "pool-1"),
		gcpStage4SecurityPrivateCACaPool(project, location, "pool-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&privatecapb.ListCaPoolsResponse{
		CaPools:       items[start:end],
		NextPageToken: next,
		Unreachable:   nil,
	})
}

func gcpStage4GRPCSecurityPrivateCAGetCaPool(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.GetCaPoolRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, ok := parseGCPStage4SecurityPrivateCACaPoolName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPrivateCACaPool(project, location, caPoolID))
}

func gcpStage4GRPCSecurityPrivateCACreateCaPool(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.CreateCaPoolRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4SecurityPrivateCALocationParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetCaPoolId()) == "" {
		return grpcInvalidArgument("ca_pool_id-required")
	}
	if !isGCPSecurityPrivateCAID(req.GetCaPoolId()) {
		return grpcInvalidArgument("ca_pool_id-invalid")
	}
	caPool := req.GetCaPool()
	if caPool == nil {
		return grpcInvalidArgument("ca_pool-required")
	}
	if name := strings.TrimSpace(caPool.GetName()); name != "" {
		bodyProject, bodyLocation, bodyCaPoolID, nameOK := parseGCPStage4SecurityPrivateCACaPoolName(name)
		if !nameOK || bodyProject != project || bodyLocation != location || bodyCaPoolID != req.GetCaPoolId() {
			return grpcInvalidArgument("ca_pool.name-mismatch")
		}
	}
	return grpcProtoSuccess(gcpStage4SecurityPrivateCAOperation(project, location, "create-ca-pool-"+req.GetCaPoolId()))
}

func gcpStage4GRPCSecurityPrivateCAListCertificateAuthorities(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.ListCertificateAuthoritiesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, ok := parseGCPStage4SecurityPrivateCACaPoolName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*privatecapb.CertificateAuthority{
		gcpStage4SecurityPrivateCACertificateAuthority(project, location, caPoolID, "ca-1"),
		gcpStage4SecurityPrivateCACertificateAuthority(project, location, caPoolID, "ca-disabled"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&privatecapb.ListCertificateAuthoritiesResponse{
		CertificateAuthorities: items[start:end],
		NextPageToken:          next,
	})
}

func gcpStage4GRPCSecurityPrivateCAGetCertificateAuthority(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.GetCertificateAuthorityRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, caID, ok := parseGCPStage4SecurityPrivateCACertificateAuthorityName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPrivateCACertificateAuthority(project, location, caPoolID, caID))
}

func gcpStage4GRPCSecurityPrivateCACreateCertificateAuthority(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.CreateCertificateAuthorityRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, ok := parseGCPStage4SecurityPrivateCACaPoolName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetCertificateAuthorityId()) == "" {
		return grpcInvalidArgument("certificate_authority_id-required")
	}
	if !isGCPSecurityPrivateCAID(req.GetCertificateAuthorityId()) {
		return grpcInvalidArgument("certificate_authority_id-invalid")
	}
	certificateAuthority := req.GetCertificateAuthority()
	if certificateAuthority == nil {
		return grpcInvalidArgument("certificate_authority-required")
	}
	if certificateAuthority.GetType() == privatecapb.CertificateAuthority_TYPE_UNSPECIFIED {
		return grpcInvalidArgument("certificate_authority.type-required")
	}
	if certificateAuthority.GetConfig() == nil {
		return grpcInvalidArgument("certificate_authority.config-required")
	}
	if name := strings.TrimSpace(certificateAuthority.GetName()); name != "" {
		bodyProject, bodyLocation, bodyCaPoolID, bodyCAID, nameOK := parseGCPStage4SecurityPrivateCACertificateAuthorityName(name)
		if !nameOK || bodyProject != project || bodyLocation != location || bodyCaPoolID != caPoolID || bodyCAID != req.GetCertificateAuthorityId() {
			return grpcInvalidArgument("certificate_authority.name-mismatch")
		}
	}
	return grpcProtoSuccess(gcpStage4SecurityPrivateCAOperation(project, location, "create-certificate-authority-"+req.GetCertificateAuthorityId()))
}

func gcpStage4GRPCSecurityPrivateCAListCertificates(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.ListCertificatesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, ok := parseGCPStage4SecurityPrivateCACaPoolName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*privatecapb.Certificate{
		gcpStage4SecurityPrivateCACertificate(project, location, caPoolID, "cert-1"),
		gcpStage4SecurityPrivateCACertificate(project, location, caPoolID, "cert-revoked"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&privatecapb.ListCertificatesResponse{
		Certificates:  items[start:end],
		NextPageToken: next,
		Unreachable:   nil,
	})
}

func gcpStage4GRPCSecurityPrivateCAGetCertificate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.GetCertificateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, certificateID, ok := parseGCPStage4SecurityPrivateCACertificateName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4SecurityPrivateCACertificate(project, location, caPoolID, certificateID))
}

func gcpStage4GRPCSecurityPrivateCACreateCertificate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.CreateCertificateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, ok := parseGCPStage4SecurityPrivateCACaPoolName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetCertificateId()) == "" {
		return grpcInvalidArgument("certificate_id-required")
	}
	if !isGCPSecurityPrivateCAID(req.GetCertificateId()) {
		return grpcInvalidArgument("certificate_id-invalid")
	}
	certificate := req.GetCertificate()
	if certificate == nil {
		return grpcInvalidArgument("certificate-required")
	}
	if certificate.GetLifetime() == nil {
		return grpcInvalidArgument("certificate.lifetime-required")
	}
	if name := strings.TrimSpace(certificate.GetName()); name != "" {
		bodyProject, bodyLocation, bodyCaPoolID, bodyCertificateID, nameOK := parseGCPStage4SecurityPrivateCACertificateName(name)
		if !nameOK || bodyProject != project || bodyLocation != location || bodyCaPoolID != caPoolID || bodyCertificateID != req.GetCertificateId() {
			return grpcInvalidArgument("certificate.name-mismatch")
		}
	}
	return grpcProtoSuccess(gcpStage4SecurityPrivateCACertificate(project, location, caPoolID, req.GetCertificateId()))
}

func gcpStage4GRPCSecurityPrivateCARevokeCertificate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &privatecapb.RevokeCertificateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, caPoolID, certificateID, ok := parseGCPStage4SecurityPrivateCACertificateName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetReason() == privatecapb.RevocationReason_REVOCATION_REASON_UNSPECIFIED {
		return grpcInvalidArgument("reason-required")
	}
	if gcpSecurityPrivateCACertificateState(certificateID) == "REVOKED" {
		return grpcFailedPrecondition("certificate-already-revoked")
	}
	certificate := gcpStage4SecurityPrivateCACertificate(project, location, caPoolID, certificateID)
	certificate.RevocationDetails = &privatecapb.Certificate_RevocationDetails{
		RevocationState: privatecapb.RevocationReason_KEY_COMPROMISE,
		RevocationTime:  timestamppb.New(gcpStage4ReferenceTime.Add(3 * time.Hour)),
	}
	return grpcProtoSuccess(certificate)
}

func gcpStage4GRPCSecurityPublicCACreateExternalAccountKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &publiccapb.CreateExternalAccountKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4LocationName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if location != "global" {
		return grpcFailedPrecondition("location-global-required")
	}
	externalAccountKey := req.GetExternalAccountKey()
	if externalAccountKey == nil {
		return grpcInvalidArgument("external_account_key-required")
	}
	keyID := strings.TrimSpace(externalAccountKey.GetKeyId())
	if keyID == "" {
		keyID = "eak-1"
	}
	if name := strings.TrimSpace(externalAccountKey.GetName()); name != "" {
		nameProject, nameLocation, parsedKeyID, parsed := parseGCPStage4SecurityPublicCAExternalAccountKeyName(name)
		if !parsed {
			return grpcInvalidArgument("external_account_key.name-invalid")
		}
		if nameProject != project || nameLocation != location {
			return grpcInvalidArgument("external_account_key.name-parent-mismatch")
		}
		keyID = parsedKeyID
	}
	return grpcProtoSuccess(&publiccapb.ExternalAccountKey{
		Name:      fmt.Sprintf("projects/%s/locations/%s/externalAccountKeys/%s", project, location, keyID),
		KeyId:     keyID,
		B64MacKey: []byte("stackyard-mac-key"),
	})
}

func parseGCPStage4SecurityPrivateCALocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4SecurityPublicCAExternalAccountKeyName(name string) (project, location, keyID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "externalAccountKeys" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	keyID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || keyID == "" {
		return "", "", "", false
	}
	return project, location, keyID, true
}

func parseGCPStage4SecurityPrivateCACaPoolName(name string) (project, location, caPoolID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "caPools" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	caPoolID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || caPoolID == "" {
		return "", "", "", false
	}
	return project, location, caPoolID, true
}

func parseGCPStage4SecurityPrivateCACertificateAuthorityName(name string) (project, location, caPoolID, caID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "caPools" || parts[6] != "certificateAuthorities" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	caPoolID = strings.TrimSpace(parts[5])
	caID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || caPoolID == "" || caID == "" {
		return "", "", "", "", false
	}
	return project, location, caPoolID, caID, true
}

func parseGCPStage4SecurityPrivateCACertificateName(name string) (project, location, caPoolID, certificateID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "caPools" || parts[6] != "certificates" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	caPoolID = strings.TrimSpace(parts[5])
	certificateID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || caPoolID == "" || certificateID == "" {
		return "", "", "", "", false
	}
	return project, location, caPoolID, certificateID, true
}

func gcpStage4SecurityPrivateCACaPool(project, location, caPoolID string) *privatecapb.CaPool {
	return &privatecapb.CaPool{
		Name: gcpSecurityPrivateCACaPoolName(project, location, caPoolID),
		Tier: privatecapb.CaPool_ENTERPRISE,
		PublishingOptions: &privatecapb.CaPool_PublishingOptions{
			PublishCaCert: true,
			PublishCrl:    true,
		},
		Labels: map[string]string{
			"env": "staged",
		},
	}
}

func gcpStage4SecurityPrivateCACertificateAuthority(project, location, caPoolID, caID string) *privatecapb.CertificateAuthority {
	return &privatecapb.CertificateAuthority{
		Name:     gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, caID),
		Type:     privatecapb.CertificateAuthority_SELF_SIGNED,
		Tier:     privatecapb.CaPool_ENTERPRISE,
		State:    gcpStage4SecurityPrivateCACertificateAuthorityState(caID),
		Lifetime: durationpb.New(365 * 24 * time.Hour),
		KeySpec: &privatecapb.CertificateAuthority_KeyVersionSpec{
			KeyVersion: &privatecapb.CertificateAuthority_KeyVersionSpec_Algorithm{
				Algorithm: privatecapb.CertificateAuthority_RSA_PKCS1_4096_SHA256,
			},
		},
		Config: &privatecapb.CertificateConfig{
			SubjectConfig: &privatecapb.CertificateConfig_SubjectConfig{
				Subject: &privatecapb.Subject{
					CommonName: "Stackyard " + caID,
				},
			},
		},
		PemCaCertificates: []string{
			"-----BEGIN CERTIFICATE-----\nSTACKYARD-PRIVATECA\n-----END CERTIFICATE-----",
		},
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Minute)),
		Labels: map[string]string{
			"env": "staged",
		},
	}
}

func gcpStage4SecurityPrivateCACertificate(project, location, caPoolID, certificateID string) *privatecapb.Certificate {
	certificate := &privatecapb.Certificate{
		Name:                       gcpSecurityPrivateCACertificateName(project, location, caPoolID, certificateID),
		IssuerCertificateAuthority: gcpSecurityPrivateCACertificateAuthorityName(project, location, caPoolID, "ca-1"),
		Lifetime:                   durationpb.New(24 * time.Hour),
		PemCertificate:             "-----BEGIN CERTIFICATE-----\nSTACKYARD-CERT\n-----END CERTIFICATE-----",
		PemCertificateChain: []string{
			"-----BEGIN CERTIFICATE-----\nSTACKYARD-CERT-CHAIN\n-----END CERTIFICATE-----",
		},
		CreateTime: timestamppb.New(gcpStage4ReferenceTime.Add(1 * time.Hour)),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Hour)),
		Labels: map[string]string{
			"env": "staged",
		},
	}
	if gcpSecurityPrivateCACertificateState(certificateID) == "REVOKED" {
		certificate.RevocationDetails = &privatecapb.Certificate_RevocationDetails{
			RevocationState: privatecapb.RevocationReason_KEY_COMPROMISE,
			RevocationTime:  timestamppb.New(gcpStage4ReferenceTime.Add(3 * time.Hour)),
		}
	}
	return certificate
}

func gcpStage4SecurityPrivateCAOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: gcpSecurityPrivateCAOperationName(project, location, operationID),
		Done: true,
	}
}

func gcpStage4SecurityPrivateCACertificateAuthorityState(caID string) privatecapb.CertificateAuthority_State {
	switch gcpSecurityPrivateCACertificateAuthorityState(caID) {
	case "DISABLED":
		return privatecapb.CertificateAuthority_DISABLED
	case "AWAITING_USER_ACTIVATION":
		return privatecapb.CertificateAuthority_AWAITING_USER_ACTIVATION
	case "DELETED":
		return privatecapb.CertificateAuthority_DELETED
	default:
		return privatecapb.CertificateAuthority_ENABLED
	}
}

func gcpStage4GRPCRecommenderListInsights(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.ListInsightsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, insightType, ok := parseGCPRecommenderInsightTypeParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recommenderpb.Insight{
		gcpStage4RecommenderInsight(scopeType, scopeID, location, insightType, "insight-1", recommenderpb.InsightStateInfo_ACTIVE),
		gcpStage4RecommenderInsight(scopeType, scopeID, location, insightType, "insight-2", recommenderpb.InsightStateInfo_ACTIVE),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recommenderpb.ListInsightsResponse{
		Insights:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRecommenderGetInsight(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.GetInsightRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, insightType, insightID, _, ok := parseGCPRecommenderInsightName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := recommenderpb.InsightStateInfo_ACTIVE
	if strings.Contains(strings.ToLower(insightID), "accepted") {
		state = recommenderpb.InsightStateInfo_ACCEPTED
	}
	return grpcProtoSuccess(gcpStage4RecommenderInsight(scopeType, scopeID, location, insightType, insightID, state))
}

func gcpStage4GRPCRecommenderMarkInsightAccepted(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.MarkInsightAcceptedRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, insightType, insightID, _, ok := parseGCPRecommenderInsightName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(req.GetEtag()) == "" {
		return grpcInvalidArgument("etag-required")
	}
	if req.GetEtag() != gcpRecommenderInsightETag(insightID) {
		return grpcFailedPrecondition("etag-mismatch")
	}
	if strings.Contains(strings.ToLower(insightID), "accepted") {
		return grpcFailedPrecondition("insight-already-accepted")
	}
	insight := gcpStage4RecommenderInsight(scopeType, scopeID, location, insightType, insightID, recommenderpb.InsightStateInfo_ACCEPTED)
	insight.Etag = gcpRecommenderInsightETag(insightID) + "-accepted"
	if len(req.GetStateMetadata()) > 0 {
		insight.StateInfo.StateMetadata = req.GetStateMetadata()
	}
	return grpcProtoSuccess(insight)
}

func gcpStage4GRPCRecommenderListRecommendations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.ListRecommendationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, recommenderID, ok := parseGCPRecommenderParentName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recommenderpb.Recommendation{
		gcpStage4RecommenderRecommendation(scopeType, scopeID, location, recommenderID, "recommendation-1", recommenderpb.RecommendationStateInfo_ACTIVE),
		gcpStage4RecommenderRecommendation(scopeType, scopeID, location, recommenderID, "recommendation-2", recommenderpb.RecommendationStateInfo_ACTIVE),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recommenderpb.ListRecommendationsResponse{
		Recommendations: items[start:end],
		NextPageToken:   next,
	})
}

func gcpStage4GRPCRecommenderGetRecommendation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.GetRecommendationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, recommenderID, recommendationID, _, ok := parseGCPRecommenderRecommendationName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(
		gcpStage4RecommenderRecommendation(
			scopeType,
			scopeID,
			location,
			recommenderID,
			recommendationID,
			gcpStage4RecommenderRecommendationStateEnum(gcpRecommenderRecommendationStateForID(recommendationID)),
		),
	)
}

func gcpStage4GRPCRecommenderMarkRecommendationDismissed(grpcReqBody []byte) ([]byte, string, string, bool) {
	return gcpStage4GRPCRecommenderRecommendationAction(grpcReqBody, "markDismissed")
}

func gcpStage4GRPCRecommenderMarkRecommendationClaimed(grpcReqBody []byte) ([]byte, string, string, bool) {
	return gcpStage4GRPCRecommenderRecommendationAction(grpcReqBody, "markClaimed")
}

func gcpStage4GRPCRecommenderMarkRecommendationSucceeded(grpcReqBody []byte) ([]byte, string, string, bool) {
	return gcpStage4GRPCRecommenderRecommendationAction(grpcReqBody, "markSucceeded")
}

func gcpStage4GRPCRecommenderMarkRecommendationFailed(grpcReqBody []byte) ([]byte, string, string, bool) {
	return gcpStage4GRPCRecommenderRecommendationAction(grpcReqBody, "markFailed")
}

func gcpStage4GRPCRecommenderRecommendationAction(grpcReqBody []byte, action string) ([]byte, string, string, bool) {
	var (
		name          string
		etag          string
		stateMetadata map[string]string
	)
	switch action {
	case "markDismissed":
		req := &recommenderpb.MarkRecommendationDismissedRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name, etag = req.GetName(), req.GetEtag()
		stateMetadata = nil
	case "markClaimed":
		req := &recommenderpb.MarkRecommendationClaimedRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name, etag, stateMetadata = req.GetName(), req.GetEtag(), req.GetStateMetadata()
	case "markSucceeded":
		req := &recommenderpb.MarkRecommendationSucceededRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name, etag, stateMetadata = req.GetName(), req.GetEtag(), req.GetStateMetadata()
	case "markFailed":
		req := &recommenderpb.MarkRecommendationFailedRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name, etag, stateMetadata = req.GetName(), req.GetEtag(), req.GetStateMetadata()
	default:
		return grpcInvalidArgument("action-invalid")
	}

	scopeType, scopeID, location, recommenderID, recommendationID, _, ok := parseGCPRecommenderRecommendationName(name)
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(etag) == "" {
		return grpcInvalidArgument("etag-required")
	}
	if etag != gcpRecommenderRecommendationETag(recommendationID) {
		return grpcFailedPrecondition("etag-mismatch")
	}
	currentState := gcpRecommenderRecommendationStateForID(recommendationID)
	if !gcpRecommenderRecommendationTransitionAllowed(currentState, action) {
		return grpcFailedPrecondition("state-transition-not-allowed")
	}
	nextState, ok := gcpRecommenderRecommendationNextState(action)
	if !ok {
		return grpcInvalidArgument("action-invalid")
	}

	recommendation := gcpStage4RecommenderRecommendation(scopeType, scopeID, location, recommenderID, recommendationID, gcpStage4RecommenderRecommendationStateEnum(nextState))
	recommendation.Etag = gcpRecommenderRecommendationETag(recommendationID) + "-" + strings.TrimPrefix(strings.ToLower(action), "mark")
	if len(stateMetadata) > 0 {
		recommendation.StateInfo.StateMetadata = stateMetadata
	}
	return grpcProtoSuccess(recommendation)
}

func gcpStage4GRPCRecommenderGetRecommenderConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.GetRecommenderConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, recommenderID, ok := parseGCPRecommenderConfigName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RecommenderConfig(scopeType, scopeID, location, recommenderID))
}

func gcpStage4GRPCRecommenderUpdateRecommenderConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.UpdateRecommenderConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetRecommenderConfig() == nil {
		return grpcInvalidArgument("recommender_config-required")
	}
	scopeType, scopeID, location, recommenderID, ok := parseGCPRecommenderConfigName(req.GetRecommenderConfig().GetName())
	if !ok {
		return grpcInvalidArgument("recommender_config.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpRecommenderValidUpdateMaskPaths(req.GetUpdateMask().GetPaths()) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	config := gcpStage4RecommenderConfig(scopeType, scopeID, location, recommenderID)
	config.Etag = gcpRecommenderConfigETag(recommenderID) + "-updated"
	if displayName := strings.TrimSpace(req.GetRecommenderConfig().GetDisplayName()); displayName != "" {
		config.DisplayName = displayName
	}
	if len(req.GetRecommenderConfig().GetAnnotations()) > 0 {
		config.Annotations = req.GetRecommenderConfig().GetAnnotations()
	}
	if req.GetRecommenderConfig().GetRecommenderGenerationConfig() != nil && req.GetRecommenderConfig().GetRecommenderGenerationConfig().GetParams() != nil {
		config.RecommenderGenerationConfig = req.GetRecommenderConfig().GetRecommenderGenerationConfig()
	}
	return grpcProtoSuccess(config)
}

func gcpStage4GRPCRecommenderGetInsightTypeConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.GetInsightTypeConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	scopeType, scopeID, location, insightType, ok := parseGCPRecommenderInsightTypeConfigName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RecommenderInsightTypeConfig(scopeType, scopeID, location, insightType))
}

func gcpStage4GRPCRecommenderUpdateInsightTypeConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recommenderpb.UpdateInsightTypeConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetInsightTypeConfig() == nil {
		return grpcInvalidArgument("insight_type_config-required")
	}
	scopeType, scopeID, location, insightType, ok := parseGCPRecommenderInsightTypeConfigName(req.GetInsightTypeConfig().GetName())
	if !ok {
		return grpcInvalidArgument("insight_type_config.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	if !gcpRecommenderValidUpdateMaskPaths(req.GetUpdateMask().GetPaths()) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	config := gcpStage4RecommenderInsightTypeConfig(scopeType, scopeID, location, insightType)
	config.Etag = gcpRecommenderInsightTypeConfigETag(insightType) + "-updated"
	if displayName := strings.TrimSpace(req.GetInsightTypeConfig().GetDisplayName()); displayName != "" {
		config.DisplayName = displayName
	}
	if len(req.GetInsightTypeConfig().GetAnnotations()) > 0 {
		config.Annotations = req.GetInsightTypeConfig().GetAnnotations()
	}
	if req.GetInsightTypeConfig().GetInsightTypeGenerationConfig() != nil && req.GetInsightTypeConfig().GetInsightTypeGenerationConfig().GetParams() != nil {
		config.InsightTypeGenerationConfig = req.GetInsightTypeConfig().GetInsightTypeGenerationConfig()
	}
	return grpcProtoSuccess(config)
}

func gcpStage4RecommenderInsight(scopeType, scopeID, location, insightType, insightID string, state recommenderpb.InsightStateInfo_State) *recommenderpb.Insight {
	content := gcpStage4RecommenderStruct(map[string]any{
		"grantedPermissionsCount": 42,
	})
	return &recommenderpb.Insight{
		Name:            fmt.Sprintf("%s/%s/locations/%s/insightTypes/%s/insights/%s", scopeType, scopeID, location, insightType, insightID),
		Description:     fmt.Sprintf("Stackyard insight %s for %s", insightID, insightType),
		TargetResources: []string{fmt.Sprintf("//compute.googleapis.com/projects/%s/zones/%s-a/instances/instance-1", scopeID, location)},
		InsightSubtype:  "PERMISSIONS_USAGE",
		Content:         content,
		LastRefreshTime: timestamppb.New(gcpStage4ReferenceTime),
		ObservationPeriod: &durationpb.Duration{
			Seconds: 604800,
		},
		StateInfo: &recommenderpb.InsightStateInfo{
			State: state,
			StateMetadata: map[string]string{
				"source": "stackyard",
			},
		},
		Category: recommenderpb.Insight_COST,
		Severity: recommenderpb.Insight_HIGH,
		Etag:     gcpRecommenderInsightETag(insightID),
		AssociatedRecommendations: []*recommenderpb.Insight_RecommendationReference{
			{
				Recommendation: fmt.Sprintf("%s/%s/locations/%s/recommenders/google.compute.instance.MachineTypeRecommender/recommendations/recommendation-1", scopeType, scopeID, location),
			},
		},
	}
}

func gcpStage4RecommenderRecommendation(scopeType, scopeID, location, recommenderID, recommendationID string, state recommenderpb.RecommendationStateInfo_State) *recommenderpb.Recommendation {
	return &recommenderpb.Recommendation{
		Name:               fmt.Sprintf("%s/%s/locations/%s/recommenders/%s/recommendations/%s", scopeType, scopeID, location, recommenderID, recommendationID),
		Description:        fmt.Sprintf("Stackyard recommendation %s for %s", recommendationID, recommenderID),
		RecommenderSubtype: "CHANGE_MACHINE_TYPE",
		LastRefreshTime:    timestamppb.New(gcpStage4ReferenceTime),
		PrimaryImpact: &recommenderpb.Impact{
			Category: recommenderpb.Impact_COST,
			Projection: &recommenderpb.Impact_CostProjection{
				CostProjection: &recommenderpb.CostProjection{
					Cost:     &moneypb.Money{CurrencyCode: "USD", Units: -10},
					Duration: durationpb.New(30 * 24 * time.Hour),
				},
			},
		},
		Priority: recommenderpb.Recommendation_P2,
		Content: &recommenderpb.RecommendationContent{
			OperationGroups: []*recommenderpb.OperationGroup{
				{
					Operations: []*recommenderpb.Operation{
						{
							Action:       "replace",
							ResourceType: "compute.googleapis.com/Instance",
							Resource:     fmt.Sprintf("//compute.googleapis.com/projects/%s/zones/%s-a/instances/instance-1", scopeID, location),
							Path:         "/machineType",
							PathValue: &recommenderpb.Operation_Value{
								Value: structpb.NewStringValue("zones/us-central1-a/machineTypes/e2-medium"),
							},
						},
					},
				},
			},
			Overview: gcpStage4RecommenderStruct(map[string]any{
				"currentMachineType":     "e2-standard-4",
				"recommendedMachineType": "e2-medium",
			}),
		},
		StateInfo: &recommenderpb.RecommendationStateInfo{
			State: state,
			StateMetadata: map[string]string{
				"source": "stackyard",
			},
		},
		Etag: gcpRecommenderRecommendationETag(recommendationID),
		AssociatedInsights: []*recommenderpb.Recommendation_InsightReference{
			{
				Insight: fmt.Sprintf("%s/%s/locations/%s/insightTypes/google.iam.policy.Insight/insights/insight-1", scopeType, scopeID, location),
			},
		},
		XorGroupId: "xor-group-1",
	}
}

func gcpStage4RecommenderConfig(scopeType, scopeID, location, recommenderID string) *recommenderpb.RecommenderConfig {
	return &recommenderpb.RecommenderConfig{
		Name: fmt.Sprintf("%s/%s/locations/%s/recommenders/%s/config", scopeType, scopeID, location, recommenderID),
		RecommenderGenerationConfig: &recommenderpb.RecommenderGenerationConfig{
			Params: gcpStage4RecommenderStruct(map[string]any{
				"lookbackDays":          30,
				"minimumSavingsPercent": 10,
			}),
		},
		Etag:       gcpRecommenderConfigETag(recommenderID),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
		RevisionId: "00000001",
		Annotations: map[string]string{
			"source": "stackyard",
		},
		DisplayName: "Stackyard Recommender Config",
	}
}

func gcpStage4RecommenderInsightTypeConfig(scopeType, scopeID, location, insightType string) *recommenderpb.InsightTypeConfig {
	return &recommenderpb.InsightTypeConfig{
		Name: fmt.Sprintf("%s/%s/locations/%s/insightTypes/%s/config", scopeType, scopeID, location, insightType),
		InsightTypeGenerationConfig: &recommenderpb.InsightTypeGenerationConfig{
			Params: gcpStage4RecommenderStruct(map[string]any{
				"lookbackDays": 14,
			}),
		},
		Etag:       gcpRecommenderInsightTypeConfigETag(insightType),
		UpdateTime: timestamppb.New(gcpStage4ReferenceTime),
		RevisionId: "00000001",
		Annotations: map[string]string{
			"source": "stackyard",
		},
		DisplayName: "Stackyard Insight Type Config",
	}
}

func gcpStage4RecommenderRecommendationStateEnum(state string) recommenderpb.RecommendationStateInfo_State {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "ACTIVE":
		return recommenderpb.RecommendationStateInfo_ACTIVE
	case "CLAIMED":
		return recommenderpb.RecommendationStateInfo_CLAIMED
	case "SUCCEEDED":
		return recommenderpb.RecommendationStateInfo_SUCCEEDED
	case "FAILED":
		return recommenderpb.RecommendationStateInfo_FAILED
	case "DISMISSED":
		return recommenderpb.RecommendationStateInfo_DISMISSED
	default:
		return recommenderpb.RecommendationStateInfo_STATE_UNSPECIFIED
	}
}

func gcpStage4RecommenderStruct(values map[string]any) *structpb.Struct {
	st, err := structpb.NewStruct(values)
	if err != nil {
		return &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}
	return st
}

func gcpStage4GRPCRecaptchaEnterpriseCreateAssessment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.CreateAssessmentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetAssessment() == nil || req.GetAssessment().GetEvent() == nil {
		return grpcInvalidArgument("assessment.event-required")
	}
	event := req.GetAssessment().GetEvent()
	if strings.TrimSpace(event.GetToken()) == "" && strings.TrimSpace(event.GetSiteKey()) == "" {
		return grpcInvalidArgument("token-or-site_key-required")
	}
	accountID := ""
	if event.GetUserInfo() != nil {
		accountID = event.GetUserInfo().GetAccountId()
	}
	return grpcProtoSuccess(gcpStage4RecaptchaAssessment(project, "assessment-1", event.GetToken(), event.GetSiteKey(), accountID))
}

func gcpStage4GRPCRecaptchaEnterpriseAnnotateAssessment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.AnnotateAssessmentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPRecaptchaAssessmentName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.AnnotateAssessmentResponse{})
}

func gcpStage4GRPCRecaptchaEnterpriseCreateKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.CreateKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetKey() == nil || strings.TrimSpace(req.GetKey().GetDisplayName()) == "" {
		return grpcInvalidArgument("key.display_name-required")
	}
	keyID := "site-key-1"
	if _, parsedKeyID, ok := parseGCPRecaptchaKeyName(req.GetKey().GetName()); ok {
		keyID = parsedKeyID
	} else if trimmed := strings.TrimSpace(req.GetKey().GetName()); trimmed != "" {
		keyID = trimmed
	}
	key := gcpStage4RecaptchaKey(project, keyID)
	key.DisplayName = req.GetKey().GetDisplayName()
	return grpcProtoSuccess(key)
}

func gcpStage4GRPCRecaptchaEnterpriseListKeys(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.ListKeysRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recaptchaenterprisepb.Key{
		gcpStage4RecaptchaKey(project, "site-key-1"),
		gcpStage4RecaptchaKey(project, "site-key-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.ListKeysResponse{
		Keys:          items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRecaptchaEnterpriseRetrieveLegacySecretKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.RetrieveLegacySecretKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, keyID, ok := parseGCPRecaptchaKeyName(req.GetKey())
	if !ok {
		return grpcInvalidArgument("key-required")
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.RetrieveLegacySecretKeyResponse{
		LegacySecretKey: "legacy-secret-" + keyID,
	})
}

func gcpStage4GRPCRecaptchaEnterpriseGetKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.GetKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, keyID, ok := parseGCPRecaptchaKeyName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RecaptchaKey(project, keyID))
}

func gcpStage4GRPCRecaptchaEnterpriseUpdateKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.UpdateKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetKey() == nil {
		return grpcInvalidArgument("key-required")
	}
	project, keyID, ok := parseGCPRecaptchaKeyName(req.GetKey().GetName())
	if !ok {
		return grpcInvalidArgument("key.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	key := gcpStage4RecaptchaKey(project, keyID)
	if trimmed := strings.TrimSpace(req.GetKey().GetDisplayName()); trimmed != "" {
		key.DisplayName = trimmed
	}
	return grpcProtoSuccess(key)
}

func gcpStage4GRPCRecaptchaEnterpriseDeleteKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.DeleteKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPRecaptchaKeyName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCRecaptchaEnterpriseMigrateKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.MigrateKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, keyID, ok := parseGCPRecaptchaKeyName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	key := gcpStage4RecaptchaKey(project, keyID)
	key.Labels["migrated"] = "true"
	return grpcProtoSuccess(key)
}

func gcpStage4GRPCRecaptchaEnterpriseAddIpOverride(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.AddIpOverrideRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPRecaptchaKeyName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetIpOverrideData() == nil || strings.TrimSpace(req.GetIpOverrideData().GetIp()) == "" {
		return grpcInvalidArgument("ip_override_data-required")
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.AddIpOverrideResponse{})
}

func gcpStage4GRPCRecaptchaEnterpriseRemoveIpOverride(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.RemoveIpOverrideRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPRecaptchaKeyName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	if req.GetIpOverrideData() == nil || strings.TrimSpace(req.GetIpOverrideData().GetIp()) == "" {
		return grpcInvalidArgument("ip_override_data-required")
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.RemoveIpOverrideResponse{})
}

func gcpStage4GRPCRecaptchaEnterpriseListIpOverrides(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.ListIpOverridesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPRecaptchaKeyName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recaptchaenterprisepb.IpOverrideData{
		{Ip: "203.0.113.1", OverrideType: recaptchaenterprisepb.IpOverrideData_ALLOW},
		{Ip: "198.51.100.0/24", OverrideType: recaptchaenterprisepb.IpOverrideData_OVERRIDE_TYPE_UNSPECIFIED},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.ListIpOverridesResponse{
		IpOverrides:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCRecaptchaEnterpriseGetMetrics(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.GetMetricsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, keyID, ok := parseGCPRecaptchaMetricsName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RecaptchaMetrics(project, keyID))
}

func gcpStage4GRPCRecaptchaEnterpriseCreateFirewallPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.CreateFirewallPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetFirewallPolicy() == nil {
		return grpcInvalidArgument("firewall_policy-required")
	}
	policyID := "policy-1"
	if _, id, parsed := parseGCPRecaptchaFirewallPolicyName(req.GetFirewallPolicy().GetName()); parsed {
		policyID = id
	}
	return grpcProtoSuccess(gcpStage4RecaptchaFirewallPolicy(project, policyID))
}

func gcpStage4GRPCRecaptchaEnterpriseListFirewallPolicies(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.ListFirewallPoliciesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recaptchaenterprisepb.FirewallPolicy{
		gcpStage4RecaptchaFirewallPolicy(project, "policy-1"),
		gcpStage4RecaptchaFirewallPolicy(project, "policy-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.ListFirewallPoliciesResponse{
		FirewallPolicies: items[start:end],
		NextPageToken:    next,
	})
}

func gcpStage4GRPCRecaptchaEnterpriseGetFirewallPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.GetFirewallPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, policyID, ok := parseGCPRecaptchaFirewallPolicyName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4RecaptchaFirewallPolicy(project, policyID))
}

func gcpStage4GRPCRecaptchaEnterpriseUpdateFirewallPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.UpdateFirewallPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetFirewallPolicy() == nil {
		return grpcInvalidArgument("firewall_policy-required")
	}
	project, policyID, ok := parseGCPRecaptchaFirewallPolicyName(req.GetFirewallPolicy().GetName())
	if !ok {
		return grpcInvalidArgument("firewall_policy.name-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	return grpcProtoSuccess(gcpStage4RecaptchaFirewallPolicy(project, policyID))
}

func gcpStage4GRPCRecaptchaEnterpriseDeleteFirewallPolicy(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.DeleteFirewallPolicyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, ok := parseGCPRecaptchaFirewallPolicyName(req.GetName()); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCRecaptchaEnterpriseReorderFirewallPolicies(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.ReorderFirewallPoliciesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, ok := parseGCPRecaptchaProjectName(req.GetParent()); !ok {
		return grpcInvalidArgument("parent-required")
	}
	if len(req.GetNames()) == 0 {
		return grpcInvalidArgument("names-required")
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.ReorderFirewallPoliciesResponse{})
}

func gcpStage4GRPCRecaptchaEnterpriseListRelatedAccountGroups(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.ListRelatedAccountGroupsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recaptchaenterprisepb.RelatedAccountGroup{
		{Name: fmt.Sprintf("projects/%s/relatedaccountgroups/group-1", project)},
		{Name: fmt.Sprintf("projects/%s/relatedaccountgroups/group-2", project)},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.ListRelatedAccountGroupsResponse{
		RelatedAccountGroups: items[start:end],
		NextPageToken:        next,
	})
}

func gcpStage4GRPCRecaptchaEnterpriseListRelatedAccountGroupMemberships(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.ListRelatedAccountGroupMembershipsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, groupID, ok := parseGCPRecaptchaRelatedAccountGroupName(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recaptchaenterprisepb.RelatedAccountGroupMembership{
		{
			Name:      fmt.Sprintf("projects/%s/relatedaccountgroups/%s/memberships/member-1", project, groupID),
			AccountId: "user-1",
		},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.ListRelatedAccountGroupMembershipsResponse{
		RelatedAccountGroupMemberships: items[start:end],
		NextPageToken:                  next,
	})
}

func gcpStage4GRPCRecaptchaEnterpriseSearchRelatedAccountGroupMemberships(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &recaptchaenterprisepb.SearchRelatedAccountGroupMembershipsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := parseGCPRecaptchaProjectName(req.GetProject())
	if !ok {
		return grpcInvalidArgument("project-required")
	}
	if strings.TrimSpace(req.GetAccountId()) == "" {
		return grpcInvalidArgument("account_id-required")
	}
	if req.GetPageSize() < 0 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}
	items := []*recaptchaenterprisepb.RelatedAccountGroupMembership{
		{
			Name:      fmt.Sprintf("projects/%s/relatedaccountgroups/group-1/memberships/%s", project, req.GetAccountId()),
			AccountId: req.GetAccountId(),
		},
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return grpcProtoSuccess(&recaptchaenterprisepb.SearchRelatedAccountGroupMembershipsResponse{
		RelatedAccountGroupMemberships: items[start:end],
		NextPageToken:                  next,
	})
}

func gcpStage4RecaptchaAssessment(project, assessmentID, token, siteKey, accountID string) *recaptchaenterprisepb.Assessment {
	if strings.TrimSpace(token) == "" {
		token = "stackyard-token"
	}
	if strings.TrimSpace(siteKey) == "" {
		siteKey = fmt.Sprintf("projects/%s/keys/site-key-1", project)
	}
	return &recaptchaenterprisepb.Assessment{
		Name: fmt.Sprintf("projects/%s/assessments/%s", project, assessmentID),
		Event: &recaptchaenterprisepb.Event{
			Token:   token,
			SiteKey: siteKey,
			UserInfo: &recaptchaenterprisepb.UserInfo{
				AccountId: accountID,
			},
		},
		RiskAnalysis: &recaptchaenterprisepb.RiskAnalysis{
			Score:   0.9,
			Reasons: []recaptchaenterprisepb.RiskAnalysis_ClassificationReason{recaptchaenterprisepb.RiskAnalysis_AUTOMATION},
		},
		TokenProperties: &recaptchaenterprisepb.TokenProperties{
			Valid:         true,
			Action:        "login",
			InvalidReason: recaptchaenterprisepb.TokenProperties_INVALID_REASON_UNSPECIFIED,
		},
	}
}

func gcpStage4RecaptchaKey(project, keyID string) *recaptchaenterprisepb.Key {
	return &recaptchaenterprisepb.Key{
		Name:        fmt.Sprintf("projects/%s/keys/%s", project, keyID),
		DisplayName: "Stackyard Key " + keyID,
		Labels: map[string]string{
			"env": "staged",
		},
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		PlatformSettings: &recaptchaenterprisepb.Key_WebSettings{
			WebSettings: &recaptchaenterprisepb.WebKeySettings{
				AllowAllDomains: true,
				IntegrationType: recaptchaenterprisepb.WebKeySettings_SCORE,
			},
		},
	}
}

func gcpStage4RecaptchaMetrics(project, keyID string) *recaptchaenterprisepb.Metrics {
	return &recaptchaenterprisepb.Metrics{
		Name:      fmt.Sprintf("projects/%s/keys/%s/metrics", project, keyID),
		StartTime: timestamppb.New(gcpStage4ReferenceTime),
		ScoreMetrics: []*recaptchaenterprisepb.ScoreMetrics{
			{
				OverallMetrics: &recaptchaenterprisepb.ScoreDistribution{},
			},
		},
		ChallengeMetrics: []*recaptchaenterprisepb.ChallengeMetrics{
			{
				PageloadCount:  20,
				NocaptchaCount: 15,
				FailedCount:    2,
				PassedCount:    18,
			},
		},
	}
}

func gcpStage4RecaptchaFirewallPolicy(project, policyID string) *recaptchaenterprisepb.FirewallPolicy {
	return &recaptchaenterprisepb.FirewallPolicy{
		Name:      fmt.Sprintf("projects/%s/firewallpolicies/%s", project, policyID),
		Path:      "/*",
		Condition: "true",
	}
}

func grpcProtoSuccess(msg proto.Message) ([]byte, string, string, bool) {
	payload, ok := marshalProtoMessage(msg)
	if !ok {
		return nil, "", "", false
	}
	return payload, "0", "", true
}

func grpcInvalidArgument(reason string) ([]byte, string, string, bool) {
	return nil, "3", strings.TrimSpace(reason), true
}

func grpcFailedPrecondition(reason string) ([]byte, string, string, bool) {
	return nil, "9", strings.TrimSpace(reason), true
}

func grpcNotFound(reason string) ([]byte, string, string, bool) {
	return nil, "5", strings.TrimSpace(reason), true
}

func grpcAlreadyExists(reason string) ([]byte, string, string, bool) {
	return nil, "6", strings.TrimSpace(reason), true
}

func grpcAborted(reason string) ([]byte, string, string, bool) {
	return nil, "10", strings.TrimSpace(reason), true
}

func grpcUnimplemented(reason string) ([]byte, string, string, bool) {
	return nil, "12", strings.TrimSpace(reason), true
}
