package server

type cognitoIdentityOperation struct {
	Name string
}

// Amazon Cognito Federated Identities operations sourced from:
// https://docs.aws.amazon.com/cognitoidentity/latest/APIReference/API_Operations.html
var cognitoIdentityOperations = []cognitoIdentityOperation{
	{Name: "CreateIdentityPool"},
	{Name: "DeleteIdentities"},
	{Name: "DeleteIdentityPool"},
	{Name: "DescribeIdentity"},
	{Name: "DescribeIdentityPool"},
	{Name: "GetCredentialsForIdentity"},
	{Name: "GetId"},
	{Name: "GetIdentityPoolRoles"},
	{Name: "GetOpenIdToken"},
	{Name: "GetOpenIdTokenForDeveloperIdentity"},
	{Name: "GetPrincipalTagAttributeMap"},
	{Name: "ListIdentities"},
	{Name: "ListIdentityPools"},
	{Name: "ListTagsForResource"},
	{Name: "LookupDeveloperIdentity"},
	{Name: "MergeDeveloperIdentities"},
	{Name: "SetIdentityPoolRoles"},
	{Name: "SetPrincipalTagAttributeMap"},
	{Name: "TagResource"},
	{Name: "UnlinkDeveloperIdentity"},
	{Name: "UnlinkIdentity"},
	{Name: "UntagResource"},
	{Name: "UpdateIdentityPool"},
}

var cognitoIdentityOperationByName = func() map[string]cognitoIdentityOperation {
	out := make(map[string]cognitoIdentityOperation, len(cognitoIdentityOperations))
	for _, op := range cognitoIdentityOperations {
		out[op.Name] = op
	}
	return out
}()
