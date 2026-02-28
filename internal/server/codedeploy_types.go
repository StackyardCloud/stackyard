package server

type codeDeployDataType struct {
	Name string
}

// AWS CodeDeploy data types sourced from:
// https://docs.aws.amazon.com/codedeploy/latest/APIReference/API_Types.html
var codeDeployDataTypes = []codeDeployDataType{
	{Name: "Alarm"},
	{Name: "AlarmConfiguration"},
	{Name: "ApplicationInfo"},
	{Name: "AppSpecContent"},
	{Name: "AutoRollbackConfiguration"},
	{Name: "AutoScalingGroup"},
	{Name: "BlueGreenDeploymentConfiguration"},
	{Name: "BlueInstanceTerminationOption"},
	{Name: "CloudFormationTarget"},
	{Name: "DeploymentConfigInfo"},
	{Name: "DeploymentGroupInfo"},
	{Name: "DeploymentInfo"},
	{Name: "DeploymentOverview"},
	{Name: "DeploymentReadyOption"},
	{Name: "DeploymentStyle"},
	{Name: "DeploymentTarget"},
	{Name: "Diagnostics"},
	{Name: "EC2TagFilter"},
	{Name: "EC2TagSet"},
	{Name: "ECSService"},
	{Name: "ECSTarget"},
	{Name: "ECSTaskSet"},
	{Name: "ELBInfo"},
	{Name: "ErrorInformation"},
	{Name: "GenericRevisionInfo"},
	{Name: "GitHubLocation"},
	{Name: "GreenFleetProvisioningOption"},
	{Name: "InstanceInfo"},
	{Name: "InstanceSummary"},
	{Name: "InstanceTarget"},
	{Name: "LambdaFunctionInfo"},
	{Name: "LambdaTarget"},
	{Name: "LastDeploymentInfo"},
	{Name: "LifecycleEvent"},
	{Name: "LoadBalancerInfo"},
	{Name: "MinimumHealthyHosts"},
	{Name: "MinimumHealthyHostsPerZone"},
	{Name: "OnPremisesTagSet"},
	{Name: "RawString"},
	{Name: "RelatedDeployments"},
	{Name: "RevisionInfo"},
	{Name: "RevisionLocation"},
	{Name: "RollbackInfo"},
	{Name: "S3Location"},
	{Name: "Tag"},
	{Name: "TagFilter"},
	{Name: "TargetGroupInfo"},
	{Name: "TargetGroupPairInfo"},
	{Name: "TargetInstances"},
	{Name: "TimeBasedCanary"},
	{Name: "TimeBasedLinear"},
	{Name: "TimeRange"},
	{Name: "TrafficRoute"},
	{Name: "TrafficRoutingConfig"},
	{Name: "TriggerConfig"},
	{Name: "ZonalConfig"},
	{Name: "UpdateDeploymentGroup"},
}

var codeDeployDataTypeByName = func() map[string]codeDeployDataType {
	out := make(map[string]codeDeployDataType, len(codeDeployDataTypes))
	for _, dt := range codeDeployDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
