package server

type appRunnerDataType struct {
	Name string
}

// AWS App Runner data types sourced from:
// https://docs.aws.amazon.com/apprunner/latest/api/API_Types.html
var appRunnerDataTypes = []appRunnerDataType{
	{Name: "AuthenticationConfiguration"},
	{Name: "AutoScalingConfiguration"},
	{Name: "AutoScalingConfigurationSummary"},
	{Name: "CertificateValidationRecord"},
	{Name: "CodeConfiguration"},
	{Name: "CodeConfigurationValues"},
	{Name: "CodeRepository"},
	{Name: "Connection"},
	{Name: "ConnectionSummary"},
	{Name: "CustomDomain"},
	{Name: "EgressConfiguration"},
	{Name: "EncryptionConfiguration"},
	{Name: "HealthCheckConfiguration"},
	{Name: "ImageConfiguration"},
	{Name: "ImageRepository"},
	{Name: "IngressConfiguration"},
	{Name: "IngressVpcConfiguration"},
	{Name: "InstanceConfiguration"},
	{Name: "ListVpcIngressConnectionsFilter"},
	{Name: "NetworkConfiguration"},
	{Name: "ObservabilityConfiguration"},
	{Name: "ObservabilityConfigurationSummary"},
	{Name: "OperationSummary"},
	{Name: "Service"},
	{Name: "ServiceObservabilityConfiguration"},
	{Name: "ServiceSummary"},
	{Name: "SourceCodeVersion"},
	{Name: "SourceConfiguration"},
	{Name: "Tag"},
	{Name: "TraceConfiguration"},
	{Name: "VpcConnector"},
	{Name: "VpcDNSTarget"},
	{Name: "VpcIngressConnection"},
	{Name: "VpcIngressConnectionSummary"},
}

var appRunnerDataTypeByName = func() map[string]appRunnerDataType {
	out := make(map[string]appRunnerDataType, len(appRunnerDataTypes))
	for _, dt := range appRunnerDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
