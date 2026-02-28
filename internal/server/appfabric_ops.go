package server

type appFabricOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS AppFabric actions sourced from:
// https://docs.aws.amazon.com/appfabric/latest/api/API_Operations.html
var appFabricOperations = []appFabricOperation{
	{Name: "BatchGetUserAccessTasks", Method: "POST", URI: "/useraccess/batchget"},
	{Name: "ConnectAppAuthorization", Method: "POST", URI: "/appbundles/{appBundleIdentifier}/appauthorizations/{appAuthorizationIdentifier}/connect"},
	{Name: "CreateAppAuthorization", Method: "POST", URI: "/appbundles/{appBundleIdentifier}/appauthorizations"},
	{Name: "CreateAppBundle", Method: "POST", URI: "/appbundles"},
	{Name: "CreateIngestion", Method: "POST", URI: "/appbundles/{appBundleIdentifier}/ingestions"},
	{Name: "CreateIngestionDestination", Method: "POST", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/ingestiondestinations"},
	{Name: "DeleteAppAuthorization", Method: "DELETE", URI: "/appbundles/{appBundleIdentifier}/appauthorizations/{appAuthorizationIdentifier}"},
	{Name: "DeleteAppBundle", Method: "DELETE", URI: "/appbundles/{appBundleIdentifier}"},
	{Name: "DeleteIngestion", Method: "DELETE", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}"},
	{Name: "DeleteIngestionDestination", Method: "DELETE", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/ingestiondestinations/{ingestionDestinationIdentifier}"},
	{Name: "GetAppAuthorization", Method: "GET", URI: "/appbundles/{appBundleIdentifier}/appauthorizations/{appAuthorizationIdentifier}"},
	{Name: "GetAppBundle", Method: "GET", URI: "/appbundles/{appBundleIdentifier}"},
	{Name: "GetIngestion", Method: "GET", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}"},
	{Name: "GetIngestionDestination", Method: "GET", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/ingestiondestinations/{ingestionDestinationIdentifier}"},
	{Name: "ListAppAuthorizations", Method: "GET", URI: "/appbundles/{appBundleIdentifier}/appauthorizations"},
	{Name: "ListAppBundles", Method: "GET", URI: "/appbundles"},
	{Name: "ListIngestionDestinations", Method: "GET", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/ingestiondestinations"},
	{Name: "ListIngestions", Method: "GET", URI: "/appbundles/{appBundleIdentifier}/ingestions"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "StartIngestion", Method: "POST", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/start"},
	{Name: "StartUserAccessTasks", Method: "POST", URI: "/useraccess/start"},
	{Name: "StopIngestion", Method: "POST", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/stop"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateAppAuthorization", Method: "PATCH", URI: "/appbundles/{appBundleIdentifier}/appauthorizations/{appAuthorizationIdentifier}"},
	{Name: "UpdateIngestionDestination", Method: "PATCH", URI: "/appbundles/{appBundleIdentifier}/ingestions/{ingestionIdentifier}/ingestiondestinations/{ingestionDestinationIdentifier}"},
}

var appFabricOperationByName = func() map[string]appFabricOperation {
	out := make(map[string]appFabricOperation, len(appFabricOperations))
	for _, op := range appFabricOperations {
		out[op.Name] = op
	}
	return out
}()
