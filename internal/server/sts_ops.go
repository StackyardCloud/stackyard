package server

type stsOperation struct {
	Name string
}

// AWS Security Token Service (STS) operations sourced from:
// https://docs.aws.amazon.com/STS/latest/APIReference/API_Operations.html
var stsOperations = []stsOperation{
	{Name: "AssumeRole"},
	{Name: "AssumeRoleWithSAML"},
	{Name: "AssumeRoleWithWebIdentity"},
	{Name: "AssumeRoot"},
	{Name: "DecodeAuthorizationMessage"},
	{Name: "GetAccessKeyInfo"},
	{Name: "GetCallerIdentity"},
	{Name: "GetDelegatedAccessToken"},
	{Name: "GetFederationToken"},
	{Name: "GetSessionToken"},
	{Name: "GetWebIdentityToken"},
	{Name: "Operations"},
}

var stsOperationByName = func() map[string]stsOperation {
	out := make(map[string]stsOperation, len(stsOperations))
	for _, op := range stsOperations {
		out[op.Name] = op
	}
	return out
}()
