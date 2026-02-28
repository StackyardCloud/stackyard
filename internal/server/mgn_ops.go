package server

type mgnOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Application Migration Service (MGN) operations sourced from:

// https://docs.aws.amazon.com/mgn/latest/APIReference/API_Operations.html

var mgnOperations = []mgnOperation{
	{Name: "ArchiveApplication", Method: "POST", URI: "/ArchiveApplication"},
	{Name: "ArchiveWave", Method: "POST", URI: "/ArchiveWave"},
	{Name: "AssociateApplications", Method: "POST", URI: "/AssociateApplications"},
	{Name: "AssociateSourceServers", Method: "POST", URI: "/AssociateSourceServers"},
	{Name: "ChangeServerLifeCycleState", Method: "POST", URI: "/ChangeServerLifeCycleState"},
	{Name: "CreateApplication", Method: "POST", URI: "/CreateApplication"},
	{Name: "CreateConnector", Method: "POST", URI: "/CreateConnector"},
	{Name: "CreateLaunchConfigurationTemplate", Method: "POST", URI: "/CreateLaunchConfigurationTemplate"},
	{Name: "CreateReplicationConfigurationTemplate", Method: "POST", URI: "/CreateReplicationConfigurationTemplate"},
	{Name: "CreateWave", Method: "POST", URI: "/CreateWave"},
	{Name: "DeleteApplication", Method: "POST", URI: "/DeleteApplication"},
	{Name: "DeleteConnector", Method: "POST", URI: "/DeleteConnector"},
	{Name: "DeleteJob", Method: "POST", URI: "/DeleteJob"},
	{Name: "DeleteLaunchConfigurationTemplate", Method: "POST", URI: "/DeleteLaunchConfigurationTemplate"},
	{Name: "DeleteReplicationConfigurationTemplate", Method: "POST", URI: "/DeleteReplicationConfigurationTemplate"},
	{Name: "DeleteSourceServer", Method: "POST", URI: "/DeleteSourceServer"},
	{Name: "DeleteVcenterClient", Method: "POST", URI: "/DeleteVcenterClient"},
	{Name: "DeleteWave", Method: "POST", URI: "/DeleteWave"},
	{Name: "DescribeJobLogItems", Method: "POST", URI: "/DescribeJobLogItems"},
	{Name: "DescribeJobs", Method: "POST", URI: "/DescribeJobs"},
	{Name: "DescribeLaunchConfigurationTemplates", Method: "POST", URI: "/DescribeLaunchConfigurationTemplates"},
	{Name: "DescribeReplicationConfigurationTemplates", Method: "POST", URI: "/DescribeReplicationConfigurationTemplates"},
	{Name: "DescribeSourceServers", Method: "POST", URI: "/DescribeSourceServers"},
	{Name: "DescribeVcenterClients", Method: "GET", URI: "/DescribeVcenterClients"},
	{Name: "DisassociateApplications", Method: "POST", URI: "/DisassociateApplications"},
	{Name: "DisassociateSourceServers", Method: "POST", URI: "/DisassociateSourceServers"},
	{Name: "DisconnectFromService", Method: "POST", URI: "/DisconnectFromService"},
	{Name: "FinalizeCutover", Method: "POST", URI: "/FinalizeCutover"},
	{Name: "GetLaunchConfiguration", Method: "POST", URI: "/GetLaunchConfiguration"},
	{Name: "GetReplicationConfiguration", Method: "POST", URI: "/GetReplicationConfiguration"},
	{Name: "InitializeService", Method: "POST", URI: "/InitializeService"},
	{Name: "ListApplications", Method: "POST", URI: "/ListApplications"},
	{Name: "ListConnectors", Method: "POST", URI: "/ListConnectors"},
	{Name: "ListExportErrors", Method: "POST", URI: "/ListExportErrors"},
	{Name: "ListExports", Method: "POST", URI: "/ListExports"},
	{Name: "ListImportErrors", Method: "POST", URI: "/ListImportErrors"},
	{Name: "ListImports", Method: "POST", URI: "/ListImports"},
	{Name: "ListManagedAccounts", Method: "POST", URI: "/ListManagedAccounts"},
	{Name: "ListSourceServerActions", Method: "POST", URI: "/ListSourceServerActions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListTemplateActions", Method: "POST", URI: "/ListTemplateActions"},
	{Name: "ListWaves", Method: "POST", URI: "/ListWaves"},
	{Name: "MarkAsArchived", Method: "POST", URI: "/MarkAsArchived"},
	{Name: "PauseReplication", Method: "POST", URI: "/PauseReplication"},
	{Name: "PutSourceServerAction", Method: "POST", URI: "/PutSourceServerAction"},
	{Name: "PutTemplateAction", Method: "POST", URI: "/PutTemplateAction"},
	{Name: "RemoveSourceServerAction", Method: "POST", URI: "/RemoveSourceServerAction"},
	{Name: "RemoveTemplateAction", Method: "POST", URI: "/RemoveTemplateAction"},
	{Name: "ResumeReplication", Method: "POST", URI: "/ResumeReplication"},
	{Name: "RetryDataReplication", Method: "POST", URI: "/RetryDataReplication"},
	{Name: "StartCutover", Method: "POST", URI: "/StartCutover"},
	{Name: "StartExport", Method: "POST", URI: "/StartExport"},
	{Name: "StartImport", Method: "POST", URI: "/StartImport"},
	{Name: "StartReplication", Method: "POST", URI: "/StartReplication"},
	{Name: "StartTest", Method: "POST", URI: "/StartTest"},
	{Name: "StopReplication", Method: "POST", URI: "/StopReplication"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "TerminateTargetInstances", Method: "POST", URI: "/TerminateTargetInstances"},
	{Name: "UnarchiveApplication", Method: "POST", URI: "/UnarchiveApplication"},
	{Name: "UnarchiveWave", Method: "POST", URI: "/UnarchiveWave"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateApplication", Method: "POST", URI: "/UpdateApplication"},
	{Name: "UpdateConnector", Method: "POST", URI: "/UpdateConnector"},
	{Name: "UpdateLaunchConfiguration", Method: "POST", URI: "/UpdateLaunchConfiguration"},
	{Name: "UpdateLaunchConfigurationTemplate", Method: "POST", URI: "/UpdateLaunchConfigurationTemplate"},
	{Name: "UpdateReplicationConfiguration", Method: "POST", URI: "/UpdateReplicationConfiguration"},
	{Name: "UpdateReplicationConfigurationTemplate", Method: "POST", URI: "/UpdateReplicationConfigurationTemplate"},
	{Name: "UpdateSourceServer", Method: "POST", URI: "/UpdateSourceServer"},
	{Name: "UpdateSourceServerReplicationType", Method: "POST", URI: "/UpdateSourceServerReplicationType"},
	{Name: "UpdateWave", Method: "POST", URI: "/UpdateWave"},
}

var mgnOperationByName = func() map[string]mgnOperation {
	out := make(map[string]mgnOperation, len(mgnOperations))
	for _, op := range mgnOperations {
		out[op.Name] = op
	}
	return out
}()
