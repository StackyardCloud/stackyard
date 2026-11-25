package server

type mstdOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Elemental MediaStore Data Plane operations sourced from:
// https://github.com/aws/aws-sdk-go-v2/tree/main/service/mediastoredata (the MediaStore
// API reference is no longer published on docs.aws.amazon.com)
var mstdOperations = []mstdOperation{
	{Name: "DeleteObject", Method: "DELETE", URI: "/{Path+}"},
	{Name: "DescribeObject", Method: "HEAD", URI: "/{Path+}"},
	{Name: "GetObject", Method: "GET", URI: "/{Path+}"},
	{Name: "ListItems", Method: "GET", URI: "/"},
	{Name: "PutObject", Method: "PUT", URI: "/{Path+}"},
}

var mstdOperationByName = func() map[string]mstdOperation {
	out := make(map[string]mstdOperation, len(mstdOperations))
	for _, op := range mstdOperations {
		out[op.Name] = op
	}
	return out
}()
