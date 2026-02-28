package server

type mskConnectDataType struct {
	Name string
}

// Amazon MSK Connect data types sourced from:
// https://docs.aws.amazon.com/MSKC/latest/mskc/API_Types.html
var mskConnectDataTypes = []mskConnectDataType{
	{Name: "ApacheKafkaCluster"},
	{Name: "ApacheKafkaClusterDescription"},
	{Name: "AutoScaling"},
	{Name: "AutoScalingDescription"},
	{Name: "AutoScalingUpdate"},
	{Name: "Capacity"},
	{Name: "CapacityDescription"},
	{Name: "CapacityUpdate"},
	{Name: "CloudWatchLogsLogDelivery"},
	{Name: "CloudWatchLogsLogDeliveryDescription"},
	{Name: "ConnectorOperationStep"},
	{Name: "ConnectorOperationSummary"},
	{Name: "ConnectorSummary"},
	{Name: "CustomPlugin"},
	{Name: "CustomPluginDescription"},
	{Name: "CustomPluginFileDescription"},
	{Name: "CustomPluginLocation"},
	{Name: "CustomPluginLocationDescription"},
	{Name: "CustomPluginRevisionSummary"},
	{Name: "CustomPluginSummary"},
	{Name: "FirehoseLogDelivery"},
	{Name: "FirehoseLogDeliveryDescription"},
	{Name: "KafkaCluster"},
	{Name: "KafkaClusterClientAuthentication"},
	{Name: "KafkaClusterClientAuthenticationDescription"},
	{Name: "KafkaClusterDescription"},
	{Name: "KafkaClusterEncryptionInTransit"},
	{Name: "KafkaClusterEncryptionInTransitDescription"},
	{Name: "LogDelivery"},
	{Name: "LogDeliveryDescription"},
	{Name: "Plugin"},
	{Name: "PluginDescription"},
	{Name: "ProvisionedCapacity"},
	{Name: "ProvisionedCapacityDescription"},
	{Name: "ProvisionedCapacityUpdate"},
	{Name: "S3Location"},
	{Name: "S3LocationDescription"},
	{Name: "S3LogDelivery"},
	{Name: "S3LogDeliveryDescription"},
	{Name: "ScaleInPolicy"},
	{Name: "ScaleInPolicyDescription"},
	{Name: "ScaleInPolicyUpdate"},
	{Name: "ScaleOutPolicy"},
	{Name: "ScaleOutPolicyDescription"},
	{Name: "ScaleOutPolicyUpdate"},
	{Name: "StateDescription"},
	{Name: "Vpc"},
	{Name: "VpcDescription"},
	{Name: "WorkerConfiguration"},
	{Name: "WorkerConfigurationDescription"},
	{Name: "WorkerConfigurationRevisionDescription"},
	{Name: "WorkerConfigurationRevisionSummary"},
	{Name: "WorkerConfigurationSummary"},
	{Name: "WorkerLogDelivery"},
	{Name: "WorkerLogDeliveryDescription"},
	{Name: "WorkerSetting"},
}

var mskConnectDataTypeByName = func() map[string]mskConnectDataType {
	out := make(map[string]mskConnectDataType, len(mskConnectDataTypes))
	for _, dt := range mskConnectDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
