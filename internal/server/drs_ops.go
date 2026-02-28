package server

type drsOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elastic Disaster Recovery operations sourced from:
// https://docs.aws.amazon.com/drs/latest/APIReference/API_Operations.html
var drsOperations = []drsOperation{
	{Name: "AssociateSourceNetworkStack", Method: "POST", URI: "/AssociateSourceNetworkStack"},
	{Name: "CreateExtendedSourceServer", Method: "POST", URI: "/CreateExtendedSourceServer"},
	{Name: "CreateLaunchConfigurationTemplate", Method: "POST", URI: "/CreateLaunchConfigurationTemplate"},
	{Name: "CreateReplicationConfigurationTemplate", Method: "POST", URI: "/CreateReplicationConfigurationTemplate"},
	{Name: "CreateSourceNetwork", Method: "POST", URI: "/CreateSourceNetwork"},
	{Name: "DeleteJob", Method: "POST", URI: "/DeleteJob"},
	{Name: "DeleteLaunchAction", Method: "POST", URI: "/DeleteLaunchAction"},
	{Name: "DeleteLaunchConfigurationTemplate", Method: "POST", URI: "/DeleteLaunchConfigurationTemplate"},
	{Name: "DeleteRecoveryInstance", Method: "POST", URI: "/DeleteRecoveryInstance"},
	{Name: "DeleteReplicationConfigurationTemplate", Method: "POST", URI: "/DeleteReplicationConfigurationTemplate"},
	{Name: "DeleteSourceNetwork", Method: "POST", URI: "/DeleteSourceNetwork"},
	{Name: "DeleteSourceServer", Method: "POST", URI: "/DeleteSourceServer"},
	{Name: "DescribeJobLogItems", Method: "POST", URI: "/DescribeJobLogItems"},
	{Name: "DescribeJobs", Method: "POST", URI: "/DescribeJobs"},
	{Name: "DescribeLaunchConfigurationTemplates", Method: "POST", URI: "/DescribeLaunchConfigurationTemplates"},
	{Name: "DescribeRecoveryInstances", Method: "POST", URI: "/DescribeRecoveryInstances"},
	{Name: "DescribeRecoverySnapshots", Method: "POST", URI: "/DescribeRecoverySnapshots"},
	{Name: "DescribeReplicationConfigurationTemplates", Method: "POST", URI: "/DescribeReplicationConfigurationTemplates"},
	{Name: "DescribeSourceNetworks", Method: "POST", URI: "/DescribeSourceNetworks"},
	{Name: "DescribeSourceServers", Method: "POST", URI: "/DescribeSourceServers"},
	{Name: "DisconnectRecoveryInstance", Method: "POST", URI: "/DisconnectRecoveryInstance"},
	{Name: "DisconnectSourceServer", Method: "POST", URI: "/DisconnectSourceServer"},
	{Name: "ExportSourceNetworkCfnTemplate", Method: "POST", URI: "/ExportSourceNetworkCfnTemplate"},
	{Name: "GetFailbackReplicationConfiguration", Method: "POST", URI: "/GetFailbackReplicationConfiguration"},
	{Name: "GetLaunchConfiguration", Method: "POST", URI: "/GetLaunchConfiguration"},
	{Name: "GetReplicationConfiguration", Method: "POST", URI: "/GetReplicationConfiguration"},
	{Name: "InitializeService", Method: "POST", URI: "/InitializeService"},
	{Name: "ListExtensibleSourceServers", Method: "POST", URI: "/ListExtensibleSourceServers"},
	{Name: "ListLaunchActions", Method: "POST", URI: "/ListLaunchActions"},
	{Name: "ListStagingAccounts", Method: "GET", URI: "/ListStagingAccounts"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutLaunchAction", Method: "POST", URI: "/PutLaunchAction"},
	{Name: "RetryDataReplication", Method: "POST", URI: "/RetryDataReplication"},
	{Name: "ReverseReplication", Method: "POST", URI: "/ReverseReplication"},
	{Name: "StartFailbackLaunch", Method: "POST", URI: "/StartFailbackLaunch"},
	{Name: "StartRecovery", Method: "POST", URI: "/StartRecovery"},
	{Name: "StartReplication", Method: "POST", URI: "/StartReplication"},
	{Name: "StartSourceNetworkRecovery", Method: "POST", URI: "/StartSourceNetworkRecovery"},
	{Name: "StartSourceNetworkReplication", Method: "POST", URI: "/StartSourceNetworkReplication"},
	{Name: "StopFailback", Method: "POST", URI: "/StopFailback"},
	{Name: "StopReplication", Method: "POST", URI: "/StopReplication"},
	{Name: "StopSourceNetworkReplication", Method: "POST", URI: "/StopSourceNetworkReplication"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "TerminateRecoveryInstances", Method: "POST", URI: "/TerminateRecoveryInstances"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateFailbackReplicationConfiguration", Method: "POST", URI: "/UpdateFailbackReplicationConfiguration"},
	{Name: "UpdateLaunchConfiguration", Method: "POST", URI: "/UpdateLaunchConfiguration"},
	{Name: "UpdateLaunchConfigurationTemplate", Method: "POST", URI: "/UpdateLaunchConfigurationTemplate"},
	{Name: "UpdateReplicationConfiguration", Method: "POST", URI: "/UpdateReplicationConfiguration"},
	{Name: "UpdateReplicationConfigurationTemplate", Method: "POST", URI: "/UpdateReplicationConfigurationTemplate"},
}

var drsOperationByName = func() map[string]drsOperation {
	out := make(map[string]drsOperation, len(drsOperations))
	for _, op := range drsOperations {
		out[op.Name] = op
	}
	return out
}()
