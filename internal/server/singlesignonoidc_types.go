package server

type singleSignOnOIDCDataType struct {
	Name string
}

// AWS IAM Identity Center OIDC (sso-oidc) data types sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/OIDCAPIReference/API_Types.html
var singleSignOnOIDCDataTypes = []singleSignOnOIDCDataType{
	{Name: "AwsAdditionalDetails"},
}

var singleSignOnOIDCDataTypeByName = func() map[string]singleSignOnOIDCDataType {
	out := make(map[string]singleSignOnOIDCDataType, len(singleSignOnOIDCDataTypes))
	for _, dt := range singleSignOnOIDCDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
