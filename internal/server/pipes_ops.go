package server

type pipesOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon EventBridge Pipes operations sourced from:
// https://docs.aws.amazon.com/eventbridge/latest/pipes-reference/API_Operations.html
var pipesOperations = []pipesOperation{
	{Name: "CreatePipe", Method: "POST", URI: "/v1/pipes/{Name}"},
	{Name: "DeletePipe", Method: "DELETE", URI: "/v1/pipes/{Name}"},
	{Name: "DescribePipe", Method: "GET", URI: "/v1/pipes/{Name}"},
	{Name: "ListPipes", Method: "GET", URI: "/v1/pipes"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartPipe", Method: "POST", URI: "/v1/pipes/{Name}/start"},
	{Name: "StopPipe", Method: "POST", URI: "/v1/pipes/{Name}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdatePipe", Method: "PUT", URI: "/v1/pipes/{Name}"},
}

var pipesOperationByName = func() map[string]pipesOperation {
	out := make(map[string]pipesOperation, len(pipesOperations))
	for _, op := range pipesOperations {
		out[op.Name] = op
	}
	return out
}()
