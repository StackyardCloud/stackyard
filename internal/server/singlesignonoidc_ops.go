package server

type singleSignOnOIDCOperation struct {
	Name string
}

// AWS IAM Identity Center OIDC (sso-oidc) operations sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/API_Operations.html
var singleSignOnOIDCOperations = []singleSignOnOIDCOperation{
	{Name: "CreateToken"},
	{Name: "CreateTokenWithIAM"},
	{Name: "RegisterClient"},
	{Name: "StartDeviceAuthorization"},
}

var singleSignOnOIDCOperationByName = func() map[string]singleSignOnOIDCOperation {
	out := make(map[string]singleSignOnOIDCOperation, len(singleSignOnOIDCOperations))
	for _, op := range singleSignOnOIDCOperations {
		out[op.Name] = op
	}
	return out
}()
