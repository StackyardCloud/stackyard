package server

type pcsDataType struct {
	Name string
}

// AWS PCS data types sourced from:
// https://docs.aws.amazon.com/pcs/latest/APIReference/API_Types.html
var pcsDataTypes = []pcsDataType{
	{Name: "Accounting"},
	{Name: "AccountingRequest"},
	{Name: "Cluster"},
	{Name: "ClusterSlurmConfiguration"},
	{Name: "ClusterSlurmConfigurationRequest"},
	{Name: "ClusterSummary"},
	{Name: "ComputeNodeGroup"},
	{Name: "ComputeNodeGroupConfiguration"},
	{Name: "ComputeNodeGroupSlurmConfiguration"},
	{Name: "ComputeNodeGroupSlurmConfigurationRequest"},
	{Name: "ComputeNodeGroupSummary"},
	{Name: "CustomLaunchTemplate"},
	{Name: "Endpoint"},
	{Name: "ErrorInfo"},
	{Name: "InstanceConfig"},
	{Name: "JwtAuth"},
	{Name: "JwtKey"},
	{Name: "Networking"},
	{Name: "NetworkingRequest"},
	{Name: "Queue"},
	{Name: "QueueSlurmConfiguration"},
	{Name: "QueueSlurmConfigurationRequest"},
	{Name: "QueueSummary"},
	{Name: "ScalingConfiguration"},
	{Name: "ScalingConfigurationRequest"},
	{Name: "Scheduler"},
	{Name: "SchedulerRequest"},
	{Name: "SlurmAuthKey"},
	{Name: "SlurmCustomSetting"},
	{Name: "SlurmRest"},
	{Name: "SlurmRestRequest"},
	{Name: "SpotOptions"},
	{Name: "UpdateAccountingRequest"},
	{Name: "UpdateClusterSlurmConfigurationRequest"},
	{Name: "UpdateComputeNodeGroupSlurmConfigurationRequest"},
	{Name: "UpdateQueue"},
	{Name: "UpdateQueueSlurmConfigurationRequest"},
	{Name: "UpdateSlurmRestRequest"},
	{Name: "ValidationExceptionField"},
}

var pcsDataTypeByName = func() map[string]pcsDataType {
	out := make(map[string]pcsDataType, len(pcsDataTypes))
	for _, t := range pcsDataTypes {
		out[t.Name] = t
	}
	return out
}()
