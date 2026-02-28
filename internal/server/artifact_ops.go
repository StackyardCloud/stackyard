package server

type artifactOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Artifact actions sourced from:
// https://docs.aws.amazon.com/artifact/latest/APIReference/API_Operations.html
var artifactOperations = []artifactOperation{
	{Name: "AcceptAgreement", Method: "POST", URI: "/v1/agreement/accept"},
	{Name: "AcceptNdaForAgreement", Method: "POST", URI: "/v1/agreement/acceptNdaForAgreement"},
	{Name: "GetAccountSettings", Method: "GET", URI: "/v1/account-settings/get"},
	{Name: "GetAgreement", Method: "GET", URI: "/v1/agreement/get"},
	{Name: "GetCustomerAgreement", Method: "GET", URI: "/v1/customer-agreement/get"},
	{Name: "GetNdaForAgreement", Method: "GET", URI: "/v1/agreement/getNdaForAgreement"},
	{Name: "GetReport", Method: "GET", URI: "/v1/report/get"},
	{Name: "GetReportMetadata", Method: "GET", URI: "/v1/report/getMetadata"},
	{Name: "GetTermForReport", Method: "GET", URI: "/v1/report/getTermForReport"},
	{Name: "ListAgreements", Method: "GET", URI: "/v1/agreement/list"},
	{Name: "ListCustomerAgreements", Method: "GET", URI: "/v1/customer-agreement/list"},
	{Name: "ListReports", Method: "GET", URI: "/v1/report/list"},
	{Name: "ListReportVersions", Method: "GET", URI: "/v1/report/listVersions"},
	{Name: "PutAccountSettings", Method: "PUT", URI: "/v1/account-settings/put"},
	{Name: "TerminateAgreement", Method: "POST", URI: "/v1/customer-agreement/terminate"},
}

var artifactOperationByName = func() map[string]artifactOperation {
	out := make(map[string]artifactOperation, len(artifactOperations))
	for _, op := range artifactOperations {
		out[op.Name] = op
	}
	return out
}()
