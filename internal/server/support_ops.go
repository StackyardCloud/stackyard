package server

type supportOperation struct {
	Name string
}

// AWS Support operations sourced from:
// https://docs.aws.amazon.com/awssupport/latest/APIReference/API_Operations.html
var supportOperations = []supportOperation{
	{Name: "AddAttachmentsToSet"},
	{Name: "AddCommunicationToCase"},
	{Name: "CreateCase"},
	{Name: "DescribeAttachment"},
	{Name: "DescribeCases"},
	{Name: "DescribeCommunications"},
	{Name: "DescribeCreateCaseOptions"},
	{Name: "DescribeIssueTypes"},
	{Name: "DescribeServices"},
	{Name: "DescribeSeverityLevels"},
	{Name: "DescribeSupportedLanguages"},
	{Name: "DescribeTrustedAdvisorCheckRefreshStatuses"},
	{Name: "DescribeTrustedAdvisorCheckResult"},
	{Name: "DescribeTrustedAdvisorCheckSummaries"},
	{Name: "RefreshTrustedAdvisorCheck"},
	{Name: "ResolveCase"},
}

var supportOperationByName = func() map[string]supportOperation {
	out := make(map[string]supportOperation, len(supportOperations))
	for _, op := range supportOperations {
		out[op.Name] = op
	}
	return out
}()
