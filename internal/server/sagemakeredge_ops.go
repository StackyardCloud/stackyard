package server

type sagemakeredgeOperation struct{ Name string }

// Operations sourced from:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_Sagemaker_Edge_Manager.html
var sagemakeredgeOperations = []sagemakeredgeOperation{
	{Name: "GetDeployments"},
	{Name: "GetDeviceRegistration"},
	{Name: "SendHeartbeat"},
}

var sagemakeredgeOperationByName = func() map[string]sagemakeredgeOperation {
	out := make(map[string]sagemakeredgeOperation, len(sagemakeredgeOperations))
	for _, op := range sagemakeredgeOperations {
		out[op.Name] = op
	}
	return out
}()
