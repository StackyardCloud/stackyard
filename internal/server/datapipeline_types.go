package server

type dataPipelineDataType struct {
	Name string
}

// AWS Data Pipeline data types sourced from:
// https://docs.aws.amazon.com/datapipeline/latest/APIReference/API_Types.html
var dataPipelineDataTypes = []dataPipelineDataType{
	{Name: "Field"},
	{Name: "InstanceIdentity"},
	{Name: "Operator"},
	{Name: "ParameterAttribute"},
	{Name: "ParameterObject"},
	{Name: "ParameterValue"},
	{Name: "PipelineDescription"},
	{Name: "PipelineIdName"},
	{Name: "PipelineObject"},
	{Name: "Query"},
	{Name: "Selector"},
	{Name: "Tag"},
	{Name: "TaskObject"},
	{Name: "ValidationError"},
	{Name: "ValidationWarning"},
}

var dataPipelineDataTypeByName = func() map[string]dataPipelineDataType {
	out := make(map[string]dataPipelineDataType, len(dataPipelineDataTypes))
	for _, dt := range dataPipelineDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
