package server

type imageBuilderOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS EC2 Image Builder actions sourced from:
// https://docs.aws.amazon.com/imagebuilder/latest/APIReference/API_Operations.html
var imageBuilderOperations = []imageBuilderOperation{
	{Name: "CancelImageCreation", Method: "PUT", URI: "/CancelImageCreation"},
	{Name: "CancelLifecycleExecution", Method: "PUT", URI: "/CancelLifecycleExecution"},
	{Name: "CreateComponent", Method: "PUT", URI: "/CreateComponent"},
	{Name: "CreateContainerRecipe", Method: "PUT", URI: "/CreateContainerRecipe"},
	{Name: "CreateDistributionConfiguration", Method: "PUT", URI: "/CreateDistributionConfiguration"},
	{Name: "CreateImage", Method: "PUT", URI: "/CreateImage"},
	{Name: "CreateImagePipeline", Method: "PUT", URI: "/CreateImagePipeline"},
	{Name: "CreateImageRecipe", Method: "PUT", URI: "/CreateImageRecipe"},
	{Name: "CreateInfrastructureConfiguration", Method: "PUT", URI: "/CreateInfrastructureConfiguration"},
	{Name: "CreateLifecyclePolicy", Method: "PUT", URI: "/CreateLifecyclePolicy"},
	{Name: "CreateWorkflow", Method: "PUT", URI: "/CreateWorkflow"},
	{Name: "DeleteComponent", Method: "DELETE", URI: "/DeleteComponent?componentBuildVersionArn="},
	{Name: "DeleteContainerRecipe", Method: "DELETE", URI: "/DeleteContainerRecipe?containerRecipeArn="},
	{Name: "DeleteDistributionConfiguration", Method: "DELETE", URI: "/DeleteDistributionConfiguration?distributionConfigurationArn="},
	{Name: "DeleteImage", Method: "DELETE", URI: "/DeleteImage?imageBuildVersionArn="},
	{Name: "DeleteImagePipeline", Method: "DELETE", URI: "/DeleteImagePipeline?imagePipelineArn="},
	{Name: "DeleteImageRecipe", Method: "DELETE", URI: "/DeleteImageRecipe?imageRecipeArn="},
	{Name: "DeleteInfrastructureConfiguration", Method: "DELETE", URI: "/DeleteInfrastructureConfiguration?infrastructureConfigurationArn="},
	{Name: "DeleteLifecyclePolicy", Method: "DELETE", URI: "/DeleteLifecyclePolicy?lifecyclePolicyArn="},
	{Name: "DeleteWorkflow", Method: "DELETE", URI: "/DeleteWorkflow?workflowBuildVersionArn="},
	{Name: "DistributeImage", Method: "PUT", URI: "/DistributeImage"},
	{Name: "GetComponent", Method: "GET", URI: "/GetComponent?componentBuildVersionArn="},
	{Name: "GetComponentPolicy", Method: "GET", URI: "/GetComponentPolicy?componentArn="},
	{Name: "GetContainerRecipe", Method: "GET", URI: "/GetContainerRecipe?containerRecipeArn="},
	{Name: "GetContainerRecipePolicy", Method: "GET", URI: "/GetContainerRecipePolicy?containerRecipeArn="},
	{Name: "GetDistributionConfiguration", Method: "GET", URI: "/GetDistributionConfiguration?distributionConfigurationArn="},
	{Name: "GetImage", Method: "GET", URI: "/GetImage?imageBuildVersionArn="},
	{Name: "GetImagePipeline", Method: "GET", URI: "/GetImagePipeline?imagePipelineArn="},
	{Name: "GetImagePolicy", Method: "GET", URI: "/GetImagePolicy?imageArn="},
	{Name: "GetImageRecipe", Method: "GET", URI: "/GetImageRecipe?imageRecipeArn="},
	{Name: "GetImageRecipePolicy", Method: "GET", URI: "/GetImageRecipePolicy?imageRecipeArn="},
	{Name: "GetInfrastructureConfiguration", Method: "GET", URI: "/GetInfrastructureConfiguration?infrastructureConfigurationArn="},
	{Name: "GetLifecycleExecution", Method: "GET", URI: "/GetLifecycleExecution?lifecycleExecutionId="},
	{Name: "GetLifecyclePolicy", Method: "GET", URI: "/GetLifecyclePolicy?lifecyclePolicyArn="},
	{Name: "GetMarketplaceResource", Method: "POST", URI: "/GetMarketplaceResource"},
	{Name: "GetWorkflow", Method: "GET", URI: "/GetWorkflow?workflowBuildVersionArn="},
	{Name: "GetWorkflowExecution", Method: "GET", URI: "/GetWorkflowExecution?workflowExecutionId="},
	{Name: "GetWorkflowStepExecution", Method: "GET", URI: "/GetWorkflowStepExecution?stepExecutionId="},
	{Name: "ImportComponent", Method: "PUT", URI: "/ImportComponent"},
	{Name: "ImportDiskImage", Method: "PUT", URI: "/ImportDiskImage"},
	{Name: "ImportVmImage", Method: "PUT", URI: "/ImportVmImage"},
	{Name: "ListComponentBuildVersions", Method: "POST", URI: "/ListComponentBuildVersions"},
	{Name: "ListComponents", Method: "POST", URI: "/ListComponents"},
	{Name: "ListContainerRecipes", Method: "POST", URI: "/ListContainerRecipes"},
	{Name: "ListDistributionConfigurations", Method: "POST", URI: "/ListDistributionConfigurations"},
	{Name: "ListImageBuildVersions", Method: "POST", URI: "/ListImageBuildVersions"},
	{Name: "ListImagePackages", Method: "POST", URI: "/ListImagePackages"},
	{Name: "ListImagePipelineImages", Method: "POST", URI: "/ListImagePipelineImages"},
	{Name: "ListImagePipelines", Method: "POST", URI: "/ListImagePipelines"},
	{Name: "ListImageRecipes", Method: "POST", URI: "/ListImageRecipes"},
	{Name: "ListImageScanFindingAggregations", Method: "POST", URI: "/ListImageScanFindingAggregations"},
	{Name: "ListImageScanFindings", Method: "POST", URI: "/ListImageScanFindings"},
	{Name: "ListImages", Method: "POST", URI: "/ListImages"},
	{Name: "ListInfrastructureConfigurations", Method: "POST", URI: "/ListInfrastructureConfigurations"},
	{Name: "ListLifecycleExecutionResources", Method: "POST", URI: "/ListLifecycleExecutionResources"},
	{Name: "ListLifecycleExecutions", Method: "POST", URI: "/ListLifecycleExecutions"},
	{Name: "ListLifecyclePolicies", Method: "POST", URI: "/ListLifecyclePolicies"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/"},
	{Name: "ListWaitingWorkflowSteps", Method: "POST", URI: "/ListWaitingWorkflowSteps"},
	{Name: "ListWorkflowBuildVersions", Method: "POST", URI: "/ListWorkflowBuildVersions"},
	{Name: "ListWorkflowExecutions", Method: "POST", URI: "/ListWorkflowExecutions"},
	{Name: "ListWorkflowStepExecutions", Method: "POST", URI: "/ListWorkflowStepExecutions"},
	{Name: "ListWorkflows", Method: "POST", URI: "/ListWorkflows"},
	{Name: "PutComponentPolicy", Method: "PUT", URI: "/PutComponentPolicy"},
	{Name: "PutContainerRecipePolicy", Method: "PUT", URI: "/PutContainerRecipePolicy"},
	{Name: "PutImagePolicy", Method: "PUT", URI: "/PutImagePolicy"},
	{Name: "PutImageRecipePolicy", Method: "PUT", URI: "/PutImageRecipePolicy"},
	{Name: "RetryImage", Method: "PUT", URI: "/RetryImage"},
	{Name: "SendWorkflowStepAction", Method: "PUT", URI: "/SendWorkflowStepAction"},
	{Name: "StartImagePipelineExecution", Method: "PUT", URI: "/StartImagePipelineExecution"},
	{Name: "StartResourceStateUpdate", Method: "PUT", URI: "/StartResourceStateUpdate"},
	{Name: "TagResource", Method: "POST", URI: "/tags/"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/"},
	{Name: "UpdateDistributionConfiguration", Method: "PUT", URI: "/UpdateDistributionConfiguration"},
	{Name: "UpdateImagePipeline", Method: "PUT", URI: "/UpdateImagePipeline"},
	{Name: "UpdateInfrastructureConfiguration", Method: "PUT", URI: "/UpdateInfrastructureConfiguration"},
	{Name: "UpdateLifecyclePolicy", Method: "PUT", URI: "/UpdateLifecyclePolicy"},
}

var imageBuilderOperationByName = func() map[string]imageBuilderOperation {
	out := make(map[string]imageBuilderOperation, len(imageBuilderOperations))
	for _, op := range imageBuilderOperations {
		out[op.Name] = op
	}
	return out
}()
