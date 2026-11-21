package server

type sagemakerruntimeOperation struct {
	Name string
}

// Amazon SageMaker Runtime operations sourced from:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_Runtime.html
var sagemakerruntimeOperations = []sagemakerruntimeOperation{
	{Name: "InvokeEndpoint"},
	{Name: "InvokeEndpointAsync"},
	{Name: "InvokeEndpointWithResponseStream"},
}

var sagemakerruntimeOperationByName = func() map[string]sagemakerruntimeOperation {
	out := make(map[string]sagemakerruntimeOperation, len(sagemakerruntimeOperations))
	for _, op := range sagemakerruntimeOperations {
		out[op.Name] = op
	}
	return out
}()
