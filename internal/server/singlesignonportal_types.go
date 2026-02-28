package server

type singleSignOnPortalDataType struct {
	Name string
}

// AWS IAM Identity Center Portal (sso) data types sourced from:
// https://docs.aws.amazon.com/singlesignon/latest/PortalAPIReference/API_Types.html
var singleSignOnPortalDataTypes = []singleSignOnPortalDataType{
	{Name: "AccountInfo"},
	{Name: "RoleCredentials"},
	{Name: "RoleInfo"},
}

var singleSignOnPortalDataTypeByName = func() map[string]singleSignOnPortalDataType {
	out := make(map[string]singleSignOnPortalDataType, len(singleSignOnPortalDataTypes))
	for _, dt := range singleSignOnPortalDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
