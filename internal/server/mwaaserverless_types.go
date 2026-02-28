package server

type mwaaServerlessDataType struct {
	Name string
}

// Amazon Managed Workflows for Apache Airflow Serverless data types sourced from:
// https://docs.aws.amazon.com/mwaa-serverless/latest/APIReference/API_Types.html
var mwaaServerlessDataTypes = []mwaaServerlessDataType{
	{Name: "DefinitionS3Location"},
	{Name: "EncryptionConfiguration"},
	{Name: "LoggingConfiguration"},
	{Name: "NetworkConfiguration"},
	{Name: "RunDetailSummary"},
	{Name: "ScheduleConfiguration"},
	{Name: "TaskInstanceSummary"},
	{Name: "ValidationExceptionField"},
	{Name: "WorkflowRunDetail"},
	{Name: "WorkflowRunSummary"},
	{Name: "WorkflowSummary"},
	{Name: "WorkflowVersionSummary"},
}

var mwaaServerlessDataTypeByName = func() map[string]mwaaServerlessDataType {
	out := make(map[string]mwaaServerlessDataType, len(mwaaServerlessDataTypes))
	for _, dt := range mwaaServerlessDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
