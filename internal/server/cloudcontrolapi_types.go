package server

type cloudControlAPIDataType struct {
	Name string
}

// AWS Cloud Control API data types sourced from:
// https://docs.aws.amazon.com/cloudcontrolapi/latest/APIReference/API_Types.html
var cloudControlAPIDataTypes = []cloudControlAPIDataType{
	{Name: "HookProgressEvent"},
	{Name: "ProgressEvent"},
	{Name: "ResourceDescription"},
	{Name: "ResourceRequestStatusFilter"},
}

var cloudControlAPIDataTypeByName = func() map[string]cloudControlAPIDataType {
	out := make(map[string]cloudControlAPIDataType, len(cloudControlAPIDataTypes))
	for _, dt := range cloudControlAPIDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
