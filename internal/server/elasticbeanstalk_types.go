package server

type elasticBeanstalkDataType struct {
	Name string
}

// Elastic Beanstalk data types sourced from:
// https://docs.aws.amazon.com/elasticbeanstalk/latest/api/API_Types.html
var elasticBeanstalkDataTypes = []elasticBeanstalkDataType{
	{Name: "ApplicationDescription"},
	{Name: "ApplicationMetrics"},
	{Name: "ApplicationResourceLifecycleConfig"},
	{Name: "ApplicationVersionDescription"},
	{Name: "ApplicationVersionLifecycleConfig"},
	{Name: "AutoScalingGroup"},
	{Name: "BuildConfiguration"},
	{Name: "Builder"},
	{Name: "CPUUtilization"},
	{Name: "ConfigurationOptionDescription"},
	{Name: "ConfigurationOptionSetting"},
	{Name: "ConfigurationSettingsDescription"},
	{Name: "CustomAmi"},
	{Name: "Deployment"},
	{Name: "EnvironmentDescription"},
	{Name: "EnvironmentInfoDescription"},
	{Name: "EnvironmentLink"},
	{Name: "EnvironmentResourceDescription"},
	{Name: "EnvironmentResourcesDescription"},
	{Name: "EnvironmentTier"},
	{Name: "EventDescription"},
	{Name: "Instance"},
	{Name: "InstanceHealthSummary"},
	{Name: "Latency"},
	{Name: "LaunchConfiguration"},
	{Name: "LaunchTemplate"},
	{Name: "Listener"},
	{Name: "LoadBalancer"},
	{Name: "LoadBalancerDescription"},
	{Name: "ManagedAction"},
	{Name: "ManagedActionHistoryItem"},
	{Name: "MaxAgeRule"},
	{Name: "MaxCountRule"},
	{Name: "OptionRestrictionRegex"},
	{Name: "OptionSpecification"},
	{Name: "PlatformBranchSummary"},
	{Name: "PlatformDescription"},
	{Name: "PlatformFilter"},
	{Name: "PlatformFramework"},
	{Name: "PlatformProgrammingLanguage"},
	{Name: "PlatformSummary"},
	{Name: "Queue"},
	{Name: "ResourceQuota"},
	{Name: "ResourceQuotas"},
	{Name: "S3Location"},
	{Name: "SearchFilter"},
	{Name: "SingleInstanceHealth"},
	{Name: "SolutionStackDescription"},
	{Name: "SourceBuildInformation"},
	{Name: "SourceConfiguration"},
	{Name: "StatusCodes"},
	{Name: "SystemStatus"},
	{Name: "Tag"},
	{Name: "Trigger"},
	{Name: "ValidateConfigurationSettings"},
	{Name: "ValidationMessage"},
}

var elasticBeanstalkDataTypeByName = func() map[string]elasticBeanstalkDataType {
	out := make(map[string]elasticBeanstalkDataType, len(elasticBeanstalkDataTypes))
	for _, dt := range elasticBeanstalkDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
