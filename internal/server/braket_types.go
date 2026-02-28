package server

type braketType struct {
	Name string
}

// Amazon Braket data types sourced from:
// https://docs.aws.amazon.com/braket/latest/APIReference/API_Types.html
var braketTypes = []braketType{
	{Name: "ActionMetadata"},
	{Name: "AlgorithmSpecification"},
	{Name: "Association"},
	{Name: "ContainerImage"},
	{Name: "DataSource"},
	{Name: "DeviceConfig"},
	{Name: "DeviceQueueInfo"},
	{Name: "DeviceSummary"},
	{Name: "ExperimentalCapabilities"},
	{Name: "HybridJobQueueInfo"},
	{Name: "InputFileConfig"},
	{Name: "InstanceConfig"},
	{Name: "JobCheckpointConfig"},
	{Name: "JobEventDetails"},
	{Name: "JobOutputDataConfig"},
	{Name: "JobStoppingCondition"},
	{Name: "JobSummary"},
	{Name: "ProgramSetValidationFailure"},
	{Name: "QuantumTaskQueueInfo"},
	{Name: "QuantumTaskSummary"},
	{Name: "S3DataSource"},
	{Name: "ScriptModeConfig"},
	{Name: "SearchDevicesFilter"},
	{Name: "SearchJobsFilter"},
	{Name: "SearchQuantumTasksFilter"},
	{Name: "SearchSpendingLimitsFilter"},
	{Name: "SpendingLimitSummary"},
	{Name: "TimePeriod"},
	{Name: "UpdateSpendingLimit"},
}

var braketTypeByName = func() map[string]braketType {
	out := make(map[string]braketType, len(braketTypes))
	for _, dt := range braketTypes {
		out[dt.Name] = dt
	}
	return out
}()
