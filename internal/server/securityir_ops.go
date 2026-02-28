package server

type securityIROperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Security Incident Response operations sourced from:
// https://docs.aws.amazon.com/security-ir/latest/APIReference/API_Operations.html
var securityIROperations = []securityIROperation{
	{Name: "BatchGetMemberAccountDetails", Method: "POST", URI: "/v1/membership/{membershipId}/batch-member-details"},
	{Name: "CancelMembership", Method: "PUT", URI: "/v1/membership/{membershipId}"},
	{Name: "CloseCase", Method: "POST", URI: "/v1/cases/{caseId}/close-case"},
	{Name: "CreateCase", Method: "POST", URI: "/v1/create-case"},
	{Name: "CreateCaseComment", Method: "POST", URI: "/v1/cases/{caseId}/create-comment"},
	{Name: "CreateMembership", Method: "POST", URI: "/v1/membership"},
	{Name: "GetCase", Method: "GET", URI: "/v1/cases/{caseId}/get-case"},
	{Name: "GetCaseAttachmentDownloadUrl", Method: "GET", URI: "/v1/cases/{caseId}/get-presigned-url/{attachmentId}"},
	{Name: "GetCaseAttachmentUploadUrl", Method: "POST", URI: "/v1/cases/{caseId}/get-presigned-url"},
	{Name: "GetMembership", Method: "GET", URI: "/v1/membership/{membershipId}"},
	{Name: "ListCaseEdits", Method: "POST", URI: "/v1/cases/{caseId}/list-case-edits"},
	{Name: "ListCases", Method: "POST", URI: "/v1/list-cases"},
	{Name: "ListComments", Method: "POST", URI: "/v1/cases/{caseId}/list-comments"},
	{Name: "ListInvestigations", Method: "GET", URI: "/v1/cases/{caseId}/list-investigations"},
	{Name: "ListMemberships", Method: "POST", URI: "/v1/memberships"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/v1/tags/{resourceArn}"},
	{Name: "SendFeedback", Method: "POST", URI: "/v1/cases/{caseId}/feedback/{resultId}/send-feedback"},
	{Name: "TagResource", Method: "POST", URI: "/v1/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/v1/tags/{resourceArn}"},
	{Name: "UpdateCase", Method: "POST", URI: "/v1/cases/{caseId}/update-case"},
	{Name: "UpdateCaseComment", Method: "PUT", URI: "/v1/cases/{caseId}/update-case-comment/{commentId}"},
	{Name: "UpdateCaseStatus", Method: "POST", URI: "/v1/cases/{caseId}/update-case-status"},
	{Name: "UpdateMembership", Method: "PUT", URI: "/v1/membership/{membershipId}/update-membership"},
	{Name: "UpdateResolverType", Method: "POST", URI: "/v1/cases/{caseId}/update-resolver-type"},
}

var securityIROperationByName = func() map[string]securityIROperation {
	out := make(map[string]securityIROperation, len(securityIROperations))
	for _, op := range securityIROperations {
		out[op.Name] = op
	}
	return out
}()
