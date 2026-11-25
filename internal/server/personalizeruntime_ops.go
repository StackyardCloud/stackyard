package server

type personalizeRuntimeOperation struct{ Name string }

// Amazon Personalize Runtime operations sourced from:
// https://docs.aws.amazon.com/personalize/latest/dg/API_Operations_Amazon_Personalize_Runtime.html
var personalizeRuntimeOperations = []personalizeRuntimeOperation{
	{Name: "GetActionRecommendations"},
	{Name: "GetPersonalizedRanking"},
	{Name: "GetRecommendations"},
}

var personalizeRuntimeOperationByName = func() map[string]personalizeRuntimeOperation {
	out := make(map[string]personalizeRuntimeOperation, len(personalizeRuntimeOperations))
	for _, op := range personalizeRuntimeOperations {
		out[op.Name] = op
	}
	return out
}()
