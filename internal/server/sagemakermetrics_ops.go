package server

type sagemakermetricsOperation struct{ Name string }

// Operations sourced from:
// https://docs.aws.amazon.com/sagemaker/latest/APIReference/API_Operations_Amazon_SageMaker_Metrics_Service.html
var sagemakermetricsOperations = []sagemakermetricsOperation{
	{Name: "BatchGetMetrics"},
	{Name: "BatchPutMetrics"},
}

var sagemakermetricsOperationByName = func() map[string]sagemakermetricsOperation {
	out := make(map[string]sagemakermetricsOperation, len(sagemakermetricsOperations))
	for _, op := range sagemakermetricsOperations {
		out[op.Name] = op
	}
	return out
}()
