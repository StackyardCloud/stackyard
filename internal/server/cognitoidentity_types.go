package server

type cognitoIdentityDataType struct {
	Name string
}

// Amazon Cognito Federated Identities data types sourced from:
// https://docs.aws.amazon.com/cognitoidentity/latest/APIReference/API_Types.html
var cognitoIdentityDataTypes = []cognitoIdentityDataType{
	{Name: "CognitoIdentityProvider"},
	{Name: "Credentials"},
	{Name: "IdentityDescription"},
	{Name: "IdentityPoolShortDescription"},
	{Name: "MappingRule"},
	{Name: "RoleMapping"},
	{Name: "RulesConfigurationType"},
	{Name: "UnprocessedIdentityId"},
}

var cognitoIdentityDataTypeByName = func() map[string]cognitoIdentityDataType {
	out := make(map[string]cognitoIdentityDataType, len(cognitoIdentityDataTypes))
	for _, dt := range cognitoIdentityDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
