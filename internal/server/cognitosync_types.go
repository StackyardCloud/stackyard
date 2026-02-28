package server

type cognitoSyncDataType struct {
	Name string
}

// Amazon Cognito Sync data types sourced from:
// https://docs.aws.amazon.com/cognitosync/latest/APIReference/API_Types.html
var cognitoSyncDataTypes = []cognitoSyncDataType{
	{Name: "CognitoStreams"},
	{Name: "Dataset"},
	{Name: "IdentityPoolUsage"},
	{Name: "IdentityUsage"},
	{Name: "PushSync"},
	{Name: "Record"},
	{Name: "RecordPatch"},
}

var cognitoSyncDataTypeByName = func() map[string]cognitoSyncDataType {
	out := make(map[string]cognitoSyncDataType, len(cognitoSyncDataTypes))
	for _, dt := range cognitoSyncDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
