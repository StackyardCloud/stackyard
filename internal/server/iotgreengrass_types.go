package server

type iotGreengrassDataType struct {
	Name string
}

// AWS IoT Greengrass (V2) data types sourced from:
// https://docs.aws.amazon.com/greengrass/v2/APIReference/API_Types.html
var iotGreengrassDataTypes = []iotGreengrassDataType{
	{Name: "AssociateClientDeviceWithCoreDeviceEntry"},
	{Name: "AssociateClientDeviceWithCoreDeviceErrorEntry"},
	{Name: "AssociatedClientDevice"},
	{Name: "CloudComponentStatus"},
	{Name: "Component"},
	{Name: "ComponentCandidate"},
	{Name: "ComponentConfigurationUpdate"},
	{Name: "ComponentDependencyRequirement"},
	{Name: "ComponentDeploymentSpecification"},
	{Name: "ComponentLatestVersion"},
	{Name: "ComponentPlatform"},
	{Name: "ComponentRunWith"},
	{Name: "ComponentVersionListItem"},
	{Name: "ConnectivityInfo"},
	{Name: "CoreDevice"},
	{Name: "Deployment"},
	{Name: "DeploymentComponentUpdatePolicy"},
	{Name: "DeploymentConfigurationValidationPolicy"},
	{Name: "DeploymentIoTJobConfiguration"},
	{Name: "DeploymentPolicies"},
	{Name: "DisassociateClientDeviceFromCoreDeviceEntry"},
	{Name: "DisassociateClientDeviceFromCoreDeviceErrorEntry"},
	{Name: "EffectiveDeployment"},
	{Name: "EffectiveDeploymentStatusDetails"},
	{Name: "InstalledComponent"},
	{Name: "IoTJobAbortConfig"},
	{Name: "IoTJobAbortCriteria"},
	{Name: "IoTJobExecutionsRolloutConfig"},
	{Name: "IoTJobExponentialRolloutRate"},
	{Name: "IoTJobRateIncreaseCriteria"},
	{Name: "IoTJobTimeoutConfig"},
	{Name: "LambdaContainerParams"},
	{Name: "LambdaDeviceMount"},
	{Name: "LambdaEventSource"},
	{Name: "LambdaExecutionParameters"},
	{Name: "LambdaFunctionRecipeSource"},
	{Name: "LambdaLinuxProcessParams"},
	{Name: "LambdaVolumeMount"},
	{Name: "ResolvedComponentVersion"},
	{Name: "SystemResourceLimits"},
	{Name: "ValidationExceptionField"},
}

var iotGreengrassDataTypeByName = func() map[string]iotGreengrassDataType {
	out := make(map[string]iotGreengrassDataType, len(iotGreengrassDataTypes))
	for _, dt := range iotGreengrassDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
