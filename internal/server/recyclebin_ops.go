package server

type recycleBinOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Recycle Bin actions sourced from:
// https://docs.aws.amazon.com/recyclebin/latest/APIReference/API_Operations.html
var recycleBinOperations = []recycleBinOperation{
	{Name: "CreateRule", Method: "POST", URI: "/rules"},
	{Name: "DeleteRule", Method: "DELETE", URI: "/rules/{identifier}"},
	{Name: "GetRule", Method: "GET", URI: "/rules/{identifier}"},
	{Name: "ListRules", Method: "POST", URI: "/list-rules"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "LockRule", Method: "PATCH", URI: "/rules/{identifier}/lock"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UnlockRule", Method: "PATCH", URI: "/rules/{identifier}/unlock"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateRule", Method: "PATCH", URI: "/rules/{identifier}"},
}

var recycleBinOperationByName = func() map[string]recycleBinOperation {
	out := make(map[string]recycleBinOperation, len(recycleBinOperations))
	for _, op := range recycleBinOperations {
		out[op.Name] = op
	}
	return out
}()
