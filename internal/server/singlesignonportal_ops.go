package server

type singleSignOnPortalOperation struct {
	Name string
}

// AWS IAM Identity Center Portal (sso) operations sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/PortalAPIReference/API_Operations.html
var singleSignOnPortalOperations = []singleSignOnPortalOperation{
	{Name: "GetRoleCredentials"},
	{Name: "ListAccountRoles"},
	{Name: "ListAccounts"},
	{Name: "Logout"},
}

var singleSignOnPortalOperationByName = func() map[string]singleSignOnPortalOperation {
	out := make(map[string]singleSignOnPortalOperation, len(singleSignOnPortalOperations))
	for _, op := range singleSignOnPortalOperations {
		out[op.Name] = op
	}
	return out
}()
