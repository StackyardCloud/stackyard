package server

type cloudfrontkeyvaluestoreOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon CloudFront KeyValueStore operations sourced from:
// https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_Operations_Amazon_CloudFront_KeyValueStore.html
var cloudfrontkeyvaluestoreOperations = []cloudfrontkeyvaluestoreOperation{
	{Name: "DeleteKey", Method: "DELETE", URI: "/key-value-stores/{KvsARN}/keys/{Key}"},
	{Name: "DescribeKeyValueStore", Method: "GET", URI: "/key-value-stores/{KvsARN}"},
	{Name: "GetKey", Method: "GET", URI: "/key-value-stores/{KvsARN}/keys/{Key}"},
	{Name: "ListKeys", Method: "GET", URI: "/key-value-stores/{KvsARN}/keys"},
	{Name: "PutKey", Method: "PUT", URI: "/key-value-stores/{KvsARN}/keys/{Key}"},
	{Name: "UpdateKeys", Method: "POST", URI: "/key-value-stores/{KvsARN}/keys"},
}

var cloudfrontkeyvaluestoreOperationByName = func() map[string]cloudfrontkeyvaluestoreOperation {
	out := make(map[string]cloudfrontkeyvaluestoreOperation, len(cloudfrontkeyvaluestoreOperations))
	for _, op := range cloudfrontkeyvaluestoreOperations {
		out[op.Name] = op
	}
	return out
}()
