package server

type verifiedPermissionsOperation struct{ Name string }

// Amazon Verified Permissions operations sourced from:
// https://docs.aws.amazon.com/verifiedpermissions/latest/apireference/API_Operations.html
var verifiedPermissionsOperations = []verifiedPermissionsOperation{
	{Name: "BatchGetPolicy"},
	{Name: "BatchIsAuthorized"},
	{Name: "BatchIsAuthorizedWithToken"},
	{Name: "CreateIdentitySource"},
	{Name: "CreatePolicy"},
	{Name: "CreatePolicyStore"},
	{Name: "CreatePolicyStoreAlias"},
	{Name: "CreatePolicyTemplate"},
	{Name: "DeleteIdentitySource"},
	{Name: "DeletePolicy"},
	{Name: "DeletePolicyStore"},
	{Name: "DeletePolicyStoreAlias"},
	{Name: "DeletePolicyTemplate"},
	{Name: "GetIdentitySource"},
	{Name: "GetPolicy"},
	{Name: "GetPolicyStore"},
	{Name: "GetPolicyStoreAlias"},
	{Name: "GetPolicyTemplate"},
	{Name: "GetSchema"},
	{Name: "IsAuthorized"},
	{Name: "IsAuthorizedWithToken"},
	{Name: "ListIdentitySources"},
	{Name: "ListPolicies"},
	{Name: "ListPolicyStoreAliases"},
	{Name: "ListPolicyStores"},
	{Name: "ListPolicyTemplates"},
	{Name: "ListTagsForResource"},
	{Name: "PutSchema"},
	{Name: "TagResource"},
	{Name: "UntagResource"},
	{Name: "UpdateIdentitySource"},
	{Name: "UpdatePolicy"},
	{Name: "UpdatePolicyStore"},
	{Name: "UpdatePolicyTemplate"},
}
