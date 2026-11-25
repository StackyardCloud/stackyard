package server

type mstOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental MediaStore (control plane) operations sourced from:
// https://github.com/aws/aws-sdk-go-v2/tree/main/service/mediastore (the MediaStore
// API reference is no longer published on docs.aws.amazon.com)
var mstOperations = []mstOperation{
	{Name: "CreateContainer", Method: "POST", URI: "/"},
	{Name: "DeleteContainer", Method: "POST", URI: "/"},
	{Name: "DeleteContainerPolicy", Method: "POST", URI: "/"},
	{Name: "DeleteCorsPolicy", Method: "POST", URI: "/"},
	{Name: "DeleteLifecyclePolicy", Method: "POST", URI: "/"},
	{Name: "DeleteMetricPolicy", Method: "POST", URI: "/"},
	{Name: "DescribeContainer", Method: "POST", URI: "/"},
	{Name: "GetContainerPolicy", Method: "POST", URI: "/"},
	{Name: "GetCorsPolicy", Method: "POST", URI: "/"},
	{Name: "GetLifecyclePolicy", Method: "POST", URI: "/"},
	{Name: "GetMetricPolicy", Method: "POST", URI: "/"},
	{Name: "ListContainers", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "PutContainerPolicy", Method: "POST", URI: "/"},
	{Name: "PutCorsPolicy", Method: "POST", URI: "/"},
	{Name: "PutLifecyclePolicy", Method: "POST", URI: "/"},
	{Name: "PutMetricPolicy", Method: "POST", URI: "/"},
	{Name: "StartAccessLogging", Method: "POST", URI: "/"},
	{Name: "StopAccessLogging", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
}

var mstOperationByName = func() map[string]mstOperation {
	out := make(map[string]mstOperation, len(mstOperations))
	for _, op := range mstOperations {
		out[op.Name] = op
	}
	return out
}()
