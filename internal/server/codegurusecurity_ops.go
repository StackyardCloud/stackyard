package server

type cgsOperation struct{ Name, Method, URI string }

// Amazon CodeGuru Security operations, from the AWS CLI 2.42.1 model
// (botocore/data/codeguru-security/2018-05-10/service-2.json). The API
// reference at docs.aws.amazon.com was retired with the service's end of
// support; the model is mirrored at
// https://github.com/boto/botocore/blob/develop/botocore/data/codeguru-security/2018-05-10/service-2.json
var cgsOperations = []cgsOperation{
	{Name: "BatchGetFindings", Method: "POST", URI: "/batchGetFindings"},
	{Name: "CreateScan", Method: "POST", URI: "/scans"},
	{Name: "CreateUploadUrl", Method: "POST", URI: "/uploadUrl"},
	{Name: "GetAccountConfiguration", Method: "GET", URI: "/accountConfiguration/get"},
	{Name: "GetFindings", Method: "GET", URI: "/findings/{scanName}"},
	{Name: "GetMetricsSummary", Method: "GET", URI: "/metrics/summary"},
	{Name: "GetScan", Method: "GET", URI: "/scans/{scanName}"},
	{Name: "ListFindingsMetrics", Method: "GET", URI: "/metrics/findings"},
	{Name: "ListScans", Method: "GET", URI: "/scans"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateAccountConfiguration", Method: "PUT", URI: "/updateAccountConfiguration"},
}
