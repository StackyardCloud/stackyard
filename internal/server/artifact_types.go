package server

type artifactDataType struct {
	Name string
}

// AWS Artifact data types sourced from:
// https://docs.aws.amazon.com/artifact/latest/APIReference/API_Types.html
var artifactDataTypes = []artifactDataType{
	{Name: "AccountSettings"},
	{Name: "AgreementSummary"},
	{Name: "CustomerAgreementSummary"},
	{Name: "ReportDetail"},
	{Name: "ReportSummary"},
	{Name: "TerminateCustomerAgreementSummary"},
	{Name: "ValidationExceptionField"},
}

var artifactDataTypeByName = func() map[string]artifactDataType {
	out := make(map[string]artifactDataType, len(artifactDataTypes))
	for _, dt := range artifactDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
