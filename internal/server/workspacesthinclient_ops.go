package server

type workspacesThinClientOperation struct {
	Name   string
	Method string
	URI    string
}

// Amazon WorkSpaces Thin Client actions sourced from:

// https://docs.aws.amazon.com/workspaces-thin-client/latest/api/API_Operations.html

var workspacesThinClientOperations = []workspacesThinClientOperation{
	{Name: "CreateEnvironment", Method: "POST", URI: "/environments"},
	{Name: "DeleteDevice", Method: "DELETE", URI: "/devices/{id}?clientToken={clientToken}"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/environments/{id}?clientToken={clientToken}"},
	{Name: "DeregisterDevice", Method: "POST", URI: "/deregister-device/{id}"},
	{Name: "GetDevice", Method: "GET", URI: "/devices/{id}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/environments/{id}"},
	{Name: "GetSoftwareSet", Method: "GET", URI: "/softwaresets/{id}"},
	{Name: "ListDevices", Method: "GET", URI: "/devices?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListEnvironments", Method: "GET", URI: "/environments?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListSoftwareSets", Method: "GET", URI: "/softwaresets?maxResults={maxResults}&nextToken={nextToken}"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}?tagKeys={tagKeys}"},
	{Name: "UpdateDevice", Method: "PATCH", URI: "/devices/{id}"},
	{Name: "UpdateEnvironment", Method: "PATCH", URI: "/environments/{id}"},
	{Name: "UpdateSoftwareSet", Method: "PATCH", URI: "/softwaresets/{id}"},
}

var workspacesThinClientOperationByName = func() map[string]workspacesThinClientOperation {
	out := make(map[string]workspacesThinClientOperation, len(workspacesThinClientOperations))
	for _, op := range workspacesThinClientOperations {
		out[op.Name] = op
	}
	return out
}()
