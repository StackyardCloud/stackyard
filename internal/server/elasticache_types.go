package server

type elastiCacheDataType struct {
	Name string
}

// Amazon ElastiCache data types sourced from:
// https://docs.aws.amazon.com/AmazonElastiCache/latest/APIReference/API_Types.html
var elastiCacheDataTypes = []elastiCacheDataType{
	{Name: "Authentication"},
	{Name: "AuthenticationMode"},
	{Name: "AvailabilityZone"},
	{Name: "CacheCluster"},
	{Name: "CacheEngineVersion"},
	{Name: "CacheNode"},
	{Name: "CacheNodeTypeSpecificParameter"},
	{Name: "CacheNodeTypeSpecificValue"},
	{Name: "CacheNodeUpdateStatus"},
	{Name: "CacheParameterGroup"},
	{Name: "CacheParameterGroupStatus"},
	{Name: "CacheSecurityGroup"},
	{Name: "CacheSecurityGroupMembership"},
	{Name: "CacheSubnetGroup"},
	{Name: "CacheUsageLimits"},
	{Name: "CloudWatchLogsDestinationDetails"},
	{Name: "ConfigureShard"},
	{Name: "CustomerNodeEndpoint"},
	{Name: "DataStorage"},
	{Name: "DestinationDetails"},
	{Name: "EC2SecurityGroup"},
	{Name: "ECPUPerSecond"},
	{Name: "Endpoint"},
	{Name: "EngineDefaults"},
	{Name: "Event"},
	{Name: "Filter"},
	{Name: "GlobalNodeGroup"},
	{Name: "GlobalReplicationGroup"},
	{Name: "GlobalReplicationGroupInfo"},
	{Name: "GlobalReplicationGroupMember"},
	{Name: "KinesisFirehoseDestinationDetails"},
	{Name: "LogDeliveryConfiguration"},
	{Name: "LogDeliveryConfigurationRequest"},
	{Name: "NodeGroup"},
	{Name: "NodeGroupConfiguration"},
	{Name: "NodeGroupMember"},
	{Name: "NodeGroupMemberUpdateStatus"},
	{Name: "NodeGroupUpdateStatus"},
	{Name: "NodeSnapshot"},
	{Name: "NotificationConfiguration"},
	{Name: "Parameter"},
	{Name: "ParameterNameValue"},
	{Name: "PendingLogDeliveryConfiguration"},
	{Name: "PendingModifiedValues"},
	{Name: "ProcessedUpdateAction"},
	{Name: "RecurringCharge"},
	{Name: "RegionalConfiguration"},
	{Name: "ReplicationGroup"},
	{Name: "ReplicationGroupPendingModifiedValues"},
	{Name: "ReservedCacheNode"},
	{Name: "ReservedCacheNodesOffering"},
	{Name: "ReshardingConfiguration"},
	{Name: "ReshardingStatus"},
	{Name: "ScaleConfig"},
	{Name: "SecurityGroupMembership"},
	{Name: "ServerlessCache"},
	{Name: "ServerlessCacheConfiguration"},
	{Name: "ServerlessCacheSnapshot"},
	{Name: "ServiceUpdate"},
	{Name: "SlotMigration"},
	{Name: "Snapshot"},
	{Name: "Subnet"},
	{Name: "SubnetOutpost"},
	{Name: "Tag"},
	{Name: "TestMigration"},
	{Name: "TimeRangeFilter"},
	{Name: "UnprocessedUpdateAction"},
	{Name: "UpdateAction"},
	{Name: "User"},
	{Name: "UserGroup"},
	{Name: "UserGroupPendingChanges"},
	{Name: "UserGroupsUpdateStatus"},
}

var elastiCacheDataTypeByName = func() map[string]elastiCacheDataType {
	out := make(map[string]elastiCacheDataType, len(elastiCacheDataTypes))
	for _, dt := range elastiCacheDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
