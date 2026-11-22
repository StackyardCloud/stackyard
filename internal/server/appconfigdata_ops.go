package server

type appConfigDataOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS AppConfig Data operations sourced from:
// https://docs.aws.amazon.com/appconfig/2019-10-09/APIReference/API_Operations_AWS_AppConfig_Data.html
var appConfigDataOperations = []appConfigDataOperation{
	{Name: "GetLatestConfiguration", Method: "GET", URI: "/configuration"},
	{Name: "StartConfigurationSession", Method: "POST", URI: "/configurationsessions"},
}

var appConfigDataOperationByName = func() map[string]appConfigDataOperation {
	out := make(map[string]appConfigDataOperation, len(appConfigDataOperations))
	for _, op := range appConfigDataOperations {
		out[op.Name] = op
	}
	return out
}()
