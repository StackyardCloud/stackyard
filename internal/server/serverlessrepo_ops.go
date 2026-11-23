package server

type serverlessrepoOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Serverless Application Repository operations sourced from:
// https://docs.aws.amazon.com/serverlessrepo/latest/devguide/resources.html
var serverlessrepoOperations = []serverlessrepoOperation{
	{Name: "CreateApplication", Method: "POST", URI: "/applications"},
	{Name: "CreateApplicationVersion", Method: "PUT", URI: "/applications/{applicationId}/versions/{semanticVersion}"},
	{Name: "CreateCloudFormationChangeSet", Method: "POST", URI: "/applications/{applicationId}/changesets"},
	{Name: "CreateCloudFormationTemplate", Method: "POST", URI: "/applications/{applicationId}/templates"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/applications/{applicationId}"},
	{Name: "GetApplication", Method: "GET", URI: "/applications/{applicationId}"},
	{Name: "GetApplicationPolicy", Method: "GET", URI: "/applications/{applicationId}/policy"},
	{Name: "GetCloudFormationTemplate", Method: "GET", URI: "/applications/{applicationId}/templates/{templateId}"},
	{Name: "ListApplicationDependencies", Method: "GET", URI: "/applications/{applicationId}/dependencies"},
	{Name: "ListApplicationVersions", Method: "GET", URI: "/applications/{applicationId}/versions"},
	{Name: "ListApplications", Method: "GET", URI: "/applications"},
	{Name: "PutApplicationPolicy", Method: "PUT", URI: "/applications/{applicationId}/policy"},
	{Name: "UnshareApplication", Method: "POST", URI: "/applications/{applicationId}/unshare"},
	{Name: "UpdateApplication", Method: "PATCH", URI: "/applications/{applicationId}"},
}

var serverlessrepoOperationByName = func() map[string]serverlessrepoOperation {
	out := make(map[string]serverlessrepoOperation, len(serverlessrepoOperations))
	for _, op := range serverlessrepoOperations {
		out[op.Name] = op
	}
	return out
}()
