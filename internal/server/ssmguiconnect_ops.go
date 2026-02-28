package server

type ssmGUIConnectOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Systems Manager GUI Connect operations sourced from:
// https://docs.aws.amazon.com/ssm-guiconnect/latest/APIReference/API_Operations.html
var ssmGUIConnectOperations = []ssmGUIConnectOperation{
	{Name: "DeleteConnectionRecordingPreferences", Method: "POST", URI: "/DeleteConnectionRecordingPreferences"},
	{Name: "GetConnectionRecordingPreferences", Method: "POST", URI: "/GetConnectionRecordingPreferences"},
	{Name: "UpdateConnectionRecordingPreferences", Method: "POST", URI: "/UpdateConnectionRecordingPreferences"},
}

var ssmGUIConnectOperationByName = func() map[string]ssmGUIConnectOperation {
	out := make(map[string]ssmGUIConnectOperation, len(ssmGUIConnectOperations))
	for _, op := range ssmGUIConnectOperations {
		out[op.Name] = op
	}
	return out
}()
