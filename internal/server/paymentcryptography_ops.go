package server

type paymentCryptographyOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Payment Cryptography Control Plane operations sourced from:
// https://docs.aws.amazon.com/payment-cryptography/latest/APIReference/API_Operations.html
var paymentCryptographyOperations = []paymentCryptographyOperation{
	{Name: "CreateAlias", Method: "POST", URI: "/"},
	{Name: "CreateKey", Method: "POST", URI: "/"},
	{Name: "DeleteAlias", Method: "POST", URI: "/"},
	{Name: "DeleteKey", Method: "POST", URI: "/"},
	{Name: "ExportKey", Method: "POST", URI: "/"},
	{Name: "GetAlias", Method: "POST", URI: "/"},
	{Name: "GetKey", Method: "POST", URI: "/"},
	{Name: "GetParametersForExport", Method: "POST", URI: "/"},
	{Name: "GetParametersForImport", Method: "POST", URI: "/"},
	{Name: "GetPublicKeyCertificate", Method: "POST", URI: "/"},
	{Name: "ImportKey", Method: "POST", URI: "/"},
	{Name: "ListAliases", Method: "POST", URI: "/"},
	{Name: "ListKeys", Method: "POST", URI: "/"},
	{Name: "ListTagsForResource", Method: "POST", URI: "/"},
	{Name: "RestoreKey", Method: "POST", URI: "/"},
	{Name: "StartKeyUsage", Method: "POST", URI: "/"},
	{Name: "StopKeyUsage", Method: "POST", URI: "/"},
	{Name: "TagResource", Method: "POST", URI: "/"},
	{Name: "UntagResource", Method: "POST", URI: "/"},
	{Name: "UpdateAlias", Method: "POST", URI: "/"},
}

var paymentCryptographyOperationByName = func() map[string]paymentCryptographyOperation {
	out := make(map[string]paymentCryptographyOperation, len(paymentCryptographyOperations))
	for _, op := range paymentCryptographyOperations {
		out[op.Name] = op
	}
	return out
}()
