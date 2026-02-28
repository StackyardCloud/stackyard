package server

type grafanaOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon Managed Grafana operations sourced from:
// https://docs.aws.amazon.com/grafana/latest/APIReference/API_Operations.html
var grafanaOperations = []grafanaOperation{
	{Name: "AssociateLicense", Method: "POST", URI: "/workspaces/{workspaceId}/licenses/{licenseType}"},
	{Name: "CreateWorkspace", Method: "POST", URI: "/workspaces"},
	{Name: "CreateWorkspaceApiKey", Method: "POST", URI: "/workspaces/{workspaceId}/apikeys"},
	{Name: "CreateWorkspaceServiceAccount", Method: "POST", URI: "/workspaces/{workspaceId}/serviceaccounts"},
	{Name: "CreateWorkspaceServiceAccountToken", Method: "POST", URI: "/workspaces/{workspaceId}/serviceaccounts/{serviceAccountId}/tokens"},
	{Name: "DeleteWorkspace", Method: "DELETE", URI: "/workspaces/{workspaceId}"},
	{Name: "DeleteWorkspaceApiKey", Method: "DELETE", URI: "/workspaces/{workspaceId}/apikeys/{keyName}"},
	{Name: "DeleteWorkspaceServiceAccount", Method: "DELETE", URI: "/workspaces/{workspaceId}/serviceaccounts/{serviceAccountId}"},
	{Name: "DeleteWorkspaceServiceAccountToken", Method: "DELETE", URI: "/workspaces/{workspaceId}/serviceaccounts/{serviceAccountId}/tokens/{tokenId}"},
	{Name: "DescribeWorkspace", Method: "GET", URI: "/workspaces/{workspaceId}"},
	{Name: "DescribeWorkspaceAuthentication", Method: "GET", URI: "/workspaces/{workspaceId}/authentication"},
	{Name: "DescribeWorkspaceConfiguration", Method: "GET", URI: "/workspaces/{workspaceId}/configuration"},
	{Name: "DisassociateLicense", Method: "DELETE", URI: "/workspaces/{workspaceId}/licenses/{licenseType}"},
	{Name: "ListPermissions", Method: "GET", URI: "/workspaces/{workspaceId}/permissions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "ListVersions", Method: "GET", URI: "/versions"},
	{Name: "ListWorkspaceServiceAccounts", Method: "GET", URI: "/workspaces/{workspaceId}/serviceaccounts"},
	{Name: "ListWorkspaceServiceAccountTokens", Method: "GET", URI: "/workspaces/{workspaceId}/serviceaccounts/{serviceAccountId}/tokens"},
	{Name: "ListWorkspaces", Method: "GET", URI: "/workspaces"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdatePermissions", Method: "PATCH", URI: "/workspaces/{workspaceId}/permissions"},
	{Name: "UpdateWorkspace", Method: "PUT", URI: "/workspaces/{workspaceId}"},
	{Name: "UpdateWorkspaceAuthentication", Method: "POST", URI: "/workspaces/{workspaceId}/authentication"},
	{Name: "UpdateWorkspaceConfiguration", Method: "PUT", URI: "/workspaces/{workspaceId}/configuration"},
}

var grafanaOperationByName = func() map[string]grafanaOperation {
	out := make(map[string]grafanaOperation, len(grafanaOperations))
	for _, op := range grafanaOperations {
		out[op.Name] = op
	}
	return out
}()
