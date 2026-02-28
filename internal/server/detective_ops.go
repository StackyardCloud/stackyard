package server

type detectiveOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Detective operations sourced from:
// https://docs.aws.amazon.com/detective/latest/APIReference/API_Operations.html
var detectiveOperations = []detectiveOperation{
	{Name: "AcceptInvitation", Method: "PUT", URI: "/invitation"},
	{Name: "BatchGetGraphMemberDatasources", Method: "POST", URI: "/graph/datasources/get"},
	{Name: "BatchGetMembershipDatasources", Method: "POST", URI: "/membership/datasources/get"},
	{Name: "CreateGraph", Method: "POST", URI: "/graph"},
	{Name: "CreateMembers", Method: "POST", URI: "/graph/members"},
	{Name: "DeleteGraph", Method: "POST", URI: "/graph/removal"},
	{Name: "DeleteMembers", Method: "POST", URI: "/graph/members/removal"},
	{Name: "DescribeOrganizationConfiguration", Method: "POST", URI: "/orgs/describeOrganizationConfiguration"},
	{Name: "DisableOrganizationAdminAccount", Method: "POST", URI: "/orgs/disableAdminAccount"},
	{Name: "DisassociateMembership", Method: "POST", URI: "/membership/removal"},
	{Name: "EnableOrganizationAdminAccount", Method: "POST", URI: "/orgs/enableAdminAccount"},
	{Name: "GetInvestigation", Method: "POST", URI: "/investigations/getInvestigation"},
	{Name: "GetMembers", Method: "POST", URI: "/graph/members/get"},
	{Name: "ListDatasourcePackages", Method: "POST", URI: "/graph/datasources/list"},
	{Name: "ListGraphs", Method: "POST", URI: "/graphs/list"},
	{Name: "ListIndicators", Method: "POST", URI: "/investigations/listIndicators"},
	{Name: "ListInvestigations", Method: "POST", URI: "/investigations/listInvestigations"},
	{Name: "ListInvitations", Method: "POST", URI: "/invitations/list"},
	{Name: "ListMembers", Method: "POST", URI: "/graph/members/list"},
	{Name: "ListOrganizationAdminAccounts", Method: "POST", URI: "/orgs/adminAccountslist"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{ResourceArn}"},
	{Name: "RejectInvitation", Method: "POST", URI: "/invitation/removal"},
	{Name: "StartInvestigation", Method: "POST", URI: "/investigations/startInvestigation"},
	{Name: "StartMonitoringMember", Method: "POST", URI: "/graph/member/monitoringstate"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{ResourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{ResourceArn}"},
	{Name: "UpdateDatasourcePackages", Method: "POST", URI: "/graph/datasources/update"},
	{Name: "UpdateInvestigationState", Method: "POST", URI: "/investigations/updateInvestigationState"},
	{Name: "UpdateOrganizationConfiguration", Method: "POST", URI: "/orgs/updateOrganizationConfiguration"},
}

var detectiveOperationByName = func() map[string]detectiveOperation {
	out := make(map[string]detectiveOperation, len(detectiveOperations))
	for _, op := range detectiveOperations {
		out[op.Name] = op
	}
	return out
}()
