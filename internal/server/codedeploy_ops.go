package server

type codeDeployOperation struct {
	Name string
}

// AWS CodeDeploy operations sourced from:
// https://docs.aws.amazon.com/codedeploy/latest/APIReference/API_Operations.html
var codeDeployOperations = []codeDeployOperation{
	{Name: "AddTagsToOnPremisesInstances"},
	{Name: "BatchGetApplicationRevisions"},
	{Name: "BatchGetApplications"},
	{Name: "BatchGetDeploymentGroups"},
	{Name: "BatchGetDeploymentInstances"},
	{Name: "BatchGetDeployments"},
	{Name: "BatchGetDeploymentTargets"},
	{Name: "BatchGetOnPremisesInstances"},
	{Name: "ContinueDeployment"},
	{Name: "CreateApplication"},
	{Name: "CreateDeployment"},
	{Name: "CreateDeploymentConfig"},
	{Name: "CreateDeploymentGroup"},
	{Name: "DeleteApplication"},
	{Name: "DeleteDeploymentConfig"},
	{Name: "DeleteDeploymentGroup"},
	{Name: "DeleteGitHubAccountToken"},
	{Name: "DeleteResourcesByExternalId"},
	{Name: "DeregisterOnPremisesInstance"},
	{Name: "GetApplication"},
	{Name: "GetApplicationRevision"},
	{Name: "GetDeployment"},
	{Name: "GetDeploymentConfig"},
	{Name: "GetDeploymentGroup"},
	{Name: "GetDeploymentInstance"},
	{Name: "GetDeploymentTarget"},
	{Name: "GetOnPremisesInstance"},
	{Name: "ListApplicationRevisions"},
	{Name: "ListApplications"},
	{Name: "ListDeploymentConfigs"},
	{Name: "ListDeploymentGroups"},
	{Name: "ListDeploymentInstances"},
	{Name: "ListDeployments"},
	{Name: "ListDeploymentTargets"},
	{Name: "ListGitHubAccountTokenNames"},
	{Name: "ListOnPremisesInstances"},
	{Name: "ListTagsForResource"},
	{Name: "PutLifecycleEventHookExecutionStatus"},
	{Name: "RegisterApplicationRevision"},
	{Name: "RegisterOnPremisesInstance"},
	{Name: "RemoveTagsFromOnPremisesInstances"},
	{Name: "SkipWaitTimeForInstanceTermination"},
	{Name: "StopDeployment"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateApplication"},
	{Name: "UpdateDeploymentGroup"},
}

var codeDeployOperationByName = func() map[string]codeDeployOperation {
	out := make(map[string]codeDeployOperation, len(codeDeployOperations))
	for _, op := range codeDeployOperations {
		out[op.Name] = op
	}
	return out
}()
