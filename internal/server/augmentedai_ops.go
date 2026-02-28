package server

type augmentedAIOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Augmented AI Runtime operations sourced from:
// https://docs.aws.amazon.com/augmented-ai/2019-11-07/APIReference/API_Operations.html
var augmentedAIOperations = []augmentedAIOperation{
	{Name: "DeleteHumanLoop", Method: "DELETE", URI: "/human-loops/{HumanLoopName}"},
	{Name: "DescribeHumanLoop", Method: "GET", URI: "/human-loops/{HumanLoopName}"},
	{Name: "ListHumanLoops", Method: "GET", URI: "/human-loops"},
	{Name: "StartHumanLoop", Method: "POST", URI: "/human-loops"},
	{Name: "StopHumanLoop", Method: "POST", URI: "/human-loops/stop"},
}

var augmentedAIOperationByName = func() map[string]augmentedAIOperation {
	out := make(map[string]augmentedAIOperation, len(augmentedAIOperations))
	for _, op := range augmentedAIOperations {
		out[op.Name] = op
	}
	return out
}()
