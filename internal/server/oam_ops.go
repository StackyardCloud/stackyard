package server

type oamOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CloudWatch Observability Access Manager operations sourced from:
// https://docs.aws.amazon.com/OAM/latest/APIReference/API_Operations.html
var oamOperations = []oamOperation{
	{Name: "CreateLink", Method: "POST", URI: "/CreateLink"},
	{Name: "CreateSink", Method: "POST", URI: "/CreateSink"},
	{Name: "DeleteLink", Method: "POST", URI: "/DeleteLink"},
	{Name: "DeleteSink", Method: "POST", URI: "/DeleteSink"},
	{Name: "GetLink", Method: "POST", URI: "/GetLink"},
	{Name: "GetSink", Method: "POST", URI: "/GetSink"},
	{Name: "GetSinkPolicy", Method: "POST", URI: "/GetSinkPolicy"},
	{Name: "ListAttachedLinks", Method: "POST", URI: "/ListAttachedLinks"},
	{Name: "ListLinks", Method: "POST", URI: "/ListLinks"},
	{Name: "ListSinks", Method: "POST", URI: "/ListSinks"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "PutSinkPolicy", Method: "POST", URI: "/PutSinkPolicy"},
	{Name: "TagResource", Method: "PUT", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateLink", Method: "POST", URI: "/UpdateLink"},
}

var oamOperationByName = func() map[string]oamOperation {
	out := make(map[string]oamOperation, len(oamOperations))
	for _, op := range oamOperations {
		out[op.Name] = op
	}
	return out
}()
