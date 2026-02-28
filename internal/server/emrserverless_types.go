package server

type emrServerlessDataType struct {
	Name string
}

// Amazon EMR Serverless data types sourced from:
// https://docs.aws.amazon.com/emr-serverless/latest/APIReference/API_Types.html
var emrServerlessDataTypes = []emrServerlessDataType{
	{Name: "Application"},
	{Name: "ApplicationSummary"},
	{Name: "AutoStartConfig"},
	{Name: "AutoStopConfig"},
	{Name: "CloudWatchLoggingConfiguration"},
	{Name: "Configuration"},
	{Name: "ConfigurationOverrides"},
	{Name: "DiskEncryptionConfiguration"},
	{Name: "Hive"},
	{Name: "IdentityCenterConfiguration"},
	{Name: "IdentityCenterConfigurationInput"},
	{Name: "ImageConfiguration"},
	{Name: "ImageConfigurationInput"},
	{Name: "InitialCapacityConfig"},
	{Name: "InteractiveConfiguration"},
	{Name: "JobDriver"},
	{Name: "JobLevelCostAllocationConfiguration"},
	{Name: "JobRun"},
	{Name: "JobRunAttemptSummary"},
	{Name: "JobRunExecutionIamPolicy"},
	{Name: "JobRunSummary"},
	{Name: "ManagedPersistenceMonitoringConfiguration"},
	{Name: "MaximumAllowedResources"},
	{Name: "MonitoringConfiguration"},
	{Name: "NetworkConfiguration"},
	{Name: "PrometheusMonitoringConfiguration"},
	{Name: "ResourceUtilization"},
	{Name: "RetryPolicy"},
	{Name: "S3MonitoringConfiguration"},
	{Name: "SchedulerConfiguration"},
	{Name: "SparkSubmit"},
	{Name: "TotalResourceUtilization"},
	{Name: "UpdateApplication"},
	{Name: "WorkerResourceConfig"},
	{Name: "WorkerTypeSpecification"},
	{Name: "WorkerTypeSpecificationInput"},
}

var emrServerlessDataTypeByName = func() map[string]emrServerlessDataType {
	out := make(map[string]emrServerlessDataType, len(emrServerlessDataTypes))
	for _, dt := range emrServerlessDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
