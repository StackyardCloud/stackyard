package server

type protonDataType struct {
	Name string
}

// AWS Proton data types sourced from:
// https://docs.aws.amazon.com/proton/latest/APIReference/API_Types.html
var protonDataTypes = []protonDataType{
	{Name: "AccountSettings"},
	{Name: "CompatibleEnvironmentTemplate"},
	{Name: "CompatibleEnvironmentTemplateInput"},
	{Name: "Component"},
	{Name: "ComponentState"},
	{Name: "ComponentSummary"},
	{Name: "CountsSummary"},
	{Name: "Deployment"},
	{Name: "DeploymentState"},
	{Name: "DeploymentSummary"},
	{Name: "Environment"},
	{Name: "EnvironmentAccountConnection"},
	{Name: "EnvironmentAccountConnectionSummary"},
	{Name: "EnvironmentState"},
	{Name: "EnvironmentSummary"},
	{Name: "EnvironmentTemplate"},
	{Name: "EnvironmentTemplateFilter"},
	{Name: "EnvironmentTemplateSummary"},
	{Name: "EnvironmentTemplateVersion"},
	{Name: "EnvironmentTemplateVersionSummary"},
	{Name: "ListServiceInstancesFilter"},
	{Name: "Output"},
	{Name: "ProvisionedResource"},
	{Name: "Repository"},
	{Name: "RepositoryBranch"},
	{Name: "RepositoryBranchInput"},
	{Name: "RepositorySummary"},
	{Name: "RepositorySyncAttempt"},
	{Name: "RepositorySyncDefinition"},
	{Name: "RepositorySyncEvent"},
	{Name: "ResourceCountsSummary"},
	{Name: "ResourceSyncAttempt"},
	{Name: "ResourceSyncEvent"},
	{Name: "Revision"},
	{Name: "S3ObjectSource"},
	{Name: "Service"},
	{Name: "ServiceInstance"},
	{Name: "ServiceInstanceState"},
	{Name: "ServiceInstanceSummary"},
	{Name: "ServicePipeline"},
	{Name: "ServicePipelineState"},
	{Name: "ServiceSummary"},
	{Name: "ServiceSyncBlockerSummary"},
	{Name: "ServiceSyncConfig"},
	{Name: "ServiceTemplate"},
	{Name: "ServiceTemplateSummary"},
	{Name: "ServiceTemplateVersion"},
	{Name: "ServiceTemplateVersionSummary"},
	{Name: "SyncBlocker"},
	{Name: "SyncBlockerContext"},
	{Name: "Tag"},
	{Name: "TemplateSyncConfig"},
	{Name: "TemplateVersionSourceInput"},
}

var protonDataTypeByName = func() map[string]protonDataType {
	out := make(map[string]protonDataType, len(protonDataTypes))
	for _, dt := range protonDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
