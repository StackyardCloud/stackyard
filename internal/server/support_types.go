package server

type supportDataType struct {
	Name string
}

// AWS Support data types sourced from:
// https://docs.aws.amazon.com/awssupport/latest/APIReference/API_Types.html
var supportDataTypes = []supportDataType{
	{Name: "Attachment"},
	{Name: "AttachmentDetails"},
	{Name: "AttachmentIdNotFound"},
	{Name: "AttachmentLimitExceeded"},
	{Name: "AttachmentSetExpired"},
	{Name: "AttachmentSetIdNotFound"},
	{Name: "CaseDetails"},
	{Name: "Category"},
	{Name: "Communication"},
	{Name: "CreateCaseRequest"},
	{Name: "CreateCaseResponse"},
	{Name: "DescribeAttachmentRequest"},
	{Name: "DescribeAttachmentResponse"},
	{Name: "DescribeCasesRequest"},
	{Name: "DescribeCasesResponse"},
	{Name: "DescribeCommunicationsRequest"},
	{Name: "DescribeCommunicationsResponse"},
	{Name: "DescribeTrustedAdvisorCheckResultResponse"},
	{Name: "RecentCaseCommunications"},
	{Name: "Service"},
	{Name: "SeverityLevel"},
}

var supportDataTypeByName = func() map[string]supportDataType {
	out := make(map[string]supportDataType, len(supportDataTypes))
	for _, dt := range supportDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
