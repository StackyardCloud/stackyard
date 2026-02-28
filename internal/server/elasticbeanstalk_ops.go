package server

type elasticBeanstalkOperation struct {
	Name string
}

var elasticBeanstalkOperations = []elasticBeanstalkOperation{
	{Name: "AbortEnvironmentUpdate"},
	{Name: "ApplyEnvironmentManagedAction"},
	{Name: "AssociateEnvironmentOperationsRole"},
	{Name: "CheckDNSAvailability"},
	{Name: "ComposeEnvironments"},
	{Name: "CreateApplication"},
	{Name: "CreateApplicationVersion"},
	{Name: "CreateConfigurationTemplate"},
	{Name: "CreateEnvironment"},
	{Name: "CreatePlatformVersion"},
	{Name: "CreateStorageLocation"},
	{Name: "DeleteApplication"},
	{Name: "DeleteApplicationVersion"},
	{Name: "DeleteConfigurationTemplate"},
	{Name: "DeleteEnvironmentConfiguration"},
	{Name: "DeletePlatformVersion"},
	{Name: "DescribeAccountAttributes"},
	{Name: "DescribeApplicationVersions"},
	{Name: "DescribeApplications"},
	{Name: "DescribeConfigurationOptions"},
	{Name: "DescribeConfigurationSettings"},
	{Name: "DescribeEnvironmentHealth"},
	{Name: "DescribeEnvironmentManagedActionHistory"},
	{Name: "DescribeEnvironmentManagedActions"},
	{Name: "DescribeEnvironmentResources"},
	{Name: "DescribeEnvironments"},
	{Name: "DescribeEvents"},
	{Name: "DescribeInstancesHealth"},
	{Name: "DescribePlatformVersion"},
	{Name: "DisassociateEnvironmentOperationsRole"},
	{Name: "ListAvailableSolutionStacks"},
	{Name: "ListPlatformBranches"},
	{Name: "ListPlatformVersions"},
	{Name: "ListTagsForResource"},
	{Name: "RebuildEnvironment"},
	{Name: "RequestEnvironmentInfo"},
	{Name: "RestartAppServer"},
	{Name: "RetrieveEnvironmentInfo"},
	{Name: "SwapEnvironmentCNAMEs"},
	{Name: "TerminateEnvironment"},
	{Name: "UpdateApplication"},
	{Name: "UpdateApplicationResourceLifecycle"},
	{Name: "UpdateApplicationVersion"},
	{Name: "UpdateConfigurationTemplate"},
	{Name: "UpdateEnvironment"},
	{Name: "UpdateTagsForResource"},
	{Name: "ValidateConfigurationSettings"},
}

var elasticBeanstalkOperationByName = func() map[string]elasticBeanstalkOperation {
	out := make(map[string]elasticBeanstalkOperation, len(elasticBeanstalkOperations))
	for _, op := range elasticBeanstalkOperations {
		out[op.Name] = op
	}
	return out
}()
