package server

type mskResource struct {
	Name string
}

// Amazon MSK API v2 resources sourced from:
// https://docs.aws.amazon.com/MSK/2.0/APIReference/resources.html
// and linked resource model pages.
var mskResources = []mskResource{
	{Name: "BrokerAZDistribution"},
	{Name: "BrokerCountUpdateInfo"},
	{Name: "BrokerEBSVolumeInfo"},
	{Name: "BrokerLogs"},
	{Name: "BrokerNodeGroupInfo"},
	{Name: "BrokerSoftwareInfo"},
	{Name: "ClientAuthentication"},
	{Name: "ClientBroker"},
	{Name: "CloudWatchLogs"},
	{Name: "Cluster"},
	{Name: "ClusterOperationStep"},
	{Name: "ClusterOperationStepInfo"},
	{Name: "ClusterOperationV2"},
	{Name: "ClusterOperationV2Provisioned"},
	{Name: "ClusterOperationV2Serverless"},
	{Name: "ClusterOperationV2Summary"},
	{Name: "ClusterState"},
	{Name: "ClusterType"},
	{Name: "ConfigurationInfo"},
	{Name: "ConnectivityInfo"},
	{Name: "CreateClusterV2Request"},
	{Name: "CreateClusterV2Response"},
	{Name: "CustomerActionStatus"},
	{Name: "DescribeClusterOperationV2Response"},
	{Name: "DescribeClusterV2Response"},
	{Name: "EBSStorageInfo"},
	{Name: "EncryptionAtRest"},
	{Name: "EncryptionInTransit"},
	{Name: "EncryptionInfo"},
	{Name: "EnhancedMonitoring"},
	{Name: "ErrorInfo"},
	{Name: "Firehose"},
	{Name: "IAM"},
	{Name: "JmxExporter"},
	{Name: "JmxExporterInfo"},
	{Name: "ListClusterOperationsV2Response"},
	{Name: "ListClustersV2Response"},
	{Name: "LoggingInfo"},
	{Name: "MutableClusterInfo"},
	{Name: "NodeExporter"},
	{Name: "NodeExporterInfo"},
	{Name: "OpenMonitoring"},
	{Name: "OpenMonitoringInfo"},
	{Name: "Prometheus"},
	{Name: "PrometheusInfo"},
	{Name: "Provisioned"},
	{Name: "ProvisionedRequest"},
	{Name: "ProvisionedThroughput"},
	{Name: "PublicAccess"},
	{Name: "Rebalancing"},
	{Name: "RebalancingStatus"},
	{Name: "S3"},
	{Name: "Sasl"},
	{Name: "Scram"},
	{Name: "Serverless"},
	{Name: "ServerlessClientAuthentication"},
	{Name: "ServerlessRequest"},
	{Name: "ServerlessSasl"},
	{Name: "StateInfo"},
	{Name: "StorageInfo"},
	{Name: "StorageMode"},
	{Name: "Tls"},
	{Name: "Unauthenticated"},
	{Name: "UserIdentity"},
	{Name: "UserIdentityType"},
	{Name: "VpcConfig"},
	{Name: "VpcConnectionInfo"},
	{Name: "VpcConnectionInfoServerless"},
	{Name: "VpcConnectivity"},
	{Name: "VpcConnectivityClientAuthentication"},
	{Name: "VpcConnectivityIAM"},
	{Name: "VpcConnectivitySasl"},
	{Name: "VpcConnectivityScram"},
	{Name: "VpcConnectivityTls"},
}

var mskResourceByName = func() map[string]mskResource {
	out := make(map[string]mskResource, len(mskResources))
	for _, r := range mskResources {
		out[r.Name] = r
	}
	return out
}()
