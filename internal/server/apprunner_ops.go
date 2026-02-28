package server

type appRunnerOperation struct {
	Name string
}

// AWS App Runner actions sourced from:
// https://docs.aws.amazon.com/apprunner/latest/api/API_Operations.html
var appRunnerOperations = []appRunnerOperation{
	{Name: "AssociateCustomDomain"},
	{Name: "CreateAutoScalingConfiguration"},
	{Name: "CreateConnection"},
	{Name: "CreateObservabilityConfiguration"},
	{Name: "CreateService"},
	{Name: "CreateVpcConnector"},
	{Name: "CreateVpcIngressConnection"},
	{Name: "DeleteAutoScalingConfiguration"},
	{Name: "DeleteConnection"},
	{Name: "DeleteObservabilityConfiguration"},
	{Name: "DeleteService"},
	{Name: "DeleteVpcConnector"},
	{Name: "DeleteVpcIngressConnection"},
	{Name: "DescribeAutoScalingConfiguration"},
	{Name: "DescribeCustomDomains"},
	{Name: "DescribeObservabilityConfiguration"},
	{Name: "DescribeService"},
	{Name: "DescribeVpcConnector"},
	{Name: "DescribeVpcIngressConnection"},
	{Name: "DisassociateCustomDomain"},
	{Name: "ListAutoScalingConfigurations"},
	{Name: "ListConnections"},
	{Name: "ListObservabilityConfigurations"},
	{Name: "ListOperations"},
	{Name: "ListServices"},
	{Name: "ListServicesForAutoScalingConfiguration"},
	{Name: "ListTagsForResource"},
	{Name: "ListVpcConnectors"},
	{Name: "ListVpcIngressConnections"},
	{Name: "PauseService"},
	{Name: "ResumeService"},
	{Name: "StartDeployment"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateDefaultAutoScalingConfiguration"},
	{Name: "UpdateService"},
	{Name: "UpdateVpcIngressConnection"},
}

var appRunnerOperationByName = func() map[string]appRunnerOperation {
	out := make(map[string]appRunnerOperation, len(appRunnerOperations))
	for _, op := range appRunnerOperations {
		out[op.Name] = op
	}
	return out
}()
