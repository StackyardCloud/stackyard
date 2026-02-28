package server

type mpaOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Multi-party Approval operations sourced from:
// https://docs.aws.amazon.com/mpa/latest/APIReference/API_Operations.html
var mpaOperations = []mpaOperation{
	{Name: "CancelSession", Method: "PUT", URI: "/sessions/{SessionArn}"},
	{Name: "CreateApprovalTeam", Method: "POST", URI: "/approval-teams"},
	{Name: "CreateIdentitySource", Method: "POST", URI: "/identity-sources"},
	{Name: "DeleteIdentitySource", Method: "DELETE", URI: "/identity-sources/{IdentitySourceArn}"},
	{Name: "DeleteInactiveApprovalTeamVersion", Method: "DELETE", URI: "/approval-teams/{Arn}/{VersionId}"},
	{Name: "GetApprovalTeam", Method: "GET", URI: "/approval-teams/{Arn}"},
	{Name: "GetIdentitySource", Method: "GET", URI: "/identity-sources/{IdentitySourceArn}"},
	{Name: "GetPolicyVersion", Method: "GET", URI: "/policy-versions/{PolicyVersionArn}"},
	{Name: "GetResourcePolicy", Method: "POST", URI: "/GetResourcePolicy"},
	{Name: "GetSession", Method: "GET", URI: "/sessions/{SessionArn}"},
	{Name: "ListApprovalTeams", Method: "POST", URI: "/approval-teams/"},
	{Name: "ListIdentitySources", Method: "POST", URI: "/identity-sources/"},
	{Name: "ListPolicies", Method: "POST", URI: "/policies/"},
	{Name: "ListPolicyVersions", Method: "POST", URI: "/policies/{PolicyArn}/"},
	{Name: "ListResourcePolicies", Method: "POST", URI: "/resource-policies/{ResourceArn}/"},
	{Name: "ListSessions", Method: "POST", URI: "/approval-teams/{ApprovalTeamArn}/sessions/"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "StartActiveApprovalTeamDeletion", Method: "POST", URI: "/approval-teams/{Arn}"},
	{Name: "TagResource", Method: "PUT", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateApprovalTeam", Method: "PATCH", URI: "/approval-teams/{Arn}"},
}

var mpaOperationByName = func() map[string]mpaOperation {
	out := make(map[string]mpaOperation, len(mpaOperations))
	for _, op := range mpaOperations {
		out[op.Name] = op
	}
	return out
}()
