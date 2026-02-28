package server

type managedServicesCMOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Managed Services Change Management actions sourced from:
// https://docs.aws.amazon.com/managedservices/latest/ApiReference-cm/API_Operations.html
var managedServicesCMOperations = []managedServicesCMOperation{
	{Name: "ApproveRfc", Method: "POST", URI: "/"},
	{Name: "CancelRfc", Method: "POST", URI: "/"},
	{Name: "CreateRfc", Method: "POST", URI: "/"},
	{Name: "CreateRfcAttachment", Method: "POST", URI: "/"},
	{Name: "CreateRfcCorrespondence", Method: "POST", URI: "/"},
	{Name: "GetChangeTypeVersion", Method: "POST", URI: "/"},
	{Name: "GetRfc", Method: "POST", URI: "/"},
	{Name: "GetRfcAttachment", Method: "POST", URI: "/"},
	{Name: "ListChangeTypeCategories", Method: "POST", URI: "/"},
	{Name: "ListChangeTypeClassificationSummaries", Method: "POST", URI: "/"},
	{Name: "ListChangeTypeItems", Method: "POST", URI: "/"},
	{Name: "ListChangeTypeOperations", Method: "POST", URI: "/"},
	{Name: "ListChangeTypeSubcategories", Method: "POST", URI: "/"},
	{Name: "ListChangeTypeVersionSummaries", Method: "POST", URI: "/"},
	{Name: "ListRestrictedExecutionTimes", Method: "POST", URI: "/"},
	{Name: "ListRfcAttachmentSummaries", Method: "POST", URI: "/"},
	{Name: "ListRfcCorrespondences", Method: "POST", URI: "/"},
	{Name: "ListRfcSummaries", Method: "POST", URI: "/"},
	{Name: "RejectRfc", Method: "POST", URI: "/"},
	{Name: "SubmitRfc", Method: "POST", URI: "/"},
	{Name: "UpdateRestrictedExecutionTimes", Method: "POST", URI: "/"},
	{Name: "UpdateRfc", Method: "POST", URI: "/"},
}

var managedServicesCMOperationByName = func() map[string]managedServicesCMOperation {
	out := make(map[string]managedServicesCMOperation, len(managedServicesCMOperations))
	for _, op := range managedServicesCMOperations {
		out[op.Name] = op
	}
	return out
}()
