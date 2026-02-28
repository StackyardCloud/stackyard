package server

type mgnDataType struct {
	Name string
}

// AWS Application Migration Service (MGN) data types sourced from:

// https://docs.aws.amazon.com/mgn/latest/APIReference/API_Types.html

var mgnDataTypes = []mgnDataType{
	{Name: "Application"},
	{Name: "ApplicationAggregatedStatus"},
	{Name: "CPU"},
	{Name: "ChangeServerLifeCycleStateSourceServerLifecycle"},
	{Name: "Connector"},
	{Name: "ConnectorSsmCommandConfig"},
	{Name: "DataReplicationError"},
	{Name: "DataReplicationInfo"},
	{Name: "DataReplicationInfoReplicatedDisk"},
	{Name: "DataReplicationInitiation"},
	{Name: "DataReplicationInitiationStep"},
	{Name: "DescribeJobsRequestFilters"},
	{Name: "DescribeSourceServersRequestFilters"},
	{Name: "Disk"},
	{Name: "ErrorDetails"},
	{Name: "ExportErrorData"},
	{Name: "ExportTask"},
	{Name: "ExportTaskError"},
	{Name: "ExportTaskSummary"},
	{Name: "IdentificationHints"},
	{Name: "ImportErrorData"},
	{Name: "ImportTask"},
	{Name: "ImportTaskError"},
	{Name: "ImportTaskSummary"},
	{Name: "ImportTaskSummaryApplications"},
	{Name: "ImportTaskSummaryServers"},
	{Name: "ImportTaskSummaryWaves"},
	{Name: "Job"},
	{Name: "JobLog"},
	{Name: "JobLogEventData"},
	{Name: "JobPostLaunchActionsLaunchStatus"},
	{Name: "LaunchConfigurationTemplate"},
	{Name: "LaunchTemplateDiskConf"},
	{Name: "LaunchedInstance"},
	{Name: "Licensing"},
	{Name: "LifeCycle"},
	{Name: "LifeCycleLastCutover"},
	{Name: "LifeCycleLastCutoverFinalized"},
	{Name: "LifeCycleLastCutoverInitiated"},
	{Name: "LifeCycleLastCutoverReverted"},
	{Name: "LifeCycleLastTest"},
	{Name: "LifeCycleLastTestFinalized"},
	{Name: "LifeCycleLastTestInitiated"},
	{Name: "LifeCycleLastTestReverted"},
	{Name: "ListApplicationsRequestFilters"},
	{Name: "ListConnectorsRequestFilters"},
	{Name: "ListExportsRequestFilters"},
	{Name: "ListImportsRequestFilters"},
	{Name: "ListWavesRequestFilters"},
	{Name: "ManagedAccount"},
	{Name: "NetworkInterface"},
	{Name: "OS"},
	{Name: "ParticipatingServer"},
	{Name: "PostLaunchActions"},
	{Name: "PostLaunchActionsStatus"},
	{Name: "ReplicationConfigurationReplicatedDisk"},
	{Name: "ReplicationConfigurationTemplate"},
	{Name: "S3BucketSource"},
	{Name: "SourceProperties"},
	{Name: "SourceServer"},
	{Name: "SourceServerActionDocument"},
	{Name: "SourceServerActionsRequestFilters"},
	{Name: "SourceServerConnectorAction"},
	{Name: "SsmDocument"},
	{Name: "SsmExternalParameter"},
	{Name: "SsmParameterStoreParameter"},
	{Name: "TemplateActionDocument"},
	{Name: "TemplateActionsRequestFilters"},
	{Name: "UpdateWave"},
	{Name: "ValidationExceptionField"},
	{Name: "VcenterClient"},
	{Name: "Wave"},
	{Name: "WaveAggregatedStatus"},
}

var mgnDataTypeByName = func() map[string]mgnDataType {

	out := make(map[string]mgnDataType, len(mgnDataTypes))

	for _, dt := range mgnDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
