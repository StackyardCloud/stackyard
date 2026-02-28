package server

type emrOperation struct {
	Name string
}

// Amazon EMR actions sourced from:
// https://docs.aws.amazon.com/emr/latest/APIReference/API_Operations.html
var emrOperations = []emrOperation{
	{Name: "AddInstanceFleet"},
	{Name: "AddInstanceGroups"},
	{Name: "AddJobFlowSteps"},
	{Name: "AddTags"},
	{Name: "CancelSteps"},
	{Name: "CreatePersistentAppUI"},
	{Name: "CreateSecurityConfiguration"},
	{Name: "CreateStudio"},
	{Name: "CreateStudioSessionMapping"},
	{Name: "DeleteSecurityConfiguration"},
	{Name: "DeleteStudio"},
	{Name: "DeleteStudioSessionMapping"},
	{Name: "DescribeCluster"},
	{Name: "DescribeJobFlows"},
	{Name: "DescribeNotebookExecution"},
	{Name: "DescribePersistentAppUI"},
	{Name: "DescribeReleaseLabel"},
	{Name: "DescribeSecurityConfiguration"},
	{Name: "DescribeStep"},
	{Name: "DescribeStudio"},
	{Name: "GetAutoTerminationPolicy"},
	{Name: "GetBlockPublicAccessConfiguration"},
	{Name: "GetClusterSessionCredentials"},
	{Name: "GetManagedScalingPolicy"},
	{Name: "GetOnClusterAppUIPresignedURL"},
	{Name: "GetPersistentAppUIPresignedURL"},
	{Name: "GetStudioSessionMapping"},
	{Name: "ListBootstrapActions"},
	{Name: "ListClusters"},
	{Name: "ListInstanceFleets"},
	{Name: "ListInstanceGroups"},
	{Name: "ListInstances"},
	{Name: "ListNotebookExecutions"},
	{Name: "ListReleaseLabels"},
	{Name: "ListSecurityConfigurations"},
	{Name: "ListSteps"},
	{Name: "ListStudios"},
	{Name: "ListStudioSessionMappings"},
	{Name: "ListSupportedInstanceTypes"},
	{Name: "ModifyCluster"},
	{Name: "ModifyInstanceFleet"},
	{Name: "ModifyInstanceGroups"},
	{Name: "PutAutoScalingPolicy"},
	{Name: "PutAutoTerminationPolicy"},
	{Name: "PutBlockPublicAccessConfiguration"},
	{Name: "PutManagedScalingPolicy"},
	{Name: "RemoveAutoScalingPolicy"},
	{Name: "RemoveAutoTerminationPolicy"},
	{Name: "RemoveManagedScalingPolicy"},
	{Name: "RemoveTags"},
	{Name: "RunJobFlow"},
	{Name: "SetKeepJobFlowAliveWhenNoSteps"},
	{Name: "SetTerminationProtection"},
	{Name: "SetUnhealthyNodeReplacement"},
	{Name: "SetVisibleToAllUsers"},
	{Name: "StartNotebookExecution"},
	{Name: "StopNotebookExecution"},
	{Name: "TerminateJobFlows"},
	{Name: "UpdateStudio"},
	{Name: "UpdateStudioSessionMapping"},
}

var emrOperationByName = func() map[string]emrOperation {
	out := make(map[string]emrOperation, len(emrOperations))
	for _, op := range emrOperations {
		out[op.Name] = op
	}
	return out
}()
