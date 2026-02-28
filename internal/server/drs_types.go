package server

type drsDataType struct {
	Name string
}

// AWS Elastic Disaster Recovery data types sourced from:
// https://docs.aws.amazon.com/drs/latest/APIReference/API_Types.html
var drsDataTypes = []drsDataType{
	{Name: "Account"},
	{Name: "CPU"},
	{Name: "ConversionProperties"},
	{Name: "DataReplicationError"},
	{Name: "DataReplicationInfo"},
	{Name: "DataReplicationInfoReplicatedDisk"},
	{Name: "DataReplicationInitiation"},
	{Name: "DataReplicationInitiationStep"},
	{Name: "DescribeJobsRequestFilters"},
	{Name: "DescribeRecoveryInstancesRequestFilters"},
	{Name: "DescribeRecoverySnapshotsRequestFilters"},
	{Name: "DescribeSourceNetworksRequestFilters"},
	{Name: "DescribeSourceServersRequestFilters"},
	{Name: "Disk"},
	{Name: "EventResourceData"},
	{Name: "IdentificationHints"},
	{Name: "Job"},
	{Name: "JobLog"},
	{Name: "JobLogEventData"},
	{Name: "LaunchAction"},
	{Name: "LaunchActionParameter"},
	{Name: "LaunchActionRun"},
	{Name: "LaunchActionsRequestFilters"},
	{Name: "LaunchActionsStatus"},
	{Name: "LaunchConfiguration"},
	{Name: "LaunchConfigurationTemplate"},
	{Name: "LaunchIntoInstanceProperties"},
	{Name: "Licensing"},
	{Name: "LifeCycle"},
	{Name: "LifeCycleLastLaunch"},
	{Name: "LifeCycleLastLaunchInitiated"},
	{Name: "NetworkInterface"},
	{Name: "OS"},
	{Name: "PITPolicyRule"},
	{Name: "ParticipatingResource"},
	{Name: "ParticipatingResourceID"},
	{Name: "ParticipatingServer"},
	{Name: "ProductCode"},
	{Name: "RecoveryInstance"},
	{Name: "RecoveryInstanceDataReplicationError"},
	{Name: "RecoveryInstanceDataReplicationInfo"},
	{Name: "RecoveryInstanceDataReplicationInfoReplicatedDisk"},
	{Name: "RecoveryInstanceDataReplicationInitiation"},
	{Name: "RecoveryInstanceDataReplicationInitiationStep"},
	{Name: "RecoveryInstanceDisk"},
	{Name: "RecoveryInstanceFailback"},
	{Name: "RecoveryInstanceProperties"},
	{Name: "RecoveryLifeCycle"},
	{Name: "RecoverySnapshot"},
	{Name: "ReplicationConfiguration"},
	{Name: "ReplicationConfigurationReplicatedDisk"},
	{Name: "ReplicationConfigurationTemplate"},
	{Name: "SourceCloudProperties"},
	{Name: "SourceNetwork"},
	{Name: "SourceNetworkData"},
	{Name: "SourceProperties"},
	{Name: "SourceServer"},
	{Name: "StagingArea"},
	{Name: "StagingSourceServer"},
	{Name: "StartRecoveryRequestSourceServer"},
	{Name: "StartSourceNetworkRecoveryRequestNetworkEntry"},
	{Name: "ValidationExceptionField"},
}

var drsDataTypeByName = func() map[string]drsDataType {
	out := make(map[string]drsDataType, len(drsDataTypes))
	for _, dt := range drsDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
