package server

type emrContainersDataType struct {
	Name string
}

// Amazon EMR on EKS data types sourced from:
// https://docs.aws.amazon.com/emr-on-eks/latest/APIReference/API_Types.html
var emrContainersDataTypes = []emrContainersDataType{
	{Name: "AuthorizationConfiguration"},
	{Name: "Certificate"},
	{Name: "CloudWatchMonitoringConfiguration"},
	{Name: "Configuration"},
	{Name: "ConfigurationOverrides"},
	{Name: "ContainerInfo"},
	{Name: "ContainerLogRotationConfiguration"},
	{Name: "ContainerProvider"},
	{Name: "Credentials"},
	{Name: "EksInfo"},
	{Name: "EncryptionConfiguration"},
	{Name: "Endpoint"},
	{Name: "InTransitEncryptionConfiguration"},
	{Name: "JobDriver"},
	{Name: "JobRun"},
	{Name: "JobTemplate"},
	{Name: "JobTemplateData"},
	{Name: "LakeFormationConfiguration"},
	{Name: "ManagedLogs"},
	{Name: "MonitoringConfiguration"},
	{Name: "ParametricCloudWatchMonitoringConfiguration"},
	{Name: "ParametricConfigurationOverrides"},
	{Name: "ParametricMonitoringConfiguration"},
	{Name: "ParametricS3MonitoringConfiguration"},
	{Name: "RetryPolicyConfiguration"},
	{Name: "RetryPolicyExecution"},
	{Name: "S3MonitoringConfiguration"},
	{Name: "SecureNamespaceInfo"},
	{Name: "SecurityConfiguration"},
	{Name: "SecurityConfigurationData"},
	{Name: "SparkSqlJobDriver"},
	{Name: "SparkSubmitJobDriver"},
	{Name: "TemplateParameterConfiguration"},
	{Name: "TLSCertificateConfiguration"},
	{Name: "VirtualCluster"},
	{Name: "UntagResource"},
}

var emrContainersDataTypeByName = func() map[string]emrContainersDataType {
	out := make(map[string]emrContainersDataType, len(emrContainersDataTypes))
	for _, dt := range emrContainersDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
