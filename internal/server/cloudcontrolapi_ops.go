package server

type cloudControlAPIOperation struct {
	Name string
}

// AWS Cloud Control API operations sourced from:
// https://docs.aws.amazon.com/cloudcontrolapi/latest/APIReference/API_Operations.html
var cloudControlAPIOperations = []cloudControlAPIOperation{
	{Name: "CancelResourceRequest"},
	{Name: "CreateResource"},
	{Name: "DeleteResource"},
	{Name: "GetResource"},
	{Name: "GetResourceRequestStatus"},
	{Name: "ListResourceRequests"},
	{Name: "ListResources"},
	{Name: "UpdateResource"},
}

var cloudControlAPIOperationByName = func() map[string]cloudControlAPIOperation {
	out := make(map[string]cloudControlAPIOperation, len(cloudControlAPIOperations))
	for _, op := range cloudControlAPIOperations {
		out[op.Name] = op
	}
	return out
}()
