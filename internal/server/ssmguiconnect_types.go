package server

type ssmGUIConnectDataType struct {
	Name string
}

// AWS Systems Manager GUI Connect data types sourced from:
// https://docs.aws.amazon.com/ssm-guiconnect/latest/APIReference/API_Types.html
var ssmGUIConnectDataTypes = []ssmGUIConnectDataType{
	{Name: "ConnectionRecordingPreferences"},
	{Name: "RecordingDestinations"},
	{Name: "S3Bucket"},
}

var ssmGUIConnectDataTypeByName = func() map[string]ssmGUIConnectDataType {
	out := make(map[string]ssmGUIConnectDataType, len(ssmGUIConnectDataTypes))
	for _, dt := range ssmGUIConnectDataTypes {
		out[dt.Name] = dt
	}
	return out
}()
