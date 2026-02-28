package server

type migrationHubRefactorSpacesOperation struct {
	Name   string
	Method string
	URI    string
}

// AWS Migration Hub Refactor Spaces operations sourced from:
// https://docs.aws.amazon.com/migrationhub-refactor-spaces/latest/APIReference/API_Operations.html
var migrationHubRefactorSpacesOperations = []migrationHubRefactorSpacesOperation{
	{Name: "CreateApplication", Method: "POST", URI: "/environments/{environmentIdentifier}/applications"},
	{Name: "CreateEnvironment", Method: "POST", URI: "/environments"},
	{Name: "CreateRoute", Method: "POST", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/routes"},
	{Name: "CreateService", Method: "POST", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/services"},
	{Name: "DeleteApplication", Method: "DELETE", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}"},
	{Name: "DeleteEnvironment", Method: "DELETE", URI: "/environments/{environmentIdentifier}"},
	{Name: "DeleteResourcePolicy", Method: "DELETE", URI: "/resourcepolicy/{identifier}"},
	{Name: "DeleteRoute", Method: "DELETE", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/routes/{routeIdentifier}"},
	{Name: "DeleteService", Method: "DELETE", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/services/{serviceIdentifier}"},
	{Name: "GetApplication", Method: "GET", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}"},
	{Name: "GetEnvironment", Method: "GET", URI: "/environments/{environmentIdentifier}"},
	{Name: "GetResourcePolicy", Method: "GET", URI: "/resourcepolicy/{identifier}"},
	{Name: "GetRoute", Method: "GET", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/routes/{routeIdentifier}"},
	{Name: "GetService", Method: "GET", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/services/{serviceIdentifier}"},
	{Name: "ListApplications", Method: "GET", URI: "/environments/{environmentIdentifier}/applications"},
	{Name: "ListEnvironmentVpcs", Method: "GET", URI: "/environments/{environmentIdentifier}/vpcs"},
	{Name: "ListEnvironments", Method: "GET", URI: "/environments"},
	{Name: "ListRoutes", Method: "GET", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/routes"},
	{Name: "ListServices", Method: "GET", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/services"},
	{Name: "ListTagsForResource", Method: "GET", URI: "/tags/{resourceArn}"},
	{Name: "PutResourcePolicy", Method: "PUT", URI: "/resourcepolicy"},
	{Name: "TagResource", Method: "POST", URI: "/tags/{resourceArn}"},
	{Name: "UntagResource", Method: "DELETE", URI: "/tags/{resourceArn}"},
	{Name: "UpdateRoute", Method: "PATCH", URI: "/environments/{environmentIdentifier}/applications/{applicationIdentifier}/routes/{routeIdentifier}"},
}

var migrationHubRefactorSpacesOperationByName = func() map[string]migrationHubRefactorSpacesOperation {
	out := make(map[string]migrationHubRefactorSpacesOperation, len(migrationHubRefactorSpacesOperations))
	for _, op := range migrationHubRefactorSpacesOperations {
		out[op.Name] = op
	}
	return out
}()
