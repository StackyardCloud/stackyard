package server

type elementalInferenceOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental Inference actions sourced from:
// https://docs.aws.amazon.com/elemental-inference/latest/APIReference/API_Operations.html
var elementalInferenceOperations = []elementalInferenceOperation{
	{Name: "AssociateFeed", Method: "POST", URI: "/v1/feed/{id}"},
	{Name: "CreateFeed", Method: "POST", URI: "/v1/feed"},
	{Name: "DeleteFeed", Method: "DELETE", URI: "/v1/feed/{id}"},
	{Name: "DisassociateFeed", Method: "POST", URI: "/v1/feed/{id}"},
	{Name: "GetFeed", Method: "GET", URI: "/v1/feed/{id}"},
	{Name: "ListFeeds", Method: "GET", URI: "/v1/feeds"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/v1/tags/{resourceArn}"},
	{Name: "UpdateFeed", Method: "PUT", URI: "/v1/feed/{id}"},
}

var elementalInferenceOperationByName = func() map[string]elementalInferenceOperation {
	out := make(map[string]elementalInferenceOperation, len(elementalInferenceOperations))
	for _, op := range elementalInferenceOperations {
		out[op.Name] = op
	}
	return out
}()
